# ADR-0012 — Phase 0a: thu báo cáo ngày của đội hiện trường (kho bản ghi báo cáo, cờ loại, chống trùng; chế độ B)

- **Ngày:** 22/10/2026.
- **Trạng thái:** **ĐỀ XUẤT, chưa áp mã.**
  - Mã Phase 0a (khối M2 của MVP) chỉ mở SAU khi chạy xong H4 (`docs/PROMPT-BUILD-EAIS.md` §9.2, làn 4: "0a (M2): sau khi H4 chạy xong"). H4 đã khoá (`15f04f9`) và CHƯA chạy (`docs/STATUS.md`).
  - Soát: **chưa soát độc lập**. Chưa có người nghiệp vụ duyệt (D-RM-03 = "chưa có").
- **Số ADR:** 0012. Số 0011 thuộc Phase 1 (`docs/adr/0011-phase1-hanh-dong-co-kiem-soat.md`). Số migration 0003 đã bị chiếm hai lần: nhánh làn 0 (chưa gộp) có `0003_kho_vet_ben.sql`, ADR-0011 dự kiến `0003_phieu_chenh_lech.sql`. Phase 0a không được lấy số 0003 (xem "Nơi đặt mã").
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
5. **Làn 0** (nhánh `lan0/ung-dung-va-vet`, chưa gộp tại thời điểm viết) [CODE, chưa lên main]: `POST /v1/ingest` nạp lại **thư mục dữ liệu đã cấu hình** của tenant, thân rỗng, lưu cách ly (khoá, nguồn, lý do, chi tiết) rồi thay trọn dữ liệu (`cmd/eais/server.go`, `cmd/eais/app.go`). Nghĩa: đã có nền lưu cách ly, nhưng chưa có đường nạp từng đơn vị, chưa có kết cục chờ xác nhận hay trùng, chưa có người nhận. Điểm nối cần thống nhất với làn 0 (U2).
6. **Quyết định của chủ dự án** [REPO]: D-RM-04 = A (mã Giai đoạn II mở sau H4); D-RM-03 = chưa có người nghiệp vụ; D-AI-00 (không lời gọi model trả tiền trước CỔNG TRƯỚC AI); D-MVP-00 (MVP gồm M2); D-OP-14 = a (đã vào main); D-OP-17 còn mở.
7. **Cổng QuânIT (trên giấy)**: trượt câu 1 và 8 (không có người dùng thật, không có người ký), ◐ câu 3 và 9. Không chặn việc viết giấy; giới hạn kết luận: "trên dữ liệu giả lập, chưa có người nghiệp vụ duyệt".
8. **Lệch write-set** [REPO]: `docs/PROMPT-BUILD-EAIS.md` §9.2 ghi làn 4 làm `execution/`, migration mới, pack; mã 0a lại nằm ở tầng nạp `data/ingestion/` và `platform/pgstore/` (`docs/01-architecture.md` §3).

## Quyết định

Mỗi quyết định có chi tiết và ca kiểm ở `HOP-DONG.md`; ADR chỉ ghi lựa chọn và lý do.

| # | Quyết định | Chi tiết |
|---|---|---|
| **D1** | **Kho bản ghi báo cáo**, chỉ thêm và đổi trạng thái, lưu bền, RLS FORCE. Mỗi **đơn vị đầu vào** có đúng một kết cục có lý do: `ACCEPTED`, `HELD`, `QUARANTINED`, `DUPLICATE`. `facts` chỉ là **phép chiếu**: chỉ bản ghi `ACCEPTED` loại `OCCURRED` thành fact, trong cùng giao dịch với kết cục. `aggregate` **không đổi**. | HOP-DONG §3.1, §3.6 |
| **D2** | **Nơi đặt mã:** mở rộng các gói đã có (`data/ingestion/`, `platform/pgstore/`), tệp mới; không tạo thư mục mới (bảng dưới). | bảng "Nơi đặt mã" |
| **D3** | **Cờ loại là hai trục** `kind` ∈ {`OCCURRED`, `PLAN`, `CORRECTION`} × `approx`; bốn cờ của brief suy ra từ hai trục. Gán bằng luật tất định từ lexicon của pack; luật không chắc thì `HELD`; **bảo thủ cả hai chiều**. | §3.3 |
| **D4** | **Đính chính không tự áp:** luôn `HELD`, người quyết `APPLY` hoặc `REJECT`. | §3.3, §6.2 |
| **D5** | **Chống trùng ba mức:** khoá nguồn (TR1), vân tay (TR2), nghi trùng trong cửa sổ W (TR3, chờ người quyết). **Không** khử theo giá trị, **không** "bản mới nhất thắng". | §5 |
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
| `platform/pgstore/` | có | tệp mới: kho bản ghi báo cáo (ghi từng đơn vị trong một giao dịch, quyết định, báo cáo xuất). Cùng khu vực làn 0 đang ghi bảng vết và `SaveQuarantine`: tách tệp, thống nhất bảng (U2) |
| `db/migrations/` | có | migration mới, **số kế tiếp còn trống lúc mở mã**; làn 0 (`0003_kho_vet_ben.sql`, nhánh chưa gộp) và ADR-0011 (`0003_phieu_chenh_lech.sql`, dự kiến) đã dùng số 0003, ba bên không được cùng một số |
| `tenants/construction/pack/` | có | `lexicon.yaml`: từ khoá đính chính, kế hoạch (cờ "≈" là việc của làn 1); `policy.yaml`: capability `reports.*`, vai `site_supervisor` (Q-0A-7); `sources.yaml`: loại nguồn có cấu trúc cho báo cáo của đội, danh sách bỏ qua dòng PDF. **Làn 1 giữ pack**: phối hợp khi mở mã |
| dữ liệu tenant | (cùng chỗ `entities.json`) | danh bạ người gửi (`HOP-DONG.md` §4.3) |
| `core/contracts/` | có | chỉ kiểu dùng chung nếu nhiều gói cần (kết cục, loại); không bắt buộc |
| `tenants/construction/win/`, `scripts/dot-bien/` | có | làn phép đo: bộ kiểm bảo toàn không import bộ nạp, chạy bộ ca độc lập, đột biến DB01–DB21 |
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
| 9 | Test? | 73 lớp ca âm (`HOP-DONG.md` §10), 21 đột biến (§11.3) |
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

## Chứng minh (thứ tự)

Điều kiện mở mã: H4 chạy xong; D-OP-14 đã vào main (có); hợp đồng đơn vị nạp của làn 0 chốt; write-set làn 4 mở rộng; bộ ca độc lập đóng băng và băm.

1. **Đo nền trên mã hiện tại** (luật trước và sau khi đo, `docs/PROMPT-BUILD-EAIS.md` §3): bộ kiểm bảo toàn không import bộ nạp cộng bộ ca độc lập (người không đọc mã soạn, đóng băng, băm). Ghi nền đỏ theo nhóm. Đỏ dự kiến: A (HT-9), B (HT-1, 2, 3), C (HT-5), E (HT-4), F (chưa có quyền), G (HT-10), H, I (HT-8); nhóm D có thể xanh phần lớn vì D-OP-14 [SUY, chưa đo bằng bộ độc lập].
2. **Kho bản ghi và kết cục** (D1): nền của mọi thứ sau.
3. **Chống trùng** TR1, TR2, TR3 (D5): đột biến DB05, DB06, DB07, DB16.
4. **Cờ loại** (D3, D4): đột biến DB03, DB04, DB17, DB19.
5. **Thời gian** (`occurred_on`): đột biến DB08.
6. **Quyền người gửi và danh bạ** (D6): đột biến DB09, DB10.
7. **Phục hồi** (sửa tin, thu hồi, đảo thứ tự, dừng giữa lô): đột biến DB11, DB15, DB21.
8. **Báo cáo xuất** (D7): đột biến DB14.
9. **Nhánh form / bot** (D8).
10. Chạy bộ ca độc lập **một lần** ở mã đóng băng; báo cáo theo GA1–GA11; không chỉnh lexicon theo bộ này sau khi chạy (`docs/PROMPT-BUILD-EAIS.md` §9.9).
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
- Điểm chưa kiểm chứng: `HOP-DONG.md` §16 (U1–U8). Riêng ADR này: nhánh làn 0 chưa gộp nên điểm nối `POST /v1/ingest` có thể đổi; chi phí người xác nhận và tỉ lệ `HELD` chưa đo; `Load` phối hợp chưa thử.
