# ADR-0013 — Định nghĩa sản phẩm vNext: EAIS — Enterprise AI Integration Reference Architecture

- **Ngày:** 24/10/2026.
- **Trạng thái:** **CHẤP NHẬN 24/10/2026** — GO của chủ dự án cho báo cáo audit vNext (nguyên văn ở `docs/CHO-QUYET-DINH.md` mục 0, dòng 24/10
  "vNext"): "sửa lại toàn bộ theo cái file md đó và code tách và tiếp tục làm theo md đó đi nhé tách đi chanega code đi nhé". Báo cáo đã
  khuyến nghị đúng định nghĩa này (câu Q2 = A, Q3 = A ở §16).
- **Thay:**
  - phần **định nghĩa sản phẩm** của D-RM-01 ("nền tảng multi-tenant…", `docs/ROADMAP-DAY-DU.md` §1). Phần còn lại của D-RM-01 (bảy mặt
    phẳng, luồng L0–L10, Phase 0–10, hai cổng) giữ làm bản đồ;
  - yêu cầu REQ-MND-016 (`docs/requirements/catalog.yaml`, câu "…multi-tenant Enterprise AI Systems Platform…" phải mở đầu README và
    tài liệu kiến trúc);
  - ràng buộc "chỉ QUERY, không mở COMMAND / WORKFLOW / EVENT / Agent / MCP / Python / framework chung trước khi đóng G7" của ADR-0001
    (đã bị D-RM-04 nới một phần ngày 22/10).
- **Liên quan:** báo cáo audit `docs/bao-cao/2026-10-24-vnext-audit/BAO-CAO.md` (§07 đổi tên, §11 phương án, §14 lát, §16 câu hỏi);
  ADR-0015 (ranh giới module); ADR-0016 (chủ sở hữu ACL tài liệu); `docs/history/NAME-EVOLUTION.md` (các tên trước).

## Bối cảnh (đo ở mốc `3f1bf7e`, nguồn: báo cáo audit)

- Định nghĩa đang ghi là "nền tảng multi-tenant". Thực tế: cả repo có **một** pack (`git ls-files | grep 'pack.yaml$'` → chỉ
  `tenants/construction/pack/pack.yaml`); `tenants/sports` chỉ có bộ ca; Fuel WIN **0/7** cổng trên bộ giữ lại độc lập; **0** dòng mã chế độ
  A (không có client HTTP tới app của khách, báo cáo §06.2); **0** người dùng ngoài.
- Phần chín nhất của repo là **lõi kiểm soát**: tenant từ danh tính, quyền đóng mặc định, RLS FORCE, kiểm đầu ra, vết bền, lệnh có đọc lại
  (UNKNOWN thật), bộ giữ lại băm trước. Các kiểu của lõi đã có sẵn (báo cáo §11.4), không phải phát minh.
- Tên hướng sản phẩm đã đổi bốn lần trong bốn ngày (`docs/history/NAME-EVOLUTION.md`). Đổi đường dẫn module lần nữa chạm 227 tệp `.go`
  / 578 dòng mang chuỗi import của module (576 dòng import thật, 2 dòng là chuỗi trong test) và mọi URL clone. Đo 24/10 trên `3f1bf7e`:
  `git grep -l '"github.com/luannt2002/enterprise-ai-control-layer' 3f1bf7e -- '*.go' | wc -l` → 227;
  `git grep -h '"github.com/luannt2002/enterprise-ai-control-layer' 3f1bf7e -- '*.go' | wc -l` → 578 (bỏ dấu nháy mở: 580).
  Báo cáo §07.1 ghi 583 là số dòng chứa tên repo ở mọi dạng, không phải số dòng import.

## Quyết định

1. **Sản phẩm vNext là một kiến trúc tham chiếu chạy được**, gồm:
   - **lõi kiểm soát** (kernel): danh tính → tenant, quyền / phạm vi, thực thi có kiểm, kết quả có trạng thái (DONE / FAILED / UNKNOWN),
     bằng chứng, vết từng bước;
   - **module tuỳ chọn**: QUERY (động cơ S0–S10) và RAG (tri thức có phiên bản, lọc quyền trước khi chấm); COMMAND (C0–C7) đi cùng lõi;
   - **ví dụ / case study**: cách áp lõi + module vào một việc thật. **Fuel** (pack `tenants/construction` + `win/*`) là case study đầu
     tiên, không phải sản phẩm.
2. **Tên hiển thị:** **EAIS — Enterprise AI Integration Reference Architecture**. Chỉ đổi ở các dòng tên hiển thị hiện hành (README,
   `docs/01-architecture.md`, `docs/CONVENTIONS.md` §0, `docs/ROADMAP-DAY-DU.md` dòng tên, `docs/ops/BO-NHO.md`); không replace-all.
3. **Giữ nguyên:** viết tắt **EAIS** và mọi định danh chạy — biến môi trường `EAIS_*`, GUC `eais.tenant_id`, database `eais` /
   `eais_giu_lai`, vai `eais_app`, container `eais-pg`, binary và thư mục `cmd/eais`, đường dẫn module
   `github.com/luannt2002/enterprise-ai-control-layer` (tới phát hành đầu tiên; đổi chỉ bằng ADR riêng, cùng commit đổi `gitLink` của
   `danh_tinh_test.go`). Migration đã áp và tạo phẩm đã băm không sửa.
4. **Thứ tự việc:** các lát L0–L9 của báo cáo audit §14 (phương án B′: cổng trước, tách tại chỗ, dời thư mục sau cùng). Mở một module /
   ví dụ mới vẫn qua hai cổng của `docs/ROADMAP-DAY-DU.md` §4 (cổng QuânIT + cổng gốc). G7 là cổng giá trị **của case study Fuel**, không
   còn là điều kiện để mở COMMAND hay module khác. Luật "không thêm tính năng chỉ để cải thiện benchmark" của ADR-0001 giữ nguyên.
5. **Lịch sử không sửa:** ADR-0001, D-RM-01, REQ-MND-016 giữ lời cũ; chỉ thêm ghi chú trạng thái / SUPERSEDED-BY trỏ về ADR này.

## Hệ quả

- `docs/requirements/catalog.yaml` REQ-MND-016 mang ghi chú SUPERSEDED-BY ADR-0013; câu cũ không còn là yêu cầu.
- `CLAUDE.md` §9 (tên, câu sứ mệnh), danh sách "Reference tenants" (Sports chưa có pack) và §25 (bản đồ repo) còn lời cũ. `CLAUDE.md` là
  hợp đồng của chủ dự án: agent chỉ đề xuất diff, **chủ dự án sửa** (Q13 của báo cáo).
- Cổng danh tính (`danh_tinh_test.go`) không ghim tên hiển thị, nên đổi tên không làm cổng đỏ; tài liệu lịch sử nêu tên trước D-RM-02 vẫn
  phải có mục `legacyAllowed` với hạn mức đúng bằng số lần.
- Chuỗi usage của binary (`cmd/eais/main.go`) còn tên hiển thị cũ trong ngoặc: đổi ở lát chạm mã, không ở lát tài liệu.

## Phương án đã cân nhắc

- **Giữ "Multi-tenant Platform" (D-RM-01 nguyên vẹn):** bị loại — một miền, 0 mã chế độ A, 0 người dùng ngoài; lời hứa "nền tảng" chưa có
  bằng chứng (P7: phức tạp phải có bằng chứng).
- **Đổi luôn đường dẫn module:** bị loại tới phát hành đầu tiên — vỡ import / `go get` của fork, đổi URL clone, phải đổi `gitLink`; không
  đổi được giá trị nào cho người dùng.
- **Đặt viết tắt mới:** bị loại — mọi định danh chạy, migration và bộ giữ lại đã băm đang neo vào chữ EAIS / eais.
