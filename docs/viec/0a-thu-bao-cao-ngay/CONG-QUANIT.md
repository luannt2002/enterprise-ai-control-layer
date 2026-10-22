# Cổng QuânIT — Phase 0a: thu báo cáo ngày của đội hiện trường

> **Trạng thái: GIẤY, chưa có mã.**
> - Mã Phase 0a (khối M2 của MVP) chỉ mở SAU khi chạy xong H4 (`docs/PROMPT-BUILD-EAIS.md` §9.2, làn 4: "0a (M2): sau khi H4 chạy xong").
> - H4 đã khoá (`15f04f9`); tại mốc đọc mã `2b80a38` thì CHƯA chạy, và đã chạy MỘT lần ngày 22/10 sau đó (`docs/STATUS.md`, dòng "H4, 22/10 14:59") [REPO]: điều kiện này đã đạt.
> - Soát (mã ghi chú F01–F03, R2-01…R2-09, R3-01…R3-05 và R4-01…R4-04 định nghĩa ở `docs/adr/0012-phase0a-thu-bao-cao-ngay.md`, mục "Ghi chú soát đã xử lý"):
>   - vòng 1 (người soát ≠ người viết): **ĐẠT CÓ GHI CHÚ**, ba ghi chú S3: F01, F02, F03 (`docs/bao-cao/2026-10-22-ca.md`) [REPO];
>   - vòng 2 (soát lại bản sửa F01–F03): **KHÔNG ĐẠT**, một ghi chú S2 (R2-01), sáu S3 (R2-02…R2-07), hai S4 (R2-08, R2-09) [kết luận của người soát, giao cho tác giả ở lệnh ca; phiên điều phối ghi vào `docs/bao-cao/` khi gộp];
>   - vòng 3 (soát lại bản sửa R2-01…R2-09; ở tệp này tác giả vòng 3 chỉ đổi số đột biến ở mục 5, điều kiện 4, và số dòng của D-OP-17 ở câu 8): **KHÔNG ĐẠT**, một ghi chú S2 (R3-01), ba S3 (R3-02, R3-03, R3-04), một S4 (R3-05) [kết luận của người soát, giao cho tác giả ở lệnh ca; phiên điều phối ghi vào `docs/bao-cao/` khi gộp];
>   - vòng 4 (bản chỉ sửa lỗi thuộc ngoại lệ của luật dừng (`docs/PROMPT-BUILD-EAIS.md` §9.6: an ninh, sai chiều, mất dữ liệu); ở tệp này vòng 4 chỉ đổi số đột biến và số lớp ca ở mục 5, điều kiện 4; phần nội dung chính do `HOP-DONG.md` và ADR-0012 sửa): **KHÔNG ĐẠT**, chỉ vì một chặn hẹp ở tài liệu: R4-01 (S3); thêm ba ghi chú S4 ghi nợ (R4-02, R4-03, R4-04) [kết luận của người soát, giao cho tác giả ở lệnh ca; phiên điều phối ghi vào `docs/bao-cao/` khi gộp];
>   - **bản này là vòng 5: chỉ vá R4-01 và R4-02, chưa soát lại.** Ở tệp này vòng 5 chỉ đổi số đột biến (36) và số lớp ca (84) ở mục 5, điều kiện 4; phần nội dung chính do `HOP-DONG.md` và ADR-0012 sửa. Mục 5, điều kiện 4 vẫn do vòng 1 (F03) hạ chữ và khớp M1 đã lên main.
>
> **Viết cho:**
> - chủ dự án: duyệt hoặc bác từng câu chấm;
> - phiên BUILD: làm theo `docs/viec/0a-thu-bao-cao-ngay/HOP-DONG.md` và `docs/adr/0012-phase0a-thu-bao-cao-ngay.md` khi mã được mở.
>
> **Ngày:** 22/10/2026. **Mốc đọc mã:** `2b80a38` (origin/main). Tra một dòng trích bằng `git show 2b80a38:<tệp>`. Bản nháp đầu đọc ở `7d72e33`; từ đó, D-OP-14 đã sửa bộ đọc số bằng chữ (HT-6, HT-7 hết). Làn 1 còn sửa "≈" (HT-1); khi gộp, các số [ĐO] về cờ "≈" có thể đổi: đo lại trước khi dựa vào. Phần M1 (làn 0, đã lên main) đọc thêm ở `41bc3cf` (`HOP-DONG.md` §1.4).
>
> **Nhãn bằng chứng:**
> - [ĐO]: người viết tự chạy lệnh, đo ở `7d72e33` rồi đo lại ở `2b80a38` (test thăm dò tạm, không commit; mô tả ở `HOP-DONG.md` §1.3.3);
> - [REPO]: tài liệu repo tự ghi, chưa chạy lại;
> - [CODE]: đọc thấy trong mã, `tệp:dòng` tại `2b80a38`;
> - [SUY]: suy luận của người viết;
> - [NGOÀI]: nguồn công khai ngoài repo, đọc ngày 22/10/2026;
> - "chưa đo": chưa chạy lệnh nào.
>
> **Cách chấm.** Phase 0a chưa có mã, nên dấu ở đây chỉ nói về GIẤY:
> - ✓ = tiêu chí của cổng (`docs/ROADMAP-DAY-DU.md` §4.1, cột "Qua cổng khi") đã được viết ra và có nguồn. Chưa có nghĩa là mã đã chứng minh; phần chứng minh bằng mã nằm ở cổng ra (`HOP-DONG.md` §11).
> - ◐ = viết được một phần, còn thiếu điều kiện đã nêu.
> - ✗ = không thể viết cho đủ nếu thiếu người hoặc dữ kiện bên ngoài repo.
> - **Chuẩn riêng của người viết, chặt hơn cột "Qua cổng khi" ở ba câu (1, 3, 9).** Tiêu chí của cột đã được viết ra và có nguồn, nhưng người viết vẫn không chấm ✓ vì còn thiếu điều kiện thật: câu 1 (người dùng là giả định, chưa có số nền); câu 3 (lỗi im lặng L1 còn mở, chưa đo so sánh hai đường vào); câu 9 (mới so một giải pháp một phần, bằng lập luận). Chủ dự án có thể chấm lại theo đúng chữ của cột.
> - **✓ ở đây là "viết ra, có nguồn" (giấy).** Khác ✓ của Fuel ở `docs/ROADMAP-DAY-DU.md` §7, là "đã đo". Không đặt hai bên cạnh nhau như một thước.
>
> **Dữ liệu:** mọi tên máy, dự án, người gửi, con số trong tài liệu này là giả lập (tenant `abc-infra` hư cấu, `docs/tenants/construction/01-business-map.md:3`). Mọi kết luận: **trên dữ liệu giả lập, chưa có người nghiệp vụ duyệt** (D-RM-03 = "Chưa có", `docs/CHO-QUYET-DINH.md` mục 0, dòng 22/10 11:2x).

---

## 0. Kết luận

- **9 câu, trên giấy:** ✓ 5 (câu 2, 4, 5, 6, 7) · ◐ 2 (câu 3, 9) · ✗ 2 (câu 1, 8).
- **Hai câu ✗ là hai câu Fuel cũng trượt** (`docs/ROADMAP-DAY-DU.md` §7, câu 1 và 8): không có người dùng thật, không có người ký độ chính xác. Gốc là D-RM-03.
- **Câu 3 là câu then chốt của Phase 0a.** Cả phase tồn tại vì dữ liệu của đội hiện trường là chữ tự do. Trên giấy đã có hai việc: (a) kê lỗi im lặng đã đo, (b) một nhánh form / bot dùng chung hợp đồng đầu ra (`HOP-DONG.md` §13). Chưa có phép đo so sánh hai nhánh.
- **Hệ quả** [SUY, theo `docs/ROADMAP-DAY-DU.md` §9]:
  - không chặn việc viết giấy, và không chặn mã khi H4 đã chạy;
  - mọi kết luận về Phase 0a ghi "trên dữ liệu giả lập, chưa có người nghiệp vụ duyệt";
  - không tuyên bố Phase 0a "đáng giá": phép đo giá trị chưa có số nền (câu 1).
- **Cổng gốc** (`docs/ROADMAP-DAY-DU.md` §4.2): xem §5. Điều kiện 1 (phase tiên quyết) **đã đạt**: H4 chạy xong ngày 22/10 (`docs/STATUS.md`). Điều kiện 4 (ca âm chứng minh cổng đỏ được) **chưa chứng minh**: mới viết trên giấy, chưa đột biến nào chạy.

---

## 1. Việc được chấm

> Một đội hiện trường báo cáo lượng dầu của từng máy trong ngày, bằng tin nhắn nhóm (Zalo) hoặc bằng tệp (Excel, PDF; ảnh chưa đọc được). EAIS biến **mỗi lần giao một đơn vị đầu vào** thành đúng một trong bốn kết cục có lý do (nhận · chờ xác nhận · cách ly · trùng); trùng chỉ để lại một dòng ở sổ lần giao, không tạo bản ghi (`HOP-DONG.md` §3.8). Bản ghi được nhận mang **cờ loại** (đã xảy ra · ước lượng · đính chính · kế hoạch). Con số đọc ra là tất định và không bị cộng đôi.

- **Nguồn của việc:**
  - `docs/ROADMAP-DAY-DU.md` §5b (khối M2 của MVP: "Thu báo cáo ngày đúng") và §6 (dòng Phase 0a);
  - `docs/ROADMAP-DAY-DU.md` §5, luồng L1;
  - `docs/PROMPT-BUILD-EAIS.md` §9.1 (M2), §9.2 (làn 3 làm giấy, làn 4 làm mã), §9.3 (hàng L1 và S2 của CỔNG TRƯỚC AI).
- **Chế độ B** (EAIS thu, chuẩn hoá, đối soát, lưu trữ): đội chỉ có Zalo, bảng tính, giấy, ảnh (`docs/tenants/construction/01-business-map.md:139-144`) [REPO].
- **Phạm vi dầu:** chỉ số `reported_fuel` ("Dầu đội báo", `tenants/construction/pack/metrics.yaml:9-18`) [CODE]. Bốn khe còn lại của báo cáo ngày (tiến độ, giờ máy, nhân công, thời tiết; luật BR-RPT-01 ở `01-business-map.md:404`) **ngoài phạm vi**; khối M2 chỉ đòi "một chuỗi việc dầu" (`docs/ROADMAP-DAY-DU.md` §5b).
- **Ví dụ giả lập:** tin "máy 03 sáng 2k" của một đội trưởng gửi lúc 07:30 ngày 19/10; cuối ngày có tin "chiều thêm 2k". Hai tin hợp lệ, cùng giá trị, **không** được gộp thành một. Số liệu là hạt giống giả lập, không phải của doanh nghiệp nào.
- **Hiện trạng** (đo ở `7d72e33` và đo lại ở `2b80a38`, chi tiết `HOP-DONG.md` §1.3) [ĐO]: trên 20 tin giả lập đi qua `ingestion.Ingest`, 17 thành quan sát và 3 bị cách ly có lý do; trong 17 quan sát đó có tin kế hoạch, tin đính chính và tin gửi lại đều được nhận như "đội báo, đã xảy ra, ngày gửi". Số bằng chữ đọc sai (HT-6, HT-7) đã sửa ở `2b80a38`.
- **Không thuộc Phase 0a:** nối trực tiếp với Zalo (connector, Phase 8); đọc ảnh (OCR, Phase 6); áp đính chính tự động bằng mô hình sự kiện (Phase 3); tạo phiếu khi có chênh lệch (Phase 1); mọi lời gọi model.

---

## 2. Bảng tóm tắt

| # | Câu | Trả lời ngắn | Qua? |
|---|---|---|---|
| 1 | Giá trị thật là gì? | Người dùng giả định (đội trưởng, trưởng dự án, kế toán), việc rõ, bốn phép đo đề xuất; không có người thật, không có số nền | ✗ |
| 2 | Việc có critical không? | Có: ước lượng, đính chính, kế hoạch, bản trùng thành số "chính xác" là lỗi mức S1 (CON-F-014); số này là một vế đối soát nhiên liệu | ✓ |
| 3 | Dữ liệu đã chuẩn chưa? | Chưa: chữ tự do, 10 lỗi im lặng đã đo hoặc đã đọc thấy (2 đã sửa ở `2b80a38`); đã cân nhắc nhánh form / bot, chưa đo so sánh | ◐ |
| 4 | Đã phân loại dữ liệu chưa? | Mỗi loại đầu vào và mỗi loại nội dung tin có một cách xử lý (hai bảng ở câu 4) | ✓ |
| 5 | Phần nào không cần AI? | Toàn bộ: 0 lời gọi model; phân loại tin bằng luật từ lexicon của pack | ✓ |
| 6 | Cần gom ngữ cảnh gì? | Danh mục thực thể, danh bạ người gửi + phạm vi, bản ghi đã có cùng (máy, ngày), lexicon, miền giá trị, múi giờ; kèm quyền đọc | ✓ |
| 7 | Thiếu hay mâu thuẫn thì hỏi lại thế nào? | Chờ xác nhận hoặc cách ly có mã lý do và người nhận; không đoán | ✓ |
| 8 | Ai cam kết độ chính xác? | Chưa có người nghiệp vụ thật (D-RM-03) | ✗ |
| 9 | Đồ có sẵn đã giải được chưa? | So một form có sẵn bằng nguồn công khai (nhập đúng dạng giải được, phần còn lại chưa đối chiếu); bot Zalo OA chưa so | ◐ |

---

## 3. Chi tiết từng câu

### Câu 1. Giá trị thật là gì?

- **Trả lời.**
  - Người dùng (giả định):
    - đội trưởng / giám sát (`site_supervisor`): người gửi báo cáo ngày qua chat (`01-business-map.md:308`);
    - trưởng dự án (`project_manager`): duyệt báo cáo ngày (`01-business-map.md:305`);
    - kế toán (`accountant`): dùng số đội báo làm vế của chỉ số dẫn xuất `reported_vs_issued` (`metrics.yaml:52`).
  - Việc: biến báo cáo ngày viết tự do thành bản ghi có cờ, không rơi, không cộng đôi. Hôm nay đội báo bằng nhóm chat, biểu mẫu giấy hoặc bảng tính (`01-business-map.md:139-144`).
  - Giá trị đo được (phép đo đề xuất, chưa có số):
    - (a) tỉ lệ (máy, ngày) có số đội báo được nạp đúng;
    - (b) số bản ghi "đội báo" sai mà không có cờ hay lý do, đích 0;
    - (c) thời gian từ lúc đội gửi tới lúc có số trong kho, so với quy trình hiện tại;
    - (d) số bản ghi cần người xác nhận mỗi ngày (chi phí người).
- **Bằng chứng / nguồn.**
  - KPI "Báo cáo ngày đúng hạn" ghi "chưa đo" (`01-business-map.md:496`) [REPO].
  - Tenant hư cấu, không có khách (`01-business-map.md:3`) [REPO].
- **Qua cổng? ✗.** Giống Fuel và Phase 1: chưa có người dùng thật, chưa có số nền.
- **Để qua:** có người nghiệp vụ (D-RM-03) và đo (a), (c), (d) trên quy trình hiện tại của họ.

### Câu 2. Việc có critical không?

- **Trả lời.** Có. Mức hậu quả khi sai:
  - **S1:** ước lượng ghi như số đo rồi vào tổng "chính xác" (CON-F-014); tin kế hoạch hoặc tin đính chính thành số mới; tin gửi lại bị cộng đôi; số bằng chữ đọc sai bậc độ lớn;
  - **S2:** tin bị bỏ qua mà không ai biết (thiếu số), số gán nhầm ngày;
  - số đội báo là một vế đối soát với kho xuất và cảm biến: sai làm bỏ sót thất thoát hoặc báo thất thoát giả (R-08, `01-business-map.md:479`). Phase 0a không chuyển tiền và không phát lệnh; hậu quả về tiền là gián tiếp [SUY].
- **Bằng chứng / nguồn.**
  - S1 = "sai số liệu / hành động tới tay người dùng…" (`docs/CONVENTIONS.md` §4.4) [REPO].
  - CON-F-014 mức S1: "`khoảng 4k` được ghi như số đo, mất cờ ước lượng, rồi vào tổng 'chính xác'" (`docs/tenants/construction/07-failure-catalog.md:60`) [REPO].
  - Cùng họ đã đo: K1–K5 (`docs/bao-cao/2026-10-21-phan-hoi-roadmap-k1-k5.md` §3) [REPO]; tái đo ở `2b80a38` (`HOP-DONG.md` §1.3) [ĐO].
- **Qua cổng? ✓** (mức hậu quả đã ghi). Giới hạn: thiệt hại bằng tiền chưa có người nghiệp vụ xác nhận, chưa đo.

### Câu 3. Dữ liệu đã chuẩn chưa?

- **Trả lời.** Chưa. Hai phần:
  - **(a) Đánh giá dữ liệu hiện có.** Đầu vào của đội hiện trường là chữ tự do (tin nhóm, dòng PDF) và bảng. Mười lỗi đã đo hoặc đọc thấy, mang mã HT-1…HT-10 (chi tiết `HOP-DONG.md` §1.3):
    1. (HT-1) "≈" mất cờ ước lượng [ĐO];
    2. (HT-2) tin đính chính thành số mới của ngày gửi [ĐO];
    3. (HT-3) tin kế hoạch thành quan sát của ngày gửi [ĐO];
    4. (HT-4) ngày viết trong chữ ("hôm qua", "ngày 25/10") bị bỏ, quan sát gắn ngày gửi [ĐO];
    5. (HT-5) tin gửi lại thành hai quan sát [ĐO]; kho đọc cộng mọi fact cùng nguồn, cùng ngày (`data/sourceoftruth/query.go:199-221`) [CODE], và đo ở mốc cũ cho tổng gấp đôi (K4) [REPO];
    6. (HT-6) số bằng chữ đọc sai ("một nghìn hai" → 1.002 L, "ba nghìn tám" → 3.008 L ở `7d72e33`): **đã sửa** ở `2b80a38` (D-OP-14: 1.200 L, 3.800 L) [ĐO];
    7. (HT-7) số bằng chữ không dấu không tất định ("hai muoi": 14 lần ra 10 L, 6 lần ra 20 L trong 20 tiến trình ở `7d72e33`): **đã sửa** ở `2b80a38` (20/20 lần ra 20 L) [ĐO];
    8. (HT-8) khoá bản ghi trùng khi tin thiếu hoặc trùng `record_id` [ĐO];
    9. (HT-9) dòng PDF không khớp mẫu bị bỏ im lặng (`data/ingestion/readers.go:165-177`; `docs/ROADMAP.md` §0.4 mục 12) [CODE, REPO];
    10. (HT-10) cách ly chỉ nằm trong RAM, không người nhận (`docs/01-architecture.md` §4, hàng "cách ly") [REPO]; M1 đã lưu bền khoá, nguồn, lý do, chi tiết của cách ly theo lần nạp, còn thiếu người nhận, trạng thái, xuất (`HOP-DONG.md` §1.3.2) [CODE].
  - **(b) Chuẩn hoá đầu vào trước khi đọc chữ tự do.** Đã cân nhắc trên giấy: nhánh form / bot dùng chung hợp đồng đầu ra, có bản chiếu cặp để đo hai đường trên cùng ngữ nghĩa (`HOP-DONG.md` §13). Chưa đo.
- **Bằng chứng / nguồn.** Mười lỗi ở trên; ca G1 hiện ghim 3 lệch cờ ước lượng trên 2.739 bản ghi seed (`tenants/construction/win/g1-approx-known.txt`, `docs/STATUS.md:62`) [REPO].
- **Qua cổng? ◐.** Đã đánh giá và đã cân nhắc chuẩn hoá đầu vào. Chưa có cổng kiểm mức sẵn sàng của dữ liệu; lỗi im lặng còn mở; chưa đo so sánh hai nhánh.
- **Để qua:** (a) đóng các lỗi còn mở (HT-1…HT-5, HT-8…HT-10) ở mã Phase 0a và ở bộ chuẩn hoá ("≈", làn 1); (b) đo hai nhánh trên bản chiếu cặp của cùng bộ ca độc lập.

### Câu 4. Đã phân loại dữ liệu chưa?

**Theo loại đầu vào:**

| Loại đầu vào | Loại nguồn trong pack | Cách xử lý |
|---|---|---|
| tin nhắn nhóm (chữ tự do) | `chat_message`, hạng 2 (`metrics.yaml:18`, `sources.yaml:46-52`) | phong bì tin → luật chuẩn hoá, thực thể, loại tin → bản ghi có cờ, hoặc chờ xác nhận, hoặc cách ly |
| báo cáo ngày PDF (văn bản) | `site_report_form`, hạng 1 (`metrics.yaml:17`, `sources.yaml:55-60`) | mỗi dòng `n. chữ` là một đơn vị; ngày lấy từ tên tệp (`ingest.go:240-243`); dòng lạ → cách ly `unrecognized_line` |
| bảng (xlsx / csv / json) do đội gửi | chưa có loại riêng: `excel_summary` là bảng của kho, chỉ số `issued_fuel` (`sources.yaml:18-23`) | thêm một loại nguồn có cấu trúc cho báo cáo của đội, chỉ số `reported_fuel` (việc của pack khi mở mã) |
| ảnh, tệp định dạng lạ | không có | cách ly `unsupported_media`, giữ tham chiếu; đọc ảnh là OCR, thuộc Phase 6 |
| bản ghi từ form / bot | nhánh B (`HOP-DONG.md` §13) | cùng hợp đồng đầu ra, đường nhập có cấu trúc |

**Theo loại nội dung của tin** (bốn cờ loại, `HOP-DONG.md` §3.3):

| Loại nội dung | Cách xử lý |
|---|---|
| đã xảy ra (báo lượng đã cấp / đã dùng) | nhận, tính vào tổng |
| ước lượng ("khoảng", "tầm", "≈", "~") | nhận kèm cờ `approx`; tổng mang cờ |
| đính chính ("báo nhầm", "sửa lại") | luôn chờ người xác nhận; không tự thành số mới |
| kế hoạch ("mai cần", "dự kiến") | lưu, không tính vào tổng "đã xảy ra" |

- **Bằng chứng / nguồn:** cột "Loại nguồn" và "Cách xử lý" của hai bảng; `tenants/construction/pack/sources.yaml` có 7 loại nguồn [CODE].
- **Qua cổng? ✓.**

### Câu 5. Phần nào không cần AI?

- **Trả lời.** Toàn bộ Phase 0a là tất định, **0 lời gọi model**: đọc tệp, chuẩn hoá, nhận dạng thực thể, phân loại tin, kiểm miền, chống trùng, kiểm quyền, ghi, xuất báo cáo.
- **Phần duy nhất có thể cần model:** phân loại loại tin (đính chính, kế hoạch) khi luật từ lexicon trượt trên ca đo được; đọc ảnh. Cả hai là cổng đóng: model chỉ mở khi luật trượt có số đo và đầu ra có validator (`CLAUDE.md` §19), và không lời gọi model trả tiền nào trước CỔNG TRƯỚC AI (D-AI-00; `docs/PROMPT-BUILD-EAIS.md` §9.3).
- **Bằng chứng / nguồn.** Không gói nào dưới `data/` hay `platform/pgstore/` import `intelligence/models` [ĐO `grep -rn "intelligence/models" data platform/pgstore --include=*.go` bỏ tệp test → rỗng].
- **Qua cổng? ✓.** Kiểm bằng ca đếm lời gọi model = 0 (`HOP-DONG.md` §10, lớp ca LC68).

### Câu 6. Cần gom ngữ cảnh gì?

| Ngữ cảnh | Nguồn | Quyền đọc |
|---|---|---|
| danh mục thực thể (bí danh, mã theo nguồn, cha–con) | `entities.json` của tenant (`ingest.go:44-98`) | cùng tenant (RLS) |
| danh bạ người gửi (mã người gửi → vai, phạm vi thực thể) | **mới**, dữ liệu tenant (`HOP-DONG.md` §4.3) | quản trị tenant |
| bản ghi đã có cho cùng (máy, ngày): để phát hiện trùng và tìm bản gốc của đính chính | kho bản ghi báo cáo (**mới**) | cùng tenant; người xem phải trong phạm vi thực thể |
| từ khoá: ước lượng, đính chính, kế hoạch; đơn vị | `lexicon.yaml` (`approx_markers`: dòng 26), bổ sung ở pack | không áp dụng |
| miền giá trị, đơn vị chuẩn của `reported_fuel` | `metrics.yaml:9-15` | không áp dụng |
| thời điểm gửi và múi giờ tenant | `sent_at`; `pack.yaml:6` (`Asia/Ho_Chi_Minh`) | không áp dụng |
| phiên bản pack và mã | `PackVersion` (`core/tenant/pack.go`) | không áp dụng |

- **Không gom:** lịch sử hội thoại ngoài đơn vị đang xử lý, đầu ra của model (không có model).
- **Qua cổng? ✓.**

### Câu 7. Thiếu hay mâu thuẫn thì hỏi lại thế nào?

- **Trả lời.** Không đoán. Mỗi đơn vị có một mã lý do trong danh sách đóng (`HOP-DONG.md` §3.4):
  - bí danh khớp nhiều máy → **chờ xác nhận** `ambiguous_entity`, kèm ứng viên nằm trong phạm vi người gửi;
  - thiếu lượng, thiếu máy, ngoài miền, khác chiều đo → **cách ly** (`missing_value`, `unknown_entity`, `out_of_range`, `unit_dimension`);
  - số bằng chữ mà giá trị còn do cách nói tắt quyết (dãy kết thúc bằng "linh" hoặc "lẻ", nghĩa xấp xỉ) → **chờ xác nhận** `number_reading_uncertain`; dạng nói tắt thường ("một nghìn hai") đọc theo quy ước D-OP-14 và ghi chú vào bản ghi;
  - tin đính chính → luôn **chờ xác nhận** `correction_unconfirmed`, kèm danh sách bản gốc ứng viên;
  - nghi gửi lại → **chờ xác nhận** `possible_duplicate`;
  - mâu thuẫn giữa tin chat và PDF cho cùng (máy, ngày): đối soát theo thẩm quyền lúc hỏi như hiện nay (`data/sourceoftruth/query.go`), không làm ở lúc nạp.
- **Giới hạn:**
  - chưa có kênh trả lời ngược cho người gửi trên Zalo (connector, Phase 8): "hỏi lại" ở Phase 0a là một mục trong hàng chờ có câu hỏi và người nhận, không phải tin nhắn gửi đi;
  - chưa có mã cho hàng chờ: `docs/01-architecture.md` §5, đường I3, ghi "chưa hỏi lại người gửi, chưa lưu `clarifications`" [REPO].
- **Bằng chứng / nguồn:** `HOP-DONG.md` §3.4, §6; từ vựng lý do hiện có ở `data/ingestion/sources.go:18-26` [CODE].
- **Qua cổng? ✓** (luật đã viết, chưa có mã nên chưa kiểm).

### Câu 8. Ai cam kết độ chính xác?

- **Trả lời.** Chưa có ai. D-RM-03 = "Chưa có người nghiệp vụ thật" (`docs/CHO-QUYET-DINH.md` mục 0, dòng 22/10 11:2x) [REPO].
- **Phần phải có người nghiệp vụ ký:**
  1. ngưỡng độ chính xác và độ phủ của cổng ra (`HOP-DONG.md` §11, dòng GA6): hai số trong tài liệu là **ĐỀ XUẤT** của người viết, không suy từ dữ liệu;
  2. nghĩa của hậu tố "k" ("2k" là số đo hay ước lượng): D-OP-17 (`docs/CHO-QUYET-DINH.md`, mục 3, dòng D-OP-17: dòng 134 tại `269a539`), cột trả lời trống;
  3. khi nào một tin đính chính được áp, và ai được áp;
  4. định nghĩa "trùng": cửa sổ thời gian, có tính người gửi khác nhau không;
  5. ai nhận bản ghi chờ xác nhận và cách ly;
  6. lịch ca và giờ cắt ngày của ca đêm (BR-RPT-02, `01-business-map.md:405`, ghi "ca đêm tính vào ngày bắt đầu ca"; pack chưa có lịch ca);
  7. vai nào được gửi báo cáo cho máy nào (vai `site_supervisor` có ở `01-business-map.md:308` nhưng chưa có ở `policy.yaml:5-10`);
  8. nguồn nào thắng khi tin chat và PDF cùng báo một (máy, ngày): hiện PDF hạng 1, chat hạng 2 (`metrics.yaml:16-18`), chưa người duyệt.
- **Phần kỹ thuật không cần người nghiệp vụ để kiểm:** bảo toàn đơn vị đầu vào, cờ loại đúng, không cộng đôi, đọc số tất định. Đó là cam kết kỹ thuật kiểm bằng ca (`HOP-DONG.md` §11), **không** phải cam kết độ chính xác nghiệp vụ.
- **Qua cổng? ✗.** Mọi kết luận ghi "trên dữ liệu giả lập, chưa có người nghiệp vụ duyệt". Câu gửi chủ dự án: `HOP-DONG.md` §15.

### Câu 9. Đồ có sẵn đã giải được chưa?

- **Trả lời.** So một giải pháp có sẵn (form) bằng lập luận có nguồn công khai (§4). Kết luận: form có sẵn giải được phần "nhập đúng dạng" ngay tại nguồn; các phần còn lại của hợp đồng (chống trùng khi gửi lại, tham chiếu bản gốc của đính chính, quyền theo máy, cách ly có người nhận) **chưa đối chiếu** với tài liệu của form. Vì vậy Phase 0a giữ **một hợp đồng đầu ra chung** cho đường chat và đường form / bot; chọn đường nào là việc của chủ dự án theo số đo (`HOP-DONG.md` §13).
- **Bằng chứng / nguồn:** §4: hai trang trợ giúp của Google [NGOÀI] và bảng so [SUY]. Phép thử trên form thật: **chưa có**.
- **Qua cổng? ◐.** Đã so một giải pháp một phần, bằng lập luận. Chưa so bot Zalo OA (thân trang tài liệu không đọc được), chưa so Microsoft Forms hay form khác, chưa thử thật.

---

## 4. Câu 9: so với đồ có sẵn

**Giải pháp so:** form có sẵn, ví dụ Google Forms. **Chưa so:** bot Zalo OA / mini app (xem cuối mục).

| Tiêu chí (lấy từ cổng ra của Phase 0a, `HOP-DONG.md` §11) | Form có sẵn (Google Forms) | Phase 0a tự làm (chế độ B) |
|---|---|---|
| Nhập đúng dạng ngay tại nguồn (trường bắt buộc, kiểm số, chọn từ danh sách) | **Giải được phần lớn.** Trang trợ giúp ghi: bật "Required" để người trả lời không bỏ trống; "Response validation" cho câu trả lời ngắn gồm Number, Text, Length, Regular expression; câu hỏi nhiều ô chọn có Select at least / at most / exactly [NGOÀI]. Loại câu hỏi thả xuống (drop-down) có tồn tại (được nhắc ở phần xáo trộn đáp án) [NGOÀI] | phải đọc chữ tự do bằng luật và bộ chuẩn hoá (khó hơn) |
| Cờ loại (đã xảy ra / ước lượng / đính chính / kế hoạch) | Thêm một câu hỏi chọn loại được [SUY, chưa thử]. "Đính chính cho bản ghi nào" cần tham chiếu tới bản đã gửi: hai trang đã đọc không nói về điều này → **chưa so** | luật từ lexicon của pack; chờ người xác nhận khi không chắc |
| Chống trùng khi gửi lại | Hai trang đã đọc không mô tả khoá chống trùng cho người gửi → **chưa so** | ba mức (`HOP-DONG.md` §5) |
| Quyền theo máy ("ai báo cho máy nào") | Hai trang đã đọc không nói → **chưa so** | danh bạ người gửi + phạm vi (`HOP-DONG.md` §4) |
| Cách ly có lý do, có người nhận, xuất báo cáo | Form chặn lúc nhập thay vì cách ly sau; trong hai trang đã đọc không thấy hàng chờ người duyệt có lý do [chưa so tính năng khác] | kho bản ghi + báo cáo cách ly (`HOP-DONG.md` §6) |
| Giữ kênh hiện tại của đội (Zalo) | Không: là một kênh khác; đội phải đổi thói quen. Chi phí đổi: chưa đo [SUY] | Có: đọc tin nhóm đã xuất; form / bot là nhánh thay thế |
| Đưa dữ liệu vào EAIS | cần xuất tệp hoặc connector (Phase 8); nhánh có cấu trúc của `HOP-DONG.md` §13 nhận được [SUY] | trực tiếp |

**Nguồn [NGOÀI], đọc ngày 22/10/2026 qua công cụ tải trang (bản tóm tắt tự động của công cụ, người viết chưa đối chiếu từng chữ với bản gốc):**
- Google Docs Editors Help, "How to set rules for your form": `https://support.google.com/docs/answer/3378864?hl=en`. Nêu "Response validation" trong menu "More" của câu hỏi; loại luật phụ thuộc loại câu hỏi.
- Google Docs Editors Help (trang `answer/2839737`): `https://support.google.com/docs/answer/2839737?hl=en`. Nêu "To prevent people from not answering, turn on Required"; nhắc câu hỏi drop-down khi nói xáo trộn đáp án.
- Hai trang trên **không** nói về: giới hạn một phản hồi mỗi người, nơi lưu phản hồi, tải tệp lên (công cụ báo "không có trên trang này"). Vì vậy các ô "chưa so" ở bảng là chưa so, không phải đã so và thấy thiếu.

**Chưa so: bot Zalo OA / mini app.** Công cụ đọc trang của người viết chỉ lấy được tiêu đề (không lấy được thân trang) của hai trang tài liệu `developers.zalo.me`: "Sự kiện người dùng gửi tin nhắn" (`https://developers.zalo.me/docs/official-account/webhook/tin-nhan/su-kien-nguoi-dung-gui-tin-nhan`) và "Sự kiện tin nhắn được gửi tới nhóm" (`https://developers.zalo.me/docs/official-account/webhook/nhom-chat-gmf/messgae_webhook`). Chưa biết điều kiện, giới hạn, phí, và OA có đọc được nhóm Zalo của đội hay không. Một chủ đề cộng đồng cùng nơi tên "Nhận 2 webhooks khi nhắn 1 message" (`https://developers.zalo.me/community/detail/f8c7b4248861613f3870`) gợi ý webhook có thể giao trùng; chưa đọc nội dung nên **không dùng làm bằng chứng**. Thiết kế chống trùng của Phase 0a (`HOP-DONG.md` §5) không dựa vào gợi ý này.

**Kết luận.**
- Với phần "nhập đúng dạng", form có sẵn là lựa chọn rẻ hơn tự đọc chữ tự do [SUY]; đó là lý do `HOP-DONG.md` §13 đặt nhánh form / bot **song song** chứ không phải phương án lùi.
- Với các phần còn lại, chưa đủ nguồn để nói form có sẵn thiếu hay đủ. Dù chọn đường nào, biên kiểm soát (chống trùng, cách ly có lý do, quyền theo máy) phải có ở phía EAIS vì hợp đồng đầu ra là một [SUY].
- **Để ◐ thành ✓:** đọc được thân trang tài liệu Zalo OA; so thêm một bot / form khác theo cùng bảng tiêu chí; thử một form thật với bản chiếu cặp.

---

## 5. Cổng gốc (`docs/ROADMAP-DAY-DU.md` §4.2)

| # | Điều kiện | Trạng thái | Nguồn |
|---|---|---|---|
| 1 | Phase tiên quyết đã đạt | **Đạt.** Mã 0a mở sau H4 (`docs/PROMPT-BUILD-EAIS.md` §9.2, làn 4); H4 đã chạy MỘT lần ngày 22/10 (`docs/STATUS.md`). Điều kiện kỹ thuật thêm: M1 (làn 0) đã lên main ở `41bc3cf` với `POST /v1/ingest` (principal từ token, nạp lại thư mục đã cấu hình); đường nạp từng đơn vị là việc của M2 (`HOP-DONG.md` §1.4, §14.1). D-OP-14 (bộ đọc số, làn 1) đã vào main (`7e1d8ca`, `017495c`) | `docs/STATUS.md`; `docs/CHO-QUYET-DINH.md` mục 0 (D-OP-14, D-RM-04) |
| 2 | Có lỗi đo được hoặc ca nghiệp vụ đòi năng lực mới | **Có.** K1–K5 và `docs/ROADMAP.md` §0.4 mục 1, 3, 12, 15, 16, 17; CON-F-014 (S1); thêm các lỗi người viết đo ở `2b80a38` (`HOP-DONG.md` §1.3) | mục 3 câu 2, câu 3 |
| 3 | Trả lời được "ai sở hữu?" | Trả lời đủ 10 mặt ở `HOP-DONG.md` §8 (giấy) | `docs/05-master-roadmap.md:30` |
| 4 | Cổng ra nhị phân, có ca âm chứng minh cổng đỏ được | **Chưa chứng minh.** Trên giấy: `HOP-DONG.md` §11.1 có 11 dòng GA, mỗi dòng có ít nhất một đột biến bắt buộc đỏ (36 đột biến ở §11.3, mỗi cặp đột biến và dòng ghi lớp ca của chính dòng đó làm nó đỏ), hai bảng §11.1 và §11.3 đối xứng, mọi lớp ca âm ở §10 (84 lớp) thuộc một dòng GA. Chưa bộ ca độc lập nào được soạn và chưa đột biến nào chạy, nên "cổng đỏ được" mới là điều phải chứng minh ở bước đột biến (`HOP-DONG.md` §14.2), chưa phải điều đã chứng minh | `docs/ROADMAP-DAY-DU.md` §6 |

---

## 6. Điều kiện đổi ✗ / ◐ thành ✓ (việc của người, không phải của mã)

| Câu | Cần | Ai |
|---|---|---|
| 1 | có người nghiệp vụ; đo (a), (c), (d) trên quy trình hiện tại của họ | chủ dự án (D-RM-03) |
| 3 | đóng lỗi im lặng ở mã Phase 0a và ở bộ chuẩn hoá; đo hai nhánh trên bản chiếu cặp | BUILD (làn 4, làn 1) |
| 8 | người nghiệp vụ ký tám mục ở câu 8 | chủ dự án (D-RM-03) |
| 9 | đọc được tài liệu Zalo OA; so thêm một bot / form; thử thật | BUILD hoặc người soát |

Chừng nào câu 1 và 8 còn ✗: Phase 0a chỉ được báo là "cổng ra kỹ thuật đạt trên dữ liệu giả lập, chưa có người nghiệp vụ duyệt", không báo là "đáng giá".
