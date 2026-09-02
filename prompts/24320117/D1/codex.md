# 

> "C:\Users\yejunbo\Documents\Codex\2026-09-02\ih\outputs\24320106-course-package\prompts\D1\day1-development-prompt.txt"按照这里的要求写代码，规范在这里"C:\Users\yejunbo\Desktop\24320106-course-package"

已完成场景列表、详情与练习配置最小流程，修改位于 Gitee 工作区：

`C:\Users\yejunbo\Documents\Codex\2026-09-02\ih\work\gitee-repo`

主要修改：

- 新增 Flutter 场景模型、Wire Client、JSON 解码器。
- 新增 `PreparationController`，支持加载、空列表、失败重试、详情版本校验、角色切换和选项筛选。
- 新增 Practice 页面和路由入口。
- Go 服务端实现 `/v1/scenes`、详情和角色接口。
- 新增 OpenAPI 契约及 Controller、服务端测试。
- 删除旧的重复 `SceneClient` 声明。
- 未修改 `daily/`、`prompts/`、`docs/`，也未实现实时语音、练习报告、JD、简历和计划功能。

验证结果：

- `gofmt`
- `go test ./...`：通过
- `go vet ./...`：通过
- `git diff --check`：通过
- `flutter pub get`、`dart format`、`flutter analyze`、`flutter test`：未执行成功，原因是当前环境未安装 `flutter` 和 `dart`。

建议提交信息：

```text
feat(coaching): implement scene selection flow
```

当前仍需在安装 Flutter SDK 的环境中执行移动端格式化、静态检查和测试。

> 给出对应issue和pr文案

下面是可直接复制到 Gitee 的文案。

## Issue

**标题**

```text
[功能] 完成场景列表、详情和练习目标选择
```

**内容**

```markdown
## 功能描述

实现 SpeakUp 进入 Practice 后的场景选择最小流程：

进入 Practice
-> 加载场景列表
-> 按练习类型展示场景
-> 查看场景详情
-> 选择用户角色
-> 选择练习模式
-> 生成完整场景选择结果

## 实现范围

- 实现场景列表接口和客户端调用。
- 支持场景列表加载中、加载成功、空列表和请求失败状态。
- 请求失败时支持重试。
- 按服务端返回顺序展示场景。
- 实现场景详情和角色信息加载。
- 校验详情的 `scene_id` 和 `scene_version`。
- 展示场景描述、练习目标、用户角色、AI 角色、人物设定和练习重点。
- 实现用户角色选择。
- 根据角色筛选兼容的练习模式。
- 展示练习模式、名称和建议时长。
- 未完成角色或练习模式选择时，开始按钮不可用。
- 生成包含场景、角色 ID 和练习选项 ID 的选择结果。
- 补充 Flutter Controller 测试和 Go 服务端接口测试。

## 验收标准

- [ ] 可以进入 Practice 页面。
- [ ] 场景列表可以从接口加载。
- [ ] 场景按照服务端返回顺序展示。
- [ ] 场景列表为空时展示空状态，不构造默认场景。
- [ ] 场景列表请求失败时展示错误信息和重试入口。
- [ ] 点击场景后可以加载详情。
- [ ] 场景详情版本不一致时拒绝继续选择。
- [ ] 可以查看场景描述、目标、角色和练习重点。
- [ ] 切换角色后，练习选项会重新筛选。
- [ ] 未完成必要选择时，开始按钮不可用。
- [ ] 完成选择后生成正确的场景选择结果。
- [ ] 不实现实时语音、回答提交、评分和练习报告。

## 技术范围

- `mobile/lib/features/coaching/scene/`
- `mobile/lib/features/coaching/preparation/`
- `mobile/lib/app/`
- `mobile/test/coaching/preparation/`
- `api/modules/scene.yaml`
- `server/internal/coaching/scene/`
- `server/internal/coaching/http.go`

## 不在本 Issue 范围内

- 岗位 JD 和简历上传。
- 练习计划管理。
- 实时问答。
- 麦克风录音、语音转写和语音播放。
- 练习中的回答提交和轮次推进。
- 评分、复盘和练习报告。
- 远程 Issue、Commit 或 Pull Request 自动创建。
```

## PR

**标题**

```text
feat(coaching): implement scene selection flow
```

**内容**

```markdown
## 功能描述

完成 Practice 场景选择最小用户流程，包括场景列表、场景详情、角色选择和练习模式选择。

## 变更内容

- 新增场景领域模型：
  - `SceneDefinition`
  - `RoleDefinition`
  - `PracticeOption`
  - `PracticeObjective`
  - `SceneSelectionSnapshot`
- 新增场景 JSON Wire 解码和 HTTP Client。
- 新增场景列表加载、空状态、错误状态和失败重试。
- 新增场景详情加载和 `scene_id`、`scene_version` 校验。
- 新增角色选择和兼容练习选项筛选。
- 新增完整选择结果回调。
- 新增 Practice 页面和 `/practice` 路由。
- 实现场景服务端目录、详情接口和角色接口。
- 新增场景 OpenAPI 契约。
- 删除启动骨架中的重复 `SceneClient` 声明。
- 新增 Flutter Controller 测试和 Go 服务端测试。

## 实现思路

Flutter 端采用以下分层：

```text
Practice 页面
  -> PreparationController
  -> SceneClient
  -> WireSceneClient
  -> HTTP API
```

Controller 负责页面状态和选择流程，页面不直接访问 HTTP。详情加载失败时会清空旧详情，避免使用过期数据继续开始练习。角色切换后会重新计算兼容的 PracticeOption，并清理失效的旧选项。

服务端当前使用内存场景目录，保持服务端定义的场景顺序，后续可以替换为数据库或其他 Provider。

## 影响范围

- Flutter 场景选择页面和准备流程。
- Go 场景只读接口。
- 场景 OpenAPI 契约。

不会影响：

- 实时练习。
- 音频和语音处理。
- 练习回答提交。
- 评分、复盘和练习报告。
- 岗位准备、简历和练习计划。

## 实际执行的验证

```bash
cd server
gofmt -w internal/coaching/scene/scene.go internal/coaching/scene/scene_test.go internal/coaching/http.go internal/coaching/http_test.go
go test ./...
go vet ./...
```

结果：

- `go test ./...` 通过。
- `go vet ./...` 通过。
- `git diff --check` 通过。

## 未执行的验证

以下命令未能执行，原因是当前环境未安装 Flutter 和 Dart SDK：

```bash
cd mobile
flutter pub get
dart format --output=none --set-exit-if-changed lib test
flutter analyze
flutter test
```

## 手工验证

- 已通过 Go HTTP Handler 测试验证场景列表、场景详情、角色接口和不存在场景响应。
- Flutter 页面尚未在模拟器或真机运行验证。

## 已知限制

- 服务端场景数据当前保存在内存中，尚未接入数据库。
- 当前仓库环境没有 Flutter/Dart SDK，移动端测试和视觉验收需要在具备 Flutter 工具链的环境中补充。
- 本 PR 只完成场景选择流程，不负责创建练习会话或进入实时练习。

## 关联 Issue

```text
关联 #待填写
```

## AI 辅助说明

本次代码实现使用 AI Coding 工具辅助完成。开发者已检查修改范围、接口契约、状态流程和测试结果，并确认本 PR 不包含实时语音、评分、复盘和岗位准备功能。

## 提交前检查

- [x] 未修改 `daily/`、`prompts/`、`docs/` 课程记录目录。
- [x] 未提交密钥、Token、`.env`、缓存或构建产物。
- [x] 未创建远程 Commit、Issue 或 PR。
- [x] Go 格式化、测试和静态检查通过。
- [ ] Flutter 格式化、分析和测试待安装 SDK 后执行。
- [ ] 手工 Flutter 页面验收待完成。
```

推荐分支名：

```text
feature/<issue-number>-scene-selection
```

推荐 Commit：

```text
feat(coaching): implement scene selection flow
```

> 源头分支和目标分支分别是哪个

- 源头分支：个人功能分支，例如 `feature/<issue-number>-scene-selection`
- 目标分支：`main`

例如 Issue 编号是 `12`：

```text
源头分支：feature/12-scene-selection
目标分支：main
```

当前工作区仍在 `main`，尚未创建个人功能分支。

> pr内容再给我一份，不要一部分md，一部分文本的

```markdown
# PR 标题

feat(coaching): implement scene selection flow

## 功能描述

完成 Practice 场景选择最小流程：

进入 Practice
-> 加载场景列表
-> 按练习类型展示场景
-> 查看场景详情
-> 选择用户角色
-> 选择练习模式
-> 生成完整场景选择结果

## 变更内容

1. 新增场景领域模型：

   - `SceneDefinition`
   - `RoleDefinition`
   - `PracticeOption`
   - `PracticeObjective`
   - `SceneSelectionSnapshot`

2. 新增场景 JSON Wire 编解码和 HTTP Client。

3. 新增场景列表状态：

   - 加载中
   - 加载成功
   - 空列表
   - 请求失败
   - 失败重试

4. 新增场景详情加载和 `scene_id`、`scene_version` 校验。

5. 新增角色选择和练习选项筛选。

6. 角色切换后自动清理不兼容的旧练习选项。

7. 未完成角色或练习模式选择时，开始按钮不可用。

8. 新增 Practice 页面和 `/practice` 路由。

9. 服务端实现以下场景接口：

   - `GET /v1/scenes`
   - `GET /v1/scenes/{scene_id}`
   - `GET /v1/scenes/{scene_id}/roles`

10. 新增场景 OpenAPI 接口契约。

11. 新增 Flutter Controller 测试和 Go 服务端测试。

12. 删除启动骨架中的重复 `SceneClient` 声明。

## 实现思路

Flutter 端采用以下分层：

```text
Practice 页面
  -> PreparationController
  -> SceneClient
  -> WireSceneClient
  -> HTTP API
```

页面不直接访问 HTTP。Controller 负责加载状态、详情校验、角色选择和练习模式筛选。

详情请求失败时会清空旧详情，防止用户继续使用过期场景数据。角色切换后会重新计算兼容的练习选项，并清理失效的旧选择。

服务端当前使用内存场景目录，并按照服务端定义的顺序返回场景。后续可以替换为数据库或其他 Provider。

## 修改文件

- `mobile/lib/main.dart`
- `mobile/lib/app/app_routes.dart`
- `mobile/lib/app/speak_up_app.dart`
- `mobile/lib/features/coaching/coaching_clients.dart`
- `mobile/lib/features/coaching/scene/`
- `mobile/lib/features/coaching/preparation/`
- `mobile/test/coaching/preparation/preparation_controller_test.dart`
- `api/modules/scene.yaml`
- `server/internal/coaching/http.go`
- `server/internal/coaching/http_test.go`
- `server/internal/coaching/scene/scene.go`
- `server/internal/coaching/scene/scene_test.go`

## 影响范围

本 PR 影响：

- Practice 场景列表。
- Practice 场景详情。
- 用户角色选择。
- 练习目标展示。
- 练习模式选择。
- 进入后续准备流程前的场景选择结果。

本 PR 不影响：

- 实时问答。
- 麦克风录音。
- 语音转写和语音播放。
- 练习中的回答提交。
- 评分、复盘和练习报告。
- 岗位 JD。
- 简历上传。
- 练习计划管理。

## 实际执行的验证

```bash
cd server
gofmt -w internal/coaching/scene/scene.go internal/coaching/scene/scene_test.go internal/coaching/http.go internal/coaching/http_test.go
go test ./...
go vet ./...
```

验证结果：

- `go test ./...`：通过。
- `go vet ./...`：通过。
- `git diff --check`：通过。

## 未执行的验证

当前环境未安装 Flutter 和 Dart SDK，以下命令未执行成功：

```bash
cd mobile
flutter pub get
dart format --output=none --set-exit-if-changed lib test
flutter analyze
flutter test
```

原因：

```text
flutter: command not found
dart: command not found
```

## 手工验证

已通过 Go HTTP Handler 测试验证：

- 场景列表接口。
- 场景详情接口。
- 角色列表接口。
- 不存在场景的 `404` 响应。

Flutter 页面尚未在模拟器或真机中进行手工验证。

## 已知限制

1. 服务端场景数据当前保存在内存中，尚未接入数据库。
2. 当前环境缺少 Flutter/Dart SDK，移动端格式化、静态检查和测试需要在具备 Flutter 工具链的环境中补充。
3. 本 PR 不创建练习会话，也不进入实时练习页面。
4. 当前仅实现今天要求的场景选择最小流程。

## 关联 Issue

关联 Issue：`#待填写`

## 分支信息

源头分支：

```text
feature/<issue-number>-scene-selection
```

目标分支：

```text
main
```

## AI 辅助说明

本次代码使用 AI Coding 工具辅助完成。开发者已检查修改范围、接口契约、状态流程和测试结果，确认本 PR 未实现实时语音、评分、复盘、岗位准备和练习计划功能。

## 提交前检查

- [x] 未修改 `daily/`、`prompts/`、`docs/` 课程记录目录。
- [x] 未提交密钥、Token、`.env`、缓存或构建产物。
- [x] 未创建远程 Commit、Issue 或 PR。
- [x] Go 格式化、测试和静态检查已通过。
- [ ] Flutter 格式化、静态检查和测试待安装 SDK 后执行。
- [ ] Flutter 模拟器或真机手工验收待完成。
```

> 全部md格式给我

# feat(coaching): implement scene selection flow

## 功能描述

完成 Practice 场景选择最小流程：

进入 Practice
-> 加载场景列表
-> 按练习类型展示场景
-> 查看场景详情
-> 选择用户角色
-> 选择练习模式
-> 生成完整场景选择结果

## 变更内容

- 新增 `SceneDefinition`、`RoleDefinition`、`PracticeOption`、`PracticeObjective` 和 `SceneSelectionSnapshot` 模型。
- 新增场景 JSON Wire 编解码和 HTTP Client。
- 支持场景列表加载中、加载成功、空列表、请求失败和失败重试。
- 新增场景详情加载和 `scene_id`、`scene_version` 校验。
- 新增用户角色选择和练习选项筛选。
- 角色切换后自动清理不兼容的练习选项。
- 未完成角色或练习模式选择时，开始按钮不可用。
- 新增 Practice 页面和 `/practice` 路由。
- 实现以下服务端接口：
  - `GET /v1/scenes`
  - `GET /v1/scenes/{scene_id}`
  - `GET /v1/scenes/{scene_id}/roles`
- 新增场景 OpenAPI 接口契约。
- 新增 Flutter Controller 测试和 Go 服务端测试。
- 删除启动骨架中的重复 `SceneClient` 声明。

## 实现思路

Flutter 端采用以下分层：

```text
Practice 页面
  -> PreparationController
  -> SceneClient
  -> WireSceneClient
  -> HTTP API
```

页面不直接访问 HTTP。Controller 负责加载状态、详情校验、角色选择和练习模式筛选。

详情请求失败时会清空旧详情，防止用户继续使用过期场景数据。角色切换后会重新计算兼容的练习选项，并清理失效的旧选择。

服务端当前使用内存场景目录，并按照服务端定义的顺序返回场景。

## 修改文件

- `mobile/lib/main.dart`
- `mobile/lib/app/app_routes.dart`
- `mobile/lib/app/speak_up_app.dart`
- `mobile/lib/features/coaching/coaching_clients.dart`
- `mobile/lib/features/coaching/scene/`
- `mobile/lib/features/coaching/preparation/`
- `mobile/test/coaching/preparation/preparation_controller_test.dart`
- `api/modules/scene.yaml`
- `server/internal/coaching/http.go`
- `server/internal/coaching/http_test.go`
- `server/internal/coaching/scene/scene.go`
- `server/internal/coaching/scene/scene_test.go`

## 影响范围

本 PR 影响：

- Practice 场景列表。
- Practice 场景详情。
- 用户角色选择。
- 练习目标展示。
- 练习模式选择。
- 进入后续准备流程前的场景选择结果。

本 PR 不影响：

- 实时问答。
- 麦克风录音。
- 语音转写和语音播放。
- 练习中的回答提交。
- 评分、复盘和练习报告。
- 岗位 JD。
- 简历上传。
- 练习计划管理。

## 实际执行的验证

```bash
cd server
gofmt -w internal/coaching/scene/scene.go internal/coaching/scene/scene_test.go internal/coaching/http.go internal/coaching/http_test.go
go test ./...
go vet ./...
```

验证结果：

- `go test ./...`：通过。
- `go vet ./...`：通过。
- `git diff --check`：通过。

## 未执行的验证

当前环境未安装 Flutter 和 Dart SDK，以下命令未执行成功：

```bash
cd mobile
flutter pub get
dart format --output=none --set-exit-if-changed lib test
flutter analyze
flutter test
```

原因：

```text
flutter: command not found
dart: command not found
```

## 手工验证

已通过 Go HTTP Handler 测试验证：

- 场景列表接口。
- 场景详情接口。
- 角色列表接口。
- 不存在场景的 `404` 响应。

Flutter 页面尚未在模拟器或真机中进行手工验证。

## 已知限制

1. 服务端场景数据当前保存在内存中，尚未接入数据库。
2. 当前环境缺少 Flutter/Dart SDK，移动端格式化、静态检查和测试需要在具备 Flutter 工具链的环境中补充。
3. 本 PR 不创建练习会话，也不进入实时练习页面。
4. 当前仅实现今天要求的场景选择最小流程。

## 关联 Issue

关联 Issue：`#待填写`

## 分支信息

源头分支：

`feature/<issue-number>-scene-selection`

目标分支：

`main`

## AI 辅助说明

本次代码使用 AI Coding 工具辅助完成。开发者已检查修改范围、接口契约、状态流程和测试结果，确认本 PR 未实现实时语音、评分、复盘、岗位准备和练习计划功能。

## 提交前检查

- [x] 未修改 `daily/`、`prompts/`、`docs/` 课程记录目录。
- [x] 未提交密钥、Token、`.env`、缓存或构建产物。
- [x] 未创建远程 Commit、Issue 或 PR。
- [x] Go 格式化、测试和静态检查已通过。
- [ ] Flutter 格式化、静态检查和测试待安装 SDK 后执行。
- [ ] Flutter 模拟器或真机手工验收待完成。

> 直接生成pr的md文档给我

已生成 PR Markdown 文档：

[下载 pr-scene-selection.md](./outputs/pr-scene-selection.md)