import 'dart:typed_data';

import 'package:flutter_test/flutter_test.dart';
import 'package:ai_speak/features/voice_stream/src/ring_buffer.dart';

void main() {
  group('SpscRingBuffer', () {
    test('读写往返保持字节序', () {
      final ring = SpscRingBuffer(8);
      final out = Uint8List(8);

      expect(ring.write(Uint8List.fromList([1, 2, 3, 4, 5])), 5);
      expect(ring.readInto(out), 5);
      expect(out.sublist(0, 5), [1, 2, 3, 4, 5]);
      expect(ring.isEmpty, isTrue);
      expect(ring.droppedBytes, 0);
    });

    test('跨回绕点后数据仍连续，peek 返回两段视图', () {
      final ring = SpscRingBuffer(4);
      ring.write(Uint8List.fromList([1, 2, 3]));
      ring.advance(2); // 只剩 idx2 的字节 3。

      ring.write(Uint8List.fromList([4, 5, 6])); // 3 写满回绕。

      final views = ring.peek();
      expect(views, hasLength(2));
      expect(views[0], [3, 4]);
      expect(views[1], [5, 6]);

      ring.advance(4);
      expect(ring.isEmpty, isTrue);
    });

    test('写满后覆盖最旧数据并累计 droppedBytes', () {
      final ring = SpscRingBuffer(4);
      ring.write(Uint8List.fromList([1, 2, 3, 4]));
      ring.write(Uint8List.fromList([5, 6]));

      expect(ring.droppedBytes, 2);
      final out = Uint8List(4);
      expect(ring.readInto(out), 4);
      expect(out, [3, 4, 5, 6]);
    });

    test('单次写入超过容量时只保留末尾', () {
      final ring = SpscRingBuffer(4);
      ring.write(Uint8List.fromList([1, 2, 3, 4, 5, 6, 7]));

      expect(ring.droppedBytes, 3);
      final out = Uint8List(8);
      expect(ring.readInto(out), 4);
      expect(out.sublist(0, 4), [4, 5, 6, 7]);
    });

    test('peek 限定字节数，advance 分批消费', () {
      final ring = SpscRingBuffer(8);
      ring.write(Uint8List.fromList([1, 2, 3, 4, 5, 6]));

      final head = ring.peek(2);
      expect(head.single, [1, 2]);
      ring.advance(2);

      final out = Uint8List(8);
      expect(ring.readInto(out), 4);
      expect(out.sublist(0, 4), [3, 4, 5, 6]);
    });

    test('readInto 目标缓冲小于可读量时截断', () {
      final ring = SpscRingBuffer(8);
      ring.write(Uint8List.fromList([9, 8, 7, 6]));

      final out = Uint8List(2);
      expect(ring.readInto(out), 2);
      expect(out, [9, 8]);
      expect(ring.lengthBytes, 2);
    });

    test('空缓冲 peek 返回空视图', () {
      final ring = SpscRingBuffer(8);
      expect(ring.peek(), isEmpty);
    });

    test('advance 越界抛出 ArgumentError', () {
      final ring = SpscRingBuffer(8);
      ring.write(Uint8List.fromList([1, 2]));

      expect(() => ring.advance(3), throwsArgumentError);
      expect(() => ring.advance(-1), throwsArgumentError);
    });

    test('容量必须为正数', () {
      expect(() => SpscRingBuffer(0), throwsArgumentError);
      expect(() => SpscRingBuffer(-1), throwsArgumentError);
    });
  });
}
