# Bàn giao từ phiên RAG — U7a chạm tệp dùng chung (nhánh `rag/0-viec`, chưa gộp `main`)

> **Viết cho:** team BUILD. **Ngày:** 22/10/2026, cập nhật 23/10 (U6a). Theo lệnh ca MVP ALL-IN, team không đổi hành vi route RAG / HYBRID;
> phiên RAG báo trước những chỗ nó sửa ngoài vùng của mình để khi gộp không va.

| Tệp | Thay đổi | Ảnh hưởng tới team |
|---|---|---|
| `core/contracts/contracts.go` | **thêm** `KnowledgeNotice`, `KnowledgeNoticeKind`, `DocumentVersionInfo` (không đổi kiểu sẵn có) | không |
| `core/pipeline/ask.go` | chỉ trong `answerKnowledge` + hàm mới `versionNotice`: thông báo phiên bản → mẫu câu + outcome (`SOURCE_CONFLICT` / `INSUFFICIENT_EVIDENCE`), 0 lời gọi model; HYBRID giữ phần số khi truy xuất / model lỗi | route số liệu không đổi; 117 / H1 / H3 không đổi dòng nội dung (xem báo cáo hồi quy của nhánh) |
| `core/tenant/pack.go` | `knowledgeTemplates` thêm 5 mẫu bắt buộc khi pack có route RAG | pack có route RAG phải có 5 mẫu dưới |
| `core/tenant/pack_test.go` | fixture "pack tri thức hợp lệ" thêm 5 mẫu (hợp đồng đổi có chủ đích); thêm ca âm: thiếu `knowledge_version_conflict` thì nạp pack lỗi | hồi quy có PG trước khi sửa fixture: đúng 1 test đỏ (`TestLoadPackTriThuc`), do chính thay đổi này |
| `tenants/construction/pack/templates.yaml` | thêm `knowledge_version_conflict`, `knowledge_no_effective`, `hybrid_knowledge_version_conflict`, `hybrid_knowledge_no_effective`, `hybrid_knowledge_error`; dùng biến có sẵn `{title}`, `{reason}` | không đổi mẫu cũ |
| `tenants/construction/pipeline_rag_test.go` | tệp test mới | không |
| `db/migrations/0005_tai_lieu_phien_ban.sql` | **mới** (U6a, 23/10): bảng `documents` thêm `version, status, effective_from, effective_to, scope, hold`; khoá chính `(tenant_id, id, version)`; CHECK dạng ngày | **số 0005**: team đã dùng 0004 cho phiếu (M4/K2, `0004_phieu_chenh_lech.sql`); RAG lấy 0005, migration kế tiếp lấy 0006. Dòng cũ đọc ra y như trước (mặc định = không phiên bản) |
| `platform/pgstore/load.go`, `pgstore.go` | ghi / đọc đủ các cột trên; đọc theo `id, version` | nạp dữ liệu WIN không đổi (winreport 117 / H1 / H3 / H4 hai đường: 0 dòng khác `f344f50`) |
| `platform/pgstore/migrate_test.go`, `vet_test.go` | danh sách migration mong đợi thêm `0005_tai_lieu_phien_ban.sql` (hợp đồng đổi có chủ đích) | trước khi sửa: đúng 2 test đỏ vì danh sách ghim cứng (`TestMigrateDongThoiTrenDBTrong`, `TestVetMigration0003TrenDBTrong`) |
| `core/contracts/tenantctx.go` | **thêm** `KnowledgeQuery` + `WithKnowledgeQuery` / `KnowledgeQueryFrom`: ngày hỏi + luật phạm vi của lượt | không |
| `core/pipeline/ask.go` | truy xuất tri thức truyền `KnowledgeQuery{AsOf: now, InScope: policy.InScope}` qua ngữ cảnh; ngày / luật của lượt thắng trường của chỉ mục | route số liệu không đổi |
| `evals/rag/pg_test.go` | test mới: 5 ca tri thức qua PG khớp bộ nhớ; phạm vi trong cùng tenant qua PG | cần PG như mọi test pgstore |
| `platform/pgstore/ready.go` | `Ready` kiểm thêm cột `documents.version` (migration 0005): binary mới trên kho chưa migrate thì từ chối khởi động, không phục vụ rồi lỗi ở lượt RAG / lần nạp đầu | server trên kho cũ báo "chạy `eais migrate`" |
| `platform/pgstore/load.go` | `acl` rỗng (tài liệu không có trong `acl.json`) ghi `'{}'` thay vì NULL: trước đây một tài liệu như vậy làm hỏng CẢ lần nạp PG (đường bộ nhớ thì nạp được, không ai đọc) | lỗi có từ trước U6a, soát độc lập U6a phát hiện |
| `platform/pgstore/tai_lieu_test.go` | test mới: CHECK ngày / bản âm, ACL rỗng, nâng cấp kho có dữ liệu trước 0005 → 0005, `Ready`, chạy lại trần 0005 | cần PG |

| `intelligence/rag/rag.go` | IDF chỉ trên đoạn người hỏi thấy (trước: mọi đoạn của tenant) — điểm không phụ thuộc tài liệu người hỏi không được đọc, đường bộ nhớ = đường PG | winreport 117 / H1 / H3 / H4 hai đường: 0 dòng khác `f344f50` |
| `evals/rag/s7_test.go`, `tenants/construction/knowledge/s7/` | bộ đo dòng S7 + bộ phát triển (`S7-BO-CA-DOC-LAP.md`); đọc biến `EAIS_S7_BO` cho lần chạy bộ độc lập | dòng S7 của `CONG-TRUOC-AI.md` (tệp của team): số phát triển 83,3% KHÔNG ĐẠT, chờ bộ độc lập |

| `core/pipeline/ask.go` (ráp lên `baaf834`, 23/10) | `answerKnowledge` giữ tham số `why` của R2-03 + `KnowledgeQuery` / thông báo phiên bản của RAG; thêm `noteNotice`: khi trả THÔNG BÁO phiên bản, và khi HYBRID bỏ phần quy trình vì lỗi truy xuất, vết S7 ghi lý do (trước đó hai đường này thoát trước `noteRetrieved` nên S7 hiện "chưa ghi") | test `TestRAGVetViSaoKhiThongBaoVaLoi` |

| `core/contracts/tenantctx.go`, `core/pipeline/ask.go`, `core/pipeline/chu_de.go` (mới) | `KnowledgeQuery.Topic` + `pipeline.KnowledgeTopic`: chủ đề = câu bỏ span số / lượng / tiền / ngày / kỳ / giờ / mã của S2; `ask.go` chỉ thêm `Topic: KnowledgeTopic(in)` vào lời gọi truy xuất. Chỉ mục im khi ≥ 1/2 từ của chủ đề ngoài kho người hỏi thấy (P1 biết im) | route số liệu không đổi; winreport 0 dòng khác |
| `tenants/construction/pipeline_hoi_quy_r_test.go` (test của team) | `TestLuongTaiLieuTrangTriVaNhanDonViKhongLot`: dữ liệu thử `| Máy xúc | 4.000 | lít/ca |` → `| Định mức cấp phát máy xúc | 4.000 | lít/ca |`; kiểm giữ nguyên | hàng trần không nêu "định mức cấp phát" nay dừng ở truy xuất (INSUFFICIENT_EVIDENCE, không lộ số) nên không tới lớp kiểm mà test canh; lý do ghi trong test |

| `platform/pgstore/pgstore.go` (`Retrieve`) | chỉ mục tri thức giữ sẵn theo tenant, dựng lại khi dấu nạp đổi (xmin của dòng `documents` + số dòng — lần nạp từ tiến trình khác cũng đổi dấu); quyền đọc từng tài liệu (ACL, phạm vi) do chỉ mục xét cho từng lượt, SQL không còn lọc `acl && vai` (RLS theo tenant vẫn ở SQL) | 437 ms → 22 ms mỗi câu ở ~1.000 tài liệu; test `TestChiMucGiuSanKhongCu` (nạp lại cùng / khác tiến trình không trả dữ liệu cũ) |

| `cmd/eais/rag_e2e_test.go` (test mới trong gói của team) | luồng RAG trọn qua ứng dụng: tài liệu có phiên bản trong thư mục dữ liệu → nạp → PG → `/v1/ask` theo token (director, kỹ sư PRJ-A / PRJ-B) → `/v1/trace`; nạp lại qua `POST /v1/ingest` | cần PG như TestServer; 4 đột biến (vùng đệm không dựng lại, bỏ chủ đề, PG bỏ trạng thái, vết không ghi thông báo) đều bị bắt |

**Ráp lên `baaf834` (23/10 11:1x):** 4 tệp giao nhau (`contracts.go`, `ask.go` ráp tự động; `migrate_test.go`, `vet_test.go` xung đột đúng một dòng danh sách migration). Migration RAG đổi `0004` → `0005` vì team đã dùng 0004 cho phiếu. Cổng của bản ráp: 1.109 test pass (1.054 của team + 55 của RAG), so theo tên với `baaf834` 0 mất / 0 đổi; winreport 117 / H1 / H3 / H4 hai đường 0 dòng khác; smoke 21/21.

**Ráp lên main `086fff9` (23/10 chiều):** main đã lấy số 0005 (`0005_audit_va_tenant.sql`) ⇒ migration RAG đổi `0005` → `0006_tai_lieu_phien_ban.sql` (đúng luật "ai gộp main trước giữ số"). Tệp chạy lại được nên DB đã áp bản mang số cũ chỉ chạy lại nó, không hỏng. Đổi theo: danh sách migration của `migrate_test.go` / `vet_test.go`, cột kiểm của `ready.go`, `tai_lieu_test.go`. Migration RAG kế tiếp lấy số sau số lớn nhất của main lúc gộp.

**Vận hành:** chỉ nạp tài liệu có phiên bản khi mọi binary trước U6a đã dừng — binary cũ đọc bảng mới nhưng bỏ qua cột trạng thái / hiệu lực / phạm vi, sẽ phục vụ bản nháp như bản chính. Không có migration ngược.

**Việc còn cần team (không tự sửa):**
1. ~~`platform/pgstore` từ chối tài liệu có phiên bản~~ — không cần nữa: U6a (23/10) cho PG lưu và phục vụ đủ phiên bản.
2. ~~`cmd/eais` đặt `MemoryIndex.Now` / `InScope`~~ — không cần nữa: pipeline truyền ngày hỏi và luật phạm vi của lượt cho
   mọi chỉ mục (bộ nhớ, PG), kể cả khi phát lại.
3. `tenants/construction/pack/policy.yaml`: các ca RAG dùng vai `site_supervisor`, `equipment_manager` chưa có trong policy.

## Đợt động cơ thật (23/10 tối) — tệp chung bị chạm

> Viết cho: team build. Chỉ để biết — không cần gộp hay so lúc này. Mặc định mọi thứ dưới đây TẮT: `cmd/eais` không đổi, vẫn model giả.

| Tệp | Đổi gì | Ảnh hưởng tới phần của team |
|---|---|---|
| `core/contracts/contracts.go` | `Draft.Refused` (`omitempty`): model trả đúng khuôn và nói bằng chứng không trả lời được | không: gateway có sẵn không đặt cờ này; JSON cũ không đổi |
| `core/pipeline/ask.go` (`answerKnowledge`) | `Draft.Refused` ⇒ INSUFFICIENT_EVIDENCE thay vì S9 chặn; bản nháp rỗng KHÔNG cờ vẫn qua S9 như cũ | không với model giả (không bao giờ đặt cờ) |
| `intelligence/models` | mới: `caller.go`, `chat.go` (`EAIS_MODEL_PROVIDER=chat_completions`), `embed.go`; `messages.go` chỉ thêm một nhánh trong `New` | `Messages` (G7) không đổi một dòng hành vi |
| `platform/pgstore` | `Store.Semantic` (nil = như cũ), `vector.go`, `knowledgeIndex` tách `readDocuments` | đường từ khoá giữ nguyên; danh sách migration trong `migrate_test.go` / `vet_test.go` thêm 0007 |
| `db/migrations/0007_vector_tai_lieu.sql` | bảng `document_vectors` + RLS | số 0007: lấy số kế tiếp sau 0006 của nhánh; ai gộp main trước giữ số |

Cổng của cây này (so với main sạch `086fff9` qua bản gộp `521867f`): `go test -json ./...` có PG 2034 pass + 1 skip (lệnh đo động cơ
thật, cần bật tay), so theo tên 0 mất / 0 đổi; winreport 117 / H1 / H3 / H4 × bộ nhớ / PG 0 dòng ca khác main; smoke 21/21.

**Phát hiện cho phần S4 của team:** đo luồng trọn với động cơ thật, route tri thức chỉ nhận 7/92 câu S7 độc lập (OUT_OF_SCOPE 39,
CLARIFICATION_REQUIRED 26, PERMISSION_DENIED 19) dù truy hồi trúng 83/92 — chi tiết `docs/ROADMAP-RAG.md` §10.2. Chưa sửa: đổi ranh
giới S4 cần chủ dự án quyết.

## U3b cắt đoạn (23/10 khuya) — chỉ trong `intelligence/rag/`

> Viết cho: team build. Chỉ để biết — không cần gộp hay so lúc này.

| Tệp | Đổi gì | Ảnh hưởng tới phần của team |
|---|---|---|
| `intelligence/rag/rag.go` | dòng tiêu đề markdown trong THÂN tài liệu thành đường dẫn mục của các đoạn bên dưới (bản thân không còn là đoạn, trừ tiêu đề không có nội dung hoặc chỉ có bảng); câu dẫn kết thúc bằng ":" + các mục con ngay sau (kể cả mục lồng) thành một đoạn | không với kho hiện có: 48/48 tài liệu ở `tenants/construction/knowledge` không có tiêu đề trong thân lẫn câu dẫn nên đoạn y như cũ. Tài liệu MỚI có tiêu đề / danh sách thì điểm rerank đổi → chọn lại `EAIS_RAG_MIN_SCORE` (ROADMAP-RAG §6.3) |
| `intelligence/rag/source.go`, `platform/pgstore/knowledge_sql.go` | trường cấu hình nguồn `tieu_de` (mẫu → cấp; nguồn postgres: khai là lỗi, chưa hỗ trợ) | không: trống = như cũ |

Cổng của cây này (so với `17213ce`): `go test -json ./...` có PG 2045 pass + 2 skip (hai lệnh đo động cơ thật, bật tay), so theo tên
0 mất / 0 đổi / 5 test mới; winreport 117 / H1 / H3 / H4 × bộ nhớ / PG 0 dòng khác ngoài dòng thời gian; smoke 21/21.
