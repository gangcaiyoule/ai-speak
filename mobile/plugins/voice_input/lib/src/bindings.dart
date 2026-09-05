/// @file bindings.dart
/// @brief 绑定层条件导出门面：按平台选择真实 FFI 绑定或 web 桩。
///
/// - 原生平台（Android/iOS/桌面，dart.library.io）：加载
///   [bindings_native.dart]，经 dart:ffi 对接 vi_*/vo_* C ABI；
/// - web 平台：加载 [bindings_unsupported.dart] 桩——dart:ffi 不存在于
///   web，浏览器侧麦克风/扬声器走 voice_stream 模块的平台实现
///   （WebMicSource 等），不经本绑定层。
library;

export 'bindings_unsupported.dart'
    if (dart.library.io) 'bindings_native.dart';
