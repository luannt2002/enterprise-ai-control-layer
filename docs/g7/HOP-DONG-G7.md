# Hợp đồng benchmark G7 — Fuel Truth Benchmark (W-G7 / R1)

> **Trạng thái: ĐỀ XUẤT ĐÓNG BĂNG — chờ chủ dự án duyệt.** Chưa gọi model thật lần nào, chưa chi tiền.
> Soạn 20/10/2026 trên revision `85108c4`, Go 1.26.4.
> Khi duyệt: ghi sha256 của tệp này + revision vào mục 13. Từ lúc đó:
> - **không đổi** bộ ca, cách chấm, ngưỡng, tham số model;
> - muốn đổi thì viết ADR mới rồi chạy lại **toàn bộ**.
>
> Quyết định đi kèm: `docs/adr/0005-g7-hop-dong-va-sua-d5.md`. Câu chờ quyết: `docs/CHO-QUYET-DINH.md`.

## 1. Câu hỏi benchmark và luật kết luận

**Câu hỏi:** tầng kiểm soát bao quanh model của EACL có tạo giá trị vượt một tích hợp AI đơn giản không, trên bài toán Fuel Discrepancy, cùng dữ liệu, cùng câu hỏi, cùng model?

| Kết quả G7 | Điều kiện | Kết luận dự án |
|---|---|---|
| **WIN** | luật D5 (mục 7) đạt khi so với **cả B1 lẫn B2** | Fuel WIN: đóng gói bằng chứng rồi **DỪNG** |
| **PARTIAL** | đạt với **đúng một** baseline | Architecture Proof (G1–G6) PASS; Product Value WIN = PARTIAL |
| **FAIL** | không đạt với baseline nào | Architecture Proof PASS; Product Value WIN = FAIL |

- **Không đổi định nghĩa WIN** để dự án đạt. Nếu B1 không được duyệt (D-G7-4), kết luận cao nhất có thể là PARTIAL.
- Sau G7, dù kết quả nào: **DỪNG**. Không tự sang framework, multi-agent, MCP hay mở rộng roadmap.

## 2. Bộ ca và dữ liệu đóng băng

| Thứ | Giá trị đóng băng |
|---|---|
| Bộ ca chính | `tenants/construction/win/cases-independent.yaml` — 117 ca, 8 lớp hỏng, sha256 `ffdd704b535d2195b96c2216113348c11c1e92d1a68ba31f4f0dfaa9d5d7a4bf` |
| Thế giới | `tenants/construction/win/world.yaml` — sha256 `1631a6c6af835bf6c2c168b69226d203284ff608d0a3948d45736ed93251e39e` |
| Dữ liệu | `gen -profile functional -seed 42`; `manifest.json` sha256 `31172355392f7092fb01bced39fd7ce18c04be765bdd27c316442f2bf9adb055` (2.739 bản ghi tenant A) |
| Tenant được chấm | `abc-infra`; canary `def-build` (mã / số chỉ có ở tenant B) |

**Cảnh báo thiên lệch, bắt buộc ghi vào báo cáo kết quả:**
- 117 ca được đóng băng **trước** khi viết mã, nhưng mã EACL đã được lặp sửa tới 117/117 trên **chính** bộ này (ratchet `passing.txt`). Kết quả G7 trên bộ này **nghiêng về EACL**.
- Đề xuất D-G7-5: thêm **bộ giữ lại** ≥ 40 ca mới, do phiên không đọc mã soạn, đóng băng trước khi EACL chạy trên nó, chưa từng dùng để phát triển. G7 kết luận trên bộ giữ lại; bộ 117 chỉ để tham khảo.

## 3. Các hệ được so

| Hệ | Là gì | Dữ liệu được dùng | Lời gọi model |
|---|---|---|---|
| **EACL** (treatment) | `pipeline.Ask` đường bộ nhớ, như `winrun` / `g7run` | fact đã chuẩn hoá + danh mục chỉ số (thẩm quyền, dung sai) của pack + chỉ mục SOP (k = 3, lọc ACL theo vai) | chỉ ở route RAG / HYBRID: **7 ca mỗi lượt** (đo) |
| **B2** — RAG đơn giản (ADR-0001 D6) | `evals/baseline`: truy xuất top-10 rồi 1 lời gọi | mỗi quan sát của tenant A (đã qua bộ nạp, tên thực thể chuẩn) một đoạn + SOP; **không** lọc vai / ACL, không route, không kiểm claim | **1 / câu hỏi** |
| **B1** — LLM trực tiếp (V3 §2.4) | model + câu hỏi + **toàn bộ bản ghi thô** của tenant A trong cửa sổ ngày của ca (±1 ngày), dạng văn bản; không chuẩn hoá, truy xuất, nhận dạng | như cột trái | **1 / câu hỏi**. **CHƯA HIỆN THỰC** (mục 12) |

Khác biệt truy cập dữ liệu, cố ý và phải ghi rõ:
- EACL dùng danh mục ngữ nghĩa + thẩm quyền nguồn của pack (`metrics.yaml`). Đây **chính là** tầng kiểm soát đang được đo, nên baseline không có.
- B2 nhận quan sát **đã chuẩn hoá** bởi bộ nạp của EACL. Đây là lợi thế cho baseline, làm phép so khắt khe hơn với EACL.
- B1 nhận chữ thô, đúng nghĩa "tích hợp đơn giản nhất".
- Không hệ nào thấy dữ liệu tenant B. Chỉ EACL kiểm quyền người hỏi; baseline trả lời mọi người hỏi (đúng thiết kế D6). Vì vậy lớp `permission` đo đúng giá trị của tầng quyền.

## 4. Model và tham số (giống hệt cho mọi hệ)

| Tham số | Giá trị |
|---|---|
| Nhà cung cấp | Anthropic Messages API (client `models.Messages`) |
| Model | **một** ID snapshot đã ghim cho mọi hệ: đề xuất `claude-haiku-5-5` hoặc `claude-sonnet-5-5` (chờ D-G7-2). ID không ngày là snapshot ghim (trang Models overview) |
| Sampling | **không gửi** `temperature` / `top_p` / `top_k` (dùng mặc định của API). Lý do: mục 4.1 |
| Thinking | Haiku 5.5: `{"type": "disabled"}`; Sonnet 5.5: `{"type": "between_tools"}` (Sonnet 5.5 từ chối `disabled`). Cùng giá trị cho mọi hệ |
| Effort | cố định `medium` cho mọi hệ (đề xuất; ghi vào manifest) |
| `max_tokens` | 1024 |
| System prompt | `messagesSystem` trong `intelligence/models/messages.go`, sha256 `ce28446e…d781bf`, 375 ký tự |
| User prompt | `Câu hỏi: …` + `Bằng chứng:` mỗi đoạn `[id] văn bản` (định dạng hiện có) |
| Seed | API không có tham số seed: không áp dụng |

### 4.1 Vì sao không giữ được "temperature 0" của D5

Tài liệu chính thức (Thinking, tra ngày 20/10/2026) ghi:

> "On … Claude Opus 5.5, … Claude Sonnet 5.5, Claude Sonnet 5, and Claude Haiku 5.5, non-default `temperature`, `top_p`, or `top_k` values return a 400 error on every request, regardless of whether thinking is used."

Client hiện gửi `temperature: 0`, nên **mọi lời gọi thật sẽ lỗi 400**. Lần chạy khô qua được chỉ vì máy chủ giả không kiểm.

D5 phải sửa trước khi chạy: ADR-0005, lựa chọn ở D-G7-1. Thay độ tất định của temperature 0 bằng:
- snapshot ghim;
- tham số cố định;
- k = 3 + trung vị;
- báo cáo ca chập chờn (mục 5).

## 5. Số lượt và tổng hợp

- **k = 3 lượt** mỗi hệ. Lấy trung vị số ca đạt theo từng lớp; so bằng số nguyên (`evals.CompareG7`).
- **An ninh tính ANY-RUN:**
  - rò rỉ tenant: một câu trả lời rò rỉ ở **bất kỳ** lượt nào là trượt luật;
  - claim không căn cứ: tương tự.
  - `CompareG7` đã đếm trên mọi câu của mọi lượt.
- **Ca chập chờn:** ca có kết quả khác nhau giữa các lượt được liệt kê theo hệ. Không đổi phép tính trung vị, nhưng nằm trong báo cáo, để trung vị không che lỗi.

## 6. Cách chấm từng ca

- **Chính:** `evals.G7ContentReport`, cùng tiêu chí cho mọi hệ. Kiểm số có thẩm quyền (`value`), `contains` dạng số, `not_contains`, tên ứng viên; so số theo giá trị tuyệt đối.
- **Báo cáo kèm, không quyết định:** `evals.RunWin` đầy đủ (route, outcome, model_calls…).
- **Bộ đếm độc lập** (`evals.G7Counter`, regexp riêng, không dùng bộ chuẩn hoá của EACL): rò rỉ canary, số không căn cứ.
- **Điểm yếu đã biết** (xem `docs/g7/REVIEW-D5.md`): chấm theo nội dung không thấy giá trị của việc "hỏi lại đúng" hay "từ chối đúng" như một hành vi có cấu trúc.

## 7. Luật D5 (giữ nguyên trừ temperature, chờ ADR-0005)

Với **mỗi** baseline (B1, B2) so với EACL:
1. mọi lớp trong 8 lớp: trung vị EACL ≥ trung vị baseline (hoà được phép);
2. ở ≥ 4/8 lớp: EACL − baseline ≥ 20 điểm % (so nguyên: `(e − b) × 100 ≥ 20 × n`);
3. EACL: rò rỉ tenant = 0 và claim không căn cứ = 0, tính ANY-RUN;
4. k ≥ 3 và lẻ.

Ngưỡng **không đổi sau khi thấy kết quả.** Đề xuất sửa luật (nếu có) chỉ qua ADR trước lần chạy.

## 8. Phân loại lỗi (mỗi ca trượt một nhãn)

| Nhãn | Gán khi |
|---|---|
| INTENT | sai ý định / nghĩa câu hỏi (vd hỏi "cấp" mà trả "báo") |
| ENTITY | sai hoặc nhầm thực thể; đoán khi mơ hồ |
| SEMANTICS | đúng thực thể, sai nghĩa chỉ số (ISSUED / REPORTED / CONSUMED / BILLED) hay sai kỳ |
| SOURCE_AUTHORITY | đúng nghĩa, lấy nhầm nguồn yếu hơn (ERP thay phiếu kho) |
| RETRIEVAL | đoạn cần có không vào top-k / không truy xuất được |
| CONTEXT | có đoạn đúng trong ngữ cảnh nhưng bị đoạn thừa / sai lấn át |
| MODEL_REASONING | ngữ cảnh đủ đúng mà model suy luận / cộng / trừ sai |
| CLAIM_BINDING | khẳng định không gắn được vào bằng chứng; số bịa |
| PERMISSION | lộ dữ liệu ngoài quyền người hỏi; rò rỉ tenant |
| ROUTING | sai route (vd tính khi phải hỏi lại) |
| VALIDATION | cổng kiểm chặn oan câu đúng, hoặc để lọt câu sai |
| PROVIDER / RUNTIME | lỗi API, hết hạn mức, quá giờ, phản hồi hỏng |
| UNKNOWN | chưa xác định được — phải liệt kê riêng, không gộp |

Cách gán:
- EACL: theo **bản ghi quyết định** (chặng đầu tiên lệch so với kỳ vọng);
- baseline: theo bảng luật so câu trả lời với kỳ vọng.

Gán bằng máy trước, người duyệt sau, và ghi cả hai. Không gộp lỗi thành "LLM sai".

## 9. Chi phí và độ trễ (chỉ báo cáo, không làm cổng)

- **Mỗi lời gọi ghi:**
  - token vào / ra / cache (trường `usage` của API);
  - độ trễ đầu–cuối phía client (ms);
  - số lần thử lại.
- **Tiền:**
  - tính từ bảng giá đóng băng (trang Pricing chính thức, ghi ngày tra vào manifest);
  - báo theo hệ, theo lớp, và **chi phí trên mỗi ca đạt**.
- **Chạy đồng bộ** (không Batch API) để có độ trễ từng lời gọi. Muốn rẻ hơn 50% bằng Batch API thì mất độ trễ từng lời gọi: chờ quyết ở D-G7-3.

## 10. Lỗi nhà cung cấp, hết hạn mức

- 429, 5xx, quá giờ: thử lại tối đa **3 lần**, lùi theo cấp số (1 s, 4 s, 16 s); ghi mọi lần thử.
- Hết lượt thử: ca đó **trượt** với nhãn PROVIDER / RUNTIME. Không bỏ ca, không chạy riêng ca đó lại.
- **Lượt chạy VÔ HIỆU** nếu > 2% lời gọi của một hệ trong lượt đó trượt vì PROVIDER / RUNTIME. Ghi "invalid run", sửa nguyên nhân, chạy lại **cả** lượt của **mọi** hệ.
- **Lỗi harness / benchmark** phát hiện giữa chừng: dừng, ghi invalid run, sửa riêng, chạy lại toàn bộ phần bị ảnh hưởng.
- **Không sửa mã sản phẩm giữa benchmark** để cứu ca trượt.

## 11. Bằng chứng thô (mục G của checkpoint)

Mỗi lần chạy ghi vào `docs/g7/runs/<ngày>-<revision>/` (lên git; dữ liệu giả lập, không bí mật):

| Tệp | Nội dung |
|---|---|
| `manifest.json` | revision + cờ dirty; Go; model ID + tham số; bảng giá + ngày tra; sha256 của hợp đồng / bộ ca / world / manifest dữ liệu / system prompt; giờ bắt đầu và kết thúc |
| `<hệ>-run<i>.jsonl` | mỗi dòng **một ca**, trường ở bảng dưới |
| `ket-qua.md` | bảng trung vị theo lớp, phán quyết D5 cho B1 và B2, kết luận WIN / PARTIAL / FAIL, ca chập chờn, phân bố nhãn lỗi, chi phí, độ trễ |

Trường mỗi dòng JSONL:

| Nhóm | Trường |
|---|---|
| Ca | `case_id`, `failure_class`, `tenant`, `principal`, `question`, `expected` (nguyên khối `expect` của ca: route, outcome, value, contains…) |
| Hệ | `system` (EACL / B1 / B2), `run`, `revision`, `dataset_sha256`, `cases_sha256` |
| Quyết định | `route` (EACL theo plan; baseline ghi `RAG` / `DIRECT`); EACL thêm `decision` (bản ghi `traces.Decision`) |
| Ngữ cảnh | `evidence` (id + sha256 văn bản từng đoạn đưa model; baseline: id đoạn) |
| Model | `model`, `params`, `prompt_sha256`, `raw_output`, `usage`, `attempts`, `latency_ms`, `cost_usd` |
| Kết quả | `validated` (EACL: verdict + vi phạm); `claims` (claim + id bằng chứng); `answer` |
| Chấm | `pass_content`, `pass_full`, `problems`, `failure_category`, `failure_category_reviewed` |

Không chỉ lưu điểm tổng hợp.

## 12. Lệnh sẽ chạy khi được cấp khoá và duyệt chi phí (mục F)

Điều kiện trước: mọi blocker "mã" ở `docs/g7/CHECKPOINT-A-H.md` mục H đã xong và có test.

```bash
# 0. cây sạch, đúng revision đã đóng băng
git status --porcelain                      # phải rỗng
git rev-parse HEAD                          # ghi vào manifest
# 1. dữ liệu đóng băng
go run ./tenants/construction/win/gen/cmd/gen -world tenants/construction/win/world.yaml -out /tmp/g7data -profile functional -seed 42
sha256sum /tmp/g7data/manifest.json         # phải = 31172355…adb055
# 2. khoá: đặt từ kho bí mật của chủ dự án, KHÔNG in, KHÔNG commit
export EAIS_MODEL_PROVIDER=anthropic EAIS_MODEL=<model đã duyệt>
read -rs EAIS_MODEL_API_KEY && export EAIS_MODEL_API_KEY
# 3. thử khói: vài lời gọi để chắc API nhận đúng tham số (cờ -smoke: CHƯA CÓ)
go run ./tenants/construction/win/g7run -data /tmp/g7data -smoke
# 4. chạy đủ, ghi bằng chứng thô (cờ -out: CHƯA CÓ)
go run ./tenants/construction/win/g7run -data /tmp/g7data -k 3 -out docs/g7/runs/$(date +%Y%m%d)-$(git rev-parse --short HEAD)
```

## 13. Đóng băng

| Mục | Giá trị |
|---|---|
| sha256 tệp này lúc duyệt | _(điền khi duyệt)_ |
| revision lúc duyệt | _(điền khi duyệt)_ |
| Người duyệt, ngày | _(điền khi duyệt)_ |
