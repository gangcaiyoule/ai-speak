/// @file native_audio_sink.dart
/// @brief AudioSink 的平台实现：FFI 写入原生播放队列（R7）。
///
/// 数据面：本类是原生播放队列的唯一生产者（write 全有或全无），音频回调
/// 线程是唯一消费者（取数补静音）。欠载策略（预缓冲/迟滞）与丢帧/欠载统计
/// 在原生 playback_queue 状态机内完成，本层只转发与读计数。
///
/// 生命周期：[AudioSink] 契约没有 start——设备在首次 [write] 时按构造传入
/// 的格式惰性打开；[stop] 释放后再次 [write] 会重新打开（可随时中断与
/// 重连的语义，与 [NativeMicSource] 对齐）。
library;

import 'package:voice_input/voice_input.dart';

import 'contracts.dart';

/// @brief 基于 FFI 播放绑定的 [AudioSink] 实现（Android Oboe / iOS RemoteIO）。
class NativeAudioSink implements AudioSink {
  /// @brief 构造播放端。
  ///
  /// @param format 播放格式；首次 write 时作为 vo_start 的请求采样率
  ///        （实际以协商后格式为准，见接口文档 8.1）。
  /// @param bindings 原生绑定；缺省经绑定层平台工厂创建（原生 FFI 真实
  ///        实现 / web 桩），测试注入假绑定。
  /// @param capacityMs 播放缓冲容量按毫秒预算（默认 1000ms = 16kHz 下 32KiB）。
  /// @param primingMs 预缓冲阈值；避免启动期连续欠载，负值取原生默认。
  NativeAudioSink({
    this.format = const AudioFormat(
      sampleRateHz: 16000,
      channelCount: 1,
      bitsPerSample: 16,
    ),
    VoiceOutputBindings? bindings,
    this.capacityMs = 1000,
    this.primingMs = 40,
  }) : _bindings = bindings ?? createDefaultOutputBindings();

  /// @brief 播放格式（请求值非保证值）。
  final AudioFormat format;

  /// @brief 播放缓冲容量按毫秒预算。
  final int capacityMs;

  /// @brief 预缓冲阈值，单位毫秒。
  final int primingMs;

  final VoiceOutputBindings _bindings;
  bool _opened = false;

  @override
  bool write(AudioFrame frame) {
    final samples = frame.samples;
    if (samples.isEmpty) {
      return true; // 空帧无数据可播，视为接受。
    }
    if (!_opened) {
      _bindings.start(
        sampleRateHz: format.sampleRateHz,
        capacityMs: capacityMs,
        primingMs: primingMs,
      );
      _opened = true;
    }
    // 原生层全有或全无：接受字节数等于帧长即整帧入队。
    return _bindings.write(samples) == samples.length;
  }

  @override
  int get underrunBytes => _bindings.underrunBytes();

  /// @brief 当前缓冲中的可播放字节数（观测口，超出契约的附加统计）。
  int get bufferedBytes => _bindings.bufferedBytes();

  /// @brief 空间不足被整块拒绝的累计字节数（播放侧丢帧统计）。
  int get droppedBytes => _bindings.droppedBytes();

  /// @brief 协商后的实际格式；未启动时为 null。
  ({int sampleRateHz, int channelCount, int bitsPerSample})?
      get negotiatedFormat => _bindings.negotiatedFormat();

  @override
  Future<void> stop() async {
    _bindings.stop(); // 幂等；释放后再次 write 会重新打开设备。
    _opened = false;
  }
}
