# ADR-0014 — Cổng trước, tách sau: ba cổng phải nhìn thấy thay đổi trước mọi lát tách

- **Ngày:** 24/10/2026.
- **Trạng thái:** **CHẤP NHẬN 24/10/2026** theo GO của chủ dự án (phương án B′: cổng trước, tách tại chỗ, dời thư mục để sau).
- **Liên quan:** báo cáo audit vNext 24/10 (chỉ đọc, ngoài repo): §10 phiếu F-011, F-014, F-025, F-034; §14 lát L1. `architecture_test.go`;
  `testdata/tu-mien-cho-phep.tsv`; `scripts/cong/` (`cong.sh`, `so-than/`, `so_ten.py`, `skip-cho-phep.txt`); `scripts/kiem-skip-test.py`;
  `.github/workflows/ci.yml`; `scripts/dot-bien/w5-g4-acl.sh`, `scripts/dot-bien/w5-rag-hybrid.sh`.

## Bối cảnh

Các lát tách sau (sửa S4, hạ miền nhiên liệu xuống case study, ranh giới RAG, rồi dời thư mục) đều nhận "xong" bằng câu "cổng xanh, 0 mất,
0 đổi". Ở mốc `3f1bf7e` câu đó không chứng minh được hành vi giữ nguyên, vì bốn chỗ mù đã đo:

| Mã | Cổng | Chỗ mù (đã chạy lại) |
|---|---|---|
| F-011 | `architecture_test.go` | quét theo danh sách cây `sharedRoots`: literal `"construction"` ở `modules/x/x.go` → PASS (ở `core/x` thì FAIL); chỉ dò tên tenant, từ ngành (`equipment`, `lít`, `dự án`…) lọt hết |
| F-034 | `scripts/cong/so_ten.py` | chỉ so (gói, tên) + trạng thái: `d404ba4` đảo kỳ vọng, `ddf4037` giữ tên "để so theo tên không mất test" ⇒ "0 đổi" |
| F-014 | bước CI cấm SKIP | đỏ trên main (3 test đo của `evals/rag` tự SKIP); `cong.sh` không chạy `kiem-skip-test.py` nên đợt gộp D-RAG-05 không thấy |
| F-025 | neo đột biến | hai chuỗi gốc (ACL của RAG, "đoạn mang từ tiêu đề") không còn trong `intelligence/rag/rag.go` ⇒ `mutate.sh` rc=2, bằng chứng ngày 20/10 không còn áp |

Tách khi cổng còn mù thì mỗi lát có thể tắt cổng (dời gói ra khỏi `sharedRoots`), đổi kỳ vọng (giữ tên), hay SKIP một test mà vẫn báo xanh.
Vì vậy lát L1 chỉ sửa cổng, không đổi hành vi sản phẩm, và mỗi cổng phải đỏ ít nhất một lần khi cố tình phá.

## Quyết định

1. **Cổng kiến trúc là danh sách LOẠI TRỪ.** `sharedGoFiles` quét mọi tệp `.go` không phải test của repo, trừ `tenants/`, `examples/`,
   `case-studies/` (thư mục gốc), mọi `testdata/` và mọi thư mục ẩn. `TestSharedCodeHasNoTenantLiterals` và `TestRequestTypesHaveNoTenantField`
   dùng chung bộ duyệt. Gói mới hay gói dời sang `modules/`, `adapters/` tự rơi vào cổng.
2. **Cổng từ miền** `TestSharedCodeHasNoDomainLiterals`: literal chuỗi (AST, không tính comment) của mã dùng chung mang từ ngành thì đỏ.
   - Từ, so theo ranh giới chữ Unicode, không phân biệt hoa thường: `project(s)`, `equipment`, `fuel`, `nhiên liệu`, `dầu`, `lít`, `khối`,
     `thiết bị`, `dự án`, `chuyến`, `xe`, `PRJ-n`, `EX-n`.
   - `máy` chỉ tính khi đứng một mình thành cả literal (danh sách danh từ đếm): trong câu chữ nó là nghĩa chung ("máy chủ", "máy dev"),
     5 trên 6 lần trúng ở mốc này.
   - Nợ hiện có nằm ở `testdata/tu-mien-cho-phep.tsv`, mỗi dòng MỘT lần xuất hiện: tệp, literal dạng `strconv.Quote`, mã phát hiện, lý do.
     Khớp chính xác theo (tệp, literal), không theo số dòng. Ở mốc: 29 dòng — F-004 ×11, F-005 ×5, F-030 ×12, F-011 ×1 (FALSE_POSITIVE
     "khối chữ" của giao thức model). Literal mới, hay literal nợ bị sửa, thì đỏ; mục không còn trong mã cũng đỏ (mục thừa).
3. **So thân test** `scripts/cong/so-than` (Go, `go/parser` + `go/scanner`): so mọi khai báo mức gói của `*_test.go` giữa hai cây — hàm
   `Test*` / `Benchmark*` / `Fuzz*`, cả bảng kỳ vọng và hàm trợ giúp — theo dãy token (đổi comment, định dạng, chỗ đặt không tính).
   - Dùng: `go run ./scripts/cong/so-than <cây gốc> <cây mới> [tệp đổi có chủ ý]`, cây gốc dựng bằng `git worktree add --detach`.
   - Tệp đổi có chủ ý: mỗi dòng `<tên> <lý do>`; đi kèm báo cáo lát, người soát độc lập đọc từng dòng.
   - Thoát 1 khi có khai báo đổi mà không khai, hoặc mục khai thừa. Trên cặp `d404ba4^` / `ddf4037` nó báo 6 test đổi thân, có
     `TestClassifyGiaiThichSoLuongYDinhYeuVaCumTuKhoa`.
4. **Cổng SKIP.** `scripts/cong/skip-cho-phep.txt` liệt kê ba test đo của `evals/rag` kèm lý do (cần tệp cấu hình riêng hoặc bên thứ ba,
   gửi dữ liệu ra ngoài). Bước CI và cuối `scripts/cong/cong.sh` chạy `kiem-skip-test.py` với cùng tệp. `cong.sh` chạy thêm `so-than` khi
   đặt `CONG_MOC=<cây mốc>` (và `CONG_DOI_CO_CHU_Y=<tệp>`), và thoát 1 khi go test, cổng SKIP hay so thân đỏ.
5. **Neo đột biến theo mã hiện tại.** Chạy lại hai danh mục `w5-*` ở mốc này thì ngoài hai neo F-025 còn ba neo gãy nữa (rc=2): AD, AH, AV.
   Cả năm được đặt lại:
   - AL: `if vis[d] = ix.visible(d, pr); !vis[d] {` (lọc ACL nay ở `visible()`). Neo `if !vis[c.doc] {` ở vòng chấm điểm SỐNG SÓT: tài liệu
     không thấy đã không vào `serving` / `live`.
   - AR: `terms: indexTerms(d.Title + " " + scored)`.
   - AD: `ok := pol.Allows(pr, kdef.Capability)`.
   - AH: `if a.quantityMarker(in.Raw) == "" {` (`b7ec80e`).
   - AV: điều kiện của `check("ý định tri thức khớp kèm", …)` (`07d6665`).

## Luật cập nhật danh sách

- Lát nào gỡ một literal nợ (L5: F-004, F-005, F-030) thì xoá ĐÚNG mục của nó trong `tu-mien-cho-phep.tsv` trong CÙNG commit. Quên thì cổng
  đỏ "mục thừa".
- Không thêm mục để cổng xanh. Mục mới phải có mã phát hiện và lý do, và qua soát độc lập. Mục dương tính giả ghi `FALSE_POSITIVE:` và nói
  vì sao.
- Sửa chữ trong một literal nợ là literal mới: cập nhật mục trong cùng commit, giữ mã phát hiện cũ. Dời tệp thì sửa đường dẫn của mục,
  cùng commit với neo đột biến (luật của lát HOÃN).
- Thêm thư mục gốc được mang tên tenant (vd khi dời sang `case-studies/`) thì sửa `tenantRoots` và ghi vào ADR này.
- `skip-cho-phep.txt` chỉ nhận test ĐO cần thứ CI cố ý không có. Test cần PG hay công cụ cài được thì không bao giờ vào đây. Mục không còn
  SKIP làm cổng đỏ (THỪA): xoá mục.
- Neo đột biến gãy (rc=2) thì sửa chuỗi gốc theo mã mới rồi chạy lại cả danh mục; không xoá dòng cho xanh.

## Hệ quả

- Cổng đỏ đúng các ca âm của L1:
  - literal tenant hoặc từ miền ở `modules/x`;
  - `"equipment"` mới ở `core/`;
  - cặp `d404ba4^` / `ddf4037`;
  - một SKIP lạ, mục cho phép thiếu hoặc thừa.
- Cổng xanh trên cây của lát này, cổng danh tính không đổi.
- Giới hạn đã biết:
  - `so-than` chỉ đọc `*_test.go`. Kỳ vọng nằm trong tệp dữ liệu (yaml ca, `passing.txt`) đổi thì không hiện. `d404ba4` cũng sửa
    `tenants/construction/win/passing.txt`.
  - Cổng từ miền chỉ khớp chữ dựng sẵn NFC (mọi tệp `.go` hiện là NFC). "máy" trong câu chữ không bị bắt.
  - Bỏ `--cho-phep` khỏi bước CI không có test cục bộ nào bắt (đột biến sống). Nó hỏng theo hướng an toàn: CI đỏ ngay lần chạy đầu.
  - Kiểm `!vis[c.doc]` ở vòng chấm điểm của RAG không có test nào bắt khi bỏ. Ca chưa test: hai bản cùng mã tài liệu, cùng hiệu lực, khác ACL.
- Tài liệu cũ nhắc `sharedRoots` (ADR-0011, `docs/viec/1-hanh-dong/HOP-DONG.md`) mô tả mã tại commit chúng dẫn. Chúng không được sửa ở đây.
