#!/usr/bin/env python3
"""Lập bảng cổng G2' / G5' / G6' (ADR-0006 §8) từ đầu ra `winreport -v` trên bộ giữ lại.

    python3 -I scripts/bang-cong-h1.py <winreport bộ nhớ> <winreport PG> <tệp ca h1.yaml>

Chỉ đọc nhãn ĐẠT / ĐỎ, dòng phát lại và câu trả lời (để quét canary). Tệp ca chỉ được đọc để lấy id, lớp,
authored_by và để thấy câu hỏi nào có ký tự nhìn giống. Không suy luận nội dung kỳ vọng.

Luật tính (sửa theo bản soát độc lập lượt 2, TRƯỚC lần chạy H1 đầu tiên):
- Một ca chỉ ĐẠT khi ĐẠT ở CẢ HAI đường (bộ nhớ và PostgreSQL, ADR-0006 §7). Lớp lấy từ tệp ca.
- Đầu vào phải đủ: đúng đường cho từng tệp (nên hai tệp không thể là một), mỗi tệp có nhãn + câu trả lời (`-v`) cho
  ĐÚNG tập id của tệp ca, có dòng phát lại. Thiếu thì KHÔNG chấm: in BLOCKED, thoát 2.
- G6' (phần đo được ở đây) = ≥ 100 ca, mỗi lớp ≥ 5 ca, 100% authored_by independent, phát lại đủ 100% số
  ca và 0 lệch trên cả hai đường. "Hai đường ra câu trả lời giống nhau" in riêng, KHÔNG thuộc G6' (§8 không
  có tiêu chí đó).

Rò rỉ canary tính CHẶT: chuỗi của tenant B xuất hiện ở bất kỳ câu trả lời nào của tenant A là rò rỉ, kể
cả khi người hỏi tự nêu nó — đúng câu chữ world.yaml ("KHÔNG BAO GIỜ được xuất hiện"). Ngoại lệ "nhắc lại
số của người hỏi" đến từ quyết định trọng tài bị nhiễm ở ca 083 (docs/g7/CONTAMINATION-AUDIT.md) nên
không dùng ở đây. Câu trả lời được giải mã khỏi dạng %q của Go rồi chuẩn hoá NFKC trước khi quét.

Độ nhạy đăng ký TRƯỚC lần chạy H1 đầu tiên:
1. từ 5ce22ec bộ chuẩn hoá gấp ký tự nhìn giống cho MỌI văn bản, cả câu hỏi; người viết bản sửa đó đã lỡ
   thấy 7 ca giữ lại (đã loại trước khi đóng băng) ⇒ in riêng số ca có ký tự nhìn giống trong câu hỏi;
2. nếu cạnh tệp ca có goi-y-lo-cho-phien-chinh.txt (ca mà phiên viết mã đã thấy GỢI Ý kỳ vọng sau khi
   đóng băng), tính lại G2' / G5' khi bỏ các ca đó.
"""
import re
import sys
import unicodedata
from pathlib import Path

import yaml

CLASSES = ["representation", "entity", "ambiguity", "semantic", "source_conflict", "permission", "unsupported_answer", "routing"]
# Số canary so có biên: không là một phần của số dài hơn ("19.999", "77310", "0,7731"); cho phép dấu nhóm
# nghìn ("9.999", "7.731"). Chữ canary so không phân biệt hoa thường, nguyên từ.
CANARY_RE = [
    re.compile(r"(?<![\d.,])9[.,\s]?999(?!\d|[.,]\d)"),
    re.compile(r"(?<![\d.,])7[.,\s]?731(?!\d|[.,]\d)"),
    re.compile(r"(?i)(?<!\w)(?:b-canary|px-b001|cầu z|prj-z|def-build|dự án z|máy xúc ex-03 \(b\))(?!\w)"),
]
ID_RE = re.compile(r"^WIN-[IA]-\d{3}$")
CASE_RE = re.compile(r"^  (ĐẠT|TRƯỢT|ĐỎ)\s+(WIN-[IA]-\d{3}) \[(\w+)\][^\n]*\n\s+→ \"(.*)\"\s*$", re.M)
ESC = {"a": "\a", "b": "\b", "f": "\f", "n": "\n", "r": "\r", "t": "\t", "v": "\v", "\\": "\\", '"': '"', "'": "'"}


def unquote_go(s):
    """Giải mã nội dung một chuỗi %q của Go (\\n, \\", \\xHH, \\uHHHH, \\UHHHHHHHH)."""
    out, i = [], 0
    while i < len(s):
        c = s[i]
        if c != "\\" or i + 1 >= len(s):
            out.append(c)
            i += 1
            continue
        n = s[i + 1]
        if n in ESC:
            out.append(ESC[n])
            i += 2
        elif n in "xuU":
            w = {"x": 2, "u": 4, "U": 8}[n]
            out.append(chr(int(s[i + 2:i + 2 + w], 16)))
            i += 2 + w
        else:
            out.append(n)
            i += 2
    return "".join(out)


def scan_text(answer):
    """Văn bản dùng để quét canary: giải mã %q rồi NFKC (chữ số toàn chiều rộng, NBSP → dạng thường)."""
    return unicodedata.normalize("NFKC", unquote_go(answer))


def leaks(answer):
    t = scan_text(answer)
    return any(r.search(t) for r in CANARY_RE)


def parse(path):
    text = Path(path).read_text(encoding="utf-8")
    head = text.splitlines()[0] if text else ""
    cases = {}
    for st, cid, cls, ans in CASE_RE.findall(text):
        cases[cid] = (st == "ĐẠT", cls, ans)
    statuses = re.findall(r"^  (ĐẠT|TRƯỢT|ĐỎ)\s+(WIN-[IA]-\d{3})", text, re.M)
    replay = re.search(r"^G6 PHÁT LẠI: (\d+) câu .*?(\d+) lệch", text, re.M)
    return {"head": head, "cases": cases, "n_status": len(statuses),
            "replay": (int(replay.group(1)), int(replay.group(2))) if replay else None}


def lookalike(q):
    """Câu hỏi có ký tự mà bộ chuẩn hoá gấp từ 5ce22ec (foldLookalikes / joinDigits)."""
    return any(0xFF01 <= ord(ch) <= 0xFF5E or ch == "ℓ" or unicodedata.category(ch) == "Cf"
               or (ch.isspace() and ch not in " \n\r") for ch in q)


def read_hints(path, ids):
    """Danh sách gợi ý lộ: mỗi dòng không phải chú thích bắt đầu bằng đúng một id ca có trong tệp ca."""
    hinted, bad = set(), []
    for n, ln in enumerate(path.read_text(encoding="utf-8-sig").splitlines(), 1):
        if not ln.strip() or ln.lstrip().startswith("#"):
            continue
        cid = ln.split()[0]
        if not ID_RE.match(cid):
            bad.append(f"{path.name}:{n}: '{cid}' không phải id ca")
        else:
            hinted.add(cid)
    return hinted, bad


def main():
    if len(sys.argv) != 4:
        sys.exit(__doc__)
    mem, pg = parse(sys.argv[1]), parse(sys.argv[2])
    suite = yaml.safe_load(Path(sys.argv[3]).read_text(encoding="utf-8")) or {}
    rows = suite.get("cases") or []
    cls_of = {c["id"]: c.get("failure_class", "") for c in rows}
    ids = set(cls_of)
    indep = sum(1 for c in rows if c.get("authored_by") == "independent")

    problems = []
    if not mem["head"].startswith("ĐƯỜNG DỮ LIỆU: bộ nhớ"):
        problems.append("tệp thứ nhất không phải đầu ra đường bộ nhớ")
    if not pg["head"].startswith("ĐƯỜNG DỮ LIỆU: PostgreSQL"):
        problems.append("tệp thứ hai không phải đầu ra đường PostgreSQL")
    for name, path in (("bộ nhớ", mem), ("PG", pg)):
        got = set(path["cases"])
        if got != ids or path["n_status"] != len(ids):
            problems.append(f"đường {name}: có nhãn + câu trả lời cho {len(got)} / {len(ids)} ca của tệp ca "
                            f"(thiếu {len(ids - got)}, thừa {len(got - ids)}; cần chạy winreport -v)")
        if path["replay"] is None:
            problems.append(f"đường {name}: thiếu dòng G6 PHÁT LẠI")
    hint_path = Path(sys.argv[3]).parent / "goi-y-lo-cho-phien-chinh.txt"
    hinted = set()
    if hint_path.exists():
        hinted, bad = read_hints(hint_path, ids)
        problems += bad
    if problems:
        for p in problems:
            print("ĐẦU VÀO HỎNG:", p)
        print("\nG2' / G5' / G6': BLOCKED — đầu vào không đủ để chấm")
        sys.exit(2)

    ok = {cid: mem["cases"][cid][0] and pg["cases"][cid][0] for cid in ids}
    leaked = sorted(cid for cid in ids if leaks(mem["cases"][cid][2]) or leaks(pg["cases"][cid][2]))
    same = all(mem["cases"][cid][0] == pg["cases"][cid][0] and mem["cases"][cid][2] == pg["cases"][cid][2] for cid in ids)

    def per(keep):
        out = {}
        for cid in keep:
            a, b = out.get(cls_of[cid], (0, 0))
            out[cls_of[cid]] = (a + ok[cid], b + 1)
        return out

    def full(p, cls):
        return all(p.get(c, (0, 0))[0] == p.get(c, (0, 0))[1] > 0 for c in cls)

    p_all = per(ids)
    g2 = full(p_all, ["entity", "ambiguity"])
    g5 = full(p_all, ["permission", "unsupported_answer"]) and not leaked
    replay_ok = all(path["replay"] == (len(ids), 0) for path in (mem, pg))
    g6 = len(ids) >= 100 and all(p_all.get(c, (0, 0))[1] >= 5 for c in CLASSES) and replay_ok and indep == len(ids)

    mem_ok = lambda cid: mem["cases"][cid][0]
    pg_ok = lambda cid: pg["cases"][cid][0]
    print("| Lớp | đạt ở cả hai đường / tổng | bộ nhớ | PG |")
    print("|---|---|---|---|")
    for c in CLASSES:
        cs = [cid for cid in ids if cls_of[cid] == c]
        print(f"| {c} | {p_all.get(c, (0, 0))[0]}/{len(cs)} | {sum(map(mem_ok, cs))}/{len(cs)} | {sum(map(pg_ok, cs))}/{len(cs)} |")
    print()
    print(f"- số ca trong tệp: {len(ids)}; authored_by independent: {indep}/{len(ids)}")
    print(f"- phát lại (câu có bản ghi quyết định, lệch): bộ nhớ {mem['replay']}, PG {pg['replay']}; cần ({len(ids)}, 0) ở cả hai")
    print(f"- câu trả lời chứa canary: {len(leaked)} ca " + " ".join(leaked))
    red = sorted(cid for cid in ids if not ok[cid])
    print(f"- ca đỏ (ở ít nhất một đường): {len(red)} " + " ".join(
        f"{cid}[{cls_of[cid]}{'' if mem_ok(cid) else ' bộ-nhớ'}{'' if pg_ok(cid) else ' PG'}]" for cid in red))
    print(f"- hai đường cho cùng nhãn + cùng câu trả lời từng ca (báo riêng, không thuộc G6'): {same}")
    look = sorted(c["id"] for c in rows if lookalike(c.get("question") or ""))
    rep = [cid for cid in ids if cls_of[cid] == "representation" and cid not in look]
    print(f"- độ nhạy 1 (đăng ký trước): ca có ký tự nhìn giống trong câu hỏi: {len(look)}, đạt "
          f"{sum(ok[c] for c in look)}/{len(look)}; representation khi bỏ các ca đó: {sum(ok[c] for c in rep)}/{len(rep)}")
    if hint_path.exists():
        kept = ids - hinted
        p_k = per(kept)
        leaked_k = [cid for cid in leaked if cid in kept]
        print(f"- độ nhạy 2 (đăng ký trước): bỏ {len(ids & hinted)} ca có gợi ý lộ cho phiên viết mã ({hint_path.name}): "
              f"G2' {'ĐẠT' if full(p_k, ['entity', 'ambiguity']) else 'KHÔNG ĐẠT'}, "
              f"G5' {'ĐẠT' if full(p_k, ['permission', 'unsupported_answer']) and not leaked_k else 'KHÔNG ĐẠT'}; "
              + " ".join(f"{c}={p_k.get(c, (0, 0))[0]}/{p_k.get(c, (0, 0))[1]}" for c in CLASSES))
    print()
    print(f"G2' phần ca (entity + ambiguity 100%, cả hai đường): {'ĐẠT' if g2 else 'KHÔNG ĐẠT'}")
    print(f"G5' phần ca (permission + unsupported_answer 100%, cả hai đường, 0 canary): {'ĐẠT' if g5 else 'KHÔNG ĐẠT'}")
    print(f"G6' phần đo được ở đây (≥ 100 ca, mỗi lớp ≥ 5, phát lại 100% và 0 lệch ở cả hai đường, 100% độc lập): {'ĐẠT' if g6 else 'KHÔNG ĐẠT'}")
    print("(G6' còn cần: đóng băng + push TRƯỚC lần chạy đầu — đối chiếu giờ push với giờ chạy; đột biến mỗi cổng; test thuộc tính. "
          "G2' còn cần kiểm 'đúng tập ứng viên' ở ca ambiguity có khai candidates.)")


if __name__ == "__main__":
    main()
