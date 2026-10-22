-- 0003 — kho vết bền theo tenant (làn 0, khối M1 của MVP). Ba bảng:
--   decision_records   bản ghi quyết định của MỌI lượt hỏi, kể cả lượt lỗi (fault + thông điệp đã làm sạch);
--   quarantine_records bản ghi cách ly của một lần nạp, luôn có lý do (CHECK reason <> '');
--   usage_records      usage model theo lượt (mặc định không gọi model nên thường vắng).
-- Cách ly tenant y hệt 0001 / 0002: tenant_id nằm trong khoá chính, ENABLE + FORCE RLS, chính sách tenant_isolation đọc GUC
-- eais.tenant_id (GUC chưa đặt ⇒ 0 dòng, đóng mặc định). Vai ứng dụng eais_app CHỈ có SELECT + INSERT: vết là bằng chứng, ghi
-- một lần, không sửa, không xoá — dọn theo thời hạn lưu là việc của vai quản trị.
-- Ghi lại cùng khoá không nhân đôi: ứng dụng dùng INSERT ... ON CONFLICT DO NOTHING rồi so băm nội dung — y nguyên thì bỏ qua
-- (có báo), khác nội dung thì lỗi (xem platform/pgstore/vet.go).
-- Chạy lại được: CREATE ... IF NOT EXISTS, chính sách DROP rồi CREATE, quyền REVOKE rồi GRANT — chạy lần hai (kể cả khi
-- schema_migrations bị mất dòng) không lỗi và đưa về đúng một trạng thái.

CREATE TABLE IF NOT EXISTS decision_records (
  tenant_id      text        NOT NULL,
  trace_id       text        NOT NULL,
  recorded_at    timestamptz NOT NULL DEFAULT now(),
  asked_at       timestamptz NOT NULL, -- đồng hồ của lượt (Decision.Now), không phải giờ ghi
  principal_id   text        NOT NULL,
  principal_kind text        NOT NULL DEFAULT '',
  question       text        NOT NULL,
  intent         text        NOT NULL DEFAULT '',
  route          text        NOT NULL DEFAULT '',
  outcome        text        NOT NULL DEFAULT '', -- rỗng ở lượt lỗi: luồng hỏi không trả Outcome khi lỗi
  verdict        text        NOT NULL DEFAULT '',
  answer         text        NOT NULL DEFAULT '',
  fault          text        NOT NULL DEFAULT '', -- nhãn lỗi (input / policy / data / code / infra:… / third_party:…); rỗng = lượt không lỗi
  error_message  text        NOT NULL DEFAULT '', -- đã che bí mật và cắt độ dài
  pack_version   text        NOT NULL DEFAULT '',
  code_version   text        NOT NULL DEFAULT '',
  data_version   text        NOT NULL DEFAULT '',
  total_ms       bigint      NOT NULL DEFAULT 0,
  unaccounted_ms bigint      NOT NULL DEFAULT 0,
  decision       jsonb       NOT NULL, -- traces.Decision đầy đủ: nguồn thật của bản ghi, các cột phẳng ở trên chỉ để lọc / báo cáo
  steps          jsonb       NOT NULL DEFAULT '[]', -- []traces.Step: vết từng bước S0…S10
  spans          jsonb       NOT NULL DEFAULT '[]', -- []traces.Span: thời gian từng chặng
  content_sha256 text        NOT NULL, -- băm nội dung bản ghi đã làm sạch: phân biệt ghi lại y nguyên với đụng trace_id
  PRIMARY KEY (tenant_id, trace_id),
  CHECK (trace_id <> ''),
  CHECK (jsonb_typeof(decision) = 'object' AND jsonb_typeof(steps) = 'array' AND jsonb_typeof(spans) = 'array'),
  CHECK (content_sha256 ~ '^[0-9a-f]{64}$')
);

-- Bản ghi bị cách ly của một lần nạp. Danh tính = (lần nạp, khoá bản ghi, lý do): một bản ghi cách ly vì hai lý do là hai dòng.
CREATE TABLE IF NOT EXISTS quarantine_records (
  tenant_id   text        NOT NULL,
  load_id     text        NOT NULL,
  record_key  text        NOT NULL, -- "<tệp>#<mã bản ghi | số dòng>", cùng khoá với manifest của bộ sinh
  reason      text        NOT NULL,
  source      text        NOT NULL DEFAULT '',
  detail      text        NOT NULL DEFAULT '',
  recorded_at timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY (tenant_id, load_id, record_key, reason),
  CHECK (load_id <> '' AND record_key <> '' AND reason <> '')
);

-- Usage theo (lượt, model). Khoá ngoại về decision_records: mọi chi phí truy được về một lượt có vết (P8), nên lưu bản ghi quyết
-- định TRƯỚC rồi mới lưu usage.
CREATE TABLE IF NOT EXISTS usage_records (
  tenant_id     text        NOT NULL,
  trace_id      text        NOT NULL,
  model         text        NOT NULL,
  calls         int         NOT NULL,
  input_tokens  bigint      NOT NULL,
  output_tokens bigint      NOT NULL,
  cache_read    bigint      NOT NULL,
  cache_write   bigint      NOT NULL,
  recorded_at   timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY (tenant_id, trace_id, model),
  FOREIGN KEY (tenant_id, trace_id) REFERENCES decision_records (tenant_id, trace_id),
  CHECK (model <> ''),
  CHECK (calls >= 0 AND input_tokens >= 0 AND output_tokens >= 0 AND cache_read >= 0 AND cache_write >= 0)
);

DO $$
DECLARE t text;
BEGIN
  FOREACH t IN ARRAY ARRAY['decision_records', 'quarantine_records', 'usage_records'] LOOP
    EXECUTE format('ALTER TABLE %I ENABLE ROW LEVEL SECURITY', t);
    EXECUTE format('ALTER TABLE %I FORCE ROW LEVEL SECURITY', t);
    EXECUTE format('DROP POLICY IF EXISTS tenant_isolation ON %I', t);
    EXECUTE format($p$CREATE POLICY tenant_isolation ON %I
      USING (tenant_id = current_setting('eais.tenant_id', true))
      WITH CHECK (tenant_id = current_setting('eais.tenant_id', true))$p$, t);
    EXECUTE format('REVOKE ALL ON %I FROM PUBLIC', t);
    EXECUTE format('REVOKE ALL ON %I FROM eais_app', t);
    EXECUTE format('GRANT SELECT, INSERT ON %I TO eais_app', t);
  END LOOP;
END
$$;
