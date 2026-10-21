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
| R2-1 | CAO | một cụm xác nhận, HAI phát biểu cùng con số ("EX-03 … 3850 L, dự án A 3850 L, đúng không?" — dự án A thật là 5.050 L) ⇒ "Đúng." | ĐÃ ĐÓNG: con số xác nhận nhắc lại chỉ là cùng claim khi đứng liền kề (chỉ cách khoảng trắng / ngoặc) |
| R2-2 | TB-CAO | mệnh đề hỏi chỉ nêu cấp cha ("…, dự án A đúng không?"), luật "phạm vi hỏi có tên thì phải có chủ thể" dùng phạm vi đã lùi về đầu câu | ĐÃ ĐÓNG: luật chủ thể dùng đầu mệnh đề chứa cụm xác nhận |
| R2-3 | TB | lượng có dấu chấm / phẩy mà đơn vị chỉ suy từ thuật ngữ ("dầu 1,2") | ĐÃ ĐÓNG: không tính là có đơn vị (cái giá: "dầu 3.850" mất phán quyết) |
| R2-4 | THẤP-TB | tên nguồn ở mệnh đề NGỮ CẢNH không chứa lượng ("Phiếu kho, RL-02 … đúng không?") không đối chiếu nguồn đã dùng | ghi lại, chưa làm |
| R2-5 | TB | máy và dự án cùng nêu, nối bằng dấu rời ("Lu 02 : Dự án A …") — hệ chọn máy làm chủ thể | ghi lại: quyết định thiết kế (chủ thể = thực thể cụ thể nhất) |
| R2-6…R2-9 | THẤP | bí danh mơ hồ có ứng viên là cấp cha; từ chỉ ngày trước span tháng / tuần; tên sai dấu vẫn khớp; "3,850 lít" đọc là nghìn | ghi lại |

- Ba dòng vá R2-1 / R2-2 / R2-3 là bản vá thử CỦA NGƯỜI SOÁT; phiên chính áp nguyên văn và đo lại bằng test + bộ câu có đáp án độc lập của người soát — chưa có người soát thứ ba cho riêng phần này.
- Test mới `TestCongPhanQuyetSoatDoiKhangVong2` (7 câu không phán quyết + 5 đối chứng), ghim thêm ba nhánh chưa có test đỏ: cấp cha nhiều tầng (danh mục giả lập 4 tầng `seed/entities-bon-tang.yaml`), từ chỉ lượng ở ngữ cảnh, hai kỳ đọc được.
- Đo: test có PG 451/0/0; bộ 117 117/117 và H1 89/104 hai đường, 0 ca đổi nhãn, số ca có phán quyết không đổi so với T2.1 (15; 5); hai phát biểu một cụm (20.000 câu có đáp án độc lập): 11.375 → 0 phán quyết; văn phạm có dấu (60.000 câu): 33.209 phán quyết đúng, 0 sai; n43 41 / 41.
- Cái giá ghi nhận (S3): câu không dấu mất 100% phán quyết; trần một lần đo áp cho tổng nhiều ngày / nhiều máy làm mất phán quyết đúng ("EX-03 tuần trước … 21000 L").
