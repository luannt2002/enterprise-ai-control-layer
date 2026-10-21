# Migration danh tính EACL — 21/10/2026

Báo cáo theo đặc tả migration của chủ dự án (§28). EACL = Enterprise AI Control Layer; repo `github.com/luannt2002/enterprise-ai-control-layer`.

## Mốc

| | |
|---|---|
| START HEAD | `096fc1b` (= origin/main lúc bắt đầu) |
| END HEAD | commit chứa báo cáo này (cha: `9dede54`) |
| CANONICAL REMOTE | `git@github-luannt2002:luannt2002/enterprise-ai-control-layer.git` — chủ dự án đặt khi clone lại; `git ls-remote origin` trả đúng HEAD đã push. Không sửa remote. |

## Thay đổi

| Bề mặt | Đổi gì | Commit |
|---|---|---|
| Module / import | `go.mod` module `github.com/luannt2002/enterprise-ai-control-layer`; 77 tệp Go đổi import. `go mod tidy` KHÔNG áp: nó đổi đồ thị phụ thuộc có từ trước (thêm 2 phụ thuộc gián tiếp của test, đánh dấu `fpdf` / `excelize` là trực tiếp) — việc riêng, không thuộc đổi danh tính | `88f4dcc` |
| Biến môi trường | `EACL_MODEL_*` (9 biến model), `EACL_PG_ADMIN_DSN`, `EACL_PG_DSN` | `88f4dcc` |
| Tương thích | MỘT hàm `models.modelEnv`: biến `EACL_MODEL*` đã đặt (kể cả rỗng) luôn thắng; không đặt thì đọc biến cùng tên mang tiền tố trước ngày 21/10 (`legacyModelEnvPrefix`; khoá model của môi trường chạy được cấp theo tên đó). DSN PG chỉ dùng ở dev: đổi hẳn; test PG BÁO LỖI (không bỏ qua lặng lẽ) khi môi trường chỉ còn DSN tiền tố cũ | `88f4dcc` |
| Database / GUC | GUC `eacl.tenant_id`, vai `eacl_app`, database `eacl` và `eacl_giu_lai` (chạy bộ giữ lại). Sửa tại chỗ migration `0001_win.sql`: dữ liệu PG chỉ ở tmpfs, mỗi lần chạy bắt đầu sạch, không DB bền nào đã áp. Container dev dựng lại | `88f4dcc` |
| Docker | compose project `eacl`, container `eacl-pg`, `POSTGRES_DB: eacl`, healthcheck | `88f4dcc` |
| Định danh mã | harness G7 (`EACLPass`, `EACLOnly`, `eaclAsker`, …), `package eacl_test`, chú thích gói | `88f4dcc` |
| Script | `scripts/chay-giu-lai.sh` (DSN, database `eacl_giu_lai`), danh mục đột biến theo chuỗi mới | `88f4dcc`, `9dede54` (MS8) |
| CI | repo không có CI (`.github/` không tồn tại) | — |
| Tài liệu | README (định nghĩa chuẩn, mô tả ngắn, North Star, trạng thái thật 21/10), kiến trúc, 02–04, STATUS, CONVENTIONS, ops (clone, biến, DSN), G7, ADR (thuật ngữ; quyết định không đổi), tenant, v3 (D-V3-03 / TASK-R0-03 / R0-05 đã làm; C1 đã giải), CHO-QUYET-DINH (biến khoá) | `35c8e61` |
| Hướng dẫn agent | `CLAUDE.md` §9 định nghĩa EACL + tên ở mọi mục; commit riêng theo D-V3-06. Memory vận hành repo (máy) đổi theo | `3c51bb0` |
| Cổng danh tính | `danh_tinh_test.go`: mọi chỗ còn tên trước 21/10 phải có lý do + không vượt số lần cho phép; ca âm; đột biến n42. Sau soát: hạn mức đúng từng tệp | `fb5319e`, commit này |
| Sửa sau soát độc lập | xem mục "Soát độc lập" | `9dede54`, commit này |
| Tệp / thư mục | không có tệp / thư mục nào mang tên cũ; thư mục máy do chủ dự án clone lại thành `enterprise-ai-control-layer` | — |

## Giữ nguyên có chủ đích

| Nhóm | Lý do |
|---|---|
| `docs/g7/holdout/**`: bộ giữ lại, manifest | IMMUTABLE_HASHED_ARTIFACT: `dong-bang-giu-lai.py --kiem` = KHỚP 189 ca sau migration |
| `docs/g7/holdout/ket-qua-h1/chay.txt` (ghi tên database lúc chạy H1) | HISTORICAL_EVIDENCE: đầu ra gốc của lần chạy duy nhất; không thuộc manifest băm, vẫn không sửa |
| Báo cáo 20/10 và 20→21/10, `docs/g7/REVIEW-D5.md`, `docs/g7/CONTAMINATION-AUDIT.md`, `docs/ops/NHAT-KY-2026-10-20.md` | HISTORICAL_EVIDENCE: ghi đúng tên tại thời điểm đó; thêm nhãn "Historical artifact" (chỉ siêu dữ liệu, nội dung không đổi) |
| `docs/requirements/**`, `docs/05-master-roadmap.md`, `docs/v3/00-CONTEXT-AUDIT.md` | HISTORICAL_EVIDENCE (nhãn "Historical artifact" ở đầu tệp); tên tệp brief nguồn là EXTERNAL_REFERENCE |
| mục CHANGELOG 19–20/10; văn bản quyết định D-V3-00, D-V3-10 | HISTORICAL_EVIDENCE |
| `models.modelEnv` + test, chốt DSN trong `pgstore_test.go` | COMPATIBILITY_ALIAS (`pgstore_test.go` chỉ chặn, không đọc biến cũ) |
| mẫu grep trong tiêu chí nghiệm thu TASK-R0-03 / R0-05; chuỗi đột biến n42; chính mẫu của cổng | FALSE_POSITIVE |

Danh sách đầy đủ, có số lần tối đa từng tệp: `legacyAllowed` trong `danh_tinh_test.go` — cổng chạy trong `go test ./...`.

## Kiểm

| Kiểm | Kết quả |
|---|---|
| `go build ./...`, `go vet ./...`, `gofmt -l .` | sạch |
| `go test ./... -count=1` có PG (container `eacl-pg` mới dựng) | lúc migration: 438 đạt / 0 trượt / 0 bỏ qua trước cổng danh tính (431 của mốc `15ed135` + 7 test mới của ca). Đo lại tại `9dede54`: **440 / 0 / 0** (317 test cấp cao); không PG: 435 + 5 bỏ qua (5 test PG) |
| Bộ 117 ca (`winreport`) | 117/117 |
| H1 làm bộ hồi quy, bộ nhớ + PG | 89/104 cả hai đường; 0 ca đổi nhãn so với mốc `016422e` |
| `scripts/scale-smoke.sh` | ĐẠT; 117 ca giống từng byte giữa hai hồ sơ; G1 smoke 359.945/359.945 |
| Tenant / RLS | test PG (`TestWinTrenPG`, GUC đọc lại, từ chối khi thiếu tenant, WITH CHECK) xanh trên vai `eacl_app`; danh mục `w4-pg-rls` bị bắt hết |
| Đột biến: 33 danh mục cũ trên clone `88f4dcc` | 589 bị bắt / 0 sống / 0 không biên dịch / 1 chuỗi gốc lệch: MS8, vì `modelEnv` đổi chỗ đọc biến. Sửa chuỗi ở `9dede54`; chạy lại `n7` trên `9dede54`: 13/13 bị bắt, kể cả MS8 |
| Cổng danh tính + biến tương thích (n42) | 8/8 bị bắt |
| Tổng | 598 đột biến bị bắt, 0 sống |

## Soát độc lập

Một agent riêng, chỉ đọc, làm trên clone sạch `fb5319e`. Kết quả: không có S1 (không hỏng build, không lệch băm đóng băng, không lệch hành vi). Phát hiện và cách xử lý:

| # | Mức | Phát hiện | Xử lý |
|---|---|---|---|
| 1 | S2 | chuỗi gốc MS8 không còn khớp | sửa ở `9dede54`, chạy lại bị bắt |
| 2 | S2 | không tài liệu nào nối tên cũ với EACL, trong khi 4 tài liệu lịch sử không nhãn vẫn hiện tên cũ | **không** thêm câu ánh xạ: đặc tả §0 của chủ dự án cấm kiểu "tên cũ → tên mới" trong tài liệu hiện hành, lịch sử do git giữ. Làm đúng cách đặc tả cho phép: nhãn "Historical artifact" cho 4 tài liệu đó |
| 3 | S2 | mốc "224 PASS" trong `docs/ops/README.md` đã cũ | đo lại tại `9dede54`, ghi 440 / 435 + 5 |
| 4 | S2 | `docs/01-architecture.md`: định nghĩa mới (ứng dụng giữ workflow) mâu thuẫn thân tài liệu (EACL tự có workflow engine) | thêm ghi chú phạm vi V3 ở đầu tệp: phần workflow / command / event là kiến trúc đích V2, đang hoãn |
| 5 | S2 | STATUS / BO-NHO còn dòng 20/10 mâu thuẫn số 21/10 | gắn nhãn ngày, sửa "G6' ĐẠT" thành PARTIAL, mục đêm "đã chốt", danh sách việc 20/10 "đã thay" |
| 6 | S2 | README "đang sửa" ≠ STATUS "CHƯA sửa, chờ D-OP-10" | đồng bộ: D-OP-10 đã chốt 21/10 (mục 0 của `CHO-QUYET-DINH.md`); mã chính CHƯA sửa |
| 9 | S3 | danh sách cho phép: 1 mục chết; thư mục cho phép không giới hạn; nhãn sai | hạn mức đúng từng tệp, bỏ mục chết, sửa nhãn. Cổng siết lại bắt ngay 2 chỗ tên cũ trong bản nháp báo cáo này (đã viết lại) |
| 11 | S3 | thiếu cách dựng lại DB đã áp migration cũ | thêm câu dựng lại container ở `docs/ops/README.md` §3. Cách tạo `eacl_giu_lai`: ghi khi chạy H3 (đo thật trước khi ghi) |
| 12 | S3 | README thiếu repo / module, không định nghĩa Fuel WIN / H1 | thêm |
| 14 | S3 | ADR-0006 §10 đổi tên database sau khi đăng ký | ghi ở CHANGELOG (chỉ đổi tên, luật không đổi) |
| 7, 8 | S3 | lệnh đo ở `docs/v3/02` mang tên biến hiện hành (khác lúc đo); AC của TASK-R0-03 / R0-05 trong backlog không còn tái lập | để lại: là bản ghi tiêu chí lúc đó; ghi việc sau |
| 10 | S3 | bí danh biến model chỉ có trong mã | để lại: đặc tả yêu cầu bí danh cô lập, "deprecation clearly internal" |
| 13 | S3 | khác chữ trong câu định nghĩa; `CLAUDE.md` §24/§25 chưa sửa | chờ chủ dự án (sửa `CLAUDE.md` cần duyệt) |
| 15 | S3 | cổng quét cả tệp chưa commit | giữ: bắt tên cũ ngay ở bản nháp, trước khi commit |

Các sửa ở commit này CHƯA được người soát độc lập kiểm lại; bằng chứng hiện có là lệnh tự chạy (bảng "Kiểm").

## Hồi quy

- Do migration: không thấy (các số trên trùng mốc).
- Cổng danh tính lần chạy đầu bắt 2 chỗ: CHANGELOG có 3 lần tên cũ (ước 2) và chính danh mục đột biến n42 — sửa danh sách lý do, không sửa nội dung lịch sử.

## Sự cố liên quan

Lúc 09:14, thư mục làm việc cũ bị xoá khi chủ dự án clone lại repo dưới tên mới. Mất các thứ chỉ có ở máy (một commit chưa push, tệp git bỏ qua, nhánh cục bộ, bản nháp bộ giữ lại đang soạn). Đã dựng lại từ nhật ký phiên và từ git; mất thật duy nhất là 21 tệp đầu ra thô của một lần đo đêm 20/10 (số liệu chốt vẫn ở báo cáo đêm).

## Kết luận

**YES.** Mã, cấu hình, dữ liệu dev, script và tài liệu hiện hành dùng một danh tính EACL; cổng danh tính chặn tên cũ quay lại (hạn mức từng tệp, có ca âm, đột biến 8/8); hành vi không đổi.
- Điều kiện: người soát độc lập chấm cổng "không còn danh tính cũ không giải thích được" là ĐẠT khi có câu ánh xạ hoặc khi chủ dự án coi nhãn + lý do trong `legacyAllowed` là đủ. Bản này chọn nhãn theo đặc tả §0.
- Còn mở (tài liệu, không chặn): S3 số 7, 8, 13.
