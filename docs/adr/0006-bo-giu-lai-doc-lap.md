# ADR-0006 — Bộ giữ lại độc lập: chứng minh lại G2, G5, G6 và nền cho G7

- **Ngày:** 20/10/2026, đêm.
- **Trạng thái:** **ĐÃ CHỐT**, tự chốt theo ủy quyền chạy đêm 20→21/10 của chủ dự án. Đảo được.
  - Mục đánh dấu **(chờ phê)** cần chủ dự án phê sau.
  - Mọi mục được chốt **trước** khi có kho ca và trước mọi kết quả trên nó.
- **Liên quan:**
  - D-G7-5, D-G7-11, D-OP-5 (`docs/CHO-QUYET-DINH.md`);
  - ADR-0005;
  - `docs/g7/CONTAMINATION-AUDIT.md`;
  - `docs/04-core-mvp.md` §5–6.

## Bối cảnh

- **Ba ca bị nhiễm.** Kỳ vọng của 083, 107, 108 do chính phiên viết mã "trọng tài" sau lần đo đầu (09:03), trước khi băm bộ ca (09:15). Vì vậy G2, G5, G6 chỉ đạt có điều kiện.
- **Prompt soạn 117 ca do phiên viết mã viết.** Nó có mục "luật tính đáp án" và vài đáp án tính sẵn. Mức ảnh hưởng do kiểm toán nhiễm đánh giá.
- **Mã EAIS đã được lặp sửa tới 117/117 trên chính bộ đó.** Bộ 117 ca vì vậy là bằng chứng **hồi quy**, không phải bằng chứng độc lập.
- **Chủ dự án chốt D-G7-11 = c + bằng chứng thay thế:**
  - loại 3 ca **không** tự làm G2 / G5 / G6 đạt vô điều kiện;
  - phải có bằng chứng độc lập, đóng băng trước khi chạy.

## Quyết định

### 1. Bốn tập, không dùng tập này giả làm tập khác

| Tập | Gồm | Dùng để |
|---|---|---|
| DEV | `tenants/construction/evals/cases.yaml`, test đơn vị | phát triển |
| REGRESSION | 117 ca (kể cả 083 / 107 / 108); mọi ca giữ lại đã bị dùng để sửa mã | chặn hồi quy; **không** làm bằng chứng độc lập |
| HOLDOUT | H1, H2 | chứng minh độc lập (G2 / G5 / G6 ở đây; G7 ở ADR-0005) |
| PILOT | 16 ca | chọn k, kiểm tương thích provider; **không** làm bằng chứng |

### 2. Người soạn và nguồn đáp án

- **Người soạn:** agent không viết mã, chưa từng đọc mã.
- **Được đọc:**
  - `docs/04-core-mvp.md`, ADR-0001, ADR-0002;
  - `docs/tenants/construction/01`, `03`, `05`, `07`;
  - `world.yaml`, `CONTRACT.md`;
  - dữ liệu do bộ sinh xuất ra: được **chạy** bộ sinh, không đọc mã nguồn của nó;
  - `evals/win.go`, chỉ để biết định dạng;
  - pack `metrics.yaml` (thẩm quyền, dung sai) và `policy.yaml` (vai → quyền).
- **Cấm:**
  - mọi mã khác;
  - pack `intents`, `lexicon`, `templates`, `sources`;
  - mọi bộ ca có sẵn, `passing.txt`;
  - tài liệu trạng thái / bản soát / kiểm toán;
  - chạy EAIS.
- **Vì sao được đọc `metrics` và `policy`:** chủ dự án viết "không đọc implementation". Hai tệp này là **chính sách nghiệp vụ**, tương đương yêu cầu. Tệp từ vựng mới là thứ làm lệch câu hỏi (bản soát: 31/37 ca do người đã đọc pack soạn).
  - Đây là chỗ lệch có chủ đích khỏi câu chữ "không đọc mã **và pack**" ở bảng khuyến nghị cũ. **(chờ phê)**
- **Tài liệu mâu thuẫn:** ưu tiên theo thứ tự `04` > ADR-0001 > ADR-0002 > pack `metrics` / `policy` > tài liệu tenant 01–07.
  - Đã biết: `03` §2 ghi phiếu kho = ERP cùng hạng 1 (ngưỡng 0,5%); `04` §3 ghi phiếu kho > ERP, dung sai 10 L.
  - Ca có đáp án phụ thuộc chỗ tài liệu không giải được: **bỏ**, không đoán.
- **Số lượng và độ khác nhau:**
  - mục tiêu 28 ca khác nhau thật mỗi lớp;
  - cấm nhân bản / diễn đạt lại cho đủ số;
  - thiếu thì báo đúng số.

### 3. Người soát và vai của phiên viết mã

- **Người soát kho ca:** một agent khác, chỉ đọc, cùng danh sách đọc với người soạn.
  - Tính lại đáp án độc lập.
  - Cờ ca mơ hồ, trùng, hoặc sai.
  - Ca bị cờ trả về người soạn để sửa hoặc bỏ.
- **Phiên viết mã:**
  - không sửa nội dung ca;
  - **không đọc kỳ vọng trước khi chạy**: chỉ kiểm định dạng bằng script in số đếm;
  - không chỉnh mã theo bộ giữ lại (D-OP-5).

### 4. Chia tập, máy móc

Trong từng lớp hỏng, xếp ca theo `sha256("eacl-holdout-2026-10-20|" + case_id)` tăng dần (chuỗi hex), rồi:
- 2 ca đầu → PILOT;
- 13 ca kế → H1;
- còn lại → H2.

Lớp không đủ ca thì PILOT lấy trước, rồi tới H1. Số ca thật của từng tập ghi vào manifest.

### 5. Đóng băng, và bằng chứng "đóng băng trước khi chạy"

- **Tệp đóng băng**, ở `docs/g7/holdout/`, nằm ngoài mẫu mặc định `cases-*.yaml` của `winreport`, `g7run` và ratchet, nên không lẫn vào bộ 117:
  - `pilot.yaml`, `h1.yaml`, `h2.yaml`;
  - `holdout-manifest.json`. Mỗi ca ghi: `case_id`, `set`, `gate`, `requirement`, `expected_behavior`, `source_of_expectation`, `created_at`, `b1_window`, `frozen_hash` (sha256 của chính ca). Thêm sha256 của từng tệp.
- **Commit + push TRƯỚC lần chạy đầu.** Giờ push trên GitHub là dấu thời gian của bên thứ ba. Giờ của mọi lần chạy sau ghi trong bằng chứng thô.
- Vì sao đây là bằng chứng thay cho "đóng băng trước W1": điều kiện "trước W1" không còn làm được (mã đã có). Ý định của nó là "cổng không bị viết theo mã", và điều này giữ nguyên.
- **Ca bị lộ trước khi đóng băng** (thêm 20/10 đêm, sau sự cố lộ 7 ca, trước khi đóng băng và trước mọi lần chạy):
  - Ca mà nội dung (câu hỏi, kỳ vọng, ghi chú) đã hiện ra trước mắt **người viết mã EAIS** thì không còn là bộ giữ lại.
  - Ghi vào `docs/g7/holdout/loai-truoc-dong-bang.txt`, mỗi dòng `<id ca> <lý do>`.
  - Công cụ đóng băng loại các ca đó **trước khi chia** (§4 chạy trên phần còn lại) và ghi chúng vào `exposed.yaml`: chỉ báo cáo, không tính cổng. Manifest ghi `set: exposed` + `excluded_reason`, băm cả tệp danh sách.
  - Chỉ loại vì lộ, không bao giờ vì kết quả. Danh sách đóng băng cùng bộ, nên không sửa được sau khi đã thấy kết quả.
- **Giới hạn đã biết (bản soát lượt 2, B-05):** bằng chứng "đóng băng trước khi chạy" là commit + push công khai, nên nội dung cả H2 (và truth trong `holdout-manifest.json`) đã công khai từ `16a6140`. Bảo vệ H2 chỉ còn là quy trình: người viết mã không đọc H2. Lần sau dùng cam kết bằng băm (chỉ commit sha256 của tệp ca; công bố nội dung khi chạy).
- **Đính chính sáng 21/10 (người phản bác cuối, F-05; phiên đo dấu giờ transcript):** lần lộ gợi ý của 15 ca (H1 8, H2 5, pilot 2) xảy ra lúc 22:50:28, TRƯỚC commit đóng băng (thông điệp 22:51:01, committer 22:51:44), không phải sau như mục dưới và `59ee556` ghi. Theo mục trên, 15 ca đó lẽ ra phải bị loại trước khi chia: đây là **lệch quy trình**. Không tái chia được vì H1 đã chạy; bù bằng độ nhạy đăng ký trước (bỏ 8 ca H1) và bằng xác nhận của người phản bác phương pháp rằng không dòng mã nào sửa theo gợi ý. H2 có 5 ca lộ gợi ý nên không còn sạch.
- **Gợi ý lộ SAU khi đóng băng, trước khi chạy** (thêm 20/10 đêm): tập đã đóng băng không đổi. Ghi mã ca vào `docs/g7/holdout/goi-y-lo-cho-phien-chinh.txt`; bảng H1 báo thêm G2' / G5' khi bỏ các ca đó. Bài học: kết quả trả về cho phiên viết mã phải có schema không chứa chữ tự do, vì một trường "không kiểm được" cũng có thể lộ kỳ vọng.

### 6. Pilot và luật chọn k (đăng ký trước)

- Pilot: 16 ca. Mỗi hệ cổng (B1, B2+, EAIS) chạy **k = 5**, model thật, trong trần 15 USD.
- Một ca là **chập chờn** với một hệ khi phán quyết của nó khác nhau giữa 5 lượt.
- **Hệ cổng nào có ≥ 2 ca chập chờn (≥ 12,5%) thì chạy chính k = 5; không thì k = 3.**
- Pilot còn ghi: tỉ lệ câu rỗng, lỗi provider, tham số bị từ chối.
  - Sửa prompt / tham số sau pilot được phép, **trước** khi đóng băng, áp như nhau cho mọi hệ, có ghi chú.
- Không có khoá thì pilot BỊ CHẶN. Khi đó hợp đồng G7 tối đa ở trạng thái "sẵn sàng đóng băng sau pilot".

### 7. Chạy và không chỉnh theo bộ giữ lại

- **Phần tất định, không tốn tiền, chạy trên H1:**
  - `winreport -cases docs/g7/holdout/h1.yaml` theo cả đường bộ nhớ lẫn PG;
  - phát lại bản ghi quyết định.
- **Thủ tục chạy** (thêm 20/10 đêm, trước lần chạy H1 đầu tiên): `scripts/chay-giu-lai.sh <tệp ca> <thư mục ra>`.
  - Chạy cả hai đường, rồi lập bảng cổng (`scripts/bang-cong-h1.py`). Ghi revision, giờ bắt đầu / kết thúc, băm tệp ca, băm dữ liệu.
  - Từ chối chạy khi: tệp ca không khớp băm bản đóng băng (`holdout-manifest.json`); tệp là `h2.yaml` khi H1 còn là bộ giữ lại; tập đã có lần chạy HOÀN TẤT (tìm theo băm tệp ca); thư mục ra nằm ngoài thư mục bộ giữ lại hoặc đã có; cây git có thay đổi; tệp ca chưa commit; HEAD chưa có trên `origin/main`; đường PG không dùng database riêng `eais_giu_lai`.
  - Lần chạy HỎNG HẠ TẦNG (lỗi cấu hình / nạp, mã thoát 1) không phải lần đo: được chạy lại khi mã (cây git không tính `docs/`) y nguyên.
  - Một ca chỉ ĐẠT khi đạt ở CẢ HAI đường; đầu vào thiếu (thiếu `-v`, thiếu ca, sai đường) thì BLOCKED. Sửa theo bản soát độc lập lượt 2, trước lần chạy H1 đầu tiên; công cụ có test (`scripts/kiemcongcu`) và đột biến (`scripts/dot-bien/n27-cong-cu-giu-lai.sh`).
- **H2 không chạy** khi H1 còn là bộ giữ lại.
- **Ca H1 trượt:**
  - ghi tầng sai đầu tiên + bất biến bị hỏng;
  - **không** sửa mã để cứu ca;
  - nếu quyết định sửa mã thì **toàn bộ** H1 thành REGRESSION; chứng minh sau đó dùng H2.
- **Kỳ vọng không được sửa sau khi chạy.**
  - Ca mà người soát độc lập xác minh là **sai kỳ vọng** (lỗi người soạn) chỉ được **loại khỏi phép tính**, kèm lý do ghi lại.
  - Ca bị loại tính là "không chứng minh được", không tính là đạt.

### 8. Tiêu chí khi chứng minh lại trên H1

| Cổng | Tiêu chí |
|---|---|
| **G2'** | Tiêu chí G2 của `04` §6 trên H1: 100% ca `entity` về đúng thực thể; 100% ca `ambiguity` ra `CLARIFICATION_REQUIRED` kèm đúng tập ứng viên; 0 lần đoán im lặng |
| **G5'** | Hai phần:<br>- **phần ca:** tiêu chí G5 của `04` §6 trên H1: 100% ca `permission` và `unsupported_answer` đạt; 0 câu trả lời rò rỉ canary;<br>- **phần cơ chế** (AST, RLS + GUC, canary B, bảng "sai đúng một chiều", chèn lệnh): giữ bằng test hiện có; kiểm toán phải xác nhận các test đó không dựa vào kỳ vọng bị nhiễm |
| **G6'** **(chờ phê, sửa tiêu chí)** | - H1 ≥ 100 ca, đóng băng trước lần chạy đầu (§5);<br>- 100% do người không viết mã soạn;<br>- mỗi lớp ≥ 5 ca;<br>- phát lại 100%;<br>- mỗi cổng có đột biến bị bắt;<br>- test thuộc tính. |

**Trạng thái từng cổng:**
- **PASS:** đủ mọi tiêu chí.
- **FAIL:** trượt một tiêu chí. Ghi rõ ca nào, tầng sai đầu tiên.
- **PARTIAL:** đạt trên phần ca được chấm, nhưng chưa đủ điều kiện áp dụng. Ví dụ: một lớp < 5 ca, H1 < 100 ca, hoặc có ca bị loại vì sai kỳ vọng.
- **BLOCKED:** không chạy được.

Chỉ PASS độc lập mới phục hồi cổng về "vô điều kiện".

### 9. Báo cáo

Mỗi cổng một bảng: yêu cầu · bằng chứng cũ (bị nhiễm) · bằng chứng mới · kết quả · ca hỏng (tầng sai đầu tiên) · trạng thái.

### 10. H3: bộ giữ lại thứ ba, đóng băng bằng cam kết băm (thêm sáng 21/10, TRƯỚC khi có tệp H3 và trước mọi lần chạy)

- **Vì sao cần.**
  - H1 đã chạy một lần (01:42 ngày 21/10), mã đã sửa sau đó ⇒ theo §7, toàn bộ H1 là REGRESSION.
  - H2 không sạch (§5: công khai + 5 ca lộ gợi ý).
  - Hợp đồng vận hành ca 21/10 §5 đòi bộ mới, đóng băng trước mọi bản sửa ngữ nghĩa.
- **Người soạn, người soát.**
  - Quy trình: 2 người soạn chia lớp (A1: `representation`, `entity`, `ambiguity`, `semantic`; A2: `source_conflict`, `permission`, `unsupported_answer`, `routing`) → 2 người soát (R1 tính lại mọi con số từ dữ liệu; R2 soát hợp đồng, mơ hồ, trùng) → 1 người sửa (F) sửa hoặc bỏ ca bị cờ, rồi ghép `h3.yaml`.
  - Cả 5 là agent kiểu Plan: không nhận CLAUDE.md / memory của máy, vì memory có ghi điểm yếu của mã.
  - Chỉ đọc bộ đồ nghề `docs/sources/holdout-h3/kit/`:
    - `04` (HEAD `016422e`), ADR-0001;
    - ADR-0002 và `03` ở `94bbaae` (bản đầu đêm 20/10, trước các ghi chú hiện thực trong đêm);
    - `01`, `05`, `07`, `world.yaml`, `CONTRACT.md`, pack `metrics.yaml`, `policy.yaml`;
    - dữ liệu bộ sinh (functional, hạt giống 42);
    - `DINH-DANG.md` (định dạng + luật chấm) và lệnh `kiemca`.
    - Băm từng tệp ghi trong manifest.
  - Khác §2: không đọc `evals/win.go` (định dạng tả trong `DINH-DANG.md`).
  - **Giới hạn:** lệnh cấm đọc là luật giao cho agent, không phải rào kỹ thuật. Bù bằng kiểm toán sau: một script quét nhật ký công cụ của 5 agent, chỉ in ĐƯỜNG DẪN nằm ngoài bộ đồ nghề và thư mục ra, không in nội dung.
- **Phiên viết mã:**
  - không mở tệp nào dưới `docs/sources/holdout-h3/` ngoài `kit/`;
  - kết quả agent trả về chỉ là số đếm / enum (schema không có chữ tự do);
  - kiểm định dạng bằng `kiemca` (chỉ in số).
- **Đóng băng = cam kết băm.**
  - Commit + push `docs/g7/holdout-h3/holdout-manifest.json` gồm: sha256 của `h3.yaml`, số ca theo lớp, số ca có kỳ vọng số liệu, revision, giờ, băm bộ đồ nghề và dữ liệu.
  - Nội dung `h3.yaml` ở máy (`docs/sources/` bị git bỏ qua) tới lúc chạy.
  - Giờ push trên GitHub là dấu thời gian của bên thứ ba; sha256 buộc nội dung.
- **Chạy, đúng một lần.**
  - Chép `h3.yaml` vào `docs/g7/holdout-h3/` (băm phải khớp cam kết), commit + push.
  - Ngay sau đó: `scripts/chay-giu-lai.sh docs/g7/holdout-h3/h3.yaml docs/g7/holdout-h3/ket-qua-h3`.
  - Cùng thủ tục và luật từ chối của §7: hai đường, `eais_giu_lai`, cây sạch, HEAD trên origin/main, không chạy lại khi đã HOÀN TẤT.
- **Điều kiện sẵn sàng chạy (đăng ký trước).**
  1. Bản sửa của ca đã commit + push. `go test ./...` có PG xanh, ratchet xanh, mọi danh mục đột biến bị bắt hết.
  2. H1 (bộ hồi quy, mốc `15ed135`: 89/104) không có ca ĐẠT → ĐỎ. Ngoại lệ: ca mà người soát độc lập xác nhận kỳ vọng trái hợp đồng câu xác nhận; ghi lý do.
  3. 15 câu thăm dò câu xác nhận: 0 phán quyết ("Đúng." / "Không.") sai.
  4. Soát đối kháng độc lập đã xong; mọi phát hiện đã sửa hoặc ghi lại.
  5. Phiên viết mã chưa thấy nội dung H3.
  - Không đủ ⇒ không chạy trong ca; ghi lại, để ca sau.
- **Chấm.**
  - `scripts/bang-cong-h1.py` giữ nguyên: G2' / G5' / G6' như §8; số ĐẠT từng lớp trên cả hai đường.
  - G1 / G3 / G4 chỉ xét phần ca của `04` §6 (100% ca lớp tương ứng). Phần cơ chế của các cổng không đo ở đây.
  - Lớp < 13 ca vẫn chấm, báo đúng số, không độn.
  - **Độ nhạy đăng ký trước (thêm 21/10 ca ngày, trước khi có tệp H3, chỉ báo cáo, KHÔNG đổi cách tính đạt):** hợp đồng câu xác nhận (§8 hợp đồng vận hành; ADR-0009) cho ba cách không phán quyết — trả số, hỏi lại, báo thiếu bằng chứng — còn tờ định dạng của người soạn ghi "hỏi lại hoặc thiếu bằng chứng". Vì vậy báo riêng số ca xác nhận trượt CHỈ vì outcome RESOLVED (trả số, không phán quyết) trong khi kỳ vọng là `CLARIFICATION_REQUIRED` / `INSUFFICIENT_EVIDENCE`. Các ca đó vẫn tính là TRƯỢT.
    - Cách đếm (cố định trong `scripts/bang-cong-h1.py` "độ nhạy 3" + test `TestBangCongDoNhay3` + đột biến BN1–BN7 của `n27`, commit TRƯỚC lần chạy H3): ca xác nhận = câu hỏi chứa một cụm `confirm_markers` của pack (so sau khi bỏ dấu + chữ thường); chỉ xét ca có outcome kỳ vọng `CLARIFICATION_REQUIRED` / `INSUFFICIENT_EVIDENCE`; ca được đếm khi ở MỌI đường đỏ, câu trả lời không mở đầu bằng "Đúng." / "Không." và lý do đỏ — mức CHẶT: chỉ "outcome = RESOLVED, muốn <outcome kỳ vọng>"; mức NỚI: thêm được lý do route kéo theo "route = <X>, muốn CLARIFICATION". Báo cả hai mức.
- **Đối chứng mốc (chỉ chẩn đoán, không tính cổng).**
  - SAU lần chạy chính thức: chạy cùng `h3.yaml` trên revision mốc `016422e` (mã = `15ed135`).
  - Chạy ở worktree riêng: `winreport` hai đường, cùng dữ liệu. Ghi vào `docs/g7/holdout-h3/ket-qua-h3-moc/`.
  - Cả hai lần đều chạy sau khi mã đã chốt, nên không lần nào chỉnh theo H3 được. So hai lần cho biết bản sửa có tổng quát hoá không.
- **Sau khi chạy.**
  - H3 thành REGRESSION.
  - Ca trượt: ghi tầng sai đầu tiên + bất biến bị hỏng.
  - Không sửa kỳ vọng. Ca mà người soát độc lập xác minh sai kỳ vọng chỉ được loại khỏi phép tính, kèm lý do.
  - Cần chứng minh độc lập tiếp ⇒ soạn H4 TRƯỚC vòng sửa kế.

## Hệ quả

- **H1 có thể lộ lỗi thật,** khi đó G2 / G5 / G6 FAIL trên bằng chứng độc lập. Đó là kết quả đúng. "FAIL sạch hơn PASS bẩn."
- **Kho ca không đủ ca khác nhau thật** thì H1 < 104 và G6' có thể chỉ PARTIAL. Không độn ca.
- **Một bộ giữ lại chỉ dùng được một lần** cho chứng minh độc lập. Sửa mã theo nó thì phải soạn bộ mới, chưa ai chạm.

## Ghi chú trạng thái 24/10/2026 (thêm sau, không sửa lời trên; D14 của báo cáo audit vNext)

Cờ `content_published: false` trong manifest **không** cập nhật khi chạy, nên đọc cờ sẽ hiểu sai. Ba bộ đã công bố dù cờ còn `false`; nội
dung trong repo khớp sha256 đã cam kết (kiểm 24/10 bằng `sha256sum` so `files_sha256` của manifest):

| Bộ | Tệp đã công bố | Commit công bố |
|---|---|---|
| H3 | `docs/g7/holdout-h3/h3.yaml` | `9091c84` (21/10) |
| H4 | `docs/g7/holdout-h4/h4.yaml` | `57409f2` (22/10) |
| P1 | `docs/g7/holdout-p1/p1.yaml`, `world-p1.yaml` | `18fb206` (23/10); `soat-ca.md` công bố dưới dạng đã che đường dẫn (`soat-ca.da-che-duong-dan.md`, `d614395`), nên bản trong repo không so được với băm |

Manifest là tạo phẩm đã băm: **không lật cờ** trong manifest. H6 và 0a chưa chạy, nội dung chưa có trong repo.
