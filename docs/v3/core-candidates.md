# Ứng viên lõi chung — CHỈ ghi nhận, KHÔNG trích xuất

> **Khung (D-RM-05, 22/10/2026):** tệp để tra cứu — roadmap hiện hành là `docs/ROADMAP-DAY-DU.md` (D-RM-01). Tên sản phẩm trong thân tệp là tên lúc viết; tên hiện hành là **EAIS** (D-RM-02).

> Soạn đêm 20→21/10/2026 theo prompt cố vấn §28, được chủ dự án duyệt cho luồng đêm. **Trích xuất lõi chung đang bị CẤM.**
> - Logic riêng của Fuel ở lại Fuel; logic riêng của Customer Ops ở lại Customer Ops.
> - Tệp này chỉ ghi khái niệm có thể dùng lại, để lúc có bằng chứng từ miền thứ hai thì cân nhắc.
>
> **Trạng thái mọi dòng: `CANDIDATE_ONLY`.**
> - Cột "Customer Ops dùng thế nào" là **giả thuyết**: app tham chiếu Customer Ops chưa có dòng mã nào (D-V3-09: chưa implement khi G7 / Fuel còn là đường găng).
> - Chỉ nâng một ứng viên lên lõi khi cả hai miền đã có mã chạy dùng nó, cùng bất biến, và có test ở cả hai phía (`CLAUDE.md` §24, §34).

| # | Khái niệm | Fuel dùng ở đâu (mã có thật) | Customer Ops dùng thế nào (giả thuyết, Story 1 / 2) | Bất biến chung | Rủi ro rò rỉ miền | Trạng thái |
|---|---|---|---|---|---|---|
| 1 | Tenant lấy từ Principal đã xác thực, không từ payload | `core/contracts/tenantctx.go`; S1 trong `core/pipeline/ask.go`; `architecture_test.go` (không kiểu `*Request` nào có trường tenant) | đọc khách hàng / hoạt động / tác vụ của đúng tenant; tạo tác vụ đúng tenant | tenant chỉ đến từ danh tính; mọi truy cập dữ liệu mang tenant | thấp: không chứa khái niệm nghiệp vụ | CANDIDATE_ONLY |
| 2 | RLS FORCE + GUC tenant đặt bằng `SET LOCAL` trong transaction | `db/migrations/0001_win.sql`, `platform/pgstore` | bảng Customer / Activity / Task / Document | mọi bảng nghiệp vụ có RLS; vai ứng dụng không vượt RLS | thấp | CANDIDATE_ONLY |
| 3 | Kết quả có kiểu (`Outcome`: RESOLVED, CLARIFICATION_REQUIRED, INSUFFICIENT_EVIDENCE, PERMISSION_DENIED, HUMAN_REVIEW_REQUIRED, OUT_OF_SCOPE…) | `core/contracts/contracts.go` | bản tóm tắt khách hàng; hỏi lại khi "Acme" mơ hồ; từ chối khi không có quyền | không đoán im lặng; thiếu bằng chứng thì nói thiếu | trung bình: tập giá trị có thể phải mở rộng cho hành động (đề xuất / đã thực thi / bị chặn) | CANDIDATE_ONLY |
| 4 | Ràng buộc claim ↔ bằng chứng theo 6 chiều (chủ thể / chỉ số / kỳ / giá trị / đơn vị / nguồn) + bộ đếm số độc lập | `reliability/validation/binding.go`, `knowledge.go`, `numeric.go`; S9 trong `ask.go` | mỗi claim của bản tóm tắt (doanh số, số hoạt động, hạn tác vụ) gắn với bản ghi | claim chỉ được giao khi gắn được bằng chứng | **cao**: "chỉ số", "đơn vị" hiện mang hình dạng nhiên liệu (lít, m³). Cần chứng minh trên dữ liệu không phải số lượng (ngày hẹn, trạng thái) trước khi chung hoá | CANDIDATE_ONLY |
| 5 | Thẩm quyền nguồn TRONG cùng một nghĩa + dung sai + lộ xung đột | `data/sourceoftruth` (`query.go`, `catalog.go`); pack `metrics.yaml` | ví dụ email khách so với CRM; ngày ký hợp đồng ở hai hệ | thẩm quyền ≠ mới nhất ≠ giống câu hỏi; chỉ so trong cùng nghĩa; cùng hạng lệch ⇒ người duyệt | **cao**: dung sai số học là đặc thù số lượng; Customer Ops nhiều trường phi số | CANDIDATE_ONLY |
| 6 | Nhận dạng thực thể với bí danh theo nguồn + mơ hồ ⇒ hỏi lại kèm ứng viên | `data/entityresolution`; bí danh nguồn trong `entities.json` | "Acme" ⇒ Acme Corp / Acme Ltd | không chọn im lặng khi > 1 ứng viên | trung bình: luật dấu tiếng Việt và mã thiết bị nằm trong cùng gói | CANDIDATE_ONLY |
| 7 | Bản ghi quyết định + phát lại tất định | `observability/traces/decision.go`, `Replay` trong `ask.go` | phát lại bản tóm tắt; với hành động: đề xuất → duyệt → thực thi → đọc lại | chạy lại các chặng tất định phải ra đúng phản hồi | thấp | CANDIDATE_ONLY |
| 8 | Gateway model: không gửi sampling, tham số qua cấu hình, thử lại có ghi vết, khoá không vào thân yêu cầu | `intelligence/models/messages.go` | tóm tắt hoạt động; diễn giải yêu cầu hành động | model là đề xuất, không là sự thật; route không cần model thì không gọi model | thấp | CANDIDATE_ONLY |
| 9 | Truy xuất tài liệu có ACL theo vai, tất định | `intelligence/rag/rag.go` | tài liệu khách hàng (hợp đồng, biên bản) | không vượt ACL; thứ tự tất định | trung bình: từ dừng / cụm nghi vấn tiếng Việt viết cứng (nợ D-OP-6) | CANDIDATE_ONLY |
| 10 | Pack mô tả chính sách (ý định, chỉ số, mẫu câu, quyền) | `tenants/construction/pack/*.yaml`, `core/tenant` | pack của tenant Customer Ops | pack mô tả, mã hành xử (`CLAUDE.md` §17) | **cao**: nguy cơ YAML thành ngôn ngữ lập trình | CANDIDATE_ONLY |
| 11 | Bộ ca có `failure_class` + `authored_by`, đóng băng trước lần chạy đầu; bộ giữ lại tách riêng | `evals/win.go`; ADR-0006; `scripts/dong-bang-giu-lai.py` | bộ ca cho Story 1 / 2 | DEV / REGRESSION / HOLDOUT tách bạch; không chỉnh mã theo bộ giữ lại | thấp | CANDIDATE_ONLY |
| 12 | Hành động: AI đề xuất, hệ quyết (đề xuất → kiểm định dạng → quyền → chính sách → idempotency → thực thi → đọc lại → kiểm) | **không có ở Fuel** (WIN chỉ có QUERY) | Story 2: tạo một tác vụ theo dõi | AI có thể đề xuất, không được tự cấp quyền | — | chưa là ứng viên: chưa miền nào có mã |

## Ghi chú

- **Ứng viên mang rủi ro rò rỉ miền cao (4, 5, 10):** chính là chỗ Fuel định hình mạnh nhất. Nếu Customer Ops cần chúng ở dạng khác hẳn, đó là bằng chứng chúng thuộc miền, không thuộc lõi.
- **Không tạo gói `core/…` mới, không đổi tên module** chỉ để chuẩn bị cho các ứng viên này (D-V3-03: đổi tên mã là mẻ riêng sau khi bằng chứng ổn định).
