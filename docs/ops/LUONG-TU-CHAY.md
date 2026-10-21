# Luồng tự chạy (AUTORUN) — cách dựng, luật chạy, cách dừng

Tài liệu này mô tả luồng mà một phiên Claude Code tự chạy để làm WIN MVP mà không hỏi người. Dùng nó để dựng lại luồng trên máy hoặc server khác.

Nhật ký của lần chạy đầu (20/10/2026, dừng 10:33 theo lệnh chủ dự án): `docs/ops/NHAT-KY-2026-10-20.md`.

## 1. Thành phần

| Thành phần | Ở đâu | Vai trò |
|---|---|---|
| Hợp đồng vận hành | `CLAUDE.md` (gốc repo, có trong git) | luật bắt buộc của mọi phiên |
| Tệp điều khiển | `AUTORUN.local.md` ở gốc repo — **chỉ ở máy** (`*.local.md` bị gitignore) | cơ chế, luật mỗi lượt, quyết định tự chốt, hàng đợi, nhật ký giờ |
| Bàn giao | `HANDOFF.local.md` (chỉ ở máy); bản công khai `docs/ops/README.md` | prompt mở phiên, chỗ lưu mọi thứ |
| Lịch | `CronCreate` trong chính phiên Claude Code | 3 lịch: nhịp lặp, hạ nhịp, chốt sổ |
| Trạng thái công khai | `docs/STATUS.md`, `CHANGELOG.md` | tiến độ theo 7 cổng, lệnh chứng minh |

Lịch của `CronCreate` **chỉ sống trong phiên đang mở**: đóng cửa sổ là mất lịch, phải dựng lại.

Lịch chỉ nổ khi phiên đang rảnh; phiên đang bận thì nhịp chờ tới lúc rảnh. Một lượt chạm giới hạn phiên 5 giờ thì hỏng, nhịp sau tự thử lại.

## 2. Luật mỗi lượt (không hỏi người)

1. Kiểm giờ bằng `date`. Quá giờ dừng thì chốt sổ (mục 4) rồi dừng hẳn.
2. Thu hoạch agent nền đã xong: **đo lại lời khai bằng lệnh**, không chép số của agent.
3. Có việc `ĐANG` thì làm tiếp; không thì lấy việc `CHỜ` kế tiếp theo thứ tự hàng đợi.
4. Mỗi thay đổi mã đi đúng bốn bước:
   - test đỏ trước;
   - sửa đúng tầng;
   - `go vet ./...` + `gofmt -l .` + `go test ./...`;
   - đột biến bằng `scripts/mutate.sh` để chứng minh test bắt được.
5. Không commit / push / merge khi chủ dự án chưa cho. Không đụng repo khác.
6. Agent con:
   - chỉ model nhỏ (sonnet / haiku), tối đa 3 cùng lúc;
   - danh sách tệp không giao nhau, mỗi agent một thư mục scratch riêng;
   - người soát độc lập dùng loại agent không có công cụ sửa tệp.
7. Câu cần người quyết mà đảo lại được thì tự chốt theo khuyến nghị, ghi vào bảng "Quyết định tự chốt" + ADR. Việc không đảo được (xoá dữ liệu, chi tiền, đụng hệ người khác) thì để `CHẶN` và báo một lần.
8. Cuối mỗi việc: cập nhật hàng đợi + `docs/STATUS.md` (tiến độ theo 7 cổng) + một dòng nhật ký.
9. Làm tối đa khoảng 40 phút mỗi lượt. Kết quả ghi ra tệp trước khi kết thúc lượt, vì ngữ cảnh có thể bị nén.

## 3. Prompt của ba lịch

Thay `<NGÀY>`, `<THÁNG>` cho ngày chạy. Đường dẫn tính từ gốc repo.

**Nhịp lặp** — `CronCreate(cron: "3,13,23,33,43,53 8-17 <NGÀY> <THÁNG> *", recurring: true)`:

```text
[EACL AUTORUN — nhịp tự động; chủ dự án đã cho phép tự chạy KHÔNG HỎI tới 18:00 ngày <NGÀY>/<THÁNG>] Làm theo AUTORUN.local.md ở gốc repo EACL: (1) kiểm giờ bằng `date`; nếu ≥ 18:00 thì chốt sổ theo tệp đó rồi CronDelete mọi lịch EACL và dừng. (2) Thu hoạch agent nền đã xong (đo lại lời khai). (3) Nếu có việc ĐANG thì làm tiếp; không thì lấy việc CHỜ kế tiếp theo thứ tự. (4) Mỗi thay đổi mã: test đỏ trước → sửa → vet/gofmt/test → đột biến bằng scripts/mutate.sh. (5) Làm tối đa ~40 phút rồi cập nhật hàng đợi + docs/STATUS.md + nhật ký và kết thúc lượt. Không hỏi chủ dự án, không commit/push, không đụng repo khác.
```

**Hạ nhịp** — `CronCreate(cron: "41 17 <NGÀY> <THÁNG> *", recurring: false)`. Bản gốc bị cắt khi liệt kê lịch; dưới đây là bản dựng lại:

```text
[EACL AUTORUN — 17:40 BẮT ĐẦU HẠ NHỊP] Không mở việc lớn mới. Làm nốt việc đang dở tới một điểm dừng an toàn (test xanh, không để tệp dở); ghi trạng thái vào hàng đợi + nhật ký.
```

**Chốt sổ** — `CronCreate(cron: "2 18 <NGÀY> <THÁNG> *", recurring: false)`. Bản dựng lại:

```text
[EACL AUTORUN — 18:00 CHỐT SỔ] Dừng mọi việc EACL. Chạy kiểm cuối (go vet ./..., gofmt -l ., go test ./... -count=1, scripts/scale-smoke.sh). Cập nhật docs/STATUS.md, CHANGELOG.md, HANDOFF.local.md (kèm prompt mở phiên sau), nhật ký trong AUTORUN.local.md và memory EACL. Dừng agent nền. CronDelete mọi lịch EACL. Báo kết quả theo mẫu Changed / Verified / Not verified kèm MỘT con số tiến độ %.
```

## 4. Dừng luồng

Chủ dự án nói dừng, hoặc tới giờ chốt sổ, thì làm theo thứ tự:

1. Xoá mọi lịch (`CronDelete` từng id; `CronList` để xem).
2. Dừng agent nền (`TaskStop`). Việc của agent đó trong hàng đợi về lại `CHỜ`, ghi agent đã làm tới đâu.
3. Đo cuối: vet / gofmt / test / scale smoke. Ghi số vào STATUS và nhật ký.
4. Cập nhật STATUS, CHANGELOG, HANDOFF, memory.
5. Chỉ commit / push khi chủ dự án yêu cầu. Repo này PUBLIC: xem mục "Không đưa lên git" ở `docs/ops/README.md`.

## 5. Dựng lại luồng trên máy khác

1. Lấy mã và chạy được mốc: `docs/ops/README.md` mục 2–3.
2. Tạo `AUTORUN.local.md` ở gốc repo theo mẫu ở mục 6 (tệp này không có trong git).
3. Mở phiên Claude Code ở gốc repo rồi dán:

```text
Đọc CLAUDE.md, docs/ops/README.md, docs/ops/LUONG-TU-CHAY.md, docs/STATUS.md. Chạy lại mốc theo docs/ops/README.md mục 3 và so với docs/STATUS.md; lệch thì báo tôi trước. Rồi dựng luồng tự chạy theo docs/ops/LUONG-TU-CHAY.md: tạo AUTORUN.local.md từ mẫu mục 6 (hàng đợi lấy từ "Việc tiếp theo" của docs/STATUS.md), dựng 3 lịch bằng CronCreate với prompt ở mục 3 cho ngày hôm nay, rồi chạy nhịp đầu ngay.
```

## 6. Mẫu `AUTORUN.local.md`

```markdown
# AUTORUN — EACL, tự chạy <ngày> tới <giờ dừng>

> CHỈ Ở MÁY (gitignore `*.local.md`). Lệnh của chủ dự án: "<nguyên văn>".

## Cơ chế
- Nhịp: CronCreate trong chính phiên này, khoảng mỗi 10 phút. Mã lịch: <id nhịp> · <id hạ nhịp> · <id chốt sổ>.
- Một bên ghi duy nhất: phiên này. Phiên khác trên cùng cây chỉ đọc.

## Luật mỗi lượt
(chép mục 2 của docs/ops/LUONG-TU-CHAY.md)

## Quyết định tự chốt (chủ dự án vắng, ĐẢO ĐƯỢC)
| # | Câu | Chốt | Lý do |
|---|---|---|---|

## Hàng đợi
Trạng thái: `CHỜ` · `ĐANG (giờ)` · `XONG (giờ)` · `CHẶN (lý do)`.
1. `CHỜ` …

## Nhật ký
- <giờ>: …
```

## 7. Bài học của lần chạy 20/10/2026

- **Đo lại lời khai của agent bằng đột biến của người khác.** Agent khai "44 đột biến đều bị bắt"; một đột biến ở biên luật D5 (20 → 19 điểm) do người khác chọn vẫn sống sót.
- **Test xanh ngay lần đầu chưa phải bằng chứng.** Test tất định của truy xuất xanh trên mã lỗi vì dữ liệu test cho mọi từ cùng IDF. Phải làm nó đỏ trên mã cũ trước.
- **Phát lại (G6) là cổng bắt lỗi thật, không phải nghi lễ.** Nó bắt được thứ tự bằng chứng không tất định mà 115 ca không bắt được.
- **Agent soát chỉ đọc** dùng loại agent không có công cụ sửa tệp, để nó không thể tự sửa rồi tự khen.
- **Cổng 100% trên bộ ca cố định chưa chứng minh tổng quát hoá.** Cần bộ ca đội đỏ mới, do người không đọc mã soạn, đặt ngoài bộ đóng băng. Việc này đang dở (hàng đợi việc 13).
