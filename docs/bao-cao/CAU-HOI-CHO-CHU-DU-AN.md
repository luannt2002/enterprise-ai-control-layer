# Câu hỏi chờ chủ dự án — đọc sau

> Tệp sống. Chủ dự án dặn (22/10/2026 17:0x): làm tiếp MVP liên tục, **không hỏi giữa chừng**; câu cần quyết ghi ra để đọc sau.
> Mỗi câu có dòng gốc ở `docs/CHO-QUYET-DINH.md` (cột "Trả lời" để trống tới khi chủ dự án trả lời) hoặc ở mục câu mở của hợp đồng việc.
> Trong lúc chờ, BUILD chạy theo **mặc định an toàn, đảo lại được** ghi ở cột "Đang chạy".
> Mọi kết luận: **trên dữ liệu giả lập, chưa có người nghiệp vụ duyệt** (D-RM-03).

Cập nhật: 23/10/2026 09:58. Sáng 23/10 chủ dự án đã trả lời mọi câu chặn MVP và nhóm câu đang chạy theo mặc định (`docs/CHO-QUYET-DINH.md`
mục 0, các dòng "23/10 sáng"). Từ 23/10 có luật mới ở brief §9: câu có mặc định an toàn và đảo lại được thì BUILD làm theo mặc định ngay, chỉ
dừng chờ khi tốn tiền, đổi / hạ ngưỡng hay cổng, xoá / ghi đè dữ liệu thật, đổi quyền.

## 1. Đang chặn một khối MVP

Không còn câu nào (23/10 sáng).

## 2. Không chặn — đang chạy theo mặc định

| Câu | Hỏi gì | Đang chạy |
|---|---|---|
| Q-P1-9 | Mở rộng write-set của làn 4 cho Phase 1 (`observability/traces/step.go`, một tệp mới ở `platform/pgstore/`, `cmd/eais/`, `scripts/dot-bien/`, sửa nhỏ `policy.yaml`) | có, theo từng tệp ADR-0011 đã liệt kê |
| Q-P1-10 | Người có nhiều vai: quyền và phạm vi lấy theo hợp các vai, hay phải cùng một vai | giữ như hiện tại (chưa có người nhiều vai); **phải quyết trước khi cấp nhiều vai (Phase 5)**; khuyến nghị: cùng một vai |
| Q-P1-11 | `retry` của kết cục UNKNOWN: `none` (bảng §2.4) hay `same_key` (§2.3, §9.2.2) | theo bảng §2.4; khuyến nghị: `same_key` |
| Q-P1-12 | R4 (không chênh lệch) hay R5 (đã có phiếu mở) thắng khi cả hai đúng | R4 trước, ghim bằng một ca |
| Q-P1-13 | Bước C6 của lượt DONE sau phân loại lại: "lệch" hay "ok" kèm lý do | "ok" kèm lý do, theo A29(b) |
| Q-0A-13…16 | Bản rời nhóm TR3; kế thừa thực thể xuyên lô; ai làm việc ngoài write-set làn 4; chống trùng đơn vị không có `sent_at` | mặc định ở `docs/viec/0a-thu-bao-cao-ngay/HOP-DONG.md` §15 (13: a; 14: a; 15: chờ, các phần đó giữ ĐỎ; 16: a) |
| D-AI-01 | Ngưỡng của CỔNG TRƯỚC AI | ngưỡng §9.3 giữ nguyên (không hạ) |

## 3. BUILD đã tự quyết khi chờ (đảo lại được)

| Lúc | Quyết định | Căn cứ | Đảo lại thế nào |
|---|---|---|---|
| 22/10 17:0x | Làm tiếp sau giờ ca, nhịp 30 phút suốt 22–23/10 | lời chủ dự án 16:5x, 17:0x | xoá cron của phiên |
| 22/10 17:0x | Nợ S4 sau H4 không gộp sau hai vòng soát KHÔNG ĐẠT | luật dừng brief BUILD §9.6 | (chủ dự án trả lời D-OP-18 = c ngày 23/10: gộp vào vòng sửa S4) |
| 22/10 17:2x | Trần agent cùng lúc 3 → 5 | lời chủ dự án 17:2x | — |
| 22/10 17:3x | "Lượt" của vế 100% dòng Vết = lượt đã xác thực (lượt 401 chỉ bộ đếm + log) | chưa có tenant để ghi vết theo RLS | (chủ dự án chấp nhận ngày 23/10) |
| 22/10 18:2x | Ngày 23/10 cũng dừng lúc 21:30 như tối 22/10 | chủ dự án mới nói giờ của tối 22/10 | nói giờ khác |
| 22/10 21:2x | Giấy M2 (Phase 0a) làm tiếp vòng 7 | luật dừng §9.6, ngoại lệ mất dữ liệu | (đảo ngày 23/10 theo luật mới của chủ dự án: giấy khoá ở vòng 6; vòng 7 dừng giữa chừng, bản vòng 7 giữ ở nhánh `m2/phase0a-vong7-bo`, không gộp) |
| 22/10 21:2x | R2-03 làm tiếp vòng 4 | luật dừng §9.6, ngoại lệ an ninh: N1' | (chủ dự án xác nhận ngày 23/10: mã, an ninh, được làm vòng 4) |
| 23/10 09:5x | Bộ đồ nghề H6: tài liệu nghiệp vụ lấy bản hiện hành, riêng `03-data-map.md` và ADR-0002 dùng bản lúc soạn H4 | bản hiện hành của hai tệp có thêm ghi chú về cách mã đang xử lý (nhóm nghìn; đính chính kiểm toán), người soạn không được thấy | dựng lại bộ đồ nghề trước khi băm H6 |
| 23/10 09:58 | Mã Phase 0a mở song song với việc soạn bộ ca độc lập Phase 0a (HOP-DONG 0a §14.1 điều kiện 5 đòi bộ ca đóng băng trước mã) | lệnh 23/10 mục F2 "mã Phase 0a theo giấy đã khoá"; giữ độc lập bằng: người viết mã không đọc thư mục bộ ca (kiểm nhiễm nhật ký), bộ ca băm và đóng băng TRƯỚC khi mã 0a lên main | dừng mã, chờ bộ ca đóng băng |
