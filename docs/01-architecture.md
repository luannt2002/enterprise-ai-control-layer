# Kiến trúc EAIS

> **EAIS is a multi-tenant Enterprise AI Systems Platform for turning domain-specific business data, knowledge, workflows, and tools into governed, testable AI capabilities without coupling the core platform to a single industry.**

Tài liệu này mô tả kiến trúc **đích** và **luồng mới**. Quy ước tên gọi ở [CONVENTIONS.md](CONVENTIONS.md). Lý do của từng lựa chọn ở [adr/](adr/).

> **Phạm vi hiện hành (V3):** EAIS không có workflow engine riêng; ứng dụng doanh nghiệp giữ trạng thái, giao dịch và workflow (`docs/v3/05-R0-DEFINITION-FREEZE.md`: Phase 2 Workflow và Phase 3 Event không có trong V3). Các phần workflow / command / event / human task bên dưới là kiến trúc đích V2, đang hoãn.

---

## 1. Ý tưởng trung tâm

```text
        SAI (model-centric)                    ĐÚNG (system-centric)

           RAG / MODEL                                EAIS
               ▲                          ┌────────────┼────────────┐
               │                          │            │            │
         mọi thứ khác                    DATA      DECISION      ACTION
                                          └────────────┼────────────┘
                                                       │
                                               AI CAPABILITIES
                                              /        |        \
                                            RAG      Model     Agent
```

**Model chỉ là một capability xác suất nằm giữa một hệ phần mềm tất định.** Số liệu chính xác đến từ dữ liệu có cấu trúc; tri thức đến từ tài liệu có nguồn; quyết định đi đường nào do router theo luật; câu trả lời của model luôn bị kiểm bằng code trước khi tới người dùng.

Mười nguyên tắc ràng buộc mọi thiết kế bên dưới:

| # | Nguyên tắc | Hệ quả kỹ thuật |
|---|---|---|
| P1 | Model ≠ System | model chỉ gọi ở chặng `generate`, và chỉ khi route cần diễn đạt |
| P2 | Phần mềm tất định bao quanh lõi xác suất | chuẩn hoá, nhận dạng, định tuyến, dựng ngữ cảnh, kiểm đầu ra đều là code có test |
| P3 | Sự thật đến từ nguồn có thẩm quyền, không đến từ model | câu hỏi số liệu chính xác đi `SQL_QUERY`; model không bao giờ là nguồn của một con số |
| P4 | Garbage in → garbage out | dữ liệu được chuẩn hoá và kiểm **lúc nạp**, báo lỗi về chủ dữ liệu thay vì nuốt im |
| P5 | Mơ hồ thì hỏi lại hoặc từ chối, không đoán | trường bắt buộc thiếu/mơ hồ → `CLARIFICATION`; không đủ bằng chứng → từ chối có lý do |
| P6 | Nền tảng dùng chung ≠ tri thức dùng chung | khác biệt tenant nằm trong **pack** (dữ liệu cấu hình), core không có nhánh theo tên tenant |
| P7 | Độ phức tạp phải được bằng chứng đòi | capability vào core khi có ca hỏng tái lập được + test (ADR-0002) |
| P8 | Mọi câu trả lời truy được về nguồn | mỗi lượt ghi route, bằng chứng (nguồn + phiên bản), phán quyết kiểm, phiên bản pack |
| P9 | Benchmark ≠ chất lượng thật | đánh giá theo từng thành phần trên bộ ca của chính tenant, có ca bẩn/mơ hồ/xung đột |
| P10 | Sai thì tìm tầng sai trước khi đổi model | trace tách từng chặng; lỗi mang nhãn `fault` đúng tầng |
| P11 | Model được **đề xuất** chuyển trạng thái; **workflow engine sở hữu** trạng thái | trạng thái, phép chuyển, quyền, bất biến, giao dịch thuộc phần mềm; model chỉ đọc, phân loại, trích xuất, đề xuất, giải thích |
| P12 | Pack mô tả chính sách và cấu hình; **mã cài hành vi** | YAML không có điều kiện lồng, vòng lặp, biểu thức tự do; cần luồng điều khiển thì viết domain module có hợp đồng và test |
| P13 | Điều khiển tất định ≠ mọi thành phần phải tất định | control plane được phép dùng một classifier xác suất **dưới ngưỡng tin cậy và có đường lùi**: luật trước → classifier (có cổng) → không chắc thì hỏi lại |

---

## 2. Bốn mặt phẳng và ba lớp

```text
┌────────────────────────────────────────────────────────────────────┐
│ INTELLIGENCE   Router · Intent · Model Gateway · Skills · (Agent)  │
├────────────────────────────────────────────────────────────────────┤
│ CONTEXT        Context Engine · RAG · Facts · Events · Memory      │
├────────────────────────────────────────────────────────────────────┤
│ ACTION         Tool Runtime · Connectors · (MCP)                   │
├────────────────────────────────────────────────────────────────────┤
│ CONTROL        Tenancy · Auth · Permissions · Validation · Eval ·  │
│                Trace · Cost · Audit · SLA                          │
└────────────────────────────────────────────────────────────────────┘

        EAIS CORE  (mã dùng chung, không biết tenant nào)
            │
        TENANT RUNTIME  (nạp pack, cô lập dữ liệu, ngân sách, quyền)
            │
   ┌────────┴─────────┐
   ▼                  ▼
 construction       sports        ← pack: ontology · lexicon · schema · rules ·
 (abc-infra)      (xyz-sports)        intents · tools · skills · templates · evals
```

Dấu ngoặc `( )` là capability **có cổng**: chỉ bật khi có ca hỏng chứng minh workflow cố định không đủ (xem [§9](#9-capability-có-cổng)).

### 2.1 Ba lối vào: hỏi, lệnh, sự kiện

Doanh nghiệp không chỉ **hỏi**. Phần lớn giá trị nằm ở **lệnh** làm đổi trạng thái và **sự kiện** kích hoạt quy trình — ở đó model có thể không xuất hiện.

```text
                          EAIS ENTRY
                              │
           ┌──────────────────┼──────────────────┐
           ▼                  ▼                  ▼
         QUERY             COMMAND             EVENT
  "Hôm nay dùng bao   "Tạo phiếu kiểm tra   cảm biến/tin báo:
   nhiêu dầu?"          máy EX-03"           nhiệt độ EX-03 vượt ngưỡng
           │                  │                  │
           ▼                  ▼                  ▼
     ASK PIPELINE       ACTION PIPELINE    TRIGGER PIPELINE
       S0 … S10            C0 … C7             E0 … E6
           │                  │                  │
           └──────────────────┼──────────────────┘
                              ▼
     CAPABILITY PLANE: normalize · resolve · facts · retrieval · tools · model · validation
                              │
                              ▼
     WORKFLOW ENGINE (sở hữu trạng thái) ──► audit · thông báo · việc cho người (human task)
```

| Lối vào | Kết quả mong muốn | Ai quyết | Model có mặt khi |
|---|---|---|---|
| QUERY | câu trả lời có nguồn | router + nguồn sự thật | cần diễn đạt tri thức tài liệu |
| COMMAND | trạng thái đổi đúng quy trình, có audit | workflow engine (quyền + bất biến) | trích tham số từ câu tự do — kết quả vẫn phải qua kiểm schema |
| EVENT | quy trình được khởi động/đẩy tiếp | luật kích hoạt (trigger) + workflow engine | hầu như không; chỉ khi giải thích bất thường cho người |

### 2.2 Bốn loại vai trò — chỉ một loại nằm trong prompt

> Enterprise AI Engineering không chủ yếu là làm model thông minh hơn; nó là xây một hệ thống kiểm soát model **được thấy dữ liệu gì, hiểu ngữ cảnh nào, được quyết định gì, được hành động gì**, và chứng minh đầu ra/hành động của nó đáng tin.

| Loại | Ví dụ | Nằm ở đâu | Ai thực thi |
|---|---|---|---|
| 1. Vai nghiệp vụ (user role) | trưởng dự án, kế toán, kỹ sư hiện trường, giám đốc | `principals.roles` | `governance/permissions` |
| 2. Phạm vi dữ liệu (data access) | dự án nào, tài liệu nào, chỉ số nào được xem | thuộc tính ABAC của principal + ACL tài liệu | `governance/permissions` + lọc ở `sourceoftruth`, `rag`, `context` |
| 3. Quyền capability (system capability) | được tìm tài liệu, truy vấn số liệu, tạo báo cáo, duyệt, gọi tool nào | chính sách vai → capability trong pack | `governance/permissions` trước MỌI tool/lệnh |
| 4. Vai của model (model role) | "bạn hỗ trợ lập báo cáo ngày công trường" | mẫu prompt trong pack | **chỉ** định hướng văn phong — KHÔNG BAO GIỜ dùng để phân quyền |

Hệ quả: **model không bao giờ thấy dữ liệu mà người hỏi không được xem.** Lọc quyền xảy ra trước khi bằng chứng vào `ContextPackage`, không phải dặn model "đừng tiết lộ".

EAIS gồm bốn loại thành phần, và phần không phải AI chiếm đa số là dấu hiệu kiến trúc trưởng thành:

| Loại | Thành phần |
|---|---|
| Kỹ thuật phần mềm | auth, tenant, RBAC/ABAC, DB, API, giao dịch, trạng thái, workflow, audit, retry, timeout, idempotency, quan sát |
| Kỹ thuật dữ liệu | đọc tài liệu, chuẩn hoá, schema chuẩn, nguồn sự thật, dòng dõi dữ liệu, phiên bản, index, tìm kiếm, dựng ngữ cảnh |
| Trí tuệ xác suất | hiểu ngôn ngữ mơ hồ, phân loại ngữ nghĩa, truy xuất ngữ nghĩa, so khớp thực thể mờ, trích xuất, suy luận, tóm tắt, giải thích, sinh văn bản, thị giác, rerank học được, lập kế hoạch (agent) |
| Điều khiển | router, ngưỡng tin cậy, cổng capability, validation, chính sách lùi |

### 2.3 Workflow là khái niệm hạng nhất

Mọi thứ có vòng đời (phiếu đề nghị vật tư, sự cố bảo dưỡng, phiếu nghiệm thu, sửa dữ liệu trận đấu) là một **state machine (máy trạng thái)** khai báo trong pack và chạy bằng một engine dùng chung:

```text
Phiếu đề nghị vật tư:  DRAFT → SUBMITTED → REVIEWING → APPROVED → ORDERED → RECEIVED → RECONCILED → CLOSED
                                     └──────────► REJECTED
```

- Mỗi phép chuyển có **guard (điều kiện chặn)**: quyền của principal + bất biến nghiệp vụ. Guard là **tên** do pack chọn từ danh mục mã đã cài, không phải biểu thức trong YAML (P12).
- Bước chờ người (duyệt, kiểm tra hiện trường) là **human task** có hạn chót và người nhận.
- Model có thể gợi ý "nên chuyển sang APPROVED", nhưng chỉ engine thực hiện, sau khi guard qua (P11).
- MVP cài engine tối giản (đồng bộ, lưu trong PostgreSQL, timer quét định kỳ); không dùng engine ngoài cho tới khi số đo đòi.

---

## 3. Cây thư mục

Một module Go: `github.com/luannt2002/enterprise-ai-control-layer`. Go là ngôn ngữ của hệ thống và control plane; Python chỉ được tồn tại **sau ranh giới capability tường minh** khi hệ sinh thái ML/tài liệu/dữ liệu của Python mang lại giá trị đo được (§11). V1 có đúng một dịch vụ Python: `python/document-parser`.

```text
cmd/eais/                 một binary, nhiều lệnh con: serve · migrate · seed · simulate · eval
core/
  contracts/              kiểu dữ liệu dùng chung (Evidence, Span, RoutePlan, Verdict, Fault…)
  tenant/                 Tenant Runtime: nạp + kiểm pack, phiên bản pack
  pipeline/               điều phối 3 lối vào: ask (S0–S10) · command (C0–C7) · trigger (E0–E6)
data/
  ingestion/              connector: tải URL an toàn (chặn SSRF), xuất Google, nhận tin nhắn/bản ghi
  parsing/                client Go gọi python/document-parser; cổng chất lượng cấu trúc trả về
  normalization/          bộ máy chuẩn hoá: số, lượng + đơn vị, tiền, ngày/giờ, mã, viết tắt
  entityresolution/       nhận dạng thực thể: bí danh, so khớp mờ, ánh xạ theo provider
  sourceoftruth/          quan sát → đo lường → fact → fact dẫn xuất; ma trận thẩm quyền; hoà giải xung đột
intelligence/
  intent/                 phân loại ý định theo luật của pack
  router/                 ý định → kế hoạch route (12 route), ngưỡng tin cậy, đường lùi
  context/                Context Engine: gom bằng chứng nhiều nguồn, lọc quyền, khử trùng, xếp, ngân sách
  rag/                    chunking · embeddings · indexing · retrieval (hybrid dense + sparse + RRF)
  models/                 Model Gateway: nhà cung cấp theo cấu hình, structured output, giá
execution/
  tools/                  registry + runtime: schema vào/ra, quyền, timeout, idempotency, audit
  workflows/              workflow engine: state machine, guard theo tên, human task, timer — SỞ HỮU trạng thái
  triggers/               luật kích hoạt: sự kiện + điều kiện so sánh khai báo → lệnh/khởi động workflow
  skills/                 quy trình nhiều bước cho câu hỏi/báo cáo (ví dụ báo cáo ngày), chạy trên tool + engine
reliability/
  validation/             kiểm đầu ra: số, mã, ngày, trích dẫn, luật nghiệp vụ
  clarification/          câu hỏi lại + ứng viên
governance/
  auth/                   khoá API → principal (tenant lấy từ danh tính, KHÔNG từ body)
  permissions/            RBAC/ABAC: ai được xem dữ liệu nào, gọi tool nào
  tenancy/                phiên DB có `eais.tenant_id`, RLS
  audit/                  nhật ký chỉ-ghi-thêm có chuỗi băm
observability/
  traces/                 đo từng chặng, `unaccounted_ms`, request id, một dòng log mỗi lượt
  cost/                   bảng giá model, chi phí mỗi lượt
platform/
  httpapi/                HTTP (chi v5): /v1/ask · /v1/commands · /v1/events · /v1/workflows · /v1/tasks · /v1/ingest/* · /v1/health
  store/                  pool pgx, migration runner
domain/
  sportsevents/           domain module của thể thao: event store, khử trùng, sửa muộn, projection
db/migrations/            SQL migration (Go là bên duy nhất chạy migration)
tenants/
  construction/           pack + dữ liệu giả lập + bộ ca của ABC Infrastructure
  sports/                 pack + dữ liệu giả lập + bộ ca của XYZ Sports Data
simulations/              bộ sinh dữ liệu giả lập có seed (đầu ra đóng băng thành tệp)
evals/                    runner + chỉ số thành phần + E2E
schemas/                  JSON Schema của hợp đồng xuyên ngôn ngữ Go ↔ Python (một nguồn, hai bên cùng test)
python/                   dịch vụ tính toán chuyên dụng, mỗi thư mục một dịch vụ stateless có pyproject riêng
  document-parser/        V1: bytes → DocumentStructure (trang, heading, đoạn, bảng, ô, độ tin cậy)
docs/                     tài liệu (xem docs/README.md)
deploy/                   docker compose (PostgreSQL 17 + pgvector, document-parser)
```

Bảng ánh xạ từ cây trong brief gốc sang tên package Go nằm ở [ADR-0003](adr/0003-stack-and-layout.md). Thư mục chỉ được tạo khi có mã thật; `execution/agents` và `execution/mcp` chưa có cho tới khi cổng mở (§9).

---

## 4. Luồng hỏi (S0–S10)

```text
 request ──► S0 ACCEPT ──► S1 TENANT ──► S2 NORMALIZE ──► S3 RESOLVE ──► S4 INTENT
                                                                            │
     ┌──────────────────────────────────────────────────────────────────────┘
     ▼
 S5 ROUTE ──► S6 EXECUTE ──► S7 CONTEXT ──► S8 GENERATE ──► S9 VALIDATE ──► S10 RESPOND
  │              │  tools / facts / events / retrieval / calculation   │ fail
  │              │                                                     ▼
  └─ CLARIFICATION / REJECT / HUMAN_ESCALATION (không gọi model)   fallback / clarify / escalate
```

| Chặng | Việc | Tất định? | Đầu ra | Chỉ số |
|---|---|---|---|---|
| S0 accept | xác thực khoá API → principal; kiểm schema; idempotency theo `(conversation_id, message_id)` | có | `Principal`, `AskRequest` | tỉ lệ 4xx theo nhãn |
| S1 tenant | nạp Tenant Runtime (pack đã kiểm, có `pack_version`) | có | `*tenant.Runtime` | — |
| S2 normalize | chuẩn hoá văn bản + trích span có kiểu (số, lượng, tiền, ngày, mã, viết tắt) theo lexicon của pack | có | `NormalizedInput` | normalization accuracy |
| S3 resolve | span/cụm từ → thực thể chuẩn (dự án, máy, đội, cầu thủ) + độ tin cậy | có | `Resolution` | entity resolution accuracy |
| S4 intent | luật của pack chấm ý định + khe (slot) | có (model chỉ khi pack bật và có số đo đòi) | `Intent` | intent accuracy |
| S5 route | ý định + khe đủ/thiếu + quyền → `RoutePlan` | có | `RoutePlan` | router accuracy |
| S6 execute | chạy bước của kế hoạch: tool, truy vấn facts/events, truy xuất tài liệu, phép tính | có (trừ bước model của skill) | `[]Evidence` | tool success, recall@k |
| S7 context | khử trùng, lọc quyền, hoà giải nguồn, xếp, cắt theo ngân sách, gắn nguồn | có | `ContextPackage` | context precision |
| S8 generate | **chỉ khi route cần diễn đạt**: model nhận gói bằng chứng, trả câu + `claims` gắn mã bằng chứng; route số liệu dùng **mẫu câu của tenant** | không | `Draft` | groundedness |
| S9 validate | số/mã/ngày trong câu trả lời phải có trong bằng chứng; trích dẫn hợp lệ; luật nghiệp vụ; chính sách | có | `Verdict` | validator catch rate, false block rate |
| S10 respond | trả lời + trích dẫn + trace id; ghi `interactions` | có | `AskResponse` | latency, cost |

Ba luật cứng của luồng:

1. **Route không cần diễn đạt thì không gọi model.** `SQL_QUERY`, `CALCULATION`, `CLARIFICATION`, `REJECT`, `HUMAN_ESCALATION` trả lời bằng mẫu câu (template) do tenant viết. Ứng dụng không tự sáng tác chữ cho khách.
2. **Model không bao giờ là nguồn của con số.** Mọi con số trong câu trả lời phải khớp (sau chuẩn hoá) một con số trong gói bằng chứng, hoặc là kết quả của bước `CALCULATION` có ghi phép tính.
3. **Kiểm thất bại thì không giao câu đó.** Hành động lùi do pack quy định: `fallback` (mẫu câu "chưa đủ dữ liệu chính xác"), `clarify`, hoặc `escalate`.

Router theo P13:

```text
ý định từ luật ── đủ tin cậy? ──yes──► route
                     │ no
                     ▼
          classifier (CHỈ khi cổng đã mở) ── đủ tin cậy? ──yes──► route
                     │ no / cổng đóng
                     ▼
               CLARIFICATION (hỏi lại kèm ứng viên)
```

### 4.1 Pipeline lệnh (C0–C7)

```text
C0 accept (xác thực, idempotency key) ─► C1 tenant ─► C2 hiểu lệnh (chuẩn hoá + nhận dạng thực thể;
model chỉ trích tham số khi cổng mở) ─► C3 kiểm schema tham số ─► C4 quyền (RBAC/ABAC)
─► C5 workflow engine: guard + chuyển trạng thái trong MỘT giao dịch ─► C6 tool có tác dụng phụ
(idempotent, timeout, kết quả được kiểm) ─► C7 audit + thông báo + trả về trạng thái mới
```

- Lệnh thiếu tham số bắt buộc → trả về câu hỏi lại, **không** đoán.
- Lệnh có tác dụng phụ lớn (theo pack) → tạo human task chờ duyệt thay vì thực thi ngay.
- Gửi lại cùng idempotency key → trả kết quả cũ, không thực thi lần hai.

### 4.2 Pipeline sự kiện (E0–E6)

```text
E0 nhận sự kiện (tin nhắn, bản ghi, cảm biến, provider) ─► E1 chuẩn hoá + nhận dạng thực thể
─► E2 ghi quan sát (observation) có nguồn ─► E3 luật kích hoạt của pack: so sánh khai báo
(metric op ngưỡng, trạng thái = X) ─► E4 khởi động/đẩy workflow qua engine ─► E5 hành động
(tạo sự cố, gán người, thông báo) ─► E6 audit
```

- Luật kích hoạt chỉ gồm so sánh khai báo; logic phức tạp hơn là domain module (P12).
- Mỗi sự kiện xử lý **đúng một lần theo khoá khử trùng**; đến lại thì bỏ qua và ghi nhận.

---

## 5. Luồng nạp (I1–I4)

### 5.0 Bốn tầng của dữ liệu sự thật

```text
OBSERVATION     điều một nguồn đã nói/ghi, nguyên văn + nguồn + thời điểm
   │              "đội trưởng nhắn: dầu 4k" · "phiếu kho PX-0912: 3.850 L" · "cảm biến: 3.782 L" · "hoá đơn: 4.100 L"
   ▼ interpretation (chuẩn hoá, nhận dạng thực thể)
MEASUREMENT     giá trị có kiểu + đơn vị + thực thể + kỳ, vẫn gắn với quan sát gốc
   ▼ business semantics (chỉ số ngữ nghĩa do pack định nghĩa)
FACT            reported_fuel · issued_fuel · measured_consumption · billed_fuel   ← BỐN chỉ số KHÁC nhau
   ▼ authority / reconciliation (chỉ giữa các fact CÙNG chỉ số, cùng thực thể, cùng kỳ)
DERIVED FACT    chênh lệch issued − measured, tổng theo dự án, cờ bất thường — luôn ghi công thức + fact đầu vào
```

Hai luật rút ra:

1. **Ma trận thẩm quyền chỉ áp trong cùng một chỉ số ngữ nghĩa.** "Kho xuất 3.850" không *thắng* "đội trưởng báo 4.000" — chúng đo hai thứ khác nhau. Chênh lệch giữa chúng là một **derived fact** đáng điều tra, không phải xung đột cần chọn bên.
2. **Authority ≠ recency ≠ similarity ≠ số lần lặp.** Nguồn mới hơn, giống câu hỏi hơn hay được nhắc nhiều hơn đều không tự động đúng hơn.

### I1 — Tài liệu (SOP, hợp đồng, sổ tay thiết bị, quy định)

```text
connector (URL/tải lên) ─► parsing service (Python, stateless) ─► cấu trúc: heading · đoạn · bảng(header, hàng, ô) · trang
  ─► kiểm chất lượng đọc (bảng còn đủ ô? số còn đủ?) ─► chunking theo cấu trúc (bảng theo nhóm hàng, lặp header)
  ─► metadata (loại tài liệu, phiên bản, hiệu lực, ACL) ─► embedding (ghi model + số chiều + phiên bản)
  ─► index (tsvector 'simple' + vector) ─► công bố phiên bản mới NGUYÊN TỬ (một giao dịch)
```

Bảng số liệu trong tài liệu (ví dụ bảng định mức) còn được trích thành **bản ghi có kiểu** vào kho sự thật, để phép tính dùng số chứ không dùng chữ.

### I2 — Bản ghi có cấu trúc (ERP, phiếu kho, bảng tính)

```text
connector ─► observation (nguyên văn + nguồn) ─► ánh xạ về schema chuẩn của pack ─► chuẩn hoá giá trị
  ─► kiểm schema + luật nghiệp vụ (miền giá trị, chéo trường) ─► measurement ─► fact theo chỉ số ngữ nghĩa của loại nguồn
  ─► hoà giải với fact CÙNG chỉ số/thực thể/kỳ ─► xung đột vượt ngưỡng → hàng đợi duyệt ─► derived fact (chênh lệch, tổng)
```

### I3 — Tin nhắn hiện trường (chat/Zalo)

```text
tin nhắn ─► chuẩn hoá (S2) ─► nhận dạng thực thể (S3) ─► trích xuất theo luật của pack (khe: máy, lượng, đơn vị, thời điểm)
  ─► kiểm (bắt buộc, miền giá trị, đơn vị hợp lệ) ─► CỔNG MƠ HỒ
        ├─ chắc chắn  → observation + measurement + fact `reported_*` (span gốc làm bằng chứng)
        └─ mơ hồ/thiếu → câu hỏi lại cho người gửi (lưu `clarifications`), observation vẫn được giữ
```

Trích xuất bằng model chỉ là **phương án lùi có cổng** cho phần luật không bắt được, và kết quả của nó đi qua đúng các bước kiểm như trên.

### I4 — Luồng sự kiện nhiều provider (domain module thể thao)

```text
provider adapter (ánh xạ trường theo pack) ─► sự kiện chuẩn (kỳ, phút, bù giờ, loại, đội, cầu thủ)
  ─► nhận dạng thực thể theo bảng ánh xạ provider ─► khoá khử trùng ─► event store (chỉ ghi thêm, có phiên bản)
  ─► sửa muộn / rút lại = sự kiện mới trỏ về sự kiện cũ ─► projection (tỉ số, diễn biến, bảng xếp hạng) ─► API
```

---

## 6. Hợp đồng (contracts)

Kiểu dữ liệu dùng chung nằm ở `core/contracts`. Interface do **bên dùng** khai báo (thành ngữ Go), nên chúng nằm cạnh pipeline và context engine; bảng dưới là hình dạng của chúng.

| Hợp đồng | Chữ ký (rút gọn) | Ai cung cấp |
|---|---|---|
| `Normalizer` | `Normalize(ctx, rt, text) (NormalizedInput, error)` | `data/normalization` (luật từ pack) |
| `EntityResolver` | `Resolve(ctx, rt, in) (Resolution, error)` | `data/entityresolution` |
| `IntentClassifier` | `Classify(ctx, rt, in, res) (Intent, error)` | `intelligence/intent` |
| `Router` | `Route(ctx, rt, intent, principal) (RoutePlan, error)` | `intelligence/router` |
| `ContextSource` | `Fetch(ctx, rt, principal, step) ([]Evidence, error)` | facts, events, retrieval, calculation |
| `KnowledgeRetriever` | `Retrieve(ctx, rt, query) ([]Evidence, error)` | `intelligence/rag` — EAIS chỉ biết "tôi cần bằng chứng", không biết bên trong là BM25 hay vector |
| `StructuredDataSource` | `QueryMetric(ctx, rt, principal, q) ([]Evidence, []Conflict, error)` | `data/sourceoftruth` |
| `Tool` | `Spec() ToolSpec` · `Invoke(ctx, rt, principal, args) (ToolResult, error)` | `execution/tools` + pack |
| `ContextBuilder` | `Build(ctx, rt, plan, evidence) (ContextPackage, error)` | `intelligence/context` |
| `ModelGateway` | `Generate(ctx, rt, req) (Draft, error)` | `intelligence/models` |
| `Validator` | `Validate(ctx, rt, draft, pkg) []Violation` | `reliability/validation` |
| `WorkflowEngine` | `Start(ctx, rt, principal, def, subject, input) (Instance, error)` · `Transition(ctx, rt, principal, instanceID, transition, input) (Instance, error)` | `execution/workflows` — duy nhất được đổi trạng thái |
| `Guard` | `Check(ctx, rt, principal, instance, input) error` | mã cài sẵn, pack chọn theo tên |
| `TriggerEvaluator` | `Evaluate(ctx, rt, observation) ([]TriggerMatch, error)` | `execution/triggers` |
| `CommandHandler` | `Handle(ctx, rt, principal, cmd) (CommandResult, error)` | `core/pipeline` (lệnh → engine/tool) |

Kiểu trung tâm:

```go
// Evidence là đơn vị duy nhất đi vào context và là thứ duy nhất câu trả lời được phép dựa vào.
type Evidence struct {
    ID         string       // "E1", "E2"… ổn định trong một lượt
    Kind       EvidenceKind // fact | passage | event | calculation | policy | history
    Text       string       // dạng người đọc được, đưa vào prompt/mẫu câu
    Values     []Value      // giá trị có kiểu nằm trong bằng chứng (để kiểm số)
    Source     SourceRef    // nguồn, loại nguồn, tài liệu + phiên bản, URI
    Authority  int          // hạng thẩm quyền của nguồn (nhỏ = mạnh hơn)
    ObservedAt time.Time
    Confidence float64
}
```

---

## 7. Mô hình dữ liệu (PostgreSQL 17 + pgvector)

Mọi bảng nghiệp vụ có `tenant_id`, bật `ROW LEVEL SECURITY` + `FORCE`, chính sách đọc `current_setting('eais.tenant_id')`. Ứng dụng chạy bằng vai không có `BYPASSRLS`; khởi động **từ chối** chạy nếu vai có quyền vượt RLS.

| Nhóm | Bảng | Ghi chú |
|---|---|---|
| Danh tính | `tenants`, `principals`, `api_keys` | khoá lưu dạng băm; principal có vai trò + thuộc tính ABAC |
| Thực thể | `entities`, `entity_aliases` | khoá chuẩn theo loại; bí danh có nguồn (provider) và độ tin cậy |
| Sự thật | `sources`, `observations`, `facts`, `fact_conflicts`, `derived_facts` | observation giữ nguyên văn + nguồn; fact có chỉ số ngữ nghĩa · thực thể · kỳ · giá trị · đơn vị · thẩm quyền · observation gốc · trạng thái; derived fact ghi công thức + fact đầu vào |
| Quy trình | `workflow_instances`, `workflow_transitions`, `human_tasks`, `commands` | định nghĩa state machine ở pack; instance + lịch sử chuyển ở DB; lệnh có idempotency key |
| Tri thức | `documents`, `document_versions`, `chunks` | chunk thuộc đúng một phiên bản tài liệu; ghi model nhúng; ACL theo tài liệu |
| Sự kiện | `provider_events_raw`, `match_events`, `match_states` | thuộc domain module thể thao, migration riêng |
| Tương tác | `interactions`, `clarifications`, `review_queue` | mỗi lượt: route, bằng chứng, phán quyết, `pack_version`, chi phí |
| Quản trị | `audit_log`, `model_pricing` | audit chỉ ghi thêm, chuỗi băm |

---

## 8. Pack — nơi duy nhất chứa khác biệt giữa tenant

```text
tenants/<domain>/pack/
  pack.yaml          định danh, phiên bản, ngôn ngữ, model policy, ngân sách
  lexicon.yaml       thuật ngữ chuẩn, bí danh, viết tắt theo phạm vi, đơn vị, quy đổi, mẫu mã
  entities.yaml      loại thực thể + khoá + luật so khớp
  metrics.yaml       chỉ số có kiểu (fuel_liters…), đơn vị chuẩn, miền giá trị, ma trận thẩm quyền nguồn
  intents.yaml       ý định: từ khoá/mẫu, khe bắt buộc, route, ngưỡng
  tools.yaml         tool: schema vào/ra, quyền, timeout, cài đặt (truy vấn có tham số)
  skills.yaml        quy trình nhiều bước phục vụ câu hỏi/báo cáo
  workflows.yaml     state machine: trạng thái, phép chuyển, guard (TÊN), human task, hạn chót
  triggers.yaml      sự kiện + điều kiện so sánh khai báo → lệnh hoặc khởi động workflow
  templates.yaml     mẫu câu trả lời/hỏi lại/từ chối do tenant viết
  validation.yaml    validator bật/tắt + hành động khi vi phạm
```

Core đọc pack qua `core/tenant`, kiểm toàn bộ lúc khởi động (**cấu hình hỏng thì không chạy**), và đóng dấu `pack_version` (băm nội dung) vào mọi lượt.

Giới hạn của pack (P12) — để pack không thành "ngôn ngữ lập trình bằng YAML":

| Được | Không được |
|---|---|
| danh sách, bảng tra, ngưỡng, đơn vị, bí danh, mẫu câu | `if/when/unless/foreach` lồng nhau |
| so sánh một bước: `metric op value`, `state == X` | biểu thức tự do, gọi hàm tuỳ ý |
| chọn guard/validator/tool **theo tên** từ danh mục mã | định nghĩa hành vi mới trong YAML |

Khi nhu cầu vượt cột trái, đó là tín hiệu viết **domain module** có hợp đồng và test, không phải mở rộng YAML.

---

## 9. Capability có cổng

| Capability | Cổng mở khi |
|---|---|
| Phân loại ý định bằng model | luật của pack trượt ≥ một ngưỡng đo được trên bộ ca router |
| Trích xuất tin nhắn bằng model | luật trích xuất bỏ sót trên bộ ca bẩn, và có validator chặn kết quả sai |
| Rerank | Recall@20 > Recall@5 rõ rệt trên bộ ca truy xuất |
| Agent | có ca nghiệp vụ mà workflow cố định không giải được, đo được |
| MCP | có bên tiêu thụ thật cần khám phá tool theo chuẩn |
| Thị giác máy / fine-tune | có dữ liệu thật và ca hỏng mà context/tool không chữa được |

---

## 10. Stack

| Thành phần | Chọn | Ghi chú |
|---|---|---|
| Ngôn ngữ chính | Go 1.26.x | `go 1.26.0` + `toolchain` bản vá |
| Router HTTP | chi v5 | |
| DB | PostgreSQL 17 + pgvector, driver pgx v5 | ảnh Docker ghim phiên bản |
| Python | 3.12, dịch vụ tính toán sau ranh giới §11 | V1: chỉ `document-parser`; không truy cập DB, không chứa luật nghiệp vụ |
| Hàng đợi/cache | chưa dùng | Redis/Kafka chỉ thêm khi số đo đòi (ADR-0004) |
| Model | qua Model Gateway, nhà cung cấp theo cấu hình tenant | không hard code model/URL trong mã |

---

## 11. Ranh giới Go ↔ Python — "Go first, Python by evidence"

> **Go là ngôn ngữ của hệ thống và control plane. Python chỉ được tồn tại sau một ranh giới capability tường minh, khi hệ sinh thái ML/tài liệu/dữ liệu của Python mang lại giá trị đo được. Dịch vụ Python KHÔNG BAO GIỜ sở hữu: phân quyền tenant, giao dịch nghiệp vụ, trạng thái workflow, chính sách nguồn sự thật, hay điều phối lõi.**

Luật chọn ngôn ngữ cho một module mới:

```text
Module mới ── có sở hữu quyết định hệ thống/nghiệp vụ không? ──YES──► GO
                  │ NO
                  ▼
      phụ thuộc mạnh PyTorch / HF / OCR / layout / CV / khoa học dữ liệu? ──YES──► PYTHON (sau ranh giới)
                  │ NO
                  ▼
                 GO
```

| Thứ | Ngôn ngữ | Vì sao |
|---|---|---|
| API, tenant, auth, RBAC/ABAC, router, workflow, luật nghiệp vụ, validation, tool runtime, giao dịch, audit, trace, chi phí, SLA | Go | "ai được làm gì, số nào đúng, request đi đâu, hành động nào được xảy ra" phải có **một** chủ |
| nguồn sự thật, hoà giải theo luật, context builder, Model Gateway, structured output | Go | điều phối và quyết định; gọi model là gọi HTTP, không cần Python |
| RAG: vòng đời tài liệu, chunking, pgvector, tsvector, hybrid + RRF, lọc quyền, trích dẫn | Go | là hạ tầng tìm kiếm và điều phối, không phải ML |
| nhúng qua API nhà cung cấp | Go | không lập dịch vụ Python chỉ để gọi một API |
| xử lý sự kiện thể thao: thứ tự, khử trùng, sửa muộn, projection | Go | là bài toán hệ phân tán, không phải AI |
| đọc PDF/DOCX/XLSX phức tạp, OCR, nhận dạng bố cục, trích bảng nâng cao | Python | hệ sinh thái tài liệu |
| model nhúng chạy tại chỗ, cross-encoder rerank, so khớp thực thể bằng ML, classifier ý định bằng ML, thị giác máy | Python (**có cổng**, §9) | suy luận ML |
| huấn luyện, fine-tune, thí nghiệm dữ liệu | Python (ngoại tuyến, không phải dịch vụ chạy cùng request) | ML stack |

Hợp đồng bắt buộc cho mọi dịch vụ Python:

1. **Stateless, không truy cập DB, không giữ khoá tenant.** Go gửi dữ liệu cần xử lý; Python trả cấu trúc hoặc ứng viên kèm điểm số. Tenant id chỉ đi kèm để ghi log.
2. **Python đề xuất, Go quyết.** Python trả `DocumentStructure` hoặc `[{candidate, score}]`; ngưỡng, chính sách, chấp nhận hay đưa người duyệt là việc của Go.
3. **Hợp đồng có kiểu, một nguồn**: JSON Schema trong `schemas/`, cả test Go lẫn test Python kiểm cùng một bộ mẫu. Bắt đầu bằng HTTP + JSON; chỉ chuyển sang gRPC khi số đo đòi.
4. **Lỗi có nhãn**: Go bọc mọi lỗi gọi dịch vụ Python thành `infra:<service>` (sập, hết giờ) hoặc `third_party:<service>:bad_response` (trả sai schema).
5. **Tránh đường nóng**: đọc tài liệu chạy lúc nạp, không chạy trong một lượt hỏi. Dịch vụ nào phải nằm trên đường hỏi (rerank, classifier) phải có ngân sách độ trễ đo được trước khi bật.

Lộ trình dịch vụ Python — mỗi bước chỉ mở khi cổng §9 có bằng chứng:

| Giai đoạn | Dịch vụ |
|---|---|
| V1 | `document-parser` |
| V2 (nếu bằng chứng đòi) | `+ ml-inference` (rerank, nhúng tại chỗ, so khớp thực thể ML, classifier) · `+ vision` |
| V3 (nếu thật sự huấn luyện) | `+ training/` (ngoại tuyến) |

Đánh giá chia hai, một cổng phát hành:

```text
Go — eval hệ thống: router, chuẩn hoá, tool, SQL, quyền, cô lập tenant, workflow, validation, độ trễ, E2E
Python — eval ML: nhúng, rerank, classifier, thị giác, huấn luyện
                     └──────────────┬──────────────┘
                              cổng phát hành chung
```
