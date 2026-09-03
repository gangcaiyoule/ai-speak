import 'package:flutter_test/flutter_test.dart';
import 'package:ai_speak/features/voice_stream/src/contracts.dart';
import 'package:ai_speak/features/voice_stream/src/session.dart';

void main() {
  group('VoiceSessionConfig', () {
    test('合法幂等键通过校验', () {
      final config = VoiceSessionConfig(
        idempotencyKey: 'session-20260902-a1',
        format: const AudioFormat(sampleRateHz: 16000, channelCount: 1, bitsPerSample: 16),
      );
      expect(config.idempotencyKey, 'session-20260902-a1');
    });

    test('幂等键过短被拒绝', () {
      expect(
        () => VoiceSessionConfig(
          idempotencyKey: 'short',
          format: const AudioFormat(sampleRateHz: 16000, channelCount: 1, bitsPerSample: 16),
        ),
        throwsArgumentError,
      );
    });

    test('幂等键过长被拒绝', () {
      expect(
        () => VoiceSessionConfig(
          idempotencyKey: 'k' * 129,
          format: const AudioFormat(sampleRateHz: 16000, channelCount: 1, bitsPerSample: 16),
        ),
        throwsArgumentError,
      );
    });

    test('幂等键首尾空白被拒绝', () {
      expect(
        () => VoiceSessionConfig(
          idempotencyKey: ' session-key-01 ',
          format: const AudioFormat(sampleRateHz: 16000, channelCount: 1, bitsPerSample: 16),
        ),
        throwsArgumentError,
      );
    });
  });

  group('VoiceSessionLifecycle', () {
    test('正常路径 idle → active → finishing → closed', () {
      final lifecycle = VoiceSessionLifecycle();
      expect(lifecycle.phase, VoiceSessionPhase.idle);

      lifecycle.begin();
      expect(lifecycle.phase, VoiceSessionPhase.active);
      expect(lifecycle.canSendFrame, isTrue);

      lifecycle.requestFinish();
      expect(lifecycle.phase, VoiceSessionPhase.finishing);
      expect(lifecycle.canSendFrame, isFalse);

      lifecycle.complete();
      expect(lifecycle.isClosed, isTrue);
      expect(lifecycle.canSendFrame, isFalse);
    });

    test('finishing 阶段不允许发送音频帧', () {
      final lifecycle = VoiceSessionLifecycle();
      lifecycle.begin();
      lifecycle.requestFinish();
      expect(lifecycle.canSendFrame, isFalse);
    });

    test('active 阶段可直接取消进入终态', () {
      final lifecycle = VoiceSessionLifecycle();
      lifecycle.begin();
      lifecycle.abort();
      expect(lifecycle.isClosed, isTrue);
    });

    test('finishing 阶段可失败进入终态', () {
      final lifecycle = VoiceSessionLifecycle();
      lifecycle.begin();
      lifecycle.requestFinish();
      lifecycle.abort();
      expect(lifecycle.isClosed, isTrue);
    });

    test('对已关闭会话 abort 幂等', () {
      final lifecycle = VoiceSessionLifecycle();
      lifecycle.begin();
      lifecycle.abort();
      expect(() => lifecycle.abort(), returnsNormally);
    });

    test('begin 重复调用抛 StateError', () {
      final lifecycle = VoiceSessionLifecycle();
      lifecycle.begin();
      expect(() => lifecycle.begin(), throwsStateError);
    });

    test('idle 阶段直接 requestFinish 抛 StateError', () {
      final lifecycle = VoiceSessionLifecycle();
      expect(() => lifecycle.requestFinish(), throwsStateError);
    });

    test('active 阶段未 finish 不允许 complete 抛 StateError', () {
      final lifecycle = VoiceSessionLifecycle();
      lifecycle.begin();
      expect(() => lifecycle.complete(), throwsStateError);
    });
  });
}
