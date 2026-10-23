-- 0007 — tài liệu tri thức có phiên bản (nhánh RAG, docs/viec/2-tra-quy-trinh/KE-HOACH.md U6; BR-DOC-01/02/03).
-- Trước bản này bảng documents chỉ giữ id / tiêu đề / thân / acl với khoá (tenant_id, id): nạp tài liệu có phiên bản thì bản nháp,
-- bản hết hiệu lực, bản ngoài phạm vi bị phục vụ như bản chính, còn hai bản cùng mã thì nạp hỏng vì trùng khoá.
-- Cột mới mang đúng các trường của rag.Document; giá trị mặc định = tài liệu không khai phiên bản (version 0, chưa giới hạn gì),
-- nên dòng cũ đọc ra y như trước. Ngày giữ dạng chữ "YYYY-MM-DD" vì chỉ mục so ngày theo chuỗi — CHECK chặn dạng sai, không
-- để một ngày viết lệch làm bản hết hiệu lực được phục vụ. RLS và quyền của bảng giữ nguyên (0001, 0002).
-- Chạy lại được: ADD COLUMN IF NOT EXISTS, khoá chính DROP rồi ADD.

ALTER TABLE documents
  ADD COLUMN IF NOT EXISTS version        int    NOT NULL DEFAULT 0 CHECK (version >= 0),
  ADD COLUMN IF NOT EXISTS status         text   NOT NULL DEFAULT '',
  ADD COLUMN IF NOT EXISTS effective_from text   NOT NULL DEFAULT '' CHECK (effective_from ~ '^([0-9]{4}-[0-9]{2}-[0-9]{2})?$'),
  ADD COLUMN IF NOT EXISTS effective_to   text   NOT NULL DEFAULT '' CHECK (effective_to ~ '^([0-9]{4}-[0-9]{2}-[0-9]{2})?$'),
  ADD COLUMN IF NOT EXISTS scope          text[] NOT NULL DEFAULT '{}',
  ADD COLUMN IF NOT EXISTS hold           text   NOT NULL DEFAULT '';

ALTER TABLE documents DROP CONSTRAINT IF EXISTS documents_pkey;
ALTER TABLE documents ADD CONSTRAINT documents_pkey PRIMARY KEY (tenant_id, id, version);
