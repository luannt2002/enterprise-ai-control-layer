# Tiến độ BUILD — tệp sống, cập nhật mỗi lần push

Khuôn theo `docs/PROMPT-BUILD-EACL.md` §7:
- phần "Bây giờ" ghi đè mỗi lần push;
- phần "Nhật ký" chỉ thêm dòng, không xoá dòng cũ.

## Bây giờ (ghi đè mỗi lần)

- **Cập nhật lúc:** 21/10/2026 15:0x · **HEAD:** `49c841a` (cộng commit chứa tệp này)
- **Đang làm:**
  - soát độc lập lần ba cho 3 dòng vá T2.2 (§4 mục 5): 2 agent, mỗi người một worktree, đang chạy;
  - soát độc lập thay đổi cổng danh tính `c366036`: đang chạy;
  - soát độc lập BYPASSRLS `cc84b83`: chờ có khe agent.
- **Xong từ lần trước:**
  - **cổng danh tính đỏ do môi trường đã sửa** — `c366036`. Worktree của agent dựng trong repo làm cổng báo 34 vi phạm ở bản chép; nay cổng bỏ qua checkout git lồng, có ca âm, `.gitignore` bỏ qua `.claude/worktrees/`; n42 11/11 bị bắt;
  - **K7 BYPASSRLS** — `cc84b83` (§4 mục 3, việc an ninh). `pgstore.Open` từ chối vai superuser / BYPASSRLS; ca âm gồm 3 vai; `w4-pg-rls` 9/9 bị bắt. Không đổi hành vi hỏi / trả lời;
  - **ADR-0010 (ĐỀ XUẤT)**, tầng ý định S4 theo bất biến — `49c841a` (§4 mục 2; chưa áp mã);
  - H4 đóng băng — `15f04f9`.
- **Tiếp theo:**
  1. thu kết quả 3 lượt soát độc lập; có lỗi thì sửa tiếp;
  2. chốt ca 16:40;
  3. ca sau: phần còn lại của audit, vòng S4 theo ADR-0010, harness canary, handoff 01.
- **Bị chặn:**
  - lượt W-G7 trả tiền: chờ khoá, D-G7-18 và câu cho phép chi;
  - code R2: chờ D-V3-09;
  - sửa `CLAUDE.md`: gộp vào đợt duyệt R0;
  - đổi hành vi tầng nạp trước W-G7: chưa được phép;
  - (c) của ADR-0010: chờ D-OP-11;
  - cột Trả lời của D-H1-3; `HOP-DONG-G7.md` §12.
- **Test** (15:0x): `go vet ./...` sạch; `gofmt` trên tệp đang theo dõi 0; `go test -count=1 ./...` có PG 452/0/0. So theo tên với 14:43: +1 test (`TestOpenTuChoiVaiVuotRLS`), 0 mất, 0 đổi.
- **Handoff / prompt đang mở:**
  - `docs/debug-handoff/2026-10-21-01-…`: NHẬN;
  - `PROMPT-AUDIT-ROADMAP.md`: bản giữa chừng (K7 nay đã sửa ở `cc84b83`);
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
| 21/10 15:0x | commit chứa dòng này | Cập nhật tiến độ, push | `go test -count=1 ./...` có PG 452/0/0 | — |
