# Sổ yêu cầu có truy vết của EAIS

> Tệp đọc cho người. **Nguồn thật là [catalog.yaml](catalog.yaml)** (một danh sách, mỗi mục một yêu cầu `REQ-<AREA>-NNN`). Mục đích duy nhất: không để mất logic của hai tệp ghi chú học và mười một brief — mỗi ý có giá trị trong nguồn là một yêu cầu có mã, gắn vào thành phần kiến trúc và cách kiểm.
>
> **Nguồn đã quét đủ (13/13)**: `z-luannt-learn-ai-with-quan-ai.v0.md` · `z-luannt-learn-ai-with-quan-ai.v1.md` · `brief-01-case-study-construction.md` · `brief-02-v2-two-tenants.md` · `brief-03-eais-naming.md` · `brief-04-rag-to-eais-expansion.md` · `brief-05-architecture-review.md` · `brief-06-model-vs-enterprise-system.md` · `brief-07-data-realism-proof.md` · `brief-08-rag-vs-eais-v0-v4.md` · `brief-09-blind-spot-audit-29.md` · `brief-10-working-hypothesis-first-principles.md` · `brief-11-claude-md-rationale.md`. **Chưa quét: không còn** (nguồn nào xuất hiện sau thời điểm này phải được thêm bằng REQ mới kèm cập nhật BẢNG PHỦ NGUỒN).

---

## 1. Cách dùng sổ và luật

### 1.1 Một mục gồm những gì

| Khoá | Ý nghĩa |
|---|---|
| `id` | `REQ-<AREA>-NNN`, 3 chữ số, không tái dùng số đã xoá (CONVENTIONS §3) |
| `area` | khớp tiền tố của `id`; danh sách AREA ở CONVENTIONS §3 — gồm `WFL` (workflow/state machine, lệnh, human task) và `PCK` (giới hạn và cấu trúc pack), hai AREA được thêm khi brief 05 mở rộng sang lệnh/sự kiện/workflow và giới hạn của pack |
| `statement` | một yêu cầu kiểm được, tiếng Việt; câu hỏi/checklist của nguồn được liệt kê đủ trong chính `statement` |
| `rationale` | vì sao (ngắn); tiền tố `[VIDEO]` = ý người nói nêu trực tiếp trong video, `[GIẢI THÍCH]`/`[HIỆU CHỈNH]` = nhãn của chính ghi chú nguồn |
| `sources` | vị trí trong tệp nguồn: `file`, `section` (tiêu đề chứa dòng đầu), `lines` (khoảng dòng) |
| `applies_to` | package theo `docs/01-architecture.md` §3, hoặc `tenants/<domain>`, `docs`, `evals` |
| `phase` | MVP · V1 · V2 · PROD · GATED · NA (xem 1.2) |
| `verification` | `kind` ∈ test · eval · review · none và `ref` mô tả cách kiểm; chỉ số chưa đo ghi **chưa đo** (CONVENTIONS §6) |
| `note` | bắt buộc khi `phase` = NA (vì sao không áp dụng) hoặc GATED (điều kiện mở cổng) |

### 1.2 Nghĩa của `phase`

| Phase | Nghĩa |
|---|---|
| `MVP` | luồng hỏi/nạp cơ bản cho 2 tenant, chuẩn hoá, router luật, context, kiểm số, eval thành phần; kèm engine workflow tối giản, pipeline lệnh C0–C7 và sự kiện E0–E6 (docs/01-architecture.md §2.1–§2.3, §4.1–§4.2) |
| `V1` | độ bền, quyền chi tiết, đa phiên bản tài liệu, skill nhiều bước, dịch vụ đọc tài liệu Python |
| `V2` | tối ưu, mở rộng domain (tenant thứ ba, nghiệp vụ lớp trên của sports) |
| `PROD` | SLA, vận hành, quy mô |
| `GATED` | capability chỉ mở khi có ca hỏng/số đo chứng minh (agent, MCP, rerank, classifier ý định bằng model, model-router, fine-tune, thị giác máy, hàng đợi/engine ngoài) |
| `NA` | không áp dụng cho nền tảng (lời khuyên nghề nghiệp, số liệu "5%/95%" cần kiểm chứng...) — vẫn có mục để không mất ý, kèm `note` |

### 1.3 Luật

1. **Mọi thay đổi kiến trúc phải trỏ về REQ.** ADR, PR đổi ranh giới giữa các module, hoặc capability mới đều nêu id REQ mà nó thoả mãn; chưa có REQ thì tạo REQ (kèm nguồn hoặc ca hỏng) trước. Không có REQ → chưa thêm (khớp ADR-0002 và luật core).
2. **REQ đổi `phase` phải ghi lý do** trong ADR hoặc commit và cập nhật `note`; hạ một REQ từ MVP xuống phase sau, hoặc mở cổng GATED, đều cần số đo hoặc ca hỏng đi kèm.
3. **Không tái dùng số.** REQ bị bác không xoá mà đổi `phase: NA` với `note` giải thích.
4. **Số liệu trong REQ** tuân CONVENTIONS §6: kết quả đo phải kèm lệnh tái lập, chưa đo ghi `chưa đo`; ví dụ/dữ liệu mẫu là giả lập.
5. **`docs/sources/` chỉ nằm ở máy** (CONVENTIONS §7); logic của chúng nằm ở đây. Khi sửa `sources` (khoảng dòng) của một REQ, phải cập nhật lại BẢNG PHỦ NGUỒN ở §3.
6. **Mỗi cổng GATED** có điều kiện mở đo được ghi ở `note`, ADR kèm số đo, và test âm chứng minh cổng chặn khi chưa đủ bằng chứng.

### 1.4 Kiểm tự động

```bash
python3 -c "import yaml;d=yaml.safe_load(open('docs/requirements/catalog.yaml'));print(len(d))"
grep -o 'id: REQ-[A-Z]*-[0-9]*' docs/requirements/catalog.yaml | sort | uniq -d   # phải rỗng
python3 - <<'PY'
import yaml
d = yaml.safe_load(open('docs/requirements/catalog.yaml'))
bad = [r['id'] for r in d if not r['id'].startswith('REQ-' + r['area'] + '-')]
bad += [r['id'] for r in d if not r['sources']]
bad += [r['id'] for r in d if r['phase'] in ('NA', 'GATED') and not r['note']]
print('lỗi:', bad)   # phải là []
PY
```

---

## 2. Bảng tóm tắt

### 2.1 Số REQ theo AREA và phase

| AREA | Tầng | MVP | V1 | V2 | PROD | GATED | NA | Tổng |
|---|---|--:|--:|--:|--:|--:|--:|--:|
| `MND` | tư duy nền (mindset) | 51 | 6 | 0 | 0 | 0 | 6 | **63** |
| `BIZ` | nghiệp vụ/giá trị | 16 | 1 | 3 | 0 | 1 | 1 | **22** |
| `DAT` | nền dữ liệu, kiểm kê | 7 | 5 | 0 | 0 | 0 | 0 | **12** |
| `DIG` | đọc hiểu tài liệu | 6 | 2 | 0 | 0 | 1 | 0 | **9** |
| `NRM` | chuẩn hoá | 16 | 1 | 0 | 0 | 0 | 0 | **17** |
| `ENT` | nhận dạng thực thể | 8 | 2 | 0 | 0 | 1 | 0 | **11** |
| `SOT` | nguồn sự thật | 11 | 2 | 0 | 0 | 0 | 0 | **13** |
| `RTE` | router | 17 | 0 | 0 | 0 | 1 | 0 | **18** |
| `CTX` | context engine | 13 | 2 | 0 | 0 | 0 | 0 | **15** |
| `RAG` | truy xuất tri thức | 16 | 5 | 1 | 1 | 3 | 0 | **26** |
| `MDL` | model gateway | 12 | 2 | 0 | 0 | 1 | 0 | **15** |
| `SLA` | cam kết dịch vụ | 0 | 0 | 0 | 5 | 0 | 1 | **6** |
| `TRN` | huấn luyện/fine-tune | 3 | 0 | 0 | 1 | 1 | 2 | **7** |
| `TOL` | tools | 9 | 4 | 0 | 0 | 0 | 0 | **13** |
| `SKL` | skills/workflow nhiều bước | 8 | 2 | 0 | 0 | 0 | 0 | **10** |
| `MCP` | MCP/tích hợp | 1 | 0 | 0 | 0 | 5 | 0 | **6** |
| `AGT` | agent | 0 | 0 | 0 | 0 | 6 | 0 | **6** |
| `MEM` | memory/state | 1 | 2 | 0 | 0 | 0 | 0 | **3** |
| `VAL` | kiểm đầu ra/độ tin cậy | 9 | 3 | 0 | 0 | 0 | 0 | **12** |
| `CLR` | hỏi lại/từ chối | 9 | 1 | 0 | 0 | 0 | 0 | **10** |
| `EVL` | đánh giá | 24 | 11 | 2 | 0 | 1 | 0 | **38** |
| `OBS` | quan sát/trace/chi phí | 4 | 8 | 0 | 2 | 0 | 0 | **14** |
| `SEC` | bảo mật/quản trị | 12 | 5 | 0 | 0 | 0 | 0 | **17** |
| `TEN` | đa tenant | 12 | 3 | 1 | 0 | 0 | 0 | **16** |
| `CV` | thị giác máy | 0 | 0 | 0 | 0 | 3 | 0 | **3** |
| `RT` | realtime/luồng sự kiện | 7 | 1 | 1 | 1 | 1 | 0 | **11** |
| `WFL` | workflow/state machine, lệnh, human task | 15 | 5 | 0 | 0 | 1 | 0 | **21** |
| `PCK` | giới hạn và cấu trúc pack | 7 | 1 | 0 | 0 | 0 | 0 | **8** |
| **Tổng** | | **294** | **74** | **8** | **10** | **26** | **10** | **422** |

### 2.2 Số REQ theo nguồn (một REQ có thể trích nhiều nguồn)

| Nguồn | Số REQ có trích nguồn này | Số mục tiêu đề trong bảng phủ |
|---|--:|--:|
| `z-luannt-learn-ai-with-quan-ai.v0.md` | 130 | 155 |
| `z-luannt-learn-ai-with-quan-ai.v1.md` | 218 | 225 |
| `brief-01-case-study-construction.md` | 68 | 22 |
| `brief-02-v2-two-tenants.md` | 123 | 37 |
| `brief-03-eais-naming.md` | 15 | 4 |
| `brief-04-rag-to-eais-expansion.md` | 31 | 18 |
| `brief-05-architecture-review.md` | 56 | 15 |
| `brief-06-model-vs-enterprise-system.md` | 30 | 13 |
| `brief-07-data-realism-proof.md` | 14 | 15 |
| `brief-08-rag-vs-eais-v0-v4.md` | 8 | 10 |
| `brief-09-blind-spot-audit-29.md` | 33 | 35 |
| `brief-10-working-hypothesis-first-principles.md` | 8 | 8 |
| `brief-11-claude-md-rationale.md` | 2 | 1 |

---

## 3. BẢNG PHỦ NGUỒN

Mỗi hàng là một mục tiêu đề `#`/`##` (và một số `###` có nội dung riêng, hoặc đoạn không tiêu đề) của tệp nguồn; cột cuối liệt kê mọi REQ có khoảng `lines` giao với phần nội dung của mục. Không có hàng nào trống: mục thuần dẫn nhập ghi "dẫn nhập — không có yêu cầu" kèm lý do; mục chỉ có tiêu đề nhóm gộp REQ của các mục con. Tên riêng của kênh/tác giả video trong cột tiêu đề được thay bằng "video nguồn"; số dòng vẫn đối chiếu được với tệp nguồn.

### 3.1 `z-luannt-learn-ai-with-quan-ai.v0.md` — Ghi chú học v0 — study guide 44 mục + ví dụ cầu đường

155 mục; 154 mục có REQ; 1 mục dẫn nhập.

| Dòng | Mục | REQ phủ |
|---|---|---|
| 1–83 | (mở đầu, chưa có tiêu đề) | REQ-MND-009, REQ-MND-010, REQ-MND-011, REQ-MND-013 |
| 84–85 | AI IS JUST ANOTHER TECH STACK | REQ-MND-010 *(tiêu đề nhóm, gộp từ mục con)* |
| 86–103 | › Study Guide & Enterprise AI Mindset Roadmap --- từ video nguồn | REQ-MND-010 |
| 104–153 | MỤC LỤC | dẫn nhập — không có yêu cầu: mục lục liệt kê tiêu đề 44 mục; nội dung từng mục được phủ ở hàng tương ứng bên dưới. |
| 154–200 | 1. CÁCH ĐỌC VIDEO NÀY | REQ-MND-003 |
| 201–202 | 2. THESIS TRUNG TÂM --- AI CHỈ LÀ MỘT TECH STACK | REQ-MND-004 |
| 203–246 | › Ý của tác giả | REQ-MND-004 |
| 247–256 | › Câu hỏi phải nhớ | REQ-MND-004 |
| 257–274 | 3. PHASE 0 --- TÁCH HYPE KHỎI ENGINEERING | REQ-MND-034, REQ-BIZ-001 |
| 275–287 | › Mental model | REQ-MND-034, REQ-BIZ-001 |
| 288–297 | › Keyword | REQ-MND-034, REQ-BIZ-001 |
| 298–303 | › Bài học | REQ-MND-034, REQ-BIZ-001 |
| 304–362 | 4. PHASE 1 --- TỪ CÔNG NGHỆ MỚI ĐẾN MỘT STACK ỔN ĐỊNH | REQ-MND-008 |
| 363–380 | › Tầm nhìn | REQ-MND-008 |
| 381–384 | 5. PHASE 2 --- DEMO AI VÀ ENTERPRISE AI LÀ HAI BÀI TOÁN KHÁC NHAU | REQ-MND-038, REQ-PCK-007, REQ-SEC-013 |
| 385–402 | › Non-critical AI | REQ-MND-038, REQ-PCK-007, REQ-SEC-013 |
| 403–427 | › Enterprise / critical AI | REQ-MND-038, REQ-PCK-007, REQ-SEC-013 |
| 428–439 | › Công thức | REQ-MND-038, REQ-PCK-007, REQ-SEC-013 |
| 440–483 | 6. PHASE 3 --- PROBLEM #1: GARBAGE IN, GARBAGE OUT | REQ-MND-005, REQ-DAT-003, REQ-DAT-005 |
| 484–498 | › Bài học | REQ-MND-005, REQ-DAT-003, REQ-DAT-005 |
| 499–526 | 7. PHASE 4 --- DATA AUDIT TRƯỚC AI | REQ-DAT-001, REQ-DAT-007, REQ-DIG-008, REQ-SOT-004, REQ-SOT-011 |
| 527–539 | › Câu hỏi | REQ-DAT-001, REQ-DAT-007, REQ-DIG-008, REQ-SOT-004, REQ-SOT-011 |
| 540–556 | › Keywords | REQ-DAT-001, REQ-DAT-007, REQ-DIG-008, REQ-SOT-004, REQ-SOT-011 |
| 557–588 | 8. PHASE 5 --- DATA DIGESTION / DOCUMENT UNDERSTANDING | REQ-DIG-001, REQ-DIG-003 |
| 589–612 | › Pipeline tốt hơn | REQ-DIG-001, REQ-DIG-002, REQ-DIG-004 |
| 613–634 | › Insight | REQ-DIG-004, REQ-DIG-005 |
| 635–664 | 9. PHASE 6 --- PROBLEM #2: NON-DETERMINISM | REQ-MND-005, REQ-MDL-005, REQ-EVL-013 |
| 665–677 | › Insight | REQ-MND-005, REQ-MDL-005, REQ-EVL-013 |
| 678–705 | 10. PHASE 7 --- DETERMINISTIC SHELL QUANH PROBABILISTIC CORE | REQ-MND-001 |
| 706–712 | › Principle | REQ-MND-001 |
| 713–736 | 11. PHASE 8 --- ROUTER / CLASSIFIER / WORKFLOW | REQ-RTE-001, REQ-RTE-003, REQ-SKL-009 |
| 737–754 | › Router hỏi | REQ-RTE-001, REQ-RTE-003 |
| 755–772 | › Roadmap con | REQ-RTE-001, REQ-RTE-003, REQ-RTE-007 |
| 773–795 | › Correction cần nhớ | REQ-RTE-001, REQ-RTE-003, REQ-RTE-006 |
| 796–809 | 12. PHASE 9 --- CONTEXT BUILDER | REQ-CTX-001, REQ-CTX-002 |
| 810–813 | › Câu hỏi | REQ-CTX-001, REQ-CTX-002 |
| 814–839 | › Context Builder | REQ-CTX-001, REQ-CTX-002, REQ-CTX-003 |
| 840–853 | › Context ≠ RAG | REQ-CTX-001, REQ-CTX-002 |
| 854–886 | 13. PHASE 10 --- VÍ DỤ CẦU ĐƯỜNG VÀ "4K LÍT DẦU" | REQ-NRM-001, REQ-NRM-002, REQ-NRM-003, REQ-NRM-005, REQ-NRM-011, REQ-CLR-008 |
| 887–912 | › Pipeline nên nghĩ tới | REQ-NRM-001, REQ-NRM-002, REQ-NRM-003, REQ-NRM-005, REQ-NRM-010, REQ-NRM-011, REQ-NRM-015, REQ-VAL-003, REQ-VAL-004, REQ-CLR-008 |
| 913–927 | › Keywords | REQ-NRM-001, REQ-NRM-002, REQ-NRM-003, REQ-NRM-005, REQ-NRM-010, REQ-NRM-011, REQ-NRM-015, REQ-VAL-003, REQ-VAL-004, REQ-CLR-008 |
| 928–954 | 14. PHASE 11 --- CLARIFICATION, AMBIGUITY VÀ MISSING/CONFLICTING DATA | REQ-CLR-001, REQ-CLR-002, REQ-CLR-005 |
| 955–961 | › Principle | REQ-CLR-001, REQ-CLR-003, REQ-CLR-005 |
| 962–981 | 15. PHASE 12 --- SLA VÀ TRÁCH NHIỆM TRONG ENTERPRISE | REQ-MND-038, REQ-SLA-001, REQ-SLA-002, REQ-SLA-003 |
| 982–994 | › Keywords | REQ-MND-038, REQ-SLA-001, REQ-SLA-002, REQ-SLA-003 |
| 995–1004 | › Mindset | REQ-MND-038, REQ-SLA-001, REQ-SLA-002, REQ-SLA-003 |
| 1005–1008 | 16. PHASE 13 --- PROBLEM #3: HALLUCINATION | REQ-MND-005, REQ-VAL-008 |
| 1009–1016 | › Không nên debug như sau | REQ-MND-005, REQ-VAL-008 |
| 1017–1033 | › Nên debug | REQ-MND-005, REQ-VAL-008 |
| 1034–1050 | › Mitigation map | REQ-MND-005, REQ-MDL-004, REQ-VAL-001, REQ-VAL-006, REQ-VAL-007, REQ-CLR-006 |
| 1051–1088 | 17. PHASE 14 --- CONTEXT, DOMAIN VÀ CÙNG MỘT TỪ NHƯNG KHÁC NGHĨA | REQ-ENT-002, REQ-ENT-003 |
| 1089–1102 | › Enterprise version | REQ-ENT-002, REQ-ENT-003 |
| 1103–1114 | › Questions | REQ-ENT-002, REQ-ENT-003 |
| 1115–1130 | › Keywords | REQ-ENT-002, REQ-ENT-003, REQ-ENT-004 |
| 1131–1155 | 18. PHASE 15 --- MODEL TRAINING, REWARD VÀ BENCHMARK | REQ-MND-007, REQ-TRN-006 |
| 1156–1172 | 19. PHASE 16 --- BENCHMARK ≠ PRODUCTION QUALITY | REQ-EVL-001, REQ-EVL-008, REQ-EVL-019 |
| 1173–1184 | › Cần phân biệt | REQ-EVL-001, REQ-EVL-008, REQ-EVL-019 |
| 1185–1197 | › Keywords | REQ-EVL-001, REQ-EVL-008, REQ-EVL-019 |
| 1198–1219 | 20. PHASE 17 --- KHI NON-DETERMINISM LÀ FEATURE VÀ KHI LÀ BUG | REQ-MDL-005, REQ-EVL-013 |
| 1220–1225 | › Decision question | REQ-MDL-005, REQ-EVL-013 |
| 1226–1270 | 21. PHASE 18 --- PHÂN LOẠI DỮ LIỆU TRƯỚC KHI CHỌN KỸ THUẬT | REQ-DAT-006 |
| 1271–1274 | 22. PHASE 19 --- FINE-TUNING VÀ RAG | REQ-TRN-001, REQ-TRN-002 |
| 1275–1286 | › Fine-tuning | REQ-TRN-001, REQ-TRN-002 |
| 1287–1296 | › RAG | REQ-TRN-001, REQ-TRN-002 |
| 1297–1317 | › First-principles distinction | REQ-TRN-001, REQ-TRN-002 |
| 1318–1345 | 23. PHASE 20 --- RAG DƯỚI GÓC NHÌN CỦA VIDEO | REQ-RTE-016 |
| 1346–1377 | › Production expansion | REQ-RAG-003, REQ-RAG-018 |
| 1378–1383 | › Principle | REQ-RAG-002 |
| 1384–1405 | 24. PHASE 21 --- SEMANTIC SEARCH | REQ-RAG-012, REQ-RAG-019 |
| 1406–1417 | › Important correction | REQ-RAG-012, REQ-RAG-019 |
| 1418–1430 | › Keywords | REQ-RAG-012, REQ-RAG-019 |
| 1431–1462 | 25. PHASE 22 --- MCP | REQ-MCP-001, REQ-MCP-004 |
| 1463–1481 | › Principle | REQ-MCP-001, REQ-MCP-004 |
| 1482–1508 | 26. PHASE 23 --- TOOLS | REQ-TOL-001 |
| 1509–1530 | › Production Tool Flow | REQ-TOL-002, REQ-TOL-004, REQ-TOL-010 |
| 1531–1563 | 27. PHASE 24 --- SKILLS | REQ-SKL-001 |
| 1564–1576 | › Keywords | REQ-SKL-001, REQ-SKL-006 |
| 1577–1618 | 28. PHASE 25 --- YOLO VÀ BÀI HỌC "MODEL TỐT ≠ SYSTEM TỐT" | REQ-CV-001 |
| 1619–1638 | › Insight | REQ-EVL-018, REQ-CV-001 |
| 1639–1653 | 29. PHASE 26 --- TRAINING INFRASTRUCTURE, GPU, COLAB, NỀN TẢNG GÁN NHÃN | REQ-TRN-007, REQ-CV-003 |
| 1654–1669 | › nền tảng gán nhãn example | REQ-TRN-007, REQ-CV-003 |
| 1670–1690 | › Bài học lớn | REQ-TRN-007, REQ-CV-003 |
| 1691–1708 | 30. PHASE 27 --- NHỮNG HƯỚNG MỚI TÁC GIẢ MUỐN DEVELOPER THEO DÕI | REQ-MND-008, REQ-MND-011 |
| 1709–1715 | › Principle | REQ-MND-008, REQ-MND-011 |
| 1716–1781 | 31. BỨC TRANH LỚN --- AI SYSTEM ARCHITECTURE RÚT RA TỪ VIDEO | REQ-MND-037, REQ-MND-052 |
| 1782–1783 | 32. ROADMAP LỚN VÀ CÁC ROADMAP CON | REQ-MND-021, REQ-MND-022 |
| 1784–1880 | › Master Roadmap | REQ-MND-021, REQ-MND-022, REQ-TOL-012 |
| 1881–1903 | 33. NHỮNG PHẦN HOÀN TOÀN KHÔNG PHẢI AI NHƯNG CỰC KỲ QUAN TRỌNG | REQ-MND-018, REQ-TOL-005, REQ-VAL-004, REQ-OBS-007, REQ-SEC-004, REQ-SEC-009 |
| 1904–1916 | › Core lesson | REQ-MND-018, REQ-TOL-005, REQ-VAL-004, REQ-OBS-007, REQ-SEC-004, REQ-SEC-009 |
| 1917–1952 | 34. PROBLEM MAP --- CÁC VẤN ĐỀ PHẢI NHỚ | REQ-MND-049 |
| 1953–1954 | 35. KEYWORD MAP --- TỪ KHÓA CẦN NGHIÊN CỨU SÂU | REQ-DAT-008, REQ-MND-050 |
| 1955–1958 | › Business | REQ-DAT-008, REQ-MND-050 |
| 1959–1964 | › Data | REQ-DAT-008, REQ-MND-050 |
| 1965–1969 | › Documents | REQ-MND-050 |
| 1970–1974 | › Language / Meaning | REQ-ENT-005, REQ-MND-050 |
| 1975–1979 | › Context | REQ-CTX-009, REQ-MND-050 |
| 1980–1984 | › Retrieval | REQ-RAG-012, REQ-MND-050 |
| 1985–1989 | › Routing | REQ-RTE-015, REQ-MND-050 |
| 1990–1994 | › Model | REQ-MND-050 |
| 1995–1999 | › Tools / Integration | REQ-MND-050 |
| 2000–2004 | › Reliability | REQ-MDL-008, REQ-VAL-009, REQ-MND-050 |
| 2005–2009 | › Evaluation | REQ-MND-050 |
| 2010–2015 | › Operations | REQ-MND-050 |
| 2016–2055 | 36. QUESTION MAP --- CÂU HỎI PHẢI BIẾT ĐẶT TRƯỚC KHI BUILD | REQ-MND-024, REQ-CTX-007 |
| 2056–2106 | 37. DEBUGGING FLOW --- KHI AI SYSTEM TRẢ LỜI SAI | REQ-MND-036, REQ-OBS-004 |
| 2107–2119 | › Principle | REQ-MND-036, REQ-OBS-004 |
| 2120–2177 | 38. FIRST-PRINCIPLES DECISION TREE | REQ-RTE-014, REQ-MDL-003, REQ-EVL-012 |
| 2178–2179 | 39. ANTI-PATTERNS MÀ VIDEO CẢNH BÁO | REQ-MND-035 |
| 2180–2189 | › Anti-pattern 1 | REQ-MND-035 |
| 2190–2199 | › Anti-pattern 2 | REQ-MND-035 |
| 2200–2211 | › Anti-pattern 3 | REQ-MND-035 |
| 2212–2221 | › Anti-pattern 4 | REQ-MND-035, REQ-VAL-001 |
| 2222–2229 | › Anti-pattern 5 | REQ-MND-035, REQ-EVL-008 |
| 2230–2237 | › Anti-pattern 6 | REQ-MND-035, REQ-MDL-010 |
| 2238–2247 | › Anti-pattern 7 | REQ-MND-035 |
| 2248–2259 | › Anti-pattern 8 | REQ-MND-035 |
| 2260–2299 | 40. GIÁ TRỊ THẬT SỰ CỦA VIDEO | REQ-MND-037 |
| 2300–2304 | 41. NHỮNG CHỖ NÊN HIỂU LÀ MENTAL MODEL, KHÔNG PHẢI ĐỊNH NGHĨA KỸ THUẬT TUYỆT ĐỐI | REQ-MND-009 |
| 2305–2308 | › "AI chỉ là statistical model" | REQ-MND-009 |
| 2309–2312 | › Hallucination và MoE routing | REQ-MND-009, REQ-VAL-007 |
| 2313–2316 | › Semantic similarity kiểu "95% true" | REQ-MND-009, REQ-RAG-019 |
| 2317–2321 | › Router làm AI deterministic | REQ-MND-009, REQ-RTE-006 |
| 2322–2326 | › Các số 5%, 95% | REQ-MND-009, REQ-MND-010 |
| 2327–2332 | › Các tên "Jeff/Jev", "Tinker/Thinker" | REQ-MND-009, REQ-MND-011 |
| 2333–2334 | 42. CHECKLIST HỌC LẠI VIDEO | REQ-MND-047 |
| 2335–2342 | › Level 1 --- Mental Model | REQ-MND-047 |
| 2343–2352 | › Level 2 --- Data | REQ-MND-047 |
| 2353–2361 | › Level 3 --- Determinism | REQ-MND-047 |
| 2362–2370 | › Level 4 --- Context | REQ-MND-047 |
| 2371–2378 | › Level 5 --- Retrieval | REQ-MND-047 |
| 2379–2387 | › Level 6 --- Application Architecture | REQ-MND-047 |
| 2388–2397 | › Level 7 --- Model | REQ-MND-047 |
| 2398–2408 | › Level 8 --- Production | REQ-EVL-018, REQ-TRN-005, REQ-MND-047 |
| 2409–2418 | › Level 9 --- Domain | REQ-EVL-018, REQ-TRN-005, REQ-MND-047 |
| 2419–2482 | 43. ONE-PAGE CHEAT SHEET | REQ-MND-037, REQ-MND-052 |
| 2483–2490 | › 3 problems video nhấn mạnh | REQ-MND-005 |
| 2491–2503 | › 3 phản xạ engineering | REQ-MND-006 |
| 2504–2515 | › 1 principle lớn nhất | REQ-MND-001, REQ-MND-006 |
| 2516–2578 | 44. KẾT LUẬN | REQ-MND-018, REQ-MND-037 |
| 2579–2618 | › Tầm nhìn cuối cùng | REQ-MND-013 |
| 2619–2622 | › STUDY LOOP ĐỀ XUẤT | REQ-MND-048 |
| 2623–2631 | › Vòng 1 --- Mindset | REQ-MND-048 |
| 2632–2643 | › Vòng 2 --- Architecture | REQ-MND-048 |
| 2644–2654 | › Vòng 3 --- Reliability | REQ-MND-048 |
| 2655–2678 | › Vòng 4 --- First Principles | REQ-BIZ-014, REQ-MND-048 |
| 2679–2680 | (không tiêu đề) mở đầu phần ví dụ cầu đường — dòng 2679-2680 | REQ-BIZ-018 |
| 2681–2724 | › 1. Hãy biến toàn bộ video thành một case study thật | REQ-BIZ-018 |
| 2725–2781 | 2. Làm thật thì hình dạng hệ thống thay đổi hoàn toàn | REQ-BIZ-018 |
| 2782–2828 | 3. Ví dụ: “Hôm nay công trường A tiêu thụ bao nhiêu dầu?” | REQ-RTE-011, REQ-CTX-001, REQ-CTX-013, REQ-MDL-002 |
| 2829–2867 | 4. Nhưng hỏi câu khác thì RAG xuất hiện | REQ-RTE-010 |
| 2868–2913 | 5. Và câu hỏi của bạn về multi-tenant mới rất đáng đào sâu | REQ-TEN-007 |
| 2914–2977 | 6. Shared cái gì, custom cái gì? | REQ-TEN-005, REQ-TEN-007 |
| 2978–3045 | 7. Ví dụ “4k lít” cho thấy tại sao tenant/domain customization tồn tại | REQ-NRM-006 |
| 3046–3122 | 8. Vì vậy tôi sẽ sửa nhẹ cái hình “6 vệ tinh” ban đầu | REQ-MND-020 |
| 3123–3151 | › Và RAG nằm ở đâu? | REQ-BIZ-014, REQ-RAG-001 |

### 3.2 `z-luannt-learn-ai-with-quan-ai.v1.md` — Ghi chú học v1 — transcript thô + workshop + roadmap + problem map

225 mục; 225 mục có REQ; 0 mục dẫn nhập.

| Dòng | Mục | REQ phủ |
|---|---|---|
| 1–7 | [Transcript thô] "data raww video" — dòng 2-4 (nguồn của các ý [VIDEO]) | REQ-MND-004, REQ-MND-005, REQ-MND-008, REQ-MND-010, REQ-MND-012, REQ-MND-034, REQ-MND-039, REQ-BIZ-001, REQ-DAT-005, REQ-DAT-006, REQ-DIG-003, REQ-NRM-002, REQ-RTE-016, REQ-CTX-002, REQ-CTX-013, REQ-RAG-019, REQ-RAG-024, REQ-MDL-005, REQ-MDL-010, REQ-SKL-009, REQ-PCK-007, REQ-CLR-001, REQ-CLR-005, REQ-EVL-013, REQ-SLA-001, REQ-SLA-002, REQ-SLA-003, REQ-TRN-006, REQ-TRN-007, REQ-CV-001, REQ-BIZ-020 |
| 8–27 | (không tiêu đề) mở đầu hình "Core AI model + satellite systems" — dòng 8-27 | REQ-MND-020 |
| 28–56 | › 1. AI Model — “bộ não” nhưng không phải toàn bộ hệ thống | REQ-MND-007, REQ-MDL-012 |
| 57–108 | 2. Router — “đi đâu để giải quyết request này?” | REQ-RTE-003, REQ-RTE-011 |
| 109–174 | 3. RAG — “model cần biết thêm thông tin gì?” | REQ-CTX-001, REQ-RAG-002 |
| 175–242 | 4. Tools — “model cần thực hiện hành động gì?” | REQ-TOL-003, REQ-TOL-005 |
| 243–298 | 5. MCP — không phải Tool; nó là “chuẩn kết nối” | REQ-MCP-001, REQ-MCP-004 |
| 299–356 | 6. Skills — “model nên làm công việc này như thế nào?” | REQ-SKL-001 |
| 357–362 | 7. Bây giờ ghép cả 5 satellite systems | REQ-SKL-010 |
| 363–370 | › Router | REQ-SKL-010 |
| 371–384 | › Skill | REQ-SKL-010 |
| 385–395 | › Tools | REQ-SKL-010 |
| 396–406 | › MCP | REQ-MND-002, REQ-SKL-010 |
| 407–418 | › RAG | REQ-MND-002, REQ-SKL-010 |
| 419–460 | › AI Model | REQ-MND-002, REQ-SKL-010 |
| 461–504 | 8. Còn “Recursive training” trong ảnh là gì? | REQ-OBS-009, REQ-TRN-004 |
| 505–584 | 9. Nếu sửa hình này thành kiến trúc kỹ thuật chính xác hơn | REQ-MND-002 |
| 585–588 | (không tiêu đề) mở đầu ghi chú workshop — dòng 585-588 | REQ-MND-009 |
| 589–663 | › Trả lời thẳng 3 điểm bạn đang thắc mắc | REQ-MND-011, REQ-MND-020, REQ-RTE-006 |
| 664–719 | › Video có chất lượng không? | REQ-MND-002, REQ-MND-003, REQ-MND-010, REQ-MND-051 |
| 720–721 | AI IS JUST ANOTHER TECH STACK — Workshop Notes & Engineering Checklist | REQ-MND-009 |
| 722–733 | › 0. Mục tiêu của tài liệu | REQ-MND-009 |
| 734–735 | 1. Thesis lớn nhất của video | REQ-MND-003 |
| 736–762 | › AI không phải phép thuật; AI là một thành phần của software system | REQ-MND-003 |
| 763–784 | 2. “Core AI model + satellite systems” nghĩa là gì? | REQ-MND-020 |
| 785–819 | › Đừng hiểu satellite system là một chuẩn kiến trúc chính thức | REQ-MND-020 |
| 820–821 | 3. Router là gì? | REQ-RTE-003, REQ-RTE-011, REQ-SKL-009 |
| 822–847 | › Câu hỏi Router giải quyết | REQ-RTE-003, REQ-RTE-011, REQ-SKL-009 |
| 848–859 | › Router không nhất thiết phải là LLM | REQ-RTE-003 |
| 860–891 | › “Đưa non-deterministic về deterministic” phải hiểu thế nào? | REQ-RTE-003, REQ-RTE-006, REQ-MDL-002 |
| 892–893 | 4. Context Builder là gì? | REQ-NRM-013, REQ-CTX-002 |
| 894–929 | › Câu hỏi Context Builder giải quyết | REQ-NRM-013, REQ-CTX-002 |
| 930–951 | › Vì sao tác giả nói Context Builder “không liên quan AI nhiều”? | REQ-NRM-013, REQ-CTX-002 |
| 952–955 | 5. Tại sao học RAG lại phải hiểu Context Builder? | REQ-CTX-001 |
| 956–1001 | › RAG chỉ là một cách lấy context | REQ-CTX-001 |
| 1002–1027 | 6. Garbage In, Garbage Out | REQ-DAT-001, REQ-DAT-005, REQ-SOT-001 |
| 1028–1042 | › Checklist trước khi làm AI | REQ-DAT-001, REQ-SOT-001 |
| 1043–1093 | 7. Data digestion / document understanding | REQ-DIG-001, REQ-DIG-002, REQ-DIG-003, REQ-DIG-004, REQ-DIG-005 |
| 1094–1142 | 8. Ví dụ 4k lít dầu — bài học về deterministic boundary | REQ-NRM-001, REQ-NRM-011, REQ-VAL-003 |
| 1143–1160 | 9. Non-determinism | REQ-MND-001, REQ-VAL-001 |
| 1161–1184 | › Pattern nên dùng | REQ-MND-001, REQ-VAL-001 |
| 1185–1216 | 10. Hallucination | REQ-VAL-007 |
| 1217–1254 | 11. RAG | REQ-RAG-003 |
| 1255–1267 | › RAG không đồng nghĩa Vector DB | REQ-RAG-002 |
| 1268–1290 | 12. Semantic Search | REQ-RAG-012, REQ-RAG-019 |
| 1291–1320 | 13. Tools | REQ-TOL-003 |
| 1321–1345 | 14. Skills | REQ-SKL-001 |
| 1346–1370 | 15. MCP | REQ-MCP-001 |
| 1371–1374 | 16. Fine-tuning vs RAG | REQ-TRN-002 |
| 1375–1385 | › Fine-tuning | REQ-TRN-002 |
| 1386–1400 | › RAG | REQ-TRN-002 |
| 1401–1443 | 17. Training và benchmark | REQ-EVL-001, REQ-EVL-005, REQ-EVL-007, REQ-EVL-008 |
| 1444–1477 | 18. Evaluation — phần nên bổ sung mạnh hơn video | REQ-EVL-010, REQ-EVL-011 |
| 1478–1510 | 19. “Recursive training” trong slide | REQ-OBS-009, REQ-TRN-004 |
| 1511–1533 | 20. YOLO / Computer Vision — lesson tổng quát | REQ-EVL-018, REQ-CV-001 |
| 1534–1535 | 21. Enterprise AI: demo vs production | REQ-MND-038 |
| 1536–1541 | › Demo | REQ-MND-038 |
| 1542–1575 | › Production | REQ-MND-038 |
| 1576–1599 | 22. SLA / reliability | REQ-EVL-017, REQ-SLA-001 |
| 1600–1611 | 23. Claim “chỉ 5% enterprise AI tạo giá trị” | REQ-MND-010 |
| 1612–1613 | 24. Những điểm video làm tốt | REQ-MND-009 |
| 1614–1625 | › Rất tốt | REQ-MND-009 |
| 1626–1634 | › Đúng hướng nhưng đơn giản hóa | REQ-MND-009 |
| 1635–1642 | › Claim cần nguồn trước khi coi là fact | REQ-MND-009, REQ-MND-010, REQ-MND-011, REQ-MND-012 |
| 1643–1644 | 25. Đánh giá chất lượng video | REQ-MND-009, REQ-MND-051 |
| 1645–1658 | › Nếu xem như workshop/overview cho developer: 8/10 | REQ-MND-009, REQ-MND-051 |
| 1659–1674 | › Nếu xem như tài liệu kỹ thuật chính xác để học định nghĩa: khoảng 6.5–7/10 | REQ-MND-009, REQ-MND-051 |
| 1675–1720 | 26. Bức tranh AI Engineering nên rút ra | REQ-MND-020, REQ-MND-052 |
| 1721–1722 | 27. Checklist học tập sau video | REQ-MND-047 |
| 1723–1731 | › Level 1 — Foundation | REQ-MND-047 |
| 1732–1741 | › Level 2 — Data | REQ-MND-047 |
| 1742–1752 | › Level 3 — Retrieval / Context | REQ-MND-047 |
| 1753–1764 | › Level 4 — Agent/Application architecture | REQ-MND-047 |
| 1765–1775 | › Level 5 — Reliability | REQ-MND-047 |
| 1776–1788 | › Level 6 — Evaluation | REQ-MND-047 |
| 1789–1800 | › Level 7 — Model adaptation | REQ-MDL-011, REQ-EVL-019, REQ-TRN-001, REQ-MND-047 |
| 1801–1833 | 28. First-principles checklist khi có người nói “tích hợp AI” | REQ-MND-025 |
| 1834–1856 | 29. Một câu để nhớ toàn bộ workshop | REQ-MND-002 |
| 1857–1877 | (không tiêu đề) capability map 6 workstream — dòng 1857-1917 | REQ-MND-021 |
| 1878–1917 | › Tại sao không nên gọi chúng là 6 “phase”? | REQ-MND-021, REQ-RTE-008 |
| 1918–2058 | › Ví dụ Router tự nó đã là một roadmap nhỏ | REQ-RTE-005, REQ-RTE-007, REQ-RAG-023, REQ-MDL-007, REQ-MDL-008, REQ-MDL-009, REQ-TOL-012, REQ-SKL-007, REQ-SKL-008, REQ-MCP-005, REQ-TRN-003, REQ-MDL-013 |
| 2059–2158 | › Nhưng bức tranh lớn còn thiếu | REQ-MND-020, REQ-MND-021, REQ-MEM-001 |
| 2159–2176 | (không tiêu đề) mở đầu audit RAG — dòng 2159-2176 | REQ-RAG-001 |
| 2177–2241 | 1. Nếu “zoom” vào vòng tròn RAG, hình dạng thật của nó là gì? | REQ-RAG-001 |
| 2242–2299 | 2. Vậy RAG có phải một “roadmap con” không? | REQ-RAG-001, REQ-RAG-023 |
| 2300–2327 | 3. Raw video của bạn thực tế đang dạy roadmap nào? | REQ-RAG-023 |
| 2328–2367 | 4. PHASE 0 — RAG Mental Model | REQ-RAG-002 |
| 2368–2375 | › Retrieval | REQ-RAG-002 |
| 2376–2387 | › Augmentation | REQ-RAG-002 |
| 2388–2401 | › Generation | REQ-RAG-002 |
| 2402–2468 | 5. PHASE 1 — Knowledge / Indexing Pipeline | REQ-RAG-005 |
| 2469–2535 | 6. Chunking tự nó lại là một roadmap | REQ-RAG-005, REQ-RAG-006 |
| 2536–2588 | 7. PHASE 2 — tác giả chuyển sang “Failure-first RAG” | REQ-RAG-016 |
| 2589–2636 | 8. Nhưng tôi sẽ sửa taxonomy của video | REQ-CTX-011, REQ-CTX-015, REQ-RAG-017 |
| 2637–2684 | 9. PHASE 3 — Retrieval Quality | REQ-RAG-012, REQ-RAG-014 |
| 2685–2719 | 10. Retrieval bản thân nó cũng là một subsystem | REQ-RAG-013 |
| 2720–2781 | 11. Và đây là chỗ RAG chạm vào Context Builder | REQ-CTX-001, REQ-CTX-005, REQ-CTX-006, REQ-CTX-008, REQ-CTX-009, REQ-CTX-011, REQ-RAG-013 |
| 2782–2832 | 12. PHASE 4 — Production Engineering | REQ-RAG-008, REQ-RAG-009, REQ-RAG-010, REQ-RAG-011 |
| 2833–2890 | 13. PHASE 5 — Evaluation | REQ-RAG-008, REQ-VAL-006, REQ-EVL-006, REQ-EVL-009, REQ-EVL-010 |
| 2891–2915 | 14. PHASE 6 — Advanced RAG | REQ-MND-027, REQ-MND-030, REQ-RAG-020, REQ-RAG-021, REQ-AGT-005 |
| 2916–2970 | 15. Chỗ tôi sẽ “audit ngược” tác giả | REQ-MND-027, REQ-MND-030, REQ-RAG-020, REQ-RAG-021, REQ-AGT-005 |
| 2971–3009 | 16. Ví dụ Graph RAG | REQ-MND-027, REQ-RAG-021 |
| 3010–3046 | 17. Multimodal cũng vậy | REQ-MND-027, REQ-DIG-009 |
| 3047–3143 | 18. Đây mới là roadmap RAG tôi sẽ rút ra từ file | REQ-RAG-006, REQ-RAG-007, REQ-RAG-014, REQ-RAG-015, REQ-RAG-018, REQ-RAG-020, REQ-RAG-023, REQ-MDL-004, REQ-VAL-006, REQ-CLR-003 |
| 3144–3205 | 19. Và bây giờ ghép ngược lại vào video nguồn | REQ-RAG-001 |
| 3206–3269 | 20. Và đây là connection quan trọng nhất giữa hai video | REQ-MND-008, REQ-RAG-023 |
| 3270–3283 | (không tiêu đề) mở đầu audit context/7 subsystem — dòng 3270-3283 | REQ-MND-020, REQ-MND-053 |
| 3284–3344 | › 1. Bức tranh nên nâng cấp thành thế này | REQ-MND-020, REQ-CTX-004, REQ-MCP-006, REQ-MEM-001, REQ-MEM-002, REQ-MND-053 |
| 3345–3381 | 2. Có 7 subsystem chính quanh Model đáng mổ | REQ-MND-020, REQ-MND-053 |
| 3382–3404 | 3. CONTEXT ENGINEERING — thực ra đây mới có thể là “trung tâm” | REQ-CTX-014 |
| 3405–3460 | › Context của một request có thể là | REQ-CTX-003, REQ-CTX-004, REQ-CTX-009, REQ-CTX-014 |
| 3461–3504 | 4. Context Builder chính là “assembly line” | REQ-CTX-001, REQ-CTX-005, REQ-CTX-010 |
| 3505–3588 | 5. ROUTER — “traffic controller” | REQ-RTE-003, REQ-RTE-005, REQ-RTE-011, REQ-MDL-003 |
| 3589–3628 | 6. Router production thực ra lại bung thành cả subsystem | REQ-RTE-008, REQ-RTE-009, REQ-SEC-012 |
| 3629–3702 | 7. TOOLS — “hands” của model | REQ-TOL-002, REQ-TOL-003, REQ-TOL-011 |
| 3703–3774 | 8. Và Tool cần được audit production riêng | REQ-TOL-001, REQ-TOL-002, REQ-TOL-004, REQ-TOL-010, REQ-TOL-011, REQ-WFL-010, REQ-SEC-009 |
| 3775–3819 | 9. MCP — cái này hay bị hiểu nhầm nhất | REQ-MCP-001, REQ-MCP-004 |
| 3820–3873 | 10. MCP vs Tool | REQ-MCP-001 |
| 3874–3932 | 11. SKILL — lại là một thứ hoàn toàn khác | REQ-SKL-001, REQ-MCP-006 |
| 3933–3980 | 12. Skill production cũng có lifecycle | REQ-SKL-002, REQ-SKL-006 |
| 3981–4046 | 13. MEMORY / STATE — component còn thiếu nữa | REQ-MEM-001, REQ-MEM-002 |
| 4047–4103 | 14. Ghép lại thì bắt đầu thành Agent | REQ-AGT-001 |
| 4104–4147 | 15. Như vậy Agent không phải thêm một “magic component” | REQ-AGT-001 |
| 4148–4196 | 16. RAG nằm ở đâu trong Agent? | REQ-AGT-001, REQ-AGT-005 |
| 4197–4230 | 17. Một correction nữa: Context có thể quan trọng hơn Model | REQ-CTX-015 |
| 4231–4287 | 18. Context thực tế là “bus” nối mọi thứ vào model | REQ-CTX-001 |
| 4288–4324 | 19. Production architecture phải thêm deterministic shell | REQ-MND-001, REQ-SEC-011, REQ-SEC-012 |
| 4325–4359 | 20. Và vẫn còn một tầng chúng ta chưa audit: Control Plane | REQ-TOL-003, REQ-OBS-002, REQ-OBS-007, REQ-OBS-008, REQ-SEC-010 |
| 4360–4401 | 21. Nếu audit toàn bộ theo First Principles, tôi sẽ chia thành 4 plane | REQ-MND-020 |
| 4402–4422 | 22. Và mỗi subsystem lại có câu hỏi First Principles riêng | REQ-MND-023 |
| 4423–4464 | 23. Và khi system trả lời sai, đừng lập tức blame Model | REQ-MND-036, REQ-MEM-003, REQ-VAL-008, REQ-OBS-006 |
| 4465–4570 | 24. Từ đây roadmap audit của chúng ta đã khá rõ | REQ-MND-046 |
| 4571–4574 | (không tiêu đề) mở đầu master roadmap — dòng 4571-4574 | REQ-MND-022 |
| 4575–4677 | MASTER ROADMAP — AI SYSTEMS ENGINEERING | REQ-MND-022, REQ-OBS-009 |
| 4678–4733 | ROADMAP 0 — BUSINESS PROBLEM: trước cả AI | REQ-BIZ-001 |
| 4734–4789 | ROADMAP 1 — DATA FOUNDATION | REQ-MND-039, REQ-DAT-001, REQ-DAT-006, REQ-DAT-007 |
| 4790–4857 | ROADMAP 2 — DATA DIGESTION / DOCUMENT INTELLIGENCE | REQ-DIG-001, REQ-DIG-002, REQ-DIG-005 |
| 4858–4910 | ROADMAP 3 — CONTEXT ENGINEERING | REQ-CTX-005, REQ-CTX-010 |
| 4911–4994 | ROADMAP 4 — RAG | REQ-RAG-003, REQ-RAG-004, REQ-RAG-015 |
| 4995–5040 | ROADMAP 5 — SEARCH / RETRIEVAL | REQ-RAG-002 |
| 5041–5122 | ROADMAP 6 — ROUTER | REQ-RTE-001, REQ-RTE-006, REQ-RTE-007, REQ-RTE-008, REQ-RTE-009, REQ-RTE-015 |
| 5123–5181 | ROADMAP 7 — DETERMINISTIC ENGINEERING | REQ-NRM-001, REQ-SKL-005, REQ-CLR-002 |
| 5182–5246 | ROADMAP 8 — TOOLS | REQ-TOL-001, REQ-TOL-008, REQ-TOL-010, REQ-TOL-012 |
| 5247–5312 | ROADMAP 9 — SKILLS | REQ-SKL-002, REQ-SKL-006, REQ-SKL-007, REQ-SKL-008 |
| 5313–5391 | ROADMAP 10 — MCP / INTEGRATION | REQ-MCP-001, REQ-MCP-002, REQ-MCP-003, REQ-MCP-005 |
| 5392–5442 | ROADMAP 11 — MODEL | REQ-MDL-001, REQ-MDL-006, REQ-MDL-010, REQ-MDL-011, REQ-MDL-013 |
| 5443–5476 | ROADMAP 12 — FINE-TUNING / TRAINING | REQ-TRN-001, REQ-TRN-002, REQ-TRN-003 |
| 5477–5540 | ROADMAP 13 — COMPUTER VISION / YOLO | REQ-CV-001, REQ-CV-002, REQ-CV-003 |
| 5541–5594 | ROADMAP 14 — RELIABILITY | REQ-TOL-008, REQ-TOL-009, REQ-WFL-007, REQ-WFL-008, REQ-PCK-007, REQ-VAL-002, REQ-VAL-009, REQ-VAL-010, REQ-CLR-006 |
| 5595–5637 | ROADMAP 15 — EVALUATION | REQ-EVL-001, REQ-EVL-005, REQ-EVL-007, REQ-EVL-010, REQ-EVL-021 |
| 5638–5688 | ROADMAP 16 — OBSERVABILITY | REQ-MEM-003, REQ-OBS-002, REQ-OBS-003, REQ-OBS-006, REQ-OBS-007 |
| 5689–5724 | ROADMAP 17 — SLA / ENTERPRISE CONTRACT | REQ-OBS-011, REQ-SLA-001, REQ-SLA-002, REQ-SLA-004 |
| 5725–5844 | Và đây mới là ROADMAP LỚN cuối cùng | REQ-MND-020, REQ-MND-022 |
| 5845–5902 | › Quan trọng nhất: đừng học roadmap này như danh sách công nghệ | REQ-MND-018, REQ-RTE-013 |
| 5903–5939 | › Nếu nén toàn bộ video xuống một công thức | REQ-MND-003 |
| 5940–5946 | (không tiêu đề) mở đầu Problem Map — dòng 5940-5946 | REQ-BIZ-019 |
| 5947–5947 | ENTERPRISE AI PROBLEM MAP | REQ-BIZ-019 |
| 5948–5976 | › Keyword + Question Roadmap extracted from the workshop nguồn | REQ-BIZ-001, REQ-BIZ-019 |
| 5977–5979 | PART I — AI ≠ PRODUCT | REQ-BIZ-001 |
| 5980–6013 | › 01. AI có thật sự cần thiết không? | REQ-BIZ-001, REQ-RTE-013 |
| 6014–6016 | PART II — DETERMINISTIC vs PROBABILISTIC | REQ-RTE-013 |
| 6017–6045 | › 02. Phần nào phải deterministic? | REQ-RTE-013 |
| 6046–6068 | › 03. Phần nào nên probabilistic? | REQ-RTE-013 |
| 6069–6071 | PART III — LANGUAGE ≠ MEANING | REQ-ENT-002, REQ-ENT-003, REQ-ENT-005, REQ-CLR-001 *(tiêu đề nhóm, gộp từ mục con)* |
| 6072–6146 | › 04. Cùng một từ nhưng context khác → nghĩa khác | REQ-ENT-002, REQ-ENT-003, REQ-ENT-005, REQ-CLR-001 |
| 6147–6149 | PART IV — AMBIGUITY MANAGEMENT | REQ-CLR-001 |
| 6150–6201 | › 05. AI không hiểu chắc thì làm gì? | REQ-CLR-001, REQ-CLR-003, REQ-CLR-004, REQ-CLR-005 |
| 6202–6204 | PART V — GARBAGE IN, GARBAGE OUT | REQ-DAT-003, REQ-DAT-004, REQ-DAT-007, REQ-DAT-008 *(tiêu đề nhóm, gộp từ mục con)* |
| 6205–6238 | › 06. Data Quality | REQ-DAT-003, REQ-DAT-004, REQ-DAT-007, REQ-DAT-008 |
| 6239–6241 | PART VI — NORMALIZATION | REQ-NRM-001, REQ-NRM-002, REQ-NRM-015, REQ-MDL-004 *(tiêu đề nhóm, gộp từ mục con)* |
| 6242–6287 | › 07. Nhiều representation → một business value | REQ-NRM-001, REQ-NRM-002, REQ-NRM-015, REQ-MDL-004 |
| 6288–6290 | PART VII — STRUCTURED EXTRACTION | REQ-NRM-015, REQ-MDL-004 |
| 6291–6332 | › 08. Natural Language → Typed Data | REQ-NRM-015, REQ-MDL-004, REQ-VAL-003, REQ-VAL-004 |
| 6333–6335 | PART VIII — BUSINESS VALIDATION | REQ-VAL-003, REQ-VAL-004 |
| 6336–6375 | › 09. Syntactically valid ≠ Business valid | REQ-SOT-001, REQ-SOT-004, REQ-SOT-005, REQ-SOT-011, REQ-VAL-003, REQ-VAL-004, REQ-VAL-005 |
| 6376–6378 | PART IX — SOURCE OF TRUTH | REQ-SOT-001, REQ-SOT-004, REQ-SOT-005, REQ-SOT-011 |
| 6379–6418 | › 10. Khi nhiều nguồn nói khác nhau | REQ-DIG-001, REQ-SOT-001, REQ-SOT-002, REQ-SOT-004, REQ-SOT-005, REQ-SOT-011 |
| 6419–6421 | PART X — DOCUMENT INTELLIGENCE | REQ-DIG-001 |
| 6422–6484 | › 11. PDF không phải chỉ là text | REQ-DIG-001, REQ-DIG-002, REQ-DIG-003, REQ-DIG-004, REQ-DIG-009, REQ-RAG-012, REQ-RAG-024 |
| 6485–6487 | PART XI — RETRIEVAL | REQ-RAG-012, REQ-RAG-024 |
| 6488–6518 | › 12. Tìm đúng evidence | REQ-RAG-012, REQ-RAG-024 |
| 6519–6521 | PART XII — RAG | REQ-CTX-001, REQ-RAG-002, REQ-RAG-003, REQ-RAG-004, REQ-RAG-007, REQ-SEC-007 *(tiêu đề nhóm, gộp từ mục con)* |
| 6522–6590 | › 13. RAG không phải Vector DB | REQ-CTX-001, REQ-RAG-002, REQ-RAG-003, REQ-RAG-004, REQ-RAG-007, REQ-SEC-007 |
| 6591–6593 | PART XIII — CONTEXT ENGINEERING | REQ-CTX-001 |
| 6594–6636 | › 14. Model nên nhìn thấy cái gì? | REQ-CTX-001, REQ-CTX-003, REQ-CTX-007, REQ-CTX-008, REQ-CTX-009 |
| 6637–6639 | PART XIV — ROUTING | REQ-RTE-015, REQ-RTE-016, REQ-VAL-008 *(tiêu đề nhóm, gộp từ mục con)* |
| 6640–6681 | › 15. Request này phải đi đâu? | REQ-RTE-015, REQ-RTE-016, REQ-VAL-008 |
| 6682–6684 | PART XV — HALLUCINATION | REQ-VAL-008 |
| 6685–6719 | › 16. "Model nói sai" chưa phải root cause | REQ-TOL-001, REQ-VAL-006, REQ-VAL-008 |
| 6720–6722 | PART XVI — TOOLS | REQ-TOL-001 |
| 6723–6741 | › 17. Model cần hành động | REQ-TOL-001, REQ-SKL-001 |
| 6742–6744 | PART XVII — SKILLS | REQ-SKL-001 |
| 6745–6766 | › 18. AI cần procedure | REQ-SKL-001, REQ-MCP-001 |
| 6767–6769 | PART XVIII — MCP / INTEGRATION | REQ-MCP-001 |
| 6770–6787 | › 19. AI kết nối external systems như thế nào? | REQ-MCP-001 |
| 6788–6790 | PART XIX — NON-DETERMINISM | REQ-MDL-004, REQ-MDL-005, REQ-VAL-007 *(tiêu đề nhóm, gộp từ mục con)* |
| 6791–6813 | › 20. Một input → nhiều output | REQ-MDL-004, REQ-MDL-005, REQ-VAL-007 |
| 6814–6816 | PART XX — HALLUCINATION CONTROL | REQ-VAL-007 |
| 6817–6846 | › 21. Probabilistic core phải được bao quanh | REQ-MDL-001, REQ-MDL-006, REQ-MDL-007, REQ-MDL-009, REQ-MDL-010, REQ-VAL-007 |
| 6847–6849 | PART XXI — MODEL SELECTION | REQ-MDL-001, REQ-MDL-006, REQ-MDL-007, REQ-MDL-009, REQ-MDL-010 |
| 6850–6875 | › 22. Không phải task nào cũng cần model mạnh nhất | REQ-MDL-001, REQ-MDL-006, REQ-MDL-007, REQ-MDL-009, REQ-MDL-010, REQ-TRN-001, REQ-TRN-003 |
| 6876–6878 | PART XXII — TRAINING / FINE-TUNING | REQ-TRN-001, REQ-TRN-003 |
| 6879–6914 | › 23. Khi nào thay đổi model? | REQ-EVL-008, REQ-TRN-001, REQ-TRN-003 |
| 6915–6917 | PART XXIII — BENCHMARKS | REQ-EVL-008 |
| 6918–6940 | › 24. Benchmark cao ≠ production tốt | REQ-EVL-008, REQ-TRN-005, REQ-CV-001, REQ-CV-003 |
| 6941–6943 | PART XXIV — COMPUTER VISION / YOLO LESSON | REQ-TRN-005, REQ-CV-001, REQ-CV-003 |
| 6944–6974 | › 25. Model tốt nhưng environment sai vẫn fail | REQ-EVL-005, REQ-EVL-006, REQ-EVL-020, REQ-TRN-005, REQ-CV-001, REQ-CV-003 |
| 6975–6977 | PART XXV — EVALUATION | REQ-EVL-005, REQ-EVL-006, REQ-EVL-020 |
| 6978–7010 | › 26. Làm sao chứng minh system tốt? | REQ-MDL-008, REQ-MDL-009, REQ-TOL-008, REQ-TOL-009, REQ-VAL-009, REQ-VAL-010, REQ-EVL-005, REQ-EVL-006, REQ-EVL-020 |
| 7011–7013 | PART XXVI — RELIABILITY | REQ-MDL-008, REQ-MDL-009, REQ-TOL-008, REQ-TOL-009, REQ-VAL-009, REQ-VAL-010 |
| 7014–7031 | › 27. AI fail thì chuyện gì xảy ra? | REQ-MDL-008, REQ-MDL-009, REQ-TOL-008, REQ-TOL-009, REQ-VAL-009, REQ-VAL-010 |
| 7032–7034 | PART XXVII — SECURITY / GOVERNANCE | REQ-CLR-007, REQ-SEC-004, REQ-SEC-005, REQ-SEC-006, REQ-SEC-008, REQ-SEC-009 |
| 7035–7061 | › 28. Model được phép thấy và làm gì? | REQ-CLR-007, REQ-OBS-001, REQ-OBS-002, REQ-OBS-003, REQ-OBS-011, REQ-SEC-004, REQ-SEC-005, REQ-SEC-006, REQ-SEC-008, REQ-SEC-009 |
| 7062–7064 | PART XXVIII — OBSERVABILITY | REQ-OBS-001, REQ-OBS-002, REQ-OBS-003, REQ-OBS-011 |
| 7065–7104 | › 29. Output sai → debug được không? | REQ-OBS-001, REQ-OBS-002, REQ-OBS-003, REQ-OBS-011, REQ-SLA-001, REQ-SLA-004 |
| 7105–7107 | PART XXIX — SLA / PRODUCTION | REQ-OBS-011, REQ-SLA-001, REQ-SLA-004 |
| 7108–7136 | › 30. Demo → Enterprise | REQ-EVL-011, REQ-OBS-006, REQ-OBS-011, REQ-SLA-001, REQ-SLA-004, REQ-TRN-004 |
| 7137–7139 | PART XXX — SYSTEM IMPROVEMENT | REQ-EVL-011, REQ-OBS-006, REQ-TRN-004 |
| 7140–7168 | › 31. Khi hệ thống fail, sửa cái gì? | REQ-EVL-011, REQ-OBS-005, REQ-OBS-006, REQ-TRN-004 |
| 7169–7171 | PART XXXI — THE DEBUGGING QUESTION TREE | REQ-OBS-005 |
| 7172–7215 | › 32. Khi AI trả lời sai, hỏi theo thứ tự này | REQ-RTE-014, REQ-EVL-012, REQ-OBS-005 |
| 7216–7267 | PART XXXII — FIRST PRINCIPLES DECISION TREE | REQ-MND-037, REQ-RTE-014, REQ-EVL-012 |
| 7268–7309 | PART XXXIII — FINAL MENTAL MODEL | REQ-MND-037 |
| 7310–7321 | THE ONE QUESTION TO REMEMBER | REQ-MND-013 |
| 7322–7330 | (không tiêu đề) lời kết Problem Map — dòng 7322-7329 | REQ-BIZ-019 |

### 3.3 `brief-01-case-study-construction.md` — Brief 01 — case study xây dựng cầu đường

22 mục; 22 mục có REQ; 0 mục dẫn nhập.

| Dòng | Mục | REQ phủ |
|---|---|---|
| 1–6 | (mở đầu, chưa có tiêu đề) | REQ-BIZ-007, REQ-BIZ-008 |
| 7–14 | Case Study chính: Doanh nghiệp xây dựng hạ tầng / cầu đường | REQ-BIZ-007, REQ-BIZ-008 |
| 15–48 | › Vì sao tôi chọn cầu đường thay vì giày-da? | REQ-BIZ-007, REQ-BIZ-008, REQ-BIZ-016 |
| 49–102 | Tôi đề xuất dựng một doanh nghiệp giả lập | REQ-BIZ-004, REQ-SEC-014 |
| 103–108 | Và đây mới là phần cực hay | REQ-BIZ-012 |
| 109–181 | › Case 01 — Báo cáo công trường | REQ-BIZ-012, REQ-NRM-001, REQ-NRM-002, REQ-NRM-003, REQ-NRM-004, REQ-NRM-005, REQ-NRM-012, REQ-NRM-015, REQ-CLR-001, REQ-CLR-008 |
| 182–221 | Case 02 — Giám đốc hỏi số liệu chính xác | REQ-BIZ-012, REQ-RTE-011, REQ-MDL-002, REQ-VAL-001, REQ-RTE-018 |
| 222–255 | Case 03 — Hỏi kiến thức nghiệp vụ | REQ-BIZ-012, REQ-RTE-011, REQ-RAG-018 |
| 256–292 | Case 04 — Một câu hỏi cần cả SQL + RAG | REQ-BIZ-012, REQ-DIG-007, REQ-RTE-010, REQ-TOL-005, REQ-VAL-005 |
| 293–347 | Case 05 — Data conflict | REQ-BIZ-012, REQ-SOT-001, REQ-SOT-004, REQ-SOT-005, REQ-SOT-006, REQ-CLR-006 |
| 348–404 | Case 06 — Cùng một từ nhưng nghĩa khác | REQ-BIZ-012, REQ-ENT-002, REQ-ENT-003, REQ-ENT-005 |
| 405–454 | Case 07 — PDF hồ sơ | REQ-BIZ-012, REQ-DIG-001, REQ-DIG-003, REQ-DIG-006, REQ-DIG-007, REQ-DIG-008, REQ-RAG-005 |
| 455–487 | Case 08 — Ảnh công trường | REQ-BIZ-008, REQ-BIZ-012, REQ-CV-002 |
| 488–524 | Case 09 — Tool / MCP | REQ-BIZ-012, REQ-TOL-006, REQ-TOL-007, REQ-MCP-001, REQ-MCP-002 |
| 525–562 | Case 10 — Skill | REQ-BIZ-012, REQ-SKL-001, REQ-SKL-003, REQ-SKL-005, REQ-WFL-007, REQ-VAL-004, REQ-VAL-005 |
| 563–608 | Case 11 — Agent | REQ-BIZ-012, REQ-AGT-001, REQ-AGT-003 |
| 609–661 | Case 12 — Evaluation | REQ-BIZ-012, REQ-RTE-002, REQ-RAG-014, REQ-EVL-001 |
| 662–718 | Case 13 — Production | REQ-BIZ-012, REQ-SEC-004, REQ-SEC-005, REQ-SEC-009, REQ-SEC-011, REQ-TEN-004, REQ-SLA-005 |
| 719–749 | Đây chính là lý do tôi chọn CẦU ĐƯỜNG | REQ-BIZ-007, REQ-BIZ-013 |
| 750–795 | › Còn cơ quan nhà nước thì sao? | REQ-BIZ-017, REQ-SOT-011 |
| 796–862 | Tôi sẽ xây bộ 3 case study như thế này | REQ-BIZ-015, REQ-BIZ-016 |
| 863–933 | Thứ tự tôi khuyên | REQ-BIZ-014, REQ-BIZ-015, REQ-TEN-005, REQ-TEN-007, REQ-TEN-011, REQ-TEN-012, REQ-TEN-015 |

### 3.4 `brief-02-v2-two-tenants.md` — Brief 02 — V2 hai tenant

37 mục; 37 mục có REQ; 0 mục dẫn nhập.

| Dòng | Mục | REQ phủ |
|---|---|---|
| 1–12 | (mở đầu, chưa có tiêu đề) | REQ-BIZ-015, REQ-BIZ-016, REQ-OBS-010, REQ-TEN-009, REQ-RT-002 |
| 13–14 | ENTERPRISE AI SYSTEMS CASE STUDY — V2 | REQ-BIZ-003, REQ-SEC-005, REQ-SEC-007, REQ-SEC-013 *(tiêu đề nhóm, gộp từ mục con)* |
| 15–74 | › ROLE | REQ-BIZ-003, REQ-SEC-005, REQ-SEC-007, REQ-SEC-013 |
| 75–96 | PRIMARY OBJECTIVE | REQ-TEN-001 |
| 97–134 | CORE HYPOTHESIS TO TEST | REQ-PCK-001, REQ-TEN-001 |
| 135–136 | TWO REFERENCE TENANTS | REQ-BIZ-004, REQ-BIZ-005, REQ-BIZ-006, REQ-ENT-007, REQ-OBS-010, REQ-RT-005, REQ-RT-010, REQ-RT-011 *(tiêu đề nhóm, gộp từ mục con)* |
| 137–171 | › TENANT A — ABC INFRASTRUCTURE | REQ-BIZ-004 |
| 172–228 | › TENANT B — SPORTS DATA PLATFORM | REQ-BIZ-005, REQ-BIZ-006, REQ-ENT-007, REQ-OBS-010, REQ-RT-005, REQ-RT-010, REQ-RT-011 |
| 229–271 | PART I — MODEL THE BUSINESS BEFORE MODELING AI | REQ-BIZ-002, REQ-EVL-021 |
| 272–327 | PART II — TENANT A: CONSTRUCTION DOMAIN DECOMPOSITION | REQ-BIZ-009, REQ-BIZ-010 |
| 328–371 | PART III — CONSTRUCTION RAW REALITY | REQ-NRM-001, REQ-NRM-003, REQ-NRM-004, REQ-NRM-005, REQ-NRM-014, REQ-SOT-006, REQ-SEC-014 |
| 372–434 | PART IV — DOMAIN LANGUAGE | REQ-ENT-001, REQ-ENT-002, REQ-ENT-003, REQ-CLR-004 |
| 435–490 | PART V — DATA AUDIT | REQ-DAT-001, REQ-DAT-002, REQ-DAT-003, REQ-DAT-004, REQ-NRM-012, REQ-SEC-013, REQ-RT-001 |
| 491–529 | PART VI — SOURCE OF TRUTH | REQ-SOT-001, REQ-SOT-002, REQ-SOT-003, REQ-PCK-008, REQ-CLR-006, REQ-SEC-009 |
| 530–582 | PART VII — NORMALIZATION ENGINE | REQ-NRM-007, REQ-NRM-008, REQ-NRM-009, REQ-NRM-010, REQ-NRM-012 |
| 583–621 | PART VIII — DETERMINISTIC / PROBABILISTIC BOUNDARY | REQ-RTE-013 |
| 622–659 | PART IX — ROUTER | REQ-RTE-001, REQ-RTE-002, REQ-RTE-009, REQ-MDL-003, REQ-PCK-008, REQ-CLR-007, REQ-SLA-004 |
| 660–706 | PART X — CONTEXT ENGINEERING | REQ-CTX-001, REQ-CTX-003, REQ-CTX-004, REQ-CTX-005, REQ-CTX-006, REQ-CTX-007, REQ-CTX-008, REQ-CTX-010, REQ-CTX-011, REQ-CTX-012 |
| 707–776 | PART XI — RAG | REQ-DIG-006, REQ-DIG-008, REQ-ENT-008, REQ-RAG-003, REQ-RAG-004, REQ-RAG-007, REQ-RAG-015, REQ-RAG-016, REQ-RAG-018, REQ-RAG-022, REQ-SEC-007 |
| 777–811 | PART XII — TOOLS | REQ-TOL-001, REQ-TOL-006, REQ-TOL-007 |
| 812–844 | PART XIII — SKILLS | REQ-SKL-002, REQ-SKL-003, REQ-SKL-004, REQ-SKL-006 |
| 845–872 | PART XIV — AGENT | REQ-AGT-002, REQ-AGT-003, REQ-AGT-004 |
| 873–913 | PART XV — MCP | REQ-MCP-002, REQ-MCP-003, REQ-MCP-004 |
| 914–950 | PART XVI — TENANT B: SPORTS DATA PLATFORM | REQ-BIZ-011, REQ-ENT-006, REQ-RT-009, REQ-RT-011 |
| 951–1000 | PART XVII — SPORTS EVENT MODEL | REQ-ENT-006, REQ-WFL-013, REQ-RT-001, REQ-RT-002 |
| 1001–1047 | PART XVIII — REALTIME DATA | REQ-ENT-006, REQ-ENT-008, REQ-OBS-010, REQ-RT-002, REQ-RT-003, REQ-RT-004, REQ-RT-005, REQ-RT-006, REQ-RT-007, REQ-RT-009 |
| 1048–1078 | PART XIX — SPORTS AI QUESTIONS | REQ-RTE-012, REQ-RT-002, REQ-RT-008 |
| 1079–1118 | PART XX — CROSS-TENANT COMPARISON | REQ-TEN-002 |
| 1119–1152 | PART XXI — FIND THE TRUE PLATFORM BOUNDARY | REQ-TEN-003, REQ-TEN-008, REQ-TEN-010 |
| 1153–1199 | PART XXII — MULTI-TENANCY | REQ-MCP-003, REQ-PCK-008, REQ-OBS-008, REQ-SEC-010, REQ-SEC-011, REQ-TEN-004, REQ-TEN-005, REQ-TEN-006 |
| 1200–1232 | PART XXIII — FAILURE CATALOG | REQ-EVL-004, REQ-OBS-003 |
| 1233–1290 | PART XXIV — TEST CASE GENERATION | REQ-EVL-002, REQ-EVL-003 |
| 1291–1325 | PART XXV — EVALUATION | REQ-ENT-009, REQ-EVL-005, REQ-EVL-006, REQ-EVL-007, REQ-TEN-006 |
| 1326–1365 | PART XXVI — OBSERVABILITY | REQ-OBS-001 |
| 1366–1407 | PART XXVII — FIRST PRINCIPLES GATE | REQ-MND-026, REQ-MND-027 |
| 1408–1467 | PART XXVIII — REQUIRED FINAL DELIVERABLE | REQ-MND-040, REQ-TEN-011 |
| 1468–1508 | FINAL RULE | REQ-BIZ-005, REQ-BIZ-016, REQ-TEN-008, REQ-TEN-009, REQ-TEN-015 |

### 3.5 `brief-03-eais-naming.md` — Brief 03 — đặt tên EAIS

4 mục; 4 mục có REQ; 0 mục dẫn nhập.

| Dòng | Mục | REQ phủ |
|---|---|---|
| 1–6 | (mở đầu, chưa có tiêu đề) | REQ-MND-016 |
| 7–10 | Multi-Tenant Enterprise AI Systems Platform | REQ-MND-016 |
| 11–54 | EAIS — Enterprise AI Systems | REQ-MND-016, REQ-MDL-001, REQ-PCK-001, REQ-TEN-001, REQ-TEN-004, REQ-TEN-013 |
| 55–138 | › Tôi sẽ đặt hierarchy tên như sau | REQ-MND-003, REQ-MND-016, REQ-MND-043, REQ-TEN-003, REQ-TEN-009, REQ-TEN-010, REQ-TEN-011, REQ-TEN-012, REQ-TEN-013, REQ-TEN-014 |

### 3.6 `brief-04-rag-to-eais-expansion.md` — Brief 04 — mở rộng RAG → EAIS

18 mục; 18 mục có REQ; 0 mục dẫn nhập.

| Dòng | Mục | REQ phủ |
|---|---|---|
| 1–45 | (mở đầu, chưa có tiêu đề) | REQ-MND-032, REQ-MND-044 |
| 46–109 | 1. Vấn đề dự án cũ không phải RAG làm sai | REQ-RAG-005, REQ-RAG-008, REQ-RAG-009, REQ-EVL-009, REQ-MND-044 |
| 110–200 | 2. Vậy V2 không phải “RAG V2” | REQ-MND-044, REQ-MND-045 |
| 201–302 | 3. Đừng rewrite RAG. Bọc nó lại | REQ-MND-032, REQ-MDL-001, REQ-TEN-014 |
| 303–342 | 4. Thay đổi lớn nhất: thêm một lớp trước RAG | REQ-RTE-011 |
| 343–358 | › Request A | REQ-RTE-011 |
| 359–372 | › Request B | REQ-RTE-011 |
| 373–398 | › Request C | REQ-RTE-010, REQ-RTE-011 |
| 399–420 | › Request D | REQ-RTE-011 |
| 421–506 | 5. Thực ra bạn đang chuyển từ Model-centric → System-centric | REQ-MND-033, REQ-TEN-001, REQ-TEN-012 |
| 507–567 | 6. Và đây là nơi Construction Case Study cực kỳ quan trọng | REQ-MND-028, REQ-NRM-014, REQ-ENT-004, REQ-SOT-005, REQ-SOT-006 |
| 568–672 | 7. Đây mới đúng First Principles | REQ-MND-028, REQ-MND-029, REQ-NRM-001 |
| 673–743 | 8. Construction tenant cũng không được chui vào Core | REQ-PCK-001, REQ-PCK-004, REQ-TEN-008, REQ-RT-007 |
| 744–792 | 9. Tôi sẽ giữ RAG cũ thành một bounded capability | REQ-RAG-001 |
| 793–892 | 10. Roadmap migration của project cũ → EAIS | REQ-MND-031, REQ-RAG-009, REQ-RAG-010, REQ-TEN-010 |
| 893–924 | 11. Và tôi sẽ đặt một luật cực mạnh cho dự án | REQ-MND-028 |
| 925–954 | › Vậy project cũ của bạn thực chất rất có giá trị | REQ-MND-032 |
| 955–959 | RAG → EAIS Architectural Expansion | REQ-MND-032 |

### 3.7 `brief-05-architecture-review.md` — Brief 05 — nhận xét kiến trúc (execution, workflow, fact)

15 mục; 15 mục có REQ; 0 mục dẫn nhập.

| Dòng | Mục | REQ phủ |
|---|---|---|
| 1–4 | (mở đầu, chưa có tiêu đề) | REQ-MND-017 |
| 5–53 | › Điểm tôi đánh giá rất đúng | REQ-MND-001, REQ-MND-036, REQ-TEN-005 |
| 54–101 | › S0 → S10 là phần mạnh nhất | REQ-VAL-001, REQ-VAL-002, REQ-RTE-017 |
| 102–204 | Nhưng tôi thấy một vấn đề lớn tiếp theo | REQ-WFL-002, REQ-WFL-007, REQ-WFL-011, REQ-WFL-014, REQ-WFL-015, REQ-WFL-016, REQ-WFL-017 |
| 205–233 | Tôi sẽ thêm một execution architecture song song | REQ-WFL-006, REQ-WFL-008, REQ-WFL-009, REQ-WFL-014, REQ-WFL-015, REQ-WFL-019 |
| 234–242 | › QUERY | REQ-WFL-006, REQ-WFL-008, REQ-WFL-009, REQ-WFL-014, REQ-WFL-015, REQ-WFL-019 |
| 243–254 | › COMMAND | REQ-WFL-006, REQ-WFL-008, REQ-WFL-009, REQ-WFL-010, REQ-WFL-014, REQ-WFL-015, REQ-WFL-019 |
| 255–269 | › EVENT | REQ-WFL-006, REQ-WFL-008, REQ-WFL-009, REQ-WFL-011, REQ-WFL-012, REQ-WFL-013, REQ-WFL-014, REQ-WFL-015, REQ-WFL-019 |
| 270–327 | Thiếu tiếp: Workflow/State Machine cần trở thành first-class | REQ-AGT-004, REQ-WFL-001, REQ-WFL-002, REQ-WFL-003, REQ-WFL-004, REQ-WFL-005, REQ-WFL-006, REQ-WFL-007, REQ-PCK-006 |
| 328–398 | Source-of-truth của file rất tốt, nhưng nên tách “Fact” và “Observation” | REQ-SOT-002, REQ-SOT-006, REQ-SOT-007, REQ-SOT-008, REQ-SOT-009, REQ-SOT-010, REQ-CTX-006 |
| 399–457 | Pack hiện tại rất tốt nhưng đang có nguy cơ trở thành “God Config” | REQ-WFL-005, REQ-WFL-012, REQ-PCK-002, REQ-PCK-003, REQ-PCK-004, REQ-PCK-005, REQ-PCK-006 |
| 458–511 | Một điểm nữa tôi muốn audit: “router theo luật” | REQ-RTE-004, REQ-RTE-005 |
| 512–545 | Một điểm tôi rất thích: capability gates | REQ-MND-030, REQ-NRM-011, REQ-RTE-005, REQ-RAG-014, REQ-MCP-002, REQ-WFL-003, REQ-TRN-003 |
| 546–575 | Tôi chấm architecture hiện tại như này | REQ-MND-030, REQ-WFL-002, REQ-WFL-003, REQ-WFL-006, REQ-WFL-007, REQ-WFL-018, REQ-WFL-019, REQ-EVL-014, REQ-EVL-016, REQ-RT-003, REQ-RT-004 |
| 576–653 | › V3 tôi sẽ ép Construction phá architecture này | REQ-MND-028, REQ-MND-029, REQ-MND-031, REQ-WFL-002, REQ-WFL-009, REQ-WFL-011, REQ-WFL-014, REQ-WFL-016, REQ-WFL-017, REQ-EVL-014, REQ-EVL-015, REQ-SEC-007, REQ-TEN-006 |

### 3.8 `brief-06-model-vs-enterprise-system.md` — Brief 06 — model vs enterprise system

13 mục; 13 mục có REQ; 0 mục dẫn nhập.

| Dòng | Mục | REQ phủ |
|---|---|---|
| 1–6 | (mở đầu, chưa có tiêu đề) | REQ-MND-017 |
| 7–60 | › 1. Bản thân Model AI thực ra nhỏ hơn cái chữ “AI System” rất nhiều | REQ-MND-002, REQ-MND-007, REQ-DIG-008, REQ-CTX-014, REQ-MDL-012 |
| 61–112 | 2. RAG cũ của bạn đang làm một phần của việc đó | REQ-CTX-001 |
| 113–194 | 3. “Enterprise” nằm rất nhiều ở phần trước và sau model | REQ-NRM-013, REQ-ENT-004, REQ-MDL-002, REQ-TOL-005, REQ-PCK-008, REQ-RTE-018 |
| 195–200 | 4. Vậy AI thật sự nằm ở đâu? | REQ-MND-014, REQ-MND-017, REQ-MND-019 |
| 201–221 | › A. Software Engineering | REQ-MND-017, REQ-MND-019 |
| 222–237 | › B. Data / Information Engineering | REQ-MND-017, REQ-MND-019 |
| 238–277 | › C. Probabilistic Intelligence | REQ-MND-017, REQ-MND-019, REQ-ENT-008, REQ-ENT-009 |
| 278–351 | 5. Còn `Role` không phải là quyền doanh nghiệp | REQ-CTX-007, REQ-CLR-007, REQ-SEC-001, REQ-SEC-002, REQ-SEC-004, REQ-SEC-006 |
| 352–401 | 6. Enterprise AI thực chất là kiểm soát “model được nhìn thấy thế giới nào” | REQ-MND-014 |
| 402–470 | 7. Và đây mới là cái “Role” thú vị của Enterprise AI | REQ-MDL-011, REQ-SEC-001, REQ-SEC-003, REQ-SEC-006 |
| 471–516 | 8. Vì vậy cái bạn đang xây không phải “làm AI model” | REQ-MND-017, REQ-MND-018, REQ-RTE-017 |
| 517–555 | › Một câu có thể dùng làm định nghĩa EAIS | REQ-MND-001, REQ-MND-013, REQ-MND-015, REQ-MND-017 |

### 3.9 `brief-07-data-realism-proof.md` — Brief 07 — dữ liệu thực tế và Proof Harness

15 mục; 15 mục có REQ; 0 mục dẫn nhập.

| Dòng | Mục | REQ phủ |
|---|---|---|
| 1–4 | (mở đầu, chưa có tiêu đề) | REQ-EVL-022 |
| 5–47 | › Vấn đề thật sự là 3 chiều khác nhau | REQ-EVL-022 |
| 48–127 | Và chuyện “nhiều format → quy chuẩn về một cái” chính là một bài toán lớn | REQ-NRM-016 |
| 128–173 | Case testing của bạn phải thay đổi theo phát hiện này | REQ-EVL-023 |
| 174–220 | › Tôi sẽ chia test data EAIS thành 5 cấp | REQ-EVL-024, REQ-EVL-026 |
| 221–255 | Đây là lý do `simulations/` trong architecture của bạn rất đáng giá | REQ-EVL-025, REQ-EVL-027 |
| 256–312 | Ví dụ Construction simulation | REQ-EVL-025, REQ-EVL-028 |
| 313–369 | Sports còn test scale rất hay | REQ-EVL-029 |
| 370–373 | Và cần tách 2 câu hỏi | REQ-EVL-030 |
| 374–380 | › Correctness | REQ-EVL-030 |
| 381–400 | › Scalability | REQ-EVL-030 |
| 401–455 | Tôi sẽ thêm một trục nữa: Cardinality | REQ-EVL-031 |
| 456–514 | Quan trọng hơn nữa: test transformation pipeline | REQ-DAT-009 |
| 515–563 | Ví dụ một test cực giá trị | REQ-NRM-017 |
| 564–626 | Vậy phát hiện của bạn làm roadmap thay đổi thế nào? | REQ-EVL-027, REQ-EVL-032 |

### 3.10 `brief-08-rag-vs-eais-v0-v4.md` — Brief 08 — RAG vs EAIS (V0–V4)

10 mục; 10 mục có REQ; 0 mục dẫn nhập.

| Dòng | Mục | REQ phủ |
|---|---|---|
| 1–2 | (mở đầu, chưa có tiêu đề) | REQ-MND-054, REQ-CLR-009, REQ-EVL-033, REQ-BIZ-021, REQ-TEN-009, REQ-EVL-027, REQ-EVL-032, REQ-MND-055 *(tiêu đề nhóm, gộp từ mục con)* |
| 3–28 | › RAG ban đầu của bạn | REQ-MND-054 |
| 29–76 | › EAIS hiện tại | REQ-MND-054 |
| 77–126 | › Cùng là “hỏi đáp”, nhưng hai thứ đang test khác nhau | REQ-MND-054 |
| 127–132 | Ví dụ sẽ thấy ngay sự khác biệt | REQ-CLR-009 |
| 133–190 | › RAG-centric | REQ-CLR-009 |
| 191–254 | Đây cũng là lý do phát hiện vừa rồi của bạn về data scale rất quan trọng | REQ-EVL-033 |
| 255–324 | Và đây mới là chỗ Construction case study có giá trị | REQ-BIZ-021 |
| 325–368 | Sports tenant cũng không phải để làm chatbot thể thao | REQ-TEN-009 |
| 369–440 | › Vì vậy tôi sẽ mô tả quá trình tiến hóa dự án của bạn như này | REQ-EVL-027, REQ-EVL-032, REQ-MND-054, REQ-MND-055 |

### 3.11 `brief-09-blind-spot-audit-29.md` — Brief 09 — audit 29 điểm mù

35 mục; 35 mục có REQ; 0 mục dẫn nhập.

| Dòng | Mục | REQ phủ |
|---|---|---|
| 1–11 | (mở đầu, chưa có tiêu đề) | REQ-MND-057 |
| 12–69 | 1. 🔴 Bạn đang canonicalize data, nhưng chưa canonicalize business meaning | REQ-SOT-012 |
| 70–108 | › Blind spot | REQ-SOT-012 |
| 109–162 | 2. 🔴 EAIS chưa có Enterprise Ontology / Semantic Model đủ mạnh | REQ-ENT-010 |
| 163–223 | 3. 🔴 Time hiện đang quá đơn giản | REQ-SOT-013 |
| 224–275 | 4. 🔴 Identity Resolution mới giải “entity nào?”, chưa giải identity lifecycle | REQ-ENT-011 |
| 276–305 | 5. 🔴 Query pipeline mạnh, nhưng Command/Event/Workflow vẫn chưa phải first-class | REQ-WFL-020 |
| 306–329 | › Đây có thể thành EAIS V4 architecture change: | REQ-WFL-020 |
| 330–391 | 6. 🔴 Thiếu transaction boundary và consistency model | REQ-WFL-021 |
| 392–428 | 7. 🔴 Retry có thể gây business damage | REQ-TOL-013 |
| 429–469 | 8. 🟠 Bạn có Source Authority nhưng chưa đủ Data Ownership | REQ-DAT-010 |
| 470–503 | 9. 🟠 Data Quality chưa có SLO | REQ-DAT-011 |
| 504–554 | 10. 🟠 Schema evolution chưa được stress-test | REQ-DAT-012 |
| 555–599 | 11. 🟠 Pack version tạo ra một vấn đề mới: reproducibility | REQ-OBS-012 |
| 600–648 | 12. 🟠 RAG index freshness chưa đồng nghĩa Source-of-Truth freshness | REQ-RAG-025 |
| 649–692 | 13. 🟠 Permission không chỉ là “được xem document không” | REQ-SEC-015 |
| 693–746 | 14. 🔴 Prompt injection qua enterprise data | REQ-SEC-016 |
| 747–806 | 15. 🔴 “Validator kiểm số xuất hiện trong Evidence” vẫn có thể cho sai nhưng trông đúng | REQ-VAL-011 |
| 807–838 | 16. 🟠 Confidence đang có nhưng chưa có calibration | REQ-VAL-012 |
| 839–870 | 17. 🟠 Human-in-the-loop chưa có economics | REQ-CLR-010 |
| 871–916 | 18. 🟠 Cost model không chỉ là token cost | REQ-OBS-013 |
| 917–950 | 19. 🟠 SLA cũng phải là business SLA | REQ-SLA-006 |
| 951–984 | 20. 🟠 Multi-tenancy còn bài toán noisy neighbor | REQ-SEC-017 |
| 985–1018 | 21. 🟠 Tenant onboarding chưa được coi là metric | REQ-TEN-016 |
| 1019–1060 | 22. 🔴 Bạn cần test model replacement | REQ-MDL-014 |
| 1061–1087 | 23. 🟠 Tương tự: Retrieval replacement test | REQ-RAG-026 |
| 1088–1129 | 24. 🔴 Một blind spot lớn: business invariants | REQ-EVL-034 |
| 1130–1168 | 25. 🔴 Case count có thể trở thành vanity metric | REQ-EVL-035 |
| 1169–1216 | 26. 🔴 Synthetic data cũng có thể tạo “synthetic illusion” | REQ-EVL-026, REQ-EVL-036 |
| 1217–1256 | 27. 🟠 Observability có trace, nhưng chưa chắc có causal debugging | REQ-OBS-014 |
| 1257–1290 | 28. 🟠 Migration từ RAG cũ vẫn là một rủi ro riêng | REQ-EVL-037 |
| 1291–1360 | 29. 🔴 Có nguy cơ EAIS trở thành “architecture research project” thay vì product | REQ-MND-056 |
| 1361–1393 | Tổng hợp audit | REQ-MND-057 |
| 1394–1464 | Bức tranh mới sau audit | REQ-MND-058 |
| 1465–1487 | › Nếu gom tất cả audit đến giờ thành một câu | REQ-MND-059 |

### 3.12 `brief-10-working-hypothesis-first-principles.md` — Brief 10 — working hypothesis và nguyên lý đầu

8 mục; 8 mục có REQ; 0 mục dẫn nhập.

| Dòng | Mục | REQ phủ |
|---|---|---|
| 1–51 | (mở đầu, chưa có tiêu đề) | REQ-MND-060 |
| 52–93 | › Điểm rất quan trọng: roadmap vừa audit không phải đang “đi xa khỏi AI” | REQ-MND-060, REQ-MND-062 |
| 94–154 | › Và phát hiện mới về data cũng đúng cùng một logic | REQ-EVL-038 |
| 155–255 | Thậm chí audit vừa rồi còn giúp sửa roadmap một chút | REQ-MND-058, REQ-MND-061, REQ-AGT-006 |
| 256–327 | Vì thế Construction case study cũng phải hiểu lại | REQ-MND-056, REQ-BIZ-022 |
| 328–367 | Đây là điểm tôi đánh giá roadmap hiện tại đúng nhất | REQ-MND-058, REQ-AGT-006 |
| 368–411 | › Tôi chỉ chỉnh một điểm quan trọng trong roadmap hiện tại | REQ-MND-062 |
| 412–464 | › Nên bức tranh hiện tại của dự án, theo tôi, là: | REQ-MND-056, REQ-MND-062 |

### 3.13 `brief-11-claude-md-rationale.md` — Brief 11 — lý do cấu trúc CLAUDE.md

1 mục; 1 mục có REQ; 0 mục dẫn nhập.

| Dòng | Mục | REQ phủ |
|---|---|---|
| 1–10 | (mở đầu, chưa có tiêu đề) | REQ-MND-063, REQ-MDL-015 |

---

## Phụ lục A. Xung đột và thay thế giữa các nguồn

| # | Điều lệch | Cách xử lý trong sổ |
|---|---|---|
| 1 | Brief 01 xếp tenant thứ hai là giày-da và thứ ba là cơ quan nhà nước; brief 02 chốt hai tenant tham chiếu là construction + sports và lùi giày-da thành Tenant C | brief 02 thay thế (REQ-BIZ về thứ tự tenant, REQ-TEN tenant sau) |
| 2 | Brief 01 Case 02 viết "LLM formats answer" cho câu số liệu; kiến trúc quy định route số liệu dùng mẫu câu của tenant, 0 lần gọi model | REQ-MDL về 0 lần gọi model theo kiến trúc; ghi nhận ý gốc |
| 3 | Brief 01 Case 05 coi bốn con số là bốn nguồn của một fact; brief 05 làm rõ chúng là bốn ngữ nghĩa (reported/issued/measured/billed) | brief 05 làm rõ (REQ-SOT về chỉ số ngữ nghĩa và thẩm quyền chỉ trong cùng chỉ số) |
| 4 | Hai bản cây quyết định đặt khác thứ tự (fact có cấu trúc/hiện tại so với tri thức tài liệu; quy trình đã biết so với hành động) | chọn thứ tự khớp P3 (fact chính xác có cấu trúc trước, quy trình trước hành động); ghi ở REQ-RTE về cây quyết định |
| 5 | Ba danh sách câu hỏi nguyên lý đầu (35 câu của study guide, 20 câu của workshop note, 15 câu cổng của brief 02) | giữ cả ba, mỗi danh sách một REQ; không gộp để khỏi mất câu |
| 6 | Sơ đồ "model ở giữa + năm vệ tinh" bị chính ghi chú nguồn đánh giá là chưa phải kiến trúc chuẩn | REQ-MND về bốn mặt phẳng, tám tầng, bảy subsystem |
| 7 | Hai cách đánh số bậc tiến hoá: brief 04 (V1 model-centric → V4 tenant-agnostic) và brief 08 (V0 LLM → V4 EAIS PROOF) | giữ cả hai, mỗi cách một REQ-MND; tài liệu roadmap ghi quy ước dùng |
| 8 | Brief 05 nêu ba lối vào QUERY/COMMAND/EVENT; brief 09 thêm WORKFLOW, DECISION, NOTIFICATION | MVP có QUERY/COMMAND/EVENT/WORKFLOW (REQ-WFL), DECISION và NOTIFICATION khai báo ở V1 |
| 9 | Brief 09/10 chốt North Star "một tenant, một quy trình" (Fuel Reconciliation) trong khi phạm vi MVP của kiến trúc gồm hai tenant tham chiếu | giữ cả hai: REQ-MND về lát cắt dọc đặt thứ tự làm; chủ dự án cần chốt lát cắt đầu tiên (Phụ lục B) |

## Phụ lục B. Mục nguồn còn phân vân

| Mục | Điều phân vân | Hiện đang xử lý |
|---|---|---|
| v0/v1 — các phát biểu 5%/95%, 90%/100%, 95% độ chính xác công cụ đọc bảng, tên nghe không rõ, claim overfit benchmark | chưa kiểm chứng | `phase: NA` có `note`, hàm ý đúng (benchmark ≠ production, đo bằng bộ ca tenant) đã thành REQ-EVL |
| v0 §42 và v1 §27 — checklist học | thuộc tài liệu học hơn là hành vi hệ thống | REQ-MND V1 về đường học, mỗi chủ đề trỏ REQ |
| v1 — đoạn mô tả MCP Skills extension và đặc tả MCP stateless | phụ thuộc thông tin ngoài nguồn | REQ-MCP GATED, chỉ xem xét khi cổng mở |
| brief 06 — TIME RESOLUTION: "tháng này" là từ mùng 1 đến hôm nay hay cả tháng lịch | nguồn không nói | pack định nghĩa quy tắc và câu trả lời luôn nêu khoảng đã dùng (REQ-NRM) |
| brief 02 — số lượng tối thiểu (30 domain, 50 ca router, 50+50+30 ca hỏng, 100 ca V3) | khối lượng lớn cho MVP | phase MVP cho khung và 50 ca router; 100 ca V3 ở REQ-EVL, ca chi tiết do tài liệu tenant sinh |
| brief 09 §29 và brief 10 — North Star một lát cắt dọc Fuel Reconciliation | mâu thuẫn tiềm tàng với MVP hai tenant | REQ-MND về lát cắt dọc ở MVP; thứ tự làm do chủ dự án chốt |
| brief 07 — cấp dữ liệu D4 (dữ liệu thật ẩn danh/replay) | cần căn cứ pháp lý chưa có | REQ-EVL GATED, mặc định đóng |
| brief 09 #25 — ma trận độ phủ (tám chiều) | chưa có ngưỡng độ phủ tối thiểu | REQ-EVL MVP chỉ yêu cầu báo cáo chiều có 0 ca; ngưỡng đặt sau lần đo đầu |

## Phụ lục C. Yêu cầu NA và GATED

| id | phase | note |
|---|---|---|
| REQ-MND-010 | NA | Phát biểu trong video, chưa kiểm chứng. Nếu cần một con số, đo trên bộ ca của tenant theo CONVENTIONS §6 (dataset, chỉ số, cỡ mẫu). Con số 5% chỉ nên đọc như tín hiệu khoảng cách pilot→production, không thành câu '95% dự án AI thất bại'. |
| REQ-MND-011 | NA | Chưa xác minh được tên. Phạm trù đã được phủ ở các REQ-TRN và REQ-MND về radar công nghệ; không cần tên để vận hành nền tảng. |
| REQ-MND-012 | NA | Claim về một model cụ thể chưa có bằng chứng độc lập. Bài học tổng quát (benchmark ≠ production) đã thành yêu cầu ở lĩnh vực EVL. |
| REQ-MND-042 | NA | Chỉ đạo quy trình/git của một phiên làm việc, không phải yêu cầu hệ thống; ý chuyển hoá luồng được phủ bởi REQ-MND về strangler và migration audit. |
| REQ-MND-043 | NA | Quyết định thương mại/đặt tên của chủ dự án, không ảnh hưởng hành vi hệ thống. |
| REQ-BIZ-008 | GATED | Điều kiện mở cổng: có dữ liệu ảnh/cảm biến thật và ca hỏng tái lập được mà context/tool/luật không giải; theo docs/01-architecture.md §9. |
| REQ-DIG-009 | GATED | Điều kiện mở cổng: có tài liệu thật mà thông tin quan trọng nằm ở bố cục/biểu đồ và parser văn bản + bảng thất bại đo được trên bộ ca đọc; theo nguyên tắc 'earned by evidence'. |
| REQ-ENT-009 | GATED | Điều kiện mở cổng: so khớp tất định trượt ≥ ngưỡng đo được trên bộ ca entity resolution; mở bằng ADR kèm số đo và ngân sách độ trễ; dịch vụ ML thuộc giai đoạn V2 (ml-inference) của ranh giới Go↔Python. |
| REQ-RTE-005 | GATED | Điều kiện mở cổng: luật của pack trượt ≥ một ngưỡng đo được trên bộ ca router (docs/01-architecture.md §9); phải có test âm chứng minh cổng chặn khi chưa đủ bằng chứng. |
| REQ-RAG-014 | GATED | Điều kiện mở cổng (docs/01-architecture.md §9): Recall@20 > Recall@5 rõ rệt trên bộ ca truy xuất; phải có test âm chứng minh cổng chặn khi chưa đủ số đo; dịch vụ ML (nếu có) thuộc giai đoạn V2 của ranh giới Go↔Python. |
| REQ-RAG-020 | GATED | Điều kiện mở cổng: hybrid (+ rerank nếu mở) vẫn trượt vì mảnh tài liệu thiếu ngữ cảnh, đo được trên bộ ca truy xuất; kèm ADR và số đo trước–sau. |
| REQ-RAG-021 | GATED | Điều kiện mở cổng: có bộ ca câu hỏi quan hệ nhiều bước tái lập được mà truy xuất hybrid thất bại đo được; chưa có ca như vậy ở hai tenant tham chiếu. |
| REQ-MDL-007 | GATED | Điều kiện mở cổng: policy tĩnh không đạt ngân sách chi phí hoặc độ trễ đo được trên bộ ca của tenant; kèm ADR và số đo trước–sau. |
| REQ-MCP-002 | GATED | Điều kiện mở cổng (docs/01-architecture.md §9): có bên tiêu thụ thật cần khám phá tool theo chuẩn; chưa có consumer ở hai tenant tham chiếu nên execution/mcp chưa tạo thư mục. |
| REQ-MCP-003 | GATED | Chỉ áp dụng sau khi cổng MCP (REQ-MCP-002) mở; kiểm bằng ADR và test cô lập tenant. |
| REQ-MCP-004 | GATED | Chỉ áp dụng sau khi cổng MCP mở; mọi capability MCP phải đi qua cùng luồng tool nên không tạo đường tắt quyền. |
| REQ-MCP-005 | GATED | Chỉ áp dụng sau khi cổng MCP mở; trước đó roadmap là tài liệu kế hoạch. |
| REQ-MCP-006 | GATED | Chưa có nhu cầu; chỉ xem xét khi cổng MCP mở và có consumer cần tiêu thụ skill qua giao thức. |
| REQ-AGT-001 | GATED | Điều kiện mở cổng (docs/01-architecture.md §9): có ca nghiệp vụ mà workflow cố định không giải được và đo được; phải có test âm chứng minh route AGENT bị từ chối khi cổng đóng. |
| REQ-AGT-002 | GATED | Chỉ áp dụng khi có agent candidate; điều kiện mở theo cổng agent. |
| REQ-AGT-003 | GATED | Chỉ áp dụng cho ca được đề xuất làm agent; cổng agent đóng cho tới khi có thất bại đo được của workflow cố định. |
| REQ-AGT-004 | GATED | Chỉ áp dụng sau khi cổng agent mở; các giới hạn là điều kiện bắt buộc của mọi agent. |
| REQ-AGT-005 | GATED | Điều kiện mở cổng: bộ ca truy xuất một lượt (kể cả hybrid + rerank nếu mở) thất bại đo được và cổng agent đã mở. |
| REQ-WFL-003 | GATED | Điều kiện mở cổng: số đo tải/độ trễ/độ bền cho thấy engine đồng bộ trên PostgreSQL không đạt; mở bằng ADR (ADR-0004), kèm test âm chứng minh cổng. |
| REQ-SLA-003 | NA | Quyết định pháp lý/hợp đồng thuộc chủ dự án và khách hàng, không phải hành vi hệ thống; phần kỹ thuật (bằng chứng, trace, audit) đã có ở REQ-OBS và REQ-SEC. |
| REQ-TRN-003 | GATED | Điều kiện mở cổng (docs/01-architecture.md §9): có dữ liệu thật và ca hỏng mà context/tool không chữa được; phải có test âm chứng minh cổng chặn; huấn luyện chỉ ngoại tuyến. |
| REQ-TRN-006 | NA | Kiến thức nền, không phải hành vi hệ thống; hàm ý đã nằm ở REQ-MND (đầu ra model là đề xuất) và REQ-EVL (benchmark ≠ production). |
| REQ-TRN-007 | NA | Lời khuyên nghề nghiệp/tư vấn; chỉ trở thành yêu cầu khi cổng fine-tune/CV mở (khi đó chi phí và hạ tầng nằm trong ADR của cổng). |
| REQ-CV-001 | GATED | Điều kiện mở cổng (docs/01-architecture.md §9): có dữ liệu thật và ca hỏng mà context/tool không chữa được; dịch vụ thị giác thuộc giai đoạn V2 (python vision) của ranh giới Go↔Python. |
| REQ-CV-002 | GATED | Chỉ áp dụng sau khi cổng thị giác mở; ca nhận dạng ảnh công trường chưa có dữ liệu thật hợp pháp. |
| REQ-CV-003 | GATED | Chỉ áp dụng sau khi cổng thị giác mở; chưa có dataset. |
| REQ-RT-004 | GATED | Điều kiện mở cổng: đo được thông lượng hoặc độ trễ của đường đồng bộ trên dữ liệu mô phỏng cỡ lớn không đạt mục tiêu SLO; mở bằng ADR-0004. |
| REQ-MND-051 | NA | Nhận xét chất lượng nguồn học, không phải hành vi hệ thống; ý hữu ích (dùng video làm mental model) đã nằm ở REQ-MND về nhãn [VIDEO]/[GIẢI THÍCH]/[HIỆU CHỈNH]. |
| REQ-BIZ-020 | NA | Lời khuyên nghề nghiệp/quan hệ khách hàng, không phải hành vi hệ thống; ý hữu ích (chuẩn hoá dữ liệu trước khi gắn AI) đã nằm ở REQ-MND về mức sẵn sàng dữ liệu. |
| REQ-EVL-026 | GATED | Điều kiện mở cổng: có thoả thuận/chấp thuận hợp pháp của chủ dữ liệu, đã ẩn danh theo quy trình được duyệt; mặc định đóng và repo chỉ chứa dữ liệu giả lập. |
| REQ-AGT-006 | GATED | Điều kiện mở cổng giữ như REQ-AGT-001; REQ này thêm yêu cầu 'phải thắng workflow cố định trên eval' trước khi giữ lại agent. |

