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

**Vận hành:** chỉ nạp tài liệu có phiên bản khi mọi binary trước U6a đã dừng — binary cũ đọc bảng mới nhưng bỏ qua cột trạng thái / hiệu lực / phạm vi, sẽ phục vụ bản nháp như bản chính. Không có migration ngược.

**Việc còn cần team (không tự sửa):**
1. ~~`platform/pgstore` từ chối tài liệu có phiên bản~~ — không cần nữa: U6a (23/10) cho PG lưu và phục vụ đủ phiên bản.
2. ~~`cmd/eais` đặt `MemoryIndex.Now` / `InScope`~~ — không cần nữa: pipeline truyền ngày hỏi và luật phạm vi của lượt cho
   mọi chỉ mục (bộ nhớ, PG), kể cả khi phát lại.
3. `tenants/construction/pack/policy.yaml`: các ca RAG dùng vai `site_supervisor`, `equipment_manager` chưa có trong policy.
