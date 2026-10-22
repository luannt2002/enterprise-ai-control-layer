#!/usr/bin/env python3
"""Đếm "số đúng" của tầng S6 trên đầu ra `winreport -v` (dòng S6 của CỔNG TRƯỚC AI, docs/bao-cao/CONG-TRUOC-AI.md).

    python3 -I scripts/do-s6-so-dung.py <tệp ca .yaml> <đầu ra winreport -v> [<tệp ca> <đầu ra> …]

Chỉ xét ca có kỳ vọng số (expect.value). Ca có lý do "route = …" bị loại khỏi mẫu số: câu chưa tới S6 đúng đường. Trong số còn lại, ca
ĐẠT hoặc ĐỎ mà không có lý do "thiếu số …" tính là có số đúng trong câu trả lời. Ca "route đúng mà thiếu / sai số" được liệt kê để người
phân tích xác định tầng sai đầu tiên (thường là S4 chọn sai chỉ số: S6 tính đúng cho chỉ số sai). Script không tự quy tầng.
"""
import re
import sys

import yaml

LINE = re.compile(r"^  (ĐẠT|ĐỎ)\s+(WIN-I-\d+)\s*(?:\[[a-z_]+\])?\s*(.*)$")


def main(args):
    if len(args) < 2 or len(args) % 2:
        print(__doc__, file=sys.stderr)
        return 2
    total = [0, 0]
    for cases_path, report_path in zip(args[::2], args[1::2]):
        doc = yaml.safe_load(open(cases_path, encoding="utf-8"))
        cases = doc["cases"] if isinstance(doc, dict) else doc
        expect = {c["id"]: (c.get("expect") or {}) for c in cases}
        status = {}
        for line in open(report_path, encoding="utf-8"):
            m = LINE.match(line.rstrip("\n"))
            if m:
                status[m.group(2)] = (m.group(1), m.group(3))
        missing = [cid for cid, e in expect.items() if e.get("value") is not None and cid not in status]
        if missing:
            print(f"{report_path}: thiếu {len(missing)} ca có kỳ vọng số trong đầu ra (cần winreport -v đủ ca)", file=sys.stderr)
            return 2
        considered, wrong = 0, []
        for cid, e in expect.items():
            if e.get("value") is None:
                continue
            verdict, why = status[cid]
            if "route =" in why:
                continue
            considered += 1
            if verdict == "ĐỎ" and "thiếu số" in why:
                wrong.append(cid)
        total[0] += considered
        total[1] += len(wrong)
        print(f"{cases_path}: route đúng và có kỳ vọng số {considered}; số đúng {considered - len(wrong)}; thiếu / sai số {len(wrong)}"
              + (f" ({', '.join(wrong)})" if wrong else ""))
    print(f"TỔNG: route đúng và có kỳ vọng số {total[0]}; số đúng {total[0] - total[1]}; thiếu / sai số {total[1]}")
    return 0


if __name__ == "__main__":
    sys.exit(main(sys.argv[1:]))
