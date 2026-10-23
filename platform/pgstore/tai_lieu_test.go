package pgstore_test

import (
	"context"
	"strings"
	"testing"

	"github.com/luannt2002/enterprise-ai-control-layer/core/contracts"
	"github.com/luannt2002/enterprise-ai-control-layer/db"
	"github.com/luannt2002/enterprise-ai-control-layer/intelligence/rag"
	"github.com/luannt2002/enterprise-ai-control-layer/platform/pgstore"
)

// Tài liệu có phiên bản trên PG (migration 0005, nhánh RAG U6a) — các ca soát độc lập U6a chỉ ra là chưa có gì canh.

// CHECK của 0005 chặn ngày sai dạng và bản âm: chỉ mục so ngày theo chuỗi, một ngày viết lệch sẽ làm bản hết hiệu lực được phục vụ.
// Tài liệu không có trong acl.json (ACL rỗng) nạp được và không ai đọc được — như đường bộ nhớ, không làm hỏng cả lần nạp.
func TestTaiLieuPhienBanNapPG(t *testing.T) {
	s, _, _ := open(t)
	tenant := uniq("tai-lieu")
	ctx := contracts.WithTenant(context.Background(), tenant)
	sop := rag.Document{ID: "SOP", Title: "Quy trình đổ bê tông", ACL: []string{"director"}, Version: 1, Status: rag.StatusApproved,
		Body: "1. Kiểm tra cốp pha trước khi đổ bê tông."}
	t.Cleanup(func() { _ = s.Load(ctx, nil, nil, nil) })
	for name, d := range map[string]rag.Document{
		"ngày sai dạng": func() rag.Document { x := sop; x.EffectiveFrom = "2026/01/01"; return x }(),
		"bản âm":        func() rag.Document { x := sop; x.Version = -1; return x }(),
	} {
		if err := s.Load(ctx, nil, nil, []rag.Document{d}); err == nil {
			t.Errorf("%s: Load phải lỗi", name)
		}
	}
	khongACL := rag.Document{ID: "NHAP", Title: "Quy trình đổ bê tông nháp", Body: "1. Đổ bê tông không cần kiểm tra."}
	if err := s.Load(ctx, nil, nil, []rag.Document{sop, khongACL}); err != nil {
		t.Fatalf("tài liệu ACL rỗng làm hỏng lần nạp: %v", err)
	}
	ev, err := s.Retrieve(ctx, contracts.Principal{ID: "u", TenantID: tenant, Roles: []string{"director"}}, "quy trình đổ bê tông", 5)
	if err != nil || len(ev) == 0 {
		t.Fatalf("director phải đọc được SOP: %v %v", ev, err)
	}
	for _, e := range ev {
		if e.Source.DocumentID == "NHAP" {
			t.Errorf("tài liệu ACL rỗng không ai được đọc, lại lọt: %+v", e)
		}
	}
}

// Nâng cấp kho ĐANG CÓ dữ liệu trước 0005 lên 0005: dòng cũ còn nguyên (bản 0) và đọc được; Ready từ chối kho chưa áp 0005 (bảng có
// mặt chưa chứng minh migration đã áp — thiếu cột thì lượt RAG / lần nạp đầu tiên mới lỗi); chính tệp 0005 chạy lại trần không lỗi.
func TestMigration0005NangCapVaReady(t *testing.T) {
	_, admin, app := open(t)
	fa, fp := freshDB(t, admin, app)
	ctx := context.Background()
	if err := pgstore.Migrate(ctx, fa); err != nil {
		t.Fatal(err)
	}
	conn := connect(t, fa)
	for _, q := range []string{ // đưa kho về đúng hình dạng sau 0003, kèm một tài liệu cũ
		"ALTER TABLE documents DROP CONSTRAINT documents_pkey",
		"ALTER TABLE documents DROP COLUMN version, DROP COLUMN status, DROP COLUMN effective_from, DROP COLUMN effective_to, DROP COLUMN scope, DROP COLUMN hold",
		"ALTER TABLE documents ADD CONSTRAINT documents_pkey PRIMARY KEY (tenant_id, id)",
		"DELETE FROM schema_migrations WHERE version = '0005_tai_lieu_phien_ban.sql'",
		"INSERT INTO documents (tenant_id, id, title, body, acl) VALUES ('t-cu', 'SOP', 'Quy trình đổ bê tông', '1. Kiểm tra cốp pha.', '{director}')",
	} {
		if _, err := conn.Exec(ctx, q); err != nil {
			t.Fatalf("%s: %v", q, err)
		}
	}
	s, err := pgstore.Open(ctx, fp)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	if err := s.Ready(ctx); err == nil || !strings.Contains(err.Error(), "documents.version") {
		t.Errorf("kho chưa áp 0005: Ready phải từ chối và nêu cột thiếu, được %v", err)
	}
	if err := pgstore.Migrate(ctx, fa); err != nil {
		t.Fatalf("0005 trên kho có dữ liệu: %v", err)
	}
	if err := s.Ready(ctx); err != nil {
		t.Errorf("sau 0005 Ready phải đạt: %v", err)
	}
	pr := contracts.Principal{ID: "u", TenantID: "t-cu", Roles: []string{"director"}}
	ev, err := s.Retrieve(contracts.WithTenant(ctx, "t-cu"), pr, "quy trình đổ bê tông", 3)
	if err != nil || len(ev) == 0 || ev[0].Source.DocumentVersion != 0 || !strings.HasPrefix(ev[0].ID, "SOP#") {
		t.Errorf("tài liệu cũ sau nâng cấp phải đọc được y như trước (mã SOP#…, bản 0): %+v %v", ev, err)
	}
	sql, err := db.Migrations.ReadFile("migrations/0005_tai_lieu_phien_ban.sql")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := conn.Exec(ctx, string(sql)); err != nil {
		t.Errorf("chạy lại trần 0005: %v", err)
	}
}
