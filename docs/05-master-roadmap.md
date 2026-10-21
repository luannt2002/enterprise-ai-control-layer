# 05 — EAIS Master Roadmap

> **Historical artifact** (D-V3-10, chủ dự án duyệt 20/10/2026): roadmap hiện hành ở `docs/v3/`. Giữ tệp này để truy nguồn gốc quyết định; không lập kế hoạch mới theo nó.
>
> **Từ tối 21/10/2026 (D-RM-01):** roadmap hiện hành là `docs/ROADMAP-DAY-DU.md`, dựng trên Phase 0–10 của tệp này; `docs/v3/` chỉ còn để tra cứu. Lập kế hoạch theo `docs/ROADMAP-DAY-DU.md`, không theo tệp này.
>
> **Mô tả EAIS có thể tiến hoá tới đâu, với điều kiện năng lực trước đó đã giành được quyền tồn tại.**
>
> Trạng thái: hướng đi đã chốt theo user ngày 19/10/2026 ("làm cái win trước, đạt rồi mới mở rộng ra roadmap đầy đủ"). Từng phase là **giả thuyết có điều kiện**, không phải danh sách việc.

## 1. Ba câu hỏi, hai tài liệu

| Câu hỏi | Tài liệu | Vai trò |
|---|---|---|
| EAIS **là gì**? Bỏ thứ gì đi thì mất bản chất? | **CORE**, mục 2 của `04-core-mvp.md` | bản sắc, bất biến |
| Bằng chứng **nhỏ nhất** cho thấy CORE tạo ra giá trị là gì? | **WIN MVP**, `04-core-mvp.md` | trạm kiểm (checkpoint), hợp đồng chống phình phạm vi |
| Thắng rồi thì EAIS **có thể** thành gì? | **MASTER ROADMAP**, tài liệu này | bản đồ, chống thiển cận |

- Có bản đồ mà không có trạm kiểm thì phạm vi phình mãi. Có trạm kiểm mà không có bản đồ thì tối ưu cục bộ.
- WIN MVP không phải roadmap thu nhỏ. Mỗi vòng sau phải chứng minh **thêm một lời khẳng định mới**.

## 2. Luật của roadmap

1. **Roadmap không phải TODO list.** Mỗi phase chỉ mở khi đủ **hai vế**: (a) các phase tiên quyết ở cột "Mở khi" đã đạt; (b) có lỗi đo được hoặc ca nghiệp vụ đòi năng lực mới. Thiếu một vế thì DỪNG hoặc HOÃN. Một phase "đạt" khi lời khẳng định ở cột 2 được chứng minh bằng cổng nhị phân, theo kiểu `04` §6; phase tự định nghĩa cổng của nó lúc mở.
2. **WIN thất bại thì không đi tiếp roadmap.** Quay lại sửa luận đề hoặc core.
3. **Roadmap không kéo ngược tính năng vào MVP.** Roadmap có Agent, Sports, Kafka không có nghĩa MVP cần chúng.
4. **MVP không giới hạn tầm nhìn.** "MVP chưa cần reranker" chỉ nghĩa là reranker *chưa giành được* độ phức tạp của nó, không phải "EAIS không bao giờ cần".
5. **Thiết kế cho hướng đã biết, chỉ cài thứ đã chứng minh cần.** Ví dụ: không có `if tenant == …` trong core vì Phase 4 sẽ đòi tính di động; nhưng chưa xây domain thứ hai (WIN chỉ có canary B cùng ngành).
6. **Mỗi năng lực chỉ có 5 trạng thái** (MOVE / DECOUPLE / REPLACE là *động từ di trú* của `docs/02`, không phải trạng thái): KEEP (giữ thứ đã giải được) · FIX (sửa cài đặt, khái niệm vẫn đúng) · BUILD (có bằng chứng thì xây) · DEFER (chưa đủ bằng chứng hoặc ROI) · REMOVE (độ phức tạp không chứng minh được giá trị).
7. **Cổng toàn vẹn khái niệm ("ai sở hữu?")** trước khi thêm abstraction. Phải trả lời được: ai sở hữu nó · trạng thái gì · nguồn sự thật nào · ai ghi · ai đọc · vòng đời ra sao · hỏng thế nào · phục hồi thế nào · ranh giới bảo mật ở đâu · kiểm chứng ra sao. Ví dụ: Router sở hữu định tuyến, workflow sở hữu trạng thái, policy sở hữu quyền, nguồn có thẩm quyền sở hữu sự thật. Vậy **Agent sở hữu gì?** Không trả lời được thì không thêm. Hai thành phần cùng quyết một việc là xung đột thẩm quyền.

## 3. Các phase

| Phase | Lời khẳng định phải chứng minh | Mở khi | Năng lực được thêm |
|---|---|---|---|
| **0 — Core Truth (WIN MVP) ← ĐANG LÀM** | dữ liệu doanh nghiệp bẩn → câu trả lời nghiệp vụ đáng tin, truy vết được, hơn baseline RAG | — | xem `04-core-mvp.md` |
| 1 — Controlled Action (MVP-1) | không chỉ trả lời đúng mà còn **hành động có kiểm soát** ("tạo phiếu yêu cầu kiểm tra chênh lệch EX-03") | WIN 7/7 + ca nghiệp vụ cần hành động | command · tool · quyền + policy · kiểm nghiệp vụ · idempotency · audit |
| 2 — Workflow (MVP-2) | quản được **quy trình**, không chỉ một yêu cầu một lần: phát hiện chênh lệch → điều tra → tổ trưởng duyệt → kho xác minh → đối soát → duyệt/từ chối → đóng | Phase 1 đạt + một hành động lẻ không đủ | máy trạng thái · chuyển trạng thái · duyệt · người trong vòng · phục hồi · trạng thái dài hạn |
| 3 — Event | **phản ứng** khi doanh nghiệp thay đổi: chênh lệch vượt ngưỡng → sự kiện → luật → workflow | Phase 2 đạt + ca cần hệ tự kích hoạt | nạp sự kiện · event store · xử lý bất đồng bộ · thứ tự · chống trùng · idempotency |
| 4 — Portability (MVP-3) | core **thật sự độc lập** với ngành xây dựng | Phase 0–2 đạt + ca nghiệp vụ cần domain thứ hai | domain thứ hai đủ khác (thể thao); cổng: số dòng sửa trong core ≈ 0, đo bằng `git diff --stat <commit mốc ngay trước khi bắt đầu domain thứ hai>..HEAD -- <các thư mục dùng chung của architecture_test.go>`, chỉ tính commit của nhánh domain thứ hai. Sửa phải nằm ở pack / adapter / schema / eval. Thành phần nào bị tenant thứ hai buộc sửa thì chưa được gọi là core đã chứng minh |
| 5 — Platform hardening | vận hành được như một nền tảng | Phase 4 đạt + nhu cầu vận hành thật | vòng đời tenant và pack · phiên bản cấu hình · migration · bí mật · hạn mức · ngân sách chi phí · SLA · sao lưu/phục hồi · triển khai · HTTP API đầy đủ · công cụ quản trị |
| 6 — Advanced intelligence | AI nâng cao **tốt hơn đo được** so với baseline đơn giản | có lỗi đo được mà luật / truy xuất cơ bản không giải được | ML entity resolution · ML intent · truy xuất nâng cao · rerank · GraphRAG · knowledge graph · model local · vision · fine-tune |
| 7 — Agentic | có bước tiếp theo mà workflow cố định không xác định nổi một cách hợp lý | Phase 2 + ca chứng minh | agent có biên tự chủ · lập kế hoạch · chọn công cụ động · eval agent. **Agent ≠ thẩm quyền ≠ nguồn sự thật ≠ bộ máy phân quyền ≠ chủ trạng thái workflow** |
| 8 — Integration standardization | hệ sinh thái tích hợp đủ lớn để cần khám phá và liên thông | số tích hợp / bên dùng thật đòi | MCP (không phải "có MCP nên phải dùng") |
| 9 — Enterprise scale | **đúng trước, nhanh sau**: chịu tải thật | đúng nghĩa đã chứng minh + số đo cho thấy đường đồng bộ không đạt | tải · đồng thời · tenant lớn · kho tri thức lớn · HA / DR · Redis · hàng đợi · Kafka · worker phân tán · sharding (mở theo số đo) |
| 10 — Full vision | QUERY + COMMAND + EVENT trên nhiều domain | các phase trước đạt + nhu cầu | toàn cảnh như `01-architecture.md` |

Riêng Phase 6: mỗi năng lực đi đủ chuỗi lỗi quan sát được → ca tái hiện → baseline → ứng viên → benchmark → tốt hơn thì KEEP, không thì REMOVE.

Tầm nhìn đích (Phase 10):

```text
QUERY · COMMAND · EVENT → ROUTER → DATA (SQL/fact) · KNOWLEDGE (RAG) · ACTION (tool/API)
  → CONTEXT → MODEL (khi cần) → VALIDATION → WORKFLOW/STATE → BUSINESS OUTCOME
CONTROL PLANE: tenant · danh tính · quyền · policy · thẩm quyền nguồn · kiểm · bảo mật · trace · audit · eval · chi phí
TENANT LAYER:  Construction │ Sports │ Domain C │ …
```

## 4. Thứ đã có nằm ở đâu trên bản đồ

| Thứ đã có | Phase |
|---|---|
| `01-architecture.md` | giả thuyết của Phase 10, **không phải phạm vi MVP** |
| `03-proof-harness.md` (thang bằng chứng, luật tuyên bố) | áp cho **mọi** phase |
| `docs/tenants/sports/**`, `tenants/sports/evals/` (80 ca, 14 kịch bản cố ý đỏ) | Phase 4 (DEFER), giữ làm đầu vào |
| Ca COMMAND / WORKFLOW / EVENT trong `tenants/construction/evals/cases.yaml` | Phase 1–3 (DEFER) |
| Tác vụ duyệt xung đột, lệnh chốt (bản 04 cũ, lát M3) | Phase 1–2 |
| Hồ sơ quy mô lớn (1 triệu+ fact) | Phase 9. WIN chỉ có *scale smoke* |
| Chiều sâu mô hình dữ liệu: ontology, song thời / đính chính muộn, hiệu lực thực thể theo kỳ, truy vấn tại thời điểm T, phân quyền mức trường | Phase 5 (DEFER có điều kiện: một ca nghiệp vụ hoặc ca WIN đỏ vì thiếu) |
| Kiểm tuân thủ định mức ("vượt định mức không?") | Phase 1 |
| Bộ đọc Python, bảng PDF, OCR | Phase 6 |
| Đa agent | Phase 7 |
| Kubernetes, tách microservice | Phase 9 |
| `docs/requirements/catalog.yaml` (`phase` do agent dán: MVP 294…) | phải xếp lại theo WIN / Phase 1–10 |

## 5. Quan hệ giữa các tài liệu

```text
05-master-roadmap.md  "Ta có thể đi tới đâu?"
├── Phase 0 ← HIỆN TẠI
│     └── 04-core-mvp.md  "Thứ nhỏ nhất phải thắng là gì?"
├── Phase 1 Action · 2 Workflow · 3 Event · 4 Portability
├── Phase 5 Platform · 6 Advanced AI · 7 Agent · 8 MCP
└── Phase 9 Scale · 10 Full vision
```
