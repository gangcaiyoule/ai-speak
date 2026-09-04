/// @file playback_queue.h
/// @brief 播放队列：SPSC 环缓 + 预缓冲/欠载状态机（C11，header-only，R7）。
///
/// 与采集侧（voice_input.h 一族）共用 spsc_ring_buffer.h，但生产者/消费者
/// 角色反转：**生产者是 Dart 线程（vo_write 喂数据），消费者是音频回调线程
/// （render/output 回调取数据）**。本组件在环缓之上实现播放特有的欠载策略：
///
/// - **预缓冲（priming）**：启动后回调不立即取数，攒够 priming_bytes 才开始
///   消费，避免边写边读造成启动期连续欠载；欠载后回到预缓冲状态（迟滞），
///   重新攒够阈值再恢复，避免反复抖动。
/// - **写入全有或全无**：空间不足时整块拒绝（契约见 contracts.dart
///   AudioSink.write「缓冲满则拒绝，永不阻塞」），拒绝量累计到 rejected，
///   不做覆盖丢旧——播放语义下覆盖最旧数据会造成跳音序错乱。
/// - **欠载统计**：播放已开始（ever_primed）后，回调每次输出静音的字节数
///   累计到 underrun；启动预缓冲期的静音不计入。
///
/// 线程纪律（与 spsc_ring_buffer.h 一致）：
/// - vsc_pbq_write 只允许生产者（Dart 线程）调用；
/// - vsc_pbq_acquire/vsc_pbq_commit 只允许消费者（音频回调）调用，且必须
///   同步成对：commit 前不得把 view 交给其他线程或持有到下一轮回调；
/// - 回调内零分配、零锁、零阻塞；状态机字段只由消费者触碰。
///
/// 全部逻辑无平台依赖，供 NDK/Xcode 直接编译，并有独立 C 单测
/// （test_playback_queue.c，不进应用构建）。
#ifndef VSC_PLAYBACK_QUEUE_H_
#define VSC_PLAYBACK_QUEUE_H_

#include <stdbool.h>
#include <stdint.h>
#include <stdatomic.h>

#include "spsc_ring_buffer.h"

#ifdef __cplusplus
extern "C" {
#endif

/// @brief 播放队列实例。
typedef struct {
  vsc_spsc_ring_t *ring;   ///< 底层 SPSC 环缓（外部持有存储区）。
  uint32_t priming_bytes;  ///< 预缓冲阈值，单位字节。
  bool primed;             ///< 当前是否处于正常消费状态（仅消费者线程触碰）。
  bool ever_primed;        ///< 是否已完成过一次预缓冲（区分启动静音与欠载）。
  _Atomic uint64_t underrun; ///< 欠载静音累计字节数（消费者线程写）。
  _Atomic uint64_t rejected; ///< 空间不足被整块拒绝的字节数（生产者线程写）。
} vsc_pbq_t;

/// @brief 初始化播放队列。
///
/// @param q 实例指针。
/// @param ring 底层环缓；生命周期必须覆盖整个使用期。
/// @param priming_bytes 预缓冲阈值；0 表示不预缓冲（有数据即播），
///        不得大于环缓容量（否则永远无法达到阈值）。
/// @return 0 成功；-1 参数非法。
static inline int vsc_pbq_init(vsc_pbq_t *q, vsc_spsc_ring_t *ring,
                               uint32_t priming_bytes) {
  if (q == NULL || ring == NULL || priming_bytes > ring->capacity) {
    return -1;
  }
  q->ring = ring;
  q->priming_bytes = priming_bytes;
  q->primed = false;
  q->ever_primed = false;
  atomic_store_explicit(&q->underrun, 0, memory_order_relaxed);
  atomic_store_explicit(&q->rejected, 0, memory_order_relaxed);
  return 0;
}

/// @brief 当前缓冲的可播放字节数。
static inline uint32_t vsc_pbq_buffered(const vsc_pbq_t *q) {
  return vsc_rb_size(q->ring);
}

/// @brief 欠载静音累计字节数。
static inline uint64_t vsc_pbq_underrun(const vsc_pbq_t *q) {
  return atomic_load_explicit(&q->underrun, memory_order_acquire);
}

/// @brief 空间不足被整块拒绝的累计字节数。
static inline uint64_t vsc_pbq_rejected(const vsc_pbq_t *q) {
  return atomic_load_explicit(&q->rejected, memory_order_acquire);
}

/// @brief 生产者写入一帧；全有或全无，永不阻塞。
///
/// 空间不足（或 len 超过容量）时整块拒绝：计入 rejected 并返回 0，
/// 缓冲内容不变。SPSC 语义下检查与写入之间消费者只会腾出空间，
/// 不会出现检查后放不下的竞态。
///
/// @param q 实例指针。
/// @param data 待写入字节。
/// @param len 待写入长度。
/// @return 实际接受的字节数：len（全部接受）或 0（整块拒绝）。
static inline uint32_t vsc_pbq_write(vsc_pbq_t *q, const uint8_t *data,
                                     uint32_t len) {
  if (q == NULL || data == NULL || len == 0) {
    return 0;
  }
  const uint32_t capacity = vsc_rb_capacity(q->ring);
  if (len > capacity || len > capacity - vsc_rb_size(q->ring)) {
    atomic_fetch_add_explicit(&q->rejected, len, memory_order_relaxed);
    return 0;
  }
  (void)vsc_rb_write(q->ring, data, len);
  return len;
}

/// @brief 消费者取数：按预缓冲/欠载状态机返回本次可播放的字节。
///
/// 状态迁移与统计：
/// - 未达预缓冲阈值：返回 0（回调输出静音）；若播放已开始过
///   （ever_primed），这批静音计入 underrun。
/// - 已预缓冲但缓冲被取空：回退到预缓冲状态（迟滞），静音计入 underrun。
/// - 部分取数（可取 < 请求）：照常消费已有数据，缺口计入 underrun，
///   不回退预缓冲状态（数据仍在流入，回退只会白增延迟）。
///
/// @param q 实例指针。
/// @param max_bytes 本次回调需要的字节数（numFrames × frame_bytes）。
/// @param views 输出数组，容量至少 2；语义同 vsc_rb_peek。
/// @return 实际取到的字节数；0 表示本轮回调应输出静音。
/// @note 取到的字节必须用 vsc_pbq_commit 同步消费，成对约束同 peek/advance。
static inline uint32_t vsc_pbq_acquire(vsc_pbq_t *q, uint32_t max_bytes,
                                       vsc_rb_view_t views[2]) {
  const uint32_t size = vsc_rb_size(q->ring);
  if (!q->primed) {
    if (size < q->priming_bytes || size == 0) {
      if (q->ever_primed) {
        atomic_fetch_add_explicit(&q->underrun, max_bytes,
                                  memory_order_relaxed);
      }
      return 0;
    }
    q->primed = true;
    q->ever_primed = true;
  }
  if (size == 0) {
    q->primed = false;
    atomic_fetch_add_explicit(&q->underrun, max_bytes, memory_order_relaxed);
    return 0;
  }
  const int nviews = vsc_rb_peek(q->ring, max_bytes, views);
  uint32_t take = 0;
  for (int i = 0; i < nviews; ++i) {
    take += views[i].len;
  }
  if (take < max_bytes) {
    atomic_fetch_add_explicit(&q->underrun, max_bytes - take,
                              memory_order_relaxed);
  }
  return take;
}

/// @brief 消费最近一次 acquire 取出的字节（语义同 vsc_rb_advance）。
///
/// @param q 实例指针。
/// @param bytes 待消费字节数，不得超出最近一次 acquire 的返回值。
/// @return 0 成功；-1 越界。
static inline int vsc_pbq_commit(vsc_pbq_t *q, uint32_t bytes) {
  return vsc_rb_advance(q->ring, bytes);
}

#ifdef __cplusplus
}
#endif

#endif // VSC_PLAYBACK_QUEUE_H_
