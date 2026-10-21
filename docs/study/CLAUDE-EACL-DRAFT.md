# CLAUDE.md — EACL Engineering Operating Contract

> Bản đề xuất thay thế root `CLAUDE.md`, ngày 21/10/2026. **Chưa được kích hoạt làm luật repo.**
> Root hiện hành vẫn có hiệu lực cho đến khi thay thế có chủ đích. Bản này không tự duyệt R0, ngưỡng V3, benchmark trả phí hay triển khai Customer Ops.
> Khi đưa bản này lên root, đường dẫn trong dấu backtick được hiểu tương đối với gốc repository.

## 0. Mục tiêu và cách dùng

Tạo thay đổi nhỏ, đúng, kiểm được, tái lập được và tin được với chi phí kiểm lại thấp.
Không lấy số dòng code, số task hoàn tất hoặc độ dài báo cáo thay cho kết quả nghiệp vụ.

Áp dụng toàn repo. Hướng dẫn trong thư mục con được bổ sung, không âm thầm phá invariant.
Tuân chỉ dẫn người dùng và luật nhà trên box; trước khi sửa code đọc nguồn luật nhà được AGENTS.md chỉ định và `~/.zcode/AGENTS.md`.
Quy trình commit, MR, soát độc lập và người approve theo luật nhà; tài liệu này không cấp quyền bỏ cổng.

Đọc tài liệu theo task; không nạp toàn bộ repo/docs/skills chỉ vì chúng tồn tại.
Mã, identifier, khóa cấu hình tiếng Anh; comment, tài liệu, commit tiếng Việt có dấu và giữ từ khóa kỹ thuật.

## 1. EACL là gì và không sở hữu gì

**EACL — Enterprise AI Control Layer**: kiến trúc tham chiếu và lớp kiểm soát thực thi để đưa capability AI xác suất vào phần mềm doanh nghiệp hiện có, trong khi ứng dụng nghiệp vụ giữ quyền sở hữu business truth, permissions, state, transactions và workflows.

Model là một capability, không phải database, calculator, permission engine, transaction manager hay audit log.
EACL không tự trở thành CRM thay thế, chatbot framework, agent framework hoặc nền tảng model.
Chưa gọi là framework/SDK dùng lại được khi chưa có portability proof và framework gate theo roadmap.

Hướng sản phẩm đã được duyệt ở `docs/CHO-QUYET-DINH.md`, D-V3-00:

- Fuel giữ vai Truth Benchmark, R1.
- Customer Ops là tích hợp tham chiếu tiếp theo: hiểu tình hình khách hàng và tạo task có kiểm soát.
- Duyệt hướng không đồng nghĩa đã có implementation hoặc được phép bỏ prerequisite. D-V3-09 chưa cho implement khi G7/Fuel còn là đường găng.
- Sports hiện là tài liệu/ca mô tả, chưa chứng minh domain thứ hai chạy được. Tenant B của Fuel là canary cùng ngành.

Tên sản phẩm trong tài liệu mới là EACL. Từ 21/10/2026, module (`github.com/luannt2002/enterprise-ai-control-layer`), biến môi trường (`EACL_*`) và tên kỹ thuật đang chạy cũng đã theo danh tính EACL (D-V3-03, đã làm trong một mẻ riêng). Không blind search/replace vào evidence đóng băng hoặc lịch sử.

## 2. Bảy nguyên tắc

1. **Business reality first.** Hiểu job, event/process, cơ chế quan sát, người chịu trách nhiệm, nguồn và rủi ro trước khi chọn AI.
2. **Software remains authority.** Model không sở hữu truth, quyền, state, transaction hay business invariants; app nghiệp vụ vẫn là nơi thực thi quyền và giao dịch của nó.
3. **AI where uncertainty creates value.** Ưu tiên capability nhỏ nhất đủ giải bài toán; dùng AI cho bất định có giá trị, không đặt AI vào mọi request. Deterministic control around probabilistic capability.
4. **Proposal ≠ authorization.** Model có thể diễn giải, suy luận, tổng hợp và đề xuất; đề xuất không tự thành fact hay hành động được phép.
5. **Output ≠ verified outcome.** Câu trả lời cần evidence; hành động có hậu quả cần kiểm trạng thái thực tế sau thực thi theo hợp đồng.
6. **Claims ≠ proof.** Lời khai, code có tồn tại, test đạt, E2E và bằng chứng độc lập là các mức khác nhau. Test chỉ chứng minh assertion nó thực sự kiểm.
7. **Complexity must be earned.** Business case → failure → reproducer → root cause → missing property → smallest sufficient solution → measurement → regression gate.

## 3. Trạng thái và nguồn thẩm quyền

Mỗi mô tả kiến trúc phải rõ: **CURRENT** (đang có), **TARGET** (đích được mô tả), **HYPOTHESIS**, **DEFERRED** hoặc **NON-GOAL**.
Invariant bắt buộc là yêu cầu, không mặc nhiên đã được chứng minh. Chỉ ghi verified khi có scope, revision, lệnh và bằng chứng phù hợp.

Nhãn báo cáo:

- `[MEASURED]`: tự chạy, ghi lệnh, revision, môi trường và kết quả.
- `[CODE]`: trực tiếp đọc cấu trúc/nhánh, ghi `tệp:dòng`; chưa phải thử hành vi.
- `[REPO]`: báo cáo/tài liệu có sẵn tuyên bố, chưa đo lại.
- `[INFERRED]`: suy từ evidence, nêu điều kiện.
- `[HYPOTHESIS]`: chưa chứng minh, kèm cách kiểm.

Nguồn cần tra:

| Cần biết | Đọc |
|---|---|
| Quyết định đã duyệt hoặc còn treo | `docs/CHO-QUYET-DINH.md`; phân biệt cột khuyến nghị với cột trả lời |
| Hướng và kế hoạch EACL | `docs/v3/`; nhiều mục vẫn là bản nháp, không tự kế thừa quyền triển khai |
| Hợp đồng Fuel | `docs/04-core-mvp.md`, ADR-0005/0006/0007 và các quyết định liên quan |
| Hành vi đang chạy | code, pack, migrations, tests tại revision thực tế |
| Trạng thái/bằng chứng cũ | `docs/STATUS.md`, `docs/g7/`, `docs/bao-cao/`; kiểm thời điểm và scope |
| Lịch sử chuyển hệ cũ | `docs/02-migration-audit.md`; hướng dẫn học `docs/study/legacy-to-eacl.md` |
| Học và debug | `docs/study/README.md`; không phải hợp đồng kỹ thuật |
| Quy ước | `docs/CONVENTIONS.md` |

Code là nguồn về hành vi hiện tại; requirement/policy được duyệt là nguồn về hành vi mong muốn.
Nếu khác nhau: ghi mismatch và xử lý theo scope, không chọn bên thuận tiện hoặc sửa test để code sai thành đúng.
Số cũ và tài liệu cũ không tự thành kết quả hiện hành.

## 4. Phạm vi đã triển khai và ranh giới mục tiêu

**CURRENT tại mốc khảo sát `5ee31c8`:** Fuel QUERY chạy trong tiến trình qua harness/CLI; ingestion nhiều định dạng; normalize/entity/intent bằng luật; routing; fact reconciliation; retrieval keyword/IDF; template/model gateway; validators; trace/replay; bộ nhớ và PostgreSQL store/RLS.

Chưa coi là đã có: API sản phẩm hoàn chỉnh, dịch vụ xác thực đầy đủ, Customer Ops, executor ghi nghiệp vụ, approval workflow, read-back action, memory hội thoại đầy đủ, agent/MCP hoặc framework đa miền đã chứng minh.
Contract có tên route/kiểu không chứng minh capability đã nối. Kiểm nhánh thực thi và test trước khi tuyên bố reachable.

### Ask hiện tại

```text
S0 ACCEPT → S1 TENANT → S2 NORMALIZE → S3 RESOLVE → S4 INTENT
→ S5 ROUTE → S6 EXECUTE → S7 CONTEXT → S8 GENERATE → S9 VALIDATE → S10 RESPOND
```

Đây là bản đồ logic của Ask, không phải yêu cầu mọi interaction EACL đi qua đúng chuỗi này hoặc có một service mỗi bước.
Router chọn capability trước khi truy xuất. Với Fuel, authority/reconciliation diễn ra trong `Catalog.Query()` lúc thực thi; không giả định mọi authoritative fact đã tính sẵn.
Route tất định không gọi model; exact facts không đến từ model generation; validation fail không giao draft như thành công.

### Trusted context responsibilities

Context chỉ chứa evidence được phép dùng, có provenance và semantics phù hợp.
Theo nhu cầu job, kiểm authority, freshness, conflict, deduplication, relevance và ngân sách context.
Đây là các thuộc tính cần thiết theo hợp đồng từng job, không bắt buộc tạo một Context Engine/service.
CURRENT: logic gói evidence nằm trong Ask và các capability; chưa chứng minh đầy đủ mọi thuộc tính nêu trên.

### RAG boundary

RAG là capability trả evidence; không sở hữu global routing, business authority, transaction hay quyền doanh nghiệp.
Tenant và ACL phải được thực thi ngoài model. Pipeline chỉ phụ thuộc hợp đồng retriever cần dùng.
Embedding, hybrid retrieval, reranker là các lựa chọn nội bộ khi có evidence cần; keyword/IDF hiện có không tự buộc phải thay.
Model nhận evidence đã lọc, trả draft; evidence id hợp lệ không tự chứng minh toàn bộ câu văn đúng nghĩa với nguồn.

### Action/workflow boundary — TARGET, chưa nối trong Fuel

```text
Proposal → schema/entity/time validation → tenant/permission/policy/business rules
→ approval nếu hợp đồng yêu cầu → idempotent execution qua app
→ authoritative read-back → verify intended vs actual → outcome → audit/evidence
```

App nghiệp vụ giữ authorization và transaction cuối cùng; kiểm ở EACL không thay kiểm phía app.
Kiểm lại điều kiện có thể đổi trước thực thi. Với mất phản hồi sau ghi, không retry mù tạo bản ghi trùng.
Read-back chỉ chứng minh trạng thái quan sát được theo hợp đồng, không mọi tác động ngoài đời.
Chuỗi đã biết ưu tiên workflow/state machine; chỉ mở agent khi có case chứng minh cách cố định không đủ.
Không tuyên bố “đã chuyển người phụ trách” nếu chỉ trả một nhãn mà chưa có chứng từ giao việc.

## 5. Dữ liệu, semantics và authority

Reality → business event/process → observation mechanism → record.
Record là dấu vết; nguồn nói X không tự chứng minh X xảy ra.
Trách nhiệm nghiệp vụ giúp xác định authority, nhưng authority còn có phạm vi, hiệu lực và chất lượng; owner không tự bảo đảm đúng.

Phân biệt source fact, authoritative result và câu trả lời đúng yêu cầu.
Issued, consumed, reported, billed có thể là các nghĩa khác nhau; chỉ hòa giải nguồn trong cùng nghĩa và phạm vi cần so.
Authority ≠ recency ≠ similarity ≠ frequency. Ngang thẩm quyền lệch ngoài policy thì hỏi lại/chuyển người, không chọn tùy ý.
Derived fact phải có công thức và evidence đầu vào. Số nghiệp vụ phải giữ kiểu/đơn vị/precision theo hợp đồng, không để model làm calculator.
Không mặc định source type quyết đủ semantics cho nguồn ngôn ngữ hỗn hợp; đánh dấu giới hạn mapping của Fuel.

Ambiguity và thiếu evidence phải được thể hiện bằng outcome phù hợp; không đoán im lặng.
Correction, ngày gửi, ngày sự kiện và phiên bản hiệu lực phải được phân biệt khi job đòi; không nhận là đã hỗ trợ bitemporal chỉ vì giữ raw text.

## 6. Quyền, tenant và pack

Tenant đến từ principal đã xác thực, không từ request body hoặc lời model. Struct Principal không tự là cơ chế đăng nhập.
Quyền đóng mặc định; kiểm cả capability lẫn data scope, tenant và ACL trước khi nội dung được đưa tới model/người dùng.
Không lộ ứng viên entity hoặc evidence ngoài phạm vi qua câu hỏi lại, log hay fallback.
RLS bảo vệ DB khi role/config đúng; kiểm allow, deny, canary tồn tại thật và role không bypass. Không gọi test SKIP là đã chứng minh RLS.

Shared control/code không đồng nghĩa shared data, knowledge, credentials, permissions hoặc state.
Core không rẽ nhánh theo tên tenant để chữa một ca riêng; chỉ thay boundary chung khi case và ADR chứng minh cần.
Pack là cơ chế hiện tại của Fuel, chưa phải yêu cầu mọi integration EACL phải dùng YAML.
Pack mô tả config/policy; code sở hữu hành vi. Nếu YAML thành vòng lặp/biểu thức/state machine phức tạp, dừng và đánh giá domain module.
Tích hợp app khác phải giữ quyền sở hữu dữ liệu/quy trình của app; không ép schema Fuel vào app đó.

Repo public: không secret, dữ liệu thật, link nội bộ, DSN hoặc transcript riêng tư trong code/docs/evidence.
Không log token; không để retrieval/tool vượt authorization. Security test chỉ chạy trong scope được phép.
Mọi thay đổi auth/tenant/ACL/action cần cả ca cho phép và từ chối phù hợp.

## 7. Cách làm task và debug

Trước code: job, scope, hợp đồng/expectation, current behavior, invariant, giải pháp nhỏ nhất và cách chứng minh.
Plan gọn theo luật nhà; phản biện giả định trước khi thi hành.
Quyết định thường lệ, đảo được và đã được ủy quyền thì tự làm; hỏi khi thông tin thiếu làm đổi bản chất task, quyền, dữ liệu thật, chi phí hoặc quyết định không dễ đảo.
Không hỏi lại quyền đã được cấp, không dừng giữa task để né công việc còn rõ.

Debug theo evidence:

1. Ghi expected theo hợp đồng và nhu cầu nghiệp vụ; nếu chưa định nghĩa hoặc mâu thuẫn thì làm rõ trước phán quyết nghiệp vụ.
2. Reproduce với principal, đồng hồ, code/pack/data version xác định.
3. So đầu vào/đầu ra từng bước; tìm **first wrong layer**, gồm ingestion, business semantics và intent.
4. Tìm phép thử có thể bác chẩn đoán; ghi các lỗi độc lập phía sau nếu có.
5. Sửa đúng tầng; regression test trước/sau và negative case khi cần.

Không đổi model để che data/parser/intent/router/retrieval/context/permission/validation/infra bug.
Danh sách fault trong báo cáo theo taxonomy hiện hành của `docs/CONVENTIONS.md` và ADR-0008; không tự tạo vocabulary cạnh tranh.

Code đúng pack nhưng pack sai nhu cầu là policy/requirement mismatch; code khác pack là lỗi tuân thủ có thể kiểm; scorer khác oracle có căn cứ là vấn đề bộ chấm riêng.
Tầng sau chạy đúng trên input sai không làm câu trả lời đúng nghiệp vụ.

## 8. Simple first và capability gates

Mỗi dòng sửa truy được về requirement, bug, invariant, test hoặc hệ quả cần thiết.
Không cleanup/rename/refactor ngoài scope, tạo folder vì sơ đồ có tên, hoặc thêm dependency/service dự phòng.
Phát hiện ngoài scope: ghi observed, impact, follow-up rồi tiếp tục nếu không block.

Capability mới cần failure tái hiện, cách nhỏ hơn đã cân nhắc, output contract/validator, metric và regression gate:

| Capability | Evidence cần có trước khi mở |
|---|---|
| Model intent/extraction | Luật không giải được nhóm ca đo được; uncertainty cần AI; kiểm được output và fallback |
| Reranker | Candidate retrieval đủ tốt nhưng ranking trượt |
| Agent | Fixed workflow không giải hợp lý business case |
| MCP | Integration/consumer thực sự hưởng lợi từ protocol boundary |
| Queue/cache | Đo tải/độ trễ chứng minh đường hiện tại không đủ; có consistency/invalidation contract |
| Vision/fine-tuning/memory | Nhu cầu và failure cụ thể, dữ liệu đủ, cách đơn giản hơn chưa đủ, có eval |

Default implementation policy của repo: **Go first, Python by evidence**. Đây không phải luật EACL bắt buộc mọi hệ phải dùng Go.
Python chỉ qua boundary rõ khi ecosystem tạo lợi ích đo được; không dựng service Python chỉ để bọc HTTP model API mà Go làm đơn giản được.
Không tự chuyển logic sang ngôn ngữ khác chỉ để khớp sơ đồ.

## 9. Học hệ cũ và thay đổi kiến trúc

Không viết lại từ trắng hoặc chuyển mọi file theo danh mục công nghệ.
Dùng `docs/02-migration-audit.md` như chỉ mục lịch sử; đọc code nguồn đúng revision, kiểm lại các finding trước khi mang sang.
Phân loại từng phần: KEEP / MOVE / GENERALIZE / DECOUPLE / REPLACE / DELETE; ghi lý do, regression và compatibility.
Chuyển một vertical slice đang cần, giữ hệ cũ hoạt động nếu có deployment thật; không đổi routing/traffic hoặc đụng dữ liệu thật nếu chưa được ủy quyền.
Không chuyển golden/corpus/link/credentials thật từ repo cũ vào repo public; tạo fixture tổng hợp giữ failure property.

Cross-cutting change phải nêu: problem/reproducer, boundary sai, cách nhỏ hơn, vì sao thuộc core, tác động compatibility, migration/rollback, tests/metric và ADR khi cần.
`docs/v3/core-candidates.md` chỉ là ứng viên, không phải lệnh trích xuất lõi. Chưa có proof đa miền thì chưa generalize thành framework.

## 10. Verification và evaluation

Ưu tiên test boundary vừa đổi; đối chiếu theo tên/identity, không chỉ đếm tổng.
Test đỏ phân loại ENVIRONMENT / BASELINE / NEW REGRESSION; baseline đỏ không tự là lỗi của change, nhưng có thể làm mất khả năng xác minh.
Không sửa test cho code sai thành xanh; thay expectation phải có requirement/contract đổi có chủ ý hoặc chứng minh test sai.
Gate mới cần negative case/đột biến chứng minh chặn được failure và đối chứng dương tránh chặn oan.
Không dùng LLM-as-judge làm bằng chứng duy nhất khi có oracle tất định khả thi.

Đo theo boundary: normalize/entity/intent/router, retrieval, citation/groundedness, validation catch/false block, quyền/tenant, task/workflow outcome, latency/cost.
Build pass ≠ runtime pass; unit pass ≠ E2E; ratchet pass có thể vẫn còn lỗi đã ghi; benchmark ≠ production quality.
DEV / REGRESSION / HOLDOUT tách bạch. Không đọc hoặc dùng holdout sạch để học/sửa code; contamination được ghi, không xóa lịch sử.

Fuel dùng **W-G1…W-G7**, theo hợp đồng riêng và ADR cập nhật; V3 dùng G1…G6. Không trộn hai thang hoặc chép ngưỡng Fuel sang Customer Ops.
W-G7 cần model thật, scorer đủ điều kiện, evaluation set hợp lệ, contract và chi phí được ủy quyền; chạy giả/dry-run không chứng minh value.
Theo code tại mốc khảo sát, `scorerFrozen=false` chặn lượt chính; khi trạng thái đổi phải đo lại.
Provider/model/sampling cấu hình theo gateway và hợp đồng benchmark; không tự đổi tham số giữa lượt chạy hoặc hardcode lựa chọn vào logic nghiệp vụ.

## 11. Review, automation, commit và bàn giao

Ba lỗi phải chặn: làm qua loa, tự chấm bài và đi lạc scope.
Khi yêu cầu independent verification, reviewer là phiên khác, ưu tiên read-only; không tự sửa rồi tự chấm. Self-review phải gọi đúng tên.
Chỉ dùng agent khi được phép theo chỉ dẫn hiện hành, workstream độc lập, tài nguyên đủ, scope/write-set và verification rõ; luật box dùng chung được ưu tiên.
Automation chỉ cho việc mechanical, recurring, verifiable; không tự approve thay người hoặc chạy open-ended feature bằng routine.

Commit nhỏ, coherent, rollback độc lập; không trộn cleanup. Không commit/push/merge/deploy vượt quyền đã cấp; giữ cổng người approve theo luật nhà.
Nếu chưa được yêu cầu commit/push trong task tài liệu, để diff local để review; khi commit bản thay root này phải là mẻ riêng theo D-V3-06.

DONE theo scope: requirement đáp ứng, diff đúng phạm vi, kiểm phù hợp có evidence, negative/regression/security cần thiết đã chạy, docs/contracts đồng bộ, giới hạn nói rõ.
Bàn giao ngắn: **Changed · Verified (lệnh + kết quả) · Not verified/limitations**.
Unknown ảnh hưởng công việc ghi `Unknown | Impact | How to verify | Blocked work`; không dùng bảng nghi lễ khi không có unknown đáng kể.

## 12. Ghi chú tương thích khi thay root

Bản root cũ dùng §§0–38; bản này tổ chức lại theo trách nhiệm. Trước kích hoạt, cập nhật tham chiếu đang sống; không chỉnh evidence/lịch sử đóng băng để đổi số mục.

| Mục cũ | Nơi giữ ý nghĩa trong bản mới |
|---|---|
| §§0–8 | §§0, 3, 7, 8, 10, 11 |
| §9, §24 | §§1, 2, 9 |
| §10 | §§2–8, 10 |
| §§11–15 | §4 |
| §§16–21 | §§5, 6, 8 |
| §§22–23, §32 | §10 |
| §25–26, §35 | §§3, 4, 9; cây thư mục lấy từ repo thực, không từ hình đích |
| §§27–31, §33–34, §36–38 | §§7–11 |

Không công bố đây là bản root đã được áp dụng chỉ vì file draft tồn tại.
