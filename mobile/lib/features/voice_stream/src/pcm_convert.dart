/// @file pcm_convert.dart
/// @brief PCM 格式转换纯函数（web 采集链路用，可单测）。
library;

import 'dart:typed_data';

/// @brief Float32 归一化样本转 16 位小端 PCM 字节。
///
/// 浏览器 AudioWorklet 交付的是 [-1, 1] 归一化 Float32 样本；本函数做
/// 限幅（±1 外截断）与量化，与原生侧 I16 PCM（Q0.15）对齐。
///
/// @param samples 归一化 Float32 样本。
/// @return 16 位小端 PCM 字节（每样本 2 字节）。
Uint8List pcm16FromFloat32(Float32List samples) {
  final out = Uint8List(samples.length * 2);
  final view = ByteData.sublistView(out);
  for (var i = 0; i < samples.length; i++) {
    final s = samples[i];
    final v = s <= -1.0
        ? -32768
        : s >= 1.0
            ? 32767
            : (s * 32767).round();
    view.setInt16(i * 2, v, Endian.little);
  }
  return out;
}
