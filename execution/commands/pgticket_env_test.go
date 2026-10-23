package commands

// Đồ nghề test CÓ PostgreSQL của kho phiếu (pgticket.go): môi trường, tenant riêng cho từng test, đếm bằng kết nối riêng của
// harness (vai quản trị, ngoài đường ứng dụng, HOP-DONG §10 "Cách đếm"), và một "thế giới" chạy pipeline lệnh trên kho PG với lớp
// bọc probe của K1. Mọi thứ chỉ nằm trong _test.go.
//
// Cách ly giữa các test: mỗi test dùng tenant duy nhất (envTenant) nên không giẫm nhau dù chạy chung database với gói khác; eais_app
// chỉ có SELECT + INSERT nên dọn dòng bằng vai quản trị khi test xong (đặt EAIS_KEEP_TEST_ROWS=1 để giữ lại soi tay).

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"strconv"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/luannt2002/enterprise-ai-control-layer/core/contracts"
	"github.com/luannt2002/enterprise-ai-control-layer/platform/pgstore"
)

var tenantSeq atomic.Int64

// pgEnv là một database đã áp migration, một Store vai ứng dụng (đường chạy thật) và một kết nối vai quản trị (đếm, dọn).
type pgEnv struct {
	t        *testing.T
	adminDSN string
	appDSN   string
	store    *pgstore.Store
	admin    *pgxpool.Pool
	pg       *PGStore

	mu      sync.Mutex
	tenants []string
}

// newPGEnv nối tới PG của deploy/docker-compose.yml (EAIS_PG_ADMIN_DSN / EAIS_PG_DSN). Thiếu biến thì bỏ qua: ở CI có PG, cổng
// kiem-skip-test coi test bị SKIP là ĐỎ.
func newPGEnv(t *testing.T) *pgEnv {
	t.Helper()
	admin, app := pgstore.EnvDSN()
	if admin == "" || app == "" {
		t.Skip("chưa có EAIS_PG_ADMIN_DSN / EAIS_PG_DSN — xem deploy/docker-compose.yml")
	}
	return openPGEnv(t, admin, app, 8)
}

// openPGEnv áp migration lên database của admin rồi mở Store ứng dụng với pool tối đa appConns kết nối.
func openPGEnv(t *testing.T, admin, app string, appConns int) *pgEnv {
	t.Helper()
	ctx := context.Background()
	if err := pgstore.Migrate(ctx, admin); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	st, err := pgstore.Open(ctx, withPoolMax(app, appConns))
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	ad, err := pgxpool.New(ctx, withPoolMax(admin, 3))
	if err != nil {
		st.Close()
		t.Fatalf("kết nối quản trị: %v", err)
	}
	e := &pgEnv{t: t, adminDSN: admin, appDSN: app, store: st, admin: ad, pg: NewPGStore(st)}
	t.Cleanup(func() {
		e.cleanup()
		ad.Close()
		st.Close()
	})
	return e
}

// withPoolMax đặt cỡ pool của pgxpool qua DSN dạng URL; DSN dạng khác giữ nguyên. Chỉ dùng cho pgstore.Open / pgxpool: pgx.Connect
// trần không biết tham số này và từ chối.
func withPoolMax(dsn string, n int) string {
	u, err := url.Parse(dsn)
	if err != nil || u.User == nil {
		return dsn
	}
	q := u.Query()
	q.Set("pool_max_conns", strconv.Itoa(n))
	u.RawQuery = q.Encode()
	return u.String()
}

// tenant cấp một tenant duy nhất cho test và nhớ để dọn.
func (e *pgEnv) tenant(label string) contracts.TenantID {
	id := fmt.Sprintf("k2-%s-%d-%d", label, os.Getpid(), tenantSeq.Add(1))
	e.mu.Lock()
	e.tenants = append(e.tenants, id)
	e.mu.Unlock()
	return contracts.TenantID(id)
}

// cleanup xoá dòng của các tenant đã cấp, bảng con trước (sổ lệnh có khoá ngoại tới phiếu).
func (e *pgEnv) cleanup() {
	if os.Getenv("EAIS_KEEP_TEST_ROWS") != "" {
		return
	}
	e.mu.Lock()
	tenants := append([]string(nil), e.tenants...)
	e.mu.Unlock()
	if len(tenants) == 0 {
		return
	}
	ctx := context.Background()
	for _, table := range []string{"commands", "audit_log", "discrepancy_tickets"} {
		if _, err := e.admin.Exec(ctx, "DELETE FROM "+table+" WHERE tenant_id = ANY($1)", tenants); err != nil {
			fmt.Fprintln(os.Stderr, "dọn dòng test của kho phiếu:", table, err)
		}
	}
}

// ctxOf là ngữ cảnh mang tenant, như C1 gắn.
func ctxOf(tn contracts.TenantID) context.Context {
	return contracts.WithTenant(context.Background(), tn)
}

// --- đếm bằng kết nối riêng của harness (vai quản trị bỏ qua RLS) ---

func (e *pgEnv) countWhere(table string, tn contracts.TenantID, where string, args ...any) int {
	e.t.Helper()
	var n int
	q := "SELECT count(*) FROM " + table + " WHERE tenant_id = $1"
	if where != "" {
		q += " AND (" + where + ")"
	}
	if err := e.admin.QueryRow(context.Background(), q, append([]any{string(tn)}, args...)...).Scan(&n); err != nil {
		e.t.Fatalf("đếm %s: %v", table, err)
	}
	return n
}

func (e *pgEnv) openTickets(tn contracts.TenantID) int {
	e.t.Helper()
	return e.countWhere("discrepancy_tickets", tn, "status = 'OPEN'")
}

func (e *pgEnv) ledgerRows(tn contracts.TenantID) int {
	e.t.Helper()
	return e.countWhere("commands", tn, "")
}

// auditRows đếm dòng audit khớp; outcome, reason hoặc caller rỗng nghĩa là không lọc theo khoá đó.
func (e *pgEnv) auditRows(tn contracts.TenantID, outcome AuditOutcome, reason ReasonCode, caller string) int {
	e.t.Helper()
	return e.countWhere("audit_log", tn,
		"($2 = '' OR outcome = $2) AND ($3 = '' OR reason_code = $3) AND ($4 = '' OR caller_id = $4)",
		string(outcome), string(reason), caller)
}

// ticketIDs trả mã các phiếu của tenant theo thứ tự thời gian tạo.
func (e *pgEnv) ticketIDs(tn contracts.TenantID) []string {
	e.t.Helper()
	rows, err := e.admin.Query(context.Background(), "SELECT id FROM discrepancy_tickets WHERE tenant_id = $1 ORDER BY created_at, id", string(tn))
	if err != nil {
		e.t.Fatal(err)
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			e.t.Fatal(err)
		}
		out = append(out, id)
	}
	if err := rows.Err(); err != nil {
		e.t.Fatal(err)
	}
	return out
}

// --- thế giới chạy pipeline lệnh trên kho PG ---

// pgWorld là một runtime của một tenant (duy nhất cho test) trên kho PG, có lớp bọc probe của K1 và các tay nắm để phá nó. Cùng
// hình dạng với world của fake_test.go, chỉ khác kho và cách đếm.
type pgWorld struct {
	t      *testing.T
	env    *pgEnv
	tenant contracts.TenantID
	probe  *probe
	data   *mutableData
	deps   Deps
	pipe   *Pipeline
}

func newPGWorld(t *testing.T, opts ...func(*Deps)) *pgWorld {
	t.Helper()
	env := newPGEnv(t)
	return env.newWorld(opts...)
}

// newWorld dựng world trên môi trường đã có (nhiều world cùng dùng một database vẫn tách nhau nhờ tenant riêng).
func (e *pgEnv) newWorld(opts ...func(*Deps)) *pgWorld {
	e.t.Helper()
	tn := e.tenant("w")
	pb := &probe{inner: e.pg}
	data := &mutableData{facts: buildFacts(1), dir: buildDirectory(1)}
	deps := Deps{
		Tenant: string(tn), Location: ict, Tool: testTool(), Policy: testPolicy(), Catalog: testCatalog(e.t),
		Facts: data, Directory: data, Store: pb, PackVersion: "pack-test", CodeVersion: "code-test",
		Now: func() time.Time { return fixedNow },
	}
	for _, o := range opts {
		o(&deps)
	}
	w := &pgWorld{t: e.t, env: e, tenant: tn, probe: pb, data: data, deps: deps}
	w.restart()
	return w
}

// user dựng Principal của tenant của world; scope rỗng thì không có entity_scope.
func (w *pgWorld) user(id, role, scope string) contracts.Principal {
	p := contracts.Principal{ID: id, TenantID: w.tenant, Kind: contracts.PrincipalUser, Roles: []string{role}}
	if scope != "" {
		p.Attributes = map[string]string{"entity_scope": scope}
	}
	return p
}

// restart dựng pipeline MỚI trên cùng kho: giả lập khởi động lại tiến trình (F4).
func (w *pgWorld) restart() {
	w.t.Helper()
	p, err := New(w.deps)
	if err != nil {
		w.t.Fatal(err)
	}
	w.pipe = p
}

// other dựng runtime của tenant khác trên CÙNG kho, với dữ liệu hạt giống riêng (A06).
func (w *pgWorld) other() (contracts.TenantID, *Pipeline) {
	w.t.Helper()
	tn := w.env.tenant("o")
	d := w.deps
	d.Tenant = string(tn)
	data := &mutableData{facts: buildFacts(1), dir: buildDirectory(1)}
	d.Facts, d.Directory = data, data
	p, err := New(d)
	if err != nil {
		w.t.Fatal(err)
	}
	return tn, p
}

func (w *pgWorld) run(pr contracts.Principal, r Request) Outcome {
	return w.pipe.Run(context.Background(), pr, r)
}

// race chạy len(reqs) lời gọi SONG SONG qua rào F7 và trả kết quả theo thứ tự khai. Các lời gọi tới rào trước khi mở giao dịch, nên
// khi thả rào cả N cùng vào đơn vị ghi phiếu thật trên PG và chỉ còn khoá duy nhất của PG phân xử.
func (w *pgWorld) race(prs []contracts.Principal, reqs []Request) []Outcome {
	w.t.Helper()
	bar := newBarrier(len(reqs))
	w.probe.mu.Lock()
	w.probe.bar = bar
	w.probe.mu.Unlock()
	out := make([]Outcome, len(reqs))
	var wg sync.WaitGroup
	for i := range reqs {
		wg.Add(1)
		go func() {
			defer wg.Done()
			out[i] = w.pipe.Run(context.Background(), prs[i], reqs[i])
		}()
	}
	wg.Wait()
	if bar.timedOut.Load() {
		w.t.Fatalf("F7: không đủ %d lần tới rào trong hạn (ca ĐỎ, không được bỏ qua)", len(reqs))
	}
	w.probe.mu.Lock()
	w.probe.bar = nil
	w.probe.mu.Unlock()
	return out
}

// check so số đếm ở PG (kết nối quản trị) của một tenant với kỳ vọng; số 0 nghĩa là "đúng 0".
func (w *pgWorld) check(tn contracts.TenantID, c want) {
	w.t.Helper()
	e := w.env
	got := want{
		tickets: e.openTickets(tn), ledger: e.ledgerRows(tn),
		executed:     e.auditRows(tn, AuditExecuted, "", ""),
		replayed:     e.auditRows(tn, AuditReplayed, "", ""),
		refused:      e.auditRows(tn, AuditRefused, "", ""),
		verifyFailed: e.auditRows(tn, AuditVerifyFailed, "", ""),
	}
	if got != c {
		w.t.Fatalf("số đếm ở PG (tenant %s): được %+v, muốn %+v", tn, got, c)
	}
}

func (w *pgWorld) counters(wantWrite, wantAudit int) {
	w.t.Helper()
	wr, au := w.probe.counters()
	if wr != wantWrite || au != wantAudit {
		w.t.Fatalf("bộ đếm lớp bọc: write_unit_calls=%d audit_writes=%d, muốn %d và %d", wr, au, wantWrite, wantAudit)
	}
}

// onlyTicketID trả mã phiếu duy nhất của tenant của world; khác 1 phiếu thì ca đỏ.
func (w *pgWorld) onlyTicketID() string {
	w.t.Helper()
	ids := w.env.ticketIDs(w.tenant)
	if len(ids) != 1 {
		w.t.Fatalf("muốn đúng 1 phiếu ở PG, có %d", len(ids))
	}
	return ids[0]
}
