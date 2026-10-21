# Tiến độ BUILD — tệp sống, cập nhật mỗi lần push

Khuôn theo `docs/PROMPT-BUILD-EACL.md` §7:
- phần "Bây giờ" ghi đè mỗi lần push;
- phần "Nhật ký" chỉ thêm dòng, không xoá dòng cũ.

## Bây giờ (ghi đè mỗi lần)

- **Cập nhật lúc:** 21/10/2026 15:2x · **HEAD:** `87b07fa` (cộng commit chứa tệp này)
- **Đang làm:**
  - soát độc lập BYPASSRLS `cc84b83`: đang chạy;
  - ghi ADR-0009 "Vòng 3";
  - chốt ca 16:40.
- **Xong từ lần trước:**
  - **Soát lần ba T2.2 (§4 mục 5): KHÔNG ĐẠT.** Hai người soát độc lập cùng tìm ra một hồi quy S1 ở dòng vá (ii): câu hai vế bị phán quyết SAI chiều. Đã sửa ở `01d4348` (T2.3): kiểm chủ thể ở cả phạm vi câu lẫn mệnh đề hỏi; kèm sửa S3 của (iv) ("3,85k"). Test đỏ trước; n43 47/47 bị bắt. Bộ 117 / H1 hai đường / H3: 0 ca đổi nhãn, 0 câu trả lời đổi chữ.
  - **Soát cổng danh tính `c366036`: KHÔNG ĐẠT** (S1: luật theo mục `.git` rộng hơn git). Đã sửa ở `87b07fa`: chỉ bỏ qua mục tên `.git` và đúng các đường git bỏ qua; n42 11/11 bị bắt.
- **Còn mở, không phải hồi quy (có từ T2.1):**
  - S1 dạng gương "3850 L (3850 L của dự án A) đúng không?" → "Đúng.";
  - S2 các biến thể dấu câu của R2-2.

  Cách sửa đề xuất (E3) có cái giá, nên làm ở ca sau với soát độc lập.
- **Tiếp theo:**
  1. thu kết quả soát BYPASSRLS;
  2. chốt ca 16:40;
  3. ca sau: phần còn lại của audit, E3 cho cổng phán quyết, vòng S4 theo ADR-0010.
- **Bị chặn:**
  - lượt W-G7 trả tiền: chờ khoá, D-G7-18 và câu cho phép chi;
  - code R2: chờ D-V3-09;
  - sửa `CLAUDE.md`: gộp vào đợt duyệt R0;
  - đổi hành vi tầng nạp trước W-G7: chưa được phép;
  - (c) của ADR-0010: chờ D-OP-11;
  - cột Trả lời của D-H1-3; `HOP-DONG-G7.md` §12.
- **Test** (15:19): `go vet ./...` sạch; `gofmt` trên tệp đang theo dõi 0; `go test -count=1 ./...` có PG 453/0/0. So theo tên: +1 test (`TestCongPhanQuyetSoatLan3`), 0 mất, 0 đổi.
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
| 21/10 15:2x | commit chứa dòng này | Cập nhật tiến độ, push | `go test -count=1 ./...` có PG 453/0/0 | — |
