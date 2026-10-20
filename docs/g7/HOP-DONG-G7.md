# Hợp đồng benchmark G7 — Fuel Truth Benchmark (W-G7 / R1)

> **Trạng thái: CHƯA ĐÓNG BĂNG.**
> - Chưa gọi model thật lần nào, chưa chi tiền.
> - Cập nhật đêm 20→21/10/2026 theo quyết định chủ dự án (`docs/CHO-QUYET-DINH.md` D-G7-1…11) và ADR-0005 (đã duyệt), ADR-0006.
> - Bản đề xuất trước: commit `b882172`.
>
> **Chỉ đóng băng khi đủ mọi điều kiện ở mục 13** (D-G7-7: agent được tự đóng băng khi đủ). Từ lúc đóng băng:
> - **không đổi** bộ ca, cách chấm, ngưỡng, tham số model, prompt;
> - muốn đổi thì viết ADR mới, đó là một thí nghiệm mới, chạy lại **toàn bộ**.

## 1. Câu hỏi benchmark và luật kết luận

**Câu hỏi:** tầng kiểm soát bao quanh model của EACL có tạo giá trị vượt một tích hợp AI đơn giản **nhưng có năng lực** không? Đo trên bài toán Fuel Discrepancy, cùng dữ liệu, cùng câu hỏi, cùng model.

| Kết quả G7 | Điều kiện (ADR-0005 §6) | Kết luận dự án |
|---|---|---|
| **PASS** | luật D5 sửa đổi (mục 7) đạt với **cả** B1 lẫn B2+; đủ lượt hợp lệ | Fuel WIN nếu G1–G6 đạt vô điều kiện + scale smoke: đóng băng "Fuel Truth Benchmark v1" rồi **DỪNG** Fuel |
| **PARTIAL** | đạt với đúng một baseline cổng; hoặc phải cắt hệ vì trần chi phí | Architecture Proof có thể PASS; Product Value WIN = PARTIAL |
| **FAIL** | vi phạm bất biến cứng (R3), hoặc hỏng R1 / R2 với B2+ | Architecture Proof có thể PASS; Product Value WIN = FAIL |

- **Không đổi định nghĩa WIN** để dự án đạt.
- G7 FAIL thì:
  - không sửa ngưỡng;
  - không sửa mã rồi chạy lại cùng bộ giữ lại;
  - phân loại tầng sai đầu tiên của từng ca hỏng;
  - ca đã thấy trượt thành REGRESSION.

## 2. Bộ ca và dữ liệu

| Thứ | Giá trị |
|---|---|
| **Bộ ca kết luận** | bộ giữ lại độc lập **H1** (`docs/g7/holdout/h1.yaml`); dùng **H2** nếu H1 đã bị dùng để sửa mã (ADR-0006 §1, §7). Băm: mục 13, điền lúc đóng băng bộ giữ lại |
| Pilot | `docs/g7/holdout/pilot.yaml`, 16 ca: chỉ để chọn k và kiểm tương thích, **không** là bằng chứng |
| Bộ 117 ca | `tenants/construction/win/cases-independent.yaml` (sha256 `ffdd704b…a4bf`): nay là **REGRESSION**, chỉ báo cáo. Kèm độ nhạy khi bỏ 083 / 107 / 108 |
| Thế giới | `tenants/construction/win/world.yaml`, sha256 `1631a6c6…e39e` |
| Dữ liệu | `gen -profile functional -seed 42`; `manifest.json` sha256 `31172355…adb055` (2.739 bản ghi tenant A) |
| Tenant được chấm | `abc-infra`; canary `def-build` |

**Thiên lệch đã biết, bắt buộc ghi vào báo cáo kết quả:**
- `docs/g7/CONTAMINATION-AUDIT.md`: bộ 117 có 3 ca nhiễm (083, 107, 108) và 12 ca không chắc. Bộ đó được băm sau lần chạy đầu. Mã EACL đã được lặp sửa tới 117/117 trên chính bộ đó.
- Bộ giữ lại do người không đọc mã soạn, người khác soát, và đóng băng + push **trước** lần chạy đầu. Chỗ tài liệu mâu thuẫn được giải theo thứ tự ưu tiên `04` > ADR-0001 > ADR-0002 > pack `metrics` / `policy` (ADR-0006 §2). Ca rơi vào mâu thuẫn `04` / D7 được báo riêng (D-G7-12).

## 3. Các hệ được so (ADR-0005 §2)

| Hệ | Vai | Dữ liệu được dùng | Lời gọi model |
|---|---|---|---|
| **EACL** | hệ được chấm | `pipeline.Ask` đường bộ nhớ: fact đã chuẩn hoá, danh mục chỉ số (thẩm quyền, dung sai) của pack, chỉ mục SOP (k = 3, lọc ACL theo vai) | chỉ ở route RAG / HYBRID |
| **B1** | baseline cổng | LLM trực tiếp. Mỗi câu hỏi nhận:<br>- `entities.json` và mọi tài liệu;<br>- mọi **bản ghi thô** (đọc thẳng tệp, kể cả bản ghi bẩn và PDF) có ngày trong `b1_window` của ca, do người soạn ca gán. Bản ghi không đọc được ngày luôn có mặt.<br>Kèm ngày hiện tại | 1 / câu hỏi |
| **B2+** | baseline cổng | RAG đơn giản **có năng lực**, top-10:<br>- mỗi quan sát một đoạn, có dự án + bí danh + mã theo nguồn của thực thể;<br>- kèm ngày hiện tại;<br>- **không** có tầng kiểm soát (biên ADR-0005 §2) | 1 / câu hỏi |
| B2 | chỉ báo cáo | baseline cũ: đoạn thiếu bí danh | 1 / câu hỏi |
| B2″ | chỉ báo cáo | B2+ cộng luật thẩm quyền + dung sai sinh máy móc từ `metrics.yaml`. Giới hạn: chưa gồm luật mặc định | 1 / câu hỏi |
| B2-oracle | chỉ báo cáo | B2+ nhưng thay truy xuất bằng **đúng** các đoạn của bản ghi ca trỏ tới (`truth.record_ids` của hồ sơ ca → khoá bản ghi của bộ sinh), cùng id và chữ như chỉ mục B2+; kèm ngày hiện tại. Ca không có `record_ids` (tài liệu, ngoài phạm vi, thiếu dữ liệu) hỏi B2+ y nguyên, nên chênh lệch với B2+ là phần do truy xuất. Cần `-manifest` có `truth` | 1 / câu hỏi |

Khác biệt truy cập dữ liệu, cố ý:
- **EACL dùng danh mục ngữ nghĩa + thẩm quyền nguồn của pack.** Đây chính là tầng kiểm soát đang được đo.
- **B2 / B2+ nhận quan sát đã chuẩn hoá** bởi bộ nạp của EACL. Đây là lợi thế cho baseline, làm phép so khắt khe hơn với EACL.
- **Không hệ nào thấy dữ liệu tenant B.** Chỉ EACL kiểm quyền người hỏi. Vì thế lớp `permission` bị loại khỏi phép đếm "lớp hơn".

## 4. Model và tham số (giống hệt cho mọi hệ)

| Tham số | Giá trị | Biến môi trường |
|---|---|---|
| Nhà cung cấp | Anthropic Messages API, client `models.Messages` | `EAIS_MODEL_PROVIDER=anthropic` |
| Model | **`claude-sonnet-5-5`** (D-G7-2). Haiku 5.5 chỉ cho báo cáo độ nhạy riêng | `EAIS_MODEL` |
| Sampling | **không gửi** `temperature` / `top_p` / `top_k`. Lần chạy là NGẪU NHIÊN | (không đặt `EAIS_MODEL_TEMPERATURE`) |
| Thinking | `{"type": "between_tools"}`: mức thấp nhất của Sonnet 5.5, chạy ở effort low / medium / high (tài liệu Effort, tra 20/10/2026) | `EAIS_MODEL_THINKING=between_tools` |
| Effort | `medium`, đặt ở `output_config.effort` | `EAIS_MODEL_EFFORT=medium` |
| `max_tokens` | 1024 | mặc định |
| Thử lại | 2 lần cho lỗi tạm (không tới được, quá giờ, 429, 5xx), lùi 1 s rồi 2 s; mọi lần gửi được ghi vết | `EAIS_MODEL_MAX_RETRIES=2` (mặc định) |
| System prompt | `messagesSystem`, sha256 `ce28446e…d781bf`. **Có thể đổi** theo bộ chấm v2, trước khi đóng băng | — |
| User prompt | `Câu hỏi: …` + `Bằng chứng:` mỗi đoạn `[id] văn bản`; B2+ / B2″ / B1 thêm phần mở đầu (ngày hiện tại; B2″ thêm luật) trước câu hỏi | — |
| Seed | API không có seed: không áp dụng | — |

**Thử khói bắt buộc trước lượt chạy:** `g7run -smoke 3`. Mọi câu hỏi phải không lỗi, mọi lời gọi kết thúc bằng HTTP 200. Tham số nào bị từ chối thì sửa **trước** khi đóng băng, kèm ghi chú ở ADR-0005.

- Mã thoát (`1276b5f`, C-13): **0 chỉ khi endpoint là API chính thức** và đạt; trượt là 1; hệ cổng bị chặn là KHÔNG ĐẠT; chạy khô (máy chủ giả, proxy) mà đạt là 2; model giả là 2.
- Giới hạn: thử khói lấy N ca **đầu**, nên đường gọi model của EACL có thể chưa được thử (dòng EACL ghi "lời gọi model 0"). Mọi hệ dùng chung một gateway nên bộ tham số vẫn được thử qua baseline; B1 (prompt lớn nhất) chỉ được thử nếu có trong N ca đầu.

## 5. Số lượt và tổng hợp (ADR-0005 §5, hiện thực ở `evals.CompareG7`, commit `a507e81`)

- **k:** pilot chạy k = 5. Hệ cổng nào có ≥ 2 ca chập chờn thì chạy chính k = 5, không thì k = 3 (ADR-0006 §6; `g7run -pilot`).
- **Điểm từng ca:** `pass_count / k`. Ca chập chờn (đạt ở 0 < n < k lượt) được liệt kê riêng cho từng hệ.
- **Điểm lớp:** trung vị qua k lượt của số ca đạt. Riêng `permission` và `unsupported_answer` tính **ANY-RUN**: ca chỉ đạt khi đạt ở mọi lượt, áp cho mọi hệ.
- **Lượt hợp lệ:** đường tất định của EACL giống nhau giữa các lượt:
  - câu KHÔNG gọi model: so đủ route, outcome, ứng viên, xung đột, trích dẫn;
  - câu CÓ gọi model: **chỉ so route** (chọn trước lời gọi model). Trích dẫn là id do model chọn, outcome phụ thuộc việc kiểm bản nháp ngẫu nhiên (bản soát mã độc lập S2-5, `a94b5fb`; `askRecord.path` trong `g7run/evidence.go`).
  - Khác thì lượt vô hiệu. `g7run` không tự chạy lại: nó ghi ca không ổn định vào `aggregate.json` (`eacl_unstable_cases`), in "LƯỢT VÔ HIỆU" và không bao giờ thoát 0 (`gatePass`). Người vận hành sửa nguyên nhân rồi chạy lại cả lượt (§10); lượt chạy lại vẫn khác thì G7 FAIL.

## 6. Cách chấm từng ca — ĐANG THIẾT KẾ LẠI

- **Bộ chấm v1** (`evals.G7ContentReport`) đã bị một người tấn công độc lập phá:
  - 170 đòn; 109 câu sai mà ĐẠT; 24 câu đúng mà TRƯỢT;
  - một câu chỉ liệt kê mọi số của bản ghi ĐẠT 73/117 ca.
  - Gốc: "túi số", không có phân cực, không ràng buộc thực thể / chỉ số / kỳ / nguồn, không kiểm đơn vị, không kiểm lịch ngày.
  - **v1 không được dùng làm bằng chứng G7.**
- **Bộ chấm v2:** do người không đọc mã EACL thiết kế (hai hướng + giám khảo), ghi vào ADR-0007 **trước** khi đóng băng. Sau đó hiện thực có test cho từng đòn, rồi chính người tấn công chạy lại đòn.
- Bộ giữ lại có trường `truth` có cấu trúc (thực thể, chỉ số, kỳ, giá trị, đơn vị, nguồn, id bản ghi, giá trị nguồn khác), để ràng buộc con số đúng vào đúng thứ nó nói.
- **Bất biến đã chốt:**
  - một tiêu chí cho mọi hệ;
  - bỏ qua nhãn kiến trúc EACL (route, outcome, `model_calls`, trích dẫn có cấu trúc);
  - câu trả lời rỗng của model = từ chối cho mọi hệ (D-G7-9 mục 6);
  - chuẩn hoá số trước khi so chuỗi cấm (mục 7);
  - canary tên + dạng chữ (mục 12).

## 7. Luật D5 sửa đổi (tên luật trong `CompareG7`)

Với **mỗi** baseline cổng (B1, B2+) so với EACL:

| Luật | Tên | Nội dung |
|---|---|---|
| R0 | `runs` | k ≥ 3 và lẻ |
| R1 | `no_class_behind` | mọi lớp trong 8 lớp: điểm EACL ≥ điểm baseline (hoà được phép); lớp không có ca là không đo được |
| R2 | `lead` | ≥ 4 trong 7 lớp (bỏ `permission`) có EACL − baseline ≥ 20 điểm %. Lớp dưới 12 ca không được tính |
| R3a | `no_leak` | EACL: 0 câu rò rỉ canary, mọi lượt |
| R3b | `no_unsupported_claim` | EACL: 0 số không căn cứ (bộ đếm độc lập), mọi lượt |
| R3c | `no_permission_exposure` | EACL: 0 lần lộ số được bảo vệ ở ca kỳ vọng `PERMISSION_DENIED`, mọi lượt, không miễn số của câu hỏi |
| R4 | `mcnemar` | McNemar chính xác một phía trên từng ca (đạt = đạt ở đa số lượt), p ≤ 0,01. Dưới 10 cặp lệch thì "không đủ điều kiện": không chặn, không tuyên bố ý nghĩa thống kê |

- Kèm theo: khoảng Wilson 95% từng lớp cho từng hệ.
- Ngưỡng **không đổi sau khi thấy kết quả.**

## 8. Phân loại lỗi (mỗi ca trượt một nhãn TẦNG SAI ĐẦU TIÊN)

DATA · NORMALIZATION · ENTITY · INTENT · SEMANTICS · SOURCE_AUTHORITY · PERMISSION · RETRIEVAL · CONTEXT · ROUTING · MODEL · CLAIM_BINDING · VALIDATION · SCORER · PROVIDER · UNKNOWN

- **Cách gán:**
  - EACL: theo bản ghi quyết định (chặng đầu tiên lệch so với kỳ vọng);
  - baseline: theo luật so câu trả lời với `truth`.
- **Quy trình:** máy gán trước (cột `first_wrong_layer` của `failures.jsonl`, mặc định "CHƯA PHÂN LOẠI"), người duyệt sau; ghi cả hai.
- **Ghi rõ:** bất biến nào hỏng. Ca rơi vào mâu thuẫn `04` / D7 ghi SEMANTICS kèm chú thích "xung đột chính sách" (D-G7-12).
- Không gộp lỗi thành "LLM sai".

## 9. Chi phí và độ trễ (chỉ báo cáo, không làm cổng)

- **Bảng giá đóng băng** (Pricing chính thức, tra 20/10/2026): Sonnet 5.5 vào 2 USD, ra 10 USD mỗi triệu token. Nằm trong `manifest.json`.
- **Trần 15 USD** cho toàn bộ G7 (pilot + chính, D-G7-3). `docs/g7/COST-PLAN.md` dự toán theo bộ đã đóng băng. Dự toán > 15 USD thì không chạy phần vượt.
- **Trần là của TỪNG lần chạy `g7run`** (cờ `-budget`, bắt buộc > 0 và ≤ 15, ghi đúng vào `manifest.json`; C-07, soát lượt 2). Chưa có sổ chi tiêu cộng dồn tự động, nên người vận hành đặt `-budget` của lần sau = 15 − tổng `spent_usd` trong `manifest.json` của mọi lần trước (thử khói, pilot, lượt chạy lại). Lời gọi hết hạn phía client có thể vẫn bị tính tiền mà không có usage để cộng: chừa phần dư khi đặt trần.
- **Ghi lại:**
  - `cost.json`: token vào / ra / cache theo hệ;
  - độ trễ từng câu hỏi và từng lần gửi HTTP trong `raw-runs.jsonl`.
- **Gọi đồng bộ**, không Batch API.

## 10. Lỗi nhà cung cấp, hết hạn mức

- 429 / 5xx / quá giờ / không tới được: thử lại 2 lần, lùi 1 s, 2 s. Hết lượt thì ca đó trượt với nhãn PROVIDER; không bỏ ca, không chạy riêng ca đó lại.
- **Lượt VÔ HIỆU** khi:
  - > 2% lời gọi của một hệ trong lượt đó trượt vì PROVIDER; hoặc
  - đường tất định của EACL khác giữa các lượt (mục 5).
  Khi đó ghi "invalid run", sửa nguyên nhân, chạy lại cả lượt của mọi hệ. Chạy lại là việc THỦ CÔNG: `g7run` chỉ báo lượt vô hiệu (in ra; `aggregate.json` có `eacl_unstable_cases` / `invalid_runs`; mã thoát khác 0), không tự gọi lại model.
- **Lỗi harness / bộ chấm** phát hiện giữa chừng: dừng, ghi invalid run, sửa, chạy lại toàn bộ phần bị ảnh hưởng (ADR mới nếu sau đóng băng).
- **Không sửa mã sản phẩm giữa benchmark** để cứu ca trượt.

## 11. Bằng chứng thô (`g7run -out <thư mục>`)

Mỗi lần chạy ghi vào `docs/g7/runs/<ngày>-<revision>/`:

| Tệp | Nội dung |
|---|---|
| `manifest.json` | revision + cờ dirty; Go; OS; model + tham số hiệu lực; sha256 system prompt; k; top-k; hệ; hệ cổng bị chặn; sha256 bộ ca / world / pack / hồ sơ ca / hợp đồng; sha256 manifest dữ liệu; bảng giá; ngân sách; ca EACL có đường tất định không ổn định |
| `raw-runs.jsonl` | mỗi câu hỏi × lượt × hệ một dòng: câu hỏi, câu trả lời, route, outcome, ứng viên, xung đột, trích dẫn, số lời gọi model, độ trễ, lỗi; mọi lần gửi HTTP (lần thử, mã, độ trễ, thân yêu cầu / phản hồi, token); với EACL thêm `decision` = bản ghi quyết định S2–S10 (`traces.Decision`) để gán tầng sai đầu tiên (§8) mà không gọi lại model (C-11). Không có khoá |
| `scored.jsonl` | chấm nội dung + chấm đủ (RunWin) từng dòng |
| `aggregate.json` | phán quyết D5 với từng baseline, `pass_count`, ca chập chờn, k (và k đề xuất nếu là pilot), số đếm độc lập |
| `failures.jsonl` | ca trượt + `first_wrong_layer` |
| `cost.json` | token + USD theo hệ |
| `ket-qua.md` | bản đọc cho người |

Không chỉ lưu điểm tổng hợp.

**Đọc bằng chứng khi tập còn là bộ giữ lại** (bản soát lượt 2, C-11). `scored.jsonl`, `failures.jsonl`, `ket-qua.md` và `raw-runs.jsonl` chứa câu hỏi và kỳ vọng dạng chữ ("thiếu số 3850", "route = RAG, muốn SQL_QUERY"). Vì vậy:
- chạy khô hay thử khói trên một tập còn là bộ giữ lại: thư mục `-out` đặt NGOÀI repo;
- phiên viết mã chỉ đọc qua công cụ in số (`scripts/du-toan-g7.py`), không mở các tệp trên;
- sau lần chạy thật, tập đã dùng xong, nên đọc được.
- Chưa có chế độ che nội dung (`-redact`): nợ, ghi ở CHANGELOG.

## 12. Lệnh (khi đủ mục 13 và có khoá do chủ môi trường đặt)

```bash
# 0. cây sạch, đúng revision đã đóng băng
git status --porcelain                       # phải rỗng
# 1. dữ liệu đóng băng
go run ./tenants/construction/win/gen/cmd/gen -world tenants/construction/win/world.yaml -out /tmp/g7data -profile functional -seed 42
sha256sum /tmp/g7data/manifest.json          # phải = 31172355…adb055
# 2. cấu hình model (khoá do chủ môi trường đặt; KHÔNG in, KHÔNG commit, agent KHÔNG tự tìm)
export EAIS_MODEL_PROVIDER=anthropic EAIS_MODEL=claude-sonnet-5-5 EAIS_MODEL_THINKING=between_tools EAIS_MODEL_EFFORT=medium
# 3. thử khói
go run ./tenants/construction/win/g7run -data /tmp/g7data -cases docs/g7/holdout/pilot.yaml -manifest docs/g7/holdout/holdout-manifest.json -smoke 3
# 4. pilot (k = 5) → k của lượt chính theo luật ADR-0006 §6
go run ./tenants/construction/win/g7run -data /tmp/g7data -cases docs/g7/holdout/pilot.yaml -manifest docs/g7/holdout/holdout-manifest.json -k 5 -pilot -out docs/g7/runs/pilot-$(git rev-parse --short HEAD)
# 5. lượt chính với k đã chọn. TẬP CHỜ D-H1-3: H1 đã lộ cho phiên viết mã lúc chạy G2'/G5'/G6' (21/10 01:42);
#    H2 không đủ n ≥ 12 / lớp; khuyến nghị bộ giữ lại mới H3. Dòng dưới giữ nguyên văn bản cũ.
go run ./tenants/construction/win/g7run -data /tmp/g7data -cases docs/g7/holdout/h1.yaml -manifest docs/g7/holdout/holdout-manifest.json -k <3|5> -out docs/g7/runs/$(date +%Y%m%d)-$(git rev-parse --short HEAD)
```

## 13. Đóng băng — điều kiện D-G7-7 (agent tự đóng băng khi đủ MỌI dòng)

| # | Điều kiện | Trạng thái (cập nhật 20/10 đêm) | Bằng chứng |
|---|---|---|---|
| 1 | D-G7-1…11 đã áp | ĐANG.<br>- Đã áp vào mã / tài liệu: 1, 2, 3, 4, 6 (phần baseline), 8, 10, 11.<br>- D-G7-9 đã làm: mục 2, 3, 4 (B2+, B2″, B1), 5 (B2-oracle), 7 (phần bộ đếm từng ca), 10, 11, 12 (canary tên), 15 (công khai `d5probe`).<br>- D-G7-9 mục 6 (rỗng = từ chối; hệ lỗi khi hỏi luôn trượt): đã làm `6b7d963`.<br>- D-G7-9 còn chờ: phần còn lại của 7 (thuộc bộ chấm v2, ADR-0007); 13 cần lượt chạy thật.<br>- Đang làm: D-G7-5 (bộ giữ lại), D-G7-6 (sửa bộ chấm) | ADR-0005, ADR-0006; `750307a`, `a1a5b6b`, `fdb3a18`, `a507e81`, `9264aa6`, `7f6711c`, `d03f962`, `15cb1a8`, `6b7d963` |
| 2 | Kiểm toán nhiễm xong | XONG | `docs/g7/CONTAMINATION-AUDIT.md` (`a135c90`) |
| 3 | Bộ giữ lại đóng băng + băm | **XONG 20/10 22:51** (`16a6140`, push 22:51:48). 189 ca sau soát (bỏ 40/229); loại 7 ca bị lộ trước khi chia ⇒ pilot 16, H1 104 (13 / lớp), H2 62, exposed 7; 0 lỗi định dạng. 15 ca có gợi ý kỳ vọng lộ cho phiên viết mã sau đóng băng: tập không đổi, bảng H1 báo độ nhạy (`59ee556`) | `docs/g7/holdout/holdout-manifest.json` (sha256 0f09e007…) |
| 4 | Test bộ chấm xanh | CHƯA. Hướng v2 đã chốt ở mức đề xuất (ADR-0007: nền bản 1, bản 2 không làm cổng). Không đóng băng được đêm nay: cần hiện thực (ước 10–15 ngày công), hiệu chỉnh trên pilot thật với người chấm mù (D-G7-16); lần nhìn đầu ngoài mẫu: trượt oan 4,6 %, đạt nhầm 9,0 % | `docs/adr/0007-bo-cham-g7-v2.md`, `docs/g7/BO-CHAM-V2-DAC-TA.md` |
| 5 | Biên B2+ đóng băng | XONG về định nghĩa (ADR-0005 §2) và mã (`a1a5b6b`); băm khi đóng băng hợp đồng | — |
| 6 | Pilot xong (dùng để chọn k) | **CHẶN**: cần khoá model; chưa có trong môi trường | — |
| 7 | Harness xuất được bằng chứng thô | XONG (chạy khô bằng model giả và máy chủ HTTP giả) | `a1a5b6b`, test `TestRunWritesEvidence`, `TestSmokeThroughFakeProvider` |
| 8 | Dự toán cuối ≤ 15 USD | ĐO XONG phần chạy khô (21/10). Chỉ hệ cổng, k = 3, không chạy lại: kỳ vọng 12,74 USD. Thêm một lần chạy lại: 16,36; k = 5: 19,82. Tỉ lệ ký tự / token CHƯA ĐO (cần khoá) ⇒ tính lại sau pilot; D-G7-18 | `docs/g7/COST-PLAN.md` |

| Mục | Giá trị |
|---|---|
| sha256 tệp này lúc đóng băng | _(điền khi đóng băng)_ |
| revision lúc đóng băng | _(điền khi đóng băng)_ |
| sha256 `h1.yaml` / `h2.yaml` / `pilot.yaml` / `holdout-manifest.json` | `00685c0d…` / `ea031baf…` / `3e1dd1fe…` / `0f09e007…` (đủ 64 ký tự trong thông điệp commit `16a6140`) |
