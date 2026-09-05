/// @file voice_input_oboe.cpp
/// @brief Android 采集实现：Oboe 输入流回调写入 SPSC 环缓（R4）。
///
/// 实时约束（对应 Oboe FullGuide Do's/Don'ts）：
/// - 回调线程只做 memcpy 进环缓与丢旧计数，无锁、无分配、无阻塞；
/// - 回调内不 read/write 流、不 stop/close（stop/close 由 Dart 线程经
///   vi_stop 触发，Oboe 内部保证安全）；
/// - 采样率/格式以打开流后的实际值为准（请求不是保证），经 vi_format 回读。
///
/// 输入预设默认 VoiceRecognition（低延迟语音识别优化），即语音对话场景
/// 应保持的预设。
#include "voice_input.h"

#include <atomic>
#include <cstring>
#include <mutex>
#include <new>
#include <vector>

#include <oboe/Oboe.h>

#include "spsc_ring_buffer.h"

namespace {

/// @brief 采集回调：仅把 PCM 写入环缓（丢旧由环缓语义消化）。
class InputCallback final : public oboe::AudioStreamCallback {
 public:
  explicit InputCallback(vsc_spsc_ring_t *ring) : ring_(ring) {}

  oboe::DataCallbackResult onAudioReady(oboe::AudioStream *,
                                        void *audioData, int32_t numFrames) override {
    const int32_t frame_bytes = kBytesPerSample * channel_count_.load(
                                    std::memory_order_acquire);
    if (frame_bytes > 0 && audioData != nullptr && numFrames > 0) {
      vsc_rb_write(ring_, static_cast<const uint8_t *>(audioData),
                   static_cast<uint32_t>(numFrames * frame_bytes));
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

  oboe::Result lastError() const {
    return error_.load(std::memory_order_acquire);
  }

 private:
  static constexpr int kBytesPerSample = 2; ///< I16 位深对应字节数。
  vsc_spsc_ring_t *ring_;
  std::atomic<int32_t> channel_count_{1};
  std::atomic<oboe::Result> error_{oboe::Result::OK};
};

vsc_spsc_ring_t g_ring;                        ///< 共享环缓（生产者=回调线程）。
std::vector<uint8_t> g_storage;                ///< 环缓存储区。
oboe::ManagedStream g_stream;                  ///< Oboe 输入流。
InputCallback *g_callback = nullptr;           ///< 回调实例（流关闭时释放）。
std::atomic<bool> g_running{false};
std::mutex g_control_mutex;                    ///< 保护 start/stop 控制面。

}  // namespace

int32_t vi_start(const vi_config_t *cfg) {
  if (cfg == nullptr || cfg->capacity_ms <= 0) {
    return -3;
  }
  std::lock_guard<std::mutex> lock(g_control_mutex);
  if (g_running.load(std::memory_order_acquire)) {
    return -1;
  }

  // 容量按毫秒预算：请求采样率下 16bit 单声道字节数；下限 100ms。
  int32_t rate = cfg->sample_rate;
  if (rate <= 0) {
    rate = 48000;  // 未指定时按 Oboe 常见默认预估，打开后以实际格式回读。
  }
  const uint32_t capacity =
      static_cast<uint32_t>(rate / 1000 * 2 * cfg->capacity_ms);
  g_storage.assign(capacity > 0 ? capacity : 1u, 0);
  vsc_spsc_ring_init(&g_ring, g_storage.data(),
                     static_cast<uint32_t>(g_storage.size()));

  g_callback = new (std::nothrow) InputCallback(&g_ring);
  if (g_callback == nullptr) {
    return -2;
  }

  oboe::AudioStreamBuilder builder;
  builder.setDirection(oboe::Direction::Input)
      .setPerformanceMode(oboe::PerformanceMode::LowLatency)
      .setSharingMode(oboe::SharingMode::Exclusive)
      .setFormat(oboe::AudioFormat::I16)
      .setChannelCount(oboe::ChannelCount::Mono)
      .setInputPreset(oboe::InputPreset::VoiceRecognition)
      .setSampleRate(rate > 0 ? rate : 0)
      .setCallback(g_callback);

  oboe::Result result = builder.openManagedStream(g_stream);
  if (result != oboe::Result::OK) {
    // 很多设备或模拟器对录音输入不支持 Exclusive 独占模式，运行时降级尝试 Shared 共享模式。
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
    g_stream->close();
    g_stream.reset();
    delete g_callback;
    g_callback = nullptr;
    g_storage.clear();
    g_storage.shrink_to_fit();
    return -2;
  }

  g_running.store(true, std::memory_order_release);
  return 0;
}

int32_t vi_stop(void) {
  std::lock_guard<std::mutex> lock(g_control_mutex);
  if (!g_running.load(std::memory_order_acquire)) {
    return 0;  // 幂等。
  }
  if (g_stream) {
    g_stream->stop();
    g_stream->close();
    g_stream.reset();
  }
  delete g_callback;
  g_callback = nullptr;
  g_running.store(false, std::memory_order_release);
  return 0;
}

int32_t vi_read(uint8_t *dst, int32_t max_bytes) {
  if (dst == nullptr || max_bytes <= 0) {
    return -1;
  }
  return static_cast<int32_t>(
      vsc_rb_read(&g_ring, dst, static_cast<uint32_t>(max_bytes)));
}

uint64_t vi_dropped(void) { return vsc_rb_dropped(&g_ring); }

int32_t vi_format(int32_t *sample_rate, int32_t *channels, int32_t *bits) {
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
