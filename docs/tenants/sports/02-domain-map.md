# 02 — Bản đồ nghiệp vụ theo domain (tenant `xyz-sports`)

> Mọi giải, đội, cầu thủ, provider là **hư cấu**; mọi con số (phút, giây, điểm, số sự kiện) là **giả lập** để minh hoạ, chưa đo. Chỉ dữ liệu thông tin thể thao: không có tính năng cá cược.

Tài liệu chia nghiệp vụ thể thao thành **26 domain** (24 domain tối thiểu của yêu cầu nền cộng `regulations` và `compliance`). Mỗi domain có đủ **10 loại ca** (normal, edge, failure, dirty data, conflicting, missing, delayed, unauthorized, human error, system failure), mỗi loại **một dòng cụ thể**. Slug của domain là khoá `domain:` trong [cases.yaml](../../../tenants/sports/evals/cases.yaml). Dữ liệu và thẩm quyền ở [03-data-map.md](03-data-map.md); luồng ở [04-realtime-flow.md](04-realtime-flow.md); kịch bản ở [05-problem-catalog.md](05-problem-catalog.md).

## 1. Mô hình thi đấu theo môn

Bóng đá là phạm vi MVP; các môn khác mô tả để thấy mô hình **kỳ, đồng hồ, điểm số** khác nhau (nên "phút" của bóng đá không tái dùng được):

| Môn | Kỳ | Đồng hồ | Điểm | Sự kiện đặc trưng |
|---|---|---|---|---|
| Bóng đá | hiệp 1, nghỉ, hiệp 2, hiệp phụ, luân lưu | đếm xuôi, **có bù giờ** (`45+2'`) | số bàn thắng | bàn thắng, thẻ, thay người, VAR |
| Bóng rổ | 4 hiệp, hiệp phụ | **đếm ngược** theo hiệp | 1, 2, 3 điểm mỗi lần | ném 3 điểm, ném phạt, thời gian nghỉ |
| Quần vợt | set (3 hoặc 5) | **không có** | điểm → game → set; tie-break | giao bóng, break, bỏ cuộc |
| Bóng chuyền | set (tối đa 5) | **không có** | điểm mỗi pha bóng, đến 25 (set 5: 15), cách biệt 2 | đổi giao bóng, thời gian nghỉ |
| Bóng chày | inning (trên/dưới) | **không có** | số lần chạy về; số out | out, chạy về, đổi hiệp |
| Cầu lông | ván (tối đa 3) | **không có** | đến 21, cách biệt 2, trần 30 | điểm sau mỗi pha, đổi sân |

Hệ quả: `period`, `clock`, `stoppage`, `clock_direction` của sự kiện chuẩn là thuộc tính **theo môn**; mô hình tính điểm là **mã** (domain module), không phải cấu hình YAML (P12).

## 2. Quy trình nghiệp vụ (`business_process`)

| Quy trình | Mô tả | Domain chính |
|---|---|---|
| `provider-ingestion`, `provider-health`, `provider-configuration`, `provider-reconciliation`, `incident-management` | nhận, giám sát, cấu hình, hoà giải, xử lý sự cố provider | `provider_integration`, `cards`, `goals` |
| `event-normalization`, `event-deduplication`, `late-event-handling`, `correction-propagation` | chuẩn hoá, khử trùng, sự kiện muộn, lan truyền bản sửa | `event_timeline`, `goals`, `var_correction` |
| `entity-mapping`, `entity-clarification`, `entity-maintenance`, `mapping-review`, `competition-mapping` | nhận dạng và duyệt ánh xạ thực thể | `team_identity`, `player_identity`, `competition_mapping`, `season_mapping` |
| `live-score-query`, `match-events-query`, `standings-query`, `fixture-lookup`, `historical-lookup`, `form-analysis`, `rules-lookup` | trả lời câu hỏi | các domain đọc |
| `match-lifecycle`, `match-control`, `fixture-sync` | vòng đời trận, khoá, đổi lịch | `live_match`, `fixture` |
| `data-correction`, `correction-approval`, `official-reconciliation` | sửa dữ liệu và đối chiếu biên bản | `data_correction` |
| `match-summary`, `match-reporting`, `match-explanation`, `localization`, `content-search`, `notification-subscription` | nội dung, đa ngôn ngữ, tìm kiếm, thông báo | `content_generation`, `ai_explanation`, `localization`, `search`, `notification` |
| `access-control`, `projection-maintenance` | quyền, chạy lại projection | `compliance`, `event_timeline` |

---

## 3. Các domain (1–9)

### 3.1 `provider_integration` — tích hợp provider

Phạm vi: adapter, giám sát nhịp, hợp đồng payload, cấu hình thẩm quyền. Chủ sở hữu: Tích hợp provider. Kịch bản: 015, 020, 026, 027, 042, 050. Ca hỏng: `SPT-F-002`, `SPT-F-008`, `SPT-F-009`, `SPT-F-035`, `SPT-F-037`–`SPT-F-039`.

| Loại ca | Tình huống cụ thể |
|---|---|
| Normal | FeedAlpha gửi 120 sự kiện của một trận vòng 9 đúng thứ tự, mỗi sự kiện có `seq` liền nhau |
| Edge | FeedBeta đang bị giới hạn tần suất nên chỉ trả lời mỗi 30 giây thay vì 10 giây |
| Failure | khoá API FeedAlpha hết hạn lúc 14:00Z, mọi yêu cầu trả 401 |
| Dirty data | FeedGamma gửi lô CSV bị cụt dòng cuối giữa lúc ghi |
| Conflicting | hai provider đồng ý có bàn thắng 23' nhưng khác tên người ghi |
| Missing | FeedBeta thôi gửi trường `period`, chỉ còn `minute` |
| Delayed | FeedGamma trễ ổn định khoảng một phút so với FeedAlpha |
| Unauthorized | khách API gói cơ bản đòi payload gốc của FeedAlpha |
| Human error | kỹ sư cấu hình nhầm múi giờ của FeedBeta là UTC thay vì giờ Việt Nam |
| System failure | worker nạp bị tắt đột ngột giữa lô, worker mới nhận lại cùng lô |

### 3.2 `competition_mapping` — ánh xạ giải đấu

Phạm vi: liên kết mã giải của provider với `COMP_*`, qua đổi tên. Chủ sở hữu: Biên tập. Kịch bản: 032. Ca hỏng: `SPT-F-020`.

| Loại ca | Tình huống cụ thể |
|---|---|
| Normal | mã `C-118` của FeedAlpha ánh xạ `COMP_0001` Giải Vô địch Lạc Hồng |
| Edge | một provider gộp Giải Hạng Nhất và Giải Trẻ U21 vào một mã chung |
| Failure | giải đổi tên thương mại bị coi là giải mới, bảng xếp hạng tách đôi |
| Dirty data | tên giải viết `LAC HONG CHAMPIONSHIP` lúc hoa lúc thường lẫn khoảng trắng thừa |
| Conflicting | FeedAlpha dùng mã `C-221`, FeedBeta vẫn dùng tên cũ cho cùng giải |
| Missing | provider gửi trận không kèm mã giải |
| Delayed | tên thương mại mới chỉ được liên đoàn công bố sau vòng đầu của mùa |
| Unauthorized | nhân viên hỗ trợ cố thêm ánh xạ giải |
| Human error | người duyệt gán `C-221` vào Giải Hạng Nhất thay vì Giải Vô địch |
| System failure | hàng đợi duyệt ánh xạ mất một mục do lỗi ghi |

### 3.3 `season_mapping` — ánh xạ mùa giải

Phạm vi: mùa vắt qua hai năm, nhãn `2026/27`. Chủ sở hữu: Biên tập. Kịch bản liên quan: 008. Ca hỏng: `SPT-F-020`.

| Loại ca | Tình huống cụ thể |
|---|---|
| Normal | mùa `2026/27` của Giải Vô địch Lạc Hồng bắt đầu 15/08/2026 |
| Edge | hỏi "mùa 2026" khi có cả mùa 2025/26 vừa kết thúc và 2026/27 đang diễn ra |
| Failure | trận tháng 01/2027 bị gán vào mùa theo năm dương lịch 2027 |
| Dirty data | FeedBeta gửi mùa là `2026` thay vì `2026/27` |
| Conflicting | hai provider gán cùng một trận vào hai mùa khác nhau |
| Missing | provider không gửi mùa cho trận giao hữu đầu mùa |
| Delayed | mùa mới được tạo sau khi vòng 1 đã nhận sự kiện |
| Unauthorized | khách API đòi bảng xếp hạng của mùa chưa công bố |
| Human error | biên tập viên tạo nhầm mùa `2026/28` |
| System failure | job chuyển mùa dừng giữa chừng, nửa số trận ở mùa cũ |

### 3.4 `team_identity` — nhận dạng đội

Phạm vi: bí danh theo provider, đội một và đội dự bị. Chủ sở hữu: Vận hành dữ liệu. Kịch bản: 007, 017, 039, 051. Ca hỏng: `SPT-F-017`, `SPT-F-021`, `SPT-F-022`.

| Loại ca | Tình huống cụ thể |
|---|---|
| Normal | `SONG HONG FC`, `Sông Hồng`, `SHG` cùng trỏ `TEAM_00101` |
| Edge | `Sông Hồng FC B` (đội dự bị) khác `Sông Hồng FC` dù tên gần giống |
| Failure | bí danh `Thành Cổ` trỏ cả Thành Cổ City và Thành Cổ Youth |
| Dirty data | FeedGamma viết `PNU` rồi `P.Nui` rồi `Pho Nui Utd` trong cùng một lô |
| Conflicting | hai provider gán cùng mã đội cho hai đội khác nhau |
| Missing | trận mới có đội chưa có bí danh nào |
| Delayed | bí danh `Pho Nui United FC` chờ người duyệt hai ngày |
| Unauthorized | người hâm mộ thử thêm bí danh đội qua lệnh |
| Human error | vận hành thêm nhầm bí danh `Thành Cổ` cho đội U21 |
| System failure | bảng bí danh không nạp được lúc khởi động, hệ thống từ chối chạy |

### 3.5 `player_identity` — nhận dạng cầu thủ

Phạm vi: bí danh, trùng tên, chuyển CLB, gộp bản ghi. Chủ sở hữu: Vận hành dữ liệu. Kịch bản: 008, 028, 029, 036. Ca hỏng: `SPT-F-018`, `SPT-F-019`, `SPT-F-023`.

| Loại ca | Tình huống cụ thể |
|---|---|
| Normal | `A-P-8841` của FeedAlpha ánh xạ `PLAYER_05210` Kiều Anh Bảo |
| Edge | hai cầu thủ cùng tên Lương Đình Hậu (tiền đạo Sông Hồng FC, tiền vệ Thành Cổ Youth) |
| Failure | `HAU L.` được tự chấp nhận vào cầu thủ trẻ vì độ giống 0,70 |
| Dirty data | `Kiều Anh Bao` thiếu dấu, `BAO K.A.` viết tắt, cùng một người |
| Conflicting | FeedGamma ghi Kiều Anh Bảo thuộc Cảng Biển Athletic, FeedAlpha ghi Phố Núi United |
| Missing | provider chỉ gửi số áo, không có tên |
| Delayed | quan hệ chuyển CLB 15/07/2026 đến sau trận đầu của đội mới |
| Unauthorized | nhân viên hỗ trợ cố gộp hai cầu thủ |
| Human error | gộp nhầm `PLAYER_04571` với `PLAYER_09102` |
| System failure | phép gộp dừng giữa chừng, một nửa sự kiện đã chuyển hướng |

### 3.6 `fixture` — trận đã xếp lịch

Phạm vi: giờ khai cuộc, sân, trọng tài, hoãn, đổi giờ. Chủ sở hữu: Biên tập. Kịch bản: 033, 040, 046. Ca hỏng: `SPT-F-025`, `SPT-F-026`.

| Loại ca | Tình huống cụ thể |
|---|---|
| Normal | `MATCH_0000912` khai cuộc 2026-10-18T12:30:00Z tại Sân vận động Hồng Hà |
| Edge | trận khai cuộc 23:30 giờ Việt Nam, thuộc ngày 18 theo giờ Việt Nam nhưng ngày 18 theo UTC là 16:30 |
| Failure | giờ không offset của FeedBeta bị coi là UTC, khai cuộc lệch 7 giờ |
| Dirty data | FeedGamma gửi epoch mili giây thay vì giây |
| Conflicting | lịch chính thức 12:30Z, FeedBeta quy ra 13:30Z |
| Missing | trận chưa có sân hay trọng tài |
| Delayed | liên đoàn công bố hoãn trận 3 giờ trước giờ đá |
| Unauthorized | khách API cố đổi giờ trận |
| Human error | biên tập viên nhập 20:00 ngày 19/10 thành 20:00 ngày 18/10 |
| System failure | dịch vụ thông báo đổi giờ không gửi được, thông báo nằm lại hàng đợi |

### 3.7 `live_match` — trận đang diễn ra

Phạm vi: tỉ số, trạng thái, đồng hồ, vòng đời trận; các môn khác. Chủ sở hữu: Vận hành dữ liệu. Kịch bản: 001, 014, 025, 031, 035, 044, 045, 047, 055. Ca hỏng: `SPT-F-007`, `SPT-F-010`, `SPT-F-014`, `SPT-F-027`, `SPT-F-039`, `SPT-F-056`.

| Loại ca | Tình huống cụ thể |
|---|---|
| Normal | `MATCH_0000912` đi `SCHEDULED → LIVE_1H → HT → LIVE_2H → FT → FINAL`, tỉ số 2-1 |
| Edge | trận bóng rổ vào hiệp phụ khi đồng hồ đếm ngược chạm 00:00 hiệp 4 |
| Failure | trạng thái nhảy lùi từ `FT` về `LIVE_2H` vì snapshot cũ |
| Dirty data | quần vợt gửi `6-4, 3-6, 2-1` ở FeedBeta, `6-4 3-6 2-1*` ở FeedAlpha |
| Conflicting | FeedAlpha `FT`, FeedBeta còn `LIVE` phút 90, FeedGamma `HT` |
| Missing | provider không gửi nhịp đồng hồ trong 90 giây |
| Delayed | snapshot FeedGamma tạo lúc 13:58:30Z đến lúc 14:00:10Z |
| Unauthorized | người hâm mộ gửi lệnh khoá trận |
| Human error | biên tập viên khoá nhầm trận đang diễn ra |
| System failure | worker chết sau khi ghi sự kiện nhưng trước khi cập nhật `match_states` |

### 3.8 `event_timeline` — dòng thời gian sự kiện

Phạm vi: thứ tự theo giờ trận, phút bù giờ, đến muộn, projection. Chủ sở hữu: Kỹ thuật. Kịch bản: 018, 022, 023, 041. Ca hỏng: `SPT-F-004`, `SPT-F-005`, `SPT-F-011`, `SPT-F-024`.

| Loại ca | Tình huống cụ thể |
|---|---|
| Normal | bảy sự kiện của `MATCH_0000912` sắp theo giờ trận từ 23' đến 90+4' |
| Edge | phút 47 ở hiệp 1 là `45+2'`, phút 47 ở hiệp 2 là `47'` |
| Failure | thẻ đỏ 78' đến trước bàn 71' và làm timeline đảo thứ tự |
| Dirty data | phút `130` trong trận không có hiệp phụ |
| Conflicting | FeedAlpha ghi 45+2', FeedBeta ghi 47 ở hiệp 1, FeedGamma ghi 45 |
| Missing | sự kiện không có `period` nên không xếp được |
| Delayed | bàn 88' đến sau còi mãn cuộc 40 giây |
| Unauthorized | khách API gói cơ bản đòi toàn bộ sự kiện kèm tham chiếu provider |
| Human error | vận hành bấm chạy lại projection cả mùa khi chỉ cần một trận |
| System failure | một lô chạy lại projection lỗi giữa chừng |

### 3.9 `lineup` — đội hình

Phạm vi: đội hình xuất phát và dự bị, sơ đồ, thay đổi trước giờ đá. Chủ sở hữu: Biên tập. Kịch bản: 056. Ca hỏng: `SPT-F-060`.

| Loại ca | Tình huống cụ thể |
|---|---|
| Normal | FeedAlpha và FeedBeta cùng liệt kê mười một cầu thủ xuất phát của Phố Núi United |
| Edge | thủ môn Hứa Thanh Bình bị đổi trước giờ đá 10 phút |
| Failure | đội hình của một provider được công bố như đã xác nhận dù lệch một vị trí |
| Dirty data | số áo trùng nhau giữa hai cầu thủ trong cùng đội hình |
| Conflicting | FeedAlpha có Vi Hồng Thạch, FeedBeta và biên bản liên đoàn có Phan Quốc Hải |
| Missing | provider chỉ gửi đội hình xuất phát, thiếu ghế dự bị |
| Delayed | biên bản đội hình chính thức đến sau giờ khai cuộc 5 phút |
| Unauthorized | khách gói cơ bản đòi sơ đồ và số áo chi tiết (trường gói cao) |
| Human error | biên tập viên nhập nhầm cầu thủ dự bị vào đội hình xuất phát |
| System failure | việc nạp đội hình thất bại, hệ thống hiển thị đội hình cũ kèm nhãn `stale` |

## 4. Các domain (10–18)

### 4.1 `substitution` — thay người

Phạm vi: cặp ra/vào, hợp lệ theo đội hình. Chủ sở hữu: Vận hành dữ liệu. Kịch bản: 016.

| Loại ca | Tình huống cụ thể |
|---|---|
| Normal | phút 58 Sông Hồng FC: Trương Quang Tín ra, Mạc Văn Kiên vào |
| Edge | thay người ngay trước còi hết hiệp 1, tại `45+1'` |
| Failure | hệ thống tạo hai sự kiện thay người từ hai provider cho cùng một lần thay |
| Dirty data | FeedGamma chỉ gửi nửa "vào" cho `MAC K.` |
| Conflicting | FeedAlpha ghi Tín ra, FeedBeta ghi Phan Quốc Hải ra |
| Missing | provider thiếu cầu thủ "ra" |
| Delayed | sự kiện thay người đến sau bàn thắng mà người vào ghi |
| Unauthorized | người hâm mộ gửi lệnh sửa cặp thay người |
| Human error | biên tập viên chọn nhầm cầu thủ "ra" đang không trên sân |
| System failure | ghép cặp thất bại vì đội hình trên sân chưa nạp |

### 4.2 `cards` — thẻ phạt

Phạm vi: thẻ vàng, thẻ vàng thứ hai, thẻ đỏ, đồng thuận. Chủ sở hữu: Vận hành dữ liệu. Kịch bản: 030, 054. Ca hỏng: `SPT-F-029`.

| Loại ca | Tình huống cụ thể |
|---|---|
| Normal | thẻ vàng 38' của Tạ Văn Lộc được hai provider xác nhận |
| Edge | thẻ vàng thứ hai chuyển thành thẻ đỏ trong cùng một phút |
| Failure | thẻ đỏ đơn nguồn được công bố như đã xác nhận |
| Dirty data | mã thẻ `R` của FeedBeta lẫn với `RC` của FeedAlpha |
| Conflicting | FeedAlpha có thẻ đỏ 78', FeedBeta và FeedGamma không có |
| Missing | thẻ không kèm cầu thủ nhận |
| Delayed | thẻ đỏ đến muộn 40 giây so với diễn biến |
| Unauthorized | khách API đòi lịch sử kỷ luật chi tiết ngoài gói |
| Human error | biên tập viên gán thẻ đỏ cho cầu thủ cùng số áo của đội kia |
| System failure | việc chờ quorum hết thời gian mà hàng đợi biên tập không nhận được việc |

### 4.3 `goals` — bàn thắng

Phạm vi: bàn thắng, phản lưới, phạt đền, người ghi bàn. Chủ sở hữu: Vận hành dữ liệu. Kịch bản: 002, 019, 053. Ca hỏng: `SPT-F-001`, `SPT-F-030`, `SPT-F-034`.

| Loại ca | Tình huống cụ thể |
|---|---|
| Normal | bàn 23' của Lương Đình Hậu (kiến tạo Trương Quang Tín), tỉ số 1-0 |
| Edge | hai bàn của cùng cầu thủ trong 90 giây, không được gộp thành một |
| Failure | ba provider báo cùng một bàn và bị đếm thành ba bàn |
| Dirty data | `GL`, `G`, `goal` cùng nghĩa bàn thắng |
| Conflicting | người ghi bàn: Hậu (FeedAlpha), Tín (FeedBeta), `HAU L.` (FeedGamma) |
| Missing | bàn thắng không có người ghi |
| Delayed | bàn 88' đến sau còi mãn cuộc |
| Unauthorized | người dùng cố sửa người ghi bàn qua câu hỏi |
| Human error | biên tập viên nhập bàn cho đội nhà thay vì đội khách |
| System failure | worker chết giữa hai bàn, projection thiếu một bàn |

### 4.4 `var_correction` — sửa kiểu VAR

Phạm vi: xem lại VAR, huỷ bàn, công nhận lại. Chủ sở hữu: Vận hành dữ liệu. Kịch bản: 021. Ca hỏng: `SPT-F-003`, `SPT-F-012`, `SPT-F-013`, `SPT-F-015`, `SPT-F-059`.

| Loại ca | Tình huống cụ thể |
|---|---|
| Normal | bàn 71' bị huỷ vì việt vị, tỉ số về 1-1 |
| Edge | bàn bị huỷ rồi được công nhận lại sau khi xem lại lần hai (ba phiên bản) |
| Failure | sự kiện rút lại đến nhưng cache vẫn hiển thị 2-1 |
| Dirty data | FeedBeta xoá bàn khỏi ảnh chụp mà không có sự kiện giải thích |
| Conflicting | FeedGamma vẫn giữ bàn 2-1 thêm một lượt |
| Missing | VAR bắt đầu nhưng không có sự kiện kết thúc |
| Delayed | quyết định VAR đến sau bàn kế tiếp |
| Unauthorized | nhân viên hỗ trợ cố rút bàn thắng trực tiếp |
| Human error | biên tập viên rút nhầm bàn 88' thay vì 71' |
| System failure | một instance mất tín hiệu vô hiệu cache, hai người thấy hai tỉ số |

### 4.5 `statistics` — thống kê

Phạm vi: kiểm soát bóng, cú sút, phong độ, thống kê cầu thủ. Chủ sở hữu: Sản phẩm. Kịch bản: 006. Ca hỏng: `SPT-F-032`.

| Loại ca | Tình huống cụ thể |
|---|---|
| Normal | phong độ 5 trận của Phố Núi United: 3 thắng, 1 hoà, 1 thua, ghi 8 bàn, thủng lưới 5 |
| Edge | đội mới chỉ có hai trận, không đủ năm |
| Failure | trung bình kiểm soát bóng tính cả giá trị 130% |
| Dirty data | kiểm soát bóng gửi dạng `0.52` ở nguồn này, `52%` ở nguồn kia |
| Conflicting | FeedAlpha ghi kiểm soát bóng 54%, FeedBeta 61% cho cùng trận |
| Missing | trận không có số cú sút |
| Delayed | thống kê cuối trận đến sau kết quả 10 phút |
| Unauthorized | gói cơ bản đòi thống kê chi tiết theo cầu thủ |
| Human error | biên tập viên nhập kiểm soát bóng của hai đội không cộng 100% |
| System failure | job tính thống kê mùa dừng giữa chừng |

### 4.6 `standings` — bảng xếp hạng

Phạm vi: điểm, tiêu chí phụ, tính lại sau sửa. Chủ sở hữu: Vận hành dữ liệu. Kịch bản: 003. Ca hỏng: `SPT-F-016`, `SPT-F-033`.

| Loại ca | Tình huống cụ thể |
|---|---|
| Normal | sau vòng 9: Sông Hồng FC 20 điểm, Phố Núi United 18 điểm |
| Edge | hai đội bằng điểm, xếp theo đối đầu trước hiệu số |
| Failure | bảng dùng sai tiêu chí phụ của mùa 2026/27 |
| Dirty data | trận hoãn vẫn được tính điểm |
| Conflicting | bảng của provider khác bảng tính từ sự kiện |
| Missing | một trận của vòng chưa có kết quả `FINAL` |
| Delayed | bản sửa kết quả đến sau khi bảng đã công bố |
| Unauthorized | khách API đòi bảng của mùa chưa công bố |
| Human error | quy tắc xếp hạng của giải bị nhập thiếu một tiêu chí phụ |
| System failure | job tính lại bảng dừng giữa chừng, bảng nửa cũ nửa mới |

### 4.7 `ranking` — thứ hạng (theo tuần, theo môn)

Phạm vi: thứ hạng quần vợt, cầu lông do liên đoàn từng môn công bố. Chủ sở hữu: Sản phẩm.

| Loại ca | Tình huống cụ thể |
|---|---|
| Normal | Phạm Bích Ngọc hạng 12 đơn nữ tuần này (số giả lập) |
| Edge | hai tay vợt bằng điểm, cùng hạng |
| Failure | trả thứ hạng tuần trước cho câu hỏi tuần này |
| Dirty data | tên tay vợt viết `PHAM B.N.` và `B. Ngoc Pham` |
| Conflicting | hai nguồn ghi hạng khác nhau cho cùng tuần |
| Missing | tay vợt mới chưa có hạng |
| Delayed | bảng hạng tuần chỉ về sau thứ Hai |
| Unauthorized | khách gói cơ bản đòi lịch sử hạng ba năm |
| Human error | nhập nhầm môn (cầu lông thay quần vợt) |
| System failure | job nhập bảng hạng hằng tuần lỗi |

### 4.8 `schedule` — lịch thi đấu theo ngày, đội, giải

Phạm vi: lọc và hiển thị lịch, múi giờ người xem. Chủ sở hữu: Biên tập. Kịch bản: 004. Ca hỏng: `SPT-F-025`.

| Loại ca | Tình huống cụ thể |
|---|---|
| Normal | "Tối nay Sông Hồng FC đá lúc 19:30 giờ Việt Nam" |
| Edge | trận 23:30 giờ Việt Nam được hỏi bởi người ở múi giờ UTC (ngày khác) |
| Failure | lịch hiển thị 02:30 hôm sau vì coi giờ địa phương là UTC |
| Dirty data | ngày `18/10` lẫn `10/18` giữa hai nguồn |
| Conflicting | hai nguồn lệch một giờ cho cùng trận |
| Missing | trận chưa có giờ khai cuộc |
| Delayed | lịch vòng sau chỉ công bố muộn |
| Unauthorized | đối tác widget đòi lịch trước thời điểm cho phép của giấy phép |
| Human error | biên tập viên đổi giờ nhầm trận |
| System failure | dịch vụ múi giờ không phản hồi, hệ thống hiển thị UTC kèm nhãn múi giờ |

### 4.9 `historical_data` — dữ liệu lịch sử

Phạm vi: H2H, các mùa trước, nhập lô lịch sử. Chủ sở hữu: Vận hành dữ liệu. Kịch bản: 005. Ca hỏng: `SPT-F-020`, `SPT-F-052`.

| Loại ca | Tình huống cụ thể |
|---|---|
| Normal | lịch sử đối đầu Sông Hồng FC và Phố Núi United qua các mùa |
| Edge | giải đổi tên thương mại giữa hai mùa nhưng vẫn là một giải |
| Failure | hỏi hạng mùa 2025/26 mà nhận bảng hiện tại |
| Dirty data | kết quả hiệp phụ bị ghi vào tỉ số 90 phút trong tệp lịch sử |
| Conflicting | hai nguồn lịch sử ghi tỉ số khác nhau cho một trận cũ |
| Missing | mùa 2019/20 thiếu một vòng |
| Delayed | dữ liệu lịch sử của giải mới chỉ nhập sau nhiều tuần |
| Unauthorized | gói cơ bản đòi dữ liệu lịch sử quá phạm vi gói |
| Human error | nhập lô gán trận vào sai mùa |
| System failure | nhập lô lịch sử dừng giữa chừng, một số mùa nửa vời |

## 5. Các domain (19–26)

### 5.1 `data_correction` — sửa dữ liệu

Phạm vi: workflow `correction-request`, đối chiếu biên bản chính thức, sự kiện sau `FINAL`. Chủ sở hữu: Biên tập. Kịch bản: 024, 034, 037, 038, 048, 049, 057. Ca hỏng: `SPT-F-006`, `SPT-F-034`, `SPT-F-051`, `SPT-F-052`, `SPT-F-057`, `SPT-F-058`, `SPT-F-062`, `SPT-F-070`.

| Loại ca | Tình huống cụ thể |
|---|---|
| Normal | bản sửa CR-0008 đi `DRAFT → SUBMITTED → REVIEWING → APPROVED → PUBLISHED` với người duyệt khác người yêu cầu |
| Edge | rút lại bàn thắng ở giây thứ 301 sau khi nhập, vừa quá cửa sổ tự sửa 300 giây |
| Failure | bản sửa quá hạn mà vẫn nằm ở `REVIEWING`, không ai nhận việc |
| Dirty data | quan sát muộn sau `FINAL` đến từ FeedBeta, ngoài cửa sổ muộn |
| Conflicting | biên bản chính thức ghi bàn 23' là phản lưới của Tạ Văn Lộc, khác dữ liệu đã công bố |
| Missing | lệnh sửa thiếu lý do và bằng chứng |
| Delayed | biên bản chính thức đến sau trận một giờ |
| Unauthorized | nhân viên hỗ trợ cố sửa người ghi bàn trực tiếp |
| Human error | biên tập viên tự duyệt bản sửa của chính mình |
| System failure | gửi lại cùng idempotency key sau khi mạng đứt, không được tạo bản sửa thứ hai |

### 5.2 `localization` — đa ngôn ngữ

Phạm vi: tên đội và cầu thủ theo ngôn ngữ, dịch bản tin. Chủ sở hữu: Biên tập. Kịch bản: 012. Ca hỏng: `SPT-F-049`.

| Loại ca | Tình huống cụ thể |
|---|---|
| Normal | dịch bản tin sang tiếng Anh, giữ nguyên tỉ số 2-1 và tên cầu thủ |
| Edge | tên có dấu thanh `Lương Đình Hậu` phải giữ nguyên dạng hiển thị khi dịch |
| Failure | bản dịch đổi tỉ số 2-1 thành 2-0 |
| Dirty data | nguồn bản tin trộn tiếng Việt và tiếng Anh trong một câu |
| Conflicting | hai bản tên tiếng Anh của cùng một đội trong bảng `localized_names` |
| Missing | đội chưa có tên tiếng Anh |
| Delayed | tên tiếng Anh của giải mới chỉ có sau khi biên tập duyệt |
| Unauthorized | đối tác gói cơ bản đòi bản dịch tự động ngoài gói |
| Human error | biên tập viên nhập nhầm tên tiếng Anh của hai đội |
| System failure | Model Gateway hết giờ, hệ thống trả bản gốc kèm nhãn chưa dịch |

### 5.3 `search` — tìm kiếm

Phạm vi: tìm đội, cầu thủ, bài viết; không dấu; gợi ý khi gõ. Chủ sở hữu: Sản phẩm. Kịch bản: 011. Ca hỏng: `SPT-F-036`.

| Loại ca | Tình huống cụ thể |
|---|---|
| Normal | gõ `Sông Hồng` ra Sông Hồng FC và Sông Hồng FC B, xếp đội một trước |
| Edge | gõ `hau` ra hai cầu thủ tên Lương Đình Hậu |
| Failure | bài viết lưu tên dạng NFD không khớp truy vấn không dấu |
| Dirty data | truy vấn có ký tự Cyrillic giả dạng chữ Latin |
| Conflicting | hai bài cùng tiêu đề nói hai người ghi bàn khác nhau |
| Missing | không có kết quả cho tên đội chưa có trong từ điển |
| Delayed | bài mới chưa vào chỉ mục |
| Unauthorized | khách xem bài viết dành cho đối tác trả phí |
| Human error | biên tập viên gắn bài vào sai cầu thủ cùng tên |
| System failure | chỉ mục tìm kiếm không phản hồi, hệ thống trả lỗi có nhãn hạ tầng |

### 5.4 `notification` — thông báo

Phạm vi: đăng ký theo dõi, thông báo bàn thắng, kết quả, đính chính. Chủ sở hữu: Sản phẩm. Kịch bản: 043. Ca hỏng: `SPT-F-051`, `SPT-F-059`.

| Loại ca | Tình huống cụ thể |
|---|---|
| Normal | người dùng theo dõi Sông Hồng FC và nhận thông báo bàn 23' đã xác nhận |
| Edge | người dùng theo dõi cả hai đội của một trận, nhận đúng một thông báo mỗi bàn |
| Failure | thông báo gửi khi bàn còn ở trạng thái tạm thời, rồi phải đính chính |
| Dirty data | ba lần gửi cùng lệnh đăng ký do mạng chập chờn |
| Conflicting | hai provider báo bàn ở hai phút khác nhau, thông báo gửi hai lần |
| Missing | thiết bị không còn mã đẩy hợp lệ |
| Delayed | thông báo kết quả đến sau khi trận đã chốt `FINAL` mười phút |
| Unauthorized | khách chưa đăng nhập cố đăng ký thông báo |
| Human error | biên tập viên bấm gửi thông báo thủ công cho trận sai |
| System failure | dịch vụ đẩy sập giữa chừng, một nửa người nhận thiếu thông báo đính chính |

### 5.5 `content_generation` — sinh nội dung

Phạm vi: tóm tắt trận, bản tin sau trận, workflow `match-report`. Chủ sở hữu: Biên tập. Kịch bản: 009, 052. Ca hỏng: `SPT-F-045`, `SPT-F-047`, `SPT-F-053`, `SPT-F-054`, `SPT-F-065`.

| Loại ca | Tình huống cụ thể |
|---|---|
| Normal | tóm tắt trận 2-1 nêu đúng ba bàn thắng, mỗi khẳng định có mã bằng chứng |
| Edge | tóm tắt khi trận còn diễn ra phải nói rõ "đang diễn ra", không như chung cuộc |
| Failure | model nêu người ghi bàn không có trong sự kiện |
| Dirty data | bằng chứng có bàn đã bị rút lẫn với bàn hiệu lực |
| Conflicting | người ghi bàn đang `disputed`, bản tin phải mang nhãn bất đồng |
| Missing | trận chưa có thống kê, bản tin không được bịa số liệu |
| Delayed | bản tin soạn trước khi biên bản chính thức đến |
| Unauthorized | khách gói cơ bản đòi bản tin chuyên sâu trả phí |
| Human error | biên tập viên duyệt bản tin khi còn bản sửa mở |
| System failure | model hết giờ, hệ thống dùng bản tóm tắt tất định (tỉ số và danh sách bàn thắng) |

### 5.6 `ai_explanation` — giải thích bằng AI

Phạm vi: giải thích bàn bị huỷ, phong độ, chênh lệch; chỉ trên bằng chứng. Chủ sở hữu: Sản phẩm. Kịch bản: 010. Ca hỏng: `SPT-F-044`, `SPT-F-048`.

| Loại ca | Tình huống cụ thể |
|---|---|
| Normal | "Vì sao bàn 71' bị huỷ?" trả lời: việt vị theo sự kiện VAR, dẫn điều luật |
| Edge | hỏi nguyên nhân thất bại của một đội mà dữ liệu chỉ có kết quả |
| Failure | giải thích nêu chiến thuật hay chấn thương không có trong bằng chứng |
| Dirty data | bài viết cũ trong chỉ mục nêu tỉ số tạm thời |
| Conflicting | sự kiện VAR nói việt vị, bài viết nói lỗi khác |
| Missing | không có sự kiện VAR cho bàn bị huỷ |
| Delayed | lý do VAR đến sau bàn bị huỷ vài phút |
| Unauthorized | khách gói cơ bản đòi phân tích sâu ngoài gói |
| Human error | người hỏi nhầm trận, giải thích gắn vào trận khác |
| System failure | kho điều luật không truy xuất được, hệ thống nói chưa có thông tin điều luật |

### 5.7 `regulations` — điều lệ và luật thi đấu

Phạm vi: điều lệ giải, luật thi đấu, tài liệu có phiên bản (RAG). Chủ sở hữu: Biên tập. Kịch bản: 013. Ca hỏng: `SPT-F-043`.

| Loại ca | Tình huống cụ thể |
|---|---|
| Normal | điều lệ Cúp Lạc Hồng 2026/27: hoà sau 90 phút thì hai hiệp phụ rồi luân lưu |
| Edge | câu hỏi gồm hai ý (thể thức khi hoà và luật việt vị) |
| Failure | trích điều lệ mùa 2025/26 đã hết hiệu lực |
| Dirty data | bản PDF điều lệ quét lỗi thiếu một điều khoản |
| Conflicting | hai phiên bản điều lệ cùng hiệu lực nói khác nhau về tiêu chí phụ |
| Missing | giải chưa có điều lệ nạp |
| Delayed | điều lệ sửa đổi giữa mùa chưa được nạp |
| Unauthorized | khách đòi bản điều lệ nội bộ chưa công bố |
| Human error | nạp nhầm điều lệ của giải khác vào Cúp Lạc Hồng |
| System failure | dịch vụ đọc tài liệu sập lúc nạp, phiên bản mới không được công bố nguyên tử |

### 5.8 `compliance` — chính sách, giấy phép, cô lập tenant

Phạm vi: không cá cược, giấy phép dữ liệu, gói truy cập, tenant từ danh tính. Chủ sở hữu: Pháp chế và Kỹ thuật. Kịch bản: 058, 059, 060. Ca hỏng: `SPT-F-040`, `SPT-F-060`, `SPT-F-061`, `SPT-F-063`, `SPT-F-064`.

| Loại ca | Tình huống cụ thể |
|---|---|
| Normal | hỏi "kèo trận tối nay" bị từ chối bằng mẫu câu của tenant, gợi ý câu hỏi thay thế |
| Edge | câu hỏi lách luật: `soi k3o tài xỉu` và "dự đoán tỉ số để chơi" |
| Failure | câu hỏi kèo bị route sang `RAG` và model trả lời về cá cược |
| Dirty data | body request tự khai `tenant_id` của tenant khác |
| Conflicting | khách có gói cơ bản nhưng khoá API gắn quyền gói cao |
| Missing | request thiếu khoá API |
| Delayed | gói của khách hết hạn nhưng quyền còn hiệu lực trong cache |
| Unauthorized | khách gói cơ bản đòi payload gốc FeedAlpha hoặc đội hình chi tiết |
| Human error | quản trị cấp nhầm gói premium cho khách cơ bản |
| System failure | dịch vụ phân quyền không phản hồi, hệ thống từ chối (không cho phép mặc định) |

---

## 6. Phủ domain

26 domain; mỗi domain có đủ 10 loại ca (260 dòng). Trong `cases.yaml`, 25 domain có ít nhất một ca; domain `ranking` chưa có ca vì chưa có kịch bản riêng trong 60 kịch bản của [05-problem-catalog.md](05-problem-catalog.md) (mô tả ở đây để giữ đủ 24 domain tối thiểu của yêu cầu nền).
