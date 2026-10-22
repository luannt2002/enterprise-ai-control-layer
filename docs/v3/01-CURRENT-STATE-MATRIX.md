# 01 — Current State Matrix: repo có gì so với EACL V3

> **Khung (D-RM-05, 22/10/2026):** tệp để tra cứu — roadmap hiện hành là `docs/ROADMAP-DAY-DU.md` (D-RM-01). Tên sản phẩm trong thân tệp là tên lúc viết; tên hiện hành là **EAIS** (D-RM-02).

> Trạng thái: đo 20/10/2026 trên HEAD `85108c4` (cây sạch). Mốc test ở `02`.
> Luật: **CODE EXISTS ≠ CAPABILITY EXISTS.**
> - REACHABLE = nối vào đường chạy thật.
> - TESTED = test liên quan thật sự chạy và đạt.
> - VERIFIED = runtime khớp trạng thái hiện tại.
>
> Trạng thái chấm theo bằng chứng đã thấy (lệnh, test, tệp), không theo tên tệp.
> Cột "Fuel" là bài toán đang có, chạy trên dữ liệu giả lập D2. Cột "Customer Ops" là MVP V3; ở cột này hầu hết là NOT_FOUND, vì repo chưa có domain này.

## 1. Ma trận

Trạng thái: NOT_FOUND · DEFINED_ONLY · IMPLEMENTED · REACHABLE · TESTED · FAILURE_TESTED · VERIFIED · VALUE_PROVEN.

| # | Capability | V3 cần | Fuel | Customer Ops | Bằng chứng · tệp | Còn thiếu | Phase |
|---|---|---|---|---|---|---|---|
| 1 | Identity (Principal) | có | TESTED (trong tiến trình) | NOT_FOUND | `core/contracts` `Principal`; Principal dựng từ registry/khoá, không từ request (`architecture_test.go`, đột biến TA/TB) | danh tính từ xác thực thật; ủy quyền thay người dùng (on-behalf-of) sang app | R2, R3 |
| 2 | Tenant | có | FAILURE_TESTED · VERIFIED (dev, D2) | NOT_FOUND | `core/tenant`, `platform/pgstore`, `db/migrations/0001_win.sql`: RLS FORCE theo GUC `eacl.tenant_id`; canary B; `winreport -pg` 117/117 (`02`); đột biến PA, PD–PH bị bắt | tenant trong app tham chiếu; truyền tenant qua adapter | R2 |
| 3 | Authentication | có | NOT_FOUND | NOT_FOUND | STATUS: "HTTP API, xác thực, `cmd/eacl`: chưa có" | toàn bộ: token, phiên, hết hạn | R2 |
| 4 | Authorization | có | FAILURE_TESTED | NOT_FOUND | `intelligence/router` `Allows` đóng mặc định (đột biến I, J; lỗi `Allows("")` đã sửa); `entity_scope`; ACL tài liệu hai lớp SQL + Go; ca `permission` 18/18 | quyền `customer.read`, `task.create` ở app; kiểm lại quyền lúc thực thi | R2, R4 |
| 5 | Entity Resolution | có | FAILURE_TESTED | NOT_FOUND | `data/entityresolution`: bí danh dài nhất, mã theo nguồn, mơ hồ ⇒ ứng viên; ca `entity` 13/13, `ambiguity` 13/13; đột biến K–O | khách hàng trùng tên (cùng tenant, khác tenant); dữ liệu ứng viên lấy từ app | R3 |
| 6 | Business Semantics | có | FAILURE_TESTED | NOT_FOUND | pack `metrics.yaml` tách issued/consumed/reported/billed…; ca `semantic` 14/14 | nghĩa nghiệp vụ Customer Ops (task mở/quá hạn, hoạt động gần nhất…) | R3 |
| 7 | Source Authority | có | FAILURE_TESTED (có lỗ, xem ghi chú 1) | NOT_FOUND | `data/sourceoftruth`: thẩm quyền trong từng chỉ số, dung sai, ngang hạng ⇒ hỏi người; ca `source_conflict` 12/12, ca biên mọi cặp nguồn | phạm vi ghi chú xung đột ở câu tổng (ca 115); với Customer Ops, app là nguồn có thẩm quyền duy nhất (adaptive truth, Blueprint §13) | R1, R3 |
| 8 | Normalization | có | FAILURE_TESTED | NOT_FOUND (cần ngày tương đối "thứ Hai") | `data/normalization`: số Việt, đơn vị, ngày tương đối, luật dấu `ToneKey`; vector NRM-V-001…049; metamorphic ≥ 4 biểu diễn | quy tắc "thứ Hai" khi hôm nay là thứ Hai; múi giờ tenant cho hạn task | R4 |
| 9 | Intent | có | TESTED | NOT_FOUND | `intelligence/intent`: theo luật, khớp dài nhất, mệnh đề trọng tâm | ý định Job A (brief, đếm fact chính xác), Job B (tạo follow-up); intent bằng model đóng mặc định | R3, R4 |
| 10 | Router | có | FAILURE_TESTED | NOT_FOUND | `intelligence/router`; 7 route WIN; ca `routing` 17/17 đếm `model_calls` bằng gateway giả; đột biến BA/BB | route `API`, `ACTION_PROPOSAL`, `MODEL` (tổng hợp) theo Blueprint §9; `DIRECT_LLM` có khai báo nhưng không dùng | R3, R4 |
| 11 | Context Builder | có | REACHABLE · TESTED (qua đầu–cuối) | NOT_FOUND | `contracts.ContextPackage`; chặng S7 trong `core/pipeline` (856 dòng, **0 dòng test riêng**; phủ qua 18 test `tenants/construction`) | ngữ cảnh khách hàng có lọc quyền, gắn nguồn/phiên bản, ngân sách, tách dữ liệu khỏi chỉ thị | R3 |
| 12 | SQL capability | có | FAILURE_TESTED | — | `platform/pgstore` (RLS); store bộ nhớ; hai đường ra từng byte giống nhau (scale smoke) | không dùng cho Customer Ops: app là chủ dữ liệu, EACL đọc qua API | — |
| 13 | API capability (gọi API nghiệp vụ) | có | NOT_FOUND | NOT_FOUND | HTTP client duy nhất là client model | adapter Customer Ops: `get_customer`, `find_customer`, `list_activities`, `list_tasks`, `get_documents`, `create_task`, `get_task` (Blueprint §30) | R3 |
| 14 | Search (toàn văn) | tuỳ | IMPLEMENTED (bộ nhớ, IDF theo dòng SOP) | NOT_FOUND | ADR-0002 D9; chỉ mục toàn văn PG chưa có | chỉ khi ca cần | DEFER |
| 15 | RAG | tuỳ | FAILURE_TESTED | NOT_FOUND | `intelligence/rag`: chỉ mục bộ nhớ, ACL theo vai, tenant; ca chèn lệnh SOP không đổi được route/quyền/số; lỗi không tất định đã sửa | Job A: tài liệu theo `customer_id` + kỳ là truy vấn tất định, RAG chỉ khi ca chứng minh cần | R3 (tuỳ) |
| 16 | Model Gateway | có | TESTED (model giả + máy chủ giả) | NOT_FOUND | `intelligence/models`: `Gateway`; `extractive` (giả, tất định, đếm lời gọi); `anthropic` (Messages API); cấu hình qua `EACL_MODEL_*`, không hard code. **Chưa từng gọi API thật** (STATUS) | khoá + ngân sách (quyết định D-V3-01); timeout/thử lại theo nhãn lỗi; ghi chi phí | R1, R3 |
| 17 | Structured Output | có | TESTED (máy chủ giả) | NOT_FOUND | `messages.go` đọc JSON `{answer, evidence_ids}` thành `Draft{Text, Claims}` | schema brief nhiều mục; schema `ActionProposal`; xử lý JSON hỏng | R3, R4 |
| 18 | Validation | có | FAILURE_TESTED | NOT_FOUND | `reliability/validation`: `Fidelity`, `BindClaims` 6 chiều, `ForbidDimensions`/`ForbidUnitWords`; bảng "sai đúng một chiều" 6 × 3 bị chặn 100%; 0 chặn oan (13 + 117 ca) | kiểm brief (id bằng chứng có thật, số/ngày khớp fact); kiểm đề xuất hành động | R3, R4 |
| 19 | Evidence | có | VERIFIED (dev, D2) | NOT_FOUND | `contracts.Evidence` mang chủ thể/chỉ số/kỳ; trích dẫn; phát lại 117/117 không lệch | bằng chứng là id bản ghi app + phiên bản; lưu bền | R3, R4 |
| 20 | Policy (nghiệp vụ, rủi ro) | có | NOT_FOUND (mới có quyền capability) | NOT_FOUND | — | luật rủi ro hành động, điều kiện cần duyệt, kiểm lại policy lúc thực thi | R4 |
| 21 | Permission | có | xem #4 | NOT_FOUND | — | — | R2, R4 |
| 22 | Approval | có (cho hành động rủi ro cao) | NOT_FOUND | NOT_FOUND | `HUMAN_ESCALATION` chỉ trả `HUMAN_REVIEW_REQUIRED`, không tạo việc duyệt (04 §4) | móc duyệt tối thiểu | R4 |
| 23 | Action Proposal | có | NOT_FOUND | NOT_FOUND | grep `Proposal`: 0 tệp | kiểu `ActionProposal` (Blueprint §12) | R4 |
| 24 | Idempotency | có | NOT_FOUND | NOT_FOUND | grep `idempot`: 0 tệp | khoá idempotency phía app + khoá suy ra phía EACL | R2, R4 |
| 25 | Execution | có | NOT_FOUND | NOT_FOUND | — | executor chỉ nhận `AuthorizedAction` | R4 |
| 26 | Read-back Verification | có | NOT_FOUND | NOT_FOUND | — | đọc lại theo id/khoá, so với đề xuất | R4 |
| 27 | Audit (lưu bền) | có | NOT_FOUND | NOT_FOUND | bản ghi quyết định chỉ ở bộ nhớ/test (STATUS: "chưa lưu bền") | bảng audit/bằng chứng có RLS | R4 |
| 28 | Tracing | có | TESTED | NOT_FOUND | `TraceID` mỗi phản hồi; `observability/traces` `Decision` + `Diff` + đo thời gian chặng; `Replay` | trường V3 (Blueprint §27): adapter, phiên bản ngữ cảnh, đề xuất, kết quả thực thi, đọc lại, chi phí; lưu bền | R3, R4 |
| 29 | Cost Attribution | có | DEFINED_ONLY | NOT_FOUND | `contracts.Usage` (token); `g7run` chỉ "báo cáo" độ trễ/chi phí | chi phí theo job/capability/provider/thử lại/kết quả; bảng giá cấu hình được, không hard code (Blueprint §25) | R6 (đặt móc từ R3) |
| 30 | Fuel Benchmark | có | VERIFIED cho cổng cũ G1–G6 (D2, 117 ca); **không** VALUE_PROVEN | — | `tenants/construction/win/*`, `winreport`, `TestWinRatchet`; G7 cũ CHẶN | B1 LLM trực tiếp; chạy model thật k = 3; ước chi phí; câu trả lời WIN/PARTIAL/FAIL | R1 |
| 31 | Customer Operations | có | — | NOT_FOUND | grep `Customer`, `Task`: 0 tệp | app tham chiếu tất định + vertical EACL | R2–R4 |
| 32 | Value Benchmark | có | IMPLEMENTED (B2 + bộ đếm D5, mới chạy bằng model giả) | NOT_FOUND | `evals/baseline` (8 test), `evals/g7.go`, `g7run` | B0 người, B1 LLM trực tiếp; Customer Ops cả bốn đường | R1, R6 |
| 33 | Security Tests | có | FAILURE_TESTED (đường QUERY) | NOT_FOUND | chéo tenant (SQL, truy xuất, nhận dạng, bằng chứng), quyền, chèn lệnh SOP, AST tenant | đường hành động: model tự cấp quyền, lối vào thay thế, chèn lệnh trong hoạt động khách hàng | R5 |
| 34 | Failure Injection | có | TESTED một phần | NOT_FOUND | đột biến mã (~90, `scripts/dot-bien/`); nhãn lỗi `Fault` (timeout, rate_limit, 5xx, bad_response); `TestMessagesGateway` dùng máy chủ giả | lỗi lúc chạy: provider timeout, JSON hỏng, API timeout, ghi xong mất phản hồi, đọc lại lệch, đổi quyền giữa chừng; lớp thử lại theo nhãn | R4, R5 |

**Ghi chú 1 — lỗ phạm vi ghi chú xung đột (quan sát mới, E4).**
- Ca WIN-I-115 hỏi "Dự án A hôm qua xuất bao nhiêu dầu?". Đáp 6.550 L đúng.
- Câu giao kèm: "Nguồn khác lệch số: ERP 3.900 L". Số 3.900 L là của **riêng máy EX-03**, nhưng câu không nói chủ thể, nên người đọc hiểu đó là số của cả dự án.
- Nguyên nhân: mẫu `conflict_note` trong `tenants/construction/pack/templates.yaml` chỉ in `{nguồn số}`.
- Ca vẫn ĐẠT vì kỳ vọng không cấm chuỗi "3.900". Bên trong, `BindClaims` biết claim thuộc EX-03; chỉ câu chữ giao cho người là sai phạm vi.
- Việc sửa: TASK-R1-02.

## 2. Gộp theo câu hỏi của V3

| Câu hỏi | Trả lời từ bằng chứng |
|---|---|
| Truth Proof (Fuel) đã có gì? | Cổng cũ G1–G6 có bằng chứng ở mức FAILURE_TESTED/VERIFIED trên D2. Câu hỏi giá trị chưa có câu trả lời: chưa so model thật, chưa có B1 |
| Product Proof (Customer Ops) đã có gì? | **Chưa có gì của domain.** Có các khối dùng lại được: tenant/RLS, Principal, quyền đóng mặc định, nhận dạng thực thể, ràng buộc claim, gateway model, trace + phát lại, nhãn lỗi |
| Đường hành động (COMMAND) đã có gì? | Không có: đề xuất, idempotency, thực thi, đọc lại, audit, duyệt đều NOT_FOUND |
| Portability đã có gì? | Mới có luật (`architecture_test.go` cấm tên tenant trong mã chung). Chưa có domain thứ hai: thể thao bị DEFER, canary B cùng ngành |

**Tiến độ MVP EACL theo một con số:** **0/17 ô của MVP DoD (Playbook §7.6) = 0%.** Job A, Job B chưa tồn tại.

Fuel (R1) báo riêng: 6/7 cổng cũ, cổng VALUE còn CHẶN. Đừng cộng hai số này với nhau.
(Đính chính 20/10 tối: vô điều kiện chỉ 3/7 — G1, G3, G4. G2, G5, G6 đạt có điều kiện, chờ D-G7-11 ở `docs/CHO-QUYET-DINH.md`.)

## 3. Cây thư mục hiện tại so với hình đích (Blueprint §19)

Blueprint tự dặn **không tái cấu trúc ngay**; hình đích chỉ áp sau R7 khi có bằng chứng.

| Hiện tại | Vai trò | Ứng với đích (R7, nếu bằng chứng ủng hộ) |
|---|---|---|
| `core/contracts`, `core/tenant`, `core/decimal` | kiểu chung, tenant | `core/identity`, `core/evidence`… |
| `core/pipeline` | luồng QUERY S0–S10 | `core/routing` + `core/context` (sau khi so với Customer Ops) |
| `data/normalization`, `data/entityresolution`, `data/sourceoftruth`, `data/ingestion` | truth của Fuel | `core/entity`, `core/semantics`, `core/source` (chỉ phần dùng chung) + `verticals/fuel_truth` |
| `intelligence/intent`, `intelligence/router` | ý định, định tuyến | `core/intent`, `core/routing` |
| `intelligence/rag`, `intelligence/models` | truy xuất, gateway | `capabilities/retrieval`, `capabilities/models` |
| `reliability/validation` | kiểm claim | `core/validation` |
| `observability/traces` | trace, phát lại | `core/audit` |
| `platform/pgstore`, `db/migrations` | lưu trữ Fuel | `integrations/sql` |
| `evals`, `evals/baseline` | runner, baseline | `evals/truth`, `evals/value` |
| `tenants/construction` | pack + WIN | `verticals/fuel_truth` |
| (chưa có) | app tham chiếu | `examples/reference_customer_ops` |
| (chưa có) | vertical Customer Ops trong EACL | `verticals/customer_ops` + `integrations/customer_ops` |

Đề xuất đặt chỗ cho mã mới ở `06` (TASK-R2-01).
