<!-- Bản công bố của soat-ca.md (tệp thứ ba của holdout-manifest.json) sau lần chạy duy nhất của P1, 23/10/2026: chỉ thay đường dẫn nháp của máy bằng $S / <tmp> (repo công khai), nên sha256 KHÁC giá trị ghi trong manifest; bản nguyên văn khớp băm nằm ở bản sao lưu ngoài repo. -->

# Soát độc lập bộ ca giữ lại Phase 1 (lệnh `open_discrepancy_ticket`)

> Người soát độc lập (không phải người soạn, không phải người viết mã). Ngày soát: 2026-10-22.
> Đối tượng: `p1.yaml` (154 ca) và `world-p1.yaml` (thế giới bổ sung).
> **Kết luận: ĐẠT CÓ GHI CHÚ.** Không có ca kỳ vọng sai, không có ca phải bỏ (S1 = 0). Còn 3 phát hiện S2 và 15 ca S3 cần xử lý trước khi đóng băng (mục 4).

## 0. Tệp soát, phạm vi đọc, dấu vân tay

| Tệp | sha256 lúc soát | So với lời khai của người soạn |
|---|---|---|
| `p1.yaml` | `612fe2137040e53ce812652d5f3e81d6391badb6f084c1785258444a640b809d` | khớp |
| `world-p1.yaml` | `85e6aa67b8108bff4421f37a7272da8029dfef048b6358492718ef9cc2c40b9e` | khớp |
| `kit/du-lieu/world.yaml` (bản gốc) | `1631a6c6af835bf6c2c168b69226d203284ff608d0a3948d45736ed93251e39e` | không có để so |

**Đã đọc:** `kit/HOP-DONG-phan-duoc-nhan.md`, `kit/CONVENTIONS-muc-5.md`, `kit/du-lieu/world.yaml`, `p1.yaml`, `world-p1.yaml`, và tệp tạm tiền tố `hp1s-` của người soát.
**Không đọc:** mã `*.go`, test, pack (`tenants/*/pack`), `evals/`, `scripts/`, `docs/` khác, git, `kit-manifest.json`, scratchpad của người khác. Không chạy EAIS, không gọi model, không tìm khoá. Mọi lệnh python chạy với `-I`.

Ghi chú kiểm toán (để người đối chiếu nhật ký công cụ khỏi hiểu nhầm):
- Có một lần `ls -la` ba thư mục được phép (`holdout-p1/`, `kit/`, `kit/du-lieu/`): thấy tên `kit-manifest.json`, không mở.
- Nội dung `CLAUDE.md` của repo xuất hiện trong ngữ cảnh do harness nạp tự động sau lần đọc đầu tiên trong cây; người soát không mở tệp đó và không dùng nó làm căn cứ chấm.
- Một lần xuất quá cỡ khiến harness tự lưu đầu ra ra tệp `tool-results`; người soát không đọc tệp đó, đã chạy lại với đầu ra nhỏ hơn.

## 1. Kết quả ngắn

| Mức | Số lượng | Nội dung |
|---|---|---|
| S1 (kỳ vọng sai / mơ hồ phải sửa hoặc bỏ) | **0 ca** | mô hình độc lập khớp 154/154, không ca nào dựa vào thứ tự đến |
| S2 (thiếu phủ, sai khuôn) | **3 phát hiện** | S2-01 mẫu id (hệ thống, 154 ca); S2-02 lớp không khai được bằng §12.1 (do khuôn); S2-03 thiếu biến thể A10 (sửa được) |
| S3 (nên sửa) | **15 ca** / 4 nhóm | S3-01 (7 ca), S3-02 (4 ca), S3-03 (1 ca), S3-04 (3 ca) |
| S4 (ghi chú) | 77 ca nêu tên (67 ca chỉ có S4) | mục 4.3; không cần sửa nội dung ca |

**Quy ước đếm.** `so_ca_co_co` = số ca khác nhau có ít nhất một cờ S1 đến S3 mang tính riêng của ca (= 15). Cờ S2-01 áp cho 154/154 ca nhưng là MỘT phát hiện hệ thống, không cộng vào số ca. Ghi chú S4 không cộng.

Loại cờ (cột "loại" ở dưới): `sai_ky_vong`, `mo_ho` (hai cách đọc hợp đồng cho hai kỳ vọng), `trung`, `sai_khuon`, `phu_thuoc_thu_tu_den`, `gia_dinh_ngoai_hop_dong` (kỳ vọng đúng với hợp đồng nhưng treo vào giả định nằm ngoài phần được nhận), `van_ban_mau_thuan`, `thieu_phu`, `ghi_chu`.

| Loại | Số ca (S1 đến S3) |
|---|---|
| `sai_ky_vong` | 0 |
| `mo_ho` | 1 (S3-03) |
| `trung` | 0 (ca trùng hoàn toàn: không có; xem S4-06 cho ca thừa) |
| `phu_thuoc_thu_tu_den` | 0 |
| `gia_dinh_ngoai_hop_dong` | 11 (S3-01: 7, S3-02: 4) |
| `van_ban_mau_thuan` | 3 (S3-04) |
| `sai_khuon` | 0 ca riêng lẻ; 154 ca vì mẫu id (S2-01, hệ thống) |
| `thieu_phu` | 9 mục (7 lớp, A29 (a) đến (d), 1 biến thể A10); không phải ca |

**Việc cần làm trước khi đóng băng (theo thứ tự):**
1. Người điều phối chốt mẫu id (S2-01); sau khi băm thì không đổi được nếu không đóng băng lại.
2. Thêm 1 đến 2 ca A10 với người gọi tenant lạ và thiếu quyền, kèm người gọi tương ứng trong `world-p1.yaml` (S2-03).
3. Nới chênh lệch của RL-32 và thay RL-31 ở ca 013, 014, 020 (S3-01).
4. Xác nhận với chủ bộ sinh việc render và nạp cờ approx của hai cách ghi (S3-02).
5. Tách ca 060 thành hai ca (S3-03).
6. Sửa dòng văn bản mâu thuẫn ở ca 004, 027, 074 (S3-04).
7. Tuỳ chọn: gắn nhãn "phụ thuộc Q-P1-7" (S4-01) và "phụ thuộc pack" (S4-03) vào `business_rules`.
8. Sau sửa: tính lại dấu vân tay, chạy lại soát (mục 10).

## 2. Phương pháp và bằng chứng

Mọi con số dưới đây là kết quả lệnh vừa chạy trên hai tệp có dấu vân tay ở mục 0. Mô hình của người soát được viết từ hợp đồng và `world-p1.yaml`, KHÔNG xem mô hình hay tệp tạm của người soạn. Mô hình dùng cùng hợp đồng nên đây là kiểm chéo hai cách suy, không phải bằng chứng hợp đồng được hiểu đúng.

| # | Việc | Lệnh / cách làm | Kết quả |
|---|---|---|---|
| 1 | Kỳ vọng từng ca | mô hình tham chiếu `hp1s-model.py`: C0 đến C7, R1 đến R7, F1 đến F8, sổ lệnh, audit, ba bộ đếm; ca đua duyệt MỌI thứ tự commit | 154/154 khớp (có xét audit của bên thua đua); không thứ tự nào làm ca lệch |
| 2 | Ca đua | `hp1s-model.py`, 14 ca đua, 482 thứ tự commit; kiểm cả N lời gọi đều qua C0 đến C5 để tới rào | 0 ca lệch ở mọi thứ tự; 14/14 ca đều tới rào |
| 3 | Đo độ nhạy của bộ so | `hp1s-mut.py`: sửa từng giá trị kỳ vọng (+1 hoặc đảo dấu hoặc cờ) | 1647 lần sửa, 1647 lần bị mô hình bắt, 0 lọt |
| 4 | Hình dạng §12.1 và §5 | `hp1s-shape.py` | 0 lỗi hình dạng; 0/154 id khớp `CON-T-NNN` (xem S2-01) |
| 5 | Kiểu YAML | `yaml.compose` tìm scalar không nháy bị hiểu khác chuỗi | chỉ còn số nguyên, true/false, null đúng ý; không có ngày hay `y/n/on/off` trần ở tệp ca |
| 6 | Khoá trùng trong bản đồ | loader tự viết báo khoá trùng | 0 (cả hai tệp) |
| 7 | Ký tự điều khiển | quét byte thô | 0 byte điều khiển, 0 CRLF, 0 tab, 0 BOM; 4 ca NUL dùng chuỗi thoát `\u0000` trong chuỗi có nháy |
| 8 | `world-p1.yaml` so với gốc | `diff world.yaml world-p1.yaml` | 0 dòng bị sửa hoặc xoá; 218 dòng thêm; 54 dòng chứa `[P1+]` |
| 9 | Thế giới | tập lệnh riêng (mục 6) | bản ghi gốc nguyên văn; không trùng id, `ref`, `source_ids`, bí danh (ngoài cặp `máy 07` có sẵn) |
| 10 | Người gọi, vai, phạm vi | đối chiếu từng mục `input` với `principals` của world | 0 lệch |
| 11 | `sources`, `preconditions` | so với nguồn thực sự của (máy, ngày) | `sources` khớp 154/154; 180 dòng `world-p1:` của `preconditions` khớp số trong bản ghi (0 lệch) |
| 12 | Đột biến độc lập | `hp1s-mutants.py`: 77 lỗi cài đặt giả định trên mô hình | 74 bị ít nhất 1 ca bắt; 3 sống sót (mục 7) |
| 13 | Đối chiếu văn bản với kỳ vọng | quét `must_not_do`, `acceptance_criteria` so với `tickets_open`, `ledger_rows`, số lời gọi | 3 ca mâu thuẫn (S3-04), 15 ca viết theo lời gọi (S4-09) |

Các mệnh đề hợp đồng làm căn cứ chính: thứ tự C0, C1, C3, C4, C4b, C5 (§3 và §7.5); khoá lệnh (tenant, `Principal.ID`, `command_id`) và fingerprint (§6.2); lệnh bị từ chối không tiêu thụ khoá (§6.2); phân loại lại sau 23505 (§6.4); một dòng audit cho mọi kết cục từ C4 (§7.3, §3 dòng C7); ba bộ đếm (§10); tính chất F1 đến F8 và cách chạy (§12.1).

## 3. Phủ

### 3.1 Phủ lớp §10

| Lớp | Số ca | Biến thể đã có |
|---|---|---|
| A01 | 7 | vai (director, accountant, project_manager), gửi 2 và 3 lần, xen kẽ hai khoá, chênh lệch âm, thập phân, vế kho xuất chỉ ERP, chỉ bảng Excel, ngày 18/10 |
| A02 | 4 | N = 2, 3, 4, 5; `repeat` và mục riêng; vai project_manager; thập phân |
| A02b | 5 | N = 2 đến 5; khác máy, khác ngày, trộn |
| A03 | 7 | khác máy, khác ngày, khác cả hai; sau INVALID_REQUEST và PERMISSION_DENIED vẫn chưa tiêu thụ khoá; nội dung trùng phiếu của khoá khác |
| A04 | 6 | 2 và 3 khoá, hai người gọi, trộn, số ước lượng, bị từ chối lặp (bộ đếm 1 và 2) |
| A04b | 5 | N = 2, 3, 4, 5; bốn người gọi khác nhau; hai tầng chống trùng cùng lúc |
| A05 | 7 | 2 vai, 3 người, cùng vai khác id, nội dung khác, gửi lại sau, hai project_manager khác dự án |
| A06 | 7 | thứ tự đảo, khoá khác, mỗi tenant gửi lại, cách ly số liệu (B thiếu vế, B đủ vế), id người gọi trùng giữa tenant |
| A07 | 7 | site_engineer, driver, trong và ngoài phạm vi, hai người, sau đó người có quyền dùng cùng UUID |
| A08 | 8 | ngoài phạm vi, không tồn tại, ký tự SQL (4 kiểu), chữ Cyrillic, bí danh và mã nguồn, tenant B gọi máy chỉ có ở A, quyền kiểm trước số liệu, không lộ qua phiếu |
| A10 | 5 | hai chiều tenant, sau khi A đã có phiếu, hai runtime xen kẽ, thứ tự C1 trước C3 |
| A13 | 26 | thiếu, rỗng, không phải UUID (5 kiểu), NUL ở 4 trường, ngày sai dạng (8 kiểu), hôm nay, ngày mai, xa, chỉ số ngoài danh sách (6 giá trị), khoá là dự án (2 vai), 3 ca thứ tự chặng, 1 ca dương (UUID phiên bản 0 và 1) |
| A14 | 7 | bằng nhau (số nguyên), bằng nhau khác cách viết thập phân, ngày 18/10 gửi hai lần cùng khoá, hai khoá khác nhau, +1, -1, +7 |
| A15 | 10 | thiếu vế đo, thiếu vế kho xuất, không bản ghi, ngày khác, máy gốc, chỉ báo cáo và hoá đơn, ba máy |
| A16 | 7 | ba cặp xung đột (hai chiều lệch), máy gốc thiếu vế đo, xung đột kèm thiếu vế |
| A17 | 4 | ước lượng có raw và xlsx, ước lượng không raw, gửi lại, đối chứng có nhiễu |
| A18 | 4 | phát lại sau Load, khoá khác sau Load, ngày khác sau Load, Load trước cả hai |
| A19 | 4 | F1 một lời gọi, gửi lại cùng khoá, gửi lại khoá mới, vai project_manager |
| A20 | 4 | F2 một lời gọi, gửi lại cùng khoá, khoá mới, thập phân |
| A21 | 9 | F3 (4 ca), F4 (5 ca): gửi lại 1 và 2 lần, khoá mới, thập phân |
| A22 | 3 | F5: gửi lại 1 và 2 lần, khoá mới |
| A23 | 5 | F6 lỗi và F6 lệch; gửi lại 1 và 2 lần |
| A29 | 3 | chỉ phần (e): F8 trên lượt DONE, DONE rồi gửi lại, lượt bị từ chối |

Theo điểm chèn lỗi: F1 4, F2 4, F3 4, F4 5, F5 3, F6 `readback_fail` 3, F6 `readback_stale` 2, F7 14, F8 3 (tổng 42 ca có chèn lỗi; 112 ca không).
Tổng 300 lời gọi sau khi mở `repeat`; 63 ca một lời gọi, 52 ca hai, 27 ca ba, 8 ca bốn, 4 ca năm.

### 3.2 Lớp không phủ

| Mục | Lý do | Mức |
|---|---|---|
| A09 | theo đề giao (ca phát triển dựng Policy bằng tay) | không tính |
| A11, A12, A24, A25, A26, A27, A28 | §12.1 không có khoá để dựng pack hỏng, trường lạ trong thân, đếm lời gọi model, câu lệnh tự do, quyền cơ sở dữ liệu, so hai commit, hay cửa HTTP. Người soát tự đối chiếu từng lớp với bảng khoá: lời khai của người soạn đúng | S2-02 |
| A29 (a) đến (d) và hai ý của (e) | không có khoá đọc vết hay bộ đếm lỗi lưu vết | S2-02 |

## 4. Cờ chi tiết

### 4.1 Mức S2

**S2-01 · `sai_khuon` · mẫu `id` (154/154 ca)**
- Căn cứ: §12.1, đoạn "Tệp ca": "`id` đúng mẫu `CON-T-NNN` của `evals.Validate` và duy nhất trong tệp". `p1.yaml` dùng `CON-T-701` đến `CON-T-854`; 0/154 khớp `CON-T-NNN`. Người soạn đã khai theo đề giao và báo trước.
- Rủi ro: nếu `evals.Validate` kiểm mẫu bằng biểu thức cứng thì cả 154 ca bị từ chối trước khi chạy ca nào (người soát không đọc mã nên chưa đo). Ngược lại giữ tiền tố `CON-T-` có thể trùng id của bộ ca khác (hợp đồng tự dẫn `CON-T-010` ở §1.5 như một id đã có), nên đề giao có thể có lý do.
- Cách sửa: người điều phối chốt MỘT hướng TRƯỚC khi đóng băng: (a) đổi tiền tố hàng loạt, nội dung ca không đổi, tính lại băm; hoặc (b) xác nhận bằng chạy `evals.Validate` rằng tiền tố `CON-P1-` được nhận và ghi quyết định vào §12.1. Đổi sau khi băm là đóng băng lại.

**S2-02 · `thieu_phu` · lớp và quy tắc không khai được bằng §12.1 (do khuôn, không phải lỗi người soạn)**
- Lớp: A11, A12, A24, A25, A26, A27, A28, A29 (a) đến (d) (mục 3.2).
- Hai quy tắc của hợp đồng mà MỌI ca §12.1 đều không bắt được: (i) "Quyền vẫn kiểm trước khi phát lại" (§3): hai mục cùng `id` phải cùng vai và phạm vi nên không dựng được người đã mất quyền; một bản cài phát lại trước kiểm quyền không ca nào làm đỏ (đột biến sống sót, mục 7). (ii) "A21 (a) vẫn đúng sau khi dựng pipeline mới": một ca chỉ có một điểm chèn lỗi, F3 và F4 không kết hợp được.
- Cách xử lý: ghi rõ trong báo cáo cổng rằng `p1.yaml` KHÔNG phủ các lớp và quy tắc này; cần kênh khác (kiểm tĩnh, bộ ca HTTP giữ lại, đột biến) hoặc thêm khoá vào §12.1 (đổi khuôn là đóng băng lại).

**S2-03 · `thieu_phu` · A10 thiếu biến thể "người gọi tenant lạ và thiếu quyền" (sửa được)**
- Mọi người gọi tenant lạ trong A10 đều là `director` (đủ quyền). Vì vậy một bản cài kiểm quyền trước kiểm tenant (C4 trước C1) cho kết quả y hệt: trong 77 đột biến thử, đột biến này sống sót (0 ca bắt). Hợp đồng §3: C1 từ chối `PERMISSION_DENIED` KHÔNG audit; C4 từ chối có audit `REFUSED`; hai thứ chỉ phân biệt được khi người gọi thiếu quyền.
- Cách sửa: thêm 1 đến 2 ca A10: người gọi có `tenant` khác `runtime`, vai thiếu quyền (driver hoặc site_engineer), yêu cầu hợp lệ về mặt cấu trúc; kỳ vọng `results: {PERMISSION_DENIED: 1}`, mọi dòng audit bằng 0, `tickets_open` và `ledger_rows` bằng 0. Thêm người gọi tương ứng vào `world-p1.yaml` (vai thiếu quyền của tenant còn lại).

### 4.2 Mức S3

| Mã cờ | Loại | Ca | Lý do | Cách sửa |
|---|---|---|---|---|
| S3-01 | `gia_dinh_ngoai_hop_dong` | CON-T-705, 011, 013, 014, 020, 133, 142 | Các ca này dùng RL-32 (chênh lệch 0.2 L) hoặc RL-31 (7 L) chỉ để thử việc khác (định dạng thập phân, đua, đổi khoá, F2, F4). Chúng đỏ nếu `min_abs_diff` của pack lớn hơn mức đó. §5 R4 nói mặc định 0, nhưng §1.4 nói ngưỡng của khối M4 nằm đúng ở tham số này. Ở ca đua (011, 013, 014) lời gọi bị R4 từ chối không tới rào nên "không đủ N lần tới rào" biến lệch tham số thành lỗi đồng thời. Mô hình đo: ngưỡng 1 L làm lệch 7 ca (005, 011, 013, 098, 099, 133, 142); ngưỡng 10 L làm lệch 10 ca (thêm 014, 020, 100) | Nới chênh lệch của RL-32 trong `world-p1.yaml` (ví dụ kho xuất "1000.3", cảm biến "960.1", hiệu "40.2"); trong ca 013 thay RL-31 và RL-32, trong ca 014 và 020 thay RL-31, bằng máy có chênh lệch lớn (EX-11, EX-12, EX-19, DZ-41). Sửa lại `diff`, `left`, `right` của 005, 011, 142 và văn bản `preconditions` của mọi ca bị đổi. Chỉ giữ các ca biên 098, 099, 100 phụ thuộc ngưỡng (S4-03) |
| S3-02 | `gia_dinh_ngoai_hop_dong` | CON-T-728, 119, 120, 121 | Kỳ vọng `approx: true` đi qua đường nạp. §5 R6 tự cảnh báo: "đường nạp hiện còn làm mất cờ ở vài định dạng". Thế giới gốc không có bản ghi có cấu trúc nào mang cờ approx (chỉ tin chat và báo cáo hiện trường pdf), nên hai cách ghi của người soạn (`raw` "≈ 2,6 m3" kèm `format: xlsx`; không `raw`, chỉ cờ) chưa từng được bộ sinh hay đường nạp thử. Nếu cờ mất thì ca đỏ vì nạp (L1), không vì lệnh | Trước khi đóng băng, xin chủ bộ sinh (W1) xác nhận bằng văn bản rằng cả hai cách ghi được render và nạp giữ cờ. Chưa xác nhận được thì bỏ hoặc thay cách ghi đó. Ghi vào báo cáo cổng cách phân loại đỏ vì nạp khác đỏ vì lệnh |
| S3-03 | `mo_ho` | CON-T-760 | Ca gộp 4 yêu cầu trong một ca: hai bí danh (`máy 11`, `MX-03`) và hai mã nguồn khác (`EX011` là mã cảm biến, `asset-8392` là mã ERP). §2.1 chỉ nói "không bí danh, không so khớp mờ"; không nói mã nguồn khác có thuộc "bí danh" hay không (người soạn dẫn §2.1 cho cả hai). Một yêu cầu sai làm đỏ cả ca và che ba yêu cầu còn lại. Đột biến "nhận bí danh hoặc mã nguồn làm khoá" chỉ ca này bắt được | Tách làm hai ca: (a) hai bí danh (giữ); (b) hai mã nguồn, gắn nhãn "suy rộng §2.1", hoặc bỏ (b) nếu chủ hợp đồng không xác nhận |
| S3-04 | `van_ban_mau_thuan` | CON-T-704, 027, 074 | Dòng văn bản mẫu của lớp mâu thuẫn với chính kỳ vọng của ca. 004: `must_not_do` "không tạo phiếu thứ hai" trong khi kỳ vọng hai phiếu (hai khoá, hai máy). 027: cùng dòng đó và dòng nghiệm thu mẫu A04 "phiếu = 1, sổ lệnh = 1" trong khi `tickets_open`, `ledger_rows` bằng 3 (dòng biến thể ghi "ba phiếu"). 074 (ca dương UUID phiên bản 0 và 1, mong 2 phiếu): mang dòng mẫu A13 âm "không ghi phiếu", "không tiêu thụ khoá lệnh khi từ chối", "không điền giá trị mặc định" | Viết lại các dòng theo biến thể: ví dụ "không tạo phiếu thứ hai cho cùng khoá (cùng máy, ngày)". Bộ chạy không đọc các trường này nhưng người đọc ca đỏ sẽ bị dẫn sai |

### 4.3 Mức S4 (ghi chú, không cần sửa nội dung ca)

| Mã | Ca | Nội dung |
|---|---|---|
| S4-01 | 24 ca: 001 đến 007, 020, 039, 045, 120, 123, 126, 135, 138 đến 142, 144, 145, 148, 151, 153 | Phụ thuộc thứ tự C4b trước C5 (tinh chỉnh `CLAUDE.md` §14), mà hợp đồng §3 nói còn là ĐỀ XUẤT, chờ câu Q-P1-7. Hợp đồng chỉ nêu "ca A01, A21 đổi theo"; thực tế 24 ca trong 10 lớp phụ thuộc (A01: 7, A21: 6, A18: 2, A22: 2, A23: 2, A03, A05, A06, A17, A29: mỗi lớp 1). Đề nghị gắn dòng "phụ thuộc Q-P1-7" vào `business_rules` các ca này để nếu chủ dự án đổi thứ tự thì chỉ làm lại đúng nhóm này |
| S4-02 | 14 ca đua: 008 đến 016, 030 đến 034 | Khẳng định số dòng audit `REPLAYED` hoặc `REFUSED` của bên thua đua. §6.4 không nói; suy từ §3 dòng C7 ("Kết cục nào từ C4 trở đi ... được ghi một dòng audit riêng") và §7.3 ("Mọi lệnh đi qua C4 trở đi để lại đúng một dòng"). Nếu hiểu đường phân loại lại không ghi audit thì cả 14 ca lệch chỉ ở dòng audit. Khuyến nghị GIỮ: nó bắt lỗi cài "bên thua bỏ audit"; người soạn đã nêu đúng đây là suy ra |
| S4-03 | 098, 099, 100 | Ca biên +1 L, -1 L, +7 L đúng khi `min_abs_diff` của pack = mặc định 0 (đã ghi trong `acceptance_criteria`). Giữ, gắn nhãn phụ thuộc pack |
| S4-04 | 068, 075, 078, 092, 093, 094 | Thứ tự chặng C0 < C1 < C3 < C4 suy từ thứ tự bảng §3 và cột "Dừng ở đây thì" của §7.5, không có câu nào viết thẳng. 075 coi khoá rỗng là "thiếu" (§2.4, §12.1 "khoá vắng là trường rỗng") chứ không phải "không tồn tại"; 078 theo ngoại lệ NUL của §4.4 |
| S4-05 | `diff` âm: 002, 025, 035, 099, 121, 124, 130, 144, 148, 153; thập phân: 005, 011, 142 | Chuỗi `"-80"`, `"-1"`, `"0.2"`: §2.5 chỉ minh hoạ `"68"`. Dạng chuẩn của `core/decimal` là suy ra hợp lý; không ca nào khẳng định dạng có chữ số 0 thừa |
| S4-06 | 079, 080, 081 | Không tách được kiểm NUL khỏi kiểm định dạng: UUID, `metric`, `day` chứa NUL vốn đã sai định dạng. Chỉ ca 078 (NUL trong khoá máy) phân biệt được. Không sai; giữ như ca chống treo, hoặc bỏ |
| S4-07 | 144, 145, 146 | Kết quả lời gọi sau F5 dựa vào việc COMMIT bị giữ đã thực sự hạ xuống trước khi lời gọi sau bắt đầu; §12.1 cam kết điều đó ("cố định"). Nếu bộ chạy thả không đồng bộ thì ca ngẫu nhiên; trách nhiệm của bộ chạy |
| S4-08 | 148, 151 | A23 viết "không bao giờ DONE" cho lượt hỏng đọc lại; hai ca khẳng định DONE ở lần gửi lại, đúng theo §9 T5 |
| S4-09 | 017 đến 023, 055, 066, 067, 127, 129, 130, 132, 134 | Dòng mẫu như "không ghi phiếu hay sổ lệnh mới", "không để lại phiếu hay dòng sổ lệnh sau F1" nói về MỘT lời gọi nhưng đọc như cả ca, trong khi ca có lời gọi khác ghi phiếu. Đề nghị thêm "ở lời gọi bị từ chối:" |
| S4-10 | 056 đến 063 | Cả 8 ca A08 mang dòng "không so khớp bí danh hay mã nguồn khác (§2.1)", chỉ ca 060 có biến thể đó |
| S4-11 | toàn bộ | Tỉ lệ `category`: happy_path 14 (9,1%), dirty_data 36 (23,4%), ambiguity 4 (2,6%), conflict 13 (8,4%), permission 32 (20,8%), infrastructure 32 (20,8%), adversarial 23 (14,9%). Tỉ lệ mục tiêu của CONVENTIONS là cho bộ ca mỗi tenant, không áp cho bộ giữ lại theo lớp |
| S4-12 | toàn bộ | 16 tên `metrics` do người soạn tự đặt; nếu bảng cổng nhóm theo tên chỉ số có sẵn thì đây là tên mới |
| S4-13 | A14 | Nửa sau của A14 ("không vượt `min_abs_diff`", khác 0) không dựng được khi chưa biết tham số của pack |
| S4-14 | `world-p1.yaml` | Thêm ngày 2026-10-18 (gốc không có); quy ước "dữ liệu nền không đụng vào thực thể và ngày ở đây" nay cũng chặn ngày này cho thực thể liệt kê. Máy thêm đều thuộc PRJ-A, PRJ-B nên đổi tổng cấp dự án của ngày 18 và 19/10 nếu có ca của bộ gốc chạy trên tệp này (không phải ý định) |
| S4-15 | tệp tạm | Tệp tạm của người soát có phép suy kỳ vọng (mục 10): không đưa cho người viết mã; xoá sau khi đóng băng |

## 5. Ca đua: không dựa vào thứ tự đến

Kết quả riêng cho yêu cầu "cờ ca dựa vào thứ tự đến ở ca song song": **0 ca**.
- 14 ca `race_barrier`, tổng 482 thứ tự commit được duyệt; ở mỗi thứ tự, phân bố `results`, `replayed_false`, `replayed_true`, `distinct_ticket_ids`, các trường `ticket` được khẳng định, `tickets_open`, `ledger_rows` và số dòng audit đều giữ nguyên.
- Các ca ép đua chỉ khẳng định các trường dùng chung của `ticket` (không nêu `subject_key` hay `day` khi hai bên khác máy hoặc khác ngày): đúng.
- Ca có hai khoá, mỗi khoá gửi hai lần song song (034): mọi thứ tự cho cùng phân bố (2 DONE gồm 1 phát lại, 2 DUPLICATE_OPEN_TICKET).
- Không ca nào khẳng định `write_unit_calls` hay `audit_writes` ở ca đua: đúng.

## 6. `world-p1.yaml`

- **Gốc nguyên văn:** 25 bản ghi gốc, các máy gốc, `projects`, `principals` gốc đều là tiền tố của danh sách mới, so từng phần tử (bằng nhau). `diff`: 0 dòng gốc bị sửa hoặc xoá.
- **Phần thêm:** abc-infra thêm 24 máy (EX-11 đến EX-23: 13 máy; RL-31 đến RL-37: 7 máy; DZ-41 đến DZ-44: 4 máy), 55 bản ghi, 3 người gọi (`u-pm-a`, `u-pm-b`, `u-director-2`); def-build thêm 1 máy (EX-11, cùng khoá với tenant A), 4 bản ghi, 1 người gọi (`u-director`, cùng id với tenant A). Tổng 84 bản ghi (gốc 25).
- **Định dạng:** cùng khoá và cùng kiểu với `world.yaml` (bản đồ một dòng, `value_l` là chuỗi, `on:` kế thừa từ gốc); 0 id bản ghi trùng; 0 `ref` trùng (kể cả giữa tenant); 0 `source_ids` trùng; bí danh chỉ trùng ở cặp `máy 07` có sẵn; mọi bản ghi trỏ về máy có trong danh mục (trừ `EX-99` cố ý của gốc); mọi `value_l` là số thập phân hợp lệ; loại máy khớp tiền tố.
- **Tính chất được chú thích:** người soát kiểm từng máy thêm với bản ghi của nó, đúng cho cả 25 máy thêm (24 của abc-infra, 1 của def-build): hai ngày dương (EX-11), âm (EX-12), bằng nhau hai ngày (EX-13), chỉ vế kho xuất (EX-14), chỉ vế đo (EX-15), không bản ghi (EX-16), ước lượng có `raw` và `xlsx` (EX-17), ước lượng không `raw` kèm âm (EX-22), ba cặp xung đột erp và excel_summary (EX-18, EX-21, DZ-44; lệch 200, 100, 50 L > 10 L), xung đột kèm thiếu vế đo (EX-23), đủ hai vế (EX-19, EX-20), chênh +7 (RL-31), thập phân (RL-32), bằng nhau về giá trị khác cách viết (RL-33), +1 và -1 (RL-36, RL-37), hai ngày dự án B (DZ-41), chỉ ERP (DZ-42), chỉ Excel (RL-34), có nhiễu (DZ-43), chỉ báo cáo và hoá đơn (RL-35), tenant B đủ hai vế hai ngày. Mỗi vế "sạch" đúng một bản ghi có cấu trúc; máy xung đột chỉ dùng cặp (erp, excel_summary) mà chú thích gốc của DZ-05 nêu là ngang thẩm quyền.
- **Có làm sai ca dựa trên `world.yaml` gốc không:** không. Máy và bản ghi gốc không đổi; các ca dùng máy gốc (EX-03 ở tenant A và B, DZ-05, RL-02) ra cùng kết cục như khi chỉ có `world.yaml`. Người soạn tránh đúng chỗ gốc không đủ để suy: EX-03, EX-07, RL-07 ngày 19/10 có hai nguồn (phiếu kho và ERP) mà tệp gốc không đủ để suy thứ hạng, nên không ca nào tính số trên các máy đó.
- **Điểm phải xử lý:** RL-32 nên có chênh lệch lớn hơn (S3-01); cách ghi cờ approx của EX-17 và EX-22 cần chủ bộ sinh xác nhận (S3-02); thiếu người gọi vai thiếu quyền ở tenant còn lại để dựng S2-03; ngày 18/10 mới (S4-14).

## 7. Đột biến độc lập

77 lỗi cài đặt giả định (thứ tự chặng, khoá lệnh, fingerprint, khoá nghiệp vụ, audit, bộ đếm, quyền và phạm vi, ngày, UUID, NUL, thập phân, dấu, approx, F1 đến F8, đua, Load, cách ly tenant). Phân bố số ca bắt: **0 ca: 3 đột biến; 1 ca: 13; 2 ca: 12; 3 đến 5 ca: 21; từ 6 ca: 28.**

Sống sót (0 ca bắt):
1. Phát lại trước kiểm quyền: không khai được bằng §12.1 (S2-02).
2. Khoá không phân biệt hoa thường: hợp đồng im lặng, người soạn cố ý tránh. Đúng.
3. Kiểm quyền trước kiểm tenant với người gọi tenant lạ: sửa được bằng S2-03.

Mỏng (chỉ 1 ca bắt): C1 trước C0 (094); C3 sau C4 (092); C4 trước C1 (092, nhưng đột biến này còn đặt C4 trước C3); C3 trước C1 (068); hoán C4 lên giữa C3 và C1 (068); UUID lỏng (073); chỉ nhận UUID phiên bản 4 và 7 (074); bỏ qua NUL (078); nhận bí danh hoặc mã nguồn (060); so sánh thập phân bằng chuỗi (096); approx lẫn từ chỉ số khác cùng ngày (122); mượn fact của tenant khác khi thiếu vế (046); phát lại mất cờ approx (120). Mỏng (2 ca): khoá nghiệp vụ bỏ ngày (027, 125); nhận ngày hôm nay (022, 084); ngày định dạng lỏng (082, 092); driver được tạo phiếu (050, 054); câu thông báo riêng cho ngoài phạm vi (056, 057); không kiểm loại thực thể (090, 091); thiếu vế trước xung đột (113, 116); F6 lệch vẫn DONE và thiếu audit `VERIFY_FAILED` (149, 150); lỗi lưu vết đổi DONE thành lỗi (152, 153); điền mặc định cho trường thiếu (076, 077).
Chỗ mỏng này khớp lời khai của người soạn; ngoài S2-03 không cần thêm ca, chỉ cần biết khi đọc báo cáo cổng.

## 8. Vấn đề ở mức hợp đồng (không phải lỗi ca)

1. §12.1 viết id `CON-T-NNN`, đề giao viết `CON-P1-NNN` (S2-01).
2. §6.4 không nói dòng audit nào được ghi cho bên thua đua; suy từ §3 dòng C7 và §7.3 (S4-02). Nên thêm một câu vào §6.4.
3. Thứ tự C4b trước C5 còn là ĐỀ XUẤT (Q-P1-7) và ảnh hưởng 24 ca ở 10 lớp, nhiều hơn A01 và A21 mà §3 nêu (S4-01).
4. §4.2 (ma trận vai) và §4.4 (một mã cho "không có" và "ngoài phạm vi") còn ĐỀ XUẤT; A05, A07, A08 phụ thuộc; §4.4 tự nói đổi sẽ đổi ca A08.
5. R4 nói `min_abs_diff` mặc định 0 nhưng §1.4 cho phép khối M4 đặt ngưỡng ở cùng tham số; các ca có chênh lệch nhỏ đúng có điều kiện (S3-01, S4-03).
6. R6 thừa nhận đường nạp mất cờ approx ở vài định dạng; chưa có tổ hợp nguồn có cấu trúc và định dạng nào được biết là giữ cờ (S3-02).
7. §2.1 "không bí danh, không so khớp mờ" không nói mã nguồn khác của cùng thực thể, chuẩn hoá hoa thường, khoảng trắng có thuộc phạm vi cấm không (S3-03).
8. §2.3 (invalid_fields "các trường") và §7.5 (C0 "trường sai đầu tiên") không thống nhất khi nhiều trường cùng sai; người soạn tránh bằng ca một trường sai.
9. Thứ hạng thẩm quyền giữa các loại nguồn nằm ở pack ngoài phần được nhận; bộ ca chỉ dựa vào cặp (erp, excel_summary) ngang thẩm quyền mà chú thích của `world.yaml` nêu, không phải hợp đồng.
10. §12.1 không nói thẳng bảng phiếu, sổ lệnh, audit được làm rỗng trước MỖI ca ("cơ sở dữ liệu riêng của lần chạy" đọc được là theo lần chạy cả bộ). 154 ca đều giả định mỗi ca bắt đầu từ trạng thái rỗng (các ví dụ 1 đến 5 của §12.1 cũng giả định như vậy). Nên viết thẳng.
11. §12.1 F5 nói COMMIT được thả "ngay sau khi lời gọi đó đã trả UNKNOWN, trước khi lời gọi kế tiếp bắt đầu" nhưng không nói việc thả có được chờ xong không (S4-07).
12. Câu A23 "không bao giờ DONE" chưa nói rõ chỉ áp cho lượt hỏng đọc lại (S4-08).
13. §2.3 không nói câu `detail` của "thiếu quyền" có trùng câu của "không tồn tại, ngoài phạm vi, khoá lạ" không; bộ ca chỉ khẳng định `distinct_details` trên tập cùng nguyên nhân.
14. Dạng chuỗi của số thập phân và số âm trong `ticket` chỉ có ví dụ `"68"` (S4-05).
15. Quy tắc "quyền kiểm trước khi phát lại" (§3) không có cách nào kiểm bằng khuôn §12.1 (S2-02).
16. A14 nhắc "không vượt `min_abs_diff`" nhưng không dựng được hạt giống khi chưa biết tham số của pack (S4-13).

## 9. Không kiểm được

- Không đọc mã nên chưa đo `evals.Validate` có nhận tiền tố id hay không.
- Chưa đo bộ đọc YAML của bộ chạy (Go) có giữ NUL từ chuỗi thoát `\u0000` hay không; máy không có bộ đọc thứ hai.
- Không đọc pack: `min_abs_diff`, danh sách chỉ số của tool, `entity_types`, giá trị dung sai, thứ hạng nguồn đều lấy từ hợp đồng và chú thích của `world.yaml`; chưa xác minh với pack thật.
- Chưa đo bộ sinh có render và nạp đúng `world-p1.yaml` (U11), đặc biệt cờ approx, số thập phân có chữ số 0 thừa, bản ghi `excel_summary` và `erp` đơn nguồn.
- Chưa đo bộ chạy thật: rào F7, thời gian thả của F5, hạn chót, việc làm rỗng cơ sở dữ liệu giữa các ca, hành vi chặn và khoá thật của PostgreSQL trong ca đua. Mô hình của người soát duyệt thứ tự commit, không mô phỏng khoá.
- Chưa kiểm tác động lên các ca WIN gốc (ngoài phạm vi đọc); `world-p1.yaml` chỉ dùng cho bộ giữ lại nên không ảnh hưởng nếu bộ gốc vẫn chạy trên `world.yaml`.
- Không có người nghiệp vụ duyệt; mọi kỳ vọng chỉ là suy từ hợp đồng trên dữ liệu giả lập.
- Mô hình người soát dùng cùng hợp đồng với người soạn nên hai cách suy có thể cùng hiểu sai một chỗ; điểm tựa duy nhất là câu chữ của hợp đồng đã dẫn.
- Lớp A11, A12, A24 đến A28 và A29 (a) đến (d) không có trong `p1.yaml`; chưa xem kênh khác phủ chúng.
- Không đọc tệp tạm của người soạn nên không kiểm được tính độc lập của phép suy bên đó; chỉ so đầu ra.

## 10. Tệp tạm của người soát (chứa phép suy kỳ vọng; không đưa cho người viết mã; xoá sau khi đóng băng)

- `$S/hp1s-dump.py`
- `$S/hp1s-model.py`
- `$S/hp1s-model2.py`
- `$S/hp1s-mut.py`
- `$S/hp1s-mutants.py`
- `$S/hp1s-mutants-out.txt`
- `$S/hp1s-shape.py`

Soát lại sau khi sửa: chạy lại `hp1s-model.py`, `hp1s-shape.py`, `hp1s-mutants.py` trên tệp mới (chi phí thấp), rồi tính lại dấu vân tay.

## 11. Đã xử lý (người soạn, lượt sửa 1)

> Người soạn (lượt sửa), tiền tố tệp tạm `hp1r-`. Ngày: 2026-10-22. Kết quả: **154 ca thành 157 ca**; không ca nào bị bỏ; không dùng lại id nào; id mới đánh tiếp từ `CON-T-855` và nằm cuối `p1.yaml`.
> Cách đọc: "sửa" = đã đổi tệp; "giữ" = cố ý không đổi, kèm lý do; "nhãn" = chỉ thêm dòng ghi chú vào `business_rules` (bộ chạy không đọc trường này).

### 11.1 Phạm vi đọc và ghi chú kiểm toán

**Đã đọc:** `kit/HOP-DONG-phan-duoc-nhan.md`, `kit/CONVENTIONS-muc-5.md`, `kit/du-lieu/world.yaml`, `kit/kit-manifest.json`, `p1.yaml`, `world-p1.yaml`, `soat-ca.md`, và tệp tạm tiền tố `hp1r-`.
**Không đọc:** mã `*.go`, test, pack (`tenants/*/pack`), `evals/`, `scripts/`, `docs/` khác, git, tệp tạm tiền tố khác (kể cả của người soát và của người soạn lượt đầu), scratchpad của người khác. Không chạy EAIS, không gọi model, không tìm khoá. Mọi lệnh python chạy với `-I`.

Ghi chú để người đối chiếu nhật ký công cụ khỏi hiểu nhầm:
- Một lần `ls -la` ba thư mục được phép (`holdout-p1/`, `kit/`, `kit/du-lieu/`).
- Có mở `kit/kit-manifest.json` (504 byte: danh sách mục và băm của ba tệp kit). Tệp nằm trong `kit/` nên người soạn coi là trong phạm vi, nhưng đề giao chỉ nêu tên ba tệp kia; ghi lại để người kiểm toán quyết định.
- Nội dung `CLAUDE.md` của repo xuất hiện trong ngữ cảnh do harness nạp tự động; người soạn không mở tệp đó bằng công cụ và không dùng nó làm căn cứ.
- Một lần xuất quá cỡ khiến harness tự lưu đầu ra vào tệp `tool-results`; người soạn không đọc tệp đó, đã chạy lại với đầu ra gọn hơn.

### 11.2 Kết quả ngắn

| Việc | Kết quả |
|---|---|
| Số ca | 154 thành **157**: mới `CON-T-855`, `CON-T-856` (A10) và `CON-T-857` (A08). 0 ca bỏ; ca cũ giữ id và thứ tự |
| Ca cũ có đổi | 63 ca (nêu ở 11.3); 91 ca cũ nguyên văn |
| Số lời gọi sau khi mở `repeat` | 300 thành 302 (65 ca một lời gọi, 54 ca hai, 27 ca ba, 7 ca bốn, 4 ca năm) |
| `world-p1.yaml` | 0 dòng gốc của `world.yaml` bị sửa hoặc xoá; 225 dòng thêm so với gốc (trước sửa: 218); đổi 3 bản ghi [P1+] (`R-P032`, `R-P004`, `R-P025`), thêm 1 người gọi (`u-b-eng`) |
| sha256 `p1.yaml` | `415436321b11b8c5c5992f80e79d89e29a8c6e3f7436bd7995d72b9672745ed3` |
| sha256 `world-p1.yaml` | `f46c3b467fd7149023ff2a46b02d9fd91fad606c95394b3076ee7fe69a47c27f` |
| Theo lớp | A01 7 · A02 4 · A02b 5 · A03 7 · A04 6 · A04b 5 · A05 7 · A06 7 · A07 7 · A08 9 (+1) · A10 7 (+2) · A13 26 · A14 7 · A15 10 · A16 7 · A17 4 · A18 4 · A19 4 · A20 4 · A21 9 · A22 3 · A23 5 · A29 3 |

### 11.3 Từng cờ: sửa, bỏ hay giữ

| Cờ | Xử lý | Chi tiết và lý do |
|---|---|---|
| S2-01 mẫu `id` | **giữ** (ngoài quyền người soạn) | Mẫu `CON-P1-NNN` do đề giao quy định; chọn (a) đổi tiền tố hay (b) xác nhận validator nhận tiền tố là việc của người điều phối (người soạn không đọc `evals`). Không đổi hàng loạt vì sẽ lệch đề giao và có thể đụng id của bộ khác. Nếu chọn (a): thay MỌI chỗ `CON-P1-` bằng tiền tố đã chốt trong `p1.yaml` (cả dòng `id:` lẫn vài dòng chữ nhắc id ở đầu tệp và ở ca 157), nội dung ca không đổi, rồi tính lại băm. Id mới đã đánh tiếp 155 đến 157 nên đổi tiền tố vẫn là phép thay thuần tuý |
| S2-02 lớp không khai được | **giữ** (do khuôn) | A11, A12, A24, A25, A26, A27, A28, A29 (a) đến (d) và hai ý của (e) không có khoá tương ứng trong §12.1. Hai quy tắc cũng không dựng được: (i) quyền kiểm trước khi phát lại cần một người đã mất quyền, nhưng khuôn bắt hai mục cùng `id` cùng vai và phạm vi, mà cùng fingerprint thì quyền cũng trùng; (ii) A21 (a) vẫn đúng sau khi dựng pipeline mới cần F3 và F4 trong một ca, khuôn chỉ cho một điểm chèn lỗi. Người soạn không thêm khoá (đổi khuôn là đóng băng lại). Việc cho báo cáo cổng: ghi rõ `p1.yaml` KHÔNG phủ các lớp và quy tắc này; cần kênh khác (kiểm tĩnh, bộ ca HTTP giữ lại, đột biến) |
| S2-03 A10 thiếu người gọi tenant lạ và thiếu quyền | **sửa** | Thêm `CON-T-855` (người tenant B, vai kỹ sư công trường, gọi runtime A) và `CON-T-856` (người tenant A, vai lái xe, gọi runtime B): yêu cầu hợp lệ về cấu trúc, kỳ vọng từ chối ở C1, mọi dòng audit bằng 0 (ở ca 156: cả runtime B); ca 155 còn khẳng định hai bộ đếm `write_unit_calls` và `audit_writes` bằng 0. `world-p1.yaml`: thêm người gọi `u-b-eng` (tenant B, vai thiếu `tickets.create`). Đo bằng đột biến "kiểm quyền (C4) chạy trước kiểm tenant (C1)": người soát báo 0 ca bắt; nay hai ca mới bắt cả hai biến thể thứ tự (C0, C4, C1, C3 và C0, C3, C4, C1), bản gốc chỉ bắt được qua các ca về thứ tự C3 (068, 092) |
| S3-01 RL-32, RL-31 phụ thuộc `min_abs_diff` | **sửa** | `world-p1.yaml`: RL-32 nới chênh lệch từ 0.2 L lên 40.2 L (cảm biến 960.1): vẫn thập phân, phép trừ số thực vẫn lệch ("40.19999999999993") nên ca còn bắt cách cài so số thực. Ca 005, 011, 133, 142 sửa số theo (005, 011, 142: `diff`, giá trị vế đo; 133: dòng precondition). Ca 013, 014, 020 thay RL-31 bằng DZ-41 (013 giữ RL-32 vì nay đã đủ lớn). Thêm một chỗ người soát chưa nêu: EX-11 ngày 18/10 (chênh 12 L) nới lên 45 L (cảm biến 2565) vì các ca 015, 016, 027, 043, 125 cần phiếu hoặc cần tới rào ở ngày đó; sửa dòng precondition ở 015, 016, 018, 022, 027, 043, 125. Còn phụ thuộc ngưỡng chỉ ba ca biên 098, 099, 100 (đã gắn nhãn, xem S4-03). Đo bằng mô hình của người soạn: ngưỡng 1 L làm lệch 7 ca ở bản cũ (005, 011, 013, 098, 099, 133, 142) và 10 ca ở ngưỡng 10 L (thêm 014, 020, 100), khớp số của người soát; bản mới: 2 ca ở 1 L (098, 099), 3 ca ở mọi ngưỡng từ 7 L đến 37 L (098, 099, 100); từ 38 L trở lên (số chênh lệch nhỏ nhất của các ca thường) 12 ca đổi |
| S3-02 cờ `approx` đi qua đường nạp | **sửa một phần, giữ có nhãn** | Người soạn không có kênh tới chủ bộ sinh và không đọc bộ sinh nên không xác nhận được. Việc rẻ nhất làm được: bỏ cách ghi "chỉ cờ, không `raw`" của EX-22 (mọi bản ghi `approx` của `world.yaml` gốc đều có `raw`; theo người soát, kiểu không `raw` chưa từng được bộ sinh hay đường nạp thử), thay bằng đúng kiểu của EX-17: `raw` có dấu "≈", đơn vị m3, `format: xlsx`. Còn lại MỘT kiểu ghi cho cả bốn ca (028, 119, 120, 121). Đã gắn nhãn vào bốn ca: cờ mất ở khâu nạp thì ca đỏ vì nạp (L1), không vì lệnh; báo cáo cổng phân loại riêng. Việc còn lại của người điều phối: chủ bộ sinh xác nhận `R-P013` và `R-P025` được render và nạp giữ cờ; chưa xác nhận được thì bỏ cả bốn ca (không thay thế được: R6 buộc có ca một vế ước lượng) |
| S3-03 ca 060 gộp bốn yêu cầu | **sửa (tách)** | Ca 060 giữ hai bí danh (có trong danh mục). Hai mã nguồn khác (mã cảm biến, mã ERP) chuyển sang ca mới `CON-T-857`, gắn nhãn "SUY RỘNG §2.1" và ghi rõ cách đọc "khoá chuẩn" là chỉ mã `id` của danh mục; bỏ ca 157 nếu chủ hợp đồng không xác nhận. Đo bằng hai đột biến: nhận bí danh làm khoá chỉ ca 060 bắt; nhận mã nguồn làm khoá chỉ ca 157 bắt (bản cũ: một ca 060 bắt cả hai) |
| S3-04 dòng mẫu mâu thuẫn kỳ vọng | **sửa** | Ca 004 (hai khoá, hai phiếu), 027 (ba phiếu, lời gọi thứ tư mới trùng), 074 (ca dương UUID: viết lại must_not_do và business_rules theo biến thể) |
| S4-01 phụ thuộc Q-P1-7 (24 ca) | **nhãn** | Gắn vào `business_rules` của 001 đến 007, 020, 039, 045, 120, 123, 126, 135, 138 đến 142, 144, 145, 148, 151, 153. Danh sách do mô hình của người soạn tính (chạy với C4b đặt sau luật R5, đúng 24 ca đổi kết cục) và trùng danh sách của người soát |
| S4-02 audit của bên thua đua | **giữ** | Khuyến nghị của người soát là giữ; §6.4 nên thêm một câu (mục 11.5) |
| S4-03 biên phụ thuộc pack | **nhãn** | Gắn vào 098, 099, 100: đúng khi `min_abs_diff` của pack bằng mặc định 0 |
| S4-04 thứ tự chặng suy từ bảng | **giữ** | Không có câu hợp đồng nào đảo được; ca 068, 075, 078, 092, 093, 094 giữ nguyên |
| S4-05 dạng chuỗi số âm, thập phân | **giữ** | §2.5 chỉ có một ví dụ; không ca nào khẳng định chữ số 0 thừa. RL-32 mới ("40.2") vẫn ở dạng chuẩn |
| S4-06 NUL không tách khỏi định dạng | **giữ** | Ca 079, 080, 081 giữ làm ca chống treo; chỉ ca 078 phân biệt được. Bốn chuỗi thoát NUL trong tệp ca không bị đụng (đếm lại: 4) |
| S4-07 F5 thả COMMIT | **giữ** | §12.1 cam kết "cố định"; trách nhiệm của bộ chạy |
| S4-08 A23 "không bao giờ DONE" | **giữ** | Hai ca 148, 151 khẳng định DONE ở lần gửi lại, đúng §9 T5 |
| S4-09 dòng mẫu nói một lời gọi nhưng đọc như cả ca | **sửa chữ** | Thêm giới hạn "lời gọi bị từ chối" hoặc "lời gọi gặp F1 / F2" ở 017 đến 023, 055, 066, 067, 127, 129, 130, 132, 134; thêm cùng loại chỉnh ở 059 (acceptance), 132, 134 (acceptance F2) và 154 (ca từ chối, không phải DONE). Không đổi kỳ vọng nào |
| S4-10 dòng "bí danh hoặc mã nguồn khác" ở cả tám ca A08 | **sửa chữ** | Bỏ khỏi 056, 057, 059, 061, 062, 063; ca 058 (chữ trông giống) đổi sang "không so khớp mờ"; ca 060 chỉ còn "bí danh"; ca 157 là ca "mã nguồn khác" |
| S4-11 tỉ lệ `category` | **giữ** | Tỉ lệ mục tiêu của CONVENTIONS là cho bộ ca mỗi tenant, không áp cho bộ giữ lại theo lớp. Sau sửa: happy_path 14, dirty_data 36, ambiguity 4, conflict 13, permission 34, infrastructure 32, adversarial 24 |
| S4-12 16 tên `metrics` tự đặt | **giữ** | Ca mới dùng lại tên có sẵn, không thêm tên mới |
| S4-13 nửa sau của A14 | **giữ** | Không dựng được khi chưa biết tham số của pack; ba ca biên còn lại đã gắn nhãn |
| S4-14 ngày 18/10 mới ở `world-p1.yaml` | **sửa chữ** | Thêm ghi chú ở đầu `world-p1.yaml`: tệp chỉ dùng cho bộ giữ lại; ca WIN gốc vẫn chạy trên `world.yaml`; tổng cấp dự án ngày 18 và 19/10 khác nhau |
| S4-15 tệp tạm của người soát | **ghi nhận** | Không động tới; người soạn có tệp tạm riêng (mục 11.6) |
| Đột biến sống sót (mục 7) | **xử lý hoặc giữ** | (1) phát lại trước kiểm quyền: giữ (S2-02); (2) khoá không phân biệt hoa thường: giữ, hợp đồng im lặng; (3) kiểm quyền trước kiểm tenant: sửa (S2-03) |

### 11.4 Tự kiểm sau sửa

Mọi con số là kết quả lệnh vừa chạy trên hai tệp có băm ở 11.2. Mô hình của người soạn (`hp1r-model.py`) viết lại từ hợp đồng và `world-p1.yaml`, không xem tệp tạm của người soát; vẫn dùng cùng hợp đồng nên không phải bằng chứng hợp đồng được hiểu đúng.

| # | Việc | Kết quả |
|---|---|---|
| 1 | Khuôn: 23 khoá đúng thứ tự §5, `entry_type` COMMAND, `expected_route` TOOL, `expected_tools`, `expected_fallback` | 157/157 ca đúng |
| 2 | Hình dạng §12.1: khoá của `input`, `expected_state`, `expected_output_properties`; `parallel` bằng số lời gọi sau khi mở `repeat`; tổng `results` bằng số kết quả nhận được (trừ lời gọi bị F4 bỏ); `caller` có trong ca; mọi `reason_code` thuộc §2.4 | 0 lỗi |
| 3 | Kiểu YAML | 0 scalar không nháy bị hiểu thành ngày, số thực, bool hay null lạ; 0 khoá trùng; hai bộ nạp của PyYAML (C và thuần) cho cùng kết quả |
| 4 | Ký tự | 0 byte điều khiển thô, 0 CRLF, 0 tab, 0 BOM; 0 ký tự nhóm Cf, Co, Cn; 4 chuỗi thoát NUL trong chuỗi có nháy (giữ nguyên) |
| 5 | `id` | `CON-T-701` đến `CON-T-857` liên tục, không trùng; ca cũ giữ thứ tự |
| 6 | Người gọi | 205 lượt đối chiếu `id`, vai, phạm vi với `principals` của `world-p1.yaml` (đúng tenant của người gọi): 0 lệch |
| 7 | Máy, ngày | 285 lời gọi dùng máy có trong danh mục của runtime: 265 có bản ghi ở ngày đó; 20 không có (11 ngày sai dạng, rỗng hoặc có NUL; 5 ngày không phải ngày đã qua; 4 ngày đã qua hợp lệ nhưng máy thiếu vế, đúng các ca thiếu bằng chứng); 15 lời gọi dùng khoá không có trong danh mục (chủ đích); 2 lời gọi dùng khoá dự án (chủ đích) |
| 8 | `preconditions` | 182 dòng `world-p1:` khớp số trong bản ghi (0 lệch) |
| 9 | Đếm theo lớp | như 11.2; 42 ca có chèn lỗi (F1 4, F2 4, F3 4, F4 5, F5 3, F6 `readback_fail` 3, F6 `readback_stale` 2, F7 14, F8 3), 115 ca không |
| 10 | `world-p1.yaml` so với `world.yaml` gốc | 0 dòng gốc bị sửa hoặc xoá; 225 dòng thêm; id, `ref`, `source_ids` không trùng |
| 11 | Mô hình tham chiếu | 157/157 ca khớp; 625 lượt chạy kể cả mọi thứ tự commit ở 14 ca đua; không ca đua nào phụ thuộc thứ tự đến |
| 12 | Độ nhạy của bộ so | 1909 lần sửa giá trị kỳ vọng, 1898 bị mô hình bắt, 11 lọt: cả 11 đều là `cross_tenant_rows` (bộ đếm của bộ chạy mà mô hình không dựng) |
| 13 | Phụ thuộc Q-P1-7 | đảo C4b sau R5: đúng 24 ca đổi, trùng 24 ca đã gắn nhãn |
| 14 | Phụ thuộc `min_abs_diff` | xem S3-01 |
| 15 | So tệp cũ và mới theo từng ca | 63 ca cũ có đổi (khoá đổi: `input` ở 013, 014, 020, 060; `expected_output_properties` ở 005, 011, 060, 142; `expected_state` chỉ ở 060; còn lại chỉ `preconditions`, `must_not_do`, `business_rules`, `acceptance_criteria`); 0 ca mất; 3 ca mới |

### 11.5 Việc còn lại cho người điều phối (không thuộc quyền người soạn)

1. Chốt mẫu `id` (S2-01) trước khi băm.
2. Chủ bộ sinh xác nhận `R-P013` và `R-P025` (phiếu kho, xlsx, `raw` có "≈", `approx`) được render và nạp giữ cờ; chưa xác nhận thì bỏ bốn ca 028, 119, 120, 121 (S3-02).
3. Chủ hợp đồng xác nhận cách đọc §2.1 cho mã nguồn khác (ca 157); không xác nhận thì bỏ ca 157 (S3-03).
4. Chủ dự án trả lời câu Q-P1-7: 24 ca đã gắn nhãn phụ thuộc (S4-01).
5. Xác nhận `min_abs_diff` của pack bằng mặc định 0 (ba ca biên 098, 099, 100).
6. Thêm vào hợp đồng: một câu ở §6.4 nói dòng audit của bên thua đua (14 ca đua phụ thuộc cách suy từ §3 và §7.3); một câu ở §12.1 nói bảng phiếu, sổ lệnh, audit làm rỗng trước MỖI ca; một câu cho F5 nói việc thả COMMIT có được chờ xong không; một câu cho A23 nói "không bao giờ DONE" chỉ áp cho lượt hỏng đọc lại.
7. Tính lại băm hai tệp; chạy lại soát độc lập trên bản mới.

### 11.6 Tệp tạm của người soạn (chứa phép suy kỳ vọng; không đưa cho người viết mã; xoá sau khi đóng băng)

Thư mục: `$S/`

- `hp1r-dump.py`, `hp1r-dump-v0.txt` (bản in gọn các ca trước sửa), `hp1r-lines-v0.txt`
- `hp1r-model.py` (mô hình tham chiếu, có cờ `--swap`, `--thr=N`, `--mut-*`), `hp1r-mut.py` (độ nhạy)
- `hp1r-check.py` (tự kiểm hình dạng và đối chiếu world), `hp1r-diff.py`, `hp1r-diff-out.txt`
- `hp1r-apply.py` (áp lượt sửa từ các bản sao gốc), `hp1r-md.py` (ghi mục này)
- `hp1r-orig-p1.yaml`, `hp1r-orig-world-p1.yaml`, `hp1r-orig-soat-ca.md` (bản sao nguyên trạng trước lượt sửa)

Soát lại sau khi sửa: chạy lại `hp1s-model.py`, `hp1s-shape.py`, `hp1s-mutants.py` của người soát trên tệp mới, rồi tính lại dấu vân tay.
