/// @file 单生产者-单消费者（SPSC）字节环形缓冲。
///
/// @brief 定容环形缓冲，支持覆盖最旧与零拷贝取视图。
///
/// 本实现是 C 层共享缓冲（NDK / Xcode 编译同一份语义）的 Dart 参考实现，
/// 语义约束逐条对齐：
///
/// - 生产者永不阻塞：@c write 遇到容量不足时覆盖最旧数据，丢弃量计入
///   droppedBytes。
/// - 消费者零拷贝：@c peek 返回内部存储的只读视图，@c advance 之后该区间
///   才可被生产者覆盖。peek 与 advance 必须同步成对调用，不得异步持有视图。
/// - C 层落地时读写指针使用 acquire/release 原子序，单写单读下无锁。
library;

import 'dart:typed_data';

/// @brief 单生产者-单消费者定容字节环形缓冲。
class SpscRingBuffer {
  /// @brief 构造环形缓冲。
  ///
  /// @param capacityBytes 容量，单位字节，必须为正数；按毫秒预算换算，
  /// 例如 16kHz × 16bit × 单声道 × 1000ms = 32KiB。
  SpscRingBuffer(int capacityBytes) : _storage = Uint8List(capacityBytes) {
    if (capacityBytes <= 0) {
      throw ArgumentError.value(capacityBytes, 'capacityBytes', '必须为正数');
    }
  }

  final Uint8List _storage;

  int _read = 0; ///< 下一个待读字节的下标。
  int _write = 0; ///< 下一个待写字节的下标。
  int _size = 0; ///< 当前可读字节数。
  int _dropped = 0; ///< 因覆盖被丢弃的历史字节总数。

  /// @brief 缓冲容量，单位字节。
  int get capacityBytes => _storage.length;

  /// @brief 当前可读字节数。
  int get lengthBytes => _size;

  /// @brief 因覆盖而丢弃的历史字节总数。
  int get droppedBytes => _dropped;

  /// @brief 是否没有可读数据。
  bool get isEmpty => _size == 0;

  /// @brief 写入数据；空间不足时覆盖最旧数据，永不阻塞、永不失败。
  ///
  /// 若 [data] 长度超过容量，只保留其末尾 capacityBytes 字节，其余全部
  /// 计入丢弃。
  ///
  /// @param data 待写入字节。
  /// @return 写入后缓冲中的可读字节数。
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

  /// @brief 零拷贝取出可读数据的只读视图。
  ///
  /// 数据跨越回绕点时返回两段视图，否则返回一段。视图与内部存储共享内存，
  /// 调用 @c advance 前有效；peek 与 advance 之间不得插入其他操作。
  ///
  /// @param maxBytes 最多取出的字节数；缺省取全部可读数据。
  /// @return 只读视图列表，1 或 2 段；无数据时为空列表。
  List<Uint8List> peek([int? maxBytes]) {
    final want = maxBytes ?? _size;
    if (want <= 0 || _size == 0) {
      return const [];
    }
    final take = want > _size ? _size : want;
    final first = _storage.length - _read;
    if (take <= first) {
      return [_view(_read, take)];
    }
    return [
      _view(_read, first),
      _view(0, take - first),
    ];
  }

  /// @brief 消费最近一次 peek 取出的字节。
  ///
  /// @param bytes 待消费字节数，不得超出可读范围，否则抛 ArgumentError。
  void advance(int bytes) {
    if (bytes < 0 || bytes > _size) {
      throw ArgumentError.value(bytes, 'bytes', '超出可读范围');
    }
    _read = (_read + bytes) % _storage.length;
    _size -= bytes;
  }

  /// @brief 一次性拷贝读取。
  ///
  /// 供不想管理视图生命周期的调用方使用；热点路径请用 peek/advance。
  ///
  /// @param destination 目标缓冲。
  /// @return 实际读取字节数。
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

  /// @brief 内部：丢弃最旧数据并累计丢弃计数。
  ///
  /// @param bytes 待丢弃字节数，调用方保证不超出可读范围。
  void _discard(int bytes) {
    _read = (_read + bytes) % _storage.length;
    _size -= bytes;
    _dropped += bytes;
  }

  /// @brief 内部：把数据分段写入存储并推进写指针。
  ///
  /// @param data 已钳制到容量范围内的待写入数据。
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

  /// @brief 内部：在 [start] 处取 [length] 长度的只读视图。
  Uint8List _view(int start, int length) =>
      Uint8List.sublistView(_storage, start, start + length);
}
