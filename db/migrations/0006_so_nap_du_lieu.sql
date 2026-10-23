-- 0006 — sổ nạp: phiên bản dữ liệu của MỖI lần nạp đã commit, theo tenant (việc 5e của lệnh ca chiều 23/10; README, R2-01).
-- Trước đây phiên bản dữ liệu ghi vào vết là băm thư mục trên đĩa, tính lúc server khởi động và sau mỗi POST /v1/ingest: không gì bảo đảm thư
-- mục đó là thứ đã nạp vào PG (thư mục đổi sau lần nạp, serve trỏ nhầm thư mục, `eais ingest` chạy khi server đang chạy). Giờ pgstore.Load ghi
-- MỘT dòng vào bảng này TRONG CÙNG giao dịch thay dữ liệu của tenant (dưới cùng khoá tư vấn theo tenant), nên "dữ liệu của tenant" và "phiên
-- bản của dữ liệu đó" cùng commit hoặc cùng huỷ; server đọc dòng mới nhất của tenant mỗi lượt (pgstore.LatestLoad).
--   load_records   chỉ-thêm, một dòng mỗi lần nạp: id (thứ tự nạp của tenant), mã lần nạp, phiên bản dữ liệu, giờ của cơ sở dữ liệu.
-- Dòng có data_version = '' (và load_id = '') là lần nạp KHÔNG kèm phiên bản (đường Load cũ, ví dụ bộ chạy winrun và test): vẫn phải có một dòng,
-- vì dữ liệu đã bị thay và dòng cũ không còn mô tả nó — "không biết" phải đọc ra là "không biết", không phải phiên bản của lần nạp trước. Ngoài hai
-- trường hợp đó, data_version là băm 64 hex (cùng dạng với băm thư mục dữ liệu của ứng dụng).
-- id cấp lúc INSERT, mà INSERT nằm sau khoá tư vấn theo tenant của Load (nhả khi giao dịch kết thúc): các lần nạp CÙNG tenant xếp hàng nên id tăng
-- đúng theo thứ tự commit của tenant đó; có khe hở khi giao dịch huỷ.
-- Cách ly tenant y hệt 0003 / 0004: tenant_id nằm trong khoá, ENABLE + FORCE RLS, chính sách tenant_isolation đọc GUC eais.tenant_id (GUC chưa đặt
-- ⇒ 0 dòng, đóng mặc định); tenant rỗng bị chặn ngay ở lược đồ (như 0005, C3). Vai ứng dụng eais_app CHỈ có SELECT + INSERT: sổ là bằng chứng,
-- không sửa, không xoá — dọn theo thời hạn lưu là việc của vai quản trị.
-- Chạy lại được: CREATE ... IF NOT EXISTS, chính sách DROP rồi CREATE, quyền REVOKE rồi GRANT. Tên tệp được ghim ở ba test: platform/pgstore/
-- migrate_test.go, platform/pgstore/vet_test.go và platform/pgstore/so_nap_test.go — đổi số migration (luật 23/10: bên gộp main trước lấy số) thì
-- sửa cả ba.

CREATE TABLE IF NOT EXISTS load_records (
  tenant_id    text        NOT NULL,
  id           bigint      GENERATED ALWAYS AS IDENTITY,
  load_id      text        NOT NULL,
  data_version text        NOT NULL,
  loaded_at    timestamptz NOT NULL DEFAULT now(), -- đồng hồ của cơ sở dữ liệu (now() của giao dịch nạp), bên gọi không đưa
  PRIMARY KEY (tenant_id, id),
  CONSTRAINT load_records_tenant_khong_rong CHECK (tenant_id <> ''),
  CHECK (data_version = '' OR data_version ~ '^[0-9a-f]{64}$'),
  CHECK ((data_version = '') = (load_id = ''))
);

ALTER TABLE load_records ENABLE ROW LEVEL SECURITY;
ALTER TABLE load_records FORCE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS tenant_isolation ON load_records;
CREATE POLICY tenant_isolation ON load_records
  USING (tenant_id = current_setting('eais.tenant_id', true))
  WITH CHECK (tenant_id = current_setting('eais.tenant_id', true));
REVOKE ALL ON load_records FROM PUBLIC;
REVOKE ALL ON load_records FROM eais_app;
GRANT SELECT, INSERT ON load_records TO eais_app;
