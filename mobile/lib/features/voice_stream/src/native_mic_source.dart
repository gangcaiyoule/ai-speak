/// @file native_mic_source.dart
/// @brief MicSource 的平台实现：FFI 采集 → 环缓出口 → 切帧 → 帧流（R4/R5）。
///
/// 数据面：原生回调线程写环缓（丢旧永不阻塞）；本类在 Dart 侧周期性
/// drain（拷贝读），交给 [FrameSlicer] 切帧后发入流。环缓 dropped 计数
/// 增长即上游丢弃，转换为下一帧的 gapBefore 标志（README 第 3 节语义）。
///
/// 时间戳语义：v1 采用切帧器墙钟基准（drain 时刻起算），误差上限为
/// drain 周期；回环时间戳法（R4 真机延迟测量）在 R8 联调时接回环协议，
/// 不在本层伪造精度。
library;

import 'dart:async';
import 'dart:typed_data';

import 'package:voice_input/voice_input.dart';

import 'contracts.dart';
import 'frame_slicer.dart';

/// @brief 基于 FFI 采集绑定的 [MicSource] 实现（Android Oboe / iOS RemoteIO）。
class NativeMicSource implements MicSource {
  /// @brief 构造采集源。
  ///
  /// @param bindings 原生绑定；缺省经绑定层平台工厂创建（原生 FFI 真实
  ///        实现 / web 桩），测试注入假绑定。
  /// @param drainInterval 出口轮询周期；10ms ≈ 半个 20ms 帧，延迟与
  ///        唤醒次数的折中。
  /// @param capacityMs 环缓容量按毫秒预算（覆盖丢旧的缓冲上限）。
  NativeMicSource({
    VoiceInputBindings? bindings,
    this.drainInterval = const Duration(milliseconds: 10),
    this.capacityMs = 2000,
  }) : _bindings = bindings ?? createDefaultInputBindings();

  final VoiceInputBindings _bindings;
  final Duration drainInterval;
  final int capacityMs;

  StreamController<AudioFrame>? _controller;
  Timer? _timer;
  FrameSlicer? _slicer;
  Uint8List? _readBuffer;
  int _lastDropped = 0;

  @override
  Stream<AudioFrame> start(AudioFormat format, int frameDurationMs) {
    if (_controller != null) {
      throw StateError('NativeMicSource 已在采集；重复 start 视为错误');
    }
    final controller = StreamController<AudioFrame>(onCancel: stop);
    _controller = controller;
    try {
      _bindings.start(
        sampleRateHz: format.sampleRateHz,
        capacityMs: capacityMs,
      );
    } catch (_) {
      _controller = null;
      rethrow;
    }
    _slicer = FrameSlicer(format: format, frameDurationMs: frameDurationMs);
    // 单次读出按 100ms 音频预算；实际格式若高于请求值也只是读得粗一点。
    final chunkBytes = (format.sampleRateHz / 1000 * 2 * 100).round();
    _readBuffer = Uint8List(chunkBytes < 256 ? 256 : chunkBytes);
    _lastDropped = _bindings.droppedBytes();
    _timer = Timer.periodic(drainInterval, (_) => _drain());
    return controller.stream;
  }

  @override
  Future<void> stop() async {
    _timer?.cancel();
    _timer = null;
    _slicer = null;
    final controller = _controller;
    _controller = null;
    _bindings.stop(); // 幂等。
    // 关闭帧流：有监听者则收到 done；无监听时 close 的 done 无从派发，
    // 不能阻塞 stop（否则从未订阅就 stop 的调用会永远挂起）。
    unawaited(controller?.close());
  }

  /// @brief 内部：周期出口——gap 检查 → 拷贝读 → 切帧 → 发流。
  void _drain() {
    final controller = _controller;
    final slicer = _slicer;
    final buffer = _readBuffer;
    if (controller == null || slicer == null || buffer == null) {
      return;
    }
    if (controller.isClosed) {
      return;
    }
    final dropped = _bindings.droppedBytes();
    if (dropped > _lastDropped) {
      _lastDropped = dropped;
      slicer.markGap();
    }
    final n = _bindings.readInto(buffer);
    if (n <= 0) {
      return;
    }
    final chunk =
        n == buffer.length ? buffer : Uint8List.sublistView(buffer, 0, n);
    for (final frame in slicer.push(chunk)) {
      controller.add(frame);
    }
  }
}
