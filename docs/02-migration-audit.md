# EAIS-MIGRATION-AUDIT — từ dịch vụ answer cũ sang EAIS

> Viết cho: chủ dự án và người thực thi chuyển đổi.
> Nguồn chuyển: dịch vụ trả lời của hệ thống cũ (Go phục vụ + Python nạp). Đo ngày 19/10/2026.
> Cách chuyển: **strangler** — phần RAG cũ không bị vứt mà hạ thành một capability (`intelligence/rag`) bên trong kiến trúc ở [01-architecture.md](01-architecture.md).

---

## 0. Bảy hành động

| Hành động | Nghĩa |
|---|---|
| `KEEP` | giữ logic, chỉ đổi chỗ và tên import |
| `MOVE` | dời sang package mới, đổi kiểu cho khớp hợp đồng EAIS |
| `GENERALIZE` | bỏ giả định "một bot, một loại dữ liệu"; nhận cấu hình từ pack |
| `DECOUPLE` | tách một khối làm nhiều việc thành nhiều thành phần theo hợp đồng |
| `REPLACE` | viết lại vì sai ranh giới (ví dụ phần Python ngoài việc đọc tài liệu chuyển sang Go) |
| `DELETE` | bỏ, không chuyển |
| `NEW` | EAIS cần mà hệ cũ không có |

Mỗi dòng bên dưới có: tệp nguồn · cỡ · việc hiện tại · hành động · đích EAIS · khe phải sửa **ngay khi chuyển** (mã `F..` là phát hiện của lượt soát độc lập phần Go ngày 19/10/2026).

---

## 1. Mốc đo trước khi chuyển

| Phần | Lệnh | Kết quả |
|---|---|---|
| Go build + vet | `go build ./... && go vet ./...` | sạch |
| Go test (không DB) | `env -u ANSWER_DATABASE_URL go test ./internal/... ./cmd/server/... ./cmd/room-sweep/... -count=1` | 151 PASS · 18 SKIP · 0 FAIL |
| Go race | `go test -race ./internal/... -count=1` | không có data race |
| Python test (không DB, bỏ ca chậm) | `pytest -p no:cacheprovider -m "not slow"` trong `ingest/` | 785 passed · 38 deselected · 3,92 s |

Chưa chạy: 18 test Go cần DB, các test Python cần DB, ca `slow` (thư viện đọc tài liệu thật).

---

## 2. Phần Go (6.450 dòng mã, 267 hàm test)

| Tệp nguồn | Dòng | Việc hiện tại | Hành động | Đích EAIS | Sửa khi chuyển |
|---|---:|---|---|---|---|
| `internal/flow/flow.go` | 748 | toàn bộ một lượt trả lời trong một hàm | `DECOUPLE` | `core/pipeline` (S0–S10) · `intelligence/context` (dựng prompt, lịch sử) · `reliability/validation` | F01 không kiểm số trong câu trả lời · F02 không ngưỡng liên quan · F03 cùng `message_id` khác câu · F04 ghi DB không hạn giờ · F06 lịch sử mở đầu bằng tin bot · F16 prompt không mang tên/phiên bản tài liệu · F17 không trần ngân sách prompt · F19 câu truy vấn nối không trần · F30 không chuẩn hoá đầu vào · F31 luôn ghép câu trước |
| `internal/flow/strategy.go` | 164 | chọn fulltext/retrieval theo cỡ corpus | `GENERALIZE` | `intelligence/rag` — cách RAG lấy ngữ cảnh; **router** quyết có vào RAG hay không | router mới theo ý định (P13) |
| `internal/flow/cache.go` | 142 | khoá cache câu trả lời | `MOVE` | `intelligence/models` (cache câu trả lời) | khoá thêm `pack_version` + tham số truy xuất |
| `internal/flow/fault.go` | 82 | nhãn lỗi ai sai | `KEEP` | `core/contracts` (thêm `policy`, `data`) | — |
| `internal/flow/trace.go` | 108 | đo từng chặng, `unaccounted_ms` | `KEEP` | `observability/traces` | F25 ghi khi lỗi phải nằm trong đồng hồ |
| `internal/flow/ports.go` | 296 | interface của luồng cũ | `REPLACE` | `core/contracts` + interface phía bên dùng | — |
| `internal/flow/options.go` | 53 | tuỳ chọn một lượt | `MOVE` | `core/pipeline` | — |
| `internal/store/store.go` | 453 | tra bot, đo corpus, đọc corpus, hybrid search, kiểm schema, giao dịch có tenant | `DECOUPLE` | `governance/tenancy` (giao dịch có tenant) · `intelligence/rag/retrieval` (hybrid search) · `platform/store` (kiểm schema) | F14 đo corpus mỗi lượt · F15 sắp xếp kéo theo nội dung · F32 tsquery OR mọi từ · F38 quá nhiều vòng DB |
| `internal/store/retriever.go` | 71 | nhúng câu hỏi + tìm | `MOVE` | `intelligence/rag/retrieval` | F24 tách đồng hồ nhúng/SQL |
| `internal/store/room.go` | 118 | lịch sử phòng chat + nhật ký lượt + dọn | `DECOUPLE` | lịch sử → memory của pipeline · nhật ký → `interactions` | F34 dọn với `keep=1` cắt mọi phòng |
| `internal/store/cache.go` | 96 | bảng cache câu trả lời | `MOVE` | `intelligence/models` | F11 không dọn · F12 không có test |
| `internal/store/pricing*.go` | 150 | bảng giá model 4 khoản | `MOVE` | `observability/cost` | F35 giá không có hiệu lực |
| `internal/llm/client.go` | 196 | client tương thích Anthropic | `MOVE` + `GENERALIZE` | `intelligence/models` | F07 lỗi HTTP không nhãn · F08 thử lại vượt hạn tổng · F10 hard code tham số/model/URL · F13 lý do dừng lạ vẫn ra 200 |
| `internal/llm/gemini.go`, `adapter.go` | 285 | client Gemini + chuẩn hoá kết quả | `MOVE` | `intelligence/models` | F07 · F13 |
| `internal/embed/jina.go` | 191 | nhúng câu hỏi, có nhãn lỗi nhà cung cấp | `MOVE` | `intelligence/rag/embeddings` (dùng chung cho nạp và hỏi) | F19 cắt đầu vào quá dài |
| `internal/config/config.go` | 147 | đọc cấu hình từ DB, sai kiểu thì nuốt | `REPLACE` | `core/tenant` (pack kiểm lúc khởi động) + cấu hình nền tảng từ env | F09 cấu hình hỏng phải chặn khởi động |
| `internal/domain/domain.go` | 94 | Bot, Corpus, Chunk | `REPLACE` | `core/contracts` | — |
| `internal/httpapi/api.go` | 363 | handler `/v1/answer`, kiểm 4 khoá, dòng log kết thúc | `DECOUPLE` | `platform/httpapi` · `governance/auth` · `observability/traces` | tenant lấy từ **khoá API**, không từ body · F05 ký tự NUL/độ dài · F21 nhãn `input` · F22 request id khi panic · F36 404/405 trả JSON |
| `internal/httpapi/requestid.go` | 76 | request id | `MOVE` | `observability/traces` | — |
| `internal/app/app.go` | 362 | lắp ráp toàn bộ | `REPLACE` | `cmd/eais` | F20 thiếu retrieval mà vẫn khởi động · F27 lộ DSN dạng key=value · F29 không chặn vai vượt RLS · F33 timeout server |
| `cmd/server` | 57 | main | `REPLACE` | `cmd/eais serve` | — |
| `cmd/room-sweep` | 93 | dọn lịch sử | `GENERALIZE` | `cmd/eais` job lưu giữ, chính sách theo pack | — |
| `cmd/golden` | 2.105 | runner: chấm theo sự kiện phải có/không được có, recall@k, cổng nguồn | `GENERALIZE` | `evals` — runner chung cho mọi tenant, ca theo [CONVENTIONS §5](CONVENTIONS.md) | giữ: chấm theo ranh giới token, exit 2 khi lỗi hạ tầng |

---

## 3. Phần Python (≈4.000 dòng mã, 20 tệp test)

Theo [01-architecture.md §11](01-architecture.md) ("Go first, Python by evidence"), ở V1 Python chỉ còn **đọc tài liệu**. Mọi thứ khác sang Go.

| Tệp nguồn | Dòng | Việc hiện tại | Hành động | Đích EAIS |
|---|---:|---|---|---|
| `adapters/parser.py` | 789 | nhận dạng loại tệp + đọc xlsx/docx/html/csv | `KEEP` (Python) | `python/document-parser` — chỉ trả `DocumentStructure` theo `schemas/`: trang · heading · đoạn · bảng (header, hàng, ô) · độ tin cậy |
| `adapters/fetcher.py` | 637 | tải URL, xuất Google, chặn SSRF, trần byte | `REPLACE` (Go) | `data/ingestion` — giữ nguyên luật chặn SSRF và trần |
| `chunker.py` | 246 | chunk theo cấu trúc, bảng theo nhóm hàng lặp header | `REPLACE` (Go) | `intelligence/rag/chunking` |
| `chunk_audit.py` | 433 | 7 cổng chất lượng chunk | `REPLACE` (Go) | `intelligence/rag/chunking` |
| `adapters/embedder.py` | 343 | nhúng theo lô, giãn nhịp theo hạn mức | `REPLACE` (Go) | `intelligence/rag/embeddings` |
| `adapters/writer.py` | 209 | ghi tài liệu/chunk, tsvector `simple`, phiên bản, thay nguyên tử | `REPLACE` (Go) | `intelligence/rag/indexing` |
| `pipeline.py` | 402 | 8 chặng nạp có đồng hồ | `REPLACE` (Go) | luồng I1 |
| `app.py` | 271 | HTTP `/v1/ingest` | `REPLACE` | `platform/httpapi` + API của parser service |
| `faults.py`, `trace.py`, `domain.py`, `ports.py`, `settings.py`, `migrate.py` | 637 | hạ tầng nội bộ | `REPLACE` | bản Go tương ứng |
| `ingest/migrations/0001–0017` | — | schema, RLS, vai, phiên bản tài liệu, giá | `REPLACE` | `db/migrations` (schema đa tenant mới) — **giữ ý**: RLS `FORCE`, vai không vượt RLS, thay tài liệu nguyên tử, số phiên bản tài liệu, tsvector `simple`, giá 4 khoản |
| `ingest/tests` (20 tệp) | 6.762 | test | `GENERALIZE` | ca thuần logic thành vector test Go; ca đọc tài liệu ở lại parser service; **bỏ mọi link và dữ liệu thật** |

---

## 4. Không chuyển

| Thứ | Vì sao |
|---|---|
| `golden/` (bộ ca, link nguồn, văn bản thô) | dữ liệu thật của tổ chức; repo EAIS là public |
| link Google trong 8 tệp test Python | như trên |
| `reports/`, `WORKFLOW.md`, `VERIFY_PLAN.md` | tài liệu của hệ cũ; bài học đã chưng cất thành ADR và `docs/requirements/` |
| `.venv`, `.pytest_cache`, `.coverage`, `scripts/sao-luu-db.sh` | sản phẩm phụ của máy; EAIS có script sao lưu riêng cho DB của nó |

---

## 5. NEW — thứ hệ cũ không có, và ca hỏng đòi nó

| Capability | Ca hỏng cụ thể đòi nó |
|---|---|
| `data/normalization` | giá `1.602.000` bị trả thành `1.600.000đ` không ai chặn (F01); `4k`, `205 55 r16`, `16/9` đi thẳng vào truy vấn (F30) |
| `data/entityresolution` | "máy 03", "MX-03", "máy xúc số 3" là một máy; "Man Utd"/"Manchester United" ở hai provider |
| `data/sourceoftruth` (observation → fact → derived) | đội trưởng báo 4.000 L, kho xuất 3.850 L, cảm biến 3.782 L: bốn chỉ số khác nhau, không phải bốn phiên bản của một số |
| `intelligence/intent` + `intelligence/router` | "xin chào" vẫn chạy truy xuất + gọi model; câu số liệu đi RAG |
| `intelligence/context` nhiều nguồn | câu "máy 03 có vượt định mức không" cần cả số thực tế (SQL) lẫn định mức (tài liệu) |
| `reliability/validation` | câu trả lời sai số vẫn HTTP 200 (F01) |
| `reliability/clarification` | thiếu dự án/máy thì model đoán |
| `execution/workflows` + `triggers` + lệnh | "tạo phiếu kiểm tra máy EX-03", "nhiệt độ EX-03 vượt ngưỡng" không biểu diễn được trong hệ chỉ biết hỏi–đáp |
| `execution/tools` | phép tổng, chênh lệch, tra trạng thái phải là mã có hợp đồng, không là chữ của model |
| `governance/auth` + `permissions` | hệ cũ tin `tenant_id` trong body; ai tới được cổng là hỏi được dữ liệu mọi tenant |
| `domain/sportsevents` | sự kiện trùng, đến muộn, bị huỷ cần event store chỉ-ghi-thêm có phiên bản |
| `evals` thành phần | hệ cũ chỉ đo đầu-cuối nên không biết sai ở tầng nào |

Ca hỏng chi tiết theo tenant: `docs/tenants/construction/07-failure-catalog.md`, `docs/tenants/sports/07-failure-catalog.md`.

---

## 6. Thứ tự chuyển và cổng chứng minh

> **Cập nhật 20/10/2026 (ADR-0001):** thứ tự 9 pha dưới đây xếp theo công nghệ và **đã được thay** bằng các lát W của `04-core-mvp.md` §8 (WIN) và các Phase của `05-master-roadmap.md` §3. Chữ "MVP" trong bảng không còn nghĩa là WIN. Bảng giữ lại để tra cổng chứng minh của từng nhóm tệp khi dời.

| Pha | Việc | Cổng (lệnh phải xanh) |
|---|---|---|
| 0 | kiểm kê + mốc đo (tài liệu này) | — |
| 1 | module Go, `core/contracts`, `observability/traces` + nhãn lỗi (KEEP), nạp + kiểm pack | `go build`, `go vet`, `go test`; test "core không chứa tên tenant" |
| 2 | `db/migrations` mới, `governance/tenancy`, `governance/auth` | test RLS: tenant A đọc dữ liệu tenant B ra 0 dòng; vai vượt RLS bị từ chối khởi động |
| 3 | `data/normalization`, `data/entityresolution`, `data/sourceoftruth` | 100% vector chuẩn hoá của bộ MVP; ca bốn nguồn nhiên liệu ra đúng bốn chỉ số + derived fact chênh lệch |
| 4 | `intent`, `router`, `context`, `validation`, mẫu câu → `/v1/ask` cho cầu đường (SQL_QUERY · CLARIFICATION · REJECT) | ca MVP route đúng; validator chặn ca sai số |
| 5 | RAG: parser service + chunk/nhúng/index/truy xuất bằng Go | chunk khớp vector test chuyển từ Python; recall trên corpus giả lập; ca RAG + HYBRID của MVP |
| 6 | workflow engine + lệnh + luật kích hoạt (sự cố bảo dưỡng, đề nghị vật tư) | ca COMMAND/EVENT/WORKFLOW của MVP; gửi lệnh lần hai không thực thi lại |
| 7 | tenant thể thao: `domain/sportsevents` + pack + bộ giả lập provider | ca MVP thể thao; **diff của core = 0** khi thêm tenant thứ hai (nếu phải sửa core thì ghi ADR vì sao) |
| 8 | runner eval + báo cáo chỉ số thành phần + ma trận capability xuyên tenant | báo cáo sinh từ lệnh, không viết tay |
