# PROMPT — Audit lại `docs/ROADMAP.md`, trả lời câu hỏi, đồng bộ hai phía, rồi commit lên lại

> **Gửi:** phiên team agent đang code EACL (server khác).
> **Từ:** phiên học / audit của chủ dự án (Luân), ngày 21/10/2026.
> **Mốc khi gửi:** code tại `8280f40` (chưa đổi code sau đó); tài liệu tới commit chứa tệp này.
> **Cách dùng:** đây là một prompt tự chứa. Đọc hết rồi làm theo thứ tự ở §2. Kết quả **bắt buộc** commit và push lên `origin/main` theo khuôn §5, để chủ dự án pull về đọc lại toàn bộ: câu đã hỏi, câu team trả lời, việc team đã quyết.

---

## 1. Bối cảnh trong 10 dòng

1. `docs/ROADMAP.md` (commit `fe2cb3f`) là roadmap hợp nhất do phiên học viết.
   - Nó audit hiện trạng tại `8280f40`.
   - Nó đo thêm 7 phép thử bằng cách bơm dữ liệu vào bản sao trong scratchpad, không sửa dữ liệu repo (Phụ lục B).
   - Nó đề xuất một bàn quyết định ở §4.7.
2. ROADMAP **không phải nguồn quyết định**. Thứ bậc nguồn: `CHO-QUYET-DINH.md` > ADR > hợp đồng (`docs/04`, `HOP-DONG-G7.md`) > `docs/v3/` > `CLAUDE.md` > ROADMAP.
3. `docs/study/` (commit cùng đợt với tệp này) là tài liệu học do một phiên khác viết sáng 21/10, **chưa ai soát**. Có 5 dòng chứa tên sản phẩm cũ, đã được sửa tối thiểu để qua cổng danh tính (lý do ở §7).
4. Phiên học có thể sai. Nó **đã sai một lần**: khuyến nghị cũ "chạy W-G7 trên H3, ghim mã `a1595f0`" trái ADR-0006 §10 (Phụ lục D của ROADMAP). Vì vậy cần team soát chéo, không nhận lời khai.

## 2. Việc phải làm, theo thứ tự

1. `git pull --ff-only`, ghi HEAD.
2. **Đo lại Phụ lục C của ROADMAP** tại HEAD của bạn. Ghi từng lệch so với cột chú thích.
3. **Đọc trọn ROADMAP.** Người soát phải khác người viết phần code liên quan (`CLAUDE.md` §2.1).
4. **Đối chiếu các khẳng định ở §3:** XÁC NHẬN / BÁC (kèm lệnh) / ĐÃ ĐỔI (kèm commit).
5. **Trả lời các câu hỏi ở §4.**
6. **Xử lý bàn quyết định §4.7 của ROADMAP theo ba loại:**
   - **Loại 1** (kỹ thuật, đảo được, trong phạm vi được uỷ quyền): team **tự quyết và làm**, có soát độc lập. Quyết định có hệ quả thì ghi ADR. Commit.
   - **Loại 2** (chính sách bằng chứng / sản phẩm) và **loại 3** (tiền, khoá, người chấm): team **không tự quyết**. Viết đề xuất có lý do, hệ quả, khuyến nghị; ghi trạng thái `BLOCKED_FOR_OWNER`. Thêm câu mới vào `docs/CHO-QUYET-DINH.md` theo đúng khuôn bảng ở đó, **để trống cột "Trả lời"**.
7. **Đồng bộ ROADMAP:**
   - Sửa trực tiếp chỗ sai sự thật trong `docs/ROADMAP.md`, giữ cấu trúc, và ghi mỗi lần sửa vào Phụ lục F.
   - **Không xoá** phần đề xuất của chủ dự án.
   - Chỗ team không đồng ý thì thêm dòng `Phản hồi team (<ngày>, <commit>): …` ngay bên dưới, không âm thầm đè.
8. **Viết tệp phản hồi** `docs/bao-cao/2026-10-21-phan-hoi-roadmap.md` theo khuôn §5. Đổi ngày trong tên tệp nếu làm vào ngày khác.
9. **Chạy cổng trước khi push:** `go vet ./...`, `gofmt -l .`, `go test -count=1 ./...` (có PG nếu môi trường có). So test **theo tên**, không chỉ đếm tổng. `TestDanhTinhChiConEACL` phải xanh.
10. **Commit nhỏ, tách theo việc. Push thường lên `origin/main`.** Không force push, không viết lại lịch sử.

## 3. Khẳng định cần XÁC NHẬN hoặc BÁC

Tái lập các mục K1–K5: dùng §5.2 và Phụ lục B của ROADMAP. Sinh dữ liệu seed 42 vào thư mục tạm, chép ra bản sao, bơm thêm đúng một dòng vào `messages.jsonl` của bản sao, rồi chạy `winreport` với câu hỏi tương ứng. **Không** sửa dữ liệu hay bộ ca trong repo.

| Mã | Khẳng định (ROADMAP) | Nơi trong ROADMAP | Gợi ý kiểm |
|---|---|---|---|
| K1 | `≈` không bật cờ ước lượng. EX-03 đội báo trả 4.000 L không ghi chú, trong khi manifest của bộ sinh ghi `approx: true` cho dòng PDF đó. W-G1 không so cờ `approx` | §0.4 #1; Phụ lục B #20–21 | `engine.go:198-210`, `lexicon.yaml:26`; `grep -n approx tenants/construction/win/winrun/winrun.go` |
| K2 | Tin "hôm qua báo nhầm, máy 03 là 3k" gửi 20/10 trở thành 3.000 L đội báo **của ngày 20/10**; ngày 19/10 không đổi. Sai im lặng | §0.4 #15; B #26 | `ingest.go:234`, `ingest.go:255-290` |
| K3 | Tin "mai cần 4k máy 03" gửi 19/10 được nạp thành quan sát của ngày gửi; câu hỏi 19/10 chuyển sang fallback | §0.4 #16; B #24 | như K2 |
| K4 | Tin gửi lại làm câu hỏi chuyển sang fallback, không cộng gấp đôi im lặng. Seed 42 không có ca trùng | §0.4 #3; B #25 | `query.go:199-220` |
| K5 | "một tỷ hai" đọc thành 1.000.000.002 (ở `8280f40` bị cổng miền giá trị che) | §0.4 #17; B #22 | bảng số bằng chữ trong `data/normalization` |
| K6 | Canary tenant B không thể đỏ vì harness chỉ nạp tenant A. "0 canary" ở H1/H3 không phải bằng chứng cách ly | §0.4 #6 | `winrun.go:116-118`, `g7run/main.go:633` |
| K7 | `pgstore.Open` không kiểm vai BYPASSRLS / superuser; `Load` chạy `DELETE FROM` dựa vào RLS | §0.4 #9 | `pgstore.go:24-34`, `load.go:28` |
| K8 | HYBRID: phần tài liệu bị chặn thì câu trả lời nói như "thiếu tài liệu" (sai lý do), `Verdict.Pass` vẫn `true` | §0.4 #7 | `ask.go:482-487`, `templates.yaml:35` |
| K9 | Usage của model bị bỏ rơi; bản ghi quyết định không lưu bền | §0.4 #8 | `ask.go:68`, `ask.go:518-522` |
| K10 | S9 không đạt → `SYSTEM_FAILURE` + `fallback_unverified` | §2.4; §3.3 | `ask.go:463-466`, `ask.go:533` |
| K11 | H3 đã chạy và công bố nên là hồi quy (ADR-0006 §7, §10). Lượt chính W-G7 cần tập mới (H4), hoặc một ngoại lệ có ghi | §4.7 #1 | ADR-0006; `HOP-DONG-G7.md` §12; báo cáo ca `f85ee7f` |
| K12 | 2 ca permission đỏ trên H3 (WIN-I-625, 632) không phải rò rỉ: một ca hiểu sai thành ngoài phạm vi, một ca chặn oan | §0.2 | `ket-qua-h3/winreport-bo-nho.txt` |
| K13 | Tài liệu lệch: `CHO:28` thiếu D-G7-18, D-H1-1/2/3, D-OP-9; `docs/v3/06` ghi R0-02 và R0-06 BLOCKED dù đã trả lời; `docs/v3/01` đo ở `85108c4`; `README` / `STATUS` gọi W-G là "G1–G7"; `05` §2.4 còn "temperature 0" | §6.3 | đọc từng tệp |
| K14 | Tên và chú thích của `TestCauXacNhanMotVeVanPhanQuyet` nói "vẫn được phán quyết", nhưng kỳ vọng mới là không phán quyết | §6.3 | `tenants/construction/pipeline_xac_nhan_an_toan_test.go` |
| K15 | `CLAUDE.md:647-650` giao Context Engine "permission filtering; authority", ngược ranh giới sở hữu của V3 (app sở hữu) | §1.4 | `CLAUDE.md` §13 |

## 4. Câu hỏi team phải trả lời

Mỗi câu trả lời có nhãn bằng chứng [ĐO] / [CODE] / [REPO] / [SUY]. Không biết thì ghi "chưa đo" và cách đo.

| Mã | Câu hỏi |
|---|---|
| Q1 | **H4:** kế hoạch soạn (tác giả độc lập, ≥ 13 ca mỗi lớp, cam kết băm), ai soạn, mốc thời gian? Phần nào cần chủ dự án quyết? |
| Q2 | **Canary:** thiết kế sửa harness (nạp hai tenant + đối chứng dương chứng minh canary có thể đỏ). Làm trước W-G7 được không? |
| Q3 | **BYPASSRLS:** thêm kiểm vai ở `pgstore.Open`, có ca âm? |
| Q4 | **Điều tra S4:** 20 ca đỏ ở S4 của H3 và 15 ca đỏ của H1, gom theo cụm C3–C8. Mỗi cụm thiếu thuộc tính gì, bất biến là gì, ca âm nào? Kế hoạch so rule / model / hybrid trước khi mở cổng `CLAUDE.md` §19? |
| Q5 | **W-G1 có nên so cờ `approx` không?** Đây là lỗ của **phép đo**, nên theo luật "sửa phép đo trước benchmark" (§4.7 của ROADMAP) có thể làm trước W-G7. Team đồng ý không? |
| Q6 | **Tin đính chính và tin kế hoạch:** hướng xử lý ở tầng nạp (trạng thái sự kiện: đã xảy ra / dự kiến / đính chính). Chỉ đề xuất; không đổi hành vi trước W-G7 nếu chủ dự án chưa cho |
| Q7 | **Cổng phán quyết:** trên H3 mất 12 phán quyết đúng mà chưa chặn được lỗi nào. Team đề xuất các mức độ phủ nào để chủ dự án chọn, mỗi mức kèm số đo? |
| Q8 | **T2.2:** 3 dòng vá đã có người soát thứ ba chưa? Nếu chưa, ai soát, khi nào? |
| Q9 | **G6':** thuộc tính nào được chứng minh thêm so với G6 gốc? Ghi thành một câu nhị phân |
| Q10 | **Bộ chấm v2:** lịch làm; yêu cầu người chấm mù (mù danh tính hệ, rubric đóng băng): cần bao nhiêu người, bao nhiêu câu, mất bao lâu? |
| Q11 | **D-V3-09 và R2:** team có đồng ý định nghĩa "hết đường găng = R1 có kết luận đáng tin (WIN / PARTIAL / FAIL)" không? Nếu được làm phần giấy: kế hoạch cho ADR-0003, hợp đồng API, R3-00 / R4-00? |
| Q12 | **R0:** cập nhật `docs/v3/05` §5 để gỡ D-V3-01 khỏi danh sách chặn (chủ dự án đã giao việc chọn model cho R3 / R6)? |
| Q13 | **Bắt buộc:** ROADMAP sai, thiếu, hoặc đánh giá quá tay ở đâu? Nếu "không có" thì ghi các lệnh đã chạy để kết luận như vậy |

## 5. Khuôn tệp phản hồi (bắt buộc)

Tệp: `docs/bao-cao/2026-10-21-phan-hoi-roadmap.md`. Nhãn bằng chứng giống ROADMAP.

1. **Mốc:** HEAD lúc bắt đầu, HEAD lúc push; môi trường (có PG không, Go version).
2. **Đo lại Phụ lục C:** bảng `lệnh | kỳ vọng trong ROADMAP | kết quả | lệch?`.
3. **K1–K15:** bảng `mã | XÁC NHẬN / BÁC / ĐÃ ĐỔI | bằng chứng (lệnh hoặc tệp:dòng) | ghi chú`.
4. **Q1–Q13:** trả lời từng câu.
5. **Bàn quyết định §4.7 (16 việc):** bảng `# | loại | team quyết gì (chỉ loại 1) hoặc BLOCKED_FOR_OWNER + đề xuất | commit`.
6. **Việc đã làm trong lượt này:** commit hash; test trước / sau theo tên; ai soát độc lập.
7. **Việc chưa làm và vì sao;** rủi ro còn lại.
8. **Câu cần chủ dự án trả lời:** ngắn, mỗi câu có khuyến nghị và hệ quả nếu chưa quyết. Các câu này cũng phải nằm trong `CHO-QUYET-DINH.md`.

## 6. Luật và giới hạn

- Theo `CLAUDE.md`: người chấm ≠ người làm; đòi bằng chứng, không nhận lời khai; sửa phẫu thuật; không sửa test để code sai thành xanh; phân loại test đỏ (môi trường / có sẵn / hồi quy mới).
- **Không** gọi model trả tiền; không tìm, không in khoá.
- **Không** đọc bộ giữ lại chưa chạy (H4 khi có).
- **Không** ghi vào cột "Trả lời" của `CHO-QUYET-DINH.md`: cột đó chỉ chủ dự án ghi.
- **Không** đổi hành vi của hệ đang bị đo trước khi chủ dự án chọn tập chạy W-G7 (§4.7 #1). Ngoại lệ: sửa **phép đo** (canary harness, cờ `approx` ở W-G1 nếu đồng ý Q5) và **an ninh** (BYPASSRLS), có soát độc lập.
- Cổng danh tính: tệp mới không chứa tên sản phẩm cũ.
- Repo public: không đường dẫn máy, không bí mật, không dữ liệu thật.
- Commit nhỏ, tách theo việc; push thường lên `origin/main`; không force.

## 7. Lý do của yêu cầu này

- **Chủ dự án đang học để sở hữu EACL, không chỉ để giao việc.** Phiên học đã audit độc lập và có phát hiện mới mà team chưa ghi:
  - 3 lỗi im lặng ở tầng nạp dữ liệu (K2, K3, K5);
  - W-G1 không kiểm cờ ước lượng (K1);
  - H3 đã dùng nên W-G7 thiếu tập sạch (K11).

  Những phát hiện này cần team soát chéo, vì chính phiên học đã sai một lần (K11 sửa khuyến nghị cũ của nó).
- **Một tài liệu đồng bộ cho cả hai phía.** Chủ dự án chỉ cần đọc `docs/ROADMAP.md` và tệp phản hồi để biết: mình đã hỏi gì, team trả lời gì, team đã tự quyết gì, còn gì chờ mình.
- **Ranh giới quyết định rõ ràng.** Loại 1, team tự quyết và chịu trách nhiệm bằng bằng chứng. Loại 2 và 3 về chủ dự án qua `CHO-QUYET-DINH.md`. Đây chính là luận đề của EACL áp vào cách xây EACL: agent đề xuất → bằng chứng hỗ trợ → chủ dự án quyết → phần mềm thi hành → kết quả được xác minh.
- **Vì sao `docs/study/` có 5 dòng đã sửa:**
  - Commit nguyên trạng sẽ làm đỏ `TestDanhTinhChiConEACL` trên main.
  - Năm dòng chứa tên sản phẩm cũ được sửa tối thiểu: giữ nghĩa, và cập nhật các sự thật đã đổi ngày 21/10 (module, biến môi trường, `CLAUDE.md` §9).
  - Các đường dẫn tương đối của hệ cũ (`internal/...`) giữ nguyên, vì cùng loại đường dẫn đã công khai ở `docs/02-migration-audit.md` theo D-V3-07.
  - Phần còn lại của `docs/study/` chưa ai soát; team soát khi có thời gian, không bắt buộc ở lượt này.
