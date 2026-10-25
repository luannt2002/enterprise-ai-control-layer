# Báo cáo thi công vNext 24/10/2026 tối — đã làm gì theo lát, số đo, sổ nợ bền, câu chờ quyết

> **Viết cho:** chủ dự án và phiên BUILD.
>
> **Quyết định:** GO 24/10 của chủ dự án cho báo cáo audit vNext (`docs/CHO-QUYET-DINH.md` mục 0, dòng "vNext"): phương án B′ — cổng
> trước, tách TẠI CHỖ, dời thư mục để sau; làm L5 trước H6 (tệp ca H6 không có trên máy); giấy phép MIT; L7 (xác thực thật + CRM giả) để
> đợt sau. Báo cáo audit (nguồn sự thật của đợt): `docs/bao-cao/2026-10-24-vnext-audit/BAO-CAO.md`, phiếu phát hiện §10 (`VNEXT-F-0xx`).
>
> **Nhánh:** `vnext/tach-2410`, đi từ main `3f1bf7e`, đỉnh `a35f3d8`. **CHƯA gộp main.** Sau đỉnh còn hai lát sửa theo soát độc lập cuối,
> chạy song song: tài liệu (`vnext/sua-soat-docs`, có tệp này) và mã (`vnext/sua-soat-ma`).
>
> **Nhãn:** [ĐO] = lệnh đã chạy, ghi ai chạy · [REPO] = thông điệp commit / tài liệu repo ghi · [ĐỌC MÃ] = đọc mã, chưa chạy.

## 1. Đã làm theo lát

Lấy danh sách: `git log --oneline --first-parent 3f1bf7e..a35f3d8`. Cột "Soát" chép từ thông điệp merge [REPO]; dòng nào thông điệp
không ghi kết luận soát thì ghi đúng như vậy.

| Merge | Lát | Phát hiện đã đóng | Bằng chứng chính | Soát |
|---|---|---|---|---|
| `8045dae` | L2 — sửa hẹp S4CS-02 | F-001 (vòng 1) | 7/7 câu S4CS-02 không còn RESOLVED; 5/5 cặp tối thiểu giữ nguyên; winreport bộ nhớ 117 / H1 / H3 / H4 giống hệt `3f1bf7e` | 21/21 đột biến bị bắt, gồm X3, X4 của người soát |
| `0428293` | L3 — sự thật tài liệu | F-016 (MIT), F-020, F-033, F-038; D4–D9, D13, D14; ADR-0013 / 0015 / 0016 | 18 commit chỉ tài liệu + `LICENSE` + danh sách cho phép của cổng danh tính; không đổi hành vi mã | — |
| `0ba1da0` | R-01 — chết khoá kho báo cáo 0a | F-002 | khoá tư vấn theo bản ghi, cùng thứ tự tăng dần với khoá ô, thay `FOR UPDATE`; test tải 14.700 thao tác: mã cũ 133 lần 40P01 (điều phối đo lại), mã mới 0 | hai đột biến (bỏ khoá bản ghi / đặt lại `FOR UPDATE`) làm test đỏ |
| `3308b66` | L3 vòng 2 | F5, F6, F7 của người soát | nhãn lịch sử nói đúng chỗ đã đổi; số dòng import đo lại kèm lệnh (227 tệp, 578 dòng); tách lời chủ dự án khỏi suy diễn trong CHO và ADR-0013 / 0015 / 0016 | sửa theo soát độc lập |
| `a134412` | L1 — cổng trước | F-011, F-014, F-025, F-034; ADR-0014 | cổng kiến trúc quét MỌI thư mục + cổng từ miền với danh sách nợ khớp chính xác; `so-than` so THÂN test giữa hai cây; danh sách SKIP cho phép có lý do; đặt lại 5 neo đột biến gãy | điều phối cố tình phá: thêm `modules/x` → đỏ; thiếu mục SKIP → rc=1 |
| `af1f24e`, `a34549f`, `e989f56` | khai đổi thân có chủ ý | F-034 | `scripts/cong/doi-co-chu-y-3f1bf7e.txt`: mỗi test đổi thân có lý do; 30 dòng của L4 (bỏ tên hãng, thay 1:1) | cổng danh tính bắt một lần nhắc tên cũ trong tệp khai → sửa (`a34549f`) |
| `20434d2` | L4 — `core/policy`, bỏ tên hãng | F-031, F-015, F-035 | `core/policy` chỉ phụ thuộc `core/contracts`; `execution/commands` không còn kéo `intelligence/` (`TestRanhGioiImport`); provider theo GIAO THỨC `messages`, endpoint / header từ `.env`; `g7run` đọc model / endpoint / bảng giá từ `.env`; cổng tên hãng so băm từng từ | điều phối phá thử: chèn tên hãng → đỏ. Lỗ hai vai Q-P1-10 đo được, để Skip kèm mã câu hỏi (sửa ở Q11) |
| `ecf0bdb` | gia cố PG | F-010, F-036, F-037 (migration); ADR-0017 | `Ready` kiểm RLS bật + FORCE + chính sách tenant trên 11 bảng; `schema_migrations` có checksum, migrate dừng khi tệp đã áp bị sửa / đổi tên; 0010 thêm CHECK `tenant_id <> ''` cho 8 bảng | điều phối chạy lại `TestReadyDongKhiRLSTat` có PG: PASS (không SKIP) |
| `f952f6e` | L6 — ranh giới RAG | F-017, F-018, F-019, F-022, F-023, F-024, F-028, F-037 (phần mã) | nạp chỉ tài liệu; `DELETE` của Load lọc `tenant_id`; nạp lại không hạ bản đang công bố, báo cáo nạp đếm bản bị giữ; chữ bản chưa duyệt không tới embedder / reranker; `PruneVectorSpaces`; `examples/01-knowledge-base` trung tính | S7 bộ phát triển + `TestCaTriThuc` giống từng câu. F-021 để lại (cần quyết, mục 4) |
| `a692525` | L2 vòng 2 | F-001 | cắt ở "thì", phủ định LIỀN trước, lệnh đứng đầu mệnh đề | 9 câu soát vòng 1 nêu nay hỏi lại CÓ NHÃN; 31/31 đột biến bị bắt |
| `3d5c33d` | Q11 — cùng một vai | Q-P1-10 = A (SUY DIỄN của điều phối) | RAG và dự phòng S4 chỉ khớp ACL / phạm vi bằng vai có capability đọc; bỏ nhánh dự phòng phạm vi của RAG; C4 của lệnh đòi capability + `metrics.read` + phạm vi trên cùng một vai | test hai vai của L4 bỏ Skip, điều phối chạy lại: PASS; S7 41/41 câu giống |
| `d0efb3e` | L2 vòng 3 | F-001 | lời nhờ không biến lệnh thành câu hỏi | 6 câu soát vòng 2 nêu nay hỏi lại có nhãn; 38/38 đột biến bị bắt |
| `0fbb164` | L5 — Fuel xuống case study | F-004, F-007, F-009; F-005, F-008, F-029 một phần; F-028 (pack thứ hai) | lược đồ danh mục do pack khai; bản ghi trùng vào cách ly; route chưa cài bị từ chối lúc nạp; pack thứ hai `examples/02-sme-minimal` chạy S0–S10, commit đưa nó vào chỉ đụng `examples/` | đo chạy lùi: pack thứ hai cần 7 tệp lõi; nợ chữ miền 29 → 14 |
| `6fb1e90` | KHOÁ S4CS-02 | F-001 (khoá) | rút "là" khỏi từ đóng vế điều kiện | sửa hồi quy S1 mà soát vòng 3 bắt; nợ LỌT N8 giữ ở test Skip có lý do |
| `34b0fa4` | dọn mã | F-035 (đường log), F-015 (tên hãng còn sót), F-018 (CLI / log nạp) | log thân lệnh bị từ chối chỉ ghi LOẠI lỗi; dòng NẠP nói số bản bị giữ; đột biến R-01 (M35, M36) và m4 (M2, M4, M16 trên database dựng lại sau F-036) | thông điệp merge không ghi kết luận soát |
| `0470094`, `11872ad` | Q11b — đường số liệu S5 | Q-P1-10 phần S5 | router xét phạm vi bằng vai có capability của ý định; ứng viên hỏi lại, bộ che phạm vi, dòng S3 dùng vai có `metrics.read`; `CapMetricsRead` một nguồn ở `core/policy` | đo của lát (mục 2) |
| `0e43aff`, `a35f3d8` | L5b — từ miền ra pack | F-030, F-005 (phần còn lại), F-008 (phần còn lại, migration 0011) | danh từ nhãn / đếm ra lexicon pack; lời giải thích kiểu cha theo pack; cột `facts.note`; `Ready` đòi cột đó (`a35f3d8`, test đỏ trước) | 20/20 đột biến bị bắt; L5b phát hiện S209 không tương đương, đã khoá bằng test. Lệch yêu cầu ban đầu điều phối chấp nhận: `Options.Labels` rỗng ⇒ chỉ từ chức năng |

Phát hiện CHƯA thuộc đợt này: F-003 (xác thực chỉ có dev) → L7, đợt sau; F-006 (HYBRID thiếu quyền tài liệu từ chối cả câu) chờ câu Q7;
F-039, F-040 là lịch sử (REPO_CLAIM), không có việc mã. Các mục còn mở khác ở mục 3.

Soát độc lập cuối trên `a35f3d8`: KHÔNG ĐẠT vì một S2 (mốc chống tự chấm của G7 chuyển sang `.env`) và vài S3 / S4 tài liệu; KHÔNG thấy chỗ
nào nới quyền / RLS, winreport `3f1bf7e` ↔ `a35f3d8` khác 0 dòng, 4 lần phá thử cổng đều đỏ. Các mục được sửa ở hai lát sau:

| Merge | Lát | Đã sửa | Bằng chứng |
|---|---|---|---|
| `4e0054b` | tài liệu theo soát cuối | S3 tài liệu vận hành cấu hình model; S3 ADR-0017 đính chính comment 0010; S3 ADR-0016 ghi mục chưa làm; S4 CHO Q-P1-10; báo cáo này | chỉ `docs/`, README, CHANGELOG |
| `86eee70` | mã theo soát cuối | **S2**: mốc hợp đồng G7 ghim sha256 trong mã (`.env` trỏ máy giả không còn thành bằng chứng); S4 `DELETE` của `EmbedKnowledge` lọc `tenant_id`; loại SKIP "nợ đã khoá có mã ADR"; 8 lý do đổi thân L4 viết đúng. **Bổ sung**: G7 qua giao thức khác `messages` bị từ chối trước lời gọi model | S2: 7/7 đột biến bị bắt; `chat_completions`: đỏ trước đo được **124 lời gọi model không trần ngân sách** (httptest đếm), sau sửa mã 2, 0 lời gọi |
| `2f058d7` | điều phối | `docs/g7/CHI-PHI-G7.md` mục 8 khớp mã ghim băm cuối; `.env.example` ghi `EAIS_G7_*` phải khớp băm | cổng danh tính / tên hãng xanh |

## 2. Số đo do điều phối chạy

| Phép đo | Kết quả |
|---|---|
| Cổng có PG trên `a35f3d8`: `scripts/cong/cong.sh` [ĐO, điều phối] | **2.475** tên test, 2.471 pass, 4 skip, **0 fail**; `kiem-skip` XANH (4 SKIP có lý do: 3 eval cần động cơ ngoài + nợ S4CS-02-N8); `so-than` XANH |
| So tên với mốc có PG `3f1bf7e` (2.398 tên) [ĐO, điều phối; lát này chạy lại `so_ten.py` trên cùng hai tệp: khớp] | mất **8** (subtest của `TestEndpointTonTienVietKhacVanBiKiemTruoc` từng mang tên endpoint của hãng — đổi tên khi bỏ tên hãng, pass → pass), đổi **0**, mới **85** |
| Test không PG trên `0e43aff` [ĐO, điều phối] | 2.093 tên, 1.880 pass, 213 skip, 0 fail |
| Smoke `scripts/chay-thu.sh` trên `a35f3d8` [ĐO, điều phối] | 21 đạt, 0 hỏng |
| winreport bộ nhớ `3f1bf7e` ↔ `a35f3d8` [ĐO, người soát cuối] | 117 / H1 / H3 / H4: khác **0** dòng |
| R-01 [ĐO, lát R-01 + điều phối đo lại] | 14.700 thao tác: mã cũ 133 lần 40P01, mã mới 0 |
| Q11b, người hai vai [ĐO, lát Q11b] | mã cũ: 328 câu khác + 433 chỗ lộ trên 483 câu; mã mới: 0 (y hệt vai đủ quyền một mình) |
| Pack thứ hai [ĐO, lát L5, chạy lùi] | cần 7 tệp lõi |
| Nợ chữ miền trong lõi (danh sách của cổng từ miền) [REPO, L5 / L5b] | 29 → 14 → **4** mục |
| Cổng SKIP của CI | trước đỏ (rc=1 trên `3f1bf7e`), nay xanh. CI vẫn CHƯA chạy trên GitHub |
| **CHỐT — cổng có PG trên `2f058d7`** (đỉnh nhánh lúc đẩy) [ĐO, điều phối] | **2.482** tên, 2.478 pass, 4 skip, **0 fail**; `kiem-skip` XANH; `so-than` XANH; so tên với mốc có PG `3f1bf7e`: **mất 0**, đổi 0, mới 84 (8 subtest g7run lấy lại tên cũ: tên đọc từ hợp đồng lúc chạy, mã không chứa tên hãng) |
| **CHỐT — smoke** `scripts/chay-thu.sh` trên `2f058d7` [ĐO, điều phối] | 21 đạt, 0 hỏng; binary nhúng `code_version=2f058d7…` (cây sạch) |

## 3. Sổ nợ bền

Chiều sai: **an toàn** = sai về phía từ chối / hỏi lại / lỗi ồn · **LỌT** = trả lời sai hoặc lộ mà không ai báo.

| Mã | Chiều sai | Nơi ghi | Cách gỡ |
|---|---|---|---|
| S4CS-02-N1…N9 (N7 đã đóng lúc khoá) | N1–N5, N9 **an toàn** (hỏi lại thừa); N6, N8 **LỌT** — có từ trước L2 (`3f1bf7e` cũng trả số) | ADR-0010 bảng nợ S4CS-02; N8: test Skip `TestS4CS02NoN8LaDongVeDieuKien` + `scripts/cong/skip-cho-phep.txt` | vòng sửa S4 sau H6; H6 sẽ chấm mã còn các nợ này |
| F-021 công bố tài liệu + vector không nguyên tử | an toàn (thiếu vector ⇒ lượt lỗi, không lộ); chỉ đường lai, hiện chỉ bộ đo chạy | phiếu F-021; merge `f952f6e` | chủ dự án quyết (câu 2, mục 4) — mọi cách sửa đảo một hợp đồng đã có test |
| `Ready` không kiểm `document_vectors`, `report_*` | an toàn (thiếu bảng lỗi ở lượt đầu thay vì lúc khởi động); app chưa dùng `report_*` | `platform/pgstore/ready.go`, `readyNeeds` (11 bảng) | thêm vào `readyNeeds` cùng commit nối reports / đường lai vào app |
| Ghi chú suy luận chưa vào `Evidence` | **LỌT** nhãn: số suy luận lúc nạp có `facts.note` (0011) nhưng `contracts.Evidence` không có trường, câu trả lời không nói số đó là suy luận | F-008 phần còn lại; `core/contracts/contracts.go`, kiểu `Evidence` | thêm trường + S9 kiểm nhãn; test đỏ trước bằng một fact có `note` |
| Vết `/v1/ask` còn lọt dạng `password=\"…` | **LỌT** (bí mật trong vết) | soát độc lập cuối; giới hạn regex của bộ che | sửa cần đổi phản hồi client — cần quyết cùng người dùng API |
| `Decide` của reports chưa kiểm capability; gói reports chưa nối `cmd/eais` | hôm nay chưa có đường gọi từ app; nối vào mà chưa sửa là **LỌT** quyền | merge `0ba1da0` | đổi API `Decide` nhận principal + policy, rồi mới nối |
| Đường dữ liệu dùng tên quyền cố định `metrics.read` | an toàn (pack đặt tên khác ⇒ mọi lượt số liệu bị chặn) | `core/policy/policy.go:18`, `CapMetricsRead` | pack khai capability của đường số liệu |
| F-012, F-013, F-027 | hiệu năng / vận hành, không sai dữ liệu; **chưa đo** | phiếu F-012 (tắt server khi đang nạp), F-013 (đọc danh mục nhiều lần mỗi lượt), F-027 (dấu nạp quét cả bảng mỗi câu) | đo trước (benchmark), sửa khi số đòi |
| F-026, F-032 | DEFER | phiếu F-026 (ngân sách token ngữ cảnh), F-032 (adapter PG gom RAG) | F-032 đi cùng việc dời thư mục |
| Dời thư mục `modules/` | HOÃN theo B′ | ADR-0014, ADR-0015 | lát riêng, sau khi các cổng đã đỏ được |
| L7 (xác thực thật + CRM giả, F-003) | — | lời chủ dự án 24/10 | đợt sau |
| Tên hãng trong docs lịch sử | không ảnh hưởng chạy; cổng `TestKhongTenHangTrongMa` bỏ qua `docs/` | [ĐO, lát này] trên `a35f3d8`: 35 tệp, 267 từ trúng danh sách băm của cổng tên hãng (cùng cách tách từ); sau lát này 33 tệp, 264 từ | chủ dự án quyết (câu 3) |
| Đột biến S209; neo gãy có sẵn | S209 từng sống sót, đã khoá (`tenants/construction/l5b_test.go`). **Mới nghi:** toàn bộ danh mục `scripts/dot-bien/` chưa chạy lại trên `a35f3d8`, có thể còn neo gãy | `scripts/dot-bien/n45-y-dinh-s4.sh`; ADR-0014 mục 5 | chạy mọi danh mục trên bản clone riêng, đếm dòng "neo gãy" (rc=2) |
| Đường ĐỌC tài liệu / vector không lọc `tenant_id` dưới vai vượt RLS (`readDocuments`, `knowledgeIndex` ở `pgstore.go`; `SELECT` ở `vector.go` ~27; `loadVectors`) | an toàn với app (`pgstore.Open` từ chối vai superuser / BYPASSRLS); **LỌT** chỉ khi một công cụ dùng pool quản trị. **Mới nghi:** `EmbedKnowledge(A)` dưới vai quản trị có thể nhúng chữ của tenant B rồi ghi dưới A | lát sửa mã (soát cuối) đo khi viết `TestNhungChiXoaVectorCuaTenantKhiVuotRLS`: bản đầu xanh oan vì `readDocuments` đọc cả B | thêm `WHERE tenant_id` cho các câu đọc (phòng thủ chiều sâu, như `load.go`); test đếm `document_vectors WHERE tenant_id=A` sau khi A nhúng |
| H6 + bộ 0a | **BLOCKED**: tệp ca không có trên máy | ADR-0010; `docs/CHO-QUYET-DINH.md` | chủ dự án chọn "làm L5 trước H6", nên H6 chấm mã đã tách; chạy MỘT lần |

**Đã chứng minh, đã sửa (`75e8197`, merge `86eee70`):** `g7run` với `EAIS_MODEL_PROVIDER=chat_completions` trước đây không bọc trần ngân
sách và không qua kiểm trước khi tiêu tiền — đo đỏ trước: một lượt k=1 gọi model **124 lần** (có từ trước `3f1bf7e`). Nay giao thức khác
`messages` bị từ chối ngay sau khi dựng client: mã 2, 0 lời gọi (`TestGiaoThucNgoaiHopDongBiTuChoiTruocLoiGoiModel`, 2/2 đột biến bị bắt).

## 4. Câu cho chủ dự án

| # | Câu | A | B | Khuyến nghị |
|---|---|---|---|---|
| 1 | Q-P1-10: quyền và phạm vi / ACL lấy từ CÙNG một vai (Q11, Q11b đã làm theo A — SUY DIỄN từ GO chung) | duyệt A | đảo về hợp các vai: hoàn tác `3d5c33d`, `0470094` | **A**: đóng mặc định; mã cũ lộ 433 chỗ trên 483 câu với người hai vai |
| 2 | F-021: tài liệu + vector công bố không nguyên tử | nhúng TRƯỚC, ghi vector trong giao dịch của Load (nạp phụ thuộc embedder) | đoạn chưa có vector đi đường từ khoá + dòng vết (đảo kỳ vọng `TestVectorTriThucPG`: "kho chưa nhúng" ra lỗi) | **hoãn** tới khi đường lai vào app (hôm nay chỉ bộ đo chạy nó); lúc đó chọn **A** — giữ đóng mặc định |
| 3 | Tên hãng trong docs lịch sử (35 tệp / 267 từ trên `a35f3d8`) | giữ nguyên tài liệu lịch sử; thêm cổng cho docs hiện hành với danh sách cho phép theo tệp, hạn mức đúng bằng số lần (như `legacyAllowed`) | thay bằng chữ trung tính ở mọi tệp | **A**: B sửa bằng chứng lịch sử và tệp đã băm (hợp đồng G7 được băm vào `manifest.json` của mỗi lượt) |
| 4 | Tệp ca H6 (đã băm ở `6734cff`) ở đâu | chủ dự án chép tệp ca vào máy chạy, chạy H6 MỘT lần trên đỉnh đã gộp | soạn bộ mới bởi người soạn khác, băm trước | **A** |
| 5 | Gộp `vnext/tach-2410` vào main khi nào | sau khi hai lát sửa theo soát cuối gộp, cổng có PG xanh, và CI chạy một lần trên GitHub | chờ H6 chạy xong | **A**: H6 chấm đúng mã sẽ lên main; CI chưa từng chạy trên GitHub là lỗ đáng đóng trước khi gộp |
| 6 | `CLAUDE.md` còn ba chỗ lời cũ (§9, danh sách tenant, §25) | chủ dự án sửa theo phụ lục | giữ nguyên | **A**: `CLAUDE.md` là hợp đồng của chủ dự án, agent không sửa |

## 5. Kiểm lại

```bash
git worktree add --detach /tmp/moc-3f1 3f1bf7e                           # cây mốc, đường tuyệt đối
CONG_MOC=/tmp/moc-3f1 CONG_DOI_CO_CHU_Y=scripts/cong/doi-co-chu-y-3f1bf7e.txt \
  scripts/cong/cong.sh . /tmp/cong.json                                   # go test có PG + kiem-skip + so-than; rc=0 khi cả ba xanh
python3 scripts/cong/so_ten.py <json có PG của 3f1bf7e> /tmp/cong.json   # mất / đổi / mới theo TÊN; số tên phải > 0
python3 -I scripts/kiem-skip-test.py /tmp/cong.json --cho-phep scripts/cong/skip-cho-phep.txt
go run ./scripts/cong/so-than /tmp/moc-3f1 . scripts/cong/doi-co-chu-y-3f1bf7e.txt
scripts/cong/win-so.sh /tmp/moc-3f1 . /tmp/win                            # 117 / H1 / H3 / H4, số dòng khác
scripts/chay-thu.sh                                                       # smoke đầu–cuối
git grep -h '"github.com/luannt2002/enterprise-ai-control-layer' 3f1bf7e -- '*.go' | wc -l   # 578
```

## Phụ lục — đề xuất sửa `CLAUDE.md` (chủ dự án sửa; agent không sửa)

Ba chỗ do ADR-0013 mục "Hệ quả" và ADR-0015 nêu. Bản diff gốc của lát L3 không nằm trong repo; câu chữ dưới đây soạn lại từ ADR-0013 /
ADR-0015 và cây hiện tại (`tenants/sports` chỉ có `evals/cases.yaml`; `cmd/eais`, `execution/commands`, `execution/reports`, `examples/` đã có).

```diff
@@ §9 — dòng tên và câu sứ mệnh @@
-EAIS = Enterprise AI Systems.
+EAIS = Enterprise AI Integration Reference Architecture (ADR-0013). Viết tắt và mọi định danh chạy giữ nguyên.
 
 EAIS là:
 
-> Multi-tenant Enterprise AI Systems Platform biến dữ liệu, tri thức,
-> workflow và tools theo domain thành AI capabilities có governance,
-> testability và observability mà không coupling shared core vào một ngành.
+> Kiến trúc tham chiếu chạy được — lõi kiểm soát, module tuỳ chọn (QUERY, RAG), ví dụ / case study — biến dữ liệu,
+> tri thức, workflow và tools theo domain thành AI capabilities có governance, testability và observability,
+> mà không coupling lõi vào một ngành.
 
-Nguồn: câu gốc ngày 20/10 (`d70ebc8`) cộng hai chế độ sở hữu của `docs/ROADMAP-DAY-DU.md` §1 (D-RM-01, D-RM-02).
+Nguồn: ADR-0013 (24/10, thay phần định nghĩa của D-RM-01); hai chế độ sở hữu của `docs/ROADMAP-DAY-DU.md` §1 giữ nguyên.
@@ §9 — danh sách tenant @@
-Reference tenants hiện tại:
-
-- Construction
-- Sports Data
-
-Hai tenant tồn tại để kiểm chứng abstraction của platform.
+Tenant / ví dụ hiện có:
+
+- Construction — case study Fuel (`tenants/construction/`, có pack);
+- `examples/01-knowledge-base` (pack chỉ tài liệu) và `examples/02-sme-minimal` (pack thứ hai, bậc B);
+- Sports — DOCS_ONLY: chỉ tài liệu và `tenants/sports/evals/cases.yaml`, chưa có pack.
+
+Pack thứ hai tồn tại để kiểm chứng abstraction của lõi (ADR-0015).
@@ §25 — bản đồ repo @@
-    cmd/eais/          đích — mặt phẳng P7 Vận hành
+    cmd/eais/          mặt phẳng P7 Vận hành
@@
-    execution/         đích — mặt phẳng P4 Hành động
+    execution/         mặt phẳng P4 Hành động (commands, reports)
@@
+    examples/          ví dụ / pack mẫu (ADR-0013)
```

## Phụ lục — bảng đổi SHA (viết lại danh tính tác giả trước khi đẩy)

Trước khi đẩy, 142 commit của đợt được viết lại tác giả / người commit về danh tính của repo (máy đang đặt danh tính toàn cục của một
tài khoản khác). Nội dung KHÔNG đổi: cây của mọi cặp giống hệt, thông điệp giống hệt (kiểm từng cặp). SHA trong tài liệu đã thay sang
SHA mới; SHA nhắc trong THÔNG ĐIỆP commit / merge vẫn là SHA cũ — tra cột trái của bảng dưới.

| SHA cũ | SHA mới | Chủ đề |
|---|---|---|
| `5861291` | `31ca271` | fix(S4CS-02): từ phủ định / dấu bỏ qua / từ nối điều kiện chỉ miễn cụm báo nghĩa theo vị trí |
| `c2fc428` | `056c7b8` | fix(S4CS-02): mệnh đề nêu sẵn giá trị không thành câu hỏi nhờ cụm hỏi lượng |
| `d51b4f2` | `8045dae` | merge: lát L2 — sửa hẹp S4CS-02 (F-001), sẵn sàng chạy H6 |
| `6f91613` | `6a2de61` | docs(license): giấy phép MIT (D-OP-3, chủ dự án chọn 24/10) — F-016 |
| `f9306e6` | `575ea26` | docs(adr): ADR-0013 — định nghĩa sản phẩm vNext "EAIS — Enterprise AI Integration Reference Architecture" |
| `f9d483e` | `db0b641` | docs(adr): ADR-0015 — ranh giới module và luật "≥ 2 miền mới vào lõi" |
| `6774246` | `f8b2e73` | docs(adr): ADR-0016 — chủ sở hữu ACL tài liệu: lõi cấp bộ lọc, RAG chỉ áp |
| `43bd30f` | `e0ed62c` | docs: tên hiển thị "EAIS — Enterprise AI Integration Reference Architecture" ở các dòng tên hiện hành (ADR-… |
| `b4d3311` | `bfca199` | docs(requirements): REQ-MND-016 ghi SUPERSEDED-BY ADR-0013, không sửa câu cũ (D6) |
| `91f34fc` | `928c27d` | docs(architecture): nhãn STALE ở đầu 01-architecture — mốc bff56c4, trỏ go list + ADR-0015 (F-033, D1) |
| `026a518` | `06e9bc5` | docs(adr): ghi chú trạng thái 24/10 ở cuối ADR-0001 / 0010 / 0012, không sửa lời cũ (F-038) |
| `63480c0` | `a875ea5` | docs(readme): mọi số RAG gắn nhãn đường đo — app chỉ từ khoá + model trích, số lai / model thật / B1 là của… |
| `5abac39` | `39e2bcc` | docs: câu CI thành "định nghĩa, chưa chạy trên GitHub" (D4, F-014) |
| `b8249a8` | `f9630e8` | docs(readme): "mọi dữ liệu là giả lập" → "giả lập hoặc văn bản công khai; không có dữ liệu khách" (D13) |
| `c093e74` | `ea42e05` | docs(rag): ghi chú đầu ROADMAP-RAG — thứ tự "không mở mã RAG trước MVP" đã bị D-RAG-01/04/05 thay (D8) |
| `d618715` | `bab64f8` | docs(roadmap): nhãn mốc cho trạng thái cũ của ROADMAP-DAY-DU, D-RM-04 đã trả lời, router theo tên symbol (D… |
| `251890e` | `de393df` | docs(tenants): Sports gắn nhãn DOCS_ONLY ở nơi tài liệu gọi là tenant tham chiếu (D5) |
| `0edc392` | `4793a75` | docs(adr): ADR-0006 ghi H3 / H4 / P1 đã công bố dù cờ content_published trong manifest còn false (D14) |
| `914343b` | `1d042c0` | docs(cho): quyết định 24/10 — GO báo cáo audit vNext (B′), bốn câu trả lời nguyên văn; D-AI-05 sửa hẹp ở L2… |
| `836ff5b` | `de25910` | docs(bao-cao): chép báo cáo audit vNext 24/10 vào repo — đường dẫn máy, tên tệp nội bộ, tên hãng thay bằng … |
| `16b6771` | `e5d5e36` | docs(changelog): mục 24/10 — vNext lát L3 (ADR-0013 / 0015 / 0016, MIT, sự thật tài liệu) |
| `5470082` | `0428293` | merge: lát L3 — sự thật tài liệu, đổi tên hiển thị (ADR-0013), MIT, ghi quyết định 24/10 |
| `1b4c1da` | `fe2ed6d` | fix(reports): khoá tư vấn theo bản ghi thay khoá hàng FOR UPDATE — hết chết khoá 40P01 (R-01) |
| `5007732` | `fc6e73f` | docs(reports): doc.go nói đúng thứ tự khoá — khoá bản ghi cùng thứ tự khoá ô, không khoá hàng (R-01) |
| `52d08dd` | `0ba1da0` | merge: R-01 — hết chết khoá 40P01 ở kho báo cáo Phase 0a (F-002) |
| `48b63f3` | `def8b9b` | docs: nhãn lịch sử nói đúng chỗ đã đổi — dòng sứ mệnh của 01-architecture, ô router.go:71 trả nguyên văn (s… |
| `e044b20` | `2790fc4` | docs(adr): ADR-0013 ghi đúng số dòng import của đường dẫn module, kèm lệnh (soát L3 F6) |
| `9c4b755` | `2cc2921` | docs: tách lời chủ dự án khỏi suy diễn — "KHÔNG gộp main" là mô tả lựa chọn; ADR-0013/0015/0016 chấp nhận t… |
| `2388a55` | `a75e8d2` | test(danh-tinh): comment đầu cổng ghi tên hiển thị theo ADR-0013 (soát L3, mục thêm) |
| `912a70a` | `3308b66` | merge: L3 vòng 2 — sửa theo soát độc lập (F5, F6, F7) |
| `8474186` | `7c47ed3` | test(kien-truc): cổng tenant quét MỌI thư mục trừ tenants/examples/case-studies (F-011) |
| `0187981` | `0e1e5e1` | test(kien-truc): cổng TỪ MIỀN cho literal mã dùng chung + danh sách nợ khớp chính xác (F-011) |
| `aaaaba6` | `feaaba2` | feat(cong): so-than so THÂN test giữa hai cây — đổi kỳ vọng giữ tên không còn "0 đổi" (F-034) |
| `70505f0` | `f30ea6e` | ci: danh sách SKIP cho phép có lý do, truyền vào bước cổng SKIP (F-014) |
| `fbdbe96` | `ab52fe7` | scripts(cong): cong.sh chạy cổng SKIP của CI ở cuối, so thân khi có CONG_MOC; thoát 1 khi đỏ (F-014, F-034) |
| `3347f0a` | `92be3c7` | test(dot-bien): đặt lại 5 neo gãy của w5-g4-acl / w5-rag-hybrid theo mã hiện tại (F-025) |
| `20d46f3` | `34f82d4` | docs(adr): ADR-0014 cổng trước, tách sau (F-011, F-034, F-014, F-025) |
| `23155e7` | `a134412` | merge: lát L1 — cổng trước, tách sau (F-011, F-034, F-014, F-025, ADR-0014) |
| `0737f15` | `af1f24e` | chore(cong): khai báo test đổi thân có chủ ý so với 3f1bf7e cho đợt vNext (F-034) |
| `7d9cd52` | `ba07e30` | refactor(policy): ranh giới quyền ra core/policy, router giữ bí danh (F-031) |
| `d2f9dd8` | `9428b06` | refactor(commands): lệnh ghi, pack và xác thực import thẳng core/policy; cổng ranh giới import (F-031) |
| `24087cd` | `6e8f14b` | test(construction): chéo phạm vi RAG dự phòng ↔ Policy.InScope và người hai vai (F-031, Q-P1-10) |
| `42a0d1b` | `10fbfb5` | docs(rag): comment InScope trỏ policy.Policy.InScope (F-031) |
| `98e313b` | `6daed19` | fix(commands): vết C0 không ghi nguyên văn tên trường lạ của bên gọi (F-035) |
| `e269e63` | `d00b1a3` | refactor(models): Messages chọn theo GIAO THỨC "messages", endpoint và header phiên bản từ cấu hình (F-015) |
| `d2e7774` | `288a0da` | refactor(g7run): model, endpoint chính thức và bảng giá của hợp đồng G7 lấy từ .env (F-015) |
| `e7ebf4a` | `00c176c` | test(cong): cổng tên hãng trong mã — so từng từ với danh sách sha256 (F-015) |
| `8b80433` | `e628afe` | chore(eais): chuỗi usage mang tên hiển thị mới theo ADR-0013 |
| `95a7ff6` | `20434d2` | merge: lát L4 — Policy ra core/policy, bỏ tên hãng khỏi mã, vết C0 không ghi tên trường lạ (F-031, F-015, F… |
| `6d2f2a6` | `a34549f` | fix(cong): tệp khai đổi có chủ ý không nhắc tên cũ ngoài danh sách cho phép (cổng danh tính bắt) |
| `ec4436f` | `e989f56` | chore(cong): khai 30 khai báo test đổi thân của lát L4 (bỏ tên hãng, thay 1:1) — F-034, F-015 |
| `b04b34f` | `b39279f` | fix(pgstore): Ready đóng mặc định khi cách ly tenant bị gỡ — F-010 |
| `2808dbd` | `baebd73` | fix(pgstore): migration nhận diện bằng checksum nội dung — F-036 |
| `468ae7d` | `74be061` | feat(db): 0010 chặn tenant rỗng ở lược đồ cho bảng của 0001 / 0003 / 0008 — F-037 (phần migration) |
| `7bf97ff` | `33eb67f` | docs(adr): ADR-0017 danh tính migration bằng checksum, cấm đổi số / sửa nội dung sau khi đẩy — F-036 |
| `e76bae9` | `ecf0bdb` | merge: gia cố PG — Ready đóng khi RLS tắt, migration theo checksum, 0010 chặn tenant rỗng (F-010, F-036, F-… |
| `d515e4a` | `5dc78ae` | feat(pgstore): LoadDocuments — nạp chỉ tài liệu, giữ danh mục và fact (F-019) |
| `5d8d0ef` | `2936106` | fix(pgstore): lệnh xoá của Load lọc tenant_id, không chỉ dựa vào RLS (F-037) |
| `76e16ca` | `63ca0c1` | feat(eais): báo cáo nạp đếm bản bị cổng chất lượng giữ lại, kèm lý do (F-018) |
| `4ed5fa1` | `50c1a0e` | fix(rag, pgstore): nạp lại không hạ âm thầm bản đang công bố (F-018) |
| `bd994eb` | `6f5ba09` | fix(rag): chữ của bản chưa duyệt không tới embedder và reranker (F-017) |
| `f87370d` | `9b221aa` | fix(rag): tài liệu ra 0 chữ bị cổng chất lượng giữ lại, không nạp im lặng (F-023) |
| `2a50027` | `e79622e` | fix(rag): bảng mất dòng phân cách ở bản đầu tiên bị giữ lại (F-024) |
| `0a960b8` | `345bb90` | feat(pgstore): PruneVectorSpaces — dọn vector của không gian nhúng cũ (F-022) |
| `8698195` | `0767e78` | feat(examples): 01-knowledge-base — kho tri thức trung tính, pack chỉ có tài liệu (F-028) |
| `9464c70` | `f588328` | test(dot-bien): danh mục đột biến L6 — ranh giới RAG (23 dòng) |
| `eb8a6ea` | `f952f6e` | merge: lát L6 — ranh giới RAG (F-019, F-018, F-017, F-022, F-023, F-024, F-037 phần mã, F-028) |
| `ab68787` | `105b6eb` | fix(S4CS-02): vế điều kiện đóng ở từ đóng vế của pack ("thì") — cụm sau "thì" là điều đang hỏi |
| `53d18a8` | `72749e1` | fix(S4CS-02): từ phủ định / dấu bỏ qua đứng trước cụm chỉ phủ nhận cụm khi LIỀN trước |
| `2f5a65d` | `e5e945b` | fix(S4CS-02, F2): lệnh nêu sẵn giá trị chỉ khi cụm ghi / đánh giá đứng đầu mệnh đề — chủ ngữ trước cụm là l… |
| `7bb781d` | `f7263f1` | docs(ADR-0010): ghi chú trạng thái 24/10 — S4CS-02 sau L2 vòng 2, nợ có mã S4CS-02-N1…N4 |
| `baf59ec` | `a692525` | merge: L2 vòng 2 — S4CS-02 theo đúng luật cấu trúc: cắt ở "thì", phủ định LIỀN trước, lệnh đứng đầu mệnh đề |
| `b76413f` | `55a79ee` | fix(pipeline): RAG chỉ khớp ACL / phạm vi tài liệu bằng vai có capability đọc (Q-P1-10 / Q11) |
| `188666d` | `e561c17` | fix(pipeline): dự phòng S4 dò chỉ mục bằng vai có capability đọc (Q11) |
| `81bb00c` | `0bfb304` | fix(rag): bỏ nhánh dự phòng phạm vi — không có luật phạm vi thì tài liệu có phạm vi đóng (F-031, Q11) |
| `883ca12` | `b7710bc` | fix(commands): C4 — capability của lệnh, metrics.read và phạm vi thực thể do CÙNG một vai thoả (Q-P1-10 / Q11) |
| `149bac7` | `5926892` | test(cong): khai bốn test đổi thân có chủ ý của lát Q11 so với 3f1bf7e |
| `3e44efa` | `77b2323` | docs(cho): Q-P1-10 / Q11 = A — quyền và phạm vi / ACL từ cùng một vai (suy diễn của điều phối từ GO chung 2… |
| `5670f98` | `c3f08b7` | test(dot-bien): danh mục đột biến lát Q11 — quyền và phạm vi / ACL từ cùng một vai |
| `927ebf2` | `3d5c33d` | merge: lát Q11 — quyền và phạm vi / ACL từ CÙNG một vai (Q-P1-10 = A theo GO chung, ghi là suy diễn) |
| `afd7faa` | `793cee3` | fix(S4CS-02): lời nhờ trước cụm ghi / đánh giá không biến lệnh nêu sẵn giá trị thành câu hỏi — hướng an toàn |
| `0df7fa7` | `0e3d37b` | fix(S4CS-02): "là" đóng vế điều kiện; mệnh đề lịch sự không phải vế kết quả của mệnh đề điều kiện |
| `e2c58ec` | `faa8597` | docs(pack): chú thích condition_markers khớp luật vế kết quả (S4CS-02 vòng 2–3) |
| `15cfcd7` | `3bf805b` | docs(ADR-0010): ghi chú 24/10 — S4CS-02 sau L2 vòng 3 (vòng cuối), nợ ghi rõ chiều sai an toàn / LỌT |
| `cfbfb32` | `d0efb3e` | merge: S4CS-02 vòng 3 (vòng cuối) — lời nhờ không biến lệnh thành câu hỏi; "là" đóng vế điều kiện |
| `ffbf804` | `150d671` | feat(ingestion): lược đồ danh mục do pack khai — loại thực thể, quan hệ cha, loại của chữ tự do (F-004) |
| `86bb14b` | `7b77c05` | test(ingestion): ca "thiếu entity_lists" khoá đúng lý do thiếu lược đồ (đột biến L5-5 sống sót) |
| `cc08688` | `13f6d98` | fix(ingestion): bản ghi trùng khoá vào cách ly lý do duplicate, không làm hỏng cả lần nạp (F-009) |
| `1666aa1` | `88ff64d` | fix(ingestion): fact nạp từ tin nhắn giữ người gửi và thời điểm gửi (F-008, một phần) |
| `d5cf116` | `a9e800f` | feat(router): kiểu cha để gỡ bí danh mơ hồ do pack khai, vết dùng tên gọi của pack (F-005) |
| `aa93bcd` | `9eb69b4` | fix(intent): route chưa cài bị từ chối lúc nạp pack, không thành 500 lúc chạy (F-007) |
| `0cd1da5` | `671fd42` | refactor(tenant): tách phần meta / băm / đọc chặt sang core/tenant/meta; mẫu câu số liệu chỉ bắt khi pack c… |
| `80ff9b4` | `8aa9c3d` | feat(examples): pack thứ hai tối thiểu 02-sme-minimal — khách hàng → đơn hàng, hai chỉ số số theo ngày (F-0… |
| `05a953e` | `824550e` | chore(cong): khai ba test đổi thân có chủ ý của lát L5 (F-004, F-005) |
| `ad91b6e` | `c3f33e9` | test(dot-bien): L5-9 viết lại cho biên dịch được; L5-8 không phải đột biến chỉ-PG |
| `44b724c` | `889b4e6` | test(router, dot-bien): kiểu rỗng không phải kiểu cha; nhóm F-005 chạy cả ví dụ 02 (đột biến L5-10, L5-13 s… |
| `dff7ea0` | `07c90b3` | test(dot-bien): L5-9 đột biến ở chỗ trả giờ gửi (bản trước vẫn để biến t thừa, không biên dịch) |
| `2364526` | `0fbb164` | merge: lát L5 — hạ Fuel xuống case study tại chỗ + pack thứ hai examples/02-sme-minimal |
| `9c3faba` | `8d7f643` | fix(S4CS-02, khoá): rút "là" khỏi condition_close_markers — "là" là hệ từ của vế hỏi |
| `be63c34` | `baf28d9` | docs(ADR-0010): ghi chú 24/10 — S4CS-02 KHOÁ sau vòng 3; rút "là" và lý do; nợ N8 (LỌT), N9 (an toàn) |
| `4ecb0ef` | `6fb1e90` | merge: KHOÁ S4CS-02 sau vòng 3 — rút "là" khỏi từ đóng vế điều kiện (hồi quy S1 của vòng 3) |
| `1c4c7f4` | `ac00fc6` | fix(eais): log thân lệnh bị từ chối chỉ ghi LOẠI lỗi, không ghi chữ của bên gọi (F-035, đường log) |
| `4672628` | `4e3326d` | docs(traces): comment "Giới hạn đã biết" nói đúng đường chạm tới của giá trị mở bằng nháy thoát sau F-035 |
| `1ea85b3` | `eb3d64c` | refactor(cong): bỏ tên hãng còn sót ở cmd/eais/unit_test.go và scripts/du-toan-g7.py, rút hai mục khỏi danh… |
| `2398cbe` | `ce5b978` | feat(eais): dòng NẠP của `eais ingest` và log nạp lại qua HTTP nói số bản bị giữ, số bản giữ nguyên công bố… |
| `5a24dd1` | `20a373f` | test(dot-bien): m2b thêm hai đột biến R-01 nhắm TestPG_R01_ — bỏ khoá bản ghi, đặt lại FOR UPDATE (M35, M36) |
| `2781e32` | `5b7f69f` | test(dot-bien): m4 — M2, M4, M16 (sửa migration 0004) chạy trên database dựng lại (mm) sau F-036 |
| `91cc072` | `34b0fa4` | merge: dọn mã — log không ghi chữ của bên gọi, hết tên hãng trong mã, CLI/log nạp in bản bị giữ, đột biến R… |
| `77bd98a` | `5e34d32` | fix(router): phạm vi dữ liệu S5 do CÙNG vai có capability của ý định thoả (Q-P1-10 / Q11b) |
| `c8b9b25` | `e9bae36` | fix(pipeline): ứng viên câu hỏi lại và che phạm vi của vết chỉ xét vai có metrics.read (Q-P1-10 / Q11b) |
| `c6e94da` | `0125f66` | test(dot-bien): danh mục đột biến lát Q11b — đường số liệu xét phạm vi bằng cùng vai có quyền đọc |
| `1ce56b6` | `0470094` | merge: lát Q11b — đường số liệu S5, ứng viên hỏi lại và che vết dùng CÙNG một vai (Q-P1-10 / Q11 = A) |
| `e195b17` | `11872ad` | refactor(commands): CapMetricsRead trỏ về core/policy — một nguồn cho tên quyền đọc số liệu (Q11b) |
| `6bbce57` | `0cbd109` | refactor(normalization): danh từ nhãn / đếm của ngành ra lexicon pack, đọc qua Engine (F-030) |
| `e4fd311` | `8f2dd45` | fix(intent): lời giải thích khe thiếu gọi kiểu cha bằng tên gọi do pack khai, không viết cứng "dự án" (F-005) |
| `90dc086` | `11eaccb` | test(normalization): pack cầu đường khai đúng danh từ nhãn / đếm cũ và tenant.Load nối chúng tới tầng ý địn… |
| `dff82cf` | `37d9014` | fix(ingestion, pgstore): fact giữ ghi chú suy luận khi nạp — cột facts.note, migration 0011 (F-008, phần cò… |
| `032bfab` | `841a392` | test(construction): khoá hai chỗ đột biến sống sót — số đếm của pack ở bộ dò ngày dd-mm (F-030) và S209 khô… |
| `e1bfb27` | `2a03abf` | test(dot-bien): RV3 đặt lại trên mã hiện tại — hằng NarrowByProject đã gỡ ở L5, dòng cũ không biên dịch |
| `da36c1e` | `16f0ae7` | docs(examples): 01-knowledge-base bỏ bốn mẫu câu số liệu thừa, README nói đúng giới hạn sau F-029 |
| `aa813d1` | `12b1f82` | test(dot-bien): danh mục đột biến lát L5b — F-030, F-005, F-008 (20 dòng, 2 dòng [PG]) |
| `31d38cf` | `0e43aff` | merge: lát L5b — danh từ nhãn / đếm ra lexicon pack (F-030), lời giải thích kiểu cha theo pack (F-005), cột… |
| `f589cf6` | `a35f3d8` | fix(pgstore): Ready nêu cột facts.note khi kho chưa áp 0011 (F-008, lát L5b) |
| `76eed0c` | `76201cf` | docs(ops): cấu hình model theo giao thức sau lát L4 — provider `messages`, endpoint bắt buộc, header phiên … |
| `706279f` | `9f1ee35` | docs(adr): ADR-0017 đính chính — 0010 gặp dòng tenant rỗng thì 0011 không áp, Ready đòi facts.note nên app … |
| `2e945e8` | `05827fb` | docs(adr): ADR-0016 ghi chú trạng thái 24/10 — thi công một phần; mục 3 (vai lạ lúc nạp) CHƯA LÀM, mục 2 ch… |
| `111b405` | `0c99b47` | docs(cho): Q-P1-10 — ghi chú cập nhật: định tuyến S5 / ứng viên / che vết theo cùng một vai đã làm ở lát Q11b |
| `76a7376` | `21488c8` | docs(bao-cao): bản chép audit vNext — đính chính "583 dòng import" thành 578 dòng trong 227 tệp |
| `62de0fa` | `64f411d` | docs(bao-cao): báo cáo tích hợp 24/10 — ghi chú cập nhật: R-01 đã sửa, S4CS-02 đã khoá kèm nợ (nhánh vnext) |
| `3232444` | `26e8129` | docs(bao-cao): báo cáo thi công vNext 24/10 — lát đã gộp, số đo, sổ nợ bền, câu chờ quyết, đề xuất sửa CLAU… |
| `ed14ae5` | `7e059d2` | docs(status): khối vNext đầu tệp — nhánh vnext/tach-2410 chưa gộp main, lát đã gộp, số đo, việc chưa làm; l… |
| `1acc087` | `fb35762` | docs(changelog): mục 24/10 — vNext thi công trên vnext/tach-2410, đủ các lát (ngắn), trỏ báo cáo thi công |
| `cd9b7c1` | `f8d6be3` | docs(readme): giới hạn đã biết — lỗ F-017 đã đóng ở L6, cổng SKIP cục bộ nay xanh (CI chưa chạy trên GitHub… |
| `b37a231` | `4e0054b` | merge: tài liệu theo soát độc lập cuối + báo cáo thi công đợt vNext, sổ nợ bền |
| `e992227` | `1cb5c44` | fix(g7run): mốc hợp đồng G7 ghim bằng sha256 trong mã — .env trỏ máy giả không thành bằng chứng G7 (S2, soá… |
| `804b8ec` | `e7b1054` | fix(pgstore): lệnh xoá vector của EmbedKnowledge lọc theo tenant ngay trong câu lệnh (S4, soát độc lập cuối) |
| `2546cf8` | `0879a2d` | chore(cong): luật SKIP cho phép nêu loại thứ hai "nợ đã khoá có mã ADR"; mục N8 ghi rõ loại đó (S4, soát độ… |
| `36ab3f3` | `ee1dae5` | chore(cong): lý do đổi thân của lát L4 nói đúng phép kiểm bị bỏ / đổi theo hành vi mới (S4, soát độc lập cuối) |
| `6e0c37a` | `75e8197` | fix(g7run): giao thức khác messages bị từ chối TRƯỚC lời gọi model — hết đường gọi model không trần ngân sá… |
| `48eeea2` | `86eee70` | merge: sửa mã theo soát độc lập cuối — mốc hợp đồng G7 ghim sha256, giao thức ngoài hợp đồng bị từ chối trư… |
| `204e0cf` | `2f058d7` | docs(g7): mục 8 khớp mã cuối — so băm ghim (contractModelSHA256, officialEndpointSHA256), chuẩn hoá endpoin… |
| `ff4d7e1` | `d96e703` | docs(bao-cao): báo cáo thi công vNext — số đo CHỐT trên 204e0cf (2.482 test có PG, 0 fail, mất 0; smoke 21/… |
