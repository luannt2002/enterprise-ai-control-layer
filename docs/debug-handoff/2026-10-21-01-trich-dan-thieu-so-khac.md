# Handoff 2026-10-21-01: số "các số khác cùng kỳ" không có trong trích dẫn và bản ghi quyết định

**Từ:** phiên điều phối + debug, debug vòng 1 (trace câu chuẩn S0 → S10). **Mức:** THẤP: không sai số, chỉ thiếu truy vết.

```text
Revision: ac98cf4 (code giống 8280f40)
Scenario: "Hôm qua máy 03 được cấp 4k dầu đúng không?" — u-director, đồng hồ 20/10/2026 10:00,
          dữ liệu sinh seed 42 (manifest 31172355…), đường bộ nhớ, model giả tất định
Expected: mọi con số trong câu trả lời truy được về nguồn qua resp.Citations và Decision.Evidence
          (CLAUDE.md P8 "mọi kết quả quan trọng truy được").
Actual:   câu trả lời có 3 số "Các số khác cùng kỳ": hoá đơn 4.100 L, tiêu thụ 3.782 L, đội báo 4.000 L.
          Decision.Evidence chỉ có: fact:erp.json#R-005, fact:warehouse.csv#R-004, calc:issued_fuel.

First wrong layer: S10 / ghi vết. Câu trả lời đúng; S9 CÓ kiểm 3 số này.
Cause evidence [CODE]:
  - ask.go:336-340  resp.Citations chỉ lấy từ result.Evidence (chỉ số chính)
  - ask.go:443-449  bằng chứng của otherMetrics chỉ được thêm vào pkg (gói kiểm của S9)
  - ask.go:266-268  Decision.Evidence lấy từ resp.Citations
Detection failure: không có — S9 kiểm đúng; chỉ lớp vết thiếu.
Invariant violated: P8 (truy vết đủ); Replay / Diff so Evidence nên cũng không thấy phần thiếu.

Reproducer (chỉ đọc): dựng pipeline giống winrun (tenant.Load → ingestion.Ingest → pipeline.Ask),
  gọi Handle với câu trên, in out.Decision; so các số trong Answer với Decision.Evidence.
Negative case: câu tra số thường (không phải câu xác nhận, không có "các số khác") → không lệch.

Confidence: CONFIRMED (đọc code + vết đo bằng chương trình thăm dò ngoài repo).
Suggested BUILD investigation: thêm bằng chứng của otherMetrics vào citations (và vì vậy vào Decision.Evidence);
  xét Replay có cần so phần này không; thêm test khẳng định mọi số trong câu xác nhận có id trong citations.
```

## Trạng thái BUILD

(BUILD điền: NHẬN / ĐANG LÀM / XONG `<commit>` / BÁC: `<lý do + lệnh>` / BLOCKED_FOR_OWNER — kèm ngày)
