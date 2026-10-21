# 04 — Luồng dữ liệu thời gian thực (tenant `xyz-sports`)

> Mọi giải, đội, cầu thủ, provider là **hư cấu**. Mọi con số về thời gian (giây, phút, cửa sổ) là **mục tiêu thiết kế hoặc mặc định cấu hình giả lập**, **chưa đo**; lệnh đo sẽ có khi có `eais simulate` và bộ chạy ca. Chỉ dữ liệu thông tin thể thao, không có tính năng cá cược.

Tài liệu mô tả luồng pipeline sự kiện E0–E6 của kiến trúc áp vào thể thao (luồng nạp I4), tầng dữ liệu `OBSERVATION → MEASUREMENT → FACT → DERIVED FACT` (xem [03-data-map.md](03-data-map.md)), vòng đời trận như một máy trạng thái, và các ca hỏng realtime. Kịch bản ở [05-problem-catalog.md](05-problem-catalog.md), ca hỏng ở [07-failure-catalog.md](07-failure-catalog.md).

Thuật ngữ lần đầu: *event store* (kho sự kiện chỉ ghi thêm), *projection* (bản dẫn xuất tính từ sự kiện), *idempotent* (chạy nhiều lần cho cùng một kết quả), *watermark* (mốc chốt thời gian cho sự kiện muộn), *guard* (điều kiện chặn một phép chuyển trạng thái), *quorum* (số nguồn tối thiểu phải đồng ý), *SLO* (mục tiêu mức dịch vụ), *snapshot* (ảnh chụp trạng thái tại một lúc).

## 1. Năm nguyên tắc

1. **Chỉ ghi thêm.** `provider_events_raw` và `match_events` không bị sửa hay xoá. Sửa và rút lại là **sự kiện mới trỏ về sự kiện cũ** (`supersedes`), không sửa đè.
2. **Tỉ số, trạng thái hiển thị, bảng xếp hạng là DERIVED FACT**, tính từ sự kiện hiệu lực; không ai ghi chúng tay hay để provider ghi đè.
3. **Thứ tự theo giờ trận, không theo thứ tự đến.** Projection phải cho cùng kết quả với mọi hoán vị thứ tự đến.
4. **Đồng thuận và ưu tiên chỉ áp giữa các quan sát về CÙNG một sự kiện**, sau bước tương quan.
5. **Model không đổi trạng thái (P11).** Trạng thái trận, phép chuyển, quyền, bất biến thuộc workflow engine; model chỉ đọc, phân loại, đề xuất, giải thích.

---

## 2. Luồng đầy đủ

```text
Provider ─► Ingestion ─► Hàng đợi ─► Normalization ─► Entity resolution ─► Tương quan + khử trùng ─► Event store
(Alpha/Beta/Gamma/      (adapter,    (bảng PG,       (kỳ/phút/giờ,       (bí danh theo nguồn,    (hạng thẩm quyền          match_events
 LEAGUE_OFFICIAL)        nhận raw)    SKIP LOCKED)    mã trạng thái)      đội hình, hiệu lực)     theo trường, phiên bản)   (chỉ ghi thêm)
                                                                                                          │
                          Web/App ◄─ API ◄─ Cache ◄─ Projection (idempotent) ◄───────────────────────────┘
                                   (/v1/ask,   (khoá theo      match_states · standings · player_stats
                                    /v1/…)     state_version)         │
                                                                      └─► luật kích hoạt E3 ─► workflow engine ─► thông báo · việc cho người
```

| Chặng | Việc | Vào → ra | Idempotency | Ca hỏng điển hình | Package |
|---|---|---|---|---|---|
| Provider | đẩy hoặc để hỏi dữ liệu | payload thô | — | im lặng, trễ, đổi schema | — |
| Ingestion (E0) | nhận, kiểm schema, ghi nguyên văn | payload → `provider_events_raw` (OBSERVATION) | khoá `(tenant, provider, provider_event_ref)` duy nhất | `SPT-F-002`, `SPT-F-009`, `SPT-F-035`, `SPT-F-037` | `data/ingestion` |
| Hàng đợi | trung gian trong PostgreSQL: hàng chờ xử lý, lấy bằng `FOR UPDATE SKIP LOCKED` theo khoá trận | raw → lô | một trận một lúc một worker | `SPT-F-011` | `platform/store` |
| Normalization (E1) | kỳ, phút, bù giờ, múi giờ, mã trạng thái theo phiên bản adapter | OBSERVATION → MEASUREMENT | thuần hàm của (payload, phiên bản adapter) | `SPT-F-024`, `SPT-F-025`, `SPT-F-026`, `SPT-F-028` | `data/normalization` |
| Entity resolution | bí danh theo nguồn, ràng buộc đội hình và hiệu lực | tên → `team_id`, `player_id` + độ tin cậy | thuần hàm của (bí danh, ngày) | `SPT-F-017`–`SPT-F-023` | `data/entityresolution` |
| Tương quan + khử trùng | nhận ra "cùng một sự cố", rồi áp thẩm quyền theo trường | MEASUREMENT → FACT (`match_events`) | khoá khử trùng (xem §3) | `SPT-F-001`, `SPT-F-029`, `SPT-F-030` | `domain/sportsevents` + `data/sourceoftruth` |
| Event store | ghi thêm, có `version`, `supersedes`, `provider_refs` | FACT | chỉ ghi thêm | `SPT-F-003`, `SPT-F-006` | `domain/sportsevents`, `platform/store` |
| Projection | tính tỉ số, trạng thái, bảng xếp hạng từ sự kiện hiệu lực | FACT → DERIVED FACT (`match_states`) | tính lại được; ghi bằng `state_version` lớn hơn | `SPT-F-004`, `SPT-F-007`, `SPT-F-031` | `domain/sportsevents` |
| Cache | bản đọc nhanh | DERIVED FACT → bản đọc | khoá theo `state_version` | `SPT-F-012`–`SPT-F-016` | `domain/sportsevents` + `governance/tenancy` |
| API | `/v1/ask` (QUERY), `/v1/commands`, `/v1/events`, `/v1/workflows` | bản đọc → người dùng | idempotency key ở lệnh | `SPT-F-041`, `SPT-F-050` | `platform/httpapi`, `core/pipeline` |
| Web/App | hiển thị | — | — | — | — |

---

## 3. Khoá khử trùng ba tầng

| Tầng | Khoá | Dùng cho | Ví dụ |
|---|---|---|---|
| K1 chính xác | `(tenant, provider, provider_event_ref)` | phát lại cùng một provider | FeedAlpha gửi lại `a-77120` sau khi kết nối lại: bỏ qua, ghi nhận bộ đếm |
| K2 theo hệ quả | `(match_id, kind, team_id, score_after)` khi provider gửi `score_after` | bàn thắng có tỉ số sau bàn | Alpha `score_after 1-1` cho PNU: cùng một bàn nếu Beta/Gamma khớp đội và kỳ |
| K3 cửa sổ thời gian | `(match_id, kind, team_id, player_id hoặc null, kỳ, độ lệch phút ≤ dung sai)` | provider không có id chung | bàn 45+2' (Alpha), 47 (Beta, kèm kỳ 1), 45 (Gamma): trong dung sai 2 phút (mặc định giả lập) |

Quy tắc: thử K1, rồi K2, rồi K3. Nếu K3 trả **hơn một** ứng viên (ví dụ hai bàn của cùng cầu thủ trong 2 phút), không tự gộp: giữ hai sự kiện `provisional` và mở việc. Ba quan sát gộp được là **một** sự cố, không phải xung đột (`SPT-F-030`). Tương quan này là *`incident_correlation`* ([05 §5](05-problem-catalog.md#5-primitive-còn-thiếu-do-thể-thao-chỉ-ra)): chưa có trong core; đặt ở `domain/sportsevents` và cấp khoá cho `data/sourceoftruth`.

---

## 4. Thứ tự, sự kiện đến muộn, sự kiện đảo thứ tự

- **Thứ tự chuẩn của sự kiện** trong một trận: (thứ tự kỳ, `clock`, `stoppage`, thứ tự trong phút, `ingested_at`). Bóng rổ đếm ngược: quy ra thời gian đã trôi trong kỳ. Thứ tự theo `store_seq` (đến trước ghi trước) chỉ dùng làm bước phá thế cân bằng cuối.
- **Đảo thứ tự đến:** projection sắp lại theo thứ tự chuẩn rồi tính; tính chất kiểm thử: mọi hoán vị thứ tự đến cho cùng `match_states` (ca `SPT-T-022`).
- **Sự kiện muộn:** trận chưa `FINAL` thì chèn đúng chỗ theo giờ trận, tăng `state_version`; trận `FT` giữ `FT` và **hoãn `FT → FINAL`** tới hết **cửa sổ muộn** (mặc định 15 phút, giả lập). Đây là *`event_time_watermark`* ([05 §5](05-problem-catalog.md#5-primitive-còn-thiếu-do-thể-thao-chỉ-ra)).
- **Sau `FINAL`:** quan sát muộn **không** được áp thầm lặng; sinh ứng viên sửa (`correction-request` ở `DRAFT`) và việc cho biên tập viên (ca `SPT-T-024`).
- **Trạng thái trận chỉ đi tiến:** snapshot cũ không kéo trận lùi (guard `forward_only`).
- **Luật kích hoạt (E3) đọc thay đổi của projection**, không đọc quan sát thô, để không báo bàn thắng chưa khử trùng (`SPT-F-059`).
- **Khoảng trống số thứ tự (seq)** của provider có số thứ tự: bộ dò theo trận, yêu cầu gửi bù (`SPT-F-009`).

---

## 5. Sửa và rút lại

Sửa và rút lại là sự kiện mới. Ví dụ bàn 71' của Sông Hồng FC (giả lập): hàng cũ **không** bị đổi.

| store_seq | logical_event | version | kind | giờ trận | status | supersedes | nguồn |
|---|---|---|---|---|---|---|---|
| 11 | `LEV_912_05` | 1 | `GOAL` | 71' | confirmed | — | FeedAlpha `a-77130`, FeedBeta, FeedGamma |
| 12 | `LEV_912_05` | 2 | `RETRACTION` | 71' (ghi nhận 74') | confirmed | hàng 11 | FeedAlpha `goal_cancelled` (lý do `OFFSIDE`) |

Hiệu lực = phiên bản lớn nhất của mỗi `logical_event_id` mà nguồn đủ thẩm quyền. Quan sát hạng thấp (FeedGamma còn giữ 2-1) **không** dựng lại bàn đã bị rút bởi nguồn hạng cao hơn. Hệ quả dây chuyền khi một bản sửa được công bố: tính lại `match_states` → vô hiệu cache → tính lại `standings` nếu kết quả đổi → thông báo đính chính cho **đúng những người đã nhận** thông báo cũ. Dây chuyền này là *`derived_fact_invalidation`*, *`read_model_cache_contract`*, *`supersedable_notification`* (chưa có trong core; ca `SPT-T-021`, `SPT-T-037`).

Bản sửa của con người đi workflow `correction-request` (xem §7.2), không ghi thẳng.

---

## 6. Phiên bản và truy vết

- `logical_event_id`: danh tính của sự cố thật; `version`: thứ tự các phiên bản; `supersedes`: trỏ về hàng liền trước.
- `state_version` của `match_states` tăng mỗi lần projection đổi; trace của câu trả lời ghi `state_version` để dựng lại điều đã công bố (*`as_of_query`*, `SPT-F-070`, ca `SPT-T-057`).
- Truy vấn "tại thời điểm T" dựng lại từ `match_events` theo `ingested_at ≤ T`, không dùng trạng thái hiện tại.

---

## 7. Máy trạng thái

Trạng thái, phép chuyển, quyền, bất biến, giao dịch thuộc **workflow engine** (P11, §2.2 của kiến trúc). Guard là **tên** chọn từ danh mục mã đã cài, không phải biểu thức trong YAML (P12). Hạn chót là **timer** quét định kỳ. Model chỉ **đề xuất**: ví dụ gợi ý lý do hoãn từ văn bản thông báo, hay gợi ý ứng viên ánh xạ; mọi phép chuyển do engine thực hiện sau khi guard qua.

### 7.1 `match-lifecycle` (bóng đá)

```text
SCHEDULED ─► LIVE_1H ─► HT ─► LIVE_2H ─► FT ─► FINAL
    │           │                │        │
    │           └─► SUSPENDED ◄──┘        └─► LIVE_ET1 ─► ET_HT ─► LIVE_ET2 ─► FT ─► PENALTIES ─► FT
    ├─► POSTPONED ─► SCHEDULED          SUSPENDED ─► (trở lại kỳ cũ) | ABANDONED ─► FINAL (xử thắng/huỷ kết quả)
    └─► CANCELLED
```

| Từ → Đến | Kích hoạt | Guard (tên) | Ai hoặc điều gì quyết |
|---|---|---|---|
| `SCHEDULED → LIVE_1H` | quan sát `KICK_OFF` | `provider_quorum`, `not_before_scheduled_window`, `match_not_locked` | quan sát provider + engine |
| `LIVE_1H → HT` | quan sát `PERIOD_END` | `provider_quorum`, `clock_at_least_regulation`, `forward_only` | quan sát + engine |
| `HT → LIVE_2H` | quan sát bắt đầu hiệp 2 | `provider_quorum`, `forward_only` | quan sát + engine |
| `LIVE_2H → FT` | quan sát hết giờ | `provider_quorum`, `clock_at_least_regulation`, `forward_only` | quan sát + engine |
| `FT → LIVE_ET1` | hoà ở giải loại trực tiếp | `competition_allows_extra_time` | luật của giải + engine |
| `LIVE_ET2 → FT`, `FT → PENALTIES` | hết hiệp phụ, vẫn hoà | `clock_at_least_regulation`, `level_after_extra_time` | quan sát + engine |
| `FT → FINAL` | hết cửa sổ muộn hoặc có biên bản | `late_window_elapsed` hoặc `official_report_received`, `no_open_reconciliation`, `no_pending_var` | timer + engine; người nếu còn xung đột |
| `SCHEDULED → POSTPONED` | thông báo chính thức | `official_notice_or_editor_approval` | người xác nhận (model có thể đề xuất lý do) |
| `POSTPONED → SCHEDULED` | lịch mới | `official_notice_or_editor_approval`, `kickoff_not_started` | biên tập viên |
| `LIVE_* → SUSPENDED` | thông báo hoặc quorum | `provider_quorum` hoặc `official_notice_or_editor_approval` | engine |
| `SUSPENDED → LIVE_*` | tiếp tục | `resume_same_period` | engine |
| `SUSPENDED → ABANDONED → FINAL` | thông báo chính thức | `official_notice_or_editor_approval` | liên đoàn quyết kết quả (`result_type` `AWARDED` hoặc `VOID`) |
| `SCHEDULED/POSTPONED → CANCELLED` | thông báo chính thức | `official_notice_or_editor_approval` | người xác nhận |

`FINAL` là "đã chốt", **không** phải bất biến: sau `FINAL` chỉ đổi được qua `correction-request`; trạng thái vòng đời vẫn `FINAL`, còn `state_version` tăng. Ca `SPT-T-044` đến `SPT-T-047`, `SPT-T-055`.

### 7.2 Các workflow phụ

| Workflow | Trạng thái | Guard (tên) | Người/điều quyết |
|---|---|---|---|
| `correction-request` | `DRAFT → SUBMITTED → REVIEWING → APPROVED → PUBLISHED`; nhánh `REJECTED`, `EXPIRED` | `has_reason_and_evidence`, `requester_has_editor_role`, `target_event_effective`, `approver_differs_from_requester`, `target_version_unchanged`, `own_entry_within_grace` | biên tập viên + người duyệt khác; hết hạn do timer |
| `provider-incident` | `DETECTED → ACKNOWLEDGED → MITIGATING → RESOLVED → CLOSED` | `lag_below_threshold_for_window` | luật kích hoạt mở; kỹ sư tích hợp nhận |
| `mapping-review` | `CANDIDATE → REVIEWING → APPROVED → ACTIVE`; nhánh `REJECTED` | `confidence_below_auto_accept`, `alias_not_taken`, `entity_not_merged` | người duyệt; **model/bộ so khớp chỉ đề xuất** |
| `match-report` | `DRAFTED → EDITOR_REVIEW → APPROVED → PUBLISHED`; nhánh `REJECTED` | `match_is_final`, `all_claims_grounded`, `no_open_reconciliation` | biên tập viên; model soạn nháp |

Lệnh có tác dụng phụ lớn (chạy lại projection cả mùa) tạo human task chờ duyệt thay vì chạy ngay (C-pipeline). Việc nhiều workflow cùng tác động một trận (bản sửa mở trong lúc đóng `FINAL`) được chặn bởi guard `no_open_reconciliation` và `target_version_unchanged`. Thiếu: chuyển người nhận khi quá hạn (*`human_task_escalation`*, ca `SPT-T-049`).

---

## 8. Projection idempotent

- Hàm thuần: `state = apply(events_hieu_luc_sap_theo_thu_tu_chuan)`; không phụ thuộc thứ tự đến hay số lần chạy.
- Ghi bằng `UPSERT ... WHERE state_version < excluded.state_version`; chạy lại cùng lô không đổi `state_version` (ca `SPT-T-020`, `SPT-T-064`).
- Mỗi trận một khoá tư vấn; worker mới lấy lại lô sau khi worker cũ chết phải bắt kịp đúng một lần (ca `SPT-T-041`, `SPT-T-064`).
- Công thức DERIVED FACT được ghi cùng kết quả: tỉ số = số bàn hiệu lực (phản lưới cộng cho đội đối phương); điểm = 3/1/0 theo kết quả trận `FINAL` + tiêu chí phụ của giải/mùa (cấu hình + mã). Hai ca `SPT-F-033` và `SPT-F-034` canh giữ.
- Bất biến kiểm định kỳ: tỉ số dẫn xuất = số bàn hiệu lực; chênh lệch với tỉ số provider chỉ **cảnh báo** (`SPT-F-067`).

---

## 9. Cache và vô hiệu hoá

MVP chưa có Redis; "cache" gồm: bảng `match_states` làm bản đọc, ETag/`Cache-Control` ở HTTP, bộ nhớ trong của từng instance.

| Chính sách | Nội dung |
|---|---|
| Khoá | `(tenant_id, match_id, state_version)`; không có `tenant_id` là lỗi (`SPT-F-013`) |
| TTL theo trạng thái | `LIVE_*`: rất ngắn; `HT`, `FT`: ngắn; `FINAL`: dài; lịch sử: dài; giá trị cụ thể là cấu hình của pack (chưa đo) |
| Vô hiệu | khi projection đổi `state_version`; nhiều instance **không** dựa riêng vào tín hiệu trong bộ nhớ mà kiểm phiên bản rẻ trước khi trả bản cache (`SPT-F-015`, ca `SPT-T-062`) |
| Lan truyền | sửa kết quả trận làm vô hiệu cả bản đọc trận lẫn `standings` (`SPT-F-016`) |
| Không cache theo ngữ nghĩa dữ kiện live | câu trả lời "tỉ số" không dùng cache ngữ nghĩa; bản cache cũ hơn `state_version` không được trả |
| Dữ liệu cũ phải nói ra | nguồn im lặng hay đọc lỗi thì trả bản gần nhất kèm nhãn `stale` và `as_of` (ca `SPT-T-063`) |

## 10. Mục tiêu độ trễ (SLO) — chưa đo

Mọi con số dưới đây là **mục tiêu thiết kế giả lập**, ghi để có thứ so khi đo. Lệnh tái lập: **chưa có** (cần `eais simulate` + bộ chạy ca). Chưa đo thì không được viết là đạt.

| Chặng | Mục tiêu |
|---|---|
| FeedAlpha: nhận → hiển thị ở API | p95 ≤ vài giây |
| FeedBeta (hỏi 10 giây một lần): → hiển thị | p95 ≤ 30 giây |
| FeedGamma (lô 60 giây): → hiển thị | p95 ≤ 90 giây |
| `GET` bản đọc trận | p95 ≤ 150 ms |
| Từ khi sự kiện rút lại được ghi tới khi mọi instance trả đúng | ≤ một lần đọc kế tiếp |
| Phát hiện provider im lặng | ≤ 30 giây (ngưỡng `provider_lag_seconds`, giả lập) |

## 11. Mười hai ca hỏng realtime

Đối chiếu từng ca hỏng với `SPT-F` và kịch bản ở [07 §3](07-failure-catalog.md#3-đối-chiếu-mười-hai-ca-hỏng-realtime-của-yêu-cầu-nền): provider trễ, provider không khả dụng, provider bất đồng, bàn thắng trùng, bàn bị huỷ/sửa, ánh xạ sai cầu thủ, trạng thái không nhất quán, đảo thứ tự, cache cũ, sai múi giờ, giải đổi tên, cầu thủ chuyển CLB.

## 12. MVP: PostgreSQL làm event store; chưa dùng Kafka hay Redis

- **MVP dùng PostgreSQL** cho `provider_events_raw`, `match_events`, `match_states`, hàng đợi (bảng + `SKIP LOCKED`) và bản đọc. **Không Kafka, không Redis** cho tới khi số đo đòi (ADR-0004).
- Điều kiện để xem xét thêm: chỉ khi **đo được** (trên bộ mô phỏng có seed) rằng đường đồng bộ hiện tại không đạt mục tiêu ở §10; trước đó mọi đề xuất là chưa có bằng chứng.
- Bảng liệt kê ở §2 theo tên của kiến trúc §7; các bảng ngoài đó (`standings`, `player_stats`, `subscriptions`) là **đề xuất** của tài liệu, migration thật do người điều phối viết.

## 13. Core, cấu hình hay domain module

| Phần | Phân loại (CONVENTIONS §4.5) |
|---|---|
| Ghi nhận quan sát nguyên văn, khoá K1, idempotency | `SHARED CORE` (`data/ingestion`, `platform/store`) |
| Ma trận thẩm quyền, hạng nguồn, dung sai, cửa sổ muộn, TTL | `CONFIGURABLE` (pack) |
| Tương quan K2/K3, thứ tự theo giờ trận, projection, mô hình kỳ/đồng hồ theo môn, tiêu chí xếp hạng | `DOMAIN MODULE` (`domain/sportsevents`) |
| Máy trạng thái, guard theo tên, human task, timer | `SHARED CORE` (`execution/workflows`) |
| Tên đội, giải, nguồn trong từ điển | `TENANT-SPECIFIC` (pack, dữ liệu) |

Core không chứa tên giải, đội, môn hay provider trong nhánh điều kiện (CONVENTIONS §8).

## 14. Quan sát được

Chỉ số: `provider_lag_seconds` theo provider; số quan sát bị bỏ/ghi nhận/cách ly; số sự kiện gộp; số sự kiện muộn; số rút lại; độ lệch giữa tỉ số dẫn xuất và tỉ số provider; tuổi hàng đợi; `state_version` mỗi câu trả lời. Mọi hàng `match_events` có `provider_refs`; mọi câu trả lời có trace và `state_version`.
