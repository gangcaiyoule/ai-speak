# 2026-09-08 场景目录修复对话记录

## 用户
请帮我在这个 Go 项目里找一个范围小、容易验证的修复性改动。要求从最新 `origin/main` 开发，保留工作区里其他人的未提交文件；完成代码、测试和本地验证后只推送分支，Issue 和 PR 由我自己创建。顺便准备对应的 Issue、PR 文案和今天的日报。

## 模型
我检查了仓库，发现 `server/internal/coaching/scene/scene.go` 的 `cloneScene` 只复制了切片外层。`Roles` 里的 `PracticeObjectives`、`Prompt.TurnBlueprints` 和 `PracticeOptions.RoleDefinitionID` 仍然共享底层数据。调用方修改读取结果时，会污染目录内部状态。

我会从最新 `origin/main` 建立独立分支，给这些嵌套字段补充深拷贝，并增加测试验证 `GetScene`、`ListScenes`、`ListRoles` 的返回值互不影响。之后执行 `gofmt`、`go test ./...`、`go vet ./...` 和 `go build ./...`。

## 用户
这个问题的测试怎么证明真的修好了？

## 模型
测试先读取一个场景，再修改返回对象中的提示词、角色名称、练习目标和角色 ID；随后重新读取同一场景，并与全新目录实例的基线进行深度比较。如果两者一致，说明第一次读取返回的是独立数据，没有泄漏目录内部状态。角色列表接口也用同样方式检查。

## 用户
按这个方案实施，并把结果写进提示词和日报。

## 模型
已完成深拷贝修复和 `TestCatalogReadsDoNotExposeMutableState` 测试。服务端执行 `go test ./...`、`go vet ./...`、`go build ./...` 均通过，并已运行 `gofmt`。提交为 `23bbd4b fix(scene): protect catalog from mutable reads`，分支 `fix/scene-catalog-copy` 已推送。Issue 和 PR 文案使用 M3“缺陷修复”里程碑，编号留给你创建后填写。
