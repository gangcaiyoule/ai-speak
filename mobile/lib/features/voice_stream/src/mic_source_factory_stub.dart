/// @file mic_source_factory_stub.dart
/// @brief 平台工厂兜底桩：既非原生也非 web 的平台不可采集。
library;

import 'contracts.dart';

/// @brief 不可用平台上的默认采集源：调用即抛错。
MicSource createDefaultMicSource() =>
    throw UnsupportedError('当前平台不支持音频采集（既非原生也非 web）。');
