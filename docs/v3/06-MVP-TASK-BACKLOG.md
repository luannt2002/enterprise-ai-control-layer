# 06 — MVP Task Backlog: R0 → R6

> **Khung (D-RM-05, 22/10/2026):** tệp để tra cứu — roadmap hiện hành là `docs/ROADMAP-DAY-DU.md` (D-RM-01). Tên sản phẩm trong thân tệp là tên lúc viết; tên hiện hành là **EAIS** (D-RM-02).

> Trạng thái: bản lập kế hoạch 20/10/2026.
>
> | Nhãn | Nghĩa |
> |---|---|
> | READY | đủ Definition of Ready (Playbook §19) |
> | BLOCKED | chờ quyết định hoặc task khác |
> | ALREADY_DONE | đã xong, có bằng chứng |
> | NEEDS_VERIFICATION | có mã, chưa chứng minh |
> | NEEDS_DETAIL | khung, lập chi tiết khi phase mở |
> | DEFERRED | hoãn |
>
> **Lập kế hoạch cuốn chiếu.** Task của đợt đầu (R0, R1, R2, soạn bộ ca) có hợp đồng đầy đủ. Task R3–R6 ở mức khung: mục tiêu, AC, test, phụ thuộc, model. Chúng phải qua DoR khi phase mở, vì kết quả R2 sẽ đổi chi tiết.
>
> Mọi task dùng chung DoD mặc định của Playbook §1.4 cộng mục "DoD riêng". Không task nào được báo DONE khi thiếu bằng chứng ghi trong `09`.

## Luật chung cho mọi task cài đặt

1. Chạy mốc EV-B01…B04 (`02`) trước khi báo xong. Task chạm Fuel chạy thêm B05/B06.
2. Chỉ sửa tệp nằm trong "Tệp được sửa". Cần sửa ngoài danh sách thì dừng và báo lead (`CLAUDE.md` §33).
3. Sửa vào mã chung (`core/`, `data/`, `intelligence/`, `reliability/`, `observability/`, `platform/`) phải ghi một dòng lý do trong báo cáo task. `git diff --stat` thư mục chung là bằng chứng portability sớm.
4. Không commit/push. Lead gom và hỏi chủ dự án.
5. Báo cáo theo khuôn `CLAUDE.md` §31: Changed / Verified (lệnh + kết quả) / Not verified.

---

## R0 — Definition Freeze

### TASK-R0-01 — Chụp nguồn V3 có băm · ALREADY_DONE
- **Bằng chứng:** `docs/sources/eacl-v3/SHA256SUMS` (3 tệp; thư mục ignore). EV-R0-01.

### TASK-R0-02 — Duyệt gói định nghĩa và chốt D-V3-01…10 · BLOCKED (chờ chủ dự án)
- **Parent:** R0 · G1–G6 (định nghĩa).
- **Story:** là chủ dự án, tôi muốn chốt một lần thế nào là thắng, để agent không tự định nghĩa.
- **Tiền điều kiện:** `05` đã viết.
- **Request:** chủ dự án đọc `05` §2, §4; trả lời từng D-V3-*.
- **Ngoài phạm vi:** sửa mã.
- **AC-01:** mỗi D-V3-* có quyết định (chấp nhận/sửa/từ chối), ghi vào `05` §4 kèm ngày.
- **AC-02:** `05` §5 không còn ô "chờ duyệt" nào ngoài mục chủ dự án cố ý để mở.
- **DoD riêng:** revision ghi ở `09` (EV-R0-02).
- **PASS:** 0 blocker định nghĩa. **FAIL:** còn mâu thuẫn ở `05` §3 chưa có hướng xử lý.
- **Model:** lead + chủ dự án.

### TASK-R0-03 — Đổi tên trong tài liệu + đánh dấu tài liệu lịch sử · XONG 21/10/2026
- **Parent:** R0. **Yêu cầu:** C1, C2, C8, D-V3-06, D-V3-10.
- **Story:** là người đọc repo, tôi muốn thấy đúng một định nghĩa sản phẩm, để không làm theo bản cũ.
- **Tiền điều kiện:** R0-02 chốt tên và câu chữ.
- **Request / phạm vi:**
  - `README.md`: tên EACL, câu định nghĩa, trỏ `docs/v3/`;
  - thêm đầu tệp "Historical artifact; V3 xem `docs/v3/`" cho `docs/04-core-mvp.md`, `docs/05-master-roadmap.md`, `docs/requirements/catalog.md`;
  - trong tài liệu **mới**, cổng cũ ghi W-G1…W-G7;
  - soạn **đề xuất** câu chữ cho `CLAUDE.md` §9/§24/§25 vào `docs/v3/de-xuat-claude-md.md`.
- **Ngoài phạm vi:** sửa `CLAUDE.md` (tệp của chủ dự án); sửa nội dung `04`/`05` ngoài dòng đầu; sửa mã.
- **Tệp được sửa:** `README.md`, ba dòng đầu của 3 tệp trên, `docs/v3/de-xuat-claude-md.md` (mới).
- **AC-01:** `grep -n "Enterprise AI Systems Platform" README.md` ra 0.
- **AC-02:** cả ba tệp lịch sử có dòng đánh dấu ở 5 dòng đầu.
- **AC-03:** tệp đề xuất có câu thay thế cho từng mục §9/§24/§25, kèm lý do một dòng.
- **Test:** các lệnh `grep` trên; EV-B01…B03 không đổi (chỉ sửa tài liệu).
- **PASS:** đủ 3 AC. **FAIL:** đụng tệp ngoài danh sách.
- **Model:** Haiku (cơ học). **Soát:** Sonnet chỉ đọc.

### TASK-R0-04 — Bản công khai của Blueprint và Playbook · READY sau R0-02
- **Parent:** R0. **Yêu cầu:** C7, C11, C19.
- **Story:** là người đọc repo public, tôi muốn đọc đúng định nghĩa V3 mà không lộ dữ liệu riêng, không bị 14.000 dòng khuôn đánh lừa.
- **Tiền điều kiện:** R0-02.
- **Đặc tả:**
  - **(a) Playbook:** chép nguyên văn từ `docs/sources/eacl-v3/EACL_V3_SUCCESS_DEFINITION_PLAYBOOK.raw.md` vào `docs/v3/EACL_V3_SUCCESS_DEFINITION_PLAYBOOK.md`. Chỉ sửa fence của mẫu TASK để §4 không rơi vào khối thụt lề.
  - **(b) Blueprint:** chép nguyên văn §0–41, §47–49, §51–52 vào `docs/v3/EACL_V3_MASTER_BLUEPRINT.md`.
    - Thay tên hai hệ của công ty cũ bằng "hệ A (AI nhúng)" / "hệ B (AI đặt ngoài)".
    - Thay §42–46 bằng một mục "Catalog — nội dung riêng biệt": 10 lớp lỗi, 12 nhóm REQ, 12 quyết định, 7 nguồn bài học; kèm bảng đo nhân bản của `00` §4.1.
    - Bỏ §50 (rỗng).
- **Ngoài phạm vi:** viết lại câu chữ; dịch.
- **Tệp được sửa:** hai tệp mới ở `docs/v3/`.
- **AC-01:** script so dòng (ví dụ `diff <(sed …)`) chứng minh: ngoài các chỗ thay đã liệt kê, nội dung khớp từng dòng với bản gốc.
- **AC-02:** `grep -rn -i -E -f docs/sources/eacl-v3/ten-rieng.txt docs/v3/` ra 0. Danh sách tên riêng nằm trong thư mục bị ignore, không ghi vào tài liệu công khai.
- **AC-03:** Playbook công khai hiển thị §4 là tiêu đề, không phải khối mã (kiểm bằng `grep -n '^# 4\.'`).
- **PASS:** đủ 3 AC + script nằm trong báo cáo. **FAIL:** có câu bị sửa ngoài danh sách.
- **Model:** Haiku. **Soát:** script + Sonnet chỉ đọc.

### TASK-R0-05 — Đổi tên module, thư mục, repo · XONG 21/10/2026
- **Parent:** R0.
- **Story:** là chủ dự án, tôi muốn tên mã nguồn khớp tên sản phẩm trước khi mã V3 vào, để không phải đổi hai lần.
- **Đặc tả (nếu duyệt):**
  - `go.mod` module `github.com/luannt2002/enterprise-ai-control-layer` (khớp tên repo GitHub, để `go get` tìm được);
  - thay mọi đường import;
  - đổi thư mục máy;
  - repo GitHub đổi tên: chủ dự án tự làm, hoặc duyệt lệnh `gh` cụ thể;
  - sửa đường dẫn trong `docs/ops/*`.
- **AC-01:** `grep -rn 'enterprise-ai-systems' --include='*.go' .` ra 0.
- **AC-02:** EV-B01…B06 giống mốc (224/0/0, 117/117, scale smoke ĐẠT).
- **AC-03:** remote mới `git ls-remote` được.
- **Model:** Haiku (cơ học) + lead kiểm. **Ngoài phạm vi:** đổi tên gói hay định danh trong mã.

### TASK-R0-06 — Ghi revision gói R0 · BLOCKED (D-V3-08)
- **Đặc tả:** sau khi chủ dự án cho commit, ghi commit/tag `v3.0.0-r0` vào `09`.
- **PASS:** `git show <rev>:docs/v3/05-R0-DEFINITION-FREEZE.md` khớp bản đã duyệt.

---

## R1 — Fuel Truth Benchmark

### TASK-R1-01 — Giữ hồi quy Fuel xanh · ALREADY_DONE, kiểm liên tục
- **Bằng chứng:** EV-B03…B07. Mọi task sau chạy lại.

### TASK-R1-02 — Ghi chú xung đột nói đúng phạm vi ở câu tổng (OBS-01) · READY
- **Parent:** R1 · G2 TRUTH. **Yêu cầu:** V3-FUEL-02.
- **Story:** là kỹ sư công trường, tôi muốn biết số lệch thuộc máy nào, để không hiểu nhầm cả dự án chỉ xuất 3.900 L.
- **Lỗi nghiệp vụ:** ca WIN-I-115 giao "Nguồn khác lệch số: ERP 3.900 L" sau con số 6.550 L của cả dự án (`02` §4).
- **Tiền điều kiện:** mốc `02` xanh.
- **Đặc tả:**
  - khi câu trả lời là tổng của nhiều chủ thể, ghi chú xung đột phải nêu chủ thể của từng chỗ lệch (ví dụ "ERP ghi EX-03 3.900 L");
  - câu một chủ thể giữ nguyên chữ hiện tại;
  - sửa ở tầng mẫu câu/pack và chỗ dựng `{conflicts}`, không vá riêng ca 115;
  - `BindClaims` phải vẫn ràng buộc được số mới (chủ thể đúng).
- **Ngoài phạm vi:** sửa bộ ca đông cứng (giữ băm `ffdd704b…`); đổi luật thẩm quyền.
- **Tệp được sửa:** `tenants/construction/pack/templates.yaml` và mã dựng ghi chú xung đột (xác định khi đọc mã); test mới trong `tenants/construction/`.
- **AC-01:** test mới: câu tổng dự án có một chỗ lệch ⇒ ghi chú chứa tên chủ thể của chỗ lệch. Trên mã cũ test **đỏ**, sau sửa xanh.
- **AC-02:** câu một máy (ca "3.850 L. Nguồn khác lệch số: ERP 3.900 L") không đổi chữ.
- **AC-03:** 117/117 giữ nguyên trên cả hai đường; phát lại 0 lệch.
- **Test:** test mới + EV-B03…B07; một đột biến (bỏ chủ thể) phải bị test mới bắt.
- **PASS:** đủ 3 AC + đột biến bị bắt. **FAIL:** đổi kỳ vọng ca cũ để xanh.
- **Model:** Sonnet. **Soát:** Sonnet độc lập chỉ đọc.

### TASK-R1-03 — B1 "LLM trực tiếp" cho Fuel · READY
- **Parent:** R1 · G6. **Yêu cầu:** V3-FUEL-03, C14, C18.
- **Story:** là kiến trúc sư, tôi muốn so EACL với cách tích hợp ngây thơ nhất, để biết tầng sự thật có đáng giá.
- **Tiền điều kiện:** `05` §2.4 được duyệt (hoặc lead tạm dùng, đảo được).
- **Đặc tả:**
  - B1 nhận câu hỏi + **toàn bộ bản ghi thô** của tenant A trong cửa sổ ngày của ca (±1 ngày), viết thành văn bản. Không chuẩn hoá, không nhận dạng, không truy xuất.
  - Gọi **cùng** `models.Gateway`. Prompt cố định trong mã, có băm.
  - Đầu ra đi qua cùng bộ đếm độc lập như B2.
- **Ngoài phạm vi:** gọi API thật; đổi B2.
- **Tệp được sửa:** gói mới `evals/direct/` (hoặc cạnh `evals/baseline`); cờ chọn hệ trong `g7run`.
- **AC-01:** test bằng gateway giả đếm lời gọi: mỗi câu đúng 1 lời gọi.
- **AC-02:** ngữ cảnh chỉ chứa bản ghi của tenant A trong cửa sổ, kiểm bằng test.
- **AC-03:** chạy hai lần ra prompt giống từng byte.
- **AC-04:** `g7run` in kết quả ba hệ.
- **Test:** unit + chạy `g7run` với provider giả.
- **PASS:** đủ AC; mốc giữ xanh. **FAIL:** B1 dùng bất kỳ thành phần kiểm soát nào của EACL.
- **Model:** Sonnet. **Soát:** Sonnet độc lập.

### TASK-R1-04 — Ước chi phí offline cho lần chạy thật · READY sau R1-03
- **Parent:** R1 · G6. **Yêu cầu:** V3-FUEL-04, D-V3-01.
- **Story:** là chủ dự án, tôi muốn biết trước số token và tiền, để duyệt ngân sách có cơ sở.
- **Đặc tả:**
  - dựng **đúng** prompt mà B1, B2, B3 sẽ gửi cho 117 ca (+ ca R1-05 nếu đã đông cứng), **không gọi API**;
  - đếm ký tự, ước token theo hệ số ghi rõ; nếu có endpoint đếm token thì dùng sau khi có khoá;
  - nhân k = 3;
  - xuất bảng: theo hệ, theo lớp lỗi, tổng; kèm giá theo bảng giá cấu hình, không hard code.
- **AC-01:** bảng có đủ ba hệ và tổng.
- **AC-02:** ghi rõ hệ số ước lượng và "CHƯA ĐO token thật".
- **AC-03:** lệnh tái lập được.
- **PASS:** chủ dự án có con số để quyết D-V3-01. **Model:** Sonnet. **Soát:** lead.

### TASK-R1-05 — Bộ ca đội đỏ Fuel (~40 ca mới) · READY
- **Parent:** R1 · G1/G2. **Yêu cầu:** V3-FUEL-07.
- **Story:** là kiến trúc sư, tôi muốn biết hệ có tổng quát hoá ngoài 117 ca không.
- **Đặc tả:**
  - Tác giả là agent **không được đọc mã**. Chỉ đọc `docs/04-core-mvp.md`, `docs/tenants/construction/*`, `tenants/construction/win/world.yaml`.
  - Soạn ca theo 8 lớp `04` §5, định dạng như `cases-independent.yaml`, ghi `authored_by: independent-r1`.
  - Đông cứng sha256 **trước** khi chạy.
- **AC-01:** ≥ 40 ca, mỗi lớp ≥ 4.
- **AC-02:** tệp có băm ghi trong `09` trước lần chạy đầu.
- **AC-03:** chạy tất định ra bảng đạt/đỏ. Ca đỏ **không sửa**; mỗi ca đỏ thành mục điều tra.
- **PASS:** có số tổng quát hoá + danh sách ca đỏ có phân loại. **FAIL:** tác giả đọc mã, hoặc sửa ca sau khi thấy kết quả.
- **Model:** Sonnet (độc lập).

### TASK-R1-06 — Chạy model thật k = 3 cho B1/B2/B3 · BLOCKED (D-V3-01, D-V3-02)
- **Đặc tả:** temperature 0; cùng model; trần chi phí do chủ dự án đặt, harness dừng khi chạm trần; lưu số thô; áp luật `05` §2.5.
- **AC:** bảng theo lớp × hệ × lượt; trung vị; kết luận máy tính ra.
- **Model:** Sonnet chạy; lead đọc số thô.

### TASK-R1-07 — Báo cáo R1 · BLOCKED (R1-06)
- **Đặc tả:** `docs/bao-cao/<ngày>-r1-truth-benchmark.md`: phương pháp, revision, băm bộ ca, model, chi phí, số thô, thất bại, kết luận WIN/PARTIAL/FAIL, giới hạn (D2, ca đội đỏ).
- **Model:** lead.

### TASK-R1-08 — Soát độc lập mã W4–W7 · READY
- **Phạm vi:** S9 trong `core/pipeline`, `platform/pgstore`, so sánh/mệnh đề trọng tâm (`intelligence/intent`).
- **Đặc tả:** phiên **không viết** mã đó; chỉ đọc + chạy test. Mỗi phát hiện phải có ca tái hiện hoặc lý do.
- **AC:** danh sách phát hiện có mức nghiêm trọng; 0 sửa mã trong task này.
- **Model:** Sonnet chỉ đọc. Phát hiện quay về task sửa riêng.

---

## R2 — App tham chiếu Customer Operations (không AI)

### TASK-R2-01 — ADR-0003: kiến trúc app tham chiếu + hợp đồng API · READY (sau R0-02)
- **Parent:** R2. **Yêu cầu:** D-V3-09, V3-APP-*, R-08.
- **Story:** là kiến trúc sư, tôi muốn app tham chiếu là một "hệ có sẵn" thật, để số đo công tích hợp không bị sân nhà làm lệch.
- **Đặc tả: quyết định cần ghi trong `docs/adr/0003-app-tham-chieu-customer-ops.md`.**

  1. **Vị trí và ranh giới.**
     - App ở `examples/customerops/`: tiến trình riêng, `cmd/customerops`.
     - Vertical EACL ở `tenants/customerops/`, theo cách đặt của `tenants/construction`.
     - Test kiến trúc **cấm import hai chiều**.
     - Khuyến nghị cùng `go.mod` (đơn giản, `go test ./...` phủ hết). Phương án module riêng ghi trong phần đã cân nhắc.
  2. **Dữ liệu.**
     - DB riêng `customerops`, vai riêng, cùng container PG dev.
     - Lọc tenant ở tầng app + test. EACL **không** được dựa vào việc app có RLS.
  3. **Xác thực.**
     - Token bearer cho người dùng seed (dữ liệu giả, ghi rõ chỉ cho dev, không phải bí mật thật).
     - Tenant chỉ lấy từ token.
     - EACL gọi app **thay mặt người dùng** bằng token của chính người đó; EACL không có quyền nào hơn người dùng.
  4. **API.** REST JSON, `net/http` thư viện chuẩn (Go 1.22+ có pattern theo method), đúng 7 thao tác của Blueprint §30:
     - `GET /v1/customers/{id}`;
     - `GET /v1/customers?q=`;
     - `GET /v1/customers/{id}/activities?from=&to=`;
     - `GET /v1/customers/{id}/tasks?status=`;
     - `GET /v1/customers/{id}/documents?from=&to=`;
     - `POST /v1/tasks` (header `Idempotency-Key`);
     - `GET /v1/tasks/{id}` và `GET /v1/tasks?idempotency_key=`.
  5. **Quyền:** `customer.read`, `document.read`, `task.create`. Mọi đường đều kiểm.
  6. **Mã lỗi:** 401, 403, 404 (kể cả id của tenant khác, không lộ là có tồn tại), 409 (cùng khoá khác nội dung), 400.
  7. **Fixture:**
     - tenant A: ACME **là khách duy nhất khớp "Acme"** trong A, có 30 ngày activity, task mở/đóng, tài liệu;
     - tenant A: một **cặp gần tên riêng** cho ca mơ hồ (ví dụ "Bình Minh Steel" / "Bình Minh Logistics"), như Fuel dùng "máy 03" duy nhất và cặp "máy 07" cho ca mơ hồ;
     - canary B: một "Acme" trùng tên (để thử chéo tenant);
     - người dùng: `sales_a` (đọc + tạo), `viewer_a` (chỉ đọc khách), `sales_b`;
     - ngày tuyệt đối, có băm.
  8. **Ngoài phạm vi:** UI (để D-V3-04), sửa/xoá khách, Contact (C9), workflow.
- **AC-01:** ADR có đủ 8 mục + phương án đã cân nhắc + điều kiện đảo.
- **AC-02:** chủ dự án duyệt.
- **AC-03:** đặc tả fixture đủ để agent soạn ca (R3-00/R4-00) mà không cần đọc mã.
- **Model:** lead (Opus).

### TASK-R2-02 — Lược đồ, migration, seed tất định · READY sau R2-01
- **Story:** là kỹ sư, tôi muốn dữ liệu nghiệp vụ tái lập được từng byte, để mọi ca đo trên cùng một thế giới.
- **Đặc tả:**
  - bảng `tenants`, `users`, `user_permissions`, `customers`, `activities`, `tasks` (`idempotency_key`, `request_hash`, `version`, `created_by`, `created_at`), `documents` (ACL), `app_audit`;
  - migration chạy hai lần vẫn không lỗi;
  - seed theo ADR, có lệnh in băm.
- **Tệp được sửa:** `examples/customerops/{migrations,store,seed}/…`.
- **AC-01:** migrate + seed trên DB sạch thành công hai lần liên tiếp.
- **AC-02:** băm seed ổn định qua hai lần chạy.
- **AC-03:** `UNIQUE (tenant_id, idempotency_key)` tồn tại; test chèn trùng bị DB chặn.
- **Test:** tích hợp với PG (bỏ qua có lý do khi không có biến PG, như mốc hiện tại).
- **PASS:** đủ AC + EV-B03 không giảm. **Model:** Sonnet. **Soát:** ở R2-07.

### TASK-R2-03 — Xác thực, quyền, cách ly tenant · READY sau R2-02
- **Story:** là chủ app, tôi muốn mọi lời gọi đều xác định được người gọi và tenant, để không ai đọc chéo.
- **Đặc tả:**
  - middleware token ⇒ `actor{id, tenant, permissions}`;
  - tenant lấy **chỉ** từ token; tham số tenant trong query/body bị bỏ qua;
  - helper kiểm quyền theo thao tác.
- **AC-01:** token thiếu/sai ⇒ 401.
- **AC-02:** thiếu quyền ⇒ 403. Cho phép và từ chối đều có test cho **mỗi** route.
- **AC-03:** id của tenant khác ⇒ 404, nội dung phản hồi giống hệt id không tồn tại.
- **Test:** bảng cho phép/từ chối × route; một đột biến bỏ kiểm quyền phải bị bắt.
- **Model:** Sonnet.

### TASK-R2-04 — API đọc · READY sau R2-03
- **Đặc tả:** `get_customer`, `find_customer` (khớp tên trong tenant, trả **mọi** ứng viên, không tự chọn), `list_activities` (kỳ), `list_tasks` (trạng thái), `get_documents` (kỳ, lọc ACL).
- **Thân thiện audit:** mọi phản hồi có `request_id`, mỗi bản ghi có `version`, `updated_at`.
- **AC-01:** dữ liệu khớp seed theo bảng kỳ vọng.
- **AC-02:** `find_customer("Acme")` trong A ra đúng 1 (ACME của A), trong B ra 1 (Acme của B, khác id). `find_customer("Bình Minh")` trong A ra 2 ứng viên.
- **AC-03:** tài liệu ACL riêng không trả cho người thiếu quyền.
- **Model:** Sonnet.

### TASK-R2-05 — Tạo task có idempotency và kiểm hợp lệ · READY sau R2-03
- **Story:** là chủ app, tôi muốn gửi lại cùng yêu cầu không tạo thêm task, để bên gọi thử lại an toàn.
- **Đặc tả:**
  - `POST /v1/tasks` cần `Idempotency-Key`;
  - cùng khoá, cùng nội dung (so `request_hash`) ⇒ trả task cũ (200, đánh dấu `replayed`);
  - cùng khoá, khác nội dung ⇒ 409;
  - kiểm: khách tồn tại trong tenant và đang hoạt động, hạn hợp lệ (không ở quá khứ theo múi giờ tenant), tiêu đề không rỗng;
  - `GET /v1/tasks/{id}`; `GET /v1/tasks?idempotency_key=`;
  - ghi `app_audit`.
- **AC-01:** gửi 3 lần cùng khoá ⇒ 1 dòng trong DB.
- **AC-02:** gửi đồng thời 10 lần cùng khoá ⇒ 1 dòng (race test).
- **AC-03:** 409 khi khác nội dung.
- **AC-04:** mọi trường sai ⇒ 400 có mã lỗi.
- **AC-05:** tra theo khoá ra đúng task.
- **Model:** Sonnet. **Soát:** lead đọc thiết kế khoá trước khi cài.

### TASK-R2-06 — Móc bơm lỗi chỉ dùng trong test · READY sau R2-05
- **Story:** là kỹ sư R4/R5, tôi muốn tái hiện "ghi xong mất phản hồi" một cách tất định.
- **Đặc tả:**
  - tuỳ chọn chỉ bật khi khởi tạo app trong test: trễ N ms, cắt kết nối **sau** khi commit, trả 5xx trước khi ghi;
  - không đọc từ biến môi trường production.
- **AC-01:** test chứng minh "cắt sau commit" có ghi DB mà client nhận lỗi.
- **AC-02:** build `cmd/customerops` không thể bật móc này.
- **Model:** Sonnet.

### TASK-R2-07 — Soát độc lập + báo cáo R2 · READY sau R2-02…06
- **Đặc tả:**
  - phiên không viết mã app chạy bộ test từ môi trường sạch, soát theo V3-APP-01…07 và V3-SEC-08, thử ít nhất 3 đột biến (bỏ kiểm quyền, bỏ lọc tenant, bỏ UNIQUE khoá);
  - lead viết `docs/bao-cao/<ngày>-r2.md` với kết luận PASS/FAIL.
- **PASS (Playbook R2):** không AI; xác thực, cách ly tenant, fixture, tạo task, idempotency, API thân thiện audit đều có bằng chứng.
- **Model:** Sonnet chỉ đọc + lead.

---

## R3 — UNDERSTAND (Job A) · khung, NEEDS_DETAIL trừ R3-00

### TASK-R3-00 — Bộ ca Job A đông cứng (tác giả độc lập) · READY sau R2-01
- **Đặc tả:**
  - agent **không đọc mã**, chỉ đọc ADR-0003 + đặc tả fixture + `05` §2;
  - soạn ca cho các lớp áp dụng của `05` §2.6;
  - mỗi ca: câu hỏi, người hỏi, kỳ vọng (outcome, fact chính xác, chuỗi cấm, `model_calls`, ứng viên khi mơ hồ);
  - có ca fact chính xác ("Acme có bao nhiêu task đang mở?" ⇒ 0 lời gọi model);
  - đông cứng sha256.
- **AC:** ≥ 30 ca Job A; mỗi lớp áp dụng ≥ 5; `authored_by` đầy đủ. **Model:** Sonnet độc lập.

| Task | Mục tiêu | AC/yêu cầu | Phụ thuộc | Model |
|---|---|---|---|---|
| R3-01 | Adapter Customer Ops: interface không phụ thuộc transport; client HTTP gửi token người dùng; bản mock cho unit test; nhãn lỗi `third_party:customerops:*` | V3-SEC-01, V3-SEC-05 | R2 PASS | Sonnet |
| R3-02 | Resolver khách: theo tenant, mơ hồ ⇒ `CLARIFICATION_REQUIRED` kèm ứng viên; bí danh từ pack | V3-AC01, V3-AC02 | R3-01 | Sonnet |
| R3-03 | Ý định + route Job A: brief ⇒ `MODEL`; fact chính xác ⇒ `API`, 0 lời gọi model | V3-AC06 | R3-02 | Sonnet |
| R3-04 | Trusted context builder: đọc có lọc quyền, id/phiên bản nguồn, đánh dấu xung đột/bất định, tách dữ liệu khỏi chỉ thị, ngân sách | V3-AC03, V3-SEC-02/04/07 | R3-02 | Sonnet; **lead soát** |
| R3-05 | Schema brief (mục, gạch, `evidence_ids`, fact có kiểu) + gọi gateway + xử lý JSON hỏng | V3-AC04, V3-FAIL-08 | R3-04 | Sonnet |
| R3-06 | Validator brief: id có trong ngữ cảnh, số/ngày/trạng thái khớp fact, khẳng định quan trọng bắt buộc có bằng chứng | V3-AC04, V3-AC05 | R3-05 | Sonnet; **lead soát** |
| R3-07 | Trace Job A đủ trường + móc ghi chi phí | V3-AC12, V3-OBS-01, V3-COST-01 | R3-03 | Sonnet |
| R3-08 | Runner eval Job A (model giả đối kháng) + báo cáo G1/G2/G3/G5; smoke model thật nếu D-V3-01 cho phép | R3 PASS | R3-00…07 | Sonnet + soát độc lập |

---

## R4 — ACT (Job B) · khung, NEEDS_DETAIL trừ R4-00

### TASK-R4-00 — Bộ ca Job B đông cứng (tác giả độc lập) · READY sau R2-01
- **Đặc tả:** như R3-00 cho Job B, gồm:
  - ca hợp lệ;
  - khách mơ hồ;
  - "thứ Hai" khi hôm nay là thứ Hai / Chủ nhật;
  - người thiếu `task.create`;
  - ngày quá khứ;
  - thử lại;
  - mất phản hồi;
  - trạng thái cũ;
  - đổi quyền giữa chừng;
  - model tự cấp quyền.
- **AC:** ≥ 30 ca Job B; ≥ 5 cho lớp Action Safety. **Model:** Sonnet độc lập.

| Task | Mục tiêu | AC/yêu cầu | Phụ thuộc | Model |
|---|---|---|---|---|
| R4-01 | ADR-0004 thiết kế hành động: `ActionProposal` (Blueprint §12); `AuthorizedAction` chỉ gói policy dựng được; máy trạng thái PROPOSED → VALIDATED → AUTHORIZED → EXECUTING → EXECUTED/UNKNOWN → VERIFIED/VERIFY_FAILED; khoá idempotency = băm(tenant, actor, đề xuất chuẩn hoá, mã yêu cầu); luật thử lại theo nhãn lỗi; đọc lại | V3-AC07…13 | R2-05 | **lead (Opus)** |
| R4-02 | Bộ phân tích tất định câu follow-up (khách, ngày theo múi giờ tenant, tiêu đề). Diễn giải bằng model nằm sau cổng capability **đóng mặc định**, chỉ mở khi bộ tất định trượt trên bộ câu đông cứng | V3-AC07, V3-AC09 | R4-01, R3-02 | Sonnet |
| R4-03 | Validator + quyền + policy (mức rủi ro, cần duyệt) ⇒ `AuthorizedAction` | V3-AC08/09, V3-SEC-03/10 | R4-02 | Sonnet; **lead soát an ninh** |
| R4-04 | Executor: khoá, máy trạng thái, thử lại, đọc lại theo id/khoá, so với đề xuất | V3-AC10/11/13, V3-FAIL-09…11 | R4-03 | Sonnet; **lead soát** |
| R4-05 | Audit lưu bền (bảng có RLS trong DB của EACL) + trace hành động | V3-AC12, V3-SEC-09 | R4-04 | Sonnet |
| R4-06 | Móc duyệt tối thiểu (`PENDING_APPROVAL` → người có quyền duyệt → thực thi) | V3-SEC-10 | R4-03 | Sonnet |
| R4-07 | Test lỗi Job B + test kiến trúc "một đường ghi" | V3-FAIL-05/06/08–12, V3-SEC-11 | R4-04 | Sonnet |
| R4-08 | Runner eval Job B + báo cáo G4/G5 | R4 PASS | R4-00…07 | Sonnet + soát độc lập |

---

## R5 — Security + Failure Injection · khung

| Task | Mục tiêu | Yêu cầu | Model |
|---|---|---|---|
| R5-01 | Bộ đối kháng hợp nhất A + B + Fuel, do đội đỏ độc lập soạn (không đọc mã) | V3-FAIL-01…12 | Sonnet độc lập |
| R5-02 | Bơm lỗi lúc chạy qua transport tiêm được của gateway và adapter (timeout, 429/5xx, JSON hỏng, cắt sau commit) | V3-FAIL-07/08/10 | Sonnet |
| R5-03 | Quét prompt/log: không bí mật, không nội dung bị cấm | V3-SEC-07, V3-OBS-02 | Sonnet |
| R5-04 | Báo cáo rủi ro còn lại + kết luận R5 | Playbook R5 | lead |

## R6 — Baseline + Value Benchmark · khung

| Task | Mục tiêu | Yêu cầu | Trạng thái | Model |
|---|---|---|---|---|
| R6-01 | Đông cứng phương pháp + ngưỡng (băm, có trước lần chạy) | V3-VAL-01 | BLOCKED (D-V3-02) | lead + chủ dự án |
| R6-02 | Harness B1/B2 cho Customer Ops theo `05` §2.4 | V3-AC14 | NEEDS_DETAIL | Sonnet |
| R6-03 | Bảng giá cấu hình + chi phí trên kết quả đã xác minh | V3-COST-01/02 | NEEDS_DETAIL | Sonnet |
| R6-04 | Đo công tích hợp (dòng mã adapter, `git diff --stat` thư mục chung, giờ công) | V3-VAL-02 | NEEDS_DETAIL | Haiku |
| R6-05 | Giao thức B0 (+ trang HTML tối thiểu nếu D-V3-04 chọn) | V3-VAL-03 | BLOCKED (D-V3-04) | Sonnet + chủ dự án |
| R6-06 | Chạy thật k = 3 bốn đường | V3-AC14 | BLOCKED (D-V3-01) | Sonnet |
| R6-07 | Báo cáo quyết định MVP: WIN / PARTIAL / FAIL; "Architecture Proof PASS; Product MVP …" nếu G6 trượt | V3-AC15 | BLOCKED | lead |

## DEFERRED (có điều kiện mở)

| Hạng mục | Mở khi |
|---|---|
| Contact | một ca đông cứng cần (C9) |
| UI app | D-V3-04 chọn |
| RAG cho tài liệu khách | R3-08 cho thấy truy vấn tất định theo `customer_id` + kỳ không đủ (có ca đỏ) |
| Model diễn giải ý định | bộ tất định trượt trên bộ câu đông cứng (R4-02) |
| Máy đổi dự án giữa kỳ (lỗ hổng 5 cũ) | một ca đỏ vì nó |
| HTTP API của EACL | tích hợp thứ hai cần gọi EACL qua mạng (R8) |
| Agent, MCP, fine-tune, RAG nâng cao, workflow, ontology | `05` §2.10 |
