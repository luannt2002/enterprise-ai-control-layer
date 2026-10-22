# Tiến độ BUILD — tệp sống, cập nhật mỗi lần push

Khuôn theo `docs/PROMPT-BUILD-EAIS.md` §7:
- phần "Bây giờ" ghi đè mỗi lần push;
- phần "Nhật ký" chỉ thêm dòng, không xoá dòng cũ.

## Bây giờ (ghi đè mỗi lần)

- **Cập nhật lúc:** 22/10/2026 14:4x · **HEAD:** `f8b295e` (cộng commit chứa tệp này). Chủ dự án xin push phần đã làm để đọc.
- **Xong từ lần trước:**
  - soát S4 vòng 3 (vòng cuối theo §9.6): KHÔNG ĐẠT có điều kiện, 0 S1, không lỗi thuộc ngoại lệ ⇒ **mã S4 ĐÓNG BĂNG ở `a665298`**; nợ ghi ở
    ADR-0010 (F6' số tiền xa "dầu", S4R-03, hai thực thể khác loại…); thêm test cho nửa "chỉ số rỗng" (`f8b295e`, không đổi mã);
  - soát khoá vòng 3: bản sửa vòng 2 tự nó đạt, nhưng còn 2 lỗ SAI CHIỀU có từ trước ⇒ mở lại theo ngoại lệ §9.6, sửa `be91cd0`;
  - làn 3: giấy Phase 1 lên main (`2b80a38`).
- **Đang làm:** soát khoá vòng 4 (ngoại lệ sai chiều) cho `be91cd0`; làn 0 (M1: `cmd/eais` + vết) đang soát U0.2 trên nhánh riêng; làn 3 viết
  giấy Phase 0a trên nhánh riêng.
- **Tiếp theo:**
  1. Soát khoá vòng 4 kết luận (chỉ sửa tiếp nếu còn phán quyết sai chiều).
  2. H4 chạy MỘT lần (`scripts/chay-giu-lai.sh`, database `eais_giu_lai` đã tạo; băm tệp ca khớp bản đóng băng) → báo cáo theo tầng sai đầu
     tiên, có đối chứng mốc `7d72e33` → H4 thành bộ hồi quy.
  3. Gộp làn 0, làn 3 (Phase 0a) sau soát; sửa 4 ghi chú S3 của giấy Phase 1 trước khi mở mã Phase 1.
- **Bị chặn:**
  - mã Giai đoạn II (COMMAND / WORKFLOW / EVENT, gồm M4 tạo phiếu): mở ngay sau khi chạy xong H4 (D-RM-04 = A);
  - dữ liệu / người nghiệp vụ thật: không có (D-RM-03) — mọi kết luận ghi "trên dữ liệu giả lập, chưa có người nghiệp vụ duyệt";
  - W-G7 trả tiền: chưa cấp;
  - D-OP-16, D-OP-17: chờ chủ dự án (mặc định đang chạy: (a), coi "2k" như cũ).
- **Test** (`f8b295e`, container `eais-pg`): `go vet ./...` 0 dòng; `gofmt -l` 0 tệp; `go test -count=1 ./...` có PG: 479 pass, so theo tên với
  `a665298`: 0 đổi, 0 mất, 3 test mới; winreport 117 / H1 / H3 hai đường: 0 dòng khác (117/117, H1 92/104, H3 113/134).
- **Handoff / prompt đang mở:** `docs/PROMPT-BUILD-EAIS.md` §9 (bảy làn, MVP trước); PROMPT-DOI-TEN Việc 3 còn mục 3–7; PROMPT-AUDIT-ROADMAP còn
  Q2–Q7, Q9–Q13, Phụ lục C.
- **Ca ngày 22/10** 09:00 → 17:00 (cron của phiên). Báo cáo ca: `docs/bao-cao/2026-10-22-ca.md`.

## Nhật ký (thêm dòng, không xoá)

| Lúc | Commit | Việc | Bằng chứng (lệnh → kết quả) | Ghi chú |
|---|---|---|---|---|
| 21/10 trước 14:00 | `e28f8d5` … `8280f40` | Ca ngày 21/10: migration EACL; H3 đóng băng, chạy một lần; cổng phán quyết T2 → T2.2 | Xem `docs/bao-cao/2026-10-21-ca-ngay.md` | Trước khi có tệp này |
| 21/10 14:19 | `ac98cf4` | D-OP-13 ghi vào `CHO` mục 0 | Quét riêng tư 0 mẫu; cổng danh tính ĐẠT | Lời chốt của chủ dự án |
| 21/10 14:4x | `f38b4eb` | Phản hồi audit ROADMAP, bản giữa chừng | K6–K15: 6 XÁC NHẬN, 3 MỘT PHẦN, 1 BÁC. 9 test có sẵn chạy lại: 8 ĐẠT, 1 SKIP (không PG) | Người kiểm độc lập; phiên BUILD đo lại |
| 21/10 14:4x | `a299f06` | Nhận handoff 01; sửa giờ ghi D-OP-13 | — | Chưa sửa mã |
| 21/10 14:4x | `3185eb8` | Tạo tệp tiến độ | `go vet` 0 dòng; `gofmt -l` 0 tệp; `go test -count=1 ./...` có PG: 451/0/0 | — |
| 21/10 14:5x | `15f04f9` | Đóng băng H4 bằng cam kết băm (ADR-0006 §10; mục đích theo D-OP-13) | `cam-ket-giu-lai.py` → sha256 `7fc5a485…8150d`, 119 ca; `kiemca` 0 lỗi; kiểm toán nhiễm 7 nhật ký: 0 đường dẫn thật ra ngoài phạm vi | routing 11 < 13: chấm và báo đúng số, không độn |
| 21/10 15:0x | `c366036` | Cổng danh tính bỏ qua checkout git lồng; `.gitignore` bỏ qua `.claude/worktrees/` | Ca âm đỏ trước khi sửa; n42 trong gương: 11/11 bị bắt, 0 lệch (DT3 cập nhật theo mã mới) | Cổng từng đỏ do môi trường (34 vi phạm ở bản chép) |
| 21/10 15:0x | `cc84b83` | `pgstore.Open` từ chối vai vượt RLS (K7) | Ca âm đỏ trước khi sửa; `w4-pg-rls` trong gương: 9/9 bị bắt (bản đầu BR3 SỐNG ⇒ thêm vai SUPERUSER NOBYPASSRLS); vai tạm sót 0 | Bản đầu sót 2 vai tạm (defer trước Cleanup), đã xoá |
| 21/10 15:0x | `49c841a` | ADR-0010 ĐỀ XUẤT (S4) | — | Chưa áp mã |
| 21/10 15:0x | `2f1b8c2` | Cập nhật tiến độ, push | `go test -count=1 ./...` có PG 452/0/0 | — |
| 21/10 15:1x | `01d4348` | Cổng phán quyết T2.3: sửa hồi quy S1 của dòng vá (ii) và S3 của (iv) T2.2 (soát lần ba) | Test đỏ trước (`TestCongPhanQuyetSoatLan3`, 6 ca `TestVerdictGap`); n43 47/47 bị bắt (CG30 lần đầu không biên dịch ⇒ viết lại); 117: 117/117, H1 89/104 hai đường, H3 109/134 — 0 đổi nhãn / 0 đổi câu trả lời | Còn mở: S1 dạng gương, S2 dấu câu (có từ T2.1) |
| 21/10 15:1x | `87b07fa` | Cổng danh tính: bỏ luật theo mục `.git` (soát độc lập: S1) | Ca âm đỏ trước (`stale/f.md`); n42 11/11 bị bắt | — |
| 21/10 15:2x | `1973d2b` | Cập nhật tiến độ, push | `go test -count=1 ./...` có PG 453/0/0 | — |
| 21/10 15:2x | `8a506b8` | ADR-0009 Vòng 3 | — | — |
| 21/10 15:28 | `a853e96` | E3: đóng S1 dạng gương, S2 dấu câu | Test đỏ trước (8 câu "Đúng."); n43 49/49 (CG30 từng sống ⇒ ghim 2 câu mệnh đề hỏi đứng trước số); bộ 117 / H1 / H3 0 đổi; test có PG 453/0/0 | Đang soát độc lập |
| 21/10 15:32 | `87b304d` | pgstore: vai tạm NOLOGIN qua `role=`; ca dương `current_user` (sau soát BYPASSRLS) | `w4-pg-rls` 10/10 (BR4 mới); vai sót 0; test có PG 453/0/0 | Soát: ĐẠT CÓ GHI CHÚ |
| 21/10 15:4x | `9a1d8b0` | Tài liệu chốt ca (báo cáo, STATUS, CHANGELOG, ADR-0009, tiến độ) | Cổng danh tính ĐẠT | Bản trước khi có kết quả 2 lượt soát cuối |
| 21/10 15:55 | `936d090` | Sửa sau soát T2.3 (R4-1 "1,2k5"); ghim E3 (soát: ĐẠT CÓ GHI CHÚ) | Test đỏ trước; n43 53/53; bộ 117 / H1 hai đường / H3: 0 đổi; test có PG 453/0/0 | Còn mở R4-2…R4-4 (có từ T2.1) |
| 21/10 16:0x | `2e4897b` | Chốt ca: ADR-0009 Vòng 4, báo cáo ca bản cuối, STATUS, CHANGELOG, tiến độ | Cổng danh tính ĐẠT | Ca 21/10 kết thúc |
| 21/10 16:4x | `d6afad7` | Chốt ca lần cuối: mục sự cố 09:14 trong báo cáo, đo cuối, ghi nhận phân tích thiết kế R4 bị DỪNG giữa chừng | `go test -count=1 ./...` có PG 453/0/0; 117 / H1 / H3 không đổi | Lịch ca xoá sau commit này |
| 21/10 18:1x | `ca62760` | NHẬN PROMPT-DOI-TEN-VA-LUONG-MOI (HEAD `32c8cfe`) | — | Bắt đầu Việc 1 (kiểm kê trước khi sửa) |
| 21/10 19:5x | `945777c` … `903623d` (14 commit) + commit chứa dòng này | Việc 1 đổi tên lại EAIS (D-RM-02) + sửa sau 3 lượt soát độc lập; Việc 2 (ROADMAP-DAY-DU, 01-architecture, CLAUDE.md §9 / §25, README / STATUS / CONVENTIONS); D-RM-05 | vet sạch; gofmt 0; test có PG 455/0/0 (theo tên: 0 đổi, 4 cặp đổi tên, 2 mới); winreport 5 đường 0 dòng khác; đột biến 676/676, n42 25/25, w4 26/26; cổng danh tính xanh ở cây của TỪNG commit; quét riêng tư 0 dòng nghi | Tệp phản hồi theo khuôn §6: commit kế tiếp |
| 21/10 20:2x | `1ccac76`, `4e18680` + commit chứa dòng này | Tệp phản hồi PROMPT-DOI-TEN (khuôn §6); chú thích lệch mã; push cả chuỗi sau khi mạng có lại | vet sạch; gofmt 0; test có PG 455/0/0 (cùng tên); cổng danh tính xanh ở cây của từng commit mới; quét riêng tư 0 dòng nghi | Mạng đứt 19:55 → 20:22 |
| 21/10 21:1x | commit chứa dòng này | Việc 3 mục 2: đo lại bản thử R4 (V5) của cổng phán quyết; không đổi mã | Bộ câu dựng lại khớp md5; chấm lại từ đầu ra thô: đóng 25.669 / 25.669, 0 đổi chiều, 0 phán quyết mới; fuzz 10.774 / 10.774; 117 / H1 / H3 0 dòng khác; go test không PG 455 theo tên, 0 đổi | Chưa áp: thiếu test bảo vệ nhánh mới, lỗ ngày dạng số, chặt hơn luật thử R4-4. Hồ sơ đo chỉ ở máy BUILD |
| 21/10 21:2x | commit chứa dòng này | Soát mã độc lập bản thử R4 (V5); không đổi mã | Agent chỉ đọc: 17 đầu vào xấu, 0 crash; tệp phủ: 16 / 57 khối mã mới không test nào chạy. BUILD đo lại R-1 bằng binary test dựng sẵn: "… ERP, 3900 L đúng không?" → "Không." ở HEAD và V5 | R-1 cùng cơ chế R2-4 của ADR-0009 (theo người soát); sai chiều nên ghi thành lỗ S1, đóng trước H4 |
| 21/10 21:3x | commit chứa dòng này | K1–K5 của phản hồi audit ROADMAP (tệp mới `2026-10-21-phan-hoi-roadmap-k1-k5.md`); không đổi mã | Bơm một dòng vào bản sao seed 42 + winreport, có đối chứng; 2 người đo + 2 người đo lại độc lập: đồng ý 5 / 5; BUILD đo lại bằng binary, dữ liệu, tệp ca riêng: K4 lu 02 1.200 → 2.400 L; "… lu 02 một nghìn hai lít … đúng không?" → "Không."; "hai muoi lit" 11 / 20 lần 10 L, 9 / 20 lần 20 L | Hai phát hiện mới: lỗ S1 đổi chiều (K5 trong miền) và bảng không dấu không tất định |
| 21/10 21:3x | commit chứa dòng này | Câu mở D-OP-14 vào `CHO` mục 3 (cột Trả lời để trống) | Dạng mơ hồ trong bộ 117 / H1 / H3, seed, test: 0 (grep; 4 chỗ khớp thừa soát tay đều là số đầy đủ: "một nghìn năm trăm", "ba trăm năm chục") | Khuyến nghị a; chưa quyết thì c |
| 21/10 22:0x | commit chứa dòng này | Chốt ca tối: STATUS, CHANGELOG, tiến độ | Cổng cuối ở `25a73bf`: vet 0; gofmt 0; test có PG 455 / 0 / 0, so theo tên với `ca62760`: 0 đổi, 4 cặp đổi tên, 2 mới; winreport 117 / H1 / H3 hai đường 0 dòng khác mốc | Ca tối kết thúc; ca ngày 22/10 bắt đầu 09:00 |
| 22/10 09:0x | commit chứa dòng này | NHẬN ca ngày 22/10 (09:00 → 17:00), HEAD `ffb1efb` | — | Không có PROMPT / handoff / câu trả lời CHO mới từ tối qua |
| 22/10 09:1x–10:5x | `03c2c54`, `489c98a` | Cổng phán quyết Vòng 5: đóng R4-2…R4-4, R5-1, R5-2 | Test đỏ trước mỗi vòng; test có PG 459/0/0 (455 cũ theo tên); 713.097 câu 0 đổi chiều / 0 PQ mới; 117 / H1 / H3 hai đường 0 dòng khác; n43 210/210; soát 3 vòng | Vòng 3: còn R5-4, R5-5 (cùng họ, có từ trước) |
| 22/10 11:0x | commit chứa dòng này | ADR-0009 Vòng 5; STATUS; CHANGELOG; câu mở D-OP-15 (điều tra S4) | Cổng danh tính xanh; quét riêng tư | Chủ dự án xin push để đọc mã |
| 22/10 12:1x | `962ae9c`, `1666099`, `297a934` + commit chứa dòng này | Cổng phán quyết Vòng 5: đóng R5-4, R5-5 (soát độc lập vòng 4: ĐẠT CÓ GHI CHÚ, thêm ca ghim V5S4-01); `CHO` ghi lời chủ dự án 22/10 (D-OP-15 a2 + 2 điều kiện, D-OP-14 a, D-RM-05, D-RM-04 A, D-RM-03, W-G7 chưa cấp); ADR-0009 cập nhật. Nhận `docs/PROMPT-BUILD-EAIS.md` §9 (MVP M1–M5): làn 0 (M1) đã phóng | Test đỏ trước trên `c086fe7`; test có PG 459 / 0 / 0; 117 / H1 / H3 hai đường 0 dòng khác; 713.097 câu 0 phán quyết mới / 0 đổi chiều; n43 215 / 215 + CG211 | Còn sai chiều cùng họ (V5S4-02…04) ⇒ bước KHOÁ (§9.6) làm tiếp; D-OP-14 đã sửa ở máy, chờ soát |
| 22/10 13:1x | 7 commit làn 2 (`6e670da` … `649f5b7`) + commit chứa dòng này | LÀN 2 (M5): canary 2 tenant có đối chứng dương (`winreport -canary`, K6) và G1 so cờ ước lượng (K1); sửa theo soát R1–R7 (ratchet ghim chiều lệch, scale-smoke đọc tệp ghim, từ chối `-canary -pg` trước mọi tác dụng phụ). Soát độc lập: ĐẠT CÓ GHI CHÚ | Điều phối đo lại trên origin/main: vet 0; gofmt 0; test có PG 469 / 0 / 0 (459 cũ theo tên không đổi + 10 mới); 117 / H1 / H3 hai đường: chỉ dòng G1 khác (2.736/2.739 + 3 dòng lệch đã biết), mọi dòng ca / câu trả lời giống hệt; scale-smoke ĐẠT; n36 12/12; canary bộ 117: 0 rò rỉ | 3 lệch cờ ước lượng: R-003 "≈" chờ làn 1, R-001/R-002 chờ D-OP-17. Sự cố quy trình: người soát xoá nhầm tệp tạm của tác giả (trùng tiền tố) — không ảnh hưởng nhánh |
| 22/10 13:5x | `9c5b14f` … `a665298` (9 commit) + commit chứa dòng này | Làn 1 (M3): D-OP-14; KHOÁ cổng phán quyết + sửa theo 2 vòng soát; vòng sửa S4 (a2 + 2 điều kiện) + sửa theo 2 vòng soát đối kháng; D-RM-05; câu mở D-OP-16, D-OP-17 | Test có PG 476 pass, theo tên 0 đổi / 0 mất; 117 / H1 / H3 hai đường 0 dòng khác; corpus 716.133 câu + E3 + lưới của người soát: 0 phán quyết mới / 0 đổi chiều / 0 mất so với `7e1d8ca`; 0/700 câu (ii); đột biến n45 46/46, n44 13/13 (n43 180/180 ở `7e1d8ca`, đang chạy lại ở `a665298`) | Chủ dự án xin push; soát vòng 3 (cuối) của khoá và S4 chưa xong; H4 CHƯA chạy |
| 22/10 14:0x | commit chứa dòng này | LÀN 3: giấy Phase 1 — ADR-0011 (ĐỀ XUẤT, chưa áp mã), hợp đồng việc "tạo một phiếu kiểm tra chênh lệch", cổng QuânIT 9 câu (`docs/viec/1-hanh-dong/`) | Soát độc lập vòng 1 KHÔNG ĐẠT (L3S-01, L3S-02 S2) → sửa; soát lại ĐẠT CÓ GHI CHÚ (đo lại A25; nền PG của A02, A02b, A04b, M2, M4, M23 kiểm bằng psql). Quét: 0 đường dẫn máy, 0 tên cũ, 0 ký tự lạ; cổng danh tính xanh | Chỉ giấy, không mã. Ghi chú S3 chờ sửa: L3C-01 (A04 / M21 mâu thuẫn §7.3), L3C-02 (khuôn ca đồng thời), L3C-03 (số migration 0003 trùng làn 0), L3C-04 (nơi đặt mã lệch write-set làn 4 §9.2) |
| 22/10 14:4x | `be91cd0`, `f8b295e` + commit chứa dòng này | Làn 1: cổng phán quyết sửa theo soát khoá vòng 3 (2 lỗ sai chiều có từ trước, mở lại theo §9.6); S4 ĐÓNG BĂNG ở `a665298` sau soát vòng 3; test nửa "chỉ số rỗng" | Test có PG 479 pass, theo tên 0 đổi / 0 mất; 117 / H1 / H3 hai đường 0 dòng khác; corpus 716.133 câu + E3 + lưới: 0 phán quyết mới / 0 đổi chiều, mất 1 (nợ vòng 2); đột biến n43 187/187, n44 16/16, n45 46/46 (ở `be91cd0`) + S10, S49 bị bắt (chạy riêng ở `f8b295e`) | Chủ dự án xin push; soát khoá vòng 4 đang chạy; H4 CHƯA chạy |
