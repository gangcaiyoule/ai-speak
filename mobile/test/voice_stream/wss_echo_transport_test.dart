import 'dart:async';
import 'dart:io';
import 'dart:typed_data';

import 'package:flutter_test/flutter_test.dart';
import 'package:ai_speak/features/voice_stream/src/contracts.dart';
import 'package:ai_speak/features/voice_stream/src/wss_echo_transport.dart';

/// @brief 起一个进程内 WSS 回声服务器：一切消息原样回传。
///
/// @param upgradeDelay 模拟握手耗时的延迟，用于确定性测试待发队列。
Future<HttpServer> startEchoServer({Duration upgradeDelay = Duration.zero}) async {
  final server = await HttpServer.bind(InternetAddress.loopbackIPv4, 0);
  server.listen((request) async {
    if (upgradeDelay != Duration.zero) {
      await Future<void>.delayed(upgradeDelay);
    }
    final ws = await WebSocketTransformer.upgrade(request);
    ws.listen(ws.add);
  });
  return server;
}

AudioFrame frame(int seq) => AudioFrame(
      seq: seq,
      timestampMs: 1000 + seq * 20,
      samples: Uint8List.fromList(List.generate(64, (i) => (i + seq) % 256)),
    );

void main() {
  group('WssEchoTransport', () {
    test('二进制帧经真实回声服务器往返，字段完整还原', () async {
      final server = await startEchoServer();
      final transport = WssEchoTransport(
        uri: Uri.parse('ws://127.0.0.1:${server.port}'),
      );

      final received = <TransportAudioFrame>[];
      final allReceived = Completer<void>();
      final sub = transport.events.listen((event) {
        if (event is TransportAudioFrame) {
          received.add(event);
          if (received.length == 3 && !allReceived.isCompleted) {
            allReceived.complete();
          }
        }
      });

      // 不等连接直接发：覆盖待发队列路径。
      transport.sendFrame(frame(0));
      transport.sendFrame(frame(1));
      transport.sendFrame(frame(2));
      await transport.connected;
      await allReceived.future.timeout(const Duration(seconds: 5));

      expect(
        [for (final e in received) e.frame.seq],
        [0, 1, 2],
      );
      expect(received[1].frame.timestampMs, 1020);
      expect(received[1].frame.samples, frame(1).samples);
      expect(received[1].frame.flags, AudioFrameFlags.none);

      await sub.cancel();
      await transport.close();
      await server.close(force: true);
    });

    test('文本控制帧往返为 TransportMessage', () async {
      final server = await startEchoServer();
      final transport = WssEchoTransport(
        uri: Uri.parse('ws://127.0.0.1:${server.port}'),
      );

      final messages = <TransportMessage>[];
      final gotMessage = Completer<void>();
      final sub = transport.events.listen((event) {
        if (event is TransportMessage) {
          messages.add(event);
          if (!gotMessage.isCompleted) {
            gotMessage.complete();
          }
        }
      });

      await transport.connected;
      transport.sendText('{"type":"start"}');
      await gotMessage.future.timeout(const Duration(seconds: 5));
      expect(messages.single.payload, '{"type":"start"}');

      await sub.cancel();
      await transport.close();
      await server.close(force: true);
    });

    test('连接就绪前超出待发上限丢最旧帧，就绪后冲刷余量', () async {
      // 握手延迟 300ms，保证发送全部发生在连接就绪前。
      final server = await startEchoServer(
        upgradeDelay: const Duration(milliseconds: 300),
      );
      final transport = WssEchoTransport(
        uri: Uri.parse('ws://127.0.0.1:${server.port}'),
        pendingFrameLimit: 2,
      );

      final received = <TransportAudioFrame>[];
      final allReceived = Completer<void>();
      final stats = <TransportStats>[];
      final sub = transport.events.listen((event) {
        if (event is TransportAudioFrame) {
          received.add(event);
          if (received.length == 2 && !allReceived.isCompleted) {
            allReceived.complete();
          }
        } else if (event is TransportStats) {
          stats.add(event);
        }
      });

      transport.sendFrame(frame(0));
      transport.sendFrame(frame(1));
      transport.sendFrame(frame(2));
      transport.sendFrame(frame(3)); // 上限 2，丢最旧 2 帧。

      await transport.connected;
      await allReceived.future.timeout(const Duration(seconds: 5));

      // 只回传了未丢的 2 帧。
      expect([for (final e in received) e.frame.seq], [2, 3]);

      await sub.cancel();
      await transport.close();
      final finalStats = stats.last;
      expect(finalStats.droppedFrames, 2);
      expect(finalStats.sentFrames, 2);
      await server.close(force: true);
    });

    test('服务端断连以流错误暴露', () async {
      final server = await startEchoServer();
      final transport = WssEchoTransport(
        uri: Uri.parse('ws://127.0.0.1:${server.port}'),
      );
      await transport.connected;

      final errorEvent = expectLater(transport.events, emitsError(isA<StateError>()));
      await server.close(force: true);
      await errorEvent.timeout(const Duration(seconds: 5));

      await transport.close();
    });

    test('close 幂等', () async {
      final server = await startEchoServer();
      final transport = WssEchoTransport(
        uri: Uri.parse('ws://127.0.0.1:${server.port}'),
      );
      await transport.connected;
      await transport.close();
      await transport.close(); // 不抛异常即幂等。
      expect(transport.isClosed, isTrue);
      await server.close(force: true);
    });
  });
}
