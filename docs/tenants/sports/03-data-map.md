# 03 — Bản đồ dữ liệu (tenant `xyz-sports`)

> Mọi giải, đội, cầu thủ, provider, mã trận trong tài liệu là **hư cấu**. Mọi con số (độ trễ, số sự kiện, điểm, thời điểm) là **giả lập** hoặc **mục tiêu thiết kế**, chưa có phép đo nào. Chỉ dữ liệu thông tin thể thao; không có dữ liệu cá cược.

Tài liệu này kiểm kê dữ liệu của XYZ Sports Data JSC, nêu cách các provider nói khác nhau về cùng một trận, ma trận thẩm quyền **theo từng trường**, từ điển thực thể và schema chuẩn. Luồng xử lý ở [04-realtime-flow.md](04-realtime-flow.md); kịch bản ở [05-problem-catalog.md](05-problem-catalog.md).

Thuật ngữ lần đầu: *OBSERVATION* (quan sát: điều một nguồn đã nói), *MEASUREMENT* (đo lường: quan sát đã chuẩn hoá), *FACT* (sự thật đã hợp nhất, có phiên bản), *DERIVED FACT* (sự thật dẫn xuất, tính bằng công thức), *provisional* (tạm thời, chưa xác nhận), *disputed* (đang có nguồn bất đồng), *stale* (có thể đã cũ), *quorum* (số nguồn tối thiểu phải đồng ý).

---

## 1. Bốn tầng dữ liệu áp vào thể thao

```text
OBSERVATION     điều MỘT provider nói, nguyên văn + nguồn + thời điểm nhận        provider_events_raw / observations
   │              "FeedBeta: G, phút 47, Phố Núi Utd"
   ▼ interpretation (chuẩn hoá phút, múi giờ; nhận dạng đội, cầu thủ)
MEASUREMENT     sự kiện chuẩn của MỘT nguồn: kỳ, phút, bù giờ, loại, đội, cầu thủ  (vẫn trỏ về observation gốc)
   ▼ correlation (nhiều quan sát = cùng một sự cố?) + authority theo trường
FACT            sự kiện chuẩn hợp nhất, có phiên bản: GOAL v1, RETRACTION v2…       match_events
   ▼ projection (công thức ghi lại)
DERIVED FACT    tỉ số, trạng thái hiển thị, bảng xếp hạng, thống kê cầu thủ, phong độ
```

| Tầng | Thể thao | Ai được ghi | Sửa được không |
|---|---|---|---|
| OBSERVATION | payload thô của FeedAlpha/Beta/Gamma, biên bản liên đoàn | adapter provider | không; chỉ ghi thêm |
| MEASUREMENT | sự kiện chuẩn theo từng nguồn | `data/normalization`, `data/entityresolution` | không; chỉ ghi thêm |
| FACT | `match_events` hợp nhất, có `version`, `supersedes` | `domain/sportsevents` + `data/sourceoftruth` | **không sửa đè**; sửa = sự kiện mới trỏ về cũ |
| DERIVED FACT | `match_states` (tỉ số), `standings`, thống kê | projection (tính từ FACT) | không ghi tay; chỉ tính lại |

Hai luật: (1) **tỉ số và bảng xếp hạng là DERIVED FACT**, công thức: tỉ số = số bàn thắng hiệu lực (loại sự kiện đã bị rút/thay thế; bàn phản lưới cộng cho đội đối phương); bảng xếp hạng = điểm theo kết quả các trận `FINAL` + tiêu chí phụ của giải/mùa. (2) **Thẩm quyền và đồng thuận chỉ áp giữa các quan sát về CÙNG một sự kiện**; hai bàn thắng khác nhau không "xung đột". Việc nhận ra "các quan sát này nói về cùng một bàn" là bước tương quan, đi **trước** (xem *`incident_correlation`* ở [05 §5](05-problem-catalog.md#5-primitive-còn-thiếu-do-thể-thao-chỉ-ra)).

---

## 2. Kiểm kê dữ liệu

Cột "Freshness/độ trễ" là **mục tiêu thiết kế** (chưa đo). "Tầng" theo §1; `ref` = dữ liệu tham chiếu, `master` = dữ liệu chủ, `txn` = giao dịch.

| Dataset | Loại | Tầng | Owner · Producer → Consumer | Thẩm quyền | Freshness/độ trễ (mục tiêu) | Lưu giữ | PII · Quyền | Cập nhật |
|---|---|---|---|---|---|---|---|---|
| `provider_events_raw` | semi-structured, streaming | OBSERVATION | Tích hợp provider · adapter → normalization | thấp (chưa diễn giải) | gần thời gian thực | dài (phục vụ phát lại) | không PII · nội bộ | ghi thêm |
| `observations` (đã diễn giải) | structured | MEASUREMENT | Vận hành dữ liệu · normalization → sourceoftruth | theo hạng provider | vài giây | dài | không · nội bộ | ghi thêm |
| `match_events` | structured, streaming | FACT | Vận hành dữ liệu · `domain/sportsevents` → projection, API | cao nhất sau khi hợp nhất | vài giây | vĩnh viễn (event store) | không · đọc theo gói | ghi thêm, có phiên bản |
| `match_states` | structured, derived | DERIVED FACT | Kỹ thuật · projection → API, cache | dẫn xuất | vài giây | theo mùa | không · đọc theo gói | tính lại từ `match_events` |
| `standings` | structured, derived | DERIVED FACT | Vận hành dữ liệu · projection → API | dẫn xuất | sau mỗi sự kiện đổi kết quả | theo mùa | không · công khai | tính lại |
| `player_stats`, `team_form` | derived | DERIVED FACT | Sản phẩm · projection → API | dẫn xuất | sau trận | theo mùa | không · gói premium cho chi tiết | tính lại |
| `fixtures` | master/txn | FACT | Biên tập · lịch chính thức + provider → API | lịch chính thức > provider | phút | theo mùa | không · công khai | đồng bộ + lệnh đổi lịch |
| `competitions`, `seasons`, `stages_rounds` | master | FACT | Biên tập · lệnh/duyệt ánh xạ | liên đoàn | ngày | vĩnh viễn | không · công khai | lệnh, `mapping-review` |
| `teams`, `players`, `venues`, `officials` | master | FACT | Biên tập · liên đoàn + provider | liên đoàn | ngày | vĩnh viễn | **tên người thật-hư cấu**: dữ liệu công khai, không PII nhạy cảm | lệnh, gộp có redirect |
| `entity_aliases` | ref | FACT | Vận hành dữ liệu · entity resolution | theo nguồn + hiệu lực | ngay khi duyệt | vĩnh viễn | không · nội bộ | lệnh thêm, `mapping-review` |
| `player_affiliations` | ref, temporal | FACT | Biên tập · liên đoàn | liên đoàn | ngày | vĩnh viễn | không · công khai | ghi thêm theo hiệu lực |
| `league_official_reports` | semi-structured | MEASUREMENT (hạng 1) | Vận hành dữ liệu · liên đoàn → sourceoftruth | **cao nhất** | 15 đến 60 phút sau trận (giả lập) | vĩnh viễn | không · nội bộ | nhận một lần/trận |
| `matches_history` | historical | FACT | Vận hành dữ liệu · nhập lô | liên đoàn | theo mùa | vĩnh viễn | không · công khai | nhập lô có `result_type` |
| `rankings` | ref | FACT | Sản phẩm · liên đoàn từng môn | liên đoàn | hằng tuần | theo mùa | không · công khai | nhập lô |
| `localized_names` | ref | FACT | Biên tập · biên tập | biên tập | ngày | vĩnh viễn | không · công khai | lệnh |
| `editorial_articles` | unstructured | — | Biên tập · biên tập viên, model soạn nháp → search | không dùng làm bằng chứng cho dữ kiện live | phút | dài | bản quyền nội dung · theo gói | workflow `match-report` |
| `regulations_corpus` | unstructured | — | Biên tập · điều lệ, luật thi đấu → RAG | theo phiên bản có hiệu lực | theo mùa | mọi phiên bản | không · công khai | nạp phiên bản mới nguyên tử |
| `provider_contracts` | unstructured | — | Pháp chế · hợp đồng/giấy phép → tuân thủ | — | khi đổi hợp đồng | theo hợp đồng | **nội bộ, nhạy cảm thương mại** | nạp thủ công |
| `provider_health_metrics` | streaming | — | Kỹ thuật · adapter, trace → vận hành | đo trực tiếp | giây | ngắn | không · nội bộ | liên tục |
| `subscriptions`, `notification_log` | txn | — | Sản phẩm · app → thông báo | — | giây | ngắn | **mã thiết bị: PII nhẹ** · chủ tài khoản | lệnh `subscribe_team` |
| `api_clients`, `api_usage` | txn | — | Sản phẩm · quản trị → API gateway | — | giây | theo hợp đồng | **khoá API băm: nhạy cảm** | quản trị |
| `workflow_instances`, `human_tasks`, `commands`, `audit_log` | txn | — | Kỹ thuật · engine → vận hành | — | giây | dài | không · nội bộ | engine |
| `media_assets` (logo, ảnh) | multimodal | — | Biên tập · biên tập | — | — | — | giấy phép ảnh | **chưa dùng** (chưa có ca hỏng đòi thị giác máy) |

---

## 3. Ba provider nói về cùng một trận

Trận `MATCH_0000912`: Giải Vô địch Lạc Hồng, vòng 9, Sông Hồng FC (nhà) gặp Phố Núi United (khách), khai cuộc 2026-10-18T12:30:00Z (19:30 giờ Việt Nam), chung cuộc 2-1. Dữ liệu dưới đây là **mẫu giả lập**.

### 3.1 Bàn thắng phút 45+2' của Phố Núi United

FeedAlpha (đẩy qua kết nối giữ mở, JSON, UTC):

```json
{"feed":"FeedAlpha","seq":118,"event_id":"a-77120","match_id":"A-MX-55120","type":"goal",
 "team":{"id":"A-T-31","name":"PHO NUI UTD"},"player":{"id":"A-P-8841","name":"K. Anh Bao"},
 "time":{"period":"1H","minute":45,"added":2},"score_after":{"home":1,"away":1},
 "ts":"2026-10-18T13:16:45Z"}
```

FeedBeta (hỏi mỗi 10 giây, ảnh chụp trạng thái, giờ địa phương không offset, phút là số phút đã trôi):

```json
{"provider":"FeedBeta","fetched_at":"2026-10-18 20:17:10",
 "match":{"id":"b_9912","league":"Giải VĐQG Lạc Hồng","round":9,"home":"Sông Hồng","away":"Phố Núi Utd","status":"1"},
 "score":{"h":1,"a":1},
 "events":[{"i":0,"t":"G","min":23,"team":"Sông Hồng","player":"Lương Đình Hậu"},
           {"i":1,"t":"YC","min":38,"team":"Phố Núi Utd","player":"Tạ Văn Lộc"},
           {"i":2,"t":"G","min":47,"team":"Phố Núi Utd","player":"Kiều Anh Bao"}]}
```

FeedGamma (lô CSV mỗi 60 giây, epoch giây, mã trạng thái số, tên viết tắt, chỉ bàn thắng và thẻ):

```text
match_code,seq,type,side,minute,player,team_code,epoch,status
G:LHC:0912,3,GL,A,45,"BAO K.A.",CBA,1792329405,1
```

### 3.2 Chênh lệch cần xử lý

| Điểm khác | FeedAlpha | FeedBeta | FeedGamma | Xử lý |
|---|---|---|---|---|
| Tên đội | `PHO NUI UTD` | `Phố Núi Utd` | `PNU` (nhưng ghi nhầm đội của cầu thủ là `CBA`) | bảng `entity_aliases` theo nguồn → `TEAM_00102`; đội phải thuộc hai đội của trận |
| Phút | `{1H, 45, added 2}` | `47` (số phút đã trôi) | `45` (làm tròn xuống) | chuẩn hoá về (`FIRST_HALF`, clock 45, stoppage 2); Gamma gắn `minute_precision=coarse` |
| Giờ | ISO UTC `…Z` | `2026-10-18 20:17:10` không offset | epoch giây `1792329405` | UTC nội bộ; múi giờ của Beta khai báo trong pack; epoch phải nhận ra giây hay mili giây |
| Trùng bàn | id `a-77120`, có `score_after` | mục `i=2`, id đổi theo lượt chụp | dòng `seq 3` | một sự cố, ba tham chiếu provider (khoá tương quan) |
| Bàn bị huỷ (71', việt vị) | `VAR_REVIEW` rồi `goal_cancelled` trỏ `a-77130` | bàn **biến mất** khỏi ảnh chụp, tỉ số 2-1 → 1-1, không giải thích | không bao giờ rút; còn 2-1 thêm một lượt | `RETRACTION v2` trỏ về `GOAL v1`; quan sát hạng thấp không dựng lại bàn đã rút |
| Cầu thủ | id `A-P-8841` đã ánh xạ | `Kiều Anh Bao` thiếu dấu | `BAO K.A.` + đội cũ `CBA` | ánh xạ theo bí danh; đội theo quan hệ **có hiệu lực theo thời gian** (chuyển CLB 15/07/2026) |
| Tên trùng | — | — | `HAU L.` (hai người: Lương Đình Hậu 04571 và 09102) | ràng buộc "thuộc đội hình của trận"; dưới ngưỡng thì chỉ là ứng viên chờ duyệt |

### 3.3 Dòng thời gian chuẩn của trận (sự kiện hiệu lực sau khi hợp nhất, giả lập)

| Giờ trận | Loại | Đội | Cầu thủ | Ghi chú |
|---|---|---|---|---|
| 23' | GOAL | Sông Hồng FC | Lương Đình Hậu (kiến tạo Trương Quang Tín) | 1-0, UTC 12:52:10 |
| 38' | YELLOW_CARD | Phố Núi United | Tạ Văn Lộc | UTC 13:07:30 |
| 45+2' | GOAL | Phố Núi United | Kiều Anh Bảo | 1-1, UTC 13:16:45 |
| 58' | SUBSTITUTION | Sông Hồng FC | ra Trương Quang Tín, vào Mạc Văn Kiên | UTC 13:45:20 |
| 71' | GOAL → RETRACTION (74') | Sông Hồng FC | Lương Đình Hậu | huỷ sau VAR (việt vị), về 1-1 |
| 78' | RED_CARD | Phố Núi United | Vi Hồng Thạch | UTC 14:05:05 |
| 88' | GOAL | Sông Hồng FC | Mạc Văn Kiên | 2-1, UTC 14:15:20 |
| 90+4' | FULL_TIME | — | — | UTC 14:21:30 |

---

## 4. Lỗi dữ liệu theo mười một loại

| Loại | Ví dụ trong thể thao | Ca hỏng / kịch bản |
|---|---|---|
| duplicate (trùng) | cùng bàn thắng từ ba provider; phát lại sau kết nối lại | `SPT-F-001`, `SPT-F-002` · S-019, S-020 |
| conflict (xung đột) | người ghi bàn khác nhau; thẻ đỏ có/không | `SPT-F-029`, `SPT-F-030` · S-053, S-054 |
| stale (cũ) | snapshot chậm; provider im lặng | `SPT-F-007`, `SPT-F-008` · S-025, S-026 |
| missing (thiếu) | thay người chỉ có nửa "vào"; bàn không có người ghi | S-016 |
| invalid (không hợp lệ) | phút 130 trong trận không có hiệp phụ; phút bù giờ âm | `SPT-F-024` · T-079 |
| outlier (ngoại lai) | kiểm soát bóng 130% | `SPT-F-032` |
| schema drift (trôi schema) | `minute` đổi từ số sang chuỗi | `SPT-F-035` · S-027 |
| semantic drift (trôi ngữ nghĩa) | mã trạng thái 4: hiệp phụ ở v1, tạm dừng ở v2 | `SPT-F-028` |
| unit mismatch (lệch đơn vị) | epoch giây và mili giây; điểm quần vợt `40-15` là chuỗi | `SPT-F-026` |
| timezone mismatch (lệch múi giờ) | giờ địa phương không offset | `SPT-F-025` · S-033 |
| version mismatch (lệch phiên bản) | điều lệ mùa cũ; cầu thủ chưa cập nhật đội mới | `SPT-F-043`, `SPT-F-019` |

---

## 5. Ma trận thẩm quyền theo từng trường (Source Authority Matrix)

Hạng thẩm quyền (nhỏ = mạnh hơn): `EDITOR` 0 (chỉ qua workflow `correction-request`) · `LEAGUE_OFFICIAL` 1 · `FeedAlpha` 2 · `FeedBeta` 3 · `FeedGamma` 4. Luật chung: **authority ≠ recency ≠ similarity ≠ số lần lặp**; chỉ áp giữa các quan sát về cùng một sự kiện; ngưỡng thời gian bên dưới là mặc định cấu hình trong pack (**giả lập**).

| Trường | Chính sách | Chọn thế nào | Khi bất đồng | Chung cuộc |
|---|---|---|---|---|
| Tỉ số | **dẫn xuất**, không chọn nguồn | đếm bàn thắng hiệu lực; tỉ số provider chỉ để đối chiếu | lệch > 0 quá 60 giây thì cảnh báo và mở việc; không tự sửa | `LEAGUE_OFFICIAL`; bỏ cuộc/xử thua: quyết định chính thức |
| Có bàn thắng hay không | ưu tiên + đồng thuận | công bố khi FeedAlpha báo, hoặc hai provider khớp trong dung sai 2 phút | chỉ một nguồn hạng thấp: chờ nguồn thứ hai tối đa 30 giây rồi để `provisional` | biên bản chính thức |
| Người ghi bàn | ưu tiên theo hạng, kiểm danh tính | lấy nguồn hạng cao nhất có id ánh xạ độ tin cậy ≥ 0,90 và thuộc đội hình trận | nguồn thấp khác: giữ lựa chọn, gắn nhãn `disputed` | biên bản chính thức (có thể đổi thành phản lưới) |
| Phút (kỳ, phút, bù giờ) | ưu tiên theo cấu trúc | dạng có cấu trúc (FeedAlpha) thắng dạng số vô hướng; Beta cần kỳ mới chuẩn hoá được; Gamma chỉ để lấp chỗ trống | lệch > 2 phút thì gắn cờ, không tự chọn | biên bản chính thức |
| Thẻ vàng | đồng thuận nhẹ | FeedAlpha, hoặc hai nguồn khớp | chỉ một nguồn hạng thấp: `provisional` | biên bản chính thức |
| Thẻ đỏ | **đồng thuận bắt buộc** | cần ít nhất hai nguồn hoặc nguồn chính thức | đơn nguồn sau 120 giây: việc cho biên tập viên; hiển thị nhãn chưa xác nhận | biên bản chính thức |
| Đội hình | ưu tiên theo hạng, nguồn chính thức thắng | công bố khi có nguồn chính thức hoặc hai provider khớp | lệch một vị trí: `provisional` nêu vị trí bất đồng | `LEAGUE_OFFICIAL` thắng mọi provider |
| Trạng thái trận | máy trạng thái + quorum | FeedAlpha đủ nếu đồng hồ phù hợp; nếu không cần hai provider; chỉ đi tiến | nguồn đứng yên > 60 giây bị loại khỏi quorum và mở sự cố | `FINAL` khi hết cửa sổ muộn và đối soát xong |
| Giờ khai cuộc | lịch chính thức trước | lịch chính thức > FeedAlpha > FeedBeta > FeedGamma | lệch lịch chính thức: gắn cờ | lịch chính thức |
| Thống kê (kiểm soát bóng, cú sút) | **một nguồn chỉ định cho mỗi chỉ số**, không đồng thuận | FeedAlpha; FeedBeta chỉ khi thiếu | giá trị ngoài miền bị loại và ghi nhận | không có nguồn chính thức; ghi rõ nguồn khi hiển thị |

Cấu hình ưu tiên là **dữ liệu của pack**; đổi lúc chạy cần bản ghi đè có phiên bản (*`runtime_config_override`*, ca `SPT-S-042`).

---

## 6. Từ điển thực thể

Bảng nêu **biến thể mà provider hay gửi** (mẫu giả lập). Cột "Chuẩn" là khoá nội bộ; bí danh có nguồn, hiệu lực và độ tin cậy trong `entity_aliases`.

| Loại | Chuẩn | Bí danh / biến thể | Nguồn · ghi chú |
|---|---|---|---|
| Đội | `TEAM_00101` Sông Hồng FC | `SONG HONG FC`, `Sông Hồng`, `SHG`, `SHFC`, `Song Hong FC` | Alpha · Beta · Gamma |
| Đội | `TEAM_00102` Phố Núi United | `PHO NUI UTD`, `Phố Núi Utd`, `PNU`, `P.Núi`, `Pho Nui United FC` | `Pho Nui United FC` là ứng viên chờ duyệt |
| Đội | `TEAM_00103` Cảng Biển Athletic | `CANG BIEN ATH`, `Cảng Biển AC`, `CBA` | |
| Đội | `TEAM_00104` Thành Cổ City | `THANH CO CITY`, `Thành Cổ City`, `TCC` | `Thành Cổ` **mơ hồ** với TEAM_00109 |
| Đội | `TEAM_00109` Thành Cổ Youth | `THANH CO YOUTH`, `Thành Cổ U21`, `TCY` | đội U21 |
| Đội | `TEAM_00110` Sông Hồng FC B | `SONG HONG FC B`, `Sông Hồng B`, `Sông Hồng II`, `SHB` | đội dự bị, **khác** TEAM_00101 |
| Đội | `TEAM_00105` Đồng Bằng Rovers | `DONG BANG ROVERS`, `ĐB Rovers`, `DBR` | |
| Đội | `TEAM_00106` Hải Đăng FC | `HAI DANG FC`, `HDF` | |
| Đội | `TEAM_00107` Núi Xanh Wanderers | `NUI XANH WAND`, `NXW` | |
| Đội | `TEAM_00108` Lạc Việt Town | `LAC VIET TOWN`, `LVT` | |
| Đội | `TEAM_00201` Cửu Long Tigers | `CUU LONG TIGERS`, `Cửu Long`, `CLT` | bóng rổ |
| Đội | `TEAM_00202` Hương Giang Hawks | `HUONG GIANG HAWKS`, `HGH` | bóng rổ |
| Cầu thủ | `PLAYER_04571` Lương Đình Hậu | `L. Đình Hậu`, `Luong Dinh Hau`, `HAU L.D.`, `HAU L.` | `HAU L.` không có id, trùng PLAYER_09102 |
| Cầu thủ | `PLAYER_09102` Lương Đình Hậu (sinh 2006) | cùng tên với 04571 | Thành Cổ Youth; phân biệt bằng đội, năm sinh |
| Cầu thủ | `PLAYER_04572` Trương Quang Tín | `T. Quang Tín`, `Truong Quang Tin`, `TIN T.Q.` | |
| Cầu thủ | `PLAYER_04574` Mạc Văn Kiên | `M. Văn Kiên`, `MAC K.`, `Mac Van Kien` | |
| Cầu thủ | `PLAYER_05210` Kiều Anh Bảo | `K. Anh Bao`, `BAO K.A.`, `Kiều Anh Bao` | chuyển CBA → PNU ngày 15/07/2026 |
| Cầu thủ | `PLAYER_05211` Tạ Văn Lộc | `T. Văn Lộc`, `TA V.L.` | |
| Cầu thủ | `PLAYER_05212` Vi Hồng Thạch | `V. Hồng Thạch`, `VI H.T.` | |
| Cầu thủ | `PLAYER_05214` Phan Quốc Hải | `P. Quốc Hải`, `PHAN Q.H.` | |
| Cầu thủ | `PLAYER_06001` Orlan Veskoff | `O. Veskov`, `VESKOFF O.` | phiên âm khác nhau |
| Cầu thủ | `PLAYER_07001` Dario Quelmont | `D. Quelmont`, `QUELMONT D.` | |
| Cầu thủ | `PLAYER_20001` Phạm Bích Ngọc | `B. Ngoc Pham`, `PHAM B.N.` | quần vợt |
| Cầu thủ | `PLAYER_50001` Quách Minh Dương | `M. Duong Quach`, `QUACH M.D.` | cầu lông |
| Giải | `COMP_0001` Giải Vô địch Lạc Hồng | `LAC HONG CHAMPIONSHIP` (Alpha C-118), `Giải VĐQG Lạc Hồng`, `LHC`, `Sao Mai Championship` (Alpha C-221, từ 2026/27) | đổi tên thương mại |
| Giải | `COMP_0002` Cúp Lạc Hồng | `LAC HONG CUP`, `Cúp LH` | loại trực tiếp, có hiệp phụ |
| Giải | `COMP_0003` Giải Hạng Nhất Lạc Hồng | `LAC HONG DIV 1`, `Hạng Nhất` | |
| Giải | `COMP_0004` Giải Trẻ U21 Lạc Hồng | `LH U21`, `Giải trẻ` | |
| Giải | `COMP_0101` Giải Bóng rổ Cửu Long | `CUU LONG BASKETBALL`, `CLBL` | |
| Giải | `COMP_0201` Giải Quần vợt Mở rộng Phù Sa | `PHU SA OPEN` | |
| Giải | `COMP_0301` Giải Bóng chuyền Hải Âu | `HAI AU VOLLEY` | |
| Giải | `COMP_0401` Giải Bóng chày Đông Hải | `DONG HAI BASEBALL` | |
| Giải | `COMP_0501` Giải Cầu lông Mặt Trời | `MAT TROI BADMINTON` | |
| Mùa | `SEASON_0001_2026` | `2026/27`, `2026-27`, `2026` (mơ hồ) | |
| Mùa | `SEASON_0001_2025` | `2025/26`, `2025-26` | |
| Trạng thái | `SCHEDULED` | `NS`, `scheduled`, `0` (Gamma), `chưa đá` | |
| Trạng thái | `LIVE_1H` | `1H`, `1` (Beta), `1` (Gamma), `hiệp 1` | |
| Trạng thái | `HT` | `HT`, `half-time`, `2` (Gamma), `nghỉ giữa hiệp` | |
| Trạng thái | `LIVE_2H` | `2H`, `2` (Beta), `3` (Gamma), `hiệp 2` | |
| Trạng thái | `FT` | `FT`, `full-time`, `9` (Gamma), `hết giờ` | chưa phải `FINAL` |
| Trạng thái | `LIVE_ET1`, `LIVE_ET2` | `ET1`, `ET2`, `hiệp phụ`, `4` (Gamma v1) | |
| Trạng thái | `PENALTIES` | `PEN`, `luân lưu`, `5` (Gamma) | |
| Trạng thái | `SUSPENDED` | `SUSP`, `tạm dừng`, `4` (Gamma v2) | cùng mã 4, khác nghĩa theo phiên bản |
| Trạng thái | `POSTPONED` | `PST`, `hoãn`, `-1` (Gamma) | |
| Trạng thái | `ABANDONED` | `ABD`, `bỏ cuộc` | |
| Trạng thái | `CANCELLED` | `CANC`, `huỷ` | |
| Trạng thái | `FINAL` | — | nội bộ, không do provider gửi |
| Loại sự kiện | `GOAL` | `goal`, `G`, `GL`, `bàn thắng` | |
| Loại sự kiện | `OWN_GOAL` | `og`, `OG`, `phản lưới` | cộng cho đội đối phương |
| Loại sự kiện | `PENALTY_GOAL` | `pen_goal`, `PG`, `phạt đền vào` | |
| Loại sự kiện | `PENALTY_MISSED` | `pen_miss`, `PM`, `phạt đền hỏng` | |
| Loại sự kiện | `YELLOW_CARD` | `yellow`, `YC`, `Y`, `thẻ vàng` | |
| Loại sự kiện | `SECOND_YELLOW` | `2Y`, `yellow_red`, `thẻ vàng thứ hai` | |
| Loại sự kiện | `RED_CARD` | `red`, `RC`, `R`, `thẻ đỏ` | cần đồng thuận |
| Loại sự kiện | `SUBSTITUTION` | `sub`, `SUB`, `thay người` | cặp ra/vào |
| Loại sự kiện | `VAR_REVIEW` | `var`, `VAR_CHECK`, `xem lại VAR` | |
| Loại sự kiện | `RETRACTION` | `goal_cancelled`, `GC`, `XG` (Gamma); bàn biến mất khỏi snapshot (Beta) | sự kiện mới trỏ về sự kiện cũ |
| Loại sự kiện | `CORRECTION` | — | nội bộ; bản sửa đã công bố |
| Loại sự kiện | `KICK_OFF`, `PERIOD_END`, `FULL_TIME` | `ko`, `PERIOD_START`, `end_period`, `ft`, `MATCH_END` | điều khiển vòng đời |
| Loại sự kiện | `LINEUP` | `lineup`, `XI`, `đội hình` | |
| Loại sự kiện | `THREE_POINT_GOAL`, `FREE_THROW_GOAL` | `3PT`, `FT_GOAL` | bóng rổ |
| Kỳ | `FIRST_HALF` | `1H`, `1`, `hiệp 1`, `first half` | bóng đá |
| Kỳ | `SECOND_HALF` | `2H`, `2`, `hiệp 2` | bóng đá |
| Kỳ | `EXTRA_TIME_FIRST`, `EXTRA_TIME_SECOND` | `ET1`, `ET2`, `hiệp phụ 1`, `hiệp phụ 2` | Cúp có hiệp phụ |
| Kỳ | `PENALTY_SHOOTOUT` | `PEN`, `luân lưu` | |
| Kỳ (bóng rổ) | `Q1`…`Q4`, `OT1` | `quarter 1`, `hiệp 1`, `OT` | đồng hồ đếm ngược |
| Kỳ (quần vợt, bóng chuyền) | `SET_1`…`SET_5` | `S1`…`S5`, `ván 1`…`ván 5` | không có đồng hồ |
| Kỳ (bóng chày) | `INNING_n_TOP`, `INNING_n_BOTTOM` | `1T`, `1B`, `hiệp 1 trên`, `hiệp 1 dưới` | không có đồng hồ; có số out |
| Kỳ (cầu lông) | `GAME_1`…`GAME_3` | `G1`…`G3`, `ván 1`…`ván 3` | không có đồng hồ |
| Môn | `football` | `soccer`, `bóng đá` | phạm vi MVP |
| Môn | `basketball` | `BB`, `bóng rổ` | |
| Môn | `tennis` | `quần vợt` | |
| Môn | `volleyball` | `bóng chuyền` | |
| Môn | `baseball` | `bóng chày` | |
| Môn | `badminton` | `cầu lông` | |
| Bên | `home` | `H`, `nhà`, `home` | cờ nhà/khách của FeedGamma có thể đảo |
| Bên | `away` | `A`, `khách`, `away` | |
| Vị trí | `GK` | `G`, `thủ môn` | |
| Vị trí | `DF` | `D`, `hậu vệ` | |
| Vị trí | `MF` | `M`, `tiền vệ` | |
| Vị trí | `FW` | `F`, `tiền đạo` | |
| Gói API | `partner_basic`, `partner_premium` | `cơ bản`, `basic`, `premium`, `nâng cao` | quyết định trường được xem |

---

## 7. Chuẩn hoá phút

| Đầu vào | Điều kiện | Chuẩn |
|---|---|---|
| `{1H, 45, added 2}` | cấu trúc | `FIRST_HALF`, clock 45, stoppage 2, hiển thị `45+2'` |
| `47` | kèm kỳ `1H` | `FIRST_HALF`, clock 45, stoppage 2 |
| `47` | kèm kỳ `2H` | `SECOND_HALF`, clock 47, stoppage 0 |
| `47` | **không** kèm kỳ | không đoán: tra trạng thái trận; không xác định được thì `CLARIFICATION` hoặc cách ly |
| `90+3` | `2H` | `SECOND_HALF`, clock 90, stoppage 3 |
| `105+1`, `120+2` | hiệp phụ | `EXTRA_TIME_FIRST` clock 105, stoppage 1; `EXTRA_TIME_SECOND` clock 120, stoppage 2 |
| `Q4 00:42` (đếm ngược) | bóng rổ | kỳ `Q4`, `clock_direction=DOWN`, còn 42 giây |
| `130` hay bù giờ âm | ngoài biên kỳ | không hợp lệ: cách ly, không cắt về 90 |

---

## 8. Schema chuẩn (JSON Schema rút gọn)

Khoá tiếng Anh. Mọi bảng có `tenant_id`. Mọi thời điểm là UTC (`…Z`). Các schema dưới đây là **đề xuất** cho `schemas/` và migration; mã thật do người điều phối viết.

### 8.1 `team`

```json
{"$id":"team","type":"object","required":["team_id","tenant_id","sport","name","kind"],
 "properties":{"team_id":{"type":"string","pattern":"^TEAM_[0-9]{5}$"},"tenant_id":{"type":"string"},
  "sport":{"enum":["football","basketball","tennis","volleyball","baseball","badminton"]},
  "name":{"type":"string"},"short_name":{"type":"string"},
  "kind":{"enum":["first_team","reserve","youth","women","national"]},
  "parent_team_id":{"type":["string","null"]},"localized_names":{"type":"object"},
  "merged_into":{"type":["string","null"]}}}
```

### 8.2 `player`

```json
{"$id":"player","type":"object","required":["player_id","tenant_id","name"],
 "properties":{"player_id":{"type":"string","pattern":"^PLAYER_[0-9]{5}$"},"tenant_id":{"type":"string"},
  "name":{"type":"string"},"birth_date":{"type":["string","null"],"format":"date"},"position":{"type":["string","null"]},
  "affiliations":{"type":"array","items":{"type":"object","required":["team_id","valid_from"],
   "properties":{"team_id":{"type":"string"},"valid_from":{"type":"string","format":"date"},"valid_to":{"type":["string","null"],"format":"date"}}}},
  "merged_into":{"type":["string","null"]}}}
```

### 8.3 `match`

```json
{"$id":"match","type":"object","required":["match_id","tenant_id","competition_id","season_id","home_team_id","away_team_id","kickoff_utc","status"],
 "properties":{"match_id":{"type":"string","pattern":"^MATCH_[0-9]{7}$"},"tenant_id":{"type":"string"},
  "competition_id":{"type":"string"},"season_id":{"type":"string"},"round":{"type":["string","null"]},
  "home_team_id":{"type":"string"},"away_team_id":{"type":"string"},"venue_id":{"type":["string","null"]},
  "kickoff_utc":{"type":"string","format":"date-time"},
  "status":{"enum":["SCHEDULED","LIVE_1H","HT","LIVE_2H","FT","LIVE_ET1","ET_HT","LIVE_ET2","PENALTIES","FINAL","SUSPENDED","POSTPONED","ABANDONED","CANCELLED"]},
  "result_type":{"enum":["REGULAR","EXTRA_TIME","PENALTIES","AWARDED","VOID",null]},
  "locked":{"type":"boolean"},"state_version":{"type":"integer","minimum":0},
  "provider_refs":{"type":"array","items":{"type":"object","required":["provider","ref"],"properties":{"provider":{"type":"string"},"ref":{"type":"string"}}}}}}
```

### 8.4 `canonical_event` (sự kiện chuẩn)

```json
{"$id":"canonical_event","type":"object",
 "required":["event_id","logical_event_id","tenant_id","match_id","kind","period","version","status","provider_refs","ingested_at"],
 "properties":{"event_id":{"type":"string"},"logical_event_id":{"type":"string"},"tenant_id":{"type":"string"},
  "match_id":{"type":"string"},
  "kind":{"enum":["GOAL","OWN_GOAL","PENALTY_GOAL","PENALTY_MISSED","YELLOW_CARD","SECOND_YELLOW","RED_CARD","SUBSTITUTION","VAR_REVIEW","LINEUP","KICK_OFF","PERIOD_END","FULL_TIME","RETRACTION","CORRECTION","THREE_POINT_GOAL","FREE_THROW_GOAL"]},
  "period":{"type":"string"},"clock":{"type":["integer","null"]},"stoppage":{"type":"integer","minimum":0},
  "clock_direction":{"enum":["UP","DOWN","NONE"]},"minute_precision":{"enum":["MINUTE","SECOND","PERIOD_ONLY","NONE","coarse"]},
  "display_minute":{"type":["string","null"]},
  "team_id":{"type":["string","null"]},"player_id":{"type":["string","null"]},"related_player_id":{"type":["string","null"]},
  "payload":{"type":"object"},
  "version":{"type":"integer","minimum":1},"supersedes":{"type":["string","null"]},
  "status":{"enum":["provisional","confirmed","disputed","retracted","superseded","pending_review"]},
  "authority":{"type":"integer"},"confidence":{"type":"number","minimum":0,"maximum":1},
  "provider_refs":{"type":"array","minItems":1,"items":{"type":"object","required":["provider","ref","received_at"],
   "properties":{"provider":{"type":"string"},"ref":{"type":"string"},"provider_ts":{"type":["string","null"]},"received_at":{"type":"string"}}}},
  "occurred_at":{"type":["string","null"]},"ingested_at":{"type":"string"}}}
```

Ghi chú: `logical_event_id` là danh tính của sự cố thật (cùng một bàn thắng qua mọi phiên bản); mỗi lần sửa hoặc rút lại là một hàng mới có `version` lớn hơn và `supersedes` trỏ về hàng trước. `RETRACTION` không có người ghi bàn mới; hàng bị trỏ tới chuyển `superseded` trong **góc nhìn** hiệu lực, còn hàng gốc không bị sửa.

### 8.5 `standing_row`

```json
{"$id":"standing_row","type":"object","required":["competition_id","season_id","team_id","rank","played","won","drawn","lost","goals_for","goals_against","points","as_of"],
 "properties":{"competition_id":{"type":"string"},"season_id":{"type":"string"},"stage_id":{"type":["string","null"]},
  "team_id":{"type":"string"},"rank":{"type":"integer","minimum":1},
  "played":{"type":"integer"},"won":{"type":"integer"},"drawn":{"type":"integer"},"lost":{"type":"integer"},
  "goals_for":{"type":"integer"},"goals_against":{"type":"integer"},"goal_difference":{"type":"integer"},"points":{"type":"integer"},
  "form_last5":{"type":"string","pattern":"^[WDL]{0,5}$"},"tiebreak_trace":{"type":"array","items":{"type":"string"}},
  "formula":{"type":"string"},"input_match_ids":{"type":"array","items":{"type":"string"}},"as_of":{"type":"string","format":"date-time"}}}
```

Ví dụ (giả lập, sau vòng 9):

| rank | team | played | won | drawn | lost | goals_for | goals_against | goal_difference | points |
|---|---|---|---|---|---|---|---|---|---|
| 1 | `TEAM_00101` | 9 | 6 | 2 | 1 | 17 | 8 | +9 | 20 |
| 2 | `TEAM_00102` | 9 | 5 | 3 | 1 | 14 | 8 | +6 | 18 |
| 3 | `TEAM_00103` | 9 | 5 | 2 | 2 | 13 | 9 | +4 | 17 |
| 4 | `TEAM_00104` | 9 | 4 | 4 | 1 | 12 | 8 | +4 | 16 |

Công thức điểm trong ví dụ: thắng 3, hoà 1, thua 0; tiêu chí phụ theo điều lệ của giải/mùa (ví dụ: đối đầu, rồi hiệu số, rồi bàn thắng). Hai ca `SPT-F-033` và `SPT-F-016` canh giữ việc dùng đúng tiêu chí và việc bảng không cũ.

---

## 9. Lineage, phiên bản, lưu giữ

- Mọi hàng `match_events` truy được về `provider_events_raw` qua `provider_refs` (ca hỏng `SPT-F-068`).
- `state_version` của `match_states` tăng mỗi khi projection đổi; cache khoá theo `(tenant, match_id, state_version)`.
- Hợp đồng adapter có **phiên bản** (ví dụ bảng mã trạng thái FeedGamma v1/v2); payload mang phiên bản nào thì dùng bảng của phiên bản đó.
- Retention: `provider_events_raw` và `match_events` giữ để phát lại; `provider_health_metrics` giữ ngắn. Con số lưu giữ cụ thể: **chưa quyết** (chưa đo dung lượng).
