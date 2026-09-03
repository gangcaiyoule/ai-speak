/// @file voice_input.dart
/// @brief voice_input 插件公共 API：麦克风采集的高层控制面。
///
/// 低层 FFI 绑定见 `src/bindings.dart`；本库只暴露启动/停止/读出三件事，
/// 不做数据搬运的语义决策（切帧与 gap 标志在 voice_stream 模块的
/// [FrameSlicer] 完成）。
library;

import 'dart:async';
import 'dart:typed_data';

import 'src/bindings.dart';

export 'src/bindings.dart'
    show VoiceInputBindings, VoiceInputException, FfiVoiceInputBindings;

/// @brief 麦克风采集的高层封装。
///
/// 典型用法（App 装配层，R8）：
/// ```dart
/// final mic = VoiceInput();
/// await mic.start(sampleRateHz: 16000, capacityMs: 2000);
/// final chunk = Uint8List(4096);
/// final n = mic.readInto(chunk); // 交给 FrameSlicer.push
/// ```
class VoiceInput {
  /// @brief 构造采集器。
  ///
  /// @param bindings 原生绑定；缺省 FFI 真实实现，测试可注入假绑定。
  VoiceInput({VoiceInputBindings? bindings})
      : _bindings = bindings ?? FfiVoiceInputBindings();

  final VoiceInputBindings _bindings;
  bool _running = false;

  /// @brief 是否已启动。
  bool get isRunning => _running;

  /// @brief 打开输入流并开始采集。
  ///
  /// @param sampleRateHz 请求采样率；实现按协商后实际格式交付。
  /// @param capacityMs 环缓容量按毫秒预算，覆盖丢旧。
  /// @throws VoiceInputException 原生层打开失败或重复启动。
  Future<void> start({
    int sampleRateHz = 16000,
    int capacityMs = 2000,
  }) async {
    if (_running) {
      throw StateError('VoiceInput 已在运行');
    }
    _bindings.start(sampleRateHz: sampleRateHz, capacityMs: capacityMs);
    _running = true;
  }

  /// @brief 拷贝读出一段已采集 PCM；无数据返回空视图语义（0 字节拷贝）。
  ///
  /// @param destination 目标缓冲。
  /// @return 实际读出字节数。
  int readInto(Uint8List destination) =>
      _bindings.readInto(destination);

  /// @brief 累计丢旧字节数（环缓覆盖丢弃）。
  int get droppedBytes => _bindings.droppedBytes();

  /// @brief 协商后的实际格式；未启动或未协商时为 null。
  ({int sampleRateHz, int channelCount, int bitsPerSample})?
      get negotiatedFormat => _bindings.negotiatedFormat();

  /// @brief 停止采集并释放设备；幂等。
  Future<void> stop() async {
    if (!_running) {
      return;
    }
    _running = false;
    _bindings.stop();
  }
}
