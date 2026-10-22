# Cổng QuânIT — Phase 1: tạo một phiếu kiểm tra chênh lệch nhiên liệu

> **Trạng thái: GIẤY, chưa có mã.**
> - Mã Giai đoạn II chỉ mở sau khi chạy xong H4: D-RM-04 = A (`docs/CHO-QUYET-DINH.md` mục 0, dòng 22/10 11:2x).
> - H4 đã chạy MỘT lần ngày 22/10 14:59 ở `57409f2` (104/119 hai đường; `docs/STATUS.md`). Dòng S4 của CỔNG TRƯỚC AI KHÔNG ĐẠT, chờ D-AI-02 [REPO, `docs/STATUS.md`]: dòng đó chặn M3 và M5, không chặn điều kiện mở mã M4; Phase 1 không đi qua S4.
> - Cập nhật 22/10/2026 theo bốn ghi chú L3C-01…04 của lần soát lại (xem đầu `HOP-DONG.md`): cửa vào `cmd/eais`, vết của lượt lệnh, nơi đặt mã dưới `execution/commands`. Dấu chấm của chín câu **không đổi**.
> - Soát: lượt 3 (22/10/2026) ĐẠT CÓ GHI CHÚ, mười hai ghi chú L3D-01…12 (xem đầu `HOP-DONG.md`). Tệp này sửa L3D-03 (câu 7: `invalid_fields`, `detail`) và L3D-06 (đầu tệp: dòng S4 của CỔNG TRƯỚC AI; mục 1 và câu 1: làn 0 đã gộp); các dòng "H4 đã chạy" và cổng gốc điều kiện 1 (mục 5) đã đúng từ bản trước; không sửa mục nào khác; **phần sửa này chưa được soát lại**. Dấu chấm của chín câu vẫn **không đổi**.
> - Lượt 4 (soát bản sửa L3D, 22/10/2026 tối): ĐẠT CÓ GHI CHÚ (L3E-01…13); điều phối áp đề xuất của người soát cho L3E-01…04, 06, 08, 09, 12 khi gộp (xem đầu `HOP-DONG.md`), L3E-05, 07, 10, 11, 13 ghi nợ.
>
> **Viết cho:**
> - chủ dự án: duyệt hoặc bác từng câu chấm;
> - phiên BUILD: làm theo `docs/viec/1-hanh-dong/HOP-DONG.md` và `docs/adr/0011-phase1-hanh-dong-co-kiem-soat.md` khi mã được mở.
>
> **Ngày:** 22/10/2026. **Mốc đọc mã:** `c086fe7`. Tra một dòng trích bằng `git show c086fe7:<tệp>`. Nhánh `main` có thể đã đổi các tệp được trích (ví dụ `ask.go`, `router.go`, `intent.go`, pack), nên số dòng ở `main` có thể lệch; khi gộp phải đối chiếu lại các dòng trích.
>
> **Nhãn bằng chứng:**
> - [ĐO]: người viết tự chạy lệnh, ghi kèm lệnh;
> - [REPO]: tài liệu repo tự ghi, chưa chạy lại;
> - [CODE]: đọc thấy trong mã, `tệp:dòng` tại `c086fe7`;
> - [SUY]: suy luận của người viết;
> - [NGOÀI]: nguồn công khai ngoài repo, đọc ngày 22/10/2026;
> - "chưa đo": chưa chạy lệnh nào.
>
> **Cách chấm.** Phase 1 chưa có mã, nên dấu ở đây chỉ nói về GIẤY:
> - ✓ = tiêu chí của cổng (`docs/ROADMAP-DAY-DU.md` §4.1, cột "Qua cổng khi") đã được viết ra và có nguồn. Chưa có nghĩa là mã đã chứng minh; phần chứng minh bằng mã nằm ở cổng ra (`HOP-DONG.md` §11).
> - ◐ = viết được một phần, còn thiếu điều kiện đã nêu.
> - ✗ = không thể viết cho đủ nếu thiếu người hoặc dữ kiện bên ngoài repo.
> - **Chuẩn riêng của người viết, chặt hơn cột "Qua cổng khi" ở ba câu (1, 3, 9).** Tiêu chí của cột đã được viết ra và có nguồn, nhưng người viết vẫn không chấm ✓ vì còn thiếu điều kiện thật: câu 1 (người dùng là giả định, chưa có số nền); câu 3 (lỗi im lặng L1 còn mở, chưa có cổng kiểm mức sẵn sàng dữ liệu); câu 9 (mới so một giải pháp bằng lập luận, chưa thử thật). Chủ dự án có thể chấm lại theo đúng chữ của cột.
> - **✓ ở đây là "viết ra, có nguồn" (giấy).** Khác ✓ của Fuel ở `docs/ROADMAP-DAY-DU.md` §7, là "đã đo [ĐO]". Không đặt hai bên cạnh nhau như một thước.
>
> **Dữ liệu:** mọi tên máy, dự án, con số trong tài liệu này là giả lập (tenant `abc-infra` hư cấu). Mọi kết luận: **trên dữ liệu giả lập, chưa có người nghiệp vụ duyệt** (D-RM-03).

---

## 0. Kết luận

- **9 câu, trên giấy:** ✓ 5 (câu 2, 4, 5, 6, 7) · ◐ 2 (câu 3, 9) · ✗ 2 (câu 1, 8).
- **Hai câu ✗ là hai câu Phase 0b (Fuel) cũng trượt** (`docs/ROADMAP-DAY-DU.md` §7, câu 1 và 8): không có người dùng thật, không có người ký độ chính xác. Nguyên nhân gốc là D-RM-03, chủ dự án đã trả lời "chưa có" ngày 22/10.
- **Hệ quả** [SUY, theo `docs/ROADMAP-DAY-DU.md` §9]:
  - không chặn việc viết giấy, và không chặn mã khi D-RM-04 cho phép;
  - mọi kết luận về Phase 1 ghi "trên dữ liệu giả lập, chưa có người nghiệp vụ duyệt";
  - không tuyên bố Phase 1 "đáng giá": phép đo giá trị chưa có số nền (câu 1).
- **Cổng gốc** (`docs/ROADMAP-DAY-DU.md` §4.2): xem §5. Điều kiện 1 (phase tiên quyết) **đã đạt**: H4 đã chạy một lần (22/10 14:59).

---

## 1. Việc được chấm

> Một người có quyền yêu cầu EAIS mở **đúng một** phiếu kiểm tra cho **một** chênh lệch nhiên liệu của **một** máy trong **một** ngày. EAIS tự tính chênh lệch từ số liệu có thẩm quyền, kiểm quyền, chống tạo trùng, đọc lại để xác nhận phiếu có thật, rồi ghi audit.

- **Nguồn của việc:**
  - `docs/ROADMAP-DAY-DU.md` §5 luồng L3 và §6, bảng phase, dòng Phase 1;
  - `docs/05-master-roadmap.md:37` (câu gốc: "tạo phiếu yêu cầu kiểm tra chênh lệch EX-03");
  - `docs/ROADMAP-DAY-DU.md` §5b, khối M4 của MVP (D-MVP-00; đọc ở `main` sau mốc `c086fe7`): Phase 1 là phần tạo phiếu của chuỗi "phát hiện chênh lệch → tạo phiếu" (`HOP-DONG.md` §1.4); kịch bản demo hai lượt của khối M4: `HOP-DONG.md` §2.6.4;
  - ca nghiệp vụ CON-S-026 (`docs/tenants/construction/05-problem-catalog.md` mục 3), CON-T-010 (`tenants/construction/evals/cases.yaml:314`), ca hỏng CON-F-020 (`docs/tenants/construction/07-failure-catalog.md:76`).
- **Chế độ B** (EAIS giữ sự thật, quyền, giao dịch): doanh nghiệp chỉ có Zalo, Excel, giấy nên chưa có hệ phiếu nào. Chế độ A (tạo task trên CRM của khách) là Phase 4b, không thuộc đây.
- **Chênh lệch** là chỉ số dẫn xuất `issue_vs_measured` = dầu kho xuất − dầu tiêu thụ đo được (`tenants/construction/pack/metrics.yaml:50-51`) [CODE].
- **Ví dụ giả lập:** máy EX-03 trong một ngày: kho xuất 3.850 L, cảm biến đo 3.782 L, chênh lệch 68 L. Hai vế lấy theo ví dụ của `CLAUDE.md` §20; test `tenants/construction/construction_test.go:105-106` (ý định `fuel_variance`, route `CALCULATION`) kiểm câu trả lời chứa "68" [CODE]. Đây là dữ liệu hạt giống giả lập, không phải số của doanh nghiệp nào.
- **Hiện trạng:** 0 dòng mã thi hành lệnh. Thư mục `execution/`, `governance/`, `domain/` chưa có tệp nào được git theo dõi [ĐO `git ls-files | grep -E '^(execution|governance|domain)/' | wc -l` → 0 tại `269a539`]; `cmd/` có 10 tệp từ khi làn 0 gộp (`cmd/eais`: ứng dụng HTTP và CLI, không thi hành lệnh; `HOP-DONG.md` §1.3) [ĐO `git ls-files cmd/ | wc -l` → 10 tại `269a539`]. `EntryCommand` và route `TOOL` chỉ được khai báo (`core/contracts/contracts.go:196`, `:210`); route `TOOL` rơi vào nhánh lỗi `core/pipeline/ask.go:187-190` [CODE]. 17 ca COMMAND tồn tại dạng dữ liệu, chưa có bộ chạy [ĐO `grep -c "entry_type: COMMAND" tenants/construction/evals/cases.yaml` → 17].
- **Đường hỏi coi câu lệnh là câu hỏi** [ĐO, test thăm dò tạm không commit, tại `c086fe7`, `1e5457b` và `7d72e33`; chi tiết `HOP-DONG.md` §1.5]: "Tạo phiếu kiểm tra chênh lệch máy EX-03 hôm qua" được trả lời như câu hỏi tính toán (route `CALCULATION`, ý định `fuel_variance`): "Chưa có dữ liệu Chênh lệch dầu kho xuất và dầu tiêu thụ đo được của Máy xúc EX-03 ngày 18/10/2026."; cùng câu với "hôm nay" cho "68 L". Động từ "Tạo phiếu" bị bỏ qua im lặng: người dùng không được báo là chưa có phiếu nào. Cùng họ với CON-F-020 nhưng chưa có lời "đã tạo". Chặn ở đường hỏi nằm ngoài Phase 1 (nợ cho S4 hoặc Phase 6).
- **Ánh xạ CON-S-026 / CON-T-010 sang Phase 1** (không trùng khít: hai ca này thuộc domain bảo trì thiết bị, Phase 1 là chênh lệch nhiên liệu). Thuộc tính của CON-T-010 (`tenants/construction/evals/cases.yaml:314-337` tại `c086fe7`) và cách Phase 1 xử lý:

  | Thuộc tính của CON-T-010 | Phase 1 |
  |---|---|
  | domain `maintenance`, quy trình `equipment-maintenance`; đầu vào chữ tự do "Tạo phiếu kiểm tra máy EX-03" | domain nhiên liệu; đầu vào có cấu trúc {máy, chỉ số, ngày}; **không** phủ câu chữ tự do |
  | `expected_route: WORKFLOW`; `expected_tools: [start_workflow, assign_task]` | không có workflow, không có `assign_task`: pipeline lệnh C0–C7 và bảng phiếu riêng (ADR-0011 mục B1); hoãn Phase 2 |
  | `expected_state: {from: null, to: OPEN, human_task: assign_technician}` | **đáp ứng một phần**: phiếu mới ở trạng thái `OPEN`; `human_task` giao người nhận hoãn Phase 2 |
  | `expected_output_properties`: trả mã phiếu mới và trạng thái `OPEN`; báo rõ phiếu đã được tạo | **đáp ứng**: `DONE` kèm mã phiếu và `OPEN` (`HOP-DONG.md` §2.3), chỉ sau khi đọc lại thấy phiếu (I3) |
  | `must_not_do`: không trả lời "đã tạo" khi chưa có bản ghi `workflow_instances` | **tương đương**: không `DONE` khi chưa đọc lại thấy phiếu trong `discrepancy_tickets` (I3); Phase 1 không có `workflow_instances` |
  | `preconditions`: chưa có phiếu đang mở; idempotency key mới | R5 (không trùng) và `command_id` (`HOP-DONG.md` §6) |
  | `permissions: {principal: equipment_manager}`; `business_rules: [BR-MNT-02]` | vai `equipment_manager` không có trong `policy.yaml:5-10`; Phase 1 dùng vai ở `HOP-DONG.md` §4.2; không dùng BR-MNT-02 |

- **Chỗ gộp của roadmap.** `docs/05-master-roadmap.md:37` gộp "kiểm tra máy" với "kiểm tra chênh lệch" (câu gốc "tạo phiếu yêu cầu kiểm tra chênh lệch EX-03"); CON-S-026 mô tả "mở phiếu kiểm tra thiết bị cho EX-03, gán người nhận, có audit" (`docs/tenants/construction/05-problem-catalog.md:94`). Phase 1 **không chứng minh** CON-T-010 nguyên văn: nó chứng minh biên kiểm soát (quyền, chống trùng, đọc lại, audit) mà CON-T-010 cũng cần. CON-T-010 nguyên văn chạy được khi Phase 2 (workflow, giao người nhận) có mặt.

---

## 2. Bảng tóm tắt

| # | Câu | Trả lời ngắn | Qua? |
|---|---|---|---|
| 1 | Giá trị thật là gì? | Người dùng giả định (kế toán, quản lý dự án), việc rõ, phép đo đề xuất; không có người thật, không có số nền | ✗ |
| 2 | Việc có critical không? | Có: ghi sai hoặc báo "đã tạo" khi không có là S1; phiếu không tự đổi số tiền | ✓ |
| 3 | Dữ liệu đã chuẩn chưa? | Đầu vào lệnh có cấu trúc, chỉ dùng hai vế từ nguồn có cấu trúc; lỗi im lặng L1 còn mở | ◐ |
| 4 | Đã phân loại dữ liệu chưa? | Mỗi loại đầu vào và mỗi loại nguồn có một cách xử lý (bảng ở câu 4) | ✓ |
| 5 | Phần nào không cần AI? | Toàn bộ: 0 lời gọi model; trích tham số từ chữ tự do là cổng đóng | ✓ |
| 6 | Cần gom ngữ cảnh gì? | Danh tính, thực thể + tổ tiên, hai vế chênh lệch, chính sách, phiếu đã có, múi giờ; kèm quyền đọc | ✓ |
| 7 | Thiếu hay mâu thuẫn thì hỏi lại thế nào? | Từ chối có mã, không đoán; mâu thuẫn ngang thẩm quyền chuyển người duyệt | ✓ |
| 8 | Ai cam kết độ chính xác? | Chưa có người nghiệp vụ thật (D-RM-03) | ✗ |
| 9 | Đồ có sẵn đã giải được chưa? | So một giải pháp bằng lập luận có nguồn công khai; chưa thử thật | ◐ |

---

## 3. Chi tiết từng câu

### Câu 1. Giá trị thật là gì?

- **Trả lời.**
  - Người dùng (giả định): kế toán (`accountant`) làm đối soát nhiên liệu cuối kỳ; quản lý dự án (`project_manager`) của dự án chứa máy.
  - Việc: biến một chênh lệch đã phát hiện thành một bản ghi có chủ, có vết, không trùng. Hôm nay chênh lệch chỉ hiện trong câu trả lời của luồng hỏi; không có bản ghi nào có chủ, có trạng thái, có người nhận (từ khi làn 0 gộp, câu trả lời nằm trong vết gỡ lỗi `decision_records`, nhưng vết đó không phải phiếu).
  - Giá trị đo được (đề xuất phép đo, chưa có số): (a) thời gian từ lúc có chênh lệch vượt ngưỡng tới lúc có phiếu; (b) tỉ lệ chênh lệch vượt ngưỡng có đúng một phiếu; (c) số lần hệ báo "đã tạo" mà không có phiếu, đích 0.
- **Bằng chứng / nguồn.**
  - Vai và bước đối soát: `docs/tenants/construction/01-business-map.md` §3.11 (kế toán: "đối soát nhiên liệu"; ngoại lệ "lệch giữa các chỉ số nhiên liệu") [REPO].
  - Chênh lệch chỉ hiện trong câu trả lời: route `CALCULATION` ở `intelligence/router/router.go:239` [CODE]; làn 0 (đã gộp vào `main`) lưu bền vết từng lượt hỏi ở bảng `decision_records` (migration `0003_kho_vet_ben.sql`), nhưng đó là vết gỡ lỗi, không phải bản ghi nghiệp vụ có chủ [CODE, `9ed7a6a`]; dòng "lưu bền bản ghi quyết định: chưa có" của `docs/01-architecture.md` §4 L7 đã lỗi thời ở điểm này [REPO].
  - Lời hứa không có việc đứng sau: mẫu câu "đã chuyển người phụ trách đối soát" (`tenants/construction/pack/templates.yaml:16-17`) [CODE]. **Phase 1 một mình không sửa được lời hứa này**: chưa có người nhận phiếu cho tới Phase 2.
  - Số nền: KPI "Chênh lệch nhiên liệu" ghi "chưa đo" (`01-business-map.md:498`) [REPO].
- **Qua cổng? ✗.** Giống Fuel (`docs/ROADMAP-DAY-DU.md` §7): tenant hư cấu (`01-business-map.md:3`), chưa có người dùng thật, chưa có số nền.
- **Để qua:** có người nghiệp vụ (D-RM-03) và đo (a), (b) trên quy trình hiện tại của họ.

### Câu 2. Việc có critical không?

- **Trả lời.** Có. Mức hậu quả khi sai:
  - báo "đã tạo phiếu" mà không có phiếu: chênh lệch không ai xử lý. Đây là CON-F-020, mức **S1**;
  - tạo phiếu sai hoặc trùng: điều tra thừa, mất thời gian người;
  - phiếu **không** sửa số liệu nguồn, không phát lệnh chi, không chuyển tiền [SUY: Phase 1 chỉ tạo một bản ghi]. Hậu quả về tiền là gián tiếp: bỏ sót thất thoát nhiên liệu.
- **Bằng chứng / nguồn.**
  - S1 = "sai số liệu / hành động tới tay người dùng… hành động ghi sai" (`docs/CONVENTIONS.md` §4.4) [REPO].
  - CON-F-020 mức S1, hậu quả "người dùng tin việc đã làm; máy không được kiểm tra" (`07-failure-catalog.md:76`) [REPO].
  - Nhiên liệu là chi phí chính (`01-business-map.md:18`), rủi ro R-08 "thất thoát nhiên liệu" (`:479`) [REPO].
- **Qua cổng? ✓** (mức hậu quả đã ghi). Giới hạn: mức S1 là theo định nghĩa của dự án; thiệt hại bằng tiền chưa có người nghiệp vụ xác nhận, chưa đo.

### Câu 3. Dữ liệu đã chuẩn chưa?

- **Trả lời.** Chưa chuẩn hoàn toàn. Thiết kế chọn đường ít phụ thuộc chữ tự do nhất:
  - **Đầu vào của lệnh có cấu trúc** (khoá máy, chỉ số, ngày); không nhận chữ tự do, không dựng lệnh từ tin Zalo. Đây là cách "chuẩn hoá đầu vào trước khi đọc chữ tự do" mà câu 3 đòi.
  - **Chỉ số đầu tiên chỉ chạm nguồn có cấu trúc:** vế kho xuất từ phiếu kho / ERP / bảng tổng hợp, vế tiêu thụ từ cảm biến. Chỉ số `reported_vs_issued` có vế trái là tin chat và mẫu báo cáo (chữ tự do), nên **hoãn** (xem `HOP-DONG.md` §5).
  - **Lỗi im lặng của luồng L1 còn mở** (`docs/ROADMAP-DAY-DU.md` §5 L1): `≈` mất cờ ước lượng; tin đính chính thành số mới; tin kế hoạch thành quan sát; tin gửi lại thành `SYSTEM_FAILURE`; "máy 03" thành 3 L; nhóm nghìn viết bằng dấu cách (D-OP-9); cách ly chỉ nằm trong RAM. Riêng `aggregate` cộng mọi fact cùng nguồn, cùng ngày (`data/sourceoftruth/query.go:201`), nên **có thể cộng đôi** [CODE, chưa tái hiện].
  - **Giảm nhẹ trong thiết kế:** phiếu mang bản chụp bằng chứng và cờ `approx`; hai nguồn ngang thẩm quyền lệch thì không chọn số. Thiết kế **không** bắt được một fact sai im lặng đã vào kho.
- **Bằng chứng / nguồn.** `tenants/construction/pack/sources.yaml` (7 loại nguồn [ĐO `grep -c "^  - type:" tenants/construction/pack/sources.yaml` → 7]; chỉ `chat_message` và `site_report_form` đánh dấu `free_text`); `metrics.yaml:50-51`; `docs/ROADMAP-DAY-DU.md` §5 L1 và §7 (câu 3 của Fuel trượt vì cùng lý do) [REPO].
- **Qua cổng? ◐.** Đã đánh giá dữ liệu và đã chọn đầu vào chuẩn hoá. Chưa có cổng kiểm mức sẵn sàng dữ liệu; lỗi im lặng còn mở.
- **Để qua:** (a) các lỗi im lặng đóng ở Phase 0a / 3; (b) cờ ước lượng sửa xong (`docs/PROMPT-BUILD-EAIS.md` §4 mục 3); (c) đo rủi ro cộng đôi bằng bơm dữ liệu.

### Câu 4. Đã phân loại dữ liệu chưa?

| Loại | Cách xử lý trong lệnh | Nguồn |
|---|---|---|
| tham số lệnh (khoá máy, chỉ số, ngày, khoá lệnh) | kiểm schema; tra danh mục; không bí danh, không mặc định ngầm | `HOP-DONG.md` §2 |
| fact kho xuất (`warehouse_slip` hạng 1; `erp`, `excel_summary` hạng 2) | thẩm quyền theo hạng, dung sai 10 L; ngang hạng lệch quá dung sai thì chuyển người duyệt | `metrics.yaml`, `data/sourceoftruth/query.go:64-197` [CODE] |
| fact tiêu thụ đo được (`fuel_sensor` hạng 1) | như trên | `metrics.yaml` |
| fact đội báo (`chat_message`, `site_report_form`), hoá đơn (`invoice`) | **không dùng** ở Phase 1 | `HOP-DONG.md` §5 |
| vai và phạm vi thực thể | quyền ba lớp, đóng mặc định | `intelligence/router/router.go:127-161` [CODE] |
| phiếu, sổ lệnh, audit | ghi chỉ-thêm, RLS FORCE | `HOP-DONG.md` §7 |

- **Bằng chứng / nguồn:** cột "Nguồn" của bảng; `tenants/construction/pack/sources.yaml` (7 loại nguồn, hạng thẩm quyền) và `metrics.yaml` [CODE].
- **Qua cổng? ✓.**

### Câu 5. Phần nào không cần AI?

- **Trả lời.** Toàn bộ Phase 1 là tất định, **0 lời gọi model**: kiểm schema, tenant, quyền, tính chênh lệch, luật nghiệp vụ, chống trùng, ghi, đọc lại, audit.
- **Phần duy nhất có thể cần model:** trích tham số lệnh từ câu tự do (bước C2). Cổng này đóng: model trích xuất chỉ mở khi luật tất định trượt trên ca đo được và đầu ra có validator (`CLAUDE.md` §19; `docs/01-architecture.md` §4.1 và §9, "hiện mọi cổng… đang đóng"). Phase 1 không đi qua S4 và không dùng chữ tự do, nên không đo S4 (sàn của hệ, ADR-0010) thay cho đo biên hành động. Phase 1 không có phần AI nào, nên nằm trọn trong vùng "trước AI" của D-AI-00 (`docs/CHO-QUYET-DINH.md` mục 0, dòng 22/10: không lời gọi model trả tiền nào trước khi cổng trước AI đạt; đọc ở `main` sau mốc `c086fe7`) [REPO].
- **Qua cổng? ✓.** Kiểm bằng ca đếm lời gọi model = 0 (`HOP-DONG.md` §10, ca A24).

### Câu 6. Cần gom ngữ cảnh gì?

| Ngữ cảnh | Nguồn | Quyền đọc |
|---|---|---|
| danh tính bên gọi (tenant, vai, `entity_scope`) | Principal lấy từ token của `cmd/eais` (ở dev: token công khai, chỉ loopback); **chưa có xác thực thật** (Phase 5) | không áp dụng |
| máy cần kiểm tra và các tổ tiên của nó | danh mục thực thể, `entityresolution.Lineage` (`data/entityresolution/resolver.go:249-251`) | phạm vi thực thể (`router.go:143-161`) |
| hai vế chênh lệch | fact `issued_fuel`, `measured_consumption` qua `Catalog.Derive` (`query.go:251`): thẩm quyền, dung sai, xung đột | `metrics.read` cho **cả hai vế**, như câu so sánh ở luồng hỏi (`router.go:203-209`) |
| chính sách | `policy.yaml` (vai → capability); `tools.yaml` (ĐÍCH) | không áp dụng |
| phiếu và lệnh đã có | bảng `commands`, `discrepancy_tickets` (ĐÍCH) dưới RLS | cùng tenant |
| "hôm nay" | múi giờ của tenant (`pack.yaml`) | không áp dụng |

- **Không gom:** tin chat, tài liệu SOP, lịch sử hội thoại, đầu ra của model.
- **Bằng chứng / nguồn:** cột "Nguồn" và "Quyền đọc" của bảng; xác thực thật chưa có, token dev của làn 0 là công khai (`HOP-DONG.md` §1.3): `docs/01-architecture.md` §4 L7 [REPO].
- **Qua cổng? ✓.**

### Câu 7. Thiếu hay mâu thuẫn thì hỏi lại thế nào?

- **Trả lời.** Lệnh có cấu trúc nên "hỏi lại" là việc của bên gọi; hệ trả mã có kiểu, **không đoán và không có mặc định ngầm** (khác luồng hỏi, nơi kỳ mặc định là hôm nay):
  - thiếu hoặc sai tham số: `INVALID_REQUEST` kèm `invalid_fields` (danh sách tên trường sai) và `detail` (thông điệp công khai), hai trường của `CommandResult` (`HOP-DONG.md` §2.3);
  - máy không tồn tại hoặc ngoài phạm vi: cùng một mã `PERMISSION_DENIED`;
  - thiếu một vế: `INSUFFICIENT_EVIDENCE`;
  - hai nguồn ngang thẩm quyền lệch quá dung sai: `HUMAN_REVIEW_REQUIRED`, không chọn số;
  - không có chênh lệch: `NO_DISCREPANCY`;
  - đã có phiếu mở cho cùng (máy, chỉ số, ngày): `DUPLICATE_OPEN_TICKET`, trả mã phiếu cũ.
- **Giới hạn:** chưa có hàng đợi người; "chuyển người" ở Phase 1 chỉ là mã từ chối. Hàng đợi, người nhận, hạn chờ là Phase 2.
- **Giới hạn của đường hỏi:** hiện **không** từ chối câu lệnh viết bằng chữ tự do khi câu còn khớp một ý định (mục 1, `HOP-DONG.md` §1.5). Chặn ở đó không thuộc Phase 1.
- **Bằng chứng / nguồn:** `HOP-DONG.md` §2.4 và §5; từ vựng kết quả có sẵn ở `core/contracts/contracts.go:298-305` [CODE].
- **Qua cổng? ✓** (luật đã viết, chưa có mã nên chưa kiểm).

### Câu 8. Ai cam kết độ chính xác?

- **Trả lời.** Chưa có ai. D-RM-03 = "Chưa có" (`docs/CHO-QUYET-DINH.md` mục 0, dòng 22/10 11:2x) [REPO].
- **Phần phải có người nghiệp vụ ký:**
  1. định nghĩa "chênh lệch cần phiếu": chỉ số, ngưỡng và mẫu số. BR-FUEL-03 chỉ ghi ví dụ "±1%/ngày" mà không nói phần trăm của vế nào (`01-business-map.md:382`);
  2. vai nào được tạo phiếu (`HOP-DONG.md` §4.2 là đề xuất);
  3. chính sách trùng: một phiếu mở cho mỗi (máy, chỉ số, ngày) có đúng nghiệp vụ không;
  4. mức chấp nhận về thời gian từ lệnh tới phiếu: chưa có, chưa đo.
- **Phần kỹ thuật không cần người nghiệp vụ để kiểm:** đúng một phiếu cho mỗi lệnh; không ghi khi thiếu quyền; không báo "đã làm" khi chưa đọc lại thấy phiếu. Đó là cam kết kỹ thuật kiểm bằng ca (`HOP-DONG.md` §11), **không** phải cam kết độ chính xác nghiệp vụ.
- **Bằng chứng / nguồn:** `docs/CHO-QUYET-DINH.md` mục 0 (D-RM-03, 22/10); `docs/ROADMAP-DAY-DU.md` §7 (Fuel câu 8: D-G7-12, D-G7-16 còn treo) và §9; `01-business-map.md:382` (ngưỡng). Người ký: **chưa có**.
- **Qua cổng? ✗.** Mọi kết luận ghi "trên dữ liệu giả lập, chưa có người nghiệp vụ duyệt". Câu hỏi gửi chủ dự án: `HOP-DONG.md` §14.

### Câu 9. Đồ có sẵn đã giải được chưa?

- **Trả lời.** So một giải pháp có sẵn bằng lập luận có nguồn công khai (mục 4). Kết luận: thứ Phase 1 cần chứng minh là **biên kiểm soát** (quyền, chống trùng, đọc lại, audit), không phải bản thân "phiếu". Biên này cần có dù phiếu nằm ở đâu.
- **Bằng chứng / nguồn:** mục 4: ba nguồn công khai [NGOÀI] và bảng so tiêu chí [SUY]. Phép thử trên hệ có sẵn: **chưa có**.
- **Qua cổng? ◐.** Đã so một giải pháp bằng lập luận; chưa thử thật, chưa so CRM nào, chưa so form hay bảng tính có sẵn.

---

## 4. Câu 9: so với đồ có sẵn

**Giải pháp so:** ticket có sẵn, ví dụ Jira Cloud tạo qua REST (một lệnh tạo issue). **Mẫu tham chiếu cho chống trùng:** khoá chống trùng của API thanh toán Stripe và bản nháp IETF về header `Idempotency-Key`. Đây là mẫu thiết kế, không phải giải pháp cho việc.

| Tiêu chí (lấy từ cổng ra của Phase 1) | Ticket có sẵn (Jira Cloud qua REST) | Phase 1 tự làm (chế độ B) |
|---|---|---|
| Chống trùng khi gửi lại hoặc timeout | Cộng đồng Atlassian chưa nêu khoá chống trùng có sẵn cho lệnh tạo issue; cách làm được nhắc tới là ghi mã duy nhất vào trường tuỳ chỉnh rồi tìm trước khi tạo [NGOÀI, thread cộng đồng, không phải tài liệu chính thức] | sổ lệnh + khoá duy nhất + phát lại kết quả |
| Tìm kiếm sau khi tạo | Một người dùng báo một lần tìm kiếm ~5 giây sau khi tạo vẫn không thấy issue; người khác không tái hiện được. Thread nói về **tìm kiếm** [NGOÀI, thread cộng đồng] | không dùng tìm kiếm: đọc theo mã từ nơi ghi |
| Đọc theo mã sau khi tạo | **Chưa biết**: thread không nói về đọc theo mã issue; chưa đối chiếu tài liệu REST chính thức (`HOP-DONG.md` U6) | đọc lại từ nơi ghi, cùng RLS, trong chính lượt chạy |
| Quyền theo phạm vi thực thể của EAIS | không biết `entity_scope` của EAIS [SUY] | quyền ba lớp có sẵn trong EAIS |
| Nội dung phiếu là số liệu có thẩm quyền | chỉ lưu thứ được gửi tới [SUY] | hệ tự tính từ fact, bên gọi không đưa giá trị |
| Quy trình duyệt, người nhận, thông báo | thế mạnh của loại sản phẩm này [SUY, chưa so tính năng] | không có (Phase 2) |
| Chi phí vận hành thêm một hệ | tài khoản, giấy phép, vận hành riêng [SUY, chưa đo] | dùng PostgreSQL đã có |

**Mẫu chống trùng của API thanh toán** [NGOÀI]:
- Stripe lưu mã trạng thái và thân kết quả của lần đầu theo khoá; gửi lại cùng khoá trả kết quả đã lưu; so tham số với lần gốc và báo lỗi nếu khác; khoá có thể bị xoá sau ít nhất 24 giờ (API v1). Nguồn: `https://docs.stripe.com/api/idempotent_requests`.
- Bản nháp IETF `draft-ietf-httpapi-idempotency-key-header-07` (đã hết hạn, chưa là chuẩn): thiếu khoá thì 400; cùng khoá khác nội dung thì 422; khoá còn đang xử lý thì 409 (đều "SHOULD"). Nguồn: `https://datatracker.ietf.org/doc/draft-ietf-httpapi-idempotency-key-header/`.
- Phase 1 dùng ý "khoá + so nội dung", khác ở hai chỗ: lệnh đồng thời dựa vào khoá duy nhất của cơ sở dữ liệu nên không cần trả 409; khoá không hết hạn trong Phase 1.

**Nguồn về Jira:** thread "How do I ensure issue creation is idempotent?", cộng đồng nhà phát triển Atlassian, 07–10/2025: `https://community.developer.atlassian.com/t/how-do-i-ensure-issue-creation-is-idempotent/93517`. Không có nhân viên Atlassian trả lời trong thread; không ai khẳng định hay phủ nhận khoá chống trùng có sẵn. Thread đặt ở mục Jira Server; người không tái hiện được đã thử trên Cloud (theo người soát độc lập đọc lại 22/10/2026; tác giả chưa đọc lại trang này ở lượt sửa).

**Kết luận.**
- Với chế độ B, tự lưu phiếu rẻ hơn thêm một hệ ngoài [SUY], và biên kiểm soát (quyền, chống trùng, đọc lại, audit) phải tự làm dù chọn hệ nào [SUY]. Nguồn công khai tìm được **không đủ** để nói ticket có sẵn thiếu chống trùng và đọc lại: thread cộng đồng chỉ cho thấy chưa ai nêu khoá chống trùng có sẵn cho lệnh tạo issue, và nói về tìm kiếm chứ không về đọc theo mã. Chưa đối chiếu tài liệu chính thức (U6).
- Với chế độ A (khách đã có Jira hoặc CRM), Phase 4b gọi API của họ qua **cùng biên** và không tự lưu phiếu. ADR-0011 vì vậy tách "biên kiểm soát" khỏi "nơi lưu phiếu".
- **Chưa so:** tính năng quy trình của ticket có sẵn; CRM cụ thể; form, bảng tính, Zalo (cách làm hiện tại của tenant); chi phí thật. **Chưa thử thật:** không có phiên bản Jira nào để thử; chưa đối chiếu với tài liệu REST chính thức của Atlassian.
- **Để ◐ thành ✓:** đọc tài liệu chính thức của Atlassian và đối chiếu; so thêm một CRM hoặc một form có sẵn theo cùng bảng tiêu chí; ghi kết quả.

---

## 5. Cổng gốc (`docs/ROADMAP-DAY-DU.md` §4.2)

| # | Điều kiện | Trạng thái | Nguồn |
|---|---|---|---|
| 1 | Phase tiên quyết đã đạt | **Đạt.** D-RM-04 = A: mã mở "sau khi chạy xong H4"; H4 đã chạy một lần 22/10 14:59 (`57409f2`) | `docs/CHO-QUYET-DINH.md` mục 0; `docs/STATUS.md` |
| 2 | Có lỗi đo được hoặc ca nghiệp vụ đòi năng lực mới | **Có ca nghiệp vụ** (CON-S-026, CON-T-010, CON-F-020; hai ca đầu thuộc domain bảo trì, Phase 1 đáp ứng một phần: ánh xạ ở mục 1); **có một lỗi đo được**: đường hỏi coi câu lệnh là câu hỏi (mục 1, [ĐO]; chưa xếp mức, chưa đo thiệt hại). Quan sát trong mã: mẫu câu hứa chuyển người mà không có việc nào đứng sau (`templates.yaml:16-17`) | mục 1, câu 1 |
| 3 | Trả lời được "ai sở hữu?" | Trả lời đủ 10 mặt ở `HOP-DONG.md` §8 (giấy) | `docs/05-master-roadmap.md:30` |
| 4 | Cổng ra nhị phân, có ca âm chứng minh cổng đỏ được | Viết ở `HOP-DONG.md` §10 (ca âm), §11 (cổng ra, đột biến) (giấy; chưa chạy) | `docs/ROADMAP-DAY-DU.md` §6 |

---

## 6. Điều kiện đổi ✗ / ◐ thành ✓ (việc của người, không phải của mã)

| Câu | Cần | Ai |
|---|---|---|
| 1 | có người nghiệp vụ; đo thời gian từ chênh lệch tới phiếu trên quy trình hiện tại của họ | chủ dự án (D-RM-03) |
| 3 | đóng lỗi im lặng L1 (Phase 0a / 3); cờ ước lượng; đo rủi ro cộng đôi bằng bơm dữ liệu | BUILD |
| 8 | người nghiệp vụ ký bốn mục ở câu 8 | chủ dự án (D-RM-03) |
| 9 | đối chiếu tài liệu chính thức của Atlassian; so thêm một giải pháp theo bảng mục 4 | BUILD hoặc người soát |

Chừng nào câu 1 và 8 còn ✗: Phase 1 chỉ được báo là "cổng ra kỹ thuật đạt trên dữ liệu giả lập", không báo là "đáng giá".
