# Tiến độ BUILD — tệp sống, cập nhật mỗi lần push

Khuôn theo `docs/PROMPT-BUILD-EACL.md` §7:
- phần "Bây giờ" ghi đè mỗi lần push;
- phần "Nhật ký" chỉ thêm dòng, không xoá dòng cũ.

## Bây giờ (ghi đè mỗi lần)

- **Cập nhật lúc:** 21/10/2026 14:4x · **HEAD:** `a299f06` (cộng commit chứa tệp này)
- **Đang làm:** H4, pha sửa → ghép → kiểm toán nhiễm → cam kết băm. Thuộc §4 mục 2, theo điều kiện của D-OP-13: H4 chỉ đóng băng, KHÔNG chạy trước khi mã S4 đóng băng.
  - Phần A2 đã sửa xong: 56 ca. Lớp routing còn 11 ca: theo ADR-0006 §10, lớp dưới 13 ca vẫn chấm, ghi đúng số, không soạn thêm.
  - Phần A1: người sửa còn chạy.
- **Xong từ lần trước:**
  - phản hồi audit ROADMAP, bản giữa chừng (K6–K15) — `f38b4eb`;
  - nhận handoff 01 và sửa giờ ghi D-OP-13 — `a299f06`;
  - D-OP-13 — `ac98cf4`.
- **Tiếp theo:**
  1. đóng băng H4 bằng cam kết băm, push manifest (nội dung ở máy);
  2. chốt ca 16:40: báo cáo ca, kèm mục "Trả lời handoff / prompt";
  3. ca sau: phần còn lại của audit (Phụ lục C, K1–K5, Q1–Q13, bàn §4.7), rồi vòng sửa S4 theo D-OP-13.
- **Bị chặn:**
  - lượt W-G7 trả tiền: chờ khoá, D-G7-18 và câu cho phép chi;
  - code R2: chờ D-V3-09;
  - sửa `CLAUDE.md`: gộp vào đợt duyệt R0;
  - đổi hành vi tầng nạp trước W-G7: chưa được phép;
  - câu mới ở tệp phản hồi mục 8: cột Trả lời của D-H1-3; `HOP-DONG-G7.md` §12 lệch D-OP-13.
- **Test** (14:43, mã không đổi từ `8280f40`):
  - `go vet ./...` sạch; `gofmt -l .` rỗng;
  - `go test -count=1 ./...` có PG: 451 ĐẠT / 0 đỏ / 0 SKIP.
- **Handoff / prompt đang mở:**
  - `docs/debug-handoff/2026-10-21-01-…`: NHẬN;
  - `PROMPT-AUDIT-ROADMAP.md`: bản giữa chừng, phần còn lại ở ca sau;
  - `PROMPT-BUILD-EACL.md`: đang làm theo.

## Nhật ký (thêm dòng, không xoá)

| Lúc | Commit | Việc | Bằng chứng (lệnh → kết quả) | Ghi chú |
|---|---|---|---|---|
| 21/10 trước 14:00 | `e28f8d5` … `8280f40` | Ca ngày 21/10: migration EACL; H3 đóng băng, chạy một lần; cổng phán quyết T2 → T2.2 | Xem `docs/bao-cao/2026-10-21-ca-ngay.md` | Trước khi có tệp này |
| 21/10 14:19 | `ac98cf4` | D-OP-13 ghi vào `CHO` mục 0 | Quét riêng tư 0 mẫu; cổng danh tính ĐẠT | Lời chốt của chủ dự án |
| 21/10 14:4x | `f38b4eb` | Phản hồi audit ROADMAP, bản giữa chừng | K6–K15: 6 XÁC NHẬN, 3 MỘT PHẦN, 1 BÁC. 9 test có sẵn chạy lại: 8 ĐẠT, 1 SKIP (không PG) | Người kiểm độc lập; phiên BUILD đo lại |
| 21/10 14:4x | `a299f06` | Nhận handoff 01; sửa giờ ghi D-OP-13 | — | Chưa sửa mã |
| 21/10 14:4x | commit chứa dòng này | Tạo tệp tiến độ | `go vet` 0 dòng; `gofmt -l` 0 tệp; `go test -count=1 ./...` có PG: 451/0/0 | — |
