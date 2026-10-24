package pgstore_test

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/luannt2002/enterprise-ai-control-layer/intelligence/rag"
	"github.com/luannt2002/enterprise-ai-control-layer/platform/pgstore"
)

// Nguồn tri thức "database có sẵn": đọc theo câu truy vấn của cấu hình, cột theo tên (NULL giữ giá trị cấu hình), giao dịch READ ONLY
// (câu truy vấn ghi bị chặn, bảng còn nguyên — kể cả chuỗi nhiều lệnh khi chuỗi kết nối chọn giao thức đơn giản), vai superuser bị từ
// chối, mật khẩu không lọt vào lỗi; cột lạ / thiếu cột bắt buộc / biến môi trường trống / quyền đọc rỗng là lỗi. Vai thử có cả DELETE
// để chứng minh chính các rào chặn ghi, không phải việc thiếu quyền.
func TestNguonTriThucPostgres(t *testing.T) {
	_, admin, _ := open(t)
	ctx := context.Background()
	conn, err := pgx.Connect(ctx, admin)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close(ctx)
	tbl := fmt.Sprintf("kb_thu_%d", time.Now().UnixNano())
	if _, err := conn.Exec(ctx, `CREATE TABLE `+tbl+` (id text, title text, body text, acl text[], version int, status text, effective_from date);
		INSERT INTO `+tbl+` VALUES
		  ('QT', 'Quy trình nghiệm thu', '1. Biên bản do tư vấn giám sát lập.', '{ks}', 2, 'approved', '2026-03-01'),
		  ('FAQ', 'Hỏi đáp kho', '1. Phiếu xuất ghi sổ rồi không được xoá.', NULL, NULL, NULL, NULL)`); err != nil {
		t.Fatal(err)
	}
	vai := tbl + "_doc"
	if _, err := conn.Exec(ctx, "CREATE ROLE "+vai+" LOGIN; GRANT SELECT, DELETE ON "+tbl+" TO "+vai); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_, _ = conn.Exec(context.Background(), "DROP TABLE IF EXISTS "+tbl)
		_, _ = conn.Exec(context.Background(), "DROP ROLE IF EXISTS "+vai)
	})
	cfg, err := pgx.ParseConfig(admin)
	if err != nil {
		t.Fatal(err)
	}
	dsnVai := fmt.Sprintf("postgres://%s@%s:%d/%s?sslmode=disable", vai, cfg.Host, cfg.Port, cfg.Database)
	t.Setenv("EAIS_KB_THU_DSN", dsnVai)
	t.Setenv("EAIS_KB_DON_GIAN_DSN", dsnVai+"&default_query_exec_mode=simple_protocol")
	t.Setenv("EAIS_KB_QUAN_TRI_DSN", admin)
	t.Setenv("EAIS_KB_SAI_DSN", "host=127.0.0.1 password = MATKHAU_THU_123 sslmode=bogus")
	kinds := rag.SourceKinds{"postgres": pgstore.KnowledgeSQL}
	spec := rag.SourceSpec{Kind: "postgres", DSNEnv: "EAIS_KB_THU_DSN", ACL: []string{"thu_kho"},
		Query: "SELECT id, title, body, acl, version, status, effective_from::text AS effective_from FROM " + tbl + " ORDER BY id"}
	docs, err := rag.Collect(ctx, []rag.SourceSpec{spec}, kinds, "t", "")
	if err != nil {
		t.Fatal(err)
	}
	if len(docs) != 2 || docs[1].ID != "QT" || docs[1].Version != 2 || docs[1].EffectiveFrom != "2026-03-01" || docs[1].ACL[0] != "ks" ||
		docs[0].ACL[0] != "thu_kho" || docs[0].Version != 0 || docs[0].Status != "" || docs[0].Tenant != "t" {
		t.Fatalf("đọc sai: %+v", docs)
	}

	for name, c := range map[string]struct {
		mut  func(*rag.SourceSpec)
		want string
	}{
		"truy vấn ghi (READ ONLY)": {func(s *rag.SourceSpec) {
			s.Query = "WITH x AS (DELETE FROM " + tbl + " RETURNING id, title, body) SELECT * FROM x"
		}, "read-only"},
		"cột lạ":              {func(s *rag.SourceSpec) { s.Query = "SELECT id, title, body, 1 AS mat_khau FROM " + tbl }, "lạ"},
		"tieu_de chưa hỗ trợ": {func(s *rag.SourceSpec) { s.TieuDe = []rag.HeadingRule{{Mau: `^Điều \d+\.`, Cap: 3}} }, "tieu_de"},
		"cột lạ, không hàng nào": {func(s *rag.SourceSpec) {
			s.Query = "SELECT id, title, body, 1 AS mat_khau FROM " + tbl + " WHERE false"
		}, "lạ"},
		"thiếu cột body":           {func(s *rag.SourceSpec) { s.Query = "SELECT id, title FROM " + tbl }, "thiếu cột body"},
		"không quyền đọc":          {func(s *rag.SourceSpec) { s.ACL, s.Query = nil, "SELECT id, title, body FROM "+tbl }, "acl"},
		"vai superuser bị từ chối": {func(s *rag.SourceSpec) { s.DSNEnv = "EAIS_KB_QUAN_TRI_DSN" }, "superuser"},
		"nhiều lệnh, giao thức đơn giản": {func(s *rag.SourceSpec) {
			s.DSNEnv, s.Query = "EAIS_KB_DON_GIAN_DSN", "SELECT id, title, body FROM "+tbl+"; COMMIT; DELETE FROM "+tbl
		}, "42601"},
		"quyền đọc rỗng": {func(s *rag.SourceSpec) {
			s.ACL, s.Query = nil, "SELECT id, title, body, acl FROM "+tbl+" WHERE id = 'FAQ'"
		}, "quyền đọc rỗng"},
		"biến môi trường trống": {func(s *rag.SourceSpec) { s.DSNEnv = "EAIS_KB_KHONG_CO" }, "trống"},
		"thiếu dsn_env / query": {func(s *rag.SourceSpec) { s.Query = "" }, "dsn_env"},
	} {
		s := spec
		c.mut(&s)
		if _, err := rag.Collect(ctx, []rag.SourceSpec{s}, kinds, "t", ""); err == nil || !strings.Contains(err.Error(), c.want) {
			t.Errorf("%s: muốn lỗi chứa %q, được %v", name, c.want, err)
		}
	}
	s := spec
	s.DSNEnv = "EAIS_KB_SAI_DSN"
	if _, err := rag.Collect(ctx, []rag.SourceSpec{s}, kinds, "t", ""); err == nil || strings.Contains(err.Error(), "MATKHAU_THU_123") {
		t.Errorf("chuỗi kết nối hỏng: muốn lỗi KHÔNG chứa mật khẩu, được %v", err)
	}
	var n int
	if err := conn.QueryRow(ctx, "SELECT count(*) FROM "+tbl).Scan(&n); err != nil || n != 2 {
		t.Errorf("READ ONLY: bảng phải còn 2 dòng, còn %d (%v)", n, err)
	}
}
