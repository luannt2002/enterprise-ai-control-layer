# Chấm một lượt chạy ngưỡng ~0 tại các ngưỡng CỐ ĐỊNH (vì "biết im" chỉ so điểm đoạn đầu, áp ngưỡng sau = chạy với ngưỡng đó, ở tầng
# truy hồi; model trích nguyên văn không phụ thuộc ngưỡng). Truy hồi: câu có đáp án mà top-3 chứa đủ chuỗi bắt buộc và qua ngưỡng.
# Trả lời: câu có đáp án mà bộ chấm cho Đạt và qua ngưỡng; câu phải từ chối mà dưới ngưỡng.
# Dùng: cham_nguong.py <thư mục kết quả> <biến thể>=<ngưỡng>[,<ngưỡng>…] …
import json, sys
D = sys.argv[1]
for arg in sys.argv[2:]:
    v, ts = arg.split("=")
    rows = []
    for l in open(f"{D}/{v}/ket-qua.jsonl"):
        r = json.loads(l)
        if r.get("KhongCham"):
            continue
        tr = r.get("TruyHoi") or []
        rows.append((r["Ma"], tr[0]["Diem"] if tr else -1, r["TruyHoiTrung"], r["Dat"], "phải từ chối" in r["KyVong"]))
    co = [x for x in rows if not x[4]]; tc = [x for x in rows if x[4]]
    for t in map(float, ts.split(",")):
        th = sum(1 for x in co if x[2] and x[1] >= t); tl = sum(1 for x in co if x[3] and x[1] >= t); im = sum(1 for x in tc if x[1] < t)
        print(f"{v} @ {t}: truy hồi top-3 đủ {th}/{len(co)} ({100*th//len(co)}%) · trả lời đạt (model trích) {tl}/{len(co)} · từ chối đúng {im}/{len(tc)}")
