# 05 — R0 Definition Freeze: gói định nghĩa EACL V3

> Trạng thái: **ỨNG VIÊN đông cứng, chờ chủ dự án duyệt.**
> - Mục nào ghi **ĐỀ XUẤT** là do lead tự đề xuất, đảo được.
> - Mục nào ghi **NGUYÊN VĂN** lấy từ Blueprint/Playbook.
>
> R0 đạt (Playbook §9) khi đủ: tên/định nghĩa, giả thuyết, hai job, non-goals, G1–G6, baseline, mẫu task/story đều đông cứng; có luật kiểm soát thay đổi; **0 blocker định nghĩa chưa giải**. Trạng thái từng điều kiện ở mục 5.

## 1. Vì sao cần R0 trước mọi dòng mã

Mã V3 sẽ do nhiều phiên và agent viết. Nếu "brief đúng", "task an toàn", "thắng" chưa được đông cứng bằng câu nhị phân, mỗi agent sẽ hiểu một kiểu. Khi đó cổng bị viết theo mã, đúng kiểu hỏng "tự chấm bài" (`CLAUDE.md` §1).

R0 không làm AI. Nó trả lời: **giải vấn đề gì, cho ai, thắng nghĩa là gì, đo bằng gì.**

## 2. Gói định nghĩa

### 2.1 Tên và định nghĩa — NGUYÊN VĂN

- Tên: **EACL — Enterprise AI Control Layer**. Tên cũ: EAIS.
- Định nghĩa: *"EACL is a reference architecture and executable control layer for integrating probabilistic AI capabilities into existing enterprise applications while the applications retain ownership of business truth, permissions, state, transactions and workflows."*
- Câu hỏi sống còn: *"Can a small reusable control layer add useful AI to existing enterprise software while the business application keeps ownership of truth, permissions, state, transactions, and workflows?"*
- Năm luật:
  1. Business first.
  2. AI only where uncertainty creates value.
  3. Software remains authority.
  4. Important outcomes require evidence.
  5. Complexity must earn its place through measured value.

### 2.2 Hai job MVP — NGUYÊN VĂN, kèm phạm vi app

| Job | Câu người dùng | Kết quả đòi |
|---|---|---|
| A — UNDERSTAND | "Acme 30 ngày qua thế nào? Có gì cần chú ý?" | brief có bằng chứng; fact chính xác khớp trạng thái app; khẳng định không căn cứ bị chặn hoặc đánh dấu bất định |
| B — ACT | "Tạo task follow-up cho Acme vào thứ Hai." | đúng **một** task, đi đường có quyền, đã kiểm hợp lệ, idempotent, đã đọc lại xác minh, có audit |

App tham chiếu chỉ có **Customer, Activity, Task, Document**, cộng người dùng/vai/quyền và tenant. Contact chỉ thêm khi một ca đông cứng cần (xem C9).

Không có:
- dự báo cơ hội bán hàng;
- gửi email;
- chiến lược bán hàng tự trị;
- sửa trạng thái CRM ngoài tạo task;
- multi-agent.

(Playbook §10)

### 2.3 Cổng G1–G6, và cổng cũ được đổi tên

Định nghĩa đầy đủ PASS/FAIL/DoD của từng cổng: Playbook §8 (nguyên văn, xem bản công khai ở TASK-R0-04).

**Bắt buộc từ R0:** bảy cổng WIN cũ của Fuel đổi tên thành **W-G1…W-G7** trong mọi tài liệu mới, để không lẫn với G1–G6 của V3.

| Cổng cũ | Nghĩa cũ (`04` §6) | Ứng với cổng V3 |
|---|---|---|
| W-G1 DATA | nạp đủ, 0 bản ghi rơi im lặng, metamorphic | G2 TRUTH (đầu vào đúng) |
| W-G2 IDENTITY | thực thể, mơ hồ ⇒ hỏi lại | G1 UNDERSTAND |
| W-G3 TRUTH | nghĩa nghiệp vụ, thẩm quyền, dung sai | G2 TRUTH (+ phần nghĩa của G1) |
| W-G4 DECISION | định tuyến, đếm lời gọi model | G3 ROUTE |
| W-G5 CONTROL | tenant, quyền, ràng buộc claim, chèn lệnh | G4 CONTROL (+ phần "claim không căn cứ" của G2) |
| W-G6 PROOF | bộ ca đông cứng, trace, phát lại, đột biến | G5 PROOF |
| W-G7 VALUE | so baseline RAG bằng model thật | G6 VALUE của Fuel (R1 thêm B1) |

### 2.4 Baseline — ĐỀ XUẤT cách hiện thực hoá (định nghĩa B0–B3 là nguyên văn Blueprint §16)

| | Fuel (R1) | Customer Ops (R6) |
|---|---|---|
| B0 người / app thường | không đo (Blueprint §5 không đòi) | người dùng tự làm trên app tham chiếu, bấm giờ trên mẫu 10 ca. **Chỉ báo cáo, không làm cổng** (D-V3-04) |
| B1 LLM trực tiếp | model + câu hỏi + **toàn bộ bản ghi thô** của tenant A trong cửa sổ ngày của ca (±1 ngày), dạng văn bản. Không chuẩn hoá, không truy xuất, không nhận dạng. Cửa sổ ngày lấy từ ca: lợi cho B1, tức là so **bảo thủ** | app tìm khách bằng `LIKE` đơn giản, lấy bản ghi khớp đầu, đổ toàn bộ activity/task/document 30 ngày vào model. Job B: model trả JSON gọi `create_task`, app thực thi bằng token người dùng, **không** khoá idempotency, **không** đọc lại |
| B2 AI/RAG đơn giản | `evals/baseline` hiện có (mỗi quan sát một đoạn, top-k = 10, cùng retriever + model) | truy xuất top-k trên mọi bản ghi của tenant, rồi model. Job B như B1 |
| B3 EACL | đường hiện có (S0–S10) | đường V3 (R3/R4) |

Luật công bằng, áp cho mọi đường:
- cùng model, temperature 0, cùng `max_tokens`;
- prompt đông cứng **trước** khi chạy;
- cùng bộ ca;
- cùng kịch bản lỗi bơm (ví dụ "ghi xong mất phản hồi");
- app vẫn tự kiểm quyền với mọi đường, vì đó là app có sẵn.

### 2.5 Ngưỡng — ĐỀ XUẤT, phải chốt trước lần chạy model thật đầu tiên (D-V3-02)

Playbook đòi ngưỡng số phải đông cứng **trước** benchmark cuối, nhưng không đưa số nào. Đề xuất dưới đây theo đúng khuôn D5 mà repo đã dùng (ADR-0001).

**Bộ ca Customer Ops**
- ≥ 60 ca tổng cho Job A + Job B;
- mỗi lớp lỗi áp dụng được (mục 2.6) có ≥ 5 ca;
- ≥ 40% ca do phiên **không đọc mã** soạn;
- băm sha256, đóng băng trước khi bắt đầu mã R3/R4.

**Fuel — G6 của R1**
- Giữ D5 với B2: temperature 0; k = 3, lấy trung vị; mọi lớp EACL ≥ B2; ≥ 4/8 lớp EACL ≥ B2 + 20 điểm %; EACL rò rỉ tenant = 0, claim không căn cứ = 0.
- Áp **cùng luật** với B1.
- Kết luận:
  - WIN = đạt với cả B1 lẫn B2;
  - PARTIAL = đạt với đúng một;
  - FAIL = không đạt với cái nào.
- Chi phí và độ trễ: chỉ báo cáo. Route tất định của EACL gọi model 0 lần.

**Customer Ops — G1…G6**

| Cổng | Ngưỡng đề xuất |
|---|---|
| G1 | 100% ca thực thể quan trọng đúng **hoặc** hỏi lại đúng; 0 lần đoán im lặng; 0 lần nhận dạng chéo tenant. Hỏi lại thừa trên ca không mơ hồ ≤ 5% |
| G2 | 100% fact chính xác trong brief (số đếm, ngày, trạng thái) khớp trạng thái app; 0 khẳng định quan trọng không căn cứ được giao; mọi khẳng định trỏ tới id bằng chứng có trong ngữ cảnh đã lọc quyền |
| G3 | 100% câu hỏi fact chính xác trả lời với **0** lời gọi model (đếm bằng gateway giả); mỗi brief gọi model đúng 1 lần; route truy vết được trên 100% ca |
| G4 | 0 lần ghi không có quyền; 0 lần đọc/ghi chéo tenant; 0 task trùng khi thử lại hoặc mất phản hồi; 100% đề xuất sai bị chặn/hỏi lại trước thực thi; 0 hành động cần duyệt chạy mà không có duyệt; test kiến trúc chứng minh **chỉ một** đường mã thực thi được ghi |
| G5 | 100% câu/hành động đã giao có trace đủ 11 trường (Playbook §8 G5); phát lại chặng tất định 0 lệch |
| G6 | Với từng lớp áp dụng: B3 ≥ B1 và ≥ B2. Ở ≥ 3 lớp, B3 hơn ≥ 20 điểm %. B3 giữ đủ các số 0 cứng của G4. Chi phí trên mỗi kết quả thành công đã xác minh của B3 ≤ 2 × của baseline rẻ hơn. Công tích hợp **ghi lại, chưa đặt ngưỡng** trong MVP; ngưỡng đặt ở R8/R9. Chỉ số ghi lấy từ bộ đã nêu trong lịch sử dự án: thời gian tích hợp, % mã riêng của tích hợp, công ánh xạ, số thay đổi schema, số test phải thêm, cộng số tệp chung bị sửa |

### 2.6 Phân loại lỗi — giữ 10 lớp của Blueprint §42 (thân chung chung, chỉ lấy tên lớp)

| Lớp | Áp cho | Ứng với lớp Fuel hiện có (`04` §5) |
|---|---|---|
| Identity & Tenant | A, B, Fuel | `permission` |
| Entity Resolution | A, B, Fuel | `entity`, `ambiguity` |
| Business Semantics | A, Fuel | `semantic`, `representation` |
| Source Authority | A, Fuel | `source_conflict` |
| Context | A | — (mới: ngữ cảnh thừa, thiếu, cũ, không được phép) |
| Routing | A, B, Fuel | `routing` |
| Model Output | A, B, Fuel | `unsupported_answer` |
| Policy & Permission | A, B, Fuel | `permission` |
| Action Safety | B | — (mới) |
| Reliability | A, B | — (mới: provider/API/mạng) |

### 2.7 Mười hai quyết định nền (Blueprint §44, bỏ phần nhân bản)

| # | Quyết định | Đã có ở repo? |
|---|---|---|
| 1 | Giữ app nghiệp vụ là nguồn sự thật | `CLAUDE.md` P4; mới với Customer Ops |
| 2 | RAG không bắt buộc | `CLAUDE.md` §12 |
| 3 | Đầu ra model là đề xuất | `CLAUDE.md` P3 |
| 4 | Cổng policy ở mức capability | mới: I6 |
| 5 | Thẩm quyền nguồn thích ứng (app đã có thẩm quyền thì dùng thẳng) | mới với Customer Ops |
| 6 | Mức trưởng thành bằng chứng E0–E6 | `docs/03` có D0–D4; dùng cả hai |
| 7 | Không nhận là framework trước portability | mới, R10 |
| 8 | Không multi-agent trong MVP | `04` §4 |
| 9 | Không fine-tune khi chưa có benchmark trượt | `CLAUDE.md` §19 |
| 10 | Ưu tiên workflow engine sẵn có | `CLAUDE.md` §15 |
| 11 | Thay được provider | gateway theo cấu hình |
| 12 | Thực thi hành động idempotent | mới, R2/R4 |

### 2.8 Đánh số phase — ĐỀ XUẤT dùng số của Blueprint/Playbook

Playbook là nguồn số 2 về thẩm quyền và dùng R0…R10, khớp Blueprint. Hai cách đánh số khác (bài giảng R0–R9, hội thoại cũ P0–P7) chỉ để tra ngược.

| V3 (chuẩn) | Bài giảng | Hội thoại cũ | Roadmap EAIS cũ (`05`) |
|---|---|---|---|
| R0 Definition Freeze | R0 | P0 | — |
| R1 Fuel Truth Benchmark | R1 | P1 | Phase 0 (WIN, W-G7) |
| R2 Reference Customer Ops App | R2 | P2 | — |
| R3 UNDERSTAND | R3 | P2 | — |
| R4 ACT | R4 | P3 | Phase 1 (hành động, khi đó trên Fuel) |
| R5 Security + Failure Injection | R5 | — | — |
| R6 Baseline + Value Benchmark → **quyết MVP** | R6 | P4 | — |
| R7 Core Extraction | R7 | P5 | — |
| R8 Second materially different integration | R8 | P6 | Phase 4 |
| R9 Portability benchmark | R8 | P6 | Phase 4 |
| R10 Framework Gate | R9 | P7 | — |
| R11 Enterprise hardening (chỉ khi có lý do) | — | — | Phase 5, 9 |
| R12 Advanced RAG/Agent/Fine-tune (có cổng) | — | — | Phase 6, 7, 8 |

Phase 2 Workflow và Phase 3 Event của roadmap cũ **không có trong V3**. V3 dùng workflow của app (Blueprint §24).

### 2.9 Ba thuật ngữ thắng, không được trộn

| Thuật ngữ | Nghĩa |
|---|---|
| **Architecture Proof** | G1–G5 đạt trên Customer Ops |
| **MVP WIN** | G1–G6 đạt trên Customer Ops (Playbook §7.7) |
| **V3 WIN** | MVP WIN + R1 có kết luận có baseline + tích hợp thứ hai khác hẳn dùng lại core mà không thiết kế lại (Blueprint §0.6), tức là cần tới R9 |

### 2.10 Kiểm soát thay đổi — NGUYÊN VĂN Playbook §13

Sau R0, tính năng mới vào MVP cần đủ 8 mục:
1. lỗi nghiệp vụ có tên;
2. vì sao MVP hiện tại không chứng minh được luận đề nếu thiếu nó;
3. user story;
4. AC;
5. chỉ số kỳ vọng;
6. độ phức tạp thêm vào;
7. phương án đã cân nhắc;
8. quyết định ACCEPT / DEFER / REJECT.

**Mặc định là DEFER.** Agent, MCP, fine-tune, RAG nâng cao, workflow chung, ontology toàn cục, chợ connector giữ DEFER cho tới khi một lỗi đo được đòi chúng.

## 3. Mâu thuẫn tìm thấy và cách xử lý đề xuất

| # | Mâu thuẫn | Nguồn | Đề xuất |
|---|---|---|---|
| C1 | `CLAUDE.md` §9 và `README.md` gọi sản phẩm là "Multi-tenant Enterprise AI Systems Platform" | repo ↔ Blueprint §3 | Chủ dự án sửa `CLAUDE.md` (tệp của chủ dự án; agent chỉ đề xuất câu chữ, D-V3-06). README sửa ở TASK-R0-03 |
| C2 | Cổng cũ G1–G7 trùng tên, khác nghĩa với G1–G6 V3 | `04` ↔ Playbook | đổi tên cũ thành W-G1…W-G7 (mục 2.3) |
| C3 | Bốn cách đánh số phase. Lịch sử còn **≥ 6 bộ roadmap EAIS** khác số: M0–M9, M0–M8, Phase 0–10, 8 từ khoá, P0–P8, [0]–[6]. Bộ [0]–[6] mới nhất trước V3 khớp khung R0–R10 | Blueprint, bài giảng, hội thoại, `05` cũ | dùng số Blueprint/Playbook (mục 2.8); các bộ cũ chỉ để tra ngược |
| C4 | `CLAUDE.md` §9/§24: tenant tham chiếu là Construction + Sports. V3: Fuel + Customer Ops | repo ↔ V3 | Sports thành ứng viên cho R8 (khác hẳn: sự kiện thời gian thực). Chọn domain R8 khi tới R7 |
| C5 | "V3 WIN" (Blueprint §0.6, có portability) khác "MVP WIN" (G1–G6) | Blueprint ↔ Playbook | ba thuật ngữ ở mục 2.9 |
| C6 | Không tài liệu nào có ngưỡng số (G1 "ngưỡng đã thống nhất", G6 "lợi thế có nghĩa", R8/R9 "ngưỡng chấp nhận"). Lịch sử có 3 bộ tiêu chí WIN cho Product MVP (A1–A10, P1–P10, baseline A–D); chỉ hai bộ đầu có "10/10", **không bộ nào có ngưỡng so với baseline** | Playbook + lịch sử | mục 2.5, chờ D-V3-02 |
| C7 | Catalog Blueprint §42–46 là khuôn nhân bản; §43 gán cổng xoay vòng | Blueprint | không dùng làm ca/truy vết/backlog; giữ 10 lớp + 12 quyết định (`00` §4.1) |
| C8 | `docs/requirements/catalog.yaml` dán 294 REQ là "MVP" | repo ↔ V3 | đông cứng sổ cũ thành **lịch sử EAIS V1/V2**. Truy vết V3 dùng mã `V3-…` ở `03`. Chỉ đổi `phase` một REQ cũ khi task V3 cần nó |
| C9 | Contact: Blueprint §4.1 có; Playbook R2 PASS và hội thoại V3 §28 không có | Blueprint ↔ Playbook | Playbook thắng: Contact **DEFER**, chỉ thêm khi một ca đông cứng cần |
| C10 | Prompt điều phối cho dùng Opus làm agent con. Luật đang hiệu lực của chủ dự án: agent con chỉ sonnet/haiku, tối đa 3 | prompt ↔ luật vận hành | việc tầm Opus do **phiên lead** làm. Muốn agent con Opus thì chủ dự án nói rõ (D-V3-05) |
| C11 | Blueprint nêu tên hai hệ thống của công ty cũ | Blueprint ↔ repo PUBLIC | bản công khai ẩn danh ("hệ A", "hệ B"); bản gốc ở `docs/sources/` (ignore) |
| C12 | `04` §6: WIN = 7/7. Playbook R1: chỉ cần "câu hỏi giá trị có câu trả lời rõ" (WIN/PARTIAL/FAIL) | repo ↔ Playbook | R1 đóng được bằng PARTIAL/FAIL trung thực; không cần thắng mới sang R2 |
| C13 | Việc treo từ bàn giao 20/10: duyệt ADR-0002 (D7–D17); 2 đường dẫn nội bộ của hệ cũ trong `docs/02-migration-audit.md`; danh sách nhóm lỗi P10 trong `CLAUDE.md` lệch `CONVENTIONS` §4.2; `go mod tidy` offline | bàn giao | gom vào R0 (D-V3-07) |
| C14 | Bộ khung W-G7 chỉ có B2; V3 R1 đòi cả B1 | repo ↔ Blueprint §5 | TASK-R1-03 |
| C15 | Hội thoại đầu tách "AI-native CRM" thành roadmap riêng; sau gộp thành vertical; V3: "app tham chiếu, không phải CRM" | lịch sử ↔ V3 | V3 thắng: không có sản phẩm CRM trong MVP |
| C16 | Hội thoại đầu: DEW là "cách ta xây phần mềm". Prompt điều phối: chỉ Claude Code | lịch sử ↔ prompt | dự án này xây bằng Claude Code; DEW không nằm trên đường găng |
| C17 | Blueprint §19 hình repo đích (`eacl/core`, `verticals/`, `integrations/`) khác cây hiện tại | Blueprint ↔ repo | **không tái cấu trúc trước R7** (Blueprint tự dặn). Mã mới đặt theo TASK-R2-01 |
| C18 | Định nghĩa B1 "LLM trực tiếp" không có cách hiện thực | Blueprint | mục 2.4 |
| C19 | Playbook §4 (mẫu EPIC) hỏng định dạng: rơi vào khối thụt lề vì fence mẫu TASK không đóng | Playbook | sửa định dạng khi đưa bản công khai (TASK-R0-04), không đổi chữ |

## 4. Câu cần chủ dự án chốt (mỗi câu kèm khuyến nghị)

| # | Câu | Khuyến nghị | Vì sao | Chưa chốt thì |
|---|---|---|---|---|
| D-V3-01 | Model thật + trần chi phí cho R1/R3/R6 | Claude Haiku 5.5 qua client `anthropic` đã có. **Trước khi chi đồng nào**, chạy TASK-R1-04 ra con số token ước lượng, rồi chốt trần | client có sẵn; mọi đường dùng chung một model nên so công bằng; dữ liệu là giả lập, không có dữ liệu thật đi ra ngoài | R1-05, smoke model thật của R3, R6 CHẶN. R0, R2, phần tất định của R3/R4/R5 vẫn chạy |
| D-V3-02 | Ngưỡng mục 2.5 | duyệt như đề xuất hoặc sửa số | cùng khuôn D5 đã dùng; có số 0 cứng cho an toàn | không chạy được benchmark có kết luận |
| D-V3-03 | Đổi tên mã nguồn và repo | (a) đổi tên trong tài liệu và README ngay ở R0. (b) Đổi module `github.com/luannt2002/eacl` + thư mục + repo GitHub trong **một commit riêng, trước khi mã R2 vào** | hôm nay 26 gói phải đổi import; càng để lâu càng nhiều. GitHub tự chuyển hướng URL cũ. (b) đụng remote công khai nên phải hỏi | giữ tên cũ trong mã, tài liệu gọi EACL |
| D-V3-04 | Đo B0 | quyết lúc chuẩn bị R6. Khuyến nghị: thêm vài trang HTML chỉ đọc + form tạo task cho app (~vài trăm dòng), chủ dự án tự làm 10 ca, bấm giờ | B0 qua `curl` không giống thật, làm EACL trông có lợi giả | G6 không có số "tiết kiệm thời gian"; ghi CHƯA ĐO |
| D-V3-05 | Agent con Opus? | Không. Lead (Opus) làm kiến trúc, an ninh, soát pha. Agent con sonnet/haiku, tối đa 3 | đúng luật đang hiệu lực; đỡ tốn | — |
| D-V3-06 | Sửa `CLAUDE.md` §9/§24/§25 | chủ dự án sửa theo câu chữ đề xuất ở TASK-R0-03 | tệp của chủ dự án | agent đọc thấy hai định nghĩa sản phẩm khác nhau |
| D-V3-07 | Việc treo C13 | duyệt ADR-0002; thay 2 đường dẫn nội bộ bằng "hệ thống cũ"; chốt một danh sách P10 | dọn trước khi thêm tài liệu V3 | nợ tài liệu |
| D-V3-08 | Commit/push `docs/v3/` | chỉ khi chủ dự án nói. Bộ tài liệu đã ẩn danh, không có đường dẫn máy | repo PUBLIC | tài liệu nằm ở máy |
| D-V3-09 | Kiến trúc app tham chiếu (ADR-0003, TASK-R2-01) | tiến trình riêng, DB riêng (`customerops`, vai riêng, cùng container PG), REST JSON bằng `net/http` thư viện chuẩn, token bearer cho người dùng seed; **cấm import hai chiều** giữa app và EACL (test kiến trúc) | đo "integration tax" phải trung thực. App là "hệ có sẵn" thật, EACL chỉ nói chuyện qua API như với hệ của khách | rủi ro "sân nhà" (R-08 ở `08`) |
| D-V3-10 | Vị trí tài liệu cũ | `04`, `05`, `catalog.yaml` ghi đầu tệp "lịch sử EAIS; V3 xem `docs/v3/`". `04` vẫn là hợp đồng của R1 | tránh hai nguồn sự thật | agent đọc nhầm `05` cũ |

## 5. Điều kiện R0 đạt (Playbook §9) — trạng thái lúc viết

| Điều kiện | Trạng thái |
|---|---|
| Tên/định nghĩa EACL đông cứng | ứng viên ở mục 2.1, chờ duyệt |
| Giả thuyết sản phẩm đông cứng | ứng viên (`00` mục D), chờ duyệt |
| Hai job MVP đông cứng | ứng viên ở mục 2.2, chờ duyệt |
| Non-goals đông cứng | ứng viên (`00` mục G), chờ duyệt |
| G1–G6 đông cứng | định nghĩa có; **ngưỡng số chờ D-V3-02** |
| Baseline đông cứng | cách hiện thực ở mục 2.4, chờ duyệt |
| Mẫu task/story được dùng | có (Playbook §3, §23; `06` theo mẫu) |
| Có luật kiểm soát thay đổi | mục 2.10 |
| DoD: một gói định nghĩa có revision + 0 blocker | **CHƯA**: blocker còn mở là D-V3-01 (chỉ chặn phần cần model), D-V3-02, D-V3-03 |

Duyệt xong thì ghi revision (commit hoặc tag `v3.0.0-r0`) vào `09-EVIDENCE-INDEX.md`. Từ đó, đổi mục 2 phải đi đường mục 2.10.
