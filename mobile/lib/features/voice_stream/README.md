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
| web 采集 | Dart（`package:web`，getUserMedia + AudioWorklet） | web 无 dart:ffi 与原生库；浏览器麦克风走 getUserMedia，C 层不参与 |
| 服务端 | Go（仓库既有） | 复用现有骨架 |

明确不写：Kotlin/Swift 业务逻辑。Android 采集走 Oboe（AAudio 路径），
不走 Java `AudioRecord`。

平台装配走条件导入（web 打包可编译、原生行为不变）：插件绑定层
`bindings.dart`（原生 FFI ↔ web 桩）与模块平台工厂
`src/mic_source_factory.dart`（[NativeMicSource] ↔ [WebMicSource]）。
web 侧的 Float32→I16 转换在 `src/pcm_convert.dart`（纯函数可单测）。
App 装配层注册了调试路由 `/voice-debug`（`voice_debug_page.dart`）：
经工厂取当前平台 [MicSource]，web 打包后在浏览器授权麦克风即可实测
宿主系统麦克风（Windows 上为 Chrome/Edge 授权的系统默认麦克风）。

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

`flags` 标记丢帧空洞：上游（环缓丢旧/采集断流）drop 后由切帧器把
`gapBefore = 0x0001` 打在下一帧上；与 seq 跳号互为补充——跳号是接收方
被动观测，本位是发送方主动标记，服务端据此估算丢包率。

R3 已实现 `src/frame_slicer.dart`：`FrameSlicer`（纯 Dart 切帧，`push`/
`drain`/`flush`/`markGap`，`drain` 直接接环形缓冲出口并同步成对
peek/advance）与 `FrameHeaderCodec`（帧头小端序编解码）。

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

### 5.1 会话层（v0.2 新增，参照上游已验证语义）

`src/session.dart` 在 `AudioTransport` 之上叠加会话层，语义参照原仓库
XE3-ESL 的 `speakup.voice-input.v1`（start/finish/cancel 控制帧、幂等键、
partial/final 区分、kind+retryable 失败模型），但只定义抽象，不绑定传输栈：

- **生命周期**：`idle → active → (finish → 等终态 | cancel) → closed`。
  `finish()` 是优雅结束（等服务端 final/failed 终态事件），
  `cancel()` 是立即中断；终态后不可复用，重试须新开会话。
- **幂等键**：`VoiceSessionConfig.idempotencyKey`（8–128 字符、首尾无空白），
  同一逻辑会话跨重连重试保持不变，服务端据此去重。
- **事件模型**：`SessionStarted / SessionPartial / SessionFinal /
  SessionFailed(kind, retryable) / SessionStats`。`retryable=true` 时
  可用同一幂等键重开；传输映射细节由 R6 的 WSS 实现完成。

`AudioTransport` 原契约不变——不破已有用户空间；`VoiceSessionLifecycle`
状态机是纯 Dart 组件，实现层组合复用，语义由单测固定。

## 6. 路线图（直接上手，不设 smoke test）

- R1 ✅ 接口抽象 + Dart 参考环形缓冲 + 单测
- R2 ✅ C 版 SPSC 环形缓冲（header-only，NDK/Xcode 双端编译），语义对齐 R1
- R3 ✅ 切帧器（纯 Dart + 单测）+ 帧头编解码，接在缓冲出口
- R4 ◐ Android Oboe 输入流已落地（`plugins/voice_input` + 共享 C ABI
  `native/voice_input.h`）；真机验延迟（回环时间戳法）待 R8
- R5 ◐ iOS RemoteIO 输入流 + ObjC++ 会话壳已落地（同一 C ABI）；
  需 mac + 真机编译验证
- R6 ✅ 回包链路：WSS echo 打通协议（客户端 `WssEchoTransport` +
  会话适配 `TransportVoiceSession`；服务端 `/ws/voice/echo`）；
  真实云服务接入另行推进
- R7 ◐ AudioSink 播放路径已落地（`NativeAudioSink` + Oboe 输出流 + RemoteIO 渲染回调 + `playback_queue` 欠载状态机）；真机实测待 R8
- R8 端到端联调：弱网（丢包/延迟注入）下验证传输方案选型与缓冲预算
- web ✅ 打包支持落地：`mobile/web` 脚手架启用，`flutter build web` 编译图
  含采集链路 web 分支（getUserMedia + AudioWorklet）；`/voice-debug` 路由
  供浏览器实测；App 装配层 HTTP 客户端已改为平台无关实现（package:http）

### 6.1 与三段式语音链路（其他成员实现）的关系

其他成员的语音功能走三段式：整段录音 → speech2text → 大模型 → text2speech
→ 整段播放，全程以文本/音频文件为载体，不产生流式音频。取舍如下：

- 三段式的代价：录音、STT、LLM、TTS 四段串行，首音延迟是各段之和；
  播放中无法打断重说；也没有弱网丢帧语义（整段上传成败二元）。
  在这个形态里，Oboe 流式采集与 playback_queue 确实用不上——任意录音
  插件加媒体播放器即可完成，本模块对该形态是超配。
- 本模块的定位：瞄准流式升级路径。云厂商实时语音 API 是同一条 WebSocket
  内 ASR/LLM/TTS 融合：partial 识别实时回流、合成音频分块下发、随时打断。
  R6 会话层语义（幂等键、partial/final、kind+retryable）正是按该形态设计，
  Oboe 采集帧（seq/timestampMs/gapBefore）的端到端延迟与丢包观测也只有
  在流式形态下才有意义。
- 并存方式：契约不变，装配层换实现。三段式上线的短期场景里，本模块的
  [AudioSink] 仍可复用——TTS 音频分块回包交给 playback_queue 迟滞起播，
  平滑合成端抖动；采集侧继续用任意录音实现。R8 联调优先验证流式链路，
  三段式作为功能兜底并行存在。

## 7. 验证口径

本目录改动必须跑的：`flutter analyze`、`flutter test`（R1/R3/R6 阶段），
C 层用各端单元测试入口回归；真机项（R4/R5/R7/R8）以实测延迟/丢包数字为准。
本机当前没有 Flutter SDK 时，验证由有环境的机器执行，不在记录里写没跑过的结果。

### 7.1 验证记录

- 2026-09-05 web 打包支持，环境：GitHub Codespace
  `musical-invention-p49pw6j5g5jcrwvp`（`/workspaces/ai-speak`，Flutter
  3.47.2 stable / Dart 3.13.2）：
  - `flutter analyze`：0 error / 0 warning（余 5 条 info 为 coaching
    preparation 既有提示，与本模块无关）。
  - `flutter test`（VM）：72 通过，2 失败（与本模块无关的既有用例，不涉及
    本模块与本次改动路径，另立任务处理，细节口头同步）。
  - `flutter build web`：通过；web 编译图覆盖 `mic_source_factory` web
    分支、`WebMicSource`、`pcm_convert` 与插件绑定层 web 桩全链路。
  - 浏览器内 getUserMedia 实测（Windows 宿主麦克风）：待人工执行——
    `flutter run -d web-server` 后经 Codespace 端口转发在 Windows 浏览器
    打开 `/voice-debug` 授权麦克风；浏览器运行时用例（`--platform
    chrome`）因容器无浏览器暂未跑。

## 8. 编码约定

注释统一 Doxygen 风格：Dart 用 `/// @brief` / `@param` / `@return` /
`@note`，成员行内注释用 `///<`；C 层实现沿用同一套标签（`@file` /
`@brief` / `@param` / `@return`）。接口语义变更时同步更新
`docs/interfaces.md`（本模块目录内，`mobile/lib/features/voice_stream/docs/interfaces.md`）。
