/// @file mic_source_factory_io.dart
/// @brief 原生平台的 MicSource 默认工厂（Android Oboe / iOS RemoteIO）。
library;

import 'contracts.dart';
import 'native_mic_source.dart';

/// @brief 创建当前平台的默认采集源（原生 FFI 实现）。
MicSource createDefaultMicSource() => NativeMicSource();
