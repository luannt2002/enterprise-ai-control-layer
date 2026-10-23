# Báo cáo soát độc lập ngày 23/10/2026 (nguyên văn)

Lệnh ca chiều 23/10 của chủ dự án: "Commit nguyên văn báo cáo soát, không chỉ tóm tắt trong TIEN-DO." Mỗi tệp `.json` là đầu ra có cấu trúc
của một người soát độc lập (agent chỉ đọc, khác tác giả), chép nguyên văn; chỉ thay đường dẫn máy vì repo công khai: `$S` là thư mục nháp
của phiên điều phối, `<repo>` là gốc repo, `<tmp>` là thư mục tạm của máy, `~` là thư mục nhà. Bằng chứng gốc mà người soát nhắc (tệp ở
`$S`) không nằm trong repo. Báo cáo soát của bộ ca giữ lại (H6, 0a…) KHÔNG ở đây: chúng chứa nội dung ca, công bố khi chạy.

| Tệp | Soát việc gì | Câu đầu của kết luận | Trạng thái |
|---|---|---|---|
| `fl2-soat.json` | flock vòng 2 (nhánh m1/r5-01-flock) | ĐẠT cho m1/r5-01-flock @ a1f8560: 0 S1, 0 S2, 0 S3, 3 S4 (đều không chặn, không cần sửa mã) | gộp c97a989 |
| `k1-soat.json` | K1 lõi lệnh vòng 1 (nhánh m4/k1-loi-lenh, d5166a5) | KHÔNG ĐẠT (nghi ngờ thì không đạt) | sửa ở vòng 2 |
| `k1v2-soat.json` | K1 vòng 2 (nhánh m4/k1-loi-lenh) | ĐẠT CÓ GHI CHÚ | gộp e1ce950 |
| `k2-soat.json` | K2 kho PG của phiếu, migration 0004 (nhánh m4/k2-kho-pg) | ĐẠT CÓ GHI CHÚ | gộp d919ebf |
| `r203v4-soat.json` | R2-03 vòng 4 — vết xuất lý do và phương án đã xét (nhánh m1/r203-vet-du) | ĐẠT (soát độc lập vòng 4, vòng ngoại lệ an ninh §9.6; nhánh m1/r203-vet-du, HEAD e731ce7, nền f344f50) | gộp 5fd463f |
| `s4a-soat.json` | vòng sửa S4 vòng 1 (nhánh m3/s4-vong-sua-2310, c7afacd) | KHÔNG ĐẠT (nghi ngờ thì KHÔNG ĐẠT; nhánh m3/s4-vong-sua-2310, đầu c7afacd, cắt từ a9ce6ed) | KHÔNG ĐẠT — vòng 2 đang chạy |
| `s5abc-soat.json` | 5a / 5b / 5c — che bí mật, phân loại lỗi PG, nền xanh của mutate.sh (nhánh m1/che-bi-mat-5a, m1/loi-pg-5b, m1/dot-bien-nen-5c) | 5a: KHÔNG ĐẠT (nhẹ, 2 mục S2 sửa nhỏ) · 5b: ĐẠT CÓ GHI CHÚ · 5c: ĐẠT CÓ GHI CHÚ | đã sửa theo soát ở m1/sua-5bc, m1/sua-5abc; chờ soát lại |
| `s2a-soat.json` | D-OP-9 (a') nhóm nghìn bằng dấu cách (nhánh m3/s2-nhom-nghin-2310, 70fec38) | ĐẠT CÓ GHI CHÚ — chưa gộp cho tới khi sửa hai câu "hiện an toàn" sai trong hồ sơ bàn giao và ghi nợ có nhãn (S2AS-01, 02, 06) | gộp 23/10 cùng các commit hồ sơ theo soát |
| `s4b-soat.json` | vòng sửa S4 vòng 2 (nhánh m3/s4-vong-sua-2310, dc84cff) | KHÔNG ĐẠT (nghi ngờ thì KHÔNG ĐẠT) | vòng 3 (vòng cuối §9.6) đang chạy; S4BS-02 thành D-AI-04 |
| `c17-soat.json` | bảy lỗi P2 C1–C7 của M4 vòng 1 (nhánh m4/sua-p2-c1-c7, 16584dd) | KHÔNG ĐẠT (2 S2: C17S-01 hồi quy vết phát lại, C17S-02 test chưa chứng minh 6/15 phép so) | sửa ở vòng 2, gộp cùng K3 sau soát lại `m4r-soat.json` — main `3892be2` |
| `k3-soat.json` | K3 route lệnh `cmd/eais` (nhánh m4/k3-route, 1a56f9b) | ĐẠT CÓ GHI CHÚ (S1 = 0; 4 ghi chú S3, 4 ghi chú S4) | K3S-01…04, K3S-07 sửa; K3S-05/06/08 ghi nợ; gộp sau soát lại `m4r-soat.json` — main `3892be2` |
| `c811-soat.json` | bốn lỗi P2 C8–C11 của R2-03 vòng 1 (nhánh m1/sua-p2-c8-c11, 1132145) | KHÔNG ĐẠT (C9-S1: vết vẫn lộ khoá ngoài phạm vi qua Decision.Entities / S3.Output / S4) | sửa vòng 2, soát vòng 2 `c812-soat.json` — gộp main `3aaaf09` |
| `m4r-soat.json` | soát lại cây gộp M4 = C1–C7 vòng 2 + K3 + sửa theo soát K3 + Ready theo cột (nhánh m4/gop-k3-c17, bc97dc9) | ĐẠT CÓ GHI CHÚ (S1 = 0, S2 = 0, S3 = 5, S4 = 4) | M4R-01…08 sửa trước khi gộp (mỗi mục một commit), M4R-09 ở dòng nợ TIEN-DO; gộp main `3892be2` |
| `s4c-soat.json` | vòng sửa S4 vòng 3 — VÒNG CUỐI §9.6 (nhánh m3/s4-vong-sua-2310, 7868598) | KHÔNG ĐẠT (S4CS-02 hồi quy S1 mới: 7 câu nền hỏi lại nay trả số không nhãn; S4CS-01 lệnh có "?") | KHOÁ, không gộp; H6 chưa chạy; câu D-AI-05 ở CHO |
| `c812-soat.json` | C8–C11 vòng 2 (nhánh m1/gop-c8-c11, 5635b31) | ĐẠT CÓ GHI CHÚ (C9-S1 đóng: 0 chỗ lộ trên 38.115 lượt của bộ quét riêng; 3 S3 + 4 S4) | C9-ID-KHAC-KHOA, C9-CHI-PHI, C9-TEST-CHUA-GHIM sửa trước khi gộp (mỗi mục một commit); ba S4 ghi giới hạn; gộp main `3aaaf09` |
| `k4-soat.json` | K4 bộ chạy ca lệnh, điểm chèn lỗi, danh mục đột biến, demo (nhánh m4/k4-bo-chay-ca, f328378) | ĐẠT CÓ GHI CHÚ (S1 = 0; 2 S2 tác giả tự khai: F8 chưa nối mã thật, M5 lệch §11.2) | K4S-01, 03, 04, 05, 07, 08, 09 (iv), 10 sửa trước khi gộp (mỗi mục một commit); K4S-02 thành Q-P1-14; K4S-06, 09 (i)(iii), 11 ghi nợ; gộp main `b22e1e6` |
| `s5r-soat.json` | soát lại các bản sửa 5a / 5b / 5c (nhánh m1/sua-5bc, m1/sua-5abc) | 5a KHÔNG ĐẠT (5a-R1, S1: Cookie sau giá trị kết thúc bằng `\` lọt) · 5b ĐẠT CÓ GHI CHÚ · 5c ĐẠT CÓ GHI CHÚ | 5b + 5c gộp main `32f4930` (5b-R1, 5b-R2, 5c-R1, 5c-R2 sửa trước khi gộp); 5a sửa vòng kế (S1 an ninh: ngoại lệ của trần 3 vòng) |
| `xx-soat.json` | bản sửa chuẩn hoá `"≈"` — tầng sai của 4 ca đỏ bộ P1 (nhánh m5/chuan-hoa-dau-xap-xi, b1eb150) | ĐẠT CÓ GHI CHÚ. Không có S1, không có S2; 1 S3 và 5 S4, không cái nào chặn gộp | XXS-01, XXS-04 sửa trước khi gộp (mỗi mục một commit); XXS-02, 03, 05, 06 ở dòng nợ TIEN-DO; gộp main `d43a1bf` |
