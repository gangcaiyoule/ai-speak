import 'dart:typed_data';

import 'package:flutter_test/flutter_test.dart';
import 'package:ai_speak/features/voice_stream/src/contracts.dart';
import 'package:ai_speak/features/voice_stream/src/frame_slicer.dart';
import 'package:ai_speak/features/voice_stream/src/ring_buffer.dart';

AudioFormat get _format => const AudioFormat(
      sampleRateHz: 16000,
      channelCount: 1,
      bitsPerSample: 16,
    ); // 20ms = 640B。

void main() {
  group('FrameSlicer', () {
    test('AudioFrame.flags 缺省为 0，既有构造不受影响', () {
      final frame = AudioFrame(seq: 1, timestampMs: 2, samples: Uint8List(4));
      expect(frame.flags, AudioFrameFlags.none);
    });

    test('非整块输入切成定长帧，seq 连续、timestamp 按帧推进', () {
      var now = 1000;
      final slicer = FrameSlicer(
        format: _format,
        clock: () => now,
      ); // frameBytes = 640。

      final first = slicer.push(Uint8List(1000));
      expect(first, hasLength(1));
      expect(first[0].seq, 0);
      expect(first[0].timestampMs, 1000);
      expect(first[0].samples.length, 640);
      expect(first[0].flags, AudioFrameFlags.none);
      expect(slicer.pendingBytes, 360);

      final second = slicer.push(Uint8List(300));
      expect(second, hasLength(1));
      expect(second[0].seq, 1);
      expect(second[0].timestampMs, 1020);
      expect(slicer.pendingBytes, 20);
      expect(slicer.nextSeq, 2);
    });

    test('空块输入与空残余 flush 都返回空', () {
      final slicer = FrameSlicer(format: _format);
      expect(slicer.push(Uint8List(0)), isEmpty);
      expect(slicer.flush(), isEmpty);
      expect(slicer.hasPending, isFalse);
    });

    test('flush 收尾输出短帧，之后新帧流重新取基准时刻', () {
      var now = 500;
      final slicer = FrameSlicer(format: _format, clock: () => now);

      slicer.push(Uint8List(700)); // 产出 1 整帧，剩 60B。
      final tail = slicer.flush();
      expect(tail, hasLength(1));
      expect(tail[0].seq, 1);
      expect(tail[0].timestampMs, 520); // 500 + 1 × 20。
      expect(tail[0].samples.length, 60);
      expect(slicer.hasPending, isFalse);

      // 新帧流：重新取时钟基准。
      now = 9000;
      final next = slicer.push(Uint8List(640));
      expect(next, hasLength(1));
      expect(next[0].seq, 2); // seq 跨帧流连续。
      expect(next[0].timestampMs, 9000);
    });

    test('markGap 只落在下一产出的帧上', () {
      final slicer = FrameSlicer(format: _format);

      slicer.markGap();
      final frames = slicer.push(Uint8List(640 * 2));
      expect(frames, hasLength(2));
      expect(frames[0].flags, AudioFrameFlags.gapBefore);
      expect(frames[1].flags, AudioFrameFlags.none); // 只标记一帧。
    });

    test('drain 对接环形缓冲：跨回绕两段视图拼接正确且 advance 成对', () {
      final slicer = FrameSlicer(format: _format);
      final ring = SpscRingBuffer(2048); // 容量须容纳 200 + 1720 = 1920B。

      ring.write(Uint8List.fromList(List.generate(1000, (i) => i % 256)));
      ring.advance(800); // 制造回绕：剩 200B。
      ring.write(Uint8List.fromList(List.generate(1720, (i) => (i + 7) % 256)));
      // 环缓共 1920B = 640 × 3。

      final frames = slicer.drain(ring);
      expect(frames, hasLength(3));
      for (var i = 0; i < 3; i++) {
        expect(frames[i].seq, i);
        expect(frames[i].samples.length, 640);
      }
      expect(ring.isEmpty, isTrue);
      expect(ring.droppedBytes, 0);

      // 校验内容与写入顺序一致。
      final expected = <int>[
        ...List.generate(1000, (i) => i % 256).skip(800),
        ...List.generate(1720, (i) => (i + 7) % 256),
      ];
      for (var i = 0; i < 3; i++) {
        expect(frames[i].samples, expected.sublist(i * 640, (i + 1) * 640));
      }
    });

    test('drain 空缓冲返回空且不推进环缓', () {
      final slicer = FrameSlicer(format: _format);
      final ring = SpscRingBuffer(1024);
      expect(slicer.drain(ring), isEmpty);
      expect(ring.isEmpty, isTrue);
    });

    test('帧头编解码往返保持全部字段', () {
      final frame = AudioFrame(
        seq: 0xDEADBEEF & 0xFFFFFFFF,
        timestampMs: 123456789,
        samples: Uint8List.fromList([1, 2, 3, 4, 5]),
        flags: AudioFrameFlags.gapBefore,
      );

      final packet = FrameHeaderCodec.encode(frame);
      expect(packet.length, frameHeaderBytes + 5);

      final decoded = FrameHeaderCodec.decode(packet);
      expect(decoded.seq, frame.seq);
      expect(decoded.timestampMs, frame.timestampMs);
      expect(decoded.flags, frame.flags);
      expect(decoded.samples, frame.samples);
    });

    test('帧头小端序布局符合文档第 7 节', () {
      final packet = FrameHeaderCodec.encode(
        AudioFrame(seq: 1, timestampMs: 2, samples: Uint8List(3)),
      );
      final view = ByteData.sublistView(packet);
      expect(view.getUint32(0, Endian.little), 1);
      expect(view.getUint32(4, Endian.little), 2);
      expect(view.getUint16(8, Endian.little), 3);
      expect(view.getUint16(10, Endian.little), 0);
    });

    test('decode 对长度异常的包抛 ArgumentError', () {
      expect(
        () => FrameHeaderCodec.decodeHeader(Uint8List(4)),
        throwsArgumentError,
      );
      final packet = FrameHeaderCodec.encode(
        AudioFrame(seq: 0, timestampMs: 0, samples: Uint8List(8)),
      );
      expect(
        () => FrameHeaderCodec.decode(Uint8List.sublistView(packet, 0, 15)),
        throwsArgumentError,
      );
    });

    test('非法构造参数抛出 ArgumentError', () {
      expect(
        () => FrameSlicer(format: _format, frameDurationMs: 0),
        throwsArgumentError,
      );
      // 8kHz 以下且 1ms 帧时长可能换算出 0B，覆盖 frameBytes <= 0 分支。
      expect(
        () => FrameSlicer(
          format: const AudioFormat(
            sampleRateHz: 100,
            channelCount: 1,
            bitsPerSample: 16,
          ),
          frameDurationMs: 1,
        ),
        throwsArgumentError,
      );
    });
  });
}
