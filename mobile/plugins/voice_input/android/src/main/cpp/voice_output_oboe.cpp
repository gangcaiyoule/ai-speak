/// @file voice_output_oboe.cpp
/// @brief Android 播放实现：Oboe 输出流回调消费播放队列（R7）。
///
/// 实时约束（对应 Oboe FullGuide Do's/Don'ts，与采集侧 voice_input_oboe.cpp
/// 同一套纪律，仅生产者/消费者角色反转）：
/// - 回调线程是播放队列的唯一消费者：acquire → memcpy → commit，空则
///   memset 静音；无锁、无分配、无阻塞；
/// - 回调内不 read/write 流、不 stop/close（由 Dart 线程经 vo_stop 触发）；
/// - 采样率/格式以打开流后的实际值为准（请求不是保证），经 vo_format 回读。
///
/// 欠载策略在 playback_queue.h 状态机内：启动预缓冲（默认 40ms）→ 正常
/// 消费 → 取空回退预缓冲（迟滞）；静音字节数由状态机计入 vo_underrun。
///
/// 播放路由：Usage::Media + ContentType::Speech——不劫持设备路由（采集侧
/// VoiceRecognition 预设不影响播放路由），R8 真机联调时如需与通话链路
/// 对齐再评估切到 Usage::VoiceCommunication。
#include "voice_output.h"

#include <atomic>
#include <cstring>
#include <mutex>
#include <new>
#include <vector>

#include <oboe/Oboe.h>

#include "playback_queue.h"

namespace {

/// @brief 播放默认预缓冲阈值，单位毫秒。
constexpr int32_t kDefaultPrimingMs = 40;
/// @brief I16 位深对应字节数。
constexpr int kBytesPerSample = 2;

/// @brief 播放回调：从播放队列取数，缺口补静音。
class OutputCallback final : public oboe::AudioStreamCallback {
 public:
  explicit OutputCallback(vsc_pbq_t *pbq) : pbq_(pbq) {}

  oboe::DataCallbackResult onAudioReady(oboe::AudioStream *,
                                        void *audioData,
                                        int32_t numFrames) override {
    const int32_t frame_bytes = kBytesPerSample * channel_count_.load(
                                    std::memory_order_acquire);
    if (frame_bytes <= 0 || audioData == nullptr || numFrames <= 0) {
      return oboe::DataCallbackResult::Continue;
    }
    auto *out = static_cast<uint8_t *>(audioData);
    const uint32_t needed =
        static_cast<uint32_t>(numFrames) * static_cast<uint32_t>(frame_bytes);

    vsc_rb_view_t views[2];
    const uint32_t got = vsc_pbq_acquire(pbq_, needed, views);
    uint32_t copied = 0;
    // peek 只填充实际返回的视图段；单段时 views[0].len == got，
    // 复制完即 break，不会触碰未初始化的 views[1]。
    for (int i = 0; i < 2 && copied < got; ++i) {
      const uint32_t len =
          views[i].len < got - copied ? views[i].len : got - copied;
      memcpy(out + copied, views[i].data, len);
      copied += len;
    }
    if (got > 0) {
      vsc_pbq_commit(pbq_, got);
    }
    if (copied < needed) {
      memset(out + copied, 0, needed - copied);  ///< 欠载/预缓冲期静音。
    }
    return oboe::DataCallbackResult::Continue;
  }

  /// @brief 流错误后不在此回调内 stop/close（违规），仅记录待处理。
  void onErrorAfterClose(oboe::AudioStream *, oboe::Result error) override {
    error_.store(error, std::memory_order_release);
  }

  void setChannelCount(int32_t channels) {
    channel_count_.store(channels, std::memory_order_release);
  }

 private:
  vsc_pbq_t *pbq_;
  std::atomic<int32_t> channel_count_{1};
  std::atomic<oboe::Result> error_{oboe::Result::OK};
};

vsc_spsc_ring_t g_ring;              ///< 播放环缓（生产者=Dart 线程）。
vsc_pbq_t g_pbq;                     ///< 播放队列状态机（消费者=回调线程）。
std::vector<uint8_t> g_storage;      ///< 环缓存储区。
oboe::ManagedStream g_stream;        ///< Oboe 输出流。
OutputCallback *g_callback = nullptr;  ///< 回调实例（流关闭时释放）。
std::atomic<bool> g_running{false};
std::mutex g_control_mutex;          ///< 保护 start/stop 控制面。

/// @brief 释放流与回调实例（持锁调用）。
void TeardownLocked() {
  g_running.store(false, std::memory_order_release);
  if (g_stream) {
    g_stream->stop();
    g_stream->close();
    g_stream.reset();
  }
  delete g_callback;
  g_callback = nullptr;
  g_storage.clear();
  g_storage.shrink_to_fit();
}

}  // namespace

int32_t vo_start(const vo_config_t *cfg) {
  if (cfg == nullptr || cfg->capacity_ms <= 0) {
    return -3;
  }
  std::lock_guard<std::mutex> lock(g_control_mutex);
  if (g_running.load(std::memory_order_acquire)) {
    return -1;
  }

  // 容量与预缓冲阈值按毫秒预算：请求采样率下 16bit 单声道字节数。
  int32_t rate = cfg->sample_rate;
  if (rate <= 0) {
    rate = 48000;  // 未指定时按 Oboe 常见默认预估，打开后以实际格式回读。
  }
  const int32_t priming_ms =
      cfg->priming_ms < 0 ? kDefaultPrimingMs : cfg->priming_ms;
  const uint32_t capacity =
      static_cast<uint32_t>(rate / 1000 * 2 * cfg->capacity_ms);
  const uint32_t priming_bytes =
      static_cast<uint32_t>(rate / 1000 * 2 * priming_ms);
  g_storage.assign(capacity > 0 ? capacity : 1u, 0);
  vsc_spsc_ring_init(&g_ring, g_storage.data(),
                     static_cast<uint32_t>(g_storage.size()));
  // 预缓冲阈值按实际容量截断（容量毫秒数小于阈值时不至于永不启动）。
  vsc_pbq_init(&g_pbq, &g_ring,
               priming_bytes > capacity ? capacity : priming_bytes);

  g_callback = new (std::nothrow) OutputCallback(&g_pbq);
  if (g_callback == nullptr) {
    g_storage.clear();
    g_storage.shrink_to_fit();
    return -2;
  }

  oboe::AudioStreamBuilder builder;
  builder.setDirection(oboe::Direction::Output)
      .setPerformanceMode(oboe::PerformanceMode::LowLatency)
      .setSharingMode(oboe::SharingMode::Exclusive)
      .setFormat(oboe::AudioFormat::I16)
      .setChannelCount(oboe::ChannelCount::Mono)
      .setUsage(oboe::Usage::Media)
      .setContentType(oboe::ContentType::Speech)
      .setSampleRate(rate > 0 ? rate : 0)
      .setCallback(g_callback);

  oboe::Result result = builder.openManagedStream(g_stream);
  if (result != oboe::Result::OK) {
    // 部分设备独占模式不可用，降级共享模式重试（与采集侧一致）。
    builder.setSharingMode(oboe::SharingMode::Shared);
    result = builder.openManagedStream(g_stream);
  }
  if (result != oboe::Result::OK) {
    delete g_callback;
    g_callback = nullptr;
    g_storage.clear();
    g_storage.shrink_to_fit();
    return -2;
  }

  g_callback->setChannelCount(g_stream->getChannelCount());
  result = g_stream->requestStart();
  if (result != oboe::Result::OK) {
    TeardownLocked();
    return -2;
  }

  g_running.store(true, std::memory_order_release);
  return 0;
}

int32_t vo_stop(void) {
  std::lock_guard<std::mutex> lock(g_control_mutex);
  if (!g_running.load(std::memory_order_acquire)) {
    return 0;  // 幂等。
  }
  TeardownLocked();
  return 0;
}

int32_t vo_write(const uint8_t *src, int32_t len) {
  if (!g_running.load(std::memory_order_acquire)) {
    return 0;
  }
  if (src == nullptr || len <= 0) {
    return 0;
  }
  // 与采集侧 vi_read 同理：Dart 主 isolate 顺序调用，start/stop 不会与
  // write 并发；回调线程只消费不触碰存储区生命周期。
  return static_cast<int32_t>(
      vsc_pbq_write(&g_pbq, src, static_cast<uint32_t>(len)));
}

int32_t vo_buffered(void) {
  if (!g_running.load(std::memory_order_acquire)) {
    return 0;
  }
  return static_cast<int32_t>(vsc_pbq_buffered(&g_pbq));
}

uint64_t vo_underrun(void) { return vsc_pbq_underrun(&g_pbq); }

uint64_t vo_dropped(void) { return vsc_pbq_rejected(&g_pbq); }

int32_t vo_format(int32_t *sample_rate, int32_t *channels, int32_t *bits) {
  if (sample_rate == nullptr || channels == nullptr || bits == nullptr) {
    return -1;
  }
  std::lock_guard<std::mutex> lock(g_control_mutex);
  if (g_stream) {
    *sample_rate = g_stream->getSampleRate();
    *channels = g_stream->getChannelCount();
    *bits = (g_stream->getFormat() == oboe::AudioFormat::I16) ? 16 : 0;
  } else {
    *sample_rate = 0;
    *channels = 0;
    *bits = 0;
  }
  return 0;
}
