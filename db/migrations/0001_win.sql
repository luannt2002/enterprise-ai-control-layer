-- 0001 — lược đồ dữ liệu WIN.
-- Mọi bảng nghiệp vụ có tenant_id và bật RLS FORCE: một dòng chỉ thấy / ghi được khi
-- tenant_id = GUC eais.tenant_id của giao dịch. Ứng dụng đặt GUC bằng SET LOCAL từ Principal đã
-- xác thực, không từ thân yêu cầu. GUC chưa đặt ⇒ current_setting trả NULL ⇒ 0 dòng (đóng mặc định).
-- Ứng dụng chạy bằng vai eais_app (không superuser, không BYPASSRLS): superuser bỏ qua RLS.

DO $$
BEGIN
  IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'eais_app') THEN
    CREATE ROLE eais_app LOGIN NOSUPERUSER NOBYPASSRLS;
  END IF;
END
$$;

CREATE TABLE entities (
  tenant_id text NOT NULL,
  id        text NOT NULL,
  type      text NOT NULL,
  key       text NOT NULL,
  name      text NOT NULL,
  parent_id text,
  ord       int  NOT NULL, -- thứ tự nạp: danh mục trả đúng thứ tự như bản trong bộ nhớ
  PRIMARY KEY (tenant_id, id)
);

CREATE TABLE entity_aliases (
  tenant_id text NOT NULL,
  entity_id text NOT NULL,
  alias     text NOT NULL,
  source    text NOT NULL DEFAULT '',
  ord       int  NOT NULL,
  FOREIGN KEY (tenant_id, entity_id) REFERENCES entities (tenant_id, id)
);

-- Một lần đo đã chuẩn hoá: chỉ số ngữ nghĩa + thực thể + ngày + giá trị ở đơn vị chuẩn.
CREATE TABLE facts (
  tenant_id   text        NOT NULL,
  id          text        NOT NULL,
  metric      text        NOT NULL,
  entity_id   text        NOT NULL,
  occurred_on date        NOT NULL,
  value       numeric     NOT NULL,
  unit        text        NOT NULL,
  approx      boolean     NOT NULL DEFAULT false,
  source_type text        NOT NULL,
  source_ref  text        NOT NULL DEFAULT '',
  raw         text        NOT NULL DEFAULT '',
  observed_at timestamptz,
  reported_by text        NOT NULL DEFAULT '',
  ord         bigint      NOT NULL,
  PRIMARY KEY (tenant_id, id)
);
CREATE INDEX facts_lookup ON facts (tenant_id, metric, entity_id, occurred_on);

-- Tài liệu tri thức: acl = các vai được đọc. Truy xuất lọc acl trong SQL, tenant bằng RLS.
CREATE TABLE documents (
  tenant_id text   NOT NULL,
  id        text   NOT NULL,
  title     text   NOT NULL,
  body      text   NOT NULL,
  acl       text[] NOT NULL DEFAULT '{}',
  PRIMARY KEY (tenant_id, id)
);

DO $$
DECLARE t text;
BEGIN
  FOREACH t IN ARRAY ARRAY['entities', 'entity_aliases', 'facts', 'documents'] LOOP
    EXECUTE format('ALTER TABLE %I ENABLE ROW LEVEL SECURITY', t);
    EXECUTE format('ALTER TABLE %I FORCE ROW LEVEL SECURITY', t);
    EXECUTE format($p$CREATE POLICY tenant_isolation ON %I
      USING (tenant_id = current_setting('eais.tenant_id', true))
      WITH CHECK (tenant_id = current_setting('eais.tenant_id', true))$p$, t);
    EXECUTE format('GRANT SELECT, INSERT, DELETE ON %I TO eais_app', t);
  END LOOP;
END
$$;
