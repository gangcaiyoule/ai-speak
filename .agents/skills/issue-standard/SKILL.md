---
name: issue-standard
description: Create, split, edit, or audit GitHub Issues for this repository when work requires issue scope, acceptance criteria, title prefixes, duplicate checks, labels, or milestone assignment and verification.
---

# Issue Standard

Create one independently deliverable and verifiable Issue for each change. Inspect the target repository's current GitHub state before creating or changing anything, and preserve existing conventions unless this skill states a project requirement.

## Authorization And Scope

- Use this skill when the user asks to create, split, edit, or audit an Issue, or when an authorized repository change must begin with an Issue under `AGENTS.md`.
- Read-only questions, explanations, reviews, and diagnostics do not create an Issue unless the user also authorizes a change.
- Do not create branches, commits, Pull Requests, labels, Milestones, or releases as part of this skill unless separately authorized and governed by the repository workflow.
- Do not batch-edit unrelated or historical Issues.

## Inspect Before Creating

1. Confirm the target repository and its default branch.
2. Search open and closed Issues using task-specific keywords. Reuse an open Issue when its scope already matches; do not create a duplicate.
3. Inspect the repository's existing Issue style, labels, and all open Milestones.
4. Keep the Issue limited to one clear outcome. Split work when parts can be delivered or verified independently.

## Title

Use one Chinese type prefix followed by a concrete, verifiable description:

| Prefix | Use |
|---|---|
| `[功能]` | New user-facing or system capability |
| `[修复]` | Defect correction |
| `[重构]` | Internal restructuring without intended behavior change |
| `[文档]` | Documentation-only work |
| `[调研]` | Research, comparison, or proof of concept |
| `[杂项]` | CI, build, tooling, configuration, or repository maintenance |

Do not use vague titles such as "优化一下" or "完善功能". Keep the title focused on the observable result.

## Body

Use this structure unless an existing repository template requires more:

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

Only include requirements actually requested or supported by repository context. Do not invent labels, dependencies, test results, CI status, or implementation details.

## Milestone Selection

Every new change Issue must be assigned to a suitable open Milestone.

1. Query the remote repository for all open Milestones before creating the Issue.
2. If exactly one open Milestone clearly matches the task, select it.
3. If multiple Milestones exist, select one only when the task has an unambiguous match; otherwise ask the user.
4. If no open Milestone exists, or the only open Milestone clearly conflicts with the task, stop and ask the user. Do not silently omit the Milestone or create one without authorization.
5. Pass the exact Milestone title when creating the Issue, or assign it immediately afterward.

## Verify After Mutation

After creating or editing an Issue, read it back from GitHub and verify:

- the Issue number, title, state, and URL are correct;
- the body reflects the agreed scope and acceptance criteria;
- `milestone` is not null and its title matches the selected Milestone;
- any labels or dependencies actually requested are present;
- no duplicate Issue was created.

If verification fails, correct the scoped metadata and verify again. Report the Issue number, URL, and selected Milestone only after the remote state is confirmed.
