package pgstore_test

import (
	"context"
	"strings"
	"testing"

	"github.com/luannt2002/enterprise-ai-control-layer/core/contracts"
	"github.com/luannt2002/enterprise-ai-control-layer/platform/pgstore"
)

// CA ÂM + CHO PHÉP của Ready: database trống (chưa migrate) thì kho CHƯA sẵn sàng và lỗi nói rõ thiếu bảng, mang nhãn hạ tầng; sau
// Migrate thì sẵn sàng; thu một quyền cần thiết của vai ứng dụng thì lại chưa sẵn sàng và lỗi nêu đúng quyền.
func TestReadyKhoSanSang(t *testing.T) {
	_, admin, app := open(t)
	fa, fp := freshDB(t, admin, app)
	ctx := context.Background()

	// chưa migrate: Open thành công (vai eais_app có sẵn ở cụm) nhưng bảng chưa có
	s, err := pgstore.Open(ctx, fp)
	if err != nil {
		t.Fatalf("Open trên DB trống: %v", err)
	}
	defer s.Close()
	err = s.Ready(ctx)
	if err == nil || !strings.Contains(err.Error(), "bảng decision_records chưa có") || !strings.Contains(err.Error(), "eais migrate") {
		t.Fatalf("DB chưa migrate: muốn lỗi nêu bảng thiếu và lệnh migrate, được %v", err)
	}
	if got := contracts.FaultOf(err); got != contracts.Infra("postgres") {
		t.Errorf("nhãn lỗi = %q, muốn %q", got, contracts.Infra("postgres"))
	}

	if err := pgstore.Migrate(ctx, fa); err != nil {
		t.Fatal(err)
	}
	if err := s.Ready(ctx); err != nil {
		t.Fatalf("sau Migrate: %v", err)
	}

	// thu quyền INSERT của vai ứng dụng trên bảng vết: kho không còn ghi được vết ⇒ KHÔNG sẵn sàng
	conn := connect(t, fa)
	if _, err := conn.Exec(ctx, "REVOKE INSERT ON decision_records FROM eais_app"); err != nil {
		t.Fatal(err)
	}
	err = s.Ready(ctx)
	if err == nil || !strings.Contains(err.Error(), "thiếu quyền INSERT trên decision_records") {
		t.Fatalf("thu INSERT: muốn lỗi nêu quyền thiếu, được %v", err)
	}
	if _, err := conn.Exec(ctx, "GRANT INSERT ON decision_records TO eais_app"); err != nil {
		t.Fatal(err)
	}
	if err := s.Ready(ctx); err != nil {
		t.Fatalf("trả quyền rồi mà vẫn chưa sẵn sàng: %v", err)
	}
}

// F-010: Ready đóng mặc định khi cách ly tenant của một bảng ứng dụng bị gỡ (RLS tắt, bỏ FORCE, mất chính sách tenant, thêm chính sách
// không lọc tenant) — một thao tác quản trị sai không được để server khởi động rồi rò / xoá chéo tenant. Chạy trên database riêng của test:
// các kiểu trôi chỉ sống trong nó và được trả lại ngay sau mỗi ca.
func TestReadyDongKhiRLSTat(t *testing.T) {
	_, admin, app := open(t)
	fa, fp := freshDB(t, admin, app)
	ctx := context.Background()
	if err := pgstore.Migrate(ctx, fa); err != nil {
		t.Fatal(err)
	}
	s, err := pgstore.Open(ctx, fp)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	if err := s.Ready(ctx); err != nil {
		t.Fatalf("sau Migrate: %v", err)
	}
	conn := connect(t, fa)
	exec := func(q string) {
		t.Helper()
		if _, err := conn.Exec(ctx, q); err != nil {
			t.Fatalf("%s: %v", q, err)
		}
	}
	check := func(drift, restore, want string) {
		t.Helper()
		exec(drift)
		err := s.Ready(ctx)
		if err == nil || !strings.Contains(err.Error(), want) {
			t.Errorf("%s: Ready phải lỗi nêu %q, được %v", drift, want, err)
		} else if got := contracts.FaultOf(err); got != contracts.Infra("postgres") {
			t.Errorf("%s: nhãn lỗi = %q, muốn %q", drift, got, contracts.Infra("postgres"))
		}
		exec(restore)
		if err := s.Ready(ctx); err != nil {
			t.Errorf("trả lại sau %q mà vẫn chưa sẵn sàng: %v", drift, err)
		}
	}
	const tenantPolicy = "tenant_id = current_setting('eais.tenant_id', true)"
	check("ALTER TABLE facts DISABLE ROW LEVEL SECURITY", "ALTER TABLE facts ENABLE ROW LEVEL SECURITY", "RLS trên facts đang tắt")
	check("ALTER TABLE facts NO FORCE ROW LEVEL SECURITY", "ALTER TABLE facts FORCE ROW LEVEL SECURITY", "RLS trên facts không FORCE")
	check("DROP POLICY tenant_isolation ON facts", "CREATE POLICY tenant_isolation ON facts USING ("+tenantPolicy+") WITH CHECK ("+tenantPolicy+")",
		"facts không có chính sách tenant")
	check("CREATE POLICY mo_toang ON facts USING (true)", "DROP POLICY mo_toang ON facts", "facts có chính sách không lọc tenant: mo_toang")
	check("ALTER POLICY tenant_isolation ON facts WITH CHECK (true)", "ALTER POLICY tenant_isolation ON facts WITH CHECK ("+tenantPolicy+")",
		"facts có chính sách không lọc tenant: tenant_isolation")
	// mọi bảng ứng dụng đụng tới (readyNeeds), không chỉ facts
	for _, table := range []string{"entities", "entity_aliases", "facts", "documents", "decision_records", "quarantine_records", "usage_records",
		"discrepancy_tickets", "commands", "audit_log", "load_records"} {
		check("ALTER TABLE "+table+" DISABLE ROW LEVEL SECURITY", "ALTER TABLE "+table+" ENABLE ROW LEVEL SECURITY", "RLS trên "+table+" đang tắt")
	}
}

// Pool đã đóng thì Ready và Ping báo lỗi (không panic, không treo): healthz dựa vào Ping để trả 503.
func TestReadyPoolDong(t *testing.T) {
	_, _, app := open(t)
	s, err := pgstore.Open(context.Background(), app)
	if err != nil {
		t.Fatal(err)
	}
	s.Close()
	if err := s.Ready(context.Background()); err == nil {
		t.Fatal("pool đã đóng mà Ready không báo lỗi")
	}
	if err := s.Ping(context.Background()); err == nil || contracts.FaultOf(err) != contracts.Infra("postgres") {
		t.Fatalf("pool đã đóng: Ping muốn lỗi nhãn hạ tầng, được %v", err)
	}
}
