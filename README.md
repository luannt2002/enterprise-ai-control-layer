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

Chạy nhanh (Go 1.26, Docker, `pdftotext`, `flock`):

```bash
docker compose -f deploy/docker-compose.yml up -d --wait
export EAIS_PG_ADMIN_DSN='postgres://postgres@127.0.0.1:25432/eais?sslmode=disable'
export EAIS_PG_DSN='postgres://eais_app@127.0.0.1:25432/eais?sslmode=disable'
go test ./... -count=1 && scripts/scale-smoke.sh
```

Trước khi chạy **Một lệnh** (mục "Chạy thử trong 5 phút") trong cùng shell: `unset EAIS_PG_ADMIN_DSN EAIS_PG_DSN` — đã `export` cả hai biến này thì `scripts/chay-thu.sh` ghi thẳng vào PG đó mà không hỏi.

Mọi dữ liệu trong repo là dữ liệu giả lập; repo công khai không chứa dữ liệu thật của tenant nào.

## Chạy thử trong 5 phút

Ứng dụng `eais` (một binary: HTTP + CLI) chạy trên PostgreSQL với dữ liệu giả lập seed 42. Mặc định **không gọi model**: luồng hỏi dùng model tất định (extractive) và ứng dụng không có cờ hay đường nào tới model trả tiền. Cần: Go (theo `go.mod`), Docker, `curl`, `python3`, `pdftotext` (`poppler-utils`), `flock` (util-linux; tuỳ chọn cho script — thiếu thì script chạy không có khoá dự án và in cảnh báo —, nhưng bắt buộc cho bộ test `go test ./cmd/eais/`; macOS: `brew install flock`).

**Một lệnh** (dựng PG → migrate → sinh seed 42 → nạp → chạy server nền → smoke E2E bằng curl → dừng server):

```bash
git clone https://github.com/luannt2002/enterprise-ai-control-layer && cd enterprise-ai-control-layer
scripts/chay-thu.sh          # EAIS_PG_DOWN=1 để gỡ PG khi xong
```

> **Cảnh báo: script GHI vào PG mà nó dùng** (migrate rồi nạp THAY toàn bộ dữ liệu của tenant trong pack construction). Với giá trị mặc định nó chỉ chạy trên PG do chính nó dựng (container `eais-pg`, cổng `127.0.0.1:25432`) và **từ chối** — thoát 2, chưa ghi gì — nếu container đó đang chạy, cổng đó đã có tiến trình nghe, hoặc dự án compose (`eais`) đã có container nào đang chạy hay tạm dừng, kể cả PG còn sót lại của lần chạy trước. Đã có PG khác đang chạy: đổi **cả ba** `EAIS_PG_CONTAINER`, `EAIS_PG_PORT`, `EAIS_COMPOSE_PROJECT` để script dựng PG riêng (đổi tên và cổng mà giữ dự án thì `docker compose up` cùng dự án sẽ tạo lại container đang chạy và dữ liệu của nó mất, nên script từ chối); chỉ khi chắc PG đang chạy là của bạn và ghi đè được mới đặt `EAIS_REUSE_PG=1`. Ngoại lệ: đã `export` cả `EAIS_PG_ADMIN_DSN` và `EAIS_PG_DSN` (như ở mục "Chạy nhanh" trên) thì script dùng đúng PG đó, không dựng gì và không hỏi — bỏ chúng khỏi shell (`unset`) nếu không muốn script ghi vào PG ấy.

Thời gian đo ngày 22/10/2026 từ một clone sạch (16 vCPU, ảnh PG đã kéo sẵn, module Go có sẵn; thời gian kéo ảnh lần đầu **chưa đo**): **9,6 giây** với bộ nhớ đệm build rỗng (PG do chính script dựng), **3,9 giây** với bộ nhớ đệm ấm. Thành công khi in `smoke E2E: 21 đạt, 0 hỏng`. Đổi tên container / cổng / địa chỉ nghe bằng `EAIS_PG_CONTAINER`, `EAIS_PG_PORT`, `EAIS_COMPOSE_PROJECT`, `EAIS_ADDR` (đầu `scripts/chay-thu.sh` ghi đủ biến); script không gỡ container PG đang chạy hay tạm dừng (nó từ chối); container đã **dừng hẳn** của cùng dự án thì không chặn: cùng tên và cổng thì `docker compose up` khởi động lại chính container đó (dữ liệu PG trên tmpfs đã mất lúc dừng, mã thoát cũ bị ghi đè, log cũ còn); đổi tên hoặc cổng mà giữ dự án thì compose xoá container cũ rồi tạo mới và log cũ mất (cần `docker logs` của PG đã chết thì lấy trước khi chạy lại, hoặc đổi cả ba biến). Mỗi dự án compose chỉ một lần chạy tại một lúc (mục "Khoá theo dự án" ngay dưới). Server của script nghe ở cổng do hệ điều hành chọn (`127.0.0.1:0`) và script chỉ gọi đúng địa chỉ mà server của nó đã báo, nên một server khác đang giữ cổng cũng không bị gọi nhầm (đặt `EAIS_ADDR` cố định mà cổng bận thì script dừng với "server thoát sớm").

**Khoá theo dự án** (`flock`):
- Hai lần chạy đồng thời cùng `EAIS_COMPOSE_PROJECT` thì lần sau bị **từ chối** (thoát 2, chưa làm gì). Khoá giữ suốt đời script, kể cả bước gỡ PG (`docker compose down`), và vẫn áp dụng khi đặt `EAIS_REUSE_PG=1`. Dự án khác chạy song song bình thường; đường dùng PG có sẵn (cả hai DSN) không chạm compose nên không khoá.
- Tệp khoá là `$EAIS_LOCK_DIR/eais-chay-thu-<dự án>.lock` (0 byte, script không bao giờ xoá). `EAIS_LOCK_DIR` mặc định là `/tmp` viết cứng, **không theo `TMPDIR`**: hai lần chạy cùng dự án từ hai nơi có `TMPDIR` khác nhau (shell thường, IDE, sandbox dùng chung `/tmp`, cron) vẫn khoá cùng một tệp, miễn là cùng thấy một `/tmp` (xem gạch cuối). Đừng đổi `EAIS_LOCK_DIR` giữa hai lần chạy cùng dự án.
- Script bị `SIGKILL` thì khoá nhả khi tiến trình con của bước đang chạy thoát (đã đo với smoke; `docker compose up`, `go build` cùng cơ chế nhưng chưa đo); server nền đã đóng fd khoá nên server mồ côi không giữ khoá hộ (có test và đột biến canh dòng này).
- Khoá chỉ che các lần chạy **cùng thấy một tệp khoá**. Chưa che: người dùng unix khác (không mở được tệp do người kia tạo thì script cảnh báo rồi chạy không khoá; chưa thử với hai tài khoản) và namespace `/tmp` riêng (container, sandbox) mà dùng chung một daemon docker. Che cả hai cần khoá phía daemon, chưa làm.
- Máy không có `flock` (macOS mặc định: `brew install flock`), `EAIS_LOCK_DIR` không ghi được, hay `flock` báo mã lạ (khác 0 và 1, vd hệ tệp không hỗ trợ khoá): script in cảnh báo và đi tiếp **không có khoá**, nên đừng chạy hai lần cùng dự án cùng lúc trên máy đó.

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
| `POST /v1/commands/<tool>` | một lượt lệnh của tool khai trong `tools.yaml` của pack (Phase 1, `docs/viec/1-hanh-dong/HOP-DONG.md` §2.6); trả `CommandResult` + `trace_id` | principal có capability của tool theo `policy.yaml` và thực thể trong phạm vi; còn lại 403, cùng một lời cho thực thể ngoài phạm vi và thực thể không có |

- **Tenant, người hỏi, vai chỉ đến từ token**, không bao giờ từ thân: thân có trường `tenant`, `principal`, vai hay bất kỳ trường lạ nào bị từ chối 400. Tenant khác đọc vết ⇒ 404, y hệt "không có".
- **Toàn quyền dữ liệu** tính theo `policy.yaml` của pack, không nêu tên vai: có một vai phạm vi `all` nắm **mọi** capability mà policy định nghĩa. Với pack hiện tại chỉ `director` (accountant có phạm vi `all` nhưng không có `documents.read`).
- **Mọi lượt đã xác thực để lại vết bền** (kể cả lượt lỗi, lượt bị chính sách chặn, thân sai), gồm usage (extractive: 0 token). Lượt hỏi không lưu được vết thì không giao câu trả lời. Route lệnh làm NGƯỢC LẠI, có chủ ý (HOP-DONG §7.5 "Mất vết"): lệnh có thể đã có tác dụng (phiếu đã commit và đã đọc lại) nên kết quả vẫn được trả, chỉ để `trace_id` rỗng, tăng bộ đếm `trace_save_failed` và ghi log lỗi; lượt lệnh không có usage (không gọi model). Lượt chưa xác thực được tenant (401) không có dòng vết theo tenant; chúng chỉ được đếm trong log tổng kết khi tắt server.
- Mã lần nạp là băm của (dữ liệu, pack, mã): nạp lại cùng dữ liệu cho cùng mã và không nhân đôi bản ghi cách ly. Mỗi lần nạp thành công thêm một dòng vào sổ nạp `load_records` (kể cả nạp lại cùng dữ liệu): dòng mới nhất là phiên bản mà vết khai.
- **Phiên bản mã** (`code_version` của vết, và phần "mã" của mã lần nạp) là mã git của cây đã dựng binary: `scripts/chay-thu.sh` nhúng nó bằng `-ldflags` (thêm `+dirty` khi cây có sửa chưa commit; CI đặt `EAIS_CODE_VERSION` = commit đang kiểm) rồi đối chiếu với phiên bản mà server báo lúc khởi động. `go build` tay trong một clone thường tự ghi mã git vào binary; trong git worktree liên kết hay cây không có `.git` thì không có, và khi đó ứng dụng dùng băm sha256 của chính binary với tiền tố `bin-sha256:` — không bao giờ ghi `(devel)`. Băm binary là dấu của BINARY, không phải của mã nguồn: chỉ so được giữa các vết do cùng một binary ghi (cùng commit dựng ở hai thư mục cho hai băm khác nhau, vì Go nhúng đường dẫn nguồn khi không có `-trimpath`). Hai bản sửa khác nhau chưa commit trên cùng một commit vẫn cùng phiên bản `+dirty`.

**Giới hạn đã biết** (ghi thẳng):
- `serve` chỉ chạy được với `-dev` (token công khai, chỉ nghe loopback, chặn `Host` không phải loopback). Chưa có cơ chế xác thực cho môi trường thật — đang chờ quyết định.
- Phiên bản dữ liệu ghi vào vết (`traces.Versions.Data`, cột `decision_records.data_version`) là phiên bản của lần nạp **đã commit vào PG** của đúng tenant, đọc từ PG (R2-01, migration 0006): `Load` ghi một dòng vào sổ nạp `load_records` (mã lần nạp, băm dữ liệu) trong **cùng giao dịch** thay dữ liệu của tenant, và server đọc dòng mới nhất của tenant ở đầu **mỗi** lượt hỏi và lượt lệnh. Nên vết không còn khai một phiên bản chưa từng được nạp khi thư mục dữ liệu đổi sau lần nạp, khi `serve` trỏ nhầm thư mục khác (ca README từng tái hiện: thêm một dòng trống vào bản sao thư mục), hay khi PG vừa được nạp bằng `eais ingest` lúc server đang chạy — không cần khởi động lại server (ba ca đều có test đỏ trên mã cũ: `cmd/eais/data_version_test.go`). Tenant chưa có lần nạp nào được ghi nhận, hoặc lần gần nhất không kèm phiên bản (đường `Load` cũ của bộ chạy `winrun` và test), thì vết khai `(chưa ghi nhận lần nạp)`, không bịa một băm — kể cả PG đã nạp từ trước migration 0006, cho tới lần nạp kế tiếp. Không đọc được sổ nạp: lượt hỏi lỗi (503; có vết khai phiên bản rỗng, không có câu trả lời); lượt lệnh vẫn chạy (như "Mất vết" ở trên) và vết khai rỗng; `serve` không khởi động. Thư mục `-data` chỉ còn dùng cho `eais ingest` và `POST /v1/ingest` (và kiểm lúc khởi động rằng nó dùng được): `serve` trỏ nhầm thư mục không làm vết khai sai, nhưng `POST /v1/ingest` sẽ nạp đúng thư mục nhầm đó vào PG. Kho bộ nhớ (chỉ bản dựng thử trong test) nạp thẳng từ thư mục nên vết vẫn khai băm thư mục như trước.
  Giá đo được: một lần đọc ≈ 70 µs (`BenchmarkLatestLoad`, đo ngày 23/10/2026, 8 lượt chạy × 3000 lần, PG 17 cùng máy: 67–71 µs ở sổ 1 dòng, 68–74 µs ở sổ 5000 dòng của một tenant; một lượt chạy đầu sau khi tạo bảng đo 1062 µs, không tái hiện ở 7 lượt sau, nguyên nhân chưa rõ). Một lượt hỏi đầu-cuối qua HTTP ≈ 7–8 ms trên máy dùng chung này, nhiễu giữa các lượt đo lớn hơn 70 µs nên chênh lệch của cả lượt **chưa tách được**.
  **Còn lại (giới hạn):** phiên bản được đọc **một lần ở đầu lượt**; một lần nạp commit **giữa** lượt (cửa sổ dài bằng một lượt, vài mili giây) làm lượt đọc dữ liệu mới mà vết vẫn khai phiên bản cũ — `TestDataVersionInTurnWindowKnownLimit` ghim đúng điều đó, và sẽ đỏ khi ai khép được cửa sổ (sửa test cùng dòng này). Suy luận, chưa làm, chưa đo: khép được mà không đổi `core/pipeline` bằng đọc lại phiên bản sau lượt rồi chạy lại lượt hỏi nếu nó đổi (thêm một truy vấn mỗi lượt; lượt lệnh không chạy lại được vì đã ghi phiếu), hoặc bằng đọc dữ liệu và phiên bản trong một snapshot của cả lượt (giữ một kết nối PG suốt lượt, kể cả lúc chờ model); chưa làm vì chưa có ca thật đòi (CLAUDE.md, P7).
  **Nâng cấp từ bản chưa có sổ nạp (trước việc 5e):** sau khi áp migration 0006, MỌI bên nạp dữ liệu vào PG phải chạy binary có sổ nạp. Binary cũ không biết bảng `load_records`: nó vẫn thay dữ liệu của tenant nhưng không ghi dòng sổ. Hậu quả đo được (tái hiện ngày 23/10/2026 bằng hai binary, trước và sau 5e, trên một PG riêng): khi sổ đã có ít nhất một dòng do binary mới ghi, một lần `eais ingest` bằng binary cũ với bộ dữ liệu khác để sổ nguyên một dòng mà dữ liệu sống đã đổi, và server mới vẫn khai trong vết phiên bản của lần nạp TRƯỚC — đúng loại lỗi mà sổ nạp sinh ra để chặn, và không kiểm tra nào phát hiện. (DB đã áp 0006 mà chưa có dòng sổ nào thì an toàn: vết khai `(chưa ghi nhận lần nạp)`.) Trình tự nâng cấp: (1) dừng MỌI bên nạp bản cũ — cron, `eais ingest`, server cũ (`POST /v1/ingest`) — TRƯỚC khi migrate; (2) `eais migrate` (chạy lại được); (3) chạy `serve` và mọi bên nạp bằng binary mới (bản mới từ chối mở kho khi chưa áp 0006 và nói rõ cần `eais migrate`: `TestReadySoNap`); (4) nạp lại MỘT lần (`eais ingest`) để sổ có dòng đầu tiên — tới lúc đó vết khai `(chưa ghi nhận lần nạp)`. Bước (1) không có gì ép: nhiều bản chạy dùng chung một PG dev là chỗ dễ vướng.
- Lượt mà `Handle` trả lỗi (kể cả bị chính sách chặn, vd tenant lạ) chỉ để lại MỘT bước vết `S?` đánh dấu TRỐNG, không chỉ được tầng lỗi, vì `Handle` trả kết quả rỗng khi lỗi (`core/pipeline` ngoài phạm vi của ứng dụng). Chỉ tiêu "người chỉ đọc vết mà chỉ đúng tầng sai đầu tiên ở ≥ 90% ca đỏ" **chưa đo**, và với lượt lỗi cơ chế hiện tại không cho đạt: cần `Handle` trả kết quả một phần khi lỗi.
- Chưa có vết các bước nạp (`L1.*`) và chưa có chính sách dọn / lưu giữ vết và sổ nạp `load_records` (một dòng mỗi lần nạp; vai ứng dụng không có quyền xoá).
- CI ở `.github/workflows/ci.yml` chưa chạy trên GitHub (xem ghi chú đầu tệp).
