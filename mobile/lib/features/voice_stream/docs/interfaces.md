# voice_stream 接口文档

- 模块路径：`mobile/lib/features/voice_stream/`
- 文档位置：`mobile/lib/features/voice_stream/docs/interfaces.md`（2026-09-03 由
  `docs/24320106/voice-stream-interfaces.md` 迁入，接口规格随模块一起维护）
- 状态：v0.4（R7：AudioSink 播放路径、欠载状态机与丢帧统计已落地；平台装配：原生 ↔ web 条件导入）
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
| 切帧器与帧头编解码 | `src/frame_slicer.dart` |
| 会话层契约与生命周期状态机 | `src/session.dart` |
| 平台工厂（原生 ↔ web 条件装配） | `src/mic_source_factory.dart` |

### 2.x 平台实现与 web 装配（v0.4 新增）

`MicSource` 的平台实现经条件导出装配，UI/控制器只依赖工厂：

- **原生**（`dart.library.io`）：`src/native_mic_source.dart`——Oboe/
  RemoteIO 经 dart:ffi；插件绑定层 `plugins/voice_input/lib/src/bindings.dart`
  同样按条件导出（原生 FFI ↔ web 桩，web 上 dart:ffi 不存在）。
- **web**（`dart.library.js_interop`）：`src/web_mic_source.dart`——
  `getUserMedia`（约束：请求采样率、单声道、回声消除/降噪/AGC）+
  AudioWorklet（每 128 样本回调，Blob URL 注入处理器）；Float32 归一化
  样本经 `src/pcm_convert.dart` 转 16 位小端 PCM 后接切帧器。采样率同样
  是请求不是保证：回读 `AudioContext.sampleRate` 用于切帧。
- 误用兜底：web 上调用插件绑定会抛 `UnsupportedError`（明确报错而非
  编译失败）；工厂兜底桩同理。
| 采集端平台实现 | `src/native_mic_source.dart` |
| 播放端平台实现与欠载状态机 | `src/native_audio_sink.dart` / `native/playback_queue.h` |

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
| `flags` | `int` | 标志位，见 [AudioFrameFlags]；缺省 0，既有构造不受影响 |

`AudioFrameFlags` 位定义：`none = 0x0000`；`gapBefore = 0x0001`（本帧之前
存在丢帧空洞，发送方主动标记，与 seq 跳号互为补充）。

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

### 4.1 NativeAudioSink 与 playback_queue 状态机（v0.4 新增，R7）

`src/native_audio_sink.dart` 实现了 `AudioSink` 契约，经 `package:voice_input` 播放绑定（C ABI `vo_*` 一族，`native/voice_output.h`）操作原生播放队列：

- **生产者（Dart 线程）**：通过 `write(frame)` 全有或全无写入原生队列；缓冲满时整块拒绝返回 `false`，永不阻塞；拒绝量累计至 `droppedBytes`。
- **消费者（音频回调线程）**：Oboe（Android）或 RemoteIO（iOS）渲染回调经 `vsc_pbq_acquire` 零拷贝取数填入音频缓冲，取数后 `vsc_pbq_commit` 推进读指针；缺口部分补零静音。
- **预缓冲（priming）与迟滞（hysteresis）**：启动后攒够 `primingMs`（默认 40ms）才开始出数；取空后回退到预缓冲状态，避免边读边写反复欠载。
- **欠载统计**：首次完成预缓冲后，回调每次输出静音的字节数累计到 `underrunBytes`（启动预缓冲期的初始静音不计入）。
- **惰性启动与重连**：首次 `write` 时惰性调用 `start`；`stop()` 释放后再次 `write` 会重新打开设备。

单测基线：
- C 原生状态机：`mobile/lib/features/voice_stream/native/test_playback_queue.c`（7 个用例：参数校验、全有或全无、预缓冲阈值、欠载迟滞、部分取数缺口、跨回绕双视图、零阈值直通）。
- Dart 实现：`mobile/test/voice_stream/native_audio_sink_test.dart`（6 个用例：惰性启动与写入、空帧直通、满载拒绝、统计量透传、stop 幂等与重连、启动异常回滚）。

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

### 5.1 VoiceSession 会话层契约（v0.2 新增）

`src/session.dart` 把上游 XE3-ESL `speakup.voice-input.v1` 已验证的会话
语义抽象化后叠加在 `AudioTransport` 之上；`AudioTransport` 契约保持不变。

**生命周期与状态机**（`VoiceSessionPhase` + `VoiceSessionLifecycle`）：

```
idle ──begin()──→ active ──requestFinish()──→ finishing ──complete()──→ closed
                     │                            │
                     └──────────abort()───────────┘   （失败/取消；对 closed 幂等）
```

- `sendFrame` 仅在 `active` 允许；`finishing` 起不再收帧。
- `complete()` 只能从 `finishing` 进入；非法迁移抛 `StateError`。

**配置**（`VoiceSessionConfig`）：

| 字段 | 约定 |
|---|---|
| `idempotencyKey` | 长度 8–128、首尾无空白；同一逻辑会话跨重连重试不变，服务端据此去重 |
| `format` | 上行音频格式（请求值非保证值，见 8.1 节） |

**事件**（`VoiceSessionEvent` sealed）：

| 子类型 | 字段 | 语义 |
|---|---|---|
| `SessionStarted` | — | 会话建立，可上行音频 |
| `SessionPartial` | `payload: String` | 中间识别/评测结果，JSON 文本 |
| `SessionFinal` | `payload: String` | 最终结果（终态之一） |
| `SessionFailed` | `kind: String` / `retryable: bool` | 失败终态；`retryable=true` 时可用同一幂等键重开 |
| `SessionStats` | `sentFrames` / `droppedFrames` / `bufferedBytes` | 会话层量化统计 |

**`VoiceSession` 接口方法**：

| 方法 | 语义 |
|---|---|
| `sendFrame(AudioFrame)` | 非阻塞上行一帧；仅 active 阶段，其余抛 `StateError` |
| `finish() → Future` | 优雅结束：等服务端终态事件后完成 |
| `cancel() → Future` | 立即中断；对已关闭会话幂等成功 |
| `events → Stream` | 上述事件流 |
| `config` / `phase` | 配置与当前阶段 |

传输实现职责（R6）：把 `begin/finish/cancel` 映射为具体控制帧（如 WSS 的
`start/finish/cancel`），把服务端事件映射为 `SessionStarted/Partial/
Final/Failed`；断连一律映射为 `SessionFailed`，不静默吞掉。

单测基线：`mobile/test/voice_stream/session_test.dart`（12 个用例：
幂等键三项静态校验、正常生命周期、finishing 禁发帧、active 直接取消、
finishing 失败、abort 幂等、三类非法迁移）。

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

## 7. 帧头格式（R3 已实现）

切帧器按时间窗（默认 20ms @ 16kHz/16bit/mono = 640B）切出定长帧，
外发时加 12 字节帧头：

```
offset  size  字段
0       4     uint32 seq          帧序号
4       4     uint32 timestamp_ms 采集时间戳
8       2     uint16 size         负载长度
10      2     uint16 flags        标志位
```

小端序。`flags` 位定义：`gapBefore = 0x0001`——本帧之前存在丢帧空洞，
由发送方（切帧器，经上游 `markGap()` 告知）主动标记；`0x0000` 为无标志。

实现位于 `src/frame_slicer.dart`：

| 组件 | 语义 |
|---|---|
| `FrameSlicer.push(chunk) → List<AudioFrame>` | 任意长度字节块切帧；残余缓冲，凑满即出帧 |
| `FrameSlicer.drain(ring) → List<AudioFrame>` | 接环形缓冲出口；peek/advance 同步成对，跨回绕两段视图无缝拼接 |
| `FrameSlicer.flush() → List<AudioFrame>` | 流结束收尾，残余以短帧输出；seq 跨帧流连续，timestamp 基准重新起算 |
| `FrameSlicer.markGap()` | 下一产出的帧携带 `gapBefore` 标志（只标一帧） |
| `FrameHeaderCodec.encode/decode/decodeHeader` | 帧头小端序编解码；传输字节 = 帧头 12B + 负载 |

时间戳语义：每帧 `timestampMs` = 帧流首字节到达时刻（`clock` 可注入，
缺省本地墙钟）+ 帧序 × 帧时长；输出的 `samples` 一律为拷贝，不与内部
缓冲共享内存。

单测基线：`mobile/test/voice_stream/frame_slicer_test.dart`（11 个用例：
flags 缺省兼容、非整块切帧、空输入、flush 短帧与基准复位、gap 标记只落
一帧、drain 跨回绕拼接、drain 空缓冲、编解码往返、小端序布局、长度异常、
非法构造参数）。

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
