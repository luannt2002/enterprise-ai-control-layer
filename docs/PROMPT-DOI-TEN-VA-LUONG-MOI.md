# PROMPT — Phiên BUILD: pull về, đổi tên lại toàn bộ, viết lại mọi luồng theo roadmap mới

> **Gửi:** phiên BUILD (team agent trên server), gồm cả các agent con và workflow do phiên đó điều phối.
> **Từ:** chủ dự án, qua phiên điều phối. **Ngày:** tối 21/10/2026.
> **Loại:** yêu cầu có khuôn trả lời (kênh `docs/PROMPT-*.md`, brief BUILD §2). Làm xong thì commit tệp phản hồi theo §6.
> **Căn cứ:**
> - `docs/CHO-QUYET-DINH.md` §0: **D-RM-01** (roadmap mới phạm vi rộng nhất) và **D-RM-02** (đổi tên lại, giữ link git);
> - §4 của cùng tệp: câu còn mở D-RM-03, D-RM-04;
> - `docs/ROADMAP-DAY-DU.md`;
> - `docs/PROMPT-BUILD-EACL.md` §4.

---

## 0. Pull về và đọc trước

1. `git fetch origin && git pull --ff-only origin main`. HEAD phải chứa `65467eb` (brief BUILD theo D-RM-01).
2. Đọc **trọn** các mục sau:
   - `docs/CHO-QUYET-DINH.md` §0, hai dòng D-RM-01, D-RM-02, và §4;
   - `docs/ROADMAP-DAY-DU.md`;
   - `docs/PROMPT-BUILD-EACL.md` §4.
3. Ghi vào `docs/bao-cao/TIEN-DO-BUILD.md`: "NHẬN PROMPT-DOI-TEN-VA-LUONG-MOI", kèm giờ và HEAD.

## 1. Tên mới là gì

**Tên mới = tên gốc ngày 20/10.** Đọc nguyên văn bằng:

```bash
git show d70ebc8:CLAUDE.md | sed -n 313,320p
git show d70ebc8:docs/01-architecture.md | sed -n 1,4p
```

Trong tệp này ký hiệu như sau:
- `<TÊN>`: tên viết tắt chữ hoa;
- `<tên>`: tên viết tắt chữ thường;
- `<TÊN ĐẦY ĐỦ>`: tên dài.

Cả ba lấy từ hai dòng trên. Tệp này không chép nguyên chữ vì cổng danh tính hiện hành còn chặn.

**Lời chủ dự án:** "đổi hết tất cả tên ở code, ở doc, ở md lại thành cái tên kia; **chỉ có cái link git chưa đổi** thôi."

## 2. Việc 1: đổi tên lại toàn bộ (D-RM-02)

### 2.1 Kiểm kê trước khi sửa

Lập bảng kiểm kê. Không sửa gì khi chưa có bảng. Bảng xếp số chỗ xuất hiện tên hiện tại (`EACL`, `eacl`, tên dài hiện tại) theo 6 nhóm:

| Nhóm | Ví dụ | Xử lý |
|---|---|---|
| A. Link git | tên repo, remote, đường dẫn module Go `github.com/luannt2002/enterprise-ai-control-layer`, mọi import | **GIỮ NGUYÊN** |
| B. Mã | comment, chuỗi, định danh, tên test (`TestDanhTinhChiConEACL`), thông báo lỗi | đổi |
| C. Định danh chạy | tiền tố biến môi trường `EACL_*`, GUC `eacl.tenant_id`, vai PG `eacl_app`, DB phụ `eacl_giu_lai`, khoá JSON `eacl_unstable_cases` | đổi, theo §2.3 |
| D. Tài liệu hiện hành | README, `CLAUDE.md`, `docs/` hiện hành, `CONVENTIONS.md`, tên tệp có chữ tên (ví dụ `docs/PROMPT-BUILD-EACL.md`) | đổi; đổi tên tệp thì sửa mọi đường dẫn trỏ tới nó |
| E. Bằng chứng lịch sử, tệp đã băm | `legacyAllowed` của `danh_tinh_test.go` (HISTORICAL_EVIDENCE, IMMUTABLE_HASHED_ARTIFACT); bộ giữ lại H1 / H3 / H4 cùng cam kết băm; báo cáo ca cũ; mục cũ của CHANGELOG; dòng quyết định cũ trong `CHO` | **KHÔNG CHẠM** |
| F. Ngoài phạm vi | `docs/DEBUG-LO-TRINH.md` (chưa commit, của phiên debug) | không chạm |

Lệnh gợi ý:

```bash
git grep -n -i -E 'eacl|enterprise[- ]ai[- ]control[- ]layer' -- . ':!docs/bao-cao/' | wc -l
```

Rồi tách theo nhóm. Chép số trước và sau vào tệp phản hồi.

### 2.2 Ánh xạ tên

| Hiện tại | Mới |
|---|---|
| `EACL` | `<TÊN>` |
| `eacl` | `<tên>` |
| tên dài hiện tại | `<TÊN ĐẦY ĐỦ>`. Câu định nghĩa thì dùng §1 của `ROADMAP-DAY-DU.md` |
| `TestDanhTinhChiConEACL` | `TestDanhTinhChiCon<TÊN>` |
| `docs/PROMPT-BUILD-EACL.md` | `docs/PROMPT-BUILD-<TÊN>.md` |

Chỗ nào đổi tên làm câu sai nghĩa (ví dụ câu kể lịch sử "đổi tên sang EACL ngày 21/10") thì **giữ nguyên** và xếp vào nhóm E.

### 2.3 Định danh chạy (nhóm C)

- **Biến môi trường:** đọc `<TÊN>_*` trước, vẫn nhận `EACL_*` làm bí danh tương thích. Làm giống lần đổi trước đã giữ bí danh cho tên cũ (`intelligence/models/messages.go:409`). Có test cho cả hai.
- **GUC RLS và vai PG:**
  - đổi bằng **migration mới**, không sửa migration đã áp;
  - chứng minh RLS vẫn đỏ đúng chỗ bằng test có PG và `scripts/dot-bien/w4-pg-rls.sh`;
  - kiểm `pgstore.Open` vẫn từ chối vai vượt RLS (`cc84b83`).
- **Khoá JSON trong bằng chứng thô:** đổi khoá mới. Công cụ đọc bằng chứng cũ phải nhận cả khoá cũ, để bằng chứng đã lưu vẫn đọc được.
- **Script và đột biến** (`scripts/`, `scripts/dot-bien/`): sửa theo, rồi chạy lại.

### 2.4 Cổng danh tính: đảo chiều

- `legacyIdentity` trong `danh_tinh_test.go` chuyển sang bắt **tên hiện tại**.
- Mỗi tệp còn tên hiện tại phải có lý do trong `legacyAllowed`: bằng chứng lịch sử, bí danh tương thích, tham chiếu ngoài.
- Giữ test âm chứng minh cổng đỏ được. Đột biến n42 phải bắt.
- Nếu cổng cũ đang chặn tên gốc ở tệp nào thì gỡ.

### 2.5 Không đổi hành vi: bằng chứng phải nộp

- `go vet ./...` sạch; `gofmt -l .` rỗng.
- `go test -count=1 ./...` **có PG**, so theo **tên** với lần chạy trước khi đổi:
  - 0 test mất;
  - 0 test đổi kết quả;
  - test bị đổi tên thì ghi cặp tên cũ → tên mới.
- Bộ 117 ca; H1 hai đường; H3: **0 ca đổi nhãn** (`winreport` trước và sau).
- Đột biến n42 (cổng danh tính) và `w4-pg-rls`: bắt đủ.
- **Soát độc lập** (agent chỉ đọc), trước khi push: tìm chỗ sót, chỗ đổi nhầm nhóm E, chỗ đổi nhầm link git.

### 2.6 Commit

Gợi ý tách thành:
- (a) mã + định danh chạy + migration;
- (b) tài liệu + tên tệp;
- (c) cổng danh tính.

Mỗi commit qua cổng §6 của brief. Push thường, không force.

## 3. Việc 2: viết lại mọi luồng theo tên mới

Sau khi đổi tên xong:

1. **`docs/ROADMAP-DAY-DU.md`**
   - Thay cách gọi tạm "dự án" bằng `<TÊN>`.
   - Bỏ câu "tạm gọi là dự án" ở phần đầu.
   - Giữ nguyên nội dung các luồng L0–L10.
2. **`docs/01-architecture.md`:** viết lại phần luồng cho khớp `ROADMAP-DAY-DU.md` §3 (7 mặt phẳng) và §5 (luồng L0–L10).
   - Mỗi luồng ghi rõ **HIỆN CÓ** (kèm `tệp:dòng` hoặc test) và **ĐÍCH** (phase nào).
   - Không vẽ luồng đích rồi gọi là runtime.
3. **`CLAUDE.md` §9:** câu sứ mệnh lấy §1 của roadmap mới (tên + hai chế độ sở hữu).
   - Commit riêng, diff rõ.
   - Không xoá luật lịch sử còn cần (điều kiện của D-V3-06).
   - `CLAUDE.md` §11 (luồng Ask) và §25 (bản đồ repo): đối chiếu với L2 và các mặt phẳng; chỉ sửa chỗ lệch.
4. **README, `docs/STATUS.md`, `docs/CONVENTIONS.md`:** tên mới, trỏ tới roadmap mới.
5. **Ghi nhận ở tài liệu cũ:**
   - `docs/ROADMAP.md`, `docs/v3/*`: đã có khung "tra cứu". Chỉ đổi tên ở phần còn hiện hành, nếu có.
   - Thân lịch sử: giữ nguyên.

## 4. Việc 3: tiếp tục theo brief

Sau Việc 1 và Việc 2, làm tiếp `docs/PROMPT-BUILD-<TÊN>.md` §4 mục 2–7:
- đóng Phase 0b (R4-2…4 → vòng S4 → H4 một lần);
- phép đo;
- làm giấy Phase 0a và Phase 1;
- Phase 6 nếu S4 vẫn là sàn;
- tài liệu lệch, handoff.

Mã COMMAND / WORKFLOW / EVENT **chờ D-RM-04**.

## 5. Không được

- đổi tên repo, remote, đường dẫn module Go;
- sửa tệp nhóm E;
- đọc nội dung H4 / H5 chưa chạy;
- gọi model trả tiền khi chưa có quyền chi;
- force push, viết lại lịch sử đã push;
- ghi cột "Trả lời" của `CHO` khi chủ dự án chưa nói;
- gộp đổi tên với thay đổi hành vi trong cùng một commit.

## 6. Khuôn trả lời

Commit tệp `docs/bao-cao/<ngày>-phan-hoi-doi-ten.md`, gồm:
1. **Bảng kiểm kê** trước và sau, theo 6 nhóm A–F, kèm lệnh đếm.
2. **Ánh xạ đã dùng**, kèm danh sách chỗ giữ nguyên có lý do.
3. **Định danh chạy:** đã đổi gì, migration nào, bí danh tương thích nào, test nào chứng minh.
4. **Bằng chứng §2.5:** lệnh và kết quả; so test theo tên (tên đổi → cặp cũ / mới).
5. **Kết quả soát độc lập:** phát hiện và đã sửa gì.
6. **Việc 2:** các tệp đã viết lại; luồng nào HIỆN CÓ, luồng nào ĐÍCH.
7. **Câu cần chủ dự án quyết:** nếu có, đưa vào `CHO` và để trống cột Trả lời.

Cập nhật `TIEN-DO-BUILD.md` mỗi lần push (brief §7).
