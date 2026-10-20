# COST-PLAN — dự toán chi phí G7 theo bộ đã đóng băng

- **Ngày đo:** 21/10/2026 rạng sáng, trên mã `9de63d2`. Pipeline, client model và `g7run` đã đổi sau đó (tới bản chốt sáng 21/10): số lời gọi / ký tự CHƯA đo lại. Dữ liệu functional seed 42, manifest `31172355…adb055`.
- **Đã đo (chạy khô):**
  - máy chủ giả `scripts/g7-chay-kho.py` ở cổng 25480, `g7run -k 1`, mọi hệ (EACL, B1, B2+, B2, B2″, B2-oracle);
  - tham số đúng hợp đồng §4 (`claude-sonnet-5-5`, thinking `between_tools`, effort `medium`, không temperature);
  - tập: `pilot.yaml` (16 ca) và `h1.yaml` (104 ca).
  - Đo được số lời gọi model và số ký tự prompt (system + user) của từng lời gọi, lấy từ thân yêu cầu thật trong `raw-runs.jsonl`.
- **Chưa đo:**
  - tỉ lệ ký tự / token (cần `count_tokens` hoặc usage thật, tức cần khoá);
  - số token ra thật.
  - Hai số này là **ƯỚC** theo ba kịch bản của `scripts/du-toan-g7.py`.
- **Lệnh:** `python3 -I scripts/du-toan-g7.py --pilot <out pilot> --main <out H1> --cap 15`. Công cụ chỉ in số, không in nội dung ca.
- Từ 21/10, cờ vượt trần xét mọi tổng. Cột "dự phòng" = thử khói 3 ca + chạy lại TRỌN lượt chính với cùng k sau lượt vô hiệu (bản soát lượt 2, C-17). Lượt chạy lại k = 1 không ra kết luận vì R0 đòi k ≥ 3, lẻ (soát C2, F-09); bản đầu của bảng này tính k = 1 nên dự phòng thấp hơn thật khoảng 7 USD.

## Bảng (đầu ra nguyên văn)

## Đo được (chạy khô, k = 1)

| Hệ | Lời gọi pilot | Ký tự pilot | Lời gọi H1 | Ký tự H1 | Ký tự TB / lời gọi (H1) |
|---|---|---|---|---|---|
| EACL | 0 | 0 | 4 | 3392 | 848 |
| B1 | 16 | 430580 | 104 | 3749982 | 36058 |
| B2+ | 16 | 39418 | 104 | 274414 | 2639 |
| B2 | 16 | 19947 | 104 | 135269 | 1301 |
| B2″ | 16 | 55578 | 104 | 379454 | 3649 |
| B2-oracle | 16 | 25828 | 104 | 171169 | 1646 |

max_tokens = 1024

## Ước lượng: pilot k = 5 + lượt chính H1 k = 3 (USD)

| Kịch bản | EACL | B1 | B2+ | B2 | B2″ | B2-oracle | Tổng cổng | Tổng cổng + dự phòng cổng | Tổng mọi hệ | Dự phòng | Tổng mọi hệ + dự phòng |
|---|---|---|---|---|---|---|---|---|---|---|---|
| tối thiểu | 0.01 | 9.13 | 0.88 | 0.53 | 1.14 | 0.62 | 10.02 | 18.46 ⚠ vượt trần | 12.32 | 10.29 | 22.61 ⚠ vượt trần |
| kỳ vọng | 0.03 | 11.31 | 1.40 | 0.99 | 1.72 | 1.10 | 12.74 | 23.44 ⚠ vượt trần | 16.56 ⚠ vượt trần | 13.78 | 30.34 ⚠ vượt trần |
| xấu nhất | 0.14 | 21.88 | 5.37 | 4.69 | 5.90 | 4.87 | 27.40 ⚠ vượt trần | 50.20 ⚠ vượt trần | 42.86 ⚠ vượt trần | 35.25 | 78.11 ⚠ vượt trần |

## Ước lượng: pilot k = 5 + lượt chính H1 k = 5 (USD)

| Kịch bản | EACL | B1 | B2+ | B2 | B2″ | B2-oracle | Tổng cổng | Tổng cổng + dự phòng cổng | Tổng mọi hệ | Dự phòng | Tổng mọi hệ + dự phòng |
|---|---|---|---|---|---|---|---|---|---|---|---|
| tối thiểu | 0.02 | 14.24 | 1.35 | 0.82 | 1.75 | 0.96 | 15.60 ⚠ vượt trần | 29.62 ⚠ vượt trần | 19.13 ⚠ vượt trần | 17.10 | 36.23 ⚠ vượt trần |
| kỳ vọng | 0.04 | 17.62 | 2.16 | 1.52 | 2.64 | 1.69 | 19.82 ⚠ vượt trần | 37.60 ⚠ vượt trần | 25.67 ⚠ vượt trần | 22.89 | 48.56 ⚠ vượt trần |
| xấu nhất | 0.23 | 34.01 | 8.24 | 7.18 | 9.04 | 7.46 | 42.48 ⚠ vượt trần | 80.36 ⚠ vượt trần | 66.16 ⚠ vượt trần | 58.55 | 124.71 ⚠ vượt trần |

Trần: 15 USD cho toàn bộ G7 (pilot + chính). Kịch bản 'xấu nhất' là cận trên chỉ khi ký tự / token ≥ 1,5 — chưa đo bằng tokenizer thật.

## Đọc bảng

1. **B1 là gần hết chi phí.** Trung bình 36.058 ký tự / lời gọi trên H1 (cửa sổ bản ghi thô từng ca). B2+ 2.639; EACL chỉ gọi model 4 lần trên 104 ca (các route còn lại tất định).
2. **Trong trần 15 USD (kỳ vọng) chỉ khi đủ cả ba:**
   - chạy riêng hệ cổng (EACL, B1, B2+);
   - lượt chính k = 3;
   - KHÔNG phải chạy lại.
   - Khi đó: 12,74 USD (tối thiểu 10,02). Thêm dự phòng một lần chạy lại trọn lượt ⇒ 23,44 USD, vượt trần.
3. **Pilot chọn k = 5** (ADR-0006 §6) thì riêng hệ cổng đã 19,82 USD (kỳ vọng): vượt trần.
4. **Chạy cả ba hệ chỉ-báo-cáo** (B2, B2″, B2-oracle) thêm khoảng 3,8 USD (k = 3, kỳ vọng) ⇒ 16,56 USD, vượt trần. Theo §9: không chạy phần vượt; cắt hệ chỉ-báo-cáo không đổi phán quyết.
5. **Bất định lớn nhất là ký tự / token.** Kịch bản "xấu nhất" (1,5 ký tự / token, ra = max_tokens) gấp khoảng 2,2 lần kỳ vọng.
   - Pilot (≈ 2,1 USD hệ cổng, kỳ vọng) sẽ ĐO được usage thật.
   - Sau pilot: tính lại bảng này bằng tỉ lệ đo được, TRƯỚC lượt chính.
6. **Trần là của từng lần chạy `g7run`** (§9, C-07). Đặt `-budget` của lần sau = 15 − tổng `spent_usd` các lần trước.
7. **Tập của lượt chính có thể đổi** (D-H1-3: H1 đã lộ cho phiên viết mã). Một bộ giữ lại mới cùng cỡ và cùng cửa sổ B1 thì chi phí xấp xỉ bảng trên; cỡ khác thì nhân theo số ca.

## Đề xuất (chủ dự án quyết, D-G7-18)

- **A — giữ trần 15 USD:**
  - chỉ hệ cổng; pilot k = 5, lượt chính k = 3; dừng sau pilot nếu bảng tính lại vượt trần;
  - k = 5 hay một lần chạy lại thì không đủ tiền ⇒ G7 tối đa PARTIAL (§1: cắt vì trần).
- **B — nâng trần:** hệ cổng k = 3 + dự phòng một lần chạy lại trọn lượt cần khoảng 23,44 USD (kỳ vọng); mọi hệ k = 3 + dự phòng khoảng 30,34 USD. Trần 25 USD chỉ đủ cho hệ cổng, và sát.
- **C — giảm chi phí B1 bằng cách đổi biên B1:** KHÔNG khuyến nghị. Biên B1 đã chốt (ADR-0005 §2); đổi vì tiền là đổi đối tượng đo.

Khuyến nghị: **A** cho tới khi pilot đo được token thật, rồi quyết lại bằng số đo.
