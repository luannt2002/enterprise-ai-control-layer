# EAIS — Enterprise AI Systems

**Nền tảng multi-tenant biến dữ liệu, tri thức, quy trình và công cụ của doanh nghiệp thành năng lực AI có kiểm soát, kiểm thử được và quan sát được.**
- **Dùng lại** hệ thống doanh nghiệp đã có khi hệ đó đang giữ sự thật (chế độ A, tích hợp).
- **Tự lo** phần doanh nghiệp còn thiếu: thu dữ liệu, chuẩn hoá, đối soát (chế độ B).
- Core **không gắn** vào một ngành.

EAIS is a multi-tenant Enterprise AI Systems Platform for turning domain-specific business data, knowledge, workflows, and tools into governed, testable AI capabilities without coupling the core platform to a single industry.

Phần mềm tất định giữ quyền quyết định về danh tính, tenant, quyền hạn, sự thật nghiệp vụ, trạng thái, giao dịch, chính sách, kiểm và thực thi. AI chỉ làm phần bất định: diễn giải, phân loại, trích xuất, tổng hợp, đề xuất. Model là một năng lực bên trong hệ, không phải nguồn sự thật.

```text
Business Reality → Trusted Context → Useful Intelligence → Controlled Action → Verified Outcome → Measured Value
```

**Trạng thái (21/10/2026):**
- Bài chứng minh đầu tiên là **Fuel WIN**: đối soát nhiên liệu cho tenant xây dựng, đạt khi qua đủ 7 cổng G1–G7 ([`docs/04-core-mvp.md`](docs/04-core-mvp.md) §6). **H1** là bộ ca giữ lại độc lập: do người không đọc mã soạn, đóng băng trước, chỉ chạy một lần (ADR-0006).
- Fuel WIN: **0/7 cổng** trên H1 (104 ca): 89/104 ca đạt; G1–G5 chưa đạt, G6' một phần, G7 bị chặn (chưa có khoá model, bộ chấm v2 chưa có).
- Bộ giữ lại độc lập thứ hai **H3** (134 ca, đóng băng trước mọi bản sửa, chạy một lần ngày 21/10): **109/134**; mốc trước bản sửa chạy trên cùng H3: 113/134. Phần ca G1–G5 vẫn chưa đạt ⇒ vẫn **0/7**.
- Câu xác nhận ("… đúng không?") còn phán quyết sai trên 15/15 câu thăm dò đối kháng. Đã sửa ngày 21/10 bằng cổng phán quyết fail-closed ([ADR-0009](docs/adr/0009-cong-phan-quyet-cau-xac-nhan.md)): chỉ mở đầu bằng "Đúng." / "Không." khi claim đã giải đủ chiều; 15 câu thăm dò còn 0 phán quyết sai. Trên H3 (câu do người độc lập soạn): 0 phán quyết sai chiều ở cả mốc lẫn bản sửa; cổng giữ lại 12 phán quyết đúng (28 → 16 trên 42 ca xác nhận), 4 trong số đó làm ca trượt — cái giá đo được của "không phán quyết khi chưa giải đủ".
- Chi tiết: [`docs/STATUS.md`](docs/STATUS.md), báo cáo mới nhất ở [`docs/bao-cao/`](docs/bao-cao/).

| Đọc gì | Ở đâu |
|---|---|
| **Câu đang chờ chủ dự án quyết** | [`docs/CHO-QUYET-DINH.md`](docs/CHO-QUYET-DINH.md) |
| **Roadmap hiện hành** (D-RM-01): Phase 0–10, 7 mặt phẳng, luồng L0–L10 | [`docs/ROADMAP-DAY-DU.md`](docs/ROADMAP-DAY-DU.md) |
| G7: hợp đồng benchmark, chi phí, checkpoint | [`docs/g7/`](docs/g7/) |
| Kế hoạch V3 (tra cứu; thay bởi roadmap đầy đủ, D-RM-01) | [`docs/v3/`](docs/v3/) |
| Chạy lại mốc / làm tiếp trên máy khác | [`docs/ops/README.md`](docs/ops/README.md) |
| Hợp đồng vận hành (luật bắt buộc) | [`CLAUDE.md`](CLAUDE.md) |
| WIN MVP: 7 cổng, phạm vi | [`docs/04-core-mvp.md`](docs/04-core-mvp.md) |
| Roadmap gốc 20/10 (Phase 0–10 mà roadmap hiện hành dựa vào) | [`docs/05-master-roadmap.md`](docs/05-master-roadmap.md) |
| Kiến trúc | [`docs/01-architecture.md`](docs/01-architecture.md) |
| Quyết định | [`docs/adr/`](docs/adr/) |
| Báo cáo ngày | [`docs/bao-cao/`](docs/bao-cao/) |
| Thay đổi | [`CHANGELOG.md`](CHANGELOG.md) |

Repo: `github.com/luannt2002/enterprise-ai-control-layer` (cũng là đường dẫn module Go).

Chạy nhanh (Go 1.26, Docker, `pdftotext`):

```bash
docker compose -f deploy/docker-compose.yml up -d --wait
export EAIS_PG_ADMIN_DSN='postgres://postgres@127.0.0.1:25432/eais?sslmode=disable'
export EAIS_PG_DSN='postgres://eais_app@127.0.0.1:25432/eais?sslmode=disable'
go test ./... -count=1 && scripts/scale-smoke.sh
```

Mọi dữ liệu trong repo là dữ liệu giả lập; repo công khai không chứa dữ liệu thật của tenant nào.

## Chạy thử trong 5 phút

Ứng dụng `eais` (một binary: HTTP + CLI) chạy trên PostgreSQL với dữ liệu giả lập seed 42. Mặc định **không gọi model**: luồng hỏi dùng model tất định (extractive) và ứng dụng không có cờ hay đường nào tới model trả tiền. Cần: Go (theo `go.mod`), Docker, `curl`, `python3`, `pdftotext` (`poppler-utils`).

**Một lệnh** (dựng PG → migrate → sinh seed 42 → nạp → chạy server nền → smoke E2E bằng curl → dừng server):

```bash
git clone https://github.com/luannt2002/enterprise-ai-control-layer && cd enterprise-ai-control-layer
scripts/chay-thu.sh          # EAIS_PG_DOWN=1 để gỡ PG khi xong
```

> **Cảnh báo: script GHI vào PG mà nó dùng** (migrate rồi nạp THAY toàn bộ dữ liệu của tenant trong pack construction). Với giá trị mặc định nó chỉ chạy trên PG do chính nó dựng (container `eais-pg`, cổng `127.0.0.1:25432`) và **từ chối** — thoát 2, chưa ghi gì — nếu container đó đang chạy hoặc cổng đó đã có tiến trình nghe, kể cả PG còn sót lại của lần chạy trước. Đã có PG khác đang chạy: đổi `EAIS_PG_CONTAINER`, `EAIS_PG_PORT`, `EAIS_COMPOSE_PROJECT` để script dựng PG riêng; chỉ khi chắc PG đang chạy là của bạn và ghi đè được mới đặt `EAIS_REUSE_PG=1`. Ngoại lệ: đã `export` cả `EAIS_PG_ADMIN_DSN` và `EAIS_PG_DSN` (như ở mục "Chạy nhanh" trên) thì script dùng đúng PG đó, không dựng gì và không hỏi — bỏ chúng khỏi shell (`unset`) nếu không muốn script ghi vào PG ấy.

Thời gian đo ngày 22/10/2026 từ một clone sạch (16 vCPU, ảnh PG đã kéo sẵn, module Go có sẵn; thời gian kéo ảnh lần đầu **chưa đo**): **9,6 giây** với bộ nhớ đệm build rỗng (PG do chính script dựng), **3,9 giây** với bộ nhớ đệm ấm. Thành công khi in `smoke E2E: 21 đạt, 0 hỏng`. Đổi tên container / cổng / địa chỉ nghe bằng `EAIS_PG_CONTAINER`, `EAIS_PG_PORT`, `EAIS_COMPOSE_PROJECT`, `EAIS_ADDR` (đầu `scripts/chay-thu.sh` ghi đủ biến); script không bao giờ gỡ container PG đã chạy từ trước. Server của script nghe ở cổng do hệ điều hành chọn (`127.0.0.1:0`) và script chỉ gọi đúng địa chỉ mà server của nó đã báo, nên một server khác đang giữ cổng cũng không bị gọi nhầm (đặt `EAIS_ADDR` cố định mà cổng bận thì script dừng với "server thoát sớm").

**Từng bước** (cùng việc script làm). Các lệnh dùng `$W/eais` và `$W/data`, nên **đặt `W` trước** (không đặt thì thành `/eais` và `/data`):

```bash
W=$(mktemp -d)
docker compose -f deploy/docker-compose.yml up -d --wait
export EAIS_PG_ADMIN_DSN='postgres://postgres@127.0.0.1:25432/eais?sslmode=disable' EAIS_PG_DSN='postgres://eais_app@127.0.0.1:25432/eais?sslmode=disable'
go build -o $W/eais ./cmd/eais && $W/eais migrate
go run ./tenants/construction/win/gen/cmd/gen -world tenants/construction/win/world.yaml -out $W/data -profile functional -seed 42
$W/eais ingest -pack tenants/construction/pack -data $W/data       # in số quan sát / cách ly và mã lần nạp
$W/eais serve -dev -pack tenants/construction/pack -world tenants/construction/win/world.yaml -data $W/data -now world   # 127.0.0.1:8080
```

Gọi thử (token dev `dev-<tenant>-<principal>` suy từ `world.yaml`, là thông tin công khai):

```bash
H='Authorization: Bearer dev-abc-infra-u-director'
curl -s -X POST localhost:8080/v1/ask -H "$H" -H 'Content-Type: application/json' \
  -d '{"text":"Hôm qua máy 03 được cấp bao nhiêu dầu?","conversation_id":"c1","message_id":"m1"}'   # có trace_id
curl -s "localhost:8080/v1/trace/<trace_id>?format=table" -H "$H"        # bảng vết S0→S10, đánh dấu bước lệch
$W/eais trace -tenant abc-infra <trace_id>                                # cùng bảng, từ CLI
$W/eais trace diff -tenant abc-infra <id1> <id2>                          # bước ĐẦU TIÊN hai lượt cho kết quả khác nhau
$W/eais quarantine -tenant abc-infra <mã lần nạp>                         # bản ghi cách ly kèm lý do
```

| Điểm cuối | Việc | Ai được gọi |
|---|---|---|
| `GET /healthz` | 200, hoặc 503 khi PG không trả lời | không cần token |
| `POST /v1/ask` | thân `{"text","conversation_id","message_id"}`; trả câu trả lời + `trace_id` | mọi principal có token |
| `GET /v1/trace/{id}` | vết một lượt (JSON, hoặc `?format=table`) | chính người đã hỏi, hoặc principal toàn quyền dữ liệu; còn lại 404 |
| `POST /v1/ingest` | nạp lại thư mục dữ liệu đã cấu hình của tenant của principal; thân phải rỗng hoặc `{}` | principal toàn quyền dữ liệu |

- **Tenant, người hỏi, vai chỉ đến từ token**, không bao giờ từ thân: thân có trường `tenant`, `principal`, vai hay bất kỳ trường lạ nào bị từ chối 400. Tenant khác đọc vết ⇒ 404, y hệt "không có".
- **Toàn quyền dữ liệu** tính theo `policy.yaml` của pack, không nêu tên vai: có một vai phạm vi `all` nắm **mọi** capability mà policy định nghĩa. Với pack hiện tại chỉ `director` (accountant có phạm vi `all` nhưng không có `documents.read`).
- **Mọi lượt đã xác thực để lại vết bền** (kể cả lượt lỗi, lượt bị chính sách chặn, thân sai), gồm usage (extractive: 0 token). Lượt không lưu được vết thì không giao câu trả lời. Lượt chưa xác thực được tenant (401) không có dòng vết theo tenant; chúng chỉ được đếm trong log tổng kết khi tắt server.
- Mã lần nạp là băm của (dữ liệu, pack, mã): nạp lại cùng dữ liệu cho cùng mã và không nhân đôi bản ghi cách ly.
- **Phiên bản mã** (`code_version` của vết, và phần "mã" của mã lần nạp) là mã git của cây đã dựng binary: `scripts/chay-thu.sh` nhúng nó bằng `-ldflags` (thêm `+dirty` khi cây có sửa chưa commit; CI đặt `EAIS_CODE_VERSION` = commit đang kiểm) rồi đối chiếu với phiên bản mà server báo lúc khởi động. `go build` tay trong một clone thường tự ghi mã git vào binary; trong git worktree liên kết hay cây không có `.git` thì không có, và khi đó ứng dụng dùng băm sha256 của chính binary với tiền tố `bin-sha256:` — không bao giờ ghi `(devel)`. Hai bản sửa khác nhau chưa commit trên cùng một commit vẫn cùng phiên bản `+dirty`.

**Giới hạn đã biết** (ghi thẳng):
- `serve` chỉ chạy được với `-dev` (token công khai, chỉ nghe loopback, chặn `Host` không phải loopback). Chưa có cơ chế xác thực cho môi trường thật — đang chờ quyết định.
- Phiên bản dữ liệu ghi vào vết được tính từ thư mục dữ liệu lúc server khởi động và sau mỗi `POST /v1/ingest`, không đọc ngược từ PG. **Hậu quả: vết có thể mang một phiên bản dữ liệu CHƯA TỪNG được nạp vào PG, và không kiểm tra nào phát hiện** — thư mục đổi sau lần nạp, hay `serve` trỏ nhầm thư mục khác (đã tái hiện: thêm một dòng trống vào một tệp của bản sao thư mục rồi chạy server trên bản sao khi PG giữ lần nạp gốc ⇒ vết khai phiên bản của bản sao). Nạp bằng CLI (`eais ingest`) khi server đang chạy: các lần nạp cùng tenant xếp hàng nhau ở kho (khoá tư vấn theo tenant) nên không giẫm nhau, nhưng server không biết dữ liệu đã đổi ⇒ vết sau đó mang phiên bản dữ liệu cũ; khởi động lại server, hoặc nạp bằng `POST /v1/ingest`. Còn một cửa sổ cực ngắn giữa lúc lần nạp ghi xong vào kho và lúc server cập nhật phiên bản: lượt hỏi rơi vào đó đọc dữ liệu mới nhưng ghi phiên bản cũ. Chữa dứt điểm cần ghi phiên bản đã nạp vào PG cùng giao dịch nạp rồi đọc lại (một bảng nhỏ ⇒ migration mới; **chưa làm**); trong lúc chờ, `eais ingest` và `eais serve` phải dùng CÙNG một thư mục dữ liệu và server phải được khởi động lại sau mỗi lần nạp ngoài server.
- Lượt mà `Handle` trả lỗi (kể cả bị chính sách chặn, vd tenant lạ) chỉ để lại MỘT bước vết `S?` đánh dấu TRỐNG, không chỉ được tầng lỗi, vì `Handle` trả kết quả rỗng khi lỗi (`core/pipeline` ngoài phạm vi của ứng dụng). Chỉ tiêu "người chỉ đọc vết mà chỉ đúng tầng sai đầu tiên ở ≥ 90% ca đỏ" **chưa đo**, và với lượt lỗi cơ chế hiện tại không cho đạt: cần `Handle` trả kết quả một phần khi lỗi.
- Chưa có vết các bước nạp (`L1.*`) và chưa có chính sách dọn / lưu giữ vết (vai ứng dụng không có quyền xoá).
- CI ở `.github/workflows/ci.yml` chưa chạy trên GitHub (xem ghi chú đầu tệp).
