-- 0008 — vector của đoạn tài liệu tri thức cho truy hồi lai (nhánh RAG, docs/ROADMAP-RAG.md §6.4).
-- Khoá = (tenant, không gian vector, băm chữ đã nhúng). Không gian = model + số chiều của embedder: đổi model là không gian khác, vector
-- cũ không bị so nhầm — chỉ mục báo thiếu vector và phải nhúng lại. Chữ không đổi giữa các lần nạp thì không phải nhúng lại.
-- Vector là mảng real thường: cosine tính trong ứng dụng, CHỈ trên đoạn người hỏi được đọc (lọc quyền / phiên bản trước), nên không
-- cần phần mở rộng nào của PG. RLS và quyền như các bảng nghiệp vụ khác (0001, 0002). Chạy lại được.

CREATE TABLE IF NOT EXISTS document_vectors (
  tenant_id text   NOT NULL,
  space     text   NOT NULL CHECK (space <> ''),
  text_hash text   NOT NULL CHECK (text_hash ~ '^[0-9a-f]{64}$'),
  vec       real[] NOT NULL CHECK (cardinality(vec) > 0),
  PRIMARY KEY (tenant_id, space, text_hash)
);

ALTER TABLE document_vectors ENABLE ROW LEVEL SECURITY;
ALTER TABLE document_vectors FORCE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS tenant_isolation ON document_vectors;
CREATE POLICY tenant_isolation ON document_vectors
  USING (tenant_id = current_setting('eais.tenant_id', true))
  WITH CHECK (tenant_id = current_setting('eais.tenant_id', true));
GRANT SELECT, INSERT, DELETE ON document_vectors TO eais_app;
