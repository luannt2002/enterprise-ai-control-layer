# Đo "người soát CHỈ ĐỌC VẾT chỉ đúng tầng sai đầu tiên" — lần 1 (22/10/2026 tối)

> Trên dữ liệu giả lập, chưa có người nghiệp vụ duyệt (D-RM-03).
> Đo vế còn thiếu của cổng ra M1 (`docs/ROADMAP-DAY-DU.md` §5b) và dòng "Vết / debug" của CỔNG TRƯỚC AI (`docs/PROMPT-BUILD-EAIS.md`
> §9.3): người soát CHỈ ĐỌC VẾT chỉ đúng tầng sai đầu tiên ở ≥ 90% ca đỏ của bộ 117 / H1 / H3.
> Nhãn: [ĐO] đã chạy lệnh · [REPO] tài liệu ghi · [SUY] suy luận.

## 1. Kết luận

**KHÔNG ĐẠT.** Người đọc mù chỉ đúng tầng sai đầu tiên ở **29 / 33 = 87,9%** ca đỏ của H1 + H3 (ngưỡng ≥ 90%). Con số này gần bằng mức
"luôn đoán S4" trên cùng tập (84,8%), nên vết hiện tại chưa đủ để phân biệt tầng. Bộ 117 không có ca đỏ.

| Tập | Khớp nhãn chuẩn | Mức "luôn đoán lớp đông nhất" |
|---|---|---|
| H1 + H3 (tập của tiêu chí) | **29 / 33 = 87,9%** | 84,8% (S4) |
| H4 (báo riêng, không thuộc tiêu chí) | 15 / 15 = 100% | 66,7% (S4) |

Bốn ca lệch, cả bốn là chỗ vết thiếu thông tin chứ không phải người đọc đọc sai chữ:

| Ca | Nhãn chuẩn | Người đọc | Vết thiếu gì |
|---|---|---|---|
| WIN-I-201 | S4 | S5 (tin cậy thấp) | S4 chỉ ghi "không nhận ra ý định"; không ghi ý định nào đã xét, khớp một phần, thiếu thuật ngữ nào — nên người đọc đổ cho S5 |
| WIN-I-538, 539, 543 | S4 | S5 (tin cậy vừa) | danh sách ứng viên hỏi lại chỉ có trong thân câu trả lời; vết không ghi ứng viên nào đến từ S3 (bí danh mơ hồ), ứng viên nào do S4 gộp thêm |

Người đọc tự ghi "vết thiếu" ở 29 / 48 thẻ; thứ được nhắc nhiều nhất: phương án route đã xét (S5), luật sinh đoạn và chỗ mơ hồ (S2), các ý
định đã xét (S4), nguồn của ứng viên, lý do của cổng phán quyết (S9). Đây đúng là ghi chú R2-03 của soát độc lập khối M1.

## 2. Cách đo

1. **Thẻ** [ĐO]: 48 ca đang đỏ ở mã `41bc3cf` (H1 12, H3 21, H4 15; danh sách lấy từ đầu ra winreport của cổng cuối 22/10). Một agent dựng
   thẻ chạy `eais serve` trên database PostgreSQL riêng (nạp cùng dữ liệu sinh seed 42), hỏi từng câu bằng token của principal của ca, rồi
   `eais trace -tenant <t> <trace_id>`. Mỗi thẻ: câu hỏi, vai người hỏi, khoá kỳ vọng của ca (chép nguyên văn), câu trả lời của server, bảng
   vết. Không có dòng "ĐỎ …" của winreport hay phân tích nào. Câu trả lời của server trùng câu của winreport ở 48 / 48 ca.
2. **Người đọc mù:** một agent chỉ được đọc 48 thẻ (không mã, không tài liệu, không phân tích), chọn bước sai đầu tiên S0–S10 cho từng thẻ.
3. **Nhãn chuẩn:**
   - H4: bảng "Kết quả H4" của ADR-0010 (người phân tích + người kiểm chéo độc lập đồng ý 13 / 13);
   - H1 + H3: một người phân tích độc lập khác, có đọc mã, không đọc các phân tích cũ; mỗi nhãn kèm phép thử chạy lại qua pipeline thật
     (sửa riêng tầng được gán, câu hỏi giữ nguyên, thì ca đạt — trừ 4 ca có lỗi thứ hai ở tầng sau, ghi tầng sớm nhất). Kết quả: S4 ở 28
     ca, cổng phán quyết ở 4 ca (H3-507, 564, 602, 650), S5 ở 1 ca (H3-548);
   - đối chiếu với phân tích H1 ngày 21/10 (`docs/g7/holdout/ket-qua-h1/phan-tich.md`): 10 / 12 ca khớp; H1-318 và H1-429 ngày 21/10 ghi S2,
     lần này ghi S4 kèm phép thử cho thấy lệch ở S2 không ảnh hưởng kết quả.
4. "Cổng phán quyết câu xác nhận" tính là S9 cho cả người đọc lẫn nhãn chuẩn.

## 3. Giới hạn

- Mẫu nhỏ: 33 ca; một người đọc.
- Chưa có **ca xanh làm đối chứng âm**: người đọc biết mọi thẻ đều là ca sai, nên số đo không nói gì về việc người đọc có "thấy lỗi" ở ca
  đúng hay không.
- Nhãn chuẩn của H1 + H3 là của một người phân tích (có phép thử); hai nguồn nhãn của H1 lệch ở 2 / 12 ca.

## 4. Tiếp theo

1. R2-03 (đang làm): vết xuất lý do và phương án đã xét / bị loại ở S2–S5, S7 và cổng phán quyết; không đổi câu trả lời nào (kiểm bằng bộ
   117, H1, H3, H4 hai đường).
2. Đo lại bằng một người đọc **mới**, trộn ca xanh làm đối chứng âm; báo cả tỉ lệ chỉ đúng tầng ở ca đỏ lẫn tỉ lệ nói "không sai" ở ca xanh,
   và mức "luôn đoán lớp đông nhất".
