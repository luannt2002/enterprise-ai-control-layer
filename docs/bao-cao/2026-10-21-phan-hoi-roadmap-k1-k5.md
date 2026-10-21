# Phản hồi audit ROADMAP — phần tiếp theo: K1–K5 (21/10/2026 tối)

Nối tiếp `docs/bao-cao/2026-10-21-phan-hoi-roadmap.md` (bản giữa chừng chiều 21/10; tệp đó giữ nguyên). Phạm vi theo brief
BUILD §4 mục 7: "phần còn lại của phản hồi `PROMPT-AUDIT-ROADMAP.md`: chỉ các khẳng định K về sự thật trong mã". Đề bài K1–K5:
`docs/PROMPT-AUDIT-ROADMAP.md` dòng 40–48.

Nhãn bằng chứng giống ROADMAP: [ĐO] [CODE] [REPO] [SUY].

## 1. Mốc

- **Mã lúc đo:** `7cad584`. HEAD nhích lên `d5048b9`, `23b957f` trong lúc đo nhưng chỉ đổi `docs/bao-cao/TIEN-DO-BUILD.md`:
  `git diff --stat 7cad584 HEAD -- . ':!docs'` rỗng [ĐO]. Người đo lại chạy lại trọn gói ở `23b957f`: kết quả giống hệt.
- **Đường chạy:** bộ nhớ (`winreport` không `-pg`). Đường PostgreSQL chưa đo.

## 2. Cách đo

- Đúng phương pháp của đề: sinh dữ liệu seed 42 vào thư mục tạm
  (`go run ./tenants/construction/win/gen/cmd/gen -world tenants/construction/win/world.yaml -out <tạm> -profile functional -seed 42`;
  manifest sha256 `31172355…`, trùng băm ROADMAP Phụ lục B), chép bản sao, bơm **đúng một dòng** vào `abc-infra/messages.jsonl`
  của bản sao (`diff -r` gốc / bản sao: 1 dòng), rồi chạy `winreport -data <bản> -v -cases <tệp ca tạm>`. Mỗi phép đo chạy cả trên
  bản gốc làm đối chứng. Không sửa dữ liệu hay bộ ca trong repo.
- Ba lớp đo, độc lập nhau:
  1. hai agent đo (nhóm K1, K4, K5 và nhóm K2, K3);
  2. mỗi nhóm một agent khác đo lại từ đầu trong thư mục riêng, thử biến thể dòng bơm, kiểm tệp:dòng, và tìm cách bác (mục 5).
     Bằng chứng nhân quả bằng vá thử / đột biến chỉ làm trong worktree riêng, đã gỡ;
  3. phiên BUILD tự đo lại K2, K4, K5 và phát hiện ngoài K (mục 4) bằng binary, dữ liệu và tệp ca của riêng mình, rồi đối chiếu
     từng tệp:dòng trích dẫn tại HEAD.

## 3. Kết quả

| Mã | Kết quả | Bằng chứng | Ghi chú |
|---|---|---|---|
| K1 | XÁC NHẬN | [ĐO] Bản sao bơm `máy ủi 05 ≈ 3k` (gửi 19/10): "Dầu đội báo của Máy ủi DZ-05 ngày 19/10/2026: 3.000 L.", không ghi chú ước lượng. Cùng dòng với `~ 3k`, `khoảng 3k`, `xấp xỉ 3k`: thêm "Trong tổng có số liệu ước lượng.". Đối chứng gốc: PDF "Machine 03 ≈ 4 m³" ra 4.000 L không ghi chú, dù manifest ghi `approx: true`. Mốc `1c8005b` và `8280f40`: như HEAD.<br>[CODE] `data/normalization/engine.go:199-202`: chỉ ký hiệu `~` được nhận; `≈` là token ký hiệu (nhánh mặc định `text.go:179-181`) nên không tới vòng bí danh (`:205`); `pack/lexicon.yaml:26` khai `≈` nhưng vô hiệu. Vá thử ở `:199` hoặc `:202` (worktree riêng) làm PDF có cờ ước lượng ⇒ đúng chỗ.<br>W-G1: struct manifest ở `win/winrun/winrun.go:61-70` không có trường approx, `:302` chỉ so thực thể / chỉ số / nguồn / ngày / số. Đổi cờ approx trong bản sao manifest: G1 vẫn 2739 / 2739; đổi `value_l`: G1 đỏ | Việc brief §4 mục 3 (cờ approx ở W-G1). Thêm: "2k" (R-001, R-002) manifest ghi `approx: true` nhưng bộ nạp ra `approx=false`; chưa phân xử manifest hay bộ nạp đúng |
| K2 | XÁC NHẬN | [ĐO] Bơm tin gửi 20/10 08:00 "hôm qua báo nhầm, máy 03 là 3k". "Hôm qua đội báo máy 03 bao nhiêu dầu?" vẫn "…ngày 19/10/2026: 4.000 L." (không dấu vết đính chính). "Hôm nay đội báo máy 03 bao nhiêu dầu?" từ INSUFFICIENT_EVIDENCE thành RESOLVED "…ngày 20/10/2026: 3.000 L.", không cảnh báo. "Tuần này đội báo máy 03…" từ 4.000 L thành **7.000 L**: tin đính chính làm tổng tăng. Biến thể người gửi, giờ (cả giờ UTC sang ngày 20/10 giờ +07), tin có "ngày 19/10", tin "đính chính: …": như nhau.<br>[CODE] `data/ingestion/ingest.go:239-253` cắt ngày quan sát về 00:00 ngày GỬI (`:250`); `:235` `On: on`. Span ngày "hôm qua" tính ra ở `:257` bị bỏ: nhận dạng thực thể chỉ đọc chuỗi gốc (`data/entityresolution/resolver.go:81`); đột biến bỏ `Now: on` cho đầu ra y hệt. Trong `data/ingestion` không chỗ nào đọc giá trị ngày | Sai im lặng. Dòng `On: on` ở `:235` (đề ghi `:234`) |
| K3 | XÁC NHẬN | [ĐO] Bơm tin gửi 19/10 18:00 "mai cần 4k máy 03": quan sát 4.000 L gắn ngày GỬI 19/10. "Hôm qua đội báo máy 03…" từ RESOLVED 4.000 L thành SYSTEM_FAILURE (fallback): chat cộng 2000 + 2000 + 4000 = 8.000 lệch PDF 4.000 (dung sai 0); bản nháp có 8.000 L mà không bằng chứng nào chứa giá trị đó nên S9 chặn. "Hôm nay…" vẫn INSUFFICIENT_EVIDENCE. **Cùng tin gửi 20/10 08:00: thành báo cáo thật của 20/10, "…ngày 20/10/2026: 4.000 L.", RESOLVED, im lặng** (không nguồn nào khác phủ ngày đó).<br>[CODE] Nạp như K2; `data/normalization/datetime.go:38` chỉ có "ngày mai", không có "mai" đứng một mình, nhưng không phải nguyên nhân ("ngày mai máy 03 cần 4k" cho cùng kết quả vì tầng nạp không đọc span ngày); `data/sourceoftruth/query.go:199-221` cộng mọi fact cùng loại nguồn; `pack/metrics.yaml:13` dung sai "0" | Lỗi im lặng; chỉ hỏng công khai khi ngày đó tình cờ có nguồn khác (PDF) để lệch. Tin báo thật "máy 03 tối thêm 1k" cũng ra SYSTEM_FAILURE: fallback không đến từ việc nhận ra tin kế hoạch |
| K4 | MỘT PHẦN | [ĐO] Bơm tin trùng (chữ, giờ, người gửi y hệt, chỉ đổi mã tin).<br>- EX-03 (có PDF thẩm quyền cao hơn): SYSTEM_FAILURE, đúng như khẳng định.<br>- **Lu 02 (không có nguồn nào khác cùng chỉ số): RESOLVED "Dầu đội báo của Lu rung RL-02 ngày 19/10/2026: 2.400 L." (gốc 1.200 L), không xung đột, không ghi chú: cộng gấp đôi IM LẶNG.** Gửi lại sau 10 phút, khác người gửi, hay bơm lại đúng mã tin R-017: vẫn 2.400 L.<br>[CODE] `data/sourceoftruth/query.go:91` gom theo loại nguồn; `:201-220` cộng mọi fact cùng loại nguồn (`decimal.Add`, `:216`), không khử trùng theo mã tin, chữ hay giờ. W-G1 cũng không thấy: `winrun.go:276` `obs[o.Key] = o` gộp theo khoá, nên mã tin trùng vẫn ra 2739 / 2739 | "Không cộng gấp đôi im lặng" chỉ đúng khi tình cờ có nguồn khác thẩm quyền làm lệch. Fallback ở EX-03 do S9 chặn số 6.000 trong chú thích xung đột, không phải cơ chế khử trùng. Khử trùng theo giá trị sẽ sai (R-001, R-002 hợp lệ và bằng nhau) |
| K5 | XÁC NHẬN, rộng hơn khẳng định | [ĐO] Hàm chuẩn hoá: "một tỷ hai lít" → 1.000.000.002 L; "một triệu hai" → 1.000.002; "một nghìn hai" → 1.002; "một trăm hai" → 102; "hai nghìn ba" → 2.003 (đối chứng đúng: "một nghìn hai trăm" 1.200, "hai nghìn rưỡi" 2.500). Câu "…được cấp một tỷ hai lít dầu đúng không?": ở `1c8005b` ra "Không. … không phải 1.000.000.002 L." (đúng Phụ lục B #22); ở `8280f40` và HEAD bị cổng miền giá trị che (max 20.000).<br>**Trong miền giá trị thì không che, và phán quyết ĐỔI CHIỀU:** "Hôm qua đội báo lu 02 một nghìn hai lít dầu đúng không?" → "Không. … là 1.200 L, không phải 1.002 L." trong khi "… 1200 lít …" và "… một nghìn hai trăm lít …" → "Đúng.". Ở tầng nạp: tin "máy ủi 05 một nghìn hai lít" thành 1.002 L, RESOLVED, im lặng.<br>[CODE] `data/normalization/numwords_vi.go:106-111`, `:138-146`, `:185-188`: chữ số đứng sau từ chỉ hàng bị cộng như hàng đơn vị | Phán quyết sai chiều ở cổng câu xác nhận: lỗ mức S1, cùng tầm R4 của ADR-0009. "Một nghìn hai = 1.200" là cách đọc khẩu ngữ phổ biến; repo chưa định nghĩa |

## 4. Phát hiện ngoài K1–K5

1. **Đọc số bằng chữ không dấu KHÔNG TẤT ĐỊNH** [ĐO, BUILD]: bơm tin "máy ủi 05 hai muoi lit", hỏi "Hôm qua đội báo máy ủi 05
   bao nhiêu dầu?" 20 lần cùng binary, cùng dữ liệu: 11 lần "10 L", 9 lần "20 L". Người đo lại đo ở hàm chuẩn hoá: 16 / 40 và 24 / 40.
   [CODE] `numwords_vi.go:43-53` dựng bảng bỏ dấu bằng `for w, v := range base` (thứ tự map ngẫu nhiên); "mười" (số 10 đứng đầu)
   và "mươi" (nhân 10) cùng bỏ dấu thành "muoi", từ nào duyệt trước thì chiếm. Bộ 117 / H1 / H3, seed và test: 0 lần "muoi"
   [ĐO, grep] ⇒ các số đo hiện có không bị ảnh hưởng; lỗi đang ẩn. Trái P2 (vỏ tất định) của `CLAUDE.md`.
2. Câu "Ngày 21/10 đội báo máy 03…" khi đồng hồ là 20/10/2026 được hiểu thành ngày 21/10/**2025** [ĐO, người đo lại; BUILD chưa đo
   lại, chưa tìm nguyên nhân].

## 5. Đo lại độc lập

| Mã | Người đo đầu | Người đo lại | Đồng ý | Tệp:dòng đúng | BUILD đo lại |
|---|---|---|---|---|---|
| K1 | XÁC NHẬN | XÁC NHẬN | có | có (sửa một trích dẫn: nhánh tạo token ký hiệu là `text.go:179-181`) | đối chiếu mã |
| K2 | XÁC NHẬN | XÁC NHẬN | có | có (sửa một chi tiết: span ngày bị bỏ, không dùng cho nhận dạng thực thể) | đo + mã |
| K3 | XÁC NHẬN | XÁC NHẬN | có | có | đối chiếu mã |
| K4 | MỘT PHẦN | MỘT PHẦN | có | có | đo (lu 02) + mã |
| K5 | XÁC NHẬN | XÁC NHẬN | có | có | đo (cả câu đổi chiều) + mã |

## 6. Hệ quả cho việc BUILD (đề xuất, chưa làm)

- **K5 trong miền là lỗ S1 của cổng phán quyết** ⇒ đóng trước khi chạy H4 (H4 có ca xác nhận), cùng quy trình R4: test đỏ trước →
  sửa nhỏ nhất → đo → đột biến → soát độc lập.
  - Cách nhỏ nhất không chạm tầng nạp: cổng không phán quyết khi con số bằng chữ có dạng "<từ chỉ hàng> <một chữ số>" (đọc hai
    nghĩa), fail-closed theo ADR-0009 §2.
  - Sửa bộ đọc ("một nghìn hai" = 1.200) đổi cả hành vi tầng nạp; brief §4 chưa cho sửa hành vi tầng nạp trước W-G7 ⇒ chờ quyết.
- **Bảng không dấu không tất định (mục 4.1):** sửa nhỏ là chọn tất định cho "muoi" hoặc bỏ bản bỏ dấu khi hai từ có dấu khác nghĩa
  cùng gấp về một chữ. Cũng là tầng chuẩn hoá dùng chung cho đường nạp ⇒ cùng câu hỏi quyết như trên.
- K2, K3, K4: lỗi tầng nạp và tầng nguồn sự thật. Theo roadmap mới, chỗ sửa tận gốc là Phase 3. K2 (tổng tuần tăng), K3 (gửi ngày
  khác thì im lặng) và K4 (cộng gấp đôi im lặng) đều nặng hơn ROADMAP mô tả.
- K1: phần W-G1 (so cờ approx) là việc §4 mục 3. W-G1 cũng mù trước mã tin trùng (K4): cùng chỗ sửa `checkG1`. Sửa nhận `≈` ở
  `engine.go` là sửa tầng chuẩn hoá.

## 7. Chỗ ROADMAP sai hoặc lệch (để phiên học sửa; BUILD không sửa ROADMAP)

1. K4: "không cộng gấp đôi im lặng" sai khi chỉ số không có nguồn thẩm quyền khác (lu 02: 1.200 → 2.400 L im lặng).
2. K5: "bị cổng miền giá trị che" chỉ đúng khi số đọc sai vượt miền; trong miền thì câu trả lời dội lại số sai và có thể đổi chiều.
3. K3: xếp "lỗi im lặng" (§0.4 #16) là đúng, dù ở ngày 19/10 của seed 42 nó hiện ra thành fallback.
4. K2: số dòng `ingest.go:234` nay là `:235` (nhỏ).

## 8. Giới hạn

- Chỉ đường bộ nhớ; đường PostgreSQL chưa đo.
- Người gửi và giờ của dòng bơm do người đo chọn (đề không nêu); đã thử vài biến thể cho K2, K3, K4, chưa đo tin nhiều dòng hay
  vị trí dòng bơm khác cuối tệp.
- K4 chưa đo trùng ở nguồn không phải chat (CSV, XLSX, PDF) hay trùng giữa hai nguồn ngang thẩm quyền.
- K2, K3, K4 chưa đo ở `1c8005b` / `8280f40`; K1, K5 người đo lại đã đo ở cả hai mốc. Kết quả ở HEAD khớp ROADMAP Phụ lục B #24, #26.
- Chưa chạy bộ test của `data/normalization` với vá thử nào; vá thử chỉ để chứng minh nhân quả, không phải đề xuất sửa.
- Phần còn lại của phản hồi audit (Q2–Q7, Q9–Q13, Phụ lục C) chưa làm; bàn §4.7 hết hiệu lực theo D-RM-01.
