# EAIS — Enterprise AI Systems

Nền tảng AI doanh nghiệp đa tenant. Nó biến dữ liệu, tri thức, quy trình và công cụ theo ngành thành năng lực AI có quản trị, kiểm thử được và quan sát được, mà lõi dùng chung không bị buộc vào một ngành nào. Model là một năng lực bên trong hệ, không phải nguồn sự thật.

**Trạng thái (20/10/2026):**
- WIN MVP: 3/7 cổng đạt vô điều kiện (G1, G3, G4); G2, G5, G6 đạt có điều kiện, chờ quyết 3 kỳ vọng trọng tài (D-G7-11).
- G7 (so với baseline RAG bằng model thật) đang chờ khoá model.
- Chi tiết: [`docs/STATUS.md`](docs/STATUS.md).

| Đọc gì | Ở đâu |
|---|---|
| **Câu đang chờ chủ dự án quyết** | [`docs/CHO-QUYET-DINH.md`](docs/CHO-QUYET-DINH.md) |
| G7: hợp đồng benchmark, chi phí, checkpoint | [`docs/g7/`](docs/g7/) |
| Kế hoạch EACL V3 (bản nháp, chờ duyệt) | [`docs/v3/`](docs/v3/) |
| Chạy lại mốc / làm tiếp trên máy khác | [`docs/ops/README.md`](docs/ops/README.md) |
| Hợp đồng vận hành (luật bắt buộc) | [`CLAUDE.md`](CLAUDE.md) |
| WIN MVP: 7 cổng, phạm vi | [`docs/04-core-mvp.md`](docs/04-core-mvp.md) |
| Roadmap có điều kiện | [`docs/05-master-roadmap.md`](docs/05-master-roadmap.md) |
| Kiến trúc | [`docs/01-architecture.md`](docs/01-architecture.md) |
| Quyết định | [`docs/adr/`](docs/adr/) |
| Báo cáo ngày | [`docs/bao-cao/`](docs/bao-cao/) |
| Thay đổi | [`CHANGELOG.md`](CHANGELOG.md) |

Chạy nhanh (Go 1.26, Docker, `pdftotext`):

```bash
docker compose -f deploy/docker-compose.yml up -d --wait
export EAIS_PG_ADMIN_DSN='postgres://postgres@127.0.0.1:25432/eais?sslmode=disable'
export EAIS_PG_DSN='postgres://eais_app@127.0.0.1:25432/eais?sslmode=disable'
go test ./... -count=1 && scripts/scale-smoke.sh
```

Mọi dữ liệu trong repo là dữ liệu giả lập; repo công khai không chứa dữ liệu thật của tenant nào.
