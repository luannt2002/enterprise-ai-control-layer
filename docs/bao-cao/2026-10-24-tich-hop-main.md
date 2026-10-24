# Báo cáo tích hợp 24/10/2026 tối — main mới nhất có gì, đã làm gì, tiến độ ra sao

> **Viết cho:** chủ dự án và phiên BUILD.
>
> **Quyết định:** D-RAG-05 (`docs/CHO-QUYET-DINH.md`) — chủ dự án: "merge hết về main, main là đồng bộ hết mới nhất… chạy lại test, debug,
> update doc / report / state".
>
> **Main:** `5ea3ae3 → 4fe8e0b` (đẩy nhanh, không mất commit nào), rồi các commit tài liệu của đợt này. Mọi nhánh trên GitHub đều nằm trong main.
>
> **Nhãn:** [ĐO] = lệnh chạy trong đợt này · [REPO] = tài liệu repo ghi.

## 1. Main mới nhất có gì (theo luồng)

| Luồng / khối | Mới vào main đợt này | Trạng thái soát của chính việc đó (trước khi gộp) |
|---|---|---|
| **L2 S4** (M3) | vòng sửa S4 theo cấu trúc (ADR-0010; D-OP-18 = c, D-OP-11 = b), nghĩa chưa hỗ trợ (tiền, dự báo, báo giá, cảnh báo, lệnh ghi, đánh giá), số ngoài miền; nợ S4 sau H4 (S4R-03) | **KHOÁ sau vòng 3, KHÔNG ĐẠT** [REPO, D-AI-05]: S4CS-02 hồi quy S1 (7 câu nền hỏi lại nay trả số không nhãn), S4CS-01, -03, -04 là nợ ⇒ gộp hết = D-AI-05 phương án (a) |
| **M1 vết** | vết S8 / S9 / S10 (bản nháp, dấu vân tay câu giao, lối thoát sớm); che bí mật 5a vòng 5 | S8–S10 soát vòng 1 KHÔNG ĐẠT (V8AS-01, -02) [REPO]; 5a vòng 5 đang soát [REPO] |
| **L1 Phase 0a** (M2) | bước 2–3b + kho PG `execution/reports`, migration **0009_kho_bao_cao_ngay** (đổi từ 0006 — số đã thuộc sổ nạp); giấy 0a giữ KHOÁ vòng 6, nháp vòng 7 ở `docs/viec/0a-thu-bao-cao-ngay/nhap-vong-7/` | soát KHÔNG ĐẠT hẹp: **R-01 chết khoá (40P01)** khi nạp chạy cùng quyết định trên PG, đang sửa [REPO]; gói chưa nối vào `cmd/eais` |
| **L6 RAG** | (đã vào main sáng 24/10, D-RAG-01) + móc dự phòng S4 (mặc định TẮT), client model quyết định có kiểu và thí nghiệm S4 / chấm đoạn (chỉ ở bộ đo) — D-RAG-04 | soát độc lập 2 vòng; số ở `docs/ROADMAP-RAG.md` 10.2b |
| Vận hành | `scripts/cong/` (cổng có PG so theo tên, winreport cũ / mới, quét / chấm ngưỡng); `docs/viec/2-tra-quy-trinh/CACH-LAM-PHIEN-RAG.md` | — |

## 2. Đã làm gì — cách gộp từng nhánh

- **Gộp nội dung:** `m1/gop-5a-v5`, `m1/vet-s8-s10`, `m3/s4-vong-sua-2310`, `lead/no-s4-sau-h4`, `m2/0a-ma-truoc-dong-bang`,
  `m2/phase0a-vong7-bo` (giấy giữ vòng 6), `rag/1-quyet-dinh`.
- **Ghi nhận, cây không đổi** (`git merge -s ours`) — 13 nhánh:
  - 7 nhánh có mọi commit đã nằm trong main theo patch-id: 5e, 5b, 5bc, C8–C11, M4 K3, M4 C1–C7, M5 dấu ≈;
  - các bản 5a cũ, 5c sớm và `thu/nhom-nghin` (đã chọn D-OP-9 (a')).
  - Soát độc lập so TỪNG DÒNG (không chỉ tiêu đề): 0 việc bị mất.
- **Giải xung đột:**
  - mọi khối xung đột đều là hai bên cùng thêm vào một chỗ;
  - bảng từng khối: `docs/debug-handoff/2026-10-24-01-giao-nhau-rag-voi-nhanh-team.md`.
  - Kỳ vọng test của nhánh S4 nêu tên thực thể ngoài phạm vi người hỏi (viết trước lớp che C9) ⇒ sửa theo bất biến che.
- **Không test nào bị rơi** [ĐO]: so tên mọi hàm `Test*` / `Benchmark*` của 7 nhánh gộp nội dung và 13 nhánh ghi nhận với cây gộp ⇒ 0 thiếu.

## 3. Soát độc lập sau gộp (3 agent chỉ đọc) — lỗi bắt được và đã sửa

Cách làm: viết test đỏ trước (`tenants/construction/giao_nhau_2410_test.go`); bỏ từng chỗ sửa thì test đỏ (đột biến).

| Mức | Lỗi | Nguồn | Đã sửa |
|---|---|---|---|
| P1 | câu NÊU giá trị gõ không dấu "may xuc 03 … 3782 L dau": "may" (= máy) khớp chữ hỏi lượng "mấy" ⇒ trả số không nhãn như đã ghi (trái D-OP-18 = c) | vòng sửa S4 thêm "mấy" vào `quantity_markers` | S4 không tính chữ trong tên thực thể khi xét câu nêu giá trị |
| P2 | "lệch dầu thì mấy tiếng phải lập biên bản?" bị chốt lượng của route tri thức chặn oan (đúng dạng S7D-01 / 02) | giao nhau "mấy" (S4) × chốt lượng (RAG) | chữ hỏi lượng ngay trước đơn vị của chiều đo khác (lexicon: giờ, tiếng…) không tính |
| P2 | dự phòng S4 (B1, đang tắt) trả lời bằng tài liệu câu "Nhắc tôi khi…", "Dự báo…" (nghĩa chưa hỗ trợ, không có thực thể) | RAG × nghĩa chưa hỗ trợ | dự phòng không hỏi khi câu có cụm nghĩa chưa hỗ trợ của pack |
| P2 | vết: model từ chối ⇒ S10 nói sai "câu giao chứa bản nháp"; thông báo phiên bản không ghi S8 (ô trống bị giấu); HYBRID lỗi thì lỗi biến mất khỏi vết | vết S8–S10 × lối ra của RAG | ghi đúng chặng (S6 / S8 / S10) |
| P2 | test kiểm che bí mật duyệt danh sách chặng viết tay, thiếu S6 / S8 / S10 | — | duyệt mọi chặng bằng phản chiếu |
| P2 | 5 neo đột biến gãy (IV1, IV5, XV26, AV4, S150) | đổi mã sau gộp | khớp lại đúng 1 chỗ; cả 5 bị test bắt |
| P2 | `win-so.sh` chạy binary cũ trên pack mới ⇒ mọi dòng "khác" | lỗi của chính script | mỗi bên chạy trong cây của nó |

Không có P0, không có đường lộ bí mật hay vượt quyền [soát]. Ghi nhận `-s ours` không làm mất việc [soát].

## 4. Số đo trên main

| Phép đo | Kết quả |
|---|---|
| Cổng có PG, so theo tên (`scripts/cong/`), trên đúng `4fe8e0b` đã đẩy [ĐO] | **2398** test theo tên (2395 pass, 3 skip), 0 mất, 0 đổi, 5 mới (test giao nhau), **0 đỏ** — so với cây gộp trước các bản sửa |
| Trên cây gộp trước các bản sửa ở mục 3 [ĐO] | 2393 test theo tên, 0 mất, 0 đổi, 309 mới, 0 đỏ |
| winreport trên `4fe8e0b`, so main cũ `5ea3ae3` (bộ nhớ = PG) [ĐO] | 117/117 (0 dòng khác); H1 91 → **96**/104; H3 112 → **122**/134; H4 103 → **109**/119; **0 ca xanh → đỏ**, 21 ca đỏ → xanh. H1 / H3 / H4 là bộ ĐÃ công bố — không phải chấm độc lập |
| Smoke đầu–cuối trên `4fe8e0b`, PG riêng [ĐO] | 21 đạt, 0 hỏng |
| `go vet ./...`, `go test ./...` (không PG) trên `4fe8e0b` [ĐO] | sạch, xanh |

## 5. Tiến độ MVP (`docs/ROADMAP-DAY-DU.md` §5b) sau tích hợp

| Khối | Trước (23/10) | Sau tích hợp | Còn thiếu để RA |
|---|---|---|---|
| M1 | vế đọc vết 78,9% | mã vết S8–S10 + 5a đã ở main | đo lại người đọc vết (≥ 90%); soát vòng 2 S8–S10; soát 5a vòng 5 |
| M2 | mã 0a ở nhánh, bộ 0a đã băm | mã 0a ở main (migration 0009) | sửa R-01 (chết khoá PG); nối `cmd/eais` + `Ready` biết 4 bảng `report_*`; chạy bộ 0a MỘT lần |
| M3 | S4 khoá ở nhánh, H6 chờ | S4 ở main kèm nợ S4CS-01…04 | sửa S4CS-02 (hồi quy S1) rồi chạy H6 MỘT lần — chủ dự án quyết |
| M4 | P1 153/157, "≈" đã sửa | không đổi | Q-P1-14, Q-P1-15 |
| M5 | S2–S4, L1 KHÔNG ĐẠT | không đổi | các dòng của CỔNG TRƯỚC AI |

## 6. Còn mở — cần chủ dự án / team

1. **S4CS-02 (S1) nay ở main.** Chạy H6 ngay = chấm bản còn hồi quy. Sửa hẹp trước = phương án (b) của D-AI-05. H6 và bộ 0a là bộ giữ lại dùng MỘT lần: chưa chạy.
2. **R-01** (chết khoá PG của mã 0a, khi nạp chạy cùng quyết định) — đang sửa ở phía team.
3. D-RAG-02: 12 vai của ACL kho tri thức không có trong policy. D-RAG-03: bật dự phòng S4 B1 — sau tích hợp cần đo lại B0 / B1 trên bộ MỚI.
4. Migration tiếp theo: **0010**. DB dev đã áp `0006_kho_bao_cao_ngay` của nhánh cũ: chạy `eais migrate` vẫn được, vì 0009 chạy lại được an toàn; chỉ để lại một dòng thừa trong `schema_migrations`. (Đính chính thông điệp commit `71c6b45`.)
5. Giấy 0a vòng 7: bản nháp, không hiệu lực; mở lại là quyết định của chủ dự án.

## 7. Kiểm lại

```bash
scripts/cong/cong.sh . /tmp/cong.json && python3 scripts/cong/so_ten.py <mốc.json> /tmp/cong.json
scripts/cong/win-so.sh <cây main cũ> . /tmp/win
scripts/chay-thu.sh
go test -count=1 -run GiaoNhau ./tenants/construction/
```
