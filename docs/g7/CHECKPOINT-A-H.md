# Checkpoint G7 — A đến H (20/10/2026 tối)

> Theo yêu cầu của chủ dự án: chuẩn bị G7 mà **không gọi model thật, không chi tiền**. Làm xong A–H thì **DỪNG**, chờ duyệt.
> Mọi câu cần quyết: `docs/CHO-QUYET-DINH.md`.
>
> **Đính chính 20/10 tối (bản soát độc lập `docs/g7/REVIEW-D5.md`):**
> - Ba ca 083, 107, 108 có kỳ vọng do chính phiên viết mã "trọng tài" **sau** lần đo đầu (09:03), **trước** khi băm bộ ca (09:15).
> - Vì vậy G2, G5, G6 chỉ đạt **có điều kiện**. Đạt vô điều kiện: **3/7** (G1, G3, G4).
> - Chờ chủ dự án quyết D-G7-11 (`docs/CHO-QUYET-DINH.md`).

## A. Phân lớp ADR-0002 (D7–D17)

Chi tiết: `docs/adr/0002-win-quyet-dinh-trien-khai.md`, mục "Phân lớp".

| Tầng | Quyết định |
|---|---|
| CORE INVARIANT | D10 (nguyên lý: tài liệu không có thẩm quyền về số liệu); D11a (RLS FORCE + GUC từ Principal); D12 (ràng buộc claim 6 chiều) |
| Lõi chung (cơ chế đúng / sai) | D15 (số âm); `Catalog.Compare` của D13; `ToneKey` của D9 |
| DOMAIN PACK / TENANT CONFIG | giá trị và câu chữ: D7 `diesel_fuel`, D10 `quantity_markers`, D13 `comparison_markers` + mẫu, D14 `confirm_markers`, D17 mẫu "Trích văn bản quy trình" |
| WIN-ONLY IMPLEMENTATION | heuristic: D7, D8, D13, D14 (đều **chỉ chạy khi pack bật**); D9 truy xuất; D16 dữ liệu thử |
| DEV-ONLY | D11b (`trust` + loopback + tmpfs) |

Kết luận:
- **Không quyết định nào sai tầng tới mức phải dời mã ngay.**
- Một nợ đặt chỗ: D9 viết cứng từ dừng / cụm nghi vấn tiếng Việt trong `intelligence/rag`, áp mọi tenant. Câu chờ quyết: D-OP-6.

## B. Hợp đồng benchmark G7 (đề xuất đóng băng)

`docs/g7/HOP-DONG-G7.md` + `docs/adr/0005-g7-hop-dong-va-sua-d5.md`. Hợp đồng đóng băng:
- bộ ca + băm;
- dữ liệu + băm;
- 3 hệ (EACL, B1, B2) và khác biệt truy cập dữ liệu;
- model và tham số;
- k = 3, trung vị, ANY-RUN cho an ninh;
- cách chấm;
- 13 nhãn lỗi;
- chi phí, độ trễ;
- xử lý lỗi nhà cung cấp, luật lượt vô hiệu;
- định dạng bằng chứng thô;
- luật WIN / PARTIAL / FAIL.

**Phát hiện chặn:** D5 đòi temperature 0, nhưng model hiện hành trả 400 với mọi temperature khác mặc định (tài liệu chính thức, trích trong hợp đồng mục 4.1). Phải sửa D5 trước khi chạy (D-G7-1).

## C. Soát độc lập D5

`docs/g7/REVIEW-D5.md`: agent riêng, chỉ đọc, không viết công thức hay harness. Kết luận:
1. **Thiên vị EACL: có.**
   - 117/117 là trong mẫu: điểm đi 76 → 105 → 117 nhờ sửa mã theo chính các ca.
   - 3 kỳ vọng bị phiên viết mã trọng tài trước khi băm. Với baseline, đây là tuning không đối xứng.
   - 37 câu diễn đạt mới (chưa độc lập hẳn) đạt 31/37; nhóm đồng nghĩa "dùng" 0/4.
2. **Ceiling effect: có.** EACL ở trần 100%, nên D5 trên bộ dev thực chất chỉ đo "baseline ≲ 80%".
3. **Baseline B2 như hiện thực có phần là hình nộm.** Đoạn thiếu bí danh nên trần chỉ 4/13 (`entity`) và 5/13 (`ambiguity`) dù model hoàn hảo; thêm bí danh thì lên 12/13 và 13/13. `representation` và `semantic` có thể hoà.
4. **Trung vị của 3 lượt che được lỗi chập chờn.**
5. **ANY-RUN cho an ninh đã có trong mã**; phạm vi canary còn mỏng.
6. **20 điểm là quy ước**, khoảng 1–2 sai số chuẩn với n = 12–18.
7. **Công bằng dữ liệu:** EACL có lợi ngoài ý muốn ở đồng hồ và quy ước mặc định; baseline có lợi ở bản ghi đã chuẩn hoá.
8. **Bộ chấm nội dung có lỗ ở cả hai phía.**

14 đề xuất sửa (cộng 1 của phiên chính) thành câu D-G7-9; điểm yếu đồng nghĩa: D-G7-10; 3 kỳ vọng trọng tài: D-G7-11.

## D. Số lời gọi model chính xác

Đo bằng chạy khô đúng đường mã thật tới máy chủ giả trên máy. Chi tiết: `docs/g7/CHI-PHI-G7.md`.

| | k = 3 |
|---|---|
| EACL (7 ca RAG / HYBRID mỗi lượt) | **21** (đo) |
| B2 RAG đơn giản | **351** (đo) |
| **Tổng EACL + B2** | **372** (đo) |
| B1 LLM trực tiếp (chưa có mã) | 351 (theo thiết kế) |
| **Tổng EACL + B1 + B2** | **723** |
| Thử khói | ~3 |

## E. Token và chi phí (ước tính)

| Model | Tổng B1 + B2 + EACL | Chỉ B2 + EACL |
|---|---|---|
| `claude-haiku-5-5` | ≈ 0,3–0,6 USD | ≈ 0,03–0,06 USD |
| `claude-sonnet-5-5` | ≈ 5–10 USD | ≈ 0,5–1,2 USD |
| `claude-opus-5-5` | ≈ 10–20 USD | ≈ 1,1–2,4 USD |

- Giá chính thức tra 20/10/2026. Token suy từ số ký tự đo được với tỉ lệ 2,0–3,0 ký tự / token.
- Token thật: **CHƯA ĐO** (cần khoá).
- Model định dùng: **đề xuất `claude-sonnet-5-5`** (D-G7-2).
- Giảm lời gọi mà không làm yếu phép đo: Batch API (−50% tiền) hoặc cache phần dữ liệu thô của B1. Không giảm k.

## F. Lệnh sẽ chạy khi được cấp khoá

`docs/g7/HOP-DONG-G7.md` mục 12. Hai cờ `-smoke` và `-out` **chưa có** (mục H).

## G. Tệp bằng chứng sẽ sinh

`docs/g7/runs/<ngày>-<revision>/`:
- `manifest.json`;
- `<hệ>-run<i>.jsonl` (mỗi ca một dòng, đủ trường ở hợp đồng mục 11);
- `ket-qua.md`.

## H. Blocker còn lại

| # | Blocker | Loại | Mở khi |
|---|---|---|---|
| 1 | D-G7-1…7 và 9–11: sửa D5, model, trần chi phí, B1, bộ giữ lại, cách chấm, đóng băng hợp đồng; gói đề xuất của bản soát; câu đồng nghĩa "dùng"; 3 kỳ vọng trọng tài | chủ dự án quyết | — |
| 2 | Client model: bỏ `temperature`; thêm `thinking` / `effort` từ cấu hình; thử lại có lùi + ghi lần thử | mã, không cần model | sau D-G7-1 |
| 3 | `g7run -smoke` (vài lời gọi kiểm tham số được nhận) và `-out` (ghi `manifest.json` + JSONL đủ trường + `ket-qua.md`) | mã, không cần model | sau D-G7-7 |
| 4 | Baseline **B1** + bảng kỳ của từng ca (người không đọc mã gán) | mã + tài liệu | nếu D-G7-4 = có |
| 5 | Bộ ca giữ lại ≥ 96 ca | tài liệu (agent không đọc mã) | nếu D-G7-5 = có |
| 6 | Gán nhãn lỗi (luật máy + người duyệt); báo cáo ca chập chờn; tính tiền theo bảng giá đóng băng | mã | sau D-G7-7 |
| 7 | Soát độc lập mã ngày 20/10 (dừng dở lúc 10:33) | soát | D-OP-5 |
| 8 | Khoá model đặt ở máy chạy (không qua repo / chat) | chủ dự án | sau D-G7-7 |

Theo hợp đồng mục 10: mọi blocker mã phải xong và có test **trước** lượt chạy thật. Sửa giữa benchmark thì lượt đó vô hiệu.
