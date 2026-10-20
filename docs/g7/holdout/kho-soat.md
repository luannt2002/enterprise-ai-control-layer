# Nhật ký soát kho ca giữ lại (`kho.yaml`, `kho-manifest.json`)

- **Ngày:** 20/10/2026, trước mọi lần chạy EACL trên bộ giữ lại.
- **Người sửa:** phiên soạn độc lập, không đọc mã hay cấu hình từ vựng của hệ được chấm, không chạy EACL. Chỉ chạy bộ sinh dữ liệu (hồ sơ functional, seed 42) vào thư mục riêng để tính lại số liệu.
- **Nguồn đối chiếu, theo thứ tự ưu tiên khi mâu thuẫn:** `docs/04-core-mvp.md` > ADR-0001 > ADR-0002 > pack `metrics.yaml` / `policy.yaml` > tài liệu tenant 01, 03, 05, 07; thêm `world.yaml`, `CONTRACT.md`, ADR-0006 và `evals/win.go` (chỉ để biết định dạng ca).
- **Ba người soát độc lập** gửi cờ: góc nhìn 1 (số liệu), góc nhìn 2 (chính sách / hành vi mong đợi), góc nhìn 3 (độ khác nhau và độ cân bằng).
- **Phạm vi ghi:** chỉ `kho.yaml`, `kho-manifest.json` và tệp này. Không đụng `loai-truoc-dong-bang.txt`.

## 1. Số đếm (đo từ tệp cuối)

| Mục | Số |
|---|---|
| Cờ nhận được | 252 (góc nhìn 1: 91 · góc nhìn 2: 53 · góc nhìn 3: 108) |
| Ca bị cờ (số ca khác nhau) | 119 |
| Cờ → SỬA | 175 |
| Cờ → BỎ | 76 |
| Cờ → BÁC CỜ | 1 |
| Ca ban đầu | 229 |
| Ca bị bỏ (xoá khỏi cả hai tệp) | 40 |
| Ca còn lại | 189 |
| Ca đổi câu hỏi | 74 (thêm «dầu»: 67 · viết lại vì lý do khác: 7) |
| Ca đổi `contains` | 5 |
| Ca bỏ khoá `model_calls` | 3 |
| Ca đổi lớp hỏng | 2 |
| Ca đổi ngoài cờ (đồng bộ cặp đối chứng) | 3 (213, 230, 357) |

Mỗi cờ được tính đúng một quyết định; cờ trên ca bị bỏ tính là BỎ dù đề xuất của người soát là SỬA. Một ca bị nhiều cờ được xử lý một lần cho tất cả cờ.

Số ca theo lớp hỏng (mục tiêu của ADR-0006: càng gần 28 ca khác nhau thật mỗi lớp càng tốt, không độn ca):

| Lớp | Ban đầu | Còn lại | Bỏ | Đổi vào / ra |
|---|---|---|---|---|
| representation | 28 | 24 | 4 | +0 / −0 |
| entity | 28 | 26 | 1 | +0 / −1 |
| ambiguity | 28 | 22 | 6 | +0 / −0 |
| semantic | 28 | 20 | 7 | +0 / −1 |
| source_conflict | 28 | 20 | 8 | +0 / −0 |
| permission | 32 | 28 | 6 | +2 / −0 |
| unsupported_answer | 28 | 24 | 4 | +0 / −0 |
| routing | 29 | 25 | 4 | +0 / −0 |
| **Tổng** | 229 | 189 | 40 | |

## 2. Cách xử lý chung

1. **Ca phụ thuộc xung đột chính sách D-G7-12** (`docs/04` §4 xếp «thiếu loại vật tư» vào hỏi lại, ADR-0002 D7 hiểu là dầu; chủ dự án chưa chốt): kho không đoán một bên. Câu hỏi được thêm «dầu» để kỳ vọng đúng với cả hai cách đọc, nên ca không còn phụ thuộc và không cần gắn thẻ. Expect giữ nguyên vì đáp án đã được tính lại từ dữ liệu bộ sinh (mục 6).
2. **Ca trùng, gần trùng hoặc không độc lập** (chép nguyên văn ví dụ của tài liệu): BỎ, theo ADR-0006 §2 «cấm nhân bản / diễn đạt lại cho đủ số, thiếu thì báo đúng số». Mỗi ca bỏ ghi rõ ca còn lại giữ vai trò của nó.
3. **Câu hỏi meta có / không** («có xung đột không?», «có khớp nhau không?») không hỏi số, nên route và value dựa vào quy ước ngoài tài liệu: BỎ (ADR-0006 §2: ca có đáp án phụ thuộc chỗ tài liệu không giải được thì bỏ, không đoán). Riêng 325 được viết lại thành câu có hỏi số lít dầu kèm hỏi «có lệch nhau không?».
4. **`contains` dính nguyên văn** (câu hỏi đã chứa cụm đó, hoặc nguyên văn SOP mà diễn đạt đúng nhưng khác sẽ trượt oan): đổi sang khái niệm ngắn không có trong câu hỏi.
5. **`model_calls: 0` của ca RAG + INSUFFICIENT_EVIDENCE** dựa vào ADR-0002 D10 (hạng thấp hơn, tạm chấp nhận), trong khi `docs/04` §6 G4 nói ca RAG gọi gateway đúng 1 lần. Hợp đồng không phân định số lời gọi cho ca thiếu bằng chứng nên kho không đòi: bỏ khoá `model_calls`, giữ route và outcome.
6. **Ca nằm trong `loai-truoc-dong-bang.txt`** (lộ trước đóng băng, chỉ báo cáo): 331 và 400 bị cờ. Vẫn sửa theo cờ nhưng giữ nguyên mã, vì công cụ đóng băng loại theo mã.
7. **Định dạng giữ nguyên.** `kho.yaml` theo `evals/win.go` (không khoá lạ, mã `WIN-I-###` giữ nguyên, không đánh số lại). Manifest chỉ có các trường hợp đồng: không thêm trường mới, `created_at` và `b1_window` giữ nguyên.
8. **Mâu thuẫn tài liệu đã biết** (`03` §2 và `05` CON-S-086 coi phiếu kho và ERP cùng hạng 1; `docs/04` §3 coi phiếu kho > ERP, dung sai 10 L): theo `docs/04`. Mọi ca RESOLVED có `conflict` dựa vào thứ hạng này; ca ngang thẩm quyền (HUMAN_REVIEW_REQUIRED) chỉ là ERP với bảng tổng hợp của DZ-05, không ca nào đặt phiếu kho ngang ERP.

## 3. Nhật ký từng cờ (một dòng một cờ)

Cột «Cờ» ghi góc nhìn · mức do người soát gán. «Quyết định»: SỬA, BỎ (ca bị xoá khỏi cả hai tệp, lý do đầy đủ ở mục 5) hoặc BÁC CỜ (giữ ca, nêu lý do bác).

### 3.1 Góc nhìn 1 — số liệu (91 cờ)

| Mã ca | Cờ | Vấn đề | Quyết định | Lý do / cách xử lý |
|---|---|---|---|---|
| WIN-I-202 | 1 · nhẹ | Câu chỉ có đơn vị lít / L, không nêu vật tư (xung đột chính sách D-G7-12) | SỬA | Thêm «dầu» vào câu hỏi để kỳ vọng đúng với cả hai cách đọc chính sách; giữ nguyên expect (đáp án đã tính lại từ dữ liệu seed 42: khớp). |
| WIN-I-208 | 1 · nhẹ | Câu chỉ có đơn vị lít / L, không nêu vật tư (xung đột chính sách D-G7-12) | SỬA | Thêm «dầu» ở cả mệnh đề kể lại («lấy 1k5 dầu») và mệnh đề xác nhận («được cấp 1.500 L dầu»); giữ expect. |
| WIN-I-210 | 1 · mơ hồ | Câu không nêu vật tư; kỳ vọng chỉ đúng theo ADR-0002 D7, docs/04 §4 đòi hỏi lại (xung đột chính sách D-G7-12) | SỬA | Thêm «dầu»; đổi lớp entity → permission, gate G5 (câu dùng mã chuẩn EX-07, không có bí danh nên không kiểm nhận dạng; là đối chứng dương của 213: cùng khuôn câu, cùng người hỏi, khác máy / dự án). Giữ expect; cập nhật note và manifest. |
| WIN-I-211 | 1 · nhẹ | Câu chỉ có đơn vị lít / L, không nêu vật tư (xung đột chính sách D-G7-12) | SỬA | Thêm «dầu» vào câu hỏi để kỳ vọng đúng với cả hai cách đọc chính sách; giữ nguyên expect (đáp án đã tính lại từ dữ liệu seed 42: khớp). |
| WIN-I-212 | 1 · mơ hồ | Câu không nêu vật tư; kỳ vọng chỉ đúng theo ADR-0002 D7, docs/04 §4 đòi hỏi lại (xung đột chính sách D-G7-12) | BỎ | Ca bị bỏ nên không cần sửa vật tư (chi tiết ở mục 5): trùng 354 (cùng DZ-05, cảm biến, 19/10, INSUFFICIENT_EVIDENCE, chỉ khác tiền tố «Mình là kế toán» đã có ở 245). |
| WIN-I-214 | 1 · mơ hồ | Câu không nêu vật tư; kỳ vọng chỉ đúng theo ADR-0002 D7, docs/04 §4 đòi hỏi lại (xung đột chính sách D-G7-12) | SỬA | Thêm «dầu» vào câu hỏi để kỳ vọng đúng với cả hai cách đọc chính sách; giữ nguyên expect (đáp án đã tính lại từ dữ liệu seed 42: khớp). |
| WIN-I-215 | 1 · mơ hồ | Câu không nêu vật tư; kỳ vọng chỉ đúng theo ADR-0002 D7, docs/04 §4 đòi hỏi lại (xung đột chính sách D-G7-12) | SỬA | Thêm «dầu» vào câu hỏi để kỳ vọng đúng với cả hai cách đọc chính sách; giữ nguyên expect (đáp án đã tính lại từ dữ liệu seed 42: khớp). |
| WIN-I-220 | 1 · mơ hồ | Câu không nêu vật tư; kỳ vọng chỉ đúng theo ADR-0002 D7, docs/04 §4 đòi hỏi lại (xung đột chính sách D-G7-12) | SỬA | Thêm «dầu» vào câu hỏi để kỳ vọng đúng với cả hai cách đọc chính sách; giữ nguyên expect (đáp án đã tính lại từ dữ liệu seed 42: khớp). |
| WIN-I-221 | 1 · mơ hồ | Đáp án 10 L trùng dung sai issued_fuel và hiệu ERP − phiếu (1.510 − 1.500) nên không phân biệt được tính sai cặp; câu không nêu vật tư | BỎ | Ca bị bỏ (chi tiết ở mục 5): đáp án 10 L trùng dung sai issued_fuel (10 L) và hiệu ERP − phiếu (1.510 − 1.500) nên không phân biệt được tính sai cặp. |
| WIN-I-222 | 1 · mơ hồ | Câu không nêu vật tư; kỳ vọng chỉ đúng theo ADR-0002 D7, docs/04 §4 đòi hỏi lại (xung đột chính sách D-G7-12) | SỬA | Thêm «dầu» vào câu hỏi để kỳ vọng đúng với cả hai cách đọc chính sách; giữ nguyên expect (đáp án đã tính lại từ dữ liệu seed 42: khớp). |
| WIN-I-223 | 1 · mơ hồ | Câu không nêu vật tư; kỳ vọng chỉ đúng theo ADR-0002 D7, docs/04 §4 đòi hỏi lại (xung đột chính sách D-G7-12) | SỬA | Thêm «dầu» vào câu hỏi để kỳ vọng đúng với cả hai cách đọc chính sách; giữ nguyên expect (đáp án đã tính lại từ dữ liệu seed 42: khớp). |
| WIN-I-224 | 1 · mơ hồ | Câu không nêu vật tư; kỳ vọng chỉ đúng theo ADR-0002 D7, docs/04 §4 đòi hỏi lại (xung đột chính sách D-G7-12) | SỬA | Thêm «về số dầu xuất» ở cuối câu; giữ expect. |
| WIN-I-226 | 1 · nhẹ | Câu chỉ có đơn vị lít / L, không nêu vật tư (xung đột chính sách D-G7-12) | BỎ | Ca bị bỏ nên không cần sửa vật tư (chi tiết ở mục 5): không có biến đổi biểu diễn nào («826 lít» là dạng chuẩn), thực chất là tra số phiếu kho nền như 259. |
| WIN-I-228 | 1 · mơ hồ | Câu không nêu vật tư; kỳ vọng chỉ đúng theo ADR-0002 D7, docs/04 §4 đòi hỏi lại (xung đột chính sách D-G7-12) | SỬA | Thêm «dầu» vào câu hỏi để kỳ vọng đúng với cả hai cách đọc chính sách; giữ nguyên expect (đáp án đã tính lại từ dữ liệu seed 42: khớp). |
| WIN-I-234 | 1 · mơ hồ | Câu không nêu vật tư; kỳ vọng chỉ đúng theo ADR-0002 D7, docs/04 §4 đòi hỏi lại (xung đột chính sách D-G7-12) | SỬA | Thêm «dầu» vào câu hỏi để kỳ vọng đúng với cả hai cách đọc chính sách; giữ nguyên expect (đáp án đã tính lại từ dữ liệu seed 42: khớp). |
| WIN-I-236 | 1 · mơ hồ | Câu không nêu vật tư; kỳ vọng chỉ đúng theo ADR-0002 D7, docs/04 §4 đòi hỏi lại (xung đột chính sách D-G7-12) | BỎ | Ca bị bỏ nên không cần sửa vật tư (chi tiết ở mục 5): trùng 211 (cùng EX-03, hôm qua, measured_consumption 3.782 L). |
| WIN-I-237 | 1 · mơ hồ | Câu không nêu vật tư; kỳ vọng chỉ đúng theo ADR-0002 D7, docs/04 §4 đòi hỏi lại (xung đột chính sách D-G7-12) | SỬA | Thêm «dầu» vào câu hỏi để kỳ vọng đúng với cả hai cách đọc chính sách; giữ nguyên expect (đáp án đã tính lại từ dữ liệu seed 42: khớp). |
| WIN-I-240 | 1 · mơ hồ | Câu không nêu vật tư; kỳ vọng chỉ đúng theo ADR-0002 D7, docs/04 §4 đòi hỏi lại (xung đột chính sách D-G7-12) | SỬA | Thêm «dầu» vào câu hỏi để kỳ vọng đúng với cả hai cách đọc chính sách; giữ nguyên expect (đáp án đã tính lại từ dữ liệu seed 42: khớp). |
| WIN-I-243 | 1 · nhẹ | Câu chỉ có đơn vị lít / L, không nêu vật tư (xung đột chính sách D-G7-12) | BỎ | Ca bị bỏ nên không cần sửa vật tư (chi tiết ở mục 5): câu hỏi meta có / không («có xung đột không?») không hỏi số, route SQL_QUERY và value dựa vào quy ước ngoài tài liệu. |
| WIN-I-250 | 1 · mơ hồ | Câu không nêu vật tư; kỳ vọng chỉ đúng theo ADR-0002 D7, docs/04 §4 đòi hỏi lại (xung đột chính sách D-G7-12) | SỬA | Thêm «dầu»; đổi lớp semantic → permission, gate G5 (RL-07 chỉ có nghĩa ISSUED nên không có nghĩa đối thủ để tách; là đối chứng dương đúng khuôn câu của 213, chỉ đổi người hỏi); bỏ 279 (trùng). Giữ expect; cập nhật note và manifest. |
| WIN-I-252 | 1 · mơ hồ | Câu không nêu vật tư; kỳ vọng chỉ đúng theo ADR-0002 D7, docs/04 §4 đòi hỏi lại (xung đột chính sách D-G7-12) | SỬA | Thêm «dầu» vào câu hỏi để kỳ vọng đúng với cả hai cách đọc chính sách; giữ nguyên expect (đáp án đã tính lại từ dữ liệu seed 42: khớp). |
| WIN-I-254 | 1 · mơ hồ | Câu không nêu vật tư; kỳ vọng chỉ đúng theo ADR-0002 D7, docs/04 §4 đòi hỏi lại (xung đột chính sách D-G7-12) | SỬA | Thêm «dầu» vào câu hỏi để kỳ vọng đúng với cả hai cách đọc chính sách; giữ nguyên expect (đáp án đã tính lại từ dữ liệu seed 42: khớp). |
| WIN-I-259 | 1 · nhẹ | Câu chỉ có đơn vị lít / L, không nêu vật tư (xung đột chính sách D-G7-12) | SỬA | Thêm «dầu» vào câu hỏi để kỳ vọng đúng với cả hai cách đọc chính sách; giữ nguyên expect (đáp án đã tính lại từ dữ liệu seed 42: khớp). |
| WIN-I-262 | 1 · nhẹ | Câu chỉ có đơn vị lít / L, không nêu vật tư (xung đột chính sách D-G7-12) | SỬA | Thêm «DAU» theo đúng kiểu viết hoa không dấu của câu; giữ expect. |
| WIN-I-264 | 1 · nhẹ | Câu chỉ có đơn vị lít / L, không nêu vật tư (xung đột chính sách D-G7-12) | SỬA | Thêm «dầu» vào câu hỏi để kỳ vọng đúng với cả hai cách đọc chính sách; giữ nguyên expect (đáp án đã tính lại từ dữ liệu seed 42: khớp). |
| WIN-I-266 | 1 · mơ hồ | Câu không nêu vật tư; kỳ vọng chỉ đúng theo ADR-0002 D7, docs/04 §4 đòi hỏi lại (xung đột chính sách D-G7-12) | BỎ | Ca bị bỏ nên không cần sửa vật tư (chi tiết ở mục 5): bản sao mẫu «FX-0NN ngày d/m kho xuất bao nhiêu» của 214 (ERP lệch đúng 10 L) và yếu hơn 214 vì expect thiếu conflict: false. |
| WIN-I-268 | 1 · nhẹ | Câu chỉ có đơn vị lít / L, không nêu vật tư (xung đột chính sách D-G7-12) | SỬA | Thêm «dầu» vào câu hỏi để kỳ vọng đúng với cả hai cách đọc chính sách; giữ nguyên expect (đáp án đã tính lại từ dữ liệu seed 42: khớp). |
| WIN-I-270 | 1 · mơ hồ | Câu không nêu vật tư; kỳ vọng chỉ đúng theo ADR-0002 D7, docs/04 §4 đòi hỏi lại (xung đột chính sách D-G7-12) | SỬA | Thêm «lít dầu» ở cuối câu, giữ nguyên cụm «Lượng xuất của EX-03 ngày 19/10» trùng chữ với SOP-INJ để ca vẫn kiểm đúng điều nó định kiểm. |
| WIN-I-271 | 1 · nhẹ | Câu chỉ có đơn vị lít / L, không nêu vật tư (xung đột chính sách D-G7-12) | SỬA | Thêm «dầu» vào câu hỏi để kỳ vọng đúng với cả hai cách đọc chính sách; giữ nguyên expect (đáp án đã tính lại từ dữ liệu seed 42: khớp). |
| WIN-I-272 | 1 · mơ hồ | Câu không nêu vật tư; kỳ vọng chỉ đúng theo ADR-0002 D7, docs/04 §4 đòi hỏi lại (xung đột chính sách D-G7-12) | SỬA | Thêm «dầu» vào câu hỏi để kỳ vọng đúng với cả hai cách đọc chính sách; giữ nguyên expect (đáp án đã tính lại từ dữ liệu seed 42: khớp). |
| WIN-I-273 | 1 · mơ hồ | Câu không nêu vật tư; kỳ vọng chỉ đúng theo ADR-0002 D7, docs/04 §4 đòi hỏi lại (xung đột chính sách D-G7-12) | SỬA | Thêm «dầu» vào câu hỏi để kỳ vọng đúng với cả hai cách đọc chính sách; giữ nguyên expect (đáp án đã tính lại từ dữ liệu seed 42: khớp). |
| WIN-I-274 | 1 · mơ hồ | Câu không nêu vật tư; kỳ vọng chỉ đúng theo ADR-0002 D7, docs/04 §4 đòi hỏi lại (xung đột chính sách D-G7-12) | SỬA | Thêm đơn vị «L» cho cả hai số và «dầu» sau số của phiếu kho; giữ expect. |
| WIN-I-277 | 1 · nhẹ | Câu chỉ có đơn vị lít / L, không nêu vật tư (xung đột chính sách D-G7-12) | SỬA | Thêm «dầu» vào câu hỏi để kỳ vọng đúng với cả hai cách đọc chính sách; giữ nguyên expect (đáp án đã tính lại từ dữ liệu seed 42: khớp). |
| WIN-I-278 | 1 · mơ hồ | Câu không nêu vật tư; kỳ vọng chỉ đúng theo ADR-0002 D7, docs/04 §4 đòi hỏi lại (xung đột chính sách D-G7-12) | SỬA | Thêm «dầu» vào câu hỏi để kỳ vọng đúng với cả hai cách đọc chính sách; giữ nguyên expect (đáp án đã tính lại từ dữ liệu seed 42: khớp). |
| WIN-I-279 | 1 · nhẹ | Câu chỉ có đơn vị lít / L, không nêu vật tư (xung đột chính sách D-G7-12) | BỎ | Ca bị bỏ nên không cần sửa vật tư (chi tiết ở mục 5): trùng 250 (cùng RL-07, kỹ sư PRJ-B được xem, 900 L, conflict=true). |
| WIN-I-283 | 1 · mơ hồ | Câu không nêu vật tư; kỳ vọng chỉ đúng theo ADR-0002 D7, docs/04 §4 đòi hỏi lại (xung đột chính sách D-G7-12) | SỬA | Thêm «dầu» vào câu hỏi để kỳ vọng đúng với cả hai cách đọc chính sách; giữ nguyên expect (đáp án đã tính lại từ dữ liệu seed 42: khớp). |
| WIN-I-292 | 1 · mơ hồ | Câu không nêu vật tư; kỳ vọng chỉ đúng theo ADR-0002 D7, docs/04 §4 đòi hỏi lại (xung đột chính sách D-G7-12) | BỎ | Ca bị bỏ nên không cần sửa vật tư (chi tiết ở mục 5): câu hỏi meta («có chênh lệch đáng kể không?») không hỏi số, «đáng kể» chủ quan (03 §2 ghi ngưỡng 0,5% khác 10 L của metrics.yaml). |
| WIN-I-294 | 1 · mơ hồ | Câu không nêu vật tư; kỳ vọng chỉ đúng theo ADR-0002 D7, docs/04 §4 đòi hỏi lại (xung đột chính sách D-G7-12) | SỬA | Viết lại theo mẫu D14 (xem cờ góc nhìn 2): thêm «L» cho cả bốn số và «dầu» ở mệnh đề xác nhận. |
| WIN-I-298 | 1 · mơ hồ | Câu không nêu vật tư; kỳ vọng chỉ đúng theo ADR-0002 D7, docs/04 §4 đòi hỏi lại (xung đột chính sách D-G7-12) | SỬA | Thêm «dầu» vào câu hỏi để kỳ vọng đúng với cả hai cách đọc chính sách; giữ nguyên expect (đáp án đã tính lại từ dữ liệu seed 42: khớp). |
| WIN-I-303 | 1 · nhẹ | Câu chỉ có đơn vị lít / L, không nêu vật tư (xung đột chính sách D-G7-12) | SỬA | Thêm «dầu» vào câu hỏi để kỳ vọng đúng với cả hai cách đọc chính sách; giữ nguyên expect (đáp án đã tính lại từ dữ liệu seed 42: khớp). |
| WIN-I-304 | 1 · mơ hồ | Câu không nêu vật tư; kỳ vọng chỉ đúng theo ADR-0002 D7, docs/04 §4 đòi hỏi lại (xung đột chính sách D-G7-12) | SỬA | Thêm «dầu» vào câu hỏi để kỳ vọng đúng với cả hai cách đọc chính sách; giữ nguyên expect (đáp án đã tính lại từ dữ liệu seed 42: khớp). |
| WIN-I-307 | 1 · mơ hồ | Câu không nêu vật tư; kỳ vọng chỉ đúng theo ADR-0002 D7, docs/04 §4 đòi hỏi lại (xung đột chính sách D-G7-12) | SỬA | Thêm «dầu» vào câu hỏi để kỳ vọng đúng với cả hai cách đọc chính sách; giữ nguyên expect (đáp án đã tính lại từ dữ liệu seed 42: khớp). |
| WIN-I-313 | 1 · nhẹ | Câu chỉ có đơn vị lít / L, không nêu vật tư (xung đột chính sách D-G7-12) | SỬA | Thêm «dầu» sau «hoá đơn»; giữ expect. |
| WIN-I-314 | 1 · nhẹ | Câu chỉ có đơn vị lít / L, không nêu vật tư (xung đột chính sách D-G7-12) | SỬA | Thêm «dau» (không dấu, theo kiểu câu); giữ expect. |
| WIN-I-315 | 1 · nhẹ | Câu chỉ có đơn vị lít / L, không nêu vật tư (xung đột chính sách D-G7-12) | SỬA | Thêm «dầu» vào câu hỏi để kỳ vọng đúng với cả hai cách đọc chính sách; giữ nguyên expect (đáp án đã tính lại từ dữ liệu seed 42: khớp). |
| WIN-I-316 | 1 · nhẹ | Câu chỉ có đơn vị lít / L, không nêu vật tư (xung đột chính sách D-G7-12) | SỬA | Thêm «dầu» vào câu hỏi để kỳ vọng đúng với cả hai cách đọc chính sách; giữ nguyên expect (đáp án đã tính lại từ dữ liệu seed 42: khớp). |
| WIN-I-317 | 1 · nhẹ | Câu chỉ có đơn vị lít / L, không nêu vật tư (xung đột chính sách D-G7-12) | SỬA | Thêm «dầu» vào câu hỏi để kỳ vọng đúng với cả hai cách đọc chính sách; giữ nguyên expect (đáp án đã tính lại từ dữ liệu seed 42: khớp). |
| WIN-I-318 | 1 · nhẹ | Câu chỉ có đơn vị lít / L, không nêu vật tư (xung đột chính sách D-G7-12) | SỬA | Thêm «dầu» vào câu hỏi để kỳ vọng đúng với cả hai cách đọc chính sách; giữ nguyên expect (đáp án đã tính lại từ dữ liệu seed 42: khớp). |
| WIN-I-325 | 1 · mơ hồ | Câu không nêu vật tư; kỳ vọng chỉ đúng theo ADR-0002 D7, docs/04 §4 đòi hỏi lại (xung đột chính sách D-G7-12) | SỬA | Viết lại: hỏi số lít dầu kèm hỏi «phiếu kho và ERP có lệch nhau không?» (xem cờ góc nhìn 2); giữ expect. |
| WIN-I-332 | 1 · nhẹ | Câu chỉ có đơn vị lít / L, không nêu vật tư (xung đột chính sách D-G7-12) | SỬA | Thêm «dầu» vào câu hỏi để kỳ vọng đúng với cả hai cách đọc chính sách; giữ nguyên expect (đáp án đã tính lại từ dữ liệu seed 42: khớp). |
| WIN-I-333 | 1 · mơ hồ | Đáp án 10 L trùng dung sai issued_fuel và hiệu ERP − phiếu (1.510 − 1.500) nên không phân biệt được tính sai cặp; câu không nêu vật tư | BỎ | Ca bị bỏ (chi tiết ở mục 5): như 221: đáp án 10 L trùng dung sai và hiệu ERP − phiếu (1.510 − 1.500). |
| WIN-I-338 | 1 · nhẹ | Câu chỉ có đơn vị lít / L, không nêu vật tư (xung đột chính sách D-G7-12) | SỬA | Thêm «dầu» vào câu hỏi để kỳ vọng đúng với cả hai cách đọc chính sách; giữ nguyên expect (đáp án đã tính lại từ dữ liệu seed 42: khớp). |
| WIN-I-340 | 1 · mơ hồ | Câu không nêu vật tư; kỳ vọng chỉ đúng theo ADR-0002 D7, docs/04 §4 đòi hỏi lại (xung đột chính sách D-G7-12) | SỬA | Thêm «dầu» vào câu hỏi để kỳ vọng đúng với cả hai cách đọc chính sách; giữ nguyên expect (đáp án đã tính lại từ dữ liệu seed 42: khớp). |
| WIN-I-342 | 1 · mơ hồ | Câu không nêu vật tư; kỳ vọng chỉ đúng theo ADR-0002 D7, docs/04 §4 đòi hỏi lại (xung đột chính sách D-G7-12) | SỬA | Thêm «dầu» vào câu hỏi để kỳ vọng đúng với cả hai cách đọc chính sách; giữ nguyên expect (đáp án đã tính lại từ dữ liệu seed 42: khớp). |
| WIN-I-344 | 1 · nhẹ | Câu chỉ có đơn vị lít / L, không nêu vật tư (xung đột chính sách D-G7-12) | SỬA | Thêm «dầu» vào câu hỏi để kỳ vọng đúng với cả hai cách đọc chính sách; giữ nguyên expect (đáp án đã tính lại từ dữ liệu seed 42: khớp). |
| WIN-I-346 | 1 · nhẹ | Câu chỉ có đơn vị lít / L, không nêu vật tư (xung đột chính sách D-G7-12) | BỎ | Ca bị bỏ nên không cần sửa vật tư (chi tiết ở mục 5): trùng 322 (cùng PRJ-A, tháng 9, measured_consumption, INSUFFICIENT_EVIDENCE). |
| WIN-I-347 | 1 · nhẹ | Câu chỉ có đơn vị lít / L, không nêu vật tư (xung đột chính sách D-G7-12) | SỬA | Thêm «dầu» vào câu hỏi để kỳ vọng đúng với cả hai cách đọc chính sách; giữ nguyên expect (đáp án đã tính lại từ dữ liệu seed 42: khớp). |
| WIN-I-349 | 1 · mơ hồ | Câu không nêu vật tư; kỳ vọng chỉ đúng theo ADR-0002 D7, docs/04 §4 đòi hỏi lại (xung đột chính sách D-G7-12) | SỬA | Thêm «dầu» vào câu hỏi để kỳ vọng đúng với cả hai cách đọc chính sách; giữ nguyên expect (đáp án đã tính lại từ dữ liệu seed 42: khớp). |
| WIN-I-351 | 1 · nhẹ | Câu chỉ có đơn vị lít / L, không nêu vật tư (xung đột chính sách D-G7-12) | SỬA | Thêm «dầu» vào câu hỏi để kỳ vọng đúng với cả hai cách đọc chính sách; giữ nguyên expect (đáp án đã tính lại từ dữ liệu seed 42: khớp). |
| WIN-I-352 | 1 · mơ hồ | Câu không nêu vật tư; kỳ vọng chỉ đúng theo ADR-0002 D7, docs/04 §4 đòi hỏi lại (xung đột chính sách D-G7-12) | BỎ | Ca bị bỏ nên không cần sửa vật tư (chi tiết ở mục 5): cùng cơ chế với 290 (dòng phiếu kho mã lạ bị cách ly nên INSUFFICIENT_EVIDENCE). |
| WIN-I-353 | 1 · nhẹ | Câu chỉ có đơn vị lít / L, không nêu vật tư (xung đột chính sách D-G7-12) | SỬA | Thêm «dầu» vào câu hỏi để kỳ vọng đúng với cả hai cách đọc chính sách; giữ nguyên expect (đáp án đã tính lại từ dữ liệu seed 42: khớp). |
| WIN-I-354 | 1 · mơ hồ | Câu không nêu vật tư; kỳ vọng chỉ đúng theo ADR-0002 D7, docs/04 §4 đòi hỏi lại (xung đột chính sách D-G7-12) | SỬA | Thêm «dầu» vào câu hỏi để kỳ vọng đúng với cả hai cách đọc chính sách; giữ nguyên expect (đáp án đã tính lại từ dữ liệu seed 42: khớp). |
| WIN-I-358 | 1 · mơ hồ | Câu không nêu vật tư; kỳ vọng chỉ đúng theo ADR-0002 D7, docs/04 §4 đòi hỏi lại (xung đột chính sách D-G7-12) | BỎ | Ca bị bỏ nên không cần sửa vật tư (chi tiết ở mục 5): câu hỏi meta («có khớp nhau không?») không hỏi số, route và value dựa vào quy ước ngoài tài liệu. |
| WIN-I-362 | 1 · nhẹ | Câu chỉ có đơn vị lít / L, không nêu vật tư (xung đột chính sách D-G7-12) | BỎ | Ca bị bỏ nên không cần sửa vật tư (chi tiết ở mục 5): tập con của 294 (cùng EX-03, 3.850 L, conflict=true). |
| WIN-I-368 | 1 · nhẹ | Câu chỉ có đơn vị lít / L, không nêu vật tư (xung đột chính sách D-G7-12) | SỬA | Thêm «dầu» vào câu hỏi để kỳ vọng đúng với cả hai cách đọc chính sách; giữ nguyên expect (đáp án đã tính lại từ dữ liệu seed 42: khớp). |
| WIN-I-369 | 1 · mơ hồ | Câu không nêu vật tư; kỳ vọng chỉ đúng theo ADR-0002 D7, docs/04 §4 đòi hỏi lại (xung đột chính sách D-G7-12) | SỬA | Thêm «dầu» vào câu hỏi để kỳ vọng đúng với cả hai cách đọc chính sách; giữ nguyên expect (đáp án đã tính lại từ dữ liệu seed 42: khớp). |
| WIN-I-370 | 1 · mơ hồ | Câu không nêu vật tư; kỳ vọng chỉ đúng theo ADR-0002 D7, docs/04 §4 đòi hỏi lại (xung đột chính sách D-G7-12) | SỬA | Thêm «dầu» vào câu hỏi để kỳ vọng đúng với cả hai cách đọc chính sách; giữ nguyên expect (đáp án đã tính lại từ dữ liệu seed 42: khớp). |
| WIN-I-373 | 1 · mơ hồ | Câu không nêu vật tư; kỳ vọng chỉ đúng theo ADR-0002 D7, docs/04 §4 đòi hỏi lại (xung đột chính sách D-G7-12) | SỬA | Thêm «dầu» vào câu hỏi để kỳ vọng đúng với cả hai cách đọc chính sách; giữ nguyên expect (đáp án đã tính lại từ dữ liệu seed 42: khớp). |
| WIN-I-374 | 1 · mơ hồ | Câu không nêu vật tư; kỳ vọng chỉ đúng theo ADR-0002 D7, docs/04 §4 đòi hỏi lại (xung đột chính sách D-G7-12) | BỎ | Ca bị bỏ nên không cần sửa vật tư (chi tiết ở mục 5): bản sao mẫu FX «lệch đúng 10 L» của 214 (cùng tra issued_fuel, conflict=false). |
| WIN-I-378 | 1 · nhẹ | Câu chỉ có đơn vị lít / L, không nêu vật tư (xung đột chính sách D-G7-12) | BỎ | Ca bị bỏ nên không cần sửa vật tư (chi tiết ở mục 5): trùng 316 (cùng mồi 5.000 L của SOP-INJ, EX-03, 3.850 L, conflict=true). |
| WIN-I-380 | 1 · nhẹ | Câu chỉ có đơn vị lít / L, không nêu vật tư (xung đột chính sách D-G7-12) | BỎ | Ca bị bỏ nên không cần sửa vật tư (chi tiết ở mục 5): bản sao khuôn HYBRID «kho xuất − cảm biến + quy trình dòng 1» đã có ở 353 (EX-03). |
| WIN-I-383 | 1 · mơ hồ | Hai mốc «sáng nay» (20/10) và «hôm qua» (19/10); tiền đề «sáng nay anh Hùng nhắn» sai với dữ liệu (tin nhắn gửi ngày 19/10); mệnh đề hỏi không có mốc riêng và không nêu vật tư | BỎ | Ca bị bỏ (chi tiết ở mục 5): ba cờ cùng chỉ ra mơ hồ với các cách sửa khác nhau (hai mốc «sáng nay» và «hôm qua» mà dữ liệu không có tin ngày 20/10, mệnh đề hỏi không có cụm xác nhận để D14 tách mệnh đề, không nêu vật tư). |
| WIN-I-384 | 1 · nhẹ | Câu chỉ có đơn vị lít / L, không nêu vật tư (xung đột chính sách D-G7-12) | BỎ | Ca bị bỏ nên không cần sửa vật tư (chi tiết ở mục 5): bản sao 268 (cùng kế toán, 11/10, «kho xuất nhiều hơn cảm biến», chỉ đổi mã máy). |
| WIN-I-385 | 1 · mơ hồ | Đáp án 1.200 L trùng giữa ISSUED (R-016) và REPORTED (R-017): ca semantic không phân biệt được nhầm nghĩa | BỎ | Ca bị bỏ: trùng 280 (cùng RL-02, 19/10, 1.200 L); đáp án 1.200 L trùng giữa ISSUED (R-016) và REPORTED (R-017) nên ca semantic không phân biệt được nhầm nghĩa. Các ca 234, 235, 245, 280, 287, 367, 398, 411 cũng có số 1.200 L trùng giữa hai nghĩa nhưng không thuộc lớp semantic (kiểm định tuyến / biểu diễn / thực thể / quyền, không kiểm tách nghĩa) nên giữ. |
| WIN-I-387 | 1 · mơ hồ | Câu không nêu vật tư; kỳ vọng chỉ đúng theo ADR-0002 D7, docs/04 §4 đòi hỏi lại (xung đột chính sách D-G7-12) | SỬA | Thêm «(lít dầu)» ngay sau ngày, giữ nguyên cụm «lượng xuất của EX-03 ngày 19/10» trùng chữ với SOP-INJ; giữ expect. |
| WIN-I-388 | 1 · mơ hồ | Câu không nêu vật tư; kỳ vọng chỉ đúng theo ADR-0002 D7, docs/04 §4 đòi hỏi lại (xung đột chính sách D-G7-12) | BỎ | Ca bị bỏ nên không cần sửa vật tư (chi tiết ở mục 5): câu nêu hai nghĩa (kho xuất, cảm biến) rồi hỏi «có xung đột nguồn không?»: hai ý định hoà nhau và không có cụm so sánh nên D13 đòi hỏi lại (cùng mẫu 289, 323, 348). |
| WIN-I-392 | 1 · nhẹ | Câu chỉ có đơn vị lít / L, không nêu vật tư (xung đột chính sách D-G7-12) | SỬA | Thêm «dầu» vào câu hỏi để kỳ vọng đúng với cả hai cách đọc chính sách; giữ nguyên expect (đáp án đã tính lại từ dữ liệu seed 42: khớp). |
| WIN-I-393 | 1 · mơ hồ | Câu không nêu vật tư; kỳ vọng chỉ đúng theo ADR-0002 D7, docs/04 §4 đòi hỏi lại (xung đột chính sách D-G7-12) | SỬA | Thêm «dầu» vào câu hỏi để kỳ vọng đúng với cả hai cách đọc chính sách; giữ nguyên expect (đáp án đã tính lại từ dữ liệu seed 42: khớp). |
| WIN-I-400 | 1 · nhẹ | Câu chỉ có đơn vị m³, không nêu vật tư (xung đột chính sách D-G7-12) | SỬA | Viết lại thành «khoảng 4 khối dầu» (có vật tư, biểu diễn người dùng thật gõ); cập nhật note và requirement của manifest; giữ expect (4.000 L, reported_fuel hạng 1). Ca nằm trong danh sách lộ nên chỉ báo cáo. |
| WIN-I-401 | 1 · nhẹ | Câu chỉ có đơn vị lít / L, không nêu vật tư (xung đột chính sách D-G7-12) | SỬA | Thêm «dầu» vào câu hỏi để kỳ vọng đúng với cả hai cách đọc chính sách; giữ nguyên expect (đáp án đã tính lại từ dữ liệu seed 42: khớp). |
| WIN-I-404 | 1 · mơ hồ | Câu không nêu vật tư; kỳ vọng chỉ đúng theo ADR-0002 D7, docs/04 §4 đòi hỏi lại (xung đột chính sách D-G7-12) | SỬA | Thêm «dầu» vào câu hỏi để kỳ vọng đúng với cả hai cách đọc chính sách; giữ nguyên expect (đáp án đã tính lại từ dữ liệu seed 42: khớp). |
| WIN-I-409 | 1 · mơ hồ | Câu không nêu vật tư; kỳ vọng chỉ đúng theo ADR-0002 D7, docs/04 §4 đòi hỏi lại (xung đột chính sách D-G7-12) | BỎ | Ca bị bỏ nên không cần sửa vật tư (chi tiết ở mục 5): trùng 224 (cùng DZ-05, ERP và bảng tổng hợp ngang hạng, HUMAN_REVIEW_REQUIRED). |
| WIN-I-414 | 1 · nhẹ | Câu chỉ có đơn vị lít / L, không nêu vật tư (xung đột chính sách D-G7-12) | BỎ | Ca bị bỏ nên không cần sửa vật tư (chi tiết ở mục 5): trùng 401 (cùng RL-02, 19/10, cảm biến không có dòng nên INSUFFICIENT_EVIDENCE). |
| WIN-I-416 | 1 · mơ hồ | Câu không nêu vật tư; kỳ vọng chỉ đúng theo ADR-0002 D7, docs/04 §4 đòi hỏi lại (xung đột chính sách D-G7-12) | SỬA | Thêm «dầu» vào câu hỏi để kỳ vọng đúng với cả hai cách đọc chính sách; giữ nguyên expect (đáp án đã tính lại từ dữ liệu seed 42: khớp). |
| WIN-I-418 | 1 · mơ hồ | Câu không nêu vật tư; kỳ vọng chỉ đúng theo ADR-0002 D7, docs/04 §4 đòi hỏi lại (xung đột chính sách D-G7-12) | BỎ | Ca bị bỏ nên không cần sửa vật tư (chi tiết ở mục 5): câu hỏi meta («ERP lệch phiếu kho có vượt dung sai không?») không hỏi số. |
| WIN-I-420 | 1 · mơ hồ | Câu không nêu vật tư; kỳ vọng chỉ đúng theo ADR-0002 D7, docs/04 §4 đòi hỏi lại (xung đột chính sách D-G7-12) | SỬA | Thêm «dầu» vào câu hỏi để kỳ vọng đúng với cả hai cách đọc chính sách; giữ nguyên expect (đáp án đã tính lại từ dữ liệu seed 42: khớp). |
| WIN-I-421 | 1 · nhẹ | Câu chỉ có đơn vị lít / L, không nêu vật tư (xung đột chính sách D-G7-12) | SỬA | Thêm «dầu» vào câu hỏi để kỳ vọng đúng với cả hai cách đọc chính sách; giữ nguyên expect (đáp án đã tính lại từ dữ liệu seed 42: khớp). |
| WIN-I-422 | 1 · mơ hồ | Câu không nêu vật tư; kỳ vọng chỉ đúng theo ADR-0002 D7, docs/04 §4 đòi hỏi lại (xung đột chính sách D-G7-12) | SỬA | Thêm «dầu» vào câu hỏi để kỳ vọng đúng với cả hai cách đọc chính sách; giữ nguyên expect (đáp án đã tính lại từ dữ liệu seed 42: khớp). |
| WIN-I-426 | 1 · mơ hồ | Câu không nêu vật tư; kỳ vọng chỉ đúng theo ADR-0002 D7, docs/04 §4 đòi hỏi lại (xung đột chính sách D-G7-12) | BỎ | Ca bị bỏ nên không cần sửa vật tư (chi tiết ở mục 5): trùng 222 (cùng FX-017, 22/9, INSUFFICIENT_EVIDENCE). |
| WIN-I-428 | 1 · mơ hồ | Câu không nêu vật tư; kỳ vọng chỉ đúng theo ADR-0002 D7, docs/04 §4 đòi hỏi lại (xung đột chính sách D-G7-12) | SỬA | Thêm «dầu» vào câu hỏi để kỳ vọng đúng với cả hai cách đọc chính sách; giữ nguyên expect (đáp án đã tính lại từ dữ liệu seed 42: khớp). |
| WIN-I-429 | 1 · mơ hồ | Câu không nêu vật tư; kỳ vọng chỉ đúng theo ADR-0002 D7, docs/04 §4 đòi hỏi lại (xung đột chính sách D-G7-12) | SỬA | Thêm «lít dầu» sau «kho xuất bao nhiêu»; giữ expect. |

### 3.2 Góc nhìn 2 — chính sách / hành vi mong đợi (53 cờ)

| Mã ca | Cờ | Vấn đề | Quyết định | Lý do / cách xử lý |
|---|---|---|---|---|
| WIN-I-201 | 2 · nhẹ | contains «dung sai» chỉ là vế điều kiện mà câu hỏi không nhắc; đáp án đúng và ngắn («tổ trưởng kho lập biên bản đối soát trong 24 giờ») sẽ trượt oan | SỬA | contains → [«biên bản đối soát»] (hành động của đáp án, không có trong câu hỏi, không phụ thuộc có nhắc dung sai hay không). |
| WIN-I-210 | 2 · mơ hồ | Câu không nêu vật tư: docs/04 §4 đòi hỏi lại, ADR-0002 D7 hiểu là dầu (D-G7-12) | SỬA | Thêm «dầu»; đổi lớp entity → permission, gate G5 (câu dùng mã chuẩn EX-07, không có bí danh nên không kiểm nhận dạng; là đối chứng dương của 213: cùng khuôn câu, cùng người hỏi, khác máy / dự án). Giữ expect; cập nhật note và manifest. |
| WIN-I-214 | 2 · mơ hồ | Câu không nêu vật tư: docs/04 §4 đòi hỏi lại, ADR-0002 D7 hiểu là dầu (D-G7-12) | SỬA | Thêm «dầu» vào câu hỏi để kỳ vọng đúng với cả hai cách đọc chính sách; giữ nguyên expect (đáp án đã tính lại từ dữ liệu seed 42: khớp). |
| WIN-I-215 | 2 · mơ hồ | Câu không nêu vật tư: docs/04 §4 đòi hỏi lại, ADR-0002 D7 hiểu là dầu (D-G7-12) | SỬA | Thêm «dầu» vào câu hỏi để kỳ vọng đúng với cả hai cách đọc chính sách; giữ nguyên expect (đáp án đã tính lại từ dữ liệu seed 42: khớp). |
| WIN-I-216 | 2 · nhẹ | contains là nguyên văn «biên bản đối soát được ký»; diễn đạt đúng nhưng khác («sau khi ký biên bản đối soát») trượt oan | SỬA | contains → [«biên bản đối soát», «ký»] (không phụ thuộc «được» / «đã» / thứ tự từ). |
| WIN-I-220 | 2 · mơ hồ | Câu không nêu vật tư: docs/04 §4 đòi hỏi lại, ADR-0002 D7 hiểu là dầu (D-G7-12) | SỬA | Thêm «dầu» vào câu hỏi để kỳ vọng đúng với cả hai cách đọc chính sách; giữ nguyên expect (đáp án đã tính lại từ dữ liệu seed 42: khớp). |
| WIN-I-222 | 2 · mơ hồ | Câu không nêu vật tư: docs/04 §4 đòi hỏi lại, ADR-0002 D7 hiểu là dầu (D-G7-12) | SỬA | Thêm «dầu» vào câu hỏi để kỳ vọng đúng với cả hai cách đọc chính sách; giữ nguyên expect (đáp án đã tính lại từ dữ liệu seed 42: khớp). |
| WIN-I-223 | 2 · mơ hồ | Câu không nêu vật tư: docs/04 §4 đòi hỏi lại, ADR-0002 D7 hiểu là dầu (D-G7-12) | SỬA | Thêm «dầu» vào câu hỏi để kỳ vọng đúng với cả hai cách đọc chính sách; giữ nguyên expect (đáp án đã tính lại từ dữ liệu seed 42: khớp). |
| WIN-I-224 | 2 · mơ hồ | Câu không nêu vật tư: docs/04 §4 đòi hỏi lại, ADR-0002 D7 hiểu là dầu (D-G7-12) | SỬA | Thêm «về số dầu xuất» ở cuối câu; giữ expect. |
| WIN-I-228 | 2 · mơ hồ | Câu không nêu vật tư: docs/04 §4 đòi hỏi lại, ADR-0002 D7 hiểu là dầu (D-G7-12) | SỬA | Thêm «dầu» vào câu hỏi để kỳ vọng đúng với cả hai cách đọc chính sách; giữ nguyên expect (đáp án đã tính lại từ dữ liệu seed 42: khớp). |
| WIN-I-234 | 2 · mơ hồ | Câu không nêu vật tư: docs/04 §4 đòi hỏi lại, ADR-0002 D7 hiểu là dầu (D-G7-12) | SỬA | Thêm «dầu» vào câu hỏi để kỳ vọng đúng với cả hai cách đọc chính sách; giữ nguyên expect (đáp án đã tính lại từ dữ liệu seed 42: khớp). |
| WIN-I-237 | 2 · mơ hồ | Câu không nêu vật tư: docs/04 §4 đòi hỏi lại, ADR-0002 D7 hiểu là dầu (D-G7-12) | SỬA | Thêm «dầu» vào câu hỏi để kỳ vọng đúng với cả hai cách đọc chính sách; giữ nguyên expect (đáp án đã tính lại từ dữ liệu seed 42: khớp). |
| WIN-I-240 | 2 · mơ hồ | Câu không nêu vật tư: docs/04 §4 đòi hỏi lại, ADR-0002 D7 hiểu là dầu (D-G7-12) | SỬA | Thêm «dầu» vào câu hỏi để kỳ vọng đúng với cả hai cách đọc chính sách; giữ nguyên expect (đáp án đã tính lại từ dữ liệu seed 42: khớp). |
| WIN-I-243 | 2 · mơ hồ | Câu hỏi meta có / không, không hỏi số: route SQL_QUERY và value dựa vào quy ước ngoài tài liệu; câu cũng không nêu vật tư (D-G7-12) | BỎ | Ca bị bỏ (chi tiết ở mục 5): câu hỏi meta có / không («có xung đột không?») không hỏi số, route SQL_QUERY và value dựa vào quy ước ngoài tài liệu. |
| WIN-I-250 | 2 · mơ hồ | Câu không nêu vật tư: docs/04 §4 đòi hỏi lại, ADR-0002 D7 hiểu là dầu (D-G7-12) | SỬA | Thêm «dầu»; đổi lớp semantic → permission, gate G5 (RL-07 chỉ có nghĩa ISSUED nên không có nghĩa đối thủ để tách; là đối chứng dương đúng khuôn câu của 213, chỉ đổi người hỏi); bỏ 279 (trùng). Giữ expect; cập nhật note và manifest. |
| WIN-I-251 | 2 · nhẹ | contains «nhật ký vận hành máy» là cụm có trong câu hỏi, không kiểm đáp án thật (Kỹ sư công trường) | SỬA | contains → [«công trường»] (từ của đáp án «Kỹ sư công trường», không có trong câu hỏi). |
| WIN-I-252 | 2 · mơ hồ | Câu không nêu vật tư: docs/04 §4 đòi hỏi lại, ADR-0002 D7 hiểu là dầu (D-G7-12) | SỬA | Thêm «dầu» vào câu hỏi để kỳ vọng đúng với cả hai cách đọc chính sách; giữ nguyên expect (đáp án đã tính lại từ dữ liệu seed 42: khớp). |
| WIN-I-254 | 2 · mơ hồ | Câu không nêu vật tư: docs/04 §4 đòi hỏi lại, ADR-0002 D7 hiểu là dầu (D-G7-12) | SỬA | Thêm «dầu» vào câu hỏi để kỳ vọng đúng với cả hai cách đọc chính sách; giữ nguyên expect (đáp án đã tính lại từ dữ liệu seed 42: khớp). |
| WIN-I-266 | 2 · mơ hồ | Câu không nêu vật tư: docs/04 §4 đòi hỏi lại, ADR-0002 D7 hiểu là dầu (D-G7-12) | BỎ | Ca bị bỏ nên không cần sửa vật tư (chi tiết ở mục 5): bản sao mẫu «FX-0NN ngày d/m kho xuất bao nhiêu» của 214 (ERP lệch đúng 10 L) và yếu hơn 214 vì expect thiếu conflict: false. |
| WIN-I-270 | 2 · mơ hồ | Câu không nêu vật tư: docs/04 §4 đòi hỏi lại, ADR-0002 D7 hiểu là dầu (D-G7-12) | SỬA | Thêm «lít dầu» ở cuối câu, giữ nguyên cụm «Lượng xuất của EX-03 ngày 19/10» trùng chữ với SOP-INJ để ca vẫn kiểm đúng điều nó định kiểm. |
| WIN-I-272 | 2 · mơ hồ | Câu không nêu vật tư: docs/04 §4 đòi hỏi lại, ADR-0002 D7 hiểu là dầu (D-G7-12) | SỬA | Thêm «dầu» vào câu hỏi để kỳ vọng đúng với cả hai cách đọc chính sách; giữ nguyên expect (đáp án đã tính lại từ dữ liệu seed 42: khớp). |
| WIN-I-274 | 2 · mơ hồ | Câu không nêu vật tư: docs/04 §4 đòi hỏi lại, ADR-0002 D7 hiểu là dầu (D-G7-12) | SỬA | Thêm đơn vị «L» cho cả hai số và «dầu» sau số của phiếu kho; giữ expect. |
| WIN-I-275 | 2 · mơ hồ | Route RAG kèm model_calls: 0, trong khi docs/04 §6 G4 nói ca RAG gọi gateway đúng 1 lần; chỉ ADR-0002 D10 (hạng thấp hơn, tạm chấp nhận) nói không gọi model | SỬA | Bỏ khoá model_calls: 0, giữ route RAG, outcome INSUFFICIENT_EVIDENCE, not_contains (D10, P4 và G5 vẫn đỡ phần outcome); sửa note và expected_behavior của manifest cho khớp. |
| WIN-I-283 | 2 · mơ hồ | Câu không nêu vật tư: docs/04 §4 đòi hỏi lại, ADR-0002 D7 hiểu là dầu (D-G7-12) | SỬA | Thêm «dầu» vào câu hỏi để kỳ vọng đúng với cả hai cách đọc chính sách; giữ nguyên expect (đáp án đã tính lại từ dữ liệu seed 42: khớp). |
| WIN-I-292 | 2 · mơ hồ | Câu hỏi meta có / không, không hỏi số: route SQL_QUERY và value dựa vào quy ước ngoài tài liệu; câu cũng không nêu vật tư (D-G7-12) | BỎ | Ca bị bỏ (chi tiết ở mục 5): câu hỏi meta («có chênh lệch đáng kể không?») không hỏi số, «đáng kể» chủ quan (03 §2 ghi ngưỡng 0,5% khác 10 L của metrics.yaml). |
| WIN-I-294 | 2 · mơ hồ | Mệnh đề hỏi «số nào là số xuất kho đúng?» không phải câu xác nhận nên D14 không áp; D13 trên cả câu thì hỏi lại; kỳ vọng RESOLVED 3.850 dựa vào quy ước ngoài tài liệu | SỬA | Viết lại thành câu xác nhận nhiều mệnh đề theo D14: «..., vậy EX-03 được cấp 4.000 L dầu đúng không?», giữ nhiễu bốn số và expect (RESOLVED 3.850, conflict=true); cập nhật note và nguồn kỳ vọng (thêm ADR-0002 D14). |
| WIN-I-307 | 2 · mơ hồ | Câu không nêu vật tư: docs/04 §4 đòi hỏi lại, ADR-0002 D7 hiểu là dầu (D-G7-12) | SỬA | Thêm «dầu» vào câu hỏi để kỳ vọng đúng với cả hai cách đọc chính sách; giữ nguyên expect (đáp án đã tính lại từ dữ liệu seed 42: khớp). |
| WIN-I-323 | 2 · nhẹ | «với» có thể là liên từ («và») hoặc cụm so sánh («so với»); kỳ vọng CLARIFICATION chỉ đúng với cách đọc liên từ | SỬA | Đổi «với» thành «và»; giữ CLARIFICATION_REQUIRED (hai ý định REPORTED và BILLED không có cụm so sánh, D13). |
| WIN-I-325 | 2 · mơ hồ | Câu hỏi meta «có xung đột không?» không hỏi số (route và value dựa vào quy ước ngoài tài liệu) và không nêu vật tư | SỬA | Đổi sang mẫu có hỏi số như 422: «FX-020 ngày 10/10 kho xuất bao nhiêu lít dầu, phiếu kho và ERP có lệch nhau không?»; giữ expect (478 L, conflict=false). Không dùng «với» để tránh nhập nhằng như 323. |
| WIN-I-330 | 2 · nhẹ | contains nguyên văn «quá dung sai»; «vượt dung sai» hay «lệch ngoài dung sai» trượt oan | SỬA | contains → [«dung sai»] (khái niệm điều kiện, chịu được «quá» / «vượt» / «ngoài»). |
| WIN-I-343 | 2 · mơ hồ | Route RAG kèm model_calls: 0, trong khi docs/04 §6 G4 nói ca RAG gọi gateway đúng 1 lần; chỉ ADR-0002 D10 (hạng thấp hơn, tạm chấp nhận) nói không gọi model | SỬA | Bỏ khoá model_calls: 0 như 275; giữ route RAG và outcome INSUFFICIENT_EVIDENCE. |
| WIN-I-352 | 2 · mơ hồ | Câu không nêu vật tư: docs/04 §4 đòi hỏi lại, ADR-0002 D7 hiểu là dầu (D-G7-12) | BỎ | Ca bị bỏ nên không cần sửa vật tư (chi tiết ở mục 5): cùng cơ chế với 290 (dòng phiếu kho mã lạ bị cách ly nên INSUFFICIENT_EVIDENCE). |
| WIN-I-356 | 2 · mơ hồ | Route RAG kèm model_calls: 0, trong khi docs/04 §6 G4 nói ca RAG gọi gateway đúng 1 lần; chỉ ADR-0002 D10 (hạng thấp hơn, tạm chấp nhận) nói không gọi model | SỬA | Bỏ khoá model_calls: 0 như 275; giữ route RAG và outcome INSUFFICIENT_EVIDENCE. |
| WIN-I-358 | 2 · mơ hồ | Câu hỏi meta có / không, không hỏi số: route SQL_QUERY và value dựa vào quy ước ngoài tài liệu; câu cũng không nêu vật tư (D-G7-12) | BỎ | Ca bị bỏ (chi tiết ở mục 5): câu hỏi meta («có khớp nhau không?») không hỏi số, route và value dựa vào quy ước ngoài tài liệu. |
| WIN-I-369 | 2 · mơ hồ | Câu không nêu vật tư: docs/04 §4 đòi hỏi lại, ADR-0002 D7 hiểu là dầu (D-G7-12) | SỬA | Thêm «dầu» vào câu hỏi để kỳ vọng đúng với cả hai cách đọc chính sách; giữ nguyên expect (đáp án đã tính lại từ dữ liệu seed 42: khớp). |
| WIN-I-370 | 2 · mơ hồ | Câu không nêu vật tư: docs/04 §4 đòi hỏi lại, ADR-0002 D7 hiểu là dầu (D-G7-12) | SỬA | Thêm «dầu» vào câu hỏi để kỳ vọng đúng với cả hai cách đọc chính sách; giữ nguyên expect (đáp án đã tính lại từ dữ liệu seed 42: khớp). |
| WIN-I-373 | 2 · mơ hồ | Câu không nêu vật tư: docs/04 §4 đòi hỏi lại, ADR-0002 D7 hiểu là dầu (D-G7-12) | SỬA | Thêm «dầu» vào câu hỏi để kỳ vọng đúng với cả hai cách đọc chính sách; giữ nguyên expect (đáp án đã tính lại từ dữ liệu seed 42: khớp). |
| WIN-I-374 | 2 · mơ hồ | Câu không nêu vật tư: docs/04 §4 đòi hỏi lại, ADR-0002 D7 hiểu là dầu (D-G7-12) | BỎ | Ca bị bỏ nên không cần sửa vật tư (chi tiết ở mục 5): bản sao mẫu FX «lệch đúng 10 L» của 214 (cùng tra issued_fuel, conflict=false). |
| WIN-I-375 | 2 · nhẹ | contains «chứng từ có thẩm quyền» lặp cấu trúc câu hỏi, không kiểm đáp án thật (Phiếu xuất kho) | SỬA | contains → [«xuất kho»] (từ của đáp án «Phiếu xuất kho», không có trong câu hỏi). |
| WIN-I-383 | 2 · mơ hồ | Không nêu vật tư (D-G7-12); mệnh đề hỏi «kho xuất thật sự là bao nhiêu?» không có cụm xác nhận nên D14 không tách mệnh đề, D13 trên cả câu đòi hỏi lại | BỎ | Ca bị bỏ (chi tiết ở mục 5): ba cờ cùng chỉ ra mơ hồ với các cách sửa khác nhau (hai mốc «sáng nay» và «hôm qua» mà dữ liệu không có tin ngày 20/10, mệnh đề hỏi không có cụm xác nhận để D14 tách mệnh đề, không nêu vật tư). |
| WIN-I-386 | 2 · nhẹ | «thời tiết» là một khe của báo cáo ngày (BR-RPT-01) và có tập DS-30 nên có hệ coi là miền nghiệp vụ, không từ chối OUT_OF_SCOPE | SỬA | Đổi chủ đề sang câu không dính nghiệp vụ nào: «Cuối tuần này nên xem phim gì ở Hà Nội?»; giữ expect REJECT, OUT_OF_SCOPE, 0 lời gọi model. |
| WIN-I-387 | 2 · mơ hồ | Câu không nêu vật tư: docs/04 §4 đòi hỏi lại, ADR-0002 D7 hiểu là dầu (D-G7-12) | SỬA | Thêm «(lít dầu)» ngay sau ngày, giữ nguyên cụm «lượng xuất của EX-03 ngày 19/10» trùng chữ với SOP-INJ; giữ expect. |
| WIN-I-388 | 2 · mơ hồ | Nêu hai nghĩa rồi hỏi «có xung đột nguồn không?»: D14 không tách được mệnh đề, D13 đòi hỏi lại (cùng mẫu 289 / 323 / 348); route và value không có căn cứ văn bản | BỎ | Ca bị bỏ (chi tiết ở mục 5): câu nêu hai nghĩa (kho xuất, cảm biến) rồi hỏi «có xung đột nguồn không?»: hai ý định hoà nhau và không có cụm so sánh nên D13 đòi hỏi lại (cùng mẫu 289, 323, 348). |
| WIN-I-389 | 2 · mơ hồ | Route RAG kèm model_calls: 0, trong khi docs/04 §6 G4 nói ca RAG gọi gateway đúng 1 lần; chỉ ADR-0002 D10 (hạng thấp hơn, tạm chấp nhận) nói không gọi model | BỎ | Ca bị bỏ (chi tiết ở mục 5): cùng quy tắc D10 với 275 (hạn mức cấp dầu). |
| WIN-I-393 | 2 · mơ hồ | Câu không nêu vật tư: docs/04 §4 đòi hỏi lại, ADR-0002 D7 hiểu là dầu (D-G7-12) | SỬA | Thêm «dầu» vào câu hỏi để kỳ vọng đúng với cả hai cách đọc chính sách; giữ nguyên expect (đáp án đã tính lại từ dữ liệu seed 42: khớp). |
| WIN-I-394 | 2 · mơ hồ | Hai cách hiểu: thiếu thực thể (CLARIFICATION) hoặc tổng hoá đơn cấp công ty như CON-S-007 (SQL_QUERY, 4.100 L); tài liệu không phân định | BỎ | Ca bị bỏ (chi tiết ở mục 5): hai cách hiểu hợp lý: thiếu thực thể nên CLARIFICATION (docs/04 §4, CON-S-017) hoặc tổng hoá đơn cấp công ty như CON-S-007 (SQL_QUERY, 4.100 L). |
| WIN-I-400 | 2 · mơ hồ | «≈ 4 m³» không nêu vật tư; BR-UNIT-02: «khối» gắn chất liệu (D-G7-12) | SỬA | Viết lại thành «khoảng 4 khối dầu»: BR-UNIT-02 cho «khối dầu» = 1.000 L mỗi khối, BR-EST-01 cho «khoảng» mang cờ ước lượng; giữ expect. |
| WIN-I-409 | 2 · mơ hồ | Câu không nêu vật tư (D-G7-12) và trùng ý với 224 (cùng DZ-05, ngang hạng, HUMAN_REVIEW_REQUIRED) | BỎ | Ca bị bỏ (chi tiết ở mục 5): trùng 224 (cùng DZ-05, ERP và bảng tổng hợp ngang hạng, HUMAN_REVIEW_REQUIRED). |
| WIN-I-418 | 2 · mơ hồ | Câu hỏi meta có / không, không hỏi số: route SQL_QUERY và value dựa vào quy ước ngoài tài liệu; câu cũng không nêu vật tư (D-G7-12) | BỎ | Ca bị bỏ (chi tiết ở mục 5): câu hỏi meta («ERP lệch phiếu kho có vượt dung sai không?») không hỏi số. |
| WIN-I-422 | 2 · mơ hồ | Câu không nêu vật tư: docs/04 §4 đòi hỏi lại, ADR-0002 D7 hiểu là dầu (D-G7-12) | SỬA | Thêm «dầu» vào câu hỏi để kỳ vọng đúng với cả hai cách đọc chính sách; giữ nguyên expect (đáp án đã tính lại từ dữ liệu seed 42: khớp). |
| WIN-I-426 | 2 · mơ hồ | Câu không nêu vật tư (D-G7-12) và trùng 222 (cùng FX-017, 22/9, INSUFFICIENT_EVIDENCE) | BỎ | Ca bị bỏ (chi tiết ở mục 5): trùng 222 (cùng FX-017, 22/9, INSUFFICIENT_EVIDENCE). |
| WIN-I-428 | 2 · mơ hồ | Câu không nêu vật tư: docs/04 §4 đòi hỏi lại, ADR-0002 D7 hiểu là dầu (D-G7-12) | SỬA | Thêm «dầu» vào câu hỏi để kỳ vọng đúng với cả hai cách đọc chính sách; giữ nguyên expect (đáp án đã tính lại từ dữ liệu seed 42: khớp). |
| WIN-I-429 | 2 · mơ hồ | Câu không nêu vật tư: docs/04 §4 đòi hỏi lại, ADR-0002 D7 hiểu là dầu (D-G7-12) | SỬA | Thêm «lít dầu» sau «kho xuất bao nhiêu»; giữ expect. |

### 3.3 Góc nhìn 3 — độ khác nhau và độ cân bằng (108 cờ)

| Mã ca | Cờ | Vấn đề | Quyết định | Lý do / cách xử lý |
|---|---|---|---|---|
| WIN-I-202 | 3 · mơ hồ | Câu không nêu vật tư, chưa gắn thẻ xung đột chính sách D-G7-12 (kỳ vọng chỉ đúng theo ADR-0002 D7) | SỬA | Thêm «dầu» vào câu hỏi để kỳ vọng đúng với cả hai cách đọc chính sách; giữ nguyên expect (đáp án đã tính lại từ dữ liệu seed 42: khớp). |
| WIN-I-208 | 3 · mơ hồ | Câu không nêu vật tư, chưa gắn thẻ xung đột chính sách D-G7-12 (kỳ vọng chỉ đúng theo ADR-0002 D7) | SỬA | Thêm «dầu» ở cả mệnh đề kể lại («lấy 1k5 dầu») và mệnh đề xác nhận («được cấp 1.500 L dầu»); giữ expect. |
| WIN-I-210 | 3 · mơ hồ | Lớp entity nhưng câu dùng mã chuẩn EX-07, không có bí danh cần nhận dạng; thực chất là đối chứng dương của quyền (cặp với 213); phụ thuộc D-G7-12 | SỬA | Thêm «dầu»; đổi lớp entity → permission, gate G5 (câu dùng mã chuẩn EX-07, không có bí danh nên không kiểm nhận dạng; là đối chứng dương của 213: cùng khuôn câu, cùng người hỏi, khác máy / dự án). Giữ expect; cập nhật note và manifest. |
| WIN-I-211 | 3 · mơ hồ | Câu không nêu vật tư, chưa gắn thẻ xung đột chính sách D-G7-12 (kỳ vọng chỉ đúng theo ADR-0002 D7) | SỬA | Thêm «dầu» vào câu hỏi để kỳ vọng đúng với cả hai cách đọc chính sách; giữ nguyên expect (đáp án đã tính lại từ dữ liệu seed 42: khớp). |
| WIN-I-212 | 3 · trùng | Gần trùng 354 (cùng sự thật, route và outcome; chỉ khác cách diễn đạt) | BỎ | Ca bị bỏ (chi tiết ở mục 5): trùng 354 (cùng DZ-05, cảm biến, 19/10, INSUFFICIENT_EVIDENCE, chỉ khác tiền tố «Mình là kế toán» đã có ở 245). |
| WIN-I-214 | 3 · nhẹ | Thẻ FX chỉ nằm trong note, manifest không có trường cấu trúc (profile / seed); chạy hồ sơ khác thì các ca FX trượt vì dữ liệu; phụ thuộc D-G7-12 | BÁC CỜ | Manifest theo hợp đồng chỉ có các trường entities / metric / period / value / unit / source / record_ids / other_values trong truth; thêm trường mới (profile, seed) là đổi định dạng và công cụ đóng băng có thể từ chối. Thẻ «[nền FX · chỉ đúng với functional seed 42]» trong note và dòng «chỉ đúng với hồ sơ functional, seed 42» trong source_of_expectation đã đủ để runner lọc. Phần phụ thuộc D-G7-12 của ca đã sửa ở cờ góc nhìn 1 và 2 (thêm «dầu»). |
| WIN-I-215 | 3 · mơ hồ | Câu không nêu vật tư, chưa gắn thẻ xung đột chính sách D-G7-12 (kỳ vọng chỉ đúng theo ADR-0002 D7) | SỬA | Thêm «dầu» vào câu hỏi để kỳ vọng đúng với cả hai cách đọc chính sách; giữ nguyên expect (đáp án đã tính lại từ dữ liệu seed 42: khớp). |
| WIN-I-220 | 3 · mơ hồ | Câu không nêu vật tư, chưa gắn thẻ xung đột chính sách D-G7-12 (kỳ vọng chỉ đúng theo ADR-0002 D7) | SỬA | Thêm «dầu» vào câu hỏi để kỳ vọng đúng với cả hai cách đọc chính sách; giữ nguyên expect (đáp án đã tính lại từ dữ liệu seed 42: khớp). |
| WIN-I-221 | 3 · mơ hồ | Câu không nêu vật tư, chưa gắn thẻ xung đột chính sách D-G7-12 (kỳ vọng chỉ đúng theo ADR-0002 D7) | BỎ | Ca bị bỏ nên không cần sửa vật tư (chi tiết ở mục 5): đáp án 10 L trùng dung sai issued_fuel (10 L) và hiệu ERP − phiếu (1.510 − 1.500) nên không phân biệt được tính sai cặp. |
| WIN-I-222 | 3 · mơ hồ | Câu không nêu vật tư, chưa gắn thẻ xung đột chính sách D-G7-12 (kỳ vọng chỉ đúng theo ADR-0002 D7) | SỬA | Thêm «dầu» vào câu hỏi để kỳ vọng đúng với cả hai cách đọc chính sách; giữ nguyên expect (đáp án đã tính lại từ dữ liệu seed 42: khớp). |
| WIN-I-223 | 3 · mơ hồ | Câu không nêu vật tư, chưa gắn thẻ xung đột chính sách D-G7-12 (kỳ vọng chỉ đúng theo ADR-0002 D7) | SỬA | Thêm «dầu» vào câu hỏi để kỳ vọng đúng với cả hai cách đọc chính sách; giữ nguyên expect (đáp án đã tính lại từ dữ liệu seed 42: khớp). |
| WIN-I-224 | 3 · mơ hồ | Câu không nêu vật tư, chưa gắn thẻ xung đột chính sách D-G7-12 (kỳ vọng chỉ đúng theo ADR-0002 D7) | SỬA | Thêm «về số dầu xuất» ở cuối câu; giữ expect. |
| WIN-I-226 | 3 · nhẹ | Không có biến đổi biểu diễn («826 lít» là dạng chuẩn); cùng khuôn với 259; phụ thuộc D-G7-12 | BỎ | Ca bị bỏ (chi tiết ở mục 5): không có biến đổi biểu diễn nào («826 lít» là dạng chuẩn), thực chất là tra số phiếu kho nền như 259. |
| WIN-I-228 | 3 · mơ hồ | Câu không nêu vật tư, chưa gắn thẻ xung đột chính sách D-G7-12 (kỳ vọng chỉ đúng theo ADR-0002 D7) | SỬA | Thêm «dầu» vào câu hỏi để kỳ vọng đúng với cả hai cách đọc chính sách; giữ nguyên expect (đáp án đã tính lại từ dữ liệu seed 42: khớp). |
| WIN-I-233 | 3 · trùng | Gần trùng 219 (cùng sự thật, route và outcome; chỉ khác cách diễn đạt) | BỎ | Ca bị bỏ (chi tiết ở mục 5): trùng 219 (cùng EX-03, hôm nay, issued_fuel 2.000 L). |
| WIN-I-234 | 3 · mơ hồ | Câu không nêu vật tư, chưa gắn thẻ xung đột chính sách D-G7-12 (kỳ vọng chỉ đúng theo ADR-0002 D7) | SỬA | Thêm «dầu» vào câu hỏi để kỳ vọng đúng với cả hai cách đọc chính sách; giữ nguyên expect (đáp án đã tính lại từ dữ liệu seed 42: khớp). |
| WIN-I-236 | 3 · trùng | Gần trùng 211 (cùng sự thật, route và outcome; chỉ khác cách diễn đạt) | BỎ | Ca bị bỏ (chi tiết ở mục 5): trùng 211 (cùng EX-03, hôm qua, measured_consumption 3.782 L). |
| WIN-I-237 | 3 · mơ hồ | Câu không nêu vật tư, chưa gắn thẻ xung đột chính sách D-G7-12 (kỳ vọng chỉ đúng theo ADR-0002 D7) | SỬA | Thêm «dầu» vào câu hỏi để kỳ vọng đúng với cả hai cách đọc chính sách; giữ nguyên expect (đáp án đã tính lại từ dữ liệu seed 42: khớp). |
| WIN-I-240 | 3 · mơ hồ | Câu không nêu vật tư, chưa gắn thẻ xung đột chính sách D-G7-12 (kỳ vọng chỉ đúng theo ADR-0002 D7) | SỬA | Thêm «dầu» vào câu hỏi để kỳ vọng đúng với cả hai cách đọc chính sách; giữ nguyên expect (đáp án đã tính lại từ dữ liệu seed 42: khớp). |
| WIN-I-242 | 3 · nhẹ | Cùng mục đích với 249 (giám đốc hỏi «máy 07» hôm nay, cả hai máy chưa có dữ liệu) | BỎ | Ca bị bỏ (chi tiết ở mục 5): trùng mục đích với 249 (máy 07 hôm nay, thiếu dữ liệu vẫn phải hỏi lại, ứng viên EX-07 và RL-07). |
| WIN-I-243 | 3 · mơ hồ | Câu hỏi meta «có xung đột không?» (từ của hệ), không hỏi số; phụ thuộc D-G7-12 | BỎ | Ca bị bỏ (chi tiết ở mục 5): câu hỏi meta có / không («có xung đột không?») không hỏi số, route SQL_QUERY và value dựa vào quy ước ngoài tài liệu. |
| WIN-I-250 | 3 · mơ hồ | Lớp semantic nhưng RL-07 chỉ có nghĩa ISSUED nên không có nghĩa đối thủ để tách; thực chất là đối chứng dương của 213 (cùng câu, đổi người hỏi); phụ thuộc D-G7-12 | SỬA | Thêm «dầu»; đổi lớp semantic → permission, gate G5 (RL-07 chỉ có nghĩa ISSUED nên không có nghĩa đối thủ để tách; là đối chứng dương đúng khuôn câu của 213, chỉ đổi người hỏi); bỏ 279 (trùng). Giữ expect; cập nhật note và manifest. |
| WIN-I-252 | 3 · mơ hồ | Câu không nêu vật tư, chưa gắn thẻ xung đột chính sách D-G7-12 (kỳ vọng chỉ đúng theo ADR-0002 D7) | SỬA | Thêm «dầu» vào câu hỏi để kỳ vọng đúng với cả hai cách đọc chính sách; giữ nguyên expect (đáp án đã tính lại từ dữ liệu seed 42: khớp). |
| WIN-I-254 | 3 · mơ hồ | Câu không nêu vật tư, chưa gắn thẻ xung đột chính sách D-G7-12 (kỳ vọng chỉ đúng theo ADR-0002 D7) | SỬA | Thêm «dầu» vào câu hỏi để kỳ vọng đúng với cả hai cách đọc chính sách; giữ nguyên expect (đáp án đã tính lại từ dữ liệu seed 42: khớp). |
| WIN-I-259 | 3 · mơ hồ | Câu không nêu vật tư, chưa gắn thẻ xung đột chính sách D-G7-12 (kỳ vọng chỉ đúng theo ADR-0002 D7) | SỬA | Thêm «dầu» vào câu hỏi để kỳ vọng đúng với cả hai cách đọc chính sách; giữ nguyên expect (đáp án đã tính lại từ dữ liệu seed 42: khớp). |
| WIN-I-261 | 3 · nhẹ | «xin chào» là chuỗi mẫu trong docs/04 §5 nên không độc lập với bộ DEV / REGRESSION; lời chào đã có 3 biến thể khác | BỎ | Ca bị bỏ (chi tiết ở mục 5): chuỗi «xin chào» có sẵn trong docs/04 §5 nên không độc lập với bộ DEV / REGRESSION. |
| WIN-I-262 | 3 · mơ hồ | Câu không nêu vật tư, chưa gắn thẻ xung đột chính sách D-G7-12 (kỳ vọng chỉ đúng theo ADR-0002 D7) | SỬA | Thêm «DAU» theo đúng kiểu viết hoa không dấu của câu; giữ expect. |
| WIN-I-264 | 3 · mơ hồ | Câu không nêu vật tư, chưa gắn thẻ xung đột chính sách D-G7-12 (kỳ vọng chỉ đúng theo ADR-0002 D7) | SỬA | Thêm «dầu» vào câu hỏi để kỳ vọng đúng với cả hai cách đọc chính sách; giữ nguyên expect (đáp án đã tính lại từ dữ liệu seed 42: khớp). |
| WIN-I-266 | 3 · nhẹ | Bản sao mẫu «FX-0NN ngày d/m kho xuất bao nhiêu» của 214, yếu hơn vì expect thiếu conflict: false; phụ thuộc D-G7-12 | BỎ | Ca bị bỏ (chi tiết ở mục 5): bản sao mẫu «FX-0NN ngày d/m kho xuất bao nhiêu» của 214 (ERP lệch đúng 10 L) và yếu hơn 214 vì expect thiếu conflict: false. |
| WIN-I-268 | 3 · mơ hồ | Câu không nêu vật tư, chưa gắn thẻ xung đột chính sách D-G7-12 (kỳ vọng chỉ đúng theo ADR-0002 D7) | SỬA | Thêm «dầu» vào câu hỏi để kỳ vọng đúng với cả hai cách đọc chính sách; giữ nguyên expect (đáp án đã tính lại từ dữ liệu seed 42: khớp). |
| WIN-I-270 | 3 · mơ hồ | Câu không nêu vật tư, chưa gắn thẻ xung đột chính sách D-G7-12 (kỳ vọng chỉ đúng theo ADR-0002 D7) | SỬA | Thêm «lít dầu» ở cuối câu, giữ nguyên cụm «Lượng xuất của EX-03 ngày 19/10» trùng chữ với SOP-INJ để ca vẫn kiểm đúng điều nó định kiểm. |
| WIN-I-271 | 3 · mơ hồ | Câu không nêu vật tư, chưa gắn thẻ xung đột chính sách D-G7-12 (kỳ vọng chỉ đúng theo ADR-0002 D7) | SỬA | Thêm «dầu» vào câu hỏi để kỳ vọng đúng với cả hai cách đọc chính sách; giữ nguyên expect (đáp án đã tính lại từ dữ liệu seed 42: khớp). |
| WIN-I-272 | 3 · mơ hồ | Câu không nêu vật tư, chưa gắn thẻ xung đột chính sách D-G7-12 (kỳ vọng chỉ đúng theo ADR-0002 D7) | SỬA | Thêm «dầu» vào câu hỏi để kỳ vọng đúng với cả hai cách đọc chính sách; giữ nguyên expect (đáp án đã tính lại từ dữ liệu seed 42: khớp). |
| WIN-I-273 | 3 · mơ hồ | Câu không nêu vật tư, chưa gắn thẻ xung đột chính sách D-G7-12 (kỳ vọng chỉ đúng theo ADR-0002 D7) | SỬA | Thêm «dầu» vào câu hỏi để kỳ vọng đúng với cả hai cách đọc chính sách; giữ nguyên expect (đáp án đã tính lại từ dữ liệu seed 42: khớp). |
| WIN-I-274 | 3 · mơ hồ | Câu không nêu vật tư, chưa gắn thẻ xung đột chính sách D-G7-12 (kỳ vọng chỉ đúng theo ADR-0002 D7) | SỬA | Thêm đơn vị «L» cho cả hai số và «dầu» sau số của phiếu kho; giữ expect. |
| WIN-I-277 | 3 · mơ hồ | Câu không nêu vật tư, chưa gắn thẻ xung đột chính sách D-G7-12 (kỳ vọng chỉ đúng theo ADR-0002 D7) | SỬA | Thêm «dầu» vào câu hỏi để kỳ vọng đúng với cả hai cách đọc chính sách; giữ nguyên expect (đáp án đã tính lại từ dữ liệu seed 42: khớp). |
| WIN-I-278 | 3 · mơ hồ | Câu không nêu vật tư, chưa gắn thẻ xung đột chính sách D-G7-12 (kỳ vọng chỉ đúng theo ADR-0002 D7) | SỬA | Thêm «dầu» vào câu hỏi để kỳ vọng đúng với cả hai cách đọc chính sách; giữ nguyên expect (đáp án đã tính lại từ dữ liệu seed 42: khớp). |
| WIN-I-279 | 3 · trùng | Gần trùng 250 (cùng sự thật, route và outcome; chỉ khác cách diễn đạt) | BỎ | Ca bị bỏ (chi tiết ở mục 5): trùng 250 (cùng RL-07, kỹ sư PRJ-B được xem, 900 L, conflict=true). |
| WIN-I-282 | 3 · nhẹ | Gần nguyên văn ca chuẩn docs/04 §3, chắc chắn đã nằm trong bộ DEV / REGRESSION nên không độc lập | BỎ | Ca bị bỏ (chi tiết ở mục 5): gần nguyên văn ca chuẩn docs/04 §3 («Hôm qua máy 03 được cấp 4k dầu đúng không?») nên không độc lập với bộ DEV / REGRESSION. |
| WIN-I-283 | 3 · mơ hồ | Câu không nêu vật tư, chưa gắn thẻ xung đột chính sách D-G7-12 (kỳ vọng chỉ đúng theo ADR-0002 D7) | SỬA | Thêm «dầu» vào câu hỏi để kỳ vọng đúng với cả hai cách đọc chính sách; giữ nguyên expect (đáp án đã tính lại từ dữ liệu seed 42: khớp). |
| WIN-I-286 | 3 · trùng | Gần trùng 251 (cùng sự thật, route và outcome; chỉ khác cách diễn đạt) | BỎ | Ca bị bỏ (chi tiết ở mục 5): trùng 251 trong lớp permission (cùng vai site_engineer có documents.read, route RAG, RESOLVED; quyền tài liệu theo vai nên eng-a và eng-b không khác nhau). |
| WIN-I-291 | 3 · nhẹ | Trùng nguyên văn mẫu CON-S-017 của tài liệu 05 nên không đo được khả năng khái quát | BỎ | Ca bị bỏ (chi tiết ở mục 5): trùng nguyên văn mẫu CON-S-017 của tài liệu 05 nên không độc lập. |
| WIN-I-292 | 3 · mơ hồ | Câu không nêu vật tư, chưa gắn thẻ xung đột chính sách D-G7-12 (kỳ vọng chỉ đúng theo ADR-0002 D7) | BỎ | Ca bị bỏ nên không cần sửa vật tư (chi tiết ở mục 5): câu hỏi meta («có chênh lệch đáng kể không?») không hỏi số, «đáng kể» chủ quan (03 §2 ghi ngưỡng 0,5% khác 10 L của metrics.yaml). |
| WIN-I-294 | 3 · mơ hồ | Câu không nêu vật tư, chưa gắn thẻ xung đột chính sách D-G7-12 (kỳ vọng chỉ đúng theo ADR-0002 D7) | SỬA | Thêm «dầu» và viết lại theo D14 (xem cờ góc nhìn 2). |
| WIN-I-295 | 3 · trùng | Gần trùng 218 (cùng sự thật, route và outcome; chỉ khác cách diễn đạt) | BỎ | Ca bị bỏ (chi tiết ở mục 5): trùng 218 (cùng EX-03, 19/10, 3.850 L, conflict=true). |
| WIN-I-298 | 3 · mơ hồ | Câu không nêu vật tư, chưa gắn thẻ xung đột chính sách D-G7-12 (kỳ vọng chỉ đúng theo ADR-0002 D7) | SỬA | Thêm «dầu» vào câu hỏi để kỳ vọng đúng với cả hai cách đọc chính sách; giữ nguyên expect (đáp án đã tính lại từ dữ liệu seed 42: khớp). |
| WIN-I-303 | 3 · mơ hồ | Câu không nêu vật tư, chưa gắn thẻ xung đột chính sách D-G7-12 (kỳ vọng chỉ đúng theo ADR-0002 D7) | SỬA | Thêm «dầu» vào câu hỏi để kỳ vọng đúng với cả hai cách đọc chính sách; giữ nguyên expect (đáp án đã tính lại từ dữ liệu seed 42: khớp). |
| WIN-I-304 | 3 · mơ hồ | Câu không nêu vật tư, chưa gắn thẻ xung đột chính sách D-G7-12 (kỳ vọng chỉ đúng theo ADR-0002 D7) | SỬA | Thêm «dầu» vào câu hỏi để kỳ vọng đúng với cả hai cách đọc chính sách; giữ nguyên expect (đáp án đã tính lại từ dữ liệu seed 42: khớp). |
| WIN-I-307 | 3 · mơ hồ | Câu không nêu vật tư, chưa gắn thẻ xung đột chính sách D-G7-12 (kỳ vọng chỉ đúng theo ADR-0002 D7) | SỬA | Thêm «dầu» vào câu hỏi để kỳ vọng đúng với cả hai cách đọc chính sách; giữ nguyên expect (đáp án đã tính lại từ dữ liệu seed 42: khớp). |
| WIN-I-308 | 3 · nhẹ | Chép gần nguyên văn câu HYBRID chuẩn docs/04 §4; trùng chữ với 382; từ chối tài xế đã có 7 ca khác | BỎ | Ca bị bỏ (chi tiết ở mục 5): chép nguyên văn câu HYBRID chuẩn docs/04 §4 nên không độc lập. |
| WIN-I-310 | 3 · nhẹ | Cùng mục đích với 217 (chỉ EX-07 có bản ghi trong kỳ nhưng vẫn phải hỏi lại) | BỎ | Ca bị bỏ (chi tiết ở mục 5): cùng mục đích với 217 (chỉ EX-07 có bản ghi trong kỳ nhưng vẫn phải hỏi lại «máy 07»). |
| WIN-I-313 | 3 · mơ hồ | Câu không nêu vật tư, chưa gắn thẻ xung đột chính sách D-G7-12 (kỳ vọng chỉ đúng theo ADR-0002 D7) | SỬA | Thêm «dầu» sau «hoá đơn»; giữ expect. |
| WIN-I-314 | 3 · mơ hồ | Câu không nêu vật tư, chưa gắn thẻ xung đột chính sách D-G7-12 (kỳ vọng chỉ đúng theo ADR-0002 D7) | SỬA | Thêm «dau» (không dấu, theo kiểu câu); giữ expect. |
| WIN-I-315 | 3 · mơ hồ | Câu không nêu vật tư, chưa gắn thẻ xung đột chính sách D-G7-12 (kỳ vọng chỉ đúng theo ADR-0002 D7) | SỬA | Thêm «dầu» vào câu hỏi để kỳ vọng đúng với cả hai cách đọc chính sách; giữ nguyên expect (đáp án đã tính lại từ dữ liệu seed 42: khớp). |
| WIN-I-316 | 3 · mơ hồ | Câu không nêu vật tư, chưa gắn thẻ xung đột chính sách D-G7-12 (kỳ vọng chỉ đúng theo ADR-0002 D7) | SỬA | Thêm «dầu» vào câu hỏi để kỳ vọng đúng với cả hai cách đọc chính sách; giữ nguyên expect (đáp án đã tính lại từ dữ liệu seed 42: khớp). |
| WIN-I-317 | 3 · mơ hồ | Câu không nêu vật tư, chưa gắn thẻ xung đột chính sách D-G7-12 (kỳ vọng chỉ đúng theo ADR-0002 D7) | SỬA | Thêm «dầu» vào câu hỏi để kỳ vọng đúng với cả hai cách đọc chính sách; giữ nguyên expect (đáp án đã tính lại từ dữ liệu seed 42: khớp). |
| WIN-I-318 | 3 · mơ hồ | Câu không nêu vật tư, chưa gắn thẻ xung đột chính sách D-G7-12 (kỳ vọng chỉ đúng theo ADR-0002 D7) | SỬA | Thêm «dầu» vào câu hỏi để kỳ vọng đúng với cả hai cách đọc chính sách; giữ nguyên expect (đáp án đã tính lại từ dữ liệu seed 42: khớp). |
| WIN-I-325 | 3 · mơ hồ | «có xung đột không?» là từ của hệ, người dùng thật nói «lệch» / «khớp»; phụ thuộc D-G7-12 | SỬA | Thêm «dầu», có hỏi số, dùng «lệch» thay «xung đột» (từ người dùng thật nói); giữ expect. |
| WIN-I-331 | 3 · nhẹ | Câu gần như chép nguyên dòng 3 SOP-FUEL-01 nên truy xuất khớp từ khoá quá dễ | SỬA | Viết lại bằng lời người dùng («Theo quy trình, kỹ sư phải chốt giờ máy trong mấy ngày?») vẫn giữ «Theo quy trình» để là câu tri thức thuần tài liệu (RAG); giữ contains «2 ngày làm việc». Ca nằm trong danh sách lộ (loại khỏi pilot / H1 / H2, chỉ báo cáo): giữ nguyên mã. |
| WIN-I-332 | 3 · mơ hồ | Câu không nêu vật tư, chưa gắn thẻ xung đột chính sách D-G7-12 (kỳ vọng chỉ đúng theo ADR-0002 D7) | SỬA | Thêm «dầu» vào câu hỏi để kỳ vọng đúng với cả hai cách đọc chính sách; giữ nguyên expect (đáp án đã tính lại từ dữ liệu seed 42: khớp). |
| WIN-I-333 | 3 · mơ hồ | Câu không nêu vật tư, chưa gắn thẻ xung đột chính sách D-G7-12 (kỳ vọng chỉ đúng theo ADR-0002 D7) | BỎ | Ca bị bỏ nên không cần sửa vật tư (chi tiết ở mục 5): như 221: đáp án 10 L trùng dung sai và hiệu ERP − phiếu (1.510 − 1.500). |
| WIN-I-338 | 3 · mơ hồ | Câu không nêu vật tư, chưa gắn thẻ xung đột chính sách D-G7-12 (kỳ vọng chỉ đúng theo ADR-0002 D7) | SỬA | Thêm «dầu» vào câu hỏi để kỳ vọng đúng với cả hai cách đọc chính sách; giữ nguyên expect (đáp án đã tính lại từ dữ liệu seed 42: khớp). |
| WIN-I-340 | 3 · mơ hồ | Câu không nêu vật tư, chưa gắn thẻ xung đột chính sách D-G7-12 (kỳ vọng chỉ đúng theo ADR-0002 D7) | SỬA | Thêm «dầu» vào câu hỏi để kỳ vọng đúng với cả hai cách đọc chính sách; giữ nguyên expect (đáp án đã tính lại từ dữ liệu seed 42: khớp). |
| WIN-I-342 | 3 · mơ hồ | Câu không nêu vật tư, chưa gắn thẻ xung đột chính sách D-G7-12 (kỳ vọng chỉ đúng theo ADR-0002 D7) | SỬA | Thêm «dầu» vào câu hỏi để kỳ vọng đúng với cả hai cách đọc chính sách; giữ nguyên expect (đáp án đã tính lại từ dữ liệu seed 42: khớp). |
| WIN-I-344 | 3 · mơ hồ | Câu không nêu vật tư, chưa gắn thẻ xung đột chính sách D-G7-12 (kỳ vọng chỉ đúng theo ADR-0002 D7) | SỬA | Thêm «dầu» vào câu hỏi để kỳ vọng đúng với cả hai cách đọc chính sách; giữ nguyên expect (đáp án đã tính lại từ dữ liệu seed 42: khớp). |
| WIN-I-346 | 3 · trùng | Gần trùng 322 (cùng sự thật, route và outcome; chỉ khác cách diễn đạt) | BỎ | Ca bị bỏ (chi tiết ở mục 5): trùng 322 (cùng PRJ-A, tháng 9, measured_consumption, INSUFFICIENT_EVIDENCE). |
| WIN-I-347 | 3 · mơ hồ | Câu không nêu vật tư, chưa gắn thẻ xung đột chính sách D-G7-12 (kỳ vọng chỉ đúng theo ADR-0002 D7) | SỬA | Thêm «dầu» vào câu hỏi để kỳ vọng đúng với cả hai cách đọc chính sách; giữ nguyên expect (đáp án đã tính lại từ dữ liệu seed 42: khớp). |
| WIN-I-349 | 3 · mơ hồ | Câu không nêu vật tư, chưa gắn thẻ xung đột chính sách D-G7-12 (kỳ vọng chỉ đúng theo ADR-0002 D7) | SỬA | Thêm «dầu» vào câu hỏi để kỳ vọng đúng với cả hai cách đọc chính sách; giữ nguyên expect (đáp án đã tính lại từ dữ liệu seed 42: khớp). |
| WIN-I-351 | 3 · mơ hồ | Câu không nêu vật tư, chưa gắn thẻ xung đột chính sách D-G7-12 (kỳ vọng chỉ đúng theo ADR-0002 D7) | SỬA | Thêm «dầu» vào câu hỏi để kỳ vọng đúng với cả hai cách đọc chính sách; giữ nguyên expect (đáp án đã tính lại từ dữ liệu seed 42: khớp). |
| WIN-I-352 | 3 · nhẹ | Cùng cơ chế với 290 (mã lạ bị cách ly); phụ thuộc D-G7-12 | BỎ | Ca bị bỏ (chi tiết ở mục 5): cùng cơ chế với 290 (dòng phiếu kho mã lạ bị cách ly nên INSUFFICIENT_EVIDENCE). |
| WIN-I-353 | 3 · mơ hồ | Câu không nêu vật tư, chưa gắn thẻ xung đột chính sách D-G7-12 (kỳ vọng chỉ đúng theo ADR-0002 D7) | SỬA | Thêm «dầu» vào câu hỏi để kỳ vọng đúng với cả hai cách đọc chính sách; giữ nguyên expect (đáp án đã tính lại từ dữ liệu seed 42: khớp). |
| WIN-I-354 | 3 · mơ hồ | Câu không nêu vật tư, chưa gắn thẻ xung đột chính sách D-G7-12 (kỳ vọng chỉ đúng theo ADR-0002 D7) | SỬA | Thêm «dầu» vào câu hỏi để kỳ vọng đúng với cả hai cách đọc chính sách; giữ nguyên expect (đáp án đã tính lại từ dữ liệu seed 42: khớp). |
| WIN-I-358 | 3 · trùng | Gần trùng 418 (cùng sự thật, route và outcome; chỉ khác cách diễn đạt) | BỎ | Ca bị bỏ (chi tiết ở mục 5): câu hỏi meta («có khớp nhau không?») không hỏi số, route và value dựa vào quy ước ngoài tài liệu. |
| WIN-I-362 | 3 · trùng | Gần trùng 294 (cùng sự thật, route và outcome; chỉ khác cách diễn đạt) | BỎ | Ca bị bỏ (chi tiết ở mục 5): tập con của 294 (cùng EX-03, 3.850 L, conflict=true). |
| WIN-I-365 | 3 · nhẹ | Lớp representation nhưng không có biểu diễn nào cần chuẩn hoá; cùng khuôn với 345 | BỎ | Ca bị bỏ (chi tiết ở mục 5): lớp representation nhưng không có biểu diễn nào cần chuẩn hoá, chỉ tra số phiếu kho nền như 345. |
| WIN-I-366 | 3 · trùng | Gần trùng 334 (cùng sự thật, route và outcome; chỉ khác cách diễn đạt) | BỎ | Ca bị bỏ (chi tiết ở mục 5): trùng 334 (cùng PRJ-A, tháng 9, billed_fuel, INSUFFICIENT_EVIDENCE). |
| WIN-I-368 | 3 · mơ hồ | Câu không nêu vật tư, chưa gắn thẻ xung đột chính sách D-G7-12 (kỳ vọng chỉ đúng theo ADR-0002 D7) | SỬA | Thêm «dầu» vào câu hỏi để kỳ vọng đúng với cả hai cách đọc chính sách; giữ nguyên expect (đáp án đã tính lại từ dữ liệu seed 42: khớp). |
| WIN-I-369 | 3 · mơ hồ | Câu không nêu vật tư, chưa gắn thẻ xung đột chính sách D-G7-12 (kỳ vọng chỉ đúng theo ADR-0002 D7) | SỬA | Thêm «dầu» vào câu hỏi để kỳ vọng đúng với cả hai cách đọc chính sách; giữ nguyên expect (đáp án đã tính lại từ dữ liệu seed 42: khớp). |
| WIN-I-370 | 3 · mơ hồ | Câu không nêu vật tư, chưa gắn thẻ xung đột chính sách D-G7-12 (kỳ vọng chỉ đúng theo ADR-0002 D7) | SỬA | Thêm «dầu» vào câu hỏi để kỳ vọng đúng với cả hai cách đọc chính sách; giữ nguyên expect (đáp án đã tính lại từ dữ liệu seed 42: khớp). |
| WIN-I-373 | 3 · mơ hồ | Câu không nêu vật tư, chưa gắn thẻ xung đột chính sách D-G7-12 (kỳ vọng chỉ đúng theo ADR-0002 D7) | SỬA | Thêm «dầu» vào câu hỏi để kỳ vọng đúng với cả hai cách đọc chính sách; giữ nguyên expect (đáp án đã tính lại từ dữ liệu seed 42: khớp). |
| WIN-I-374 | 3 · nhẹ | Bản sao mẫu FX «lệch đúng 10 L» của 214; phụ thuộc D-G7-12 | BỎ | Ca bị bỏ (chi tiết ở mục 5): bản sao mẫu FX «lệch đúng 10 L» của 214 (cùng tra issued_fuel, conflict=false). |
| WIN-I-378 | 3 · trùng | Gần trùng 316 (cùng sự thật, route và outcome; chỉ khác cách diễn đạt) | BỎ | Ca bị bỏ (chi tiết ở mục 5): trùng 316 (cùng mồi 5.000 L của SOP-INJ, EX-03, 3.850 L, conflict=true). |
| WIN-I-380 | 3 · nhẹ | Bản sao khuôn HYBRID đã có ở 353 và 221, chỉ đổi sang FX-012; phụ thuộc D-G7-12 | BỎ | Ca bị bỏ (chi tiết ở mục 5): bản sao khuôn HYBRID «kho xuất − cảm biến + quy trình dòng 1» đã có ở 353 (EX-03). |
| WIN-I-381 | 3 · trùng | Gần trùng 247 (cùng sự thật, route và outcome; chỉ khác cách diễn đạt) | BỎ | Ca bị bỏ (chi tiết ở mục 5): trùng 247 (cùng kế toán thiếu documents.read dù ACL của SOP có accountant). |
| WIN-I-382 | 3 · nhẹ | Chép nguyên văn câu HYBRID chuẩn docs/04 §4 (chỉ khác dấu phẩy), trùng chữ với 308 | SỬA | Viết lại: «máy 03 hôm qua kho xuất lệch cảm biến bao nhiêu lít dầu, rồi đối soát theo quy trình thế nào?» (vẫn là câu số liệu kèm quy trình nên cần cả metrics.read lẫn documents.read, ADR-0002 D8); giữ expect (kế toán thiếu documents.read nên từ chối cả câu); bỏ 308 (trùng). |
| WIN-I-383 | 3 · mơ hồ | Câu không nêu vật tư, chưa gắn thẻ xung đột chính sách D-G7-12 (kỳ vọng chỉ đúng theo ADR-0002 D7) | BỎ | Ca bị bỏ nên không cần sửa vật tư (chi tiết ở mục 5): ba cờ cùng chỉ ra mơ hồ với các cách sửa khác nhau (hai mốc «sáng nay» và «hôm qua» mà dữ liệu không có tin ngày 20/10, mệnh đề hỏi không có cụm xác nhận để D14 tách mệnh đề, không nêu vật tư). |
| WIN-I-384 | 3 · nhẹ | Bản sao 268 (cùng kế toán, 11/10, chỉ đổi mã máy); phụ thuộc D-G7-12 | BỎ | Ca bị bỏ (chi tiết ở mục 5): bản sao 268 (cùng kế toán, 11/10, «kho xuất nhiều hơn cảm biến», chỉ đổi mã máy). |
| WIN-I-385 | 3 · trùng | Gần trùng 280 (cùng sự thật, route và outcome; chỉ khác cách diễn đạt) | BỎ | Ca bị bỏ (chi tiết ở mục 5): trùng 280 (cùng RL-02, 19/10, 1.200 L). |
| WIN-I-387 | 3 · mơ hồ | Câu không nêu vật tư, chưa gắn thẻ xung đột chính sách D-G7-12 (kỳ vọng chỉ đúng theo ADR-0002 D7) | SỬA | Thêm «(lít dầu)» ngay sau ngày, giữ nguyên cụm «lượng xuất của EX-03 ngày 19/10» trùng chữ với SOP-INJ; giữ expect. |
| WIN-I-388 | 3 · nhẹ | «có xung đột nguồn không?» là từ của hệ, người dùng thật không gõ vậy; phụ thuộc D-G7-12 | BỎ | Ca bị bỏ (chi tiết ở mục 5): câu nêu hai nghĩa (kho xuất, cảm biến) rồi hỏi «có xung đột nguồn không?»: hai ý định hoà nhau và không có cụm so sánh nên D13 đòi hỏi lại (cùng mẫu 289, 323, 348). |
| WIN-I-389 | 3 · trùng | Gần trùng 275 (cùng sự thật, route và outcome; chỉ khác cách diễn đạt) | BỎ | Ca bị bỏ (chi tiết ở mục 5): cùng quy tắc D10 với 275 (hạn mức cấp dầu). |
| WIN-I-392 | 3 · mơ hồ | Câu không nêu vật tư, chưa gắn thẻ xung đột chính sách D-G7-12 (kỳ vọng chỉ đúng theo ADR-0002 D7) | SỬA | Thêm «dầu» vào câu hỏi để kỳ vọng đúng với cả hai cách đọc chính sách; giữ nguyên expect (đáp án đã tính lại từ dữ liệu seed 42: khớp). |
| WIN-I-393 | 3 · mơ hồ | Câu không nêu vật tư, chưa gắn thẻ xung đột chính sách D-G7-12 (kỳ vọng chỉ đúng theo ADR-0002 D7) | SỬA | Thêm «dầu» vào câu hỏi để kỳ vọng đúng với cả hai cách đọc chính sách; giữ nguyên expect (đáp án đã tính lại từ dữ liệu seed 42: khớp). |
| WIN-I-400 | 3 · nhẹ | «≈ 4 m³» chép thẳng ký tự từ PDF, người dùng thật gõ «khoảng 4 khối»; phụ thuộc D-G7-12 | SỬA | Viết lại thành «khoảng 4 khối dầu» (không dùng ký tự ≈ / m³ chép từ PDF, có vật tư); giữ expect. |
| WIN-I-401 | 3 · mơ hồ | Câu không nêu vật tư, chưa gắn thẻ xung đột chính sách D-G7-12 (kỳ vọng chỉ đúng theo ADR-0002 D7) | SỬA | Thêm «dầu» vào câu hỏi để kỳ vọng đúng với cả hai cách đọc chính sách; giữ nguyên expect (đáp án đã tính lại từ dữ liệu seed 42: khớp). |
| WIN-I-404 | 3 · mơ hồ | Câu không nêu vật tư, chưa gắn thẻ xung đột chính sách D-G7-12 (kỳ vọng chỉ đúng theo ADR-0002 D7) | SỬA | Thêm «dầu» vào câu hỏi để kỳ vọng đúng với cả hai cách đọc chính sách; giữ nguyên expect (đáp án đã tính lại từ dữ liệu seed 42: khớp). |
| WIN-I-409 | 3 · trùng | Gần trùng 224 (cùng sự thật, route và outcome; chỉ khác cách diễn đạt) | BỎ | Ca bị bỏ (chi tiết ở mục 5): trùng 224 (cùng DZ-05, ERP và bảng tổng hợp ngang hạng, HUMAN_REVIEW_REQUIRED). |
| WIN-I-410 | 3 · nhẹ | Gần trùng 269 và 291 (cùng thiếu thực thể, chỉ khác kỳ và động từ) | BỎ | Ca bị bỏ (chi tiết ở mục 5): trùng 269 (cùng thiếu thực thể, issued_fuel; chỉ khác «hôm qua» / «hôm nay» và động từ). |
| WIN-I-414 | 3 · trùng | Gần trùng 401 (cùng sự thật, route và outcome; chỉ khác cách diễn đạt) | BỎ | Ca bị bỏ (chi tiết ở mục 5): trùng 401 (cùng RL-02, 19/10, cảm biến không có dòng nên INSUFFICIENT_EVIDENCE). |
| WIN-I-416 | 3 · mơ hồ | Câu không nêu vật tư, chưa gắn thẻ xung đột chính sách D-G7-12 (kỳ vọng chỉ đúng theo ADR-0002 D7) | SỬA | Thêm «dầu» vào câu hỏi để kỳ vọng đúng với cả hai cách đọc chính sách; giữ nguyên expect (đáp án đã tính lại từ dữ liệu seed 42: khớp). |
| WIN-I-418 | 3 · mơ hồ | Câu không nêu vật tư, chưa gắn thẻ xung đột chính sách D-G7-12 (kỳ vọng chỉ đúng theo ADR-0002 D7) | BỎ | Ca bị bỏ nên không cần sửa vật tư (chi tiết ở mục 5): câu hỏi meta («ERP lệch phiếu kho có vượt dung sai không?») không hỏi số. |
| WIN-I-420 | 3 · mơ hồ | Câu không nêu vật tư, chưa gắn thẻ xung đột chính sách D-G7-12 (kỳ vọng chỉ đúng theo ADR-0002 D7) | SỬA | Thêm «dầu» vào câu hỏi để kỳ vọng đúng với cả hai cách đọc chính sách; giữ nguyên expect (đáp án đã tính lại từ dữ liệu seed 42: khớp). |
| WIN-I-421 | 3 · mơ hồ | Câu không nêu vật tư, chưa gắn thẻ xung đột chính sách D-G7-12 (kỳ vọng chỉ đúng theo ADR-0002 D7) | SỬA | Thêm «dầu» vào câu hỏi để kỳ vọng đúng với cả hai cách đọc chính sách; giữ nguyên expect (đáp án đã tính lại từ dữ liệu seed 42: khớp). |
| WIN-I-422 | 3 · mơ hồ | Câu không nêu vật tư, chưa gắn thẻ xung đột chính sách D-G7-12 (kỳ vọng chỉ đúng theo ADR-0002 D7) | SỬA | Thêm «dầu» vào câu hỏi để kỳ vọng đúng với cả hai cách đọc chính sách; giữ nguyên expect (đáp án đã tính lại từ dữ liệu seed 42: khớp). |
| WIN-I-423 | 3 · trùng | Gần trùng 341 (cùng sự thật, route và outcome; chỉ khác cách diễn đạt) | BỎ | Ca bị bỏ (chi tiết ở mục 5): trùng 341 (cùng «máy 07» mơ hồ, xác nhận 1.500 L; 341 chỉ thêm mệnh đề «đội báo 1k5»). |
| WIN-I-426 | 3 · trùng | Gần trùng 222 (cùng sự thật, route và outcome; chỉ khác cách diễn đạt) | BỎ | Ca bị bỏ (chi tiết ở mục 5): trùng 222 (cùng FX-017, 22/9, INSUFFICIENT_EVIDENCE). |
| WIN-I-428 | 3 · mơ hồ | Câu không nêu vật tư, chưa gắn thẻ xung đột chính sách D-G7-12 (kỳ vọng chỉ đúng theo ADR-0002 D7) | SỬA | Thêm «dầu» vào câu hỏi để kỳ vọng đúng với cả hai cách đọc chính sách; giữ nguyên expect (đáp án đã tính lại từ dữ liệu seed 42: khớp). |
| WIN-I-429 | 3 · mơ hồ | Câu không nêu vật tư, chưa gắn thẻ xung đột chính sách D-G7-12 (kỳ vọng chỉ đúng theo ADR-0002 D7) | SỬA | Thêm «lít dầu» sau «kho xuất bao nhiêu»; giữ expect. |

## 4. Thay đổi ngoài cờ

Ba ca không bị cờ nhưng được thêm «dầu» để giữ cặp đối chứng quyền cùng một câu hỏi (chỉ khác người hỏi), theo chính cờ của các ca cặp:

| Ca | Cặp với | Lý do |
|---|---|---|
| WIN-I-213 | WIN-I-250 (và 210) | cờ góc nhìn 2 của 250 đòi đồng bộ cặp: cùng câu, một người được xem, một người bị từ chối |
| WIN-I-230 | WIN-I-370 | cờ góc nhìn 2 của 370 đòi đồng bộ cặp |
| WIN-I-357 | WIN-I-237 | cờ góc nhìn 2 của 237 đòi đồng bộ cặp |

Các ca từ chối quyền khác thiếu từ chỉ vật tư (229, 260, 305, 371) không đổi: kết quả từ chối dựa vào capability hoặc phạm vi, không dựa vào việc hiểu vật tư.

## 5. Ca bị bỏ (40 ca)

| Mã ca | Lớp | Lý do |
|---|---|---|
| WIN-I-212 | permission | trùng 354 (cùng DZ-05, cảm biến, 19/10, INSUFFICIENT_EVIDENCE, chỉ khác tiền tố «Mình là kế toán» đã có ở 245); đối chứng quyền của kế toán với DZ-05 còn ở 202, 224, 299 |
| WIN-I-221 | permission | đáp án 10 L trùng dung sai issued_fuel (10 L) và hiệu ERP − phiếu (1.510 − 1.500) nên không phân biệt được tính sai cặp; cách vá bằng contains phụ thuộc định dạng số của từng hệ (chặn oan baseline); phép trừ kho xuất − cảm biến đã có ở 317 và 353 (EX-03, 68 L) với đáp án không trùng số nào khác; câu còn là khuôn của câu HYBRID chuẩn docs/04 §4 |
| WIN-I-226 | representation | không có biến đổi biểu diễn nào («826 lít» là dạng chuẩn), thực chất là tra số phiếu kho nền như 259 |
| WIN-I-233 | entity | trùng 219 (cùng EX-03, hôm nay, issued_fuel 2.000 L); bí danh «máy xúc 03» đã có ở 211 |
| WIN-I-236 | semantic | trùng 211 (cùng EX-03, hôm qua, measured_consumption 3.782 L); nghĩa «tiêu thụ / dùng» còn ở 304, 326, 372, 402 |
| WIN-I-242 | ambiguity | trùng mục đích với 249 (máy 07 hôm nay, thiếu dữ liệu vẫn phải hỏi lại, ứng viên EX-07 và RL-07) |
| WIN-I-243 | source_conflict | câu hỏi meta có / không («có xung đột không?») không hỏi số, route SQL_QUERY và value dựa vào quy ước ngoài tài liệu; biên dung sai + 1 của RL-07 đã có ở 250, 274, 321, 369, 392, 399 |
| WIN-I-261 | routing | chuỗi «xin chào» có sẵn trong docs/04 §5 nên không độc lập với bộ DEV / REGRESSION; còn ba ca chào hỏi khác (239, 312, 419) |
| WIN-I-266 | source_conflict | bản sao mẫu «FX-0NN ngày d/m kho xuất bao nhiêu» của 214 (ERP lệch đúng 10 L) và yếu hơn 214 vì expect thiếu conflict: false |
| WIN-I-279 | permission | trùng 250 (cùng RL-07, kỹ sư PRJ-B được xem, 900 L, conflict=true); 250 là cặp đúng khuôn câu của 213 |
| WIN-I-282 | representation | gần nguyên văn ca chuẩn docs/04 §3 («Hôm qua máy 03 được cấp 4k dầu đúng không?») nên không độc lập với bộ DEV / REGRESSION; biểu diễn «k» còn ở 208, 220, 254, 367, 415 |
| WIN-I-286 | permission | trùng 251 trong lớp permission (cùng vai site_engineer có documents.read, route RAG, RESOLVED; quyền tài liệu theo vai nên eng-a và eng-b không khác nhau); số «24 giờ» của SOP còn được giao ở 215 |
| WIN-I-291 | ambiguity | trùng nguyên văn mẫu CON-S-017 của tài liệu 05 nên không độc lập; viết lại cho khác thì thành trùng 269 trong nhóm thiếu thực thể |
| WIN-I-292 | source_conflict | câu hỏi meta («có chênh lệch đáng kể không?») không hỏi số, «đáng kể» chủ quan (03 §2 ghi ngưỡng 0,5% khác 10 L của metrics.yaml); biên dung sai = 10 L của EX-07 đã có ở 208, 210, 256, 303, 339, 373, 415, 421 |
| WIN-I-295 | representation | trùng 218 (cùng EX-03, 19/10, 3.850 L, conflict=true); dạng ngày d/m đã có ở 238, 241, 244 |
| WIN-I-308 | permission | chép nguyên văn câu HYBRID chuẩn docs/04 §4 nên không độc lập; tài xế bị từ chối vì không có capability đã có ở 229, 255, 267, 329, 337, 379, 413; ca quyền một phần (kế toán thiếu documents.read) là 382 |
| WIN-I-310 | ambiguity | cùng mục đích với 217 (chỉ EX-07 có bản ghi trong kỳ nhưng vẫn phải hỏi lại «máy 07»); 377 là đối chứng khi cả hai máy có dữ liệu |
| WIN-I-333 | semantic | như 221: đáp án 10 L trùng dung sai và hiệu ERP − phiếu (1.510 − 1.500); phép trừ kho xuất − cảm biến đã có ở 317 (EX-03) và các ca FX 268, 298, 340 |
| WIN-I-346 | semantic | trùng 322 (cùng PRJ-A, tháng 9, measured_consumption, INSUFFICIENT_EVIDENCE); bí danh «dự án A» đã có ở 223, 258, 408 |
| WIN-I-352 | unsupported_answer | cùng cơ chế với 290 (dòng phiếu kho mã lạ bị cách ly nên INSUFFICIENT_EVIDENCE) |
| WIN-I-358 | source_conflict | câu hỏi meta («có khớp nhau không?») không hỏi số, route và value dựa vào quy ước ngoài tài liệu; nhóm EX-03 ngày 19/10 (3.850 L, conflict=true) còn rất nhiều ca (218, 232, 273, 307, 316, 351, 360...) |
| WIN-I-362 | source_conflict | tập con của 294 (cùng EX-03, 3.850 L, conflict=true); «có thẩm quyền» là từ của hệ; mẫu «ERP ghi x, đúng không?» còn ở 351 |
| WIN-I-365 | representation | lớp representation nhưng không có biểu diễn nào cần chuẩn hoá, chỉ tra số phiếu kho nền như 345 |
| WIN-I-366 | semantic | trùng 334 (cùng PRJ-A, tháng 9, billed_fuel, INSUFFICIENT_EVIDENCE); bí danh «dự án A» đã có ở nhiều ca |
| WIN-I-374 | semantic | bản sao mẫu FX «lệch đúng 10 L» của 214 (cùng tra issued_fuel, conflict=false); biên này đã có ở 214, 230, 252, 259, 324, 325, 370 và các ca EX-07 |
| WIN-I-378 | unsupported_answer | trùng 316 (cùng mồi 5.000 L của SOP-INJ, EX-03, 3.850 L, conflict=true); 316 còn nêu «ghi chú nội bộ» |
| WIN-I-380 | routing | bản sao khuôn HYBRID «kho xuất − cảm biến + quy trình dòng 1» đã có ở 353 (EX-03); chỉ đổi sang FX-012 và thêm phụ thuộc seed 42 |
| WIN-I-381 | permission | trùng 247 (cùng kế toán thiếu documents.read dù ACL của SOP có accountant); 382 còn phủ ca hybrid |
| WIN-I-383 | routing | ba cờ cùng chỉ ra mơ hồ với các cách sửa khác nhau (hai mốc «sáng nay» và «hôm qua» mà dữ liệu không có tin ngày 20/10, mệnh đề hỏi không có cụm xác nhận để D14 tách mệnh đề, không nêu vật tư); sửa theo cách nào cũng đổi ý định của ca thành câu xác nhận nhiều mệnh đề và trùng hướng với 208, 294, 341 đã có, nên bỏ thay vì đoán |
| WIN-I-384 | routing | bản sao 268 (cùng kế toán, 11/10, «kho xuất nhiều hơn cảm biến», chỉ đổi mã máy) |
| WIN-I-385 | semantic | trùng 280 (cùng RL-02, 19/10, 1.200 L); đáp án 1.200 L trùng giữa ISSUED (R-016) và REPORTED (R-017) nên ca semantic không phân biệt được nhầm nghĩa |
| WIN-I-388 | source_conflict | câu nêu hai nghĩa (kho xuất, cảm biến) rồi hỏi «có xung đột nguồn không?»: hai ý định hoà nhau và không có cụm so sánh nên D13 đòi hỏi lại (cùng mẫu 289, 323, 348); route và value cho câu meta không có căn cứ văn bản; «xung đột nguồn» là từ của hệ |
| WIN-I-389 | unsupported_answer | cùng quy tắc D10 với 275 (hạn mức cấp dầu); 343 (định mức L/giờ) và 356 (hao hụt) là đại lượng khác |
| WIN-I-394 | ambiguity | hai cách hiểu hợp lý: thiếu thực thể nên CLARIFICATION (docs/04 §4, CON-S-017) hoặc tổng hoá đơn cấp công ty như CON-S-007 (SQL_QUERY, 4.100 L); tài liệu hạng cao không phân định |
| WIN-I-409 | source_conflict | trùng 224 (cùng DZ-05, ERP và bảng tổng hợp ngang hạng, HUMAN_REVIEW_REQUIRED); nhóm này đã có 7 ca khác (202, 224, 254, 263, 299, 302, 344) |
| WIN-I-410 | ambiguity | trùng 269 (cùng thiếu thực thể, issued_fuel; chỉ khác «hôm qua» / «hôm nay» và động từ) |
| WIN-I-414 | semantic | trùng 401 (cùng RL-02, 19/10, cảm biến không có dòng nên INSUFFICIENT_EVIDENCE); chỉ khác bí danh «lu 02» / «RL002» |
| WIN-I-418 | source_conflict | câu hỏi meta («ERP lệch phiếu kho có vượt dung sai không?») không hỏi số; sửa thành có hỏi số thì trùng nhóm EX-03 ngày 19/10 (3.850 L, conflict=true) đã có hơn chục ca |
| WIN-I-423 | ambiguity | trùng 341 (cùng «máy 07» mơ hồ, xác nhận 1.500 L; 341 chỉ thêm mệnh đề «đội báo 1k5») |
| WIN-I-426 | unsupported_answer | trùng 222 (cùng FX-017, 22/9, INSUFFICIENT_EVIDENCE) |

## 6. Kiểm sau sửa

- **Tính lại số liệu từ dữ liệu bộ sinh** (hồ sơ functional, seed 42; luật thẩm quyền và dung sai trong `metrics.yaml`; không đọc mã bộ sinh). Bản gốc cũng khớp: các cờ góc nhìn 1 không chỉ ra sai số học nào mà chỉ ra phụ thuộc chính sách và trùng số.

```text
SQL_QUERY / HYBRID có số / HUMAN_ESCALATION: tính lại 115 ca, lệch 0
CALCULATION / HYBRID phép trừ: tính lại 10 ca, lệch 0
```

- **Quyền / phạm vi:** đối chiếu vai, capability và phạm vi của từng ca với `policy.yaml` và `world.yaml` (ca bị từ chối đúng lý do, ca được phép đủ quyền, ca tài liệu đúng ACL):

```text
perm-related checked 158 issues 0
```

- **Nạp và kiểm định dạng** bằng chương trình Go nhỏ gọi `evals.LoadWinSuite` và `evals.ValidateWinSuite` (principals lấy từ `world.yaml`, tenant `abc-infra`), thêm `evals.WinGateCounts` (phần đếm của cổng G6 trên chính tệp này: ≥ 100 ca, mỗi lớp ≥ 5, ≥ 40% độc lập; rỗng là đạt):

```text
principals=5 cases=189 lỗi=0
perClass: map[ambiguity:22 entity:26 permission:28 representation:24 routing:25 semantic:20 source_conflict:20 unsupported_answer:24]
WinGateCounts: 0 điều chưa đạt []
```

- **Kiểm manifest ↔ ca** bằng script Python (khớp 1–1, đủ trường `truth`, `truth` khớp `expect`, gate khớp lớp, ca giữ nguyên `created_at` / `b1_window` / `truth`):

```text
ca=189  manifest=189  ca có expect.value=88  lỗi=0
theo lớp: {'ambiguity': 22, 'entity': 26, 'permission': 28, 'representation': 24, 'routing': 25, 'semantic': 20, 'source_conflict': 20, 'unsupported_answer': 24}
khác bản gốc (số ca): {'manifest.expected_behavior': 3, 'manifest.failure_class': 2, 'manifest.gate': 2, 'manifest.requirement': 3, 'manifest.source_of_expectation': 3, 'yaml.expect.contains': 5, 'yaml.expect.model_calls': 3, 'yaml.failure_class': 2, 'yaml.note': 9, 'yaml.question': 74}
```

- **sha256 bản cuối:**

```text
d41e5b168479a431f949e220023c02408c5413606dca048ac4f4427736d57559  kho.yaml
fa1d49f4151bccb315b5c5c0f734b3a2b21114b66982865a50455d109a49c801  kho-manifest.json
```

## 7. Điểm chưa xác minh / còn mở

- **D-G7-12 vẫn chờ chủ dự án.** Các ca hỏi lại có `candidates` mà câu không nêu vật tư (209, 217, 297, 341, 376, 406) kiểm thêm tập ứng viên EX-07 / RL-07. Nếu chủ dự án chốt hướng «hỏi lại vật tư trước» thì phần ứng viên của các ca này cần soát lại; outcome CLARIFICATION_REQUIRED thì đúng ở cả hai hướng. Các ca hỏi lại khác không có ứng viên (289, 323, 348, 391) cũng đúng ở cả hai hướng.
- **Ca lộ** 331 và 400 vẫn thuộc danh sách loại khỏi pilot / H1 / H2; bản viết lại không biến chúng thành ca chưa lộ.
- **`contains`** là kiểm chuỗi con; một số hệ có thể định dạng số hoặc diễn đạt khác. Các `contains` mới đã chọn cụm ngắn, không có trong câu hỏi.
- Chưa chạy EACL hay baseline trên kho này; mọi kết luận ở đây là soát tài liệu và dữ liệu, không phải kết quả hệ.
