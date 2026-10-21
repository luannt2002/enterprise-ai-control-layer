# Roadmap đầy đủ: nền tảng AI cho doanh nghiệp, phạm vi rộng nhất

> **Trạng thái: HIỆN HÀNH** từ tối 21/10/2026, theo **D-RM-01** (`docs/CHO-QUYET-DINH.md` §0). Tệp này thay vai trò roadmap hiện hành của V3 (R0–R12, `docs/v3/04`). `docs/ROADMAP.md` từ nay chỉ dùng để tra cứu.
>
> **Viết cho:**
> - chủ dự án;
> - phiên BUILD (team agent). Đầu mỗi ca, BUILD đọc tệp này cùng `docs/PROMPT-BUILD-EACL.md`.
>
> **Tên:** theo **D-RM-02**.
> - Toàn bộ code, luồng và tài liệu đổi lại về **tên gốc ngày 20/10** (`d70ebc8`).
> - **Link git giữ nguyên**: tên repo, remote, đường dẫn module Go.
> - BUILD làm việc đổi tên (brief §4, mục 1).
> - Tới lúc đó, tệp này gọi tắt là "dự án", để không làm đỏ cổng danh tính.
>
> **Mốc đo:** `d6afad7` (origin/main, 21/10 16:44).
>
> **Nhãn:**
> - [ĐO]: phiên viết tự chạy lệnh;
> - [REPO]: tài liệu repo tự ghi, chưa chạy lại;
> - [CODE]: đọc thấy trong mã;
> - [ĐỀ XUẤT]: chưa quyết, đảo được.
>
> **Thứ bậc khi mâu thuẫn:** `CHO-QUYET-DINH.md` > ADR > hợp đồng (`docs/04`, `docs/g7/HOP-DONG-G7.md`) > `CLAUDE.md` > tệp này > `docs/ROADMAP.md`.

---

## 0. Năm câu

1. Dự án là **nền tảng multi-tenant**: biến dữ liệu, tri thức, quy trình và công cụ của doanh nghiệp thành năng lực AI có kiểm soát. Dự án đi theo **phạm vi rộng nhất** của roadmap gốc 20/10 (`docs/05-master-roadmap.md`, Phase 0–10), không thu hẹp như V3.
2. Có **hai chế độ sở hữu**:
   - **A. Tích hợp:** app của khách giữ sự thật, quyền, giao dịch.
   - **B. Tự lo:** dự án tự thu, chuẩn hoá và đối soát dữ liệu.
3. Mọi việc đi qua **hai cổng**:
   - **cổng QuânIT:** 9 câu hỏi phía doanh nghiệp;
   - **cổng gốc:** điều kiện tiên quyết, bằng chứng, câu hỏi "ai sở hữu?".
4. **Đi rộng bằng lát mỏng xuyên suốt.** Mỗi loại tương tác (QUERY, COMMAND, WORKFLOW, EVENT) phải có một lát chạy được từ đầu tới cuối trước, rồi mới làm sâu.
5. Hiện chỉ **Phase 0b (Fuel, QUERY)** có mã, và đang **0/7 cổng** [REPO `STATUS.md`]. Mọi phase khác là kế hoạch.

---

## 1. Định nghĩa

> Nền tảng multi-tenant biến dữ liệu, tri thức, quy trình và công cụ của doanh nghiệp thành năng lực AI **có kiểm soát, kiểm thử được và quan sát được**.
> - **Dùng lại** hệ thống doanh nghiệp đã có khi hệ đó đang giữ sự thật.
> - **Tự lo** phần doanh nghiệp còn thiếu: thu dữ liệu, chuẩn hoá, đối soát.
> - Core **không gắn** vào một ngành.

Câu gốc ngày 20/10 (`d70ebc8`: `CLAUDE.md` §9, `docs/01-architecture.md:3`) được giữ, thêm **hai chế độ sở hữu**. Đây chính là chỗ đã mâu thuẫn giữa Fuel và V3.

| Chế độ | Khi nào | Ai giữ sự thật, quyền, giao dịch | Ví dụ |
|---|---|---|---|
| **A. Tích hợp** | doanh nghiệp đã có CRM / ERP / app có API | **app của khách**; dự án đọc và ghi qua API, dùng token của người dùng | CRM: tóm tắt khách, tạo task |
| **B. Tự lo** | doanh nghiệp chỉ có Zalo / Excel / giấy | **dự án**: thu, chuẩn hoá, đối soát, lưu trữ | báo cáo dầu qua Zalo (Fuel) |

**Không làm:**
- thay CRM / ERP của khách khi khách đã có;
- framework chatbot / RAG / agent bán riêng;
- ontology chung cho mọi ngành;
- đọc database của khách vòng qua quyền của app (chế độ A);
- thêm năng lực khi chưa qua hai cổng (§4).

---

## 2. Phạm vi rộng nhất

| Chiều | Phạm vi đầy đủ |
|---|---|
| **Loại tương tác** | QUERY (hỏi) · COMMAND (làm) · WORKFLOW (quy trình nhiều bước) · EVENT (tự phản ứng) |
| **Kiểu doanh nghiệp** | ① CRM trưởng thành · ② phần mềm nội bộ + database · ③ doanh nghiệp nhỏ (chat / Excel) · ④ nhiều hệ, ID / thời gian / thẩm quyền lệch nhau |
| **Ngành** | xây dựng (dầu, vật tư) → bán hàng / CSKH (CRM) → ngành thứ ba khác hẳn (thể thao realtime, hoặc tài chính) |
| **Kênh** | Zalo · email · web / app · API · tệp (Excel, PDF) |
| **Tích hợp** | tệp · database · REST API · webhook · MCP |
| **Tách bạch bắt buộc** | **tenant ≠ ngành ≠ tích hợp ≠ kênh**. Hiện cả bốn bị gộp trong một pack (`core/tenant/pack.go`, `tenants/construction/pack/sources.yaml`) [CODE] |

---

## 3. Bảy mặt phẳng (cũng là 7 đường làm việc song song)

| Mặt phẳng | Gồm | Hiện có (`d6afad7`) |
|---|---|---|
| **P1 Dữ liệu và sự thật** | thu, chuẩn hoá, thực thể, nghĩa, thời gian, thẩm quyền, đối soát, cách ly, đính chính | ● Fuel: 6 định dạng; 2.702 quan sát, 37 cách ly [ĐO]. Còn lỗi im lặng (§5 L1) |
| **P2 Tri thức** | tài liệu, RAG, trích dẫn, ACL tài liệu | ◐ chỉ mục từ khoá trong bộ nhớ; ACL chỉ theo vai [CODE] |
| **P3 Quyết định** | hiểu ý định, định tuyến, policy, hỏi lại | ● chỉ có luật từ khoá; **S4 là sàn** của hệ [REPO ADR-0010] |
| **P4 Hành động** | command, tool, workflow, event, chống trùng, đọc lại | ○ **0 dòng**. `EntryCommand`/`EntryEvent` khai nhưng không dùng; route TOOL / WORKFLOW / AGENT trả lỗi `ask.go:187-190` [CODE] |
| **P5 Kiểm soát** | tenant, danh tính, quyền, kiểm đầu ra, audit, vết, chi phí | ● một phần: tenant lấy từ danh tính; RLS FORCE; từ chối vai vượt RLS (`cc84b83`); kiểm số. **Chưa có:** xác thực, lưu bền bản ghi quyết định, chi phí |
| **P6 Chứng minh** | bộ ca, bộ giữ lại, baseline, bộ chấm, đột biến, soát độc lập | ● **mạnh nhất hiện có**: H1, H3 đã chạy; H4 đã khoá; đột biến; soát độc lập [REPO] |
| **P7 Vận hành** | HTTP API, CI có PG, triển khai, quản trị, SLA, sao lưu, scale | ○ gần như trống. Chỉ có `deploy/docker-compose.yml` cho PG dev; không có `cmd/`, không có CI [CODE] |

---

## 4. Hai cổng cho mọi bước

### 4.1 Cổng QuânIT: cổng vào, áp cho từng **việc nghiệp vụ**

Lấy từ transcript video QuânIT (trích nguyên văn, có thể kiểm bằng `grep -F`).

| # | Câu | Lời gốc | Qua cổng khi |
|---|---|---|---|
| 1 | Giá trị thật là gì? | "giải quyết được cái vấn đề gì… đem lại giá trị gì" | ghi được người dùng, việc, giá trị đo được |
| 2 | Việc có critical không? | "ứng dụng vào doanh nghiệp á là nó sẽ rất là critical… liên quan tới tiền… vận hành" | ghi rõ mức hậu quả khi sai |
| 3 | Dữ liệu đã chuẩn chưa? | "bắt anh chuẩn hóa cái phần mềm nội bộ của anh trước" | có đánh giá dữ liệu; đã cân nhắc chuẩn hoá đầu vào (form / bot) **trước khi** đọc chữ tự do |
| 4 | Đã phân loại dữ liệu chưa? | "phải phân loại được categorize… mỗi một dạng thì mày sẽ xử lý như thế nào" | mỗi loại nguồn có một cách xử lý |
| 5 | Phần nào không cần AI? | "có khi một số cái chẳng cần ai" | ghi phần tất định, phần cần model |
| 6 | Cần gom ngữ cảnh gì? | "tao cần A B C D E thông tin… gom hết… mới quăng qua cho cái con model" | liệt kê nguồn ngữ cảnh và quyền đọc |
| 7 | Thiếu hay mâu thuẫn thì hỏi lại thế nào? | "mâu thuẫn thì sao… thiếu khóa… clarification question" | có luật hỏi lại / từ chối / chuyển người |
| 8 | Ai cam kết độ chính xác? | "chính xác 100% tại vì cái đó là critical"; SLA; "ai là người chịu trách nhiệm" | có người nghiệp vụ ký mức độ chính xác |
| 9 | Đồ có sẵn đã giải được chưa? | "phải biết… để có thể nói chuyện với khách hàng một cái cách… intelligently" | đã so ít nhất một giải pháp có sẵn |

### 4.2 Cổng gốc (giữ từ roadmap 20/10)

1. Phase tiên quyết đã đạt.
2. Có **lỗi đo được** hoặc **ca nghiệp vụ** đòi năng lực mới.
3. Trả lời được **"ai sở hữu?"**: trạng thái, nguồn sự thật, ai ghi, ai đọc, vòng đời, hỏng thế nào, phục hồi thế nào.
4. Cổng ra là **nhị phân**, có ca âm chứng minh cổng đỏ được.

### 4.3 Luật dừng

Luật này thay luật "WIN thất bại thì không đi tiếp roadmap" của bản 20/10 và luật C12 của V3. Hai luật cũ mâu thuẫn nhau. [ĐỀ XUẤT: chờ **D-RM-04**]

- Một phase **FAIL** thì không mở phase **dựa trực tiếp vào lời khẳng định** của nó.
- Các lát của Giai đoạn II **không** dựa vào kết quả W-G7, nên không bị W-G7 chặn.
  - **Chưa có hiệu lực:** dòng ADR-0001 ở `CHO` §0 vẫn cấm mở COMMAND / WORKFLOW / EVENT trước khi đóng G7.
  - Cho tới khi có D-RM-04, Giai đoạn II **chỉ làm phần giấy**.
- **Mọi tuyên bố "hơn RAG / đáng giá"** phải chờ W-G7 (trên H5) hoặc chờ phép đo giá trị của phase tương ứng.
- Không chạy lại một bộ giữ lại đã dùng để lấy kết luận mới (ADR-0006).

---

## 5. Tất cả các luồng

Mỗi luồng ghi theo khuôn: **Đích** → **Hiện có** (kèm bằng chứng) → **Thiếu** → **Phase**.

### L0. Luồng đưa AI vào một việc của doanh nghiệp

```text
việc nghiệp vụ
  → [CỔNG QUÂNIT, 9 câu]
  → hợp đồng tối thiểu theo việc (chỉ nghĩa của các trường mà việc dùng)
  → chọn chế độ A / B
  → xây lát mỏng
  → [cổng chứng minh của phase]
  → việc "ra"
  → chủ dự án debug để hiểu
  → làm sâu
```

- **Hiện có:** mới áp ngầm cho Fuel. Fuel đạt câu 4, 5, 6 và trượt câu 1, 3, 8, 9 (§7).
- **Thiếu:** áp tường minh cho mọi việc mới.

### L1. Dữ liệu (chế độ B; chế độ A thì đọc qua API của app)

```text
nguồn (tệp / DB / API / kênh)
  → đọc theo loại
  → chuẩn hoá biểu diễn ("4k" → 4.000 L)
  → thực thể ("máy 03" → EX-03)
  → nghĩa (cấp / tiêu thụ / báo / hoá đơn / KẾ HOẠCH / ĐÍNH CHÍNH)
  → thời gian (xảy ra · gửi · hiệu lực)
  → kiểm miền
  → cách ly, có người nhận
  → lưu quan sát + nguồn gốc
  → đối soát theo thẩm quyền
  → sự thật đã đối soát
```

- **Hiện có:**
  - 6 định dạng [ĐO];
  - nghĩa gán **theo loại nguồn**, mọi tin chat là "đội báo" (`sources.yaml`) [CODE];
  - người gửi chỉ dùng làm khoá kế thừa tên máy (`ingest.go:256-285`) [CODE];
  - thẩm quyền theo chỉ số, có dung sai; ngang hạng thì chuyển người duyệt (`sourceoftruth/query.go`) [CODE].
- **Thiếu (lỗi im lặng đã đo, `docs/ROADMAP.md` §0.4):**
  - `≈` mất cờ ước lượng;
  - tin đính chính thành số mới;
  - tin kế hoạch ("mai cần 4k") thành quan sát;
  - tin gửi lại thành `SYSTEM_FAILURE`;
  - "máy 03" thành 3 L ở 6 ca H1 [REPO];
  - nhóm nghìn viết cách bằng dấu cách (D-OP-9);
  - cách ly chỉ nằm trong RAM;
  - nạp một lần, thay toàn bộ; `aggregate` cộng mọi fact cùng nguồn, cùng ngày (`query.go:199-221`), nên **có thể cộng đôi** [CODE, chưa repro].
- **Phase:** 0a, 3.

### L2. Hỏi (QUERY), S0–S10

```text
S0 nhận → S1 tenant lấy từ danh tính → S2 chuẩn hoá → S3 thực thể → S4 ý định
→ S5 định tuyến + kiểm quyền (capability, phạm vi thực thể)
→ S6 thực thi (SQL / tính / so sánh) → S7 ngữ cảnh
→ S8 sinh câu (mẫu; model chỉ dùng ở RAG / HYBRID)
→ S9 kiểm → S10 trả kết quả + bản ghi quyết định
```

- **Hiện có** [ĐO, 16 câu thăm dò trên dữ liệu seed 42, model giả]:
  - SQL: "được cấp" → 3.850 L, kèm "ERP 3.900 L";
  - CALC: chênh lệch → 68 L;
  - RAG, HYBRID: 1 lời gọi model;
  - hỏi lại: "máy 07" → EX-07 / RL-07;
  - từ chối ngoài phạm vi;
  - chặn theo vai (accountant hỏi SOP) và theo phạm vi (eng-b, PRJ-B);
  - **0 lời gọi model** trên mọi route số liệu.
- **Thiếu:**
  - S4 chỉ dùng luật. Tầng sai đầu tiên của 13/15 ca đỏ H1 và 20/25 ca đỏ H3 [REPO ADR-0010].
  - S9 không bắt được lỗi chọn sai nghĩa: "Máy 03 hôm qua **hết** bao nhiêu dầu?" trả kho xuất 3.850 L, RESOLVED [ĐO].
  - Mẫu câu hứa "đã chuyển người phụ trách" mà không chuyển (`templates.yaml:16-17`) [CODE].
  - HYBRID báo sai lý do mà Pass vẫn true [REPO].
  - Không có xác thực, không có HTTP.
- **Phase:** 0b, 6 (model hiểu ý định).

### L3. Làm (COMMAND)

```text
yêu cầu / đề xuất → kiểm schema → tenant → quyền (chế độ A: app quyết, bằng token người dùng)
→ policy → luật nghiệp vụ → khoá chống trùng → thực thi → ĐỌC LẠI xác minh → audit
→ báo kết quả: ĐÃ LÀM / CHƯA BIẾT / THẤT BẠI
```

- **Hiện có:** 0 dòng [CODE].
- **Phase:** 1 (tạo phiếu kiểm tra chênh lệch, chế độ B) · 4b (tạo task trên CRM, chế độ A).

### L4. Quy trình (WORKFLOW)

```text
phát hiện chênh lệch → mở hồ sơ → điều tra → tổ trưởng duyệt → kho xác minh
→ đối soát → duyệt / từ chối → đóng
(máy trạng thái · người trong vòng · hạn chờ · phục hồi sau khởi động lại · trạng thái dài hạn)
```

- **Hiện có:** 0.
- **Phase:** 2.
- **Trước khi viết mã:** so với đồ có sẵn (Temporal, n8n…), theo câu 9 của cổng QuânIT. Chỉ tự viết phần quyền và bằng chứng.

### L5. Sự kiện (EVENT)

```text
sự kiện (tin mới · phiếu mới · vượt ngưỡng · đính chính · sự kiện đến muộn)
→ nạp sự kiện → khử trùng → thứ tự → luật → kích hoạt COMMAND / WORKFLOW
→ outbox → idempotency
```

- **Hiện có:** 0 (`EntryEvent` chưa dùng) [CODE].
- **Phase:** 3. Đây là chỗ **sửa tận gốc** các lỗi đính chính / kế hoạch / sự kiện muộn của L1.

### L6. Tri thức (RAG)

```text
tài liệu → đọc / OCR → chia đoạn → chỉ mục → truy xuất (lọc ACL) → (rerank)
→ ngữ cảnh → model → trích dẫn → kiểm
```

- **Hiện có:** chỉ mục từ khoá trong bộ nhớ, mỗi dòng một đoạn; ACL theo vai (`pgstore.go:163`, `rag.go:134`) [CODE].
- **Thiếu:** ACL theo phạm vi thực thể; phiên bản tài liệu; bảng / OCR.
- **Phase:** 6 (khi lỗi đo được đòi).

### L7. Kiểm soát

- tenant lấy từ danh tính;
- **xác thực (chưa có)**;
- quyền đóng mặc định (`router.go:127`);
- RLS FORCE + từ chối vai vượt RLS;
- kiểm đầu ra;
- audit;
- **lưu bền bản ghi quyết định (chưa có)**;
- **chi phí theo lượt (chưa có)**.

**Phase:** 0b (phần đã có) · 5.

### L8. Chứng minh

```text
bộ ca phát triển
→ bộ giữ lại (soạn độc lập, khoá băm, chạy MỘT lần)
→ baseline (B1, B2+, B2″; THÊM "app + model + kiểm cơ bản" và "đồ có sẵn")
→ bộ chấm (hiệu chỉnh bằng người chấm mù)
→ cổng nhị phân → đột biến → soát độc lập
```

- **Hiện có:** H1 89/104 · H3 109/134 · H4 đã khoá (119 ca) · danh mục đột biến · soát độc lập nhiều lượt [REPO].
- **Thiếu:**
  - bộ chấm v2 chưa khoá (`scorerFrozen = false`, `g7run/main.go:67`);
  - người chấm mù;
  - baseline mức app.
- **Phase:** áp cho mọi phase.

### L9. Vận hành

HTTP API · **CI có Postgres** (hiện test PG bị SKIP khi thiếu DSN [ĐO]) · triển khai · bí mật · hạn mức · SLA · sao lưu · scale.

- **Hiện có:** chỉ docker-compose PG cho dev.
- **Phase:** 5, 9.

### L10. Điều phối người–agent

```text
chủ dự án quyết ở CHO-QUYET-DINH.md
→ BUILD xây theo docs/PROMPT-BUILD-EACL.md, commit + push, tệp tiến độ sống
→ phiên điều phối / debug đọc qua git, gửi lỗi qua docs/debug-handoff/
```

**Hiện có:** đang chạy.

---

## 6. Giai đoạn và phase

Giữ **số phase của roadmap gốc 20/10**, cột cuối quy đổi sang V3. Trạng thái tính tại `d6afad7`.

| Giai đoạn | Phase | Việc / lời khẳng định | Luồng | Chế độ | Cổng ra | Trạng thái | V3 |
|---|---|---|---|---|---|---|---|
| **I. Đóng nền** | **0b** Sự thật hỏi–đáp (Fuel) | hỏi số dầu đúng nghĩa, đúng nguồn, có bằng chứng | L1, L2 | B | chạy H4 **một lần**. Sau đó W-G7 trên H5 (cần chủ dự án duyệt chi tiền), hoặc đóng với kết luận **PARTIAL: chưa đo giá trị** | ● 0/7; H4 đã khoá | R1 |
| | **0a** Thu báo cáo ngày (việc gốc của QuânIT) | tin Zalo → trích → chuẩn hoá → kiểm → người xác nhận → báo cáo | L0, L1 | B | độ chính xác ≥ ngưỡng SLA trên bộ ca độc lập; có nhánh so sánh "đổi sang form / bot" | ○ mới | — |
| **II. Xương sống rộng** (chuỗi chênh lệch nhiên liệu, đủ 4 loại tương tác) | **1** Hành động có kiểm soát | tạo **đúng một** phiếu kiểm tra chênh lệch: có quyền, chống trùng, đọc lại, audit | L3 | B | ca âm: gửi đôi, thiếu quyền, timeout sau khi ghi | ○ | (R4) |
| | **2** Workflow | điều tra → duyệt → xác minh → đối soát → đóng, có người trong vòng | L4 | B | phục hồi sau khởi động lại; không chuyển trạng thái trái luật | ○ | bị V3 bỏ |
| | **3** Event | sự kiện nạp / vượt ngưỡng / **đính chính / kế hoạch / muộn** | L5, L1 | B | sửa các lỗi im lặng của L1, có ca âm | ○ | bị V3 bỏ |
| **III. Thực tế doanh nghiệp** | **4a** Tách tenant / ngành / tích hợp / kênh | cấu trúc lại pack và runtime | L1–L7 | A + B | 1 tenant chạy 2 ngành + 2 kênh | ○ | — |
| | **4b** Ngành thứ hai, chế độ A | **CRM**: tóm tắt khách + tạo task qua API của app (Job A/B của V3) | L2, L3, L6 | A | số dòng phải sửa trong core ≈ 0 (`git diff --stat` các thư mục chung); dùng app **không** viết riêng cho dự án | ○ | R2–R6 |
| | **4c** Bốn kiểu doanh nghiệp | kiểu ①–④ mỗi kiểu một việc, hợp đồng tối thiểu theo việc | L0 | A + B | ma trận 4 kiểu đều có kết quả đo | ○ | — |
| | **4d** Ngành thứ ba | khác hẳn (thể thao realtime hoặc tài chính) | tất cả | A / B | lõi vẫn ≈ 0 dòng phải sửa | ○ (thể thao mới có bộ ca) | R8–R9 |
| **IV. Nền tảng** | **5** Cứng hoá | HTTP API · xác thực thật · vòng đời tenant / pack · phiên bản cấu hình · migration · bí mật · hạn mức · ngân sách chi phí · **CI có PG** · lưu bền bản ghi quyết định · quản trị | L7, L9 | — | mọi cổng "0 vi phạm" có đối chứng dương | ○ | R11 |
| | **8** Tích hợp chuẩn hoá | connector Zalo / email / CRM / ERP · webhook · **MCP** (dùng chuẩn có sẵn) | L1, L3 | A | ≥ 2 bên dùng thật | ○ | R12 |
| **V. Trí tuệ nâng cao** | **6** AI nâng cao | **model hiểu ý định** (đã có bằng chứng đòi, xem dưới bảng) · đọc PDF / bảng / OCR · rerank · nhận dạng thực thể bằng ML · model local · thị giác máy (train riêng cho từng môi trường) · fine-tune | L2, L6 | — | mỗi năng lực đi đủ chuỗi: lỗi quan sát được → ca tái hiện → baseline → ứng viên → benchmark | ○ | R12 |
| | **7** Agent có biên | khi luồng cố định không quyết được; phải trả lời "Agent sở hữu gì?" | L3, L4 | — | ca chứng minh | ○ | R12 |
| **VI. Quy mô và tầm nhìn** | **9** Scale | tải thật · HA / DR · hàng đợi · Kafka · sharding: mở theo số đo | L9 | — | số đo cho thấy đường đồng bộ không đạt | ○ | R11 |
| | **10** Toàn cảnh | QUERY + COMMAND + WORKFLOW + EVENT trên ≥ 3 ngành, nhiều tenant, nhiều tích hợp | tất cả | A + B | các phase trước đạt | ○ | bị V3 bỏ |

**Phase 6, model hiểu ý định:** bằng chứng đòi năng lực này đã có (`CLAUDE.md` §19). S4 là tầng sai đầu tiên ở 13/15 ca đỏ H1 và 20/25 ca đỏ H3. Theo ADR-0010, phải sửa bằng luật bất biến trước; chạy H4 xong mà S4 vẫn là sàn thì mở so sánh rule / model / hybrid (cần khoá model và câu cho phép chi tiền).

**V3 → phase:** R0 → định nghĩa (§1, D-RM-01) · R1 → 0b · R2–R6 → 4b · R7 → sau 4b / 4d, chỉ trích lõi khi 2 ngành cùng dùng · R8–R9 → 4d · R10 → bỏ (nền tảng là đích; phép đo "lõi ≈ 0 dòng phải sửa" ở 4b / 4d thay cho cổng framework) · R11 → 5, 9 · R12 → 6, 7, 8.

---

## 7. Phase 0b (Fuel) chấm theo cổng QuânIT

| # | Câu | Fuel | Bằng chứng |
|---|---|---|---|
| 1 | Giá trị thật? | ✗ | tenant hư cấu, không có khách [REPO `docs/tenants/construction/01-business-map.md:3`] |
| 2 | Critical? | ✓ | số tiền, số vận hành |
| 3 | Dữ liệu chuẩn chưa? | ✗ | đọc chữ Zalo tự do bằng luật; không có cổng kiểm mức sẵn sàng; còn lỗi im lặng L1 |
| 4 | Phân loại dữ liệu? | ✓ | 6 định dạng, mỗi loại một bộ đọc [ĐO] |
| 5 | Phần nào không cần AI? | ✓ | 0 lời gọi model trên route số liệu [ĐO] |
| 6 | Gom ngữ cảnh? | ✓ | S6–S7, kiểm quyền trước khi đọc |
| 7 | Hỏi lại? | ◐ | có hỏi lại; nhưng S4 từ chối sai câu thuộc phạm vi (ADR-0010) |
| 8 | Ai cam kết độ chính xác? | ✗ | chưa có người nghiệp vụ thật (D-G7-12, D-G7-16 còn treo) |
| 9 | Đồ có sẵn? | ✗ | chưa so lần nào |

⇒ **Fuel đạt các câu kỹ thuật và trượt các câu phía doanh nghiệp.** Đó là lý do có Phase 0a và phụ thuộc số 1 (§9).

---

## 8. Bài học đã đo (20–21/10, 135 commit) và chỗ cài vào roadmap

| Bài học | Nằm ở |
|---|---|
| Hiểu câu hỏi bằng luật là sàn của hệ (H1 13/15, H3 20/25) | Phase 6, mở sớm sau H4 |
| Lỗi im lặng ở tầng nạp | Phase 0a, 3 |
| Bộ chấm cũng sai: v1 lệch 133/170 đòn (109 câu sai mà ĐẠT + 24 câu đúng mà TRƯỢT) [REPO ADR-0007] | L8: hiệu chỉnh bằng người chấm mù |
| Bộ giữ lại khoá băm, chạy một lần là có hiệu quả | L8, mọi phase |
| Canary vô hiệu; test PG bị SKIP; không có CI | Phase 5 (đưa CI có PG lên sớm nếu được) |
| Tenant bị đồng nhất với ngành ngay từ bản vẽ đầu tiên | Phase 4a |
| Một nửa số commit là xây bộ máy đo; sản phẩm chưa ra khỏi Phase 0 | §0 câu 4: đi rộng bằng lát mỏng |
| Lõi chung còn mang giả định Fuel (`router.go:71` viết cứng `"project"`; thực thể Projects / Equipment; sự thật chỉ là số; từ của miền xây dựng trong `qualifier.go`) | Phase 4a, 4b |

---

## 9. Phụ thuộc số 1: doanh nghiệp hoặc người nghiệp vụ thật (D-RM-03, còn mở)

Dữ liệu giả lập không chứng minh được tính thực tế ở tầng doanh nghiệp. Cần ít nhất một doanh nghiệp, hoặc người nghiệp vụ thật, để:
1. cung cấp dữ liệu, hoặc duyệt dữ liệu giả lập;
2. định nghĩa nghĩa của các con số;
3. ký mức độ chính xác chấp nhận được (cổng QuânIT câu 8).

Chưa có thì mọi phase vẫn chạy được trên dữ liệu giả lập, nhưng **kết luận phải ghi rõ "trên dữ liệu giả lập, chưa có người nghiệp vụ duyệt"**.

---

## 10. Hiện trạng đo được (mốc `d6afad7`)

| Mục | Số | Nguồn |
|---|---|---|
| Commit | 135, từ 20/10 10:43 tới 21/10 16:44 | [ĐO `git rev-list`] |
| Mã Go không tính test / test / Markdown | 15.772 / 12.238 / 18.224 dòng | [ĐO `git grep -c`] |
| Test không có PG | 446 đạt / 0 lỗi / 5 bỏ qua (tại `acd09c8`) | [ĐO] |
| Test có PG | 453 / 0 / 0 | [REPO TIEN-DO-BUILD] |
| H1 / H3 | 89/104 · 109/134 (mốc so sánh 113/134) | [REPO STATUS] |
| Cổng Fuel đạt trọn trên dữ liệu độc lập | **0/7**. W-G6 ở mức PARTIAL; W-G7 bị chặn | [REPO STATUS] |
| Luồng có mã | chỉ L1, L2 (+ một phần L6, L7, L8) | [CODE] |

---

## 11. Ngay bây giờ

Thứ tự chi tiết ở `docs/PROMPT-BUILD-EACL.md` §4:
1. **Đổi tên lại** theo D-RM-02 (link git giữ nguyên).
2. **Đóng Phase 0b:** R4-2…R4-4 của cổng phán quyết → vòng S4 theo ADR-0010 → chạy H4 một lần.
3. Phép đo trước W-G7: canary hai tenant, cờ ước lượng.
4. **Phase 0a trên giấy:** hợp đồng việc "thu báo cáo ngày" qua cổng QuânIT, kèm bộ ca SLA giả lập.
5. **Giai đoạn II, Phase 1:** ADR + cổng QuânIT cho việc "tạo phiếu kiểm tra chênh lệch" (giấy). **Mã chờ D-RM-04.**

---

## 12. Câu còn mở cho chủ dự án

| Mã | Câu |
|---|---|
| **D-RM-03** | Doanh nghiệp hoặc người nghiệp vụ thật nào cho §9? |
| **D-RM-04** | Có mở Giai đoạn II **trước khi đóng G7** không (luật dừng §4.3)? Khuyến nghị: mở sau khi đổi tên xong và chạy H4 |
| D-G7-8, D-G7-18 + câu cho phép chi tiền | W-G7 có chạy không (trần 15 USD đã duyệt có điều kiện) |
| Thứ tự trong Giai đoạn III | Làm 4b (CRM) trước 4a, hay sau |

---

## 13. Quan hệ với các tài liệu cũ

| Tài liệu | Từ nay |
|---|---|
| `docs/05-master-roadmap.md` | **gốc của phạm vi** roadmap này; bộ luật §2 của nó vẫn giữ giá trị |
| `docs/04-core-mvp.md`, `docs/g7/HOP-DONG-G7.md` | vẫn là hợp đồng của Phase 0b |
| `docs/v3/*` | tham khảo. Thứ tự R0–R12 không còn là roadmap hiện hành; Job A/B dùng ở Phase 4b; danh sách rủi ro (`08`) vẫn dùng |
| `docs/ROADMAP.md` | tra cứu: §0.4 lỗi im lặng, §1 dòng dõi, §3 bản đồ, các phụ lục. Không lập kế hoạch theo §4 / §5 của nó |
| `docs/PROMPT-AUDIT-ROADMAP.md` | các khẳng định K về sự thật trong mã vẫn có giá trị; phần bàn §4.7 hết hiệu lực |
| `docs/requirements/catalog.yaml` | lịch sử (422 yêu cầu, chỉ 38 có lý do trực tiếp từ video) |
