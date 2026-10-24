-- 0010 — tenant rỗng bị chặn ở LƯỢC ĐỒ cho mọi bảng theo tenant còn nợ (F-037 của báo cáo audit vNext; nợ ghi ở 0005, C3). Trước bản này
-- chỉ ba bảng của 0004 (qua 0005), load_records (0006) và bốn bảng báo cáo (0009) có CHECK; bảng dữ liệu nghiệp vụ (0001), vết bền (0003)
-- và vector tài liệu (0008) thì không: vai quản trị hay một đường ghi khác vẫn tạo được dòng tenant '' — dòng mà một phiên đặt GUC
-- eais.tenant_id = '' sẽ thấy và Load ("xoá rồi ghi lại", chỉ dựa vào RLS) sẽ xoá. Phòng thủ chiều sâu, cùng lớp với F-010 (Ready).
-- Không sửa migration cũ (ADR-0017). Ràng buộc cùng tên và cùng chữ với 0005 (<bảng>_tenant_khong_rong, tenant_id <> '').
-- Chạy lại được: bỏ rồi thêm ràng buộc cùng tên (như 0005), nên chạy lại đưa về đúng một trạng thái.
-- Không dùng NOT VALID rồi VALIDATE: Migrate chạy mỗi tệp trong MỘT giao dịch, khoá ACCESS EXCLUSIVE của ADD CONSTRAINT giữ tới lúc commit,
-- nên VALIDATE trong cùng tệp quét dưới đúng khoá đó — không lợi gì. Bảng lớn tới mức cần thì tách VALIDATE ra một migration sau.
-- Dữ liệu bẩn: database đã có dòng tenant '' thì ADD CONSTRAINT báo "violated by some row" và CẢ tệp hoàn tác (đóng mặc định như 0005;
-- ứng dụng vẫn chạy vì Ready không đòi 0010). Đếm trước khi áp, bằng vai quản trị:
--   SELECT 'entities', count(*) FROM entities WHERE tenant_id = '' UNION ALL SELECT 'entity_aliases', count(*) FROM entity_aliases WHERE tenant_id = ''
--   UNION ALL SELECT 'facts', count(*) FROM facts WHERE tenant_id = '' UNION ALL SELECT 'documents', count(*) FROM documents WHERE tenant_id = ''
--   UNION ALL SELECT 'decision_records', count(*) FROM decision_records WHERE tenant_id = ''
--   UNION ALL SELECT 'quarantine_records', count(*) FROM quarantine_records WHERE tenant_id = ''
--   UNION ALL SELECT 'usage_records', count(*) FROM usage_records WHERE tenant_id = ''
--   UNION ALL SELECT 'document_vectors', count(*) FROM document_vectors WHERE tenant_id = '';
DO $$
DECLARE t text;
BEGIN
  FOREACH t IN ARRAY ARRAY['entities', 'entity_aliases', 'facts', 'documents', 'decision_records', 'quarantine_records', 'usage_records',
    'document_vectors'] LOOP
    EXECUTE format('ALTER TABLE %I DROP CONSTRAINT IF EXISTS %I', t, t || '_tenant_khong_rong');
    EXECUTE format('ALTER TABLE %I ADD CONSTRAINT %I CHECK (tenant_id <> %L)', t, t || '_tenant_khong_rong', '');
  END LOOP;
END
$$;
