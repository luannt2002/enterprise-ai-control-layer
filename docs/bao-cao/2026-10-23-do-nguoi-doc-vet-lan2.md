# Đo "người soát CHỈ ĐỌC VẾT chỉ đúng tầng sai đầu tiên" — lần 2 (23/10/2026 chiều), bộ thẻ mới

> Trên dữ liệu giả lập, chưa có người nghiệp vụ duyệt (D-RM-03).
> Đo lại vế còn thiếu của cổng ra M1 (`docs/ROADMAP-DAY-DU.md` §5b) và dòng "Vết / debug" của CỔNG TRƯỚC AI: người soát CHỈ ĐỌC VẾT chỉ
> đúng tầng sai đầu tiên ở ≥ 90% ca đỏ. Lệnh chủ dự án 23/10 chiều, mục (4): bộ thẻ mới → đo lại; commit cả thẻ lẫn câu trả lời của
> người đọc. Nhãn: [ĐO] đã chạy lệnh · [REPO] tài liệu ghi · [SUY] suy luận.

## 1. Kết luận

**KHÔNG ĐẠT.** [ĐO] Người đọc mù chỉ đúng tầng sai đầu tiên ở **30 / 38 = 78,9%** thẻ đỏ (ngưỡng ≥ 90%), và nói đúng "không sai" ở
**38 / 38** thẻ xanh (0 báo nhầm). Sáu tầng đúng trọn (S2, S3, S4, S5, S6, S9: 5 / 5 mỗi tầng); hai tầng sai trọn: **S10 0 / 5** (cả năm
đọc thành S8) và **S8 0 / 3** (cả ba đọc thành S6). Cả tám nhãn sai đều kèm độ tin "thấp". M1 CHƯA RA.

## 2. Cách đo

| | |
|---|---|
| Mã | origin/main `086fff9` (R2-03 và C8–C11 đã gộp). Mọi binary dựng cùng `-ldflags -X main.buildVersion=…` để cột "mã" của bảng vết giống hệt ở mọi thẻ |
| Gieo lỗi | Một agent riêng (biết đáp án) gieo MỘT lỗi nhỏ, thực tế ở mỗi tầng S2, S3, S4, S5, S6, S8, S9, S10 (S0, S1 không có vết theo tenant / không gieo; S7 là RAG, ngoài MVP). Mô tả từng lỗi: `do-vet-2026-10-23/gieo-loi.md` |
| Ca | 474 câu của bộ 117 / H1 / H3 / H4 qua binary gốc (423 đạt, khớp 100% danh sách đạt / đỏ của cổng §6) và qua từng binary lỗi; giữ ca đỏ mà bước lệch đầu tiên của vết (so vết JSON từng bước với bản gốc) ĐÚNG tầng gieo — ca lệch tầng khác bị loại (S2 loại 171, S5 loại 33, S9 loại 1) |
| Thẻ | 38 đỏ (5 mỗi tầng, S8 chỉ 3: trong 423 ca đạt chỉ 3 ca đi nhánh RAG thoát sớm) + 38 xanh đối chứng; thứ tự trộn theo băm, tên ẩn danh `the-001` … `the-076`; mỗi thẻ: lớp ca, câu hỏi, vai, các khoá kỳ vọng chép nguyên văn từ tệp ca, câu trả lời của server, bảng vết `eais trace` |
| Người đọc | Một agent khác (sonnet), CHỈ đọc thư mục thẻ; với mỗi thẻ: đúng kỳ vọng (`KHONG_SAI`) hay bước sai đầu tiên, lý do trích từ vết, độ tin, vết đủ / thiếu |
| Chấm | Bảng ánh xạ kín của người gieo (`ban-do.tsv`), so từng thẻ (`cham.txt`) |

## 3. Kết quả theo tầng [ĐO]

| Tầng gieo | Lỗi gieo (tóm) | Đúng | Người đọc nói |
|---|---|---|---|
| S2 chuẩn hoá | "3.850 lít" đọc thành thập phân kiểu Anh | 5 / 5 | — |
| S3 thực thể | bí danh trùng chọn ứng viên đầu thay vì báo mơ hồ | 5 / 5 | — |
| S4 ý định | đảo thứ tự ưu tiên ý định | 5 / 5 | — |
| S5 định tuyến | phạm vi chỉ xét máy, không xét dự án cha | 5 / 5 | — |
| S6 thực thi | biên dung sai: chênh đúng bằng dung sai coi là xung đột | 5 / 5 | — |
| S8 sinh câu | nhánh RAG thoát sớm trả OUT_OF_SCOPE thay vì INSUFFICIENT_EVIDENCE | **0 / 3** | S6 (3) |
| S9 kiểm đầu ra | ngày cuối của kỳ bị từ chối (DATE_UNBOUND) | 5 / 5 | — |
| S10 trả lời | cắt câu trả lời ở 120 ký tự rồi thêm "…" | **0 / 5** | S8 (5) |

## 4. Vì sao trượt: hai khoảng trống của vết

Người đọc nêu đúng chỗ vết thiếu ở cả tám thẻ (trường "vết thiếu"), và người gieo lỗi đã ghi điều (1) trước khi có kết quả đọc:

1. **S8 không xuất bản nháp ra khỏi `Handle`.** Bảng vết ở S8 chỉ ghi "model diễn đạt" / "mẫu câu của tenant …" / "model chưa được gọi"
   kèm outcome; chữ câu trả lời chỉ hiện ở S10. Câu giao bị cắt ở S10 vì vậy không phân biệt được với lỗi ở S8.
2. **Nhánh RAG thoát sớm không ghi lý do thoát và tên mẫu.** S6 / S7 / S8 "skipped" rồi có outcome, nên lỗi chọn sai outcome ở nhánh này
   bị quy cho S6.

Thêm (người đọc ghi, không làm sai nhãn ở lần này): S9 không xuất bản nháp bị loại; dấu "BƯỚC LỆCH ĐẦU TIÊN" của bảng vết đánh S6 ở bốn
lượt chỉ có xung đột dữ liệu HỢP LỆ. [ĐO, người gieo] `eais trace diff` cho tầng khác tầng gieo ở 13 / 38 cặp vì nó quy Decision về bước theo
quy ước (verdict → S9, outcome → S6, bằng chứng → S7, câu → S8).

## 5. So với lần 1 (22/10)

Lần 1: 29 / 33 = 87,9% trên ca đỏ TỰ NHIÊN của H1 + H3 (dồn về S4; "luôn đoán S4" được 84,8%). Lần 2 gieo đều theo tầng nên lộ hai tầng
mà lần 1 không có ca. Hai lần không cùng mẫu số, không cộng dồn.

## 6. Hệ quả và việc kế

- M1 CHƯA RA; dòng "Vết / debug" của CỔNG TRƯỚC AI vẫn KHÔNG ĐẠT ở vế đọc vết.
- Việc kế (đang làm, nhánh `m1/vet-s8-s10`): S8 xuất bản nháp (cách sinh, số ký tự, băm ngắn, đoạn đầu) và lý do + kết cục của nhánh RAG
  thoát sớm; S9 xuất số ký tự + băm của bản nháp bị loại; S10 xuất số ký tự + băm của câu giao. Rồi đo lại bằng thẻ MỚI, người đọc MỚI
  (người viết mã không được đọc thư mục thẻ của lần này).

## 7. Giới hạn

- Một người đọc (agent), một bộ thẻ; S8 chỉ 3 thẻ. Đỏ lệch về bộ H3 (20 / 38) và lớp representation (10 / 38); xanh đều hơn.
- [ĐO, người gieo] Chấm đỏ / đạt bằng bản Python của `evals.RunWin` (khớp 423 / 423 đạt và 51 / 51 đỏ với cổng §6), không chạy RunWin của Go
  trên câu trả lời HTTP; binary lỗi chưa chạy lại lần hai để kiểm tất định riêng.
- Chưa loại trừ tín hiệu rò tinh trong thẻ ngoài những thứ đã kiểm (cột "mã" giống hệt, thời gian trung vị như nhau, không id ca / tên bộ /
  nhãn trong thẻ).

## 8. Tệp (thư mục `docs/bao-cao/do-vet-2026-10-23/`)

`the/the-001.md` … `the-076.md` (thẻ người đọc đã đọc) · `doc-mu.json` (câu trả lời của người đọc, nguyên văn) · `ban-do.tsv` (đáp án: thẻ →
id ca, bộ, đỏ / đạt, tầng gieo, mã lỗi) · `gieo-loi.md`, `gieo-loi.json` (mô tả và báo cáo của người gieo, nguyên văn; đường dẫn máy đã thay)
· `cham.txt` (bảng chấm).
