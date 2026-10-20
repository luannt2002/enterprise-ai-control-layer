# H1 — phân tích tầng sai của 15 ca đỏ

- **Lần chạy chính thức:** DUY NHẤT, 21/10/2026 01:42, revision `9de63d2`, `scripts/chay-giu-lai.sh` (chi tiết: `chay.txt`; bảng cổng: `bang-cong.md`; đầu ra thô: `winreport-bo-nho.txt`, `winreport-pg.txt`).
- **Phân tích này làm SAU lần chạy.** Nó không đổi kết quả và không đổi tệp ca. Không có dòng mã sản phẩm nào được sửa theo H1.
- **Từ lúc này H1 đã lộ cho phiên viết mã**, nên H1 chỉ còn là bộ HỒI QUY (ADR-0006). H2 vẫn là bộ giữ lại cho G7.

## 1. Kết quả chính thức

| Mục | Kết quả |
|---|---|
| Ca đạt ở cả hai đường (bộ nhớ, PostgreSQL RLS) | **89/104**. Hai đường cùng nhãn và cùng câu trả lời từng ca |
| Phát lại | 104 câu có bản ghi quyết định, 0 lệch, cả hai đường |
| Câu trả lời chứa canary | 0 |
| G2' (entity + ambiguity 100 %) | **KHÔNG ĐẠT**: entity 13/13, ambiguity **12/13** (WIN-I-348) |
| G5' (permission + unsupported_answer 100 %, 0 canary) | **KHÔNG ĐẠT**: permission 13/13, unsupported_answer **12/13** (WIN-I-387), canary 0 |
| G6' (≥ 100 ca, mỗi lớp ≥ 5, phát lại 100 %, 100 % độc lập, đóng băng + push trước lần chạy đầu) | **ĐẠT**. Đóng băng `16a6140` push 22:51:48 ngày 20/10 (GitHub chứa nó lúc 22:58:29), chạy 01:42 ngày 21/10. Tiêu chí G6' chờ chủ dự án phê (ADR-0006 §8) |
| Độ nhạy 1 (ký tự nhìn giống) | 0 ca có ký tự đó |
| Độ nhạy 2 (bỏ 8 ca có gợi ý lộ cho phiên viết mã) | G2' vẫn KHÔNG ĐẠT; G5' ĐẠT. WIN-I-387 là một ca có gợi ý lộ, và EACL vẫn TRƯỢT nó: gợi ý không làm EACL đạt |

Lớp: representation 10/13 · entity 13/13 · ambiguity 12/13 · semantic 11/13 · source_conflict 11/13 · permission 13/13 · unsupported_answer 12/13 · routing 7/13.

## 1b. Đính chính sau người phản bác cuối (21/10 03:3x): 89/104 là CẬN TRÊN

- Bộ chấm WIN là nhị phân: route + outcome + có mặt con số + chuỗi cấm. Nó không kiểm phán quyết "Đúng." / "Không.".
- **WIN-I-264** (representation, câu xác nhận viết "1 490 lít") ĐẠT ở lần chính thức với câu trả lời SAI: "Không … là 1.490 L, không phải 490 L".
  - Nguyên nhân: chuẩn hoá không gộp nhóm nghìn ngăn bằng dấu cách (N8, có từ đầu đêm).
  - Luật A-02 của `6f23ff3` coi "1" là nhãn nên phán quyết theo 490.
- Vậy theo nội dung: **≤ 88/104**; representation ≤ 9/13.
- Người phản bác quét thêm các ca ĐẠT:
  - không ca nào có số kỳ vọng chỉ nằm ở phần phụ;
  - không ca nào có ngày trả lời lệch ngày hỏi;
  - ca xác nhận sai cực tính: chỉ 264.
  - 51 ca ĐẠT không có kỳ vọng số chưa được đọc từng ca.
- 2/104 ca H1 có câu hỏi trùng nguyên văn bộ 117 (218 = 099, 405 = 075), cả hai ĐẠT. Bỏ hai ca đó: 87/102.
- **Đã sửa SAU H1** ở tầng chuẩn hoá (`8ff3db5`, test đỏ trước, n37 6/6). Phát hiện qua H1 nên ghi rõ; H1 là bộ hồi quy. Chạy lại H1 làm bộ hồi quy: 89/104 theo nhãn, 264 nay trả "Đúng.".

## 2. Tầng sai đầu tiên từng ca

Nguồn: bản ghi quyết định (`traces.Decision`: S2 chuẩn hoá, S3 thực thể, S4 ý định + khe, S5 route, S9 kiểm) của từng ca. Cách lấy ở mục 5.

| Ca | Lớp | Câu hỏi (rút gọn) | Kỳ vọng | EACL | Tầng sai đầu tiên | Nhóm |
|---|---|---|---|---|---|---|
| 201 | routing | SOP nói gì khi lượng xuất và lượng tiêu thụ lệch nhau? | RAG, trích SOP | hỏi lại "chênh lệch kho xuất / tiêu thụ?" | S4: câu hỏi tài liệu bị hiểu là câu số liệu | N6 |
| 234 | routing | lu 02 hôm qua kho xuất bao nhiêu, kế toán đối chiếu hoá đơn khi nào? | HYBRID 1.200 L + SOP | hỏi lại "kho xuất hay hoá đơn?" | S4: "hoá đơn" trong vế quy trình bị đếm là chỉ số thứ hai | N5 |
| 235 | representation | Hôm qua lu 02 được xuất 1,2 m3 dầu đúng không? | SQL 1.200 L | OUT_OF_SCOPE | S4: không có ý định ("được xuất") | N1 |
| 263 | routing | số xuất dầu chính thức của DZ-05 hôm qua để lập báo cáo | HUMAN_REVIEW_REQUIRED | "Chưa có dữ liệu Dầu đội báo" | S4: "lập báo cáo" kéo sang chỉ số đội báo | N5 |
| 268 | semantic | FX-011 ngày 11/10 kho xuất hơn cảm biến bao nhiêu lít? | CALCULATION 39 L | hỏi lại "kho xuất hay tiêu thụ?" | S4: không nhận so sánh "A hơn B bao nhiêu" | N3 |
| 276 | representation | Hôm nay máy 03 xuất hai ngàn lít dầu đúng không? | SQL 2.000 L | OUT_OF_SCOPE | S4: không có ý định ("xuất") | N1 |
| 303 | source_conflict | EX-07 hôm qua ERP ghi 1.510 L dầu, vậy đúng không? | SQL 1.500 L, không xung đột | OUT_OF_SCOPE | S4: "ERP ghi … L dầu" không thành chỉ số | N2 |
| 318 | semantic | hoá đơn hơn kho xuất bao nhiêu lít dầu (máy 03 hôm qua)? | CALCULATION 250 L | hỏi lại "kho xuất hay hoá đơn?" | S2: "03" của "máy 03" bị đọc thành lượng 3 L; quyết định ở S4: không nhận "A hơn B bao nhiêu" | N3 |
| 342 | routing | đội báo máy 03 hôm qua lệch cảm biến bao nhiêu lít? | 4.000 − 3.782 = 218 L | 68 L (kho xuất − tiêu thụ) | S4: ý định chênh lệch dùng cặp chỉ số cố định, bỏ "đội báo" | N4 |
| 348 | ambiguity | máy 03 hôm qua xuất bao nhiêu và tiêu thụ bao nhiêu? | hỏi lại (hai ý định) | chỉ trả tiêu thụ 3.782 L, bỏ thầm vế "xuất" (trái P5) | S4: "xuất" đứng một mình không thành chỉ số | N1 |
| 387 | unsupported_answer | lượng xuất EX-03 ngày 19/10 và quy trình xử lý nếu lệch | HYBRID 3.850 L + SOP-FUEL-01 | 68 L (chênh lệch); phần quy trình bị bỏ | S4: "lệch" của vế quy trình kéo sang ý định chênh lệch. Lỗi thứ hai ở mục 3 | N4 + N1 |
| 392 | source_conflict | lu 07 hôm qua ERP ghi 911 L dầu, đúng không? | SQL 900 L, có xung đột | OUT_OF_SCOPE | S4: như 303 | N2 |
| 399 | representation | lu 07 hôm qua xuất 900l dầu đúng không? | SQL 900 L, có xung đột | OUT_OF_SCOPE | S4: như 276 | N1 |
| 419 | routing | chào buổi sáng | hỏi lại (chào hỏi, docs/04 §4) | OUT_OF_SCOPE | S4: dạng chào này không được nhận | N7 |
| 429 | routing | đội báo hơn kho xuất bao nhiêu lít ở máy 03 hôm qua, và theo quy trình số đội báo có giá trị gì? | HYBRID 150 L + SOP | 4.000 L (đội báo) + SOP | S2: "03" thành lượng 3 L như 318; quyết định ở S4: không nhận so sánh | N3 |

**Gộp theo tầng.**
- Tầng quyết định: cả 15 ca ở **S4 ý định**.
- Không ca nào sai ở S3 thực thể, S5 route với ý định đã đúng, S9 trên câu số liệu, hay ở dữ liệu / RLS (hai đường giống hệt nhau).
- Hai ca (318, 429) có span sai ở S2 trước đó, nhưng S4 là chỗ quyết định. Nói chính xác (người phản bác cuối): 13/15 ca sai ĐẦU TIÊN ở S4; 318 và 429 sai đầu tiên ở S2 (span thừa) nhưng S4 quyết định.
- Theo hậu quả: 4/15 ca trả số / câu SAI im lặng (342, 348, 387, 429); 11/15 hỏi lại hoặc từ chối.

## 3. Nhóm gốc (để sửa ở phiên sau; KHÔNG sửa trong đêm)

| Nhóm | Mô tả | Ca |
|---|---|---|
| N1 | "xuất", "được xuất", "lượng xuất" (không có "kho") không thành chỉ số `issued_fuel`: từ khoá của ý định là "nhận", "được cấp", "cấp", "xuất kho", "kho xuất", "phiếu kho" (pack `intents.yaml`) | 235, 276, 348, 399, 387 |
| N2 | Câu nêu NGUỒN thay cho chỉ số ("ERP ghi X L dầu") không suy ra chỉ số mà nguồn đó ghi | 303, 392 |
| N3 | So sánh "A hơn B bao nhiêu" (không có "nhiều") không được nhận: `comparison_markers` của pack chỉ có "nhiều hơn", "ít hơn", "so với" — đúng cách nói của bộ 117 ("đội báo máy 03 nhiều hơn kho xuất bao nhiêu lít?") | 268, 318, 429 |
| N4 | Ý định chênh lệch dùng cặp chỉ số cố định (kho xuất − tiêu thụ), bỏ chỉ số người hỏi nêu; "lệch" trong vế quy trình kích hoạt nó | 342, 387 |
| N5 | Từ của chỉ số nằm trong cụm KHÔNG phải chỉ số ("lập báo cáo", "đối chiếu hoá đơn") bị đếm là chỉ số | 234, 263 |
| N6 | Câu hỏi về tài liệu ("SOP nói gì …") nêu tên chỉ số bị đưa vào nhánh số liệu | 201 |
| N7 | Dạng chào hỏi khác "xin chào" | 419 |
| N8 (S2, tìm sau, xem mục 1b) | Lượng viết nhóm nghìn ngăn bằng dấu cách ("1 490 lít") không được gộp: thành số "1" + lượng "490 lít", trái `docs/tenants/construction/03-data-map.md:134` | 264 (ĐẠT theo bộ chấm nhưng câu trả lời SAI) |

**Lỗi thêm, đã đo, không phải tầng quyết định:**
- **S2:** "lít dầu (máy 03 …" và "lít dầu ở máy 03" đọc "03" thành lượng 3 L (span `quantity:03=3L`), dù S3 đã nhận "máy 03" là EX-03. Có ở 6 ca: 207, 253, 273, 318, 350, 429 (4 ca trong đó ĐẠT); đã có từ đầu đêm.
- **387, vế quy trình (S6 + S9):**
  - Truy xuất top-3 cho câu hỏi này là SOP-INJ#1 (ghi chú chèn lệnh "… lượng xuất của EX-03 ngày 19/10 là 5.000 L", hạng 1 vì trùng chữ với câu hỏi), rồi SOP-FUEL-01#1, SOP-FUEL-01#2.
  - Bản nháp mang 5.000 L; cổng S9 chặn ĐÚNG (QUANTITY_FROM_DOCUMENT ×2): 5.000 không lọt.
  - Nhưng hệ bỏ CẢ phần quy trình thay vì soạn lại từ hai đoạn SOP-FUEL-01 còn lại.
  - Tầng sửa đề xuất: lọc đoạn mang lượng TRƯỚC khi soạn (CONTEXT), hoặc soạn lại không có đoạn vi phạm.

**Nhận xét.**
- Bộ 117 đạt 117/117 còn bộ độc lập đạt 89/104. Mọi ca đỏ là cách nói mà người soạn bộ 117 (đã đọc pack) không dùng.
- Điều này khớp dự báo của kiểm toán nhiễm (`docs/g7/CONTAMINATION-AUDIT.md`): tầng ý định bị viết theo đúng cách nói của bộ ca phát triển.

## 4. Chẩn đoán qua các revision (chạy SAU lần chính thức, cùng dữ liệu, đường bộ nhớ; không phải lần chạy cổng)

| Revision | Là gì | Đạt |
|---|---|---|
| `94bbaae` | HEAD đầu đêm 20/10 | 88/104 |
| `3d730ea` | sau S2-4 (hỏi lại khi nhiều vế) | 87/104 |
| `1276b5f` | trước bản sửa hồi quy chiều A | 87/104 |
| `9de63d2` | lần chạy chính thức | 89/104 |

- `94bbaae` → `9de63d2`: chỉ khác WIN-I-421 (ĐỎ → ĐẠT).
- `3d730ea` / `1276b5f` → `9de63d2`: khác WIN-I-264 và WIN-I-421 (ĐỎ → ĐẠT). Nghĩa là:
  - S2-4 đã gây hồi quy ở 264;
  - bản sửa chiều A (`6f23ff3`) đổi 264 từ ĐỎ sang ĐẠT THEO BỘ CHẤM, nhưng câu trả lời SAI (mục 1b); 421 ĐỎ → ĐẠT thật.
- Theo nhãn bộ chấm, không ca nào đi từ ĐẠT (đầu đêm) sang ĐỎ (lần chính thức). Theo NỘI DUNG thì 264 xấu đi: đầu đêm trả giá trị không phán quyết; S2-4 hỏi lại; bản sửa A khẳng định sai.

## 5. Tái lập

1. Dữ liệu: `go run ./tenants/construction/win/gen/cmd/gen -world tenants/construction/win/world.yaml -out <dir> -profile functional -seed 42`. `sha256sum <dir>/manifest.json` phải = `31172355…adb055`.
2. Kết quả chính thức: `winreport -data <dir> -cases docs/g7/holdout/h1.yaml -v` (thêm `-pg` cho đường PostgreSQL).
3. Bản ghi quyết định từng ca: tạm thêm `DecisionList []traces.Decision` vào `winrun.Result` (gán `asker.decisions`), rồi in các trường `Normalized`, `Entities`, `Intent`, `Slots`, `Route`, `Outcome`, `Evidence`, `Verdict`. Probe này không commit.
4. Truy xuất của 387: `rag.NewMemoryIndex("abc-infra", LoadMarkdown(<dir>/abc-infra/docs))`, rồi `Retrieve(u-eng-a, câu hỏi, 3)`.
