# ADR-0005 — Đóng băng hợp đồng G7 và sửa phần "temperature 0" của D5

- **Ngày:** 20/10/2026
- **Trạng thái:** **ĐỀ XUẤT — chờ chủ dự án duyệt.** Chưa duyệt thì **không** gọi model thật.
- **Liên quan:** ADR-0001 D5, D6; `docs/g7/HOP-DONG-G7.md`; `docs/g7/CHI-PHI-G7.md`; `docs/g7/REVIEW-D5.md`; `docs/v3/05-R0-DEFINITION-FREEZE.md` §2.4–2.5.
- **Số hiệu:** ADR-0003 và ADR-0004 đã được bộ tài liệu V3 giữ chỗ, nên dùng 0005.

## Bối cảnh

D5 (ADR-0001) đòi "temperature 0; k = 3 lần, trung vị". Tài liệu chính thức của nhà cung cấp (Thinking, tra ngày 20/10/2026) ghi:

> "On … Claude Opus 5.5, … Claude Sonnet 5.5, Claude Sonnet 5, and Claude Haiku 5.5, non-default `temperature`, `top_p`, or `top_k` values return a 400 error on every request, regardless of whether thinking is used."

Client `models.Messages` đang gửi `temperature: 0`, nên mọi lời gọi tới model hiện hành sẽ lỗi 400. Lần chạy khô ngày 20/10 qua được chỉ vì máy chủ giả không kiểm tham số.

Theo luật của chủ dự án: muốn đổi D5 thì ghi ADR sửa đổi **trước** khi chạy, không đổi sau khi thấy kết quả.

## Quyết định đề xuất

1. **Bỏ yêu cầu temperature 0, thay bằng cố định mọi thứ khác** (phương án A bên dưới):
   - một model snapshot đã ghim, chung cho mọi hệ;
   - không gửi tham số sampling (dùng mặc định của API);
   - thinking cố định (Haiku 5.5 `disabled`, Sonnet 5.5 `between_tools`);
   - effort cố định;
   - giữ **k = 3 + trung vị**;
   - **liệt kê ca chập chờn** để trung vị không che lỗi.
2. **Giữ nguyên** mọi luật còn lại của D5: mọi lớp ≥ baseline; ≥ 4/8 lớp hơn ≥ 20 điểm %; EACL rò rỉ = 0 và claim không căn cứ = 0 (tính ANY-RUN).
3. **Đóng băng** `docs/g7/HOP-DONG-G7.md` (bộ ca, dữ liệu, hệ so sánh, tham số, chấm, tổng hợp, phân loại lỗi, chi phí, độ trễ, xử lý lỗi nhà cung cấp, định dạng bằng chứng, luật WIN / PARTIAL / FAIL). Ghi sha256 khi duyệt.

| Phương án | Là gì | Được | Mất |
|---|---|---|---|
| **A** (đề xuất) | model hiện hành, sampling mặc định, cố định thinking + effort, k = 3 + liệt kê ca chập chờn | kết luận đúng với model người ta dùng thật | bỏ chữ "temperature 0" của D5 |
| B | model đời cũ còn nhận temperature 0 (ví dụ Haiku 4.5 / Sonnet 4.5, tắt thinking) | giữ đúng chữ D5 | kết luận trên đời model cũ, sắp ngừng; ít giá trị cho sản phẩm |
| C | như A nhưng k = 5 | trung vị vững hơn | thêm khoảng 67% lời gọi |

## Đề xuất bổ sung từ bản soát độc lập (chờ chọn)

`docs/g7/REVIEW-D5.md` đề xuất thêm nhiều sửa đổi với D5 và hợp đồng. Chủ dự án chọn ở D-G7-9 xong thì gộp các mục được chọn vào ADR này **trước** khi duyệt. Các mục:
- bộ giữ lại ≥ 96 ca;
- B2+ làm baseline cổng; B2″ và B2-oracle để báo cáo;
- k = 5 và luật vô hiệu cho đường tất định;
- kiểm gộp McNemar;
- bỏ `permission` khỏi phép đếm 4/8;
- ghi "mọi lượt" vào luật an ninh.

## Hệ quả

- Trước lần chạy thật phải sửa client:
  - bỏ `temperature`;
  - thêm `thinking` / `effort` lấy từ cấu hình;
  - thêm thử lại có ghi lần thử;
  - thêm thử khói.
  Đây là blocker mã ở `docs/g7/CHECKPOINT-A-H.md` mục H, làm sau khi ADR này được duyệt.
- Thiếu `temperature 0`, kết quả giữa các lượt có thể khác nhau. Trung vị k = 3 và danh sách ca chập chờn là cách đo độ khác nhau đó, không giấu nó.
- Đổi bất kỳ điều gì trong hợp đồng sau khi duyệt đều cần ADR mới và chạy lại toàn bộ.
