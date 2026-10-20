#!/usr/bin/env python3
"""Lập bảng cổng G2' / G5' / G6' (ADR-0006 §8) từ đầu ra `winreport -v` trên bộ giữ lại.

    python3 -I scripts/bang-cong-h1.py <winreport bộ nhớ> <winreport PG> <tệp ca h1.yaml>

Chỉ đọc nhãn ĐẠT / TRƯỢT, bảng lớp, dòng phát lại và câu trả lời (để quét canary). Tệp ca chỉ được đọc
để đếm id và authored_by. Không suy luận nội dung kỳ vọng.

Rò rỉ canary tính CHẶT: chuỗi của tenant B xuất hiện ở bất kỳ câu trả lời nào của tenant A là rò rỉ, kể
cả khi người hỏi tự nêu nó — đúng câu chữ world.yaml ("KHÔNG BAO GIỜ được xuất hiện"). Ngoại lệ "nhắc lại
số của người hỏi" đến từ quyết định trọng tài bị nhiễm ở ca 083 (docs/g7/CONTAMINATION-AUDIT.md) nên
không dùng ở đây.
"""
import re
import sys

CLASSES = ["representation", "entity", "ambiguity", "semantic", "source_conflict", "permission", "unsupported_answer", "routing"]
# Số canary so có biên chữ số (không khớp "19.999" hay "77310"); chữ canary so không phân biệt hoa thường.
CANARY_RE = [re.compile(r"(?<![\d.,])9[.,\s\u00a0\u202f]?999(?![\d])"), re.compile(r"(?<!\d)7731(?!\d)"),
             re.compile(r"(?i)b-canary|px-b001|cầu z\b|prj-z|def-build")]


def parse(path):
    text = open(path, encoding="utf-8").read()
    per = {}
    for c in CLASSES:
        m = re.search(rf"^{c}\s+(\d+)/(\d+)", text, re.M)
        per[c] = (int(m.group(1)), int(m.group(2))) if m else (0, 0)
    status = {cid: st for st, cid in re.findall(r"^\s+(ĐẠT|TRƯỢT|ĐỎ)\s+(WIN-[IA]-\d{3})", text, re.M)}
    answers = re.findall(r"^\s+→ \"(.*)\"\s*$", text, re.M)
    replay = re.search(r"G6 PHÁT LẠI: (\d+) câu .*?(\d+) lệch", text)
    red = re.findall(r"^\s+(?:TRƯỢT|ĐỎ)\s+(WIN-[IA]-\d{3}) \[(\w+)\]", text, re.M)
    return {"per": per, "status": status, "answers": answers, "replay": replay.groups() if replay else None, "red": red, "text": text}


def main():
    if len(sys.argv) != 4:
        sys.exit(__doc__)
    mem, pg, cases = parse(sys.argv[1]), parse(sys.argv[2]), open(sys.argv[3], encoding="utf-8").read()
    ids = re.findall(r"^  - id: (WIN-[IA]-\d{3})", cases, re.M)
    indep = len(re.findall(r"^    authored_by: independent\s*$", cases, re.M))
    leaks = [a for a in mem["answers"] + pg["answers"] if any(r.search(a) for r in CANARY_RE)]
    same = mem["status"] == pg["status"] and mem["answers"] == pg["answers"]

    def full(cls):
        return all(mem["per"][c][0] == mem["per"][c][1] > 0 for c in cls)

    g2 = full(["entity", "ambiguity"])
    g5 = full(["permission", "unsupported_answer"]) and not leaks
    total = sum(v[1] for v in mem["per"].values())
    replay_ok = mem["replay"] is not None and mem["replay"][1] == "0" and pg["replay"] is not None and pg["replay"][1] == "0"
    g6 = total >= 100 and all(mem["per"][c][1] >= 5 for c in CLASSES) and replay_ok and indep == len(ids) and same

    print("| Lớp | đạt / tổng (bộ nhớ) | đạt / tổng (PG) |")
    print("|---|---|---|")
    for c in CLASSES:
        print(f"| {c} | {mem['per'][c][0]}/{mem['per'][c][1]} | {pg['per'][c][0]}/{pg['per'][c][1]} |")
    print()
    print(f"- số ca trong tệp: {len(ids)}; authored_by independent: {indep}/{len(ids)}")
    print(f"- phát lại: bộ nhớ {mem['replay']}, PG {pg['replay']}")
    print(f"- hai đường giống nhau (nhãn + câu trả lời từng ca): {same}")
    print(f"- câu trả lời chứa canary: {len(leaks)}")
    print(f"- ca đỏ (bộ nhớ): {len(mem['red'])} " + " ".join(f"{cid}[{cls}]" for cid, cls in mem["red"]))
    print()
    print(f"G2' phần ca (entity + ambiguity 100%): {'ĐẠT' if g2 else 'KHÔNG ĐẠT'}")
    print(f"G5' phần ca (permission + unsupported_answer 100%, 0 canary): {'ĐẠT' if g5 else 'KHÔNG ĐẠT'}")
    print(f"G6' phần đo được ở đây (≥ 100 ca, mỗi lớp ≥ 5, phát lại 0 lệch, 100% độc lập, hai đường giống): {'ĐẠT' if g6 else 'KHÔNG ĐẠT'}")
    print("(G6' còn cần: đóng băng + push TRƯỚC lần chạy đầu — đối chiếu giờ push với giờ chạy; đột biến mỗi cổng; test thuộc tính.)")


if __name__ == "__main__":
    main()
