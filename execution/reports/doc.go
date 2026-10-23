// Package reports là kho bản ghi báo cáo ngày của đội hiện trường (Phase 0a, khối M2 của MVP): HOP-DONG 0a §3, §5, §14.2 bước 2 và 3,
// ADR-0012, quyết định Q-0A-12 = b (gói MỚI dưới execution/, không sửa data/ingestion hay tệp cũ của platform/pgstore).
//
// Gói làm HAI việc:
//
//   - bước 2: kho bản ghi (mỗi đơn vị một bản ghi hiện hành kèm lịch sử), sổ lần giao chỉ thêm (nơi DUY NHẤT của DUPLICATE), biên
//     nhận lô, bốn kết cục ACCEPTED / HELD / QUARANTINED / DUPLICATE, danh sách đóng mã lý do (§3.4), hai kho (bộ nhớ cho test, PG);
//   - bước 3: khử trùng ba mức TR1 (khoá nguồn), TR2 (vân tay), TR3 (nghi trùng trong cửa sổ W): thứ tự chuẩn (sent_at, unit_key),
//     đánh giá lại chỉ-siết, khoá tuần tự hoá theo (tenant, thực thể, ngày xảy ra), quyết định của người, bản rời nhóm.
//
// # Bất biến theo SỰ KIỆN (R6-01, R6-02)
//
// Giấy khoá ở vòng 6 để lại hai nợ có mã, mã này chặn bằng test đỏ: mọi sự kiện làm đổi kho (nạp, quyết định, thu hồi, sửa) đều kết
// thúc bằng MỘT bước settle (tr3.go) chạy tới điểm bất động trong CÙNG giao dịch. Bước đó không biết sự kiện nào vừa xảy ra; nó chỉ
// thấy kho và giữ ba điều: (1) hai bản liền kề trong nhóm, cách ≤ W, người chưa xét cặp thì bản đứng sau không ACCEPTED; (2) bản
// HELD possible_duplicate luôn trỏ đúng bản đứng ngay trước nó; (3) bản REJECTED bởi MARK_DUPLICATE mà bản được giữ của nó thôi được
// tính (vì BẤT KỲ lý do gì) thì quay lại HELD possible_duplicate. Vì kiểm theo trạng thái chứ không theo đường đi, một đường rời nhóm
// mới thêm sau này không cần nhớ gọi thêm việc nào.
//
// # Cố ý KHÔNG làm ở đây (giao diện cho các bước sau)
//
//   - cờ loại kind / approx từ chữ (bước 4), ngày xảy ra đọc từ chữ (bước 5), quyền người gửi, danh bạ, capability reports.decide
//     (bước 6): gói nhận Item đã có Verdict (kết cục luật đọc đề xuất) và không tự đọc chữ; Command không kiểm quyền người quyết;
//   - phục hồi (bước 7): Retract và Amend là nguyên thủy ÁP DỤNG, không kiểm người gửi, không tạo dấu thu hồi, không tạo phiên bản
//     chờ khi bản đã có quyết định của người (BB7); lớp phục hồi gọi chúng sau khi đã kiểm, kể cả khi người APPLY;
//   - báo cáo xuất (bước 8), nhánh form (bước 9), nối POST /v1/ingest và phép chiếu sang bảng facts (U2, việc của làn 0);
//   - DISMISS và REPROCESS của bản QUARANTINED (chưa có trạng thái đích: U14 (b); REPROCESS cần luật đọc).
package reports
