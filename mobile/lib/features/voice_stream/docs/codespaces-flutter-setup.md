# Codespaces Flutter 环境使用指南

本文记录如何用 GitHub Codespaces 搭建 Flutter 编译测试环境，以及日常使用命令。适用于 `mobile/` 下语音实时流模块（`voice_stream`）的开发验证。

> 2026-09-03：本文由 `docs/24320106/codespaces-flutter-setup.md` 迁入模块目录
> （`mobile/lib/features/voice_stream/docs/`），日报和提示词记录里出现的旧路径按此对应。

## 架构概览

- `.devcontainer/Dockerfile`：基于 `mcr.microsoft.com/devcontainers/base:ubuntu-24.04`，在**镜像构建阶段**安装 Flutter stable 到 `/opt/flutter`（只装 universal 产物，不装 Android/iOS SDK），并预装 Go 1.22（devcontainer feature）。
- `.devcontainer/devcontainer.json`：使用上述 Dockerfile；创建容器后只执行 `cd mobile && flutter pub get`；包含 `sshd` feature（供 `gh codespace ssh` 远程连接）。
- `.devcontainer/setup-flutter.sh`：备用脚本，仅在镜像内 Flutter 不可用时手动执行。

## 一次性准备（已完成）

1. Gitee 仓库配置了从 GitHub 的自动同步镜像；日常开发以 GitHub 为主仓库。
2. Fork：`https://github.com/moment-NEW/ai-speak`，本地已添加 remote `fork`。
3. 本地安装 GitHub CLI（`scoop install main/gh`），`gh auth login` 登录并补充 `codespace` 权限：
   `gh auth refresh -h github.com -s codespace`。

## 创建 Codespace

在网页上：仓库切到 `dev/voice_stream` 分支 → Code → Codespaces → `+`。

或命令行：

```bash
gh codespace create -R moment-NEW/ai-speak -b dev/voice_stream
```

机型选 2 核 8G 即可。镜像已缓存时创建很快；首次触发预构建（prebuild）时需要等 10–25 分钟构建镜像，可跳过直接创建。

## 常用命令（本地执行）

```bash
gh codespace list                      # 列出机器
gh codespace ssh -c <机器名>           # SSH 连入
gh codespace stop -c <机器名>          # 停机省额度
gh codespace delete -c <机器名>        # 删除机器
gh codespace logs -c <机器名>          # 查看创建日志
```

## 在 Codespace 内编译测试

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

## 2026-09-03 首次搭建验证结果

- 机器：`musical-invention-p49pw6j5g5jcrwvp`（2 核 8G，32G 存储）
- Flutter 3.47.2 stable / Dart 3.13.2
- `flutter pub get`：成功
- `flutter analyze`：No issues found
- `flutter test`：21 个用例全部通过（ring_buffer / session 状态机）

## 编译验证方式

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

## 2026-09-03 编译结果

- `flutter build linux --release`：✓ Built `build/linux/x64/release/bundle/ai_speak`
- 产物 bundle 约 21M（含引擎 so、icudtl.dat、应用二进制 24K）

## 踩坑记录

- 不要在 `onCreateCommand` 里现场 `git clone` Flutter SDK：创建阶段耗时过长会触发平台诊断刷屏甚至卡死。SDK 必须打进镜像。
- 自定义镜像需要显式加入 `ghcr.io/devcontainers/features/sshd:1`，否则 `gh codespace ssh` 报 "failed to start SSH server"。
- 配置改动后需在 Codespace 内 `git pull` 再 `Rebuild Container` 才生效。
- 预构建（prebuild）是可选的镜像缓存加速，不是创建 Codespace 的必要条件；卡在 Uploading 时可直接选择无预构建创建。
- Windows 本机新版 OpenSSH 与 `gh codespace cp` 不兼容（报 "dest open ... No such file or directory"，即使远端路径存在）：需强制旧 scp 协议，加 `-- -O`，如 `gh codespace cp -c <机器名> -- -O 本地文件 remote:远端文件`；前缀用 `remote:`（本机 gh 版本不认 `cpsc:`）。

## 2026-09-03 R3 验证结果

- 流程：本地提交并 `git push fork dev/voice_stream` → `gh codespace ssh`
  内 `git pull --ff-only` → 在 `/workspaces/ai-speak/mobile` 执行
  （本机无 Flutter/Docker/WSL）：
- `flutter analyze`：No issues found
- `flutter test`：32 个用例全部通过（ring_buffer 9 / session 12 / frame_slicer 11）
