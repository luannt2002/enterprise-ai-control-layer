# 04 — EACL Core MVP: Enterprise Truth & Routing Proof

> **Hợp đồng của R1** (Fuel Truth Benchmark, cổng W-G1…W-G7; D-V3-10, chủ dự án duyệt 20/10/2026); luật D5 sửa đổi ở ADR-0005. Hướng tổng thể: `docs/v3/`.
>
> **Chứng minh phiên bản nhỏ nhất của EACL đáng được tồn tại.**
>
> Trạng thái:
> - **Hướng đã chốt** (user, 19/10/2026 21:01): làm WIN trước, không làm quá rộng; đạt WIN rồi mới mở rộng theo roadmap đầy đủ.
> - **Ngưỡng ở mục 6 và quyết định D1–D6 (mục 9): tự chốt ngày 20/10/2026**, khi user cho phép chạy tự động không hỏi. Ghi ở `docs/adr/0001-win-mvp.md`. **User đảo được bất cứ lúc nào.**
> - Bản đồ dài hạn nằm ở `05-master-roadmap.md`. **Roadmap không được kéo tính năng ngược vào tài liệu này.**

## 0. MVP là bằng chứng tối thiểu, không phải tính năng tối thiểu

| | Trả lời câu hỏi |
|---|---|
| CORE | EACL là gì: 5 bất biến (mục 2) |
| **WIN MVP** | thí nghiệm nhỏ nhất kiểm tra luận đề (mục 1) có đúng không |
| ROADMAP | thắng rồi thì đi đâu (`05`) |

**Luật khoá phạm vi.** Tính năng nào không cần để thắng bài thử "Fuel Discrepancy: EACL vs baseline RAG" thì **mặc định không thuộc MVP**.

`phase: MVP` trong `docs/requirements/catalog.yaml` (294 REQ) **KHÔNG phải phạm vi WIN**. Phạm vi WIN là G1–G7 (mục 6). REQ nào không truy được về một cổng thì coi là Phase 1+ cho tới khi một ca WIN đỏ đòi nó.

## 1. Luận đề phải chứng minh

> Khi dữ liệu nghiệp vụ đến từ nhiều định dạng và nhiều nguồn, lại chứa mơ hồ và xung đột, thì một **hệ điều khiển tất định có RAG/LLM làm capability bên trong** cho ra câu trả lời nghiệp vụ **đáng tin và truy vết được hơn** một hệ lấy RAG làm trung tâm.

Không chứng minh được thì **dừng và sửa kiến trúc**, không đi tiếp roadmap.

## 2. CORE: 5 bất biến

Bài thử: bỏ Agent, MCP, reranker, thể thao, hay bỏ LLM khỏi một yêu cầu, thì EACL **vẫn là EACL**. Nhưng bỏ một trong 5 bất biến dưới đây thì **mất bản chất**.

| Mã | Bất biến | Nghĩa |
|---|---|---|
| C1 | Nghĩa nghiệp vụ chuẩn | Dữ liệu nhiều định dạng, nhiều nguồn, bẩn → **nghĩa** nghiệp vụ chuẩn. `4k` · `4.000 L` · `4 m3` · `bốn ngàn lít` → cùng một lượng. Các nghĩa ISSUED · RECEIVED · CONSUMED · REPORTED · ESTIMATED không được trộn. Một nghĩa có thể có nhiều nguồn khác thẩm quyền (ví dụ ISSUED: phiếu xuất kho hạng 1, ERP hạng 2) |
| C2 | Sự thật ≠ model | Số đúng đến từ fact có cấu trúc, tài liệu có nguồn, sự kiện, phép tính, nguồn có thẩm quyền. Model không tạo ra sự thật |
| C3 | Tất định trước | Phần mềm giải chính xác được (SQL / luật / phép tính / tìm kiếm) thì không dùng AI. Bất định thật mới dùng RAG / model |
| C4 | AI bị kiểm soát | Đầu ra model là **đề xuất** → kiểm → ràng buộc bằng chứng → policy → nhận / từ chối / hỏi lại. Đi kèm hai bất biến nền: tenant lấy từ danh tính đã xác thực; quyền kiểm ngoài model, đóng mặc định (CLAUDE.md §11, §21) |
| C5 | Chứng minh được | Kết quả → quyết định → route → bằng chứng → nguồn → phiên bản → nghĩa nghiệp vụ. Mọi cổng có ca âm chứng minh nó chặn được lỗi |

## 3. Một bài toán duy nhất: Construction · Fuel Discrepancy Resolution

> Khi nhiều nguồn trong doanh nghiệp báo lượng nhiên liệu khác nhau cho cùng một thiết bị và một khoảng thời gian, EACL phải hiểu chúng đang nói về **fact nghiệp vụ nào**, nguồn nào **có thẩm quyền cho đúng câu đang hỏi**, phát hiện xung đột hoặc mơ hồ, và trả kết quả **có bằng chứng** thay vì để model đoán.

**Ca chuẩn**, đi xuyên qua cả core. Trong thế giới WIN, "máy 03" chỉ thuộc EX-03; ca mơ hồ dùng một cặp máy khác.

```text
Tin nhắn hiện trường : "máy 03 sáng 2k chiều thêm 2k"            → REPORTED (ước lượng)
Báo cáo ngày (PDF)   : Machine 03 ≈ 4 m³                        → REPORTED / ESTIMATED
Phiếu xuất kho       : EX-03 xuất 3.850 L                        → ISSUED, nguồn hạng 1
ERP                  : asset-8392 (= EX-03) ghi xuất 3.900 L     → ISSUED, nguồn hạng 2
Cảm biến             : EX003 tiêu thụ 3.782 L                    → CONSUMED
SOP                  : quy trình xử lý chênh lệch (văn bản cho RAG)

Hỏi: "Hôm qua máy 03 được cấp 4k dầu đúng không?"
```

| Bước | Thành phần | Kết quả |
|---|---|---|
| "máy 03" | nhận dạng thực thể | EX-03 |
| "hôm qua" | thời gian | ngày cụ thể theo múi giờ tenant |
| "được cấp" | nghĩa nghiệp vụ | ISSUED (≠ CONSUMED ≠ REPORTED) |
| "4k" | chuẩn hoá | 4.000 L |
| "đúng không?" | ý định | **xác nhận một con số**, không phải tra số |
| nhiều giá trị | thẩm quyền trong ISSUED | phiếu xuất kho > ERP; lệch 50 L > dung sai 10 L thì lộ xung đột |
| câu hỏi đòi số chính xác | route | SQL / fact |
| kết quả | kiểm + trace | có bằng chứng |

**EACL phải trả lời theo kiểu:** "Không. Lượng xuất kho có thẩm quyền (phiếu xuất kho) là 3.850 L. ERP ghi 3.900 L, lệch quá dung sai. Khoảng 4.000 L là số đội báo, cảm biến đo tiêu thụ 3.782 L." Kèm trích dẫn và trace.

**Baseline RAG dễ trả lời:** "Khoảng 4.000 L, đúng."

## 4. Phạm vi WIN

```text
Tenant/Identity → Normalize → Entity → Business semantics → Source authority → Intent/Router
   → { SQL · CALCULATION · RAG } → Context builder → Model (CHỈ khi route cần) → Validator → Trace/Evidence
```

| Hạng mục | Trong WIN |
|---|---|
| Lối vào | **chỉ QUERY**, gọi trong tiến trình qua CLI `eacl ask` / `eacl eval`. HTTP API để sau WIN |
| Route (7, tên theo hợp đồng mã) | `SQL_QUERY` · `CALCULATION` · `RAG` · `HYBRID_RETRIEVAL` · `CLARIFICATION` · `REJECT` · `HUMAN_ESCALATION`. Trong WIN, `HUMAN_ESCALATION` chỉ trả kết quả `HUMAN_REVIEW_REQUIRED`, **không tạo tác vụ** (tác vụ thuộc Phase 2) |
| Kết quả có kiểu | trường `Outcome` trên phản hồi:<br>• `RESOLVED`: có câu trả lời từ nguồn có thẩm quyền; xung đột với nguồn thấp hơn vẫn là RESOLVED nhưng **phải lộ ra** (`conflict`)<br>• `CLARIFICATION_REQUIRED`: mơ hồ / thiếu thực thể, thiếu loại vật tư, nhiều ý định, chào hỏi<br>• `INSUFFICIENT_EVIDENCE`: không có dữ liệu cho (thực thể, nghĩa, kỳ)<br>• `HUMAN_REVIEW_REQUIRED`: ngang thẩm quyền, lệch quá dung sai<br>• `PERMISSION_DENIED`: thiếu capability hoặc ngoài phạm vi<br>• `OUT_OF_SCOPE`: ngoài phạm vi trợ lý<br>• `SYSTEM_FAILURE`<br>• `SOURCE_CONFLICT`: để dành cho policy khác, WIN không dùng |
| Nguồn | ERP (JSON) · phiếu kho (CSV / XLSX) · báo cáo ngày (PDF **văn bản**) · tin nhắn hiện trường · cảm biến (CSV) · SOP (markdown, cho RAG) |
| Tenant | **A**: đủ bộ dữ liệu Fuel Discrepancy. **B**: *canary* cách ly, cùng ngành, dữ liệu tối thiểu. Không phải tenant thứ hai đầy đủ, **không phải thể thao** |
| Lưu trữ | PostgreSQL 17 + RLS, vì cổng G5 phải chứng minh được cách ly ở tầng DB |
| RAG | tối thiểu: SOP → chunk → index → retrieval → `Evidence` qua `KnowledgeRetriever`. Theo `docs/02` §2–§3: retriever + embed **dời từ hệ cũ**; hybrid search **tách** khỏi tệp store lớn của hệ cũ (sửa luôn F14/F15/F32/F38); chunk / index **viết mới bằng Go**. Phần tenancy / cache / room của hệ cũ **không** dời. Không có khoá nhúng thì chạy toàn văn PG và ghi rõ. Đạt recall trên ca WIN thì **DỪNG**: không rerank, không GraphRAG |
| Model | một gateway; provider / model lấy từ cấu hình, không hard code; chỉ dùng cho `RAG` / `HYBRID_RETRIEVAL`. **Lý do mở** (cổng CLAUDE.md §19): luận đề phải chứng minh AI *bị kiểm soát*, và câu hỏi quy trình SOP không trả lời được bằng mẫu câu |
| Câu HYBRID chuẩn | cùng bộ dữ liệu đối soát: "EX-03 hôm qua kho xuất và cảm biến chênh bao nhiêu, và quy trình xử lý chênh lệch là gì?" → SQL + CALCULATION (hiệu) + RAG (đoạn quy trình) + model diễn đạt. Kiểm tuân thủ định mức là **Phase 1+**, không thuộc WIN |
| Python | **không dùng trong WIN** (D4). Báo cáo ngày là PDF văn bản, đọc bằng thư viện Go, đo ở W1 bước 0. Bảng PDF và bộ đọc Python (torch) để sau WIN |
| Dữ liệu | FUNCTIONAL (khoảng 1k–10k bản ghi) · REALISM (đủ bẩn / nhiều định dạng / xung đột cho từng lớp hỏng) · SCALE SMOKE (đủ lớn để lộ lỗi kiến trúc hiển nhiên; **không phải tuyên bố quy mô**) |
| Đối thủ | **Baseline RAG** (D6). Chỉ nhận dữ liệu **tenant A** (mỗi bản ghi nguồn thành một đoạn văn bản), cùng retriever, cùng model, cùng ngân sách. **Không có** tầng tất định, bộ lọc vai / phạm vi, hay ràng buộc claim, vì đó là tầng kiểm soát của EACL. Rò rỉ tenant đo trên EACL (phải bằng 0); lớp quyền so ở mức vai / phạm vi |

**KHÔNG thuộc WIN** (đã có chỗ trong `05`):

- agent, đa agent, MCP;
- lệnh / hành động (Phase 1); workflow, tác vụ duyệt (Phase 2); nền tảng sự kiện (Phase 3); thể thao (Phase 4);
- GraphRAG, knowledge graph, ML entity resolution, ML intent, rerank, vision / OCR, fine-tune, model local, bộ đọc Python;
- Kafka, Redis, Kubernetes, tách microservice;
- HTTP API đầy đủ; truy vấn "theo hiểu biết tại thời điểm T"; phân quyền mức trường; kiểm tuân thủ định mức; benchmark trăm triệu dòng.
- Ontology đầy đủ (audit #2), tiến hoá schema (#10), đính chính muộn / dữ liệu song thời (#3), hiệu lực thực thể theo kỳ (#4): **DEFER, mở khi một ca WIN đỏ vì thiếu chúng** (`05` §4).

## 5. Ma trận lớp hỏng: trận nào EACL phải thắng

`failure_class` trong bộ ca WIN lấy đúng các mã dưới đây.

| Mã | Lớp hỏng | Baseline dễ sai thế nào | EACL phải làm | Cổng |
|---|---|---|---|---|
| `representation` | Biểu diễn | `4k` / `4 m³` / `4.000L` hiểu lệch | chuẩn hoá về một lượng | G1 |
| `entity` | Thực thể | EX003 / Machine-03 / asset-8392 thành ba máy | về đúng EX-03 | G2 |
| `ambiguity` | Mơ hồ | hai "máy 07" thì đoán một | `CLARIFICATION_REQUIRED` kèm ứng viên | G2 |
| `semantic` | Ngữ nghĩa | lẫn lượng xuất / tiêu thụ / báo cáo | tách đúng nghĩa câu hỏi | G3 |
| `source_conflict` | Xung đột nguồn | chọn 4.000 vì "nghe hợp lý", lặp nhiều | thẩm quyền trong cùng nghĩa; lộ xung đột | G3 |
| `permission` | Quyền / tenant | lộ dữ liệu ngoài vai / phạm vi | từ chối; 0 rò rỉ | G5 |
| `unsupported_answer` | Câu không có căn cứ | model nói khác bằng chứng | chặn: claim phải khớp chủ thể / chỉ số / thời gian / giá trị / đơn vị / nguồn | G5 |
| `routing` | Định tuyến | câu số liệu đi RAG; "xin chào" gọi model | route đúng; route tất định không gọi model | G4 |

## 6. 7 cổng WIN

| Cổng | Đạt khi (chạy bằng lệnh, ra đạt / không đạt) |
|---|---|
| **G1 DATA** | Bộ sinh ghi **manifest** (id bản ghi nguồn, quan sát kỳ vọng, biểu diễn). Một chương trình kiểm **không import bộ nạp** so tập id của manifest với (quan sát ∪ `Quarantine{RecordID, Reason}`): thiếu hoặc thừa id là KHÔNG ĐẠT, tức 0 bản ghi rơi im lặng. Test metamorphic: **cùng một bản ghi của cùng một nguồn** ở ≥ 4 biểu diễn (CSV / XLSX / JSON; tin nhắn viết nhiều kiểu) ra **cùng một** quan sát chuẩn, so theo phép chiếu (thực thể, nghĩa, giá trị + đơn vị chuẩn, ngày, loại nguồn). **Khác nguồn thì khác nghĩa hoặc khác thẩm quyền, không được gộp.** 100% ca `representation` |
| **G2 IDENTITY** | 100% ca `entity` (bí danh, mã theo nguồn) về đúng thực thể. 100% ca `ambiguity` ra `Outcome = CLARIFICATION_REQUIRED` kèm đúng tập ứng viên. 0 lần đoán im lặng |
| **G3 TRUTH** | 100% ca `semantic` + `source_conflict` lấy đúng nguồn có thẩm quyền **trong đúng nghĩa câu hỏi**. Mỗi cặp nguồn có ca biên: lệch = dung sai thì không xung đột; lệch = dung sai + 1 thì xung đột được lộ (`Outcome = RESOLVED`, `conflict = true`). Ngang thẩm quyền: 0 lần trả một con số (`HUMAN_REVIEW_REQUIRED`). Câu xác nhận ("4k đúng không?") trả lời theo nghĩa đúng, kèm các giá trị khác được gắn nhãn |
| **G4 DECISION** | Hai cột trong bảng ca: route router chọn và `Outcome` cuối, tên theo mã. 100% ca `routing` đúng cả hai. Đo bằng **gateway giả đếm lời gọi**, không dựa trace do hệ tự ghi: ca `RAG` / `HYBRID_RETRIEVAL` gọi gateway **đúng 1 lần** (đối chứng dương); ca route tất định gọi **0 lần**. Đột biến "SQL gọi gateway" và "RAG bỏ gateway" đều phải bị bắt |
| **G5 CONTROL** | **Tenant:** `Principal` chỉ dựng từ khoá / registry, không kiểu request nào có trường tenant (test kiểu AST như `architecture_test.go`). Store và danh mục công bố tenant của chúng, S1 so với `Principal`. Từ W4: `SET LOCAL eacl.tenant_id` lấy từ `Principal` + test đọc lại GUC. Canary B: 0 dòng / đoạn / thực thể / bằng chứng của B tới được từ A (SQL, truy xuất, nhận dạng, bằng chứng), có test cho phép + test từ chối. **Quyền:** đóng mặc định; 100% ca `permission` đúng; truy xuất tôn trọng quyền tài liệu (cho phép + từ chối). **Model:** mọi số trong câu đã giao được đếm bằng **bộ đếm độc lập** (không dùng chính bộ chuẩn hoá); mỗi số phải ràng buộc đủ bộ (chủ thể / chỉ số / thời gian / giá trị / đơn vị / nguồn) với fact hay phép tính. Số trích từ SOP chỉ qua khi **không mang đơn vị của chỉ số đang được hỏi** (lượng nhiên liệu: L, m³…), ví dụ số bước, hạn "24 giờ", và được gắn nhãn "trích văn bản quy trình". Số lượng nhiên liệu nằm trong SOP **bị chặn**. Bảng "sai đúng một chiều": mỗi chiều trong 6 chiều có ≥ 3 ca, 100% bị chặn. ≥ 10 ca hợp lệ: 0 chặn oan. 100% ca prompt injection trong SOP **không đổi được** route / quyền / con số |
| **G6 PROOF** | Bộ ca WIN **đóng băng trước W1** (băm sha256 ghi vào báo cáo); mỗi ca có `failure_class` + `authored_by`. ≥ 100 ca; mỗi lớp mục 5 ≥ 5 ca; ≥ 40% ca do **phiên không viết mã** soạn. Mọi câu đã giao có trace ghi **quyết định** (chuẩn hoá, thực thể, ý định, route, id bằng chứng, phán quyết) + phiên bản (pack / mã / bộ dữ liệu). **Kiểm phát lại:** chạy lại các chặng tất định từ trace phải ra đúng phản hồi; đột biến đổi route sau khi ghi trace phải bị bắt. Mỗi cổng kiểm có đột biến bị bắt. Thuộc tính: tổng của chỉ số có `min ≥ 0` thì ≥ 0; fact dẫn xuất được âm và \|giá trị\| = \|a − b\| với a, b lấy từ bằng chứng; A không bao giờ thấy B |
| **G7 VALUE** | Cùng dữ liệu (tenant A), cùng bộ ca; model thật, temperature 0; mỗi hệ chạy **k = 3** lần, lấy trung vị. Chấm bằng **bộ đếm độc lập** với đáp án lấy từ manifest của bộ sinh, không lấy từ EACL. **Luật:** với **mọi** lớp mục 5, EACL ≥ baseline (hoà được phép); ở **≥ 4/8 lớp**, EACL hơn baseline **≥ 20 điểm %**; EACL: rò rỉ tenant = 0 và claim không căn cứ = 0. Độ trễ / chi phí **chỉ báo cáo** |

**WIN = 7/7 cổng đạt VÀ scale smoke đạt.** Không cần thể thao, Agent, MCP, Kafka hay trăm triệu dòng.

**Model giả hay thật.**
- G4 / G5 chạy với **model giả tất định**, tức đầu ra phá có chủ ý, để có đạt / không đạt.
- G7 chạy với **model thật**, cùng một model cho cả hai hệ. Kết quả ghi cấp bằng chứng (`03` §4).
- Không có khoá model trên máy thì G7 là `CHẶN`, ghi rõ, **không** thay bằng model giả.

**Scale smoke** đạt khi chạy lại bộ ca trên hồ sơ smoke ra **cùng kết quả đúng** như trên FUNCTIONAL và không lỗi. Thời gian / RAM ghi kèm lệnh, không đặt ngưỡng, vì chưa tuyên bố quy mô.

## 7. Hiện trạng so với WIN (trạng thái theo `05` §2: KEEP · FIX · BUILD · DEFER · REMOVE)

MOVE / DECOUPLE / REPLACE là **động từ di trú** của `docs/02`, không phải trạng thái năng lực.

| Thành phần | Trạng thái | Việc cho WIN |
|---|---|---|
| `core/contracts`, `core/decimal`, `core/tenant` | KEEP + FIX | thêm `Outcome`; thêm trường ràng buộc claim. **Xong 20/10** |
| `data/normalization` | KEEP | FIX theo ca REALISM |
| `data/entityresolution` | FIX | bí danh theo nguồn (EX003, asset-8392, Machine-03). **Xong 20/10** |
| `data/sourceoftruth` | FIX | truy vấn kiểm claim: **xong 20/10** (bằng chứng mang chủ thể / chỉ số / kỳ; `Compare`). Nghĩa RECEIVED / ESTIMATED: chưa ca WIN nào đòi |
| `intelligence/intent`, `intelligence/router` | FIX | ý định xác nhận; thiếu thuật ngữ thì hỏi lại; "dùng" → CONSUMED (D3); route `RAG` / `HYBRID_RETRIEVAL`; quyền đóng mặc định. **Xong 20/10** |
| `reliability/validation` | **BUILD** | ràng buộc claim ↔ bằng chứng. Phép kiểm "có mặt" hiện tại giữ làm lớp lọc đầu. **Xong 20/10** (`BindClaims`, ADR-0002 D12) |
| `core/pipeline` | FIX | nối `RAG` / `HYBRID_RETRIEVAL`; `Outcome`; trace đủ chặng. **Xong 20/10** |
| `simulations/fixtures` + `tenants/construction/seed` | REMOVE (cũ) + BUILD (mới) | thay bằng bộ nạp nhiều định dạng + bộ sinh dữ liệu WIN có seed. BUILD **xong 20/10**; REMOVE chưa (test đầu–cuối cũ còn dùng seed) |
| `evals` | BUILD | bộ ca WIN + runner + baseline RAG + ma trận lớp hỏng. Bộ ca, runner, ma trận: **xong 20/10**; baseline: đang dựng |
| `observability/traces` | FIX | ghi quyết định + id bằng chứng (G6): **xong 20/10** (`Decision` + `Replay`). Lưu bền: chưa (chưa có HTTP API) |
| `scripts/mutate.sh`, `architecture_test.go` | KEEP | |
| Retrieval, embed, client LLM của hệ cũ | KEEP (dời theo `docs/02`) | dời đúng lúc lát W5 cần |
| Tài liệu và ca thể thao; ca COMMAND / WORKFLOW / EVENT | DEFER | Phase 1–4 (`05` §4) |

## 8. Lát WIN: thứ tự theo cổng

| Lát | Nội dung | Đóng cổng |
|---|---|---|
| W0 ✓ | hỏi số liệu tất định trong bộ nhớ, dữ liệu D0/D1 (19/10); quyền đóng mặc định (20/10) | một phần G2 / G3 / G4 / G5 |
| W0.5 ✓ | **đóng băng bộ ca WIN** (QUERY, `failure_class` + `authored_by`, ca phá do agent không đọc mã soạn) + `evals` kiểm số ca | điều kiện của G6 |
| W1 ✓ | đo bộ đọc PDF Go (bước 0); bộ sinh dữ liệu WIN có seed + manifest (tenant A + canary B); nạp CSV / XLSX / JSON / tin nhắn / PDF văn bản → quan sát chuẩn / `Quarantine` | G1 |
| W2 ✓ | bí danh theo nguồn, mơ hồ, thời gian, các nghĩa nghiệp vụ; **`Outcome`** | G2 |
| W3 ✓ | thẩm quyền trong từng nghĩa, ca biên dung sai, câu xác nhận, thiếu thuật ngữ → hỏi lại | G3 |
| W4 ✓ | PG17 + migrations + RLS + store (SQL + danh mục + bằng chứng fact) + canary B + GUC tenant | G5 (phần dữ liệu) |
| W5a ✓ | bảng tài liệu / đoạn + ACL tài liệu + RLS; nạp SOP; retrieval | G5 (phần truy xuất) |
| W5b ✓ | gateway model (cấu hình + model giả); route `RAG` / `HYBRID_RETRIEVAL`; validator ràng buộc claim; ca injection | G4, G5 (phần model) |
| W6 | runner eval + baseline RAG + trace quyết định + kiểm phát lại | G6 ✓; G7 **CHẶN** (chưa có model thật) |
| W7 ✓ | chạy lại **cả 7 cổng** trên đường PG + scale smoke + báo cáo WIN sinh từ lệnh | WIN (trừ G7) |

Mỗi lát làm theo vòng: test đỏ → sửa đúng tầng → đột biến chứng minh test bắt được (`scripts/mutate.sh`).

## 9. Quyết định đã tự chốt (20/10/2026, user đảo được)

Chi tiết và lý do: `docs/adr/0001-win-mvp.md`.

| # | Câu | Chốt |
|---|---|---|
| D1 | Ngưỡng 7 cổng | như mục 6 |
| D2 | ERP mang nghĩa gì | nguồn hạng 2 của ISSUED (khớp pack); không tạo nghĩa riêng |
| D3 | "Dùng bao nhiêu dầu" | CONSUMED (`measured_consumption`) |
| D4 | Đọc PDF | PDF văn bản đọc bằng Go; Python / bảng PDF để sau WIN |
| D5 | Công thức G7 | như G7 mục 6 |
| D6 | Baseline | RAG đơn giản trong repo, chỉ dữ liệu tenant A, không có tầng kiểm soát EACL |
| D7–D17 | Quyết định triển khai khi làm W2–W7 (vật tư mặc định, ghép HYBRID, truy xuất, PG, ràng buộc claim, so sánh, câu nhiều mệnh đề, số âm…) | `docs/adr/0002-win-quyet-dinh-trien-khai.md` |
