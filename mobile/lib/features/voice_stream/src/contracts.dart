/// voice_stream 的核心契约：读（采集）、传（上行/回包）、放（播放）。
///
/// UI 与控制器只允许依赖本文件声明的抽象；平台实现
/// （Oboe / RemoteIO / WSS / WebRTC 传输）各自挂接在这些接口后面。
library;

import 'dart:typed_data';

/// 描述一路 PCM 音频流的格式。
class AudioFormat {
  /// 创建音频格式描述。
  const AudioFormat({
    required this.sampleRateHz,
    required this.channelCount,
    required this.bitsPerSample,
  });

  /// 采样率，单位 Hz。
  final int sampleRateHz;

  /// 声道数，实时语音链路固定为 1（单声道）。
  final int channelCount;

  /// 每样本位深，实时语音链路固定为 16。
  final int bitsPerSample;

  /// 每帧持续 [durationMs] 毫秒时的字节数。
  int bytesPerFrame(int durationMs) {
    final bytesPerSecond =
        sampleRateHz * channelCount * (bitsPerSample ~/ 8);
    return bytesPerSecond * durationMs ~/ 1000;
  }
}

/// 一帧定长 PCM 音频。
class AudioFrame {
  /// 创建音频帧。
  const AudioFrame({
    required this.seq,
    required this.timestampMs,
    required this.samples,
  });

  /// 单调递增的帧序号；跳号即表示上游丢帧。
  final int seq;

  /// 采集时刻的毫秒时间戳，用于端到端延迟测量。
  final int timestampMs;

  /// PCM 样本（16 位小端，交错排布）。
  final Uint8List samples;
}

/// 采集端契约：把麦克风抽象为可随时中断的帧流。
abstract interface class MicSource {
  /// 开始采集并返回帧流；重复调用视为错误。
  ///
  /// 实现必须保证：采集回调线程不阻塞 UI，弱网/背压下按丢帧策略前行。
  Stream<AudioFrame> start(AudioFormat format, int frameDurationMs);

  /// 停止采集并释放设备；重复调用应为幂等。
  Future<void> stop();
}

/// 播放端契约：把扬声器抽象为帧写入点。
abstract interface class AudioSink {
  /// 写入一帧待播放音频，返回是否被立即接受（缓冲满则拒绝）。
  bool write(AudioFrame frame);

  /// 播放缓冲当前欠载的字节数，用于欠载策略调参。
  int get underrunBytes;

  /// 关闭播放并释放设备；重复调用应为幂等。
  Future<void> stop();
}

/// 服务端回包事件。
sealed class TransportEvent {
  /// 创建传输事件。
  const TransportEvent();
}

/// 服务端发来的识别/评测结果（文本片段或结构化负载）。
class TransportMessage extends TransportEvent {
  /// 创建消息事件。
  const TransportMessage(this.payload);

  /// JSON 文本负载，具体结构由服务端协议定义。
  final String payload;
}

/// 传输层统计：本端丢帧或网络丢包的量化出口。
class TransportStats extends TransportEvent {
  /// 创建统计事件。
  const TransportStats({
    required this.sentFrames,
    required this.droppedFrames,
    required this.bufferedBytes,
  });

  /// 已发送的音频帧数。
  final int sentFrames;

  /// 采集侧或发送侧丢弃的音频帧数。
  final int droppedFrames;

  /// 发送缓冲中当前积压的字节数。
  final int bufferedBytes;
}

/// 上行传输契约：音频帧外发 + 服务端事件回流。
abstract interface class AudioTransport {
  /// 发送一帧音频；实现必须非阻塞，丢弃时通过 [events] 体现。
  void sendFrame(AudioFrame frame);

  /// 服务端事件流：识别结果、评测结果、统计。
  Stream<TransportEvent> get events;

  /// 关闭连接；重复调用应为幂等。
  Future<void> close();
}
