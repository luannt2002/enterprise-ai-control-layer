# 03 — EAIS Proof: chứng minh, không chỉ xây

> Trạng thái: quyết định hướng đi, chốt ngày 19/10/2026. Áp cho mọi tuyên bố về chất lượng của EAIS.
> Tài liệu này không mô tả code đã có, trừ mục 7 (đo ngày 19/10/2026).

## 1. Vì sao cần tài liệu này

Test xanh trên dữ liệu do chính người viết mã soạn chỉ chứng minh **logic chạy trên ví dụ của chính mình**. Lúc này đề do người làm tự ra và cũng tự chấm, đúng kiểu hỏng mà CLAUDE.md §1 muốn chặn.

EAIS không cần chứng minh "chatbot trả lời được". EAIS phải chứng minh **hệ thống biến dữ liệu doanh nghiệp lộn xộn thành câu trả lời hoặc hành động kiểm chứng được**. Chat chỉ là giao diện, không phải kiến trúc.

Có bốn ảo giác cần tránh:

| Ảo giác | Biểu hiện | Thuốc |
|---|---|---|
| Kiến trúc | đủ module ≠ giải được bài toán doanh nghiệp | một lát dọc nghiệp vụ đi thật sâu (mục 2) |
| Dữ liệu | có schema chuẩn + dữ liệu sinh ≠ hiểu đúng nghĩa nghiệp vụ | chuẩn hoá *nghĩa*, không chỉ *định dạng* (mục 5) |
| Bằng chứng | 500 test xanh ≠ sẵn sàng cho doanh nghiệp | ma trận độ phủ theo chiều hỏng (mục 4) |
| Audit | audit thêm ≠ tiến gần sản phẩm | mỗi phát hiện phải thành ca tái hiện, hoặc bị hoãn kèm điều kiện mở (mục 8) |

## 2. Sao Bắc Đẩu: một lát dọc thật sâu

```text
MỘT tenant · MỘT quy trình nghiệp vụ · dữ liệu giống thật · lỗi thật · đầu–cuối · có số đo
```

Lát dọc đầu tiên là **Đối soát nhiên liệu** (tenant cầu đường). Ta cố tình phá nó lần lượt ở các tầng:

ngữ nghĩa → thời gian → thẩm quyền nguồn → xung đột → phân quyền → giao dịch → quy mô → sự cố → phục hồi.

Nguyên thuỷ (primitive) nào sống sót qua chuỗi phá đó mới xứng vào core.

Tenant thể thao phá theo một trục khác: sự kiện thời gian thực, nhiều nhà cung cấp, bản ghi trùng, đến sai thứ tự, đính chính muộn, rút lại sự kiện, ánh xạ định danh ("Man Utd"/"MUN"), ký pháp thời gian ("45+2").

Core chịu được cả hai mà không có `if tenant == …` thì mới có bằng chứng rằng abstraction dùng lại được.

**Kiến trúc trong `01-architecture.md` là giả thuyết làm việc, không phải đáp án.** Module hoặc abstraction không chứng minh được giá trị trước lỗi thật thì bị sửa hoặc bị xoá. Capability không có lỗi nào đòi thì không bao giờ được mở.

Thứ tự luôn là: bài toán → lỗi hiện tại → nguyên nhân gốc → thuộc tính còn thiếu → capability tối thiểu → công nghệ. Không đi theo chiều công nghệ → đi tìm chỗ dùng.

## 3. Luật tuyên bố

**EAIS không tuyên bố "sẵn sàng cho doanh nghiệp" chỉ dựa trên fixture viết tay, DB giả và happy path.**

Mọi tuyên bố phải ghi rõ cấp bằng chứng. Ví dụ đúng:

- ✓ đã kiểm logic thành phần trên N ca (D0/D1);
- ✗ chưa chứng minh quy mô doanh nghiệp;
- ✗ chưa chứng minh dữ liệu giống thật;
- ✗ chưa chứng minh khả năng phục hồi.

Đúng sai và quy mô là **hai câu hỏi tách biệt**. Không dùng load test để chứng minh đúng nghĩa, cũng không dùng unit test để tuyên bố chịu tải.

## 4. Thang bằng chứng và ma trận độ phủ

### Năm cấp dữ liệu

| Cấp | Dữ liệu | Chứng minh |
|---|---|---|
| D0 | fixture viết tay | logic cơ bản |
| D1 | fixture bẩn: `4k` · `4.000 L` · `4,000L` · `4 m3` · thiếu đơn vị · thực thể lạ · trùng · xung đột · ngày sai | ca biên, đúng sai |
| D2 | dữ liệu sinh có seed: "thế giới nghiệp vụ nhất quán" rồi bơm lỗi có chủ ý | độ rộng và khối lượng |
| D3 | dữ liệu có hình dạng như production | độ giống thật |
| D4 | dữ liệu thật đã ẩn danh hoặc phát lại (chỉ khi được phép) | tin cậy production |

### Thang chống tự xác nhận

D2 dễ sinh "ảo giác dữ liệu tổng hợp": bộ sinh được viết theo giả định của chính kiến trúc nên kiến trúc dĩ nhiên qua. Vì vậy thang bằng chứng phải là:

viết tay → sinh có lỗi bẩn → **ca phá do người KHÔNG viết mã thiết kế** → mẫu giống production → mẫu thật đã ẩn danh → phát lại → shadow/canary.

### Đếm độ phủ, không đếm số ca

Câu hỏi đúng không phải "bao nhiêu ca" mà là "đã phủ những chiều hỏng nào":

- cách biểu diễn dữ liệu;
- xung đột nguồn;
- ranh giới quyền;
- chuyển trạng thái;
- bất biến nghiệp vụ;
- hồ sơ quy mô;
- đường phục hồi.

Tỉ lệ ca theo loại lối vào trong CONVENTIONS là điều kiện cần, chưa đủ. Bộ ca phải kèm ma trận chiều hỏng × ca.

### Hồ sơ quy mô

Hồ sơ quy mô mô tả cả **số lượng phân biệt** (cardinality: số tenant, thực thể, bí danh), không chỉ số dòng. 10 triệu fact của một máy khác hẳn 10 triệu fact trải trên 2 triệu thực thể.

Hồ sơ là **mục tiêu giả lập**, không phải tuyên bố năng lực. Tỉ lệ lỗi bơm vào là tham số giả lập, không phải tỉ lệ của doanh nghiệp thật.

## 5. Chuẩn hoá nghĩa, không chỉ định dạng

Đổi mọi thứ sang JSON không có nghĩa là đã chuẩn hoá. Đường chuẩn hoá dữ liệu gồm:

```text
RAW → PARSE → ÁNH XẠ SCHEMA NGUỒN → CHUẨN HOÁ (kiểu, đơn vị) → NHẬN DẠNG THỰC THỂ
    → DIỄN GIẢI NGỮ NGHĨA (quan sát này nói về fact nghiệp vụ nào) → KIỂM
    → hợp lệ: FACT/EVENT → THẨM QUYỀN NGUỒN → ĐỐI SOÁT
    → hỏng:   CÁCH LY (quarantine)
```

**Mỗi mũi tên có eval riêng**, không chỉ chấm câu trả lời cuối.

Bốn con số xuất / nhận / đo / hoá đơn có thể **đều đúng**, vì chúng là bốn nghĩa nghiệp vụ khác nhau. Không được gộp sớm thành một trường `fuel = …` rồi để thẩm quyền "chọn một".

**Thuộc tính metamorphic** là kiểu test chính của đường này: đổi cách biểu diễn mà không đổi nghĩa thì kết quả chuẩn phải tương đương. Ví dụ đúng: **cùng một phiếu xuất kho** ở dạng dòng CSV, dòng XLSX, JSON và bảng PDF (`EX003 | 4 | m3`, `Máy 03 | Dầu | 4.000 | L` …) phải ra cùng một quan sát. Kiểu test này so các biến thể với nhau, không so với đáp án tự soạn.

**Sai** (sửa 19/10, chỉnh lại 20/10 theo ADR-0001 D2): coi JSON ERP, phiếu kho và tin nhắn hiện trường là "cùng một fact". ERP và phiếu kho **cùng nghĩa ISSUED nhưng khác thẩm quyền**: so bằng thẩm quyền và lộ xung đột. Tin nhắn hiện trường là REPORTED, một nghĩa khác. Không nguồn nào được gộp thành cùng một quan sát.

## 6. Cấu trúc dự kiến

Theo CLAUDE.md §25, thư mục chỉ được tạo khi có mã thật:

```text
simulations/<tenant>/   bộ sinh có seed: thế giới nghiệp vụ, kịch bản, lỗi bẩn, xung đột, trôi schema, hồ sơ quy mô; đầu ra đóng băng thành tệp
evals/                  runner: chỉ số theo thành phần + ma trận chiều hỏng; báo cáo sinh từ lệnh
tenants/<tenant>/evals/ ca của tenant, ghi chiều hỏng mà mỗi ca phủ
```

## 7. Hiện trạng bằng chứng (đo 19/10/2026)

| Tuyên bố | Cấp | Bằng chứng |
|---|---|---|
| Hỏi số liệu nhiên liệu cầu đường đúng nghĩa trên 18 ca đầu–cuối | D0 | `go test ./tenants/construction/` |
| Bốn chỉ số của cùng máy, cùng ngày ra bốn số khác nhau, thẩm quyền chỉ so trong một chỉ số | D0 | các ca "kho xuất", "đội báo", "cảm biến", "hoá đơn" |
| Đổi cách viết câu hỏi không đổi kết quả (câu gốc + 9 cách viết khác, có gõ không dấu) | D1, metamorphic | `TestDoiCachVietKhongDoiNghia`; **bắt 2 lỗi thật ngay lần chạy đầu** |
| Cửa nạp chặn dữ liệu vô lý (vượt miền, âm, chỉ số/thực thể lạ, đơn vị khác chiều) | D1 | `TestNapFactHongBiChan` |
| Các cổng thật sự chặn | — | 8 đột biến (A–H), cả 8 bị bắt (xem STATUS) |
| Quy mô, dữ liệu giống thật, phục hồi | ✗ | **chưa chứng minh** |

## 8. Phân loại audit điểm mù (19/10/2026)

Nguyên văn bản audit 29 điểm lưu trong `docs/sources/` (không public).

> **Cập nhật 19/10 21:01, sau khi chốt WIN (`04`):** #2, #3, #4, #10 chuyển từ CA sang **HOÃN**: chỉ mở khi một ca WIN đỏ vì thiếu chúng. Chỗ khác trong bảng nói "Pha N" là theo 9 pha cũ của `02`, nay thay bằng các lát W của `04` và các Phase của `05`.

Mỗi điểm được xếp vào một trong ba loại:

- **CÓ**: đã có trong mã, kèm bằng chứng;
- **CA**: phải thành ca tái hiện trong lát dọc Đối soát nhiên liệu;
- **HOÃN**: chỉ mở khi có điều kiện kích hoạt (theo cổng capability, CLAUDE.md §19).

| # | Điểm | Loại | Ghi chú đối chiếu mã |
|---|---|---|---|
| 1 | Chuẩn hoá nghĩa nghiệp vụ | CÓ một phần + CA | Chỉ số tách nghĩa (`reported/issued/measured/billed`), thẩm quyền chỉ so trong một chỉ số. Thiếu bước diễn giải ngữ nghĩa lúc nạp: fixture đang gán sẵn chỉ số bằng tay |
| 2 | Mô hình ngữ nghĩa / ontology | CA | Hiện chỉ có quan hệ cha–con giữa thực thể |
| 3 | Thời gian song thời (bitemporal) | CA | `Fact` chỉ có ngày xảy ra + thời điểm quan sát. Thiếu `recorded_at`, `corrected_at`, hiệu lực |
| 4 | Vòng đời định danh | **CA ưu tiên** | `Expand` dùng cây cha–con HIỆN TẠI cho mọi kỳ. Máy đổi dự án thì truy vấn tháng cũ gộp sai dự án |
| 5 | QUERY → COMMAND / EVENT / WORKFLOW | CA | Mới có thiết kế (C0–C7, E0–E6), chưa có mã |
| 6–7 | Ranh giới giao dịch, thử lại gây hại | CA khi có lệnh đầu tiên | Thiết kế có idempotency; chưa có ca "side effect thành công, mất phản hồi, thử lại" |
| 8 | Chủ sở hữu dữ liệu | HOÃN | Mở khi có luồng sửa dữ liệu/cách ly |
| 9 | SLO chất lượng dữ liệu | CA cùng cửa nạp | Đo tỉ lệ đầy đủ, nhận dạng thực thể, đơn vị lạ, xung đột tồn đọng |
| 10 | Tiến hoá schema | CA cùng ánh xạ schema nguồn | |
| 11 | Tái lập kết quả | CÓ một phần | Có `pack_version` trong phản hồi; thiếu phiên bản mã, phiên bản dữ liệu |
| 12 | Index RAG lệch nguồn sự thật | HOÃN tới pha RAG | Cần ca bơm lỗi giữa ghi DB và ghi index |
| 13 | Phân quyền mịn tới trường | CÓ một phần | capability + phạm vi thực thể theo cây; chưa có mức trường |
| 14 | Prompt injection qua dữ liệu | CA trước khi mở route có model | Nội dung truy xuất / công cụ là DỮ LIỆU, không phải lệnh |
| 15 | Kiểm claim theo nghĩa | **CA ưu tiên** | `validation.Fidelity` mới kiểm số CÓ MẶT trong bằng chứng. "107" của số nhân viên sẽ lọt làm "doanh thu 107". **Chưa được mở route sinh số bằng model khi chưa có ràng buộc claim ↔ bằng chứng** |
| 16 | Hiệu chuẩn độ tin cậy | HOÃN | Độ tin cậy hiện là hằng số theo luật, chưa phải xác suất |
| 17–19 | Kinh tế duyệt tay, chi phí, SLA nghiệp vụ | HOÃN tới khi có runner eval | Đơn vị đo: chi phí trên mỗi việc nghiệp vụ thành công |
| 20 | Hàng xóm ồn (noisy neighbor) | HOÃN | Cần ca, chưa cần cài đặt |
| 21 | Chi phí thêm tenant | CÓ (cổng) | Pha 7: diff core = 0 khi thêm tenant thể thao |
| 22–23 | Thay model / thay retriever | HOÃN tới khi có gateway / RAG | |
| 24 | Bất biến nghiệp vụ, test theo thuộc tính | CA | Có miền min/max + kiểm tenant; chưa có property-based |
| 25 | Số ca thành chỉ số khoe | CÓ luật | Mục 4: đếm độ phủ |
| 26 | Ảo giác dữ liệu tổng hợp | CA | Ca phá do phiên KHÔNG viết mã soạn (CLAUDE.md §2.1) |
| 27 | Gỡ lỗi nhân quả | HOÃN | Diff cấu hình/dữ liệu giữa hai lần chạy |
| 28 | Mốc so sánh với hệ thống cũ | CÓ một phần | Có mốc test cũ; chưa có mốc hành vi trên bộ câu hỏi chung |
| 29 | Thành dự án nghiên cứu kiến trúc | CÓ (mục 2) | Sao Bắc Đẩu một lát dọc |
