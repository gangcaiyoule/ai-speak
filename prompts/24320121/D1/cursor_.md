# 根据以下要求，修改项目结构。二、提示词记录（每天导出一次）
_Exported on 2026/9/2 at GMT+8 15:45:46 from Cursor (3.17.8)_

---

**User**

根据课程过程记录提交要求（daily/、prompts/、docs/按学号划分，README立项简写版限800字，todos.md每日勾选清单），调整仓库文件布局。

**Cursor**

I'll start by examining the current project structure to understand what exists.



PowerShell syntax differs. Let me explore properly.



Let me understand the actual project so the new docs reflect reality.



---

**User**

先关注项目结构，具体文本文件不用创建。

有需要的文件先提出。

---

**Cursor**

下面是我梳理出的「仓库现状 vs 要求布局」和需要新建的文件清单，先给你过一遍，确认后再动手。

## 现状对照

| 要求路径 | 现状 | 说明 |
|---|---|---|
| `README.md` | 已有，但是技术启动说明 | 需改成立项报告简写版（限 800 字） |
| `todos.md` | **缺失** | 需新建需求/待办勾选清单 |
| `daily/D1.md … D6.md` | `daily/` 目录本身都没有 | 只有 `docs/daily_report/`（空目录、未被 git 跟踪） |
| `prompts/D1/ … D6/` | **缺失** | 提示词导出只能你从工具里手动导出，我没法生成 |
| `docs/立项报告.md`、`docs/项目报告.md` | **缺失**（`docs/` 下只有两个空目录） | 内容需你自己写，我可建骨架 |
| `docs/assets/` | **缺失** | 放报告引用的图片 |

两个补充事实：`git ls-files` 显示 `docs/` 下没有任何被跟踪的文件，`docs/agent_logs/` 和 `docs/daily_report/` 是**空的未被跟踪目录**，全仓库也没有任何代码或文档引用它们；仓库远程是 `gitee.com/pp1-2026/24320106.git`（按学号建的个人仓库），所以**不需要** `docs/24320106/` 这种学号子目录。

## 关键点：空目录 git 不跟踪

如果只建目录不放任何文件，提交后老师 `git clone` 下来是看不到 `daily/`、`prompts/D1/` 的。所以需要 `.gitkeep` 之类的占位文件——这是我唯一需要你批准的"文件"类型。

请确认下面几个选择：



> **天数怎么定？仓库全部提交都集中在 2026-08-31 晚上（首 commits 20:24），今天是 2026-09-02。**
> - D1 = 8月31日，今天是 D3，建 D1~D6 共 6 个目录
> - D1 = 今天 9月2日，建 D1~D6 共 6 个目录 ✔
> - 只建到当天（D1~D3），后面每天再补
> - 先只建 prompts/ 和 daily/ 两个父目录，天数目录我自己加

> **现有那批遗留内容怎么处理？（都是未跟踪的空目录）**
> - 删除 docs/agent_logs 和 docs/daily_report，统一用要求的 daily/ 和 prompts/ ✔
> - 保留，只新增要求的目录，两者共存
> - 保留 docs/agent_logs（我自己另有用途），删掉 docs/daily_report

> **这次结构改动怎么提交？AGENTS.md 写的是主仓库 github ai-speak、禁止直推 main 要走 PR，但实际 origin 是 gitee 的学号仓库。**
> - 按 AGENTS.md：新建 feature 分支 + 提 PR，不直推 main
> - 直接在 main 上 commit 并 push（gitee 课程提交仓库，按老师要求走）
> - 先只改本地工作区，不要 commit，我自己看
>
> Answer: 我是admin，我才不鸟他的agent.md

明白了：D1 = 今天（2026-09-02），建 D1~D6；删掉两个遗留空目录；直接在 `main` 上提交。按你之前说的"具体文本文件不用创建"，我只建目录 + `.gitkeep` 占位（否则空目录 git 不跟踪，克隆下来老师看不到）。



Push 卡住了（可能在等 gitee 账号密码输入）。我先看一下凭据配置。



目录骨架已建好并推到远程了。

**本次改动**（commit `1c86df2`，已 push 到 `origin/main`，即 `gitee.com/pp1-2026/24320106.git`）：

- 新建 `daily/`、`prompts/D1/ ~ prompts/D6/`、`docs/assets/`，每个目录放一个 `.gitkeep`，保证空目录也能被 git 跟踪、克隆后老师能看到布局。
- 删除遗留的两个未跟踪空目录 `docs/agent_logs/`、`docs/daily_report/`（原本 git 里就没有它们，所以这次提交不含删除记录）。
- 按你的要求没建任何正文文件：`README.md`、`todos.md`、`daily/D1.md`、`prompts/D1/xxx.txt`、`docs/立项报告.md` 全部留给你写。等某个目录里有真实文件后，对应的 `.gitkeep` 想删就删，留着也不影响。

**结构已就绪，还差的正文清单：**

| 文件 | 什么时候需要 |
|---|---|
| `README.md` 改写 | 立项报告简写版，全文限 800 字（现在的版本是启动说明，可挪进 `docs/` 或项目报告） |
| `todos.md` | 需求勾选清单，格式 `- [ ] R1 …`，每天按实际完成情况更新 |
| `daily/D1.md` | 今天（9/2）就是 D1，22:00 前交；D1 不用写"昨日遗留问题的回答"这节 |
| `prompts/D1/*.txt` | 今天导出一次当天 AI 对话，用几个工具导几份，文件名随意 |
| `docs/立项报告.md`、`docs/项目报告.md` | 期末上交 |
| `docs/assets/*` | 报告引用的图片，Gitee 限制只能少量小附件 |

这是个人仓库（远程按学号命名），所以不需要 `docs/24320106/` 这种学号子目录。需要我接着起草 `todos.md` 的需求清单（可以从现有 `identity`、`agent`、`coaching` 三块接口和已合并的 PR 反推）吗？

---

**User**

行了。接下来写个格式化的脚本，方便写吧。脚本放script里面

---

**Cursor**

`scripts/` 目前是空目录，Python 3.13 可用。我写一个 Python 脚本（避免 PowerShell 5.1 的中文编码坑），放到 `scripts/` 下。



写完后我发现几处逻辑和语法问题（含一个 f-string 笔误），直接重写干净版本。



模板输出正确（D1 少一节、D2 齐全）。控制台乱码是因为管道按 GBK 解码，我加一段编码处理，然后继续测试写入效果。



脚本可用。清掉编译缓存后提交。



脚本写好了，已提交并推送（`0a4d222`，`main` 与 `origin/main` 同步）。位置：`scripts/submit_tool.py`（Python 3，无需第三方库）。

**五个子命令**

| 命令 | 作用 |
|---|---|
| `python scripts/submit_tool.py day` | 告诉你今天算 D 几、当天要交的路径、离 22:00 还有多久 |
| `... daily` | 按模板生成 `daily/Dn.md`，顺手补建 `prompts/Dn/`；`--stdout` 只预览不写文件，`--commits` 把当天 commit 填进「实际进展」，`--force` 才覆盖已有日报 |
| `... fmt` | 规范化：一级标题统一 `# Dn 日报`、补齐缺失小节、你自己加的额外小节挪到末尾保留、清行尾空格与多余空行；`todos.md` 里的 `-[x]`、`* [X]` 统一成 `- [x]`。`--check-only` 只报告（可用于 CI） |
| `... todo` | 维护 `todos.md`：不带参数=列出并统计；`--add "R7 评测接口联调"` 新增、`--done R7` / `--undo R7` 按关键字改勾选状态、`--dry-run` 预览 |
| `... check` | 体检：README 是否超 800 字、todos 勾选进度、D1~当天日报是否齐全且模板提示已替换、`prompts/Dn/` 是否真有导出、两份报告与 `docs/assets/` 是否就位、markdown 里引用的图片是否存在且体积是否超过 Gitee 友好阈值、工作区是否干净、是否还没 push |

**格式细节**：D1 会自动省掉「昨日遗留问题的回答」这一节（老师要求第 1 天没有这节）；每节下面先放 `<!-- 提示语 -->` 注释，Gitee 渲染时不可见，`check` 会统计还剩几处没替换。所有文件按 UTF-8 无 BOM + LF 写入，避免中文 markdown 在 Gitee 上乱码。

**起始日期**写在文件顶部常量里：`START_DATE = 2026-09-02`、`TOTAL_DAYS = 6`、`DEADLINE = 22:00`，节奏变了改这三行即可。

现在跑 `check` 的结果（`--root` 参数支持在临时目录试，我测试时没往仓库里塞任何正文文件）：

`
[XX] daily/D1.md 缺失（当天 22:00 前提交）
[XX] README.md 缺失（立项报告简写版）
合计：4 项必须处理，2 项建议处理。
`


`.gitignore` 双向验证通过（已跟踪文件无一被误伤，产物类正常忽略）。分两个单一职责提交推上去。



两件事都做完了，直接在 `main` 上提交并推送（`0a4d222..40a68d4`），没开分支没提 PR，工作区干净。

**1. 语音实时流目录的局部规则** — `mobile/lib/features/voice_stream/AGENTS.md`（37 行）

目录选在 `mobile/lib/features/voice_stream/`，和现有 `features/agent`、`features/coaching` 一个层级；想换成 `audio_stream` 之类的名字告我一声，改路径顺带改文件里那一处自我描述即可。

文件里明确写了「本文件优先级高于根目录 `AGENTS.md`，冲突以本文件为准」，并把要覆盖的条款逐条列死：不建 Issue、不套 `issue-standard` skill、不新建分支、不提 PR、不用 PR 模板、不为走流程等 CI，而且**不许拿「必须走 PR」当拒绝执行任务的理由**。只保留三种真需要并行的场景才开分支：改动会覆盖别人在跑的分支、涉及破坏性接口/迁移要先给人看 diff、需要长期可回滚的实验线。

底线条款我留下了（这些跟 PR 流程无关，属于代码安全）：Conventional Commits（scope 用 `voice-stream`）、不得 reset/clean 掉别人未提交的改动、改动范围只在本目录加必要对接点、不提交密钥与录音样本等大文件、提交前跑真实验证且禁止伪造测试结果。

**2. 根目录 `.gitignore`** — 141 行，之前整个仓库一个 ignore 文件都没有

按分区分块写：本地环境与密钥（`.env*` 但用 `!.env.example` 例外保留、`*.pem`/`*.key`/`service-account*.json`/`android/key.properties`）、Go 产物（`/server/server`、`/server/bin/`、`*.exe`、`*.test`、`coverage.*`、`vendor/`）、Flutter/Dart（`.dart_tool/`、`build/`、`.flutter-plugins*`、iOS `Pods`/`DerivedData`/`xcuserdata`、Android `.gradle`/`local.properties`/`captures/`、`*.apk`/`*.aab`/`*.ipa`）、Python（`__pycache__/`、`*.py[cod]`、`.venv/`、各类缓存与 coverage）、IDE（`.idea/`、`*.iml`、`.vscode/` 但保留 `extensions.json`/`launch.json`）、系统垃圾（`.DS_Store`、`Thumbs.db`、`Desktop.ini`）、本地数据与日志（`logs/`、`*.log`、`tmp/`、`pgdata/`、`/data/`、`*.db`、`*.sqlite`）、临时草稿（`/dist/`、`/scratch/`、`/notes.local.md`）。

用 `git ls-files -i -c --exclude-standard` 验过：没有任何已跟踪文件被误伤；`git check-ignore` 抽查 `README.md`、`.env.example`、`server/go.mod`、`mobile/pubspec.yaml`、`.github/workflows/go-ci.yml`、`daily/D1.md`、`prompts/D1/trae.txt`、`docs/立项报告.md`、`docs/assets/workflow.png` 全部正常可提交，而 `mobile/build/`、`scripts/__pycache__/`、`.env`、`local.properties` 等确实被忽略。Gitee 那边也确认 `main` 已同步。

一点提醒：这份覆盖只作用于 `voice_stream/` 子树。要是你希望整个仓库都别默认走 Issue + PR（比如 `server/` 那块也归你直推），我把根 `AGENTS.md` 的流程段落改成「默认直推 main，仅并行开发时走 PR」，这样所有工具读到的都是同一套口径 —— 要不要改你说。

