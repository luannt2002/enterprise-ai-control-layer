# Cổng QuânIT — việc 2: tra căn cứ đang hiệu lực khi xử lý chênh lệch nhiên liệu

> **Trạng thái:** ĐỀ XUẤT. Mã đang làm trên nhánh `rag/*` ở phiên điều phối (`KE-HOACH.md`), chưa gộp `main`.
>
> **Viết cho:** chủ dự án (duyệt hoặc bác từng câu chấm); team BUILD (đọc tham khảo, không sửa vùng RAG).
>
> **Ngày:** 22/10/2026. **Mốc đọc mã:** nhánh `rag/0-viec` dựng trên `32130f7`.
>
> **Nhãn:** [ĐO] tự chạy lệnh · [CODE] đọc trong mã · [REPO] tài liệu repo tự ghi · [SUY] suy luận · [NGOÀI] audit ngoài repo của chủ dự án (`docs/ROADMAP-RAG.md` §1).
>
> **Cách chấm:** giống `docs/viec/1-hanh-dong/CONG-QUANIT.md`. ✓ = tiêu chí đã viết ra và có nguồn (giấy) · ◐ = một phần · ✗ = thiếu người hoặc dữ kiện ngoài repo.
>
> **Dữ liệu:** mọi tài liệu, con số ở đây là giả lập (tenant `abc-infra` hư cấu). Mọi kết luận: **trên dữ liệu giả lập, chưa có người nghiệp vụ duyệt** (D-RM-03).

---

## 0. Kết luận

- **9 câu:** ✓ 5 (câu 2, 4, 5, 6, 7) · ◐ 2 (câu 3, 9) · ✗ 2 (câu 1, 8).
- Hai câu ✗ giống Phase 0b và Phase 1: chưa có người dùng thật, chưa có người ký độ chính xác (D-RM-03).
- **Không chặn việc build** theo D-RAG-00 + lời chủ dự án ngày 22/10 ("build cho tôi chuẩn cái RAG"). Mọi kết luận ghi "trên dữ liệu giả lập".

## 1. Việc được chấm

> Người có quyền đọc tài liệu hỏi EAIS **quy trình hoặc định mức đang hiệu lực** liên quan tới xử lý chênh lệch nhiên liệu. EAIS trả **đoạn trích của bản đã duyệt, đang hiệu lực vào ngày hỏi**, kèm mã tài liệu, phiên bản, vị trí. Không có bản hiệu lực, hoặc hai bản cùng hiệu lực, thì nói rõ chứ không chọn ngầm. Người không được đọc thì thấy như không có tài liệu.

- **Nguồn:** `docs/ROADMAP-RAG.md` §9 RAG-0; BR-DOC-01, 02, 03 (`docs/tenants/construction/01-business-map.md:415-417`) [REPO]; các ca CON-T-004, 005, 006, 008, 032, 061, 071, 081, 082, 094 (`tenants/construction/evals/cases.yaml`) [REPO].
- **Không thuộc việc này:** số vận hành (kho xuất, máy đốt), phép tính chênh lệch, tạo phiếu. Định mức làm đầu vào CALC (BR-FUEL-05) là U4b, cần phối hợp team MVP (`KE-HOACH.md`).
- **Hiện trạng** [ĐO `go test ./evals/rag/`]: 5 ca chạy thật ở mức truy hồi; 4 xanh, CON-T-004 đỏ (thiếu loại máy của EX-01, xem `HOP-DONG.md` §5).

## 2. Bảng tóm tắt

| # | Câu | Trả lời ngắn | Qua? |
|---|---|---|---|
| 1 | Giá trị thật | Người dùng giả định (kỹ sư công trường, quản lý dự án) cần căn cứ đúng bản khi xử lý chênh lệch; chưa có người thật, chưa có số nền | ✗ |
| 2 | Critical không | Có: trích bản hết hiệu lực hay bản nháp làm căn cứ xử lý là S1 (sai tiền, sai trách nhiệm) | ✓ |
| 3 | Dữ liệu chuẩn chưa | Có mô hình phiên bản / trạng thái / hiệu lực; chưa kiểm kê tài liệu thật, chưa biết tỉ lệ bản scan | ◐ |
| 4 | Phân loại | SOP đánh số, bảng định mức, biên bản, hợp đồng, bản scan: mỗi loại một cách đọc và cắt | ✓ |
| 5 | Phần nào không cần AI | Lọc tenant / quyền / phạm vi / phiên bản, chọn đoạn, kiểm trích dẫn: tất định. Model chỉ diễn đạt (mặc định trích nguyên văn) | ✓ |
| 6 | Gom ngữ cảnh gì | Danh tính (tenant, vai, phạm vi), ngày hỏi, tài liệu đã duyệt + đang hiệu lực, kèm quyền đọc | ✓ |
| 7 | Thiếu / mâu thuẫn | Không có → mẫu câu chung với "không được xem"; hai bản cùng hiệu lực → báo xung đột, chuyển chủ tài liệu | ✓ |
| 8 | Ai cam kết độ chính xác | Chưa có người nghiệp vụ (D-RM-03) | ✗ |
| 9 | Đồ có sẵn | So bằng lập luận với tìm kiếm toàn văn của Postgres và một dịch vụ RAG đã đo (nguồn E); chưa thử thật trên tài liệu của doanh nghiệp | ◐ |

## 3. Chi tiết từng câu

### Câu 1. Giá trị thật
- **Trả lời:** hôm nay EAIS trả lời "lệch bao nhiêu" nhưng không trả lời được "xử lý theo căn cứ nào, bản nào". Giá trị đề xuất đo: (a) tỉ lệ câu trả lời trích đúng bản hiệu lực; (b) số lần trích bản hết hiệu lực hay bản nháp, đích 0; (c) thời gian tra căn cứ so với tra tay.
- **Nguồn:** KPI "Tiêu hao so định mức" dùng "SOP đã trích", ghi "chưa đo" (`01-business-map.md:499`) [REPO].
- **Qua? ✗.** Để qua: có người nghiệp vụ và số nền của cách tra hiện tại.

### Câu 2. Critical không
- **Trả lời:** có. Trích bản cũ (định mức 16 thay vì 17 L/giờ) làm sai kết luận vượt định mức; trích bản nháp làm căn cứ chưa được duyệt; lộ tài liệu tenant khác là S1 an ninh.
- **Nguồn:** BR-DOC-01, BR-DOC-03 [REPO]; ca CON-T-005, 032, 061, 071 [REPO]; lỗi thật ở mã cũ: hỏi định mức trả cả "16 L/giờ" của bản đã hết hiệu lực [ĐO, test đỏ trước của U1].
- **Qua? ✓.**

### Câu 3. Dữ liệu chuẩn chưa
- **Trả lời:** mô hình tài liệu đã có phiên bản, trạng thái, hiệu lực, phạm vi; nạp sai khai báo thì báo lỗi [CODE `intelligence/rag/rag.go` `LoadMarkdown`]. Chưa kiểm kê tài liệu thật: bao nhiêu PDF chữ, bao nhiêu bản scan, bảng phức tạp cỡ nào.
- **Qua? ◐.** Để qua: kiểm kê kho thật (`ROADMAP-RAG.md` §11 dòng 1).

### Câu 4. Phân loại
| Loại | Cách đọc | Cách cắt |
|---|---|---|
| SOP đánh số, quy chế | markdown / `pdftotext` (ADR-0001) | mỗi bước / mỗi dòng một đoạn; U3b thêm đường dẫn mục |
| Bảng định mức | như trên | mỗi hàng một đoạn kèm header (U3, đã làm) |
| Hợp đồng, biên bản | như trên | theo Điều / mục (U3b) |
| Bản scan, bảng phức tạp | công cụ chuyên dụng, chọn bằng đo (U5) | như loại tương ứng sau khi đọc |
- **Qua? ✓** (giấy); mã mới có 2 dòng đầu.

### Câu 5. Phần nào không cần AI
- Lọc tenant / vai / phạm vi / phiên bản, chấm điểm, chọn đoạn, mẫu câu thông báo, kiểm trích dẫn: tất định [CODE `rag.go` `Retrieve`]. Model chỉ ở S8 của route RAG / HYBRID (`core/pipeline/ask.go`, hàm `answerKnowledge`) [CODE]; mặc định là model giả trích nguyên văn (brief §9.2 làn 0).
- **Qua? ✓.**

### Câu 6. Gom ngữ cảnh gì
- Danh tính đã xác thực (tenant, vai, `entity_scope`); ngày hỏi (giờ của tenant); tài liệu đã duyệt + đang hiệu lực mà người hỏi đọc được. Không lấy vai hay tenant từ prompt hay body.
- **Khoảng thiếu** [ĐO `tenants/construction/pack/policy.yaml`]: vai `site_supervisor`, `equipment_manager` mà các ca dùng không có trong policy; khi nối pipeline (U7), router sẽ chặn các vai này. Cần team bổ sung pack hoặc đổi vai trong ca.
- **Qua? ✓** (giấy).

### Câu 7. Thiếu / mâu thuẫn
- Không có tài liệu, hoặc có nhưng ngoài quyền: **cùng một** mẫu câu (`no_knowledge`: "Chưa có tài liệu bạn được xem trả lời được câu hỏi này.", `templates.yaml:34`) [CODE]; lý do thật chỉ ở vết.
- Không có bản hiệu lực: nói rõ, không dùng bản cũ làm căn cứ (CON-T-032). Hai bản cùng hiệu lực: báo xung đột, chuyển chủ tài liệu (BR-DOC-01, CON-T-061).
- **Qua? ✓.**

### Câu 8. Ai cam kết độ chính xác
- **Qua? ✗.** Chưa có người nghiệp vụ (D-RM-03).

### Câu 9. Đồ có sẵn
- So bằng lập luận: tìm kiếm toàn văn có sẵn của Postgres (`tsvector`) làm được chọn đoạn theo từ khoá nhưng không tự biết phiên bản / hiệu lực / quyền theo tài liệu; một dịch vụ RAG đã đo trên hệ dev (nguồn E) có đoạn biết tài liệu + bản nhưng tin tenant trong payload và không có quyền theo tài liệu [NGOÀI].
- **Qua? ◐.** Để qua: thử thật một giải pháp có sẵn trên cùng bộ ca (U8).

## 4. Cổng gốc (`docs/ROADMAP-DAY-DU.md` §4.2)

1. **Phase tiên quyết:** H4 đã chạy (104/119) [REPO `docs/STATUS.md`]. Đạt.
2. **Lỗi đo được / ca nghiệp vụ:** ca CON-T-005 đỏ ở mã cũ (trích bản hết hiệu lực) [ĐO]. Đạt.
3. **Ai sở hữu:** sự thật = bản tài liệu đã duyệt (chủ tài liệu của doanh nghiệp duyệt, EAIS chỉ ghi trạng thái); ai ghi = bộ nạp; ai đọc = người có `documents.read` + đúng phạm vi; vòng đời = `HOP-DONG.md` §3. Đạt trên giấy.
4. **Cổng nhị phân có ca âm:** `evals/rag` + 12 đột biến bị bắt [ĐO]. Đạt cho các nấc đã làm.
