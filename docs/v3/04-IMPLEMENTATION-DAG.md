# 04 — Implementation DAG: thứ tự làm và vì sao

> Trạng thái: bản lập kế hoạch 20/10/2026. Task chi tiết ở `06`, giao model ở `07`.
> Cỡ việc ghi S/M/L theo **phạm vi**, không theo giờ, vì chưa đo tốc độ làm của dự án này:
> - S = một gói, một kết quả;
> - M = 2–3 gói hoặc có test tích hợp;
> - L = nhiều gói, có ranh giới tin cậy hoặc DB.

## 1. Đồ thị

```text
                         ┌──────────────────────────────────────────────┐
                         │ R0 DEFINITION FREEZE                         │
                         │ duyệt 05 · chốt D-V3-01…10 · đổi tên · bản   │
                         │ công khai Blueprint/Playbook                 │
                         └───────┬──────────────────────────┬───────────┘
                                 │                          │
             ┌───────────────────▼─────────┐      ┌─────────▼──────────────────────────┐
             │ R1 FUEL TRUTH (nhánh phụ)   │      │ R2 APP CUSTOMER OPS (không AI)      │
             │ R1-02 OBS-01 · R1-03 B1     │      │ R2-01 ADR-0003 → R2-02 lược đồ/seed │
             │ R1-04 ước chi phí ·         │      │ → R2-03 xác thực/quyền → R2-04 đọc  │
             │ R1-05 đội đỏ · R1-08 soát   │      │ → R2-05 tạo task/idempotency        │
             │ R1-06 chạy model thật ◄─────┼──┐   │ → R2-06 móc bơm lỗi → R2-07 soát    │
             │ R1-07 báo cáo R1            │  │   └──────┬──────────────────────────────┘
             └──────────────┬──────────────┘  │          │    (song song với R2, sau R2-01:
                            │                 │          │     R3-00 / R4-00 soạn bộ ca độc lập)
                            │                 │   ┌──────▼──────────────────────────────┐
                            │                 │   │ R3-01 adapter · R3-02 resolver      │
                            │                 │   │ (dùng chung cho Job A và Job B)     │
                            │                 │   └──────┬───────────────────┬──────────┘
                            │                 │   ┌──────▼─────────┐  ┌──────▼──────────┐
                            │                 │   │ R3 UNDERSTAND  │  │ R4 ACT          │
                            │                 │   │ R3-03…R3-08    │  │ R4-01…R4-08     │
                            │                 │   └──────┬─────────┘  └──────┬──────────┘
                            │                 │          └─────────┬─────────┘
                            │                 │           ┌────────▼────────┐
                            │                 │           │ R5 SECURITY +   │
                            │                 │           │ FAILURE         │
                            │                 │           └────────┬────────┘
                            │      D-V3-01 ───┤           ┌────────▼────────┐
                            │      (model,    └──────────►│ R6 VALUE        │
                            │       ngân sách)            │ → QUYẾT MVP     │
                            │      D-V3-02 (ngưỡng) ─────►└────────┬────────┘
                            │                                      │ WIN/PARTIAL
                            └──────────────► R7 CORE EXTRACTION ◄──┘
                                                   │  (cần R1 đã có kết luận)
                                                   ▼
                                     R8 tích hợp thứ hai → R9 đo portability → R10 FRAMEWORK GATE
```

**Đường găng:**

R0-02 → R2-01 → R2-02 → R2-03 → R2-05 → R3-01 → R3-02 → { R3-03…08 ∥ R4-01…08 } → R5 → R6-01 → **R6-06 (model thật)** → R6-07.

Hai cái chặn từ bên ngoài nằm trên đường găng:
- **D-V3-01** (model + ngân sách), chặn R6-06, smoke model thật của R3 và R1-06;
- **D-V3-02** (ngưỡng), chặn R6-01 và R1-06.

Cả hai cần chốt **trước khi R5 xong**, nếu không đường găng đứng.

## 2. Hai chỗ đổi thứ tự so với Blueprint, và lý do

| Blueprint | Kế hoạch này | Lý do |
|---|---|---|
| R1 → R2 nối tiếp | R1 là **nhánh song song** với R2 | R1 và R2 không phụ thuộc nhau. Bước then chốt của R1 (chạy model thật) bị chặn bởi quyết định bên ngoài (D-V3-01). Bắt R2 chờ là để đường găng đứng không. R7 vẫn đòi R1 có kết luận nên Blueprint không bị phá |
| R3 → R4 nối tiếp | R3 và R4 **song song** sau phần chung R3-01/R3-02 | Job B cần adapter và resolver, không cần phần tổng hợp brief của Job A. Viết nối tiếp làm R4 chờ vô cớ. Hai nhánh ghi vào hai gói khác nhau (`understand/`, `act/`), không giao tệp |
| Bộ ca soạn trong R3/R4 | Bộ ca soạn **từ sau R2-01**, song song với R2 | Ca chỉ cần hợp đồng API và đặc tả fixture, không cần mã. Ca phải đông cứng **trước** mã R3/R4 để cổng không viết theo mã (`docs/04` §6 W-G6, `CLAUDE.md` §1) |

## 3. Nút theo phase

Mỗi nút đủ các trường prompt điều phối yêu cầu. Điều kiện PASS gốc: Playbook §9.

### R0 — Definition Freeze
- **Mục tiêu:** một gói định nghĩa có revision; 0 blocker định nghĩa.
- **Story:** là chủ dự án, tôi muốn mọi phiên/agent dùng cùng một định nghĩa "thắng", để không ai tự tuyên bố xong theo cách hiểu riêng.
- **Tiền điều kiện:** nguồn V3 đã chụp và băm (TASK-R0-01, xong).
- **Phụ thuộc:** không.
- **AC:**
  - `05` §5 đủ ô;
  - D-V3-01…10 có quyết định ghi lại;
  - tài liệu mới chỉ dùng W-G* cho cổng cũ;
  - bản công khai không chứa tên hệ thống của công ty cũ.
- **DoD:** revision của gói ghi trong `09`; `grep` tên riêng trên `docs/v3/` ra 0.
- **Test:** script kiểm bản công khai khớp bản gốc (trừ chỗ ẩn danh và chỗ sửa định dạng đã liệt kê).
- **Bằng chứng:** diff + kết quả script + quyết định của chủ dự án.
- **Rủi ro:** R-01 (phình phạm vi), R-14 (lệch tài liệu và runtime).
- **Cỡ:** S. **Model:** lead (Opus) cho quyết định; Haiku cho việc chép. **Song song:** có (R0-03, R0-04).
- **Chặn:** mọi phase. **Bị chặn bởi:** chủ dự án duyệt.

### R1 — Fuel Truth Benchmark
- **Mục tiêu:** câu hỏi "máy móc semantic/authority có đáng độ phức tạp không" có câu trả lời WIN/PARTIAL/FAIL bằng model thật, so B1 và B2.
- **Story:** là kiến trúc sư, tôi muốn biết tầng sự thật có thắng LLM trực tiếp và RAG đơn giản không, để quyết giữ hay gọt.
- **Tiền điều kiện:** mốc `02` xanh.
- **Phụ thuộc:** R0. R1-06 cần D-V3-01 và D-V3-02.
- **AC:** V3-FUEL-01…07 (`03` §5).
- **DoD:** báo cáo có số thô, revision, model, chi phí, phương pháp, thất bại, kết luận.
- **Test:** hồi quy Fuel; test B1 bằng model giả; ước chi phí; chạy thật k = 3.
- **Bằng chứng:** EV-R1-* trong `09`.
- **Rủi ro:** R-07 (chi phí), R-11 (gian lận benchmark), R-15 (model không có).
- **Cỡ:** M. **Model:** Sonnet cài đặt; Sonnet soát độc lập; lead duyệt phương pháp. **Song song:** có, với R2.
- **Chặn:** R7. **Bị chặn bởi:** D-V3-01, D-V3-02 (chỉ R1-06/07).

### R2 — App tham chiếu Customer Operations (không AI)
- **Mục tiêu:** một ứng dụng nghiệp vụ nhỏ, đúng, **đứng độc lập**, có tenant, xác thực, quyền, idempotency, API thân thiện audit.
- **Story:** là kỹ sư, tôi muốn phần mềm gốc đúng trước khi thêm AI, để lỗi AI không lẫn với lỗi nghiệp vụ.
- **Tiền điều kiện:** ADR-0003 được duyệt.
- **Phụ thuộc:** R0.
- **AC:** V3-APP-01…07.
- **DoD:** bộ tích hợp tất định xanh từ môi trường sạch; soát độc lập không còn phát hiện mức chặn; 0 import giữa app và EACL.
- **Test:** tích hợp với PG; cho phép/từ chối; chéo tenant; idempotency; bơm lỗi.
- **Bằng chứng:** EV-R2-*.
- **Rủi ro:** R-08 (sân nhà), R-01.
- **Cỡ:** L. **Model:** lead làm R2-01; Sonnet cài đặt; Sonnet soát độc lập. **Song song:** trong R2 thì nối tiếp (cùng tệp app); song song với R1 và với việc soạn ca R3-00/R4-00.
- **Chặn:** R3, R4. **Bị chặn bởi:** R0.

### R3 — UNDERSTAND (Job A)
- **Mục tiêu:** brief 30 ngày có bằng chứng, đạt G1/G2/G3/G5 trên bộ ca đông cứng.
- **Story:** US-001.
- **Tiền điều kiện:** R2 PASS; bộ ca Job A đã đông cứng (R3-00).
- **Phụ thuộc:** R2.
- **AC:** V3-AC01…06, V3-AC12, V3-SEC-01/02/04/07.
- **DoD:** US-001 DoD (Playbook §10).
- **Test:** model giả đối kháng (tất định); smoke model thật nếu có ngân sách.
- **Bằng chứng:** EV-R3-*.
- **Rủi ro:** R-04 (ảo giác), R-12 (ngữ cảnh lộ quyền), R-16 (uốn Customer Ops theo pipeline Fuel).
- **Cỡ:** L. **Model:** Sonnet cài đặt; lead soát context builder + validator; Sonnet soát độc lập. **Song song:** với R4 sau R3-02.
- **Chặn:** R5. **Bị chặn bởi:** R2.

### R4 — ACT (Job B)
- **Mục tiêu:** follow-up có kiểm soát, đạt G4/G5 kể cả khi bơm lỗi.
- **Story:** US-002.
- **Tiền điều kiện:** R2 PASS; ADR-0004 được duyệt; bộ ca Job B đã đông cứng (R4-00).
- **Phụ thuộc:** R3-01, R3-02.
- **AC:** V3-AC07…13, V3-SEC-03/06/09/10/11, V3-FAIL-05/06/08–12.
- **DoD:** US-002 DoD.
- **Test:** happy path; từ chối quyền; đề xuất hỏng; thử lại trùng; mất phản hồi; trạng thái cũ; đổi quyền giữa chừng.
- **Bằng chứng:** EV-R4-*.
- **Rủi ro:** R-09 (hành động trùng), R-06 (vượt quyền), R-10 (trạng thái cũ).
- **Cỡ:** L. **Model:** lead làm ADR-0004 + soát an ninh; Sonnet cài đặt; Sonnet soát độc lập. **Song song:** với R3.
- **Chặn:** R5. **Bị chặn bởi:** R3-01/02, ADR-0004.

### R5 — Security + Failure Injection
- **Mục tiêu:** không bất biến kiểm soát nào phụ thuộc vào việc model tuân thủ.
- **Story:** là chủ dự án, tôi muốn thấy hệ vẫn an toàn khi AI sai, bị tấn công, timeout, thử lại.
- **Tiền điều kiện:** R3, R4 có test đạt.
- **Phụ thuộc:** R3, R4.
- **AC:** V3-FAIL-01…12, V3-OBS-02.
- **DoD:** bộ ca chặn xanh; rủi ro còn lại có tài liệu.
- **Test:** đội đỏ độc lập; bơm lỗi transport.
- **Bằng chứng:** EV-R5-*.
- **Rủi ro:** R-05 (chèn lệnh), R-06.
- **Cỡ:** M. **Model:** Sonnet đội đỏ (không đọc mã); Sonnet cài bơm lỗi; lead soát. **Song song:** một phần (soạn ca ∥ cài móc).
- **Chặn:** R6. **Bị chặn bởi:** R3, R4.

### R6 — Baseline + Value Benchmark
- **Mục tiêu:** kết luận MVP WIN/PARTIAL/FAIL theo ngưỡng đông cứng từ trước.
- **Story:** là chủ dự án, tôi muốn biết EACL có đáng hơn cách đơn giản không.
- **Tiền điều kiện:** R5 PASS; D-V3-01, D-V3-02 đã chốt; phương pháp đã đông cứng (băm).
- **Phụ thuộc:** R5.
- **AC:** V3-AC14/15, V3-VAL-01…03, V3-COST-01/02.
- **DoD:** Playbook R6 DoD.
- **Test:** chạy B0 (mẫu), B1, B2, B3 trên cùng bộ ca.
- **Bằng chứng:** EV-R6-*.
- **Rủi ro:** R-11, R-07, R-17 (B0 một người).
- **Cỡ:** M. **Model:** Sonnet harness; Haiku đo công tích hợp; lead viết kết luận. **Song song:** harness ∥ đo công.
- **Chặn:** R7. **Bị chặn bởi:** D-V3-01, D-V3-02, R5.

### R7–R10 (chỉ định nghĩa; lập task khi có kết luận R6)
- **R7 — Core extraction.**
  - Mục tiêu: so Fuel với Customer Ops, chỉ nâng hợp đồng thật sự dùng chung.
  - Bị chặn bởi: R6 có kết luận WIN/PARTIAL; R1 có kết luận.
  - DoD: hai bộ test vertical xanh sau khi tách; ADR cho mỗi trừu tượng được nâng.
- **R8/R9 — Portability.**
  - Mục tiêu: domain thứ ba khác hẳn; đo dòng mã domain, số thay đổi vào core, công ánh xạ, thời gian tới kết quả hữu ích đầu tiên.
  - Ngưỡng: đông cứng trước khi bắt đầu.
  - Ứng viên: thể thao (sự kiện thời gian thực, đã có tài liệu ở `docs/tenants/sports/`), soát hồ sơ mua sắm, phân loại sự cố vận hành.
- **R10 — Framework Gate.** Đạt ⇒ ứng viên SDK. Trượt ⇒ giữ kiến trúc tham chiếu. Cả hai đều là kết quả chấp nhận được.

## 4. Song song an toàn (tối đa 3 agent con cùng lúc, tệp không giao nhau)

| Nhóm | Agent 1 | Agent 2 | Agent 3 |
|---|---|---|---|
| Đợt 1: sau khi duyệt R0 | R2-01 do lead làm (không phải agent con) | R1-02 → R1-03 → R1-04 (Sonnet; `tenants/construction`, `evals`) | R0-03 → R0-04 (Haiku; chỉ `docs/`) |
| Đợt 2: sau R2-01 | R2-02 → R2-07 (Sonnet; chỉ `examples/customerops/`) | R3-00 + R4-00 soạn ca (Sonnet độc lập; chỉ `tenants/customerops/evals/`) | R1-05 đội đỏ Fuel (Sonnet độc lập; `tenants/construction/win/` tệp ca mới) hoặc R1-08 soát (chỉ đọc) |
| Đợt 3: sau R2 PASS | R3-01 → R3-02 (Sonnet) | soát độc lập R2/R3-0x (Sonnet chỉ đọc) | — |
| Đợt 4 | R3-03…R3-08 (Sonnet; `understand/`) | R4-02…R4-07 (Sonnet; `act/`) | soát độc lập luân phiên (chỉ đọc) |

Luật đợt:
- người viết mã không phải người soát (`CLAUDE.md` §2.1);
- mỗi agent nhận đúng danh sách tệp được ghi;
- lead đo lại lời khai bằng lệnh, không tin báo cáo của agent.

## 5. Học theo phase (cho chủ dự án)

Mỗi phase: **một** câu First Principles, **một** việc trên repo thật, **một** lệnh, **một** tiêu chí "hiểu rồi".
- Lệnh có ghi "đã chạy 20/10" là lệnh đã chạy thử khi lập kế hoạch.
- Phase chưa có mã thì trỏ tới task sẽ sinh ra thứ để học.

| Phase | Câu hỏi gốc | Việc | Lệnh | Hiểu rồi khi |
|---|---|---|---|---|
| R0 | Ta giải lỗi nghiệp vụ nào, "thắng" nghĩa là gì? | đọc `05` §2.5 rồi tự chốt D-V3-02 | — | với mỗi ngưỡng, nói được "nếu đặt sai số này thì ta tự lừa mình kiểu gì" |
| R1 | DATA ≠ FACT ≠ BUSINESS MEANING ≠ AUTHORITY | đọc `tenants/construction/pack/metrics.yaml` (dòng 1–48: bốn chỉ số là bốn nghĩa, thẩm quyền chỉ so trong cùng chỉ số, dung sai ISSUED 10 L) và `intents.yaml` (ý định `fuel_issued`: "được cấp" → `issued_fuel`), rồi chạy ca WIN-I-110 | `go run ./tenants/construction/win/gen/cmd/gen -world tenants/construction/win/world.yaml -out /tmp/win -profile functional -seed 42 && go run ./tenants/construction/win/winreport -data /tmp/win -v \| grep -A1 WIN-I-110` (đã chạy 20/10) | giải thích bằng chính dòng YAML vì sao: (1) trả 3.850 L (phiếu kho hạng 1); (2) ERP 3.900 L bị nêu là lệch (50 L > dung sai 10 L); (3) 3.782 / 4.000 / 4.100 chỉ là "số khác cùng kỳ" (khác chỉ số, không phải xung đột). Rồi tự hỏi: từ khoá "nhận" cũng ra `issued_fuel`; có câu nào "nhận" mang nghĩa khác không? |
| R2 | Bỏ AI đi, phần mềm gốc có đúng không? | khi R2-05 xong: chạy test đua 10 yêu cầu cùng `Idempotency-Key` | tên test chốt ở R2-05 | nói được vì sao chỉ có 1 dòng (UNIQUE theo tenant + khoá, so `request_hash`) và vì sao khác nội dung thì trả 409 |
| R3 | AI giúp hiểu, nhưng khẳng định nào phải có bằng chứng? | khi R3-06 xong: cho model giả trả một gạch không có `evidence_ids` | runner của R3-08 | thấy validator chặn, đọc được trace ghi vì sao |
| R4 | AI đề xuất, phần mềm quyết | khi R4-04 xong: chạy ca "ghi xong mất phản hồi" (V3-FAIL-10) | test của R4-07 | vẽ lại được EXECUTING → UNKNOWN → đọc lại theo khoá → VERIFIED, và chỉ ra chỗ nào bảo đảm chỉ có 1 task |
| R5 | Khi mọi thứ sai thì sao? | đọc ca chèn lệnh đang có của Fuel | `go test ./tenants/construction/ -run TestChenLenhTrongTaiLieuKhongDoiDuocGi -v -count=1` (đã chạy 20/10: PASS) | nói được vì sao lệnh nằm trong SOP không đổi được route, quyền hay con số: chúng được quyết **trước** và **ngoài** model |
| R6 | Có đáng hơn cách đơn giản không? | đọc `evals/baseline/baseline_test.go` | `go test ./evals/baseline/ -run TestBaselineFailsCasesThatNeedControlLayers -v -count=1` (đã chạy 20/10: PASS) | nói được baseline trượt ca nào vì thiếu tầng kiểm soát, **và** vì sao đó chưa phải bằng chứng giá trị: model giả, chưa có B1, chưa có chi phí |
