# Học hệ cũ và viết lại đúng ranh giới EACL

> Ngày 21/10/2026. Hướng dẫn học và chuẩn bị migration, không phải lệnh triển khai.
> Phân biệt ba mốc: dịch vụ answer cũ → Fuel hiện có (tên sản phẩm cũ, trước 21/10/2026) → hướng EACL tích hợp app nghiệp vụ.
> Chưa xác định checkout/revision của dịch vụ answer cũ trong phiên này. Các đường dẫn `internal/...`, `ingest/...` bên dưới lấy từ audit lịch sử, chưa được kiểm trực tiếp trên code gốc.

## 1. Phân tích CLAUDE.md và bản góp ý

**[MEASURED]** `wc -l CLAUDE.md` tại mốc khảo sát `5ee31c8` trả **1274** dòng. Bản góp ý nói 2.549 dòng; không dùng con số đó để xác nhận đã đọc đúng phiên bản.

**[CODE/tài liệu]** Lúc viết, root định nghĩa sản phẩm (tên cũ) là một nền tảng AI đa dụng (§9), còn D-V3-00 trong `docs/CHO-QUYET-DINH.md` đã chọn EACL Control Layer. Đây là mismatch về product ownership, không chỉ tên gọi. Từ commit `3c51bb0` (21/10/2026), `CLAUDE.md` §9 đã dùng định nghĩa EACL.

| Nhận xét | Đánh giá / cách xử lý |
|---|---|
| Giữ engineering discipline | Giữ evidence, independent review, simple first, surgical changes, negative tests, scope và DoD. Không rút gọn đến mức mất security hoặc cổng người duyệt. |
| Đổi platform thesis sang EACL | Có căn cứ từ D-V3-00; app nghiệp vụ giữ truth, quyền, state, transaction và workflow. |
| CURRENT/TARGET/HYPOTHESIS | Cần. Một boundary bắt buộc không có nghĩa component đã triển khai hoặc invariant đã proven. |
| “Probabilistic core” | Đổi thành probabilistic capability; có request hoàn toàn tất định. |
| Context Engine | Chuyển trọng tâm sang thuộc tính trusted context; không tự thêm module/service. |
| Action/read-back | Giữ như ranh giới TARGET, ghi rõ Fuel hiện chỉ QUERY, không tạo workflow ghi nghiệp vụ. |
| Authority từ business owner | Bổ sung phạm vi/hiệu lực/chất lượng. Owner không tự bảo đảm record đúng reality. |
| Pack và Go là luật EACL chung | Điều chỉnh: cơ chế Fuel và implementation policy của repo; khả năng dùng lại phải được chứng minh. |
| Sơ đồ authority/fact trước capability | Không dùng làm runtime Fuel. Router chọn capability trước truy xuất; `Catalog.Query()` hòa giải khi chạy. |
| Scores 9/10, 70–80% đáng giữ, nhãn P0 | Ý kiến của tác giả, không có rubric/phép đo đủ để coi là bằng chứng chất lượng hoặc severity sự cố production. |
| Agent chỉ được đề xuất sửa CLAUDE | Tài liệu nháp R0 ghi như vậy, nhưng cột trả lời D-V3-06 cho phép sửa có điều kiện: giữ luật còn cần, diff rõ, commit riêng, compatibility note. |

Root cũ còn trộn operating rules, product definition, architecture boundaries và handbook. Bản thay thế đề xuất nằm ở [CLAUDE-EACL-DRAFT.md](CLAUDE-EACL-DRAFT.md); root chưa đổi.

## 2. Học ba thế hệ bằng câu hỏi, không bằng danh sách folder

| Thế hệ | Điều cần tìm hiểu | Không được suy ra |
|---|---|---|
| Dịch vụ answer cũ | Vì sao query/history → retrieval → prompt → model → answer được viết như vậy? Những rủi ro nào đã được xử lý? | Audit cũ đề xuất MOVE không có nghĩa code đã chuyển hoặc finding vẫn còn. |
| Fuel hiện có | Vì sao phải thêm normalize/entity/semantics/authority/router/validator? Theo trace, lỗi đầu tiên nằm đâu? | Harness/test xanh không chứng minh hiểu mọi ngôn ngữ hoặc có business action. |
| EACL hướng mới | App đang sở hữu dữ liệu và hành động; control layer cần can thiệp ở đâu để có outcome kiểm được? | Không biến EACL thành app mới sở hữu lại toàn bộ business DB. |

Đọc theo thứ tự **job → call path → data/authority → failure → invariant → code boundary → verification**.
Sau mỗi lát, Luân phải giải thích được “vì sao đoạn này tồn tại, bỏ nó thì ca nào hỏng”, rồi mới bàn viết lại.

## 3. Index đọc hệ answer cũ — cần xác minh trên checkout gốc

Nguồn dẫn đường: [audit migration lịch sử](../02-migration-audit.md). Đây là các path **bên trong repo cũ**, không phải path đang tồn tại trong repo này.

| Lát đọc | Entry point lịch sử | So với Fuel hiện có | Sản phẩm học |
|---|---|---|---|
| Nhận request và identity | `cmd/server`, `internal/httpapi/api.go`, `internal/app/app.go` | `Principal`, `Ask.Handle`, tenant context | Request đi qua ai? Tenant/quyền được tin ở đâu? Dependency được lắp thế nào? |
| Điều phối một lượt | `internal/flow/flow.go`, `ports.go`, `options.go` | `core/pipeline/ask.go`, intent/router | Vẽ call graph một request; ai quyết route, ai đọc dữ liệu, ai trả lời? |
| Retrieval | `internal/flow/strategy.go`, `internal/store/retriever.go`, `store.go`, `internal/embed/jina.go` | `intelligence/rag`, `sourceoftruth` | Tách “có cần retrieval?” khỏi “retrieval chạy cách nào”; kiểm query scope và ACL. |
| History và context | `internal/store/room.go`, phần prompt trong `flow.go` | ContextPackage, evidence trong Ask | Tin trước nào được dùng, vì sao? Có đổi nghĩa request hoặc làm lộ dữ liệu không? |
| Model và lỗi | `internal/llm/`, `internal/flow/fault.go`, `trace.go` | `intelligence/models`, contracts/fault, traces | Timeout/retry/stop reason/cost/citations được xử lý ra sao; không mặc định model mới chữa lỗi. |
| Ingestion | `ingest/pipeline.py`, `adapters/parser.py`, `fetcher.py`, `chunker.py`, `writer.py` | `data/ingestion`, sources mapping | Phân biệt cấu trúc tài liệu với fact nghiệp vụ; giữ SSRF/byte limit/atomic replacement khi chúng thuộc lát chuyển. |
| Tests và đánh giá | `cmd/golden`, test Go/Python liên quan | `evals`, winrun/g7run | Oracle kiểm gì, bỏ lọt gì; test đọc được file khác test trả đúng nghiệp vụ. |

Không sao chép corpus, link nội bộ, dữ liệu/credentials thật từ hệ cũ sang tài liệu này hoặc repo public.
Mỗi finding Fxx trong audit là `[REPO]` cho đến khi tự đọc/đo lại đúng code nguồn; không coi cả danh sách là bug hiện hành.

## 4. Vòng học và chuyển một vertical slice

1. **Chốt job và expectation.** Một câu hỏi/công việc cụ thể; ai được dùng, app nào là authority, kết quả đúng là gì. Nếu policy hiện hành khác nhu cầu mới, ghi cả hai.
2. **Đóng dấu baseline.** Repo/revision cũ và mới, dữ liệu giả lập, đồng hồ, config đã bỏ secret, lệnh chạy. Không dùng holdout còn sạch.
3. **Theo request cũ từ đầu đến cuối.** Ghi input/output và chủ sở hữu mỗi quyết định; đọc cả caller, callee và test liên quan.
4. **Đo failure.** Dự đoán trước khi chạy; tìm first wrong layer. Nếu chỉ đọc code thì ghi `[CODE]`/`[INFERRED]`, không gọi là reproduce.
5. **Rút invariant.** Ví dụ số chính xác lấy từ nguồn có thẩm quyền cho đúng nghĩa; tenant không lấy từ body; dữ liệu phải được lọc quyền trước model.
6. **Chọn cách nhỏ nhất.** Có thể KEEP logic cũ, bọc boundary, hoặc REPLACE một đoạn. Không rewrite toàn luồng chỉ vì tên cũ hoặc nhãn RAG.
7. **Thực hiện khi scope/plan được duyệt.** Lát mới có entry point, contract và rollback rõ; chỉ chuyển dependency nó thực sự cần.
8. **Verify và soát.** Ca đúng, ca lỗi, ca từ chối, ca mơ hồ; regression và đối chứng; review độc lập theo quy trình. Báo improvement cùng giới hạn.

Phiếu cho mỗi lát:

```text
Job / actor / quyền:
Repo + revision cũ và mới:
Expected theo hợp đồng / nhu cầu mới:
Call path cũ:
Data và authority owner:
Failure + evidence + first wrong layer:
Invariant cần giữ:
KEEP / MOVE / GENERALIZE / DECOUPLE / REPLACE / DELETE + lý do:
Boundary/contract mới và compatibility:
Negative case / regression / rollback:
Kết quả tự đo / chưa kiểm:
Luân tự giải thích được gì:
```

Đích migration trong audit năm trước là **đề xuất theo kiến trúc lúc viết**, không phải lệnh tạo toàn bộ folder hôm nay. Chỉ chuyển phần đang cần cho scope EACL đã được duyệt.

## 5. Lát bắt đầu đề xuất

**Lát đầu: một câu hỏi exact fact so với câu hỏi tài liệu.** Dùng fixture tổng hợp, cùng actor/ngày/entity; chưa có action ghi.

- Trên hệ cũ: đo nó có luôn retrieval/gọi model không, lấy số ở đâu, trả gì khi thiếu dữ liệu. Chưa chạy thì để trống kết quả.
- Trên Fuel: lần theo S2 → S4 → S5 → S6 → S9; exact metric đi store/calculation/template, câu SOP đi retriever/model.
- Giải thích vì sao các boundary đó cần, và thử một ca hiểu sai intent dù số trong evidence đúng.
- Không coi việc giảm model call là thắng nghiệp vụ nếu correctness hoặc khả năng trả lời giảm.

Sau đó chọn lát thứ hai theo failure thực tế: tenant/auth, context/history, ingestion hoặc model errors. Không mang cache, embedding, queue, agent cùng lúc để “đủ stack”.

Customer Ops là lát tích hợp tương lai theo quyết định hiện hành: trước hết app không AI chạy đúng; rồi read/brief qua API; action proposal, authorization, idempotency và read-back theo story được duyệt. Không lấy bài học Fuel làm phép mở triển khai ngay.

## 6. Áp dụng bản CLAUDE mới có kiểm soát

1. Review draft với root hiện tại, kiểm luật nào bị mất hoặc yếu đi; bản draft có bảng ánh xạ các mục cũ.
2. Kiểm mọi tham chiếu `CLAUDE.md §...` đang sống trước khi đổi hệ đánh số. Các tài liệu như `core-candidates.md` đang trỏ §17/§24/§34; cần sửa tham chiếu khi kích hoạt, không sửa evidence đóng băng.
3. Thay root và cập nhật các tham chiếu cần thiết trong một mẻ tài liệu riêng theo D-V3-06. Không trộn rename module/remote hay sửa runtime.
4. Kiểm link, diff, compatibility và quyền commit/push theo phiên; không tự tuyên bố V3 hoặc WIN đã đạt sau đổi tên tài liệu.

Trong phiên này chỉ tạo draft và hướng dẫn học; chưa sửa root, chưa mở checkout hệ cũ, chưa migrate code, chưa gọi model trả phí.
