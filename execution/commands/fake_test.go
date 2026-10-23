package commands

// Đồ nghề test của gói: thế giới giả lập (danh mục, số liệu, vai), KHO GIẢ trong bộ nhớ có khoá duy nhất như
// PostgreSQL, và lớp bọc quanh kho (bộ đếm, điểm chèn lỗi F1/F3/F5/F6, rào đồng bộ F7). Mọi thứ ở đây chỉ nằm trong
// _test.go: đường chạy thường không có móc nào (HOP-DONG §10).
//
// Số đếm "bằng kết nối riêng của harness" (§10, "Cách đếm"): các hàm count* đọc thẳng memStore, không đi qua pipeline.

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/luannt2002/enterprise-ai-control-layer/core/contracts"
	"github.com/luannt2002/enterprise-ai-control-layer/data/entityresolution"
	"github.com/luannt2002/enterprise-ai-control-layer/data/sourceoftruth"
	"github.com/luannt2002/enterprise-ai-control-layer/intelligence/router"
)

const (
	tenantA = contracts.TenantID("tenant-a")
	tenantB = contracts.TenantID("tenant-b")
	day1    = "2026-10-19" // "hôm qua" của đồng hồ cố định
	day2    = "2026-10-18"
)

var (
	ict      = time.FixedZone("ICT", 7*3600)
	fixedNow = time.Date(2026, 10, 20, 10, 0, 0, 0, ict)
)

// ---------------------------------------------------------------------------
// Thế giới giả lập
// ---------------------------------------------------------------------------

type seedFact struct {
	key, metric, source, number, day string
	approx                           bool
}

// seedFacts: mỗi máy một tính chất để một lớp ca §10 dùng.
var seedFacts = []seedFact{
	// EX-03: hai vế đủ, chênh lệch 68 (ví dụ của hợp đồng §1.1)
	{"EX-03", "issued_fuel", "warehouse_slip", "3850", day1, false},
	{"EX-03", "measured_consumption", "fuel_sensor", "3782", day1, false},
	{"EX-03", "issued_fuel", "warehouse_slip", "100", day2, false},
	{"EX-03", "measured_consumption", "fuel_sensor", "90", day2, false},
	// EX-04, EX-05: hai máy khác nhau cho ca đua khác nội dung (A02b)
	{"EX-04", "issued_fuel", "warehouse_slip", "2000", day1, false},
	{"EX-04", "measured_consumption", "fuel_sensor", "1950", day1, false},
	{"EX-05", "issued_fuel", "warehouse_slip", "1500", day1, false},
	{"EX-05", "measured_consumption", "fuel_sensor", "1400", day1, false},
	// EX-01: hai vế bằng nhau (A14)
	{"EX-01", "issued_fuel", "warehouse_slip", "1000", day1, false},
	{"EX-01", "measured_consumption", "fuel_sensor", "1000", day1, false},
	// EX-02: thiếu vế đo (A15)
	{"EX-02", "issued_fuel", "warehouse_slip", "800", day1, false},
	// EX-06: vế xuất có hai nguồn NGANG thẩm quyền lệch quá dung sai (A16)
	{"EX-06", "issued_fuel", "erp", "3000", day1, false},
	{"EX-06", "issued_fuel", "excel_summary", "3500", day1, false},
	{"EX-06", "measured_consumption", "fuel_sensor", "3000", day1, false},
	// EX-07: như EX-06 và THIẾU vế đo: xung đột phải thắng thiếu vế (A16)
	{"EX-07", "issued_fuel", "erp", "3000", day1, false},
	{"EX-07", "issued_fuel", "excel_summary", "3500", day1, false},
	// EX-08: vế đo là số ước lượng (A17)
	{"EX-08", "issued_fuel", "warehouse_slip", "500", day1, false},
	{"EX-08", "measured_consumption", "fuel_sensor", "480", day1, true},
	// EX-09: máy của dự án B, ngoài phạm vi của người chỉ được giao dự án A (A08)
	{"EX-09", "issued_fuel", "warehouse_slip", "700", day1, false},
	{"EX-09", "measured_consumption", "fuel_sensor", "650", day1, false},
	// EX-10: kho xuất ÍT hơn lượng đo: chênh lệch âm vẫn là chênh lệch
	{"EX-10", "issued_fuel", "warehouse_slip", "900", day1, false},
	{"EX-10", "measured_consumption", "fuel_sensor", "950", day1, false},
}

type seedEntity struct{ key, typ, parent string }

var seedEntities = []seedEntity{
	{"PRJ-A", "project", ""}, {"PRJ-B", "project", ""},
	{"EX-01", "equipment", "PRJ-A"}, {"EX-02", "equipment", "PRJ-A"}, {"EX-03", "equipment", "PRJ-A"},
	{"EX-04", "equipment", "PRJ-A"}, {"EX-05", "equipment", "PRJ-A"}, {"EX-06", "equipment", "PRJ-A"},
	{"EX-07", "equipment", "PRJ-A"}, {"EX-08", "equipment", "PRJ-A"}, {"EX-10", "equipment", "PRJ-A"},
	{"EX-09", "equipment", "PRJ-B"}, {"EX-11", "equipment", "PRJ-A"}, // EX-11 không có số liệu nào
}

// entityID: mã thực thể phụ thuộc "lần nạp" gen, để thử Load thay toàn bộ dữ liệu bằng mã mới (A18).
func entityID(gen int, key string) string { return fmt.Sprintf("g%d-%s", gen, key) }

func buildDirectory(gen int) entityresolution.MemoryDirectory {
	var out entityresolution.MemoryDirectory
	for _, e := range seedEntities {
		ent := entityresolution.Entity{ID: entityID(gen, e.key), Type: e.typ, Key: e.key, Name: e.key}
		if e.parent != "" {
			ent.Parent = entityID(gen, e.parent)
		}
		out = append(out, ent)
	}
	return out
}

func buildFacts(gen int) sourceoftruth.MemoryStore {
	var out sourceoftruth.MemoryStore
	for i, f := range seedFacts {
		on, _ := time.ParseInLocation(time.DateOnly, f.day, ict)
		out = append(out, sourceoftruth.Fact{
			ID: fmt.Sprintf("f%d-%d", gen, i), Metric: f.metric, EntityID: entityID(gen, f.key), OccurredOn: on,
			Value: contracts.Value{Kind: contracts.ValueQuantity, Number: f.number, Unit: "L", Approx: f.approx},
			Source: sourceoftruth.Observation{
				ID: fmt.Sprintf("o%d-%d", gen, i), SourceType: f.source, SourceRef: fmt.Sprintf("%s-%d", f.source, i), ObservedAt: fixedNow,
			},
		})
	}
	return out
}

func testCatalog(t *testing.T) *sourceoftruth.Catalog {
	t.Helper()
	cat, err := sourceoftruth.Compile(
		[]sourceoftruth.MetricDef{
			{ID: "issued_fuel", Label: "Lượng xuất", Unit: "L", EntityTypes: []string{"equipment"}, Tolerance: "10",
				Sources: []sourceoftruth.SourceRank{{Type: "warehouse_slip", Authority: 1}, {Type: "erp", Authority: 2}, {Type: "excel_summary", Authority: 2}}},
			{ID: "measured_consumption", Label: "Lượng đo", Unit: "L", EntityTypes: []string{"equipment"},
				Sources: []sourceoftruth.SourceRank{{Type: "fuel_sensor", Authority: 1}}},
		},
		[]sourceoftruth.DerivedDef{{ID: "issue_vs_measured", Label: "Chênh lệch", Op: "subtract", Left: "issued_fuel", Right: "measured_consumption"}},
		func(string) bool { return true })
	if err != nil {
		t.Fatal(err)
	}
	return cat
}

func testPolicy() router.Policy {
	both := []string{"tickets.create", "metrics.read"}
	return router.Policy{Roles: map[string]router.RoleDef{
		"director":        {Capabilities: both, Scope: "all"},
		"accountant":      {Capabilities: both, Scope: "all"},
		"project_manager": {Capabilities: both, Scope: "attribute"},
		"site_engineer":   {Capabilities: []string{"metrics.read"}, Scope: "attribute"},
		"driver":          {Scope: "attribute"},
	}}
}

func testTool() Tool {
	return Tool{Name: CommandOpenDiscrepancyTicket, Capability: "tickets.create", Metrics: []string{"issue_vs_measured"}, MinAbsDiff: "0"}
}

// user dựng Principal của tenantA; scope rỗng thì không có entity_scope.
func user(id, role, scope string) contracts.Principal {
	p := contracts.Principal{ID: id, TenantID: tenantA, Kind: contracts.PrincipalUser, Roles: []string{role}}
	if scope != "" {
		p.Attributes = map[string]string{"entity_scope": scope}
	}
	return p
}

func cmdID(n int) string { return fmt.Sprintf("6f1c0d52-0000-4000-8000-%012d", n) }

func req(n int, key, day string) Request {
	return Request{CommandID: cmdID(n), SubjectKey: key, Metric: "issue_vs_measured", Day: day}
}

// mutableData là kho số liệu + danh mục có thể bị "nạp lại" giữa chừng (A18) hoặc báo lỗi (lỗi hạ tầng).
type mutableData struct {
	mu       sync.Mutex
	facts    sourceoftruth.MemoryStore
	dir      entityresolution.MemoryDirectory
	factsErr error
	dirErr   error
}

func (d *mutableData) Facts(ctx context.Context, metric string, ids []string, from, to time.Time) ([]sourceoftruth.Fact, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.factsErr != nil {
		return nil, d.factsErr
	}
	return d.facts.Facts(ctx, metric, ids, from, to)
}

func (d *mutableData) Entities(ctx context.Context, types ...string) ([]entityresolution.Entity, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.dirErr != nil {
		return nil, d.dirErr
	}
	return d.dir.Entities(ctx, types...)
}

func (d *mutableData) reload(gen int, facts bool) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.dir = buildDirectory(gen)
	d.facts = nil
	if facts {
		d.facts = buildFacts(gen)
	}
}

// ---------------------------------------------------------------------------
// Kho GIẢ trong bộ nhớ: có khoá duy nhất như PostgreSQL
// ---------------------------------------------------------------------------

type tenantRows struct {
	tickets map[string]TicketRecord
	ledger  map[[2]string]LedgerRecord // (người gọi, command_id): khoá chính của sổ lệnh
	open    map[[3]string]string       // (khoá thực thể, chỉ số, ngày) → mã phiếu: chỉ mục duy nhất từng phần khi OPEN
	audits  []Audit
}

type memStore struct {
	mu            sync.Mutex
	rows          map[contracts.TenantID]*tenantRows
	failAuditInTx int // F2: số lần INSERT audit trong giao dịch bị từ chối
}

func newMemStore() *memStore { return &memStore{rows: map[contracts.TenantID]*tenantRows{}} }

func (m *memStore) rowsOf(tn contracts.TenantID) *tenantRows {
	r := m.rows[tn]
	if r == nil {
		r = &tenantRows{tickets: map[string]TicketRecord{}, ledger: map[[2]string]LedgerRecord{}, open: map[[3]string]string{}}
		m.rows[tn] = r
	}
	return r
}

// scoped bắt chước tenantOf của pgstore: không có tenant trong ngữ cảnh thì từ chối mọi truy vấn.
func (m *memStore) scoped(ctx context.Context) (*tenantRows, error) {
	tn, ok := contracts.TenantFrom(ctx)
	if !ok {
		return nil, errors.New("không có tenant trong ngữ cảnh")
	}
	return m.rowsOf(tn), nil
}

func (m *memStore) FindLedger(ctx context.Context, caller, commandID string) (LedgerRecord, bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	r, err := m.scoped(ctx)
	if err != nil {
		return LedgerRecord{}, false, err
	}
	l, ok := r.ledger[[2]string{caller, commandID}]
	return l, ok, nil
}

func (m *memStore) FindOpenTicket(ctx context.Context, subjectKey, metric, day string) (TicketRef, bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	r, err := m.scoped(ctx)
	if err != nil {
		return TicketRef{}, false, err
	}
	id, ok := r.open[[3]string{subjectKey, metric, day}]
	return TicketRef{ID: id, Status: TicketOpen}, ok, nil
}

// WriteTicket là đơn vị ghi phiếu: tất cả hoặc không gì. Thứ tự và loại lỗi y như PostgreSQL sẽ cho.
func (m *memStore) WriteTicket(ctx context.Context, u WriteUnit) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	r, err := m.scoped(ctx)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrBeforeCommit, err)
	}
	key := [3]string{u.Ticket.SubjectKey, u.Ticket.Metric, u.Ticket.Day}
	if _, dup := r.open[key]; dup && u.Ticket.Status == TicketOpen { // (1) INSERT phiếu
		return fmt.Errorf("%w: chỉ mục duy nhất của phiếu mở", ErrUniqueViolation)
	}
	lk := [2]string{u.Ledger.Caller, u.Ledger.CommandID}
	if _, dup := r.ledger[lk]; dup { // (2) INSERT sổ lệnh
		return fmt.Errorf("%w: khoá chính của sổ lệnh", ErrUniqueViolation)
	}
	if m.failAuditInTx > 0 { // (3) INSERT audit EXECUTED bị từ chối: huỷ cả ba
		m.failAuditInTx--
		return fmt.Errorf("%w: INSERT audit bị từ chối", ErrBeforeCommit)
	}
	u.Ticket.Evidence = slices.Clone(u.Ticket.Evidence)
	r.tickets[u.Ticket.ID] = u.Ticket
	r.open[key] = u.Ticket.ID
	r.ledger[lk] = u.Ledger
	r.audits = append(r.audits, u.Audit)
	return nil
}

func (m *memStore) ReadBack(ctx context.Context, caller, commandID, ticketID string) (Stored, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	r, err := m.scoped(ctx)
	if err != nil {
		return Stored{}, err
	}
	var st Stored
	if t, ok := r.tickets[ticketID]; ok {
		t.Evidence = slices.Clone(t.Evidence)
		st.Tickets = append(st.Tickets, t)
	}
	if l, ok := r.ledger[[2]string{caller, commandID}]; ok {
		st.Ledger = append(st.Ledger, l)
	}
	for _, a := range r.audits {
		if a.Outcome == AuditExecuted && a.Caller.ID == caller && a.CommandID == commandID {
			st.Executed = append(st.Executed, a)
		}
	}
	return st, nil
}

func (m *memStore) WriteAudit(ctx context.Context, a Audit) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	r, err := m.scoped(ctx)
	if err != nil {
		return err
	}
	r.audits = append(r.audits, a)
	return nil
}

// --- đếm bằng "kết nối riêng" của harness ---

func (m *memStore) countOpen(tn contracts.TenantID) int {
	m.mu.Lock()
	defer m.mu.Unlock()
	n := 0
	for _, t := range m.rowsOf(tn).tickets {
		if t.Status == TicketOpen {
			n++
		}
	}
	return n
}

func (m *memStore) countLedger(tn contracts.TenantID) int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.rowsOf(tn).ledger)
}

// audits trả các dòng audit khớp; reason hoặc caller rỗng nghĩa là không lọc theo khoá đó.
func (m *memStore) audits(tn contracts.TenantID, outcome AuditOutcome, reason ReasonCode, caller string) []Audit {
	m.mu.Lock()
	defer m.mu.Unlock()
	var out []Audit
	for _, a := range m.rowsOf(tn).audits {
		if (outcome == "" || a.Outcome == outcome) && (reason == "" || a.ReasonCode == reason) && (caller == "" || a.Caller.ID == caller) {
			out = append(out, a)
		}
	}
	return out
}

func (m *memStore) ticketList(tn contracts.TenantID) []TicketRecord {
	m.mu.Lock()
	defer m.mu.Unlock()
	var out []TicketRecord
	for _, t := range m.rowsOf(tn).tickets {
		out = append(out, t)
	}
	return out
}

// ---------------------------------------------------------------------------
// Lớp bọc quanh kho: bộ đếm, điểm chèn lỗi, rào F7
// ---------------------------------------------------------------------------

type barrier struct {
	n        int
	mu       sync.Mutex
	arrived  int
	release  chan struct{}
	timedOut atomic.Bool
}

func newBarrier(n int) *barrier { return &barrier{n: n, release: make(chan struct{})} }

// wait giữ lời gọi tới khi đủ n lần đã tới rào. Không đủ n trong hạn thì ca ĐỎ (timedOut), không được bỏ qua.
func (b *barrier) wait() error {
	b.mu.Lock()
	b.arrived++
	if b.arrived == b.n {
		close(b.release)
	}
	b.mu.Unlock()
	select {
	case <-b.release:
		return nil
	case <-time.After(5 * time.Second):
		b.timedOut.Store(true)
		return errors.New("rào F7: không đủ số lần tới rào trong hạn")
	}
}

type probe struct {
	inner Store

	mu             sync.Mutex
	writeUnitCalls int // bộ đếm (1): đơn vị ghi phiếu
	auditWrites    int // bộ đếm (2): ghi audit riêng
	f1             int // before_begin
	f3             int // commit_ack_lost: commit THẬT rồi báo UNKNOWN
	f5             int // commit_delayed: báo UNKNOWN ngay, commit đến sau
	f6fail         int // readback_fail
	f6stale        int // readback_stale
	staleMode      string
	delayed        []func()
	bar            *barrier
}

func (p *probe) FindLedger(ctx context.Context, caller, commandID string) (LedgerRecord, bool, error) {
	return p.inner.FindLedger(ctx, caller, commandID)
}

func (p *probe) FindOpenTicket(ctx context.Context, subjectKey, metric, day string) (TicketRef, bool, error) {
	return p.inner.FindOpenTicket(ctx, subjectKey, metric, day)
}

func (p *probe) WriteTicket(ctx context.Context, u WriteUnit) error {
	p.mu.Lock()
	p.writeUnitCalls++
	var f1, f3, f5 bool
	switch {
	case p.f1 > 0:
		p.f1--
		f1 = true
	case p.f3 > 0:
		p.f3--
		f3 = true
	case p.f5 > 0:
		p.f5--
		f5 = true
	}
	bar := p.bar
	p.mu.Unlock()
	if bar != nil { // F7: rào ngay trước đơn vị ghi
		if err := bar.wait(); err != nil {
			return fmt.Errorf("%w: %v", ErrBeforeCommit, err)
		}
	}
	switch {
	case f1:
		return fmt.Errorf("%w: F1 trước BEGIN", ErrBeforeCommit)
	case f5:
		held := context.WithoutCancel(ctx)
		p.mu.Lock()
		p.delayed = append(p.delayed, func() { _ = p.inner.WriteTicket(held, u) })
		p.mu.Unlock()
		return fmt.Errorf("%w: F5 commit đến muộn", ErrCommitUnknown)
	}
	err := p.inner.WriteTicket(ctx, u)
	if err == nil && f3 {
		return fmt.Errorf("%w: F3 mất ack", ErrCommitUnknown)
	}
	return err
}

// releaseDelayed thả các COMMIT bị giữ (F5).
func (p *probe) releaseDelayed() {
	p.mu.Lock()
	fns := p.delayed
	p.delayed = nil
	p.mu.Unlock()
	for _, f := range fns {
		f()
	}
}

func (p *probe) ReadBack(ctx context.Context, caller, commandID, ticketID string) (Stored, error) {
	p.mu.Lock()
	fail, stale, mode := p.f6fail > 0, p.f6stale > 0, p.staleMode
	switch {
	case fail:
		p.f6fail--
	case stale:
		p.f6stale--
	}
	p.mu.Unlock()
	if fail {
		return Stored{}, errors.New("F6: đọc lại hỏng")
	}
	st, err := p.inner.ReadBack(ctx, caller, commandID, ticketID)
	if err != nil || !stale {
		return st, err
	}
	switch mode {
	case "missing": // đọc lại không thấy phiếu
		st.Tickets = nil
	case "diff": // đọc lại thấy phiếu lệch một trường
		st.Tickets[0].Diff = "999"
	case "subject": // đọc lại thấy phiếu mang khoá thực thể khác (kiểm phát lại so với chính yêu cầu)
		st.Tickets[0].SubjectKey = "EX-99"
	default: // "audit-<cột>": dòng audit EXECUTED đọc lại lệch đúng một cột (C2: so đủ mọi cột như HOP-DONG §7.2)
		for i := range st.Executed {
			a := &st.Executed[i]
			switch mode {
			case "audit-fp":
				a.Fingerprint = strings.Repeat("e", 64)
			case "audit-trace":
				a.TraceID = "vet-khac"
			case "audit-caller":
				a.Caller.ID = "p-khac"
			case "audit-roles":
				a.Caller.Roles = []string{"driver"}
			case "audit-day":
				a.Day = "2026-01-01"
			case "audit-action":
				a.Action = "lenh_khac"
			case "audit-reason":
				a.ReasonCode = ReasonPermissionDenied
			}
		}
	}
	return st, nil
}

func (p *probe) WriteAudit(ctx context.Context, a Audit) error {
	p.mu.Lock()
	p.auditWrites++
	p.mu.Unlock()
	return p.inner.WriteAudit(ctx, a)
}

func (p *probe) counters() (writeUnits, audits int) {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.writeUnitCalls, p.auditWrites
}

// ---------------------------------------------------------------------------
// world: một runtime của tenantA + các tay nắm để phá nó
// ---------------------------------------------------------------------------

type world struct {
	t     *testing.T
	store *memStore
	probe *probe
	data  *mutableData
	deps  Deps
	pipe  *Pipeline
}

func newWorld(t *testing.T, opts ...func(*Deps)) *world {
	t.Helper()
	store := newMemStore()
	pb := &probe{inner: store}
	data := &mutableData{facts: buildFacts(1), dir: buildDirectory(1)}
	deps := Deps{
		Tenant: string(tenantA), Location: ict, Tool: testTool(), Policy: testPolicy(), Catalog: testCatalog(t),
		Facts: data, Directory: data, Store: pb, PackVersion: "pack-test", CodeVersion: "code-test",
		Now: func() time.Time { return fixedNow },
	}
	for _, o := range opts {
		o(&deps)
	}
	w := &world{t: t, store: store, probe: pb, data: data, deps: deps}
	w.restart()
	return w
}

// restart dựng pipeline MỚI trên cùng kho: giả lập khởi động lại tiến trình (F4).
func (w *world) restart() {
	w.t.Helper()
	p, err := New(w.deps)
	if err != nil {
		w.t.Fatal(err)
	}
	w.pipe = p
}

// other dựng runtime của tenant khác trên CÙNG kho, với dữ liệu hạt giống riêng (A06).
func (w *world) other(tn contracts.TenantID) *Pipeline {
	w.t.Helper()
	d := w.deps
	d.Tenant = string(tn)
	data := &mutableData{facts: buildFacts(1), dir: buildDirectory(1)}
	d.Facts, d.Directory = data, data
	p, err := New(d)
	if err != nil {
		w.t.Fatal(err)
	}
	return p
}

func (w *world) run(pr contracts.Principal, r Request) Outcome {
	return w.pipe.Run(context.Background(), pr, r)
}

// race chạy len(reqs) lời gọi SONG SONG qua rào F7 và trả kết quả theo thứ tự khai.
func (w *world) race(prs []contracts.Principal, reqs []Request) []Outcome {
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

// want là số đếm kỳ vọng của một tenant; số 0 nghĩa là "đúng 0".
type want struct{ tickets, ledger, executed, replayed, refused, verifyFailed int }

func (w *world) check(tn contracts.TenantID, c want) {
	w.t.Helper()
	got := want{
		tickets: w.store.countOpen(tn), ledger: w.store.countLedger(tn),
		executed:     len(w.store.audits(tn, AuditExecuted, "", "")),
		replayed:     len(w.store.audits(tn, AuditReplayed, "", "")),
		refused:      len(w.store.audits(tn, AuditRefused, "", "")),
		verifyFailed: len(w.store.audits(tn, AuditVerifyFailed, "", "")),
	}
	if got != c {
		w.t.Fatalf("số đếm ở kho (tenant %s): được %+v, muốn %+v", tn, got, c)
	}
}

func (w *world) counters(wantWrite, wantAudit int) {
	w.t.Helper()
	wr, au := w.probe.counters()
	if wr != wantWrite || au != wantAudit {
		w.t.Fatalf("bộ đếm lớp bọc: write_unit_calls=%d audit_writes=%d, muốn %d và %d", wr, au, wantWrite, wantAudit)
	}
}

// ---------------------------------------------------------------------------
// Khẳng định dùng chung
// ---------------------------------------------------------------------------

func wantFail(t *testing.T, o Outcome, status Status, code ReasonCode) {
	t.Helper()
	r := o.Result
	if r.Status != status || r.ReasonCode != code {
		t.Fatalf("muốn %s/%s, được %s/%s (detail=%q, fault=%q)", status, code, r.Status, r.ReasonCode, r.Detail, r.Fault)
	}
	if r.Detail == "" || r.Fault == "" || r.Retry == "" {
		t.Fatalf("kết quả không thành công phải có detail, fault, retry: %+v", r)
	}
	if r.Replayed {
		t.Fatalf("replayed chỉ có nghĩa khi DONE: %+v", r)
	}
	if r.TraceID == "" {
		t.Fatalf("lượt đã xác thực phải có trace_id: %+v", r)
	}
}

func wantDone(t *testing.T, o Outcome, replayed bool) *Ticket {
	t.Helper()
	r := o.Result
	if r.Status != StatusDone || r.ReasonCode != "" || r.Detail != "" || r.Fault != "" || r.Retry != RetryNone || len(r.InvalidFields) != 0 {
		t.Fatalf("muốn DONE sạch, được %+v", r)
	}
	if r.Replayed != replayed {
		t.Fatalf("replayed = %v, muốn %v", r.Replayed, replayed)
	}
	if r.Ticket == nil || r.Ticket.ID == "" || r.Ticket.Status != TicketOpen || r.TraceID == "" {
		t.Fatalf("DONE phải mang phiếu OPEN có mã và trace_id: %+v", r)
	}
	return r.Ticket
}

// assertSteps kiểm hình dạng vết (A29 (a), (b), phần dữ liệu thô): đúng chín bước theo thứ tự, C2 luôn skipped, bước lệch
// đầu tiên đúng chặng wantStop ("" = không có), bước giữa bước dừng và C7 là skipped, C7 không bao giờ skipped.
func assertSteps(t *testing.T, o Outcome, wantStop string) {
	t.Helper()
	codes := []string{"C0", "C1", "C2", "C3", "C4", "C4b", "C5", "C6", "C7"}
	if len(o.Steps) != len(codes) {
		t.Fatalf("vết phải có %d bước, được %d", len(codes), len(o.Steps))
	}
	first := ""
	for i, s := range o.Steps {
		if s.Code != codes[i] {
			t.Fatalf("bước %d là %q, muốn %q", i, s.Code, codes[i])
		}
		if first == "" && (s.Status == StepDeviant || s.Status == StepError) {
			first = s.Code
		}
		if s.Code == "C2" && s.Status != StepSkipped {
			t.Fatalf("C2 luôn skipped, được %q", s.Status)
		}
		if s.Code == "C7" && s.Status == StepSkipped {
			t.Fatalf("C7 không bao giờ skipped")
		}
		if s.Status == StepSkipped && s.Reason == "" {
			t.Fatalf("bước %s skipped phải có lý do", s.Code)
		}
		if s.Status == StepError && s.Fault == "" {
			t.Fatalf("bước %s lỗi phải mang nhãn fault", s.Code)
		}
	}
	if first != wantStop {
		t.Fatalf("bước lệch đầu tiên là %q, muốn %q", first, wantStop)
	}
	if wantStop == "" {
		return
	}
	stopped := false
	for _, s := range o.Steps {
		switch {
		case s.Code == wantStop:
			stopped = true
		case s.Code == "C2" || s.Code == "C7":
		case stopped && s.Status != StepSkipped:
			t.Fatalf("bước %s sau bước dừng %s phải skipped, được %q", s.Code, wantStop, s.Status)
		case !stopped && s.Status != StepOK:
			t.Fatalf("bước %s trước bước dừng %s phải ok, được %q", s.Code, wantStop, s.Status)
		}
	}
}

// routerRole dựng một vai phạm vi "all" chỉ với các capability cho trước (dựng Policy bằng tay cho ca phát triển).
func routerRole(caps ...string) router.RoleDef {
	return router.RoleDef{Capabilities: caps, Scope: "all"}
}

// failReads bọc một Store và làm hỏng đúng chỗ ta chọn: đọc sổ lệnh, tìm phiếu mở, đơn vị ghi phiếu, hoặc treo tới khi hết hạn.
type failReads struct {
	Store
	ledger, open, write error
	block               bool
}

func (f *failReads) FindLedger(ctx context.Context, caller, commandID string) (LedgerRecord, bool, error) {
	switch {
	case f.block:
		<-ctx.Done()
		return LedgerRecord{}, false, ctx.Err()
	case f.ledger != nil:
		return LedgerRecord{}, false, f.ledger
	}
	return f.Store.FindLedger(ctx, caller, commandID)
}

func (f *failReads) FindOpenTicket(ctx context.Context, subjectKey, metric, day string) (TicketRef, bool, error) {
	if f.open != nil {
		return TicketRef{}, false, f.open
	}
	return f.Store.FindOpenTicket(ctx, subjectKey, metric, day)
}

func (f *failReads) WriteTicket(ctx context.Context, u WriteUnit) error {
	if f.write != nil {
		return f.write
	}
	return f.Store.WriteTicket(ctx, u)
}

// onlyTicket trả phiếu duy nhất của tenantA ở kho; khác 1 phiếu thì ca đỏ (không panic, để các ca sau vẫn chạy).
func (w *world) onlyTicket() TicketRecord {
	w.t.Helper()
	list := w.store.ticketList(tenantA)
	if len(list) != 1 {
		w.t.Fatalf("muốn đúng 1 phiếu ở kho, có %d", len(list))
	}
	return list[0]
}
