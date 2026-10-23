#!/usr/bin/env python3
"""Sinh đột biến cho phần GIẢI THÍCH của vết (R2-03) từ chính mã hiện tại, để mọi dòng giải thích đều có test khoá.

Chạy ở gốc repo: `python3 -I scripts/dot-bien-vet-sinh.py`. In ra các dòng `m <tệp> <chuỗi gốc> <chuỗi đột biến> <nhãn>` (đã bọc nháy cho shell)
mà scripts/dot-bien-vet.sh nạp bằng `eval`. Vì sinh từ mã hiện tại nên chuỗi gốc luôn khớp đúng một lần: mã đổi thì danh mục tự theo, không phải sửa tay.

Ba họ đột biến, mỗi họ trả lời một câu hỏi "vết có thể nói SAI mà test vẫn xanh không?":
  NV  cộng 1 vào MỘT số được in trong dòng giải thích (len(...), ưu tiên, số lần…): số in ra có được khoá nguyên chữ không;
  QV  bỏ contracts.Quote khỏi MỘT chỗ nhúng chữ của người hỏi: chữ có luôn nằm trong cặp “…” (để Why.WithholdQuoted che được) không;
  LV  bỏ hẳn MỘT câu ghi giải thích (bọc trong `if false { … }` để vẫn biên dịch): dòng đó có test nào đòi nó không.

Họ nào sinh ra ít hơn ngưỡng tối thiểu thì thoát với mã 3 (mã đổi kiểu viết khiến bộ sinh không còn nhận ra câu lệnh — phải sửa bộ sinh, không được
để danh mục lặng lẽ yếu đi). Đột biến chỉ đổi phần ghi giải thích, không đổi quyết định của luồng hỏi.
"""
import re
import shlex
import sys

WHY, INTENT, ROUTER, ASK = "core/pipeline/why.go", "intelligence/intent/intent.go", "intelligence/router/router.go", "core/pipeline/ask.go"
MIN = {"NV": 20, "QV": 12, "LV": 50}

# lệnh ghi một dòng giải thích
EMIT = re.compile(r"(\.Reason\(|\.Reject\(|\.reason\(|\.reject\(|fmt\.Sprintf\(|n\.Reasons = append)")
# số được in ra: len(...), ưu tiên, điểm, số lần, các biến đếm có tên cố định
NUM = re.compile(
    r"len\((?:[^()]|\([^()]*\))*\)|\b(?:d|top)\.Priority\b|score\[[^\]]+\]|\bg\.n\b|\bdropped\b|\bentities\b|\bfacts\b|\bnMain\b"
    r"|groups\[src\]\.(?:authority|count)\b|\bk\b(?=\))|\bnar\.Slot\b(?!\s*!=)"
)
# câu lệnh ghi giải thích gọn trên MỘT dòng
STMT = re.compile(r"^\s*(n\.Reason\(|n\.Reject\(|w\.reason\(|w\.reject\(|why\.Context\.Reject\()(.*)\)$")
QUOTE = "contracts.Quote("


def read(path):
    with open(path, encoding="utf-8") as f:
        return f.read()


def numeric(path):
    """NV: thêm +1 sau mỗi số in ra trong đối số của lệnh ghi giải thích."""
    src = read(path)
    lines = src.split("\n")
    out = []
    for i, line in enumerate(lines):
        prev = lines[i - 1] if i > 0 else ""
        # dòng có %d, hoặc dòng đối số nối tiếp ngay sau một dòng có %d và kết thúc bằng dấu phẩy
        if "%d" not in line and not ("%d" in prev and prev.rstrip().endswith(",")):
            continue
        if not (EMIT.search(line) or EMIT.search(prev)):
            continue
        if path == INTENT and not any(k in (line + prev) for k in ("w.reason", "w.reject", "Sprintf")):
            continue
        cut = line.rfind('"') if '"' in line else -1  # chỉ đột biến phần đối số, sau chuỗi định dạng
        for m in NUM.finditer(line):
            if m.start() < cut:
                continue
            frm = line.strip()
            if src.count(frm) != 1:
                continue
            to = (line[: m.end()] + "+1" + line[m.end():]).strip()
            out.append((path, frm, to, f"[sinh: +1 vào số in ra] {path.rsplit('/', 1)[-1]}: {m.group(0)} ở {frm[:48]}"))
    return out


def unquote(path):
    """QV: bỏ contracts.Quote( … ) ở từng chỗ nhúng chữ của người hỏi."""
    src = read(path)
    out = []
    for line in src.split("\n"):
        if line.strip().startswith("//"):
            continue
        pos = 0
        while True:
            k = line.find(QUOTE, pos)
            if k < 0:
                break
            j, depth = k + len(QUOTE), 1
            while j < len(line) and depth > 0:
                depth += (line[j] == "(") - (line[j] == ")")
                j += 1
            expr = line[k + len(QUOTE): j - 1]
            frm = line.strip()
            if src.count(frm) == 1:
                to = (line[:k] + "(" + expr + ")" + line[j:]).strip()
                out.append((path, frm, to, f"[sinh: bỏ Quote] {path.rsplit('/', 1)[-1]}: {expr[:48]}"))
            pos = k + 1
    return out


def drop(path):
    """LV: bọc `if false { … }` quanh từng câu lệnh ghi giải thích trên một dòng."""
    src = read(path)
    out = []
    for line in src.split("\n"):
        if not STMT.match(line):
            continue
        frm = line.strip()
        if src.count(frm) != 1 or frm.count("(") != frm.count(")"):
            continue
        out.append((path, frm, "if false { " + frm + " }", f"[sinh: bỏ dòng giải thích] {path.rsplit('/', 1)[-1]}: {frm[:60]}"))
    return out


def main():
    families = {
        "NV": [m for p in (WHY, INTENT, ROUTER) for m in numeric(p)],
        "QV": [m for p in (WHY, INTENT) for m in unquote(p)],
        "LV": [m for p in (WHY, INTENT, ASK) for m in drop(p)],
    }
    low = {k: len(v) for k, v in families.items() if len(v) < MIN[k]}
    print("sinh: " + " · ".join(f"{k}={len(v)}" for k, v in families.items()), file=sys.stderr)
    if low:
        print(f"bộ sinh nhận ra quá ít câu lệnh (tối thiểu {MIN}, được {low}): kiểu viết mã đã đổi, sửa scripts/dot-bien-vet-sinh.py", file=sys.stderr)
        return 3
    for code, muts in families.items():
        for n, (path, frm, to, label) in enumerate(muts, 1):
            print("m", path, shlex.quote(frm), shlex.quote(to), shlex.quote(f"{code}{n} {label}"))
    return 0


if __name__ == "__main__":
    sys.exit(main())
