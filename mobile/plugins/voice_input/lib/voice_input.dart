/// @file voice_input.dart
/// @brief voice_input 插件公共 API：采集与播放的高层控制面。
///
/// 低层 FFI 绑定见 `src/bindings.dart`；采集（VoiceInput）只暴露启动/停止/
/// 读出三件事，播放（VoiceOutput）只暴露启动/停止/写入三件事，不做数据
/// 搬运的语义决策（切帧与 gap 标志在 voice_stream 模块的 [FrameSlicer]
/// 完成，欠载/丢帧统计由原生 playback_queue 状态机累计）。
library;

import 'dart:async';
import 'dart:typed_data';

import 'src/bindings.dart';

export 'src/bindings.dart'
    show
        VoiceInputBindings,
        VoiceInputException,
        FfiVoiceInputBindings,
        VoiceOutputBindings,
        VoiceOutputException,
        FfiVoiceOutputBindings;

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

/// @brief 扬声器播放的高层封装。
///
/// 与 [VoiceInput] 对称：Dart 侧是播放队列的唯一生产者（[write] 全有或
/// 全无），音频回调线程是唯一消费者（取数补静音）；欠载策略（预缓冲/迟滞）
/// 在原生 playback_queue 状态机内完成，上层只读统计。
///
/// 典型用法（App 装配层，R8）：
/// ```dart
/// final speaker = VoiceOutput();
/// await speaker.start(sampleRateHz: 16000, capacityMs: 1000);
/// final accepted = speaker.write(frame.samples); // false = 缓冲满被拒
/// await speaker.stop();
/// ```
class VoiceOutput {
  /// @brief 构造播放器。
  ///
  /// @param bindings 原生绑定；缺省 FFI 真实实现，测试可注入假绑定。
  VoiceOutput({VoiceOutputBindings? bindings})
      : _bindings = bindings ?? FfiVoiceOutputBindings();

  final VoiceOutputBindings _bindings;
  bool _running = false;

  /// @brief 是否已启动。
  bool get isRunning => _running;

  /// @brief 打开输出流并开始播放。
  ///
  /// @param sampleRateHz 请求采样率；实际以协商后格式为准（vo_format 回读）。
  /// @param capacityMs 播放缓冲容量按毫秒预算。
  /// @param primingMs 预缓冲阈值；缺省 40ms，负值取原生默认。
  /// @throws VoiceOutputException 原生层打开失败或重复启动。
  Future<void> start({
    int sampleRateHz = 16000,
    int capacityMs = 1000,
    int primingMs = 40,
  }) async {
    if (_running) {
      throw StateError('VoiceOutput 已在运行');
    }
    _bindings.start(
      sampleRateHz: sampleRateHz,
      capacityMs: capacityMs,
      primingMs: primingMs,
    );
    _running = true;
  }

  /// @brief 拷贝写入一帧待播放 PCM（全有或全无）。
  ///
  /// @param pcm PCM 字节（16 位小端）。
  /// @return 是否被完整接受；缓冲满则整块拒绝返回 false，永不阻塞。
  bool write(Uint8List pcm) => _bindings.write(pcm) == pcm.length;

  /// @brief 当前缓冲中的可播放字节数。
  int get bufferedBytes => _bindings.bufferedBytes();

  /// @brief 欠载静音累计字节数（欠载策略调参观测口）。
  int get underrunBytes => _bindings.underrunBytes();

  /// @brief 空间不足被整块拒绝的累计字节数（播放侧丢帧统计）。
  int get droppedBytes => _bindings.droppedBytes();

  /// @brief 协商后的实际格式；未启动或未协商时为 null。
  ({int sampleRateHz, int channelCount, int bitsPerSample})?
      get negotiatedFormat => _bindings.negotiatedFormat();

  /// @brief 停止播放并释放设备；幂等。
  Future<void> stop() async {
    if (!_running) {
      return;
    }
    _running = false;
    _bindings.stop();
  }
}
