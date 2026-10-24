# Kiến trúc EAIS

> **EAIS (Enterprise AI Integration Reference Architecture) is a runnable reference architecture for turning domain-specific business data, knowledge, workflows, and tools into governed, testable AI capabilities: a control kernel, optional modules (QUERY, RAG), and examples / case studies, without coupling the kernel to a single industry.**

Tài liệu này mô tả EAIS theo **bảy mặt phẳng** (§2) và **các luồng L0–L10** (§4) của roadmap hiện hành (`docs/ROADMAP-DAY-DU.md` §3, §5; D-RM-01). Mỗi phần tách hai thứ:
- **HIỆN CÓ**: có mã chạy thật, kèm `tệp:dòng` hoặc tên test. Số dòng đo tại `bff56c4` (21/10/2026 tối);
- **ĐÍCH**: chưa có mã; ghi phase của roadmap sẽ làm.

Phần nào chỉ là đích thì ghi "ĐÍCH", không gọi là đang chạy. Quy ước tên gọi ở [CONVENTIONS.md](CONVENTIONS.md). Lý do của từng lựa chọn ở [adr/](adr/).

> **Phạm vi (D-RM-01).** Hai chế độ sở hữu:
> - **A. Tích hợp:** app của khách giữ sự thật, quyền, giao dịch; EAIS đọc và ghi qua API bằng token của người dùng.
> - **B. Tự lo:** EAIS thu, chuẩn hoá, đối soát, lưu trữ.
>
> Hiện chỉ **Phase 0b** (Fuel, QUERY, chế độ B) có mã. COMMAND / WORKFLOW / EVENT thuộc Giai đoạn II: chỉ làm phần giấy tới khi có D-RM-04 (`docs/ROADMAP-DAY-DU.md` §4.3, §6). Workflow engine, pipeline lệnh, pipeline sự kiện, human task ở dưới là thiết kế ĐÍCH của các phase đó.

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

Mười ba nguyên tắc ràng buộc mọi thiết kế bên dưới:

| # | Nguyên tắc | Hệ quả kỹ thuật |
|---|---|---|
| P1 | Model ≠ System | model chỉ gọi ở chặng `generate`, và chỉ khi route cần diễn đạt |
| P2 | Phần mềm tất định bao quanh lõi xác suất | chuẩn hoá, nhận dạng, định tuyến, dựng ngữ cảnh, kiểm đầu ra đều là code có test |
| P3 | Sự thật đến từ nguồn có thẩm quyền, không đến từ model | câu hỏi số liệu chính xác đi `SQL_QUERY`; model không bao giờ là nguồn của một con số |
| P4 | Garbage in → garbage out | dữ liệu được chuẩn hoá và kiểm **lúc nạp**, báo lỗi về chủ dữ liệu thay vì nuốt im |
| P5 | Mơ hồ thì hỏi lại hoặc từ chối, không đoán | trường bắt buộc thiếu/mơ hồ → `CLARIFICATION`; không đủ bằng chứng → từ chối có lý do |
| P6 | Nền tảng dùng chung ≠ tri thức dùng chung | khác biệt tenant nằm trong **pack** (dữ liệu cấu hình), core không có nhánh theo tên tenant |
| P7 | Độ phức tạp phải được bằng chứng đòi | capability vào core khi có ca hỏng tái lập được + test (ADR-0002) |
| P8 | Mọi câu trả lời truy được về nguồn | mỗi lượt ghi route, bằng chứng (nguồn + phiên bản), phán quyết kiểm, phiên bản pack (HIỆN CÓ: bản ghi quyết định trong bộ nhớ, bằng chứng ghi theo mã; nguồn + phiên bản và lưu bền: ĐÍCH) |
| P9 | Benchmark ≠ chất lượng thật | đánh giá theo từng thành phần trên bộ ca của chính tenant, có ca bẩn/mơ hồ/xung đột |
| P10 | Sai thì tìm tầng sai trước khi đổi model | trace tách từng chặng; lỗi mang nhãn `fault` đúng tầng |
| P11 | Model được **đề xuất** chuyển trạng thái; **workflow engine sở hữu** trạng thái | trạng thái, phép chuyển, quyền, bất biến, giao dịch thuộc phần mềm; model chỉ đọc, phân loại, trích xuất, đề xuất, giải thích |
| P12 | Pack mô tả chính sách và cấu hình; **mã cài hành vi** | YAML không có điều kiện lồng, vòng lặp, biểu thức tự do; cần luồng điều khiển thì viết domain module có hợp đồng và test |
| P13 | Điều khiển tất định ≠ mọi thành phần phải tất định | control plane được phép dùng một classifier xác suất **dưới ngưỡng tin cậy và có đường lùi**: luật trước → classifier (có cổng) → không chắc thì hỏi lại |

---

## 2. Bảy mặt phẳng

Bảy mặt phẳng cũng là bảy đường làm việc song song (`docs/ROADMAP-DAY-DU.md` §3). Mã P1–P7 ở bảng này là mặt phẳng, không phải nguyên tắc P1–P13 ở §1.

| Mặt phẳng | Gồm | HIỆN CÓ | ĐÍCH (phase) |
|---|---|---|---|
| **P1 Dữ liệu và sự thật** | thu, chuẩn hoá, thực thể, nghĩa, thời gian, thẩm quyền, đối soát, cách ly, đính chính | Fuel, chế độ B: đọc tệp cục bộ 5 định dạng (`data/ingestion/readers.go:38-52`); chuẩn hoá (`data/normalization/engine.go:42`); thực thể (`data/entityresolution/resolver.go:49`); 4 chỉ số gán theo loại nguồn (`tenants/construction/pack/sources.yaml`, `metrics.yaml`); đối soát theo thẩm quyền lúc hỏi (`data/sourceoftruth/query.go:64-197`); cách ly có lý do, chỉ trong RAM. Test: `TestIngestFiveFormatsToObservationsOrQuarantine`, `TestKhacNguonCungChiSoChonTheoThamQuyen`, `TestWinRatchet` (G1) | đính chính, kế hoạch, ba mốc thời gian, cách ly có người nhận, lưu quan sát, khử trùng: 0a, 3. Nguồn DB / API / kênh: 4a, 8 |
| **P2 Tri thức** | tài liệu, RAG, trích dẫn, ACL tài liệu | Markdown + `acl.json` → chỉ mục từ khoá + IDF trong bộ nhớ, mỗi dòng một đoạn (`intelligence/rag/rag.go:109-132`); ACL theo vai (`rag.go:134-143`, `platform/pgstore/pgstore.go:175`); trích dẫn tài liệu + dòng. Test: `TestRetrieveTenantVaACL`, `TestHoiQuyTrinh` | phiên bản tài liệu; PDF / OCR / bảng; ACL theo phạm vi thực thể; vector / hybrid / rerank (có cổng, §9): 6, 4b |
| **P3 Quyết định** | hiểu ý định, định tuyến, policy, hỏi lại | luật từ khoá của pack (`intelligence/intent/intent.go:221`); định tuyến + kiểm quyền (`intelligence/router/router.go:170-261`); hỏi lại không lộ ứng viên ngoài phạm vi. Test: `TestHoiDauCauDuong`, `TestAllowsIsFailClosed`, `TestHoiLaiKhongLoUngVienNgoaiPhamVi`. **S4 là sàn của hệ** (ADR-0010) | S4 theo bất biến: 0b (ADR-0010). Model hiểu ý định, có ngưỡng tin cậy và đường lùi: 6 |
| **P4 Hành động** | command, tool, workflow, event, chống trùng, đọc lại | **0 dòng mã thi hành.** Chỉ có khai báo: kiểu `EntryType` (`core/contracts/contracts.go:192-197`, không nơi nào dùng); 5/12 route (TOOL, WORKFLOW, AGENT, DIRECT_LLM, FULL_TEXT_SEARCH) rơi vào nhánh lỗi `core/pipeline/ask.go:187-190`; ca spec dạng dữ liệu chưa có runner (`tenants/construction/evals/cases.yaml`: 17 COMMAND, 10 WORKFLOW, 23 EVENT) | L3: 1, 4b · L4: 2 · L5: 3. Mã chờ D-RM-04 |
| **P5 Kiểm soát** | tenant, danh tính, quyền, kiểm đầu ra, audit, vết, chi phí | tenant lấy từ Principal (`core/pipeline/ask.go:89-95`); quyền đóng mặc định (`intelligence/router/router.go:127-139`); RLS ENABLE + FORCE (`db/migrations/0001_win.sql`, `0002_dinh_danh_eais.sql`); từ chối vai vượt RLS (`platform/pgstore/pgstore.go:23-45`); kiểm đầu ra (`reliability/validation`, `core/pipeline/claim.go:27`); vết từng chặng, chỉ trong bộ nhớ (`observability/traces`). Test: `TestDauVaoHongVaSaiTenant`, `TestRLSCanaryB`, `TestOpenTuChoiVaiVuotRLS`, `TestMigration0002DinhDanhEAIS` | xác thực, audit, lưu bền bản ghi quyết định, chi phí theo lượt: 5 (audit cho lệnh: 1) |
| **P6 Chứng minh** | bộ ca, bộ giữ lại, baseline, bộ chấm, đột biến, soát độc lập | bộ 117 ca + cổng G1–G7 (`evals/`, `tenants/construction/win/`); H1, H3 đã chạy, H4 đã khoá; baseline B1, B2+, B2, B2″ (`evals/baseline`, `tenants/construction/win/g7run/systems.go`); 35 danh mục đột biến (`scripts/dot-bien/`); soát độc lập nhiều lượt | bộ chấm v2 đóng băng (`tenants/construction/win/g7run/main.go:67`: `scorerFrozen = false`), người chấm mù, baseline mức app: theo từng phase |
| **P7 Vận hành** | HTTP API, CI có PG, triển khai, quản trị, SLA, sao lưu, scale | PG dev (`deploy/docker-compose.yml`, tmpfs); bộ chạy migration (`platform/pgstore/migrate.go:17`); đọc DSN / khoá từ biến môi trường (`platform/pgstore/env.go`, `intelligence/models/messages.go`); ánh xạ lỗi → mã HTTP (`core/contracts/fault.go:97`, chưa nơi nào gọi); `scripts/scale-smoke.sh` (cỡ dữ liệu, không phải tải); 5 công cụ đo `package main` dưới `tenants/construction/win/` | HTTP API, `cmd/eais`, CI có PG, kho bí mật, hạn mức, SLA, sao lưu: 5. Scale, HA / DR: 9 |

```text
        EAIS CORE  (mã dùng chung, không biết tenant nào)
            │
        TENANT RUNTIME  (nạp pack, cô lập dữ liệu, ngân sách, quyền)
            │
   ┌────────┴─────────┐
   ▼                  ▼
 construction       sports        ← pack: ontology · lexicon · schema · rules ·
 (abc-infra)      (xyz-sports)        intents · tools · skills · templates · evals
```

- **HIỆN CÓ:**
  - core (`core/`, `data/`, `intelligence/`, `reliability/`, `observability/`, `platform/`);
  - Tenant Runtime: nạp + kiểm pack lúc khởi tạo, đóng dấu `pack_version` (`core/tenant/pack.go`);
  - pack `construction` (7 tệp, §8).
- **CHƯA CÓ:**
  - tenant `sports` chỉ có tài liệu và bộ ca dạng dữ liệu (`tenants/sports/evals/cases.yaml`), chưa có pack chạy được;
  - "ngân sách" của Tenant Runtime là ĐÍCH.

### 2.1 Bốn loại tương tác: hỏi, lệnh, quy trình, sự kiện

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
     (HIỆN CÓ)          (ĐÍCH, Phase 1)    (ĐÍCH, Phase 3)
           │                  │                  │
           └──────────────────┼──────────────────┘
                              ▼
     CAPABILITY PLANE: normalize · resolve · facts · retrieval · tools · model · validation
                              │
                              ▼
     WORKFLOW ENGINE (sở hữu trạng thái) ──► audit · thông báo · việc cho người (human task)
                       (ĐÍCH, Phase 2)
```

| Lối vào | Kết quả mong muốn | Ai quyết | Model có mặt khi | Hiện trạng |
|---|---|---|---|---|
| QUERY | câu trả lời có nguồn | router + nguồn sự thật | cần diễn đạt tri thức tài liệu | HIỆN CÓ (L2) |
| COMMAND | trạng thái đổi đúng quy trình, có audit | workflow engine (quyền + bất biến) | trích tham số từ câu tự do — kết quả vẫn phải qua kiểm schema | ĐÍCH (L3, Phase 1, 4b) |
| WORKFLOW | quy trình nhiều bước, người trong vòng, chạy qua khởi động lại | workflow engine | hầu như không | ĐÍCH (L4, Phase 2) |
| EVENT | quy trình được khởi động / đẩy tiếp | luật kích hoạt (trigger) + workflow engine | hầu như không; chỉ khi giải thích bất thường cho người | ĐÍCH (L5, Phase 3) |

Kiểu `EntryType` (`EntryQuery`, `EntryCommand`, `EntryEvent`, `core/contracts/contracts.go:192-197`) đã khai nhưng chưa nơi nào dùng: luồng hỏi không mang trường lối vào.

### 2.2 Bốn loại vai trò — chỉ một loại nằm trong prompt

> Enterprise AI Engineering không chủ yếu là làm model thông minh hơn; nó là xây một hệ thống kiểm soát model **được thấy dữ liệu gì, hiểu ngữ cảnh nào, được quyết định gì, được hành động gì**, và chứng minh đầu ra/hành động của nó đáng tin.

| Loại | Ví dụ | Nằm ở đâu | Ai thực thi |
|---|---|---|---|
| 1. Vai nghiệp vụ (user role) | trưởng dự án, kế toán, kỹ sư hiện trường, giám đốc | `Principal.Roles` (ĐÍCH: bảng `principals`) | `intelligence/router` (Policy) — ĐÍCH: `governance/permissions` |
| 2. Phạm vi dữ liệu (data access) | dự án nào, tài liệu nào, chỉ số nào được xem | thuộc tính ABAC của principal (`entity_scope`) + ACL tài liệu | phạm vi thực thể ở `intelligence/router` (InScope); ACL tài liệu theo vai ở `intelligence/rag`, `platform/pgstore` — ĐÍCH: `governance/permissions`, ACL tài liệu theo phạm vi thực thể, `context` |
| 3. Quyền capability (system capability) | được tìm tài liệu, truy vấn số liệu, tạo báo cáo, duyệt, gọi tool nào | chính sách vai → capability trong pack (`policy.yaml`) | `intelligence/router` trước mọi truy vấn số liệu / tài liệu — ĐÍCH: `governance/permissions` trước MỌI tool / lệnh |
| 4. Vai của model (model role) | "bạn hỗ trợ lập báo cáo ngày công trường" | mẫu prompt trong pack | **chỉ** định hướng văn phong — KHÔNG BAO GIỜ dùng để phân quyền |

Hệ quả: **model không bao giờ thấy dữ liệu mà người hỏi không được xem.** Lọc quyền xảy ra trước khi bằng chứng vào `ContextPackage`, không phải dặn model "đừng tiết lộ".

EAIS gồm bốn loại thành phần, và phần không phải AI chiếm đa số là dấu hiệu kiến trúc trưởng thành. Bảng dưới phân loại thành phần của kiến trúc ĐÍCH; phần HIỆN CÓ xem bảng bảy mặt phẳng (§2):

| Loại | Thành phần |
|---|---|
| Kỹ thuật phần mềm | auth, tenant, RBAC/ABAC, DB, API, giao dịch, trạng thái, workflow, audit, retry, timeout, idempotency, quan sát |
| Kỹ thuật dữ liệu | đọc tài liệu, chuẩn hoá, schema chuẩn, nguồn sự thật, dòng dõi dữ liệu, phiên bản, index, tìm kiếm, dựng ngữ cảnh |
| Trí tuệ xác suất | hiểu ngôn ngữ mơ hồ, phân loại ngữ nghĩa, truy xuất ngữ nghĩa, so khớp thực thể mờ, trích xuất, suy luận, tóm tắt, giải thích, sinh văn bản, thị giác, rerank học được, lập kế hoạch (agent) |
| Điều khiển | router, ngưỡng tin cậy, cổng capability, validation, chính sách lùi |

### 2.3 Workflow là khái niệm hạng nhất

> **ĐÍCH (L4, Phase 2).** Chưa có mã: không máy trạng thái, không human task, không bảng `workflow_*`. Mã chờ D-RM-04.

Mọi thứ có vòng đời (phiếu đề nghị vật tư, sự cố bảo dưỡng, phiếu nghiệm thu, sửa dữ liệu trận đấu) là một **state machine (máy trạng thái)** khai báo trong pack và chạy bằng một engine dùng chung:

```text
Phiếu đề nghị vật tư:  DRAFT → SUBMITTED → REVIEWING → APPROVED → ORDERED → RECEIVED → RECONCILED → CLOSED
                                     └──────────► REJECTED
```

- Mỗi phép chuyển có **guard (điều kiện chặn)**: quyền của principal + bất biến nghiệp vụ. Guard là **tên** do pack chọn từ danh mục mã đã cài, không phải biểu thức trong YAML (P12).
- Bước chờ người (duyệt, kiểm tra hiện trường) là **human task** có hạn chót và người nhận.
- Model có thể gợi ý "nên chuyển sang APPROVED", nhưng chỉ engine thực hiện, sau khi guard qua (P11).
- Trước khi viết mã: so với đồ có sẵn (Temporal, n8n…) theo câu 9 của cổng QuânIT (`docs/ROADMAP-DAY-DU.md` L4); chỉ tự viết phần quyền và bằng chứng.

---

## 3. Cây thư mục

Một module Go: `github.com/luannt2002/enterprise-ai-control-layer` (link git giữ nguyên theo D-RM-02). Go là ngôn ngữ của hệ thống và control plane. Python chỉ được tồn tại **sau ranh giới capability tường minh**, khi hệ sinh thái ML / tài liệu / dữ liệu của Python mang lại giá trị đo được (§11). Hiện chưa có dịch vụ Python nào.

Dòng ghi **ĐÍCH** là thư mục chưa có (kiểm bằng `git ls-files`), chỉ có trong kiến trúc đích. Các dòng còn lại có mã thật.

```text
cmd/eais/                 ĐÍCH (mặt phẳng P7, Phase 5): một binary, nhiều lệnh con: serve · migrate · seed · simulate · eval
core/
  contracts/              kiểu dữ liệu dùng chung (Evidence, Span, RoutePlan, Verdict, Fault…), tenant trong ngữ cảnh
  decimal/                số thập phân chính xác cho giá trị nghiệp vụ
  tenant/                 Tenant Runtime: nạp + kiểm pack, phiên bản pack
  pipeline/               luồng hỏi ask (S0–S10) + cổng phán quyết câu xác nhận claim (ADR-0009)
                          ĐÍCH: command (C0–C7) · trigger (E0–E6)
data/
  ingestion/              đọc tệp cục bộ 5 định dạng theo sources.yaml; cách ly có lý do (trong RAM)
                          ĐÍCH: connector URL / DB / API / kênh
  parsing/                ĐÍCH: client Go gọi python/document-parser
  normalization/          bộ máy chuẩn hoá: số, lượng + đơn vị, tiền, ngày / giờ, mã, viết tắt
  entityresolution/       nhận dạng thực thể: bí danh, cấp cha (so khớp mờ: ĐÍCH)
  sourceoftruth/          quan sát → fact → fact dẫn xuất; ma trận thẩm quyền; hoà giải lúc hỏi
intelligence/
  intent/                 phân loại ý định theo luật của pack
  router/                 ý định → kế hoạch route; quyền: vai → capability, phạm vi thực thể (ngưỡng tin cậy: ĐÍCH)
  context/                ĐÍCH: Context Engine (gom bằng chứng, khử trùng, xếp, ngân sách)
  rag/                    chỉ mục từ khoá + IDF trong bộ nhớ, lọc tenant + ACL theo vai
                          ĐÍCH: chunking theo cấu trúc, embeddings, hybrid dense + sparse + RRF
  models/                 Model Gateway: model giả tất định + client kiểu Messages, cấu hình qua biến môi trường
execution/                ĐÍCH (mặt phẳng P4): tools · workflows · triggers · skills
reliability/
  validation/             kiểm đầu ra: số, mã, ngày, trích dẫn, chiều đo, ràng buộc claim
  clarification/          ĐÍCH (hỏi lại hiện nằm ở intelligence/router + core/pipeline)
governance/               ĐÍCH (mặt phẳng P5): auth · permissions · tenancy · audit
                          (quyền hiện ở intelligence/router; tenant ở core/contracts + platform/pgstore)
observability/
  traces/                 đo từng chặng, unaccounted_ms, bản ghi quyết định + phát lại (trong bộ nhớ)
  cost/                   ĐÍCH
platform/
  pgstore/                PostgreSQL: pool pgx, migration runner, Load, truy vấn qua RLS, đọc DSN từ biến môi trường
  httpapi/                ĐÍCH (mặt phẳng P7): /v1/ask · /v1/commands · /v1/events · /v1/workflows · /v1/tasks · /v1/ingest/* · /v1/health
domain/                   ĐÍCH: sportsevents (event store, khử trùng, sửa muộn, projection)
db/migrations/            0001_win.sql (4 bảng nghiệp vụ + RLS), 0002_dinh_danh_eais.sql (vai, GUC theo tên EAIS)
tenants/
  construction/           pack (7 tệp, §8) + bộ ca + công cụ đo WIN / G7 (win/: gen, winrun, winreport, g7run, kiemca, d5probe)
  sports/                 bộ ca dạng dữ liệu (evals/cases.yaml); chưa có pack chạy được
simulations/              dữ liệu giả lập cho test (fixtures)
evals/                    runner WIN + G7, bộ đếm độc lập, baseline RAG
scripts/                  bộ giữ lại (đóng băng, cam kết băm, chạy một lần), đột biến (dot-bien/), scale smoke, kiểm kê
schemas/, python/         ĐÍCH (ranh giới Go ↔ Python, §11): JSON Schema xuyên ngôn ngữ, python/document-parser
docs/                     tài liệu
deploy/                   docker compose: PostgreSQL 17 dev (tmpfs). ĐÍCH: pgvector, document-parser
```

Thư mục chỉ được tạo khi có mã thật (`CLAUDE.md` §25). `execution/agents` và `execution/mcp` chưa có cho tới khi cổng mở (§9).

---

## 4. Các luồng (L0–L10)

Khuôn mỗi luồng (như `docs/ROADMAP-DAY-DU.md` §5): **Đích** (sơ đồ) → **HIỆN CÓ** (bước có mã, kèm `tệp:dòng` / test) → **Thiếu** → **Phase**. L2 chạy trọn; L1 và L8 có cho Fuel (L1 còn lỗi im lặng); L0, L6, L7, L9 có một phần; L10 là quy trình người–agent; L3, L4, L5 là ĐÍCH.

| Luồng | Trạng thái | Phase |
|---|---|---|
| L0 Đưa AI vào một việc | một lát (Fuel), cổng QuânIT chưa áp tường minh | 0a, 4c |
| L1 Dữ liệu | HIỆN CÓ, còn lỗi im lặng | 0a, 3 (nguồn DB / API / kênh: 4a, 8) |
| L2 Hỏi (QUERY) | HIỆN CÓ | 0b, 6 (xác thực, HTTP, lưu bền: 5) |
| L3 Làm (COMMAND) | ĐÍCH (0 dòng) | 1, 4b — chờ D-RM-04 |
| L4 Quy trình (WORKFLOW) | ĐÍCH (0 dòng) | 2 — chờ D-RM-04 |
| L5 Sự kiện (EVENT) | ĐÍCH (0 dòng) | 3 — chờ D-RM-04 |
| L6 Tri thức (RAG) | một phần | 6, 4b |
| L7 Kiểm soát | một phần | 0b, 5 (audit cho lệnh: 1) |
| L8 Chứng minh | HIỆN CÓ cho Fuel | mọi phase |
| L9 Vận hành | gần như chưa có | 5, 9 |
| L10 Điều phối người–agent | đang chạy (quy trình, không mã) | — |

### L0. Đưa AI vào một việc của doanh nghiệp

```text
việc nghiệp vụ → [CỔNG QUÂNIT, 9 câu] → hợp đồng tối thiểu theo việc → chọn chế độ A / B → xây lát mỏng
→ [cổng chứng minh của phase] → việc "ra" → chủ dự án debug để hiểu → làm sâu
```

- **HIỆN CÓ:**
  - một lát mỏng: Fuel, chế độ B, L1 + L2 (`core/pipeline/ask.go:81` `Ask.Handle`, `tenants/construction/win/winrun/winrun.go:105` `Run`; test `TestWinRatchet`);
  - cổng chứng minh của phase: G1–G7, chỉ cho Fuel (`evals/win.go:133`, `tenants/construction/win/g7run`);
  - kênh debug: `docs/debug-handoff/` (1 handoff).
- **Thiếu:**
  - cổng QuânIT mới là văn bản (`docs/ROADMAP-DAY-DU.md` §4.1), chưa áp tường minh cho việc nào; chấm ngầm cho Fuel (bảng §7 của roadmap): đạt câu 2, 4, 5, 6; một phần câu 7; trượt câu 1, 3, 8, 9;
  - hợp đồng tối thiểu theo việc;
  - chọn chế độ A / B: mã chỉ chạy kiểu B;
  - chưa việc nào "ra": Fuel 0/7 cổng.
- **Phase:** 0a (lần đầu áp, phần giấy), 4c.

### L1. Dữ liệu (chế độ B; chế độ A thì đọc qua API của app)

```text
nguồn (tệp / DB / API / kênh) → đọc theo loại → chuẩn hoá biểu diễn ("4k" → 4.000 L) → thực thể ("máy 03" → EX-03)
→ nghĩa (cấp / tiêu thụ / báo / hoá đơn / KẾ HOẠCH / ĐÍNH CHÍNH) → thời gian (xảy ra · gửi · hiệu lực) → kiểm miền
→ cách ly, có người nhận → lưu quan sát + nguồn gốc → đối soát theo thẩm quyền → sự thật đã đối soát
```

| Bước | HIỆN CÓ | Test | Thiếu |
|---|---|---|---|
| nguồn | chỉ tệp cục bộ (`data/ingestion/ingest.go:120-150`) | `TestIngestFiveFormatsToObservationsOrQuarantine` | DB / API / webhook / kênh; chế độ A |
| đọc theo loại | 5 định dạng: csv, xlsx, json, jsonl, pdf_lines (`data/ingestion/readers.go:38-52`, `data/ingestion/sources.go:60`); cấu hình sai bị chặn lúc khởi động (`sources.go:63-130`) | `TestIngestFiveFormatsToObservationsOrQuarantine`, `TestLoadSourcesRejectsBrokenMaps` | dòng PDF không khớp mẫu bị bỏ im lặng, không vào cách ly (`readers.go:165-177`) |
| chuẩn hoá biểu diễn | `ingest.go:295-333` → `data/normalization/engine.go:42` | `TestNormalizeVectors` | "≈" mất cờ ước lượng (`engine.go:198-211`, suy từ mã); nhóm nghìn bằng dấu cách ngoài danh sách nhãn (D-OP-9) |
| thực thể | `ingest.go:258` → `data/entityresolution/resolver.go:49`; nguồn có cấu trúc tra mã theo nguồn (`ingest.go:202-208`) | `TestResolve` | so khớp mờ; mã theo nguồn trùng khoá bị ghi đè (`ingest.go:94`) |
| nghĩa | gán theo LOẠI NGUỒN: 4 chỉ số (`tenants/construction/pack/sources.yaml`, `metrics.yaml`) | `TestBonChiSoKhacNhauKhongDeNhau` | kế hoạch, đính chính: mọi tin chat thành `reported_fuel` |
| thời gian | một mốc ngày (`ingest.go:239-253`) | (trong test nạp) | giờ gửi, hiệu lực; `observed_at`, `reported_by` không được nạp (`data/ingestion/bridge.go:36-45`) |
| kiểm miền | `ingest.go:229-232` → `data/sourceoftruth/catalog.go:147-159`; chiều đơn vị (`ingest.go:323-331`) | ca R-021 … R-025 của test nạp | kiểm chéo trường, chuỗi thời gian |
| cách ly | có lý do (`ingest.go:34-36`); G1: mọi bản ghi hoặc thành quan sát hoặc bị cách ly có lý do | `TestWinRatchet`, `TestCheckG1BatDuocLech` | chỉ trong RAM; không người nhận, không hàng đợi |
| lưu quan sát + nguồn gốc | bảng `facts` có `source_type`, `source_ref`, `raw`; `Load` thay trọn dữ liệu của tenant trong một giao dịch (`platform/pgstore/load.go:21-48`) | `TestLoadThayDuLieuMotTenant` (PG) | bảng quan sát, lịch sử, nạp thêm (không thay trọn) |
| đối soát theo thẩm quyền | lúc HỎI: xếp thẩm quyền, dung sai, ngang hạng thì chuyển người duyệt (`data/sourceoftruth/query.go:64-197`) | `TestKhacNguonCungChiSoChonTheoThamQuyen`, `TestNgangThamQuyenThiKhongTuChon` | lệch trong dung sai không được nhắc; cộng mọi fact cùng nguồn, cùng ngày (`query.go:199-221`, đúng thiết kế theo `TestCungNguonCungNgayLaCongDon`) nên tin gửi lại có thể bị cộng đôi — chưa tái lập đầu-cuối |
| sự thật đã đối soát | tính lại mỗi lượt hỏi (`query.go:50-61`) | `TestTrichDanVaXungDotGiuNguonGoc` | lưu bền; khử trùng bản ghi gửi lại |

- Lỗi im lặng khác mà roadmap ghi theo số đo của `docs/ROADMAP.md` §0.4 (chưa đo lại ở đợt này): tin gửi lại thành `SYSTEM_FAILURE`; "máy 03" thành 3 L ở 6 ca H1 (đường nạp đã chặn bằng luật loại trừ ở `data/ingestion/ingest.go:266-269`, nên nhiều khả năng lỗi nằm ở luồng hỏi, bộ chuẩn hoá dùng chung).
- **Phase:** 0a, 3 (sửa tận gốc ở L5). Nguồn DB / API / kênh: 4a, 8.
- Bốn tầng của dữ liệu sự thật và các đường nạp đích: §5.

### L2. Hỏi (QUERY), S0–S10

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

| Chặng | Việc | Tất định? | HIỆN CÓ (`tệp:dòng`; test) | ĐÍCH còn thiếu |
|---|---|---|---|---|
| S0 accept | kiểm đầu vào: text rỗng / quá 2.000 ký tự / ký tự NUL / thiếu `conversation_id`, `message_id` → `FaultInput` | có | `core/pipeline/ask.go:85-88`, `:1058-1070`; `TestDauVaoHongVaSaiTenant` (nhánh text rỗng) | xác thực khoá API → principal, idempotency theo `(conversation_id, message_id)`, nhận qua HTTP (Phase 5) |
| S1 tenant | tenant lấy từ Principal, so với Runtime đã gắn, gắn vào ngữ cảnh | có | `ask.go:89-95`, `core/contracts/tenantctx.go:10-19`; `TestDauVaoHongVaSaiTenant`, `TestRequestTypesHaveNoTenantField` | tra Tenant Runtime theo từng yêu cầu (hiện một `Ask` gắn đúng một Runtime) |
| S2 normalize | chuẩn hoá văn bản + span có kiểu theo lexicon của pack | có | `ask.go:96-99` → `data/normalization/engine.go:42`; `TestDoiCachVietKhongDoiNghia` | — |
| S3 resolve | span / cụm từ → thực thể chuẩn | có | `ask.go:100-106` → `data/entityresolution/resolver.go:49`; `TestResolve`, `TestHoiLaiKhongLoUngVienNgoaiPhamVi` | so khớp mờ |
| S4 intent | luật của pack chấm ý định + khe | có | `ask.go:107-110` → `intelligence/intent/intent.go:221`; `TestHoiDauCauDuong`. `Confidence` luôn bằng 1 (`intent.go:412`), không ai đọc | S4 theo bất biến (ADR-0010, Phase 0b); model hiểu ý định có cổng, ngưỡng tin cậy, đường lùi (Phase 6) |
| S5 route | ý định + khe + quyền → `RoutePlan`; thiếu / mơ hồ thì hỏi lại, không gọi model | có | `ask.go:111-147` → `intelligence/router/router.go:170-261` (capability `:198-216`, phạm vi thực thể `:217-223`, hỏi lại `:173-184`, `:224-234`); `TestAllowsIsFailClosed`, `TestRouteHoiLaiNoiDungLyDo` | — |
| S6 execute | truy vấn fact / phép tính / so sánh | có | `ask.go:322-334` → `data/sourceoftruth/query.go:64` (Query), `:251` (Derive), `:264` (Compare); nguồn fact bộ nhớ hoặc PG (`platform/pgstore/pgstore.go:87`); `TestHoiDauCauDuong`, `TestSoSanhHaiNghia` | chạy `plan.Steps` (router dựng ở `router.go:247-259`, chưa ai chạy: thực thi rẽ nhánh theo `plan.Route`, `ask.go:155-191`); 5/12 route (TOOL, WORKFLOW, AGENT, DIRECT_LLM, FULL_TEXT_SEARCH) trả lỗi ở `ask.go:187-190`, không test |
| S7 context | gói bằng chứng + xung đột; tri thức: 3 đoạn đầu + giá trị có kiểu | có | `ask.go:335-340`, `:38`, `:514-517`; ACL tài liệu theo vai (L6); `TestTrichDanVaXungDotGiuNguonGoc` | Context Engine: khử trùng, xếp, ngân sách token, độ tươi |
| S8 generate | route số liệu: **mẫu câu của tenant, 0 lời gọi model**; RAG / HYBRID: model nhận gói bằng chứng | không (RAG / HYBRID) | `ask.go:451-456` + `tenants/construction/pack/templates.yaml`; `ask.go:518-524` (model); `TestHoiDauCauDuong` (không gắn model mà mọi ca số liệu vẫn đạt), `TestHoiQuyTrinh` (đếm lời gọi: 0 khi từ chối, 1 khi RAG) | model thật chỉ được nối trong harness đo G7 |
| S9 validate | số / mã / ngày / claim / chiều đo phải có trong bằng chứng; phán quyết câu xác nhận; hỏng thì không giao | có | `ask.go:458-468`, `:525-535` → `reliability/validation` (`numeric.go:24`, `binding.go:45`, `knowledge.go:45`, `:306`, `:333`); `core/pipeline/claim.go:27` (ADR-0009); `TestMauCauChenSoLaBiChan`, `TestClaimKhongGanBangChungBiChan`, `TestVerdictGap` | bắt lỗi chọn sai nghĩa chỉ số ở S4 |
| S10 respond | trả kết quả có kiểu + bản ghi quyết định; phát lại các chặng tất định | có | `ask.go:192-195`, `:224-279`, `:292-305` (Replay); `observability/traces/decision.go`; `TestOutcomeCoKieu`, `TestPhatLaiTuBanGhiQuyetDinh` | lưu bền `interactions`, trả qua HTTP, một dòng log mỗi lượt (Phase 5) |

Ba luật cứng của luồng (giữ bằng cấu trúc rẽ nhánh `ask.go:155-191`):

1. **Route không cần diễn đạt thì không gọi model.** `SQL_QUERY`, `CALCULATION`, `CLARIFICATION`, `REJECT`, `HUMAN_ESCALATION` trả lời bằng mẫu câu (template) do tenant viết. Chữ gửi khách đến từ mẫu câu của tenant; ngoại lệ hiện có: ghi chú giải thích do mã ghép vào câu hỏi lại (`core/pipeline/ask.go:139`, `:146`, `:1236`, `:1239`).
2. **Model không bao giờ là nguồn của con số.** Mọi con số trong câu trả lời phải khớp (sau chuẩn hoá) một con số trong gói bằng chứng, hoặc là kết quả của bước `CALCULATION` có ghi phép tính.
3. **Kiểm thất bại thì không giao câu đó.** Hành động lùi: `fallback` (mẫu câu "chưa đủ dữ liệu chính xác"), `clarify`, hoặc `escalate`. HIỆN CÓ: mã chọn hành động theo nhánh (`core/pipeline/ask.go:166`, `:359`, `:465`), pack chỉ viết mẫu câu (`templates.yaml`). Pack quy định hành động lùi: ĐÍCH (`validation.yaml`, §8).

Router theo P13. HIỆN CÓ: chỉ nhánh luật. Classifier có cổng và ngưỡng tin cậy là ĐÍCH (Phase 6):

```text
ý định từ luật ── đủ tin cậy? ──yes──► route
                     │ no
                     ▼
          classifier (CHỈ khi cổng đã mở) ── đủ tin cậy? ──yes──► route
                     │ no / cổng đóng
                     ▼
               CLARIFICATION (hỏi lại kèm ứng viên)
```

- **Thiếu (đã đo hoặc đọc thấy):**
  - S4 là tầng sai đầu tiên của 13/15 ca đỏ H1 và 20/25 ca đỏ H3 (ADR-0010; số tài liệu ghi);
  - S9 không bắt lỗi chọn sai nghĩa: "Máy 03 hôm qua **hết** bao nhiêu dầu?" đi chỉ số kho xuất vì `tenants/construction/pack/intents.yaml:101-113` ("hết bao nhiêu" → `issued_fuel`);
  - mẫu câu hứa "đã chuyển người phụ trách" mà không có việc chuyển nào (`templates.yaml:16-17`);
  - HYBRID: phần tri thức bị chặn thì câu trả lời ghi sai lý do mà `Verdict.Pass` vẫn true (`ask.go:482-486`).
- **Phase:** 0b (S4 theo bất biến, ADR-0010), 6 (model hiểu ý định), 5 (xác thực, HTTP, lưu bền).

### L3. Làm (COMMAND) — ĐÍCH

```text
yêu cầu / đề xuất → kiểm schema → tenant → quyền (chế độ A: app quyết, bằng token người dùng) → policy → luật nghiệp vụ
→ khoá chống trùng → thực thi → ĐỌC LẠI xác minh → audit → báo kết quả: ĐÃ LÀM / CHƯA BIẾT / THẤT BẠI
```

#### 4.1 Pipeline lệnh (C0–C7) — ĐÍCH

```text
C0 accept (xác thực, idempotency key) ─► C1 tenant ─► C2 hiểu lệnh (chuẩn hoá + nhận dạng thực thể;
model chỉ trích tham số khi cổng mở) ─► C3 kiểm schema tham số ─► C4 quyền (RBAC/ABAC)
─► C5 workflow engine: guard + chuyển trạng thái trong MỘT giao dịch ─► C6 tool có tác dụng phụ
(idempotent, timeout, kết quả được kiểm) ─► C7 audit + thông báo + trả về trạng thái mới
```

- Lệnh thiếu tham số bắt buộc → trả về câu hỏi lại, **không** đoán.
- Lệnh có tác dụng phụ lớn (theo pack) → tạo human task chờ duyệt thay vì thực thi ngay.
- Gửi lại cùng idempotency key → trả kết quả cũ, không thực thi lần hai.

- **HIỆN CÓ:** 0 dòng mã thi hành. Chỉ có khai báo `EntryCommand` (`core/contracts/contracts.go:196`) và route `TOOL` (`contracts.go:210`, trả lỗi ở `core/pipeline/ask.go:187-190`); 17 ca COMMAND dạng dữ liệu (`tenants/construction/evals/cases.yaml`) mới được kiểm hình dạng (`TestBoCaTenantDungHopDong`). Dùng lại được: tenant từ danh tính, quyền đóng mặc định (đang phục vụ luồng hỏi).
- **Phase:** 1 (tạo một phiếu kiểm tra chênh lệch, chế độ B), 4b (tạo task trên CRM, chế độ A). Mã chờ D-RM-04.

### L4. Quy trình (WORKFLOW) — ĐÍCH

```text
phát hiện chênh lệch → mở hồ sơ → điều tra → tổ trưởng duyệt → kho xác minh → đối soát → duyệt / từ chối → đóng
(máy trạng thái · người trong vòng · hạn chờ · phục hồi sau khởi động lại · trạng thái dài hạn)
```

- **HIỆN CÓ:**
  - phát hiện chênh lệch chỉ khi có người hỏi, và chỉ hiện trong câu trả lời (route `CALCULATION`, `intelligence/router/router.go:239`);
  - đối soát theo thẩm quyền (L1).
- **Thiếu:**
  - hồ sơ, máy trạng thái, người trong vòng, hạn chờ, phục hồi;
  - mẫu câu hứa chuyển người, nhưng không có hàng đợi.
- Thiết kế đích: §2.3.
- **Phase:** 2. Trước khi viết mã, so với đồ có sẵn (Temporal, n8n…) theo câu 9 của cổng QuânIT; chỉ tự viết phần quyền và bằng chứng. Mã chờ D-RM-04.

### L5. Sự kiện (EVENT) — ĐÍCH

```text
sự kiện (tin mới · phiếu mới · vượt ngưỡng · đính chính · sự kiện đến muộn) → nạp sự kiện → khử trùng → thứ tự → luật
→ kích hoạt COMMAND / WORKFLOW → outbox → idempotency
```

#### 4.2 Pipeline sự kiện (E0–E6) — ĐÍCH

```text
E0 nhận sự kiện (tin nhắn, bản ghi, cảm biến, provider) ─► E1 chuẩn hoá + nhận dạng thực thể
─► E2 ghi quan sát (observation) có nguồn ─► E3 luật kích hoạt của pack: so sánh khai báo
(metric op ngưỡng, trạng thái = X) ─► E4 khởi động/đẩy workflow qua engine ─► E5 hành động
(tạo sự cố, gán người, thông báo) ─► E6 audit
```

- Luật kích hoạt chỉ gồm so sánh khai báo; logic phức tạp hơn là domain module (P12).
- Mỗi sự kiện xử lý **đúng một lần theo khoá khử trùng**; đến lại thì bỏ qua và ghi nhận.

- **HIỆN CÓ:** 0. `EntryEvent` khai ở `core/contracts/contracts.go:197`, không nơi nào dùng; đường nạp hiện có là nạp lô một lần (`platform/pgstore/load.go:21`); 23 ca EVENT dạng dữ liệu chưa có runner.
- **Phase:** 3. Đây là chỗ **sửa tận gốc** các lỗi đính chính / kế hoạch / sự kiện muộn của L1. Mã chờ D-RM-04. Đường nạp sự kiện nhiều provider (thể thao): §5, I4.

### L6. Tri thức (RAG)

```text
tài liệu → đọc / OCR → chia đoạn → chỉ mục → truy xuất (lọc ACL) → (rerank) → ngữ cảnh → model → trích dẫn → kiểm
```

| Bước | HIỆN CÓ | Test | Thiếu |
|---|---|---|---|
| tài liệu | Markdown + `acl.json` (`intelligence/rag/rag.go:197-219`); bảng `documents` | `TestWinRatchet` (nạp gián tiếp); bản / trạng thái / hiệu lực / phạm vi trên PG: `TestCaTriThucQuaPG`, `TestPhamViQuaPG` (nhánh RAG U6a, migration 0005) | loại tài liệu |
| đọc / OCR | chỉ `.md` | — | PDF / OCR / bảng cho tài liệu tri thức |
| chia đoạn | mỗi dòng không rỗng một đoạn (`rag.go:117-126`) | `TestRetrieveTenantVaACL`, `TestRetrieveTatDinh` | chia theo cấu trúc / bảng |
| chỉ mục | từ khoá + IDF trong bộ nhớ (`rag.go:109-132`); đường PG đọc mọi tài liệu được phép rồi dựng lại chỉ mục mỗi lượt (`platform/pgstore/pgstore.go:164-194`) | `TestRetrieveLuatDauVaCumNghiVan` | embedding, vector, tsvector, hybrid + RRF |
| truy xuất | lọc tenant + ACL theo vai (`rag.go:147-195`, `allowed` ở `rag.go:134-143`; SQL `WHERE acl && $1::text[]` ở `pgstore.go:175`); capability `documents.read` kiểm ở router | `TestRetrieveACLRongKhongAiDoc`, `TestHoiQuyTrinh`, `TestRetrieveACL` (PG) | tài liệu khai `scope` thì lọc theo phạm vi (U6a, ACL và phạm vi cùng một vai); SOP của seed chưa khai phạm vi nên `TestHoiQuyTrinh` vẫn khoá hành vi kỹ sư phạm vi PRJ-B đọc SOP |
| (rerank) | — | — | ĐÍCH, có cổng (§9) |
| ngữ cảnh | 3 đoạn đầu + giá trị có kiểu (`core/pipeline/ask.go:38`, `:514-517`) | `TestHoiQuyTrinh` | khử trùng, ngân sách token, xung đột giữa đoạn |
| model | 1 lời gọi (`ask.go:519-521`); mọi test dùng model giả tất định | `TestHoiQuyTrinh`, `TestHoiGhepSoLieuVaQuyTrinh` | — |
| trích dẫn | tài liệu + dòng (`ask.go:608-624`, `rag.go:186-193`) | `TestClaimKhongGanBangChungBiChan`, `TestBound` | phiên bản, trang |
| kiểm | `Bound`, `Fidelity`, `ForbidDimensions`, `ForbidUnitWords` (`ask.go:526-535`) | `TestChenLenhTrongTaiLieuKhongDoiDuocGi`, `TestForbidDimensions` | bộ đo truy xuất (Recall@K / MRR / NDCG) |

- **Phase:** 6 (khi lỗi đo được đòi), 4b.

### L7. Kiểm soát

| Mục | Trạng thái | HIỆN CÓ (`tệp:dòng`; test) |
|---|---|---|
| tenant lấy từ danh tính | có | `core/pipeline/ask.go:89-95`, `core/contracts/tenantctx.go:10-19`, `platform/pgstore/pgstore.go:50-70`; `TestDauVaoHongVaSaiTenant`, `TestRequestTypesHaveNoTenantField`, `TestGUCLayTuNguCanh` (PG) |
| xác thực | **chưa có** | Principal do bên gọi dựng (test, harness đo); không có HTTP, không có khoá API của người dùng |
| quyền đóng mặc định | có | `intelligence/router/router.go:127-139` (Allows), `:143-161` (InScope); nạp pack chặn ý định chạm dữ liệu mà thiếu capability (`intelligence/intent/intent.go:101-104`); `TestAllowsIsFailClosed`, `TestDataRouteRequiresCapability`, `TestLoadChanPackHong` |
| RLS ENABLE + FORCE | có | `db/migrations/0001_win.sql`, `0002_dinh_danh_eais.sql` (4 bảng nghiệp vụ); `TestRLSCanaryB`, `TestMigration0002DinhDanhEAIS` (khẳng định cờ ENABLE + FORCE và đúng tập quyền) |
| từ chối vai vượt RLS | có | `platform/pgstore/pgstore.go:23-45`; `TestOpenTuChoiVaiVuotRLS`. Chưa có tiến trình dịch vụ nào gọi lúc khởi động |
| kiểm đầu ra | có | xem L2 S9 |
| vết | một phần | đo từng chặng, chỉ trong bộ nhớ (`observability/traces/trace.go:40-45`, `:85-94`); `TestMeasureGhiChangVaKhongNuotLoi` |
| audit | **chưa có** | không mã, không bảng |
| lưu bền bản ghi quyết định | **chưa có** | `Decision` chỉ trả trong `Outcome` (`ask.go:224-279`); harness G7 đổ ra tệp đo, không phải kho của luồng chạy |
| chi phí theo lượt | **chưa có** | gateway điền `Draft.Usage` (`intelligence/models/messages.go:189-199`) nhưng pipeline bỏ rơi |

- **Phase:** 0b (phần đã có), 5 (xác thực, lưu bền, chi phí, audit); audit cho lệnh: 1.

### L8. Chứng minh

```text
bộ ca phát triển → bộ giữ lại (soạn độc lập, khoá băm, chạy MỘT lần) → baseline (B1, B2+, B2″; THÊM "app + model + kiểm cơ bản"
và "đồ có sẵn") → bộ chấm (hiệu chỉnh bằng người chấm mù) → cổng nhị phân → đột biến → soát độc lập
```

- **HIỆN CÓ:**
  - bộ ca phát triển + cổng G1–G7 (`evals/win.go`, `evals/runner.go:45`, `tenants/construction/win/`; `TestWinRatchet`, `TestWinGatesBlockBadSuites`);
  - bộ giữ lại:
    - công cụ: chia và đóng băng (`scripts/dong-bang-giu-lai.py`), cam kết băm (`scripts/cam-ket-giu-lai.py`), chạy một lần (`scripts/chay-giu-lai.sh`; đột biến `n27`), kiểm định dạng không lộ nội dung (`tenants/construction/win/kiemca`);
    - kết quả: H1 89/104, H3 109/134 đã chạy; H4 đã khoá (119 ca);
  - baseline B1, B2+, B2, B2″, B2-oracle (`evals/baseline`, `tenants/construction/win/g7run/systems.go`; `TestBaselineFailsCasesThatNeedControlLayers`);
  - bộ chấm nội dung v1 + bộ đếm độc lập (`evals/g7.go`; `TestG7ScorerAttacks`);
  - cổng nhị phân có ca âm (`g7run` verdict, `winreport`);
  - 35 danh mục đột biến (`scripts/mutate.sh`, `scripts/dot-bien/`);
  - soát độc lập nhiều lượt.
- **Thiếu:**
  - bộ chấm v2 chưa đóng băng (`tenants/construction/win/g7run/main.go:67`: `scorerFrozen = false`);
  - người chấm mù;
  - baseline mức app và "đồ có sẵn";
  - CI tự chạy cổng (đột biến và cổng hiện chạy tay).
- **Phase:** áp cho mọi phase. Phase 0b: H4 một lần, rồi W-G7 trên H5.

### L9. Vận hành

HTTP API · **CI có Postgres** · triển khai · bí mật · hạn mức · SLA · sao lưu · scale.

- **HIỆN CÓ:**
  - PG dev (`deploy/docker-compose.yml`: PostgreSQL 17 ghim digest, chỉ `127.0.0.1`, tmpfs);
  - bộ chạy migration (`platform/pgstore/migrate.go:17`);
  - bí mật chỉ qua biến môi trường (`platform/pgstore/env.go`, `intelligence/models/messages.go`; `TestBangChungKhongGhiBiMatTrongURL`);
  - ngân sách của MỘT phép đo G7 (`tenants/construction/win/g7run/main.go:75`);
  - `scripts/scale-smoke.sh` (cỡ dữ liệu, không phải tải đồng thời).
- **Thiếu:**
  - HTTP API (`core/contracts/fault.go:97` ánh xạ lỗi → mã HTTP, chưa nơi nào gọi);
  - `cmd/`;
  - CI: không có `.github`, Makefile hay hook; test PG bị SKIP khi thiếu DSN;
  - kho bí mật, hạn mức theo tenant / người dùng, SLA, sao lưu, quản trị.
- **Phase:** 5, 9.

### L10. Điều phối người–agent

```text
chủ dự án quyết ở docs/CHO-QUYET-DINH.md
→ BUILD xây theo docs/PROMPT-BUILD-EAIS.md, commit + push, tệp tiến độ sống docs/bao-cao/TIEN-DO-BUILD.md
→ phiên điều phối / debug đọc qua git, gửi lỗi qua docs/debug-handoff/
```

- **HIỆN CÓ:** đang chạy. Là quy trình người–agent, không có mã. Bằng chứng là lịch sử git và các tệp trên.

---

## 5. Dữ liệu sự thật: bốn tầng và các đường nạp (I1–I4)

Phần này là chi tiết của L1, L5, L6 (§4). Trạng thái HIỆN CÓ / ĐÍCH ghi ở đầu mỗi đường.

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

> **HIỆN CÓ:** OBSERVATION → FACT. `Fact` nhúng quan sát gốc và đã là một lần đo có kiểu (`data/sourceoftruth/query.go:26-34`), không có kiểu MEASUREMENT riêng. DERIVED FACT tính lúc hỏi (`query.go:251-317`), không lưu.

Hai luật rút ra (đang giữ: `TestBonChiSoKhacNhauKhongDeNhau`, `TestThamQuyenKhongPhaiTenHayDoMoi`):

1. **Ma trận thẩm quyền chỉ áp trong cùng một chỉ số ngữ nghĩa.** "Kho xuất 3.850" không *thắng* "đội trưởng báo 4.000" — chúng đo hai thứ khác nhau. Chênh lệch giữa chúng là một **derived fact** đáng điều tra, không phải xung đột cần chọn bên.
2. **Authority ≠ recency ≠ similarity ≠ số lần lặp.** Nguồn mới hơn, giống câu hỏi hơn hay được nhắc nhiều hơn đều không tự động đúng hơn.

### I1 — Tài liệu (SOP, hợp đồng, sổ tay thiết bị, quy định)

> **ĐÍCH.** HIỆN CÓ: `rag.LoadMarkdown` (`intelligence/rag/rag.go:197-219`) → `Store.Load` thay trọn tài liệu của tenant trong một giao dịch (`platform/pgstore/load.go:21-48`). Chưa có parsing service, phiên bản, embedding hay tsvector.

```text
connector (URL/tải lên) ─► parsing service (Python, stateless) ─► cấu trúc: heading · đoạn · bảng(header, hàng, ô) · trang
  ─► kiểm chất lượng đọc (bảng còn đủ ô? số còn đủ?) ─► chunking theo cấu trúc (bảng theo nhóm hàng, lặp header)
  ─► metadata (loại tài liệu, phiên bản, hiệu lực, ACL) ─► embedding (ghi model + số chiều + phiên bản)
  ─► index (tsvector 'simple' + vector) ─► công bố phiên bản mới NGUYÊN TỬ (một giao dịch)
```

Bảng số liệu trong tài liệu (ví dụ bảng định mức) còn được trích thành **bản ghi có kiểu** vào kho sự thật, để phép tính dùng số chứ không dùng chữ.

### I2 — Bản ghi có cấu trúc (ERP, phiếu kho, bảng tính)

> **HIỆN CÓ một phần:** tệp → quan sát → fact theo chỉ số của loại nguồn (L1). Hoà giải xảy ra lúc HỎI (`data/sourceoftruth/query.go:64-197`), không lúc nạp. Không có hàng đợi duyệt: chỉ trả `HUMAN_REVIEW_REQUIRED` trong câu trả lời (`core/pipeline/ask.go:346-361`).

```text
connector ─► observation (nguyên văn + nguồn) ─► ánh xạ về schema chuẩn của pack ─► chuẩn hoá giá trị
  ─► kiểm schema + luật nghiệp vụ (miền giá trị, chéo trường) ─► measurement ─► fact theo chỉ số ngữ nghĩa của loại nguồn
  ─► hoà giải với fact CÙNG chỉ số/thực thể/kỳ ─► xung đột vượt ngưỡng → hàng đợi duyệt ─► derived fact (chênh lệch, tổng)
```

### I3 — Tin nhắn hiện trường (chat/Zalo)

> **HIỆN CÓ một phần:** chuẩn hoá + thực thể + trích theo luật của pack (`data/ingestion/ingest.go:256-291`). Mơ hồ / thiếu thì CÁCH LY có lý do, chỉ trong RAM (`ingest.go:262-264`, `:283`, `:287`): chưa hỏi lại người gửi, chưa lưu `clarifications`.

```text
tin nhắn ─► chuẩn hoá (S2) ─► nhận dạng thực thể (S3) ─► trích xuất theo luật của pack (khe: máy, lượng, đơn vị, thời điểm)
  ─► kiểm (bắt buộc, miền giá trị, đơn vị hợp lệ) ─► CỔNG MƠ HỒ
        ├─ chắc chắn  → observation + measurement + fact `reported_*` (span gốc làm bằng chứng)
        └─ mơ hồ/thiếu → câu hỏi lại cho người gửi (lưu `clarifications`), observation vẫn được giữ
```

Trích xuất bằng model chỉ là **phương án lùi có cổng** cho phần luật không bắt được, và kết quả của nó đi qua đúng các bước kiểm như trên.

### I4 — Luồng sự kiện nhiều provider (domain module thể thao)

> **ĐÍCH**, chưa có mã.

```text
provider adapter (ánh xạ trường theo pack) ─► sự kiện chuẩn (kỳ, phút, bù giờ, loại, đội, cầu thủ)
  ─► nhận dạng thực thể theo bảng ánh xạ provider ─► khoá khử trùng ─► event store (chỉ ghi thêm, có phiên bản)
  ─► sửa muộn / rút lại = sự kiện mới trỏ về sự kiện cũ ─► projection (tỉ số, diễn biến, bảng xếp hạng) ─► API
```

---

## 6. Hợp đồng (contracts)

Kiểu dữ liệu dùng chung nằm ở `core/contracts`. Interface do **bên dùng** khai báo (thành ngữ Go), nên chúng nằm cạnh pipeline và context engine; bảng dưới là hình dạng của chúng.

> **Trạng thái.** Bảng dưới là hợp đồng ĐÍCH. HIỆN CÓ bảy interface Go (không tính test). Hai cái khớp bảng: `KnowledgeRetriever` với `Retrieve(ctx, pr, query, k)` (`core/pipeline/ask.go:43-45`) và `models.Gateway` với `Generate(ctx, Request)` (`intelligence/models/models.go:20-22`). Các điểm tiếp hợp khác: `sourceoftruth.Store` (nguồn fact cho hợp đồng `StructuredDataSource`, `data/sourceoftruth/query.go:37`; bản PG ở `platform/pgstore/pgstore.go:87`), `entityresolution.Directory` (`data/entityresolution/resolver.go:43`), `ingestion.TextExtractor` (`data/ingestion/readers.go:20`), `evals.Asker` và `evals.ModelCounter` (`evals/runner.go:14`, `:21`). Chuẩn hoá, ý định, định tuyến, kiểm là hàm / kiểu cụ thể với chữ ký khác bảng (ví dụ `router.Route`, `intelligence/router/router.go:170`). `Tool`, `ContextBuilder`, `WorkflowEngine`, `Guard`, `TriggerEvaluator`, `CommandHandler`: chưa có.

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

## 7. Mô hình dữ liệu (PostgreSQL 17; pgvector: ĐÍCH)

Mọi bảng nghiệp vụ có `tenant_id`, bật `ROW LEVEL SECURITY` + `FORCE`, chính sách đọc `current_setting('eais.tenant_id')`. Ứng dụng chạy bằng vai không có `BYPASSRLS`; khởi động **từ chối** chạy nếu vai có quyền vượt RLS.

> **Trạng thái.** HIỆN CÓ: 4 bảng `entities`, `entity_aliases`, `facts`, `documents`, RLS ENABLE + FORCE, chính sách đọc `eais.tenant_id`, vai ứng dụng `eais_app` (`db/migrations/0001_win.sql`, `0002_dinh_danh_eais.sql`); `documents` có bản / trạng thái / hiệu lực / phạm vi từ `0005_tai_lieu_phien_ban.sql`. "Từ chối vai vượt RLS" hiện là hàm `pgstore.Open` (`platform/pgstore/pgstore.go:23-45`); chưa có tiến trình dịch vụ khởi động. Các nhóm bảng khác trong bảng dưới: ĐÍCH.

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

> **Trạng thái.** HIỆN CÓ (`tenants/construction/pack/`, 7 tệp): `pack.yaml`, `lexicon.yaml`, `metrics.yaml`, `intents.yaml`, `templates.yaml`, cộng `policy.yaml` (vai → capability, phạm vi) và `sources.yaml` (loại nguồn → định dạng, cột, chỉ số) chưa có trong cây trên. Danh mục thực thể đọc từ `entities.json` của thư mục dữ liệu. `entities.yaml`, `tools.yaml`, `skills.yaml`, `workflows.yaml`, `triggers.yaml`, `validation.yaml`: ĐÍCH. Trong tệp có thật: `pack.yaml` hiện chỉ có `id`, `tenant`, `version`, `language`, `timezone`; model policy, ngân sách và ngưỡng ý định là ĐÍCH.

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

Hiện mọi cổng trong bảng đang **đóng**: chưa capability nào ở đây có mã.

---

## 10. Stack

| Thành phần | Chọn | Ghi chú |
|---|---|---|
| Ngôn ngữ chính | Go 1.26.x | `go 1.26.0` + `toolchain` bản vá |
| Router HTTP | chi v5 | ĐÍCH: chưa có HTTP, `go.mod` chưa có chi |
| DB | PostgreSQL 17, driver pgx v5 | ảnh Docker ghim digest; pgvector: ĐÍCH |
| Python | 3.12, dịch vụ tính toán sau ranh giới §11 | ĐÍCH: chưa có dịch vụ Python nào. Dịch vụ đầu tiên dự kiến: `document-parser`; không truy cập DB, không chứa luật nghiệp vụ |
| Hàng đợi/cache | chưa dùng | Redis/Kafka chỉ thêm khi số đo đòi (`CLAUDE.md` §19; `docs/ROADMAP-DAY-DU.md` §6, Phase 9) |
| Model | qua Model Gateway | HIỆN CÓ: cấu hình bằng biến môi trường toàn tiến trình (`EAIS_MODEL_*`), URL mặc định của nhà cung cấp gán ở `intelligence/models/messages.go:437`. Cấu hình theo tenant và ngân sách: ĐÍCH |

---

## 11. Ranh giới Go ↔ Python — "Go first, Python by evidence"

> **Go là ngôn ngữ của hệ thống và control plane. Python chỉ được tồn tại sau một ranh giới capability tường minh, khi hệ sinh thái ML/tài liệu/dữ liệu của Python mang lại giá trị đo được. Dịch vụ Python KHÔNG BAO GIỜ sở hữu: phân quyền tenant, giao dịch nghiệp vụ, trạng thái workflow, chính sách nguồn sự thật, hay điều phối lõi.**

> **Trạng thái.** Chưa có dịch vụ Python nào. PDF báo cáo ngày (L1) đọc bằng `pdftotext` (`data/ingestion/readers.go:26-35`). Lộ trình dịch vụ Python ở cuối mục là ĐÍCH.

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
