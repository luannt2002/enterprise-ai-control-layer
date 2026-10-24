# Cách làm của phiên RAG — luồng, cổng, bài học đã trả giá (cho BUILD đọc)

> **Viết cho:** phiên BUILD và người soát của team.
>
> **Mục đích:** biết phiên RAG đã làm việc thế nào để đọc, chạy lại, tiếp tục được. Tài liệu chỉ mô tả cách làm, không thay luật ở
> `CLAUDE.md` / `docs/CHO-QUYET-DINH.md`.
>
> **Cập nhật:** 24/10/2026. Lệnh dùng `scripts/cong/` (bản chạy được trên máy bất kỳ có Go theo `go.mod` và PG theo
> `deploy/docker-compose.yml`).

## 1. Luồng một việc (đã chạy suốt 21–24/10)

```text
việc (CHO / ROADMAP-RAG §10.2) → đọc mã + tài liệu liên quan → tái hiện bằng test ĐỎ (hoặc phép đo trên bộ phát triển)
→ sửa nhỏ nhất → đột biến chứng minh test cắn → cổng (go test có PG, so theo TÊN + winreport + smoke)
→ soát độc lập CHỈ ĐỌC → sửa phát hiện P1 → commit chẻ nhỏ → push nhánh → (chủ dự án quyết) gộp main
```

- **Bộ đo, không phải cổng, khi đo chất lượng.** Bộ phát triển để chọn (ngưỡng, cách cắt, nhãn). Bộ giữ lại thì:
  - do agent khác soạn, mù với mã;
  - băm TRƯỚC khi chạy;
  - ngưỡng và quy ước chấm khai trước ở sổ đăng ký;
  - chạy đúng MỘT lần mỗi phương án;
  - bộ chấm sai thì CHẤM LẠI từ kết quả thô đã ghi, không chạy lại để tuyên bố.
- **Bộ phát triển đánh lừa — có số.** 24/10, phương án "chấm đoạn thay xếp lại" thắng ở bộ phát triển (30 → 32) nhưng thua ở bộ
  giữ lại (33 → 27).

## 2. Cổng (lệnh)

```bash
scripts/cong/cong.sh <cây> <ra.json> [database] [gói]        # go test -json ./... có PG, không nạp .env
python3 scripts/cong/so_ten.py <mốc.json> <ra.json>           # mất / đổi / mới theo TÊN; rỗng không phải xanh
scripts/cong/win-so.sh <cây cũ> <cây mới> <thư mục ra>        # 117 / H1 / H3 / H4 × bộ nhớ / PG: số dòng khác
scripts/chay-thu.sh                                            # smoke đầu–cuối từ clone sạch
python3 scripts/cong/quet_nguong.py <kết quả> <lượt>…          # quét ngưỡng "biết im" trên lượt chạy ngưỡng ~0
python3 scripts/cong/cham_nguong.py <kết quả> <lượt>=<t>[,<t>…] # chấm lại một lượt ở ngưỡng cố định (không gọi lại động cơ)
```

## 3. Luật làm việc đã trả giá (mỗi luật kèm sự cố)

| Luật | Vì sao (sự cố thật) |
|---|---|
| Số đưa vào tài liệu phải từ lệnh vừa chạy; chưa chạy ghi "chưa đo" | nhiều lần một con số "nhớ" lệch số đo lại |
| So test theo TÊN, kiểm số tên > 0 | đếm tổng vẫn xanh khi một test bị xoá; tệp kết quả rỗng trông như "0 fail" |
| Lệnh chạy nền dùng đường dẫn tuyệt đối; chờ đúng dấu kết thúc của CHÍNH lượt đó | một lần chờ trúng dấu "rc=" của lượt cũ trong cùng tệp ⇒ chấm một lượt mới chạy 7 câu (bắt kịp, không công bố) |
| Mỗi chốt mới phải có đột biến bị test bắt; gỡ đột biến bằng `cp` bản sao, CẤM `git checkout -- <tệp>` khi cây có sửa chưa commit | `checkout` từng xoá một bản sửa chưa commit; nhiều test "xanh" mà bỏ chốt vẫn xanh (24/10: 2 / 7 chốt sống sót lần đầu) |
| Soát độc lập: agent riêng, CHỈ ĐỌC, phải chạy được lệnh | mọi vòng soát đều bắt lỗi thật. 24/10: B1 đẩy câu số liệu không có "bao nhiêu" sang tài liệu (9/9 câu tái lập); bộ chấm "trích đúng" chỉ so tài liệu, không so đoạn |
| Vế an toàn của bộ đo phải có ca thật sự thử nó | "0 câu số liệu tới tri thức" trên bộ có 11/12 câu chứa "bao nhiêu" gần như không chứng minh gì |
| Bên thứ ba: báo chủ dự án TRƯỚC (dùng cho gì, bằng chứng, API hay máy, dữ liệu ra ngoài); tên hãng / endpoint / model / khoá chỉ ở `.env`, mã chỉ biết GIAO THỨC | repo công khai; bậc miễn phí của một số dịch vụ dùng dữ liệu gửi lên để huấn luyện ⇒ chỉ gửi dữ liệu giả lập / công khai |
| Dữ liệu khách chỉ chạy với động cơ trên máy | — |
| Trước mỗi lần push: quét tên riêng, đường dẫn máy, bí mật trong diff | repo công khai |
| Worktree: `.env` nối mềm (symlink), không sao chép; tệp nháp chưa theo dõi trong `docs/` dời ra khi chạy test gốc | thiếu `.env` thì test vẫn chạy mà ra số sai; test gốc quét `docs/` |
| Máy dùng chung: tối đa 3 tiến trình Go build / test cùng lúc, `GOMAXPROCS=2` | quá tải RAM từng giết tiến trình khác |
| Số migration: ai vào main trước giữ số; mã tìm migration theo mẫu tên, không theo số | RAG đổi 0005 → 0007 / 0008; Phase 0a đổi 0006 → 0009 khi gộp 24/10 |
| Gộp nhánh: gộp khô (`git merge-tree`) vào main TRƯỚC và SAU thay đổi để tách xung đột có sẵn khỏi xung đột mới; nhánh đã vào main / bị bản sau thay thì ghi nhận `git merge -s ours` (cây không đổi), không gộp nội dung cũ | gộp 3 chiều nội dung cũ chèn lại dòng vòng soát sau đã bỏ |

## 4. Giao nhau đã gặp khi gộp mọi nhánh (24/10)

- Vòng sửa S4 thêm "mấy" vào `quantity_markers`. Chốt câu hỏi lượng của dự phòng S4 dùng chung danh sách đó, nên câu quy trình
  "trong mấy tiếng" không được dự phòng cứu nữa. Chọn phía an toàn (câu số liệu không bao giờ bị trả lời bằng tài liệu).
- Kỳ vọng test của nhánh S4 (viết trước lớp che C9) nêu tên thực thể ngoài phạm vi người hỏi; mã gộp che đúng ⇒ sửa kỳ vọng theo bất
  biến che.
- Danh sách `Notes` của vết: RAG thêm `Execute`, M1 thêm `Generate` / `Respond` ⇒ hợp đủ.
- Chi tiết từng khối xung đột: `docs/debug-handoff/2026-10-24-01-giao-nhau-rag-voi-nhanh-team.md`.

## 5. Routine chạy không người trực (cách đã dùng)

- Một tệp kế hoạch tự chứa: bối cảnh đã xác minh, hàng đợi có thứ tự, luật, lệnh cổng, định dạng báo cáo.
- Dấu khởi động chống chạy trùng hai lượt.
- Lịch trong phiên, kèm một dòng cron dự phòng chạy CLI với danh sách công cụ TRẮNG (không push / merge / xoá). Dòng cron tự gỡ khi
  lượt bắt đầu.
- Hết việc trong hàng đợi thì báo; việc cần chủ dự án quyết thì ghi vào báo cáo rồi sang việc kế, không dừng hỏi giữa chừng.
- Báo cáo cuối: đã làm (có lệnh), chưa làm, cần quyết, số đo kèm lệnh tái lập.
