# 03 — Bản đồ dữ liệu (ABC Infrastructure)

> Tenant `abc-infra` (mã `CON`). Mọi giá trị, mã phiếu, ngưỡng và **con số** trong tài liệu là **ví dụ giả lập**, không phải kết quả đo (CONVENTIONS §6, §7).
> Liên quan: [01-business-map](01-business-map.md) · [04-workflow-map](04-workflow-map.md) · [05-problem-catalog](05-problem-catalog.md) · [07-failure-catalog](07-failure-catalog.md) · [`cases.yaml`](../../../tenants/construction/evals/cases.yaml).

## 0. Bốn tầng dữ liệu và luật nền

Kiến trúc ([§5.0](../../01-architecture.md)) tách dữ liệu sự thật thành bốn tầng:

```text
OBSERVATION   điều một nguồn đã nói/ghi: nguyên văn + nguồn + thời điểm
   ▼ interpretation (chuẩn hoá, nhận dạng thực thể)
MEASUREMENT   giá trị có kiểu + đơn vị + thực thể + kỳ, vẫn gắn observation gốc
   ▼ business semantics (chỉ số ngữ nghĩa do pack định nghĩa)
FACT          reported_* · issued_* · measured_* · accepted_* · billed_* … (các chỉ số KHÁC NHAU)
   ▼ authority / reconciliation (chỉ giữa các fact CÙNG chỉ số, cùng thực thể, cùng kỳ)
DERIVED FACT  chênh lệch, tổng, cờ bất thường — luôn ghi công thức + fact đầu vào
```

Ba luật:

1. **Không gộp nguồn thành một "số dầu".** "Đội trưởng báo 4k", "kho xuất 3.850 L", "đồng hồ đo 3.782 L", "hoá đơn 4.100 L" là **bốn chỉ số ngữ nghĩa khác nhau**, không phải bốn phiên bản của một fact.
2. **Ma trận thẩm quyền chỉ áp giữa các nguồn CÙNG một chỉ số.** Chênh lệch giữa hai chỉ số khác nhau là **derived fact** đáng điều tra (có công thức), không phải xung đột cần chọn bên.
3. **Authority ≠ recency ≠ similarity ≠ số lần lặp.** Nguồn mới hơn, giống câu hỏi hơn, hoặc được lặp nhiều lần hơn không tự động đúng hơn.

---

## 1. Danh mục chỉ số ngữ nghĩa

Mỗi chỉ số do `metrics.yaml` của pack định nghĩa (kiểu, đơn vị chuẩn, miền giá trị). Dưới đây là sáu họ chỉ số của ABC.

| Họ | Chỉ số ngữ nghĩa | Nghĩa | Đơn vị chuẩn | Nguồn điển hình |
|---|---|---|---|---|
| Nhiên liệu | `reported_fuel` | lượng dầu **người báo** (đội trưởng/giám sát) | L | nhóm chat, báo cáo ngày |
| | `issued_fuel` | lượng dầu **kho xuất** theo phiếu | L | PXK, giao dịch ERP kho |
| | `measured_consumption` | lượng dầu **máy tiêu thụ đo được** | L | đồng hồ/cảm biến, nhật ký cấp cho máy |
| | `billed_fuel` | lượng dầu **trên hoá đơn** của nhà cung cấp | L | hoá đơn, bản ghi hoá đơn ERP |
| Khối lượng thi công | `reported_progress` | khối lượng **người báo** | m, m², m³ theo hạng mục | chat, báo cáo ngày |
| | `measured_quantity` | khối lượng **đo đạc** | như trên | biên bản đo |
| | `accepted_quantity` | khối lượng **đã nghiệm thu** (đủ xác nhận) | như trên | biên bản nghiệm thu |
| | `billed_quantity` | khối lượng **đưa vào hồ sơ thanh toán** | như trên | hồ sơ thanh toán |
| Giờ máy | `reported_machine_hours` | giờ máy người báo | giờ | chat, báo cáo ngày |
| | `logged_engine_hours` | giờ máy theo **đồng hồ/telematics** | giờ | đồng hồ giờ máy |
| | `timesheet_machine_hours` | giờ máy theo **bảng chấm công ca máy** | giờ | bảng chấm công ca máy |
| Nhân công | `reported_headcount` | số người **người báo** | người | chat, báo cáo ngày |
| | `attendance_headcount` | số người **có mặt theo chấm công** | người | hệ thống chấm công |
| | `paid_manhours` | giờ công **tính lương** | giờ công | bảng lương đã chốt |
| Vật tư | `requested_material` | số lượng **đề nghị** | theo vật tư | phiếu đề nghị |
| | `issued_material` | số lượng **kho xuất** | theo vật tư | PXK vật tư, ERP |
| | `billed_material` | số lượng **trên hoá đơn** | theo vật tư | hoá đơn |
| Chi phí | `committed_cost` | giá trị **cam kết** (PO) | đồng | PO trong ERP |
| | `invoiced_cost` | giá trị **hoá đơn** | đồng | hoá đơn |
| | `paid_cost` | giá trị **đã thanh toán** | đồng | lệnh chi/ngân hàng, ERP |

Quy ước câu hỏi chung chung (pack): câu "bao nhiêu dầu" **không nêu chỉ số** mặc định là `issued_fuel` (dầu **kho xuất**); "tiêu thụ/máy dùng" mặc định là `measured_consumption`, thiếu thì dùng `issued_fuel` **kèm nhãn**; "làm được bao nhiêu mét" mặc định là `measured_quantity`; "đã nghiệm thu" là `accepted_quantity`. Câu trả lời **luôn** nêu chỉ số đã dùng. Xem BR-FUEL-02, BR-PRG-01/02 ở [01 §7](01-business-map.md).

---

## 2. Source Authority Matrix (ma trận thẩm quyền nguồn)

**Chỉ áp giữa các nguồn của CÙNG một chỉ số ngữ nghĩa** (cùng thực thể, cùng kỳ). Hạng nhỏ = mạnh hơn; **cùng hạng thì không tự chọn**. Hạng mạnh hơn không dựa vào độ mới, độ giống câu hỏi hay số lần lặp. Mọi ngưỡng là ví dụ giả lập.

| Chỉ số ngữ nghĩa | Hạng 1 | Hạng 2 | Hạng 3 | Ngưỡng xung đột | Khi lệch |
|---|---|---|---|---|---|
| `reported_fuel` | báo cáo ngày đã nộp | tin nhắn giám sát/đội trưởng | tin nhắn người khác, bảng Excel chép lại | khác giá trị cùng người, cùng ngày | hỏi lại người báo; trùng giá trị thì gộp (xác nhận chéo) |
| `issued_fuel` | **PXK có chữ ký thủ kho** và **giao dịch ERP kho** (cùng hạng 1) | PXK bản sửa chưa có chữ ký thủ kho trưởng; bảng Excel kho | tin nhắn thủ kho | 0,5% | hạng 1 lệch nhau → `review_queue`; bản sửa chưa ký **không** thay bản gốc |
| `measured_consumption` | đồng hồ/cảm biến đã hiệu chuẩn | nhật ký cấp cho máy có xác nhận hai bên | báo của thợ máy qua chat | 2% | cờ hiệu chuẩn; người thiết bị quyết |
| `billed_fuel` | hoá đơn hợp lệ của nhà cung cấp | bản ghi hoá đơn trong ERP | bảng đối chiếu Excel | 0% (số hoá đơn là số hoá đơn) | hoá đơn trùng → `duplicate_key_check` |
| `reported_progress` | báo cáo ngày đã nộp | tin nhắn giám sát | bảng tổng hợp Excel | khác giá trị cùng người | hỏi lại |
| `measured_quantity` | biên bản đo có chữ ký kỹ sư | số liệu đo máy toàn đạc nạp trực tiếp | bảng tính của đội đo | 1% | kỹ sư trưởng quyết |
| `accepted_quantity` | biên bản nghiệm thu đủ xác nhận TVGS và CĐT | bản ghi nghiệm thu trong ERP | xác nhận chưa chính thức (email) | 0% | không dùng hạng 3 cho thanh toán |
| `billed_quantity` | hồ sơ thanh toán được CĐT chấp nhận | bản nháp hồ sơ thanh toán | — | 0% | kế toán quyết |
| `reported_machine_hours` | báo cáo ngày đã nộp | tin nhắn giám sát | Excel | khác giá trị cùng người | hỏi lại |
| `logged_engine_hours` | đồng hồ giờ máy/telematics | ảnh đồng hồ có xác nhận | ghi tay | 2% | người thiết bị quyết |
| `timesheet_machine_hours` | bảng chấm công ca máy đã ký | bản nháp | — | 0% | người thiết bị + kế toán |
| `reported_headcount` | báo cáo ngày đã nộp | tin nhắn giám sát | — | khác giá trị cùng người | hỏi lại |
| `attendance_headcount` | hệ thống chấm công | bảng chấm công giấy nhập lại | — | 0% | nhân sự quyết |
| `paid_manhours` | bảng lương đã chốt | bản nháp bảng lương | — | 0% | nhân sự + kế toán |
| `issued_material` | **PXK có chữ ký** và **giao dịch ERP** (cùng hạng 1) | bảng Excel kho | tin nhắn thủ kho | 0,5% | cùng hạng lệch → `review_queue` |
| `committed_cost` | PO trong ERP | báo giá | — | 0% | vật tư quyết |
| `invoiced_cost` | hoá đơn hợp lệ | bản ghi ERP | — | 0% | kế toán quyết |
| `paid_cost` | sao kê/lệnh chi ngân hàng | ghi nhận thanh toán trong ERP | — | 0% | tài chính quyết |

Chính sách chung: **Version Policy** — cùng chỉ số, cùng thực thể, cùng kỳ: bản đã ký thay bản chưa ký; bản mới chỉ thay bản cũ khi **cùng hạng thẩm quyền trở lên** và có xác nhận người có quyền. **Conflict Policy** — lệch vượt ngưỡng → `fact_conflicts` + `review_queue`, giá trị hiện hành tạm thời là nguồn hạng cao nhất. **Audit Trail** — mọi quyết định hoà giải ghi người, thời điểm, lý do vào `audit_log`.

---

## 3. Derived fact (fact dẫn xuất) và công thức

Chênh lệch **giữa** hai chỉ số khác nhau là derived fact. Mỗi derived fact ghi công thức và danh sách fact đầu vào; khi một fact đầu vào bị rút lại thì derived fact bị đánh dấu lỗi thời (hành vi lùi an toàn, xem F-005 — chưa có primitive lan truyền tự động).

| Derived fact | Công thức | Fact đầu vào | Ý nghĩa/ngưỡng (ví dụ) |
|---|---|---|---|
| `issued_minus_reported_fuel` | `issued_fuel − reported_fuel` | hai chỉ số cùng dự án-ngày | báo lệch với kho xuất |
| `issued_minus_measured_fuel` | `issued_fuel − measured_consumption` | cùng dự án-ngày | tồn bồn tăng hoặc hao hụt; vượt 1% → cờ |
| `billed_minus_issued_fuel` | `billed_fuel − issued_fuel` | cùng kỳ | hoá đơn lệch kho xuất; vượt 1% → `fuel-reconciliation` |
| `fuel_norm_variance` | `measured_consumption − định_mức_L_mỗi_giờ × logged_engine_hours` | tiêu thụ, giờ máy, định mức SOP hiệu lực | vượt dung sai +15% → vượt định mức |
| `fuel_norm_ratio` | `measured_consumption ÷ (định_mức × logged_engine_hours)` | như trên | tỉ lệ rất lớn (ví dụ hàng chục lần) → lệch cỡ, hỏi lại |
| `machine_hours_gap` | `reported_machine_hours − logged_engine_hours` (và `timesheet_machine_hours − logged_engine_hours`) | ba chỉ số giờ máy | cờ lệch giờ máy |
| `headcount_gap` | `reported_headcount − attendance_headcount` | hai chỉ số nhân công | báo sai so với chấm công |
| `progress_gap_reported_measured` | `reported_progress − measured_quantity` | hai chỉ số khối lượng | báo khống/thiếu |
| `progress_gap_measured_accepted` | `measured_quantity − accepted_quantity` | đo và nghiệm thu | khối lượng chưa được chấp nhận |
| `schedule_variance` | `measured_quantity lũy kế − kế hoạch lũy kế` | tiến độ và kế hoạch | chậm/nhanh so kế hoạch |
| `consecutive_days_over_norm` | số ngày liên tiếp có `fuel_norm_variance` vượt dung sai | chuỗi `fuel_norm_variance` | chỉ số dẫn xuất do **mã cài**; trigger chỉ so sánh `≥ 3` (P12) |
| `hours_since_last_maintenance` | `logged_engine_hours − giờ máy ở lần bảo dưỡng trước` | giờ máy, phiếu bảo dưỡng | trigger bảo dưỡng định kỳ (ví dụ ≥ 250 giờ) |

---

## 4. Raw reality — người Việt gõ thật trông ra sao

Dữ liệu hiện trường không sạch. Bảng dưới liệt kê **50 biến thể** (giả lập) và phân loại mỗi giá trị:

- **representation (cách biểu diễn)**: cùng một giá trị viết khác nhau (`4k` = `4.000L` = `bốn ngàn lít`) → chuẩn hoá được;
- **measurement (số đo/số ghi)**: giá trị do một nguồn có thể kiểm chứng ghi (phiếu kho, đồng hồ, hoá đơn, biên bản) → giữ nguyên, gắn chỉ số ngữ nghĩa;
- **estimate (ước lượng)**: giá trị có từ chỉ xấp xỉ ("khoảng", "tầm") → mang cờ `estimate`, **không** vào tổng chính xác.

Mỗi dòng trả lời bảy câu hỏi của brief: đâu là representation / measurement / estimate; nguồn nào có thẩm quyền (xem §2, chỉ trong cùng chỉ số); giá trị nào chuẩn hoá được; xung đột nào cần hoà giải; mơ hồ nào cần hỏi lại. Cột "Hành động" ghi `normalize` (chuẩn hoá), `reconcile` (hoà giải/derived fact), `clarify` (hỏi lại), `dedupe`, `giữ`.

| # | Raw (nguyên văn) | Phân loại | Chỉ số ngữ nghĩa | Chuẩn hoá thành | Hành động |
|---|---|---|---|---|---|
| 1 | `dầu 4k` | representation | `reported_fuel` | 4.000 L | normalize |
| 2 | `DA: 4.000` | representation | `reported_fuel` (DA = dầu theo phạm vi nhóm nhiên liệu; thiếu đơn vị) | 4.000 L (độ tin cậy trung bình) | normalize nếu nhóm nhiên liệu; **clarify** nếu kênh chung |
| 3 | `cấp thêm 4 khối dầu` | representation | `reported_fuel` | 4.000 L ("khối" của chất lỏng = m³) | normalize |
| 4 | `máy 03 lấy 2k sáng + 2k chiều` | representation | `reported_fuel` (EX-03) | hai lần 2.000 L, tổng 4.000 L; cờ lệch cỡ so dung tích bình | normalize + clarify (cấp vào máy hay bồn?) |
| 5 | `4,000 L` | representation | `reported_fuel` | 4.000 L (dấu phẩy phân cách nghìn kiểu Anh) | normalize |
| 6 | `4.000L` | representation | `reported_fuel` | 4.000 L (dấu chấm phân cách nghìn) | normalize |
| 7 | `bốn ngàn lít` | representation | `reported_fuel` | 4.000 L | normalize (số chữ) |
| 8 | `phiếu kho 3.850L nhưng đội trưởng báo 4k` | measurement + representation | `issued_fuel` 3.850 L và `reported_fuel` 4.000 L | hai fact **khác chỉ số** | reconcile: derived `issued_minus_reported_fuel` = −150 L, không chọn bên |
| 9 | `ERP = 3900, Excel = 4000, message = khoảng 4k` | measurement / representation / estimate | ERP: `issued_fuel`; Excel: bản chép; message: `reported_fuel` | 3.900 L; 4.000 L; ~4.000 L (estimate) | reconcile ERP với PXK (cùng chỉ số, cùng hạng); estimate không vào tổng |
| 10 | `khoảng 4k` | estimate | `reported_fuel` | ~4.000 L cờ estimate | normalize + giữ cờ |
| 11 | `tầm 4 khối` | estimate | `reported_fuel` | ~4.000 L cờ estimate | normalize + giữ cờ |
| 12 | `dầu 4k` rồi `dầu 4 khối` (cùng người, cùng ngày) | representation (trùng) | `reported_fuel` | một fact 4.000 L | dedupe, giữ cả hai observation |
| 13 | `dau 4K` | representation | `reported_fuel` | 4.000 L | normalize (không dấu, viết hoa) |
| 14 | `4 000 lít` | representation | `reported_fuel` | 4.000 L | normalize (khoảng trắng nghìn). Hiện thực đêm 21/10: gộp, trừ khi con số đứng sau từ nhãn / đếm / kỳ trong danh sách của bộ chuẩn hoá — danh sách rò (`bể 2 500 lít` nạp thành 2.500 L, soát C3, C3-02); hướng sửa chờ D-OP-9 |
| 15 | `4000l` | representation | `reported_fuel` | 4.000 L | normalize |
| 16 | `4k5 dầu` | representation | `reported_fuel` | 4.500 L | normalize ("4k5" = 4,5 nghìn) |
| 17 | `2,5 khối dầu` | representation | `reported_fuel` | 2.500 L | normalize (phẩy thập phân) |
| 18 | `0.5 khối dầu` | representation | `reported_fuel` | 500 L | normalize (chấm + một chữ số = thập phân) |
| 19 | `KL 0.35km` | representation | `reported_progress` | 350 m | normalize (KL = khối lượng thi công) |
| 20 | `350m` | representation | `reported_progress` | 350 m | normalize |
| 21 | `ba trăm năm chục mét` | representation | `reported_progress` | 350 m | normalize (số chữ) |
| 22 | `khoảng ba trăm năm chục mét` | estimate | `reported_progress` | ~350 m cờ estimate | normalize + giữ cờ |
| 23 | `làm được 1,2km` | representation | `reported_progress` | 1.200 m | normalize |
| 24 | `biên bản đo: 332 m` | measurement | `measured_quantity` | 332 m | giữ |
| 25 | `TVGS xác nhận 320 m` | measurement | `accepted_quantity` | 320 m (chỉ là hạng 1 khi đủ cả TVGS và CĐT) | giữ |
| 26 | `đổ 45 khối bê tông cột trụ P2` | representation | `reported_progress` (m³ thể tích) | 45 m³ ("khối" bê tông ≠ "khối" dầu) | normalize |
| 27 | `NC 22` | representation | `reported_headcount` | 22 người (mặc định, không kèm "công") | normalize |
| 28 | `22 công` | representation | `reported_mandays` (số công, xem ghi chú) | 22 công (ngày công) | normalize; **không** so với chấm công như cùng chỉ số |
| 29 | `chấm công: 20 người có mặt` | measurement | `attendance_headcount` | 20 người | giữ |
| 30 | `máy xúc chạy 7h` | representation | `reported_machine_hours` | 7 giờ (h ở khe giờ máy là thời lượng) | normalize |
| 31 | `mưa từ 14h` | representation | thời tiết | mưa từ 14:00 (h ở khe thời tiết là giờ trong ngày) | normalize |
| 32 | `7h máy` | representation | `reported_machine_hours` | 7 giờ | normalize |
| 33 | `đồng hồ giờ máy 6,4h` | measurement | `logged_engine_hours` | 6,4 giờ | giữ |
| 34 | `bảng chấm công ca máy: 1 ca` | measurement | `timesheet_machine_hours` | 8 giờ (1 ca = 8 giờ theo cấu hình pack) | normalize |
| 35 | `ca sáng` / `ca chiều` / `ca đêm` | representation | khoảng thời gian làm việc | khung giờ theo lịch ca của pack; ca đêm tính vào ngày bắt đầu | normalize |
| 36 | `xi 2k` | representation (thiếu đơn vị) | `issued_material` hoặc `requested_material` | không chuẩn hoá được (2.000 kg? 2 tấn? 2.000 bao?) | **clarify** |
| 37 | `xi 2 tấn` | representation | `reported` vật tư | 2.000 kg | normalize |
| 38 | `thep D16 20 tan nhap` | representation | nhập kho | thép D16, 20 tấn, hành động nhập | normalize (không dấu) |
| 39 | `500k tiền xe` | representation | chi phí phát sinh báo | 500.000 đồng (k = nghìn đồng) | normalize |
| 40 | `2tr5 tiền ăn` | representation | chi phí phát sinh báo | 2.500.000 đồng | normalize |
| 41 | `1 tỷ 2` | representation | giá trị | 1.200.000.000 đồng | normalize |
| 42 | `19/10`, `19-10-2026`, `19.10.26` | representation | ngày | 2026-10-19 (dd/mm, năm mặc định theo ngữ cảnh) | normalize |
| 43 | `23:50 UTC` cùng giao dịch ghi `20/10 sáng` | representation | thời điểm giao dịch | 2026-10-20 06:50 giờ Asia/Ho_Chi_Minh | normalize (múi giờ) |
| 44 | `PXK 0412`, `pxk-0412`, `PXK-0412` | representation | mã phiếu | `PXK-0412` | normalize |
| 45 | `máy 03`, `EX-03`, `EX03`, `xúc 03` | representation | mã thiết bị | `EX-03` theo phạm vi dự án | normalize; **clarify** nếu nhiều ứng viên |
| 46 | `máy 03` trong nhóm PRJ-B | representation | mã thiết bị | `ĐB-03` (theo phạm vi kênh) | normalize theo kênh |
| 47 | `INV-NL-1031: 4,1 m3` | measurement | `billed_fuel` | 4.100 L (m³ chất lỏng, phẩy thập phân) | normalize |
| 48 | `kho 3.850L, cảm biến 3.782L` | measurement + measurement | `issued_fuel` và `measured_consumption` | hai fact khác chỉ số | reconcile: derived `issued_minus_measured_fuel` = +68 L |
| 49 | `hoá đơn 4.100L` | measurement | `billed_fuel` | 4.100 L | giữ; derived `billed_minus_issued_fuel` = +250 L |
| 50 | `Dự án A hôm qua cấp 4k dầu đúng không?` | câu hỏi xác nhận + representation | **mơ hồ**: `reported_fuel`? `issued_fuel`? `measured_consumption`? `billed_fuel`? | **không xác nhận** | **clarify**: "4k" không rõ chỉ số, lại có xung đột giữa các nguồn; hỏi lại người dùng muốn xác nhận chỉ số nào |

Ghi chú: `reported_mandays` (số công do người báo) là chỉ số nhân công phụ của pack, đứng riêng với `reported_headcount` và `attendance_headcount` (BR-LAB-01). Dòng 50 là **ca xác nhận**: hệ thống không được trả "đúng"/"sai" khi chưa rõ chỉ số và khi các nguồn đang lệch; nó phải hỏi lại, rồi trả lời theo chỉ số được chọn kèm các chênh lệch derived fact.

---

## 5. Kiểm kê dữ liệu (Data Audit)

Ba mươi tập dữ liệu `DS-01` … `DS-30`, đủ các loại: có cấu trúc, bán cấu trúc, không cấu trúc, luồng (streaming), đa phương thức, dẫn xuất, dữ liệu gốc (master), giao dịch, tham chiếu, lịch sử. Mỗi tập có **đủ 16 thuộc tính** của brief, chia hai bảng nối nhau bằng mã `DS`. Các đánh giá chất lượng là **giả lập** (chưa đo).

### 5.1 Bảng A — định danh, thẩm quyền, cập nhật

| DS | Tập dữ liệu | Loại | Owner | Producer | Consumer | Schema | Authority | Update mechanism |
|---|---|---|---|---|---|---|---|---|
| DS-01 | Danh mục dự án | master, có cấu trúc | quản lý dự án | IT/Data | mọi phòng | `project(code,name,type,client,status)` | cao: nguồn gốc | lệnh có duyệt, audit |
| DS-02 | Danh mục thiết bị | master, có cấu trúc | máy móc | máy móc | công trường, kế toán, EACL | `equipment(code,type,project,tank_capacity_L)` | cao | lệnh có duyệt; gộp/tách cần duyệt |
| DS-03 | Danh mục vật tư | master/tham chiếu | vật tư | vật tư | kho, kế toán, EACL | `material(code,name,unit,density?)` | cao | lệnh có duyệt |
| DS-04 | Danh mục nhà cung cấp và nhà thầu phụ | master | vật tư/pháp chế | vật tư | kế toán, EACL | `party(code,name,type)` | cao | lệnh có duyệt |
| DS-05 | Danh mục nhân sự và đội | master, nhạy cảm | nhân sự | nhân sự | công trường, EACL (hạn chế) | `person(code,name,team,project)` | cao | hệ thống nhân sự |
| DS-06 | Bảng định mức (trích từ SOP) | tham chiếu, có kiểu | kỹ thuật | `data/parsing` + EACL | EACL tính định mức | `norm(equipment_class,L_per_hour,tolerance,doc_version)` | theo SOP hiệu lực | trích lúc nạp tài liệu (I1), công bố nguyên tử |
| DS-07 | ERP: tồn kho và giao dịch kho | giao dịch, có cấu trúc | kho/kế toán | ERP | kế toán, EACL | `inventory_txn(txn_id,warehouse,material,qty,unit,ts_utc)` | hạng 1 cho `issued_*` | kết nối ERP (I2) theo lô/thời gian thực |
| DS-08 | Phiếu nhập/xuất kho (PNK/PXK) | giao dịch, giấy + bản số hoá | kho | thủ kho | kế toán, EACL | `slip(slip_no,type,qty,unit,signed_by,version)` | hạng 1 khi có chữ ký | nhập tay/số hoá; bản sửa là phiên bản mới |
| DS-09 | Nhật ký cấp dầu cho máy | giao dịch | máy móc | thợ máy/thủ kho | thiết bị, EACL | `fuel_log(equipment,qty_L,ts,confirmed_by)` | hạng 2 cho `measured_consumption` | ghi tay/biểu mẫu |
| DS-10 | Đồng hồ giờ máy và cảm biến | luồng (streaming) | máy móc | thiết bị/telematics | EACL (sự kiện) | `reading(reading_id,equipment,metric,value,ts)` | hạng 1 cho `logged_engine_hours` | đẩy liên tục; trùng/đến trễ là bình thường |
| DS-11 | Báo cáo ngày chính thức | bán cấu trúc | quản lý dự án | giám sát | trưởng dự án, EACL | `daily_report(project,date,fields…)` | hạng 1 cho `reported_*` | workflow `daily-site-report` |
| DS-12 | Tin nhắn hiện trường (nhóm chat) | không cấu trúc, luồng | công trường | giám sát/đội trưởng | EACL | văn bản tự do + ảnh | hạng 2–3 cho `reported_*` | liên tục (I3), cổng mơ hồ |
| DS-13 | Bảng tính Excel tổng hợp | bán cấu trúc, dẫn xuất | từng phòng | nhân viên | quản lý | cột tuỳ người, nhiều phiên bản | thấp (bản chép) | thủ công; nạp I2 nếu cột chuẩn |
| DS-14 | Chấm công và bảng lương | giao dịch, nhạy cảm | nhân sự | nhân sự | kế toán | `attendance(person,date,hours)`, `payroll(...)` | hạng 1 cho `attendance_*`, `paid_*` | chốt theo kỳ |
| DS-15 | BOQ (bảng khối lượng) | bán cấu trúc | đấu thầu/kỹ thuật | dự toán | kỹ thuật, kế toán | `boq_item(code,desc,unit,qty,price,version)` | tài liệu hợp đồng | phiên bản; trích bảng lúc nạp |
| DS-16 | Hợp đồng và phụ lục | không cấu trúc | pháp chế | pháp chế | quản lý dự án, kế toán | PDF/DOCX + metadata phiên bản/hiệu lực | hạng 1 theo phiên bản hiệu lực | nạp I1, phụ lục là phiên bản mới |
| DS-17 | Bản vẽ và metadata | không cấu trúc/đa phương thức | kỹ thuật | tư vấn thiết kế | công trường, kỹ thuật | PDF/CAD + `revision` | hạng 1 theo revision hiệu lực | workflow `document-revision` |
| DS-18 | SOP và quy định nội bộ | không cấu trúc | các phòng | phòng chủ quản | mọi phòng, EACL (RAG) | văn bản có heading/bảng | hạng 1 cho quy định nội bộ hiệu lực | workflow `document-revision`, I1 |
| DS-19 | Sổ tay thiết bị | không cấu trúc | máy móc | nhà sản xuất | thiết bị, EACL | PDF kỹ thuật | tham khảo, thấp hơn SOP nội bộ | nạp I1 |
| DS-20 | Biên bản nghiệm thu và kiểm tra | bán cấu trúc | kỹ thuật/QA | kỹ thuật/TVGS | kế toán, EACL | biểu mẫu + chữ ký | hạng 1 khi đủ xác nhận | workflow `quantity-acceptance` |
| DS-21 | Kết quả thí nghiệm vật liệu | bán cấu trúc | QA/QC | phòng thí nghiệm | QA/QC, EACL | `lab_result(sample,test,value,unit)` | hạng 1 | nạp I1/I2 khi có kết quả |
| DS-22 | Hồ sơ an toàn và sự cố | bán cấu trúc, nhạy cảm | an toàn | giám sát/an toàn | BGĐ, EACL | `incident(id,severity,desc,persons)` | hạng 1 | workflow `safety-incident` |
| DS-23 | Hoá đơn và hồ sơ thanh toán | giao dịch | kế toán | nhà cung cấp/kế toán | tài chính, EACL | `invoice(no,party,qty,amount_before_tax,total)` | hạng 1 cho `billed_*` | ERP, workflow `payment-certification` |
| DS-24 | Ngân sách và chi phí | giao dịch/dẫn xuất | tài chính | ERP | BGĐ, quản lý dự án | `cost(project,item,committed,invoiced,paid)` | hạng 1 theo từng chỉ số chi phí | ERP theo kỳ |
| DS-25 | Kế hoạch tiến độ | bán cấu trúc | quản lý dự án | quản lý dự án | EACL tính `schedule_variance` | `plan(item,start,end,qty_cumulative)` | kế hoạch chuẩn hiện hành | phiên bản kế hoạch |
| DS-26 | Ảnh/video công trường | đa phương thức | công trường | giám sát | an toàn, quản lý | tệp ảnh/video + thời điểm | không dùng làm số liệu (cổng thị giác đóng) | tải lên qua chat |
| DS-27 | RFI và phát sinh | bán cấu trúc | kỹ thuật | kỹ thuật | quản lý, pháp chế | `rfi(...)`, `change_order(...)` | hạng 1 | workflow `rfi-handling`, `change-order` |
| DS-28 | Số liệu lịch sử (kỳ đã khoá) | lịch sử | quản lý dự án | EACL/ERP | phân tích, báo cáo | fact và báo cáo của kỳ đã `LOCKED` | cao (đã khoá) | chỉ ghi thêm |
| DS-29 | Kho sự thật (fact, derived fact) | dẫn xuất | IT/Data | EACL | mọi truy vấn | `facts`, `derived_facts`, `fact_conflicts` | theo ma trận §2 | I2/I3/E; tính lại khi đầu vào đổi |
| DS-30 | Cảnh báo thời tiết | tham chiếu, ngoài | an toàn | nguồn cảnh báo ngoài | công trường, an toàn | `weather_alert(area,level,valid_from,valid_to)` | tham khảo | sự kiện đến |

### 5.2 Bảng B — chất lượng và vòng đời

| DS | Freshness | Latency | Retention | Version | PII/Sensitivity | Permission | Quality | Completeness | Consistency | Lineage |
|---|---|---|---|---|---|---|---|---|---|---|
| DS-01 | khi đổi | thấp | suốt vòng đời công ty | có (lịch sử đổi) | thấp | mọi vai đọc | cao | cao | cao | tạo bởi lệnh, có audit |
| DS-02 | khi đổi | thấp | suốt vòng đời tài sản | có | thấp | thiết bị ghi, còn lại đọc | trung bình (trùng mã) | trung bình | trung bình | lệnh → audit |
| DS-03 | khi đổi | thấp | dài | có | thấp | vật tư ghi | trung bình | trung bình | trung bình | lệnh → audit |
| DS-04 | khi đổi | thấp | dài | có | trung bình | theo vai | trung bình | trung bình | trung bình | lệnh → audit |
| DS-05 | theo kỳ | thấp | theo quy định lưu hồ sơ | có | cao | hr_officer, director | trung bình | cao | trung bình | hệ thống nhân sự |
| DS-06 | theo phiên bản SOP | thấp | theo vòng đời SOP | có (gắn `doc_version`) | thấp | mọi vai đọc | phụ thuộc cổng parser | phụ thuộc bảng | cao nếu một bản hiệu lực | trích từ DS-18 |
| DS-07 | gần thời gian thực | giây–phút | theo quy định kế toán | giao dịch bất biến | thấp | kho, kế toán | cao | cao | trung bình (lệch múi giờ) | ERP → observation |
| DS-08 | theo ngày | giờ–ngày (phiếu giấy) | theo quy định hồ sơ | có (bản sửa = bản mới) | thấp | kho, kế toán | thấp–trung bình (viết tay) | trung bình | thấp (hai bản) | giấy → số hoá → observation |
| DS-09 | theo ngày | giờ | theo kỳ | không rõ | thấp | thiết bị, kho | trung bình | trung bình | trung bình | ghi tay |
| DS-10 | gần thời gian thực | giây | thời hạn ngắn cho số đo thô | theo `reading_id` | thấp | thiết bị | trung bình (trùng/đến trễ) | trung bình (mất sóng) | trung bình | thiết bị → E0 |
| DS-11 | theo ngày | giờ | suốt dự án + hợp đồng | có (trả lại sửa) | thấp | trưởng dự án, giám sát | trung bình | trung bình | trung bình | từ DS-12 qua workflow |
| DS-12 | liên tục | phút | theo chính sách chat | không | trung bình (số điện thoại, tên) | theo nhóm | thấp | thấp | thấp | người gõ → I3 |
| DS-13 | không đều | không đều | tuỳ người | nhiều bản | thấp–trung bình | theo phòng | thấp | thấp | thấp | chép từ nhiều nguồn |
| DS-14 | theo kỳ | ngày | theo quy định hồ sơ | có | cao | hr_officer, director, kế toán | cao | cao | cao | hệ thống nhân sự |
| DS-15 | theo phiên bản | thấp | suốt hợp đồng | có | thấp | kỹ thuật, kế toán | trung bình | cao | trung bình (lệch bản vẽ) | từ hợp đồng |
| DS-16 | theo phiên bản | thấp | suốt hợp đồng + thời hạn lưu | có (gốc + phụ lục) | trung bình | legal_officer, quản lý | cao | cao | trung bình (phụ lục mâu thuẫn) | ký → I1 |
| DS-17 | theo revision | thấp | suốt dự án | có (`revision`) | thấp | kỹ thuật, công trường | cao | cao | trung bình (bản lỗi thời ở hiện trường) | thiết kế → I1 |
| DS-18 | theo phiên bản | thấp | suốt vòng đời SOP | có | thấp | mọi vai đọc | cao | cao | trung bình (hai bản cùng hiệu lực) | chủ SOP → I1 |
| DS-19 | cố định | thấp | suốt vòng đời máy | có | thấp | thiết bị | cao | cao | cao | nhà sản xuất → I1 |
| DS-20 | theo đợt | ngày | suốt hợp đồng | có | thấp | kỹ thuật, QA, kế toán | cao | cao | cao | workflow → I1 |
| DS-21 | theo đợt thí nghiệm | ngày | suốt dự án | có | thấp | QA/QC | cao | cao | cao | phòng thí nghiệm → I1 |
| DS-22 | theo sự cố | giờ | theo quy định | có | cao | safety_officer, director | trung bình | trung bình | trung bình | workflow |
| DS-23 | theo ngày | ngày | theo quy định kế toán | có | thấp | kế toán, tài chính | cao | cao | trung bình (trùng hoá đơn) | ERP |
| DS-24 | theo kỳ | ngày | theo quy định kế toán | có theo kỳ | thấp | tài chính, quản lý | cao | cao | cao | ERP |
| DS-25 | theo phiên bản | thấp | suốt dự án | có | thấp | quản lý, kỹ thuật | trung bình | cao | trung bình | quản lý dự án |
| DS-26 | theo tải lên | phút | theo chính sách | không | trung bình (người trong ảnh) | theo nhóm | không dùng làm số liệu | thấp | — | giám sát → chat |
| DS-27 | theo vụ việc | ngày | suốt hợp đồng | có | thấp | kỹ thuật, pháp chế | trung bình | trung bình | trung bình | workflow |
| DS-28 | theo kỳ đã khoá | thấp | dài | chỉ ghi thêm | thấp | quản lý, BGĐ | cao | cao | cao | từ DS-29/DS-11 |
| DS-29 | theo nguồn đầu vào | giây–phút | dài, chỉ ghi thêm (cần chính sách ẩn danh, xem F-006) | có (derived fact ghi công thức) | trung bình | lọc theo ABAC | phụ thuộc đầu vào | theo nguồn | cao nếu hoà giải đúng | observation → fact → derived |
| DS-30 | gần thời gian thực | phút | ngắn | không | thấp | mọi vai đọc | trung bình | trung bình | — | nguồn ngoài → E0 |

### 5.3 Các vấn đề chất lượng cần tìm

| Vấn đề | Ví dụ ở tập dữ liệu | Cách phát hiện | Ca hỏng |
|---|---|---|---|
| duplicate (trùng) | cùng báo cáo vào hai nhóm chat (DS-12); hai bản phiếu (DS-08) | `duplicate_key_check` | `CON-F-001` |
| conflict (xung đột) | PXK 3.850 L và ERP 3.900 L cùng `issued_fuel` (DS-07/08) | `fact_conflicts` | `CON-F-002` |
| stale (cũ) | tồn kho ERP quá hạn (DS-07) | `freshness_check` | `CON-F-003` |
| missing (thiếu) | báo cáo ngày thiếu giờ máy (DS-11) | `required_fields_complete` | `CON-F-004` |
| invalid (không hợp lệ) | 4.000 L cho máy xúc 7 giờ (DS-12) | `cross_field_range` | `CON-F-062` |
| outlier (ngoại lai) | số đo cảm biến vọt (DS-10) | ngưỡng + trigger | `CON-F-050` |
| schema drift (lệch lược đồ) | Excel đổi tên/thứ tự cột (DS-13) | `schema_valid` lúc nạp | `CON-F-009` |
| semantic drift (lệch nghĩa) | "KL", "NC", "DA" đổi nghĩa theo phòng/tài liệu | lexicon theo phạm vi | `CON-F-016`, `CON-F-018` |
| unit mismatch (lệch đơn vị) | "4 khối dầu" so với "4 khối bê tông" (DS-12) | `unit_material_consistency` | `CON-F-012` |
| timezone mismatch (lệch múi giờ) | giao dịch ERP UTC so với báo cáo giờ địa phương (DS-07/11) | chuẩn hoá thời gian | `CON-F-049` |
| version mismatch (lệch phiên bản) | SOP v2 và v3 cùng hiệu lực (DS-18) | metadata hiệu lực | `CON-F-023` |

---

## 6. Domain Dictionary (từ điển miền)

Từ điển do `lexicon.yaml` của pack nạp; **cùng từ khác phòng ban → khác nghĩa**, **cùng viết tắt khác tài liệu → khác nghĩa**, **cùng con số khác đơn vị → khác nghĩa nghiệp vụ**. Luồng giải nghĩa: `RAW TOKEN → tenant → domain → phòng ban/phạm vi kênh → loại tài liệu → trạng thái workflow → ngữ cảnh xung quanh → từ điển → các nghĩa ứng viên → độ tin cậy → giải nghĩa hoặc hỏi lại`. Ví dụ và phản ví dụ là giả lập.

| Canonical | Aliases | Viết tắt | Nghĩa theo phòng ban | Nghĩa theo ngữ cảnh | Đơn vị | Kiểu | Luật kiểm | Ví dụ | Phản ví dụ |
|---|---|---|---|---|---|---|---|---|---|
| dầu | dầu DO, diesel, dầu máy | DA, DO | kho: hàng xuất theo phiếu; thiết bị: nhiên liệu tiêu thụ; kế toán: hàng trên hoá đơn | `DA` = dầu trong nhóm báo cáo nhiên liệu | L | decimal | > 0, trong miền khe | `DA 4.000L` → 4.000 L | `DA A: 4.000 m³` ở báo cáo quản lý: DA là dự án A |
| dự án | công trình | DA, PRJ | quản lý: đơn vị quản lý chi phí/tiến độ | `DA` = dự án ở cấp quản lý | — | mã `PRJ-x` | mã tồn tại, người hỏi có quyền | `DA B` → `PRJ-B` | `DA 4.000L` trong nhóm nhiên liệu |
| lít | l, L | L | mọi phòng | — | L | decimal | số ≥ 0 | `4000l` | `4 lít/giờ` không phải tổng |
| khối | m3, m³ | m³ | kho: chất lỏng 1 khối = 1.000 L; kỹ thuật/QA: thể tích bê tông/đất | phụ thuộc chất liệu | m³ hoặc L | decimal | `unit_material_consistency` | `4 khối dầu` = 4.000 L | `45 khối bê tông` = 45 m³, không đổi sang lít |
| k | K, ngàn, nghìn | k | mọi phòng | sau số ở khe lượng/tiền = nghìn | nhân 1.000 | hệ số | cần khe/đơn vị đi kèm | `dầu 4k`, `500k tiền xe` | `xi 2k` thiếu đơn vị → hỏi lại |
| triệu | tr | tr | kế toán, đấu thầu | `2tr5` = 2,5 triệu | đồng | decimal | ≥ 0 | `2tr5` = 2.500.000 | `2tr5 m³` không hợp lệ |
| tỷ | tỉ | tỷ | tài chính | `1 tỷ 2` = 1,2 tỷ | đồng | decimal | ≥ 0 | `1 tỷ 2` | — |
| tấn | t, T | t | kho, vật tư | 1 tấn = 1.000 kg | tấn, kg | decimal | ≥ 0 | `20 tấn thép` | `20t` trong ngữ cảnh nhiệt độ: không phải tấn |
| kilôgam | kg | kg | kho | — | kg | decimal | ≥ 0 | `2.000 kg xi măng` | — |
| mét | m | m | kỹ thuật, công trường | tiến độ chiều dài | m | decimal | theo hạng mục | `350m` | `P2` không phải mét |
| ki-lô-mét | km | km | công trường (đường) | 1 km = 1.000 m | km → m | decimal | đổi sang m | `0.35km` = 350 m | `km 12` là cột mốc, không phải độ dài |
| mét vuông | m2, m² | m² | kỹ thuật | diện tích mặt đường/vỉa hè | m² | decimal | ≥ 0 | `1.200 m²` | — |
| giờ | h, tiếng | h | thiết bị: thời lượng (giờ máy); công trường: giờ trong ngày | khe giờ máy = thời lượng; khe thời tiết = giờ trong ngày | giờ | decimal/time | theo khe | `chạy 7h` = 7 giờ; `mưa từ 14h` = 14:00 | `7h sáng` là giờ trong ngày |
| ca | ca làm việc | — | công trường | ca sáng/chiều/đêm theo lịch ca của pack | khung giờ | enum | trong lịch ca | `ca đêm` bắt đầu 19:00 | `ca máy` là đơn vị khác |
| ca máy | công máy | CM | thiết bị, kế toán | 1 ca máy = 8 giờ theo cấu hình | ca → giờ | decimal | ≥ 0 | `1 CM` = 8 giờ | `15 cm` là centimét, không phải ca máy |
| giờ máy | tiếng máy, h máy | — | thiết bị | giờ vận hành thực | giờ | decimal | ≤ 24 giờ/ngày/máy | `máy xúc chạy 7h` | `7h` ở khe thời tiết |
| nhân công | công nhân, thợ | NC | công trường: số người; kế toán/nhân sự: số công | `NC 22` không kèm "công" = 22 người | người / công | integer | ≥ 0, ≤ quân số dự án | `NC 22` = 22 người | `22 công` là ngày công |
| công | ngày công | — | nhân sự, kế toán | man-day | công | decimal | ≥ 0 | `22 công` | không so với `attendance_headcount` |
| khối lượng | KL, lượng | KL | kỹ thuật: khối lượng thi công; kho: khối lượng vật tư; vật lý: mass | `KL 0.35km` = tiến độ chiều dài | theo hạng mục | decimal | đúng chỉ số (reported/measured/accepted/billed) | `KL 0.35km` → 350 m | `KL` = kilôgam trong sổ kho |
| tiến độ | progress | — | quản lý dự án | khối lượng làm được so kế hoạch | m, m³, % | decimal | trong 0–100% khi là tỉ lệ | `đạt 80%` | `80%` không phải khối lượng tuyệt đối |
| định mức | ĐM, norm | ĐM | thiết bị/kỹ thuật | tiêu hao chuẩn theo loại máy/việc | L/giờ máy | decimal | từ SOP hiệu lực | máy xúc 1,2 m³: 17 L/giờ (giả lập) | định mức trong sổ tay hãng khác SOP nội bộ |
| dung sai | tolerance | — | thiết bị/kỹ thuật | ngưỡng chênh được chấp nhận | % | decimal | ≥ 0 | +15% | — |
| phiếu xuất kho | PXK | PXK | kho: chứng từ xuất; kế toán: căn cứ hạch toán | thẩm quyền hạng 1 cho `issued_*` khi có chữ ký | — | mã `PXK-NNNN` | số phiếu duy nhất | `PXK-0412` | `pxk 0412` chưa chuẩn hoá |
| phiếu nhập kho | PNK | PNK | kho | chứng từ nhập | — | mã `PNK-NNNN` | duy nhất | `PNK-0233` | — |
| đề nghị vật tư | ĐNVT, yêu cầu vật tư | ĐNVT | công trường: yêu cầu; vật tư: đầu vào mua sắm | phiếu có vòng đời `material-request` | — | mã `VT-NNNN` | có hạn mức/đính kèm | `VT-0087` | — |
| đơn đặt hàng | PO | PO | vật tư, kế toán | cam kết mua (`committed_cost`) | — | mã `PO-NNNN` | khớp phiếu duyệt | `PO-0456` | — |
| hoá đơn | HĐ GTGT | HĐ | kế toán | `billed_*` | đồng/số lượng | mã | duy nhất | `INV-2026-0912` | `HĐ` còn là hợp đồng |
| hợp đồng | HĐ | HĐ | pháp chế, quản lý | tài liệu có phiên bản/phụ lục | — | mã `HD-…` | phiên bản hiệu lực | `HD-A-2026-01` | `HĐ` trên hoá đơn là hoá đơn |
| phụ lục | PL | PL | pháp chế | sửa đổi hợp đồng, là phiên bản mới | — | mã | gắn hợp đồng gốc | `Phụ lục 1 của HD-NTP-A-03` | — |
| chủ đầu tư | bên A | CĐT | quản lý, pháp chế | bên ký hợp đồng, nghiệm thu, thanh toán | — | thực thể | có trong danh mục bên | `CĐT dự án A` | — |
| tư vấn giám sát | giám sát bên A | TVGS | kỹ thuật, QA | xác nhận khối lượng/chất lượng thay CĐT | — | thực thể | có trong danh mục bên | `TVGS xác nhận 320 m` | `giám sát` hiện trường của ABC là `site_supervisor` |
| nhà thầu phụ | thầu phụ | NTP | quản lý, pháp chế | thi công hạng mục chuyên biệt | — | thực thể | có hợp đồng | `NTP cọc khoan nhồi` | — |
| nhà cung cấp | NCC | NCC | vật tư, kế toán | bán vật tư/nhiên liệu | — | thực thể | có trong danh mục | `NCC thép` | `NC` là nhân công |
| ban quản lý dự án | BQLDA | BQLDA | quản lý | đại diện CĐT điều hành dự án | — | thực thể | — | `BQLDA` | — |
| bản vẽ | BV, drawing | BV | kỹ thuật | tài liệu thiết kế có `revision` | — | mã + revision | dùng revision hiệu lực | `BV-A-P2-03 rev C` | bản vẽ rev cũ ở hiện trường |
| bảng khối lượng | BOQ | BOQ | đấu thầu, kỹ thuật | danh sách hạng mục + khối lượng + đơn giá | theo hạng mục | bảng | khớp bản vẽ hiện hành | `BOQ-A-v2` | BOQ lệch bản vẽ là xung đột |
| hạng mục | HM | HM | kỹ thuật, kế toán | đơn vị khối lượng/thanh toán | — | mã | thuộc một dự án | `trụ P2` | — |
| trụ (cầu) | P1, P2… | P | kỹ thuật (cầu) | hạng mục kết cấu của cầu | — | mã hạng mục | thuộc dự án cầu | `trụ P2` | `P2` không phải mét |
| bê tông | BT | BT | QA/kỹ thuật | vật liệu kết cấu | m³ | decimal | cấp bền theo thiết kế | `đổ 45 khối bê tông` | — |
| bê tông nhựa | BTN | BTN | công trường (đường) | hỗn hợp nhựa đường rải mặt | tấn / m² | decimal | có phiếu cân | `rải BTN` | `BTN` không phải bê tông thường |
| bê tông xi măng | BTXM | BTXM | công trường | mặt đường/kết cấu bê tông xi măng | m³ | decimal | cấp bền | `BTXM mặt cầu` | — |
| cấp phối đá dăm | CPĐD | CPĐD | công trường (đường) | lớp móng đường | m³ / tấn | decimal | thí nghiệm đầm chặt | `CPĐD lớp 2` | — |
| xi măng | xi | — | kho | vật tư rời hoặc bao | kg, tấn, bao | decimal | đơn vị bắt buộc | `2 tấn xi măng` | `xi 2k` thiếu đơn vị → hỏi lại |
| thép | thép D16 | — | kho, kỹ thuật | thép xây dựng theo đường kính | kg, tấn | decimal | đường kính hợp lệ | `thép D16` | `D16` không phải mã dự án |
| nhựa đường | bitum | — | kho, công trường | vật liệu rải mặt | tấn | decimal | có phiếu cân | `20 tấn nhựa đường` | — |
| nghiệm thu | NT | NT | kỹ thuật, quản lý | xác nhận khối lượng/chất lượng hoàn thành | — | quy trình | đủ xác nhận TVGS và CĐT | `nghiệm thu hạng mục M2` | `nghiệm thu` không đồng nghĩa "đã đổ" |
| biên bản nghiệm thu | BBNT | BBNT | kỹ thuật | chứng từ nghiệm thu | — | mã `BB-NT-NNNN` | đủ chữ ký | `BB-NT-0415` | — |
| phát sinh | PS | PS | kỹ thuật, pháp chế | thay đổi khối lượng/điều kiện so hợp đồng | m³… | mã `PS-NNN` | ghi cơ sở đo (nguyên khối/tơi xốp) | `PS-012` | — |
| yêu cầu làm rõ | RFI | RFI | kỹ thuật | câu hỏi gửi TVGS/thiết kế | — | mã `RFI-NNN` | có hạn trả lời | `RFI-031` | — |
| báo cáo không phù hợp | NCR | NCR | QA/QC | ghi nhận hạng mục không đạt | — | mã | không đóng khi còn mở | `NCR` mẫu bê tông không đạt | — |
| kiểm tra chất lượng | QA/QC, KCS | KCS | QA/QC | kiểm soát chất lượng | — | quy trình | — | `KCS bê tông` | — |
| an toàn lao động | ATLĐ | ATLĐ | an toàn | bảo đảm an toàn người lao động | — | lĩnh vực | — | `huấn luyện ATLĐ` | — |
| sự cố | SC | SC | an toàn, thiết bị | sự kiện gây/có thể gây thiệt hại | — | mã `SC-NNN` | có mức nghiêm trọng | `SC-014` | `sự cố máy` là phiếu bảo dưỡng |
| suýt xảy ra | near-miss | — | an toàn | sự cố không gây thiệt hại | — | enum | tách khỏi sự cố gây thiệt hại | `ngã nhưng không bị thương` | — |
| mức nghiêm trọng | severity | — | an toàn | 1–4 (ví dụ giả lập) | — | integer | ≥ mức tối thiểu của luật | mức 2 cho ngã trên cao | mức do model đề xuất thấp hơn luật |
| nhật ký thi công | NKTC | NKTC | công trường | ghi chép chính thức hằng ngày, có người ký | — | tài liệu | có chữ ký | `nhật ký thi công 19/10` | báo cáo ngày tự động chưa phải nhật ký chính thức |
| báo cáo ngày | BCN | BCN | công trường, quản lý | báo cáo hằng ngày theo năm khe | — | workflow | đủ năm khe | `Daily Project Report` | — |
| bảo dưỡng | BD | BD | thiết bị | kiểm tra, sửa chữa theo lịch/sự cố | — | phiếu `MT-NNNN` | đúng hạn giờ máy | `BD 250 giờ` | `BD-B1` là mã xe ủi, không phải bảo dưỡng |
| máy xúc | xúc | EX | thiết bị, công trường | máy đào bánh xích | — | mã `EX-NN` | duy nhất toàn công ty | `EX-01` | `máy 03` chưa rõ loại |
| xe ủi | ủi | BD (mã) | công trường (đường) | máy ủi san nền | — | mã | duy nhất | `BD-B1` | — |
| đầm bánh | xe lu | ĐB | công trường (đường) | máy lu/đầm bánh thép | — | mã `ĐB-NN` | duy nhất | `ĐB-03` | — |
| xe bồn | bồn dầu | — | kho, thiết bị | phương tiện/bồn chứa nhiên liệu | L | mã | dung tích khai báo | bồn dầu công trường A | — |
| tồn kho | TK | TK | kho | số lượng còn trong kho | theo vật tư | decimal | ≥ 0, không âm | `tồn xi măng WH-A1` | số `ObservedAt` cũ là dữ liệu cũ |
| mức an toàn | safety stock | — | vật tư, kho | tồn tối thiểu cần giữ | theo vật tư | decimal | ≥ 0 | thép D16: 10 tấn | — |
| thanh toán | TT | TT | kế toán, tài chính | trả tiền theo hoá đơn/khối lượng | đồng | quy trình | khớp ba bên | `thanh toán INV-2026-0912` | — |
| giữ lại | retention | — | tài chính | phần tiền giữ lại theo hợp đồng | đồng, % | decimal | theo hợp đồng | `giữ lại 5%` (giả lập) | — |

Số thuật ngữ: **66** (đếm các dòng của bảng trên bằng script tạm khi viết tài liệu).

---

## 7. Schema chuẩn (JSON Schema)

Schema chuẩn của **measurement/fact** do pack khai báo; mọi bản ghi vào kho sự thật phải qua `schema_valid` và luật chéo trường. Các định nghĩa dùng chung:

```json
{
  "$defs": {
    "quantity": {
      "type": "object",
      "required": ["value", "unit"],
      "properties": {
        "value": {"type": "number"},
        "unit": {"type": "string"},
        "estimate": {"type": "boolean", "default": false}
      }
    },
    "evidence": {
      "type": "object",
      "required": ["observation_id", "source_type", "observed_at"],
      "properties": {
        "observation_id": {"type": "string"},
        "source_type": {"type": "string"},
        "span": {"type": "string"},
        "observed_at": {"type": "string", "format": "date-time"}
      }
    }
  }
}
```

### 7.1 Báo cáo ngày (`daily_report`)

```json
{
  "$id": "daily_report", "title": "Báo cáo ngày", "type": "object",
  "required": ["report_id", "project_code", "work_date", "status", "progress", "machine_hours", "fuel", "headcount", "weather", "evidence"],
  "properties": {
    "report_id": {"type": "string"},
    "project_code": {"type": "string", "pattern": "^PRJ-[A-Z]$"},
    "work_date": {"type": "string", "format": "date"},
    "status": {"enum": ["DRAFT", "SUBMITTED", "UNDER_REVIEW", "APPROVED", "LOCKED"]},
    "progress": {"type": "object", "properties": {"semantic_metric": {"const": "reported_progress"}, "item_code": {"type": "string"}, "quantity": {"$ref": "#/$defs/quantity"}}},
    "machine_hours": {"type": "array", "items": {"type": "object", "required": ["equipment_code", "hours"], "properties": {"equipment_code": {"type": "string"}, "hours": {"type": "number", "minimum": 0, "maximum": 24}}}},
    "fuel": {"type": "object", "properties": {"semantic_metric": {"const": "reported_fuel"}, "quantity": {"$ref": "#/$defs/quantity"}}},
    "headcount": {"type": "object", "required": ["value", "basis"], "properties": {"value": {"type": "number", "minimum": 0}, "basis": {"enum": ["person", "manday"]}}},
    "weather": {"type": "object", "properties": {"condition": {"type": "string"}, "from_time": {"type": "string"}, "to_time": {"type": "string"}, "estimate": {"type": "boolean"}}},
    "evidence": {"type": "array", "items": {"$ref": "#/$defs/evidence"}, "minItems": 1},
    "confidence": {"type": "number", "minimum": 0, "maximum": 1}
  }
}
```

### 7.2 Xuất nhiên liệu (`fuel_issue`)

```json
{
  "$id": "fuel_issue", "title": "Xuất nhiên liệu", "type": "object",
  "required": ["slip_no", "project_code", "warehouse_code", "target", "quantity", "semantic_metric", "issued_at", "evidence"],
  "properties": {
    "slip_no": {"type": "string", "pattern": "^PXK-[0-9]{4}[A-Z]?$"},
    "project_code": {"type": "string"},
    "warehouse_code": {"type": "string"},
    "target": {"type": "object", "required": ["kind", "code"], "properties": {"kind": {"enum": ["tank", "equipment"]}, "code": {"type": "string"}}},
    "material": {"const": "diesel"},
    "quantity": {"$ref": "#/$defs/quantity"},
    "semantic_metric": {"const": "issued_fuel"},
    "issued_at": {"type": "string", "format": "date-time"},
    "signed_by": {"type": ["string", "null"]},
    "version": {"type": "integer", "minimum": 1},
    "supersedes": {"type": ["string", "null"]},
    "evidence": {"type": "array", "items": {"$ref": "#/$defs/evidence"}, "minItems": 1}
  }
}
```

### 7.3 Giờ máy (`machine_hours`)

```json
{
  "$id": "machine_hours", "title": "Giờ máy", "type": "object",
  "required": ["equipment_code", "work_date", "semantic_metric", "hours", "source_type", "evidence"],
  "properties": {
    "equipment_code": {"type": "string"},
    "work_date": {"type": "string", "format": "date"},
    "semantic_metric": {"enum": ["reported_machine_hours", "logged_engine_hours", "timesheet_machine_hours"]},
    "hours": {"type": "number", "minimum": 0, "maximum": 24},
    "source_type": {"enum": ["daily_report", "chat_message", "engine_meter", "timesheet"]},
    "estimate": {"type": "boolean", "default": false},
    "evidence": {"type": "array", "items": {"$ref": "#/$defs/evidence"}, "minItems": 1}
  }
}
```

### 7.4 Khối lượng thi công (`work_quantity`)

```json
{
  "$id": "work_quantity", "title": "Khối lượng thi công", "type": "object",
  "required": ["project_code", "item_code", "work_date", "semantic_metric", "quantity", "evidence"],
  "properties": {
    "project_code": {"type": "string"},
    "item_code": {"type": "string"},
    "work_date": {"type": "string", "format": "date"},
    "semantic_metric": {"enum": ["reported_progress", "measured_quantity", "accepted_quantity", "billed_quantity"]},
    "quantity": {"$ref": "#/$defs/quantity"},
    "volume_basis": {"enum": ["in_place", "loose", "not_applicable"]},
    "confirmed_by": {"type": "array", "items": {"enum": ["site_engineer", "tvgs", "client"]}},
    "evidence": {"type": "array", "items": {"$ref": "#/$defs/evidence"}, "minItems": 1}
  }
}
```

### 7.5 Xuất vật tư (`material_issue`)

```json
{
  "$id": "material_issue", "title": "Xuất vật tư", "type": "object",
  "required": ["slip_no", "project_code", "warehouse_code", "material_code", "quantity", "semantic_metric", "issued_at", "evidence"],
  "properties": {
    "slip_no": {"type": "string", "pattern": "^PXK-[0-9]{4}[A-Z]?$"},
    "project_code": {"type": "string"},
    "warehouse_code": {"type": "string"},
    "material_code": {"type": "string"},
    "quantity": {"$ref": "#/$defs/quantity"},
    "request_no": {"type": ["string", "null"], "pattern": "^VT-[0-9]{4}$"},
    "purpose": {"type": "string"},
    "semantic_metric": {"const": "issued_material"},
    "issued_at": {"type": "string", "format": "date-time"},
    "signed_by": {"type": ["string", "null"]},
    "evidence": {"type": "array", "items": {"$ref": "#/$defs/evidence"}, "minItems": 1}
  }
}
```

### 7.6 Sự cố (`incident`)

```json
{
  "$id": "incident", "title": "Sự cố an toàn", "type": "object",
  "required": ["incident_id", "project_code", "occurred_at", "category", "severity_final", "kind", "evidence"],
  "properties": {
    "incident_id": {"type": "string", "pattern": "^SC-[0-9]{3,}$"},
    "project_code": {"type": "string"},
    "occurred_at": {"type": "string", "format": "date-time"},
    "location": {"type": "string"},
    "category": {"enum": ["fall_from_height", "equipment", "electrical", "traffic", "other"]},
    "kind": {"enum": ["damage", "near_miss"]},
    "severity_proposed": {"type": "object", "properties": {"level": {"type": "integer", "minimum": 1, "maximum": 4}, "by": {"enum": ["model", "human"]}}},
    "severity_min_by_rule": {"type": "integer", "minimum": 1, "maximum": 4},
    "severity_final": {"type": "integer", "minimum": 1, "maximum": 4, "description": "do engine quyết: lớn hơn giữa đề xuất và mức tối thiểu của luật"},
    "injured_count": {"type": "integer", "minimum": 0},
    "site_locked": {"type": "boolean"},
    "notified": {"type": "array", "items": {"type": "string"}},
    "description": {"type": "string"},
    "evidence": {"type": "array", "items": {"$ref": "#/$defs/evidence"}, "minItems": 1}
  }
}
```

Ví dụ kết quả chuẩn hoá của `dầu 4k` (giả lập), áp dụng cho một **measurement** trước khi thành fact:

```json
{
  "semantic_metric": "reported_fuel",
  "project_code": "PRJ-A",
  "work_date": "2026-10-19",
  "quantity": {"value": 4000, "unit": "L", "estimate": false},
  "confidence": 0.93,
  "evidence": {"observation_id": "OBS-000123", "source_type": "chat_message", "span": "dầu 4k", "observed_at": "2026-10-19T10:12:00+07:00"}
}
```
