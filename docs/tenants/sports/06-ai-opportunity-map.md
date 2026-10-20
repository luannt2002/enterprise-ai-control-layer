# 06 — Bản đồ cơ hội AI (tenant `xyz-sports`)

> Mọi giải, đội, cầu thủ, provider là **hư cấu**; mọi con số là **giả lập hoặc mục tiêu**, chưa đo. Chỉ dữ liệu thông tin thể thao: không có tính năng cá cược; hỏi kèo, tỉ lệ, dự đoán kết quả đi route `REJECT`.

Mục tiêu: chỉ ra **đâu thật sự cần xác suất** trong một nền tảng dữ liệu thể thao, và đâu giải tốt hơn bằng SQL, mã, luật, workflow. Phần lớn giá trị của tenant này nằm ở phần tất định (nạp, khử trùng, thứ tự, hoà giải, tính tỉ số); model chỉ ở rìa: diễn đạt, giải thích, đề xuất.

## 1. Nguyên tắc

1. **Không dùng model để bịa sự kiện live.** Tỉ số, người ghi bàn, phút, thẻ, đội hình đến từ event store; model không bao giờ là nguồn của một dữ kiện hay một con số (P3, P4).
2. **Model chỉ đề xuất (P11).** Chuyển trạng thái trận, duyệt bản sửa, chấp nhận ánh xạ, gộp cầu thủ do workflow engine và con người quyết.
3. **Xác suất phải bị bọc:** đầu vào là gói bằng chứng có mã; đầu ra là câu kèm `claims` gắn mã bằng chứng; validator kiểm số, tên, phút; không qua thì dùng **mẫu câu tất định** lùi.
4. **Nhãn chưa chắc phải hiện ra:** dữ liệu `provisional`, `disputed`, `stale` được nói ra, không giấu bằng giọng văn tự tin.
5. **Không tư vấn cá cược, không dự đoán kết quả** dưới bất kỳ dạng nào.

## 2. Bảng ranh giới tất định / xác suất

`Có` = lựa chọn của tác vụ này; `Lai` = xác suất bọc trong kiểm tất định. Route theo CONVENTIONS §4.1; tác vụ không đi router hỏi ghi pipeline.

| # | Tác vụ | Tất định? | Xác suất? | Lai? | Vì sao | Hệ quả khi sai | Route / pipeline |
|---|---|---|---|---|---|---|---|
| 1 | Tỉ số hiện tại | Có | Không | Không | DERIVED FACT từ sự kiện hiệu lực | tỉ số sai tới người dùng | `SQL_QUERY` |
| 2 | Ai ghi bàn | Có | Không | Không | dữ liệu có cấu trúc, trường có thẩm quyền | người ghi bàn sai | `SQL_QUERY` |
| 3 | Diễn biến theo thứ tự trận | Có | Không | Không | thứ tự theo giờ trận là luật | dòng thời gian sai | `SQL_QUERY` |
| 4 | Trạng thái trận (nghỉ giữa hiệp, hết giờ) | Có | Không | Không | máy trạng thái | báo kết thúc sớm hay muộn | `SQL_QUERY` |
| 5 | Phút và bù giờ hiển thị (45+2') | Có | Không | Không | chuẩn hoá theo kỳ | sai hiệp, sai thứ tự | `SQL_QUERY` |
| 6 | Đội hình xuất phát | Có | Không | Không | dữ liệu theo trường + quyền gói | lộ trường premium; đội hình sai | `SQL_QUERY` |
| 7 | Thống kê trận (kiểm soát bóng, cú sút) | Có | Không | Không | một nguồn chỉ định cho mỗi chỉ số | số vô lý | `SQL_QUERY` |
| 8 | Bảng xếp hạng | Có | Không | Không | công thức điểm + tiêu chí phụ | thứ hạng sai | `SQL_QUERY` |
| 9 | Lịch thi đấu và giờ khai cuộc | Có | Không | Không | UTC + múi giờ hiển thị | giờ lệch nhiều giờ | `SQL_QUERY` |
| 10 | Đối đầu (H2H) | Có | Không | Không | truy vấn lịch sử | thống kê sai | `SQL_QUERY` |
| 11 | Hạng mùa trước, giải đổi tên | Có | Không | Không | truy vấn theo hiệu lực | mất lịch sử | `SQL_QUERY` |
| 12 | Phong độ N trận (số liệu) | Có | Không | Không | phép cộng có ghi công thức | số sai | `CALCULATION` |
| 13 | Hiệu số bàn thắng, điểm | Có | Không | Không | phép tính | số sai | `CALCULATION` |
| 14 | Đổi múi giờ hiển thị | Có | Không | Không | quy đổi | lệch ngày | `CALCULATION` |
| 15 | Thứ hạng quần vợt theo tuần | Có | Không | Không | dữ liệu tham chiếu có `as_of` | trả bảng sai tuần | `SQL_QUERY` |
| 16 | Trạng thái set/game/điểm, hiệp bóng chày | Có | Không | Không | mô hình theo môn (mã) | trạng thái sai | `SQL_QUERY` |
| 17 | Kiểm quyền theo gói và tenant | Có | Không | Không | ABAC ngoài model | rò quyền, rò tenant | `REJECT` |
| 18 | Chặn hỏi kèo, tỉ lệ, dự đoán | Có | Không | Không | chính sách là luật, không phải ý kiến | vi phạm chính sách | `REJECT` |
| 19 | Khử trùng và tương quan sự kiện | Có | Không | Không | khoá + dung sai cấu hình | bàn nhân đôi hay mất | pipeline `E0–E6` |
| 20 | Chuẩn hoá phút, múi giờ, mã trạng thái | Có | Không | Không | hàm thuần theo phiên bản adapter | sai hiệp, lệch giờ | pipeline `E0–E6` |
| 21 | Nhận dạng thực thể theo bí danh chính xác | Có | Không | Không | bảng ánh xạ có nguồn, hiệu lực | gán sai đội | pipeline `E0–E6` |
| 22 | Nhận dạng thực thể mờ dưới ngưỡng | Không | Có | **Lai** | cần so khớp gần đúng; quyết định không thuộc model | gán sai người | workflow `mapping-review` |
| 23 | Chuyển trạng thái vòng đời trận | Có | Không | Không | engine + guard theo tên | kết quả chốt sai | `engine` |
| 24 | Duyệt và công bố bản sửa | Có | Không | Không | quy trình bốn mắt | sửa không kiểm soát | `engine` |
| 25 | Phát hiện provider im lặng | Có | Không | Không | so sánh ngưỡng khai báo | dữ liệu cũ không cờ | trigger `E3` |
| 26 | Tìm bài viết theo tên | Có | Không | Không | tìm chuỗi + chuẩn hoá không dấu | bỏ sót bài | `FULL_TEXT_SEARCH` |
| 27 | Hỏi lại khi thực thể mơ hồ | Có | Không | Không | luật ngưỡng | đoán nhầm | `CLARIFICATION` |
| 28 | Chuyển người khi xung đột không giải được | Có | Không | Không | quorum + hết hạn | công bố điều chưa chắc | `HUMAN_ESCALATION` |
| 29 | Tóm tắt trận từ sự kiện | Không | Có | **Lai** | diễn đạt là xác suất; dữ kiện là tất định | bịa người ghi bàn, phút | `WORKFLOW` (skill `match_summary`) |
| 30 | Bản tin sau trận | Không | Có | **Lai** | model soạn nháp; người duyệt | công bố bản sai | workflow `match-report` |
| 31 | Giải thích vì sao bàn bị huỷ | Không | Có | **Lai** | cần sự kiện VAR + điều luật | nguyên nhân bịa | `HYBRID_RETRIEVAL` |
| 32 | Giải thích phong độ (số + lời) | Không | Có | **Lai** | số tính tất định, lời do model | khẳng định không có số | `WORKFLOW` |
| 33 | "Vì sao đội X chơi tệ?" | Không | Có | **Lai** | chỉ giải thích được trên số liệu có trong gói; nguyên nhân ngoài dữ liệu thì nói không biết | nhân quả bịa | `WORKFLOW` |
| 34 | Tìm trận lịch sử tương tự | Không | Có | **Lai** | lọc theo cấu trúc, xếp theo độ giống | trận "tương tự" sai | `HYBRID_RETRIEVAL` |
| 35 | Hỏi luật, thể thức | Không | Có | Không | tri thức ở tài liệu có phiên bản; cần trích dẫn | luật sai mùa | `RAG` |
| 36 | Dịch bản tin | Không | Có | **Lai** | dịch là ngôn ngữ; số, tên phải bảo toàn | đổi tỉ số khi dịch | `DIRECT_LLM` |
| 37 | Viết lại câu cho gọn | Không | Có | **Lai** | ngôn ngữ thuần; kiểm số không thêm | thêm số lạ | `DIRECT_LLM` |
| 38 | Gợi ý lý do hoãn từ thông báo văn bản | Không | Có | **Lai** | trích từ văn bản là đề xuất; người xác nhận | ghi lý do sai | human task trong `match-lifecycle` |
| 39 | Điều tra nguyên nhân gốc độ trễ tăng | Không | Có | **Lai** | nhiều bước động (cổng đóng) | kết luận sai | `AGENT` (đóng) → lùi về `WORKFLOW`/`HUMAN_ESCALATION` |
| 40 | Phân loại ý định bằng model khi luật trượt | Không | Có | **Lai** | chỉ khi đo thấy luật không đủ | route sai | (cổng đóng) |
| 41 | Dự đoán kết quả trận | — | — | — | **không làm**; liền kề cá cược | vi phạm chính sách | `REJECT` |

## 3. Tool và skill của tenant

| Tên | Loại | Đọc/ghi | Quyền tối thiểu | Ghi chú |
|---|---|---|---|---|
| `get_match_state`, `get_match_events`, `get_match_as_of` | tool | đọc | theo gói | `as_of` là *`as_of_query`* (thiếu ở core) |
| `get_fixtures`, `get_standings`, `get_h2h`, `get_team_form` | tool | đọc | theo gói | `get_team_form` là phép tính |
| `search_entities`, `search_articles` | tool | đọc | công khai | chuẩn hoá không dấu |
| `get_provider_health`, `get_provider_discrepancies` | tool | đọc | `data_ops`, `integration_engineer` | độ trễ theo provider |
| `append_correction_event`, `lock_match`, `reschedule_match` | tool | ghi (lệnh) | `editor` | qua `correction-request`, có audit |
| `merge_players`, `add_entity_alias`, `rebuild_projection` | tool | ghi (lệnh) | `data_ops` | lệnh nặng chờ duyệt |
| `set_provider_priority` | tool | ghi (lệnh) | `integration_engineer` | bản ghi đè có phiên bản |
| `subscribe_team` | tool | ghi (lệnh) | `fan_registered` | idempotent |
| `match_summary`, `post_match_report`, `provider_reconciliation` | skill | nhiều bước | theo bước | model chỉ ở bước diễn đạt |

## 4. Cổng "nguyên lý đầu tiên" cho các ứng viên dùng model

| Câu hỏi | Tóm tắt trận | Giải thích bàn bị huỷ | Đề xuất ánh xạ bí danh |
|---|---|---|---|
| Vấn đề thật là gì? | diễn đạt chuỗi sự kiện thành văn | nối sự kiện VAR với điều luật | gợi ý đội/cầu thủ khớp tên |
| Bỏ AI thì giải được bao nhiêu? | danh sách bàn thắng và tỉ số bằng mẫu: **gần hết dữ kiện**, thiếu mượt mà | sự kiện + lý do mã: đủ dữ kiện, thiếu giải thích | so khớp mờ bằng luật: đủ cho phần lớn biến thể (chưa đo) |
| Dữ kiện nằm đâu? | event store | event store + điều lệ | `entity_aliases` |
| Chính xác hay xấp xỉ? | số, tên, phút: **chính xác** | sự kiện chính xác; lời diễn đạt xấp xỉ | xấp xỉ |
| Có mơ hồ? | không (dữ kiện), có (giọng văn) | thấp | cao |
| Cần truy xuất? | không | có (điều luật) | không |
| Hệ quả sai | bịa người ghi bàn | nhân quả bịa | gán sai người |
| Kiểm | validator số/tên/phút + mẫu lùi | validator + trích dẫn | người duyệt; model không tự ghi |
| Số đo | `groundedness`, `validator_catch_rate`, `false_block_rate` | `citation_correctness` | `entity_resolution_accuracy` |
| Kết luận | **dùng model có bọc**, lùi tất định | dùng `HYBRID_RETRIEVAL` | **chỉ đề xuất**, quyết định bởi người |

## 5. Validator của tenant

- Mọi tên, số, phút trong câu trả lời phải có trong gói bằng chứng (sau chuẩn hoá `45+2` = `45+2'`); không thì chặn và dùng mẫu lùi.
- Sự kiện đã bị rút (`retracted`) không được nêu như hiệu lực; nếu nhắc phải nói rõ đã huỷ.
- Khẳng định nhân quả (chiến thuật, chấn thương) phải gắn bằng chứng; thiếu thì không khẳng định.
- Bản dịch/viết lại: tập số và tên đầu ra ⊆ đầu vào.
- Dữ liệu `provisional`, `disputed`, `stale` phải mang nhãn tương ứng trong câu trả lời.
- Đo bằng `validator_catch_rate` (bắt được lỗi) và `false_block_rate` (chặn nhầm câu đúng); chưa đo.

## 6. Capability có cổng (đang đóng)

| Capability | Cổng mở khi (kiến trúc §9) | Trạng thái |
|---|---|---|
| Phân loại ý định bằng model | luật trượt một ngưỡng đo được trên bộ ca router | đóng |
| Trích xuất bằng model | luật bỏ sót trên ca bẩn và có validator chặn | đóng |
| Rerank | Recall@20 > Recall@5 rõ rệt | đóng |
| Agent | có ca mà workflow cố định không giải được, đo được | đóng (kịch bản 039 là ứng viên) |
| MCP | có bên tiêu thụ thật | đóng |
| Thị giác máy, fine-tune | có dữ liệu thật và ca hỏng mà context/tool không chữa được | đóng; `media_assets` chưa dùng |

Hàng đợi/cache ngoài PostgreSQL cũng đóng cho tới khi số đo đòi (ADR-0004).
