/// @file mic_source_factory_web.dart
/// @brief web 平台的 MicSource 默认工厂（getUserMedia + AudioWorklet）。
library;

import 'contracts.dart';
import 'web_mic_source.dart';

/// @brief 创建当前平台的默认采集源（web 实现，浏览器授权后出帧）。
MicSource createDefaultMicSource() => WebMicSource();
