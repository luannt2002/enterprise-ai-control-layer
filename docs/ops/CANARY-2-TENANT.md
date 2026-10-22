# Canary 2 tenant và cờ approx của G1 (làn 2, 22/10/2026)

Hai phép đo mới của harness WIN. **Không đổi câu trả lời của hệ:** chế độ mới tắt mặc định, và G1 chỉ thêm một phép so.
Mọi số dưới đây đo trên dữ liệu giả lập (seed 42), chưa có người nghiệp vụ duyệt (D-RM-03). Nhãn: [ĐO] [CODE] [REPO] [SUY].

## 1. Canary 2 tenant (phản hồi audit K6)

**Vấn đề.** Harness cũ chỉ nạp tenant đang chạy (`winrun.go`: `Join(cfg.Data, cfg.Tenant)`), nên "0 canary" ở H1 và H3 đúng
theo cấu trúc: dữ liệu tenant B không bao giờ nằm trong kho. Đó chưa phải bằng chứng cách ly.

**Cách chạy** (đường bộ nhớ; `-canary` chưa hỗ trợ cùng `-pg`, chạy chung thì báo lỗi; đường PG chưa đo):

```bash
go run ./tenants/construction/win/gen/cmd/gen -world tenants/construction/win/world.yaml -out "$DATA" -profile functional -seed 42
go run ./tenants/construction/win/winreport -data "$DATA" -v -canary                                       # bộ 117
go run ./tenants/construction/win/winreport -data "$DATA" -v -canary -cases docs/g7/holdout/h1.yaml        # H1
```

**Nó làm gì.** Nạp mọi tenant khác của `world.yaml` (ở seed 42: `def-build`) vào CÙNG kho với tenant được chấm, mỗi lần đọc chỉ
thấy tenant của Principal (cùng luật với RLS). Hỏi các câu của tenant được chấm, rồi quét câu chữ, trích dẫn, xung đột và câu hỏi
lại của từng câu trả lời tìm **canary**: chữ chỉ có ở dữ liệu của tenant khác (token chữ-số của `evals.DeriveCanaries`, cộng mã tenant,
mã / tên / bí danh thực thể, mã tài liệu, khoá bản ghi). Chuỗi dùng chung như `EX-03` không là canary; câu trả lời nhắc lại canary
mà chính câu hỏi đã có thì không tính. Canary rỗng là lỗi (kiểm tra sẽ mù). Mã: `winrun/canary.go`.

**Đo** [ĐO, `winreport -canary`, đường bộ nhớ]:

| Bộ ca | Câu đã quét | Rò rỉ |
|---|---|---|
| 117 | 117 | 0 |
| H1 | 104 | 0 |
| H3 | 134 | 0 |

Kho chung: `abc-infra` 2.702 quan sát, 51 thực thể, 2 tài liệu; `def-build` 1 quan sát, 2 thực thể, 1 tài liệu. Chín canary:
`7731`, `9999`, `Cầu Z`, `Máy xúc EX-03 (B)`, `PRJ-Z`, `SOP-B`, `def-build`, `dự án Z`, `warehouse.csv#R-B01`.

**Đối chứng dương** (`winrun/canary_internal_test.go`, `go test -run TestCanary ./tenants/construction/win/winrun/`) [ĐO]:

| Kho cố ý rò ở cổng | Câu lộ canary |
|---|---|
| fact (bỏ lọc tenant) | 42/117 (ví dụ WIN-I-001: khoá `warehouse.csv#R-B01` trong trích dẫn) |
| tri thức (mất nhãn tenant của tài liệu) | 1/117 (WIN-I-072: `7731`, `SOP-B`) |
| danh bạ (bỏ lọc tenant) | **0/117** với bộ ca đóng băng; 1/118 khi thêm một ca mồi hỏi bằng "dự án Z" |

Đột biến trên mã thật: cho `sharedStore.Facts` bỏ lọc tenant thì test sạch ĐỎ (WIN-I-001 từ 3.850 L thành 13.849 L, tức 3.850 + 9.999).

**Giới hạn đã biết**
1. **Cổng danh bạ mù với bộ ca đóng băng:** không ca nào nhắc bí danh riêng của B, nên danh bạ rò vẫn 0/117. Muốn phủ cần ít nhất một ca
   mồi trong bộ ca (bộ ca đóng băng, ngoài phạm vi làn này).
2. **Canary số mù trước phép cộng:** số riêng 9.999 của B cộng vào 3.850 của A ra 13.849 L, không còn chữ `9999` trong câu trả lời. Ca này
   chỉ bị bắt nhờ khoá bản ghi `warehouse.csv#R-B01` trong trích dẫn.
3. Kho chung là bản bộ nhớ do harness dựng; việc lọc theo tenant ở đó là của harness. Mã thật được thử: S1 gắn tenant từ Principal
   (`ask.go:95`), `rag.NewMemoryIndex` loại tài liệu của tenant khác. RLS của PostgreSQL không đi qua chế độ này (đối chứng cấp thành phần:
   `TestRLSCanaryB`, chưa chạy ở lượt này).
4. Tenant B ở seed 42 chỉ có 1 quan sát và 1 tài liệu: độ phủ nhỏ. B được nạp bằng pack của tenant được chấm (cùng ngành), chưa thử tenant khác ngành.

## 2. Cờ approx ở G1 (phản hồi audit K1)

Manifest của bộ sinh có trường `approx` nhưng `checkG1` không so. Nay G1 so cờ hai chiều: manifest `approx: true` mà quan sát `approx = false`
(hoặc ngược lại) là một lệch; lệch trường chính (thực thể, chỉ số, nguồn, ngày, số) báo trước, mỗi bản ghi tối đa một vấn đề.

**Đo** [ĐO, seed 42]: G1 `2739/2739` thành `2736/2739`. Ba lệch, không bản nào khác:

| Khoá manifest | Bản ghi | Nguyên nhân |
|---|---|---|
| `messages.jsonl#R-001`, `#R-002` | "máy 03 sáng 2k", "chiều thêm 2k" | manifest ghi `approx: true` (`world.yaml:59-60`), bộ nạp ra `approx=false`. [SUY] câu không có dấu ước lượng, chỉ có hậu tố k. Chưa phân xử bên nào đúng |
| `reports/2026-10-19.pdf#1` | PDF "Machine 03 ≈ 4 m³" (R-003) | [CODE] `data/normalization/engine.go:198-210`: `matchApprox` chỉ nhận ký hiệu `~`; `≈` là ký hiệu khác nên không tới bảng bí danh |

Không sửa bộ chuẩn hoá hay bộ sinh để G1 xanh lại: việc của làn dữ liệu và chủ dự án. `winreport` nay thoát 1 trên seed 42 vì G1 đỏ.
`TestWinRatchet` nhận ba khoá trên từ `tenants/construction/win/g1-approx-known.txt` (ratchet chỉ đi xuống): khoá MỚI lệch cờ, hay khoá đã biết
lệch thêm trường khác, vẫn đỏ; khoá đã hết lệch thì test nhắc xoá khỏi tệp.
