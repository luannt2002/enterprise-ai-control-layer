# 01 — Bản đồ nghiệp vụ (tenant `xyz-sports`)

> **DOCS_ONLY** (nhãn 24/10/2026): tenant này mới có tài liệu ở `docs/tenants/sports/` và bộ ca `tenants/sports/evals/cases.yaml`;
> **chưa có pack** nên chưa chạy được trên `cmd/eais`. Chữ "tenant tham chiếu" dưới đây là vai trò dự kiến, chưa phải hiện trạng (ADR-0015 mục 5).

> **XYZ Sports Data JSC** là công ty **hư cấu**; mọi giải, đội, cầu thủ, provider trong tài liệu cũng hư cấu. Mọi con số (khách hàng, thời gian, mục tiêu KPI) là **giả lập hoặc mục tiêu thiết kế**, chưa đo. Chỉ dữ liệu thông tin thể thao: **không có tính năng cá cược**; câu hỏi về kèo, tỉ lệ cược, dự đoán kết quả đi route `REJECT`.

Tenant `xyz-sports` (mã `SPT`) là tenant tham chiếu thứ hai của EAIS, dùng để **tấn công** kiến trúc: dữ liệu thời gian thực nhiều nguồn, thứ tự sự kiện, nhận dạng thực thể, sửa muộn, cache, độ trễ. Tài liệu này dựng nghiệp vụ **trước** khi nói tới AI.

## 1. Mô hình kinh doanh

| Thành phần | Nội dung |
|---|---|
| Giá trị cung cấp | dữ liệu trận đấu đúng, nhanh, nhất quán từ nhiều nguồn: tỉ số, diễn biến, đội hình, thống kê, bảng xếp hạng, lịch, lịch sử |
| Sản phẩm | **XYZ Live** (app và web cho người hâm mộ) · **XYZ Data API** (B2B, theo gói cơ bản/nâng cao) · **XYZ Widgets** (khung hiển thị cho đối tác truyền thông) |
| Nguồn doanh thu (giả định) | quảng cáo/tài trợ trên app; thuê bao API theo gói; cấp phép widget |
| Chi phí chính (giả định) | giấy phép dữ liệu provider; hạ tầng; biên tập |
| Môn | bóng đá (MVP); bóng rổ, quần vợt, bóng chuyền, bóng chày, cầu lông ở mức mô hình |
| Ngoài phạm vi | **cá cược** (kèo, tỉ lệ, tư vấn đặt cược), dự đoán kết quả, dữ liệu cá nhân người chơi |

Nền tảng nhận dữ liệu từ nhiều provider (hư cấu: **FeedAlpha**, **FeedBeta**, **FeedGamma**) và biên bản chính thức của liên đoàn (`LEAGUE_OFFICIAL`). Giá trị nằm ở việc **hợp nhất đúng** (nhận dạng, khử trùng, sửa muộn), không ở việc có thêm một nguồn.

## 2. Khách hàng

| Nhóm | Họ cần gì | Gói / quyền | Ghi chú |
|---|---|---|---|
| Người hâm mộ (app/web) | tỉ số tức thời, diễn biến, bảng xếp hạng, lịch, thông báo | `fan_anonymous`, `fan_registered` | số lượng lớn, đọc nhiều, nhạy độ trễ |
| Đối tác truyền thông | widget, bản tin, đồ hoạ tỉ số | `media_partner` | có thể phải chịu độ trễ tối thiểu theo giấy phép |
| Khách hàng API (B2B) | dữ liệu chuẩn hoá qua API | `partner_basic`, `partner_premium` | trường nâng cao (đội hình chi tiết, thống kê sâu) chỉ ở gói cao; **không** nhận payload thô của provider |
| Nội bộ | biên tập, vận hành, hỗ trợ, kỹ thuật, sản phẩm | `editor`, `editor_approver`, `data_ops`, `integration_engineer`, `support_agent`, `product_manager`, `platform_admin` | dùng trợ lý hỏi đáp và lệnh nghiệp vụ |

Hệ thống adapter provider chạy dưới principal hệ thống `provider_adapter`.

## 3. Phòng ban, actor, hệ thống

| Phòng ban | Actor chính | Hệ thống chạm tới |
|---|---|---|
| Vận hành dữ liệu (Data Operations) | nhân viên vận hành dữ liệu (`data_ops`) | event store, `entity_aliases`, hàng đợi duyệt, bảng giám sát |
| Tích hợp provider (Provider Integration) | kỹ sư tích hợp (`integration_engineer`) | adapter provider, cấu hình thẩm quyền, chỉ số sức khoẻ provider |
| Biên tập (Editorial) | biên tập viên (`editor`), người duyệt (`editor_approver`) | workflow sửa dữ liệu, bài viết, lịch, đội hình |
| Sản phẩm (Product) | quản lý sản phẩm (`product_manager`) | gói API, thông báo, thống kê dùng |
| Hỗ trợ khách hàng (Customer Support) | nhân viên hỗ trợ (`support_agent`) | tra cứu chỉ đọc, nâng yêu cầu sửa |
| Kỹ thuật (Engineering/Platform) | kỹ sư nền tảng (`platform_admin`) | PostgreSQL, triển khai, quan sát, chi phí |

Hệ thống ngoài: provider (Alpha, Beta, Gamma), cổng biên bản liên đoàn, dịch vụ thông báo đẩy. Hệ thống nội bộ: EAIS (ask/command/event pipeline), event store `match_events`, `match_states`, kho tài liệu (điều lệ, bài viết).

## 4. Quy trình theo phòng ban

Mỗi phòng ban có **một bảng** với hai luồng chính. Dòng: Actor → Trigger → Input → Process → Decision → Output → Downstream → Data produced → System touched → Exceptions → Failure consequences.

### 4.1 Vận hành dữ liệu

| | Luồng A: hoà giải xung đột provider | Luồng B: duyệt ánh xạ và gộp thực thể |
|---|---|---|
| Actor | `data_ops` | `data_ops` |
| Trigger | hàng đợi có bất đồng chưa giải (người ghi bàn, thẻ đỏ, trạng thái) | ứng viên bí danh dưới ngưỡng; hai bản ghi cầu thủ trùng |
| Input | các quan sát cùng một sự kiện + ma trận thẩm quyền | ứng viên ánh xạ kèm độ tin cậy; hồ sơ hai thực thể |
| Process | xem bằng chứng từng nguồn; chọn theo thẩm quyền; hỏi biên tập nếu cần | duyệt hoặc từ chối; gộp có chuyển hướng |
| Decision | giải được bằng thẩm quyền? hay chuyển biên tập | bí danh có thuộc đội khác không; gộp có đảo ngược được không |
| Output | sự kiện hiệu lực có nhãn; việc cho biên tập | bí danh `ACTIVE`; thực thể bị gộp trỏ về thực thể chính |
| Downstream | projection, API, thông báo | mọi sự kiện, thống kê, tìm kiếm theo thực thể |
| Data produced | `fact_conflicts`, nhật ký quyết định | `entity_aliases`, quan hệ gộp |
| System touched | event store, `human_tasks` | `entities`, `entity_aliases`, projection |
| Exceptions | nguồn chính thức đến sau, khác | bí danh trùng đội khác; gộp nhầm |
| Failure consequences | người ghi bàn hoặc thẻ sai tới người dùng | bàn gán sai cầu thủ; thống kê sai kéo dài |

### 4.2 Tích hợp provider

| | Luồng A: thêm và giám sát provider | Luồng B: đổi cấu hình thẩm quyền |
|---|---|---|
| Actor | `integration_engineer` | `integration_engineer` |
| Trigger | ký giấy phép mới; sự cố (im lặng, 401, đổi schema) | chất lượng một provider giảm ở một giải |
| Input | tài liệu hợp đồng, schema payload, khoá API | số liệu chênh lệch, độ trễ theo provider |
| Process | viết adapter có phiên bản; kiểm hợp đồng; theo dõi nhịp | tạo bản ghi đè hạng thẩm quyền có hiệu lực từ giờ T |
| Decision | payload sai schema thì cách ly hay nhận | hạ ưu tiên ở giải nào, từ khi nào |
| Output | luồng quan sát ổn định; sự cố có người nhận | cấu hình có phiên bản |
| Downstream | normalization, entity resolution | hoà giải các quan sát mới |
| Data produced | `provider_events_raw`, chỉ số `provider_lag_seconds`, `workflow_instances` | bản ghi đè cấu hình, audit |
| System touched | adapter, hàng đợi, workflow `provider-incident` | pack, `core/tenant` |
| Exceptions | provider đổi nghĩa mã trạng thái; khoá hết hạn; bị giới hạn tần suất | đổi hồi tố (không cho phép) |
| Failure consequences | mất sự kiện thầm lặng; dữ liệu cũ không cờ | hoà giải sai nguồn |

### 4.3 Biên tập

| | Luồng A: sửa dữ liệu trận | Luồng B: bản tin và lịch |
|---|---|---|
| Actor | `editor`, `editor_approver` | `editor` |
| Trigger | biên bản chính thức khác dữ liệu đã công bố; khiếu nại | trận `FINAL`; liên đoàn đổi lịch |
| Input | sự kiện đích + lý do + bằng chứng | sự kiện hiệu lực; thông báo chính thức |
| Process | tạo bản sửa → duyệt bởi người khác → công bố | duyệt bản tin nháp do model soạn; xác nhận hoãn/đổi giờ |
| Decision | bản sửa có hợp lệ và chưa lỗi thời | bản tin có khẳng định nào thiếu bằng chứng |
| Output | sự kiện sửa mới (không sửa đè) | bản tin công bố; lịch mới |
| Downstream | tỉ số, bảng xếp hạng, cache, thông báo đính chính | app, đối tác, thông báo đổi giờ |
| Data produced | `correction-request`, `match_events` mới, audit | `match-report`, `fixtures` phiên bản mới |
| System touched | workflow engine, event store | workflow `match-report`, `fixtures` |
| Exceptions | người duyệt vắng; trận bị khoá | bản sửa còn mở khi muốn công bố |
| Failure consequences | kết quả đã chốt đổi không dấu vết, hay sai kéo dài | bản tin nêu kết quả sắp đổi |

### 4.4 Sản phẩm

| | Luồng A: gói và quyền truy cập | Luồng B: thông báo và theo dõi |
|---|---|---|
| Actor | `product_manager` | `product_manager`, người hâm mộ |
| Trigger | khách hàng đổi gói; ra trường dữ liệu mới | người dùng theo dõi đội |
| Input | bảng gói ↔ trường dữ liệu | đăng ký theo dõi, sự kiện hiệu lực |
| Process | cấu hình quyền theo gói | gửi thông báo khi sự kiện **đã xác nhận** |
| Decision | trường nào thuộc gói nào | gửi thông báo hay chờ xác nhận |
| Output | khách chỉ thấy đúng trường | một thông báo mỗi sự kiện; đính chính khi rút |
| Downstream | API, widget | app người dùng |
| Data produced | `api_clients`, `api_usage` | `subscriptions`, `notification_log` |
| System touched | cổng API, `governance/permissions` | dịch vụ thông báo, workflow |
| Exceptions | khách đòi payload thô | gửi trùng khi mạng chập chờn |
| Failure consequences | rò trường nâng cao; vi phạm giấy phép | thông báo sai hoặc đúp |

### 4.5 Hỗ trợ khách hàng

| | Luồng A: tiếp nhận phản ánh sai dữ liệu | Luồng B: giải đáp câu hỏi người dùng |
|---|---|---|
| Actor | `support_agent` | `support_agent` |
| Trigger | khách báo tỉ số hay người ghi bàn sai | khách hỏi lịch, giờ, thể thức |
| Input | mô tả của khách; dữ liệu hiện hành | câu hỏi |
| Process | tra cứu chỉ đọc; nâng yêu cầu cho biên tập | dùng trợ lý hỏi đáp (SQL hoặc RAG theo câu hỏi) |
| Decision | có đủ bằng chứng để nâng yêu cầu? | dữ kiện live hay tri thức luật |
| Output | yêu cầu có người nhận | câu trả lời có nguồn |
| Downstream | biên tập | khách hàng |
| Data produced | `human_tasks`, ghi chú | `interactions` |
| System touched | tra cứu, hàng đợi biên tập | `/v1/ask` |
| Exceptions | nhân viên muốn sửa trực tiếp (bị chặn) | câu hỏi về kèo (bị từ chối) |
| Failure consequences | sửa không qua duyệt | trả lời kèo vi phạm chính sách |

### 4.6 Kỹ thuật

| | Luồng A: vận hành luồng sự kiện | Luồng B: dựng lại và phát lại |
|---|---|---|
| Actor | `platform_admin` | `data_ops`, `platform_admin` |
| Trigger | độ trễ hay hàng đợi vượt ngưỡng; sự cố hạ tầng | cần tính lại sau sửa lớn hoặc đổi quy tắc xếp hạng |
| Input | chỉ số quan sát; nhật ký | danh sách trận; `match_events` |
| Process | phát hiện, giảm thiểu, theo dõi sự cố | duyệt lệnh nặng; chạy lô idempotent |
| Decision | có cần mở năng lực mới (hàng đợi/cache) không: chỉ khi **số đo** đòi | dừng lô nếu lỗi |
| Output | luồng khoẻ; số đo | projection bằng phát lại |
| Downstream | mọi phòng ban | API, bảng xếp hạng |
| Data produced | `traces`, chỉ số | `match_states`, `standings` tính lại |
| System touched | PostgreSQL, quan sát | worker projection |
| Exceptions | hai worker cùng một trận | lô lỗi giữa chừng |
| Failure consequences | mất cập nhật; dữ liệu cũ | trạng thái nửa vời |

---

## 5. Luật nghiệp vụ

Mã luật (slug) dùng làm khoá `business_rules` trong [cases.yaml](../../../tenants/sports/evals/cases.yaml). Cột "Nơi cưỡng chế" là package/cơ chế (không phải cấu hình YAML có điều kiện).

| Mã luật | Phát biểu | Nơi cưỡng chế |
|---|---|---|
| `events_are_append_only` | sự kiện chỉ ghi thêm, không sửa, không xoá | `domain/sportsevents`, `platform/store` |
| `correction_is_new_event` | sửa và rút lại là sự kiện mới trỏ về sự kiện cũ | `domain/sportsevents` |
| `score_is_derived_from_effective_goals` | tỉ số là phép tính từ bàn thắng hiệu lực, không ghi tay hay lấy từ provider | projection |
| `dedup_before_projection` | khử trùng trước khi tính | `domain/sportsevents` |
| `reconcile_only_same_incident` | thẩm quyền/đồng thuận chỉ áp giữa quan sát cùng một sự kiện | `domain/sportsevents` + `data/sourceoftruth` |
| `order_by_match_time_not_arrival` | thứ tự theo giờ trận, không theo thứ tự đến | projection |
| `late_window_before_final` | sự kiện muộn trong cửa sổ muộn vẫn được chèn; `FINAL` chỉ sau khi hết cửa sổ | guard `late_window_elapsed` |
| `final_is_closed_except_correction` | sau `FINAL` chỉ đổi qua workflow sửa dữ liệu | `execution/workflows` |
| `status_only_moves_forward` | trạng thái trận chỉ đi tiến | guard `forward_only` |
| `authority_applies_per_field` | thẩm quyền nguồn tính theo từng trường | `data/sourceoftruth` |
| `authority_not_recency` | nguồn đến sau không tự đúng hơn | `data/sourceoftruth` |
| `official_source_outranks_providers` | nguồn chính thức thắng mọi provider | ma trận thẩm quyền |
| `official_report_wins_after_final` | biên bản chính thức sau trận thắng dữ liệu đã công bố; sửa qua bản sửa | workflow `correction-request` |
| `red_card_needs_quorum_or_official` | thẻ đỏ cần hai nguồn hoặc nguồn chính thức | guard `provider_quorum` |
| `unresolved_conflict_goes_to_human` | xung đột không giải được thì thành việc cho người | `execution/workflows` |
| `provisional_label_required` | dữ liệu chưa xác nhận hay đang bất đồng phải mang nhãn | API |
| `stale_data_must_be_flagged` | dữ liệu có thể đã cũ phải kèm cờ `stale` và `as_of` | API, trigger |
| `entity_must_fit_match_context` | cầu thủ/đội phải thuộc đội hình/hai đội của trận | `data/entityresolution` |
| `affiliation_is_time_bounded` | đội của cầu thủ, tên giải theo hiệu lực thời gian | quan hệ có hiệu lực |
| `competition_identity_survives_rename` | giải đổi tên vẫn là một giải | `mapping-review` |
| `ambiguous_entity_asks_back` | thực thể mơ hồ thì hỏi lại, không đoán | `reliability/clarification` |
| `missing_parameter_asks_back` | lệnh thiếu tham số bắt buộc thì hỏi lại | pipeline lệnh C2–C3 |
| `merge_is_reversible` | gộp thực thể có chuyển hướng và đảo ngược được | `data/entityresolution` |
| `quarantine_invalid_payload` | payload sai schema hay ngoài miền thì cách ly và báo chủ dữ liệu | `data/ingestion` |
| `timezone_explicit_utc_internal` | thời điểm lưu UTC; múi giờ khai báo rõ khi hiển thị và khi nạp | `data/normalization` |
| `period_model_per_sport` | mỗi môn có mô hình kỳ/đồng hồ riêng | `domain/sportsevents` |
| `cache_keyed_by_state_version` | bản đọc cache khoá theo `state_version` và tenant | cache |
| `retraction_invalidates_derived_and_cache` | rút lại làm tính lại dẫn xuất và vô hiệu cache | projection |
| `notify_only_confirmed_events` | chỉ thông báo sự kiện đã xác nhận; đính chính khi rút | `execution/triggers` |
| `live_facts_never_from_documents` | dữ kiện live chỉ từ sự kiện, không từ tài liệu/RAG | Context Engine |
| `numbers_must_match_evidence` | số trong câu trả lời phải có trong bằng chứng | `reliability/validation` |
| `model_never_invents_events` | model không bịa sự kiện, người ghi bàn, phút, nguyên nhân | structured output + validator |
| `model_proposes_engine_decides` | model chỉ đề xuất; engine + guard đổi trạng thái | `execution/workflows` |
| `corrections_need_reason_and_four_eyes` | sửa dữ liệu cần lý do, bằng chứng và người duyệt khác (trừ cửa sổ tự sửa ngắn) | guard của `correction-request` |
| `commands_are_idempotent` | cùng idempotency key chỉ thực thi một lần | pipeline lệnh C0 |
| `lock_blocks_provider_override` | trận đã khoá không bị quan sát provider ghi đè | guard `match_not_locked` |
| `no_betting_content` | không kèo, tỉ lệ cược, tư vấn đặt cược | route `REJECT` |
| `no_result_prediction` | không dự đoán kết quả trận | route `REJECT` |
| `no_raw_provider_redistribution` | không trả payload thô của provider cho khách | `governance/permissions` |
| `tier_gates_data_fields` | trường dữ liệu theo gói | `governance/permissions` |
| `tenant_from_identity_only` | tenant lấy từ danh tính khoá API, không từ body | `governance/auth` |

## 6. Tuân thủ

- **Bản quyền và giấy phép dữ liệu provider (khái quát):** dữ liệu của provider có thể bị hạn chế về hiển thị công khai, độ trễ tối thiểu cho gói thấp, **cấm tái phân phối payload thô**, và cấm dùng cho cá cược. Hệ thống chỉ xuất dữ liệu chuẩn hoá, kiểm trên đầu ra, không chuyển tiếp payload gốc. Điều khoản cụ thể: **chưa có** (hợp đồng là dữ liệu của tenant, không nằm trong repo).
- **Không cá cược:** nền tảng không cung cấp kèo, tỉ lệ, dự đoán; điều khoản khách hàng cấm dùng dữ liệu cho cá cược; câu hỏi liên quan đi `REJECT`.
- **Dữ liệu cá nhân:** chỉ mã thiết bị nhẹ cho thông báo; không lưu thông tin người chơi. Cầu thủ trong tài liệu là hư cấu.
- **Nội dung biên tập:** bản tin do model soạn phải qua người duyệt trước khi công bố.

## 7. Rủi ro

| Rủi ro | Hệ quả | Giảm thiểu | Ca hỏng |
|---|---|---|---|
| tỉ số sai tới người dùng | mất tin cậy | tỉ số dẫn xuất, bất biến, khử trùng | `SPT-F-001`, `SPT-F-003`, `SPT-F-031` |
| provider im lặng hay đổi hợp đồng | dữ liệu cũ hay mất | cờ `stale`, sự cố, cách ly payload | `SPT-F-008`, `SPT-F-035` |
| nhận dạng sai đội/cầu thủ | bàn gán nhầm | bí danh có nguồn và hiệu lực, hỏi lại | `SPT-F-017`–`SPT-F-023` |
| cache cũ sau khi sửa | người dùng thấy kết quả đã huỷ | khoá theo `state_version` | `SPT-F-012`–`SPT-F-016` |
| vi phạm giấy phép | rủi ro pháp lý | không xuất payload thô, quyền theo gói | `SPT-F-060`, `SPT-F-061` |
| nội dung cá cược | trái chính sách | `REJECT` ngoài model | `SPT-F-040` |
| model bịa sự kiện | thông tin sai | validator + mẫu câu lùi | `SPT-F-047`, `SPT-F-048` |

## 8. KPI

Các mục tiêu là **giả lập**, chưa đo; bộ dữ liệu đo là `cases.yaml` và bộ sinh dữ liệu có seed (chưa có). Không có số "chính xác X%" nào ở đây được hiểu là đã đạt.

| Nhóm | Chỉ số | Cách đo (khi có) | Mục tiêu |
|---|---|---|---|
| Độ trễ dữ liệu | từ lúc provider gửi tới lúc API trả bản mới (theo provider) | `provider_lag_seconds`, `ingest_to_visible` | theo [04 §10](04-realtime-flow.md#10-mục-tiêu-độ-trễ-slo--chưa-đo) |
| Độ đúng | tỉ lệ sự kiện hiệu lực đúng biên bản chính thức | so với `LEAGUE_OFFICIAL` trên bộ mô phỏng | chưa đo |
| Độ đúng | tỉ lệ ca khử trùng đúng; tỉ lệ nhận dạng thực thể đúng | `dedup_accuracy`, `entity_resolution_accuracy` | chưa đo |
| Độ phủ | số giải và số môn có dữ liệu | đếm | MVP: một môn |
| Độ tin cậy | số lần bản đọc cũ hơn `state_version`; số vi phạm chính sách | `cache_staleness_rate`, `permission_violations` | 0 vi phạm chính sách là yêu cầu cứng |
| Hỗ trợ | tỉ lệ yêu cầu sửa được xử lý đúng quy trình | `workflow_success` | chưa đo |
