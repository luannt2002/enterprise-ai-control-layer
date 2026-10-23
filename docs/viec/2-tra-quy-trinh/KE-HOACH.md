# Kế hoạch build RAG chuẩn prod trong EAIS

> **Trạng thái:** ĐỀ XUẤT, làm trên nhánh `rag/*` ở phiên điều phối. Team BUILD không sửa vùng này (lệnh ca MVP ALL-IN).
> **Đích:** việc "tra căn cứ đang hiệu lực khi xử lý chênh lệch nhiên liệu" (quy trình + định mức) chạy đúng tới RAG-5 của `docs/ROADMAP-RAG.md`.
> **Cách làm mỗi đơn vị:** ca đỏ trước → mã nhỏ nhất → test + đột biến chứng minh test cắn → agent soát chỉ-đọc → sửa → hồi quy theo tên + 117 / H1 / H3 không đổi dòng nội dung → chờ chủ dự án cho commit.

## 0. Năm nguồn đã học (chi tiết ở `docs/ROADMAP-RAG.md` §1, §7, §8)

| Nguồn | Lấy | Không lấy |
|---|---|---|
| A. Khoá học production RAG | 5 kiểu hỏng; hybrid + RRF; kiểm ngân sách trước khi gọi model; kho nhỏ thì nạp thẳng | "cấm cắt cố định", "semantic tốt hơn 90%" (tự mâu thuẫn) |
| B. Hiện thực AdapChunk | khối nguyên (bảng, công thức), đường dẫn mục, ghi lý do quyết định cắt | LLM chọn cách cắt (chưa từng đo đầu–cuối) |
| C. Bài báo Adaptive Chunking | bất biến "đoạn liền mạch, phủ hết văn bản"; bảng dài lặp header; metric BI / SC làm cổng | chọn cách cắt theo trung bình metric (chọn và chấm trên cùng metric) |
| D. Hệ RAG nội bộ chưa dùng | bản đồ đủ các tầng (cache, guard, outbox) để biết mình bỏ gì | xây rộng trước khi đo đúng sai; lọc quyền không tác dụng; guard chỉ ghi log |
| E. Dịch vụ trả lời viết lại (đã đo trên hệ dev) | chọn nạp thẳng hay RAG theo cỡ kho (kho ~20k token: nạp thẳng rẻ ~3 lần; ~150k: RAG rẻ ~2,5 lần); nạp thẳng gửi văn bản gốc, không gửi đoạn (đoạn lặp header phồng 13–15%); chế độ chỉ tìm kiếm để đo truy hồi (rẻ ~300 lần); đoạn biết tài liệu + bản; phá hoà tất định; escape thẻ khung; cụt ở `max_tokens` là lỗi; nhãn lỗi theo bên gây lỗi; một dòng log mỗi lượt kèm `unaccounted_ms` | tin tenant trong payload; không kiểm câu trả lời của model |

## 1. Đã xong (chưa commit)

| Đơn vị | Nội dung | Bằng chứng |
|---|---|---|
| U1 | `Document` có phiên bản / trạng thái / hiệu lực / phạm vi; lọc trước khi tìm; `VersionNotice` (xung đột / không có bản hiệu lực); ngoài vai hoặc phạm vi = không tồn tại; nạp phần đầu YAML có kiểm | test đỏ trước; 12/12 đột biến bị bắt; hồi quy theo tên 0 mất / 0 đổi; 117 / H1 / H3 0 dòng nội dung khác |
| U2 | `evals/rag`: chạy THẬT 5 ca của `cases.yaml` trên dữ liệu mẫu | 032, 061, 071 xanh; 004, 005 đỏ biết trước; chốt chặn đỏ khi cài lỗi |
| Soát vòng 1 | 16 điểm (1 S1, 5 S2, 10 S3) | đã sửa F1–F12, F14, F16; F13, F15 thuộc U7 |
| U3 | Bảng: chỉ là bảng khi dòng sau là dòng phân cách; mỗi hàng một đoạn kèm header, chỉ chữ của hàng được chấm điểm; bảng viết liền mang header riêng; dòng phân cách không thành đoạn | test đỏ trước; soát vòng 1: 6 điểm (0 S1, 1 S2, 5 S3), đã sửa T1–T5, T6 ghi nợ cho U7; 7/7 đột biến bị bắt; CON-T-005 xanh; CON-T-004 canh thêm "Xe lu" |
| U5a | Cổng chất lượng đọc (BR-DOC-03) lúc nạp: bản khai "đã duyệt" mà bảng có hàng sai số ô, hoặc mất cột so với bản đã duyệt gần nhất → tự "chờ soát" kèm lý do (`Hold`); một luật nhận bảng dùng chung cho chỉ mục và cổng (`parseBody`) | test đỏ trước; soát vòng 1 (11 điểm, 1 S1) + vòng 2 (N1 S1 đếm cột theo hàng, N2, G6, N4…) đã sửa; 17 đột biến biên dịch được đều bị bắt; CON-T-032 dùng cổng tự động |
| U7a | Lát mỏng tới người dùng: pipeline đổi thông báo phiên bản thành câu trả lời (xung đột → `SOURCE_CONFLICT`, không có bản hiệu lực → `INSUFFICIENT_EVIDENCE`, 0 lời gọi model, không hứa "đã chuyển"); HYBRID giữ phần số khi truy xuất / model lỗi; kiểu thông báo nằm ở `core/contracts` | test đỏ trước (đúng F15: thông báo làm hỏng cả lượt, HYBRID mất số); 5/5 đột biến bị bắt; bàn giao: `docs/viec/2-tra-quy-trinh/BAN-GIAO-U7A.md` |
| U8a | Phép so ở mức bằng chứng (bản rẻ của luận đề gốc): cùng 5 ca, ba cách chọn thứ đưa cho model — EAIS / "RAG làm trung tâm" (có tenant + ACL vai, không biết phiên bản) / nạp thẳng (`evals/rag/so_sanh_test.go`) | EAIS 5/5 ca, 0 tài liệu không làm căn cứ được lọt; RAG 1/5, 6 lượt lọt; nạp thẳng 1/5, 6 lượt lọt. Giới hạn: bộ ca xoay quanh phiên bản (nghiêng về EAIS); đếm theo bản, không theo hàng (ca 004 vẫn lọt hàng máy ủi của đúng v3). Chốt chặn đỏ khi EAIS bỏ qua ngày hết hiệu lực **Mở rộng 23/10 trên bộ S7 phát triển** (`TestSoSanhBaHeTrenS7`): EAIS top-3 30/36, 0 ngoài quyền, 0 không căn cứ; RAG thường 25/36, 4 + 19; nạp thẳng 25 + 64 |

## 2. Đơn vị tiếp theo

| # | Nấc | Mục tiêu | Ca đỏ trước | Cổng ra |
|---|---|---|---|---|
| U3b | RAG-2 | Phần còn lại của cắt theo khối: đường dẫn mục (Chương / Điều / bước), gom nhóm hàng cho bảng dài theo ngân sách token | ca có tài liệu nhiều mục | bất biến phủ hết; 117 / H1 / H3 0 dòng khác |
| U4a | RAG-1 | Trích bảng của bản đã duyệt + đang hiệu lực thành dòng có cấu trúc (loại máy, định mức, đơn vị lấy từ header, dung sai, nguồn = mã@bản#hàng) — CHỈ để truy hồi đúng hàng và trích dẫn | ca: hỏi định mức theo loại máy, không lọt hàng khác | nằm trong `intelligence/rag/`, không đổi contract |
| U4b | — | Dùng dòng định mức đó làm đầu vào CALC (BR-FUEL-05 ở `01-business-map.md:384`: "định mức lấy từ SOP đang hiệu lực, nhân giờ máy"), cần EX-01 → loại máy | CON-T-004 | **phối hợp team MVP**: chạm thực thể (seed chưa có EX-01, chưa có thuộc tính loại máy), pack, CALC; cần chủ dự án chốt nguồn có thẩm quyền và người duyệt |
| U5 | RAG-1 | Nạp đủ định dạng: PDF chữ (`pdftotext`, đã có), XLSX (`excelize`, đã có trong go.mod), DOCX (đo trên tài liệu thật trước khi chọn bộ đọc); đo độ phủ chữ; cách ly có lý do (cổng chất lượng bảng: đã làm ở U5a) | CON-T-032 (bảng mất cột → không công bố), CON-T-081 (bộ đọc sập → bản cũ vẫn phục vụ) | mỗi định dạng có tệp thật; 0 mất chữ âm thầm |
| U6 | RAG-3 | Truy hồi trên Postgres: bảng tài liệu có phiên bản + RLS; `tsvector` + truy vấn OR + dạng bỏ dấu; lọc trong SQL, không lọc sau; pgvector chỉ khi có ca đồng nghĩa trượt. Tách từ tiếng Việt: không mặc định, đo trên câu văn trước khi thêm | ca của U2 chạy cả hai đường (bộ nhớ, PG), kết quả bằng nhau | **U6a xong 23/10**: bảng có phiên bản (migration 0005), 5 ca bộ nhớ = PG (`evals/rag/pg_test.go`), ngày hỏi + luật phạm vi đi theo lượt. **U6b chưa làm** (`tsvector`, chấm điểm trong SQL): chờ số đo kho đủ lớn để thấy chấm trong Go chậm (IDF chỉ trên tài liệu người hỏi thấy: xong 23/10 cùng U8b). **U6c chưa làm**: lưu trạng thái công bố để nạp lại không hạ bản đang công bố (HOP-DONG §3, giới hạn của cổng chất lượng) |
| U7 | RAG-4 | Nối pipeline: `VersionNotice` → mẫu câu + outcome; HYBRID giữ phần số khi truy hồi lỗi; "không có" = "ngoài quyền"; vết RAG (ROADMAP-RAG §6.7); `Now` của lượt hỏi (F5); `InScope = policy.InScope` (F13) | ca HYBRID khi truy hồi lỗi; ca F15 | **chạm `core/pipeline/`**: báo team trước qua `docs/debug-handoff/` |
| U8 | RAG-5 | Bộ đo nội dung: bộ ca người soạn từ tài liệu gốc; kiểm tất định trích dẫn + số; chế độ chỉ tìm kiếm; 2 baseline (nạp thẳng, trích đoạn); chọn nạp thẳng hay RAG theo cỡ kho; judge khác họ chỉ là tín hiệu | — | số của 2 baseline; bộ giữ lại khoá băm, chạy một lần **U8b — dòng S7 (23/10):** bộ đo `evals/rag/s7_test.go` + bộ phát triển 41 câu / kho 15 tài liệu: 30/36 = 83,3% cả hai đường, 0 ngoài quyền / không căn cứ / thiếu nguồn ⇒ KHÔNG ĐẠT (số phát triển). **Bộ độc lập (106 câu, chạy một lần 23/10): top-3 87/92 = 94,6%, 0 / 0 / 0 ⇒ dòng S7 ĐẠT**; RAG thường 91/92 nhưng 20 ngoài quyền + 37 không căn cứ (`S7-BO-CA-DOC-LAP.md`). **P1 biết im** sửa 23/10 (luật 2' trên chủ đề): câu không có đáp án nhận đoạn 5/5 → 2/5, top-3 giữ 30/36 |
| U9 | RAG-7 | Vận hành: nhãn lỗi, một dòng log mỗi lượt + `unaccounted_ms`, `max_tokens` là lỗi, escape khung, timeout, hợp đồng embedding + nhúng lại khi đổi model | — | ca âm an ninh đỏ đúng |

Chưa làm: vector, rerank, làm giàu đoạn, viết lại câu nối tiếp, agent, graph (RAG-6, RAG-8). Chỉ mở khi có lỗi đo được trên bộ ca.

## 3. Go hay dịch vụ ngoài

Mọi logic ở trên viết bằng Go. Bộ đọc tài liệu khó (bản scan, bảng phức tạp) chọn bằng **đo trên cùng tài liệu gốc**: độ đúng của bảng, thời gian, RAM. Công cụ có thể là Python, bản native hay API. Embedding / rerank: Go gọi API; chạy model ở máy mình thì dùng dịch vụ suy luận riêng, Go gọi HTTP.

## 4. Chờ chủ dự án

1. Phương án (b) cho định mức (U4).
2. Cho commit U1 + U2 lên nhánh; ghi D-RAG-01 (RAG làm song song ở phiên điều phối) khi gộp vào `main`.

## 5. Bài học của các dự án cũ → chốt ở đâu

| # | Lỗi đã thấy (ROADMAP-RAG §7) | Chốt trong EAIS |
|---|---|---|
| 1 | Khối code không đóng nuốt nửa tài liệu (B) | bộ cắt EAIS không có nhánh khối code; bất biến phủ hết `TestCatBangPhuHetVanBan` |
| 2 | Bộ làm sạch xoá header bảng (D) | không có bước xoá dòng lặp; header đi kèm từng hàng `TestCatBangLapHeader`, ca 005 |
| 3 | ID ngẫu nhiên, nạp lại nhân đôi (B) | ID tất định `mã@bản#dòng`; trùng `mã@bản` là lỗi nạp; upsert ở U6 |
| 4 | Excel mất hàng header (C) | U5, kiểm bằng tệp thật |
| 5 | Semantic / proposition mất heading (B) | không dùng; bất biến phủ hết |
| 6, 7 | Đổi model không nhúng lại; lệch số chiều xoá cả chỉ mục (D, B) | U6 / U9: hợp đồng embedding lưu cạnh vector, lỗi to thay vì tự xoá |
| 8 | Lọc quyền có trong mã nhưng không tác dụng (D) | `TestRetrievePhamViKhongLo`, `TestRetrievePhamViTheoTungBan`, ca 071 |
| 9 | Từ khoá không hiểu tiếng Việt (B, D) | luật dấu + bỏ dấu đã có (`TestRetrieveLuatDauVaCumNghiVan`); U6 cho Postgres |
| 10 | Rerank lỗi còn 1 đoạn (D) | chưa mở rerank (RAG-6); khi mở: ngưỡng đúng thang + bộ đếm dự phòng |
| 11 | Ngữ cảnh không delimiter, chèn lệnh được (B) | U7: escape thẻ khung (lấy từ E); ca `SOP-INJ` |
| 12 | Stream phát trước khi guard chạy (D) | U7 / U9 khi có stream |
| 13 | Câu chào đầu câu bị coi là chitchat (D) | U7: thêm ca "Dạ, cho hỏi …" vào bộ chạy |
| 14 | Eval chấm trên ngữ cảnh vàng; golden lệch kho (D) | `evals/rag` chạy truy hồi thật; câu hỏi phải trùng nguyên văn `cases.yaml` |
| 15 | Judge cùng họ, câu hỏi sinh từ chính hệ (B, C, D) | U8 |
| + | Tệp CRLF / BOM làm bản nháp được phục vụ (soát vòng 1, F1) | `TestLoadMarkdownCRLFvaBOM` |
| + | Tài liệu đã thay thế che tài liệu mới (soát vòng 1, F16) | `TestRetrieveBanDaThayTheKhongCheTaiLieuMoi` |

## 6. Tận dụng từ nguồn E (dịch vụ trả lời viết lại)

Viết lại theo khuôn EAIS (tenant từ danh tính, `contracts`, `traces.Measure`), không chép nguyên khối.

| Thành phần của E | Dùng ở | Cách |
|---|---|---|
| Hybrid trong một câu SQL: dense + sparse + RRF, phá hoà theo tài liệu → vị trí → id, `to_tsquery` nối OR từ `quote_literal` | U6 `platform/pgstore` | lấy cấu trúc câu SQL; khoá đổi từ bot sang tenant + ACL + phạm vi + bản hiệu lực; giữ phá hoà |
| Đoạn biết tài liệu + bản; vân tay cache gồm bản + băm nội dung; xoá tài liệu kéo theo đoạn (migration 0017) | U6 schema | lấy thiết kế bảng |
| Chọn nạp thẳng hay RAG theo cỡ kho; nạp thẳng gửi văn bản gốc | U8 (baseline) → U7 (chọn đường) | lấy logic; ngưỡng đặt trong pack |
| Chế độ chỉ tìm kiếm | đã có dạng trong tiến trình (`evals/rag`) | qua HTTP thì đề xuất cho làn 0 |
| Escape thẻ khung trong đoạn và lịch sử | U7 | EAIS chưa có (grep `intelligence/models` = 0) |
| Cụt `max_tokens` / không chữ là lỗi | — | EAIS đã có (`intelligence/models/messages.go:182-187`) |
| Nhãn lỗi theo bên gây lỗi | U9 | khớp dạng nhãn ca đã dùng: `infra:document-parser` (CON-T-081), `third_party:model:timeout` (CON-T-082) |
| Một dòng log mỗi lượt + `unaccounted_ms`; cổng AST "mọi bước đo phải khai mức bằng chứng" | U9 | đề xuất cho làn 0; bản cho các bước RAG đặt ở `evals/rag` |
| Câu nối tiếp = tin trước + câu hiện tại, không gọi LLM | RAG-6 (hội thoại) | dùng khi EAIS có hội thoại |
| Bộ đọc tài liệu: docling cho DOCX / PDF có bảng; XLSX đọc thẳng; bỏ tách từ | U5 | đo trên cùng tài liệu; XLSX dùng `excelize` |
| Phát lại theo `message_id`, room, dọn lịch sử | — | không thuộc RAG |
