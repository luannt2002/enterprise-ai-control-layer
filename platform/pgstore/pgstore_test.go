package pgstore_test

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/luannt2002/enterprise-ai-control-layer/core/contracts"
	"github.com/luannt2002/enterprise-ai-control-layer/data/entityresolution"
	"github.com/luannt2002/enterprise-ai-control-layer/data/sourceoftruth"
	"github.com/luannt2002/enterprise-ai-control-layer/intelligence/rag"
	"github.com/luannt2002/enterprise-ai-control-layer/platform/pgstore"
)

// open nối tới PG của deploy/docker-compose.yml. Thiếu biến môi trường thì bỏ qua (ĐỎ-MT, không
// phải đạt): báo cáo WIN ghi rõ đường PG có chạy hay không.
func open(t *testing.T) (*pgstore.Store, string, string) {
	t.Helper()
	admin, app := pgstore.EnvDSN()
	if admin == "" || app == "" {
		t.Skip("chưa có EAIS_PG_ADMIN_DSN / EAIS_PG_DSN — xem deploy/docker-compose.yml")
	}
	ctx := context.Background()
	if err := pgstore.Migrate(ctx, admin); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	s, err := pgstore.Open(ctx, app)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	t.Cleanup(s.Close)
	return s, admin, app
}

var day = time.Date(2026, 10, 19, 0, 0, 0, 0, time.FixedZone("ICT", 7*3600))

// seed nạp cho một tenant: máy EX-03 (cùng khoá ở cả hai tenant, khác tên), một fact, một tài liệu.
func seed(t *testing.T, s *pgstore.Store, tenant contracts.TenantID, name, value, secret string) {
	t.Helper()
	ctx := contracts.WithTenant(context.Background(), tenant)
	es := []entityresolution.Entity{{ID: "EX-03", Type: "equipment", Key: "EX-03", Name: name,
		Aliases: []entityresolution.Alias{{Text: "máy 03", Source: "manual"}}}}
	fs := []sourceoftruth.Fact{{ID: "R-1", Metric: "issued_fuel", EntityID: "EX-03", OccurredOn: day,
		Value:  contracts.Value{Kind: contracts.ValueQuantity, Number: value, Unit: "L"},
		Source: sourceoftruth.Observation{ID: "R-1", SourceType: "warehouse_slip", SourceRef: "R-1"}}}
	ds := []rag.Document{{ID: "SOP", Title: "Quy trình xử lý chênh lệch nhiên liệu", ACL: []string{"director"},
		Body: "1. Lập biên bản đối soát trong vòng 24 giờ. " + secret}}
	if err := s.Load(ctx, es, fs, ds); err != nil {
		t.Fatalf("nạp %s: %v", tenant, err)
	}
}

func uniq(p string) contracts.TenantID {
	return contracts.TenantID(fmt.Sprintf("%s-%d", p, time.Now().UnixNano()))
}

// GUC eais.tenant_id của mọi lệnh lấy từ tenant gắn vào ngữ cảnh (S1 gắn từ Principal); ngữ cảnh
// không mang tenant thì lệnh bị từ chối trước khi chạm DB.
func TestGUCLayTuNguCanh(t *testing.T) {
	s, _, _ := open(t)
	a := uniq("ta")
	got, err := s.Setting(contracts.WithTenant(context.Background(), a))
	if err != nil || got != string(a) {
		t.Fatalf("GUC = %q (lỗi %v), muốn %q", got, err, a)
	}
	bare := context.Background()
	if _, err := s.Facts(bare, "issued_fuel", nil, day, day); contracts.FaultOf(err) != contracts.FaultPolicy {
		t.Errorf("Facts không tenant: muốn lỗi %s, được %v", contracts.FaultPolicy, err)
	}
	if _, err := s.Entities(bare); contracts.FaultOf(err) != contracts.FaultPolicy {
		t.Errorf("Entities không tenant: muốn lỗi %s, được %v", contracts.FaultPolicy, err)
	}
	if err := s.Load(bare, nil, nil, nil); contracts.FaultOf(err) != contracts.FaultPolicy {
		t.Errorf("Load không tenant: muốn lỗi %s, được %v", contracts.FaultPolicy, err)
	}
}

// Open chỉ nhận vai KHÔNG vượt được RLS: DSN quản trị trỏ nhầm vào Store thì Load (DELETE FROM dựa vào RLS) xoá dữ liệu
// của MỌI tenant. Ca âm: vai quản trị dựng sẵn (superuser + BYPASSRLS), một vai chỉ BYPASSRLS và một vai chỉ SUPERUSER (vẫn
// vượt RLS dù NOBYPASSRLS) đều bị từ chối với lỗi chính sách. Ca dương: vai ứng dụng; và vai ứng dụng đặt bằng role= trên DSN
// quản trị — Open xét vai HIỆU LỰC (current_user), không xét vai đăng nhập.
// Vai tạm là NOLOGIN, vào bằng tham số role= trên DSN quản trị: không cần mật khẩu / pg_hba, và vai sót lại khi tiến trình bị
// kill thì không đăng nhập được.
func TestOpenTuChoiVaiVuotRLS(t *testing.T) {
	_, admin, app := open(t)
	ctx := context.Background()
	u, err := url.Parse(admin)
	if err != nil || u.User == nil {
		t.Skipf("DSN quản trị không ở dạng URL có tên vai (docs/ops/README.md) — không dựng được DSN role=: %v", err)
	}
	withRole := func(role string) string {
		r := *u
		q := r.Query()
		q.Set("role", role)
		r.RawQuery = q.Encode()
		return r.String()
	}
	conn, err := pgx.Connect(ctx, admin)
	if err != nil {
		t.Fatal(err)
	}
	// Cleanup chạy ngược thứ tự đăng ký: xoá vai tạm trước, đóng kết nối sau (defer sẽ đóng kết nối TRƯỚC Cleanup).
	t.Cleanup(func() { conn.Close(context.Background()) })
	suffix := fmt.Sprintf("%d_%d", os.Getpid(), time.Now().UnixNano()%1_000_000)
	refuse := map[string]string{"quản trị dựng sẵn": admin}
	for name, attrs := range map[string]string{"bypass": "NOSUPERUSER BYPASSRLS", "super": "SUPERUSER NOBYPASSRLS"} {
		role := "eais_test_" + name + "_" + suffix
		if _, err := conn.Exec(ctx, "CREATE ROLE "+role+" NOLOGIN "+attrs); err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() {
			if _, err := conn.Exec(context.Background(), "DROP ROLE IF EXISTS "+role); err != nil {
				t.Errorf("xoá vai tạm %s: %v", role, err)
			}
		})
		refuse[attrs] = withRole(role)
	}
	for name, dsn := range refuse {
		s, err := pgstore.Open(ctx, dsn)
		if err == nil {
			s.Close()
			t.Errorf("%s: Open phải từ chối vai vượt RLS", name)
			continue
		}
		if contracts.FaultOf(err) != contracts.FaultPolicy || !strings.Contains(err.Error(), "RLS") {
			t.Errorf("%s: muốn lỗi %s nêu RLS, được %v", name, contracts.FaultPolicy, err)
		}
	}
	for name, dsn := range map[string]string{"vai ứng dụng": app, "vai ứng dụng qua role= trên DSN quản trị": withRole("eais_app")} {
		s, err := pgstore.Open(ctx, dsn)
		if err != nil {
			t.Fatalf("%s phải mở được: %v", name, err)
		}
		s.Close()
	}
}

// Canary B: không dòng / thực thể / fact / tài liệu nào của B tới được từ A (cho phép + từ chối),
// và dữ liệu của B CÓ thật trong DB (đối chứng dương: số 0 của A không phải do bảng rỗng).
func TestRLSCanaryB(t *testing.T) {
	s, admin, app := open(t)
	a, b := uniq("ta"), uniq("tb")
	seed(t, s, a, "Máy xúc EX-03", "3850", "")
	seed(t, s, b, "Máy xúc EX-03 của B", "9999", "Mã bí mật canary: B-CANARY-7731.")
	ctxA, ctxB := contracts.WithTenant(context.Background(), a), contracts.WithTenant(context.Background(), b)

	// cho phép: mỗi tenant thấy đúng dữ liệu của mình
	for _, c := range []struct {
		ctx         context.Context
		name, value string
	}{{ctxA, "Máy xúc EX-03", "3850"}, {ctxB, "Máy xúc EX-03 của B", "9999"}} {
		es, err := s.Entities(c.ctx)
		if err != nil || len(es) != 1 || es[0].Name != c.name || len(es[0].Aliases) != 1 {
			t.Errorf("Entities: được %+v (lỗi %v), muốn đúng một %q", es, err, c.name)
		}
		for _, ids := range [][]string{{"EX-03"}, nil} { // nil = không lọc thực thể, như bản bộ nhớ
			fs, err := s.Facts(c.ctx, "issued_fuel", ids, day, day)
			if err != nil || len(fs) != 1 || fs[0].Value.Number != c.value || !fs[0].OccurredOn.Equal(day) {
				t.Errorf("Facts(%v): được %+v (lỗi %v), muốn đúng một fact %s", ids, fs, err, c.value)
			}
		}
	}
	// từ chối: truy xuất của A không ra tài liệu / mã của B
	ev, err := s.Retrieve(ctxA, contracts.Principal{ID: "u", TenantID: a, Roles: []string{"director"}}, "Mã bí mật canary B-CANARY-7731 trong quy trình xử lý chênh lệch", 3)
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range ev {
		if strings.Contains(e.Text, "7731") {
			t.Errorf("rò tài liệu của B sang A: %q", e.Text)
		}
	}
	// từ chối: principal A với ngữ cảnh B
	if _, err := s.Retrieve(ctxB, contracts.Principal{ID: "u", TenantID: a, Roles: []string{"director"}}, "quy trình", 3); contracts.FaultOf(err) != contracts.FaultPolicy {
		t.Errorf("principal A trong ngữ cảnh B: muốn lỗi %s, được %v", contracts.FaultPolicy, err)
	}

	ctx := context.Background()
	appConn, err := pgx.Connect(ctx, app)
	if err != nil {
		t.Fatal(err)
	}
	defer appConn.Close(ctx)
	count := func(conn *pgx.Conn, guc, where string, args ...any) int {
		t.Helper()
		var n int
		err := pgx.BeginFunc(ctx, conn, func(tx pgx.Tx) error {
			if guc != "" {
				if _, err := tx.Exec(ctx, "SELECT set_config('eais.tenant_id', $1, true)", guc); err != nil {
					return err
				}
			}
			return tx.QueryRow(ctx, "SELECT count(*) FROM facts WHERE "+where, args...).Scan(&n)
		})
		if err != nil {
			t.Fatal(err)
		}
		return n
	}
	// vai ứng dụng, không đặt GUC: 0 dòng (đóng mặc định)
	if n := count(appConn, "", "true"); n != 0 {
		t.Errorf("không GUC: thấy %d fact, muốn 0", n)
	}
	// vai ứng dụng, GUC = A, hỏi đích danh dòng của B: 0 dòng
	if n := count(appConn, string(a), "tenant_id = $1", string(b)); n != 0 {
		t.Errorf("GUC A hỏi dòng của B: thấy %d, muốn 0", n)
	}
	// đối chứng dương: quản trị (bỏ qua RLS) thấy dòng của B
	adminConn, err := pgx.Connect(ctx, admin)
	if err != nil {
		t.Fatal(err)
	}
	defer adminConn.Close(ctx)
	if n := count(adminConn, "", "tenant_id = $1", string(b)); n != 1 {
		t.Errorf("đối chứng: quản trị thấy %d fact của B, muốn 1 — số 0 ở trên mới có nghĩa", n)
	}
	// từ chối ghi: GUC = A mà ghi dòng mang tenant B → RLS WITH CHECK chặn
	err = pgx.BeginFunc(ctx, appConn, func(tx pgx.Tx) error {
		if _, err := tx.Exec(ctx, "SELECT set_config('eais.tenant_id', $1, true)", string(a)); err != nil {
			return err
		}
		_, err := tx.Exec(ctx, `INSERT INTO facts (tenant_id, id, metric, entity_id, occurred_on, value, unit, source_type, ord)
			VALUES ($1, 'R-X', 'issued_fuel', 'EX-03', '2026-10-19', 1, 'L', 'erp', 0)`, string(b))
		return err
	})
	if err == nil || !strings.Contains(err.Error(), "row-level security") {
		t.Errorf("ghi chéo tenant: muốn bị RLS chặn, được %v", err)
	}
}

// Migration 0002 (D-RM-02): vai ứng dụng là eais_app, chính sách RLS đọc GUC eais.tenant_id. Kiểm ĐÚNG tập quyền, không chỉ vài quyền:
// eais_app có SELECT / INSERT / DELETE như vai cũ của 0001 và không gì hơn — TRUNCATE (không chịu RLS: xoá sạch bảng của mọi tenant),
// REFERENCES, TRIGGER, UPDATE, MAINTAIN (PG 17) hay quyền kèm GRANT OPTION đều là quyền thừa, kể cả khi đến qua PUBLIC; vai eacl_app
// của 0001 không còn quyền nào (vai không có trong cụm thì coi là không có quyền); RLS vẫn ENABLE + FORCE trên cả 4 bảng; đặt GUC tên
// cũ không mở dòng nào (đối chứng dương: GUC mới thấy đúng dòng của tenant).
func TestMigration0002DinhDanhEAIS(t *testing.T) {
	s, admin, app := open(t)
	a := uniq("ta")
	seed(t, s, a, "Máy xúc EX-03", "3850", "")
	ctx := context.Background()
	adminConn, err := pgx.Connect(ctx, admin)
	if err != nil {
		t.Fatal(err)
	}
	defer adminConn.Close(ctx)
	privs := []string{"SELECT", "INSERT", "UPDATE", "DELETE", "TRUNCATE", "REFERENCES", "TRIGGER"}
	var version int
	if err := adminConn.QueryRow(ctx, "SELECT current_setting('server_version_num')::int").Scan(&version); err != nil {
		t.Fatal(err)
	}
	if version >= 170000 {
		privs = append(privs, "MAINTAIN")
	}
	has := func(role, table, priv string) bool {
		t.Helper()
		var got bool
		if err := adminConn.QueryRow(ctx, "SELECT CASE WHEN to_regrole($1) IS NULL THEN false ELSE has_table_privilege($2, $3, $4) END",
			role, role, table, priv).Scan(&got); err != nil {
			t.Fatal(err)
		}
		return got
	}
	for _, table := range []string{"entities", "entity_aliases", "facts", "documents"} {
		for _, priv := range privs {
			want := priv == "SELECT" || priv == "INSERT" || priv == "DELETE"
			if got := has("eais_app", table, priv); got != want {
				t.Errorf("eais_app %s trên %s: được %v, muốn %v", priv, table, got, want)
			}
			if has("eais_app", table, priv+" WITH GRANT OPTION") {
				t.Errorf("eais_app %s WITH GRANT OPTION trên %s: không được có", priv, table)
			}
			if has("eacl_app", table, priv) {
				t.Errorf("vai cũ eacl_app còn %s trên %s", priv, table)
			}
		}
		var enabled, forced bool
		if err := adminConn.QueryRow(ctx, "SELECT relrowsecurity, relforcerowsecurity FROM pg_class WHERE oid = $1::regclass", table).Scan(&enabled, &forced); err != nil {
			t.Fatal(err)
		}
		if !enabled || !forced {
			t.Errorf("%s: RLS ENABLE = %v, FORCE = %v, muốn cả hai bật", table, enabled, forced)
		}
	}
	appConn, err := pgx.Connect(ctx, app)
	if err != nil {
		t.Fatal(err)
	}
	defer appConn.Close(ctx)
	for guc, want := range map[string]int{"eacl.tenant_id": 0, "eais.tenant_id": 1} {
		var n int
		err := pgx.BeginFunc(ctx, appConn, func(tx pgx.Tx) error {
			if _, err := tx.Exec(ctx, "SELECT set_config($1, $2, true)", guc, string(a)); err != nil {
				return err
			}
			return tx.QueryRow(ctx, "SELECT count(*) FROM facts WHERE tenant_id = $1", string(a)).Scan(&n)
		})
		if err != nil || n != want {
			t.Errorf("GUC %s = tenant A: thấy %d fact (lỗi %v), muốn %d", guc, n, err, want)
		}
	}
}

// Nạp lại A thay trọn dữ liệu của A, không đụng dữ liệu của B (lệnh xoá cũng đi qua RLS).
func TestLoadThayDuLieuMotTenant(t *testing.T) {
	s, _, _ := open(t)
	a, b := uniq("ta"), uniq("tb")
	seed(t, s, a, "Máy xúc EX-03", "3850", "")
	seed(t, s, b, "Máy xúc EX-03 của B", "9999", "")
	seed(t, s, a, "Máy xúc EX-03", "4000", "") // nạp lại A với số mới
	for _, c := range []struct {
		tenant contracts.TenantID
		want   string
	}{{a, "4000"}, {b, "9999"}} {
		fs, err := s.Facts(contracts.WithTenant(context.Background(), c.tenant), "issued_fuel", nil, day, day)
		if err != nil || len(fs) != 1 || fs[0].Value.Number != c.want {
			t.Errorf("%s sau khi nạp lại A: được %+v (lỗi %v), muốn đúng một fact %s", c.tenant, fs, err, c.want)
		}
	}
}

// Truy xuất lọc ACL trong SQL: vai không có trong acl thì không đoạn nào ra khỏi DB.
func TestRetrieveACL(t *testing.T) {
	s, _, _ := open(t)
	a := uniq("ta")
	seed(t, s, a, "Máy xúc EX-03", "3850", "")
	ctx := contracts.WithTenant(context.Background(), a)
	q := "Quy trình xử lý chênh lệch nhiên liệu là gì?"
	if ev, err := s.Retrieve(ctx, contracts.Principal{ID: "u", TenantID: a, Roles: []string{"director"}}, q, 3); err != nil || len(ev) == 0 || ev[0].Source.DocumentID != "SOP" {
		t.Errorf("vai có trong acl: muốn đoạn của SOP, được %+v (lỗi %v)", ev, err)
	}
	if ev, err := s.Retrieve(ctx, contracts.Principal{ID: "u", TenantID: a, Roles: []string{"driver"}}, q, 3); err != nil || len(ev) != 0 {
		t.Errorf("vai ngoài acl: muốn 0 đoạn, được %d (lỗi %v)", len(ev), err)
	}
}
