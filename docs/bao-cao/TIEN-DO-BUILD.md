# Tiến độ BUILD — tệp sống, cập nhật mỗi lần push

Khuôn theo `docs/PROMPT-BUILD-EAIS.md` §7:
- phần "Bây giờ" ghi đè mỗi lần push;
- phần "Nhật ký" chỉ thêm dòng, không xoá dòng cũ.

## Bây giờ (ghi đè mỗi lần)

- **Cập nhật lúc:** 21/10/2026 20:2x · **HEAD:** `4e18680` (cộng commit chứa tệp này). Mạng đứt 19:55 → 20:22 nên các commit từ `945777c` lên `origin/main` cùng một lần push.
- **Xong từ lần trước** (PROMPT-DOI-TEN-VA-LUONG-MOI):
  - Việc 1, đổi tên lại EAIS (D-RM-02): `945777c` (cổng danh tính đảo chiều), `423e4d3` (mã + định danh chạy + migration 0002), `730926f` (tài liệu + `docs/PROMPT-BUILD-EAIS.md`);
  - soát độc lập 2 lượt (workflow, sonnet, chỉ đọc): lượt 1 — 0 S1, 3 S2; lượt 2 — 0 S1, 1 S2 (Việc 2); soát riêng 01-architecture — 0 S1, 5 S2. Đã sửa hết S2 và các S3 rẻ: `dbdf7ca`, `7738a3d`, `cdf7d84`, `efc2cad`, `2b8fb94`, `903623d`;
  - Việc 2: `52c9399` (ROADMAP-DAY-DU dùng tên EAIS; README / STATUS / CONVENTIONS), `34240a9` (CLAUDE.md §9, commit riêng), `bff56c4` (§25), `9da639c` (01-architecture theo 7 mặt phẳng + L0–L10);
  - câu mở mới D-RM-05 (`93649ba`).
- **Xong thêm:** tệp phản hồi `docs/bao-cao/2026-10-21-phan-hoi-doi-ten.md` (khuôn §6, `1ccac76`); hai chú thích lệch mã (`4e18680`, brief §4 mục 7).
- **Đang làm:** Việc 3 mục 2 — đo lại thiết kế R4 bằng workflow (dựng lại corpus từ transcript của lượt phân tích bị dừng, soát mã bản diff, đo trên corpus + 117 / H1 / H3); chưa sửa mã cổng phán quyết.
- **Tiếp theo:**
  1. Việc 3 mục 2: đóng R4-2…R4-4 của cổng phán quyết (đo lại thiết kế V4 / V5 trước khi tin) → vòng S4 (ADR-0010) → H4 MỘT lần;
  2. Việc 3 mục 3–7: phép đo, giấy Phase 0a / 1, tài liệu lệch (danh sách trong tệp phản hồi mục 7).
- **Bị chặn:**
  - mã COMMAND / WORKFLOW / EVENT: chờ D-RM-04;
  - dữ liệu / người nghiệp vụ thật: chờ D-RM-03;
  - khung "tên lúc viết" cho `docs/study`, `docs/v3`: chờ D-RM-05;
  - W-G7 trả tiền: chờ khoá, D-G7-18 và câu cho phép chi.
- **Test** (`903623d`, container `eais-pg`): `go test -count=1 ./...` có PG 455 / 0 / 0; không PG 448 + 7 SKIP (7 test PG). So theo tên với trước đổi tên (`ca62760`, 453): 0 đổi kết quả, 4 cặp đổi tên, 2 test mới. winreport bộ 117 / H1 / H3, hai đường: 0 dòng khác ngoài số đo thời gian (đo ở trạng thái `423e4d3`; mã lõi không đổi tới HEAD). Đột biến: toàn bộ 35 danh mục 676/676 (trạng thái `423e4d3`); n42 25/25 và w4-pg-rls 26/26 ở HEAD.
- **Handoff / prompt đang mở:**
  - `PROMPT-DOI-TEN-VA-LUONG-MOI.md`: Việc 1 + Việc 2 XONG, tệp phản hồi đang viết; Việc 3 chưa bắt đầu;
  - `docs/debug-handoff/2026-10-21-01-…`: NHẬN;
  - `PROMPT-AUDIT-ROADMAP.md`: bản giữa chừng.
- **Ca tối 21/10** tới khoảng 22:00 theo lời chủ dự án; ca ngày 22/10 bắt đầu 09:00 (cron của phiên).

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
