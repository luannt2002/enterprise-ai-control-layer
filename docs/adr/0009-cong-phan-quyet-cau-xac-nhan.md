# ADR-0009 — Cổng phán quyết của câu xác nhận: chỉ nói "Đúng." / "Không." khi claim đã giải đủ chiều

- **Ngày:** 21/10/2026, ca ngày.
- **Trạng thái:** **ĐÃ CHỐT** theo hợp đồng vận hành ca 21/10 (mục câu xác nhận, CONFIRMATION SAFETY CONTRACT) — chỉ đạo của chủ dự án, đóng câu chờ quyết D-OP-10. Đảo được.
- **Liên quan:** `docs/04-core-mvp.md` §3 (câu trả lời mở đầu bằng phán quyết), §6 G3; báo cáo đêm 20→21/10 mục 8 (15 câu thăm dò); ADR-0006 §10 (H3).

## Bối cảnh

- **Đo trên `15ed135`:** 15 câu thăm dò đối kháng do người soát dựng đều bị phán quyết SAI (đầu đêm sai 8/15). 7 câu là hồi quy của ba bản sửa liên tiếp trong đêm (`3d730ea`, `6f23ff3`, `a8b0858`) ⇒ luật chống quay vòng (hợp đồng §19) BẮT BUỘC dừng vá, quay về bất biến.
- **Gốc lỗi (phân tích T1, vai B):**
  - phán quyết dựa vào "đúng MỘT con số ứng viên và 0 mốc nghi ngờ" — đó là bằng chứng về số lượng ứng viên, không phải về việc hiểu câu;
  - tập ứng viên dựng bằng LOẠI TRỪ theo danh sách đóng (nhãn, số đếm) và BỎ IM LẶNG (lượng khác chiều đo, số ở mệnh đề trước);
  - kỳ rơi về mặc định khi câu có mốc chưa đọc được ("Q3", "tuần44", "15th Sep", "sáng nay");
  - phủ định, chặn, xấp xỉ không được mô hình hoá;
  - S9 không bắt được vì claim của người hỏi được đóng dấu chủ thể / chỉ số / kỳ của chính hệ.
- **Danh sách loại trừ hỏng khi THIẾU:** mỗi biến thể chưa liệt kê thành một phán quyết sai. Đó đúng là "danh sách từ khoá đóng cứ phải mở rộng" mà hợp đồng §8 cấm.

## Quyết định

### 1. Bất biến

**PHÁN QUYẾT ⇒ CLAIM ĐÃ GIẢI ĐỦ CHIỀU.** Hệ chỉ được mở đầu bằng "Đúng." / "Không." khi claim {chủ thể, chỉ số, kỳ, giá trị, đơn vị, chiều} của người hỏi được giải bằng BẰNG CHỨNG DƯƠNG trong câu, và MỌI phần của câu hỏi đều được giải thích. Thiếu một mắt xích ⇒ KHÔNG phán quyết.

### 2. Cổng đóng khi thiếu (fail-closed), không phải danh sách chặn

Điều kiện chung: câu VIẾT CÓ DẤU (câu không dấu được khớp bằng bản bỏ dấu nên nhiều chữ trùng nhau: "co" = có / cỡ, "dau" = dầu / đầu) và có ĐÚNG MỘT cụm xác nhận (hai cụm là hai claim; "có đúng không" là một cụm).

Một câu xác nhận chỉ qua cổng khi phần chữ còn lại sau khi trừ đi các phần ĐÃ HIỂU là rỗng. Phần đã hiểu gồm:

| Phần | Lấy từ |
|---|---|
| tên thực thể | S3 (mention), chỉ khi cụm tên chỉ về CHỦ THỂ hệ đã trả lời hoặc một cấp cha của nó ("Máy 03 của dự án A"); phạm vi hỏi có nêu tên thì phải nêu chủ thể. "EX-03 của dự án B", "Dự án A … cho EX-03", "EX-03 … 3850 L, vậy dự án A … đúng không?" ⇒ chủ thể chưa giải |
| mốc thời gian đọc được | S2 (span kỳ / ngày) + từ chỉ NGÀY đứng ngay trước ("ngày", "hôm", "mùng", "mồng" của "ngày 13/10"; "tuần 15/9" là tuần chứa 15/9 nên "tuần" là phần dư) + mốc viết bằng chữ khớp kỳ đã đọc ("hôm nay"); kỳ MẶC ĐỊNH của pack KHÔNG tính, và câu phải có đúng MỘT kỳ đọc được |
| nghĩa chỉ số | từ khoá của CHÍNH ý định đã khớp (ý định "yếu" — chỉ số theo quy ước của pack — không được phán quyết) + thuật ngữ vật tư của chính ý định ("dầu") + tên nguồn của chính chỉ số (nhãn + bí danh khai ở pack, vd "phiếu kho") CHỈ KHI con số trả lời lấy đúng và chỉ từ nguồn đó (`Result.UsedSources`): "phiếu xuất kho ghi 3900 L đúng không?" mà 3.900 L là của ERP ⇒ không phán quyết |
| con số xác nhận | mọi con số trong phạm vi hỏi phải CHÍNH LÀ giá trị được xác nhận (số khác ⇒ không phán quyết), có đơn vị cùng chiều đo với chỉ số (viết rõ, hoặc suy từ thuật ngữ vật tư: "4k dầu"), hoặc số có chữ ("4k", "bốn ngàn"); số trần có dấu chấm / phẩy ("3.850", "1,2") KHÔNG tính — mơ hồ phần nghìn / thập phân và không nói đơn vị; và nằm trong miền min / max của MỘT lần đo của chỉ số (ngoài miền ⇒ có thể đã đọc sai định dạng / đơn vị: "3.850 m3" thành 3.850.000 L). Dấu xấp xỉ / chặn / mẫu số ("khoảng", "~", ">", "/ngày") nằm ngoài span số nên thành phần dư — không cần nhánh riêng |
| cụm xác nhận | `confirm_markers` của pack |
| từ trung tính | khoá pack MỚI `claim_neutral_words`: lịch sự, trợ từ, trợ động từ ("cho hỏi", "ạ", "đã", "vậy"…) |
| dấu câu ngăn cách | `, . ; : ? ! ( ) [ ] " ' “ ” ‘ ’ … -`; gạch ngắn / dài ("–", "—") KHÔNG tính (dính vào số là dấu âm bộ chuẩn hoá không đọc); mọi ký tự khác (cả "~", ">", "%", "/") phải được giải thích |

- Mọi thứ khác — số thứ hai, nhãn ("bể 7", "phiếu 912", "ca 2"), mốc chưa đọc ("Q3", "sáng"), phủ định ("không", "chưa"), so sánh / chặn ("trên", "hơn"), nguồn hay nghĩa khác ("theo ERP", "đội báo"), từ lạ — đều là phần dư ⇒ không phán quyết.
- **Danh sách DUY NHẤT mới là danh sách CHO PHÉP (từ trung tính), không phải danh sách CHẶN.** Thiếu một từ chỉ làm câu mất phán quyết (an toàn), không bao giờ làm phán quyết sai. Mở rộng nó chỉ để tăng độ phủ, có test.
- Số trần trơn ("được cấp 3850") không tự giải được chiều đơn vị ⇒ không phán quyết. Luật "số trần ngay sau từ khoá" của bản phân tích bị bác: từ khoá danh từ ("phiếu kho 2") làm nhãn thành con số.
- Phạm vi hỏi là mệnh đề chứa cụm xác nhận và các mệnh đề sau nó; mệnh đề hỏi không chứa số ("…, vậy đúng không?") thì phạm vi hỏi là cả câu.
- Mệnh đề đứng TRƯỚC phạm vi hỏi (ngữ cảnh) được chứa thêm: lượng tự mang đơn vị ("Định mức 5.000 L, vậy …"), từ khoá của các ý định SỐ LIỆU (route SQL_QUERY, không "yếu": "Đội báo 2k, vậy …" — KHÔNG gồm ý định quy trình / chào hỏi / phép tính dẫn xuất: "Làm gì có, …", "Khi lệch, …"), cách gọi nguồn của mọi chỉ số, từ chỉ lượng của pack — tức sự kiện người hỏi nêu về chỉ số khác. Số trần ở ngữ cảnh (có thể là nhãn / mốc) và mọi phần khác vẫn phải được giải thích ("Không tính ERP, …" vẫn mất phán quyết).

### 3. Khi không phán quyết thì trả gì (hợp đồng §8: "factual answer without binary verdict; clarification; insufficient evidence")

- **Giữ nguyên** hai nhánh hỏi lại đã có: nhiều con số khác nhau ("rival"), mốc viết bằng số không đọc được ("doubt").
- Còn lại: **trả số liệu có thẩm quyền** như câu hỏi thường (mẫu `metric_total`, RESOLVED, kèm xung đột / trích dẫn như cũ), thêm mẩu cố định `confirm_withheld`. Mẩu không chép chữ của người hỏi, vì chữ đó có thể mang số làm lớp kiểm số chặn oan.
- Câu không nêu mốc ("EX-03 được cấp 3850 L đúng không?"): kỳ mặc định của pack chỉ dùng để trả số, không để phán quyết.

### 4. Không làm trong ADR này

- Đảo chân trị theo phủ định, phán quyết cho quan hệ "trên / dưới / khoảng": cần ca thật đòi (P7). Tạm thời không phán quyết.
- Đọc mốc dưới ngày, "Q3", "tuần 44" thành kỳ: thuộc cụm C2 (kỳ mặc định lặng lẽ cho MỌI câu số liệu), một thay đổi riêng.
- Sửa tầng ý định (cụm C3–C7 của phân tích T1).

## Hệ quả

- Câu xác nhận đúng mà có từ ngoài danh sách cho phép sẽ mất phán quyết (vẫn được trả số đúng). Đo trên cây chính + bản sửa, TRƯỚC khi chạy H3:
  - bộ 117: số ca có phán quyết 23 → 18; mất ở 008, 009 (xấp xỉ), 041 (số trần), 049, 103 (từ ngữ cảnh ngoài danh sách); 0 ca thêm phán quyết, 0 ca đổi chiều;
  - H1 (hồi quy, hai đường): 6 → 5 (mất ở 316); 0 ca thêm, 0 ca đổi chiều;
  - 0 ca của hai bộ đổi nhãn ĐẠT / ĐỎ (kỳ vọng của hai bộ không kiểm chữ "Đúng." / "Không.").
- Test cũ: 55 dòng kỳ vọng phán quyết trong 5 tệp test đổi thành "không phán quyết", 1 dòng mới có phán quyết. Lý do: chúng mã hoá luật danh sách đóng (nhãn / số đếm, "ca 2", "phiếu 912"…) mà hợp đồng câu xác nhận thay thế. Không dòng nào đổi chiều phán quyết.
- H3 (đã đóng băng trước bản sửa) đo cả hai mặt: phán quyết sai (an toàn) và câu sai mà không được trả "Không." (độ phủ).

## Chứng minh

- 15 câu thăm dò: 0 phán quyết sai (`TestCauThamDoXacNhanKhongPhanQuyetSai`).
- Test thuộc tính (metamorphic): chen một cụm không được giải thích vào câu đang được phán quyết đúng ⇒ mất phán quyết, không bao giờ đổi chiều (`TestCongPhanQuyetCumLaLamMatPhanQuyet`, 75 phép chen). Chen từ trung tính ⇒ giữ phán quyết (`TestCongPhanQuyetTuTrungTinhGiuPhanQuyet`).
- Bộ 117 và H1 (hồi quy, hai đường): 0 ca đổi nhãn.
- Đột biến: danh mục `n43-cong-phan-quyet.sh` (20 đột biến, mỗi nhánh của cổng) bị bắt 20/20; toàn bộ 35 danh mục trên cây có bản sửa: 618 bị bắt, 0 sống, 0 không biên dịch, 0 chuỗi lệch (chuỗi gốc BA của `w5-g4-acl.sh` đổi theo dòng dựng câu mới).
- Soát đối kháng độc lập (vai D, `aa0e8d4`): khoảng 352 nghìn câu (soạn tay, chèn từ, ngẫu nhiên, văn phạm có đáp án độc lập), 0 phán quyết đảo chiều, không có đường vòng qua cổng; nhưng 21 câu (8 nhóm) vẫn phán quyết khi claim chưa giải — cổng hứa đóng mà chưa đóng (cả 21 câu cũng phán quyết trên bản TRƯỚC cổng). Xem mục "Bổ sung sau soát đối kháng".

## Bổ sung sau soát đối kháng (T2.1, 21/10, trước khi chạy H3)

| Nhóm của người soát | Chiều chưa giải | Đóng bằng |
|---|---|---|
| A, B: hai thực thể, tiền đề phân cấp sai | chủ thể | cụm tên phải chỉ về chủ thể hoặc cấp cha; phạm vi hỏi có tên thì phải có chủ thể; đúng một cụm xác nhận |
| C: "1,2", "1,200 khối", "3.850 m3" | đơn vị / giá trị | số trần có dấu chấm / phẩy không tính là có đơn vị; con số phải trong miền min / max của chỉ số |
| D: câu không dấu ("co", "do", "the", "dau") | nghĩa | câu không dấu không phán quyết |
| E: "tuần 15/9" | kỳ | chỉ từ chỉ NGÀY là một phần của mốc ngày |
| F: "phiếu xuất kho ghi …" khi số là của ERP | nguồn | tên nguồn chỉ được giải thích khi số lấy đúng và chỉ từ nguồn đó; pack khai bí danh nguồn |
| G: "–3850 L" | chiều (dấu âm) | gạch ngắn / dài không còn là dấu ngăn cách |
| H: "Làm gì có, …", "Khi lệch, …" | ngữ cảnh | ngữ cảnh chỉ nhận từ khoá của ý định SỐ LIỆU |

- Đo lại: 21 / 21 câu của người soát nay không phán quyết (test `TestCongPhanQuyetSoatDoiKhang`, kèm 8 câu đối chứng VẪN phán quyết); trên 945 câu soạn tay của người soát: phán quyết đảo chiều 2 → 0, câu có phán quyết 418 → 240.
- Bộ 117: số ca có phán quyết 18 → 15 — mất thêm ở 001, 011 (không dấu) và 010 ("1tr dầu" = 1.000.000 L, ngoài miền một lần đo; trước đó ra "Không." ĐÚNG — cái giá của luật miền giá trị); H1 không đổi (5); 0 ca đổi nhãn ĐẠT / ĐỎ, 0 ca đổi chiều.
- Cái giá (mất độ phủ, không sai): câu không dấu; tổng lớn hơn trần một lần đo (tổng dự án / nhiều ngày lớn); số trần có dấu chấm / phẩy; câu nêu nguồn khi số trộn nhiều nguồn.
- Test cũ bị nới thành vô hiệu (`TestCauXacNhanMotConSoVanTraLoi` so tiền tố rỗng) được siết lại; hai câu ngữ cảnh trả về kỳ vọng phán quyết cũ (vẫn đúng). Đếm của người soát: 60 câu cũ đổi kỳ vọng (55 dòng ghi ở trên đếm theo dòng mã).
- Đột biến `n43`: 35 (20 cũ, chuỗi gốc đổi theo mã mới + 15 cho các nhánh mới) — 35 / 35 bị bắt.
- Chưa làm (S3 của người soát, ghi lại): thêm từ trung tính để tăng độ phủ ("dạ", "giúp"…); gộp phần chia phạm vi hỏi đang lặp giữa `verdictGap` và `claimNumbers`.

### Vòng 2 (T2.2, 21/10, trước khi chạy H3)

Người soát D soát lại T2.1 (~0,77 triệu câu): 21 / 21 câu vòng 1 đóng; 0 phán quyết đảo chiều; không hồi quy. Còn các lỗ có sẵn từ T2:

| Mã | Mức | Lỗ | Xử lý |
|---|---|---|---|
| R2-1 | CAO | một cụm xác nhận, HAI phát biểu cùng con số ("EX-03 … 3850 L, dự án A 3850 L, đúng không?" — dự án A thật là 5.050 L) ⇒ "Đúng." | ĐÓNG MỘT PHẦN (xem Vòng 3): con số xác nhận nhắc lại chỉ là cùng claim khi đứng liền kề (chỉ cách khoảng trắng / ngoặc); dạng gương — tên đứng SAU lần nhắc lại — còn mở |
| R2-2 | TB-CAO | mệnh đề hỏi chỉ nêu cấp cha ("…, dự án A đúng không?"), luật "phạm vi hỏi có tên thì phải có chủ thể" dùng phạm vi đã lùi về đầu câu | ĐÓNG MỘT PHẦN (xem Vòng 3): luật chủ thể dùng đầu mệnh đề chứa cụm xác nhận; dòng vá này gây hồi quy S1, đã sửa ở T2.3; biến thể dấu câu còn mở |
| R2-3 | TB | lượng có dấu chấm / phẩy mà đơn vị chỉ suy từ thuật ngữ ("dầu 1,2") | ĐÃ ĐÓNG: không tính là có đơn vị (cái giá: "dầu 3.850" mất phán quyết); T2.3 trả lại phán quyết khi nguyên văn có chữ ("3,85k") |
| R2-4 | THẤP-TB | tên nguồn ở mệnh đề NGỮ CẢNH không chứa lượng ("Phiếu kho, RL-02 … đúng không?") không đối chiếu nguồn đã dùng | ĐÃ SỬA phần chính ở Vòng 5 (R5-1: đề mục thuần cùng câu, cụm ở ngữ cảnh đứng ngay trước con số); phần còn lại ghi ở cái giá của Vòng 5 |
| R2-5 | TB | máy và dự án cùng nêu, nối bằng dấu rời ("Lu 02 : Dự án A …") — hệ chọn máy làm chủ thể | ghi lại: quyết định thiết kế (chủ thể = thực thể cụ thể nhất) |
| R2-6…R2-9 | THẤP | bí danh mơ hồ có ứng viên là cấp cha; từ chỉ ngày trước span tháng / tuần; tên sai dấu vẫn khớp; "3,850 lít" đọc là nghìn | ghi lại |

- Ba dòng vá R2-1 / R2-2 / R2-3 là bản vá thử CỦA NGƯỜI SOÁT; phiên chính áp nguyên văn và đo lại bằng test + bộ câu có đáp án độc lập của người soát — chưa có người soát thứ ba cho riêng phần này.
- Test mới `TestCongPhanQuyetSoatDoiKhangVong2` (7 câu không phán quyết + 5 đối chứng), ghim thêm ba nhánh chưa có test đỏ: cấp cha nhiều tầng (danh mục giả lập 4 tầng `seed/entities-bon-tang.yaml`), từ chỉ lượng ở ngữ cảnh, hai kỳ đọc được.
- Đo: test có PG 451/0/0; bộ 117 117/117 và H1 89/104 hai đường, 0 ca đổi nhãn, số ca có phán quyết không đổi so với T2.1 (15; 5); hai phát biểu một cụm (20.000 câu có đáp án độc lập): 11.375 → 0 phán quyết; văn phạm có dấu (60.000 câu): 33.209 phán quyết đúng, 0 sai; n43 41 / 41.
- Cái giá ghi nhận (S3): câu không dấu mất 100% phán quyết; trần một lần đo áp cho tổng nhiều ngày / nhiều máy làm mất phán quyết đúng ("EX-03 tuần trước … 21000 L").

### Vòng 3 (T2.3, 21/10, sau H3: soát lần ba cho ba dòng vá T2.2)

Hai người soát độc lập, mỗi người một worktree, chạy câu thật trên pack và dữ liệu hạt giống:
- góc A tìm phản ví dụ;
- góc B đo độ phủ test và đột biến.

Cả hai cùng KHÔNG ĐẠT.

| Mã | Mức | Phát hiện | Xử lý |
|---|---|---|---|
| R3-1 | **S1, HỒI QUY** | Dòng vá R2-2 thu hẹp kiểm chủ thể từ CÂU về mệnh đề chứa cụm xác nhận. Câu hai vế bị phán quyết theo EX-03, sai chiều (dự án A thật 5.050 L; T2.1 không phán quyết):<br>- "EX-03 hôm nay được cấp 3850 L. Dự án A được cấp 5050 L, đúng không?" → "Không.";<br>- "… Dự án A được cấp 3850 L, đúng không?" → "Đúng.".<br>15 câu hai vế đổi từ không phán quyết sang có phán quyết; ≥ 12 câu sai chiều | **ĐÃ SỬA (T2.3):** kiểm chủ thể ở CẢ phạm vi câu [lo, len) lẫn mệnh đề hỏi [qs, len) |
| R3-2 | S3, hồi quy | Dòng vá R2-3 giữ oan "dầu 3,85k", "1,2 nghìn dầu": chữ đã nói hàng của con số | **ĐÃ SỬA (T2.3):** chỉ coi là mơ hồ khi nguyên văn không có chữ |
| R3-3 | **S1, có từ T2.1** | Dạng gương: "EX-03 hôm nay được cấp 3850 L (3850 L của dự án A) đúng không?" → "Đúng." (dự án A thật 5.050 L). Phép kiểm liền kề chỉ nhìn đoạn GIỮA hai lần nhắc, không nhìn tên đứng sau | **ĐÃ SỬA (E3, `a853e96`):** "sau con số xác nhận đầu tiên chỉ được nhắc lại CHÍNH chủ thể". Test đỏ trước (8 câu ra "Đúng."); bộ 117 / H1 / H3 không đổi nhãn hay câu trả lời. Cái giá: định ngữ cấp cha đứng sau số mất phán quyết. Soát độc lập: ĐẠT CÓ GHI CHÚ (Vòng 4) |
| R3-4 | S2, có từ T2.1 | R2-2 chỉ đóng khi dấu ngăn là `, ; . ? !` + khoảng trắng. Với ":", "-", "(", "[", xuống dòng, dấu phẩy thừa ("…, dự án A, đúng không?"), hoặc cấp cha đặt sau cụm xác nhận, hệ vẫn trả "Đúng." | **ĐÓNG MỘT PHẦN:** E3 (`a853e96`) đóng phía SAU con số. Phía TRƯỚC con số (mệnh đề khác, dấu ngăn lạ) **còn mở**: Vòng 4 R4-2. Phép kiểm ở mệnh đề hỏi vẫn cần cho câu mà mệnh đề hỏi đứng trước con số ("EX-03, dự án A đúng không, hôm nay được cấp 3850 L?"); có test ghim |
| R3-5 | S4 | Ba dòng vá chỉ được ghim bởi một test tích hợp; gói `core/pipeline` không tự bắt CG29–CG31; năm đột biến tinh còn sống | **ĐÃ SỬA:** 6 ca trong `TestVerdictGap`; 4 ca ghim trong `TestCongPhanQuyetSoatLan3`; n43 thêm CG35–CG40 |

**Đo:**
- Test đỏ trước, trên HEAD `2f1b8c2`: `TestCongPhanQuyetSoatLan3` gồm 7 câu không phán quyết (có bản 4 tầng) và 7 đối chứng / ghim.
- Test có PG 453/0/0. n43: 47 / 47 bị bắt.
- Bộ 117 117/117, H1 89/104 hai đường, H3 109/134: 0 ca đổi nhãn, 0 câu trả lời đổi chữ so với T2.2.

**Bài học:** bộ 20.000 câu "hai phát biểu một cụm" của vòng 2 chỉ phủ dạng có dấu phẩy, nên không thấy dạng hai câu. Một bản vá do người soát đề xuất vẫn phải qua một người soát khác: ở đây chính bản vá đó sinh phán quyết sai chiều.

### Vòng 4 (21/10 chiều: soát độc lập T2.3 và E3)

Hai người soát, mỗi người một worktree, chạy câu thật:
- soát T2.3 (`01d4348`): KHÔNG ĐẠT;
- soát E3 (`a853e96`): ĐẠT CÓ GHI CHÚ.

| Mã | Mức | Phát hiện | Xử lý |
|---|---|---|---|
| R4-1 | S2, HỒI QUY của T2.3 | Ngoại lệ "có chữ" (R3-2) mở phán quyết cho số nhân hỏng có chữ số sau hậu tố: "dầu 1,2k5" bị đọc 1.700 rồi ra "Không."; "dầu 3,5k35" ra "Đúng." (24/40 câu thử) | **ĐÃ SỬA (`936d090`):** chỉ hết mơ hồ khi nguyên văn KẾT THÚC bằng chữ ("3,85k", "3,85K", "1,2 nghìn") |
| R4-2 | **S1, có từ T2.1** | Tên cấp cha đứng TRƯỚC con số, ở mệnh đề khác hoặc sau dấu ngăn lạ, vẫn bị phán quyết theo máy:<br>- "EX-03 hôm nay được cấp 3850 L. Dự án A, 5050 L đúng không?" → "Không." (claim về dự án A là ĐÚNG);<br>- "EX-03: dự án A hôm nay được cấp 3850 L đúng không?" → "Đúng."; tương tự với " - ", "(", "[", xuống dòng, phẩy không khoảng trắng | **ĐÃ SỬA (Vòng 5, `03c2c54`, `489c98a`)** — luật "thứ gần nhất trước con số", xem Vòng 5. Lúc ghi Vòng 4: Luật thử của người soát: cấp cha chỉ được chấp nhận khi là định ngữ "của" đứng liền sau một lần nhắc chủ thể. Đã chạy thử: 5 gói test xanh, bộ 117 / H1 / H3 0 dòng khác |
| R4-3 | **S1, có từ T2.1** | Danh từ chung "dự án", "công trình" ở mệnh đề ngữ cảnh được che như thuật ngữ, nên con số ở mệnh đề hỏi bị gán cho chủ thể đầu: "EX-03 … 3850 L. Dự án, 5050 L đúng không?" → "Không." | **ĐÃ SỬA (Vòng 5, `03c2c54`, `489c98a`).** Lúc ghi Vòng 4: Ở ngữ cảnh chỉ che thuật ngữ vật tư của ý định số liệu; hoặc coi "dự án" đứng một mình là tên chưa giải |
| R4-4 | S2, có từ T2.1 | Bí danh mơ hồ "máy 03" có chủ thể trong ứng viên được coi là CHÍNH chủ thể kể cả khi tách câu: "EX-03 … 3850 L. Máy 03 được cấp 3000 L, đúng không?" → "Không." (3.000 L là số của DZ-03) | **ĐÃ SỬA (Vòng 5, `03c2c54`, `489c98a`)** — luật bí danh có bằng chứng dương, xem Vòng 5. Lúc ghi Vòng 4: Luật thử: bí danh mơ hồ chỉ là chủ thể khi không có ngắt mệnh đề / câu giữa nó và lần nhắc rõ gần nhất của chủ thể. Đã chạy thử: 5 gói xanh, 117 / H1 / H3 0 dòng khác |
| R4-5 | INFO | Cái giá của E3 rộng hơn ví dụ đã ghi: mất cả "Hôm nay cấp 3850 L cho EX-03 của dự án A đúng không?" (số đứng trước chủ thể + định ngữ cấp cha); trên 20.000 câu fuzz: 1.084 câu mất phán quyết, 0 đổi chiều, 0 câu phán quyết thêm | Ghi nhận. Luật của R4-2 có thể trả lại phán quyết cho dạng "của" liền sau chủ thể |
| R4-6 | S3 / S4 | Thiếu đối chứng dương và ca đơn vị cho E3; đột biến IsLetter → IsLower sống; khối không phán quyết của SoatLan3 không kiểm Outcome | **ĐÃ SỬA (`936d090`):** đối chứng "(máy xúc EX-03)", "máy 03" sau số ⇒ "Đúng."; 2 ca TestVerdictGap want "subject"; kiểm Outcome; n43 CG43–CG46 ⇒ 53 / 53 |

**Đo của người soát:**
- T2.3 trên 241.920 câu hai vế có đáp án độc lập: T2.2 có 4.608 phán quyết sai chiều + 10.240 phán quyết cho claim chưa giải; T2.3 còn 0 + 0, đóng thêm 22.528 câu, mở thêm 0;
- 15.840 câu rõ nghĩa có cấp cha cạnh chủ thể không bị đóng oan câu nào.

**Thứ tự kế tiếp:** đóng R4-2, R4-3, R4-4 bằng test đỏ trước, đo lại bộ 117 / H1 / H3, rồi soát độc lập, **trước khi chạy H4** (H4 có ca xác nhận).

### Vòng 5 (22/10 sáng: đóng R4-2…R4-4, R5-1 và câu đổi chiều của số bằng chữ) — mã `03c2c54`, `489c98a`

Đầu vào:
- bản thử V5 của lượt thiết kế 21/10 (dừng lúc chốt ca), đo lại và soát mã độc lập tối 21/10;
- hai lỗ S1 mới tìm ra tối 21/10: R5-1 (bản soát mã) và R5-2 (phản hồi audit ROADMAP K5, `docs/bao-cao/2026-10-21-phan-hoi-roadmap-k1-k5.md`).

| Mã | Mức | Phát hiện | Xử lý |
|---|---|---|---|
| R5-1 | **S1, có từ T2.1** (gồm R2-4) | Tên nguồn, từ khoá chỉ số khác hay từ chỉ lượng ở ngữ cảnh là nguồn / chỉ số của chính claim mà không được đối chiếu: "EX-03 hôm nay được cấp 3850 L. ERP, 3900 L đúng không?" → "Không." dù ERP ghi 3.900 L; "ERP, EX-03 hôm nay được cấp 3900 L đúng không?" → "Không."; "… Định mức, 5000 L …" → "Không."; "Đội báo, EX-03 hôm nay được cấp 3850 L đúng không?" → "Đúng." (đội báo 4.000 L). Dấu phẩy làm cụm đó rơi vào ngữ cảnh, nơi được chứa tên nguồn bất kỳ | **ĐÃ SỬA (Vòng 5):** luật 2 và luật 3 dưới đây |
| R5-2 | **S1** | Số bằng chữ kết thúc bằng "<hàng> <một chữ số>" (có thể kèm "rưỡi") bị đọc như hàng đơn vị: "một nghìn hai" → 1.002 (nói thường 1.200), "ba nghìn tám rưỡi" → 3.008,5 (nói thường 3.850). "Hôm qua đội báo lu 02 một nghìn hai lít dầu đúng không?" → "Không." dù đội báo đúng 1.200 L | **ĐÃ SỬA ở cổng (Vòng 5):** dạng này không phán quyết. Bộ đọc giữ nguyên: sửa nó đổi cả tầng nạp, chờ D-OP-14 |
| R5-3 | S2, lỗ của V5 | Bí danh sau con số cùng ngày viết bằng số: "EX-03 ngày 19/10 được cấp 3850 L máy 03 đúng không?" mất phán quyết (cùng câu với "hôm nay" thì giữ), trái đối chứng đã ghim ở R4-6 | **ĐÃ SỬA (Vòng 5):** "/", "-", ":" giữa hai chữ số không phải dấu ngắt |

**Luật (Vòng 5):**
1. Thứ gần nhất đứng TRƯỚC con số xác nhận đầu tiên phải gắn vào chủ thể: chính chủ thể nhắc rõ; bí danh mơ hồ có bằng chứng dương (luật 4); hoặc định ngữ nối vào thứ liền trước, mà thứ đó đã gắn, bằng đúng chữ "của" hay ngoặc trần ("EX-03 của dự án A", "EX-03 (dự án A)"). Không có thứ nào đứng trước thì phần sau con số do E3 kiểm.
2. Thứ có thể sở hữu con số: tên thực thể; thuật ngữ khác thuật ngữ của chính ý định ("dự án", "nhân công"); và ở ngữ cảnh, khi không có con số nào xen giữa nó và phạm vi hỏi, các "cụm nguồn / chỉ số khác": tên nguồn khác nguồn đã tạo con số trả lời, từ khoá của ý định số liệu khác, từ chỉ lượng. Không tính: từ khoá của ý định yếu ("bao nhiêu"); tên đúng nguồn đã dùng; từ khoá của chính ý định, trừ khi nó cũng là tên / bí danh của một nguồn khác nguồn đã dùng (R5-5: "phiếu kho" khi số trả lời lấy từ ERP).
3. Đề mục thuần: đoạn từ đầu câu chứa phạm vi hỏi tới phạm vi hỏi chỉ gồm cụm nguồn / chỉ số khác (cùng từ trung tính, mốc thời gian kể cả "ngày" / "hôm" đứng trước mốc số, dấu ngăn, và chữ của chính claim — từ khoá của chính ý định, tên đúng nguồn đã dùng, thuật ngữ vật tư của chính ý định, R5-4) thì đó là nguồn / chỉ số của chính claim ⇒ không phán quyết ("ERP, EX-03 …", "Đội báo, EX-03 …"). Đoạn có chữ khác ("Bao nhiêu định mức, vậy EX-03 …", "Theo ERP, …") không phải đề mục. Bản luật rộng hơn ("mọi cụm ở ngữ cảnh không kèm số") đã thử và bỏ: nó chặn oan "Bao nhiêu định mức, vậy EX-03 hôm nay được cấp 3850 L đúng không?" (đối chứng của Vòng 2).
4. Bí danh mơ hồ chứa chủ thể chỉ là chủ thể khi câu cho bằng chứng dương: không có lần nhắc rõ nào; đứng một mình trong ngoặc, không ngắt mệnh đề khỏi lần nhắc rõ; đứng sau con số mà đoạn từ lần nhắc rõ tới nó chỉ có chữ, số, khoảng trắng và dấu nằm giữa hai chữ số; hoặc dạng "bí danh (lần nhắc rõ)".
5. Con số xác nhận viết bằng chữ, kết thúc bằng "<hàng> <một chữ số>" (có thể kèm "rưỡi"), không phán quyết.

**Năm dạng xử lý theo mặc định fail-closed (§2), chưa ghim bằng test, chủ dự án đảo được:**

| Dạng | Ví dụ | Hành vi |
|---|---|---|
| viết liền chủ thể + cấp cha | "EX-03 dự án A hôm nay được cấp 3850 L đúng không?" | không phán quyết |
| đồng vị phẩy | "EX-03, dự án A, hôm nay được cấp 3850 L, đúng không?" | không phán quyết (R3-4 đã coi là lỗ) |
| bí danh chen giữa, không ngắt | "EX-03 hôm nay Máy 03 được cấp 3850 L đúng không?" | không phán quyết (chặt hơn luật thử của R4-4) |
| cấp cha trong ngoặc trần | "EX-03 (dự án A) hôm nay được cấp 3850 L đúng không?" | phán quyết (lỏng hơn luật thử của R4-2) |
| bí danh rồi lần nhắc rõ trong ngoặc | "Máy 03 (EX-03) hôm nay được cấp 3850 L đúng không?" | phán quyết |

**Đo:**
- Test đỏ trước, trên HEAD `ffb1efb`: `TestCongPhanQuyetVong5` đỏ đúng 20 ca âm (15 R4, 3 R5-1, 2 R5-2), 0 ca dương; `TestVerdictGap` đỏ 5 dòng mới.
- Lần sửa đầu làm đỏ một test cũ (`TestCauXacNhanMotConSoVanTraLoi`: "EX-03 hôm nay đội báo 3k, vậy được cấp 4000 L đúng không?" mất "Không.") ⇒ thêm điều kiện "không con số nào xen giữa" vào luật 2; câu đó nay được ghim.
- Test có PG ở mã cuối 459 / 0 / 0: 455 test cũ giữ nguyên kết quả theo tên, 4 test mới.
- Bộ 117 / H1 / H3, đường bộ nhớ và PostgreSQL: 0 dòng khác so với trước Vòng 5; số ca có phán quyết không đổi (15 / 5 / 16).
- Bộ câu tổng hợp 122.429 câu (dựng lại từ lượt thiết kế, khớp md5): đóng 25.669 / 25.669 phán quyết ở câu chưa giải (R4-2 12.941, trong đó 7.168 sai chiều; R4-3 3.656; R4-4 9.072); câu rõ nghĩa mất 0 (132 theo nhãn gốc: đúng dạng bí danh chen giữa ở bảng trên); 0 đổi chiều, 0 phán quyết mới. Fuzz 77.062 câu: đóng 10.774 / 10.774, mất 0.
- Tổng mọi bộ có sẵn (736.784 câu, gồm 20.000 câu của người soát E3 và 30.000 câu nhiễu): 0 đổi chiều, 0 phán quyết mới so với trước Vòng 5. So với bản thử V5: khác 79 câu, toàn dạng ngày + bí danh sau số, đều lấy lại đúng phán quyết cũ.
- R5-1 và R5-2 không có câu nào trong các bộ có sẵn: kiểm bằng test, 40 câu viết tay và lưới 453 câu sinh từ seed (đóng 8 / 8 câu R5-1 và 60 / 60 câu R5-2 có phán quyết; 137 câu đối chứng giữ nguyên). Mức mất phán quyết của hai luật này trên câu tự nhiên ở quy mô lớn: CHƯA ĐO.
- Điểm S3 còn lại: aliasTied chọn lần nhắc rõ gần bí danh nhất theo khoảng cách byte, nên câu nhắc chủ thể hai lần ("… EX-03 ngày 19/10 3,85k máy 03 được cấp EX-03 …") có thể mất phán quyết tuỳ độ dài chữ ở giữa (1 câu trong 20.000; không đổi chiều).
- Soát độc lập ba vòng, mỗi vòng một người soát khác, chỉ đọc, thử câu trong worktree riêng:
  - vòng 1 (bản đầu): KHÔNG ĐẠT vì tồn dư, không hồi quy — "<hàng> <chữ số> rưỡi" (S1), đề mục nguồn ở đầu câu R2-4 (S2), thiếu test (S2), ADR chưa có (S3);
  - vòng 2: KHÔNG ĐẠT vì tồn dư — đề mục có "ngày dd/mm" không được coi là đề mục (S2), 4 đột biến sống ở luật đề mục (S2);
  - vòng 3 (kiểm dòng vá "ngày" do người soát vòng 2 đề xuất): dòng vá ĐÚNG, chỉ đóng thêm (204 câu × 3 cây: 0 phán quyết mới, 0 đổi chiều); KHÔNG ĐẠT vì còn hai lỗ cùng họ, có từ trước (R5-4, R5-5 dưới đây).
- Đột biến (`scripts/dot-bien/n43-cong-phan-quyet.sh`, thêm CG153–CG204 và NW12–NW20): ở mã cuối, 210 / 210 dòng chạy bị bắt, 0 sống, 0 không biên dịch. Các đột biến tương đương hay chờ quyết ghi thành dòng chú thích kèm lý do và số đo. Ba đột biến cũ (CG24, CG30, CG35) sống từ khi có luật "thứ gần nhất trước con số", vì luật này bao trùm phép kiểm chủ thể cũ: CG24 nay bị bắt ở mức đơn vị (`TestSubjectGapPhamViHoi`), CG30 / CG35 ghi chờ quyết (giữ làm phòng thủ chiều sâu hay xoá phép kiểm thừa).

**R5-4, R5-5 (cùng họ R5-1, có từ trước Vòng 5, tìm ra ở vòng soát 3): ĐÃ SỬA** — mã `962ae9c` (luật 2, luật 3 ở trên đã ghi theo bản sửa):
- R5-4: đề mục có thêm chữ mà phép kiểm chính vẫn giải thích — thuật ngữ vật tư ("dầu"), từ khoá của chính ý định ("kho xuất"), tên đúng nguồn đã dùng — thì không còn "thuần" nên vẫn phán quyết: "Dầu ERP, EX-03 hôm nay được cấp 3900 L đúng không?" → "Không." (ERP ghi 3.900 L); "Dầu đội báo, EX-03 hôm nay được cấp 3850 L đúng không?" → "Đúng." (đội báo 4.000 L); "Phiếu xuất kho, ERP, EX-03 … 3850 L" → "Đúng.".
- R5-5: bí danh "phiếu kho" trùng từ khoá của chính ý định nên không bị coi là nguồn khác khi nguồn đã dùng là ERP: "Phiếu kho, EX-03 hôm nay được cấp 3900 L đúng không?" → "Đúng." dù phiếu kho không có bản ghi.
- Đo bản sửa: test đỏ trước trên `c086fe7` (`TestVerdictGap` 4 dòng, `TestCongPhanQuyetVong5` 6 ca âm; ca dương xanh ở cả hai); test có PG 459 / 0 / 0; 117 / H1 / H3 hai đường 0 dòng khác; 713.097 câu corpus + 13 tệp E3: 0 phán quyết mới, 0 đổi chiều; danh mục đột biến 215 / 215 + CG211 bị bắt.
- Soát độc lập vòng 4 (người soát khác người đề xuất bản vá): **ĐẠT CÓ GHI CHÚ**, 0 S1 / S2. Trên 138.870 lượt lưới đề mục tổng hợp: 0 phán quyết mới, 0 đổi chiều; phán quyết sai chiều theo chỉ số nêu ở đề mục giảm 2.273 → 448 (bộ seed chuẩn). Ghi chú:
  - V5S4-01 (lỗ test): ghim thêm "EX-03 hôm nay phiếu kho, được cấp 3900 L đúng không?" (không phán quyết) và đột biến CG211;
  - V5S4-02…04 (cùng họ, có từ trước): đề mục kèm "bao nhiêu" ("Bao nhiêu ERP, EX-03 … 3900 L đúng không?"), kèm danh từ chung ("Dự án ERP, EX-03 …"), hay đứng thành câu / dòng riêng ("Dầu ERP. EX-03 …", "ERP: bao nhiêu? EX-03 …") vẫn ra "Không." dù ERP ghi 3.900 L — phán quyết sai chiều theo nguồn nêu ở đề mục ⇒ đóng ở bước khoá (dưới);
  - V5S4-05 (văn bản): luật 2, luật 3 đã sửa chữ cho khớp R5-4 / R5-5.

**Cái giá chấp nhận và điểm còn mở khác (đều S3, không đổi chiều):**
- Đề mục chỉ xét trong cùng câu: đề mục đứng thành câu hay dòng riêng ("ERP. EX-03 hôm nay được cấp 3900 L đúng không?") vẫn được phán quyết như trước Vòng 5 (soát vòng 4: đó là phán quyết sai chiều theo nguồn nêu ở đề mục, V5S4-04 — đóng ở bước khoá).
- aliasTied chọn lần nhắc rõ gần bí danh nhất theo khoảng cách byte: câu nhắc chủ thể hai lần có thể mất phán quyết tuỳ độ dài chữ ở giữa.
- Chờ quyết chính sách (đột biến ghi "CHỜ QUYẾT" trong danh mục): R2-6 (bí danh mơ hồ có ứng viên là cấp cha sau "của": "EX-03 của chương trình mới …" vẫn phán quyết); lần nhắc rõ nào quyết bí danh khi cả hai phía đều có.

**Thứ tự kế tiếp:** KHOÁ cổng theo `docs/PROMPT-BUILD-EAIS.md` §9.6 — vì soát vòng 4 còn thấy phán quyết sai chiều cùng họ (V5S4-02…04), bước khoá dùng luật fail-closed rộng: mọi cụm nguồn / chỉ số khác ở ngữ cảnh mà không có con số riêng của nó ⇒ không phán quyết (giá đã biết: đối chứng Vòng 2 "Bao nhiêu định mức, vậy EX-03 … 3850 L" mất "Đúng.") → bộ đọc số bằng chữ (D-OP-14 = a) → vòng sửa tầng ý định S4 theo ADR-0010 và D-OP-15 (a2 kèm 2 điều kiện) → đóng băng mã S4 → chạy H4 MỘT lần.

### Bước khoá (22/10 trưa, brief BUILD §9.6) — mã `f0e1e93`

**Vì sao mở lại sau R5-4, R5-5:** soát vòng 4 tìm thêm ba dạng cùng họ R5-1 mà hệ vẫn phán quyết SAI CHIỀU theo nguồn nêu ở đề mục
(V5S4-02…04: đề mục kèm "bao nhiêu", kèm danh từ chung, đứng thành câu / dòng riêng). §9.6 chỉ cho mở lại cổng khi gặp phán quyết sai
chiều; vá từng dạng thì họ lỗi chưa đóng (bốn vòng soát, vòng nào cũng còn dạng mới), nên bước khoá đổi sang luật fail-closed rộng rồi KHOÁ.

**Luật (thay luật 3 và phần "cụm ở ngữ cảnh làm chủ" của luật 2, Vòng 5):**
- Cụm nguồn khác / chỉ số khác (`qualifierPhrases`: tên / bí danh nguồn khác nguồn đã dùng; từ khoá của ý định số liệu khác; từ chỉ
  lượng) ở BẤT KỲ đâu trong ngữ cảnh [0, lo), mà KHÔNG có con số nào xen giữa nó và phạm vi hỏi ⇒ không phán quyết: "source" nếu là
  tên nguồn, còn lại "metric". Cụm đã có số của riêng nó ("Đội báo 3k, vậy được cấp 4000 L …") là sự kiện riêng, không chặn.
- Chữ của chính claim không tính: từ khoá của ý định yếu, tên đúng nguồn đã dùng, từ khoá của chính ý định trừ khi nó cũng là tên
  nguồn khác (R5-5). Thuật ngữ vật tư ("dầu") không phải cụm nguồn / chỉ số.
- Bỏ `topicQualifier` (đề mục thuần) và cụm ngữ cảnh trong `owners`: luật mới bao trùm cả hai. Luật 1 (chủ gần nhất) nay chỉ còn tên
  và thuật ngữ.
- Đảo lại ghi chú của luật 3 Vòng 5 ("bản luật rộng hơn … đã thử và bỏ"): cái giá đó nay được chấp nhận — "Bao nhiêu định mức, vậy
  EX-03 … 3850 L đúng không?" mất "Đúng." (đối chứng Vòng 2 chuyển sang ca âm).

**Đo:**
- Test đỏ trước trên `1e5457b`: `TestCongPhanQuyetVong5` 9 ca âm mới, `TestVerdictGap` 4 dòng mới. Ba dòng `TestVerdictGap` cũ đổi nhãn
  "subject" → "metric" / "source" (vẫn không phán quyết; bước khoá bắt trước với nhãn của cụm), ghi lý do ở test.
- Test có PG 459 / 0 / 0, so theo tên không đổi; 117 / H1 / H3 hai đường: 0 dòng khác.
- 716.133 câu corpus + 13 tệp E3: 0 phán quyết mới, 0 đổi chiều; mất 1.118 lượt = 549 câu khác nhau, đều dạng "Bao nhiêu định mức,
  vậy …" (nhãn gốc: phán quyết đúng) — đúng cái giá đã biết.
- Lưới của người soát vòng 4: 138.870 lượt — sai chiều theo chỉ số nêu ở đề mục 448 / 123 / 123 → 0 (bộ chuẩn / bốn tầng / nguồn thấp),
  không cơ sở 1.297 / 2.668 / 2.668 → 0, 0 phán quyết mới, 0 đổi chiều; mất 882 / 222 / 222 phán quyết mô hình chấm là đúng (đề mục
  có "bao nhiêu" cạnh nguồn / chỉ số). Lưới 2 (83.640 lượt): mất 486 (bốn tầng / nguồn thấp: "phiếu (xuất) kho … bao nhiêu"), 0 mới,
  0 đổi chiều. Nhóm câu viết tay (321 lượt): mất 8, 0 mới, 0 đổi chiều.
- Đột biến n43 viết lại: bỏ 42 dòng nhắm mã đã xoá, viết lại 6, thêm KH1–KH7; 185 / 185 dòng chạy bị bắt (CG134 thành tương đương,
  ghi lý do). Danh mục mới n44 cho bộ đọc số bằng chữ: 13 / 13.

**Soát độc lập vòng 1 của bước khoá + D-OP-14 (KHÔNG ĐẠT, hai S2) và bản sửa `017495c`:**
- F1 (do D-OP-14): "… một nghìn hai **lẻ** lít dầu đúng không?" đổi "Không." → "Đúng." — luật 5 không báo dãy kết thúc bằng "linh" /
  "lẻ". Sửa: luật 5 dựng lại trên chính bộ đọc — số bằng chữ là mơ hồ khi giá trị do CÁCH NÓI TẮT quyết (chữ số cuối dãy ngay sau từ chỉ
  hàng, "rưỡi" sau "<hàng> <chữ số>") hoặc dãy kết thúc bằng "linh" / "lẻ" (xấp xỉ; mở rộng có chủ ý: "hai trăm lẻ" cũng thôi phán quyết).
- F2 (có từ trước, SAI CHIỀU): ngoại lệ "cụm có số của riêng nó" nhận mọi số có đơn vị và cả số mốc ("ERP 8h, EX-03 … 3900 L đúng
  không?" → "Không." dù ERP ghi 3.900 L; tương tự "tuần 43", "2 ca", "5 người"). Sửa (mở lại theo §9.6 vì sai chiều): chỉ lượng CÙNG
  CHIỀU ĐO với chỉ số (hay số trần) mới là số của riêng cụm; số mốc và lượng khác chiều đo không tính. Ranh giới có chủ ý (hợp đồng R-1):
  "ERP 1 L, …", "Đội báo 2k, …", "Đội báo 4 khối, vậy …" vẫn là sự kiện riêng của ngữ cảnh — vẫn phán quyết theo nguồn có thẩm quyền.
- Đo bản sửa: bộ tái lập của người soát (275 câu đối kháng): mất 99 phán quyết, 0 mới, 0 đổi chiều; 22 câu số bằng chữ: mất 2; 1.500
  câu đối chứng dương: không đổi. Ở mã sửa: test có PG 463 / 0 / 0 theo tên; 117 / H1 / H3 hai đường 0 dòng khác; corpus 716.133 câu và lưới vòng 4: 0 khác.
- F3 (S3): giá của khoá rộng hơn mẫu "Bao nhiêu định mức, vậy …": câu hai vế tự nhiên "Hôm nay đội báo bao nhiêu? EX-03 được cấp 3850 L
  đúng không?" cũng mất phán quyết (vẫn trả số). Chấp nhận (fail-closed); tần suất tự nhiên CHƯA ĐO (không có nhật ký câu thật).
- Nợ ghi nhận (F4, F5, không sai chiều với dầu): bộ đọc chỉ sửa chữ số CUỐI dãy ("hai trăm năm nghìn" vẫn 205.000, nói thường 250.000;
  "ba triệu hai trăm" = 3.000.200) — ngoài miền chỉ số dầu nên cổng ra "value"; "hai lăm nghìn" = 5.000 (có từ trước); đường nạp lưu
  1.200 thay 1.002 cho tin "một nghìn hai", không có bước nạp lại dữ liệu cũ (dữ liệu giả lập hiện có 0 dạng này).

**Soát độc lập vòng 2 (KHÔNG ĐẠT: phần F1 / F2 đã nêu ĐÃ ĐÓNG; còn ba lỗ SAI CHIỀU cùng họ, có từ trước) và bản sửa `7e1d8ca`:**
- "<chục> trăm" ("Lu 02 … được cấp **mười hai trăm** lít dầu đúng không?" → "Không." vì bộ đọc cộng dồn ra 210): luật 5 coi hàng trăm đứng
  sau hàng chục là nói tắt (178 / 178 dạng "T trăm" được báo; 0 / 360.000 dạng viết đủ báo oan — số đo của người soát).
- Số trần chỉ có đơn vị nhờ chữ "dầu" đứng cạnh ("**Dầu ERP 19-10**, EX-03 … đúng không?" → "Đúng." / "Không." theo kho xuất dù ERP ghi
  3.900 L): không phải lượng của riêng cụm (có thể là ngày, số hiệu). Giá: "Đội báo dầu 4000, vậy …" mất phán quyết.
- Số của cụm KHÁC đứng sau ("**ERP, đội báo 4000 L**, EX-03 … 3850 L đúng không?" → "Đúng."): số riêng của một cụm phải đứng trước cụm
  qualifier kế tiếp.
- Ghi nợ (S3 / S4, không sai chiều với dầu hay câu gượng): số bằng chữ không đơn vị ("ERP hai mươi, …") vẫn là số riêng của cụm trong khi
  "20" thì không; "lẻ" đứng đầu dãy; "một triệu hai trăm" = 1.000.200 (chỉ chặn nhờ miền chỉ số); bỏ `slices.Clone` ở ctxNumbers không test
  nào bắt (giữ Clone; ghi trong danh mục).
- Đo bản sửa: test đỏ trước (`TestAmbiguousNumberWords` 3 dòng, `TestVerdictGap` 3 dòng, `TestCongPhanQuyetVong5` 7 ca âm); test có PG
  474 pass, 0 đổi / 0 mất theo tên; 117 / H1 / H3 hai đường: không dòng nào khác ngoài dòng G1 của làn 2; corpus 716.133 câu + E3: 0 đổi chiều,
  0 phán quyết mới, 0 mất; lưới vòng 4: 0 khác; bộ tái lập vòng 1: 0 khác; bộ tái lập vòng 2 (752 câu): phần đã có bản vá thử của người soát
  (745 câu) trùng bản vá đó; so với trước bản sửa chỉ mất phán quyết (9 câu viết tay, 420 / 704 câu số trần cạnh "dầu", 6 / 7 câu "ERP, đội báo 4000 L, …"), 0 mới, 0 đổi
  chiều; đột biến n43 180 / 180, n44 13 / 13.
- Soát độc lập vòng 3 (vòng cuối theo §9.6) cho bản sửa `7e1d8ca`: bản sửa tự nó ĐẠT — 3 lỗ đã đóng, 0 S1, chỉ bớt phán quyết (người soát
  chứng minh đơn điệu bằng đọc mã và đo trên 150.586 câu lưới); KHÔNG ĐẠT vì còn hai lỗ SAI CHIỀU cùng họ, CÓ TỪ TRƯỚC ⇒ mở lại theo ngoại lệ
  §9.6, bản sửa `be91cd0`:
  - S2-1: số của tên nguồn ĐÃ DÙNG được tính là số riêng của cụm đứng trước ("ERP, phiếu xuất kho 3850 L, EX-03 … 3850 L đúng không?" →
    "Đúng." dù ERP ghi 3.900 L). Sửa: tên + bí danh nguồn đã dùng là ranh giới cụm (không chặn).
  - S2-2: số bằng chữ không đơn vị, không hàng nghìn ("ERP mười chín, …", "Dầu ERP bốn mươi ba, …") được coi là tự mang chiều đo. Sửa: số
    không đơn vị chỉ tự mang chiều đo khi có từ chỉ hàng nghìn trở lên ("bốn ngàn", "3 nghìn") hay hậu tố tiếng lóng của pack ("3k", "4tr");
    "mười chín", "hai chục", "3 trăm" là số nhỏ như "19" (dạng "3 trăm" do điều phối tự tìm thêm, cùng họ).
  - Đo: test đỏ trước (`TestVerdictGap` 7 dòng, `TestCongPhanQuyetVong5` 8 ca âm); test có PG 478 pass, so theo tên 0 đổi / 0 mất; 117 / H1 /
    H3 hai đường 0 dòng khác; corpus 716.133 câu + E3 + lưới: 0 phán quyết mới, 0 đổi chiều, mất 1 ("ERP hai mươi, …" — món nợ vòng 2, nay
    đóng); đột biến n43 187 / 187, n44 16 / 16.
  - Ghi nợ (không sai chiều): "X triệu Y trăm" ("một triệu hai trăm" đọc 1.000.200, chỉ chặn nhờ miền chỉ số); CG67 chỉ có test đơn vị; bỏ
    `slices.Clone` ở ctxNumbers nhiều khả năng là đột biến tương đương.
- Soát độc lập vòng 4 (ngoại lệ sai chiều, người soát thứ năm) cho `be91cd0`: **ĐẠT CÓ GHI CHÚ** — S2-1, S2-2 đã đóng (mọi tên nguồn đã dùng
  thành ranh giới, kể cả bộ fact tự dựng cho bảng tổng hợp / báo cáo ngày / hoá đơn / cảm biến; số nhỏ bằng chữ và nhân "trăm" / "chục" hết
  phán quyết ở mọi dạng); 0 S1, 0 S2; chỉ bớt phán quyết (đọc mã + lưới 260.411 câu duy nhất qua hai nhị phân: 0 phán quyết mới, 0 đổi
  chiều); ca dương R-1 giữ nguyên. Cái giá mới: "được cấp 39 trăm", "ba k" và số nhỏ bằng chữ ở ngữ cảnh ("ERP hai mươi, …") mất phán quyết.
- Ghi nợ (S3, có từ trước, không sai chiều theo hợp đồng R-1): số gán theo vị trí cho cụm chưa có số qua chữ trung tính / từ khoá ý định /
  "bao nhiêu" ("ERP, xuất kho 3850 L, …" → "Đúng.", đọc được như số của ERP — cùng cấu trúc ca dương R-1); tên nguồn đã dùng đứng SAU số
  ("ERP: 3850 L (phiếu xuất kho), …"). S4: biến thể chính tả tên nguồn ("hóa đơn" / "hoá đơn") chỉ có ở từ khoá, không là ranh giới.

**KHOÁ HẲN (lệnh ca MVP ALL-IN của chủ dự án, 22/10):** không mở vòng 5. Lỗ mới cùng họ chỉ ghi nợ, giữ fail-closed.

**KHOÁ:** từ mã này cổng phán quyết không nhận thêm luật. Lỗ mới cùng họ: ghi nợ, giữ fail-closed (không phán quyết); chỉ mở lại khi
gặp phán quyết sai chiều (brief §9.6).
