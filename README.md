# EACL — Enterprise AI Control Layer

**Controlled AI Integration for Enterprise Software.**

EACL is a reference architecture and executable control layer for integrating probabilistic AI capabilities into existing enterprise applications while those applications retain authority over business truth, permissions, state, transactions and workflows.

Phần mềm tất định giữ quyền quyết định về danh tính, tenant, quyền hạn, sự thật nghiệp vụ, trạng thái, giao dịch, chính sách, kiểm và thực thi. AI chỉ làm phần bất định: diễn giải, phân loại, trích xuất, tổng hợp, đề xuất. Model là một năng lực bên trong hệ, không phải nguồn sự thật.

```text
Business Reality → Trusted Context → Useful Intelligence → Controlled Action → Verified Outcome → Measured Value
```

**Trạng thái (21/10/2026):**
- Bài chứng minh đầu tiên là **Fuel WIN**: đối soát nhiên liệu cho tenant xây dựng, đạt khi qua đủ 7 cổng G1–G7 ([`docs/04-core-mvp.md`](docs/04-core-mvp.md) §6). **H1** là bộ ca giữ lại độc lập: do người không đọc mã soạn, đóng băng trước, chỉ chạy một lần (ADR-0006).
- Fuel WIN: **0/7 cổng** trên H1 (104 ca): 89/104 ca đạt; G1–G5 chưa đạt, G6' một phần, G7 bị chặn (chưa có khoá model, bộ chấm v2 chưa có).
- Câu xác nhận ("… đúng không?") còn phán quyết sai trên 15/15 câu thăm dò đối kháng. Đã sửa ngày 21/10 bằng cổng phán quyết fail-closed ([ADR-0009](docs/adr/0009-cong-phan-quyet-cau-xac-nhan.md)): chỉ mở đầu bằng "Đúng." / "Không." khi claim đã giải đủ chiều; 15 câu thăm dò còn 0 phán quyết sai. Bản sửa vào mã SAU khi bộ giữ lại mới H3 đóng băng (`259d38d`); H3 chưa chạy.
- Chi tiết: [`docs/STATUS.md`](docs/STATUS.md), báo cáo mới nhất ở [`docs/bao-cao/`](docs/bao-cao/).

| Đọc gì | Ở đâu |
|---|---|
| **Câu đang chờ chủ dự án quyết** | [`docs/CHO-QUYET-DINH.md`](docs/CHO-QUYET-DINH.md) |
| G7: hợp đồng benchmark, chi phí, checkpoint | [`docs/g7/`](docs/g7/) |
| Kế hoạch EACL V3 (bản nháp, chờ duyệt) | [`docs/v3/`](docs/v3/) |
| Chạy lại mốc / làm tiếp trên máy khác | [`docs/ops/README.md`](docs/ops/README.md) |
| Hợp đồng vận hành (luật bắt buộc) | [`CLAUDE.md`](CLAUDE.md) |
| WIN MVP: 7 cổng, phạm vi | [`docs/04-core-mvp.md`](docs/04-core-mvp.md) |
| Roadmap V1/V2 (lịch sử) | [`docs/05-master-roadmap.md`](docs/05-master-roadmap.md) |
| Kiến trúc | [`docs/01-architecture.md`](docs/01-architecture.md) |
| Quyết định | [`docs/adr/`](docs/adr/) |
| Báo cáo ngày | [`docs/bao-cao/`](docs/bao-cao/) |
| Thay đổi | [`CHANGELOG.md`](CHANGELOG.md) |

Repo: `github.com/luannt2002/enterprise-ai-control-layer` (cũng là đường dẫn module Go).

Chạy nhanh (Go 1.26, Docker, `pdftotext`):

```bash
docker compose -f deploy/docker-compose.yml up -d --wait
export EACL_PG_ADMIN_DSN='postgres://postgres@127.0.0.1:25432/eacl?sslmode=disable'
export EACL_PG_DSN='postgres://eacl_app@127.0.0.1:25432/eacl?sslmode=disable'
go test ./... -count=1 && scripts/scale-smoke.sh
```

Mọi dữ liệu trong repo là dữ liệu giả lập; repo công khai không chứa dữ liệu thật của tenant nào.
