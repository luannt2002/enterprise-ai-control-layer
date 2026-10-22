# 09 — Evidence Index

> **Khung (D-RM-05, 22/10/2026):** tệp để tra cứu — roadmap hiện hành là `docs/ROADMAP-DAY-DU.md` (D-RM-01). Tên sản phẩm trong thân tệp là tên lúc viết; tên hiện hành là **EAIS** (D-RM-02).

> Mỗi khẳng định PASS hoặc mỗi số trong bộ `docs/v3/` trỏ về một dòng ở đây.
> Khuôn bản ghi theo Playbook §12, rút gọn thành cột.
> Không có dòng ở đây thì khẳng định đó là **E0 CLAIM**.

## 1. Bằng chứng của lượt lập kế hoạch (20/10/2026)

| ID | Khẳng định | Lệnh / cách tái lập | Kết quả | Revision · môi trường | Mức |
|---|---|---|---|---|---|
| EV-R0-01 | Nguồn V3 được chụp nguyên vẹn | `sha256sum -c docs/sources/eacl-v3/SHA256SUMS` (thư mục ignore, chỉ ở máy) | bản hội thoại `65aabdbc…` (36.742 dòng); Blueprint gốc `fa5de366…` (15.132 dòng); Playbook gốc `b2fe3a1a…` (1.691 dòng) | ngoài git | E1 |
| EV-P-01 | Catalog Blueprint §42–46 là khuôn nhân bản | script Python tách mục theo tiêu đề, gom thân giống nhau (chạy trên bản Blueprint gốc) | §42: 120 mục / 20 thân khác nhau (10 lớp lỗi); §43: 120 / 12 nhóm, cổng gán xoay vòng 20 mỗi cổng; §44: 60 / 12; §45: 70 / 7; §46: 150 / 1 thân | — | E4 (đo trên tệp) |
| EV-P-02 | Kiểm kê mã | `go list ./...` + đếm dòng tệp `.go` từng gói | 26 gói; 10.613 dòng mã không kể test; 5.190 dòng test; 127 hàm `Test` | `85108c4` | E4 |
| EV-P-03 | Năng lực hành động chưa có | `grep -rIl --include='*.go' -e idempot -e Proposal -e approv -e Audit -e Customer .` | 0 tệp cho mỗi từ khoá; đã đọc thêm danh sách kiểu của `core/contracts` để xác nhận | `85108c4` | E1 (vắng mặt, không chỉ dựa vào `grep`) |
| EV-P-04 | Playbook không chứa tên riêng; Blueprint có | quét theo `docs/sources/eacl-v3/ten-rieng.txt` trên hai tệp gốc | Playbook: 0; Blueprint: §1.3–1.4, §45, §51 | — | E4 |
| EV-P-05 | `docs/v3/` không lộ tên riêng | `grep -rn -i -E -f docs/sources/eacl-v3/ten-rieng.txt docs/v3/` | 0 khớp, chạy lại sau khi sửa AC của R0-04 | cây làm việc | E4 |
| EV-B01…B07 | Mốc trước V3 | `02` §2 | vet sạch; gofmt rỗng; **224/0/0**; scale smoke ĐẠT; winreport 117/117 trên bộ nhớ + PG; phát lại 0 lệch; băm manifest `31172355…`, bộ ca `ffdd704b…` | `85108c4`, PG 17.11, 20/10 15:22 | E4 |
| EV-OBS-01 | Ghi chú xung đột sai phạm vi ở câu tổng | đầu ra `winreport -v` ca WIN-I-115 + `grep -n conflict_note tenants/construction/pack/templates.yaml` | thấy trên cả hai đường; ca vẫn đạt | `85108c4` | E4 |

## 2. Bằng chứng cũ được thừa kế (chưa đo lại trong lượt này, ghi để truy)

| Khẳng định | Nguồn | Ghi chú |
|---|---|---|
| ~90 đột biến, mọi đột biến sống sót đều sinh test | `docs/STATUS.md` (20/10) | lượt này chỉ chạy lại test, chưa chạy lại đột biến |
| 6 lỗi thật bắt được ngày 20/10 | `docs/STATUS.md` | — |
| Clone HTTPS sạch build được, 224/0/0 | bàn giao 20/10 | chưa đo lại |

## 3. Nhật ký giao việc (lead ghi trước mỗi lần giao)

| Giờ | Task | Model | Lý do | Kết quả |
|---|---|---|---|---|
| 20/10 | đọc tóm tắt phần lịch sử V1→V2.1 (dòng 1–7.293; hỗ trợ lập kế hoạch, không phải task V3) | Sonnet | đọc khối lượng lớn, lead kiểm lại | xong: 7.293/7.293 dòng, 8 câu trả lời; lead kiểm 4 khẳng định (dòng 2.221, 3.290, 6.447, 7.177–7.231), khớp |
| 20/10 | đọc tóm tắt tài liệu học: study guide + workshop (dòng 9.373–14.280) | Sonnet | như trên | xong: 4.908/4.908 dòng, 5 câu trả lời; lead kiểm 3 khẳng định (dòng 10.798, 12.267, 14.079), khớp |
| 20/10 | đọc tóm tắt phần RAG → Problem Map (dòng 14.281–19.797) | Sonnet | như trên | xong: 5.517/5.517 dòng, 6 câu trả lời; lead kiểm 2 khẳng định (dòng 16.769, 18.777), khớp |

## 4. Chỗ ghi bằng chứng từ R0 trở đi

Khuôn một dòng: `EV-<phase>-<nn> | khẳng định | lệnh | kết quả nguyên văn | revision + môi trường + bộ dữ liệu/bộ ca (băm) | provider/model + chi phí nếu có | mức E0–E6 | giới hạn`.
