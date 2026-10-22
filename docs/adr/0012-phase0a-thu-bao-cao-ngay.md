# ADR-0012 — Phase 0a: thu báo cáo ngày của đội hiện trường (kho bản ghi báo cáo, cờ loại, chống trùng; chế độ B)

- **Ngày:** 22/10/2026.
- **Trạng thái:** **ĐỀ XUẤT, chưa áp mã.**
  - Mã Phase 0a (khối M2 của MVP) chỉ mở SAU khi chạy xong H4 (`docs/PROMPT-BUILD-EAIS.md` §9.2, làn 4: "0a (M2): sau khi H4 chạy xong"). H4 đã khoá (`15f04f9`) và đã chạy MỘT lần ngày 22/10 (`docs/STATUS.md`, dòng "H4, 22/10 14:59"): điều kiện này đã đạt.
  - Soát: vòng 1 (người soát ≠ người viết) kết luận **ĐẠT CÓ GHI CHÚ**, ba ghi chú S3: F01, F02, F03 (`docs/bao-cao/2026-10-22-ca.md`) [REPO]. Bản này sửa ba ghi chú đó (quyết định D1 và D5, mục G và H ở "Phương án đã cân nhắc") và khớp M1 đã lên main; **chưa soát lại sau khi sửa**. Chưa có người nghiệp vụ duyệt (D-RM-03 = "chưa có").
- **Số ADR:** 0012. Số 0011 thuộc Phase 1 (`docs/adr/0011-phase1-hanh-dong-co-kiem-soat.md`). Migration của Phase 0a lấy **số kế tiếp sau migration cuối của main lúc mở mã**. Tại `f28ce06` migration cuối là `0003_kho_vet_ben.sql` (M1, đã lên main) nên số kế tiếp là 0004; ADR-0011 tại mốc đó vẫn dự kiến `0003_phieu_chenh_lech.sql`, số đã bị M1 chiếm: hai phase không được cùng một số, bên gộp sau lấy số kế tiếp (xem "Nơi đặt mã").
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
8. **Lệch write-set** [REPO]: `docs/PROMPT-BUILD-EAIS.md` §9.2 ghi làn 4 làm `execution/`, migration mới, pack; mã 0a lại nằm ở tầng nạp `data/ingestion/` và `platform/pgstore/` (`docs/01-architecture.md` §3).

## Quyết định

Mỗi quyết định có chi tiết và ca kiểm ở `HOP-DONG.md`; ADR chỉ ghi lựa chọn và lý do.

| # | Quyết định | Chi tiết |
|---|---|---|
| **D1** | **Kho bản ghi báo cáo**, chỉ thêm và đổi trạng thái, lưu bền, RLS FORCE, khoá duy nhất `(tenant, unit_key)`: mỗi đơn vị một bản ghi (`ACCEPTED`, `HELD`, `QUARANTINED`). Cộng **sổ lần giao** chỉ thêm: mỗi **lần giao** của một đơn vị có đúng một dòng với một kết cục có lý do (`ACCEPTED`, `HELD`, `QUARANTINED`, `DUPLICATE`); `DUPLICATE` chỉ có ở sổ, trỏ về bản đã có, không tạo bản ghi (mục G). `facts` chỉ là **phép chiếu**: chỉ bản ghi `ACCEPTED` loại `OCCURRED` thành fact, trong cùng giao dịch với kết cục. `aggregate` **không đổi**. | HOP-DONG §3.1, §3.6, §3.8 |
| **D2** | **Nơi đặt mã:** mở rộng các gói đã có (`data/ingestion/`, `platform/pgstore/`), tệp mới; không tạo thư mục mới (bảng dưới). | bảng "Nơi đặt mã" |
| **D3** | **Cờ loại là hai trục** `kind` ∈ {`OCCURRED`, `PLAN`, `CORRECTION`} × `approx`; bốn cờ của brief suy ra từ hai trục. Gán bằng luật tất định từ lexicon của pack; luật không chắc thì `HELD`; **bảo thủ cả hai chiều**. | §3.3 |
| **D4** | **Đính chính không tự áp:** luôn `HELD`, người quyết `APPLY` hoặc `REJECT`. | §3.3, §6.2 |
| **D5** | **Chống trùng ba mức:** khoá nguồn (TR1), vân tay (TR2), nghi trùng trong cửa sổ W (TR3, chờ người quyết). TR1 và TR2 cho `DUPLICATE` ở sổ lần giao; TR3 đánh giá theo **thứ tự chuẩn** `(sent_at, unit_key)` chứ không theo thứ tự đến, đánh giá lại (chỉ siết) khi bản gửi sớm đến muộn, và chạy dưới **khoá tư vấn** theo (tenant, thực thể, ngày xảy ra) (mục H). **Không** khử theo giá trị, **không** "bản mới nhất thắng". | §5, §5.6 |
| **D6** | **Quyền người gửi theo máy:** danh bạ người gửi là dữ liệu tenant; vai và capability (`reports.submit`, `reports.review`, `reports.decide`) ở pack; đóng mặc định; tenant chỉ từ danh tính. | §4 |
| **D7** | **Cách ly và chờ xác nhận lưu bền, có mã lý do trong danh sách đóng, có người nhận; báo cáo xuất tất định.** | §3.4, §6 |
| **D8** | **Nhánh form / bot dùng chung hợp đồng đầu ra** với đường chat; đo song song bằng bản chiếu cặp; chọn kênh theo số đo và chi phí đổi thói quen (không phải việc của ADR). | §13 |
| **D9** | **Không** model, **không** interface chung, **không** thư mục theo tên kiến trúc đích, **không** đụng S4 hay `aggregate`. | dưới |

## Nơi đặt mã

Theo `docs/01-architecture.md` §3 và `CLAUDE.md` §25: thư mục chỉ được tạo khi có mã thật cần boundary đó. Bảng này là kế hoạch cho lúc mã được mở; **hiện chưa tạo gì**.

| Đường dẫn | Hiện có? | Phase 0a làm gì |
|---|---|---|
| `data/ingestion/` | có | tệp mới trong gói: phong bì tin, phân loại `kind` và `approx`, chống trùng TR1–TR3, quyền người gửi, kết cục. Hợp đồng `Ingest` hiện có cho nguồn có cấu trúc khác không đổi |
| `data/ingestion/bridge.go` | có | `Facts()` chỉ chiếu bản ghi `ACCEPTED` loại `OCCURRED`; điền `ObservedAt` và `ReportedBy`; sửa nhỏ |
| `platform/pgstore/` | có | tệp mới: kho bản ghi báo cáo và sổ lần giao (ghi từng đơn vị trong một giao dịch, quyết định, báo cáo xuất, khoá tuần tự hoá TR3). M1 đã có `vet.go` (`SaveQuarantine`, bảng vết) và khoá tư vấn của `Load`: tách tệp, thống nhất bảng (U2) |
| `db/migrations/` | có | migration mới, **số kế tiếp sau migration cuối của main lúc mở mã** (tại `f28ce06`: 0004, vì `0003_kho_vet_ben.sql` của M1 đã lên main); ADR-0011 dự kiến `0003_phieu_chenh_lech.sql`, số đã bị chiếm: hai phase không được cùng một số |
| `tenants/construction/pack/` | có | `lexicon.yaml`: từ khoá đính chính, kế hoạch (cờ "≈" là việc của làn 1); `policy.yaml`: capability `reports.*`, vai `site_supervisor` (Q-0A-7); `sources.yaml`: loại nguồn có cấu trúc cho báo cáo của đội, danh sách bỏ qua dòng PDF. **Làn 1 giữ pack**: phối hợp khi mở mã |
| dữ liệu tenant | (cùng chỗ `entities.json`) | danh bạ người gửi (`HOP-DONG.md` §4.3) |
| `core/contracts/` | có | chỉ kiểu dùng chung nếu nhiều gói cần (kết cục, loại); không bắt buộc |
| `tenants/construction/win/`, `scripts/dot-bien/` | có | làn phép đo: bộ kiểm bảo toàn không import bộ nạp, chạy bộ ca độc lập, đột biến DB01–DB28 |
| `execution/`, `governance/`, `cmd/`, `domain/` | chưa | **không tạo**: sự kiện đầy đủ là Phase 3; xác thực và `cmd` là Phase 5 hoặc làn 0; connector là Phase 8 |

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
| 9 | Test? | 75 lớp ca âm (`HOP-DONG.md` §10), 28 đột biến (§11.3), mỗi dòng cổng có ít nhất một |
| 10 | Metric? | 11 dòng cổng GA1–GA11 (§11) |
| 11 | ADR? | đây |

## Phương án đã cân nhắc

### A. Chỗ lưu và quan hệ với `facts`

| Phương án | Ưu | Nhược |
|---|---|---|
| **A1 (chọn):** kho bản ghi riêng cộng phép chiếu sang `facts` | khớp bốn tầng OBSERVATION → FACT (`docs/01-architecture.md` §5.0); lọc ở biên cầu nối nên **không đổi `aggregate`** (lõi hỏi–đáp của làn 1); `HELD`, `QUARANTINED`, `DUPLICATE` có chỗ ở | thêm một bảng và một đường chiếu; `Load` thay trọn `facts` (`load.go:27`) nên phải phối hợp (U2) |
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
| **Sổ lần giao riêng, chỉ thêm (chọn):** mỗi lần giao một dòng; kho bản ghi chỉ có `ACCEPTED`, `HELD`, `QUARANTINED` | khoá duy nhất `(tenant, unit_key)` chỉ áp cho kho bản ghi nên không đụng bản trùng; nạp lại cho cùng trạng thái cuối (sổ thêm dòng, kho không đổi); BB1 kiểm được từ kho trên cả lần giao lại; cùng mẫu `quarantine_records` của M1 (khoá theo mã lần nạp, chỉ thêm) | thêm một bảng và một dòng ghi cho mỗi đơn vị mỗi lần giao (dung lượng chưa đo) |
| `DUPLICATE` là dòng thứ hai trong kho bản ghi | một bảng | với TR1, bản trùng cùng `unit_key` với bản gốc nên vi phạm khoá duy nhất; đổi khoá thì mỗi lần nạp lại thêm dòng và "trạng thái cuối giống hệt" sai |
| `DUPLICATE` chỉ ở vết | không thêm bảng | bộ kiểm bảo toàn (không import bộ nạp) phải tin vết, mà vết là đối tượng của GA10; lần giao trùng rơi im lặng không bị bắt; vết của đơn vị nạp chưa có chỗ lưu (M1 chỉ lưu vết lượt hỏi) |

### H. Thứ tự và đua của TR3 (ghi chú soát F02; quyết định D5)

| Phương án | Ưu | Nhược |
|---|---|---|
| **Thứ tự chuẩn `(sent_at, unit_key)`, đánh giá lại khi bản gửi sớm đến muộn (chỉ siết), cộng khoá tư vấn theo (tenant, thực thể, ngày) (chọn)** | kết cục là hàm của tập bản: giữ nguyên "đảo thứ tự không đổi tổng, cờ, tập `HELD`" (§7.4, LC63) và đúng cả khi hai tiến trình chạy song song; khoá cùng kiểu với `Load` của M1 | một bản đã `ACCEPTED` có thể thành `HELD` khi bản sớm hơn đến muộn; khoá làm các giao dịch cùng (thực thể, ngày) xếp hàng |
| Giữ "so với bản đã có lúc đến", hạ LC63 và §7.4 xuống "cùng tổng và cùng số `HELD`" | không đánh giá lại | chuỗi ba bản cho số `HELD` khác nhau theo thứ tự đến (`HOP-DONG.md` §5.6, tính tay), nên điều hạ xuống vẫn sai; không giải đua |
| Giữ như cũ, chỉ thêm khoá | nhỏ nhất | LC63 và §7.4 vẫn mâu thuẫn với TR3: đúng điều ghi chú F02 nêu |
| Mức cô lập SERIALIZABLE kèm thử lại | không tự quản khoá | mã thử lại ở mọi đường ghi; báo lỗi tuần tự hoá oan; khoá tư vấn đã có tiền lệ ở `Load` |

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
9. **Write-set của làn 4** phải mở rộng (Q-0A-12).
10. **Pack đổi** (`lexicon`, `policy`, `sources`) làm `PackVersion` đổi: các chỗ ghim phiên bản (bằng chứng đã đóng băng) cần kiểm khi mở mã.
11. **Câu trả lời cũ không tự cập nhật** khi đính chính được áp: thuộc Phase 3.
12. **Kết luận chỉ "trên dữ liệu giả lập, chưa có người nghiệp vụ duyệt"**: câu 1 và 8 của cổng QuânIT còn ✗.
13. **Đánh giá lại TR3 có thể chuyển một bản đã `ACCEPTED` sang `HELD`** khi bản gửi sớm hơn đến muộn (`HOP-DONG.md` §5.6): tổng đọc loại đúng một giá trị trùng, nhưng bản được tính đổi; tần suất bản đến muộn chưa đo (U4).
14. **Khoá tư vấn theo (tenant, thực thể, ngày xảy ra)** làm các giao dịch cùng (thực thể, ngày) xếp hàng; thông lượng chưa đo (U3).
15. **Sổ lần giao thêm một dòng cho mỗi đơn vị mỗi lần giao**, kể cả `ACCEPTED`; dung lượng chưa đo, chính sách lưu giữ chưa có (Phase 5).
16. **`id_conflict` phụ thuộc thứ tự đến** (ngoại lệ có tên của §7.4 và LC63): luật đối xứng chưa chọn (U9).

## Chứng minh (thứ tự)

Điều kiện mở mã: H4 chạy xong; D-OP-14 đã vào main (có); hợp đồng đơn vị nạp của làn 0 chốt; write-set làn 4 mở rộng; bộ ca độc lập đóng băng và băm.

1. **Đo nền trên mã hiện tại** (luật trước và sau khi đo, `docs/PROMPT-BUILD-EAIS.md` §3): bộ kiểm bảo toàn không import bộ nạp cộng bộ ca độc lập (người không đọc mã soạn, đóng băng, băm). Ghi nền đỏ theo nhóm. Đỏ dự kiến: A (HT-9), B (HT-1, 2, 3), C (HT-5), E (HT-4), F (chưa có quyền), G (HT-10), H, I (HT-8); nhóm D có thể xanh phần lớn vì D-OP-14 [SUY, chưa đo bằng bộ độc lập].
2. **Kho bản ghi, sổ lần giao và kết cục** (D1): nền của mọi thứ sau; đột biến DB28.
3. **Chống trùng** TR1, TR2, TR3 (D5): đột biến DB05, DB06, DB07, DB16, DB26, DB27.
4. **Cờ loại** (D3, D4): đột biến DB03, DB04, DB17, DB19.
5. **Thời gian** (`occurred_on`): đột biến DB08.
6. **Quyền người gửi và danh bạ** (D6): đột biến DB09, DB10.
7. **Phục hồi** (sửa tin, thu hồi, đảo thứ tự, dừng giữa lô): đột biến DB11, DB15, DB21, DB27.
8. **Báo cáo xuất** (D7): đột biến DB14.
9. **Nhánh form / bot** (D8): đột biến DB25.
10. Chạy bộ ca độc lập **một lần** ở mã đóng băng; báo cáo theo GA1–GA11; không chỉnh lexicon theo bộ này sau khi chạy (`docs/PROMPT-BUILD-EAIS.md` §9.9). Đột biến DB22, DB23 (GA6) và DB24 (GA10) chạy cùng bước này.
11. Ghi "PHASE 0A RA" (M2 ra khi GA1–GA6 đạt), kèm câu "trên dữ liệu giả lập, chưa có người nghiệp vụ duyệt".

Mỗi bước: test đỏ trước → sửa nhỏ nhất → đo → đột biến → soát độc lập. Quá ba vòng soát còn lỗi cùng họ thì khoá và ghi nợ (§9.6 của brief). Một dòng không đạt được chỉ bằng luật: ghi vào CHO để chủ dự án chọn, **không tự nới ngưỡng**.

## Không làm

- Không viết mã trước khi H4 chạy (D-RM-04).
- Không sửa `core/pipeline/`, `intelligence/`, `data/normalization/`, `data/sourceoftruth/query.go`.
- Không gọi model; không đọc ảnh (OCR là Phase 6).
- Không áp đính chính tự động; không xác thực người gửi; không connector Zalo.
- Không tạo thư mục mới; không dùng bộ ca độc lập để chỉnh luật.

## Câu mở và điểm chưa kiểm chứng

- Câu mở cho chủ dự án: `HOP-DONG.md` §15 (Q-0A-1…Q-0A-12). Chặn nhiều nhất: Q-0A-12 (write-set làn 4), Q-0A-4, Q-0A-5, Q-0A-6, Q-0A-7 (chính sách), D-OP-17 (nghĩa của "k").
- Điểm chưa kiểm chứng: `HOP-DONG.md` §16 (U1–U9). Riêng ADR này: điểm nối với M1 (mã lô cấp mới so với mã lần nạp suy từ nội dung; quyền đổi trạng thái của kho bản ghi so với vai `eais_app` chỉ SELECT + INSERT) chưa chốt; chi phí người xác nhận và tỉ lệ `HELD` chưa đo; `Load` phối hợp chưa thử; khoá tuần tự hoá TR3 và sổ lần giao chưa đo thông lượng và dung lượng.
