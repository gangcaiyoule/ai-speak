/// @file voice_output_ios.mm
/// @brief iOS 播放实现：RemoteIO 输出侧渲染回调消费播放队列（R7）。
///
/// 与采集端（voice_input_ios.mm）共用同一套职责划分与实时纪律：
/// - AVAudioSession 激活是 ObjC++ 壳的规定动作（PlayAndRecord + VoiceChat，
///   与采集侧一致，保证全双工时两端共享同一路会话配置）；
/// - AudioUnit 是 C API，渲染回调从 playback_queue 取数写入 ioData，
///   空则 memset 静音：回调内无分配、无锁、无阻塞；
/// - 欠载策略（预缓冲/迟滞）与统计在 playback_queue.h 状态机内完成。
///
/// @note 采样率是请求不是保证：initialize 后回读 element 0 输入侧实际格式，
///       经 vo_format 交给上层（接口文档 8.1 节）。
#import <AVFoundation/AVFoundation.h>
#import <AudioToolbox/AudioToolbox.h>

#include <atomic>
#include <cstring>
#include <mutex>
#include <thread>
#include <vector>

#include "playback_queue.h"
#include "voice_output.h"

namespace {

/// @brief 播放默认预缓冲阈值，单位毫秒。
constexpr int32_t kDefaultPrimingMs = 40;

vsc_spsc_ring_t g_ring;          ///< 播放环缓（生产者=Dart 线程）。
vsc_pbq_t g_pbq;                 ///< 播放队列状态机（消费者=渲染回调线程）。
std::vector<uint8_t> g_storage;  ///< 环缓存储区。
AudioUnit g_unit = nullptr;      ///< RemoteIO 实例。
AudioStreamBasicDescription g_asbd = {};  ///< 协商后的输出格式。
std::atomic<bool> g_running{false};
std::atomic<int32_t> g_active_callbacks{0};  ///< 正在执行的渲染回调计数（退出同步）。
std::mutex g_control_mutex;      ///< 保护 start/stop 控制面。

/// @brief 输出渲染回调：从播放队列取数填 ioData，缺口补静音。
OSStatus OutputRenderCallback(void * /*inRefCon*/,
                              AudioUnitRenderActionFlags *ioActionFlags,
                              const AudioTimeStamp * /*inTimeStamp*/,
                              UInt32 /*inBusNumber*/,
                              UInt32 /*inNumberFrames*/,
                              AudioBufferList *ioData) {
  if (!g_running.load(std::memory_order_acquire) || ioData == nullptr ||
      ioData->mNumberBuffers < 1) {
    return noErr;
  }
  g_active_callbacks.fetch_add(1, std::memory_order_acq_rel);

  AudioBuffer &buffer = ioData->mBuffers[0];
  auto *out = static_cast<uint8_t *>(buffer.mData);
  const uint32_t needed = buffer.mDataByteSize;

  vsc_rb_view_t views[2];
  const uint32_t got = vsc_pbq_acquire(&g_pbq, needed, views);
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
    vsc_pbq_commit(&g_pbq, got);
  }
  if (copied < needed) {
    memset(out + copied, 0, needed - copied);  ///< 欠载/预缓冲期静音。
    // 提示系统本轮回调为静音，便于省电与日志归因（可选标记）。
    if (ioActionFlags != nullptr && got == 0) {
      *ioActionFlags |= kAudioUnitRenderAction_OutputIsSilence;
    }
  }

  g_active_callbacks.fetch_sub(1, std::memory_order_acq_rel);
  return noErr;
}

/// @brief 配置并激活音频会话：PlayAndRecord + VoiceChat，请求 16kHz。
/// 与采集端语义一致；纯播放场景下该类别同样可正常出声。
OSStatus ActivateVoiceSession(NSError **out_error, int32_t requested_rate) {
  AVAudioSession *session = [AVAudioSession sharedInstance];
  [session setCategory:AVAudioSessionCategoryPlayAndRecord
                  mode:AVAudioSessionModeVoiceChat
               options:AVAudioSessionCategoryOptionDefaultToSpeaker |
                       AVAudioSessionCategoryOptionAllowBluetooth
                 error:out_error];
  if (*out_error != nil) {
    return -1;
  }
  if (requested_rate > 0) {
    [session setPreferredSampleRate:(double)requested_rate error:out_error];
    if (*out_error != nil) {
      return -1;
    }
  }
  [session setActive:YES error:out_error];
  if (*out_error != nil) {
    return -1;
  }
  return noErr;
}

/// @brief 构建 RemoteIO 输出单元并初始化。
OSStatus BuildOutputUnit(int32_t requested_rate) {
  AudioComponentDescription desc = {};
  desc.componentType = kAudioUnitType_Output;
  desc.componentSubType = kAudioUnitSubType_RemoteIO;
  desc.componentManufacturer = kAudioUnitManufacturer_Apple;

  AudioComponent component = AudioComponentFindNext(nullptr, &desc);
  if (component == nullptr) {
    return -1;
  }
  OSStatus status = AudioComponentInstanceNew(component, &g_unit);
  if (status != noErr) {
    return status;
  }

  // 输出侧（element 0）默认启用；挂渲染回调到全局 scope。
  AURenderCallbackStruct callback = {};
  callback.inputProc = OutputRenderCallback;
  callback.inputProcRefCon = nullptr;
  status = AudioUnitSetProperty(g_unit, kAudioUnitProperty_SetRenderCallback,
                                kAudioUnitScope_Global, 0, &callback,
                                sizeof(callback));
  if (status != noErr) {
    return status;
  }

  // 播放格式请求：I16 单声道；实际格式 initialize 后回读。
  AudioStreamBasicDescription asbd = {};
  asbd.mSampleRate = requested_rate > 0 ? (Float64)requested_rate : 16000.0;
  asbd.mFormatID = kAudioFormatLinearPCM;
  asbd.mFormatFlags = kAudioFormatFlagIsSignedInteger | kAudioFormatFlagIsPacked;
  asbd.mBytesPerPacket = 2;
  asbd.mFramesPerPacket = 1;
  asbd.mBytesPerFrame = 2;
  asbd.mChannelsPerFrame = 1;
  asbd.mBitsPerChannel = 16;
  status = AudioUnitSetProperty(g_unit, kAudioUnitProperty_StreamFormat,
                                kAudioUnitScope_Input, 0, &asbd,
                                sizeof(asbd));
  if (status != noErr) {
    return status;
  }

  status = AudioUnitInitialize(g_unit);
  if (status != noErr) {
    return status;
  }

  // 回读协商后的实际格式（请求不是保证）。
  UInt32 size = sizeof(g_asbd);
  status = AudioUnitGetProperty(g_unit, kAudioUnitProperty_StreamFormat,
                                kAudioUnitScope_Input, 0, &g_asbd, &size);
  return status;
}

/// @brief 释放音频单元与环缓存储（持锁调用）。
void TeardownLocked() {
  g_running.store(false, std::memory_order_release);
  if (g_unit != nullptr) {
    AudioOutputUnitStop(g_unit);
    AudioUnitUninitialize(g_unit);
    AudioComponentInstanceDispose(g_unit);
    g_unit = nullptr;
  }
  // 等待可能正在执行的渲染回调彻底退出，避免野指针访问（Use-After-Free）。
  while (g_active_callbacks.load(std::memory_order_acquire) > 0) {
    std::this_thread::yield();
  }
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

  // 会话激活放最前：采样率请求作用于会话层。
  NSError *error = nil;
  if (ActivateVoiceSession(&error, cfg->sample_rate) != noErr) {
    return -2;
  }

  int32_t rate = cfg->sample_rate > 0 ? cfg->sample_rate : 16000;
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

  OSStatus status = BuildOutputUnit(cfg->sample_rate);
  if (status != noErr) {
    TeardownLocked();
    return -2;
  }

  g_running.store(true, std::memory_order_release);
  status = AudioOutputUnitStart(g_unit);
  if (status != noErr) {
    TeardownLocked();
    return -2;
  }

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
  // 与采集端 vi_read 同理：Dart 主 isolate 顺序调用，start/stop 不会与
  // write 并发；渲染回调线程只消费不触碰存储区生命周期。
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
  if (g_running.load(std::memory_order_acquire)) {
    *sample_rate = (int32_t)g_asbd.mSampleRate;
    *channels = (int32_t)g_asbd.mChannelsPerFrame;
    *bits = (int32_t)g_asbd.mBitsPerChannel;
  } else {
    *sample_rate = 0;
    *channels = 0;
    *bits = 0;
  }
  return 0;
}
