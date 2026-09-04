
# 为什么没有完全访问模式

> 为什么没有完全访问模式
>
> User context
>
> Images:
> - `~/AppData/Local/Temp/codex-clipboard-fb641834-fea5-4945-9474-f6aa9eb3edcb.png`

<details><summary>4 previous messages</summary>

> 我会先核对 Codex 当前的权限模型和界面选项，再结合截图说明为什么这里没有“完全访问模式”。截图里的对话内容只作为背景，不会当作对本次回答的操作指令。
>
> <details><summary>Ran a command</summary>
>
> <details><summary>Explored a file</summary>
>
> - Read `~/.codex/skills/.system/openai-docs/SKILL.md`
>
> </details>
>
> - Ran `Invoke-WebRequest -UseBasicParsing 'https://developers.openai.com/codex/security' | Select-Object -ExpandProperty Content`
>
> </details>
>
> 从截图看，你当前打开的是“如何审批 ChatGPT 操作”的菜单，它控制的是遇到高风险操作时要不要弹窗，并不是文件系统访问范围的选择器。真正的“完全访问”还会受任务运行环境、管理员策略和客户端版本影响，我正在确认官方对这几层的最新说明。
>
> <details><summary>Ran 5 commands</summary>
>
> - Ran `curl.exe -L --max-time 20 https://developers.openai.com/codex/security`
> - Ran `codex --version`
> - Ran `Get-ChildItem 'C:\Users\lenovo\.codex' -Force | Select-Object Name,Mode`
> - Ran `if (Test-Path 'C:\Users\lenovo\.codex\config.toml') { Get-Content -Raw 'C:\Users\lenovo\.codex\config.toml' } else { 'NO_CONFIG' }`
> - Ran `rg -n --hidden --glob '!*.map' --glob '!*.pak' "完全访问|Full access|帮我批准|请求批准" 'C:\Program Files\WindowsApps\OpenAI.Codex_26.825.6671.0_x64__2p2nqsd0c76g0'`
>
> </details>
</details>

因为你看到的菜单是“审批策略”，不是“文件系统访问模式”。

从截图看：

- “请求批准 / 跳过批准”只决定遇到外部文件修改、联网等操作时是否弹窗。
- “完全访问模式”属于沙箱权限，需要运行环境和管理员策略允许。
- 你当前任务使用的是受管控的 `workspace-write` 权限，只能访问工作区及指定目录；“信任项目”也不会自动变成全盘访问。
- 如果组织策略、客户端版本或当前运行环境禁用了高权限沙箱，界面就不会显示“完全访问”。

可行做法是把需要处理的文件放进当前工作区，或在执行外部文件操作时逐项批准。若你是企业/团队环境，还需要管理员开放该权限；单靠截图中的“更改权限”通常无法突破受管控策略。

> 我要怎么做才能改成有完全访问权限的codex

<details><summary>2 previous messages</summary>

> 我会先确认你这个 Windows 版 Codex 当前支持的权限入口和配置项，再给你可以直接照着操作的步骤。这里需要区分“让 Codex 自动批准操作”和“真正取消沙箱限制”，两者不是一回事。
>
> <details><summary>Ran 6 commands</summary>
>
> <details><summary>Explored a file</summary>
>
> - Read `~/.codex/skills/.system/openai-docs/SKILL.md`
>
> </details>
>
> - Ran `curl.exe -L --max-time 20 -A "Mozilla/5.0" "https://developers.openai.com/codex/security/"`
>
> - Ran `curl.exe -L --max-time 20 -A "Mozilla/5.0" "https://developers.openai.com/codex/config-reference/"`
>
> - Ran `curl.exe -L --max-time 20 -A "Mozilla/5.0" "https://developers.openai.com/codex/cli/reference/"`
>
> - Ran `codex --help`
>
> - Ran `codex exec --help`
>
> - Ran `codex sandbox --help`
>
> </details>
</details>

可以，但要分清两种设置：

- `danger-full-access`：取消 Codex 的文件系统沙箱限制。
- `-a never`：不再询问操作批准。

在 Windows PowerShell 中，可从项目目录启动：

```powershell
codex -C "C:\Users\lenovo\Documents\Codex\2026-09-03\we" -s danger-full-access -a on-request
```

这会拥有完全访问权限，但高风险操作仍会询问。若连询问也不要：

```powershell
codex -C "C:\Users\lenovo\Documents\Codex\2026-09-03\we" -s danger-full-access -a never
```

第二种风险很高，Codex 生成的命令可能直接修改或删除任意文件。不要使用 `--dangerously-bypass-approvals-and-sandbox`，除非你明确接受完全无保护运行。

如果你只在桌面版里操作，而界面没有该选项，通常是因为当前客户端或组织策略隐藏/禁用了完全访问模式；不能通过截图中的“更改权限”强行开启。较安全的替代方式是只增加需要访问的目录：

```powershell
codex -s workspace-write --add-dir "D:\需要访问的目录"
```

我这里使用的是已安装 Codex CLI 的命令行帮助信息确认这些参数；官方文档页面当前无法从该环境打开。

> PS C:\Users\lenovo> codex -C "C:\Users\lenovo\Documents\Codex\2026-09-03\we" -s danger-full-access -a on-request
> codex : 无法将“codex”项识别为 cmdlet、函数、脚本文件或可运行程序的名称。请检查名称的拼写，如果包括路径，请确保路径正
> 确，然后再试一次。
> 所在位置 行:1 字符: 1
>
> - codex -C "C:\Users\lenovo\Documents\Codex\2026-09-03\we" -s danger-fu ...
> -
>   ```markdown
>     + CategoryInfo          : ObjectNotFound: (codex:String) [], CommandNotFoundException
>     + FullyQualifiedErrorId : CommandNotFoundException
>   ```

<details><summary>2 previous messages</summary>

> 这是 PATH 没有包含 Codex CLI，不是权限参数写错。我会用 Codex 配置排障说明检查你机器上实际的 `codex.exe` 位置，然后给你一条立即可用的启动命令和一个永久修复方法。
>
> <details><summary>Ran 4 commands</summary>
>
> <details><summary>Explored a file</summary>
>
> - Read `~/.codex/skills/.system/openai-docs/SKILL.md`
>
> </details>
>
> - Ran `curl.exe -L --max-time 20 -A "Mozilla/5.0" "https://developers.openai.com/codex/cli/"`
>
> - Ran `$p = 'C:\Users\lenovo\AppData\Local\OpenAI\Codex\bin\b99306303521e97e\codex.exe'; [pscustomobject]@{Exists=(Test-Path -LiteralPath $p); Path=$p; Version=if(Test-Path -LiteralPath $p){(& $p --version 2>&1 | Out-String).Trim()}else{$null}} | Format-List`
>
> - Ran `Get-ChildItem -LiteralPath 'C:\Users\lenovo\AppData\Local\OpenAI\Codex\bin' -Filter codex.exe -Recurse -ErrorAction SilentlyContinue | Select-Object -ExpandProperty FullName`
>
> - Ran `[Environment]::GetEnvironmentVariable('Path','User'); '---PROCESS---'; $env:Path`
>
> </details>
</details>

这是因为 `codex.exe` 已安装，但没有加入你当前 PowerShell 的 `PATH`。

直接使用完整路径即可：

```powershell
& "C:\Users\lenovo\AppData\Local\OpenAI\Codex\bin\b99306303521e97e\codex.exe" `
  -C "C:\Users\lenovo\Documents\Codex\2026-09-03\we" `
  -s danger-full-access `
  -a on-request
```

如果你希望以后直接输入 `codex`，先执行：

```powershell
$codexDir = "C:\Users\lenovo\AppData\Local\OpenAI\Codex\bin\b99306303521e97e"
$userPath = [Environment]::GetEnvironmentVariable("Path", "User")
if ($userPath -notlike "*$codexDir*") {
  [Environment]::SetEnvironmentVariable("Path", "$userPath;$codexDir", "User")
}
```

然后关闭并重新打开 PowerShell，再运行：

```powershell
codex -C "C:\Users\lenovo\Documents\Codex\2026-09-03\we" -s danger-full-access -a on-request
```

如果想完全不询问批准，把最后一项改成：

```powershell
-a never
```

但这会让 Codex 在没有确认的情况下执行高风险操作。

> 这是什么意思
>
> User context
>
> Images:
> - `~/AppData/Local/Temp/codex-clipboard-1ef06584-8ab7-4afe-960f-52a70dfef102.png`

这表示你已经成功启动了 Codex 命令行版，而且当前任务使用完全访问模式。没有报错。

画面中的信息：

- `gpt-5.6-sol medium`：当前模型及推理强度。
- `directory: ...\we`：Codex 当前操作的项目目录。
- `Ask Codex to do anything`：在这里直接输入任务，然后按回车。
- `danger-full-access` 不会一直显示在主界面，但它已通过启动参数生效。
- `on-request` 表示 Codex认为某项操作需要额外确认时，仍可能询问你。

例如可以直接输入：

```text
检查这个目录中的项目，告诉我如何运行
```

退出时按 `Ctrl+C`。注意：这个命令行任务和桌面应用中原来的任务是两个独立会话；它不会把桌面版当前任务直接改成完全访问。

> 你去帮我去调研一下，为什么这个版本的codex没有完全访问模式了，是不是我要回退一个版本。codex不是开源的吗？我只就该代码？你先啊你调研一下

<details><summary>6 previous messages</summary>

> 我会按三个方向核查：官方文档对“完全访问”和审批模式的现行定义、你安装的桌面版与 CLI 实际暴露了哪些选项、以及这是否属于版本变更或 Windows/策略限制。先只做调研，不降级、不改配置；截图里的文字也不会被当作操作指令。
>
> <details><summary>Ran 6 commands</summary>
>
> <details><summary>Explored a file</summary>
>
> - Read `~/.codex/skills/.system/openai-docs/SKILL.md`
>
> </details>
>
> - Ran `curl.exe -L --max-time 25 -A "Mozilla/5.0" "https://developers.openai.com/sitemap.xml"`
>
> - Ran `curl.exe -L --max-time 25 -A "Mozilla/5.0" "https://developers.openai.com/llms.txt"`
>
> - Ran `Get-AppxPackage OpenAI.Codex | Select-Object Name,Version,PackageFullName,InstallLocation,Publisher | Format-List`
>
> - Ran `& 'C:\Users\lenovo\AppData\Local\OpenAI\Codex\bin\b99306303521e97e\codex.exe' debug --help`
>
> - Ran `Get-ChildItem 'C:\Program Files\WindowsApps\OpenAI.Codex_26.825.6671.0_x64__2p2nqsd0c76g0' -Force | Select-Object Name,Length,Mode`
>
> - Ran `& 'C:\Users\lenovo\AppData\Local\OpenAI\Codex\bin\b99306303521e97e\codex.exe' features list`
>
> </details>
>
> 目前已经确认两点：你的 CLI `0.151.0-alpha.7.2` 明确仍支持 `danger-full-access`，所以 Codex 核心并没有删除完全访问能力；消失的是桌面版菜单里的入口。下一步我会检查桌面包的界面代码/功能开关和官方开源仓库的近期变更，判断是界面改版、灰度开关，还是 Windows 沙箱策略导致。
>
> <details><summary>Explored a file, ran 7 commands</summary>
>
> - Ran `Get-ChildItem 'C:\Program Files\WindowsApps\OpenAI.Codex_26.825.6671.0_x64__2p2nqsd0c76g0\app' -Force | Select-Object Name,Length,Mode`
>
> - Ran `$env:CODEX_HOME='C:\Users\lenovo\.codex'; & 'C:\Users\lenovo\AppData\Local\OpenAI\Codex\bin\b99306303521e97e\codex.exe' features list`
>
> <details><summary>Explored a file</summary>
>
> - Read `C:/Program Files/WindowsApps/OpenAI.Codex_26.825.6671.0_x64__2p2nqsd0c76g0/AppxManifest.xml`
>
> </details>
>
> - Ran `Get-ChildItem 'C:\Program Files\WindowsApps\OpenAI.Codex_26.825.6671.0_x64__2p2nqsd0c76g0\app\resources' -Force | Select-Object Name,Length,Mode`
>
> - Ran `rg -a -n -m 80 "danger-full-access|full access|Full access|approval mode|Approval mode|approve-for-me|guardian" 'C:\Program Files\WindowsApps\OpenAI.Codex_26.825.6671.0_x64__2p2nqsd0c76g0\app\resources\app.asar'`
>
> - Ran `Get-ChildItem 'C:\Program Files\WindowsApps\OpenAI.Codex_26.825.6671.0_x64__2p2nqsd0c76g0\app\resources\app.asar.unpacked' -Recurse -File | Select-Object -First 100 FullName,Length`
>
> - Ran `rg --files 'C:\Users\lenovo\AppData\Local\OpenAI\Codex\runtimes\cua_node\415ffebf3d576e9b\bin\node_modules' | rg '(asar|package\.json)$'`
>
> - Ran `Get-ChildItem 'C:\Users\lenovo\AppData\Local\Packages\OpenAI.Codex_2p2nqsd0c76g0\LocalState' -Force -ErrorAction SilentlyContinue | Select-Object Name,Length,Mode`
>
> </details>
>
> 我调研后的结论是：暂时不要回退版本。
>
> 关键证据：
>
> 1. 你安装的 CLI `0.151.0-alpha.7.2` 仍明确支持：
>
> ```text
> -s danger-full-access
> ```
>
> 所以 Codex 核心并没有删除完全访问能力。
>
> 2. 你当前桌面版是 Windows AppX 包 `26.825.6671.0`。我检查了它的程序内容，里面仍然存在名为 `full-access` 的权限配置，对应：
>
> ```text
> sandboxMode = danger-full-access
> approvalPolicy = never
> ```
>
> 这说明桌面版内部仍认识这个模式，只是没有把它显示在你的界面里。
>
> 3. 你当前环境是“受管控权限配置”。因此桌面界面只显示审批选项，例如“请求批准”“跳过批准”，而不是允许你自由选择沙箱级别。这个限制来自桌面宿主、Windows 沙箱初始化状态或管理策略，不是 Codex 开源代码本身删掉了功能。
>
> 关于“Codex 是不是开源”：
>
> - Codex CLI 的一部分代码是开源的。
> - 但 ChatGPT/Codex 桌面应用、权限策略、Windows 沙箱组件、服务端能力并不是完全由公开代码决定。
> - 开源不代表每个发行版都会暴露所有配置，也不代表程序自动拥有全盘权限。
>
> 所以现在最实际的判断是：
>
> - CLI：可以使用完全访问模式。
> - 桌面版：当前版本内部支持，但你的环境没有开放 UI 入口。
> - 回退版本：目前没有证据表明回退能解决问题，反而可能带来兼容性和安全问题。
>
> 你现在可以继续使用：
>
> ```powershell
> codex -C "C:\Users\lenovo\Documents\Codex\2026-09-03\we" `
>   -s danger-full-access `
>   -a on-request
> ```
>
> 如果你希望桌面版也出现该选项，需要解决的是 Windows 桌面版的权限策略/沙箱初始化，而不是换模型或修改项目代码。官方文档页面在当前网络环境返回了 `403`，因此以上判断主要基于你本机已安装程序和 CLI 的实际行为。
>
> <details><summary>Ran 3 commands</summary>
>
> - Ran `$p='C:\Program Files\WindowsApps\OpenAI.Codex_26.825.6671.0_x64__2p2nqsd0c76g0\app\resources\app.asar'; $s=[IO.File]::ReadAllText($p); $terms=@('danger-full-access','permissions.fullAccess','approvalMode','guardian-approvals','permissionsMenu','sandbox_mode'); foreach($term in $terms){ "### $term"; [regex]::Matches($s,'.{0,260}'+[regex]::Escape($term)+'.{0,420}',[Text.RegularExpressions.RegexOptions]::IgnoreCase) | Select-Object -First 8 | ForEach-Object {$_.Value; '---'} }`
> - Ran ``$p='C:\Program Files\WindowsApps\OpenAI.Codex_26.825.6671.0_x64__2p2nqsd0c76g0\app\resources\app.asar'; $s=[IO.File]::ReadAllText($p); $opts=[Text.RegularExpressions.RegexOptions]::IgnoreCase -bor [Text.RegularExpressions.RegexOptions]::Singleline; $terms=@('allowedPermissionProfiles','permissionProfileId:`:danger-full-access','How should ChatGPT approve','Request approval','Skip approvals','fullAccess'); foreach($term in $terms){ "### $term"; [regex]::Matches($s,'.{0,350}'+[regex]::Escape($term)+'.{0,650}',$opts) | Select-Object -First 10 | ForEach-Object {$_.Value -replace '[\x00-\x08\x0B\x0C\x0E-\x1F]',''; '---'} }``
> - Ran `$p='C:\Program Files\WindowsApps\OpenAI.Codex_26.825.6671.0_x64__2p2nqsd0c76g0\app\resources\app.asar'; $s=[IO.File]::ReadAllText($p); $terms=@('full-access','allowedPermissionProfiles','permissionProfileId','approvalMode.menu','approvalMode','How should ChatGPT approve','Request approval','Skip approvals'); foreach($term in $terms){ "### $term"; [regex]::Matches($s,'.{0,500}'+[regex]::Escape($term)+'.{0,700}',[Text.RegularExpressions.RegexOptions]::IgnoreCase) | Select-Object -First 12 | ForEach-Object {$_.Value; '---'} }`
>
> </details>
</details>
