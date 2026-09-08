import 'package:flutter_test/flutter_test.dart';

import 'package:ai_speak/features/voice_stream/src/contracts.dart';
import 'package:ai_speak/features/voice_stream/src/mic_source_factory.dart';

/// @brief 平台工厂编译冒烟测试（web 分支链路验证）。
///
/// 本测试在任意平台编译运行：VM 上工厂走原生分支（Oboe/RemoteIO FFI），
/// 用 `flutter test --platform chrome` 运行时工厂走 web 分支——此时
/// WebMicSource（getUserMedia + AudioWorklet）+ package:web 采集链路被
/// 强制参与编译，可验证条件导出与 web 实现的编译正确性。两类平台下
/// 断言相同：工厂产物履行 [MicSource] 契约。
void main() {
  test('当前平台工厂产物履行 MicSource 契约', () {
    final source = createDefaultMicSource();
    expect(source, isA<MicSource>());
  });
}
