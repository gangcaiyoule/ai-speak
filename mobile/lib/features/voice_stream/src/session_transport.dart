/// @file 会话适配层：把 [VoiceSession] 生命周期映射到回声协议控制帧。
///
/// @brief R6 的 [VoiceSession] 实现：叠加在 [TextCapableTransport] 之上，
///        用文本控制帧 start/finish/cancel 驱动回声服务端，断连映射为
///        [SessionFailed]。
///
/// 回声协议下的语义约定（真实云服务接入时由对应协议实现替换）：
///
/// - `start`：订阅传输事件流 + 发送 start 控制帧；[onReady] 完成后发
///   [SessionStarted]。
/// - 传输层 [TransportMessage] 在 active 阶段映射为 [SessionPartial]；
///   在 finishing 阶段收到的第一条文本回显视为终态回执，映射为
///   [SessionFinal]（回声服务端原样回显 finish 控制帧）。
/// - 传输层 [TransportStats] 映射为 [SessionStats]；[TransportAudioFrame]
///   不进会话层（回声音频由播放/UI 直接订阅传输事件流）。
/// - 传输层流错误或关闭 → [SessionFailed(kind: transport, retryable: true)]
///   并进入 closed。
/// - finish 超过 [finishTimeout] 未收到终态回执 → SessionFailed(kind:
///   finish-timeout, retryable: true)。
library;

import 'dart:async';

import 'contracts.dart';
import 'session.dart';

/// @brief 基于文本控制帧传输的 [VoiceSession] 实现。
class TransportVoiceSession implements VoiceSession {
  /// @brief 构造会话。
  ///
  /// @param transport 文本能力传输（WSS 回声或真实云协议实现）。
  /// @param config 会话配置（幂等键 + 音频格式）。
  /// @param onReady 传输就绪探针；start() 等待其完成后发
  ///        [SessionStarted]。传 null 表示传输同步可用。
  /// @param finishTimeout finish 后等待终态回执的超时，超时按可重试失败处理。
  TransportVoiceSession({
    required TextCapableTransport transport,
    required this.config,
    Future<void> Function()? onReady,
    this.finishTimeout = const Duration(seconds: 10),
  }) : _transport = transport,
       _onReady = onReady;

  /// @brief 会话配置（幂等键 + 音频格式）。
  @override
  final VoiceSessionConfig config;

  /// @brief 终态回执等待超时。
  final Duration finishTimeout;

  final TextCapableTransport _transport;
  final Future<void> Function()? _onReady;
  final VoiceSessionLifecycle _lifecycle = VoiceSessionLifecycle();
  final StreamController<VoiceSessionEvent> _events =
      StreamController<VoiceSessionEvent>.broadcast();
  StreamSubscription<TransportEvent>? _subscription;
  Completer<void>? _finishWaiter;

  @override
  VoiceSessionPhase get phase => _lifecycle.phase;

  @override
  Stream<VoiceSessionEvent> get events => _events.stream;

  /// @brief 开始会话：idle → active，建立传输事件订阅并发送 start 控制帧。
  ///
  /// @throws StateError 会话已开始过。
  Future<void> start() async {
    _lifecycle.begin();
    _subscription = _transport.events.listen(
      _onTransportEvent,
      onError: _onTransportError,
      onDone: _onTransportDone,
      cancelOnError: true,
    );
    _transport.sendText(
      '{"type":"start","idempotencyKey":"${config.idempotencyKey}"}',
    );
    final ready = _onReady;
    if (ready != null) {
      await ready();
    }
    if (!_lifecycle.isClosed) {
      _events.add(const SessionStarted());
    }
  }

  @override
  void sendFrame(AudioFrame frame) {
    if (!_lifecycle.canSendFrame) {
      throw StateError('sendFrame 仅允许在 active 阶段，当前 ${_lifecycle.phase}');
    }
    _transport.sendFrame(frame);
  }

  @override
  Future<void> finish() async {
    _lifecycle.requestFinish();
    _transport.sendText('{"type":"finish","idempotencyKey":"${config.idempotencyKey}"}');
    final waiter = _finishWaiter = Completer<void>();
    var timedOut = false;
    final timer = Timer(finishTimeout, () {
      if (waiter.isCompleted) {
        return;
      }
      timedOut = true;
      unawaited(_fail('finish-timeout')); // _fail 内部会完成 waiter。
    });
    try {
      await waiter.future;
    } finally {
      timer.cancel();
    }
    if (timedOut) {
      return; // _fail 已进入 closed 并关闭事件流。
    }
    await _teardownTransport();
    if (!_events.isClosed) {
      final stats = _lastStats;
      _events.add(SessionStats(
        sentFrames: stats?.sentFrames ?? 0,
        droppedFrames: stats?.droppedFrames ?? 0,
        bufferedBytes: stats?.bufferedBytes ?? 0,
      ));
      await _events.close();
    }
  }

  @override
  Future<void> cancel() async {
    if (_lifecycle.isClosed) {
      return; // 幂等。
    }
    final wasActiveOrFinishing = !_lifecycle.isClosed;
    _lifecycle.abort();
    if (_finishWaiter?.isCompleted != true) {
      _finishWaiter?.complete();
    }
    if (wasActiveOrFinishing) {
      _transport.sendText('{"type":"cancel","idempotencyKey":"${config.idempotencyKey}"}');
    }
    await _teardownTransport();
    await _events.close();
  }

  TransportStats? _lastStats; ///< 最近一次传输层统计。

  /// @brief 内部：传输事件分发。
  void _onTransportEvent(TransportEvent event) {
    switch (event) {
      case TransportMessage(:final payload):
        if (_lifecycle.phase == VoiceSessionPhase.finishing) {
          _events.add(SessionFinal(payload));
          _lifecycle.complete();
          if (_finishWaiter?.isCompleted != true) {
            _finishWaiter!.complete();
          }
        } else {
          _events.add(SessionPartial(payload));
        }
      case TransportAudioFrame():
        break; // 回声音频不进会话层，播放/UI 直接订阅传输事件流。
      case TransportStats(:final sentFrames, :final droppedFrames, :final bufferedBytes):
        _lastStats = event;
        _events.add(
          SessionStats(
            sentFrames: sentFrames,
            droppedFrames: droppedFrames,
            bufferedBytes: bufferedBytes,
          ),
        );
    }
  }

  /// @brief 内部：传输层错误一律映射为可重试失败。
  void _onTransportError(Object error) {
    _fail('transport');
  }

  /// @brief 内部：传输层事件流关闭（非本端发起）视作断连失败。
  void _onTransportDone() {
    if (!_lifecycle.isClosed) {
      _fail('transport');
    }
  }

  /// @brief 内部：进入失败终态并收尾。
  Future<void> _fail(String kind) async {
    _events.add(SessionFailed(kind: kind, retryable: true));
    _lifecycle.abort();
    final waiter = _finishWaiter;
    if (waiter != null && !waiter.isCompleted) {
      waiter.complete();
    }
    await _teardownTransport();
    await _events.close();
  }

  /// @brief 内部：取消订阅并关闭传输（幂等）。
  Future<void> _teardownTransport() async {
    await _subscription?.cancel();
    _subscription = null;
    await _transport.close();
  }
}
