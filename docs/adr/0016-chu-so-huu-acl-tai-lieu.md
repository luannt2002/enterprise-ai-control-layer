# ADR-0016 — Chủ sở hữu ACL tài liệu: lõi cấp bộ lọc, RAG chỉ áp

- **Ngày:** 24/10/2026.
- **Trạng thái:** **CHẤP NHẬN 24/10/2026** — GO của chủ dự án cho báo cáo audit vNext (`docs/CHO-QUYET-DINH.md` mục 0, dòng 24/10
  "vNext"): chấp nhận theo GO chung 24/10 cho toàn bộ khuyến nghị của báo cáo; chủ dự án chưa trả lời riêng Q2 / Q3 / Q5. Mã chưa đổi: phần thi công thuộc lát L4 (`core/policy`) và L6 (ranh giới RAG) của báo cáo §14.
- **Giải:** mâu thuẫn D12 của báo cáo (§17: hai nơi giữ hai nửa quyền đọc tài liệu); nguồn gốc của D-RAG-02 (12 vai của ACL kho không có
  trong policy). Lựa chọn cấu hình cụ thể cho pack construction (D-RAG-02 a / b / c) vẫn là câu của chủ dự án.
- **Liên quan:** ADR-0013, ADR-0015 (lõi / module); `CLAUDE.md` §12 (RAG không sở hữu "tenant authorization", "enterprise permission"); báo cáo audit
  `docs/bao-cao/2026-10-24-vnext-audit/BAO-CAO.md` §04.2, §05, §17 (D12, D15); F-006 (HYBRID thiếu quyền tài liệu, câu Q7 còn mở).

## Bối cảnh (đọc mã ở mốc `3f1bf7e`)

Quyền đọc tài liệu hôm nay bị chia làm hai nửa, ở hai nơi, với hai bộ tên vai:

| Nửa | Ở đâu | Tên vai lấy từ |
|---|---|---|
| capability `documents.read` của route tri thức / HYBRID | S5: `Route` gọi `Policy.Allows` (`intelligence/router/router.go`) — đủ khi **bất kỳ** vai nào của người hỏi có capability | `policy.yaml` của pack (construction: 5 vai) |
| ACL theo vai + phạm vi của từng tài liệu | `MemoryIndex.visible` / `allowed` trong `intelligence/rag/rag.go`, lọc trong Go **trước khi chấm**; đường PG nạp mọi tài liệu của tenant rồi lọc cùng chỗ (comment ở `0001_win.sql` nói lọc trong SQL — D15) | `acl.json` / phần đầu tài liệu (kho S7: 16 vai) |

Hệ quả đã đo / đọc thấy:
- **D-RAG-02 [ĐO, `evals/rag/s4_test.go`]:** 12 vai có trong ACL kho nhưng không có trong policy ⇒ mọi câu tri thức của họ bị S5 chặn dù
  S4 đúng; lệch này không hiện ra ở lúc nạp, chỉ lộ khi đo.
- **[SUY, đọc mã, chưa có test]:** capability và ACL được thoả bởi hai vai KHÁC nhau của cùng người hỏi — vai A có `documents.read` nhưng
  không có trong ACL, vai B có trong ACL nhưng không có `documents.read` — thì S5 cho qua (nhờ A) và `visible` cho đọc (nhờ B). Luật "cùng
  một vai" mới áp cho cặp ACL ↔ phạm vi, chưa áp cho cặp capability ↔ ACL.

## Quyết định

1. **Lõi sở hữu quyết định "ai được đọc tài liệu nào".** Từ `Principal` + `Policy` của tenant, lõi dựng **một bộ lọc** cho mỗi lượt hỏi:
   tenant, tập vai **vừa có `documents.read` vừa được dùng để khớp ACL** (từng vai một, không ghép hai vai), phạm vi của chính vai đó, ngày
   hỏi. Đây là phần mở rộng của `Policy` (sẽ ở `core/policy`, ADR-0015 mục 5).
2. **RAG chỉ áp bộ lọc được trao, TRƯỚC khi chấm.** RAG không đọc policy, không định nghĩa vai, không nới bộ lọc; không có bộ lọc thì 0
   tài liệu (đóng mặc định). Các luật hiện có giữ nguyên: lọc trước khi chấm, IDF chỉ trên đoạn người hỏi thấy, "không có" = "không được
   xem", không đếm tài liệu ngoài quyền vào vết của người hỏi.
3. **Một bộ tên vai.** Vai trong ACL tài liệu phải là vai mà policy của tenant khai. Vai lạ là **lỗi cấu hình phải hiện ra lúc nạp** (đếm
   được, ghi trong báo cáo nạp), không âm thầm thành "không ai đọc được".
4. **Tài liệu khai ACL, lõi quyết.** ACL / phạm vi vẫn là thuộc tính của tài liệu (do người sở hữu tài liệu khai); thứ đổi chủ là phép áp
   chúng lên người hỏi.

## Hệ quả

- Thi công: lát L4 tách `Policy` ra `core/policy` và thêm phép dựng bộ lọc; lát L6 cho `intelligence/rag` nhận bộ lọc thay vì tự đọc
  `pr.Roles`. Mỗi bước có test đỏ trước, gồm ca hai vai ở mục "SUY" (đỏ ở mã hiện tại nếu suy luận đúng).
- D-RAG-02: cả hai lựa chọn a (thêm vai vào policy) / b (sửa ACL kho về 5 vai) đều là sửa cấu hình và đều thoả mục 3; lựa chọn c (giữ
  nguyên) không còn im lặng được vì vai lạ bị đếm lúc nạp. Báo cáo khuyến nghị b khi còn dữ liệu giả lập (câu Q12).
- F-006 (HYBRID thiếu quyền tài liệu từ chối cả câu) không đổi bởi ADR này; vẫn chờ câu Q7.
- Comment "lọc acl trong SQL" ở migration 0001 (D15) không sửa được trong migration đã áp; ghi đúng ở đây: lọc trong Go, sau khi nạp.

## Phương án đã cân nhắc

- **RAG tự sở hữu ACL (như hôm nay):** bị loại — hai bộ tên vai lệch nhau mà không ai báo (D-RAG-02), và RAG phải biết luật vai của lõi
  để không cho hai vai ghép quyền.
- **Lọc ACL bằng SQL / RLS theo vai:** để sau; hôm nay mọi tài liệu của tenant đã nằm trong RAM sau RLS theo tenant, và chưa có số đo
  nào đòi đẩy phép lọc xuống DB.
