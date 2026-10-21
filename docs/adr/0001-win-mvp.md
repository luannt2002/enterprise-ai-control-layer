# ADR-0001 — Khoá phạm vi WIN MVP và các quyết định D1–D6

- **Ngày:** 20/10/2026
- **Trạng thái:** CHẤP NHẬN. D1–D6 được **tự chốt** khi chủ dự án cho phép chạy tự động không hỏi; chủ dự án **đảo được** bất cứ lúc nào bằng một ADR mới.
- **Liên quan:** `docs/04-core-mvp.md` (hợp đồng WIN), `docs/05-master-roadmap.md` (bản đồ có điều kiện), CLAUDE.md §19, §24, §34.
- **Xác nhận 20/10/2026 (tối):** chủ dự án giữ nguyên ADR này. Phạm vi WIN vẫn là Fuel Discrepancy Resolution; không mở COMMAND / WORKFLOW / EVENT / Agent / MCP / Python / framework chung trước khi đóng G7; không thêm tính năng chỉ để cải thiện benchmark. Riêng chữ "temperature 0" của D5 không chạy được với model hiện hành, nên có đề xuất sửa ở ADR-0005 (chờ duyệt).

## Bối cảnh

Kiến trúc đích (`01`) rộng: QUERY / COMMAND / EVENT, nhiều capability, nhiều tenant. Làm theo nó như danh sách việc thì dự án phình mà vẫn chưa chứng minh được lý do tồn tại.

Ngày 19/10, chủ dự án chốt: làm WIN trước, không làm quá rộng; đạt WIN rồi mới mở rộng theo roadmap. Hai lượt soát độc lập (20 + 24 phát hiện) đã chỉ ra các chỗ cần chốt để cổng WIN đo được.

## Quyết định

**Phạm vi.** Phạm vi WIN là đúng những gì `04` mô tả: chỉ QUERY, một bài toán Fuel Discrepancy Resolution, tenant A + canary B cùng ngành, bắt buộc có baseline RAG, 7 cổng G1–G7. Mọi thứ khác thuộc `05` và chỉ mở theo điều kiện ở đó.

| # | Câu hỏi | Quyết định | Lý do | Đảo khi |
|---|---|---|---|---|
| D1 | Ngưỡng 7 cổng | như `04` §6 | bộ ca cố định thì 100% là ngưỡng đúng cho luật tất định; G7 có biên rõ | chủ dự án đặt ngưỡng khác |
| D2 | ERP mang nghĩa gì | **nguồn hạng 2 của ISSUED** (khớp `tenants/construction/pack/metrics.yaml`); không tạo nghĩa "ghi nhận" riêng | ERP nhập lại cùng giao dịch xuất kho; tách nghĩa khi chưa có ca nào đòi là phình phạm vi | chủ nghiệp vụ xác nhận ERP ghi một sự kiện khác với xuất kho |
| D3 | "Dùng bao nhiêu dầu" | **CONSUMED** (`measured_consumption`) | nghĩa thông thường của "dùng"; khớp luật BR-FUEL-02 trong tài liệu tenant | chủ nghiệp vụ chốt khác |
| D4 | Đọc PDF | báo cáo ngày là PDF **văn bản**. Sinh bằng `go-pdf/fpdf` v0.9.0 + font DejaVuSans. **Đọc bằng `pdftotext` (poppler)** gọi từ Go sau interface `TextExtractor`. **Không Python.** Bảng PDF để sau WIN; G1 metamorphic dùng CSV / XLSX / JSON / tin nhắn | **Đo 20/10 08:44** (`scratchpad/pdf-spike`): thư viện Go thuần `ledongthuc/pdf` đọc **sai dấu tiếng Việt** ("Dự án Cầu A" → "Dñ án C§u A", "≈" → "H"); `pdftotext -layout` đúng 3/3 dòng. Python / torch nặng hơn mà không cần | máy chạy thiếu poppler; hoặc có thư viện Go đọc đúng ToUnicode |
| D5 | Công thức G7 | temperature 0; k = 3 lần, trung vị; mọi lớp EACL ≥ baseline; ≥ 4/8 lớp hơn ≥ 20 điểm %; EACL rò rỉ tenant = 0, claim không căn cứ = 0 | biên rõ, cho phép hoà, chống thắng do nhiễu | số đo cho thấy biên vô lý (ví dụ baseline đã đạt 100% một lớp) |
| D6 | Baseline | RAG đơn giản trong repo, chỉ dữ liệu tenant A, cùng retriever + model; **không có** tầng kiểm soát EACL | so công bằng; không cần DB / khoá của hệ cũ; lớp quyền so ở mức vai / phạm vi | chủ dự án muốn dùng hệ cũ làm baseline thứ hai |

## Hệ quả

- Lệnh, workflow, sự kiện, thể thao, agent, MCP, Python, kiểm tuân thủ định mức: **không làm** cho tới khi WIN đạt.
- Bộ ca WIN phải **đóng băng trước W1** (W0.5) để cổng không bị viết theo mã.
- Ca thể thao và ca COMMAND / WORKFLOW / EVENT đã có giữ làm đầu vào cho Phase 1–4.
- `phase: MVP` của sổ yêu cầu không còn nghĩa là WIN; phải xếp lại.

## Phương án đã cân nhắc

- **Bản 04 đề xuất 19/10 lúc 20:5x** (MVP gồm cả tác vụ duyệt, lệnh, đầu dò thể thao): bị thay vì rộng hơn mức cần để chứng minh luận đề.
- **Không có baseline:** bị loại vì chỉ chứng minh được "EACL chạy", không chứng minh được "EACL hơn".
