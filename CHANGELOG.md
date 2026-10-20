# Changelog

Mọi thay đổi đáng kể của EAIS được ghi ở đây, theo tinh thần [Keep a Changelog](https://keepachangelog.com/).
Dự án chưa phát hành phiên bản nào. Commit đầu tiên: 20/10/2026.
Trạng thái chi tiết và bằng chứng nằm ở `docs/STATUS.md`.

## [Chưa phát hành]

### Đêm 20→21/10/2026 — khôi phục bằng chứng (chưa gọi model thật, chi 0 USD)

Báo cáo: `docs/bao-cao/2026-10-21-overnight.md`. Câu chờ chủ dự án quyết: `docs/CHO-QUYET-DINH.md`.

**Bằng chứng và phương pháp**
- Kiểm toán nhiễm bộ 117 ca (`docs/g7/CONTAMINATION-AUDIT.md`): 102 sạch, 3 nhiễm (083, 107, 108), 12 không chắc. Đính chính ADR-0002 D7.
- ADR-0005 (luật D5 sửa đổi, đã duyệt), ADR-0006 (bộ giữ lại độc lập), ADR-0008 (một bộ từ vựng nhóm lỗi).
- Bộ giữ lại độc lập `docs/g7/holdout/`: 229 ca do người không đọc mã soạn → ba người soát độc lập + người sửa → 189 ca; loại 7 ca bị lộ cho người viết mã TRƯỚC khi chia; đóng băng `16a6140` trước mọi lần chạy: pilot 16, H1 104, H2 62, exposed 7. 15 ca có gợi ý kỳ vọng lộ sau khi đóng băng: tập không đổi, bảng H1 báo độ nhạy.
- Công cụ: `scripts/dong-bang-giu-lai.py` (chia máy móc, loại ca bị lộ), `scripts/bang-cong-h1.py` (G2' / G5' / G6' + hai độ nhạy đăng ký trước), `scripts/chay-giu-lai.sh` (chạy MỘT lần), `scripts/du-toan-g7.py`, `tenants/construction/win/d5probe` (chương trình đo của người soát D5, công khai).

**G7 (harness; chạy thật vẫn cần khoá + pilot + bộ chấm v2)**
- Client model: không gửi sampling mặc định (temperature từng làm mọi lời gọi lỗi 400); thinking / effort / thử lại từ cấu hình; thời gian lùi có trần 30 s; số lần thử lại ≤ 10; huỷ ctx trả đúng lỗi.
- `g7run`: baseline B2+, B1 (bản ghi thô theo cửa sổ), B2″, B2-oracle; bằng chứng thô `-out`; `-smoke`, `-pilot`, `-budget`; nhận biết chạy khô; lượt vô hiệu khi lỗi gọi model > 2 %; nhãn PROVIDER.
- `CompareG7` theo D5 sửa đổi: ANY-RUN cho quyền / claim không căn cứ; ≥ 4 / 7 lớp (n ≥ 12) hơn ≥ 20 điểm; McNemar một phía; Wilson; kiểm đầu vào (id, lớp).
- Bộ chấm nội dung: 170 đòn tấn công thành test ratchet (nợ 122 → 118); rỗng / giả rỗng = từ chối; hệ lỗi khi hỏi luôn trượt; bộ đếm claim không căn cứ có số được phép riêng từng ca; canary tên.

**Sửa (từ bản soát mã độc lập lượt 1)**
- `g7run` từng ra PASS giả khi mọi lời gọi model của baseline lỗi (S1-1), cùng S2-5, S2-6, S3-2, S3-4, S4 (`a94b5fb`).
- Pipeline S2-1 (biến mẫu thay nhiều lượt có thể ghép thành số mới), S2-2 (lượng trong tài liệu lọt cả hai lớp khi viết ký tự lạ), S2-3 (che chuỗi con khi kiểm số), S2-4 (câu nhiều vế bị trả lời một vế: xem mục Sửa bên dưới) — `5ce22ec`, `3d730ea`.
- Test hổng: ACL rỗng, lộ số viết khác dạng, huỷ khi chờ lùi (`fe22e2a`).

**Sửa (từ bản soát mã độc lập lượt 2, chiều pipeline; người phản bác độc lập đã xác nhận)**

Phần lớn là hồi quy của `5ce22ec` và `3d730ea`: hỏi lại OAN một câu chỉ có một vế sai như cắt thầm (P5), vì thông điệp hỏi lại nói sai sự thật ("nhiều con số khác nhau"). Mỗi lỗi có test đỏ trên mã cũ; đột biến `scripts/dot-bien/n28-sua-hoi-quy.sh` (90 dòng) phủ các nhánh mới, n20 và n25 sửa chuỗi theo mã mới.
- `core/pipeline` — MỘT định nghĩa "con số xác nhận" (`claimNumbers`) dùng chung cho `claimedValue` (trả "Đúng." / "Không.") và `rivalClaims` (hỏi lại), để hai nơi không lệch nhau:
  - A-01: số trong TÊN thực thể ("03" của "máy 03") bị loại theo vị trí của MỌI cụm tên — kể cả bí danh mơ hồ mà router gỡ về một máy (`contracts.AmbiguousMention` nay có `Start`/`End`; trước đây thực thể lấy từ danh sách ứng viên nên không có vị trí) và lần nhắc thứ hai của cùng một máy ("EX-03 (máy 03)"). Câu hỏi lại về con số không còn kể máy ứng viên (từng kể cả máy ngoài dự án đã nêu).
  - A-02 và mục "rivalClaims lệch claimedValue": số trần khác ("năm 2026", "ca 2", "phiếu 912", "2 lần", "tuần 43"; kể cả ở mệnh đề SAU mệnh đề hỏi) là nhãn khi mệnh đề hỏi đã có lượng viết rõ đơn vị — hết hỏi lại oan; số khác chiều đo ("7h", "2 ca") không phải con số xác nhận; số trần có dạng lượng ("4k", "4.000") hoặc đứng ĐẦU một mệnh đề ("3850 L đúng không, 4000 thì sao?") vẫn là con số cạnh tranh. Đánh đổi đã chấp nhận: số trần thuần đứng sau một từ nối ("3850 L đúng không, hay 4000?", "… Còn 4000 thì sao?") nay trả "Đúng." cho 3850 L thay vì hỏi lại — không phân biệt được với nhãn ("Ca 2", "Đội 3") bằng cấu trúc.
  - A-03: mệnh đề chứa cụm xác nhận mà không có số ("… 3850 L và 4000 L, đúng không?") thì câu hỏi áp cho các số đứng trước trong CÙNG câu — trước đây trả tổng như thể chỉ có một vế. Hệ quả: "…, vậy tổng báo 4k, đúng không?" sau các số ngữ cảnh cũng hỏi lại (không phân biệt được với một danh sách); không có dấu phẩy trước cụm xác nhận thì số ngữ cảnh vẫn không tính (WIN-I-103).
- `intelligence/intent`: A-04 — câu ghép số liệu + quy trình nêu ≥ 2 thực thể cùng loại, hoặc có ngày hỏng, mà không có mốc ngày hợp lệ nay hỏi lại thay vì chuyển sang câu hỏi quy trình rồi bỏ thầm phần số liệu. Kỳ LỒNG nhau liền kề ("tháng 10 năm nay", "hôm qua trong tháng 10", "ngày 19/10 năm nay") là MỘT kỳ — kỳ hẹp; "tháng 10 và hôm qua", "tuần trước, từ 12/10" vẫn là nhiều kỳ.
- `data/normalization` (ngày, hậu tố nhân):
  - A-05: năm hai chữ số là 20yy ("19/10/26", "16/9/26" từng bị báo "không hợp lệ" dù là ngày có thật); dấu "." ("19.10.26", "19.10.2026") và — khi có chữ "ngày" đứng ngay trước — "ngày 19.10", "ngày 19-10", "ngày 19-10-26", "ngày 19 / 10" đọc được, theo `docs/tenants/construction/03-data-map.md:162` và `02-domain-map.md:71`, `:296`. Ngày hỏng viết kiểu đó ("31.02", "31-02", "31 / 02", "31/02/26") được ghi lại thay vì rơi về hôm nay. Không có chữ "ngày" thì "19.10" vẫn là số thập phân, "10-12 giờ" khoảng giờ, "TK-01-02-03" mã. Chữ "ngày" đứng trước một LƯỢNG không biến lượng thành ngày: nhóm 3 chữ số ("mỗi ngày 4.000 lít") không phải tháng; khoảng hay số thập phân có đơn vị theo sau ("mỗi ngày 10-12 giờ", "ngày 2.5 tấn") là lượng. Đánh đổi: "ngày 19.10 ca đêm" không đọc được vì "ca" cũng là đơn vị ("2-3 ca"); "ngày 19/10 ca đêm" thì đọc được.
  - A-06: chỉ cụm CÓ DẠNG ngày (ngày ≤ 31 và tháng ≤ 12; hoặc có chữ "ngày" / năm bốn chữ số kèm theo) mới bị ghi là ngày hỏng; "24/24", "60/40", "12/34", "12/34/56", "1/2/3" không còn bị hỏi lại "Ngày … không hợp lệ" khi người hỏi không nêu ngày.
  - A-07: "ngày D tháng M [năm Y | /Y]" là MỘT ngày (trước đây chỉ đọc "tháng M" và bỏ "ngày D": "hôm nay, ngày 20 tháng 10" bị hỏi lại oan, "ngày 15 tháng 9" trả cả tháng); "từ ngày D đến/tới ngày D2 tháng M" là hai đầu mút (đầu mút thứ nhất mượn tháng, năm của đầu mút sau) nên hỏi lại; "ngày 31 tháng 2" là ngày hỏng.
  - Mục người phản bác tìm thêm: cụm dd/mm dính sau mã chữ ("PX-12/10", "BB-24/50") hay là phần sau của một dãy số dài ("2/3" của "1/2/3") không phải ngày.
  - A-13: hậu tố nhân cách số bằng nhiều dấu cách hay ký tự vô hình ("5  k lít", "5<ZWSP>k lít") là cùng lượng với "5k lít".
- Lớp kiểm lượng trong tài liệu (`reliability/validation` và `data/normalization` ở chế độ Layout; hai lớp vẫn viết độc lập):
  - A-08: nay bị chặn ở cả hai lớp: "mét khối" và "m^3" (**pack: hai bí danh MỚI của đơn vị m3, `tenants/construction/pack/lexicon.yaml` — đổi cả cách đọc câu hỏi: "5 mét khối dầu" là 5.000 L thay vì "5 mét"; `PackVersion` đổi theo băm nội dung**), đơn vị mở đầu ngoặc kèm chữ phụ ("5.000 (lít dầu)"), số trong ngoặc ("(4.000) lít"), đơn vị ghép đứng trước số ("lít/ca: 4.000", "(lít/ngày): 5.000").
  - A-09: hết chặn oan câu quy trình hợp lệ: mẫu số của giá ("đồng/lít: 25.000"), nhãn bộ phận / mục ("khối 2", "L 3"), số đếm ("lít 2 liên"), số đứng đầu mục trong ngoặc ("(1) Khối tài chính"). Đơn vị đứng trước số chỉ tính khi có dấu cột rõ (":", "=", ")") hoặc chỉ cách số bằng khoảng trắng mà số có dạng lượng (dấu nghìn / thập phân, hoặc từ 2 chữ số). Đánh đổi: "L 5" (MỘT chữ số, chỉ cách bằng dấu cách) không còn bị chặn, còn "khối 12" (nhãn bộ phận hai chữ số) vẫn bị chặn.
- A-10 (test): các ca vắng của A-01…A-09, A-13 và ba mục người phản bác tìm thêm. Hai test từng KHẲNG ĐỊNH hành vi sai "16/9/26 không hợp lệ" (`TestNgayHongDuocGhiLai`, `TestNgayKhongHopLeThiHoiLai`) đổi sang "31/09/26" (năm hai chữ số, tháng 9 không có ngày 31).
- A-11 (tài liệu): mục S2-4 từng viết "không bao giờ trả lời riêng ngày đầu" — chỉ đúng khi CẢ HAI đầu mút nhận ra được. **Còn lọt, đã đo, CHƯA làm** (trả lời MỘT mốc im lặng, P5): "từ ngày 19/10 đến nay" (chỉ ngày đầu), "10/10-15/10" và "từ 10-15/10" (chỉ ngày CUỐI), "7 ngày qua", "quý 3", "từ đầu tháng" (rơi về hôm nay). Phân số trông như ngày hợp lệ ("3/4 ca", "1/2 ngày", "24/7") đứng cạnh một kỳ khác vẫn thành "nhiều mốc thời gian" (an toàn; tác giả S2-4 đã đo và chấp nhận).
- Không sửa: A-12 (người phản bác bác bỏ: số đột biến đã đúng, chú thích toạ độ không sai); A-08b (bộ đếm độc lập của G7 ở `evals/` và `tenants/construction/win/g7run`: ngoài phạm vi bản sửa này — cùng điểm mù lượng viết "mét khối", "m^3", "liters"… với A-08).

**Sửa (hồi quy R-xx do bản sửa lượt 2 gây ra; bản soát độc lập vòng sau, chiều pipeline)**

`6f23ff3` bỏ hỏi lại oan nhưng làm hỏng đúng những câu mà bản cha `1276b5f` hỏi lại / không phán quyết: câu xác nhận trả "Đúng." / "Không." cho SAI kỳ hay SAI con số, hoặc rơi im lặng về hôm nay. Mục tiêu bản này: khôi phục hành vi an toàn của bản cha ở đúng các câu đó mà KHÔNG mất các sửa A-01…A-13 (mọi `*_hoi_quy_test.go` của `6f23ff3` giữ nguyên, kể cả kỳ vọng). Test đỏ trên mã trước khi sửa: `tenants/construction/pipeline_hoi_quy_r_test.go` (13 hàm đầu–cuối; 12 đỏ trên mã trước khi sửa, hàm còn lại là đối chứng của A-02) + test đơn vị ở từng tầng; danh mục đột biến `scripts/dot-bien/n35-sua-hoi-quy-R.sh`; n20 / n25 / n28 / w5-rag-hybrid sửa chuỗi gốc theo mã mới, không xoá dòng.
- `core/pipeline`: `rivalClaims` nay là `confirmClaims` và trả thêm các mốc thời gian không kiểm được (`claimSet.doubts`); `claimNumbers` trả `claimSet`.
- R-02 (`core/pipeline`, `claimNumbers`): số trần NỐI với một lượng bằng từ nối ("từ 3800 đến 3900 L", "3800-3900 L", "3850 hay / hoặc / chứ không phải 4000 L", "3850 L hay 4000"; "3850 L (hoặc 4000)") dùng chung đơn vị của lượng đó và là con số cạnh tranh → hỏi lại, không còn "Không. … không phải 3.900 L" cho dải chứa 3.850 L (`joinClaims`); mệnh đề hỏi mở đầu bằng "hay / hoặc / chứ không phải" sau dấu phẩy nối tiếp mệnh đề trước ("…3850 L, hay 4000 L đúng không?"); lượng viết kèm mẫu số ("5.000 L/ngày": định mức, tốc độ) không còn là "lượng viết rõ đơn vị" nên không cho phép bỏ thầm số trần 3850 của câu.
- R-01 (`data/normalization` + `core/pipeline`): "tháng M năm Y" đọc đủ năm bằng `monthYearAt` (trước đây "tháng 9 năm 2025" thành tháng 9 NĂM NAY — sửa cả dạng không xác nhận); câu xác nhận có mốc thời gian viết bằng chữ mà bộ chuẩn hoá không đọc ("năm 2025", "tuần 41", "quý 3", "ngày 12", "hôm 15", "15 tháng 9", "7 ngày qua", "19-10", "19-10-26") thì HỎI LẠI khe thời gian và nói rõ mốc nào (`normalization.PeriodQualifier`, `DashDateAt`, `claimSet.doubts`) — trừ khi kỳ đã đọc nằm TRỌN trong mốc ("hôm nay tuần 43" khi hôm nay thuộc tuần 43: bối cảnh vô hại, test A-02 giữ nguyên; "hôm nay tuần 44", "hôm nay năm 2025", "tháng 10 ngày 12" thì hỏi lại). Tuần là tuần ISO 8601, quý là quý dương lịch — chỉ để KIỂM khớp, không để đọc mốc.
- R-04 (cùng hàm): nhãn và số đếm ("phiếu 912", "ca 2", "đội 3", "xe số 7", "khối 2", "L 5", "2 máy", "3 lần", "3-4 chuyến", "mỗi ngày 3 chuyến" — `normalization.LabelContext`, hai danh sách đóng) không còn là con số xác nhận kể cả khi câu không nêu lượng nào ("Tuần 43 EX-03 được cấp dầu đúng không?" hỏi lại mốc; "… theo phiếu 912 đúng không?" trả số, không phán quyết). Hệ quả tốt ngoài lời giao: hết hỏi lại oan "nhiều con số" ở các câu có nhãn mà không có lượng viết rõ đơn vị.
- R-05 (`data/normalization`): "mỗi / hàng / mọi / từng / một / cứ + ngày" nói TỐC ĐỘ nên "mỗi ngày 3-4 chuyến", "mỗi ngày 2.5 chuyến" không còn đọc thành ngày 03/04 hay 02/05 (`afterDayWord`).
- R-06 (`intelligence/intent`): câu QUY TRÌNH thuần nêu hai thực thể ("Khi lệch phiếu kho của EX-03 và RL-02 thì phải làm gì?", "Thủ tục xin cấp dầu cho dự án A và dự án B?") lại là câu hỏi quy trình (RAG); chỉ hỏi lại khi câu có dấu hiệu hỏi SỐ LIỆU — cụm hỏi lượng (`quantity_markers` của pack, mới nối vào `intent.Options.QuantityMarkers`), cụm xác nhận, mốc ngày / ngày hỏng, hay một mệnh đề riêng chỉ có từ khoá số liệu (`Set.asksData`); mọi ca của A-04 giữ nguyên.
- R-07 (`data/normalization`, `dateLike`): đảo vế tháng/ngày ("10/19", "10/19/26", "11/30") và ngày lố một chút ("32/10" … "39/12") lại là ngày hỏng → hỏi lại.
- R-10 (`core/pipeline`, `sentenceStart`): dấu chấm lửng "..." giữa câu không phải hết câu — "Hôm nay EX-03 được cấp 3850 L ... đúng không?" lại có "Đúng."; một dấu chấm đơn rồi "Đúng không?" vẫn là câu mới (chủ ý của A-03).
- R-03 (`reliability/validation`, CHỈ lớp kiểm ĐỘC LẬP `ForbidUnitWords`; hai lớp vẫn viết riêng): số và đơn vị bị chen dấu trang trí — Markdown (`**4.000** lít`, `` `4.000` lít ``, `*4.000* *lít*`), thẻ HTML (`<b>4.000</b> lít`), dấu phẩy (`4.000, lít`), ô bảng (`| 4.000 | lít/ca |`: đơn vị phải là TOÀN BỘ ô kế tiếp) — và nhãn đơn vị trong ngoặc (`(đơn vị: lít)`, `(ĐVT: lít)`, `(tính bằng lít)`) nay bị chặn.

Đánh đổi đã chấp nhận:
- R-07: "19/13", "24/24" (một vế hợp lệ, vế kia chỉ hơn mức một chút) không phân biệt được với tỉ lệ / số phiếu bằng cấu trúc nên KHÔNG hỏi lại khi thiếu chữ "ngày" (có chữ "ngày" thì vẫn hỏi lại); ngược lại tỉ lệ nhỏ trên lớn ("5/20", "1/25") có dạng đảo vế nên bị ghi là ngày hỏng — hỏi lại thừa còn hơn rơi im lặng về hôm nay.
- R-06: câu quy trình tách mệnh đề bằng dấu phẩy mà một mệnh đề chỉ có từ khoá số liệu ("Việc cấp dầu cho EX-03 và RL-02 phải làm thế nào, khi lệch phiếu kho thì xử lý ra sao?") vẫn bị hỏi lại; pack không khai `quantity_markers` và `confirm_markers` thì mọi câu nhiều thực thể giữ như A-04 (hỏi lại).
- R-01: mốc chưa đọc được chỉ làm hỏi lại ở câu XÁC NHẬN; câu không xác nhận ("EX-03 tuần 41 được cấp bao nhiêu dầu?") vẫn rơi về hôm nay như trước (có sẵn, đã khai ở mục "Chưa làm" phía trên) — CHƯA làm. "quý 4 năm 2026" đứng một mình hỏi lại (không đọc quý / năm thành kỳ). Khoảng đếm ("7 ngày", "2 tuần") luôn hỏi lại, kể cả khi câu có kỳ đọc được ("hôm nay chạy 7 ngày liên tục … đúng không?"): không kiểm được nó với kỳ nào. Tần suất "6 tháng 1 lần" (mỗi 6 tháng một lần) có dạng "N tháng M" nên bị hỏi lại ("6 tháng 1" đọc như ngày 6 tháng 1; trước bản này nó chỉ thành "tháng 1" im lặng).
- Nhãn / số đếm là danh sách đóng (nhãn: phiếu, số, mã, ca, đội, tổ, lô, đợt, thứ, khu, trạm, khối, lít, L; đếm: lần, máy, xe, chuyến, đội, tổ, đợt, lượt, phiếu, phút, giây, thiết bị, dự án, vị trí): nhãn lạ vẫn theo luật A-02 (bỏ khi câu có lượng viết rõ đơn vị, tính khi không). Từ nối chỉ nối hai con số LIỀN NHAU; "3850, 4000 L đúng không?" (dấu phẩy thuần) vẫn coi 3850 là ngữ cảnh như bản cha; "3850 L đúng không, hay 4000?" vẫn là đánh đổi A-02 đã khai.
- R-03: lớp chuẩn hoá (chế độ Layout) CHƯA bắt các dạng chen dấu trang trí — chỉ lớp độc lập bắt; hai lớp vẫn không dùng chung mã.

Nợ còn lại (đã đo, KHÔNG làm ở bản này): R-08 ("ngày 19.10 công trình A", "ngày 19-10 người lái" rơi về hôm nay: "công", "người" là đơn vị; chú thích "dd/mm có dấu '/' luôn đọc được" ở `pipeline_hoi_quy_test.go` đúng cho "/" nhưng sai cho "ngày 19 / 10 ca đêm"); R-09 (bí danh "mét khối" viết hở: "5 mét  khối", "5 mét\nkhối", "5 met khoi" lọt cả hai lớp); R-11 ("ngày 1 và ngày 2 tháng 10" chỉ ra ngày 02/10; "từ ngày 28 đến ngày 3 tháng 10" mượn tháng 10 và lùi năm); R-03 ở lớp chuẩn hoá Layout; mốc chưa đọc được ở câu không xác nhận; A-08b.

**Sửa (sau H1 — H1 chỉ còn là bộ hồi quy; người phản bác cuối và các người soát độc lập vòng sau, không ai thấy bộ giữ lại)**
- Harness G7 (`g7run`, client model):
  - soát lượt 2 chiều C: không ra kết luận G7 khi bộ chấm chưa đóng băng, model / tham số lệch hợp đồng, chạy khô hay model giả (C-06, `d4b80be`); trần ngân sách bắt buộc, ≤ 15 USD, ghi đúng manifest (C-07); bản ghi quyết định của EACL trong bằng chứng thô (C-11); thử khói có hệ cổng bị chặn là KHÔNG ĐẠT (C-13); không theo chuyển hướng HTTP, khoá chỉ qua https hoặc loopback (C-18);
  - soát C2 (F-01…F-11, `a928747` … `2304d93`): lượt chạy thật bị từ chối TRƯỚC lời gọi model khi lệch tham số hợp đồng, chấm full, bộ chấm chưa đóng băng, thiếu / không ghi được `-out`, bộ ca sai định dạng, id trùng, hệ cổng bị chặn; URL có `user:pass@` bị từ chối;
  - soát C3 (`246eabb`): endpoint so theo NGHĨA (`:443`, chữ hoa, dấu chấm cuối tên máy là endpoint chính thức — trước đây bị coi là chạy khô, bỏ qua mọi kiểm và gọi hết các ca bằng khoá thật); mọi endpoint không phải loopback (cả proxy) qua kiểm trước; thử khói tốn tiền N ≤ 3 và `-out` phải ghi được; lượt chính phải đủ hệ cổng; thông báo lỗi và `manifest.json` chỉ ghi `scheme://host`.
  - soát `fd4d19c` + `246eabb` (F6, `15ed135`): lời gọi model lỗi truyền tải (kết nối bị từ chối, hết giờ) từng in URL ĐẦY ĐỦ có khoá trong path / query / fragment ra stdout, `ket-qua.md`, `raw-runs.jsonl` — nay bóc về `scheme://host`.
- Luồng hỏi, câu xác nhận (`37e8a08`): gỡ luật A-02 ("số trần cạnh lượng viết rõ đơn vị là nhãn"). Hai người soát bản sửa R tìm ra luật này đổi câu mơ hồ thành phán quyết SAI ("Định mức 5.000 L nên … 3850", "3800 ~ 3900 L", "4000 L hay 3850"), và vá bằng danh sách đóng vẫn rò. Nay MỌI con số ứng viên đều tính (trừ số trong tên thực thể, số thuộc kỳ đã đọc, số hiệu sau từ nhãn); từ hai con số trở lên thì hỏi lại. Từ chỉ kỳ xét trước từ nhãn ("Quý 3 dự án A" là mốc); "tuần thứ 41", "ngày số 12" là mốc. Lớp kiểm tài liệu: "Khối" mở đầu một tên riêng ("Bước 2, Khối Kế hoạch", "Mục 3 Khối Vận hành") không phải đơn vị của số thứ tự đứng trước — hết chặn oan; "**5** khối", "**5** L", "| 5 | khối |", "4.000 Khối Dầu" vẫn bị chặn (`64daad0`: bản miễn đầu tiên của `37e8a08` theo dấu trang trí làm lọt chúng — người soát 37e8a08 tìm ra, RM-1 / F7). "N-M" có dạng ngày đứng sau từ nhãn ("phiếu 15-9", "phiếu 12-10") là nghi vấn ngày ⇒ hỏi lại (`64daad0` gỡ bản miễn R-A14 của `37e8a08`, F6).
- Nhóm nghìn ngăn bằng MỘT dấu cách (`03-data-map.md` hàng 14; `8ff3db5`, `d1a764b`, `9b54b6f`): "3 850 lít", "4 000 lít" là MỘT lượng, trừ khi con số đứng sau từ nhãn / đếm / kỳ trong danh sách của bộ chuẩn hoá ("đội 3 500 lít" là đội 3, 500 L). Ca 264 của H1 nay "Đúng. … 1.490 L". Còn lỗi đã đo (D-OP-9): C3-02 nạp "bể 2 500 lít" thành 2.500 L; C3-05 "tổng số 3 850 lít" → "không phải 850 L"; "bể #2 500 lít" vẫn gộp. Hai bản thay thế bị người soát độc lập bác: `fd4d19c` (không gộp khi có chữ đứng trước) sinh SỐ MỒ CÔI — "… 3850 lít, máy chạy 12 000 km đúng không?" → "không phải 12 L", "sửa chữa 500 000 đồng" nạp thành 500 L — đã revert (`2a87e63`); `3066044` (luôn gộp + cờ chưa phân định) để nhóm nuốt số của tên thiết bị — "máy 3 850 lít, phiếu 912" nạp thành 912 L — không vào main (nhánh cục bộ `thu/nhom-nghin-3066044`).
- Đột biến: danh mục mới n36 (G1 / G2), n37 (nhóm nghìn), n38 (luật xác nhận an toàn), n39 (endpoint tốn tiền); n28 / n35 theo mã mới — 6 dòng của luật A-02 đã gỡ được bỏ có lý do; A02b, R02a, R02f ghi tương đương có lý do; các đột biến sống khác có thêm ca test.

Đánh đổi đã chấp nhận:
- Câu xác nhận có từ hai con số ứng viên trở lên luôn hỏi lại, kể cả khi người đọc đoán được ("… cho bể 7 3850 L đúng không?", "theo phiếu 12-10"): hỏi lại thừa thay vì phán quyết sai.
- H1 làm bộ hồi quy trên bản chốt (`15ed135`): 89/104, 0 ca đổi nhãn so với lần chạy chính thức; ca 264 nay đúng cả nội dung ("Đúng. … 1.490 L"; lần chạy chính thức tính ĐẠT dù câu trả lời lúc đó SAI).

Đính chính (người phản bác cuối, sáng 21/10): 15 ca có gợi ý kỳ vọng lộ cho phiên chính (H1 8, H2 5, pilot 2) lộ lúc 22:50:28, TRƯỚC commit đóng băng 33–76 giây — không phải "sau khi đóng băng" như `59ee556` ghi. Theo ADR-0006 §5 lẽ ra phải loại trước khi chia: lệch quy trình, ghi ở ADR-0006 §5 và báo cáo sáng mục 3; H2 không còn sạch.

Nợ còn lại (đã đo, KHÔNG làm đêm nay): câu xác nhận — trên 15 câu thăm dò của người soát, đầu đêm `94bbaae` phán quyết sai 8/15, bản cuối sai 15/15; 7 câu là HỒI QUY của chính các bản sửa đêm nay (`3d730ea`: "kho 2 được cấp dầu"; `6f23ff3`: "3850 kg", "15-9 máy xúc", "ngày 12 ca 2", "hôm nay ca 2"; `a8b0858`: "con số 3850", "con số 4000 …, kho xuất 3850 L"), 8 câu sai từ trước ("tại bể 7", "chứng từ 912", "Q3", "tuần44", "không được cấp", "trên 3000 L", "15th Sep", "sáng nay") — bảng ở báo cáo sáng mục 8, câu chờ quyết D-OP-10; C3-01 (bộ đếm v1 đọc "9 999 lít" thành hai số — làm ở bộ chấm v2 trước khi đóng băng); C3-03 (câu HYBRID mất phần số liệu khi phần tài liệu lỗi `bad_response`); `joinClaims` sau khi gỡ A-02 chỉ còn dùng chung đơn vị — ứng viên gỡ để đơn giản hoá.

### Thêm (20/10/2026) — WIN MVP: G1–G6 có bằng chứng đạt, G7 chặn ở khoá model

Chi tiết và lệnh chứng minh: `docs/STATUS.md`; các quyết định tự chốt: `docs/adr/0002-win-quyet-dinh-trien-khai.md`.

**Dữ liệu và bộ ca WIN**
- `tenants/construction/win/`:
  - thế giới đóng băng (`world.yaml`) + `CONTRACT.md`;
  - bộ sinh dữ liệu có seed: hồ sơ functional ~2,7 nghìn bản ghi, smoke ~360 nghìn; manifest + biến thể metamorphic;
  - 117 ca do phiên không đọc mã soạn (`cases-independent.yaml`);
  - `winrun`, `winreport`; ratchet `passing.txt` 117/117.
- `data/ingestion`: nạp CSV / XLSX / JSON / JSONL / PDF văn bản (`pdftotext`) theo `pack/sources.yaml`; mọi bản ghi không nạp được vào cách ly kèm lý do.

**Luồng hỏi**
- Kết quả có kiểu (`Outcome`).
- Câu xác nhận so tuyệt đối với nguồn có thẩm quyền; câu nhiều mệnh đề.
- Vật tư mặc định (D7); ý định yếu; cụm bỏ qua.
- So sánh hai nghĩa (`Catalog.Compare`).
- Route `RAG` / `HYBRID_RETRIEVAL` với gateway model theo cấu hình (model giả tất định có đếm lời gọi).
- Bản ghi quyết định + `Replay` (G6).
- `intelligence/rag`: chỉ mục tri thức theo tenant + ACL tài liệu, so khớp theo luật dấu (`normalization.ToneKey`).
- `reliability/validation`:
  - `BindClaims` + `CountNumbers`: ràng buộc claim 6 chiều, bộ đếm số độc lập;
  - `Bound`: claim gắn bằng chứng;
  - `ForbidDimensions` + `ForbidUnitWords`: lượng dầu không lấy từ tài liệu.

**PostgreSQL**
- `deploy/docker-compose.yml`: PG 17.11 ghim digest, chỉ loopback, tmpfs.
- `db/migrations/0001_win.sql`: RLS FORCE theo GUC `eais.tenant_id`, vai `eais_app`.
- `platform/pgstore`: Migrate / Load / Facts / Entities / Retrieve.
- Bộ ca WIN chạy trên PG ra từng câu giống đường bộ nhớ.

**G7 (bộ khung; chạy thật cần khoá model)**
- `evals/baseline`: baseline RAG theo D6.
- `evals/g7.go`: luật D5, chấm theo nội dung, bộ đếm độc lập rò rỉ / số không căn cứ.
- `tenants/construction/win/g7run`: lệnh chạy G7.
- `intelligence/models`: client API kiểu Messages (`Messages`), `New` + `ConfigFromEnv`.

**Vận hành và đồng bộ** (phiên dừng 10:33 theo lệnh chủ dự án)
- `docs/ops/`: bàn giao công khai, chạy lại mốc ở máy khác (`README.md`), luồng tự chạy (`LUONG-TU-CHAY.md`), nhật ký ngày đã làm sạch (`NHAT-KY-2026-10-20.md`), bộ nhớ dự án (`BO-NHO.md`).
- `docs/bao-cao/2026-10-20-win-mvp.md`: báo cáo ngày.
- `scripts/dot-bien/`: 69 đột biến có danh mục, chạy lại được.
- `docs/02-migration-audit.md`: thay đường dẫn monorepo nội bộ của hệ cũ bằng "hệ thống cũ" (repo public).

**Cổng và công cụ**
- `architecture_test.go`: kiểu `*Request` không được mang tenant.
- `scripts/scale-smoke.sh`: scale smoke một lệnh.
- `scripts/mutate.sh`: nhận `PKGS`.

### Đính chính (20/10/2026 tối)

- Bản soát độc lập công thức G7 (`docs/g7/REVIEW-D5.md`) chỉ ra ba kỳ vọng (083, 107, 108) do phiên viết mã trọng tài sau lần đo đầu, trước khi băm bộ ca. Kết luận "6/7 cổng đạt" là có điều kiện; vô điều kiện là 3/7 (G1, G3, G4). Chờ D-G7-11.
- D5 đòi temperature 0 nhưng model hiện hành trả lỗi 400 với mọi temperature khác mặc định. Đề xuất sửa ở ADR-0005; checkpoint G7 ở `docs/g7/`.

### Sửa (20/10/2026)

- `Policy.Allows("")` từng **mở mặc định**. Nay đóng mặc định, và pack nào có ý định chạm dữ liệu / model mà thiếu capability thì bị chặn lúc nạp.
- `sourceoftruth.Query` duyệt map nguồn nên thứ tự bằng chứng không tái lập được. Phát lại G6 bắt được.
- Bộ chuẩn hoá đọc "-150 L" thành 150, nên hiệu âm bị cổng S9 chặn khi giao.
- Truy xuất so khớp bỏ dấu: "bí" khớp "bị".
- Truy xuất cộng điểm theo thứ tự duyệt map, nên đoạn hoà điểm đổi chỗ giữa các lần hỏi. Nay cộng theo thứ tự từ cố định; hoà điểm thì xếp theo tài liệu rồi dòng.
- S2-4 (bản soát độc lập, P5): câu nêu **nhiều thực thể cùng loại, nhiều kỳ, khoảng ngày "từ X đến Y", ngày không có trên lịch, hoặc câu xác nhận nhiều con số khác nhau** từng bị cắt thầm về vế đầu (hoặc rơi về ngày hôm nay với "31/02") và trả RESOLVED / INSUFFICIENT_EVIDENCE như thể đó là cả câu hỏi. Nay là `CLARIFICATION_REQUIRED` bằng mẫu `clarify_missing` sẵn có của pack:
  - `intelligence/intent`: khe chỉ nhận MỘT giá trị — `Intent.Missing` giữ khe đó, `Result.Unclear` ghi các thực thể / kỳ / ngày hỏng đã nhận ra (thứ người hỏi viết rõ không đi qua `Candidates`, nên router không gỡ thay họ bằng quyền xem). Máy kèm dự án chứa nó ("EX-03 của dự án A") vẫn là một chủ thể.
  - `data/normalization`: `NormalizedInput.InvalidDates` ghi cụm có dạng ngày mà không có trên lịch ("31/02", "32/10", "29/02/2026", "31/09/26"); span không đổi (31/2 vẫn là hai số trần). Lượt soát 2 sửa các dạng ngày (xem mục "Sửa (từ bản soát mã độc lập lượt 2…)" ở trên): "16/9/26" là ngày có thật, không còn ghi là ngày hỏng.
  - `core/pipeline`: con số xác nhận tranh nhau (mệnh đề hỏi và các mệnh đề sau; số ngữ cảnh ở mệnh đề trước và giá trị trùng sau khi đổi đơn vị không tính) → hỏi lại với khe `claimed`; câu hỏi lại vẫn lọc ứng viên theo phạm vi dữ liệu của người hỏi; ngày hỏng nằm trong vết quyết định.
  - "từ X đến Y" **chưa được trả lời như một khoảng** (cần quy tắc chuẩn hoá mới): khi CẢ HAI đầu mút nhận ra được ("từ 10/10 đến 15/10", "từ ngày 10 đến ngày 15 tháng 10") thì là hai kỳ nên hỏi lại, không trả lời riêng ngày đầu. Đầu mút chưa nhận ra thì VẪN trả lời MỘT mốc im lặng — danh sách các dạng còn lọt ở mục "Sửa (từ bản soát mã độc lập lượt 2…)" ở trên.
  - Bằng chứng: `scripts/dot-bien/n25-nhieu-ve.sh` (37 đột biến).

### Thêm (19/10/2026)

**Tài liệu**

- `docs/01-architecture.md`: nguyên tắc P1–P13, ba lối vào (QUERY / COMMAND / EVENT), pipeline S0–S10 / C0–C7 / E0–E6, bốn tầng dữ liệu (quan sát → đo → fact → fact dẫn xuất), hợp đồng thành phần, pack, cổng capability, ranh giới Go ↔ Python.
- `docs/02-migration-audit.md`: đối chiếu chuyển đổi từng tệp, mốc đo, các pha 0–8 kèm cổng chứng minh.
- `docs/CONVENTIONS.md`: mã định danh, 12 route, nhóm và nhãn lỗi, định dạng ca kiểm thử, luật con số, luật dữ liệu cho repo public.
- `docs/03-proof-harness.md`: hướng PROVE (thang bằng chứng D0–D4, đếm độ phủ thay cho đếm số ca, luật tuyên bố, chuẩn hoá *nghĩa*, test metamorphic), Sao Bắc Đẩu là lát dọc "Đối soát nhiên liệu", bảng phân loại 29 điểm audit điểm mù.
- `docs/04-core-mvp.md`: **WIN MVP — Enterprise Truth & Routing Proof** (hướng đã chốt 19/10 21:01).
  - Nội dung: luận đề, 5 bất biến CORE, một bài toán Fuel Discrepancy Resolution, chỉ QUERY, tenant A + canary B, baseline RAG bắt buộc, 7 cổng G1–G7, danh sách KHÔNG làm, các lát W1–W7.
  - Thay bản đề xuất lúc 20:5x. Bản đó rộng hơn (lệnh, tác vụ duyệt, thể thao) và bị thay sau khi user thu hẹp phạm vi; 10 điểm còn hiệu lực từ lượt soát độc lập đã được sửa.
- `docs/05-master-roadmap.md`: bản đồ Phase 0–10 **có điều kiện** (không phải TODO list); luật KEEP/FIX/BUILD/DEFER/REMOVE; cổng "ai sở hữu?"; chỗ của các thứ đã có trên bản đồ.
- `docs/STATUS.md`: tiến độ, bằng chứng đã chạy, lỗ hổng đã đo, quan sát ngoài phạm vi, việc tiếp theo.
- `docs/requirements/catalog.yaml` + `catalog.md`: sổ 422 yêu cầu truy về 13 nguồn (chưa soát độc lập).
- `docs/tenants/**`, `tenants/*/evals/cases.yaml`: tài liệu và ca của hai tenant tham chiếu (100 ca cầu đường, 80 ca thể thao; chưa soát độc lập).

**Mã Go** (module `github.com/luannt2002/enterprise-ai-systems`, Go 1.26, toolchain go1.26.4)

- `core/contracts`: kiểu dùng chung và nhãn lỗi → mã HTTP.
- `core/decimal`: số thập phân không float.
- `core/tenant`: nạp và kiểm pack lúc khởi động.
- `core/pipeline`: lối vào QUERY S0–S10 cho các route `SQL_QUERY`, `CALCULATION`, `CLARIFICATION`, `REJECT`, `HUMAN_ESCALATION`.
- `data/normalization`, `data/entityresolution`, `data/sourceoftruth`: chuẩn hoá tiếng Việt, nhận dạng thực thể, nguồn sự thật theo thẩm quyền trong từng chỉ số.
- `intelligence/intent`, `intelligence/router`: ý định theo luật; route kèm capability và phạm vi dữ liệu.
- `reliability/validation`: S9 kiểm số và mã trong câu với bằng chứng.
- `observability/traces`: đo thời gian từng bước.
- `simulations/fixtures`: cửa nạp dữ liệu giả lập đã đóng băng (đổi đơn vị, kiểm miền giá trị).

**Tenant, test và công cụ**

- `tenants/construction`: pack 6 tệp, seed giả lập 8 thực thể + 12 fact, test đầu–cuối 18 ca, test metamorphic (câu gốc + 9 cách viết khác), ca nạp hỏng, ca sai tenant.
- `architecture_test.go`: cổng "mã dùng chung không mang literal tên tenant".
- `scripts/mutate.sh`: chạy một đột biến, khôi phục và kiểm sha256, tách riêng đột biến không biên dịch.
- `evals`: nạp + kiểm hợp đồng bộ ca theo CONVENTIONS §5 (23 khoá, id, tenant, loại lối vào, category, severity, route), có ca âm.

### Sửa (19/10/2026)

- **Luật dấu.** Khớp kiểu bỏ dấu chỉ khi cả câu gõ không dấu. Trước đây "bao" trong "bao nhiêu" khớp nhầm "báo".
- **Ý định.** Áp luật khớp dài nhất trên từ khoá của mọi ý định. Trước đây câu gõ không dấu "bao nhieu dau" bị hiểu thành "dầu đội báo" và trả sai chỉ số.

- **`docs/03-proof-harness.md` §5.** Ví dụ metamorphic cũ coi ERP / phiếu kho / tin nhắn là "cùng một fact", trái luật tách nghĩa. Đã sửa thành cùng một bản ghi của cùng một nguồn ở nhiều định dạng (phát hiện của soát độc lập).

### Bảo mật (19/10/2026)

- **S1 TENANT.** Principal thuộc tenant khác với runtime bị từ chối với lỗi `policy`, trước khi chạm vào pack hay dữ liệu.
