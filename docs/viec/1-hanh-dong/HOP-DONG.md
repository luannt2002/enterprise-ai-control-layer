# Hợp đồng việc — Phase 1: tạo một phiếu kiểm tra chênh lệch nhiên liệu

> **Trạng thái: ĐỀ XUẤT, GIẤY, chưa áp mã.**
> - Mã Giai đoạn II chỉ mở sau khi chạy xong H4 (D-RM-04 = A, `docs/CHO-QUYET-DINH.md` mục 0, dòng 22/10 11:2x).
> - Soát độc lập lượt 1 (22/10/2026): KHÔNG ĐẠT, đã sửa theo soát. Soát lại sau sửa (22/10/2026): ĐẠT CÓ GHI CHÚ, bốn ghi chú mức S3 (L3C-01…04). Bản này sửa cả bốn và thêm cửa vào cho kịch bản demo của khối M4 (§2.6, câu Q-P1-8); **phần sửa này chưa được soát**. Chưa có người nghiệp vụ duyệt (D-RM-03 = "chưa có").
> - Chỗ sửa của bốn ghi chú: L3C-01 (A04, M21, định nghĩa "đơn vị ghi phiếu": §10, §11.2) · L3C-02 (ca đồng thời, nhiều người gọi: §12.1) · L3C-03 (cửa `cmd/eais`, số migration: §1.3, §2.6, §7.4) · L3C-04 (nơi đặt mã dưới `execution/commands`: ADR-0011, "Nơi đặt mã").
>
> **Viết cho:**
> - chủ dự án: duyệt hợp đồng và trả lời câu mở ở §14;
> - phiên BUILD: viết mã đúng hợp đồng khi được mở;
> - người soạn bộ ca độc lập (§12): bản hợp đồng này là đầu vào duy nhất của họ.
>
> **Đi kèm:** `docs/viec/1-hanh-dong/CONG-QUANIT.md` (9 câu cổng QuânIT) và `docs/adr/0011-phase1-hanh-dong-co-kiem-soat.md` (nơi đặt mã, phương án đã cân nhắc, thứ tự chứng minh).
>
> **Thứ bậc khi mâu thuẫn:** như đầu `docs/ROADMAP-DAY-DU.md`. Chỗ tệp này tinh chỉnh `CLAUDE.md` §14 được nêu rõ ở §3.
>
> **Ngày:** 22/10/2026. **Mốc đọc mã:** `c086fe7`. Tra một dòng trích bằng `git show c086fe7:<tệp>`. Nhánh `main` có thể đã đổi các tệp được trích (ví dụ `ask.go`, `router.go`, `intent.go`, pack), nên số dòng ở `main` có thể lệch; khi gộp phải đối chiếu lại các dòng trích. **Mốc đọc làn 0** (cửa `cmd/eais`, vết, migration): nhánh làn 0 tại `b5be82e`, chưa gộp lúc viết và còn đang sửa theo soát; các dòng trích ghi "làn 0" tra bằng `git show b5be82e:<tệp>` nếu mốc còn trong lịch sử, nếu không thì đối chiếu lại ở `main` sau khi gộp.
>
> **Nhãn bằng chứng:** [ĐO] người viết tự chạy lệnh · [REPO] tài liệu repo tự ghi, chưa chạy lại · [CODE] đọc thấy trong mã (`tệp:dòng` tại `c086fe7`) · [SUY] suy luận của người viết · [ĐỀ XUẤT] chưa quyết, đảo được · "chưa đo" = chưa chạy lệnh nào.
>
> **Dữ liệu:** mọi tên máy, dự án, vai, con số trong tệp này là giả lập (tenant `abc-infra` hư cấu). Kết luận: **trên dữ liệu giả lập, chưa có người nghiệp vụ duyệt**.

---

## 0. Tóm tắt

**Một lệnh:** `open_discrepancy_ticket`. Một người có quyền đưa vào {khoá lệnh, máy, chỉ số, ngày}, qua một route lệnh của `cmd/eais` (§2.6). EAIS tự tính chênh lệch từ số liệu có thẩm quyền và tạo **đúng một** phiếu ở trạng thái `OPEN`.

**Mười một bảo đảm** (bất biến, mỗi cái có ca âm ở §10 và đột biến ở §11.2):

| # | Bảo đảm |
|---|---|
| I1 | Tenant đến từ Principal, không từ thân yêu cầu; bảng mới có RLS FORCE. |
| I2 | Không có model, không có chữ tự do: 0 lời gọi model; đầu vào có cấu trúc. |
| I3 | Không bao giờ báo `DONE` nếu chưa đọc lại thấy phiếu trong chính lượt đó. |
| I4 | Phiếu, sổ lệnh và audit `EXECUTED` cùng có hoặc cùng không (một giao dịch). |
| I5 | Mỗi (tenant, người gọi, `command_id`) tối đa **một lần ghi** (các lần gửi lại nhận `DONE` cùng mã phiếu, `replayed` = true; đúng một lần `replayed` = false); mỗi (tenant, máy, chỉ số, ngày) tối đa một phiếu `OPEN`. |
| I6 | Quyền đóng mặc định: thiếu quyền tạo, thiếu quyền đọc hai vế, hoặc ngoài phạm vi ⇒ 0 ghi (§10): chỉ còn dòng audit `REFUSED`. |
| I7 | Giá trị chênh lệch do hệ tính từ fact có thẩm quyền; bên gọi không đưa số; không chọn số khi hai nguồn ngang thẩm quyền lệch. |
| I8 | Ghi chỉ-thêm: vai ứng dụng không có quyền UPDATE, DELETE trên ba bảng mới. |
| I9 | Không đổi đường hỏi: bộ 117 / H1 / H3 / H4 không đổi một chữ nào; bốn route sẵn có của `cmd/eais` (`/v1/ask`, `/v1/ingest`, `/v1/trace/{id}`, `/healthz`) không đổi hành vi. |
| I10 | Không biết thì nói `UNKNOWN`; không làm tròn thành `FAILED` hay `DONE`. |
| I11 | Mọi lượt lệnh đã xác thực để lại vết từng bước xem được bằng `eais trace`, kể cả lượt bị từ chối hay lỗi (trừ khi chính kho vết hỏng: kết quả lệnh không đổi, có bộ đếm, §7.5); vết không lộ điều mà kết quả trả về không lộ. |

**Ngoài phạm vi:** sửa, huỷ, đổi trạng thái, giao người nhận, thông báo (Phase 2); xử lý sự kiện đầy đủ: đính chính, kế hoạch, đến muộn (Phase 3; luật vượt dung sai của khối M4 chỉ là một bên gọi lệnh, xem §1.4); xác thực thật và nghe ngoài máy dev (Phase 5; cửa vào `cmd/eais` của làn 0 chỉ chạy ở chế độ `-dev`, §1.3); CRM của khách (Phase 4b); câu lệnh viết bằng chữ tự do (Phase 6).

---

## 1. Việc và phạm vi

### 1.1 Việc

- **Chế độ B:** EAIS giữ sự thật, quyền và giao dịch của phiếu, vì doanh nghiệp tham chiếu chỉ có Zalo, Excel, giấy.
- **Phiếu** ghi lại rằng "chênh lệch giữa dầu kho xuất và dầu tiêu thụ đo được của máy M trong ngày D cần được kiểm tra", kèm giá trị tại lúc tính và bằng chứng.
- **Chênh lệch** là chỉ số dẫn xuất `issue_vs_measured` = `issued_fuel` − `measured_consumption` (`tenants/construction/pack/metrics.yaml:50-51`) [CODE]. Dương nghĩa là kho xuất nhiều hơn lượng đo được.
- **Ví dụ minh hoạ** (dữ liệu hạt giống giả lập): EX-03 hôm đó kho xuất 3.850 L, cảm biến đo 3.782 L, chênh lệch 68 L. Hai vế lấy theo ví dụ của `CLAUDE.md` §20; test `tenants/construction/construction_test.go:105-106` chỉ kiểm hiệu "68" [CODE].

### 1.2 Làm và không làm

| Làm trong Phase 1 | Không làm (phase) |
|---|---|
| một lệnh `open_discrepancy_ticket`: tạo phiếu | sửa, huỷ, xoá, đổi trạng thái phiếu (Phase 2) |
| một loại phiếu, một trạng thái `OPEN` | giao người nhận, hạn chờ, thông báo, human task (Phase 2) |
| chỉ số `issue_vs_measured` | `reported_vs_issued` (vế trái là chat), `billed_fuel` (sau; xem §5) |
| kỳ = đúng một ngày đã qua | kỳ nhiều ngày, tháng (cần ràng buộc loại trừ chồng lấn) |
| đầu vào có cấu trúc | chữ tự do; trích tham số từ câu hỏi (C2, S4); mẫu câu "đã tạo" ở luồng hỏi; chặn câu lệnh ở luồng hỏi (nợ, §1.5) |
| Principal lấy từ token của `cmd/eais` (ở dev: token công khai, chỉ loopback); một route lệnh (§2.6) | xác thực thật, khoá API, nghe ngoài máy dev (Phase 5); route cho lệnh khác (Phase 2 trở đi) |
| PostgreSQL một nút, một giao dịch | CRM hay API ngoài (chế độ A, Phase 4b); hàng đợi, outbox, worker nền |
| audit chỉ-thêm | chuỗi băm, chính sách giữ và hết hạn audit, sao lưu (Phase 5) |
| 0 lời gọi model | model đề xuất lệnh (cổng đóng, `CLAUDE.md` §19) |
| đọc lại nội bộ để xác nhận | liệt kê, tìm phiếu cho người dùng (Phase 2 / 5) |
| vết từng bước của lượt lệnh, xem bằng `eais trace` (§7.5) | so hai lượt lệnh bằng `eais trace diff` (chưa kiểm, U9) |

### 1.3 Giả định danh tính (đọc kỹ)

- Principal lấy từ **token** ở tiêu đề `Authorization` của `cmd/eais` (làn 0, khối M1), không từ thân yêu cầu, không từ query. Ở chế độ `-dev` token là thông tin công khai suy từ `world.yaml` (dạng `dev-<tenant>-<principal>`, Principal loại `user`); `serve` từ chối chạy khi thiếu `-dev`, chỉ nghe loopback và chặn `Host` không phải loopback (`cmd/eais/auth.go:48`, `cmd/eais/main.go:163-180`, `cmd/eais/server.go:120`, `:138` tại `b5be82e`) [CODE, làn 0, chưa gộp lúc viết]. Xác thực thật vẫn chưa có (`docs/01-architecture.md` §4 L7, dòng "xác thực: chưa có") [REPO].
- Ai biết token dev của một vai thì mang vai đó (ví dụ `director`). Vì vậy Phase 1 **chứng minh phân quyền** (có Principal đúng thì quyền đúng), **không chứng minh xác thực**.
- Hệ quả: route lệnh (§2.6) chỉ sống trong chế độ `-dev` của `serve` (loopback, token công khai). Lệnh này **không được mở ra ngoài máy dev, không được nối kênh chat** trước Phase 5. Ghi vào ADR-0011 như một ràng buộc.

### 1.4 Liên hệ với MVP khối M4

Sau mốc `c086fe7`, `docs/ROADMAP-DAY-DU.md` §5b (D-MVP-00, 22/10; đề xuất của phiên điều phối) đặt Phase 1 thành **khối M4** của MVP: "phát hiện chênh lệch → tạo đúng một phiếu kiểm tra", trong đó một **luật vượt dung sai kích hoạt** việc tạo phiếu [REPO, bản trên `main`; chưa có ở `c086fe7`]. Hợp đồng này dùng được cho M4, với ba điều phải nói rõ:
- **Bên kích hoạt chỉ là một bên gọi.** Luật vượt dung sai (so sánh một bước, đúng P12) gọi lệnh `open_discrepancy_ticket`; các chặng C0–C7 giữ nguyên. `min_abs_diff` (R4) là chỗ đặt ngưỡng của luật.
- **Cần Principal loại `service`** có vai riêng với `tickets.create` và `metrics.read`. §4.2 chưa có vai này (đóng mặc định), và token dev của làn 0 chỉ cấp Principal loại `user` (`cmd/eais/auth.go:55` [CODE, làn 0]), nên bên kích hoạt tự động còn cần một nguồn Principal `service`. Vai, phạm vi và nguồn của nó là câu mở Q-P1-6. Kịch bản demo của khối M4 (§2.6.4) do một **người** gọi nên không cần Principal `service`: Q-P1-6 chỉ chặn bên kích hoạt tự động.
- **Khoá lệnh phải tất định** khi không có người giữ UUID, ví dụ UUID dẫn xuất từ (tenant, máy, chỉ số, ngày) [ĐỀ XUẤT]. Khi đó luật chạy lại bao nhiêu lần cho cùng một ngày cũng chỉ ra một phiếu: hai tầng chống trùng (§6) làm việc này mà không cần người canh.

Không đổi: kỳ một ngày, chỉ số `issue_vs_measured`, ba trị kết quả, đọc lại, audit. Xử lý đính chính, kế hoạch, đến muộn vẫn là Phase 3.

### 1.5 Hiện trạng: đường hỏi coi câu lệnh là câu hỏi

Người dùng gõ một câu lệnh vào luồng hỏi hôm nay **không được báo là chưa có phiếu nào**: động từ "Tạo phiếu" bị bỏ qua im lặng và câu được trả lời như câu hỏi tính toán. Cùng họ với CON-F-020 (lệnh bị coi là câu hỏi, `docs/tenants/construction/07-failure-catalog.md:76`) nhưng chưa có lời "đã tạo". Chưa xếp mức nghiêm trọng.

**[ĐO]** bằng một test thăm dò tạm, không commit: pack THẬT, dữ liệu hạt giống của `tenants/construction/construction_test.go`, "hôm nay" cố định 19/10/2026, năm vai (`director`, `accountant`, `project_manager`, `site_engineer`, `driver`) nhân sáu câu, 30 lượt. Kết quả **giống nhau** ở `c086fe7`, `1e5457b` và `7d72e33` (`origin/main` lúc đo cuối, 22/10/2026):

| Câu | Kết quả của bốn vai có `metrics.read` |
|---|---|
| "Tạo phiếu kiểm tra chênh lệch máy EX-03 hôm qua" (câu của A25) | route `CALCULATION`, ý định `fuel_variance`: "Chưa có dữ liệu Chênh lệch dầu kho xuất và dầu tiêu thụ đo được của Máy xúc EX-03 ngày 18/10/2026." |
| như trên, thêm chữ "dầu" | y hệt dòng trên |
| "Tạo phiếu kiểm tra chênh lệch máy EX-03 hôm nay" | route `CALCULATION`, ý định `fuel_variance`: "Chênh lệch dầu kho xuất và dầu tiêu thụ đo được của Máy xúc EX-03 ngày 19/10/2026: 68 L." |
| "Tạo phiếu kiểm tra máy EX-03" (đầu vào của CON-T-010) | route `REJECT`, không ý định: "Câu hỏi này nằm ngoài phạm vi trợ lý vận hành của ABC Infrastructure." |

- Mỗi câu lệnh về chênh lệch cho **đúng** câu trả lời của câu hỏi thường cùng ngày ("Chênh lệch dầu cấp và tiêu thụ của EX-03 hôm qua?" và "...hôm nay?"). Không câu trả lời nào trong 30 lượt chứa chữ "đã tạo".
- `driver` (không có `metrics.read`): mọi câu về chênh lệch, kể cả hai câu hỏi thường, nhận route `REJECT`, "Bạn không có quyền xem dữ liệu này."; câu "Tạo phiếu kiểm tra máy EX-03" giống các vai khác.
- **Hệ quả cho Phase 1:**
  1. Đường hỏi **không** từ chối câu lệnh khi câu còn khớp một ý định. A25 vì vậy không đòi "từ chối hoặc hỏi lại" (§10). Muốn chặn động từ lệnh ở đó phải sửa `ask.go`, `router.go`, `intent.go`: trái I9 và mục "Không làm" của ADR-0011. Ghi nợ cho vòng S4 hoặc Phase 6.
  2. Phase 1 chỉ bảo đảm ba điều ở A25: đường hỏi không nói "đã tạo", không ghi phiếu, và không đổi so với trước khi có mã Phase 1 (A25, A27).
  3. Vì vậy lượt "Tạo phiếu kiểm tra" của kịch bản demo **không** đi qua `/v1/ask`: đó là một yêu cầu có cấu trúc tới route lệnh riêng (§2.6.1, câu Q-P1-8).
- **Chưa đo:** câu lệnh khác ngoài sáu câu trên; mốc sau `7d72e33`.

---

## 2. Đầu vào và đầu ra

### 2.1 Yêu cầu `CommandRequest` (tên đề xuất)

| Trường | Kiểu | Bắt buộc | Luật |
|---|---|---|---|
| `command_id` | UUID đúng dạng 8-4-4-4-12, **không kiểm phiên bản** | có | bên gọi sinh **mỗi ý định một lần**, dùng lại nguyên khi gửi lại (§6); ngẫu nhiên (v4, v7) hoặc dẫn xuất tất định từ (tenant, máy, chỉ số, ngày) cho khối M4 (§1.4) |
| `subject_key` | khoá chuẩn trong danh mục, ví dụ `EX-03` | có | không bí danh, không so khớp mờ; loại thực thể phải nằm trong `entity_types` của hai vế (hiện: `equipment`) |
| `metric` | mã chỉ số dẫn xuất | có | phải thuộc danh sách của tool trong pack; Phase 1: `issue_vs_measured` |
| `day` | ngày `YYYY-MM-DD`, theo múi giờ tenant | có | phải là ngày đã qua, tức trước "hôm nay" của tenant |

- **Không có** trường nào khác. Giải mã chặt: trường lạ ⇒ `INVALID_REQUEST`. Đặc biệt **không có**: tenant, người gọi, vai, phạm vi, giá trị chênh lệch, trạng thái, mã phiếu, thời gian, chữ tự do (ghi chú, lý do).
- Tên kiểu kết thúc bằng `Request` để cổng sẵn có `TestRequestTypesHaveNoTenantField` (`architecture_test.go:80`) tự phủ [CODE]. Cổng quét cả `execution` và `cmd` (`sharedRoots`, `architecture_test.go:19` tại `5d69d24`) [CODE], nên kiểu `Request` của gói `execution/commands` và của lớp HTTP đều được phủ, không cần sửa cổng.
- Độ dài và ký tự: `command_id` đúng dạng UUID; không trường nào chứa ký tự NUL (cùng tinh thần S0, `core/pipeline/ask.go:1058-1070`) [CODE]. Ký tự NUL bị chặn ở C0 bằng `INVALID_REQUEST`, trước mọi tra cứu; mọi chuỗi lạ khác là khoá hợp lệ về cấu trúc và xử lý theo §4.4.

### 2.2 Hệ tự điền

| Thứ | Lấy từ |
|---|---|
| tenant | Principal (S1 của luồng hỏi làm đúng việc này: `ask.go:89-95`) |
| người tạo | `Principal.ID` |
| mã phiếu | hệ sinh (UUID); bên gọi không chọn |
| hai vế, chênh lệch, đơn vị, cờ `approx`, công thức, bản chụp bằng chứng | `Catalog.Derive` (`data/sourceoftruth/query.go:251`) trên fact có thẩm quyền |
| thời gian tạo | đồng hồ của cơ sở dữ liệu |
| phiên bản pack, phiên bản mã | `Runtime.PackVersion`; `codeVersion` (`ask.go:200`) |
| `trace_id` | hệ sinh, như `AskResponse.TraceID` |

### 2.3 Kết quả `CommandResult` (tên đề xuất)

| Trường | Giá trị | Ghi chú |
|---|---|---|
| `status` | `DONE` (ĐÃ LÀM) · `UNKNOWN` (CHƯA BIẾT) · `FAILED` (THẤT BẠI) | ba trị của luồng L3 (`docs/ROADMAP-DAY-DU.md` §5) |
| `reason_code` | bảng §2.4 | rỗng khi `DONE` |
| `fault` | nhãn lỗi `docs/CONVENTIONS.md` §4.3 | khi `status` ≠ `DONE` |
| `retry` | `none` · `same_key` · `fix_request` · `later` | xem dưới |
| `replayed` | true / false | chỉ có nghĩa khi `DONE` |
| `ticket` | mã phiếu, trạng thái, và (khi `DONE`) các trường §7.4 | cũng có khi `DUPLICATE_OPEN_TICKET` (chỉ mã và trạng thái) |
| `trace_id` | chuỗi | để tra vết bằng `eais trace` (§7.5); rỗng khi không lưu được vết |

**Ba trị, định nghĩa chặt:**
- `DONE`: phiếu **đã được ghi** và **đã được đọc lại thấy đúng** trong chính lượt này (kể cả khi phát lại).
- `FAILED`: **biết chắc** lượt này không để lại phiếu mới (bị từ chối, hoặc ghi không thành và đã xác nhận rollback).
- `UNKNOWN`: không biết phiếu đã được ghi hay chưa, hoặc đã ghi nhưng đọc lại không xác nhận được. Bên gọi gửi lại cùng `command_id`.

**`retry`:**
- `none`: không thử lại;
- `same_key`: gửi lại nguyên văn với cùng `command_id`;
- `fix_request`: sửa yêu cầu rồi gửi;
- `later`: dữ liệu có thể đổi, gửi lại sau.

Khoá chỉ bị tiêu thụ khi lệnh đã được **ghi** (§6.2), nên lệnh bị từ chối có thể dùng lại cùng `command_id`.

### 2.4 Mã lý do

Từ vựng dùng lại tên sẵn có ở `core/contracts/contracts.go:298-305` khi nghĩa trùng (`PERMISSION_DENIED`, `INSUFFICIENT_EVIDENCE`, `HUMAN_REVIEW_REQUIRED`) [CODE].

| `reason_code` | `status` | `fault` | `retry` | Khi nào |
|---|---|---|---|---|
| (rỗng) | `DONE` | — | `none` | phiếu mới, hoặc phát lại |
| `INVALID_REQUEST` | `FAILED` | `input` | `fix_request` | thiếu, thừa, sai trường; ký tự NUL trong một trường; chỉ số ngoài danh sách; ngày sai dạng hoặc không phải ngày đã qua; loại thực thể không hợp lệ cho chỉ số |
| `IDEMPOTENCY_KEY_REUSED` | `FAILED` | `input` | `fix_request` | cùng khoá, khác nội dung (§6) |
| `PERMISSION_DENIED` | `FAILED` | `policy` | `none` | thiếu `tickets.create`; thiếu `metrics.read`; ngoài phạm vi; máy không tồn tại (cùng một mã, §4.4); Principal thuộc tenant khác |
| `INSUFFICIENT_EVIDENCE` | `FAILED` | `data` | `later` | thiếu một vế |
| `HUMAN_REVIEW_REQUIRED` | `FAILED` | `data` | `later` | một vế có hai nguồn ngang thẩm quyền lệch quá dung sai; cần người quyết trước |
| `NO_DISCREPANCY` | `FAILED` | `data` | `none` | chênh lệch không vượt ngưỡng (mặc định: bằng 0) |
| `DUPLICATE_OPEN_TICKET` | `FAILED` | `data` | `none` | đã có phiếu `OPEN` cho cùng (máy, chỉ số, ngày); trả mã phiếu cũ |
| `STORE_UNAVAILABLE` | `FAILED` | `infra:postgres` | `same_key` | hạ tầng lỗi **trước** khi gửi COMMIT, đã xác nhận không ghi |
| `COMMIT_UNKNOWN` | `UNKNOWN` | `infra:postgres` | `same_key` | lỗi hoặc hết hạn **từ lúc gửi COMMIT** mà không biết kết quả |
| `READBACK_UNAVAILABLE` | `UNKNOWN` | `infra:postgres` | `same_key` | đã commit; đọc lại lỗi hoặc hết hạn |
| `VERIFY_MISMATCH` | `UNKNOWN` | `code` | `none` | đã commit; đọc lại không thấy hoặc lệch trường: cần người xem audit |
| `INTERNAL_ERROR` | `FAILED` nếu lỗi trước BEGIN; `UNKNOWN` nếu sau BEGIN mà không xác nhận rollback | `code` | `none` | lỗi mã không lường trước |

HTTP: mã HTTP của lệnh lấy từ nhãn `fault` của kết quả qua `contracts.HTTPStatus` (`core/contracts/fault.go:97`) [CODE]; bảng ở §2.6.2. Thân phản hồi luôn là `CommandResult`: bên gọi dựa vào `status` và `reason_code` trong thân, không dựa vào mã HTTP.

### 2.5 Hình dạng minh hoạ

Chưa chạy; chỉ để người đọc thấy hình dạng. Mã UUID là giả.

Yêu cầu:

```json
{
  "command_id": "6f1c0d52-0000-4000-8000-000000000001",
  "subject_key": "EX-03",
  "metric": "issue_vs_measured",
  "day": "2026-10-21"
}
```

Kết quả `DONE`:

```json
{
  "status": "DONE",
  "reason_code": "",
  "retry": "none",
  "replayed": false,
  "ticket": {
    "id": "b7d3c1aa-0000-4000-8000-0000000000a1",
    "status": "OPEN",
    "subject_key": "EX-03",
    "metric": "issue_vs_measured",
    "day": "2026-10-21",
    "left":  { "metric": "issued_fuel", "value": "3850", "unit": "L" },
    "right": { "metric": "measured_consumption", "value": "3782", "unit": "L" },
    "diff": "68",
    "approx": false
  },
  "trace_id": "…"
}
```

| Tình huống | `status` | `reason_code` | `replayed` | `ticket` |
|---|---|---|---|---|
| gửi lại cùng khoá, cùng nội dung | `DONE` | rỗng | true | cùng mã phiếu |
| khác khoá, phiếu tương đương còn mở | `FAILED` | `DUPLICATE_OPEN_TICKET` | — | mã và trạng thái phiếu cũ |
| mất kết nối sau khi gửi COMMIT | `UNKNOWN` | `COMMIT_UNKNOWN` | — | không có |

### 2.6 Cửa vào qua `cmd/eais` và kịch bản demo của khối M4

Lệnh ca MVP của chủ dự án (22/10) đòi một kịch bản demo hai lượt: hỏi "Máy 03 hôm qua lệch bao nhiêu dầu?", rồi yêu cầu "Tạo phiếu kiểm tra", rồi xem `eais trace` từng bước của **cả hai** lượt. Mục này chốt lượt thứ hai đi vào hệ bằng cửa nào và để lại vết gì. Mọi thứ ở đây là **thiết kế, chưa chạy** (mã Phase 1 chưa có); đường dẫn, tên hàm, tên kiểu là đề xuất. Kết luận: **trên dữ liệu giả lập, chưa có người nghiệp vụ duyệt**.

#### 2.6.1 Chọn cửa [ĐỀ XUẤT]

| Phương án | Ưu | Nhược | Chọn |
|---|---|---|---|
| **a. Route lệnh riêng của `cmd/eais`:** `POST /v1/commands/open_discrepancy_ticket`, thân là `CommandRequest` (§2.1), Principal từ token | khớp D2 và §2.1 (đầu vào có cấu trúc, không chữ tự do); không đổi một dòng của luồng hỏi (I9); không cần C2, không cần ngữ cảnh của lượt trước; chỉ thêm một route và một handler, tái dùng xác thực, giải mã chặt và lưu vết của làn 0; tên lệnh nằm ở đường dẫn nên thân giữ đúng bốn trường | lượt thứ hai không phải câu chữ gõ tay: người demo đưa máy, chỉ số, ngày dưới dạng khoá | **có** |
| b. Câu chữ "Tạo phiếu kiểm tra" vào `/v1/ask`, hệ nhận ra đó là lệnh | đúng cảm giác hội thoại | đòi C2 (hiểu lệnh, trích tham số từ chữ tự do) và đòi biết "máy 03 hôm qua" của lượt trước là máy nào, ngày nào (ngữ cảnh hội thoại, chưa có); phải sửa `ask.go`, `router.go`, `intent.go`: trái I9 và mục "Không làm" của ADR-0011, đụng vòng S4 đã khoá; nợ đã ghi ở §1.5 (S4 hoặc Phase 6) | không |
| c. Chỉ hàm Go hoặc lệnh con CLI | nhỏ nhất | lệnh CLI chạy bằng quyền của người vận hành có DSN, không có Principal (`cmd/eais/cli_trace.go`, `traceOpen` [CODE, làn 0]), nên không có người để kiểm quyền; không phải cửa mà chủ dự án dùng để demo | không |

Chọn a vì đó là cách nhỏ nhất vẫn khớp hợp đồng: một route, không đổi luồng hỏi, không cần hiểu chữ tự do. Gõ "Tạo phiếu kiểm tra" vào `/v1/ask` vẫn được trả lời như câu hỏi (§1.5, A25); kịch bản demo **không** đi đường đó. Muốn demo bằng câu chữ thì phải chọn phương án b: câu Q-P1-8 (§14).

#### 2.6.2 Route lệnh [ĐỀ XUẤT]

| Mục | Quy định |
|---|---|
| Đường | `POST /v1/commands/open_discrepancy_ticket`. Mỗi tool khai trong `tools.yaml` có một route, đăng ký lúc khởi động; tên ở đường dẫn là tên tool. Tên lạ rơi vào 404 mặc định của mux, trước xác thực: không audit, không vết (chưa có yêu cầu hợp lệ). `/v1/commands` là namespace đích của `docs/01-architecture.md` §3 (`platform/httpapi`) [REPO] |
| Xác thực | y hệt `/v1/ask`: tiêu đề `Authorization: Bearer <token>`, Principal từ token. Thiếu, sai hay lạ token: 401 cùng một phản hồi, **không** có vết (chưa có tenant; chỉ tăng bộ đếm `Unauthenticated` của làn 0) [CODE, làn 0] |
| Thân | `CommandRequest` (§2.1), giải mã chặt như `decodeAsk` (`cmd/eais/server.go:317` [CODE, làn 0]): `Content-Type: application/json`, có trần kích thước, trường lạ bị từ chối (kể cả `tenant`, `principal`, `role`), không có nội dung thừa. Tenant, người gọi, vai, phạm vi chỉ đến từ token (C1) |
| Kết quả | thân luôn là `CommandResult` (§2.3) cho mọi lượt đã xác thực, kể cả `INVALID_REQUEST` do lớp giải mã; tiêu đề `X-Trace-Id` mang `trace_id` khi lưu được vết |
| Mã HTTP | `DONE`, kể cả phát lại: 200. Còn lại lấy từ `fault` qua `contracts.HTTPStatus`, không lập bảng riêng: `input` 400 (sai `Content-Type` 415, thân quá lớn 413, như `/v1/ask`); `policy` 403; `data` 409; `infra:*` 503; `code` 500. Theo bảng §2.4: 400 = `INVALID_REQUEST`, `IDEMPOTENCY_KEY_REUSED`; 403 = `PERMISSION_DENIED`; 409 = `INSUFFICIENT_EVIDENCE`, `HUMAN_REVIEW_REQUIRED`, `NO_DISCREPANCY`, `DUPLICATE_OPEN_TICKET`; 503 = `STORE_UNAVAILABLE`, `COMMIT_UNKNOWN`, `READBACK_UNAVAILABLE`; 500 = `VERIFY_MISMATCH`, `INTERNAL_ERROR` |
| Chế độ | chỉ khi `serve -dev` (§1.3) |

#### 2.6.3 Hai lượt, hai cửa, hai vết

| Lượt | Cửa | Pipeline | Vết |
|---|---|---|---|
| 1. hỏi "Máy 03 hôm qua lệch bao nhiêu dầu?" | `POST /v1/ask` | luồng hỏi S0–S10 (không đổi) | 11 bước S0…S10 (làn 0) |
| 2. yêu cầu tạo phiếu | `POST /v1/commands/open_discrepancy_ticket` | pipeline lệnh C0–C7 | 9 bước C0…C7 (§7.5) |

#### 2.6.4 Kịch bản demo [ĐỀ XUẤT, chưa chạy]

Chạy trên server dev của làn 0 (`scripts/chay-thu.sh`, hoặc từng bước ở README), đồng hồ `-now world`, dữ liệu giả lập của làn 0. Người gọi là **một người dùng** có `tickets.create` và `metrics.read` (ví dụ vai `accountant` hoặc `director`), nên demo **không** cần Principal `service` (câu Q-P1-6). `<tenant>` là tenant của pack.

| Bước | Việc | Chủ dự án thấy |
|---|---|---|
| 1 | hỏi: `POST /v1/ask` với `text` = "Máy 03 hôm qua lệch bao nhiêu dầu?" | câu trả lời của luồng hỏi và `trace_id` T1 |
| 2 | `eais trace -tenant <tenant> T1` | bảng 11 bước S0…S10 của làn 0, đánh dấu bước lệch nếu có |
| 3 | yêu cầu: `POST /v1/commands/open_discrepancy_ticket`, thân `{"command_id": <UUID mới>, "subject_key": "EX-03", "metric": "issue_vs_measured", "day": <ngày hôm qua của tenant, YYYY-MM-DD>}` | `DONE`, mã phiếu, trạng thái `OPEN`, hai vế, chênh lệch, `approx`, `trace_id` T2 |
| 4 | `eais trace -tenant <tenant> T2` | bảng 9 bước C0…C7; lượt `DONE` không có bước lệch |
| 5 (thử phá, tuỳ chọn) | gửi lại **nguyên văn** bước 3 | `DONE`, `replayed` = true, **cùng** mã phiếu; vết mới: C4b ghi "phát lại", C5 và C6 `skipped` |
| 6 (thử phá, tuỳ chọn) | gửi lại với `command_id` **mới** | `FAILED` `DUPLICATE_OPEN_TICKET` kèm mã phiếu cũ; vết dừng ở C5 |

Lượt 2 **tính lại** chênh lệch ở C5 (R7), không chép từ câu trả lời của lượt 1: lượt lệnh không mang ngữ cảnh nào từ lượt hỏi, và `subject_key`, `day` do người gọi đưa dưới dạng khoá. Hai điều chủ dự án tự đối chiếu [SUY, chưa đo]: (i) nếu lượt 1 trả lời bằng một chênh lệch, con số đó phải bằng chênh lệch ở C5 của T2, vì cả hai đến từ `Catalog.Derive` trên cùng dữ liệu (lệch là lỗi phải điều tra); (ii) bảng `discrepancy_tickets` có đúng một dòng `OPEN` cho (EX-03, `issue_vs_measured`, ngày đó), đếm bằng truy vấn quản trị ngoài đường ứng dụng.

Giới hạn của demo: lượt 2 không phải câu chữ (câu Q-P1-8); `eais trace diff` chưa có nghĩa cho lượt lệnh (U9); chỉ chạy ở `-dev` (§1.3); kết luận chỉ là trên dữ liệu giả lập, chưa có người nghiệp vụ duyệt. Kịch bản được đóng gói thành `scripts/demo-m4.sh` (tên đề xuất) khi M4 qua cổng; script chỉ cho thấy "chạy được", không thay cho cổng X1–X9.

---

## 3. Luồng xử lý C0–C7

Khuôn lấy từ `docs/01-architecture.md` §4.1 (pipeline lệnh, ĐÍCH) và `CLAUDE.md` §14. Tất cả chặng đều tất định. Số chặng **không** trùng hết với §4.1: xem bảng đối chiếu ngay dưới bảng chặng.

| Chặng | Việc | Luật gốc (§14) | Từ chối bằng |
|---|---|---|---|
| **C0 accept** | giải mã chặt `CommandRequest` (ở lớp HTTP của `cmd/eais`, §2.6.2); kiểm kiểu, UUID, ký tự NUL; gắn `trace_id` | Schema Validation | `INVALID_REQUEST` (không audit) |
| **C1 tenant** | `Principal.TenantID` phải bằng tenant của runtime; gắn tenant vào ngữ cảnh | Tenant Check | `PERMISSION_DENIED`, `fault` = `policy` (không audit) |
| **C2 hiểu lệnh** | **không dùng ở Phase 1**: đầu vào đã có cấu trúc; chỉ tra `subject_key` trong danh mục theo khoá chính xác | — | — |
| **C3 schema tham số** | `metric` thuộc danh sách tool của pack; `day` đúng dạng và là ngày đã qua | Schema Validation | `INVALID_REQUEST` (không audit) |
| **C4 quyền** | `tickets.create`; `metrics.read` cho hai vế; phạm vi thực thể (§4); khoá không tồn tại cũng bị `PERMISSION_DENIED` | Permission | `PERMISSION_DENIED` |
| **C4b tra sổ lệnh** | tìm (tenant, người gọi, `command_id`): thấy và cùng fingerprint ⇒ phát lại (đọc lại phiếu, trả `DONE` `replayed`); thấy mà khác fingerprint ⇒ từ chối; không thấy ⇒ đi tiếp | Idempotency (nửa đọc) | `IDEMPOTENCY_KEY_REUSED` |
| **C5 chính sách, luật, tính** | loại thực thể hợp lệ cho chỉ số (kiểm **sau** quyền, để không lộ loại của máy ngoài phạm vi); tính chênh lệch bằng `Derive`; chạy R1–R6 (§5) | Policy, Business Rules | `INVALID_REQUEST` (loại thực thể), `INSUFFICIENT_EVIDENCE`, `HUMAN_REVIEW_REQUIRED`, `NO_DISCREPANCY`, `DUPLICATE_OPEN_TICKET` |
| **C6 ghi** | một giao dịch: phiếu, sổ lệnh, audit `EXECUTED`, theo thứ tự đó (§7.1); vi phạm khoá duy nhất thì phân loại lại (§6.4) | Idempotency (nửa ghi), Execute | `STORE_UNAVAILABLE`, `COMMIT_UNKNOWN` |
| **C7 đọc lại, trả** | đọc lại bằng giao dịch mới từ nơi ghi, so với bản chụp ý định (§7.2); `DONE` chỉ khi khớp. Kết cục nào từ C4 trở đi (trừ `EXECUTED`, đã nằm trong đơn vị ghi phiếu) được ghi một dòng audit riêng (§7.3) | Verify, Audit | `READBACK_UNAVAILABLE`, `VERIFY_MISMATCH` |

**Đối chiếu số chặng với `docs/01-architecture.md` §4.1** (ĐÍCH; cùng khuôn C0–C7 nhưng nghĩa của C5–C7 khác):

| Chặng ở đây | Chặng cùng số ở §4.1 | Khác |
|---|---|---|
| C0 accept | C0 accept (xác thực, idempotency key) | xác thực: lớp HTTP của `cmd/eais` làm trước C0 (token → Principal, §1.3; xác thực thật: Phase 5); idempotency tách ra C4b và C6 |
| C1 tenant, C3 schema tham số, C4 quyền | cùng tên, cùng nghĩa | không khác |
| C2 hiểu lệnh | C2 hiểu lệnh | không dùng ở Phase 1 |
| C4b tra sổ lệnh | (không có) | nửa đọc của Idempotency, thêm theo câu Q-P1-7 |
| C5 chính sách, luật, tính | C5 workflow engine (guard, chuyển trạng thái) | **khác nghĩa**: Phase 1 không có workflow (Phase 2); luật R1–R6 và phép tính nằm ở C5 |
| C6 ghi | C6 tool có tác dụng phụ (idempotent, timeout, kết quả được kiểm) | ghi là tác dụng phụ duy nhất; không có interface `Tool` (ADR-0011 mục E) |
| C7 đọc lại, trả | C7 audit, thông báo, trả trạng thái mới | audit `EXECUTED` nằm trong đơn vị ghi phiếu (C6); C7 đọc lại và ghi audit của các kết cục khác; thông báo: Phase 2 |

**Tinh chỉnh `CLAUDE.md` §14: thứ tự Idempotency.**
- §14 xếp Idempotency sau Business Rules. Hợp đồng này tách Idempotency làm hai nửa: **nửa đọc** (C4b) đặt ngay sau quyền; **nửa ghi** (khoá duy nhất) ở C6.
- Lý do: nếu luật "không trùng phiếu mở" (R5) chạy trước tra sổ lệnh, lần gửi lại hợp lệ sau một timeout sẽ bị báo `DUPLICATE_OPEN_TICKET` thay vì `DONE` phát lại. Bên gọi không phân biệt được "lần trước của tôi đã thành công" với "người khác đã tạo". Đó đúng là ca của cổng ra (A21).
- Quyền vẫn kiểm **trước** khi phát lại, nên người đã mất quyền không lấy lại được kết quả.
- [SUY] Ca A01, A21 và đột biến M5 kiểm điều này. Nếu chủ dự án muốn giữ đúng thứ tự §14, phải đổi cách phân biệt ở R5 (khi đó M5 thành thiết kế đúng, không còn là đột biến).
- Chỗ lệch này cũng lệch `docs/ROADMAP-DAY-DU.md` §5 luồng L3 (khoá chống trùng đứng sau luật nghiệp vụ) và cách đánh số của `docs/01-architecture.md` §4.1 (bảng trên). Thứ bậc ở đầu `docs/ROADMAP-DAY-DU.md` cho ADR và hợp đồng đứng trên `CLAUDE.md`, nhưng cả hai còn ĐỀ XUẤT: **chưa coi là được chấp nhận cho tới khi chủ dự án trả lời câu Q-P1-7 (§14)**.

**Hạn chót.** Cả lệnh (C0 tới hết C7) chạy dưới một hạn chót: cái đến trước giữa ngữ cảnh của bên gọi và `timeout` của tool trong pack. Giá trị `timeout`: chưa chọn, chưa đo.

---

## 4. Quyền

### 4.1 Ba lớp, đóng mặc định

1. **`tickets.create`** (capability mới). Dùng `Policy.Allows` (`intelligence/router/router.go:127`): vai không khai thì không có; capability rỗng không cấp cho ai [CODE].
2. **`metrics.read` cho cả hai vế.** Phiếu chứa giá trị của hai chỉ số và kết quả lệnh trả lại các giá trị đó, nên người tạo phải được đọc cả hai, như câu so sánh ở luồng hỏi (`router.go:203-209`) [CODE]. **Phiếu không được là đường vòng quyền đọc.** Hiện cả bốn chỉ số nhiên liệu cùng dùng capability `metrics.read`; nếu sau này mỗi chỉ số có capability riêng thì kiểm từng cái. Kiểm **hai lần**: lúc nạp pack (§4.3) và lúc chạy (phòng thủ chiều sâu).
3. **Phạm vi thực thể.** `Policy.InScope` (`router.go:143`) trên khoá máy và các tổ tiên (`entityresolution.Lineage`, `data/entityresolution/resolver.go:251`) [CODE]. Vai `scope: all` thấy mọi máy; vai `attribute` chỉ thấy máy thuộc `entity_scope` của mình.

Quyền của người **đọc** phiếu về sau (liệt kê, tìm) là việc của Phase 2 / 5. Phase 1 chỉ trả phiếu cho chính người tạo trong kết quả lệnh.

### 4.2 Ma trận vai [ĐỀ XUẤT, chưa có người nghiệp vụ duyệt]

Vai và capability hiện có ở `tenants/construction/pack/policy.yaml:5-10` [CODE]. Cột `tickets.create` là phần đề xuất thêm.

| Vai | `scope` | `metrics.read` | `tickets.create` | Dùng làm ca âm |
|---|---|---|---|---|
| `director` | all | có | **có** | — |
| `accountant` | all | có | **có** (người đối soát nhiên liệu, `docs/tenants/construction/01-business-map.md` §3.11) | — |
| `project_manager` | attribute | có | **có**, chỉ máy trong `entity_scope` | máy của dự án khác ⇒ ngoài phạm vi (A08) |
| `site_engineer` | attribute | có | **không** | đọc được số liệu mà không tạo được phiếu (A07) |
| `driver` | attribute | không | **không** | không có capability nào (A07) |

Principal loại `service` không có vai nào ở trên, nên Phase 1 không cho dịch vụ nào tạo phiếu: đóng mặc định. Khối M4 của MVP cần một vai như vậy cho luật kích hoạt (§1.4, câu Q-P1-6).

Thêm `tickets.create` vào `policy.yaml` đổi tập capability của pack, nên đổi cả phép **toàn quyền dữ liệu** của `cmd/eais`: vai phạm vi `all` nắm mọi capability của pack thì được nạp lại dữ liệu và đọc vết của người khác (`cmd/eais/auth.go:104` [CODE, làn 0]). `director` phải có `tickets.create` (đã có ở bảng trên); thiếu thì `director` mất quyền toàn quyền. Ca kiểm: bộ test của `cmd/eais` về nạp lại và đọc vết giữ nguyên (A27).

### 4.3 Luật nạp pack (cấu hình hỏng thì không chạy)

Như `intelligence/intent/intent.go:101-104` đã làm cho ý định ("quên khai capability không được biến thành ai cũng được") [CODE]. Pack nạp lỗi nếu:
1. một tool không khai capability;
2. một vai có `tickets.create` mà thiếu `metrics.read`;
3. một chỉ số trong danh sách của tool không phải chỉ số dẫn xuất đã khai ở `metrics.yaml`.

**Ai nạp.** `tools.yaml` do gói `execution/commands` nạp và kiểm chéo với `policy.yaml` và `metrics.yaml` mà `tenant.Load` đã nạp; `cmd/eais` gọi cả hai lúc khởi động, lỗi thì không chạy. `tenant.Load` không đổi (ADR-0011, "Nơi đặt mã"), nên luồng hỏi và công cụ đo (WIN, G7) giữ nguyên hành vi nạp. "Pack nạp lỗi" ở các luật trên và ở ca A12 nghĩa là bộ tệp pack lỗi qua một trong hai bước đó.

Ca âm: A12. Mẫu test sẵn có: `TestLoadChanPackHong` (`core/tenant/pack_test.go:126`), `TestDataRouteRequiresCapability` (`intelligence/intent/intent_test.go:14`).

### 4.4 Một mã cho "không có" và "ngoài phạm vi" [ĐỀ XUẤT]

- Máy không tồn tại, máy ngoài phạm vi, và khoá có ký tự lạ (ví dụ ký tự SQL, dấu nháy) cùng trả `PERMISSION_DENIED`, **cùng thông báo**. Lý do tồn tại thật chỉ nằm ở `reason_detail` của dòng audit. Ngoại lệ duy nhất: ký tự NUL bị C0 chặn bằng `INVALID_REQUEST` trước khi tra danh mục (§2.1).
- Lý do: lệnh là đường ghi; hai thông báo khác nhau cho phép người gọi dò xem máy nào tồn tại.
- Cứng hơn luồng hỏi một bậc: luồng hỏi trả `forbidden` cho thực thể đã giải mà ngoài phạm vi (`router.go:217-223`) [CODE]. Đảo được; chọn mềm hơn thì đổi ca A08.

### 4.5 Model không cấp quyền cho mình

Không có đường nào để model (hay bất kỳ bên đề xuất nào) tự cấp quyền: yêu cầu không mang vai hay phạm vi (§2.1); quyền đến từ Principal và pack. Lệnh do model đề xuất (nếu cổng mở ở Phase 6) đi qua đúng các chặng trên với Principal của người thật.

---

## 5. Luật nghiệp vụ và cách tính

| # | Luật | Chi tiết |
|---|---|---|
| R1 | Chỉ số được phép | `metric` thuộc danh sách của tool trong pack. Phase 1: chỉ `issue_vs_measured`. Hai vế đều từ nguồn có cấu trúc (phiếu kho, ERP, bảng tổng hợp; cảm biến). `reported_vs_issued` hoãn vì vế trái là tin chat và mẫu báo cáo (`CONG-QUANIT.md` câu 3). Thêm chỉ số sau này là đổi pack và thêm ca âm, không đổi mã (mục tiêu, chưa kiểm). |
| R2 | Kỳ | Đúng một ngày `day`, trước "hôm nay" theo múi giờ tenant. Ngày chưa kết thúc thì số liệu chưa đủ để gọi là chênh lệch [SUY]. Một ngày để khoá trùng so bằng nhau được (kỳ nhiều ngày cần ràng buộc loại trừ chồng lấn). |
| R3 | Đủ bằng chứng | Gọi `Catalog.Derive(issue_vs_measured, [máy], day, day)`. Cần `Total` có giá trị và `NeedsReview` = false. Thiếu một vế ⇒ `INSUFFICIENT_EVIDENCE`. Hai nguồn ngang thẩm quyền lệch quá dung sai ở một vế ⇒ `HUMAN_REVIEW_REQUIRED`, **không chọn số** (`query.go:64-197`; `combine` trả không có tổng khi `NeedsReview`) [CODE]. |
| R4 | Có chênh lệch | \|chênh lệch\| > `min_abs_diff` (tham số của tool trong pack, đơn vị lít, mặc định 0 tức mọi chênh lệch khác 0), so sánh thập phân chính xác (`core/decimal`). Ngưỡng theo phần trăm (BR-FUEL-03 ghi "±1%/ngày", `01-business-map.md:382`) chưa hỗ trợ vì chưa rõ phần trăm của vế nào (câu Q-P1-2). |
| R5 | Không trùng | Chưa có phiếu `OPEN` cùng (tenant, máy, chỉ số, ngày). Kiểm ở C5 để trả lời nhanh; **bảo đảm thật** bằng chỉ mục duy nhất ở C6. |
| R6 | Cờ ước lượng | Nếu một vế là số ước lượng thì **vẫn tạo** phiếu, phiếu và kết quả mang `approx = true`: phiếu để kiểm tra, không để kết luận. Cờ không được mất. Lưu ý: đường nạp hiện còn làm mất cờ ở vài định dạng (`docs/ROADMAP-DAY-DU.md` §5 L1) nên `approx = false` **không** chứng minh số chính xác [REPO]. |
| R7 | Nguồn của số | Giá trị chênh lệch luôn do hệ tính; bên gọi và model không đưa số (P3, P4 của `CLAUDE.md`). |

**Bản chụp bằng chứng.** Phiếu lưu danh sách bằng chứng đi kèm kết quả `Derive` (mã, loại nguồn, tham chiếu nguồn, giá trị, đơn vị, ngày, hạng thẩm quyền), không chỉ mã. Lý do: `Load` thay toàn bộ fact của tenant (`platform/pgstore/load.go:21-31`) [CODE], nên mã fact cũ có thể không còn.

---

## 6. Chống trùng

### 6.1 Hai tầng

| Tầng | Khoá | Trả lời câu | Bảo đảm bằng | Gặp thì |
|---|---|---|---|---|
| Lệnh | (tenant, người gọi, `command_id`) cộng fingerprint nội dung | "đây có phải **chính** yêu cầu đã gửi?" | khoá chính của sổ lệnh | phát lại hoặc từ chối |
| Nghiệp vụ | (tenant, máy, chỉ số, ngày), khi phiếu `OPEN` | "đã có phiếu tương đương chưa?" | chỉ mục duy nhất từng phần trên phiếu | `DUPLICATE_OPEN_TICKET` |

Hai tầng bổ sung nhau: tầng lệnh chống gửi lại; tầng nghiệp vụ chống hai khoá khác nhau cho cùng một việc (bấm đôi tạo hai UUID, hai người cùng lúc). Mỗi tầng có ca đua riêng, ép bằng F7 (§10): A02b cho tầng lệnh, A04b cho tầng nghiệp vụ.

### 6.2 Khoá lệnh: từ gì

- `command_id` là UUID do **bên gọi** sinh, mỗi ý định một lần, dùng lại nguyên khi gửi lại.
- Phạm vi khoá: (tenant, `Principal.ID`). Người khác dùng cùng UUID là lệnh mới, không nhận lại kết quả của người kia.
- **Fingerprint** = sha256 của dạng chuẩn {`subject_key`, `metric`, `day`}; không gồm `command_id`.
- Khoá chỉ bị tiêu thụ khi lệnh đã được **ghi** (commit): sổ lệnh chỉ chứa lệnh đã ghi, kể cả lệnh mà lần đọc lại chưa xác nhận được (`UNKNOWN`). Lệnh bị từ chối không để lại dấu trong sổ lệnh (chỉ để lại dòng audit, §7.3).
- Không hết hạn trong Phase 1. Chính sách hết hạn: Phase 5. Stripe và bản nháp IETF đều đặt chính sách hết hạn (`CONG-QUANIT.md` mục 4) [NGOÀI]; ở đây hoãn vì xoá khoá là thêm một đường ghi mới.

### 6.3 Gửi đôi trả gì

| Tình huống | Kết quả | Dấu vết |
|---|---|---|
| lần đầu | `DONE`, `replayed` = false | 1 phiếu, 1 dòng sổ lệnh, audit `EXECUTED` |
| gửi lại tuần tự: cùng khoá, cùng nội dung | `DONE`, `replayed` = true, **cùng mã phiếu** | không ghi phiếu; audit `REPLAYED` |
| N lần đồng thời: cùng khoá, cùng nội dung | mọi lần `DONE` cùng mã phiếu (đúng một lần `replayed` = false); lần nào hết hạn khi đang chờ khoá thì `FAILED` `STORE_UNAVAILABLE` (chưa gửi COMMIT: quy tắc §7.1), gửi lại cùng khoá | 1 phiếu (A02); không lần nào nhận lỗi vi phạm khoá |
| cùng khoá, **khác** nội dung | `FAILED` `IDEMPOTENCY_KEY_REUSED` | không ghi phiếu; audit `REFUSED`; phiếu cũ nguyên vẹn |
| N lần đồng thời: cùng khoá, **khác** nội dung | đúng một lần `DONE` (`replayed` = false); các lần còn lại `FAILED` `IDEMPOTENCY_KEY_REUSED` | 1 phiếu, 1 dòng sổ lệnh (A02b) |
| khác khoá, cùng (máy, chỉ số, ngày), phiếu còn mở | `FAILED` `DUPLICATE_OPEN_TICKET` kèm mã phiếu cũ | không ghi phiếu; audit `REFUSED` (giao dịch riêng, §7.3) |
| N lần đồng thời: khác khoá, cùng (máy, chỉ số, ngày) | đúng một lần `DONE`; các lần còn lại `FAILED` `DUPLICATE_OPEN_TICKET` kèm mã phiếu của lần thắng | 1 phiếu (A04b) |
| người gọi khác, cùng khoá | lệnh mới, thường rơi vào `DUPLICATE_OPEN_TICKET` | không lộ kết quả của người kia |
| tenant khác, cùng khoá | độc lập hoàn toàn | mỗi tenant một phiếu của mình |

### 6.4 Phân loại lại sau vi phạm khoá duy nhất

Hai lệnh đua nhau có thể làm một giao dịch gặp lỗi vi phạm khoá duy nhất (SQLSTATE 23505). Giao dịch đó đã huỷ; pipeline **không** trả lỗi thô mà đọc lại bằng giao dịch mới:
1. có dòng sổ lệnh cùng (tenant, người gọi, `command_id`): fingerprint bằng ⇒ phát lại; khác ⇒ `IDEMPOTENCY_KEY_REUSED`;
2. không có, nhưng có phiếu `OPEN` cùng khoá nghiệp vụ ⇒ `DUPLICATE_OPEN_TICKET`;
3. không thấy gì ⇒ `INTERNAL_ERROR` (không lường trước).

Bên thua cuộc đua vì thế nhận kết quả đúng nghĩa, không phải lỗi 500. Ca A02, A02b, A04b (cả ba ép đua bằng F7, §10); đột biến M20.

---

## 7. Ghi, đọc lại, audit

### 7.1 Đơn vị ghi

- **Một giao dịch PostgreSQL** đặt GUC tenant từ Principal (như `platform/pgstore/pgstore.go:59-70`), ba INSERT theo thứ tự cố định: (1) phiếu, (2) sổ lệnh, (3) audit `EXECUTED`. Cả ba cùng có hoặc cùng không (I4). Giao dịch này, từ BEGIN tới COMMIT, gọi là **đơn vị ghi phiếu** ở các mục sau. Các dòng audit khác (`REFUSED`, `REPLAYED`, `VERIFY_FAILED`) **không** thuộc nó: chúng ghi bằng giao dịch riêng (§7.3), và harness đếm hai thứ tách nhau (§10).
- **Vì sao thứ tự này.** Sổ lệnh có khoá ngoại **thường** (không `DEFERRABLE`) tới phiếu (§7.4), nên phiếu phải có trước; đảo thì INSERT sổ lệnh vi phạm khoá ngoại ở mọi lần tạo (đột biến M23). Audit đứng cuối để F2 (A20) thử đúng điều khó: hai dòng đã INSERT phải bị huỷ khi dòng thứ ba hỏng. Phiếu giữ `command_id` dạng cột thường, không có khoá ngoại ngược tới sổ lệnh (sẽ thành vòng tròn và buộc phải hoãn kiểm tra khoá ngoại).
- Mức cô lập mặc định (READ COMMITTED) đủ vì mọi bảo đảm duy nhất đến từ chỉ mục, không từ đọc rồi ghi. Chưa đo tranh chấp.
- **Không dùng lại `inTenant` nguyên trạng.** Nó gom mọi lỗi, kể cả lỗi COMMIT, thành `infra:postgres` (`pgstore.go:59-70`) [CODE], nên không phân biệt "chắc chắn chưa ghi" với "không biết". Đơn vị ghi phiếu của Phase 1 phải tách được lỗi COMMIT. `Store.pool` và `inTenant` không xuất ra khỏi `platform/pgstore` [CODE `pgstore.go:21`, `:59` tại `5d69d24`], nên mã của `execution/commands` mở giao dịch qua một hàm mới, nhỏ, của `platform/pgstore` (mở giao dịch đã đặt GUC tenant, trả giao dịch để bên gọi tự `Commit` / `Rollback`): ADR-0011, "Nơi đặt mã".
- **Phân loại lỗi trong giao dịch:**
  - PG báo lỗi câu lệnh và giao dịch đã huỷ ⇒ chắc chắn không ghi ⇒ `FAILED` `STORE_UNAVAILABLE` (hoặc §6.4 nếu là vi phạm khoá duy nhất);
  - lỗi kết nối hay hết hạn **trước** khi gửi COMMIT (kể cả hết hạn khi đang chờ khoá của giao dịch khác, §6.3) ⇒ giao dịch chưa commit không tồn tại sau khi kết nối đứt ⇒ `FAILED` `STORE_UNAVAILABLE`;
  - lỗi hay hết hạn **từ lúc gửi COMMIT** ⇒ `UNKNOWN` `COMMIT_UNKNOWN`.
  - Cơ chế chính xác của trình điều khiển ở ranh giới COMMIT: chưa kiểm (mục 15, U1).

### 7.2 Đọc lại sau ghi

- Sau commit, một giao dịch **mới** (kết nối khác trong pool, cùng GUC tenant) đọc **từ nơi ghi** (nút ghi, không replica) ba thứ: phiếu theo mã, dòng sổ lệnh theo khoá, dòng audit `EXECUTED` theo `command_id`.
- So từng trường với **bản chụp ý định** dựng trong bộ nhớ **trước** khi ghi. **Không** so với kết quả `RETURNING` của chính lệnh INSERT: làm vậy thì đọc lại không kiểm gì.
- `DONE` chỉ khi: đúng 1 phiếu, đúng 1 dòng sổ lệnh, ít nhất 1 dòng `EXECUTED`, mọi trường khớp (trừ thời gian do DB gán), gồm `command_id` ghi trên phiếu (cột thường, không có khoá ngoại tới sổ lệnh; §7.4).
- Phát lại cũng đọc lại (I3): trả `DONE` `replayed` chỉ sau khi thấy lại phiếu.
- Đọc lại không thấy hoặc lệch: **không** `DONE`; trả `UNKNOWN` `VERIFY_MISMATCH` và ghi audit `VERIFY_FAILED`. Không tự sửa (không có đường UPDATE, DELETE).

### 7.3 Audit

- **Ghi gì.** Mọi lệnh đi qua C4 trở đi để lại **đúng một dòng** cho kết cục:
  - `EXECUTED` (trong đơn vị ghi phiếu);
  - `REPLAYED`;
  - `REFUSED` kèm `reason_code` (gồm cả `PERMISSION_DENIED`, `DUPLICATE_OPEN_TICKET`, các từ chối do luật);
  - `VERIFY_FAILED` (bất thường, có thể đi kèm `EXECUTED` trước đó).
- **Cột:** thời gian (đồng hồ DB), tenant, người gọi (`Principal.ID`, vai, `entity_scope`), `command_id`, hành động (`open_discrepancy_ticket`), kết cục, `reason_code`, `reason_detail` (lý do thật, ví dụ phân biệt "không tồn tại" với "ngoài phạm vi"), `subject_key`, `metric`, `day`, mã phiếu (nếu có), fingerprint, phiên bản pack, phiên bản mã, `trace_id`.
- **Không ghi:** bí mật, token, bất cứ thứ gì ngoài danh sách cột trên.
- **Ở đâu.** Bảng `audit_log` cùng PostgreSQL, cùng tenant, RLS FORCE. `EXECUTED` nằm **trong** đơn vị ghi phiếu (§7.1). Các dòng khác ghi bằng giao dịch riêng, cố gắng tối đa: ghi audit của một lần từ chối hỏng thì kết quả từ chối không đổi, nhưng để lại vết trong trace với nhãn `infra:postgres`.
- **Không có dòng audit** cho lệnh bị chặn ở C0, C1, C3 (cấu trúc sai, sai tenant, tham số sai): chưa có yêu cầu hợp lệ hoặc chưa có tenant tin cậy để gắn; lúc đó `tenantOf` từ chối mọi truy vấn không mang tenant (`pgstore.go:50`) [CODE]. Vết nằm ở trace.
- **Chưa làm:** chuỗi băm của audit (đích của `docs/01-architecture.md` §7, "audit chỉ ghi thêm, chuỗi băm"). Phase 1 làm phần "chỉ ghi thêm" bằng quyền; chuỗi băm cần tuần tự hoá ghi theo tenant và chưa có lỗi nào đòi (P7). Hoãn Phase 5.

### 7.4 Lưu trữ và quyền

- Ba bảng mới (tên đề xuất): `commands`, `discrepancy_tickets`, `audit_log`. `commands` và `audit_log` đã có tên trong `docs/01-architecture.md` §7 (nhóm Quy trình, nhóm Quản trị). `discrepancy_tickets` là tên **mới** của Phase 1: §7 chưa có bảng phiếu (bảng đích của phiếu là `workflow_instances` và `human_tasks`; ADR-0011 mục B nêu lý do không dùng ở Phase 1). Migration: `db/migrations/<NNNN>_phieu_chenh_lech.sql`, với `<NNNN>` là **số kế tiếp sau migration cuối của `main` lúc mở mã** (chọn lúc viết mã, không cố định ở giấy; chỉ tạo khi có mã). Làn 0 đã dùng `0003_kho_vet_ben.sql` (ba bảng `decision_records`, `quarantine_records`, `usage_records`) [CODE, làn 0], và Phase 0a (ADR-0012) cũng cần một số: không bên nào được trùng số. Tên ba bảng của Phase 1 không trùng bảng nào của làn 0. `audit_log` (audit nghiệp vụ, Phase 1) khác `decision_records` (vết gỡ lỗi từng lượt, làn 0): xem §7.5.
- Mỗi bảng: `tenant_id`; RLS ENABLE và FORCE; chính sách đọc GUC `eais.tenant_id` như `db/migrations/0002_dinh_danh_eais.sql:20-23` [CODE].
- Vai `eais_app` chỉ **SELECT, INSERT** trên ba bảng. **Không UPDATE, không DELETE**: tính chỉ-thêm do quyền bảo đảm, không do thói quen (I8). Hiện vai này có DELETE trên bốn bảng cũ (`0002:23`) nhưng đó là vì `Load`.
- **Không khoá ngoại** từ ba bảng mới tới `entities`, `entity_aliases`, `facts`, `documents`, vì `Load` xoá và nạp lại bốn bảng đó trong một giao dịch (`load.go:21-31`) [CODE]: khoá ngoại sẽ làm `Load` hỏng hoặc phiếu mồ côi. Phiếu giữ khoá và mã thực thể dạng chữ cùng bản chụp bằng chứng. Sổ lệnh có khoá ngoại **thường** (không `DEFERRABLE`) tới phiếu cùng tenant (cả hai bảng mới); phiếu không có khoá ngoại ngược tới sổ lệnh (§7.1).
- `status` của phiếu có ràng buộc CHECK = `OPEN` ở Phase 1; chỉ mục duy nhất từng phần (tenant, máy, chỉ số, ngày) khi `OPEN`.

**Trường của phiếu** (đề xuất; chốt khi viết migration):

| Trường | Nguồn |
|---|---|
| `id` | hệ sinh |
| `status` | `OPEN` |
| thực thể: mã, khoá | danh mục |
| `metric`, `day` | yêu cầu |
| vế trái: chỉ số, giá trị, đơn vị | `Derive` |
| vế phải: chỉ số, giá trị, đơn vị | `Derive` |
| `diff`, `approx`, công thức | `Derive` |
| bản chụp bằng chứng | `Derive` |
| `created_by`, `created_at` | Principal; DB |
| phiên bản pack, phiên bản mã, `command_id` | hệ |

### 7.5 Vết từng bước của lượt lệnh, cho `eais trace` [ĐỀ XUẤT]

**Vết khác audit.** Audit (§7.3) là bản ghi nghiệp vụ chỉ-thêm của quyết định; `EXECUTED` nằm trong đơn vị ghi phiếu. Vết là bản ghi **gỡ lỗi từng bước** của một lượt, kể cả lượt bị từ chối ở C0 đến C3 (không có audit), để chủ dự án và người soát đọc đúng chặng gây ra kết cục. Hai thứ nối nhau bằng `trace_id` (cột của audit, §7.3). Nguồn sự thật về phiếu là bảng phiếu cộng audit, không phải vết.

**Lưu ở đâu.** Cùng kho vết của làn 0: `traces.Record` vào bảng `decision_records` bằng `SaveDecision`, đọc bằng `eais trace` và `GET /v1/trace/{id}`, với quyền đọc của làn 0 (người đã gọi lượt đó, hoặc principal toàn quyền dữ liệu; còn lại 404) [CODE, làn 0: `cmd/eais/server.go:415`, `platform/pgstore/vet.go`]. **Không thêm bảng vết.** `trace_id` do hệ tự cấp (UUID) ở C0. Bản ghi được dựng một lần cho mỗi lượt, thử lại bằng chính bản đó (hợp đồng của `vet.go`), và qua `traces.Sanitize` của làn 0 trước khi ghi.

**Chín bước, luôn đủ.** Mỗi lượt đã xác thực ghi đủ chín bước theo thứ tự dưới đây (C4b nằm giữa C4 và C5). Mã bước là tên chặng ở §3; tên là tên ngắn hiện trong bảng vết.

| Mã bước | Tên | Ghi vào bước (vào → ra) | Lý do và bằng chứng | Dừng ở đây thì |
|---|---|---|---|---|
| `C0` | `accept` | yêu cầu đã làm sạch (`command_id`, `subject_key`, `metric`, `day`) → `accepted` hoặc trường sai đầu tiên | luật giải mã chặt: trường lạ, UUID, ký tự NUL | `deviant`, `INVALID_REQUEST` |
| `C1` | `tenant` | Principal (mã, loại, vai, `entity_scope`) → tenant | "tenant lấy từ token, khớp tenant của runtime, không lấy từ thân" | `deviant`, `PERMISSION_DENIED` (`fault` = `policy`) |
| `C2` | `understand` | không có | "đầu vào đã có cấu trúc, Phase 1 không dùng C2" | không bao giờ dừng: luôn `skipped` |
| `C3` | `params` | `metric`, `day`, "hôm nay" của tenant → hợp lệ | `metric` thuộc danh sách tool; `day` là ngày đã qua | `deviant`, `INVALID_REQUEST` |
| `C4` | `permission` | capability cần (`tickets.create`, `metrics.read` hai vế), phạm vi → cho phép hoặc từ chối | ba lớp (§4.1). Từ chối **không** tách "không tồn tại" với "ngoài phạm vi" (§4.4) | `deviant`, `PERMISSION_DENIED` |
| `C4b` | `ledger` | (tenant, người gọi, `command_id`) và fingerprint → mới, phát lại, hoặc dùng lại khoá | mã dòng sổ lệnh khi phát lại | `deviant`, `IDEMPOTENCY_KEY_REUSED`; phát lại là `ok`, ghi "phát lại" |
| `C5` | `rules` | máy, chỉ số, ngày → hai vế, chênh lệch, `approx`, mã bằng chứng; luật dừng (R1–R6) | bằng chứng của `Derive`; hai nguồn ngang thẩm quyền lệch thì nguồn và số bị loại ghi ở "loại", không chọn số | `deviant`: `INVALID_REQUEST` (loại thực thể), `INSUFFICIENT_EVIDENCE`, `HUMAN_REVIEW_REQUIRED`, `NO_DISCREPANCY`, `DUPLICATE_OPEN_TICKET` |
| `C6` | `write` | bản chụp ý định → mã phiếu; `commit` hoặc `rollback` | thứ tự ba INSERT; phân loại lỗi COMMIT (§7.1) | `error`: `STORE_UNAVAILABLE`, `COMMIT_UNKNOWN`, `INTERNAL_ERROR`; `deviant`: kết cục phân loại lại sau vi phạm khoá duy nhất (§6.4) |
| `C7` | `verify` | mã phiếu → đọc lại khớp; dòng audit đã ghi hoặc không ghi được | so với bản chụp ý định (§7.2) | `error`: `READBACK_UNAVAILABLE`, `VERIFY_MISMATCH` |

- **Trạng thái.** Bước đi qua là `ok`. Bước dừng là `deviant` (từ chối theo luật) hoặc `error` (hạ tầng, mã) kèm nhãn `fault`. Bước nằm giữa bước dừng và C7 là `skipped` kèm lý do "lượt dừng ở C<k>". C7 luôn chạy: trả kết quả và ghi audit của lượt, nếu lượt có audit (§7.3); phần đọc lại chỉ chạy khi C6 đã ghi hoặc khi phát lại (§7.2). `INTERNAL_ERROR` ở bước nào thì bước đó là `error`. Vì vậy dòng "BƯỚC LỆCH ĐẦU TIÊN" của `eais trace` trỏ đúng chặng ở cột "Dừng ở đây thì" của bảng trên (ca A29); cột đó là cột "Từ chối bằng" của §3, thêm các kết cục hạ tầng và kết cục phân loại lại ở C6.
- **Trường quyết định.** Dùng đúng kiểu `traces.Decision` của làn 0; trường nào luồng lệnh không có nghĩa thì để rỗng, không bịa. `Question` = dạng chuẩn của yêu cầu đã làm sạch (`open_discrepancy_ticket subject_key=… metric=… day=… command_id=…`; rỗng khi thân không đọc được); `Intent` = tên lệnh; `Slots` = `metric=…`, `day=…`; `Route` = `TOOL` (một trong 12 route, `docs/CONVENTIONS.md` §4.1); `Entities` = khoá máy, chỉ khi đã qua C4; `Evidence` = mã bằng chứng của bản chụp (từ C5); `Outcome` = `DONE` hoặc `reason_code`; `Answer` = `status=… reason_code=… replayed=… ticket=…`; `Normalized` và `Verdict` rỗng.
- **Không lộ.** Vết của lượt `PERMISSION_DENIED` không phân biệt "máy không tồn tại" với "máy ngoài phạm vi" (§4.4): cùng một quyết định, cùng một lý do chung; lý do thật chỉ ở `reason_detail` của audit (đọc bằng vai quản trị). Phép tra khoá máy trong danh mục (§3 nhắc ở C2) không ghi kết quả "có" hay "không" vào C2: C2 luôn `skipped`, kết quả chỉ vào quyết định của C4. Vết không chứa giá trị ngoài quyền đọc của người gọi (C5 chỉ chạy sau C4) và không chứa bí mật (làm sạch của làn 0).
- **Mất vết.** Lưu vết xảy ra sau C7. Không lưu được vết thì kết quả lệnh **không đổi**: phiếu đã commit và đã đọc lại là sự thật không rút lại được, và đổi `DONE` thành lỗi chỉ khiến bên gọi thử lại mù. `trace_id` để rỗng, bộ đếm lỗi lưu vết của làn 0 tăng, ghi log; gửi lại cùng khoá phát lại kết quả và ghi vết mới. Khác luồng hỏi: ở đó mất vết thì không giao câu trả lời (`cmd/eais/server.go:205`, hàm `ask` [CODE, làn 0]), vì câu trả lời không có tác dụng phụ.
- **Giới hạn đã biết.** Tiến trình chết giữa C6 và lúc lưu vết: phiếu, sổ lệnh, audit `EXECUTED` (kèm `trace_id`) có, vết không có; gửi lại cùng khoá phát lại và ghi vết mới (T4). `ValidStage` hiện chỉ nhận `S0`…`S10`, `S?`, `L1.<tên>` (`observability/traces/step.go:60`) và `Record.Validate` từ chối mã khác (`record.go:59`) [CODE, làn 0]: phải nhận thêm `C0`…`C7` và `C4b` (ADR-0011, "Nơi đặt mã"). Bảng vết in nhãn "câu hỏi:" cho `Decision.Question` (`render.go`): ở lượt lệnh đó là dạng chuẩn của yêu cầu. `eais trace diff` chưa có nghĩa cho lượt lệnh (U9).

---

## 8. Trạng thái phiếu và ai sở hữu

Trả lời 10 mặt của cổng "ai sở hữu?" (`docs/05-master-roadmap.md:30`; `docs/ROADMAP-DAY-DU.md` §4.2 mục 3) cho phiếu ở Phase 1.

| Mặt | Trả lời |
|---|---|
| 1. Ai sở hữu nó | **EAIS** (chế độ B). Trong EAIS: pipeline lệnh sở hữu *quyết định tạo*; lớp lưu trữ sở hữu *bản ghi*; pack sở hữu *chính sách* (chỉ số được phép, ngưỡng, vai → capability). Model và luồng hỏi không sở hữu gì. |
| 2. Trạng thái gì | Phiếu: đúng một trạng thái `OPEN` (CHECK). Sổ lệnh chỉ chứa lệnh đã được ghi. Mọi chuyển trạng thái sau `OPEN` thuộc engine của Phase 2. |
| 3. Nguồn sự thật nào | Bảng `discrepancy_tickets` là nguồn sự thật duy nhất cho câu "phiếu có tồn tại không". Giá trị chênh lệch trong phiếu là **bản chụp** tại lúc tính; nguồn sự thật của số liệu vẫn là fact có thẩm quyền (L1). Câu chữ của luồng hỏi và kết quả trả về không phải nguồn sự thật. |
| 4. Ai ghi | Chỉ đường lệnh (pipeline lệnh gọi đơn vị ghi phiếu của lớp lưu trữ) bằng vai `eais_app`, trong một giao dịch. Không có đường UPDATE, DELETE. Luồng hỏi và model không ghi vào ba bảng. |
| 5. Ai đọc | Người tạo, qua kết quả lệnh, trong giới hạn `metrics.read` và phạm vi. Pipeline đọc lại nội bộ. Liệt kê, tìm phiếu cho người dùng: ngoài Phase 1. Điều tra vận hành: truy vấn trực tiếp bằng vai quản trị ngoài đường ứng dụng. Vết của chính lượt lệnh (§7.5): người đã gọi, hoặc principal toàn quyền dữ liệu của `cmd/eais`. |
| 6. Vòng đời | tạo (`OPEN`) → [Phase 2: điều tra → duyệt → xác minh → đối soát → đóng]. Phase 1 không xoá, không sửa, không hết hạn. Sổ lệnh và audit giữ vô hạn; chính sách giữ: Phase 5. |
| 7. Hỏng thế nào | §9 (T1–T6) và các từ chối ở §2.4. |
| 8. Phục hồi thế nào | Gửi lại cùng `command_id` luôn an toàn. Không có job nền. Khởi động lại tiến trình không mất gì vì trạng thái ở PostgreSQL. `VERIFY_MISMATCH` cần người xem audit. Sao lưu và khôi phục cơ sở dữ liệu: Phase 5. |
| 9. Ranh giới bảo mật | tenant (RLS FORCE, GUC từ Principal); vai ứng dụng không vượt được RLS (`pgstore.Open` từ chối vai superuser hoặc BYPASSRLS, `pgstore.go:25`); quyền ba lớp đóng mặc định; không chữ tự do; audit chỉ-thêm. **Chưa có xác thực thật**: token dev công khai, chỉ loopback (§1.3). |
| 10. Kiểm chứng ra sao | Ca âm §10, cổng ra và đột biến §11, bộ giữ lại độc lập §12, hồi quy đường hỏi (A27). |

---

## 9. Hỏng thế nào, phục hồi thế nào

### 9.1 Các điểm hỏng

| Mã | Điểm hỏng | Dữ liệu thật sau đó | Kết quả báo | Phục hồi |
|---|---|---|---|---|
| T1 | trước khi mở giao dịch: cơ sở dữ liệu không tới được, hoặc hết hạn | 0 dòng | `FAILED` `STORE_UNAVAILABLE` | gửi lại cùng khoá |
| T2 | giữa giao dịch: PG báo lỗi câu lệnh (ví dụ INSERT audit bị từ chối) | rollback: 0 dòng | `FAILED` `STORE_UNAVAILABLE` (không có phiếu) | gửi lại cùng khoá |
| T3 | từ lúc gửi COMMIT: mất kết nối hoặc hết hạn trước khi biết kết quả | 0 hoặc 3 dòng, **không biết** | `UNKNOWN` `COMMIT_UNKNOWN` | gửi lại cùng khoá: phát lại nếu đã commit; thực thi mới nếu chưa; **không bao giờ hai phiếu** |
| T4 | commit xong, phản hồi không tới bên gọi (bên gọi hết hạn, hay tiến trình chết rồi khởi động lại) — **"timeout sau khi ghi"** của cổng ra | 3 dòng | phía bên gọi: không có kết quả; nếu hệ còn sống: `UNKNOWN` | gửi lại cùng khoá ⇒ `DONE` `replayed`; không cần bộ nhớ nào của tiến trình |
| T5 | commit xong, đọc lại lỗi hoặc hết hạn | 3 dòng | `UNKNOWN` `READBACK_UNAVAILABLE` | gửi lại ⇒ tra sổ lệnh và đọc lại |
| T6 | commit xong, đọc lại không thấy hoặc lệch trường | tuỳ lỗi | `UNKNOWN` `VERIFY_MISMATCH`, audit `VERIFY_FAILED` | người xem audit; không tự sửa |

### 9.2 Nguyên tắc

1. **Không biết thì nói `UNKNOWN`.** Không làm tròn thành `FAILED` (sẽ khiến bên gọi tạo phiếu thứ hai bằng khoá mới) và không làm tròn thành `DONE`.
2. **Gửi lại cùng khoá luôn an toàn**, nhờ khoá duy nhất và so fingerprint, kể cả khi lần trước commit muộn (A22).
3. **Phase 1 không có việc nền:** không job đối soát, không xoá, không thử lại tự động. Bên gọi chủ động gửi lại.
4. **`FAILED` chỉ khi chắc chắn không ghi.**
5. Không HA, không sao lưu ở Phase 1 (Phase 5, 9).

---

## 10. Ca âm bắt buộc

**Cách đếm.** Mọi khẳng định "phiếu = 1", "0 ghi" được đếm bằng **kết nối riêng của harness** (vai ngoài đường ứng dụng), không dựa vào điều pipeline tự báo. **"0 ghi"** nghĩa là 0 dòng mới ở `discrepancy_tickets` và `commands` và 0 dòng audit `EXECUTED`. Các dòng audit `REFUSED`, `REPLAYED`, `VERIFY_FAILED` **không** tính là "ghi": chúng ghi bằng giao dịch riêng (§7.3) và được đếm riêng ở ca nào nói tới.

**Điểm chèn lỗi** (test-only, qua một lớp bọc quanh lớp lưu trữ; không phải cờ chạy thật):

| Điểm | Ý nghĩa |
|---|---|
| F1 `before_begin` | lớp lưu trữ báo lỗi trước khi mở giao dịch |
| F2 `in_tx_audit_insert` | lỗi ở lần INSERT audit trong giao dịch |
| F3 `commit_ack_lost` | giao dịch **được commit** nhưng lớp lưu trữ báo lỗi hoặc hết hạn cho pipeline |
| F4 `reply_lost` | pipeline hoàn tất, bên gọi không nhận được kết quả; harness dựng pipeline mới (giả lập khởi động lại) |
| F5 `commit_delayed` | commit đến sau khi pipeline đã báo `UNKNOWN` |
| F6 `readback_fail` / `readback_stale` | đọc lại trả lỗi / trả thiếu hoặc lệch dòng |
| F7 `race_barrier` | rào đồng bộ ngay trước đơn vị ghi (đầu C6, trước khi mở giao dịch): giữ N lần chạy song song cho tới khi cả N đã qua C4b và C5, rồi thả cùng lúc. Không đủ N lần tới rào trong hạn của harness thì ca ĐỎ, không được bỏ qua |
| F8 `trace_save_fail` | kho vết trả lỗi khi lưu vết của lượt (sau C7); lớp bọc nằm quanh kho vết, không quanh lớp lưu trữ của phiếu (§7.5) |

**Đơn vị ghi phiếu và bộ đếm của lớp bọc.** *Đơn vị ghi phiếu* là lời gọi **duy nhất** của lớp lưu trữ ở C6: mở một giao dịch, ba INSERT (phiếu, sổ lệnh, audit `EXECUTED`), COMMIT (§7.1). Lớp bọc quanh lớp lưu trữ giữ ba bộ đếm lời gọi **tách nhau**: (1) đơn vị ghi phiếu; (2) ghi audit riêng, tức mỗi dòng `REFUSED`, `REPLAYED`, `VERIFY_FAILED` là một giao dịch riêng (§7.3); (3) đọc (tra sổ lệnh, tìm phiếu mở, đọc lại). Ca nào nói "đơn vị ghi phiếu không được gọi" (A04) chỉ đọc bộ đếm (1); một dòng audit không bao giờ làm bộ đếm (1) tăng. Bộ đếm là của lớp bọc (phía mã); mọi khẳng định về số dòng vẫn được kiểm chéo bằng đếm ở kết nối riêng của harness.

**Vì sao F7** [SUY]: không có rào, lần đến sau thường bị C4b hoặc R5 chặn trước C6, nên đường vi phạm khoá duy nhất (§6.4) không chạy và ca đồng thời xanh nhờ lịch chạy, kể cả khi M2, M4 hay M20 đang sống. F7 buộc cả N lần cùng qua C4b và C5 khi chưa lần nào ghi, nên chỉ còn khoá của cơ sở dữ liệu phân xử. Ai thắng thì ca không đoán trước, nên ca chỉ khẳng định số lượng và loại kết quả, không khẳng định lần nào thắng.

| Mã | Tình huống | Phải thấy |
|---|---|---|
| **A01** | gửi đôi tuần tự, cùng khoá, cùng nội dung | lần 1 `DONE`; lần 2 `DONE` `replayed` = true, cùng mã phiếu; phiếu = 1, sổ lệnh = 1; audit `EXECUTED` = 1, `REPLAYED` = 1 |
| **A02** | gửi đôi đồng thời, ép đua bằng F7: N ≥ 2 lần song song, cùng khoá, cùng nội dung | phiếu = 1; mọi lần `DONE` cùng mã phiếu, đúng một lần `replayed` = false (hạn chót đủ dài: không lần nào hết hạn chờ khoá); không lần nào nhận lỗi vi phạm khoá hay `INTERNAL_ERROR` |
| **A02b** | đua đồng thời, ép bằng F7: N ≥ 2 lần song song, cùng khoá, cùng người gọi, **khác** nội dung (máy khác nhau, đều trong phạm vi và có số liệu) | đúng một lần `DONE` (`replayed` = false); các lần còn lại `IDEMPOTENCY_KEY_REUSED`; phiếu = 1, sổ lệnh = 1; không `INTERNAL_ERROR` |
| **A03** | cùng khoá, khác nội dung (khác máy, khác ngày) | `IDEMPOTENCY_KEY_REUSED`; không ghi mới; phiếu cũ nguyên vẹn |
| **A04** | khác khoá, cùng (máy, chỉ số, ngày), phiếu còn mở; **tuần tự** | `DUPLICATE_OPEN_TICKET` kèm mã phiếu cũ; phiếu = 1, sổ lệnh = 1; **đơn vị ghi phiếu** không được gọi cho lần thứ hai: bộ đếm (1) của lớp bọc = 1 sau cả hai lần (từ chối đến từ R5 ở C5, không phải từ chỉ mục duy nhất). Dòng audit `REFUSED` (`reason_code` = `DUPLICATE_OPEN_TICKET`) **vẫn** được ghi bằng giao dịch riêng (§7.3): audit `REFUSED` = 1, bộ đếm (2) tăng đúng 1 ở lần thứ hai. Ca **không** đòi "không có giao dịch ghi nào" |
| **A04b** | đua đồng thời, ép bằng F7: N ≥ 2 lần song song, **khác** khoá, cùng (máy, chỉ số, ngày) | đúng một lần `DONE`; các lần còn lại `DUPLICATE_OPEN_TICKET` kèm mã phiếu của lần thắng; phiếu = 1; không `INTERNAL_ERROR` |
| **A05** | người gọi khác (cùng tenant), cùng khoá, cùng nội dung | **không** phát lại kết quả của người kia; xử lý như lệnh mới ⇒ `DUPLICATE_OPEN_TICKET`; audit ghi người gọi thứ hai |
| **A06** | hai tenant dùng cùng khoá | mỗi tenant đúng 1 phiếu của mình; đọc chéo bằng vai ứng dụng = 0 dòng |
| **A07** | thiếu `tickets.create`: `site_engineer` (có `metrics.read`) và `driver` | `PERMISSION_DENIED`; 0 ghi; audit `REFUSED` |
| **A08** | có `tickets.create` nhưng máy ngoài phạm vi; máy không tồn tại; khoá chứa ký tự SQL | cùng `PERMISSION_DENIED`, cùng thông báo; 0 ghi; không lỗi SQL, không `INTERNAL_ERROR` |
| **A09** | có `tickets.create` nhưng thiếu `metrics.read`: **chỉ ca phát triển** (dựng Policy bằng tay, vì pack hợp lệ không cho cấu hình này, §4.3) | `PERMISSION_DENIED`; 0 ghi |
| **A10** | Principal thuộc tenant khác với runtime | `FAILED` `fault` = `policy`; 0 ghi; không có dòng audit (§7.3) |
| **A11** | yêu cầu mang trường tenant, vai, Principal hoặc trường lạ | `INVALID_REQUEST`; 0 ghi; cổng kiến trúc không thấy kiểu yêu cầu nào mang trường tenant (ở lớp HTTP: A28) |
| **A12** | pack hỏng: tool không capability; vai có `tickets.create` thiếu `metrics.read`; chỉ số không phải dẫn xuất | pack không nạp được, lỗi gom đủ thông tin (§4.3) |
| **A13** | thiếu trường; `command_id` không phải UUID; ký tự NUL trong một trường (kể cả khoá máy); ngày sai dạng, hôm nay, tương lai; chỉ số ngoài danh sách (kể cả chỉ số gốc `issued_fuel`); khoá thực thể là dự án, không phải máy, với người có quyền mọi máy | `INVALID_REQUEST` kèm trường sai; 0 ghi; không có giá trị mặc định ngầm; riêng biến thể loại thực thể (kiểm sau quyền) có audit `REFUSED` |
| **A14** | hai vế bằng nhau (hoặc không vượt `min_abs_diff`) | `NO_DISCREPANCY`; 0 ghi |
| **A15** | thiếu một vế (không có fact trong ngày) | `INSUFFICIENT_EVIDENCE`; 0 ghi |
| **A16** | một vế có hai nguồn ngang thẩm quyền lệch quá dung sai | `HUMAN_REVIEW_REQUIRED`; 0 ghi; không có số nào được chọn |
| **A17** | một vế là số ước lượng | phiếu được tạo; phiếu và kết quả cùng mang `approx` = true |
| **A18** | `Load` thay toàn bộ dữ liệu sau khi đã tạo phiếu | `Load` vẫn chạy; phiếu vẫn đọc được cùng bản chụp bằng chứng; gửi lại cùng khoá vẫn phát lại |
| **A19** | F1 trước khi mở giao dịch | `FAILED` `STORE_UNAVAILABLE`; 0 ghi; gửi lại thì thực thi bình thường |
| **A20** | F2 lỗi INSERT audit giữa giao dịch | rollback đủ ba dòng: 0 phiếu, 0 sổ lệnh, 0 `EXECUTED`; không `DONE` |
| **A21** | **timeout sau khi ghi**, hai biến thể: (a) F3; (b) F4 | (a) bên gọi nhận `UNKNOWN` `COMMIT_UNKNOWN`; (b) bên gọi không nhận gì; cả hai: phiếu = 1; gửi lại cùng khoá ⇒ `DONE` `replayed` = true, cùng mã; vẫn đúng sau khi dựng pipeline mới |
| **A22** | F5 commit đến muộn | gửi lại ⇒ tổng cộng đúng 1 phiếu, không hai |
| **A23** | F6 đọc lại hỏng | không bao giờ `DONE`; `UNKNOWN`; lệch trường thì có audit `VERIFY_FAILED` |
| **A24** | đếm lời gọi model trên đường lệnh | = 0, kể cả khi gateway model được gắn |
| **A25** | câu lệnh tự do vào luồng hỏi, với các vai có `metrics.read`: "Tạo phiếu kiểm tra chênh lệch máy EX-03 hôm qua" và biến thể (thêm chữ "dầu"; "hôm nay"; vai khác) | (1) câu trả lời không chứa lời khẳng định một phiếu đã được tạo hay mở (ví dụ "đã tạo", "đã mở phiếu", mã phiếu); (2) số phiếu = 0, đếm bằng kết nối riêng của harness; (3) đường hỏi **không đổi** so với bản chụp lấy ngay trước dòng mã Phase 1 đầu tiên (route, ý định, kết cục, câu trả lời; sau khi vòng S4 đóng băng). **Không** đòi từ chối hay hỏi lại: hiện hành câu này được trả lời như câu hỏi tính toán (§1.5, [ĐO] ở `c086fe7`, `1e5457b` và `7d72e33`); chặn động từ lệnh là nợ của S4 hoặc Phase 6 |
| **A26** | vai ứng dụng thử UPDATE, DELETE trên ba bảng; chèn trực tiếp một dòng sổ lệnh trỏ tới phiếu không tồn tại; vai vượt RLS | UPDATE, DELETE bị từ chối quyền; dòng sổ lệnh mồ côi bị khoá ngoại từ chối; `pgstore.Open` từ chối vai vượt RLS (test có sẵn) |
| **A27** | hồi quy đường hỏi | bộ 117 / H1 / H3 / H4, đường bộ nhớ và PostgreSQL: 0 câu trả lời đổi chữ; test so **theo tên**: 0 mất, 0 đổi; cùng phép so cho bộ test của `cmd/eais` (làn 0): `/v1/ask`, `/v1/ingest`, `/v1/trace/{id}`, `/healthz` giữ nguyên hành vi |
| **A28** | cửa HTTP của lệnh (§2.6.2) bằng token dev thật: (a) không token, token lạ; (b) thân mang trường `tenant`, `principal` hay `role`, hoặc trường lạ; thân rỗng; JSON hỏng; nội dung thừa; sai `Content-Type`; (c) tên lệnh lạ ở đường dẫn; (d) một lượt cho mỗi nhóm kết cục: `DONE`, từ chối do luật, `UNKNOWN` (F3) | (a) 401, cùng một phản hồi, 0 ghi, không có vết; (b) `INVALID_REQUEST`, 400 (415 cho sai `Content-Type`), 0 ghi, **có** vết dừng ở C0, thông điệp nói rõ tenant, vai, phạm vi chỉ đến từ token; (c) 404, 0 ghi, không vết; (d) mã HTTP đúng bảng §2.6.2 và thân là `CommandResult` đủ trường; bốn route cũ cho kết quả như trước (A27) |
| **A29** | vết của lượt lệnh (§7.5): với mỗi kết cục của các ca A01, A03, A04, A07, A08, A10, A13, A14, A15, A16, A19, A20, A21 (a), A23, mở vết bằng `trace_id` của kết quả, qua `eais trace` và `GET /v1/trace/{id}`; thêm (e) lỗi lưu vết do F8 | (a) đúng chín bước C0, C1, C2, C3, C4, C4b, C5, C6, C7 theo thứ tự, C2 luôn `skipped`; (b) bước lệch đầu tiên (bước đầu có `deviant` hoặc `error`) là đúng chặng ở cột "Dừng ở đây thì" của §7.5; lượt `DONE`, kể cả phát lại, không có bước lệch; bước giữa bước dừng và C7 là `skipped`; (c) hai lượt `PERMISSION_DENIED` (máy không tồn tại; máy ngoài phạm vi) cho hai vết chỉ khác nhau ở khoá máy người gọi đã gõ, `trace_id`, đồng hồ và thời gian: mọi quyết định, lý do, trạng thái bước y hệt, và vết không chứa lý do thật; (d) vết lượt `DONE` mang mã phiếu và mã bằng chứng khớp bản chụp trong phiếu; (e) F8 không đổi kết quả đã ghi: 1 phiếu, kết quả như cũ, `trace_id` rỗng, bộ đếm lỗi lưu vết tăng |

---

## 11. Cổng ra nhị phân

### 11.1 Điều kiện

Cổng ra là **AND** của các dòng dưới; mỗi dòng nhị phân. X1–X3 là ba ca âm do roadmap đòi (`docs/ROADMAP-DAY-DU.md` §6, dòng Phase 1); X4–X9 là phần người viết **thêm** để cổng đủ chặt (X9 thêm ở lượt sửa theo lệnh ca MVP: cửa vào và vết của demo), chủ dự án có thể gạch bỏ.

| Cổng | Điều kiện đạt | Cách đo |
|---|---|---|
| **X1 gửi đôi** | A01–A06, A02b, A04b đều đạt | đếm dòng bằng kết nối riêng của harness; ca đua ép bằng F7 |
| **X2 thiếu quyền** | A07–A12 đều đạt | đếm dòng; kiểm mã kết quả |
| **X3 timeout sau khi ghi** | A21 và A22 đều đạt | điểm chèn lỗi F3, F4, F5 |
| X4 nguyên tử và hỏng giữa chừng | A19, A20, A23 đều đạt | điểm chèn lỗi F1, F2, F6 |
| X5 dữ liệu và luật | A13–A18 đều đạt | ca có đáp án tính tay |
| X6 biên | A24–A26 đều đạt | đếm lời gọi model; thử SQL trực tiếp |
| X7 không đổi đường hỏi | A27 đạt | so theo tên test; so từng câu trả lời của 117 / H1 / H3 / H4 |
| X8 đột biến | mỗi đột biến M1–M25 làm ít nhất một ca đỏ | chạy danh mục đột biến mới (tên và số do BUILD chọn) |
| X9 cửa vào và vết (demo M4) | A28 và A29 đạt | gọi route lệnh bằng token dev thật; đọc vết bằng `eais trace` |

**Kèm theo các cổng chung của repo** (`docs/PROMPT-BUILD-EAIS.md` §6): `go vet ./...` sạch; `gofmt -l` rỗng; `go test -count=1 ./...` xanh có PG, so theo tên; cổng danh tính xanh.

**Quy tắc đột biến.** Đột biến nào sống sót thì cổng đỏ, cho tới khi có ca bắt được, hoặc đột biến được ghi là tương đương kèm lý do và số đo (cách `scripts/dot-bien/n43-cong-phan-quyet.sh` đã làm).

### 11.2 Danh mục đột biến

| # | Đột biến | Ca phải đỏ |
|---|---|---|
| M1 | bỏ bước tra sổ lệnh (C4b) | A01, A21 |
| M2 | bỏ khoá chính (tenant, người gọi, `command_id`) | A02b |
| M3 | bỏ so fingerprint | A03 |
| M4 | bỏ chỉ mục duy nhất nghiệp vụ | A04b |
| M5 | đặt tra sổ lệnh **sau** luật "không trùng" | A01, A21 |
| M6 | bỏ kiểm `tickets.create` | A07 |
| M7 | bỏ kiểm phạm vi thực thể | A08 |
| M8 | bỏ kiểm `metrics.read` lúc chạy, hoặc bỏ luật nạp pack số 2 (§4.3) | A09 (ca phát triển); A12 |
| M9 | lấy tenant từ thân yêu cầu, hoặc bỏ đặt GUC tenant | A06, A10, A11, A28 |
| M10 | trả mã khác nhau cho "không tồn tại" và "ngoài phạm vi" | A08 |
| M11 | ghi audit `EXECUTED` ngoài giao dịch, sau commit | A20 |
| M12 | đọc lại bằng `RETURNING` hoặc bỏ đọc lại | A23 |
| M13 | gộp lỗi COMMIT vào `FAILED` (như `inTenant`) | A21, A22 |
| M14 | bỏ cờ `approx` khi lập phiếu | A17 |
| M15 | cấp UPDATE, DELETE cho `eais_app` trên ba bảng | A26 |
| M16 | thêm khoá ngoại từ phiếu tới `facts` hoặc `entities` | A18 |
| M17 | gọi model gateway trên đường lệnh | A24 |
| M18 | thêm mẫu câu "đã tạo phiếu" ở luồng hỏi | A25 |
| M19 | chọn một số khi hai nguồn ngang thẩm quyền lệch | A16 |
| M20 | bỏ phân loại lại sau vi phạm khoá duy nhất (§6.4) | A02, A02b, A04b |
| M21 | bỏ kiểm R5 ở C5 (còn chỉ mục duy nhất và §6.4) | A04 (nhờ điều kiện "đơn vị ghi phiếu không được gọi cho lần thứ hai"; thiếu điều kiện đó M21 tương đương về kết quả) |
| M22 | bỏ khoá ngoại từ sổ lệnh tới phiếu (§7.4) | A26 |
| M23 | đảo thứ tự ghi: sổ lệnh trước phiếu (§7.1) | A01 (khoá ngoại làm hỏng mọi lần tạo) |
| M24 | vết của lượt `PERMISSION_DENIED` ghi lý do thật (tách "không tồn tại" với "ngoài phạm vi"), hoặc vết lộ giá trị ngoài quyền đọc của người gọi | A29 (c) |
| M25 | không lưu vết cho lượt không phải `DONE` (chỉ lưu lượt thành công), hoặc để lỗi lưu vết đổi kết quả đã ghi | A29 (b), (e) |

**Ghi chú ánh xạ** [SUY; chưa chạy đột biến vì chưa có mã]:
- M2 chỉ bị giết bởi ca đua **khác nội dung** (A02b). Với ca cùng nội dung (A02), chỉ mục duy nhất của phiếu vẫn chặn lệnh thứ hai, rồi §6.4 phát lại: kết quả giống bản nguyên.
- M4 chỉ bị giết bởi ca đua **khác khoá** (A04b). Ca tuần tự (A04) bị R5 ở C5 chặn trước khi tới chỉ mục.
- M21 chỉ bị giết nhờ điều kiện "đơn vị ghi phiếu không được gọi cho lần thứ hai" của A04. Bỏ R5 ở C5 thì lần thứ hai vẫn gọi đơn vị ghi phiếu (bộ đếm (1) = 2); chỉ mục duy nhất và §6.4 rồi cho cùng kết quả `DUPLICATE_OPEN_TICKET` và cùng một dòng audit `REFUSED`. Vì vậy kết quả và audit đều không phân biệt được M21 với bản đúng; chỉ bộ đếm (1) phân biệt được. Dòng audit `REFUSED` có ở cả hai phía, nên không dùng nó để giết M21.
- M23 chỉ bị giết khi khoá ngoại của sổ lệnh còn nguyên (M22 chưa áp): thứ tự ghi là hệ quả của khoá ngoại, không phải bất biến riêng.

### 11.3 Mức bằng chứng

- Theo `docs/03-proof-harness.md` §4: dữ liệu cấp **D1** (fixture bẩn) và ca do người không viết mã soạn (thang chống tự xác nhận). **Không** phải D3 hay D4.
- Cổng ra **không đo hiệu năng**: chưa đặt trần độ trễ hay thông lượng (chưa đo; việc của Phase 5).
- Báo kết quả theo từng cổng X1–X9, không gộp một con số. Kết luận kèm câu: "trên dữ liệu giả lập, chưa có người nghiệp vụ duyệt".

### 11.4 Cổng ra của khối M4 theo lệnh ca (22/10)

Lệnh ca MVP của chủ dự án đặt bốn điều kiện cho cổng ra của M4 (điều kiện thứ nhất gồm ba nhóm ca âm), và đòi giấy này đo được chúng; mục 8 của lệnh ca thêm kịch bản demo. Bảng nói mỗi điều kiện đo bằng cổng và ca nào, và đột biến nào làm ca đó đỏ: bảy dòng, không dòng nào đo bằng lời.

| Điều kiện của lệnh ca | Cổng và ca đo | Đột biến làm ca đỏ (§11.2) |
|---|---|---|
| ca âm gửi đôi đỏ đúng | X1: A01–A06, A02b, A04b | M1 đến M5, M20, M21 |
| ca âm thiếu quyền đỏ đúng | X2: A07–A12 | M6 đến M10 |
| ca âm timeout sau khi ghi đỏ đúng | X3: A21, A22 | M1, M5, M13 |
| đúng MỘT phiếu cho mỗi chênh lệch | I5; A01, A02, A02b, A04, A04b: sau mọi tổ hợp gửi (tuần tự, đồng thời, cùng khoá, khác khoá), số phiếu `OPEN` của mỗi (máy, chỉ số, ngày) đếm bằng kết nối riêng của harness = 1 | M2, M4, M20, M21 |
| có đọc lại xác minh | I3; A21, A23 (X3, X4): `DONE` chỉ sau khi đọc lại thấy phiếu, kể cả phát lại | M12 |
| có audit | I4, §7.3; A01, A07, A20 (X1, X2, X4): đúng một dòng audit cho kết cục, `EXECUTED` cùng giao dịch với phiếu | M11 |
| demo chạy được, xem được vết cả hai lượt (lệnh ca mục 8) | X9: A28, A29 | M24, M25 (và M9 ở cửa HTTP) |

"Đỏ đúng" nghĩa là ca đỏ **vì đúng điều nó canh**: đột biến ở cột cuối làm ca đỏ ở đúng dòng "Phải thấy" của nó (§11.1, quy tắc đột biến), và bản đúng thì ca xanh. Đạt cả bảy dòng thì khối M4 qua cổng ra kỹ thuật, **trên dữ liệu giả lập, chưa có người nghiệp vụ duyệt**.

---

## 12. Giao cho người soạn bộ ca độc lập

Việc này **giao người khác, không làm ở tệp này.** Hợp đồng chỉ nêu ranh giới:

- **Hai tầng.**
  1. Ca phát triển: người viết mã soạn khi viết test đỏ trước.
  2. Bộ giữ lại độc lập: người **không đọc mã, không đọc pack thật**; chạy MỘT lần ở cổng ra, sau đó thành bộ hồi quy (cùng nguyên tắc `docs/adr/0006-bo-giu-lai-doc-lap.md`).
- **Được nhận:** §1, §2, §4.2, §5, §6, §7, §8, §9, §10 và §12.1 của tệp này; danh mục thực thể giả lập và dữ liệu hạt giống dạng dữ liệu; định nghĩa các điểm chèn lỗi F1–F8.
- **Không nhận:** mã; pack thật; ADR-0011 (phần phương án); §11.2 (danh mục đột biến).
- **Khuôn ca:** `docs/CONVENTIONS.md` §5 với `entry_type: COMMAND`; điểm chèn lỗi ghi ở khoá `failure_injection`; ca nhiều lời gọi và ca đồng thời điền giá trị theo §12.1.
- **Đóng băng:** soạn xong, băm và commit cam kết băm **trước** khi mã Phase 1 được viết. Quy ước ở §12.1 nằm trong phần được băm.
- **Phủ:** mọi lớp ca ở §10, trừ A09 (chỉ ca phát triển); mỗi lớp đủ biến thể để không khớp đề theo ví dụ trong tệp này. Ca đồng thời (A02, A02b, A04b, ép bằng F7) thuộc diện **bắt buộc** của người soạn độc lập: người viết mã không được là người duy nhất soạn ca đua (`CLAUDE.md` §2.1). Ca nhiều người gọi (A05, A06) cũng do người soạn độc lập viết, theo §12.1.

### 12.1 Khuôn ca cho ca nhiều lời gọi và ca đồng thời

**Vì sao cần.** Khuôn ở `docs/CONVENTIONS.md` §5 chỉ có MỘT `input`, MỘT `permissions.principal` và MỘT khoá `failure_injection`. Ca A02, A02b, A04b cần N lần chạy song song (A02b: N máy khác nhau; A04b: N khoá lệnh khác nhau); ca A05 cần hai người gọi; ca A06 cần hai tenant. Khuôn chưa có chỗ khai những thứ đó. Để trống thì người viết mã phải tự quyết số lời gọi, ai gọi và kết quả nào là đúng, đúng điều mục này cấm.

**Cách khai, không thêm khoá nào.** Giữ nguyên 23 khoá của §5, nên cổng hình dạng `evals.Validate` (`evals/cases.go`) không từ chối ca nào [CODE]. Mục này chỉ quy định cách điền giá trị khi `entry_type: COMMAND` và ca có nhiều lời gọi:

| Khoá của §5 | Ca một lời gọi | Ca nhiều lời gọi |
|---|---|---|
| `input` | một mục yêu cầu: `command_id`, `subject_key`, `metric`, `day` | **danh sách** mục yêu cầu; mỗi mục có `id`, `principal`, `command_id`, `subject_key`, `metric`, `day`. Mục có thêm `repeat: n` nghĩa là n lời gọi y hệt (cùng khoá, cùng nội dung) |
| `permissions` | `{principal: <vai>}` như §5 | `null`: người gọi nằm ở từng mục |
| `principal` của một mục | không có | `{tenant, id, role, entity_scope}`; `tenant` bỏ trống là tenant của ca; hai mục cùng `id` là cùng một người gọi |
| `failure_injection` | `{point: <F1…F6 hoặc F8>}` hoặc `null` | như ca một lời gọi, hoặc `{point: race_barrier, parallel: N}` (F7): N bằng số lời gọi sau khi mở `repeat`; hệ chỉ thả khi cả N đã tới rào |
| `preconditions`, `sources` | như §5 | như §5; dữ liệu hạt giống của tenant thứ hai (A06) khai ở đây |
| `expected_state` | trạng thái đếm được | số dòng đếm bằng kết nối riêng của harness: `tickets_open`, `ledger_rows`, và số dòng audit theo kết cục |
| `expected_output_properties` | tính chất kết quả | **số lượng** kết quả theo loại, ví dụ "đúng một lần `DONE` với `replayed` = false; N−1 lần `DUPLICATE_OPEN_TICKET`"; **không** nói lời gọi nào thắng |

Thứ tự trong danh sách là thứ tự **khai**, không phải thứ tự đến: ở ca song song, thứ tự đến do lịch chạy quyết, nên ca không được dựa vào nó (§10, F7).

**Ai viết gì.**

| Việc | Ai viết | Ghi chú |
|---|---|---|
| tệp ca: danh sách yêu cầu, người gọi, khoá, `parallel`, kỳ vọng | người soạn độc lập | không đọc mã, không đọc pack thật; kỳ vọng chỉ là số lượng và loại kết quả |
| bộ chạy chung: đọc tệp ca, dựng N lời gọi song song, rào F7, đếm bằng kết nối riêng, so với kỳ vọng | người viết mã (BUILD) | MỘT bộ chạy cho mọi ca, không rẽ nhánh theo mã ca. Nó phải gắn vào lớp bọc lưu trữ và rào F7 của chính mã, nên người soạn độc lập (không đọc mã) không viết được |
| kiểm bộ chạy | người soát chỉ đọc, cộng đột biến | M2 ⇒ A02b, M4 ⇒ A04b, M20 ⇒ A02, A02b, A04b phải đỏ: bộ chạy vô tình tuần tự hoá lời gọi thì các đột biến này sống sót và cổng đỏ (§11.1). F7 không đủ N lời gọi trong hạn thì ca đỏ (§10) |

**Đóng băng.** Quy ước ở mục này thuộc bộ ca đóng băng: băm gồm cả văn bản của §12.1; đổi quy ước sau khi đóng băng là đổi bộ ca, phải đóng băng lại kèm lý do. Nhờ vậy người viết mã không chỉnh được khuôn cho vừa bộ chạy của mình.

---

## 13. Điều kiện mở mã và thứ tự chứng minh

Chi tiết ở `docs/adr/0011-phase1-hanh-dong-co-kiem-soat.md` (mục "Chứng minh"). Tóm tắt điều kiện **trước khi có dòng mã đầu tiên**:
1. H4 đã chạy một lần (D-RM-04 = A).
2. Hợp đồng này đã qua soát độc lập và chủ dự án đã trả lời §14 (hoặc chấp nhận mặc định).
3. Bộ giữ lại độc lập (§12) đã đóng băng và băm.
4. ADR-0011 chuyển từ ĐỀ XUẤT sang CHỐT.

Riêng **route lệnh** (§2.6.2) chỉ thêm được khi `cmd/eais` của làn 0 (khối M1) đã có trên `main`; pipeline lệnh, kho phiếu và migration không phụ thuộc điều đó.

---

## 14. Câu mở cho chủ dự án

| Mã | Câu | Mặc định nếu chưa trả lời | Vì sao |
|---|---|---|---|
| **Q-P1-1** | Vai nào được tạo phiếu? | bảng §4.2 (`director`, `accountant`, `project_manager`) | quyền tối thiểu; đảo bằng sửa `policy.yaml`. Cần người nghiệp vụ (D-RM-03) |
| **Q-P1-2** | Ngưỡng mở phiếu: dạng nào, mẫu số nào? BR-FUEL-03 chỉ ghi "±1%/ngày" | ngưỡng tuyệt đối `min_abs_diff`, mặc định 0 | chưa rõ phần trăm của vế nào; ngưỡng tuyệt đối đo được ngay |
| **Q-P1-3** | Bắt đầu chỉ với `issue_vs_measured`? | có | hai vế từ nguồn có cấu trúc; `reported_vs_issued` phụ thuộc chữ tự do (lỗi im lặng L1) |
| **Q-P1-4** | Phiếu tạo nhầm khi chưa có huỷ thì xử lý sao? | chấp nhận trên dữ liệu giả lập; huỷ là một chuyển trạng thái của Phase 2 | thêm đường UPDATE ở Phase 1 phá I8 |
| **Q-P1-5** | Ghi audit cho cả lần gửi lại và lần từ chối? | có | vết đầy đủ; chi phí lưu nhỏ (chưa đo) |
| **Q-P1-6** | (khối M4) Luật vượt dung sai chạy bằng Principal `service` nào, vai và phạm vi nào? | chưa có vai nào, đóng mặc định: **bên kích hoạt tự động** của M4 chưa chạy được cho tới khi trả lời; lệnh do người gọi và kịch bản demo (§2.6.4) không bị chặn | cần người nghiệp vụ (D-RM-03) và chủ dự án; xem §1.4 |
| **Q-P1-7** | Chấp nhận **C4b** (tách Idempotency làm hai nửa: tra sổ lệnh ngay sau quyền), tức lệch thứ tự `CLAUDE.md` §14 và `docs/ROADMAP-DAY-DU.md` §5 luồng L3? | có | lần gửi lại sau timeout nhận `DONE` phát lại thay vì `DUPLICATE_OPEN_TICKET` (ca A21). Không chọn: R5 phải tự phân biệt "phiếu của chính tôi" với "của người khác"; ca A01, A21 và đột biến M5 đổi theo (§3) |
| **Q-P1-8** | (demo M4) Lượt "Tạo phiếu kiểm tra" của kịch bản demo là một **yêu cầu có cấu trúc** tới route lệnh riêng `POST /v1/commands/open_discrepancy_ticket` (a), hay là **câu chữ** gõ vào `/v1/ask` mà hệ nhận ra là lệnh (b)? | (a) | (a) đủ để làm M4 và demo, không đổi đường hỏi (I9), không cần C2 hay ngữ cảnh hội thoại. (b) cần hiểu lệnh từ chữ tự do và nối với "máy 03 hôm qua" của lượt trước: sửa `ask.go`, `router.go`, `intent.go` và đụng vòng S4 đã khoá, thuộc Phase 6. Chọn (b) thì ca A25 và mục "Không làm" của ADR-0011 đổi theo, và cần một ADR mới (§2.6.1) |

Ghi chú về `tenants/construction/pack/templates.yaml:16-17`: mẫu câu hứa "đã chuyển người phụ trách". Phase 1 **không đụng** vào đó và **không** nối kết quả lệnh vào mẫu câu của luồng hỏi. Chỉ khi có phiếu đọc lại được và có người nhận (Phase 2) mới được sửa lời hứa.

---

## 15. Điểm chưa kiểm chứng

| # | Chưa biết | Ảnh hưởng | Cách kiểm | Việc bị chặn |
|---|---|---|---|---|
| U1 | trình điều khiển pgx trả lỗi gì khi ngữ cảnh hết hạn đúng lúc COMMIT; có phân biệt được "chưa gửi" với "đã gửi" không | cách phân loại `COMMIT_UNKNOWN` ở §7.1 | thử nghiệm có chèn lỗi trên PostgreSQL thật khi mã mở | đơn vị ghi |
| U2 | thêm `tools.yaml` đổi `PackVersion` (băm mọi tệp `.yaml` của pack, `core/tenant/pack.go:304`); nơi nào ghim phiên bản (bằng chứng của `g7run`) | tái lập của W-G7 | `grep` các chỗ ghim; chạy lại danh mục đột biến liên quan | thêm tệp vào pack |
| U3 | `Derive` đọc hai vế bằng hai giao dịch riêng (`combine` gọi `Query` hai lần, `query.go:277`); `Load` xen giữa có thể cho hai vế lệch phiên bản dữ liệu [SUY] | chênh lệch sai trong cửa sổ hẹp | thử có chèn `Load` giữa hai lần đọc | tin vào R3 |
| U4 | `aggregate` có cộng đôi fact trùng hay không (`query.go:201`) | chênh lệch phồng | bơm dữ liệu trùng, đo | `CONG-QUANIT.md` câu 3 |
| U5 | chi phí lưu audit cho từ chối và phát lại | dung lượng | đo sau khi có mã | Q-P1-5 |
| U6 | Jira: chưa đối chiếu tài liệu REST chính thức; chưa thử thật | độ mạnh của câu 9 | đọc tài liệu Atlassian; thử trong môi trường thử | `CONG-QUANIT.md` câu 9 |
| U7 | người nghiệp vụ thật chốt vai, ngưỡng, chính sách trùng | mọi kết luận giá trị | D-RM-03 | tuyên bố giá trị |
| U8 | thiết kế một-giao-dịch chỉ đúng ở chế độ B; chế độ A (ghi sang hệ khác) cần thiết kế riêng cho "timeout sau khi ghi" | Phase 4b | ADR riêng khi tới 4b | Phase 4b |
| U9 | vết lượt lệnh với bảng vết của làn 0: (1) `ValidStage` nhận `C0`…`C7`, `C4b` mà không làm hỏng chỗ nào khác; (2) `FormatTable` in đúng "BƯỚC LỆCH ĐẦU TIÊN" cho vết lệnh; (3) `eais trace diff` (`FirstDifferentStep` chỉ so các trường quyết định của luồng hỏi [CODE `observability/traces/record.go`, làn 0]) có nghĩa gì với hai lượt lệnh; (4) làn 0 còn đang sửa theo soát, nên các dòng trích có thể lệch | demo M4, ca A29 | thử trên mã làn 0 đã gộp, khi route lệnh được thêm | §2.6, §7.5 |
