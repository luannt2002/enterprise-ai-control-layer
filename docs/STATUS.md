# STATUS — trạng thái EAIS

> Cập nhật: **24/10/2026 tối — TÍCH HỢP: mọi nhánh đã vào main (D-RAG-05)**, báo cáo `docs/bao-cao/2026-10-24-tich-hop-main.md`. Phần dưới khối
> "Tích hợp 24/10" giữ nguyên chữ của 23/10 17:26 (báo cáo ca `docs/bao-cao/2026-10-23-ca.md`); tiến độ từng lần push: `docs/bao-cao/TIEN-DO-BUILD.md`.
> Báo cáo ngày: `docs/bao-cao/2026-10-20-win-mvp.md`. Chạy lại / làm tiếp ở máy khác: `docs/ops/README.md`.
> **Roadmap hiện hành: `docs/ROADMAP-DAY-DU.md` (D-RM-01).** Tên sản phẩm: EAIS (D-RM-02, đổi tên lại tối 21/10). Việc đang làm: khối "MVP" ngay dưới (khối "Việc hiện hành (tối 21/10)" ở mục Việc tiếp theo là lịch sử).
> **Câu đang chờ chủ dự án quyết: `docs/CHO-QUYET-DINH.md`.** Checkpoint G7 (A–H, chưa gọi model thật): `docs/g7/CHECKPOINT-A-H.md`.
> Số ở mục H1 và mục đêm 20→21/10 đo ngày 21/10; các mục ghi 20/10 (bảng bộ 117, bằng chứng, đột biến) đo ngày 20/10. Chỗ ghi **CHƯA ĐO** là chưa đo.

## Tích hợp 24/10 tối (D-RAG-05): main = mọi nhánh + RAG — MVP vẫn CHƯA RA

- Main `4fe8e0b` (rồi các commit tài liệu): gộp nội dung vòng sửa S4 (KHOÁ, kèm nợ S4CS-01…04 — tức D-AI-05 = a), nợ S4 sau H4, vết
  S8–S10, 5a vòng 5, mã Phase 0a (migration **0009**; R-01 chết khoá PG còn mở), RAG (D-RAG-01, D-RAG-04); 13 nhánh đã vào / bị thay chỉ ghi
  nhận. Soát độc lập sau gộp: 0 P0, 1 P1 + 6 P2 do giao nhau — đã sửa có test đỏ trước (bảng ở báo cáo tích hợp mục 3).
- Đo [lệnh ở báo cáo mục 7]: winreport 117/117, H1 **96**/104, H3 **122**/134, H4 **109**/119 (bộ đã công bố), 0 ca xanh → đỏ, bộ nhớ = PG;
  smoke 21/21; cổng có PG trên `4fe8e0b`: 2398 test theo tên, 0 mất, 0 đổi, 0 đỏ.
- CHƯA chạy (dùng một lần, chờ chủ dự án): H6, bộ ca 0a. Migration mới tiếp theo: **0010**.

## MVP (`docs/ROADMAP-DAY-DU.md` §5b): CHƯA RA — M1 vế đọc vết 78,9% (< 90%); M2 bộ ca 0a đã băm, mã đang soát; M3 chờ D-AI-05; M4 đã gộp K1–K4, P1 153 / 157, chờ Q-P1-15

Trên dữ liệu giả lập, chưa có người nghiệp vụ duyệt (D-RM-03). Bảng lệnh và kết quả: báo cáo ca 22/10 mục 7; từng dòng CỔNG TRƯỚC AI:
`docs/bao-cao/CONG-TRUOC-AI.md`. Từ 23/10, `docs/ROADMAP-DAY-DU.md` §5b là nguồn duy nhất của cổng ra từng khối (chủ dự án 23/10).

- **M1 CHƯA RA** — phần chạy đạt (smoke, CI có PG, vết bền); vế đọc vết KHÔNG ĐẠT (29 / 33 ngày 22/10; đo lại bằng thẻ mới 23/10: 30 / 38 = 78,9%, S10 / S8 không phân biệt được — `docs/bao-cao/2026-10-23-do-nguoi-doc-vet-lan2.md`). 5e (R2-01) gộp `6fa6ec6`: phiên bản dữ liệu của vết là của lần nạp đã commit vào PG (sổ nạp, migration 0006). Ngày 22/10 ghi "RA theo cổng ra của lệnh ca":
  mục 3a của lệnh ca 22/10 thiếu vế "người soát chỉ đọc vết chỉ đúng tầng sai đầu tiên ≥ 90%" của §5b; chủ dự án đính chính 23/10 (lỗi của
  lệnh ca, không phải của BUILD). Đã có: `cmd/eais` (POST /v1/ask, POST /v1/ingest, GET /v1/trace/{id}, GET /healthz; principal từ token,
  không lấy tenant từ thân); vết từng bước S0–S10 lưu bền theo tenant cho mọi lượt đã xác thực (migration 0003, RLS), xem bằng `eais trace`,
  kể cả lượt lỗi; một lệnh `scripts/chay-thu.sh` từ clone sạch (smoke E2E 21 / 21; từ chối dùng lại PG đang chạy trừ khi `EAIS_REUSE_PG=1`, và
  từ chối khi compose project đã có container chạy / tạm dừng); `code_version` của vết là mã git, giá trị nhúng bằng `-ldflags`, hoặc băm của
  binary; CI có PG, cấm SKIP — định nghĩa ở `.github/workflows/ci.yml`, chưa chạy trên GitHub (ghi chú 24/10, F-014); vết xuất lý do và phương án đã xét ở S2–S5, S7, cổng phán quyết (R2-03, `5fd463f`). Tiếp: đo lại trên bộ
  thẻ mới chia đều theo tầng, có ca xanh đối chứng.
- **M3 chưa RA**: cổng phán quyết KHOÁ HẲN (`59557c2`); H4 chạy một lần 104 / 119 hai đường; dòng S4 KHÔNG ĐẠT (10 / 119 = 8,4% > 5%; 3 câu
  sai nghĩa không nhãn). Chủ dự án 23/10: D-AI-02 = a — vòng sửa S4 bằng luật đang chạy (gộp D-OP-18 = c, D-OP-11 = b); chấm trên H6 do người
  soạn khác soạn, băm trước khi bản sửa lên main, chạy một lần. H6 đã băm (`6734cff`). Vòng sửa S4 KHOÁ sau vòng 3 (23/10 14:10, soát vòng 3
  KHÔNG ĐẠT: S4CS-02 hồi quy S1 mới) — không gộp, H6 chưa chạy; chờ chủ dự án **D-AI-05** (khuyến nghị: một vòng ngoại lệ hẹp) và D-AI-04.
- **M4**: ADR-0011 CHỐT; bộ ca giữ lại độc lập Phase 1 ĐÓNG BĂNG bằng cam kết băm (`1ddd10b`, `docs/g7/holdout-p1/holdout-manifest.json`,
  157 ca / 23 lớp) ⇒ đủ điều kiện mở mã. K1 (lõi lệnh `execution/commands`, kho giả) đã gộp (`e1ce950`; soát vòng 2 ĐẠT CÓ GHI CHÚ);
  K2 (kho PG của phiếu, migration 0004) đã gộp (`d919ebf`; soát ĐẠT CÓ GHI CHÚ); K3 (route `POST /v1/commands/<tool>` ở `cmd/eais`) cùng bảy
  lỗi P2 C1–C7 (migration 0005: vai ứng dụng không đọc được `audit_log.reason_detail`, CHECK tenant không rỗng) đã gộp (`3892be2`; soát lại
  độc lập ĐẠT CÓ GHI CHÚ, ghi chú M4R-01…08 sửa trước khi gộp); K4 (bộ chạy ca, điểm chèn lỗi, đột biến, demo) đã gộp (`b22e1e6`). **Bộ giữ
  lại P1 chạy MỘT lần** (23/10 14:29): 153 / 157 đạt, 4 đỏ cùng một lỗi chuẩn hoá (`"≈"` không nhận là dấu ước lượng ⇒ phiếu mất cờ `approx`);
  X1, X5 ĐỎ ⇒ M4 CHƯA RA; phân tích `docs/g7/holdout-p1/ket-qua-p1/phan-tich.md`, câu Q-P1-15. Lỗi `"≈"` đã sửa và gộp (`d43a1bf`; soát
  độc lập ĐẠT CÓ GHI CHÚ); bộ P1 KHÔNG chạy lại — đo lại M4 độc lập chờ Q-P1-15. X8 (danh mục đột biến M1–M26, mã `32f4930`): 32 đúng chỗ,
  0 sống sót, 0 khác; ngoại lệ M5 của §11.2 chờ Q-P1-14.
- **M2**: giấy Phase 0a KHOÁ ở vòng 6 (luật 23/10: giấy tối đa 3 vòng soát); R6-01, R6-02 thành nợ có mã, chặn bằng test đỏ trong mã Phase 0a.
  **Bộ ca giữ lại độc lập 0a ĐÓNG BĂNG** (`a52eb67`, `docs/g7/holdout-0a/holdout-manifest.json`): 249 ca, 83 / 84 lớp (LC10 = 0), hai người
  soạn không đọc mã, kiểm nhiễm 0 vi phạm. Mã bước 2–3b + kho PG (nhánh `m2/0a-ma-truoc-dong-bang` — gộp main 24/10 `71c6b45`, migration **0009**, R6-01 chặn bằng test
  đỏ): soát 23/10 KHÔNG ĐẠT hẹp (R-01 chết khoá trên PG; S1 = 0) — đang sửa; bộ 0a chạy MỘT lần sau khi mã đóng băng.
- **M5 chưa đạt**: S2, S3, S4 KHÔNG ĐẠT; L1 KHÔNG ĐẠT (lỗi đã đo); S5, S6 CHƯA ĐẠT — chưa có bộ ca độc lập / ca âm (số theo tầng ở
  `docs/bao-cao/CONG-TRUOC-AI.md`); S7 NGOÀI MVP — **MVP không gồm câu hỏi quy trình (RAG)** (D-AI-03 = a); S9 chưa có bộ ca độc lập (giấy
  đo đã khoá; bộ ca độc lập đang soạn 23/10: 36 kịch bản gốc, bốn lô); Vết: vế ≥ 90% KHÔNG ĐẠT (đo lại 23/10 bằng thẻ mới: 30 / 38 = 78,9%).
- Cổng cuối (cổng §6 ở cây tích hợp `32f4930` = main `d614395` + 15 commit 5b / 5c (11 cherry-pick + 4 sửa theo soát lại); sau đó chỉ đổi tài
  liệu; lúc 23/10 14:49, database PG riêng `lead_5bcf`, 5 agent chạy song song): `go vet ./...` 0 dòng; `gofmt -l $(git ls-files '*.go')` 0 tệp;
  `go test -count=1 -json ./...` có PG: **1956** test pass, 0 fail, 0 skip — so theo tên với cổng `b22e1e6` (1942): 0 đổi, 0 mất, 14 mới;
  `python3 -I scripts/kiem-skip-test.py` XANH; winreport (`-data <du-lieu-42> -v`, bộ nhớ và `-pg`): bộ 117 117/117, **H1 91/104, H3 112/134,
  H4 103/119** (hai đường giống nhau) — mốc so: đầu ra cổng `b22e1e6`; 0 dòng khác ở cả 7 tệp. CI GitHub: chờ lượt của commit gộp.

## Tiến độ WIN MVP: trên bộ giữ lại độc lập H1 và H3, phần ca của G1–G5 KHÔNG ĐẠT — CHƯA phải WIN

> **H4, 22/10 14:59** (`docs/g7/holdout-h4/`): đóng băng bằng cam kết băm (`15f04f9`), chạy MỘT lần ở `57409f2` (mã S4 đóng băng ở `a665298`, cổng phán quyết khoá hẳn): **104/119 ở cả hai đường**, trạng thái HOÀN TẤT, phát lại 119/119 · 0 lệch, 0 canary. Theo lớp: representation 16/18, entity 16/16, ambiguity 13/14, semantic 13/14, source_conflict 12/15, permission 16/16, unsupported_answer 9/15, routing 9/11. G2' và G5' (phần ca) KHÔNG ĐẠT; G6' phần đo được ĐẠT. Đối chứng mốc `7d72e33` (chỉ chẩn đoán): 102/119 — 2 ca đỏ → xanh (744, 745), 0 ca xanh → đỏ; 15 ca đỏ đều đã đỏ ở mốc. Từ nay H4 là bộ HỒI QUY; kết luận mới chờ H5. Tầng sai đầu tiên của 15 ca đỏ (người phân tích + người kiểm chéo độc lập, đồng ý 13 / 13): S4 ở 10 ca, cổng phán quyết (đóng thừa, an toàn) ở 5 ca ⇒ **dòng S4 của CỔNG TRƯỚC AI KHÔNG ĐẠT** (10 / 119 = 8,4% > 5%; 3 câu sai nghĩa không nhãn: 758, 847, 854) ⇒ **M3 chưa RA, chờ D-AI-02** (báo cáo theo tầng: ADR-0010 "Kết quả H4").
> - 119 ca, 82 ca có số; lớp routing 11 (< 13, chấm và báo đúng số, không độn).
> - Theo D-OP-13, H4 chấm vòng sửa tầng ý định S4 (ADR-0010, ĐỀ XUẤT) và chỉ chạy MỘT lần sau khi mã S4 đóng băng. Lượt chính W-G7 dùng H5.
>
> **Cổng phán quyết T2.3, 21/10** (`01d4348`): soát lần ba tìm ra dòng vá (ii) của T2.2 gây phán quyết SAI chiều ở câu hai vế; đã sửa.
> - Bộ 117 ca, H1, H3: 0 ca đổi nhãn, 0 câu trả lời đổi chữ.
> - E3 (`a853e96`) đóng S1 dạng gương "(3850 L của dự án A)"; sửa sau soát ở `936d090`.
> - **Cổng phán quyết Vòng 5 (22/10, `03c2c54`, `489c98a`; ADR-0009 Vòng 5):** đóng R4-2, R4-3 (S1), R4-4 (S2), R5-1 (S1: tên nguồn / chỉ số khác ở ngữ cảnh là nguồn của chính claim, gồm R2-4) và R5-2 (S1: số bằng chữ "một nghìn hai", "ba nghìn tám rưỡi" không phán quyết). Bộ câu tổng hợp: 25.669 / 25.669 phán quyết sai chỗ đóng (7.168 sai chiều); 713.097 câu: 0 đổi chiều, 0 phán quyết mới; 117 / H1 / H3 hai đường 0 dòng khác; đột biến 210 / 210 bị bắt; ba vòng soát độc lập.
> - **R5-4, R5-5 đã đóng** (`962ae9c`). **Cổng phán quyết KHOÁ** (22/10, brief BUILD §9.6; `f0e1e93`, sửa theo hai vòng soát `017495c`, `7e1d8ca`; ADR-0009 "Bước khoá"; soát vòng 1, 2 KHÔNG ĐẠT đã sửa; vòng 3 tìm thêm 2 lỗ SAI CHIỀU có từ trước ⇒ mở lại theo ngoại lệ §9.6, sửa `be91cd0`; vòng 4 ĐẠT CÓ GHI CHÚ ⇒ KHOÁ HẲN): cụm nguồn / chỉ số khác ở ngữ cảnh mà không kèm số riêng ⇒ không phán quyết. Lưới của người soát: sai chiều 448 / 123 / 123 → 0; corpus 716.133 câu: 0 phán quyết mới, 0 đổi chiều. Cái giá: mất phán quyết ở câu "Bao nhiêu định mức, vậy …" (vẫn trả số). Từ nay lỗ mới cùng họ chỉ ghi nợ, giữ fail-closed.
> - **D-OP-14 (bộ đọc số bằng chữ, `9c5b14f`):** "một nghìn hai" = 1.200, "ba nghìn tám rưỡi" = 3.850; cách nói tắt ("… lẻ", "mười hai trăm") thì cổng không phán quyết.
> - **Vòng sửa S4 (ADR-0010 (a), D-OP-15 = a2 kèm 2 điều kiện; `723a570`, sửa theo soát vòng 1 `32771ff` và vòng 2 `a665298`; vòng 3 — vòng cuối — KHÔNG ĐẠT có điều kiện, không lỗi ngoại lệ ⇒ ĐÓNG BĂNG ở `a665298`):** câu có thực thể + thuật ngữ mà không từ khoá nào khớp không còn bị từ chối "ngoài phạm vi" — trả lời theo ý định chung kèm câu giả định khi câu không nói gì thêm, còn lại hỏi lại nghĩa số liệu. Đo ở `a665298`: test có PG 476 pass (so theo tên 0 đổi, 0 mất); 117 / H1 / H3 hai đường: 0 dòng khác so với trước vòng 3 (117/117, H1 92/104, H3 113/134 — H1 / H3 KHÔNG phải bằng chứng, xem ADR-0010); corpus 716.133 câu: 0 phán quyết đổi; 700 câu điều kiện (ii): 0 câu a2 trả số kho xuất. Nợ đã ghi: S4R-03 (từ khoá khớp chữ con, câu sai nghĩa không nhãn, có từ trước), D-OP-16, danh sách đóng "dầu" nghĩa khác, lượng tiền đọc thành lít (có nhãn).
>
> **H3, 21/10 13:08** (`docs/g7/holdout-h3/`). Bộ giữ lại độc lập thứ ba: 134 ca, đóng băng bằng cam kết băm 11:04 (`259d38d`) TRƯỚC mọi bản sửa ngữ nghĩa của ca, chạy MỘT lần ở `9091c84` (mã T2.2 `a1595f0`).
> - Kết quả: **109/134** ở cả hai đường; phát lại 134/134, 0 lệch; 0 canary. Theo lớp: representation 16/17, entity 14/14, ambiguity 12/18, semantic 14/16, source_conflict 12/18, permission 16/18, unsupported_answer 14/18, routing 11/15 ⇒ phần ca G1–G5 KHÔNG ĐẠT; G6' phần đo được ĐẠT.
> - Đối chứng mốc `016422e` (chỉ chẩn đoán): **113/134**. Bản sửa của ca trượt ròng 4 ca (507, 564, 602, 650; 0 ca ngược lại) — cả 4 là câu xác nhận có lời khẳng định sai mà cổng phán quyết giữ lại một "Không." vốn đúng.
> - 42 ca xác nhận: mốc 28 phán quyết đúng chiều / 0 sai chiều / 13 không phán quyết; bản sửa 16 / 0 / 26. Độ nhạy 3 (đăng ký trước): chặt 0, nới 1.
> - H3 từ nay là bộ HỒI QUY. Tầng sai đầu tiên của 25 ca đỏ: xem báo cáo ca ngày 21/10.

> **H1, 21/10 01:42** (`docs/g7/holdout/ket-qua-h1/`). Bộ giữ lại độc lập: 104 ca, 13 / lớp, do người không đọc mã soạn, đóng băng 22:51 ngày 20/10, chạy MỘT lần ở revision `9de63d2`.
> - Kết quả: **89/104** ở cả hai đường (bộ nhớ, PostgreSQL) theo bộ chấm nhị phân. Theo nội dung **≤ 88/104**: ca 264 ĐẠT với câu trả lời SAI ("1 490 lít" không được gộp nhóm nghìn); người phản bác cuối tìm ra. Sửa sau H1 ở `8ff3db5`: bản chốt trả "Đúng. … 1.490 L"; H1 làm bộ hồi quy trên bản chốt (`15ed135`): 89/104, 0 ca đổi nhãn. Hai bản thay thế để đóng các lỗ còn lại đều bị soát bác (`fd4d19c` sinh số mồ côi — revert `2a87e63`; `3066044` nuốt số của tên thiết bị — không vào main) ⇒ nhóm nghìn còn lỗi đã đo (D-OP-9). Phát lại 104/104 lệch 0; canary 0.
> - Theo lớp và cổng (`04` §6 đòi 100 % mỗi lớp):
>   - representation 10/13 (G1);
>   - entity 13/13, ambiguity 12/13 (G2);
>   - semantic 11/13, source_conflict 11/13 (G3);
>   - routing 7/13 (G4);
>   - permission 13/13, unsupported_answer 12/13 (G5).
> - Vậy phần ca của G1–G5 đều KHÔNG ĐẠT trên dữ liệu độc lập. 15 ca đỏ quyết định sai ở tầng ý định (S4; 13/15 sai đầu tiên ở đó, 2/15 ở S2); 4/15 trả lời sai không báo, 1/15 báo thiếu dữ liệu của chỉ số sai (`phan-tich.md`).
> - G6' PARTIAL (dòng tiến độ ngay dưới); tiêu chí sửa đổi chờ chủ dự án phê. G7 CHẶN.
> - **Câu xác nhận ("… đúng không?")**: trên 15 câu thăm dò của người soát, đầu đêm `94bbaae` phán quyết sai 8/15, bản cuối sai **15/15** — 7 câu là HỒI QUY của chính các bản sửa đêm nay (`3d730ea`, `6f23ff3`, `a8b0858`). D-OP-10 đã chốt ngày 21/10: chỉ phán quyết khi claim đã giải đủ chiều. Đã sửa bằng cổng phán quyết fail-closed (ADR-0009), vào mã SAU khi H3 đóng băng (`259d38d`): 15 câu thăm dò còn 0 phán quyết sai; câu chưa giải đủ chiều trả số liệu kèm mẩu "Chưa kết luận đúng hay sai…". Soát đối kháng độc lập tìm thêm 21 câu cổng hứa đóng mà chưa đóng (chủ thể, đơn vị, câu không dấu, kỳ, nguồn, dấu âm, ngữ cảnh) — đã đóng ở T2.1 (ADR-0009 "Bổ sung sau soát đối kháng"); 0 câu đảo chiều. H3 chưa chạy.
> - **Tiến độ (cổng đạt trọn, vô điều kiện, theo dữ liệu độc lập): 0/7.**
>   - G6' là PARTIAL (đột biến mỗi cổng, test thuộc tính chưa đủ; 15 ca lộ gợi ý trước đóng băng không bị loại): chỉ thành 1/7 khi chủ dự án phê tiêu chí VÀ làm đủ phần còn thiếu.
>   - Nếu D-H1-1 chọn giữ bộ 117 cho G1 / G3 / G4: 3/7 (không khuyến nghị: bộ 117 không phải bằng chứng độc lập).
> - Bảng dưới là kết quả trên bộ 117, giữ để đối chiếu.

> **Đính chính 20/10 tối (bản soát độc lập `docs/g7/REVIEW-D5.md`):**
> - Ba ca 083, 107, 108 có kỳ vọng do chính phiên viết mã "trọng tài" **sau** lần đo đầu (09:03), **trước** khi băm bộ ca (09:15).
> - Vì vậy G2, G5, G6 chỉ đạt **có điều kiện**. Đạt vô điều kiện: **3/7** (G1, G3, G4).
> - Chờ chủ dự án quyết D-G7-11 (`docs/CHO-QUYET-DINH.md`).

### Bảng bộ 117 ca — 20/10/2026 (đã thay bằng kết quả H1 ở trên; giữ để đối chiếu)

Roadmap hiện hành: `docs/ROADMAP-DAY-DU.md` (D-RM-01, Phase 0–10). Phase đang làm là **0b — WIN MVP Fuel** (`docs/04-core-mvp.md`); mọi phase khác là kế hoạch.

**WIN = 7/7 cổng + scale smoke** (04 §6). Cách tính tiến độ: mỗi cổng 1/7, chỉ tính cổng đạt trọn và vô điều kiện ⇒ **3/7 ≈ 43%**. Nếu D-G7-11 chọn "c" (loại 3 ca trọng tài khỏi phép tính cổng) hoặc "a" (duyệt trọng tài) thì 6/7 ≈ 86%.

- G1, G3, G4 đạt theo đúng tiêu chí của 04 §6. G2, G5, G6 đạt trên bộ ca hiện tại nhưng phụ thuộc 3 kỳ vọng trọng tài (xem đính chính trên). (Từ 22/10 G1 so cả cờ ước lượng: KHÔNG ĐẠT, 2.736/2.739 — bảng dưới.)
- Scale smoke đạt trên cả đường bộ nhớ lẫn đường PostgreSQL.
- **G7 VALUE: CHẶN.** Cần model thật, tức khoá API và chủ dự án đồng ý chi phí. 04 §6 cấm thay bằng model giả.

| Cổng | Kết quả | Bằng chứng |
|---|---|---|
| G1 DATA | **KHÔNG ĐẠT** — 2.737/2.739 (từ `d43a1bf`, 23/10), 2 lệch cờ ước lượng "2k" đã ghim (nhãn sửa theo lời chủ dự án 23/10) | - **Từ 22/10 (làn 2, K1) G1 so cả cờ ước lượng: 2.736/2.739** — 3 lệch đã biết, ghim ở `tenants/construction/win/g1-approx-known.txt` (ratchet chỉ đi xuống): R-001, R-002 "2k" (nghĩa nghiệp vụ, câu mở D-OP-17), R-003 "≈" — đã sửa (`d43a1bf`, 23/10: dấu ký hiệu pack khai được nhận), dòng ghim đã xoá, G1 2.736 → 2.737. Trước 22/10: G1: 2.739/2.739 (functional), 359.945/359.945 (smoke).<br>- Metamorphic: R-004 × 4 biểu diễn, R-001 × 4 cách viết, 0 lệch.<br>- Bộ kiểm có test âm.<br>- `representation` 16/16. |
| G2 IDENTITY | ĐẠT **có điều kiện** (107, 108 là kỳ vọng trọng tài) | - `entity` 13/13.<br>- `ambiguity` 13/13, ứng viên đúng tập. |
| G3 TRUTH | ĐẠT | - `semantic` 14/14, `source_conflict` 12/12.<br>- Ca biên dung sai sinh từ pack cho MỌI cặp nguồn: 4 cặp × "= dung sai" / "dung sai + 1", gồm cặp ngang hạng. |
| G4 DECISION | ĐẠT | - `routing` 17/17.<br>- Runner chấm `model_calls` bằng gateway giả đếm lời gọi.<br>- Đột biến bắt buộc "SQL gọi gateway" (BA) và "RAG bỏ gateway" (BB) đều bị bắt. |
| G5 CONTROL | ĐẠT **có điều kiện** (083 là kỳ vọng trọng tài) | - `permission` 18/18.<br>- Cổng AST: không kiểu `*Request` nào mang tenant.<br>- PG: RLS FORCE + GUC đọc lại; canary B có test cho phép / từ chối, đối chứng dương, ghi chéo tenant bị WITH CHECK chặn.<br>- Ràng buộc claim 6 chiều. Bảng "sai đúng một chiều" 6 × 3 bị chặn 100%; 0 chặn oan trên 13 ca mẫu + 117 ca thật.<br>- Lượng dầu trong tài liệu bị chặn bởi hai lớp, một lớp độc lập với bộ chuẩn hoá.<br>- Chèn lệnh trong SOP không đổi được route / quyền / số. |
| G6 PROOF | ĐẠT **có điều kiện**: vi phạm "đóng băng trước W1" ở 3 ca | - 117 ca do phiên không viết mã soạn (08:47–08:58); mọi lớp ≥ 12 ca.<br>- `cases-independent.yaml` sha256 `ffdd704b535d2195…`, băm ghi 09:15, **sau** lần đo đầu 09:03 và sau khi phiên viết mã trọng tài 3 kỳ vọng (083, 107, 108).<br>- Bản ghi quyết định + phát lại: 117/117 không lệch.<br>- Thuộc tính: tổng ≥ 0; dẫn xuất = a − b lấy từ bằng chứng; A không thấy B.<br>- Mỗi cổng có đột biến bị bắt. |
| G7 VALUE | **CHẶN** | - Bộ khung xong: baseline RAG (`evals/baseline`), luật D5 + bộ đếm độc lập (`evals/g7.go`), lệnh `g7run`, client model thật `models.Messages` cấu hình qua biến môi trường.<br>- Chạy ống nối bằng model giả: D5 ĐẠT, nhưng KHÔNG phải bằng chứng (04 §6).<br>- Thiếu: khoá model + đồng ý chi phí. Có rồi thì chạy `EAIS_MODEL_PROVIDER=anthropic EAIS_MODEL=<model> EAIS_MODEL_API_KEY=<khoá> go run ./tenants/construction/win/g7run -data <dir>`. |
| Scale smoke | ĐẠT | - `scripts/scale-smoke.sh`: 117/117 ở cả functional và smoke, câu trả lời giống từng byte; smoke 3,67 s, peak RSS 562,8 MiB (đo 10:35).<br>- Đường PG (`winreport -pg`): 117/117, từng ca và từng câu trả lời giống đường bộ nhớ; smoke 6,08 s, 653,7 MiB, kể cả nạp ~355 nghìn fact vào PG. |

Mốc "≈ 20%" của tối 19/10 bỏ, vì nó ước theo cảm nhận. Từ nay chỉ đếm cổng có lệnh chứng minh.

## Đêm 20→21/10 — khôi phục bằng chứng (đã chốt 06:04 ngày 21/10)

Mục tiêu đêm: khôi phục bằng chứng đáng tin rồi mới xác định trạng thái WIN thật; không tối ưu tỉ lệ đạt. Chưa gọi model thật, chi 0 USD (khoá model không có trong môi trường).

| Việc | Trạng thái | Bằng chứng |
|---|---|---|
| Ghi quyết định chủ dự án (D-G7-1…11, D-V3, D-OP) | XONG | `docs/CHO-QUYET-DINH.md` (`c5ce51e`) |
| Tái lập mốc `94bbaae` | XONG: test 224/0/0; 117/117 bộ nhớ + PG; phát lại 0 lệch; scale smoke ĐẠT; đột biến 69/69 | đầu ra thô ở máy |
| ADR phương pháp, đăng ký TRƯỚC khi có bộ giữ lại | XONG | ADR-0005 (đã duyệt, D5 sửa đổi), ADR-0006 (bộ giữ lại) (`6f7b4f4`); ADR-0008 (`0eb2531`) |
| Kiểm toán nhiễm 117 ca | XONG: CLEAN 102, CONTAMINATED 3 (083/107/108), UNCERTAIN 12; G6 không đạt điều kiện "đóng băng trước W1" | `docs/g7/CONTAMINATION-AUDIT.md` (`a135c90`) |
| Client model | XONG: không gửi `temperature` (từng làm mọi lời gọi lỗi 400); thinking / effort / thử lại từ cấu hình; lùi có trần | `750307a`, `fe22e2a` |
| Harness G7: B2+ / B1 / B2″ / B2-oracle; `-out`, `-smoke`, `-pilot`, `-budget`; lượt vô hiệu; PROVIDER | XONG | `a1a5b6b`, `fdb3a18`, `a94b5fb`, `15cb1a8` |
| `CompareG7` theo D5 sửa đổi | XONG | `a507e81` |
| **Bộ chấm G7 v1** | **BỊ PHÁ** (người tấn công khai 133/170 đòn thành công; test máy xác nhận 122 đòn không tranh cãi bị chấm sai) ⇒ không dùng làm bằng chứng. Đã vá phần chắc chắn: ratchet 170 đòn (nợ 122 → 118), rỗng = từ chối, hệ lỗi khi hỏi luôn trượt, số được phép từng ca, canary tên | `82baa71`, `6b7d963`, `9264aa6`, `7f6711c` |
| Bộ chấm G7 v2 | ĐỀ XUẤT (ADR-0007). **Không đóng băng được đêm nay**: điều kiện bắt buộc là hiện thực rồi hiệu chỉnh trên đầu ra pilot thật (cần khoá) với người chấm mù | ADR-0007 (`b5df4c8`), `docs/g7/BO-CHAM-V2-DAC-TA.md` |
| Bộ giữ lại độc lập | **ĐÃ ĐÓNG BĂNG 22:51** trước mọi lần chạy: 229 ca → 3 người soát (252 cờ) → 189; loại 7 ca bị lộ trước khi chia ⇒ pilot 16, **H1 104**, H2 62, exposed 7 | `16a6140`; `af4054a`; `docs/g7/holdout/` |
| Sự cố lộ (đã xử lý) | (1) 7 ca lộ cho người viết mã ⇒ loại trước khi chia; (2) 15 ca có gợi ý kỳ vọng lộ cho phiên chính (H1 8, H2 5, pilot 2) lúc 22:50:28 — TRƯỚC đóng băng 33–76 giây, không phải sau như đã ghi ở `59ee556`; theo ADR-0006 §5 lẽ ra phải loại trước khi chia (lệch quy trình, đính chính sáng 21/10) ⇒ tập không đổi, bảng H1 báo độ nhạy | `af4054a`, `59ee556`; D-G7-13 |
| Chứng minh lại G2 / G5 / G6 trên H1 | **XONG 01:42** (chạy MỘT lần, sau khi sửa mọi phát hiện S1/S2 của soát lượt 2):<br>- 89/104;<br>- G2' KHÔNG ĐẠT, G5' KHÔNG ĐẠT, G6' ĐẠT (chờ phê);<br>- phần ca của G1 / G3 / G4 cũng KHÔNG ĐẠT (D-H1-1).<br>15 ca đỏ đều ở S4; không sửa mã theo H1 | `56101be`, `d17144f`; `docs/g7/holdout/ket-qua-h1/` |
| Soát độc lập mã (D-OP-5) | Lượt 1 XONG 21:22: sửa S1-1, S2-1…S2-6, S3-x, S4, test hổng. Lượt 2 XONG trước H1 (pipeline, công cụ bộ giữ lại, harness G7; mỗi chiều có người phản bác). Sau H1 (không ai thấy bộ giữ lại): soát bản sửa A, hai người soát bản sửa R, C2, C3, hai người soát `37e8a08`, một người soát `fd4d19c` + `246eabb` (tìm ra hồi quy S1 / S2 của `fd4d19c` ⇒ revert), hai người soát `3066044` + `15ed135` (hồi quy S1 của `3066044` ⇒ không vào main); người phản bác cuối ba góc | `a94b5fb` … `64daad0`; báo cáo sáng mục 8 |
| Harness G7: kiểm TRƯỚC khi tiêu tiền | XONG: từ chối trước lời gọi model khi lệch tham số hợp đồng, chấm full, bộ chấm chưa đóng băng, thiếu / không ghi được `-out`, hệ cổng bị chặn hay thiếu; endpoint so theo nghĩa, mọi endpoint không phải loopback đều qua kiểm; trần thử khói 3 ca; log / bằng chứng chỉ ghi `scheme://host` | `a928747` … `2304d93` (C2), `246eabb` (C3) |
| Câu xác nhận + nhóm nghìn | Luật an toàn (`37e8a08`, sửa hồi quy `64daad0`). Nhóm nghìn: D-OP-9 = (a') đã gộp 23/10 (`5392833` … `5744d65`; soát độc lập ĐẠT CÓ GHI CHÚ): nhóm hợp lệ luôn gộp, không đứng đầu văn bản ⇒ cờ chưa phân định (đường nạp cách ly, luồng hỏi hỏi lại), chồng tên thiết bị ⇒ cách ly / hỏi lại, ô có cấu trúc luôn chắc chắn; C3-02, C3-05 đã đóng; giá: 3 ca H (264, 509, 717) ĐẠT → ĐỎ có chủ ý. Nợ có từ trước: S2AS-01, S2AS-02 (S1, `docs/bao-cao/CONG-TRUOC-AI.md` dòng S2). Câu xác nhận: bản cuối phán quyết sai 15/15 câu thăm dò (đầu đêm 8/15), CHƯA sửa (D-OP-10) | báo cáo sáng mục 8; `docs/CHO-QUYET-DINH.md` D-OP-9, D-OP-10; `docs/bao-cao/soat/2026-10-23/s2a-soat.json` |
| Chương trình đo của người soát D5 | Công khai, tái lập đúng số của bản soát | `tenants/construction/win/d5probe` (`d03f962`) |
| `go mod tidy` (D-OP-2) | DỪNG theo luật (thêm 2 module chỉ cho test của yaml/v3; 0 gói EAIS dùng); build / vet / test sạch sau tidy ⇒ khuyến nghị chấp nhận, chờ chủ dự án | `ba83265` |
| Câu chờ quyết mới | D-G7-12 (`04` dòng 83 trái D7), D-G7-13 (7 ca lộ: đã loại), D-G7-14 (gấp ký tự nhìn giống cho câu hỏi) | `docs/CHO-QUYET-DINH.md` |

## Bằng chứng đã chạy (20/10/2026)

> Số đo của mục này là ngày 20/10. Mốc hiện hành (lệnh + số phải ra): `docs/ops/README.md` §3.

```bash
docker compose -f deploy/docker-compose.yml up -d --wait          # PG 17.11, ghim digest, chỉ 127.0.0.1:25432
export EAIS_PG_ADMIN_DSN='postgres://postgres@127.0.0.1:25432/eais?sslmode=disable'
export EAIS_PG_DSN='postgres://eais_app@127.0.0.1:25432/eais?sslmode=disable'
go vet ./...                       # sạch
gofmt -l .                         # rỗng
go test ./... -count=1             # 224 PASS / 0 FAIL / 0 SKIP (có PG, đo 10:35). Không có biến PG: 5 test PG SKIP
scripts/scale-smoke.sh             # SCALE SMOKE: ĐẠT
go run ./tenants/construction/win/gen/cmd/gen -world tenants/construction/win/world.yaml -out /tmp/win -profile functional -seed 42
go run ./tenants/construction/win/winreport -data /tmp/win -v       # đường bộ nhớ
go run ./tenants/construction/win/winreport -data /tmp/win -v -pg   # đường PostgreSQL (RLS)
```

`TestWinRatchet` biến bộ ca WIN thành cổng hồi quy của `go test`. Nó đòi đủ bốn điều:
- G1 khớp manifest;
- metamorphic ≥ 4 biểu diễn mỗi bản ghi;
- mọi ca trong `passing.txt` (117/117) vẫn đạt;
- phát lại 0 lệch.

## Đã có

| Vùng | Nội dung | Bằng chứng |
|---|---|---|
| `core/contracts` | kiểu dùng chung; `Outcome` có kiểu; claim đủ 6 chiều; bằng chứng mang chủ thể / chỉ số / kỳ; tenant trong ngữ cảnh (`WithTenant`) | test fault + test đầu–cuối |
| `core/pipeline` | lối vào QUERY S0–S10 cho 7 route WIN. S1 gắn tenant của Principal vào ngữ cảnh. S9 có 4 lớp kiểm số liệu, 4 lớp kiểm tri thức. Có bản ghi quyết định + `Replay` | `tenants/construction` (đầu–cuối) |
| `data/normalization` | số kiểu Việt (cả số âm); đơn vị; thuật ngữ; viết tắt; ngày tương đối; luật dấu; `ToneKey` ("hoá" = "hóa", "bí" ≠ "bị") | vector NRM-V-001…049 |
| `data/entityresolution` | bí danh dài nhất, mã theo nguồn, mơ hồ ⇒ ứng viên, vị trí byte của cụm | `resolver_test.go` |
| `data/sourceoftruth` | thẩm quyền trong từng chỉ số, xung đột / ngang hạng, dẫn xuất, so hai chỉ số (`Compare`); thứ tự bằng chứng tất định | test + thuộc tính |
| `data/ingestion` | nạp CSV / XLSX / JSON / JSONL / PDF văn bản theo `sources.yaml`; cách ly có lý do | `ingest_test.go`, G1 |
| `intelligence/intent`, `router` | ý định theo luật, khớp dài nhất, ý định yếu, D7, mệnh đề trọng tâm, so sánh hai nghĩa, ghép RAG / HYBRID; quyền đóng mặc định | test + đầu–cuối |
| `intelligence/rag`, `models` | chỉ mục tri thức trong bộ nhớ (tenant + ACL theo vai); gateway model theo cấu hình; model giả tất định có đếm lời gọi | `rag_test.go`, `models_test.go` |
| `reliability/validation` | `Fidelity` (lớp lọc đầu), `BindClaims` + `CountNumbers` (6 chiều, bộ đếm độc lập), `Bound`, `ForbidDimensions` + `ForbidUnitWords` | bảng sai đúng một chiều |
| `platform/pgstore`, `db/migrations` | PG17: RLS FORCE theo GUC `eais.tenant_id`, vai `eais_app`, Load thay trọn dữ liệu tenant, ACL tài liệu lọc trong SQL | test PG + `TestWinTrenPG` |
| `observability/traces` | đo thời gian chặng; `Decision` + `Diff` | `decision_test.go` |
| `evals` | hợp đồng ca WIN, runner (chấm `model_calls`), đếm cổng G6 | `runner_test.go`, `win_test.go` |
| `tenants/construction` | pack thật; `win/`: world đóng băng, bộ sinh có seed (functional ~2,7 nghìn / smoke ~360 nghìn bản ghi), 117 ca độc lập, `winrun`, `winreport` | ratchet |
| `architecture_test.go` | mã dùng chung không mang tên tenant; kiểu `*Request` không mang tenant | đột biến G, TA, TB |
| `deploy/`, `scripts/` | compose PG; `mutate.sh`; `scale-smoke.sh` | — |

## Đột biến (20/10/2026)

Mỗi đột biến sửa sai một chỗ, chạy test, khôi phục, kiểm sha256 (`scripts/mutate.sh`). Danh mục chạy lại được: `scripts/dot-bien/` (69 đột biến, mọi chuỗi gốc khớp mã ngày 20/10).

| Nhóm | Số | Kết quả |
|---|---|---|
| quyền đóng mặc định (I, J) | 2 | bị bắt |
| nạp + resolver (K–O) | 5 | bị bắt; O lần đầu SỐNG SÓT → thêm ca |
| outcome / xác nhận (P–U) | 6 | bị bắt |
| ratchet (V–AC) | 8 | bị bắt; Y, AA lần đầu SỐNG SÓT → thêm ca |
| RAG / HYBRID (AD–AZ, BA, BB) | 25 | bị bắt; AZ lần đầu SỐNG SÓT (095/096 chưa vào ratchet) |
| PG (PA, PD–PH) | 6 | bị bắt |
| phát lại G6 (GA–GG) | 7 | bị bắt; GF lần đầu SỐNG SÓT → thêm `TestDiffMoiChang` |
| thuộc tính (PR1–PR3) | 3 | bị bắt |
| AST tenant (TA, TB) | 2 | bị bắt |
| ràng buộc claim (BC1–BC13) | 13 | bị bắt |
| so sánh / mệnh đề / số âm (CM1–CM4, FC1–FC2, NEG1–NEG2) | 8 | bị bắt |
| bộ đọc JSON (VM1) | 1 | SỐNG SÓT: đột biến sai chỗ (số lượng ghi dạng chuỗi). Thay bằng test âm trực tiếp cho bộ kiểm metamorphic |

Lọc ACL tài liệu có hai lớp, SQL và Go. Bỏ một lớp thì test vẫn xanh nhờ lớp kia. Đây là thiết kế phòng thủ hai lớp, không phải lỗ.

## Lỗi thật tìm ra hôm nay (đã sửa, đều có test)

1. **Thứ tự bằng chứng không tất định.** `sourceoftruth.Query` duyệt map nguồn nên thứ tự trích dẫn đổi giữa hai lần chạy. Phát lại G6 bắt được ở 17 ca.
2. **Bộ chuẩn hoá rơi dấu âm.** "-150 L" đọc thành 150, nên mọi hiệu âm của phép tính bị cổng S9 chặn khi giao. Test so sánh ngược chiều bắt được.
3. **Slice `nil` tới PG là NULL.** `cardinality(NULL) = 0` cho NULL, nên danh mục rỗng. Test PG bắt được.
4. **Truy xuất bỏ dấu.** "bí" khớp "bị", kéo ra tài liệu bị chèn lệnh (ca 072). "làm" khớp "làm việc", lấy nhầm bước (ca 116): **bộ ca không bắt lỗi này**, ca vẫn đạt. Đã khoá bằng test truy xuất.
5. **`Policy.Allows("")` mở mặc định.** Lỗi bảo mật, soát độc lập tìm ra. Đã sửa sáng 20/10.
6. **Truy xuất RAG không tất định.** Điểm cộng theo thứ tự duyệt map, nên đoạn hoà điểm đổi chỗ giữa các lần hỏi. Agent dựng baseline G7 đo được: 23/117 ca ra top-10 khác nhau qua 200 lần. Test lặp 200 lần đỏ trên mã cũ.

## Lỗ hổng của bảng 19/10 — trạng thái

| # | Lỗ hổng | Trạng thái |
|---|---|---|
| 1 | câu xác nhận "4k đúng không?" | XONG: so tuyệt đối với nguồn có thẩm quyền, kê số khác có nhãn; câu nhiều mệnh đề |
| 2 | thiếu thuật ngữ rơi về "ngoài phạm vi" | XONG: hỏi lại; D7 vật tư mặc định khi có thực thể |
| 3 | "dùng" là nghĩa nào | XONG: D3 = tiêu thụ đo được |
| 4 | mã theo nguồn (EX003, asset-8392…) | XONG: bí danh theo không gian tên nguồn |
| 5 | máy đổi dự án giữa kỳ | CHƯA, DEFER: chỉ mở khi một ca đỏ vì nó |
| 6 | câu chờ duyệt lộ mã nguồn | XONG: không đưa số, nhãn nguồn dễ đọc |
| 7 | kết quả có kiểu | XONG: `Outcome` |
| 8 | capability rỗng mở mặc định | XONG: đóng mặc định + luật lúc nạp pack |

Kiểm claim ở S9 không còn là "kiểm có mặt" đơn thuần. `BindClaims` đòi đủ chủ thể / chỉ số / kỳ / giá trị / đơn vị / nguồn. Route sinh số bằng model **vẫn chưa mở**: chỉ có model giả trích nguyên văn.

## Chưa có / chưa kiểm

| Hạng mục | Tình trạng |
|---|---|
| G7 với model thật | CHẶN: cần khoá + đồng ý chi phí. Client đã có (`models.Messages`, test bằng máy chủ giả), chưa từng gọi API thật. Tiêu chí chấm theo nội dung (`G7ContentReport`), top-k baseline = 10 và cách viết đoạn của baseline: chờ user duyệt |
| HTTP API, xác thực, `cmd/eais` | chưa có. Bản ghi quyết định mới nằm trong bộ nhớ / test, chưa lưu bền |
| Chỉ mục toàn văn PG | chưa có. Tài liệu lưu nguyên văn, cắt đoạn và chấm điểm trong Go lúc truy vấn; đủ cho WIN, chưa đo ở quy mô tài liệu lớn |
| Soát độc lập mã W4–W7 | CHƯA: mã hôm nay mới qua tự soát + đột biến |
| `go mod tidy` | chạy offline lỗi (cache thiếu dependency test của yaml). pgx v5.11.0 thêm từ cache; chạy lại khi có mạng |
| Thể thao; COMMAND / EVENT / WORKFLOW | DEFER theo dòng ADR-0001 ở `docs/CHO-QUYET-DINH.md` §0 và D-RM-04 (`docs/ROADMAP-DAY-DU.md` §4.3) |
| Tài liệu `docs/tenants/**`, `docs/requirements/**` | do agent viết; chưa soát độc lập; `phase` trong sổ yêu cầu chưa xếp lại theo WIN |

## Quan sát ngoài phạm vi (chưa sửa)

| Quan sát | Tác động | Đề xuất |
|---|---|---|
| `docs/02-migration-audit.md` nhắc đường dẫn nội bộ của hệ thống cũ (2 chỗ) | repo PUBLIC | user quyết trước commit: giữ, hoặc thay bằng "hệ thống cũ" |
| "Zalo" xuất hiện 9 chỗ (`01-architecture.md`, `docs/tenants/construction/01-business-map.md`, sổ yêu cầu) | tên kênh chat của đội công trường, không phải tên dự án cũ | chấp nhận |
| Nhóm lỗi P10 trong `CLAUDE.md` khác CONVENTIONS §4.2 | hai danh sách lệch | chốt một danh sách |
| Sổ yêu cầu dán nhãn 294 REQ là "MVP" | rộng hơn hẳn WIN | xếp lại theo WIN / Phase 1–10 |
| `docs/tenants/sports/08-test-cases.md` có thể bị sinh lại nhầm lúc 20:49 ngày 19/10 | chưa chứng minh không mất sửa tay | chạy lại generator thể thao rồi diff |
| Bộ ca cầu đường cũ: số ca theo loại lối vào lệch mục tiêu | thiếu ca lệnh / workflow | bổ sung khi làm Phase 1–2 |
| Pack khai `reported_vs_issued` (dẫn xuất) nhưng không ý định nào dùng | thừa; câu so sánh đã đi đường `Compare` chung | giữ hoặc bỏ khi soát pack |

## Việc tiếp theo (20/10 — đã thay)

> **Việc hiện hành (tối 21/10)** — lịch sử, đã thay bởi khối "MVP" 22/10 ở đầu tệp. Thứ tự theo `docs/PROMPT-BUILD-EAIS.md` §4, `docs/PROMPT-DOI-TEN-VA-LUONG-MOI.md` và D-OP-13:
> 1. đổi tên lại EAIS (D-RM-02): XONG — `945777c` … `cdf7d84`; soát độc lập 3 góc: 0 S1, 3 S2 đã sửa;
> 2. viết lại luồng theo tên mới (`docs/ROADMAP-DAY-DU.md`, `docs/01-architecture.md` theo 7 mặt phẳng + L0–L10, `CLAUDE.md` §9): XONG — `52c9399`, `34240a9`, `bff56c4`, `9da639c`, `903623d`; tệp phản hồi đổi tên `1ccac76`;
> 3. đóng Phase 0b: R4-2…R4-4, R5-1, R5-2 XONG (Vòng 5); còn R5-4, R5-5 → vòng sửa S4 theo ADR-0010 (điều tra xong 22/10; D-OP-15 chờ) → chạy H4 MỘT lần;
> 4. phép đo (harness canary, cờ ước lượng, CI có PG); giấy Phase 0a và Phase 1 (mã COMMAND / WORKFLOW / EVENT chờ D-RM-04); Phase 6 nếu S4 vẫn là sàn; tài liệu lệch, handoff;
> 5. phản hồi audit ROADMAP: K1–K15 XONG (K6–K15 ở bản giữa chừng `docs/bao-cao/2026-10-21-phan-hoi-roadmap.md`, K1–K5 ở `docs/bao-cao/2026-10-21-phan-hoi-roadmap-k1-k5.md`); còn Q2–Q7, Q9–Q13, Phụ lục C.
>
> Câu chờ quyết: `docs/CHO-QUYET-DINH.md` (mới từ 21/10: D-RM-03, D-RM-04, D-RM-05, D-OP-11, D-OP-12, D-OP-14, D-OP-15; đã quyết: D-RM-01, D-RM-02, D-OP-13; danh sách đủ ở §0 của tệp đó).

> Thứ tự của đêm 20→21/10 (bảng ngay trên):
> 1. đóng băng bộ giữ lại;
> 2. chứng minh lại G2 / G5 / G6 trên H1;
> 3. bộ chấm v2 (ADR-0007) + test từ 170 đòn;
> 4. dự toán chi phí;
> 5. hợp đồng G7 sẵn sàng đóng băng sau pilot.
>
> Pilot và lượt G7 thật cần khoá do chủ môi trường đặt. Các mục dưới là danh sách ban ngày 20/10, giữ để đối chiếu.

1. Chủ dự án soát các commit ngày 20/10 trên GitHub. Repo PUBLIC: thứ cố ý không lên git ở `docs/ops/README.md` mục 7.
2. **G7:** làm theo `docs/g7/CHECKPOINT-A-H.md`:
   - chủ dự án quyết D-G7-1…11 (sửa D5 vì temperature 0 không chạy được, model, trần chi phí, B1, bộ giữ lại, đóng băng hợp đồng, cấp khoá, gói đề xuất của bản soát, câu đồng nghĩa "dùng", 3 kỳ vọng trọng tài);
   - làm các blocker mã (client, `-smoke`, `-out`, B1);
   - thử khói, rồi chạy k = 3.
   Đây là cổng duy nhất còn lại của WIN.
3. Soát độc lập (phiên không viết mã) cho mã hôm nay: pipeline S9, pgstore, so sánh / mệnh đề trọng tâm. Agent soát đã dừng 10:33 theo lệnh, chưa có phát hiện.
3b. Bộ ca đội đỏ (kế hoạch cũ ≈ 40 ca; nếu làm bộ giữ lại G7 thì ≥ 96 ca theo D-G7-5 / D-OP-5; do người không đọc mã soạn, ngoài bộ đóng băng) để đo tổng quát hoá. Hiện CHƯA có bằng chứng tổng quát hoá ngoài 117 ca. Agent dừng 10:33, chưa ghi tệp.
4. Lưu bền bản ghi quyết định (bảng `interactions`) khi có HTTP API.
5. DEFER theo `05`: tác vụ duyệt / lệnh (Phase 1–2), thể thao (Phase 4), lỗ hổng 5.
