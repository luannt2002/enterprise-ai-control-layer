# Báo cáo thi công vNext 24/10/2026 tối — đã làm gì theo lát, số đo, sổ nợ bền, câu chờ quyết

> **Viết cho:** chủ dự án và phiên BUILD.
>
> **Quyết định:** GO 24/10 của chủ dự án cho báo cáo audit vNext (`docs/CHO-QUYET-DINH.md` mục 0, dòng "vNext"): phương án B′ — cổng
> trước, tách TẠI CHỖ, dời thư mục để sau; làm L5 trước H6 (tệp ca H6 không có trên máy); giấy phép MIT; L7 (xác thực thật + CRM giả) để
> đợt sau. Báo cáo audit (nguồn sự thật của đợt): `docs/bao-cao/2026-10-24-vnext-audit/BAO-CAO.md`, phiếu phát hiện §10 (`VNEXT-F-0xx`).
>
> **Nhánh:** `vnext/tach-2410`, đi từ main `3f1bf7e`, đỉnh `f589cf6`. **CHƯA gộp main.** Sau đỉnh còn hai lát sửa theo soát độc lập cuối,
> chạy song song: tài liệu (`vnext/sua-soat-docs`, có tệp này) và mã (`vnext/sua-soat-ma`).
>
> **Nhãn:** [ĐO] = lệnh đã chạy, ghi ai chạy · [REPO] = thông điệp commit / tài liệu repo ghi · [ĐỌC MÃ] = đọc mã, chưa chạy.

## 1. Đã làm theo lát

Lấy danh sách: `git log --oneline --first-parent 3f1bf7e..f589cf6`. Cột "Soát" chép từ thông điệp merge [REPO]; dòng nào thông điệp
không ghi kết luận soát thì ghi đúng như vậy.

| Merge | Lát | Phát hiện đã đóng | Bằng chứng chính | Soát |
|---|---|---|---|---|
| `d51b4f2` | L2 — sửa hẹp S4CS-02 | F-001 (vòng 1) | 7/7 câu S4CS-02 không còn RESOLVED; 5/5 cặp tối thiểu giữ nguyên; winreport bộ nhớ 117 / H1 / H3 / H4 giống hệt `3f1bf7e` | 21/21 đột biến bị bắt, gồm X3, X4 của người soát |
| `5470082` | L3 — sự thật tài liệu | F-016 (MIT), F-020, F-033, F-038; D4–D9, D13, D14; ADR-0013 / 0015 / 0016 | 18 commit chỉ tài liệu + `LICENSE` + danh sách cho phép của cổng danh tính; không đổi hành vi mã | — |
| `52d08dd` | R-01 — chết khoá kho báo cáo 0a | F-002 | khoá tư vấn theo bản ghi, cùng thứ tự tăng dần với khoá ô, thay `FOR UPDATE`; test tải 14.700 thao tác: mã cũ 133 lần 40P01 (điều phối đo lại), mã mới 0 | hai đột biến (bỏ khoá bản ghi / đặt lại `FOR UPDATE`) làm test đỏ |
| `912a70a` | L3 vòng 2 | F5, F6, F7 của người soát | nhãn lịch sử nói đúng chỗ đã đổi; số dòng import đo lại kèm lệnh (227 tệp, 578 dòng); tách lời chủ dự án khỏi suy diễn trong CHO và ADR-0013 / 0015 / 0016 | sửa theo soát độc lập |
| `23155e7` | L1 — cổng trước | F-011, F-014, F-025, F-034; ADR-0014 | cổng kiến trúc quét MỌI thư mục + cổng từ miền với danh sách nợ khớp chính xác; `so-than` so THÂN test giữa hai cây; danh sách SKIP cho phép có lý do; đặt lại 5 neo đột biến gãy | điều phối cố tình phá: thêm `modules/x` → đỏ; thiếu mục SKIP → rc=1 |
| `0737f15`, `6d2f2a6`, `ec4436f` | khai đổi thân có chủ ý | F-034 | `scripts/cong/doi-co-chu-y-3f1bf7e.txt`: mỗi test đổi thân có lý do; 30 dòng của L4 (bỏ tên hãng, thay 1:1) | cổng danh tính bắt một lần nhắc tên cũ trong tệp khai → sửa (`6d2f2a6`) |
| `95a7ff6` | L4 — `core/policy`, bỏ tên hãng | F-031, F-015, F-035 | `core/policy` chỉ phụ thuộc `core/contracts`; `execution/commands` không còn kéo `intelligence/` (`TestRanhGioiImport`); provider theo GIAO THỨC `messages`, endpoint / header từ `.env`; `g7run` đọc model / endpoint / bảng giá từ `.env`; cổng tên hãng so băm từng từ | điều phối phá thử: chèn tên hãng → đỏ. Lỗ hai vai Q-P1-10 đo được, để Skip kèm mã câu hỏi (sửa ở Q11) |
| `e76bae9` | gia cố PG | F-010, F-036, F-037 (migration); ADR-0017 | `Ready` kiểm RLS bật + FORCE + chính sách tenant trên 11 bảng; `schema_migrations` có checksum, migrate dừng khi tệp đã áp bị sửa / đổi tên; 0010 thêm CHECK `tenant_id <> ''` cho 8 bảng | điều phối chạy lại `TestReadyDongKhiRLSTat` có PG: PASS (không SKIP) |
| `eb8a6ea` | L6 — ranh giới RAG | F-017, F-018, F-019, F-022, F-023, F-024, F-028, F-037 (phần mã) | nạp chỉ tài liệu; `DELETE` của Load lọc `tenant_id`; nạp lại không hạ bản đang công bố, báo cáo nạp đếm bản bị giữ; chữ bản chưa duyệt không tới embedder / reranker; `PruneVectorSpaces`; `examples/01-knowledge-base` trung tính | S7 bộ phát triển + `TestCaTriThuc` giống từng câu. F-021 để lại (cần quyết, mục 4) |
| `baf59ec` | L2 vòng 2 | F-001 | cắt ở "thì", phủ định LIỀN trước, lệnh đứng đầu mệnh đề | 9 câu soát vòng 1 nêu nay hỏi lại CÓ NHÃN; 31/31 đột biến bị bắt |
| `927ebf2` | Q11 — cùng một vai | Q-P1-10 = A (SUY DIỄN của điều phối) | RAG và dự phòng S4 chỉ khớp ACL / phạm vi bằng vai có capability đọc; bỏ nhánh dự phòng phạm vi của RAG; C4 của lệnh đòi capability + `metrics.read` + phạm vi trên cùng một vai | test hai vai của L4 bỏ Skip, điều phối chạy lại: PASS; S7 41/41 câu giống |
| `cfbfb32` | L2 vòng 3 | F-001 | lời nhờ không biến lệnh thành câu hỏi | 6 câu soát vòng 2 nêu nay hỏi lại có nhãn; 38/38 đột biến bị bắt |
| `2364526` | L5 — Fuel xuống case study | F-004, F-007, F-009; F-005, F-008, F-029 một phần; F-028 (pack thứ hai) | lược đồ danh mục do pack khai; bản ghi trùng vào cách ly; route chưa cài bị từ chối lúc nạp; pack thứ hai `examples/02-sme-minimal` chạy S0–S10, commit đưa nó vào chỉ đụng `examples/` | đo chạy lùi: pack thứ hai cần 7 tệp lõi; nợ chữ miền 29 → 14 |
| `4ecb0ef` | KHOÁ S4CS-02 | F-001 (khoá) | rút "là" khỏi từ đóng vế điều kiện | sửa hồi quy S1 mà soát vòng 3 bắt; nợ LỌT N8 giữ ở test Skip có lý do |
| `91cc072` | dọn mã | F-035 (đường log), F-015 (tên hãng còn sót), F-018 (CLI / log nạp) | log thân lệnh bị từ chối chỉ ghi LOẠI lỗi; dòng NẠP nói số bản bị giữ; đột biến R-01 (M35, M36) và m4 (M2, M4, M16 trên database dựng lại sau F-036) | thông điệp merge không ghi kết luận soát |
| `1ce56b6`, `e195b17` | Q11b — đường số liệu S5 | Q-P1-10 phần S5 | router xét phạm vi bằng vai có capability của ý định; ứng viên hỏi lại, bộ che phạm vi, dòng S3 dùng vai có `metrics.read`; `CapMetricsRead` một nguồn ở `core/policy` | đo của lát (mục 2) |
| `31d38cf`, `f589cf6` | L5b — từ miền ra pack | F-030, F-005 (phần còn lại), F-008 (phần còn lại, migration 0011) | danh từ nhãn / đếm ra lexicon pack; lời giải thích kiểu cha theo pack; cột `facts.note`; `Ready` đòi cột đó (`f589cf6`, test đỏ trước) | 20/20 đột biến bị bắt; L5b phát hiện S209 không tương đương, đã khoá bằng test. Lệch yêu cầu ban đầu điều phối chấp nhận: `Options.Labels` rỗng ⇒ chỉ từ chức năng |

Phát hiện CHƯA thuộc đợt này: F-003 (xác thực chỉ có dev) → L7, đợt sau; F-006 (HYBRID thiếu quyền tài liệu từ chối cả câu) chờ câu Q7;
F-039, F-040 là lịch sử (REPO_CLAIM), không có việc mã. Các mục còn mở khác ở mục 3.

Soát độc lập cuối trên `f589cf6`: các mục S3 / S4 của người soát chia cho hai lát sửa song song (tài liệu: ADR-0016, ADR-0017, tài liệu vận
hành cấu hình model, ghi chú Q-P1-10 — lát này; mã: lát `vnext/sua-soat-ma`).

## 2. Số đo do điều phối chạy

| Phép đo | Kết quả |
|---|---|
| Cổng có PG trên `f589cf6`: `scripts/cong/cong.sh` [ĐO, điều phối] | **2.475** tên test, 2.471 pass, 4 skip, **0 fail**; `kiem-skip` XANH (4 SKIP có lý do: 3 eval cần động cơ ngoài + nợ S4CS-02-N8); `so-than` XANH |
| So tên với mốc có PG `3f1bf7e` (2.398 tên) [ĐO, điều phối; lát này chạy lại `so_ten.py` trên cùng hai tệp: khớp] | mất **8** (subtest của `TestEndpointTonTienVietKhacVanBiKiemTruoc` từng mang tên endpoint của hãng — đổi tên khi bỏ tên hãng, pass → pass), đổi **0**, mới **85** |
| Test không PG trên `31d38cf` [ĐO, điều phối] | 2.093 tên, 1.880 pass, 213 skip, 0 fail |
| Smoke `scripts/chay-thu.sh` trên `f589cf6` [ĐO, điều phối] | 21 đạt, 0 hỏng |
| winreport bộ nhớ `3f1bf7e` ↔ `f589cf6` [ĐO, người soát cuối] | 117 / H1 / H3 / H4: khác **0** dòng |
| R-01 [ĐO, lát R-01 + điều phối đo lại] | 14.700 thao tác: mã cũ 133 lần 40P01, mã mới 0 |
| Q11b, người hai vai [ĐO, lát Q11b] | mã cũ: 328 câu khác + 433 chỗ lộ trên 483 câu; mã mới: 0 (y hệt vai đủ quyền một mình) |
| Pack thứ hai [ĐO, lát L5, chạy lùi] | cần 7 tệp lõi |
| Nợ chữ miền trong lõi (danh sách của cổng từ miền) [REPO, L5 / L5b] | 29 → 14 → **4** mục |
| Cổng SKIP của CI | trước đỏ (rc=1 trên `3f1bf7e`), nay xanh. CI vẫn CHƯA chạy trên GitHub |

## 3. Sổ nợ bền

Chiều sai: **an toàn** = sai về phía từ chối / hỏi lại / lỗi ồn · **LỌT** = trả lời sai hoặc lộ mà không ai báo.

| Mã | Chiều sai | Nơi ghi | Cách gỡ |
|---|---|---|---|
| S4CS-02-N1…N9 (N7 đã đóng lúc khoá) | N1–N5, N9 **an toàn** (hỏi lại thừa); N6, N8 **LỌT** — có từ trước L2 (`3f1bf7e` cũng trả số) | ADR-0010 bảng nợ S4CS-02; N8: test Skip `TestS4CS02NoN8LaDongVeDieuKien` + `scripts/cong/skip-cho-phep.txt` | vòng sửa S4 sau H6; H6 sẽ chấm mã còn các nợ này |
| F-021 công bố tài liệu + vector không nguyên tử | an toàn (thiếu vector ⇒ lượt lỗi, không lộ); chỉ đường lai, hiện chỉ bộ đo chạy | phiếu F-021; merge `eb8a6ea` | chủ dự án quyết (câu 2, mục 4) — mọi cách sửa đảo một hợp đồng đã có test |
| `Ready` không kiểm `document_vectors`, `report_*` | an toàn (thiếu bảng lỗi ở lượt đầu thay vì lúc khởi động); app chưa dùng `report_*` | `platform/pgstore/ready.go`, `readyNeeds` (11 bảng) | thêm vào `readyNeeds` cùng commit nối reports / đường lai vào app |
| Ghi chú suy luận chưa vào `Evidence` | **LỌT** nhãn: số suy luận lúc nạp có `facts.note` (0011) nhưng `contracts.Evidence` không có trường, câu trả lời không nói số đó là suy luận | F-008 phần còn lại; `core/contracts/contracts.go`, kiểu `Evidence` | thêm trường + S9 kiểm nhãn; test đỏ trước bằng một fact có `note` |
| Vết `/v1/ask` còn lọt dạng `password=\"…` | **LỌT** (bí mật trong vết) | soát độc lập cuối; giới hạn regex của bộ che | sửa cần đổi phản hồi client — cần quyết cùng người dùng API |
| `Decide` của reports chưa kiểm capability; gói reports chưa nối `cmd/eais` | hôm nay chưa có đường gọi từ app; nối vào mà chưa sửa là **LỌT** quyền | merge `52d08dd` | đổi API `Decide` nhận principal + policy, rồi mới nối |
| Đường dữ liệu dùng tên quyền cố định `metrics.read` | an toàn (pack đặt tên khác ⇒ mọi lượt số liệu bị chặn) | `core/policy/policy.go:18`, `CapMetricsRead` | pack khai capability của đường số liệu |
| F-012, F-013, F-027 | hiệu năng / vận hành, không sai dữ liệu; **chưa đo** | phiếu F-012 (tắt server khi đang nạp), F-013 (đọc danh mục nhiều lần mỗi lượt), F-027 (dấu nạp quét cả bảng mỗi câu) | đo trước (benchmark), sửa khi số đòi |
| F-026, F-032 | DEFER | phiếu F-026 (ngân sách token ngữ cảnh), F-032 (adapter PG gom RAG) | F-032 đi cùng việc dời thư mục |
| Dời thư mục `modules/` | HOÃN theo B′ | ADR-0014, ADR-0015 | lát riêng, sau khi các cổng đã đỏ được |
| L7 (xác thực thật + CRM giả, F-003) | — | lời chủ dự án 24/10 | đợt sau |
| Tên hãng trong docs lịch sử | không ảnh hưởng chạy; cổng `TestKhongTenHangTrongMa` bỏ qua `docs/` | [ĐO, lát này] trên `f589cf6`: 35 tệp, 267 từ trúng danh sách băm của cổng tên hãng (cùng cách tách từ); sau lát này 33 tệp, 264 từ | chủ dự án quyết (câu 3) |
| Đột biến S209; neo gãy có sẵn | S209 từng sống sót, đã khoá (`tenants/construction/l5b_test.go`). **Mới nghi:** toàn bộ danh mục `scripts/dot-bien/` chưa chạy lại trên `f589cf6`, có thể còn neo gãy | `scripts/dot-bien/n45-y-dinh-s4.sh`; ADR-0014 mục 5 | chạy mọi danh mục trên bản clone riêng, đếm dòng "neo gãy" (rc=2) |
| H6 + bộ 0a | **BLOCKED**: tệp ca không có trên máy | ADR-0010; `docs/CHO-QUYET-DINH.md` | chủ dự án chọn "làm L5 trước H6", nên H6 chấm mã đã tách; chạy MỘT lần |

**Mới nghi (lát này đọc mã, chưa chạy):** `g7run` với `EAIS_MODEL_PROVIDER=chat_completions` không bọc trần ngân sách và không qua kiểm
trước khi tiêu tiền: trong `run()` (`tenants/construction/win/g7run/main.go`), `dry, costly := false, false` chỉ được gán trong nhánh
`*models.Messages`. Kết luận vẫn CHẶN ("không phải client model thật") nên không thành bằng chứng giả, nhưng lời gọi tới endpoint trả tiền
không có trần. Cách kiểm / sửa: test đỏ trước — provider `chat_completions` trỏ máy chủ giả ⇒ `g7run` phải thoát 2 trước khi máy chủ nhận
yêu cầu nào. `docs/g7/CHI-PHI-G7.md` mục 8 đã ghi "đừng chạy `g7run` với `chat_completions`".

## 4. Câu cho chủ dự án

| # | Câu | A | B | Khuyến nghị |
|---|---|---|---|---|
| 1 | Q-P1-10: quyền và phạm vi / ACL lấy từ CÙNG một vai (Q11, Q11b đã làm theo A — SUY DIỄN từ GO chung) | duyệt A | đảo về hợp các vai: hoàn tác `927ebf2`, `1ce56b6` | **A**: đóng mặc định; mã cũ lộ 433 chỗ trên 483 câu với người hai vai |
| 2 | F-021: tài liệu + vector công bố không nguyên tử | nhúng TRƯỚC, ghi vector trong giao dịch của Load (nạp phụ thuộc embedder) | đoạn chưa có vector đi đường từ khoá + dòng vết (đảo kỳ vọng `TestVectorTriThucPG`: "kho chưa nhúng" ra lỗi) | **hoãn** tới khi đường lai vào app (hôm nay chỉ bộ đo chạy nó); lúc đó chọn **A** — giữ đóng mặc định |
| 3 | Tên hãng trong docs lịch sử (35 tệp / 267 từ trên `f589cf6`) | giữ nguyên tài liệu lịch sử; thêm cổng cho docs hiện hành với danh sách cho phép theo tệp, hạn mức đúng bằng số lần (như `legacyAllowed`) | thay bằng chữ trung tính ở mọi tệp | **A**: B sửa bằng chứng lịch sử và tệp đã băm (hợp đồng G7 được băm vào `manifest.json` của mỗi lượt) |
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
