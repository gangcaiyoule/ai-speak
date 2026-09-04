# Codespaces Flutter 环境使用指南与验证记录

> **材料归属与协作说明**：
> - **作者**：陈冠亨（学号：24320121）
> - **负责模块**：AI 对话与实时语音链路（`voice_stream`）
> - **背景与协作**：在本地开发机缺乏 Flutter SDK 及 Docker/WSL 环境的情况下，本文档由本人独立攻关搭建并编写，记录了通过 GitHub Codespaces 云端开发容器构建 Flutter 3.47.2 镜像、自动化测试（`flutter analyze` / `flutter test`）以及 Windows OpenSSH scp 协议踩坑与修复全过程。
> - **工程位置对照**：模块工程内的共享文档位于 `mobile/lib/features/voice_stream/docs/codespaces-flutter-setup.md`；按课程提交规范，现将本人维护的完整版本归入个人材料空间 `docs/24320121/`，供课程组作为个人工程能力与云端自动化闭环证据进行核验。

---

## 一、架构概览

- `.devcontainer/Dockerfile`：基于 `mcr.microsoft.com/devcontainers/base:ubuntu-24.04`，在**镜像构建阶段**安装 Flutter stable 到 `/opt/flutter`（只装 universal 产物，不装 Android/iOS SDK），并预装 Go 1.22（devcontainer feature）。
- `.devcontainer/devcontainer.json`：使用上述 Dockerfile；创建容器后只执行 `cd mobile && flutter pub get`；包含 `sshd` feature（供 `gh codespace ssh` 远程连接）。
- `.devcontainer/setup-flutter.sh`：备用脚本，仅在镜像内 Flutter 不可用时手动执行。

## 二、一次性准备（已完成）

1. Gitee 仓库配置了从 GitHub 的自动同步镜像；日常开发以 GitHub 为主仓库。
2. Fork：`https://github.com/moment-NEW/ai-speak`，本地已添加 remote `fork`。
3. 本地安装 GitHub CLI（`scoop install main/gh`），`gh auth login` 登录并补充 `codespace` 权限：
   `gh auth refresh -h github.com -s codespace`。

## 三、创建 Codespace

在网页上：仓库切到 `dev/voice_stream` 分支 → Code → Codespaces → `+`。

或命令行：

```bash
gh codespace create -R moment-NEW/ai-speak -b dev/voice_stream
```

机型选 2 核 8G 即可。镜像已缓存时创建很快；首次触发预构建（prebuild）时需要等 10–25 分钟构建镜像，可跳过直接创建。

## 四、常用命令（本地执行）

```bash
gh codespace list                      # 列出机器
gh codespace ssh -c <机器名>           # SSH 连入
gh codespace stop -c <机器名>          # 停机省额度
gh codespace delete -c <机器名>        # 删除机器
gh codespace logs -c <机器名>          # 查看创建日志
```

## 五、在 Codespace 内编译测试

```bash
cd /workspaces/ai-speak/mobile
flutter pub get
flutter analyze
flutter test
```

服务端测试（Go 已预装）：

```bash
cd /workspaces/ai-speak/server
go test ./...
```

## 六、2026-09-03 首次搭建验证结果

- 机器：`musical-invention-p49pw6j5g5jcrwvp`（2 核 8G，32G 存储）
- Flutter 3.47.2 stable / Dart 3.13.2
- `flutter pub get`：成功
- `flutter analyze`：No issues found
- `flutter test`：21 个用例全部通过（ring_buffer / session 状态机）

## 七、编译验证方式

`mobile/` 目前只有 `lib/` 和 `test/`，没有平台目录，因此没有直接的 `flutter build` 目标。验证完整编译采用临时工程法（不改动仓库文件）：

```bash
rm -rf /tmp/build_check
flutter create --project-name ai_speak --platforms linux /tmp/build_check
cp -r /workspaces/ai-speak/mobile/lib /tmp/build_check/
cp /workspaces/ai-speak/mobile/pubspec.yaml /tmp/build_check/
cd /tmp/build_check && flutter pub get && flutter build linux --release
```

Linux 桌面构建依赖需先在机器上安装（当前 Codespace 已装）：

```bash
sudo apt-get install -y --no-install-recommends \
  clang cmake ninja-build pkg-config libgtk-3-dev liblzma-dev
```

## 八、2026-09-03 编译结果

- `flutter build linux --release`：✓ Built `build/linux/x64/release/bundle/ai_speak`
- 产物 bundle 约 21M（含引擎 so、icudtl.dat、应用二进制 24K）

## 九、踩坑记录

- 不要在 `onCreateCommand` 里现场 `git clone` Flutter SDK：创建阶段耗时过长会触发平台诊断刷屏甚至卡死。SDK 必须打进镜像。
- 自定义镜像需要显式加入 `ghcr.io/devcontainers/features/sshd:1`，否则 `gh codespace ssh` 报 "failed to start SSH server"。
- 配置改动后需在 Codespace 内 `git pull` 再 `Rebuild Container` 才生效。
- 预构建（prebuild）是可选的镜像缓存加速，不是创建 Codespace 的必要条件；卡在 Uploading 时可直接选择无预构建创建。
- Windows 本机新版 OpenSSH 与 `gh codespace cp` 不兼容（报 "dest open ... No such file or directory"，即使远端路径存在）：需强制旧 scp 协议，加 `-- -O`，如 `gh codespace cp -c <机器名> -- -O 本地文件 remote:远端文件`；前缀用 `remote:`（本机 gh 版本不认 `cpsc:`）。

## 十、2026-09-03 R3 验证结果

- `git pull --ff-only` 同步本地推送到 fork 分支的 R3 commit（`ccfa931`）。
- `flutter test` 运行 32 个用例全部通过（耗时约 4 秒）：
  - `ring_buffer_test.dart`（9 个）：覆盖回绕、丢旧、视图只读、超量保尾、负数越界等。
  - `session_test.dart`（12 个）：覆盖生命周期状态转移、幂等键校验、事件流广播、超时重试。
  - `frame_slicer_test.dart`（11 个）：覆盖整除切分、跨 push 拼接、零填充、flush、gapBefore 标记、帧头编解码等。
