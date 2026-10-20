# ADR-0005 — Hợp đồng G7 và D5 sửa đổi (bỏ temperature 0)

- **Ngày:** 20/10/2026.
  - Đề xuất 18:5x (commit `b882172`).
  - Nối bản soát độc lập 19:2x (`94bbaae`).
  - **Chốt 20:1x** theo quyết định của chủ dự án.
- **Trạng thái:** **ĐÃ DUYỆT** (chủ dự án 20/10 19:4x, `docs/CHO-QUYET-DINH.md` D-G7-1…11). Hợp đồng thi hành `docs/g7/HOP-DONG-G7.md` chỉ chuyển FROZEN khi đủ điều kiện D-G7-7. Chưa đủ thì **không** gọi model thật.
- **Liên quan:**
  - ADR-0001 D5, D6;
  - ADR-0006 (bộ giữ lại độc lập, pilot);
  - `docs/g7/REVIEW-D5.md`, `docs/g7/CHI-PHI-G7.md`;
  - `docs/v3/05-R0-DEFINITION-FREEZE.md` §2.4–2.5.
- **Số hiệu:** ADR-0003 và ADR-0004 đã được bộ tài liệu V3 giữ chỗ.

## Bối cảnh

1. **Temperature 0 không chạy được.**
   - D5 (ADR-0001) đòi "temperature 0; k = 3 lần, trung vị".
   - Tài liệu chính thức của nhà cung cấp (Thinking, tra ngày 20/10/2026) ghi:

     > "On … Claude Opus 5.5, … Claude Sonnet 5.5, Claude Sonnet 5, and Claude Haiku 5.5, non-default `temperature`, `top_p`, or `top_k` values return a 400 error on every request, regardless of whether thinking is used."

   - Client `models.Messages` đang gửi `temperature: 0`, nên với model hiện hành mọi lời gọi lỗi 400. Lần chạy khô ngày 20/10 qua được chỉ vì máy chủ giả không kiểm tham số.
2. **Bản soát độc lập** (`docs/g7/REVIEW-D5.md`) chỉ ra bốn điểm:
   - D5 thiên vị EACL, vì 117/117 là kết quả trong mẫu;
   - baseline B2 bị "què": đoạn thiếu bí danh, nên trần của nó chỉ 4/13 ở lớp `entity` và 5/13 ở `ambiguity`;
   - trung vị 3 lượt che được ca chập chờn;
   - bộ chấm có lỗ ở cả hai phía.
3. **Luật của chủ dự án:** muốn đổi D5 thì ghi ADR **trước** khi chạy. Không đổi sau khi thấy kết quả.

## Quyết định

### 1. Model và tham số

- **Một model cho mọi hệ: `claude-sonnet-5-5`.**
  - Không dùng model khác nhau giữa baseline và EACL.
  - Haiku 5.5 chỉ cho báo cáo độ nhạy riêng, nếu còn ngân sách; không trộn vào kết quả chính.
- **Không gửi** `temperature`, `top_p`, `top_k`.
  - Lần chạy model là **NGẪU NHIÊN**; không tuyên bố là tất định.
  - Tham số mặc định của API **không** làm lần chạy tất định.
- **Cố định mọi thứ provider cho phép, ghi vào manifest mỗi lần chạy:**
  - model;
  - thinking (Sonnet 5.5: `{"type": "between_tools"}`, theo tài liệu tra 20/10);
  - effort (`medium`);
  - `max_tokens`;
  - system prompt (băm sha256);
  - top-k và cách dựng ngữ cảnh.
- **Thử khói trước lần chạy:** tham số nào bị từ chối thì sửa **trước** khi đóng băng hợp đồng, kèm ghi chú sửa đổi ở đây.
- **Gọi đồng bộ** để có độ trễ thật.
- **Trần chi phí 15 USD** cho toàn bộ G7 (pilot + chính). Dự toán sau đóng băng vượt trần thì không chạy phần vượt.

### 2. Các hệ được so

| Hệ | Vai | Là gì |
|---|---|---|
| EACL | hệ được chấm | luồng S0–S10 |
| **B1** | baseline cổng | LLM trực tiếp, **một** lời gọi mỗi câu hỏi:<br>- bản ghi thô của tenant trong cửa sổ ngày của ca (`b1_window`, do người soạn ca gán, không phải phiên viết mã);<br>- `entities.json`.<br>Không truy xuất |
| **B2+** | baseline cổng | RAG đơn giản **có năng lực**: như B2, cộng hai thứ:<br>- mỗi đoạn có tên hiển thị + bí danh + mã theo nguồn của thực thể (lấy từ `entities.json`);<br>- câu hỏi kèm ngày hiện tại |
| B2 | chỉ báo cáo | baseline cũ: đoạn thiếu bí danh |
| B2″ | chỉ báo cáo | B2+, cộng luật miền của pack (thẩm quyền, dung sai, mặc định) viết thành chữ trong prompt. Mục đích: tách phần thắng do "tri thức" khỏi phần thắng do "kiến trúc" |
| B2-oracle | chỉ báo cáo | B2+ nhưng nhận đúng đoạn bằng chứng của ca. Mục đích: tách lỗi truy xuất khỏi lỗi thiếu tầng kiểm soát. Chỉ chạy được khi hồ sơ ca có id bản ghi |

**Chung cho mọi baseline:**
- cùng model, cùng tham số, cùng system prompt với lời gọi model của EACL;
- một chỉ mục = một tenant: dữ liệu tenant khác không vào baseline;
- câu trả lời rỗng của model ghi là **từ chối**.

**Biên của B2+ (đóng băng cùng hợp đồng):**
- **Được** những tiền xử lý tối thiểu mà một RAG hợp lý cần để đọc cùng dữ liệu: gắn bí danh / mã nguồn vào đoạn (dữ liệu gốc), biết ngày hiện tại.
- **Không được** bất kỳ thứ gì thuộc tầng kiểm soát EACL:
  - xếp nguồn theo thẩm quyền;
  - luật dung sai;
  - ràng buộc claim;
  - lọc quyền theo người hỏi;
  - chọn nghĩa nghiệp vụ;
  - phép tính tất định.

Câu hỏi mà G7 trả lời là: **EACL so với một RAG đơn giản nhưng có năng lực**, không phải so với một RAG cố tình bị làm hỏng.

### 3. Bộ ca

- **Kết luận G7 chỉ trên bộ giữ lại độc lập** (ADR-0006):
  - dùng H1 nếu H1 còn sạch;
  - H1 đã bị dùng để sửa mã thì dùng H2.
- **117 ca** (nay là REGRESSION) chỉ báo cáo. Kèm báo độ nhạy khi bỏ 083 / 107 / 108.
- **Pilot 16 ca** (ADR-0006) chỉ để chọn k và kiểm tương thích; **không** là bằng chứng G7.

### 4. Chấm từng câu trả lời: một tiêu chí cho mọi hệ

- **Sửa bộ chấm TRƯỚC khi đóng băng**, theo các đòn của người tấn công bộ chấm. Mỗi đòn có một test hồi quy. Các sửa đổi:
  - chuẩn hoá số trước khi so chuỗi cấm;
  - giữ tiêu chí chữ là nội dung thật ("24 giờ", "2 ngày", "Không.");
  - câu rỗng = từ chối: đạt ở ca kỳ vọng từ chối / ngoài phạm vi, trượt ở ca khác;
  - bộ đếm "không bịa số" từng ca, áp cho mọi hệ.
- **Vẫn bỏ qua nhãn kiến trúc của EACL:** route, outcome, `model_calls`, trích dẫn có cấu trúc.
- **Sau khi đóng băng: không đổi bộ chấm.**

### 5. Tổng hợp và luật D5 sửa đổi

Mỗi hệ chạy k lượt trên cùng bộ ca.

- **Lượt hợp lệ:**
  - đường tất định của EACL (route, outcome, id bằng chứng) phải giống nhau giữa các lượt;
  - khác thì lượt đó vô hiệu, chạy lại một lần; vẫn khác thì G7 **FAIL** (tất định hỏng);
  - lỗi provider còn lại sau thử lại: ca đó ghi lỗi `PROVIDER`, không tính đạt.
- **Điểm từng ca:** `pass_count / k`.
  - Ca có `0 < pass_count < k` là **CA CHẬP CHỜN** (FLAKY MODEL CASE), liệt kê riêng cho từng hệ.
  - Không nén hành vi ngẫu nhiên vào một trung vị.
- **Điểm lớp:**
  - trung vị qua k lượt của số ca đạt (như D5 cũ);
  - riêng `permission` và `unsupported_answer`: ca chỉ tính đạt khi đạt ở **mọi** lượt (ANY-RUN), áp cho mọi hệ.
- **R1 — Không thua:** ở mọi lớp, EACL ≥ từng baseline cổng (B1, B2+). Hoà được phép.
- **R2 — Hơn rõ:**
  - với từng baseline cổng, ≥ 4 trong 7 lớp có EACL − baseline ≥ 20 điểm %;
  - 7 lớp là 8 lớp trừ `permission` (lớp thắng theo thiết kế);
  - lớp có n < 12 ca không được tính là "lớp hơn".
- **R3 — Bất biến cứng của EACL:** mọi lượt, mọi ca; một vi phạm là **FAIL**.
  - Rò rỉ tenant = 0. Canary dạng số, mã và tên của tenant khác: `9999`, `B-CANARY-7731`, `Cầu Z`, `PRJ-Z`, `def-build`. Ngày tháng không đếm.
  - Số không căn cứ = 0 (bộ đếm độc lập).
  - Lộ số được bảo vệ ở ca kỳ vọng từ chối = 0.
- **R4 — Thống kê, nếu đủ điều kiện:**
  - McNemar chính xác một phía trên từng ca (đạt = đạt ở đa số lượt), EACL vs B2+ và EACL vs B1, p ≤ 0,01;
  - khoảng tin cậy Wilson 95% cho từng lớp;
  - số cặp lệch < 10 thì R4 "không đủ điều kiện": ghi giới hạn, không chặn, **không** tuyên bố có ý nghĩa thống kê.
- **k:** 3 trước pilot. Luật chọn k từ pilot được đăng ký trước ở ADR-0006 §6. Không đổi k sau khi đã xem kết quả chạy chính.

### 6. Kết luận

| Kết luận | Điều kiện |
|---|---|
| **G7 PASS** | R1–R3 đạt với **cả** B1 và B2+; đủ lượt hợp lệ; R4 đạt nếu đủ điều kiện |
| **G7 PARTIAL** | đạt với một baseline cổng mà không đạt với baseline kia; hoặc phải cắt hệ vì trần chi phí |
| **G7 FAIL** | R3 vi phạm; hoặc R1 / R2 hỏng với B2+ |

- **WIN (sản phẩm)** = G1–G6 đạt vô điều kiện + G7 PASS + scale smoke.
- G1–G6 đạt mà G7 FAIL thì "ARCHITECTURE PROOF" có thể đạt, còn "PRODUCT VALUE WIN" là FAIL / PARTIAL.
- **Không đổi định nghĩa WIN để dự án đạt.**

### 7. Sau khi chạy

- **G7 FAIL:**
  - không sửa ngưỡng;
  - không sửa mã rồi chạy lại cùng bộ giữ lại;
  - phân loại tầng sai đầu tiên cho từng ca hỏng;
  - ca đã thấy trượt thành REGRESSION.
- **G7 PASS:** đóng băng "Fuel Truth Benchmark v1", gồm revision, băm hợp đồng / dữ liệu / bộ ca, bằng chứng thô, chi phí, lệnh tái lập.
- **Đổi bất kỳ điều gì của hợp đồng sau khi đóng băng** = thí nghiệm mới: ADR mới + chạy lại toàn bộ. Đổi model cũng vậy.

## Phương án đã cân nhắc (ngày 18:5x)

| Phương án | Là gì | Kết cục |
|---|---|---|
| **A** | model hiện hành, sampling mặc định, cố định thinking + effort, liệt kê ca chập chờn | **chọn** (D-G7-1) |
| B | model đời cũ còn nhận temperature 0 | loại: kết luận trên đời model sắp ngừng |
| C | như A nhưng k = 5 cố định | thay bằng: chọn k theo pilot (ADR-0006 §6) |

## Hệ quả

- **Phải sửa mã trước lần chạy thật:**
  - client: bỏ `temperature`; `thinking` / `effort` lấy từ cấu hình; thử lại có lùi + ghi lần thử;
  - `g7run`: `-smoke`, `-out` (bằng chứng thô), B1, B2+, B2″, B2-oracle, luật tổng hợp mới;
  - bộ chấm: các sửa ở §4.
  Mọi thứ có test, chạy khô bằng máy chủ giả.
- **Thiếu temperature 0 thì kết quả giữa các lượt có thể khác nhau.** `pass_count / k` và danh sách ca chập chờn là cách đo độ khác nhau đó, không phải cách giấu nó.
