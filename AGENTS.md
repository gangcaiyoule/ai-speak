# AGENTS.md

本项目的仓库工作流 Codex skills 来源为 `.agents/skills/`。只加载与当前任务直接相关的 skill；系统级或用户级通用 skill 不受此来源限制。

## Skills

| Skill | 用途 |
|---|---|
| `issue-standard` | 创建、拆分、修改或检查 Gitee Issue，包括标题类型、范围、验收标准、重复检查和 Milestone 关联 |

## Skill 使用原则

- 创建、拆分、修改或检查 Gitee Issue 时，必须使用 `issue-standard`。
- 用户只要求解释、Review、报告状态或只读诊断时，不因这些任务自动创建 Issue。
- skill 负责具体操作流程；下列仓库协作规则始终有效。

## 仓库协作规则

- 主仓库为 `https://gitee.com/pp1-2026/24320106.git`，`gitee` 指向该仓库；所有开发分支从最新 `gitee/main` 创建，禁止直接向 `main` 推送，必须通过 Gitee Pull Request 合并。
- 每项改动先创建范围单一、验收清楚且关联当前 Milestone 的 Issue；一个 Issue 对应一个短分支和一个 Pull Request，PR 目标分支固定为 `main`。
- 分支命名使用 `feature/<issue>-<description>`、`fix/<issue>-<description>`、`docs/<issue>-<description>`、`refactor/<issue>-<description>` 等格式。
- 保留用户未提交的代码；工作区不干净或存在并行任务时使用独立 worktree，不得覆盖、删除或重置其他任务的修改。
- 只修改当前 Issue 范围，优先复用现有实现，不提交密钥、Token、密码、`.env`、缓存、日志、构建产物或无关文件。
- Commit 使用 `<type>(<scope>): <subject>` 格式，遵循 Conventional Commits。
- 提交 PR 前必须执行与改动相关的格式化、静态检查、测试、构建或手工验证；PR 中只能填写实际执行过的验证结果，不得伪造测试、CI 或 Review 状态。
- 提 PR 后必须检查 CI 和 Review；CI 未全部通过、Review 尚未完成或存在未处理的重要意见时，不得宣称完成；合并后关闭对应 Issue，并仅清理已确认合并且无人使用的任务分支。
