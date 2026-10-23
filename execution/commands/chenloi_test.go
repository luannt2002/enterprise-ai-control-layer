package commands

// Lớp chèn lỗi TEST-ONLY của HOP-DONG §10 và §12.1: chín điểm chèn lỗi bằng một lớp bọc quanh lớp lưu trữ (faultStore) và một rào
// đồng bộ F7 (rao), cùng một bộ điều khiển dùng chung cho CẢ CA (faultCtl). Đường chạy thường không có móc nào: mọi thứ ở đây chỉ
// nằm trong _test.go (xem chú thích đầu pipeline.go).
//
// Luật áp dụng (§12.1): trừ F7, điểm chèn lỗi áp cho lời gọi ĐẦU TIÊN, theo thứ tự chạy, ĐI TỚI điểm đó, và đúng MỘT lần; các lời gọi
// sau chạy thường. "Đi tới điểm" nghĩa là tới đúng chỗ code thật sẽ chạm điểm đó, nên một lời gọi bị chặn từ trước (bị từ chối ở
// C4, phát lại ở C4b...) không "đi tới" F1, F2, F3, F5 và không làm mất lượt bắn của lời gọi sau. faultCtl.reach là nơi duy nhất
// quyết định điều đó; test của tệp này chứng minh từng điểm bắn đúng chỗ và đúng một lần.
//
// Bộ đếm của lớp bọc thuộc harness, tích luỹ suốt ca (kể cả qua lần dựng pipeline mới của F4) và tách theo runtime: mỗi runtime
// một faultStore, nhưng chung MỘT faultCtl của ca.
//
// F8 (trace_save_fail) ở bộ chạy chỉ bắn ở chỗ lưu GIẢ của chính bộ chạy (memTraceSaver qua traceSaver, bên dưới): ca A29e kiểm sổ sách của
// bộ chạy, KHÔNG chứng minh gì về đường lưu vết thật. K3 đã gộp (3892be2) nhưng persistTrace nằm ở gói main cmd/eais, bộ chạy (gói
// execution/commands) không gọi tới được; A29 (e) của mã thật do test của K3 đo — TestLenhHTTPMatVetKhongDoiKetQua (lỗi lưu vết không đổi kết
// quả đã ghi, có đột biến M25a). Soát K4, K4S-01: đừng ghi F8 là "đã bắn đúng chỗ" ở bằng chứng X4 / X9.

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/luannt2002/enterprise-ai-control-layer/core/contracts"
)

// Chín tên điểm chèn lỗi, đúng chữ ở bảng §10 và ở khoá failure_injection.point của khuôn ca (§12.1).
const (
	fpBeforeBegin     = "before_begin"       // F1: lỗi trước khi mở giao dịch
	fpInTxAuditInsert = "in_tx_audit_insert" // F2: INSERT audit trong giao dịch bị từ chối
	fpCommitAckLost   = "commit_ack_lost"    // F3: commit THẬT rồi báo lỗi cho pipeline
	fpReplyLost       = "reply_lost"         // F4: bên gọi không nhận kết quả; dựng pipeline mới
	fpCommitDelayed   = "commit_delayed"     // F5: commit đến sau khi pipeline đã báo UNKNOWN
	fpReadbackFail    = "readback_fail"      // F6: đọc lại trả lỗi
	fpReadbackStale   = "readback_stale"     // F6: đọc lại trả thiếu (bản sao chậm)
	fpRaceBarrier     = "race_barrier"       // F7: rào đồng bộ ngay trước đơn vị ghi phiếu
	fpTraceSaveFail   = "trace_save_fail"    // F8: kho vết lỗi khi lưu vết của lượt
)

var faultPoints = []string{fpBeforeBegin, fpInTxAuditInsert, fpCommitAckLost, fpReplyLost, fpCommitDelayed,
	fpReadbackFail, fpReadbackStale, fpRaceBarrier, fpTraceSaveFail}

func validFaultPoint(p string) bool {
	for _, x := range faultPoints {
		if x == p {
			return true
		}
	}
	return false
}

// raoWait là hạn chờ mặc định của rào F7: không đủ N lời gọi tới rào trong hạn này thì ca ĐỎ (không được bỏ qua, §10).
const raoWait = 5 * time.Second

// rao là rào F7: giữ N lời gọi cho tới khi cả N đã tới, rồi thả CÙNG LÚC.
type rao struct {
	n        int
	wait     time.Duration
	mu       sync.Mutex
	arrived  int
	released bool
	release  chan struct{}
	timedOut atomic.Bool
}

func newRao(n int, wait time.Duration) *rao {
	return &rao{n: n, wait: wait, release: make(chan struct{})}
}

// await giữ lời gọi tới khi đủ n lần đã tới rào. Hết hạn thì trả lỗi và đánh dấu timedOut để bộ chạy làm ca ĐỎ.
func (b *rao) await() error {
	b.mu.Lock()
	b.arrived++
	if b.arrived == b.n {
		b.released = true
		close(b.release)
	}
	b.mu.Unlock()
	select {
	case <-b.release:
		return nil
	case <-time.After(b.wait):
		b.timedOut.Store(true)
		return fmt.Errorf("rào F7: chỉ %d trong %d lần tới rào trong hạn %s", b.arrivedCount(), b.n, b.wait)
	}
}

func (b *rao) arrivedCount() int {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.arrived
}

func (b *rao) isReleased() bool {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.released
}

// faultCtl là điểm chèn lỗi của MỘT ca: chọn đúng một điểm, nhớ đã bắn chưa, giữ rào F7 và các COMMIT bị giữ của F5.
type faultCtl struct {
	point string
	rao   *rao // chỉ khi point = race_barrier

	mu                sync.Mutex
	fired             bool
	reached           map[string]int // số lần có lời gọi ĐI TỚI mỗi điểm (để test chứng minh)
	held              []func()
	traceSaveFailures int
	restarts          int
}

// newFaultCtl dựng bộ điều khiển. point rỗng là ca không chèn lỗi. parallel chỉ có nghĩa với race_barrier và là số lời gọi N của rào.
func newFaultCtl(point string, parallel int, wait time.Duration) (*faultCtl, error) {
	if point != "" && !validFaultPoint(point) {
		return nil, fmt.Errorf("điểm chèn lỗi lạ %q (chín tên hợp lệ: %v)", point, faultPoints)
	}
	if point == fpRaceBarrier && parallel < 2 {
		return nil, fmt.Errorf("race_barrier cần parallel >= 2, được %d", parallel)
	}
	if point != fpRaceBarrier && parallel != 0 {
		return nil, fmt.Errorf("parallel chỉ dùng với race_barrier (điểm %q)", point)
	}
	c := &faultCtl{point: point, reached: map[string]int{}}
	if point == fpRaceBarrier {
		c.rao = newRao(parallel, wait)
	}
	return c, nil
}

// reach báo một lời gọi vừa ĐI TỚI điểm p. Trả true đúng MỘT lần cho cả ca: lần đầu tiên p là điểm đã chọn. Mọi lời gọi khác (và mọi
// điểm khác) chạy thường. race_barrier không đi qua đây: rào tác động lên cả N lời gọi.
func (c *faultCtl) reach(p string) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.reached[p]++
	if p != c.point || c.fired {
		return false
	}
	c.fired = true
	return true
}

func (c *faultCtl) reachedCount(p string) int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.reached[p]
}

func (c *faultCtl) didFire() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.fired
}

// hold giữ một COMMIT lại (F5); releaseHeld thả nó.
func (c *faultCtl) hold(fn func()) {
	c.mu.Lock()
	c.held = append(c.held, fn)
	c.mu.Unlock()
}

// releaseHeld thả các COMMIT bị giữ. Bộ chạy gọi ngay sau khi lời gọi đã trả kết quả, trước khi lời gọi kế tiếp bắt đầu (§12.1).
func (c *faultCtl) releaseHeld() {
	c.mu.Lock()
	fns := c.held
	c.held = nil
	c.mu.Unlock()
	for _, f := range fns {
		f()
	}
}

// faultStore bọc một Store: bộ đếm của lớp bọc và các điểm chèn lỗi nằm ở chính lớp lưu trữ (F1, F3, F5, F6, F7). F2 nằm TRONG đơn vị
// ghi của kho gốc (giữa INSERT sổ lệnh và INSERT audit) nên do kho gốc cài (xem armF2 ở bochay_test.go); F4 và F8 do bộ chạy.
type faultStore struct {
	inner Store
	ctl   *faultCtl

	mu             sync.Mutex
	writeUnitCalls int // bộ đếm (1): số lần pipeline gọi đơn vị ghi phiếu, kể cả khi lần gọi hỏng vì F1, F2
	auditWrites    int // bộ đếm (2): số lần pipeline ghi audit riêng
	innerWrites    int // số lần đơn vị ghi của kho GỐC thật sự được gọi (F1 không bao giờ làm số này tăng)
}

var _ Store = (*faultStore)(nil)

func (s *faultStore) bump(f func()) { s.mu.Lock(); f(); s.mu.Unlock() }

func (s *faultStore) counters() (writeUnits, audits int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.writeUnitCalls, s.auditWrites
}

func (s *faultStore) FindLedger(ctx context.Context, caller, commandID string) (LedgerRecord, bool, error) {
	return s.inner.FindLedger(ctx, caller, commandID)
}

func (s *faultStore) FindOpenTicket(ctx context.Context, subjectKey, metric, day string) (TicketRef, bool, error) {
	return s.inner.FindOpenTicket(ctx, subjectKey, metric, day)
}

// WriteTicket là đơn vị ghi phiếu của lớp bọc. Thứ tự các điểm theo đúng thứ tự thật: bộ đếm tăng ngay lúc gọi; rào F7 đứng ngay
// trước mọi thứ khác (đầu C6, trước khi mở giao dịch); rồi F1 (trước BEGIN); rồi F5 (commit bị giữ); rồi kho gốc; rồi F3 (mất ack).
func (s *faultStore) WriteTicket(ctx context.Context, u WriteUnit) error {
	s.bump(func() { s.writeUnitCalls++ })
	if s.ctl.rao != nil {
		if err := s.ctl.rao.await(); err != nil {
			return fmt.Errorf("%w: %v", ErrBeforeCommit, err)
		}
	}
	if s.ctl.reach(fpBeforeBegin) { // F1
		return fmt.Errorf("%w: F1 before_begin: lớp lưu trữ báo lỗi trước khi mở giao dịch", ErrBeforeCommit)
	}
	if s.ctl.reach(fpCommitDelayed) { // F5: toàn bộ đơn vị ghi bị giữ, pipeline thấy UNKNOWN; thả sau khi lời gọi trả kết quả
		held := context.WithoutCancel(ctx)
		s.ctl.hold(func() {
			s.bump(func() { s.innerWrites++ })
			_ = s.inner.WriteTicket(held, u)
		})
		return fmt.Errorf("%w: F5 commit_delayed: COMMIT đến sau khi pipeline đã báo UNKNOWN", ErrCommitUnknown)
	}
	s.bump(func() { s.innerWrites++ })
	err := s.inner.WriteTicket(ctx, u)
	if err == nil && s.ctl.reach(fpCommitAckLost) { // F3: giao dịch ĐÃ commit, lớp lưu trữ báo lỗi
		return fmt.Errorf("%w: F3 commit_ack_lost: đã commit nhưng ack bị mất", ErrCommitUnknown)
	}
	return err
}

// ReadBack: F6. readback_fail trả lỗi; readback_stale trả "chưa thấy gì" như đọc từ một bản sao chậm (thiếu phiếu, sổ lệnh và audit).
func (s *faultStore) ReadBack(ctx context.Context, caller, commandID, ticketID string) (Stored, error) {
	if s.ctl.reach(fpReadbackFail) {
		return Stored{}, errors.New("F6 readback_fail: đọc lại hỏng")
	}
	if s.ctl.reach(fpReadbackStale) {
		return Stored{}, nil
	}
	return s.inner.ReadBack(ctx, caller, commandID, ticketID)
}

func (s *faultStore) WriteAudit(ctx context.Context, a Audit) error {
	s.bump(func() { s.auditWrites++ })
	return s.inner.WriteAudit(ctx, a)
}

// traceSaver là chỗ lưu vết của một lượt, SAU C7 (HOP-DONG §7.5, "Mất vết"): lưu hỏng thì kết quả đã ghi KHÔNG đổi, chỉ mất trace_id và tăng bộ đếm lỗi lưu vết.
// Đường lưu vết thật (persistTrace ở cmd/eais, kho decision_records) là của K3 và nằm ngoài tầm của bộ chạy (gói main); bộ chạy dùng memTraceSaver làm
// chỗ lưu giả, và faultTraceSaver bọc nó theo ĐÚNG giao diện này để F8 bắn ở chỗ lưu giả đó (xem chú thích đầu tệp: không chứng minh mã thật).
type traceSaver interface {
	Save(ctx context.Context, out Outcome) error
}

// memTraceSaver ghi nhớ trace_id đã lưu.
type memTraceSaver struct {
	mu    sync.Mutex
	saved []string
}

func (m *memTraceSaver) Save(_ context.Context, out Outcome) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.saved = append(m.saved, out.Result.TraceID)
	return nil
}

// faultTraceSaver bọc một traceSaver: F8 (trace_save_fail) làm lần lưu ĐẦU TIÊN hỏng (đúng một lần), các lần sau chạy thường.
type faultTraceSaver struct {
	inner traceSaver
	ctl   *faultCtl
}

func (f *faultTraceSaver) Save(ctx context.Context, out Outcome) error {
	if f.ctl.reach(fpTraceSaveFail) {
		return errors.New("F8 trace_save_fail: kho vết lỗi khi lưu vết của lượt")
	}
	return f.inner.Save(ctx, out)
}

// ---------------------------------------------------------------------------
// Test: từng điểm bắn đúng chỗ, đúng một lần
// ---------------------------------------------------------------------------

// faultWorld là một runtime của tenantA trên kho giả trong bộ nhớ, bọc bằng faultStore, với MỘT điểm chèn lỗi.
type faultWorld struct {
	t    *testing.T
	mem  *memStore
	fs   *faultStore
	ctl  *faultCtl
	pipe *Pipeline
}

func newFaultWorld(t *testing.T, point string, parallel int, wait time.Duration) *faultWorld {
	t.Helper()
	ctl, err := newFaultCtl(point, parallel, wait)
	if err != nil {
		t.Fatal(err)
	}
	mem := newMemStore()
	if point == fpInTxAuditInsert {
		mem.failAuditInTx = 1 // F2 của kho giả: bắn ở INSERT audit, sau hai INSERT đầu (xem memStore.WriteTicket)
	}
	fs := &faultStore{inner: mem, ctl: ctl}
	data := &mutableData{facts: buildFacts(1), dir: buildDirectory(1)}
	p, err := New(Deps{
		Tenant: string(tenantA), Location: ict, Tool: testTool(), Policy: testPolicy(), Catalog: testCatalog(t),
		Facts: data, Directory: data, Store: fs, PackVersion: "pack-test", CodeVersion: "code-test",
		Now: func() time.Time { return fixedNow },
	})
	if err != nil {
		t.Fatal(err)
	}
	return &faultWorld{t: t, mem: mem, fs: fs, ctl: ctl, pipe: p}
}

func (w *faultWorld) run(pr contracts.Principal, r Request) Outcome {
	return w.pipe.Run(context.Background(), pr, r)
}

func TestChenLoiChinTenDiem(t *testing.T) {
	want := []string{"before_begin", "in_tx_audit_insert", "commit_ack_lost", "reply_lost", "commit_delayed",
		"readback_fail", "readback_stale", "race_barrier", "trace_save_fail"}
	if len(faultPoints) != len(want) {
		t.Fatalf("muốn đúng chín điểm, có %d", len(faultPoints))
	}
	for i, p := range want {
		if faultPoints[i] != p || !validFaultPoint(p) {
			t.Fatalf("điểm %d: %q, muốn %q", i, faultPoints[i], p)
		}
	}
	for _, bad := range []string{"", "before-begin", "BEFORE_BEGIN", "f1", "commit"} {
		if validFaultPoint(bad) {
			t.Fatalf("tên %q không được hợp lệ", bad)
		}
	}
	if _, err := newFaultCtl("điểm_lạ", 0, time.Second); err == nil {
		t.Fatal("điểm lạ phải bị từ chối")
	}
	if _, err := newFaultCtl(fpRaceBarrier, 1, time.Second); err == nil {
		t.Fatal("race_barrier với parallel < 2 phải bị từ chối")
	}
	if _, err := newFaultCtl(fpBeforeBegin, 3, time.Second); err == nil {
		t.Fatal("parallel chỉ dùng với race_barrier")
	}
}

// reach: đúng một lần cho cả ca, chỉ ở điểm đã chọn.
func TestChenLoiReachDungMotLan(t *testing.T) {
	c, err := newFaultCtl(fpReadbackFail, 0, time.Second)
	if err != nil {
		t.Fatal(err)
	}
	if c.reach(fpBeforeBegin) || c.reach(fpReadbackStale) {
		t.Fatal("điểm không được chọn không được bắn")
	}
	if !c.reach(fpReadbackFail) {
		t.Fatal("lần đầu tới điểm đã chọn phải bắn")
	}
	if c.reach(fpReadbackFail) || c.reach(fpReadbackFail) {
		t.Fatal("điểm đã bắn rồi không được bắn lần nữa")
	}
	if c.reachedCount(fpReadbackFail) != 3 || !c.didFire() {
		t.Fatalf("thống kê đi-tới: được %d, muốn 3", c.reachedCount(fpReadbackFail))
	}
}

// F1: bắn TRƯỚC khi kho gốc được gọi (innerWrites = 0) nhưng bộ đếm (1) vẫn tăng; lời gọi sau chạy thường.
func TestChenLoiF1BanTruocKhiMoGiaoDich(t *testing.T) {
	w := newFaultWorld(t, fpBeforeBegin, 0, time.Second)
	dir := user("p-1", "director", "")
	o := w.run(dir, req(1, "EX-03", day1))
	wantFail(t, o, StatusFailed, ReasonStoreUnavailable)
	if w.fs.innerWrites != 0 || w.mem.countOpen(tenantA) != 0 || w.mem.countLedger(tenantA) != 0 {
		t.Fatalf("F1 phải bắn trước kho gốc: innerWrites=%d phiếu=%d sổ lệnh=%d", w.fs.innerWrites, w.mem.countOpen(tenantA), w.mem.countLedger(tenantA))
	}
	if wr, _ := w.fs.counters(); wr != 1 {
		t.Fatalf("bộ đếm (1) tăng ngay lúc gọi kể cả khi F1: được %d, muốn 1", wr)
	}
	wantDone(t, w.run(dir, req(1, "EX-03", day1)), false) // gửi lại cùng khoá: chạy thường
	if w.fs.innerWrites != 1 || w.mem.countOpen(tenantA) != 1 {
		t.Fatalf("lời gọi sau F1 phải chạy thường: innerWrites=%d phiếu=%d", w.fs.innerWrites, w.mem.countOpen(tenantA))
	}
	if w.ctl.reachedCount(fpBeforeBegin) != 2 {
		t.Fatalf("hai lời gọi đã đi tới F1, được %d", w.ctl.reachedCount(fpBeforeBegin))
	}
}

// Lời gọi bị chặn từ trước (từ chối ở C4) KHÔNG đi tới F1, nên không làm mất lượt bắn của lời gọi sau.
func TestChenLoiLoiGoiBiChanTruocKhongLamMatLuotBan(t *testing.T) {
	w := newFaultWorld(t, fpBeforeBegin, 0, time.Second)
	wantFail(t, w.run(user("p-7", "site_engineer", "PRJ-A"), req(1, "EX-03", day1)), StatusFailed, ReasonPermissionDenied) // dừng ở C4
	if w.ctl.didFire() || w.ctl.reachedCount(fpBeforeBegin) != 0 {
		t.Fatal("lời gọi bị từ chối ở C4 không được đi tới F1")
	}
	wantFail(t, w.run(user("p-1", "director", ""), req(2, "EX-03", day1)), StatusFailed, ReasonStoreUnavailable) // lời gọi sau mới là lượt bắn
	if !w.ctl.didFire() {
		t.Fatal("F1 phải bắn ở lời gọi đầu tiên đi tới nó")
	}
}

// F2 (kho giả): bắn BÊN TRONG kho gốc (innerWrites = 1, khác F1) và huỷ cả ba dòng; chỉ bắn khi hai INSERT đầu đã qua.
func TestChenLoiF2BanTrongDonViGhi(t *testing.T) {
	w := newFaultWorld(t, fpInTxAuditInsert, 0, time.Second)
	dir := user("p-1", "director", "")
	wantFail(t, w.run(dir, req(1, "EX-03", day1)), StatusFailed, ReasonStoreUnavailable)
	if w.fs.innerWrites != 1 {
		t.Fatalf("F2 bắn trong kho gốc: innerWrites=%d, muốn 1", w.fs.innerWrites)
	}
	w.mem.mu.Lock()
	left := w.mem.failAuditInTx
	w.mem.mu.Unlock()
	if left != 0 {
		t.Fatalf("lần INSERT audit đã tiêu thụ lượt bắn, còn %d", left)
	}
	if w.mem.countOpen(tenantA) != 0 || w.mem.countLedger(tenantA) != 0 || len(w.mem.audits(tenantA, AuditExecuted, "", "")) != 0 {
		t.Fatal("F2 phải huỷ cả ba dòng: phiếu, sổ lệnh, audit EXECUTED")
	}
	wantDone(t, w.run(dir, req(1, "EX-03", day1)), false) // lượt gửi lại chạy thường
	if w.mem.countOpen(tenantA) != 1 {
		t.Fatal("lời gọi sau F2 phải chạy thường")
	}
}

// F2 không bắn khi INSERT sổ lệnh đã hỏng trước: nó chỉ bắn ở INSERT audit, tức SAU hai INSERT đầu.
func TestChenLoiF2KhongBanKhiChuaToiInsertAudit(t *testing.T) {
	w := newFaultWorld(t, fpInTxAuditInsert, 0, time.Second)
	dir := user("p-1", "director", "")
	// lượt 1 hỏng ở INSERT audit (tiêu thụ lượt bắn), lượt 2 ghi thật phiếu EX-03
	wantFail(t, w.run(dir, req(1, "EX-03", day1)), StatusFailed, ReasonStoreUnavailable)
	wantDone(t, w.run(dir, req(2, "EX-03", day1)), false)
	w.mem.mu.Lock()
	w.mem.failAuditInTx = 1 // nạp lại lượt bắn để thử vị trí
	w.mem.mu.Unlock()
	// đơn vị ghi cùng khoá lệnh với lượt 2 nhưng KHÁC máy: INSERT phiếu qua, INSERT sổ lệnh vi phạm khoá chính ⇒ không tới INSERT audit
	u := WriteUnit{
		Ticket: TicketRecord{ID: "t-x", Status: TicketOpen, SubjectKey: "EX-04", Metric: "issue_vs_measured", Day: day1, CommandID: cmdID(2), CreatedBy: "p-1"},
		Ledger: LedgerRecord{Caller: "p-1", CommandID: cmdID(2), Fingerprint: "f", TicketID: "t-x"},
	}
	err := w.mem.WriteTicket(contracts.WithTenant(context.Background(), tenantA), u)
	if !errors.Is(err, ErrUniqueViolation) {
		t.Fatalf("muốn vi phạm khoá duy nhất ở INSERT sổ lệnh, được %v", err)
	}
	w.mem.mu.Lock()
	left := w.mem.failAuditInTx
	w.mem.mu.Unlock()
	if left != 1 {
		t.Fatalf("hỏng ở INSERT sổ lệnh không được tiêu thụ lượt bắn F2: còn %d, muốn 1", left)
	}
}

// F3: kho gốc ĐÃ commit (phiếu có thật) nhưng pipeline thấy UNKNOWN; lượt gửi lại phát lại.
func TestChenLoiF3CommitRoiMoiBaoLoi(t *testing.T) {
	w := newFaultWorld(t, fpCommitAckLost, 0, time.Second)
	dir := user("p-1", "director", "")
	wantFail(t, w.run(dir, req(1, "EX-03", day1)), StatusUnknown, ReasonCommitUnknown)
	if w.mem.countOpen(tenantA) != 1 || w.mem.countLedger(tenantA) != 1 || w.fs.innerWrites != 1 {
		t.Fatalf("F3 phải bắn SAU commit: phiếu=%d sổ lệnh=%d innerWrites=%d", w.mem.countOpen(tenantA), w.mem.countLedger(tenantA), w.fs.innerWrites)
	}
	wantDone(t, w.run(dir, req(1, "EX-03", day1)), true)
	if w.mem.countOpen(tenantA) != 1 {
		t.Fatal("gửi lại sau F3 không được tạo phiếu thứ hai")
	}
}

// F5: COMMIT bị giữ (chưa có gì ở kho) cho tới khi thả; thả xong thì có đúng một phiếu và lượt gửi lại phát lại.
func TestChenLoiF5CommitDenMuon(t *testing.T) {
	w := newFaultWorld(t, fpCommitDelayed, 0, time.Second)
	dir := user("p-1", "director", "")
	wantFail(t, w.run(dir, req(1, "EX-03", day1)), StatusUnknown, ReasonCommitUnknown)
	if w.mem.countOpen(tenantA) != 0 || w.fs.innerWrites != 0 {
		t.Fatalf("COMMIT bị giữ: kho chưa được có gì, được phiếu=%d innerWrites=%d", w.mem.countOpen(tenantA), w.fs.innerWrites)
	}
	w.ctl.releaseHeld()
	if w.mem.countOpen(tenantA) != 1 || w.fs.innerWrites != 1 {
		t.Fatalf("sau khi thả: phải có đúng 1 phiếu, được phiếu=%d innerWrites=%d", w.mem.countOpen(tenantA), w.fs.innerWrites)
	}
	wantDone(t, w.run(dir, req(1, "EX-03", day1)), true)
	if w.mem.countOpen(tenantA) != 1 {
		t.Fatal("tổng cộng đúng một phiếu, không hai")
	}
}

// F6 readback_fail: phiếu đã commit, đọc lại lỗi ⇒ UNKNOWN; chỉ lượt đầu.
func TestChenLoiF6DocLaiLoi(t *testing.T) {
	w := newFaultWorld(t, fpReadbackFail, 0, time.Second)
	dir := user("p-1", "director", "")
	wantFail(t, w.run(dir, req(1, "EX-03", day1)), StatusUnknown, ReasonReadbackUnavailable)
	if w.mem.countOpen(tenantA) != 1 {
		t.Fatal("F6 bắn SAU commit: phiếu phải có")
	}
	wantDone(t, w.run(dir, req(1, "EX-03", day1)), true) // lượt gửi lại: đọc lại chạy thường
	if w.ctl.reachedCount(fpReadbackFail) != 2 {
		t.Fatalf("hai lượt đã đi tới đọc lại, được %d", w.ctl.reachedCount(fpReadbackFail))
	}
}

// F6 readback_stale: đọc lại thấy thiếu ⇒ UNKNOWN VERIFY_MISMATCH và audit VERIFY_FAILED, không bao giờ DONE.
func TestChenLoiF6DocLaiCu(t *testing.T) {
	w := newFaultWorld(t, fpReadbackStale, 0, time.Second)
	dir := user("p-1", "director", "")
	wantFail(t, w.run(dir, req(1, "EX-03", day1)), StatusUnknown, ReasonVerifyMismatch)
	if n := len(w.mem.audits(tenantA, AuditVerifyFailed, "", "")); n != 1 {
		t.Fatalf("audit VERIFY_FAILED: được %d, muốn 1", n)
	}
	if w.mem.countOpen(tenantA) != 1 {
		t.Fatal("phiếu đã commit; chỉ phần đọc lại bị cũ")
	}
}

// F7: cả N lời gọi cùng tới rào TRƯỚC khi lời gọi nào ghi; sau đó chỉ khoá duy nhất của kho phân xử.
func TestChenLoiF7ThaCungLuc(t *testing.T) {
	const n = 3
	w := newFaultWorld(t, fpRaceBarrier, n, 5*time.Second)
	out := make([]Outcome, n)
	var wg sync.WaitGroup
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			out[i] = w.run(user("p-1", "director", ""), req(10+i, "EX-03", day1)) // khác khoá lệnh, cùng (máy, chỉ số, ngày)
		}()
	}
	wg.Wait()
	if w.ctl.rao.arrivedCount() != n || !w.ctl.rao.isReleased() || w.ctl.rao.timedOut.Load() {
		t.Fatalf("rào phải giữ đủ %d lời gọi rồi thả: đến=%d thả=%v hết hạn=%v", n, w.ctl.rao.arrivedCount(), w.ctl.rao.isReleased(), w.ctl.rao.timedOut.Load())
	}
	if wr, _ := w.fs.counters(); wr != n {
		t.Fatalf("cả %d lời gọi phải tới đơn vị ghi (không lần nào bị chặn trước C6): được %d", n, wr)
	}
	done, dup := 0, 0
	for _, o := range out {
		switch {
		case o.Result.Status == StatusDone:
			done++
		case o.Result.ReasonCode == ReasonDuplicateOpenTicket:
			dup++
		}
	}
	if done != 1 || dup != n-1 || w.mem.countOpen(tenantA) != 1 {
		t.Fatalf("muốn 1 DONE và %d DUPLICATE_OPEN_TICKET, được %d và %d; phiếu=%d", n-1, done, dup, w.mem.countOpen(tenantA))
	}
}

// F7: không đủ N lời gọi tới rào trong hạn thì ca ĐỎ, không được bỏ qua.
func TestChenLoiF7KhongDuLoiGoiLaDo(t *testing.T) {
	w := newFaultWorld(t, fpRaceBarrier, 2, 150*time.Millisecond)
	o := w.run(user("p-1", "director", ""), req(1, "EX-03", day1)) // chỉ một lời gọi tới rào
	if !w.ctl.rao.timedOut.Load() {
		t.Fatal("rào phải ghi nhận hết hạn để ca ĐỎ")
	}
	wantFail(t, o, StatusFailed, ReasonStoreUnavailable)
	if w.mem.countOpen(tenantA) != 0 {
		t.Fatal("lời gọi bị rào chặn không được ghi gì")
	}
}

// F2 trên PostgreSQL: móc nằm GIỮA INSERT sổ lệnh và INSERT audit trong cùng giao dịch. Chứng minh bằng ba đơn vị ghi: (1) ghi bình thường;
// (2) cùng khoá lệnh khác máy: INSERT phiếu qua, INSERT sổ lệnh vi phạm khoá chính nên KHÔNG tới INSERT audit và không tiêu thụ lượt bắn;
// (3) đơn vị mới: móc bắn đúng một lần, cả ba dòng bị huỷ (phiếu và sổ lệnh đã INSERT cũng mất); gửi lại thì ghi bình thường.
func TestChenLoiF2PGBanGiuaInsertSoLenhVaInsertAudit(t *testing.T) {
	e := newScratchEnv(t, "chenloi")
	tn := e.tenant("f2")
	ctl, err := newFaultCtl(fpInTxAuditInsert, 0, time.Second)
	if err != nil {
		t.Fatal(err)
	}
	mustWrite(t, e, tn, richUnit(1, "p-1", "EX-03")) // (1) chưa gắn móc
	e.pg.beforeAuditInsert = func() error {          // cùng móc mà bộ chạy gắn cho điểm in_tx_audit_insert (pgBackend.arm)
		if ctl.reach(fpInTxAuditInsert) {
			return errors.New("F2 in_tx_audit_insert: INSERT audit bị từ chối giữa giao dịch")
		}
		return nil
	}
	defer func() { e.pg.beforeAuditInsert = nil }()
	if err := e.pg.WriteTicket(ctxOf(tn), richUnit(1, "p-1", "EX-04")); !errors.Is(err, ErrUniqueViolation) { // (2)
		t.Fatalf("muốn vi phạm khoá chính của sổ lệnh, được %v", err)
	}
	if ctl.reachedCount(fpInTxAuditInsert) != 0 || ctl.didFire() {
		t.Fatal("hỏng ở INSERT sổ lệnh thì không được tới móc F2 (móc đứng SAU hai INSERT đầu)")
	}
	u := richUnit(2, "p-1", "EX-05") // (3)
	if err := e.pg.WriteTicket(ctxOf(tn), u); !errors.Is(err, ErrBeforeCommit) {
		t.Fatalf("F2 phải làm đơn vị ghi hỏng TRƯỚC COMMIT, được %v", err)
	}
	if ctl.reachedCount(fpInTxAuditInsert) != 1 || !ctl.didFire() {
		t.Fatalf("móc phải được gọi đúng một lần, được %d", ctl.reachedCount(fpInTxAuditInsert))
	}
	if e.openTickets(tn) != 1 || e.ledgerRows(tn) != 1 || e.auditRows(tn, AuditExecuted, "", "") != 1 {
		t.Fatalf("F2 huỷ cả ba dòng của đơn vị bị hỏng: phiếu=%d sổ lệnh=%d EXECUTED=%d, muốn 1 mỗi loại (chỉ đơn vị (1))",
			e.openTickets(tn), e.ledgerRows(tn), e.auditRows(tn, AuditExecuted, "", ""))
	}
	mustWrite(t, e, tn, u) // gửi lại: móc đã bắn rồi, chạy thường
	if ctl.reachedCount(fpInTxAuditInsert) != 2 || e.openTickets(tn) != 2 {
		t.Fatalf("lần gửi lại ghi bình thường: móc được gọi %d lần, phiếu=%d", ctl.reachedCount(fpInTxAuditInsert), e.openTickets(tn))
	}
}

// F8: bắn ở chỗ lưu vết, ở lần lưu đầu tiên, đúng một lần; lần hỏng không để lại trace_id đã lưu.
func TestChenLoiF8BanOLanLuuVetDauTien(t *testing.T) {
	ctl, err := newFaultCtl(fpTraceSaveFail, 0, time.Second)
	if err != nil {
		t.Fatal(err)
	}
	mem := &memTraceSaver{}
	var sv traceSaver = &faultTraceSaver{inner: mem, ctl: ctl}
	if err := sv.Save(context.Background(), Outcome{Result: Result{TraceID: "t-1"}}); err == nil {
		t.Fatal("lần lưu vết đầu tiên phải hỏng")
	}
	if err := sv.Save(context.Background(), Outcome{Result: Result{TraceID: "t-2"}}); err != nil {
		t.Fatalf("lần lưu sau chạy thường: %v", err)
	}
	if len(mem.saved) != 1 || mem.saved[0] != "t-2" || ctl.reachedCount(fpTraceSaveFail) != 2 {
		t.Fatalf("chỉ lượt sau được lưu: %v (đi tới %d lần)", mem.saved, ctl.reachedCount(fpTraceSaveFail))
	}
}
