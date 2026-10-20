# 02 — Bản đồ domain (ABC Infrastructure)

> Tenant `abc-infra` (mã `CON`). Mọi tên, mã, **con số** trong các ca dưới đây là **ví dụ giả lập**, không phải kết quả đo (CONVENTIONS §6, §7).
> Liên quan: [01-business-map](01-business-map.md) · [03-data-map](03-data-map.md) · [04-workflow-map](04-workflow-map.md) · [05-problem-catalog](05-problem-catalog.md) · [07-failure-catalog](07-failure-catalog.md).

## 1. Cách đọc

Ba mươi domain nghiệp vụ của ABC (brief PART II). Mỗi domain có **khoá `snake_case`** — chính là giá trị `domain:` trong [`cases.yaml`](../../../tenants/construction/evals/cases.yaml) và cột Domain ở [05](05-problem-catalog.md) — và **đủ mười loại ca**, mỗi loại một dòng cụ thể:

| Loại ca | Nghĩa |
|---|---|
| Normal | luồng bình thường |
| Edge | biên của luật, trường hợp hiếm nhưng hợp lệ |
| Failure | hệ thống/quy trình làm sai |
| Dirty data | dữ liệu bẩn: viết tắt, lẫn đơn vị, sai chính tả |
| Conflicting data | hai nguồn nói khác nhau |
| Missing data | thiếu trường/tài liệu |
| Delayed data | dữ liệu đến muộn hoặc sai thứ tự |
| Unauthorized data | truy cập/hành động ngoài quyền |
| Human error | người nhập/duyệt nhầm |
| System failure | hạ tầng/tích hợp/model hỏng |

Dấu `→ CON-S-nnn` / `→ CON-F-nnn` chỉ kịch bản/ca hỏng đã có ở 05/07 phản ánh dòng đó. Chỉ số (`reported_fuel`, `issued_fuel`…) theo [03 §1](03-data-map.md).

## 2. Tổng quan ba mươi domain

| # | Khoá | Domain | Phòng ban chính | Quy trình / chỉ số tiêu biểu |
|---|---|---|---|---|
| 01 | `project_planning` | Project Planning | Quản lý dự án | kế hoạch tuần/tháng, mốc |
| 02 | `tender_bid` | Tender / Bid | Đấu thầu | hồ sơ dự thầu, BOQ dự thầu |
| 03 | `contract_management` | Contract Management | Pháp chế | hợp đồng, phụ lục, điều khoản phạt |
| 04 | `boq_quantity` | BOQ / Quantity | Kỹ thuật, đấu thầu | khối lượng thiết kế, `design_quantity` |
| 05 | `budget` | Budget | Tài chính | ngân sách dự án, `committed_cost` |
| 06 | `procurement` | Procurement | Vật tư | `material-request`, PO |
| 07 | `supplier` | Supplier | Vật tư | danh mục và đánh giá nhà cung cấp |
| 08 | `warehouse` | Warehouse | Kho | PNK/PXK, tồn kho, `issued_*` |
| 09 | `material` | Material | Kho, vật tư | thép, xi măng, nhựa đường |
| 10 | `equipment` | Equipment | Máy móc | giờ máy, điều động |
| 11 | `fuel` | Fuel | Kho, máy móc | bốn chỉ số nhiên liệu, `fuel-reconciliation` |
| 12 | `workforce` | Workforce | Nhân sự, công trường | `reported_headcount`, `attendance_headcount` |
| 13 | `subcontractor` | Subcontractor | Quản lý dự án, pháp chế | hợp đồng thầu phụ, khối lượng thầu phụ |
| 14 | `daily_site_operations` | Daily Site Operations | Công trường | `daily-site-report` |
| 15 | `progress_tracking` | Progress Tracking | Quản lý dự án, kỹ thuật | `reported_progress`, `measured_quantity`, `accepted_quantity` |
| 16 | `scheduling` | Scheduling | Quản lý dự án | kế hoạch tiến độ, `schedule_variance` |
| 17 | `qaqc` | QA/QC | QA/QC | thí nghiệm, NCR, `qaqc-inspection` |
| 18 | `inspection` | Inspection | QA/QC, kỹ thuật | kiểm tra hiện trường định kỳ |
| 19 | `safety` | Safety | An toàn | ATLĐ, huấn luyện, cảnh báo |
| 20 | `incident` | Incident | An toàn | `safety-incident` |
| 21 | `drawing_technical_docs` | Drawing / Technical Documents | Kỹ thuật | bản vẽ, thuyết minh, revision |
| 22 | `document_control` | Document Control | Pháp chế, IT | phiên bản, hiệu lực, `document-revision` |
| 23 | `change_order` | Change Order | Kỹ thuật, pháp chế | `change-order` |
| 24 | `rfi` | RFI | Kỹ thuật | `rfi-handling` |
| 25 | `acceptance` | Acceptance | Kỹ thuật, QA | `quantity-acceptance` |
| 26 | `payment_invoice` | Payment / Invoice | Kế toán | `payment-certification`, `billed_*` |
| 27 | `cost_control` | Cost Control | Tài chính | chi phí cam kết/hoá đơn/đã trả |
| 28 | `project_reporting` | Project Reporting | Quản lý dự án | báo cáo tuần/tháng cho CĐT |
| 29 | `maintenance` | Maintenance | Máy móc | `equipment-maintenance` |
| 30 | `management_dashboard` | Management Dashboard | Ban giám đốc | `management-briefing`, bảng điều khiển |

---

## 3. Ba mươi domain

### 01 `project_planning` — Project Planning

| Loại ca | Ca cụ thể |
|---|---|
| Normal | trưởng dự án lập kế hoạch tuần 43 của PRJ-A từ BOQ và tiến độ tổng |
| Edge | PRJ-D (bảo trì) không có BOQ, kế hoạch lặp theo chu kỳ tuần |
| Failure | kế hoạch dựa trên BOQ-A-v1 trong khi v2 đã hiệu lực → khối lượng kế hoạch sai (→ CON-F-023) |
| Dirty data | bảng kế hoạch Excel lẫn ngày `19.10.26` và `19/10/2026` |
| Conflicting data | kế hoạch tuần của trưởng dự án lệch kế hoạch tổng của ban giám đốc về hạn hạng mục M2 |
| Missing data | thiếu mốc hoàn thành hạng mục M3 trong kế hoạch |
| Delayed data | kế hoạch tuần nộp sau khi tuần đã bắt đầu 2 ngày |
| Unauthorized data | giám sát hiện trường xin xem kế hoạch giá thành/dòng tiền (ngoài vai) (→ CON-F-053) |
| Human error | nhập nhầm năm 2025 thay 2026 cho mốc hạng mục |
| System failure | ERP dự án không đọc được, hệ thống nói rõ đang dùng kế hoạch cũ kèm cờ (→ CON-F-003) |

### 02 `tender_bid` — Tender / Bid

| Loại ca | Ca cụ thể |
|---|---|
| Normal | dự toán lập giá dự thầu gói cầu mới từ BOQ mẫu và đơn giá nội bộ |
| Edge | hồ sơ mời thầu bổ sung phụ lục sau khi đã lập giá |
| Failure | giá dự thầu dùng đơn giá thép lỗi thời → chào giá lỗ |
| Dirty data | đơn giá ghi `15tr5/tấn` và `15.500.000` lẫn nhau |
| Conflicting data | bản vẽ ghi 1.250 m³ bê tông, BOQ mẫu ghi 1.200 m³ (→ CON-S-088) |
| Missing data | thiếu bản vẽ một hạng mục khi bóc khối lượng |
| Delayed data | báo giá nhà cung cấp đến sau hạn nộp thầu |
| Unauthorized data | nhân viên công trường truy vấn giá chào của gói đang thầu |
| Human error | nhân nhầm đơn giá cho khối lượng sai đơn vị (m² thay m³) |
| System failure | kho tài liệu không mở được hồ sơ mời thầu, hệ thống không bịa điều kiện thầu |

### 03 `contract_management` — Contract Management

| Loại ca | Ca cụ thể |
|---|---|
| Normal | hỏi điều khoản điều chỉnh giá nhiên liệu trong HD-A-2026-01 và nhận trích dẫn Điều 8 (→ CON-S-025) |
| Edge | phụ lục 1 sửa mức phạt của điều khoản gốc: phải nêu cả hai (→ CON-F-026) |
| Failure | trích điều khoản của hợp đồng khác do tên gần giống |
| Dirty data | hợp đồng scan nghiêng, bảng điều khoản mất cột (→ CON-F-007) |
| Conflicting data | phụ lục mâu thuẫn điều khoản gốc về hạn thanh toán |
| Missing data | chưa nạp phụ lục 2 nhưng người dùng hỏi mức phạt hiện hành |
| Delayed data | phụ lục ký ngày 15/10 nạp ngày 19/10, trả lời ngày 17/10 cần nêu chưa có |
| Unauthorized data | giám sát hỏi điều khoản giá trị hợp đồng (chỉ pháp chế/BGĐ) |
| Human error | gắn nhãn "đang hiệu lực" cho cả hai phiên bản hợp đồng (→ CON-S-087) |
| System failure | dịch vụ đọc tài liệu sập khi nạp phụ lục mới (→ CON-F-009) |

### 04 `boq_quantity` — BOQ / Quantity

| Loại ca | Ca cụ thể |
|---|---|
| Normal | tra khối lượng bê tông trụ P2 theo BOQ-A-v2 |
| Edge | một hạng mục có hai đơn vị trong BOQ (m³ và tấn) |
| Failure | BOQ và bản vẽ lệch mà hệ thống gộp thành một số (→ CON-F-029) |
| Dirty data | khối lượng ghi `1.2k m3` trong ô bảng tính |
| Conflicting data | BOQ-A-v2 ghi 1.200 m³, bản vẽ ghi 1.250 m³ (→ CON-S-088) |
| Missing data | hạng mục mới chưa có dòng BOQ |
| Delayed data | BOQ cập nhật theo phát sinh nhưng chưa nạp |
| Unauthorized data | nhà thầu phụ xem đơn giá BOQ của ABC |
| Human error | nhập khối lượng nguyên khối thay cho tơi xốp (→ CON-S-038) |
| System failure | trích bảng BOQ từ PDF bị mất hàng |

### 05 `budget` — Budget

| Loại ca | Ca cụ thể |
|---|---|
| Normal | xem ngân sách còn lại của PRJ-B theo hạng mục |
| Edge | ngân sách điều chỉnh giữa kỳ theo phụ lục |
| Failure | ngân sách tính trên `committed_cost` nhưng báo như `paid_cost` (→ CON-F-063) |
| Dirty data | ngân sách ghi `1 tỷ 2` và `1.200 triệu` lẫn nhau |
| Conflicting data | ngân sách ERP lệch bảng tính của trưởng dự án |
| Missing data | hạng mục chưa có ngân sách phân bổ |
| Delayed data | số chi phí tháng chưa chốt khi báo cáo giữa tháng |
| Unauthorized data | trưởng dự án B xem ngân sách dự án A (→ CON-S-093) |
| Human error | nhập ngân sách theo triệu thay vì đồng |
| System failure | ERP tài chính không phản hồi, không dùng số cache hết hạn (→ CON-F-057) |

### 06 `procurement` — Procurement

| Loại ca | Ca cụ thể |
|---|---|
| Normal | đề nghị 2 tấn thép D16 đi qua duyệt của trưởng dự án rồi tạo PO (→ CON-S-029) |
| Edge | đề nghị vượt hạn mức cần thêm giám đốc (→ CON-S-070) |
| Failure | phiếu được duyệt khi thiếu báo giá (→ CON-F-041) |
| Dirty data | `20 tan thep d16` không dấu, không đơn vị chuẩn |
| Conflicting data | tồn thấp kích hoạt đề nghị mới trong khi đã có phiếu mở cho cùng vật tư |
| Missing data | đề nghị thiếu kho nhận và đơn vị (→ CON-S-042) |
| Delayed data | nhà cung cấp báo hết hàng sau khi phiếu đã APPROVED |
| Unauthorized data | người lập phiếu tự duyệt (→ CON-F-042) |
| Human error | gõ nhầm 20 tấn thay 2 tấn |
| System failure | ERP trả 503 khi tạo PO (→ CON-S-071) |

### 07 `supplier` — Supplier

| Loại ca | Ca cụ thể |
|---|---|
| Normal | xem nhà cung cấp thép đã giao đúng hạn trong quý |
| Edge | nhà cung cấp đổi tên pháp lý giữa hai hợp đồng |
| Failure | hai bản ghi cùng một nhà cung cấp làm đánh giá bị tách đôi |
| Dirty data | tên `Cty Thep Dai Viet` và `Công ty Thép Đại Việt` |
| Conflicting data | hai mã số thuế khác nhau cho cùng nhà cung cấp |
| Missing data | nhà cung cấp mới chưa có điều khoản thanh toán |
| Delayed data | phiếu đánh giá giao hàng nhập muộn một tháng |
| Unauthorized data | công trường xem điều khoản giá của nhà cung cấp |
| Human error | chọn nhầm nhà cung cấp cùng tên khác tỉnh |
| System failure | không đọc được danh mục nhà cung cấp, không đoán tên |

### 08 `warehouse` — Warehouse

| Loại ca | Ca cụ thể |
|---|---|
| Normal | thủ kho ghi PXK-0412 xuất 3.850 L, tồn còn 8.150 L (→ CON-S-033) |
| Edge | xuất đúng bằng tồn còn lại, tồn về 0 |
| Failure | xuất 3.850 L khi tồn 3.000 L làm âm kho (→ CON-F-035) |
| Dirty data | phiếu `pnk 0233` viết thường, thiếu gạch ngang (→ BR-UNIT-06) |
| Conflicting data | PXK-0412 3.850 L và ERP 3.900 L (→ CON-S-086) |
| Missing data | phiếu xuất không ghi số phiếu |
| Delayed data | phiếu giấy nhập ERP trễ 2 ngày, tồn kho cũ (→ CON-F-003) |
| Unauthorized data | công nhân hỏi tồn và đơn giá kho của dự án khác |
| Human error | ghi nhầm xuất thành nhập |
| System failure | tool ghi nhập kho hết giờ giữa chừng (→ CON-F-036) |

### 09 `material` — Material

| Loại ca | Ca cụ thể |
|---|---|
| Normal | tồn thép D16 kho A1 còn 8 tấn dưới mức an toàn 10 tấn (→ CON-S-059) |
| Edge | vật tư đổi quy cách (D16 sang D18) giữa đợt đặt |
| Failure | vật tư rời "xi 2k" bị đoán thành 2.000 kg (→ CON-S-055) |
| Dirty data | `xi 2 tấn`, `2000kg xi`, `40 bao xi` cùng một lượng |
| Conflicting data | phiếu nhập 20 tấn nhưng hoá đơn 19,5 tấn |
| Missing data | vật tư chưa có mã trong danh mục |
| Delayed data | phiếu giao hàng đến sau khi vật tư đã dùng |
| Unauthorized data | nhà thầu phụ xem tồn kho vật tư của ABC |
| Human error | nhập nhầm đơn vị bao thay vì tấn |
| System failure | ảnh phiếu giao hàng không đọc được (→ CON-F-008) |

### 10 `equipment` — Equipment

| Loại ca | Ca cụ thể |
|---|---|
| Normal | tổng giờ máy EX-01 tuần này theo đồng hồ (→ CON-S-002) |
| Edge | máy điều động giữa PRJ-A và PRJ-B trong cùng ngày |
| Failure | "máy 03" gán nhầm sang máy của dự án khác (→ CON-F-015) |
| Dirty data | `máy xúc chạy 7h`, `7 tiếng`, `7 tiếng máy` |
| Conflicting data | giờ báo 7 h, đồng hồ 6,4 h, chấm công ca máy 8 h (→ CON-S-084) |
| Missing data | đồng hồ giờ máy không có số liệu ngày hôm qua |
| Delayed data | số đo giờ máy đến sau khi báo cáo ngày đã nộp |
| Unauthorized data | giám sát xem chi phí ca máy |
| Human error | gộp nhầm EX-03 và EX03 (→ CON-S-040) |
| System failure | đồng hồ/cảm biến mất sóng, trùng số đo (→ CON-F-050) |

### 11 `fuel` — Fuel

| Loại ca | Ca cụ thể |
|---|---|
| Normal | "Dự án A tháng này tiêu thụ bao nhiêu dầu?" → tổng `measured_consumption` có nhãn (→ CON-S-001) |
| Edge | thiếu `measured_consumption`, câu trả lời dùng `issued_fuel` và nói rõ nhãn |
| Failure | trộn bốn chỉ số thành một số "đúng" (→ CON-F-063) |
| Dirty data | `dầu 4k`, `DA 4.000L`, `4 khối dầu`, `bốn ngàn lít` (→ CON-S-046…049) |
| Conflicting data | báo 4.000, kho 3.850, đồng hồ 3.782, hoá đơn 4.100 → derived fact, không chọn bên (→ CON-S-081) |
| Missing data | chưa có hoá đơn tháng; chỉ có kho và báo |
| Delayed data | hoá đơn nhiên liệu đến sau khi kỳ đã đối soát (→ CON-S-060) |
| Unauthorized data | giám sát xem chi phí nhiên liệu toàn công ty |
| Human error | "xác nhận" `4k dầu hôm qua đúng không?` khi chưa rõ chỉ số: phải hỏi lại, không xác nhận |
| System failure | PostgreSQL chập chờn, không dùng số cũ làm số hiện tại (→ CON-S-096) |

### 12 `workforce` — Workforce

| Loại ca | Ca cụ thể |
|---|---|
| Normal | số người có mặt của PRJ-B theo chấm công (→ CON-S-003) |
| Edge | người làm hai dự án trong một ngày |
| Failure | `NC 22` ghi như 22 công (→ CON-F-016/BR-LAB-01) |
| Dirty data | `NC 22`, `22 người`, `22 nhân công`, `hai hai` |
| Conflicting data | báo 22, chấm công 20 (→ CON-S-085) |
| Missing data | thiếu chấm công của nhà thầu phụ |
| Delayed data | bảng công chốt cuối kỳ nhưng hỏi giữa kỳ |
| Unauthorized data | giám sát hỏi lương đội trưởng (→ CON-S-092) |
| Human error | nhập 220 thay 22 |
| System failure | hệ thống chấm công không phản hồi; trả lỗi có nhãn, không bịa số |

### 13 `subcontractor` — Subcontractor

| Loại ca | Ca cụ thể |
|---|---|
| Normal | tra điều khoản phạt chậm tiến độ trong HD-NTP-A-03 (→ CON-S-016) |
| Edge | nhà thầu phụ thi công hai dự án với hai hợp đồng khác nhau |
| Failure | trích điều khoản gốc, bỏ sót Phụ lục 1 (→ CON-F-026) |
| Dirty data | khối lượng thầu phụ báo `5 khối cọc` không nêu đường kính |
| Conflicting data | khối lượng thầu phụ báo lớn hơn khối lượng đo của ABC |
| Missing data | thiếu biên bản xác nhận khối lượng thầu phụ tháng này |
| Delayed data | hoá đơn thầu phụ đến sau kỳ thanh toán |
| Unauthorized data | thầu phụ khác xem hợp đồng của thầu phụ này |
| Human error | gán khối lượng của thầu phụ A cho thầu phụ B |
| System failure | không đọc được PDF hợp đồng thầu phụ (→ CON-F-009) |

### 14 `daily_site_operations` — Daily Site Operations

| Loại ca | Ca cụ thể |
|---|---|
| Normal | `Ca sáng làm 350m / máy xúc 7h / dầu 4k / 22 người / mưa từ 14h` thành 5 fact (→ CON-S-052) |
| Edge | báo cáo ca đêm nhận lúc 00:40 ngày hôm sau (→ CON-S-065) |
| Failure | báo cáo nộp dù thiếu giờ máy (→ CON-F-004) |
| Dirty data | `DA 4.000L / KL 0.35km / NC 22 / Mưa chiều` (→ CON-S-047, 050, 053) |
| Conflicting data | hai giám sát báo hai số khác nhau cho cùng ngày |
| Missing data | không có tin nhắn nào của PRJ-C đến 17:00 (→ CON-S-062) |
| Delayed data | tin nhắn đến sau khi báo cáo đã SUBMITTED |
| Unauthorized data | chỉ huy trưởng dự án khác sửa báo cáo |
| Human error | gửi cùng báo cáo vào hai nhóm chat (→ CON-S-063) |
| System failure | nhóm chat không nhận được câu hỏi lại, thông báo thất bại (→ 01 §6) |

### 15 `progress_tracking` — Progress Tracking

| Loại ca | Ca cụ thể |
|---|---|
| Normal | khối lượng bê tông trụ P2 đã nghiệm thu lũy kế (→ CON-S-004) |
| Edge | tiến độ đo bằng hai đơn vị (m và m³) cho cùng hạng mục |
| Failure | báo `reported_progress` bị coi là `accepted_quantity` |
| Dirty data | `khoảng ba trăm năm chục mét` → 350 m cờ ước lượng (→ CON-S-051) |
| Conflicting data | báo 350 m, đo 332 m, nghiệm thu 320 m (→ CON-S-083) |
| Missing data | hạng mục chưa có biên bản đo tuần này |
| Delayed data | xác nhận TVGS đến muộn 3 ngày |
| Unauthorized data | dự án khác xem tiến độ |
| Human error | nhập 3.500 thay 350 |
| System failure | không truy được bảng tiến độ, không dùng số cũ (→ CON-F-003) |

### 16 `scheduling` — Scheduling

| Loại ca | Ca cụ thể |
|---|---|
| Normal | "Dự án nào đang chậm tiến độ nhất?" tính `schedule_variance` (→ CON-S-023) |
| Edge | hạng mục có hai kế hoạch (gốc và điều chỉnh) |
| Failure | so tiến độ với kế hoạch cũ đã bị thay |
| Dirty data | ngày kế hoạch lẫn định dạng `19-10` và `19/10` |
| Conflicting data | kế hoạch ERP và bảng tính chênh nhau về hạn M2 |
| Missing data | thiếu khối lượng kế hoạch lũy kế tuần này |
| Delayed data | kế hoạch điều chỉnh có hiệu lực từ 15/10 nhưng nạp ngày 19/10 |
| Unauthorized data | giám sát đổi kế hoạch |
| Human error | kéo nhầm hạng mục sang tuần khác |
| System failure | công cụ lịch lỗi, không bịa số ngày còn lại |

### 17 `qaqc` — QA/QC

| Loại ca | Ca cụ thể |
|---|---|
| Normal | mẫu bê tông đạt, phiếu kiểm tra PASSED rồi CLOSED |
| Edge | mẫu đạt nhưng NCR cũ của hạng mục còn mở (không đóng được) (→ BR-QC / guard `no_open_ncr`) |
| Failure | đóng phiếu khi còn NCR mở |
| Dirty data | kết quả ghi `28MPa`, `28 MPa`, `28 mpa` |
| Conflicting data | hai phòng thí nghiệm cho hai kết quả cho cùng mẫu |
| Missing data | thiếu kết quả mẫu 7 ngày |
| Delayed data | kết quả 28 ngày đến sau khi đã đổ lớp trên |
| Unauthorized data | kỹ sư QA duyệt thanh toán (→ CON-S-099) |
| Human error | gắn kết quả cho sai hạng mục |
| System failure | phòng thí nghiệm không gửi được kết quả điện tử |

### 18 `inspection` — Inspection

| Loại ca | Ca cụ thể |
|---|---|
| Normal | kiểm tra hiện trường định kỳ tuần 43, biên bản đủ chữ ký |
| Edge | TVGS kiểm tra đột xuất ngoài lịch |
| Failure | bỏ sót hạng mục trong danh mục kiểm tra |
| Dirty data | biên bản viết tay, số đo lẫn đơn vị |
| Conflicting data | biên bản của TVGS và của ABC khác nhau về một sai lệch |
| Missing data | thiếu ảnh minh chứng |
| Delayed data | biên bản nộp sau 5 ngày |
| Unauthorized data | nhà thầu phụ xem biên bản nội bộ của ABC |
| Human error | ghi nhầm vị trí kiểm tra (P2 thành P3) |
| System failure | ảnh minh chứng không tải lên được; xin gửi lại, không coi là đã nộp |

### 19 `safety` — Safety

| Loại ca | Ca cụ thể |
|---|---|
| Normal | huấn luyện ATLĐ đầu tháng, danh sách đủ người |
| Edge | cảnh báo mưa lớn ngày mai → human task quyết hoãn đổ bê tông (→ CON-S-064) |
| Failure | không nhắc huấn luyện cho người mới điều động |
| Dirty data | danh sách huấn luyện thiếu mã người |
| Conflicting data | danh sách huấn luyện và danh sách có mặt không khớp |
| Missing data | thiếu biên bản huấn luyện của nhà thầu phụ |
| Delayed data | cảnh báo thời tiết đến trễ |
| Unauthorized data | công nhân xem hồ sơ sự cố có tên người |
| Human error | bỏ qua cảnh báo vì tin nhắn trôi trong nhóm |
| System failure | nguồn cảnh báo thời tiết không phản hồi (→ 01 §6) |

### 20 `incident` — Incident

| Loại ca | Ca cụ thể |
|---|---|
| Normal | mở sự cố ngã ở trụ P2, mức tối thiểu 2, human task xác nhận (→ CON-S-036) |
| Edge | sự cố suýt xảy ra (near-miss) cần thống kê tách riêng (→ BR-SAF-03) |
| Failure | mức do model đề xuất thấp hơn luật (→ CON-F-033) |
| Dirty data | mô tả `ngã nhẹ nhưng gãy tay` mâu thuẫn mức |
| Conflicting data | hai người kể hai thời điểm khác nhau |
| Missing data | thiếu người chứng kiến |
| Delayed data | báo sự cố sau 1 ngày |
| Unauthorized data | thông tin sự cố gửi sang nhóm không liên quan |
| Human error | chọn nhầm dự án cho sự cố |
| System failure | thông báo cán bộ an toàn thất bại, vẫn ghi sự cố và tạo human task |

### 21 `drawing_technical_docs` — Drawing / Technical Documents

| Loại ca | Ca cụ thể |
|---|---|
| Normal | tra bản vẽ `BV-A-P2-03` đang ở revision C |
| Edge | hai revision cùng được phát hành trong một tuần |
| Failure | hiện trường dùng revision B đã bị thay |
| Dirty data | tên tệp `BV A P2 03 revC.pdf` lẫn dấu cách/gạch |
| Conflicting data | bản vẽ ghi 1.250 m³, BOQ ghi 1.200 m³ (→ CON-S-088) |
| Missing data | thiếu bản vẽ chi tiết cốt thép của trụ P3 |
| Delayed data | revision mới phát hành nhưng chưa nạp |
| Unauthorized data | nhà thầu phụ xem bản vẽ ngoài hạng mục của họ |
| Human error | tải lên nhầm bản vẽ của dự án khác |
| System failure | parser mất bảng ghi chú trên bản vẽ (→ CON-F-007) |

### 22 `document_control` — Document Control

| Loại ca | Ca cụ thể |
|---|---|
| Normal | tìm biên bản `BB-NT-0415` theo mã (→ CON-S-015) |
| Edge | SOP đổi phiên bản đúng ngày nghiệp vụ cần hỏi |
| Failure | trích dẫn SOP bản hết hiệu lực (→ CON-F-065) |
| Dirty data | mã `BB NT 0415`, `bb-nt-0415` |
| Conflicting data | SOP v2 và v3 cùng nhãn hiệu lực (→ CON-S-087) |
| Missing data | SOP chưa được nạp (→ CON-F-024) |
| Delayed data | bản mới ký hôm qua nhưng công bố hôm nay |
| Unauthorized data | tenant khác hỏi tài liệu của ABC (→ CON-S-091) |
| Human error | gắn nhãn hiệu lực cho cả hai bản |
| System failure | nạp SOP v4 khi parser sập, giữ v3 (→ CON-S-097) |

### 23 `change_order` — Change Order

| Loại ca | Ca cụ thể |
|---|---|
| Normal | `PS-012` đi qua kỹ thuật, định giá, CĐT, phụ lục hợp đồng (→ CON-S-075) |
| Edge | phát sinh nhỏ dưới hạn mức không cần CĐT |
| Failure | lập phát sinh thiếu cơ sở đo khối lượng (→ BR-CHG-01) |
| Dirty data | `đào thêm 300 khối đá lẫn` không nói nguyên khối hay tơi xốp |
| Conflicting data | khối lượng phát sinh của ABC lệch số của TVGS |
| Missing data | thiếu ảnh/biên bản hiện trường chứng minh phát sinh |
| Delayed data | CĐT trả lời sau 3 tuần |
| Unauthorized data | công trường tự duyệt phát sinh |
| Human error | nhập nhầm đơn giá của hạng mục khác |
| System failure | lỗi khi lập phụ lục, phát sinh giữ ở APPROVED, human task soạn thủ công |

### 24 `rfi` — RFI

| Loại ca | Ca cụ thể |
|---|---|
| Normal | RFI thiếu cao độ đáy móng trụ P3 gửi TVGS (→ CON-S-037) |
| Edge | một RFI liên quan hai bản vẽ |
| Failure | RFI quá hạn không ai biết (→ CON-F-043) |
| Dirty data | mô tả RFI chỉ ghi `thiếu cao độ` không rõ bản vẽ |
| Conflicting data | phản hồi RFI mâu thuẫn bản vẽ hiện hành |
| Missing data | RFI thiếu bản vẽ đính kèm |
| Delayed data | TVGS trả lời sau hạn 5 ngày (→ CON-S-076) |
| Unauthorized data | nhà thầu phụ đóng RFI của ABC |
| Human error | gửi RFI cho sai TVGS |
| System failure | gửi RFI thất bại, vẫn giữ nháp và báo lỗi |

### 25 `acceptance` — Acceptance

| Loại ca | Ca cụ thể |
|---|---|
| Normal | nghiệm thu hạng mục M2 đủ xác nhận TVGS và CĐT (→ CON-S-072) |
| Edge | nghiệm thu từng phần một hạng mục |
| Failure | chốt nghiệm thu khi mới có một bên xác nhận (→ CON-F-044) |
| Dirty data | biên bản đo `332m` và `0.332km` |
| Conflicting data | số đo của ABC và số của TVGS lệch |
| Missing data | thiếu chữ ký CĐT |
| Delayed data | TVGS xác nhận chậm 3 ngày |
| Unauthorized data | kỹ sư QA ký thay CĐT |
| Human error | gắn biên bản vào sai hạng mục |
| System failure | không nạp được biên bản scan |

### 26 `payment_invoice` — Payment / Invoice

| Loại ca | Ca cụ thể |
|---|---|
| Normal | đối chiếu ba bên hoá đơn `INV-2026-0912` và duyệt (→ CON-S-074) |
| Edge | hoá đơn chia hai đợt giao |
| Failure | duyệt khi PO và nhập kho lệch |
| Dirty data | số tiền `341tr` và `341.000.000` lẫn nhau |
| Conflicting data | hai hoá đơn cùng số khác giá trị |
| Missing data | thiếu biên bản nhập kho để đối chiếu |
| Delayed data | hoá đơn đến sau kỳ thanh toán |
| Unauthorized data | kỹ sư QA duyệt thanh toán (→ CON-S-099) |
| Human error | thanh toán hai lần cùng hoá đơn |
| System failure | ERP lỗi khi ghi lệnh chi |

### 27 `cost_control` — Cost Control

| Loại ca | Ca cụ thể |
|---|---|
| Normal | chi phí nhiên liệu đã nhận theo hoá đơn tháng 9 (→ CON-S-007) |
| Edge | chi phí phát sinh chưa có hoá đơn (ghi nhận tạm) |
| Failure | trộn `committed_cost` và `paid_cost` |
| Dirty data | `500k tiền xe`, `2tr5 tiền ăn` (→ BR-UNIT-01) |
| Conflicting data | chi phí ERP lệch bảng tính dự án |
| Missing data | hạng mục chưa có mã chi phí |
| Delayed data | hoá đơn nhiên liệu đến muộn làm chênh lệch tồn tại (→ CON-S-060) |
| Unauthorized data | giám sát xem giá vốn |
| Human error | gán chi phí vào dự án khác |
| System failure | ERP tài chính trễ, báo số cũ (→ CON-F-057) |

### 28 `project_reporting` — Project Reporting

| Loại ca | Ca cụ thể |
|---|---|
| Normal | báo cáo tuần của PRJ-A tổng hợp từ báo cáo ngày đã duyệt |
| Edge | tuần có hai ngày mưa không thi công |
| Failure | báo cáo tuần gộp ước lượng vào tổng chính xác (→ CON-F-014) |
| Dirty data | báo cáo ngày thiếu đơn vị ở một số dòng |
| Conflicting data | báo cáo tuần của PRJ-B lệch tổng của các báo cáo ngày |
| Missing data | thiếu báo cáo ngày thứ Tư |
| Delayed data | báo cáo ngày nộp sau khi tuần đã chốt |
| Unauthorized data | chỉ huy trưởng xem báo cáo dự án khác (→ CON-S-093) |
| Human error | gửi báo cáo tuần cho sai CĐT |
| System failure | dựng báo cáo lỗi giữa chừng, không gửi bản cụt |

### 29 `maintenance` — Maintenance

| Loại ca | Ca cụ thể |
|---|---|
| Normal | phiếu `MT-0145` đi OPEN → ASSIGNED → INSPECTING → VERIFIED → CLOSED (→ CON-S-077) |
| Edge | máy được điều sang dự án khác khi phiếu đang mở |
| Failure | mở hai phiếu cho cùng một cảnh báo (→ CON-F-050) |
| Dirty data | cảnh báo `EX-03 112C` thiếu đơn vị nhiệt độ |
| Conflicting data | cảm biến báo 112 °C, thợ báo "bình thường" |
| Missing data | máy không có lịch sử bảo dưỡng trước |
| Delayed data | số đo cũ đến sau số đo mới (→ CON-F-061) |
| Unauthorized data | công nhân đóng phiếu bảo dưỡng |
| Human error | giao việc cho kỹ thuật viên đã rời dự án (→ CON-F-047) |
| System failure | cảm biến sập, không có số đo, không mở phiếu sai |

### 30 `management_dashboard` — Management Dashboard

| Loại ca | Ca cụ thể |
|---|---|
| Normal | giám đốc xem tổng hợp năm chiều của PRJ-A (→ CON-S-020, workflow `management-briefing`) |
| Edge | một chiều không có dữ liệu → hiển thị "chưa có dữ liệu" |
| Failure | kết luận "không có vấn đề" dù thiếu chiều dữ liệu (→ CON-F-040) |
| Dirty data | chỉ số lẫn nhãn (`dầu` không nêu chỉ số) |
| Conflicting data | bảng điều khiển lệch báo cáo tháng |
| Missing data | dự án chưa có báo cáo ngày hôm nay |
| Delayed data | số đối soát nhiên liệu cuối kỳ chưa có |
| Unauthorized data | phó giám đốc xem dự án ngoài được giao |
| Human error | hỏi `Nên mua cổ phiếu nào?` (ngoài phạm vi) (→ CON-S-021) |
| System failure | dịch vụ model hết giờ, trả bản dựng sẵn từ số liệu (→ CON-F-031) |
