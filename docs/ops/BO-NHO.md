# Bộ nhớ dự án EACL (bản công khai)

Đây là phần EACL của memory Claude Code ở máy gốc, đã bỏ tên tài khoản, đường dẫn máy và mọi thứ của dự án khác. Máy mới không cần chép tệp này vào memory: `CLAUDE.md` tự nạp, và `docs/ops/README.md` mục 5 có prompt mở phiên. Muốn có memory thì tạo một tệp memory trỏ về đây.

## Dự án

- **EACL — Enterprise AI Control Layer.** Controlled AI Integration for Enterprise Software. Mục tiêu: chuyển từ hệ lấy model làm trung tâm (RAG ở giữa) sang hệ lấy hệ thống làm trung tâm. Chứng minh bằng cổng có lệnh chạy, không bằng tài liệu.
- **Stack:** Go 1.26 + PostgreSQL 17. Python chỉ khi có bằng chứng ("Go first, Python by evidence"). Không Redis / Kafka khi chưa có số đo đòi.
- **Cách làm:** strangler từ dịch vụ trả lời của hệ cũ. Không viết lại từ trắng; mỗi tệp dời đúng lúc lát cần nó (`docs/02-migration-audit.md`).
- **Hướng đã chốt (19/10/2026):** WIN MVP trước (`docs/04-core-mvp.md`: chỉ QUERY, Fuel Discrepancy Resolution, tenant A + canary B, baseline RAG, 7 cổng), roadmap sau (`docs/05-master-roadmap.md`). Thứ gì không cần để thắng 7 cổng thì không làm.

## Luật làm việc chủ dự án đã chốt

- Repo **PUBLIC**: không dữ liệu thật, không link nội bộ, không transcript, không khoá.
- Commit / push chỉ khi chủ dự án yêu cầu.
- Báo tiến độ bằng **một con số %**, đếm cổng có lệnh chứng minh.
- Không viết con số chưa đo; chưa đo thì ghi "CHƯA ĐO".
- Mã và định danh tiếng Anh; comment, tài liệu, commit tiếng Việt có dấu; tên tệp tài liệu / script không dấu, gạch nối.
- Agent con:
  - model nhỏ, tối đa 3 cùng lúc;
  - tệp không giao nhau, scratch riêng;
  - lời khai phải đo lại;
  - người soát không được là người viết.

## Trạng thái 20/10/2026 (đã thay; hiện hành ở `docs/STATUS.md`)

- WIN: 3/7 cổng đạt vô điều kiện (G1, G3, G4); G2, G5, G6 có điều kiện vì 3 kỳ vọng do phiên viết mã trọng tài sau khi đo (D-G7-11); G7 chặn ở khoá model + chi phí.
- Scale smoke ~360 nghìn bản ghi đạt ở cả hai đường dữ liệu.
- 224 test xanh; 69 đột biến trong danh mục, đều bị bắt.
- Quyết định tự chốt chờ duyệt: `docs/adr/0002-win-quyet-dinh-trien-khai.md`.

## Bài học kỹ thuật dùng lại được

1. **Duyệt map của Go không có thứ tự.** Dùng nó để dựng danh sách bằng chứng thì trích dẫn đổi giữa hai lần chạy. Cộng số thực theo thứ tự duyệt map thì điểm "bằng nhau" lệch bit cuối và xếp hạng đổi. Luôn duyệt khoá đã sắp (`slices.Sorted(maps.Keys(m))`). Cổng phát lại (G6) bắt được lỗi loại này.
2. **Slice `nil` gửi sang pgx là NULL.** `cardinality(NULL) = 0` cho NULL chứ không cho true. Dùng `coalesce(cardinality($1::text[]), 0) = 0`.
3. **COPY không chạy trên bảng có RLS.** Nạp bằng `INSERT … SELECT * FROM unnest(...)` theo lô.
4. **Tiếng Việt:**
   - so khớp bỏ dấu làm "bí" = "bị". Giữ luật dấu: gõ có dấu thì khớp đúng dấu; chỗ đặt dấu thanh không tính ("hoá" = "hóa", `normalization.ToneKey`);
   - cụm nghi vấn ("làm gì", "bao lâu") không phải nội dung khi truy xuất;
   - dấu "-" ngay trước chữ số, sau khoảng trắng, là dấu âm; "EX-03" thì không.
5. **Cổng kiểm "số có mặt trong bằng chứng" không đủ.** Nó không bắt ngày bịa hay số đúng giá trị nhưng sai nghĩa. Cần ràng buộc claim đủ 6 chiều (chủ thể / chỉ số / kỳ / giá trị / đơn vị / nguồn) và bộ đếm số độc lập với bộ chuẩn hoá.
6. **Quy trình:**
   - test xanh ngay lần đầu chưa là bằng chứng — làm nó đỏ trên mã cũ trước;
   - đột biến do chính người viết test chọn không phủ biên — cần người khác đo lại;
   - 100% trên bộ ca cố định chưa chứng minh tổng quát hoá — cần ca đội đỏ mới, do người không đọc mã soạn.
