#!/usr/bin/env python3
"""Dự toán chi phí G7 từ bằng chứng thô của lượt CHẠY KHÔ (g7run -out, máy chủ giả, k = 1).

    python3 -I scripts/du-toan-g7.py --pilot <thư mục -out của pilot> --main <thư mục -out của H1> [--cap 15]

Đo được (từ thân yêu cầu thật trong raw-runs.jsonl): số lời gọi model mỗi hệ, số ký tự prompt
(system + user) mỗi lời gọi. Ước lượng (ghi rõ, chưa đo bằng tokenizer thật vì count_tokens cần khoá):
- token vào = ký tự / (ký tự mỗi token): tối thiểu 3,0 · kỳ vọng 2,5 · xấu nhất 1,5;
- token ra mỗi lời gọi: tối thiểu 50 · kỳ vọng 150 · xấu nhất = max_tokens (trần cứng của API, gồm cả
  thinking nếu có).
Giá Sonnet 5.5 (Pricing chính thức, tra 20/10/2026): vào 2 USD, ra 10 USD mỗi triệu token.
Pilot chạy k = 5 (ADR-0006 §6); lượt chính tính cả k = 3 và k = 5. Hệ cổng: EACL, B1, B2+;
chỉ báo cáo: B2, B2″.
"""
import argparse
import json
from pathlib import Path

PRICE_IN, PRICE_OUT = 2.0, 10.0  # USD / triệu token, claude-sonnet-5-5
SCEN = {"tối thiểu": (3.0, 50), "kỳ vọng": (2.5, 150), "xấu nhất": (1.5, None)}  # None = max_tokens
GATE = ["eacl", "b1", "b2plus"]
REPORT = ["b2", "b2pp"]
LABEL = {"eacl": "EACL", "b1": "B1", "b2plus": "B2+", "b2": "B2", "b2pp": "B2″"}


def measure(d):
    calls, chars, max_tok = {}, {}, 1024
    for line in open(Path(d) / "raw-runs.jsonl", encoding="utf-8"):
        r = json.loads(line)
        for c in r.get("calls") or []:
            req = c.get("request") or {}
            if not req:
                continue
            user = "".join(m.get("content", "") for m in req.get("messages", []) if isinstance(m.get("content"), str))
            s = r["system"]
            calls[s] = calls.get(s, 0) + 1
            chars[s] = chars.get(s, 0) + len(req.get("system", "")) + len(user)
            max_tok = req.get("max_tokens", max_tok)
    return calls, chars, max_tok


def cost(calls, chars, k, cpt, out_per_call):
    tin = chars / cpt * k
    tout = calls * out_per_call * k
    return tin, tout, (tin * PRICE_IN + tout * PRICE_OUT) / 1e6


def main():
    ap = argparse.ArgumentParser()
    ap.add_argument("--pilot", required=True)
    ap.add_argument("--main", required=True)
    ap.add_argument("--cap", type=float, default=15.0)
    a = ap.parse_args()
    pc, pch, mt = measure(a.pilot)
    mc, mch, _ = measure(a.main)
    systems = [s for s in GATE + REPORT if s in pc or s in mc]

    print("## Đo được (chạy khô, k = 1)\n")
    print("| Hệ | Lời gọi pilot | Ký tự pilot | Lời gọi H1 | Ký tự H1 | Ký tự TB / lời gọi (H1) |")
    print("|---|---|---|---|---|---|")
    for s in systems:
        avg = mch.get(s, 0) / mc[s] if mc.get(s) else 0
        print(f"| {LABEL[s]} | {pc.get(s, 0)} | {pch.get(s, 0)} | {mc.get(s, 0)} | {mch.get(s, 0)} | {avg:.0f} |")
    print(f"\nmax_tokens = {mt}\n")

    for km in (3, 5):
        print(f"## Ước lượng: pilot k = 5 + lượt chính H1 k = {km} (USD)\n")
        print("| Kịch bản | " + " | ".join(LABEL[s] for s in systems) + " | Tổng cổng | Tổng mọi hệ |")
        print("|---|" + "---|" * (len(systems) + 2))
        for name, (cpt, opc) in SCEN.items():
            opc = opc if opc is not None else mt
            row, gate_total, all_total = [], 0.0, 0.0
            for s in systems:
                _, _, p = cost(pc.get(s, 0), pch.get(s, 0), 5, cpt, opc)
                _, _, m = cost(mc.get(s, 0), mch.get(s, 0), km, cpt, opc)
                row.append(f"{p + m:.2f}")
                all_total += p + m
                if s in GATE:
                    gate_total += p + m
            flag = "" if gate_total <= a.cap else " ⚠ vượt trần"
            print(f"| {name} | " + " | ".join(row) + f" | {gate_total:.2f}{flag} | {all_total:.2f} |")
        print()
    print(f"Trần: {a.cap:.0f} USD cho toàn bộ G7 (pilot + chính). Kịch bản 'xấu nhất' là cận trên chỉ khi ký tự / token ≥ 1,5 — chưa đo bằng tokenizer thật.")


if __name__ == "__main__":
    main()
