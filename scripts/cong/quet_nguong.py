# Quét ngưỡng "biết im" trên lượt hiệu chỉnh (ngưỡng ~0): đúng = câu có đáp án mà top-3 chứa đủ chuỗi bắt buộc VÀ điểm đầu ≥ t,
# + câu phải từ chối mà điểm đầu < t. In đường cong và ngưỡng tốt nhất của từng biến thể.
import json, sys
D = sys.argv[1]
for v in sys.argv[2:]:
    rs = [json.loads(l) for l in open(f"{D}/{v}/ket-qua.jsonl")]
    rows = []
    for r in rs:
        if r.get("KhongCham"):
            continue
        tr = r.get("TruyHoi") or []
        top = tr[0]["Diem"] if tr else -1
        tu_choi = "phải từ chối" in r["KyVong"]
        rows.append((r["Ma"], top, r["TruyHoiTrung"], tu_choi))
    co = [x for x in rows if not x[3]]; tc = [x for x in rows if x[3]]
    print(f"\n{v}: {len(co)} câu có đáp án · {len(tc)} câu phải từ chối · top-3 chứa đủ (ngưỡng ~0): {sum(x[2] for x in co)}")
    best = None
    for t in [0.0, 0.02, 0.04, 0.06, 0.08, 0.10, 0.12, 0.14, 0.16, 0.175, 0.19, 0.21, 0.23, 0.25]:
        a = sum(1 for x in co if x[2] and x[1] >= t); b = sum(1 for x in tc if x[1] < t)
        print(f"  t={t:<6} trúng+qua ngưỡng {a:3}/{len(co)} · từ chối đúng {b:2}/{len(tc)} · tổng {a+b}")
        if best is None or a + b > best[1]:
            best = (t, a + b)
    print("  tốt nhất:", best)
    print("  điểm đầu của câu phải từ chối:", sorted(round(x[1], 3) for x in tc))
