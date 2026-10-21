# Quy ước dự án EAIS

Tài liệu này là **một nguồn duy nhất** cho tên gọi, mã định danh, định dạng và luật viết. Mọi tài liệu và mã khác phải theo đây; chỗ nào lệch thì sửa chỗ đó, không sửa quy ước cho vừa.

## 0. Tên sản phẩm

| Thứ | Quy ước |
|---|---|
| Tên sản phẩm | **EAIS** — Enterprise AI Systems (D-RM-02) |
| Link git | tên repo, remote, đường dẫn module Go `github.com/luannt2002/enterprise-ai-control-layer`: giữ nguyên (D-RM-02) |
| Định danh chạy | biến môi trường tiền tố `EAIS_`; PostgreSQL: GUC `eais.tenant_id`, vai `eais_app`, database `eais`, `eais_giu_lai` |
| Định danh chạy khác | container PG dev `eais-pg`; database tạm của đột biến migration `eais_dot_bien`; mã hệ G7 `eais`, khoá `eais_unstable_cases` |
| Bí danh tương thích | tên dùng ngày 21/10 trước D-RM-02 vẫn được đọc khi tên hiện hành không đặt: `docs/ops/README.md` §4 |
| Roadmap hiện hành | `docs/ROADMAP-DAY-DU.md` (D-RM-01) |

## 1. Ngôn ngữ

| Thứ | Ngôn ngữ |
|---|---|
| Mã nguồn, tên biến/hàm/kiểu/package, tên bảng/cột SQL, khoá YAML | tiếng Anh |
| Comment trong mã, tài liệu, thông báo lỗi cho người vận hành | tiếng Việt có dấu |
| Thuật ngữ kỹ thuật | giữ tiếng Anh, kèm giải thích ngắn lần đầu xuất hiện, ví dụ *router (bộ định tuyến)* |
| Câu trả lời cho người dùng cuối | theo `language` của tenant (mặc định `vi`) |

## 2. Hai tenant tham chiếu

Cả hai đều **giả lập hoàn toàn**: công ty, người, máy móc, giải đấu, cầu thủ đều là hư cấu. Không dùng dữ liệu của doanh nghiệp thật nào.

| Mã | Slug | Tên | Domain | Vai trò kiến trúc |
|---|---|---|---|---|
| `CON` | `abc-infra` | ABC Infrastructure JSC | xây dựng cầu đường, hạ tầng | Reference Tenant #01: **khám phá** kiến trúc (dữ liệu người nhập bẩn → tài liệu → nguồn sự thật → workflow → số liệu chính xác) |
| `SPT` | `xyz-sports` | XYZ Sports Data JSC | dữ liệu thể thao nhiều nguồn | Reference Tenant #02: **tấn công** kiến trúc (realtime → thứ tự sự kiện → nhận dạng thực thể → nhiều provider → sửa muộn → cache → độ trễ) |

Mã `PLT` dùng cho thứ thuộc nền tảng/xuyên tenant.

## 3. Mã định danh

| Loại | Mẫu | Ví dụ |
|---|---|---|
| Yêu cầu (requirement) | `REQ-<AREA>-NNN` | `REQ-NRM-004` |
| Kịch bản nghiệp vụ (scenario) | `<T>-S-NNN` | `CON-S-037` |
| Ca hỏng (failure case) | `<T>-F-NNN` | `SPT-F-012` |
| Ca kiểm thử (test case) | `<T>-T-NNN` | `CON-T-104` |
| Ca router | `RTE-T-NNN` | `RTE-T-021` |
| Vector chuẩn hoá (normalization) | `NRM-V-NNN` | `NRM-V-042` |
| Quyết định kiến trúc | `ADR-NNNN` | `ADR-0005` |

`<T>` ∈ {`CON`, `SPT`, `PLT`}. Số chạy 3 chữ số, không tái dùng số đã xoá.

`<AREA>` của yêu cầu:

| AREA | Tầng | AREA | Tầng |
|---|---|---|---|
| `MND` | tư duy nền (mindset) | `TOL` | tools |
| `BIZ` | nghiệp vụ/giá trị | `SKL` | skills/workflow |
| `DAT` | nền dữ liệu, kiểm kê | `MCP` | MCP/tích hợp |
| `DIG` | đọc hiểu tài liệu | `AGT` | agent |
| `NRM` | chuẩn hoá | `MEM` | memory/state |
| `ENT` | nhận dạng thực thể | `VAL` | kiểm đầu ra/độ tin cậy |
| `SOT` | nguồn sự thật | `CLR` | hỏi lại/từ chối |
| `RTE` | router | `EVL` | đánh giá |
| `CTX` | context engine | `OBS` | quan sát/trace/chi phí |
| `RAG` | truy xuất tri thức | `SEC` | bảo mật/quản trị |
| `MDL` | model gateway | `TEN` | đa tenant |
| `SLA` | cam kết dịch vụ | `CV` | thị giác máy |
| `TRN` | huấn luyện/fine-tune | `RT` | realtime/luồng sự kiện |
| `WFL` | workflow/state machine, lệnh, human task | `PCK` | giới hạn và cấu trúc pack |

## 4. Bộ từ vựng cố định

### 4.1 Mười hai route (đường xử lý)

| Route | Khi nào | Có gọi model không |
|---|---|---|
| `DIRECT_LLM` | việc ngôn ngữ thuần, không cần dữ kiện doanh nghiệp (dịch, viết lại câu đã cho) | có |
| `SQL_QUERY` | số liệu chính xác nằm trong dữ liệu có cấu trúc | không bắt buộc (diễn đạt bằng mẫu) |
| `FULL_TEXT_SEARCH` | tìm mã/tên/chuỗi chính xác trong tài liệu | không bắt buộc |
| `RAG` | tri thức nằm trong tài liệu (quy định, SOP, hợp đồng) | có |
| `HYBRID_RETRIEVAL` | cần cả số liệu có cấu trúc lẫn tri thức tài liệu | có |
| `CALCULATION` | phép tính tất định trên số liệu đã có | không |
| `TOOL` | gọi một tool có hợp đồng (đọc hoặc ghi) | tuỳ tool |
| `WORKFLOW` | quy trình đã biết nhiều bước (skill) | tuỳ bước |
| `AGENT` | nhiều bước động, chỉ khi workflow cố định không giải được (có cổng) | có |
| `CLARIFICATION` | đầu vào mơ hồ/thiếu trường bắt buộc | không |
| `HUMAN_ESCALATION` | rủi ro cao, xung đột cần người quyết, vượt quyền | không |
| `REJECT` | ngoài phạm vi, trái chính sách, không có quyền | không |

### 4.2 Nhóm lỗi (failure category)

Tập hợp lớn hơn theo ADR-0008. Danh sách P10 của `CLAUDE.md` và phân loại tầng sai của G7 đều là tập con của danh sách này.

`DATA` · `PARSING` · `NORMALIZATION` · `ENTITY` · `INTENT` · `SEMANTICS` · `SOURCE_AUTHORITY` · `ROUTER` · `ROUTING` · `RETRIEVAL` · `CONTEXT` · `MODEL` · `CLAIM_BINDING` · `TOOL` · `SKILL` · `WORKFLOW` · `INTEGRATION` · `AUTH` · `PERMISSION` · `TENANCY` · `CACHE` · `STREAM` · `VALIDATION` · `OBSERVABILITY` · `INFRASTRUCTURE` · `PROVIDER` · `SCORER` · `UNKNOWN`

Ranh giới các cặp dễ nhầm:
- `ROUTER` là lỗi của thành phần; `ROUTING` là chọn sai đường cho một ca.
- `AUTH` là cơ chế xác thực / phân quyền hỏng; `PERMISSION` là câu trả lời lộ dữ liệu ngoài quyền người hỏi; `TENANCY` là vượt ranh giới tenant.
- `MODEL` là model sai khi ngữ cảnh đủ đúng; `PROVIDER` là API lỗi / hết hạn mức / quá giờ; `INFRASTRUCTURE` là hạ tầng của ta.
- `SCORER` là chính bộ chấm sai.
- `UNKNOWN` phải liệt kê riêng, không gộp.

### 4.3 Nhãn lỗi runtime (ai sai)

Mọi lỗi ra khỏi hệ thống mang đúng một nhãn `fault`:

| Nhãn | Nghĩa | HTTP gợi ý |
|---|---|---|
| `input` | bên gọi gửi sai | 400/422 |
| `policy` | bị chính sách chặn (quyền, phạm vi) | 403 |
| `data` | dữ liệu tenant thiếu/hỏng/xung đột không tự giải được | 409/422 |
| `code` | lỗi của chính EAIS | 500 |
| `third_party:<provider>:<kind>` | nhà cung cấp ngoài (model, embedding, nguồn dữ liệu) | 502/503/504 |
| `infra:<component>` | hạ tầng (postgres, parser service) | 503 |

### 4.4 Mức nghiêm trọng

| Mức | Nghĩa |
|---|---|
| `S1` | sai số liệu/hành động tới tay người dùng, rò dữ liệu giữa tenant, hành động ghi sai |
| `S2` | sai nhưng bị chặn/khó lan; mất chức năng chính |
| `S3` | trải nghiệm kém, chậm, tốn tiền vô ích |
| `S4` | thẩm mỹ, nhật ký thiếu |

### 4.5 Phân loại capability xuyên tenant

`SHARED CORE` · `CONFIGURABLE` · `DOMAIN MODULE` · `TENANT-SPECIFIC` · `NOT APPLICABLE`

Một abstraction chỉ chạy được cho một tenant thì **không** phải `SHARED CORE`.

## 5. Định dạng ca kiểm thử

Mọi ca kiểm thử (`<T>-T-NNN`, `RTE-T-NNN`) dùng đúng các khoá sau (YAML). Khoá nào không áp dụng thì ghi `null`, không xoá khoá.

```yaml
id: CON-T-001
tenant: abc-infra
domain: fuel                    # domain nghiệp vụ (xem tài liệu domain-map của tenant)
business_process: daily-site-report
scenario: CON-S-001             # kịch bản gốc
entry_type: QUERY               # QUERY | COMMAND | EVENT | WORKFLOW | RECONCILIATION | SECURITY
category: happy_path            # happy_path | dirty_data | ambiguity | conflict | permission | infrastructure | adversarial
input: "dầu 4k"
preconditions: []               # trạng thái dữ liệu cần có trước
sources: []                     # nguồn dữ liệu tham gia
expected_route: SQL_QUERY
expected_tools: []
expected_context: []            # bằng chứng phải có trong context package
expected_output_properties: []  # tính chất câu trả lời (không so chuỗi nguyên văn)
expected_state: null            # lệnh/sự kiện/workflow: {workflow: material-request, from: SUBMITTED, to: REVIEWING}
must_not_do: []
business_rules: []
permissions: {principal: site_supervisor}
failure_injection: null
expected_fallback: null
metrics: [normalization_accuracy]
severity: S1
acceptance_criteria: []
```

Tỉ lệ mục tiêu của bộ ca mỗi tenant: 20% happy path · 20% dirty data · 15% ambiguity · 15% conflict · 10% permission/security · 10% infrastructure failure · 10% adversarial/edge.

## 6. Luật về con số

- Con số nào trong tài liệu là **kết quả đo** thì phải kèm lệnh tái lập. Chưa đo thì ghi `chưa đo`.
- Con số trong ví dụ, dữ liệu mẫu, kịch bản là **giả lập** và phải đặt trong ngữ cảnh rõ là ví dụ.
- Không viết "chính xác 99%" mà không kèm: bộ dữ liệu nào, chỉ số nào, cỡ mẫu bao nhiêu.

## 7. Luật về dữ liệu (repo PUBLIC)

- Chỉ dữ liệu giả lập được vào git. Dữ liệu thật của bất kỳ tổ chức nào nằm trong thư mục bị `.gitignore` (`tenants/*/real/`).
- `docs/sources/` (ghi chú học, transcript, brief gốc) chỉ nằm ở máy; logic của chúng đã được chưng cất thành `docs/requirements/`.

## 8. Luật về core

- Mã trong `core/`, `data/`, `intelligence/`, `execution/`, `reliability/`, `governance/`, `observability/`, `platform/` **không được** chứa tên tenant, slug tenant hay tên domain (`construction`, `sports`, `abc-infra`, `xyz-sports`, `fuel`, `goal`…) trong nhánh điều kiện. Kiểm bằng test quét mã (`core_has_no_tenant_names_test.go`).
- Mọi khác biệt giữa tenant đi qua **pack** (dữ liệu cấu hình) hoặc **domain module** có hợp đồng.
- Một capability chỉ được vào core khi có ít nhất **một ca hỏng nghiệp vụ tái lập được** và **một test** chứng minh (ADR-0002).
