# Vận hành EAIS — tiếp tục trên máy hoặc server khác

Tệp này là bản bàn giao công khai: pull repo về là đủ để chạy lại mốc và làm tiếp mà không mất luồng. Bản chi tiết có đường dẫn máy và tài khoản (`HANDOFF.local.md`, `AUTORUN.local.md`) chỉ nằm ở máy gốc. Mục 7 nói rõ thứ gì không có trong git và vì sao.

## 1. Trạng thái (sáng 21/10/2026, sau luồng chạy đêm 20→21/10)

**WIN MVP: CHƯA phải WIN.** Trên bộ giữ lại độc lập H1 (104 ca, chạy MỘT lần 01:42 ngày 21/10), phần ca của G1–G5 đều KHÔNG ĐẠT (89/104). Theo dữ liệu độc lập: 0/7 cổng đạt trọn (G6' còn PARTIAL); theo cách đọc chỉ G2 / G5 / G6 (ADR-0006 §8, không khuyến nghị): 3/7 — chủ dự án chốt ở D-H1-1.
- Bộ 117 ca (bộ hồi quy) đạt 117/117; nạp dữ liệu 2.739/2.739; phát lại 104/104 lệch 0.
- Câu xác nhận ("… đúng không?"): bản cuối phán quyết sai 15/15 câu thăm dò của người soát (đầu đêm 8/15; 7 câu là hồi quy của chính đêm nay), báo cáo sáng mục 8. D-OP-10 đã chốt ngày 21/10: chỉ phán quyết khi claim đã giải đủ chiều. Đã sửa bằng cổng phán quyết (ADR-0009), vào mã sau khi H3 đóng băng (`259d38d`); 15 câu thăm dò còn 0 phán quyết sai.
- **G7 CHẶN**: chưa có khoá model; bộ chấm v2 chưa đóng băng.

Chi tiết: `docs/STATUS.md`. Báo cáo đêm: `docs/bao-cao/2026-10-21-overnight.md`. Báo cáo ngày 20/10: `docs/bao-cao/2026-10-20-win-mvp.md`.

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
git clone https://github.com/luannt2002/enterprise-ai-control-layer.git && cd enterprise-ai-control-layer   # đã có thì: git pull
go mod download
docker compose -f deploy/docker-compose.yml up -d --wait     # PG chỉ nghe 127.0.0.1:25432, dữ liệu tmpfs
export EAIS_PG_ADMIN_DSN='postgres://postgres@127.0.0.1:25432/eais?sslmode=disable'
export EAIS_PG_DSN='postgres://eais_app@127.0.0.1:25432/eais?sslmode=disable'
go vet ./... && gofmt -l . && go test ./... -count=1
scripts/scale-smoke.sh
```

Test PG (`platform/pgstore`) tạo tạm các vai `eais_test_*` trong chính cụm PG mà `EAIS_PG_ADMIN_DSN` trỏ tới, tức cụm dùng chung với `eais_giu_lai`:
- vai tạm là `NOLOGIN`, vào bằng tham số `role=` trên DSN quản trị, và bị xoá khi test xong;
- nếu tiến trình bị kill giữa chừng, vai có thể sót nhưng không đăng nhập được;
- kiểm vai sót: `SELECT rolname FROM pg_roles WHERE rolname LIKE 'eais_test_%'`.

Mốc phải ra (đo 21/10/2026 tại `9dede54`, Go 1.26.4; đếm mọi sự kiện pass / fail / skip của `go test -json`, kể cả test con):
- `go vet` sạch, `gofmt -l` rỗng;
- `go test`: **440 PASS / 0 FAIL / 0 SKIP** (317 test cấp cao). Không có biến PG: 435 PASS + 5 SKIP (5 test PG);
- `scale-smoke.sh`: **SCALE SMOKE: ĐẠT** — 117/117 ở cả functional và smoke; G1 smoke 359.945/359.945; phát lại 0 lệch.

Lệch mốc thì dừng lại tìm nguyên nhân trước khi làm gì khác. Phân loại trước khi sửa: môi trường, đỏ sẵn, hay hồi quy thật (CLAUDE.md §32).

Container PG dựng trong khoảng 09:29 → 18:4x ngày 21/10/2026 mang tên project compose cũ (`eacl`, container `eacl-pg`, database `eacl`): gỡ bằng `docker compose -p eacl down` rồi dựng lại bằng lệnh trên (dữ liệu tmpfs, không mất gì). Container dựng trước đó (20/10) đã mang tên hiện hành. Database đã áp `0001_win.sql` thì migration `0002_dinh_danh_eais.sql` tự chuyển GUC và vai ứng dụng sang tên hiện hành (`eais.tenant_id`, `eais_app`) và gỡ quyền của vai cũ.

Bộ giữ lại (`scripts/chay-giu-lai.sh`) chỉ chạy trên database riêng `eais_giu_lai`: tạo bằng `docker exec eais-pg psql -U postgres -c "CREATE DATABASE eais_giu_lai"`, rồi đặt `EAIS_PG_ADMIN_DSN` / `EAIS_PG_DSN` trỏ vào nó (migration tự áp khi chạy; 21/10 đã thử với database bộ giữ lại tên khi đó: đường PG 117/117).

Lệnh WIN rời:

```bash
go run ./tenants/construction/win/gen/cmd/gen -world tenants/construction/win/world.yaml -out /tmp/win -profile functional -seed 42
go run ./tenants/construction/win/winreport -data /tmp/win -v        # đường bộ nhớ
go run ./tenants/construction/win/winreport -data /tmp/win -v -pg    # đường PostgreSQL (RLS)
go run ./tenants/construction/win/g7run -data /tmp/win               # G7: không có model thật thì in CHẶN, thoát 2
```

Đột biến: `scripts/mutate.sh <tệp> <chuỗi gốc> <chuỗi đột biến> [nhãn]`. Danh mục ở `scripts/dot-bien/` (35 tệp, mỗi tệp một nhóm cổng; kỳ vọng mọi dòng "BỊ BẮT"). Lần chạy đủ gần nhất (21/10 sáng): 598 đột biến bị bắt, 0 sống — `docs/bao-cao/2026-10-21-migration-eacl.md`. `mutate.sh` sửa tệp tại chỗ: chạy trên bản clone riêng, đừng đo gì khác trong clone đó cùng lúc.

## 4. Biến môi trường

| Biến | Ý nghĩa | Ghi chú |
|---|---|---|
| `EAIS_PG_ADMIN_DSN` | DSN vai quản trị (migrate) | máy dev: `trust` + loopback, không mật khẩu |
| `EAIS_PG_DSN` | DSN vai ứng dụng `eais_app` (không bỏ qua RLS) | như trên |
| `EAIS_MODEL_PROVIDER` | `""` / `extractive` = model giả tất định; `anthropic` = API kiểu Messages | |
| `EAIS_MODEL` | tên model | bắt buộc với model thật — không có mặc định |
| `EAIS_MODEL_API_KEY` | khoá API | **không bao giờ ghi vào repo hay log** |
| `EAIS_MODEL_BASE_URL` | endpoint, cho dịch vụ tương thích | trống = endpoint chính thức. Chỉ nhận `https://` hoặc `http://` tới loopback; client không theo chuyển hướng (khoá không đi theo, `d4b80be`). `g7run` so endpoint theo nghĩa (`:443`, chữ hoa, dấu chấm cuối vẫn là API chính thức); endpoint khác là CHẠY KHÔ, nhưng mọi endpoint không phải loopback (cả proxy) vẫn qua kiểm trước khi tiêu tiền (`246eabb`) |
| `EAIS_MODEL_MAX_TOKENS` | trần token đầu ra | trống = 1024; giá trị hỏng là lỗi cấu hình, không lặng lẽ về 1024 |
| `EAIS_MODEL_THINKING` | kiểu thinking gửi cho nhà cung cấp | hợp đồng G7 §4: `between_tools` |
| `EAIS_MODEL_EFFORT` | `output_config.effort` | hợp đồng G7 §4: `medium` |
| `EAIS_MODEL_TEMPERATURE` | sampling | trống = KHÔNG gửi (model hiện hành trả 400 với temperature khác mặc định; ADR-0005) |
| `EAIS_MODEL_MAX_RETRIES` | số lần thử lại lỗi tạm (429, 5xx, quá giờ) | trống = 2, lùi 1 s rồi 2 s, có trần |

Bí danh tương thích (D-RM-02): biến cùng tên tiền tố `EACL_` (`EACL_MODEL_*`, `EACL_PG_ADMIN_DSN`, `EACL_PG_DSN`, tên dùng ngày 21/10/2026 trước khi đổi tên lại) vẫn được đọc khi biến `EAIS_` cùng tên KHÔNG được đặt; biến `EAIS_` đã đặt, kể cả rỗng, luôn thắng. Nơi đọc: `models.modelEnv`, `pgstore.EnvDSN`, `scripts/chay-giu-lai.sh`.

## 5. Mở phiên Claude Code để làm tiếp

Mở phiên ở gốc repo (`CLAUDE.md` tự nạp) rồi dán:

```text
Tiếp tục dự án EAIS. Đọc CLAUDE.md, docs/ops/README.md, docs/STATUS.md, docs/04-core-mvp.md §6, docs/adr/0002-win-quyet-dinh-trien-khai.md, docs/adr/0006-bo-giu-lai-doc-lap.md và docs/bao-cao/2026-10-21-overnight.md. Chạy lại mốc theo docs/ops/README.md mục 3 và so với mốc ghi ở đó; lệch thì báo tôi trước khi làm gì khác. Sau đó trình tôi các câu còn mở ở docs/CHO-QUYET-DINH.md (mỗi câu kèm khuyến nghị), rồi làm theo "Việc tiếp theo" của docs/STATUS.md. Muốn tự chạy không hỏi thì dựng luồng theo docs/ops/LUONG-TU-CHAY.md. Luật: repo PUBLIC — không đưa dữ liệu thật / khoá / transcript; WIN trước, roadmap sau; báo tiến độ bằng một con số %.
```

## 6. Câu chờ chủ dự án quyết

**Một nguồn duy nhất: `docs/CHO-QUYET-DINH.md`.** Mở phiên thì đọc tệp đó trước. Câu nào đã quyết thì nằm ở mục 0 của tệp.

Tóm tắt 20/10 tối:
- ADR-0001 giữ nguyên; ADR-0002 tạm chấp nhận;
- G7 chờ: sửa D5 (temperature 0 không chạy được với model hiện hành), chọn model, trần chi phí, B1, bộ giữ lại, đóng băng hợp đồng (`docs/g7/`);
- gói định nghĩa V3 (`docs/v3/05`) chờ duyệt.

## 7. Không đưa lên git (repo PUBLIC) — và lấy lại thế nào

| Thứ | Vì sao không lên git | Ở máy khác thì sao |
|---|---|---|
| `docs/sources/` | nguồn thô: brief gốc, transcript, ghi chú học, có thể chứa dữ liệu thật / link nội bộ | logic đã chưng cất vào `docs/requirements/`. Cần bản gốc thì chép qua kênh riêng tư, không qua repo |
| `*.local.md` (`HANDOFF`, `AUTORUN`) | đường dẫn máy, tên tài khoản, mã lịch của phiên | bản công khai: tệp này + `docs/ops/LUONG-TU-CHAY.md` + `docs/ops/NHAT-KY-2026-10-20.md`; tạo lại `AUTORUN.local.md` theo mẫu |
| Memory Claude Code của máy | nằm chung với memory các dự án khác của chủ máy | phần EAIS (đã làm sạch): `docs/ops/BO-NHO.md` |
| Dữ liệu sinh của bộ ca WIN | tái tạo được, tất định theo seed | `gen … -seed 42` |
| Dữ liệu PostgreSQL | tmpfs, chỉ dữ liệu test | test và `winreport -pg` tự nạp lại |
| Khoá, `.env` | bí mật | đặt bằng biến môi trường (mục 4) |
