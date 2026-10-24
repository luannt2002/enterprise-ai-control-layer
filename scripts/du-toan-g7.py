#!/usr/bin/env python3
"""Dự toán chi phí G7 từ bằng chứng thô của lượt CHẠY KHÔ (g7run -out, máy chủ giả, k = 1).

    python3 -I scripts/du-toan-g7.py --pilot <thư mục -out của pilot> --main <thư mục -out của H1> [--cap 15]

Đo được (từ thân yêu cầu thật trong raw-runs.jsonl): số lời gọi model mỗi hệ, số ký tự prompt
(system + user) mỗi lời gọi. Ước lượng (ghi rõ, chưa đo bằng tokenizer thật vì count_tokens cần khoá):
- token vào = ký tự / (ký tự mỗi token): tối thiểu 3,0 · kỳ vọng 2,5 · xấu nhất 1,5;
- token ra mỗi lời gọi: tối thiểu 50 · kỳ vọng 150 · xấu nhất = max_tokens (trần cứng của API, gồm cả
  thinking nếu có).
Giá USD / triệu token (vào, ra) của model hợp đồng: EAIS_G7_MODEL tra trong EAIS_G7_PRICES (.env, cùng dạng
"model=vào/ra;…" của g7run; giá tra trang Pricing chính thức, docs/g7/CHI-PHI-G7.md mục 5). Thiếu hay sai thì dừng, không dự toán.
Pilot chạy k = 5 (ADR-0006 §6); lượt chính tính cả k = 3 và k = 5. Hệ cổng: EAIS, B1, B2+;
chỉ báo cáo: B2, B2″, B2-oracle.
Trần 15 USD là của TOÀN BỘ G7 (hợp đồng §9): cờ vượt trần xét cả tổng hệ cổng, tổng mọi hệ được chạy, và tổng
kèm dự phòng (thử khói 3 ca + chạy lại TRỌN lượt chính với cùng k sau lượt vô hiệu, §10: lượt k = 1 không ra
kết luận vì R0 đòi k ≥ 3, lẻ) — bản soát lượt 2, C-17; soát C2, F-09.
"""
import argparse
import json
import math
import os
import sys
from pathlib import Path

SCEN = {"tối thiểu": (3.0, 50), "kỳ vọng": (2.5, 150), "xấu nhất": (1.5, None)}  # None = max_tokens
GATE = ["eais", "b1", "b2plus"]
REPORT = ["b2", "b2pp", "b2oracle"]
LABEL = {"eais": "EAIS", "b1": "B1", "b2plus": "B2+", "b2": "B2", "b2pp": "B2″", "b2oracle": "B2-oracle"}
# Mã hệ trong bằng chứng thô đã lưu trước D-RM-02 (g7run ghi "eacl" từ 20/10 tới tối 21/10/2026): đọc như mã hiện hành.
LEGACY_SYSTEM = {"eacl": "eais"}


def measure(d):
    calls, chars, max_tok = {}, {}, 1024
    for line in open(Path(d) / "raw-runs.jsonl", encoding="utf-8"):
        r = json.loads(line)
        for c in r.get("calls") or []:
            req = c.get("request") or {}
            if not req:
                continue
            user = "".join(m.get("content", "") for m in req.get("messages", []) if isinstance(m.get("content"), str))
            s = LEGACY_SYSTEM.get(r["system"], r["system"])
            calls[s] = calls.get(s, 0) + 1
            chars[s] = chars.get(s, 0) + len(req.get("system", "")) + len(user)
            max_tok = req.get("max_tokens", max_tok)
    return calls, chars, max_tok


def price():
    """(vào, ra) USD / triệu token của EAIS_G7_MODEL trong EAIS_G7_PRICES; luật đọc như priceTable của g7run."""
    model, table = os.environ.get("EAIS_G7_MODEL", "").strip(), {}
    for i, part in enumerate(os.environ.get("EAIS_G7_PRICES", "").split(";"), 1):
        if not part.strip():
            continue
        name, ok, pr = part.partition("=")
        tin, ok2, tout = pr.partition("/")
        name = name.strip()
        try:
            a, b = float(tin), float(tout)
        except ValueError:
            a = b = math.nan
        if not ok or not ok2 or not name or name in table or not (a >= 0 and b >= 0) or math.isinf(a) or math.isinf(b):
            sys.exit(f"EAIS_G7_PRICES mục thứ {i}: phải là model=giá vào/giá ra (USD / triệu token, số không âm), model không trùng")
        table[name] = (a, b)
    if not model or model not in table:
        sys.exit("chưa có giá của model hợp đồng: đặt EAIS_G7_MODEL và giá của nó trong EAIS_G7_PRICES (.env)")
    return table[model]


def cost(calls, chars, k, cpt, out_per_call, pr):
    tin = chars / cpt * k
    tout = calls * out_per_call * k
    return tin, tout, (tin * pr[0] + tout * pr[1]) / 1e6


def main():
    ap = argparse.ArgumentParser()
    ap.add_argument("--pilot", required=True)
    ap.add_argument("--main", required=True)
    ap.add_argument("--cap", type=float, default=15.0)
    ap.add_argument("--pilot-cases", type=int, default=16, help="số ca của pilot, để suy chi phí thử khói 3 ca")
    a = ap.parse_args()
    pr = price()
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
        print("| Kịch bản | " + " | ".join(LABEL[s] for s in systems) + " | Tổng cổng | Tổng cổng + dự phòng cổng | Tổng mọi hệ | Dự phòng | Tổng mọi hệ + dự phòng |")
        print("|---|" + "---|" * (len(systems) + 5))
        for name, (cpt, opc) in SCEN.items():
            opc = opc if opc is not None else mt
            row, gate_total, all_total, spare, spare_gate = [], 0.0, 0.0, 0.0, 0.0
            for s in systems:
                _, _, p = cost(pc.get(s, 0), pch.get(s, 0), 5, cpt, opc, pr)
                _, _, m = cost(mc.get(s, 0), mch.get(s, 0), km, cpt, opc, pr)
                row.append(f"{p + m:.2f}")
                # dự phòng: thử khói 3 ca (theo trung bình mỗi ca của pilot) + chạy lại trọn lượt chính cùng k
                n_pilot = max(1, a.pilot_cases)
                _, _, sm = cost(pc.get(s, 0) * 3 / n_pilot, pch.get(s, 0) * 3 / n_pilot, 1, cpt, opc, pr)
                _, _, rr = cost(mc.get(s, 0), mch.get(s, 0), km, cpt, opc, pr)
                spare += sm + rr
                if s in GATE:
                    spare_gate += sm + rr
                all_total += p + m
                if s in GATE:
                    gate_total += p + m
            def flag(x):
                return "" if x <= a.cap else " ⚠ vượt trần"
            print(f"| {name} | " + " | ".join(row) + f" | {gate_total:.2f}{flag(gate_total)} | {gate_total + spare_gate:.2f}{flag(gate_total + spare_gate)} | {all_total:.2f}{flag(all_total)} | {spare:.2f} | {all_total + spare:.2f}{flag(all_total + spare)} |")
        print()
    print(f"Trần: {a.cap:.0f} USD cho toàn bộ G7 (pilot + chính). Kịch bản 'xấu nhất' là cận trên chỉ khi ký tự / token ≥ 1,5 — chưa đo bằng tokenizer thật.")


if __name__ == "__main__":
    main()
