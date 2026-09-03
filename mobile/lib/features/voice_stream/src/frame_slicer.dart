/// @file 切帧器与帧头编解码。
///
/// @brief 把任意长度的字节块按时间窗切成定长 [AudioFrame]，并提供
///        12 字节帧头的编解码（小端序，帧 = 帧头 12B + 负载）。
///
/// 切帧器是纯 Dart 无 IO 组件：输入是采集侧吐出的任意长度字节块
/// （通常来自 [SpscRingBuffer] 出口），输出定长帧流；残余不足一帧的
/// 字节先缓冲，流结束时用 @c flush 收尾出短帧。上游丢弃通过
/// @c markGap 显式告知，落在下一帧的 flags @c gapBefore 位上。
library;

import 'dart:typed_data';

import 'contracts.dart';
import 'ring_buffer.dart';

/// @brief 帧头大小（字节）：uint32 seq + uint32 timestamp_ms + uint16 size + uint16 flags。
const int frameHeaderBytes = 12;

/// @brief 12 字节帧头解析结果视图。
class FrameHeaderView {
  /// @brief 构造帧头视图。
  const FrameHeaderView({
    required this.seq,
    required this.timestampMs,
    required this.size,
    required this.flags,
  });

  /// @brief 帧序号。
  final int seq;

  /// @brief 采集时间戳，单位毫秒。
  final int timestampMs;

  /// @brief 负载长度（字节）。
  final int size;

  /// @brief 标志位，见 [AudioFrameFlags]。
  final int flags;
}

/// @brief 帧头编解码：小端序；传输字节 = 帧头([frameHeaderBytes]) + 负载。
abstract final class FrameHeaderCodec {
  /// @brief 把一帧编码为「帧头 + 负载」的传输字节。
  ///
  /// @param frame 待编码帧。
  /// @return 新分配的传输字节，长度为 [frameHeaderBytes] + 负载长度。
  static Uint8List encode(AudioFrame frame) {
    final packet = Uint8List(frameHeaderBytes + frame.samples.length);
    final view = ByteData.sublistView(packet);
    view.setUint32(0, frame.seq, Endian.little);
    view.setUint32(4, frame.timestampMs, Endian.little);
    view.setUint16(8, frame.samples.length, Endian.little);
    view.setUint16(10, frame.flags, Endian.little);
    packet.setAll(frameHeaderBytes, frame.samples);
    return packet;
  }

  /// @brief 解析传输字节前 [frameHeaderBytes] 字节的帧头。
  ///
  /// @param packet 传输字节，长度不足帧头大小时抛 ArgumentError。
  /// @return 帧头字段视图。
  static FrameHeaderView decodeHeader(Uint8List packet) {
    if (packet.length < frameHeaderBytes) {
      throw ArgumentError.value(
        packet.length,
        'packet',
        '长度不足帧头大小 $frameHeaderBytes',
      );
    }
    final view = ByteData.sublistView(packet, 0, frameHeaderBytes);
    return FrameHeaderView(
      seq: view.getUint32(0, Endian.little),
      timestampMs: view.getUint32(4, Endian.little),
      size: view.getUint16(8, Endian.little),
      flags: view.getUint16(10, Endian.little),
    );
  }

  /// @brief 把「帧头 + 负载」的传输字节还原为 [AudioFrame]。
  ///
  /// 负载与输入共享内存（零拷贝视图），调用方如需长期持有请自行拷贝。
  ///
  /// @param packet 传输字节；总长度必须等于帧头 + 帧头声明的负载长度。
  /// @return 还原出的音频帧。
  static AudioFrame decode(Uint8List packet) {
    final header = decodeHeader(packet);
    if (packet.length != frameHeaderBytes + header.size) {
      throw ArgumentError.value(
        packet.length,
        'packet',
        '总长度与帧头声明的负载长度 ${header.size} 不符',
      );
    }
    return AudioFrame(
      seq: header.seq,
      timestampMs: header.timestampMs,
      samples: Uint8List.sublistView(packet, frameHeaderBytes),
      flags: header.flags,
    );
  }
}

/// @brief 切帧器：任意长度字节块 → 定长 [AudioFrame] 流。
///
/// - 帧长按 [AudioFormat.bytesPerFrame] 由格式与帧时长换算，默认
///   20ms @ 16kHz/16bit/mono = 640B。
/// - seq 从 0 起连续递增；timestampMs 按帧在流内的位置推算：
///   首字节到达时刻 + 帧序 × 帧时长；@c flush 收尾后下一帧流重新取基准时刻。
/// - 残余不足一帧的字节缓冲，@c flush 时以短帧输出（flags 不变，负载
///   长度由帧头 size 如实携带）。
/// - 输出的 samples 一律是拷贝，不与内部缓冲共享内存。
class FrameSlicer {
  /// @brief 构造切帧器。
  ///
  /// @param format 上行音频格式；帧长 = format.bytesPerFrame(frameDurationMs)。
  /// @param frameDurationMs 帧时间窗，单位毫秒，必须为正数。
  /// @param clock 时间戳来源，返回毫秒时间戳；缺省本地墙钟。
  ///        测试可注入确定性时钟。
  FrameSlicer({
    required AudioFormat format,
    this.frameDurationMs = 20,
    int Function()? clock,
  })  : _clock = clock ?? _wallClock,
        _frameBytes = format.bytesPerFrame(frameDurationMs) {
    if (frameDurationMs <= 0) {
      throw ArgumentError.value(frameDurationMs, 'frameDurationMs', '必须为正数');
    }
    if (_frameBytes <= 0) {
      throw ArgumentError.value(
        _frameBytes,
        'frameBytes',
        '格式与帧时长换算结果必须为正数',
      );
    }
  }

  /// @brief 帧时间窗，单位毫秒。
  final int frameDurationMs;

  final int Function() _clock;
  final int _frameBytes;

  Uint8List _buf = Uint8List(_initialCapacity); ///< 残余字节累积缓冲。
  int _pendingLen = 0; ///< _buf 中待切字节数。
  int _seq = 0; ///< 下一帧序号。
  int? _baseMs; ///< 当前帧流的首字节基准时刻；flush 后置空。
  int _framesSinceBase = 0; ///< 基准时刻以来已产出的帧数。
  bool _pendingGap = false; ///< 是否给下一帧打 gapBefore 标志。

  static const int _initialCapacity = 4096;

  /// @brief 下一帧将使用的序号。
  int get nextSeq => _seq;

  /// @brief 当前缓冲中不足一帧的残余字节数。
  int get pendingBytes => _pendingLen;

  /// @brief 单帧负载字节数。
  int get frameBytes => _frameBytes;

  /// @brief 是否已缓冲不足一帧的残余字节。
  bool get hasPending => _pendingLen > 0;

  /// @brief 推入一块任意长度的采集字节，返回已凑满的完整帧。
  ///
  /// 输入立即拷贝进内部缓冲，调用后即可释放 [chunk]。凑满一帧即产出，
  /// 残余留待后续调用或 [flush]。
  ///
  /// @param chunk 采集字节块；空块直接返回空列表。
  /// @return 本次凑满的帧，可能为空。
  List<AudioFrame> push(Uint8List chunk) {
    if (chunk.isEmpty) {
      return const [];
    }
    _baseMs ??= _clock();
    _append(chunk);
    return _emitFull();
  }

  /// @brief 从环形缓冲出口取数切帧（peek 与 advance 同步成对）。
  ///
  /// peek 返回的视图在 [SpscRingBuffer.advance] 前有效，本方法先拷贝
  /// 进内部缓冲、再 advance，满足环缓「同步成对」的生命周期约束；
  /// 跨回绕点的两段视图由本方法无缝拼接。
  ///
  /// @param ring 共享环形缓冲。
  /// @return 本次凑满的帧，可能为空。
  List<AudioFrame> drain(SpscRingBuffer ring) {
    final views = ring.peek();
    if (views.isEmpty) {
      return const [];
    }
    var total = 0;
    for (final view in views) {
      _append(view);
      total += view.length;
    }
    ring.advance(total);
    _baseMs ??= _clock();
    return _emitFull();
  }

  /// @brief 声明上游发生了丢弃，下一产出的帧将携带 gapBefore 标志。
  ///
  /// 切帧器自身从不丢帧；上游（环缓丢旧、采集断流、消费方主动跳过）
  /// 丢弃后调用本方法，把空洞显式传给服务端。
  void markGap() {
    _pendingGap = true;
  }

  /// @brief 流结束收尾：把残余字节以短帧输出并复位帧流。
  ///
  /// @return 残余字节构成的短帧（0 字节残余时为空列表）；之后下一帧流
  ///         重新取基准时刻、seq 继续递增。
  List<AudioFrame> flush() {
    if (_pendingLen == 0) {
      _resetStream();
      return const [];
    }
    final samples = Uint8List.fromList(
      Uint8List.sublistView(_buf, 0, _pendingLen),
    );
    final frame = _makeFrame(samples);
    _pendingLen = 0;
    _resetStream();
    return [frame];
  }

  /// @brief 内部：复位当前帧流的时间基准；seq 跨帧流连续。
  void _resetStream() {
    _baseMs = null;
    _framesSinceBase = 0;
  }

  /// @brief 内部：从缓冲头部连续切出所有完整帧并压实残余。
  List<AudioFrame> _emitFull() {
    final frames = <AudioFrame>[];
    var produced = 0;
    while (_pendingLen - produced >= _frameBytes) {
      final samples = Uint8List.fromList(
        Uint8List.sublistView(_buf, produced, produced + _frameBytes),
      );
      frames.add(_makeFrame(samples));
      produced += _frameBytes;
    }
    if (produced > 0) {
      _buf.setRange(0, _pendingLen - produced, _buf, produced);
      _pendingLen -= produced;
    }
    return frames;
  }

  /// @brief 内部：构造一帧并推进 seq/帧计数，结算 gap 标志。
  AudioFrame _makeFrame(Uint8List samples) {
    final flags = _pendingGap ? AudioFrameFlags.gapBefore : AudioFrameFlags.none;
    _pendingGap = false;
    final frame = AudioFrame(
      seq: _seq++,
      timestampMs:
          (_baseMs ?? _clock()) + _framesSinceBase * frameDurationMs,
      samples: samples,
      flags: flags,
    );
    _framesSinceBase++;
    return frame;
  }

  /// @brief 内部：把数据追加进累积缓冲，容量不足时倍增。
  void _append(Uint8List data) {
    if (_pendingLen + data.length > _buf.length) {
      var capacity = _buf.length * 2;
      while (capacity < _pendingLen + data.length) {
        capacity *= 2;
      }
      final grown = Uint8List(capacity);
      grown.setRange(0, _pendingLen, _buf);
      _buf = grown;
    }
    _buf.setRange(_pendingLen, _pendingLen + data.length, data);
    _pendingLen += data.length;
  }

  static int _wallClock() => DateTime.now().millisecondsSinceEpoch;
}
