# Bên thứ ba của nhánh RAG — cái gì cần, khi nào, ổn định tới đâu, cần chủ dự án quyết gì

> **Viết cho:** chủ dự án — mọi thứ phụ thuộc bên ngoài (dịch vụ, model, API, phần mềm cài thêm) phải được nêu ở đây TRƯỚC khi
> dùng. **Cập nhật:** 23/10/2026 tối. Tài liệu này KHÔNG nêu tên nhà cung cấp, endpoint hay model: code chỉ biết GIAO THỨC, nhà cung
> cấp cụ thể nằm trong `.env` của máy chạy (bị git chặn; khuôn: `.env.example`). Giá và hạn mức: tra lúc chọn, không lấy số ở đây.

## 1. Hiện trạng (đo bằng lệnh, 23/10)

- **Ứng dụng (`cmd/eais`) vẫn không gọi dịch vụ bên ngoài nào**: model gắn cứng là model giả tất định; truy hồi chỉ từ khoá. Luật chi
  tiền (`docs/PROMPT-BUILD-EAIS.md` §9.3) và test cổng `TestKhongCoDuongGoiModelThat` giữ điều đó — mở là quyết định của chủ dự án
  (ghi ở `CHO`), không lẻn qua một cờ.
- **Động cơ đã có trong code, mặc định TẮT**, bật bằng biến môi trường:

| Vai trò | Giao thức | Mã | Bật bằng |
|---|---|---|---|
| LLM sinh câu trả lời (S8) | chat completions | `intelligence/models/chat.go` | `EAIS_MODEL_PROVIDER=chat_completions` + `EAIS_MODEL_*` |
| Embedding (truy hồi lai) | `/embeddings` | `intelligence/models/embed.go`, `intelligence/rag/semantic.go`, bảng `document_vectors` (0007) | `EAIS_EMBED_*` |
| Rerank (xếp lại nhóm đầu) | `/rerank` | `intelligence/models/embed.go` | `EAIS_RERANK_*` |

  Đường dùng chúng hôm nay: `evals/rag/dong_co_that_test.go` (bật bằng `EAIS_RAG_DONG_CO=1`) — cùng PG, cùng chỉ mục, cùng pipeline
  S0–S10 như ứng dụng, chỉ khác chỗ dựng.
- Mã khác gửi HTTP ra ngoài: gateway kiểu Messages có sẵn của team (`intelligence/models/messages.go`, công cụ benchmark G7).
- Chương trình ngoài chạy trên máy: `pdftotext` (đọc PDF chữ cho nạp báo cáo); `git` (g7run). Thư viện Go: `pgx`, `yaml`. Ảnh
  Docker: `postgres:17` ghim băm. Vector là mảng `real[]` thường — không cần phần mở rộng nào của PG.

## 2. Độ ổn định đo được của các dịch vụ đang thử (23/10)

Dữ liệu gửi đi: CHỈ bộ S7 giả lập và một văn bản pháp quy công khai.

| Vai trò | Chất lượng đo được | Ổn định đo được | Kết luận |
|---|---|---|---|
| LLM | S7 độc lập: dẫn đúng 90/90, từ chối đúng 14/14 câu không có đáp án, 0 từ chối nhầm. Văn bản pháp quy (23 câu có đáp án / 7 câu bẫy): 15/23 chứa đáp án, chặn 6/7 bẫy, từ chối nhầm 4 | 136 lượt gọi, **0 lỗi**; trễ trung vị 1,7 s, lớn nhất 10,9 s | Ổn để thử. Dịch vụ đang dùng đi qua tài khoản cá nhân, không phải API chính thức, không cam kết dịch vụ |
| Rerank | S7 độc lập top-3 90/92; văn bản pháp quy top-3 11/23; ngưỡng "biết im" chọn trên S7 phát triển chặn 6/7 bẫy, 0 chặn nhầm | 177 lượt gọi, **0 lỗi, 0 lần thử lại** | Ổn nhất. Giới hạn: hạn mức token miễn phí — ~30 đoạn mỗi câu (số token mỗi câu: chưa đo) |
| Embedding — API thứ nhất (bậc miễn phí) | S7 phát triển 36/36 (cả API riêng lẫn `/embeddings` chuẩn), S7 độc lập 91/92 (API riêng, đo bằng script) | **Không ổn**: tính hạn mức theo từng chữ; 3 lượt nhúng chết (lượt qua code Go: 429 suốt 10 lần thử, 758 s, chưa nhúng xong 73 chữ); gọi lẻ thì qua | **Đã thay** (23/10 tối). Bậc miễn phí còn dùng dữ liệu gửi lên để cải thiện sản phẩm |
| Embedding — API thứ hai (đang dùng, có tách kiểu câu hỏi / đoạn) | Qua code Go, đường lai + rerank + ngưỡng: S7 phát triển top-3 34/36; **S7 độc lập 83/92, 0/14 câu không có đáp án nhận đoạn**, 0 lọt quyền ⇒ dòng S7 ĐẠT; văn bản pháp quy 14/19 | nhúng 73 chữ 2 s, 94 chữ 2 s, 612 chữ 15 s; **0 lỗi** trong 3 lượt | Ổn. Giới hạn: hạn mức token miễn phí dùng chung với rerank (chưa đo số token đã tiêu) |

## 3. Danh sách

| # | Thành phần | Dùng cho | Trạng thái | Mở khi nào (bằng chứng) | Chủ dự án phải quyết / cung cấp | Dữ liệu ra ngoài? |
|---|---|---|---|---|---|---|
| 1 | **LLM sinh câu trả lời** (S8) | diễn đạt câu trả lời từ đoạn đã truy hồi | adapter có + test máy chủ giả; đo thật qua lệnh đo | ứng dụng: khi CỔNG TRƯỚC AI đạt + chủ dự án cho phép (§9.3) | nhà cung cấp + model; khoá (ngoài repo); ngân sách lượt; **chính sách gửi tài liệu ra ngoài** | **CÓ**: câu hỏi + đoạn tài liệu |
| 2 | **Embedding** | tìm theo nghĩa khi người hỏi dùng chữ khác tài liệu | adapter + truy hồi lai + bảng vector có; đo thật qua lệnh đo | bằng chứng: 6/36 câu S7 phát triển trượt vì đồng nghĩa ở đường từ khoá; chỉ vector (đo bằng script) 36/36 | API trả tiền hay model chạy trên máy (khi kho thật lớn, bậc miễn phí không đủ thông lượng); đổi model ⇒ nhúng lại toàn kho | **CÓ**: toàn bộ chữ của tài liệu khi nhúng + câu hỏi |
| 3 | **Rerank** | xếp lại nhóm đầu + ngưỡng "biết im" theo điểm | adapter có; đo thật qua lệnh đo | bằng chứng: văn bản pháp quy — ngưỡng điểm rerank chặn 6/7 bẫy so với 2/7 của luật từ khoá | như #2 | **CÓ**: câu hỏi + ~30 đoạn mỗi câu |
| 4 | **Đọc PDF chữ** (`pdftotext`) | nạp tài liệu tri thức PDF | đã cài (dùng cho nạp báo cáo); chưa nối cho tri thức | khi có tài liệu PDF thật | đưa tài liệu thật để đo độ phủ chữ | không |
| 5 | **Đọc DOCX / XLSX** | nạp tài liệu tri thức | chưa | khi có tài liệu thật dạng Word / Excel | đưa tài liệu thật | không (nếu chạy trên máy) |
| 6 | **OCR** | PDF scan, ảnh chụp tài liệu | chưa | chỉ khi tài liệu thật là bản scan | có tài liệu scan không; API hay máy | CÓ với API |
| 7 | **LLM nhỏ** cho làm giàu đoạn / viết lại câu hỏi nối tiếp | chất lượng truy hồi | chưa | chỉ khi đo thấy đoạn đúng không được tìm vì thiếu ngữ cảnh | như #1 | như #1 |
| 8 | **Nhà cung cấp dự phòng** | khi nhà cung cấp chính lỗi | chưa | sau khi #1–#3 chạy thật trong ứng dụng | chọn nhà cung cấp thứ hai, **đã thử thật** | như #1 |

## 4. Luật

1. Không gọi dịch vụ bên ngoài, không cài phần mềm mới, không thêm thư viện cho chức năng RAG nếu chưa ghi ở bảng trên và chưa có
   lời đồng ý của chủ dự án.
2. Mặc định là model giả, chạy trên máy. Mỗi món mở bằng một lỗi đo được (CLAUDE.md §19), không mở vì "phổ biến".
3. Gửi tài liệu nội bộ ra ngoài là quyết định về dữ liệu, không phải quyết định kỹ thuật.
4. Không tên nhà cung cấp, endpoint, model hay khoá trong code / comment / test / tài liệu: chỉ trong `.env`. Nhãn lỗi phân biệt
   `third_party:<giao thức>` với lỗi của mình.
