# Ví dụ 02 — SME bán hàng tối thiểu

**Độ chín: TESTED trên dữ liệu GIẢ LẬP.** Chạy trong test (đường nạp dùng chung + trọn luồng hỏi S0–S10, trong bộ nhớ). Chưa nối
vào ứng dụng `cmd/eais`, chưa đo bằng bộ câu độc lập, chưa có người dùng thật (không phải BUSINESS_VALIDATED).

Đây là pack thứ hai có SỐ LIỆU, khác hình dạng pack tham chiếu ở `tenants/` (bậc B của báo cáo audit vNext §06.3): phân cấp
**khách hàng → đơn hàng** thay cho dự án → thiết bị, đơn vị thùng và tiền thay cho lít. Không đọc gì của `tenants/`
(`go list -deps -test ./examples/02-sme-minimal/...` không có gói nào dưới `tenants/`).

## Có gì

- `pack/`: pack của một nhà phân phối **hư cấu** (`sme-mau`).
  - `sources.yaml` khai lược đồ danh mục (`entity_lists`: mảng `khach_hang`, mảng `don_hang` trỏ khách qua trường `khach`).
  - `metrics.yaml`: hai chỉ số số theo ngày của một đơn — số thùng xuất kho (phiếu xuất), doanh thu ghi nhận (sổ bán hàng).
  - `intents.yaml`: hai ý định số liệu; `narrow_parent` cho router gỡ bí danh mơ hồ theo khách hàng đã nêu trong câu.
  - `policy.yaml`: hai vai — `quan_ly` xem mọi khách, `nhan_vien` chỉ xem khách trong `entity_scope` của mình.
- `data/sme-mau/`: dữ liệu **tự viết, giả lập** — hai khách, bốn đơn, hai bảng CSV. Không lấy từ doanh nghiệp thật nào.
  Có một dòng phiếu trỏ mã đơn không tồn tại để thấy bản ghi vào cách ly.
- `sme_test.go`:
  - `TestNapDuLieuSME`: nạp qua `data/ingestion` theo `sources.yaml` → 8 quan sát + 1 cách ly.
  - `TestLuongHoiSME`: ba câu số liệu (theo đơn, theo khách, doanh thu), một câu hỏi lại ("đơn 7" của hai khách), câu cùng bí
    danh mà nêu khách thì được gỡ theo kiểu cha `khach_hang`, và vai phạm vi hẹp bị từ chối đơn của khách khác.

## Chạy

```sh
go test -count=1 ./examples/02-sme-minimal/
```

## Giới hạn đã biết

- Danh từ đếm / nhãn của bộ chuẩn hoá ("máy", "xe", "chuyến", "lít", "khối"…) vẫn viết trong mã dùng chung
  (`data/normalization/qualifier.go`, F-030), chưa đọc từ `lexicon.yaml`; pack này không cần chúng để chạy.
- Chưa có đường PostgreSQL riêng cho ví dụ này; đường nạp PG của `data/ingestion` đã có test với lược đồ khác construction.
