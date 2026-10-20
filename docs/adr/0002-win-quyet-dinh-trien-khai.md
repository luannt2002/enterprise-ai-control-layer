# ADR-0002 — Các quyết định triển khai WIN tự chốt ngày 20/10/2026

- **Trạng thái:** đề xuất. Tự chốt trong phiên tự chạy khi chủ dự án vắng; **đảo được**, chờ duyệt.
- **Ngày:** 20/10/2026.
- **Liên quan:** `docs/04-core-mvp.md` §6 (7 cổng), ADR-0001 (phạm vi WIN, D1–D6).

## Bối cảnh

Chủ dự án giao chạy tự động để làm WIN trong ngày, không hỏi lại. Các câu dưới đây cần trả lời mới đi tiếp được.

Luật chọn (CLAUDE.md §4):
- có cách mặc định rõ và đảo lại được thì tự chốt rồi ghi ở đây;
- việc không đảo được (xoá dữ liệu, chi tiền) thì dừng — G7 với model thật thuộc loại này.

## Quyết định

| # | Câu hỏi | Quyết định | Lý do / số đo | Đảo khi |
|---|---|---|---|---|
| D7 | Câu nhắc máy mà không nói vật tư ("máy 03 xuất bao nhiêu?") | hiểu là **dầu**: `default_terms` của pack. Chỉ áp khi câu có thực thể và không có đại lượng đo được nào khác ("giá", "nhân công") | trợ lý nhiên liệu; hỏi lại mọi câu như vậy là phiền vô ích; ca 107 do agent độc lập soạn đòi đúng vậy | tenant có nhiều vật tư ngang nhau |
| D8 | Câu vừa hỏi số liệu vừa hỏi quy trình | có máy / dự án ⇒ **HYBRID** (số từ fact + quy trình từ tài liệu, cần cả `metrics.read` lẫn `documents.read`). Không có ⇒ **RAG**, trừ khi câu có mốc thời gian cụ thể ⇒ hỏi lại máy / dự án | ca 068, 095, 096, 116; câu "hôm nay cấp bao nhiêu dầu, quy trình khi lệch là gì?" mà đi RAG thì ra INSUFFICIENT_EVIDENCE — tệ hơn hỏi lại | có ca chứng minh cần trả lời nửa câu |
| D9 | Truy xuất cho WIN | chỉ mục trong bộ nhớ, đoạn = một dòng SOP, điểm = tổng IDF.<br>- So khớp theo **luật dấu** của hệ (`ToneKey`: "hoá" = "hóa", "bí" ≠ "bị"); đoạn mang từ của tiêu đề; bỏ cụm nghi vấn ("làm gì"…).<br>- PG lưu tài liệu nguyên văn + ACL; cắt đoạn và chấm điểm trong Go lúc truy vấn. | "bí" khớp "bị" từng kéo ra tài liệu bị chèn lệnh (ca 072); "làm" khớp "làm việc" từng lấy nhầm bước (ca 116).<br>Chỉ mục toàn văn PG chưa có ca nào đòi. | tài liệu đủ lớn để cắt đoạn lúc truy vấn chậm (đo được) |
| D10 | Hỏi **lượng** nghiệp vụ mà chỉ có tài liệu trả lời ("theo quy trình được cấp tối đa bao nhiêu lít?") | INSUFFICIENT_EVIDENCE, **không gọi model**. Cụm hỏi lượng khai ở pack (`quantity_markers`) | tài liệu không phải nguồn của số liệu (P4); lượng dầu trong SOP bị chặn ở S9 nên đằng nào cũng không giao được | tenant có định mức là fact có thẩm quyền |
| D11 | PostgreSQL cho WIN | - ảnh `postgres:17` ghim digest (PG 17.11 có sẵn trên máy);<br>- chỉ nghe `127.0.0.1`, xác thực `trust`, nên không có mật khẩu nào trong repo; dữ liệu trên tmpfs;<br>- mọi bảng nghiệp vụ RLS **FORCE** theo GUC `eais.tenant_id` đặt bằng `SET LOCAL` từ tenant của Principal; vai ứng dụng `eais_app` (NOSUPERUSER NOBYPASSRLS);<br>- `Load` thay trọn dữ liệu của một tenant trong một giao dịch. | - RLS đóng mặc định: không đặt GUC ⇒ 0 dòng;<br>- WITH CHECK chặn ghi chéo tenant;<br>- COPY không chạy trên bảng có RLS nên nạp bằng `INSERT … unnest`. | chạy ngoài máy dev — cần mật khẩu / TLS và quản lý bí mật |
| D12 | Ràng buộc claim (G5) | mỗi số trong câu số liệu có một claim đủ **chủ thể / chỉ số / kỳ / giá trị / đơn vị / nguồn** và phải khớp đủ 6 chiều với bằng chứng nó trỏ tới.<br>- Số do người hỏi đưa ra là bằng chứng loại `question`; số chỗ lệch còn lại là bằng chứng phép tính `count:conflicts`.<br>- Ngày trong câu phải là đầu hoặc cuối kỳ của một claim.<br>- Số được đếm bằng biểu thức chính quy riêng, không qua bộ chuẩn hoá. | đúng chữ của 04 §6. Lớp "có mặt" (`Fidelity`) giữ làm lớp lọc đầu: nó không bắt được ngày trôi trong mẫu câu, `BindClaims` bắt được | — |
| D13 | Câu so sánh hai nghĩa ("đội báo nhiều hơn kho xuất bao nhiêu?") | hai ý định số liệu hoà nhau + cụm so sánh của pack (`comparison_markers`) ⇒ **hiệu theo thứ tự nhắc** (vế trước − vế sau), hiển thị cả hai số. Không có cụm so sánh ⇒ vẫn hỏi lại | luật chung, không khai từng cặp chỉ số; ca 098 | có câu so sánh mà thứ tự nhắc không phải thứ tự trừ |
| D14 | Câu xác nhận nhiều mệnh đề | mệnh đề chứa cụm xác nhận là câu hỏi: ý định và con số được hỏi lấy từ đó. Mệnh đề đó không có từ khoá / con số thì dùng cả câu | ca 049 — "đội báo 4k, báo cáo ghi 4 m3, vậy máy 03 được cấp 4k dầu đúng không?" hỏi về CẤP | có ca phủ định luật này |
| D15 | Số âm | bộ chuẩn hoá đọc dấu `-` / `−` là dấu âm khi ngay sau là chữ số và ngay trước là khoảng trắng, đầu câu hoặc mở ngoặc. "EX-03", "R-004", "18-20" không đổi | thiếu nó thì mọi hiệu âm (được phép theo G6) bị S9 chặn khi giao | — |
| D16 | "≥ 4 biểu diễn" của G1 | thêm biểu diễn thứ tư cho R-004: CSV gõ tay, lượng `3.850`, đơn vị `lít` | bộ sinh cũ chỉ có 3 biểu diễn cho phiếu kho | — |
| D17 | Nhãn số trích từ SOP (G5) | mẫu `knowledge_answer` = "Trích văn bản quy trình ({title}): …" | 04 §6 đòi số trích SOP "được gắn nhãn trích văn bản quy trình" | tenant muốn câu chữ khác — sửa mẫu, không sửa mã |

## Hệ quả

- G1–G6 có bằng chứng đạt; scale smoke đạt trên cả hai đường dữ liệu. G7 là cổng duy nhất còn lại và bị chặn ở khoá model + chi phí.
- Pack có thêm 4 khoá: `quantity_markers`, `comparison_markers`, ý định `sop_question`, mẫu `knowledge_answer` / `no_knowledge` / `hybrid_no_knowledge` / `compare_total`. Bộ nạp pack đòi đủ mẫu câu khi khai các khoá đó.
- Hợp đồng `contracts` mở rộng, chỉ thêm trường, không đổi nghĩa trường cũ:
  - `Claim` đủ 6 chiều;
  - `Evidence.Subject / Metric / Period`;
  - `SourceValue.EvidenceIDs`;
  - `EvidenceQuestion`;
  - `WithTenant` / `TenantFrom`.

## Phương án đã cân nhắc

- **D13, khai từng cặp so sánh trong pack** (ý định riêng cho "đội báo nhiều hơn kho xuất"): bị loại vì từ khoá "nhiều hơn" đứng một mình bắt nhầm cả câu so sánh hai ngày.
- **D9, chỉ mục toàn văn PG (`tsvector`)**: để sau. Cấu hình `simple` + `unaccent` của PG không giữ luật dấu của hệ ("bí" ≠ "bị"), nên hai đường dữ liệu sẽ ra khác nhau.
- **D11, mật khẩu trong `.env`**: bị loại cho máy dev. `trust` + loopback + tmpfs không để lại bí mật nào trong repo.
