# Những gì chủ dự án cần quyết — gom một chỗ

> Cập nhật 20/10/2026 19:5x: **chủ dự án đã trả lời mọi câu** (cột "Trả lời" của từng bảng; câu hỏi, lựa chọn và khuyến nghị cũ giữ nguyên để đọc lại có ngữ cảnh). Câu còn mở: mục 0, dòng cuối. **Một nguồn duy nhất** cho mọi câu chờ chủ dự án. Tài liệu khác chỉ trỏ về đây.
>
> **Cách trả lời** (một trong hai):
> - sửa cột "Trả lời" trong tệp này rồi commit;
> - nói trong phiên làm việc bằng mã câu, ví dụ: `D-G7-1: A; D-G7-2: sonnet; D-G7-3: trần 15 USD, đồng bộ`.
>
> Phiên làm việc nào mở ra cũng đọc tệp này trước. Câu đã trả lời chuyển xuống mục 0 kèm ngày.
>
> **Đính chính 20/10 tối (sau bản soát độc lập `docs/g7/REVIEW-D5.md`):**
> - Kết luận "WIN 6/7 cổng đạt" là **có điều kiện**.
> - Ba ca 083, 107, 108 có kỳ vọng do chính phiên viết mã "trọng tài" sau lần đo đầu (09:03), trước khi băm bộ ca (09:15). G2, G5, G6 phụ thuộc ba ca này.
> - Đạt **vô điều kiện** hiện là **3/7** (G1, G3, G4).
> - Xem D-G7-11.

## 0. Đã quyết

| Ngày | Câu | Quyết định của chủ dự án |
|---|---|---|
| 20/10 | ADR-0001 (phạm vi WIN, D1–D6) | **Giữ nguyên.** WIN vẫn là Fuel Discrepancy. Không mở COMMAND / WORKFLOW / EVENT / Agent / MCP / Python / framework chung trước khi đóng G7. Không thêm tính năng để cải thiện benchmark |
| 20/10 | ADR-0002 (D7–D17) | **Tạm chấp nhận**, có điều kiện; phân lớp từng quyết định ở ADR-0002 mục "Phân lớp" |
| 20/10 | Chi tiền / model thật | **Chưa cấp quyền.** Phải có hợp đồng G7 đóng băng, số lời gọi chính xác, ước tính chi phí trước |
| 20/10 | Đồng bộ git | Đã commit + push theo lệnh, kể cả `docs/v3/` (D-V3-08) |
| 20/10 19:4x | Chi tiền / model thật (thay dòng "Chưa cấp quyền" ở trên) | **Cấp có điều kiện, tối đa 15 USD cho toàn bộ G7**, chỉ khi đủ cả 8 điều kiện:<br>1. khoá có sẵn hợp lệ trong môi trường;<br>2. hợp đồng G7 đã FROZEN;<br>3. dự toán cuối ≤ 15 USD;<br>4. pilot + chạy chính cùng trong trần;<br>5. model chính Sonnet 5.5;<br>6. baseline và EACL cùng model;<br>7. lưu bằng chứng thô;<br>8. không bí mật trong đầu ra / git.<br>Kiểm 19:37: khoá **chưa có** ⇒ chưa gọi lời nào, chi 0 USD |
| 20/10 19:4x | Toàn bộ câu ở mục 1–3 | Đã trả lời từng câu ở cột "Trả lời". Luật chạy đêm 20→21/10: không tối ưu tỉ lệ đạt, tối ưu độ tin của kết luận; FAIL sạch > PASS bẩn; tự quyết việc đảo được và ghi ADR |
| còn mở | — | D-G7-8 (khoá: chủ môi trường đặt biến), **D-G7-12 (mới, 20/10 đêm: `04` hay D7)**, D-V3-04 (hoãn tới R6), D-OP-3 (giấy phép, hoãn). D-G7-7 do agent tự đóng băng khi đủ điều kiện |

## 1. G7 — chặn lần chạy model thật (làm trước)

Bối cảnh: `docs/g7/CHECKPOINT-A-H.md`. Hợp đồng đề xuất: `docs/g7/HOP-DONG-G7.md`. Chi phí: `docs/g7/CHI-PHI-G7.md`. Bản soát độc lập D5: `docs/g7/REVIEW-D5.md`.

| Mã | Câu | Lựa chọn | Khuyến nghị (vì sao) | Chưa quyết thì | Trả lời |
|---|---|---|---|---|---|
| **D-G7-1** | **D5 đòi temperature 0, nhưng model hiện hành trả lỗi 400 với mọi temperature khác mặc định.** Sửa D5 thế nào? (ADR-0005) | **A** model hiện hành, không gửi sampling, cố định thinking + effort, k = 3 + liệt kê ca chập chờn · **B** model đời cũ còn nhận temperature 0 · **C** như A nhưng k = 5 | **A**: kết luận đúng với model dùng thật; B đo trên đời model sắp ngừng; C tốn thêm khoảng 67% lời gọi mà ít thêm giá trị | G7 **không chạy được**: mọi lời gọi lỗi 400 | **A** (20/10 tối). Không gửi temperature; cố định mọi tham số provider cho phép; k = 3 trước pilot; pilot cho thấy chập chờn đáng kể thì k = 5 **trước** khi đóng băng; không đổi k sau khi xem kết quả chính |
| **D-G7-2** | Model nào (chung cho mọi hệ) | `claude-haiku-5-5` · `claude-sonnet-5-5` · `claude-opus-5-5` | **Sonnet 5.5.** Model mạnh làm baseline mạnh, nên phép so khắt khe hơn với EACL; thắng nhờ baseline dùng model yếu thì ít giá trị. Chi phí ước tính vẫn nhỏ. Phiên V3 khuyên Haiku 5.5 vì rẻ (D-V3-01). Muốn xem độ nhạy theo model thì chạy cả hai, tổng < 15 USD | không chạy được | **`claude-sonnet-5-5`** cho mọi hệ (B1 / B2 / B2+ / B2″ / B2-oracle / EACL), không trộn model. Haiku chỉ cho báo cáo độ nhạy riêng nếu còn ngân sách |
| **D-G7-3** | Trần chi phí + cách gọi | trần (USD) · đồng bộ (có độ trễ từng lời gọi) hay Batch API (rẻ hơn 50%, mất độ trễ) | **Trần 15 USD nếu Sonnet, 2 USD nếu Haiku; chạy đồng bộ.** Ước tính B1 + B2: Sonnet ≈ 5–10 USD, Haiku ≈ 0,3–0,6 USD (`CHI-PHI-G7.md` mục 5). Đồng bộ để có độ trễ thật; tiền chênh không đáng kể | không chạy được | **Trần 15 USD** cho toàn bộ G7 (pilot + chính), gọi đồng bộ. Dự toán sau khi đóng băng > 15 USD thì dừng phần trả tiền và báo |
| **D-G7-4** | Có so với **B1** (LLM trực tiếp, chữ thô) không? Luật cửa sổ ngày của B1? | có (cửa sổ = ngày của ca ± 1 ngày; ca theo tháng = cả tháng; bảng kỳ của từng ca do người không đọc mã gán) · không | **Có.** V3 R1 đòi cả B1 lẫn B2; không có B1 thì kết luận cao nhất chỉ là PARTIAL. B1 chưa có mã: viết sau khi duyệt, không cần model | G7 chỉ so với B2 | **Có B1**, cùng model với EACL. B1 = AI tối thiểu; B2+ = RAG đơn giản có năng lực; EACL = kiến trúc kiểm soát bao quanh |
| **D-G7-5** | Bộ ca **giữ lại**: EACL đã được sửa tới 117/117 trên chính bộ 117 ca | có: **≥ 96 ca** (12 / lớp, lý tưởng 160), người không đọc mã **và pack** soạn, kết luận G7 trên bộ này · không: kết luận trên 117 ca, ghi rõ thiên lệch | **Có, ≥ 96 ca** (bản soát Q1, Q2): với 40 ca mỗi lớp chỉ 5 ca và 1 ca = 20 điểm, luật D5 vô nghĩa. Tốn thêm khoảng 288 lời gọi mỗi baseline | kết luận trên 117 ca, kèm cảnh báo thiên lệch | **Có bộ giữ lại độc lập**, mục tiêu ≥ 96 ca nếu soạn độc lập thật được. Độc lập + đa dạng > số lượng; cấm nhân bản / diễn đạt lại cho đủ số. Soạn → đóng băng → băm → commit **trước** khi chạy. Ca đã dùng để sửa mã thành regression |
| **D-G7-6** | Cách chấm theo **nội dung** (`G7ContentReport`), baseline top-k = 10, cách viết đoạn của baseline | duyệt nguyên · duyệt kèm các sửa ở D-G7-9 mục 2, 6, 7 | **Duyệt kèm sửa.** Giữ chấm nội dung làm cổng: chấm đủ route / outcome thì baseline 0/117, D5 không bao giờ đỏ. Bản soát chứng minh bộ chấm có lỗ ở cả hai phía, và đoạn của baseline thiếu bí danh khiến trần của nó chỉ 4/13 và 5/13 ở hai lớp (Q3, Q8) | không chạy được | **Duyệt kèm sửa.** Sửa bộ chấm trước benchmark thật; baseline cổng = B2+ |
| **D-G7-7** | **Duyệt đóng băng hợp đồng G7** (sau khi trả lời 1–6 và 9–11) | duyệt · sửa | duyệt; ghi sha256 + revision vào mục 13 của hợp đồng | không chạy được | **Chưa duyệt ngay; ủy quyền agent tự đóng băng** khi đủ: D-G7-1…11 đã áp; kiểm toán nhiễm xong; bộ giữ lại đóng băng + băm; test bộ chấm xanh; biên B2+ đóng băng; pilot xong (nếu dùng để chọn k); harness xuất được bằng chứng thô; dự toán ≤ 15 USD. Đủ thì ghi sha256 + revision, commit, chạy tiếp |
| **D-G7-8** | Cấp khoá model (sau D-G7-7) | đặt biến môi trường `EAIS_MODEL_API_KEY` ở máy chạy; không qua repo, không qua chat | — | không chạy được | Khoá chỉ lấy từ biến môi trường `EAIS_MODEL_API_KEY` do chủ môi trường cấp; cấm tìm ở repo / chat / lịch sử / dự án khác. Không có biến thì G7 trả tiền **bị chặn**, làm tiếp phần offline. **Kiểm 19:37: chưa có biến** |
| **D-G7-9** | Gói đề xuất của bản soát độc lập (`REVIEW-D5.md`, bảng cuối; mục 15 do phiên chính thêm) | chọn từng mục (bảng 1b) | xem bảng 1b | hợp đồng giữ nguyên bản hiện tại, với các lỗ đã biết | Theo bảng 1b (cột Trả lời) |
| **D-G7-10** | **Điểm yếu tổng quát hoá đã thấy:** câu đồng nghĩa "dùng" ("ngốn", "tiêu hao", "chạy hết") ra 0/4. Hệ trả lời tự tin bằng **dầu kho xuất** (3.850) thay vì tiêu thụ đo được (3.782); câu trả lời có ghi nhãn chỉ số nên không giấu giả định, nhưng vẫn sai nghĩa (bản soát Q1) | **a** không sửa trước G7, ghi là điểm yếu đã biết, để bộ giữ lại đo · **b** thêm từ đồng nghĩa trước khi soạn bộ giữ lại · **c** đổi chính sách pack: câu dầu chung chung **hỏi lại** thay vì mặc định kho xuất | **a.** Sửa bây giờ là khớp đề theo câu của người soát, đúng thứ bị cấm ("không thêm tính năng để cải thiện benchmark"). **c** là quyết định nghiệp vụ, đáng cân nhắc **sau** G7 | — | **a.** Không sửa đồng nghĩa trước benchmark; để bộ giữ lại đo; trượt thì báo trượt |
| **D-G7-11** | **Ba kỳ vọng "trọng tài" sau khi đo:** 083 (bỏ cấm "9.999"), 107 (thiếu vật tư: hỏi lại → trả lời theo D7), 108 (thiếu kỳ: hỏi lại → mặc định hôm nay) | **a** duyệt trọng tài như quyết định nghiệp vụ, ghi ngoại lệ G6 · **b** khôi phục kỳ vọng gốc: EACL trượt 3 ca, `ambiguity` 11/13, `unsupported_answer` 13/14, nên G2, G5 chưa đạt · **c** loại 3 ca khỏi phép tính cổng (còn 114 ca, EACL vẫn đạt hết), báo cáo độ nhạy riêng | **c.** Giữ trọn tính độc lập của bộ cổng mà không phải giả vờ ba ca kia chưa từng bị sửa. **a** cũng hợp lý nếu anh/chị coi D7 và "kỳ mặc định hôm nay" là quy ước nghiệp vụ đã duyệt | G2, G5, G6 ghi "đạt có điều kiện" | **c + bằng chứng thay thế.** 083 / 107 / 108 ra khỏi phép tính cổng độc lập cũ, giữ trong regression, báo độ nhạy, không xoá lịch sử. Loại 3 ca **không** tự làm G2 / G5 / G6 đạt vô điều kiện: phải có bộ giữ lại độc lập đóng băng chứng minh |
| **D-G7-12** | **Mâu thuẫn ngữ nghĩa cổng do kiểm toán nhiễm tìm ra** (`docs/g7/CONTAMINATION-AUDIT.md`, 20/10 đêm): `docs/04` dòng 83 xếp "thiếu loại vật tư" vào `CLARIFICATION_REQUIRED`, còn ADR-0002 D7 (chốt sau khi đã thấy ca 107 trượt) trả lời mặc định là dầu. Quy ước "kỳ mặc định hôm nay" (ca 108) không có dòng ADR nào | **a** giữ `04`: thiếu vật tư thì hỏi lại, bỏ D7 · **b** giữ D7, sửa `04` cho khớp, ghi rõ D7 là quy ước nghiệp vụ được chủ dự án duyệt **sau** khi đã thấy kết quả · **c** để chủ nghiệp vụ thật quyết | **a hoặc c.** D7 được chốt để một ca đang đỏ thành xanh. Muốn giữ D7 thì phải là quyết định nghiệp vụ có chủ, không phải của phiên viết mã. **Trong lúc chờ:** bộ giữ lại soạn theo thứ tự ưu tiên `04` > ADR-0002 (ADR-0006 §2). Ca nào rơi vào chỗ này mà EACL trượt thì báo riêng là "xung đột chính sách 04 / D7", kèm độ nhạy, không tính là lỗi mã | — | |

### 1b. Gói đề xuất của bản soát (D-G7-9)

Chiều: **khắt khe hơn** = khó đạt hơn cho EACL.

| # | Đề xuất (tóm tắt) | Chiều | Khuyến nghị của phiên chính | Trả lời |
|---|---|---|---|---|
| 1 | Bộ giữ lại ≥ 96 ca, kết luận chỉ trên bộ này; cấm sửa EACL, baseline, bộ chấm, prompt sau khi đóng băng | khắt khe hơn | có (= D-G7-5) | có |
| 2 | **B2+**: đoạn có bí danh + mã theo nguồn; câu hỏi viết lại theo đồng hồ hoặc có ngày hiện tại. B2+ làm baseline cổng, B2 cũ chỉ báo cáo | khắt khe hơn | có: đây là sửa lỗi "hình nộm" | có |
| 3 | **B2″** = B2+ cộng luật miền của pack (thẩm quyền, dung sai, mặc định) viết thành chữ trong prompt, để tách "tri thức" khỏi "kiến trúc" | khắt khe hơn | có, **chỉ báo cáo** ở lần chạy đầu: nó trả lời câu then chốt của luận đề mà không đổi cổng | có, **chỉ báo cáo** |
| 4 | B1: "bản ghi thô" gồm cả danh mục thực thể `entities.json` | khắt khe hơn | có | có |
| 5 | B2-oracle (đưa đúng đoạn bằng chứng), chỉ báo cáo, tách lỗi truy xuất khỏi lỗi thiếu tầng kiểm soát | báo cáo | có | có, **chỉ báo cáo** |
| 6 | Câu trả lời rỗng của model coi là từ chối, cho cả hai hệ | khắt khe hơn | có | có |
| 7 | Chuẩn hoá số trước khi so chuỗi cấm; giữ tiêu chí chữ là nội dung thật ("24 giờ", "2 ngày", "Không."); bộ đếm "không bịa số" từng ca cho cả hai hệ | sửa lỗi đo, đối xứng | có | có |
| 8 | Người chấm mù cho 43 ca không-trả-lời, rubric sáu nhãn, báo cáo riêng | trung tính | có nếu có người chấm; không thì để sau | chỉ báo cáo, và **chỉ khi có người chấm mù thật**; không giả lập bằng agent đã đọc mã |
| 9 | k = 5; lượt vô hiệu nếu đường tất định của EACL khác nhau giữa các lượt; `permission` và `unsupported_answer` tính ANY-RUN theo ca; bảng ca theo số lượt đạt | khắt khe hơn | có (thêm khoảng 67% lời gọi; ADR-0005 phương án C) | **theo pilot:** pilot quyết k = 3 hay 5 trước khi đóng băng |
| 10 | Kiểm gộp McNemar một phía p ≤ 0,01 trên bộ giữ lại; khoảng tin cậy từng lớp; lớp n < 12 không tính "lớp hơn" | khắt khe hơn | có | có nếu đủ điều kiện thống kê; không đủ thì ghi giới hạn, không giả ý nghĩa thống kê |
| 11 | Bỏ lớp `permission` (thắng theo thiết kế) khỏi phép đếm 4/8: cần ≥ 4 trong 7 lớp còn lại | khắt khe hơn | có | có |
| 12 | Ghi "mọi lượt" vào D5; thêm canary tên ("Cầu Z", "PRJ-Z", "def-build") và dạng chữ; ghi rõ ngày không được đếm | khắt khe hơn nhẹ | có | có |
| 13 | Báo cáo độ nhạy: bỏ 083 / 107 / 108; baseline không có tài liệu chèn lệnh; hai model | trung tính | có | có (báo cáo độ nhạy) |
| 14 | Bộ pilot 12–16 ca (ngoài dev và giữ lại) để đo chập chờn, hành vi "rỗng" trước khi chốt k và prompt. **Có gọi model thật** | trung tính | có; tính trong trần chi phí D-G7-3 | có (pilot), trong trần 15 USD; **không** làm bằng chứng cuối |
| 15 | Soát rồi công khai chương trình đo của người soát (đang ở `docs/sources/`, máy gốc) để ai cũng tái lập được bảng Q3 | — | có, sau khi soát | có, sau khi soát bảo mật / riêng tư |

## 2. EACL V3 — gói định nghĩa R0

Do phiên lead V3 soạn. Chi tiết, mâu thuẫn C1–C19 và lý do: `docs/v3/05-R0-DEFINITION-FREEZE.md` mục 3–4. Dưới đây chép câu hỏi và khuyến nghị **của phiên đó**. Phiên viết tệp này **chưa soát độc lập** bộ `docs/v3/`.

| Mã | Câu | Khuyến nghị của phiên V3 | Ghi chú | Trả lời |
|---|---|---|---|---|
| D-V3-00 | Duyệt hướng V3: tên **EACL**, MVP chuyển sang **Customer Ops** (hiểu + hành động trên app tham chiếu), Fuel thành R1 | (ngầm) duyệt gói R0 | quyết định lớn nhất; các câu dưới phụ thuộc | **Duyệt.** Hướng EACL — Enterprise AI Control Layer; Customer Ops là MVP / tích hợp tham chiếu tiếp theo; Fuel giữ vai Truth Benchmark (R1); không xoá lịch sử EAIS |
| D-V3-01 | Model thật + trần chi phí cho R1 / R3 / R6 | Haiku 5.5; ước token trước khi chi | trùng D-G7-2 / D-G7-3 | G7 dùng Sonnet 5.5 (D-G7-2). Pha R3 / R6 không tự thừa kế: chọn model nhỏ nhất đủ dùng lúc benchmark đó |
| D-V3-02 | Ngưỡng các cổng V3 (§2.5) | duyệt hoặc sửa số | Fuel: giữ D5, áp cả B1; WIN / PARTIAL / FAIL | **Chưa** đóng băng ngưỡng V3 từ Fuel. Fuel giữ hợp đồng riêng; ngưỡng Customer Ops / R3 / R4 / R6 suy từ rủi ro nghiệp vụ và tiêu chí nghiệm thu của story; không chép D5 máy móc |
| D-V3-03 | Đổi tên mã và repo sang EACL | (a) tài liệu + README ngay; (b) module / thư mục / repo trong một commit riêng trước mã R2 | (b) đụng remote công khai | **(a) trước:** đổi tên trong tài liệu / sản phẩm nơi hợp lý. **Không** đổi tên repo / remote / module Go trong đêm; đổi tên mã là mẻ riêng sau khi bằng chứng ổn định |
| D-V3-04 | Đo B0 (người làm tay) | quyết lúc chuẩn bị R6 | | Hoãn tới R6 |
| D-V3-05 | Agent con dùng Opus? | Không | đúng luật hiện hành | Không dùng agent con Opus mặc định; model nhỏ nhất đủ dùng |
| D-V3-06 | Sửa `CLAUDE.md` §9 / §24 / §25 theo V3 | chủ dự án tự sửa theo câu chữ đề xuất | tệp của chủ dự án | Được sửa `CLAUDE.md` cho khớp V3 nếu: không xoá luật lịch sử còn cần; diff rõ; commit riêng; có ghi chú tương thích |
| D-V3-07 | Việc treo: ADR-0002, đường dẫn nội bộ ở `docs/02`, danh sách P10, `go mod tidy` | dọn trước | ADR-0002 **đã tạm chấp nhận**. `docs/02`: chỗ đường dẫn monorepo đã thay; chỗ còn lại là đường dẫn tương đối bên trong mã cũ, cần cho bảng đối chiếu | Dọn nợ an toàn, đảo được, **trước** khi đóng băng G7 nếu nó ảnh hưởng tái lập / bằng chứng. Không refactor hành vi cho đẹp trước benchmark |
| D-V3-08 | Commit / push `docs/v3/` | khi chủ dự án nói | **Đã làm** 20/10 tối theo lệnh "commit lên hết git" | — |
| D-V3-09 | Kiến trúc app tham chiếu (ADR-0003) | tiến trình riêng, DB riêng, REST `net/http`, cấm import hai chiều | | **Duyệt hướng ADR-0003** (tiến trình riêng, DB riêng, REST `net/http`, cấm import hai chiều). **Chưa** implement khi G7 / Fuel còn là đường găng |
| D-V3-10 | Ghi đầu `04`, `05`, `catalog.yaml` là "lịch sử EAIS" | có | | Có. Đánh dấu tài liệu lịch sử EAIS; không viết lại lịch sử |

## 3. Vận hành và repo

| Mã | Câu | Lựa chọn | Khuyến nghị (vì sao) | Trả lời |
|---|---|---|---|---|
| D-OP-1 | Thứ riêng tư không lên repo public: `docs/sources/`, `*.local.md`, memory | giữ ở máy · repo **riêng tư** riêng · chép qua kênh riêng (scp) | **Repo riêng tư riêng** nếu cần dùng ở máy khác; không thì giữ ở máy. Không bao giờ đưa vào repo public | Giữ ở máy; không đưa lên repo public; chưa tạo repo riêng tư trong đêm |
| D-OP-2 | `go mod tidy` (máy gốc offline nên chưa chạy) | chạy ở máy có mạng, commit diff riêng | có. Chỉ sửa nhãn `// indirect`, không đổi bản | Có: chạy khi có mạng; chỉ đổi metadata thì commit riêng; đổi bản / hành vi ngoài dự kiến thì dừng mẻ đó và báo |
| D-OP-3 | Giấy phép cho repo public | Apache-2.0 · MIT · không giấy phép (mặc định giữ mọi quyền) | quyết trước khi chia sẻ rộng. Chưa chọn = người khác không có quyền dùng lại mã | Hoãn; agent không tự chọn giấy phép |
| D-OP-4 | Chính sách commit / push của agent từ nay | chỉ khi được bảo (như hiện nay) · tự commit ở mốc checkpoint, push khi được bảo | giữ "chỉ khi được bảo" | **Ngoại lệ đêm 20→21/10:** agent tự commit + push mốc an toàn lên `main` khi test liên quan xanh, không bí mật / dữ liệu riêng, commit mạch lạc, không rewrite / force / xoá lịch sử. Sau đêm quay về luật cũ |
| D-OP-5 | Hai việc dừng dở lúc 10:33: soát độc lập mã 20/10; bộ ca đội đỏ | chạy lại trước G7 · sau G7 · bỏ | **Trước G7**: soát độc lập không tốn model. Bộ ca đội đỏ chính là bộ giữ lại của D-G7-5 | **Trước G7:** chạy lại soát độc lập mã. Ca đội đỏ chỉ là nguồn ý tưởng cho bộ giữ lại / ca phá; người viết mã không được xem kỳ vọng bộ giữ lại rồi chỉnh mã |
| D-OP-6 | Nợ đặt chỗ D9: từ dừng / cụm nghi vấn tiếng Việt viết cứng trong `intelligence/rag` | giữ, có nhãn nợ · dời sang lexicon / pack trước khi đóng băng G7 | **Giữ.** Chưa có tenant ngôn ngữ khác; dời bây giờ đổi hành vi truy xuất ngay trước benchmark | Giữ nợ có nhãn; không dời từ dừng / cụm nghi vấn trước benchmark nếu việc đó đổi hành vi |
| D-OP-7 | `CLAUDE.md` có danh sách nhóm lỗi P10 lệch `CONVENTIONS` §4.2 | chốt danh sách nào | chủ dự án chốt (tệp của chủ dự án) | Không tự chốt khi `CLAUDE.md` và `CONVENTIONS` mâu thuẫn: kiểm kê lệch, đề xuất danh sách chuẩn, đo ảnh hưởng, ghi ADR đề xuất. Không ảnh hưởng benchmark / runtime thì chọn tập lớn hơn, đảo được; ảnh hưởng ngữ nghĩa cổng thì `BLOCKED_FOR_OWNER`. **Đã làm 20/10 đêm:** đo được không mã / bộ ca nào dùng ⇒ ADR-0008, `CONVENTIONS` §4.2 thành tập hợp 27 nhóm, `CLAUDE.md` không sửa |
