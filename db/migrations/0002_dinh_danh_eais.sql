-- 0002 — định danh chạy theo tên EAIS (D-RM-02, tối 21/10/2026). 0001 có thể đã áp ở DB đang dùng nên không sửa; tệp này đưa
-- mọi DB (đã áp 0001 từ trước, hay DB mới chạy 0001 rồi 0002) về cùng một trạng thái:
-- - vai ứng dụng eais_app, cùng quyền SELECT / INSERT / DELETE như vai cũ. Vai chưa có thì tạo NOSUPERUSER NOBYPASSRLS; vai là của cả
--   cụm PG nên vai có sẵn thì 0002 không đổi thuộc tính — chốt chặn là pgstore.Open, từ chối vai vượt RLS (superuser hoặc BYPASSRLS);
-- - chính sách tenant_isolation đọc GUC eais.tenant_id. GUC tên cũ (eacl.tenant_id) không còn mở dòng nào;
-- - vai cũ eacl_app (0001 tạo; vai là của cả cụm PG nên không xoá ở đây) mất mọi quyền trên bảng nghiệp vụ của DB này.

DO $$
BEGIN
  IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'eais_app') THEN
    CREATE ROLE eais_app LOGIN NOSUPERUSER NOBYPASSRLS;
  END IF;
END
$$;

DO $$
DECLARE t text;
BEGIN
  FOREACH t IN ARRAY ARRAY['entities', 'entity_aliases', 'facts', 'documents'] LOOP
    EXECUTE format($p$ALTER POLICY tenant_isolation ON %I
      USING (tenant_id = current_setting('eais.tenant_id', true))
      WITH CHECK (tenant_id = current_setting('eais.tenant_id', true))$p$, t);
    EXECUTE format('GRANT SELECT, INSERT, DELETE ON %I TO eais_app', t);
    IF EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'eacl_app') THEN
      EXECUTE format('REVOKE ALL ON %I FROM eacl_app', t);
    END IF;
  END LOOP;
END
$$;
