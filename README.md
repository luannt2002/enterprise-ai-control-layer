# EAIS — Enterprise AI Systems

**Nền tảng multi-tenant biến dữ liệu, tri thức, quy trình và công cụ của doanh nghiệp thành năng lực AI có kiểm soát, kiểm thử được và quan sát được.**
- **Dùng lại** hệ thống doanh nghiệp đã có khi hệ đó đang giữ sự thật (chế độ A, tích hợp).
- **Tự lo** phần doanh nghiệp còn thiếu: thu dữ liệu, chuẩn hoá, đối soát (chế độ B).
- Core **không gắn** vào một ngành.

EAIS is a multi-tenant Enterprise AI Systems Platform for turning domain-specific business data, knowledge, workflows, and tools into governed, testable AI capabilities without coupling the core platform to a single industry.

Phần mềm tất định giữ quyền quyết định về danh tính, tenant, quyền hạn, sự thật nghiệp vụ, trạng thái, giao dịch, chính sách, kiểm và thực thi. AI chỉ làm phần bất định: diễn giải, phân loại, trích xuất, tổng hợp, đề xuất. Model là một năng lực bên trong hệ, không phải nguồn sự thật.

```text
Business Reality → Trusted Context → Useful Intelligence → Controlled Action → Verified Outcome → Measured Value
```

> **Tệp này là bản đồ một-tệp của toàn dự án** (24/10/2026): sứ mệnh, 7 mặt phẳng, mọi luồng L0–L10, MVP, roadmap Phase 0–10, toàn bộ
> luồng RAG, cách chạy và giới hạn đã biết. Chi tiết và bằng chứng nằm ở các tệp được trỏ tới; khi mâu thuẫn, thứ bậc là
> `docs/CHO-QUYET-DINH.md` > ADR > hợp đồng (`docs/04-core-mvp.md`, `docs/g7/HOP-DONG-G7.md`) > `CLAUDE.md` > roadmap > README này.
> Nhãn: **[ĐO]** đã chạy lệnh · **[REPO]** tài liệu repo ghi · **[CODE]** đọc thấy trong mã · **[ĐỀ XUẤT]** chưa quyết.

---

## 0. Đọc nhanh — trạng thái 24/10/2026

| Mảng | Trạng thái | Nguồn |
|---|---|---|
| **MVP** (5 khối M1–M5, D-MVP-00) | **CHƯA RA.** M1 phần chạy đạt, vế đọc vết 78,9% (< 90%); M2 bộ ca 0a đã băm, mã đang soát; M3 chờ D-AI-05 (vòng sửa S4); M4 bộ giữ lại P1 153/157; M5 các dòng S2–S4, L1 chưa đạt | `docs/STATUS.md` |
| **Fuel WIN** (7 cổng G1–G7) | **0/7** trên bộ giữ lại độc lập; H1 91/104, H3 112/134, H4 103/119 (winreport 23/10) | `docs/04-core-mvp.md` §6, `docs/STATUS.md` |
| **RAG** (luồng L6) | Đã gộp vào main 24/10 (D-RAG-01). Ngoài phạm vi MVP (D-AI-03 = a). App chạy từ khoá + model trích; đường lai (từ khoá + vector + xếp lại) và model thật chỉ ở bộ đo | §5 dưới; `docs/ROADMAP-RAG.md` |
| **Nút thắt số 1** | Hiểu ý định S4 chỉ bằng luật: tầng sai đầu tiên ở 13/15 ca đỏ H1, 20/25 H3; với câu hỏi quy trình chỉ 7/92 câu tới được RAG | `docs/adr/` ADR-0010; `docs/ROADMAP-RAG.md` §10.2 |
| **Phụ thuộc số 1** | Chưa có doanh nghiệp / người nghiệp vụ thật (D-RM-03) ⇒ mọi kết luận "trên dữ liệu giả lập" | `docs/ROADMAP-DAY-DU.md` §9 |
| **Câu đang chờ chủ dự án** | D-AI-04, D-AI-05, Q-P1-9…15, Q-0A-13…16, D-RM-03; phía RAG: S4 phương án (b), mở model thật trong app, nhà cung cấp embedding / rerank | `docs/CHO-QUYET-DINH.md` |

Repo: `github.com/luannt2002/enterprise-ai-control-layer` (cũng là đường dẫn module Go). Mọi dữ liệu trong repo là **giả lập**; repo công
khai không chứa dữ liệu thật của tenant nào.

---

## 1. EAIS là gì, không là gì

**Mười nguyên lý** (chi tiết `CLAUDE.md` §10): P1 model ≠ hệ thống · P2 vỏ tất định quanh lõi xác suất · P3 đầu ra model là đề xuất ·
P4 sự thật từ nguồn có thẩm quyền · P5 mơ hồ phải lộ ra · P6 dùng chung nền ≠ dùng chung tri thức · P7 phức tạp phải có bằng chứng ·
P8 mọi kết quả truy vết được · P9 benchmark ≠ chất lượng nghiệp vụ · P10 debug tầng trước khi đổi model.

**Hai chế độ sở hữu:**

| Chế độ | Khi nào | Ai giữ sự thật, quyền, giao dịch | Ví dụ |
|---|---|---|---|
| **A. Tích hợp** | doanh nghiệp đã có CRM / ERP / app có API | **app của khách**; EAIS đọc / ghi qua API bằng token người dùng | CRM: tóm tắt khách, tạo task |
| **B. Tự lo** | doanh nghiệp chỉ có Zalo / Excel / giấy | **EAIS**: thu, chuẩn hoá, đối soát, lưu | báo cáo dầu qua Zalo (Fuel) |

**Không làm:** thay CRM / ERP khi khách đã có · framework chatbot / RAG / agent bán riêng · ontology chung mọi ngành · đọc DB của khách
vòng qua quyền của app · thêm năng lực khi chưa qua hai cổng (§4).

**Bảy mặt phẳng** (cũng là 7 đường làm việc song song):

| Mặt phẳng | Gồm | Hiện có (24/10) |
|---|---|---|
| **P1 Dữ liệu và sự thật** | thu, chuẩn hoá, thực thể, nghĩa, thời gian, thẩm quyền, đối soát, cách ly, đính chính | ● Fuel: 6 định dạng; 2.702 quan sát, 37 cách ly; sổ nạp `load_records` (phiên bản dữ liệu đã commit). Còn lỗi im lặng (L1) |
| **P2 Tri thức** | tài liệu, RAG, trích dẫn, ACL tài liệu | ● tài liệu có phiên bản / hiệu lực / phạm vi trên PG (RLS); lọc quyền + phiên bản TRƯỚC khi chấm; cắt theo cấu trúc; trích dẫn tất định; vết S6 (§5) |
| **P3 Quyết định** | hiểu ý định, định tuyến, policy, hỏi lại | ● chỉ luật từ khoá; **S4 là sàn** của hệ |
| **P4 Hành động** | command, tool, workflow, event, chống trùng, đọc lại | ◐ lõi lệnh `execution/commands` + route `POST /v1/commands/<tool>` (Phase 1, M4); chưa có workflow / event |
| **P5 Kiểm soát** | tenant, danh tính, quyền, kiểm đầu ra, audit, vết, chi phí | ● tenant từ danh tính; RLS FORCE; từ chối vai vượt RLS; kiểm số; vết bền mọi lượt. **Chưa:** xác thực môi trường thật, chi phí theo lượt |
| **P6 Chứng minh** | bộ ca, bộ giữ lại, baseline, bộ chấm, đột biến, soát độc lập | ● **mạnh nhất**: H1, H3, H4, P1, 0a đóng băng bằng băm; đột biến; soát độc lập |
| **P7 Vận hành** | HTTP API, CI có PG, triển khai, quản trị, SLA | ◐ `cmd/eais` (HTTP + CLI, chỉ `-dev`); một lệnh chạy thử; CI GitHub có PG |

---

## 2. Tất cả các luồng (L0–L10)

Khuôn mỗi luồng: **đích → hiện có → thiếu → phase**. Nguồn: `docs/ROADMAP-DAY-DU.md` §5.

**L0. Đưa AI vào một việc:** việc nghiệp vụ → cổng QuânIT (9 câu) → hợp đồng tối thiểu theo việc → chọn chế độ A / B → lát mỏng →
cổng chứng minh → việc "ra" → chủ dự án debug → làm sâu. *Hiện có:* áp cho Fuel, Phase 1 (`docs/viec/1-hanh-dong/`), tra quy trình
(`docs/viec/2-tra-quy-trinh/`). *Thiếu:* áp tường minh cho mọi việc mới.

**L1. Dữ liệu** (chế độ B): nguồn → đọc theo loại → chuẩn hoá biểu diễn ("4k" → 4.000 L) → thực thể ("máy 03" → EX-03) → nghĩa (cấp /
tiêu thụ / báo / hoá đơn / KẾ HOẠCH / ĐÍNH CHÍNH) → thời gian → kiểm miền → cách ly có người nhận → lưu quan sát + nguồn gốc → đối soát
theo thẩm quyền. *Hiện có:* 6 định dạng; thẩm quyền theo chỉ số có dung sai; sổ nạp. *Thiếu (lỗi im lặng đã đo):* tin đính chính thành
số mới; tin kế hoạch thành quan sát; tin gửi lại thành lỗi; nhóm nghìn viết cách; cách ly chỉ trong RAM; có thể cộng đôi cùng nguồn
cùng ngày. *Phase:* 0a (M2), 3.

**L2. Hỏi (QUERY), S0–S10:**

```text
S0 nhận → S1 tenant từ danh tính → S2 chuẩn hoá → S3 thực thể → S4 ý định → S5 định tuyến + kiểm quyền (capability, phạm vi thực thể)
→ S6 thực thi (SQL / tính / so sánh / truy hồi tri thức) → S7 ngữ cảnh → S8 sinh câu (mẫu; model chỉ ở RAG / HYBRID)
→ S9 kiểm → S10 trả kết quả + bản ghi quyết định + vết từng bước
```

*Hiện có:* SQL (3.850 L kèm "ERP 3.900 L"), CALC (chênh lệch 68 L), RAG / HYBRID, hỏi lại khi mơ hồ, từ chối ngoài phạm vi, chặn theo vai
và phạm vi, **0 lời gọi model trên route số liệu**; cổng phán quyết câu xác nhận fail-closed (ADR-0009). *Thiếu:* S4 chỉ luật (sàn của
hệ); S9 chưa bắt lỗi chọn sai nghĩa ("hết bao nhiêu dầu" trả kho xuất). *Phase:* 0b (M3), 6 (model hiểu ý định — xem §5.7).

**L3. Làm (COMMAND):** yêu cầu → schema → tenant → quyền → policy → luật → khoá chống trùng → thực thi → ĐỌC LẠI → audit → ĐÃ LÀM /
CHƯA BIẾT / THẤT BẠI. *Hiện có:* Phase 1 "tạo đúng một phiếu kiểm tra chênh lệch" (ADR-0011, K1–K4 đã gộp, bộ giữ lại P1 153/157).
*Phase:* 1 (M4), 4b (CRM, chế độ A).

**L4. Quy trình (WORKFLOW):** phát hiện → mở hồ sơ → điều tra → duyệt → xác minh → đối soát → đóng (máy trạng thái, người trong vòng,
phục hồi sau khởi động lại). *Hiện có:* 0. *Phase:* 2; so với đồ có sẵn trước khi viết mã.

**L5. Sự kiện (EVENT):** sự kiện (tin mới / phiếu mới / vượt ngưỡng / đính chính / đến muộn) → khử trùng → thứ tự → luật → kích hoạt
COMMAND / WORKFLOW → outbox → idempotency. *Hiện có:* 0. *Phase:* 3 — chỗ sửa tận gốc lỗi đính chính / kế hoạch của L1.

**L6. Tri thức (RAG):** xem **§5** (toàn bộ luồng, hiện trạng, số đo, việc tiếp).

**L7. Kiểm soát:** tenant từ danh tính · quyền đóng mặc định · RLS FORCE + từ chối vai vượt RLS · kiểm đầu ra · audit · vết bền. *Thiếu:*
xác thực môi trường thật, chi phí theo lượt. *Phase:* 0b (đã có), 5.

**L8. Chứng minh:** bộ ca phát triển → bộ giữ lại (soạn độc lập, khoá băm, chạy MỘT lần, ADR-0006) → baseline → bộ chấm (hiệu chỉnh bằng
người chấm mù) → cổng nhị phân → đột biến → soát độc lập. *Hiện có:* H1, H3, H4, P1, 0a; danh mục đột biến; soát độc lập nhiều vòng.
*Thiếu:* bộ chấm v2 chưa khoá, người chấm mù, baseline mức app.

**L9. Vận hành:** HTTP API · CI có PG · triển khai · bí mật · hạn mức · SLA · sao lưu · scale. *Hiện có:* `cmd/eais`, một lệnh chạy thử,
CI có PG cấm SKIP. *Phase:* 5, 9.

**L10. Điều phối người–agent:** chủ dự án quyết ở `docs/CHO-QUYET-DINH.md` → BUILD xây theo `docs/PROMPT-BUILD-EAIS.md`, commit + push,
tệp tiến độ sống → phiên điều phối / debug đọc qua git, gửi lỗi qua `docs/debug-handoff/`.

---

## 3. MVP (D-MVP-00): lát dọc đầu tiên chạy được ở tầng doanh nghiệp

Một doanh nghiệp xây dựng ở chế độ B, một chuỗi việc dầu từ đầu tới cuối, **không cần model trả tiền**. Nguồn duy nhất của cổng ra từng
khối: `docs/ROADMAP-DAY-DU.md` §5b.

| Khối | Là gì | Cổng ra | Trạng thái (23–24/10) |
|---|---|---|---|
| **M1** App chạy được + vết từng bước | `cmd/eais` (ask / ingest / trace / healthz / commands); một lệnh từ clone sạch; CI có PG; vết bền mọi lượt | smoke E2E xanh từ clone sạch; người chỉ đọc vết chỉ đúng tầng sai đầu tiên ≥ 90% ca đỏ | phần chạy ĐẠT (smoke 21/21); vế đọc vết 30/38 = 78,9% — CHƯA |
| **M2** Thu báo cáo ngày đúng | tin Zalo / tệp → bản ghi có cờ (≈, đính chính, kế hoạch); cách ly có lý do; không cộng đôi | độ chính xác ≥ SLA trên bộ độc lập; dòng L1, S2 của CỔNG TRƯỚC AI đạt | giấy khoá; bộ 0a (249 ca) đã băm; mã đang soát |
| **M3** Hỏi số liệu đúng nghĩa, đúng nguồn | đóng Phase 0b: cổng phán quyết khoá, vòng S4, H4 một lần | báo cáo H4; dòng S3–S7, S9 đạt (S7 ngoài MVP — D-AI-03 = a) | H4 104/119; dòng S4 10/119 > 5% — chờ D-AI-05 |
| **M4** Chênh lệch → đúng một phiếu | Phase 1: quyền, chống trùng, đọc lại, audit; ca âm | ca âm đỏ đúng; đúng một phiếu mỗi chênh lệch | bộ P1 153/157 (4 đỏ cùng lỗi "≈", đã sửa); chờ Q-P1-15 |
| **M5** CỔNG TRƯỚC AI | mọi dòng có lệnh chứng minh (`docs/bao-cao/CONG-TRUOC-AI.md`) | ghi "CỔNG TRƯỚC AI ĐẠT" | S2–S4, L1 chưa đạt; S5, S6 chưa có bộ độc lập |

**MVP RA** = M1–M5 đạt ⇒ chủ dự án debug toàn chuỗi qua `eais trace`. **Fuel WIN** (hợp đồng Phase 0b, `docs/04-core-mvp.md` §6): G1 DATA ·
G2 IDENTITY · G3 TRUTH · G4 DECISION · G5 CONTROL · G6 PROOF · G7 VALUE (model thật, cần duyệt chi tiền) — WIN = 7/7 + scale smoke.

**Sau MVP mới làm:** W-G7 và mọi lượt chi tiền model; model hiểu ý định nếu S4 vẫn là sàn; Phase 2, 3, 4a–4d, 5–10 (phần giấy vẫn được làm
khi còn tài nguyên).

---

## 4. Roadmap Phase 0–10 (D-RM-01) và hai cổng

**Hai cổng cho mọi bước:** (1) **cổng QuânIT** — 9 câu phía doanh nghiệp: giá trị thật · critical · dữ liệu chuẩn chưa · phân loại · phần
nào không cần AI · gom ngữ cảnh gì · thiếu / mâu thuẫn thì hỏi lại thế nào · ai cam kết độ chính xác · đồ có sẵn giải được chưa;
(2) **cổng gốc** — phase tiên quyết đạt · có lỗi đo được hoặc ca nghiệp vụ đòi · trả lời "ai sở hữu?" · cổng ra nhị phân có ca âm.
**Luật dừng:** phase FAIL thì không mở phase dựa trực tiếp vào lời khẳng định của nó; không chạy lại bộ giữ lại đã dùng (ADR-0006).

| Giai đoạn | Phase | Việc / lời khẳng định | Chế độ | Trạng thái |
|---|---|---|---|---|
| **I. Đóng nền** | **0b** Sự thật hỏi–đáp (Fuel) | hỏi số dầu đúng nghĩa, đúng nguồn, có bằng chứng | B | ● 0/7 cổng; H4 đã chạy |
| | **0a** Thu báo cáo ngày | tin Zalo → trích → chuẩn hoá → kiểm → người xác nhận | B | ◐ giấy khoá, bộ ca băm, mã đang soát |
| **II. Xương sống rộng** | **1** Hành động có kiểm soát | đúng một phiếu kiểm tra chênh lệch | B | ◐ mã đã gộp, chờ đo lại độc lập |
| | **2** Workflow | điều tra → duyệt → đối soát → đóng | B | ○ |
| | **3** Event | sự kiện nạp / vượt ngưỡng / đính chính / kế hoạch / muộn | B | ○ |
| **III. Thực tế doanh nghiệp** | **4a** Tách tenant / ngành / tích hợp / kênh | 1 tenant chạy 2 ngành + 2 kênh | A + B | ○ |
| | **4b** Ngành thứ hai, chế độ A | CRM: tóm tắt khách + tạo task qua API; lõi ≈ 0 dòng phải sửa | A | ○ |
| | **4c** Bốn kiểu doanh nghiệp | mỗi kiểu một việc | A + B | ○ |
| | **4d** Ngành thứ ba | khác hẳn (thể thao realtime / tài chính) | A / B | ○ |
| **IV. Nền tảng** | **5** Cứng hoá | xác thực thật, vòng đời tenant / pack, bí mật, hạn mức, chi phí, CI, quản trị | — | ○ (CI có PG đã có) |
| | **8** Tích hợp chuẩn hoá | connector Zalo / email / CRM / ERP, webhook, MCP | A | ○ |
| **V. Trí tuệ nâng cao** | **6** AI nâng cao | **model hiểu ý định** (có bằng chứng đòi), PDF / bảng / OCR, rerank, ML thực thể, model local | — | ◐ phần RAG (§5) |
| | **7** Agent có biên | khi luồng cố định không quyết được | — | ○ |
| **VI. Quy mô** | **9** Scale | tải thật, HA / DR, hàng đợi — mở theo số đo | — | ○ |
| | **10** Toàn cảnh | QUERY + COMMAND + WORKFLOW + EVENT trên ≥ 3 ngành | A + B | ○ |

Chi tiết từng phase, bài học đã đo và quan hệ với roadmap cũ: `docs/ROADMAP-DAY-DU.md` (§6–§13); gốc phạm vi: `docs/05-master-roadmap.md`;
tra cứu lịch sử: `docs/ROADMAP.md`.

---

## 5. RAG trong EAIS (luồng L6) — toàn bộ

RAG là **một năng lực**, không phải sản phẩm (`CLAUDE.md` §12): EAIS chỉ cần "bằng chứng", không phụ thuộc bên trong dùng từ khoá, vector
hay lai. Mindset, phân vai và roadmap RAG-0…RAG-8 đầy đủ: `docs/ROADMAP-RAG.md`; việc / hợp đồng / bàn giao: `docs/viec/2-tra-quy-trinh/`.

### 5.1 Tám luật cứng

1. MVP trước (RAG ngoài MVP, D-AI-03 = a). 2. Số vận hành không lấy từ văn bản (S9 chặn lượng lấy từ tài liệu). 3. **Quyền là bộ lọc phần
mềm, chạy TRƯỚC khi tìm** — model không bao giờ thấy đoạn người hỏi không được đọc. 4. Chỉ bản **đã duyệt và đang hiệu lực tại ngày hỏi**
làm căn cứ. 5. Không có gì để nói thì không gọi model. 6. Prompt "chỉ dùng tài liệu" là lời xin; phần mềm kiểm trích dẫn và số. 7. Mọi phương
án dự phòng phải đếm được (không fallback âm thầm). 8. Độ phức tạp phải có bằng chứng — mỗi kỹ thuật là một cổng riêng, mở bằng lỗi đo được.

### 5.2 Luồng nạp (đúng như mã)

```text
tài liệu (app: Markdown + acl.json · bộ đo: .md/.txt/.csv/.pdf, PostgreSQL đọc-chỉ)
  → phần đầu YAML chặt: mã · phiên bản · trạng thái · hiệu lực từ–đến · phạm vi · giữ lại · ACL theo vai
  → cổng chất lượng đọc (bảng sai số ô / mất cột so với bản đã duyệt ⇒ "cần soát", không công bố)
  → Load: xoá + ghi lại cả tenant trong MỘT giao dịch, cùng dòng sổ nạp (advisory lock)
  → chỉ mục giữ sẵn theo tenant, dựng lại khi dấu nạp đổi
  → (đường lai) nhúng theo băm nội dung → document_vectors (RLS), theo không gian model / số chiều / kiểu nhiệm vụ
```

**Cắt đoạn** (`intelligence/rag/rag.go`): mỗi bước / dòng một đoạn; **đường dẫn mục** từ tiêu đề trong thân ("Chương II › Mục 3 › Điều 18. …")
đứng đầu và được chấm cùng mọi đoạn bên dưới; câu dẫn ":" gom với mục con (kể cả mục lồng), trần 1.200 ký tự, tách giữa hai mục cấp đầu;
bảng: header lặp ở mỗi hàng, chỉ chữ của hàng được chấm; tiêu đề trống / chỉ có bảng vẫn là đoạn; tài liệu không có tiêu đề markdown dùng
luật `tieu_de` của nguồn (mẫu → cấp). ID tất định `mã@vN#vị-trí`.

### 5.3 Luồng hỏi (S6–S9)

```text
S6  lọc tenant · vai · phạm vi (cùng một vai) · bản hiệu lực tại ngày hỏi — TRƯỚC khi chấm; IDF chỉ trên đoạn người hỏi thấy
    luật im tất định: câu hỏi nhắc mục đánh số không có trong tài liệu thấy được ("Điều 999") · chủ đề ngoài kho
    từ khoá (IDF, bỏ dấu) ∥ [đường lai] cosine — RRF k=60 — xếp lại nhóm đầu (Pool 30) — ngưỡng "biết im" trên điểm xếp lại
    câu gõ KHÔNG dấu: nhúng + xếp lại bằng câu khôi phục dấu theo CẶP từ của tài liệu người hỏi thấy
    thông báo phiên bản thay cho đoạn khi tài liệu khớp nhất có ≥ 2 bản cùng hiệu lực (xung đột) / không bản nào hiệu lực
S7  giá trị có kiểu của từng đoạn; quyết định trước model: không có / xung đột / lỗi ⇒ mẫu câu, 0 lời gọi model
S8  model (app: trích nguyên văn; bộ đo: gateway theo giao thức chat completions, bằng chứng bọc thẻ có mã, vô hiệu thẻ giả trong nội dung)
S9  claim → đúng id đã đưa · số / mã có trong bằng chứng · chặn lượng nhiên liệu lấy từ tài liệu
S10 câu trả lời + nguồn (mã, phiên bản, vị trí) + vết; S6 ghi đường truy xuất, số đoạn thấy, hạng + điểm, lý do im
    (KHÔNG đếm tài liệu ngoài quyền: người hỏi đọc được vết của chính mình — "không có" = "không được xem")
```

Ra cho người hỏi (luật không lộ): có đoạn hợp lệ ⇒ câu trả lời + nguồn · không khớp **và** ngoài quyền ⇒ **cùng một** mẫu câu · chỉ còn bản
hết hiệu lực ⇒ nói rõ, không dùng làm căn cứ · hai bản cùng hiệu lực ⇒ báo xung đột (`SOURCE_CONFLICT`) · lỗi hệ thống ⇒ "lỗi", không giả
"không có" · HYBRID: truy hồi lỗi vẫn trả phần số liệu đã kiểm.

### 5.4 Luồng đo và vòng đời

- **Đo:** bộ ca do người soạn từ tài liệu gốc (trả lời được · không có trong tài liệu · không có quyền · chỉ còn bản hết hiệu lực · hai bản
  cùng hiệu lực · chèn lệnh · chéo tenant · bảng · câu số liệu phải KHÔNG đi RAG); bộ phát triển + bộ giữ lại (khoá băm, chạy một lần); chấm
  theo tầng sai đầu tiên; so chiến lược thì CỐ ĐỊNH ngân sách ngữ cảnh; cổng đo phải thử phá được (đột biến).
- **Vòng đời:** bản mới = cùng mã, phiên bản khác; hai bản cùng hiệu lực là xung đột; bản bị thay giữ để truy vết; đổi model embedding ⇒
  không gian vector mới, nhúng lại; nạp lại không nhân đôi. **Thiếu:** công bố nguyên tử tài liệu + vector, tự nhúng sau khi nạp, GC vector
  không gian cũ, cổng chặn xoá hàng loạt, cổng chất lượng có trạng thái.

### 5.5 Hiện trạng và số đo (bằng chứng ở `docs/ROADMAP-RAG.md` §10.1)

| Nấc | Đã có | Còn thiếu |
|---|---|---|
| RAG-0 việc + bộ ca | việc "tra quy trình đang hiệu lực"; hợp đồng; bộ S7 phát triển 41 câu + độc lập 106 câu (băm trước, chạy một lần); 2 baseline | người nghiệp vụ duyệt (D-RM-03) |
| RAG-1 nạp | phiên bản / trạng thái / hiệu lực / phạm vi; cổng chất lượng; ID tất định; nguồn theo chiến lược (markdown, files, postgres) | PDF / DOCX / scan thật (U5); độ phủ chữ; băm nội dung, nguồn, chủ sở hữu |
| RAG-2 cắt | theo cấu trúc (U3b) — bộ giữ lại văn bản pháp quy công khai: top-3 chứa đủ đáp án **34 → 39 / 55** | cắt theo token cho bước dài; cổng SC / BI; lý do cắt vào vết |
| RAG-3 truy hồi + quyền | lọc trước khi chấm; **0 rò** trên mọi ca âm; 22 ms / câu ở ~1.000 tài liệu (chỉ mục giữ sẵn) | ngưỡng Recall do người nghiệp vụ chốt |
| RAG-4 sinh có kiểm | §6.2 đủ; gateway chat completions (bộ đo); `Draft.Refused` ⇒ INSUFFICIENT_EVIDENCE | app vẫn model giả (quyết định chi tiền §9.3) |
| RAG-5 nghiệm thu | S7 bộ độc lập: top-3 **87/92**, 0 ngoài quyền, 0 không căn cứ (RAG thường lọt 20 + 37, nạp thẳng 100 + 222); đường lai 83/92, 0/14 câu bẫy nhận đoạn | người nghiệp vụ nghiệm thu; bộ giữ lại MỚI |
| RAG-6 nâng cấp có bằng chứng | đường lai + xếp lại; câu không dấu 0/8 → 4/8 qua ngưỡng; luật mục không tồn tại (0/94 chặn oan) | viết tắt (KL, PXK); BM25 thật; cue cấu trúc làm feature |
| RAG-7 vận hành | thử lại theo Retry-After, nhãn lỗi theo giao thức; đổi nhà cung cấp chỉ đổi `.env` | dự phòng tự động giữa nhà cung cấp; chi phí mỗi lượt; thử tải |
| RAG-8 nâng cao | — | chỉ khi việc đòi |

**Bài học đã đo:** tầng sai đầu tiên của luồng trọn là **router S4**, không phải truy hồi hay model (S7 độc lập: truy hồi 83/92 nhưng chỉ 7/92
tới S8); đổi cách cắt ⇒ phải chọn lại ngưỡng "biết im"; ngưỡng một mình không tách được câu hỏi về mục không tồn tại (điểm lệch qua lại giữa
hai lần chạy) ⇒ thêm luật tất định; câu không dấu sai ở điểm xếp lại chứ không ở truy hồi.

### 5.6 Học từ dự án mở (đánh giá chỉ-đọc 24/10, có `tệp:dòng`)

| Nguồn | Học (ý tưởng, không chép mã) | Không học |
|---|---|---|
| Adaptive Chunking (Ekimetrics, MIT) | thuật toán **split-then-merge** theo ranh giới mạnh nhất + đếm token thật (port sang Go: không gộp qua Chương / Điều, trả offset trực tiếp); **SC + BI** làm cổng mỗi lần nạp | chọn cách cắt theo trung bình metric nội tại (chọn = chấm; chỉ hơn cách cắt cố định tốt nhất **+0,39 điểm**); RC / coref (chỉ tiếng Anh, giấy phép phi thương mại) |
| AdapChunk-App (không có LICENSE ⇒ chỉ học ý) | policy chọn cách cắt **tất định**, ghi lý do theo từng đoạn; bảng tính "mỗi dòng một bản ghi, lặp header"; header bảng theo sang phần tách | LLM chọn / viết lại nội dung; nhúng bản tóm tắt thay chữ gốc; luôn trả top-k (không biết im); fallback âm thầm; ACL hậu kiểm; secret trong mã |
| Model phân loại ngữ nghĩa bên thứ ba (đánh giá cho S4) | đúng vị trí là **S4 — phân loại ý định** (đầu ra đóng, có xác suất, code quyết bước sau); thứ tự: luật → model → policy; S5 quyền vẫn là code | không giao quyền, sự thật, giao dịch, phiên bản; phải thắng baseline tất định **B1** trước |

### 5.7 Việc tiếp (gắn vào RAG-0…RAG-8)

| Thứ tự | Việc | Loại |
|---|---|---|
| 1 | Chỉ nhúng bản làm căn cứ được (không gửi chữ bản nháp / chờ duyệt ra nhà cung cấp) | sửa an toàn (RAG-1) |
| 2 | **S4 phương án (b) = baseline B1:** câu không khớp ý định ⇒ hỏi chỉ mục tri thức với ngưỡng "biết im" trước khi kết luận OUT_OF_SCOPE; vá kiểm capability ở nhánh không khớp | **chủ dự án quyết** (đổi ranh giới S4) |
| 3 | So B0 (luật) / B1 / B2 (luật + model phân loại) / B3 (model trước + policy) trên bộ lỗi S4 độc lập; cổng: giảm ≥ 30% lỗi S4, 0 hồi quy quyền / tenant, không tăng từ chối oan, đo p50 / p95 / chi phí | sau (2); model bên thứ ba cần duyệt + khoá |
| 4 | Split-then-merge theo token + cổng SC / BI / phủ hết | RAG-2 |
| 5 | Công bố nguyên tử tài liệu + vector; tự nhúng sau nạp; GC; cổng xoá hàng loạt | RAG-7 |
| 6 | BM25 thật, cue cấu trúc làm feature, viết tắt; bộ giữ lại MỚI | RAG-3 / RAG-6 |
| — | Không làm khi chưa có lỗi đo được: LLM chọn cách cắt, semantic / proposition chunking, graph, kho vector riêng (pgvector / dịch vụ ngoài) | — |

**Chạy bộ đo RAG** (dữ liệu ngoài repo, thư mục kết quả phải ở ngoài repo):

```bash
go test -count=1 ./intelligence/rag/ ./evals/rag/                          # trên máy, không gọi ra ngoài
EAIS_RAG_CASE=<cấu hình case study> go test -count=1 -run TestCaseStudy ./evals/rag/      # bộ ca bất kỳ qua bảng ánh xạ trường
set -a; . ./.env; set +a                                                    # khuôn biến ở .env.example — KHÔNG commit .env
EAIS_RAG_DONG_CO=1 go test -count=1 -timeout 40m -run TestDongCoThat ./evals/rag/        # động cơ thật: chỉ dữ liệu được phép gửi ra ngoài
```

Mọi nhà cung cấp chỉ cấu hình ở `.env` theo **giao thức** (`EAIS_MODEL_PROVIDER=chat_completions`, `EAIS_EMBED_PROVIDER=embeddings`,
`EAIS_RERANK_PROVIDER=rerank`); mã không chứa tên hãng, endpoint hay model mặc định. Ngưỡng `EAIS_RAG_MIN_SCORE` gắn với model chấm điểm
VÀ cách cắt — đổi một trong hai thì chọn lại trên bộ phát triển, kiểm trên bộ độc lập.

---

## 6. Chạy thử

Chạy nhanh (Go 1.26, Docker, `pdftotext`, `flock`):

```bash
docker compose -f deploy/docker-compose.yml up -d --wait
export EAIS_PG_ADMIN_DSN='postgres://postgres@127.0.0.1:25432/eais?sslmode=disable'
export EAIS_PG_DSN='postgres://eais_app@127.0.0.1:25432/eais?sslmode=disable'
go test ./... -count=1 && scripts/scale-smoke.sh
```

Trước khi chạy **Một lệnh** (dưới) trong cùng shell: `unset EAIS_PG_ADMIN_DSN EAIS_PG_DSN` — đã `export` cả hai biến này thì `scripts/chay-thu.sh` ghi thẳng vào PG đó mà không hỏi.

Ứng dụng `eais` (một binary: HTTP + CLI) chạy trên PostgreSQL với dữ liệu giả lập seed 42. Mặc định **không gọi model**: luồng hỏi dùng model tất định (extractive) và ứng dụng không có cờ hay đường nào tới model trả tiền. Cần: Go (theo `go.mod`), Docker, `curl`, `python3`, `pdftotext` (`poppler-utils`), `flock` (util-linux; tuỳ chọn cho script — thiếu thì script chạy không có khoá dự án và in cảnh báo —, nhưng bắt buộc cho bộ test `go test ./cmd/eais/`; macOS: `brew install flock`).

**Một lệnh** (dựng PG → migrate → sinh seed 42 → nạp → chạy server nền → smoke E2E bằng curl → dừng server):

```bash
git clone https://github.com/luannt2002/enterprise-ai-control-layer && cd enterprise-ai-control-layer
scripts/chay-thu.sh          # EAIS_PG_DOWN=1 để gỡ PG khi xong
```

> **Cảnh báo: script GHI vào PG mà nó dùng** (migrate rồi nạp THAY toàn bộ dữ liệu của tenant trong pack construction). Với giá trị mặc định nó chỉ chạy trên PG do chính nó dựng (container `eais-pg`, cổng `127.0.0.1:25432`) và **từ chối** — thoát 2, chưa ghi gì — nếu container đó đang chạy, cổng đó đã có tiến trình nghe, hoặc dự án compose (`eais`) đã có container nào đang chạy hay tạm dừng, kể cả PG còn sót lại của lần chạy trước. Đã có PG khác đang chạy: đổi **cả ba** `EAIS_PG_CONTAINER`, `EAIS_PG_PORT`, `EAIS_COMPOSE_PROJECT` để script dựng PG riêng (đổi tên và cổng mà giữ dự án thì `docker compose up` cùng dự án sẽ tạo lại container đang chạy và dữ liệu của nó mất, nên script từ chối); chỉ khi chắc PG đang chạy là của bạn và ghi đè được mới đặt `EAIS_REUSE_PG=1`. Ngoại lệ: đã `export` cả `EAIS_PG_ADMIN_DSN` và `EAIS_PG_DSN` (như ở "Chạy nhanh" trên) thì script dùng đúng PG đó, không dựng gì và không hỏi — bỏ chúng khỏi shell (`unset`) nếu không muốn script ghi vào PG ấy.

Thời gian đo ngày 22/10/2026 từ một clone sạch (16 vCPU, ảnh PG đã kéo sẵn, module Go có sẵn; thời gian kéo ảnh lần đầu **chưa đo**): **9,6 giây** với bộ nhớ đệm build rỗng (PG do chính script dựng), **3,9 giây** với bộ nhớ đệm ấm. Thành công khi in `smoke E2E: 21 đạt, 0 hỏng`. Đổi tên container / cổng / địa chỉ nghe bằng `EAIS_PG_CONTAINER`, `EAIS_PG_PORT`, `EAIS_COMPOSE_PROJECT`, `EAIS_ADDR` (đầu `scripts/chay-thu.sh` ghi đủ biến); script không gỡ container PG đang chạy hay tạm dừng (nó từ chối); container đã **dừng hẳn** của cùng dự án thì không chặn: cùng tên và cổng thì `docker compose up` khởi động lại chính container đó (dữ liệu PG trên tmpfs đã mất lúc dừng, mã thoát cũ bị ghi đè, log cũ còn); đổi tên hoặc cổng mà giữ dự án thì compose xoá container cũ rồi tạo mới và log cũ mất (cần `docker logs` của PG đã chết thì lấy trước khi chạy lại, hoặc đổi cả ba biến). Mỗi dự án compose chỉ một lần chạy tại một lúc (mục "Khoá theo dự án" ngay dưới). Server của script nghe ở cổng do hệ điều hành chọn (`127.0.0.1:0`) và script chỉ gọi đúng địa chỉ mà server của nó đã báo, nên một server khác đang giữ cổng cũng không bị gọi nhầm (đặt `EAIS_ADDR` cố định mà cổng bận thì script dừng với "server thoát sớm").

**Khoá theo dự án** (`flock`):
- Hai lần chạy đồng thời cùng `EAIS_COMPOSE_PROJECT` thì lần sau bị **từ chối** (thoát 2, chưa làm gì). Khoá giữ suốt đời script, kể cả bước gỡ PG (`docker compose down`), và vẫn áp dụng khi đặt `EAIS_REUSE_PG=1`. Dự án khác chạy song song bình thường; đường dùng PG có sẵn (cả hai DSN) không chạm compose nên không khoá.
- Tệp khoá là `$EAIS_LOCK_DIR/eais-chay-thu-<dự án>.lock` (0 byte, script không bao giờ xoá). `EAIS_LOCK_DIR` mặc định là `/tmp` viết cứng, **không theo `TMPDIR`**: hai lần chạy cùng dự án từ hai nơi có `TMPDIR` khác nhau (shell thường, IDE, sandbox dùng chung `/tmp`, cron) vẫn khoá cùng một tệp, miễn là cùng thấy một `/tmp` (xem gạch cuối). Đừng đổi `EAIS_LOCK_DIR` giữa hai lần chạy cùng dự án.
- Script bị `SIGKILL` thì khoá nhả khi tiến trình con của bước đang chạy thoát (đã đo với smoke; `docker compose up`, `go build` cùng cơ chế nhưng chưa đo); server nền đã đóng fd khoá nên server mồ côi không giữ khoá hộ (có test và đột biến canh dòng này).
- Khoá chỉ che các lần chạy **cùng thấy một tệp khoá**. Chưa che: người dùng unix khác (không mở được tệp do người kia tạo thì script cảnh báo rồi chạy không khoá; chưa thử với hai tài khoản) và namespace `/tmp` riêng (container, sandbox) mà dùng chung một daemon docker. Che cả hai cần khoá phía daemon, chưa làm.
- Máy không có `flock` (macOS mặc định: `brew install flock`), `EAIS_LOCK_DIR` không ghi được, hay `flock` báo mã lạ (khác 0 và 1, vd hệ tệp không hỗ trợ khoá): script in cảnh báo và đi tiếp **không có khoá**, nên đừng chạy hai lần cùng dự án cùng lúc trên máy đó.

**Từng bước** (cùng việc script làm). Các lệnh dùng `$W/eais` và `$W/data`, nên **đặt `W` trước** (không đặt thì thành `/eais` và `/data`):

```bash
W=$(mktemp -d)
docker compose -f deploy/docker-compose.yml up -d --wait
export EAIS_PG_ADMIN_DSN='postgres://postgres@127.0.0.1:25432/eais?sslmode=disable' EAIS_PG_DSN='postgres://eais_app@127.0.0.1:25432/eais?sslmode=disable'
go build -o $W/eais ./cmd/eais && $W/eais migrate
go run ./tenants/construction/win/gen/cmd/gen -world tenants/construction/win/world.yaml -out $W/data -profile functional -seed 42
$W/eais ingest -pack tenants/construction/pack -data $W/data       # in số quan sát / cách ly và mã lần nạp
$W/eais serve -dev -pack tenants/construction/pack -world tenants/construction/win/world.yaml -data $W/data -now world   # 127.0.0.1:8080
```

Gọi thử (token dev `dev-<tenant>-<principal>` suy từ `world.yaml`, là thông tin công khai):

```bash
H='Authorization: Bearer dev-abc-infra-u-director'
curl -s -X POST localhost:8080/v1/ask -H "$H" -H 'Content-Type: application/json' \
  -d '{"text":"Hôm qua máy 03 được cấp bao nhiêu dầu?","conversation_id":"c1","message_id":"m1"}'   # có trace_id
curl -s "localhost:8080/v1/trace/<trace_id>?format=table" -H "$H"        # bảng vết S0→S10, đánh dấu bước lệch
$W/eais trace -tenant abc-infra <trace_id>                                # cùng bảng, từ CLI
$W/eais trace diff -tenant abc-infra <id1> <id2>                          # bước ĐẦU TIÊN hai lượt cho kết quả khác nhau
$W/eais quarantine -tenant abc-infra <mã lần nạp>                         # bản ghi cách ly kèm lý do
```

## 7. Điểm cuối và luật của app

| Điểm cuối | Việc | Ai được gọi |
|---|---|---|
| `GET /healthz` | 200, hoặc 503 khi PG không trả lời | không cần token |
| `POST /v1/ask` | thân `{"text","conversation_id","message_id"}`; trả câu trả lời + `trace_id` | mọi principal có token |
| `GET /v1/trace/{id}` | vết một lượt (JSON, hoặc `?format=table`) | chính người đã hỏi, hoặc principal toàn quyền dữ liệu; còn lại 404 |
| `POST /v1/ingest` | nạp lại thư mục dữ liệu đã cấu hình của tenant của principal; thân phải rỗng hoặc `{}` | principal toàn quyền dữ liệu |
| `POST /v1/commands/<tool>` | một lượt lệnh của tool khai trong `tools.yaml` của pack (Phase 1, `docs/viec/1-hanh-dong/HOP-DONG.md` §2.6); trả `CommandResult` + `trace_id` | principal có capability của tool theo `policy.yaml` và thực thể trong phạm vi; còn lại 403, cùng một lời cho thực thể ngoài phạm vi và thực thể không có |

- **Tenant, người hỏi, vai chỉ đến từ token**, không bao giờ từ thân: thân có trường `tenant`, `principal`, vai hay bất kỳ trường lạ nào bị từ chối 400. Tenant khác đọc vết ⇒ 404, y hệt "không có".
- **Toàn quyền dữ liệu** tính theo `policy.yaml` của pack, không nêu tên vai: có một vai phạm vi `all` nắm **mọi** capability mà policy định nghĩa. Với pack hiện tại chỉ `director` (accountant có phạm vi `all` nhưng không có `documents.read`).
- **Mọi lượt đã xác thực để lại vết bền** (kể cả lượt lỗi, lượt bị chính sách chặn, thân sai), gồm usage (extractive: 0 token). Lượt hỏi không lưu được vết thì không giao câu trả lời. Route lệnh làm NGƯỢC LẠI, có chủ ý (HOP-DONG §7.5 "Mất vết"): lệnh có thể đã có tác dụng (phiếu đã commit và đã đọc lại) nên kết quả vẫn được trả, chỉ để `trace_id` rỗng, tăng bộ đếm `trace_save_failed` và ghi log lỗi; lượt lệnh không có usage (không gọi model). Lượt chưa xác thực được tenant (401) không có dòng vết theo tenant; chúng chỉ được đếm trong log tổng kết khi tắt server.
- Mã lần nạp là băm của (dữ liệu, pack, mã): nạp lại cùng dữ liệu cho cùng mã và không nhân đôi bản ghi cách ly. Mỗi lần nạp thành công thêm một dòng vào sổ nạp `load_records` (kể cả nạp lại cùng dữ liệu): dòng mới nhất là phiên bản mà vết khai.
- **Phiên bản mã** (`code_version` của vết, và phần "mã" của mã lần nạp) là mã git của cây đã dựng binary: `scripts/chay-thu.sh` nhúng nó bằng `-ldflags` (thêm `+dirty` khi cây có sửa chưa commit; CI đặt `EAIS_CODE_VERSION` = commit đang kiểm) rồi đối chiếu với phiên bản mà server báo lúc khởi động. `go build` tay trong một clone thường tự ghi mã git vào binary; trong git worktree liên kết hay cây không có `.git` thì không có, và khi đó ứng dụng dùng băm sha256 của chính binary với tiền tố `bin-sha256:` — không bao giờ ghi `(devel)`. Băm binary là dấu của BINARY, không phải của mã nguồn: chỉ so được giữa các vết do cùng một binary ghi (cùng commit dựng ở hai thư mục cho hai băm khác nhau, vì Go nhúng đường dẫn nguồn khi không có `-trimpath`). Hai bản sửa khác nhau chưa commit trên cùng một commit vẫn cùng phiên bản `+dirty`.
- **Migration:** 0001–0006 của team (0006 = sổ nạp), 0007 tài liệu có phiên bản, 0008 vector tài liệu (nhánh RAG, gộp 24/10). Migration mới tiếp theo lấy **0009**; ai gộp main trước giữ số.

## 8. Giới hạn đã biết (ghi thẳng)

- `serve` chỉ chạy được với `-dev` (token công khai, chỉ nghe loopback, chặn `Host` không phải loopback). Chưa có cơ chế xác thực cho môi trường thật — đang chờ quyết định.
- Phiên bản dữ liệu ghi vào vết (`traces.Versions.Data`, cột `decision_records.data_version`) là phiên bản của lần nạp **đã commit vào PG** của đúng tenant, đọc từ PG (R2-01, migration 0006): `Load` ghi một dòng vào sổ nạp `load_records` (mã lần nạp, băm dữ liệu) trong **cùng giao dịch** thay dữ liệu của tenant, và server đọc dòng mới nhất của tenant ở đầu **mỗi** lượt hỏi và lượt lệnh. Nên vết không còn khai một phiên bản chưa từng được nạp khi thư mục dữ liệu đổi sau lần nạp, khi `serve` trỏ nhầm thư mục khác (ca README từng tái hiện: thêm một dòng trống vào bản sao thư mục), hay khi PG vừa được nạp bằng `eais ingest` lúc server đang chạy — không cần khởi động lại server (ba ca đều có test đỏ trên mã cũ: `cmd/eais/data_version_test.go`). Tenant chưa có lần nạp nào được ghi nhận, hoặc lần gần nhất không kèm phiên bản (đường `Load` cũ của bộ chạy `winrun` và test), thì vết khai `(chưa ghi nhận lần nạp)`, không bịa một băm — kể cả PG đã nạp từ trước migration 0006, cho tới lần nạp kế tiếp. Không đọc được sổ nạp: lượt hỏi lỗi, đóng mặc định — **503** khi lỗi hạ tầng (mất kết nối, quá tải), **500** khi quyền hoặc lược đồ lệch (SQLSTATE lớp 42, vd vai ứng dụng mất quyền SELECT trên `load_records`, hay bảng không có; `platform/pgstore/pgerr_test.go` ghim 42501 và 42P01 ra 500, fault `code`); cả hai có vết khai phiên bản rỗng và không có câu trả lời; lượt lệnh vẫn chạy (như "Mất vết" ở trên) và vết khai rỗng; `serve` không khởi động. Thư mục `-data` chỉ còn dùng cho `eais ingest` và `POST /v1/ingest` (và kiểm lúc khởi động rằng nó dùng được): `serve` trỏ nhầm thư mục không làm vết khai sai, nhưng `POST /v1/ingest` sẽ nạp đúng thư mục nhầm đó vào PG. Kho bộ nhớ (chỉ bản dựng thử trong test) nạp thẳng từ thư mục nên vết vẫn khai băm thư mục như trước.
  Giá đo được: một lần đọc ≈ 70 µs (`BenchmarkLatestLoad`, đo ngày 23/10/2026, 8 lượt chạy × 3000 lần, PG 17 cùng máy: 67–71 µs ở sổ 1 dòng, 68–74 µs ở sổ 5000 dòng của một tenant; một lượt chạy đầu sau khi tạo bảng đo 1062 µs, không tái hiện ở 7 lượt sau, nguyên nhân chưa rõ). Một lượt hỏi đầu-cuối qua HTTP ≈ 7–8 ms trên máy dùng chung này, nhiễu giữa các lượt đo lớn hơn 70 µs nên chênh lệch của cả lượt **chưa tách được**.
  **Còn lại (giới hạn):** phiên bản được đọc **một lần ở đầu lượt**; một lần nạp commit **giữa** lượt (cửa sổ dài bằng một lượt: vài mili giây ở chế độ không model, như số đo ở trên; khi có model thì dài bằng thời gian lượt chờ model, chưa đo) làm lượt đọc dữ liệu mới mà vết vẫn khai phiên bản cũ — `TestDataVersionInTurnWindowKnownLimit` ghim đúng điều đó, và sẽ đỏ khi ai khép được cửa sổ (sửa test cùng dòng này). Suy luận, chưa làm, chưa đo: khép được mà không đổi `core/pipeline` bằng đọc lại phiên bản sau lượt rồi chạy lại lượt hỏi nếu nó đổi (thêm một truy vấn mỗi lượt; lượt lệnh không chạy lại được vì đã ghi phiếu), hoặc bằng đọc dữ liệu và phiên bản trong một snapshot của cả lượt (giữ một kết nối PG suốt lượt, kể cả lúc chờ model); chưa làm vì chưa có ca thật đòi (CLAUDE.md, P7).
  **Nâng cấp từ bản chưa có sổ nạp (trước việc 5e):** sau khi áp migration 0006, MỌI bên nạp dữ liệu vào PG phải chạy binary có sổ nạp. Binary cũ không biết bảng `load_records`: nó vẫn thay dữ liệu của tenant nhưng không ghi dòng sổ. Hậu quả đo được (tái hiện ngày 23/10/2026 bằng hai binary, trước và sau 5e, trên một PG riêng): khi sổ đã có ít nhất một dòng do binary mới ghi, một lần `eais ingest` bằng binary cũ với bộ dữ liệu khác để sổ nguyên một dòng mà dữ liệu sống đã đổi, và server mới vẫn khai trong vết phiên bản của lần nạp TRƯỚC — đúng loại lỗi mà sổ nạp sinh ra để chặn, và không kiểm tra nào phát hiện. (DB đã áp 0006 mà chưa có dòng sổ nào thì an toàn: vết khai `(chưa ghi nhận lần nạp)`.) Trình tự nâng cấp: (1) dừng MỌI bên nạp bản cũ — cron, `eais ingest`, server cũ (`POST /v1/ingest`) — TRƯỚC khi migrate; (2) `eais migrate` (chạy lại được); (3) chạy `serve` và mọi bên nạp bằng binary mới (bản mới từ chối mở kho khi chưa áp 0006 và nói rõ cần `eais migrate`: `TestReadySoNap`); (4) nạp lại MỘT lần (`eais ingest`) để sổ có dòng đầu tiên — tới lúc đó vết khai `(chưa ghi nhận lần nạp)`. Bước (1) không có gì ép: nhiều bản chạy dùng chung một PG dev là chỗ dễ vướng.
- Lượt mà `Handle` trả lỗi (kể cả bị chính sách chặn, vd tenant lạ) chỉ để lại MỘT bước vết `S?` đánh dấu TRỐNG, không chỉ được tầng lỗi, vì `Handle` trả kết quả rỗng khi lỗi (`core/pipeline` ngoài phạm vi của ứng dụng). Chỉ tiêu "người chỉ đọc vết mà chỉ đúng tầng sai đầu tiên ở ≥ 90% ca đỏ" **chưa đo**, và với lượt lỗi cơ chế hiện tại không cho đạt: cần `Handle` trả kết quả một phần khi lỗi.
- Chưa có vết các bước nạp (`L1.*`) và chưa có chính sách dọn / lưu giữ vết và sổ nạp `load_records` (một dòng mỗi lần nạp; vai ứng dụng không có quyền xoá).
- CI ở `.github/workflows/ci.yml` chưa chạy trên GitHub (xem ghi chú đầu tệp).
- **RAG:** app chạy đường từ khoá + model trích (7/14 câu không có đáp án vẫn nhận đoạn ở đường từ khoá — đường lai 0/14 nhưng chỉ ở bộ đo); việc nhúng hiện gửi cả chữ bản nháp ra nhà cung cấp embedding (§5.7 việc 1); tài liệu và vector chưa công bố nguyên tử; nạp chỉ Markdown ở app.

## 9. Tài liệu chi tiết

| Đọc gì | Ở đâu |
|---|---|
| **Câu đang chờ chủ dự án quyết** | [`docs/CHO-QUYET-DINH.md`](docs/CHO-QUYET-DINH.md) |
| **Roadmap hiện hành** (D-RM-01): Phase 0–10, 7 mặt phẳng, luồng L0–L10, MVP §5b | [`docs/ROADMAP-DAY-DU.md`](docs/ROADMAP-DAY-DU.md) |
| **Roadmap RAG**: mindset, 4 luồng, phân vai, RAG-0…RAG-8, hiện trạng có số | [`docs/ROADMAP-RAG.md`](docs/ROADMAP-RAG.md) |
| Việc RAG: hợp đồng, kế hoạch, bàn giao tệp chung, bảng bên thứ ba | [`docs/viec/2-tra-quy-trinh/`](docs/viec/2-tra-quy-trinh/) |
| Việc Phase 1 (tạo phiếu) | [`docs/viec/1-hanh-dong/`](docs/viec/1-hanh-dong/) |
| Trạng thái chi tiết, tiến độ từng lần push | [`docs/STATUS.md`](docs/STATUS.md), [`docs/bao-cao/`](docs/bao-cao/) |
| WIN MVP: 7 cổng, phạm vi | [`docs/04-core-mvp.md`](docs/04-core-mvp.md) |
| G7: hợp đồng benchmark, chi phí, checkpoint | [`docs/g7/`](docs/g7/) |
| Kiến trúc | [`docs/01-architecture.md`](docs/01-architecture.md) |
| Quyết định | [`docs/adr/`](docs/adr/) |
| Roadmap gốc 20/10 (gốc phạm vi) · tra cứu lịch sử | [`docs/05-master-roadmap.md`](docs/05-master-roadmap.md) · [`docs/ROADMAP.md`](docs/ROADMAP.md) |
| Kế hoạch V3 (tra cứu) | [`docs/v3/`](docs/v3/) |
| Chạy lại mốc / làm tiếp trên máy khác | [`docs/ops/README.md`](docs/ops/README.md) |
| Hợp đồng vận hành (luật bắt buộc) | [`CLAUDE.md`](CLAUDE.md) |
| Thay đổi | [`CHANGELOG.md`](CHANGELOG.md) |
