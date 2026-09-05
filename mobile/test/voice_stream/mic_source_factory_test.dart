import 'package:flutter_test/flutter_test.dart';
import 'package:ai_speak/features/voice_stream/src/contracts.dart';
import 'package:ai_speak/features/voice_stream/src/mic_source_factory.dart';
import 'package:ai_speak/features/voice_stream/src/native_mic_source.dart';

/// @brief 平台工厂（VM 测试运行在原生分支）：返回 NativeMicSource。
void main() {
  test('原生平台上 createDefaultMicSource 返回 NativeMicSource', () {
    final source = createDefaultMicSource();
    expect(source, isA<NativeMicSource>());
  });

  test('工厂产物履行 MicSource 契约', () {
    final source = createDefaultMicSource();
    expect(source, isA<MicSource>());
  });
}
