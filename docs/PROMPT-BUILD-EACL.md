# PROMPT — Phiên BUILD: xây theo roadmap đầy đủ, nhận việc và trả việc qua git

> **Gửi:** phiên BUILD (team agent trên server, kể cả các agent con / workflow do phiên đó điều phối).
> **Từ:** chủ dự án (Luân), qua phiên điều phối + debug. **Ngày:** 21/10/2026.
> **Loại:** brief đứng. Đọc lại tệp này ở **đầu mỗi ca / mỗi lượt mới**, cùng:
> - `CLAUDE.md`;
> - `docs/CHO-QUYET-DINH.md`;
> - hợp đồng vận hành ca đang hiệu lực;
> - **`docs/ROADMAP-DAY-DU.md`**, roadmap hiện hành từ tối 21/10 (D-RM-01).
>
> **Cập nhật tối 21/10:** viết lại §2 và §4 theo D-RM-01 (roadmap phạm vi rộng nhất) và D-RM-02 (đổi tên lại, giữ link git).
>
> **Thứ bậc:** `CHO-QUYET-DINH.md` > ADR > hợp đồng (`docs/04`, `HOP-DONG-G7.md`, hợp đồng vận hành ca) > `CLAUDE.md` > `ROADMAP-DAY-DU.md` > tệp này > `ROADMAP.md` (tra cứu). Tệp này mâu thuẫn với nguồn cao hơn thì nguồn cao hơn thắng; ghi mâu thuẫn vào báo cáo ca.

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
| `docs/ROADMAP-DAY-DU.md` | **roadmap hiện hành** (D-RM-01): giai đoạn, phase, cổng, các luồng L0–L10 | làm theo thứ tự §4 của tệp này. Bất đồng: ghi vào báo cáo ca và đề xuất câu cho `CHO` |
| `docs/ROADMAP.md` | tra cứu (lỗi im lặng §0.4, bản đồ §3, phụ lục) | không lập kế hoạch theo nó |

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

## 4. Hướng xây: theo roadmap đầy đủ (D-RM-01)

Roadmap hiện hành là `docs/ROADMAP-DAY-DU.md`:
- 6 giai đoạn, Phase 0–10;
- đủ QUERY / COMMAND / WORKFLOW / EVENT;
- hai chế độ sở hữu: A tích hợp, B tự lo;
- hai cổng cho mọi việc: cổng QuânIT 9 câu (§4.1) và cổng gốc (§4.2);
- các luồng L0–L10 (§5).

Thứ tự ưu tiên:

**1. ĐỔI TÊN LẠI (D-RM-02): làm đầu tiên, trước mọi việc khác.**

Lời chủ dự án: "đổi hết tất cả tên ở code, ở doc, ở md lại thành cái tên kia; chỉ có cái link git chưa đổi thôi". "Cái tên kia" là tên gốc ngày 20/10: xem `CLAUDE.md` §9 và `docs/01-architecture.md:3` ở commit `d70ebc8`.

- **GIỮ NGUYÊN (link git):**
  - tên repo GitHub, remote;
  - đường dẫn module Go `github.com/luannt2002/enterprise-ai-control-layer` và mọi import theo đường dẫn đó.
- **ĐỔI HẾT:**
  - tên sản phẩm trong mã: comment, chuỗi, định danh, tên test (ví dụ `TestDanhTinhChiConEACL`);
  - tên sản phẩm trong mọi tài liệu và `.md` hiện hành: README, `CLAUDE.md`, `docs/` hiện hành, tên tệp có tên sản phẩm như `docs/PROMPT-BUILD-EACL.md`. Đổi tên tệp thì sửa mọi đường dẫn trỏ tới nó;
  - định danh chạy: tiền tố biến môi trường `EACL_*`, GUC RLS `eacl.tenant_id`, vai PG `eacl_app`, tên DB phụ, khoá JSON có tên sản phẩm (ví dụ `eacl_unstable_cases`).
- **Cách đổi định danh chạy:**
  - giữ bí danh tương thích cho tên hiện tại, như lần đổi trước đã giữ bí danh cho tên cũ (`intelligence/models/messages.go:409`);
  - đổi GUC và vai PG bằng **migration mới**, không sửa migration đã áp;
  - chứng minh RLS vẫn đỏ đúng chỗ bằng test có PG và `scripts/dot-bien/w4-pg-rls.sh`.
- **KHÔNG CHẠM:**
  - tệp đã băm và bằng chứng lịch sử (`legacyAllowed` trong `danh_tinh_test.go`: IMMUTABLE_HASHED_ARTIFACT, HISTORICAL_EVIDENCE);
  - bộ giữ lại H1 / H3 / H4 và cam kết băm của chúng;
  - báo cáo ca cũ;
  - các mục cũ của CHANGELOG;
  - các dòng quyết định cũ trong `CHO`.
- **Cổng danh tính: đảo chiều.**
  - Tên gốc thành tên hiện hành.
  - Tên hiện tại thành tên cũ, chỉ được xuất hiện ở tệp có lý do.
  - Giữ ca âm chứng minh cổng đỏ được (đột biến n42).
- **`CLAUDE.md` §9:** thay câu sứ mệnh bằng §1 của roadmap mới (tên gốc + hai chế độ sở hữu). Commit riêng, diff rõ, không xoá luật lịch sử còn cần (điều kiện của D-V3-06).
- **Không đổi hành vi.** Bằng chứng phải nộp:
  - `go vet`, `gofmt`;
  - `go test` có PG, so theo **tên** (0 mất, 0 đổi);
  - bộ 117 / H1 / H3 không ca nào đổi nhãn;
  - n42 và `w4-pg-rls` bắt đủ.
- **Commit tách** (gợi ý): (a) mã + định danh chạy + migration; (b) tài liệu + tên tệp; (c) cổng danh tính. Soát độc lập trước khi push.

**2. Đóng Phase 0b (Giai đoạn I).**
- Đóng R4-2…R4-4 của cổng phán quyết.
- Vòng sửa S4 theo ADR-0010 và D-OP-13: điều tra → bất biến → ca âm → so rule / model / hybrid theo `CLAUDE.md` §19.
- Đóng băng mã S4, rồi chạy H4 **đúng một lần**; sau đó H4 thành bộ hồi quy.
- W-G7 chỉ chạy khi có đủ khoá, D-G7-18 và câu cho phép chi tiền. Chưa có thì ghi "chờ".

**3. Phép đo (loại 1).**
- Harness canary nạp hai tenant, có đối chứng dương chứng minh canary đỏ được.
- Cờ `approx` ở W-G1.
- CI có PG nếu làm được trong phạm vi việc này.

**4. Phase 0a (chỉ làm giấy): việc gốc của QuânIT, "thu báo cáo ngày qua Zalo".**
- Hợp đồng việc trả lời đủ 9 câu cổng QuânIT (`ROADMAP-DAY-DU.md` §4.1).
- Bộ ca SLA giả lập, soạn độc lập.
- Nhánh so sánh "đổi sang form / bot".
- Dữ liệu thật chờ D-RM-03.

**5. Phase 1 (chỉ làm giấy): "tạo đúng một phiếu kiểm tra chênh lệch".**
- ADR + hợp đồng việc + cổng QuânIT.
- Thiết kế chống trùng, đọc lại, audit, ca âm.
- **Mã chờ D-RM-04.** Nếu D-RM-04 = A, làm theo thứ tự:
  1. lát mỏng Phase 1;
  2. Phase 2 (so Temporal / n8n **trước** khi viết);
  3. Phase 3 (sự kiện đính chính / kế hoạch / đến muộn: sửa tận gốc các lỗi im lặng của luồng L1).

**6. Phase 6 (model hiểu ý định).** Chạy H4 xong mà S4 vẫn là tầng sai đầu tiên thì chuẩn bị phép so rule / model / hybrid. Lượt gọi model thật chờ quyền chi tiền.

**7. Tài liệu lệch và handoff.**
- Phần còn lại của phản hồi `PROMPT-AUDIT-ROADMAP.md`: chỉ các khẳng định K về sự thật trong mã. Phần bàn §4.7 hết hiệu lực theo D-RM-01.
- Các handoff trong `docs/debug-handoff/`.
- Đồng bộ README / STATUS với roadmap mới (gộp vào đợt đổi tên).
- Bộ chấm v2 (ADR-0007): làm phần kỹ thuật. **Đóng băng** chỉ khi đã có D-G7-15, 16, 17 và người chấm mù thật.

**Chưa làm** (chờ chủ dự án):
- mã COMMAND / WORKFLOW / EVENT (Giai đoạn II): chờ D-RM-04;
- lượt W-G7 trả tiền: cần khoá, D-G7-18 và câu cho phép chi tiền;
- dữ liệu hoặc người nghiệp vụ thật: chờ D-RM-03;
- Phase 4 (CRM, bốn kiểu doanh nghiệp, ngành thứ ba) trở đi: sau Giai đoạn II, theo roadmap;
- sửa hành vi tầng nạp trước W-G7: vẫn chưa được. Điều tra và đề xuất thì được. Theo roadmap mới, chỗ sửa tận gốc là Phase 3.

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
- Cổng danh tính (`danh_tinh_test.go`) xanh: tệp mới không chứa tên đã bị thay. Sau đợt đổi tên D-RM-02, "tên đã bị thay" là tên hiện tại.
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
