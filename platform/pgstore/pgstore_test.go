package pgstore_test

import (
	"context"
	"fmt"
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
	admin, app := os.Getenv("EACL_PG_ADMIN_DSN"), os.Getenv("EACL_PG_DSN")
	if admin == "" || app == "" {
		if os.Getenv("EAIS_PG_ADMIN_DSN")+os.Getenv("EAIS_PG_DSN") != "" {
			// môi trường còn đặt DSN theo tiền tố trước 21/10/2026: báo lỗi thay vì bỏ qua lặng lẽ cả đường PG (RLS)
			t.Fatal("DSN PG phải đặt ở EACL_PG_ADMIN_DSN / EACL_PG_DSN (biến EAIS_PG_* không còn được đọc)")
		}
		t.Skip("chưa có EACL_PG_ADMIN_DSN / EACL_PG_DSN — xem deploy/docker-compose.yml")
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

// GUC eacl.tenant_id của mọi lệnh lấy từ tenant gắn vào ngữ cảnh (S1 gắn từ Principal); ngữ cảnh
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
				if _, err := tx.Exec(ctx, "SELECT set_config('eacl.tenant_id', $1, true)", guc); err != nil {
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
		if _, err := tx.Exec(ctx, "SELECT set_config('eacl.tenant_id', $1, true)", string(a)); err != nil {
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
