# Tiến độ BUILD — tệp sống, cập nhật mỗi lần push

Khuôn theo `docs/PROMPT-BUILD-EACL.md` §7:
- phần "Bây giờ" ghi đè mỗi lần push;
- phần "Nhật ký" chỉ thêm dòng, không xoá dòng cũ.

## Bây giờ (ghi đè mỗi lần)

- **Cập nhật lúc:** 21/10/2026 14:5x · **HEAD:** `3185eb8` (cộng commit chứa tệp này: đóng băng H4)
- **Đang làm:** chốt ca 16:40. Đợi chốt thì soạn sẵn báo cáo ca, có mục "Trả lời handoff / prompt".
- **Xong từ lần trước:** **H4 đã đóng băng bằng cam kết băm** — `docs/g7/holdout-h4/holdout-manifest.json`, commit chứa tệp này.
  - Đóng băng 14:48:31 tại `3185eb8`, TRƯỚC mọi mã S4.
  - 119 ca, 82 ca có số.
  - Số ca theo lớp: representation 18, entity 16, permission 16, source_conflict 15, unsupported_answer 15, ambiguity 14, semantic 14, routing 11.
  - Nội dung ở máy và một bản sao ngoài repo. KHÔNG chạy cho tới khi mã S4 đóng băng (D-OP-13).
- **Tiếp theo:**
  1. chốt ca 16:40;
  2. ca sau: phần còn lại của audit ROADMAP (Phụ lục C, K1–K5, Q1–Q13, bàn §4.7);
  3. ca sau: vòng sửa S4 theo D-OP-13 (ca âm → so rule / model / hybrid → sửa → đóng băng mã → chạy H4 MỘT lần);
  4. ca sau: handoff 01.
- **Bị chặn:**
  - lượt W-G7 trả tiền: chờ khoá, D-G7-18 và câu cho phép chi;
  - code R2: chờ D-V3-09;
  - sửa `CLAUDE.md`: gộp vào đợt duyệt R0;
  - đổi hành vi tầng nạp trước W-G7: chưa được phép;
  - câu ở tệp phản hồi mục 8: cột Trả lời của D-H1-3; `HOP-DONG-G7.md` §12 lệch D-OP-13.
- **Test:** từ lần đo 14:43 (451/0/0 có PG) mã không đổi. Lần push này: cổng danh tính chạy lại.
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
| 21/10 14:5x | commit chứa dòng này | Đóng băng H4 bằng cam kết băm (ADR-0006 §10; mục đích theo D-OP-13) | `cam-ket-giu-lai.py` → sha256 `7fc5a485…8150d`, 119 ca; `kiemca` 0 lỗi; kiểm toán nhiễm 7 nhật ký: 0 đường dẫn thật ra ngoài phạm vi | routing 11 < 13: chấm và báo đúng số, không độn |
