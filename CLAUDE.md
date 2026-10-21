# CLAUDE.md — EAIS Engineering Operating Contract

> Áp dụng cho toàn repository EAIS.
> Hướng dẫn sâu hơn trong thư mục con được phép bổ sung nhưng không được âm thầm
> phá các invariant của tài liệu này.
>
> Mục tiêu không phải "AI viết được nhiều code".
> Mục tiêu là tạo ra thay đổi NHỎ, ĐÚNG, KIỂM ĐƯỢC, TÁI LẬP ĐƯỢC và TIN ĐƯỢC.

---

# 0. MỤC TIÊU VẬN HÀNH

Nút thắt của AI coding không còn chủ yếu là VIẾT CODE.

Nút thắt là:

> Có thể TIN kết quả với chi phí kiểm lại thấp hay không?

Thước đo cho mọi quy trình, abstraction, agent, skill, test và automation:

> Nó làm "tin được kết quả" RẺ hơn hay ĐẮT hơn?

- Rẻ hơn → giữ.
- Đắt hơn mà không tăng bằng chứng tương ứng → bỏ.
- Chưa đo → ghi CHƯA ĐO.
- Chưa chứng minh → không gọi là hoàn thành.

Luật thiên về CHẮC hơn NHANH.

Với việc nhỏ, dùng phán đoán.
Không biến quy trình thành nghi lễ.

---

# 1. BA KIỂU HỎNG PHẢI CHẶN

| Kiểu hỏng | Dấu hiệu |
|---|---|
| Làm qua loa | bỏ bước khó nhưng báo "xong" |
| Tự chấm bài | code sai nhưng sửa test/gate để xanh |
| Đi lạc việc | yêu cầu nhỏ nhưng thay đổi lan rộng |

Một process/gate không chặn được lỗi thực tế nào thì chưa chứng minh được giá trị.

Nếu một gate mới chưa từng fail:

> cố tình tạo ít nhất một negative case để chứng minh gate thực sự chặn.

---

# 2. BA LUẬT LÕI

## 2.1 Người chấm ≠ người làm

Author không phải independent reviewer.

Khi task yêu cầu independent verification:

- dùng session/agent riêng;
- reviewer ưu tiên read-only;
- reviewer không tự sửa code rồi tự xác nhận;
- finding quay lại implementation flow để sửa;
- sau sửa phải verify lại.

Self-review vẫn hữu ích nhưng phải ghi đúng là:

> SELF REVIEW

không gọi nó là independent verification.

---

## 2.2 Đòi bằng chứng, không nhận lời khai

Không chấp nhận:

- "test pass";
- "build OK";
- "đã fix";
- "không còn lỗi";
- "works correctly";

nếu không có bằng chứng tái lập phù hợp.

Bằng chứng tốt gồm:

- command vừa chạy;
- test reproducer;
- negative test;
- truth table;
- diff;
- trace;
- query;
- metric;
- benchmark;
- fixture;
- deterministic assertion.

`rc=0` chỉ chứng minh command chạy thành công.

Nó không tự chứng minh business behavior đúng.

---

## 2.3 Sửa ở tầng cao nhất hợp lý

Nếu lỗi:

- xảy ra một lần → sửa tại chỗ;
- lặp lại → rule/test/skill;
- xảy ra cả lớp → lint/CI/gate;
- do routine → sửa routine;
- do architecture → sửa boundary/contract/ADR.

Không tạo abstraction lớn chỉ vì một lỗi đơn lẻ.

---

# 3. THINK BEFORE CODING

Trước implementation phải hiểu:

1. Người dùng thực sự muốn kết quả gì?
2. Scope nằm ở đâu?
3. Success criteria là gì?
4. Existing behavior là gì?
5. Invariant nào không được phá?
6. Cách nhỏ nhất để giải đúng là gì?
7. Làm sao chứng minh nó đúng?

Phân biệt rõ:

- ĐÃ XÁC MINH
- CHƯA XÁC MINH
- GIẢ ĐỊNH
- MỤC TIÊU
- ĐỀ XUẤT

Không biến giả định thành dữ kiện.

Không biến target architecture thành current implementation.

---

# 4. HỎI CHO ĐÁNG

Trước khi hỏi user:

1. Đã hiểu yêu cầu chưa?
2. Có default/best practice hợp lý không?
3. Nếu chọn sai có dễ rollback không?

Nếu có default rõ và rollback được:

> tự quyết và tiếp tục.

Hỏi khi:

- hai cách hiểu tạo ra implementation khác hẳn;
- quyết định không dễ rollback;
- liên quan xoá/ghi đè/dữ liệu thật/quyền/chi phí;
- mâu thuẫn với quyết định đã chốt;
- đoán sai sẽ làm task đi sai hướng hoàn toàn.

Khi hỏi:

- nói đã hiểu gì;
- nói phần nào đã tự quyết;
- đưa hệ quả của từng lựa chọn;
- đưa recommendation;
- nói vì sao;
- kèm số đo nếu có.

Không hỏi để né trách nhiệm.

Không hỏi "có làm tiếp không?" giữa task khi phần tiếp theo đã rõ.

---

# 5. SIMPLE FIRST

Viết lượng code nhỏ nhất giải ĐÚNG bài.

Không tự thêm:

- feature;
- abstraction;
- configuration;
- framework;
- dependency;
- service;
- agent;
- model;
- MCP;
- cache;
- queue;
- retry layer;
- error handling giả định;

nếu requirement/evidence chưa cần.

Nếu 200 dòng mà 50 dòng đủ:

> đơn giản hóa.

Hỏi:

> Một senior engineer đọc đoạn này có nói "đang over-engineer" không?

Nếu có → simplify.

---

# 6. SURGICAL CHANGES

Chỉ thay đổi những gì requirement bắt buộc.

Không:

- refactor code bên cạnh;
- format cả file vô cớ;
- rename unrelated symbols;
- sửa comment không liên quan;
- xóa dead code cũ;
- "tiện tay cải thiện";
- thay architecture ngoài scope.

Mỗi changed line phải truy ngược được về:

- requirement;
- bug;
- invariant;
- test;
- necessary consequence của change.

Cleanup imports/variables/functions chỉ khi chính change hiện tại làm chúng thừa.

Giữ style/convention hiện tại của repository.

---

# 7. ĐO, ĐỪNG ĐOÁN

Số đưa vào kết luận phải đến từ measurement vừa thực hiện hoặc nguồn đã được xác định rõ.

Nếu chưa chạy:

> CHƯA ĐO.

Nếu số lấy từ tài liệu/agent khác:

> chưa phải independent measurement.

Không dùng:

`grep = 0`

để tự động kết luận:

> không tồn tại.

Khi cần, mở file/đọc code path thật.

Build success ≠ runtime success.

Syntax valid ≠ behavior correct.

Unit test pass ≠ E2E correct.

Benchmark ≠ production quality.

---

# 8. GOAL-DRIVEN EXECUTION

Chuyển task thành mục tiêu kiểm được.

Ví dụ:

"Fix router bug"

thành:

1. reproduce route sai;
2. tạo regression test;
3. sửa implementation;
4. chứng minh case cũ fail trước/sau fix pass;
5. chạy relevant regression suite.

"Add validation"

thành:

1. định nghĩa invalid inputs;
2. negative tests phải fail đúng;
3. valid inputs vẫn pass;
4. implement minimum validator;
5. chạy regression.

Plan cho task lớn phải ngắn.

Plan mô tả:

> mục tiêu → boundary → cách chứng minh.

Không viết tutorial từng phím cho agent nếu không cần.

---

# 9. EAIS — PRODUCT MISSION

EAIS = Enterprise AI Systems.

EAIS là:

> Multi-tenant Enterprise AI Systems Platform biến dữ liệu, tri thức,
> workflow và tools theo domain thành AI capabilities có governance,
> testability và observability mà không coupling shared core vào một ngành.

Nguồn: câu gốc ngày 20/10 (`d70ebc8`) cộng hai chế độ sở hữu của `docs/ROADMAP-DAY-DU.md` §1 (D-RM-01, D-RM-02).

Hai chế độ sở hữu:

| Chế độ | Khi nào | Ai giữ business truth, permissions, state, transactions |
|---|---|---|
| A. Tích hợp | doanh nghiệp đã có CRM / ERP / app có API | app của khách; EAIS đọc và ghi qua API bằng token của người dùng |
| B. Tự lo | doanh nghiệp chỉ có chat / Excel / giấy | EAIS: thu, chuẩn hoá, đối soát, lưu trữ |

- Dùng lại hệ thống doanh nghiệp đã có khi hệ đó đang giữ sự thật.
- Tự lo phần doanh nghiệp còn thiếu: thu dữ liệu, chuẩn hoá, đối soát.
- Core không gắn vào một ngành.

North Star:

Business Reality
→ Trusted Context
→ Useful Intelligence
→ Controlled Action
→ Verified Outcome
→ Measured Value

EAIS KHÔNG phải:

- chatbot framework;
- RAG wrapper;
- prompt collection;
- agent framework;
- model gateway có UI;
- CRM / ERP thay thế khi khách đã có;
- ontology chung cho mọi ngành;
- Construction-specific application;
- Sports-specific application.

Ở chế độ A, EAIS không đọc database của khách vòng qua quyền của app.

Reference tenants hiện tại:

- Construction
- Sports Data

Hai tenant tồn tại để kiểm chứng abstraction của platform.

Không encode domain-specific behavior vào shared core chỉ để một tenant chạy được.

Ghi chú tương thích (D-V3-06): câu sứ mệnh dùng ngày 21/10/2026, trước D-RM-02 — "tích hợp AI vào phần mềm doanh nghiệp đã có,
phần mềm đó giữ business truth, permissions, state, transactions và workflows" — nay là chế độ A. Chế độ B là phần thêm theo
D-RM-01. Luật ở §10–§38 không đổi.

---

# 10. EAIS — FIRST PRINCIPLES

## P1 — Model ≠ System

Model là một capability.

Model không phải:

- database;
- source of truth;
- permission engine;
- calculator;
- transaction manager;
- workflow state;
- business rule engine;
- audit log.

---

## P2 — Deterministic shell around probabilistic core

Ưu tiên software tất định cho thứ có thể xác định chính xác.

Ví dụ:

- auth;
- permissions;
- tenant isolation;
- calculation;
- SQL;
- schema validation;
- workflow transitions;
- transactions;
- business invariants;
- source authority;
- idempotency.

Dùng probabilistic intelligence khi uncertainty là bản chất của bài toán:

- semantic interpretation;
- ambiguous language;
- extraction khó;
- reasoning;
- summarization;
- generation;
- learned ranking;
- vision;
- planning khi thật sự cần.

---

## P3 — Model output is a proposal, not truth

Model output không tự động là:

- fact;
- authorized action;
- valid transaction;
- source of truth.

Output phải đi qua validation phù hợp trước khi được sử dụng downstream.

---

## P4 — Facts come from authoritative sources

Exact business facts phải đến từ:

- structured data;
- authoritative API;
- validated event;
- authoritative document/version;
- deterministic calculation trên evidence.

Không lấy exact number từ model memory/generation.

---

## P5 — Ambiguity must remain visible

Không đoán im lặng.

Khi thiếu evidence hoặc ambiguity vượt threshold:

- CLARIFICATION_REQUIRED;
- INSUFFICIENT_EVIDENCE;
- SOURCE_CONFLICT;
- HUMAN_REVIEW_REQUIRED;
- REJECT;

tùy policy.

---

## P6 — Shared Platform ≠ Shared Knowledge

Shared:

- code;
- contracts;
- runtime;
- capability interfaces.

Tenant-specific:

- data;
- corpus;
- ontology;
- terminology;
- schemas;
- rules;
- tools;
- credentials;
- skills;
- permissions;
- evaluation datasets;
- policies.

Không để cross-tenant leakage.

---

## P7 — Complexity must be earned by evidence

Không thêm capability vì nó phổ biến.

Capability mới phải trả lời được:

1. failure nào yêu cầu nó?
2. failure có reproducible không?
3. abstraction hiện tại có giải được không?
4. implementation nhỏ hơn có tồn tại không?
5. metric nào chứng minh improvement?
6. regression gate nào bảo vệ nó?

---

## P8 — Every important result must be traceable

Phải có khả năng truy:

request
→ tenant
→ identity
→ normalization
→ entity resolution
→ intent
→ route
→ evidence
→ context
→ model/tool
→ validation
→ response/action.

---

## P9 — Benchmark ≠ Business Quality

Đánh giá theo:

- component metrics;
- tenant golden cases;
- dirty data;
- ambiguity;
- conflicts;
- permission cases;
- failures;
- E2E business success.

---

## P10 — Debug layer before changing model

Khi output sai, xác định fault domain trước:

DATA
PARSING
NORMALIZATION
ENTITY
SOURCE_AUTHORITY
ROUTER
RETRIEVAL
CONTEXT
MODEL
SKILL
TOOL
WORKFLOW
AUTH
TENANCY
VALIDATION
INFRASTRUCTURE

Không mặc định:

> output sai → đổi model.

---

# 11. EAIS — RUNTIME BOUNDARY

Luồng Ask mục tiêu:

S0 ACCEPT
→ S1 TENANT
→ S2 NORMALIZE
→ S3 RESOLVE
→ S4 INTENT
→ S5 ROUTE
→ S6 EXECUTE
→ S7 CONTEXT
→ S8 GENERATE
→ S9 VALIDATE
→ S10 RESPOND

Invariant:

- route không cần model → không gọi model;
- model không phải nguồn của exact facts;
- validation fail → không giao output như thành công;
- permission được kiểm ngoài model;
- tenant không lấy từ untrusted request body;
- evidence phải giữ provenance.

Không ép mọi business interaction qua Ask pipeline.

Architecture phải có khả năng mở rộng cho:

- QUERY;
- COMMAND;
- EVENT;
- WORKFLOW;

khi business cases chứng minh cần.

---

# 12. RAG BOUNDARY

RAG là capability, không phải EAIS.

RAG chịu trách nhiệm cho:

- document ingestion;
- chunking;
- embedding;
- indexing;
- retrieval;
- hybrid retrieval;
- reranking khi được gate;
- evidence/citation.

RAG không sở hữu:

- global routing;
- tenant authorization;
- business transaction;
- workflow state;
- source-of-truth arbitration;
- exact calculation;
- enterprise permission.

Conceptual contract:

KnowledgeRetriever
    Retrieve(...)
    → []Evidence

EAIS chỉ cần biết:

> cần evidence.

EAIS core không được phụ thuộc việc retrieval bên trong dùng BM25, vector,
hybrid hay implementation khác.

---

# 13. CONTEXT ENGINE BOUNDARY

RAG chỉ là một ContextSource.

Context có thể đến từ:

- facts;
- SQL;
- events;
- RAG;
- API;
- tool result;
- calculation;
- workflow state;
- permitted history.

Context Engine chịu trách nhiệm:

- permission filtering;
- authority;
- freshness;
- deduplication;
- conflict handling;
- ranking;
- token budget;
- provenance;
- evidence binding.

Model chỉ nhận context mà system cho phép.

---

# 14. TOOL / ACTION BOUNDARY

Model có thể ĐỀ XUẤT action.

System quyết định action có được thực hiện không.

Flow:

Model/Workflow
→ Tool Proposal
→ Schema Validation
→ Tenant Check
→ Permission
→ Policy
→ Business Rules
→ Idempotency
→ Execute
→ Verify
→ Audit.

Model không tự cấp quyền cho chính nó.

Side-effecting tools cần negative permission tests.

---

# 15. WORKFLOW / STATE BOUNDARY

Model có thể:

- classify;
- interpret;
- recommend;
- summarize;
- explain.

Workflow engine/software sở hữu:

- state;
- allowed transitions;
- authorization;
- invariants;
- transaction boundaries.

Nếu sequence đã biết:

> ưu tiên workflow/state machine.

Agent chỉ được mở khi có case chứng minh fixed workflow không đủ.

---

# 16. TENANT BOUNDARY

Shared core không được có business branching kiểu:

    if tenant == "construction" { ... }

hoặc:

    if tenant == "sports" { ... }

trừ khi:

- architecture evidence chứng minh unavoidable;
- được ghi rõ là architecture debt;
- có ADR.

Khác biệt tenant nằm trong pack/domain module thích hợp.

Mọi business table phải tuân tenant isolation theo architecture hiện hành.

Cross-tenant leakage là lỗi nghiêm trọng.

---

# 17. PACK BOUNDARY

Tenant pack mô tả:

- terminology;
- aliases;
- ontology/config;
- entity mapping;
- metrics;
- intents;
- routing policy;
- tools;
- skills;
- templates;
- validation policy;
- model policy.

Pack KHÔNG được dần biến thành programming language.

Nếu YAML bắt đầu mô phỏng:

- nested procedural logic;
- loops;
- complex state machines;
- arbitrary expressions;

dừng và đánh giá domain module/code thay vì tiếp tục mở rộng config DSL.

Rule:

> Pack describes policy/configuration.
> Code implements behavior.

---

# 18. GO / PYTHON BOUNDARY

Default:

> GO FIRST. PYTHON BY EVIDENCE.

## Go sở hữu System/Control Plane

Ưu tiên Go cho:

- API;
- tenant runtime;
- auth;
- RBAC/ABAC;
- router orchestration;
- workflow/state;
- business rules;
- validation;
- tools;
- transactions;
- PostgreSQL;
- source of truth;
- reconciliation policy;
- context builder;
- model gateway;
- RAG orchestration;
- event processing;
- audit;
- observability;
- cost/SLA.

## Python dùng cho specialized compute

Python được phép sau explicit capability boundary khi ecosystem Python mang lại
giá trị đo được, ví dụ:

- document parsing;
- OCR;
- layout understanding;
- table extraction;
- local ML inference;
- learned reranker;
- local embeddings;
- ML entity matching;
- vision;
- training/fine-tuning;
- data-science experiments.

Python service không được sở hữu:

- tenant authorization;
- enterprise transactions;
- core workflow state;
- source-of-truth policy;
- global orchestration.

Không tạo Python service chỉ để gọi một HTTP model/embedding API nếu Go làm trực tiếp
đơn giản hơn.

---

# 19. CAPABILITY GATES

Capability mặc định ĐÓNG nếu chưa có evidence.

Ví dụ:

Model Intent Classifier
→ mở khi deterministic rules fail measurable cases.

Model Extraction
→ mở khi deterministic extraction không đạt threshold và output có validator.

Reranker
→ mở khi retrieval eval chứng minh candidate retrieval tốt nhưng ranking chưa đủ.

Agent
→ mở khi có business case fixed workflow không giải hợp lý được.

MCP
→ mở khi có integration/consumer thực sự hưởng lợi từ protocol boundary.

Queue/Cache
→ mở khi measurement chứng minh synchronous/storage path hiện tại không đạt.

Vision/Fine-tuning
→ mở khi có data + reproducible failure mà context/tool/prompt/rules không giải được.

Một capability chưa từng bị gate từ chối phải có negative test chứng minh gate tồn tại thật.

---

# 20. SOURCE OF TRUTH

Authority ≠ Recency ≠ Similarity ≠ Frequency.

Không để model chọn nguồn "nghe có vẻ đúng".

Phân biệt khi domain yêu cầu:

- observation;
- measurement;
- reported value;
- authoritative fact;
- derived fact;
- estimate;
- invoice value.

Ví dụ:

warehouse issued 3850 L
supervisor reported ~4000 L
sensor measured 3782 L
invoice billed 4100 L

không mặc định là bốn phiên bản của cùng một fact.

Có thể là bốn business semantics khác nhau.

Conflict phải:

detect
→ classify
→ apply authority policy
→ reconcile
→ clarify/human review khi cần.

---

# 21. SECURITY

Không:

- hardcode secret;
- ghi secret vào docs;
- log credential/token;
- tin tenant_id từ untrusted payload;
- bỏ permission check vì "model đã biết user";
- để retrieval bypass ACL;
- để tool bypass authorization.

Khi đổi:

- tenancy;
- auth;
- permissions;
- retrieval ACL;
- tool permissions;

phải có cả allow và deny tests.

Security test chỉ được thực hiện trong phạm vi và môi trường được phép.

---

# 22. TEST STRATEGY

Ưu tiên test đúng boundary vừa thay đổi.

Phân loại test đỏ:

1. ENVIRONMENT
2. BASELINE
3. REGRESSION FROM CURRENT CHANGE

Đối chiếu test theo identity/name khi cần, không chỉ đếm tổng số.

Không sửa test chỉ để implementation sai trở thành xanh.

Test có thể thay khi:

- requirement thay;
- contract thay có chủ ý;
- test thực sự sai;

và lý do phải rõ.

---

# 23. EAIS EVALUATION

Tùy component, dùng metric thích hợp:

- normalization accuracy;
- entity resolution accuracy;
- intent accuracy;
- router accuracy;
- retrieval Recall@K;
- MRR/NDCG khi phù hợp;
- citation correctness;
- groundedness;
- tool success;
- workflow success;
- validation catch rate;
- false block rate;
- permission violations;
- tenant leakage;
- latency;
- cost;
- E2E business task success.

Không dùng LLM-as-judge làm bằng chứng duy nhất nếu deterministic evaluation khả thi.

---

# 24. FAILURE-DRIVEN DEVELOPMENT

Feature/capability EAIS nên tiến hóa theo:

Business Case
→ Current Failure
→ Reproducer
→ Root Cause
→ Missing System Property
→ Minimal Capability
→ Implementation
→ Test
→ Metric
→ Regression Gate.

Không:

Cool Technology
→ Add To Core
→ Find Use Case Later.

Construction và Sports là reference tenants dùng để PHÁ abstraction.

Nếu tenant mới yêu cầu sửa shared core:

1. xác định đây là missing generic primitive hay domain-specific behavior;
2. thử giải bằng tenant pack/domain module trước;
3. chỉ generalize core khi evidence từ cases đủ mạnh.

---

# 25. REPO MAP

Target/current architecture phải được phân biệt rõ.

Các khu vực chính (đánh dấu "đích" = chưa có thư mục, chỉ có trong kiến trúc đích):

    cmd/eais/          đích — mặt phẳng P7 Vận hành
    core/
    data/
    intelligence/
    execution/         đích — mặt phẳng P4 Hành động
    reliability/
    governance/        đích — một phần của P5 Kiểm soát
    observability/
    platform/
    domain/            đích
    db/migrations/
    tenants/
    simulations/
    evals/
    docs/
    deploy/

Bảy mặt phẳng P1–P7 và các luồng L0–L10, mỗi luồng tách hiện có / đích: `docs/ROADMAP-DAY-DU.md` §3, §5 và `docs/01-architecture.md`.

Không tạo folder chỉ vì target architecture có tên đó.

Folder được tạo khi có implementation thật cần boundary đó.

Đọc architecture/ADR hiện hành trước khi thay đổi cross-module boundaries.

---

# 26. DOCUMENTATION ROUTING

Trước task, chỉ đọc tài liệu liên quan.

Ưu tiên:

1. CLAUDE.md gần scope đang sửa;
2. architecture docs;
3. ADR liên quan;
4. contract/interface;
5. implementation;
6. tests;
7. relevant skill;
8. broader docs khi cần.

Không nạp toàn bộ repository/docs/skills chỉ vì chúng tồn tại.

Khi docs và code mâu thuẫn:

- không âm thầm chọn bên thuận tiện;
- xác định current behavior;
- xác định source-of-truth cho requirement;
- báo mismatch;
- sửa đồng bộ trong scope nếu task yêu cầu.

---

# 27. SKILLS / SUBAGENTS

Giao MỤC TIÊU, không micromanage từng bước.

Subagent phải nhận đủ:

- objective;
- scope;
- relevant constraints;
- success criteria;
- evidence expected.

Không spawn nhiều agent hơn số workstream độc lập.

Parallel work chỉ khi:

- tài nguyên cho phép;
- write-set không overlap hoặc có ownership rõ;
- verification strategy đã được chia.

Reviewer và implementer không dùng chung vai khi yêu cầu independent review.

---

# 28. ROUTINES / AUTOMATION

Automation phù hợp với việc:

- mechanical;
- repeatable;
- verifiable;
- recurring.

Không dùng routine cho open-ended feature development.

Routine phải self-contained đủ để session mới chạy được.

Đo output:

- findings;
- fixed findings;
- unresolved age;
- regression caught;

không đo vanity metrics như token count hoặc "11/11 tasks completed".

---

# 29. COMMITS / REVIEW

Commit:

> nhỏ, coherent, rollback độc lập được.

Không trộn unrelated cleanup.

Review/MR:

> gom đủ context để reviewer hiểu và quyết định với ít lần đọc lại.

Không hy sinh commit quality chỉ để giảm số MR.

Không tự merge/deploy nếu quyền/gate không cho phép.

---

# 30. MODEL / PROVIDER RULE

Không hardcode model/provider nếu architecture yêu cầu Model Gateway.

Model selection phải đi qua policy/config hiện hành.

Model mạnh hơn không được dùng để che:

- data bug;
- routing bug;
- retrieval bug;
- context bug;
- validation bug;
- tool bug;
- workflow bug.

Effort/model choice không thay thế evidence.

---

# 31. DEFINITION OF DONE

Một task chưa DONE chỉ vì code đã được viết.

Tùy scope, DONE yêu cầu:

- requirement được đáp ứng;
- diff surgical;
- relevant tests pass;
- negative case được kiểm khi cần;
- regression được kiểm;
- security/tenant boundary không bị phá;
- docs/contracts được cập nhật nếu behavior đổi;
- command/evidence được ghi;
- điểm chưa kiểm chứng được nói rõ.

Bàn giao theo format ngắn:

## Changed
- ...

## Verified
- `command`
  - result

## Not verified / limitations
- ...

Không viết báo cáo dài để thay bằng chứng.

---

# 32. KHI TEST ĐỎ

Không lập tức sửa.

Phân loại:

ENVIRONMENT
BASELINE
NEW REGRESSION

Tạo clean/baseline comparison khi cần.

Không báo lỗi của mình nếu baseline đã đỏ.

Không gọi baseline failure là "không sao" nếu nó ảnh hưởng ability to verify task.

---

# 33. KHI PHÁT HIỆN VIỆC NGOÀI SCOPE

Không tự sửa.

Ghi:

> Observed: ...
> Impact: ...
> Suggested follow-up: ...

Rồi tiếp tục task hiện tại nếu không bị block.

---

# 34. ARCHITECTURE CHANGE GATE

Thay đổi cross-cutting/core boundary cần trả lời:

1. Problem gì?
2. Reproducer/business case nào?
3. Current abstraction fail ở đâu?
4. Có cách nhỏ hơn không?
5. Vì sao thuộc shared core?
6. Tenant-specific implementation có đủ không?
7. Compatibility impact?
8. Migration path?
9. Tests?
10. Metric?
11. ADR có cần không?

Nếu chưa trả lời được:

> chưa generalize core.

---

# 35. PROJECT-SPECIFIC SOURCES

Nguồn architecture chính:

- kiến trúc EAIS hiện hành trong `docs/`;
- ADR trong `docs/adr/` hoặc vị trí thực tế của repository;
- contracts trong code;
- tenant packs;
- tests/evals;
- migrations.

Không bịa path nếu repository thực tế khác.

Khi repository đã có canonical path, dùng path thật.

---

# 36. ĐIỂM CHƯA XÁC MINH

Khi thiếu thông tin, dùng bảng:

| Unknown | Impact | How to verify | Blocked work |
|---|---|---|---|

Không cần bảng nếu không có unknown ảnh hưởng task.

Không tuyên bố "toàn bộ hệ thống đã verify" chỉ vì task cục bộ pass.

---

# 37. MỘT DÒNG NHỚ

> SOÁT ≠ LÀM · BẰNG CHỨNG ≠ LỜI KHAI · MODEL ≠ SYSTEM ·
> FACT ≠ GENERATION · SHARED CORE ≠ SHARED DATA ·
> DETERMINISTIC FIRST · AI WHERE UNCERTAINTY EARNS IT ·
> SIMPLE FIRST · SURGICAL CHANGE · MEASURE DON'T GUESS ·
> COMPLEXITY MUST BE EARNED BY EVIDENCE.

---

# 38. CÂU HỎI CUỐI TRƯỚC MỌI THAY ĐỔI

Trước khi thêm code, abstraction, model, agent, RAG strategy, tool, service hoặc dependency:

> Failure cụ thể nào đang buộc EAIS phải có thứ này?

Nếu không trả lời được bằng business case hoặc engineering evidence:

> chưa thêm.