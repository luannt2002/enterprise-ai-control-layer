# ADR-0007: Bộ chấm nội dung G7 v2 — hướng thiết kế (ĐỀ XUẤT, chưa đóng băng)

- **Trạng thái:** ĐỀ XUẤT. Chưa hiện thực, chưa đóng băng. Ba câu chờ chủ dự án: D-G7-15, D-G7-16, D-G7-17 (`docs/CHO-QUYET-DINH.md`).
- **Ngày:** 21/10/2026, rạng sáng.
- **Liên quan:** ADR-0005 §4–§6 (chấm từng câu, D5 sửa đổi), D-G7-6 (sửa bộ chấm TRƯỚC benchmark thật), D-G7-9 (gói đề xuất của bản soát D5).
- **Đặc tả đầy đủ:** `docs/g7/BO-CHAM-V2-DAC-TA.md` (nguyên văn của giám khảo).

## Bối cảnh

- Bộ chấm nội dung v1 (`evals/g7.go`: `G7ContentReport` + `G7Counter`) bị một người tấn công độc lập phá: 170 đòn, 109 câu SAI mà ĐẠT, 24 câu ĐÚNG mà TRƯỢT. Gốc là "túi số": không phân cực, không ràng buộc thực thể / chỉ số / kỳ / nguồn, không kiểm đơn vị, không kiểm lịch.
- Đêm 20→21/10 đã vá phần chắc chắn của v1: 170 đòn thành test ratchet, rỗng = từ chối, hệ lỗi khi hỏi luôn trượt, số được phép từng ca, canary tên. Nợ còn **118** đòn. v1 KHÔNG dùng làm bằng chứng G7.
- Quy trình thiết kế v2:
  - hai người thiết kế độc lập, không đọc mã hay pack EACL, không mở bộ giữ lại;
  - một giám khảo chấm hai bản, chạy lại số đo, tự soạn câu thử ngoài mẫu, rồi tổng hợp đặc tả.

## Quyết định

1. **Nền là bản 1:** MỘT hàm chấm thuần, tất định, mù tên hệ: `Score(ca, câu trả lời, hồ sơ chính sách)` → PASS / FAIL / INDET kèm danh sách luật đã kích hoạt.
   - Hàm không nhận route, trích dẫn, `model_calls`: chấm nội dung thuần (ADR-0005 §4).
   - Luật xếp theo tầng: đầu vào CompareG7, chuẩn hoá hai phía, đọc số kép, đơn vị, lịch thật, thực thể, gắn claim (thực thể / chỉ số / nguồn / kỳ) vào từng con số rồi đối chiếu cơ sở sự kiện của bộ sinh, phân cực suy từ câu hỏi, rubric theo outcome, bộ đếm độc lập cho MỌI hệ.
   - Ghép có chọn lọc từ bản 2 những ý không đổi đối tượng đo, cộng 8 sửa của giám khảo (đặc tả §0).
2. **Bản 2 KHÔNG làm cổng.** Bản 2 là giao thức JSON có kiểu cho mọi hệ. Lý do:
   - nó buộc baseline khai thực thể / chỉ số / kỳ / nguồn, tức đổi đối tượng đo (ADR-0005 §2 cấm ở mức hệ);
   - nó đảo ADR-0005 §4;
   - EACL phải đi qua adapter do người biết EACL viết, nên không độc lập;
   - đo trên văn xuôi tự nhiên, luật "mọi số phải khai" trượt oan 7/14.
3. **Điều kiện đóng băng v2** (thiếu một điều kiện là không đóng băng):
   - hiện thực trong `evals/` không import gói EACL;
   - bảng 170 đòn đạt mức đặc tả §7.2;
   - hiệu chỉnh trên đầu ra pilot THẬT, có người chấm mù (D-G7-16);
   - tỉ lệ trượt oan và đạt nhầm ngoài mẫu ≤ 2 %;
   - băm mã chấm + danh sách từ + hồ sơ chính sách ghi vào mọi Report.
4. **Trước khi v2 đóng băng:** `g7run` vẫn dùng v1 đã vá, CHỈ để thử ống nối (chạy khô). Không rút kết luận G7 nào từ v1.

## Số đo (nhãn nguồn như đặc tả)

| Số | Nguồn |
|---|---|
| Bản 1 sửa 122/123 đòn lệch không tranh cãi; còn lệch 8/170. Đây là số TRONG MẪU: bộ chấm được chỉnh trên chính các đòn này | người thiết kế 1, giám khảo chạy lại khớp |
| Đáp án vàng 117/117 đạt; đột biến 12 toán tử 783/783 bị chặn; 0/1.620 phép đổi biểu diễn số làm lật phán quyết | như trên |
| **Ngoài mẫu**, lần nhìn đầu trên 230 câu giám khảo tự soạn: trượt oan 6/130 = 4,6 % (Wilson 2,1–9,7), đạt nhầm 9/100 = 9,0 % (4,8–16,2). **Vượt ngưỡng 2 %** | giám khảo đo |
| Bỏ ràng buộc thực thể / kỳ / chỉ số-nguồn ⇒ 31/170 lệch: phần hiểu ngôn ngữ là cần thiết, không cắt được | giám khảo đo |
| Mã nguyên mẫu ≈ 5,6 nghìn dòng; bản chính thức ước 10–15 ngày công | ƯỚC, chưa đo |
| Tỉ lệ INDET, rỗng, luật kích hoạt theo hệ, độ lệch thiên vị trên đầu ra thật của `claude-sonnet-5-5` | CHƯA ĐO (cần khoá + pilot) |

## Hệ quả

- **G7 vẫn CHẶN**, vì ba việc độc lập nhau:
  - khoá model (D-G7-8);
  - pilot thật, để hiệu chỉnh bộ chấm và chọn k;
  - hiện thực + hiệu chỉnh + đóng băng v2.
- **Mã nguyên mẫu giữ ở máy**, không lên git: `docs/sources/g7-scorer-v2-proto/` là thư mục riêng tư. Chỉ công khai sau khi đã soát.
- **Rủi ro đã biết** (đặc tả §10):
  - danh sách từ do một người soạn sau khi thấy 170 đòn;
  - INDET đóng thành FAIL có thể đánh vào hệ viết tự do nhiều hơn mẫu cố định.
  - Cách xử lý: kiểm bất đối xứng luật theo hệ và báo cáo khoảng bốn ô STRICT / LENIENT. Kết luận đổi dấu trong khoảng đó thì báo "không kết luận được" (D-G7-17).
- **Việc kế** khi chủ dự án trả lời D-G7-15…17: hiện thực theo bậc T0a → T0b → T1 (đặc tả §9), chỉ đổi `evals/` và harness. `git diff` trên `core intelligence data reliability platform observability` phải rỗng.
