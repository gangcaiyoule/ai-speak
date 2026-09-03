/// @file WSS 回声传输：R6 回包链路的协议打通实现。
///
/// @brief 把 [FrameHeaderCodec] 编码的音频帧经 WebSocket 上行，服务端
///        原样回显；二进制回显解码为 [TransportAudioFrame]，文本回显
///        解码为 [TransportMessage]。
///
/// 回声协议 v0（服务端 echo 一切原样回传，见 server/internal/voiceecho）：
///
/// - 客户端二进制包 = 12B 帧头 + 负载 → 服务端原样回传 → [TransportAudioFrame]。
/// - 客户端文本 = JSON 控制帧（start/finish/cancel 等）→ 服务端原样回传
///   → [TransportMessage]。
///
/// 弱网语义：连接建立前的上行帧先入待发队列（容量 [pendingFrameLimit]，
/// 溢出丢最旧并累计 droppedFrames）；断连不自动重连——重连是会话层
/// `retryable` 语义的事，本层把断连以流错误形式暴露给上层。
library;

import 'dart:async';
import 'dart:typed_data';

import 'package:web_socket_channel/web_socket_channel.dart';

import 'contracts.dart';
import 'frame_slicer.dart';

/// @brief WSS 回声传输实现（[TextCapableTransport]）。
class WssEchoTransport implements TextCapableTransport {
  /// @brief 构造 WSS 回声传输。
  ///
  /// @param uri 回声服务端 WebSocket 地址。
  /// @param pendingFrameLimit 连接建立前待发队列的帧数上限，溢出丢最旧。
  /// @param statsEvery 每收到多少帧回传音频发一条 [TransportStats]。
  /// @param channelFactory WebSocket 通道工厂；缺省 [WebSocketChannel.connect]，
  ///        测试可注入。
  WssEchoTransport({
    required this.uri,
    this.pendingFrameLimit = 64,
    this.statsEvery = 100,
    WebSocketChannel Function(Uri uri)? channelFactory,
  }) : _channelFactory = channelFactory ?? WebSocketChannel.connect;

  /// @brief 回声服务端地址。
  final Uri uri;

  /// @brief 连接建立前待发队列帧数上限。
  final int pendingFrameLimit;

  /// @brief 统计事件发放周期（单位：收到的回传音频帧数）。
  final int statsEvery;

  final WebSocketChannel Function(Uri) _channelFactory;

  final StreamController<TransportEvent> _events =
      StreamController<TransportEvent>.broadcast();
  final List<AudioFrame> _pending = <AudioFrame>[];
  int _pendingBytes = 0;
  WebSocketChannel? _channel;
  Future<void>? _connecting;
  bool _open = false; ///< WebSocket 已就绪，可直接发送。
  bool _closed = false; ///< 已调用 close()，进入终态。
  int _sentFrames = 0;
  int _droppedFrames = 0;

  @override
  Stream<TransportEvent> get events => _events.stream;

  /// @brief 是否已调用 close()。
  bool get isClosed => _closed;

  /// @brief 等待 WebSocket 就绪；连接失败时以异常完成。
  ///
  /// 未发起连接时先发起；已在连接中则复用同一 future。
  Future<void> get connected {
    _ensureConnecting();
    return _connecting ?? Future.value();
  }

  @override
  void sendFrame(AudioFrame frame) {
    if (_closed) {
      _droppedFrames++;
      return;
    }
    _ensureConnecting();
    if (!_open) {
      if (_pending.length >= pendingFrameLimit) {
        final oldest = _pending.removeAt(0);
        _pendingBytes -= oldest.samples.length;
        _droppedFrames++;
      }
      _pending.add(frame);
      _pendingBytes += frame.samples.length;
      return;
    }
    _sendNow(frame);
  }

  @override
  void sendText(String payload) {
    if (_closed) {
      return;
    }
    _ensureConnecting();
    if (!_open) {
      // 控制帧不排队：连接未就绪时的控制帧丢弃，由调用方（会话层）重试。
      _droppedFrames++;
      return;
    }
    _channel!.sink.add(payload);
  }

  @override
  Future<void> close() async {
    if (_closed) {
      return;
    }
    _closed = true;
    _emitStats();
    await _connecting?.catchError((Object _) {});
    final channel = _channel;
    _channel = null;
    _open = false;
    _pending.clear();
    _pendingBytes = 0;
    await channel?.sink.close();
    await _events.close();
  }

  /// @brief 内部：按需发起连接（幂等）。
  void _ensureConnecting() {
    if (_closed || _connecting != null) {
      return;
    }
    _connecting = _connect();
  }

  Future<void> _connect() async {
    final WebSocketChannel channel;
    try {
      channel = _channelFactory(uri);
    } catch (error) {
      _connecting = null;
      if (!_closed) {
        _events.addError(error);
      }
      return;
    }
    _channel = channel;
    channel.stream.listen(
      _onData,
      onError: _onError,
      onDone: _onDone,
      cancelOnError: true,
    );
    try {
      await channel.ready;
    } catch (error) {
      // 连接失败：暴露错误、复位状态，允许下一次 sendFrame 重新连接。
      _channel = null;
      _open = false;
      _connecting = null;
      _droppedFrames += _pending.length;
      _pending.clear();
      _pendingBytes = 0;
      if (!_closed) {
        _events.addError(error);
      }
      return;
    }
    _open = true;
    _flushPending();
  }

  /// @brief 内部：连接就绪后冲刷待发队列。
  void _flushPending() {
    while (_pending.isNotEmpty) {
      _sendNow(_pending.removeAt(0));
    }
    _pendingBytes = 0;
  }

  void _sendNow(AudioFrame frame) {
    _channel!.sink.add(FrameHeaderCodec.encode(frame));
    _sentFrames++;
  }

  /// @brief 内部：服务端回包分发。
  void _onData(dynamic data) {
    Uint8List? bytes;
    if (data is Uint8List) {
      bytes = data;
    } else if (data is List<int>) {
      bytes = Uint8List.fromList(data);
    }
    if (bytes != null) {
      _events.add(TransportAudioFrame(FrameHeaderCodec.decode(bytes)));
      _receivedAudioFrames++;
      if (statsEvery > 0 && _receivedAudioFrames % statsEvery == 0) {
        _emitStats();
      }
    } else if (data is String) {
      _events.add(TransportMessage(data));
    }
  }

  int _receivedAudioFrames = 0; ///< 已收到的回传音频帧计数（统计用）。

  /// @brief 内部：连接错误——复位并允许重连，错误暴露给上层。
  void _onError(Object error) {
    _channel = null;
    _open = false;
    _connecting = null;
    _droppedFrames += _pending.length;
    _pending.clear();
    _pendingBytes = 0;
    if (!_closed) {
      _events.addError(error);
    }
  }

  /// @brief 内部：服务端关闭连接——对客户端即断连，以错误暴露。
  void _onDone() {
    _onError(StateError('回声服务端关闭了连接'));
  }

  void _emitStats() {
    if (_events.isClosed) {
      return;
    }
    _events.add(
      TransportStats(
        sentFrames: _sentFrames,
        droppedFrames: _droppedFrames,
        bufferedBytes: _pendingBytes,
      ),
    );
  }
}
