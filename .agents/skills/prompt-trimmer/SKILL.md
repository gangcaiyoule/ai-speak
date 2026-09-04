---
name: prompt-trimmer
description: 直接截断 prompts 文本记录，使单文件或目录合计严格少于 10000 字；适用于队友传入提示词文件或 Dn 目录，不做 AI 摘要或内容分析。
---

# 提示词限字截断

使用仓库脚本 `scripts/trim_prompts.py` 处理队友传入的 `.md` 或 `.txt` 文件/目录。

## 规则

- 目标是严格少于 10000 字，脚本实际保留最多 9999 个字符。
- 采用从文件开头直接截断的轻量策略，不汇总、不调用 AI、不改写内容。
- 目录按相对路径排序，依次分配总预算；目录合计最多 9999 字。
- 默认只生成预览：单文件输出同目录的 `*.trimmed.*`，目录输出旁边的 `*.trimmed/`。
- 只有用户明确要求覆盖时才传 `--in-place`；覆盖前脚本会为每个原文件生成 `.bak` 备份。
- 只处理 `prompts/` 范围内的 `.md`、`.txt` 文件；路径不存在、编码错误或空目录时直接报告错误。

## 调用

```powershell
python scripts/trim_prompts.py prompts\24320106\D3\codex.md
python scripts/trim_prompts.py prompts\24320106\D3
python scripts/trim_prompts.py prompts\24320106\D3 --in-place
```

也可以把文件或目录拖到 `scripts\trim_prompts.cmd` 上。运行结束后报告压缩前后字符数、输出路径和是否生成备份；不要把预览结果误报为原文件已修改。
