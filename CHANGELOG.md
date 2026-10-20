# Changelog

Mọi thay đổi đáng kể của EAIS được ghi ở đây, theo tinh thần [Keep a Changelog](https://keepachangelog.com/).
Dự án chưa phát hành phiên bản nào. Commit đầu tiên: 20/10/2026.
Trạng thái chi tiết và bằng chứng nằm ở `docs/STATUS.md`.

## [Chưa phát hành]

### Thêm (20/10/2026) — WIN MVP: G1–G6 có bằng chứng đạt, G7 chặn ở khoá model

Chi tiết và lệnh chứng minh: `docs/STATUS.md`; các quyết định tự chốt: `docs/adr/0002-win-quyet-dinh-trien-khai.md`.

**Dữ liệu và bộ ca WIN**
- `tenants/construction/win/`:
  - thế giới đóng băng (`world.yaml`) + `CONTRACT.md`;
  - bộ sinh dữ liệu có seed: hồ sơ functional ~2,7 nghìn bản ghi, smoke ~360 nghìn; manifest + biến thể metamorphic;
  - 117 ca do phiên không đọc mã soạn (`cases-independent.yaml`);
  - `winrun`, `winreport`; ratchet `passing.txt` 117/117.
- `data/ingestion`: nạp CSV / XLSX / JSON / JSONL / PDF văn bản (`pdftotext`) theo `pack/sources.yaml`; mọi bản ghi không nạp được vào cách ly kèm lý do.

**Luồng hỏi**
- Kết quả có kiểu (`Outcome`).
- Câu xác nhận so tuyệt đối với nguồn có thẩm quyền; câu nhiều mệnh đề.
- Vật tư mặc định (D7); ý định yếu; cụm bỏ qua.
- So sánh hai nghĩa (`Catalog.Compare`).
- Route `RAG` / `HYBRID_RETRIEVAL` với gateway model theo cấu hình (model giả tất định có đếm lời gọi).
- Bản ghi quyết định + `Replay` (G6).
- `intelligence/rag`: chỉ mục tri thức theo tenant + ACL tài liệu, so khớp theo luật dấu (`normalization.ToneKey`).
- `reliability/validation`:
  - `BindClaims` + `CountNumbers`: ràng buộc claim 6 chiều, bộ đếm số độc lập;
  - `Bound`: claim gắn bằng chứng;
  - `ForbidDimensions` + `ForbidUnitWords`: lượng dầu không lấy từ tài liệu.

**PostgreSQL**
- `deploy/docker-compose.yml`: PG 17.11 ghim digest, chỉ loopback, tmpfs.
- `db/migrations/0001_win.sql`: RLS FORCE theo GUC `eais.tenant_id`, vai `eais_app`.
- `platform/pgstore`: Migrate / Load / Facts / Entities / Retrieve.
- Bộ ca WIN chạy trên PG ra từng câu giống đường bộ nhớ.

**G7 (bộ khung; chạy thật cần khoá model)**
- `evals/baseline`: baseline RAG theo D6.
- `evals/g7.go`: luật D5, chấm theo nội dung, bộ đếm độc lập rò rỉ / số không căn cứ.
- `tenants/construction/win/g7run`: lệnh chạy G7.
- `intelligence/models`: client API kiểu Messages (`Messages`), `New` + `ConfigFromEnv`.

**Vận hành và đồng bộ** (phiên dừng 10:33 theo lệnh chủ dự án)
- `docs/ops/`: bàn giao công khai, chạy lại mốc ở máy khác (`README.md`), luồng tự chạy (`LUONG-TU-CHAY.md`), nhật ký ngày đã làm sạch (`NHAT-KY-2026-10-20.md`), bộ nhớ dự án (`BO-NHO.md`).
- `docs/bao-cao/2026-10-20-win-mvp.md`: báo cáo ngày.
- `scripts/dot-bien/`: 69 đột biến có danh mục, chạy lại được.
- `docs/02-migration-audit.md`: thay đường dẫn monorepo nội bộ của hệ cũ bằng "hệ thống cũ" (repo public).

**Cổng và công cụ**
- `architecture_test.go`: kiểu `*Request` không được mang tenant.
- `scripts/scale-smoke.sh`: scale smoke một lệnh.
- `scripts/mutate.sh`: nhận `PKGS`.

### Sửa (20/10/2026)

- `Policy.Allows("")` từng **mở mặc định**. Nay đóng mặc định, và pack nào có ý định chạm dữ liệu / model mà thiếu capability thì bị chặn lúc nạp.
- `sourceoftruth.Query` duyệt map nguồn nên thứ tự bằng chứng không tái lập được. Phát lại G6 bắt được.
- Bộ chuẩn hoá đọc "-150 L" thành 150, nên hiệu âm bị cổng S9 chặn khi giao.
- Truy xuất so khớp bỏ dấu: "bí" khớp "bị".
- Truy xuất cộng điểm theo thứ tự duyệt map, nên đoạn hoà điểm đổi chỗ giữa các lần hỏi. Nay cộng theo thứ tự từ cố định; hoà điểm thì xếp theo tài liệu rồi dòng.

### Thêm (19/10/2026)

**Tài liệu**

- `docs/01-architecture.md`: nguyên tắc P1–P13, ba lối vào (QUERY / COMMAND / EVENT), pipeline S0–S10 / C0–C7 / E0–E6, bốn tầng dữ liệu (quan sát → đo → fact → fact dẫn xuất), hợp đồng thành phần, pack, cổng capability, ranh giới Go ↔ Python.
- `docs/02-migration-audit.md`: đối chiếu chuyển đổi từng tệp, mốc đo, các pha 0–8 kèm cổng chứng minh.
- `docs/CONVENTIONS.md`: mã định danh, 12 route, nhóm và nhãn lỗi, định dạng ca kiểm thử, luật con số, luật dữ liệu cho repo public.
- `docs/03-proof-harness.md`: hướng PROVE (thang bằng chứng D0–D4, đếm độ phủ thay cho đếm số ca, luật tuyên bố, chuẩn hoá *nghĩa*, test metamorphic), Sao Bắc Đẩu là lát dọc "Đối soát nhiên liệu", bảng phân loại 29 điểm audit điểm mù.
- `docs/04-core-mvp.md`: **WIN MVP — Enterprise Truth & Routing Proof** (hướng đã chốt 19/10 21:01).
  - Nội dung: luận đề, 5 bất biến CORE, một bài toán Fuel Discrepancy Resolution, chỉ QUERY, tenant A + canary B, baseline RAG bắt buộc, 7 cổng G1–G7, danh sách KHÔNG làm, các lát W1–W7.
  - Thay bản đề xuất lúc 20:5x. Bản đó rộng hơn (lệnh, tác vụ duyệt, thể thao) và bị thay sau khi user thu hẹp phạm vi; 10 điểm còn hiệu lực từ lượt soát độc lập đã được sửa.
- `docs/05-master-roadmap.md`: bản đồ Phase 0–10 **có điều kiện** (không phải TODO list); luật KEEP/FIX/BUILD/DEFER/REMOVE; cổng "ai sở hữu?"; chỗ của các thứ đã có trên bản đồ.
- `docs/STATUS.md`: tiến độ, bằng chứng đã chạy, lỗ hổng đã đo, quan sát ngoài phạm vi, việc tiếp theo.
- `docs/requirements/catalog.yaml` + `catalog.md`: sổ 422 yêu cầu truy về 13 nguồn (chưa soát độc lập).
- `docs/tenants/**`, `tenants/*/evals/cases.yaml`: tài liệu và ca của hai tenant tham chiếu (100 ca cầu đường, 80 ca thể thao; chưa soát độc lập).

**Mã Go** (module `github.com/luannt2002/enterprise-ai-systems`, Go 1.26, toolchain go1.26.4)

- `core/contracts`: kiểu dùng chung và nhãn lỗi → mã HTTP.
- `core/decimal`: số thập phân không float.
- `core/tenant`: nạp và kiểm pack lúc khởi động.
- `core/pipeline`: lối vào QUERY S0–S10 cho các route `SQL_QUERY`, `CALCULATION`, `CLARIFICATION`, `REJECT`, `HUMAN_ESCALATION`.
- `data/normalization`, `data/entityresolution`, `data/sourceoftruth`: chuẩn hoá tiếng Việt, nhận dạng thực thể, nguồn sự thật theo thẩm quyền trong từng chỉ số.
- `intelligence/intent`, `intelligence/router`: ý định theo luật; route kèm capability và phạm vi dữ liệu.
- `reliability/validation`: S9 kiểm số và mã trong câu với bằng chứng.
- `observability/traces`: đo thời gian từng bước.
- `simulations/fixtures`: cửa nạp dữ liệu giả lập đã đóng băng (đổi đơn vị, kiểm miền giá trị).

**Tenant, test và công cụ**

- `tenants/construction`: pack 6 tệp, seed giả lập 8 thực thể + 12 fact, test đầu–cuối 18 ca, test metamorphic (câu gốc + 9 cách viết khác), ca nạp hỏng, ca sai tenant.
- `architecture_test.go`: cổng "mã dùng chung không mang literal tên tenant".
- `scripts/mutate.sh`: chạy một đột biến, khôi phục và kiểm sha256, tách riêng đột biến không biên dịch.
- `evals`: nạp + kiểm hợp đồng bộ ca theo CONVENTIONS §5 (23 khoá, id, tenant, loại lối vào, category, severity, route), có ca âm.

### Sửa (19/10/2026)

- **Luật dấu.** Khớp kiểu bỏ dấu chỉ khi cả câu gõ không dấu. Trước đây "bao" trong "bao nhiêu" khớp nhầm "báo".
- **Ý định.** Áp luật khớp dài nhất trên từ khoá của mọi ý định. Trước đây câu gõ không dấu "bao nhieu dau" bị hiểu thành "dầu đội báo" và trả sai chỉ số.

- **`docs/03-proof-harness.md` §5.** Ví dụ metamorphic cũ coi ERP / phiếu kho / tin nhắn là "cùng một fact", trái luật tách nghĩa. Đã sửa thành cùng một bản ghi của cùng một nguồn ở nhiều định dạng (phát hiện của soát độc lập).

### Bảo mật (19/10/2026)

- **S1 TENANT.** Principal thuộc tenant khác với runtime bị từ chối với lỗi `policy`, trước khi chạm vào pack hay dữ liệu.
