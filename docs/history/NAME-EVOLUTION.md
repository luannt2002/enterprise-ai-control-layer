# Dòng thời gian tên của EAIS

Tệp lịch sử: ghi các lần đổi tên đã xảy ra, để người đọc lịch sử git hiểu vì sao cùng một dự án mang ba tên. Tên hiện hành chỉ đọc ở
ADR-0013 và `docs/CONVENTIONS.md` §0. Mỗi SHA dưới đây kiểm được bằng `git show <sha> --stat`; thời điểm lấy từ `git log --date=format:'%d/%m %H:%M'`.

| Thời điểm (2026) | SHA | Sự kiện | Kiểm bằng |
|---|---|---|---|
| 20/10 10:43 | `d70ebc8`, `977168c` | Tài liệu nền và mã đầu tiên dưới tên **EAIS — Enterprise AI Systems**; `go.mod` = `github.com/luannt2002/enterprise-ai-systems` | `git show 977168c:go.mod \| head -1`; `git log -S 'Enterprise AI Systems' --reverse` |
| 20/10 18:58 | `f4118f1`; duyệt ở `c5ce51e` (D-V3-00) | Kế hoạch V3 dùng tên **EACL — Enterprise AI Control Layer** lần đầu. Lý do ghi lúc đó: "Enterprise AI Systems" quá rộng (`docs/v3/00-CONTEXT-AUDIT.md`, dòng 57) | `git log --all -S 'Enterprise AI Control Layer' --reverse` → `f4118f1` đầu tiên |
| 21/10 09:29–09:33 | `88f4dcc`, `35c8e61` | Đường dẫn module, import và định danh chạy chuyển sang tên EACL; `go.mod` → `github.com/luannt2002/enterprise-ai-control-layer`; tài liệu hiện hành theo tên đó | `git show 88f4dcc:go.mod \| head -1`; `git log -- go.mod` |
| 21/10 18:06–19:19 | `32c8cfe`, `945777c`, `423e4d3`, `730926f`, `34240a9` | **D-RM-02**: đổi lại về tên gốc EAIS cho mã, luồng, tài liệu; **giữ link git** (tên repo, remote, đường dẫn module). Cổng danh tính đảo chiều (`danh_tinh_test.go`); bí danh biến môi trường `EACL_*` còn được đọc khi `EAIS_*` không đặt; migration 0002 gỡ quyền vai cũ | `git show 945777c --stat`; `git show 423e4d3 --stat` |
| 24/10 | ADR-0013 | Tên hiển thị **EAIS — Enterprise AI Integration Reference Architecture**. Viết tắt EAIS và mọi định danh chạy (biến `EAIS_*`, GUC `eais.tenant_id`, database `eais`, vai `eais_app`, container `eais-pg`, binary `eais`, đường dẫn module) **không đổi** | `docs/adr/0013-dinh-nghia-san-pham-vnext.md` |

Hệ quả còn thấy trong repo:
- đường dẫn module Go vẫn mang tên thời EACL (D-RM-02 giữ link git; ADR-0013 giữ tiếp, đổi chỉ bằng ADR riêng);
- tài liệu lịch sử (`docs/v3/`, `docs/g7/`, `docs/bao-cao/2026-10-2x*`, `CHANGELOG.md`, `docs/CHO-QUYET-DINH.md`) giữ nguyên tên lúc viết;
  sửa chúng là sửa lịch sử;
- muối băm chia bộ giữ lại H1 / H2 mang tên cũ và **không được đổi** (đổi muối là đổi cách chia đã đóng băng, ADR-0006 §4);
- mọi lần tên cũ xuất hiện đều phải có mục trong `legacyAllowed` của `danh_tinh_test.go`, hạn mức đúng bằng số lần.
