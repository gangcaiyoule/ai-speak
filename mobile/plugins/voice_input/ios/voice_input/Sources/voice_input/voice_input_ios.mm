/// @file voice_input_ios.mm
/// @brief iOS 采集实现：RemoteIO 输入流 + AVAudioSession 激活壳（R5）。
///
/// 职责划分（对照 Apple Audio Unit Hosting Guide）：
/// - AVAudioSession 激活是 iOS 规定动作，属 ObjC++ 壳（本文件）；
/// - AudioUnit 本身是 C API，回调直接写 SPSC 环缓（同一份
///   spsc_ring_buffer.h），实时线程纪律与 Android 侧一致：
///   回调内无分配、无锁、无阻塞，只 memcpy 进环缓。
///
/// @note 采样率是请求不是保证：激活后回读 actualSampleRate，经
///       vi_format 交给上层（接口文档 8.1 节）。
#import <AVFoundation/AVFoundation.h>
#import <AudioToolbox/AudioToolbox.h>

#include <atomic>
#include <cstring>
#include <mutex>
#include <thread>
#include <vector>

#include "spsc_ring_buffer.h"
#include "voice_input.h"

namespace {

vsc_spsc_ring_t g_ring;          ///< 共享环缓（生产者=render 回调线程）。
std::vector<uint8_t> g_storage;  ///< 环缓存储区。
std::vector<uint8_t> g_render;   ///< 预分配的渲染缓冲（回调内零分配）。
AudioUnit g_unit = nullptr;      ///< RemoteIO 实例。
AudioStreamBasicDescription g_asbd = {};  ///< 协商后的输入格式。
std::atomic<bool> g_running{false};
std::atomic<int32_t> g_active_callbacks{0}; ///< 正在执行的渲染回调计数（退出同步）。
std::mutex g_control_mutex;      ///< 保护 start/stop 控制面。

/// @brief 输入渲染回调：AudioUnitRender 取 PCM 后写入环缓。
OSStatus InputRenderCallback(void * /*inRefCon*/,
                             AudioUnitRenderActionFlags *ioActionFlags,
                             const AudioTimeStamp *inTimeStamp,
                             UInt32 /*inBusNumber*/,
                             UInt32 inNumberFrames,
                             AudioBufferList * /*ioData*/) {
  if (!g_running.load(std::memory_order_acquire)) {
    return noErr;
  }
  g_active_callbacks.fetch_add(1, std::memory_order_acq_rel);

  const UInt32 needed_bytes = inNumberFrames * 2;
  if (needed_bytes > g_render.size() || g_unit == nullptr) {
    g_active_callbacks.fetch_sub(1, std::memory_order_acq_rel);
    return -1;
  }

  AudioBufferList buffer_list;
  buffer_list.mNumberBuffers = 1;
  buffer_list.mBuffers[0].mNumberChannels = 1;
  buffer_list.mBuffers[0].mDataByteSize = needed_bytes;  ///< I16 单声道，字节数。
  buffer_list.mBuffers[0].mData = g_render.data();

  OSStatus status =
      AudioUnitRender(g_unit, ioActionFlags, inTimeStamp, 1, inNumberFrames,
                      &buffer_list);
  if (status == noErr) {
    const uint32_t bytes = buffer_list.mBuffers[0].mDataByteSize;
    if (bytes > 0) {
      vsc_rb_write(&g_ring, static_cast<const uint8_t *>(
                                buffer_list.mBuffers[0].mData),
                   bytes);
    }
  }
  g_active_callbacks.fetch_sub(1, std::memory_order_acq_rel);
  return status;
}

/// @brief 配置并激活音频会话：PlayAndRecord + VoiceChat，请求 16kHz。
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

/// @brief 构建 RemoteIO 输入单元并初始化。
OSStatus BuildInputUnit(int32_t requested_rate) {
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

  // 打开输入侧（element 1, input scope）。
  UInt32 enable_input = 1;
  status = AudioUnitSetProperty(g_unit, kAudioOutputUnitProperty_EnableIO,
                                kAudioUnitScope_Input, 1, &enable_input,
                                sizeof(enable_input));
  if (status != noErr) {
    return status;
  }

  // 上行格式请求：I16 单声道；实际格式 initialize 后回读。
  AudioStreamBasicDescription asbd = {};
  asbd.mSampleRate = requested_rate > 0 ? (Float64)requested_rate
                                        : 16000.0;
  asbd.mFormatID = kAudioFormatLinearPCM;
  asbd.mFormatFlags = kAudioFormatFlagIsSignedInteger | kAudioFormatFlagIsPacked;
  asbd.mBytesPerPacket = 2;
  asbd.mFramesPerPacket = 1;
  asbd.mBytesPerFrame = 2;
  asbd.mChannelsPerFrame = 1;
  asbd.mBitsPerChannel = 16;
  status = AudioUnitSetProperty(g_unit, kAudioUnitProperty_StreamFormat,
                                kAudioUnitScope_Output, 1, &asbd,
                                sizeof(asbd));
  if (status != noErr) {
    return status;
  }

  // 挂输入回调。
  AURenderCallbackStruct callback = {};
  callback.inputProc = InputRenderCallback;
  callback.inputProcRefCon = nullptr;
  status = AudioUnitSetProperty(g_unit, kAudioOutputUnitProperty_SetInputCallback,
                                kAudioUnitScope_Global, 0, &callback,
                                sizeof(callback));
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
                                kAudioUnitScope_Output, 1, &g_asbd, &size);
  if (status != noErr) {
    return status;
  }

  // 预分配渲染缓冲：按协商采样率 100ms 帧量封顶，回调内零分配。
  const UInt32 max_frames =
      (UInt32)(g_asbd.mSampleRate / 10.0) + 1;
  g_render.assign((size_t)max_frames * g_asbd.mBytesPerFrame, 0);
  return noErr;
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
  g_render.clear();
  g_render.shrink_to_fit();
  g_storage.clear();
  g_storage.shrink_to_fit();
}

}  // namespace

int32_t vi_start(const vi_config_t *cfg) {
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
  const uint32_t capacity =
      static_cast<uint32_t>(rate / 1000 * 2 * cfg->capacity_ms);
  g_storage.assign(capacity > 0 ? capacity : 1u, 0);
  vsc_spsc_ring_init(&g_ring, g_storage.data(),
                     static_cast<uint32_t>(g_storage.size()));

  OSStatus status = BuildInputUnit(cfg->sample_rate);
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

int32_t vi_stop(void) {
  std::lock_guard<std::mutex> lock(g_control_mutex);
  if (!g_running.load(std::memory_order_acquire)) {
    return 0;  // 幂等。
  }
  TeardownLocked();
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
