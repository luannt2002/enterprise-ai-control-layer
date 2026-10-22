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
