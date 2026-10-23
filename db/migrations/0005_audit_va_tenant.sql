-- 0005: bổ sung bảo vệ cho ba bảng của 0004 theo lệnh ca chiều 23/10 của chủ dự án.
--
-- C1. audit_log.reason_detail giữ lý do THẬT của một lần từ chối (vd phân biệt "máy không tồn tại" với "máy ngoài phạm vi", HOP-DONG
-- Phase 1 §4.4): vai ứng dụng chỉ được GHI cột này, không được ĐỌC (đo trước bản này: has_column_privilege('eais_app', 'audit_log',
-- 'reason_detail', 'SELECT') = true). PostgreSQL không thu được quyền đọc một cột khi vai còn SELECT mức bảng, nên thu SELECT mức bảng rồi
-- cấp SELECT theo từng cột, trừ reason_detail. Người vận hành đọc cột đó bằng vai quản trị. Cột mới thêm vào audit_log về sau KHÔNG tự
-- đọc được bằng vai ứng dụng: phải cấp ở migration thêm cột.
REVOKE SELECT ON audit_log FROM eais_app;
GRANT SELECT (tenant_id, id, at, outcome, reason_code, caller_id, caller_roles, entity_scope, command_id, action, subject_key, metric, day,
  ticket_id, fingerprint, pack_version, code_version, trace_id) ON audit_log TO eais_app;

-- C3. tenant_id rỗng bị chặn ở LƯỢC ĐỒ cho ba bảng của 0004 (trước đây chỉ tầng Go từ chối tenant rỗng; vai quản trị hay một đường ghi
-- khác vẫn ghi được dòng tenant ''). Chạy lại được: bỏ rồi thêm ràng buộc cùng tên. Nợ cùng loại ở các bảng của 0001 (dữ liệu nghiệp vụ)
-- và 0003 (vết bền): chưa sửa ở đây, ghi nợ ở docs/bao-cao/TIEN-DO-BUILD.md.
DO $$
DECLARE t text;
BEGIN
  FOREACH t IN ARRAY ARRAY['discrepancy_tickets', 'commands', 'audit_log'] LOOP
    EXECUTE format('ALTER TABLE %I DROP CONSTRAINT IF EXISTS %I', t, t || '_tenant_khong_rong');
    EXECUTE format('ALTER TABLE %I ADD CONSTRAINT %I CHECK (tenant_id <> %L)', t, t || '_tenant_khong_rong', '');
  END LOOP;
END
$$;
