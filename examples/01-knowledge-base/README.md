# Ví dụ 01 — kho tri thức trung tính

**Độ chín: TESTED trên dữ liệu GIẢ LẬP.** Chạy trong test (động cơ RAG, PostgreSQL, trọn luồng hỏi S0–S10). Chưa nối vào
ứng dụng `cmd/eais`, chưa đo bằng bộ câu độc lập, chưa có người dùng thật (không phải BUSINESS_VALIDATED).

Ví dụ này chứng minh RAG chạy mà **không cần pack tham chiếu** ở `tenants/`: không số liệu, không thực thể, không đọc tệp nào
của miền khác (`go list -deps -test ./examples/01-knowledge-base/` không có gói nào dưới `tenants/`).

## Có gì

- `docs/`: sổ tay quy trình nội bộ của một tổ chức **hư cấu**. Nội dung tự viết, giả lập, không lấy từ tổ chức thật nào.
  - Nghỉ phép năm: bản 1 (hết hiệu lực từ 01/01/2026) và bản 2 (đang hiệu lực).
  - Mua sắm: có bảng người phê duyệt theo giá trị khoản mua.
  - Bảo mật tài khoản quản trị: chỉ vai `quan-tri` được đọc.
  - Làm thêm giờ: bản nháp, không bao giờ được phục vụ.
  - `acl.json`: vai nào đọc được tài liệu nào.
- `pack/`: pack tối thiểu chỉ có tài liệu. Một ý định RAG, bốn vai, `metrics: []`, `units: []`.
- `kho_test.go`: ba test.
  - `TestKhoTriThucTrungTinh`: động cơ trong bộ nhớ. Kiểm phiên bản theo ngày hỏi, quyền đọc theo vai, bản nháp, hàng bảng
    mang tên cột, câu ngoài kho.
  - `TestKhoTriThucTrungTinhQuaPG`: nạp **chỉ tài liệu** vào PostgreSQL (`Store.LoadDocuments`), mọi ca ra giống đường bộ nhớ.
  - `TestLuongHoiTrungTinh`: trọn luồng hỏi với pack ở `pack/` và model trích tất định (không gọi model ngoài).

## Chạy

```sh
go test -count=1 ./examples/01-knowledge-base/
```

Test PostgreSQL bị bỏ qua khi thiếu `EAIS_PG_ADMIN_DSN` / `EAIS_PG_DSN`. Đặt hai biến đó (xem `deploy/docker-compose.yml`)
để chạy nó.

## Giới hạn đã biết

- Pack chỉ có tài liệu vẫn phải khai các mẫu câu của luồng số liệu (`no_data`, `needs_review`, `approx_note`,
  `conflict_note`), và vẫn phải có `lexicon.yaml` / `metrics.yaml` (được phép rỗng). Lõi đòi chúng cho mọi pack
  (`core/tenant/pack.go`, `requiredTemplates`).
- Đường từ khoá (đường của app) trả cả đoạn chỉ chạm vài từ. Luật "chủ đề ngoài kho" chỉ chạy khi lượt hỏi mang chủ đề, tức
  là đi qua luồng hỏi. Gọi thẳng `Retrieve` thì không có luật này.
