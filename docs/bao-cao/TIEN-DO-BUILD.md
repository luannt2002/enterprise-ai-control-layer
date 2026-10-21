# Tiến độ BUILD — tệp sống, cập nhật mỗi lần push

Khuôn theo `docs/PROMPT-BUILD-EACL.md` §7:
- phần "Bây giờ" ghi đè mỗi lần push;
- phần "Nhật ký" chỉ thêm dòng, không xoá dòng cũ.

## Bây giờ (ghi đè mỗi lần)

- **Cập nhật lúc:** 21/10/2026 16:4x, CHỐT CA · **HEAD:** `2e4897b` (cộng commit chứa tệp này)
- **Đang làm:** không còn gì. Ca 21/10 đã chốt; báo cáo `docs/bao-cao/2026-10-21-ca-ngay.md`.
- **DỪNG giữa chừng:** phân tích thiết kế R4-2…R4-4 (agent chỉ đo, không đẩy mã), dừng lúc chốt ca vì hết giờ.
  - Lời khai của agent, chưa đo lại: có một biến thể (V5) đóng 10.774/10.774 câu lỗ trên bộ fuzz mở rộng, 0 câu rõ nghĩa mất phán quyết, 0 câu đổi chiều.
  - V5 CHƯA chạy winreport (bộ 117 / H1 / H3), CHƯA soát độc lập.
  - Bản thử và kết quả được giữ ở máy (`docs/sources/`, không công khai).
- **Xong trong ca (chiều):** H4 đóng băng; BYPASSRLS; cổng danh tính; T2.3 → E3 → sửa sau soát (`936d090`); ADR-0009 Vòng 3–4; ADR-0010 ĐỀ XUẤT; phản hồi audit bản giữa chừng.
- **Tiếp theo (ca sau):**
  1. phản hồi audit ROADMAP cho trọn;
  2. đóng R4-2, R4-3 (S1) và R4-4 (S2), bắt đầu từ V4 / V5; TRƯỚC khi chạy H4;
  3. vòng sửa S4 theo ADR-0010, rồi chạy H4 MỘT lần.
- **Bị chặn:**
  - lượt W-G7 trả tiền: chờ khoá, D-G7-18 và câu cho phép chi;
  - code R2: chờ D-V3-09;
  - sửa `CLAUDE.md`: gộp vào đợt duyệt R0;
  - D-OP-11, D-OP-12; cột Trả lời của D-H1-3; `HOP-DONG-G7.md` §12.
- **Test cuối ca** (16:4x, `2e4897b`):
  - `go vet ./...` 0 dòng; `gofmt` trên tệp đang theo dõi 0;
  - `go test -count=1 ./...` có PG 453/0/0;
  - bộ 117 ca 117/117; H1 89/104 hai đường; H3 109/134.
- **Handoff / prompt đang mở:**
  - `docs/debug-handoff/2026-10-21-01-…`: NHẬN;
  - `PROMPT-AUDIT-ROADMAP.md`: bản giữa chừng;
  - `PROMPT-BUILD-EACL.md`: đang làm theo.

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
| 21/10 16:4x | commit chứa dòng này | Chốt ca lần cuối: mục sự cố 09:14 trong báo cáo, đo cuối, ghi nhận phân tích thiết kế R4 bị DỪNG giữa chừng | `go test -count=1 ./...` có PG 453/0/0; 117 / H1 / H3 không đổi | Lịch ca xoá sau commit này |
