# voice_stream：实时语音链路

麦克风采集 → 切帧 → 上行流 → 服务端识别/评测 → 流式回包 → 低延迟播放。
本模块只关心**读（采集）、传（上行/回包）、放（播放）**三条数据通路和它们的抽象，
UI（字幕、波形）不在本目录范围内，只预留数据出口。

设计原则：接口先行、纯逻辑可单测、平台实现可替换、采集线程永不阻塞。

## 1. 语言与平台决策

| 层 | 语言 | 理由 |
|---|---|---|
| 采集/播放/缓冲 | C（Android 侧 C++，因 Oboe） | 一份实现两端编：NDK 编进 Android，Xcode 直接编进 iOS |
| iOS 会话激活 | Objective-C++ 壳（约 20 行） | `AVAudioSession` 激活是 iOS 规定动作，AudioUnit 本身是 C API |
| Dart 侧 | `dart:ffi` 控制面 + 数据出口 | 只拿指针、调开关，不做数据搬运 |
| 服务端 | Go（仓库既有） | 复用现有骨架 |

明确不写：Kotlin/Swift 业务逻辑。Android 采集走 Oboe（AAudio 路径），
不走 Java `AudioRecord`。

## 2. 共享环形缓冲（核心抽象，先行实现）

两端平台差异（Oboe 回调 vs RemoteIO 回调）全部消化在同一份 C 缓冲实现里：

- **SPSC**：单生产者（音频回调线程）单消费者（FFI 侧读取），无锁，C 层用
  acquire/release 原子序。
- **生产者永不阻塞**：容量不足时覆盖最旧数据（丢旧）并累计
  `dropped_bytes`，这是弱网/背压的统一出口，上层只看计数。
- **零拷贝读**：`peek()` 返回内部存储的只读视图，`advance(bytes)` 后该区间
  才可被覆盖。约束：**peek 与 advance 必须同步成对调用**，不得异步持有视图
  （丢旧会推进读指针，覆盖未提交区间）。10–20ms 一帧几百字节，一次 peek 内
  完成转发是便宜操作。
- **容量按毫秒预算**：如 16kHz × 16bit × 单声道 × 1000ms = 32KiB，构造时传入。

Dart 参考实现 `src/ring_buffer.dart` 与 C 实现语义逐条对齐，并以此写单元测试
（往返、回绕、溢出丢旧、丢帧计数、peek/advance 配对）。C 实现落地后用同一组
用例在 NDK/Xcode 下回归。

## 3. 切帧

采集吐的是任意长度的字节块，按时间窗（默认 20ms @ 16kHz/16bit/mono = 640B）
切成定长帧，帧头 12 字节：

```
uint32 seq; uint32 timestamp_ms; uint16 size; uint16 flags;
```

`flags` 标记丢帧空洞（上游 drop 后 seq 跳号，服务端据此估算丢包率）。
切帧器是纯函数式组件，可单测。

## 4. 传输

定位是实时 AI 对话 App，上行要对接云服务（ASR/LLM/TTS），所以不自己造
传输层，用成熟的弱网语音流网络栈：

| 方案 | 组成 | 适用 |
|---|---|---|
| WSS 云服务协议 | WebSocket/TLS 对接云厂商实时语音 API | 直接用云 ASR/LLM/TTS 时集成最快；弱网靠端上环形缓冲丢帧兜底 |
| WebRTC | `flutter_webrtc`（libwebrtc）+ 服务端 Pion/LiveKit（Go） | 自建媒体网关时用：ICE/SRTP/jitter buffer/拥塞控制都是现成且经过大流量验证的弱网栈 |
| 自管 UDP+KCP | 纯 C 单文件 + Go kcp-go | 后备：仅当完全自托管且需要手调重传参数时启用 |

主推顺序：**先 WSS 跑通真实云链路**（云厂商协议本身已定义会话延续、VAD、
打断等语义，传输只是载体）；弱网实测不达标或需要自托管网关时上 WebRTC。
会话延续性（上下文、打断、多轮状态）属于协议/会话层的事，不压在传输层。

音频回传与控制流天然解耦：识别/评测结果是可靠文本流（WSS 或 WebRTC
data channel 承载均可）；音频方向本质上就是一段内存的管理问题，继续走
本模块的环形缓冲，不因传输栈更换而改动。

## 5. 接口（src/contracts.dart）

```
AudioFormat    采样率/声道/位深值对象
AudioFrame     seq + timestampMs + samples（定长 PCM）
MicSource      start() -> Stream<AudioFrame>；stop() 可随时中断
AudioSink      write(frame) 播放；欠载语义暴露给上层
AudioTransport sendFrame(frame)；events: Stream<TransportEvent>（识别/评测/丢帧统计）
```

平台实现各自挂到这三个接口后面，UI/控制器只依赖接口。

## 6. 路线图（直接上手，不设 smoke test）

- R1 接口抽象 + Dart 参考环形缓冲 + 单测（本轮）
- R2 C 版 SPSC 环形缓冲（NDK/Xcode 双端编译），对齐 R1 用例
- R3 切帧器（纯 Dart + 单测），接在缓冲出口
- R4 Android Oboe 输入流写缓冲，真机验延迟（回环时间戳法）
- R5 iOS RemoteIO 输入流 + ObjC++ 会话壳
- R6 回包链路：WSS echo（或 Pion 回声）打通协议，再接真实云服务
- R7 AudioSink 播放路径（Oboe 输出流），欠载策略与丢帧统计
- R8 端到端联调：弱网（丢包/延迟注入）下验证传输方案选型与缓冲预算

## 7. 验证口径

本目录改动必须跑的：`flutter analyze`、`flutter test`（R1/R3/R6 阶段），
C 层用各端单元测试入口回归；真机项（R4/R5/R7/R8）以实测延迟/丢包数字为准。
本机当前没有 Flutter SDK 时，验证由有环境的机器执行，不在记录里写没跑过的结果。
