# Tiến độ BUILD — tệp sống, cập nhật mỗi lần push

Khuôn theo `docs/PROMPT-BUILD-EAIS.md` §7:
- phần "Bây giờ" ghi đè mỗi lần push;
- phần "Nhật ký" chỉ thêm dòng, không xoá dòng cũ.

## Bây giờ (ghi đè mỗi lần)

- **Cập nhật lúc:** 22/10/2026 09:0x, **NHẬN CA NGÀY 22/10** (09:00 → 17:00) · **HEAD:** `ffb1efb` (cộng commit chứa tệp này). Từ tối qua không có PROMPT, handoff hay câu trả lời CHO mới.
- **Xong ca tối 21/10** (18:07 → 22:00, PROMPT-DOI-TEN-VA-LUONG-MOI + brief §4):
  - Việc 1, đổi tên lại EAIS (D-RM-02) và Việc 2, luồng theo roadmap mới: XONG, qua 3 lượt soát độc lập; tệp phản hồi `docs/bao-cao/2026-10-21-phan-hoi-doi-ten.md` (`1ccac76`);
  - Việc 3 mục 2: đo lại và soát mã bản thử R4 (V5) — CHƯA áp mã (DỪNG ở điểm an toàn: bước tiếp là test đỏ trước rồi sửa);
  - Việc 3 mục 7: K1–K5 của phản hồi audit ROADMAP (`docs/bao-cao/2026-10-21-phan-hoi-roadmap-k1-k5.md`); câu mở D-OP-14.
- **Lỗ S1 của cổng phán quyết phải đóng TRƯỚC H4:** R4-2, R4-3 (+ R4-4 S2); R-1 (tên nguồn đứng trước số ở mệnh đề khác: "EX-03 hôm nay được cấp 3850 L. ERP, 3900 L đúng không?" → "Không." dù ERP ghi 3.900 L); K5 trong miền ("Hôm qua đội báo lu 02 một nghìn hai lít dầu đúng không?" → "Không." dù đúng 1.200 L).
- **Lỗi ẩn:** đọc số bằng chữ không dấu không tất định ("hai muoi lit": 11 / 20 lần 10 L, 9 / 20 lần 20 L); bộ ca hiện có 0 lần "muoi" nên số đo không ảnh hưởng. Chờ D-OP-14.
- **Đang làm:** Việc 3 mục 2 — đóng các lỗ của cổng phán quyết trước H4 (R4-2, R4-3, R4-4, R-1, K5 trong miền), bắt đầu bằng test đỏ trước.
- **Tiếp theo:**
  1. Việc 3 mục 2: R4-2…R4-4, R-1 và câu đổi chiều K5 theo thứ tự test đỏ trước → sửa nhỏ nhất → đo bộ câu, fuzz, 117 / H1 / H3 → đột biến → soát độc lập → áp. Sau đó vòng S4 (ADR-0010) → H4 MỘT lần;
  2. Việc 3 mục 3–7: phép đo (canary harness, cờ approx W-G1, CI có PG), giấy Phase 0a / 1, Q2–Q7 / Q9–Q13 / Phụ lục C của phản hồi audit, tài liệu lệch (danh sách trong tệp phản hồi đổi tên mục 7).
- **Bị chặn:**
  - mã COMMAND / WORKFLOW / EVENT: chờ D-RM-04;
  - dữ liệu / người nghiệp vụ thật: chờ D-RM-03;
  - khung "tên lúc viết" cho `docs/study`, `docs/v3`: chờ D-RM-05;
  - sửa bộ đọc số bằng chữ ("một nghìn hai", bảng không dấu): chờ D-OP-14; chưa quyết thì BUILD chỉ chặn ở cổng phán quyết;
  - W-G7 trả tiền: chờ khoá, D-G7-18 và câu cho phép chi.
- **Test** (cổng cuối ca tối, `25a73bf`, container `eais-pg`): `go vet ./...` 0 dòng; `gofmt -l` trên tệp git theo dõi: 0; `go test -count=1 ./...` có PG 455 / 0 / 0. So theo tên với mốc trước đổi tên (`ca62760`, 453): 0 test cùng tên đổi kết quả, 4 cặp đổi tên, 2 test mới. winreport bộ 117 / H1 / H3, đường bộ nhớ và PostgreSQL: 117/117, 89/104 ×2, 109/134 ×2; 0 dòng khác mốc ngoài thời gian chạy (và dòng `exit status 1` mà `go run` in ở mốc). Đột biến: toàn bộ 35 danh mục 676/676 (trạng thái `423e4d3`); n42 25/25, w4-pg-rls 26/26.
- **Handoff / prompt đang mở:**
  - `PROMPT-DOI-TEN-VA-LUONG-MOI.md`: Việc 1 + Việc 2 XONG, tệp phản hồi XONG; Việc 3 đang làm (mục 2 dừng ở điểm an toàn);
  - `docs/debug-handoff/2026-10-21-01-…`: NHẬN;
  - `PROMPT-AUDIT-ROADMAP.md`: K1–K15 xong; còn Q2–Q7, Q9–Q13, Phụ lục C.

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
