#!/usr/bin/env python3
# -*- coding: utf-8 -*-
"""课程提交材料辅助脚本：生成日报模板、规范化 markdown 格式、检查仓库布局。

常用命令（在仓库任意目录下执行均可）：

    python scripts/submit_tool.py day                 # 今天是第几天、要写哪些文件
    python scripts/submit_tool.py daily               # 生成当天 daily/Dn.md（先加 --stdout 预览）
    python scripts/submit_tool.py daily --commits     # 把当天 commit 填进「实际进展」
    python scripts/submit_tool.py fmt                 # 规范化 daily/*.md 与 todos.md 格式
    python scripts/submit_tool.py todo                # 列出需求与勾选状态
    python scripts/submit_tool.py todo --add "R7 评测接口联调"
    python scripts/submit_tool.py todo --done R7      # 勾掉已完成需求
    python scripts/submit_tool.py check               # 检查是否满足课程提交要求

说明：
  * 脚本只负责模板与格式，正文内容要你自己写；日报写自己项目里真实发生的事。
  * 所有文件统一按 UTF-8 无 BOM、LF 换行写入，避免 Gitee 上 markdown 显示异常。
  * 想换一天当 D1，改下面 START_DATE；想改总天数，改 TOTAL_DAYS。
"""
from __future__ import annotations

import argparse
import datetime as dt
import re
import subprocess
import sys
from pathlib import Path

REPO = Path(__file__).resolve().parent.parent
ROOT = REPO  # 可被 --root 覆盖，方便在临时目录里试

START_DATE = dt.date(2026, 9, 2)  # D1 对应的日期
TOTAL_DAYS = 6
README_LIMIT = 800  # README 字数上限（不计空白字符）
ASSET_WARN_KB = 300  # Gitee 对附件限制较严，超过就提醒压缩
DEADLINE = dt.time(22, 0)  # 每天 22:00 前提交

SECTIONS = [
    "昨日遗留问题的回答",
    "今日目标",
    "实际进展",
    "遇到的问题与解决",
    "明日计划",
]
HINTS = {
    "昨日遗留问题的回答": "系统每日反馈会给你留一个问题，次日回答；第 1 天没有这节",
    "今日目标": "早上写：今天打算完成什么",
    "实际进展": "晚上写：完成了什么，可粘贴关键 commit",
    "遇到的问题与解决": "卡在哪、试了什么、谁或什么工具帮到你了（老师同学、查资料、翻书、AI 都算）",
    "明日计划": "明天的最低目标",
}
FIRST_ONLY_OMIT = "昨日遗留问题的回答"  # D1 没有这一节

BOX_RE = re.compile(r"^(\s*)([-*+])\s*\[\s*([xX ]?)\s*\]\s*")
HEAD_RE = re.compile(r"^(#{1,6})\s*(.*?)\s*$")
IMG_RE = re.compile(r"!\[[^\]]*\]\(\s*([^)\s]+)[^)]*\)")


# ------------------------------------------------------------- 基础工具

def set_root(value: str | None) -> None:
    global ROOT
    if value:
        ROOT = Path(value).resolve()


def daily_dir() -> Path:
    return ROOT / "daily"


def prompts_dir() -> Path:
    return ROOT / "prompts"


def docs_dir() -> Path:
    return ROOT / "docs"


def todos_path() -> Path:
    return ROOT / "todos.md"


def readme_path() -> Path:
    return ROOT / "README.md"


def rel(path: Path) -> str:
    try:
        return path.resolve().relative_to(ROOT).as_posix()
    except ValueError:
        return path.as_posix()


def read_text(path: Path) -> str:
    return path.read_text(encoding="utf-8-sig").replace("\r\n", "\n").replace("\r", "\n")


def write_text(path: Path, text: str) -> None:
    path.parent.mkdir(parents=True, exist_ok=True)
    path.write_text(text, encoding="utf-8", newline="\n")


def run_git(args: list[str]) -> str:
    try:
        proc = subprocess.run(
            ["git", "-C", str(ROOT)] + args,
            capture_output=True,
            text=True,
            encoding="utf-8",
        )
    except OSError as exc:
        print(f"[!!] 调用 git 失败：{exc}", file=sys.stderr)
        return ""
    return proc.stdout.strip() if proc.returncode == 0 else ""


def day_to_date(n: int) -> dt.date:
    return START_DATE + dt.timedelta(days=n - 1)


def date_to_day(date: dt.date) -> int:
    return (date - START_DATE).days + 1


def today_day() -> int:
    return date_to_day(dt.date.today())


def valid_day(n: int) -> bool:
    return 1 <= n <= TOTAL_DAYS


def resolve_day(day: int | None, date: dt.date | None) -> int:
    if day:
        return day
    return date_to_day(date or dt.date.today())


def normalize_blank(lines: list[str]) -> list[str]:
    """去掉行尾空格、压缩连续空行、去掉首尾空行。"""
    cleaned = [line.rstrip() for line in lines]
    while cleaned and not cleaned[0]:
        cleaned.pop(0)
    while cleaned and not cleaned[-1]:
        cleaned.pop()
    out: list[str] = []
    for line in cleaned:
        if not line and out and not out[-1]:
            continue
        out.append(line)
    return out


def has_content(lines: list[str]) -> bool:
    """模板提示注释不算已填写。"""
    return any(line.strip() and not line.strip().startswith("<!--") for line in lines)


def count_chars(text: str) -> int:
    return len(re.sub(r"\s", "", text))


def prompt_files(n: int) -> list[Path]:
    """某天的提示词导出文件（.gitkeep 占位不算）。"""
    folder = prompts_dir() / f"D{n}"
    if not folder.is_dir():
        return []
    return sorted(p for p in folder.iterdir() if p.is_file() and p.name != ".gitkeep")


def day_commits(n: int) -> list[str]:
    date = day_to_date(n)
    out = run_git(
        [
            "log",
            "--no-merges",
            f"--since={date.isoformat()} 00:00:00",
            f"--until={date.isoformat()} 23:59:59",
            "--format=- `%h` %s",
        ]
    )
    return out.splitlines()


# ------------------------------------------------------------- 日报模板

def section_body(section: str, parsed: dict[str, list[str]], commits: list[str]) -> list[str]:
    body = normalize_blank(list(parsed.get(section, [])))
    if has_content(body):
        return body
    if section == "实际进展" and commits:
        return list(commits)
    return [f"<!-- {HINTS[section]} -->"]


def sections_for_day(n: int, parsed: dict[str, list[str]]) -> list[str]:
    result = []
    for section in SECTIONS:
        if n == 1 and section == FIRST_ONLY_OMIT and not has_content(parsed.get(section, [])):
            continue
        result.append(section)
    return result


def render_daily(
    n: int,
    parsed: dict[str, list[str]] | None = None,
    commits: list[str] | None = None,
    preamble: list[str] | None = None,
) -> str:
    parsed = parsed or {}
    commits = commits or []
    lines: list[str] = [f"# D{n} 日报", ""]
    extra = normalize_blank(preamble or [])
    if extra:
        lines += extra + [""]

    for section in sections_for_day(n, parsed):
        lines += [f"## {section}", ""] + section_body(section, parsed, commits) + [""]

    for section, body in parsed.items():  # 自定义小节保留，不丢内容
        if section in SECTIONS:
            continue
        lines += [f"## {section}", ""] + normalize_blank(body) + [""]

    return "\n".join(normalize_blank(lines)) + "\n"


def parse_daily(text: str) -> tuple[dict[str, list[str]], list[str]]:
    """返回 {二级标题: 正文行}（同名合并）与第一个小节前的零散行。"""
    sections: dict[str, list[str]] = {}
    preamble: list[str] = []
    current: list[str] | None = None

    for raw in text.split("\n"):
        match = HEAD_RE.match(raw)
        if match:
            level = len(match.group(1))
            head = match.group(2).strip()
            if level == 1:
                current = None
                continue
            if level == 2:
                current = sections.setdefault(head, [])
                continue
        if current is None:
            if raw.strip():
                preamble.append(raw)
            continue
        current.append(raw)
    return sections, preamble


def daily_number(path: Path) -> int | None:
    match = re.match(r"^D(\d+)\.md$", path.name, re.IGNORECASE)
    return int(match.group(1)) if match else None


def fmt_daily(path: Path, check_only: bool) -> tuple[list[str], bool]:
    """规范化单个日报，返回 (说明列表, 是否需要改动)。"""
    n = daily_number(path)
    if n is None:
        return [f"{rel(path)}: 文件名不是 Dn.md 格式，跳过"], False

    text = read_text(path)
    sections, preamble = parse_daily(text)
    rebuilt = render_daily(n, sections, [], preamble)

    notes: list[str] = []
    title = f"# D{n} 日报"
    first_line = text.split("\n", 1)[0].strip() if text.strip() else ""
    if first_line != title:
        notes.append(f"一级标题统一为 `{title}`")
    missing = [s for s in SECTIONS if not (n == 1 and s == FIRST_ONLY_OMIT) and s not in sections]
    if missing:
        notes.append("补上缺失小节 " + "、".join(f"## {s}" for s in missing))
    custom = [s for s in sections if s not in SECTIONS]
    if custom:
        notes.append("自定义小节保留在后面：" + "、".join(custom))
    if normalize_blank(text.split("\n")) != normalize_blank(rebuilt.split("\n")):
        notes.append("整理空行、行尾空格与小节顺序")

    changed = text != rebuilt
    if not notes and changed:
        notes.append("重排为标准模板结构")
    if changed and not check_only:
        write_text(path, rebuilt)
    return notes, changed


def fmt_markdown(path: Path, check_only: bool) -> tuple[list[str], bool]:
    """规范化普通 markdown（主要是 todos.md 的勾选符号）。"""
    text = read_text(path)
    lines = text.split("\n")
    fixed_count = 0
    for index, line in enumerate(lines):
        match = BOX_RE.match(line)
        if not match:
            continue
        indent, _bullet, mark = match.groups()
        state = "x" if mark.lower() == "x" else " "
        fixed = f"{indent}- [{state}] {line[match.end():].rstrip()}"
        if fixed != line:
            lines[index] = fixed
            fixed_count += 1
    rebuilt = "\n".join(normalize_blank(lines)) + "\n"
    changed = rebuilt != text
    if changed and not check_only:
        write_text(path, rebuilt)
    notes = []
    if fixed_count:
        notes.append(f"勾选符号规范了 {fixed_count} 行")
    if changed and not fixed_count:
        notes.append("整理了空行与行尾空格")
    return notes, changed


# ------------------------------------------------------------- 子命令

def cmd_day(args: argparse.Namespace) -> int:
    date = args.date or dt.date.today()
    n = date_to_day(date)
    print(f"日期 {date.isoformat()} 对应 D{n}（D1 = {START_DATE.isoformat()}，共 {TOTAL_DAYS} 天）")
    if not valid_day(n):
        print("[!!] 该日期不在本次 6 天范围内，需要的话改 START_DATE")
        return 1
    report = daily_dir() / f"D{n}.md"
    folder = prompts_dir() / f"D{n}"
    exported = prompt_files(n)
    print(f"  日报  {rel(report)}" + ("" if report.exists() else "  <- 还没建"))
    print(f"  提示词 {rel(folder)}/" + (f"  已导出 {len(exported)} 份" if exported else "  <- 还没有导出文件"))
    now = dt.datetime.now()
    if now.time() >= DEADLINE:
        print(f"[!!] 现在 {now:%H:%M}，已过当天 {DEADLINE:%H:%M} 的提交点")
    else:
        print(f"提醒：{DEADLINE:%H:%M} 前提交当天日报与提示词导出")
    return 0


def cmd_daily(args: argparse.Namespace) -> int:
    n = resolve_day(args.day, args.date)
    if not valid_day(n):
        print(f"[XX] D{n} 超出 1~{TOTAL_DAYS} 范围，用 --day 或 --date 重新指定")
        return 1

    target = daily_dir() / f"D{n}.md"
    if target.exists() and not args.force:
        print(f"[!!] {rel(target)} 已存在，要用模板重来请加 --force（会覆盖已写内容）")
        return 1

    commits = day_commits(n) if args.commits else []
    text = render_daily(n, {}, commits)
    if args.stdout:
        print(text, end="")
        return 0

    write_text(target, text)
    print(f"[OK] 已生成 {rel(target)}（{day_to_date(n).isoformat()}）")
    if commits:
        print(f"     已填入当天 {len(commits)} 条 commit，请在「实际进展」里补说明")

    folder = prompts_dir() / f"D{n}"
    if not folder.exists():
        folder.mkdir(parents=True)
        (folder / ".gitkeep").touch()
        print(f"[OK] 已补建 {rel(folder)}/")
    if not prompt_files(n):
        print(f"下一步：把当天 AI 对话导出成 txt 放进 {rel(folder)}/（用几个工具就导几份，文件名随意）")
    print("写完跑：python scripts/submit_tool.py fmt && python scripts/submit_tool.py check")
    return 0


def cmd_fmt(args: argparse.Namespace) -> int:
    if args.paths:
        targets = [Path(p) for p in args.paths]
    else:
        targets = sorted(daily_dir().glob("D*.md")) if daily_dir().is_dir() else []
        todos = todos_path()
        if todos.exists():
            targets.append(todos)

    if not targets:
        print("[!!] 没有找到 daily/Dn.md 或 todos.md，先在对应目录里写内容")
        return 1

    changed_any = False
    for path in targets:
        if not path.exists():
            print(f"[XX] 文件不存在：{rel(path)}")
            changed_any = True
            continue
        is_daily = daily_number(path) is not None and path.resolve().parent == daily_dir().resolve()
        notes, changed = (fmt_daily if is_daily else fmt_markdown)(path, args.check_only)
        changed_any = changed_any or changed
        if notes:
            for note in notes:
                print(f"{rel(path)}: {note}")
        else:
            print(f"{rel(path)}: 格式已符合模板")

    if args.check_only:
        if changed_any:
            print("[!!] 有不合规文件，运行不带 --check-only 的 fmt 可自动修正")
            return 1
        print("[OK] 全部符合格式")
    return 0


def cmd_todo(args: argparse.Namespace) -> int:
    path = todos_path()
    acting = args.add is not None or args.done is not None or args.undo is not None

    if not acting:
        if not path.exists():
            print(f"[XX] {rel(path)} 不存在，用 --add 新增条目会自动创建")
            return 1
        total = done = 0
        for line in read_text(path).split("\n"):
            match = BOX_RE.match(line)
            if not match:
                continue
            total += 1
            checked = match.group(3).lower() == "x"
            done += checked
            print(f"[{'x' if checked else ' '}] {line[match.end():]}")
        print(f"合计 {total} 项，已完成 {done} 项")
        return 0

    if not path.exists():
        if args.add is None:
            print(f"[XX] {rel(path)} 不存在，先用 --add 建立清单")
            return 1
        if not args.dry_run:
            write_text(path, "# 需求清单\n")
            print(f"[OK] 已创建 {rel(path)}")
        else:
            print(f"[OK] 将创建 {rel(path)}")

    lines = normalize_blank(read_text(path).split("\n")) if path.exists() else ["# 需求清单"]
    action_taken = False

    if args.add is not None:
        item = args.add.strip()
        if not re.match(r"^[Rr]\d+\s", item):
            print("[!!] 建议用编号格式，例如 `R7 评测接口联调`，方便和验收标准对应")
        if any(item in line for line in lines):
            print(f"[!!] 已有相同条目，未重复添加：{item}")
            return 1
        lines.append(f"- [ ] {item}")
        action_taken = True
        print(f"[OK] 新增待办：{item}")

    keyword = args.done if args.done is not None else args.undo
    if keyword:
        state = "x" if args.done is not None else " "
        hit = 0
        for index, line in enumerate(lines):
            match = BOX_RE.match(line)
            if not match or keyword.lower() not in line[match.end():].lower():
                continue
            lines[index] = f"{match.group(1)}- [{state}] {line[match.end():].rstrip()}"
            hit += 1
        if hit:
            action_taken = True
            print(f"[OK] {hit} 条状态改为 [{state}]")
        else:
            print(f"[!!] 没找到包含 “{keyword}” 的条目")
            return 1

    if not action_taken:
        return 0
    if args.dry_run:
        print("--- 预览（未写文件） ---")
        print("\n".join(normalize_blank(lines)))
        return 0
    write_text(path, "\n".join(lines) + "\n")
    fmt_markdown(path, False)  # 顺手规范勾选符号与空行
    print(f"[OK] 已更新 {rel(path)}")
    return 0


def cmd_check(args: argparse.Namespace) -> int:
    results: list[tuple[str, str]] = []
    ok = lambda msg: results.append(("OK", msg))
    warn = lambda msg: results.append(("!!", msg))
    bad = lambda msg: results.append(("XX", msg))

    n_today = today_day() if valid_day(today_day()) else TOTAL_DAYS
    days = list(range(1, n_today + 1))

    readme = readme_path()
    if not readme.exists():
        bad("README.md 缺失（要求：立项报告简写版，全文限 800 字）")
    else:
        length = count_chars(read_text(readme))
        if length > README_LIMIT:
            warn(f"README.md 有 {length} 字，比上限 {README_LIMIT} 多 {length - README_LIMIT} 字")
        elif length == 0:
            bad("README.md 是空文件")
        else:
            ok(f"README.md {length} 字（上限 {README_LIMIT}）")

    todos = todos_path()
    if not todos.exists():
        bad("todos.md 缺失，需求清单要按实际进度每天更新勾选")
    else:
        boxes = [m for m in (BOX_RE.match(line) for line in read_text(todos).split("\n")) if m]
        if not boxes:
            warn("todos.md 里还没有 `- [ ] R1 用户注册登录` 这类条目")
        else:
            done = sum(1 for m in boxes if m.group(3).lower() == "x")
            ok(f"todos.md 共 {len(boxes)} 项需求，已完成 {done} 项")

    for n in days:
        report = daily_dir() / f"D{n}.md"
        if not report.exists():
            bad(f"{rel(report)} 缺失（{day_to_date(n).isoformat()}，当天 22:00 前提交）")
            continue
        text = read_text(report)
        sections, _ = parse_daily(text)
        missing = [s for s in SECTIONS if not (n == 1 and s == FIRST_ONLY_OMIT) and s not in sections]
        hints = text.count("<!--")
        if missing:
            warn(f"{rel(report)} 缺小节：" + "、".join(missing))
        if hints:
            warn(f"{rel(report)} 还有 {hints} 处模板提示没替换")
        if not missing and not hints:
            ok(f"{rel(report)} 五个小节齐全")

    for n in days:
        folder = prompts_dir() / f"D{n}"
        if not folder.is_dir():
            bad(f"{rel(folder)}/ 缺失")
            continue
        files = prompt_files(n)
        if not files:
            warn(f"{rel(folder)}/ 还没有提示词导出（文本格式，别用截图）")
        else:
            ok(f"{rel(folder)}/ 已导出 {len(files)} 份：" + "、".join(p.name for p in files))

    kickoff = docs_dir() / "立项报告.md"
    final = docs_dir() / "项目报告.md"
    if not kickoff.exists():
        bad("docs/立项报告.md 缺失")
    elif count_chars(read_text(kickoff)) == 0:
        bad("docs/立项报告.md 是空文件")
    else:
        ok(f"docs/立项报告.md 已就位（{count_chars(read_text(kickoff))} 字）")
    if not final.exists():
        warn("docs/项目报告.md 缺失（期末上交，可先建骨架）")
    else:
        ok(f"docs/项目报告.md 已就位（{count_chars(read_text(final))} 字）")

    assets = docs_dir() / "assets"
    if not assets.is_dir():
        bad("docs/assets/ 缺失，报告引用的图片放这里")
    else:
        images = [p for p in assets.rglob("*") if p.is_file() and p.name != ".gitkeep"]
        ok(f"docs/assets/ 内有 {len(images)} 个附件")
        for image in images:
            size_kb = image.stat().st_size // 1024
            if size_kb > ASSET_WARN_KB:
                warn(f"{rel(image)} 有 {size_kb} KB，Gitee 只允许少量小附件，建议压缩")

    markdowns = sorted(docs_dir().rglob("*.md")) if docs_dir().is_dir() else []
    if readme.exists():
        markdowns.append(readme)
    for md in markdowns:
        for target in IMG_RE.findall(read_text(md)):
            if target.startswith(("http://", "https://", "data:")):
                continue
            resolved = (md.parent / target.replace("%20", " ")).resolve()
            if resolved.exists():
                ok(f"{rel(md)} 图片引用有效：{target}")
            else:
                bad(f"{rel(md)} 引用的图片不存在：{target}")

    if run_git(["rev-parse", "--git-dir"]):
        dirty = run_git(["status", "--short"])
        if dirty:
            warn("工作区有未提交改动：\n      " + "\n      ".join(dirty.splitlines()[:6]))
        else:
            ok("工作区干净")
        upstream = run_git(["rev-parse", "--abbrev-ref", "@{upstream}"])
        ahead = run_git(["rev-list", "--count", "@{u}..HEAD"]) if upstream else ""
        if not upstream:
            warn("当前分支没有上游分支，确认推送目标")
        elif ahead and ahead != "0":
            warn(f"本地比远端多 {ahead} 个 commit，记得 git push")
        else:
            ok(f"已与 {upstream} 同步")
        now = dt.datetime.now()
        if now.time() >= DEADLINE and (daily_dir() / f"D{n_today}.md").exists():
            text = read_text(daily_dir() / f"D{n_today}.md")
            if text.count("<!--"):
                warn(f"已过 {DEADLINE:%H:%M}，当天日报模板还没填完")
    else:
        warn("不在 git 仓库里，跳过提交状态检查")

    order = {"XX": 0, "!!": 1, "OK": 2}
    for level, msg in sorted(results, key=lambda item: (order[item[0]], item[1])):
        print(f"[{level}] {msg}")
    bad_count = sum(1 for level, _ in results if level == "XX")
    warn_count = sum(1 for level, _ in results if level == "!!")
    print(f"\n合计：{bad_count} 项必须处理，{warn_count} 项建议处理。")
    if args.strict and warn_count:
        return 1
    return 1 if bad_count else 0


# ------------------------------------------------------------- 入口

def build_parser() -> argparse.ArgumentParser:
    parser = argparse.ArgumentParser(
        prog="submit_tool.py",
        description="课程提交材料辅助脚本：日报模板 / 格式规范化 / 布局检查",
        formatter_class=argparse.RawDescriptionHelpFormatter,
        epilog="示例：python scripts/submit_tool.py daily --commits && python scripts/submit_tool.py check",
    )
    parser.add_argument("--root", help="仓库根目录，默认脚本所在仓库")
    sub = parser.add_subparsers(dest="command", required=True)

    p = sub.add_parser("day", help="显示今天是第几天、当天要交哪些文件")
    p.add_argument("--date", type=dt.date.fromisoformat, help="按指定日期换算，默认今天")
    p.set_defaults(func=cmd_day)

    p = sub.add_parser("daily", help="生成 daily/Dn.md 模板，并补建 prompts/Dn/")
    p.add_argument("--day", type=int, help="天数序号，默认按今天换算")
    p.add_argument("--date", type=dt.date.fromisoformat, help="按指定日期换算")
    p.add_argument("--commits", action="store_true", help="把当天 commit 填进「实际进展」")
    p.add_argument("--force", action="store_true", help="覆盖已有日报")
    p.add_argument("--stdout", action="store_true", help="只打印不写文件")
    p.set_defaults(func=cmd_daily)

    p = sub.add_parser("fmt", help="规范化日报小节结构与 todos.md 勾选格式")
    p.add_argument("paths", nargs="*", help="指定文件，默认 daily/D*.md 与 todos.md")
    p.add_argument("--check-only", action="store_true", help="只报告不写文件，有改动时退出码 1")
    p.set_defaults(func=cmd_fmt)

    p = sub.add_parser("todo", help="维护 todos.md 的需求与勾选状态")
    p.add_argument("--add", metavar="TEXT", help="新增一条未勾选需求")
    p.add_argument("--done", metavar="KEYWORD", help="把含关键字的条目标为已完成")
    p.add_argument("--undo", metavar="KEYWORD", help="把含关键字的条目改回未完成")
    p.add_argument("--dry-run", action="store_true", help="只预览不写文件")
    p.set_defaults(func=cmd_todo)

    p = sub.add_parser("check", help="检查仓库布局与当天材料是否齐备")
    p.add_argument("--strict", action="store_true", help="把建议项也算失败，可用于 CI")
    p.set_defaults(func=cmd_check)
    return parser


def main(argv: list[str] | None = None) -> int:
    # 交互式控制台由 Windows 用 UTF-16 API 输出；重定向到文件或管道时显式用 UTF-8，
    # 否则中文会被 GBK 编码，`> xxx.txt` 之后打开就是乱码。
    for stream in (sys.stdout, sys.stderr):
        if hasattr(stream, "reconfigure") and not stream.isatty():
            try:
                stream.reconfigure(encoding="utf-8")
            except (OSError, ValueError):
                pass
    args = build_parser().parse_args(argv)
    set_root(args.root)
    return args.func(args)


if __name__ == "__main__":
    sys.exit(main())
