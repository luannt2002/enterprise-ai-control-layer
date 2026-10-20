# 08 — Risk Register

> Trạng thái: bản lập kế hoạch 20/10/2026. Xác suất (P) và tác động (I) là **đánh giá định tính** của lead, không phải số đo: Thấp · TB · Cao · Rất cao.
> Chủ: lead = phiên Opus điều phối; CDA = chủ dự án.
> Cập nhật khi kết thúc mỗi phase.

| ID | Rủi ro | P | I | Giảm thiểu | Test / bằng chứng phát hiện | Chủ | Trạng thái |
|---|---|---|---|---|---|---|---|
| R-01 | **Phình phạm vi**: thêm tính năng không truy được về lỗi nghiệp vụ | Cao | Cao | kiểm soát thay đổi mặc định DEFER (`05` §2.10); backlog đóng; stop rules Blueprint §48 | mỗi task trỏ về mã `V3-…` ở `03`; review từ chối task không trỏ được | lead | mở |
| R-02 | **Trừu tượng sớm**: đưa khái niệm vào core khi mới có một vertical | TB | Cao | vertical trước; core chỉ nâng ở R7 kèm ADR; ghi lý do mọi sửa vào thư mục chung | `architecture_test.go` mở rộng cho tên domain Customer Ops; `git diff --stat` thư mục chung trong báo cáo task | lead | mở |
| R-03 | **Thuế tích hợp** cao hơn tự gọi model + vài cổng | TB | Cao | đo từ đầu: dòng mã adapter, số tệp chung bị sửa, giờ công (R6-04) | bảng công tích hợp ở báo cáo R6 | lead | mở |
| R-04 | **AI ảo giác** trong brief | Cao | Cao | brief có cấu trúc, mỗi khẳng định mang `evidence_ids`; validator tất định; không căn cứ ⇒ chặn hoặc đánh dấu | model giả đối kháng (gạch không id, id giả, số lệch); đếm chặn oan | Sonnet cài, lead soát | mở |
| R-05 | **Chèn lệnh** qua activity, document hoặc phản hồi công cụ | TB | Cao | nội dung truy xuất là DỮ LIỆU; quyền và hành động quyết ngoài model | V3-SEC-04/05, V3-FAIL-03/04 | lead | mở |
| R-06 | **Vượt quyền**, gồm **lối vào thay thế** (bài học hệ A) | TB | Rất cao | một executor duy nhất; `AuthorizedAction` chỉ gói policy dựng được; app kiểm quyền có thẩm quyền; EACL gọi app bằng token của người dùng | V3-SEC-03/06/11; test kiến trúc "một đường ghi"; bảng cho phép/từ chối × route | lead | mở |
| R-07 | **Chi phí model** vượt tầm hoặc chưa có ngân sách | TB | TB | ước offline trước (R1-04); trần cứng trong harness; route tất định 0 lời gọi; model rẻ nhất đủ dùng | bảng token ước lượng; số tiền thật ghi theo lần chạy | CDA quyết, lead đo | mở (D-V3-01) |
| R-08 | **Sân nhà**: app tham chiếu viết ra để hợp EACL, làm thuế tích hợp trông nhỏ | Cao | Cao | app là tiến trình/DB/REST riêng, cấm import hai chiều; hợp đồng API + fixture chốt **trước** mã EACL; baseline dùng cùng API; R8 nên là hệ ta không thiết kế cho EACL | ADR-0003; test kiến trúc; so ngày commit hợp đồng với mã EACL | lead | mở |
| R-09 | **Hành động trùng** khi thử lại hoặc mất phản hồi | TB | Cao | khoá idempotency hai phía; đọc lại theo khoá; máy trạng thái có `UNKNOWN` | V3-FAIL-09/10; test đua 10 yêu cầu đồng thời (R2-05) | Sonnet cài, lead soát | mở |
| R-10 | **Trạng thái cũ**: thực thi trên phiên bản khách/task đã đổi | TB | TB | đề xuất mang `version`; đọc lại trước khi thực thi; kiểm lại policy | V3-FAIL-06/12 | lead | mở |
| R-11 | **Gian lận benchmark**: đặt ngưỡng sau khi thấy kết quả; ca do người viết mã soạn; prompt baseline yếu cố ý | TB | Cao | ngưỡng + bộ ca + prompt đông cứng có băm trước khi chạy; tác giả ca không đọc mã; công bố cả kết quả xấu | băm trong `09` có trước lần chạy; `authored_by` | lead | mở |
| R-12 | **Rò rỉ chéo tenant** | Thấp | Rất cao | tenant chỉ từ token/Principal; app lọc tenant + test; canary B có "Acme" trùng tên | V3-AC01, V3-SEC-08, V3-FAIL-01 | lead | mở |
| R-13 | **Tuyên bố thành công giả** ("code có = xong") | TB | Cao | thang trạng thái `00` §3; người soát ≠ người làm; "IMPLEMENTED, NOT DONE" | mọi PASS có dòng trong `09` | lead | mở |
| R-14 | **Lệch tài liệu và runtime** | Cao | TB | R0-03 sửa README và đánh dấu tài liệu lịch sử; DoD đòi tài liệu khớp hành vi | đã thấy: `CLAUDE.md` §9, README, `04`/`05` cũ (`05` C1, C8) | CDA (CLAUDE.md), lead | mở |
| R-15 | **Phụ thuộc provider**: model không có hoặc đổi hành vi | TB | TB | gateway theo cấu hình; model giả cho mọi test kiểm soát; model thật chỉ cho đo giá trị; không có model thì DỪNG, không âm thầm đổi | `TestFromConfig`; nhật ký model/phiên bản trong báo cáo | lead | mở |
| R-16 | **Uốn Customer Ops theo pipeline Fuel**: ép domain mới vào khuôn của domain cũ | TB | TB | vertical riêng ở `tenants/customerops/`; chỉ dùng lại gói chung khi vừa khít | số và lý do sửa vào thư mục chung; R7 so hai bên | lead | mở |
| R-17 | **B0 một người đo** nên lệch (người đo cũng là người xây) | Cao | TB | B0 chỉ báo cáo, không làm cổng; giao thức và cỡ mẫu ghi rõ | phiếu bấm giờ + giới hạn trong báo cáo R6 | CDA | mở (D-V3-04) |
| R-18 | **Lộ dữ liệu riêng lên repo public** (tên hệ thống công ty cũ, đường dẫn máy, tài khoản) | TB | Cao | ẩn danh; nguồn gốc ở `docs/sources/` (ignore); quét tên riêng trước mỗi commit | `grep -rn -i -E -f docs/sources/eacl-v3/ten-rieng.txt docs/` ra 0 | lead | mở; quét 20/10 ra 0 trên `docs/v3/` |
| R-19 | **Model không tất định** dù temperature 0 | Cao | TB | k = 3 lấy trung vị; cổng tất định chạy bằng model giả; lưu số thô | lặp lại cùng ca, so phân phối | lead | mở |
| R-20 | **Dữ liệu giả lập (D2) không giống thật** | Cao | TB | mọi khẳng định ghi mức dữ liệu; ca đội đỏ độc lập; D3/D4 để sau | `00` §3 | lead | chấp nhận có ghi rõ |
| R-21 | **Tài nguyên máy dùng chung** | TB | Thấp | tối đa 3 agent; `nice`; PG trên tmpfs; không đo hiệu năng song song trên cùng DB | — | lead | mở |
| R-22 | **Dùng nhầm catalog khuôn của Blueprint** làm bộ ca/backlog thật | TB | TB | `00` §4.1; bản công khai gộp catalog thành nội dung riêng biệt (R0-04) | đếm mục riêng biệt trong bản công khai | lead | giảm (đã phát hiện) |
| R-23 | **Bộ ca không bắt hết lỗi** (đã có 2 ví dụ: ca 116, OBS-01) | Cao | TB | ca đội đỏ; test hồi quy ngoài bộ đông cứng cho mỗi lỗi tìm thêm; đột biến | R1-02, R1-05 | lead | mở |
| R-24 | **Câu hỏi tồn tại chưa trả lời:** AI có sẵn của chính nhà cung cấp CRM/ERP có làm EACL thừa không? Lịch sử dự án tự ghi "chưa được chứng minh" | TB | Rất cao | ngoài phạm vi MVP; báo cáo R6 phải ghi rõ giới hạn "chưa so với AI của nhà cung cấp"; xét thêm một đường so sánh khi tới R8 nếu tiếp cận được | dòng giới hạn trong báo cáo R6 | CDA | chấp nhận có ghi rõ |
| R-25 | **Cổng fail-open**: phụ thuộc lỗi thì cho qua (bài học hệ A) | TB | Cao | I13: lỗi ⇒ từ chối; test bơm lỗi cho kho quyền, policy, đồng hồ chi phí | V3-SEC-12 | lead | mở |
