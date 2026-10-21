# PROMPT — Phiên BUILD EACL: xây tiếp theo roadmap và MVP, nhận việc và trả việc qua git

> **Gửi:** phiên BUILD (team agent trên server, kể cả các agent con / workflow do phiên đó điều phối).
> **Từ:** chủ dự án (Luân), qua phiên điều phối + debug. **Ngày:** 21/10/2026.
> **Loại:** brief đứng. Đọc lại tệp này ở **đầu mỗi ca / mỗi lượt mới**, cùng `CLAUDE.md`, `docs/CHO-QUYET-DINH.md`, hợp đồng vận hành ca đang hiệu lực, và `docs/ROADMAP.md` §4.5, §4.7.
> **Thứ bậc:** `CHO-QUYET-DINH.md` > ADR > hợp đồng (`docs/04`, `HOP-DONG-G7.md`, hợp đồng vận hành ca) > `CLAUDE.md` > tệp này > `ROADMAP.md`. Tệp này mâu thuẫn với nguồn cao hơn thì nguồn cao hơn thắng; ghi mâu thuẫn vào báo cáo ca.

---

## 1. Hai phiên, hai vai

| Phiên | Ở đâu | Làm gì | Không làm |
|---|---|---|---|
| **BUILD** (phiên nhận tệp này) | server | Code, test, soát độc lập, commit, push `main`. Tự điều phối multi-agent / workflow của mình theo §4 | Tự quyết việc loại 2 / 3 (§3). Ghi cột "Trả lời" của `CHO` khi chủ dự án chưa nói câu trả lời |
| **ĐIỀU PHỐI + DEBUG** (phiên học của chủ dự án) | máy chủ dự án | Ghim một mốc, chạy, trace, giải thích cho chủ dự án. Viết prompt và DEBUG HANDOFF. Ghi đề xuất vào `ROADMAP.md` | Sửa code; commit code. Chỉ commit tài liệu trong `docs/` khi chủ dự án bảo |

Vì sao tách: chủ dự án đang học để **sở hữu** EACL. Phiên debug cần quan sát hành vi hiện có mà không bị sửa dưới chân; phiên BUILD cần xây mà không bị kéo vào giải thích. Git là kênh chung: có lịch sử, đọc lại được, không mất ngữ cảnh khi phiên đổi.

## 2. Kênh giao việc qua git

**Vào BUILD** (đọc mỗi lần pull):

| Nơi | Là gì | BUILD làm gì |
|---|---|---|
| `docs/CHO-QUYET-DINH.md` | quyết định của chủ dự án; **nguồn duy nhất** cho quyết định | thi hành |
| `docs/PROMPT-*.md` | yêu cầu có khuôn trả lời | làm, rồi commit tệp phản hồi đúng khuôn |
| `docs/debug-handoff/*.md` | lỗi do phiên debug tìm ra, đã **dừng ở chẩn đoán** | điều tra lại, quyết cách sửa theo `CLAUDE.md`, ghi trạng thái (dưới) |
| `docs/ROADMAP.md` §4.5, §4.7 | **đề xuất**, không phải lệnh | dùng làm danh sách việc; bất đồng thì ghi "Phản hồi team" |

**Ra từ BUILD:**
- Commit nhỏ, tách theo việc. Báo cáo ca ở `docs/bao-cao/`.
- Với **mỗi** tệp handoff: điền mục "Trạng thái BUILD" ở cuối tệp đó. Mỗi trạng thái kèm ngày và commit:

  | Trạng thái | Nghĩa |
  |---|---|
  | `NHẬN` | đã đọc, chưa làm |
  | `ĐANG LÀM` | đang xử lý |
  | `XONG <commit>` | đã sửa, kèm test chứng minh |
  | `BÁC: <lý do + lệnh>` | chẩn đoán sai, kèm lệnh chứng minh |
  | `BLOCKED_FOR_OWNER` | cần chủ dự án quyết; câu đã đưa vào `CHO` |

- Câu cần chủ dự án quyết: thêm vào `CHO-QUYET-DINH.md` đúng khuôn bảng, **để trống cột Trả lời**, kèm khuyến nghị và hệ quả nếu chưa quyết.
- Báo cáo ca có thêm một mục **"Trả lời handoff / prompt"**: liệt kê trạng thái của từng handoff và prompt đang mở.

## 3. Ba loại việc

| Loại | Ai quyết | Ví dụ |
|---|---|---|
| 1. Kỹ thuật, đảo được, trong phạm vi được uỷ quyền | **BUILD tự quyết**, có soát độc lập; ghi ADR nếu có hệ quả | sửa harness đo; vá lỗi có ca tái lập; điều tra S4; chọn cách cài cho một bất biến đã chốt |
| 2. Chính sách bằng chứng / sản phẩm | chủ dự án | tập nào tính là bằng chứng; STRICT / LENIENT; độ phủ của cổng phán quyết; chính sách nghiệp vụ |
| 3. Tài nguyên / quyền | chỉ chủ dự án | chi tiền; khoá model; người chấm mù |

**Luật trước / sau khi đo:** sửa **trước** benchmark mọi thứ làm **phép đo** mất giá trị (harness, bộ chấm, đối chứng dương). Hệ **đang bị đo** thì đóng băng, đo, rồi mới sửa hành vi; lỗi đã biết ghi vào báo cáo.

## 4. Hướng xây: theo đường găng hiện tại

Phase hiện tại là **R1 (Fuel Truth Benchmark)**. MVP Customer Ops (R2–R6) đã có trong roadmap, nhưng **code R2 chưa được làm** cho tới khi chủ dự án gỡ D-V3-09 (`ROADMAP.md` §4.7 #15). Thứ tự ưu tiên:

1. **Phản hồi `docs/PROMPT-AUDIT-ROADMAP.md`** cho trọn (K1–K15, Q1–Q13, bàn §4.7), theo khuôn của prompt đó.
2. **Vòng sửa S4 theo D-OP-13:** điều tra (gom ca đỏ H1, H3 theo tầng sai đầu tiên và thuộc tính thiếu) → bất biến → ca âm → so rule / model / hybrid theo cổng `CLAUDE.md` §19 → đóng băng mã S4 → chạy H4 **đúng một lần** → H4 thành hồi quy.
3. **Phép đo trước W-G7 (loại 1):**
   - harness canary: nạp hai tenant, có đối chứng dương chứng minh canary có thể đỏ;
   - kiểm vai BYPASSRLS / superuser ở `pgstore.Open`, có ca âm;
   - cờ `approx` ở W-G1, nếu team đồng ý ở Q5 của prompt audit.
4. **Bộ chấm v2 (ADR-0007):** làm phần kỹ thuật. **Đóng băng** chỉ khi đã có D-G7-15, 16, 17 và người chấm mù thật.
5. **Soát lần thứ ba** cho 3 dòng vá T2.2.
6. **Tài liệu lệch:** K13, K14 của prompt audit; các handoff trong `docs/debug-handoff/`.

**Chưa làm** (chờ chủ dự án):
- lượt W-G7 trả tiền: cần khoá, D-G7-18, và câu cho phép chi tiền cho thử khói / pilot;
- code R2: chờ D-V3-09;
- sửa `CLAUDE.md`: gộp vào đợt duyệt R0;
- sửa hành vi tầng nạp dữ liệu (`≈`, tin đính chính, tin kế hoạch) trước W-G7. Điều tra và đề xuất thì được.

## 5. Multi-agent / workflow

Cách tổ chức là việc của BUILD. Sáu luật phải giữ:
1. Người viết ≠ người soát: soát là phiên hoặc agent riêng, chỉ đọc.
2. Write-set của các agent chạy song song không giao nhau.
3. Đo trần máy (RAM, lõi, tải) trước khi chạy song song; không chạy nhiều tiến trình build / test Go hơn máy chịu được.
4. Mỗi agent nhận đủ: mục tiêu, phạm vi tệp, tiêu chí xong, bằng chứng phải nộp.
5. Không tin lời khai của agent con: phiên điều phối đo lại bằng lệnh trước khi commit.
6. Không đẻ agent nhiều hơn số việc độc lập.

## 6. Cổng trước mỗi push

- `go vet ./...` sạch; `gofmt -l .` rỗng; `go test -count=1 ./...` xanh (có PG nếu môi trường có). So test **theo tên**, không chỉ đếm tổng; phân loại test đỏ: môi trường / có sẵn / hồi quy mới.
- `TestDanhTinhChiConEACL` xanh: tệp mới không chứa tên sản phẩm cũ.
- Cổng hoặc bất biến mới phải có ca âm hoặc đột biến chứng minh nó chặn được.
- Không đọc bộ giữ lại chưa chạy; không gọi model trả tiền khi chưa được phép.
- Repo public: không đường dẫn máy, không bí mật, không dữ liệu thật.
- Push thường lên `origin/main`; không force, không viết lại lịch sử đã push.

## 7. Làm tới đâu, commit + push tới đó, cập nhật báo cáo ngay

Phiên điều phối + debug đọc tiến độ **qua git**, nên mọi tiến độ phải lên `origin/main` ngay khi có, không dồn tới cuối ca.

1. **Xong một đơn vị việc** (một bản sửa có test, một bước điều tra có kết luận, một tệp phản hồi, một handoff đã xử lý): commit ngay, qua cổng §6, rồi **push ngay**.
2. **Mỗi lần push, cập nhật cùng commit hoặc commit liền sau tệp tiến độ sống `docs/bao-cao/TIEN-DO-BUILD.md`.** Tệp chưa có thì tạo theo khuôn dưới.
   - Phần đầu ghi đè mỗi lần, nói hiện trạng ngay lúc đó.
   - Phần nhật ký chỉ thêm dòng mới, không xoá dòng cũ.

```markdown
# Tiến độ BUILD — tệp sống, cập nhật mỗi lần push

## Bây giờ (ghi đè mỗi lần)
- Cập nhật lúc: <ngày giờ> · HEAD: <commit>
- Đang làm: <việc> (theo PROMPT-BUILD-EACL.md §4 mục <n>)
- Xong từ lần trước: <việc — commit>
- Tiếp theo: <1–3 việc>
- Bị chặn: <việc — chờ ai / câu nào trong CHO>
- Test: <go test có/không PG: PASS / FAIL / SKIP — đỏ nào, loại gì>
- Handoff / prompt đang mở: <tệp — trạng thái>

## Nhật ký (thêm dòng, không xoá)
| Lúc | Commit | Việc | Bằng chứng (lệnh → kết quả) | Ghi chú |
|---|---|---|---|---|
```

3. **Cuối ca:** báo cáo ca theo khuôn hiện hành, cộng mục "Trả lời handoff / prompt" (§2). Tệp tiến độ vẫn là nơi đọc nhanh nhất.
4. **Có câu cần chủ dự án quyết:** ghi ngay vào `CHO` và vào dòng "Bị chặn" của tệp tiến độ, không đợi cuối ca.
5. **Dừng giữa chừng** (hết giờ, lỗi hạ tầng, bị huỷ): vẫn commit phần đã làm (nếu không làm đỏ cổng) và ghi đúng "DỪNG giữa chừng — lý do" vào tệp tiến độ. Không ghi "xong" khi chưa xong.

## 8. Phía bên này làm gì với kết quả của BUILD

Phiên điều phối + debug pull về, đọc `TIEN-DO-BUILD.md`, `CHO`, trạng thái handoff, rồi:
- chạy và trace lại thay đổi của BUILD ở mốc mới, cùng chủ dự án, để hiểu vì sao hệ đổi hành vi;
- thấy lỗi → handoff mới;
- việc cần xây thêm → prompt mới hoặc đề xuất ở `ROADMAP.md`.

BUILD không cần chờ phía này; cứ xây theo §4 và đẩy tiến độ lên.
