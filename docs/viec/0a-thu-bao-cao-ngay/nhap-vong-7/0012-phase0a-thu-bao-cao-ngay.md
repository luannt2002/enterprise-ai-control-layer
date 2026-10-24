> **BẢN NHÁP VÒNG 7 — KHÔNG CÓ HIỆU LỰC.** Giấy Phase 0a KHOÁ ở vòng 6 theo luật chủ dự án 23/10 (giấy tối đa 3 vòng soát; commit `146c2b0`).
> Bản chính thức: `docs/adr/0012-phase0a-thu-bao-cao-ngay.md`. Bản này chép nguyên từ nhánh `m2/phase0a-vong7-bo` (`e747ed6`) khi gộp mọi nhánh về main (D-RAG-05, 24/10) để không mất chữ;
> mở lại vòng 7 là quyết định của chủ dự án.

# ADR-0012 — Phase 0a: thu báo cáo ngày của đội hiện trường (kho bản ghi báo cáo, cờ loại, chống trùng; chế độ B)

- **Ngày:** 22/10/2026.
- **Trạng thái:** **ĐỀ XUẤT, chưa áp mã.**
  - Mã Phase 0a (khối M2 của MVP) chỉ mở SAU khi chạy xong H4 (`docs/PROMPT-BUILD-EAIS.md` §9.2, làn 4: "0a (M2): sau khi H4 chạy xong"). H4 đã khoá (`15f04f9`) và đã chạy MỘT lần ngày 22/10 (`docs/STATUS.md`, dòng "H4, 22/10 14:59"): điều kiện này đã đạt.
  - Soát (mã ghi chú F01–F03, R2-01…R2-09, R3-01…R3-05, R4-01…R4-04, R5-01…R5-03 và R6-01…R6-02 định nghĩa ở mục "Ghi chú soát đã xử lý", cuối tệp):
    - vòng 1 (người soát ≠ người viết): **ĐẠT CÓ GHI CHÚ**, ba ghi chú S3: F01, F02, F03 (`docs/bao-cao/2026-10-22-ca.md`) [REPO];
    - vòng 2 (soát lại bản sửa F01–F03): **KHÔNG ĐẠT**, một ghi chú S2 (R2-01), sáu S3 (R2-02…R2-07), hai S4 (R2-08, R2-09) [kết luận của người soát, giao cho tác giả ở lệnh ca; phiên điều phối ghi vào `docs/bao-cao/` khi gộp];
    - vòng 3 (soát lại bản sửa R2-01…R2-09; tác giả vòng 3 sửa bằng quyết định D5 và D10, mục I, J, K ở "Phương án đã cân nhắc" và nợ 17 đến 22): **KHÔNG ĐẠT**, một ghi chú S2 (R3-01), ba S3 (R3-02, R3-03, R3-04), một S4 (R3-05) [kết luận của người soát, giao cho tác giả ở lệnh ca; phiên điều phối ghi vào `docs/bao-cao/` khi gộp];
    - vòng 4 (bản chỉ sửa lỗi thuộc ngoại lệ của luật dừng (`docs/PROMPT-BUILD-EAIS.md` §9.6: an ninh, sai chiều, mất dữ liệu, gồm cộng đôi im lặng và rơi khỏi tổng im lặng)): **KHÔNG ĐẠT**, chỉ vì một chặn hẹp ở tài liệu: R4-01 (S3, tức nợ 25); thêm ba ghi chú S4 ghi nợ (R4-02, R4-03, R4-04) [kết luận của người soát, giao cho tác giả ở lệnh ca; phiên điều phối ghi vào `docs/bao-cao/` khi gộp]. Tác giả vòng 4 đã sửa: R3-01 (quyết định D5, mục J, nợ 18), R3-04 (quyết định D5, mục I, nợ 17). Câu mở mới: Q-0A-16 (R3-03; nợ 20). Ghi nợ, không sửa luật: R3-02 (nợ 23), R3-05 (a)–(d) (nợ 24); nợ 25 (bản được giữ của `MARK_DUPLICATE` bị thu hồi sau quyết định) do tác giả vòng 4 thấy khi sửa R3-04. Điều cố ý chưa sửa là nợ ở "Hệ quả" và câu mở Q-0A-13…Q-0A-16 ở `HOP-DONG.md` §15;
    - vòng 5 (bản chỉ vá R4-01 và R4-02): **KHÔNG ĐẠT**, chỉ vì một chặn hẹp ở tài liệu: R5-01 (S3, phần còn lại của R4-01 sau bản vá vòng 5, tức nợ 25); thêm hai ghi chú S4 (R5-02, R5-03) [kết luận của người soát, giao cho tác giả ở lệnh ca; phiên điều phối ghi vào `docs/bao-cao/` khi gộp]. Tác giả vòng 5 đã sửa: R4-01 (nợ 25 thành đã vá; quyết định D5, mục J: bản `REJECTED` bởi `MARK_DUPLICATE` quay lại `HELD` khi bản được giữ rời nhóm bằng thu hồi hoặc sửa tin) và R4-02 (lệch chữ ở `HOP-DONG.md` §7.1, §7.5, §9, LC67, GA9 và ở mục G của ADR này: ngoại lệ (e) của §7.4). Ghi nợ, không sửa luật: R4-03 và R4-04 (nợ 24). Điều cố ý chưa sửa là nợ ở "Hệ quả" và câu mở Q-0A-13…Q-0A-16 ở `HOP-DONG.md` §15;
    - vòng 6 (bản chỉ vá R5-01, thêm một câu thống nhất ở `HOP-DONG.md` §3.8 (R5-03) và đo lại số của mô phỏng (R5-02)): **KHÔNG ĐẠT**, chỉ vì một chặn hẹp ở tài liệu: R6-01 (S3, cùng họ với R4-01 và R5-01, tức nợ 25: nguyên nhân `APPLY` đính chính không có trong danh sách của (c)); thêm một ghi chú S4 (R6-02) [kết luận của người soát, giao cho tác giả ở lệnh ca; phiên điều phối ghi vào `docs/bao-cao/` khi gộp]. Tác giả vòng 6 đã sửa: R5-01 (nợ 25 vá cho bản được giữ từ bất kỳ trạng thái, quyết định D5, mục J), R5-03 (`HOP-DONG.md` §3.8) và R5-02 (số đo lại ở mục J và nợ 25; giả định của mô phỏng ở `HOP-DONG.md` U8). Số lớp ca (84) và số đột biến (36) không đổi;
    - **bản này là vòng 7: chỉ sửa gốc của họ lỗi "bản bị bác theo bản được giữ rơi khỏi tổng im lặng" (R6-01, R6-02), chưa soát lại**. Sửa: quyết định D5 và mục J (điều kiện quay lại `HELD` đổi từ danh sách nguyên nhân sang sự kiện: bản được giữ thôi được tính vì bất kỳ lý do gì; thêm hàng chữ vòng 6), nợ 25 (vá theo sự kiện), nợ 18, dòng "Chứng minh" 7, hai ghi chú R6-01, R6-02; ở `HOP-DONG.md`: §5.6 mục 4 (c) và các chỗ nhắc nó, GA9 (phép kiểm tính chất), LC84 (ca con (viii), (ix)), DB36, U8, U15. Số lớp ca (84) và số đột biến (36) không đổi. Điều cố ý chưa sửa vẫn là nợ ở "Hệ quả" và câu mở Q-0A-13…Q-0A-16 ở `HOP-DONG.md` §15.
  - Chưa có người nghiệp vụ duyệt (D-RM-03 = "chưa có").
- **Số ADR:** 0012. Số 0011 thuộc Phase 1 (`docs/adr/0011-phase1-hanh-dong-co-kiem-soat.md`). Migration của Phase 0a lấy **số kế tiếp sau migration cuối của main lúc mở mã**. Tại `269a539` (mã không đổi từ `f28ce06`) migration cuối là `0003_kho_vet_ben.sql` (M1, đã lên main), nên nếu mã mở ngay thì số kế tiếp là 0004. ADR-0011 cũng ghi `<NNNN>` là số kế tiếp sau migration cuối của main lúc mở mã của nó, và làn 0 đã dùng `0003`: hai phase không được cùng một số, bên mở mã sau lấy số kế tiếp tại lúc đó (xem "Nơi đặt mã").
- **Liên quan:**
  - `docs/viec/0a-thu-bao-cao-ngay/HOP-DONG.md` (hợp đồng việc: chi tiết của mọi quyết định dưới đây);
  - `docs/viec/0a-thu-bao-cao-ngay/CONG-QUANIT.md` (9 câu cổng QuânIT, trên giấy: 5 ✓, 2 ◐, 2 ✗);
  - `docs/ROADMAP-DAY-DU.md` §4, §5 (luồng L1), §5b (khối M2), §6 (dòng Phase 0a);
  - `docs/PROMPT-BUILD-EAIS.md` §9.1–§9.3 (hàng L1 và S2), §9.9;
  - `docs/01-architecture.md` §3, §4 (luồng L1), §5 (đường I2, I3);
  - `CLAUDE.md` §10 (P2, P4, P5), §16, §17, §19, §20, §25, §34;
  - ADR-0006 (bộ giữ lại độc lập), ADR-0009 và ADR-0010 (cổng phán quyết, S4: không đổi ở đây), ADR-0011 (Phase 1: cùng khu vực `platform/pgstore/` và `db/migrations/`);
  - `docs/bao-cao/2026-10-21-phan-hoi-roadmap-k1-k5.md` (lỗi im lặng L1 đã đo); D-OP-14 (bộ đọc số, đã vào main), D-OP-17 ("2k" có là ước lượng, còn mở).
- **Nhãn bằng chứng:** [ĐO] người viết tự chạy lệnh · [REPO] tài liệu repo tự ghi · [CODE] đọc thấy trong mã (`tệp:dòng` tại `2b80a38`; tra bằng `git show 2b80a38:<tệp>`) · [SUY] suy luận · [NGOÀI] nguồn công khai (`CONG-QUANIT.md` §4) · "chưa đo".

## Bối cảnh (đã đo hoặc đọc thấy)

1. **Lỗi im lặng của luồng L1 đã đo** (`docs/bao-cao/2026-10-21-phan-hoi-roadmap-k1-k5.md` §3, đo ở `7cad584`) [REPO]:
   - K1: "≈" mất cờ ước lượng; "máy ủi 05 ≈ 3k" ra 3.000 L không ghi chú;
   - K2: tin đính chính "hôm qua báo nhầm, máy 03 là 3k" thành 3.000 L của ngày **gửi**; tổng tuần từ 4.000 L thành 7.000 L;
   - K3: tin kế hoạch "mai cần 4k máy 03" thành quan sát của ngày gửi; câu hỏi liên quan chuyển sang `SYSTEM_FAILURE`;
   - K4: tin trùng (chỉ đổi mã tin) làm lu 02 từ 1.200 L thành 2.400 L, không cảnh báo;
   - K5 và mục 4: số bằng chữ đọc sai ("một nghìn hai" → 1.002 L) và không tất định ("hai muoi lit"): **đã sửa** bởi D-OP-14 (kết quả đo lại bên dưới).
2. **Đo lại ở `2b80a38`** (`HOP-DONG.md` §1.3.3): 20 tin giả lập qua `ingestion.Ingest` cho 17 quan sát và 3 cách ly, không tin nào mất [ĐO]. Còn mở: HT-1 ("≈" ra `approx=false`), HT-2 (đính chính thành 3.000 L ngày 20/10), HT-3 (kế hoạch thành 4.000 L ngày 19/10), HT-4 (ngày viết trong chữ bị bỏ), HT-5 (tin gửi lại thành hai quan sát), HT-8 (hai tin chung khoá bản ghi `messages.jsonl#` khi thiếu `record_id`); hết: HT-6 (1.200 L, 3.800 L), HT-7 (20 L trong 20/20 tiến trình) [ĐO].
3. **Gốc cấu trúc** [CODE]:
   - `Observation` không mang giờ gửi, người gửi, loại tin (`data/ingestion/ingest.go:19-31`); ngày bị cắt về 00:00 ngày gửi (`:244-250`); span ngày của chữ được tính rồi bỏ (`:257`);
   - cách ly chỉ là một mảng trong `Result`, không lưu, không người nhận (`ingest.go:33-42`);
   - dòng PDF không khớp mẫu bị bỏ im lặng (`readers.go:165-177`);
   - `Result.Facts()` chuyển **mọi** quan sát thành fact (`bridge.go:36-45`), và `aggregate` cộng mọi fact cùng loại nguồn trong (thực thể, ngày) (`data/sourceoftruth/query.go:199-221`), đúng thiết kế (`TestCungNguonCungNgayLaCongDon`);
   - `Load` thay trọn dữ liệu của tenant (`platform/pgstore/load.go:21-48`).
4. **Dùng lại được** [CODE]: RLS ENABLE + FORCE (`db/migrations/0001_win.sql:69-70`); phạm vi thực thể `Policy.InScope` (`intelligence/router/router.go:143`) và `Lineage` (`data/entityresolution/resolver.go:251`); cờ `Value.Approx` cùng đường chuyển cờ tới tổng (`core/contracts/contracts.go:80-82`, `query.go:208-210`); `PackVersion` (`core/tenant/pack.go:98-100`); cách kiểm cấu hình lúc khởi động (`data/ingestion/sources.go:62-130`); cổng bảo toàn kiểu G1 DATA (`docs/04-core-mvp.md` §6).
5. **M1 (làn 0)**, đã lên main ở `41bc3cf` [CODE `41bc3cf`; bản nháp đầu của ADR này viết khi nhánh `lan0/ung-dung-va-vet` còn chưa gộp]: `POST /v1/ingest` (`cmd/eais/server.go:361`) nạp lại **thư mục dữ liệu đã cấu hình** của tenant, thân rỗng, principal từ token, chỉ vai toàn quyền; lưu cách ly (khoá, nguồn, lý do, chi tiết) theo mã lần nạp rồi thay trọn dữ liệu (`cmd/eais/app.go:247-292`). Migration `0003` có `decision_records`, `quarantine_records`, `usage_records`; vai `eais_app` chỉ SELECT + INSERT. Mã bước vết `L1.*` của tầng nạp đã có (`observability/traces/step.go:32-39`) nhưng chưa có chỗ phát. Nghĩa: đã có nền lưu cách ly và mã bước vết, nhưng chưa có đường nạp từng đơn vị, chưa có kết cục chờ xác nhận hay trùng, chưa có người nhận, chưa có vết của đơn vị nạp. Điểm nối cần thống nhất với M1: U2.
6. **Quyết định của chủ dự án** [REPO]: D-RM-04 = A (mã Giai đoạn II mở sau H4); D-RM-03 = chưa có người nghiệp vụ; D-AI-00 (không lời gọi model trả tiền trước CỔNG TRƯỚC AI); D-MVP-00 (MVP gồm M2); D-OP-14 = a (đã vào main); D-OP-17 còn mở.
7. **Cổng QuânIT (trên giấy)**: trượt câu 1 và 8 (không có người dùng thật, không có người ký), ◐ câu 3 và 9. Không chặn việc viết giấy; giới hạn kết luận: "trên dữ liệu giả lập, chưa có người nghiệp vụ duyệt".
8. **Lệch write-set** [REPO]: `docs/PROMPT-BUILD-EAIS.md` §9.2 ghi làn 4 làm `execution/`, migration mới, pack; mã 0a lại nằm ở tầng nạp `data/ingestion/` và `platform/pgstore/` (`docs/01-architecture.md` §3). Ngoài ra mã 0a phụ thuộc hai việc thuộc write-set của làn 0 (nối `POST /v1/ingest` và đường nạp từng đơn vị ở `cmd/eais/`; phát bước `L1.*` ở `observability/`) và một việc của làn 1 ("≈" ở `data/normalization/`). Bảng việc, làn, tệp và phương án lùi: `HOP-DONG.md` §14.1.

## Quyết định

Mỗi quyết định có chi tiết và ca kiểm ở `HOP-DONG.md`; ADR chỉ ghi lựa chọn và lý do.

| # | Quyết định | Chi tiết |
|---|---|---|
| **D1** | **Kho bản ghi báo cáo**, chỉ thêm và đổi trạng thái, lưu bền, RLS FORCE, khoá duy nhất `(tenant, unit_key)`: mỗi đơn vị một bản ghi (`ACCEPTED`, `HELD`, `QUARANTINED`). Cộng **sổ lần giao** chỉ thêm: mỗi **lần giao** của một đơn vị có đúng một dòng với một kết cục có lý do (`ACCEPTED`, `HELD`, `QUARANTINED`, `DUPLICATE`); `DUPLICATE` chỉ có ở sổ, trỏ về bản đã có, không tạo bản ghi (mục G). `facts` chỉ là **phép chiếu**: chỉ bản ghi `ACCEPTED` loại `OCCURRED` thành fact, trong cùng giao dịch với kết cục. `aggregate` **không đổi**. | HOP-DONG §3.1, §3.6, §3.8 |
| **D2** | **Nơi đặt mã:** mở rộng các gói đã có (`data/ingestion/`, `platform/pgstore/`), tệp mới; không tạo thư mục mới (bảng dưới). Hai gói này ngoài write-set làn 4 hiện hành: Q-0A-12; nếu không mở rộng thì dùng phương án lùi (gói mới dưới `execution/`). | bảng "Nơi đặt mã", `HOP-DONG.md` §14.1 |
| **D3** | **Cờ loại là hai trục** `kind` ∈ {`OCCURRED`, `PLAN`, `CORRECTION`} × `approx`; bốn cờ của brief suy ra từ hai trục. Gán bằng luật tất định từ lexicon của pack; luật không chắc thì `HELD`; **bảo thủ cả hai chiều**. | §3.3 |
| **D4** | **Đính chính không tự áp:** luôn `HELD`, người quyết `APPLY` hoặc `REJECT`. | §3.3, §6.2 |
| **D5** | **Chống trùng ba mức:** khoá nguồn (TR1), vân tay (TR2; đơn vị không có `sent_at` có vân tay riêng theo ngày xảy ra, mã nguồn trong tệp, chữ), nghi trùng trong cửa sổ W (TR3, chờ người quyết). TR1 và TR2 cho `DUPLICATE` ở sổ lần giao; thu hồi và sửa tin xét trước TR1. TR3 đánh giá theo **thứ tự chuẩn** `(sent_at, unit_key)` chứ không theo thứ tự đến, đánh giá lại (chỉ siết) khi bản gửi sớm đến muộn, chạy dưới **khoá tư vấn** theo (tenant, thực thể, ngày xảy ra) (mục H), và **chạy cho mọi đường vào nhóm, kể cả quyết định của người**: chỉ `KEEP_BOTH` miễn, cho đúng cặp đã xét; lệnh `KEEP_BOTH` và `MARK_DUPLICATE` mang `duplicate_of` mà người đã thấy, lệch lúc ghi thì từ chối (mục I). Bản rời nhóm (thu hồi, sửa, bác) không làm máy nhả `HELD`, nhưng bản `ACCEPTED` đứng ngay sau nó được đánh giá lại với bản đứng trước mới, chỉ siết; bản `REJECTED` bởi `MARK_DUPLICATE` quay lại `HELD` khi bản được giữ của nó thôi được tính vì bất kỳ lý do gì (luật theo sự kiện, cài đặt ở chỗ ghi trạng thái; thu hồi, sửa tin, `APPLY`, `AUTHORIZE`, `APPLY` đính chính… chỉ là ví dụ), từ bất kỳ trạng thái của bản được giữ, kể cả khi nó đang `REJECTED` bởi `MARK_DUPLICATE` (ngoại lệ có tên của BB7; mục J). **Không** khử theo giá trị, **không** "bản mới nhất thắng". | §3.8, §5, §5.6, §6.2 |
| **D6** | **Quyền người gửi theo máy:** danh bạ người gửi là dữ liệu tenant; vai và capability (`reports.submit`, `reports.review`, `reports.decide`) ở pack; đóng mặc định; tenant chỉ từ danh tính. | §4 |
| **D7** | **Cách ly và chờ xác nhận lưu bền, có mã lý do trong danh sách đóng, có người nhận; báo cáo xuất tất định.** | §3.4, §6 |
| **D8** | **Nhánh form / bot dùng chung hợp đồng đầu ra** với đường chat; đo song song bằng bản chiếu cặp; chọn kênh theo số đo và chi phí đổi thói quen (không phải việc của ADR). | §13 |
| **D9** | **Không** model, **không** interface chung, **không** thư mục theo tên kiến trúc đích, **không** đụng S4 hay `aggregate`. | dưới |
| **D10** | **Kế thừa thực thể** (tin chỉ có lượng, "chiều thêm 1k") chỉ trong một lô, theo `(sent_at, unit_key)`; không có tin nguồn trong lô thì cách ly `unknown_entity` (mục K). | §3.3 (LK7) |

## Nơi đặt mã

Theo `docs/01-architecture.md` §3 và `CLAUDE.md` §25: thư mục chỉ được tạo khi có mã thật cần boundary đó. Bảng này là kế hoạch cho lúc mã được mở; **hiện chưa tạo gì**.

| Đường dẫn | Hiện có? | Phase 0a làm gì |
|---|---|---|
| `data/ingestion/` | có | tệp mới trong gói: phong bì tin, phân loại `kind` và `approx`, chống trùng TR1–TR3, quyền người gửi, kết cục. Hợp đồng `Ingest` hiện có cho nguồn có cấu trúc khác không đổi. **Ngoài write-set làn 4** (Q-0A-12) |
| `data/ingestion/bridge.go` | có | `Facts()` chỉ chiếu bản ghi `ACCEPTED` loại `OCCURRED`; điền `ObservedAt` và `ReportedBy`; sửa nhỏ |
| `platform/pgstore/` | có | tệp mới: kho bản ghi báo cáo và sổ lần giao (ghi từng đơn vị trong một giao dịch, quyết định, báo cáo xuất, khoá tuần tự hoá TR3). M1 đã có `vet.go` (`SaveQuarantine`, bảng vết) và khoá tư vấn của `Load`: tách tệp, thống nhất bảng (U2). **Ngoài write-set làn 4** (Q-0A-12). Phase 1 (ADR-0011) cũng thêm ở đây một hàm mở giao dịch tenant (`Store.inTenant` không xuất): hai phase dùng chung một hàm |
| `db/migrations/` | có | migration mới, **số kế tiếp sau migration cuối của main lúc mở mã** (tại `269a539`: 0004, vì `0003_kho_vet_ben.sql` của M1 đã lên main). ADR-0011 cũng lấy số kế tiếp lúc mở mã của nó (`<NNNN>`): hai phase không được cùng một số, bên mở mã sau lấy số kế tiếp |
| `tenants/construction/pack/` | có | `lexicon.yaml`: từ khoá đính chính, kế hoạch (cờ "≈" là việc của làn 1); `policy.yaml`: capability `reports.*`, vai `site_supervisor` (Q-0A-7); `sources.yaml`: loại nguồn có cấu trúc cho báo cáo của đội, danh sách bỏ qua dòng PDF. **Làn 1 giữ pack**: phối hợp khi mở mã |
| dữ liệu tenant | (cùng chỗ `entities.json`) | danh bạ người gửi (`HOP-DONG.md` §4.3) |
| `core/contracts/` | có | chỉ kiểu dùng chung nếu nhiều gói cần (kết cục, loại); không bắt buộc |
| `tenants/construction/win/`, `scripts/dot-bien/` | có | làn phép đo: bộ kiểm bảo toàn không import bộ nạp, chạy bộ ca độc lập, đột biến DB01–DB36 |
| `execution/` | chưa (Phase 1 sẽ tạo `execution/commands`, ADR-0011) | **không tạo ở Phase 0a**, trừ phương án lùi nếu Q-0A-12 = không: một gói mới (ví dụ `execution/reports/`) cho kho bản ghi, sổ lần giao và luật nạp (`HOP-DONG.md` §14.1) |
| `governance/`, `domain/` | chưa | **không tạo**: sự kiện đầy đủ là Phase 3; xác thực là Phase 5; connector là Phase 8 |
| `cmd/eais/`, `observability/` | có (M1, làn 0) | **không thuộc write-set làn 4**: nối `POST /v1/ingest`, đường nạp từng đơn vị và phát bước `L1.*` là việc 3 và 4 của `HOP-DONG.md` §14.1 (Q-0A-15) |
| `data/normalization/` | có | **không sửa ở Phase 0a**: nhận "≈" (HT-1) là việc của làn 1; GA3 phần "≈" chờ việc đó (`HOP-DONG.md` §14.1 mục 6) |

**Việc chéo làn** (R2-07; chi tiết, tệp và phương án lùi: `HOP-DONG.md` §14.1). Mã 0a không tự đủ trong write-set của làn 4 (`docs/PROMPT-BUILD-EAIS.md` §9.2: `execution/` mới, migration mới, pack):

| Việc | Tệp | Làn có tệp trong write-set | Xử lý |
|---|---|---|---|
| kho bản ghi, sổ lần giao, quyết định, báo cáo xuất, khoá TR3 | `platform/pgstore/`, `db/migrations/` | migration: làn 4; `platform/pgstore/`: không làn nào | Q-0A-12; lùi: gói mới dưới `execution/` |
| luật nạp, phép chiếu fact | `data/ingestion/` | không làn nào | Q-0A-12; lùi như trên |
| đường nạp từng đơn vị; nối `POST /v1/ingest` | `cmd/eais/` | làn 0 | Q-0A-15 |
| phát bước `L1.*`, vết của đơn vị nạp | `observability/`, `cmd/eais/` | làn 0 | Q-0A-15 |
| nhận "≈" (HT-1) | `data/normalization/engine.go` | làn 1 | chờ làn 1; GA3 phần "≈" ĐỎ tới lúc đó |
| pack: capability, vai, loại nguồn, từ khoá | `tenants/construction/pack/` | làn 1 và làn 4 | phối hợp |

Ràng buộc kiến trúc phải giữ:
- mã dùng chung không chứa tên tenant hay từ miền trong chuỗi literal (`architecture_test.go`, `TestSharedCodeHasNoTenantLiterals`); từ khoá đính chính, kế hoạch ở lexicon, không ở mã;
- phong bì tin **không** có trường tenant (cùng tinh thần `TestRequestTypesHaveNoTenantField`);
- `core/pipeline/` và `intelligence/` không bị sửa: đó là vùng của làn 1.

### Cổng đổi kiến trúc (`CLAUDE.md` §34)

| # | Câu | Trả lời |
|---|---|---|
| 1 | Problem gì? | lỗi im lặng L1: ước lượng, đính chính, kế hoạch, bản trùng thành số "chính xác"; cách ly không bền |
| 2 | Reproducer? | HT-1…HT-5, HT-8…HT-10 (`HOP-DONG.md` §1.3.2); K1–K5 [ĐO] |
| 3 | Abstraction hiện tại hỏng ở đâu? | quan sát không mang giờ, người, loại; chỉ có hai đầu ra (quan sát, cách ly RAM) |
| 4 | Có cách nhỏ hơn? | vá từng lỗi trong `ingest.go` không cho chỗ lưu cách ly bền, không có người nhận, không có "chờ người": xem A2, A3 |
| 5 | Vì sao thuộc shared core? | biên kiểm soát nạp (bảo toàn, chống trùng, cách ly) là chung; **nhưng chỉ một tenant dùng** (nợ, hệ quả 2) |
| 6 | Tenant-specific có đủ không? | không: từ khoá thì ở pack (có); cơ chế thì chung |
| 7 | Tương thích? | `Ingest` cho nguồn có cấu trúc giữ nguyên; `aggregate` giữ nguyên; `PackVersion` đổi vì pack đổi |
| 8 | Đường chuyển? | thêm bảng và đường chiếu; `Load` phối hợp (U2) |
| 9 | Test? | 84 lớp ca âm (`HOP-DONG.md` §10), 36 đột biến (§11.3); mỗi dòng cổng có ít nhất một đột biến, và mỗi cặp (đột biến, dòng) ghi lớp ca của chính dòng đó làm nó đỏ |
| 10 | Metric? | 11 dòng cổng GA1–GA11 (§11) |
| 11 | ADR? | đây |

## Phương án đã cân nhắc

### A. Chỗ lưu và quan hệ với `facts`

| Phương án | Ưu | Nhược |
|---|---|---|
| **A1 (chọn):** kho bản ghi riêng cộng phép chiếu sang `facts` | khớp bốn tầng OBSERVATION → FACT (`docs/01-architecture.md` §5.0); lọc ở biên cầu nối nên **không đổi `aggregate`** (lõi hỏi–đáp của làn 1); `HELD`, `QUARANTINED`, `DUPLICATE` có chỗ ở | thêm hai bảng (kho bản ghi và sổ lần giao) và một đường chiếu; `Load` thay trọn `facts` (`load.go:27`) nên phải phối hợp (U2) |
| A2: thêm cột `kind`, `disposition` vào `facts`, để `aggregate` lọc | một bảng | phải sửa lõi đọc (làn 1); fact trộn lẫn bản ghi không dùng được; cách ly vẫn không có chỗ vì không phải fact |
| A3: giữ trong RAM, ghi tệp sidecar | không đổi lược đồ | không bền, không RLS, không đạt "lưu bền" của hàng L1 |

### B. Mô hình cờ loại

| Phương án | Ưu | Nhược |
|---|---|---|
| **B1 (chọn):** hai trục `kind` × `approx` | giữ thông tin ở tổ hợp (đính chính ước lượng, kế hoạch ước lượng); `approx` đã có đường chuyển tới câu trả lời | thêm một trường |
| B2: một enum bốn giá trị (đã xảy ra, ước lượng, đính chính, kế hoạch) | đơn giản | mất thông tin khi tổ hợp: ca bơm "đúng loại" không phân định được |
| B3: một chỉ số riêng cho mỗi loại trong pack (`planned_fuel`, …) | không đổi `Observation` | thẩm quyền chỉ so trong cùng một chỉ số (`metrics.yaml:1-6`); kế hoạch và đính chính không phải "nghĩa khác của một con số"; nhân số chỉ số, nhân phép dẫn xuất |
| B4: để model gán loại | tổng quát hơn | cổng đóng: model chỉ mở khi luật trượt trên ca đo được (`CLAUDE.md` §19) và không lời gọi trả tiền trước CỔNG TRƯỚC AI (D-AI-00); không tất định |

### C. Đính chính

| Phương án | Ưu | Nhược |
|---|---|---|
| **C1 (chọn):** luôn `HELD`, người quyết `APPLY` / `REJECT` | an toàn nhất: không tự đổi số đã có; đủ để M2 ngừng tạo số sai im lặng | người xác nhận mọi đính chính (tần suất chưa đo, U4) |
| C2: tự áp khi có đúng một bản gốc ứng viên | ít việc người hơn | tin đính chính nhầm ngày hay nhầm người sẽ ghi đè số đúng; cần chính sách của người nghiệp vụ (Q-0A-4) |
| C3: để hết cho Phase 3 | không làm gì ở 0a | giữ nguyên HT-2: trái chữ khối M2 ("bản ghi có cờ … đính chính") |

### D. Chống trùng

| Phương án | Ưu | Nhược |
|---|---|---|
| **D1 (chọn):** ba mức TR1 (khoá nguồn), TR2 (vân tay), TR3 (nghi trùng, chờ người) | phủ cả tin giao hai lần, tệp xuất chồng lấp, và gửi lại sau vài phút; không bao giờ tự gộp ở mức nghi vấn | TR3 giữ oan khi hai lần cấp thật cùng số trong W (chưa đo tần suất) |
| D2: chỉ khoá nguồn | đơn giản | không bắt tệp xuất chồng lấp (khác mã) hay tin gửi lại; HT-5 còn nguyên |
| D3: khử theo giá trị | bắt mọi bản trùng | sai: hai lần cấp hợp lệ cùng số (R-001, R-002 của seed) bị gộp; báo cáo K1–K5 K4 đã chỉ ra |
| D4: "bản mới nhất thắng" theo (thực thể, ngày) | không cộng đôi | sai: nhiều lần cấp trong ngày là nhiều sự kiện (`query.go:199-200`) |
| D5: khử ở lúc đọc (trong `aggregate`) | không đổi phía nạp | đụng lõi của làn 1; mỗi truy vấn tốn thêm; bản trùng vẫn nằm trong `facts` |

### E. Quyền người gửi

| Phương án | Ưu | Nhược |
|---|---|---|
| **E1 (chọn):** danh bạ người gửi (dữ liệu tenant) cộng phạm vi cộng capability ở pack; đóng mặc định | khớp `scope: attribute` đã có; giải được bí danh mơ hồ giữa hai dự án bằng phạm vi (CON-F-015, S1) | dữ liệu mới phải có người nuôi |
| E2: tin tên hiển thị của người gửi | không cần danh bạ | trùng tên, giả mạo dễ; không gắn được phạm vi |
| E3: bỏ quyền ở 0a, chờ Phase 5 | ít việc | giữ lỗ CON-F-015: "máy 03" gán nhầm sang máy cùng số của dự án khác (S1) |

### F. Nhánh form / bot

| Phương án | Ưu | Nhược |
|---|---|---|
| **F1 (chọn):** hai đường dùng chung hợp đồng đầu ra, đo song song | trả lời câu 3 của cổng QuânIT; chọn kênh bằng số đo | hai đường cùng cần đóng |
| F2: chỉ form, bỏ chat | hết lỗi đọc chữ ở nguồn | là quyết định nghiệp vụ (đổi thói quen của đội), chưa có người nghiệp vụ (D-RM-03); bỏ phần đang có |
| F3: chỉ chat | không đổi thói quen | bỏ qua câu 3 của cổng ("chuẩn hoá đầu vào trước khi đọc chữ tự do") |

### G. Chỗ ở của `DUPLICATE` (ghi chú soát F01; quyết định D1)

| Phương án | Ưu | Nhược |
|---|---|---|
| **Sổ lần giao riêng, chỉ thêm (chọn):** mỗi lần giao một dòng; kho bản ghi chỉ có `ACCEPTED`, `HELD`, `QUARANTINED` | khoá duy nhất `(tenant, unit_key)` chỉ áp cho kho bản ghi nên không đụng bản trùng; nạp lại cho cùng trạng thái cuối (sổ thêm dòng, kho không đổi; trừ chuỗi sửa tin từ hai lần trở lên: nợ 23); BB1 kiểm được từ kho trên cả lần giao lại; cùng mẫu `quarantine_records` của M1 (khoá theo mã lần nạp, chỉ thêm) | thêm một bảng và một dòng ghi cho mỗi đơn vị mỗi lần giao (dung lượng chưa đo) |
| `DUPLICATE` là dòng thứ hai trong kho bản ghi | một bảng | với TR1, bản trùng cùng `unit_key` với bản gốc nên vi phạm khoá duy nhất; đổi khoá thì mỗi lần nạp lại thêm dòng và "trạng thái cuối giống hệt" sai |
| `DUPLICATE` chỉ ở vết | không thêm bảng | bộ kiểm bảo toàn (không import bộ nạp) phải tin vết, mà vết là đối tượng của GA10; lần giao trùng rơi im lặng không bị bắt; vết của đơn vị nạp chưa có chỗ lưu (M1 chỉ lưu vết lượt hỏi) |

### H. Thứ tự và đua của TR3 (ghi chú soát F02; quyết định D5)

| Phương án | Ưu | Nhược |
|---|---|---|
| **Thứ tự chuẩn `(sent_at, unit_key)`, đánh giá lại khi bản gửi sớm đến muộn (chỉ siết), cộng khoá tư vấn theo (tenant, thực thể, ngày) (chọn)** | kết cục là hàm của tập bản: giữ nguyên "đảo thứ tự không đổi tổng, cờ, tập `HELD`" (§7.4, LC63) và đúng cả khi hai tiến trình chạy song song; khoá cùng kiểu với `Load` của M1 | một bản đã `ACCEPTED` có thể thành `HELD` khi bản sớm hơn đến muộn; khoá làm các giao dịch cùng (thực thể, ngày) xếp hàng |
| Giữ "so với bản đã có lúc đến", hạ LC63 và §7.4 xuống "cùng tổng và cùng số `HELD`" | không đánh giá lại | chuỗi ba bản cho số `HELD` khác nhau theo thứ tự đến (`HOP-DONG.md` §5.6, tính tay), nên điều hạ xuống vẫn sai; không giải đua |
| Giữ như cũ, chỉ thêm khoá | nhỏ nhất | LC63 và §7.4 vẫn mâu thuẫn với TR3: đúng điều ghi chú F02 nêu |
| Mức cô lập SERIALIZABLE kèm thử lại | không tự quản khoá | mã thử lại ở mọi đường ghi; báo lỗi tuần tự hoá oan; khoá tư vấn đã có tiền lệ ở `Load` |

### I. Quyết định của người và TR3 (ghi chú soát vòng 2 R2-01 và vòng 3 R3-04; quyết định D5)

Các con số dưới đây là [SUY, mô phỏng luật bằng script tạm không commit, không phải mã]: 30.000 chuỗi sự kiện ngẫu nhiên (3 đến 10 sự kiện gồm nạp, quyết định, `KEEP_BOTH`, `MARK_DUPLICATE`, thu hồi; 2 đến 5 bản; 195.052 trạng thái kiểm sau từng sự kiện; script vòng 3 ghi đè cặp đã xét và cho 195.026, đã sửa ở vòng 4 cho cộng dồn đúng `HOP-DONG.md` §5.6 mục 3, cùng kết quả 0 chuỗi vi phạm). Bất biến kiểm: hai bản liền kề trong nhóm, cách ≤ W, chưa được người xét cặp thì không cùng `ACCEPTED`; đánh giá lại là điểm bất động.

| Phương án | Ưu | Nhược |
|---|---|---|
| **Quyết định của người chạy TR3 như mọi đường vào nhóm; chỉ `KEEP_BOTH` miễn, cho đúng cặp đã xét (chọn)** | đóng đường cộng đôi im lặng khi người quyết một bản đang `HELD` vì lý do khác; không cần khái niệm "bản được bảo vệ"; kết cục TR3 là hàm của nhóm và của các cặp đã xét; mô phỏng: 0 chuỗi vi phạm | thêm một bước người (`KEEP_BOTH`) cho bản vừa được quyết mà nghi trùng; bản đã quyết có thể bị giữ lại khi bản gửi sớm đến muộn (không im lặng: vào hàng chờ, có người nhận) |
| Quyết định của người miễn TR3 (luật vòng 2) | một bước người | cộng đôi im lặng (R2-01): A `ACCEPTED`, B `HELD` vì lý do khác, người quyết B thì cả A và B vào tổng; chữa một chiều vẫn hở chiều ngược |
| Bản đã quyết được bảo vệ khỏi đánh giá lại; máy giữ bản kia của cặp (hai chiều) | không đổi bản người đã quyết | mô phỏng: 54 trong 30.000 chuỗi vẫn có hai bản liền kề cùng `ACCEPTED` (ví dụ sau khi bản ở giữa rời nhóm làm hai bản đã quyết liền kề); thêm khái niệm bảo vệ và luật hai chiều |
| **Lệnh `KEEP_BOTH`, `MARK_DUPLICATE` mang `duplicate_of` mà người đã thấy; lúc ghi khác thì từ chối và trả lại hàng chờ; `MARK_DUPLICATE` chỉ khi `duplicate_of` còn (chọn, vòng 4: R3-04)** | đóng đường rơi khỏi tổng im lặng: người bấm `MARK_DUPLICATE` trên màn hình cũ khi bản kia đã rời nhóm không còn làm bản bị giữ `REJECTED` mà không có bản kia; cặp đã xét luôn là cặp người đã thấy; chỉ thêm một trường vào lệnh, không thêm trạng thái | thêm một lần bấm khi màn hình cũ; lệnh thiếu trường bị từ chối (đóng mặc định); đường "quyết định đúng, rồi bản được giữ bị thu hồi" do mục J chữa ở vòng 5 và vòng 6 (nợ 25) |
| Ghi quyết định theo `duplicate_of` lúc ghi (chữ vòng 3) | không thêm trường | `REJECTED` "trùng với bản kia" khi bản kia không còn: cả hai rơi khỏi tổng và hàng chờ; cặp đã xét hoặc bản được giữ có thể không phải cặp người đã thấy |

Vòng 4 (R3-01, R3-04), cùng loại phép đo [SUY, mô phỏng luật bằng script tạm không commit, không phải mã; hạt giống cố định; 2 đến 5 bản, 4 đến 12 sự kiện gồm nạp, quyết định, mở màn hình, `KEEP_BOTH`, `MARK_DUPLICATE`, thu hồi, sửa tin đổi nhóm, bác; bộ sinh nghiêng về sự kiện hiếm nên số đếm không phải tần suất thật]: trên 300.000 chuỗi, chữ vòng 3 (bản `ACCEPTED` đứng sau bản rời nhóm không đổi; quyết định ghi theo `duplicate_of` lúc ghi) cho 180 chuỗi có hai bản liền kề, cách ≤ W, chưa xét cặp, cùng `ACCEPTED`; 1.791 chuỗi có bản `REJECTED` bởi `MARK_DUPLICATE` mà bản kia không còn trong nhóm; 256 chuỗi có cặp đã xét hoặc bản được giữ khác cặp người đã thấy. Chữ vòng 4: 0, 0 và 0 trên cùng 300.000 chuỗi; đánh giá lại là điểm bất động. Còn lại: 2.521 chuỗi có bản bị bác mà bản được giữ rời nhóm sau quyết định (nợ 25); vòng 5 vá phần còn lại này ở mục J (R4-01) và vòng 6 vá chỗ hở còn sót (R5-01).

### J. Bản rời nhóm TR3 (ghi chú soát vòng 2 R2-02, vòng 3 R3-01, vòng 4 R4-01, vòng 5 R5-01 và vòng 6 R6-01; quyết định D5)

| Phương án | Ưu | Nhược |
|---|---|---|
| **Chỉ siết: bản `HELD` ở lại khi bản đứng trước rời nhóm; bản `ACCEPTED` đứng ngay sau bản rời nhóm được đánh giá lại với bản đứng trước mới (R3-01); bản `REJECTED` bởi `MARK_DUPLICATE` quay lại `HELD` khi bản được giữ của nó thôi được tính vì bất kỳ lý do gì (luật theo sự kiện, móc ở chỗ ghi trạng thái; R4-01, R5-01, R6-01), từ bất kỳ trạng thái của bản được giữ, kể cả khi nó đang `REJECTED` bởi `MARK_DUPLICATE`; ghi ngoại lệ có tên (`HOP-DONG.md` §7.4 (c), U10, BB7) (chọn)** | giữ nguyên tắc "máy không nhả `HELD`" (BB6, BB7) và luật F02 đã qua soát; không thêm cơ chế; hai bản liền kề, cách ≤ W, chưa xét cặp không cùng `ACCEPTED` sau mọi sự kiện, kể cả bản rời nhóm; cả hai kết cục an toàn (tổng không cộng đôi, không bản nào rơi khỏi tổng hay hàng chờ, kể cả bản đã bị `MARK_DUPLICATE`) | thứ tự đến của thu hồi, sửa tin, bác đổi `HELD` hay `ACCEPTED` của bản đứng sau; bản `ACCEPTED` đứng sau có thể bị siết thành `HELD`; bản bị bác quay lại hàng chờ (thêm việc người, kể cả bản `HELD` có `duplicate_of` rỗng: nợ 24); bản trỏ tới bản được giữ đang bị bác theo một bản còn sống vẫn quay lại dù giá trị đã có bản còn sống đó (hướng nối lại `duplicate_of` sang bản còn sống, chỉ quay lại khi chuỗi hết, bị bỏ vì thêm cơ chế; R5-01) |
| Bản `ACCEPTED` đứng ngay sau bản rời nhóm không đổi (chữ vòng 3) | không thêm việc người | cộng đôi im lặng (R3-01): a, m, c nghi trùng nối tiếp, `KEEP_BOTH` cho c rồi m rời nhóm thì a và c liền kề, cách ≤ W, cùng `ACCEPTED`, cặp (c, a) chưa ai xét; số đếm ở mục I |
| Tính lại tất định: máy nhả `HELD` khi không còn bản đứng trước cách ≤ W | kết cục là hàm của tập bản kể cả khi thu hồi | máy tự nhả `HELD`, đổi nguyên tắc đã soát; phải kiểm bằng bộ ca; câu mở Q-0A-13 |
| Bản rời nhóm vẫn là "bản đứng trước" (bóng) | tất định theo thứ tự đến | giữ oan bản đúng vì một bản không còn tính vào tổng; phải giữ bóng của mọi phiên bản |
| Bản `REJECTED` bởi `MARK_DUPLICATE` giữ nguyên khi bản được giữ rời nhóm (chữ vòng 4; nợ 25 cũ) | không thêm việc người | rơi khỏi tổng im lặng (R4-01): a, b nghi trùng, người `MARK_DUPLICATE` b, rồi a bị thu hồi thì tổng rỗng và b không ở hàng chờ; mô phỏng ghép cặp, đo lại ở vòng 6 với bản `REJECTED` chạm được [SUY, script tạm không commit]: 1.399 đến 3.371 chuỗi trong 300.000 (W = 1, 2, 3, 5), chữ vòng 6: 0 trên mô hình vòng 6, chưa có `APPLY` đính chính (xem hàng R6-01 dưới đây; `HOP-DONG.md` §5.6, "Phương án đã bỏ ở vòng 5, vòng 6 và vòng 7") |
| Mục 4 (c) chỉ chạy khi bản được giữ "vừa rời nhóm" từ `ACCEPTED` hoặc `HELD` (chữ vòng 5; R5-01) | không thêm việc người khi bản được giữ đã bị bác | rơi khỏi tổng im lặng (R5-01): a, b, c nghi trùng nối tiếp, `MARK_DUPLICATE` c rồi b, thu hồi b và người `APPLY` thì c vẫn `REJECTED` theo bản đã `RETRACTED`; thu hồi a thì tổng rỗng và c không ở hàng chờ; đường thứ hai không phụ thuộc cách đọc §3.8 (phiên bản chờ có trước `MARK_DUPLICATE`); duyệt hết 3 bản, thời điểm 0 đến 2 [SUY, script tạm không commit]: 486.388 trạng thái, 7.524 rơi (W = 1) và 628.309, 10.692 (W = 2); chữ vòng 6: 0 trên mô hình vòng 6, chưa có `APPLY` đính chính (xem hàng R6-01 dưới đây; `HOP-DONG.md` §5.6) |
| Mục 4 (c) liệt kê nguyên nhân: "bằng thu hồi hoặc sửa tin" và `APPLY` lần giao đó (chữ vòng 6; R6-01) | không thêm cơ chế ngoài các lệnh đã có | rơi khỏi tổng im lặng (R6-01): a `ACCEPTED`, b nghi trùng a, `MARK_DUPLICATE` b, tin đính chính của người gửi a, người `APPLY` (a `SUPERSEDED`) thì b vẫn `REJECTED` theo bản không còn được tính, nhóm rỗng, hàng chờ rỗng; cùng họ với R3-04, R4-01, R5-01 (bốn vòng soát liền, mỗi vòng thêm một nguyên nhân; nguyên nhân kế còn có thể là `AUTHORIZE` áp lần giao của người gửi khác và `REPROCESS` [SUY]); mô phỏng đủ sự kiện của vòng 7 [SUY, script tạm không commit]: 4.134, 7.458 và 9.840 trạng thái rơi (3 bản, 10 mẫu thời điểm, W = 1, 2, 3; trên 33.763, 56.781 và 72.710 trạng thái); 4 bản, W = 2: 4.370.610 trên 21.689.269; bỏ `APPLY` đính chính và `REPROCESS` thì 0; luật theo sự kiện: 0 (`HOP-DONG.md` §5.6, "Phương án đã bỏ ở vòng 5, vòng 6 và vòng 7") |
| Coi bản được giữ là bản đã có quyết định của người (BB7): thu hồi hoặc sửa nó thì `HELD` `id_conflict`, người quyết `APPLY` hoặc `REJECT` (hướng (2) của U15 ở vòng 4) | không thêm cơ chế đưa bản về hàng chờ | chỉ dời chỗ rơi [SUY, tính tay, chưa chạy]: khi người quyết `APPLY` thu hồi thì bản bị bác vẫn `REJECTED` và tổng mất cả hai, trừ khi `APPLY` cũng đưa bản bị bác về hàng chờ, tức là phải có chính cơ chế đã chọn; thêm việc người cho mọi thu hồi, sửa tin của bản từng là bản được giữ |

### K. Kế thừa thực thể (ghi chú soát vòng 2 R2-02; quyết định D10)

| Phương án | Ưu | Nhược |
|---|---|---|
| **Chỉ trong lô, theo `(sent_at, unit_key)` (chọn)** | bỏ phụ thuộc thứ tự dòng; chặn việc nhận thực thể của tin xa hơn khi tin gửi xen giữa đến sau [SUY từ mã]; không thêm cơ chế đánh giá lại | tin chỉ có lượng ở lô khác bị cách ly `unknown_entity` (an toàn; người `CORRECT`); connector giao từng tin (Phase 8) không kế thừa được |
| Giữ thứ tự xử lý như mã hôm nay (`c.last`, `ingest.go:271-285`) | không đổi mã | đảo hai dòng thì tin thứ hai bị cách ly (người soát vòng 2 đo); và có thể nhận nhầm thực thể im lặng [SUY] |
| Xuyên lô qua kho theo `sent_at`, đánh giá lại tin đã cách ly khi tin mang thực thể đến muộn | tất định cả khi chia lô | máy nhả `QUARANTINED`; thêm cơ chế; câu mở Q-0A-14 |

## Hệ quả

**Tích cực**
- M2 đo được **độc lập với S4**: cổng đọc lại đi qua kho sự thật, không qua luồng hỏi.
- Làn 1 không bị chặn: `aggregate`, S4, `core/pipeline/` không đổi.
- Nền cho Phase 3: `kind`, `corrects`, `supersedes`, `version`, sổ kết cục.
- Cách ly có người nhận, lưu bền, xuất được báo cáo (hàng L1 của §9.3).

**Tiêu cực và nợ có nhãn**
1. **Phối hợp với `Load`** (thay trọn `facts` của tenant): phép chiếu phải được dựng lại trong cùng giao dịch, hoặc `Load` phải ghi theo nguồn. Chọn lúc mở mã (U2).
2. **Một tenant dùng**: chưa chứng minh là `SHARED CORE` (`docs/CONVENTIONS.md` §4.5). Ghi nợ "một tenant" tới Phase 4a.
3. **Việc của người tăng:** mọi đính chính và nghi trùng chờ người quyết; tần suất và chi phí chưa đo (U4).
4. **TR3 giữ oan** hai lần cấp thật cùng số trong W; W là đề xuất (Q-0A-5).
5. **Lexicon chỉ là luật:** có thể thiếu từ. Bù bằng bảo thủ hai chiều (LK3, `HELD` khi không chắc), nhưng tỉ lệ `HELD` chưa đo; nếu quá cao thì Phase 6 (model phân loại) có bằng chứng đòi.
6. **`sender_id` chưa xác thực** (Phase 5, 8).
7. **Danh bạ người gửi** là dữ liệu mới phải có người nuôi.
8. **Hình dạng tin Zalo thật chưa biết** (U5): phong bì là giả định.
9. **Write-set của làn 4** phải mở rộng (Q-0A-12), và hai việc ở làn 0, một việc ở làn 1 phải có chủ (Q-0A-15, nợ 21).
10. **Pack đổi** (`lexicon`, `policy`, `sources`) làm `PackVersion` đổi: các chỗ ghim phiên bản (bằng chứng đã đóng băng) cần kiểm khi mở mã.
11. **Câu trả lời cũ không tự cập nhật** khi đính chính được áp: thuộc Phase 3.
12. **Kết luận chỉ "trên dữ liệu giả lập, chưa có người nghiệp vụ duyệt"**: câu 1 và 8 của cổng QuânIT còn ✗.
13. **Đánh giá lại TR3 có thể chuyển một bản đã `ACCEPTED` sang `HELD`** khi bản gửi sớm hơn đến muộn (`HOP-DONG.md` §5.6): tổng đọc loại đúng một giá trị trùng, nhưng bản được tính đổi; tần suất bản đến muộn chưa đo (U4).
14. **Khoá tư vấn theo (tenant, thực thể, ngày xảy ra)** làm các giao dịch cùng (thực thể, ngày) xếp hàng; thông lượng chưa đo (U3).
15. **Sổ lần giao thêm một dòng cho mỗi đơn vị mỗi lần giao**, kể cả `ACCEPTED`; dung lượng chưa đo, chính sách lưu giữ chưa có (Phase 5).
16. **`id_conflict` phụ thuộc thứ tự đến** (ngoại lệ có tên của §7.4 và LC63): luật đối xứng chưa chọn (U9).
17. **Quyết định của người không miễn TR3** (R2-01): người quyết một bản đang `HELD` vì lý do khác mà bản đó nghi trùng thì phải quyết thêm `KEEP_BOTH`; bản đã quyết có thể bị giữ lại khi bản gửi sớm đến muộn. Lệnh `KEEP_BOTH` và `MARK_DUPLICATE` mang `duplicate_of` mà người đã thấy (R3-04): quyết trên màn hình cũ thì lệnh bị từ chối và người quyết lại trên màn hình mới. Thêm việc người (một lần bấm); tần suất chưa đo (U4).
18. **Bản rời nhóm TR3** (R2-02, R3-01): thu hồi, sửa tin, bác làm thứ tự đến đổi `HELD` hay `ACCEPTED` của bản nghi trùng đứng sau, kể cả bản `ACCEPTED` đứng ngay sau bị siết thành `HELD` khi cặp với bản đứng trước mới chưa xét (an toàn: không cộng đôi, không rơi bản; thêm việc người); hai bản `ACCEPTED` không liền kề (có bản `HELD` ở giữa), cách ≤ W, không bị chặn trực tiếp; bản `REJECTED` bởi `MARK_DUPLICATE` quay lại `HELD` khi bản được giữ thôi được tính (R4-01, nợ 25). Q-0A-13, U10.
19. **Kế thừa thực thể chỉ trong lô** (R2-02): tin chỉ có lượng ở lô khác bị cách ly; connector giao từng tin không kế thừa được. Q-0A-14, U11.
20. **Đơn vị không có `sent_at` không vào TR3** (R2-06, R3-03): cùng báo cáo xuất lại với số dòng hoặc chữ khác, hoặc đổi tên sang ngày khác, có thể bị tính hai lần; hai tệp khác nguồn cùng ngày, cùng loại nguồn, có dòng cùng mã nguồn trong tệp và cùng chữ bị gộp oan (chưa chạm được với pack hôm nay). Chủ dự án chọn ở Q-0A-16; mặc định: giữ vân tay, chưa thêm loại nguồn bảng nhận tệp của nhiều đội. U12.
21. **Việc chéo làn** (R2-07): kho PG và luật nạp ngoài write-set làn 4 (Q-0A-12); nối route và phát `L1.*` ở làn 0, "≈" ở làn 1 (Q-0A-15). Phương án lùi ở `HOP-DONG.md` §14.1. GA3 phần "≈", GA10 và đường nạp qua HTTP có thể giữ ĐỎ vì lý do làn, không vì lỗi mã 0a.
22. **Trường của hệ** (R2-04): phép so "cùng byte" bỏ `trace_id`, `batch_id`, `recorded_at`, `decided_at` và dấu thời gian do hệ sinh (`HOP-DONG.md` §3.7); bộ chạy của đường form phải đặt được đồng hồ của hệ nhận.
23. **Chuỗi sửa tin từ hai lần trở lên** (R3-02, ghi nợ): lần sửa được so với phiên bản hiện hành, không có khoá thứ tự giữa các lần sửa. Giao lại cả chuỗi thêm phiên bản; lần sửa cũ đến sau lần sửa mới làm tổng dùng giá trị cũ (giá trị mới còn ở lịch sử phiên bản); sửa tin đến trước tin gốc cho `HELD` `id_conflict` ở cả hai lần giao. Ngoại lệ có tên (e) của `HOP-DONG.md` §7.4 và §5.4. Hướng chữa: khoá thứ tự đơn điệu cho lần sửa. U13.
24. **Bốn điểm chưa định nghĩa nhỏ** (R3-05 (a)–(d), ghi nợ): `sent_at` của phiên bản do sửa tin; trạng thái đích của `DISMISS`; thu hồi hoặc sửa tin thiếu `source_message_id` (dấu thu hồi mang khoá tổng hợp nên vô hiệu); TR2 vế (a) cho bản ghi form (không có chữ để chuẩn hoá); thêm một điểm tác giả vòng 4 thấy: bản `HELD` `possible_duplicate` có `duplicate_of` rỗng chỉ còn lối ra `KEEP_BOTH`, không có quyết định bỏ bản đó (bản không mất, nằm ở hàng chờ); người soát vòng 4 (R4-04) thấy bản như vậy chiếm tỉ lệ đáng kể trong mô phỏng của họ, và vòng 5 thêm một đường tạo ra chúng (nợ 25). Thêm một điểm của người soát vòng 4 (R4-03, ghi nợ): `seen_duplicate_of` "có mặt nhưng rỗng" khác "thiếu" (lệnh thiếu trường bị từ chối; cài đặt phải phân biệt), chưa có ca riêng và DB35 chưa nêu "nhận lệnh thiếu trường"; giá trị do người gửi lệnh cấp được ghi vào audit chưa có giới hạn độ dài hay ký tự. Chọn khi mở mã. U14.
25. **Bản được giữ của `MARK_DUPLICATE` thôi được tính sau quyết định** (tác giả vòng 4 thấy khi sửa R3-04; người soát vòng 4 xếp là R4-01, vòng 5 thấy phần còn lại là R5-01, vòng 6 thấy nguyên nhân `APPLY` đính chính là R6-01; **đã vá theo sự kiện ở vòng 7**): lỗ cũ là bản bị bác vẫn `REJECTED` nên cả hai bản rời tổng, không bản nào quay lại hàng chờ; có audit nhưng không ai được hỏi lại (1.399 đến 3.371 trong 300.000 chuỗi của bộ sinh ngẫu nhiên ở mục J, W = 1, 2, 3, 5; không phải tần suất thật). Hai bản vá đầu chưa hết vì cùng một gốc, (c) gắn vào danh sách nguyên nhân. Vòng 5 vá chưa hết: (c) chỉ chạy khi bản được giữ "vừa rời nhóm" từ `ACCEPTED` hoặc `HELD` (R5-01: duyệt hết 3 bản còn 7.524 và 10.692 trạng thái rơi, W = 1 và 2). Vòng 6 vá chưa hết: (c) chỉ chạy khi bản được giữ rời bằng thu hồi, sửa tin hoặc `APPLY` lần giao đó, nên bỏ sót `APPLY` đính chính làm bản gốc `SUPERSEDED` (R6-01: 4.134, 7.458 và 9.840 trạng thái rơi trong mô hình đủ sự kiện, 3 bản, W = 1, 2, 3). Vá (vòng 7): bản bị bác quay lại `HELD` `possible_duplicate` khi bản được giữ của nó thôi được tính vì BẤT KỲ lý do gì, từ bất kỳ trạng thái của bản được giữ, kể cả khi nó đang `REJECTED` bởi `MARK_DUPLICATE`; danh sách nguyên nhân chỉ là ví dụ; cài đặt ở chỗ ghi trạng thái; GA9 kiểm tính chất sau mọi sự kiện (`HOP-DONG.md` §5.6 mục 4 (c), §3.8, §6.2, LC84, DB36, GA9, U15; đo ở vòng 7: 0 ở mọi phép duyệt hết, 3 bản và 4 bản); hướng coi bản được giữ là bản đã có quyết định của người bị bỏ vì chỉ dời chỗ rơi (mục J). Còn là nợ: thêm việc người (bản quay lại hàng chờ, kể cả bản `HELD` rỗng `duplicate_of`: nợ 24); tần suất chưa đo (U4); cài đặt phải đưa MỌI đường đổi trạng thái qua một chỗ ghi, tìm các bản `REJECTED` trỏ tới bản đích trong cùng giao dịch, dưới khoá, kể cả khi bản đích đang `REJECTED`; giả định của mô phỏng ở `HOP-DONG.md` U8; luật theo sự kiện chưa chứng minh hình thức cho mọi số bản (mới duyệt hết tới 4 bản).

## Chứng minh (thứ tự)

Điều kiện mở mã: H4 chạy xong; D-OP-14 đã vào main (có); hợp đồng đơn vị nạp của làn 0 chốt; việc chéo làn có chủ (`HOP-DONG.md` §14.1: kho PG và luật nạp theo Q-0A-12 hoặc phương án lùi; nối route và phát `L1.*` ở làn 0; "≈" ở làn 1 trước khi đo GA3); bộ ca độc lập đóng băng và băm.

1. **Đo nền trên mã hiện tại** (luật trước và sau khi đo, `docs/PROMPT-BUILD-EAIS.md` §3): bộ kiểm bảo toàn không import bộ nạp cộng bộ ca độc lập (người không đọc mã soạn, đóng băng, băm). Ghi nền đỏ theo nhóm. Đỏ dự kiến: A (HT-9), B (HT-1, 2, 3), C (HT-5), E (HT-4), F (chưa có quyền), G (HT-10), H, I (HT-8); nhóm D có thể xanh phần lớn vì D-OP-14 [SUY, chưa đo bằng bộ độc lập].
2. **Kho bản ghi, sổ lần giao và kết cục** (D1): nền của mọi thứ sau; đột biến DB01, DB02, DB20, DB28.
3. **Chống trùng** TR1, TR2, TR3 (D5): đột biến DB05, DB06, DB07, DB16, DB26, DB27, DB29, DB30, DB33.
4. **Cờ loại** (D3, D4): đột biến DB03, DB04, DB17, DB19.
5. **Thời gian** (`occurred_on`): đột biến DB08.
6. **Quyền người gửi, danh bạ và kế thừa thực thể** (D6, D10): đột biến DB09, DB10, DB32.
7. **Phục hồi** (sửa tin, thu hồi, đảo thứ tự, dừng giữa lô): đột biến DB11, DB15, DB21, DB27, DB31, DB34 (bản đứng sau bản rời nhóm), DB35 (quyết định trên màn hình cũ), DB36 (bản bị bác theo bản được giữ không quay lại hàng chờ khi bản được giữ thôi được tính bằng một nguyên nhân bất kỳ, kể cả khi bản được giữ đang bị bác; phép kiểm tính chất sau mọi sự kiện, LC84 (ix)).
8. **Báo cáo xuất** (D7): đột biến DB14.
9. **Nhánh form / bot** (D8): đột biến DB25.
10. Chạy bộ ca độc lập **một lần** ở mã đóng băng; báo cáo theo GA1–GA11; không chỉnh lexicon theo bộ này sau khi chạy (`docs/PROMPT-BUILD-EAIS.md` §9.9). Đột biến DB12, DB13, DB18 (GA5: đọc số, không model; đọc số đã có từ D-OP-14), DB22, DB23 (GA6) và DB24 (GA10) chạy cùng bước này.
11. Ghi "PHASE 0A RA" (M2 ra khi GA1–GA6 đạt), kèm câu "trên dữ liệu giả lập, chưa có người nghiệp vụ duyệt".

Mỗi bước: test đỏ trước → sửa nhỏ nhất → đo → đột biến → soát độc lập. Quá ba vòng soát còn lỗi cùng họ thì khoá và ghi nợ (§9.6 của brief). Một dòng không đạt được chỉ bằng luật: ghi vào CHO để chủ dự án chọn, **không tự nới ngưỡng**.

## Không làm

- Không viết mã trước khi H4 chạy (D-RM-04).
- Không sửa `core/pipeline/`, `intelligence/`, `data/normalization/`, `data/sourceoftruth/query.go`.
- Không gọi model; không đọc ảnh (OCR là Phase 6).
- Không áp đính chính tự động; không xác thực người gửi; không connector Zalo.
- Không tạo thư mục mới; không dùng bộ ca độc lập để chỉnh luật.

## Câu mở và điểm chưa kiểm chứng

- Câu mở cho chủ dự án: `HOP-DONG.md` §15 (Q-0A-1…Q-0A-16). Chặn nhiều nhất: Q-0A-12 và Q-0A-15 (write-set, việc chéo làn), Q-0A-4, Q-0A-5, Q-0A-6, Q-0A-7 (chính sách), D-OP-17 (nghĩa của "k"). Q-0A-13 và Q-0A-14 có mặc định an toàn; Q-0A-16 có mặc định ghi nợ (hướng sai thứ hai chưa chạm được với pack hôm nay); ba câu này không chặn.
- Điểm chưa kiểm chứng: `HOP-DONG.md` §16 (U1–U15). Riêng ADR này: điểm nối với M1 (mã lô cấp mới so với mã lần nạp suy từ nội dung; quyền đổi trạng thái của kho bản ghi so với vai `eais_app` chỉ SELECT + INSERT) chưa chốt; chi phí người xác nhận và tỉ lệ `HELD` chưa đo; `Load` phối hợp chưa thử; khoá tuần tự hoá TR3 và sổ lần giao chưa đo thông lượng và dung lượng.

## Ghi chú soát đã xử lý

Mã ghi chú do người soát đặt; nội dung đầy đủ nằm ở báo cáo của người soát (phiên điều phối ghi vào `docs/bao-cao/` khi gộp). Bảng này chỉ cho biết ghi chú nói gì và đã sửa ở đâu. Mức: S2 nặng hơn S3, S3 nặng hơn S4.

| Mã | Mức | Ghi chú (một dòng) | Sửa ở |
|---|---|---|---|
| F01 | S3 | chỗ ở của `DUPLICATE`: kho bản ghi có khoá duy nhất theo `unit_key` nên bản trùng không thể là dòng thứ hai | `HOP-DONG.md` §3.8; ADR D1, mục G |
| F02 | S3 | kết cục TR3 phụ thuộc thứ tự đến, và hai tiến trình nạp song song cho cả hai bản `ACCEPTED` | `HOP-DONG.md` §5.6, §7.4, LC74, LC75; ADR D5, mục H |
| F03 | S3 | có dòng cổng không có đột biến nào, hai bảng §11.1 và §11.3 lệch nhau | `HOP-DONG.md` §11; `CONG-QUANIT.md` §5 |
| R2-01 | S2 | quyết định của người đưa bản `HELD` (lý do khác) thành `ACCEPTED` mà không chạy TR3 với bản đứng trước: cộng đôi im lặng | `HOP-DONG.md` §5.1, §5.6 mục 3, §6.2, LC77, LC78, DB29, DB30; ADR D5, mục I, nợ 17 |
| R2-02 | S3 | thứ tự đến vẫn đổi kết cục khi có thu hồi, sửa tin, bác (bản rời nhóm) và ở kế thừa thực thể, mà §7.4 và LC63 khẳng định bất biến | `HOP-DONG.md` §5.6 mục 4, §7.4, LK7, LC63, LC81, LC82, DB31, DB32, U10, U11, Q-0A-13, Q-0A-14; ADR D10, mục J, K, nợ 18, 19 |
| R2-03 | S3 | GA11 không có ca nào làm DB25 đỏ; phải rà DB22 đến DB28 xem mỗi đột biến có ca của đúng dòng đó | `HOP-DONG.md` LC76, §11.3 (cột ca làm đỏ), LC01, LC64; ADR Cổng đổi kiến trúc hàng 9 |
| R2-04 | S3 | đòi "cùng byte" nhưng `trace_id`, `batch_id`, `recorded_at` khác nhau mỗi lần chạy | `HOP-DONG.md` §3.7 (trường của hệ), BB5, §6.4, LC54, LC69, GA5, §13.2; ADR nợ 22 |
| R2-05 | S3 | sổ lần giao thiếu bảng cho sửa tin, thu hồi, dấu thu hồi, quyết định; TR1 với `recalled`; sửa A, B, A; thua đua khoá duy nhất | `HOP-DONG.md` §3.8 (bảng lần giao), §5.1 TR1, §6.2, §7.2, §7.3, §7.5, §9, LC59 đến LC62, LC65 |
| R2-06 | S3 | vân tay TR2 cần `sender_id`, `sent_at` mà dòng PDF và hàng bảng không có | `HOP-DONG.md` §5.1 TR2, LC79, LC80, DB33, U12; ADR D5, nợ 20 |
| R2-07 | S3 | phụ thuộc chéo làn chưa đóng: kho PG và luật nạp ngoài write-set làn 4, hai việc ở làn 0, "≈" ở làn 1 | `HOP-DONG.md` §1.4, §14.1 (bảng việc, làn, tệp; phương án lùi; mục 6), Q-0A-12, Q-0A-15; ADR bối cảnh 8, D2, "Nơi đặt mã", nợ 9, 21 |
| R2-08 | S4 | sót nhỏ: "mỗi đơn vị" thành "mỗi lần giao"; "thêm một bảng"; dòng "Đỏ dự kiến" của nhóm D; bước "Chứng minh" thiếu sáu đột biến; GA6 "Đúng" so với sổ hay bản ghi; số dòng của D-OP-17 ở CHO | `HOP-DONG.md` §1.2, GA1, §14.2, §3.8, §11.2, LK5, Q-0A-2; `CONG-QUANIT.md` câu 8; ADR A1, "Chứng minh" |
| R2-09 | S4 | rebase lên origin/main; câu nói ADR-0011 còn giữ số migration cũ trở thành sai sau khi gộp (cũng là ghi chú L3D-07 của làn M4) | ADR "Số ADR", "Nơi đặt mã" |
| R3-01 | S2 | bản `ACCEPTED` đứng ngay sau bản rời nhóm không được đánh giá lại ("Bản đang `ACCEPTED` không đổi"): a, m, c với `KEEP_BOTH` cho c rồi m rời nhóm thì a và c liền kề, cách ≤ W, cùng `ACCEPTED`, cặp (c, a) chưa ai xét: cộng đôi im lặng | `HOP-DONG.md` §5.6 mục 3 và 4, LC81 (iv), DB34, GA9; ADR D5, mục J, nợ 18 |
| R3-02 | S3 | chuỗi sửa tin từ hai lần trở lên: giao lại cả chuỗi không idempotent, và lần sửa cũ đến sau ghi đè lần sửa mới (so với phiên bản hiện hành, không có khoá thứ tự) | **ghi nợ**: `HOP-DONG.md` §5.4, §7.4 (e), LC23, U13; ADR nợ 23 |
| R3-03 | S3 | TR2 vế (b): hai hướng sai im lặng (xuất lại với số dòng hoặc chữ khác bị tính hai lần; hai tệp khác nguồn cùng ngày bị gộp oan), chỉ khai ở mục U, chưa có câu mở | câu mở `HOP-DONG.md` Q-0A-16, U12; ADR nợ 20 |
| R3-04 | S3 | `KEEP_BOTH` và `MARK_DUPLICATE` không gắn với cặp người đã thấy: bản kia bị thu hồi rồi `MARK_DUPLICATE` trên màn hình cũ thì cả hai rơi khỏi tổng và hàng chờ | `HOP-DONG.md` §6.2, §5.6 mục 3 và 4, LC83, DB35, GA9; ADR D5, mục I, nợ 17 |
| R3-05 | S4 | sáu điểm chưa định nghĩa nhỏ ((a) `sent_at` của phiên bản do sửa tin; (b) đích của `DISMISS`; (c) thu hồi, sửa tin thiếu `source_message_id`; (d) TR2 vế (a) cho bản ghi form; (e) "bản kia" của `KEEP_BOTH`; (f) `MARK_DUPLICATE` khi `duplicate_of` rỗng) và một lệch ở lời khai của tác giả vòng 3 (g) | (e), (f): `HOP-DONG.md` §6.2, §5.6 mục 4; (a)–(d): **ghi nợ**, U14, ADR nợ 24; (g): nằm ở báo cáo của tác giả vòng 3, không ở ba tệp, không sửa |
| R4-01 | S3 | bản `REJECTED` bởi `MARK_DUPLICATE` rơi khỏi tổng khi bản được giữ bị thu hồi hoặc sửa tin sau quyết định (U15, nợ 25): cả hai bản rời tổng, không bản nào ở hàng chờ | `HOP-DONG.md` §5.6 mục 4 (c), §3.1 (BB7), §6.2, §7.4 (c), §8, LC84, DB36, GA9, U15; ADR D5, mục J, nợ 25 |
| R4-02 | S4 | lệch chữ: ngoại lệ (e) của §7.4 (chuỗi sửa tin từ hai lần trở lên) chưa mang sang các câu phổ quát về nạp lại và hội tụ | `HOP-DONG.md` §7.1, §7.5, §9 (hai hàng), LC67, GA9; ADR mục G |
| R4-03 | S4 | `seen_duplicate_of`: lệnh thiếu trường chưa có ca và đột biến; "rỗng khác thiếu"; giá trị do người gửi lệnh cấp ghi vào audit chưa có giới hạn | **ghi nợ**: `HOP-DONG.md` U14; ADR nợ 24 |
| R4-04 | S4 | bản `HELD` `possible_duplicate` không còn bản đứng trước (`duplicate_of` rỗng) chỉ còn lối ra `KEEP_BOTH`, chiếm tỉ lệ đáng kể trong mô phỏng của người soát | **ghi nợ**: `HOP-DONG.md` U14; ADR nợ 24 |
| R5-01 | S3 | bản `REJECTED` bởi `MARK_DUPLICATE` rơi khỏi tổng khi bản được giữ của nó cũng đang `REJECTED` bởi `MARK_DUPLICATE` rồi bị thu hồi hoặc sửa tin (người `APPLY`): mục 4 (c) chỉ chạy khi bản được giữ "rời nhóm" từ `ACCEPTED` hoặc `HELD` (U15, nợ 25) | `HOP-DONG.md` §5.6 mục 4 (c), §3.1, BB7, §6.2, §7.4 (c), LC84 (vii), DB36, U15; ADR D5, mục J, nợ 25 |
| R5-02 | S4 | bằng chứng và câu "0 chuỗi rơi" chỉ đúng dưới giả định không viết ra (bản `REJECTED` bất động, phiên bản chờ của bản không còn ở nhóm bị bỏ im lặng) | `HOP-DONG.md` §5.6 ("Phương án đã bỏ ở vòng 5, vòng 6 và vòng 7", đo lại), U8, U15; ADR mục J, nợ 25 |
| R5-03 | S4 | §3.8 nói khác nhau cho bản `REJECTED` bởi `MARK_DUPLICATE`: hàng "chạm bản ghi đã có quyết định của người" (phiên bản chờ) và ví dụ "thu hồi bản ghi đã `REJECTED`" (tổ hợp không có hàng) | `HOP-DONG.md` §3.8 |
| R6-01 | S3 | bản `REJECTED` bởi `MARK_DUPLICATE` rơi khỏi tổng khi bản được giữ của nó thôi được tính bằng `APPLY` đính chính làm bản gốc `SUPERSEDED`, một nguyên nhân không có trong danh sách của (c); cùng họ với R3-04, R4-01, R5-01 (U15, nợ 25) | `HOP-DONG.md` §5.6 mục 4 (c) (sửa gốc: điều kiện theo sự kiện), §3.1, §3.8, BB7, §6.2, §7.4 (c), §8, LC15, LC84 (ca con (viii), (ix)), DB36, GA9, U15; ADR D5, mục J, nợ 25 |
| R6-02 | S4 | các câu khẳng định hoàn tất ("đã vá", "vá hết", "không bản nào rơi khỏi tổng") chỉ đúng cho thu hồi, sửa tin và `APPLY`; U8 chưa nêu các đường rời nhóm mà mô phỏng không có; thiếu phép kiểm kiểu tính chất ở GA9 (bốn vòng liền, mỗi vòng thêm một nguyên nhân) | `HOP-DONG.md` §5.6 mục 4, §7.4 (c), BB7, U8, U15, GA9; ADR dòng Soát, mục J, nợ 25 |
