# Tiến độ BUILD — tệp sống, cập nhật mỗi lần push

Khuôn theo `docs/PROMPT-BUILD-EAIS.md` §7:
- phần "Bây giờ" ghi đè mỗi lần push;
- phần "Nhật ký" chỉ thêm dòng, không xoá dòng cũ.

## Bây giờ (ghi đè mỗi lần)

- **Cập nhật lúc:** 21/10/2026 21:2x · **HEAD:** `d5048b9` (cộng commit chứa tệp này).
- **Xong từ lần trước:** soát mã độc lập bản thử R4 (V5), agent chỉ đọc. Không đổi mã.
  - V5 chỉ BỚT phán quyết (nhánh mới chỉ trả "không phán quyết"): không đổi chiều, không thêm; 17 đầu vào xấu (rỗng, NFD, ký tự độ rộng 0, ngoặc lồng 300 tầng, 2.000 ký tự…): 0 crash.
  - Xác nhận hai điều kiện chặn áp: 16 / 57 khối mã mới không test nào chạy; lỗ ngày dạng số nằm ở hàm kiểm "đoạn trơn" gặp `/`, `-`.
  - **Phát hiện R-1, S1 còn mở ở cả HEAD và V5:** tên nguồn đứng trước số ở mệnh đề khác. "EX-03 hôm nay được cấp 3850 L. ERP, 3900 L đúng không?" → "Không." dù ERP ghi 3.900 L (BUILD đo lại). Người soát cho là cùng cơ chế với R2-4 của ADR-0009 (tên nguồn ở mệnh đề ngữ cảnh, đang xếp THẤP-TB); dạng này cho phán quyết SAI CHIỀU nên sẽ ghi thành lỗ S1 riêng và đóng trước H4.
  - Đề xuất tách giai đoạn: lõi khoảng 35 dòng đóng R4-2 / R4-3 (S1) trước; phần bí danh và dấu câu (R4-4, S2) sau. Bảng 23 test đỏ-trước (14 âm, 9 dương), kỳ vọng suy từ ADR.
- **Trạng thái R4 (bước đo lại, push trước):** V5 đóng 25.669 / 25.669 phán quyết ở câu chưa giải trên bộ câu tổng hợp, 0 đổi chiều, 0 phán quyết mới; 117 / H1 / H3 0 dòng khác. CHƯA áp.
- **Đang làm:**
  - K1–K5 của phản hồi audit ROADMAP (brief §4 mục 7): bơm một dòng vào bản sao dữ liệu seed 42 rồi chạy winreport, mỗi nhóm có người đo lại độc lập; không sửa repo.
- **Tiếp theo:**
  1. Việc 3 mục 2: R4-2…R4-4 theo thứ tự test đỏ trước → sửa nhỏ nhất (V5 hay bản nhỏ hơn; sửa lỗ ngày dạng số) → đo bộ câu, fuzz, 117 / H1 / H3 → đột biến → soát độc lập → áp; đóng luôn R-1 (S1, cùng quy trình). Sau đó vòng S4 (ADR-0010) → H4 MỘT lần;
  2. Việc 3 mục 3–7: phép đo (canary harness, cờ approx, CI có PG), giấy Phase 0a / 1, tài liệu lệch (danh sách trong tệp phản hồi đổi tên mục 7).
- **Bị chặn:**
  - mã COMMAND / WORKFLOW / EVENT: chờ D-RM-04;
  - dữ liệu / người nghiệp vụ thật: chờ D-RM-03;
  - khung "tên lúc viết" cho `docs/study`, `docs/v3`: chờ D-RM-05;
  - W-G7 trả tiền: chờ khoá, D-G7-18 và câu cho phép chi.
- **Test** (`903623d`, container `eais-pg`): `go test -count=1 ./...` có PG 455 / 0 / 0; không PG 448 + 7 SKIP (7 test PG). So theo tên với trước đổi tên (`ca62760`, 453): 0 đổi kết quả, 4 cặp đổi tên, 2 test mới. winreport bộ 117 / H1 / H3, hai đường: 0 dòng khác ngoài số đo thời gian (đo ở trạng thái `423e4d3`; mã lõi không đổi tới HEAD). Đột biến: toàn bộ 35 danh mục 676/676 (trạng thái `423e4d3`); n42 25/25 và w4-pg-rls 26/26 ở HEAD. Không đổi mã từ mốc này.
- **Handoff / prompt đang mở:**
  - `PROMPT-DOI-TEN-VA-LUONG-MOI.md`: Việc 1 + Việc 2 XONG, tệp phản hồi XONG (`1ccac76`); Việc 3 đang làm;
  - `docs/debug-handoff/2026-10-21-01-…`: NHẬN;
  - `PROMPT-AUDIT-ROADMAP.md`: bản giữa chừng; K1–K5 đang đo.
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
| 21/10 21:1x | commit chứa dòng này | Việc 3 mục 2: đo lại bản thử R4 (V5) của cổng phán quyết; không đổi mã | Bộ câu dựng lại khớp md5; chấm lại từ đầu ra thô: đóng 25.669 / 25.669, 0 đổi chiều, 0 phán quyết mới; fuzz 10.774 / 10.774; 117 / H1 / H3 0 dòng khác; go test không PG 455 theo tên, 0 đổi | Chưa áp: thiếu test bảo vệ nhánh mới, lỗ ngày dạng số, chặt hơn luật thử R4-4. Hồ sơ đo chỉ ở máy BUILD |
| 21/10 21:2x | commit chứa dòng này | Soát mã độc lập bản thử R4 (V5); không đổi mã | Agent chỉ đọc: 17 đầu vào xấu, 0 crash; tệp phủ: 16 / 57 khối mã mới không test nào chạy. BUILD đo lại R-1 bằng binary test dựng sẵn: "… ERP, 3900 L đúng không?" → "Không." ở HEAD và V5 | R-1 cùng cơ chế R2-4 của ADR-0009 (theo người soát); sai chiều nên ghi thành lỗ S1, đóng trước H4 |
