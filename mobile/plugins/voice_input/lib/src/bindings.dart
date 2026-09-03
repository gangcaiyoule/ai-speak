/// @file bindings.dart
/// @brief vi_* C ABI 的 dart:ffi 绑定与可注入接口。
///
/// 真实绑定 [FfiVoiceInputBindings] 按 Platform 选择库句柄：Android 动态
/// 打开 libvoice_input.so；iOS 侧原生库静态链入主二进制，用
/// DynamicLibrary.process()。测试用假绑定实现 [VoiceInputBindings] 即可，
/// 不需要任何原生环境。
library;

import 'dart:ffi';
import 'dart:io';
import 'dart:typed_data';

import 'package:ffi/ffi.dart';

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

/// @brief dart:ffi 真实绑定。
final class FfiVoiceInputBindings implements VoiceInputBindings {
  /// @brief 构造绑定；缺省按平台自动选择库句柄。
  ///
  /// @param library 已打开的动态库；测试可注入。
  FfiVoiceInputBindings({DynamicLibrary? library})
      : _lib = library ?? _defaultLibrary() {
    _viStart =
        _lib.lookupFunction<ViStartNative, ViStartDart>('vi_start');
    _viStop = _lib.lookupFunction<ViStopNative, ViStopDart>('vi_stop');
    _viRead = _lib.lookupFunction<ViReadNative, ViReadDart>('vi_read');
    _viDropped =
        _lib.lookupFunction<ViDroppedNative, ViDroppedDart>('vi_dropped');
    _viFormat =
        _lib.lookupFunction<ViFormatNative, ViFormatDart>('vi_format');
  }

  static DynamicLibrary _defaultLibrary() {
    if (Platform.isAndroid) {
      return DynamicLibrary.open('libvoice_input.so');
    }
    return DynamicLibrary.process(); // iOS：静态链入主二进制。
  }

  final DynamicLibrary _lib;

  late final ViStartDart _viStart;
  late final ViStopDart _viStop;
  late final ViReadDart _viRead;
  late final ViDroppedDart _viDropped;
  late final ViFormatDart _viFormat;

  Pointer<ViConfig>? _config;
  Pointer<Uint8>? _readBuffer;
  int _readBufferBytes = 0;
  Pointer<Int32>? _formatOut;

  @override
  void start({required int sampleRateHz, required int capacityMs}) {
    final config = _config ??= malloc<ViConfig>();
    config.ref.sampleRate = sampleRateHz;
    config.ref.capacityMs = capacityMs;
    final code = _viStart(config);
    if (code != 0) {
      throw VoiceInputException(code, switch (code) {
        -1 => '采集已在运行',
        -2 => '输入流打开失败',
        -3 => '启动参数非法',
        _ => 'vi_start 未知错误码',
      });
    }
  }

  @override
  void stop() {
    if (_config != null) {
      _viStop();
    }
  }

  @override
  int readInto(Uint8List destination) {
    if (destination.isEmpty) {
      return 0;
    }
    if (_readBuffer == null || _readBufferBytes < destination.length) {
      if (_readBuffer != null) {
        malloc.free(_readBuffer!);
      }
      _readBuffer = malloc<Uint8>(destination.length);
      _readBufferBytes = destination.length;
    }
    final n = _viRead(_readBuffer!, destination.length);
    if (n <= 0) {
      return 0;
    }
    destination.setAll(0, _readBuffer!.asTypedList(n));
    return n;
  }

  @override
  int droppedBytes() => _viDropped();

  @override
  ({int sampleRateHz, int channelCount, int bitsPerSample})?
      negotiatedFormat() {
    final out = _formatOut ??= malloc<Int32>(3);
    final code = _viFormat(out, out.elementAt(1), out.elementAt(2));
    if (code != 0 || out[0] == 0) {
      return null;
    }
    return (
      sampleRateHz: out[0],
      channelCount: out[1],
      bitsPerSample: out[2],
    );
  }
}

/// @brief vi_config_t 的 FFI 视图（与 voice_input.h 布局一致）。
final class ViConfig extends Struct {
  /// @brief 请求采样率，单位 Hz。
  @Int32()
  external int sampleRate;

  /// @brief 环缓容量按毫秒预算。
  @Int32()
  external int capacityMs;
}

// vi_* C ABI 的原生/Dart 签名对。
typedef ViStartNative = Int32 Function(Pointer<ViConfig>);
typedef ViStartDart = int Function(Pointer<ViConfig>);
typedef ViStopNative = Int32 Function();
typedef ViStopDart = int Function();
typedef ViReadNative = Int32 Function(Pointer<Uint8>, Int32);
typedef ViReadDart = int Function(Pointer<Uint8>, int);
typedef ViDroppedNative = Uint64 Function();
typedef ViDroppedDart = int Function();
typedef ViFormatNative = Int32 Function(
    Pointer<Int32>, Pointer<Int32>, Pointer<Int32>);
typedef ViFormatDart = int Function(
    Pointer<Int32>, Pointer<Int32>, Pointer<Int32>);
