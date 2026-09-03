import 'dart:async';
import 'dart:typed_data';

import 'package:flutter_test/flutter_test.dart';
import 'package:ai_speak/features/voice_stream/src/contracts.dart';
import 'package:ai_speak/features/voice_stream/src/native_mic_source.dart';
import 'package:voice_input/voice_input.dart';

const _format = AudioFormat(
  sampleRateHz: 16000,
  channelCount: 1,
  bitsPerSample: 16,
); // 20ms = 640B。

/// @brief 可编程假绑定：按队列供数、可注入 dropped 增量。
class _ScriptedBindings implements VoiceInputBindings {
  final chunks = <Uint8List>[];
  int dropped = 0;
  int startCalls = 0;
  int stopCalls = 0;
  int? startedRate;
  int? startedCapacityMs;

  @override
  void start({required int sampleRateHz, required int capacityMs}) {
    startCalls++;
    startedRate = sampleRateHz;
    startedCapacityMs = capacityMs;
  }

  @override
  void stop() => stopCalls++;

  @override
  int readInto(Uint8List destination) {
    if (chunks.isEmpty) {
      return 0;
    }
    final chunk = chunks.removeAt(0);
    final n = chunk.length < destination.length
        ? chunk.length
        : destination.length;
    destination.setAll(0, chunk);
    return n;
  }

  @override
  int droppedBytes() => dropped;

  @override
  ({int sampleRateHz, int channelCount, int bitsPerSample})?
      negotiatedFormat() => (sampleRateHz: 16000, channelCount: 1, bitsPerSample: 16);

  void enqueue(int bytes, int fill) => chunks.add(
        Uint8List.fromList(List.filled(bytes, fill)),
      );
}

void main() {
  group('NativeMicSource', () {
    test('start 参数透传，出口按帧切齐发出', () async {
      final bindings = _ScriptedBindings();
      final source = NativeMicSource(
        bindings: bindings,
        drainInterval: const Duration(milliseconds: 1),
      );

      bindings.enqueue(640, 1);
      bindings.enqueue(640, 2);

      final frames =
          await source.start(_format, 20).take(2).timeout(
                const Duration(seconds: 5),
              ).toList();

      expect(bindings.startCalls, 1);
      expect(bindings.startedRate, 16000);
      expect(bindings.startedCapacityMs, 2000);
      expect(frames, hasLength(2));
      expect(frames[0].seq, 0);
      expect(frames[0].samples, hasLength(640));
      expect(frames[1].seq, 1);
      expect(frames[0].flags, AudioFrameFlags.none);

      await source.stop();
    });

    test('环缓丢旧计数增长转换为下一帧 gapBefore 标志', () async {
      final bindings = _ScriptedBindings();
      final source = NativeMicSource(
        bindings: bindings,
        drainInterval: const Duration(milliseconds: 1),
      );

      bindings.enqueue(640, 1);
      bindings.dropped = 640; // 模拟环缓丢了一整帧。
      bindings.enqueue(640, 2);

      final frames =
          await source.start(_format, 20).take(2).timeout(
                const Duration(seconds: 5),
              ).toList();

      expect(frames[0].flags, AudioFrameFlags.gapBefore);
      expect(frames[1].flags, AudioFrameFlags.none); // 只标一帧。

      await source.stop();
    });

    test('采集运行中重复 start 抛 StateError', () async {
      final bindings = _ScriptedBindings();
      final source = NativeMicSource(
        bindings: bindings,
        drainInterval: const Duration(milliseconds: 1),
      );

      final stream = source.start(_format, 20);
      expect(
        () => source.start(_format, 20),
        throwsStateError,
      );

      await source.stop();
      expect(stream, isNotNull); // 未订阅即关闭是合法路径。
    });

    test('stop 幂等且释放绑定', () async {
      final bindings = _ScriptedBindings();
      final source = NativeMicSource(
        bindings: bindings,
        drainInterval: const Duration(milliseconds: 1),
      );

      source.start(_format, 20);
      await source.stop();
      await source.stop();

      expect(
        bindings.stopCalls,
        greaterThanOrEqualTo(1),
      ); // 显式 stop + 流取消均触发幂等释放。
    });

    test('start 后立即 stop：帧流关闭', () async {
      final bindings = _ScriptedBindings();
      final source = NativeMicSource(
        bindings: bindings,
        drainInterval: const Duration(milliseconds: 1),
      );

      final stream = source.start(_format, 20);
      final done = Completer<void>();
      stream.listen(null, onDone: done.complete, cancelOnError: true);
      await source.stop();
      await done.future.timeout(const Duration(seconds: 2));
      expect(bindings.stopCalls, greaterThanOrEqualTo(1));
    });
  });
}
