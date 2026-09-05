import 'package:flutter_test/flutter_test.dart';
import 'package:ai_speak/features/voice_stream/src/pcm_convert.dart';

/// @brief pcm16FromFloat32：限幅、量化、小端字节序。
void main() {
  group('pcm16FromFloat32', () {
    test('静音样本转全零字节', () {
      final out = pcm16FromFloat32(Float32List.fromList([0, 0, 0]));
      expect(out, hasLength(6));
      expect(out, everyElement(0));
    });

    test('正满幅 1.0 → 32767（小端 0x7FFF）', () {
      final out = pcm16FromFloat32(Float32List.fromList([1.0]));
      expect(out, hasLength(2));
      expect(out[0], 0xFF); // 低字节
      expect(out[1], 0x7F); // 高字节
    });

    test('负满幅 -1.0 → -32768（小端 0x8000）', () {
      final out = pcm16FromFloat32(Float32List.fromList([-1.0]));
      expect(out[0], 0x00); // 低字节
      expect(out[1], 0x80); // 高字节
    });

    test('超界样本限幅：1.5 → 32767，-1.5 → -32768', () {
      final out = pcm16FromFloat32(Float32List.fromList([1.5, -1.5]));
      expect(out[0], 0xFF);
      expect(out[1], 0x7F);
      expect(out[2], 0x00);
      expect(out[3], 0x80);
    });

    test('中间值线性量化并四舍五入', () {
      // 0.5 * 32767 = 16383.5 → 16384（round）。
      final out = pcm16FromFloat32(Float32List.fromList([0.5]));
      final value = out[0] | (out[1] << 8);
      expect(value, 16384);
    });

    test('样本数与字节量 1:2 对应', () {
      final samples = Float32List(160); // 16kHz 下 10ms。
      final out = pcm16FromFloat32(samples);
      expect(out, hasLength(320));
    });
  });
}
