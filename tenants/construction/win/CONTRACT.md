# Hợp đồng bộ dữ liệu WIN (bộ sinh ↔ bộ nạp ↔ cổng G1)

Bộ sinh (`tenants/construction/win/gen`, mã riêng tenant) đọc `tenants/construction/win/world.yaml` rồi render ra thư mục dữ liệu theo đúng bố cục dưới đây.

Bộ nạp (`data/ingestion`) **không import bộ sinh**. Nó đọc các tệp này qua ánh xạ nguồn trong pack (`tenants/construction/pack/sources.yaml`).

Cổng G1 so `manifest.json` của bộ sinh với đầu ra của bộ nạp (quan sát ∪ cách ly). Hai bên hiểu lệch hợp đồng thì G1 đỏ, và đó là việc của nó.

Mọi tệp mã hoá UTF-8. Ngày ghi `YYYY-MM-DD`; giờ ghi RFC3339 có múi giờ `+07:00`.

## 1. Thư mục

```text
<out>/
  manifest.json                 # đáp án cho G1 (mục 3)
  <tenant>/                     # một thư mục mỗi tenant: abc-infra/, def-build/
    entities.json               # dữ liệu gốc: dự án + thiết bị + bí danh + mã theo nguồn
    warehouse.csv               # phiếu xuất kho (nguồn warehouse_slip)
    warehouse.xlsx              # phiếu xuất kho dạng Excel, sheet "PhieuXuat" (bản ghi có format: xlsx)
    excel_summary.xlsx          # bảng tổng hợp, sheet "TongHop" (nguồn excel_summary)
    erp.json                    # chứng từ ERP (nguồn erp)
    sensor.csv                  # cảm biến nhiên liệu (nguồn fuel_sensor)
    invoices.csv                # hoá đơn (nguồn invoice)
    messages.jsonl              # tin nhắn hiện trường (nguồn chat_message)
    reports/<YYYY-MM-DD>.pdf    # báo cáo ngày (nguồn site_report_form)
    docs/<doc_id>.md            # tài liệu SOP cho RAG
    docs/acl.json               # {"<doc_id>": ["<vai>", ...]}
  variants/                     # cho test metamorphic G1 (mục 4)
```

Tệp nào không có bản ghi thì **vẫn tạo**, chỉ chứa header (hoặc `[]` với JSON).

## 2. Cột và trường

| Tệp | Cột / trường | Ghi chú |
|---|---|---|
| `warehouse.csv` · `warehouse.xlsx` | `record_id,slip_no,equipment_code,issue_date,quantity,unit` | `equipment_code` = `source_ids.warehouse_slip`. `quantity` ghi **như nguồn ghi** (`3850`, `1,2`; ô trống nếu bẩn). `unit` ∈ `L`, `m3`, … (bẩn: `kg`) |
| `excel_summary.xlsx` | `record_id,equipment_code,date,quantity_l` | |
| `erp.json` | mảng `{"record_id","doc_no","asset_id","posting_date","material","quantity","uom"}` | `asset_id` = `source_ids.erp`; `material` = `"DIESEL"`; `uom` = `"L"`; `quantity` là chuỗi |
| `sensor.csv` | `record_id,device_id,date,consumed_l` | `device_id` = `source_ids.fuel_sensor` |
| `invoices.csv` | `record_id,invoice_no,equipment_code,date,quantity_l` | |
| `messages.jsonl` | mỗi dòng `{"record_id","sent_at","sender","text"}` | `text` = trường `raw` của bản ghi; ngày quan sát = ngày của `sent_at` theo `+07:00` |
| `reports/<ngày>.pdf` | dòng đầu `Báo cáo ngày DD/MM/YYYY`; mỗi bản ghi một dòng `<n>. <raw>`, n đếm từ 1 | sinh bằng `go-pdf/fpdf` + DejaVuSans; đọc bằng `pdftotext` |
| `entities.json` | `{"projects":[{"id","name","aliases"}],"equipment":[{"id","type","name","project","aliases","source_ids"}]}` | chép từ world.yaml + thiết bị nền |
| `docs/<id>.md` | dòng đầu `# <title>`, sau đó là body | `acl.json` lấy từ `documents[].acl` |

`record_id` của bản ghi trong world.yaml là id world (`R-004`); bản ghi nền dùng `F-000001`…

## 3. `manifest.json` (đáp án G1)

```json
{"version": 1, "reference_date": "2026-10-20", "profile": "functional", "seed": 42,
 "records": [
   {"tenant": "abc-infra", "key": "warehouse.csv#R-004", "world_id": "R-004", "expect": "observation",
    "entity": "EX-03", "semantic": "issued_fuel", "source": "warehouse_slip", "on": "2026-10-19",
    "value_l": "3850", "approx": false},
   {"tenant": "abc-infra", "key": "warehouse.csv#R-021", "world_id": "R-021", "expect": "quarantine", "reason": "missing_value"}
 ]}
```

**Khoá `key`** = đường dẫn tương đối trong thư mục tenant + `#`, theo sau là:
- `record_id`, với CSV / XLSX / JSON / JSONL;
- `n` (số thứ tự dòng), với PDF: `reports/2026-10-19.pdf#1`.

Mọi bản ghi được render đều **phải** có trong manifest, kể cả bản ghi nền và bản ghi bẩn. Ngược lại, manifest không được có khoá nào không được render.

**Lý do cách ly** (trường `reason`):

| `reason` | Nghĩa |
|---|---|
| `missing_value` | thiếu lượng |
| `unknown_entity` | mã thiết bị không có trong `entities.json` |
| `out_of_range` | ngoài min / max của chỉ số |
| `unit_dimension` | đơn vị khác chiều đơn vị của chỉ số |
| `unparseable` | không đọc được lượng / ngày |

## 4. Biến thể metamorphic (`variants/`)

`variants/manifest.json`: mảng `{"world_id", "tenant", "source", "files": [...], "expect": {"entity", "semantic", "source", "on", "value_l"}}`.

Mỗi biến thể là **cùng một bản ghi của cùng một nguồn**, ở định dạng khác:
- phiếu kho R-004: dạng `warehouse.csv`, `warehouse.xlsx`, `warehouse.json` (mảng object cùng cột với CSV) và `warehouse.csv` gõ tay (lượng có dấu chấm nghìn `3.850`, đơn vị viết chữ `lít`) — 4 biểu diễn;
- tin nhắn R-001: ≥ 4 cách viết, ví dụ `máy 03 sáng 2k`, `may 03 sang 2k`, `máy xúc 03 sáng lấy 2.000 lít`, `MX-03 sáng 2 khối`.

Mỗi biến thể nằm trong thư mục riêng `variants/<world_id>/<n>/` theo đúng bố cục mục 1 (kèm `entities.json`). Nạp từng thư mục phải ra **cùng một** quan sát chuẩn theo phép chiếu (entity, semantic, value_l, on, source).

## 5. Dữ liệu nền (hồ sơ)

| Hồ sơ | Nội dung | Ước lượng |
|---|---|---|
| `functional` | 40 thiết bị nền `FX-001…FX-040` thuộc các **dự án nền riêng** `PRJ-F01…PRJ-F04` (không phải PRJ-A / PRJ-B, để tổng theo dự án của world không bị lẫn) × 30 ngày (2026-09-21…2026-10-20) | ≈ 3.000 bản ghi |
| `smoke` | 400 thiết bị × 365 ngày | ≈ 360.000 bản ghi (đo 20/10: 359.946 mục manifest với seed 42) |

- Mỗi thiết bị nền mỗi ngày có: phiếu kho (CSV), ERP (khoảng 50% ngày, lệch 0–20 L), cảm biến.
- Lỗi bơm có chủ ý theo seed (**tham số giả lập, không phải tỉ lệ doanh nghiệp thật**): 2% thiếu lượng, 1% mã thiết bị lạ, 0,5% đơn vị `kg`.
- Bản ghi nền **không** chạm vào thiết bị hay ngày của world.yaml.
- Seed cố định ⇒ chạy lại ra **cùng byte**. Riêng PDF cần đặt `CreationDate` cố định.
