// Package reports là kho bản ghi báo cáo ngày của đội hiện trường (Phase 0a, khối M2 của MVP): HOP-DONG 0a §3, §5, §14.2 bước 2 và 3,
// ADR-0012, quyết định Q-0A-12 = b (gói MỚI dưới execution/, không sửa data/ingestion hay tệp cũ của platform/pgstore).
//
// Gói làm BA việc:
//
//   - bước 2: kho bản ghi (mỗi đơn vị một bản ghi hiện hành kèm lịch sử), sổ lần giao chỉ thêm (nơi DUY NHẤT của DUPLICATE), biên
//     nhận lô, bốn kết cục ACCEPTED / HELD / QUARANTINED / DUPLICATE, danh sách đóng mã lý do (§3.4), hai kho (bộ nhớ cho test, PG);
//   - bước 3: khử trùng ba mức TR1 (khoá nguồn), TR2 (vân tay), TR3 (nghi trùng trong cửa sổ W): thứ tự chuẩn (sent_at, unit_key),
//     đánh giá lại chỉ-siết, khoá tuần tự hoá theo (tenant, thực thể, ngày xảy ra), quyết định của người, bản rời nhóm;
//   - kho PostgreSQL cho cả bước 2 và 3, cùng engine với kho bộ nhớ (pg.go).
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
// # Hai kho, một engine
//
// Memory (test) và PG (PostgreSQL, migration kho_bao_cao_ngay) cùng chạy engine.go, tr3.go, decide.go: mỗi kho chỉ cài giao diện txn (đọc, khoá, ghi từng bảng), nên hai kho không thể lệch nhau về
// luật; test chạy cùng một bộ trên cả hai (forEachStore) và so khớp từng trường trên chuỗi sự kiện ngẫu nhiên. Mỗi thao tác là MỘT giao dịch (khoá, đọc, tính, ghi một lượt): lỗi ở đâu cũng không để
// lại nửa đơn vị. Khoá tuần tự hoá TR3 là khoá tư vấn mức giao dịch theo (tenant, thực thể, ngày xảy ra) lấy TRƯỚC khi đọc nhóm, theo thứ tự khoá tăng dần (giao dịch chạm hai ô và cần khoá nhỏ hơn thì
// bị bỏ và chạy lại với mọi khoá đã cần lấy sẵn). Quyết định, thu hồi và sửa tin lấy thêm khoá tư vấn của CHÍNH bản ghi (tenant, unit_key) trong cùng thứ tự đó, thay cho khoá hàng: không giao dịch nào
// giữ khoá hàng trong lúc chờ khoá tư vấn (khoá hàng ngầm của UPDATE và của khoá ngoại chỉ đến lúc ghi, sau khoá tư vấn cuối cùng), nên không có chuỗi chờ vòng. Bản cũ giữ FOR UPDATE rồi còn xin
// khoá ô đích và chết khoá 40P01 dưới tải hỗn hợp (R-01, soát 23/10; test TestPG_R01_*). Va chạm khoá duy nhất (hai giao dịch cùng tạo một unit_key ở hai ô khác nhau) cũng làm giao dịch
// chạy lại: bên thua thấy bản của bên thắng và đi theo TR1.
//
// # Giao diện cho các bước sau
//
//   - Retract và Amend là nguyên thủy ÁP DỤNG: không kiểm người gửi, không tạo dấu thu hồi, không tạo phiên bản chờ khi bản đã có quyết định của người (BB7). Bước 7 gọi chúng sau khi đã kiểm, kể cả khi
//     người APPLY một phiên bản chờ; bản bị bác theo bản được giữ quay lại HELD trong cùng giao dịch dù bước 7 áp bằng đường nào, vì settle kiểm theo trạng thái. Hai nguyên thủy từ chối (không đổi gì)
//     khi bản đã RETRACTED hoặc SUPERSEDED, khi Amend mang nội dung không đổi, và khi thiếu người thực hiện.
//   - Decide chưa kiểm capability reports.decide, phạm vi thực thể của người quyết (bước 6) và chưa idempotent theo mã quyết định (Command chưa có trường đó); chưa có DISMISS, REPROCESS (U14 (b)), APPLY cho
//     id_conflict (cần phiên bản chờ, bước 7). Lệnh bị từ chối ghi một dòng audit EventRefused (kể cả khi thiếu người quyết) mà không đổi bản ghi; giá trị người gửi lệnh cấp được cắt ở auditClip ký tự (U14).
//   - Cặp đã xét (KEEP_BOTH) là về nội dung lúc xét: khi nội dung của một trong hai bản đổi nhóm hoặc giờ gửi thì cặp bị xoá và ghi EventPairReset (chỉ siết; hợp đồng không nói điều này, ghi ở báo cáo của đơn vị).
//   - Nối POST /v1/ingest, phép chiếu sang bảng facts, bước vết L1.* (làn 0, Q-0A-15) và danh sách lô nạp thư mục: chưa làm. Số của migration là số kế tiếp trên main lúc mở mã; đổi số khi gộp là đổi tên tệp và hai danh sách
//     ở platform/pgstore (migrate_test.go, vet_test.go).
//
// # Cố ý KHÔNG làm ở đây
//
//   - cờ loại kind / approx từ chữ (bước 4), ngày xảy ra đọc từ chữ (bước 5), quyền người gửi, danh bạ, capability reports.decide (bước 6): gói nhận Item đã có Verdict (kết cục luật đọc đề xuất) và không tự
//     đọc chữ; Command không kiểm quyền người quyết;
//   - phục hồi đầy đủ (bước 7: người gửi, dấu thu hồi, phiên bản chờ, so với phiên bản hiện hành), báo cáo xuất (bước 8), nhánh form (bước 9);
//   - DISMISS và REPROCESS của bản QUARANTINED (chưa có trạng thái đích: U14 (b); REPROCESS cần luật đọc).
package reports
