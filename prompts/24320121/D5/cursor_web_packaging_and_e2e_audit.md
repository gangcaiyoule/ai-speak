# Flutter web 打包支持与全链路可用性审计
_Exported on 2026/9/5 at GMT+8 16:30 from Cursor (dev/voice_stream)_

---

**User**

你先把我们目前的C Oboe，稍作调整，使其能支持flutter打包为web后依然能正常调用windows的麦克风。

**Cursor**

结论先行：Oboe 是 Android 专用 C++ API，浏览器里没有 dart:ffi 也跑不了原生库，web 端不可能复用 Oboe 代码；等价能力是浏览器的 getUserMedia + AudioWorklet。上一提交 7b56699（WebMicSource + 绑定层条件导出）已把 web 采集写好，本轮工作是真正打通「打包为 web」。实际改动（提交 e096cea / fea0681 / 1c78ab9 / e29a00c / 11328e3 / 365f32f）：

1. 启用 web 平台：mobile/ 原本只有 android/ios，flutter build web 直接不可用。经 `flutter create --platforms=web` 生成 mobile/web 脚手架（index.html/manifest/图标），并手工恢复 .metadata 中的 android/ios 记录。
2. 拆掉 web 编译障碍：main.dart 依赖链上的 identity_client.dart、wire_scene_client.dart、wire_practice_plan_client.dart 直接用 dart:io 的 HttpClient，web 上不存在。统一改为平台无关的 package:http（web 走 fetch、原生走 socket），新增 http 依赖。
3. 修真实编译错误（也是「导出名下游不可见」的根源）：下游不能引用分支专属的 FfiVoiceInputBindings/FfiVoiceOutputBindings，NativeMicSource/NativeAudioSink 改用两分支同名导出的 createDefault*Bindings() 平台工厂；WebMicSource 的 Blob 构造类型、unsafeCast（本 SDK 已移除该 API，改为 JS 类型 as 转换）、缺失的 dart:typed_data 导入、PracticePlanClientException 未导出到 practice_plan_client.dart 逐个修复；plugin 的 FFI 符号查找改惰性（late final），无原生库环境构造绑定不再抛错。
4. 新增 /voice-debug 调试路由（voice_debug_page.dart）：经平台工厂启动采集显示帧计数，web 与原生共用，供浏览器实测。

验证（环境：Codespace musical-invention-p49pw6j5g5jcrwvp，Flutter 3.47.2 stable / Dart 3.13.2，/workspaces/ai-speak）：flutter analyze 0 error 0 warning（余 5 条 info 为 coaching preparation 既有提示）；flutter test（VM）72 通过、2 失败（coaching/preparation 既有状态机用例，用例纯假实现不经本次改动路径，另立任务）；flutter build web 通过，且 speak_up_app 引入调试页后 web 编译图覆盖 mic_source_factory web 分支、WebMicSource、pcm_convert、插件 web 桩全链路。浏览器内 getUserMedia 实测（Windows 宿主麦克风）待人工：flutter run -d web-server 后经端口转发在 Windows 浏览器开 #/voice-debug 授权即可。结果按环境署名记入模块 README 7.1。

---

**User**

还好我的习惯就是接口和具体实现解耦。既然走Web的话必然无法绕开浏览器内核，那么就这么办罢。做完之后拿云端环境编一遍。然后我记得其他人做的部分是三段式的，也就是输入->speech2text->大模型返回text2speech。这其实很浪费我们的oboe，但是确实是值得注意的东西

**Cursor**

云端编译上一轮已完成（flutter build web 通过），其后仅剩纯文档提交，无需重编。三段式观察已核实并写成 README 6.1 节备忘（提交 6e9f715）：仓库已合并的 server agent 模块只有 TextGenerator 接口加 Stub（纯文本进出），无 ASR/TTS，队友的三段式不在合并代码里。要点：三段式形态下 Oboe 确实超配（整段录音加播放器即可，四段串行首音延迟为各段之和、无法打断、无丢帧语义）；但本模块瞄准流式升级路径——云厂商实时语音 API 在一条 WebSocket 融合 ASR/LLM/TTS，partial 实时回流、分块下发、可打断，R6 会话层（幂等键、partial/final、retryable）与帧头 seq/timestamp/gap 正按此设计。两者靠解耦并存：契约不动装配换实现；短期三段式下播放侧 AudioSink 仍可复用（分块 TTS 回包交 playback_queue 迟滞起播）。R8 联调建议优先验证流式链路。

---

**User**

你过一遍整体，看看这个程序是否真的完全可用，功能是否打通。如果没有，狠狠压力其他人

**Cursor**

审计方法：读服务端路由与装配（cmd/server/main.go、compose.yaml、agent/http.go、coaching/http.go），核对 git 提交归属，然后在 Codespace 实际起服务器按用户旅程逐个打接口。服务端 go build ./... 与 go test ./... 全绿（agent/coaching/evaluation/identity/practice/scene/voiceecho 全部 ok）。

实测打通：注册 201、登录 200（Bearer token）、GET /v1/me 200、无 token 401；GET /v1/scenes 200（self-introduction、project-deep-dive 含角色与练习选项）；练习计划创建 201、列表 200；R6 回声 WS 升级 101。移动端同环境 analyze 0 error、test 72 通过。

断裂点（按用户旅程）：
1. 「开始练习」按钮死路——speak_up_app.dart 装配 PreparationPage 时未传 onStartSession 回调，点「开始会话」无反应；且服务端 POST /v1/practice-sessions 也是 501。
2. 练习会话与评测六连 501：practice-sessions 的创建/答题/完成/评测/报告查询全部 not_implemented。评测域模型与 Postgres 仓库在，但仓库内零真实 Scorer/LLM 调用。
3. AI 对话零实现：服务端 agent 六端点全 Stub 501；移动端 AgentClient 只有接口无实现类、无 UI 引用。
4. 语音链路未进产品流：voice_stream 代码测试最完整（原生+web 双端）但 App 只有 /voice-debug 调试页；真实 ASR/LLM/TTS 一条未接，R6 只到回声协议。
5. 部署形态缺失：compose.yaml 只有 postgres 没有 server；不带 DATABASE_URL 时内存库重启即丢数据（实测：重启后原账号登录 401）。
6. 移动端 preparation_controller_test 两个用例红（既有，假实现不经网络路径）。

压力点（按 git 提交归属）：
- SiCheng Zhang（接口与 identity Postgres 作者）：practice-sessions HTTP 域接线，接口 58c2aed 已定义、域和端口在，只差把 notImplemented 换成真实现，是当前最大单点阻塞。
- QiuZhiyu（evaluation 域与仓库作者）：补一个真实 Scorer（先接最简 LLM 文本评测即可），并把 evaluation 仓库接进 main.go 装配——评测在服务器上目前是装饰品。
- yejunbo（coaching 移动端作者）：onStartSession 装配 + PracticeClient/EvaluationClient 的 Wire 实现（目前只有接口），顺手修两个红测试。
- 仓库管理员（19730539862）：compose 补 server 服务、定 DATABASE_URL 部署形态。
- 本人（voice_stream）：R8 端到端联调、浏览器与真机实测，但前提是上面 1–3 有东西可接。

一句话总结：服务端单测全绿、真实旅程三步断——计划建好了，开始按钮按下去是空气，AI 对话和评测是 501 装饰品；建议把「practice-sessions 接线 + 真实 Scorer」立为最高优先级，它们是移动端所有断点的下游。

---

**User**

把今天的对话上下文内容写进24320121的prompt。注意不要超过1w字数

**Cursor**

即本文件：prompts/24320121/D5/cursor_web_packaging_and_e2e_audit.md，按既有 User/Cursor 交替实录格式收录当日三轮对话的上下文与结论，全文控制在 1 万字以内。当日分支 dev/voice_stream 提交链：e096cea（web 打包启用+http 解耦+调试页）→ fea0681（web 脚手架）→ 1c78ab9（绑定工厂化+编译修复）→ e29a00c（unsafeCast 移除修复）→ 11328e3（practice-plan 请求体修复）→ 365f32f（FFI 惰性查找）→ ecab68c / 6e9f715（文档：验证记录、三段式备忘）。
