/// @file bindings_native.dart
/// @brief vi_*/vo_* C ABI 的 dart:ffi 真实绑定（仅原生平台加载）。
///
/// 本文件经 `bindings.dart` 的条件导出，只在 dart.library.io 平台
/// （Android/iOS/桌面）参与编译；web 平台走 `bindings_unsupported.dart`
/// 桩（dart:ffi 不存在于 web）。
///
/// 真实绑定按 Platform 选择库句柄：Android 动态打开 libvoice_input.so；
/// iOS 侧原生库静态链入主二进制，用 DynamicLibrary.process()。测试用假
/// 绑定实现 [VoiceInputBindings]/[VoiceOutputBindings] 即可，不需要任何
/// 原生环境。
library;

import 'dart:ffi';
import 'dart:io';
import 'dart:typed_data';

import 'package:ffi/ffi.dart';

import 'bindings_common.dart';

export 'bindings_common.dart'
    show
        VoiceInputBindings,
        VoiceInputException,
        VoiceOutputBindings,
        VoiceOutputException;

/// @brief 平台默认采集绑定工厂（与桩分支同名，供条件编译统一引用）。
VoiceInputBindings createDefaultInputBindings() => FfiVoiceInputBindings();

/// @brief 平台默认播放绑定工厂（与桩分支同名，供条件编译统一引用）。
VoiceOutputBindings createDefaultOutputBindings() => FfiVoiceOutputBindings();

/// @brief 按平台选择默认动态库句柄：Android 动态打开 libvoice_input.so；
/// iOS 原生库静态链入主二进制，用 DynamicLibrary.process()。
DynamicLibrary _defaultLibrary() {
  if (Platform.isAndroid) {
    return DynamicLibrary.open('libvoice_input.so');
  }
  return DynamicLibrary.process(); // iOS：静态链入主二进制。
}

/// @brief dart:ffi 真实采集绑定。
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
    final code = _viFormat(out, out + 1, out + 2);
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

/// @brief dart:ffi 真实播放绑定（与采集共用 libvoice_input.so）。
final class FfiVoiceOutputBindings implements VoiceOutputBindings {
  /// @brief 构造绑定；缺省按平台自动选择库句柄。
  ///
  /// @param library 已打开的动态库；测试可注入。
  FfiVoiceOutputBindings({DynamicLibrary? library})
      : _lib = library ?? _defaultLibrary() {
    _voStart =
        _lib.lookupFunction<VoStartNative, VoStartDart>('vo_start');
    _voStop = _lib.lookupFunction<VoStopNative, VoStopDart>('vo_stop');
    _voWrite = _lib.lookupFunction<VoWriteNative, VoWriteDart>('vo_write');
    _voBuffered =
        _lib.lookupFunction<VoBufferedNative, VoBufferedDart>('vo_buffered');
    _voUnderrun =
        _lib.lookupFunction<VoUnderrunNative, VoUnderrunDart>('vo_underrun');
    _voDropped =
        _lib.lookupFunction<VoDroppedNative, VoDroppedDart>('vo_dropped');
    _voFormat =
        _lib.lookupFunction<VoFormatNative, VoFormatDart>('vo_format');
  }

  final DynamicLibrary _lib;

  late final VoStartDart _voStart;
  late final VoStopDart _voStop;
  late final VoWriteDart _voWrite;
  late final VoBufferedDart _voBuffered;
  late final VoUnderrunDart _voUnderrun;
  late final VoDroppedDart _voDropped;
  late final VoFormatDart _voFormat;

  Pointer<VoConfig>? _config;
  Pointer<Uint8>? _writeBuffer;
  int _writeBufferBytes = 0;
  Pointer<Int32>? _formatOut;

  @override
  void start({
    required int sampleRateHz,
    required int capacityMs,
    required int primingMs,
  }) {
    final config = _config ??= malloc<VoConfig>();
    config.ref.sampleRate = sampleRateHz;
    config.ref.capacityMs = capacityMs;
    config.ref.primingMs = primingMs;
    final code = _voStart(config);
    if (code != 0) {
      throw VoiceOutputException(code, switch (code) {
        -1 => '播放已在运行',
        -2 => '输出流打开失败',
        -3 => '启动参数非法',
        _ => 'vo_start 未知错误码',
      });
    }
  }

  @override
  void stop() {
    if (_config != null) {
      _voStop();
    }
  }

  @override
  int write(Uint8List source) {
    if (source.isEmpty) {
      return 0;
    }
    if (_writeBuffer == null || _writeBufferBytes < source.length) {
      if (_writeBuffer != null) {
        malloc.free(_writeBuffer!);
      }
      _writeBuffer = malloc<Uint8>(source.length);
      _writeBufferBytes = source.length;
    }
    _writeBuffer!.asTypedList(source.length).setAll(0, source);
    return _voWrite(_writeBuffer!, source.length);
  }

  @override
  int bufferedBytes() => _voBuffered();

  @override
  int underrunBytes() => _voUnderrun();

  @override
  int droppedBytes() => _voDropped();

  @override
  ({int sampleRateHz, int channelCount, int bitsPerSample})?
      negotiatedFormat() {
    final out = _formatOut ??= malloc<Int32>(3);
    final code = _voFormat(out, out + 1, out + 2);
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

/// @brief vo_config_t 的 FFI 视图（与 voice_output.h 布局一致）。
final class VoConfig extends Struct {
  /// @brief 请求采样率，单位 Hz。
  @Int32()
  external int sampleRate;

  /// @brief 播放缓冲容量按毫秒预算。
  @Int32()
  external int capacityMs;

  /// @brief 预缓冲阈值，单位毫秒。
  @Int32()
  external int primingMs;
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

// vo_* C ABI 的原生/Dart 签名对。
typedef VoStartNative = Int32 Function(Pointer<VoConfig>);
typedef VoStartDart = int Function(Pointer<VoConfig>);
typedef VoStopNative = Int32 Function();
typedef VoStopDart = int Function();
typedef VoWriteNative = Int32 Function(Pointer<Uint8>, Int32);
typedef VoWriteDart = int Function(Pointer<Uint8>, int);
typedef VoBufferedNative = Int32 Function();
typedef VoBufferedDart = int Function();
typedef VoUnderrunNative = Uint64 Function();
typedef VoUnderrunDart = int Function();
typedef VoDroppedNative = Uint64 Function();
typedef VoDroppedDart = int Function();
typedef VoFormatNative = Int32 Function(
    Pointer<Int32>, Pointer<Int32>, Pointer<Int32>);
typedef VoFormatDart = int Function(
    Pointer<Int32>, Pointer<Int32>, Pointer<Int32>);
