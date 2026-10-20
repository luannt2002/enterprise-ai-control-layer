# STATUS — trạng thái EAIS

> Cập nhật: 21/10/2026 01:5x (sau H1), **luồng chạy đêm 20→21/10 đang chạy** (mục "Đêm 20→21/10" ngay dưới; báo cáo sáng: `docs/bao-cao/2026-10-21-overnight.md`). Phiên tự chạy ban ngày dừng 10:33 theo lệnh chủ dự án. Chưa phát hành phiên bản nào.
> Báo cáo ngày: `docs/bao-cao/2026-10-20-win-mvp.md`. Chạy lại / làm tiếp ở máy khác: `docs/ops/README.md`.
> **Câu đang chờ chủ dự án quyết: `docs/CHO-QUYET-DINH.md`.** Checkpoint G7 (A–H, chưa gọi model thật): `docs/g7/CHECKPOINT-A-H.md`.
> Mọi số ở đây lấy từ lệnh chạy ngày 20/10/2026, trừ chỗ ghi **CHƯA ĐO**.

## Tiến độ WIN MVP: trên bộ giữ lại độc lập H1, phần ca của G1–G5 KHÔNG ĐẠT (89/104) — CHƯA phải WIN

> **H1, 21/10 01:42** (`docs/g7/holdout/ket-qua-h1/`). Bộ giữ lại độc lập: 104 ca, 13 / lớp, do người không đọc mã soạn, đóng băng 22:51 ngày 20/10, chạy MỘT lần ở revision `9de63d2`.
> - Kết quả: **89/104** ở cả hai đường (bộ nhớ, PostgreSQL) theo bộ chấm nhị phân. Theo nội dung **≤ 88/104**: ca 264 ĐẠT với câu trả lời SAI, "1 490 lít" không được gộp nhóm nghìn; người phản bác cuối tìm ra, đã sửa sau H1 ở `8ff3db5`. Phát lại 104/104 lệch 0; canary 0.
> - Theo lớp và cổng (`04` §6 đòi 100 % mỗi lớp):
>   - representation 10/13 (G1);
>   - entity 13/13, ambiguity 12/13 (G2);
>   - semantic 11/13, source_conflict 11/13 (G3);
>   - routing 7/13 (G4);
>   - permission 13/13, unsupported_answer 12/13 (G5).
> - Vậy phần ca của G1–G5 đều KHÔNG ĐẠT trên dữ liệu độc lập. 15 ca đỏ quyết định sai ở tầng ý định (S4; 13/15 sai đầu tiên ở đó); 4/15 trả lời sai im lặng (`phan-tich.md`).
> - G6' ĐẠT, nhưng tiêu chí sửa đổi chờ chủ dự án phê. G7 CHẶN.
> - **Tiến độ (cổng đạt trọn, vô điều kiện, theo dữ liệu độc lập): 0/7.**
>   - Nếu chủ dự án phê G6': 1/7.
>   - Nếu D-H1-1 chọn giữ bộ 117 cho G1 / G3 / G4: 3/7 (4/7 với G6').
> - Bảng dưới là kết quả trên bộ 117, giữ để đối chiếu.

> **Đính chính 20/10 tối (bản soát độc lập `docs/g7/REVIEW-D5.md`):**
> - Ba ca 083, 107, 108 có kỳ vọng do chính phiên viết mã "trọng tài" **sau** lần đo đầu (09:03), **trước** khi băm bộ ca (09:15).
> - Vì vậy G2, G5, G6 chỉ đạt **có điều kiện**. Đạt vô điều kiện: **3/7** (G1, G3, G4).
> - Chờ chủ dự án quyết D-G7-11 (`docs/CHO-QUYET-DINH.md`).

Đích hiện tại là **WIN MVP** (`docs/04-core-mvp.md`), không phải cả roadmap (`docs/05-master-roadmap.md`).

**WIN = 7/7 cổng + scale smoke** (04 §6). Cách tính tiến độ: mỗi cổng 1/7, chỉ tính cổng đạt trọn và vô điều kiện ⇒ **3/7 ≈ 43%**. Nếu D-G7-11 chọn "c" (loại 3 ca trọng tài khỏi phép tính cổng) hoặc "a" (duyệt trọng tài) thì 6/7 ≈ 86%.

- G1, G3, G4 đạt theo đúng tiêu chí của 04 §6. G2, G5, G6 đạt trên bộ ca hiện tại nhưng phụ thuộc 3 kỳ vọng trọng tài (xem đính chính trên).
- Scale smoke đạt trên cả đường bộ nhớ lẫn đường PostgreSQL.
- **G7 VALUE: CHẶN.** Cần model thật, tức khoá API và chủ dự án đồng ý chi phí. 04 §6 cấm thay bằng model giả.

| Cổng | Kết quả | Bằng chứng |
|---|---|---|
| G1 DATA | ĐẠT | - G1: 2.739/2.739 (functional), 359.945/359.945 (smoke).<br>- Metamorphic: R-004 × 4 biểu diễn, R-001 × 4 cách viết, 0 lệch.<br>- Bộ kiểm có test âm.<br>- `representation` 16/16. |
| G2 IDENTITY | ĐẠT **có điều kiện** (107, 108 là kỳ vọng trọng tài) | - `entity` 13/13.<br>- `ambiguity` 13/13, ứng viên đúng tập. |
| G3 TRUTH | ĐẠT | - `semantic` 14/14, `source_conflict` 12/12.<br>- Ca biên dung sai sinh từ pack cho MỌI cặp nguồn: 4 cặp × "= dung sai" / "dung sai + 1", gồm cặp ngang hạng. |
| G4 DECISION | ĐẠT | - `routing` 17/17.<br>- Runner chấm `model_calls` bằng gateway giả đếm lời gọi.<br>- Đột biến bắt buộc "SQL gọi gateway" (BA) và "RAG bỏ gateway" (BB) đều bị bắt. |
| G5 CONTROL | ĐẠT **có điều kiện** (083 là kỳ vọng trọng tài) | - `permission` 18/18.<br>- Cổng AST: không kiểu `*Request` nào mang tenant.<br>- PG: RLS FORCE + GUC đọc lại; canary B có test cho phép / từ chối, đối chứng dương, ghi chéo tenant bị WITH CHECK chặn.<br>- Ràng buộc claim 6 chiều. Bảng "sai đúng một chiều" 6 × 3 bị chặn 100%; 0 chặn oan trên 13 ca mẫu + 117 ca thật.<br>- Lượng dầu trong tài liệu bị chặn bởi hai lớp, một lớp độc lập với bộ chuẩn hoá.<br>- Chèn lệnh trong SOP không đổi được route / quyền / số. |
| G6 PROOF | ĐẠT **có điều kiện**: vi phạm "đóng băng trước W1" ở 3 ca | - 117 ca do phiên không viết mã soạn (08:47–08:58); mọi lớp ≥ 12 ca.<br>- `cases-independent.yaml` sha256 `ffdd704b535d2195…`, băm ghi 09:15, **sau** lần đo đầu 09:03 và sau khi phiên viết mã trọng tài 3 kỳ vọng (083, 107, 108).<br>- Bản ghi quyết định + phát lại: 117/117 không lệch.<br>- Thuộc tính: tổng ≥ 0; dẫn xuất = a − b lấy từ bằng chứng; A không thấy B.<br>- Mỗi cổng có đột biến bị bắt. |
| G7 VALUE | **CHẶN** | - Bộ khung xong: baseline RAG (`evals/baseline`), luật D5 + bộ đếm độc lập (`evals/g7.go`), lệnh `g7run`, client model thật `models.Messages` cấu hình qua biến môi trường.<br>- Chạy ống nối bằng model giả: D5 ĐẠT, nhưng KHÔNG phải bằng chứng (04 §6).<br>- Thiếu: khoá model + đồng ý chi phí. Có rồi thì chạy `EAIS_MODEL_PROVIDER=anthropic EAIS_MODEL=<model> EAIS_MODEL_API_KEY=<khoá> go run ./tenants/construction/win/g7run -data <dir>`. |
| Scale smoke | ĐẠT | - `scripts/scale-smoke.sh`: 117/117 ở cả functional và smoke, câu trả lời giống từng byte; smoke 3,67 s, peak RSS 562,8 MiB (đo 10:35).<br>- Đường PG (`winreport -pg`): 117/117, từng ca và từng câu trả lời giống đường bộ nhớ; smoke 6,08 s, 653,7 MiB, kể cả nạp ~355 nghìn fact vào PG. |

Mốc "≈ 20%" của tối 19/10 bỏ, vì nó ước theo cảm nhận. Từ nay chỉ đếm cổng có lệnh chứng minh.

## Đêm 20→21/10 — khôi phục bằng chứng (đang chạy)

Mục tiêu đêm: khôi phục bằng chứng đáng tin rồi mới xác định trạng thái WIN thật; không tối ưu tỉ lệ đạt. Chưa gọi model thật, chi 0 USD (khoá model không có trong môi trường).

| Việc | Trạng thái | Bằng chứng |
|---|---|---|
| Ghi quyết định chủ dự án (D-G7-1…11, D-V3, D-OP) | XONG | `docs/CHO-QUYET-DINH.md` (`c5ce51e`) |
| Tái lập mốc `94bbaae` | XONG: test 224/0/0; 117/117 bộ nhớ + PG; phát lại 0 lệch; scale smoke ĐẠT; đột biến 69/69 | đầu ra thô ở máy |
| ADR phương pháp, đăng ký TRƯỚC khi có bộ giữ lại | XONG | ADR-0005 (đã duyệt, D5 sửa đổi), ADR-0006 (bộ giữ lại) (`6f7b4f4`); ADR-0008 (`0eb2531`) |
| Kiểm toán nhiễm 117 ca | XONG: CLEAN 102, CONTAMINATED 3 (083/107/108), UNCERTAIN 12; G6 không đạt điều kiện "đóng băng trước W1" | `docs/g7/CONTAMINATION-AUDIT.md` (`a135c90`) |
| Client model | XONG: không gửi `temperature` (từng làm mọi lời gọi lỗi 400); thinking / effort / thử lại từ cấu hình; lùi có trần | `750307a`, `fe22e2a` |
| Harness G7: B2+ / B1 / B2″ / B2-oracle; `-out`, `-smoke`, `-pilot`, `-budget`; lượt vô hiệu; PROVIDER | XONG | `a1a5b6b`, `fdb3a18`, `a94b5fb`, `15cb1a8` |
| `CompareG7` theo D5 sửa đổi | XONG | `a507e81` |
| **Bộ chấm G7 v1** | **BỊ PHÁ** (170 đòn: 109 câu sai mà ĐẠT) ⇒ không dùng làm bằng chứng. Đã vá phần chắc chắn: ratchet 170 đòn (nợ 122 → 118), rỗng = từ chối, hệ lỗi khi hỏi luôn trượt, số được phép từng ca, canary tên | `82baa71`, `6b7d963`, `9264aa6`, `7f6711c` |
| Bộ chấm G7 v2 | ĐANG: 2 hướng thiết kế xong, giám khảo đang tổng hợp. **Không đóng băng được đêm nay**: điều kiện bắt buộc là hiệu chỉnh trên đầu ra pilot thật (cần khoá) | ADR-0007 viết sau khi có kết quả |
| Bộ giữ lại độc lập | **ĐÃ ĐÓNG BĂNG 22:51** trước mọi lần chạy: 229 ca → 3 người soát (252 cờ) → 189; loại 7 ca bị lộ trước khi chia ⇒ pilot 16, **H1 104**, H2 62, exposed 7 | `16a6140`; `af4054a`; `docs/g7/holdout/` |
| Sự cố lộ (đã xử lý) | (1) 7 ca lộ cho người viết mã ⇒ loại trước khi chia; (2) 15 ca có gợi ý kỳ vọng lộ cho phiên chính sau đóng băng ⇒ tập không đổi, bảng H1 báo độ nhạy | `af4054a`, `59ee556`; D-G7-13 |
| Chứng minh lại G2 / G5 / G6 trên H1 | **XONG 01:42** (chạy MỘT lần, sau khi sửa mọi phát hiện S1/S2 của soát lượt 2):<br>- 89/104;<br>- G2' KHÔNG ĐẠT, G5' KHÔNG ĐẠT, G6' ĐẠT (chờ phê);<br>- phần ca của G1 / G3 / G4 cũng KHÔNG ĐẠT (D-H1-1).<br>15 ca đỏ đều ở S4; không sửa mã theo H1 | `56101be`, `d17144f`; `docs/g7/holdout/ket-qua-h1/` |
| Soát độc lập mã (D-OP-5) | Lượt 1 XONG 21:22: sửa S1-1, S2-1…S2-6, S3-x, S4, test hổng. Lượt 2 ĐANG (pipeline, công cụ bộ giữ lại, harness G7; mỗi chiều có người phản bác) | `a94b5fb`, `fe22e2a`, `5ce22ec`, `3d730ea` |
| Chương trình đo của người soát D5 | Công khai, tái lập đúng số của bản soát | `tenants/construction/win/d5probe` (`d03f962`) |
| `go mod tidy` (D-OP-2) | DỪNG theo luật (thêm 2 module chỉ cho test của yaml/v3; 0 gói EACL dùng); build / vet / test sạch sau tidy ⇒ khuyến nghị chấp nhận, chờ chủ dự án | `ba83265` |
| Câu chờ quyết mới | D-G7-12 (`04` dòng 83 trái D7), D-G7-13 (7 ca lộ: đã loại), D-G7-14 (gấp ký tự nhìn giống cho câu hỏi) | `docs/CHO-QUYET-DINH.md` |

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

> Thứ tự của đêm 20→21/10 (bảng ngay trên):
> 1. đóng băng bộ giữ lại;
> 2. chứng minh lại G2 / G5 / G6 trên H1;
> 3. bộ chấm v2 (ADR-0007) + test từ 170 đòn;
> 4. dự toán chi phí;
> 5. hợp đồng G7 sẵn sàng đóng băng sau pilot.
>
> Pilot và lượt G7 thật cần khoá do chủ môi trường đặt. Các mục dưới là danh sách ban ngày 20/10, giữ để đối chiếu.

1. Chủ dự án soát các commit ngày 20/10 trên GitHub. Repo PUBLIC: thứ cố ý không lên git ở `docs/ops/README.md` mục 7.
2. **G7:** làm theo `docs/g7/CHECKPOINT-A-H.md`:
   - chủ dự án quyết D-G7-1…11 (sửa D5 vì temperature 0 không chạy được, model, trần chi phí, B1, bộ giữ lại, đóng băng hợp đồng, cấp khoá, gói đề xuất của bản soát, câu đồng nghĩa "dùng", 3 kỳ vọng trọng tài);
   - làm các blocker mã (client, `-smoke`, `-out`, B1);
   - thử khói, rồi chạy k = 3.
   Đây là cổng duy nhất còn lại của WIN.
3. Soát độc lập (phiên không viết mã) cho mã hôm nay: pipeline S9, pgstore, so sánh / mệnh đề trọng tâm. Agent soát đã dừng 10:33 theo lệnh, chưa có phát hiện.
3b. Bộ ca đội đỏ (kế hoạch cũ ≈ 40 ca; nếu làm bộ giữ lại G7 thì ≥ 96 ca theo D-G7-5 / D-OP-5; do người không đọc mã soạn, ngoài bộ đóng băng) để đo tổng quát hoá. Hiện CHƯA có bằng chứng tổng quát hoá ngoài 117 ca. Agent dừng 10:33, chưa ghi tệp.
4. Lưu bền bản ghi quyết định (bảng `interactions`) khi có HTTP API.
5. DEFER theo `05`: tác vụ duyệt / lệnh (Phase 1–2), thể thao (Phase 4), lỗ hổng 5.
