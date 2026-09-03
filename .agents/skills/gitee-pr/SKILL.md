---
name: gitee-pr
description: 使用 Gitee Open API 创建、更新、关联和验证 Pull Request，适用于本仓库的分支合并流程。
---

# Gitee Pull Request 规范

本 Skill 只处理 Gitee Pull Request 的创建、更新、Issue 关联和远端验证；Issue 的范围、Milestone 和正文规范由 `issue-standard` 负责。

## 前置条件

- 主仓库是 `https://gitee.com/pp1-2026/24320106.git`，目标分支固定为 `main`。
- 开发分支必须从最新 `gitee/main` 创建，禁止直接推送 `main`。
- 创建 PR 前确认源分支已推送到 Gitee，且工作区没有无关改动。
- PAT 只从 Git Credential Manager 或安全环境读取，使用 `Authorization: token <PAT>` 请求头；不得打印、提交或写入 PR 内容。

## Gitee API 参数

写操作使用 `application/x-www-form-urlencoded`。Gitee Open API 的关键接口如下：

- 创建 PR：`POST /api/v5/repos/{owner}/{repo}/pulls`
  - 必填表单：`title`、`head`、`base`
  - 常用表单：`body`、`draft`、`milestone_number`、`close_related_issue`
  - `head` 是源分支名，`base` 是目标分支名；本仓库通常为 `base=main`。
- 更新 PR：`PATCH /api/v5/repos/{owner}/{repo}/pulls/{number}`
  - 可更新 `title`、`body`、`state`、`close_related_issue` 等字段。
- 读取 PR：`GET /api/v5/repos/{owner}/{repo}/pulls/{number}`。
- 读取 PR 关联 Issue：`GET /api/v5/repos/{owner}/{repo}/pulls/{number}/issues`。
- 读取 Issue 关联 PR：`GET /api/v5/repos/{owner}/issues/{number}/pull_requests`，仓库名通过查询参数 `repo` 传入。

## Issue 关联

创建 PR 时可以传 `issue`，但企业项目中该参数可能不稳定，尤其是 Gitee Issue 使用字符串编号时，不得只凭创建响应判断已关联。

可靠做法是：

1. PR 正文写入 `Closes #<Issue编号>`，例如 `Closes #IKCT51`。
2. 创建或更新 PR 时传 `close_related_issue=1`。
3. 通过 PR 的 `/issues` 端点和 Issue 的 `/pull_requests?repo=<repo>` 端点双向读取。
4. 两个列表都包含目标 Issue/PR 后，才报告“已关联”；若为空，应停止并说明 Gitee 企业项目关联未生效，不要猜测。

`Closes #<Issue编号>` 会在 PR 合并时请求自动关闭 Issue；如果只需要关联而不希望合并时关闭，应根据 Gitee 项目行为改用普通引用，并仍然通过双向 API 验证。

## 创建后验证

至少核对：

- PR 编号、标题、状态、源分支、目标分支和 URL；
- PR 变更文件仅属于当前 Issue 范围；
- Issue 关联已通过双向端点验证；
- CI、Review 和合并状态如实报告，不把“PR 已创建”说成“已完成”。
