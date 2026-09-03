/// @file spsc_ring_buffer.h
/// @brief 单生产者-单消费者（SPSC）定容字节环形缓冲（C11，header-only）。
///
/// 与 Dart 参考实现 `src/ring_buffer.dart` 语义逐条对齐，供 NDK（Android）
/// 与 Xcode（iOS）双端直接编译：
///
/// - 生产者永不阻塞：@c vsc_rb_write 空间不足时覆盖最旧数据，丢弃量累计到
///   @c dropped 计数。
/// - 消费者零拷贝：@c vsc_rb_peek 返回内部存储的只读视图（跨回绕点为两段），
///   @c vsc_rb_advance 之后该区间才可被覆盖。
/// - peek/advance 必须同步成对调用：不得异步持有视图（丢旧会推进读指针，
///   覆盖未提交区间）。
/// - 无锁：head/tail 为单调递增的 64 位原子量，生产者释放（release）、
///   消费者获取（acquire），单写单读下无锁。
///
/// @note 覆盖丢旧时生产者会单向推进 tail（仅前进，不回退），与消费者推进
///       同向且原子，单调性保证不会读出「未来的」数据；被覆盖区间的撕裂读
///       由 peek/advance 成对约束排除。
#ifndef VSC_SPSC_RING_BUFFER_H_
#define VSC_SPSC_RING_BUFFER_H_

#include <stdbool.h>
#include <stddef.h>
#include <stdint.h>
#include <stdatomic.h>
#include <string.h>

#ifdef __cplusplus
extern "C" {
#endif

/// @brief SPSC 环形缓冲实例。
typedef struct {
  uint8_t *storage;        ///< 外部提供的存储区。
  uint32_t capacity;       ///< 存储区容量，单位字节。
  _Atomic uint64_t head;   ///< 生产者累计写入总字节数（单调递增）。
  _Atomic uint64_t tail;   ///< 消费者累计消费总字节数（单调递增）。
  _Atomic uint64_t dropped; ///< 覆盖丢旧累计字节数。
} vsc_spsc_ring_t;

/// @brief 只读视图：内部存储的一段区间。
typedef struct {
  const uint8_t *data; ///< 区间起始指针。
  uint32_t len;        ///< 区间长度，单位字节。
} vsc_rb_view_t;

/// @brief 初始化环形缓冲。
///
/// @param rb 实例指针。
/// @param storage 外部提供的存储区；生命周期必须覆盖整个使用期。
/// @param capacity 存储区容量，必须为正数。
/// @return 0 成功；-1 参数非法。
static inline int vsc_spsc_ring_init(vsc_spsc_ring_t *rb, uint8_t *storage,
                                     uint32_t capacity) {
  if (rb == NULL || storage == NULL || capacity == 0) {
    return -1;
  }
  rb->storage = storage;
  rb->capacity = capacity;
  atomic_store_explicit(&rb->head, 0, memory_order_relaxed);
  atomic_store_explicit(&rb->tail, 0, memory_order_relaxed);
  atomic_store_explicit(&rb->dropped, 0, memory_order_relaxed);
  return 0;
}

/// @brief 缓冲容量，单位字节。
static inline uint32_t vsc_rb_capacity(const vsc_spsc_ring_t *rb) {
  return rb->capacity;
}

/// @brief 当前可读字节数。
static inline uint32_t vsc_rb_size(const vsc_spsc_ring_t *rb) {
  uint64_t head = atomic_load_explicit(&rb->head, memory_order_acquire);
  uint64_t tail = atomic_load_explicit(&rb->tail, memory_order_acquire);
  return (uint32_t)(head - tail);
}

/// @brief 是否没有可读数据。
static inline bool vsc_rb_is_empty(const vsc_spsc_ring_t *rb) {
  return vsc_rb_size(rb) == 0;
}

/// @brief 覆盖丢旧累计字节数。
static inline uint64_t vsc_rb_dropped(const vsc_spsc_ring_t *rb) {
  return atomic_load_explicit(&rb->dropped, memory_order_relaxed);
}

/// @brief 写入数据；空间不足时覆盖最旧数据，永不阻塞、永不失败。
///
/// 若 @p len 超过容量，只保留数据末尾 capacity 字节，其余计入丢弃。
///
/// @param rb 实例指针。
/// @param data 待写入字节。
/// @param len 待写入长度。
/// @return 写入后缓冲中的可读字节数。
static inline uint32_t vsc_rb_write(vsc_spsc_ring_t *rb, const uint8_t *data,
                                    uint32_t len) {
  const uint8_t *src = data;
  uint32_t n = len;
  uint64_t dropped_now = 0;

  if (n > rb->capacity) {
    const uint32_t excess = n - rb->capacity;
    dropped_now += excess;
    src += excess;
    n = rb->capacity;
  }
  if (n == 0) {
    return vsc_rb_size(rb);
  }

  const uint64_t head =
      atomic_load_explicit(&rb->head, memory_order_relaxed);
  const uint64_t tail =
      atomic_load_explicit(&rb->tail, memory_order_acquire);
  const uint64_t used = head - tail;

  if (n > rb->capacity - used) {
    const uint32_t overflow = (uint32_t)(n - (rb->capacity - used));
    dropped_now += overflow;
    atomic_store_explicit(&rb->tail, tail + overflow, memory_order_release);
  }

  const uint32_t head_idx = (uint32_t)(head % rb->capacity);
  uint32_t first = rb->capacity - head_idx;
  if (first > n) {
    first = n;
  }
  memcpy(rb->storage + head_idx, src, first);
  if (n > first) {
    memcpy(rb->storage, src + first, n - first);
  }

  atomic_store_explicit(&rb->head, head + n, memory_order_release);
  if (dropped_now != 0) {
    atomic_fetch_add_explicit(&rb->dropped, dropped_now,
                              memory_order_relaxed);
  }
  return vsc_rb_size(rb);
}

/// @brief 零拷贝取出可读数据的只读视图。
///
/// 数据跨越回绕点时返回两段视图，否则一段；无数据返回 0。视图与内部存储
/// 共享内存，@c vsc_rb_advance 前有效。
///
/// @param rb 实例指针。
/// @param max_bytes 最多取出的字节数；传 0 表示不设上限（取全部可读数据）。
/// @param views 输出数组，容量至少 2。
/// @return 视图数量：0、1 或 2。
static inline int vsc_rb_peek(const vsc_spsc_ring_t *rb, uint32_t max_bytes,
                              vsc_rb_view_t views[2]) {
  const uint64_t tail =
      atomic_load_explicit(&rb->tail, memory_order_acquire);
  const uint64_t head =
      atomic_load_explicit(&rb->head, memory_order_acquire);
  const uint64_t avail = head - tail;
  if (avail == 0) {
    return 0;
  }
  const uint32_t take =
      (max_bytes == 0 || (uint64_t)max_bytes > avail) ? (uint32_t)avail
                                                      : max_bytes;
  const uint32_t tail_idx = (uint32_t)(tail % rb->capacity);
  uint32_t first = rb->capacity - tail_idx;
  if (first > take) {
    first = take;
  }
  views[0].data = rb->storage + tail_idx;
  views[0].len = first;
  if (take > first) {
    views[1].data = rb->storage;
    views[1].len = take - first;
    return 2;
  }
  return 1;
}

/// @brief 消费最近一次 peek 取出的字节。
///
/// @param rb 实例指针。
/// @param bytes 待消费字节数，不得超出可读范围。
/// @return 0 成功；-1 越界。
static inline int vsc_rb_advance(vsc_spsc_ring_t *rb, uint32_t bytes) {
  const uint64_t head =
      atomic_load_explicit(&rb->head, memory_order_acquire);
  const uint64_t tail =
      atomic_load_explicit(&rb->tail, memory_order_relaxed);
  if ((uint64_t)bytes > head - tail) {
    return -1;
  }
  atomic_store_explicit(&rb->tail, tail + bytes, memory_order_release);
  return 0;
}

/// @brief 一次性拷贝读取。
///
/// 供不想管理视图生命周期的调用方使用；热点路径请用 peek/advance。
///
/// @param rb 实例指针。
/// @param dst 目标缓冲。
/// @param dst_len 目标缓冲长度。
/// @return 实际读取字节数。
static inline uint32_t vsc_rb_read(vsc_spsc_ring_t *rb, uint8_t *dst,
                                   uint32_t dst_len) {
  vsc_rb_view_t views[2];
  const int nviews = vsc_rb_peek(rb, dst_len, views);
  uint32_t total = 0;
  for (int i = 0; i < nviews; ++i) {
    memcpy(dst + total, views[i].data, views[i].len);
    total += views[i].len;
  }
  if (total != 0) {
    (void)vsc_rb_advance(rb, total);
  }
  return total;
}

#ifdef __cplusplus
}
#endif

#endif // VSC_SPSC_RING_BUFFER_H_
