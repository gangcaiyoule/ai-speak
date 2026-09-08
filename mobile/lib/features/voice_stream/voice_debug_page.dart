/// @file voice_debug_page.dart
/// @brief 采集链路调试页：App 装配层的最小演示入口（R8 装配前的观察点）。
///
/// 经 [createDefaultMicSource] 取当前平台的 [MicSource]：原生平台走
/// Oboe/RemoteIO（FFI），web 平台走 getUserMedia + AudioWorklet（浏览器
/// 授权后调用宿主系统麦克风，例如 Windows 上的 Chrome/Edge）。本页只做
/// 启动/停止/计数，读出的帧即弃，不做传输与语义处理。
///
/// 典型验证路径：Flutter 打包为 web 后，在浏览器打开本页并授权麦克风，
/// 观察帧计数增长与错误提示，即可确认 web 采集链路工作正常。
library;

import 'dart:async';

import 'package:flutter/material.dart';

import 'src/contracts.dart';
import 'src/mic_source_factory.dart';

/// @brief 麦克风采集调试页。
class VoiceDebugPage extends StatefulWidget {
  /// @brief 构造调试页。
  const VoiceDebugPage({super.key});

  @override
  State<VoiceDebugPage> createState() => _VoiceDebugPageState();
}

class _VoiceDebugPageState extends State<VoiceDebugPage> {
  final MicSource _source = createDefaultMicSource();
  StreamSubscription<AudioFrame>? _subscription;
  bool _running = false;
  int _frameCount = 0;
  int _gapCount = 0;
  String? _error;

  @override
  void dispose() {
    unawaited(_subscription?.cancel());
    unawaited(_source.stop());
    super.dispose();
  }

  Future<void> _start() async {
    if (_running) {
      return;
    }
    setState(() {
      _error = null;
      _frameCount = 0;
      _gapCount = 0;
    });
    try {
      final frames = _source.start(
        const AudioFormat(
          sampleRateHz: 16000,
          channelCount: 1,
          bitsPerSample: 16,
        ),
        20,
      );
      _subscription = frames.listen(
        (frame) => setState(() {
          _running = true;
          _frameCount++;
          if (frame.flags & AudioFrameFlags.gapBefore != 0) {
            _gapCount++;
          }
        }),
        onError: (Object error) => setState(() {
          _error = '$error';
          _running = false;
        }),
        onDone: () => setState(() => _running = false),
      );
      setState(() => _running = true);
    } catch (error) {
      setState(() {
        _error = '$error';
        _running = false;
      });
    }
  }

  Future<void> _stop() async {
    await _subscription?.cancel();
    _subscription = null;
    await _source.stop();
    if (mounted) {
      setState(() => _running = false);
    }
  }

  @override
  Widget build(BuildContext context) => Scaffold(
        appBar: AppBar(title: const Text('语音采集调试')),
        body: Center(
          child: Column(
            mainAxisAlignment: MainAxisAlignment.center,
            children: [
              Text(_running ? '采集中…' : '已停止'),
              const SizedBox(height: 8),
              Text('帧数：$_frameCount（含空洞标记 $_gapCount）'),
              if (_error != null) ...[
                const SizedBox(height: 8),
                Text(
                  '错误：$_error',
                  style: TextStyle(color: Theme.of(context).colorScheme.error),
                ),
              ],
              const SizedBox(height: 24),
              FilledButton(
                onPressed: _running ? null : _start,
                child: const Text('开始采集'),
              ),
              const SizedBox(height: 8),
              OutlinedButton(
                onPressed: _running ? _stop : null,
                child: const Text('停止'),
              ),
            ],
          ),
        ),
      );
}
