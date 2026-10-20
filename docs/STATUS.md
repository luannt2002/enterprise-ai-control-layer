# STATUS — trạng thái EAIS

> Cập nhật: 20/10/2026 10:35. Phiên tự chạy **dừng lúc 10:33** theo lệnh chủ dự án; mã và tài liệu đã đồng bộ lên git (commit đầu tiên). Chưa phát hành phiên bản nào.
> Báo cáo ngày: `docs/bao-cao/2026-10-20-win-mvp.md`. Chạy lại / làm tiếp ở máy khác: `docs/ops/README.md`.
> Mọi số ở đây lấy từ lệnh chạy ngày 20/10/2026, trừ chỗ ghi **CHƯA ĐO**.

## Tiến độ WIN MVP: 6/7 cổng có bằng chứng đạt — CHƯA phải WIN

Đích hiện tại là **WIN MVP** (`docs/04-core-mvp.md`), không phải cả roadmap (`docs/05-master-roadmap.md`).

**WIN = 7/7 cổng + scale smoke** (04 §6). Cách tính tiến độ: mỗi cổng 1/7, chỉ tính cổng đạt trọn ⇒ **6/7 ≈ 86%**.

- G1–G6 đạt theo đúng tiêu chí của 04 §6.
- Scale smoke đạt trên cả đường bộ nhớ lẫn đường PostgreSQL.
- **G7 VALUE: CHẶN.** Cần model thật, tức khoá API và chủ dự án đồng ý chi phí. 04 §6 cấm thay bằng model giả.

| Cổng | Kết quả | Bằng chứng |
|---|---|---|
| G1 DATA | ĐẠT | - G1: 2.739/2.739 (functional), 359.945/359.945 (smoke).<br>- Metamorphic: R-004 × 4 biểu diễn, R-001 × 4 cách viết, 0 lệch.<br>- Bộ kiểm có test âm.<br>- `representation` 16/16. |
| G2 IDENTITY | ĐẠT | - `entity` 13/13.<br>- `ambiguity` 13/13, ứng viên đúng tập. |
| G3 TRUTH | ĐẠT | - `semantic` 14/14, `source_conflict` 12/12.<br>- Ca biên dung sai sinh từ pack cho MỌI cặp nguồn: 4 cặp × "= dung sai" / "dung sai + 1", gồm cặp ngang hạng. |
| G4 DECISION | ĐẠT | - `routing` 17/17.<br>- Runner chấm `model_calls` bằng gateway giả đếm lời gọi.<br>- Đột biến bắt buộc "SQL gọi gateway" (BA) và "RAG bỏ gateway" (BB) đều bị bắt. |
| G5 CONTROL | ĐẠT | - `permission` 18/18.<br>- Cổng AST: không kiểu `*Request` nào mang tenant.<br>- PG: RLS FORCE + GUC đọc lại; canary B có test cho phép / từ chối, đối chứng dương, ghi chéo tenant bị WITH CHECK chặn.<br>- Ràng buộc claim 6 chiều. Bảng "sai đúng một chiều" 6 × 3 bị chặn 100%; 0 chặn oan trên 13 ca mẫu + 117 ca thật.<br>- Lượng dầu trong tài liệu bị chặn bởi hai lớp, một lớp độc lập với bộ chuẩn hoá.<br>- Chèn lệnh trong SOP không đổi được route / quyền / số. |
| G6 PROOF | ĐẠT | - 117 ca, 100% do phiên không viết mã soạn; mọi lớp ≥ 12 ca.<br>- `cases-independent.yaml` sha256 `ffdd704b535d2195…`, khớp băm đóng băng 20/10 09:15.<br>- Bản ghi quyết định + phát lại: 117/117 không lệch.<br>- Thuộc tính: tổng ≥ 0; dẫn xuất = a − b lấy từ bằng chứng; A không thấy B.<br>- Mỗi cổng có đột biến bị bắt. |
| G7 VALUE | **CHẶN** | - Bộ khung xong: baseline RAG (`evals/baseline`), luật D5 + bộ đếm độc lập (`evals/g7.go`), lệnh `g7run`, client model thật `models.Messages` cấu hình qua biến môi trường.<br>- Chạy ống nối bằng model giả: D5 ĐẠT, nhưng KHÔNG phải bằng chứng (04 §6).<br>- Thiếu: khoá model + đồng ý chi phí. Có rồi thì chạy `EAIS_MODEL_PROVIDER=anthropic EAIS_MODEL=<model> EAIS_MODEL_API_KEY=<khoá> go run ./tenants/construction/win/g7run -data <dir>`. |
| Scale smoke | ĐẠT | - `scripts/scale-smoke.sh`: 117/117 ở cả functional và smoke, câu trả lời giống từng byte; smoke 3,67 s, peak RSS 562,8 MiB (đo 10:35).<br>- Đường PG (`winreport -pg`): 117/117, từng ca và từng câu trả lời giống đường bộ nhớ; smoke 6,08 s, 653,7 MiB, kể cả nạp ~355 nghìn fact vào PG. |

Mốc "≈ 20%" của tối 19/10 bỏ, vì nó ước theo cảm nhận. Từ nay chỉ đếm cổng có lệnh chứng minh.

## Bằng chứng đã chạy (20/10/2026)

```bash
docker compose -f deploy/docker-compose.yml up -d --wait          # PG 17.11, ghim digest, chỉ 127.0.0.1:25432
export EAIS_PG_ADMIN_DSN='postgres://postgres@127.0.0.1:25432/eais?sslmode=disable'
export EAIS_PG_DSN='postgres://eais_app@127.0.0.1:25432/eais?sslmode=disable'
go vet ./...                       # sạch
gofmt -l .                         # rỗng
go test ./... -count=1             # 224 PASS / 0 FAIL / 0 SKIP (có PG, đo 10:35). Không có biến PG: 5 test PG SKIP
scripts/scale-smoke.sh             # SCALE SMOKE: ĐẠT
go run ./tenants/construction/win/gen/cmd/gen -world tenants/construction/win/world.yaml -out /tmp/win -profile functional -seed 42
go run ./tenants/construction/win/winreport -data /tmp/win -v       # đường bộ nhớ
go run ./tenants/construction/win/winreport -data /tmp/win -v -pg   # đường PostgreSQL (RLS)
```

`TestWinRatchet` biến bộ ca WIN thành cổng hồi quy của `go test`. Nó đòi đủ bốn điều:
- G1 khớp manifest;
- metamorphic ≥ 4 biểu diễn mỗi bản ghi;
- mọi ca trong `passing.txt` (117/117) vẫn đạt;
- phát lại 0 lệch.

## Đã có

| Vùng | Nội dung | Bằng chứng |
|---|---|---|
| `core/contracts` | kiểu dùng chung; `Outcome` có kiểu; claim đủ 6 chiều; bằng chứng mang chủ thể / chỉ số / kỳ; tenant trong ngữ cảnh (`WithTenant`) | test fault + test đầu–cuối |
| `core/pipeline` | lối vào QUERY S0–S10 cho 7 route WIN. S1 gắn tenant của Principal vào ngữ cảnh. S9 có 4 lớp kiểm số liệu, 4 lớp kiểm tri thức. Có bản ghi quyết định + `Replay` | `tenants/construction` (đầu–cuối) |
| `data/normalization` | số kiểu Việt (cả số âm); đơn vị; thuật ngữ; viết tắt; ngày tương đối; luật dấu; `ToneKey` ("hoá" = "hóa", "bí" ≠ "bị") | vector NRM-V-001…049 |
| `data/entityresolution` | bí danh dài nhất, mã theo nguồn, mơ hồ ⇒ ứng viên, vị trí byte của cụm | `resolver_test.go` |
| `data/sourceoftruth` | thẩm quyền trong từng chỉ số, xung đột / ngang hạng, dẫn xuất, so hai chỉ số (`Compare`); thứ tự bằng chứng tất định | test + thuộc tính |
| `data/ingestion` | nạp CSV / XLSX / JSON / JSONL / PDF văn bản theo `sources.yaml`; cách ly có lý do | `ingest_test.go`, G1 |
| `intelligence/intent`, `router` | ý định theo luật, khớp dài nhất, ý định yếu, D7, mệnh đề trọng tâm, so sánh hai nghĩa, ghép RAG / HYBRID; quyền đóng mặc định | test + đầu–cuối |
| `intelligence/rag`, `models` | chỉ mục tri thức trong bộ nhớ (tenant + ACL theo vai); gateway model theo cấu hình; model giả tất định có đếm lời gọi | `rag_test.go`, `models_test.go` |
| `reliability/validation` | `Fidelity` (lớp lọc đầu), `BindClaims` + `CountNumbers` (6 chiều, bộ đếm độc lập), `Bound`, `ForbidDimensions` + `ForbidUnitWords` | bảng sai đúng một chiều |
| `platform/pgstore`, `db/migrations` | PG17: RLS FORCE theo GUC `eais.tenant_id`, vai `eais_app`, Load thay trọn dữ liệu tenant, ACL tài liệu lọc trong SQL | test PG + `TestWinTrenPG` |
| `observability/traces` | đo thời gian chặng; `Decision` + `Diff` | `decision_test.go` |
| `evals` | hợp đồng ca WIN, runner (chấm `model_calls`), đếm cổng G6 | `runner_test.go`, `win_test.go` |
| `tenants/construction` | pack thật; `win/`: world đóng băng, bộ sinh có seed (functional ~2,7 nghìn / smoke ~360 nghìn bản ghi), 117 ca độc lập, `winrun`, `winreport` | ratchet |
| `architecture_test.go` | mã dùng chung không mang tên tenant; kiểu `*Request` không mang tenant | đột biến G, TA, TB |
| `deploy/`, `scripts/` | compose PG; `mutate.sh`; `scale-smoke.sh` | — |

## Đột biến (20/10/2026)

Mỗi đột biến sửa sai một chỗ, chạy test, khôi phục, kiểm sha256 (`scripts/mutate.sh`). Danh mục chạy lại được: `scripts/dot-bien/` (69 đột biến, mọi chuỗi gốc khớp mã ngày 20/10).

| Nhóm | Số | Kết quả |
|---|---|---|
| quyền đóng mặc định (I, J) | 2 | bị bắt |
| nạp + resolver (K–O) | 5 | bị bắt; O lần đầu SỐNG SÓT → thêm ca |
| outcome / xác nhận (P–U) | 6 | bị bắt |
| ratchet (V–AC) | 8 | bị bắt; Y, AA lần đầu SỐNG SÓT → thêm ca |
| RAG / HYBRID (AD–AZ, BA, BB) | 25 | bị bắt; AZ lần đầu SỐNG SÓT (095/096 chưa vào ratchet) |
| PG (PA, PD–PH) | 6 | bị bắt |
| phát lại G6 (GA–GG) | 7 | bị bắt; GF lần đầu SỐNG SÓT → thêm `TestDiffMoiChang` |
| thuộc tính (PR1–PR3) | 3 | bị bắt |
| AST tenant (TA, TB) | 2 | bị bắt |
| ràng buộc claim (BC1–BC13) | 13 | bị bắt |
| so sánh / mệnh đề / số âm (CM1–CM4, FC1–FC2, NEG1–NEG2) | 8 | bị bắt |
| bộ đọc JSON (VM1) | 1 | SỐNG SÓT: đột biến sai chỗ (số lượng ghi dạng chuỗi). Thay bằng test âm trực tiếp cho bộ kiểm metamorphic |

Lọc ACL tài liệu có hai lớp, SQL và Go. Bỏ một lớp thì test vẫn xanh nhờ lớp kia. Đây là thiết kế phòng thủ hai lớp, không phải lỗ.

## Lỗi thật tìm ra hôm nay (đã sửa, đều có test)

1. **Thứ tự bằng chứng không tất định.** `sourceoftruth.Query` duyệt map nguồn nên thứ tự trích dẫn đổi giữa hai lần chạy. Phát lại G6 bắt được ở 17 ca.
2. **Bộ chuẩn hoá rơi dấu âm.** "-150 L" đọc thành 150, nên mọi hiệu âm của phép tính bị cổng S9 chặn khi giao. Test so sánh ngược chiều bắt được.
3. **Slice `nil` tới PG là NULL.** `cardinality(NULL) = 0` cho NULL, nên danh mục rỗng. Test PG bắt được.
4. **Truy xuất bỏ dấu.** "bí" khớp "bị", kéo ra tài liệu bị chèn lệnh (ca 072). "làm" khớp "làm việc", lấy nhầm bước (ca 116): **bộ ca không bắt lỗi này**, ca vẫn đạt. Đã khoá bằng test truy xuất.
5. **`Policy.Allows("")` mở mặc định.** Lỗi bảo mật, soát độc lập tìm ra. Đã sửa sáng 20/10.
6. **Truy xuất RAG không tất định.** Điểm cộng theo thứ tự duyệt map, nên đoạn hoà điểm đổi chỗ giữa các lần hỏi. Agent dựng baseline G7 đo được: 23/117 ca ra top-10 khác nhau qua 200 lần. Test lặp 200 lần đỏ trên mã cũ.

## Lỗ hổng của bảng 19/10 — trạng thái

| # | Lỗ hổng | Trạng thái |
|---|---|---|
| 1 | câu xác nhận "4k đúng không?" | XONG: so tuyệt đối với nguồn có thẩm quyền, kê số khác có nhãn; câu nhiều mệnh đề |
| 2 | thiếu thuật ngữ rơi về "ngoài phạm vi" | XONG: hỏi lại; D7 vật tư mặc định khi có thực thể |
| 3 | "dùng" là nghĩa nào | XONG: D3 = tiêu thụ đo được |
| 4 | mã theo nguồn (EX003, asset-8392…) | XONG: bí danh theo không gian tên nguồn |
| 5 | máy đổi dự án giữa kỳ | CHƯA, DEFER: chỉ mở khi một ca đỏ vì nó |
| 6 | câu chờ duyệt lộ mã nguồn | XONG: không đưa số, nhãn nguồn dễ đọc |
| 7 | kết quả có kiểu | XONG: `Outcome` |
| 8 | capability rỗng mở mặc định | XONG: đóng mặc định + luật lúc nạp pack |

Kiểm claim ở S9 không còn là "kiểm có mặt" đơn thuần. `BindClaims` đòi đủ chủ thể / chỉ số / kỳ / giá trị / đơn vị / nguồn. Route sinh số bằng model **vẫn chưa mở**: chỉ có model giả trích nguyên văn.

## Chưa có / chưa kiểm

| Hạng mục | Tình trạng |
|---|---|
| G7 với model thật | CHẶN: cần khoá + đồng ý chi phí. Client đã có (`models.Messages`, test bằng máy chủ giả), chưa từng gọi API thật. Tiêu chí chấm theo nội dung (`G7ContentReport`), top-k baseline = 10 và cách viết đoạn của baseline: chờ user duyệt |
| HTTP API, xác thực, `cmd/eais` | chưa có. Bản ghi quyết định mới nằm trong bộ nhớ / test, chưa lưu bền |
| Chỉ mục toàn văn PG | chưa có. Tài liệu lưu nguyên văn, cắt đoạn và chấm điểm trong Go lúc truy vấn; đủ cho WIN, chưa đo ở quy mô tài liệu lớn |
| Soát độc lập mã W4–W7 | CHƯA: mã hôm nay mới qua tự soát + đột biến |
| `go mod tidy` | chạy offline lỗi (cache thiếu dependency test của yaml). pgx v5.11.0 thêm từ cache; chạy lại khi có mạng |
| Thể thao; COMMAND / EVENT / WORKFLOW | DEFER theo `05` |
| Tài liệu `docs/tenants/**`, `docs/requirements/**` | do agent viết; chưa soát độc lập; `phase` trong sổ yêu cầu chưa xếp lại theo WIN |

## Quan sát ngoài phạm vi (chưa sửa)

| Quan sát | Tác động | Đề xuất |
|---|---|---|
| `docs/02-migration-audit.md` nhắc đường dẫn nội bộ của hệ thống cũ (2 chỗ) | repo PUBLIC | user quyết trước commit: giữ, hoặc thay bằng "hệ thống cũ" |
| "Zalo" xuất hiện 9 chỗ (`01-architecture.md`, `docs/tenants/construction/01-business-map.md`, sổ yêu cầu) | tên kênh chat của đội công trường, không phải tên dự án cũ | chấp nhận |
| Nhóm lỗi P10 trong `CLAUDE.md` khác CONVENTIONS §4.2 | hai danh sách lệch | chốt một danh sách |
| Sổ yêu cầu dán nhãn 294 REQ là "MVP" | rộng hơn hẳn WIN | xếp lại theo WIN / Phase 1–10 |
| `docs/tenants/sports/08-test-cases.md` có thể bị sinh lại nhầm lúc 20:49 ngày 19/10 | chưa chứng minh không mất sửa tay | chạy lại generator thể thao rồi diff |
| Bộ ca cầu đường cũ: số ca theo loại lối vào lệch mục tiêu | thiếu ca lệnh / workflow | bổ sung khi làm Phase 1–2 |
| Pack khai `reported_vs_issued` (dẫn xuất) nhưng không ý định nào dùng | thừa; câu so sánh đã đi đường `Compare` chung | giữ hoặc bỏ khi soát pack |

## Việc tiếp theo

1. Chủ dự án soát các commit ngày 20/10 trên GitHub. Repo PUBLIC: thứ cố ý không lên git ở `docs/ops/README.md` mục 7.
2. **G7:** cấp khoá model + đồng ý chi phí ⇒ thêm client provider vào gateway ⇒ chạy so sánh k = 3 với baseline. Đây là cổng duy nhất còn lại của WIN.
3. Soát độc lập (phiên không viết mã) cho mã hôm nay: pipeline S9, pgstore, so sánh / mệnh đề trọng tâm. Agent soát đã dừng 10:33 theo lệnh, chưa có phát hiện.
3b. Bộ ca đội đỏ (≈ 40 ca mới, do người không đọc mã soạn, ngoài bộ đóng băng) để đo tổng quát hoá. Hiện CHƯA có bằng chứng tổng quát hoá ngoài 117 ca. Agent dừng 10:33, chưa ghi tệp.
4. Lưu bền bản ghi quyết định (bảng `interactions`) khi có HTTP API.
5. DEFER theo `05`: tác vụ duyệt / lệnh (Phase 1–2), thể thao (Phase 4), lỗ hổng 5.
