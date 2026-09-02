/// @file 会话层契约：一次语音交互的生命周期、事件与失败模型。
///
/// @brief voice_stream 的会话层抽象，叠加在 [AudioTransport] 之上。
///
/// 语义参照上游项目 XE3-ESL 已验证的 voice-input 会话协议
/// （start/finish/cancel 控制帧、幂等键、partial/final 区分、
/// kind + retryable 失败模型），但本文件只定义抽象，不绑定任何
/// 传输栈；控制帧到 WSS/WebRTC 具体报文的映射由传输实现（R6）
/// 自行完成。现有 AudioTransport 契约保持不变，UI 与控制器可以
/// 只依赖本层而完全不感知传输细节。
library;

import 'contracts.dart';

/// @brief 会话生命周期阶段。
enum VoiceSessionPhase {
  /// @brief 尚未开始，或已关闭后的初始不可用态。
  idle,

  /// @brief 会话已建立，可以发送音频帧。
  active,

  /// @brief 已请求优雅结束，正在等待服务端终态事件；此阶段不再收帧。
  finishing,

  /// @brief 会话终态（完成/失败/取消/关闭），不可复用；重试须新开会话。
  closed,
}

/// @brief 一次会话的配置值对象。
///
/// @note [idempotencyKey] 用于服务端幂等去重：弱网下客户端重开同名会话
///       时，服务端据此识别重复上行而不产生重复结果。约束与上游协议
///       对齐：长度 8–128，首尾不得含空白字符。
class VoiceSessionConfig {
  /// @brief 构造会话配置，并对幂等键做静态校验。
  ///
  /// @param idempotencyKey 幂等键，长度 8–128，首尾不得含空白。
  /// @param format 上行音频格式；实现按协商后的实际格式交付回包。
  VoiceSessionConfig({required this.idempotencyKey, required this.format}) {
    if (idempotencyKey.length < 8 || idempotencyKey.length > 128) {
      throw ArgumentError.value(
        idempotencyKey,
        'idempotencyKey',
        '长度必须在 8–128 之间',
      );
    }
    if (idempotencyKey.trim() != idempotencyKey) {
      throw ArgumentError.value(
        idempotencyKey,
        'idempotencyKey',
        '首尾不得含空白字符',
      );
    }
  }

  /// @brief 幂等键：同一逻辑会话跨重连重试保持不变。
  final String idempotencyKey;

  /// @brief 上行音频格式（请求值，非保证值，见接口文档 8.1 节）。
  final AudioFormat format;
}

/// @brief 会话层事件基类。
sealed class VoiceSessionEvent {
  /// @brief 构造会话事件。
  const VoiceSessionEvent();
}

/// @brief 会话已建立，服务端确认可以上行音频。
class SessionStarted extends VoiceSessionEvent {
  /// @brief 构造会话开始事件。
  const SessionStarted();
}

/// @brief 服务端中间识别/评测结果（非终态）。
class SessionPartial extends VoiceSessionEvent {
  /// @brief 构造中间结果事件。
  ///
  /// @param payload JSON 文本负载，结构由服务端协议定义。
  const SessionPartial(this.payload);

  /// @brief JSON 文本负载。
  final String payload;
}

/// @brief 服务端最终识别/评测结果（终态之一）。
class SessionFinal extends VoiceSessionEvent {
  /// @brief 构造最终结果事件。
  ///
  /// @param payload JSON 文本负载，结构由服务端协议定义。
  const SessionFinal(this.payload);

  /// @brief JSON 文本负载。
  final String payload;
}

/// @brief 会话失败（终态之一）。
class SessionFailed extends VoiceSessionEvent {
  /// @brief 构造失败事件。
  ///
  /// @param kind 失败类别，由服务端协议定义。
  /// @param retryable 是否可用同一幂等键重开会话重试。
  const SessionFailed({required this.kind, required this.retryable});

  /// @brief 失败类别。
  final String kind;

  /// @brief 是否可用同一幂等键重开会话重试。
  final bool retryable;
}

/// @brief 会话层量化统计：弱网调参与延迟观测的统一出口。
class SessionStats extends VoiceSessionEvent {
  /// @brief 构造统计事件。
  ///
  /// @param sentFrames 已发送音频帧数。
  /// @param droppedFrames 采集侧或发送侧丢弃的音频帧数。
  /// @param bufferedBytes 发送缓冲中当前积压字节数。
  const SessionStats({
    required this.sentFrames,
    required this.droppedFrames,
    required this.bufferedBytes,
  });

  /// @brief 已发送音频帧数。
  final int sentFrames;

  /// @brief 采集侧或发送侧丢弃的音频帧数。
  final int droppedFrames;

  /// @brief 发送缓冲中当前积压字节数。
  final int bufferedBytes;
}

/// @brief 一次语音交互的会话契约。
///
/// 生命周期：`start → active → (finish → 等终态 | cancel) → closed`。
/// 实现必须保证：所有方法幂等或状态机守卫明确；弱网断连映射为
/// [SessionFailed]（`retryable` 语义见 [SessionFailed.retryable]），
/// 而不是静默吞掉。
abstract interface class VoiceSession {
  /// @brief 本会话的配置（幂等键 + 音频格式）。
  VoiceSessionConfig get config;

  /// @brief 当前生命周期阶段。
  VoiceSessionPhase get phase;

  /// @brief 发送一帧上行音频。
  ///
  /// 仅在 [VoiceSessionPhase.active] 阶段允许调用；其余阶段抛
  /// StateError。实现必须非阻塞，丢弃行为经 [events] 的统计事件体现。
  ///
  /// @param frame 待发送音频帧。
  void sendFrame(AudioFrame frame);

  /// @brief 优雅结束：通知服务端不再有音频，等待终态事件。
  ///
  /// @return 收到终态事件（final/failed）并进入 closed 后完成。
  Future<void> finish();

  /// @brief 立即中断：不等服务端结果，尽快释放资源。
  ///
  /// @return 会话进入 closed 后完成；对已关闭会话调用为幂等成功。
  Future<void> cancel();

  /// @brief 会话事件流：开始、中间/最终结果、失败、统计。
  Stream<VoiceSessionEvent> get events;
}

/// @brief 会话生命周期状态机（纯 Dart，供传输实现复用与单测）。
///
/// @note 职责仅限阶段迁移守卫；帧收发与事件分发由实现自行组合。
final class VoiceSessionLifecycle {
  VoiceSessionPhase _phase = VoiceSessionPhase.idle;

  /// @brief 当前阶段。
  VoiceSessionPhase get phase => _phase;

  /// @brief 是否处于可发送音频帧的阶段。
  bool get canSendFrame => _phase == VoiceSessionPhase.active;

  /// @brief 会话是否已到终态。
  bool get isClosed => _phase == VoiceSessionPhase.closed;

  /// @brief 进入 active；重复开始视为实现错误。
  void begin() {
    if (_phase != VoiceSessionPhase.idle) {
      throw StateError('begin() 仅允许从 idle 进入，当前 $_phase');
    }
    _phase = VoiceSessionPhase.active;
  }

  /// @brief 请求优雅结束：active → finishing。
  void requestFinish() {
    if (_phase != VoiceSessionPhase.active) {
      throw StateError('finish() 仅允许从 active 进入，当前 $_phase');
    }
    _phase = VoiceSessionPhase.finishing;
  }

  /// @brief 收到终态事件，会话完结：finishing → closed。
  void complete() {
    if (_phase != VoiceSessionPhase.finishing) {
      throw StateError('complete() 仅允许从 finishing 进入，当前 $_phase');
    }
    _phase = VoiceSessionPhase.closed;
  }

  /// @brief 失败或取消：active/finishing → closed；对 closed 幂等。
  void abort() {
    if (_phase == VoiceSessionPhase.closed) {
      return;
    }
    _phase = VoiceSessionPhase.closed;
  }
}
