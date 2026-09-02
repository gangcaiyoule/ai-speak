---
name: issue-standard
description: 当任务需要确定 Issue 范围、验收标准、标题类型、重复检查、标签或 Milestone 关联与验证时，为本仓库创建、拆分、修改或检查 Gitee Issue。
---

# Issue 规范

为每项改动创建一个可以独立交付和验证的 Issue。在创建或修改 Issue 前，先检查目标仓库当前的 Gitee 状态；除非本 Skill 明确规定，否则沿用仓库现有约定。

## 授权与范围

- 当用户要求创建、拆分、修改或检查 Issue，或者根据 `AGENTS.md` 授权的仓库改动必须从 Issue 开始时，使用本 Skill。
- 用户只要求解释、Review、报告状态或只读诊断时，除非用户同时授权改动，否则不要创建 Issue。
- 本 Skill 不负责创建分支、Commit、Pull Request、标签、Milestone 或 Release；这些操作必须获得单独授权，并遵守仓库对应的工作流规则。
- 不要批量修改与当前任务无关的历史 Issue。

## 创建前检查

1. 确认目标 Gitee 仓库和默认分支。
2. 使用与任务相关的关键词搜索开放和已关闭的 Issue。如果已有开放 Issue 的范围完全匹配，应复用它，不要重复创建。
3. 检查仓库现有的 Issue 风格、标签和所有开放 Milestone。
4. 确保 Issue 只有一个清晰的结果；如果各部分可以独立交付或验证，应拆分为多个 Issue。

## Gitee API 要求

使用 Gitee Open API 时，PAT 放在 `Authorization: token <PAT>` 请求头中，不要写入仓库文件、命令输出或 Issue/PR 正文；写操作使用 `application/x-www-form-urlencoded`。

- 创建 Issue：`POST /api/v5/repos/{owner}/issues`。`owner` 在 URL 中，`repo` 在表单中；`title` 必填，`body` 和 `milestone` 按需传入。
- 更新 Issue：`PATCH /api/v5/repos/{owner}/issues/{number}`。同样必须在表单中传 `repo`；关闭时传 `state=closed`。
- 读取 Issue：`GET /api/v5/repos/{owner}/{repo}/issues/{number}`；企业项目 Issue 编号可能是字符串（例如 `IKCT51`），不要假设是数字。
- 读取 Milestone：`GET /api/v5/repos/{owner}/{repo}/milestones`；创建时传 Gitee Milestone 编号到 `milestone`，创建后必须重新读取确认。

创建或更新后不要只相信 HTTP 成功响应：必须用 Gitee API 重新读取 Issue，核对编号、标题、状态、正文和 `milestone`。

## 默认负责人

- 创建 Issue 时，如果用户没有明确指定负责人，默认将负责人设置为当前 Gitee 登录用户（即 Issue 创建者本人）。
- 如果用户明确指定了其他负责人，按用户指定值提交，不要擅自改回自己；无法解析负责人身份时先报告并停止写入。
- 创建或更新后必须从 Gitee 重新读取 `assignee`/负责人字段，确认负责人已设置为预期用户。页面显示“未设置”时，不能声称默认负责人已经生效。

Gitee 企业版写入负责人时，使用仓库所有者路径，并在表单中同时传仓库名和负责人登录名：

```text
PATCH /api/v5/repos/{owner}/issues/{number}
Content-Type: application/x-www-form-urlencoded

repo=<repo>
assignee=<gitee-login>
```

例如将 Issue `IKCW4K` 指派给当前成员 `AI0106`：

```powershell
$form = @{
  repo = '24320106'
  assignee = 'AI0106'
}
Invoke-RestMethod -Method Patch `
  -Uri 'https://gitee.com/api/v5/repos/pp1-2026/issues/IKCW4K' `
  -Headers @{ Authorization = "token <PAT>" } `
  -ContentType 'application/x-www-form-urlencoded' `
  -Body $form
```

创建 Issue 时也应在同一个表单中传 `assignee=<gitee-login>`；如果创建接口未生效，立即使用上述 `PATCH` 补设。`assignee` 使用 Gitee 登录名（如 `AI0106`），不是显示名或数字用户 ID。PAT 只能从安全凭据存储读取，不得写入脚本、仓库或输出。

## Markdown 正文编码

- `body` 必须使用真实换行符组成 Markdown，不得把 `` `n ``、`\\n` 或 HTML 转义文本当作换行直接提交。
- 使用 `application/x-www-form-urlencoded` 传递表单字段，并让 HTTP 客户端负责字段编码，不要手工拼接未经编码的 URL。
- 创建或更新后重新读取正文，确认标题、列表和段落正常分行，且不存在字面量 `` `n `` 或 `\\n` 残留；发现异常时立即修正并再次验证。
- PowerShell 可使用双引号 here-string（`@"..."@`）保留真实换行；不要使用包含字面量 `` `n `` 的单引号字符串作为 Markdown 正文。

## 标题

使用一个中文类型前缀，加上具体且可验证的描述：

| 前缀 | 使用场景 |
|---|---|
| `[功能]` | 新增面向用户或系统的能力 |
| `[修复]` | 修复缺陷 |
| `[重构]` | 预期不改变行为的内部结构调整 |
| `[文档]` | 仅修改文档 |
| `[调研]` | 调研、方案比较或概念验证 |
| `[杂项]` | CI、构建、工具、配置或仓库维护 |

不要使用“优化一下”或“完善功能”之类无法验收的标题。标题应聚焦于可以观察到的结果。

## 正文

除非仓库现有 Issue 模板要求更多内容，否则使用以下结构：

```markdown
## 背景

说明为什么现在需要这项工作，以及当前存在的问题。

## 范围

- 本 Issue 要完成的事项
- 明确不在本 Issue 中处理的事项

## 验收标准

- 可执行、可观察、可复核的完成条件
- 适用的测试、构建或手工验证要求

## 关联

- Milestone：<实际选择的 Milestone>
- 依赖：<Issue 链接或“无”>
```

只写用户实际要求或仓库上下文能够支持的内容。不要编造标签、依赖、测试结果、CI 状态或实现细节。

## Milestone 选择

每个新建的改动 Issue 都必须关联到合适的开放 Milestone。

1. 创建 Issue 前，查询目标仓库的所有开放 Milestone。
2. 如果只有一个开放 Milestone 且与任务明显匹配，选择它。
3. 如果有多个 Milestone，只有在任务与其中一个明确匹配时才能选择；无法判断时先询问用户。
4. 如果没有开放 Milestone，或者唯一的开放 Milestone 明显与任务冲突，停止并询问用户。不要静默省略 Milestone，也不要未经授权创建 Milestone。
5. 创建 Issue 时传入准确的 Milestone 标题；如果创建时未能设置，应立即补设。

## 修改后验证

创建或修改 Issue 后，从 Gitee 重新读取并确认：

- Issue 编号、标题、状态和 URL 正确；
- 正文符合已经确认的范围和验收标准；
- `milestone` 不为 `null`，并且标题与选定的 Milestone 一致；
- 用户要求的标签或依赖确实存在；
- 没有创建重复 Issue。

如果验证失败，先修正本次范围内的元数据，再重新验证。只有确认远程状态正确后，才能报告 Issue 编号、URL 和所选 Milestone。
