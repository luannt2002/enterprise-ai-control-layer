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

// Tài liệu có phiên bản trên PG (migration 0006, nhánh RAG U6a) — các ca soát độc lập U6a chỉ ra là chưa có gì canh.

// CHECK của 0006 chặn ngày sai dạng và bản âm: chỉ mục so ngày theo chuỗi, một ngày viết lệch sẽ làm bản hết hiệu lực được phục vụ.
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

// Nâng cấp kho ĐANG CÓ dữ liệu trước 0006 lên 0006: dòng cũ còn nguyên (bản 0) và đọc được; Ready từ chối kho chưa áp 0006 (bảng có
// mặt chưa chứng minh migration đã áp — thiếu cột thì lượt RAG / lần nạp đầu tiên mới lỗi); chính tệp 0006 chạy lại trần không lỗi.
func TestMigration0006NangCapVaReady(t *testing.T) {
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
		"DELETE FROM schema_migrations WHERE version = '0006_tai_lieu_phien_ban.sql'",
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
		t.Errorf("kho chưa áp 0006: Ready phải từ chối và nêu cột thiếu, được %v", err)
	}
	if err := pgstore.Migrate(ctx, fa); err != nil {
		t.Fatalf("0006 trên kho có dữ liệu: %v", err)
	}
	if err := s.Ready(ctx); err != nil {
		t.Errorf("sau 0006 Ready phải đạt: %v", err)
	}
	pr := contracts.Principal{ID: "u", TenantID: "t-cu", Roles: []string{"director"}}
	ev, err := s.Retrieve(contracts.WithTenant(ctx, "t-cu"), pr, "quy trình đổ bê tông", 3)
	if err != nil || len(ev) == 0 || ev[0].Source.DocumentVersion != 0 || !strings.HasPrefix(ev[0].ID, "SOP#") {
		t.Errorf("tài liệu cũ sau nâng cấp phải đọc được y như trước (mã SOP#…, bản 0): %+v %v", ev, err)
	}
	sql, err := db.Migrations.ReadFile("migrations/0006_tai_lieu_phien_ban.sql")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := conn.Exec(ctx, string(sql)); err != nil {
		t.Errorf("chạy lại trần 0006: %v", err)
	}
}

// Chỉ mục tri thức giữ sẵn theo tenant không được phục vụ dữ liệu cũ: nạp lại trong cùng tiến trình, và nạp từ tiến trình KHÁC
// (một Store riêng — như `eais ingest` khi server đang chạy), lượt sau phải thấy nội dung mới, kể cả khi số tài liệu không đổi.
func TestChiMucGiuSanKhongCu(t *testing.T) {
	s1, _, app := open(t)
	s2, err := pgstore.Open(context.Background(), app)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(s2.Close)
	tenant := uniq("chi-muc")
	ctx := contracts.WithTenant(context.Background(), tenant)
	t.Cleanup(func() { _ = s1.Load(ctx, nil, nil, nil) })
	pr := contracts.Principal{ID: "u", TenantID: tenant, Roles: []string{"director"}}
	sop := func(han string) []rag.Document {
		return []rag.Document{{ID: "SOP", Title: "Quy trình đối soát nhiên liệu", ACL: []string{"director"},
			Body: "1. Lập biên bản đối soát trong vòng " + han + "."}}
	}
	hoi := func(s *pgstore.Store) string {
		ev, err := s.Retrieve(ctx, pr, "biên bản đối soát nhiên liệu lập trong bao lâu", 3)
		if err != nil || len(ev) == 0 {
			t.Fatalf("truy xuất: %v / %v", ev, err)
		}
		return ev[0].Text
	}
	if err := s1.Load(ctx, nil, nil, sop("24 giờ")); err != nil {
		t.Fatal(err)
	}
	if got := hoi(s1); !strings.Contains(got, "24 giờ") {
		t.Fatalf("lượt đầu: %q", got)
	}
	if err := s1.Load(ctx, nil, nil, sop("36 giờ")); err != nil { // cùng tiến trình, cùng số tài liệu
		t.Fatal(err)
	}
	if got := hoi(s1); !strings.Contains(got, "36 giờ") {
		t.Errorf("nạp lại cùng tiến trình mà vẫn trả nội dung cũ: %q", got)
	}
	if err := s2.Load(ctx, nil, nil, sop("48 giờ")); err != nil { // tiến trình khác nạp
		t.Fatal(err)
	}
	if got := hoi(s1); !strings.Contains(got, "48 giờ") {
		t.Errorf("tiến trình khác nạp mà chỉ mục giữ sẵn vẫn trả nội dung cũ: %q", got)
	}
}
