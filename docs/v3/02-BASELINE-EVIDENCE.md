# 02 — Baseline Evidence: mốc trước V3

> Đo ngày **20/10/2026 15:22 (+07)** bằng lệnh của `docs/STATUS.md`, chưa sửa dòng mã nào.
> Đây là mốc "trước V3". Mọi thay đổi V3 phải giữ mốc này xanh, hoặc ghi rõ vì sao đổi (`CLAUDE.md` §22, §32).

## 1. Môi trường

| Mục | Giá trị |
|---|---|
| Revision | `85108c41d90eed795539fed65635435cb4e270c3` (cây sạch, 0 tệp đổi) |
| Go | `go1.26.4 linux/amd64` |
| PostgreSQL | 17.11 (container `eais-pg`, ảnh ghim digest, `127.0.0.1:25432`, tmpfs, `trust`) |
| Máy | 16 vCPU, RAM khả dụng ≈ 37,9 GB lúc đo (box dùng chung) |
| Công cụ ngoài | `pdftotext` (poppler), GNU `time` |

## 2. Lệnh và kết quả

```bash
docker compose -f deploy/docker-compose.yml up -d --wait
export EAIS_PG_ADMIN_DSN='postgres://postgres@127.0.0.1:25432/eais?sslmode=disable'
export EAIS_PG_DSN='postgres://eais_app@127.0.0.1:25432/eais?sslmode=disable'
go vet ./...
gofmt -l .
go test ./... -count=1 -json
scripts/scale-smoke.sh
go run ./tenants/construction/win/gen/cmd/gen -world tenants/construction/win/world.yaml -out <tmp>/win -profile functional -seed 42
go run ./tenants/construction/win/winreport -data <tmp>/win -v
go run ./tenants/construction/win/winreport -data <tmp>/win -v -pg
docker compose -f deploy/docker-compose.yml down
```

| ID | Lệnh | Kết quả | Thời gian |
|---|---|---|---|
| EV-B01 | `go vet ./...` | sạch (rc 0) | < 1 s |
| EV-B02 | `gofmt -l .` | rỗng | — |
| EV-B03 | `go test ./... -count=1 -json` | **224 PASS / 0 FAIL / 0 SKIP** = 127 test cấp cao + 97 test con. Gói: 20 có test, 6 không có tệp test | rc 0 |
| EV-B04 | `scripts/scale-smoke.sh` | **SCALE SMOKE: ĐẠT**: 117/117 trên cả hai hồ sơ; tập ca ĐẠT/ĐỎ giống hệt; câu trả lời giống từng byte | functional 0,10 s · 20,3 MiB; smoke 3,28 s · 566,5 MiB (peak RSS) |
| EV-B05 | `winreport -v` (đường bộ nhớ) | TỔNG 117/117; G1 2.739/2.739 bản ghi khớp manifest; metamorphic R-001 × 4, R-004 × 4, 0 lệch; **G6 phát lại 117/117, 0 lệch** | rc 0 |
| EV-B06 | `winreport -v -pg` (đường PG, RLS) | TỔNG 117/117; phát lại 0 lệch | rc 0 |
| EV-B07 | băm bộ dữ liệu, bộ ca | manifest `31172355392f7092fb01bced39fd7ce18c04be765bdd27c316442f2bf9adb055`; `cases-independent.yaml` `ffdd704b535d2195b96c2216113348c11c1e92d1a68ba31f4f0dfaa9d5d7a4bf` (khớp băm đóng băng 20/10 09:15) | — |

So với mốc trong `docs/STATUS.md` (20/10 10:35):
- 224/0/0 khớp;
- scale smoke ĐẠT khớp;
- thời gian và RAM khác chút ít (3,67 s / 562,8 MiB lúc trước so với 3,28 s / 566,5 MiB lúc này). Máy dùng chung, nên số không ổn định, chỉ để tham khảo, không phải ngưỡng.

Đầu ra thô lưu ở scratchpad của phiên lập kế hoạch, không đưa vào repo. Muốn có lại thì chạy lại các lệnh trên.

## 3. Phân loại

| Loại (`CLAUDE.md` §22) | Số | Ghi chú |
|---|---|---|
| REGRESSION | 0 | |
| EXISTING BUG | 1 quan sát mới, ca vẫn đạt | mục 4 |
| ENVIRONMENT | 0 đỏ | Không đặt biến PG thì 5 test PG **SKIP** (theo STATUS), không đỏ. Mốc V3 **bắt buộc** chạy có PG |
| MISSING DEPENDENCY | 0 đỏ | `go mod tidy` lỗi khi offline (STATUS); không ảnh hưởng build/test |
| FLAKY | 0 thấy | RAG không tất định đã sửa 20/10; chưa chạy lặp để đo flaky lần này (CHƯA ĐO) |
| EXPECTED | — | G7 cũ CHẶN: thiếu model thật, không phải test đỏ |
| UNKNOWN | 0 | |

## 4. Quan sát mới trong lúc đo

**OBS-01 — ghi chú xung đột sai phạm vi ở câu hỏi tổng.**

- Ca WIN-I-115 ("Dự án A hôm qua xuất bao nhiêu dầu?", người hỏi `u-eng-a`) giao câu:
  "Dầu kho xuất của Cầu A ngày 19/10/2026: 6.550 L. Nguồn khác lệch số: ERP 3.900 L; đã dùng nguồn có thẩm quyền cao nhất."
- 6.550 L đúng: R-004 3.850 + R-011 1.500 + R-016 1.200.
- Nhưng 3.900 L là số ERP của **riêng EX-03**, câu không nói. Người đọc hiểu "ERP nói cả dự án xuất 3.900 L". Đó là một khẳng định sai phạm vi, đúng loại lỗi G2 TRUTH và UX (Blueprint §28) phải chặn.
- Mức bằng chứng: **E4** (thấy ở đầu ra runtime, cả hai đường bộ nhớ và PG).
- Nguồn: mẫu `conflict_note` = `" Nguồn khác lệch số: {conflicts}{conflict_more}; …"` trong `tenants/construction/pack/templates.yaml`, không mang chủ thể của fact lệch.
- Bộ ca không bắt: kỳ vọng ca 115 chỉ đòi chứa "6.550" và cấm 4 chuỗi khác.
- Việc: TASK-R1-02 (test hồi quy ngoài bộ đông cứng, không đổi băm bộ ca).

## 5. Cách dùng mốc

- Mỗi task V3 chạy lại ít nhất EV-B01…EV-B04 trước khi báo xong, rồi so theo **tên test**, không chỉ đếm tổng (`CLAUDE.md` §22).
- Task chạm Fuel chạy thêm EV-B05/EV-B06.
- Không sửa test để xanh. Nếu đổi kỳ vọng (ví dụ TASK-R1-02 đổi câu chữ ghi chú xung đột), ghi lý do và đổi đúng test liên quan.
