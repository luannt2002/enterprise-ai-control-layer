# Vận hành EAIS — tiếp tục trên máy hoặc server khác

Tệp này là bản bàn giao công khai: pull repo về là đủ để chạy lại mốc và làm tiếp mà không mất luồng. Bản chi tiết có đường dẫn máy và tài khoản (`HANDOFF.local.md`, `AUTORUN.local.md`) chỉ nằm ở máy gốc. Mục 7 nói rõ thứ gì không có trong git và vì sao.

## 1. Trạng thái (20/10/2026, dừng 10:33 theo lệnh chủ dự án)

**WIN MVP: 3/7 cổng đạt vô điều kiện, 6/7 có điều kiện — CHƯA phải WIN** (đính chính 20/10 tối: `docs/CHO-QUYET-DINH.md`, D-G7-11).
- G1–G6 đạt; scale smoke đạt trên cả đường bộ nhớ lẫn PostgreSQL; bộ 117 ca độc lập đạt 117/117.
- **G7 CHẶN**: cần khoá model thật + chủ dự án đồng ý chi phí.

Chi tiết: `docs/STATUS.md`. Báo cáo ngày: `docs/bao-cao/2026-10-20-win-mvp.md`.

## 2. Yêu cầu máy

| Công cụ | Bản | Dùng cho |
|---|---|---|
| Go | 1.26.4 (`toolchain go1.26.4` trong `go.mod`) | build, test |
| Docker + compose v2 | bất kỳ bản còn hỗ trợ | PostgreSQL cho test. Ảnh ghim digest `postgres:17@sha256:f4c66b82…` (PG 17.11), máy mới tự kéo |
| poppler-utils (`pdftotext`) | bản của hệ điều hành | bộ nạp đọc báo cáo PDF (ADR-0001 D4). Thiếu thì test WIN tự SKIP |
| python3 | ≥ 3.10 | `scripts/mutate.sh` |
| git | — | đồng bộ |

## 3. Lấy mã và chạy lại mốc

```bash
git clone https://github.com/luannt2002/enterprise-ai-systems.git && cd enterprise-ai-systems   # đã có thì: git pull
go mod download
docker compose -f deploy/docker-compose.yml up -d --wait     # PG chỉ nghe 127.0.0.1:25432, dữ liệu tmpfs
export EAIS_PG_ADMIN_DSN='postgres://postgres@127.0.0.1:25432/eais?sslmode=disable'
export EAIS_PG_DSN='postgres://eais_app@127.0.0.1:25432/eais?sslmode=disable'
go vet ./... && gofmt -l . && go test ./... -count=1
scripts/scale-smoke.sh
```

Mốc phải ra (đo 20/10/2026 10:35, Go 1.26.4):
- `go vet` sạch, `gofmt -l` rỗng;
- `go test`: **224 PASS / 0 FAIL / 0 SKIP**. Không có biến PG thì 5 test PG SKIP;
- `scale-smoke.sh`: **SCALE SMOKE: ĐẠT** — 117/117 ở cả functional và smoke; G1 smoke 359.945/359.945; phát lại 0 lệch.

Lệch mốc thì dừng lại tìm nguyên nhân trước khi làm gì khác. Phân loại trước khi sửa: môi trường, đỏ sẵn, hay hồi quy thật (CLAUDE.md §32).

Lệnh WIN rời:

```bash
go run ./tenants/construction/win/gen/cmd/gen -world tenants/construction/win/world.yaml -out /tmp/win -profile functional -seed 42
go run ./tenants/construction/win/winreport -data /tmp/win -v        # đường bộ nhớ
go run ./tenants/construction/win/winreport -data /tmp/win -v -pg    # đường PostgreSQL (RLS)
go run ./tenants/construction/win/g7run -data /tmp/win               # G7: không có model thật thì in CHẶN, thoát 2
```

Đột biến: `scripts/mutate.sh <tệp> <chuỗi gốc> <chuỗi đột biến> [nhãn]`. Danh mục 69 đột biến đã chạy ngày 20/10 nằm ở `scripts/dot-bien/` (mỗi tệp một nhóm cổng; kỳ vọng mọi dòng "BỊ BẮT").

## 4. Biến môi trường

| Biến | Ý nghĩa | Ghi chú |
|---|---|---|
| `EAIS_PG_ADMIN_DSN` | DSN vai quản trị (migrate) | máy dev: `trust` + loopback, không mật khẩu |
| `EAIS_PG_DSN` | DSN vai ứng dụng `eais_app` (không bỏ qua RLS) | như trên |
| `EAIS_MODEL_PROVIDER` | `""` / `extractive` = model giả tất định; `anthropic` = API kiểu Messages | |
| `EAIS_MODEL` | tên model | bắt buộc với model thật — không có mặc định |
| `EAIS_MODEL_API_KEY` | khoá API | **không bao giờ ghi vào repo hay log** |
| `EAIS_MODEL_BASE_URL` | endpoint, cho dịch vụ tương thích | trống = endpoint chính thức |
| `EAIS_MODEL_MAX_TOKENS` | trần token đầu ra | trống = 1024 |

## 5. Mở phiên Claude Code để làm tiếp

Mở phiên ở gốc repo (`CLAUDE.md` tự nạp) rồi dán:

```text
Tiếp tục dự án EAIS. Đọc CLAUDE.md, docs/ops/README.md, docs/STATUS.md, docs/04-core-mvp.md §6, docs/adr/0002-win-quyet-dinh-trien-khai.md và docs/bao-cao/2026-10-20-win-mvp.md. Chạy lại mốc theo docs/ops/README.md mục 3 và so với mốc ghi ở đó; lệch thì báo tôi trước khi làm gì khác. Sau đó trình tôi các câu còn mở ở docs/CHO-QUYET-DINH.md (mỗi câu kèm khuyến nghị), rồi làm theo "Việc tiếp theo" của docs/STATUS.md. Muốn tự chạy không hỏi thì dựng luồng theo docs/ops/LUONG-TU-CHAY.md. Luật: repo PUBLIC — không đưa dữ liệu thật / khoá / transcript; WIN trước, roadmap sau; báo tiến độ bằng một con số %.
```

## 6. Câu chờ chủ dự án quyết

**Một nguồn duy nhất: `docs/CHO-QUYET-DINH.md`.** Mở phiên thì đọc tệp đó trước. Câu nào đã quyết thì nằm ở mục 0 của tệp.

Tóm tắt 20/10 tối:
- ADR-0001 giữ nguyên; ADR-0002 tạm chấp nhận;
- G7 chờ: sửa D5 (temperature 0 không chạy được với model hiện hành), chọn model, trần chi phí, B1, bộ giữ lại, đóng băng hợp đồng (`docs/g7/`);
- gói định nghĩa EACL V3 (`docs/v3/05`) chờ duyệt.

## 7. Không đưa lên git (repo PUBLIC) — và lấy lại thế nào

| Thứ | Vì sao không lên git | Ở máy khác thì sao |
|---|---|---|
| `docs/sources/` | nguồn thô: brief gốc, transcript, ghi chú học, có thể chứa dữ liệu thật / link nội bộ | logic đã chưng cất vào `docs/requirements/`. Cần bản gốc thì chép qua kênh riêng tư, không qua repo |
| `*.local.md` (`HANDOFF`, `AUTORUN`) | đường dẫn máy, tên tài khoản, mã lịch của phiên | bản công khai: tệp này + `docs/ops/LUONG-TU-CHAY.md` + `docs/ops/NHAT-KY-2026-10-20.md`; tạo lại `AUTORUN.local.md` theo mẫu |
| Memory Claude Code của máy | nằm chung với memory các dự án khác của chủ máy | phần EAIS (đã làm sạch): `docs/ops/BO-NHO.md` |
| Dữ liệu sinh của bộ ca WIN | tái tạo được, tất định theo seed | `gen … -seed 42` |
| Dữ liệu PostgreSQL | tmpfs, chỉ dữ liệu test | test và `winreport -pg` tự nạp lại |
| Khoá, `.env` | bí mật | đặt bằng biến môi trường (mục 4) |
