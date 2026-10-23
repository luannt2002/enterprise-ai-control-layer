# Câu hỏi chờ chủ dự án — đọc sau

> Tệp sống. Chủ dự án dặn (22/10/2026 17:0x): làm tiếp MVP liên tục, **không hỏi giữa chừng**; câu cần quyết ghi ra để đọc sau.
> Mỗi câu có dòng gốc ở `docs/CHO-QUYET-DINH.md` (cột "Trả lời" để trống tới khi chủ dự án trả lời) hoặc ở mục câu mở của hợp đồng việc.
> Trong lúc chờ, BUILD chạy theo **mặc định an toàn, đảo lại được** ghi ở cột "Đang chạy".
> Mọi kết luận: **trên dữ liệu giả lập, chưa có người nghiệp vụ duyệt** (D-RM-03).

Cập nhật: 23/10/2026 10:0x. Câu mở Q-P1-1…13 và Q-S9-1…8 gom ở `docs/CHO-QUYET-DINH.md` mục 5 (trả lời ở đó). Mới sáng 23/10:
Q-P1-10…13 (từ soát độc lập bước K1); Q-S9-4 chuyển lên nhóm "đang chặn" (mặc định của nó chặn nửa đường số liệu của phép đo S9).

## 1. Đang chặn một khối MVP

| Câu | Hỏi gì | Chặn gì | Đang chạy | Khuyến nghị |
|---|---|---|---|---|
| **D-AI-02** | Dòng S4 của CỔNG TRƯỚC AI trượt trên H4 (S4 là tầng sai đầu tiên ở 10 / 119 = 8,4%, ngưỡng 5%; 3 câu sai nghĩa không nhãn). Làm gì tiếp? | M3; dòng S4 của M5 | M3 ghi "chưa RA"; không đổi mã S4 trên main; chưa mở bộ giữ lại mới | **a**: sửa bằng luật các cụm C3 / C4 / C5 / N1 (ADR-0010), chấm trên một bộ giữ lại MỚI soạn độc lập (H6) |
| **D-AI-03** | Dòng S7 (Ngữ cảnh / RAG) có nằm trong cổng ra M3 / M5 của MVP không? (lệnh ca cấm làm RAG, cổng ra M3 lại liệt kê S7) | lời tuyên bố M3 / M5 RA | S7 ghi "CHƯA CÓ BỘ CA ĐỘC LẬP"; không tuyên bố M3 / M5 RA; không mở việc RAG | **a**: S7 ngoài MVP, ghi rõ "MVP không gồm câu hỏi quy trình (RAG)"; đo S7 ở nấc RAG-0 |
| **D-OP-9** | Nhóm nghìn bằng dấu cách ("bể 2 500 lít") — làm lại cách sửa nào | dòng S2 của CỔNG TRƯỚC AI (KHÔNG ĐẠT) | lỗi đã đo, giữ nguyên | xem dòng D-OP-9 ở `docs/CHO-QUYET-DINH.md` |
| **D-OP-11** | Hỏi lại khi bí danh mơ hồ: liệt kê ứng viên nào | dòng S3 (KHÔNG ĐẠT: H3-538, 539, 543) | test đang ghim hành vi hiện tại | **b** (theo dòng D-OP-11) |
| **D-OP-18** | Câu "ghi nhận / nhận xét + giá trị" ("Ghi nhận EX-03 tiêu thụ 3782 L dầu hôm nay") là lệnh ghi hay câu hỏi? | vòng cuối của nợ S4 sau H4 | mã ở nhánh máy, không gộp; main giữ hành vi cũ | **c**: câu có giá trị cụ thể là lệnh ghi (hỏi lại / từ chối kèm "trợ lý chưa ghi số liệu"); câu không có giá trị là câu hỏi |
| **Q-S9-4** | Cho làn 1 thêm MỘT điểm cắm bản nháp ở đường số liệu (giấy đo S9 §5.3; `nil` = không đổi hành vi)? | nửa đường số liệu của phép đo dòng S9 (M5): không có điểm cắm thì bộ chạy không thay được bản nháp ở đường số liệu, vòng kiểm hình dạng chỉ thấy phần gói có trong phản hồi | chưa mở mã điểm cắm; bộ ca và đường tri thức làm trước | **a**: có, kèm test chứng minh 117 / H1 / H3 0 dòng khác khi `nil` |

## 2. Không chặn — đang chạy theo mặc định

| Câu | Hỏi gì | Đang chạy |
|---|---|---|
| D-OP-16 | Điều kiện (ii) của D-OP-15 có áp cho nhánh từ khoá yếu không | (a) |
| D-OP-17 | Tin chat "2k" có tính là ước lượng không | giữ "2k" như cũ; G1 giữ danh sách lệch đã biết |
| Q-P1-1…Q-P1-8 | Câu mở của Phase 1 (khối M4): vai được tạo phiếu, ngưỡng mở phiếu, chỉ số đầu tiên, huỷ phiếu nhầm, audit lần gửi lại, bên kích hoạt tự động, thứ tự C4b, cửa vào của demo | mặc định ở `docs/viec/1-hanh-dong/HOP-DONG.md` §14; mã M4 viết theo các mặc định này |
| Q-P1-10…Q-P1-13 (mới 23/10) | Từ soát độc lập bước K1: người nhiều vai (quyền và phạm vi lấy theo hợp các vai hay cùng một vai), `retry` của kết cục UNKNOWN (hai chỗ hợp đồng nói khác nhau), thứ tự R4 / R5, bước C6 của lượt DONE sau phân loại lại | cột "Đang chạy" ở `docs/CHO-QUYET-DINH.md` mục 5; Q-P1-10 phải quyết trước khi cấp nhiều vai (Phase 5) |
| Q-S9-1…3, Q-S9-5…8 | Câu mở của giấy đo S9 (khối M5): dạng viết ngoài danh sách có tính là ĐÚNG không, làm tròn / đổi đơn vị, phạm vi của S9, cách chấm (cận tin cậy hay đếm thô), lọt một chiều có làm cổng đỏ, vòng kiểm hình dạng trước băm, sửa S9 cho dạng đã biết trước lần chạy | mặc định ở `docs/viec/m5-cong-truoc-ai/S9-HOP-DONG-DO.md` §9 |
| Q-0A-1…Q-0A-12 | Câu mở của Phase 0a (khối M2): ngưỡng GA6, hậu tố "k", tổng ước lượng, áp đính chính, cửa sổ trùng, người nhận hàng chờ, vai gửi báo cáo, lịch ca, tin nhiều máy, nhóm nghìn, kênh form, write-set | mặc định ở `docs/viec/0a-thu-bao-cao-ngay/HOP-DONG.md` §15 |

## 3. BUILD đã tự quyết khi chờ (đảo lại được)

| Lúc | Quyết định | Căn cứ | Đảo lại thế nào |
|---|---|---|---|
| 22/10 17:0x | Làm tiếp sau giờ ca, nhịp 30 phút suốt 22–23/10 | lời chủ dự án 16:5x, 17:0x | xoá cron của phiên |
| 22/10 17:0x | Nợ S4 sau H4 không gộp sau hai vòng soát KHÔNG ĐẠT | luật dừng brief BUILD §9.6 | trả lời D-OP-18 rồi chạy vòng cuối |
| 22/10 17:2x | Trần agent cùng lúc 3 → 5 | lời chủ dự án 17:2x | — |
| 22/10 17:3x | "Lượt" của vế 100% dòng Vết = lượt đã xác thực (lượt 401 chỉ bộ đếm + log) | chưa có tenant để ghi vết theo RLS | đổi định nghĩa trong `docs/bao-cao/CONG-TRUOC-AI.md` |
| 22/10 18:2x | Ngày 23/10 cũng dừng lúc 21:30 như tối 22/10 | chủ dự án mới nói giờ của tối 22/10 | nói giờ khác |
| 22/10 21:2x | Giấy M2 (Phase 0a) làm tiếp vòng 7 dù đã quá 3 vòng soát | luật dừng §9.6, ngoại lệ mất dữ liệu: R6-01 thuộc họ "rơi khỏi tổng im lặng" | khoá giấy ở vòng 6, ghi R6-01 thành nợ |
| 22/10 21:2x | R2-03 (vết xuất phương án đã xét) làm tiếp vòng 4 dù đã quá 3 vòng soát | luật dừng §9.6, ngoại lệ an ninh: N1' | bỏ R2-03, giữ vết như main |
