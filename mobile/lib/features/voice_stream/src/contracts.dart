/// @file 契约定义：读（采集）、传（上行/回包）、放（播放）。
///
/// @brief voice_stream 的核心契约。
///
/// UI 与控制器只允许依赖本文件声明的抽象；平台实现
/// （Oboe / RemoteIO / WSS / WebRTC 传输）各自挂接在这些接口后面。
library;

import 'dart:typed_data';

/// @brief 描述一路 PCM 音频流的格式。
class AudioFormat {
  /// @brief 构造音频格式值对象。
  ///
  /// @param sampleRateHz 采样率，单位 Hz。
  /// @param channelCount 声道数，实时语音链路固定为 1（单声道）。
  /// @param bitsPerSample 每样本位深，实时语音链路固定为 16。
  const AudioFormat({
    required this.sampleRateHz,
    required this.channelCount,
    required this.bitsPerSample,
  });

  /// @brief 采样率，单位 Hz。
  final int sampleRateHz;

  /// @brief 声道数，实时语音链路固定为 1（单声道）。
  final int channelCount;

  /// @brief 每样本位深，实时语音链路固定为 16。
  final int bitsPerSample;

  /// @brief 计算一帧持续 [durationMs] 毫秒时的字节数。
  ///
  /// @param durationMs 帧时长，单位毫秒。
  /// @return 该时长的 PCM 数据字节数，向下取整。
  int bytesPerFrame(int durationMs) {
    final bytesPerSecond =
        sampleRateHz * channelCount * (bitsPerSample ~/ 8);
    return bytesPerSecond * durationMs ~/ 1000;
  }
}

/// @brief 一帧定长 PCM 音频。
///
/// 是贯穿采集、传输、播放的统一数据单元。
class AudioFrame {
  /// @brief 构造音频帧。
  ///
  /// @param seq 单调递增帧序号；跳号即表示上游丢帧。
  /// @param timestampMs 采集时刻的毫秒时间戳，用于端到端延迟测量。
  /// @param samples PCM 样本（16 位小端，交错排布）。
  const AudioFrame({
    required this.seq,
    required this.timestampMs,
    required this.samples,
  });

  /// @brief 单调递增的帧序号；跳号即表示上游丢帧。
  final int seq;

  /// @brief 采集时刻的毫秒时间戳，用于端到端延迟测量。
  final int timestampMs;

  /// @brief PCM 样本（16 位小端，交错排布）。
  final Uint8List samples;
}

/// @brief 采集端契约：把麦克风抽象为可随时中断的帧流。
abstract interface class MicSource {
  /// @brief 开始采集并返回帧流。
  ///
  /// 实现必须保证：采集回调线程不阻塞 UI，弱网/背压下按丢帧策略前行。
  ///
  /// @param format 期望的音频格式；实现按协商后的实际格式交付帧。
  /// @param frameDurationMs 目标帧时长，单位毫秒。
  /// @return 音频帧流；流关闭即采集结束。
  Stream<AudioFrame> start(AudioFormat format, int frameDurationMs);

  /// @brief 停止采集并释放设备；重复调用应为幂等。
  ///
  /// @return 释放完成时 future 完成。
  Future<void> stop();
}

/// @brief 播放端契约：把扬声器抽象为帧写入点。
abstract interface class AudioSink {
  /// @brief 写入一帧待播放音频。
  ///
  /// @param frame 待播放音频帧。
  /// @return 是否被立即接受；缓冲满则拒绝，永不阻塞。
  bool write(AudioFrame frame);

  /// @brief 播放缓冲当前欠载的字节数，用于欠载策略调参。
  int get underrunBytes;

  /// @brief 关闭播放并释放设备；重复调用应为幂等。
  ///
  /// @return 释放完成时 future 完成。
  Future<void> stop();
}

/// @brief 服务端回包事件基类。
sealed class TransportEvent {
  /// @brief 构造传输事件。
  const TransportEvent();
}

/// @brief 服务端发来的识别/评测结果。
class TransportMessage extends TransportEvent {
  /// @brief 构造消息事件。
  ///
  /// @param payload JSON 文本负载，具体结构由服务端协议定义。
  const TransportMessage(this.payload);

  /// @brief JSON 文本负载，具体结构由服务端协议定义。
  final String payload;
}

/// @brief 传输层统计：本端丢帧或网络丢包的量化出口。
class TransportStats extends TransportEvent {
  /// @brief 构造统计事件。
  ///
  /// @param sentFrames 已发送的音频帧数。
  /// @param droppedFrames 采集侧或发送侧丢弃的音频帧数。
  /// @param bufferedBytes 发送缓冲中当前积压的字节数。
  const TransportStats({
    required this.sentFrames,
    required this.droppedFrames,
    required this.bufferedBytes,
  });

  /// @brief 已发送的音频帧数。
  final int sentFrames;

  /// @brief 采集侧或发送侧丢弃的音频帧数。
  final int droppedFrames;

  /// @brief 发送缓冲中当前积压的字节数。
  final int bufferedBytes;
}

/// @brief 上行传输契约：音频帧外发 + 服务端事件回流。
///
/// 传输栈（WSS / WebRTC）是接口后的可替换件。
abstract interface class AudioTransport {
  /// @brief 发送一帧音频。
  ///
  /// 实现必须非阻塞；丢弃行为通过 [events] 中的统计事件体现。
  ///
  /// @param frame 待发送音频帧。
  void sendFrame(AudioFrame frame);

  /// @brief 服务端事件流：识别结果、评测结果、统计。
  Stream<TransportEvent> get events;

  /// @brief 关闭连接；重复调用应为幂等。
  ///
  /// @return 关闭完成时 future 完成。
  Future<void> close();
}
