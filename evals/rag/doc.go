// Package ragcases chạy THẬT các ca tri thức của cases.yaml qua chỉ mục RAG (ROADMAP-RAG.md, nấc RAG-0):
// mỗi ca có bộ tài liệu mẫu + kỳ vọng ở tenants/construction/knowledge/ca/<mã ca>/. Chỉ kiểm phần TRUY HỒI
// (đoạn nào, bản nào, hay thông báo nào) — không gọi model, không đi qua pipeline.
package ragcases
