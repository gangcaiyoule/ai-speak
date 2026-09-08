/// @file mic_source_factory.dart
/// @brief MicSource 平台工厂的条件导出门面。
///
/// 装配规则：优先原生平台（dart.library.io：Android/iOS/桌面，走
/// Oboe/RemoteIO FFI 实现）；否则 web 平台（dart.library.js_interop，走
/// getUserMedia + AudioWorklet）；都不满足时走兜底桩。UI/控制器只依赖
/// `createDefaultMicSource()` 与 [MicSource] 契约，不感知平台差异。
library;

export 'mic_source_factory_stub.dart'
    if (dart.library.io) 'mic_source_factory_io.dart'
    if (dart.library.js_interop) 'mic_source_factory_web.dart';
