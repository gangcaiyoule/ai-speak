#!/usr/bin/env python3
"""直接截断提示词记录，使单文件或目录总字符数严格低于 10000。"""
from __future__ import annotations

import argparse
import shutil
import sys
from pathlib import Path

LIMIT = 10_000
TARGET = LIMIT - 1
TEXT_SUFFIXES = {".md", ".txt"}


def read_text(path: Path) -> str:
    try:
        return path.read_text(encoding="utf-8")
    except UnicodeDecodeError as exc:
        raise ValueError(f"不是有效的 UTF-8 文本：{path}") from exc


def files_for(path: Path) -> list[Path]:
    if path.is_file():
        if path.suffix.lower() not in TEXT_SUFFIXES:
            raise ValueError(f"只支持 .md 或 .txt 文件：{path}")
        return [path]
    if path.is_dir():
        files = sorted(
            (item for item in path.rglob("*") if item.is_file() and item.suffix.lower() in TEXT_SUFFIXES),
            key=lambda item: item.relative_to(path).as_posix().lower(),
        )
        if not files:
            raise ValueError(f"目录中没有 .md 或 .txt 文件：{path}")
        return files
    raise ValueError(f"路径不存在：{path}")


def output_path(source: Path, root: Path | None, in_place: bool) -> Path:
    if in_place:
        return source
    if root is None:
        return source.with_name(f"{source.stem}.trimmed{source.suffix}")
    destination_root = root.with_name(f"{root.name}.trimmed")
    return destination_root / source.relative_to(root)


def main(argv: list[str] | None = None) -> int:
    parser = argparse.ArgumentParser(description="直接截断提示词文件或目录到严格少于 10000 字")
    parser.add_argument("path", help="提示词文件或目录路径")
    parser.add_argument("--in-place", action="store_true", help="覆盖原文件，并先生成 .bak 备份")
    args = parser.parse_args(argv)

    source = Path(args.path).expanduser().resolve()
    try:
        files = files_for(source)
        root = source if source.is_dir() else None
        contents = [(item, read_text(item)) for item in files]
    except ValueError as exc:
        print(f"[错误] {exc}", file=sys.stderr)
        return 2

    remaining = TARGET
    results: list[tuple[Path, Path, int, int]] = []
    for item, content in contents:
        kept = content[: max(0, remaining)]
        remaining -= len(kept)
        destination = output_path(item, root, args.in_place)
        results.append((item, destination, len(content), len(kept)))

    if not args.in_place:
        for _source, destination, _old, _new in results:
            destination.parent.mkdir(parents=True, exist_ok=True)
    for item, destination, _old, _new in results:
        if args.in_place:
            backup = item.with_name(item.name + ".bak")
            shutil.copy2(item, backup)
        kept = next(content[:new] for (candidate, content), (_item, _dest, _old, new) in zip(contents, results) if candidate == item)
        destination.write_text(kept, encoding="utf-8", newline="")

    total_old = sum(old for _item, _dest, old, _new in results)
    total_new = sum(new for _item, _dest, _old, new in results)
    mode = "原地覆盖" if args.in_place else "预览输出"
    print(f"[完成] {mode}：{total_old} -> {total_new} 字符（上限 {LIMIT}，严格少于）")
    for item, destination, old, new in results:
        print(f"  {item}：{old} -> {new}，输出 {destination}")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
