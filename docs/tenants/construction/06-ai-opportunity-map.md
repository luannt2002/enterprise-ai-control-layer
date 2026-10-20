# 06 — Bản đồ cơ hội AI: ranh giới tất định / xác suất (ABC Infrastructure)

> Tenant `abc-infra` (mã `CON`). Mọi ví dụ và con số là **giả lập**, không phải kết quả đo (CONVENTIONS §6).
> Liên quan: [05-problem-catalog](05-problem-catalog.md) · [07-failure-catalog](07-failure-catalog.md) · [`cases.yaml`](../../../tenants/construction/evals/cases.yaml).

## 1. Nguyên tắc

Với **mỗi tác vụ nghiệp vụ**, bảng dưới trả lời: tất định (code, SQL, luật) hay xác suất (model) hay hybrid (lõi tất định + một bước xác suất bị kiểm)? Vì sao? Sai thì hậu quả gì? Đi route nào?

- **Không dùng model nếu SQL, code hoặc luật giải được tốt hơn** (P2, P3). Số chính xác là việc của dữ liệu có cấu trúc và phép tính tất định.
- **Model chỉ nằm ở nơi bất định là bản chất** của bài toán: hiểu ngôn ngữ mơ hồ, diễn đạt tri thức tài liệu, giải thích, tóm tắt, sinh văn bản (kiến trúc §2.2).
- **Điều khiển tất định ≠ mọi thành phần tất định (P13).** Luật chạy trước; classifier xác suất (nếu cổng mở) chỉ dưới ngưỡng tin cậy và có đường lùi; không chắc thì hỏi lại.
- **Model đề xuất, hệ thống quyết (P11).** Đề xuất chuyển trạng thái, đề xuất mức sự cố là **hybrid**: model gợi ý, guard/luật quyết.
- Ba cột "Tất định / Xác suất / Hybrid" loại trừ nhau; `✔` đánh dấu cột đúng, `·` là không.

## 2. Bảng ranh giới (52 tác vụ)

| Mã | Task | Deterministic? | Probabilistic? | Hybrid? | Why | Failure consequence | route |
|---|---|---|---|---|---|---|---|
| OPP-01 | Tổng nhiên liệu theo chỉ số, dự án, kỳ | ✔ | · | · | SUM trên dữ liệu có cấu trúc; model không bao giờ là nguồn của con số | số sai tới giám đốc, quyết định sai | `SQL_QUERY` |
| OPP-02 | Tổng giờ máy theo máy và kỳ | ✔ | · | · | cộng chính xác theo thực thể + kỳ | chi phí ca máy sai | `SQL_QUERY` |
| OPP-03 | Số người có mặt theo phạm vi (ABC, nhà thầu phụ) | ✔ | · | · | truy vấn chấm công có lọc phạm vi | trả lương/ATLĐ dựa trên số sai | `SQL_QUERY` |
| OPP-04 | Tồn kho hiện tại kèm độ tươi | ✔ | · | · | bản ghi ERP + `freshness_check` | mua/xuất theo số cũ | `SQL_QUERY` |
| OPP-05 | Tổng hoá đơn (trước thuế, tổng thanh toán) | ✔ | · | · | lấy từ bản ghi có kiểu, không đọc chữ PDF | thanh toán sai | `SQL_QUERY` |
| OPP-06 | Số ngày đến hạn nghiệm thu | ✔ | · | · | phép tính ngày | trễ hạn không phát hiện | `CALCULATION` |
| OPP-07 | So tiêu thụ với định mức × giờ máy (phép tính) | ✔ | · | · | số thực tế + định mức đã trích → phép tính ghi công thức | kết luận vượt/không vượt sai | `CALCULATION` |
| OPP-08 | Derived fact chênh lệch giữa các chỉ số (issued − measured…) | ✔ | · | · | công thức cố định + fact đầu vào | che giấu chênh lệch cần điều tra | `CALCULATION` |
| OPP-09 | Chuẩn hoá số và đơn vị (`4k`, `4.000L`, `4 khối dầu`) | ✔ | · | · | luật + bảng quy đổi theo chất liệu | sai bậc độ lớn 1.000 lần | `WORKFLOW` |
| OPP-10 | Phân tích số bằng chữ tiếng Việt | ✔ | · | · | văn phạm số hữu hạn | tiến độ sai | `WORKFLOW` |
| OPP-11 | Chuẩn hoá ngày giờ và múi giờ | ✔ | · | · | quy tắc múi giờ tenant | ghi sai ngày/kỳ | `WORKFLOW` |
| OPP-12 | Chuẩn hoá mã phiếu, mã thiết bị | ✔ | · | · | mẫu mã của pack | tra sai phiếu | `WORKFLOW` |
| OPP-13 | Kiểm miền giá trị chéo trường (dầu so định mức × giờ máy) | ✔ | · | · | luật so sánh với ngưỡng | ghi số vô lý như số thật | `CLARIFICATION` |
| OPP-14 | Nhận dạng thực thể theo phạm vi (máy 03 của dự án nào) | ✔ | · | · | luật theo kênh/dự án; không chắc thì hỏi lại | gán sai máy/dự án | `CLARIFICATION` |
| OPP-15 | Khử trùng tin nhắn/báo cáo gửi nhiều nơi | ✔ | · | · | khoá khử trùng (dự án, ngày, chỉ số, người báo) | cộng đôi số liệu | `TOOL` |
| OPP-16 | Kiểm quyền RBAC/ABAC trước mọi lệnh và truy vấn | ✔ | · | · | quyền là việc của phần mềm | rò dữ liệu, hành động trái quyền | `REJECT` |
| OPP-17 | Lọc quyền trước khi dữ liệu vào context | ✔ | · | · | model không được thấy dữ liệu ngoài quyền | lộ lương/dự án khác | `REJECT` |
| OPP-18 | Chọn guard và chuyển trạng thái workflow | ✔ | · | · | engine sở hữu trạng thái (P11) | trạng thái sai, duyệt thiếu chứng từ | `WORKFLOW` |
| OPP-19 | Kiểm hạn mức duyệt theo vai | ✔ | · | · | bảng hạn mức do pack khai báo | duyệt vượt quyền | `WORKFLOW` |
| OPP-20 | Tách nhiệm vụ (người lập ≠ người duyệt) | ✔ | · | · | so sánh danh tính | gian lận | `REJECT` |
| OPP-21 | Kiểm tồn không âm khi xuất kho | ✔ | · | · | so sánh số | sổ kho âm | `TOOL` |
| OPP-22 | Đối chiếu ba bên PO – nhập kho – hoá đơn | ✔ | · | · | so khớp số lượng/giá trị | thanh toán sai | `WORKFLOW` |
| OPP-23 | Hoà giải trong cùng chỉ số theo thẩm quyền | ✔ | · | · | ma trận thẩm quyền; cùng hạng thì chuyển người | chọn nhầm nguồn | `HUMAN_ESCALATION` |
| OPP-24 | Tính chỉ số dẫn xuất (số ngày liên tiếp vượt định mức) | ✔ | · | · | mã cài; trigger chỉ so sánh | pack thành ngôn ngữ lập trình | `WORKFLOW` |
| OPP-25 | Luật trigger `metric op ngưỡng` (nhiệt độ, tồn kho, giờ máy) | ✔ | · | · | so sánh khai báo | bỏ sót hoặc báo thừa | `WORKFLOW` |
| OPP-26 | Mở phiếu bảo dưỡng từ sự kiện (khử trùng, không mở trùng) | ✔ | · | · | khoá khử trùng + so khớp phiếu mở | phiếu trùng | `WORKFLOW` |
| OPP-27 | Gán ngày làm việc cho ca đêm | ✔ | · | · | lịch ca là cấu hình | báo cáo gán sai ngày | `WORKFLOW` |
| OPP-28 | Idempotency của lệnh có tác dụng phụ | ✔ | · | · | khoá idempotency | tạo phiếu hai lần | `TOOL` |
| OPP-29 | Công bố phiên bản tài liệu nguyên tử | ✔ | · | · | một giao dịch | phiên bản nửa vời | `TOOL` |
| OPP-30 | Từ chối câu hỏi ngoài phạm vi | ✔ | · | · | luật phạm vi của tenant | tốn tiền, trả lời lan man | `REJECT` |
| OPP-31 | Nhắc và leo thang việc quá hạn | ✔ | · | · | timer của workflow | việc kẹt không ai biết | `TOOL` |
| OPP-32 | Tìm tài liệu theo mã chính xác (`BB-NT-0415`) | ✔ | · | · | tìm toàn văn trên token chính xác | bỏ sót tài liệu | `FULL_TEXT_SEARCH` |
| OPP-33 | Tra SOP/hợp đồng và trả lời có trích dẫn | · | · | ✔ | truy xuất tất định + diễn đạt bằng model + kiểm trích dẫn | trích dẫn sai/bản hết hiệu lực | `RAG` |
| OPP-34 | Trả lời cần cả số liệu lẫn tài liệu (thực tế so định mức) | · | · | ✔ | số từ SQL, quy định từ RAG, phép tính tất định, model chỉ diễn đạt | kết luận sai | `HYBRID_RETRIEVAL` |
| OPP-35 | Giải thích vì sao tiêu thụ dầu bất thường | · | ✔ | · | nguyên nhân là suy diễn; mọi con số phải có bằng chứng | giải thích bịa | `HYBRID_RETRIEVAL` |
| OPP-36 | Tóm tắt báo cáo ngày cho trưởng dự án | · | · | ✔ | số lấy từ fact; model diễn đạt, validator kiểm số | tóm tắt thiếu/sai số | `WORKFLOW` |
| OPP-37 | Tóm tắt phụ lục hợp đồng | · | · | ✔ | đọc tài liệu có trích dẫn; văn bản trong tài liệu là dữ liệu | làm theo chỉ thị tiêm vào | `RAG` |
| OPP-38 | Viết lại câu cho lịch sự (không dữ kiện doanh nghiệp) | · | ✔ | · | việc ngôn ngữ thuần; giữ nguyên con số của người dùng | thêm số lạ | `DIRECT_LLM` |
| OPP-39 | Dịch thông báo an toàn | · | ✔ | · | việc ngôn ngữ thuần | sai nghĩa thông báo | `DIRECT_LLM` |
| OPP-40 | Trích khe từ tin nhắn mơ hồ bằng model (chỉ khi cổng mở) | · | · | ✔ | luật chạy trước; model chỉ phần luật không bắt được và phải qua kiểm schema, miền giá trị, span gốc | ghi sai khe | `WORKFLOW` |
| OPP-41 | Phân loại ý định bằng model (chỉ khi luật trượt ngưỡng đo được) | · | · | ✔ | luật trước, classifier dưới ngưỡng tin cậy, không chắc thì hỏi lại | sai route | `CLARIFICATION` |
| OPP-42 | Đề xuất mức nghiêm trọng sự cố | · | · | ✔ | model đề xuất; luật mức tối thiểu và engine quyết | hạ mức sự cố nghiêm trọng | `WORKFLOW` |
| OPP-43 | Gợi ý "nên duyệt/nên từ chối" cho phiếu | · | · | ✔ | model chỉ gợi ý; guard quyết | duyệt thiếu chứng từ | `WORKFLOW` |
| OPP-44 | Trích bảng định mức từ PDF thành bản ghi có kiểu | · | · | ✔ | parser (ML) đề xuất cấu trúc; cổng chất lượng và code quyết | định mức sai lan vào phép tính | `TOOL` |
| OPP-45 | So khớp thực thể mờ (tên máy/người viết sai) | · | · | ✔ | gợi ý ứng viên có điểm; ngưỡng và quyết định là của code | gán nhầm thực thể | `CLARIFICATION` |
| OPP-46 | Soạn nháp công văn/phụ lục | · | ✔ | · | sinh văn bản; người duyệt trước khi gửi | gửi nội dung sai | `DIRECT_LLM` |
| OPP-47 | Tổng hợp báo cáo quản lý đa chiều (năm chiều) | · | · | ✔ | workflow cố định thu đủ năm chiều; model chỉ diễn đạt nhận định có bằng chứng | bỏ sót chiều, kết luận "không có vấn đề" | `WORKFLOW` |
| OPP-48 | Phát hiện bất thường so với hôm trước | · | · | ✔ | so sánh số tất định; giải thích bằng model | bỏ sót bất thường | `WORKFLOW` |
| OPP-49 | Gợi ý nguyên nhân hỏng máy từ sổ tay thiết bị | · | · | ✔ | RAG có trích dẫn; kỹ thuật viên quyết | sửa sai hạng mục | `RAG` |
| OPP-50 | Đọc ảnh phiếu kho (OCR/thị giác) | · | ✔ | · | cần thị giác máy; cổng đóng nên hỏi lại người gửi | ghi số sai từ ảnh mờ | `CLARIFICATION` |
| OPP-51 | Điều tra nhiều bước động ("kiểm tra toàn bộ dự án A") | · | ✔ | · | cần lập kế hoạch động; cổng AGENT đóng nên lùi về `management-briefing` | agent đi lạc, vượt ngân sách bước | `AGENT` |
| OPP-52 | Dự báo tiêu thụ nhiên liệu tháng sau | · | ✔ | · | cần mô hình dự báo, không có trong phạm vi; không bịa số | số dự báo như thật | `REJECT` |

## 3. Tổng kết

Số đếm từ bảng trên (số đếm tài liệu, không phải kết quả đo): **52 tác vụ** — **32 tất định**, **7 xác suất**, **13 hybrid**. Hệ quả:

| Nhận xét | Ý nghĩa cho kiến trúc |
|---|---|
| Hơn một nửa tác vụ hoàn toàn tất định | phần mềm bao quanh (chuẩn hoá, nhận dạng, quyền, workflow, validator) là phần lớn của hệ thống |
| Mọi tác vụ hybrid có một **bước kiểm** tất định đi kèm | validator (`number_in_evidence`, `citation_valid`, `schema_valid`…) hoặc guard của workflow |
| Tác vụ xác suất thuần chỉ là việc ngôn ngữ, đọc ảnh, dự báo, điều tra động | các việc cổng đóng (ảnh, agent, dự báo) có hành vi lùi an toàn ở [05 §10](05-problem-catalog.md) |
| Số chính xác không bao giờ đến từ model | luật cứng số 2 của luồng hỏi ([kiến trúc §4](../../01-architecture.md)) |

## 4. Cổng capability liên quan

| Tác vụ | Cổng (kiến trúc §9) | Mở khi | Trạng thái |
|---|---|---|---|
| OPP-40 trích khe bằng model | Trích xuất tin nhắn bằng model | luật trích xuất bỏ sót trên bộ ca bẩn và có validator chặn kết quả sai | đóng, chưa đo |
| OPP-41 phân loại ý định bằng model | Phân loại ý định bằng model | luật của pack trượt ≥ ngưỡng đo được trên bộ ca router | đóng, chưa đo |
| OPP-44, OPP-45 ML | Rerank / ML ở dịch vụ Python | có số đo đòi | đóng, chưa đo |
| OPP-50 đọc ảnh | Thị giác máy | có dữ liệu thật và ca hỏng mà context/tool không chữa được | đóng |
| OPP-51 điều tra động | Agent | có ca workflow cố định không giải được, đo được | đóng; dùng `management-briefing` |

Mỗi cổng đang đóng cần ít nhất một **ca âm** chứng minh cổng thật sự chặn (CLAUDE.md §19): ví dụ `CON-T-096` (cổng AGENT đóng → lùi `WORKFLOW`), `CON-T-033` (cổng thị giác đóng → hỏi lại).
