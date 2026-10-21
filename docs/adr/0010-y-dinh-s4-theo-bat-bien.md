# ADR-0010 — Tầng ý định S4 theo bất biến, không theo danh sách cụm đóng

- **Ngày:** 21/10/2026, ca ngày.
- **Trạng thái:** **ĐỀ XUẤT, chưa áp mã.**
  - Vòng sửa này là vòng mà D-OP-13 giao cho H4 chấm.
  - H4 đã đóng băng bằng cam kết băm (`15f04f9`) TRƯỚC ADR này và trước mọi dòng mã S4.
  - Số 0010 từng được một bản nháp C2 dùng ở máy; bản đó bị bác và không lên main.
- **Liên quan:**
  - `docs/04-core-mvp.md` §4 (route), §5 (lớp hỏng);
  - ADR-0009 (cổng phán quyết đứng sau một ý định đúng);
  - ADR-0006 §10 (H3, H4);
  - D-OP-11 (tập ứng viên hỏi lại, chờ chủ dự án), D-OP-13 (thứ tự vòng S4);
  - báo cáo ca 21/10 mục 5.

## Bối cảnh (đã đo)

**Lỗi cấp lớp, không phải ca lẻ:**
- H1: 13/15 ca đỏ sai đầu tiên ở S4.
- H3: 20/25 ca đỏ sai đầu tiên ở S4. Cả 20 ca cũng đỏ ở mốc `016422e`, trùng nguyên văn câu trả lời.
- Một người kiểm chéo độc lập chạy lại 13 ca và đồng ý cả 13.
- Theo `CLAUDE.md` §2.3, lỗi cấp lớp phải sửa ở tầng thiết kế.

**Cơ chế hiện hành** (`intelligence/intent/intent.go`), theo bốn cụm:

1. **C3 (9 ca H3): không khớp cụm nào thì từ chối.**
   - Một ý định chỉ được xét khi có ít nhất một cụm từ khoá khớp (`Classify`: `counts[i] == 0` thì bỏ qua).
   - Không cụm nào khớp ⇒ kết quả rỗng ⇒ router từ chối OUT_OF_SCOPE (`router.go:185-190`).
   - Điều này xảy ra cả khi câu có thực thể đã giải và thuật ngữ vật tư của pack.
2. **C5 (4 ca): đếm từ khoá trên cả câu.**
   - `keywordHits` đếm trên CẢ CÂU. Chỉ câu có cụm xác nhận mới được thu hẹp về mệnh đề hỏi (`FocusClause` với `FocusMarkers`).
   - Vì vậy từ khoá của vế quy trình hay vế ngữ cảnh rò sang ý định số liệu. Ví dụ ở H3: chữ "lệch" của vế quy trình được đọc thành chỉ số chênh lệch.
3. **C4 (4 ca): người thắng chọn theo số cụm khớp.**
   - Người thắng là ý định có (priority, SỐ CỤM khớp) cao nhất. Hoà mà câu không có cụm so sánh nằm trong danh sách đóng `comparison_markers` thì thành mơ hồ.
   - Số cụm khớp không phải tín hiệu nghĩa: cặp tối thiểu "đo được" → "đo" đủ lật kết quả.
4. **C8 (3 ca): ứng viên hỏi lại bị gộp.**
   - `fill` / `clashing` gom ứng viên của bí danh mơ hồ CẢ khi khe đã điền, nên tập hỏi lại trộn thực thể đã giải với ứng viên.
   - Hành vi này đang được ghim ở `intent_test.go:208` và `intent_hoi_quy_test.go:52`.
   - Chờ chủ dự án quyết ở D-OP-11.

**Cùng họ lỗi với ADR-0009: danh sách đóng hỏng khi THIẾU một mục.**
- Ở cổng phán quyết, thiếu ⇒ không phán quyết: hỏng an toàn.
- Ở S4, thiếu ⇒ TỪ CHỐI: sai lặng, trái P5 (mơ hồ phải hiện ra).

## Bất biến đề xuất

**(a) KHÔNG KHỚP Ý ĐỊNH ≠ NGOÀI PHẠM VI.**
- Câu có tín hiệu thuộc phạm vi mà không ý định nào khớp ⇒ hỏi lại nghĩa số liệu, liệt kê các ý định đòi thuật ngữ đó. Không từ chối.
- Tín hiệu thuộc phạm vi gồm: thực thể đã giải, thuật ngữ vật tư của pack, cụm hỏi lượng, cụm xác nhận.
- Từ chối chỉ khi câu không có tín hiệu nào.
- (a) là chiều ngược của `NeedTerm`: `NeedTerm` xử lý câu có từ khoá mà thiếu thuật ngữ; (a) xử lý câu có thuật ngữ mà thiếu từ khoá.

**(b) Ý ĐỊNH LẤY THEO MỆNH ĐỀ HỎI.**
- Trong câu nhiều mệnh đề, mệnh đề chứa dấu hiệu hỏi quyết định ý định. Dấu hiệu hỏi gồm: cụm hỏi lượng, cụm xác nhận, dấu "?".
- Các mệnh đề khác chỉ là ngữ cảnh.
- (b) mở rộng cơ chế `FocusClause` có sẵn: từ chỗ chỉ dùng `FocusMarkers` sang dùng cả `QuantityMarkers` và dấu "?".

**(c) TẬP ỨNG VIÊN HỎI LẠI = ỨNG VIÊN CỦA CỤM MƠ HỒ.**
- Chỉ áp khi chủ dự án chọn (b) ở D-OP-11.
- Thực thể đã giải không vào danh sách hỏi lại; nó chỉ dùng để gỡ mơ hồ. Riêng câu có hai thực thể đều đã giải ("EX-03 và RL-02") thì giữ cách liệt kê hiện tại.

**C4: CHƯA có bất biến đủ chắc.**
- Hướng: nhận phép so sánh theo cấu trúc (hai vế chỉ số + từ so sánh bậc "hơn") thay vì theo danh sách cụm. CHƯA ĐO.
- Để sau (a)(b)(c).

## Ca âm phải có trước khi sửa

Ca âm và ca dương viết mới, dạng cặp tối thiểu. Không chép câu của H3, không đọc H4.
- **Với (a):** câu ngoài phạm vi thật vẫn bị từ chối:
  - không thực thể, không thuật ngữ ("VN-Index bao nhiêu?");
  - thuật ngữ trùng chữ khác nghĩa ("dầu ăn", "dầu gội");
  - tên máy xuất hiện trong câu chuyện phiếm, không hỏi số liệu.
- **Với (b):**
  - từ khoá chỉ nằm ở vế ngữ cảnh thì không được quyết định ý định;
  - câu một mệnh đề không đổi hành vi;
  - câu ghép số liệu + quy trình (HYBRID) vẫn ghép đúng.
- **Với (c):**
  - một thực thể đã giải + một bí danh mơ hồ;
  - hai bí danh mơ hồ;
  - hai thực thể đã giải (đối chứng: không đổi).

## So rule / model / hybrid (D-OP-13 điều kiện 1, cổng `CLAUDE.md` §19)

- Bước 1 dùng luật tất định theo (a)(b)(c).
- Classifier bằng model chỉ được mở khi luật đã sửa mà vẫn trượt trên các ca đo được. Nay chưa so được model hay hybrid: chưa có khoá, chưa có ngân sách (G7 đang chặn).
- Phần so model / hybrid ghi là BLOCKED, không suy ra "cần model" từ việc S4 là nút thắt.

## Không làm

- **Không thêm từ khoá, cụm so sánh hay lời chào theo từng ca H3.** Đã đo trên bản sao pack: thêm "xuất", "số liệu", "lượng" làm 3 ca H3 xanh. Đó là vá theo H3.
- **Không đụng cổng phán quyết (C1)** trong ADR này. Một số câu xác nhận đúng ý định rồi vẫn bị cổng giữ phán quyết (607, 646): không nới cổng.

## Chứng minh (theo thứ tự)

1. H4 đóng băng trước commit đầu tiên của vòng này: **ĐÃ XONG**, `15f04f9`.
2. Test đỏ trước: các ca âm và ca dương ở trên; thêm các ca H1 tương ứng (H1 là bộ hồi quy).
3. Sửa nhỏ nhất trong `intent.go`, thêm router nếu (a) cần nhánh hỏi lại mới. Không thêm khoá pack trừ khi bất biến đòi.
4. Chạy bộ 117 + H1 trên cả hai đường. Mọi ca đổi nhãn phải giải thích được.
5. Đột biến: danh mục mới cho S4; chạy lại n43 vì cổng phán quyết phụ thuộc ý định.
6. Soát đối kháng độc lập. Riêng với (a): đo tỉ lệ hỏi thừa trên câu ngoài phạm vi thật.
7. Đóng băng mã S4, rồi chạy H4 MỘT lần. Báo cáo theo tầng, có đối chứng mốc; sau đó H4 thành bộ hồi quy.

## Rủi ro

- (a) làm tăng số lần hỏi lại. Cần đối chứng âm lớn trước khi ghép (bài học từ C2 thử 1).
- (b) có thể đổi hành vi câu ghép đang đúng ở bộ 117 / H1. CHƯA ĐO.
- (c) đổi hợp đồng hỏi lại, cả router lẫn mẫu câu. Chỉ làm sau khi có D-OP-11.
