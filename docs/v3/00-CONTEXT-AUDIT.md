# 00 — Context Audit: EACL V3 bắt đầu từ đâu

> Trạng thái: **bản lập kế hoạch 20/10/2026, chưa được chủ dự án duyệt.** Chưa sửa dòng mã nào.
> Đọc kèm: `05-R0-DEFINITION-FREEZE.md` (các câu cần chốt), `01-CURRENT-STATE-MATRIX.md` (đã có gì).
> Luật đọc tài liệu này: mỗi khẳng định ghi rõ nó đang ở mức nào (mục 3). "Đã học" không phải "đã làm".
> "Đã làm" không phải "đã chứng minh".

## 0. Đã đọc gì và đọc tới đâu

Nguồn V3 là một bản chép hội thoại ChatGPT dài 36.742 dòng. Bản lưu ở `docs/sources/eacl-v3/` kèm `SHA256SUMS`; thư mục này nằm trong `.gitignore`, không lên repo public.

| Nguồn | Phạm vi | Cách đọc | Kết quả |
|---|---|---|---|
| Master Blueprint V3 §0–41, §47–52 | 1.135 dòng | lead đọc nguyên văn | toàn bộ định nghĩa, kiến trúc, roadmap, DoD, stop rules |
| Master Blueprint V3 §42–46 | 14.000 dòng | đo bằng script, đọc mẫu từng loại | **catalog nhân bản**, xem mục 4.1 |
| Success Definition Playbook V3 | 1.691 dòng | lead đọc nguyên văn | hợp đồng PASS/DONE, 15 MVP-AC, G1–G6, R0–R10, US-001/002 |
| Hội thoại "V3 định nghĩa" và "đã học ≠ đã chứng minh" | 2.079 dòng | lead đọc nguyên văn | lý do của V3, bài học từ hai hệ tham khảo, chấm hiện trạng EAIS |
| Hội thoại EAIS V1 → V2 → V2.1 (3 roadmap, CRM, V2, V2.1, Quân IT) | 7.293 dòng | agent Sonnet đọc 7.293/7.293 dòng, tóm 8 câu trả lời (55 quyết định theo thời gian, 15 mâu thuẫn treo); lead kiểm 4 khẳng định, khớp nguyên văn | đưa vào `05` C3, C6, §2.5 và `08` R-24 |
| Tài liệu học phần 1: study guide "AI is just another tech stack" + workshop notes | 4.908 dòng | agent Sonnet đọc 4.908/4.908 dòng, tách 5 câu trả lời; lead kiểm 3 khẳng định, khớp | đưa vào §K |
| Tài liệu học phần 2: RAG roadmap, 7 subsystem, master roadmap AI, Problem Map | 5.517 dòng | agent Sonnet đọc 5.517/5.517 dòng, tách 6 câu trả lời (+ 1 đuôi cụt); lead kiểm 2 khẳng định, khớp | đưa vào §K |
| Repo EAIS (HEAD `85108c4`) | `CLAUDE.md`, `README.md`, `docs/STATUS.md`, `03`, `04`, `05`, ADR-0001/0002, sổ yêu cầu, cây mã | lead đọc + đo | xem `01`, `02` |
| Mốc test của repo | toàn bộ `go test`, scale smoke, winreport bộ nhớ + PG | lead chạy lại 20/10 15:22 | 224/0/0, xem `02` |

**Độ phủ: 36.742/36.742 dòng đã đọc.**
- Lead đọc nguyên văn khoảng 5.300 dòng có thẩm quyền.
- Catalog khuôn khoảng 14.000 dòng được đo bằng script.
- Agent đọc 17.718 dòng còn lại; lead kiểm mẫu từng phần.

Bản tóm tắt của agent lưu ở `docs/sources/eacl-v3/` (ignore). Nguồn thẩm quyền cho V3 vẫn là Playbook và Blueprint. Phần lịch sử và tài liệu học chỉ dùng để tìm mâu thuẫn và bài học, đã đưa vào mục 2.I, 2.K và `05` §3.

## 1. Thẩm quyền tài liệu

Thứ tự ưu tiên khi hai nguồn nói khác nhau: số nhỏ thắng. Không chọn ngầm. Mọi chỗ lệch ghi ở `05` §3.

| # | Tài liệu | Mục đích | Có thẩm quyền về | Bản |
|---|---|---|---|---|
| 1 | Repo + runtime | sự thật hiện tại | mã nào có, test nào chạy, số đo | HEAD `85108c4`, mốc 20/10 15:22 |
| 2 | Success Definition Playbook V3 | HOW TO PROVE | PASS/DONE của task, story, gate, phase, MVP | sha256 `b2fe3a1a…` |
| 3 | Master Blueprint V3 (§0–41, §47–52) | WHAT/WHY | định nghĩa EACL, kiến trúc, roadmap, non-goals | sha256 `fa5de366…` |
| 4 | Test / eval hiện có | hành vi đã chốt | 117 ca WIN (sha256 `ffdd704b…`), 224 test | mốc 20/10 |
| 5 | Tài liệu hiện hành của repo | luật vận hành, hợp đồng WIN cũ | `CLAUDE.md` (của chủ dự án), `03` thang bằng chứng, `04` WIN Fuel, ADR-0001/0002 | 20/10 |
| 6 | Lịch sử EAIS | nguồn gốc quyết định | `05-master-roadmap.md`, `docs/requirements/catalog.yaml` (422 REQ), brief cũ | 19–20/10 |
| 7 | Tài liệu học | tư duy nền | ghi chú video, Problem Map | 20/10 |

Blueprint §42–46 và §50 **không có thẩm quyền** gì thêm ngoài nội dung riêng biệt liệt kê ở mục 4.1.

## 2. EACL là gì: tóm theo các câu A–N

### A. EACL là gì

**EACL — Enterprise AI Control Layer.** Lớp đứng giữa phần mềm doanh nghiệp đã có và các capability AI hoặc tất định. AI tham gia vào công việc. Ứng dụng nghiệp vụ vẫn sở hữu sự thật, quyền, trạng thái, giao dịch và quy trình.

Tên cũ EAIS ("Enterprise AI Systems") bị bỏ vì quá rộng. Nó dễ bị hiểu thành nền tảng model, agent framework, RAG framework hay một CRM thay thế. (DEFINED: Blueprint §3)

EACL **chưa phải framework**. V3 bắt đầu là kiến trúc tham chiếu cộng một bản cài đặt tham chiếu chạy được. Nó chỉ được gọi là framework/SDK khi qua Framework Gate (R10). (DEFINED: Blueprint §0, §41)

### B. V3 giải vấn đề gì

Doanh nghiệp không thiếu model. Thứ khó là cho AI xác suất tham gia vào hệ thống đang chạy mà không làm hỏng sự thật, quyền, trạng thái hay vận hành. EACL đứng ở sáu khoảng trống (DEFINED: Blueprint §2.2):

| Có | Nhưng |
|---|---|
| dữ liệu | dữ liệu ≠ ngữ cảnh đáng tin |
| tri thức | tri thức ≠ sự thật có thẩm quyền |
| model | model ≠ thẩm quyền nghiệp vụ |
| câu trả lời AI | câu trả lời ≠ quyết định đã kiểm |
| công cụ | capability ≠ quyền |
| tự động hoá | thực thi ≠ kết quả đã xác minh |

### C. North Star

**Business Reality → Trusted Context → Useful Intelligence → Controlled Action → Verified Outcome → Measured Value.** (DEFINED)

### D. Giả thuyết sản phẩm

Một lớp kiểm soát nhỏ, dùng lại được, có thể làm AI **hữu ích và đáng tin hơn rõ rệt** bên trong một ứng dụng nghiệp vụ có sẵn, so với nối model trực tiếp hoặc AI/RAG đơn giản. Lớp đó không lấy quyền sở hữu sự thật, quyền, trạng thái hay giao dịch. (DEFINED: Playbook §7.1)

Đây là **giả thuyết cần bác bỏ được**, không phải sự thật về nhu cầu thị trường (Blueprint §51).

### E. MVP

MVP là **Minimum Viable Proof**, không phải đống tính năng đầu tiên (Playbook §6).

- **App tham chiếu Customer Operations** chỉ có Customer, Activity, Task, Document. Contact là tuỳ chọn, xem `05` §3 C9. Cố ý không phải CRM đầy đủ.
- **Job A — UNDERSTAND:** "Acme 30 ngày qua thế nào? Có gì cần chú ý?" → brief 30 ngày, mỗi khẳng định quan trọng có bằng chứng.
- **Job B — ACT:** "Tạo task follow-up cho Acme vào thứ Hai." → đúng **một** task, đã kiểm quyền, đã kiểm hợp lệ, idempotent, đã đọc lại xác minh.
- **Fuel Discrepancy** (bài toán hiện có của repo) được giữ làm **Truth Benchmark** (R1), không bỏ.

### F. Cổng WIN G1–G6

| Cổng | Câu phải chứng minh (Playbook §8) |
|---|---|
| G1 UNDERSTAND | hiểu đúng ý định, thực thể, nghĩa nghiệp vụ, hoặc hỏi lại khi mơ hồ |
| G2 TRUTH | fact quan trọng lấy từ nguồn có thẩm quyền, hoặc đánh dấu bất định/xung đột |
| G3 ROUTE | dùng capability nhỏ nhất đủ dùng, không gọi AI khi không cần |
| G4 CONTROL | AI không vượt được tenant, quyền, policy, kiểm hợp lệ, duyệt, bất biến nghiệp vụ |
| G5 PROOF | dựng lại được câu trả lời/hành động từ bằng chứng, trạng thái hiện tại, route, quyết định |
| G6 VALUE | hơn cách tích hợp đơn giản một cách có nghĩa, với chi phí tích hợp và vận hành chấp nhận được |

MVP WIN chỉ khi **cả sáu** đạt. Đạt G1–G5 mà trượt G6 thì ghi bắt buộc: **"Architecture Proof PASS; Product MVP FAIL/PARTIAL"**.

Repo hiện có bảy cổng **cũ** cùng tên G1–G7 nhưng khác nghĩa. Bảng ánh xạ ở `05` §2.3.

### G. Không làm (non-goals)

EACL không phải CRM, ERP, chatbot framework, RAG framework, agent framework, BPM engine, nền tảng train model, lớp trừu tượng vector DB, MCP wrapper, ontology toàn cục.

MVP không có:
- agent bán hàng tự trị, multi-agent;
- workflow builder chung;
- train/fine-tune;
- vector DB bắt buộc;
- hàng chục module CRM;
- chợ connector.

(Blueprint §0.3, §4.5)

### H. Bất biến kiến trúc

Mỗi dòng có nguồn trong Blueprint/Playbook; cột cuối là trạng thái hiện tại, kiểm ở `01`.

| # | Bất biến | Trạng thái hiện tại |
|---|---|---|
| I1 | Tenant xác lập **trước** mọi truy xuất; tenant không lấy từ body request | FAILURE-TESTED (đường QUERY của Fuel) |
| I2 | Quyền lọc ngữ cảnh **trước** khi model thấy | FAILURE-TESTED (Fuel) |
| I3 | Chữ của model không cấp được quyền | TESTED một phần (ca chèn lệnh SOP); chưa có đường hành động |
| I4 | Tài liệu truy xuất và đầu ra công cụ ngoài là **dữ liệu**, không phải policy | FAILURE-TESTED (ca chèn lệnh SOP) |
| I5 | Đầu ra model là **đề xuất**; executor chỉ nhận `AuthorizedAction` do phần mềm tất định dựng | NOT_FOUND (chưa có đường hành động) |
| I6 | Policy gắn vào **capability/action**, không gắn vào lối vào (mọi đường vào cùng qua một cổng) | NOT_FOUND cho hành động |
| I7 | Thực thi kiểm lại quyền; ghi có idempotency; báo thành công **sau** khi đọc lại | NOT_FOUND |
| I8 | Thẩm quyền nguồn tính **theo từng fact**, không xếp hạng hệ thống chung chung | FAILURE-TESTED (Fuel) |
| I9 | Fact chính xác đến từ nguồn có thẩm quyền, không từ model | FAILURE-TESTED (Fuel: ràng buộc claim 6 chiều) |
| I10 | Mọi kết quả quan trọng dựng lại được (trace + bằng chứng + phiên bản) | TESTED (phát lại 117/117, chưa lưu bền) |
| I11 | Không có bất biến kiểm soát nào chỉ dựa vào việc model "nghe lời" prompt | chưa kiểm cho hành động |
| I12 | Khái niệm domain không vào core khi chưa có ≥ 2 vertical khác hẳn cùng cần | giữ bằng `architecture_test.go` (tên tenant) |
| I13 | Cổng kiểm soát và cổng chi phí **fail-closed**: phụ thuộc lỗi (policy store, kho quyền, đồng hồ chi phí) ⇒ từ chối, không cho qua. Bài học hệ A: cổng nào cũng "lỗi thì cho qua" | quyền đóng mặc định đã có (Fuel); chưa có test bơm lỗi phụ thuộc |

Thứ bậc tin cậy: SYSTEM POLICY > TENANT POLICY > CAPABILITY/WORKFLOW POLICY > USER REQUEST > RETRIEVED DATA > EXTERNAL TOOL CONTENT.

### I. Bài học từ EAIS V1/V2 (repo này)

**Đã chứng minh, mức TESTED tới FAILURE-TESTED, trên dữ liệu giả lập D2 và 117 ca do phiên không viết mã soạn.** Gồm chuẩn hoá nghĩa, nhận dạng thực thể, thẩm quyền theo nghĩa, ràng buộc claim, định tuyến đếm lời gọi model, RLS tenant, phát lại quyết định. Lệnh và số ở `02`.

Cổng đã bắt được 6 lỗi thật trong ngày 20/10 (`docs/STATUS.md`):
- thứ tự bằng chứng không tất định;
- mất dấu âm;
- slice `nil` thành NULL;
- "bí" khớp "bị";
- `Policy.Allows("")` mở mặc định;
- RAG không tất định.

**Chưa chứng minh:**
- G7 VALUE cũ (model thật, so baseline) đang CHẶN vì thiếu khoá model và đồng ý chi phí;
- tổng quát hoá ngoài 117 ca;
- soát độc lập mã W4–W7;
- HTTP API, xác thực, lưu bền;
- mọi thứ thuộc hành động (COMMAND).

**Bộ ca không bắt hết lỗi.** Hai ví dụ:
- "làm" khớp "làm việc" (ca 116) vẫn đạt (STATUS);
- câu trả lời tổng dự án gắn số lệch của **một máy** mà không nói máy nào (ca 115, phát hiện khi lập kế hoạch này, `02` §4).

Bài học: 117/117 là bằng chứng về **117 ca**, không phải về hệ.

**Bài học quy trình:**
- đột biến là thước đo thật của cổng (~90 lần; mỗi đột biến sống sót đều sinh test mới);
- tách người soạn ca khỏi người viết mã;
- đóng băng ca bằng sha256 trước khi làm.

V3 giữ nguyên cách làm này.

### J. Bài học từ hai hệ tham khảo đã audit

Hai hệ có AI đang chạy đã được audit bằng đọc mã (static). Chi tiết riêng tư không đưa lên repo. Mức bằng chứng của nguồn: **đọc mã, chưa chạy runtime**. Bài học chỉ dùng làm giả thuyết thiết kế.

**Hệ A — AI nhúng sâu trong backend nghiệp vụ.** Assistant/agent gọi công cụ ghi dữ liệu, có duyệt/hoàn tác, truy xuất tri thức, AI tài liệu/giọng nói/hình ảnh, tính hạn mức và credit.
- **Mẫu tốt:** công cụ của agent gọi ngược API REST của chính ứng dụng **bằng danh tính người dùng**, nên phân quyền của ứng dụng áp cả cho AI.
- **Mẫu xấu:** cùng một hành động có **nhiều lối vào**. Lối agent chính đi qua cổng AI (duyệt, chế độ làm việc, ngân sách ghi, danh sách cấm, credit). Các lối song song khác (giao thức công cụ, endpoint thực thi trực tiếp, chat đời cũ) thì không.
- ⇒ **I6: cổng gắn vào capability/action, không gắn vào lối vào.**
- Mã worker có nhưng không được khởi động ⇒ **code tồn tại ≠ capability tồn tại** (E1 ≠ E2).

**Hệ B — AI đặt ngoài backend.** Backend không có SDK model nào. Nó định tuyến câu hỏi tới dịch vụ AI ngoài (bot RAG, workflow engine, webhook đối tác có ký HMAC). Backend vẫn giữ giới hạn tốc độ, hạn mức tháng, ví credit, trạng thái, cầu nối media.
- ⇒ **"Outsourcing intelligence does not outsource governance."**
- Lỗi ở bước xử lý phản hồi AI bị nuốt, không lộ ra người vận hành ⇒ lỗi phải có nhãn và lộ ra (repo đã có nhãn lỗi `Fault`).

**Chung cho cả hai:**
- tiền AI (credit/quota) là tài nguyên nghiệp vụ, thử lại có thể trừ tiền hai lần ⇒ đo chi phí theo **kết quả nghiệp vụ đã xác minh**, không theo token;
- an ninh không phải "module sau này".

**Bổ sung sau khi đối chiếu mã (20/10, mức đọc mã, chưa chạy runtime):**
- **Hệ A:**
  - cổng tiền và cổng hạn mức đều fail-open ⇒ I13;
  - trừ tiền xong mới lưu kết quả, không có luồng hoàn;
  - thử lại không có khoá idempotency;
  - một trường ghi lệch schema bị lối thoát kiểu dữ liệu che đi, nên có thể thu tiền mà mất kết quả;
  - "chạy thử" vẫn thi hành thật một số khối có tác dụng phụ.
- **Hệ B:** lộ dữ liệu theo bot qua các API thiếu kiểm "bot thuộc người gọi". Thuê ngoài trí tuệ, nhưng lỗ nằm ngay ở tầng quyền của chính backend.

### K. Bài học từ First Principles và tư duy "AI là một tech stack"

Nguồn: Blueprint §1.1–1.2, hội thoại V3 §1–3, study guide + workshop notes (bản hội thoại dòng 9.373–14.280; đã tóm, lead kiểm 3 khẳng định khớp nguyên văn).

- **Luồng công cụ production** (dòng 10.798–10.816): đề xuất → kiểm schema → cho phép → luật nghiệp vụ → thực thi → xác minh kết quả → trả quan sát. Đây chính là khung của Job B.
- **"4k" mang bốn nghĩa theo domain** (dầu, tiền, độ phân giải, điện trở; dòng 12.267–12.331). Vì vậy cấm luật chuẩn hoá toàn cục: chuẩn hoá theo tenant → domain → ý định, rồi cổng mơ hồ chọn chấp nhận hoặc hỏi lại. Repo đã làm đúng: thuật ngữ nằm trong pack của tenant (`CLAUDE.md` P6).
- **Nền tảng dùng chung ≠ tri thức dùng chung** (dòng 12.203–12.263): dữ liệu, policy, bộ ca vàng, trace tách theo tenant.
- Các con số trong video ("5%/95%", độ chính xác tool parsing, chuyện model "cố tình overfit benchmark") được chính ghi chú đánh dấu **cần nguồn** (dòng 14.079–14.083). Không dùng làm luận cứ cho EACL.
- **Hợp lệ cú pháp ≠ hợp lệ nghiệp vụ** (Problem Map Part VIII, dòng 18.777–18.817): JSON đúng mà số vô lý vẫn phải bị chặn. Thứ tự kiểm: cú pháp → schema → ngữ nghĩa → luật nghiệp vụ. Đây là V3-AC09 / TASK-R4-03.
- **Control Plane là một tầng riêng** (dòng 16.769–16.844): có RAG, router, tools, MCP, skills, memory mà thiếu tầng kiểm soát thì vẫn chưa phải production. Đây chính là chỗ EACL đứng.
- **Gỡ lỗi theo tầng trước khi đổi model** (cây câu hỏi gỡ lỗi, dòng 19.613–19.656). Khớp `CLAUDE.md` P10.

- Bắt đầu từ **lỗi nghiệp vụ**, không từ RAG, agent, MCP hay model yêu thích. Hỏi: cái gì có thẩm quyền, cái gì phải tất định, bất định nằm ở đâu, bằng chứng nhỏ nhất là gì, bằng chứng nào bác bỏ được kiến trúc.
- AI doanh nghiệp là **kỹ nghệ phần mềm quanh trí tuệ xác suất**: làm sạch dữ liệu trước; luật nghiệp vụ vẫn là phần mềm; ngữ cảnh phải được kiểm soát; model chỉ là một capability; đầu ra phải được kiểm.
- Ví dụ "4k lít dầu": cùng một con số có thể là bốn nghĩa nghiệp vụ khác nhau. DATA ≠ FACT ≠ BUSINESS MEANING ≠ AUTHORITY. LLM OUTPUT ≠ TRUTH.
- Model tốt ≠ hệ tốt; benchmark ≠ chất lượng production.

### L. Phải giữ tất định (D/P/H — Playbook §11)

| Việc | Ai chịu | Ghi chú |
|---|---|---|
| Diễn giải ý định | P | chỉ mở model khi luật tất định trượt có đo (`CLAUDE.md` §19) |
| Phân quyền tenant | D | |
| Tìm ứng viên thực thể | D/P | |
| Chốt thực thể quan trọng | D, hoặc H khi mơ hồ | |
| Viết tóm tắt | P | |
| Kiểm khẳng định | D nếu làm được | |
| Quyền | D | |
| Cho phép hành động | D | |
| Duyệt | H/D | |
| Thực thi | D | |
| Xác minh | D | |

Một task **trượt review kiến trúc** nếu giao ngầm một việc D cho prompt hay model.

### M. AI xác suất đáng dùng ở đâu

Chỉ nơi bất định là bản chất bài toán, và mỗi chỗ phải qua cổng capability (đóng mặc định):
- tổng hợp brief từ hoạt động và tài liệu (Job A);
- diễn giải câu yêu cầu tự do thành đề xuất có cấu trúc (Job B), **chỉ khi** bộ phân tích tất định trượt trên bộ câu đông cứng;
- trả lời câu hỏi quy trình từ văn bản (route RAG/HYBRID của Fuel).

Fact chính xác (đếm task đang mở, lượng dầu xuất kho) **không** dùng model.

### N. Công nghệ hoãn

Không đưa vào khi chưa có lỗi đo được đòi nó (Blueprint §22–24; `CLAUDE.md` §19; `04` §4):
- multi-agent, agent tự trị;
- MCP-first;
- fine-tune;
- GraphRAG, rerank nâng cao, semantic cache;
- workflow engine chung;
- ontology toàn cục;
- chợ connector;
- event bus vì thẩm mỹ;
- DB mới vì thời thượng;
- Kafka/Redis khi chưa có số đo.

Công nghệ đang chạy và cần thiết thì **giữ**: PG17 + RLS, gateway model, RAG tối thiểu.

## 3. Thang trạng thái dùng trong bộ tài liệu này

Theo Playbook §2, cộng thêm hai mức hai đầu:

```text
LEARNED → DEFINED → IMPLEMENTED → REACHABLE → TESTED → FAILURE-TESTED → VERIFIED → VALUE-PROVEN
 (biết)    (có spec)  (có mã)     (nối vào đường chạy thật) (test đúng) (test lỗi/đối kháng) (runtime khớp trạng thái hiện tại) (thắng baseline)
```

Đối chiếu với thang bằng chứng E0–E6 (Blueprint §15):

| Mức | Tương ứng |
|---|---|
| E0 | LEARNED/claim |
| E1 | DEFINED/IMPLEMENTED |
| E2 | REACHABLE |
| E3 | TESTED |
| E4 | VERIFIED trên runtime dev |
| E5 | FAILURE-TESTED có phục hồi |
| E6 | dữ liệu dạng production hoặc phát lại |

Repo còn có thang dữ liệu D0–D4 (`docs/03-proof-harness.md`); Fuel hiện ở D2. Một khẳng định luôn ghi **cả mức trạng thái lẫn mức dữ liệu**. Ví dụ: "G3 TRUTH: FAILURE-TESTED, D2".

## 4. Phát hiện khi đọc nguồn V3

### 4.1 Catalog của Blueprint là khuôn nhân bản

Tin nhắn giao tệp quảng cáo "120 adversarial scenarios, 120 acceptance requirements". Đo bằng script trên bản gốc:

| Mục | Số mục trên giấy | Nội dung thật sự khác nhau |
|---|---|---|
| §42 Scenario | 120 | **10 lớp lỗi** × 12 lần, thân chung chung, không có fixture/đầu vào/kỳ vọng cụ thể. Biến thiên duy nhất: dòng "MVP relevance" (Required / Post-MVP) |
| §43 REQ | 120 | 12 nhóm × 10, câu "must preserve the X invariant for requirement NNN". **Gán cổng G1…G6 xoay vòng** (20 mục mỗi cổng), không theo nghĩa |
| §44 ADR seed | 60 | **12 quyết định** × 5 |
| §45 Learning | 70 | 7 nguồn × 10, chữ y hệt |
| §46 Backlog | 150 | **1 thân duy nhất**, 10 tiêu đề phase × 15 |
| §50 Glossary | — | chỉ có tiêu đề |

Hệ quả:
- Không dùng §42–46 làm bộ ca, truy vết hay backlog. Truy vết V3 dựng lại ở `03`, backlog ở `06`.
- 10 lớp lỗi của §42 và 12 quyết định của §44 **được giữ** (`05` §2.6, §2.7).
- Không chép nguyên tệp 15.131 dòng lên repo. Bản công khai là bản chưng cất (`06`, TASK-R0-04).

### 4.2 Các chỗ lệch khác

Đánh số phase, cổng trùng tên, ngưỡng số bị bỏ trống, tên sản phẩm trong `CLAUDE.md`, sổ 294 REQ "MVP", thực thể Contact… đều ở `05` §3, mỗi chỗ kèm đề xuất xử lý.
