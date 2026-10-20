# 04 — Bản đồ workflow (ABC Infrastructure)

> Tenant `abc-infra` (mã `CON`). Mọi tên, mã phiếu, **hạn chót và hạn mức** trong tài liệu là **tham số ví dụ giả lập** do pack cấu hình, không phải kết quả đo (CONVENTIONS §6).
> Liên quan: [01-business-map](01-business-map.md) · [05-problem-catalog](05-problem-catalog.md) · [07-failure-catalog](07-failure-catalog.md) · [`cases.yaml`](../../../tenants/construction/evals/cases.yaml).

## 1. Nguyên tắc

Mỗi workflow là một **state machine (máy trạng thái)** khai báo trong `workflows.yaml` của pack và chạy bằng **workflow engine** dùng chung (`execution/workflows`). Ba luật ràng buộc ([kiến trúc §2.3, §8](../../01-architecture.md)):

1. **Engine sở hữu trạng thái (P11).** Trạng thái, phép chuyển, quyền, bất biến và giao dịch thuộc phần mềm. Model chỉ được **đề xuất** (đọc, phân loại, trích xuất, gợi ý, giải thích); đề xuất phải qua guard như mọi lệnh khác.
2. **Guard (điều kiện chặn một phép chuyển) là TÊN, không phải biểu thức (P12).** Pack chọn guard từ danh mục mã đã cài ở §2; pack không viết điều kiện lồng, vòng lặp hay biểu thức tự do. Luật phức tạp hơn (ví dụ "vượt định mức 3 ngày liên tiếp") là **chỉ số dẫn xuất** do mã cài, pack chỉ so sánh `metric op ngưỡng`.
3. **Chờ người là human task (việc giao cho người)** có người nhận và hạn chót. Bên ngoài tổ chức (TVGS, CĐT) không có tài khoản: nhân viên hồ sơ nhập xác nhận thay, kèm chứng từ.

## 2. Danh mục guard đã cài (tên dùng trong pack)

| Guard | Ý nghĩa |
|---|---|
| `required_fields_complete` | đủ trường/khe bắt buộc |
| `values_in_domain` | giá trị đơn trường trong miền hợp lệ |
| `all_clarifications_resolved` | không còn câu hỏi lại treo |
| `has_approver_role` | principal có vai/quyền capability để chuyển |
| `not_same_principal` | người chuyển khác người lập |
| `amount_within_limit` | giá trị trong hạn mức của vai; quyết định danh sách task duyệt |
| `attachments_present` | có đính kèm bắt buộc (báo giá, biên bản đo…) |
| `all_review_tasks_done` | mọi human task của bước đã xong đúng thứ tự |
| `stock_sufficient` | tồn kho đủ, không xuất âm |
| `three_way_match_ok` | PO – nhập kho – hoá đơn khớp |
| `severity_notified` | đã khoá hiện trường và thông báo (sự cố nghiêm trọng) |
| `no_open_ncr` | không còn báo cáo không phù hợp mở |
| `assignee_in_scope` | người được giao thuộc phạm vi dự án |
| `period_not_locked` | kỳ chưa khoá |
| `variance_within_threshold` | chênh lệch derived fact trong ngưỡng |

## 3. Mười hai workflow lõi

| Mã | Workflow (`business_process`) | Trạng thái |
|---|---|---|
| WF-01 | `daily-site-report` báo cáo ngày | DRAFT → SUBMITTED → UNDER_REVIEW → APPROVED → LOCKED |
| WF-02 | `fuel-reconciliation` cấp và đối soát nhiên liệu | OPEN → VARIANCE_FLAGGED → INVESTIGATING → RESOLVED → CLOSED |
| WF-03 | `quantity-acceptance` đo khối lượng và nghiệm thu | DRAFT → MEASURED → SUBMITTED → ACCEPTED → CLOSED |
| WF-04 | `material-request` đề nghị và xuất vật tư | DRAFT → SUBMITTED → REVIEWING → APPROVED → ORDERED → RECEIVED → RECONCILED → CLOSED |
| WF-05 | `equipment-maintenance` bảo dưỡng máy | OPEN → ASSIGNED → INSPECTING → REPAIRING → VERIFIED → CLOSED |
| WF-06 | `payment-certification` thanh toán | RECEIVED → MATCHING → APPROVED → PAID → CLOSED |
| WF-07 | `change-order` phát sinh | DRAFT → TECH_REVIEW → PRICING → CLIENT_REVIEW → APPROVED → CONTRACT_AMENDED → CLOSED |
| WF-08 | `rfi-handling` yêu cầu làm rõ (RFI) | DRAFT → SUBMITTED → REPLIED → CLOSED (hoặc ESCALATED khi quá hạn) |
| WF-09 | `qaqc-inspection` kiểm tra QA/QC | PLANNED → INSPECTING → PASSED → CLOSED (hoặc FAILED → NCR_OPEN → CORRECTED → INSPECTING) |
| WF-10 | `safety-incident` sự cố an toàn | REPORTED → TRIAGED → INVESTIGATING → ACTION_PLANNED → CLOSED |
| WF-11 | `document-revision` phiên bản tài liệu/bản vẽ | DRAFT → IN_REVIEW → APPROVED → EFFECTIVE → SUPERSEDED |
| WF-12 | `management-briefing` tổng hợp cho quản lý | REQUESTED → COLLECTING → DRAFTED → REVIEWED → PUBLISHED |

Quy trình hỗ trợ không có vòng đời (dùng làm `business_process` trong ca kiểm thử): `metric-query` (truy vấn số liệu), `knowledge-lookup` (tra SOP/hợp đồng), `document-search` (tìm tài liệu theo mã), `access-governance` (quyền, tenant, dữ liệu cá nhân), `master-data-maintenance` (danh mục), `platform-operations` (nạp tài liệu, pack, hạ tầng).

Mỗi bảng dưới có cột **Model được ĐỀ XUẤT gì** (P11): đó là tất cả những gì model được làm ở bước đó; **Ai quyết** luôn là engine (guard) hoặc người.

---

## 4. Chi tiết từng workflow

### WF-01 `daily-site-report` — báo cáo ngày

- **Chủ thể (`subject`)**: (dự án, ngày làm việc). **Kích hoạt**: tin nhắn hiện trường đầu tiên hoặc lệnh mở báo cáo. **Hệ thống**: nhóm chat, ERP, kho tài liệu. **Dữ liệu**: observation → fact `reported_progress`, `reported_machine_hours`, `reported_fuel`, `reported_headcount`, thời tiết.
- **Ngoại lệ**: tin mơ hồ (hỏi lại); ca đêm (tính vào ngày bắt đầu ca); báo cáo gửi trùng nhiều nhóm; thiếu khe; quá hạn nộp.

| Từ | Đến | Phép chuyển | Guard (tên) | Human task (ai, hạn chót) | Ai quyết | Model được đề xuất |
|---|---|---|---|---|---|---|
| (chưa có) | DRAFT | `open` | — | — | engine | không |
| DRAFT | DRAFT | `add_facts` (tin nhắn → fact) | `values_in_domain` | `clarify_*` cho giám sát (trong 1 giờ làm việc) khi mơ hồ | luật chuẩn hoá + cổng mơ hồ | trích khe khi cổng trích xuất model mở và đã qua validator |
| DRAFT | SUBMITTED | `submit` | `required_fields_complete`, `all_clarifications_resolved` | `submit_daily_report` giám sát (hạn 18:00) | giám sát bấm nộp; engine kiểm guard | tóm tắt, nêu bất thường so hôm trước |
| SUBMITTED | UNDER_REVIEW | `start_review` | `has_approver_role` | `review_daily_report` trưởng dự án (trong 1 ngày làm việc) | trưởng dự án | giải thích chênh lệch |
| UNDER_REVIEW | DRAFT | `return_for_revision` | `has_approver_role` | giám sát sửa (lý do bắt buộc) | trưởng dự án | gợi ý khe nghi sai |
| UNDER_REVIEW | APPROVED | `approve` | `has_approver_role`, `not_same_principal` | — | trưởng dự án | "nên duyệt" (chỉ gợi ý) |
| APPROVED | LOCKED | `lock` (chốt kỳ) | `period_not_locked` | — | engine theo lịch kỳ | không |

Timer: quá hạn 18:00 → human task `overdue_report` leo thang lên trưởng dự án (đúng một lần).

### WF-02 `fuel-reconciliation` — cấp và đối soát nhiên liệu

- **Chủ thể**: (dự án, kỳ). **Kích hoạt**: hoá đơn nhiên liệu đến, cuối kỳ, hoặc chênh lệch vượt ngưỡng. **Hệ thống**: ERP kho, đồng hồ/cảm biến, nhóm chat, hoá đơn. **Dữ liệu**: bốn chỉ số `reported_fuel`, `issued_fuel`, `measured_consumption`, `billed_fuel` và derived fact chênh lệch (công thức ở 03).
- **Ngoại lệ**: hai bản phiếu cho một lần cấp; phiếu bị rút lại; ảnh phiếu không đọc được; lệch múi giờ.

| Từ | Đến | Phép chuyển | Guard (tên) | Human task (ai, hạn chót) | Ai quyết | Model được đề xuất |
|---|---|---|---|---|---|---|
| (chưa có) | OPEN | `open_period` | — | — | engine | không |
| OPEN | CLOSED | `compare_ok` | `variance_within_threshold` | — | engine | không |
| OPEN | VARIANCE_FLAGGED | `compare_variance` | — | `review_variance` kế toán kho (2 ngày làm việc) | derived fact + ngưỡng pack | giải thích khả năng nguyên nhân |
| VARIANCE_FLAGGED | INVESTIGATING | `assign_investigation` | `has_approver_role`, `assignee_in_scope` | `investigate_variance` thủ kho + thiết bị (5 ngày làm việc) | trưởng phòng vật tư | gợi ý người điều tra |
| VARIANCE_FLAGGED | VARIANCE_FLAGGED | `recompute_derived` (fact bị rút lại) | — | `recompute_derived_facts` kế toán kho | người xử lý (lùi an toàn, xem F-005) | không |
| INVESTIGATING | RESOLVED | `resolve` (mã kết luận bắt buộc) | `required_fields_complete` | — | người điều tra + trưởng phòng | soạn mô tả kết luận |
| RESOLVED | CLOSED | `close` | `has_approver_role` | — | kế toán trưởng | không |

### WF-03 `quantity-acceptance` — đo khối lượng và nghiệm thu

- **Chủ thể**: (hạng mục, đợt nghiệm thu). **Kích hoạt**: yêu cầu nghiệm thu của kỹ thuật. **Dữ liệu**: `measured_quantity` (đo), `accepted_quantity` (sau xác nhận), biên bản. **Ngoại lệ**: TVGS/CĐT phản hồi chậm hoặc không đồng ý; BOQ lệch bản vẽ.

| Từ | Đến | Phép chuyển | Guard (tên) | Human task (ai, hạn chót) | Ai quyết | Model được đề xuất |
|---|---|---|---|---|---|---|
| (chưa có) | DRAFT | `create` | `required_fields_complete` | — | engine | trích số đo từ biên bản nháp |
| DRAFT | MEASURED | `record_measurement` | `attachments_present`, `values_in_domain` | `measure` kỹ sư kỹ thuật (3 ngày) | kỹ sư kỹ thuật | không |
| MEASURED | SUBMITTED | `submit_for_confirmation` | `has_approver_role` | `tvgs_confirm`, `cdt_sign` do nhân viên hồ sơ nhập thay (7 ngày; hai task, chỉ đóng khi đủ cả hai — mô hình lùi cho đồng duyệt song song, xem F-044) | trưởng dự án | soạn công văn |
| SUBMITTED | ACCEPTED | `accept` | `all_review_tasks_done` | — | engine sau khi đủ xác nhận | không |
| SUBMITTED | DRAFT | `reject_and_rework` (lý do bắt buộc) | `has_approver_role` | kỹ thuật đo lại | TVGS/CĐT (qua nhân viên hồ sơ) | tóm tắt lý do |
| ACCEPTED | CLOSED | `close` | `period_not_locked` | — | engine | không |

### WF-04 `material-request` — đề nghị và xuất vật tư

- **Chủ thể**: phiếu `VT-NNNN`. **Kích hoạt**: lệnh của giám sát hoặc trigger tồn kho dưới mức an toàn. **Hệ thống**: ERP (PO, kho). **Dữ liệu**: phiếu, báo giá, PO, PNK. **Ngoại lệ**: vượt hạn mức; thiếu báo giá; NCC hết hàng; ERP lỗi khi tạo PO.
- **Hạn mức ví dụ**: ≤ 50 triệu chỉ trưởng dự án; ≤ 300 triệu thêm kế toán; trên đó thêm giám đốc (BR-PROC-02).

| Từ | Đến | Phép chuyển | Guard (tên) | Human task (ai, hạn chót) | Ai quyết | Model được đề xuất |
|---|---|---|---|---|---|---|
| (chưa có) | DRAFT | `create` | `values_in_domain` | — | engine | trích vật tư/số lượng từ câu tự do (sau kiểm schema) |
| DRAFT | SUBMITTED | `submit` | `required_fields_complete`, `attachments_present` | — | người lập | không |
| SUBMITTED | REVIEWING | `start_review` | `amount_within_limit` (quyết định danh sách task) | `pm_review`, `finance_review`, `director_review` theo thứ tự (mỗi task 1 ngày làm việc) | engine theo hạn mức | gợi ý thiếu sót |
| REVIEWING | APPROVED | `approve` | `has_approver_role`, `not_same_principal`, `all_review_tasks_done`, `attachments_present` | — | các người duyệt; engine kiểm | "nên duyệt" (chỉ gợi ý, bị guard từ chối nếu thiếu chứng từ) |
| REVIEWING | REJECTED | `reject` (lý do bắt buộc) | `has_approver_role` | người lập sửa | người duyệt | tóm tắt lý do |
| APPROVED | ORDERED | `place_order` | — | `create_po_manual` vật tư (khi ERP lỗi, xem F-048) | engine + tool `create_purchase_order` | không |
| ORDERED | RECEIVED | `receive` | `stock_sufficient` không áp; cần PNK | `confirm_receipt` thủ kho | thủ kho | không |
| RECEIVED | RECONCILED | `reconcile` | `three_way_match_ok` | kế toán | kế toán | giải thích chênh lệch |
| RECONCILED | CLOSED | `close` | — | — | engine | không |

### WF-05 `equipment-maintenance` — bảo dưỡng máy

- **Chủ thể**: phiếu `MT-NNNN` của một máy. **Kích hoạt**: lệnh, cảm biến vượt ngưỡng, đến hạn theo giờ máy (chỉ số dẫn xuất). **Dữ liệu**: `logged_engine_hours`, phiếu, kết quả kiểm tra. **Ngoại lệ**: cảm biến gửi trùng/đến trễ; máy đã có phiếu mở; kỹ thuật viên đã rời dự án.

| Từ | Đến | Phép chuyển | Guard (tên) | Human task (ai, hạn chót) | Ai quyết | Model được đề xuất |
|---|---|---|---|---|---|---|
| (chưa có) | OPEN | `open` | khoá khử trùng sự kiện (không mở trùng) | `assign_technician` trưởng bộ phận thiết bị (4 giờ làm việc) | trigger `metric op ngưỡng` hoặc lệnh | giải thích bất thường cho người |
| OPEN | ASSIGNED | `assign` | `has_approver_role`, `assignee_in_scope` | kỹ thuật viên nhận | trưởng bộ phận thiết bị | gợi ý người phù hợp |
| ASSIGNED | INSPECTING | `start_inspection` | chỉ người được gán | — | kỹ thuật viên | không |
| INSPECTING | REPAIRING | `repair_needed` | `required_fields_complete` (kết quả kiểm) | — | kỹ thuật viên | gợi ý hạng mục sửa từ sổ tay |
| INSPECTING | VERIFIED | `no_fault` | — | — | kỹ thuật viên | không |
| REPAIRING | VERIFIED | `repair_done` | `attachments_present` | — | kỹ thuật viên | không |
| VERIFIED | CLOSED | `close` | `has_approver_role` | — | trưởng bộ phận thiết bị | không |

### WF-06 `payment-certification` — thanh toán nhà cung cấp/nhà thầu phụ

- **Chủ thể**: hoá đơn. **Kích hoạt**: hoá đơn đến (ERP). **Dữ liệu**: hoá đơn (`billed_*`), PO, PNK. **Ngoại lệ**: lệch số lượng/đơn giá; hoá đơn trùng; chứng từ thiếu; giữ lại (retention) theo hợp đồng.

| Từ | Đến | Phép chuyển | Guard (tên) | Human task (ai, hạn chót) | Ai quyết | Model được đề xuất |
|---|---|---|---|---|---|---|
| (chưa có) | RECEIVED | `receive_invoice` | `duplicate_key_check` | — | engine | trích trường từ hoá đơn (sau kiểm schema) |
| RECEIVED | MATCHING | `start_matching` | `required_fields_complete` | kế toán (2 ngày làm việc) | engine | không |
| MATCHING | APPROVED | `approve` | `three_way_match_ok`, `has_approver_role`, `amount_within_limit` | `finance_approval` trưởng phòng tài chính (khi vượt hạn mức) | kế toán / trưởng phòng tài chính | giải thích lệch nhỏ |
| MATCHING | DISPUTED | `dispute` (lý do bắt buộc) | — | `resolve_dispute` kế toán + vật tư | kế toán | soạn thư tranh chấp |
| APPROVED | PAID | `pay` | `has_approver_role` | — | tài chính | không |
| PAID | CLOSED | `close` | — | — | engine | không |

### WF-07 `change-order` — phát sinh

- **Chủ thể**: phiếu `PS-NNN`. **Kích hoạt**: lệnh của kỹ thuật khi khối lượng/điều kiện hiện trường khác hợp đồng. **Dữ liệu**: khối lượng, **cơ sở đo (nguyên khối hay tơi xốp)**, đơn giá, phụ lục. **Ngoại lệ**: CĐT từ chối; đơn giá không có trong BOQ; thiếu cơ sở đo.

| Từ | Đến | Phép chuyển | Guard (tên) | Human task (ai, hạn chót) | Ai quyết | Model được đề xuất |
|---|---|---|---|---|---|---|
| (chưa có) | DRAFT | `create` | `required_fields_complete` (gồm cơ sở đo) | — | engine | trích khối lượng/lý do từ câu tự do |
| DRAFT | TECH_REVIEW | `submit` | `attachments_present` | `tech_review` kỹ sư trưởng (3 ngày) | người lập | không |
| TECH_REVIEW | PRICING | `approve_technical` | `has_approver_role` | `pricing` dự toán (3 ngày) | kỹ sư trưởng | gợi ý đơn giá tham chiếu từ BOQ |
| PRICING | CLIENT_REVIEW | `send_to_client` | `amount_within_limit` | `client_confirm` nhân viên hồ sơ nhập phản hồi CĐT (14 ngày) | trưởng dự án | soạn công văn |
| CLIENT_REVIEW | APPROVED | `client_approves` | `attachments_present` (văn bản chấp thuận) | — | CĐT (qua nhân viên hồ sơ) | không |
| CLIENT_REVIEW | REJECTED | `client_rejects` | — | — | CĐT | tóm tắt lý do |
| APPROVED | CONTRACT_AMENDED | `amend_contract` | `has_approver_role` | `draft_amendment` pháp chế | pháp chế | soạn nháp phụ lục |
| CONTRACT_AMENDED | CLOSED | `close` | — | — | engine | không |

### WF-08 `rfi-handling` — yêu cầu làm rõ (RFI)

- **Chủ thể**: `RFI-NNN` gửi TVGS. **Dữ liệu**: câu hỏi, bản vẽ liên quan, phản hồi. **Ngoại lệ**: quá hạn trả lời; phản hồi mâu thuẫn bản vẽ.

| Từ | Đến | Phép chuyển | Guard (tên) | Human task (ai, hạn chót) | Ai quyết | Model được đề xuất |
|---|---|---|---|---|---|---|
| (chưa có) | DRAFT | `create` | `required_fields_complete` | — | engine | soạn nháp câu hỏi từ mô tả hiện trường |
| DRAFT | SUBMITTED | `submit` | `attachments_present` | `await_tvgs_reply` nhân viên hồ sơ theo dõi (hạn 5 ngày làm việc) | kỹ sư kỹ thuật | không |
| SUBMITTED | ESCALATED | `timeout_escalate` (timer) | — | nhắc trưởng dự án | timer của engine | không |
| SUBMITTED | REPLIED | `record_reply` | `attachments_present` | — | nhân viên hồ sơ nhập thay TVGS | tóm tắt phản hồi |
| ESCALATED | REPLIED | `record_reply` | `attachments_present` | — | nhân viên hồ sơ | tóm tắt phản hồi |
| REPLIED | CLOSED | `close` | `has_approver_role` | — | kỹ sư kỹ thuật | không |

### WF-09 `qaqc-inspection` — kiểm tra QA/QC

- **Chủ thể**: đợt kiểm tra (hạng mục, mẫu). **Dữ liệu**: kết quả thí nghiệm, NCR. **Ngoại lệ**: mẫu không đạt; kết quả đến muộn; thiếu mẫu.

| Từ | Đến | Phép chuyển | Guard (tên) | Human task (ai, hạn chót) | Ai quyết | Model được đề xuất |
|---|---|---|---|---|---|---|
| (chưa có) | PLANNED | `plan` | `required_fields_complete` | — | engine | không |
| PLANNED | INSPECTING | `start` | `has_approver_role` | `take_sample` kỹ sư QA/QC (đúng ngày đổ) | kỹ sư QA/QC | không |
| INSPECTING | PASSED | `pass` | `values_in_domain` (kết quả so ngưỡng của tiêu chuẩn) | — | kỹ sư QA/QC | không |
| INSPECTING | FAILED | `fail` | `values_in_domain` | — | kỹ sư QA/QC | giải thích khả năng nguyên nhân |
| FAILED | NCR_OPEN | `open_ncr` | `required_fields_complete` | `corrective_action` kỹ thuật + công trường (3 ngày) | kỹ sư QA/QC | gợi ý biện pháp |
| NCR_OPEN | CORRECTED | `correct` | `attachments_present` | — | kỹ thuật | không |
| CORRECTED | INSPECTING | `reinspect` | — | — | kỹ sư QA/QC | không |
| PASSED | CLOSED | `close` | `no_open_ncr` | — | kỹ sư QA/QC | không |

### WF-10 `safety-incident` — sự cố an toàn

- **Chủ thể**: `SC-NNN`. **Dữ liệu**: mô tả, mức nghiêm trọng (1–4 ví dụ), người liên quan (nhạy cảm), biện pháp. **Ngoại lệ**: báo thấp mức; sự cố không báo; thông tin lan sang nhóm không liên quan.

| Từ | Đến | Phép chuyển | Guard (tên) | Human task (ai, hạn chót) | Ai quyết | Model được đề xuất |
|---|---|---|---|---|---|---|
| (chưa có) | REPORTED | `report` | `required_fields_complete` | `confirm_severity` cán bộ an toàn (2 giờ) | mức = lớn hơn giữa đề xuất và mức tối thiểu của luật (BR-SAF-02) | **đề xuất** mức nghiêm trọng từ mô tả |
| REPORTED | TRIAGED | `triage` | `has_approver_role` | — | cán bộ an toàn | phân loại loại sự cố |
| TRIAGED | INVESTIGATING | `start_investigation` | `severity_notified` (mức nghiêm trọng đã khoá hiện trường và thông báo) | `investigate` cán bộ an toàn (hạn theo mức) | cán bộ an toàn | tóm tắt lời khai |
| INVESTIGATING | ACTION_PLANNED | `plan_actions` | `required_fields_complete` | `execute_actions` trưởng dự án | cán bộ an toàn | gợi ý biện pháp |
| ACTION_PLANNED | CLOSED | `close` | `has_approver_role`, `attachments_present` | — | trưởng dự án + cán bộ an toàn | không |

### WF-11 `document-revision` — phiên bản tài liệu/bản vẽ

- **Chủ thể**: (tài liệu, phiên bản). **Dữ liệu**: metadata hiệu lực, ACL, trạng thái nạp. **Ngoại lệ**: hai bản cùng hiệu lực; parser lỗi; bảng bị mất cột.

| Từ | Đến | Phép chuyển | Guard (tên) | Human task (ai, hạn chót) | Ai quyết | Model được đề xuất |
|---|---|---|---|---|---|---|
| (chưa có) | DRAFT | `upload` | `schema_valid` (đầu ra parser) | — | engine | không |
| DRAFT | IN_REVIEW | `submit_review` | cổng chất lượng đọc (bảng đủ ô, số đủ) | `review_document` chủ tài liệu (3 ngày) | engine | không |
| IN_REVIEW | APPROVED | `approve` | `has_approver_role` | — | chủ tài liệu | tóm tắt thay đổi so bản trước |
| APPROVED | EFFECTIVE | `publish` (nguyên tử) | `period_not_locked`; không có bản khác cùng hiệu lực | — | engine, một giao dịch | không |
| EFFECTIVE | SUPERSEDED | `supersede` | khi bản mới EFFECTIVE | — | engine | không |

### WF-12 `management-briefing` — tổng hợp cho quản lý

- **Chủ thể**: (dự án, ngày). **Kích hoạt**: yêu cầu "kiểm tra toàn bộ dự án…". Đây là **workflow cố định** thay cho agent (cổng AGENT đóng). **Dữ liệu**: năm chiều — tiến độ, vật tư, máy, nhân công, an toàn. **Ngoại lệ**: một chiều thiếu dữ liệu (phải nói rõ, không kết luận "không có vấn đề").

| Từ | Đến | Phép chuyển | Guard (tên) | Human task (ai, hạn chót) | Ai quyết | Model được đề xuất |
|---|---|---|---|---|---|---|
| (chưa có) | REQUESTED | `request` | `has_approver_role` | — | engine | không |
| REQUESTED | COLLECTING | `collect` (chạy đủ năm truy vấn theo thứ tự cố định) | quyền theo dự án | — | engine | không |
| COLLECTING | DRAFTED | `draft` | `required_fields_complete` (đủ năm mục, kể cả mục "chưa có dữ liệu") | — | engine | diễn đạt nhận định, mỗi nhận định gắn bằng chứng |
| DRAFTED | REVIEWED | `review` | `has_approver_role` | `review_briefing` người yêu cầu | người yêu cầu | không |
| REVIEWED | PUBLISHED | `publish` | `has_approver_role` | — | người yêu cầu | không |

---

## 5. Ánh xạ sang pipeline và pack

| Lối vào | Cách chạm vào workflow | Ví dụ trong bộ ca |
|---|---|---|
| COMMAND (C0–C7) | lệnh → kiểm schema → quyền → engine (guard + chuyển trạng thái, một giao dịch) → tool → audit | `CON-S-026`, `028`, `029`, `034` |
| EVENT (E0–E6) | sự kiện → observation → luật `metric op ngưỡng` → khởi động/đẩy workflow | `CON-S-046`, `058`, `059`, `061` |
| QUERY (S0–S10) | chỉ **đọc** trạng thái workflow (không đổi) | `CON-S-020` (qua `management-briefing`) |

`workflows.yaml` của pack chỉ chứa: danh sách trạng thái, phép chuyển, **tên guard**, tên human task (vai nhận + hạn chót), ánh xạ phép chuyển → tool. Không có điều kiện lồng, vòng lặp, biểu thức (CONVENTIONS §8, kiến trúc §8). Những chỗ kiến trúc hiện tại chưa biểu diễn được và hành vi lùi an toàn: đồng duyệt song song (WF-03, F-044), tác dụng phụ ra ngoài có thử lại (WF-04, F-048), nguồn sự kiện theo lịch (WF-01, F-045), lan truyền vô hiệu hoá derived fact (WF-02, F-005) — xem [05 §10](05-problem-catalog.md).
