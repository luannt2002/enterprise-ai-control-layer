#!/usr/bin/env python3
"""Tìm ca ĐẠT mà số kỳ vọng (expect.value) chỉ khớp như CHUỖI CON của một số khác trong câu trả lời: bộ chấm (`evals/runner.go`) so
bằng strings.Contains nên "3.850" khớp cả trong "13.850". Lỗi 5d của soát độc lập 23/10.

    python3 -I scripts/kiem-so-chuoi-con.py <tệp ca .yaml> <đầu ra winreport -v> [<tệp ca> <đầu ra> …]

Số khớp "nguyên vẹn" khi không có chữ số đứng ngay trước / sau, và không có [.,] + chữ số đứng ngay trước / sau (dấu nghìn, thập phân).
In từng ca ĐẠT chỉ nhờ chuỗi con (kèm đầu câu trả lời) và dòng tổng. Thoát 1 khi có ca như vậy, 0 khi không.
"""
import re
import sys

import yaml

LINE = re.compile(r"^  (ĐẠT|ĐỎ)\s+(WIN-I-\d+)")


def vi_number(n):
    neg = n.startswith("-")
    n = n.lstrip("-")
    ip, _, fr = n.partition(".")
    g = []
    while len(ip) > 3:
        g.insert(0, ip[-3:])
        ip = ip[:-3]
    g.insert(0, ip)
    out = ".".join(g) + ("," + fr if fr else "")
    return ("-" if neg else "") + out


def main(a):
    tot_only_sub, tot_cases = [], 0
    for cases_path, rep in zip(a[::2], a[1::2]):
        doc = yaml.safe_load(open(cases_path, encoding="utf-8"))
        cases = doc["cases"] if isinstance(doc, dict) else doc
        exp = {c["id"]: c.get("expect", {}).get("value") for c in cases}
        lines = open(rep, encoding="utf-8").read().split("\n")
        status, ans = {}, {}
        for i, l in enumerate(lines):
            m = LINE.match(l)
            if m:
                status[m.group(2)] = m.group(1)
                j = i + 1
                while j < len(lines) and not lines[j].lstrip().startswith("→"):
                    if LINE.match(lines[j]):
                        break
                    j += 1
                if j < len(lines) and lines[j].lstrip().startswith("→"):
                    ans[m.group(2)] = lines[j].split("→", 1)[1]
        for cid, v in exp.items():
            if not v or v.get("number") is None:
                continue
            tot_cases += 1
            num = vi_number(str(v["number"]))
            a_ = ans.get(cid)
            if a_ is None:
                print(f"{rep}: {cid} không thấy câu trả lời trong đầu ra -v")
                continue
            sub = a_.count(num)
            whole = len(re.findall(r"(?<![\d])(?<![.,]\d)" + re.escape(num) + r"(?![\d])(?![.,]\d)", a_))
            if status.get(cid) == "ĐẠT" and sub > 0 and whole == 0:
                tot_only_sub.append((rep, cid, num))
                print(f"ĐẠT CHỈ NHỜ CHUỖI CON: {cid} số {num} — {a_[:200]}")
    print(f"ca có kỳ vọng số: {tot_cases}; ca ĐẠT chỉ nhờ chuỗi con: {len(tot_only_sub)}")
    return 1 if tot_only_sub else 0


if __name__ == "__main__":
    sys.exit(main(sys.argv[1:]))
