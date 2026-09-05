/// @file bindings_common.dart
/// @brief 采集/播放绑定层公共契约：接口与异常（全平台共享）。
///
/// 真实 FFI 实现见 `bindings_native.dart`（仅 dart.library.io 平台加载）；
/// web 平台加载 `bindings_unsupported.dart` 桩实现（dart:ffi 不存在于 web，
/// 浏览器侧麦克风/扬声器走 voice_stream 模块的 WebMicSource/WebAudioSink，
/// 不经本插件）。
library;

import 'dart:typed_data';

/// @brief 采集端原生绑定接口（与 voice_input.h C ABI 一一对应）。
///
/// 抛错语义：start 失败抛 [VoiceInputException]；其余方法不抛（读空
/// 返回 0，幂等 stop 返回即完成）。
abstract interface class VoiceInputBindings {
  /// @brief 打开输入流并启动回调（对应 vi_start）。
  ///
  /// @param sampleRateHz 请求采样率。
  /// @param capacityMs 环缓容量按毫秒预算。
  void start({required int sampleRateHz, required int capacityMs});

  /// @brief 停止并释放流（对应 vi_stop，幂等）。
  void stop();

  /// @brief 拷贝读出采集字节（对应 vi_read）。
  ///
  /// @param destination 目标缓冲。
  /// @return 实际读出字节数。
  int readInto(Uint8List destination);

  /// @brief 累计丢旧字节数（对应 vi_dropped）。
  int droppedBytes();

  /// @brief 协商后的实际格式（对应 vi_format）；未启动时为 null。
  ({int sampleRateHz, int channelCount, int bitsPerSample})?
      negotiatedFormat();
}

/// @brief 原生层返回的错误。
class VoiceInputException implements Exception {
  /// @brief 构造异常。
  ///
  /// @param code vi_start 返回的错误码。
  /// @param message 人类可读描述。
  const VoiceInputException(this.code, this.message);

  /// @brief 错误码（voice_input.h 注释）。
  final int code;

  /// @brief 人类可读描述。
  final String message;

  @override
  String toString() => 'VoiceInputException($code): $message';
}

/// @brief 播放端原生绑定接口（与 voice_output.h C ABI 一一对应）。
///
/// 抛错语义：start 失败抛 [VoiceOutputException]；其余方法不抛（write
/// 返回实际接受字节数，幂等 stop 返回即完成）。
abstract interface class VoiceOutputBindings {
  /// @brief 打开输出流并启动回调消费播放队列（对应 vo_start）。
  ///
  /// @param sampleRateHz 请求采样率。
  /// @param capacityMs 播放缓冲容量按毫秒预算。
  /// @param primingMs 预缓冲阈值；负值取原生默认。
  void start({
    required int sampleRateHz,
    required int capacityMs,
    required int primingMs,
  });

  /// @brief 停止并释放流（对应 vo_stop，幂等）。
  void stop();

  /// @brief 拷贝写入待播放 PCM（对应 vo_write，全有或全无）。
  ///
  /// @param source 待写入 PCM 字节。
  /// @return 实际接受的字节数：全量或 0（缓冲满整块拒绝）。
  int write(Uint8List source);

  /// @brief 当前缓冲中的可播放字节数（对应 vo_buffered）。
  int bufferedBytes();

  /// @brief 欠载静音累计字节数（对应 vo_underrun）。
  int underrunBytes();

  /// @brief 空间不足被整块拒绝的累计字节数（对应 vo_dropped）。
  int droppedBytes();

  /// @brief 协商后的实际格式（对应 vo_format）；未启动时为 null。
  ({int sampleRateHz, int channelCount, int bitsPerSample})?
      negotiatedFormat();
}

/// @brief 原生播放层返回的错误。
class VoiceOutputException implements Exception {
  /// @brief 构造异常。
  ///
  /// @param code vo_start 返回的错误码。
  /// @param message 人类可读描述。
  const VoiceOutputException(this.code, this.message);

  /// @brief 错误码（voice_output.h 注释）。
  final int code;

  /// @brief 人类可读描述。
  final String message;

  @override
  String toString() => 'VoiceOutputException($code): $message';
}
