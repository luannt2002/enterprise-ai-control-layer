# Những gì chủ dự án cần quyết — gom một chỗ

> Cập nhật 20/10/2026 tối. **Một nguồn duy nhất** cho mọi câu đang chờ chủ dự án. Tài liệu khác chỉ trỏ về đây.
>
> **Cách trả lời** (một trong hai):
> - sửa cột "Trả lời" trong tệp này rồi commit;
> - nói trong phiên làm việc bằng mã câu, ví dụ: `D-G7-1: A; D-G7-2: sonnet; D-G7-3: trần 15 USD, đồng bộ`.
>
> Phiên làm việc nào mở ra cũng đọc tệp này trước. Câu đã trả lời chuyển xuống mục 0 kèm ngày.

## 0. Đã quyết

| Ngày | Câu | Quyết định của chủ dự án |
|---|---|---|
| 20/10 | ADR-0001 (phạm vi WIN, D1–D6) | **Giữ nguyên.** WIN vẫn là Fuel Discrepancy. Không mở COMMAND / WORKFLOW / EVENT / Agent / MCP / Python / framework chung trước khi đóng G7. Không thêm tính năng để cải thiện benchmark |
| 20/10 | ADR-0002 (D7–D17) | **Tạm chấp nhận**, có điều kiện; phân lớp từng quyết định ở ADR-0002 mục "Phân lớp" |
| 20/10 | Chi tiền / model thật | **Chưa cấp quyền.** Phải có hợp đồng G7 đóng băng, số lời gọi chính xác, ước tính chi phí trước |
| 20/10 | Đồng bộ git | Đã commit + push theo lệnh, kể cả `docs/v3/` (D-V3-08) |

## 1. G7 — chặn lần chạy model thật (làm trước)

Bối cảnh: `docs/g7/CHECKPOINT-A-H.md`. Hợp đồng đề xuất: `docs/g7/HOP-DONG-G7.md`. Chi phí: `docs/g7/CHI-PHI-G7.md`. Bản soát độc lập D5: `docs/g7/REVIEW-D5.md`.

| Mã | Câu | Lựa chọn | Khuyến nghị (vì sao) | Chưa quyết thì | Trả lời |
|---|---|---|---|---|---|
| **D-G7-1** | **D5 đòi temperature 0, nhưng model hiện hành trả lỗi 400 với mọi temperature khác mặc định.** Sửa D5 thế nào? (ADR-0005) | **A** model hiện hành, không gửi sampling, cố định thinking + effort, k = 3 + liệt kê ca chập chờn · **B** model đời cũ còn nhận temperature 0 · **C** như A nhưng k = 5 | **A**: kết luận đúng với model dùng thật; B đo trên đời model sắp ngừng; C tốn thêm khoảng 67% lời gọi mà ít thêm giá trị | G7 **không chạy được**: mọi lời gọi lỗi 400 | |
| **D-G7-2** | Model nào (chung cho mọi hệ) | `claude-haiku-5-5` · `claude-sonnet-5-5` · `claude-opus-5-5` | **Sonnet 5.5.** Model mạnh làm baseline mạnh, nên phép so khắt khe hơn với EACL; thắng nhờ baseline dùng model yếu thì ít giá trị. Chi phí ước tính vẫn nhỏ. Phiên V3 khuyên Haiku 5.5 vì rẻ (D-V3-01). Muốn xem độ nhạy theo model thì chạy cả hai, tổng < 15 USD | không chạy được | |
| **D-G7-3** | Trần chi phí + cách gọi | trần (USD) · đồng bộ (có độ trễ từng lời gọi) hay Batch API (rẻ hơn 50%, mất độ trễ) | **Trần 15 USD nếu Sonnet, 2 USD nếu Haiku; chạy đồng bộ.** Ước tính B1 + B2: Sonnet ≈ 5–10 USD, Haiku ≈ 0,3–0,6 USD (`CHI-PHI-G7.md` mục 5). Đồng bộ để có độ trễ thật; tiền chênh không đáng kể | không chạy được | |
| **D-G7-4** | Có so với **B1** (LLM trực tiếp, chữ thô) không? Luật cửa sổ ngày của B1? | có (cửa sổ = ngày của ca ± 1 ngày; ca theo tháng = cả tháng; bảng kỳ của từng ca do người không đọc mã gán) · không | **Có.** V3 R1 đòi cả B1 lẫn B2; không có B1 thì kết luận cao nhất chỉ là PARTIAL. B1 chưa có mã: viết sau khi duyệt, không cần model | G7 chỉ so với B2 |  |
| **D-G7-5** | Bộ ca **giữ lại**: EACL đã được sửa tới 117/117 trên chính bộ 117 ca | có: ≥ 40 ca mới, người không đọc mã soạn, kết luận G7 trên bộ này · không: kết luận trên 117 ca, ghi rõ thiên lệch | **Có.** Kết quả trên bộ dev nghiêng EACL; kết luận trên bộ chưa từng thấy mới đáng tin. Tốn thêm khoảng 34% lời gọi | kết luận trên 117 ca, kèm cảnh báo thiên lệch | |
| **D-G7-6** | Cách chấm theo **nội dung** (`G7ContentReport`), baseline top-k = 10, cách viết đoạn của baseline | duyệt · sửa | **Duyệt, kèm đọc `REVIEW-D5.md`.** Chấm đủ kỳ vọng route / outcome thì baseline 0/117 theo thiết kế, D5 không bao giờ đỏ. Top-k = 10 có số đo (`evals/baseline`) | không chạy được | |
| **D-G7-7** | **Duyệt đóng băng hợp đồng G7** (sau khi trả lời 1–6) | duyệt · sửa | duyệt; ghi sha256 + revision vào mục 13 của hợp đồng | không chạy được | |
| **D-G7-8** | Cấp khoá model (sau D-G7-7) | đặt biến môi trường `EAIS_MODEL_API_KEY` ở máy chạy; không qua repo, không qua chat | — | không chạy được | |

## 2. EACL V3 — gói định nghĩa R0

Do phiên lead V3 soạn. Chi tiết, mâu thuẫn C1–C19 và lý do: `docs/v3/05-R0-DEFINITION-FREEZE.md` mục 3–4. Dưới đây chép câu hỏi và khuyến nghị **của phiên đó**. Phiên viết tệp này **chưa soát độc lập** bộ `docs/v3/`.

| Mã | Câu | Khuyến nghị của phiên V3 | Ghi chú | Trả lời |
|---|---|---|---|---|
| D-V3-00 | Duyệt hướng V3: tên **EACL**, MVP chuyển sang **Customer Ops** (hiểu + hành động trên app tham chiếu), Fuel thành R1 | (ngầm) duyệt gói R0 | quyết định lớn nhất; các câu dưới phụ thuộc | |
| D-V3-01 | Model thật + trần chi phí cho R1 / R3 / R6 | Haiku 5.5; ước token trước khi chi | trùng D-G7-2 / D-G7-3 | |
| D-V3-02 | Ngưỡng các cổng V3 (§2.5) | duyệt hoặc sửa số | Fuel: giữ D5, áp cả B1; WIN / PARTIAL / FAIL | |
| D-V3-03 | Đổi tên mã và repo sang EACL | (a) tài liệu + README ngay; (b) module / thư mục / repo trong một commit riêng trước mã R2 | (b) đụng remote công khai | |
| D-V3-04 | Đo B0 (người làm tay) | quyết lúc chuẩn bị R6 | | |
| D-V3-05 | Agent con dùng Opus? | Không | đúng luật hiện hành | |
| D-V3-06 | Sửa `CLAUDE.md` §9 / §24 / §25 theo V3 | chủ dự án tự sửa theo câu chữ đề xuất | tệp của chủ dự án | |
| D-V3-07 | Việc treo: ADR-0002, đường dẫn nội bộ ở `docs/02`, danh sách P10, `go mod tidy` | dọn trước | ADR-0002 **đã tạm chấp nhận**. `docs/02`: chỗ đường dẫn monorepo đã thay; chỗ còn lại là đường dẫn tương đối bên trong mã cũ, cần cho bảng đối chiếu | |
| D-V3-08 | Commit / push `docs/v3/` | khi chủ dự án nói | **Đã làm** 20/10 tối theo lệnh "commit lên hết git" | — |
| D-V3-09 | Kiến trúc app tham chiếu (ADR-0003) | tiến trình riêng, DB riêng, REST `net/http`, cấm import hai chiều | | |
| D-V3-10 | Ghi đầu `04`, `05`, `catalog.yaml` là "lịch sử EAIS" | có | | |

## 3. Vận hành và repo

| Mã | Câu | Lựa chọn | Khuyến nghị (vì sao) | Trả lời |
|---|---|---|---|---|
| D-OP-1 | Thứ riêng tư không lên repo public: `docs/sources/`, `*.local.md`, memory | giữ ở máy · repo **riêng tư** riêng · chép qua kênh riêng (scp) | **Repo riêng tư riêng** nếu cần dùng ở máy khác; không thì giữ ở máy. Không bao giờ đưa vào repo public | |
| D-OP-2 | `go mod tidy` (máy gốc offline nên chưa chạy) | chạy ở máy có mạng, commit diff riêng | có. Chỉ sửa nhãn `// indirect`, không đổi bản | |
| D-OP-3 | Giấy phép cho repo public | Apache-2.0 · MIT · không giấy phép (mặc định giữ mọi quyền) | quyết trước khi chia sẻ rộng. Chưa chọn = người khác không có quyền dùng lại mã | |
| D-OP-4 | Chính sách commit / push của agent từ nay | chỉ khi được bảo (như hiện nay) · tự commit ở mốc checkpoint, push khi được bảo | giữ "chỉ khi được bảo" | |
| D-OP-5 | Hai việc dừng dở lúc 10:33: soát độc lập mã 20/10; bộ ca đội đỏ | chạy lại trước G7 · sau G7 · bỏ | **Trước G7**: soát độc lập không tốn model. Bộ ca đội đỏ chính là bộ giữ lại của D-G7-5 | |
| D-OP-6 | Nợ đặt chỗ D9: từ dừng / cụm nghi vấn tiếng Việt viết cứng trong `intelligence/rag` | giữ, có nhãn nợ · dời sang lexicon / pack trước khi đóng băng G7 | **Giữ.** Chưa có tenant ngôn ngữ khác; dời bây giờ đổi hành vi truy xuất ngay trước benchmark | |
| D-OP-7 | `CLAUDE.md` có danh sách nhóm lỗi P10 lệch `CONVENTIONS` §4.2 | chốt danh sách nào | chủ dự án chốt (tệp của chủ dự án) | |
