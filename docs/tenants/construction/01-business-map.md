# 01 — Bản đồ doanh nghiệp (ABC Infrastructure)

> Tenant `abc-infra` (mã `CON`) — **ABC Infrastructure JSC**, doanh nghiệp xây dựng cầu đường và hạ tầng **hư cấu**. Mọi công ty, người, máy, kho, nhà thầu phụ, hợp đồng và **mọi con số** trong tài liệu này là **ví dụ giả lập**, không phải kết quả đo (CONVENTIONS §6, §7).
> Tài liệu liên quan: [02-domain-map](02-domain-map.md) · [03-data-map](03-data-map.md) · [04-workflow-map](04-workflow-map.md) · [05-problem-catalog](05-problem-catalog.md) · [06-ai-opportunity-map](06-ai-opportunity-map.md) · [07-failure-catalog](07-failure-catalog.md) · [08-test-cases](08-test-cases.md).

## 0. Vì sao tenant này được chọn

ABC là **Reference Tenant #01**: dùng để *khám phá* kiến trúc EAIS. Dữ liệu của nó đi từ **người nhập bẩn** (tin nhắn "dầu 4k", phiếu kho viết tay) → **tài liệu** (SOP, hợp đồng, bản vẽ) → **nguồn sự thật** (bốn chỉ số nhiên liệu khác nhau) → **workflow** (duyệt vật tư, nghiệm thu, sự cố) → **số liệu chính xác** (tổng dầu, giờ máy, khối lượng). Doanh nghiệp không chỉ *hỏi* mà còn *ra lệnh* và *nhận sự kiện* (cảm biến, ERP), nên nó ép đủ ba lối vào QUERY · COMMAND · EVENT của kiến trúc ([01-architecture §2.1](../../01-architecture.md)).

---

## 1. Mô hình kinh doanh

| Thành phần | Mô tả (ví dụ giả lập) |
|---|---|
| Sản phẩm/dịch vụ | thi công cầu; thi công đường; thi công hạ tầng đô thị (thoát nước, vỉa hè, chiếu sáng); duy tu bảo trì tuyến đường |
| Nguồn doanh thu | thanh toán theo khối lượng đã nghiệm thu của hợp đồng thi công ký với chủ đầu tư (CĐT); phát sinh được CĐT chấp thuận; hợp đồng bảo trì định kỳ |
| Chi phí chính | vật tư (thép, xi măng, đá, nhựa đường), nhiên liệu (dầu diesel), ca máy, nhân công, nhà thầu phụ, an toàn lao động (ATLĐ), chi phí chung |
| Cách làm việc | mỗi dự án có chỉ huy trưởng công trường, đội thi công, kho riêng; trụ sở có các phòng chức năng; máy móc điều động giữa dự án |
| Điểm đau số liệu | báo cáo hiện trường qua nhóm chat viết tắt, lẫn đơn vị; phiếu kho giấy; nhiều bản bảng tính; tài liệu nhiều phiên bản; các nguồn không trùng nhau |

### 1.1 Bốn dự án tham chiếu

| Mã | Loại | Tên giả lập | Đặc điểm dữ liệu |
|---|---|---|---|
| `PRJ-A` | Cầu | Cầu Bến Nứa | khối lượng bê tông/cọc theo hạng mục trụ (P1, P2, P3…); nhiều nhà thầu phụ (cọc khoan nhồi, cốp pha); máy xúc, cẩu, khoan cọc |
| `PRJ-B` | Đường | Đường Vành Đai Nam (gói 2) | tiến độ đo bằng chiều dài (m, km); xe ủi, xe lu, xe ben; nhiều nhóm chat |
| `PRJ-C` | Hạ tầng đô thị | Hạ tầng Khu đô thị Hoa Sen | nhiều hạng mục nhỏ song song (ống thoát nước, vỉa hè); nhiều đội; hồ sơ nghiệm thu từng hạng mục |
| `PRJ-D` | Bảo trì | Bảo trì tuyến Đường Tây Giang | công việc lặp theo tuần; ít máy (một máy xúc `EX-D1`); báo cáo ngắn |

---

## 2. Khách hàng và bên liên quan ngoài tổ chức

| Bên | Vai trò với ABC | Dữ liệu trao đổi | Điểm chạm vào EAIS |
|---|---|---|---|
| Chủ đầu tư (CĐT) | ký hợp đồng, nghiệm thu, thanh toán, chấp thuận phát sinh | hợp đồng, biên bản nghiệm thu, hồ sơ thanh toán | tài liệu (I1); xác nhận do nhân viên hồ sơ nhập thay (human task) |
| Tư vấn giám sát (TVGS) | giám sát chất lượng/khối lượng thay CĐT; trả lời RFI | biên bản kiểm tra, xác nhận khối lượng, phản hồi RFI | tài liệu (I1); xác nhận khối lượng (`accepted_quantity`) |
| Nhà thầu phụ | thi công hạng mục chuyên biệt (cọc, cốp pha, điện) | hợp đồng thầu phụ, khối lượng, hoá đơn | hợp đồng (I1), hoá đơn (I2), nhân công trên công trường |
| Nhà cung cấp | cấp vật tư, nhiên liệu, nhựa đường | báo giá, PO, phiếu giao hàng, hoá đơn | I2 (ERP), hoá đơn `billed_*` |
| Đơn vị thí nghiệm | thí nghiệm bê tông, đất, nhựa đường | kết quả thí nghiệm | tài liệu (I1) + bản ghi có kiểu |
| Cơ quan quản lý nhà nước | yêu cầu hồ sơ, kiểm tra ATLĐ và chất lượng (nêu khái quát, xem §8) | báo cáo định kỳ, biên bản sự cố | đầu ra của workflow sự cố/QA-QC |
| Ngân hàng/bảo lãnh | bảo lãnh thực hiện, thanh toán | thư bảo lãnh, lệnh chi | ngoài phạm vi MVP |

---

## 3. Mười bốn phòng ban

Brief gốc liệt kê 13 mục, trong đó **"Vật tư / Kho"** được tách thành hai phòng ban vì luồng dữ liệu khác nhau (mua sắm so với ghi sổ kho), nên có **14 phòng ban**. Mỗi phòng ban có một bảng theo chuỗi **Actor → Trigger → Input → Process → Decision → Output → Downstream → Data produced → System touched → Exceptions → Failure consequences**.

| # | Phòng ban | Vai trò EAIS chính (xem §4) | Domain liên quan (02) |
|---|---|---|---|
| 1 | Ban giám đốc | `director`, `deputy_director` | `management_dashboard`, `budget`, `contract_management` |
| 2 | Đấu thầu | `bidding_estimator` | `tender_bid`, `boq_quantity`, `budget` |
| 3 | Quản lý dự án | `project_manager` | `project_planning`, `scheduling`, `progress_tracking`, `project_reporting` |
| 4 | Kỹ thuật | `technical_engineer` | `drawing_technical_docs`, `rfi`, `boq_quantity`, `acceptance`, `change_order` |
| 5 | Công trường | `site_commander`, `site_supervisor` | `daily_site_operations`, `workforce`, `subcontractor`, `progress_tracking` |
| 6 | Vật tư (mua sắm) | `procurement_officer` | `procurement`, `supplier`, `material` |
| 7 | Kho | `warehouse_keeper` | `warehouse`, `material`, `fuel` |
| 8 | Máy móc thiết bị | `equipment_manager`, `maintenance_technician` | `equipment`, `fuel`, `maintenance` |
| 9 | QA/QC | `qa_qc_engineer` | `qaqc`, `inspection` |
| 10 | An toàn | `safety_officer` | `safety`, `incident` |
| 11 | Tài chính / Kế toán | `finance_manager`, `accountant` | `payment_invoice`, `cost_control`, `budget` |
| 12 | Nhân sự | `hr_officer` | `workforce` |
| 13 | Pháp chế / Hợp đồng | `legal_officer`, `records_clerk` | `contract_management`, `document_control`, `subcontractor` |
| 14 | IT / Data | `it_admin` | nền tảng, danh mục, tích hợp |

### 3.1 Ban giám đốc

| Thành phần | Nội dung |
|---|---|
| Actor | giám đốc, phó giám đốc |
| Trigger | họp giao ban tuần; cảnh báo vượt chi phí/chậm tiến độ; đề nghị vượt hạn mức của dự án |
| Input | báo cáo tổng hợp dự án, bảng chi phí, dòng tiền, danh sách phát sinh lớn, đề nghị vật tư vượt hạn mức |
| Process | soát chỉ tiêu theo dự án, so kế hoạch với thực tế, xếp ưu tiên nguồn lực (máy, vốn, nhân lực) |
| Decision | duyệt đề nghị vượt hạn mức; chấp thuận phát sinh lớn; ký hợp đồng thầu phụ lớn; điều máy giữa dự án |
| Output | chỉ đạo bằng văn bản/chat, phê duyệt trong workflow, biên bản họp |
| Downstream | quản lý dự án, kế toán, đấu thầu, pháp chế |
| Data produced | quyết định phê duyệt, biên bản họp, mức hạn mức duyệt |
| System touched | bảng điều khiển quản lý, ERP (phê duyệt), nhóm chat |
| Exceptions | báo cáo thiếu hoặc các nguồn lệch số; hai dự án tranh cùng một máy; giám đốc vắng cần uỷ quyền |
| Failure consequences | quyết định dựa trên số sai (ví dụ dầu "khoảng"); phát hiện chậm vượt chi phí; quyền duyệt không có người thay |

### 3.2 Đấu thầu

| Thành phần | Nội dung |
|---|---|
| Actor | kỹ sư dự toán/đấu thầu (`bidding_estimator`) |
| Trigger | thông báo mời thầu mới; hồ sơ mời thầu được sửa đổi |
| Input | hồ sơ mời thầu, bản vẽ thiết kế, BOQ mẫu, đơn giá nội bộ, định mức |
| Process | bóc tách khối lượng, lập đơn giá và giá dự thầu, soạn hồ sơ kỹ thuật và tài chính |
| Decision | có tham gia hay không; giá chào; điều kiện nêu trong hồ sơ |
| Output | hồ sơ dự thầu, BOQ dự thầu |
| Downstream | pháp chế (hợp đồng nếu trúng), quản lý dự án (bàn giao BOQ), BGĐ |
| Data produced | BOQ dự thầu, giả định giá, bảng so sánh đối thủ (nội bộ) |
| System touched | bảng tính Excel, kho tài liệu, ERP (đơn giá) |
| Exceptions | hồ sơ mời thầu sửa sau khi đã lập giá; thiếu bản vẽ; đơn giá vật tư lỗi thời |
| Failure consequences | chào giá sai gây lỗ hoặc bị loại; thi công theo BOQ lệch với bản vẽ |

### 3.3 Quản lý dự án

| Thành phần | Nội dung |
|---|---|
| Actor | trưởng dự án (`project_manager`) |
| Trigger | hết ngày làm việc (báo cáo ngày); đầu tuần (kế hoạch); RFI/phát sinh mới |
| Input | báo cáo ngày, tiến độ, đề nghị vật tư, nhân công, RFI, hồ sơ nghiệm thu |
| Process | theo dõi tiến độ và chi phí, duyệt báo cáo ngày, điều phối nhà thầu phụ, làm việc với CĐT/TVGS |
| Decision | duyệt báo cáo ngày và đề nghị vật tư trong hạn mức; điều chỉnh kế hoạch; trình phát sinh |
| Output | kế hoạch tuần, báo cáo tuần/tháng cho CĐT và BGĐ, phê duyệt trong workflow |
| Downstream | BGĐ, CĐT, kế toán, công trường |
| Data produced | kế hoạch, báo cáo tiến độ, biên bản họp, phê duyệt |
| System touched | ERP dự án, bảng tính tiến độ, nhóm chat, kho tài liệu |
| Exceptions | báo cáo ngày thiếu hoặc muộn; các nguồn số liệu lệch; phụ thuộc xác nhận của TVGS |
| Failure consequences | tiến độ báo "ảo"; chậm phát hiện chậm tiến độ; thanh toán và nghiệm thu sai kỳ |

### 3.4 Kỹ thuật

| Thành phần | Nội dung |
|---|---|
| Actor | kỹ sư kỹ thuật/hiện trường (`technical_engineer`) |
| Trigger | bản vẽ mới phát hành; yêu cầu nghiệm thu; RFI; đo đạc định kỳ |
| Input | bản vẽ và thuyết minh, BOQ, tiêu chuẩn kỹ thuật (nêu khái quát), số đo hiện trường |
| Process | kiểm bản vẽ, bóc khối lượng, lập biện pháp thi công, đo đạc, soạn biên bản nghiệm thu, trả lời RFI |
| Decision | chấp thuận biện pháp thi công; xác nhận khối lượng đo; có cần phát sinh hay không |
| Output | biên bản đo, biện pháp thi công, phản hồi RFI, đề nghị nghiệm thu |
| Downstream | công trường, QA/QC, quản lý dự án, kế toán (thanh toán theo khối lượng) |
| Data produced | `measured_quantity`, RFI, phiên bản bản vẽ, biên bản nghiệm thu |
| System touched | kho bản vẽ/tài liệu, bảng tính, ERP |
| Exceptions | bản vẽ lỗi thời đang dùng ở hiện trường; BOQ lệch bản vẽ; CĐT/TVGS phản hồi chậm |
| Failure consequences | thi công theo bản vẽ cũ phải làm lại; khối lượng nghiệm thu sai kéo theo thanh toán sai |

### 3.5 Công trường

| Thành phần | Nội dung |
|---|---|
| Actor | chỉ huy trưởng công trường (`site_commander`), đội trưởng/giám sát hiện trường (`site_supervisor`) |
| Trigger | bắt đầu ca (giao việc), cuối ca (báo cáo), thay đổi thời tiết |
| Input | kế hoạch ngày, vật tư và máy được cấp, nhân công có mặt, thời tiết |
| Process | thi công, ghi nhật ký, báo cáo ngày qua nhóm chat hoặc biểu mẫu, đề nghị vật tư |
| Decision | phân ca; dừng/tiếp tục thi công khi mưa; đề nghị vật tư/máy; báo sự cố |
| Output | báo cáo ngày (tin nhắn/biểu mẫu), nhật ký thi công, ảnh, đề nghị vật tư |
| Downstream | quản lý dự án, kho, máy móc, QA/QC, an toàn |
| Data produced | `reported_progress`, `reported_machine_hours`, `reported_fuel`, `reported_headcount`, thời tiết, ảnh |
| System touched | nhóm chat (ví dụ Zalo), biểu mẫu giấy/bảng tính, ảnh/video |
| Exceptions | tin nhắn viết tắt/lẫn đơn vị/thiếu đơn vị; mất sóng nên báo muộn; một báo cáo gửi vào nhiều nhóm |
| Failure consequences | dữ liệu bẩn đi thẳng vào báo cáo; bất thường không bị phát hiện; trách nhiệm không truy được |

### 3.6 Vật tư (mua sắm)

| Thành phần | Nội dung |
|---|---|
| Actor | nhân viên vật tư (`procurement_officer`) |
| Trigger | đề nghị vật tư đã được duyệt (APPROVED); tồn kho dưới mức an toàn |
| Input | phiếu đề nghị vật tư, báo giá, hợp đồng khung, danh mục nhà cung cấp |
| Process | so sánh báo giá, chọn nhà cung cấp, tạo PO trong ERP, theo dõi giao hàng |
| Decision | chọn nhà cung cấp; đặt hàng; chấp nhận giao thiếu/chậm |
| Output | PO, lịch giao hàng |
| Downstream | kho (nhận hàng), kế toán (hoá đơn), công trường |
| Data produced | PO, báo giá, đánh giá nhà cung cấp |
| System touched | ERP (mua hàng), email, bảng tính |
| Exceptions | nhà cung cấp hết hàng hoặc giao chậm; giá biến động; ERP lỗi khi tạo PO |
| Failure consequences | thiếu vật tư làm dừng thi công; mua trùng; PO không khớp phiếu duyệt |

### 3.7 Kho

| Thành phần | Nội dung |
|---|---|
| Actor | thủ kho (`warehouse_keeper`) |
| Trigger | hàng về kho; đề nghị xuất vật tư; lệnh cấp dầu cho bồn/máy |
| Input | PO, phiếu giao hàng, đề nghị xuất, tồn kho hiện có |
| Process | nhận và kiểm hàng, ghi phiếu nhập kho (PNK) và phiếu xuất kho (PXK), kiểm kê định kỳ |
| Decision | nhận hay trả hàng; xuất theo hạn mức; báo hao hụt |
| Output | PNK, PXK, báo cáo tồn kho |
| Downstream | kế toán (đối chiếu ba bên), vật tư, công trường |
| Data produced | `issued_fuel`, `issued_material`, tồn kho |
| System touched | ERP kho, phiếu giấy, bảng tính |
| Exceptions | phiếu viết tay sửa chữa/bản sửa chưa ký; hao hụt dầu; hai bản phiếu cho một lần cấp |
| Failure consequences | tồn kho sai; xuất âm kho; dầu thất thoát không phát hiện; đối soát kế toán lệch |

### 3.8 Máy móc thiết bị

| Thành phần | Nội dung |
|---|---|
| Actor | trưởng bộ phận thiết bị (`equipment_manager`), kỹ thuật viên (`maintenance_technician`) |
| Trigger | đến hạn bảo dưỡng theo giờ máy; cảm biến báo bất thường; sự cố hỏng máy |
| Input | giờ đồng hồ máy, lịch sử bảo dưỡng, sổ tay thiết bị, nhiên liệu đã cấp |
| Process | điều phối máy, mở/đóng phiếu bảo dưỡng, kiểm tra tiêu hao nhiên liệu so định mức |
| Decision | điều máy sang dự án khác; dừng máy để sửa; thay thế |
| Output | phiếu kiểm tra/bảo dưỡng, bảng điều động máy |
| Downstream | công trường, kế toán (chi phí ca máy), kho (phụ tùng) |
| Data produced | `logged_engine_hours`, `timesheet_machine_hours`, `measured_consumption`, phiếu bảo dưỡng |
| System touched | ERP tài sản, đồng hồ giờ máy/cảm biến, bảng tính |
| Exceptions | đồng hồ hỏng; ghi giờ máy bằng tay sai; hai bản ghi trùng một máy; số đo cảm biến trùng/đến trễ |
| Failure consequences | hỏng máy giữa ca; chi phí ca máy tính sai; bảo dưỡng trễ làm hỏng nặng |

### 3.9 QA/QC

| Thành phần | Nội dung |
|---|---|
| Actor | kỹ sư QA/QC (`qa_qc_engineer`) |
| Trigger | trước và sau khi đổ bê tông; vật liệu nhập về; yêu cầu nghiệm thu |
| Input | yêu cầu kỹ thuật (nêu khái quát), kết quả thí nghiệm, mẫu thử, biên bản kiểm tra |
| Process | lấy mẫu, theo dõi thí nghiệm, đánh giá đạt/không đạt, lập NCR (báo cáo không phù hợp) khi cần |
| Decision | chấp thuận hạng mục; mở/đóng NCR; yêu cầu làm lại |
| Output | biên bản kiểm tra, NCR, kết luận chất lượng |
| Downstream | kỹ thuật, TVGS/CĐT, nghiệm thu, an toàn |
| Data produced | kết quả thí nghiệm, NCR, biên bản kiểm tra |
| System touched | phòng thí nghiệm bên ngoài, kho tài liệu, ERP |
| Exceptions | mẫu không đạt; thiếu mẫu; kết quả thí nghiệm đến muộn |
| Failure consequences | nghiệm thu hạng mục lỗi; phải đập đi làm lại; mất uy tín với CĐT |

### 3.10 An toàn

| Thành phần | Nội dung |
|---|---|
| Actor | cán bộ an toàn (`safety_officer`) |
| Trigger | sự cố hoặc suýt xảy ra (near-miss); kiểm tra định kỳ; cảnh báo thời tiết |
| Input | báo cáo sự cố, kế hoạch ATLĐ, thời tiết, hồ sơ huấn luyện |
| Process | phân loại mức nghiêm trọng, điều tra, đề ra hành động khắc phục, huấn luyện |
| Decision | dừng việc hoặc khoá hiện trường; mức nghiêm trọng cuối; thông báo bên ngoài (khái quát) |
| Output | biên bản sự cố, kế hoạch khắc phục, thông báo |
| Downstream | BGĐ, quản lý dự án, nhân sự, bảo hiểm, cơ quan chức năng (khái quát) |
| Data produced | số sự cố gây thiệt hại, số suýt xảy ra, mức nghiêm trọng, biện pháp |
| System touched | biểu mẫu sự cố, ảnh, ERP an toàn |
| Exceptions | sự cố không được báo; mức bị báo thấp hơn thực tế; thông tin sự cố bị lan sang nhóm không liên quan |
| Failure consequences | tai nạn nghiêm trọng tái diễn; vi phạm nghĩa vụ báo cáo; lộ thông tin nhạy cảm |

### 3.11 Tài chính / Kế toán

| Thành phần | Nội dung |
|---|---|
| Actor | trưởng phòng tài chính (`finance_manager`), kế toán (`accountant`) |
| Trigger | nhận hoá đơn/PXK/PNK; kỳ thanh toán; cuối kỳ đối soát nhiên liệu |
| Input | hoá đơn, PO, PNK, PXK, biên bản nghiệm thu, ngân sách dự án |
| Process | đối chiếu ba bên (PO – nhập kho – hoá đơn), hạch toán, đối soát nhiên liệu, kiểm soát ngân sách, lập hồ sơ thanh toán |
| Decision | chấp nhận hoặc tranh chấp hoá đơn; duyệt chi; giữ lại (retention) theo hợp đồng |
| Output | bút toán, lệnh chi, báo cáo chi phí và dòng tiền, hồ sơ thanh toán gửi CĐT |
| Downstream | BGĐ, ngân hàng, quản lý dự án, CĐT |
| Data produced | `billed_fuel`, `billed_quantity`, chi phí đã ghi nhận, đã thanh toán |
| System touched | ERP kế toán, ngân hàng, bảng tính |
| Exceptions | hoá đơn sai/trùng; chứng từ thiếu; lệch giữa các chỉ số nhiên liệu; lệch múi giờ giao dịch |
| Failure consequences | thanh toán trùng hoặc sai; ghi nhận chi phí sai kỳ; thiếu tiền mặt không được cảnh báo |

### 3.12 Nhân sự

| Thành phần | Nội dung |
|---|---|
| Actor | cán bộ nhân sự (`hr_officer`) |
| Trigger | tuyển dụng, điều động, nghỉ việc; chốt công cuối kỳ |
| Input | hợp đồng lao động, bảng chấm công, quyết định điều động |
| Process | quản lý hồ sơ, chấm công, bảng lương, đào tạo ATLĐ |
| Decision | điều động nhân sự giữa dự án; xử lý yêu cầu về dữ liệu cá nhân |
| Output | bảng công, bảng lương, danh sách nhân sự theo dự án |
| Downstream | kế toán (trả lương), công trường, an toàn |
| Data produced | `attendance_headcount`, số công, dữ liệu lương và hồ sơ cá nhân (nhạy cảm) |
| System touched | hệ thống nhân sự/chấm công, bảng tính |
| Exceptions | nhân sự nghỉ không báo; trùng người giữa hai dự án; yêu cầu xoá dữ liệu cá nhân |
| Failure consequences | lộ dữ liệu lương; trả lương sai; vi phạm yêu cầu bảo vệ dữ liệu cá nhân |

### 3.13 Pháp chế / Hợp đồng

| Thành phần | Nội dung |
|---|---|
| Actor | cán bộ pháp chế (`legal_officer`), nhân viên hồ sơ (`records_clerk`) |
| Trigger | ký hợp đồng/phụ lục; tranh chấp; CĐT yêu cầu hồ sơ |
| Input | hợp đồng, phụ lục, đề nghị phát sinh, hồ sơ nghiệm thu |
| Process | soát điều khoản, quản lý phiên bản hợp đồng, lưu hồ sơ, ghi nhận xác nhận của bên ngoài thay mặt |
| Decision | chấp thuận phụ lục; ý kiến pháp lý về phạt/điều chỉnh giá |
| Output | hợp đồng đã ký, ý kiến pháp lý, hồ sơ lưu |
| Downstream | BGĐ, quản lý dự án, kế toán, kỹ thuật |
| Data produced | hợp đồng và phiên bản, phụ lục, mốc hiệu lực, điều khoản phạt/điều chỉnh |
| System touched | kho tài liệu, ERP hợp đồng |
| Exceptions | phụ lục mâu thuẫn bản gốc; hai bản cùng gắn nhãn hiệu lực; tài liệu chứa văn bản lạ |
| Failure consequences | áp sai điều khoản phạt; dẫn quy định đã hết hiệu lực; mất căn cứ khi tranh chấp |

### 3.14 IT / Data

| Thành phần | Nội dung |
|---|---|
| Actor | quản trị hệ thống (`it_admin`) |
| Trigger | yêu cầu cấp quyền; sự cố hệ thống; nạp tài liệu/danh mục; thay đổi pack |
| Input | yêu cầu quyền, nhật ký hệ thống, tệp tài liệu, cấu hình tenant |
| Process | quản lý tài khoản và vai, tích hợp ERP/chat, sao lưu, nạp tài liệu, kiểm pack |
| Decision | cấp/thu hồi quyền; công bố phiên bản tài liệu; chấp nhận cấu hình |
| Output | hệ thống vận hành, nhật ký audit, báo cáo sự cố hạ tầng |
| Downstream | mọi phòng ban |
| Data produced | nhật ký truy cập, danh mục principal, `pack_version` |
| System touched | ERP, hạ tầng, dịch vụ đọc tài liệu, EAIS |
| Exceptions | dịch vụ sập; lỗi tích hợp; yêu cầu uỷ quyền có thời hạn; yêu cầu xoá dữ liệu cá nhân |
| Failure consequences | rò dữ liệu giữa phạm vi/tenant; mất dữ liệu; cấu hình sai chạy trên dữ liệu thật |

---

## 4. Actor và vai trong EAIS

EAIS tách **bốn loại vai** ([kiến trúc §2.2](../../01-architecture.md)): (1) vai nghiệp vụ, (2) phạm vi dữ liệu, (3) quyền capability, (4) vai của model. *Principal (chủ thể đã xác thực, người hoặc hệ thống) mang vai theo RBAC (kiểm soát truy cập theo vai) và thuộc tính theo ABAC (kiểm soát truy cập theo thuộc tính, ví dụ dự án được gán).* **Chỉ loại 4 nằm trong prompt, và nó không bao giờ dùng để phân quyền.** Bảng dưới là loại 1 và 2; quyền capability ở §4.2.

### 4.1 Vai nghiệp vụ (`principals.roles`) và phạm vi dữ liệu

| Vai (`role`) | Tên | Phòng ban | Phạm vi dữ liệu (thuộc tính ABAC) | Ghi chú |
|---|---|---|---|---|
| `director` | giám đốc | Ban giám đốc | mọi dự án | duyệt vượt hạn mức; xem lương |
| `deputy_director` | phó giám đốc | Ban giám đốc | các dự án được giao | có thể có nhiều dự án |
| `bidding_estimator` | kỹ sư đấu thầu | Đấu thầu | hồ sơ thầu, đơn giá | không xem dữ liệu thi công hiện hành |
| `project_manager` | trưởng dự án | Quản lý dự án | dự án được gán | duyệt báo cáo ngày, đề nghị vật tư theo hạn mức |
| `technical_engineer` | kỹ sư kỹ thuật | Kỹ thuật | dự án được gán | đo đạc, RFI, phát sinh |
| `site_commander` | chỉ huy trưởng công trường | Công trường | một dự án | |
| `site_supervisor` | giám sát/đội trưởng | Công trường | một dự án (và đội) | nguồn của báo cáo ngày qua chat |
| `procurement_officer` | nhân viên vật tư | Vật tư | dự án được gán | tạo PO |
| `warehouse_keeper` | thủ kho | Kho | kho được gán | ghi PNK/PXK |
| `equipment_manager` | trưởng bộ phận thiết bị | Máy móc | các dự án có máy | điều máy |
| `maintenance_technician` | kỹ thuật viên bảo dưỡng | Máy móc | dự án được gán | nhận phiếu bảo dưỡng |
| `qa_qc_engineer` | kỹ sư QA/QC | QA/QC | dự án được gán | không duyệt thanh toán |
| `safety_officer` | cán bộ an toàn | An toàn | các dự án được giao | xem sự cố |
| `finance_manager` | trưởng phòng tài chính | Tài chính | mọi dự án (số liệu tài chính) | duyệt chi |
| `accountant` | kế toán | Kế toán | dự án được gán | đối chiếu ba bên |
| `hr_officer` | cán bộ nhân sự | Nhân sự | nhân sự toàn công ty | xem lương |
| `legal_officer` | cán bộ pháp chế | Pháp chế | hợp đồng toàn công ty | |
| `records_clerk` | nhân viên hồ sơ | Pháp chế/Hợp đồng | dự án được gán | ghi nhận xác nhận của CĐT/TVGS thay mặt |
| `it_admin` | quản trị hệ thống | IT/Data | quản trị nền tảng, không xem nội dung nghiệp vụ mặc định | cấp quyền, nạp tài liệu |

Principal không phải người (cũng đi qua `governance/auth`): `system_erp` (ERP gửi bản ghi), `system_sensor` (cảm biến), `system_scheduler` (timer/lịch), `system_model_proposer` (đại diện bước model đề xuất; **không có quyền tự đổi trạng thái**). `external_tenant_user` là người gọi thuộc **tenant khác**, chỉ dùng để kiểm thử cô lập.

### 4.2 Quyền capability (ví dụ giả lập)

| Capability | Vai thường có | Dùng cho |
|---|---|---|
| `approve_material_request` | `project_manager`, `director` | duyệt đề nghị vật tư |
| `close_material_request` | `project_manager` | đóng phiếu đề nghị |
| `match_invoice` | `accountant` | đối chiếu ba bên |
| `post_inventory_issue` / `post_inventory_receipt` | `warehouse_keeper` | ghi PXK/PNK |
| `create_maintenance_ticket` | `equipment_manager` | mở phiếu bảo dưỡng |
| `assign_task` | `equipment_manager`, `project_manager` | giao human task |
| `report_incident` | `site_supervisor`, `safety_officer` | mở sự cố |
| `create_change_order` | `technical_engineer` | lập phát sinh |
| `request_void` | `warehouse_keeper` | yêu cầu huỷ phiếu đã ghi sổ |
| `request_entity_merge` | `equipment_manager` | đề nghị hợp nhất thực thể |
| `ingest_document` | `it_admin` | nạp tài liệu |

---

## 5. Hệ thống và nguồn dữ liệu

| Hệ thống | Dữ liệu | Loại | Chủ | Đường vào EAIS |
|---|---|---|---|---|
| ERP (mua hàng, kho, tài sản, kế toán, hợp đồng) | PO, PNK/PXK, tồn kho, tài sản, bút toán, hoá đơn | có cấu trúc | tài chính/kho | I2 (bản ghi có cấu trúc) → observation → fact theo chỉ số |
| Bảng tính Excel | BOQ, tiến độ, báo cáo tổng hợp, bảng đối soát | bán cấu trúc | từng phòng ban | I2 (nếu có cột chuẩn) hoặc I1 (tài liệu); nguồn thẩm quyền thấp |
| Nhóm chat (ví dụ Zalo) | báo cáo ngày, thông báo, ảnh | không cấu trúc | công trường | I3 (tin nhắn hiện trường); cổng mơ hồ → hỏi lại |
| Kho tài liệu | SOP, hợp đồng, bản vẽ, sổ tay thiết bị, biên bản | không cấu trúc | pháp chế/kỹ thuật/QA | I1 (đọc cấu trúc → chunk → index, công bố nguyên tử) |
| Hệ thống chấm công/nhân sự | công, người có mặt, lương | có cấu trúc, nhạy cảm | nhân sự | I2 (ABAC chặt) |
| Đồng hồ giờ máy / cảm biến thiết bị | giờ máy, nhiệt độ, mức nhiên liệu | luồng | máy móc | E0–E6 (sự kiện) + observation |
| Phiếu giấy/ảnh chụp | phiếu kho, biên bản viết tay | đa phương thức | kho/công trường | hiện tại: hỏi lại người gửi (cổng thị giác đóng) |
| Phòng thí nghiệm | kết quả thí nghiệm vật liệu | bán cấu trúc | QA/QC | I1 + bản ghi có kiểu |
| Email | trao đổi với CĐT/TVGS/NCC | bán cấu trúc | các phòng | ngoài MVP; đính kèm đi qua I1 |
| EAIS | observation, measurement, fact, derived fact, workflow, audit | mới | IT/Data | chính nền tảng |

---

## 6. Tích hợp ngoài

| Tích hợp | Chiều | Dữ liệu | Nhãn lỗi (`fault`) khi hỏng | Ghi chú |
|---|---|---|---|---|
| ERP | hai chiều | đọc tồn kho/PO/hoá đơn; ghi PO | `third_party:erp:unavailable` | ghi PO là tác dụng phụ cần thử lại có kiểm soát |
| Nhóm chat | vào (và nhắn lại hỏi lại) | tin nhắn, ảnh | `third_party:chat:delivery_failed` | câu hỏi lại phải giữ được thông tin người nhận |
| Cảm biến/đồng hồ | vào | số đo kèm id và thời điểm | `third_party:telemetry:bad_response` | trùng/đến trễ là bình thường |
| Dịch vụ đọc tài liệu | ra rồi vào | tệp → cấu trúc tài liệu | `infra:document-parser`, `third_party:document-parser:bad_response` | không sở hữu quyết định nghiệp vụ |
| Nhà cung cấp model | ra | bằng chứng + yêu cầu diễn đạt | `third_party:model:timeout` | model là một capability, không phải nguồn sự thật |
| Nguồn cảnh báo thời tiết | vào | cảnh báo khu vực | `third_party:weather:unavailable` | kích hoạt human task, không tự quyết hoãn thi công |
| CĐT/TVGS | qua nhân viên hồ sơ | xác nhận, phản hồi RFI | (không tích hợp trực tiếp) | nhập thay bằng human task có chứng từ |
| Ngân hàng/bảo lãnh | ra | lệnh chi | ngoài phạm vi MVP | |

---

## 7. Luật nghiệp vụ (business rules)

Mỗi luật có mã `BR-<LĨNH VỰC>-NN` để ca kiểm thử trong [cases.yaml](../../../tenants/construction/evals/cases.yaml) trỏ tới (`business_rules`). Mọi **ngưỡng và hạn mức** ở đây là tham số **ví dụ giả lập** do pack cấu hình, không phải kết quả đo. Luật được thi hành bằng **mã đã cài** (validator, guard, bộ chuẩn hoá) mà pack chọn theo tên; pack không chứa điều kiện lồng hay vòng lặp (P12).

| Mã | Luật | Lĩnh vực | Cách thi hành |
|---|---|---|---|
| BR-FUEL-01 | Bốn chỉ số nhiên liệu — `reported_fuel`, `issued_fuel`, `measured_consumption`, `billed_fuel` — là bốn chỉ số khác nhau; không trộn, không chọn một làm "số đúng" cho cả bốn | nhiên liệu | `metric_semantics_check`; truy vấn theo chỉ số |
| BR-FUEL-02 | Câu chung chung "bao nhiêu dầu" (không nêu chỉ số) mặc định là `issued_fuel` (dầu **kho xuất**); "tiêu thụ/máy dùng" mặc định là `measured_consumption`, thiếu thì dùng `issued_fuel` và **nói rõ nhãn**; câu xác nhận một số ("4k đúng không?") khi chưa rõ chỉ số và các nguồn đang lệch thì **hỏi lại**, không xác nhận | nhiên liệu | mẫu câu của tenant gắn nhãn chỉ số + `CLARIFICATION` |
| BR-FUEL-03 | Chênh lệch giữa các chỉ số nhiên liệu vượt ngưỡng (ví dụ ±1%/ngày) thì gắn cờ và đưa `review_queue` | nhiên liệu | derived fact + ngưỡng của pack |
| BR-FUEL-04 | Mọi lần xuất dầu có số phiếu xuất kho; `issued_fuel` gắn số phiếu | nhiên liệu | schema `fuel_issue` bắt buộc `slip_no` |
| BR-FUEL-05 | Định mức nhiên liệu lấy từ SOP đang hiệu lực, nhân giờ máy; dung sai (ví dụ +15%) | nhiên liệu | `CALCULATION` + `citation_version_effective` |
| BR-FUEL-06 | Lượng dầu vượt cỡ so với định mức × giờ máy hoặc dung tích bình chứa là **vô lý về cỡ** → hỏi lại (cấp vào bồn hay máy dùng) | nhiên liệu | `cross_field_range` → `CLARIFICATION` |
| BR-UNIT-01 | `k` sau số ở khe lượng/tiền nghĩa là nghìn; vật tư rời (xi măng…) thiếu đơn vị thì hỏi lại, không đoán | đơn vị | bộ chuẩn hoá + cổng mơ hồ |
| BR-UNIT-02 | "khối" gắn chất liệu: chất lỏng → m³ = 1.000 L; bê tông/đất → m³ thể tích công trình | đơn vị | `unit_material_consistency`, bảng quy đổi của pack |
| BR-UNIT-03 | Dấu chấm đứng trước đúng ba chữ số là phân cách nghìn; dấu phẩy là thập phân; luôn kiểm miền giá trị | đơn vị | bộ chuẩn hoá số + `values_in_domain` |
| BR-UNIT-04 | Số bằng chữ tiếng Việt được phân tích chính xác (bốn ngàn → 4000; ba trăm năm chục → 350) | đơn vị | bộ phân tích số chữ |
| BR-UNIT-05 | 1 km = 1.000 m; "KL" trong báo cáo tiến độ là khối lượng thi công (không phải khối lượng vật lý) | đơn vị | lexicon theo phạm vi + bảng quy đổi |
| BR-UNIT-06 | Mã phiếu được chuẩn hoá (chữ hoa, gạch ngang) trước khi gọi tool | đơn vị | mẫu mã của pack |
| BR-UNIT-07 | Thiếu đơn vị mà độ tin cậy dưới ngưỡng thì hỏi xác nhận, chưa ghi fact | đơn vị | ngưỡng độ tin cậy + `CLARIFICATION` |
| BR-EST-01 | Từ "khoảng/chừng/tầm" làm measurement mang cờ `estimate`; không vào tổng chính xác | ước lượng | `estimate_flag_preserved` |
| BR-ENT-01 | Mã máy là duy nhất toàn công ty; "máy NN" phân giải theo dự án của kênh/ngữ cảnh; không duy nhất thì hỏi lại kèm ứng viên | thực thể | `data/entityresolution` + ngưỡng |
| BR-ENT-02 | Viết tắt theo phạm vi: `DA` = dầu trong nhóm báo cáo nhiên liệu, = dự án ở cấp quản lý; ngữ cảnh không đủ thì hỏi lại | thực thể | lexicon theo phạm vi |
| BR-ENT-03 | Lexicon song ngữ Việt/Anh ánh xạ về cùng chỉ số/thực thể chuẩn | thực thể | lexicon của pack |
| BR-ENT-04 | Hợp nhất thực thể là thao tác quản trị cần duyệt; không gộp/xoá tự động; fact cũ được giữ và gắn lại theo bí danh | thực thể | human task + audit |
| BR-LAB-01 | `NC` không kèm "công" là số người có mặt (`reported_headcount`); số công (man-day) là chỉ số khác | nhân công | lexicon + chỉ số tách biệt |
| BR-LAB-02 | Số người có mặt chính thức theo chấm công (`attendance_headcount`); báo của đội trưởng là `reported_headcount`; luôn nêu phạm vi (ABC, nhà thầu phụ) | nhân công | truy vấn theo bên sử dụng lao động |
| BR-MCH-01 | Giờ máy có ba chỉ số: `reported_machine_hours`, `logged_engine_hours` (đồng hồ), `timesheet_machine_hours`; "giờ máy vận hành" mặc định theo đồng hồ | máy móc | `metric_semantics_check` |
| BR-MNT-02 | Phiếu bảo dưỡng/kiểm tra có người nhận trong hạn; chỉ người được gán chuyển trạng thái; mỗi máy chỉ một phiếu mở cho cùng nguyên nhân | máy móc | guard + khoá khử trùng sự kiện |
| BR-PRG-01 | Khối lượng thi công có bốn chỉ số: `reported_progress`, `measured_quantity`, `accepted_quantity`, `billed_quantity`; "đã nghiệm thu" là `accepted_quantity` | tiến độ | `metric_semantics_check` |
| BR-PRG-02 | "Làm được" mặc định là `measured_quantity` nếu có, không thì `reported_progress`; luôn nêu các chỉ số còn lại | tiến độ | mẫu câu của tenant |
| BR-RPT-01 | Báo cáo ngày cần đủ năm khe: tiến độ, giờ máy, nhiên liệu, nhân công, thời tiết; không nộp khi thiếu hoặc còn câu hỏi lại treo | báo cáo | guard `required_fields_complete`, `all_clarifications_resolved` |
| BR-RPT-02 | Mỗi dự án một báo cáo mỗi ngày làm việc; ca đêm tính vào ngày bắt đầu ca | báo cáo | lịch ca của pack + kỳ của fact |
| BR-RPT-03 | Báo cáo ngày phải so với ngày trước và gắn cờ bất thường | báo cáo | bước bắt buộc của skill |
| BR-RPT-04 | Hạn nộp báo cáo ngày (ví dụ 18:00); quá hạn thì nhắc và leo thang đúng một lần | báo cáo | timer của workflow + human task |
| BR-RPT-05 | Báo cáo tổng hợp quản lý liệt kê đủ các chiều (tiến độ, vật tư, máy, nhân công, an toàn); chiều thiếu dữ liệu được nói rõ, không kết luận "không có vấn đề" | báo cáo | skill `management-briefing` |
| BR-SOT-01 | Ma trận thẩm quyền chỉ áp trong **cùng một chỉ số ngữ nghĩa**; giữa hai chỉ số là derived fact ghi công thức | nguồn sự thật | `data/sourceoftruth` |
| BR-SOT-02 | Authority ≠ recency: bản sửa chưa ký không thay bản gốc đã ký | nguồn sự thật | hạng thẩm quyền theo chữ ký |
| BR-SOT-03 | Cùng chỉ số, cùng hạng thẩm quyền mà lệch (hoặc hai tài liệu thiết kế lệch) thì không tự chọn, chuyển `review_queue` | nguồn sự thật | `fact_conflicts` + human task |
| BR-SOT-04 | Cùng một thông tin gửi nhiều lần (khác message id) được khử trùng theo (dự án, ngày, chỉ số, người báo): giữ mọi observation, chỉ một fact | nguồn sự thật | `duplicate_key_check` |
| BR-SOT-05 | Fact bị rút lại làm derived fact phụ thuộc bị đánh dấu lỗi thời và chuyển người tính lại | nguồn sự thật | cờ lỗi thời + human task (lùi an toàn; xem F-005) |
| BR-DAT-01 | Dữ liệu quá độ tươi quy định của chỉ số phải mang cờ "dữ liệu cũ"; khi hạ tầng lỗi không dùng cache hết hạn làm số hiện hành | dữ liệu | `freshness_check` |
| BR-DOC-01 | Chỉ dùng phiên bản tài liệu đang hiệu lực tại ngày nghiệp vụ; hai bản cùng hiệu lực là xung đột, chuyển người quyết | tài liệu | metadata hiệu lực + `citation_version_effective` |
| BR-DOC-02 | Trích dẫn có mã tài liệu, phiên bản, vị trí; mọi ý nêu ra phải có bằng chứng | tài liệu | `citation_valid`, `groundedness` |
| BR-DOC-03 | Phiên bản tài liệu chỉ được công bố khi qua cổng chất lượng đọc (bảng đủ ô, số đủ); không đạt thì không công bố, giữ phiên bản cũ | tài liệu | cổng chất lượng + công bố nguyên tử |
| BR-DOC-04 | Thứ tự ưu tiên tài liệu theo pack: quy định nội bộ hiệu lực > sổ tay nhà sản xuất khi kiểm soát chi phí nội bộ; hai tài liệu thiết kế lệch (BOQ, bản vẽ) thì báo lệch | tài liệu | thứ tự trong pack + cờ `Conflict` |
| BR-WH-01 | Không xuất âm kho | kho | guard `stock_sufficient` |
| BR-WH-02 | Phiếu nhập/xuất idempotent theo mã phiếu | kho | khoá idempotency |
| BR-WH-03 | Huỷ phiếu đã ghi sổ cần duyệt và phải nêu hệ quả lên số liệu dẫn xuất | kho | human task + danh sách derived fact phụ thuộc |
| BR-PROC-01 | Người lập phiếu đề nghị vật tư không tự duyệt; người duyệt phải có vai duyệt | mua sắm | guard `not_same_principal`, `has_approver_role` |
| BR-PROC-02 | Hạn mức duyệt theo vai (ví dụ: ≤ 50 triệu trưởng dự án; ≤ 300 triệu thêm kế toán; trên đó thêm giám đốc) | mua sắm | guard `amount_within_limit` |
| BR-PROC-03 | Duyệt phải có báo giá/đính kèm | mua sắm | guard `attachments_present` |
| BR-PROC-04 | Các bước duyệt theo thứ tự trưởng dự án → kế toán → giám đốc; chỉ APPROVED khi đủ mọi task | mua sắm | guard `all_review_tasks_done` |
| BR-PROC-05 | Không tạo phiếu đề nghị trùng cho cùng vật tư/kho khi đã có phiếu đang mở | mua sắm | trigger so khớp phiếu mở |
| BR-PROC-06 | Sau APPROVED phải có PO trong ERP; nếu tạo PO lỗi thì tạo human task thủ công, không tạo hai lần | mua sắm | human task (lùi an toàn; xem F-048) |
| BR-PROC-07 | Lệnh hàng loạt không thực thi khi chưa có kết quả từng phần; chuyển người duyệt | mua sắm | `HUMAN_ESCALATION` (lùi an toàn; xem F-037) |
| BR-PAY-01 | Duyệt thanh toán cần khớp ba bên (PO – nhập kho – hoá đơn) và đúng vai duyệt | thanh toán | guard `three_way_match_ok`, `has_approver_role` |
| BR-PAY-02 | Hoá đơn nêu rõ giá trị trước thuế và tổng thanh toán; không trả một số mà không nói gồm hay chưa gồm thuế | thanh toán | mẫu câu của tenant |
| BR-ACC-01 | Nghiệm thu: đo → TVGS xác nhận → CĐT ký; chỉ chốt khi đủ cả hai bên; bên ngoài do nhân viên hồ sơ ghi nhận thay kèm chứng từ | nghiệm thu | guard `all_review_tasks_done` |
| BR-CHG-01 | Phát sinh khối lượng đào đất đá phải ghi cơ sở đo (nguyên khối hay tơi xốp) | phát sinh | tham số bắt buộc của lệnh |
| BR-SAF-01 | Sự cố nghiêm trọng: khoá hiện trường và thông báo trước khi sang điều tra | an toàn | guard `severity_notified` |
| BR-SAF-02 | Mức nghiêm trọng tối thiểu theo loại sự cố (ví dụ ngã trên cao ≥ mức 2); mức cuối = lớn hơn giữa đề xuất của model và mức tối thiểu của luật, do engine quyết | an toàn | bảng mức tối thiểu của pack |
| BR-SAF-03 | Thống kê sự cố tách "gây thiệt hại" và "suýt xảy ra (near-miss)" | an toàn | truy vấn theo loại |
| BR-TIME-01 | Mọi mốc thời gian chuẩn hoá về múi giờ của tenant (Asia/Ho_Chi_Minh) trước khi gán kỳ/ngày | thời gian | bộ chuẩn hoá ngày/giờ |
| BR-EVT-01 | Mỗi sự kiện xử lý đúng một lần theo khoá khử trùng; số đo cũ hơn số đã xử lý không kích hoạt lại | sự kiện | khoá khử trùng + `ObservedAt` |
| BR-SCP-01 | Phạm vi hỗ trợ là nghiệp vụ ABC; câu ngoài phạm vi bị từ chối bằng mẫu câu, không gọi model | phạm vi | route `REJECT` |
| BR-SCP-02 | Việc ngôn ngữ thuần (viết lại, dịch) đi `DIRECT_LLM` và không được thêm con số mới | phạm vi | route `DIRECT_LLM` + `number_in_evidence` trên đầu vào |
| BR-SEC-01 | Tenant lấy từ danh tính (khoá API → principal), không từ body | bảo mật | `governance/auth` |
| BR-SEC-02 | Phạm vi dữ liệu theo dự án (ABAC): chỉ dự án được gán; lọc trước khi tổng hợp; khoá cache có phạm vi quyền | bảo mật | `permission_scope_check` |
| BR-SEC-03 | Lương/đơn giá nhân công chỉ vai nhân sự và giám đốc xem được; lọc trước khi vào context | bảo mật | ACL theo tập dữ liệu |
| BR-SEC-04 | Lệnh ghi cần quyền capability tương ứng; mọi lệnh, kể cả bị từ chối, có audit | bảo mật | `governance/permissions` + `governance/audit` |
| BR-SEC-05 | Uỷ quyền duyệt là quyền có thời hạn; không cấp vai vĩnh viễn thay uỷ quyền | bảo mật | lùi an toàn: quản trị viên xử lý (xem F-052) |
| BR-SEC-06 | Dữ liệu cá nhân trong nơi chỉ-ghi-thêm không xoá tự động; yêu cầu của chủ thể chuyển người phụ trách dữ liệu | bảo mật | lùi an toàn: human task (xem F-006) |
| BR-SEC-07 | Văn bản trong tài liệu là dữ liệu, không phải lệnh; route đọc không có tool ghi | bảo mật | `no_write_in_read_route` |
| BR-PCK-01 | Pack chỉ mô tả chính sách và so sánh một bước; luật phức tạp dùng chỉ số dẫn xuất do mã cài | pack | linter pack lúc khởi động |

---

## 8. Tuân thủ (nêu khái quát)

> Tài liệu này **không trích số hiệu văn bản pháp luật nào**. Dưới đây là các **nhóm nghĩa vụ** doanh nghiệp xây dựng thường phải đáp ứng theo pháp luật hiện hành và theo hợp đồng; khi triển khai thật, pháp chế của doanh nghiệp đối chiếu và cấu hình vào pack/chính sách. EAIS hỗ trợ ghi nhận, đối chiếu và nhắc việc; **không thay con người ký, chịu trách nhiệm hay nộp hồ sơ**.

| Nhóm | Yêu cầu khái quát | Phần của EAIS liên quan | Điều EAIS không được làm |
|---|---|---|---|
| Nhật ký thi công | ghi chép hằng ngày về công việc, vật tư, máy, nhân công, thời tiết, sự cố; có người chịu trách nhiệm ký | báo cáo ngày (`daily-site-report`), observation giữ nguyên văn | coi báo cáo tự động là nhật ký chính thức khi chưa có người ký |
| Nghiệm thu | nghiệm thu từng giai đoạn về chất lượng và khối lượng, có xác nhận của các bên theo hợp đồng | workflow `quantity-acceptance`, `accepted_quantity` | chốt nghiệm thu khi thiếu một bên xác nhận |
| An toàn lao động (ATLĐ) | huấn luyện, trang bị, báo cáo và điều tra sự cố, thông báo khi sự cố nghiêm trọng | workflow `safety-incident`, guard `severity_notified` | hạ mức sự cố bằng suy diễn của model |
| Quản lý chất lượng | thí nghiệm vật liệu, lập và đóng báo cáo không phù hợp (NCR) | workflow `qaqc-inspection`, guard `no_open_ncr` | đóng NCR khi còn tồn tại không phù hợp |
| Hợp đồng và thanh toán | thanh toán theo khối lượng đã nghiệm thu, giữ lại theo hợp đồng, chứng từ hợp lệ | workflow `payment-certification`, ba bên PO – nhập kho – hoá đơn | duyệt thanh toán khi chưa khớp ba bên |
| Quản lý tài liệu | dùng đúng phiên bản bản vẽ/quy định đang hiệu lực | workflow `document-revision`, metadata hiệu lực | trích dẫn bản đã hết hiệu lực làm căn cứ |
| Bảo vệ dữ liệu cá nhân | hạn chế truy cập, đáp ứng yêu cầu hợp lệ của chủ thể dữ liệu | ABAC, lọc trước context, ca hỏng `CON-F-006` | xoá tự động trong nơi chỉ-ghi-thêm hoặc báo "đã xoá" khi chưa xử lý |
| Lưu trữ hồ sơ | giữ hồ sơ theo thời hạn của hợp đồng/quy định | `documents`, `audit_log` | huỷ hồ sơ trước thời hạn lưu |

---

## 9. Rủi ro

| Mã | Rủi ro | Nguồn | Hệ quả | Giảm thiểu trong EAIS | Ca hỏng |
|---|---|---|---|---|---|
| R-01 | Dữ liệu hiện trường bẩn (viết tắt, lẫn đơn vị, thiếu đơn vị) | nhóm chat, phiếu viết tay | sai số liệu báo cáo | chuẩn hoá lúc nạp + cổng mơ hồ + hỏi lại | `CON-F-010` … `CON-F-014` |
| R-02 | Nhiều nguồn nói khác nhau về cùng một việc | ERP, bảng tính, chat, phiếu kho | tin nhầm một số | chỉ số ngữ nghĩa tách biệt, thẩm quyền trong cùng chỉ số, derived fact | `CON-F-002`, `CON-F-063` |
| R-03 | Tài liệu nhiều phiên bản | SOP, hợp đồng, bản vẽ | dùng quy định đã hết hiệu lực | metadata hiệu lực, công bố nguyên tử, trích dẫn có phiên bản | `CON-F-023`, `CON-F-065` |
| R-04 | Hành động ghi sai hoặc trái quyền | duyệt vật tư, thanh toán | tổn thất tài chính, gian lận | workflow engine sở hữu trạng thái, guard, tách nhiệm vụ, audit | `CON-F-041`, `CON-F-042`, `CON-F-051` |
| R-05 | Lộ dữ liệu nhạy cảm (lương, dự án khác) | context, cache | vi phạm bảo mật | ABAC lọc trước context, khoá cache có phạm vi | `CON-F-027`, `CON-F-053`, `CON-F-058` |
| R-06 | Rò dữ liệu giữa tenant | chỉ mục, danh tính | thảm hoạ uy tín | tenant từ danh tính, RLS, namespace chỉ mục | `CON-F-054` … `CON-F-056` |
| R-07 | Hạ tầng/ERP/model lỗi | tích hợp, mạng | không phục vụ hoặc phục vụ sai | nhãn `fault`, hành động lùi, thử lại có kiểm soát | `CON-F-031`, `CON-F-048`, `CON-F-067` |
| R-08 | Thất thoát nhiên liệu, vật tư | kho, công trường | chi phí tăng | đối soát bốn chỉ số, derived fact, cờ vượt ngưỡng | `CON-F-005`, `CON-F-062` |
| R-09 | Sự cố an toàn bị báo thấp/không báo | công trường | tai nạn tái diễn, vi phạm nghĩa vụ | mức tối thiểu theo luật, guard thông báo, human task | `CON-F-033` |
| R-10 | Model bịa số, làm theo chỉ thị trong tài liệu | model, RAG | số sai tới giám đốc, hành động ngoài ý muốn | `number_in_evidence`, route đọc không có tool ghi | `CON-F-030`, `CON-F-032` |
| R-11 | Việc quá hạn không ai biết | báo cáo ngày, RFI | chậm tiến độ, quá hạn phản hồi | timer workflow, leo thang một lần | `CON-F-043`, `CON-F-045` |
| R-12 | Pack thành "ngôn ngữ lập trình bằng YAML" | cấu hình tenant | khó kiểm thử, sai khó thấy | linter pack, chỉ số dẫn xuất do mã cài | `CON-F-046` |
| R-13 | Chi phí model vô ích | route sai | tốn tiền | route không cần diễn đạt thì không gọi model | `CON-F-022` |

---

## 10. KPI

Mọi **mục tiêu** dưới đây do doanh nghiệp đặt khi triển khai; hiện **chưa đo** (CONVENTIONS §6). Cột "Nguồn" cho biết chỉ số lấy từ đâu để sau này đo có kiểm chứng.

### 10.1 KPI nghiệp vụ

| KPI | Định nghĩa | Nguồn dữ liệu | Mục tiêu |
|---|---|---|---|
| Báo cáo ngày đúng hạn | tỉ lệ (dự án, ngày) có báo cáo ở trạng thái SUBMITTED trước hạn | `workflow_instances` của `daily-site-report` | chưa đo |
| Độ trễ báo cáo | thời gian từ tin nhắn đầu tiên đến SUBMITTED | observation + `workflow_transitions` | chưa đo |
| Chênh lệch nhiên liệu | `issued_fuel` so `measured_consumption` theo dự án/kỳ (derived fact) | fact nhiên liệu | chưa đo |
| Tiêu hao so định mức | `measured_consumption` so định mức × giờ máy theo máy | fact + SOP đã trích | chưa đo |
| Tỉ lệ máy nhàn rỗi | giờ máy không vận hành trên giờ sẵn sàng | `logged_engine_hours`, điều động | chưa đo |
| Tiến độ so kế hoạch | `measured_quantity` lũy kế so kế hoạch | fact tiến độ + kế hoạch | chưa đo |
| Thời gian duyệt đề nghị vật tư | từ SUBMITTED đến APPROVED | `workflow_transitions` | chưa đo |
| Hoá đơn khớp ba bên lần đầu | tỉ lệ hoá đơn qua MATCHING không sang DISPUTED | workflow `payment-certification` | chưa đo |
| NCR đóng đúng hạn | tỉ lệ NCR đóng trong hạn | workflow `qaqc-inspection` | chưa đo |
| Tần suất sự cố | số sự cố gây thiệt hại trên số giờ công (kèm số suýt xảy ra) | `incident_record`, chấm công | chưa đo |
| RFI quá hạn | số RFI ở ESCALATED | workflow `rfi-handling` | chưa đo |

### 10.2 KPI hệ thống AI (theo thành phần, rồi đến đầu–cuối)

| Nhóm | Chỉ số | Cách đo | Mục tiêu |
|---|---|---|---|
| Thành phần | normalization accuracy, entity resolution accuracy, intent accuracy, router accuracy | bộ ca [cases.yaml](../../../tenants/construction/evals/cases.yaml) theo `category` | chưa đo |
| Thành phần | retrieval Recall@K, citation correctness, groundedness | ca RAG/HYBRID | chưa đo |
| Thành phần | tool success, workflow success, skill completion | ca COMMAND/EVENT/WORKFLOW | chưa đo |
| Thành phần | validator catch rate, false block rate | ca dirty/adversarial | chưa đo |
| Thành phần | permission violations, tenant leakage | ca permission (mục tiêu: không có vi phạm) | chưa đo |
| Thành phần | latency, cost mỗi lượt | `observability/traces`, `observability/cost` | chưa đo |
| Đầu–cuối | business task success rate, critical error rate, human escalation rate, false confidence rate | toàn bộ bộ ca | chưa đo |
| Đầu–cuối | time saved, cost per successful task | so với quy trình thủ công hiện tại (cần khảo sát) | chưa đo |

Lệnh chạy bộ ca (khi runner có): `eais eval --tenant abc-infra` — **chưa có**; ghi nhận khi `evals/` được hiện thực.
