# 07 — Danh mục ca hỏng (tenant `xyz-sports`)

> Mọi tên giải, đội, cầu thủ, provider là **hư cấu**. Mọi con số (giây, phút, số sự kiện) là **giả lập**/ngưỡng cấu hình mặc định; chưa có phép đo nào. Chỉ dữ liệu thông tin thể thao, không có tính năng cá cược.

Thể thao hỏng theo hướng khác tenant xây dựng: không phải "người nhập bẩn" mà là **dòng sự kiện realtime nhiều nguồn** — thứ tự, trùng lặp, sửa muộn, nhận dạng thực thể, đồng thuận provider, cache. Vì vậy ba nhóm `STREAM`, `CACHE`, `ENTITY` chiếm đa số.

Tài liệu có **71 ca hỏng** `SPT-F-001` … `SPT-F-071`, đủ 18 nhóm lỗi của CONVENTIONS §4.2. Mỗi ca ghi: nhóm, mức nghiêm trọng, kích hoạt, hệ quả, cách phát hiện, capability giảm thiểu, nhãn `fault` (CONVENTIONS §4.3), kịch bản gốc ở [05-problem-catalog.md](05-problem-catalog.md) và ca kiểm thử ở [08-test-cases.md](08-test-cases.md).

## 1. Cách đọc

| Mức | Nghĩa (CONVENTIONS §4.4) |
|---|---|
| `S1` | sai số liệu/hành động tới tay người dùng, rò dữ liệu giữa tenant, hành động ghi sai |
| `S2` | sai nhưng bị chặn/khó lan; mất chức năng chính |
| `S3` | trải nghiệm kém, chậm, tốn tiền vô ích |
| `S4` | thẩm mỹ, nhật ký thiếu |

- **Capability giảm thiểu** là package trong [kiến trúc §3](../../01-architecture.md). Tên `in nghiêng` là *primitive còn thiếu* đã nêu ở [05 §5](05-problem-catalog.md#5-primitive-còn-thiếu-do-thể-thao-chỉ-ra); chúng chỉ vào core khi ca hỏng chạy đỏ và có test (ADR-0002).
- **fault**: `code` = lỗi của chính EAIS; `data` = dữ liệu tenant thiếu/hỏng/xung đột không tự giải được; `policy` = bị chính sách chặn; `third_party:<provider>:<kind>` = nguồn ngoài; `infra:<component>` = hạ tầng. Ở ca "hệ thống làm sai", nhãn là nhãn của lỗi nếu nó được báo ra ngoài; nếu lỗi thầm lặng, nhãn là thứ **phải** gắn khi phát hiện.
- Bốn tầng dữ liệu: `OBSERVATION` (sự kiện thô của provider) → `MEASUREMENT`/`FACT` (sự kiện chuẩn) → `DERIVED FACT` (tỉ số, bảng xếp hạng). Nhiều ca hỏng dưới đây là **lẫn tầng**.

## 2. Bảng ca hỏng

### 2.1 STREAM — dòng sự kiện, thứ tự, trùng lặp, sửa muộn (11)

| ID | Nhóm | Mức | Kích hoạt | Hệ quả | Cách phát hiện | Capability giảm thiểu | fault | Kịch bản | Ca kiểm thử |
|---|---|---|---|---|---|---|---|---|---|
| `SPT-F-001` | STREAM | S1 | cùng một bàn thắng 23' đến từ ba provider với id và phút khác nhau, bị đếm thành ba bàn | tỉ số sai (ví dụ 3-0 thay vì 1-0) hiện tới người dùng | bất biến "tỉ số dẫn xuất = số bàn hiệu lực"; đếm số lần gộp; so tỉ số provider | `domain/sportsevents` khoá khử trùng + *`incident_correlation`* | `code` | 019 | `SPT-T-019` |
| `SPT-F-002` | STREAM | S1 | FeedAlpha kết nối lại, phát lại backlog; thiếu idempotency theo `(provider, provider_event_ref)` | bàn thắng, thẻ bị ghi hai lần | tỉ lệ vi phạm khoá duy nhất; tỉ lệ trùng theo provider | `domain/sportsevents` + `platform/store` (ràng buộc duy nhất theo tenant) | `code` | 020 | `SPT-T-020`, `SPT-T-064` |
| `SPT-F-003` | STREAM | S1 | sự kiện rút lại (VAR huỷ bàn) đến nhưng projection không áp | tỉ số vẫn 2-1 dù bàn đã bị huỷ | so tỉ số dẫn xuất với phát lại event store; cảnh báo lệch | projection idempotent + *`derived_fact_invalidation`* | `code` | 021 | `SPT-T-021` |
| `SPT-F-004` | STREAM | S1 | thẻ đỏ phút 78 đến trước bàn thắng phút 71; kết quả phụ thuộc thứ tự đến | diễn biến sai thứ tự, số cầu thủ trên sân sai | thử tính chất: hoán vị thứ tự đến cho cùng một trạng thái cuối | *`event_time_watermark`* trong `domain/sportsevents` | `code` | 022 | `SPT-T-022` |
| `SPT-F-005` | STREAM | S2 | bàn thắng 88' đến sau còi mãn cuộc; mốc chốt đóng quá sớm nên bị bỏ | kết quả chung cuộc thiếu bàn | bộ đếm sự kiện muộn; so với biên bản sau trận | guard `late_window_elapsed` + *`event_time_watermark`* | `code` | 023 | `SPT-T-023` |
| `SPT-F-006` | STREAM | S1 | sự kiện đến sau `FINAL` được áp thẳng, không qua workflow sửa dữ liệu | kết quả đã chốt đổi không dấu vết | chuỗi băm audit; chèn `match_events` sau `FINAL` thiếu mã bản sửa | workflow `correction-request` + `governance/audit` | `code` | 024 | `SPT-T-024` |
| `SPT-F-007` | STREAM | S1 | snapshot cũ của FeedGamma (trễ) ghi đè trạng thái mới | tỉ số lùi 2-1 → 1-1 | cảnh báo `state_version` lùi; so snapshot với sự kiện đã nhận | tỉ số tính từ sự kiện + guard `forward_only` | `third_party:FeedGamma:delay` | 025 | `SPT-T-025` |
| `SPT-F-008` | STREAM | S2 | FeedAlpha im lặng 90 giây mà hệ thống vẫn phát dữ liệu cũ như mới | người dùng thấy tỉ số cũ mà không biết | chỉ số `provider_lag_seconds`; quá ngưỡng thì luật kích hoạt | `execution/triggers` + workflow `provider-incident` + cờ `stale` | `third_party:FeedAlpha:disconnect` | 026 | `SPT-T-026` |
| `SPT-F-009` | STREAM | S2 | khoảng trống số thứ tự (seq) của FeedAlpha không được phát hiện | mất sự kiện thầm lặng | bộ dò khoảng trống theo trận | `data/ingestion` yêu cầu gửi bù (backfill) + cảnh báo | `third_party:FeedAlpha:gap` | 026 | `SPT-T-061` |
| `SPT-F-010` | STREAM | S2 | trạng thái trận nhảy lùi `FT → LIVE_2H` vì provider gửi trạng thái cũ | trận "kết thúc" rồi "đang đá" lại | bộ đếm phép chuyển bị chặn | guard `forward_only` của `match-lifecycle` | `code` | 055 | `SPT-T-055` |
| `SPT-F-011` | STREAM | S1 | hai worker xử lý cùng một trận đồng thời, mất cập nhật ở projection | thiếu một bàn trong tỉ số | xung đột `state_version`; thử đồng thời | khoá theo trận + kiểm phiên bản lạc quan (`platform/store`) | `code` | 022 | `SPT-T-041`, `SPT-T-064` |

### 2.2 CACHE — bản đọc nhanh phải đúng sau khi sửa (5)

MVP dùng PostgreSQL làm event store và read model; chưa có Redis/Kafka cho tới khi số đo đòi (ADR-0004). "Cache" ở đây gồm cả bảng `match_states` làm bản đọc, ETag/`Cache-Control` ở HTTP, bộ nhớ trong của một instance.

| ID | Nhóm | Mức | Kích hoạt | Hệ quả | Cách phát hiện | Capability giảm thiểu | fault | Kịch bản | Ca kiểm thử |
|---|---|---|---|---|---|---|---|---|---|
| `SPT-F-012` | CACHE | S1 | bàn thắng bị huỷ nhưng cache không vô hiệu | app còn hiển thị 2-1 sau khi đã huỷ | so `state_version` trả về với `match_states` | *`read_model_cache_contract`* (khoá theo `state_version`, vô hiệu khi projection đổi) | `code` | 021 | `SPT-T-021` |
| `SPT-F-013` | CACHE | S1 | khoá cache thiếu `tenant_id` hoặc `state_version` | trả bản cũ, hoặc dữ liệu của tenant khác | test khoá cache; quét khoá thiếu tiền tố tenant | `governance/tenancy` (namespace) + khoá có `state_version` | `code` | 021, 060 | `SPT-T-021` |
| `SPT-F-014` | CACHE | S2 | TTL của trận đang diễn ra bằng TTL của trận đã kết thúc | tỉ số live trễ cả phút | `cache_staleness_rate` theo trạng thái trận | TTL theo trạng thái trận khai báo trong pack | `code` | 001 | `SPT-T-001` |
| `SPT-F-015` | CACHE | S1 | nhiều instance, chỉ vô hiệu ở một instance | hai người thấy hai tỉ số khác nhau | so `state_version` giữa các instance | vô hiệu theo phiên bản lưu trong DB (không dựa bộ nhớ riêng) | `code` | 021 | `SPT-T-062` |
| `SPT-F-016` | CACHE | S1 | một trận bị sửa kết quả nhưng bảng xếp hạng cache không vô hiệu | bảng xếp hạng sai tới hết TTL | so bảng xếp hạng cache với bản tính lại | *`derived_fact_invalidation`* (phụ thuộc: trận → bảng) | `code` | 003, 037 | `SPT-T-003`, `SPT-T-037` |

### 2.3 ENTITY — nhận dạng đội, cầu thủ, giải (7)

| ID | Nhóm | Mức | Kích hoạt | Hệ quả | Cách phát hiện | Capability giảm thiểu | fault | Kịch bản | Ca kiểm thử |
|---|---|---|---|---|---|---|---|---|---|
| `SPT-F-017` | ENTITY | S1 | bí danh "Thành Cổ" ánh xạ cùng lúc Thành Cổ City và Thành Cổ Youth, hệ thống chọn một | trả lời về sai đội | bí danh trỏ >1 thực thể; tỉ lệ hỏi lại | `data/entityresolution` + `reliability/clarification` | `code` | 008 | `SPT-T-039`, `SPT-T-066` |
| `SPT-F-018` | ENTITY | S1 | hai cầu thủ trùng tên (Lương Đình Hậu) bị gộp, hoặc so khớp mờ dưới ngưỡng được tự chấp nhận ("HAU L.") | bàn thắng ghi cho sai người | ràng buộc "cầu thủ phải thuộc đội hình của trận"; tỉ lệ chấp nhận dưới ngưỡng | `data/entityresolution` (ràng buộc ngữ cảnh) + guard `confidence_below_auto_accept` | `code` | 008, 029 | `SPT-T-008`, `SPT-T-029` |
| `SPT-F-019` | ENTITY | S1 | Kiều Anh Bảo chuyển sang Phố Núi United từ 15/07/2026, ánh xạ vẫn gắn Cảng Biển Athletic | bàn thắng ghi cho sai đội | so đội theo hiệu lực với hai đội của trận | *`temporal_relations`* (cầu thủ ↔ đội có hiệu lực theo thời gian) | `data` | 028 | `SPT-T-028` |
| `SPT-F-020` | ENTITY | S2 | giải đổi tên thương mại (Sao Mai Championship) bị coi là giải mới | mất lịch sử, bảng xếp hạng tách đôi | giải mới không có quan hệ kế thừa | *`temporal_relations`* + workflow `mapping-review` | `data` | 032, 005 | `SPT-T-005`, `SPT-T-032` |
| `SPT-F-021` | ENTITY | S1 | Sông Hồng FC B (đội dự bị) bị ánh xạ vào Sông Hồng FC | trận của đội dự bị hiện ở giải đội một | cặp (đội, giải) không hợp lệ | ràng buộc loại giải trong `data/entityresolution` | `data` | 017 | `SPT-T-017` |
| `SPT-F-022` | ENTITY | S1 | FeedGamma gán đội nhà/khách ngược | tỉ số và kết quả gán ngược đội | so bên nhà/khách với lịch chính thức | kiểm bên theo fixture trong `domain/sportsevents` | `third_party:FeedGamma:bad_response` | 017, 033 | `SPT-T-017` |
| `SPT-F-023` | ENTITY | S1 | gộp hai bản ghi cầu thủ trùng mà không giữ lineage | thống kê bàn thắng sai, không đảo ngược được | so tổng bàn thắng trước/sau gộp | *`entity_merge_split`* (chuyển hướng + tính lại + đảo ngược) | `code` | 036 | `SPT-T-036` |

### 2.4 NORMALIZATION — phút, múi giờ, đồng hồ, mã trạng thái (5)

| ID | Nhóm | Mức | Kích hoạt | Hệ quả | Cách phát hiện | Capability giảm thiểu | fault | Kịch bản | Ca kiểm thử |
|---|---|---|---|---|---|---|---|---|---|
| `SPT-F-024` | NORMALIZATION | S1 | `"47"` không kèm kỳ bị hiểu là hiệp 2 thay vì `45+2` (tương tự `105+1`, `120+2`) | bàn thắng sai hiệp, sai thứ tự | phút vượt biên của kỳ | `data/normalization` chuẩn hoá theo kỳ + bảng biên kỳ | `code` | 018 | `SPT-T-018`, `SPT-T-069`, `SPT-T-079` |
| `SPT-F-025` | NORMALIZATION | S1 | giờ địa phương không offset bị coi là UTC | giờ khai cuộc lệch 7 giờ; "trận hôm nay" sai | lệch so với lịch chính thức | múi giờ khai báo theo provider trong pack | `code` | 033, 004 | `SPT-T-004`, `SPT-T-033` |
| `SPT-F-026` | NORMALIZATION | S2 | epoch giây lẫn mili giây của FeedGamma | thời gian năm 1970 hoặc xa trong tương lai | kiểm miền giá trị thời gian | kiểm miền + tự nhận đơn vị trong `data/normalization` | `third_party:FeedGamma:bad_response` | 033 | `SPT-T-033` |
| `SPT-F-027` | NORMALIZATION | S2 | đồng hồ bóng rổ đếm ngược bị hiểu là đếm xuôi | phút sai, sự kiện sai thứ tự | phút không đơn điệu trong một hiệp | thuộc tính chiều đồng hồ theo môn trong `domain/sportsevents` | `code` | 031 | `SPT-T-031` |
| `SPT-F-028` | NORMALIZATION | S2 | mã trạng thái số của FeedGamma đổi nghĩa giữa hai phiên bản | trạng thái trận sai | phân bố trạng thái đột biến sau đổi phiên bản | bảng mã theo phiên bản + test hợp đồng adapter | `third_party:FeedGamma:semantic_drift` | 027, 055 | `SPT-T-027` |

### 2.5 DATA — thẩm quyền, tầng dữ liệu, ngữ nghĩa (6)

| ID | Nhóm | Mức | Kích hoạt | Hệ quả | Cách phát hiện | Capability giảm thiểu | fault | Kịch bản | Ca kiểm thử |
|---|---|---|---|---|---|---|---|---|---|
| `SPT-F-029` | DATA | S1 | chọn người ghi bàn theo "đến sau cùng" thay vì thẩm quyền theo từng trường | người ghi bàn sai | kiểm luật `authority_not_recency` | `data/sourceoftruth` (ma trận thẩm quyền theo trường) | `code` | 053 | `SPT-T-053` |
| `SPT-F-030` | DATA | S1 | áp thẩm quyền giữa hai bàn thắng KHÁC nhau, coi là xung đột | một bàn bị loại oan | số bàn hiệu lực < số bàn đã xác nhận | tương quan sự cố **trước** thẩm quyền (*`incident_correlation`*) | `code` | 019, 053 | `SPT-T-019`, `SPT-T-053` |
| `SPT-F-031` | DATA | S1 | tỉ số lưu như giá trị gốc từ provider rồi bị provider ghi đè (lẫn DERIVED FACT với OBSERVATION) | tỉ số do một provider quyết, không truy được | bất biến "không có cột tỉ số ghi từ provider" | tách bốn tầng ở `data/sourceoftruth` | `code` | 025, 001 | `SPT-T-025` |
| `SPT-F-032` | DATA | S2 | thống kê ngoại lai (kiểm soát bóng 130%) không bị chặn | số thống kê vô lý hiện ra | kiểm miền giá trị | miền giá trị trong `metrics.yaml` + `reliability/validation` | `third_party:FeedBeta:bad_response` | 006 | `SPT-T-078` |
| `SPT-F-033` | DATA | S1 | bảng xếp hạng dùng sai tiêu chí phụ (đối đầu so với hiệu số) theo giải/mùa | thứ hạng sai | so với bảng chính thức của liên đoàn | quy tắc xếp hạng theo giải/mùa (pack cấu hình, mã cài đặt) | `code` | 003 | `SPT-T-003` |
| `SPT-F-034` | DATA | S1 | bàn phản lưới gán điểm cho đội ghi thay vì đội đối phương | tỉ số và bảng xếp hạng sai | bất biến: phản lưới cộng cho đội còn lại | kiểu sự kiện `OWN_GOAL` + luật trong `domain/sportsevents` | `data` | 057 | `SPT-T-057` |

### 2.6 PARSING — đọc payload provider (2)

| ID | Nhóm | Mức | Kích hoạt | Hệ quả | Cách phát hiện | Capability giảm thiểu | fault | Kịch bản | Ca kiểm thử |
|---|---|---|---|---|---|---|---|---|---|
| `SPT-F-035` | PARSING | S2 | FeedBeta đổi kiểu trường `minute`; adapter bỏ cả sự kiện thay vì cách ly | mất sự kiện thầm lặng | tỉ lệ payload bị bỏ so với nhận | cách ly payload sai schema + báo chủ dữ liệu (P4) | `third_party:FeedBeta:schema_drift` | 027 | `SPT-T-027` |
| `SPT-F-036` | PARSING | S3 | tên cầu thủ Unicode NFC lẫn NFD | tên không khớp bí danh, tìm kiếm "hau" trượt | bí danh không khớp dù nhìn giống nhau | chuẩn hoá Unicode trong `data/normalization` | `code` | 011 | `SPT-T-011` |

### 2.7 INTEGRATION — provider và giấy phép (4)

| ID | Nhóm | Mức | Kích hoạt | Hệ quả | Cách phát hiện | Capability giảm thiểu | fault | Kịch bản | Ca kiểm thử |
|---|---|---|---|---|---|---|---|---|---|
| `SPT-F-037` | INTEGRATION | S2 | FeedBeta bị giới hạn tần suất vì poll quá dày | mất dữ liệu nhiều lượt liên tiếp | tỷ lệ HTTP 429 | ngân sách gọi + lùi nhịp (backoff) trong `data/ingestion` | `third_party:FeedBeta:rate_limited` | 026 | `SPT-T-061` |
| `SPT-F-038` | INTEGRATION | S2 | khoá API của FeedAlpha hết hạn, luồng dừng thầm lặng | không có dữ liệu mà không ai biết | số 401 theo provider; ngày hết hạn khoá | luật kích hoạt + workflow `provider-incident` | `third_party:FeedAlpha:auth` | 050 | `SPT-T-050` |
| `SPT-F-039` | INTEGRATION | S2 | feed dừng giữa trận, không gửi `FT`; trận kẹt `LIVE` mãi | trận "đang đá" nhiều giờ sau giờ kết thúc | không có sự kiện quá N phút sau giờ dự kiến | timer + `provider-incident` + đóng bằng nguồn chính thức | `third_party:FeedAlpha:disconnect` | 045, 055 | `SPT-T-055` |
| `SPT-F-071` | INTEGRATION | S2 | cần hạ ưu tiên một provider ở một giải lúc đang chạy nhưng pack là tệp tĩnh: phải sửa tệp và khởi động lại, hoặc thay đổi bị áp hồi tố lên hoà giải cũ | thay đổi chỉ có hiệu lực sau lần khởi động kế tiếp, hoặc kết quả đã công bố bị đổi ngầm | so thời điểm hiệu lực của thay đổi với thời điểm áp dụng; audit thiếu bản ghi cấu hình | *`runtime_config_override`* (bản ghi đè có phiên bản, hiệu lực từ T, không hồi tố) + `core/tenant` | `code` | 042 | `SPT-T-042` |

### 2.8 ROUTER, RETRIEVAL, CONTEXT (7)

| ID | Nhóm | Mức | Kích hoạt | Hệ quả | Cách phát hiện | Capability giảm thiểu | fault | Kịch bản | Ca kiểm thử |
|---|---|---|---|---|---|---|---|---|---|
| `SPT-F-040` | ROUTER | S1 | câu hỏi kèo/tỉ lệ cược bị route sang `RAG` hoặc `DIRECT_LLM` | model trả lời về cá cược, trái chính sách nền tảng | ca bắt buộc `REJECT` trên bộ biến thể (kèo, tài xỉu, soi kèo…) | `intelligence/intent` (luật chặn) + route `REJECT` | `code` | 058 | `SPT-T-058`, `SPT-T-076`, `SPT-T-077` |
| `SPT-F-041` | ROUTER | S1 | câu hỏi tỉ số live bị route `RAG` | trả tỉ số cũ từ bài viết | ca router: mọi `score_query` phải là `SQL_QUERY` | `intelligence/router` + luật intent của pack | `code` | 001 | `SPT-T-001` |
| `SPT-F-042` | ROUTER | S2 | thiếu khe "đội" nhưng hệ thống đoán thay vì `CLARIFICATION` | trả lời cho đội đoán bừa | ca router: khe thiếu → `CLARIFICATION` | `reliability/clarification` + ngưỡng định tuyến | `code` | 007 | `SPT-T-007`, `SPT-T-065`, `SPT-T-080` |
| `SPT-F-043` | RETRIEVAL | S2 | truy xuất điều lệ của mùa cũ (phiên bản không còn hiệu lực) | tiêu chí phụ/thể thức sai | kiểm phiên bản + hiệu lực của tài liệu trích | `intelligence/rag` (phiên bản, hiệu lực) + `reliability/validation` (trích dẫn) | `data` | 013 | `SPT-T-013` |
| `SPT-F-044` | RETRIEVAL | S2 | bài viết cũ trong chỉ mục (tỉ số tạm thời) được dùng làm bằng chứng cho dữ kiện live | tỉ số lỗi thời | kiểm loại bằng chứng: dữ kiện live chỉ từ sự kiện | luật `live_facts_never_from_documents` trong Context Engine | `code` | 001, 010 | `SPT-T-001`, `SPT-T-010` |
| `SPT-F-045` | CONTEXT | S1 | context chứa cả sự kiện đã bị rút lẫn sự kiện hiệu lực | tóm tắt nhắc bàn bị huỷ như đã ghi | kiểm bằng chứng: không có sự kiện `retracted` | `intelligence/context` (loại sự kiện bị rút/bị thay thế) | `code` | 009 | `SPT-T-009` |
| `SPT-F-046` | CONTEXT | S1 | context trộn hai trận của cùng một đội (cùng ngày, hai giải) | tóm tắt lẫn hai trận | kiểm `match_id` của mọi bằng chứng | `intelligence/context` (lọc theo thực thể đã nhận dạng) | `code` | 009, 008 | `SPT-T-068` |

### 2.9 MODEL, TOOL, SKILL (8)

| ID | Nhóm | Mức | Kích hoạt | Hệ quả | Cách phát hiện | Capability giảm thiểu | fault | Kịch bản | Ca kiểm thử |
|---|---|---|---|---|---|---|---|---|---|
| `SPT-F-047` | MODEL | S1 | model bịa người ghi bàn hoặc phút trong bản tóm tắt | sự kiện không có thật tới người dùng | validator: mọi tên/số/phút phải có trong bằng chứng | structured output có `claims` + `reliability/validation` + mẫu câu lùi | `third_party:model:unverified_output` | 009 | `SPT-T-009` |
| `SPT-F-048` | MODEL | S1 | model nêu nguyên nhân không có trong bằng chứng (đổi chiến thuật, chấn thương) | giải thích bịa | validator: khẳng định nhân quả phải gắn bằng chứng | giới hạn khẳng định ở số liệu có trong gói bằng chứng | `third_party:model:unverified_output` | 010 | `SPT-T-010` |
| `SPT-F-049` | MODEL | S2 | model đổi số hoặc tên khi dịch bản tin (2-1 thành 2-0) | bản dịch sai số liệu | so tập số/tên đầu vào và đầu ra | `reliability/validation` (bảo toàn số + thực thể) | `third_party:model:unverified_output` | 012 | `SPT-T-012` |
| `SPT-F-050` | TOOL | S2 | `get_match_state` hết giờ, trả lỗi mơ hồ | người dùng không biết dữ liệu có cũ không | đo `tool_success`; nhãn lỗi | lùi có nhãn: trả bản gần nhất kèm cờ `stale` hoặc mẫu "chưa đủ dữ liệu" | `infra:postgres` | 001 | `SPT-T-063` |
| `SPT-F-051` | TOOL | S1 | lệnh ghi chạy hai lần khi bên gọi gửi lại | hai bản sửa/hai thông báo | khoá idempotency trùng nhưng thực thi hai lần | idempotency key ở C0 + tool idempotent | `code` | 034, 043 | `SPT-T-034`, `SPT-T-043` |
| `SPT-F-052` | TOOL | S2 | `get_standings` bỏ qua tham số `as_of` | trả bảng hiện tại cho câu hỏi lịch sử | ca truy vấn quá khứ so với bản chụp | *`as_of_query`* trong hợp đồng `StructuredDataSource` | `code` | 005, 057 | `SPT-T-005`, `SPT-T-057` |
| `SPT-F-053` | SKILL | S2 | skill `match_summary` chạy khi trận chưa kết thúc mà không nói "đang diễn ra" | bản tóm tắt trông như chung cuộc | kiểm trạng thái trận trong bằng chứng | điều kiện đầu vào của skill (trạng thái trận phải có trong gói) | `code` | 009 | `SPT-T-009` |
| `SPT-F-054` | SKILL | S2 | skill `post_match_report` công bố khi còn bản sửa chưa đóng | bản tin dựa trên kết quả sắp đổi | guard `no_open_reconciliation` ở bước công bố | guard của workflow `match-report` | `code` | 052 | `SPT-T-052` |

### 2.10 WORKFLOW (5)

| ID | Nhóm | Mức | Kích hoạt | Hệ quả | Cách phát hiện | Capability giảm thiểu | fault | Kịch bản | Ca kiểm thử |
|---|---|---|---|---|---|---|---|---|---|
| `SPT-F-055` | WORKFLOW | S1 | model đề xuất "đánh dấu trận `FINAL`" và hệ thống thực thi trực tiếp | trạng thái đổi mà không qua guard (vi phạm P11) | ca âm: phép chuyển không có guard bị chặn | `execution/workflows` (chỉ engine đổi trạng thái) | `code` | 051, 052 | `SPT-T-051`, `SPT-T-052` |
| `SPT-F-056` | WORKFLOW | S1 | `FT → FINAL` khi còn VAR chờ hoặc xung đột mở (thiếu guard) | kết quả chốt sớm rồi phải sửa | ca âm: `no_pending_var`, `no_open_reconciliation` | guard của `match-lifecycle` | `code` | 045 | `SPT-T-045` |
| `SPT-F-057` | WORKFLOW | S2 | bản sửa quá hạn không chuyển `EXPIRED` | kẹt ở `REVIEWING`, việc không ai nhận | quét human task quá hạn | timer + *`human_task_escalation`* | `code` | 049 | `SPT-T-049` |
| `SPT-F-058` | WORKFLOW | S1 | người yêu cầu tự duyệt bản sửa của chính mình | sửa dữ liệu không có người thứ hai | ca âm: guard `approver_differs_from_requester` | guard của `correction-request` | `policy` | 048 | `SPT-T-072` |
| `SPT-F-059` | WORKFLOW | S1 | thông báo đẩy gửi trước khi sự kiện được xác nhận và khử trùng | thông báo sai/đúp rồi phải đính chính | so thông báo với sự kiện hiệu lực | trigger đọc thay đổi của projection + *`supersedable_notification`* | `code` | 021, 043 | `SPT-T-021`, `SPT-T-043` |

### 2.11 AUTH, TENANCY (5)

| ID | Nhóm | Mức | Kích hoạt | Hệ quả | Cách phát hiện | Capability giảm thiểu | fault | Kịch bản | Ca kiểm thử |
|---|---|---|---|---|---|---|---|---|---|
| `SPT-F-060` | AUTH | S1 | `partner_basic` đọc được trường gói premium (đội hình chi tiết) | rò quyền theo gói | ca âm theo từng gói | `governance/permissions` (ABAC theo gói) | `policy` | 059 | `SPT-T-073` |
| `SPT-F-061` | AUTH | S1 | API B2B trả payload thô của provider | vi phạm giấy phép dữ liệu | kiểm đầu ra: không có trường thô | `governance/permissions` + chỉ xuất dữ liệu chuẩn | `policy` | 059 | `SPT-T-059` |
| `SPT-F-062` | AUTH | S2 | nhân viên hỗ trợ sửa tỉ số trực tiếp, bỏ qua workflow | sửa dữ liệu live không duyệt | ca âm theo vai trò | `governance/permissions` + lệnh chỉ đi qua `correction-request` | `policy` | 034, 048 | `SPT-T-071` |
| `SPT-F-063` | TENANCY | S1 | chỉ mục/cache dùng chung không có namespace tenant | rò dữ liệu giữa tenant | quét khoá thiếu tenant; ca cô lập | `governance/tenancy` (RLS + namespace) | `code` | 060 | `SPT-T-060` |
| `SPT-F-064` | TENANCY | S1 | tenant lấy từ body request thay vì danh tính | khách tự khai tenant để đọc dữ liệu tenant khác | ca âm: body `tenant_id` bị bỏ qua | `governance/auth` (tenant từ khoá API) | `policy` | 060 | `SPT-T-060` |

### 2.12 VALIDATION, OBSERVABILITY (6)

| ID | Nhóm | Mức | Kích hoạt | Hệ quả | Cách phát hiện | Capability giảm thiểu | fault | Kịch bản | Ca kiểm thử |
|---|---|---|---|---|---|---|---|---|---|
| `SPT-F-065` | VALIDATION | S1 | validator không bắt tỉ số trong bản tóm tắt không có trong bằng chứng | tỉ số lạ tới người dùng | ca âm: tóm tắt cố ý sai tỉ số | `reliability/validation` (số/tên phải có trong bằng chứng) | `code` | 009 | `SPT-T-009` |
| `SPT-F-066` | VALIDATION | S2 | validator chặn nhầm vì định dạng phút `45+2` (so với `45`, `47`) | câu đúng bị chặn (false block) | `false_block_rate` | chuẩn hoá phút trước khi so | `code` | 002, 009 | `SPT-T-002`, `SPT-T-009` |
| `SPT-F-067` | VALIDATION | S1 | không có bất biến "tỉ số dẫn xuất = số bàn hiệu lực" đối chiếu provider | sai lệch kéo dài mà không ai biết | kiểm bất biến định kỳ theo trận | `reliability/validation` + cảnh báo chênh | `code` | 025, 019 | `SPT-T-019`, `SPT-T-025` |
| `SPT-F-068` | OBSERVABILITY | S3 | không ghi `provider_refs` cho sự kiện chuẩn | không truy được vì sao hiển thị bàn đó | truy ngược một câu trả lời về nguồn | `observability/traces` + trường `provider_refs` | `code` | 053, 002 | `SPT-T-002`, `SPT-T-053` |
| `SPT-F-069` | OBSERVABILITY | S2 | không đo độ trễ theo provider | vi phạm mục tiêu độ trễ mà không ai biết | thiếu chỉ số `provider_lag_seconds` | `observability/traces` + chỉ số theo provider | `code` | 015, 026 | `SPT-T-015`, `SPT-T-026` |
| `SPT-F-070` | OBSERVABILITY | S3 | trace không lưu `state_version` của câu trả lời | không tái hiện được điều đã công bố khi có khiếu nại | tái hiện một câu trả lời cũ | `observability/traces` + *`as_of_query`* | `code` | 057 | `SPT-T-057` |

---

## 3. Đối chiếu mười hai ca hỏng realtime của yêu cầu nền

| Ca hỏng realtime | Ca hỏng cụ thể | Kịch bản chính |
|---|---|---|
| provider delayed (trễ) | `SPT-F-007`, `SPT-F-008` | 025, 026 |
| provider unavailable (không khả dụng) | `SPT-F-008`, `SPT-F-009`, `SPT-F-037`, `SPT-F-038`, `SPT-F-039` | 026, 045, 050 |
| provider disagreement (bất đồng) | `SPT-F-029`, `SPT-F-030`, `SPT-F-022` | 053, 054, 055, 056 |
| duplicate goal (bàn thắng trùng) | `SPT-F-001`, `SPT-F-002` | 019, 020 |
| goal later revoked/corrected (bàn bị huỷ/sửa) | `SPT-F-003`, `SPT-F-012`, `SPT-F-016`, `SPT-F-059` | 021, 037 |
| wrong player mapping (ánh xạ sai cầu thủ) | `SPT-F-018`, `SPT-F-019` | 029, 028 |
| match status inconsistent (trạng thái không nhất quán) | `SPT-F-010`, `SPT-F-028`, `SPT-F-056` | 055, 045 |
| event arrives out of order (đảo thứ tự) | `SPT-F-004`, `SPT-F-005`, `SPT-F-006`, `SPT-F-011` | 022, 023, 024 |
| cache stale (cache cũ) | `SPT-F-012`, `SPT-F-013`, `SPT-F-014`, `SPT-F-015`, `SPT-F-016` | 021, 001 |
| timezone error (sai múi giờ) | `SPT-F-025`, `SPT-F-026` | 033, 004 |
| competition renamed (giải đổi tên) | `SPT-F-020` | 032, 005 |
| player transferred (cầu thủ chuyển CLB) | `SPT-F-019` | 028 |

## 4. Phân bố

| Nhóm | Số ca hỏng |
|---|---|
| STREAM | 11 |
| CACHE | 5 |
| ENTITY | 7 |
| NORMALIZATION | 5 |
| DATA | 6 |
| PARSING | 2 |
| INTEGRATION | 4 |
| ROUTER | 3 |
| RETRIEVAL | 2 |
| CONTEXT | 2 |
| MODEL | 3 |
| TOOL | 3 |
| SKILL | 2 |
| WORKFLOW | 5 |
| AUTH | 3 |
| TENANCY | 2 |
| VALIDATION | 3 |
| OBSERVABILITY | 3 |
| **Tổng** | **71** |

| Mức | Số ca hỏng |
|---|---|
| S1 | 41 |
| S2 | 27 |
| S3 | 3 |
| S4 | 0 |

Mỗi ca hỏng có ít nhất một ca kiểm thử trong cột cuối (kiểm bằng script ở [08-test-cases.md](08-test-cases.md#6-lệnh-kiểm)).
