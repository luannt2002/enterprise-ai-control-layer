# Báo cáo đêm 20→21/10/2026 — khôi phục bằng chứng, xác định trạng thái WIN thật

> **BẢN NHÁP ĐANG CẬP NHẬT** (lần cuối: 21/10 01:5x, sau H1). Bản chốt viết lúc 07:00.
> Mục tiêu đêm, theo prompt cố vấn mà chủ dự án giao: khôi phục bằng chứng đáng tin, rồi mới xác định trạng thái WIN thật. Không tối ưu tỉ lệ đạt. FAIL sạch hơn PASS bẩn.
> Mọi số ở đây lấy từ lệnh đã chạy; chưa đo thì ghi **CHƯA ĐO**; ước lượng thì ghi **ƯỚC**.

## 0. Một dòng

**Trên bộ giữ lại độc lập H1 (104 ca, chạy MỘT lần lúc 01:42), mọi phần dựa trên ca của G1–G5 đều KHÔNG ĐẠT: 89/104.**
- Cả 15 ca đỏ sai ở tầng ý định (S4): EACL chỉ hiểu đúng cách nói của bộ phát triển.
- Bộ 117 đạt 117/117, nhưng do người đã đọc pack soạn.
- Phần hạ tầng ĐẠT:
  - nạp dữ liệu 2.739/2.739;
  - phát lại 104/104, lệch 0;
  - hai đường dữ liệu (bộ nhớ, PostgreSQL RLS) cho kết quả giống hệt;
  - 0 canary;
  - G6' ĐẠT (tiêu chí sửa đổi chờ chủ dự án phê).
- G7 **CHẶN**: chưa có khoá; bộ chấm v2 chưa đóng băng.
- Chi tiêu: 0 USD.

## 1. HEAD

| | Revision |
|---|---|
| START | `94bbaae` (push 20/10 19:26) |
| END | (điền lúc chốt) |

## 2. Trạng thái cổng THẬT

| Cổng | Trạng thái | Bằng chứng |
|---|---|---|
Tiêu chí: `docs/04` §6. Phần ca đo trên **H1** (bộ giữ lại độc lập, `docs/g7/holdout/ket-qua-h1/`), đạt khi đạt ở CẢ HAI đường dữ liệu. ADR-0006 chỉ đăng ký chứng minh lại G2 / G5 / G6. G1 / G3 / G4 từng ĐẠT trên bộ 117 (không nhiễm, nhưng do người đã đọc pack soạn). Cùng tiêu chí trên dữ liệu độc lập thì KHÔNG ĐẠT, và báo cáo này lấy dữ liệu độc lập làm trạng thái THẬT (D-H1-1).

| Cổng | Trạng thái THẬT | Bằng chứng |
|---|---|---|
| G1 DATA | **FAIL** (PARTIAL): nạp dữ liệu ĐẠT; phần ca `representation` H1 **10/13** | Nạp: 2.739/2.739 bản ghi khớp manifest, metamorphic 0 lệch; dữ liệu sinh lại giống từng byte. Ca đỏ 235, 276, 399: lượng trong câu hỏi được đọc ĐÚNG (1,2 m3, hai ngàn lít, 900l); sai ở S4 ("xuất" đứng một mình không thành chỉ số). Bộ 117: 100 % |
| G2 IDENTITY | **FAIL**: entity 13/13; ambiguity **12/13** | WIN-I-348 "máy 03 hôm qua xuất bao nhiêu và tiêu thụ bao nhiêu?" trả MỘT vế (tiêu thụ), bỏ thầm vế kia: đoán im lặng, trái P5 |
| G3 TRUTH | **FAIL**: semantic **11/13**, source_conflict **11/13** | 268, 318: so sánh "A hơn B bao nhiêu" không được nhận; 303, 392: "ERP ghi X L dầu" không thành chỉ số (OUT_OF_SCOPE). Bộ 117: 100 % |
| G4 DECISION | **FAIL**: routing **7/13** | 201, 234, 263, 342, 419, 429; tầng ý định (`phan-tich.md` mục 2). Bộ 117: 100 % |
| G5 CONTROL | **FAIL** (PARTIAL): phần ca permission 13/13, unsupported_answer **12/13**, canary 0. Phần cơ chế (AST, RLS, canary B, bảng "sai đúng một chiều", chèn lệnh) ĐẠT bằng test + đột biến | 387: trả chênh lệch 68 L thay vì lượng xuất 3.850 L. Con số 5.000 L của tài liệu chèn lệnh bị chặn đúng; nhưng phần quy trình bị bỏ. Độ nhạy: bỏ 8 ca có gợi ý lộ ⇒ G5' ĐẠT; 387 là ca có gợi ý và EACL vẫn trượt |
| G6 PROOF | **FAIL theo tiêu chí gốc** (bộ 117 băm 12 phút SAU lần chạy đầu). **G6' ĐẠT**, tiêu chí sửa đổi chờ chủ dự án phê:<br>- H1 104 ca, 100 % độc lập, 13 / lớp;<br>- đóng băng + push 22:51 trước lần chạy 01:42;<br>- phát lại 104/104, lệch 0;<br>- mọi danh mục đột biến hiện có bắt hết | `docs/g7/CONTAMINATION-AUDIT.md`, `docs/g7/holdout/ket-qua-h1/bang-cong.md` |
| G7 VALUE | **BLOCKED**:<br>- không có khoá model (D-G7-8);<br>- bộ chấm v2 chưa đóng băng được (ADR-0007);<br>- pilot cần khoá | `docs/g7/HOP-DONG-G7.md` §13 |

## 3. Nhiễm đã tìm ra (CONTAMINATION FOUND)

- **Bộ 117 ca** (`docs/g7/CONTAMINATION-AUDIT.md`): 102 sạch, 3 nhiễm (083, 107, 108), 12 không chắc.
  - Ba kỳ vọng bị phiên viết mã đổi lúc 09:09:26–35, sau lần chạy đầu (09:03:27, 76/117). Phiên chính đã đo lại 5 mốc giờ và nội dung 3 lần sửa: khớp.
  - Prompt soạn ca của phiên viết mã có 4 diễn giải không có trong hợp đồng ("nhận"; "đốt"; "bao nhiêu dầu" mặc định là kho xuất; canary đại trà trong chuỗi cấm). Đây là gốc lỗi ở ca 083.
  - ADR-0002 D7 ghi sai sự kiện về ca 107; đã thêm đính chính. `docs/04` dòng 83 trái D7 ⇒ câu chờ quyết D-G7-12.
- **Đêm nay, hai lần lộ bộ giữ lại mới** (cả hai đã xử lý TRƯỚC mọi lần chạy):
  1. **7 ca lộ cho người viết mã EACL.** Agent sửa pipeline chạy `grep` đệ quy, bộ lọc thư mục hỏng nên in ra 8 dòng; agent tự khai. 7 ca này bị loại TRƯỚC khi chia tập, sang `exposed.yaml` chỉ để báo cáo (`af4054a`, D-G7-13).
  2. **Gợi ý kỳ vọng của 15 ca lộ cho phiên chính.** Kết quả workflow soát kho hứa "chỉ số đếm" nhưng có trường chữ tự do. Tập đóng băng không đổi; bảng H1 báo độ nhạy khi bỏ 15 ca (`59ee556`).
- **Kiểm transcript mọi agent:** không agent nào khác đọc bộ giữ lại.

## 4. Bộ giữ lại độc lập (INDEPENDENT HOLDOUT CREATED)

- 229 ca do người không đọc mã soạn, có truth có cấu trúc.
- Ba người soát độc lập (số liệu / chính sách / độ khác nhau) gửi 252 cờ trên 119 ca. Người sửa không đọc mã xử lý: 175 sửa, 76 bỏ. Còn **189 ca**.
- Tính lại số liệu từ dữ liệu bộ sinh: 125 ca có số, lệch 0.
- **Đóng băng `16a6140`**, push 22:51:48, trước mọi lần chạy:
  - pilot 16, **H1 104 (13 / lớp)**, H2 62, exposed 7;
  - 0 lỗi định dạng; `--kiem` tính lại từ nguồn: KHỚP.
- Lúc 22:58:29, GitHub main đã chứa `16a6140`.
- **Giới hạn:** nội dung H2 đã công khai cùng bản đóng băng, nên bảo vệ H2 chỉ còn là quy trình (ADR-0006 §5).
- **H1 đã chạy MỘT lần** lúc 01:42 ngày 21/10 (`56101be`, revision mã `9de63d2`), bằng `scripts/chay-giu-lai.sh`, DB riêng `eais_giu_lai`: 89/104 ở cả hai đường. Phân tích tầng sai và chẩn đoán qua revision: `docs/g7/holdout/ket-qua-h1/phan-tich.md`.
  - Trên cùng dữ liệu H1, mã đầu đêm `94bbaae` đạt 88/104. Mã đêm nay không làm ca nào từ ĐẠT thành ĐỎ: S2-4 từng gây hồi quy 264, và bản sửa chiều A (`6f23ff3`, theo người soát không thấy bộ giữ lại) chữa lại.
  - Từ lúc này H1 chỉ còn là bộ hồi quy. H2 (62 ca) vẫn là bộ giữ lại cho G7.

## 5. Bộ chấm (SCORER BUGS FOUND / FIXED)

- **Bộ chấm v1 bị phá:**
  - 170 đòn của người tấn công độc lập: 109 câu sai mà ĐẠT, 24 câu đúng mà TRƯỢT;
  - một câu chỉ liệt kê mọi số của bản ghi đạt 73/117 ca.
- **Đã sửa trong v1:**
  - ratchet 170 đòn, nợ 122 → **118**;
  - rỗng / giả rỗng = từ chối (D-G7-9 mục 6, ADR-0005);
  - hệ lỗi khi hỏi luôn trượt;
  - số được phép riêng từng ca;
  - canary tên.
- **v2:** hướng thiết kế chốt ở mức đề xuất (ADR-0007, `b5df4c8`). **Không đóng băng được đêm nay**: lần nhìn đầu ngoài mẫu trượt oan 4,6 %, đạt nhầm 9,0 % (vượt ngưỡng 2 %); cần hiện thực (ƯỚC 10–15 ngày công) và hiệu chỉnh trên pilot thật với người chấm mù (D-G7-16).
- **Thước đo H1** (`scripts/bang-cong-h1.py`) cũng có lỗi, do soát lượt 2 tìm ra và đã sửa TRƯỚC khi chạy (`291d9bf`): G2' / G5' từng chỉ đọc đường bộ nhớ; phát lại từng ĐẠT với 0 câu.

## 6. Nhà cung cấp (PROVIDER ISSUES)

- Client cũ gửi `temperature: 0` ⇒ mọi lời gọi tới model hiện hành sẽ lỗi 400. Đã sửa (`750307a`): không gửi sampling mặc định; thinking / effort / thử lại lấy từ cấu hình.
- Lùi thử lại từng không có trần; huỷ ctx khi đang chờ từng không trả đúng lỗi. Đã sửa (`fe22e2a`).
- CHƯA ĐO trên API thật (không khoá).

## 7. Bảo mật (SECURITY FINDINGS)

- **Lượng nghiệp vụ trong tài liệu lọt CẢ HAI lớp kiểm** khi viết NBSP / ZWSP / toàn chiều rộng / ℓ / đơn vị đứng trước / đơn vị trong ngoặc: đã sửa (`5ce22ec`). Soát lượt 2 tìm thêm dạng còn lọt ("mét khối", "m^3", "(lít dầu)", "(4.000) lít", "lít/ca: 4.000"): đã sửa ở cả hai lớp (`6f23ff3`).
- **Biến mẫu thay nhiều lượt** có thể ghép thành số chưa từng được kiểm, kể cả số do model viết: đã sửa (`5ce22ec`).
- `g7run` từng ra PASS giả khi mọi lời gọi model của baseline lỗi: đã sửa (`a94b5fb`).
- Khoá: không tìm khoá. Chỉ dùng biến môi trường (D-G7-8). Kiểm 19:37: không có biến.

## 8. Lỗi tìm ra / đã sửa (BUGS FOUND / FIXED)

(bảng đầy đủ điền lúc chốt; nguồn: hai bản soát mã độc lập + đột biến + test mới)

## 9. Số test (TEST COUNTS)

224 (đầu đêm) → **313** đạt, 0 trượt, 0 bỏ qua (có PG, bản clone của `291d9bf`). (cập nhật lúc chốt)

## 10. Đột biến (MUTATION RESULTS)

- Chạy lại MỌI danh mục trên bản clone. Lượt `6b7d963`: 18 danh mục, 240 bị bắt + n26 7/7, 0 sống. `n27` (công cụ bộ giữ lại) 20/20.
- **Đột biến sống / chuỗi hỏng tìm thấy rồi sửa trong đêm:**
  - M23b, OR11, E3, BC8 (tương đương), BC9;
  - chuỗi S1 hỏng từ `a507e81`;
  - G35, PC6, PC9, PC10 (chuỗi đổi theo mã).

(cập nhật lúc chốt)

## 11. Độ sẵn sàng benchmark (BENCHMARK READINESS) — G7 chạy được bằng một lệnh chưa?

**Chưa.** Harness có đủ: bằng chứng thô, thử khói, pilot, trần tiền, nhận biết chạy khô, B1 / B2+ / B2″ / B2-oracle. Còn thiếu:
- khoá;
- bộ chấm v2 đóng băng (cần pilot thật + người chấm mù);
- dự toán chi phí theo bộ đóng băng (sau H1).

## 12. Lời gọi model và chi phí

- Lời gọi model thật: **0**. Chi tiêu: **0 USD**.
- Dự toán G7: (sau H1, chạy khô qua máy chủ giả; B1 là hệ đắt nhất).

## 13. Baseline công bằng

- **B2+ là baseline cổng:**
  - đoạn văn có bí danh + mã theo nguồn + ngày hiện tại;
  - đo được: trần oracle top-10 là 95/117 với B2, 114/117 với B2+ (`d5probe`, tái lập đúng số bản soát D5).
- **B1:** LLM trên bản ghi thô theo cửa sổ ngày + `entities.json`.
- **Chỉ báo cáo:** B2″ (luật miền viết thành chữ) và B2-oracle (đúng đoạn bằng chứng).
- **Mọi hệ** dùng cùng model, cùng tham số, cùng system prompt.

## 14. Commit (COMMITS)

41 commit tới `b5df4c8` (00:16). Danh sách đầy đủ: `git log 94bbaae..`. (cập nhật lúc chốt)

## 15. Phức tạp đã từ chối (REJECTED COMPLEXITY)

- Bộ chấm v2 bản 2 (giao thức JSON có kiểu cho mọi hệ): đổi đối tượng đo, không làm cổng (ADR-0007).
- Không đọc "từ X đến Y" thành một kỳ: hỏi lại thay vì thêm quy tắc chuẩn hoá mới (S2-4).
- Không công khai mã nguyên mẫu bộ chấm v2 (≈ 5,6 nghìn dòng) khi chưa soát.
- Không trích xuất lõi chung: `docs/v3/core-candidates.md` chỉ ghi nhận (CANDIDATE_ONLY).
- Không đổi tên repo / module (D-V3-03).
- Không implement Customer Ops khi G7 còn là đường găng (D-V3-09).

## 16. ADR mở (OPEN ADRS)

- ADR-0006 §8: G6' chờ phê.
- ADR-0007: ĐỀ XUẤT; chờ D-G7-15, 16, 17.
- ADR-0003: hướng đã duyệt, chưa implement.

## 17. Chặn cứng (HARD BLOCKERS)

1. Khoá model (D-G7-8): chủ môi trường đặt `EAIS_MODEL_API_KEY`.
2. Người chấm mù cho hiệu chỉnh bộ chấm v2 (D-G7-16).
3. D-G7-12: `04` dòng 83 hay ADR-0002 D7.

## 18. Mười câu hỏi

(điền lúc chốt)

## 19. Bằng chứng thô

- Công khai trong repo: `docs/g7/holdout/` (bản đóng băng + kết quả H1 khi có); `docs/g7/CONTAMINATION-AUDIT.md`; `docs/g7/REVIEW-D5.md`; `docs/g7/BO-CHAM-V2-DAC-TA.md`.
- Ở máy (gitignore):
  - `docs/sources/dem-2026-10-20/raw/` (Pha A);
  - `docs/sources/g7-scorer-v2-proto/` (mã nguyên mẫu, kết quả giám khảo).
