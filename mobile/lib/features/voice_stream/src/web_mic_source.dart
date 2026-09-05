/// @file web_mic_source.dart
/// @brief MicSource 的 web 平台实现：getUserMedia + AudioWorklet。
///
/// Flutter 打包为 web 后，浏览器内没有 dart:ffi 与原生库；操作系统向
/// JS 暴露的麦克风接口是 getUserMedia，音频数据经 AudioWorklet 回调取出。
/// 本实现履行与原生侧（Android Oboe / iOS RemoteIO）完全相同的
/// [MicSource] 契约，逻辑层（[FrameSlicer] 切帧、传输、会话）原样复用。
///
/// 数据面：AudioWorkletProcessor 在浏览器音频线程每 128 样本回调一次，
/// postMessage Float32 块到主线程；主线程转 16 位小端 PCM 后交给切帧器。
/// 回声消除/降噪/AGC 由浏览器内核承担（对应原生侧 VoiceRecognition /
/// VoiceChat 预设的角色）。
///
/// @note 采样率是请求不是保证：AudioContext 以请求采样率创建，实际值回读
///       context.sampleRate 用于切帧时长计算（与接口文档 8.1 同语义）。
/// @note start() 内含用户授权弹窗，为异步完成；失败经帧流错误事件上报。
library;

import 'dart:async';
import 'dart:js_interop';
import 'dart:typed_data';

import 'package:web/web.dart' as web;

import 'contracts.dart';
import 'frame_slicer.dart';
import 'pcm_convert.dart';

/// @brief 基于 getUserMedia + AudioWorklet 的 [MicSource] web 实现。
class WebMicSource implements MicSource {
  /// @brief 构造 web 采集源。
  WebMicSource();

  /// @brief AudioWorklet 处理器源码：把单声道 Float32 块整块转发主线程。
  ///
  /// 以 Blob URL 注入，避免引入需要构建步骤的独立 .js 资源文件。
  static const String _workletCode = '''
class MicTap extends AudioWorkletProcessor {
  process(inputs) {
    const channel = inputs[0] && inputs[0][0];
    if (channel) {
      this.port.postMessage(channel.slice(0));
    }
    return true;
  }
}
registerProcessor('mic_tap', MicTap);
''';

  StreamController<AudioFrame>? _controller;
  web.AudioContext? _context;
  web.AudioWorkletNode? _node;
  web.MediaStreamAudioSourceNode? _source;
  web.MediaStream? _stream;
  FrameSlicer? _slicer;
  String? _workletUrl;

  @override
  Stream<AudioFrame> start(AudioFormat format, int frameDurationMs) {
    if (_controller != null) {
      throw StateError('WebMicSource 已在采集；重复 start 视为错误');
    }
    final controller = StreamController<AudioFrame>(onCancel: stop);
    _controller = controller;
    unawaited(_startAsync(format, frameDurationMs, controller));
    return controller.stream;
  }

  /// @brief 内部：异步启动（授权、建图、挂回调）；失败经流错误上报。
  Future<void> _startAsync(
    AudioFormat format,
    int frameDurationMs,
    StreamController<AudioFrame> controller,
  ) async {
    try {
      // 语音对话约束：请求采样率与单声道；回声消除/降噪/AGC 打开。
      final constraints = ({
        'audio': {
          'sampleRate': {'ideal': format.sampleRateHz},
          'channelCount': {'ideal': format.channelCount},
          'echoCancellation': true,
          'noiseSuppression': true,
          'autoGainControl': true,
        },
      }).jsify() as web.MediaStreamConstraints;
      final stream =
          await web.window.navigator.mediaDevices.getUserMedia(constraints).toDart;
      _stream = stream;

      final context = web.AudioContext(
        web.AudioContextOptions(sampleRate: format.sampleRateHz),
      );
      _context = context;

      final blob = web.Blob(
        <JSAny?>[_workletCode.toJS].toJS,
        web.BlobPropertyBag(type: 'text/javascript'),
      );
      final url = web.URL.createObjectURL(blob);
      _workletUrl = url;
      await context.audioWorklet.addModule(url).toDart;

      final source = context.createMediaStreamSource(stream);
      _source = source;
      final node = web.AudioWorkletNode(context, 'mic_tap');
      _node = node;

      // 实际采样率回读（请求不是保证）；切帧按实际格式计时长。
      _slicer = FrameSlicer(
        format: AudioFormat(
          sampleRateHz: context.sampleRate.round(),
          channelCount: 1,
          bitsPerSample: 16,
        ),
        frameDurationMs: frameDurationMs,
      );
      node.port.addEventListener('message', _onWorkletMessage.toJS);

      // Worklet 节点必须可达 destination 才会被浏览器持续拉取；经零增益
      // 节点接入，既满足拉取条件又避免采集回放环回。
      final mute = context.createGain();
      mute.gain.value = 0;
      source.connect(node);
      node.connect(mute);
      mute.connect(context.destination);
    } catch (error, stackTrace) {
      controller.addError(error, stackTrace);
      await stop();
    }
  }

  /// @brief 内部：Worklet 消息 → Float32 → 16 位 PCM → 切帧 → 发流。
  void _onWorkletMessage(web.Event event) {
    final controller = _controller;
    final slicer = _slicer;
    if (controller == null || slicer == null || controller.isClosed) {
      return;
    }
    final data = (event as web.MessageEvent).data;
    final chunk = data.unsafeCast<JSFloat32Array>().toDart;
    final bytes = pcm16FromFloat32(chunk);
    for (final frame in slicer.push(bytes)) {
      controller.add(frame);
    }
  }

  @override
  Future<void> stop() async {
    final controller = _controller;
    _controller = null;
    _slicer = null;
    try {
      _node?.disconnect();
      _source?.disconnect();
    } catch (_) {}
    _node = null;
    _source = null;
    // 停掉全部轨道，释放浏览器麦克风指示灯与设备占用。
    _stream?.getTracks().toDart.forEach((track) => track.stop());
    _stream = null;
    final context = _context;
    _context = null;
    try {
      await context?.close().toDart;
    } catch (_) {}
    final url = _workletUrl;
    _workletUrl = null;
    if (url != null) {
      web.URL.revokeObjectURL(url);
    }
    // 关闭帧流：有监听者则收到 done；无监听时 close 的 done 无从派发，
    // 不能阻塞 stop（与 NativeMicSource 同语义）。
    unawaited(controller?.close());
  }
}
