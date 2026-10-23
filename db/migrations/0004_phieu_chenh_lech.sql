-- 0004 — phiếu kiểm tra chênh lệch (Phase 1, khối M4 của MVP; HOP-DONG §7.4, ADR-0011 D4). Ba bảng:
--   discrepancy_tickets  phiếu: một dòng cho mỗi lần tạo thành công; Phase 1 chỉ có trạng thái OPEN;
--   commands             sổ lệnh: mỗi (tenant, người gọi, command_id) tối đa MỘT lần ghi, trỏ tới phiếu của lần ghi đó;
--   audit_log            audit nghiệp vụ chỉ-thêm: EXECUTED (cùng giao dịch với phiếu và sổ lệnh), REPLAYED, REFUSED, VERIFY_FAILED.
-- Cách ly tenant y hệt 0001 / 0003: tenant_id nằm trong khoá, ENABLE + FORCE RLS, chính sách tenant_isolation đọc GUC eais.tenant_id
-- (GUC chưa đặt ⇒ 0 dòng, đóng mặc định). Vai ứng dụng eais_app CHỈ có SELECT + INSERT (không UPDATE, DELETE, TRUNCATE): tính chỉ-thêm
-- do quyền bảo đảm, không do thói quen (bất biến I8). Dọn theo thời hạn lưu là việc của vai quản trị (Phase 5).
--
-- Chống trùng hai tầng (HOP-DONG §6) do chính cơ sở dữ liệu bảo đảm, không do đọc-rồi-ghi:
--   tầng lệnh      khoá chính của sổ lệnh (tenant_id, caller_id, command_id);
--   tầng nghiệp vụ chỉ mục duy nhất TỪNG PHẦN của phiếu (tenant_id, subject_key, metric, day) WHERE status = 'OPEN'.
-- Khoá nghiệp vụ dùng KHOÁ thực thể (subject_key), không dùng mã thực thể: Load thay toàn bộ danh mục nên mã có thể đổi.
--
-- Khoá ngoại: sổ lệnh → phiếu là khoá ngoại THƯỜNG (không DEFERRABLE), nên đơn vị ghi phải INSERT phiếu trước rồi mới sổ lệnh. Phiếu
-- không có khoá ngoại ngược tới sổ lệnh (sẽ thành vòng tròn). KHÔNG khoá ngoại nào tới entities, entity_aliases, facts, documents:
-- Load xoá và nạp lại bốn bảng đó trong một giao dịch, khoá ngoại sẽ làm Load hỏng hoặc phiếu mồ côi. Phiếu giữ khoá và mã thực thể
-- dạng chữ cùng bản chụp bằng chứng (evidence).
--
-- Số lượng và chênh lệch lưu kiểu numeric (không float: số thập phân giữ nguyên chữ số), ngày kiểu date. created_at / at do đồng hồ
-- của cơ sở dữ liệu gán (now() của giao dịch), bên gọi không đưa.
-- Chạy lại được: CREATE ... IF NOT EXISTS, chính sách DROP rồi CREATE, quyền REVOKE rồi GRANT.

CREATE TABLE IF NOT EXISTS discrepancy_tickets (
  tenant_id    text        NOT NULL,
  id           text        NOT NULL,                  -- mã phiếu do hệ sinh
  status       text        NOT NULL,
  entity_id    text        NOT NULL,                  -- mã thực thể trong danh mục LÚC TÍNH; không khoá ngoại
  subject_key  text        NOT NULL,                  -- khoá thực thể (khoá nghiệp vụ chống trùng)
  metric       text        NOT NULL,
  day          date        NOT NULL,
  left_metric  text        NOT NULL,
  left_value   numeric     NOT NULL,
  left_unit    text        NOT NULL,
  right_metric text        NOT NULL,
  right_value  numeric     NOT NULL,
  right_unit   text        NOT NULL,
  diff         numeric     NOT NULL,
  approx       boolean     NOT NULL,
  formula      text        NOT NULL,
  evidence     jsonb       NOT NULL,                  -- bản chụp bằng chứng của Derive, MẢNG theo thứ tự lúc ghi
  created_by   text        NOT NULL,                  -- Principal.ID
  created_at   timestamptz NOT NULL DEFAULT now(),
  pack_version text        NOT NULL,
  code_version text        NOT NULL,
  command_id   text        NOT NULL,                  -- cột thường, không khoá ngoại tới sổ lệnh
  PRIMARY KEY (tenant_id, id),
  CHECK (status = 'OPEN'),
  CHECK (id <> '' AND subject_key <> '' AND metric <> '' AND created_by <> '' AND command_id <> ''),
  CHECK (jsonb_typeof(evidence) = 'array')
);

-- Tầng nghiệp vụ: tối đa một phiếu OPEN cho mỗi (tenant, khoá thực thể, chỉ số, ngày).
CREATE UNIQUE INDEX IF NOT EXISTS discrepancy_tickets_open_key
  ON discrepancy_tickets (tenant_id, subject_key, metric, day) WHERE status = 'OPEN';

CREATE TABLE IF NOT EXISTS commands (
  tenant_id   text        NOT NULL,
  caller_id   text        NOT NULL,                   -- Principal.ID của người gọi
  command_id  text        NOT NULL,                   -- UUID chữ thường do bên gọi sinh, mỗi ý định một lần
  fingerprint text        NOT NULL,                   -- sha256 của {subject_key, metric, day}: cùng khoá khác nội dung thì từ chối
  ticket_id   text        NOT NULL,
  created_at  timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY (tenant_id, caller_id, command_id),
  FOREIGN KEY (tenant_id, ticket_id) REFERENCES discrepancy_tickets (tenant_id, id),
  CHECK (caller_id <> '' AND command_id <> ''),
  CHECK (fingerprint ~ '^[0-9a-f]{64}$')
);

CREATE TABLE IF NOT EXISTS audit_log (
  tenant_id     text        NOT NULL,
  id            bigint      GENERATED ALWAYS AS IDENTITY, -- thứ tự ghi trong tenant; có khe hở khi giao dịch huỷ
  at            timestamptz NOT NULL DEFAULT now(),
  outcome       text        NOT NULL,
  reason_code   text        NOT NULL DEFAULT '',
  reason_detail text        NOT NULL DEFAULT '',          -- lý do THẬT (vd phân biệt "không tồn tại" với "ngoài phạm vi"); chỉ nằm ở đây
  caller_id     text        NOT NULL,
  caller_roles  text[]      NOT NULL DEFAULT '{}',
  entity_scope  text        NOT NULL DEFAULT '',
  command_id    text        NOT NULL,
  action        text        NOT NULL,
  subject_key   text        NOT NULL,
  metric        text        NOT NULL,
  day           date        NOT NULL,
  ticket_id     text        NOT NULL DEFAULT '',          -- rỗng khi lượt không có phiếu (vd từ chối)
  fingerprint   text        NOT NULL,
  pack_version  text        NOT NULL,
  code_version  text        NOT NULL,
  trace_id      text        NOT NULL DEFAULT '',
  PRIMARY KEY (tenant_id, id),
  CHECK (outcome IN ('EXECUTED', 'REPLAYED', 'REFUSED', 'VERIFY_FAILED')),
  CHECK (outcome <> 'EXECUTED' OR ticket_id <> ''),       -- EXECUTED luôn đi cùng phiếu của chính đơn vị ghi
  CHECK (fingerprint ~ '^[0-9a-f]{64}$')
);

-- Đọc lại sau ghi và tra audit theo lệnh: (tenant, người gọi, command_id).
CREATE INDEX IF NOT EXISTS audit_log_by_command ON audit_log (tenant_id, caller_id, command_id);

DO $$
DECLARE t text;
BEGIN
  FOREACH t IN ARRAY ARRAY['discrepancy_tickets', 'commands', 'audit_log'] LOOP
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
