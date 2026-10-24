# ADR-0015 — Ranh giới module: lõi / module / adapter / ví dụ / case study, và luật "≥ 2 miền mới vào lõi"

- **Ngày:** 24/10/2026.
- **Trạng thái:** **CHẤP NHẬN 24/10/2026** — GO của chủ dự án cho báo cáo audit vNext, phương án **B′** (cổng trước, tách tại chỗ, dời thư
  mục sau cùng; `docs/CHO-QUYET-DINH.md` mục 0, dòng 24/10 "vNext"). B′ là khuyến nghị Q5 của báo cáo: chấp nhận theo GO chung 24/10 cho toàn bộ khuyến nghị của báo cáo; chủ dự án chưa trả lời riêng Q2 / Q3 / Q5. Cây thư mục ở mục 4
  là **ĐÍCH**, chưa dời gì.
- **Liên quan:** ADR-0013 (định nghĩa sản phẩm); ADR-0014 (cổng trước khi tách — lát L1); ADR-0016 (chủ sở hữu ACL tài liệu); báo cáo audit
  `docs/bao-cao/2026-10-24-vnext-audit/BAO-CAO.md` §04.3, §06, §11.2–§11.5, §13, §14; phát hiện F-004, F-005, F-011, F-029, F-030, F-031.

## Bối cảnh

- **`core/` hôm nay không phải một lớp.** Nó chứa gói lá (`core/contracts`, `core/decimal`) lẫn gói đỉnh. Đo lại 24/10:
  `go list -f '{{join .Imports "\n"}}' ./core/pipeline` cho 8 gói nội bộ ngoài `core/` (data/entityresolution, data/normalization,
  data/sourceoftruth, intelligence/intent, intelligence/models, intelligence/router, observability/traces, reliability/validation);
  `./core/tenant` import data/normalization, data/sourceoftruth, intelligence/intent, intelligence/router. Ở mức thư mục có chu trình
  core ↔ data / intelligence / observability / reliability (F-029).
- **RAG đã độc lập ở mức thư viện:** `go list -deps ./intelligence/rag` chỉ kéo `core/contracts`, `core/decimal`, `data/normalization`.
- **Chữ miền trong mã dùng chung:** 18 dòng mã (bỏ comment) mang từ của miền Fuel ngoài `tenants/` (báo cáo §04.3), ví dụ `narrow` trong
  `intelligence/router/router.go` chỉ thu hẹp khi kiểu thực thể là `"project"` (F-005), lược đồ `entities.json` gán cứng
  `projects` / `equipment` (F-004). Cổng kiến trúc hiện chỉ dò tên tenant nên các dòng này lọt (F-011).
- **Một pack** trong cả repo. Ước lượng (đọc mã, chưa đo bằng `git diff`): miền mới cùng hình dạng Fuel sửa 0 tệp lõi; phân cấp thực thể
  khác sửa ~6 tệp; fact không phải số ≥ 10 tệp; ngôn ngữ khác ≥ 8 tệp (báo cáo §06.3).

## Quyết định

1. **Năm lớp.**

   | Lớp | Là gì | Luật import |
   |---|---|---|
   | **Lõi** (kernel) | hợp đồng và kiểm soát dùng cho MỌI module: danh tính → tenant, quyền / phạm vi, kết quả có trạng thái, bằng chứng, vết | không import module, adapter, ví dụ, case study |
   | **Module** | một năng lực tuỳ chọn có hợp đồng riêng: QUERY (S0–S10), RAG, COMMAND (C0–C7), reports (Phase 0a) | import lõi; không import ví dụ / case study |
   | **Adapter** | nối ra ngoài: PostgreSQL, model / embedding / rerank theo giao thức, app của khách | import lõi và module mà nó phục vụ |
   | **Ví dụ** | pack trung tính + dữ liệu công khai cho người fork | chỉ ở ngoài lõi / module |
   | **Case study** | một việc thật áp kiến trúc (Fuel) | chỉ ở ngoài lõi / module; không ai trong lõi / module import nó |

   Đo luật import bằng `go list -deps <gói>` (không bằng đọc tay); cổng tự động thuộc lát L1 (ADR-0014).

2. **Luật "≥ 2 miền mới vào lõi".** Một khái niệm (kiểu, trường, chữ cố định, luật) chỉ vào lõi — hay vào mã dùng chung của một module —
   khi **ít nhất hai miền** dùng nó, mỗi miền là một pack **chạy được và có test**. Trước đó nó nằm trong pack (cấu hình) hoặc trong case
   study. Phép đo khi thêm miền thứ hai, trên nhánh thêm pack:

   ```bash
   git diff --stat <mốc>..HEAD -- core data intelligence execution reliability observability platform cmd
   ```

   - đích: **0 tệp** (chỉ thêm `tenants/<miền>/` và test);
   - mỗi tệp dùng chung phải sửa là một chỗ lõi còn mang giả định của miền đầu; ghi nó vào báo cáo của lát, không lặng lẽ sửa cho qua;
   - **> 10 tệp** với một pack bậc B (phân cấp thực thể khác) là điều kiện đảo B′ → viết lại (báo cáo §11.2).

3. **Hợp đồng runtime dùng kiểu SẴN CÓ, không phát minh** (báo cáo §11.4):

   | Hợp đồng | Kiểu đang có | Việc |
   |---|---|---|
   | ngữ cảnh yêu cầu (principal / tenant) | `contracts.Principal`, `contracts.WithTenant` / `TenantFrom` (`core/contracts/tenantctx.go`) | giữ |
   | đăng ký năng lực / route, kiểm chính sách | `router.Policy` (`Allows`, `InScope`) + `intent` | tách `Policy` ra `core/policy` kèm type alias ở router (F-031, lát L4) |
   | kết quả thực thi | `commands.Result` (DONE / FAILED / UNKNOWN, `retry`) | giữ; tổng quát hoá kho lệnh khi có lệnh thứ hai |
   | bằng chứng | `contracts.Evidence` | giữ |
   | vết | `traces.Record`, chặng S0–S10 / C0–C7 | giữ |

4. **Cây đích — CHƯA dời** (báo cáo §11.5):

   ```text
   core/        contracts, decimal, traces, policy (tách từ router), tenant-meta (tách từ core/tenant)
   modules/
     query/     ask, pack, normalize, resolve, truth, intent, route, validate   ← động cơ S0–S10
     rag/       MỘT package rag; store tài liệu qua seam LoadDocuments
     command/   C0–C7 (tổng quát hoá khi có lệnh thứ hai)
     reports/   Phase 0a (sau khi nối + sửa R-01 + kiểm quyền)
   adapters/    postgres, llm, embeddings, (crm-mock, chat-mock: chưa có)
   examples/    01-knowledge-base (pack trung tính), 02-sme-minimal (pack thứ hai)
   case-studies/fuel-reconciliation  ← tenants/construction + win/*
   ```

   Dời thư mục **HOÃN** tới khi đủ cả bốn: pack thứ hai qua cổng của mục 2 (lát L5); H6 đã chạy (manifest bộ giữ lại neo đường dẫn);
   cổng kiến trúc bao được thư mục mới; neo đột biến và manifest cập nhật **cùng commit** dời. Trước đó mọi việc tách làm **tại chỗ**.

5. **Ánh xạ thư mục hiện tại → lớp đích** (để xếp mã mới ngay từ hôm nay, không cần dời):

   | Hiện tại | Lớp đích |
   |---|---|
   | `core/contracts`, `core/decimal`, `observability/traces` | lõi |
   | `Policy` / `Allows` / `InScope` trong `intelligence/router` | lõi (`core/policy`, lát L4) |
   | `core/tenant` | tách: phần meta → lõi; phần biên dịch pack của QUERY → module query (F-029, lát L5) |
   | `core/pipeline`, `intelligence/intent`, phần route của `intelligence/router`, `data/normalization`, `data/entityresolution`, `data/sourceoftruth`, `reliability/validation` | module query |
   | `intelligence/rag` | module rag |
   | `execution/commands` | module command |
   | `execution/reports` | module reports |
   | `data/ingestion` | đường nạp của module query; lược đồ thực thể chuyển sang do pack khai (F-004, lát L5) |
   | `platform/pgstore`, `intelligence/models` | adapter |
   | `cmd/eais` | ứng dụng lắp ráp (ngoài các lớp) |
   | `evals/`, `simulations/` | bộ đo, ngoài lõi |
   | `tenants/construction` (+ `win/*`) | case study Fuel |
   | `tenants/sports` | DOCS_ONLY — chỉ có bộ ca, chưa có pack |

## Hệ quả

- Mã mới đặt theo lớp đích của mục 5 dù thư mục chưa dời; chữ của một miền mới viết vào lõi là vi phạm mục 2, kể cả khi cổng hôm nay chưa bắt.
- `docs/01-architecture.md` (bảng HIỆN CÓ / ĐÍCH đo ở `bff56c4`) lạc hậu: hiện trạng gói đọc từ `go list ./...` và ADR này.
- `CLAUDE.md` §25 (bản đồ repo) còn ghi `cmd/eais/`, `execution/` là "đích — chưa có thư mục": chủ dự án sửa.
- Đo "số tệp lõi phải sửa" là cổng ra của lát L5; con số ước lượng ở Bối cảnh chưa phải số đo.

## Phương án đã cân nhắc

- **A — chỉ viết rõ ranh giới trong tài liệu:** bị loại; miền mới vẫn sửa ~6 tệp lõi (F-004, F-005, F-029, F-030 không được giải).
- **C — viết lại sạch + cầu tương thích:** bị loại; phải chuyển lại ~2.400 test, danh mục đột biến và bộ giữ lại đang neo đường dẫn; thua B
  kể cả khi được chấm lạc quan (báo cáo §11.2: A 3,20 · B 3,90 · C 2,55).
- **Dời thư mục ngay:** bị loại; rẻ nhưng ít giá trị (RAG đã độc lập ở mức import), và làm gói mới thoát cổng kiến trúc hiện tại (F-011).
