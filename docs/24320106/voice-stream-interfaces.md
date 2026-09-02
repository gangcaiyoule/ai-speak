# voice_stream 接口文档

- 模块路径：`mobile/lib/features/voice_stream/`
- 状态：v0.1（R1 阶段，接口与 Dart 参考实现已定，平台实现未接入）
- 范围：读（采集）、传（上行/回包）、放（播放）三条数据通路的抽象；
  UI（字幕/波形）不在本文档范围内，只通过事件流消费数据。

## 1. 分层总览

```
UI / 控制器（模块外）
        │  仅依赖本契约
MicSource ─→ AudioFrame ─→ AudioTransport   （上行）
AudioTransport.events ─→ TransportEvent     （回包）
AudioSink ←── AudioFrame                    （播放）
        │
平台实现：Oboe（Android）/ RemoteIO（iOS）/ WSS·WebRTC（传输）
```

代码对应关系：

| 契约 | 文件 |
|---|---|
| 核心接口与值对象 | `src/contracts.dart` |
| 共享环形缓冲 | `src/ring_buffer.dart` |

## 2. 值对象

### 2.1 AudioFormat

描述一路 PCM 流的格式。

| 字段 | 类型 | 约定 |
|---|---|---|
| `sampleRateHz` | `int` | 采样率 Hz |
| `channelCount` | `int` | 实时链路固定 1（单声道） |
| `bitsPerSample` | `int` | 实时链路固定 16 |

方法：

| 方法 | 语义 |
|---|---|
| `bytesPerFrame(int durationMs)` | 返回 `durationMs` 毫秒音频的字节数，向下取整 |

### 2.2 AudioFrame

一帧定长 PCM 音频，是贯穿采集、传输、播放的统一数据单元。

| 字段 | 类型 | 约定 |
|---|---|---|
| `seq` | `int` | 单调递增帧序号；**跳号即表示上游丢帧**，接收方据此估算丢包率 |
| `timestampMs` | `int` | 采集时刻毫秒时间戳，用于端到端延迟测量 |
| `samples` | `Uint8List` | PCM 样本，16 位小端 |

### 2.3 TransportEvent（sealed）

服务端回包事件基类，目前两个子类型：

| 子类型 | 字段 | 语义 |
|---|---|---|
| `TransportMessage` | `payload: String` | 识别/评测结果，JSON 文本，结构由服务端协议定义 |
| `TransportStats` | `sentFrames` / `droppedFrames` / `bufferedBytes` | 传输层量化统计，弱网调参的观测出口 |

## 3. MicSource（采集端契约）

把麦克风抽象为可随时中断的帧流。

| 方法 | 签名 | 语义 |
|---|---|---|
| `start` | `Stream<AudioFrame> start(AudioFormat format, int frameDurationMs)` | 开始采集并返回帧流；重复调用视为错误 |
| `stop` | `Future<void> stop()` | 停止采集并释放设备；幂等 |

实现必须满足的硬约束：

- 采集回调线程不阻塞 UI；
- 弱网/背压下按丢帧策略前行（丢帧通过 `AudioFrame.seq` 跳号体现）；
- `stop()` 后帧流关闭，可再次 `start()`（重连语义）。

## 4. AudioSink（播放端契约）

把扬声器抽象为帧写入点。

| 成员 | 签名 | 语义 |
|---|---|---|
| `write` | `bool write(AudioFrame frame)` | 写入待播帧；缓冲满则拒绝并返回 `false`，**永不阻塞** |
| `underrunBytes` | `int`（getter） | 播放缓冲欠载字节数，供欠载策略调参 |
| `stop` | `Future<void> stop()` | 关闭播放并释放设备；幂等 |

## 5. AudioTransport（上行传输契约）

音频帧外发 + 服务端事件回流；传输栈（WSS / WebRTC）是接口后的可替换件。

| 成员 | 签名 | 语义 |
|---|---|---|
| `sendFrame` | `void sendFrame(AudioFrame frame)` | 非阻塞发送；丢弃行为通过 `events` 中的 `TransportStats` 体现 |
| `events` | `Stream<TransportEvent>` | 服务端事件流：识别结果、评测结果、统计 |
| `close` | `Future<void> close()` | 关闭连接；幂等 |

传输选型结论（详见模块 `README.md` 第 4 节）：先 WSS 对接云厂商实时语音
API；自托管网关时上 WebRTC（`flutter_webrtc` + Pion/LiveKit）；自管
UDP+KCP 为后备。会话延续（上下文、VAD、打断）属于协议/会话层，不压在本
接口上。

## 6. SpscRingBuffer 行为契约

单生产者-单消费者定容字节环形缓冲（`src/ring_buffer.dart`），是 C 层共享
缓冲（NDK/Xcode 编译同一份语义）的 Dart 参考实现。以下行为约束对两端实现
同等有效：

1. **生产者永不阻塞**：`write(data)` 空间不足时覆盖最旧数据，丢弃量累计到
   `droppedBytes`；单次写入超过容量时只保留末尾 `capacityBytes` 字节。
2. **零拷贝读**：`peek([maxBytes])` 返回内部存储的只读视图（跨回绕点时为
   两段）；`advance(bytes)` 之后该区间才可被生产者覆盖。
3. **peek/advance 必须同步成对调用**：不得异步持有 peek 返回的视图。这是
   「丢旧」与「零拷贝」兼得的生命周期约束；热点路径不适用时改用
   `readInto(dst)`（一次拷贝）。
4. **无锁**：C 层读写指针使用 acquire/release 原子序，单写单读下无锁；
   Dart 参考实现依赖单 isolate 顺序执行保证。
5. **容量按毫秒预算**：构造时传字节数，例如 16kHz × 16bit × mono ×
   1000ms = 32KiB。

| 成员 | 语义 |
|---|---|
| `write(data) → int` | 写入后缓冲中的可读字节数 |
| `peek([maxBytes]) → List<Uint8List>` | 只读视图，1 或 2 段 |
| `advance(bytes)` | 消费最近一次 peek 取出的字节数，越界抛 `ArgumentError` |
| `readInto(dst) → int` | 拷贝读取，返回实际读取字节数 |
| `capacityBytes` / `lengthBytes` / `droppedBytes` / `isEmpty` | 状态查询 |

单测基线：`mobile/test/voice_stream/ring_buffer_test.dart`（九个用例：
往返、回绕两段视图、覆盖丢旧计数、超容量保尾、分批消费、截断读、空缓冲、
越界、非法容量）。C 实现落地后用同一组用例在 NDK/Xcode 下回归。

## 7. 帧头格式（传输层预留）

切帧器（R3）按时间窗（默认 20ms @ 16kHz/16bit/mono = 640B）切出定长帧，
外发时加 12 字节帧头：

```
offset  size  字段
0       4     uint32 seq          帧序号
4       4     uint32 timestamp_ms 采集时间戳
8       2     uint16 size         负载长度
10      2     uint16 flags        标志位（丢帧空洞等，待定）
```

小端序。`flags` 具体位定义在切帧器实现时补充。

## 8. 平台符合性审计

抽象数据流逐条对照官方文档核验，证据来源：
[AAudio 指南](https://developer.android.google.cn/ndk/guides/audio/aaudio/aaudio)、
[Oboe FullGuide](https://github.com/google/oboe/blob/main/docs/FullGuide.md)、
[Apple Audio Unit Hosting Guide](https://developer.apple.com/library/archive/documentation/MusicAudio/Conceptual/AudioUnitHostingGuide_iOS/ConstructingAudioUnitApps/ConstructingAudioUnitApps.html)。

| 抽象约定 | Android 依据（AAudio/Oboe） | iOS 依据（RemoteIO/AVAudioSession） |
|---|---|---|
| MicSource 输出帧流、回调不碰 UI 线程 | data callback 运行在高优先级线程，回调内禁止 read/write 同一流 | render callback 运行在实时线程，pull 模型取数 |
| 生产者永不阻塞（环缓丢旧） | 回调 Do's/Don'ts 明令禁止 malloc/new、mutex、文件/网络、sleep、stop/close | 同类实时线程纪律；aurioTouch 示例即用环形缓冲 |
| AudioFrame 的 16bit PCM | `PCM_I16`（Q0.15）为标准格式 | RemoteIO 支持 lpcm + 有符号 16 位 |
| 弱网/背压丢帧语义 | 流可随时断连（拔耳机等），须在错误回调中换线程 stop/close 后重开 | 会话中断/路由变化由 AVAudioSession 通知 |
| stop()/start() 幂等与重连 | errorCallback → onErrorAfterClose 后可重开流 | session setActive/deActive 周期 |
| AudioSink.underrunBytes | `getXRunCount()` 官方欠载计数 | I/O buffer duration 可调（默认约 23ms，可请求约 5ms） |

### 8.1 修正：采样率/格式是请求不是保证

三方文档一致确认，`start(format)` 传入的格式只能作为**请求**：

- Apple 原文：系统"may or may not be able to grant the request"，
  激活后必须读回 `currentHardwareSampleRate`；
- Oboe 原文：打开流后"it is wise to verify the input format and be
  prepared to convert data if necessary"，`sampleRate` 等属性应查询获取；
- Oboe 提供 `setSampleRateConversionQuality()` 可让库代做重采样（默认 Medium）。

因此 `MicSource.start` 契约措辞为：实现按**协商后的实际格式**交付帧；
实现层要么用 Oboe 内建重采样，要么在管线内做 SRC。R4/R5 落地时必须
在打开流后回读实际格式并记录。

### 8.2 时间戳来源与已知坑

- AAudio：`AAudioStream_getTimestamp(CLOCK_MONOTONIC)`，官方唯一列入线程
  安全例外的 get 类函数，须在回调外调用；
- Oboe：`getTimestamp()` 在 OpenSL ES 路径返回 `ErrorUnimplemented`
  （Known Issues 原文）——回退到 OpenSL ES 的机型需用本地单调时钟替代，
  `AudioFrame.timestampMs` 语义不变；
- iOS：render callback 的 `AudioTimeStamp`（`mHostTime`/`mSampleTime`）。

### 8.3 其他实锤

- AAudio 建议跨线程传递指令用「原子队列」——与环缓的无锁设计同向；
- Oboe `InputPreset` 默认 `VoiceRecognition`，官方注明为低延迟优化，
  正是语音对话场景应保持的预设；
- 撤回记录：本节初稿曾引用 Oboe 内置 `FifoBuffer` 作为环缓先例，经核实
  当前 main 分支不存在该文件，证据作废。环缓为自研设计，依据是上文
  官方回调约束本身，而非官方组件。
