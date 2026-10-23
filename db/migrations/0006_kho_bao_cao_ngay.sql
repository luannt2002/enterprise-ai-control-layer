-- Kho bản ghi báo cáo ngày của đội hiện trường (Phase 0a, khối M2 của MVP; HOP-DONG 0a §3.2, §3.8, §5.6, §6; ADR-0012 D1). Bốn bảng, mỗi bảng theo tenant:
--   report_records   MỖI đơn vị một bản ghi hiện hành (khoá duy nhất (tenant, unit_key)): ACCEPTED, HELD, QUARANTINED và các trạng thái vòng đời SUPERSEDED, RETRACTED, REJECTED.
--                    DUPLICATE không bao giờ là trạng thái của bản ghi (§3.8). Đổi trạng thái theo quyết định của người, thu hồi, sửa tin, đánh giá lại TR3: nên eais_app được UPDATE,
--                    nhưng chỉ theo cột và KHÔNG được đổi tenant_id, unit_key;
--   report_versions  lịch sử phiên bản nội dung của bản ghi (chỉ thêm): băm nội dung chuẩn hoá cho TR1, vân tay cho TR2, nguyên văn từng byte, ảnh chụp JSON;
--   report_history   lịch sử sự kiện của bản ghi (chỉ thêm): ai, khi nào, vì sao, từ trạng thái nào sang trạng thái nào (audit, §8 mặt 8);
--   delivery_ledger  sổ lần giao (chỉ thêm): mỗi đơn vị của mỗi lô đúng một dòng, nơi DUY NHẤT của DUPLICATE; không duy nhất theo unit_key.
-- Cách ly tenant y hệt 0001 / 0003 / 0004: tenant_id nằm trong khoá, ENABLE + FORCE RLS, chính sách tenant_isolation đọc GUC eais.tenant_id (GUC chưa đặt ⇒ 0 dòng, đóng mặc định);
-- tenant_id rỗng bị chặn ở lược đồ. Vai ứng dụng eais_app chỉ có quyền cần thiết: SELECT + INSERT ở cả bốn bảng; thêm UPDATE theo cột ở report_records; không DELETE, TRUNCATE ở đâu cả.
--
-- Ràng buộc ở lược đồ là phòng thủ chiều sâu cho các bất biến của hợp đồng (đóng mặc định, kể cả khi một đường ghi khác bỏ qua tầng Go): mã lý do thuộc danh sách đóng đúng kết cục (§3.4, BB2),
-- HELD và QUARANTINED có chi tiết và người nhận, ACCEPTED đủ thực thể, ngày xảy ra, lượng, đơn vị và không mang lý do (BB6), bản REJECTED bởi MARK_DUPLICATE có bản được giữ, dòng sổ
-- khác ACCEPTED có mã lý do và chi tiết, dòng DUPLICATE mang đúng mã duplicate_*, dòng sổ trỏ về bản ghi có thật (khoá ngoại: BB1).
--
-- Thời điểm (sent_at, decided_at, at, recorded_at) do bên gọi đưa (đồng hồ của hệ nhận, đặt được trong test), không do now() của cơ sở dữ liệu: cùng đầu vào cho cùng dữ liệu ở hai kho.
-- Lượng lưu dạng chuỗi thập phân chuẩn (core/decimal), ngày xảy ra kiểu date. Chạy lại được: CREATE ... IF NOT EXISTS, chính sách DROP rồi CREATE, quyền REVOKE rồi GRANT.

CREATE TABLE IF NOT EXISTS report_records (
  tenant_id      text        NOT NULL,
  unit_key       text        NOT NULL,
  channel        text        NOT NULL DEFAULT '',
  source_type    text        NOT NULL DEFAULT '',
  sender_id      text        NOT NULL DEFAULT '',
  sent_at        timestamptz,                         -- NULL: đơn vị không có giờ gửi (dòng tệp, hàng bảng): không vào nhóm TR3
  occurred_on    date,                                -- NULL: chưa đọc được ngày
  entity         text        NOT NULL DEFAULT '',
  metric         text        NOT NULL DEFAULT '',
  value_number   text        NOT NULL DEFAULT '',     -- chuỗi thập phân chuẩn; rỗng khi chưa đọc được lượng
  value_unit     text        NOT NULL DEFAULT '',
  approx         boolean     NOT NULL DEFAULT false,
  kind           text        NOT NULL,
  corrects       text        NOT NULL DEFAULT '',
  raw            bytea       NOT NULL,                -- nguyên văn, từng byte (BB4): kể cả UTF-8 hỏng
  attachments    jsonb       NOT NULL DEFAULT '[]',
  note           text        NOT NULL DEFAULT '',
  disposition    text        NOT NULL,
  reason         text        NOT NULL DEFAULT '',
  detail         text        NOT NULL DEFAULT '',
  recipient      text        NOT NULL DEFAULT '',
  duplicate_of   text        NOT NULL DEFAULT '',
  superseded_by  text        NOT NULL DEFAULT '',
  version        integer     NOT NULL,
  decided_by     text        NOT NULL DEFAULT '',
  decided_at     timestamptz,
  decision       text        NOT NULL DEFAULT '',
  reviewed_pairs text[]      NOT NULL DEFAULT '{}',   -- unit_key các bản kia mà người đã xét bằng KEEP_BOTH, cộng dồn (§5.6 mục 3)
  pack_version   text        NOT NULL DEFAULT '',
  trace_id       text        NOT NULL DEFAULT '',
  fingerprint    text        NOT NULL,
  content_sha256 text        NOT NULL,
  PRIMARY KEY (tenant_id, unit_key),
  CONSTRAINT report_records_tenant_khong_rong CHECK (tenant_id <> ''),
  CHECK (unit_key <> ''),
  CHECK (version >= 1),
  CHECK (kind IN ('OCCURRED', 'PLAN', 'CORRECTION')),
  CHECK (disposition IN ('ACCEPTED', 'HELD', 'QUARANTINED', 'SUPERSEDED', 'RETRACTED', 'REJECTED')),
  CHECK (decision IN ('', 'KEEP_BOTH', 'MARK_DUPLICATE', 'REJECT', 'CHOOSE', 'CORRECT', 'AUTHORIZE', 'APPLY')),
  CHECK (jsonb_typeof(attachments) = 'array'),
  -- mã lý do thuộc danh sách đóng (§3.4); hai mã DUPLICATE chỉ ở sổ lần giao, không bao giờ ở bản ghi
  CHECK (reason = '' OR reason IN ('missing_value', 'unknown_entity', 'ambiguous_entity', 'out_of_range', 'unit_dimension', 'unparseable', 'multiple_items',
    'number_reading_uncertain', 'unsupported_media', 'unrecognized_line', 'unreadable_file', 'unknown_sender', 'sender_not_authorized', 'date_conflict',
    'correction_unconfirmed', 'possible_duplicate', 'ambiguous_kind', 'id_conflict')),
  CHECK (disposition <> 'HELD' OR reason IN ('ambiguous_entity', 'number_reading_uncertain', 'sender_not_authorized', 'date_conflict', 'correction_unconfirmed',
    'possible_duplicate', 'ambiguous_kind', 'id_conflict')),
  CHECK (disposition <> 'QUARANTINED' OR reason IN ('missing_value', 'unknown_entity', 'out_of_range', 'unit_dimension', 'unparseable', 'multiple_items',
    'unsupported_media', 'unrecognized_line', 'unreadable_file', 'unknown_sender')),
  CHECK (disposition NOT IN ('HELD', 'QUARANTINED') OR (detail <> '' AND recipient <> '')),                       -- BB2, CL2, CL5
  CHECK (disposition <> 'ACCEPTED' OR (reason = '' AND entity <> '' AND occurred_on IS NOT NULL AND value_number <> '' AND value_unit <> '')),  -- BB6
  CHECK (disposition <> 'REJECTED' OR decision <> 'MARK_DUPLICATE' OR duplicate_of <> '')                        -- bản bị bác phải có bản được giữ (§6.2)
);

-- Đọc nhóm TR3 của một ô (thực thể, ngày xảy ra) dưới khoá tư vấn của ô đó.
CREATE INDEX IF NOT EXISTS report_records_slot ON report_records (tenant_id, entity, occurred_on);

CREATE TABLE IF NOT EXISTS report_versions (
  tenant_id      text    NOT NULL,
  unit_key       text    NOT NULL,
  version        integer NOT NULL,
  content_sha256 text    NOT NULL,
  fingerprint    text    NOT NULL,
  raw            bytea   NOT NULL,
  snapshot       jsonb   NOT NULL,
  PRIMARY KEY (tenant_id, unit_key, version),
  FOREIGN KEY (tenant_id, unit_key) REFERENCES report_records (tenant_id, unit_key),
  CONSTRAINT report_versions_tenant_khong_rong CHECK (tenant_id <> ''),
  CHECK (version >= 1 AND jsonb_typeof(snapshot) = 'object')
);

-- TR2: tìm bản ghi có phiên bản mang vân tay này.
CREATE INDEX IF NOT EXISTS report_versions_fingerprint ON report_versions (tenant_id, fingerprint);

CREATE TABLE IF NOT EXISTS report_history (
  tenant_id        text        NOT NULL,
  unit_key         text        NOT NULL,
  seq              integer     NOT NULL,
  at               timestamptz NOT NULL,
  actor            text        NOT NULL,
  event            text        NOT NULL,
  from_disposition text        NOT NULL DEFAULT '',
  to_disposition   text        NOT NULL,
  reason           text        NOT NULL DEFAULT '',
  detail           text        NOT NULL DEFAULT '',
  version          integer     NOT NULL,
  duplicate_of     text        NOT NULL DEFAULT '',
  decision         text        NOT NULL DEFAULT '',
  pack_version     text        NOT NULL DEFAULT '',
  PRIMARY KEY (tenant_id, unit_key, seq),
  FOREIGN KEY (tenant_id, unit_key) REFERENCES report_records (tenant_id, unit_key),
  CONSTRAINT report_history_tenant_khong_rong CHECK (tenant_id <> ''),
  CHECK (seq >= 1 AND actor <> '' AND event <> '')
);

CREATE TABLE IF NOT EXISTS delivery_ledger (
  tenant_id    text        NOT NULL,
  batch_id     text        NOT NULL,
  ordinal      integer     NOT NULL,
  unit_key     text        NOT NULL,                  -- unit_key của đơn vị ĐÃ GIAO (khác record_ref khi id_conflict: khoá có hậu tố băm)
  outcome      text        NOT NULL,                  -- kết cục LÚC XỬ LÝ; về sau đổi thì ghi ở lịch sử bản ghi, không sửa sổ
  reason       text        NOT NULL DEFAULT '',
  detail       text        NOT NULL DEFAULT '',
  record_ref   text        NOT NULL,
  pack_version text        NOT NULL DEFAULT '',
  trace_id     text        NOT NULL DEFAULT '',
  recorded_at  timestamptz NOT NULL,
  PRIMARY KEY (tenant_id, batch_id, ordinal),
  FOREIGN KEY (tenant_id, record_ref) REFERENCES report_records (tenant_id, unit_key),
  CONSTRAINT delivery_ledger_tenant_khong_rong CHECK (tenant_id <> ''),
  CHECK (ordinal >= 1 AND batch_id <> '' AND unit_key <> '' AND record_ref <> ''),
  CHECK (outcome IN ('ACCEPTED', 'HELD', 'QUARANTINED', 'DUPLICATE')),
  CHECK (reason = '' OR reason IN ('missing_value', 'unknown_entity', 'ambiguous_entity', 'out_of_range', 'unit_dimension', 'unparseable', 'multiple_items',
    'number_reading_uncertain', 'unsupported_media', 'unrecognized_line', 'unreadable_file', 'unknown_sender', 'sender_not_authorized', 'date_conflict',
    'correction_unconfirmed', 'possible_duplicate', 'ambiguous_kind', 'id_conflict', 'duplicate_key', 'duplicate_fingerprint')),
  CHECK (outcome <> 'ACCEPTED' OR reason = ''),
  CHECK (outcome = 'ACCEPTED' OR (reason <> '' AND detail <> '')),                                                -- CL2
  CHECK (outcome <> 'DUPLICATE' OR reason IN ('duplicate_key', 'duplicate_fingerprint')),
  CHECK (outcome = 'DUPLICATE' OR reason NOT IN ('duplicate_key', 'duplicate_fingerprint'))
);

DO $$
DECLARE t text;
BEGIN
  FOREACH t IN ARRAY ARRAY['report_records', 'report_versions', 'report_history', 'delivery_ledger'] LOOP
    EXECUTE format('ALTER TABLE %I ENABLE ROW LEVEL SECURITY', t);
    EXECUTE format('ALTER TABLE %I FORCE ROW LEVEL SECURITY', t);
    EXECUTE format('DROP POLICY IF EXISTS tenant_isolation ON %I', t);
    EXECUTE format($p$CREATE POLICY tenant_isolation ON %I
      USING (tenant_id = current_setting('eais.tenant_id', true))
      WITH CHECK (tenant_id = current_setting('eais.tenant_id', true))$p$, t);
    EXECUTE format('REVOKE ALL ON %I FROM PUBLIC', t);
    EXECUTE format('REVOKE ALL ON %I FROM eais_app', t);   -- thu cả quyền theo cột
    EXECUTE format('GRANT SELECT, INSERT ON %I TO eais_app', t);
  END LOOP;
END
$$;

-- Trạng thái bản ghi đổi theo quyết định / thu hồi / sửa tin / đánh giá lại TR3: UPDATE theo cột, trừ khoá (tenant_id, unit_key). Cột mới thêm về sau KHÔNG tự sửa được: phải cấp ở migration thêm cột.
GRANT UPDATE (channel, source_type, sender_id, sent_at, occurred_on, entity, metric, value_number, value_unit, approx, kind, corrects, raw, attachments, note,
  disposition, reason, detail, recipient, duplicate_of, superseded_by, version, decided_by, decided_at, decision, reviewed_pairs, pack_version, trace_id,
  fingerprint, content_sha256) ON report_records TO eais_app;
