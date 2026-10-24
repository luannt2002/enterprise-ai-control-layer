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

## Vòng sửa S4 (22/10 trưa) — mã `723a570`, áp (a) theo D-OP-15 = a2 kèm 2 điều kiện

**Đã áp (bất biến (a)), trong nhánh không từ khoá nào khớp và không có ý định thiếu thuật ngữ:**
- Tín hiệu phạm vi = có thực thể (đã giải hay bí danh mơ hồ) VÀ có thuật ngữ mà một ý định đòi (định nghĩa D1s của bước điều tra).
  Thiếu một trong hai ⇒ như cũ (từ chối ngoài phạm vi).
- **a2** (trả lời theo ý định chung, không phán quyết) chỉ khi đủ cả bốn: có cụm xác nhận; đúng MỘT ý định yếu đòi thuật ngữ của câu;
  mọi chữ còn lại — ngoài tên, mốc (kể cả "ngày" trước mốc số), lượng, thuật ngữ, cụm xác nhận, từ trung tính của pack — là mảnh từ
  khoá của ý định riêng mà ý định chung quy về (cùng chỉ số theo `metric_by_term`: "xuất" của "kho xuất"); và có ÍT NHẤT một mảnh như
  thế. Từ trung tính là mảnh từ khoá của chỉ số khác ("đo" của "đo được") không được tha. Câu xác nhận không nêu chỉ số ("Hôm qua EX-03
  4000 L dầu đúng không?") hỏi lại — luật BR-FUEL-02 của tenant (`docs/tenants/construction/01-business-map.md`).
- Còn lại ⇒ **a1**: hỏi lại nghĩa số liệu (`clarify_intent`), kể các ý định số liệu đòi thuật ngữ (không kể ý định chung, quy trình,
  chào hỏi), chỉ kể ý định người hỏi có capability (không còn ⇒ từ chối như cũ, không lộ nhãn); thực thể đã giải ngoài phạm vi dữ liệu
  của người hỏi ⇒ từ chối PERMISSION_DENIED trước khi hỏi lại; bí danh mơ hồ ⇒ câu hỏi lại kèm ứng viên.
- **Điều kiện (i):** `assumption` của ý định (pack, lời chủ dự án nguyên văn) nối vào MỌI câu trả lời theo ý định chung — có số, chưa có
  dữ liệu, chờ đối soát — kể cả câu khớp từ khoá yếu có từ trước ("bao nhiêu dầu", "hết bao nhiêu dầu").
- **Điều kiện (ii):** bộ 700 câu TRONG phạm vi (20 động từ của chỉ số khác, không là từ khoá × 7 thực thể × 5 khung; do điều phối
  soạn, KHÔNG độc lập): 0 câu a2 trả số kho xuất; 476 câu trước từ chối nay hỏi lại; 119 câu dạng "… xài / hao / tiêu … bao nhiêu dầu?"
  trả số kho xuất qua nhánh từ khoá yếu "bao nhiêu" CÓ TỪ TRƯỚC (không đổi số câu), nay luôn kèm câu giả định — câu mở D-OP-16.

**Chưa làm ở vòng này:** (b) (luật thử của bước điều tra không cứu ca H1 nào, có rủi ro câu ghép); (c) (chờ D-OP-11); C4 (chưa có
bất biến). B01–B08 và X01 của 42 ca thiết kế vẫn đỏ vì thế.

**Đo** (nền `0c152e9` = trước S4; mã cuối `723a570`):
- Test đỏ trước: 4 test mới (`TestS4*`); test có PG 463 / 0 / 0, so theo tên: 0 đổi, 0 mất. Một test cũ đổi hợp
  đồng có chủ ý (`TestDoiCachVietKhongDoiNghia`: câu chung chung trả đúng số ấy + câu giả định — D-OP-15 (i)).
- Bộ 343 câu ngoài phạm vi sinh tất định (băm `a024f59a…`): 0 lật ở S1, S2, S3, S3b, S5; S6 (6 câu "giới hạn đã biết": tên máy +
  "dầu" trong chuyện phiếm) từ chối → hỏi lại. 42 ca thiết kế: xanh 20 → 33.
- Bộ 117: 117 / 117 (16 dòng khác đều là câu giả định thêm vào). H1 89 → 92 / 104 và H3 109 → 113 / 134, hai đường giống hệt — **KHÔNG
  phải bằng chứng**: ba chỉnh sau cùng (chữ nêu nghĩa, kiểm phạm vi, kể ứng viên) có động cơ từ việc đọc 4 ca H3 (542, 547, 625, 664);
  nguyên tắc lấy từ BR-FUEL-02 và `CLAUDE.md` §21, có test riêng. H4 mới là phép chấm.
- Corpus phán quyết 716.133 câu: 0 phán quyết mới, 0 đổi chiều, 0 mất; E3 f1: 735 câu từ chối → hỏi lại, 0 câu trả số.
- Đột biến: danh mục mới `scripts/dot-bien/n45-y-dinh-s4.sh` 25 / 25 bị bắt; S7, S10 tương đương trên pack hiện có (ghi lý do).

**Soát đối kháng vòng 1 (KHÔNG ĐẠT có điều kiện; 0 S1 do chính vòng S4) và bản sửa `32771ff`:**
- S4R-01 (S2): a2 coi mọi thuật ngữ và mọi lượng có đơn vị là chữ trung tính ("Đơn giá dầu xuất … là 4100 đúng không?", "… xuất 7 tấn dầu
  …" bị trả số kho xuất — vi phạm (ii)). Sửa: chỉ che thuật ngữ của ý định chung và lượng cùng chiều đo với chỉ số (Options.QuantityFits).
- S4R-04 (S2): câu ngoài phạm vi có thực thể + "dầu" nghĩa khác (dầu nhớt, dầu thuỷ lực) bị hỏi lại (84,6% nhóm đó trên bộ mù của người
  soát). Sửa: pack thêm vật tư "dầu không phải nhiên liệu" (cụm dài thắng "dầu", có đơn vị nên là đại lượng khác) — kéo theo sửa một câu
  trả lời sai nghĩa CÓ TỪ TRƯỚC ("EX-03 hôm nay nhận bao nhiêu dầu nhớt?" từng trả số dầu diesel kho xuất).
- S4R-06 (S3): "duoc" không dấu trong câu có dấu được coi là "được". Sửa: chữ còn lại so khớp trong cả câu, cùng luật dấu với từ trung tính.
- S4R-08 (S3): pack nay chặn lúc nạp ý định có câu giả định mà mẫu câu (của nó, no_data, needs_review) thiếu {assumption_note}.
- Đo bản sửa: test đỏ trước (7 câu S4 mới, `TestLoadPackCauGiaDinh`, 1 ca chặn pack); test có PG 464 / 0 / 0, so theo tên 0 đổi, 0 mất;
  117 / H1 / H3 hai đường: 0 dòng khác so với trước bản sửa; corpus 716.133 câu: 0 đổi; bộ 343 câu ngoài phạm vi: S6 về lại 6 / 6 từ chối,
  0 lật. Trên các bộ của người soát vòng 1:
  - bộ mù 368 câu: hỏi lại 64 → 43, từ chối 105 → 129;
  - bộ "dầu nghĩa khác" 115 câu: hỏi lại 28 → 7;
  - 4 câu "dầu ăn" / "dầu nhớt" CÓ TỪ TRƯỚC trả nhầm số dầu diesel nay bị từ chối;
  - bộ a1 (121 câu xác nhận nêu chỉ số khác: giá, đơn giá, nhân công, giờ máy, tấn …): a2 trả số kho xuất 78 → 10. 10 câu còn lại nêu lượng
    đổi được sang lít ("3 khối", "4k", "bốn nghìn", "4,1 triệu") hay chữ "kho" của "kho xuất", đều kèm câu giả định. "4,1 triệu" có thể là
    tiền: ghi nợ.
  Bộ (ii) 700 câu của điều phối không đổi. Đột biến n45: 31 / 31.
- Ghi nợ, KHÔNG làm ở vòng này (đều có từ trước S4, H4 đo):
  - S4R-02: nhánh từ khoá yếu ("bao nhiêu", "tổng") + động từ chỉ số khác vẫn trả số kho xuất kèm câu giả định — 24 / 136 câu "nghĩa khác"
    trên bộ mù của người soát (17,6%), 0 câu qua a2 — câu mở D-OP-16, mặc định (a);
  - S4R-03: từ khoá khớp sai nghĩa ở đường từ khoá ("nhà cung cấp" khớp "cấp", "tin nhan" khớp "nhận", "cảnh báo" khớp "báo") — câu trả
    lời sai nghĩa KHÔNG nhãn, chạm ngưỡng "0 câu sai nghĩa không nhãn" của §9.3;
  - S4R-05: câu trong phạm vi không có chữ "dầu" ("Hôm nay EX-03 xuất 3850 L đúng không?") vẫn bị từ chối;
  - S4R-07: câu từ chối vì quyền vẫn mang tên ý định; chênh hành vi giữa bí danh có / không tồn tại cho người ngoài phạm vi.

**Soát đối kháng vòng 2 (KHÔNG ĐẠT có điều kiện; 0 S1 mới — 351 câu trả lời theo ý định chung trên mọi bộ đều kèm câu giả định, không câu
nào "Đúng." / "Không.") và bản sửa `a665298`:**
- F1 (S2, hồi quy do bản sửa vòng 1): "Hôm nay dự án Cầu A xuất 5050 L dầu đúng không?" bị hỏi lại vì "dự án" (thuật ngữ project) không còn
  được che. Sửa: thuật ngữ trùng tên LOẠI của một thực thể đã giải trong câu là chữ chỉ loại, được che. "Hôm nay dự án EX-03 xuất …" vẫn
  hỏi lại (EX-03 là thiết bị); bí danh mơ hồ không góp loại (thiếu thì rơi về hỏi lại).
- F2 (S2, hồi quy do bản sửa vòng 1): số trần không đứng cạnh "dầu" ("Dầu xuất hôm nay của EX-03 là 3850 đúng không?") bị hỏi lại — nhánh
  che số trần bị bỏ nhầm là mã chết (đột biến S18 "sống" vì thiếu ca). Sửa: che lại số trần, có ca ghim.
- F3 (S2, lõi dùng chung): `generic()` đọc ý định chung đầu tiên trước khi đếm ⇒ sập với pack có ý định đòi thuật ngữ mà không có ý định chung
  cho thuật ngữ đó (chưa xảy ra với pack cầu đường). Sửa: đếm trước; test tổng hợp.
- F4 (S2, test không chứng minh được): chỉ 2 / 11 bí danh của other_oil có test. Sửa: bảng test 11 bí danh; đột biến bỏ từng bí danh.
- Đo bản sửa: test đỏ trước (panic ở test tổng hợp; 4 câu F1 / F2); test có PG 476 pass, so theo tên 0 đổi, 0 mất (2 test mới); 117 / H1 /
  H3 hai đường: 0 dòng khác. Trên 1.830 câu (1.130 câu của hai người soát + 700 câu (ii) của điều phối): chỉ 14 câu đổi — đúng 7 câu F1 và
  7 câu F2, từ hỏi lại về lại trả lời theo ý định chung (có câu giả định) như trước vòng 1; bộ (ii), bộ a1, bộ mù: 0 đổi. Corpus 716.133
  câu + E3 + lưới của người soát khoá: 0 phán quyết mới, 0 đổi chiều, 0 mất. Đột biến n45 (46 dòng): 46 / 46 bị bắt.
- Ghi nợ (không sửa ở vòng này):
  - F5 (cùng họ S4R-04): danh sách "dầu" nghĩa khác là danh sách đóng — "dầu cọ", "dầu dừa", "dầu phanh" … chưa liệt kê vẫn bị hỏi lại
    (nhóm này trên bộ mù vòng 1: hỏi lại 84,6% → 35,9%);
  - F6 (cùng họ S4R-01): lượng tiền không có đơn vị chữ ("4 củ", "4tr", "150k", ký hiệu "₫", "$") được đọc thành lít ⇒ a2 trả số kho xuất,
    CÓ câu giả định;
  - F7 = S4R-02: nhánh từ khoá yếu ("bao nhiêu", "tổng") + động từ chỉ số khác vẫn trả số kho xuất kèm câu giả định — chờ D-OP-16;
  - F8 = S4R-03 (có từ trước, không đổi qua các vòng): từ khoá khớp chữ con ("ghi nhận" ⇒ "nhận", "nhà cung cấp" ⇒ "cấp", "cảnh báo" ⇒
    "báo") cho câu trả lời sai nghĩa KHÔNG nhãn, có câu kèm "Đúng." theo dầu kho xuất. Chạm ngưỡng S4 "0 câu sai nghĩa không nhãn" (§9.3):
    H4 đo;
  - S3: chữ gõ lẫn dấu ("xuat" trong câu có dấu) và lời đệm ("anh ơi", "vừa", "khoảng", "ca sáng") làm câu kho xuất hợp lệ bị hỏi lại.
- Soát vòng 3 (vòng cuối theo §9.6): kết quả ngay dưới.

**Soát đối kháng vòng 3 (vòng cuối theo §9.6) trên `a665298`: KHÔNG ĐẠT có điều kiện — F1–F4 đã đóng, 0 S1 mới, 0 lỗi thuộc ngoại lệ §9.6
(an ninh, phán quyết sai chiều, mất dữ liệu) ⇒ mã S4 ĐÓNG BĂNG ở `a665298`; các lỗi còn lại ghi nợ:**
- Số đo của người soát: trên 2.173 câu bộ cũ, chỉ đổi đúng 14 câu so với `7a006a8` (7 F1, 7 F2). 454 câu trả lời theo ý định chung: 0 câu
  "Đúng." / "Không.", 0 câu thiếu câu giả định. 117 / H1 / H3 đường bộ nhớ: 0 dòng nội dung khác. Quyền: không rò. 17 / 17 đột biến mới bị bắt.
- Điều kiện (ii) của D-OP-15 (động từ của chỉ số khác): đạt trên 0 / 105 câu mù của người soát, 0 / 700 câu lưới và 0 / 700 câu của điều phối.
- F6' (S2, cùng họ F6 / S4R-01; phạm vi mới do che lại số trần): số tiền có hậu tố "củ", "tr", "triệu" đứng xa chữ "dầu" bị che như số trần
  ⇒ a2 trả số kho xuất (lít), CÓ câu giả định, không phán quyết. Ví dụ: "Dầu xuất cho EX-03 hôm qua 4 củ đúng không?". Trên bộ 120 câu số
  tiền của người soát, sáu dạng ("4 củ", "4,1 triệu", "5 tr", "4tr", "15 triệu", "150k") từ 0 lên 60 / 72 câu; "4 triệu đồng", "21500 đ"
  vẫn hỏi lại. Sửa đúng tầng là S2 / pack (đánh dấu hậu tố tiền), không phải S4. Đây là nhóm số tiền, không phải động từ của điều kiện (ii).
- S4R-03 (có từ trước): "… trả nhà cung cấp tiền dầu 5050 đúng không?" → số dầu kho xuất, KHÔNG nhãn ("cấp" nằm trong "cung cấp").
- Hai thực thể khác loại trong câu ("EX-03 của dự án Đường B"): hệ lấy thực thể đầu và bỏ thầm thực thể sau. Lỗi này có từ trước, ở tầng chọn
  thực thể; F1 làm nhánh a2 cũng đi qua đó.
- Chốt `len(gen) != 1 || MetricByTerm[term] == ""` thiếu test cho nửa "chỉ số rỗng": đột biến sống với pack tổng hợp; pack hiện có không với tới.
- Luật chữ chỉ loại so tên thuật ngữ với tên loại thực thể (hai không gian tên đều do pack đặt): thuật ngữ trùng tên loại mà không phải chữ chỉ
  loại sẽ bị che nhầm. Pack cầu đường chỉ trùng ở "project". Đây là ghi chú cho người viết pack.
- Ý định yếu đòi ≥ 2 thuật ngữ khác chỉ số: chỉ số lấy theo thuật ngữ đứng đầu câu. Pack hiện có không với tới.
- Thêm test, không đổi mã (`f8b295e`): `TestYDinhChungKhongChiSoThiHoiLai` ghim nửa "chỉ số rỗng" của chốt ý định chung; đột biến S10 bật lại,
  S49 mới — cả hai bị bắt.

**Đóng băng:** mã tầng ý định S4 ở `a665298`. H4 chạy MỘT lần trên mã này (kèm bản sửa cổng phán quyết `be91cd0`, chỉ bớt phán quyết).

## Kết quả H4 (22/10 14:59, chạy MỘT lần ở `57409f2`)

Trên dữ liệu giả lập, chưa có người nghiệp vụ duyệt. Bằng chứng thô: `docs/g7/holdout-h4/ket-qua-h4/`; đối chứng mốc `7d72e33`:
`docs/g7/holdout-h4/ket-qua-h4-moc/`.

- **104 / 119 ở cả hai đường** (bộ nhớ, PostgreSQL), phát lại 119 / 119, 0 lệch, 0 canary. Mốc 102 / 119: 2 ca đỏ → xanh (744, 745), 0 ca
  xanh → đỏ; 15 ca đỏ đều đã đỏ ở mốc.
- Tầng sai đầu tiên của 15 ca đỏ: một người phân tích chạy lại từng ca qua pipeline thật (vết từng tầng, cặp câu tối thiểu); một người kiểm
  chéo độc lập chạy lại 13 ca (mọi ca bị gán S4 hoặc sai nghĩa + mẫu) — **đồng ý 13 / 13** về tầng và nhãn.

| Ca | Lớp | Tầng sai đầu tiên | Sai nghĩa không nhãn | Cụm |
|---|---|---|---|---|
| 705 | representation | cổng phán quyết | không | C1 — đóng thừa ("kho cấp": "kho" không được giải thích), an toàn |
| 716 | representation | S4 | không | C3 + S4R-05 — câu không có chữ "dầu" bị từ chối |
| 740 | ambiguity | S4 | không | N1 (mới) — hoà ý định ("cảm biến báo") bỏ rơi ứng viên thực thể |
| 758 | semantic | S4 | **có** | C4 — "chênh" ép cặp mặc định thay cặp người hỏi nêu |
| 806 | source_conflict | S4 | không | C3 — danh sách đóng (a2 thành a1) |
| 808 | source_conflict | cổng phán quyết | không | C1 — đóng thừa |
| 810 | source_conflict | S4 | không | C5 — vai mệnh đề rò rỉ (bất biến (b) chưa áp) |
| 835 | unsupported_answer | cổng phán quyết | không | C1 — đóng thừa |
| 837 | unsupported_answer | cổng phán quyết | không | C1 — đóng thừa (số trần ở ngữ cảnh) |
| 838 | unsupported_answer | cổng phán quyết | không | C1 — đóng thừa |
| 841 | unsupported_answer | S4 | không | C3 + S4R-05 |
| 847 | unsupported_answer | S4 | **có** | C5 — vai mệnh đề rò rỉ |
| 848 | unsupported_answer | S4 | không | C3 + S4R-05 (kèm C1) |
| 854 | routing | S4 | **có** | C4 — tập chỉ số "thắng hết" |
| 862 | routing | S4 | không | C3 — danh sách đóng (lời chào) |

**Ngưỡng S4 của brief §9.3: KHÔNG ĐẠT.** S4 là tầng sai đầu tiên ở 10 / 119 = 8,4% (ngưỡng ≤ 5%); 3 câu trả lời sai nghĩa không nhãn
(ngưỡng 0). Năm ca dừng ở cổng phán quyết đều là đóng thừa (không phán quyết sai chiều), cổng đã khoá. Việc tiếp theo chờ chủ dự án quyết
**D-AI-02** (`docs/CHO-QUYET-DINH.md`): sửa tiếp bằng luật (khuyến nghị) và chấm trên bộ giữ lại mới, hay hạ ngưỡng, hay cho một lượt model
nhỏ. Từ nay H4 là bộ hồi quy; không chạy lại, không dùng làm bằng chứng cho bản sửa sau.

## Nợ S4 sau H4 (lệnh ca MVP ALL-IN mục 3d) — CHƯA lên main

Trên dữ liệu giả lập, chưa có người nghiệp vụ duyệt. Mã ở nhánh máy `lead/no-s4-sau-h4` (ba commit WIP), KHÔNG push: soát độc lập hai vòng
đều KHÔNG ĐẠT; vòng 3 là vòng cuối theo §9.6.

- Vòng 1 (cụm ghép vào `ignore_phrases` + kiểm miền cho mọi lượng): KHÔNG ĐẠT — S1: bỏ qua "nhà cung cấp" làm câu hỏi TIỀN bị trả số lít theo
  hoá đơn; bỏ qua "dự báo / cảnh báo / báo giá" làm câu bị trả theo chỉ số còn lại ("dự báo tiêu thụ …" ra số tiêu thụ thực đo); S2: tổng
  tháng / tuần viết rõ đơn vị (> 20.000 L) bị hỏi lại oan.
- Vòng 2 (rút 5 cụm mang nghĩa chưa hỗ trợ; kiểm miền chỉ cho số không tự viết đơn vị): S1-1, S1-2, S2-1 đóng (trở về hành vi cũ); KHÔNG ĐẠT có
  điều kiện vì "ghi nhận", "nhận xét" cũng mang nghĩa chưa hỗ trợ (lệnh ghi, đánh giá): "Ghi nhận EX-03 tiêu thụ 3782 L dầu hôm nay" bị trả như
  truy vấn, đọc lại đúng số người dùng đưa — dễ hiểu nhầm là đã ghi. Đổi lại 37 câu "… ghi nhận tiêu …" thôi trả nhầm số kho xuất.
- Đã đo ở vòng 2: test có PG 602 pass (theo tên 0 đổi / 0 mất); 117 / H1 / H3 / H4 hai đường 0 dòng khác; corpus 716.133 câu: 0 phán quyết
  mới / 0 đổi chiều; F6' (tiền nói tắt "4 củ", "dầu 4tr", "4,1 triệu dầu", "dầu 150k") đóng; số trần của cả tháng ("150000 dầu") bị hỏi lại
  (đánh đổi phía an toàn).
- **Bài học:** danh sách đóng các cụm ghép không đủ — mỗi cụm có thể mang một nghĩa hệ chưa hỗ trợ (tiền, dự báo, báo giá, cảnh báo, lệnh
  ghi, đánh giá); bỏ qua cụm thì câu bị trả theo chỉ số còn lại. Cần cơ chế "nghĩa chưa hỗ trợ ⇒ hỏi lại / từ chối" và quyết định của chủ dự
  án về nghĩa của câu "ghi nhận + giá trị" (câu mở **D-OP-18**). Nợ còn mở: S4R-03 cho các cụm đã rút, F6' ở khung "bao nhiêu" (nhánh từ khoá
  yếu, D-OP-16) và ký hiệu "$", khoảng trắng kép làm cụm hở, chiều tiền ("thanh toán" là từ khoá của dầu theo hoá đơn).

## Vòng sửa S4 (23/10) — D-AI-02 = a, D-OP-18 = c, D-OP-11 = b (nhánh `m3/s4-vong-sua-2310`, CHƯA gộp main)

Trên dữ liệu giả lập, chưa có người nghiệp vụ duyệt. Chủ dự án trả lời 23/10: **D-AI-02 = a** (sửa tiếp bằng luật, không model), **D-OP-18 = c**
(câu "ghi nhận / nhận xét + giá trị cụ thể" là lệnh ghi / đánh giá ⇒ hỏi lại; không giá trị ⇒ câu hỏi), **D-OP-11 = b** (có bí danh mơ hồ thì chỉ
liệt kê ứng viên của bí danh). Mỗi cụm một commit, mỗi commit có test đỏ trước và dòng đột biến riêng ở `scripts/dot-bien/n45-y-dinh-s4.sh`; câu thử là
câu viết mới theo cặp tối thiểu, không chép từ H3 / H4. H4 là bộ HỒI QUY ở vòng này (đã chạy một lần): số H4 dưới đây không phải bằng chứng cho bản sửa.

| Cụm | Luật (bất biến, không danh sách theo câu) | Chỗ trong mã |
|---|---|---|
| D-OP-11 = b | Tập ứng viên hỏi lại = ứng viên của bí danh mơ hồ; thực thể đã giải chỉ góp vào việc đếm chủ thể; hai thực thể đều đã giải giữ cách cũ | `clashing` |
| N1 | Hoà ý định / thiếu thuật ngữ mà câu còn bí danh mơ hồ thì mang ứng viên của bí danh; mẫu `clarify_intent` kể ứng viên | `Classify`, `templates.yaml` |
| C5 | Mệnh đề hỏi (cụm xác nhận hoặc cụm hỏi lượng) có chỉ số riêng thì mệnh đề ĐỨNG TRƯỚC nêu một con số là ngữ cảnh: chữ chỉ nguồn của con số ("phiếu kho ghi 3.850") không quyết định ý định | `premiseRanges` |
| C4 | Hai chỉ số khác nhau + dấu hiệu so sánh ("chênh", cụm so sánh) ⇒ so ĐÚNG cặp người hỏi nêu, thứ tự nhắc; "chênh lệch" chỉ trả cặp mặc định khi mọi chỉ số nêu nằm trong cặp ấy, còn lại hỏi lại | `comparison`, `Options.DerivedOperands` |
| C3 | Tín hiệu phạm vi = thực thể + (thuật ngữ, hoặc — khi thuật ngữ mặc định áp được — lượng cùng chiều đo + chữ của từ khoá chỉ số); chữ của cụm xác nhận và mã chứng từ trung tính với a2; lời chào là chữ "chào" | `scopeSignal`, `generic`, `intents.yaml` |
| F6' | Số không tự viết đơn vị mà ngoài miền một lần đo của chỉ số không phải giá trị của nó (tiền nói tắt) — phần ĐÚNG lấy lại của nhánh nợ cũ | `Options.ValueInRange` |
| A2 | Cơ chế chung "nghĩa chưa hỗ trợ ⇒ hỏi lại": pack khai `unsupported_meanings` (tiền, dự báo, báo giá, cảnh báo, lệnh ghi, đánh giá); cụm chiếm chỗ trong luật khớp dài nhất; nghĩa `needs_value` chỉ tính khi câu nêu một con số và không phải câu hỏi | `unsupportedIn`, router, `ask.go`, pack |

**Khác chữ ADR (có lý do, đã đo):**
- Dấu "?" một mình KHÔNG đánh dấu mệnh đề hỏi ở (b): bản thử đầu làm WIN-I-086 ("…được cấp 4k dầu, anh cứ xác nhận là đúng giúp em để em báo cáo gấp nhé?",
  bộ 117 đỏ 116 / 117) hoà ý định vì mệnh đề cuối chỉ nhờ xác nhận. Chỉ cụm xác nhận và cụm hỏi lượng của pack đánh dấu.
- (b) chỉ che mệnh đề trước khi mệnh đề hỏi tự nêu chỉ số (từ khoá riêng, hoặc con số riêng cùng một chữ của từ khoá): mệnh đề hỏi chỉ là cái đuôi
  ("…, tại bể 7 đúng không?") thì mệnh đề trước chính là câu hỏi — bản đầu làm 5 test cổng phán quyết đỏ.
- Nhánh nợ cũ `lead/no-s4-sau-h4`: lấy lại F6' (`ValueInRange` + test). KHÔNG lấy phần `ignore_phrases` ("ghi nhận", "nhận xét", "cấp bách") — thay bằng A2;
  "cấp bách" không được thay (S4R-03 phần chữ ghép không mang nghĩa chưa hỗ trợ còn nợ).

**Đo** (bộ nhớ và PG, đường nào cũng giống hệt từng ca; so với `a9ce6ed` = origin/main; 0 ca xanh → đỏ):
- bộ 117: 117 / 117 (0 dòng khác); H1: 92 → 94 / 104; H3: 113 → 121 / 134; H4: 104 → 110 / 119.
- 10 ca S4 của H4: 716, 740, 758, 810, 854, 862 từ đỏ thành xanh; 806, 841 từ hỏi lại / từ chối thành trả số kho xuất kèm câu giả định (còn đỏ chỉ vì a2 không
  phán quyết "Không."); 847, 848 từ trả theo hoá đơn không nhãn / từ chối thành hỏi lại nghĩa số liệu. Không ca nào của H4 còn sai nghĩa không nhãn ở S4 — chỉ đúng trong phạm vi H4 và bộ phát triển của điều phối: soát vòng 1 tìm thấy câu sai nghĩa không nhãn ngoài hai bộ đó (xem "Vòng 2" dưới, mục Nợ).
- Bộ phát triển (296 câu sinh theo bất biến: nghĩa chưa hỗ trợ 96, lệnh ghi / đánh giá 48, ngữ cảnh + hỏi 32, cặp so sánh 72, động từ chỉ số khác 48; do
  điều phối soạn, KHÔNG độc lập): câu trả lời sai nghĩa không nhãn 102 → 0 (64 + 18 + 0 + 20 + 0 trước; 0 ở mọi nhóm sau).
- Test: PG 651 pass / 0 fail / 0 skip (mốc 639); không PG 609 pass / 28 skip (mốc 597); so theo tên 0 mất, 0 đổi kết quả, 12 test mới. Test ghim đổi có chủ ý theo
  D-OP-11 = b: 4 hàng cùng một hợp đồng (2 ở `intelligence/intent`, 2 ở `tenants/construction`).
- Đột biến (chạy trên bản sao riêng, mỗi dòng "BỊ BẮT"): n43 187 / 187, n44 16 / 16, n45 90 / 90. Mười dòng n45 cũ gãy neo vì mã đổi (S1, S2, S12, S18, S26, S27, S29, S30,
  S76, S98) được viết lại neo, không xoá; S19 thành đột biến tương đương trên pack cầu đường (che theo từng chữ của cụm xác nhận) nên giữ dạng chú thích kèm lý do như S7.

**Giới hạn đã biết / ghi nợ (không làm ở vòng này):**
- Câu trong phạm vi không có chữ "dầu" và không một chữ nào của từ khoá chỉ số ("EX-03 xài 3782 L đúng không?") vẫn bị từ chối; "EX-03 nhận 20 tấn đúng không?"
  (từ khoá khớp, thuật ngữ mặc định bỏ qua lượng khác chiều đo) trả số kho xuất — có từ trước.
- a2 không phán quyết nên 806, 841 không thể xanh nếu H4 đòi "Không."; mở rộng phán quyết là việc của cổng (`claim.go` đã KHOÁ).
- 847, 848: chữ lạ ("cũng", "Con số đó") khiến a2 hỏi lại — phía an toàn; không thêm từ vào danh sách trung tính theo từng câu.
- Kiểm nạp pack của `unsupported_meanings` (thiếu mẫu câu, thiếu cụm) chưa có test riêng. Cụm báo nghĩa là danh sách theo nghĩa do pack khai: nghĩa / cụm mới
  (ví dụ "báo cáo giá") cần thêm vào pack; "đồng" không là cụm tiền vì trùng "đồng hồ".
- C4: sửa chỉ số "chênh" cho câu hai chỉ số; ba chỉ số trở lên và so sánh hai MÁY (D13 chỉ so hai chỉ số của một thực thể) vẫn hỏi lại / chưa có capability.
- Bộ giữ lại MỚI để chấm vòng này (D-AI-02 = a, "H6") chưa được soạn / chạy: vòng này chưa có phép chấm độc lập.

## Vòng 2 của vòng sửa S4 (23/10 chiều) — soát vòng 1 KHÔNG ĐẠT (S4AS-01 … S4AS-09); nhánh `m3/s4-vong-sua-2310`, CHƯA gộp main

Trên dữ liệu giả lập, chưa có người nghiệp vụ duyệt. Chủ dự án giữ nguyên D-AI-02 = a (luật các cụm C3 / C4 / C5 / N1, không model), D-OP-18 = c, D-OP-11 = b. Mỗi phát hiện một commit, test đỏ trước bằng câu
viết mới theo cặp tối thiểu (không chép từ H1 / H3 / H4), một dòng đột biến ở `scripts/dot-bien/n45-y-dinh-s4.sh`.

**Gộp main (S4AS-07).** Gộp `origin/main` `baaf834` (K1 lõi lệnh, K2 kho PG, vết R2-03, flock, 5d). Xung đột `intent.go` (6 vùng) và `router.go` (1 vùng) giải bằng tay, giữ cả hai phía: toàn bộ vết R2-03 (`Why`, `Quote`,
`WithholdQuoted`, dòng lý do, nguồn ứng viên, bậc kiểm của router) lẫn phần S4 (`Result.Unsupported`, ứng viên N1, `premiseRanges`, `comparison`, tín hiệu phạm vi). Mã S4 mới đều ghi lý do vào `Why` (mệnh đề ngữ cảnh bị che,
nghĩa chưa hỗ trợ và mệnh đề chứa cụm, ứng viên N1, so sánh, phương án bị loại). Sau gộp 5 test khoá nguyên dòng của vết đỏ vì hợp đồng S4 đổi có chủ ý (D-OP-11 = b: danh sách hỏi lại không còn thực thể rõ; C5: thêm dòng
mệnh đề ngữ cảnh; C4: dòng so sánh) — mã nay ghi vết đúng quyết định mới, test đổi theo, lý do ở commit `fix(S4AS-07)`.

| Mã | Luật (theo cấu trúc / bất biến) | Chỗ trong mã |
|---|---|---|
| S4AS-01 | Câu hỏi = câu có cụm xác nhận hay cụm hỏi lượng của pack; dấu "?" một mình KHÔNG làm câu thành câu hỏi. Lệnh ghi = cụm lệnh + con số có đơn vị, không cụm hỏi | `asking`, `unsupportedIn` |
| S4AS-02 | Nghĩa tiền theo cấu trúc: span kiểu tiền, "bao nhiêu" + đơn vị tiền của lexicon ở cuối mệnh đề (pack khai `signals`); nhường câu có lượng cùng chiều đo; các họ nghĩa còn lại mở danh sách THEO HỌ NGHĨA ở pack | `signalHit`, `askedUnit`, `intents.yaml` |
| S4AS-03 | C4: hai chỉ số khác nhau trong cùng mệnh đề + từ so sánh bậc hơn / kém (hay "chênh / lệch" — phạm vi cả câu) ⇒ so đúng cặp; ≥ 2 chỉ số mà không xác định được cặp ⇒ hoà ⇒ hỏi lại, BỎ luật số cụm khớp. C3: cụm xác nhận + thực thể + lượng cùng chiều đo là tín hiệu phạm vi đủ. C5: "mấy" là cụm hỏi lượng; mệnh đề ngữ cảnh nêu con số SAU câu hỏi cũng bị che | `comparison`, `comparisonScope`, `build`, `scopeSignal`, `premiseRanges` |
| S4AS-04 | Cụm báo nghĩa chưa hỗ trợ ở MỌI mệnh đề (kể cả ngữ cảnh) đều được nói ra; lệnh ghi đòi chính mệnh đề đó có con số và không là câu hỏi (kể cả đuôi hỏi "…, đúng không?"); ở mệnh đề ngữ cảnh cụm phải đứng đầu ("phiếu kho ghi nhận 3850 L, vậy …" là nguồn + động từ) | `unsupportedIn` |
| S4AS-05 | "triệu" trước đơn vị thể tích không phải tiền (bỏ cụm); "chào" chỉ là lời chào khi câu không còn thực thể / con số / thuật ngữ (`alone`); cụm nghĩa trong mệnh đề bị bỏ qua ("không quan tâm", "để sau") không hỏi lại (`dismiss_markers`) | `classify`, `unsupportedIn`, pack |
| S4AS-06 | Test nạp pack (7 ca lỗi) và test khử trùng ứng viên; S139–S143 SỐNG SÓT trước commit, BỊ BẮT sau | `pack_test.go`, `intent_test.go` |
| S4AS-08 | Mọi đường hỏi lại về ý định (hoà, thiếu thuật ngữ, nghĩa số liệu) lọc nhãn theo capability; không còn nhãn nào ⇒ từ chối ngoài phạm vi | `ask.go` |
| S4AS-09 | KHÔNG sửa (xem Nợ) | — |

**Đánh đổi có chủ ý (D-AI-02 = a).**
- Họ nghĩa dự báo, báo giá, cảnh báo, lệnh ghi, đánh giá vẫn là DANH SÁCH cụm, nay mở theo họ nghĩa ở `intents.yaml` (không ở mã Go, không theo từng câu của bộ ca nào). Đây là đánh đổi, không phải "đóng theo bất biến": độ phủ do bộ giữ lại
  độc lập (H6) đo. Danh sách KHÔNG được dùng để bỏ qua chữ (`ignore_phrases`) như nhánh nợ cũ — cụm báo nghĩa chỉ chiếm chỗ để chữ con của chúng không thành từ khoá, rồi câu được hỏi lại.
- Chỉ nghĩa tiền có tín hiệu cấu trúc (không cần chữ nào trong danh sách): span tiền và "bao nhiêu" + đơn vị tiền.
- Cụm chung "chào" không còn là danh sách cụm nhưng lời chào "thuần" được nhận theo cấu trúc (`alone`).

**Khác chữ lệnh / ADR (có lý do, đã đo).**
- Nhận dạng lệnh theo "cụm lệnh + con số có đơn vị, không cụm hỏi" chứ không theo "động từ lệnh đứng ĐẦU mệnh đề": đòi đứng đầu bỏ sót "Anh cập nhật …" và đánh trúng sai "ERP ghi nhận …"; câu hỏi (cụm xác nhận / hỏi lượng) mới là
  thứ tách "ERP ghi nhận 3900 L, đúng không?" khỏi lệnh. Vị trí đầu mệnh đề chỉ dùng cho cụm ở MỆNH ĐỀ NGỮ CẢNH (≤ 1 chữ trước cụm).
- Bỏ luật số cụm khiến WIN-I-086 (bộ 117) xanh → đỏ: ca từng đạt nhờ "xác nhận" khớp chữ con "nhận" (S4R-03) nên kho xuất 2 cụm thắng đội báo 1 cụm; hai nghĩa ngang nhau thì hỏi lại. Ca bị BỎ khỏi `win/passing.txt` kèm lý do,
  tệp ca không sửa. WIN-I-038 và H3 509 ("Cảm biến báo …") cùng hệ quả được giữ đạt bằng cụm "cảm biến báo" của nguồn cảm biến (cụm dài thắng "báo") — cụm của nguồn, không phải thêm theo ca vì hai ca cùng một cấu trúc nguồn + động từ.
- Thử và BỎ hai tín hiệu tiền: thuật ngữ "giá" của lexicon (khớp cả "giá trị": "báo cáo giá trị" bị hỏi về tiền) và số nói tắt ngoài miền một lần đo ("1tr dầu", "4 củ": đỏ WIN-I-010, ca ghim trả số có thẩm quyền).
- C3 đánh đổi: "EX-03 chứa được 300 lít nước đúng không?" (lượng thể tích của chất khác) nay HỎI LẠI nghĩa số liệu thay vì từ chối; hai test vòng 1 ghim "xài 3782 L" và "300 lít nước" là giới hạn đã biết nên đổi (D-AI-02 = a đòi C3 theo cụm xác nhận).

**Đo** (dữ liệu giả lập; mọi số dưới là lệnh đã chạy):
- Bộ phát triển của tác giả (325 câu viết tay theo họ nghĩa, KHÔNG độc lập, không chép từ bộ nào), câu trả lời SAI NGHĨA KHÔNG NHÃN (trả số / chỉ số một nghĩa khác, không có câu giả định "Hiểu là …"), nền `origin/main` → vòng 1 `c7afacd` → nhánh này:
  tổng 117 → 106 → 29, hỏi thừa 45 → 44 → 5. Theo nhóm: cụm trong danh sách vòng 1 (44 câu) 22 → 3 → 0; đồng nghĩa NGOÀI danh sách vòng 1 (82 câu, họ nghĩa mở sau khi đọc chính các câu này nên chỉ chứng minh luật mở danh sách chạy, KHÔNG chứng minh độ phủ) 49 → 47 → 7;
  **đồng nghĩa CHƯA THẤY** (37 câu viết SAU khi đóng danh sách, cố ý ngoài danh sách: "tiên lượng", "ghi vô sổ", "chốt số", "tạo bản ghi", "cho xin bảng giá", "có vấn đề gì không", "ngày mai … bao nhiêu"…) 22 → 21 → 22 — luật theo họ KHÔNG phủ đồng nghĩa chưa thấy
  (trừ nghĩa tiền theo cấu trúc); lệnh ghi lịch sự có "?" (14) 7 → 11 → 0; cụm nghĩa ở mệnh đề ngữ cảnh (10) 9 → 10 → 0; so sánh có dấu hiệu (18) 3 → 3 → 0 (hỏi thừa 6 → 6 → 0); hai chỉ số không dấu hiệu (16) 5 → 5 → 0; trong phạm vi không chữ "dầu" (18) từ chối oan 18 → 17 → 0 (hỏi lại);
  hỏi lượng / ngữ cảnh trước-sau (18) hỏi thừa 10 → 5 → 0.
- Còn sai trong bộ phát triển (29): 22 đồng nghĩa chưa thấy + 7 câu họ nghĩa ("đốt dầu hết 4tr / 4 củ / 5 tỷ" — số nói tắt không đơn vị, WIN-I-010 ghim; "mất bao nhiêu nghìn"; "sắp dùng"; "tính trước"; "có ổn không").
- Bộ hồi quy so với mốc `origin/main` (lead-k2g), bộ nhớ và PG giống hệt từng khối: bộ 117 117/117 → 116/117 (WIN-I-086, trái hợp đồng, ghi trên); H1 92 → 97/104 (268, 318, 342, 419, 429 đỏ → xanh); H3 113 → 123/134 (538, 539, 543, 557, 603, 611, 612, 617, 660, 663 đỏ → xanh; 509 đạt → đạt nay có
  "Đúng."; 565, 607, 646 đỏ → đỏ từ chối → hỏi lại); H4 104 → 110/119 (716, 740, 758, 810, 854, 862 đỏ → xanh; 806, 841 hỏi lại / từ chối → trả số kèm câu giả định; 847, 848 → hỏi lại). 0 ca xanh → đỏ ngoài WIN-I-086.
- Cổng: `go vet ./...` 0 dòng; `gofmt -l` trên tệp git theo dõi 0; `go test -count=1 ./...` có PG (cơ sở dữ liệu riêng) 1077 pass / 0 fail / 0 skip (mốc 1054; so theo tên 0 mất, 0 đổi kết quả, 23 mới), không PG 975 pass / 0 fail / 72 skip (PG-only);
  `scripts/kiem-skip-test.py` XANH khi có PG.
- Đột biến (chạy trên bản sao riêng): n43 187 / 187, n44 16 / 16, n45 123 / 123 (122 dòng chạy trọn bộ ở mã cuối + S107, S116 viết lại cho biên dịch được và chạy riêng; lần chạy đầu cho 115 bị bắt, 2 sống, 3 không biên dịch, 4 neo gãy — đã sửa, xem commit `test(S4): n45 vòng 2`); 0 sống, 0 lỗi khôi phục. Đột biến mới S139–S143 (kiểm nạp pack, khử trùng ứng viên) SỐNG SÓT khi chạy ở commit trước test S4AS-06 (`b5aa573`) và BỊ BẮT sau — đó là bằng chứng "test đỏ trước" của S4AS-06.

**Nợ còn mở (không sửa ở vòng này).**
- S4AS-09 (có từ trước, S4R-03): từ khoá khớp chữ con của từ ghép khác nghĩa vẫn trả số một chỉ số KHÔNG nhãn ("đã được cấp phép hoạt động chưa?" → số kho xuất; "Báo cáo tình trạng máy" → số đội báo; "xác nhận" → "nhận"). Vì vậy câu "không ca nào còn sai nghĩa không nhãn"
  chỉ đúng trong H4 và các bộ phát triển; H6 độc lập chưa chạy. Hướng: từ khoá đòi thuật ngữ + hỏi lượng, hay khớp theo từ nguyên, không theo chữ con.
- Đồng nghĩa chưa thấy của các họ dự báo / báo giá / cảnh báo / lệnh ghi / đánh giá (22 / 37 trong bộ của tác giả) vẫn trả số. Hướng đã cân nhắc, CHƯA làm vì đụng ratchet: câu có giá trị mà không phải câu hỏi (khẳng định / lệnh) coi là lệnh — cần đo tác động lên bộ 117 / H1 / H3 / H4 trước.
- Số tiền nói tắt không đơn vị ("1tr dầu", "4 củ", "5 tỷ"): bộ chuẩn hoá cho đơn vị suy từ thuật ngữ nên không phân biệt được với lít; WIN-I-010 ghim trả số có thẩm quyền. "bao nhiêu đồng một lít" không dính tín hiệu (mệnh đề phải kết thúc sau đơn vị).
- "ERP ghi nhận … 3900 L, đúng không?" (nguồn lạ + động từ) hỏi lại nghĩa số liệu thay vì trả số kho xuất + xung đột như nền: tên nguồn không thuộc tập chữ giải thích được của a2; cần pack nối tên nguồn với chỉ số.
- Nghĩa tiền / lệnh ở mệnh đề ngữ cảnh: lệnh nằm ở mệnh đề có con số nhưng cụm ở mệnh đề liền kề không bắt được ("Ghi nhận cho EX-03, dùng 3782 L").
- "Giá nhân công …" và "nặng bao nhiêu tấn" (tên máy + từ hỏi lượng) vẫn được trả theo ý định chung kèm câu giả định ("Máy xúc EX-03 nặng bao nhiêu tấn?" → số dầu kho xuất): từ khoá yếu "bao nhiêu" — D-OP-16, có từ trước.
- Bộ giữ lại mới (H6) để chấm vòng sửa này chưa soạn / chưa chạy: vòng này chưa có phép chấm độc lập; số đo trên là bộ phát triển của tác giả (một người soạn, oracle tự đặt).

## Vòng 3 của vòng sửa S4 (23/10 tối) — VÒNG CUỐI theo §9.6; soát vòng 2 KHÔNG ĐẠT (S4BS-01 … S4BS-10); nhánh `m3/s4-vong-sua-2310`, CHƯA gộp main

Trên dữ liệu giả lập, chưa có người nghiệp vụ duyệt. Chủ dự án 23/10: **D-AI-02 = a** (luật, không model), **D-OP-18 = c**, **D-OP-11 = b**; **D-AI-04** (bốn họ nghĩa đánh giá / cảnh báo / dự báo / báo giá nhận
bằng danh sách cụm ở pack) CHỜ chủ dự án (`docs/CHO-QUYET-DINH.md` mục 4) — vòng này KHÔNG tuyên bố "đóng" cho bốn họ ấy. Mỗi phát hiện một commit, test đỏ trước bằng câu viết mới theo cặp tối thiểu (không chép từ
H1 / H3 / H4), dòng đột biến ở `scripts/dot-bien/n45-y-dinh-s4.sh`. Hết vòng này, phát hiện còn lại là NỢ có mã (§9.6 / §9.10 mục 2), không có vòng 4.

**Gộp main (S4BS-09).** Gộp `origin/main` `59ac0d7` (D-OP-9 (a') sửa `data/normalization`, `data/ingestion`, `core/pipeline/ask.go`; D-AI-04 ở CHO): tự gộp sạch, không xung đột; cổng chạy lại trên cây đã gộp.

| Mã | Luật (theo cấu trúc, từ nối ngữ pháp ở pack — mã Go không có chuỗi nghiệp vụ) | Chỗ trong mã |
|---|---|---|
| S4BS-01 | Đoạn MỤC ĐÍCH (từ từ nối `purpose_markers` "để", "nhằm" tới hết mệnh đề) không góp từ khoá chỉ số của ý định SỐ LIỆU và không kích hoạt cụm báo nghĩa — trừ khi chính đoạn đó mang cụm hỏi ("để biết đội báo bao nhiêu dầu?" là câu hỏi). Ý định quy trình vẫn đếm trên câu chưa che ("dùng để làm gì theo quy trình?" là RAG). "xác nhận" (nhờ xác nhận) vào `ignore_phrases`: "nhận" của nó là chữ con của từ ghép. WIN-I-086 trả về `passing.txt` | `backgroundRanges`, `hitsBG`, `classify` |
| S4BS-03 | (i) lệnh ghi / đánh giá đòi con số ở chính mệnh đề của cụm HOẶC ở mệnh đề kề không phải câu hỏi; (ii) cụm hỏi lượng khớp theo ranh giới từ của câu gốc (`asking` thay cụm báo nghĩa bằng dấu cách trước khi tìm cụm hỏi: "định mức" không nằm trong "nhận định mức"); (iii) yêu cầu KÈM THÊM sau câu hỏi ("…, kèm nhận xét luôn?") không bị bỏ lặng — trừ khi mệnh đề nêu tên nguồn ("theo ghi nhận của cảm biến") | `unsupportedIn`, `asking`, `withoutCues`, `attachedRequest` |
| D-OP-18 = c | Câu NÊU giá trị = có thực thể + lượng có đơn vị cùng chiều đo với một chỉ số (không tính tên thực thể) + KHÔNG có dấu hiệu hỏi (cụm hỏi của pack, dấu "?", từ phủ định cuối câu `negation_markers`) ⇒ "Trợ lý chưa ghi số liệu", bất kể động từ. Tín hiệu `stated_value` do pack khai cho nghĩa lệnh ghi; chỉ là phương án cuối khi chưa nghĩa khác được báo | `statedValue`, `questioned`, `SignalStatedValue` |
| S4BS-04 | "bao nhiêu" + (từ chỉ hàng) + (đơn vị tiền) Ở BẤT KỲ ĐÂU trong mệnh đề; chữ liền sau phải là hết mệnh đề / từ trung tính / số chữ / cụm hỏi (loại "đồng hồ", "đồng bộ", "đô thị", "tỷ lệ", "triệu lít"); asked_unit không nhường câu có lượng lít (span:money vẫn nhường). Pack: USD ở chiều đo riêng `currency_usd`; "củ", "tr" là đơn vị tiền khi hỏi | `askedUnit`, `moneyPhraseEnds`, lexicon, `Engine.MultiplierAliases` |
| S4BS-05 | Cụm báo nghĩa không kích hoạt ở mệnh đề phủ định (từ phủ định không đứng cuối), hoãn / đã xong (`dismiss_markers`), điều kiện (`condition_markers` ở đầu mệnh đề), đoạn mục đích. Cụm quá chung (`weak_cues`: sẽ, dự kiến, dự tính, ước tính) chỉ kích hoạt ở chính mệnh đề hỏi một chỉ số | `unsupportedIn`, `negated`, `conditional` |
| S4BS-06 | `source_verbs: [báo]`: lúc nạp sinh cụm "<tên nguồn> báo" cho ý định của chỉ số mà nguồn cung cấp (tên nguồn ở `metrics.yaml`, MỌI nguồn); gỡ cụm khai tay "cảm biến báo" | `SetOptions`, `hitDefs`, pack |

**Khác chữ lệnh của chủ dự án (có lý do, đã đo).**
- "Con số có đơn vị" của D-OP-18 = c là lượng cùng chiều đo với một chỉ số (lít, khối) và không nằm trong tên thực thể. Bản đầu tính mọi lượng có đơn vị làm H4 WIN-I-715 ("Họp giao ban lúc 7h30, anh cho em số lít dầu cho máy 03 được cấp hôm qua với.") xanh → đỏ: "7h30" là
  giờ họp, "03" của "máy 03" đứng cạnh "dầu" bị bộ chuẩn hoá gắn đơn vị lít. Sửa trước commit.
- Dấu "?" là dấu hiệu hỏi của luật cấu trúc D-OP-18 (câu nêu giá trị kèm "?" đi đường câu xác nhận sẵn có, đúng chữ lệnh) NHƯNG vòng này KHÔNG khôi phục "?" làm dấu mệnh đề hỏi ở bất biến (b) (`premiseRanges`): chưa làm, chưa đo (nợ C5 bên dưới).
  Lệnh lịch sự có "?" mà động từ trong danh sách cụm ("Ghi nhận … nhé?") vẫn là lệnh (S4AS-01 giữ nguyên); động từ NGOÀI danh sách + "?" ("Nhập … nhé?") đi đường câu xác nhận và trả số — đánh đổi đúng chữ lệnh.
- Bản đầu của S4BS-01 che cả từ khoá quy trình trong đoạn mục đích làm H1 WIN-I-319 xanh → đỏ; sửa bằng `hitsBG` (đoạn mục đích chỉ che ý định số liệu) trước commit. Cả hai chỗ này bắt được nhờ chạy winreport TRƯỚC khi commit.

**Đo** (dữ liệu giả lập; mọi số dưới là lệnh đã chạy; `go vet ./...` 0 dòng, `gofmt -l` trên tệp git theo dõi 0):
- Test ở mã cuối `80012b7`: không PG 992 pass / 0 fail / 72 skip (PG-only); có PG (cơ sở dữ liệu riêng, đã DROP) 1094 pass / 0 fail / 0 skip; so theo tên với mốc main (1062): 0 mất, 0 đổi kết quả, 32 mới (23 của vòng 2 + 9 của vòng 3);
  `scripts/kiem-skip-test.py` XANH. Cả cổng (vet, gofmt, test hai kiểu, so tên, kiem-skip, winreport 7 lần) chạy 35 giây (13:35:43 → 13:36:18); số agent chạy song song lúc phóng: 5, chưa đo lại lúc chạy (box có tiến trình go của agent khác, load average 2–9); không test chập chờn.
- Winreport so với mốc main (bộ nhớ trùng PG từng khối ở H1, H3, H4): bộ 117 117/117, **0 dòng khác main** (WIN-I-086 đỏ → xanh so với đầu vòng 3); H1 96/104 (main 91), H3 122/134 (112), H4 109/119 (103). **0 ca xanh → đỏ ở cả bốn bộ.**
  So với đầu vòng 3 (`637d5f7`) chỉ đổi hai ca: WIN-I-086 đỏ → xanh; H1 WIN-I-263 đỏ → đỏ (từ trả sai nghĩa im lặng "Dầu đội báo của DZ-05 … chưa có dữ liệu" thành hỏi lại nghĩa số liệu). H3 và H4 không đổi.
- Bộ phát triển của tác giả (232 câu, viết TRƯỚC lần chạy đầu — bộ câu sinh bằng một script ngoài repo, mốc giờ tệp sinh đứng trước mọi lần chạy; theo họ nghĩa của soát vòng 2; KHÔNG độc lập, oracle tự đặt; nhóm câu có họ nghĩa viết khi đã biết phát hiện nên số cải thiện một phần do cách dựng):
  đúng / sai nghĩa (không nhãn + có nhãn) / hỏi lại hoặc từ chối thừa — nền main 128 / 61 / 39; đầu vòng 3 133 / 57 / 38; cuối 224 / 1 / 3. Theo nhóm (cuối; đầu vòng 3 trong ngoặc):
  mục đích / nhờ xác nhận 12 / 13 (3); lệnh có con số ở mệnh đề kề 7 / 7 (0); chữ ghép "nhận định mức" 3 / 4 (0); yêu cầu kèm thêm 4 / 4 (0); lệnh ghi bất kể động từ (24 động từ ngoài danh sách + 6 câu nêu giá trị) 30 / 30 (0); tiền theo cấu trúc 19 / 19 (4), ca âm tiền 11 / 11;
  chữ báo nghĩa ngẫu nhiên ("sẽ", tiền / chi phí bị phủ định / hoãn) 18 / 19 (2); nguồn + "báo" 7 / 7 (3); câu hỏi THƯỜNG (82 câu, đo hỏi thừa) 81 / 82 (80; nền main 78) — câu còn lại là câu hai mốc thời gian ("… so với hôm qua"), hỏi lại đúng, oracle của tác giả sai.
- Bộ 325 câu của TÁC GIẢ VÒNG 2 (không phải bộ của vòng này, không độc lập), chạy lại ở mã vòng 3 như một phép đối chứng hồi quy: câu trả lời sai nghĩa (không nhãn + có nhãn) nền main 117 → đầu vòng 3 29 → cuối 18; hỏi thừa 45 → 5 → 5. Phép đo này bắt được MỘT hồi quy của chính
  vòng 3 ("Lưu lại giúp em EX-03 hôm nay tiêu thụ 3782 L dầu được không ạ?": luật phủ định coi "không" trước từ trung tính "ạ" là phủ định, trả số tiêu thụ không nhãn) — đã sửa ở commit riêng (`contentEnd`: bỏ dấu câu và từ trung tính ở cuối trước khi xét từ phủ định), test đỏ trước.
- Fuzz 3508 câu (token-soup từ khoá / dấu / số, NFD, chữ hoa, không dấu, ký tự vô hình, emoji, câu 1500 từ, 500 dấu "?", 300 chữ "không"): 0 panic; lỗi chỉ là đầu vào rỗng (46), quá 2000 ký tự (1) và route tri thức cần chỉ mục tài liệu (328; khung thử không có chỉ mục, nền 321).
- Đột biến (bản sao riêng, mỗi dòng "BỊ BẮT"): n43 187 / 187 (mã `18a5479`; các commit sau đó chỉ đổi `contentEnd` ở `intent.go` + test / n45, không chạm cổng phán quyết hay chuẩn hoá số), n44 16 / 16 (cả `18a5479` và `38d915b`), n45 162 / 162 (159 ở `38d915b` + 3 dòng viết lại ở `80012b7`;
  0 sống sót, 0 không biên dịch, 0 lỗi khôi phục). Lần chạy n45 đầu ở `18a5479` (163 dòng) cho 137 bắt, 15 neo gãy, 7 không biên dịch, 4 sống sót (S82, S122, S169, S179): đã xử lý ở hai commit `test(S4)`; S115 và S132 không còn mã tương ứng (thay bởi S165–S167, S183 / S186), S169 tương đương trên pack cầu đường (ghi lý do trong script).
  Đột biến mới của vòng này: S147–S188. Chưa có đột biến nào cho `premiseRanges` dùng `asking` thay cụm hỏi trực tiếp ngoài S82 viết lại.

**Đánh đổi có chủ ý.**
- Dấu "?" làm câu nêu giá trị thành câu xác nhận ⇒ lệnh "Nhập EX-03 hôm nay dùng 3782 L dầu nhé?" (động từ ngoài danh sách) được TRẢ SỐ tiêu thụ (không nhãn). Đây là hệ quả đúng chữ của D-OP-18 = c; chỉ danh sách cụm mới bắt được nó.
- Mệnh đề điều kiện được nhận theo VỊ TRÍ (mệnh đề bắt đầu bằng "nếu / giả sử / hễ"): mọi cụm báo nghĩa trong mệnh đề ấy, kể cả phần "thì …", không kích hoạt ("Nếu mưa to thì EX-03 dự báo dùng bao nhiêu dầu?" bị trả số hôm nay).
- Mệnh đề đoạn mục đích mang cụm xác nhận cuối câu ("… 4k dầu để em báo cáo đúng không?") coi là câu hỏi nên "báo cáo" vẫn tính ⇒ hoà ⇒ hỏi lại (an toàn).
- Mệnh đề ngữ cảnh nêu cụm tiền mà không phủ định / hoãn ("Kế toán đang quyết toán tháng, …") vẫn bị hỏi lại (an toàn, nặng).
- USD ở chiều đo riêng (không tỷ giá) nên không quy đổi sang đồng; "k" không là đơn vị tiền ("4k lít").

**Nợ còn mở (không sửa ở vòng này, vòng cuối — có TEST ĐỎ chặn khi làm tiếp ở n45 / test của vòng 3).**
- **S4BS-02 chờ D-AI-04**: họ dự báo / báo giá / cảnh báo / đánh giá vẫn nhận bằng danh sách cụm của pack. Đã đo lại, chưa phủ đồng nghĩa chưa thấy: "Tiên lượng hôm nay EX-03 dùng bao nhiêu dầu?" → số tiêu thụ; "EX-03 hôm nay dùng 3782 L dầu có ổn không?" → số tiêu thụ; "Gọi tôi khi EX-03 dùng quá 4000 L dầu nhé?" → số tiêu thụ
  (không "?" thì bị luật nêu-giá-trị bắt, sai nhãn "chưa ghi số liệu" nhưng không im lặng). Phần cấu trúc đã đóng: lệnh ghi bất kể động từ khi không "?", tiền, mệnh đề kề, yêu cầu kèm thêm. Độ phủ do H6 độc lập đo; vòng này không có phép chấm độc lập.
- **S4BS-08**: B5 / B6 (so sánh hai chỉ số: `len(named) == 2 && len(distinct) == 2`, `len(distinct) < 3`) tương đương trên pack cầu đường (mỗi ý định một chỉ số riêng); thiếu test pack tổng hợp có hai ý định cùng chỉ số.
- **S4BS-10 = S4AS-09**: chữ con của từ ghép khác nghĩa ngoài "xác nhận" (S4BS-01), "<tên nguồn> báo" (S4BS-06), "nhận định mức" (S4BS-03): "đã được cấp phép hoạt động chưa?", "Báo cáo tình trạng máy", "tiêu thụ bao nhiêu xăng?" (vật tư khác dầu) vẫn trả số một chỉ số không nhãn; số nói tắt không đơn vị ("4 củ", "5 tỷ") còn nguyên (WIN-I-010 ghim).
- **C5 / bất biến (b)**: "?" chưa được khôi phục làm dấu mệnh đề hỏi ở `premiseRanges`: ngữ cảnh + hỏi bằng "thế nào / ra sao / là gì" (không cụm hỏi của pack) vẫn bị hỏi thừa. Chưa đo ở vòng này.
- **S4BS-05 (iii) phần từ khoá**: từ khoá của chỉ số bị phủ định / phụ ("… không phải số kho xuất, …", "đội trưởng chưa báo") vẫn hoà ⇒ hỏi lại (an toàn); vòng này chỉ xử lý cụm báo nghĩa và đoạn mục đích.
- **S4BS-06 giới hạn**: bí danh "hóa đơn" (dấu khác "hoá đơn") chưa là bí danh nguồn ở `metrics.yaml` nên "Hóa đơn báo …" vẫn hoà; "Báo em số dầu cảm biến đo được của …" ("báo" là mệnh lệnh) hỏi lại. "Nhận xét định mức tiêu thụ 3782 L …" ("định mức" là danh từ, cụm hỏi lượng của pack) trả số tiêu thụ không nhãn.
- Câu "không ca nào còn sai nghĩa không nhãn ở S4" chỉ đúng trong phạm vi đã đo: H4 và bộ phát triển 232 câu của tác giả (còn 1 câu sai nghĩa không nhãn: "Nhận xét định mức tiêu thụ …"); nhóm chưa thấy ngoài bộ (tiên lượng, "có ổn không", xăng, "4 củ") còn sai. H6 độc lập chưa chạy.

## Ghi chú trạng thái 24/10/2026 (thêm sau, không sửa lời trên)

Dòng "Trạng thái: ĐỀ XUẤT, chưa áp mã" ở đầu tệp đã lỗi thời. Mã vòng sửa S4 theo ADR này **đã ở main**: gộp `e6e335c` (24/10, D-RAG-05)
ở trạng thái KHOÁ sau vòng 3, kèm nợ S4CS-01…04 — tức hồi quy S4CS-02 đã biết cũng vào main (F-001 của báo cáo audit
`docs/bao-cao/2026-10-24-vnext-audit/BAO-CAO.md`, tái hiện 7/7). Sửa hẹp S4CS-02 (D-AI-05) đang làm ở lát L2 của đợt vNext. H6 (bộ chấm
độc lập của vòng này) **chưa chạy**: BLOCKED vì tệp ca H6 không có trên máy thi công.
