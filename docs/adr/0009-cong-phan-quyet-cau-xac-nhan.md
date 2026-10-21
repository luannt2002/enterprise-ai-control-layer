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

Một câu xác nhận chỉ qua cổng khi phần chữ còn lại sau khi trừ đi các phần ĐÃ HIỂU là rỗng. Phần đã hiểu gồm:

| Phần | Lấy từ |
|---|---|
| tên thực thể | S3 (mention) |
| mốc thời gian đọc được | S2 (span kỳ / ngày) + từ chỉ kỳ đứng ngay trước ("ngày" của "ngày 13/10") + mốc viết bằng chữ khớp kỳ đã đọc ("hôm nay"); kỳ MẶC ĐỊNH của pack KHÔNG tính, và câu phải có đúng MỘT kỳ đọc được |
| nghĩa chỉ số | từ khoá của CHÍNH ý định đã khớp (ý định "yếu" — chỉ số theo quy ước của pack — không được phán quyết) + thuật ngữ vật tư của chính ý định ("dầu") + cách gọi nguồn có thẩm quyền CAO NHẤT của chính chỉ số ("phiếu xuất kho … đúng không?"); nguồn thấp hơn ("ERP ghi …") là claim về con số của nguồn đó nên không được giải thích |
| con số xác nhận | mọi con số trong phạm vi hỏi phải CHÍNH LÀ giá trị được xác nhận (số khác ⇒ không phán quyết), có đơn vị cùng chiều đo với chỉ số (viết rõ, hoặc suy từ thuật ngữ vật tư: "4k dầu"), hoặc số có dạng lượng ("4k", "4.000", "bốn ngàn"). Dấu xấp xỉ / chặn / mẫu số ("khoảng", "~", ">", "/ngày") nằm ngoài span số nên thành phần dư — không cần nhánh riêng |
| cụm xác nhận | `confirm_markers` của pack |
| từ trung tính | khoá pack MỚI `claim_neutral_words`: lịch sự, trợ từ, trợ động từ ("cho hỏi", "ạ", "đã", "vậy"…) |
| dấu câu ngăn cách | `, . ; : ? ! ( ) [ ] " ' “ ” ‘ ’ … - – —`; mọi ký tự khác (cả "~", ">", "%", "/") phải được giải thích |

- Mọi thứ khác — số thứ hai, nhãn ("bể 7", "phiếu 912", "ca 2"), mốc chưa đọc ("Q3", "sáng"), phủ định ("không", "chưa"), so sánh / chặn ("trên", "hơn"), nguồn hay nghĩa khác ("theo ERP", "đội báo"), từ lạ — đều là phần dư ⇒ không phán quyết.
- **Danh sách DUY NHẤT mới là danh sách CHO PHÉP (từ trung tính), không phải danh sách CHẶN.** Thiếu một từ chỉ làm câu mất phán quyết (an toàn), không bao giờ làm phán quyết sai. Mở rộng nó chỉ để tăng độ phủ, có test.
- Số trần trơn ("được cấp 3850") không tự giải được chiều đơn vị ⇒ không phán quyết. Luật "số trần ngay sau từ khoá" của bản phân tích bị bác: từ khoá danh từ ("phiếu kho 2") làm nhãn thành con số.
- Phạm vi hỏi là mệnh đề chứa cụm xác nhận và các mệnh đề sau nó; mệnh đề hỏi không chứa số ("…, vậy đúng không?") thì phạm vi hỏi là cả câu.
- Mệnh đề đứng TRƯỚC phạm vi hỏi (ngữ cảnh) được chứa thêm: lượng tự mang đơn vị ("Định mức 5.000 L, vậy …"), từ khoá của mọi ý định, cách gọi nguồn của mọi chỉ số, từ chỉ lượng của pack — tức sự kiện người hỏi nêu về chỉ số khác. Số trần ở ngữ cảnh (có thể là nhãn / mốc) và mọi phần khác vẫn phải được giải thích ("Không tính ERP, …" vẫn mất phán quyết).

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
- Soát đối kháng độc lập (vai D) trước khi chạy H3.
