/// @file test_playback_queue.c
/// @brief 播放队列（playback_queue.h）的单元测试：欠载策略与统计。
///
/// 不进应用构建，手动编译运行：
/// @code
/// cc -std=c11 -Wall -Wextra -Werror -O2 test_playback_queue.c -o c_pbq_test && ./c_pbq_test
/// @endcode
///
/// 覆盖：init 参数校验、全有或全无写入与拒绝计数、预缓冲阈值与迟滞、
/// 欠载计数（启动静音不计）、部分取数缺口计数、跨回绕数据连续性。

#include <stdio.h>
#include <stdint.h>
#include <string.h>

#include "playback_queue.h"

/// @brief 断言宏：失败时打印位置并返回非零。
#define CHECK(cond)                                                     \
  do {                                                                  \
    if (!(cond)) {                                                      \
      fprintf(stderr, "FAIL %s:%d: %s\n", __FILE__, __LINE__, #cond);   \
      return 1;                                                         \
    }                                                                   \
  } while (0)

/// @brief 组装辅助：初始化容量 capacity 的队列（阈值 priming）。
static int setup(vsc_pbq_t *q, vsc_spsc_ring_t *rb, uint8_t *storage,
                 uint32_t capacity, uint32_t priming) {
  CHECK(vsc_spsc_ring_init(rb, storage, capacity) == 0);
  CHECK(vsc_pbq_init(q, rb, priming) == 0);
  return 0;
}

/// @brief 用例 1：init 参数校验（阈值不得超过容量）。
static int test_init_validation(void) {
  vsc_spsc_ring_t rb;
  uint8_t storage[8];
  CHECK(vsc_spsc_ring_init(&rb, storage, sizeof(storage)) == 0);

  vsc_pbq_t q;
  CHECK(vsc_pbq_init(NULL, &rb, 0) == -1);
  CHECK(vsc_pbq_init(&q, NULL, 0) == -1);
  CHECK(vsc_pbq_init(&q, &rb, sizeof(storage) + 1) == -1);
  CHECK(vsc_pbq_init(&q, &rb, sizeof(storage)) == 0);
  return 0;
}

/// @brief 用例 2：写入全有或全无——空间不足整块拒绝并计数，缓冲不变。
static int test_write_all_or_nothing(void) {
  vsc_pbq_t q;
  vsc_spsc_ring_t rb;
  uint8_t storage[8];
  CHECK(setup(&q, &rb, storage, sizeof(storage), 0) == 0);

  const uint8_t a[] = {1, 2, 3, 4, 5, 6};
  CHECK(vsc_pbq_write(&q, a, sizeof(a)) == sizeof(a));
  CHECK(vsc_pbq_buffered(&q) == sizeof(a));
  CHECK(vsc_pbq_rejected(&q) == 0);

  // 剩余空间 2 字节：要 3 字节，整块拒绝。
  const uint8_t b[] = {7, 8, 9};
  CHECK(vsc_pbq_write(&q, b, sizeof(b)) == 0);
  CHECK(vsc_pbq_buffered(&q) == sizeof(a));
  CHECK(vsc_pbq_rejected(&q) == sizeof(b));

  // 超过容量：整块拒绝。
  const uint8_t big[16] = {0};
  CHECK(vsc_pbq_write(&q, big, sizeof(big)) == 0);
  CHECK(vsc_pbq_rejected(&q) == sizeof(b) + sizeof(big));

  // 空出空间后同样的数据可以被接受。
  uint8_t out[8] = {0};
  vsc_rb_view_t views[2];
  CHECK(vsc_pbq_acquire(&q, sizeof(out), views) == sizeof(a));
  CHECK(views[0].len == sizeof(a));
  CHECK(memcmp(views[0].data, a, sizeof(a)) == 0);
  CHECK(vsc_pbq_commit(&q, sizeof(a)) == 0);
  (void)out;

  CHECK(vsc_pbq_write(&q, b, sizeof(b)) == sizeof(b));
  CHECK(vsc_pbq_rejected(&q) == sizeof(b) + sizeof(big));
  return 0;
}

/// @brief 用例 3：预缓冲——阈值未到不出数且不计欠载；达到后开始消费。
static int test_priming_threshold(void) {
  vsc_pbq_t q;
  vsc_spsc_ring_t rb;
  uint8_t storage[16];
  CHECK(setup(&q, &rb, storage, sizeof(storage), 8) == 0);

  vsc_rb_view_t views[2];
  // 6 字节 < 阈值 8：回调取不到数，且启动期静音不计欠载。
  const uint8_t a[6] = {1};
  CHECK(vsc_pbq_write(&q, a, sizeof(a)) == sizeof(a));
  CHECK(vsc_pbq_acquire(&q, 4, views) == 0);
  CHECK(vsc_pbq_underrun(&q) == 0);

  // 再写 2 字节凑到阈值：开始出数。
  const uint8_t b[2] = {2, 3};
  CHECK(vsc_pbq_write(&q, b, sizeof(b)) == sizeof(b));
  CHECK(vsc_pbq_acquire(&q, 6, views) == 6);
  CHECK(views[0].len == 6);
  CHECK(views[0].data[0] == 1);
  CHECK(vsc_pbq_commit(&q, 6) == 0);
  CHECK(vsc_pbq_underrun(&q) == 0);

  // 再次取数：缓冲剩 2 字节，请求 4 字节，实际取 2 字节，缺口 2 字节计入欠载。
  CHECK(vsc_pbq_acquire(&q, 4, views) == 2);
  CHECK(views[0].len == 2);
  CHECK(views[0].data[0] == 2 && views[0].data[1] == 3);
  CHECK(vsc_pbq_commit(&q, 2) == 0);
  CHECK(vsc_pbq_underrun(&q) == 2);
  return 0;
}

/// @brief 用例 4：欠载迟滞——取空后回退预缓冲，恢复前静音计入欠载。
static int test_underrun_hysteresis(void) {
  vsc_pbq_t q;
  vsc_spsc_ring_t rb;
  uint8_t storage[16];
  CHECK(setup(&q, &rb, storage, sizeof(storage), 4) == 0);

  const uint8_t data[6] = {1, 2, 3, 4, 5, 6};
  CHECK(vsc_pbq_write(&q, data, sizeof(data)) == sizeof(data));

  vsc_rb_view_t views[2];
  CHECK(vsc_pbq_acquire(&q, 6, views) == 6); // 达阈值，开始播放。
  CHECK(vsc_pbq_commit(&q, 6) == 0);

  // 取空：欠载一次，计本轮回调请求量 4。
  CHECK(vsc_pbq_acquire(&q, 4, views) == 0);
  CHECK(vsc_pbq_underrun(&q) == 4);

  // 回退到预缓冲状态：未达阈值期间继续静音并继续计数。
  const uint8_t part[2] = {7, 8};
  CHECK(vsc_pbq_write(&q, part, sizeof(part)) == sizeof(part));
  CHECK(vsc_pbq_acquire(&q, 4, views) == 0);
  CHECK(vsc_pbq_underrun(&q) == 8);

  // 攒回阈值以上：恢复正常消费，欠载不再增长。
  CHECK(vsc_pbq_write(&q, part, sizeof(part)) == sizeof(part));
  CHECK(vsc_pbq_acquire(&q, 4, views) == 4);
  CHECK(views[0].len == 4);
  CHECK(views[0].data[0] == 7 && views[0].data[1] == 8 &&
        views[0].data[2] == 7 && views[0].data[3] == 8);
  CHECK(vsc_pbq_commit(&q, 4) == 0);
  CHECK(vsc_pbq_underrun(&q) == 8);
  return 0;
}

/// @brief 用例 5：部分取数——缺口计入欠载，但不回退预缓冲状态。
static int test_partial_take_counts_gap(void) {
  vsc_pbq_t q;
  vsc_spsc_ring_t rb;
  uint8_t storage[16];
  CHECK(setup(&q, &rb, storage, sizeof(storage), 2) == 0);

  const uint8_t data[3] = {1, 2, 3};
  CHECK(vsc_pbq_write(&q, data, sizeof(data)) == sizeof(data));

  vsc_rb_view_t views[2];
  // 请求 5 只有 3：出 3 计缺口 2，状态保持 primed。
  CHECK(vsc_pbq_acquire(&q, 5, views) == 3);
  CHECK(vsc_pbq_underrun(&q) == 2);
  CHECK(vsc_pbq_commit(&q, 3) == 0);

  // 下一次取空才回退预缓冲：这次请求 4 全部计欠载。
  CHECK(vsc_pbq_acquire(&q, 4, views) == 0);
  CHECK(vsc_pbq_underrun(&q) == 6);
  return 0;
}

/// @brief 用例 6：跨回绕数据连续，acquire 返回两段视图。
static int test_wraparound_two_views(void) {
  vsc_pbq_t q;
  vsc_spsc_ring_t rb;
  uint8_t storage[4];
  CHECK(setup(&q, &rb, storage, sizeof(storage), 0) == 0);

  const uint8_t a[3] = {1, 2, 3};
  CHECK(vsc_pbq_write(&q, a, sizeof(a)) == sizeof(a));
  vsc_rb_view_t views[2];
  CHECK(vsc_pbq_acquire(&q, 2, views) == 2);
  CHECK(vsc_pbq_commit(&q, 2) == 0); // 剩 idx2 的 3。

  const uint8_t b[3] = {4, 5, 6};
  CHECK(vsc_pbq_write(&q, b, sizeof(b)) == sizeof(b)); // 3 字节进剩余 3 字节。

  CHECK(vsc_pbq_acquire(&q, 4, views) == 4);
  CHECK(views[0].len == 2 && views[0].data[0] == 3 && views[0].data[1] == 4);
  CHECK(views[1].len == 2 && views[1].data[0] == 5 && views[1].data[1] == 6);
  CHECK(vsc_pbq_commit(&q, 4) == 0);
  CHECK(vsc_pbq_buffered(&q) == 0);
  return 0;
}

/// @brief 用例 7：零阈值直通与非法入参。
static int test_zero_priming_and_bad_args(void) {
  vsc_pbq_t q;
  vsc_spsc_ring_t rb;
  uint8_t storage[8];
  CHECK(setup(&q, &rb, storage, sizeof(storage), 0) == 0);

  vsc_rb_view_t views[2];
  CHECK(vsc_pbq_acquire(&q, 4, views) == 0); // 空缓冲：静音不计欠载。
  CHECK(vsc_pbq_underrun(&q) == 0);

  const uint8_t a[2] = {9, 9};
  CHECK(vsc_pbq_write(&q, a, sizeof(a)) == sizeof(a));
  CHECK(vsc_pbq_acquire(&q, 4, views) == 2); // 阈值 0：有数据即播。
  CHECK(vsc_pbq_commit(&q, 2) == 0);

  // 非法入参：写空指针/零长度均不接受。
  CHECK(vsc_pbq_write(&q, NULL, 4) == 0);
  CHECK(vsc_pbq_write(&q, a, 0) == 0);
  CHECK(vsc_pbq_rejected(&q) == 0); // 参数错误不计入拒绝统计。
  return 0;
}

int main(void) {
  static const struct {
    const char *name;
    int (*fn)(void);
  } cases[] = {
      {"init_validation", test_init_validation},
      {"write_all_or_nothing", test_write_all_or_nothing},
      {"priming_threshold", test_priming_threshold},
      {"underrun_hysteresis", test_underrun_hysteresis},
      {"partial_take_counts_gap", test_partial_take_counts_gap},
      {"wraparound_two_views", test_wraparound_two_views},
      {"zero_priming_and_bad_args", test_zero_priming_and_bad_args},
  };
  for (size_t i = 0; i < sizeof(cases) / sizeof(cases[0]); ++i) {
    if (cases[i].fn() != 0) {
      fprintf(stderr, "case %s FAILED\n", cases[i].name);
      return 1;
    }
    printf("ok %s\n", cases[i].name);
  }
  printf("all %zu cases passed\n", sizeof(cases) / sizeof(cases[0]));
  return 0;
}
