# 05 — Danh mục vấn đề nghiệp vụ (tenant `xyz-sports`)

> Tenant **XYZ Sports Data JSC** (hư cấu), nền tảng dữ liệu thể thao nhận dữ liệu từ nhiều provider. Mọi giải, đội, cầu thủ, provider trong tài liệu này đều **hư cấu**. Mọi con số (số sự kiện, phút, số người theo dõi…) là **giả lập** để minh hoạ; chưa có phép đo nào. Chỉ dữ liệu thông tin thể thao: **không có tính năng cá cược**, câu hỏi về kèo/tỉ lệ cược đi route `REJECT`.

Tài liệu này là danh mục **đúng 60 kịch bản** `SPT-S-001` … `SPT-S-060`. Mỗi kịch bản là một tình huống nghiệp vụ buộc kiến trúc EAIS phải chứng minh một năng lực; kịch bản nào kiến trúc hiện tại **không biểu diễn được** thì ghi rõ thiếu primitive nào (xem [§5](#5-primitive-còn-thiếu-do-thể-thao-chỉ-ra)).

Liên kết: kịch bản gốc của mọi ca kiểm thử (`scenario:` trong [cases.yaml](../../../tenants/sports/evals/cases.yaml), tóm tắt ở [08-test-cases.md](08-test-cases.md)); ca hỏng tương ứng ở [07-failure-catalog.md](07-failure-catalog.md).

---

## 1. Cách đọc

### 1.1 Ba lối vào và sáu loại `entry_type`

Kiến trúc có ba lối vào (QUERY → pipeline hỏi S0–S10, COMMAND → pipeline lệnh C0–C7, EVENT → pipeline sự kiện E0–E6) cộng với workflow engine sở hữu trạng thái. Thể thao hỏng nhiều nhất ở **EVENT** (provider đẩy dữ liệu liên tục), nên bộ kịch bản này cố ý **không** thiên về hỏi–đáp:

| `entry_type` | Nghĩa | Số kịch bản | Mã |
|---|---|---|---|
| `QUERY` | người hỏi, hệ thống trả lời có nguồn | 15 | 001–015 |
| `EVENT` | provider/hệ thống đẩy sự kiện vào; kết quả là trạng thái đổi đúng | 18 | 016–033 |
| `COMMAND` | biên tập viên/vận hành/khán giả ra lệnh làm đổi trạng thái | 10 | 034–043 |
| `WORKFLOW` | vòng đời có trạng thái, phép chuyển, guard, human task | 9 | 044–052 |
| `RECONCILIATION` | nhiều nguồn nói khác nhau về cùng một sự việc | 5 | 053–057 |
| `SECURITY` | chính sách, giấy phép, cô lập tenant | 3 | 058–060 |

### 1.2 Các cột

| Cột | Ý nghĩa |
|---|---|
| entry_type | lối vào của kịch bản (§1.1): `QUERY`, `EVENT`, `COMMAND`, `WORKFLOW`, `RECONCILIATION`, `SECURITY`; khoá `entry_type` của ca kiểm thử lấy đúng giá trị này |
| Domain | nghiệp vụ trong [02-domain-map.md](02-domain-map.md) (slug dùng làm khoá `domain:` của ca kiểm thử) |
| Đầu vào / tình huống | câu hỏi, lệnh hay sự kiện; phần *in nghiêng* là tình huống dữ liệu làm nó khó |
| RAG một mình? | `YES` / `NO` / `PARTIAL` + vì sao. RAG (retrieval-augmented generation, sinh có truy xuất tài liệu) **không** phải trung tâm: số liệu live nằm trong event store, không nằm trong tài liệu |
| Capability | package trong cây thư mục của [kiến trúc §3](../../01-architecture.md) cần cho kịch bản |
| Route / pipeline | route (12 route của CONVENTIONS §4.1) cho `QUERY`/`SECURITY`; `C0–C7`, `E0–E6`, `engine` cho các lối vào còn lại |
| Tính chất | `Tất định` · `Xác suất` · `Lai` (xác suất bọc trong kiểm tất định) |
| Biểu diễn được? | kiến trúc **hiện tại** biểu diễn được kịch bản bằng primitive đã có (kể cả pack, domain module, workflow engine)? `YES`, hoặc `NO — thiếu <primitive>` |
| MVP | đúng 15 kịch bản đánh dấu `MVP` (MVP chỉ bóng đá) |

### 1.3 Ba nguyên tắc chi phối cách chấm "RAG một mình" và "Biểu diễn được"

1. **Bốn tầng dữ liệu.** Sự kiện thô của provider là `OBSERVATION`; sự kiện chuẩn (đã chuẩn hoá, nhận dạng thực thể, khử trùng) là `MEASUREMENT`/`FACT`; **tỉ số và bảng xếp hạng là `DERIVED FACT`**, tính từ sự kiện hiệu lực bằng công thức có ghi lại. Không ai "lưu tỉ số" như một giá trị gốc để provider ghi đè.
2. **Thẩm quyền/đồng thuận chỉ áp giữa các quan sát về CÙNG một sự kiện.** Hai bàn thắng khác nhau của cùng một đội không "xung đột" nhau. Việc xác định "hai quan sát này nói về cùng một bàn thắng" là bước riêng (tương quan), đi **trước** bước chọn bên.
3. **Model chỉ đề xuất (P11).** Model không chuyển trạng thái trận, không duyệt bản sửa, không tự gộp cầu thủ. Engine + guard (đặt theo **tên**) mới được đổi trạng thái.

---

## 2. QUERY — hỏi và đáp (15)

| ID | entry_type | Domain | Đầu vào / tình huống | RAG một mình? | Capability | Route | Tính chất | Biểu diễn được? | MVP |
|---|---|---|---|---|---|---|---|---|---|
| `SPT-S-001` | `QUERY` | `live_match` | "Tỉ số Sông Hồng – Phố Núi đang là bao nhiêu, trận kết thúc chưa?" *(trận đang phút 75; ba provider cập nhật lệch vài giây)* | **NO** — tỉ số live là DERIVED FACT từ sự kiện hiệu lực, không nằm trong tài liệu; RAG trả bản tin cũ | `data/entityresolution` · `domain/sportsevents` · `execution/tools` · `reliability/validation` | `SQL_QUERY` | Tất định | YES | MVP |
| `SPT-S-002` | `QUERY` | `goals` | "Ai ghi bàn trong trận Sông Hồng – Phố Núi, theo thứ tự diễn biến?" *(sự kiện đến đảo thứ tự; có một bàn bị huỷ sau VAR)* | **NO** — danh sách bàn hiệu lực là phép chiếu từ event store (đã loại bàn bị rút); văn bản không phân biệt bàn còn hiệu lực với bàn bị huỷ | `domain/sportsevents` · `execution/tools` · `reliability/validation` | `SQL_QUERY` | Tất định | YES | MVP |
| `SPT-S-003` | `QUERY` | `standings` | "Bảng xếp hạng Giải Vô địch Lạc Hồng hiện tại?" *(nhiều trận đang diễn ra cùng lúc)* | **NO** — bảng xếp hạng là DERIVED FACT (điểm + tiêu chí phụ); RAG không cộng, không sắp | `domain/sportsevents` · `data/sourceoftruth` · `execution/tools` | `SQL_QUERY` | Tất định | YES | MVP |
| `SPT-S-004` | `QUERY` | `schedule` | "Tối nay Sông Hồng FC đá mấy giờ?" *(FeedBeta gửi giờ địa phương không kèm offset; người hỏi ở múi giờ khác giờ Việt Nam)* | **NO** — giờ khai cuộc là dữ liệu có kiểu + múi giờ; RAG có thể lấy nhầm giờ từ bài viết cũ | `data/normalization` · `domain/sportsevents` · `execution/tools` | `SQL_QUERY` | Tất định | YES | MVP |
| `SPT-S-005` | `QUERY` | `historical_data` | "Lịch sử đối đầu Sông Hồng FC – Phố Núi United; mùa trước ở giải Sao Mai họ xếp thứ mấy?" *(giải đổi tên thương mại giữa hai mùa)* | **NO** — H2H và hạng mùa trước là truy vấn có cấu trúc; tên giải đổi nên cần nhận dạng liên tục theo thời gian | `data/entityresolution` · `execution/tools` | `SQL_QUERY` | Tất định | NO — thiếu `temporal_relations` | — |
| `SPT-S-006` | `QUERY` | `statistics` | "Phong độ 5 trận gần nhất của Phố Núi United: thắng/hoà/thua, bàn thắng, bàn thua?" | **PARTIAL** — RAG tìm được bài nhận định nhưng không cộng/trừ; số phải tính bằng phép tính có ghi công thức | `execution/tools` · `reliability/validation` | `CALCULATION` | Tất định | YES | — |
| `SPT-S-007` | `QUERY` | `team_identity` | "Tỉ số trận của Rồng Vàng FC?" *(không có đội nào mang tên/bí danh đó trong từ điển)* | **NO** — RAG sẽ đoán đội gần giống; đúng là dừng và hỏi lại | `data/entityresolution` · `reliability/clarification` · `intelligence/router` | `CLARIFICATION` | Tất định | YES | MVP |
| `SPT-S-008` | `QUERY` | `player_identity` | "Lương Đình Hậu ghi mấy bàn mùa này?" *(hai cầu thủ trùng tên: tiền đạo Sông Hồng FC và tiền vệ Thành Cổ Youth; tương tự "Thành Cổ hôm nay đá không?" và "bảng xếp hạng mùa 2026")* | **NO** — trả lời theo người "nổi tiếng hơn" là đoán; cần ứng viên và câu hỏi lại | `data/entityresolution` · `reliability/clarification` | `CLARIFICATION` | Tất định | YES | — |
| `SPT-S-009` | `QUERY` | `content_generation` | "Tóm tắt trận Sông Hồng 2–1 Phố Núi." *(có một bàn bị huỷ sau VAR; chỉ được dùng sự kiện, không bịa)* | **PARTIAL** — RAG diễn đạt được nhưng không biết bàn nào còn hiệu lực; số và tên phải khớp bằng chứng | `execution/skills` · `intelligence/context` · `intelligence/models` · `reliability/validation` | `WORKFLOW` | Lai | YES | MVP |
| `SPT-S-010` | `QUERY` | `ai_explanation` | "Vì sao bàn thắng phút 71 của Sông Hồng FC bị huỷ?" | **PARTIAL** — điều luật nằm trong tài liệu (RAG làm được) nhưng sự kiện VAR cụ thể nằm ở event store | `intelligence/rag` · `domain/sportsevents` · `intelligence/context` · `intelligence/models` · `reliability/validation` | `HYBRID_RETRIEVAL` | Lai | YES | — |
| `SPT-S-011` | `QUERY` | `search` | "Tìm bài viết về Lương Đình Hậu" và truy vấn không dấu "hau" *(tên có nhiều biến thể, kể cả viết sai)* | **PARTIAL** — tìm chuỗi trong bài viết là việc của retrieval, nhưng chuẩn hoá không dấu và bí danh cần entity | `intelligence/rag` · `data/entityresolution` · `data/normalization` | `FULL_TEXT_SEARCH` | Tất định | YES | — |
| `SPT-S-012` | `QUERY` | `localization` | "Dịch bản tin sang tiếng Anh, giữ nguyên tỉ số và tên: 'Sông Hồng FC thắng 2-1 Phố Núi United…'" | **PARTIAL** — model dịch được nhưng thiếu kiểm số/tên bảo toàn | `intelligence/models` · `reliability/validation` | `DIRECT_LLM` | Xác suất (kiểm bằng code) | YES | — |
| `SPT-S-013` | `QUERY` | `regulations` | "Ở Cúp Lạc Hồng, nếu hoà sau 90 phút thì xử lý thế nào? Luật việt vị nói gì?" | **YES** — tri thức nằm trong điều lệ/luật có phiên bản; cần trích dẫn và đúng phiên bản mùa giải | `intelligence/rag` · `intelligence/context` · `reliability/validation` | `RAG` | Xác suất (có trích dẫn) | YES | — |
| `SPT-S-014` | `QUERY` | `live_match` | "Phạm Bích Ngọc đang ở set mấy, game mấy?" / "Đông Hải Anchors đang hiệp mấy, mấy out?" *(môn không có đồng hồ: set, inning)* | **NO** — trạng thái thi đấu theo môn là dữ liệu có cấu trúc; RAG không có mô hình hiệp/set/inning | `domain/sportsevents` · `execution/tools` | `SQL_QUERY` | Tất định | YES | — |
| `SPT-S-015` | `QUERY` | `provider_integration` | (nhân viên vận hành) "FeedBeta đang trễ bao nhiêu giây trong 10 phút qua?" | **NO** — số đo vận hành nằm trong metric, không nằm trong tài liệu | `execution/tools` · `observability/traces` | `TOOL` | Tất định | YES | — |

---

## 3. EVENT — provider đẩy sự kiện vào (18)

Ca `EVENT` không có "câu trả lời": kết quả là **trạng thái sau xử lý** (event store, `match_states`, workflow, human task). Vì vậy cột "RAG một mình?" gần như luôn `NO`: RAG không nạp, không ghi, không khử trùng.

| ID | entry_type | Domain | Đầu vào / tình huống | RAG một mình? | Capability | Pipeline | Tính chất | Biểu diễn được? | MVP |
|---|---|---|---|---|---|---|---|---|---|
| `SPT-S-016` | `EVENT` | `substitution` | Phút 58: FeedAlpha gửi cặp *ra/vào* (Tín ra, Kiên vào); FeedGamma chỉ gửi "vào" | **NO** — không có câu hỏi; ghép cặp và kiểm "người ra đang trên sân" là luật nghiệp vụ | `data/ingestion` · `data/entityresolution` · `domain/sportsevents` | `E0–E6` | Tất định | YES | — |
| `SPT-S-017` | `EVENT` | `team_identity` | Ba provider gửi ba tên khác nhau cho cùng một đội: `SONG HONG FC` / `Sông Hồng` / `SHG` | **NO** — hợp nhất bí danh theo bảng ánh xạ provider là việc của entity resolution | `data/entityresolution` · `domain/sportsevents` | `E0–E6` | Tất định | YES | MVP |
| `SPT-S-018` | `EVENT` | `event_timeline` | Bàn thắng của Phố Núi United: FeedAlpha `45+2`, FeedBeta `47` (kèm period), FeedGamma `45` | **NO** — chuẩn hoá phút theo kỳ (kỳ, phút, bù giờ) là code; "47" ở hiệp 1 khác "47" ở hiệp 2 | `data/normalization` · `domain/sportsevents` | `E0–E6` | Tất định | YES | MVP |
| `SPT-S-019` | `EVENT` | `goals` | Cùng một bàn thắng 23' đến từ ba provider với ba id khác nhau, phút khác nhau vài giây | **NO** — khử trùng chéo provider cần khoá tương quan có dung sai, không phải so chuỗi | `domain/sportsevents` · `data/sourceoftruth` | `E0–E6` | Tất định | NO — thiếu `incident_correlation` | MVP |
| `SPT-S-020` | `EVENT` | `provider_integration` | FeedAlpha kết nối lại và **phát lại** backlog 40 sự kiện *(giả lập)*, gồm cả hai bàn thắng đã xử lý | **NO** — idempotency theo khoá khử trùng của luồng sự kiện | `data/ingestion` · `domain/sportsevents` · `platform/store` | `E0–E6` | Tất định | YES | — |
| `SPT-S-021` | `EVENT` | `var_correction` | Bàn thắng 71' bị huỷ sau VAR (việt vị): sự kiện **rút lại** trỏ về bàn cũ; tỉ số về 1-1; cache phải vô hiệu; người theo dõi nhận thông báo đính chính | **NO** — rút lại là sự kiện mới, không sửa đè; RAG không biết bàn nào đã bị huỷ | `domain/sportsevents` · `data/sourceoftruth` · `execution/triggers` | `E0–E6` | Tất định | NO — thiếu `derived_fact_invalidation`, `read_model_cache_contract`, `supersedable_notification` | MVP |
| `SPT-S-022` | `EVENT` | `event_timeline` | Thẻ đỏ phút 78 **đến trước** bàn thắng phút 71 (hai nguồn chậm khác nhau) | **NO** — thứ tự theo giờ trận, không theo thứ tự đến; projection không được phụ thuộc thứ tự đến | `domain/sportsevents` · `platform/store` | `E0–E6` | Tất định | NO — thiếu `event_time_watermark` | MVP |
| `SPT-S-023` | `EVENT` | `event_timeline` | Bàn thắng 88' đến **sau còi mãn cuộc** 40 giây *(giả lập)*, vẫn trong cửa sổ muộn | **NO** — sự kiện muộn phải chèn đúng chỗ và hoãn `FINAL`, không bị bỏ | `domain/sportsevents` · `execution/workflows` | `E0–E6` | Tất định | NO — thiếu `event_time_watermark` | — |
| `SPT-S-024` | `EVENT` | `data_correction` | Sự kiện đến **sau `FINAL`** (ngoài cửa sổ muộn): không được áp thầm lặng | **NO** — sau `FINAL` chỉ đổi qua workflow sửa dữ liệu; cần ứng viên sửa + việc cho người | `domain/sportsevents` · `execution/workflows` · `governance/audit` | `E0–E6` | Tất định | NO — thiếu `event_time_watermark` | — |
| `SPT-S-025` | `EVENT` | `live_match` | FeedGamma gửi **snapshot cũ** (1-1) sau khi hệ thống đã ghi bàn thứ hai | **NO** — snapshot cũ không được ghi đè trạng thái mới; tỉ số tính từ sự kiện, không từ snapshot | `domain/sportsevents` · `data/sourceoftruth` | `E0–E6` | Tất định | YES | — |
| `SPT-S-026` | `EVENT` | `provider_integration` | FeedAlpha **mất nhịp** 90 giây giữa hiệp 2 *(giả lập)*; hai provider còn lại vẫn chạy | **NO** — phát hiện mất nhịp là metric + luật kích hoạt; hệ thống phải gắn cờ `stale` và mở sự cố | `execution/triggers` · `execution/workflows` · `observability/traces` | `E0–E6` | Tất định | YES | — |
| `SPT-S-027` | `EVENT` | `provider_integration` | FeedBeta đổi kiểu trường `minute` từ số nguyên sang chuỗi `"45+2"` (schema drift) | **NO** — cách ly payload sai schema, báo chủ dữ liệu, không nuốt im (P4) | `data/ingestion` · `data/normalization` · `execution/workflows` | `E0–E6` | Tất định | YES | — |
| `SPT-S-028` | `EVENT` | `player_identity` | Kiều Anh Bảo chuyển từ Cảng Biển Athletic sang Phố Núi United ngày 15/07/2026; FeedGamma vẫn ghi đội cũ | **NO** — đội của cầu thủ phụ thuộc thời điểm; cần quan hệ có hiệu lực theo thời gian | `data/entityresolution` · `domain/sportsevents` | `E0–E6` | Tất định | NO — thiếu `temporal_relations` | — |
| `SPT-S-029` | `EVENT` | `player_identity` | FeedGamma gửi "HAU L." ghi bàn *(độ tin cậy thấp, trùng tên với cầu thủ trẻ)* | **NO** — so khớp mờ dưới ngưỡng không được tự chấp nhận; cần trạng thái tạm + việc cho người | `data/entityresolution` · `execution/workflows` | `E0–E6` | Tất định | YES | — |
| `SPT-S-030` | `EVENT` | `cards` | Thẻ đỏ chỉ **một** provider báo (FeedBeta), hai nguồn kia chưa có | **NO** — thẻ đỏ có hệ quả lớn; cần quorum guard và nhãn tạm, không phải "tin nguồn đến trước" | `data/sourceoftruth` · `execution/workflows` | `E0–E6` | Tất định | YES | — |
| `SPT-S-031` | `EVENT` | `live_match` | Bóng rổ (Cửu Long Tigers – Hương Giang Hawks): đồng hồ **đếm ngược** `Q4 00:42` và hiệp phụ | **NO** — mô hình kỳ/đồng hồ khác bóng đá (đếm ngược, 4 hiệp, hiệp phụ); RAG không có mô hình này | `data/normalization` · `domain/sportsevents` | `E0–E6` | Tất định | YES | — |
| `SPT-S-032` | `EVENT` | `competition_mapping` | Provider gửi giải với **tên thương mại mới + mã mới + mùa mới** (Sao Mai Championship, mùa 2026/27) | **NO** — liên tục danh tính giải qua đổi tên là ánh xạ theo thời gian + người duyệt | `data/entityresolution` · `execution/workflows` | `E0–E6` | Tất định | NO — thiếu `temporal_relations` | — |
| `SPT-S-033` | `EVENT` | `fixture` | FeedBeta gửi giờ khai cuộc `2026-10-18 19:30:00` **không kèm offset**; lệch với lịch chính thức | **NO** — múi giờ là thuộc tính cấu hình của provider; sai múi giờ lệch 7 giờ, "trận hôm nay" sai | `data/normalization` · `data/sourceoftruth` | `E0–E6` | Tất định | YES | — |

---

## 4. COMMAND, WORKFLOW, RECONCILIATION, SECURITY

### 4.1 COMMAND — lệnh làm đổi trạng thái (10)

Lệnh đi pipeline C0–C7: nhận (idempotency key) → tenant → hiểu lệnh → kiểm schema tham số → quyền → **workflow engine: guard + chuyển trạng thái trong MỘT giao dịch** → tool có tác dụng phụ → audit + thông báo. Lệnh thiếu tham số bắt buộc thì hỏi lại, không đoán.

| ID | entry_type | Domain | Lệnh / tình huống | RAG một mình? | Capability | Pipeline | Tính chất | Biểu diễn được? | MVP |
|---|---|---|---|---|---|---|---|---|---|
| `SPT-S-034` | `COMMAND` | `data_correction` | Biên tập viên: "Sửa người ghi bàn phút 23 trận Sông Hồng – Phố Núi từ Lương Đình Hậu sang Trương Quang Tín; lý do: biên bản trọng tài." | **NO** — RAG không ghi trạng thái; sửa dữ liệu live là lệnh có quyền, lý do, audit | `core/pipeline` · `governance/permissions` · `execution/workflows` · `execution/tools` · `governance/audit` | `C0–C7` | Tất định | YES | MVP |
| `SPT-S-035` | `COMMAND` | `live_match` | Biên tập viên: "Khoá trận MATCH_0000912" sau khi sửa tay, để provider không ghi đè | **NO** — khoá là trạng thái + guard `match_not_locked`, không phải tri thức | `execution/workflows` · `domain/sportsevents` · `governance/audit` | `C0–C7` | Tất định | YES | — |
| `SPT-S-036` | `COMMAND` | `player_identity` | Vận hành dữ liệu: "Gộp PLAYER_04571 và bản ghi trùng PLAYER_04571B" *(hai bản ghi cùng một người)* | **NO** — gộp thực thể phải chuyển hướng mọi tham chiếu, tính lại thống kê, đảo ngược được | `data/entityresolution` · `execution/workflows` · `domain/sportsevents` | `C0–C7` | Tất định | NO — thiếu `entity_merge_split` | — |
| `SPT-S-037` | `COMMAND` | `data_correction` | Người duyệt: "Công bố bản sửa đã APPROVED" → sinh sự kiện sửa, tính lại tỉ số và bảng xếp hạng, vô hiệu cache, thông báo đính chính | **NO** — công bố là chuyển trạng thái kéo theo dây chuyền tính lại | `execution/workflows` · `domain/sportsevents` · `data/sourceoftruth` · `execution/triggers` | `C0–C7` | Tất định | NO — thiếu `derived_fact_invalidation`, `read_model_cache_contract`, `supersedable_notification` | — |
| `SPT-S-038` | `COMMAND` | `data_correction` | Biên tập viên: "Rút lại bàn thắng vừa nhập nhầm" trong 5 phút *(giả lập)* — không cần người duyệt thứ hai; quá 5 phút phải qua duyệt | **NO** — ngoại lệ "tự sửa trong cửa sổ ngắn" là guard `own_entry_within_grace`, không phải văn bản | `execution/workflows` · `governance/permissions` | `C0–C7` | Tất định | YES | — |
| `SPT-S-039` | `COMMAND` | `team_identity` | Vận hành dữ liệu: "Thêm bí danh 'Pho Nui Utd' của FeedBeta cho Phố Núi United" *(bị chặn nếu bí danh đã thuộc đội khác)* | **NO** — thêm bí danh là ghi vào `entity_aliases` có nguồn, kiểm trùng bằng guard `alias_not_taken` | `data/entityresolution` · `execution/workflows` · `governance/audit` | `C0–C7` | Tất định | YES | — |
| `SPT-S-040` | `COMMAND` | `fixture` | Biên tập viên: "Đổi giờ đá trận … sang 20:00 ngày 19/10 giờ Việt Nam" *(chỉ khi trận còn `SCHEDULED`)* | **NO** — đổi lịch là chuyển trạng thái có guard `kickoff_not_started` và thông báo | `execution/workflows` · `data/normalization` · `domain/sportsevents` | `C0–C7` | Tất định | YES | — |
| `SPT-S-041` | `COMMAND` | `event_timeline` | Vận hành dữ liệu: "Chạy lại projection cho cả mùa" *(tác dụng phụ lớn)* | **NO** — lệnh nặng phải thành human task chờ duyệt rồi chạy idempotent | `execution/workflows` · `domain/sportsevents` · `platform/store` | `C0–C7` | Tất định | YES | — |
| `SPT-S-042` | `COMMAND` | `provider_integration` | Kỹ sư tích hợp: "Hạ ưu tiên FeedGamma ở Giải Vô địch Lạc Hồng từ 20:00 hôm nay" | **NO** — đổi chính sách nguồn lúc chạy, có hiệu lực từ thời điểm T, có audit | `core/tenant` · `data/sourceoftruth` · `governance/audit` | `C0–C7` | Tất định | NO — thiếu `runtime_config_override` | — |
| `SPT-S-043` | `COMMAND` | `notification` | Người hâm mộ: "Theo dõi Sông Hồng FC: báo bàn thắng và kết quả" *(gửi lại ba lần do mạng chập chờn)* | **NO** — đăng ký là lệnh idempotent, không phải câu hỏi | `core/pipeline` · `execution/tools` · `governance/permissions` | `C0–C7` | Tất định | YES | — |

### 4.2 WORKFLOW — vòng đời có trạng thái (9)

Trạng thái và phép chuyển do **workflow engine** sở hữu; guard chỉ là **tên** chọn từ danh mục mã đã cài (P12), danh sách guard ở [04-realtime-flow.md](04-realtime-flow.md) §7. Model chỉ **đề xuất** (P11).

| ID | entry_type | Domain | Vòng đời / tình huống | RAG một mình? | Capability | Pipeline | Tính chất | Biểu diễn được? | MVP |
|---|---|---|---|---|---|---|---|---|---|
| `SPT-S-044` | `WORKFLOW` | `live_match` | Vòng đời trận bóng đá đủ đường: `SCHEDULED → LIVE_1H → HT → LIVE_2H → FT → FINAL` (workflow `match-lifecycle`) | **NO** — trạng thái là của engine; RAG không giữ trạng thái | `execution/workflows` · `execution/triggers` · `domain/sportsevents` | `engine` | Tất định | YES | MVP |
| `SPT-S-045` | `WORKFLOW` | `live_match` | `FT → FINAL` bị chặn: còn VAR chờ, còn xung đột chưa giải, hoặc chưa hết cửa sổ muộn *(giả lập)* | **NO** — guard `no_pending_var`, `no_open_reconciliation`, `late_window_elapsed` | `execution/workflows` · `data/sourceoftruth` | `engine` | Tất định | YES | — |
| `SPT-S-046` | `WORKFLOW` | `fixture` | Nhánh hoãn: `SCHEDULED → POSTPONED → SCHEDULED` với thông báo chính thức và human task xác nhận | **NO** — chuyển nhánh cần nguồn chính thức + người xác nhận | `execution/workflows` · `execution/tools` | `engine` | Tất định | YES | — |
| `SPT-S-047` | `WORKFLOW` | `live_match` | Tạm dừng rồi bỏ cuộc: `LIVE_2H → SUSPENDED → ABANDONED → FINAL` (kết quả do liên đoàn quyết) | **NO** — kết quả bỏ cuộc là quyết định chính thức, không suy ra từ văn bản | `execution/workflows` · `data/sourceoftruth` | `engine` | Tất định | YES | — |
| `SPT-S-048` | `WORKFLOW` | `data_correction` | Duyệt bản sửa bốn mắt: `DRAFT → SUBMITTED → REVIEWING → APPROVED → PUBLISHED` (workflow `correction-request`) | **NO** — người yêu cầu ≠ người duyệt là guard `approver_differs_from_requester` | `execution/workflows` · `governance/permissions` · `governance/audit` | `engine` | Tất định | YES | — |
| `SPT-S-049` | `WORKFLOW` | `data_correction` | Bản sửa bị từ chối hoặc **hết hạn** (`REVIEWING → REJECTED / EXPIRED`) khi người duyệt vắng | **NO** — hạn chót + chuyển việc cho người khác khi quá hạn là hành vi của engine | `execution/workflows` | `engine` | Tất định | NO — thiếu `human_task_escalation` | — |
| `SPT-S-050` | `WORKFLOW` | `provider_integration` | Sự cố provider: `DETECTED → ACKNOWLEDGED → MITIGATING → RESOLVED → CLOSED` (workflow `provider-incident`) | **NO** — vòng đời sự cố có người nhận + hạn; điều kiện `RESOLVED` là guard `lag_below_threshold_for_window` | `execution/workflows` · `execution/triggers` · `observability/traces` | `engine` | Tất định | YES | — |
| `SPT-S-051` | `WORKFLOW` | `team_identity` | Duyệt ánh xạ bí danh: `CANDIDATE → REVIEWING → APPROVED → ACTIVE` (workflow `mapping-review`); model chỉ **đề xuất** ứng viên | **NO** — chấp nhận ánh xạ là quyết định của người + engine, model không tự ghi | `execution/workflows` · `data/entityresolution` · `intelligence/models` | `engine` | Lai (đề xuất xác suất, quyết định tất định) | YES | — |
| `SPT-S-052` | `WORKFLOW` | `content_generation` | Bản tin sau trận: `DRAFTED → EDITOR_REVIEW → APPROVED → PUBLISHED` (workflow `match-report`); chỉ khi trận `FINAL` và mọi khẳng định có bằng chứng | **PARTIAL** — model soạn được nhưng công bố là phép chuyển có guard `match_is_final`, `all_claims_grounded` | `execution/workflows` · `execution/skills` · `intelligence/models` · `reliability/validation` | `engine` | Lai | YES | — |

### 4.3 RECONCILIATION — nhiều nguồn nói khác nhau về cùng một sự việc (5)

Hoà giải chỉ áp giữa các quan sát về **cùng một sự kiện** (xem §1.3). Ma trận thẩm quyền theo từng trường ở [03-data-map.md](03-data-map.md).

| ID | entry_type | Domain | Xung đột | RAG một mình? | Capability | Pipeline | Tính chất | Biểu diễn được? | MVP |
|---|---|---|---|---|---|---|---|---|---|
| `SPT-S-053` | `RECONCILIATION` | `goals` | Người ghi bàn phút 23: FeedAlpha = Lương Đình Hậu, FeedBeta = Trương Quang Tín, FeedGamma = "HAU L." | **NO** — chọn theo thẩm quyền từng trường + đồng thuận, không theo "nguồn nói nhiều nhất" | `data/sourceoftruth` · `domain/sportsevents` · `data/entityresolution` | `E0–E6` | Tất định | NO — thiếu `incident_correlation` | MVP |
| `SPT-S-054` | `RECONCILIATION` | `cards` | Thẻ đỏ: FeedAlpha có, FeedBeta không, FeedGamma không, chưa có biên bản | **NO** — không giải được bằng thẩm quyền; phải thành việc cho biên tập viên | `data/sourceoftruth` · `execution/workflows` | `E0–E6` | Tất định | YES | — |
| `SPT-S-055` | `RECONCILIATION` | `live_match` | Trạng thái trận: FeedAlpha `FT`, FeedBeta còn `LIVE` phút 90, FeedGamma `HT` | **NO** — hoà giải bằng quorum + thời gian yên lặng + trạng thái chỉ đi tiến | `data/sourceoftruth` · `execution/workflows` | `E0–E6` | Tất định | YES | — |
| `SPT-S-056` | `RECONCILIATION` | `lineup` | Đội hình xuất phát: FeedAlpha và FeedBeta khác nhau một cầu thủ; biên bản liên đoàn đến sau | **NO** — thẩm quyền theo từng trường; nguồn chính thức đến sau thì thắng và tạo bản sửa | `data/sourceoftruth` · `domain/sportsevents` · `execution/workflows` | `E0–E6` | Tất định | YES | — |
| `SPT-S-057` | `RECONCILIATION` | `data_correction` | Biên bản chính thức sau trận đổi bàn 23' thành bàn phản lưới; người dùng hỏi "ban đầu hệ thống hiển thị gì?" | **NO** — cần trạng thái tại một thời điểm trong quá khứ (as-of), RAG không dựng lại được | `data/sourceoftruth` · `domain/sportsevents` · `governance/audit` | `E0–E6` | Tất định | NO — thiếu `as_of_query` | — |

### 4.4 SECURITY — chính sách, giấy phép, cô lập tenant (3)

| ID | entry_type | Domain | Tình huống | RAG một mình? | Capability | Route | Tính chất | Biểu diễn được? | MVP |
|---|---|---|---|---|---|---|---|---|---|
| `SPT-S-058` | `SECURITY` | `compliance` | "Kèo trận Sông Hồng – Phố Núi tối nay bao nhiêu? Nên đặt cửa nào?" và các biến thể: tài xỉu, tỉ lệ ăn, soi kèo, "dự đoán tỉ số để chơi" | **NO** — RAG/model sẽ đáp; chính sách phải chặn **trước** model (nền tảng không có tính năng cá cược) | `intelligence/intent` · `intelligence/router` · `governance/permissions` | `REJECT` | Tất định | YES | MVP |
| `SPT-S-059` | `SECURITY` | `compliance` | Khách `partner_basic` đòi trường gói premium (đội hình chi tiết) hoặc payload thô của FeedAlpha *(giấy phép cấm tái phân phối dữ liệu thô)* | **NO** — quyền theo gói và giấy phép là kiểm ngoài model | `governance/auth` · `governance/permissions` | `REJECT` | Tất định | YES | — |
| `SPT-S-060` | `SECURITY` | `compliance` | Người dùng gửi body `tenant_id: abc-infra` hoặc cố hỏi dữ liệu của tenant khác | **NO** — tenant lấy từ danh tính, không từ body; RAG dùng chung chỉ mục là rò rỉ | `governance/tenancy` · `governance/auth` | `REJECT` | Tất định | YES | — |

---

## 5. Primitive còn thiếu do thể thao chỉ ra

Theo nguyên tắc "capability phải sinh từ ca hỏng" (ADR-0002), mỗi kịch bản `NO` ở trên là một **ca hỏng của kiến trúc hiện tại**. Bảng dưới gom chúng theo primitive thiếu. Cột "Phân loại" theo CONVENTIONS §4.5 là **đề xuất**; chỉ vào core khi có ca hỏng tái lập được và test (xem [07-failure-catalog.md](07-failure-catalog.md) và [08-test-cases.md](08-test-cases.md)).

| Primitive thiếu | Vì sao hiện tại không đủ | Kịch bản gây ra | Phân loại đề xuất |
|---|---|---|---|
| `incident_correlation` | thẩm quyền chỉ áp giữa quan sát cùng sự kiện, nhưng khoá khớp chính xác (chỉ số, thực thể, kỳ) không nhận ra "ba quan sát này là cùng một bàn thắng" khi phút và id khác nhau; thiếu hợp đồng giữa domain module và `sourceoftruth` | 019, 053 | `DOMAIN MODULE` (`domain/sportsevents`), hợp đồng cấp khoá cho `data/sourceoftruth` |
| `event_time_watermark` | luồng E0–E6 xử lý "đúng một lần" nhưng không có thứ tự theo giờ trận, cửa sổ muộn, mốc chốt; trigger kích hoạt theo thứ tự đến | 022, 023, 024 | `DOMAIN MODULE` (`domain/sportsevents`); trigger đọc thay đổi của projection, không đọc quan sát thô |
| `derived_fact_invalidation` | derived fact ghi công thức và fact đầu vào nhưng không có cơ chế đánh dấu lỗi thời và tính lại khi fact đầu vào bị rút/sửa | 021, 037 | `SHARED CORE` (`data/sourceoftruth`); cả hai tenant cần |
| `read_model_cache_contract` | chưa có hợp đồng cache có phiên bản và vô hiệu hoá theo thay đổi nguồn; "chưa dùng cache" chưa trả lời bản đọc nhanh phải đúng sau khi rút bàn | 021, 037 | `CONFIGURABLE` (TTL, namespace theo tenant) + `DOMAIN MODULE` (khoá theo `state_version`) |
| `supersedable_notification` | thông báo là hành động E5 một chiều; chưa có liên kết thông báo ↔ fact và thông báo đính chính | 021, 037 | `CONFIGURABLE` (`execution/triggers` + mẫu câu của tenant) |
| `entity_merge_split` | `entities`/`entity_aliases` chưa có phép gộp/tách có chuyển hướng, giữ lineage, tính lại projection, đảo ngược được | 036 | `SHARED CORE` (`data/entityresolution`) |
| `temporal_relations` | quan hệ và bí danh không có hiệu lực theo thời gian (cầu thủ ↔ đội, tên giải theo mùa) | 005, 028, 032 | `SHARED CORE` (`data/entityresolution`) |
| `as_of_query` | `StructuredDataSource` không có tham số "trạng thái tại thời điểm hệ thống biết"; thanh tra và khiếu nại cần dựng lại điều đã công bố | 057 | `SHARED CORE` (hợp đồng `StructuredDataSource`) |
| `runtime_config_override` | pack là tệp tĩnh nạp lúc khởi động; đổi ưu tiên provider lúc chạy cần ghi đè có phiên bản, hiệu lực từ T, audit | 042 | `SHARED CORE` (`core/tenant`) |
| `human_task_escalation` | human task có hạn chót và người nhận, nhưng chưa nói chuyển người nhận khi quá hạn | 049 | `SHARED CORE` (`execution/workflows`) |

Mười primitive không phải mười lời hứa: mỗi cái chỉ được hiện thực khi ca hỏng tương ứng chạy đỏ trên kiến trúc hiện tại (bộ ca ở `cases.yaml`) và có test chứng minh bản sửa.

---

## 6. Thống kê và tự kiểm

Bảng đếm dưới đây được tính lại từ chính các bảng ở trên (lệnh kiểm ghi ở cuối).

| entry_type | Số kịch bản | Trong đó MVP |
|---|---|---|
| QUERY | 15 | 6 |
| EVENT | 18 | 5 |
| COMMAND | 10 | 1 |
| WORKFLOW | 9 | 1 |
| RECONCILIATION | 5 | 1 |
| SECURITY | 3 | 1 |
| **Tổng** | **60** | **15** |

| "RAG một mình giải được?" | Số kịch bản |
|---|---|
| YES | 1 |
| PARTIAL | 6 |
| NO | 53 |

| Kiến trúc hiện tại biểu diễn được? | Số kịch bản |
|---|---|
| YES | 46 |
| NO (thiếu primitive) | 14 |

Trong 15 kịch bản MVP có 9 kịch bản không phải `QUERY`.

Lệnh kiểm (chạy từ thư mục gốc repo, kết quả ghi trong báo cáo bàn giao):

```bash
python3 -I - <<'PY'
import re
t = open('docs/tenants/sports/05-problem-catalog.md', encoding='utf-8').read()
rows = re.findall(r'^\| `(SPT-S-\d{3})` \|(.*)\|\s*$', t, re.M)
print('kịch bản:', len(rows), 'duy nhất:', len({r[0] for r in rows}))
print('MVP:', sum(1 for _, b in rows if b.rstrip().endswith('| MVP')))
PY
```
