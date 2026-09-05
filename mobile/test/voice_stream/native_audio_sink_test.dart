import 'dart:typed_data';

import 'package:flutter_test/flutter_test.dart';
import 'package:ai_speak/features/voice_stream/src/contracts.dart';
import 'package:ai_speak/features/voice_stream/src/native_audio_sink.dart';
import 'package:voice_input/voice_input.dart';

/// @brief 可编程假播放绑定：记录调用、可注入失败与拒绝。
class _ScriptedOutputBindings implements VoiceOutputBindings {
  int startCalls = 0;
  int stopCalls = 0;
  int? startedRate;
  int? startedCapacityMs;
  int? startedPrimingMs;

  int underrun = 0;
  int buffered = 0;
  int dropped = 0;
  bool failNextStart = false;
  bool rejectWrites = false;
  final writtenChunks = <Uint8List>[];

  @override
  void start({
    required int sampleRateHz,
    required int capacityMs,
    required int primingMs,
  }) {
    startCalls++;
    startedRate = sampleRateHz;
    startedCapacityMs = capacityMs;
    startedPrimingMs = primingMs;
    if (failNextStart) {
      failNextStart = false;
      throw const VoiceOutputException(-2, '输出流打开失败');
    }
  }

  @override
  void stop() {
    stopCalls++;
  }

  @override
  int write(Uint8List source) {
    if (source.isEmpty) {
      return 0;
    }
    if (rejectWrites) {
      dropped += source.length;
      return 0; // 全有或全无：整块拒绝
    }
    writtenChunks.add(Uint8List.fromList(source));
    buffered += source.length;
    return source.length;
  }

  @override
  int bufferedBytes() => buffered;

  @override
  int underrunBytes() => underrun;

  @override
  int droppedBytes() => dropped;

  @override
  ({int sampleRateHz, int channelCount, int bitsPerSample})?
      negotiatedFormat() =>
          (sampleRateHz: 16000, channelCount: 1, bitsPerSample: 16);
}

AudioFrame _createFrame({int seq = 0, int length = 640}) {
  return AudioFrame(
    seq: seq,
    timestampMs: seq * 20,
    samples: Uint8List(length),
  );
}

void main() {
  group('NativeAudioSink', () {
    test('首次 write 触发惰性启动并全额写入', () async {
      final bindings = _ScriptedOutputBindings();
      final sink = NativeAudioSink(
        bindings: bindings,
        capacityMs: 1000,
        primingMs: 40,
      );

      final frame = _createFrame(seq: 0, length: 640);
      final accepted = sink.write(frame);

      expect(accepted, isTrue);
      expect(bindings.startCalls, 1);
      expect(bindings.startedRate, 16000);
      expect(bindings.startedCapacityMs, 1000);
      expect(bindings.startedPrimingMs, 40);
      expect(bindings.writtenChunks.length, 1);
      expect(sink.bufferedBytes, 640);

      await sink.stop();
    });

    test('空帧直接返回 true 且不触发惰性启动', () async {
      final bindings = _ScriptedOutputBindings();
      final sink = NativeAudioSink(bindings: bindings);

      final emptyFrame = _createFrame(seq: 0, length: 0);
      final accepted = sink.write(emptyFrame);

      expect(accepted, isTrue);
      expect(bindings.startCalls, 0);
      expect(bindings.writtenChunks, isEmpty);

      await sink.stop();
    });

    test('空间不足时整块拒绝返回 false', () async {
      final bindings = _ScriptedOutputBindings();
      final sink = NativeAudioSink(bindings: bindings);

      bindings.rejectWrites = true;
      final frame = _createFrame(seq: 0, length: 640);
      final accepted = sink.write(frame);

      expect(accepted, isFalse);
      expect(bindings.startCalls, 1);
      expect(sink.droppedBytes, 640);

      await sink.stop();
    });

    test('统计量透传（underrunBytes, bufferedBytes, droppedBytes, negotiatedFormat）',
        () async {
      final bindings = _ScriptedOutputBindings();
      final sink = NativeAudioSink(bindings: bindings);

      bindings.underrun = 320;
      bindings.buffered = 640;
      bindings.dropped = 1280;

      expect(sink.underrunBytes, 320);
      expect(sink.bufferedBytes, 640);
      expect(sink.droppedBytes, 1280);
      expect(
        sink.negotiatedFormat,
        (sampleRateHz: 16000, channelCount: 1, bitsPerSample: 16),
      );

      await sink.stop();
    });

    test('stop 幂等，且 stop 后再次写入重新惰性启动（重连语义）', () async {
      final bindings = _ScriptedOutputBindings();
      final sink = NativeAudioSink(bindings: bindings);

      sink.write(_createFrame(seq: 0));
      expect(bindings.startCalls, 1);

      await sink.stop();
      expect(bindings.stopCalls, 1);

      await sink.stop();
      expect(bindings.stopCalls, 2);

      // 再次写入：重新触发 start（重连）
      sink.write(_createFrame(seq: 1));
      expect(bindings.startCalls, 2);

      await sink.stop();
      expect(bindings.stopCalls, 3);
    });

    test('start 抛异常时不标记打开，下次写入重试 start', () async {
      final bindings = _ScriptedOutputBindings();
      final sink = NativeAudioSink(bindings: bindings);

      bindings.failNextStart = true;
      expect(
        () => sink.write(_createFrame(seq: 0)),
        throwsA(isA<VoiceOutputException>()),
      );
      expect(bindings.startCalls, 1);

      // 下次写入重新尝试 start
      final accepted = sink.write(_createFrame(seq: 0));
      expect(accepted, isTrue);
      expect(bindings.startCalls, 2);

      await sink.stop();
    });
  });
}
