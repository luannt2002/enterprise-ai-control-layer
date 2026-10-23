package pgstore

// Test của BeginTenantTx (tenanttx.go): mở giao dịch đã đặt tenant, trả giao dịch cho bên gọi tự Commit / Rollback.
// Cần PG như các test pgstore khác; thiếu biến môi trường thì bỏ qua (nhưng cổng kiem-skip-test coi SKIP là ĐỎ ở CI có PG).

import (
	"context"
	"errors"
	"net/url"
	"strconv"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/luannt2002/enterprise-ai-control-layer/core/contracts"
)

// openForTx mở Store với pool đúng maxConns kết nối. Pool một kết nối buộc hai giao dịch liên tiếp dùng CÙNG kết nối vật lý, nên
// GUC nào rò khỏi giao dịch trước sẽ lộ ra ở giao dịch sau.
func openForTx(t *testing.T, maxConns int) *Store {
	t.Helper()
	admin, app := EnvDSN()
	if admin == "" || app == "" {
		t.Skip("chưa có EAIS_PG_ADMIN_DSN / EAIS_PG_DSN — xem deploy/docker-compose.yml")
	}
	u, err := url.Parse(app)
	if err != nil || u.User == nil {
		t.Skipf("DSN ứng dụng không ở dạng URL (docs/ops/README.md) — không đặt được cỡ pool: %v", err)
	}
	q := u.Query()
	q.Set("pool_max_conns", strconv.Itoa(maxConns))
	u.RawQuery = q.Encode()
	ctx := context.Background()
	if err := Migrate(ctx, admin); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	s, err := Open(ctx, u.String())
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	t.Cleanup(s.Close)
	return s
}

// gucOf đọc GUC eais.tenant_id mà một truy vấn trần trên pool thấy (không qua BeginTenantTx): rỗng hoặc NULL là chưa đặt.
func gucOf(t *testing.T, s *Store) string {
	t.Helper()
	var v *string
	if err := s.pool.QueryRow(context.Background(), "SELECT current_setting('eais.tenant_id', true)").Scan(&v); err != nil {
		t.Fatal(err)
	}
	if v == nil {
		return ""
	}
	return *v
}

// Giao dịch trả về đã mang GUC của tenant trong ngữ cảnh, và GUC chỉ sống trong giao dịch (SET LOCAL): hai giao dịch liên tiếp trên
// cùng MỘT kết nối (pool một kết nối) không thấy GUC của nhau, cả sau Commit lẫn sau Rollback.
func TestBeginTenantTxDatGUCVaKhongRoSangGiaoDichSau(t *testing.T) {
	s := openForTx(t, 1)
	a, b := contracts.TenantID("k2-guc-a"), contracts.TenantID("k2-guc-b")
	for i, c := range []struct {
		tenant   contracts.TenantID
		commit   bool
		wantNext string // GUC mà truy vấn trần thấy SAU khi giao dịch kết thúc
	}{{a, true, ""}, {b, false, ""}, {a, false, ""}} {
		ctx := contracts.WithTenant(context.Background(), c.tenant)
		tx, err := s.BeginTenantTx(ctx)
		if err != nil {
			t.Fatalf("lượt %d: %v", i, err)
		}
		defer tx.Rollback(ctx) // ca đỏ giữa chừng không được giữ kết nối khỏi pool (Rollback sau Commit / Rollback chỉ trả ErrTxClosed)
		var got string
		if err := tx.QueryRow(ctx, "SELECT current_setting('eais.tenant_id', true)").Scan(&got); err != nil {
			t.Fatal(err)
		}
		if got != string(c.tenant) {
			t.Errorf("lượt %d: GUC trong giao dịch = %q, muốn %q", i, got, c.tenant)
		}
		if c.commit {
			err = tx.Commit(ctx)
		} else {
			err = tx.Rollback(ctx)
		}
		if err != nil {
			t.Fatalf("lượt %d kết thúc giao dịch: %v", i, err)
		}
		if g := gucOf(t, s); g != c.wantNext {
			t.Errorf("lượt %d: sau khi giao dịch kết thúc, GUC rò sang kết nối = %q, muốn %q", i, g, c.wantNext)
		}
	}
}

// Không có tenant trong ngữ cảnh (hoặc tenant rỗng) thì từ chối bằng lỗi chính sách, không chạm DB, không trả giao dịch.
func TestBeginTenantTxTuChoiKhiThieuTenant(t *testing.T) {
	s := openForTx(t, 1)
	for name, ctx := range map[string]context.Context{
		"không tenant":         context.Background(),
		"tenant rỗng":          contracts.WithTenant(context.Background(), ""),
		"đã huỷ, không tenant": func() context.Context { c, cancel := context.WithCancel(context.Background()); cancel(); return c }(),
	} {
		tx, err := s.BeginTenantTx(ctx)
		if tx != nil { // đáng lẽ không có; trả kết nối về pool để ca đỏ không treo ở các bước sau
			_ = tx.Rollback(ctx)
		}
		if tx != nil || err == nil || contracts.FaultOf(err) != contracts.FaultPolicy {
			t.Errorf("%s: muốn (nil, lỗi %s), được (%v, %v)", name, contracts.FaultPolicy, tx, err)
		}
	}
	// pool vẫn dùng được và không bị giữ kết nối: một giao dịch có tenant mở được ngay trên pool một kết nối
	ctx, cancel := context.WithTimeout(contracts.WithTenant(context.Background(), "k2-guc-a"), 3*time.Second)
	defer cancel()
	tx, err := s.BeginTenantTx(ctx)
	if err != nil {
		t.Fatalf("sau các lần từ chối, pool phải còn dùng được: %v", err)
	}
	if err := tx.Rollback(ctx); err != nil {
		t.Fatal(err)
	}
}

// Lỗi của chính lệnh COMMIT đến nguyên vẹn, KHÔNG bị gom vào nhãn infra (khác inTenant): kho phiếu cần thấy *pgconn.PgError thật
// để phân loại "PG đã trả lỗi, giao dịch đã huỷ". Dựng lỗi ở COMMIT bằng ràng buộc duy nhất HOÃN trên bảng tạm của chính giao dịch.
func TestBeginTenantTxLoiCommitKhongBiGomVaoInfra(t *testing.T) {
	s := openForTx(t, 1)
	ctx := contracts.WithTenant(context.Background(), "k2-guc-a")
	tx, err := s.BeginTenantTx(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback(ctx) // như trên: không giữ kết nối nếu ca đỏ trước COMMIT
	for _, q := range []string{
		"CREATE TEMP TABLE k2_hoan (a int UNIQUE DEFERRABLE INITIALLY DEFERRED) ON COMMIT DROP",
		"INSERT INTO k2_hoan VALUES (1), (1)", // chưa lỗi: ràng buộc hoãn tới COMMIT
	} {
		if _, err := tx.Exec(ctx, q); err != nil {
			t.Fatalf("%s: %v", q, err)
		}
	}
	err = tx.Commit(ctx)
	var pe *pgconn.PgError
	if !errors.As(err, &pe) || pe.Code != "23505" {
		t.Fatalf("COMMIT phải trả *PgError 23505, được %T %v", err, err)
	}
	if label := contracts.FaultOf(err); label != "" {
		t.Errorf("lỗi COMMIT mang nhãn %q: BeginTenantTx không được gom lỗi COMMIT vào infra", label)
	}
}

// Lỗi mở giao dịch (pool đã đóng) mang nhãn hạ tầng postgres, như mọi lỗi kết nối khác của Store, và vẫn mở ra được lỗi gốc.
func TestBeginTenantTxLoiMoGiaoDichCoNhanHaTang(t *testing.T) {
	s := openForTx(t, 1)
	s.Close()
	ctx := contracts.WithTenant(context.Background(), "k2-guc-a")
	tx, err := s.BeginTenantTx(ctx)
	if tx != nil || err == nil {
		t.Fatalf("pool đã đóng: muốn (nil, lỗi), được (%v, %v)", tx, err)
	}
	if got := contracts.FaultOf(err); got != contracts.Infra("postgres") {
		t.Errorf("nhãn lỗi = %q, muốn %q", got, contracts.Infra("postgres"))
	}
	if errors.Unwrap(err) == nil {
		t.Errorf("nhãn phải bọc lỗi gốc để bên gọi phân loại tiếp: %v", err)
	}
}
