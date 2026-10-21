#!/usr/bin/env python3
"""Kiểm toán nhiễm của người soạn / soát bộ giữ lại (ADR-0006 §10): quét nhật ký công cụ (JSONL) của các agent và
chỉ in ĐƯỜNG DẪN mà agent đã đọc / chạm nằm NGOÀI thư mục được phép. Không in nội dung lệnh, câu hỏi hay kỳ vọng,
vì nhật ký chứa chính các ca của bộ giữ lại mà phiên viết mã không được đọc.

    python3 -I scripts/kiem-nhiem-soan.py --cho-phep <thư mục> [--cho-phep …] --goc <thư mục gốc để giải đường tương đối> … <agent.jsonl> …

Mỗi lệnh gọi công cụ được xét:
- Read / Glob / Grep: đường dẫn trong tham số (Glob / Grep không có đường dẫn ⇒ "thư mục làm việc", tính là vi phạm);
- Bash: mọi từ trong lệnh trông như đường dẫn VÀ đang tồn tại trên đĩa (giải theo từng --goc); lệnh `cd`.
Một đường dẫn hợp lệ khi nằm trong một thư mục --cho-phep. Thư mục CHA của thư mục được phép ("/", gốc repo) chỉ là
vi phạm khi lệnh duyệt đệ quy (grep -r / -R, find, rg, tree, ls -R): "cd" về đó rồi đọc tệp thì tệp ấy tự hiện thành
đường dẫn vi phạm; còn lại in CẢNH BÁO kèm tên chương trình đầu lệnh. In: số lệnh theo công cụ, rồi từng đường dẫn vi
phạm (công cụ + đường dẫn). Thoát 0 khi không vi phạm, 1 khi có, 2 khi đầu vào hỏng.
"""
import argparse
import json
import os
import re
import shlex
import sys
from collections import Counter

PATH_TOKEN = re.compile(r"^[\w.~/-][\w.~/@+-]*$")


def tool_uses(path):
    """Trả (tên công cụ, tham số) của mọi lệnh gọi công cụ trong một nhật ký JSONL."""
    with open(path, encoding="utf-8") as f:
        for line in f:
            try:
                o = json.loads(line)
            except json.JSONDecodeError:
                continue
            content = (o.get("message") or {}).get("content")
            if not isinstance(content, list):
                continue
            for b in content:
                if isinstance(b, dict) and b.get("type") == "tool_use":
                    yield b.get("name", ""), b.get("input") or {}


def resolve(token, roots):
    """Trả các đường tuyệt đối đang tồn tại mà token trỏ tới (giải tương đối theo từng gốc)."""
    token = os.path.expanduser(token)
    if os.path.isabs(token):
        return [os.path.normpath(token)] if os.path.exists(token) else []
    return [os.path.normpath(os.path.join(r, token)) for r in roots if os.path.exists(os.path.join(r, token))]


def words(command):
    try:
        return shlex.split(command, comments=False, posix=True)
    except ValueError:
        return command.split()


RECURSIVE = re.compile(r"(^|\s)(grep|egrep|rg|find|tree|du)(\s|$)|\s-[a-zA-Z]*[rR][a-zA-Z]*(\s|$)")


def audit(path, allowed, roots):
    counts, bad, warn = Counter(), [], []

    def check(tool, p, command=""):
        if any(p == a or p.startswith(a + os.sep) for a in allowed):
            return
        if any(a.startswith(p.rstrip(os.sep) + os.sep) for a in allowed):  # thư mục cha của thư mục được phép
            prog = (words(command) or ["?"])[0]
            if RECURSIVE.search(command):
                bad.append((tool + " đệ quy: " + prog, p))
            else:
                warn.append((tool + ": " + prog, p))
            return
        bad.append((tool, p))

    for name, inp in tool_uses(path):
        counts[name] += 1
        if name == "Read":
            for p in resolve(str(inp.get("file_path", "")), roots) or [os.path.normpath(str(inp.get("file_path", "")))]:
                check(name, p)
        elif name in ("Glob", "Grep"):
            target = inp.get("path")
            if not target:
                bad.append((name, "<thư mục làm việc>"))
                continue
            for p in resolve(str(target), roots) or [os.path.normpath(str(target))]:
                check(name, p)
        elif name == "Bash":
            ws = words(str(inp.get("command", "")))
            for i, w in enumerate(ws):
                if w == "cd" and i + 1 < len(ws):
                    for p in resolve(ws[i + 1], roots) or [ws[i + 1]]:
                        check("Bash cd", p, "cd")
                if "/" in w and PATH_TOKEN.match(w):
                    for p in resolve(w, roots):
                        check(name, p, str(inp.get("command", "")))
    return counts, sorted(set(bad)), sorted(set(warn))


def main():
    ap = argparse.ArgumentParser()
    ap.add_argument("--cho-phep", action="append", required=True, help="thư mục được đọc (lặp lại được)")
    ap.add_argument("--goc", action="append", default=[], help="gốc để giải đường tương đối (lặp lại được)")
    ap.add_argument("logs", nargs="+")
    a = ap.parse_args()
    allowed = [os.path.normpath(os.path.abspath(x)) for x in a.cho_phep]
    roots = [os.path.abspath(x) for x in a.goc] or [os.getcwd()]
    total_bad = 0
    for log in a.logs:
        if not os.path.isfile(log):
            print(f"LỖI: không có {log}")
            return 2
        counts, bad, warn = audit(log, allowed, roots)
        print(f"{os.path.basename(log)}: " + ", ".join(f"{k} {v}" for k, v in sorted(counts.items())) + f" · vi phạm {len(bad)} · cảnh báo {len(warn)}")
        for tool, p in bad:
            print(f"  NGOÀI PHẠM VI [{tool}] {p}")
        for tool, p in warn:
            print(f"  CẢNH BÁO thư mục cha [{tool}] {p}")
        total_bad += len(bad)
    print(f"TỔNG vi phạm {total_bad}")
    return 1 if total_bad else 0


if __name__ == "__main__":
    sys.exit(main())
