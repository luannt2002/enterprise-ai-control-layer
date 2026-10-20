# Báo cáo đêm 20→21/10/2026 — khôi phục bằng chứng, xác định trạng thái WIN thật

> **BẢN NHÁP ĐANG CẬP NHẬT** (lần cuối: 21/10 01:5x, sau H1). Bản chốt viết lúc 07:00.
> Mục tiêu đêm, theo prompt cố vấn mà chủ dự án giao: khôi phục bằng chứng đáng tin, rồi mới xác định trạng thái WIN thật. Không tối ưu tỉ lệ đạt. FAIL sạch hơn PASS bẩn.
> Mọi số ở đây lấy từ lệnh đã chạy; chưa đo thì ghi **CHƯA ĐO**; ước lượng thì ghi **ƯỚC**.

## 0. Một dòng

**Trên bộ giữ lại độc lập H1 (104 ca, chạy MỘT lần lúc 01:42), mọi phần dựa trên ca của G1–G5 đều KHÔNG ĐẠT: 89/104 theo bộ chấm nhị phân, ≤ 88/104 theo nội dung.**
- Ca 264 ĐẠT với câu trả lời SAI; bộ chấm không kiểm phán quyết "Đúng." / "Không.". Người phản bác cuối tìm ra; đã sửa SAU H1 (`8ff3db5`).
- 15 ca đỏ đều quyết định sai ở tầng ý định (S4):
  - 13/15 sai đầu tiên ở S4; 2 ca có thêm span sai ở S2;
  - 4/15 trả số / câu sai im lặng, 11/15 hỏi lại hoặc từ chối.
  - Giả thuyết CHƯA đo riêng: tầng ý định khớp cách nói của bộ phát triển.
- Tiến độ, hai cách tính cạnh nhau:
  - theo D-H1-1 = A (khuyến nghị, chưa duyệt): **0/7** cổng đạt trọn, 1/7 nếu phê G6';
  - theo B (ADR-0006 §8 chỉ chứng minh lại G2 / G5 / G6): 3/7, 4/7 nếu phê G6'.
- Bộ 117 đạt 117/117, nhưng do người đã đọc pack soạn.
- Phần hạ tầng ĐẠT:
  - nạp dữ liệu 2.739/2.739;
  - phát lại 104/104, lệch 0;
  - hai đường dữ liệu (bộ nhớ, PostgreSQL RLS) cho kết quả giống hệt;
  - 0 canary;
  - G6' đạt các tiêu chí đo trực tiếp; hai tiêu chí chỉ đạt một phần (§2). Tiêu chí sửa đổi chờ chủ dự án phê.
- G7 **CHẶN**: chưa có khoá; bộ chấm v2 chưa đóng băng.
- Chi tiêu: 0 USD.

## 1. HEAD

| | Revision |
|---|---|
| START | `94bbaae` (push 20/10 19:26) |
| END | (điền lúc chốt) |

## 2. Trạng thái cổng THẬT

Tiêu chí: `docs/04` §6. Phần ca đo trên **H1** (bộ giữ lại độc lập, `docs/g7/holdout/ket-qua-h1/`), đạt khi đạt ở CẢ HAI đường dữ liệu. ADR-0006 chỉ đăng ký chứng minh lại G2 / G5 / G6. G1 / G3 / G4 từng ĐẠT trên bộ 117 (không nhiễm, nhưng do người đã đọc pack soạn). Cùng tiêu chí trên dữ liệu độc lập thì KHÔNG ĐẠT, và báo cáo này lấy dữ liệu độc lập làm trạng thái THẬT (D-H1-1 = A).
- Lập luận A: theo ADR-0006 §1, bộ 117 là bộ REGRESSION, nên chưa từng là bằng chứng độc lập cho G1 / G3 / G4. H1 là bằng chứng độc lập đầu tiên, và nó không xác nhận ba cổng này.
- Cách đọc B (ADR-0006 §8 chỉ đăng ký G2' / G5' / G6') cho 3/7.
- Cách đọc theo tầng sai đầu tiên: 15/15 ca đỏ thuộc tầng ý định / router (G4). Tầng chuẩn hoá của G1 chỉ lộ lỗi ở ca 264, ca bị bộ chấm che.

| Cổng | Trạng thái THẬT | Bằng chứng |
|---|---|---|
| G1 DATA | **FAIL** (PARTIAL): nạp dữ liệu ĐẠT; phần ca `representation` H1 **10/13** theo bộ chấm, **≤ 9/13** theo nội dung | Nạp: 2.739/2.739 bản ghi khớp manifest, metamorphic 0 lệch; dữ liệu sinh lại giống từng byte. Ca đỏ 235, 276, 399: lượng trong câu hỏi được đọc ĐÚNG (1,2 m3, hai ngàn lít, 900l); sai ở S4 ("xuất" đứng một mình không thành chỉ số). Ca 264 ĐẠT với câu trả lời SAI: "1 490 lít" không được gộp (trái `03-data-map.md:134`), lỗi tầng chuẩn hoá của chính G1; sửa sau H1 ở `8ff3db5`. Bộ 117: 100 % |
| G2 IDENTITY | **FAIL**: entity 13/13; ambiguity **12/13** | WIN-I-348 "máy 03 hôm qua xuất bao nhiêu và tiêu thụ bao nhiêu?" trả MỘT vế (tiêu thụ), bỏ thầm vế kia: đoán im lặng, trái P5 |
| G3 TRUTH | **FAIL**: semantic **11/13**, source_conflict **11/13** | 268, 318: so sánh "A hơn B bao nhiêu" không được nhận; 303, 392: "ERP ghi X L dầu" không thành chỉ số (OUT_OF_SCOPE). Bộ 117: 100 % |
| G4 DECISION | **FAIL**: routing **7/13** | 201, 234, 263, 342, 419, 429; tầng ý định (`phan-tich.md` mục 2). Bộ 117: 100 % |
| G5 CONTROL | **FAIL** (PARTIAL): phần ca permission 13/13, unsupported_answer **12/13**, canary 0. Phần cơ chế (AST, RLS, canary B, bảng "sai đúng một chiều", chèn lệnh) ĐẠT bằng test + đột biến | 387: trả chênh lệch 68 L thay vì lượng xuất 3.850 L. Con số 5.000 L của tài liệu chèn lệnh bị chặn đúng; nhưng phần quy trình bị bỏ. Độ nhạy: bỏ 8 ca có gợi ý lộ ⇒ G5' ĐẠT; 387 là ca có gợi ý và EACL vẫn trượt |
| G6 PROOF | **FAIL theo tiêu chí gốc** (bộ 117 băm 12 phút SAU lần chạy đầu). **G6'** (tiêu chí sửa đổi, chờ chủ dự án phê):<br>- ĐẠT, đo trực tiếp: H1 104 ca, 13 / lớp; đóng băng + push 22:51 trước lần chạy 01:42; phát lại 104/104 lệch 0 ở hai đường; 100 % người không viết mã soạn (nhãn + kiểm transcript người soạn);<br>- ĐẠT MỘT PHẦN: "mỗi cổng có đột biến bị bắt" (bộ sinh `win/gen` và `core/decimal` chưa có danh mục); "test thuộc tính" phủ 2/3 thuộc tính của `04` §6 ("A không thấy B" là test xác định `TestRLSCanaryB`);<br>- 2/104 ca có câu hỏi trùng nguyên văn bộ 117 (218, 405; cả hai ĐẠT). Bỏ ra thì 87/102 | `docs/g7/CONTAMINATION-AUDIT.md`, `docs/g7/holdout/ket-qua-h1/bang-cong.md` |
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
  - Trên cùng dữ liệu H1, mã đầu đêm `94bbaae` đạt 88/104.
  - Theo NHÃN bộ chấm, mã đêm nay không làm ca nào từ ĐẠT thành ĐỎ.
  - Theo NỘI DUNG, ca 264 xấu đi: đầu đêm trả giá trị không phán quyết; S2-4 hỏi lại; bản sửa chiều A (`6f23ff3`) khẳng định SAI, bộ chấm không bắt. Đã sửa SAU H1 (`8ff3db5`).
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

Theo nguồn phát hiện. "Đã sửa" nghĩa là có test đỏ trước khi sửa, xanh sau khi sửa, và đột biến cho nhánh mới.

| Nguồn | Tìm ra | Xử lý |
|---|---|---|
| Phiên chính, đọc tài liệu nhà cung cấp | client gửi `temperature: 0` ⇒ model hiện hành lỗi 400 | sửa `750307a` |
| Soát mã độc lập lượt 1 (21:22) | S1-1 g7run ra PASS giả khi mọi lời gọi baseline lỗi; S2-1/2/3 lượng tài liệu lọt + ghép số qua nhiều lượt thay; S2-4 cắt thầm vế đầu; S2-5, S2-6, S3-2, S3-4, S4; test hổng | sửa `a94b5fb`, `fe22e2a`, `5ce22ec`, `3d730ea` |
| Người tấn công bộ chấm (170 đòn) | v1: 109 câu sai mà ĐẠT, 24 câu đúng mà TRƯỢT | ratchet `82baa71`; sửa phần chắc `6b7d963` (nợ 122 → 118); v2 thiết kế ADR-0007, chưa đóng băng |
| Soát lượt 2, chiều B (công cụ bộ giữ lại) | 15 xác nhận. Ví dụ: G2' / G5' chỉ đọc đường bộ nhớ; phát lại 0 câu vẫn ĐẠT; CRLF lọt; chạy lại được nhiều lần | sửa `291d9bf` TRƯỚC H1 |
| Soát lượt 2, chiều A (pipeline) | 12 xác nhận + 3 người phản bác tìm thêm. Phần lớn là hồi quy của chính `5ce22ec` / `3d730ea` | sửa `6f23ff3` TRƯỚC H1 |
| Soát lượt 2, chiều C (harness G7) | 19 phát hiện, 3 bị bác (C-08, C-09, C-12). Ví dụ: C-06 G7 PASS được với bộ chấm v1 hỏng; C-18 khoá theo chuyển hướng HTTP; C-04 tốn tiền trước khi kiểm điều kiện | sửa C-03…C-07, C-10, C-11, C-13, C-17, C-18, C-19 (`d4b80be` … `50b1499`). C-01, C-02, C-14, C-15, C-16 thành ca thử bắt buộc của v2 (§12 đặc tả); v1 không vá vì `d4b80be` chặn mọi kết luận G7 khi bộ chấm chưa đóng băng |
| Soát bản sửa chiều A (sau H1, người soát không thấy bộ giữ lại) | 12 phát hiện. Đo lại xác nhận: R-01, R-02 (S1), R-05, R-06 (S2), R-07, R-10 là HỒI QUY do `6f23ff3`, tức câu xác nhận trả "Đúng."/"Không." cho sai kỳ hoặc dải số. R-03, R-04 (S1), R-08, R-11 có sẵn từ đầu đêm | (đang sửa: workflow, kèm hai người soát đối kháng) |
| Đột biến | sống: M23b, OR11, E3, BC9; tương đương: BC8; chuỗi gốc hỏng: S1, G35, PC6, PC9, PC10, FC2, AG | thêm test hoặc sửa chuỗi, chạy lại đến khi bắt hết |
| Phiên chính | ký tự vô hình THẬT trong test (11 ký tự) và trong một thông điệp commit (chưa push) | `7be1593`; thông điệp sửa trước khi push |
| H1 (sau lần chạy) | 15 ca đỏ, 7 nhóm gốc ở tầng ý định | KHÔNG sửa trong đêm (ADR-0006 §7); D-H1-2 |

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
- Dự toán G7 (`docs/g7/COST-PLAN.md`, đo bằng chạy khô pilot + H1; token là ƯỚC):
  - hệ cổng, lượt chính k = 3: kỳ vọng 12,74 USD (tối thiểu 10,02 · xấu nhất 27,40);
  - thêm một lần chạy lại trọn lượt: 23,44; k = 5: 19,82 (bản đầu ghi 16,36: tính sai thành chạy lại k = 1; soát C2, F-09);
  - B1 chiếm gần hết chi phí;
  - D-G7-18.

**8 điều kiện của quyền chi 15 USD** (quyết định chủ dự án 20/10, mục PAID EXECUTION AUTHORIZATION). Kiểm 21/10 02:12:

| # | Điều kiện | Trạng thái | Bằng chứng |
|---|---|---|---|
| 1 | `EAIS_MODEL_API_KEY` có trong môi trường | **KHÔNG** | Kiểm có / không, không in giá trị, lúc 19:37 và 02:12. Không tìm khoá ở nơi khác (D-G7-8) |
| 2 | Hợp đồng FROZEN | **KHÔNG** | Hợp đồng §13: bộ chấm v2 chưa đóng băng (ADR-0007); pilot cần khoá; tập lượt chính chờ D-H1-3 |
| 3 | Dự toán ≤ 15 USD | CHỈ khi hệ cổng, k = 3, không chạy lại (12,74 kỳ vọng). Chưa chắc: token chưa đo | `COST-PLAN.md` |
| 4 | Pilot + lượt chính trong cùng trần | như dòng 3 (đã gồm pilot k = 5). Trần là của từng lần chạy; luật cộng dồn ở hợp đồng §9 | `9de63d2` (C-07) |
| 5 | Model chính là Sonnet 5.5 | ĐẠT về cấu hình: hợp đồng §4; `g7run` không ra kết luận khi model / thinking / effort lệch hợp đồng | `d4b80be` (C-06) |
| 6 | Baseline và EACL cùng model | ĐẠT: một gateway chung cho mọi hệ | `g7run` build |
| 7 | Lưu bằng chứng thô | ĐẠT: `-out` (gồm bản ghi quyết định của EACL) | `a1a5b6b`, `50b1499` |
| 8 | Không có secret trong đầu ra / git | ĐẠT trong mọi lần chạy khô:<br>- test kiểm khoá không có trong `-out`;<br>- client không theo chuyển hướng HTTP, chỉ https hoặc loopback;<br>- quét riêng tư trước mỗi lần push | `TestSmokeThroughFakeProvider`, `d4b80be` |

⇒ **Paid G7 = BLOCKED** (điều kiện 1 và 2 không đạt). Không gọi model, không chi tiền.

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

1. **Giờ biết gì mà tối qua chưa biết?**
   - Trên dữ liệu độc lập, EACL đạt 89/104 theo bộ chấm nhị phân (≤ 88/104 theo nội dung), và phần ca của MỌI cổng G1–G5 trượt.
   - 15 ca đỏ đều quyết định sai ở tầng ý định (13/15 sai đầu tiên ở đó); 4/15 trả lời sai im lặng.
   - Bộ chấm WIN nhị phân có thể cho ĐẠT một câu xác nhận trả lời SAI (ca 264): nó không kiểm "Đúng." / "Không.".
   - Bộ chấm G7 v1 không dùng được: 109 câu sai mà vẫn ĐẠT.
   - Client cũ không gọi được model hiện hành: `temperature 0` bị trả 400.
   - G7 đo bằng chạy khô: chỉ hệ cổng, k = 3, tốn khoảng 12,74 USD (kỳ vọng); B1 chiếm gần hết chi phí.
   - Chính các bản sửa trong đêm có gây hồi quy, và chỉ người soát / người phản bác độc lập bắt được:
     - S2-4 làm ca 264 hỏi lại;
     - bản sửa chiều A làm 264 khẳng định sai, và gây R-01, R-02 (đã sửa: `8ff3db5`, bản sửa R).
2. **Điều gì đã chứng minh độc lập?**
   - G1 phần nạp dữ liệu.
   - G6' phần đo trực tiếp: bộ giữ lại đóng băng và push trước lần chạy, người soạn không viết mã, phát lại 104/104 lệch 0, hai đường dữ liệu giống hệt.
   - G6' phần chưa đủ: "mỗi cổng có đột biến" (thiếu bộ sinh, `core/decimal`); test thuộc tính phủ 2/3.
   - Người phản bác cuối xác nhận độc lập:
     - đóng băng trước, chạy một lần;
     - không dòng mã sản phẩm nào sửa theo gợi ý đã lộ (commit của agent giống từng byte với bản đã đưa vào main);
     - tầng sai của 15 ca;
     - chẩn đoán qua revision;
     - độ nhạy.
   - Trên H1: 0 rò canary, quyền 13/13, thực thể 13/13.
   - Phần cơ chế của G5: test + đột biến.
3. **Điều gì từng nói quá?**
   - "3/7 cổng đạt vô điều kiện (G1, G3, G4)" và "6/7 có điều kiện".
     - Bộ 117 ca không bị sửa kỳ vọng, nhưng do người đã đọc pack soạn, nên 117/117 là kết quả trong mẫu.
     - Theo ADR-0006 §1, bộ 117 là bộ REGRESSION, chưa từng là bằng chứng độc lập.
   - Tối qua chưa biết rằng bộ chấm WIN có thể cho ĐẠT một câu trả lời sai (ca 264).
   - "G7 chỉ chờ khoá": thực ra bộ chấm v1 hỏng và client lỗi 400.
   - ADR-0002 D7 ghi sai sự kiện ở ca 107 (đã đính chính).
4. **Cái gì trượt?**
   - H1: 15/104 ca (representation 3, ambiguity 1, semantic 2, source_conflict 2, unsupported_answer 1, routing 6).
   - Bộ chấm v1.
   - Bản đầu của công cụ chấm H1 (sửa trước khi chạy).
   - Vài bản sửa của chính đêm nay (mục 8).
5. **Vì sao trượt?**
   - Đo được: tầng ý định không có cách nói của 15 ca. Ví dụ:
     - "xuất" đứng một mình không thành chỉ số;
     - `comparison_markers` chỉ có "nhiều hơn", "ít hơn", "so với", nên "A hơn B bao nhiêu" không được nhận.
   - GIẢ THUYẾT chưa đo riêng: từ khoá được viết theo đúng cách nói của bộ ca phát triển (do người đã đọc pack soạn). EACL vẫn đạt nhiều cách nói không có trong bộ 117 (ví dụ "dc cấp bn dầu", ngày dạng 2026-10-19).
   - Chuẩn hoá chưa gộp nhóm nghìn ngăn bằng dấu cách (ca 264).
   - Bộ chấm v1 là "túi số": không ràng buộc con số với thực thể / kỳ / nguồn.
   - Sửa nhanh theo phát hiện cục bộ thì hỏng ở chỗ khác. Ví dụ, luật "số trần cạnh một lượng là nhãn" chữa A-02 nhưng gây R-01, R-02.
6. **Cố ý KHÔNG xây gì?** (mục 15)
   - Bộ chấm v2 bản 2; công khai mã nguyên mẫu v2.
   - Trích lõi chung; đổi tên repo; Customer Ops.
   - Sửa mã theo H1.
   - Chế độ `-redact`, sổ chi tiêu cộng dồn tự động (thay bằng luật quy trình).
   - Vá bộ đếm v1 cho C-01 / C-02.
7. **Fuel WIN đã đạt chưa?** **CHƯA.**
   - Theo D-H1-1 = A (khuyến nghị, chưa duyệt): 0/7 cổng đạt trọn; 1/7 nếu phê G6'.
   - Theo B (ADR-0006 §8): 3/7; 4/7 nếu phê G6'.
   - G7 chưa chạy lần nào.
8. **G7 chạy được bằng một lệnh chưa?** **CHƯA.**
   - Thiếu khoá.
   - Bộ chấm v2 chưa hiện thực (ƯỚC 10–15 ngày công) và chưa hiệu chỉnh trên pilot với người chấm mù.
   - Tập của lượt chính chờ D-H1-3; trần chi phí chờ D-G7-18.
   - Harness đã đủ: bằng chứng thô có bản ghi quyết định, thử khói, pilot, trần ngân sách bắt buộc, nhận biết chạy khô, dừng trước lời gọi model khi bị chặn.
9. **EACL đang ở giai đoạn nào?**
   - Vẫn là WIN MVP (Fuel), chưa đạt: đang ở bước "khôi phục bằng chứng → sửa tầng ý định", trước G7.
   - Gói định nghĩa EACL V3 R0 chờ duyệt D-V3-01…10.
10. **Một việc kế tiếp giá trị nhất?**
    - Soạn và đóng băng bộ giữ lại mới H3: người không đọc mã soạn, ≥ 13 ca / lớp, đóng băng bằng cam kết băm.
    - Làm việc này TRƯỚC khi sửa tầng ý định theo 7 nhóm gốc (dùng H1 làm bộ hồi quy).
    - Không có H3 thì không chứng minh lại được G1–G5 sau khi sửa, và G7 cũng không có tập hợp lệ (D-H1-2, D-H1-3).

## 19. Bằng chứng thô

- Công khai trong repo: `docs/g7/holdout/` (bản đóng băng + kết quả H1 khi có); `docs/g7/CONTAMINATION-AUDIT.md`; `docs/g7/REVIEW-D5.md`; `docs/g7/BO-CHAM-V2-DAC-TA.md`.
- Ở máy (gitignore):
  - `docs/sources/dem-2026-10-20/raw/` (Pha A);
  - `docs/sources/g7-scorer-v2-proto/` (mã nguyên mẫu, kết quả giám khảo).
