# ai-speak 需求清单

> 每日根据真实进展更新勾选状态。已勾选表示当前版本已有可运行实现或经过测试的代码；仅有接口契约、数据结构或占位处理器的功能仍保持未完成。

## M1 概念验证

- [x] R1 提供服务健康检查接口，确认 Go Server 正常运行
- [x] R2 展示可用的英语口语练习场景列表，并过滤非 active 场景与重复场景
- [x] R3 查看场景详情、练习目标、角色和练习选项
- [x] R4 在 Flutter 客户端按练习经验分组展示场景，并支持选择练习目标与练习模式

## M2 正式开发

- [x] R5 用户注册、登录、退出登录和当前用户信息查询（主干 PR #18 已合入并经本地集成）
- [ ] R6 创建、查看、激活和完成口语练习会话
- [ ] R7 支持练习过程中的文本回答提交与会话状态管理
- [ ] R8 支持语音流采集、传输、播放及中断恢复（voice_stream 模块由 24320121 推进）
  - [x] R8.1 接口契约抽象（contracts.dart）与 SPSC 环缓 Dart 参考实现（9 个单测转正）
  - [x] R8.2 C 版 SPSC 零拷贝环形缓冲（header-only，双端编译，无锁原子序读写）
  - [x] R8.3 定长切帧器与 12 字节帧头编解码（AudioFrame flags 丢包标记，云端 32 个单测全绿）
  - [x] R8.4 Android Oboe 低延迟采集插件与 C ABI 规范（native/voice_input.h）
  - [x] R8.5 iOS RemoteIO 采集与 AVAudioSession 激活壳
  - [x] R8.6 WSS 回声流式传输与会话生命周期状态机（客户端 TransportVoiceSession + 服务端 /ws/voice/echo）
  - [x] R8.7 AudioSink 播放路径与欠载状态机（NativeAudioSink 与 playback_queue 落地）
  - [ ] R8.8 端到端弱网联调与真机延迟实测
- [ ] R9 创建 Agent 对话线程、发送消息并运行对话任务
- [ ] R10 生成基于证据的练习评测报告，提供维度评分、反馈和复练建议
- [ ] R11 将用户、场景、练习会话、对话记录和评测报告持久化到 PostgreSQL
- [ ] R12 完成 Flutter 端练习进行中、结果查看和历史记录页面

## M3 缺陷修复与交付

- [ ] R13 为核心接口补充鉴权、参数校验、错误码和超时处理
- [x] R14 完成 Go、Flutter 自动化测试及 CI 检查（voice_stream 建立 GitHub Codespaces 云端全自动化验证闭环）
- [ ] R15 完成部署配置、运行文档和期末项目材料
