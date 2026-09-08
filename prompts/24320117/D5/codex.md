用户：Goal: 在 Ticket 2 已合并的基础上，完成文本回答提交、问题推进和会话完成闭环。

前置条件：

- Ticket 2 的会话创建、激活和问题查询已经合并。
- 会话已经能够保存问题快照和当前问题。

完成标准：

- [ ] 只有 ACTIVE 会话可以提交文本回答。
- [ ] 回答去除首尾空白后不能为空。
- [ ] 回答保存到 practice\_turns。
- [ ] 提交回答时必须校验用户身份和当前 question\_id。
- [ ] 非当前问题、重复回答和越权访问会返回明确错误。
- [ ] 回答后当前问题能够推进到下一题。
- [ ] 最后一题回答后 current\_question\_id 变为空。
- [ ] 只有没有待回答问题的 ACTIVE 会话可以完成。
- [ ] 完成后会话状态变为 COMPLETED。
- [ ] COMPLETED 会话不能再次回答或重复完成。
- [ ] Flutter 练习页面可以展示当前问题并提交文本回答。
- [ ] Flutter 提交期间禁止重复点击，并能显示错误状态。
- [ ] Go 测试覆盖状态机、回答保存、问题推进、越权和重复操作。
- [ ] 执行 gofmt、go test ./...、go vet ./... 和 go build ./...。
- [ ] 输出变更文件、测试结果、未执行项和本地 Commit。

约束：

- 不实现 LLM、Agent、语音、ASR、TTS 或评测报告。
- 不实现历史页面和报告页面。
- 不创建或修改 Gitee Issue、PR、Milestone。
- 不 push、不 merge。
- 如果 Flutter SDK 不可用，必须如实记录。
  ai：
- 已完成文本回答提交、问题推进和会话完成闭环。

  主要变更：

  - 后端新增 `practice_turns` 持久化、当前问题推进和完成条件校验。
  - 支持回答为空、非当前问题、重复回答、越权、非 ACTIVE 会话等明确错误。
  - 最后一题回答后 `current_question_id` 为空，完成后状态变为 `COMPLETED`。
  - 新增 Flutter 练习页面、文本提交、加载/提交错误状态及防重复点击。
  - 更新 OpenAPI 与迁移文件 `0008_practice_turns.sql`。
  - 补充状态机、回答保存、推进、越权和重复操作测试。

  变更文件已提交，工作区干净：

  ```
  940a963 feat(coaching): submit practice text answers
  ```

  验证结果：

  - `gofmt`：通过
  - `go test ./...`：通过
  - `go vet ./...`：通过
  - `go build ./...`：通过
  - Flutter：未执行，当前环境不存在 `flutter` 命令
  - 未进行真实 PostgreSQL 实例端到端验证，仅完成 SQLMock 与 Go 单元测试

  本次未创建 Issue/PR，未 push、未 merge。
  用户：Goal: 在 Ticket 3 已合并的基础上，完成练习准备到文本会话完成的端到端验证和中期交付材料。

  完成标准：

  - [ ] PostgreSQL 可以通过 Docker Compose 启动。
  - [ ] 数据库迁移可以成功执行。
  - [ ] Go Server 可以使用 DATABASE\_URL 启动。
  - [ ] 登录用户可以完成场景选择、创建计划、创建会话、激活会话、提交文本回答和完成会话。
  - [ ] 计划、会话、问题和回答可以从 PostgreSQL 中查询到。
  - [ ] Go 测试、go vet 和 go build 全部执行并记录真实结果。
  - [ ] Flutter 相关检查在 SDK 可用时实际执行。
  - [ ] 启动文档包含数据库、迁移、服务端和客户端命令。
  - [ ] 中期演示材料包含成功链路、接口示例和已知限制。
  - [ ] D4 日报只记录实际完成内容和真实验证结果。
  - [ ] 输出最终变更清单、验证报告、未完成项和本地 Commit。

  约束：

  - 不新增 Agent、语音、动态评测和历史复盘范围。
  - 不创建或修改 Gitee Issue、PR、Milestone。
  - 不 push、不 merge。
  - 不伪造 CI、Review 或 Flutter 验证结果。
  - 发现前置 Ticket 的功能不完整时，报告阻塞，不偷偷修复下游以外的问题。
    ai：
  - 已完成本地中期交付材料，未创建/修改 Issue、PR、Milestone，未 push/merge。

    验证结果：

    - Go：`go test ./...`、`go vet ./...`、`go build ./...` 全部通过。
    - PostgreSQL：临时 PostgreSQL 18 环境迁移执行两次成功，计划、会话、问题、回答均成功落库。
    - `DATABASE_URL`：Go Server 启动成功。
    - E2E：注册、登录、场景选择、创建计划、创建/激活会话、提交文本回答、完成会话全部通过。
    - Docker Compose：未通过。Docker Desktop 因 `dockerInference` socket 路径错误启动失败。
    - Flutter：未执行，本机未安装 Flutter/Dart SDK。

    变更文件：

    - [README.md](C:/Users/yejunbo/Documents/Codex/2026-09-04/gitee-repo-practice-worktree/README.md)
    - [中期交付材料.md](C:/Users/yejunbo/Documents/Codex/2026-09-04/gitee-repo-practice-worktree/docs/24320106/中期交付材料.md)
    - [D4.md](C:/Users/yejunbo/Documents/Codex/2026-09-04/gitee-repo-practice-worktree/daily/24320106/D4.md)

    本地 Commit：

    ```
    51155bc docs(coaching): record text practice e2e verification
    ```

    工作区已干净。未完成项为 Docker Compose 环境修复和 Flutter SDK 可用后的客户端检查。