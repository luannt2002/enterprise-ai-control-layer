# Báo cáo đêm 20→21/10/2026 — khôi phục bằng chứng, xác định trạng thái WIN thật

> **BẢN NHÁP ĐANG CẬP NHẬT** (lần cuối: 20/10 20:5x). Bản chốt viết lúc 07:00 ngày 21/10.
> Mục tiêu đêm, theo prompt cố vấn mà chủ dự án giao: khôi phục bằng chứng đáng tin, rồi mới xác định trạng thái WIN thật. Không tối ưu tỉ lệ đạt. FAIL sạch hơn PASS bẩn.
> Mọi số ở đây lấy từ lệnh đã chạy; chưa đo thì ghi **CHƯA ĐO**.

## 0. Một dòng

**WIN MVP: 3/7 cổng đạt vô điều kiện (G1, G3, G4).**
- G2, G5: chưa phục hồi; chờ chứng minh lại trên bộ giữ lại độc lập H1.
- G6: không đạt tiêu chí gốc; G6' chờ H1.
- G7: CHẶN — chưa có khoá model; bộ chấm v1 bị phá, v2 đang thiết kế.

## 1. HEAD

| | Revision |
|---|---|
| START | `94bbaae` (đã push 20/10 19:26) |
| END | _(điền lúc chốt)_ |

## 2. Trạng thái cổng THẬT

| Cổng | Trạng thái | Bằng chứng |
|---|---|---|
| G1 DATA | **PASS** | tái lập 20/10 19:51: G1 2.739/2.739, metamorphic 0 lệch; kiểm toán nhiễm: không ca nhiễm |
| G2 IDENTITY | **chưa phục hồi**: bằng chứng cũ nhiễm (107, 108) | chờ H1 |
| G3 TRUTH | **PASS** | không ca nhiễm; kết luận không đổi khi loại ca không chắc (038, 046, 057, 058) |
| G4 DECISION | **PASS** | không ca nhiễm |
| G5 CONTROL | **chưa phục hồi**: phần ca có 083 nhiễm. Phần cơ chế (AST, RLS, canary B, bảng "sai đúng một chiều", chèn lệnh) giữ bằng test | chờ H1 |
| G6 PROOF | **FAIL theo tiêu chí gốc**: bộ ca được băm 12 phút sau lần chạy đầu; W1 bắt đầu trước khi bộ ca xong. G6' (ADR-0006 §8) chờ H1 | `docs/g7/CONTAMINATION-AUDIT.md` |
| G7 VALUE | **BLOCKED**:<br>- khoá model không có trong môi trường (D-G7-8);<br>- bộ chấm v2 chưa xong;<br>- pilot cần khoá | `docs/g7/HOP-DONG-G7.md` mục 13 |

## 3. Đêm nay tìm ra

### Nhiễm (`docs/g7/CONTAMINATION-AUDIT.md`)
- 117 ca: CLEAN 102 · CONTAMINATED 3 (083, 107, 108) · UNCERTAIN 12.
- Ba kỳ vọng bị phiên viết mã đổi lúc 09:09:26–35, sau lần chạy đầu (09:03:27, 76/117). Phiên chính đã đo lại 5 mốc giờ và nội dung 3 lần sửa: khớp.
- **Prompt soạn ca của phiên viết mã** có 4 diễn giải không có trong hợp đồng:
  - "nhận"; "đốt";
  - "bao nhiêu dầu" mặc định là kho xuất;
  - đưa canary đại trà vào chuỗi cấm. Đây là gốc lỗi ở ca 083.
- **ADR-0002 D7** ghi sai sự kiện về ca 107; đã thêm đính chính. `docs/04` dòng 83 trái D7, thành câu chờ quyết D-G7-12.

### Bộ chấm G7 v1 bị phá
- 170 đòn của một người tấn công độc lập: 109 câu sai mà ĐẠT; 24 câu đúng mà TRƯỢT.
- Một câu chỉ liệt kê mọi số của bản ghi ĐẠT 73/117 ca.
- Gốc: "túi số", không có phân cực, không ràng buộc thực thể / chỉ số / kỳ / nguồn, không kiểm đơn vị, không kiểm lịch ngày.

### Lỗi nhà cung cấp
- Client cũ gửi `temperature: 0`, nên mọi lời gọi tới model hiện hành sẽ lỗi 400. Đã sửa (`750307a`).

### Lỗi mã
- **Đột biến bắt được một lỗi thật:** bí danh nguồn "master" bị xếp vào "mã theo nguồn" trong đoạn B2+. Đã sửa (`a1a5b6b`).
- **Lỗ đo chưa sửa (đưa vào bộ chấm v2):** bộ đếm "số không căn cứ" chỉ cho phép số trong `world.yaml`, nên ca hỏi thiết bị nền FX sẽ bị đếm oan.

### Vận hành
- `go mod tidy` thêm 2 module ngoài kỳ vọng (chỉ cho test của thư viện yaml). Đã dừng mẻ và hoàn nguyên, chờ chủ dự án (D-OP-2).

## 4. Đã làm (mỗi dòng một commit đã push)

| Commit | Việc |
|---|---|
| `c5ce51e` | ghi quyết định chủ dự án |
| `6f7b4f4` | ADR-0005 (D5 sửa đổi, đã duyệt) + ADR-0006 (bộ giữ lại), đăng ký trước khi có kho ca |
| `750307a` | client model: không gửi sampling; thinking / effort / thử lại / ghi vết từ cấu hình; 10/10 đột biến |
| `a1a5b6b` | g7run: B2+, B1 (bản ghi thô theo cửa sổ), B2″, `-out`, `-smoke`, `gatePass`; 19/19 đột biến |
| `fdb3a18` | pass_count / k, ca chập chờn, `-pilot`; 3/3 đột biến |
| `a507e81` | `CompareG7` theo D5 sửa đổi + kiểm đầu vào (id, lớp); 10/10 đột biến |
| `a135c90` | kiểm toán nhiễm + đính chính ADR-0002 D7 + D-G7-12 |
| `a4649b8` | hợp đồng G7 viết lại (CHƯA ĐÓNG BĂNG; bảng 8 điều kiện) |
| `ed790bb` | công cụ đóng băng bộ giữ lại |
| `dfa5174` | STATUS mục đêm |
| `78d6456` | bảng cổng H1 (luật canary chặt) |
| `ce9b64d` | D-V3-10: 04 / 05 / catalog là lịch sử EAIS |
| `b132c2e` | ứng viên lõi chung (CANDIDATE_ONLY) |

## 5. Số đo

- **Test:** 224 (đầu đêm) → 248 đạt, 0 trượt, 0 bỏ qua (có PG, 20:35).
- **Đột biến:** 8 danh mục cũ 69/69 bị bắt (tái lập) + 3 danh mục mới đêm nay (client model 10, g7run + baseline 22, CompareG7 10) đều bị bắt. Cộng 111/111.
- **Scale smoke:** ĐẠT (smoke 3,45 s, RSS 693,2 MiB; RSS không có ngưỡng).
- **Gọi model thật:** 0. **Chi tiêu:** 0 USD.

## 6. Đang chạy lúc viết nháp

- Bộ giữ lại: 229 ca do người không đọc mã soạn, có `truth` đầy đủ. Đang qua 3 người soát độc lập, rồi một người sửa không đọc mã. Sau đó đóng băng + push, **rồi** mới chạy H1.
- Bộ chấm v2: hai hướng thiết kế + một giám khảo, không đọc mã EACL.
- Soát độc lập mã (D-OP-5).

## 7. Mười câu hỏi sáng nay (điền lúc chốt)

1. Biết thêm gì so với tối qua?
2. Kết luận nào được chứng minh độc lập?
3. Kết luận nào trước đây bị nói quá?
4. Cái gì hỏng?
5. Vì sao hỏng?
6. Cái gì cố ý KHÔNG làm?
7. Fuel WIN đạt chưa?
8. G7 đã chạy được bằng một lệnh chưa?
9. Pha EACL hiện tại?
10. Một việc kế tiếp giá trị nhất?
