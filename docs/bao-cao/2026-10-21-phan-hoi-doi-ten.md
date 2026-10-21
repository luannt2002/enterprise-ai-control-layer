# Phản hồi PROMPT-DOI-TEN-VA-LUONG-MOI: đổi tên lại EAIS (D-RM-02) và viết lại luồng

> **Gửi:** chủ dự án và phiên điều phối. **Từ:** phiên BUILD, tối 21/10/2026.
> **Trả lời cho:** `docs/PROMPT-DOI-TEN-VA-LUONG-MOI.md` (khuôn §6). Nhận lúc 18:0x, HEAD `32c8cfe`.
> **Commit** (theo thứ tự trên `main`):
> - Việc 1: `945777c` (cổng danh tính đảo chiều) · `423e4d3` (mã + định danh chạy + migration 0002) · `730926f` (tài liệu + đổi tên tệp brief);
> - sửa sau soát lượt 1: `dbdf7ca` (mã, cổng, test, đột biến) · `7738a3d` (tài liệu) · `cdf7d84` (mốc vận hành đo lại);
> - Việc 2: `52c9399` (ROADMAP-DAY-DU, README, STATUS, CONVENTIONS) · `34240a9` (CLAUDE.md §9, commit riêng) · `bff56c4` (CLAUDE.md §25) · `9da639c` (01-architecture);
> - câu mở `93649ba` (D-RM-05); sửa sau soát lượt 2 và soát 01-architecture: `efc2cad` · `2b8fb94` · `903623d`;
> - tiến độ `2efed6a`; commit chứa tệp này.

## 1. Bảng kiểm kê trước và sau

Lệnh đếm (chỉ đọc, chạy được trên mọi revision):

```bash
git grep -n -i -E 'eacl|enterprise[- ]ai[- ]control[- ]layer' <rev> -- . ':!docs/bao-cao/' | wc -l   # lệnh gợi ý của prompt
python3 -I scripts/kiem-ke-doi-ten.py <rev> [--chi-tiet]                                               # tách 6 nhóm, đếm từng lần
```

Luật xếp nhóm nằm ở đầu `scripts/kiem-ke-doi-ten.py`. Thêm nhóm **FP** (khớp nhầm): chữ "eACL" trong tên test `TestRetrieveACL…` (ACL = access control list), không phải tên sản phẩm.

Cột "Sau" đo ở `730926f`, lúc đổi tên xong. Ở HEAD (sau sửa theo soát) số tăng thêm vì ca âm mới của cổng, chính công cụ kiểm kê, chuỗi đột biến và 2 nhãn `d5probe` trả về nguyên văn: lệnh gợi ý 630 dòng; 814 lần = A 231 · B 68 · C 54 · D 15 · E 436 · F 0 · FP 10. Mọi lần thêm đều có mục trong `legacyAllowed`.

| | Trước (`ca62760`) | Sau (`730926f`) | Ghi chú |
|---|---|---|---|
| Lệnh gợi ý (số dòng, bỏ `docs/bao-cao/`) | 1.126 | 591 | |
| Tổng lần xuất hiện (cả `docs/bao-cao/`) | 1.463 | 748 | |
| **A. Link git** | 219 | 226 | giữ nguyên; +7 = ca âm mới của cổng (6) + một import mới (`platform/pgstore/env_test.go`) |
| **B. Mã** | 316 | 25 | còn: mẫu và ca âm của cổng, chuỗi đột biến n42, một chú thích bí danh |
| **C. Định danh chạy** | 196 | 44 | còn: bí danh `EACL_*` + test, migration 0002 (gỡ quyền vai cũ), test 0002, chuỗi đột biến w4, mã hệ cũ trong `du-toan-g7.py` |
| **D. Tài liệu hiện hành** | 300 | 15 | còn: `docs/ops/README.md` (ghi bí danh, cách gỡ container tên cũ, tên tệp báo cáo lịch sử) và §4 mục 1 của brief (nêu tên cũ để định nghĩa phép đổi) |
| **E. Lịch sử, tệp đã băm** | 430 | 434 | 0 tệp E đổi; +5 là mục CHANGELOG mới nêu tên cũ, −1 là dòng khuôn của TIEN-DO (đường dẫn brief) |
| **F. Ngoài phạm vi** | 0 | 0 | `docs/DEBUG-LO-TRINH.md` không có trên git, không có ở máy này |
| **FP. Khớp nhầm** | 2 | 4 | +2 là lý do trong bảng cho phép của cổng có nhắc tên test `TestRetrieveACL…` |
| Tên tệp có chữ tên cũ | 4 | 3 | `docs/PROMPT-BUILD-EACL.md` → `docs/PROMPT-BUILD-EAIS.md`; 3 tệp còn lại thuộc E |

Mọi lần còn lại ở B / C / D / FP nằm trong `legacyAllowed` của `danh_tinh_test.go`, mỗi tệp một lý do và hạn mức đúng bằng số lần hiện có (bảng ở mục 2).

## 2. Ánh xạ đã dùng và chỗ giữ nguyên

| Trước | Sau |
|---|---|
| `EACL` / `eacl` | `EAIS` / `eais` (đổi đúng chữ hoa thường; định danh Go như `sysEACL`, `EACLPass`, `eaclAsker` thành `sysEAIS`, `EAISPass`, `eaisAsker`) |
| "Enterprise AI Control Layer" | "Enterprise AI Systems" |
| câu định nghĩa | §1 của `docs/ROADMAP-DAY-DU.md` (README, `docs/ops/BO-NHO.md`); câu gốc `d70ebc8` ở `docs/01-architecture.md:3` |
| `TestDanhTinhChiConEACL` | `TestDanhTinhChiConEAIS` |
| `docs/PROMPT-BUILD-EACL.md` | `docs/PROMPT-BUILD-EAIS.md` (`git mv`; sửa đường dẫn ở tài liệu hiện hành) |
| gói test gốc `eacl_test` | `eais_test` |

**Giữ nguyên, có lý do** (đủ danh sách ở `legacyAllowed`):

| Chỗ | Lý do |
|---|---|
| link git: `github.com/luannt2002/enterprise-ai-control-layer`, mọi import, URL clone | D-RM-02: link git chưa đổi |
| `CHANGELOG.md` các mục cũ | E: ghi đúng tên lúc đó |
| `docs/CHO-QUYET-DINH.md` | E: văn bản quyết định (D-G7-8 dặn đặt khoá ở `EACL_MODEL_API_KEY` ⇒ bí danh giữ cho quyết định này còn chạy được) |
| `docs/PROMPT-DOI-TEN-VA-LUONG-MOI.md`, `docs/PROMPT-AUDIT-ROADMAP.md` | E: yêu cầu có ngày |
| `docs/ROADMAP.md` | E: tra cứu theo mốc `8280f40`; chỉ thêm một dòng khung "tên trong thân tệp là tên lúc viết" |
| `docs/v3/*` | E: kế hoạch 20/10; "EACL V3" là tên riêng của gói kế hoạch; trỏ thư mục nguồn ở máy `docs/sources/eacl-v3/` |
| `docs/study/*` | E: tài liệu học phiên sáng 21/10; câu gắn danh tính lúc đó (ví dụ "tên sản phẩm cũ, trước 21/10" chỉ EAIS — đổi chữ thì sai nghĩa); `docs/ROADMAP.md` ghi "không sửa tệp đó" |
| `docs/g7/REVIEW-D5.md`, `CONTAMINATION-AUDIT.md`, `CHECKPOINT-A-H.md` | E: bản soát, kiểm toán, checkpoint ngày 20/10 |
| `docs/g7/holdout*/**` | E: bộ giữ lại đóng băng có băm và nhật ký chạy |
| `docs/bao-cao/2026-*` | E: báo cáo ca cũ (kể cả tên tệp `2026-10-21-migration-eacl.md`) |
| `db/migrations/0001_win.sql` | migration đã áp; định danh chạy đổi bằng `0002` |
| muối `"eacl-holdout-2026-10-20\|"` (`scripts/dong-bang-giu-lai.py`, ADR-0006 §4) | đổi là đổi cách chia bộ giữ lại đã đóng băng |
| §4 mục 1 của `docs/PROMPT-BUILD-EAIS.md` | nêu tên cũ để định nghĩa phép đổi |
| đường dẫn cũ `docs/PROMPT-BUILD-EACL.md` trong tệp E | E thắng luật "sửa mọi đường dẫn"; mục CHANGELOG mới ghi cặp cũ → mới để người đọc lần theo |

## 3. Định danh chạy

| Định danh | Trước | Sau | Tương thích | Chứng minh |
|---|---|---|---|---|
| biến model | `EACL_MODEL_*` (bí danh `EAIS_MODEL_*`) | `EAIS_MODEL_*` | `EACL_MODEL_*` đọc khi biến `EAIS_` cùng tên không đặt; `EAIS_` đã đặt, kể cả rỗng, luôn thắng (`models.modelEnv`) | `TestConfigFromEnvTienToTuongThich`; đột biến n42 ME1, ME2 |
| DSN PG | `EACL_PG_ADMIN_DSN`, `EACL_PG_DSN` (mỗi nơi tự đọc) | `EAIS_PG_ADMIN_DSN`, `EAIS_PG_DSN`, đọc ở MỘT hàm `pgstore.EnvDSN` (test PG, test WIN trên PG, `winreport -pg`) | cùng luật bí danh; `scripts/chay-giu-lai.sh` kiểm DSN theo đúng luật đó | `TestEnvDSNTienToTuongThich`; n42 PE1, PE2 |
| GUC RLS | `eacl.tenant_id` | `eais.tenant_id` | migration MỚI `0002_dinh_danh_eais.sql` (`ALTER POLICY` trên 4 bảng); GUC tên cũ không mở dòng nào | `TestMigration0002DinhDanhEAIS`; `TestRLSCanaryB`; `TestWinTrenPG`; w4 MG1, MG2, PA |
| vai ứng dụng | `eacl_app` | `eais_app` | 0002 tạo nếu chưa có, cấp SELECT / INSERT / DELETE (đúng bằng vai cũ), gỡ mọi quyền của `eacl_app` trên bảng nghiệp vụ; vai là của cả cụm nên không xoá | `TestMigration0002DinhDanhEAIS`: đúng tập quyền ở mức bảng (SELECT … MAINTAIN), GRANT OPTION, qua PUBLIC, mức cột; cờ ENABLE + FORCE trên 4 bảng. `TestOpenTuChoiVaiVuotRLS` (ca dương `role=eais_app`); w4 MG3–MG6, X1–X7, X9–X11, BR1–BR4 |
| PG dev | compose `eacl`, `eacl-pg`, DB `eacl` | `eais`, `eais-pg`, DB `eais` | dữ liệu tmpfs: `docker compose -p eacl down` rồi dựng lại (ghi ở `docs/ops/README.md`) | dựng lại 18:3x, test có PG chạy trên đó |
| DB bộ giữ lại | `eacl_giu_lai` | `eais_giu_lai` | `chay-giu-lai.sh` từ chối DB khác | đọc mã |
| mã hệ G7 / khoá JSON | `"eacl"`, `eacl_unstable_cases` | `"eais"`, `eais_unstable_cases` | công cụ đọc bằng chứng thô đã lưu: `scripts/du-toan-g7.py` nhận mã hệ cũ `eacl` như `eais`. Không công cụ nào đọc `aggregate.json` (chỉ test đọc đầu ra vừa ghi) | chạy trên bằng chứng chạy khô đã lưu: giống từng dòng trừ nhãn; ca âm bỏ lớp tương thích: 21 dòng khác |
| vai tạm của test | `eacl_test_*` | `eais_test_*` | — | `TestOpenTuChoiVaiVuotRLS` |

Hai đường của migration đều đã chạy:
- **nâng cấp**: database cũ đã áp 0001 (container cũ, trước khi gỡ): test `platform` + `winrun` có PG 11/11 ĐẠT; sau đó `schema_migrations` có 0001, 0002 và 4 chính sách đọc `eais.tenant_id`;
- **dựng mới**: container `eais-pg` mới: toàn bộ test có PG 455/0/0.

## 4. Bằng chứng §2.5

Mốc trước đổi tên đo lại ở `ca62760` (mã giống chốt ca `2e4897b` / `936d090`): khớp mốc chốt ca từng dòng.

| Kiểm | Trước | Sau |
|---|---|---|
| `go vet ./...` | sạch | sạch (cả 3 commit) |
| `gofmt -l` | 0 tệp | 0 tệp |
| `go test -count=1 ./...` có PG | 453 / 0 / 0 | 455 / 0 / 0 |
| winreport bộ 117 | 117/117 | 117/117 |
| H1 bộ nhớ / PG | 89/104 · 89/104 | 89/104 · 89/104 |
| H3 bộ nhớ / PG | 109/134 · 109/134 | 109/134 · 109/134 |
| winreport, so từng dòng (nhãn + chữ câu trả lời mọi ca) | — | 0 dòng khác ở cả 5 đường, ngoài 2–4 dòng số đo thời gian mỗi đường |

So test theo **tên** (453 → 455): 0 mất, 0 đổi kết quả; 2 test mới (`TestEnvDSNTienToTuongThich`, `TestMigration0002DinhDanhEAIS`); 4 cặp đổi tên:

| Cũ | Mới |
|---|---|
| `TestDanhTinhChiConEACL` | `TestDanhTinhChiConEAIS` |
| `TestG7DoesNotImportEACL` | `TestG7DoesNotImportEAIS` |
| `TestCompareG7EachRuleBlocksAlone/EACL_thua_một_lớp` | `…/EAIS_thua_một_lớp` |
| `TestCompareG7EachRuleBlocksAlone/EACL_thua_đúng_1_ca` | `…/EAIS_thua_đúng_1_ca` |

Đột biến (chạy trong worktree gương, không trên cây chính):
- n42 (cổng danh tính + bí danh) ở trạng thái commit 1: 14/14 bị bắt (thêm DT10 không bỏ link git, DT11 bỏ rộng hơn link git, DT12 bỏ hẹp hơn link git);
- toàn bộ 35 danh mục ở trạng thái commit 2 (18:35 → 18:56, có PG): **676 bị bắt / 0 sống / 0 không biên dịch / 0 chuỗi gốc lệch / 0 lỗi khôi phục**; n42 16/16 (thêm PE1, PE2), w4-pg-rls 16/16 (thêm MG1–MG6 cho 0002), n43 53/53. Lần chạy đủ trước đó (sáng 21/10): 598.
- w4-pg-rls: đột biến migration chạy trên database `eais_dot_bien` tạo lại cho TỪNG đột biến (migration chỉ áp một lần mỗi database; danh mục cũ chưa từng có đột biến migration vì lẽ đó). Đột biến "không tạo vai" và "cấp vai thành viên" không đưa vào danh mục: vai là của cả cụm PG, xoá database không gỡ.
- ở HEAD, sau sửa theo các lượt soát: **n42 25/25, w4-pg-rls 26/26** bị bắt.

Mỗi commit tự đứng xanh: `go vet ./...` và cổng danh tính chạy ở worktree đúng cây của TỪNG commit trong chuỗi; test có PG đủ bộ ở commit 1 (453/0/0), commit 3 và HEAD (455/0/0, cùng tên).

## 5. Kết quả soát độc lập

**Lượt 1** (khoảng 24 phút, xong lúc 19:10): workflow 3 người soát, sonnet, chỉ đọc (agent không có công cụ sửa tệp), trên `945777c` … `730926f`; mỗi người một góc. Kết luận cả ba: **ĐẠT CÓ GHI CHÚ, 0 S1**.

| Góc | Phát hiện | Mức | Đã làm |
|---|---|---|---|
| B. Định danh chạy | B-01: test 0002 chỉ hỏi 4 quyền ⇒ đột biến cấp thêm TRUNCATE (không chịu RLS: xoá sạch bảng của mọi tenant) sống sót | **S2** | test kiểm ĐÚNG tập quyền: `eais_app` chỉ SELECT / INSERT / DELETE, không UPDATE / TRUNCATE / REFERENCES / TRIGGER / MAINTAIN, không GRANT OPTION, kể cả qua PUBLIC; vai cũ không còn quyền nào. Đột biến w4 thêm X1–X5, cả 5 bị bắt (`dbdf7ca`) |
| C. Đối chiếu yêu cầu | C-01: cổng danh tính lọt 7 dạng viết tên dài (gạch dưới, CamelCase, viết liền, dấu chấm, nhiều dấu cách, xuống dòng) dù comment nói "mọi dạng viết khác đều bị đếm" | **S2** | mẫu nhận mọi cách nối từ (kể cả khoảng trắng Unicode); ca âm 7 dạng; đột biến n42 DT13 (`dbdf7ca`) |
| C | C-02: mốc test ở `docs/ops/README.md` là 440 tại `9dede54`, lệch HEAD ⇒ phiên mới sẽ báo động giả | **S2** | đo lại tại `7738a3d`: có PG 455 / 0 / 0 (332 test cấp cao), không PG 448 + 7 SKIP, scale smoke ĐẠT (`cdf7d84`) |
| B | B-02: không test nào khẳng định FORCE RLS sau 0002 | S3 | test khẳng định ENABLE + FORCE trên 4 bảng; đột biến X4 bị bắt |
| B | B-03: test 0002 đỏ oan khi cụm không có vai cũ | S3 | vai không có trong cụm ⇒ coi là không có quyền |
| B | B-04: lớp bí danh ở `chay-giu-lai.sh`, `winreport`, `du-toan-g7.py` chưa có test tự động giữ | S3 | **chưa sửa**, ghi vào giới hạn (mục 7): `winreport` đã dùng chung `pgstore.EnvDSN` (có test + đột biến); hai script còn thử tay |
| B | B-05: comment 0002 nói vai không superuser / không BYPASSRLS nhưng thuộc tính chỉ đặt khi tạo vai | S3 | sửa comment; chốt chặn là `pgstore.Open` |
| B | B-06: CHANGELOG chưa liệt kê đủ khoá theo mã hệ trong bằng chứng thô | S3 | bổ sung (`dbdf7ca`) |
| C | C-04: xoá link git rồi đếm có thể ghép chữ hai bên thành khớp giả | S3 | thay bằng NUL; ca âm; đột biến DT14 |
| C | C-05: hạn mức chỉ là trần ⇒ khi số lần giảm, phần thừa là chỗ cho tên cũ quay lại | S3 | hạn mức phải ĐÚNG BẰNG; mục cho tệp hết lần cũng báo; ca âm; đột biến DT15, DT16. Luật mới bắt ngay một mục thừa thật (`rag_test.go`) |
| C | C-06: "eACL" trong `RetrieveACL` buộc mục FALSE_POSITIVE, mã mới kiểu `CreateACL` sẽ làm cổng đỏ | S3 | không đếm "eACL" ngay sau chữ cái; đột biến DT17 (đếm lại) và DT18 (loại quá rộng, lọt `sysEACL`) |
| C | C-07, A-04: vài lý do trong bảng cho phép không khớp (nhãn muối băm, mốc ngày, phần v3 do đợt đổi tên sáng 21/10 đổi máy móc) | S3 | sửa chữ lý do |
| A. Sót, đổi nhầm | A-02: câu nguồn của đặc tả bộ chấm v2 nói "chỉ thay đường dẫn máy" | S3 | ghi thêm việc thay tên sản phẩm |
| A | A-03: `d5probe` là chương trình đo của bản soát D5 (bằng chứng) nhưng nhãn in ra bị đổi | S3 | trả nhãn về nguyên văn bản gốc `d03f962`, xếp nhóm E |
| A | A-06: câu container tên cũ sai mốc (container 20/10 đã mang tên hiện hành) | S3 | sửa mốc 09:29 → 18:4x ngày 21/10 |
| A, C | A-01 / C-10: `scripts/kiem-ke-doi-ten.py` chưa có mục cho phép (cổng sẽ đỏ trên cây làm việc) | S3 | có mục FALSE_POSITIVE khi commit |
| A, C | A-05, A-07, C-03, C-08, C-09, C-11: link cũ trong tệp E, câu sứ mệnh §9 còn của ngày 21/10, `docs/study`, băm `chay-giu-lai.sh` đổi, dòng khung ở `docs/ROADMAP.md`, Việc 2 | S3 | §9 và ROADMAP-DAY-DU: làm ở Việc 2 (`34240a9`, `52c9399`); các điểm còn lại ghi ở mục 7 |

Hai lỗi tự bắt trước khi push, không do người soát:
- phép đổi máy móc làm gãy link `docs/bao-cao/2026-10-21-migration-eacl.md` trong `docs/ops/README.md` ⇒ sửa trước commit 3;
- `git mv` của brief lọt vào commit 2, và dòng CHANGELOG của B-06 lọt sang commit sau hạn mức của nó ⇒ cổng danh tính sẽ đỏ ở cây giữa chừng ⇒ làm lại commit (chưa push); kiểm vet + cổng ở cây của TỪNG commit trong chuỗi.

Sau sửa (`bff56c4`): test có PG 455 / 0 / 0, trùng tên; n42 22/22; w4-pg-rls 21/21.

**Lượt 2** (2 người soát, sonnet, chỉ đọc), trên `dbdf7ca` … `bff56c4`:

| Góc | Kết luận | Phát hiện chính | Đã làm |
|---|---|---|---|
| Sửa | ĐẠT CÓ GHI CHÚ, 0 S1, 0 S2, 9 S3 — các commit sửa đóng đúng phát hiện lượt 1; tự đo lại: n42 22/22, w4 21/21, mốc 455 / 448 + 7 SKIP, scale smoke | SUA-01 tên dài còn lọt (ngắt dòng qua tiền tố comment, gạch Unicode, "/", ký tự vô hình, in đậm từng từ); SUA-03 cổng không quét tên tệp; SUA-06 quyền mức cột lọt test 0002 (đột biến `UPDATE (tenant_id)` sống); SUA-07 thiếu đột biến MAINTAIN / PUBLIC / chủ bảng; SUA-04, 05, 09 | `2b8fb94`: dò tên dài trên bản chỉ giữ chữ / số (mọi cách nối từ), đếm cả đường dẫn tệp, chặn mục thư mục có hạn mức, nhãn 0001; test 0002 kiểm mức cột; w4 thêm X6, X7, X9, X10, X11 và đọc DSN từ môi trường; n42 thêm DT19–DT21. Sau sửa: n42 25/25, w4 26/26 |
| Việc 2 | ĐẠT CÓ GHI CHÚ, 0 S1, 1 S2, 5 S3 — ROADMAP-DAY-DU chỉ đổi đúng chỗ, §9 khớp từng ký tự câu gốc `d70ebc8`, đủ 4 điều kiện D-V3-06 | **V2-01 (S2):** `docs/05-master-roadmap.md` và `docs/04-core-mvp.md` vẫn tự ghi roadmap hiện hành ở `docs/v3/`; V2-03 mã P1–P7 (mặt phẳng) trùng mã nguyên lý; V2-04 ghi chú tương thích chưa nguyên văn; V2-05, V2-06 | `efc2cad`: dòng khung ở 05, câu "hướng tổng thể" ở 04; câu phân biệt mặt phẳng / nguyên lý; trích nguyên văn câu sứ mệnh cũ (`3c51bb0`); con trỏ roadmap ở đầu STATUS; bổ sung định danh ở CONVENTIONS |

**Soát riêng bản viết lại `docs/01-architecture.md`** (1 người soát, sonnet, chỉ đọc), trên `9da639c`: 0 S1, 5 S2, 10 S3; 187 tham chiếu được kiểm; các khẳng định mạnh ("0 lời gọi model", "không ai đọc", "5/12 route", "chỉ trong RAM") đúng với mã; mọi mục "chưa có" đúng là chưa có.

| Mã | Mức | Phát hiện | Đã làm (`903623d`) |
|---|---|---|---|
| KT-01 | S2 | dẫn ADR-0004, tệp không tồn tại | căn cứ là `CLAUDE.md` §19 và roadmap §6 Phase 9 |
| KT-02 | S2 | luật cứng 3 "hành động lùi do pack quy định" là đích viết như đang chạy | ghi rõ mã chọn theo nhánh (`ask.go:166`, `:359`, `:465`); pack quy định: ĐÍCH |
| KT-03 | S2 | §6 ghi "hai interface Go", mã có bảy | liệt kê đủ bảy |
| KT-04 | S2 | hàng Model ở §10 nói cấu hình theo tenant | cấu hình bằng biến môi trường toàn tiến trình; theo tenant: ĐÍCH |
| KT-05 | S2 | `docs/requirements/catalog.md:31` trỏ §4.1–§4.2 đã bị xoá | giữ §4.1 (C0–C7), §4.2 (E0–E6) làm tiêu đề phụ trong L3, L5 |
| KT-07…KT-14 | S3 | "mười nguyên tắc" (thực 13), ngoại lệ của luật cứng 1, nhãn P8 / bảng thành phần, trường thiếu của `pack.yaml`, nhãn mặt phẳng ở §3, câu trạng thái luồng, bảng điểm Fuel ở L0, hai lỗi im lặng của roadmap ở L1 | sửa hết |
| KT-06 | S3 | `docs/requirements/catalog.yaml` còn đòi "4 mặt phẳng" | không sửa: tệp nhóm E |

Người soát 01-architecture tự khai một thao tác ghi: lỡ đặt `GOFLAGS=-mod=mod` làm `go.mod` đổi 2 dòng, đã hoàn tác ngay bằng `git checkout -- go.mod`. Kiểm lại sau đó: `git status` chỉ có phần đang sửa của phiên BUILD, `go.mod` không đổi.

## 6. Việc 2: viết lại luồng theo tên mới

| Tệp | Đã làm | Commit |
|---|---|---|
| `docs/ROADMAP-DAY-DU.md` | cách gọi tạm "dự án" (sản phẩm) → EAIS ở §0, §1, bảng phase 4b; bỏ câu "tới lúc đó tệp này gọi tắt là dự án"; L0–L10 giữ nguyên; "chủ dự án" (người) không đổi | `52c9399` |
| `docs/01-architecture.md` | viết lại theo 7 mặt phẳng (§2) và L0–L10 (§4); mỗi luồng: Đích → HIỆN CÓ (từng bước, `tệp:dòng` + test) → Thiếu → Phase; cây thư mục đánh dấu đích; §5–§11 thêm dòng trạng thái; giữ số mục mà mã và tài liệu khác trỏ tới | `9da639c` |
| `CLAUDE.md` §9 | câu sứ mệnh theo §1 của roadmap: câu gốc `d70ebc8` + hai chế độ sở hữu; ghi chú tương thích D-V3-06; commit riêng | `34240a9` |
| `CLAUDE.md` §11, §25 | §11 khớp L2: không sửa. §25 lệch với chính luật "target / current phải phân biệt rõ": đánh dấu 4 thư mục đích chưa có | `bff56c4` |
| README, `docs/STATUS.md`, `docs/CONVENTIONS.md` | trỏ roadmap hiện hành; STATUS: phase đang làm + khối "Việc hiện hành" theo thứ tự mới; CONVENTIONS §0: tên sản phẩm, link git, tiền tố định danh chạy | `52c9399` |
| `docs/ROADMAP.md` | một dòng khung "tên trong thân tệp là tên lúc viết" (ngoại lệ có chủ ý trong tệp nhóm E, theo §3 mục 5 của prompt); thân giữ nguyên | `730926f` |

Phần "dò hiện có" của 01-architecture do workflow 3 agent chỉ đọc (sonnet) làm tại `730926f`, mỗi agent một nhóm luồng; mã lõi không đổi từ đó tới HEAD. Bản viết lại được kiểm tự động (114 tham chiếu tệp đều tồn tại và trong phạm vi số dòng, 48 tên test đều có hàm) và soát độc lập riêng (mục 5).

`docs/05-master-roadmap.md` và `docs/04-core-mvp.md` thêm một dòng trỏ roadmap hiện hành (`efc2cad`, theo soát V2-01).

| Luồng | HIỆN CÓ / ĐÍCH | Phase |
|---|---|---|
| L0 Đưa AI vào một việc | một lát (Fuel); cổng QuânIT chưa áp tường minh | 0a, 4c |
| L1 Dữ liệu | HIỆN CÓ, còn lỗi im lặng (≈ mất cờ ước lượng, đính chính / kế hoạch, cách ly chỉ trong RAM, có thể cộng đôi tin gửi lại) | 0a, 3; nguồn DB / API / kênh: 4a, 8 |
| L2 Hỏi | HIỆN CÓ S0–S10 (xác thực, idempotency, HTTP, lưu bền: ĐÍCH) | 0b, 6; 5 |
| L3 Làm · L4 Quy trình · L5 Sự kiện | ĐÍCH — 0 dòng mã thi hành; chỉ có khai báo `EntryType`, route lỗi, ca spec dạng dữ liệu | 1, 4b · 2 · 3 — mã chờ D-RM-04 |
| L6 Tri thức | một phần: chỉ mục từ khoá trong bộ nhớ, ACL theo vai | 6, 4b |
| L7 Kiểm soát | một phần: tenant, quyền, RLS, từ chối vai vượt RLS, kiểm đầu ra; chưa có xác thực, audit, lưu bền, chi phí | 0b, 5 |
| L8 Chứng minh | HIỆN CÓ cho Fuel; bộ chấm v2 chưa đóng băng | mọi phase |
| L9 Vận hành | gần như chưa có (PG dev, migration, env) | 5, 9 |
| L10 Điều phối người–agent | đang chạy, quy trình không mã | — |

## 7. Câu cần chủ dự án quyết, giới hạn còn lại

**Câu mới trong `docs/CHO-QUYET-DINH.md` §4 (cột Trả lời để trống):**
- **D-RM-05:** thêm một dòng khung "tên lúc viết" cho `docs/study/README.md` và `docs/v3/*` không (tệp nhóm E: `docs/study/README.md:80` gọi biến DSN tiền tố cũ là "tên hiện hành"; `docs/v3` chưa có khung "tra cứu" như prompt tưởng); đường dẫn brief cũ trong tệp E có để nguyên không. Khuyến nghị: thêm khung cho `docs/study` và `docs/v3`; để nguyên đường dẫn cũ.

**Giới hạn còn lại (không chặn push):**
- Lớp bí danh của `scripts/chay-giu-lai.sh` và `scripts/du-toan-g7.py` chưa có test tự động; đã thử tay (bash `${A-${B-}}` dưới `set -u`; dự toán trên bằng chứng cũ + ca âm). `winreport -pg` đã dùng chung `pgstore.EnvDSN` (có test + đột biến PE1, PE2).
- Băm của `scripts/chay-giu-lai.sh` đổi so với băm ghi trong nhật ký chạy H3 (`docs/g7/holdout-h3/ket-qua-h3/chay.txt`): bản công cụ đã chạy H3 là bản ở `ca62760` (`git show ca62760:scripts/chay-giu-lai.sh | sha256sum` ra đúng băm ghi trong nhật ký, theo đo của người soát lượt 2). H4 sẽ ghi băm mới do chính script tính.
- Cổng danh tính dò tên dài trên bản chỉ giữ chữ / số: một từ có chuỗi "eacl" (ví dụ treacle) sẽ bị đếm oan; gặp thì ghi mục FALSE_POSITIVE (đã ghi trong comment của cổng).
- `docs/requirements/catalog.yaml` (nhóm E) còn câu kiểm "§2 mô tả đủ 4 mặt phẳng" và "câu 'RAG là một capability' ở §1" không còn khớp 01-architecture; không sửa vì là sổ yêu cầu lịch sử.
- Vai `eacl_app` vẫn còn trong cụm PG (vai là của cả cụm; 0002 chỉ gỡ quyền trên bảng của database đã migrate, không xoá vai).
- Tài liệu lệch mã do workflow dò tìm ra, thuộc Việc 3 mục 7 (chưa sửa ở đợt này):
  - `docs/ROADMAP-DAY-DU.md`: "6 định dạng" (mã: 5); `pgstore.go:163` (hàm ở 164, ACL ở 175); L7 liệt kê "audit" không kèm "(chưa có)"; P7 "gần như trống" bỏ sót bộ chạy migration, đọc env, ánh xạ lỗi HTTP; L0 ghi Fuel trượt câu 1, 3, 8, 9 nhưng bỏ câu 2 (đạt) và 7 (một phần) của chính §7; §5 và §6 gán phase cho L1, L6 không khớp nhau; ví dụ "máy 07 → EX-07 / RL-07" không tái lập được bằng seed của repo;
  - `evals/cases.go:1-4`: chú thích gói nói "chưa chạy ca nào qua pipeline" — đúng cho bộ ca 23 khoá, lỗi thời với runner WIN.
