# Hợp đồng việc — Phase 0a: thu báo cáo ngày của đội hiện trường (khối M2)

> **Trạng thái: GIẤY, chưa có mã.**
> - Mã Phase 0a chỉ mở SAU khi chạy xong H4 (`docs/PROMPT-BUILD-EAIS.md` §9.2, làn 4: "0a (M2): sau khi H4 chạy xong"). H4 đã khoá (`15f04f9`) và CHƯA chạy (`docs/STATUS.md`) [REPO].
> - Soát: **chưa soát độc lập** (người viết ≠ người soát; việc của phiên điều phối). Chưa có người nghiệp vụ duyệt (D-RM-03 = "Chưa có").
>
> **Viết cho:**
> - chủ dự án: duyệt hoặc sửa từng quyết định, và trả lời các câu ở §15;
> - phiên BUILD: làm theo tệp này và `docs/adr/0012-phase0a-thu-bao-cao-ngay.md` khi mã được mở;
> - người soạn bộ ca độc lập (không đọc mã): chỉ nhận **một phần** của tệp này, xem §12.
>
> **Ngày:** 22/10/2026. **Mốc đọc mã:** `2b80a38` (origin/main). Tra một dòng trích bằng `git show 2b80a38:<tệp>`. Bản nháp đầu đọc ở `7d72e33`; từ đó tới `2b80a38` mã đổi ở `data/normalization/numwords_vi.go` (D-OP-14 đã sửa bộ đọc số: HT-6 và HT-7 hết), `core/pipeline`, `core/tenant/pack.go`, `intelligence/` và ba tệp yaml của pack (`intents`, `lexicon`, `templates`); tầng nạp không đổi (§1.3.1). Làn 1 còn sửa "≈" (HT-1) và có thể đổi `data/normalization/` thêm: đo lại trước khi dựa vào.
>
> **Nhãn bằng chứng:**
> - [ĐO]: người viết tự chạy lệnh, đo ở `7d72e33` rồi đo lại ở `2b80a38` (§1.3.3 ghi cách làm; test thăm dò tạm, không commit, đã xoá);
> - [REPO]: tài liệu repo tự ghi, chưa chạy lại;
> - [CODE]: đọc thấy trong mã, `tệp:dòng` tại `2b80a38`;
> - [SUY]: suy luận của người viết;
> - [NGOÀI]: nguồn công khai ngoài repo, đọc ngày 22/10/2026 (`CONG-QUANIT.md` §4);
> - [ĐỀ XUẤT]: chưa quyết, đảo được. Mọi tên kiểu, tên trường, mã lý do, ngưỡng và số trong tệp này là [ĐỀ XUẤT], trừ chỗ ghi nguồn khác.
> - "chưa đo": chưa chạy lệnh nào.
>
> **Dữ liệu:** mọi tên máy, dự án, người gửi, con số là giả lập (tenant `abc-infra` hư cấu). Mọi kết luận: **trên dữ liệu giả lập, chưa có người nghiệp vụ duyệt**.
>
> **Mã dùng trong tệp** (chọn để không trùng M1–M5 của MVP, K1–K5 của audit, B1 / B2 của baseline G7, C0–C7 của pipeline lệnh): HT = hiện trạng · BB = bất biến · LK, LT = luật loại tin, thời gian · TR = mức chống trùng · CL = bất biến cách ly · LC = lớp ca âm · GA = dòng cổng · DB = đột biến · KV = kênh vào · Q-0A = câu mở · U = chưa kiểm chứng.

---

## 0. Tóm tắt

- **Việc:** đội hiện trường báo cáo lượng dầu của từng máy trong ngày bằng tin nhắn nhóm hoặc tệp. EAIS biến **mỗi đơn vị đầu vào** thành đúng một kết cục có lý do: nhận (`ACCEPTED`), chờ xác nhận (`HELD`), cách ly (`QUARANTINED`), trùng (`DUPLICATE`).
- **Vào:** bốn kênh (§2.1): tệp tin nhắn đã xuất, tệp báo cáo, bản ghi từ form / bot, ảnh và tệp lạ (cách ly, chưa đọc).
- **Ra:** bản ghi báo cáo mang hai trục cờ: loại (`OCCURRED`, `PLAN`, `CORRECTION`) và `approx`. Từ hai trục suy ra bốn cờ của brief: đã xảy ra, ước lượng, đính chính, kế hoạch (§3.3).
- **Năm bảo đảm chính:** (1) không đơn vị nào rơi im lặng; (2) cờ loại đúng; (3) không cộng đôi và không gộp oan; (4) đọc số tất định; (5) cách ly lưu bền, có lý do và người nhận, xuất được báo cáo.
- **Thêm:** ngày xảy ra đọc từ chữ (§3.5); quyền người gửi theo máy (§4); phục hồi: gửi lại, sửa tin, thu hồi, đảo thứ tự, dừng giữa lô (§7).
- **Phép chiếu sang kho sự thật:** chỉ bản ghi `ACCEPTED` loại `OCCURRED` thành fact; kho đọc `aggregate` không đổi (§3.6).
- **Cổng ra:** 11 dòng GA1–GA11 (§11). M2 ra khi GA1–GA6 đạt; "PHASE 0A RA" đòi thêm GA7–GA11. Ngưỡng độ chính xác (GA6) là **ĐỀ XUẤT** và chưa có người ký.
- **Nhánh form / bot** (§13): cùng hợp đồng đầu ra, đo song song với đường chat bằng bản chiếu cặp.
- **Chưa biết:** §16. Câu gửi chủ dự án: §15.

---

## 1. Việc và phạm vi

### 1.1 Việc

> Với mỗi tin nhắn, dòng báo cáo hoặc hàng bảng mà đội hiện trường gửi: EAIS đọc, chuẩn hoá, nhận dạng máy, phân loại tin (đã xảy ra, ước lượng, đính chính, kế hoạch), kiểm miền, chống trùng, kiểm quyền người gửi; rồi ghi **đúng một** kết cục có lý do. Bản ghi nhận được đi vào kho sự thật; mọi thứ khác nằm ở hàng chờ hoặc cách ly, có người nhận, có thể xuất báo cáo.

- **Nguồn của việc:** `docs/ROADMAP-DAY-DU.md` §5 (luồng L1), §5b (khối M2), §6 (dòng Phase 0a); `docs/PROMPT-BUILD-EAIS.md` §9.1–§9.3.
- **Chế độ B** (EAIS thu, chuẩn hoá, đối soát, lưu trữ): doanh nghiệp chỉ có Zalo, bảng tính, giấy, ảnh (`docs/tenants/construction/01-business-map.md:139-144`) [REPO].
- **Chỉ số:** `reported_fuel`, "Dầu đội báo", đơn vị chuẩn L, miền 0–20.000, dung sai 0, hai loại nguồn: `site_report_form` hạng 1, `chat_message` hạng 2 (`tenants/construction/pack/metrics.yaml:9-18`) [CODE].

### 1.2 Làm và không làm

| Làm | Không làm (và ở đâu) |
|---|---|
| nạp tin nhắn nhóm đã xuất (JSON Lines) và tệp báo cáo (PDF văn bản, xlsx, csv, json) cho `reported_fuel` | connector Zalo trực tiếp, gửi tin trả lời cho người gửi: Phase 8 |
| mỗi đơn vị đúng một kết cục có lý do; kho bản ghi lưu bền | đọc ảnh, OCR: Phase 6 (ảnh → cách ly `unsupported_media`) |
| hai trục cờ `kind` × `approx`, gán bằng luật tất định từ lexicon của pack | mọi lời gọi model (D-AI-00; `CLAUDE.md` §19) |
| ngày xảy ra đọc từ chữ khi có (§3.5) | áp đính chính tự động bằng mô hình sự kiện, tin đến muộn so với kỳ đã đóng: Phase 3 |
| chống trùng ba mức; không khử trùng theo giá trị (§5) | tạo phiếu khi có chênh lệch: Phase 1 (M4) |
| quyền người gửi theo máy; tenant từ danh tính (§4) | workflow duyệt báo cáo ngày, "đủ năm khe", nhắc hạn (BR-RPT-01…04, `01-business-map.md:404-407`): Phase 2 |
| quyết định của người (xác nhận, bác, sửa) có audit (§6.2) | bốn khe khác của báo cáo ngày: tiến độ, giờ máy, nhân công, thời tiết |
| xuất báo cáo cách ly và hàng chờ (§6.4) | xác thực người gửi (ai thật sự là mã `S01`): Phase 5, 8 (§4.4) |
| phục hồi: gửi lại, sửa tin, thu hồi, đảo thứ tự, dừng giữa lô (§7) | bộ đọc số (D-OP-14, đã vào main) và nhận "≈" (HT-1): việc của làn 1; hợp đồng này chỉ đặt điều kiện cho cổng GA5 |
| nhánh form / bot dùng chung hợp đồng đầu ra (§13) | sửa luồng hỏi–đáp S0–S10; cổng đọc lại đi qua kho sự thật, không qua S4 |

### 1.3 Hiện trạng ở `2b80a38` (không giao cho người soạn bộ ca độc lập)

#### 1.3.1 Mã hiện có

| Thành phần | Hiện có (`tệp:dòng`, `2b80a38`) [CODE] | Thiếu so với hợp đồng |
|---|---|---|
| đọc theo loại | 5 định dạng: csv, xlsx, json, jsonl, pdf_lines (`data/ingestion/readers.go:37-52`, `sources.go:60`); cấu hình sai bị chặn lúc khởi động (`sources.go:62-130`) | ảnh, tệp lạ |
| loại nguồn của tin chat và PDF | `chat_message`: chữ tự do, đơn vị mặc định L, kế thừa thực thể (`tenants/construction/pack/sources.yaml:46-52`); `site_report_form`: PDF (`sources.yaml:55-60`) | loại nguồn cho bảng do đội gửi; nguồn form / bot |
| quan sát | `Observation`: khoá, tenant, thực thể, chỉ số, nguồn, ngày, giá trị (kèm `Approx`), nguyên văn, ghi chú (`ingest.go:19-31`) | giờ gửi, người gửi, loại, kết cục, phiên bản, tham chiếu đính chính |
| cách ly | `Quarantine{Key, Tenant, Source, Reason, Detail}`; `Result` trả về trong RAM (`ingest.go:33-42`, `:174-176`); chỉ harness đo dùng (`tenants/construction/win/winrun/winrun.go:145`, `:315`) | lưu bền, người nhận, trạng thái, xuất báo cáo |
| mã lý do | sáu mã: `missing_value`, `unknown_entity`, `ambiguous_entity`, `out_of_range`, `unit_dimension`, `unparseable` (`sources.go:18-26`) | các mã mới ở §3.4 |
| ngày | tin chat: `sent_at` cắt về 00:00 ngày gửi theo múi giờ tenant (`ingest.go:244-250`); PDF: ngày ở tên tệp (`:240-243`); nguồn có cấu trúc: cột ngày (`:208`) | ngày viết trong chữ không được dùng: `Now: on` ở `:257` tính span ngày rồi bỏ |
| thực thể | bộ nhận dạng của luồng hỏi (`:258`); số trong tên máy bị loại khỏi lượng (`:265-270`); bí danh mơ hồ → cách ly (`:262-264`); một tin nhiều máy → cách ly (`:286-287`); kế thừa thực thể của tin trước, cùng người, cùng ngày (`:271-285`) | phạm vi người gửi; ứng viên trong phạm vi |
| lượng | đúng MỘT lượng ngoài cụm thực thể (`:295-316`); số trần lấy đơn vị mặc định (`:318-323`); đổi về đơn vị chuẩn (`:324-331`); kiểm miền (`:229-232`) | nhiều lượng thì cách ly có mã riêng |
| ước lượng | cờ `Value.Approx` (`core/contracts/contracts.go:80-82`); nhận "~" và các từ ở `approx_markers` (`data/normalization/engine.go:99-107`, `:198-211`; `pack/lexicon.yaml:26`) | "≈" không được nhận (HT-1) |
| khoá bản ghi | `<tệp>#<record_id>` (`ingest.go:173`); chú thích hứa dự phòng bằng số dòng (`:22`) nhưng mã không làm | khoá tất định khi thiếu hay trùng `record_id` (HT-8) |
| cầu sang kho sự thật | `Result.Facts()`: ID là khoá; `Source` chỉ có ID, loại nguồn, tham chiếu, nguyên văn (`data/ingestion/bridge.go:36-45`) | giờ gửi, người gửi; lọc theo kết cục và loại |
| cộng | `aggregate` cộng mọi fact cùng loại nguồn trong (thực thể, ngày) (`data/sourceoftruth/query.go:199-221`), đúng thiết kế (`TestCungNguonCungNgayLaCongDon`, `query_test.go:53`) | khử trùng ở phía nạp |
| lưu bền | bảng `facts` có `approx`, `source_type`, `source_ref`, `raw`, `observed_at`, `reported_by` (`platform/pgstore/load.go:76-95`); `Load` thay trọn dữ liệu tenant (`load.go:21-48`) | bảng bản ghi báo cáo; nạp thêm, không thay trọn |
| đo | G1 hiện so cả cờ ước lượng: 2.736/2.739, ba lệch đã biết ghim ở `tenants/construction/win/g1-approx-known.txt` (`docs/STATUS.md:62`) [REPO] | — |

Tầng nạp **không đổi**: `data/ingestion`, `data/sourceoftruth`, `data/entityresolution`, `platform/` và các tệp pack `sources.yaml`, `metrics.yaml`, `policy.yaml`, `pack.yaml` giống hệt từ `7d72e33` tới `2b80a38` [ĐO `git diff --stat 7d72e33 2b80a38 -- data/ingestion data/sourceoftruth data/entityresolution platform tenants/construction/pack/sources.yaml tenants/construction/pack/metrics.yaml tenants/construction/pack/policy.yaml tenants/construction/pack/pack.yaml` → rỗng], và từ `7cad584` (mốc đo K1–K5) tới `7d72e33` [ĐO `git diff --stat 7cad584 7d72e33 -- data core intelligence platform tenants/construction/pack` → 5 tệp, đều ngoài tầng nạp]. Đổi từ `7d72e33` tới `2b80a38`: `data/normalization/numwords_vi.go` (D-OP-14: bộ đọc số bằng chữ đọc theo lối nói tắt; `AmbiguousNumberWords` ở `numwords_vi.go:88`), `core/pipeline/`, `core/tenant/pack.go`, `intelligence/`, `intents.yaml`, `lexicon.yaml`, `templates.yaml` [ĐO `git diff --stat 7d72e33 2b80a38 -- data platform core tenants/construction/pack intelligence db` → 13 tệp].

#### 1.3.2 Mười lỗi hiện trạng (HT-1…HT-10): tám còn mở, HT-6 và HT-7 đã sửa ở `2b80a38`

| Mã | Lỗi | Bằng chứng | Nhãn |
|---|---|---|---|
| HT-1 | "≈" mất cờ ước lượng | tin 01 ra `approx=false`; đối chứng "~" và "khoảng" ra `true`. `engine.go:198-211` chỉ nhận ký hiệu `~`; `g1-approx-known.txt` ghim R-003 | [ĐO] [CODE] [REPO K1] |
| HT-2 | tin đính chính thành số mới của ngày gửi | tin 06, gửi 20/10: "hôm qua báo nhầm, máy 03 là 3k" → quan sát 3.000 L, ngày 20/10 | [ĐO] [REPO K2] |
| HT-3 | tin kế hoạch thành quan sát của ngày gửi | tin 05, gửi 19/10: "mai cần 4k máy 03" → 4.000 L, ngày 19/10 | [ĐO] [REPO K3] |
| HT-4 | ngày viết trong chữ bị bỏ | tin 13 ("hôm qua", gửi 20/10) → ngày 20/10; tin 11 ("ngày 25/10", gửi 19/10) → ngày 19/10; `ingest.go:257` tính span rồi bỏ | [ĐO] [CODE] |
| HT-5 | tin gửi lại thành hai quan sát | tin 07 và 08 (cùng chữ, cùng giờ, khác mã tin), tin 19 và 20 (cùng người, cùng chữ, cách 10 phút) → mỗi cặp 2 quan sát; kho cộng cả hai (`query.go:199-221`) | [ĐO] [CODE] [REPO K4] |
| HT-6 | số bằng chữ đọc sai (**ĐÃ SỬA** ở `2b80a38`) | ở `7d72e33`: tin 09 "một nghìn hai lít" → 1.002 L; tin 17 "ba nghìn tám lít" → 3.008 L. Ở `2b80a38` (D-OP-14): 1.200 L và 3.800 L. `AmbiguousNumberWords` (`numwords_vi.go:88`) vẫn báo các dạng nói tắt để cổng phán quyết không phán quyết chúng | [ĐO] [CODE] [REPO K5, D-OP-14] |
| HT-7 | số bằng chữ không dấu không tất định (**ĐÃ SỬA** ở `2b80a38`) | tin 18 "hai muoi lit", 20 tiến trình riêng: ở `7d72e33` 14 lần "10 L", 6 lần "20 L"; ở `2b80a38` 20/20 lần "20 L" | [ĐO] [REPO báo cáo K1–K5 mục 4] |
| HT-8 | khoá bản ghi trùng | hai tin không có `record_id` cùng khoá `messages.jsonl#`; hai tin cùng `record_id` "X-1" cùng khoá `messages.jsonl#X-1` | [ĐO] [CODE `ingest.go:173`] |
| HT-9 | dòng PDF không khớp mẫu bị bỏ im lặng | `readers.go:165-177`: chỉ dòng `n. chữ` được giữ | [CODE] [REPO `docs/ROADMAP.md` §0.4 mục 12]; chưa đo ở mốc này |
| HT-10 | cách ly chỉ trong RAM, không người nhận, không xuất | `ingest.go:33-42`; `Load` không lưu cách ly (`load.go:21-48`); `docs/01-architecture.md` §4, hàng "cách ly" | [CODE] [REPO] |

Hành vi hiện có **đúng hướng**, giữ lại [ĐO]: một tin nhiều máy → cách ly `ambiguous_entity` (tin 10); tin thiếu lượng → `missing_value` (tin 12); tin kế thừa thực thể có `note` (tin 16); 20 tin vào = 17 quan sát + 3 cách ly, không tin nào mất. Hai điểm lệch nhỏ: tin chỉ có đính kèm (text rỗng, tin 14) ra `unknown_entity` "tin không nhắc thiết bị nào", lý do gây hiểu lầm; trường lạ của tin (ví dụ `attachments`) bị bỏ im lặng (`ingest.go:164-169`: chỉ đọc cột đã ánh xạ) [ĐO + CODE].

#### 1.3.3 Phép đo của người viết

- **Cách làm:** test thăm dò tạm (không commit, đã xoá) dựng một thư mục tenant giả lập gồm `entities.json` (bốn máy EX-03, RL-02, EX-07, RL-07 như `data/ingestion/ingest_test.go`) và `messages.jsonl` 20 dòng; gọi `ingestion.Ingest` với pack `construction`; in mọi quan sát và cách ly. Chạy ở `7d72e33`, rồi chạy lại y hệt ở `2b80a38`. Lệnh: `go test ./data/ingestion -run <tên> -count=1 -v`, qua `taskset`. Riêng tin 18: biên dịch sẵn test (`go test -c`), chạy 20 tiến trình riêng. Riêng HT-8: thư mục thứ hai gồm 4 dòng.
- **Giới hạn:** đường bộ nhớ; một tenant giả lập; chưa đo PDF, xlsx, ảnh, tổng ở kho (K4 đo ở mốc cũ, [REPO]); người gửi và giờ do người đo chọn.

| Tin | Gửi (+07:00) | Người gửi | Chữ | Kết quả ở `2b80a38` (khác `7d72e33` thì ghi trong ngoặc) |
|---|---|---|---|---|
| 01 | 19/10 09:00 | S01 | `máy 03 ≈ 3k` | EX-03 3.000 L, ngày 19/10, `approx=false` |
| 02 | 19/10 09:05 | S02 | `máy 03 ~ 3k` | 3.000 L, `approx=true` |
| 03 | 19/10 09:10 | S03 | `máy 03 khoảng 3k` | 3.000 L, `approx=true` |
| 04 | 19/10 09:15 | S04 | `máy 03 3k` | 3.000 L, `approx=false` |
| 05 | 19/10 18:00 | S05 | `mai cần 4k máy 03` | 4.000 L, ngày 19/10 (kế hoạch thành quan sát) |
| 06 | 20/10 08:00 | S06 | `hôm qua báo nhầm, máy 03 là 3k` | 3.000 L, ngày 20/10 (đính chính thành số mới) |
| 07 | 19/10 10:00 | S07 | `lu 02 1200 lít` | RL-02 1.200 L |
| 08 | 19/10 10:00 | S07 | `lu 02 1200 lít` | RL-02 1.200 L (bản thứ hai) |
| 09 | 19/10 11:00 | S09 | `máy 03 một nghìn hai lít` | 1.200 L (ở `7d72e33`: 1.002 L) |
| 10 | 19/10 11:05 | S10 | `máy 03 2k, lu 02 1k2` | cách ly `ambiguous_entity`: "một tin nhắc nhiều thiết bị" |
| 11 | 19/10 11:10 | S11 | `ngày 25/10 máy 03 4k` | 4.000 L, ngày 19/10 |
| 12 | 19/10 11:15 | S12 | `máy 03` | cách ly `missing_value` |
| 13 | 20/10 08:30 | S13 | `hôm qua máy 03 2k` | 2.000 L, ngày 20/10 |
| 14 | 19/10 11:20 | S14 | (text rỗng) | cách ly `unknown_entity` |
| 15 | 19/10 12:00 | S15 | `máy xúc 07 sáng 1k` | EX-07 1.000 L |
| 16 | 19/10 15:00 | S15 | `chiều thêm 1k` | EX-07 1.000 L, `note` kế thừa thực thể |
| 17 | 19/10 12:10 | S17 | `máy 03 ba nghìn tám lít` | 3.800 L (ở `7d72e33`: 3.008 L) |
| 18 | 19/10 12:20 | S18 | `máy 03 hai muoi lit` | 20 L, 20/20 lần (ở `7d72e33`: 10 L hoặc 20 L, 14 / 6 trong 20 lần) |
| 19 | 19/10 12:30 | S19 | `máy 03 2k` | 2.000 L |
| 20 | 19/10 12:40 | S19 | `máy 03 2k` | 2.000 L (bản thứ hai, 10 phút sau) |

### 1.4 Liên hệ với các khối khác

- **M1 (làn 0):** `POST /v1/ingest` gọi hàm nạp theo hợp đồng này; principal và tenant lấy từ token, không từ body. Mỗi đơn vị có vết từng bước (đọc, chuẩn hoá, thực thể, gán nghĩa, thời gian, kiểm miền, cách ly, đối soát; `docs/PROMPT-BUILD-EAIS.md` §9.4). Cách phát vết do làn 0 quyết; hợp đồng này chỉ đòi mỗi đơn vị có vết (GA10).
- **M3 và H4:** Phase 0a không đi qua S4. Cổng đọc lại đi qua kho sự thật (`sourceoftruth`), không qua luồng hỏi S0–S10; H4 không bị đổi.
- **M5 (CỔNG TRƯỚC AI):** GA1–GA5 là các hàng L1 và S2 của `docs/PROMPT-BUILD-EAIS.md` §9.3; GA9 phủ phần "khởi động lại giữa chừng" của hàng Vận hành; GA10 phủ phần nạp của hàng Vết.
- **Phase 3:** sửa tận gốc đính chính, kế hoạch, đến muộn bằng mô hình sự kiện (`docs/ROADMAP-DAY-DU.md` §5 L5). Hợp đồng này chỉ chặn lỗi im lặng (cờ và chờ xác nhận); các trường `kind`, `corrects`, `supersedes`, `version` được chọn để Phase 3 dùng lại.
- **Phase 1 (M4):** số đội báo sai tạo chênh lệch giả; hợp đồng này giảm rủi ro đó (hệ quả 7 của ADR-0011, `docs/adr/0011-phase1-hanh-dong-co-kiem-soat.md`).

### 1.5 Giả định danh tính và tin cậy kênh

- Bên gọi hàm nạp là một Principal; tenant lấy từ Principal (như `core/pipeline/ask.go:89-95`), **không** từ nội dung nạp. Ở môi trường dev, Principal dựng tay được ghi rõ là "dev only" (làn 0).
- `sender_id` do bộ chuyển đổi của kênh giao, sau khi kênh đã xác thực người đó theo cách của kênh. Hợp đồng **không** xác thực người gửi: giả mạo `sender_id` là rủi ro của Phase 5 và 8 (§4.4).

---

## 2. Đầu vào

### 2.1 Bốn kênh vào

| Mã | Kênh | Định dạng | Hiện có? |
|---|---|---|---|
| KV1 | tin nhắn nhóm đã xuất | JSON Lines, mỗi dòng một phong bì tin (§2.2) | có `readJSONL` (`readers.go:139-162`), thiếu trường |
| KV2 | tệp báo cáo ngày | PDF văn bản (`n. chữ`); bảng xlsx, csv, json | PDF và bảng có; chưa có loại nguồn bảng của đội |
| KV3 | bản ghi từ form / bot | csv, xlsx, json có cấu trúc (§13) | đường đọc có; loại nguồn chưa có |
| KV4 | ảnh, âm thanh, tệp lạ | bất kỳ | không đọc; cách ly `unsupported_media` |

Connector trực tiếp (webhook Zalo, email) là Phase 8. Bộ chuyển đổi của connector phải giao cùng phong bì ở §2.2.

### 2.2 Phong bì tin nhắn (`MessageEnvelope`, tên đề xuất)

| Trường | Bắt buộc | Nghĩa | Hiện có? |
|---|---|---|---|
| `channel` | có | mã kênh, ví dụ `zalo_group` | không |
| `source_message_id` | không (nên có) | mã tin do kênh cấp; khoá chống trùng mức TR1 | có dạng `record_id` |
| `sent_at` | có | RFC 3339, có múi giờ | có (`sent_at`) |
| `sender_id` | có | mã người gửi ổn định, không phải tên hiển thị | không (chỉ `sender`) |
| `sender_display` | không | tên hiển thị, chỉ để người đọc | `sender` |
| `text` | có (được rỗng nếu có `attachments`) | nguyên văn | có |
| `attachments[]` | không | `{kind, ref}`; không đọc, chỉ giữ tham chiếu | không (bị bỏ im lặng) |
| `reply_to` | không | mã tin được trả lời hoặc trích dẫn | không |
| `edited_of` | không | mã tin bị sửa | không |
| `recalled` | không | tin bị thu hồi | không |

- **Tenant không nằm trong phong bì.** Trường `tenant` hay trường lạ khác không bao giờ chọn tenant: bị bỏ qua có ghi vết, hoặc từ chối.
- Trường lạ không được làm rơi đơn vị, và `attachments` không được bỏ im lặng.
- Hình dạng của kênh thật (Zalo xuất gì, có mã tin hay `reply_to` không) **chưa biết** (U5); bảng trên là giả định để thiết kế.

### 2.3 Tệp báo cáo

- **PDF văn bản:** mỗi dòng `n. chữ` là một đơn vị (`readers.go:165`); ngày lấy từ tên tệp `YYYY-MM-DD.pdf` (`ingest.go:240-243`). Tên tệp sai dạng → mỗi dòng bị cách ly `unparseable` (đã đúng hôm nay: `ingest.go:187-190`).
- **Dòng không khớp mẫu:** hoặc thuộc danh sách "bỏ qua" khai ở pack (tiêu đề "Báo cáo ngày …", số trang, dòng trống), hoặc cách ly `unrecognized_line`. Hôm nay bị bỏ im lặng (HT-9).
- **Bảng (xlsx, csv, json):** mỗi hàng dữ liệu là một đơn vị. Hàng chỉ có khoảng trắng không phải đơn vị.
- **Tệp không mở hay giải mã được:** đơn vị là cả tệp, cách ly `unreadable_file`; các tệp khác vẫn nạp. Hôm nay lỗi tệp làm dừng cả lô (`ingest.go:137-140`).

### 2.4 Đơn vị đầu vào và khoá đơn vị

- **Đơn vị** là: một tin (kể cả tin chỉ có đính kèm); một dòng PDF không rỗng; một hàng dữ liệu của bảng; một tệp không đọc được. Dòng hoặc hàng chỉ có khoảng trắng không phải đơn vị.
- **`unit_key` tất định**, duy nhất trong tenant:
  - `<kênh hoặc tệp tương đối>#<mã nguồn của đơn vị>` khi có mã nguồn (`record_id`, `source_message_id`, số dòng PDF);
  - `<tệp>#L<thứ tự dòng>` khi không có mã nguồn.
- Hai đơn vị khác nội dung **không** được chung `unit_key` (HT-8). Cùng mã nguồn, khác nội dung, không có `edited_of` → `HELD` với mã `id_conflict`; có `edited_of` → sửa tin (§7.2).

### 2.5 Điều không được làm với đầu vào

- không lấy tenant từ nội dung nạp;
- không đoán ngày, máy, lượng, loại tin: không chắc thì `HELD` hoặc `QUARANTINED` (BB6);
- không sửa nguyên văn `raw`; mọi suy luận ghi ở `note` và vết;
- không bỏ trường lạ có mang đính kèm.

---

## 3. Đầu ra

### 3.1 Bốn kết cục và các bất biến

| Kết cục | Nghĩa | Vào tổng đọc? |
|---|---|---|
| `ACCEPTED` | đọc được đủ, tự động; mang cờ loại | có, nếu `kind = OCCURRED` |
| `HELD` | cần người quyết: có ứng viên hoặc câu hỏi cụ thể (mơ hồ, nghi trùng, đính chính, …) | không |
| `QUARANTINED` | không thành bản ghi hợp lệ như đầu vào hiện có (không đọc được, ngoài miền, thiếu, lạ) | không |
| `DUPLICATE` | trùng đơn vị đã có; không tạo bản ghi nghiệp vụ mới, chỉ ghi vết trỏ về bản gốc | không (bản gốc đã tính) |

Trạng thái vòng đời thêm sau quyết định: `SUPERSEDED` (bị bản mới thay), `RETRACTED` (thu hồi), `REJECTED` (người bác). Không vào tổng.

| Mã | Bất biến |
|---|---|
| **BB1** | Bảo toàn: với mọi lô, số đơn vị vào = `ACCEPTED` + `HELD` + `QUARANTINED` + `DUPLICATE` (theo kết cục lúc xử lý). Mỗi đơn vị đúng một kết cục tại một thời điểm và có dòng trong kho. |
| **BB2** | Mọi kết cục khác `ACCEPTED` có mã lý do thuộc danh sách đóng (§3.4); mọi `HELD` và `QUARANTINED` có người nhận (§6.3). |
| **BB3** | Chỉ bản ghi `ACCEPTED` với `kind = OCCURRED` thành fact (§3.6). Mọi thứ khác không bao giờ vào tổng. |
| **BB4** | `raw` giữ nguyên byte; không chuẩn hoá lại. |
| **BB5** | Tất định: cùng đầu vào và cùng phiên bản pack thì cùng kết cục, cùng trường, cùng báo cáo xuất, qua mọi tiến trình. |
| **BB6** | Đơn điệu về an toàn: luật không chắc thì `HELD` hoặc `QUARANTINED`, không `ACCEPTED` với giá trị đoán. |
| **BB7** | Quyết định của người không bị máy ghi đè im lặng (tin sửa, nạp lại, xử lý lại luật). |
| **BB8** | Tenant chỉ từ danh tính; không bản ghi nào của tenant này hiện ra ở tenant khác. |

### 3.2 Bản ghi báo cáo (`ReportRecord`, tên đề xuất)

| Trường | Nghĩa | Hiện có? |
|---|---|---|
| `unit_key` | khoá đơn vị (§2.4) | `Observation.Key` |
| `tenant` | từ danh tính | `Tenant` |
| `channel`, `source_type` | kênh; loại nguồn trong pack | `Source` (loại nguồn) |
| `sender_id` | người gửi | không |
| `sent_at` | thời điểm gửi, RFC 3339 | không (bị cắt về ngày) |
| `occurred_on` | ngày xảy ra (§3.5) | `On` (đang là ngày gửi) |
| `entity` | thực thể đã giải | `Entity` |
| `metric` | `reported_fuel` | `Metric` |
| `value` `{number, unit}` | lượng ở đơn vị chuẩn | `Value` |
| `approx` | cờ ước lượng | `Value.Approx` |
| `kind` | `OCCURRED`, `PLAN`, `CORRECTION` | không |
| `corrects` | tham chiếu bản ghi bị đính chính, nếu biết | không |
| `raw` | nguyên văn | `Raw` |
| `note` | suy luận đã áp: kế thừa thực thể, đơn vị mặc định, ngày mặc định | `Note` |
| `disposition`, `reason`, `detail`, `recipient` | kết cục, mã lý do, chi tiết, người nhận | `Quarantine.Reason`, `Detail` |
| `duplicate_of` | trỏ về bản gốc | không |
| `version`, `supersedes` | phiên bản khi sửa tin | không |
| `decided_by`, `decided_at`, `decision` | quyết định của người | không |
| `pack_version` | phiên bản pack lúc xử lý (`core/tenant/pack.go:98-100`) | không |
| `trace_id` | vết (làn 0) | không |

### 3.3 Bốn cờ loại

Hai trục độc lập: `kind` ∈ {`OCCURRED`, `PLAN`, `CORRECTION`} và `approx` ∈ {đúng, sai}. Bốn cờ của brief (`docs/PROMPT-BUILD-EAIS.md` §9.3, hàng L1) là cách gọi theo người đọc:

| Cờ loại | Biểu diễn | Nhận dạng (luật tất định, từ lexicon của pack) | Kết cục mặc định | Vào tổng đọc? |
|---|---|---|---|---|
| ĐÃ XẢY RA | `OCCURRED`, `approx` sai | báo một lượng đã cấp hoặc đã dùng; không có dấu ước lượng, đính chính, kế hoạch | `ACCEPTED` | có |
| ƯỚC LƯỢNG | `approx` đúng (với mọi `kind`) | dấu ước lượng đứng ngay trước lượng: ký hiệu "≈", "~" hoặc từ trong `approx_markers` (`lexicon.yaml:26`) | `ACCEPTED` nếu `OCCURRED` | có, tổng mang cờ `approx` (đường chuyển cờ có sẵn: `query.go:208-210`) |
| ĐÍNH CHÍNH | `CORRECTION` | tin nói rõ sửa số đã báo ("báo nhầm", "đính chính", "sửa lại", …), hoặc `reply_to` kèm nội dung thay số | `HELD` mã `correction_unconfirmed` | không |
| KẾ HOẠCH | `PLAN` | lượng chưa xảy ra: từ chỉ tương lai hoặc nhu cầu ("mai", "dự kiến", "cần", "xin cấp", …), hoặc ngày xảy ra sau ngày gửi | `ACCEPTED` (lưu, `PLAN`) | không |

| Mã | Luật |
|---|---|
| **LK1** | Hai trục độc lập; tổ hợp (đính chính ước lượng, kế hoạch ước lượng) giữ **cả hai** thuộc tính. |
| **LK2** | Danh sách từ khoá nằm ở pack (lexicon), có phiên bản. Mã dùng chung không chứa từ miền (đã có nợ tương tự: D-OP-8, `qualifier.go`). |
| **LK3** | Phân loại bảo thủ **cả hai chiều**. Gắn `PLAN` hay `CORRECTION` oan làm số thật biến khỏi tổng (thiếu số); bỏ sót làm số không thật vào tổng (S1). Từ khoá có mặt nhưng phần còn lại không đọc chắc → `HELD` mã `ambiguous_kind`. |
| **LK4** | Không dùng model, không dùng thứ tự đến, không dùng giá trị để phân loại. |
| **LK5** | Nghĩa hậu tố "k" ("2k" là số đo viết tắt hay ước lượng) **chưa chốt**: D-OP-17 (`docs/CHO-QUYET-DINH.md:133`, mới 22/10 trưa, cột trả lời trống). Hợp đồng không định nghĩa "2k" là ước lượng hay không; bộ ca độc lập không có ca mà kỳ vọng phụ thuộc điều đó (§12). |
| **LK6** | Số ước lượng đi vào tổng kèm cờ (hiện trạng, `query.go:208-210`); tách thành hai tổng là câu mở Q-0A-3. |
| **LK7** | Tin kế thừa thực thể (`CarryEntity`, `ingest.go:277-285`) từ một tin không phải `OCCURRED` hoặc không `ACCEPTED` thì `HELD`, không tự `ACCEPTED`. |

### 3.4 Mã lý do

Danh sách đóng. Mã hiện có giữ nguyên tên; mã mới ghi "mới".

| Mã | Kết cục | Khi nào | Có sẵn? |
|---|---|---|---|
| `missing_value` | `QUARANTINED` | tin nhắc máy nhưng không có lượng (`ingest.go:310-312`) | có |
| `unknown_entity` | `QUARANTINED` | không nhận ra máy (`ingest.go:204`, `:283`) | có |
| `ambiguous_entity` | `HELD` | bí danh khớp nhiều máy; kèm ứng viên trong phạm vi người gửi (`ingest.go:262-264`) | có (hôm nay `QUARANTINED`) |
| `out_of_range` | `QUARANTINED` | ngoài miền `min`–`max` của chỉ số (`ingest.go:229-232`, `metrics.yaml:14-15`) | có |
| `unit_dimension` | `QUARANTINED` | khác chiều đo, ví dụ kg thay lít (`ingest.go:324-331`) | có |
| `unparseable` | `QUARANTINED` | không đọc được: ngày hỏng, số trần không đơn vị, tên tệp sai dạng | có |
| `multiple_items` | `QUARANTINED` | một tin có nhiều máy hoặc nhiều lượng (hôm nay là `ambiguous_entity` ở `:286-287` hoặc `unparseable` ở `:315`) | mới |
| `number_reading_uncertain` | `HELD` | số bằng chữ mà giá trị còn do cách nói tắt quyết dù bộ đọc đã sửa: dãy kết thúc bằng "linh" hoặc "lẻ" (nghĩa xấp xỉ; `numwords_vi.go:88`), hoặc bộ đọc không ra một giá trị duy nhất | mới |
| `unsupported_media` | `QUARANTINED` | ảnh, âm thanh, tệp định dạng lạ; giữ tham chiếu | mới |
| `unrecognized_line` | `QUARANTINED` | dòng PDF không khớp mẫu và không thuộc danh sách bỏ qua | mới |
| `unreadable_file` | `QUARANTINED` | đơn vị là cả tệp; không mở hay giải mã được | mới |
| `unknown_sender` | `QUARANTINED` | `sender_id` không có trong danh bạ | mới |
| `sender_not_authorized` | `HELD` | máy ngoài phạm vi của người gửi | mới |
| `date_conflict` | `HELD` | ngày trong chữ sau ngày gửi mà không có từ khoá kế hoạch; ngày không hợp lệ | mới |
| `correction_unconfirmed` | `HELD` | mọi tin đính chính | mới |
| `possible_duplicate` | `HELD` | nghi trùng mức TR3 (§5.1) | mới |
| `ambiguous_kind` | `HELD` | luật loại tin mâu thuẫn hoặc không chắc (LK3) | mới |
| `id_conflict` | `HELD` | cùng mã nguồn, khác nội dung, không có `edited_of` (§2.4) | mới |

`HELD` khác `QUARANTINED` ở chỗ: `HELD` có ứng viên hoặc câu hỏi để người **chọn, xác nhận, sửa**; `QUARANTINED` không thành bản ghi hợp lệ, người chỉ có thể **xử lý lại, bỏ qua kèm lý do, hoặc nhập bản ghi thay** (§6.2).

### 3.5 Thời gian: ngày xảy ra và ngày gửi

`sent_at` giữ nguyên, có múi giờ. `occurred_on` là ngày dầu được cấp hoặc dùng, theo múi giờ tenant (`pack.yaml:6`).

| Mã | Luật |
|---|---|
| **LT1** | Chữ có biểu thức ngày đọc được ("hôm qua", "hôm kia", "ngày 19/10", "sáng nay", "tối qua") → `occurred_on` lấy theo chữ, tính từ `sent_at`. Hôm nay bị bỏ (HT-4). |
| **LT2** | Không có biểu thức ngày → `occurred_on` = ngày gửi; `note` ghi "ngày gửi làm mặc định". |
| **LT3** | `occurred_on` sau ngày gửi → không bao giờ `ACCEPTED` loại `OCCURRED`: `PLAN` nếu có từ khoá kế hoạch, ngược lại `HELD` mã `date_conflict`. |
| **LT4** | Ngày không hợp lệ ("31/02") → không rơi về ngày gửi im lặng; `HELD` hoặc `QUARANTINED` có lý do (bộ chuẩn hoá đã có hướng này: `data/normalization/datetime.go:207`). |
| **LT5** | Ngày không có năm: lấy năm của ngày gửi; kết quả sau ngày gửi **không** lùi về năm trước (báo cáo K1–K5 mục 4.2 ghi "Ngày 21/10" khi đồng hồ là 20/10/2026 bị đọc thành 21/10/2025 [REPO]). Việc của bộ chuẩn hoá (làn 1); hợp đồng chỉ đặt điều kiện. |
| **LT6** | Tin đến muộn (hiệu `sent_at` − `occurred_on` lớn) vẫn được nhận; hai trường giữ riêng. Chính sách kỳ đã đóng là Phase 3. |
| **LT7** | Ca đêm: `01-business-map.md:405` (BR-RPT-02) ghi "ca đêm tính vào ngày bắt đầu ca" [REPO], kèm "lịch ca của pack" mà pack hiện **chưa có** (`tenants/construction/pack/` không có tệp lịch ca) [CODE]. Giờ cắt ngày của ca **chưa chốt** (Q-0A-8). Tới khi có lịch ca: `occurred_on` theo ngày lịch của múi giờ tenant. |

### 3.6 Phép chiếu sang kho sự thật

- Chỉ bản ghi `ACCEPTED` với `kind = OCCURRED` thành `Fact` (`bridge.go:36-45`): `ID` = `unit_key`; `Value.Approx` giữ nguyên; thêm `ObservedAt` = `sent_at` và `ReportedBy` = `sender_id`. Hai cột này đã có ở bảng `facts` (`load.go:80-91`) nhưng nạp chưa điền.
- `aggregate` (`query.go:199-221`) **không đổi**. Chống cộng đôi do phía nạp bảo đảm; không đổi lõi hỏi–đáp của làn 1.
- Bản ghi `PLAN`, `CORRECTION`, `HELD`, `QUARANTINED`, `DUPLICATE`, `SUPERSEDED`, `RETRACTED`, `REJECTED` không thành fact.
- Khi một bản ghi đổi kết cục (người quyết định, sửa tin, thu hồi), phép chiếu cập nhật **trong cùng giao dịch**.
- Thuộc tính phải kiểm: không có hai fact `reported_fuel` cùng (thực thể, ngày, loại nguồn) mà cùng `unit_key` hoặc cùng vân tay (§5.2).
- Bộ đếm "số mục chưa dùng được" theo (thực thể, ngày) cho phía hỏi–đáp là yêu cầu [ĐỀ XUẤT] cho làn đọc (U2); cổng của hợp đồng này đo bằng báo cáo xuất (GA2), không đòi luồng hỏi hiển thị.

### 3.7 Báo cáo cách ly và hàng chờ (xuất)

Mỗi dòng: `unit_key`, `source_type`, `channel`, `sent_at`, `sender_id`, `occurred_on` (nếu có), `raw`, `disposition`, `reason`, `detail`, `recipient`, `status` (đang chờ, đã quyết), `decided_by`, `decided_at`, `decision`, `pack_version`, `trace_id`. Đầu báo cáo: số dòng theo từng mã lý do và theo kết cục. Chi tiết ở §6.4.

---

## 4. Quyền

### 4.1 Ba lớp

| Lớp | Luật |
|---|---|
| **1. Tenant** | Tenant lấy từ Principal. Mọi bảng mới có RLS ENABLE + FORCE theo GUC tenant (như `db/migrations/0001_win.sql:69-70`; vai và GUC theo tên EAIS ở `db/migrations/0002_dinh_danh_eais.sql`) [CODE]. Canary hai tenant: 0 rò (làn 2). |
| **2. Người gửi → máy** | `sender_id` phải có trong danh bạ (§4.3). Máy đã giải phải nằm trong phạm vi của người gửi, theo cùng nghĩa `scope: attribute` của `policy.yaml:3-4` (`Policy.InScope`, `intelligence/router/router.go:143`; tổ tiên qua `entityresolution.Lineage`, `resolver.go:251`). Ngoài phạm vi → `HELD` `sender_not_authorized`. Không có trong danh bạ → `QUARANTINED` `unknown_sender`. |
| **3. Người duyệt** | Đọc hàng chờ, quyết định và xuất báo cáo **theo phạm vi thực thể của mình**, với capability tương ứng (§4.2). |

**Giải bí danh mơ hồ bằng phạm vi** [ĐỀ XUẤT]: bí danh khớp nhiều máy nhưng đúng MỘT máy nằm trong phạm vi người gửi → chọn máy đó, `note` ghi lý do. Từ hai máy trở lên → `HELD` `ambiguous_entity` kèm đúng các ứng viên trong phạm vi. Không máy nào trong phạm vi → `HELD` `sender_not_authorized`. Tên máy ngoài phạm vi của người gửi không đưa vào câu hỏi gửi người gửi (CON-F-015 mức S1, `docs/tenants/construction/07-failure-catalog.md:66`).

### 4.2 Ma trận [ĐỀ XUẤT, chưa có người nghiệp vụ duyệt]

Capability mới (thêm vào `policy.yaml` khi mở mã): `reports.submit` (gửi), `reports.review` (đọc hàng chờ, xuất báo cáo), `reports.decide` (quyết định ở §6.2).

| Hành động | `director` | `accountant` | `project_manager` | `site_supervisor` | `site_engineer` | `driver` |
|---|---|---|---|---|---|---|
| gửi báo cáo cho máy trong phạm vi | không | không | có | có | có | có, máy của mình |
| đọc hàng chờ và cách ly | tất cả | đọc, tất cả | trong phạm vi | không | không | không |
| quyết định (xác nhận, chọn, sửa, bác) | tất cả | không | trong phạm vi | không | không | không |
| xuất báo cáo cách ly | tất cả | tất cả | trong phạm vi | không | không | không |
| đọc fact `reported_fuel` | có sẵn (`metrics.read`) | có sẵn | có sẵn, theo phạm vi | không | có sẵn, theo phạm vi | không |

`site_supervisor` có ở `01-business-map.md:308` nhưng **chưa có** ở `policy.yaml:5-10` (vai hiện có: `director`, `accountant`, `project_manager`, `site_engineer`, `driver`). Thêm vai hay dùng `site_engineer`: Q-0A-7.

### 4.3 Danh bạ người gửi

- **Là dữ liệu tenant**, đặt cùng chỗ với danh mục thực thể (`entities.json`, nạp ở `ingest.go:121`); **vai và capability** ở pack (`policy.yaml`).
- Mỗi dòng: `sender_id`, tên hiển thị (có thể nhiều), vai, phạm vi thực thể (khoá dự án hoặc máy).
- **Đóng mặc định:** tenant bật kênh mà không có danh bạ hợp lệ → lỗi khi khởi động (như `LoadSources`, `sources.go:62-130`), không chạy ở chế độ "tin hết".

### 4.4 Tin cậy kênh

`sender_id` được tin như kênh giao (§1.5). Phạm vi người gửi giới hạn thiệt hại khi `sender_id` bị giả: người đó chỉ ghi được cho máy của người bị giả. Xác thực chặng kênh là Phase 5, 8.

---

## 5. Chống trùng

### 5.1 Ba mức

| Mức | Khoá | Kết cục khi gặp |
|---|---|---|
| **TR1: khoá nguồn** | (tenant, `channel`, `source_message_id`) | cùng khoá, cùng nội dung chuẩn hoá → `DUPLICATE` (trả bản gốc). Cùng khoá, khác nội dung → sửa tin nếu có `edited_of` (§7.2), ngược lại `HELD` `id_conflict` |
| **TR2: vân tay** | băm mật mã của (tenant, `channel`, `sender_id`, `sent_at` đến giây, chữ chuẩn hoá §5.2) | trùng → `DUPLICATE`. Dành cho nguồn không có mã tin (tệp xuất chồng lấp, PDF gửi lại) |
| **TR3: nghi trùng** | cùng (thực thể, `kind`, ngày xảy ra, giá trị, đơn vị) với một bản `ACCEPTED` hoặc `HELD` đã có, và `sent_at` cách nhau ≤ **W**, **bất kể người gửi và chữ** | `HELD` `possible_duplicate`, trỏ bản kia. Người quyết: `KEEP_BOTH` hoặc `MARK_DUPLICATE` |

**W = 60 phút** [ĐỀ XUẤT; Q-0A-5]. TR3 không bao giờ tự gộp và không tự cộng: bản bị giữ không vào tổng cho tới khi người quyết.

### 5.2 Chuẩn hoá chữ để băm

NFC, bỏ khoảng trắng đầu cuối, gộp khoảng trắng liên tiếp. **Không** bỏ dấu, **không** đổi hoa thường: để không gộp oan.

### 5.3 Điều không được làm

- **Không khử trùng theo giá trị.** Hai tin hợp lệ có thể cùng giá trị: một lần cấp buổi sáng và một lần cấp buổi chiều cùng số lít. Seed WIN có cặp như vậy (R-001 và R-002; báo cáo K1–K5, K4 nêu rõ rằng khử theo giá trị sẽ sai) [REPO].
- **Không "bản mới nhất thắng" theo (thực thể, ngày).** Nhiều lần cấp trong ngày là nhiều sự kiện (`query.go:199-200`).
- **Không khử trùng xuyên tenant** (BB8).

### 5.4 Nạp lại tệp

Nạp lại cùng tệp cho **cùng trạng thái cuối**: mọi đơn vị là `DUPLICATE` (nhờ `unit_key` tất định, TR1); tổng, hàng chờ, báo cáo xuất không đổi. Hôm nay `Load` thay trọn dữ liệu của tenant (`load.go:21-48`) nên "nạp lại" xoá rồi ghi lại; hợp đồng đòi nạp **thêm** idempotent cho kho bản ghi (U2).

### 5.5 Cùng máy, cùng ngày, hai loại nguồn

Tin chat và PDF cùng báo một (máy, ngày) là đối soát theo thẩm quyền lúc hỏi (`metrics.yaml:16-18`: PDF hạng 1, chat hạng 2), không phải trùng và không xử lý ở lúc nạp.

---

## 6. Cách ly và chờ xác nhận

### 6.1 Bất biến (khớp hàng L1 của `docs/PROMPT-BUILD-EAIS.md` §9.3: "100% bản ghi cách ly có lý do, lưu bền, xuất được báo cáo")

| Mã | Bất biến |
|---|---|
| CL1 | mọi đơn vị khác `ACCEPTED` có đúng một dòng `HELD` / `QUARANTINED` / `DUPLICATE` trong kho (BB1) |
| CL2 | mỗi dòng có mã lý do trong danh sách đóng và `detail` (BB2) |
| CL3 | lưu bền trong PostgreSQL, RLS FORCE, chỉ thêm và đổi trạng thái; sống sót qua khởi động lại |
| CL4 | xuất báo cáo tất định (§6.4) |
| CL5 | có người nhận (§6.3) |
| CL6 | không bao giờ vào tổng đọc (BB3) |

### 6.2 Vòng đời và quyết định của người

Mọi quyết định cần capability `reports.decide` và thực thể nằm trong phạm vi người quyết; ghi audit (ai, khi nào, vì sao); idempotent theo mã quyết định; không bao giờ sửa `raw`; phép chiếu fact cập nhật trong cùng giao dịch.

| Mã lý do | Quyết định cho phép | Hệ quả |
|---|---|---|
| `ambiguous_entity` | `CHOOSE(entity)`, `REJECT` | `ACCEPTED` với máy đã chọn |
| `number_reading_uncertain` | `CORRECT(value)`, `REJECT` | `ACCEPTED` với giá trị người nhập; bản máy đọc giữ ở lịch sử |
| `correction_unconfirmed` | `APPLY(target)`, `REJECT` | `APPLY`: bản gốc `SUPERSEDED`, bản đính chính thành bản ghi `OCCURRED` mới; `REJECT`: tổng không đổi |
| `possible_duplicate` | `KEEP_BOTH`, `MARK_DUPLICATE` | `KEEP_BOTH`: cả hai `ACCEPTED`; `MARK_DUPLICATE`: bản bị giữ thành `DUPLICATE` |
| `sender_not_authorized` | `AUTHORIZE`, `REJECT` | `ACCEPTED`, ghi `authorized_by` |
| `date_conflict`, `ambiguous_kind`, `id_conflict` | `CORRECT(...)`, `REJECT` | `CORRECT`: `ACCEPTED` với giá trị người nhập; `REJECT`: `REJECTED` |
| mọi mã của `QUARANTINED` | `REPROCESS` (sau đổi luật hay danh bạ), `DISMISS(lý do)`, `CORRECT(...)` | `CORRECT` nhập một bản ghi thay, dẫn xuất từ đơn vị đó |

### 6.3 Người nhận [ĐỀ XUẤT; Q-0A-6]

`recipient` là vai kèm phạm vi, tính lúc tạo:
1. `project_manager` có phạm vi chứa dự án của máy (dự án lấy từ danh mục thực thể: `entitiesFile.Equipment[].Project`, `ingest.go:50-57`);
2. không xác định được máy: `director`.

Hạn xử lý và nhắc việc không thuộc Phase 0a (BR-RPT-04, Phase 2).

### 6.4 Báo cáo xuất

- Cột như §3.7; sắp theo `unit_key`; định dạng cố định (JSON Lines hoặc CSV, UTF-8). **Cùng trạng thái kho thì cùng byte** (BB5).
- Lọc theo ngày hoặc kỳ, theo người nhận, theo mã lý do. Cần capability `reports.review` và phạm vi; không chứa dữ liệu tenant khác.
- Báo cáo chứa `raw` và `sender_id`, tức dữ liệu cá nhân của người trong đội: truy cập theo capability, không để ở nơi công khai.

---

## 7. Phục hồi

| Mục | Luật |
|---|---|
| **7.1 Gửi lại, nạp lại** | TR1, TR2 làm mọi lần giao lại thành `DUPLICATE`; trạng thái cuối không đổi (§5.4). Kết quả trả bên gọi chỉ rõ "trùng với bản nào". |
| **7.2 Sửa tin** | Cùng mã nguồn có `edited_of`: bản mới thay bản cũ (`version` + 1, bản cũ `SUPERSEDED`, còn trong lịch sử); tổng dùng bản mới. **Nếu bản cũ đã có quyết định của người** (BB7): bản mới `HELD`, không ghi đè. Chỉ cùng `sender_id` được sửa tin của mình. |
| **7.3 Thu hồi** | Tin bị thu hồi → `RETRACTED`, trừ khỏi tổng, còn trong lịch sử. Thu hồi đến **trước** tin gốc (đảo thứ tự) → ghi dấu, áp khi tin gốc đến; cùng trạng thái cuối. Bản đã có quyết định của người: `HELD` (BB7). |
| **7.4 Thứ tự đến** | Đảo thứ tự đến của một lô không đổi tổng, cờ, tập `HELD` — trừ tham chiếu bản gốc của đính chính (nếu bản gốc chưa đến, đính chính `HELD` và được đánh giá lại khi bản gốc đến). |
| **7.5 Dừng giữa lô, khởi động lại** | Mỗi đơn vị một giao dịch (bản ghi + phép chiếu fact). Chạy lại lô hội tụ về cùng trạng thái: không mất, không nhân đôi (khoá duy nhất `(tenant, unit_key)` ở cơ sở dữ liệu; thua đua → `DUPLICATE`). |
| **7.6 Đổi luật rồi xử lý lại** | Đổi lexicon hay danh bạ rồi `REPROCESS` các mục `QUARANTINED` / `HELD`. Mỗi quyết định ghi `pack_version`; quyết định cũ không bị xoá. Bản đã có quyết định của người không bị xử lý lại (BB7). |

---

## 8. Trạng thái và ai sở hữu

Điều kiện 3 của cổng gốc (`docs/ROADMAP-DAY-DU.md` §4.2): trạng thái, nguồn sự thật, ai ghi, ai đọc, vòng đời, hỏng thế nào, phục hồi thế nào.

| Mặt | Trả lời |
|---|---|
| 1. Trạng thái | bản ghi báo cáo (mỗi đơn vị một dòng, kèm lịch sử phiên bản); fact chỉ là phép chiếu |
| 2. Nguồn sự thật | kho bản ghi báo cáo là nguồn của `reported_fuel` ở chế độ B; `raw` là bằng chứng gốc; tin và tệp nguồn nằm ngoài EAIS |
| 3. Ai ghi | bộ nạp tất định; người duyệt (quyết định, §6.2); không ai khác; model không ghi |
| 4. Ai đọc | luồng hỏi (qua fact); người duyệt (hàng chờ); báo cáo xuất (`reports.review`) |
| 5. Vòng đời | tạo → (`ACCEPTED`, `HELD`, `QUARANTINED`, `DUPLICATE`) → quyết định → `SUPERSEDED` / `RETRACTED` / `REJECTED`; không xoá. Chính sách lưu giữ: chưa có (Phase 5) |
| 6. Hỏng thế nào | §9 |
| 7. Phục hồi | §7 |
| 8. Audit | mỗi chuyển kết cục ghi ai, khi nào, vì sao; không sửa `raw` |
| 9. Đa tenant | RLS FORCE; tenant từ danh tính; khoá chống trùng trong phạm vi tenant |
| 10. Cấu hình | luật (lexicon, miền, ánh xạ nguồn, vai) ở pack; danh bạ người gửi là dữ liệu tenant; hành vi ở mã |

---

## 9. Hỏng thế nào, phục hồi thế nào

| Điểm hỏng | Hậu quả nếu không xử lý | Xử lý |
|---|---|---|
| tệp không mở hay giải mã được | cả lô dừng (hôm nay: `ingest.go:137-140`) | đơn vị là cả tệp, `unreadable_file`; tệp khác chạy tiếp |
| thiếu `pdftotext` (`readers.go:29-35`) | như trên | như trên, nhãn hạ tầng `infra:pdftotext` |
| PostgreSQL ngắt giữa lô | ghi nửa vời | một giao dịch cho mỗi đơn vị; lô chạy lại hội tụ (§7.5) |
| tiến trình chết giữa lô | mất hoặc nhân đôi | idempotent theo `unit_key` |
| hai tiến trình nạp cùng tệp | trùng | ràng buộc duy nhất `(tenant, unit_key)`; thua đua → `DUPLICATE` |
| lexicon hay danh bạ sai | gán sai hàng loạt | `pack_version` ghi ở mỗi bản ghi; xử lý lại; không xoá lịch sử (§7.6) |
| tệp xuất lặp nhiều lần | ngập hàng chờ | TR1, TR2 chặn; báo cáo xuất đếm `DUPLICATE` theo lô |
| `sender_id` bị giả | ghi cho máy không phải của mình | ngoài 0a (Phase 5, 8); phạm vi người gửi giới hạn thiệt hại (§4.4) |

---

## 10. Lớp ca âm bắt buộc

> **Đây là các LỚP ca, không phải bộ ca.** Bộ ca độc lập do người không đọc mã soạn (§12) và **không được chép ví dụ** trong tệp này. Mỗi lớp phải có ít nhất một ca trong bộ độc lập; kỳ vọng là kết cục, lý do, trường, tổng. Chữ "kỳ vọng" dùng nghĩa của hợp đồng này, chưa phải cách đọc của mã hiện tại.

**Nhóm A: bảo toàn, không rơi im lặng (dòng GA1)**

| Mã | Lớp ca (đầu vào) | Kỳ vọng |
|---|---|---|
| LC01 | lô hỗn hợp: tin tốt, tin hỏng, dòng PDF lạ, dòng trống, tệp hỏng | số đơn vị vào = tổng kết cục (BB1); dòng trống không phải đơn vị |
| LC02 | dòng PDF không khớp mẫu `n. chữ`, không thuộc danh sách bỏ qua | `QUARANTINED` `unrecognized_line` |
| LC03 | tệp hỏng, sai định dạng hoặc thiếu công cụ đọc | `QUARANTINED` `unreadable_file` (đơn vị là tệp); các tệp khác nạp đủ |
| LC04 | tin chỉ có ảnh hoặc tệp đính kèm, `text` rỗng | `QUARANTINED` `unsupported_media`, giữ `attachments`; không `unknown_entity` |
| LC05 | tin có cả chữ và ảnh | chữ xử lý bình thường; tham chiếu ảnh giữ trong bản ghi |
| LC06 | tệp đuôi lạ trong thư mục nguồn | `QUARANTINED` `unsupported_media` |
| LC07 | trường lạ trong phong bì, kể cả `tenant` | bỏ qua có ghi vết; không đổi tenant |
| LC08 | tin rất dài, có ký tự điều khiển hay vô hình, sai UTF-8 | vẫn đúng một kết cục; không panic; `raw` giữ nguyên |

**Nhóm B: cờ loại (dòng GA3)**

| Mã | Lớp ca | Kỳ vọng |
|---|---|---|
| LC09 | ước lượng bằng ký hiệu ("≈", "~") và bằng chữ ("khoảng", "tầm", "cỡ", "xấp xỉ"), trong tin chat, dòng PDF, ô bảng | `approx` đúng, cùng giá trị |
| LC10 | từ xen giữa dấu ước lượng và số; tin có hai lượng | cờ áp đúng lượng, không lan; tin hai lượng thì `multiple_items` |
| LC11 | lượng không có dấu ước lượng | `approx` sai (không gắn cờ bừa) |
| LC12 | tin kế hoạch (tương lai, nhu cầu, đề nghị) | `PLAN`, lưu, không vào tổng |
| LC13 | tin kế hoạch gửi trước, tin báo thật gửi sau cho cùng (máy, ngày) | tổng của ngày đó chỉ gồm tin thật |
| LC14 | tin đính chính có tham chiếu rõ (`reply_to` hoặc nêu ngày và máy) | `CORRECTION`, `HELD` `correction_unconfirmed` kèm bản gốc ứng viên; tổng ngày gốc **không đổi** tới khi người quyết |
| LC15 | người quyết `APPLY` đính chính | bản gốc `SUPERSEDED`; tổng ngày gốc đổi đúng; có audit |
| LC16 | người quyết `REJECT` đính chính | tổng không đổi; đính chính `REJECTED` |
| LC17 | đính chính không tìm ra bản gốc, hoặc có từ hai bản gốc ứng viên | `HELD`, liệt kê ứng viên; không đoán |
| LC18 | tin có dấu hiệu vừa kế hoạch vừa đính chính | `HELD` `ambiguous_kind` |
| LC19 | tin báo thật chứa từ gần từ khoá (ví dụ "cần" trong "máy cần bảo dưỡng") | **không** bị gắn `PLAN` hay `CORRECTION` trong im lặng: hoặc `OCCURRED`, hoặc `HELD` `ambiguous_kind` (LK3) |
| LC20 | tổ hợp: ước lượng ∧ đính chính; ước lượng ∧ kế hoạch | giữ cả hai thuộc tính (LK1) |

**Nhóm C: trùng (dòng GA4)**

| Mã | Lớp ca | Kỳ vọng |
|---|---|---|
| LC21 | cùng tin giao hai lần (cùng `source_message_id`, cùng nội dung) | `DUPLICATE`; tổng không đổi |
| LC22 | cùng nội dung, giờ, người gửi, khác mã tin (xuất chồng lấp) | `DUPLICATE` |
| LC23 | nạp lại nguyên tệp đã nạp | mọi đơn vị `DUPLICATE`; trạng thái cuối giống hệt |
| LC24 | gửi lại sau vài phút, cùng người, cùng chữ | `HELD` `possible_duplicate`; không vào tổng |
| LC25 | người khác báo cùng (máy, ngày, giá trị) trong cửa sổ W | `HELD` `possible_duplicate` |
| LC26 | hai tin cùng giá trị, cách nhau hơn W, chữ khác nhau | cả hai `ACCEPTED`; tổng gồm cả hai (không gộp oan) |
| LC27 | hai lần cấp thật cách nhau dưới W, cùng giá trị | `HELD`; người quyết `KEEP_BOTH` thì cả hai vào tổng |
| LC28 | tin giống hệt ở hai tenant | mỗi tenant một bản ghi; không `DUPLICATE` xuyên tenant |

**Nhóm D: số và đơn vị (dòng GA5)**

| Mã | Lớp ca | Kỳ vọng |
|---|---|---|
| LC29 | số bằng chữ nói tắt (chữ số đứng ngay sau từ chỉ hàng: "một nghìn hai", "hai trăm năm", "ba nghìn tám") | đọc theo quy ước D-OP-14 (1.200; 250; 3.800), `note` ghi "đọc theo cách nói tắt"; **không bao giờ** cộng như hàng đơn vị (1.002; 205; 3.008). Dãy kết thúc bằng "linh" hoặc "lẻ" (xấp xỉ): `approx` đúng, hoặc `HELD` `number_reading_uncertain` |
| LC30 | số bằng chữ viết rõ hàng ("một nghìn hai trăm", "hai nghìn rưỡi") | đọc đúng (1.200; 2.500) |
| LC31 | số bằng chữ không dấu ("hai muoi", "ba tram") | cùng kết quả qua ít nhất 100 tiến trình; đúng nghĩa hoặc `HELD` |
| LC32 | hậu tố và đơn vị: "1k2", "2k", "2,5 khối", "1,2 m3" | đúng giá trị ở đơn vị chuẩn (L) |
| LC33 | khác chiều đo ("3850 kg") | `QUARANTINED` `unit_dimension` |
| LC34 | ngoài miền (âm, lớn hơn mức tối đa của chỉ số) | `QUARANTINED` `out_of_range` |
| LC35 | số trong tên máy cạnh chữ "dầu", không lượng | `QUARANTINED` `missing_value`; không ra số của tên máy |
| LC36 | nhiều lượng hoặc nhiều máy trong một tin | `QUARANTINED` `multiple_items` |

Nhóm D không có ca phụ thuộc cách viết nhóm nghìn bằng dấu cách (D-OP-9 còn mở).

**Nhóm E: thời gian (dòng GA7)**

| Mã | Lớp ca | Kỳ vọng |
|---|---|---|
| LC37 | "hôm qua", "hôm kia", "ngày 19/10", "sáng nay", "tối qua" | `occurred_on` đúng; `sent_at` giữ riêng |
| LC38 | ngày trong chữ sau ngày gửi, không có từ khoá kế hoạch | không bao giờ `ACCEPTED` `OCCURRED`: `PLAN` nếu có từ khoá, ngược lại `HELD` `date_conflict` |
| LC39 | ngày không hợp lệ ("31/02") | `HELD` hoặc `QUARANTINED`; không rơi về ngày gửi im lặng |
| LC40 | ngày không có năm ở tương lai gần | cùng năm của ngày gửi; không lùi về năm trước |
| LC41 | tin gửi lúc 23:30 UTC (06:30 +07 hôm sau) | ngày theo múi giờ tenant |
| LC42 | tin gửi sau nửa đêm báo "tối qua" | ngày hôm trước theo ngày lịch (không có ca phụ thuộc lịch ca đêm cho tới khi Q-0A-8 chốt) |

**Nhóm F: thực thể và quyền (dòng GA8)**

| Mã | Lớp ca | Kỳ vọng |
|---|---|---|
| LC43 | người gửi ngoài danh bạ | `QUARANTINED` `unknown_sender` |
| LC44 | người gửi báo cho máy ngoài phạm vi | `HELD` `sender_not_authorized`; không vào tổng |
| LC45 | người có quyền `AUTHORIZE` | `ACCEPTED`, ghi `authorized_by` |
| LC46 | người không có quyền, hoặc ngoài phạm vi, gọi quyết định | bị từ chối kèm audit; trạng thái không đổi |
| LC47 | bí danh mơ hồ giữa hai dự án; người gửi chỉ thuộc một dự án | chọn máy trong phạm vi, `note` ghi lý do |
| LC48 | bí danh mơ hồ; cả hai máy trong phạm vi người gửi | `HELD` `ambiguous_entity` kèm đúng hai ứng viên |
| LC49 | các biểu diễn của một máy (mã chuẩn, mã viết liền không gạch, tên theo kiểu báo cáo, bí danh viết tắt) | cùng một thực thể |
| LC50 | tin kế thừa thực thể sau tin `PLAN`, `CORRECTION` hoặc không `ACCEPTED` của cùng người | không `ACCEPTED` tự động (LK7) |
| LC51 | payload có `tenant` khác tenant của danh tính; canary hai tenant | bỏ qua hoặc từ chối; 0 bản ghi hiện ở tenant kia |

**Nhóm G: cách ly, chờ xác nhận, xuất (dòng GA2)**

| Mã | Lớp ca | Kỳ vọng |
|---|---|---|
| LC52 | mọi kết cục khác `ACCEPTED` | có mã lý do trong danh sách đóng và người nhận (BB2) |
| LC53 | dừng đột ngột dịch vụ giữa lô, khởi động lại | `HELD` và `QUARANTINED` còn nguyên (đối chứng HT-10) |
| LC54 | xuất báo cáo cách ly cho một ngày | đủ dòng, đủ cột; số dòng theo mã lý do khớp kho; cùng trạng thái cho cùng byte qua 100 lần chạy |
| LC55 | người không có `reports.review` đòi đọc hoặc xuất | bị từ chối |
| LC56 | người nhận gán đúng (`project_manager` chứa máy; không xác định máy thì `director`) | khớp §6.3 |
| LC57 | đọc lại tổng của (máy, ngày) có bản `HELD`, `QUARANTINED`, `PLAN` | tổng loại trừ chúng (BB3) |
| LC58 | đổi lexicon hoặc danh bạ rồi `REPROCESS` | quyết định cũ vẫn truy được; `pack_version` mỗi lần |

**Nhóm H: phục hồi (dòng GA9)**

| Mã | Lớp ca | Kỳ vọng |
|---|---|---|
| LC59 | sửa tin (cùng mã, có `edited_of`) | bản mới thay; tổng dùng bản mới; bản cũ `SUPERSEDED` còn lịch sử |
| LC60 | sửa tin khi bản cũ đã có quyết định của người | `HELD`, không ghi đè im lặng (BB7) |
| LC61 | thu hồi tin | `RETRACTED`; trừ khỏi tổng |
| LC62 | thu hồi đến trước tin gốc | cùng trạng thái cuối như thứ tự thường |
| LC63 | đảo thứ tự đến của cả lô | cùng tổng, cùng cờ, cùng tập `HELD` (trừ tham chiếu đính chính) |
| LC64 | dừng giữa lô rồi chạy lại | 0 mất, 0 nhân đôi |
| LC65 | hai tiến trình nạp cùng một tệp đồng thời | một kết quả |
| LC66 | cùng mã tin, khác nội dung, không có `edited_of` | `HELD` `id_conflict` |

**Nhóm I: hạ tầng và ràng buộc chung**

| Mã | Lớp ca | Kỳ vọng |
|---|---|---|
| LC67 | PostgreSQL ngắt giữa lô | lỗi nhãn hạ tầng; không ghi nửa đơn vị; chạy lại hội tụ |
| LC68 | đếm lời gọi model trên toàn bộ đường nạp | đúng 0 (câu 5 của cổng QuânIT) |
| LC69 | 100 lần chạy cùng đầu vào, tiến trình khác nhau | kết cục, trường và báo cáo xuất giống hệt (BB5) |
| LC70 | tin thiếu hoặc trùng `record_id` | không hai đơn vị khác nhau chung `unit_key` (HT-8) |

**Nhóm J: nhánh form / bot (dòng GA11)**

| Mã | Lớp ca | Kỳ vọng |
|---|---|---|
| LC71 | bản chiếu có cấu trúc của các ca nhóm B, D, E | cùng kết cục ngữ nghĩa như đường chat ở phần chung |
| LC72 | form thiếu trường bắt buộc, hoặc giá trị ngoài danh sách chọn | `QUARANTINED` (`missing_value` hoặc `unparseable`) hoặc từ chối ở điểm nhận có ghi vết |
| LC73 | `kind` và `approx` tường minh trong bản ghi form | được tôn trọng, không bị luật đọc chữ ghi đè |

---

## 11. Cổng ra nhị phân

### 11.1 Các dòng cổng

Mỗi dòng cần: bộ ca độc lập, lệnh, kết quả, ca âm (`docs/PROMPT-BUILD-EAIS.md` §9.3). **Lệnh: chưa có**; do làn viết mã và làn phép đo viết khi mở mã. Bộ kiểm cho GA1 phải **không import bộ nạp**, theo cách của G1 DATA (`docs/04-core-mvp.md` §6, dòng G1 DATA: so tập mã bản ghi của manifest với tập quan sát cộng cách ly).

| Dòng | Đo gì | Ngưỡng (nhị phân) | Nguồn của dòng | Ca âm, đột biến |
|---|---|---|---|---|
| **GA1** | Bảo toàn: tập `unit_key` của bộ ca = tập (bản ghi ∪ chờ ∪ cách ly ∪ trùng); thiếu hoặc thừa là ĐỎ | 100% đơn vị có đúng một kết cục; 0 rơi im lặng | §9.3 L1; `docs/04-core-mvp.md` §6 | nhóm A; DB01, DB02, DB20 |
| **GA2** | Cách ly: mọi kết cục khác `ACCEPTED` có mã lý do thuộc danh sách đóng và người nhận; lưu bền qua khởi động lại; báo cáo xuất khớp kho | 100% | §9.3 L1 | nhóm G; DB11, DB14 |
| **GA3** | Cờ loại: ca bơm đính chính, kế hoạch, ước lượng gắn đúng; đọc lại tổng (máy, ngày) qua kho sự thật khớp kỳ vọng | 100% ca bơm; 0 bản ghi sai loại vào tổng | §9.3 L1 | nhóm B; DB03, DB04, DB17, DB19 |
| **GA4** | Không cộng đôi, không gộp oan | 100% ca trùng TR1, TR2 không đổi tổng; 100% ca TR3 vào `HELD`; 0 ca hợp lệ lặp bị gộp | §9.3 L1 ("phiếu trùng không bị cộng đôi"); "không gộp oan" do hợp đồng này thêm | nhóm C; DB05, DB06, DB07 |
| **GA5** | Đọc số tất định và đúng | 100% trên bộ ca số và đơn vị (số bằng chữ, nhóm nghìn, không dấu); 100/100 lần chạy qua tiến trình khác nhau giống hệt; số nói tắt đọc đúng quy ước D-OP-14; dãy "linh" / "lẻ" không `ACCEPTED` với `approx` sai | §9.3 S2 | nhóm D; DB12, DB13 |
| **GA6** | Độ chính xác trên ca "tự nhiên" của bộ độc lập (SLA, §11.2) | sai im lặng S1 = 0; đúng ≥ **97%**; phủ nhận tự động ≥ **85%** [ĐỀ XUẤT] | `docs/ROADMAP-DAY-DU.md` §5b (M2), §6 (Phase 0a) | toàn bộ |
| **GA7** | Thời gian: ngày xảy ra đúng, không rơi về ngày gửi im lặng | 100% ca nhóm E (trừ ca phụ thuộc câu mở) | do hợp đồng này thêm (HT-4) | nhóm E; DB08 |
| **GA8** | Quyền người gửi và tenant | 100% ca nhóm F; canary hai tenant: 0 rò | do hợp đồng này thêm; S5 là của làn 2 | nhóm F; DB09, DB10 |
| **GA9** | Phục hồi: gửi lại, sửa tin, thu hồi, đảo thứ tự, dừng giữa lô, khởi động lại | 100% ca nhóm H; hội tụ | §9.3 Vận hành (khởi động lại) cộng phần thêm | nhóm H; DB15, DB16, DB21 |
| **GA10** | Vết: mỗi đơn vị (kể cả cách ly, trùng) có vết từng bước, kèm lý do và phương án bị loại | 100% đơn vị | §9.3 Vết / debug (phần nạp) | toàn bộ; vết do làn 0 |
| **GA11** | Nhánh form / bot: cùng hợp đồng, parity trên bản chiếu cặp; bảng so hai đường được ghi | 100% ca nhóm J; bảng so có mặt | `docs/ROADMAP-DAY-DU.md` §6 (Phase 0a: "có nhánh so sánh") | nhóm J |

- **M2 ra khi GA1–GA6 đạt** (đúng chữ khối M2: bản ghi có cờ, cách ly hiện ra, không cộng đôi, đọc số tất định, độ chính xác trên bộ độc lập).
- **"PHASE 0A RA"** đòi thêm GA7–GA11, hoặc chủ dự án gỡ dòng nào bằng lời ghi ở `docs/CHO-QUYET-DINH.md`.
- **Không tự nới ngưỡng** (§9.3, D-AI-01). Dòng không đạt được chỉ bằng luật: ghi vào CHO để chủ dự án chọn.
- Mọi kết luận kèm câu: "trên dữ liệu giả lập, chưa có người nghiệp vụ duyệt".

### 11.2 GA6: độ chính xác (SLA) [ĐỀ XUẤT]

- **Bộ ca độc lập gồm hai lớp:** (a) **ca tự nhiên**: tin hiện trường viết như người thật viết (viết tắt, lỗi gõ, cách nói khác nhau), kỳ vọng do người soạn gán; (b) **ca bơm**: các lớp LC01–LC73 ở trên.
- **Đúng** (một đơn vị): kết cục thuộc kỳ vọng; nếu `ACCEPTED` thì mọi trường khớp (thực thể, ngày xảy ra, giá trị và đơn vị, `kind`, `approx`); nếu `HELD` hay `QUARANTINED` thì mã lý do khớp.
- **Sai im lặng S1:** đơn vị `ACCEPTED` hoặc `DUPLICATE` mà có trường sai; hoặc đơn vị kỳ vọng không-nhận mà được `ACCEPTED`.
- **Phủ nhận tự động:** (đơn vị kỳ vọng `ACCEPTED` được nhận đúng) / (đơn vị kỳ vọng `ACCEPTED`). Dòng phủ có mặt để một cài đặt "cách ly hết" không qua cổng.
- **Báo cáo kèm bảng thành phần theo nhóm lớp**, không chỉ một số gộp.
- **Cỡ mẫu** [ĐỀ XUẤT]: ít nhất 100 ca tự nhiên và ít nhất 12 ca mỗi nhóm A–J. Với 100 ca tự nhiên, 97% nghĩa là tối đa 3 ca lệch; khoảng tin cậy rộng (chưa tính).
- **Vì sao 97% và 85%:** hai số là **điểm xuất phát của người viết, không suy từ dữ liệu**, không có số nền và không có người ký (câu 8 của cổng QuânIT). Mục đích: chấp nhận vài ca chọn nhầm mã lý do hoặc nhầm "chờ" với "nhận" mà **không** chấp nhận lỗi im lặng S1. Chủ dự án hoặc người nghiệp vụ sửa số (Q-0A-1). Ngưỡng thật của doanh nghiệp (SLA) do người nghiệp vụ ký, ngoài tầm của hợp đồng này.

### 11.3 Danh mục đột biến (cổng đỏ được)

| Mã | Đột biến | Dòng phải đỏ |
|---|---|---|
| DB01 | bỏ ghi cách ly (rơi im lặng) | GA1, GA2 |
| DB02 | nuốt dòng PDF không khớp mẫu (như hôm nay) | GA1 |
| DB03 | gỡ "≈" khỏi nhận dạng ước lượng (như hôm nay) | GA3 |
| DB04 | gắn `OCCURRED` cho mọi tin | GA3 |
| DB05 | tắt khử trùng TR1 và TR2 | GA4 |
| DB06 | khử trùng theo giá trị (gộp oan) | GA4 |
| DB07 | bỏ TR3 (nghi trùng) | GA4 |
| DB08 | `occurred_on` luôn bằng ngày gửi | GA7 |
| DB09 | bỏ kiểm phạm vi người gửi | GA8 |
| DB10 | lấy tenant từ nội dung nạp | GA8 |
| DB11 | cách ly chỉ giữ trong RAM | GA2, GA9 |
| DB12 | duyệt `map` ngẫu nhiên khi đọc số không dấu | GA5 |
| DB13 | nhận ngay số bằng chữ hai nghĩa (như hôm nay) | GA5 |
| DB14 | báo cáo xuất bỏ bớt một mã lý do, hoặc không sắp xếp | GA2 |
| DB15 | sửa tin tạo bản ghi mới thay vì `SUPERSEDED` | GA9 |
| DB16 | nạp lại không idempotent | GA4, GA9 |
| DB17 | phép chiếu fact lấy cả `PLAN` hoặc `HELD` | GA2, GA3 |
| DB18 | một lời gọi model trên đường nạp | LC68 |
| DB19 | đính chính tự áp, không chờ người | GA3 |
| DB20 | `unit_key` rỗng khi thiếu `record_id` (như hôm nay) | GA1 |
| DB21 | tin sửa ghi đè quyết định của người | GA9 |

### 11.4 Mức bằng chứng

1. **Ca phát triển** (do làn viết mã soạn, để dev): không phải bằng chứng cổng.
2. **Bộ ca độc lập**: người không đọc mã soạn → đóng băng → băm → commit **cam kết băm** (chỉ sha256; nội dung công bố lúc chạy, theo bài học `docs/adr/0006-bo-giu-lai-doc-lap.md` §5) → chạy **MỘT** lần trên mã đóng băng. Sau đó thành bộ hồi quy. **Cấm** dùng bộ này để chỉnh lexicon rồi lấy chính nó làm bằng chứng (`docs/PROMPT-BUILD-EAIS.md` §9.9).
3. **Đột biến** (§11.3): cổng đỏ đúng dòng.
4. **Soát độc lập** (người viết ≠ người soát; luật dừng `docs/PROMPT-BUILD-EAIS.md` §9.6).

---

## 12. Giao cho người soạn bộ ca độc lập

- **Nhận:** `HOP-DONG.md` các mục **§0, §2, §3, §4, §5, §6, §7, §10 (cột "Lớp ca" và "Kỳ vọng"), §11.1 (bỏ cột "Nguồn của dòng" và "Ca âm, đột biến") và §11.2, §13 (mô tả đường form và bản chiếu cặp) và §15** (để biết nghĩa nào chưa chốt).
- **Bản trích do phiên điều phối dựng**: bỏ mọi tham chiếu tới mã, seed, HT-n, báo cáo K1–K5 và mã bản ghi R-xxx.
- **Không nhận:**
  - mã trong mọi thư mục; pack; lexicon của pack;
  - dữ liệu seed và manifest của `tenants/construction/win/`;
  - bộ giữ lại H1, H3, H4, H5 và mọi nội dung của chúng;
  - `HOP-DONG.md` §1.3 (có đầu vào đã đo), `CONG-QUANIT.md`, ADR-0012;
  - `docs/bao-cao/` (đặc biệt báo cáo K1–K5, có dòng bơm cụ thể).
- **Tự dựng thế giới giả lập riêng** cho bộ ca: danh mục thực thể (bí danh, dự án, mã theo nguồn), danh bạ người gửi (vai, phạm vi). Bộ chạy phải nhận được thế giới đó theo định dạng của `entities.json` và danh bạ ở §4.3.
- **Định dạng mỗi ca:** `id`; `class` (mã lớp LC-xx hoặc "tự nhiên"); `input` (phong bì hoặc nội dung tệp); `world`; `expected` (kết cục, mã lý do, `kind`, `approx`, giá trị, đơn vị, thực thể, `occurred_on`, tổng đọc lại); `rationale`; `blocked_by` (nếu phụ thuộc câu mở).
- **Cỡ:** ít nhất 100 ca tự nhiên và ít nhất 12 ca mỗi nhóm A–J [ĐỀ XUẤT]. **Cấm** nhân bản hay diễn đạt lại cho đủ số (ADR-0006, D-G7-5).
- **Ca phụ thuộc quyết định chưa chốt** (D-OP-17 nghĩa của "k"; D-OP-9 nhóm nghìn bằng dấu cách; Q-0A-8 ca đêm) phải bị loại hoặc ghi `blocked_by`; không đặt kỳ vọng cho chúng.
- **Bản chiếu cặp** (§13.3): mỗi ca nhóm B, D, E có thêm bản có cấu trúc cho đường form.
- **Quy trình:** soạn → đóng băng → băm → commit cam kết băm → công bố nội dung lúc chạy. Không ai viết mã xem trước.

---

## 13. Nhánh form / bot (thay cho chat tự do)

### 13.1 Vì sao có nhánh

Câu 3 của cổng QuânIT đòi "đã cân nhắc chuẩn hoá đầu vào (form / bot) **trước khi** đọc chữ tự do" (`docs/ROADMAP-DAY-DU.md` §4.1). Cả Phase 0a tồn tại vì chữ tự do sinh ra mười lỗi ở §1.3.2. Nhánh này cho phép chuẩn hoá tại nguồn: người gửi **chọn** máy, **nhập** số, **chọn** loại và ước lượng, thay vì viết.

### 13.2 Hình dạng

Bản ghi có cấu trúc (kênh KV3):

| Trường | Cách nhập |
|---|---|
| `unit_id` | mã phiếu do form hoặc bot sinh (khoá TR1) |
| `submitted_at`, `reporter_id` | do hệ nhận sinh |
| `entity` | chọn từ danh sách (đã lọc theo phạm vi người gửi) |
| `quantity`, `unit` | số và chọn đơn vị (L, m3) |
| `occurred_on` | chọn ngày |
| `kind` | chọn: đã xảy ra, kế hoạch, đính chính |
| `approx` | hộp chọn "ước lượng" |
| `corrects` | chọn từ các phiếu đã gửi của chính người gửi (tuỳ chọn) |

Loại nguồn: một loại có cấu trúc mới cho `reported_fuel`, hạng thẩm quyền do chủ dự án chọn (đề xuất ngang `site_report_form`). Mọi bước sau (chống trùng, quyền, cách ly, phục hồi, phép chiếu fact) **giống đường chat**.

### 13.3 Bản chiếu cặp

Mỗi ca nhóm B, D, E của bộ độc lập có hai bản: chữ tự do và bản có cấu trúc, **cùng kỳ vọng ngữ nghĩa**. Đo trên mỗi đường: số ca đúng, số ca `HELD`, số ca `QUARANTINED`, số bước người cần, thời gian xử lý. Báo cáo là một **bảng theo nhóm**, không chấm "đường nào thắng" bằng một số.

### 13.4 Cái gì đơn giản hơn, cái gì vẫn phải có

- **Đơn giản hơn:** không cần đọc số bằng chữ, không cần nhận ra tin đính chính hay kế hoạch bằng từ khoá; lỗi HT-1, HT-2, HT-3, HT-6, HT-7 biến mất ở nguồn.
- **Vẫn phải có:** chống trùng (người bấm gửi hai lần), quyền người gửi, cách ly có lý do, phục hồi, phép chiếu fact.

### 13.5 Quyết định

Đổi kênh của đội là **quyết định nghiệp vụ** (D-RM-03), không phải của hợp đồng này. Hợp đồng giữ cả hai đường dùng chung lõi. [ĐỀ XUẤT, chưa đo] Dùng đường form cho các con số critical nếu đội chịu đổi; chat là đường phụ có người xác nhận. Kênh cụ thể (form có sẵn, bot Zalo OA, mini app): Q-0A-11; so với đồ có sẵn: `CONG-QUANIT.md` §4.

### 13.6 Điều kiện của dòng GA11

Đường form chạy đủ LC71–LC73 và đạt parity trên phần chung; bảng so hai đường (§13.3) được ghi vào báo cáo.

---

## 14. Điều kiện mở mã và thứ tự chứng minh

### 14.1 Điều kiện mở mã

1. H4 đã chạy xong (D-RM-04 = A; `docs/PROMPT-BUILD-EAIS.md` §9.2, làn 4).
2. D-OP-14 (bộ đọc số, làn 1) đã vào main (`7e1d8ca`, `017495c`); mọi thay đổi sau đó của `data/normalization/` phải kèm đo lại GA5.
3. Hợp đồng `POST /v1/ingest` của làn 0 đã chốt đơn vị nạp và Principal.
4. **Write-set của làn 4** mở rộng thêm `data/ingestion/` và `platform/pgstore/`: hiện `docs/PROMPT-BUILD-EAIS.md` §9.2 ghi làn 4 là `execution/`, migration mới, pack, trong khi mã 0a nằm ở tầng nạp (`docs/01-architecture.md` §3). Câu Q-0A-12.
5. Bộ ca độc lập đã soạn, đóng băng, băm (§12).

### 14.2 Thứ tự chứng minh (đỏ trước)

1. **Đo nền trên mã hiện tại** (luật trước và sau khi đo, `docs/PROMPT-BUILD-EAIS.md` §3): bộ kiểm bảo toàn độc lập cộng bộ ca độc lập → báo cáo đỏ theo nhóm. Đỏ dự kiến: A (HT-9), B (HT-1, 2, 3), C (HT-5), D (HT-6, 7), E (HT-4), F (chưa có quyền), G (HT-10), H, I (HT-8).
2. Dựng kho bản ghi và kết cục (nền của mọi thứ sau).
3. Khử trùng TR1, TR2, TR3.
4. Cờ loại.
5. Thời gian (`occurred_on`).
6. Quyền người gửi và danh bạ.
7. Phục hồi.
8. Báo cáo xuất.
9. Nhánh form.

Mỗi bước: test đỏ trước → sửa nhỏ nhất → đo → đột biến → soát độc lập; quá 3 vòng soát còn lỗi cùng họ thì khoá, ghi nợ (§9.6 của brief).

10. Chạy bộ ca độc lập **một lần** ở mã đóng băng; báo cáo theo GA1–GA11. Không chỉnh lexicon theo bộ này sau khi chạy.
11. Ghi "PHASE 0A RA" (M2 ra khi GA1–GA6 đạt) với kết luận "trên dữ liệu giả lập, chưa có người nghiệp vụ duyệt".

---

## 15. Câu mở cho chủ dự án

Khuôn của `docs/CHO-QUYET-DINH.md`. Cột "Trả lời" **để trống**. Phiên điều phối chuyển các dòng này vào CHO khi gộp (tệp này không sửa CHO).

| Mã | Câu | Lựa chọn | Khuyến nghị (vì sao) | Chưa quyết thì | Trả lời |
|---|---|---|---|---|---|
| **Q-0A-1** | Ngưỡng GA6: đúng ≥ 97%, phủ nhận tự động ≥ 85% | **a** dùng số đề xuất · **b** sửa số · **c** bỏ dòng GA6 (chỉ giữ GA1–GA5) | **a** như điểm xuất phát, vì không có số nền. Cách ly hết thì sai im lặng bằng 0 nhưng phủ bằng 0 nên cần dòng phủ | dùng a (§9.7: ngưỡng đề xuất) | |
| **Q-0A-2** | Nghĩa hậu tố "k": D-OP-17 đã có trong CHO (mục 3, `docs/CHO-QUYET-DINH.md:133`): "2k" có tính là ước lượng không | theo D-OP-17: **a** "k" = ước lượng (sửa bộ nạp theo bộ sinh) · **b** "k" = số chính xác (sửa bộ sinh) | theo khuyến nghị của D-OP-17 (a); hợp đồng này không phụ thuộc: cờ `approx` của "2k" theo quyết định đó | giữ danh sách lệch đã biết của G1; bộ ca độc lập không có ca phụ thuộc | |
| **Q-0A-3** | Số ước lượng vào tổng "đội báo" hay tách riêng | **a** cộng, kèm cờ (hiện trạng) · **b** tách hai tổng | **a**: đường chuyển cờ tới câu trả lời đã có (`query.go:208-210`, `ask.go:289`) | giữ a | |
| **Q-0A-4** | Tin đính chính được áp khi nào | **a** luôn chờ người (đề xuất) · **b** tự áp khi có đúng một bản gốc ứng viên · **c** để Phase 3 | **a**: ít rủi ro nhất; giá là người xác nhận mọi đính chính (tần suất chưa đo) | a | |
| **Q-0A-5** | Định nghĩa "trùng": cửa sổ W và có tính người gửi khác nhau không | **a** W = 60 phút, bất kể người gửi (đề xuất) · **b** chỉ cùng người gửi · **c** W khác | **a**: an toàn; giá là có lần giữ oan khi hai lần cấp thật cùng số trong W (chưa đo tần suất) | a | |
| **Q-0A-6** | Ai nhận mục `HELD` và `QUARANTINED` | **a** `project_manager` chứa máy, không xác định được thì `director` (đề xuất) · **b** kế toán · **c** người gửi trước | **a**: khớp "duyệt báo cáo ngày" của trưởng dự án (`01-business-map.md:305`) | a | |
| **Q-0A-7** | Vai gửi báo cáo | **a** thêm `site_supervisor` vào `policy.yaml` · **b** dùng `site_engineer` · **c** cho cả `driver` máy của mình | **a**: khớp bản đồ nghiệp vụ (`01-business-map.md:308`) | a | |
| **Q-0A-8** | Lịch ca và giờ cắt ngày của ca đêm. BR-RPT-02 đã ghi "ca đêm tính vào ngày bắt đầu ca" nhưng pack chưa có lịch ca | **a** thêm lịch ca (giờ bắt đầu, giờ kết thúc từng ca) vào pack · **b** dùng ngày lịch (hiện trạng) | **a**: khớp BR-RPT-02; cần người nghiệp vụ cho giờ cụ thể | b; ca độc lập không phụ thuộc | |
| **Q-0A-9** | Tin nhiều máy hoặc nhiều lượng | **a** cách ly `multiple_items` (đề xuất) · **b** tách theo dấu phẩy hay xuống dòng khi mỗi đoạn đúng một máy một lượng | **a** trước; **b** sau khi đo tỉ lệ cách ly trên dữ liệu thật (chưa đo) | a | |
| **Q-0A-10** | Nhóm nghìn bằng dấu cách (D-OP-9, đã có) | theo CHO | xem D-OP-9 | ca độc lập không phụ thuộc | |
| **Q-0A-11** | Kênh cho nhánh form / bot | form có sẵn · bot Zalo OA · mini app · chưa chọn | chưa chọn: dựng nhánh dùng chung lõi, quyết sau số đo và chi phí đổi thói quen; phụ thuộc D-RM-03 | chưa chọn | |
| **Q-0A-12** | Mở rộng write-set làn 4 thêm `data/ingestion/`, `platform/pgstore/` | có · không (đặt chỗ khác) | có: mã 0a thuộc tầng nạp (`docs/01-architecture.md` §3) | điều phối làn 4 theo mặc định hiện hành, ghi mâu thuẫn vào báo cáo ca | |

---

## 16. Điểm chưa kiểm chứng

| Mã | Điều chưa biết |
|---|---|
| U1 | Mọi số [ĐO] ở §1.3 chạy đường bộ nhớ, một tenant giả lập, 20 tin; chưa đo PDF, xlsx, ảnh, tổng ở kho; K1–K5 đo ở `7cad584`. Tầng nạp (`data/ingestion`, `sourceoftruth`, `platform`, pack `sources` / `metrics`) không đổi từ `7cad584` tới `2b80a38`. |
| U2 | Cách phối hợp kho bản ghi với `Load` (thay trọn dữ liệu tenant, `load.go:21-48`) và với bảng vết của làn 0: chọn khi mở mã (ADR-0012, quyết định D1, D3). Bộ đếm "mục chưa dùng được" cho phía hỏi–đáp là yêu cầu cho làn đọc, chưa có. |
| U3 | Tải thật (số tin mỗi ngày, cỡ lô): chưa đo. Cửa sổ W và độ phủ 85% là điểm xuất phát. |
| U4 | Tần suất tin đính chính, kế hoạch, nghi trùng trong thực tế, và chi phí người xác nhận: chưa đo. |
| U5 | Hình dạng tin nhóm Zalo thật khi xuất (có mã tin, `reply_to`, sửa, thu hồi không): chưa biết; §2.2 là giả định. |
| U6 | "hơn" nằm trong `approx_markers` (`lexicon.yaml:26`) nhưng nghĩa thường là cận dưới, không phải ước lượng đối xứng: chưa xác minh; không thuộc hợp đồng này. |
| U7 | Tài liệu bot Zalo OA: thân trang không đọc được; chưa so (`CONG-QUANIT.md` §4). |
| U8 | Hợp đồng chưa được soát độc lập; các luật LK3, LT3, TR3 chưa kiểm bằng bộ ca nào. |
