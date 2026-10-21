#!/usr/bin/env python3
"""Kiểm kê tên trước D-RM-02 theo 6 nhóm A–F (docs/PROMPT-DOI-TEN-VA-LUONG-MOI.md §2.1), trên một revision git.

    python3 -I scripts/kiem-ke-doi-ten.py <revision> [--chi-tiet]

Đếm từng LẦN XUẤT HIỆN của mẫu (?i)eacl|enterprise[- ]ai[- ]control[- ]layer trong tệp git theo dõi ở revision đó
(git grep), và in thêm số DÒNG theo đúng lệnh gợi ý của prompt (bỏ docs/bao-cao/). Luật xếp nhóm:
- A  link git: lần khớp nằm trong chuỗi "enterprise-ai-control-layer" (tên repo = đường dẫn module Go, import, URL);
- FP khớp nhầm: chữ "eACL" (ACL viết hoa ngay sau một từ, ví dụ RetrieveACL) — không phải tên sản phẩm;
- E  bằng chứng lịch sử / tệp đã băm: tệp hoặc thư mục trong E_FILES / E_PREFIX, dòng chứa muối băm chia bộ giữ lại,
     mọi dòng của CHANGELOG.md trừ dòng mở đầu;
- C  định danh chạy trong mã (không phải docs/): lần khớp nằm trong biến môi trường, GUC, vai, database, khoá JSON,
     tên container, mã hệ G7;
- B  phần còn lại trong mã (không phải docs/, không phải .md);
- D  phần còn lại trong tài liệu;
- F  ngoài phạm vi: docs/DEBUG-LO-TRINH.md (tệp của phiên debug, không commit) — luôn 0 trên git.
Công cụ chỉ đọc, không sửa gì.
"""
import re
import subprocess
import sys
from collections import Counter, defaultdict

PAT = r"(?i)eacl|enterprise[- ]ai[- ]control[- ]layer"
GITLINK = "enterprise-ai-control-layer"
SALT = "eacl-holdout-2026-10-20"
E_FILES = {
    "docs/CHO-QUYET-DINH.md", "docs/PROMPT-DOI-TEN-VA-LUONG-MOI.md", "docs/PROMPT-AUDIT-ROADMAP.md", "docs/ROADMAP.md",
    "docs/g7/REVIEW-D5.md", "docs/g7/CONTAMINATION-AUDIT.md", "docs/g7/CHECKPOINT-A-H.md", "db/migrations/0001_win.sql",
}
E_PREFIX = ("docs/v3/", "docs/study/", "docs/g7/holdout/", "docs/g7/holdout-h3/", "docs/g7/holdout-h4/", "docs/bao-cao/")
RUNTIME = re.compile(
    r"EACL_[A-Z_]*|eacl\.tenant_id|eacl_app|eacl_giu_lai|eacl_unstable_cases|eacl-pg|eacl_test_|/eacl\?|POSTGRES_DB: eacl"
    r"|\"-d\", \"eacl|^name: eacl|-p eacl|sysEACL *= \"eacl\"|\"eacl\"")


def group(path, line, m):
    lo = line.lower()
    i = lo.find(GITLINK)
    while i >= 0:
        if i <= m.start() < i + len(GITLINK):
            return "A"
        i = lo.find(GITLINK, i + 1)
    if m.group(0) == "eACL":
        return "FP"
    if path in E_FILES or path.startswith(E_PREFIX) or SALT in line:
        return "E"
    if path == "CHANGELOG.md" and not line.startswith("Mọi thay đổi đáng kể"):
        return "E"
    if path.startswith("docs/") or path.endswith(".md"):
        return "D"
    if any(r.start() <= m.start() < r.end() for r in RUNTIME.finditer(line)):
        return "C"
    return "B"


def main():
    rev = sys.argv[1]
    out = subprocess.run(["git", "grep", "-I", "-n", "-i", "-E", PAT.removeprefix("(?i)"), rev],
                         capture_output=True, text=True, check=False).stdout
    counts, per_file, prompt_lines = Counter(), defaultdict(Counter), 0
    rx = re.compile(PAT)
    for raw in out.splitlines():
        _, path, _, line = raw.split(":", 3)
        if not path.startswith("docs/bao-cao/"):
            prompt_lines += 1
        for m in rx.finditer(line):
            g = group(path, line, m)
            counts[g] += 1
            per_file[path][g] += 1
    names = subprocess.run(["git", "ls-tree", "-r", "--name-only", rev], capture_output=True, text=True, check=True).stdout
    name_hits = [p for p in names.splitlines() if re.search(r"(?i)eacl", p)]
    groups = ["A", "B", "C", "D", "E", "F", "FP"]
    print(f"revision {rev}")
    print(f"lệnh gợi ý (số dòng khớp, bỏ docs/bao-cao/): {prompt_lines}")
    print(f"tổng lần xuất hiện (cả docs/bao-cao/): {sum(counts.values())}")
    for g in groups:
        print(f"  nhóm {g}: {counts[g]}")
    print(f"tên tệp có chữ eacl: {len(name_hits)} — {', '.join(name_hits)}")
    if "--chi-tiet" in sys.argv:
        for path in sorted(per_file):
            print(path + "\t" + " ".join(f"{g}={per_file[path][g]}" for g in groups if per_file[path][g]))


main()
