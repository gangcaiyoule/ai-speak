import 'dart:async';
import 'dart:typed_data';

import 'package:flutter_test/flutter_test.dart';
import 'package:ai_speak/features/voice_stream/src/contracts.dart';
import 'package:ai_speak/features/voice_stream/src/session.dart';
import 'package:ai_speak/features/voice_stream/src/session_transport.dart';

/// @brief 可编程假传输：记录发送、手动投放事件/错误/关闭。
class _FakeTransport implements TextCapableTransport {
  final frames = <AudioFrame>[];
  final texts = <String>[];
  final controller = StreamController<TransportEvent>.broadcast();
  bool closed = false;

  @override
  void sendFrame(AudioFrame frame) => frames.add(frame);

  @override
  void sendText(String payload) => texts.add(payload);

  @override
  Stream<TransportEvent> get events => controller.stream;

  @override
  Future<void> close() async {
    closed = true;
  }

  void emit(TransportEvent event) => controller.add(event);

  void fail(Object error) => controller.addError(error);
}

VoiceSessionConfig config() => VoiceSessionConfig(
      idempotencyKey: 'test-key-0001',
      format: const AudioFormat(
        sampleRateHz: 16000,
        channelCount: 1,
        bitsPerSample: 16,
      ),
    );

AudioFrame frame(int seq) => AudioFrame(
      seq: seq,
      timestampMs: 1000,
      samples: Uint8List(64),
    );

void main() {
  group('TransportVoiceSession', () {
    test('start 后进入 active 并发出 SessionStarted，帧正常上行', () async {
      final transport = _FakeTransport();
      final session = TransportVoiceSession(transport: transport, config: config());

      final events = <VoiceSessionEvent>[];
      final sub = session.events.listen(events.add);

      await session.start();
      expect(session.phase, VoiceSessionPhase.active);
      expect(events.whereType<SessionStarted>(), hasLength(1));
      expect(transport.texts.first, contains('"type":"start"'));

      session.sendFrame(frame(0));
      expect(transport.frames, hasLength(1));

      await session.cancel();
      await sub.cancel();
    });

    test('active 阶段文本回显映射 SessionPartial，统计映射 SessionStats', () async {
      final transport = _FakeTransport();
      final session = TransportVoiceSession(transport: transport, config: config());

      final events = <VoiceSessionEvent>[];
      final sub = session.events.listen(events.add);
      await session.start();

      transport.emit(const TransportMessage('{"partial":true}'));
      transport.emit(
        const TransportStats(sentFrames: 7, droppedFrames: 1, bufferedBytes: 640),
      );
      await Future<void>.delayed(Duration.zero);

      final partial = events.whereType<SessionPartial>().single;
      expect(partial.payload, '{"partial":true}');
      final stats = events.whereType<SessionStats>().last;
      expect(stats.sentFrames, 7);
      expect(stats.droppedFrames, 1);

      await session.cancel();
      await sub.cancel();
    });

    test('回声音频帧不进会话层事件流', () async {
      final transport = _FakeTransport();
      final session = TransportVoiceSession(transport: transport, config: config());

      final events = <VoiceSessionEvent>[];
      final sub = session.events.listen(events.add);
      await session.start();

      transport.emit(TransportAudioFrame(frame(0)));
      await Future<void>.delayed(Duration.zero);
      expect(events.whereType<SessionPartial>(), isEmpty);
      expect(events.whereType<SessionStats>(), isEmpty);

      await session.cancel();
      await sub.cancel();
    });

    test('传输层错误映射为可重试 SessionFailed 并进入 closed', () async {
      final transport = _FakeTransport();
      final session = TransportVoiceSession(transport: transport, config: config());

      final events = <VoiceSessionEvent>[];
      final sub = session.events.listen(events.add);
      await session.start();

      transport.fail(StateError('断连'));
      await Future<void>.delayed(Duration.zero);

      final failed = events.whereType<SessionFailed>().single;
      expect(failed.kind, 'transport');
      expect(failed.retryable, isTrue);
      expect(session.phase, VoiceSessionPhase.closed);
      expect(transport.closed, isTrue);

      expect(() => session.sendFrame(frame(0)), throwsStateError);
      await sub.cancel();
    });

    test('finish 发送控制帧，回显即终态 SessionFinal，随后关闭传输', () async {
      final transport = _FakeTransport();
      final session = TransportVoiceSession(transport: transport, config: config());

      final events = <VoiceSessionEvent>[];
      final sub = session.events.listen(events.add);
      await session.start();

      final finishing = session.finish();
      expect(session.phase, VoiceSessionPhase.finishing);
      expect(
        () => session.sendFrame(frame(0)),
        throwsStateError,
      ); // finishing 阶段禁发帧。

      transport.emit(const TransportMessage('{"type":"finish"}'));
      await finishing.timeout(const Duration(seconds: 2));

      expect(events.whereType<SessionFinal>().single.payload, '{"type":"finish"}');
      expect(session.phase, VoiceSessionPhase.closed);
      expect(transport.texts.last, contains('"type":"finish"'));
      expect(transport.closed, isTrue);
      await sub.cancel();
    });

    test('finish 超时未收到终态回执按可重试失败处理', () async {
      final transport = _FakeTransport();
      final session = TransportVoiceSession(
        transport: transport,
        config: config(),
        finishTimeout: const Duration(milliseconds: 50),
      );

      final events = <VoiceSessionEvent>[];
      final sub = session.events.listen(events.add);
      await session.start();

      final finishing = session.finish();
      await finishing.timeout(const Duration(seconds: 2));

      final failed = events.whereType<SessionFailed>().single;
      expect(failed.kind, 'finish-timeout');
      expect(failed.retryable, isTrue);
      expect(session.phase, VoiceSessionPhase.closed);
      await sub.cancel();
    });

    test('cancel 立即中断、发送 cancel 控制帧且幂等', () async {
      final transport = _FakeTransport();
      final session = TransportVoiceSession(transport: transport, config: config());

      final events = <VoiceSessionEvent>[];
      final sub = session.events.listen(events.add);
      await session.start();

      await session.cancel();
      expect(session.phase, VoiceSessionPhase.closed);
      expect(transport.texts.last, contains('"type":"cancel"'));
      expect(transport.closed, isTrue);

      await session.cancel(); // 幂等，不抛异常。
      await sub.cancel();
    });

    test('active 阶段直接取消：不发 finish、事件流关闭', () async {
      final transport = _FakeTransport();
      final session = TransportVoiceSession(transport: transport, config: config());

      final closed = Completer<void>();
      final sub = session.events.listen(null, onDone: closed.complete);
      await session.start();
      await session.cancel();

      await closed.future.timeout(const Duration(seconds: 2));
      expect(transport.texts, hasLength(1)); // 只有 start，无 finish。
      expect(transport.texts.single, contains('"type":"start"'));
      await sub.cancel();
    });
  });
}
