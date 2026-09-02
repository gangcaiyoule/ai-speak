/// 单生产者-单消费者（SPSC）字节环形缓冲。
///
/// 本实现是 C 层共享缓冲（NDK / Xcode 编译同一份语义）的 Dart 参考实现，
/// 语义约束逐条对齐：
///
/// - 生产者永不阻塞：`write` 遇到容量不足时覆盖最旧数据，丢弃量计入
///   [droppedBytes]。
/// - 消费者零拷贝：`peek` 返回内部存储的只读视图，`advance` 之后该区间才可
///   被生产者覆盖。peek 与 advance 必须同步成对调用，不得异步持有视图。
/// - C 层落地时读写指针使用 acquire/release 原子序，单写单读下无锁。
library;

import 'dart:typed_data';

/// 定容环形缓冲，支持覆盖最旧与零拷贝取视图。
class SpscRingBuffer {
  /// 创建容量为 [capacityBytes] 字节的环形缓冲。
  SpscRingBuffer(int capacityBytes) : _storage = Uint8List(capacityBytes) {
    if (capacityBytes <= 0) {
      throw ArgumentError.value(capacityBytes, 'capacityBytes', '必须为正数');
    }
  }

  final Uint8List _storage;

  int _read = 0; // 下一个待读字节的下标。
  int _write = 0; // 下一个待写字节的下标。
  int _size = 0; // 当前可读字节数。
  int _dropped = 0; // 因覆盖被丢弃的历史字节总数。

  /// 缓冲容量，单位字节。
  int get capacityBytes => _storage.length;

  /// 当前可读字节数。
  int get lengthBytes => _size;

  /// 因覆盖而丢弃的历史字节总数。
  int get droppedBytes => _dropped;

  /// 是否没有可读数据。
  bool get isEmpty => _size == 0;

  /// 写入 [data]；空间不足时覆盖最旧数据，永不阻塞、永不失败。
  ///
  /// 返回写入后缓冲中的可读字节数。若 [data] 长度超过容量，只保留其末尾
  /// [capacityBytes] 字节，其余全部计入丢弃。
  int write(Uint8List data) {
    if (data.length > _storage.length) {
      _dropped += data.length - _storage.length;
      data = Uint8List.sublistView(
        data,
        data.length - _storage.length,
      );
    }
    var free = _storage.length - _size;
    if (data.length > free) {
      final overflow = data.length - free;
      _discard(overflow);
    }
    _writeAll(data);
    return _size;
  }

  /// 零拷贝取出最多 [maxBytes] 可读数据的只读视图。
  ///
  /// 数据跨越回绕点时返回两段视图，否则返回一段。视图与内部存储共享内存，
  /// 调用 [advance] 前有效； peek 与 advance 之间不得插入其他操作。
  List<Uint8List> peek([int? maxBytes]) {
    final want = maxBytes ?? _size;
    if (want <= 0 || _size == 0) {
      return const [];
    }
    final take = want > _size ? _size : want;
    final first = _storage.length - _read;
    if (take <= first) {
      return [_copyView(_read, take)];
    }
    return [
      _copyView(_read, first),
      _copyView(0, take - first),
    ];
  }

  /// 消费最近一次 [peek] 取出的 [bytes] 字节。
  void advance(int bytes) {
    if (bytes < 0 || bytes > _size) {
      throw ArgumentError.value(bytes, 'bytes', '超出可读范围');
    }
    _read = (_read + bytes) % _storage.length;
    _size -= bytes;
  }

  /// 一次性拷贝读取到 [destination]，返回实际读取字节数。
  ///
  /// 供不想管理视图生命周期的调用方使用；热点路径请用 peek/advance。
  int readInto(Uint8List destination) {
    final views = peek(destination.length);
    var offset = 0;
    for (final view in views) {
      destination.setAll(offset, view);
      offset += view.length;
    }
    advance(offset);
    return offset;
  }

  void _discard(int bytes) {
    _read = (_read + bytes) % _storage.length;
    _size -= bytes;
    _dropped += bytes;
  }

  void _writeAll(Uint8List data) {
    var offset = 0;
    while (offset < data.length) {
      final index = (_write + offset) % _storage.length;
      final contiguous = data.length - offset < _storage.length - index
          ? data.length - offset
          : _storage.length - index;
      _storage.setRange(index, index + contiguous, data, offset);
      offset += contiguous;
    }
    _write = (_write + data.length) % _storage.length;
    _size += data.length;
    if (_size > _storage.length) {
      // 理论上不可达；防御性钳制，避免指针错位。
      _size = _storage.length;
    }
  }

  Uint8List _copyView(int start, int length) =>
      Uint8List.sublistView(_storage, start, start + length);
}
