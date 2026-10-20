# ADR-0002 — Các quyết định triển khai WIN tự chốt ngày 20/10/2026

- **Trạng thái:** **TẠM CHẤP NHẬN** (chủ dự án, 20/10/2026), dưới dạng quyết định triển khai đảo được để tiếp tục WIN. Điều kiện kèm theo:
  - ADR này là sổ ghi quyết định; heuristic của tenant / domain **không** được nâng thành lõi EACL;
  - D9 là lựa chọn truy xuất cho WIN, **không** phải kiến trúc truy xuất chung của EACL;
  - D11 là quyết định triển khai cho máy dev / test, **không** phải khuyến nghị an ninh cho production;
  - mọi dòng giữ cột "Đảo khi" và bằng chứng hiện có.
  Phân lớp từng quyết định: mục "Phân lớp" bên dưới.
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

## Phân lớp (20/10/2026, theo yêu cầu chủ dự án)

Năm tầng:
- **CORE INVARIANT**: bất biến của lõi, đúng với mọi tenant;
- **DOMAIN PACK**: luật nghiệp vụ của một ngành;
- **TENANT CONFIG**: giá trị cấu hình của một tenant;
- **WIN-ONLY IMPLEMENTATION**: cách làm đủ cho WIN, chưa phải kiến trúc chung;
- **DEV-ONLY**: chỉ cho máy dev / test.

"Bật theo pack" nghĩa là cơ chế nằm trong mã dùng chung nhưng **không chạy** nếu pack không khai khoá tương ứng. Tenant khác không bị áp.

| # | Tầng | Ở đâu | Đúng chỗ? |
|---|---|---|---|
| D7 vật tư mặc định | cơ chế: WIN-ONLY (bật theo pack) · giá trị `diesel_fuel`: TENANT CONFIG | `intelligence/intent` · `pack/intents.yaml` `default_terms` | đúng |
| D8 ghép HYBRID / RAG, câu có mốc thời gian thì hỏi lại | WIN-ONLY (luật định tuyến mặc định; bật khi pack có ý định RAG) | `intelligence/intent` | đúng. Không phải bất biến EACL; mở cấu hình khi domain thứ hai cần khác |
| D9 truy xuất (luật dấu, từ tiêu đề, bỏ cụm nghi vấn) | WIN-ONLY. `ToneKey`: lõi chuẩn hoá tiếng Việt | `intelligence/rag`, `data/normalization` | **nợ đặt chỗ:** danh sách từ dừng và cụm nghi vấn tiếng Việt viết cứng trong `intelligence/rag`, áp mọi tenant (không bật theo pack). Dời sang lexicon / pack khi có tenant ngôn ngữ khác. Chưa dời: chưa có ca đòi, và G7 đóng băng hành vi truy xuất |
| D10 hỏi lượng thuần tài liệu thì thiếu bằng chứng | nguyên lý: CORE INVARIANT (P4, tài liệu không có thẩm quyền về số liệu) · cụm hỏi lượng: DOMAIN PACK | `core/pipeline` · `quantity_markers` | đúng |
| D11a RLS FORCE theo GUC lấy từ Principal, vai `eais_app` không BYPASSRLS | CORE INVARIANT (cách ly tenant) | `db/migrations`, `platform/pgstore` | đúng |
| D11b `trust` + loopback + tmpfs | DEV-ONLY | `deploy/docker-compose.yml` | đúng. Không dùng cho production |
| D12 ràng buộc claim 6 chiều, bộ đếm số độc lập | CORE INVARIANT (C2 sự thật ≠ model, C5 chứng minh được) | `reliability/validation`, `core/pipeline` | đúng |
| D13 so hai nghĩa | heuristic: WIN-ONLY (bật theo pack `comparison_markers`) · `Catalog.Compare`: cơ chế lõi chung · mẫu `compare_total`: TENANT CONFIG | `intelligence/intent`, `data/sourceoftruth`, pack | đúng |
| D14 câu xác nhận nhiều mệnh đề | WIN-ONLY (bật theo pack `confirm_markers`) | `intelligence/intent` | đúng |
| D15 số âm | lõi chuẩn hoá (đúng / sai của việc đọc số) | `data/normalization` | đúng |
| D16 biểu diễn thứ tư của R-004 | WIN-ONLY (dữ liệu thử của G1) | `tenants/construction/win/gen` | đúng |
| D17 nhãn "Trích văn bản quy trình" | TENANT CONFIG (câu chữ) · đòi hỏi gắn nhãn: G5 | `pack/templates.yaml` | đúng |

Kết quả soát: **không quyết định nào nằm sai tầng tới mức phải dời mã ngay.** Một nợ đặt chỗ (D9) được ghi lại kèm điều kiện kích hoạt.

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
