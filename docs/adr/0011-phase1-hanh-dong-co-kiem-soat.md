# ADR-0011 — Phase 1: hành động có kiểm soát, một lệnh tạo một phiếu (chế độ B)

- **Ngày:** 22/10/2026.
- **Trạng thái:** **ĐỀ XUẤT, chưa áp mã.**
  - Mã chỉ mở sau khi chạy xong H4: D-RM-04 = A (`docs/CHO-QUYET-DINH.md` mục 0, dòng 22/10 11:2x). H4 đã chạy MỘT lần ngày 22/10 14:59 ở `57409f2` (104/119 hai đường; `docs/STATUS.md`) — điều kiện mở mã của D-RM-04 = A đã thoả. Dòng S4 của CỔNG TRƯỚC AI KHÔNG ĐẠT (10/119 = 8,4% > 5%) và chờ câu D-AI-02 [REPO, `docs/STATUS.md`]: dòng đó chặn M3 và M5, không chặn điều kiện mở mã M4, và cổng ra của Phase 1 không đo S4 (D2).
  - Soát: lượt 1 độc lập (22/10/2026) KHÔNG ĐẠT, đã sửa theo soát. Lượt 2 (soát lại, 22/10/2026) ĐẠT CÓ GHI CHÚ, bốn ghi chú mức S3 (L3C-01…04), đã sửa (nơi đặt mã dưới `execution/commands`, cửa vào `cmd/eais`, số migration, khuôn ca đồng thời ở HOP-DONG §12.1; thêm cửa vào cho demo MVP, D10) và được lượt 3 xác nhận đã đóng. **Lượt 3 (soát lại, 22/10/2026): ĐẠT CÓ GHI CHÚ, mười hai ghi chú L3D-01…12 (bảy mức S3, năm mức S4).** Phần ADR của bản này sửa L3D-06 (trạng thái H4 và làn 0, mốc đọc làn 0, mốc so sánh đường hỏi), L3D-08 (nơi đặt mã: tệp ngoài `execution/`, write-set của làn 4, câu Q-P1-9) và L3D-11 (mục "Lệch tài liệu kiến trúc"); L3D-01…05 sửa ở HOP-DONG. **Không sửa** L3D-07 (thuộc ADR-0012, làn M2 đang sửa), L3D-09, L3D-10 và L3D-12 (lý do ở đầu HOP-DONG). **Phần sửa này chưa được soát lại.** Chưa có người nghiệp vụ duyệt (D-RM-03 = "chưa có").
  - Lượt 4 (soát bản sửa L3D, 22/10/2026 tối): ĐẠT CÓ GHI CHÚ (L3E-01…13); điều phối áp đề xuất của người soát cho L3E-01…04, 06, 08, 09, 12 khi gộp (xem đầu `HOP-DONG.md`), L3E-05, 07, 10, 11, 13 ghi nợ.
- **Liên quan:**
  - `docs/viec/1-hanh-dong/HOP-DONG.md` (hợp đồng việc: chi tiết của mọi quyết định dưới đây);
  - `docs/viec/1-hanh-dong/CONG-QUANIT.md` (9 câu cổng QuânIT, trên giấy: 5 ✓, 2 ◐, 2 ✗);
  - `docs/ROADMAP-DAY-DU.md` §4, §5 (luồng L3), §6 (dòng Phase 1);
  - `docs/01-architecture.md` §2.1, §3, §4.1, §7, §8;
  - `CLAUDE.md` §14, §15, §16, §19, §25;
  - `docs/PROMPT-BUILD-EAIS.md` §9.2 (write-set của các làn);
  - ADR-0001 (dòng "không mở COMMAND trước khi đóng G7": D-RM-04 thay điều kiện này; ADR này không tự thay), ADR-0006 (bộ giữ lại độc lập), ADR-0010 (S4 là sàn của hệ).
- **Nhãn bằng chứng:** [ĐO] người viết tự chạy lệnh · [REPO] tài liệu repo tự ghi · [CODE] đọc thấy trong mã (`tệp:dòng` tại `c086fe7`) · [SUY] suy luận · [NGOÀI] nguồn công khai ngoài repo. Tra một dòng trích bằng `git show c086fe7:<tệp>`. Nhánh `main` có thể đã đổi các tệp được trích (ví dụ `ask.go`, `router.go`, `intent.go`, pack), nên số dòng ở `main` có thể lệch; khi gộp phải đối chiếu lại các dòng trích. **Mốc đọc làn 0** (cửa `cmd/eais`, vết, migration): `9ed7a6a`, một commit nằm trong lịch sử của `main`: làn 0 (khối M1) đã gộp, mốc M1 ghi ở `41bc3cf`, `cmd/eais` có 10 tệp [ĐO `git ls-files cmd/ | wc -l` → 10 tại `269a539`]. Các dòng trích ghi "làn 0" tra bằng `git show 9ed7a6a:<tệp>`; các thư mục `cmd`, `db`, `observability`, `platform` ở `main` giống hệt `9ed7a6a` [ĐO `git diff --stat 9ed7a6a 269a539 -- cmd db observability platform` → rỗng], nên số dòng đã trích vẫn đúng ở `main` tại `269a539`. Làn 0 đổi tiếp thì đối chiếu lại.

## Bối cảnh (đã đo hoặc đọc thấy)

1. **Mặt phẳng P4 (Hành động) có 0 dòng mã thi hành.**
   - Thư mục `execution/`, `governance/`, `domain/` chưa có tệp nào được git theo dõi [ĐO `git ls-files | grep -E '^(execution|governance|domain)/' | wc -l` → 0 tại `269a539`]. Riêng `cmd/` có 10 tệp từ khi làn 0 gộp (`cmd/eais`: ứng dụng HTTP và CLI, không thi hành lệnh) [ĐO `git ls-files cmd/ | wc -l` → 10 tại `269a539`]: mục 9.
   - `EntryCommand` và route `TOOL` chỉ được khai báo (`core/contracts/contracts.go:196`, `:210`); route `TOOL` rơi vào nhánh lỗi `core/pipeline/ask.go:187-190` [CODE].
   - 17 ca COMMAND tồn tại dạng dữ liệu, chưa có bộ chạy [ĐO `grep -c "entry_type: COMMAND" tenants/construction/evals/cases.yaml` → 17].
2. **Dùng lại được:** tenant lấy từ Principal (`ask.go:89-95`); quyền đóng mặc định và phạm vi thực thể (`intelligence/router/router.go:127-161`); RLS ENABLE + FORCE và vai không vượt được RLS (`db/migrations/0002_dinh_danh_eais.sql`, `platform/pgstore/pgstore.go:25-45`); phép tính chênh lệch có thẩm quyền `Catalog.Derive` (`data/sourceoftruth/query.go:251`); `Lineage` (`data/entityresolution/resolver.go:251`) [CODE].
3. **Còn thiếu:**
   - audit: không mã, không bảng;
   - xác thực thật: chưa có (`docs/01-architecture.md` §4 L7) [REPO]; làn 0 cấp HTTP và token dev công khai (mục 9);
   - đường ghi nghiệp vụ: chỉ có `Load`, thay toàn bộ dữ liệu của tenant (`platform/pgstore/load.go:21`) [CODE];
   - lớp lưu trữ không tách được lỗi COMMIT: `inTenant` gom mọi lỗi thành `infra:postgres` (`pgstore.go:59-70`) [CODE];
   - đường hỏi coi câu lệnh là câu hỏi: "Tạo phiếu kiểm tra chênh lệch máy EX-03 hôm qua" được trả lời như câu hỏi tính toán, động từ bị bỏ qua [ĐO, test thăm dò tạm không commit, tại `c086fe7`, `1e5457b` và `7d72e33`; `HOP-DONG.md` §1.5].
4. **Ca nghiệp vụ:** CON-S-026 / CON-T-010 (tạo phiếu kiểm tra máy; domain bảo trì, đòi workflow và giao người nhận: Phase 1 đáp ứng một phần, ánh xạ ở `CONG-QUANIT.md` mục 1), CON-F-020 mức S1 (báo "đã tạo" khi không có phiếu); BR-FUEL-03 (chênh lệch vượt ngưỡng thì gắn cờ và đưa vào hàng chờ xem xét, `docs/tenants/construction/01-business-map.md:382`); mẫu câu hứa "đã chuyển người phụ trách" mà không có việc nào đứng sau (`tenants/construction/pack/templates.yaml:16-17`) [REPO, CODE].
5. **Quyết định của chủ dự án (22/10):** D-RM-04 = A (mã Giai đoạn II mở sau H4; mọi tuyên bố giá trị chờ W-G7); D-RM-03 = chưa có người nghiệp vụ; W-G7 và chi tiền chưa cấp [REPO].
6. **Việc theo roadmap:** Phase 1 = "tạo đúng một phiếu kiểm tra chênh lệch: có quyền, chống trùng, đọc lại, audit"; cổng ra: ca âm gửi đôi, thiếu quyền, timeout sau khi ghi (`docs/ROADMAP-DAY-DU.md` §6) [REPO].
7. **Cổng QuânIT (trên giấy):** trượt câu 1 và 8 (không có người dùng thật, không có người ký) và ◐ câu 3, 9. Không chặn việc viết giấy; giới hạn kết luận: chỉ "trên dữ liệu giả lập, chưa có người nghiệp vụ duyệt".
8. **MVP khối M4:** `docs/ROADMAP-DAY-DU.md` §5b (D-MVP-00; đọc ở `main` sau mốc `c086fe7`) đặt Phase 1 làm khối M4, có thêm một luật vượt dung sai kích hoạt việc tạo phiếu [REPO]. M4 không đổi quyết định nào: luật kích hoạt chỉ là một bên gọi lệnh (`HOP-DONG.md` §1.4). Phần còn mở là Principal `service` và khoá lệnh tất định (câu Q-P1-6).
9. **Làn 0 (khối M1)** [CODE, `9ed7a6a`; làn 0 đã gộp vào `main`, M1 RA theo cổng ra của lệnh ca 22/10: `docs/STATUS.md`]: `cmd/eais` có `POST /v1/ask`, `POST /v1/ingest`, `GET /v1/trace/{id}`, `GET /healthz`; Principal lấy từ token ở tiêu đề `Authorization`, không từ thân; `serve` chỉ chạy với `-dev`, chỉ nghe loopback (`cmd/eais/server.go:74`, `:138`; `cmd/eais/main.go:163-180`). Kho vết `decision_records` (migration `0003_kho_vet_ben.sql`) lưu vết S0…S10 của mọi lượt đã xác thực, kể cả lượt lỗi; `eais trace` đọc nó. Đây là cửa HTTP và nơi lưu vết của MVP: Phase 1 dùng chúng, không dựng cửa hay kho vết riêng.

## Quyết định

Mỗi quyết định có chi tiết và ca kiểm ở `HOP-DONG.md`; ADR chỉ ghi lựa chọn và lý do.

| # | Quyết định | Chi tiết |
|---|---|---|
| **D1** | **Phạm vi:** một lệnh `open_discrepancy_ticket`; một loại phiếu; chỉ **tạo**; chế độ B; chỉ số `issue_vs_measured`; kỳ là một ngày đã qua; không chữ tự do. | HOP-DONG §1, §5 |
| **D2** | **Lối vào:** yêu cầu có cấu trúc (`CommandRequest`), gọi bằng hàm của `execution/commands` và, cho người dùng, bằng MỘT route lệnh của `cmd/eais` (D10). **Không** qua luồng hỏi và S4 ở Phase 1: câu chữ tự do vẫn vào `/v1/ask` và được trả lời như câu hỏi. Principal lấy từ token của `cmd/eais`; ở dev token công khai, chỉ loopback; **không mở ra ngoài máy dev trước Phase 5** (chưa có xác thực thật). | §1.3, §2, §2.6 |
| **D3** | **Pipeline lệnh C0–C7 tất định**, 0 lời gọi model, đặt ở gói mới `execution/commands` (không ở `core/pipeline`: mục A). Idempotency tách hai nửa: tra sổ lệnh ngay sau quyền; khoá duy nhất ở bước ghi. **Lệch thứ tự `CLAUDE.md` §14 và `docs/ROADMAP-DAY-DU.md` §5 L3: câu mở Q-P1-7.** | §3 |
| **D4** | **Lưu trữ:** ba bảng `commands`, `discrepancy_tickets`, `audit_log` (migration số kế tiếp sau migration cuối của `main` lúc mở mã; xem "Nơi đặt mã"), RLS FORCE, vai `eais_app` chỉ SELECT và INSERT, không khoá ngoại tới bốn bảng mà `Load` thay thế. Ghi trong **một giao dịch** theo thứ tự phiếu, sổ lệnh (khoá ngoại thường tới phiếu), audit; đọc lại từ nơi ghi trước khi báo xong. | §7 |
| **D5** | **Chống trùng hai tầng:** khoá lệnh do bên gọi sinh, phạm vi (tenant, người gọi), kèm fingerprint nội dung; khoá nghiệp vụ (tenant, máy, chỉ số, ngày) trên phiếu đang mở. | §6 |
| **D6** | **Kết quả ba trị** ĐÃ LÀM / CHƯA BIẾT / THẤT BẠI. Không biết thì `UNKNOWN`. Không `DONE` nếu chưa đọc lại thấy phiếu. | §2.3, §9 |
| **D7** | **Quyền:** capability mới `tickets.create`, cộng `metrics.read` và phạm vi thực thể; pack kiểm nhất quán lúc nạp; một mã cho "không có" và "ngoài phạm vi". | §4 |
| **D8** | **Cấu hình trong pack, hành vi trong mã** (P12 của `docs/01-architecture.md`): tệp mới `tenants/construction/pack/tools.yaml` khai tên tool, capability, chỉ số được phép, `min_abs_diff`, `timeout`. `execution/commands` nạp và kiểm nó; `core/tenant` không đổi. | §4.3, §5 |
| **D9** | **Chỉ tạo `execution/commands`**, vì đã có mã thật cần ranh giới hành động tách khỏi luồng hỏi (`CLAUDE.md` §25). Không tạo `governance/`, `domain/`; không `execution/tools`, không interface `Tool` chung (mục E). **Không sửa tệp nào của `core/`**: xem "Nơi đặt mã". | dưới |
| **D10** | **Cửa vào cho người dùng và vết của lượt lệnh (demo khối M4):** thêm đúng một route `POST /v1/commands/open_discrepancy_ticket` vào `cmd/eais` của làn 0; thân là `CommandRequest`, Principal từ token. Vết của lượt lệnh: chín bước C0…C7 ghi vào kho vết `decision_records` của làn 0, xem bằng `eais trace`. Câu chữ "Tạo phiếu kiểm tra" gõ vào `/v1/ask` vẫn được trả lời như câu hỏi; câu Q-P1-8 hỏi chủ dự án có muốn đổi. | §2.6, §7.5 |

## Nơi đặt mã

Theo `docs/01-architecture.md` §3, `CLAUDE.md` §25 (thư mục chỉ được tạo khi có mã thật cần boundary đó) và `docs/PROMPT-BUILD-EAIS.md` §9.2 (write-set của các làn: làn 4 là `execution/` mới, migration mới, pack). Phase 1 khớp **một phần** write-set đó: ngoài `execution/`, migration và pack còn các chỗ ở bảng "Tệp ngoài `execution/`" dưới đây (đoạn "Ngoài write-set của làn 4").

**Đã đổi sau soát lại (ghi chú L3C-04).** Bản đầu của ADR này đặt mã ở `core/pipeline`, `core/contracts` và `platform/pgstore`. Soát lại chỉ ra: `core/pipeline` là write-set của làn 1 (đường găng 0b) và chung một gói Go với `ask.go`; `platform/` (bảng vết) là của làn 0; còn `execution/`, nơi §9.2 giao cho làn 4, mới là chỗ có mã thật cần ranh giới hành động. **Quyết định của điều phối (22/10):** mã lệnh Phase 1 nằm dưới `execution/commands`; chạm `core/` chỉ ở mức tối thiểu, liệt kê đích danh. Bảng dưới là kế hoạch cho lúc mã được mở; **hiện chưa tạo gì**.

**Mã mới trong gói `execution/commands`** (tên tệp là đề xuất):

| Tệp | Làm gì |
|---|---|
| `execution/commands/command.go` | kiểu `Request`, `Result`, trạng thái, mã lý do (HOP-DONG §2). Kiểu thân của lớp HTTP cũng có tên kết thúc bằng `Request` và không mang tenant |
| `execution/commands/pipeline.go` | pipeline C0–C7; interface lưu trữ do gói khai tại chỗ dùng (thành ngữ Go của `contracts.go`); nhận Principal và yêu cầu, trả kết quả và bản ghi vết |
| `execution/commands/tools.go` | kiểu và hàm nạp `tools.yaml`; ba luật nạp của HOP-DONG §4.3 (kiểm chéo với `policy.yaml`, `metrics.yaml` mà `tenant.Load` đã nạp) |
| `execution/commands/pgticket.go` | kho PG của phiếu: SQL của ba bảng, đơn vị ghi phiếu tách được lỗi COMMIT, tra sổ lệnh, đọc lại, audit riêng |
| `execution/commands/trace.go` | dựng chín bước C0…C7 và bản ghi vết của lượt lệnh (HOP-DONG §7.5) |
| test của gói | ca phát triển (tầng 1) của HOP-DONG §10 |
| `execution/commands/*_test.go` (bộ chạy ca COMMAND) | bộ chạy chung của HOP-DONG §12.1: đọc tệp ca giữ lại, gọi `evals.Validate`, dựng các runtime, lớp bọc lưu trữ và rào F7 test-only của chính gói, đếm bằng kết nối riêng. Đề xuất đặt trong test của gói, không ở `evals/` (vùng của làn 2; `evals/runner.go` hiện chạy ca WIN qua luồng hỏi): bộ chạy đi cùng lớp bọc và rào F7 của gói và không cần chạm làn khác; đổi chỗ chỉ là chuyển tệp, không đổi HOP-DONG §12.1 |

Gói này không import `core/tenant` và `core/pipeline`: thứ nó cần (`router.Policy`, `sourceoftruth.Catalog`, kho dữ kiện, danh mục thực thể, múi giờ, phiên bản pack) do `cmd/eais` truyền vào khi dựng.

**Tệp ngoài `execution/` mà mã Phase 1 phải chạm** (tối thiểu, đích danh). Chạm tệp không có trong bảng này thì phải sửa giấy trước (thêm tệp vào bảng kèm lý do) trong cùng commit.

| Tệp | Chạm? | Vì sao |
|---|---|---|
| `core/pipeline/ask.go`, `core/pipeline/claim.go` | **không** | luồng hỏi không đổi (I9); lệnh không đi qua `Handle` |
| `core/contracts/*` (`EntryCommand`, `RouteTool` đã khai sẵn) | **không** | kiểu lệnh nằm ở `execution/commands`; hai hằng sẵn có chỉ dùng làm nhãn vết (`Route` = `TOOL`) |
| `core/tenant/pack.go` | **không** | `tenant.Load` không biết `tools.yaml`: `execution/commands` nạp và kiểm nó, `cmd/eais` gọi ngay sau `tenant.Load` lúc khởi động. Giữ nguyên hành vi nạp của luồng hỏi và công cụ đo (WIN, G7); tránh vòng import (`commands` cần `Policy`, `Catalog`; `tenant` sẽ phải import `commands`). `PackVersion` vẫn đổi vì `hashPack` băm mọi `.yaml` cấp một: HOP-DONG U2 |
| `intelligence/router/*`, `intelligence/intent/*` | **không** | dùng `Policy.Allows`, `Policy.InScope` nguyên trạng |
| `architecture_test.go` | **không** | `sharedRoots` đã gồm `execution` và `cmd`, nên hai cổng (tên tenant trong literal, kiểu `Request` mang tenant) quét cả hai cây [CODE `architecture_test.go:19`] |
| `observability/traces/step.go` | **có**, nhỏ | `ValidStage` nhận thêm `C0`…`C7` và `C4b`; không thì `Record.Validate` từ chối vết lệnh [CODE, làn 0: `step.go:60`, `record.go:59` tại `9ed7a6a`] |
| `platform/pgstore/` (một tệp mới, không sửa tệp cũ) | **có**, nhỏ | hàm mở giao dịch đã đặt GUC tenant và trả giao dịch để bên gọi tự `Commit` / `Rollback`: `Store.pool` và `inTenant` không xuất ra, và `inTenant` gom lỗi COMMIT vào `infra:postgres` (HOP-DONG §7.1) [CODE `pgstore.go:59-70`] |
| `cmd/eais/server.go`, `cmd/eais/app.go` | **có**, nhỏ (làn 0 đã gộp) | thêm một route lệnh, một handler, giải mã chặt; dựng pipeline lệnh từ `app`; mở rộng giao diện `store` (hoặc thêm một giao diện hẹp riêng) (HOP-DONG §2.6) |
| `db/migrations/<NNNN>_phieu_chenh_lech.sql` | tệp mới | ba bảng, RLS, quyền, chỉ mục duy nhất, khoá ngoại thường từ sổ lệnh tới phiếu. `<NNNN>` là số kế tiếp sau migration cuối của `main` lúc mở mã: làn 0 đã dùng `0003_kho_vet_ben.sql`, Phase 0a cũng cần một số, ba bên không được cùng một số |
| `tenants/construction/pack/tools.yaml` | tệp mới | khai tool (tên tệp theo `docs/01-architecture.md` §8) |
| `tenants/construction/pack/policy.yaml` | sửa nhỏ | thêm `tickets.create` cho ba vai. Vùng pack là của làn 1 (đường găng 0b): phối hợp khi mở mã. Đổi tập capability nên đổi cả phép "toàn quyền dữ liệu" của `cmd/eais`: HOP-DONG §4.2 |
| `scripts/demo-m4.sh`, `README.md` | mới, thêm mục | kịch bản demo của khối M4 (HOP-DONG §2.6.4); chỉ khi M4 qua cổng |
| `scripts/dot-bien/<tên do BUILD chọn>` | tệp mới | danh mục đột biến M1–M26 của cổng X8 (HOP-DONG §11.1, §11.2); mẫu: `scripts/dot-bien/n43-cong-phan-quyet.sh`. Vùng `scripts/dot-bien/` là của làn 2: phối hợp khi mở mã |
| `tenants/construction/evals/<tên do người soạn đặt>` | tệp mới, do người soạn độc lập viết | tệp ca giữ lại của Phase 1 theo HOP-DONG §12.1: đóng băng bằng cam kết băm trước mã (ADR-0006, mục 5), công bố khi chạy; bộ chạy ở test của gói đọc nó lúc cổng ra. Vùng `tenants/*/evals/` là của làn 3 |
| `cmd/eais/*_test.go` (tệp test mới) | **có**, tệp mới | ca A28 (cửa HTTP của lệnh) và A29 (vết lệnh): route lệnh, thân `CommandResult`, 405, vết C0…C7. Test sẵn có của `cmd/eais` không sửa (A27). Vùng `cmd/eais/` là của làn 0 |
| `governance/`, `domain/` | chưa có | **không tạo**: chưa có mã thật cần ranh giới đó (xác thực ở Phase 5; adapter CRM ở Phase 4b) |

**Ngoài write-set của làn 4.** `docs/PROMPT-BUILD-EAIS.md` §9.2 giao làn 4: `execution/` mới, migration mới, pack. Trong bảng trên, các chỗ sau nằm ngoài write-set đó: `observability/traces/step.go`, `platform/pgstore/` (một tệp mới) và `cmd/eais/` (`server.go`, `app.go`, tệp test) là vùng của làn 0; `tenants/construction/pack/policy.yaml` là vùng pack của làn 1 (đường găng 0b), nơi cũng có phiên khác đang sửa (vòng nợ S4 sau H4): phối hợp khi mở mã; `scripts/dot-bien/` là vùng của làn 2; tệp ca giữ lại ở `tenants/*/evals/` là vùng của làn 3. Phase 0a xin mở rộng write-set của làn 4 bằng câu Q-0A-12; Phase 1 xin bằng câu Q-P1-9 (HOP-DONG §14), chưa trả lời thì mặc định của câu đó áp dụng.

Ràng buộc kiến trúc phải giữ:
- mã dùng chung không chứa tên tenant trong chuỗi literal (`architecture_test.go`, `TestSharedCodeHasNoTenantLiterals`); mã lệnh chỉ nói "chỉ số dẫn xuất" và "khoá thực thể", không nói "fuel" hay "construction";
- kiểu yêu cầu tên kết thúc `Request` và không mang tenant (`TestRequestTypesHaveNoTenantField`);
- `execution/commands` là gói Go riêng: không chung ký hiệu với `core/pipeline`, nên vòng sửa S4 (sửa `ask.go`) không thể va chạm tên với nó.

**Lệch tài liệu kiến trúc.** `docs/01-architecture.md` §3 còn ghi `core/pipeline/` có "ĐÍCH: command (C0–C7)", `execution/` là "ĐÍCH: tools · workflows · triggers · skills", và `/v1/commands` ở `platform/httpapi/` (ĐÍCH, mặt phẳng P7) [REPO]. Phase 1 đặt route lệnh ở `cmd/eais/server.go`, lớp HTTP của làn 0 (D10), và không tạo `platform/httpapi/`. Theo thứ bậc ở đầu `docs/ROADMAP-DAY-DU.md`, ADR đứng trên `CLAUDE.md` và roadmap; tài liệu kiến trúc không nằm trong thứ bậc đó. ADR này vì vậy là nguồn cho nơi đặt mã Phase 1; §3 cần đồng bộ khi ADR chuyển CHỐT (việc của phiên điều phối).

## Phương án đã cân nhắc

### A. Đường lệnh đặt ở đâu

| Phương án | Ưu | Nhược |
|---|---|---|
| **A1 (chọn, sau soát lại):** pipeline lệnh riêng ở gói mới `execution/commands`, kèm kho PG của phiếu | khớp kiến trúc đích (ACTION PIPELINE tách khỏi ASK, `docs/01` §2.1; mặt phẳng P4 Hành động là `execution/`); khớp **một phần** write-set của làn 4 (`docs/PROMPT-BUILD-EAIS.md` §9.2: `execution/`, migration, pack; các chỗ ngoài write-set: đoạn "Ngoài write-set của làn 4" ở mục "Nơi đặt mã", câu Q-P1-9); gói Go riêng nên không chung ký hiệu với `core/pipeline`, vòng sửa S4 không thể va chạm; SQL của phiếu nằm cùng chỗ với luật của phiếu (R5, §6.4); cổng ra đo biên hành động, không đo S4 | tạo một thư mục mới (đúng chỗ: đã có mã thật, `CLAUDE.md` §25); thêm một pipeline; bước tenant và quyền lặp ý của S1, S5 (giải bằng cách gọi lại `Policy.Allows`, `InScope`, không chép luật); `platform/pgstore` phải xuất một hàm mở giao dịch theo tenant; lệch `docs/01` §3 (nợ ghi ở "Nơi đặt mã") |
| A1b (bản đầu, bị loại sau soát lại): tệp mới trong `core/pipeline`, kiểu trong `core/contracts`, kho PG trong `platform/pgstore` | không tạo thư mục; `pgstore` không cần hàm xuất mới | `core/pipeline` là write-set của làn 1 và chung một gói Go với `ask.go` (phải đặt tiền tố `command` cho ký hiệu để khỏi đụng vòng S4); `platform/` (bảng vết) là của làn 0; kéo Phase 1 vào vùng của hai làn khác (ghi chú L3C-04) |
| A2: mở rộng luồng hỏi, thêm xử lý route `TOOL` ở `ask.go:187-190` | tái dùng S0–S5 | trộn QUERY và COMMAND trong một hàm lớn (`ask.go` 1.279 dòng [ĐO `wc -l`]); cổng ra phụ thuộc S4, mà S4 là tầng sai đầu tiên của 13/15 ca đỏ H1 (ADR-0010) nên đo nhầm thứ; đụng tệp đang sửa; câu chữ "đã tạo" sinh ngay cạnh việc ghi, đúng chỗ CON-F-020 hỏng |
| A3: module miền dưới `tenants/construction/` | nhanh, không đụng lõi, không nguy cơ lõi gắn ngành | biên kiểm soát (quyền, chống trùng, đọc lại, audit) là chung, không riêng ngành; Phase 4b (CRM) phải chép hoặc trích ra; ngược `docs/CONVENTIONS.md` §8 (khác biệt tenant ở pack, hành vi chung ở lõi) |

**Nợ đã nhận:** chỉ một tenant dùng đường lệnh, nên chưa chứng minh được là `SHARED CORE` (`docs/CONVENTIONS.md` §4.5). Ghi nợ "một tenant" cho tới Phase 4b. Phần riêng ngành (chỉ số nào, ngưỡng nào, vai nào) nằm ở pack.

### B. Phiếu lưu ở đâu

| Phương án | Ưu | Nhược |
|---|---|---|
| **B1 (chọn):** bảng riêng `discrepancy_tickets`, cộng sổ lệnh và audit | nhỏ, đúng một loại phiếu; không định trước engine của Phase 2 | Phase 2 có thể thêm chủ thứ hai của "trạng thái" (`workflow_instances`): ADR của Phase 2 phải chốt ai sở hữu `status` |
| B2: dùng `workflow_instances` từ đầu (như CON-T-010: `start_workflow`, trạng thái `null` → `OPEN`) | liền mạch với lược đồ đích (`docs/01` §7) | thiết kế mô hình workflow chung từ **một** ví dụ (trừu tượng sớm, P7); chốt trước kết quả so Temporal, n8n mà L4 đòi làm **trước khi viết mã** Phase 2 (`docs/ROADMAP-DAY-DU.md` §5 L4); kéo máy trạng thái và human task vào Phase 1 |
| B3: chỉ sổ lệnh, kết quả là JSON | đơn giản nhất | không có đối tượng nghiệp vụ truy vấn được; không có khoá nghiệp vụ chống trùng; Phase 2 phải làm lại |
| B4: tạo issue ở hệ ngoài (ticket, CRM) | đúng với chế độ A | sai với chế độ B (khách chưa có hệ nào); thuộc Phase 4b (`CONG-QUANIT.md` mục 4) |

### C. Khoá chống trùng

| Phương án | Ưu | Nhược |
|---|---|---|
| **C1 (chọn):** hai tầng, khoá lệnh do bên gọi sinh + khoá nghiệp vụ | khớp mẫu đã có ở API thanh toán (Stripe) và bản nháp IETF [NGOÀI]; phân biệt "gửi lại" với "ý định mới"; chặn cả bấm đôi tạo hai UUID | bên gọi phải sinh và giữ khoá |
| C2: chỉ khoá lệnh | đơn giản | hai UUID khác nhau cho cùng một việc tạo hai phiếu |
| C3: chỉ khoá nội dung (băm tenant, máy, chỉ số, ngày) | bên gọi không cần khoá | không phân biệt gửi lại với ý định mới: sau khi Phase 2 đóng phiếu, tạo phiếu mới cho cùng ngày bị chặn mãi hoặc phải thêm đệm thời gian tuỳ tiện; gửi lại sau timeout nhận `DUPLICATE_OPEN_TICKET` thay vì kết quả cũ (HOP-DONG A21) |
| C4: khoá do server cấp (xin khoá rồi gửi) | khoá không đoán được | thêm một vòng và một đường ghi, chưa có ca đòi |

### D. Đơn vị ghi và audit

| Phương án | Ưu | Nhược |
|---|---|---|
| **D1 (chọn):** một giao dịch cho phiếu, sổ lệnh, audit `EXECUTED` (theo thứ tự đó); đọc lại sau commit | không có phiếu mà thiếu audit; không cần tiến trình nền | chỉ đúng khi mọi thứ ở một cơ sở dữ liệu (chế độ B) |
| D2: outbox, worker ghi audit hoặc gửi đi | tách việc chậm khỏi đường đồng bộ | thêm tiến trình nền; `CLAUDE.md` §19 chỉ mở queue khi số đo chứng minh đường đồng bộ không đạt, mà chưa đo; cửa sổ có phiếu chưa có audit |
| D3: audit ngoài giao dịch, cố gắng tối đa | không làm hỏng việc chính khi audit lỗi | có thể có phiếu không audit: vi phạm "không có tác dụng phụ mà thiếu audit". **Chỉ dùng cho dòng của lần từ chối** (không có tác dụng phụ) |

### E. Có tạo interface `Tool` chung ngay không

| Phương án | Ưu | Nhược |
|---|---|---|
| **E1 (chọn):** không; một hàm cụ thể cho một tool, đường nối là interface lưu trữ do pipeline khai | không trừu tượng hoá từ một ví dụ; không tạo `execution/tools` chỉ vì kiến trúc đích có tên đó (chỉ `execution/commands`, nơi có mã thật) | tool thứ hai (Phase 4b) phải trích ra; chi phí trích nhỏ vì ranh giới đã sạch |
| E2: tạo `execution/tools` và `Tool` (`Spec`, `Invoke`) như bảng hợp đồng đích (`docs/01` §6) | sẵn chỗ cho tool sau | `CLAUDE.md` §25: không tạo folder chỉ vì kiến trúc đích có tên; hợp đồng `Tool` đoán từ một ví dụ |

### F. Lượt "Tạo phiếu kiểm tra" của demo đi vào bằng cửa nào

| Phương án | Ưu | Nhược |
|---|---|---|
| **F1 (chọn):** route lệnh riêng `POST /v1/commands/open_discrepancy_ticket` ở `cmd/eais`, thân có cấu trúc | khớp D2 và HOP-DONG §2.1; không đổi luồng hỏi (I9); không cần C2 hay ngữ cảnh của lượt trước; một route, một handler, tái dùng xác thực, giải mã chặt và lưu vết của làn 0 | lượt hai không phải câu chữ gõ tay; chạm `cmd/eais` (làn 0 đã gộp) |
| F2: câu chữ vào `/v1/ask`, hệ nhận ra là lệnh | đúng cảm giác hội thoại | đòi C2 và ngữ cảnh hội thoại (hiểu "máy 03 hôm qua" của lượt trước); sửa `ask.go`, `router.go`, `intent.go`: trái I9, đụng vòng S4 đã khoá; thuộc Phase 6 |
| F3: chỉ hàm Go hoặc lệnh con CLI | nhỏ nhất | CLI chạy bằng quyền người vận hành, không có Principal, nên không kiểm quyền được; không phải cửa mà chủ dự án dùng để demo |

Chi tiết, bảng mã HTTP và kịch bản: HOP-DONG §2.6. Câu mở cho chủ dự án: Q-P1-8 (mặc định F1).

### G. Vết của lượt lệnh lưu ở đâu

| Phương án | Ưu | Nhược |
|---|---|---|
| **G1 (chọn):** cùng kho vết `decision_records` của làn 0, chín bước C0…C7 | không thêm bảng; `eais trace` và `GET /v1/trace/{id}` đọc ngay, cùng quyền đọc; có cả lượt bị từ chối ở C0 đến C3 | `ValidStage` phải nhận mã bước `C…` (sửa `observability/traces/step.go`); phụ thuộc làn 0 (đã gộp) |
| G2: bảng vết riêng cho lệnh | không đụng `observability/traces` | hai kho vết, `eais trace` phải biết cả hai; thêm bảng và đường đọc mà chưa ca nào đòi (P7) |
| G3: chỉ audit, không vết từng bước | không thêm gì | audit không có lượt bị từ chối ở C0 đến C3 và không có lý do từng bước: trái yêu cầu của M1 ("mọi lượt, kể cả lượt lỗi, có vết"); người soát không chỉ ra được tầng sai đầu tiên |

Quy tắc đi kèm G1 (HOP-DONG §7.5): vết không lộ điều mà kết quả không lộ (không tách "không tồn tại" với "ngoài phạm vi"); mất vết không đổi kết quả đã ghi.

## Hệ quả

**Tích cực**
- Lát COMMAND đầu tiên, đo được **độc lập với S4**: cổng ra chỉ thấy biên hành động (quyền, chống trùng, đọc lại, audit).
- Tái dùng RLS, quyền, danh mục thực thể, phép tính có thẩm quyền; không viết lại.
- Sổ lệnh và khoá khử trùng là nền cho Phase 3 (sự kiện xử lý đúng một lần theo khoá, `docs/01` §4 L5).
- Chủ dự án debug lượt lệnh như lượt hỏi: `eais trace` in từng bước C0…C7 của cả lượt thành công lẫn lượt bị từ chối, và chỉ ra bước lệch đầu tiên (HOP-DONG §7.5).

**Tiêu cực và nợ có nhãn**
1. **Một tenant dùng** (xem mục A).
2. **Chỉ đúng ở chế độ B.** Khi ghi sang hệ khác (Phase 4b), "timeout sau khi ghi" cần thiết kế riêng (bản ghi lệnh cục bộ, mã ngoài để khử trùng): cần ADR riêng. Ghi ở HOP-DONG U8.
3. **`status` có thể có hai chủ** khi Phase 2 tới (B1): ADR của Phase 2 chốt.
4. **Audit chưa có chuỗi băm; khoá lệnh không hết hạn.** Hoãn Phase 5. Chưa có lỗi nào đòi (P7).
5. **Chưa có xác thực thật:** route lệnh chỉ sống ở `serve -dev` (token công khai, chỉ loopback); không mở ra ngoài máy dev trước Phase 5.
6. **`tools.yaml` đổi `PackVersion`** (băm mọi tệp `.yaml` của pack, `core/tenant/pack.go:304`) [CODE]; các chỗ ghim phiên bản (bằng chứng của `g7run`) chưa kiểm: HOP-DONG U2.
7. **Phiếu là bản chụp tại lúc tính.** Lỗi im lặng của luồng L1 vẫn có thể tạo chênh lệch giả (`CONG-QUANIT.md` câu 3).
8. **Không tuyên bố giá trị:** câu 1 và 8 còn ✗ (D-RM-03).
9. **Lần đầu vai ứng dụng có đường ghi nghiệp vụ** ngoài `Load`: chỉ INSERT, không UPDATE, DELETE. Cổng ra có ca thử trực tiếp (A26).
10. **Bên kích hoạt tự động (khối M4)** cần Principal `service` có vai và khoá lệnh tất định. Nằm ngoài ADR này, ghi ở câu Q-P1-6 của hợp đồng.
11. **Đường hỏi vẫn coi câu lệnh là câu hỏi** (HOP-DONG §1.5, ca A25, [ĐO]): Phase 1 không chặn; nợ cho vòng S4 hoặc Phase 6.
12. **C4b lệch thứ tự `CLAUDE.md` §14 và ROADMAP L3** (Idempotency tách hai nửa; HOP-DONG §3): giá trị là ca A21 (gửi lại sau timeout nhận `DONE` phát lại). Chưa ai duyệt: câu Q-P1-7.
13. **Lệch `docs/01-architecture.md` §3** (nơi còn ghi `core/pipeline/` ĐÍCH command và `execution/` ĐÍCH tools, workflows, triggers, skills): cần đồng bộ khi ADR chuyển CHỐT. Việc của phiên điều phối.
14. **`execution/commands` là gói đầu tiên của mặt phẳng P4:** workflow (Phase 2) và tool thứ hai (Phase 4b) phải quyết ranh giới với nó; chi phí nhỏ vì chưa có interface chung (mục E).
15. **Phụ thuộc làn 0** [CODE, `9ed7a6a`; làn 0 đã gộp vào `main`]: route lệnh và vết lệnh dựa vào `cmd/eais` và kho vết `decision_records` của làn 0. Pipeline, kho phiếu và migration không phụ thuộc làn 0, trừ `execution/commands/trace.go`: tệp đó dựng `traces.Record`, kiểu của làn 0 (`observability/traces`). Làn 0 còn đổi `cmd/eais`: đổi chỗ nào được trích thì đối chiếu lại (HOP-DONG U9).

## Không làm

- Không viết mã trước khi H4 chạy (D-RM-04). Không sửa `ask.go`, `claim.go`, `intent.go`, `router.go` và **không sửa tệp nào của `core/`**; nếu một việc buộc phải chạm, sửa giấy trước (bảng "Nơi đặt mã") trong cùng commit.
- Không nối kết quả lệnh vào mẫu câu của luồng hỏi; không sửa lời hứa `templates.yaml:16-17` ở Phase 1.
- Không thêm trường chữ tự do; không `reported_vs_issued`; không kỳ nhiều ngày.
- Không chặn câu lệnh viết bằng chữ tự do ở luồng hỏi (hiện được trả lời như câu hỏi: HOP-DONG §1.5); nợ cho vòng S4 hoặc Phase 6.
- Không gọi model; không interface `Tool` chung; không tạo `execution/tools`, `governance/`, `domain/`. Chỉ tạo `execution/commands` (D9); `cmd/eais` là của làn 0, Phase 1 chỉ thêm một route vào nó.
- Không tự thay dòng ADR-0001: điều kiện đó do D-RM-04 của chủ dự án thay.

## Chứng minh (theo thứ tự)

1. **Điều kiện mở:** H4 đã chạy MỘT lần (vòng S4 đã đóng băng trước đó). **Đã thoả** (22/10 14:59, `57409f2`).
2. **Hợp đồng được soát độc lập** (người khác với tác giả tài liệu này), và đạt, chủ dự án trả lời Q-P1-1…9 hoặc chấp nhận mặc định. (Lượt 1: KHÔNG ĐẠT, đã sửa. Lượt 2: ĐẠT CÓ GHI CHÚ, bốn ghi chú L3C-01…04, đã sửa và được lượt 3 xác nhận đã đóng. Lượt 3: ĐẠT CÓ GHI CHÚ, mười hai ghi chú L3D-01…12, sửa ở bản này trừ L3D-07, 09, 10, 12; cần soát lại phần sửa.)
3. **Bộ giữ lại độc lập** (HOP-DONG §12, §12.1): người không đọc mã soạn, đóng băng bằng cam kết băm, **trước** mã. Xong bước 1, 2 và 3 thì ADR-0011 chuyển từ ĐỀ XUẤT sang CHỐT (mục "Điều kiện chuyển" dưới).
4. **Hai thử nghiệm nhỏ, không phải mã sản phẩm,** để trả lời U1 (lỗi ở ranh giới COMMIT của trình điều khiển trên PostgreSQL thật) và U2 (nơi ghim `PackVersion`); ghi kết quả vào hợp đồng.
5. **Test đỏ trước:** ca A01–A29 của tầng 1 viết trước mã; nhìn từng ca đỏ đúng lý do.
6. **Migration kế tiếp (số theo `main` lúc mở mã) và test cách ly:** canary kiểu `TestRLSCanaryB` cho ba bảng mới; `eais_app` bị từ chối UPDATE, DELETE và dòng sổ lệnh trỏ tới phiếu không có (A26).
7. **Mã nhỏ nhất làm ca xanh,** theo thứ tự C0 → C7, mỗi chặng một commit, không chạm đường hỏi và không sửa tệp nào của `core/`.
8. **Móc route lệnh vào `cmd/eais`** (làn 0 đã gộp): một route, một handler; `ValidStage` nhận bước `C…`; ca A28, A29 xanh; trả lời U9; bộ test của `cmd/eais` giữ nguyên theo tên.
9. **Chạy bộ giữ lại độc lập MỘT lần;** báo theo từng cổng X1–X9, không gộp một con số.
10. **Đột biến M1–M26** (HOP-DONG §11.2): đột biến sống sót thì thêm ca hoặc ghi tương đương kèm lý do.
11. **Hồi quy đường hỏi:** bộ 117 / H1 / H3 / H4, đường bộ nhớ và PostgreSQL, so theo tên và theo từng câu trả lời: 0 dòng khác; cộng bộ test của `cmd/eais` (làn 0), so theo tên: 0 mất, 0 đổi. Hai phía của phép so theo mốc ở HOP-DONG §10 ("Mốc so sánh đường hỏi"): commit gốc của nhánh Phase 1 và đầu nhánh, cùng mã S4.
12. **Cổng chung:** `go vet ./...`, `gofmt -l`, `go test -count=1 ./...` có PostgreSQL, cổng danh tính.
13. **Soát đối kháng độc lập,** chỉ đọc: tìm cách tạo hai phiếu, hoặc một phiếu mà kết quả báo `FAILED` hay `UNKNOWN` sai, hoặc đọc vết mà biết được máy nào tồn tại. Finding quay lại sửa, rồi soát lại.
14. **Báo kết quả** là "cổng ra kỹ thuật Phase 1 đạt (hay không đạt) trên dữ liệu giả lập, chưa có người nghiệp vụ duyệt", không kèm tuyên bố giá trị.

## Điều kiện chuyển ĐỀ XUẤT → CHỐT

Bước 1, 2 và 3 của "Chứng minh" xong. Chưa trả lời được thì ADR giữ nguyên trạng thái ĐỀ XUẤT và không có mã.

## Rủi ro

- Hợp đồng sai ở chỗ chưa kiểm (U1–U10 của HOP-DONG §15), mỗi điểm có cách kiểm.
- Đã có ba lượt soát độc lập (22/10/2026): lượt 1 KHÔNG ĐẠT, đã sửa; lượt 2 ĐẠT CÓ GHI CHÚ (L3C-01…04), đã sửa và được lượt 3 xác nhận đã đóng; lượt 3 ĐẠT CÓ GHI CHÚ (L3D-01…12), sửa ở bản này (trừ L3D-07, 09, 10, 12). Phần sửa của lượt 3 chưa được soát: tác giả bản sửa không phải người soát, nên nó là SELF REVIEW cho tới khi soát lại.
- Làn 0 đã gộp; các dòng trích ghi "làn 0" đã đối chiếu ở `9ed7a6a` và ở `main` (mục "Mốc đọc làn 0" ở đầu tệp), nhưng làn 0 còn đổi `cmd/eais`: các dòng trích có thể lệch về sau (hệ quả 15).
- Đảo được: ADR và hợp đồng là giấy; bỏ chúng không làm đổi dòng mã nào.
