/// @file bindings_unsupported.dart
/// @brief web 平台的绑定桩：dart:ffi 不存在于 web，调用即抛错。
///
/// web 平台（Flutter 打包为 web，浏览器内运行）不加载任何原生库；
/// 麦克风/扬声器的 web 平台实现在 voice_stream 模块的
/// WebMicSource（getUserMedia + AudioWorklet），不经本插件的绑定层。
/// 桩实现保证条件导入后符号存在、误用得到明确报错而非编译失败。
library;

import 'dart:typed_data';

import 'bindings_common.dart';

export 'bindings_common.dart'
    show
        VoiceInputBindings,
        VoiceInputException,
        VoiceOutputBindings,
        VoiceOutputException;

/// @brief 平台默认采集绑定工厂（web 上返回不可用桩，误用明确报错）。
VoiceInputBindings createDefaultInputBindings() =>
    UnsupportedVoiceInputBindings();

/// @brief 平台默认播放绑定工厂（web 上返回不可用桩，误用明确报错）。
VoiceOutputBindings createDefaultOutputBindings() =>
    UnsupportedVoiceOutputBindings();

/// @brief 采集绑定桩：web 上不可用。
final class UnsupportedVoiceInputBindings implements VoiceInputBindings {
  Never _fail() => throw UnsupportedError(
      'voice_input 采集绑定仅支持原生平台（Android/iOS/桌面）；'
      'web 平台请使用 voice_stream 模块的 WebMicSource。');

  @override
  void start({required int sampleRateHz, required int capacityMs}) =>
      _fail();

  @override
  void stop() => _fail();

  @override
  int readInto(Uint8List destination) => _fail();

  @override
  int droppedBytes() => _fail();

  @override
  ({int sampleRateHz, int channelCount, int bitsPerSample})?
      negotiatedFormat() => _fail();
}

/// @brief 播放绑定桩：web 上不可用。
final class UnsupportedVoiceOutputBindings implements VoiceOutputBindings {
  Never _fail() => throw UnsupportedError(
      'voice_input 播放绑定仅支持原生平台（Android/iOS/桌面）；'
      'web 平台的播放实现待 R7 web 侧补齐。');

  @override
  void start({
    required int sampleRateHz,
    required int capacityMs,
    required int primingMs,
  }) =>
      _fail();

  @override
  void stop() => _fail();

  @override
  int write(Uint8List source) => _fail();

  @override
  int bufferedBytes() => _fail();

  @override
  int underrunBytes() => _fail();

  @override
  int droppedBytes() => _fail();

  @override
  ({int sampleRateHz, int channelCount, int bitsPerSample})?
      negotiatedFormat() => _fail();
}
