# Bên thứ ba của nhánh RAG — cái gì cần, khi nào, cần chủ dự án quyết gì

> **Viết cho:** chủ dự án — mọi thứ phụ thuộc bên ngoài (dịch vụ, model, API, phần mềm cài thêm) phải được nêu ở đây TRƯỚC khi
> dùng. **Ngày:** 23/10/2026. Giá và hạn mức của nhà cung cấp: **chưa tra / chưa đo** — tra lúc chọn, không lấy số trong tài liệu này.

## 1. Hiện trạng (đo bằng lệnh, 23/10)

- Ứng dụng (`cmd/eais`) **không gọi dịch vụ bên ngoài nào**: model gắn cứng là model giả tất định (`models.Extractive`,
  `cmd/eais/app.go:87`); truy hồi chạy trong Go trên PostgreSQL tự dựng.
- Mã DUY NHẤT gửi HTTP ra ngoài: `intelligence/models/messages.go` (gateway kiểu Anthropic Messages, cấu hình bằng biến môi
  trường `EAIS_MODEL_*`, khoá chỉ ở header). Hiện chỉ công cụ benchmark `g7run` dùng; gọi model trả tiền **chưa được cấp** (W-G7).
- Chương trình ngoài chạy trên máy: `pdftotext` (poppler) — đọc PDF chữ cho nạp báo cáo; `git` (g7run ghi bằng chứng).
- Thư viện Go trực tiếp: `pgx`, `yaml` (mã nguồn mở, không phải dịch vụ). Ảnh Docker: `postgres:17` ghim băm.

## 2. Danh sách

| # | Thành phần | Dùng cho | Hiện đã cần? | Mở khi nào (bằng chứng) | Phương án | Chủ dự án phải quyết / cung cấp | Dữ liệu ra ngoài? |
|---|---|---|---|---|---|---|---|
| 1 | **LLM sinh câu trả lời** (S8) | RAG-4 với model thật: diễn đạt câu trả lời từ đoạn đã truy hồi | **Có, để thử** — bước kế tiếp | ngay khi muốn đo trên model thật | (a) API Anthropic — giao thức đã có sẵn; (b) API chuẩn OpenAI hoặc server nội bộ chuẩn OpenAI — cần viết thêm một gateway nhỏ bằng Go; (c) server nội bộ nói giao thức Messages — dùng ngay | nhà cung cấp + tên model; khoá API (để ngoài repo); ngân sách số lượt; trần đồng thời; **chính sách dữ liệu** | **CÓ** với API: câu hỏi + đoạn tài liệu nội bộ được gửi đi |
| 2 | **Embedding** + phần mở rộng `pgvector` của PostgreSQL | RAG-6: tìm theo nghĩa khi người hỏi dùng chữ khác tài liệu | chưa | bộ S7 phát triển: 6/36 câu trượt vì đồng nghĩa — **phải xác nhận trên một bộ độc lập mới** | API embedding của một nhà cung cấp; hoặc model local qua dịch vụ Python sau ranh giới (CLAUDE.md §18). Phương án không cần bên thứ ba: từ điển đồng nghĩa trong pack (thử trước) | API hay local; nếu local: máy chạy (RAM / GPU); đồng ý đổi ảnh PG có `pgvector`; khi đổi model phải nhúng lại toàn kho | **CÓ** với API: toàn bộ nội dung tài liệu khi nhúng |
| 3 | **Rerank** (xếp lại đoạn) | RAG-6: khi đoạn đúng có trong top-20 nhưng không lên top-3 | **chưa** | hiện **không có bằng chứng**: các câu trượt là tìm không ra, không phải xếp sai | API rerank; hoặc cross-encoder local (Python) | như #2 | CÓ với API |
| 4 | **Đọc PDF chữ** — `pdftotext` (poppler) | RAG-1 / U5: nạp tài liệu tri thức dạng PDF | đã cài (dùng cho nạp báo cáo); chưa nối cho tri thức | khi có tài liệu PDF thật | chạy trên máy | đưa tài liệu thật để đo độ phủ chữ | không |
| 5 | **Đọc DOCX / XLSX** | RAG-1 / U5 | chưa | khi có tài liệu thật dạng Word / Excel | thư viện Go (`excelize` đã có gián tiếp cho XLSX); DOCX: chọn sau khi đo trên tệp thật (có thể là `docling`, Python) | đưa tài liệu thật | không (nếu chạy trên máy) |
| 6 | **OCR** | PDF scan, ảnh chụp tài liệu | chưa | chỉ khi tài liệu thật là bản scan | Tesseract trên máy; hoặc API OCR đám mây | có tài liệu scan không; API hay máy | CÓ với API |
| 7 | **LLM nhỏ** cho làm giàu đoạn / viết lại câu hỏi nối tiếp | RAG-6 | chưa | chỉ khi đo thấy: đoạn đúng không được tìm vì thiếu ngữ cảnh; hội thoại nhiều lượt trượt | như #1 | như #1 | như #1 |
| 8 | **Nhà cung cấp LLM dự phòng** | RAG-7: khi nhà cung cấp chính lỗi | chưa | sau khi có #1 và chạy thật | nhà cung cấp thứ hai, **đã thử thật** | chọn nhà cung cấp thứ hai | như #1 |

## 3. Luật

1. Không gọi dịch vụ bên ngoài, không cài phần mềm mới, không thêm thư viện cho chức năng RAG nếu chưa ghi ở bảng trên và chưa có
   lời đồng ý của chủ dự án.
2. Mặc định là model giả, chạy trên máy. Mỗi món ở bảng mở bằng một lỗi đo được (CLAUDE.md §19), không mở vì "phổ biến".
3. Gửi tài liệu nội bộ ra ngoài là quyết định về dữ liệu, không phải quyết định kỹ thuật.
4. Khoá API nằm ngoài repo; nhãn lỗi phân biệt `third_party:<tên>` với lỗi của mình.
