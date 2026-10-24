# Giao nhau RAG ↔ nhánh team chưa gộp: 3 nhánh xung đột khi rebase lên main 24/10

- **Độ tin:** CONFIRMED.
  - Lệnh: `git merge-tree --write-tree --name-only` của từng nhánh team chưa gộp, gộp khô vào hai cây:
    - main TRƯỚC RAG (`d6c463d`);
    - cây sau D-RAG-01 + D-RAG-04.
  - Xung đột chỉ có ở cây sau = do RAG gây ra.
- **Mức:** chặn rebase, không chặn chạy.
  - Không có xung đột ngữ nghĩa nào: ở mọi khối, hai bên cùng THÊM vào một chỗ.
  - Giữ cả hai là đúng.
- **Không làm:** phiên RAG không sửa nhánh của team. Ai rebase nhánh thì gộp theo bảng dưới, rồi chạy lại cổng của nhánh đó.

## Nhánh và khối xung đột do RAG

| Nhánh | Tệp | Hai bên | Cách gộp |
|---|---|---|---|
| `m3/s4-vong-sua-2310` (vòng sửa S4, `CHO` 23/10 trưa: rebase → soát → gộp → H6) | `core/pipeline/ask.go` ngay sau S4 | RAG: S4 chạy trong `traces.Measure` kèm móc dự phòng `IntentFallback` (mặc định tắt) và `if err != nil { return Outcome{}, err }`. Team: lọc ý định chưa thấy (`unseen`) cho MỌI đường hỏi lại | giữ khối `Measure` + `if err` của RAG; ngay sau đó dùng NGUYÊN khối `unseen` của team (thay khối `cls.NeedMeaning = slices.DeleteFunc…` cũ) |
| `m1/vet-s8-s10` (vết S8 / S10) | `core/pipeline/ask.go` trong `answerKnowledge` sau khi model trả bản nháp | RAG: `draft.Refused` ⇒ INSUFFICIENT_EVIDENCE, không gọi S9. Team: `a.noteModelDraft(ctx, why, prefix, draft.Model, draft.Text)` | gọi `noteModelDraft` TRƯỚC, rồi khối `if draft.Refused { … }` của RAG (vết S8 có cả lượt model từ chối) |
| `m1/vet-s8-s10` | `observability/traces/decision.go` (`Why.apply`) | RAG thêm `&w.Execute`; team thêm `&w.Generate, &w.Respond` | hợp: `Normalize, Resolve, Intent, Route, Execute, Context, Validate, Generate, Respond` |
| `m1/vet-s8-s10` | `observability/traces/step.go` chặng S6 | RAG đổi `if / else` thành `switch` (nhánh tri thức thuần dùng `Why.Execute`). Team thêm khối chú thích NỢ V8a vào nhánh `else` | giữ `switch` của RAG (`default:`), chép khối chú thích của team vào ngay dưới `default:` |
| `m1/5e-data-version` | `platform/pgstore/{ready.go, migrate_test.go, vet_test.go}` | RAG thêm `readyColumns` (cột `documents.version` của 0007) và tên migration 0007 / 0008 vào danh sách. Nhánh team là bản trước khi 5e gộp (`6fa6ec6`) | nếu còn dùng nhánh: giữ phần của RAG, danh sách migration giữ đủ 0001–0008 |

## Tệp dùng chung đã đổi (cho người rebase)

- Danh sách đủ: `git diff --name-only d6c463d main`.
- Tệp lõi:
  - `core/contracts/{contracts.go, tenantctx.go, fault.go}`;
  - `core/pipeline/{ask.go, why.go, chu_de.go, du_phong_s4.go}`;
  - `core/tenant/pack.go`;
  - `intelligence/{models, rag, router/router.go}`;
  - `observability/traces/{decision.go, step.go}`;
  - `platform/pgstore/{load.go, pgstore.go, ready.go, knowledge_sql.go, vector.go}`.
- Migration: 0007, 0008 đã lấy số. Migration mới tiếp theo là **0009**.

## Kiểm lại sau khi gộp nhánh

Chạy cổng của nhánh: `go test ./...` có PG, so theo tên với mốc của nhánh, cộng winreport 117 / H1 / H3 / H4.

Hai test chạm đúng chỗ giao:
- `tenants/construction/du_phong_s4_test.go` — móc S4;
- `core/pipeline/vet_truy_xuat_test.go` — vết S6.
