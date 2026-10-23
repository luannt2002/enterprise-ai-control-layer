package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/luannt2002/enterprise-ai-control-layer/core/contracts"
	"github.com/luannt2002/enterprise-ai-control-layer/data/entityresolution"
	"github.com/luannt2002/enterprise-ai-control-layer/data/sourceoftruth"
	"github.com/luannt2002/enterprise-ai-control-layer/intelligence/rag"
	"github.com/luannt2002/enterprise-ai-control-layer/observability/traces"
	"github.com/luannt2002/enterprise-ai-control-layer/platform/pgstore"
	"github.com/luannt2002/enterprise-ai-control-layer/tenants/construction/win/gen"
)

// Hạ tầng test của cmd/eais. Nguyên tắc: MỌI tenant trong test là tenant riêng của lượt chạy (uniqTenant) — các gói test khác dùng
// chung PG (vd winrun nạp tenant thật của pack vào PG), nên test này không bao giờ ghi vào tenant thật của pack.

const (
	seedTenant   = "abc-infra" // tên thư mục dữ liệu mà bộ sinh xuất; test chép nó sang tenant riêng của lượt chạy
	clockForTest = "2026-10-20T10:00:00+07:00"
	qSQL         = "Hôm qua máy 03 được cấp bao nhiêu dầu?"                          // route số liệu: không gọi model
	qRAG         = "Biên bản đối soát chênh lệch nhiên liệu phải lập trong bao lâu?" // route tri thức: extractive gọi 1 lần
)

func repoPath(rel string) string { return filepath.Join("..", "..", rel) }

var (
	seedOnce sync.Once
	seedPath string
	seedErr  error
	// usedTenants: tenant mà test đã cấp, để TestMain dọn dòng của chúng ở PG bằng vai quản trị (eais_app không có DELETE trên bảng vết).
	usedTenants sync.Map
)

// seed trả thư mục dữ liệu seed 42 (hồ sơ functional) do bộ sinh có sẵn xuất, sinh MỘT lần mỗi lượt chạy test.
func seed(t *testing.T) string {
	t.Helper()
	seedOnce.Do(func() {
		if seedPath, seedErr = os.MkdirTemp("", "eais-cmd-seed-"); seedErr == nil {
			seedErr = gen.Generate(repoPath("tenants/construction/win/world.yaml"), seedPath, "functional", 42)
		}
	})
	if seedErr != nil {
		t.Fatalf("sinh dữ liệu seed: %v", seedErr)
	}
	return seedPath
}

func uniqTenant(prefix string) string {
	id := fmt.Sprintf("%s-%d", prefix, time.Now().UnixNano())
	usedTenants.Store(id, struct{}{})
	return id
}

func copyTree(t *testing.T, src, dst string) {
	t.Helper()
	err := filepath.WalkDir(src, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(src, p)
		target := filepath.Join(dst, rel)
		if d.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		b, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		return os.WriteFile(target, b, 0o644)
	})
	if err != nil {
		t.Fatal(err)
	}
}

// tenantFixtures dựng pack và thư mục dữ liệu của MỘT tenant riêng: chép pack thật (đổi dòng tenant ở pack.yaml) và dữ liệu seed.
func tenantFixtures(t *testing.T, tenantID string) (packDir, dataRoot string) {
	t.Helper()
	tmp := t.TempDir()
	packDir, dataRoot = filepath.Join(tmp, "pack"), filepath.Join(tmp, "data")
	copyTree(t, repoPath("tenants/construction/pack"), packDir)
	copyTree(t, filepath.Join(seed(t), seedTenant), filepath.Join(dataRoot, tenantID))
	p := filepath.Join(packDir, "pack.yaml")
	b, err := os.ReadFile(p)
	if err != nil {
		t.Fatal(err)
	}
	out := strings.Replace(string(b), "tenant: "+seedTenant, "tenant: "+tenantID, 1)
	if out == string(b) {
		t.Fatal("pack.yaml không có dòng tenant để đổi")
	}
	if err := os.WriteFile(p, []byte(out), 0o644); err != nil {
		t.Fatal(err)
	}
	return packDir, dataRoot
}

// env là một app + server HTTP thử trên kho st, với hai tenant: A (được app phục vụ, có dữ liệu) và B (chỉ có principal, như tenant
// canary), cùng các principal mẫu theo đúng vai của policy.yaml.
type env struct {
	t                *testing.T
	st               store
	app              *app
	srv              *server
	ts               *httptest.Server
	tenantA, tenantB string
	packDir          string
	dataRoot         string
	logs             *syncBuffer
	load             ingestReport
	// decisionRows đếm dòng vết của một tenant ở kho đang thử (bộ nhớ: đếm map; PG: đếm bằng vai quản trị, bỏ qua RLS).
	decisionRows func(tenant string) int
}

const (
	tokDirector   = "t-director"   // tenant A, vai director (toàn quyền dữ liệu)
	tokAccountant = "t-accountant" // tenant A, scope all nhưng không có documents.read ⇒ KHÔNG toàn quyền
	tokEng        = "t-eng"        // tenant A, site_engineer, phạm vi PRJ-A
	tokDriver     = "t-driver"     // tenant A, driver: không capability nào
	tokDirectorB  = "t-director-b" // tenant B, vai director
	tokPM         = "t-pm"         // tenant A, project_manager, phạm vi PRJ-A (có tickets.create; soát K3, K3S-03)
)

type syncBuffer struct {
	mu sync.Mutex
	b  bytes.Buffer
}

func (s *syncBuffer) Write(p []byte) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.b.Write(p)
}
func (s *syncBuffer) String() string { s.mu.Lock(); defer s.mu.Unlock(); return s.b.String() }

// newEnv dựng env trên kho st và nạp dữ liệu seed cho tenant A (qua chính app.ingest, như lệnh `eais ingest`).
func newEnv(t *testing.T, st store) *env {
	t.Helper()
	a, b := uniqTenant("e2e-a"), uniqTenant("e2e-b")
	packDir, dataRoot := tenantFixtures(t, a)
	logs := &syncBuffer{}
	log := slog.New(slog.NewTextHandler(logs, nil))
	ap, err := newApp(config{Pack: packDir, Data: dataRoot, Now: clockForTest}, st, log)
	if err != nil {
		t.Fatalf("newApp: %v", err)
	}
	rep, err := ap.ingest(context.Background())
	if err != nil {
		t.Fatalf("ingest ban đầu: %v", err)
	}
	user := func(id string, tn string, role, scope string) contracts.Principal {
		p := contracts.Principal{ID: id, TenantID: contracts.TenantID(tn), Kind: contracts.PrincipalUser, Roles: []string{role}}
		if scope != "" {
			p.Attributes = map[string]string{"entity_scope": scope}
		}
		return p
	}
	srv := newServer(ap, map[string]contracts.Principal{
		tokDirector:   user("u-director", a, "director", ""),
		tokAccountant: user("u-accountant", a, "accountant", ""),
		tokEng:        user("u-eng-a", a, "site_engineer", "PRJ-A"),
		tokDriver:     user("u-driver", a, "driver", "EX-03"),
		tokDirectorB:  user("u-b-director", b, "director", ""),
		tokPM:         user("u-pm-a", a, "project_manager", "PRJ-A"),
	}, log)
	srv.hostGuard = true
	ts := httptest.NewServer(srv.handler())
	t.Cleanup(ts.Close)
	return &env{t: t, st: st, app: ap, srv: srv, ts: ts, tenantA: a, tenantB: b, packDir: packDir, dataRoot: dataRoot, logs: logs, load: rep}
}

// serverLog trả toàn bộ log của server thử.
func (e *env) serverLog() string { return e.logs.String() }

type reply struct {
	status int
	body   []byte
	header http.Header
}

func (e *env) do(method, path, token, contentType string, body []byte) reply {
	e.t.Helper()
	req, err := http.NewRequest(method, e.ts.URL+path, bytes.NewReader(body))
	if err != nil {
		e.t.Fatal(err)
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		e.t.Fatal(err)
	}
	defer res.Body.Close()
	b, _ := io.ReadAll(res.Body)
	return reply{res.StatusCode, b, res.Header}
}

func (e *env) post(path, token string, body any) reply {
	e.t.Helper()
	b, err := json.Marshal(body)
	if err != nil {
		e.t.Fatal(err)
	}
	return e.do(http.MethodPost, path, token, "application/json", b)
}

func (e *env) ask(token, text string) reply {
	e.t.Helper()
	return e.post("/v1/ask", token, map[string]string{"text": text, "conversation_id": "c-1", "message_id": "m-1"})
}

func (e *env) trace(token, id string) reply {
	e.t.Helper()
	return e.do(http.MethodGet, "/v1/trace/"+id, token, "", nil)
}

func decodeAs[T any](t *testing.T, r reply) T {
	t.Helper()
	var v T
	if err := json.Unmarshal(r.body, &v); err != nil {
		t.Fatalf("thân không phải JSON: %v\n%s", err, r.body)
	}
	return v
}

// ---------------------------------------------------------------------------------------------------------------------------------
// memStore: hiện thực store trong bộ nhớ, chia theo tenant lấy từ ngữ cảnh như RLS của PG, để test lớp HTTP chạy được không cần PG
// và để chèn lỗi (ghi vết thất bại, panic trong luồng hỏi). Nó KHÔNG thay test có PG: RLS, ràng buộc và quyền vai là của PG.
// ---------------------------------------------------------------------------------------------------------------------------------

type memStore struct {
	mu         sync.Mutex
	entities   map[string][]entityresolution.Entity
	facts      map[string]sourceoftruth.MemoryStore
	docs       map[string][]rag.Document
	decisions  map[string]map[string]traces.Record
	usage      map[string]map[string]pgstore.UsageRecord // tenant → "trace|model"
	quarantine map[string]map[string]pgstore.QuarantineRecord

	// chèn lỗi
	saveDecisionErrs []error         // trả lần lượt cho các lần SaveDecision (hết thì thật)
	saveAttempts     []traces.Record // mọi bản ghi mà SaveDecision đã nhận, kể cả lần lỗi
	panicEntities    bool
	pingErr          error
	touched          int // số lần gọi vào kho (để chứng minh một đường KHÔNG chạm kho)
}

func newMemStore() *memStore {
	return &memStore{entities: map[string][]entityresolution.Entity{}, facts: map[string]sourceoftruth.MemoryStore{}, docs: map[string][]rag.Document{},
		decisions: map[string]map[string]traces.Record{}, usage: map[string]map[string]pgstore.UsageRecord{}, quarantine: map[string]map[string]pgstore.QuarantineRecord{}}
}

// ctxErr mô phỏng pgx: ngữ cảnh đã huỷ thì lệnh lỗi, nhãn hạ tầng postgres như pgstore.inTenant gắn cho mọi lỗi kết nối.
func ctxErr(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return contracts.Label(contracts.Infra("postgres"), err)
	}
	return nil
}

func memTenant(ctx context.Context) (string, error) {
	t, ok := contracts.TenantFrom(ctx)
	if !ok {
		return "", contracts.Errorf(contracts.FaultPolicy, "truy vấn không mang tenant trong ngữ cảnh")
	}
	return string(t), nil
}

func (m *memStore) Entities(ctx context.Context, types ...string) ([]entityresolution.Entity, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.touched++
	if m.panicEntities {
		panic("giả lập panic trong luồng hỏi")
	}
	t, err := memTenant(ctx)
	if err != nil {
		return nil, err
	}
	var out []entityresolution.Entity
	for _, e := range m.entities[t] {
		if len(types) == 0 || slices.Contains(types, e.Type) {
			out = append(out, e)
		}
	}
	return out, nil
}

func (m *memStore) Facts(ctx context.Context, metric string, ids []string, from, to time.Time) ([]sourceoftruth.Fact, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.touched++
	t, err := memTenant(ctx)
	if err != nil {
		return nil, err
	}
	return m.facts[t].Facts(ctx, metric, ids, from, to)
}

func (m *memStore) Retrieve(ctx context.Context, pr contracts.Principal, query string, k int) ([]contracts.Evidence, error) {
	m.mu.Lock()
	t, err := memTenant(ctx)
	var docs []rag.Document
	for _, d := range m.docs[t] {
		for _, r := range pr.Roles {
			if slices.Contains(d.ACL, r) {
				docs = append(docs, d)
				break
			}
		}
	}
	m.touched++
	m.mu.Unlock()
	if err != nil {
		return nil, err
	}
	return rag.NewMemoryIndex(t, docs).Retrieve(ctx, pr, query, k)
}

func (m *memStore) Load(ctx context.Context, es []entityresolution.Entity, fs []sourceoftruth.Fact, ds []rag.Document) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.touched++
	t, err := memTenant(ctx)
	if err != nil {
		return err
	}
	m.entities[t], m.facts[t], m.docs[t] = slices.Clone(es), sourceoftruth.MemoryStore(slices.Clone(fs)), slices.Clone(ds)
	return nil
}

func (m *memStore) SaveDecision(ctx context.Context, rec traces.Record) (bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.touched++
	if err := ctxErr(ctx); err != nil {
		return false, err
	}
	m.saveAttempts = append(m.saveAttempts, rec)
	if len(m.saveDecisionErrs) > 0 {
		err := m.saveDecisionErrs[0]
		m.saveDecisionErrs = m.saveDecisionErrs[1:]
		if err != nil {
			return false, err
		}
	}
	t, err := memTenant(ctx)
	if err != nil {
		return false, err
	}
	if err := rec.Validate(); err != nil {
		return false, contracts.Errorf(contracts.FaultInput, err.Error())
	}
	if rec.Decision.Tenant != t {
		return false, contracts.Errorf(contracts.FaultPolicy, "bản ghi vết mang tenant khác tenant của ngữ cảnh")
	}
	clean, err := traces.SanitizeRecord(rec)
	if err != nil {
		return false, err
	}
	if m.decisions[t] == nil {
		m.decisions[t] = map[string]traces.Record{}
	}
	if old, dup := m.decisions[t][clean.Decision.TraceID]; dup {
		a, _ := old.Digest()
		b, _ := clean.Digest()
		if a != b {
			return false, contracts.Errorf(contracts.FaultData, "trace_id đã có bản ghi với nội dung khác; không ghi đè")
		}
		return false, nil
	}
	m.decisions[t][clean.Decision.TraceID] = clean
	return true, nil
}

func (m *memStore) Decision(ctx context.Context, id string) (traces.Record, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.touched++
	if err := ctxErr(ctx); err != nil {
		return traces.Record{}, err
	}
	t, err := memTenant(ctx)
	if err != nil {
		return traces.Record{}, err
	}
	rec, ok := m.decisions[t][id]
	if !ok {
		return traces.Record{}, contracts.Label(contracts.FaultInput, fmt.Errorf("%w: trace %q", pgstore.ErrNotFound, id))
	}
	return rec, nil
}

func (m *memStore) SaveUsage(ctx context.Context, u pgstore.UsageRecord) (bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.touched++
	if err := ctxErr(ctx); err != nil {
		return false, err
	}
	t, err := memTenant(ctx)
	if err != nil {
		return false, err
	}
	if _, ok := m.decisions[t][u.TraceID]; !ok {
		return false, contracts.Errorf(contracts.FaultInput, "usage của trace "+u.TraceID+" khi chưa có bản ghi quyết định")
	}
	if m.usage[t] == nil {
		m.usage[t] = map[string]pgstore.UsageRecord{}
	}
	k := u.TraceID + "|" + u.Model
	if _, dup := m.usage[t][k]; dup {
		return false, nil
	}
	m.usage[t][k] = u
	return true, nil
}

func (m *memStore) Usage(ctx context.Context, id string) ([]pgstore.UsageRecord, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.touched++
	t, err := memTenant(ctx)
	if err != nil {
		return nil, err
	}
	var out []pgstore.UsageRecord
	for _, u := range m.usage[t] {
		if u.TraceID == id {
			out = append(out, u)
		}
	}
	return out, nil
}

func (m *memStore) SaveQuarantine(ctx context.Context, loadID string, recs []pgstore.QuarantineRecord) (int, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.touched++
	t, err := memTenant(ctx)
	if err != nil {
		return 0, err
	}
	if m.quarantine[t] == nil {
		m.quarantine[t] = map[string]pgstore.QuarantineRecord{}
	}
	n := 0
	for _, r := range recs {
		k := loadID + "|" + r.Key + "|" + r.Reason
		if _, dup := m.quarantine[t][k]; !dup {
			m.quarantine[t][k] = r
			n++
		}
	}
	return n, nil
}

func (m *memStore) Quarantine(ctx context.Context, loadID string) ([]pgstore.QuarantineRecord, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.touched++
	t, err := memTenant(ctx)
	if err != nil {
		return nil, err
	}
	var out []pgstore.QuarantineRecord
	for k, r := range m.quarantine[t] {
		if strings.HasPrefix(k, loadID+"|") {
			out = append(out, r)
		}
	}
	return out, nil
}

func (m *memStore) Ping(context.Context) error  { m.mu.Lock(); defer m.mu.Unlock(); return m.pingErr }
func (m *memStore) Ready(context.Context) error { return m.Ping(context.Background()) }

// decisionCount đếm bản ghi vết của một tenant (kiểm "không có dòng vết" cho lượt chưa xác thực).
func (m *memStore) decisionCount(tenant string) int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.decisions[tenant])
}

var _ store = (*memStore)(nil)

// ---------------------------------------------------------------------------------------------------------------------------------
// PG thật (chỉ khi có EAIS_PG_*), và dọn dòng của test.
// ---------------------------------------------------------------------------------------------------------------------------------

// openPG mở PG của biến môi trường, migrate bằng vai quản trị. Thiếu biến thì bỏ qua (ĐỎ-MT, không phải đạt): CI có PG nên cấm SKIP.
func openPG(t *testing.T) (*pgstore.Store, string) {
	t.Helper()
	admin, app := pgstore.EnvDSN()
	if admin == "" || app == "" {
		t.Skip("chưa có EAIS_PG_ADMIN_DSN / EAIS_PG_DSN — xem deploy/docker-compose.yml")
	}
	ctx := context.Background()
	if err := pgstore.Migrate(ctx, admin); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	st, err := pgstore.Open(ctx, app)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	t.Cleanup(st.Close)
	return st, admin
}

// countRows đếm dòng của bảng cho một tenant bằng vai quản trị (bỏ qua RLS) — để kiểm cái mà vai ứng dụng không thấy được.
func countRows(t *testing.T, admin, table, tenantID string, where string, args ...any) int {
	t.Helper()
	ctx := context.Background()
	conn, err := pgx.Connect(ctx, admin)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close(ctx)
	q := "SELECT count(*) FROM " + table + " WHERE tenant_id = $1"
	if where != "" {
		q += " AND " + where
	}
	var n int
	if err := conn.QueryRow(ctx, q, append([]any{tenantID}, args...)...).Scan(&n); err != nil {
		t.Fatal(err)
	}
	return n
}

func TestMain(m *testing.M) {
	code := m.Run()
	if seedPath != "" {
		os.RemoveAll(seedPath)
	}
	cleanupPGRows()
	os.Exit(code)
}

// cleanupPGRows xoá dòng của các tenant test đã cấp, bằng vai quản trị (eais_app chỉ có SELECT + INSERT trên bảng vết). Đặt
// EAIS_KEEP_TEST_ROWS=1 để giữ lại soi bằng tay.
func cleanupPGRows() {
	admin, _ := pgstore.EnvDSN()
	if admin == "" || os.Getenv("EAIS_KEEP_TEST_ROWS") != "" {
		return
	}
	var tenants []string
	usedTenants.Range(func(k, _ any) bool { tenants = append(tenants, k.(string)); return true })
	if len(tenants) == 0 {
		return
	}
	ctx := context.Background()
	conn, err := pgx.Connect(ctx, admin)
	if err != nil {
		fmt.Fprintln(os.Stderr, "dọn dòng test: không kết nối được:", err)
		return
	}
	defer conn.Close(ctx)
	for _, table := range []string{"usage_records", "quarantine_records", "decision_records", "entity_aliases", "facts", "documents", "entities"} {
		if _, err := conn.Exec(ctx, "DELETE FROM "+table+" WHERE tenant_id = ANY($1)", tenants); err != nil {
			fmt.Fprintln(os.Stderr, "dọn dòng test:", table, err)
		}
	}
}
