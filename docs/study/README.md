# Mục lục học và debug EACL — từ mindset đến bằng chứng

> **Phiên bản 2.1 — 21/10/2026**, viết cho Luân. Trạng thái: **bản nền để bắt đầu học V0**; hiệu quả học chưa được nghiệm thu. Tài liệu không thay hợp đồng WIN, không phê duyệt triển khai V3 và không yêu cầu sửa code.
> Mốc mã và phép đo của phiên bản 1: `5ee31c8`. Phiên bản 2 bổ sung ba luồng và bài thực hành; phiên bản 2.1 làm rõ phương pháp khám phá, phân loại lỗi và trạng thái nghiệm thu. Hai lần sửa tài liệu này không đo lại runtime. Fuel dùng code EACL thật và dữ liệu giả lập.

## Mục lục

1. [Mục tiêu và giới hạn nguồn](#1-mục-tiêu-và-giới-hạn-nguồn)
2. [Đánh giá hiện trạng trước khi học](#2-đánh-giá-hiện-trạng-trước-khi-học)
3. [MVP WIN của việc học](#3-mvp-win-của-việc-học)
4. [Roadmap V0–V9 và ba luồng xuyên suốt](#4-roadmap-index--10-vòng-foundation)
5. [Index debug theo code đang chạy](#5-index-debug-theo-code-đang-chạy)
6. [WIN kỹ thuật và đường engineering](#6-win-kỹ-thuật-của-repo-và-thứ-tự-engineering)
7. [Format buổi học và bằng chứng](#7-format-cố-định-của-mỗi-buổi-học)
8. [Bắt đầu và phần học sau](#8-thứ-tự-bắt-đầu-và-phần-học-sau)
9. [Những góp ý đã nhận và điều chỉnh](#9-những-góp-ý-đã-nhận-và-điều-chỉnh)

## 1. Mục tiêu và giới hạn nguồn

Mục tiêu: Luân tự giải thích được vì sao hệ trả một câu, tìm đúng tầng sai đầu tiên (first wrong layer), và quyết định phần nào dùng phần mềm tất định, phần nào cần AI.

Có hai đối tượng cần hiểu:

| Đối tượng | Câu hỏi nó giúp trả lời | Giới hạn |
|---|---|---|
| Mindset được bốn bản trao đổi gán cho QuânIT | Bài toán nghiệp vụ là gì? AI nên tham gia ở đâu? Dữ liệu, context, phần mềm và model chia việc thế nào? | Chưa có nguồn QuânIT gốc được đối chiếu trong phiên này. Gọi là **mindset tái dựng**, không trích như lời QuânIT. |
| EACL hiện có | Các nguyên tắc được cài bằng code thế nào? Hệ sai ở đâu? Cổng kiểm có phát hiện được không? | Bản cài đặt tham chiếu trên Fuel; không đại diện cho toàn bộ nghiệp vụ doanh nghiệp hay toàn bộ AI. |

Quan hệ giữa hai bên: **mindset đưa giả thuyết → code cho thấy một cách hiện thực → thử nghiệm xác nhận, sửa hoặc bác giả thuyết**. Không thêm component chỉ để làm giống sơ đồ.

Luôn tách ba câu: **kiến trúc khái niệm muốn có gì; implementation hiện cài gì; bằng chứng đã xác minh được gì**. Một khái niệm xuất hiện trong roadmap không có nghĩa đã có component hoặc đã được kiểm chứng.

Ba hệ ký hiệu có mục đích riêng: **V0–V9** tổ chức việc học; **S0–S10** định vị bước xử lý request hiện tại; **W-G1–W-G7** quy định bằng chứng kỹ thuật của Fuel. Hoàn thành một vòng học không tự đóng một cổng kỹ thuật; test một step đạt cũng không nghiệm thu năng lực của người học.

## 2. Đánh giá hiện trạng trước khi học

### Những gì code hiện có

- `core/pipeline/ask.go`: luồng hỏi, chọn route trước khi thực thi truy xuất; trả số liệu bằng template, gọi model cho phần tri thức; kiểm trước khi giao.
- `data/normalization`, `data/entityresolution`: đọc cách viết tiếng Việt, giá trị có kiểu, bí danh và mơ hồ bằng luật.
- `data/ingestion`: đọc CSV, XLSX, JSON, JSONL, PDF văn bản; trả observation hoặc quarantine. Nguồn chữ tự do dùng lại normalizer và resolver của luồng hỏi.
- `data/sourceoftruth`: tách chỉ số nghiệp vụ, gộp theo nguồn, áp thẩm quyền trong cùng chỉ số; tính tổng/hiệu và giữ evidence.
- `intelligence/rag`: retrieval theo từ khóa với điểm IDF trên các dòng tài liệu; lọc tenant và ACL. Hiện không phải vector RAG hay learned reranker.
- `platform/pgstore`: schema, transaction và PostgreSQL RLS để cách ly tenant; đây là cơ chế lưu trữ, không phải dịch vụ xác thực người dùng hoàn chỉnh.
- `evals`, bộ sinh Fuel và `g7run`: ca thử, manifest, baseline, bộ chấm, trace và bằng chứng chạy.

### Những điểm phải giữ tỉnh táo

1. **Phần hiểu ngôn ngữ hiện chủ yếu là luật.** Keyword, mức ưu tiên, slot, phạm vi mệnh đề và mặc định của pack quyết định cách hiểu; không có model intent classifier trong luồng hiện tại.
2. **Nghĩa dữ liệu do source mapping gán.** `Metric: c.spec.Metric` trong `data/ingestion/ingest.go`: trích được máy và lượng chưa chứng minh đã hiểu “đã cấp”, “mai cần” hay “báo nhầm”.
3. **Fact trong code không mặc nhiên là nguồn thắng cuối cùng.** `Result.Facts()` chuyển observation đã nhận sang fact của từng nguồn; `Catalog.Query()` mới áp authority và hòa giải lúc truy vấn. Phân biệt fact theo nguồn với kết quả có thẩm quyền.
4. **Validator có giới hạn.** `Bound()` kiểm id evidence có trong context; nó chưa chứng minh toàn bộ câu văn được evidence suy ra. `BindClaims()` kiểm cấu trúc sáu chiều, nhưng không phải bộ hiểu mọi mệnh đề ngôn ngữ.
5. **Ask chưa có hành động ghi nghiệp vụ.** Agent, Tool, Workflow có tên trong contracts; switch thực thi hiện báo lỗi với route chưa nối. `HUMAN_REVIEW_REQUIRED` không tự tạo việc cho người duyệt.
6. **Test xanh không đồng nghĩa WIN.** Test ratchet bảo vệ các ca đã đạt và ghi nhận nợ còn tồn tại; không bắt mọi ca nghiệp vụ hiện có phải đúng.
7. **Chưa có kết luận hơn baseline bằng model thật.** `scorerFrozen = false` trong `g7run/main.go`; lượt chính bị chặn. Bộ chấm v2 và điều kiện benchmark còn phải hoàn tất.

### Bằng chứng đã chạy ở phiên bản 1

Các kết quả dưới đây là phép đo ngày 21/10/2026 của phiên bản 1, tại mốc mã nêu đầu tài liệu. Khi code, pack, dữ liệu hoặc môi trường đổi, chạy lại trước khi dùng chúng để kết luận về hiện trạng mới.

```bash
GOMAXPROCS=2 go test -p 1 ./... -count=1
```

Lệnh kết thúc với mã 0. Điều đó chứng minh các assertion hiện hành đạt trong môi trường phiên này, không chứng minh toàn bộ tiêu chí WIN đạt.

Kiểm riêng bộ chấm:

```bash
GOMAXPROCS=2 go test -p 1 ./evals -run TestG7ScorerAttacks -v -count=1
```

Đầu ra của test trong phiên:

```text
170 đòn; 118 còn chấm sai (nợ đã ghi); 8 tranh cãi chưa so
--- PASS: TestG7ScorerAttacks
```

Đây là test giữ nguyên danh sách lỗi đã biết, không phải tuyên bố bộ chấm đúng. Các test PostgreSQL kiểm riêng đã SKIP vì thiếu `EACL_PG_ADMIN_DSN` / `EACL_PG_DSN` (tên hiện hành; lúc viết là tiền tố cũ); phiên này chưa xác minh runtime RLS. Không gọi model trả phí, không chạy lại holdout chưa dùng để phục vụ việc học.

## 3. MVP WIN của việc học

**Bài thực hành xuyên suốt:** “Hôm qua máy 03 được cấp 4k dầu đúng không?” Sau đó đổi cách viết, nghĩa nghiệp vụ, nguồn, kỳ và quyền để xem kết quả phải đổi ở đâu.

WIN học tập đạt khi Luân làm được các việc sau trên một ca mới, không xem đáp án trước:

| Mã | Kết quả quan sát được |
|---|---|
| L1 — Hiểu job | Nói rõ người dùng muốn biết/làm gì và phân biệt issued, consumed, reported, billed. |
| L2 — Giải thích dữ liệu | Phân biệt record, observation, fact theo nguồn, kết quả có thẩm quyền và derived fact; truy được về nguồn. |
| L3 — Đọc trace | Đi từ câu hỏi qua normalize, entity, intent, route, evidence, validation đến answer; chỉ ra giá trị đầu vào/đầu ra cần kiểm. |
| L4 — Chẩn đoán | Tìm first wrong layer; giải thích vì sao một tầng sau chạy đúng vẫn tạo câu trả lời sai. |
| L5 — Thiết kế phép thử | Viết kỳ vọng độc lập với implementation; có ca đúng, ca sai và ca mơ hồ; phân biệt lỗi sản phẩm với lỗi scorer. |
| L6 — Chọn giải pháp | Đề xuất cách nhỏ nhất đủ giải lỗi, giải thích khi nào luật/SQL đủ, khi nào uncertainty thực sự cần AI. |
| L7 — Biết giới hạn kết luận | Tách kết quả tự đo, số repo báo, suy luận và giả thuyết; không dùng test xanh để tuyên bố WIN kỹ thuật. |

Đây là **tiêu chí học đề xuất**, không phải các cổng đã được dự án chốt. Cách kiểm: cuối lộ trình giao một ca mới, Luân tự lập phiếu debug rồi giải thích từng quyết định.

**Cách nghiệm thu:** dùng ca luyện tập mới, tách khỏi holdout kỹ thuật của dự án. Người hướng dẫn đưa tình huống và dữ liệu trước, giữ đáp án để đối chiếu sau. Mỗi L1–L7 ghi **tự làm được / cần gợi ý / chưa làm được** cùng sản phẩm cụ thể của Luân. WIN học tập chỉ ghi nhận khi cả bảy mục tự làm được; không lấy việc đọc hết bài hoặc nhớ tên layer làm bằng chứng. Đây là rubric học đề xuất, không phải kết quả đã đo.

## 4. Roadmap index — 10 vòng foundation

Mỗi vòng cùng nhìn hai phía: dữ liệu đi vào và yêu cầu người dùng. Đồng thời phân biệt **structured lookup** (tra cấu trúc đã biết) với **language interpretation** (diễn giải ngôn ngữ), vì khó khăn ngôn ngữ xuất hiện ở cả hai phía.

### Ba luồng để hiểu hệ thống

Thứ tự V0–V9 là thứ tự học; runtime không phải một pipeline duy nhất. Ba luồng dưới đây là bản đồ khái niệm, không yêu cầu mỗi mũi tên thành một service hay một bước code riêng.

**A — World → Knowledge: doanh nghiệp để lại dấu vết gì?**

```text
Reality → Business event/process → Observation mechanism → Source record
→ diễn giải: representation + entity + semantics + time + provenance
→ fact theo nguồn → kho dữ liệu
```

Authority/reconciliation xác định kết quả có thẩm quyền khi cần. Trong implementation Fuel hiện tại, công việc này chạy ở `Catalog.Query()` lúc truy vấn; không giả định kho đã chứa toàn bộ authoritative facts được tính sẵn. Những thành phần trong khối “diễn giải” có thể phụ thuộc lẫn nhau, không phải thứ tự cứng cho mọi nguồn.

**B — User → Need: người dùng cần biết hoặc làm gì?**

```text
User job → Utterance → Normalize → Entity → Intent + requested semantics
→ Time / scope → yêu cầu dữ kiện hoặc công việc cần giải
```

A và B được nối bằng năng lực do hệ lựa chọn. Với Fuel hiện có:

```text
Yêu cầu + principal + policy
→ S5 ROUTER chọn capability hoặc hỏi lại/từ chối
→ S6 SQL / calculation / retrieval đọc kho dữ liệu
  (nhánh số liệu áp authority/reconciliation tại đây)
→ S7 evidence/context → S8 template/model → S9 kiểm → S10 trả lời
```

“Fact selection” là khái niệm chọn dữ kiện phù hợp yêu cầu; không đặt nó thành một bước truy xuất đã hoàn tất trước router. Kiểm chất lượng ingestion và kiểm cách hiểu request là hai việc khác nhau: một bên đúng không làm bên kia tự đúng.

**C — Decision → World: kết quả thực thi có thực sự xảy ra?**

```text
Candidate/proposal → Validate + authorize + business rules
→ Execute có kiểm soát → World changes
→ Observe/read-back → Verify expected state → Outcome + audit
```

Đây là luồng khái niệm dành cho action, chưa được nối trong Fuel Ask. Read-back xác minh trạng thái quan sát được theo hợp đồng; không tự chứng minh mọi tác động ngoài đời đã xảy ra. Việc “đã chuyển người phụ trách” cũng cần chứng từ giao việc nếu muốn tuyên bố có hành động thật.

### Mục lục từng vòng

| Vòng | Chủ đề / câu hỏi gốc | Đọc code và pack | Bài thực hành / điều cần chứng minh |
|---|---|---|---|
| **V0** | **Business Reality & Job** — ta đang giải công việc nào? | `tenants/construction/win/world.yaml`, `pack/metrics.yaml`, `pack/policy.yaml` | Giải thích vì sao kho xuất, cảm biến, báo cáo và hóa đơn có thể khác số mà không mâu thuẫn; ghi giả định cần người nghiệp vụ xác nhận. |
| **V1** | **Raw Record & Observation** — ta thực sự biết nguồn đã ghi gì? | `data/ingestion/readers.go`, `ingest.go`, `bridge.go`, `pack/sources.yaml` | Theo một bản ghi từ nguồn đến observation/quarantine; phân biệt “nguồn nói X” với “X xảy ra”; kiểm tập id, trùng id và giao hai tập. |
| **V2** | **Normalization & Representation** — đổi cách viết có giữ nghĩa? | `data/normalization/`, `core/contracts/contracts.go`, `pack/lexicon.yaml` | Mổ `máy 03`, `3 850 lít`, `sửa chữa 500 000 đồng`; đọc span, kiểu, đơn vị, tọa độ và rule; đo hành vi hiện tại, không mặc định lỗi cũ vẫn còn. |
| **V3** | **Entity & Identity** — đang nói về đối tượng nào? | `data/entityresolution/resolver.go`, `data/ingestion/bridge.go`, `intelligence/router/router.go` | Đối chiếu EX-03, EX003, asset-8392, Machine 03; thử bí danh mơ hồ, dự án cha và phạm vi người hỏi; không chọn bừa. |
| **V4** | **Business Semantics & Request Meaning** — muốn chỉ số nào, hay muốn xác nhận điều gì? | `intelligence/intent/intent.go`, `pack/intents.yaml`, phần `claimNumbers` của `ask.go` | Giữ nguyên máy/ngày, đổi “cấp/đốt/báo/tính tiền”; phân biệt tra số với xác nhận; phát hiện keyword đúng nhưng ý nghĩa sai. |
| **V5** | **Time, Authority & Truth** — đúng theo nghĩa nào, nguồn nào và kỳ nào? | `data/normalization/datetime.go`, `data/sourceoftruth/{catalog,query}.go`, `pack/metrics.yaml` | Cùng chỉ số mới so authority; ngang hạng lệch thì chưa chọn; tách ngày gửi và ngày sự kiện; ghi rõ correction/bitemporal chưa được giải đầy đủ. |
| **V6** | **Intent → Route → Retrieval → Context** — cần năng lực và bằng chứng nào? | `intelligence/router/router.go`, `intelligence/rag/rag.go`, `platform/pgstore/pgstore.go`, `Ask.Handle` | Vẽ route trước truy xuất; phân biệt SQL, calculation, RAG và hybrid; phát hiện thiếu bằng chứng với lấy nhầm bằng chứng. |
| **V7** | **Model & Deterministic Control** — AI được làm phần nào? | `intelligence/models/`, `core/pipeline/ask.go`, `reliability/validation/` | Đếm lời gọi model; số liệu trả bằng template; phá draft, đổi evidence id, chèn số; giải thích validator chặn được gì và chưa bảo đảm gì. |
| **V8** | **Permission, Action & Verification** — ai có quyền, và làm xong kiểm thế nào? | `router.Policy`, `db/migrations/0001_win.sql`, `platform/pgstore/`, switch route của `ask.go` | Trên Fuel: kiểm allow/deny, scope, ACL và tenant. Action: chỉ phân tích ranh giới còn thiếu; read-back/idempotency là bài mở rộng sau foundation, chưa gọi là đã có. |
| **V9** | **Evaluation, Baseline & Value** — làm sao biết hệ đúng và đáng tồn tại? | `evals/`, `winrun/`, `g7run/`, `scripts/` | Phân biệt unit test, E2E, ratchet, holdout, mutation và benchmark; đọc lỗi scorer; giải thích điều kiện mới được kết luận hơn B1/B2+. |

V6 trình bày router, retrieval và context trong cùng một vòng để không dạy sai thứ tự thực thi. Intent đã bắt đầu ở V4; các khái niệm quay lại khi cần, không coi mỗi tầng chỉ được xuất hiện một lần.

### V0 — năm câu bắt buộc và sản phẩm đầu ra

1. Ngoài đời có sự kiện hoặc trạng thái nào, và ta thực sự quan sát được phần nào?
2. Quy trình nghiệp vụ nào tạo ra bản ghi đó?
3. Ai sở hữu/chịu trách nhiệm với quy trình và dữ liệu của nó?
4. Hệ thống hoặc cơ chế nào ghi nhận, với sai số và độ trễ nào?
5. Người dùng muốn biết/quyết định/làm gì từ thông tin đó?

Ví dụ: máy nhận nhiên liệu → nghiệp vụ xuất kho → phiếu xuất; máy vận hành → cơ chế đo cảm biến → bản ghi tiêu thụ. Hai record mô tả hai khía cạnh khác nhau; việc đặt nhãn `issued`/`consumed` chỉ biểu diễn khác biệt nghiệp vụ đó. Không được suy sự kiện đã xảy ra chỉ vì có một phiếu ghi nó.

**Đầu ra V0:** Luân lập bảng `event/process → owner → observation mechanism → source record → metric → user job`, kèm giả định và người cần xác nhận. Ghi rõ trách nhiệm nghiệp vụ là cơ sở đề xuất authority; cần thêm phạm vi, thời gian hiệu lực, chất lượng và chính sách được chấp nhận. “Owner” không tự động đồng nghĩa “mọi dữ liệu đều đúng”.

### V5 — khám phá “truth” từ code và evidence

**Bài thực hành trước khi đọc bảng:** chọn một ca luyện tập có hai nguồn cùng chỉ số và một nguồn khác chỉ số. Luân theo dấu record qua `Result.Facts()` đến `Catalog.Query()`, ghi nguồn nào khẳng định gì, lúc nào policy chọn/gộp kết quả, rồi đổi nghĩa hoặc kỳ của câu hỏi. Dự đoán trước, chạy sau, đối chiếu kết quả; không dùng holdout còn sạch cho bài này.

Bảng dưới là **khung thuật ngữ để đối chiếu sau bài thực hành**, không phải ba kiểu dữ liệu bắt buộc hay kết luận người học phải lặp lại. Nếu hành vi đo được không khớp, ghi điểm lệch và sửa cách hiểu; không sửa code chỉ để khớp bảng.

| Khái niệm | Nghĩa dùng trong bài học | Ánh xạ với Fuel |
|---|---|---|
| **Source fact** | Khẳng định đã có kiểu, entity, metric, kỳ và provenance của một nguồn; chưa bảo đảm đúng với reality. | `sourceoftruth.Fact`, chuyển từ observation được nhận qua `Result.Facts()`. |
| **Authoritative fact/result** | Khẳng định hoặc kết quả được chấp nhận theo authority policy trong phạm vi xác định. Có thể là tổng đã hòa giải, không chỉ fact nguyên tử. | `Catalog.Query()` chọn/gộp theo nguồn; có xung đột ngang hạng thì chưa trả số có thẩm quyền. |
| **Answer truth** | Câu trả lời trung thực với yêu cầu đã hiểu đúng, dựa trên evidence phù hợp nghĩa, entity, kỳ, mức gộp và quyền của người hỏi. | Kết quả sau intent, routing, execution, generation và validation; kiểm từng cơ chế hiện có, không coi S9 là bảo đảm toàn bộ chân lý. |

Ví dụ giả lập: kho ghi issued = 3.850 L; policy chấp nhận kho cho issued; người có quyền hỏi “được cấp bao nhiêu?” thì 3.850 L là dữ kiện phù hợp. Nếu họ hỏi consumed, cùng số đó không còn trả lời đúng chỉ vì nó có evidence. Thiếu bằng chứng hoặc không đủ quyền thì câu trả lời trung thực có thể là thiếu dữ liệu, hỏi lại hoặc từ chối.

Câu hỏi không tạo ra sự kiện nguồn. Nó xác định dữ kiện, kỳ và phép gộp cần thiết; kết quả dẫn xuất có thể được tính khi hỏi, hoặc được tính sẵn nếu thiết kế khác cần. **Authority là chính sách chấp nhận có phạm vi, không phải chứng minh tuyệt đối về reality.**

## 5. Index debug theo code đang chạy

**Luồng nạp**

```text
Source mapping → Reader → Record
→ tra mã có cấu trúc / normalize + resolve chữ tự do
→ lượng, đơn vị, ngày, metric từ cấu hình nguồn
→ kiểm miền → Observation hoặc Quarantine
→ Facts theo nguồn → Store
```

**Luồng hỏi**

| Step | Kiểm gì trước khi đi tiếp? | Nếu sai, tìm ở đâu? |
|---|---|---|
| S0 ACCEPT | Text, độ dài, id yêu cầu hợp lệ? | `validateRequest` |
| S1 TENANT | Principal thuộc đúng runtime? Ai chịu trách nhiệm xác thực principal trước khi gọi? | `Ask.Handle`, `contracts/tenantctx.go`; không coi struct Principal là cơ chế đăng nhập |
| S2 NORMALIZE | Span, kiểu, đơn vị, ngày và dấu mơ hồ đúng? | `data/normalization/` |
| S3 RESOLVE | Đúng entity, đúng tập ứng viên, đúng vị trí cụm tên? | `data/entityresolution/` |
| S4 INTENT | Đúng nghĩa, subject, period? Có nhiều ý hay phủ định/xác nhận chưa hiểu? | `intelligence/intent/`, `pack/intents.yaml` |
| S5 ROUTE | Đúng capability? Có quyền? Cần hỏi lại hay từ chối? | `intelligence/router/` |
| S6 EXECUTE | Đọc đúng metric/entity/kỳ? Authority, aggregation, phép tính và retrieval đúng? | `sourceoftruth/`, `rag/`, `pgstore/` |
| S7 CONTEXT | Bằng chứng đủ và được phép đọc? Nguồn nào bị loại, vì sao? | `answerMetric`, `answerKnowledge`; chưa giả định có context engine độc lập đầy đủ |
| S8 GENERATE | Template hay model? Draft thêm điều gì ngoài bằng chứng? | `ask.go`, `intelligence/models/` |
| S9 VALIDATE | Số, claim, nguồn, kỳ, đơn vị khớp? Có điểm mù ngôn ngữ? | `reliability/validation/` |
| S10 RESPOND | Câu giao, Outcome, citations và decision thống nhất? | `Ask.Handle`, `observability/traces/` |

Các step là bản đồ logic; code không nhất thiết có một hàm hay một timing span riêng cho mọi step.

**Cách tìm lỗi:** xác định nguồn của kỳ vọng → chạy ca → so đầu vào/đầu ra từng step → định vị sai lệch đầu tiên. Kiểm cả hợp đồng hiện hành lẫn nhu cầu nghiệp vụ; ghi rõ đang kết luận theo cái nào.

| Tình huống | Phân loại và cách xử lý |
|---|---|
| Hợp đồng/pack đã chốt, implementation làm khác | Lỗi tuân thủ hợp đồng có thể kiểm bằng reproducer. Ví dụ pack quy định “được cấp” → issued nhưng classifier chọn consumed. |
| Implementation đúng pack, người nghiệp vụ cần nghĩa khác | Vấn đề mức phù hợp của policy/requirement. Ghi đề xuất đổi hợp đồng, không âm thầm sửa expectation để hợp thức hóa code. |
| Chưa có kỳ vọng, hoặc các nguồn có thẩm quyền quy định mâu thuẫn | Definition/policy chưa rõ. Làm rõ với người chịu trách nhiệm trước khi phán quyết ca đó đúng/sai theo nghiệp vụ. |
| Scorer cho kết quả khác expectation đã có căn cứ | Kiểm lỗi bộ chấm riêng; không mặc định câu trả lời sai chỉ vì scorer báo trượt, hoặc đúng vì scorer cho đạt. |

Phiếu debug giữ cả **expected theo hợp đồng hiện hành** và **nhu cầu nghiệp vụ cần xác nhận** nếu chúng khác nhau. Code và pack là evidence về implementation/chính sách đang chạy, không tự là chứng minh chính sách phù hợp ngoài đời.

Sau khi xác định tầng sai đầu tiên, dừng việc đoán bản vá và tìm phép thử có thể bác chẩn đoán đó. Các lỗi sau vẫn được ghi lại nếu có evidence độc lập; sửa tầng đầu không bảo đảm mọi lỗi khác biến mất. Với dữ liệu đã sai từ ingestion, quay về flow A thay vì bắt đầu và kết thúc chẩn đoán trong S0–S10.

Ví dụ: S2 đọc đúng lượng, S3 nhận đúng máy, S4 hiểu “đốt” thành issued → S5 chọn SQL đúng với intent sai → S6 trả đúng số issued → S9 thấy đủ evidence. **Câu cuối vẫn sai với người hỏi; validator không sửa được cách hiểu yêu cầu sai.**

## 6. WIN kỹ thuật của repo và thứ tự engineering

Nguồn chuẩn: [hợp đồng WIN Fuel](../04-core-mvp.md), [ADR-0005](../adr/0005-g7-hop-dong-va-sua-d5.md), [ADR-0007](../adr/0007-bo-cham-g7-v2.md). Tài liệu mới gọi cổng Fuel là **W-G1…W-G7** để không lẫn với G1…G6 của V3.

| Cổng | Phải chứng minh |
|---|---|
| W-G1 DATA | Bản ghi được xử lý đầy đủ, biểu diễn tương đương giữ kết quả; không mất im lặng. |
| W-G2 IDENTITY | Đúng entity; mơ hồ được xử lý đúng, không đoán im lặng. |
| W-G3 TRUTH | Đúng nghĩa, authority, dung sai, aggregation và calculation. |
| W-G4 DECISION | Đúng route; đếm được lời gọi model, đường tất định không gọi model. |
| W-G5 CONTROL | Quyền/tenant được bảo vệ; đầu ra không được giao trái hợp đồng kiểm. |
| W-G6 PROOF | Ca thử, trace, replay, kiểm độc lập và mutation theo hợp đồng. |
| W-G7 VALUE | So với baseline bằng model thật, cùng điều kiện, scorer đáng tin và evaluation set hợp lệ. |

WIN Fuel theo hợp đồng là **đủ bảy cổng và scale smoke**; tài liệu học này không tự đổi ngưỡng hoặc tuyên bố cổng nào đã đạt.

Thứ tự engineering đề xuất, chạy độc lập với tốc độ học:

Learning và engineering chia sẻ evidence: learning dùng trace và reproducer để hiểu; engineering dùng kỳ vọng nghiệp vụ và invariant đã làm rõ để sửa. Tài liệu học không phải lệnh tự động triển khai hay quyền chạy benchmark trả phí.

1. Chốt expectation và baseline đo; phân loại số tự đo với số repo báo. Không dùng holdout còn sạch làm bài tập mở.
2. Gom failure theo first wrong layer; tách lỗi definition, code, data và scorer.
3. Với mỗi nhóm, dựng reproducer và invariant; sửa theo quy trình được duyệt, tránh chuỗi keyword/exception chỉ để xanh ca đã thấy.
4. Kiểm hồi quy và negative case; chứng minh cổng phát hiện được lỗi cần chặn.
5. Hoàn thiện scorer v2, evaluation set sạch, contract và kế hoạch chi phí/ủy quyền pilot trước benchmark trả phí.
6. Chạy benchmark khi đủ prerequisite; giữ nguyên bằng chứng và báo cả kết quả bất lợi.

## 7. Format cố định của mỗi buổi học

1. **First principle:** vấn đề buộc khái niệm này phải tồn tại.
2. **Definitions:** định nghĩa và chỗ dễ nhầm.
3. **Data/world side:** một ví dụ ở luồng nạp.
4. **Question/user side:** một ví dụ ở luồng hỏi.
5. **EACL reality:** code, phép đo và giới hạn bằng chứng.
6. **Failure modes:** first wrong layer, điều kiện tái hiện và invariant.
7. **Mental model update:** CONFIRM / REFINE / REJECT / OPEN QUESTION.
8. **Luân tự reasoning:** hai đến bốn câu, chưa đưa đáp án trước.

Trong phần 3–4, ghi vòng này đang chạm flow A, B hay C và cần tra cấu trúc hay diễn giải ngôn ngữ. Phần 5 luôn phân biệt conceptual / implemented / verified. Mỗi buổi kết thúc bằng một sản phẩm nhỏ: bảng V0, bảng span V2, bảng chọn metric V4 hoặc phiếu debug; ghi điều Luân đã tự giải thích được và điểm còn mở trước khi chuyển vòng.

**Trình tự dạy:** đưa tình huống và evidence → Luân dự đoán, tự giải thích → đọc/chạy code → đối chiếu định nghĩa → cập nhật mental model. Tám mục trên là khuôn ghi bài; người hướng dẫn không cần đọc hết definitions hoặc đưa đáp án trước bài tập. Có thể nhắc lại kiến thức cần thiết, nhưng ghi nhận rõ khi người học cần gợi ý.

Nhãn bằng chứng:

- **[MEASURED]:** tự chạy trong phiên, ghi lệnh, revision và kết quả.
- **[CODE]:** trực tiếp đọc thấy cấu trúc/nhánh trong code, kèm vị trí; chưa chạy tình huống đó.
- **[REPO]:** tài liệu hoặc báo cáo có sẵn tự ghi; chưa chạy lại.
- **[INFERRED]:** suy hành vi từ code/evidence, ghi điều kiện cần.
- **[HYPOTHESIS]:** giả thuyết chưa chứng minh và cách thử.

Phiếu debug tối thiểu: câu hỏi gốc · principal/scope · đồng hồ · dữ liệu/pack/code version · kỳ vọng và lý do nghiệp vụ · kết quả thực tế · step sai đầu tiên · evidence · invariant · phép thử có thể bác chẩn đoán.

## 8. Thứ tự bắt đầu và phần học sau

**Bắt đầu V0 rồi hoàn thiện V1.** Hai vòng này chốt nghĩa dữ liệu và công việc; sang V2 mới mổ parser. Những lỗi kể trong trao đổi cũ phải được chạy lại ở revision đang học trước khi gọi là lỗi hiện hành.

Giữ mục lục hiện tại ổn định trong lúc học. Chỉ sửa khi có evidence: khái niệm gây hiểu nhầm, bài tập không kiểm được năng lực, code/hợp đồng thay đổi hoặc nguồn gốc được xác minh thêm. Nhận xét “đủ tốt” là ý kiến review; năng lực L1–L7 chỉ được ghi đạt sau bài thực hành. Việc chính thức đóng băng curriculum cần một quyết định riêng của Luân.

Sau V9, chỉ mở khi có failure/nhu cầu đo được: advanced retrieval, model extraction/intent, memory, controlled action, agent, MCP, fine-tuning, multi-agent và framework extraction. Không thêm chúng vào foundation chỉ vì tên công nghệ phổ biến.

Với bài toán thật của Luân, kết quả chuyển giao cuối cùng là: lấy một nguồn dữ liệu và một câu hỏi thực tế, ghi các giả định nghiệp vụ, lập phiếu debug và chọn một lát ứng dụng AI nhỏ có tiêu chí thành công kiểm được. Fuel là nơi luyện phương pháp; nghiệp vụ thực cần người chịu trách nhiệm xác nhận lại semantics và authority.

## 9. Những góp ý đã nhận và điều chỉnh

| Góp ý | Xử lý trong bản hiện tại | Lý do |
|---|---|---|
| Giữ ba lớp mindset / implementation / evidence | Giữ và nêu rõ conceptual / implemented / verified. | Tránh dùng sơ đồ hay tên component thay bằng chứng. |
| Dùng ba flow xuyên curriculum | Nhận, kèm trạng thái implementation của từng flow. | Tránh hiểu thứ tự học là một runtime pipeline duy nhất. |
| Relevant facts → Router | Điều chỉnh thành requirement → router → truy xuất/hòa giải → evidence. | Khớp `Ask.Handle`, `answerMetric` và `Catalog.Query()` đã đọc ở phiên bản 1. |
| V0 đào sâu process và owner | Nhận năm câu hỏi, thêm bảng đầu ra và giả định cần xác nhận. | Hiểu vì sao record tồn tại trước khi học cách parse. |
| Source Fact → Authoritative Fact → Answer Truth | Nhận, mở rộng thành authoritative fact/result có phạm vi policy. | Khớp việc Fuel hòa giải lúc truy vấn; không đồng nhất authority với reality. |
| Freeze curriculum, bắt đầu học | Giữ V0–V9 làm bản nền ổn định để bắt đầu V0. | Chỉ sửa khi học hoặc thực nghiệm lộ thiếu sót; đây chưa phải quyết định đóng băng của chủ dự án. |
| Viện dẫn kiến trúc bên ngoài để xác nhận hướng | Không dùng làm bằng chứng trong bản này. | Các liên kết trong góp ý chưa được kiểm ở phiên sửa; sự tồn tại của kiến trúc tương tự không chứng minh EACL đúng hoặc tạo giá trị. |
| Để V5 tự khám phá thay vì học thuộc kết luận | Đưa bài thực hành trước, giữ bảng thuật ngữ để đối chiếu sau. | Phân biệt khung tham khảo với đáp án; không cần đổi mục lục. |
| Nhận xét mới nói bản được đọc vẫn có 188 dòng | Không dùng nhận xét đó để xác nhận đã review đúng V2. | Phiên đối chiếu trực tiếp đếm V2 có 294 dòng và thấy các phần bổ sung; chưa biết bản nào đã được gửi đến người review. |
| Chấm PASS rồi FREEZE curriculum | Giữ trạng thái bản nền để bắt đầu học; chưa tuyên bố nghiệm thu. | Ý kiến review không thay bài thực hành L1–L7 hoặc quyết định của Luân. |

Bước sử dụng tiếp theo là thực hiện V0 với bảng đầu ra ở mục 4, ghi phần Luân tự giải thích được và các giả định chưa xác nhận, rồi dùng chúng để dẫn sang V1.
