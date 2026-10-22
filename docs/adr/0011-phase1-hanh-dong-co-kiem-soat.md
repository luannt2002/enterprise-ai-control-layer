# ADR-0011 — Phase 1: hành động có kiểm soát, một lệnh tạo một phiếu (chế độ B)

- **Ngày:** 22/10/2026.
- **Trạng thái:** **ĐỀ XUẤT, chưa áp mã.**
  - Mã chỉ mở sau khi chạy xong H4: D-RM-04 = A (`docs/CHO-QUYET-DINH.md` mục 0, dòng 22/10 11:2x). H4 đã khoá (`15f04f9`) và CHƯA chạy (`docs/STATUS.md`).
  - Soát: lượt 1 độc lập (22/10/2026) KHÔNG ĐẠT, đã sửa theo soát; **chưa soát lại sau sửa**. Chưa có người nghiệp vụ duyệt (D-RM-03 = "chưa có").
- **Liên quan:**
  - `docs/viec/1-hanh-dong/HOP-DONG.md` (hợp đồng việc: chi tiết của mọi quyết định dưới đây);
  - `docs/viec/1-hanh-dong/CONG-QUANIT.md` (9 câu cổng QuânIT, trên giấy: 5 ✓, 2 ◐, 2 ✗);
  - `docs/ROADMAP-DAY-DU.md` §4, §5 (luồng L3), §6 (dòng Phase 1);
  - `docs/01-architecture.md` §2.1, §3, §4.1, §7, §8;
  - `CLAUDE.md` §14, §15, §16, §19, §25;
  - ADR-0001 (dòng "không mở COMMAND trước khi đóng G7": D-RM-04 thay điều kiện này; ADR này không tự thay), ADR-0006 (bộ giữ lại độc lập), ADR-0010 (S4 là sàn của hệ).
- **Nhãn bằng chứng:** [ĐO] người viết tự chạy lệnh · [REPO] tài liệu repo tự ghi · [CODE] đọc thấy trong mã (`tệp:dòng` tại `c086fe7`) · [SUY] suy luận · [NGOÀI] nguồn công khai ngoài repo. Tra một dòng trích bằng `git show c086fe7:<tệp>`. Nhánh `main` có thể đã đổi các tệp được trích (ví dụ `ask.go`, `router.go`, `intent.go`, pack), nên số dòng ở `main` có thể lệch; khi gộp phải đối chiếu lại các dòng trích.

## Bối cảnh (đã đo hoặc đọc thấy)

1. **Mặt phẳng P4 (Hành động) có 0 dòng mã thi hành.**
   - Thư mục `execution/`, `governance/`, `cmd/`, `domain/` chưa có tệp nào được git theo dõi [ĐO `git ls-files | grep -E '^(execution|governance|cmd|domain)/'` → rỗng].
   - `EntryCommand` và route `TOOL` chỉ được khai báo (`core/contracts/contracts.go:196`, `:210`); route `TOOL` rơi vào nhánh lỗi `core/pipeline/ask.go:187-190` [CODE].
   - 17 ca COMMAND tồn tại dạng dữ liệu, chưa có bộ chạy [ĐO `grep -c "entry_type: COMMAND" tenants/construction/evals/cases.yaml` → 17].
2. **Dùng lại được:** tenant lấy từ Principal (`ask.go:89-95`); quyền đóng mặc định và phạm vi thực thể (`intelligence/router/router.go:127-161`); RLS ENABLE + FORCE và vai không vượt được RLS (`db/migrations/0002_dinh_danh_eais.sql`, `platform/pgstore/pgstore.go:25-45`); phép tính chênh lệch có thẩm quyền `Catalog.Derive` (`data/sourceoftruth/query.go:251`); `Lineage` (`data/entityresolution/resolver.go:251`) [CODE].
3. **Còn thiếu:**
   - audit: không mã, không bảng;
   - xác thực: Principal do bên gọi dựng, không có HTTP (`docs/01-architecture.md` §4 L7) [REPO];
   - đường ghi nghiệp vụ: chỉ có `Load`, thay toàn bộ dữ liệu của tenant (`platform/pgstore/load.go:21`) [CODE];
   - lớp lưu trữ không tách được lỗi COMMIT: `inTenant` gom mọi lỗi thành `infra:postgres` (`pgstore.go:59-70`) [CODE];
   - đường hỏi coi câu lệnh là câu hỏi: "Tạo phiếu kiểm tra chênh lệch máy EX-03 hôm qua" được trả lời như câu hỏi tính toán, động từ bị bỏ qua [ĐO, test thăm dò tạm không commit, tại `c086fe7`, `1e5457b` và `7d72e33`; `HOP-DONG.md` §1.5].
4. **Ca nghiệp vụ:** CON-S-026 / CON-T-010 (tạo phiếu kiểm tra máy; domain bảo trì, đòi workflow và giao người nhận: Phase 1 đáp ứng một phần, ánh xạ ở `CONG-QUANIT.md` mục 1), CON-F-020 mức S1 (báo "đã tạo" khi không có phiếu); BR-FUEL-03 (chênh lệch vượt ngưỡng thì gắn cờ và đưa vào hàng chờ xem xét, `docs/tenants/construction/01-business-map.md:382`); mẫu câu hứa "đã chuyển người phụ trách" mà không có việc nào đứng sau (`tenants/construction/pack/templates.yaml:16-17`) [REPO, CODE].
5. **Quyết định của chủ dự án (22/10):** D-RM-04 = A (mã Giai đoạn II mở sau H4; mọi tuyên bố giá trị chờ W-G7); D-RM-03 = chưa có người nghiệp vụ; W-G7 và chi tiền chưa cấp [REPO].
6. **Việc theo roadmap:** Phase 1 = "tạo đúng một phiếu kiểm tra chênh lệch: có quyền, chống trùng, đọc lại, audit"; cổng ra: ca âm gửi đôi, thiếu quyền, timeout sau khi ghi (`docs/ROADMAP-DAY-DU.md` §6) [REPO].
7. **Cổng QuânIT (trên giấy):** trượt câu 1 và 8 (không có người dùng thật, không có người ký) và ◐ câu 3, 9. Không chặn việc viết giấy; giới hạn kết luận: chỉ "trên dữ liệu giả lập, chưa có người nghiệp vụ duyệt".
8. **MVP khối M4:** `docs/ROADMAP-DAY-DU.md` §5b (D-MVP-00; đọc ở `main` sau mốc `c086fe7`) đặt Phase 1 làm khối M4, có thêm một luật vượt dung sai kích hoạt việc tạo phiếu [REPO]. D1–D9 giữ nguyên: luật kích hoạt chỉ là một bên gọi lệnh (`HOP-DONG.md` §1.4). Phần còn mở là Principal `service` và khoá lệnh tất định (câu Q-P1-6).

## Quyết định

Mỗi quyết định có chi tiết và ca kiểm ở `HOP-DONG.md`; ADR chỉ ghi lựa chọn và lý do.

| # | Quyết định | Chi tiết |
|---|---|---|
| **D1** | **Phạm vi:** một lệnh `open_discrepancy_ticket`; một loại phiếu; chỉ **tạo**; chế độ B; chỉ số `issue_vs_measured`; kỳ là một ngày đã qua; không chữ tự do. | HOP-DONG §1, §5 |
| **D2** | **Lối vào:** yêu cầu có cấu trúc, gọi thẳng bằng hàm Go. **Không** qua luồng hỏi và S4 ở Phase 1. Không HTTP. Principal do bên gọi tin cậy dựng; **không nối lệnh ra ngoài trước Phase 5** (chưa có xác thực). | §1.3, §2 |
| **D3** | **Pipeline lệnh C0–C7 tất định**, 0 lời gọi model, đặt ở `core/pipeline` bằng tệp mới. Idempotency tách hai nửa: tra sổ lệnh ngay sau quyền; khoá duy nhất ở bước ghi. **Lệch thứ tự `CLAUDE.md` §14 và `docs/ROADMAP-DAY-DU.md` §5 L3: câu mở Q-P1-7.** | §3 |
| **D4** | **Lưu trữ:** ba bảng `commands`, `discrepancy_tickets`, `audit_log` (migration `0003`), RLS FORCE, vai `eais_app` chỉ SELECT và INSERT, không khoá ngoại tới bốn bảng mà `Load` thay thế. Ghi trong **một giao dịch** theo thứ tự phiếu, sổ lệnh (khoá ngoại thường tới phiếu), audit; đọc lại từ nơi ghi trước khi báo xong. | §7 |
| **D5** | **Chống trùng hai tầng:** khoá lệnh do bên gọi sinh, phạm vi (tenant, người gọi), kèm fingerprint nội dung; khoá nghiệp vụ (tenant, máy, chỉ số, ngày) trên phiếu đang mở. | §6 |
| **D6** | **Kết quả ba trị** ĐÃ LÀM / CHƯA BIẾT / THẤT BẠI. Không biết thì `UNKNOWN`. Không `DONE` nếu chưa đọc lại thấy phiếu. | §2.3, §9 |
| **D7** | **Quyền:** capability mới `tickets.create`, cộng `metrics.read` và phạm vi thực thể; pack kiểm nhất quán lúc nạp; một mã cho "không có" và "ngoài phạm vi". | §4 |
| **D8** | **Cấu hình trong pack, hành vi trong mã** (P12 của `docs/01-architecture.md`): tệp mới `tenants/construction/pack/tools.yaml` khai tên tool, capability, chỉ số được phép, `min_abs_diff`, `timeout`. | §4.3, §5 |
| **D9** | **Không tạo thư mục sớm**, không interface `Tool` chung: xem "Nơi đặt mã". | dưới |

## Nơi đặt mã

Theo `docs/01-architecture.md` §3 và `CLAUDE.md` §25: thư mục chỉ được tạo khi có mã thật cần boundary đó. Bảng này là kế hoạch cho lúc mã được mở; **hiện chưa tạo gì**.

| Đường dẫn | Hiện có? | Phase 1 làm gì |
|---|---|---|
| `core/contracts/command.go` | chưa (thư mục có) | chỉ kiểu: `CommandRequest`, `CommandResult`, trạng thái, mã lý do (như `AskRequest`; gói `contracts` chỉ chứa kiểu) |
| `core/pipeline/command.go` và test | chưa (thư mục có) | pipeline C0–C7; interface lưu trữ do pipeline khai tại chỗ dùng (thành ngữ Go của `contracts.go`). **Tệp mới; không sửa `ask.go`, `claim.go`** |
| `platform/pgstore/ticket.go` | chưa (thư mục có) | đơn vị ghi một giao dịch tách được lỗi COMMIT; tra sổ lệnh; đọc lại |
| `db/migrations/0003_phieu_chenh_lech.sql` | chưa | ba bảng, RLS, quyền, chỉ mục duy nhất, khoá ngoại thường từ sổ lệnh tới phiếu |
| `core/tenant/pack.go` | có | nạp và kiểm `tools.yaml`; ba luật nạp của HOP-DONG §4.3 |
| `tenants/construction/pack/tools.yaml` | chưa | khai tool (tên tệp theo `docs/01-architecture.md` §8) |
| `tenants/construction/pack/policy.yaml` | có | thêm `tickets.create` cho ba vai |
| test của lệnh | chưa | ca phát triển (tầng 1) của HOP-DONG §10 |
| `execution/`, `governance/`, `cmd/`, `domain/` | chưa | **không tạo**: workflow ở Phase 2; xác thực, `cmd/` ở Phase 5; adapter CRM ở Phase 4b |

Ràng buộc kiến trúc phải giữ:
- mã dùng chung không chứa tên tenant trong chuỗi literal (`architecture_test.go`, `TestSharedCodeHasNoTenantLiterals`); mã lệnh chỉ nói "chỉ số dẫn xuất" và "khoá thực thể", không nói "fuel" hay "construction";
- kiểu yêu cầu tên kết thúc `Request` và không mang tenant (`TestRequestTypesHaveNoTenantField`);
- `core/pipeline` là một gói Go chung với `ask.go`: tên hàm, kiểu mới mang tiền tố `command` để không đụng ký hiệu của vòng sửa S4.

## Phương án đã cân nhắc

### A. Đường lệnh đặt ở đâu

| Phương án | Ưu | Nhược |
|---|---|---|
| **A1 (chọn):** pipeline lệnh riêng ở `core/pipeline`, tệp mới | khớp kiến trúc đích (ACTION PIPELINE tách khỏi ASK, `docs/01` §2.1); không đụng `ask.go` đang trong vòng sửa S4; cổng ra đo biên hành động, không đo S4 | thêm một pipeline; bước tenant và quyền lặp ý của S1, S5 (giải bằng cách gọi lại `Policy.Allows`, `InScope`, không chép luật) |
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
| **E1 (chọn):** không; một hàm cụ thể cho một tool, đường nối là interface lưu trữ do pipeline khai | không trừu tượng hoá từ một ví dụ; không tạo thư mục theo tên đích | tool thứ hai (Phase 4b) phải trích ra; chi phí trích nhỏ vì ranh giới đã sạch |
| E2: tạo `execution/tools` và `Tool` (`Spec`, `Invoke`) như bảng hợp đồng đích (`docs/01` §6) | sẵn chỗ cho tool sau | `CLAUDE.md` §25: không tạo folder chỉ vì kiến trúc đích có tên; hợp đồng `Tool` đoán từ một ví dụ |

## Hệ quả

**Tích cực**
- Lát COMMAND đầu tiên, đo được **độc lập với S4**: cổng ra chỉ thấy biên hành động (quyền, chống trùng, đọc lại, audit).
- Tái dùng RLS, quyền, danh mục thực thể, phép tính có thẩm quyền; không viết lại.
- Sổ lệnh và khoá khử trùng là nền cho Phase 3 (sự kiện xử lý đúng một lần theo khoá, `docs/01` §4 L5).

**Tiêu cực và nợ có nhãn**
1. **Một tenant dùng** (xem mục A).
2. **Chỉ đúng ở chế độ B.** Khi ghi sang hệ khác (Phase 4b), "timeout sau khi ghi" cần thiết kế riêng (bản ghi lệnh cục bộ, mã ngoài để khử trùng): cần ADR riêng. Ghi ở HOP-DONG U8.
3. **`status` có thể có hai chủ** khi Phase 2 tới (B1): ADR của Phase 2 chốt.
4. **Audit chưa có chuỗi băm; khoá lệnh không hết hạn.** Hoãn Phase 5. Chưa có lỗi nào đòi (P7).
5. **Chưa có xác thực:** lệnh không được nối ra ngoài trước Phase 5.
6. **`tools.yaml` đổi `PackVersion`** (băm mọi tệp `.yaml` của pack, `core/tenant/pack.go:304`) [CODE]; các chỗ ghim phiên bản (bằng chứng của `g7run`) chưa kiểm: HOP-DONG U2.
7. **Phiếu là bản chụp tại lúc tính.** Lỗi im lặng của luồng L1 vẫn có thể tạo chênh lệch giả (`CONG-QUANIT.md` câu 3).
8. **Không tuyên bố giá trị:** câu 1 và 8 còn ✗ (D-RM-03).
9. **Lần đầu vai ứng dụng có đường ghi nghiệp vụ** ngoài `Load`: chỉ INSERT, không UPDATE, DELETE. Cổng ra có ca thử trực tiếp (A26).
10. **Bên kích hoạt tự động (khối M4)** cần Principal `service` có vai và khoá lệnh tất định. Nằm ngoài ADR này, ghi ở câu Q-P1-6 của hợp đồng.
11. **Đường hỏi vẫn coi câu lệnh là câu hỏi** (HOP-DONG §1.5, ca A25, [ĐO]): Phase 1 không chặn; nợ cho vòng S4 hoặc Phase 6.
12. **C4b lệch thứ tự `CLAUDE.md` §14 và ROADMAP L3** (Idempotency tách hai nửa; HOP-DONG §3): giá trị là ca A21 (gửi lại sau timeout nhận `DONE` phát lại). Chưa ai duyệt: câu Q-P1-7.

## Không làm

- Không viết mã trước khi H4 chạy (D-RM-04). Không sửa `ask.go`, `claim.go`, `intent.go`, `router.go`.
- Không nối kết quả lệnh vào mẫu câu của luồng hỏi; không sửa lời hứa `templates.yaml:16-17` ở Phase 1.
- Không thêm trường chữ tự do; không `reported_vs_issued`; không kỳ nhiều ngày.
- Không chặn câu lệnh viết bằng chữ tự do ở luồng hỏi (hiện được trả lời như câu hỏi: HOP-DONG §1.5); nợ cho vòng S4 hoặc Phase 6.
- Không gọi model; không interface `Tool`; không tạo `execution/`, `governance/`, `cmd/`, `domain/`.
- Không tự thay dòng ADR-0001: điều kiện đó do D-RM-04 của chủ dự án thay.

## Chứng minh (theo thứ tự)

1. **Điều kiện mở:** H4 đã chạy MỘT lần (vòng S4 đã đóng băng trước đó). **Đang chờ.**
2. **Hợp đồng được soát độc lập** (người khác với tác giả tài liệu này), và đạt, chủ dự án trả lời Q-P1-1…7 hoặc chấp nhận mặc định. (Lượt 1 đã làm, KHÔNG ĐẠT, đã sửa; cần soát lại.)
3. **Bộ giữ lại độc lập** (HOP-DONG §12): người không đọc mã soạn, đóng băng bằng cam kết băm, **trước** mã. Xong bước 1, 2 và 3 thì ADR-0011 chuyển từ ĐỀ XUẤT sang CHỐT (mục "Điều kiện chuyển" dưới).
4. **Hai thử nghiệm nhỏ, không phải mã sản phẩm,** để trả lời U1 (lỗi ở ranh giới COMMIT của trình điều khiển trên PostgreSQL thật) và U2 (nơi ghim `PackVersion`); ghi kết quả vào hợp đồng.
5. **Test đỏ trước:** ca A01–A27 của tầng 1 viết trước mã; nhìn từng ca đỏ đúng lý do.
6. **Migration `0003` và test cách ly:** canary kiểu `TestRLSCanaryB` cho ba bảng mới; `eais_app` bị từ chối UPDATE, DELETE và dòng sổ lệnh trỏ tới phiếu không có (A26).
7. **Mã nhỏ nhất làm ca xanh,** theo thứ tự C0 → C7, mỗi chặng một commit, không chạm đường hỏi.
8. **Chạy bộ giữ lại độc lập MỘT lần;** báo theo từng cổng X1–X8, không gộp một con số.
9. **Đột biến M1–M23** (HOP-DONG §11.2): đột biến sống sót thì thêm ca hoặc ghi tương đương kèm lý do.
10. **Hồi quy đường hỏi:** bộ 117 / H1 / H3 / H4, đường bộ nhớ và PostgreSQL, so theo tên và theo từng câu trả lời: 0 dòng khác.
11. **Cổng chung:** `go vet ./...`, `gofmt -l`, `go test -count=1 ./...` có PostgreSQL, cổng danh tính.
12. **Soát đối kháng độc lập,** chỉ đọc: tìm cách tạo hai phiếu, hoặc một phiếu mà kết quả báo `FAILED` hay `UNKNOWN` sai. Finding quay lại sửa, rồi soát lại.
13. **Báo kết quả** là "cổng ra kỹ thuật Phase 1 đạt (hay không đạt) trên dữ liệu giả lập, chưa có người nghiệp vụ duyệt", không kèm tuyên bố giá trị.

## Điều kiện chuyển ĐỀ XUẤT → CHỐT

Bước 1, 2 và 3 của "Chứng minh" xong. Chưa trả lời được thì ADR giữ nguyên trạng thái ĐỀ XUẤT và không có mã.

## Rủi ro

- Hợp đồng sai ở chỗ chưa kiểm (U1–U8 của HOP-DONG §15), mỗi điểm có cách kiểm.
- Đã có một lượt soát độc lập (22/10/2026): KHÔNG ĐẠT, đã sửa; chưa soát lại sau sửa. Tác giả ADR cũng là tác giả hợp đồng và cổng QuânIT, nên phần sửa này là SELF REVIEW cho tới khi soát lại.
- Đảo được: ADR và hợp đồng là giấy; bỏ chúng không làm đổi dòng mã nào.
