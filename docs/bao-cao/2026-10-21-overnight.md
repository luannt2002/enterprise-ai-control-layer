# Báo cáo đêm 20→21/10/2026 — khôi phục bằng chứng, xác định trạng thái WIN thật

> **Historical artifact** (báo cáo đêm 20→21/10/2026).

> **Bản chốt** (sáng 21/10, END mã `15ed135`; tài liệu chốt ở commit kế tiếp, cùng lần push). Đã qua người phản bác cuối ba góc (phương pháp 03:2x; số liệu ↔ bằng chứng và nói quá / nói non 05:2x); mọi đính chính của họ đã đưa vào bản này.
> Mục tiêu đêm, theo prompt cố vấn mà chủ dự án giao: khôi phục bằng chứng đáng tin, rồi mới xác định trạng thái WIN thật. Không tối ưu tỉ lệ đạt. FAIL sạch hơn PASS bẩn.
> Mọi số ở đây lấy từ lệnh đã chạy; chưa đo thì ghi **CHƯA ĐO**; ước lượng thì ghi **ƯỚC**; số chỉ khai trong phiên mà không lưu bằng chứng thì ghi **khai trong phiên**.

## 0. Một dòng

**CHƯA WIN. Trên bộ giữ lại độc lập H1 (104 ca, chạy MỘT lần lúc 01:42), mọi phần dựa trên ca của G1–G5 đều KHÔNG ĐẠT: 89/104 theo bộ chấm nhị phân, ≤ 88/104 theo nội dung. G7 CHẶN. Chi 0 USD.**
- Ca 264 ĐẠT với câu trả lời SAI ("không phải 490 L"); bộ chấm nhị phân không kiểm phán quyết "Đúng." / "Không.". Sửa SAU H1 ở tầng chuẩn hoá (`8ff3db5`): END trả "Đúng. … 1.490 L"; H1 làm bộ hồi quy trên END: 89/104, 0 ca đổi nhãn so với lần chạy chính thức. Hai bản thay thế để đóng các lỗ còn lại của `8ff3db5` đều bị người soát độc lập bác (`fd4d19c` sinh số mồ côi — đã revert; `3066044` nuốt số của tên thiết bị — không vào main) ⇒ nhóm nghìn còn lỗi đã đo (C3-02 S2, C3-05 S3), câu chờ quyết D-OP-9.
- **Câu xác nhận ("… đúng không?") là chỗ yếu nhất.** Trên 15 câu thăm dò do người soát dựng, đo lại ở năm mốc: đầu đêm `94bbaae` phán quyết sai 8/15, END phán quyết sai **15/15**. 7 câu là HỒI QUY của chính các bản sửa đêm nay (`3d730ea` 1, `6f23ff3` 4, `a8b0858` 2), 8 câu sai từ trước. Bảng cuối mục 8; câu chờ quyết D-OP-10.
- 15 ca đỏ của H1 đều quyết định sai ở tầng ý định (S4): 13/15 sai đầu tiên ở S4, 2/15 (318, 429) sai đầu tiên ở S2 rồi S4 quyết định; 4/15 trả số / câu sai không báo, 1/15 (263) báo thiếu dữ liệu của chỉ số SAI, 10/15 hỏi lại (4) hoặc ngoài phạm vi (6).
- Tiến độ, hai cách tính (chủ dự án chốt ở D-H1-1):
  - theo dữ liệu độc lập (A, khuyến nghị): **0/7**. G6' là PARTIAL (hai tiêu chí chưa đủ), nên chỉ thành 1/7 khi chủ dự án phê tiêu chí VÀ hai tiêu chí còn lại được làm xong;
  - theo B (ADR-0006 §8 chỉ chứng minh lại G2 / G5 / G6): 3/7 — KHÔNG khuyến nghị: dựa trên bộ 117 mà chính báo cáo này rút lại ở mục 18 câu 3.
- Hạ tầng đo được ĐẠT: nạp dữ liệu 2.739/2.739; phát lại 104/104 lệch 0; hai đường dữ liệu (bộ nhớ, PostgreSQL RLS) cho kết quả giống hệt ở lần chạy chính thức; 0 canary.
- Lệch quy trình phát hiện lúc chốt: 15 ca có gợi ý kỳ vọng lộ cho phiên chính 33–76 giây TRƯỚC khi đóng băng (không phải sau như đã ghi), nên theo ADR-0006 §5 lẽ ra phải loại trước khi chia (mục 3).

## 1. HEAD

| | Revision |
|---|---|
| START | `94bbaae` (push 20/10 19:26:28, reflog `origin/main`) |
| END | `15ed135` (mã; tài liệu chốt ở commit kế tiếp, cùng lần push) |

## 2. Trạng thái cổng THẬT

Tiêu chí: `docs/04` §6. Phần ca đo trên **H1** (`docs/g7/holdout/ket-qua-h1/`), đạt khi đạt ở CẢ HAI đường dữ liệu. ADR-0006 chỉ đăng ký chứng minh lại G2 / G5 / G6. G1 / G3 / G4 từng ĐẠT trên bộ 117 (không nhiễm, nhưng do người đã đọc pack soạn). Cùng tiêu chí trên dữ liệu độc lập thì KHÔNG ĐẠT, và báo cáo này lấy dữ liệu độc lập làm trạng thái THẬT (D-H1-1 = A).
- Lập luận A: theo ADR-0006 §1, bộ 117 là bộ REGRESSION, chưa từng là bằng chứng độc lập cho G1 / G3 / G4. H1 là bằng chứng độc lập đầu tiên, và nó không xác nhận ba cổng này.
- Theo tầng QUYẾT ĐỊNH: 15/15 ca đỏ ở tầng ý định / router (S4). Theo tầng sai ĐẦU TIÊN: 13/15 ở S4, 2/15 ở S2. Tầng chuẩn hoá (S2) lộ lỗi ở 7 ca H1: 264 (nhóm nghìn, bị bộ chấm che) và 6 ca đọc "03" thành 3 L (207, 253, 273, 318, 350, 429; 4 ca vẫn ĐẠT) — `phan-tich.md`.

| Cổng | Trạng thái THẬT | Bằng chứng |
|---|---|---|
| G1 DATA | **FAIL** (PARTIAL): nạp dữ liệu ĐẠT; phần ca `representation` H1 **10/13** theo bộ chấm, **≤ 9/13** theo nội dung | Nạp: 2.739/2.739 bản ghi khớp manifest, metamorphic 0 lệch; dữ liệu sinh lại giống từng byte (manifest sha256 `31172355…adb055`). Ca đỏ 235, 276, 399: lượng đọc ĐÚNG (1,2 m3, hai ngàn lít, 900l); sai ở S4 ("xuất" đứng một mình không thành chỉ số). Ca 264 ĐẠT với câu trả lời SAI: "1 490 lít" không được gộp (trái `03-data-map.md` dòng tệp 134 = hàng 14 của bảng). Sau H1 (`8ff3db5`) "1 490 lít" được gộp ⇒ END trả "Đúng. … 1.490 L"; nhóm nghìn còn lỗi đã đo (D-OP-9). Bộ 117: 100 % |
| G2 IDENTITY | **FAIL**: entity 13/13; ambiguity **12/13** | WIN-I-348 "máy 03 hôm qua xuất bao nhiêu và tiêu thụ bao nhiêu?" trả MỘT vế (tiêu thụ), bỏ thầm vế kia: đoán im lặng, trái P5 |
| G3 TRUTH | **FAIL**: semantic **11/13**, source_conflict **11/13** | 268, 318: "A hơn B bao nhiêu" không được nhận; 303, 392: "ERP ghi X L dầu" không thành chỉ số (OUT_OF_SCOPE). Bộ 117: 100 % |
| G4 DECISION | **FAIL**: routing **7/13** | 201, 234, 263, 342, 419, 429; tầng ý định (`phan-tich.md` mục 2). Bộ 117: 100 % |
| G5 CONTROL | **FAIL** (PARTIAL): phần ca permission 13/13, unsupported_answer **12/13**, canary 0. Phần cơ chế (AST, RLS, canary B, bảng "sai đúng một chiều", chèn lệnh) ĐẠT bằng test + đột biến **trên danh sách dạng viết đã thử** — danh sách đóng, còn lọt đã biết (mục 7) | 387: trả chênh lệch 68 L thay vì lượng xuất 3.850 L, phần quy trình bị bỏ; lỗi thứ hai ở S6 / S9 nên sửa S4 không đủ. Độ nhạy: bỏ 8 ca H1 có gợi ý lộ ⇒ G5' ĐẠT; 387 là ca có gợi ý và EACL vẫn trượt |
| G6 PROOF | **FAIL theo tiêu chí gốc** (bộ 117 băm 12 phút SAU lần chạy đầu). **G6' = PARTIAL** (tiêu chí sửa đổi, chờ chủ dự án phê):<br>- đạt, đo trực tiếp: H1 104 ca, 13 / lớp; commit + push đóng băng trước lần chạy 01:42; phát lại 104/104 lệch 0 ở hai đường; người soạn không viết mã;<br>- đạt MỘT PHẦN: "mỗi cổng có đột biến bị bắt" (bộ sinh `win/gen` và `core/decimal` chưa có danh mục); "test thuộc tính" phủ 2/3;<br>- lệch: 15 ca lộ gợi ý trước đóng băng không bị loại (mục 3);<br>- 2/104 ca trùng bộ 117 sau khi chuẩn hoá hoa / thường (218 = 059 = 061 = 099; 405 = 075). Bỏ ra: 87/102 theo nhãn, ≤ 86/102 theo nội dung | `docs/g7/CONTAMINATION-AUDIT.md`, `docs/g7/holdout/ket-qua-h1/bang-cong.md` |
| G7 VALUE | **BLOCKED**:<br>- không có khoá model (D-G7-8);<br>- bộ chấm v2 chưa hiện thực, chưa đóng băng (ADR-0007);<br>- pilot cần khoá | `docs/g7/HOP-DONG-G7.md` §13 |

## 3. Nhiễm đã tìm ra (CONTAMINATION FOUND)

- **Bộ 117 ca** (`docs/g7/CONTAMINATION-AUDIT.md`): 102 sạch, 3 nhiễm (083, 107, 108), 12 không chắc.
  - Ba kỳ vọng bị phiên viết mã đổi bằng lệnh Edit lúc 09:09:26–35 (kết quả ghi 09:09:36–38), sau lần chạy đầu (09:03:27, 76/117). Phiên chính đo lại 5 mốc giờ và nội dung 3 lần sửa: khớp (khai trong phiên; Phụ lục A của audit có giờ các lệnh Edit).
  - Prompt soạn ca của phiên viết mã có 4 diễn giải không có trong hợp đồng ("nhận"; "đốt"; "bao nhiêu dầu" mặc định là kho xuất; canary đại trà trong chuỗi cấm). Đây là gốc lỗi ở ca 083.
  - ADR-0002 D7 ghi sai sự kiện về ca 107; đã thêm đính chính. `docs/04` (cụm "thiếu loại vật tư", dòng 85 hiện nay; dòng 83 ở bản kiểm toán `fdb3a18`) trái D7 ⇒ câu chờ quyết D-G7-12.
- **Đêm nay, hai lần lộ bộ giữ lại mới:**
  1. **7 ca lộ cho người viết mã EACL.** Agent sửa pipeline chạy `grep` đệ quy, bộ lọc thư mục hỏng nên in ra 8 dòng; agent tự khai. 7 ca bị loại TRƯỚC khi chia tập, sang `exposed.yaml` chỉ để báo cáo (`af4054a`, D-G7-13).
  2. **Gợi ý kỳ vọng của 20 mã ca lộ cho phiên chính** (5 đã bị bỏ ở bước sửa kho; **15 còn trong bộ đóng băng: H1 8, H2 5, pilot 2**). Kết quả workflow soát kho hứa "chỉ số đếm" nhưng có trường chữ tự do. **Đính chính lúc chốt (người phản bác cuối, F-05; phiên đo lại):** thông báo kết quả vào phiên chính lúc **22:50:28** (dấu giờ transcript), TRƯỚC commit đóng băng (thông điệp 22:51:01, committer 22:51:44); phiên ghi nhận lộ lúc 22:52. Thông điệp `59ee556` ghi "sau khi đóng băng" là SAI. Theo ADR-0006 §5, ca lộ trước đóng băng phải bị loại trước khi chia; 15 ca này không bị loại ⇒ **lệch quy trình**. Đã có: độ nhạy đăng ký trước (bỏ 8 ca H1 ⇒ G5' ĐẠT); người phản bác phương pháp xác nhận không dòng mã sản phẩm nào sửa theo gợi ý. Không tái chia được vì H1 đã chạy; H2 có 5 ca lộ gợi ý nên không còn sạch (D-H1-3: bộ mới H3).
- **Kiểm transcript mọi agent:** không agent nào khác đọc bộ giữ lại (khai trong phiên; người phản bác phương pháp quét 50 transcript, cùng kết luận).

## 4. Bộ giữ lại độc lập (INDEPENDENT HOLDOUT CREATED)

- 229 ca do người không đọc mã soạn, có truth có cấu trúc.
- Ba người soát độc lập (số liệu / chính sách / độ khác nhau) gửi 252 cờ trên 119 ca. Người sửa không đọc mã xử lý: 175 sửa, 76 bỏ, 1 bác cờ (= 252) ⇒ bỏ 40 ca, còn **189 ca**.
- Tính lại số liệu từ dữ liệu bộ sinh: 125 ca có số, lệch 0.
- **Đóng băng `16a6140`**, push 22:51:48 (reflog `origin/main`), trước mọi lần chạy:
  - pilot 16, **H1 104 (13 / lớp)**, H2 62, exposed 7;
  - 0 lỗi định dạng; `--kiem` tính lại từ nguồn: KHỚP.
- **Giới hạn:** nội dung H2 đã công khai cùng bản đóng băng, nên bảo vệ H2 chỉ còn là quy trình (ADR-0006 §5); thêm 5 ca H2 lộ gợi ý (mục 3).
- **H1 đã chạy MỘT lần** lúc 01:42 ngày 21/10 (`56101be`, revision mã `9de63d2`), bằng `scripts/chay-giu-lai.sh`, DB riêng `eais_giu_lai`: 89/104 ở cả hai đường. Phân tích tầng sai và chẩn đoán qua revision: `docs/g7/holdout/ket-qua-h1/phan-tich.md`.
  - Cùng dữ liệu H1 (đường bộ nhớ): `94bbaae` 88/104, `3d730ea` 87, `1276b5f` 87, `9de63d2` 89. Từ `94bbaae` tới `9de63d2` không ca nào ĐẠT→ĐỎ theo nhãn (chỉ 421 ĐỎ→ĐẠT); `3d730ea` từng làm 264 ĐỎ, `6f23ff3` trả lại.
  - Theo NỘI DUNG, ca 264 xấu đi: đầu đêm trả giá trị không phán quyết; `6f23ff3` khẳng định SAI, bộ chấm không bắt.
  - Từ lúc chạy, H1 chỉ còn là bộ hồi quy.
  - H1 làm bộ hồi quy sau khi sửa (đường bộ nhớ, worktree sạch, chạy một–ba lần giống hệt): `37e8a08` 89, `9d30130` 89, `fd4d19c` 88, `64daad0` 88 (hai mốc này có `fd4d19c`: 264 hỏi lại), `3066044` 88 (nhánh không vào main), **END `15ed135` 89/104: 0 ca đổi nhãn so với lần chạy chính thức, 264 trả "Đúng. … 1.490 L"**.

## 5. Bộ chấm (SCORER BUGS FOUND / FIXED)

- **Bộ chấm v1 bị phá:**
  - người tấn công độc lập khai 133/170 đòn thành công (109 câu sai mà ĐẠT, 24 câu đúng mà TRƯỢT); test máy (`TestG7ScorerAttacks`) xác nhận 122 đòn không tranh cãi bị chấm sai (107 sai mà ĐẠT + 15 đúng mà TRƯỢT), 10 đòn tranh cãi chưa so, 1 đòn không tái lập;
  - "một câu chỉ liệt kê mọi số của bản ghi đạt 73/117 ca": theo người tấn công, mã ở `docs/sources` (không công khai), chưa tái lập trong repo.
- **Đã sửa trong v1:** ratchet 170 đòn, nợ 122 → **118**; rỗng / giả rỗng = từ chối (D-G7-9 mục 6, ADR-0005); hệ lỗi khi hỏi luôn trượt; số được phép riêng từng ca; canary tên.
- **v2:** hướng thiết kế chốt ở mức đề xuất (ADR-0007, `b5df4c8`). **Không đóng băng được đêm nay**: lần nhìn đầu ngoài mẫu trượt oan 4,6 % (6/130), đạt nhầm 9,0 % (9/100), vượt ngưỡng 2 %; cần hiện thực (ƯỚC 10–15 ngày công) và hiệu chỉnh trên pilot thật với người chấm mù (D-G7-16).
- **Thước đo H1** (`scripts/bang-cong-h1.py`) cũng có lỗi, do soát lượt 2 tìm ra và đã sửa TRƯỚC khi chạy (`291d9bf`): G2' / G5' từng chỉ đọc đường bộ nhớ; phát lại từng ĐẠT với 0 câu.

## 6. Nhà cung cấp (PROVIDER ISSUES)

- Client cũ gửi `temperature: 0`; theo tài liệu nhà cung cấp (tra 20/10), model hiện hành trả 400 với sampling khác mặc định. Đã sửa (`750307a`): không gửi sampling mặc định; thinking / effort / thử lại lấy từ cấu hình. **Chưa đo bằng lời gọi thật** (không khoá).
- Lùi thử lại từng không có trần; huỷ ctx khi đang chờ từng không trả đúng lỗi. Đã sửa (`fe22e2a`).

## 7. Bảo mật (SECURITY FINDINGS)

- **Lượng nghiệp vụ trong tài liệu lọt CẢ HAI lớp kiểm** khi viết NBSP / ZWSP / toàn chiều rộng / ℓ / đơn vị đứng trước / đơn vị trong ngoặc: đã sửa (`5ce22ec`); "mét khối", "m^3", "(lít dầu)", "(4.000) lít", "lít/ca: 4.000": đã sửa ở cả hai lớp (`6f23ff3`). **Còn lọt cả hai lớp, đã đo:** "5 mét  khối" (hai dấu cách, R-09), "5 mét\nkhối", "5 met khoi", "5 ㎥", "5.000&nbsp;lít", "5.000 liters"; R-03 (lớp chuẩn hoá chế độ Layout) và A-08b (bộ đếm độc lập của G7) chưa sửa.
- **Biến mẫu thay nhiều lượt** có thể ghép thành số chưa từng được kiểm: đã sửa (`5ce22ec`).
- `g7run` từng ra PASS giả khi mọi lời gọi model của baseline lỗi: đã sửa (`a94b5fb`).
- **Tiêu tiền thật mà bỏ qua mọi kiểm** (soát C3, C3-04): `g7run` nhận "chạy khô" bằng so CHUỖI với `https://api.anthropic.com`, nên `:443`, chữ hoa hay dấu chấm cuối tên máy bị coi là chạy khô và gọi hết các ca bằng khoá thật (lượt chính k = 3: 372 lời gọi). Đã sửa (`246eabb`): so theo nghĩa; mọi endpoint không phải loopback, kể cả proxy, đều qua kiểm trước.
- **Bí mật trong URL ra log / bằng chứng:** khoá đặt trong path / query / fragment của endpoint từng ra thông báo cấu hình và `manifest.json` (C3-08, sửa `246eabb`), và ra stdout, `ket-qua.md`, `raw-runs.jsonl` khi lời gọi lỗi truyền tải (F6, sửa `15ed135`: bóc URL của `*url.Error` về `scheme://host`).
- Khoá: không tìm khoá. Chỉ dùng biến môi trường (D-G7-8). Kiểm 19:37 và 02:12: không có biến.

## 8. Lỗi tìm ra / đã sửa (BUGS FOUND / FIXED)

Theo nguồn phát hiện. "Đã sửa" nghĩa là có test đỏ trước khi sửa, xanh sau khi sửa, và đột biến cho nhánh mới.

| Nguồn | Tìm ra | Xử lý |
|---|---|---|
| Phiên chính, đọc tài liệu nhà cung cấp | client gửi `temperature: 0` ⇒ model hiện hành trả 400 (theo tài liệu) | sửa `750307a` |
| Soát mã độc lập lượt 1 (21:22) | S1-1 g7run ra PASS giả khi mọi lời gọi baseline lỗi; S2-1/2/3 lượng tài liệu lọt + ghép số qua nhiều lượt thay; S2-4 cắt thầm vế đầu; S2-5, S2-6, S3-2, S3-4, S4; test hổng | sửa `a94b5fb`, `fe22e2a`, `5ce22ec`, `3d730ea` |
| Người tấn công bộ chấm (170 đòn) | v1: 122 đòn không tranh cãi bị chấm sai (mục 5) | ratchet `82baa71`; sửa phần chắc `6b7d963` (nợ 122 → 118); v2 thiết kế ADR-0007, chưa đóng băng |
| Soát lượt 2, chiều B (công cụ bộ giữ lại) | 20 phát hiện (B-01…B-20). Ví dụ: G2' / G5' chỉ đọc đường bộ nhớ; phát lại 0 câu vẫn ĐẠT; CRLF lọt; chạy lại được nhiều lần | sửa B-01…B-17 trong `291d9bf` TRƯỚC H1; nợ B-11, B-13 (phần còn lại), B-19, B-20 |
| Soát lượt 2, chiều A (pipeline) | 12 xác nhận + 3 người phản bác tìm thêm. Phần lớn là hồi quy của chính `5ce22ec` / `3d730ea` | sửa `6f23ff3` TRƯỚC H1 |
| Soát lượt 2, chiều C (harness G7) | 19 phát hiện, 3 bị bác (C-08, C-09, C-12). Ví dụ: C-06 G7 PASS được với bộ chấm v1 hỏng; C-18 khoá theo chuyển hướng HTTP; C-04 tốn tiền trước khi kiểm điều kiện | sửa C-03…C-07, C-10, C-11, C-13, C-17, C-18, C-19 (`d4b80be` … `50b1499`). C-01, C-02, C-14, C-15, C-16 thành ca thử bắt buộc của v2 |
| Soát bản sửa chiều A (sau H1, không thấy bộ giữ lại) | 12 phát hiện; R-01, R-02 (S1), R-05, R-06 (S2), R-07, R-10 là HỒI QUY do `6f23ff3` | bản sửa R `a8b0858` |
| Hai người soát bản sửa R | 27 mã (R-A01…R-A18, B-01…B-09), nhiều S1 CÒN SAI. Gốc chung: luật A-02 và các danh sách từ đóng | **đổi hướng**: luật an toàn `37e8a08` (từ hai con số ứng viên trở lên thì hỏi lại; mốc xét trước nhãn) |
| Người soát C3 (commit của phiên chính 01:3x–03:4x) | 9 phát hiện: C3-04 (S3) tiêu tiền bỏ qua kiểm; C3-02 (S2) nạp "bể 2 500 lít" thành 2.500 L; C3-05 (S3); C3-06, C3-07, C3-08 (S4); C3-01 (S2, bộ đếm v1); C3-03, C3-09 | `246eabb` (C3-04, 06, 07, 08); C3-02, C3-05: `fd4d19c` (bị bác, đã revert) rồi `3066044` (bị bác, không vào main) ⇒ CÒN MỞ (D-OP-9); C3-09 nay hỏi lại. Còn mở: C3-01 (làm ở bộ chấm v2), C3-03 |
| Soát `37e8a08` (hai người: đo lại 27 mã R; săn S1 mới) | Xác nhận R-A01/02/03/05/06/07, B-02/03 nay hỏi lại. **Hai hồi quy của chính `37e8a08`:** RM-1 / F7 (S2) lớp kiểm tài liệu lọt "**5** khối", "**5** L", "\| 5 \| khối \|"; F6 (S2) "phiếu 15-9" mất nghi vấn ngày | sửa `64daad0` |
| Soát `fd4d19c` + `246eabb` | **Hồi quy của `fd4d19c`:** F1 (S1) "… 3850 lít, máy chạy 12 000 km đúng không?" → "không phải 12 L"; F2 (S2) "sửa chữa 500 000 đồng" nạp thành 500 L dầu; F3 (S2) "12 000 km" trong tài liệu làm căn cứ cho số bịa "12 người"; F4 (S3). Có từ trước: F5 (S2) "bể #2 500 lít", "bể: 2 500 lít" vẫn gộp; F6 (S2) lỗi truyền tải in URL đầy đủ có khoá; F7 (S4) vài cách viết loopback bị từ chối | revert `2a87e63` (push ngay, main về trạng thái `9d30130` cho nhóm nghìn); `15ed135` (F6); `3066044` thử sửa F1–F5 (hàng dưới). F7: hướng an toàn, ghi nợ |
| Soát `3066044` + `15ed135` + revert (hai người) | **Hồi quy của `3066044`:** F1 (S1) nhóm nghìn nuốt số cuối của tên thiết bị — tin "máy 3 850 lít, phiếu 912" nạp thành 912 L (main: cách ly); F2 (S2) câu xác nhận bỏ mất nhóm chồng tên rồi phán quyết theo số còn lại; F3, A6 (S3); A8 (S4). Kẽ hở chung với main (không phải hồi quy): F4, A1 (S1) nhóm không đơn vị / đơn vị ngoài lexicon / thập phân vẫn tách ⇒ "tổng số 3 850 được cấp đúng không?" → "không phải 850 L"; A2, A3, A4, F5 (S2); F6 (S3); A5 (S3, `15ed135`: thân phản hồi / Location do máy chủ cấp vẫn ghi nguyên văn). Revert `2a87e63` sạch (cây trùng khớp); `15ed135` sạch trên 16 kịch bản lỗi truyền tải | **`3066044` KHÔNG vào main** (giữ ở nhánh cục bộ `thu/nhom-nghin-3066044`); END = `15ed135`. Hướng làm lại: D-OP-9 |
| Đột biến | sống: M23b, OR11, E3, BC9; tương đương: BC8; chuỗi gốc hỏng: S1, G35, PC6, PC9, PC10, FC2, AG. Sau `37e8a08`: n28 5 dòng mất mã đích (luật A-02 đã gỡ) + A02b, A02l sống; n35 R02a/c/d/f/p/q, R03i sống, R04a, R04b, AT3, KT4 không biên dịch; n37 NC6 bị luật mới che. Lượt `64daad0`: n30 PF3 sống (kiểm thiếu hệ cổng che), n39 GS1 không biên dịch | thêm test hoặc sửa chuỗi, chạy lại đến khi bắt hết; dòng của mã đã gỡ bỏ có lý do; A02b, R02a, R02f ghi tương đương có lý do (`f5f4a5a`, `9d30130`, `64daad0`, `15ed135`) |
| Phiên chính | ký tự vô hình THẬT trong test (11 ký tự) và trong một thông điệp commit (chưa push). Đo H1 trong CÙNG clone đang chạy đột biến ⇒ số nhiễm (khai trong phiên: 88/104 rồi 51/104) | `7be1593`; thông điệp sửa trước khi push. Đo lại trên worktree sạch riêng; luật: không đo trong clone đang chạy `mutate.sh` |
| H1 (sau lần chạy) | 15 ca đỏ, 7 nhóm gốc ở tầng ý định | KHÔNG sửa trong đêm (ADR-0006 §7); D-H1-2 |

**Câu xác nhận — phán quyết sai còn mở.** 15 câu thăm dò lấy từ các người soát, đo lại ở năm mốc (đường bộ nhớ, hôm nay = 19/10/2026, EX-03 được cấp 3.850 L; "Đ" = "Đúng.", "K" = "Không.", "hỏi" = hỏi lại, "số" = trả số liệu không phán quyết; **in đậm** = phán quyết sai):

| Câu (rút gọn) | `94bbaae` | `3d730ea` | `6f23ff3` | `a8b0858` | END | Lớp |
|---|---|---|---|---|---|---|
| "Định mức 5000 L, vậy … được cấp con số 3850 đúng không?" (đúng) | Đ | Đ | Đ | **K** | **K** | F1 — hồi quy `a8b0858` |
| "… được cấp con số 4000 đúng không, kho xuất 3850 L?" (sai) | K | hỏi | hỏi | **Đ** | **Đ** | F1 — hồi quy `a8b0858` |
| "… được cấp 3850 L, tại bể 7 đúng không?" (đúng) | **K** | **K** | **K** | **K** | **K** | F2 — có sẵn |
| "… đã được cấp dầu theo chứng từ 912 đúng không?" | **K** | **K** | **K** | **K** | **K** | B-05 / R-A09 — có sẵn |
| "EX-03 hôm nay kho 2 được cấp dầu đúng không?" | khác | **K** | **K** | **K** | **K** | B-05 — hồi quy `3d730ea` |
| "Dự án A Q3 được cấp 4600 L đúng không?" (đúng cho Q3) | **K** | **K** | **K** | **K** | **K** | F3 — có sẵn |
| "EX-03 tuần44 được cấp 3850 L đúng không?" | **Đ** | **Đ** | **Đ** | **Đ** | **Đ** | F3 — có sẵn |
| "Định mức 5000 L, vậy … được cấp 3850 kg đúng không?" | số | số | **K** | **K** | **K** | F4 — hồi quy `6f23ff3` |
| "EX-03 hôm nay không được cấp 3850 L đúng không?" | **Đ** | **Đ** | **Đ** | **Đ** | **Đ** | F5 — có sẵn |
| "… được cấp trên 3000 L đúng không?" | **K** | **K** | **K** | **K** | **K** | F5 — có sẵn |
| "Cho hỏi 15-9 máy xúc EX-03 được cấp 3,5 m3 phải không?" | số | hỏi | **K** | **K** | **K** | R-A04 — hồi quy `6f23ff3` |
| "EX-03 15th Sep được cấp 3850 L đúng không?" | **Đ** | **Đ** | **Đ** | **Đ** | **Đ** | R-A04 — có sẵn |
| "EX-03 ngày 12 ca 2 được cấp 3850 L đúng không?" | số | hỏi | **Đ** | **Đ** | **Đ** | B-01 — hồi quy `6f23ff3` |
| "EX-03 hôm nay ca 2 được cấp 3850 L đúng không?" | số | hỏi | **Đ** | **Đ** | **Đ** | mốc dưới ngày — hồi quy `6f23ff3` |
| "EX-03 sáng nay được cấp 3850 L đúng không?" | **Đ** | **Đ** | **Đ** | **Đ** | **Đ** | mốc dưới ngày — có sẵn |

Tổng: `94bbaae` sai 8/15 → END sai 15/15. Đây là 15 câu đối kháng do người soát dựng, không phải mẫu đại diện; tỉ lệ trên câu thật CHƯA ĐO. Test `pipeline_xac_nhan_an_toan_test.go` còn khẳng định "hôm nay ca 2 … → Đúng." là hành vi đúng: cần sửa cùng luật mới.

Hướng sửa đề xuất (câu chờ quyết **D-OP-10**): chỉ phán quyết khi con số được nêu có vai trò LƯỢNG rõ ràng (có đơn vị cùng chiều đo, dạng lượng "4k" / "4.000", hoặc số trần đứng ngay sau từ khoá của chính ý định đã khớp trong `intents.yaml`), trong mệnh đề hỏi không có số nào bị bỏ vì nhãn / mốc / khác chiều đo, không có chữ-số hay mốc dưới ngày chưa đọc ("Q3", "15th", "ca 2", "sáng nay"), và không có từ phủ định / so sánh ngoài cụm xác nhận. Mọi trường hợp khác: trả số liệu KHÔNG kèm "Đúng." / "Không.". Chưa làm đêm nay: đổi hành vi người dùng thấy và cần một vòng soát độc lập. Phương án tạm an toàn nhất trong lúc chờ: tắt hẳn phán quyết (D-OP-10 c).

## 9. Số test (TEST COUNTS)

Đo trên bản clone, có PostgreSQL (DSN `eais`), `go test ./... -count=1`, đếm cả test con (subtest):

| Mốc | Đạt / trượt / bỏ qua |
|---|---|
| đầu đêm `94bbaae` | 224 / 0 / 0 |
| trước H1 `5c1c973` | 357 / 0 / 0 |
| sau sửa chiều C2 `fbea42e` | 373 / 0 / 0 |
| sau sửa hồi quy R `a8b0858` | 404 / 0 / 0 |
| luật xác nhận an toàn `37e8a08` | 410 / 0 / 0 |
| sửa C3 + danh mục `9d30130` | 429 / 0 / 0 |
| `64daad0` (đã push) | 429 / 0 / 0 |
| END `15ed135` | 431 / 0 / 0 |

Người phản bác số liệu đếm lại không PG ở từng mốc: số đạt + 5 test PG bị bỏ qua = đúng các số trên. `TestWinRatchet` ở mọi mốc: G1 2.739/2.739, bộ 117 ca 117/117. Scale smoke trên END `15ed135`: ĐẠT (117 ca giống hệt giữa hai hồ sơ, câu trả lời giống từng byte; smoke 4,64 s, 634,2 MiB — thời gian / RAM chỉ ghi, không có ngưỡng; máy đang chạy đột biến song song).

## 10. Đột biến (MUTATION RESULTS)

Mỗi lượt chạy MỌI danh mục `scripts/dot-bien/*.sh` trên bản clone (`scripts/mutate.sh` sửa tệp tại chỗ rồi khôi phục):
- `5c1c973` (trước H1): 24 danh mục, 370 bị bắt, 0 sống, 0 không biên dịch, 0 mất chuỗi gốc.
- `a8b0858` (sau sửa hồi quy R): 28 danh mục, 549 bị bắt, 0 sống.
- `64daad0`: 31 danh mục, 567 bị bắt, 1 sống (n30 PF3), 1 không biên dịch (n39 GS1) — cả hai sửa ở `15ed135` (đo từng dòng: BỊ BẮT).
- END `15ed135`: 31 danh mục, **571 bị bắt, 0 sống, 0 không biên dịch, 0 mất chuỗi gốc**.
- Số đột biến trong danh mục đếm tĩnh (người phản bác số liệu): khớp 370, 549; riêng mốc `6b7d963` đếm được 234 + 7 (thông điệp commit ghi 240 + 7, đo trên cây làm việc trước commit; chưa giải thích 6 dòng chênh).
- Đột biến sống / chuỗi hỏng tìm thấy rồi sửa trong đêm: mục 8, dòng "Đột biến".

## 11. Độ sẵn sàng benchmark (BENCHMARK READINESS) — G7 chạy được bằng một lệnh chưa?

**Chưa.** Harness có đủ: bằng chứng thô, thử khói, pilot, trần tiền bắt buộc, nhận biết chạy khô, kiểm trước khi tiêu tiền, B1 / B2+ / B2″ / B2-oracle. Còn thiếu:
- khoá;
- bộ chấm v2 hiện thực + đóng băng (cần pilot thật + người chấm mù);
- tập của lượt chính (D-H1-3: H2 quá nhỏ cho D5 và có 5 ca lộ gợi ý);
- đo lại chi phí: `COST-PLAN.md` đo trên mã `9de63d2`, pipeline và client đã đổi sau đó.

## 12. Lời gọi model và chi phí

- Lời gọi model thật: **0**. Chi tiêu: **0 USD**.
- Dự toán G7 (`docs/g7/COST-PLAN.md`, đo bằng chạy khô pilot + H1 trên mã `9de63d2`; token là ƯỚC):
  - hệ cổng, lượt chính k = 3: kỳ vọng 12,74 USD (tối thiểu 10,02 · xấu nhất 27,40);
  - thêm một lần chạy lại trọn lượt: 23,44 (bản đầu ghi 16,36: tính sai thành chạy lại k = 1; soát C2, F-09); k = 5: 19,82;
  - B1 chiếm 88,8 % chi phí hệ cổng;
  - D-G7-18.

**8 điều kiện của quyền chi 15 USD** (quyết định chủ dự án 20/10, mục PAID EXECUTION AUTHORIZATION). Kiểm 21/10 02:12:

| # | Điều kiện | Trạng thái | Bằng chứng |
|---|---|---|---|
| 1 | `EAIS_MODEL_API_KEY` có trong môi trường | **KHÔNG** | Kiểm có / không, không in giá trị, lúc 19:37 và 02:12. Không tìm khoá ở nơi khác (D-G7-8) |
| 2 | Hợp đồng FROZEN | **KHÔNG** | Hợp đồng §13: bộ chấm v2 chưa đóng băng (ADR-0007); pilot cần khoá; tập lượt chính chờ D-H1-3 |
| 3 | Dự toán ≤ 15 USD | CHỈ khi hệ cổng, k = 3, không chạy lại (12,74 kỳ vọng). Chưa chắc: token chưa đo, mã đã đổi sau lần đo | `COST-PLAN.md` |
| 4 | Pilot + lượt chính trong cùng trần | như dòng 3 (đã gồm pilot k = 5). Trần là của từng lần chạy; luật cộng dồn ở hợp đồng §9 | `9de63d2` (C-07) |
| 5 | Model chính là Sonnet 5.5 | ĐẠT về cấu hình: hợp đồng §4; `g7run` không ra kết luận khi model / thinking / effort lệch hợp đồng | `d4b80be` (C-06) |
| 6 | Baseline và EACL cùng model | ĐẠT: một gateway chung cho mọi hệ | `g7run` build |
| 7 | Lưu bằng chứng thô | ĐẠT: `-out` (gồm bản ghi quyết định của EACL) | `a1a5b6b`, `50b1499` |
| 8 | Không có secret trong đầu ra / git | ĐẠT trong mọi lần chạy khô:<br>- khoá không có trong `-out`;<br>- client không theo chuyển hướng HTTP, chỉ https hoặc loopback;<br>- log, `manifest.json` và lỗi truyền tải chỉ ghi `scheme://host` (`246eabb`, `15ed135`);<br>- quét riêng tư trước mỗi lần push | `TestSmokeThroughFakeProvider`, `TestBangChungKhongGhiBiMatTrongURL`, `TestLoiTruyenTaiKhongInBiMatTrongURL` |

⇒ **Paid G7 = BLOCKED** (điều kiện 1 và 2 không đạt). Không gọi model, không chi tiền.

## 13. Baseline công bằng

- **B2+ là baseline cổng:** đoạn văn có bí danh + mã theo nguồn + ngày hiện tại. Đo được: trần oracle top-10 là 95/117 với B2, 114/117 với biến thể ALIASES=1 REWRITE=1 của `d5probe` (tái lập đúng số bản soát D5).
- **B1:** LLM trên bản ghi thô theo cửa sổ ngày + `entities.json`.
- **Chỉ báo cáo:** B2″ (luật miền viết thành chữ) và B2-oracle (đúng đoạn bằng chứng).
- **Mọi hệ** dùng cùng model, cùng tham số, cùng system prompt.

## 14. Commit (COMMITS)

83 commit từ `94bbaae` tới END `15ed135` (41 tới `b5df4c8` lúc 00:16), cộng commit tài liệu chốt. Danh sách: `git log 94bbaae..15ed135`. Một commit đã revert: `fd4d19c` (bởi `2a87e63`).

## 15. Phức tạp đã từ chối (REJECTED COMPLEXITY)

- Bộ chấm v2 bản 2 (giao thức JSON có kiểu cho mọi hệ): đổi đối tượng đo, không làm cổng (ADR-0007); không công khai mã nguyên mẫu v2 (≈ 5,6 nghìn dòng) khi chưa soát.
- Vá bộ đếm v1 cho C-01 / C-02 / C3-01: v1 đã bị chặn kết luận; làm ở v2.
- Không đọc "từ X đến Y" thành một kỳ: hỏi lại thay vì thêm quy tắc chuẩn hoá mới (S2-4).
- Không trích xuất lõi chung (`docs/v3/core-candidates.md` chỉ ghi nhận); không đổi tên repo / module (D-V3-03); không implement Customer Ops khi G7 còn là đường găng (D-V3-09).
- Không sửa mã theo H1 (ADR-0006 §7).
- Chế độ `-redact`, sổ chi tiêu cộng dồn tự động: thay bằng luật quy trình (hợp đồng §9, §11).
- Không vá tiếp câu xác nhận bằng danh sách từ (luật A-02, danh sách từ nhãn cho nhóm nghìn): hai vòng soát cho thấy danh sách đóng nào cũng rò; đổi sang luật an toàn.
- Chưa thêm danh sách "từ dẫn lượng" vào pack (D-OP-9 b) và chưa làm luật phán quyết mới (D-OP-10): đổi hành vi người dùng thấy, chờ chủ dự án.
- Chưa gỡ `joinClaims` (sau khi gỡ A-02 chỉ còn dùng chung đơn vị): đơn giản hoá được, nhưng không đổi mã lõi khi chưa có người soát.

## 16. ADR mở (OPEN ADRS)

- ADR-0006 §8: G6' chờ phê (và hai tiêu chí chưa đủ); §5: ghi lệch quy trình 15 ca lộ trước đóng băng (đính chính lúc chốt).
- ADR-0007: ĐỀ XUẤT; chờ D-G7-15, 16, 17.
- ADR-0003: hướng đã duyệt, chưa implement.

## 17. Chặn cứng (HARD BLOCKERS)

1. Khoá model (D-G7-8): chủ môi trường đặt `EAIS_MODEL_API_KEY`.
2. Người chấm mù cho hiệu chỉnh bộ chấm v2 (D-G7-16).
3. D-G7-12: `docs/04` (cụm "thiếu loại vật tư") hay ADR-0002 D7.

## 18. Mười câu hỏi

1. **Giờ biết gì mà tối qua chưa biết?**
   - Trên dữ liệu độc lập, EACL đạt 89/104 theo bộ chấm nhị phân (≤ 88/104 theo nội dung), và phần ca của MỌI cổng G1–G5 trượt.
   - 15 ca đỏ đều quyết định sai ở tầng ý định; 4/15 trả lời sai không báo, 1/15 báo thiếu dữ liệu của chỉ số sai.
   - Bộ chấm WIN nhị phân có thể cho ĐẠT một câu xác nhận trả lời SAI (ca 264).
   - Bộ chấm G7 v1 không dùng được: 122 đòn không tranh cãi bị chấm sai.
   - Client cũ không gọi được model hiện hành (theo tài liệu nhà cung cấp: `temperature 0` bị trả 400; chưa đo bằng lời gọi thật).
   - G7 đo bằng chạy khô: chỉ hệ cổng, k = 3, khoảng 12,74 USD (kỳ vọng); B1 chiếm gần hết.
   - **Các bản sửa luồng hỏi của chính đêm nay gây hồi quy, và chỉ người soát / người phản bác độc lập bắt được:** `3d730ea`, `6f23ff3`, `a8b0858`, `8ff3db5`, `37e8a08`, `fd4d19c`, `3066044` đều có hồi quy; phần lớn đã sửa hoặc không vào main, nhưng 7 câu xác nhận phán quyết sai ở END là hồi quy chưa sửa (mục 8).
   - Vá câu xác nhận bằng DANH SÁCH TỪ không đóng được lỗ; không gộp nhóm nghìn thì sinh số mồ côi; gộp thì phải biết ranh giới tên thiết bị. Hướng đúng là luật an toàn có một nguồn (hai cách đọc ⇒ hỏi lại / cách ly; một span cho một lượng), làm ở phiên có soát.
2. **Điều gì đã chứng minh độc lập?**
   - G1 phần nạp dữ liệu; phát lại 104/104; hai đường dữ liệu giống hệt (lần chạy chính thức).
   - G6' phần đo trực tiếp (đóng băng trước, chạy một lần, người soạn không viết mã); phần chưa đủ: đột biến mỗi cổng, test thuộc tính 2/3; lệch quy trình 15 ca (mục 3).
   - Người phản bác cuối xác nhận: chạy một lần; không dòng mã sản phẩm nào sửa theo gợi ý đã lộ; tầng sai của 15 ca; chẩn đoán qua revision (88 / 87 / 87 / 89); độ nhạy; mọi số 89/104, 88/104 theo revision, số test, dự toán tính lại khớp.
   - Trên H1: 0 rò canary, quyền 13/13, thực thể 13/13. Phần cơ chế của G5: test + đột biến (trên danh sách dạng viết đã thử).
3. **Điều gì từng nói quá?**
   - "3/7 cổng đạt vô điều kiện (G1, G3, G4)" và "6/7 có điều kiện": bộ 117 là bộ REGRESSION (ADR-0006 §1), do người đã đọc pack soạn.
   - "G7 chỉ chờ khoá": bộ chấm v1 hỏng, client lỗi 400.
   - ADR-0002 D7 ghi sai sự kiện ở ca 107 (đã đính chính).
   - **Trong chính đêm nay** (người phản bác cuối bắt):
     - bản nháp ghi lỗi nhóm nghìn "đã sửa" lần lượt ở `8ff3db5`, `fd4d19c`, `3066044` — soát tìm ra lỗi ở cả ba: `8ff3db5` rò danh sách (vẫn ở main, 264 đúng, C3-02 / C3-05 còn mở), `fd4d19c` sinh số mồ côi (revert), `3066044` nuốt số của tên thiết bị (không vào main);
     - bản nháp ghi 7 lớp phán quyết sai của câu xác nhận là "có sẵn, không phải hồi quy" — sai: 7 câu là hồi quy của chính đêm nay (mục 8);
     - thông điệp `59ee556` và bản nháp ghi 15 ca lộ gợi ý "sau khi đóng băng" — sai: trước 33–76 giây (mục 3); "H2 vẫn là bộ giữ lại cho G7" — H2 có 5 ca lộ gợi ý;
     - "1/7 nếu phê G6'" — G6' còn PARTIAL;
     - thông điệp `37e8a08` ghi "câu có số đi với danh từ lạ bị hỏi lại" — chỉ đúng khi câu CÓ nêu lượng ("kho 2 được cấp dầu đúng không?" ra "Không. … không phải 2 L").
4. **Cái gì trượt?**
   - H1: 15/104 ca (representation 3, ambiguity 1, semantic 2, source_conflict 2, unsupported_answer 1, routing 6). H1 làm bộ hồi quy trên END: 89/104, 0 ca đổi nhãn (264 nay đúng cả nội dung).
   - Bộ chấm v1; bản đầu của công cụ chấm H1 (sửa trước khi chạy).
   - Bảy bản sửa của chính đêm nay (mục 8) có hồi quy; `fd4d19c` phải revert, `3066044` không vào main.
   - Câu xác nhận: END phán quyết sai 15/15 câu thăm dò (đầu đêm 8/15), CHƯA sửa.
5. **Vì sao trượt?**
   - Đo được: tầng ý định không có cách nói của 15 ca. Ví dụ "xuất" đứng một mình không thành chỉ số; `comparison_markers` chỉ có "nhiều hơn", "ít hơn", "so với", nên "A hơn B bao nhiêu" không được nhận.
   - GIẢ THUYẾT chưa đo riêng: từ khoá được viết theo cách nói của bộ ca phát triển. EACL vẫn xử lý được vài cách nói không có trong bộ 117 ("bn", ngày ISO 2026-10-19; "dc cấp" thì đã có trong bộ 117, dạng không dấu).
   - Bộ chấm v1 là "túi số": không ràng buộc con số với thực thể / kỳ / nguồn.
   - Câu xác nhận phán quyết mỗi khi trích được ĐÚNG MỘT con số, mà việc trích có nhiều cách sai im lặng. Đúng một con số chưa phải bằng chứng là đã hiểu câu hỏi.
   - Sửa nhanh theo phát hiện cục bộ thì hỏng ở chỗ khác; chỉ người soát độc lập sau mỗi bản sửa mới bắt được.
6. **Cố ý KHÔNG xây gì?** Mục 15.
7. **Fuel WIN đã đạt chưa?** **CHƯA.**
   - Theo dữ liệu độc lập (A): 0/7; G6' PARTIAL.
   - Theo B (không khuyến nghị): 3/7.
   - G7 chưa chạy lần nào.
8. **G7 chạy được bằng một lệnh chưa?** **CHƯA.** Thiếu khoá; bộ chấm v2 chưa hiện thực (ƯỚC 10–15 ngày công) và chưa hiệu chỉnh với người chấm mù; tập của lượt chính chờ D-H1-3; trần chi phí chờ D-G7-18 và cần đo lại trên mã hiện tại. Harness đã đủ: bằng chứng thô có bản ghi quyết định, thử khói, pilot, trần ngân sách bắt buộc, nhận biết chạy khô, kiểm trước khi tiêu tiền.
9. **EACL đang ở giai đoạn nào?** Vẫn là WIN MVP (Fuel), chưa đạt: "khôi phục bằng chứng → sửa tầng ý định và câu xác nhận", trước G7. Gói định nghĩa EACL V3 R0 chờ duyệt D-V3-01…10.
10. **Một việc kế tiếp giá trị nhất?**
    - Soạn và đóng băng bộ giữ lại mới H3 (người không đọc mã soạn, ≥ 13 ca / lớp, đóng băng bằng cam kết băm, kết quả trả về chỉ có số đếm) TRƯỚC khi sửa tầng ý định và câu xác nhận (dùng H1 và 15 câu thăm dò làm bộ hồi quy).
    - Không có H3 thì không chứng minh lại được G1–G5 sau khi sửa, và G7 không có tập hợp lệ (D-H1-2, D-H1-3).

## 19. Bằng chứng thô

- Công khai trong repo: `docs/g7/holdout/` (bản đóng băng + kết quả H1); `docs/g7/CONTAMINATION-AUDIT.md`; `docs/g7/REVIEW-D5.md`; `docs/g7/BO-CHAM-V2-DAC-TA.md`.
- Ở máy (gitignore):
  - `docs/sources/dem-2026-10-20/raw/` (Pha A);
  - `docs/sources/dem-2026-10-20/soat/` (kết quả các vòng soát và người phản bác, đầu ra hồi quy H1 theo revision, scale smoke);
  - `docs/sources/g7-scorer-v2-proto/` (mã nguyên mẫu, kết quả giám khảo).
