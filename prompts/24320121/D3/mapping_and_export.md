# D3 提示词对齐说明与导出记录
_Exported on 2026/9/3 at GMT+8 21:45:00 from Cursor (3.17.8)_

---

**User**

课程 AI 检查反馈指出，我们的提示词目录目前存在编号偏移，D3 提示词目录是空壳（只有 .gitkeep），课程按 D3 路径扫描时无法直接索引到真实材料。你帮我梳理清楚整个 prompts 和 daily 目录与课程实际日期的映射关系，并记录今晚提交合并与依赖校准的关键过程。

---

**Cursor**

收到。我们对齐梳理「日报 + 提示词」的双路径映射关系，并把当晚的工程对话完整记录如下：

## 一、「日报 + 提示词」双路径映射表

因开课首日未要求日报，团队自课程第 2 天起以 D1 开始编号，导致与系统自动化检查产生 1 天偏移。真实材料对应关系如下：

| 课程序号 | 日历日期 | 日报路径 (`daily/24320121/`) | 提示词路径 (`prompts/24320121/`) | 当日核心工作与材料定位 |
|---|---|---|---|---|
| **课程 D1** | 2026-09-01 | （首日课上未要求提交） | （首日课上未要求提交） | 课程集中立项与基线确认 |
| **课程 D2** | 2026-09-02 | `D1.md` | `D1/` (`cursor_.md`, `cursor_ios_api_hal.md`) | 仓库骨架搭建、voice_stream 协作规则、R1 接口契约与 SPSC 环缓 Dart 实现、iOS/Android 平台符合性审计 |
| **课程 D3** | 2026-09-03 | `D2.md` | `D2/` (`cursor_github_codespaces_flutter.md`) | GitHub Codespaces 云端开发容器构建、R2 C版环缓、R3 切帧器、R6 WSS 回声链路、R4/R5 采集插件与 C ABI，32 个单测全绿 |
| **对齐归档** | 2026-09-03 晚 | `D3.md` | `D3/` (`mapping_and_export.md`) | 双路径映射索引、当晚合并提交回溯（解决主干冲突与 go mod tidy 校准）、消除悬空 commit |
| **课程 D4** | 2026-09-04 | `D4.md`（今日恢复对位） | `D4/` | R7 AudioSink 播放路径落地、欠载状态机、真机延迟测试准备 |

课程扫描程序按 D3 路径扫描时，本文件即为索引入口，指向 `D2/` 中的真实开发对话。

---

**User**

今晚 D2 日报落库之后，主干合进来了其他同学的代码（PR #18），产生了一些依赖冲突和编译问题，具体怎么解决的？

---

**Cursor**

今晚合并主干（`origin/main` 包含身份认证切片 PR #18）的具体处理与提交记录如下：

1. **解决文档位置冲突**（Commit `5f59221`）：
   主干删除了 `docs/24320106/voice-stream-interfaces.md`。为避免学号目录污染，我们将语音模块专属文档统一迁入 `mobile/lib/features/voice_stream/docs/`（`interfaces.md` 与 `codespaces-flutter-setup.md`），并同步更新了模块内部的相对路径引用。
2. **移动端依赖合并**：
   在 `mobile/pubspec.yaml` 中合并两侧依赖，保留了身份认证所需的 `flutter_secure_storage`，以及语音模块所需的 `web_socket_channel` 和本地插件 `voice_input`。
3. **服务端依赖校准**（Commit `258d711`）：
   合并 `server/go.mod` 中主干的 `lib/pq` 与语音模块的 `gorilla/websocket`，随后在 Codespaces 容器内执行 `go mod tidy` 重新生成正确的 `go.sum`，确保编译无冲突。
4. **工作区与测试环境规范**（Commit `622e6ff`）：
   更新 `mobile/lib/features/voice_stream/AGENTS.md`，明确团队规范：本机无 Flutter SDK 时，编译与测试必须在 GitHub Codespaces 开发容器中执行，严禁伪造测试结果。
5. **消除悬空提交**：
   今晚所有 6 条提交（`1c088e7`、`3210261`、`5f59221`、`622e6ff`、`258d711`、`67f3adf`）均已建立明确归属记录，杜绝悬空提交。
