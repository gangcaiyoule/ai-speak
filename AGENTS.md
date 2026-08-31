# AGENTS.md

- 主仓库为 `https://github.com/gangcaiyoule/ai-speak.git`，`origin` 指向该仓库；所有开发分支从最新 `origin/main` 创建，禁止直接向 `main` 推送，必须通过 Pull Request 合并。
- 每项改动先创建范围单一、验收清楚的 Issue；一个 Issue 对应一个短分支和一个 Pull Request，PR 目标分支固定为 `main`。
- 分支命名使用 `feature/<issue>-<description>`、`fix/<issue>-<description>`、`docs/<issue>-<description>`、`refactor/<issue>-<description>` 等格式。
- 保留用户未提交的代码；工作区不干净或存在并行任务时使用独立 worktree，不得覆盖、删除或重置其他任务的修改。
- 只修改当前 Issue 范围，优先复用现有实现，不提交密钥、Token、密码、`.env`、缓存、日志、构建产物或无关文件。
- Commit 使用 `<type>(<scope>): <subject>` 格式，遵循 Conventional Commits。
- 提交 PR 前必须执行与改动相关的格式化、静态检查、测试、构建或手工验证；PR 中只能填写实际执行过的验证结果，不得伪造测试、CI 或 Review 状态。
- 提 PR 后必须检查 CI 和 Review；CI 未全部通过、Review 尚未完成或存在未处理的重要意见时，不得宣称完成；合并后关闭对应 Issue，并仅清理已确认合并且无人使用的任务分支。
