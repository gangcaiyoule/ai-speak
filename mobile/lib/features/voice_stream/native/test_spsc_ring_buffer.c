/// @file test_spsc_ring_buffer.c
/// @brief SPSC 环形缓冲 C 实现的单元测试（与 Dart 用例逐条对齐）。
///
/// 不进应用构建，手动编译运行：
/// @code
/// cc -std=c11 -Wall -Wextra -Werror -O2 test_spsc_ring_buffer.c -o c_ring_test && ./c_ring_test
/// @endcode
///
/// 用例来源：mobile/test/voice_stream/ring_buffer_test.dart（九个用例），
/// 另加一条跨回绕顺序性压力用例。

#include <stdio.h>
#include <stdlib.h>
#include <string.h>

#include "spsc_ring_buffer.h"

/// @brief 断言宏：失败时打印位置并返回非零。
#define CHECK(cond)                                                     \
  do {                                                                  \
    if (!(cond)) {                                                      \
      fprintf(stderr, "FAIL %s:%d: %s\n", __FILE__, __LINE__, #cond);    \
      return 1;                                                         \
    }                                                                   \
  } while (0)

/// @brief 用例 1：读写往返保持字节序。
static int test_roundtrip(void) {
  vsc_spsc_ring_t rb;
  uint8_t storage[8];
  CHECK(vsc_spsc_ring_init(&rb, storage, sizeof(storage)) == 0);

  const uint8_t in[] = {1, 2, 3, 4, 5};
  CHECK(vsc_rb_write(&rb, in, sizeof(in)) == 5);

  uint8_t out[8] = {0};
  CHECK(vsc_rb_read(&rb, out, sizeof(out)) == 5);
  CHECK(memcmp(out, in, 5) == 0);
  CHECK(vsc_rb_is_empty(&rb));
  CHECK(vsc_rb_dropped(&rb) == 0);
  return 0;
}

/// @brief 用例 2：跨回绕点后数据仍连续，peek 返回两段视图。
static int test_wraparound_peek_two_views(void) {
  vsc_spsc_ring_t rb;
  uint8_t storage[4];
  CHECK(vsc_spsc_ring_init(&rb, storage, sizeof(storage)) == 0);

  const uint8_t a[] = {1, 2, 3};
  CHECK(vsc_rb_write(&rb, a, sizeof(a)) == 3);
  CHECK(vsc_rb_advance(&rb, 2) == 0); // 只剩 idx2 的字节 3。

  const uint8_t b[] = {4, 5, 6};
  CHECK(vsc_rb_write(&rb, b, sizeof(b)) == 4); // 3 写满回绕。

  vsc_rb_view_t views[2];
  const int nviews = vsc_rb_peek(&rb, 0, views);
  CHECK(nviews == 2);
  CHECK(views[0].len == 2 && views[0].data[0] == 3 && views[0].data[1] == 4);
  CHECK(views[1].len == 2 && views[1].data[0] == 5 && views[1].data[1] == 6);

  CHECK(vsc_rb_advance(&rb, 4) == 0);
  CHECK(vsc_rb_is_empty(&rb));
  return 0;
}

/// @brief 用例 3：写满后覆盖最旧数据并累计 dropped。
static int test_overwrite_counts_dropped(void) {
  vsc_spsc_ring_t rb;
  uint8_t storage[4];
  CHECK(vsc_spsc_ring_init(&rb, storage, sizeof(storage)) == 0);

  const uint8_t first[] = {1, 2, 3, 4};
  CHECK(vsc_rb_write(&rb, first, sizeof(first)) == 4);
  const uint8_t second[] = {5, 6};
  CHECK(vsc_rb_write(&rb, second, sizeof(second)) == 4);

  CHECK(vsc_rb_dropped(&rb) == 2);
  uint8_t out[4] = {0};
  CHECK(vsc_rb_read(&rb, out, sizeof(out)) == 4);
  const uint8_t expect[] = {3, 4, 5, 6};
  CHECK(memcmp(out, expect, 4) == 0);
  return 0;
}

/// @brief 用例 4：单次写入超过容量时只保留末尾。
static int test_oversized_write_keeps_tail(void) {
  vsc_spsc_ring_t rb;
  uint8_t storage[4];
  CHECK(vsc_spsc_ring_init(&rb, storage, sizeof(storage)) == 0);

  const uint8_t in[] = {1, 2, 3, 4, 5, 6, 7};
  CHECK(vsc_rb_write(&rb, in, sizeof(in)) == 4);
  CHECK(vsc_rb_dropped(&rb) == 3);

  uint8_t out[8] = {0};
  CHECK(vsc_rb_read(&rb, out, sizeof(out)) == 4);
  const uint8_t expect[] = {4, 5, 6, 7};
  CHECK(memcmp(out, expect, 4) == 0);
  return 0;
}

/// @brief 用例 5：peek 限定字节数，advance 分批消费。
static int test_peek_limited_advance_batched(void) {
  vsc_spsc_ring_t rb;
  uint8_t storage[8];
  CHECK(vsc_spsc_ring_init(&rb, storage, sizeof(storage)) == 0);

  const uint8_t in[] = {1, 2, 3, 4, 5, 6};
  CHECK(vsc_rb_write(&rb, in, sizeof(in)) == 6);

  vsc_rb_view_t views[2];
  CHECK(vsc_rb_peek(&rb, 2, views) == 1);
  CHECK(views[0].len == 2 && views[0].data[0] == 1 && views[0].data[1] == 2);
  CHECK(vsc_rb_advance(&rb, 2) == 0);

  uint8_t out[8] = {0};
  CHECK(vsc_rb_read(&rb, out, sizeof(out)) == 4);
  const uint8_t expect[] = {3, 4, 5, 6};
  CHECK(memcmp(out, expect, 4) == 0);
  return 0;
}

/// @brief 用例 6：readInto 目标缓冲小于可读量时截断。
static int test_read_truncates_to_dst(void) {
  vsc_spsc_ring_t rb;
  uint8_t storage[8];
  CHECK(vsc_spsc_ring_init(&rb, storage, sizeof(storage)) == 0);

  const uint8_t in[] = {9, 8, 7, 6};
  CHECK(vsc_rb_write(&rb, in, sizeof(in)) == 4);

  uint8_t out[2] = {0};
  CHECK(vsc_rb_read(&rb, out, sizeof(out)) == 2);
  CHECK(out[0] == 9 && out[1] == 8);
  CHECK(vsc_rb_size(&rb) == 2);
  return 0;
}

/// @brief 用例 7：空缓冲 peek 返回 0 段视图。
static int test_peek_empty(void) {
  vsc_spsc_ring_t rb;
  uint8_t storage[8];
  CHECK(vsc_spsc_ring_init(&rb, storage, sizeof(storage)) == 0);
  vsc_rb_view_t views[2];
  CHECK(vsc_rb_peek(&rb, 0, views) == 0);
  return 0;
}

/// @brief 用例 8：advance 越界报错。
static int test_advance_out_of_range(void) {
  vsc_spsc_ring_t rb;
  uint8_t storage[8];
  CHECK(vsc_spsc_ring_init(&rb, storage, sizeof(storage)) == 0);

  const uint8_t in[] = {1, 2};
  CHECK(vsc_rb_write(&rb, in, sizeof(in)) == 2);
  CHECK(vsc_rb_advance(&rb, 3) == -1);
  return 0;
}

/// @brief 用例 9：容量必须为正数。
static int test_invalid_capacity(void) {
  vsc_spsc_ring_t rb;
  uint8_t storage[4];
  CHECK(vsc_spsc_ring_init(&rb, storage, 0) == -1);
  return 0;
}

/// @brief 用例 10（压力）：跨回绕的写读顺序性。
///
/// 以 37 字节为写步、53 字节为读步搬运 100000 字节递增序列，逐字节校验。
static int test_streaming_order(void) {
  vsc_spsc_ring_t rb;
  static uint8_t storage[1024];
  CHECK(vsc_spsc_ring_init(&rb, storage, sizeof(storage)) == 0);

  static uint8_t in[37];
  static uint8_t out[53];
  uint64_t expect = 0;
  uint64_t produced = 0;
  uint64_t consumed = 0;
  while (consumed < 100000) {
    if (produced < 100000) {
      for (uint32_t i = 0; i < sizeof(in) && produced < 100000; ++i) {
        in[i] = (uint8_t)(produced + i);
      }
      const uint32_t n = (100000 - produced < sizeof(in))
                             ? (uint32_t)(100000 - produced)
                             : (uint32_t)sizeof(in);
      (void)vsc_rb_write(&rb, in, n);
      produced += n;
    }
    const uint32_t got = vsc_rb_read(&rb, out, sizeof(out));
    for (uint32_t i = 0; i < got; ++i) {
      CHECK(out[i] == (uint8_t)expect);
      ++expect;
      ++consumed;
    }
  }
  CHECK(vsc_rb_is_empty(&rb));
  return 0;
}

int main(void) {
  int failures = 0;
  failures += test_roundtrip();
  failures += test_wraparound_peek_two_views();
  failures += test_overwrite_counts_dropped();
  failures += test_oversized_write_keeps_tail();
  failures += test_peek_limited_advance_batched();
  failures += test_read_truncates_to_dst();
  failures += test_peek_empty();
  failures += test_advance_out_of_range();
  failures += test_invalid_capacity();
  failures += test_streaming_order();
  if (failures != 0) {
    fprintf(stderr, "%d test(s) failed\n", failures);
    return 1;
  }
  printf("all C ring buffer tests passed\n");
  return 0;
}
