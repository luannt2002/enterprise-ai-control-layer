package commands

// Ca của vòng sửa bước K1 theo bản soát độc lập vòng 1, nhóm chống trùng và lỗi mơ hồ: K1S-01 (cửa sổ giữa C4b và R5), K1S-16 (bước C6
// của lượt DONE sau phân loại lại), K1S-03 (lỗi mang nhiều dấu loại), K1S-04 (panic). Dùng kho giả, không PostgreSQL, không model; cách
// đếm số dòng và các móc dùng chung nằm ở fake_test.go, ở đây chỉ thêm những lớp bọc riêng của nhóm ca.

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/luannt2002/enterprise-ai-control-layer/core/contracts"
	"github.com/luannt2002/enterprise-ai-control-layer/data/entityresolution"
	"github.com/luannt2002/enterprise-ai-control-layer/data/sourceoftruth"
)

// ---------------------------------------------------------------------------
// K1S-01: cửa sổ giữa C4b và R5
// ---------------------------------------------------------------------------

// twinStore bọc Store để dựng ĐÚNG cửa sổ giữa C4b và R5 mà không cần rào: sau lần FindLedger thứ `after` của lượt đang thử (lần tra
// sổ lệnh ở C4b, đã trả lời "không thấy") nó chạy `action`, thường là một bản sinh đôi chạy trọn lượt và commit. Từ lần thứ `failFrom`
// nó trả lỗi (lần tra lại ở R5). Bản sinh đôi phải chạy trên pipeline KHÔNG đi qua lớp bọc này, để không tự đếm vào.
type twinStore struct {
	Store
	calls    int
	after    int
	action   func()
	failFrom int
	failErr  error
}

func (s *twinStore) arm(after int, action func()) { s.calls, s.after, s.action = 0, after, action }

func (s *twinStore) failAt(from int, err error) { s.calls, s.failFrom, s.failErr = 0, from, err }

func (s *twinStore) FindLedger(ctx context.Context, caller, commandID string) (LedgerRecord, bool, error) {
	s.calls++
	if s.failFrom > 0 && s.calls >= s.failFrom {
		return LedgerRecord{}, false, s.failErr
	}
	led, found, err := s.Store.FindLedger(ctx, caller, commandID)
	if s.after > 0 && s.calls == s.after && s.action != nil {
		s.action() // chạy SAU khi lần tra này đã trả lời xong: đúng khoảng giữa C4b và R5
	}
	return led, found, err
}

// twinWorld dựng thế giới có lớp bọc twinStore và một pipeline thứ hai chạy thẳng trên kho giả (không qua lớp bọc) làm bản sinh đôi.
func twinWorld(t *testing.T) (*world, *twinStore, *Pipeline) {
	t.Helper()
	var raw Store
	hook := &twinStore{}
	w := newWorld(t, func(d *Deps) { raw, hook.Store, d.Store = d.Store, d.Store, hook })
	d := w.deps
	d.Store = raw
	twin, err := New(d)
	if err != nil {
		t.Fatal(err)
	}
	return w, hook, twin
}

// Bản gửi trước CÙNG khoá, CÙNG nội dung chạy xong và commit sau khi C4b của lượt này đã trả lời "không thấy sổ lệnh". R5 thấy phiếu OPEN
// của chính bản đó. Kết quả đúng là DONE phát lại (§6.3 hàng 3, I5): bên gọi không được nhận DUPLICATE_OPEN_TICKET cho phiếu của chính mình
// (§3: không phân biệt được "lần trước của tôi đã thành công" với "người khác đã tạo").
func TestR5ThayPhieuCuaBanSinhDoiThiPhatLai(t *testing.T) {
	w, hook, twin := twinWorld(t)
	p, r := user("p-1", "director", ""), req(900, "EX-03", day1)
	var first Outcome
	hook.arm(1, func() { first = twin.Run(context.Background(), p, r) })

	second := w.run(p, r)
	tk := wantDone(t, first, false)
	again := wantDone(t, second, true)
	if again.ID != tk.ID {
		t.Fatalf("phát lại phải trả đúng phiếu của bản sinh đôi: %s và %s", again.ID, tk.ID)
	}
	assertSteps(t, second, "") // lượt DONE kể cả phát lại không có bước lệch
	if c4b := second.Steps[stLedger]; c4b.Status != StepOK || c4b.Decision != "phát lại" {
		t.Fatalf("C4b phải ok và ghi 'phát lại': %+v", c4b)
	}
	if second.Steps[stRules].Status != StepSkipped || second.Steps[stWrite].Status != StepSkipped {
		t.Fatalf("phát lại: C5 và C6 skipped; được %+v", second.Steps[stRules:stVerify])
	}
	w.check(tenantA, want{tickets: 1, ledger: 1, executed: 1, replayed: 1})
	w.counters(1, 1) // đơn vị ghi phiếu chỉ do bản sinh đôi gọi; lượt này chỉ ghi audit REPLAYED
}

// Biến thể khác nội dung: R5 thấy phiếu OPEN của một lệnh khác, nhưng lúc đó sổ lệnh đã có dòng của CHÍNH command_id này với nội dung
// khác. Khoá bị dùng lại cho yêu cầu khác nội dung thì kết quả đúng là IDEMPOTENCY_KEY_REUSED, y như khi hai lượt chạy tuần tự (C4b
// bắt trước R5); không được báo DUPLICATE_OPEN_TICKET.
func TestR5ThayPhieuMoKhiKhoaVuaBiDungChoNoiDungKhacThiLaKhoaDungLai(t *testing.T) {
	w, hook, twin := twinWorld(t)
	other := user("p-0", "accountant", "")
	wantDone(t, w.run(other, req(899, "EX-04", day1)), false) // phiếu OPEN sẵn cho (EX-04, ngày): người khác, khoá khác
	p := user("p-1", "director", "")
	rA, rB := req(900, "EX-03", day1), req(900, "EX-04", day1) // cùng khoá, khác nội dung
	hook.arm(1, func() { wantDone(t, twin.Run(context.Background(), p, rA), false) })

	b := w.run(p, rB)
	wantFail(t, b, StatusFailed, ReasonIdempotencyKeyReused)
	if b.Result.Fault != "input" || b.Result.Retry != RetryFixRequest || b.Result.Ticket != nil {
		t.Fatalf("IDEMPOTENCY_KEY_REUSED: fault input, retry fix_request, không phiếu: %+v", b.Result)
	}
	assertSteps(t, b, "C4b")
	w.check(tenantA, want{tickets: 2, ledger: 2, executed: 2, refused: 1})
	if n := len(w.store.audits(tenantA, AuditRefused, ReasonIdempotencyKeyReused, "p-1")); n != 1 {
		t.Fatalf("audit REFUSED IDEMPOTENCY_KEY_REUSED = %d, muốn 1", n)
	}
}

// Không tra lại được sổ lệnh sau khi R5 thấy phiếu mở: chưa mở giao dịch ghi nào nên biết chắc chưa ghi gì, FAILED STORE_UNAVAILABLE,
// bước C4b lỗi (mọi kết cục của việc tra sổ lệnh đều ở C4b). Ca này đếm "lần tra thứ hai" nên chỉ có nghĩa với cách cài tra lại ở R5.
func TestR5KhongTraLaiDuocSoLenhThiStoreUnavailable(t *testing.T) {
	w, hook, _ := twinWorld(t)
	wantDone(t, w.run(user("p-0", "accountant", ""), req(901, "EX-03", day1)), false)
	boom := errors.New("lỗi thô: kho không trả lời lần tra thứ hai")
	hook.failAt(2, boom)

	o := w.run(user("p-1", "director", ""), req(902, "EX-03", day1))
	wantFail(t, o, StatusFailed, ReasonStoreUnavailable)
	if o.Result.Fault != "infra:postgres" || o.Result.Retry != RetrySameKey || !errors.Is(o.Cause, boom) {
		t.Fatalf("STORE_UNAVAILABLE: infra:postgres, same_key, lỗi gốc ở Cause: %+v, %v", o.Result, o.Cause)
	}
	assertSteps(t, o, "C4b")
	w.check(tenantA, want{tickets: 1, ledger: 1, executed: 1})
	w.counters(1, 0) // không ghi thêm gì, không có audit riêng cho kết cục hạ tầng
}

// jitterReads chèn trễ ngẫu nhiên nhỏ trước mỗi lần đọc của kho để lịch chạy tự nhiên (KHÔNG rào F7) cắt vào các cửa sổ đọc-rồi-ghi.
type jitterReads struct {
	Store
	n atomic.Int64
}

func (j *jitterReads) pause() {
	rng := rand.New(rand.NewSource(j.n.Add(1)))
	if rng.Intn(3) == 0 {
		runtime.Gosched()
	}
	time.Sleep(time.Duration(rng.Intn(300)) * time.Microsecond)
}

func (j *jitterReads) FindLedger(ctx context.Context, caller, commandID string) (LedgerRecord, bool, error) {
	j.pause()
	return j.Store.FindLedger(ctx, caller, commandID)
}

func (j *jitterReads) FindOpenTicket(ctx context.Context, subjectKey, metric, day string) (TicketRef, bool, error) {
	j.pause()
	return j.Store.FindOpenTicket(ctx, subjectKey, metric, day)
}

// N lượt song song CÙNG người gọi, CÙNG khoá, CÙNG nội dung, KHÔNG rào F7, nhiều vòng: mọi lượt phải DONE, đúng một lượt replayed=false
// (§6.3 hàng 3, I5, A02). A02 ép đua bằng F7 nên không thấy lỗi này; ca này không dựa vào F7 để chứng minh.
func TestGuiLaiDongThoiKhongRaoMoiLuotDeuDone(t *testing.T) {
	const lanes, rounds = 6, 100
	p, r := user("p-1", "director", ""), req(903, "EX-03", day1)
	for round := range rounds {
		w := newWorld(t, func(d *Deps) { d.Store = &jitterReads{Store: d.Store} })
		outs := make([]Outcome, lanes)
		var wg sync.WaitGroup
		for i := range lanes {
			wg.Add(1)
			go func() {
				defer wg.Done()
				outs[i] = w.pipe.Run(context.Background(), p, r)
			}()
		}
		wg.Wait()
		ids, fresh := map[string]bool{}, 0
		for _, o := range outs {
			if o.Result.Status != StatusDone {
				t.Fatalf("vòng %d: lượt gửi lại của chính một lệnh nhận %s/%s, muốn DONE (cùng khoá, cùng nội dung)", round, o.Result.Status, o.Result.ReasonCode)
			}
			ids[o.Result.Ticket.ID] = true
			if !o.Result.Replayed {
				fresh++
			}
		}
		if len(ids) != 1 || fresh != 1 || w.store.countOpen(tenantA) != 1 {
			t.Fatalf("vòng %d: muốn đúng 1 mã phiếu và 1 lượt replayed=false, được %d mã, %d lượt mới, %d phiếu ở kho", round, len(ids), fresh, w.store.countOpen(tenantA))
		}
	}
}

// ---------------------------------------------------------------------------
// K1S-16: bước C6 của lượt DONE sau phân loại lại vì vi phạm khoá duy nhất
// ---------------------------------------------------------------------------

// A29 (b): lượt DONE, kể cả phát lại, không có bước lệch. Lượt thua cuộc đua gặp 23505 rồi phát lại (§6.4) vẫn là DONE, nên C6 là ok kèm
// lý do; "deviant" ở C6 dành cho kết cục phân loại lại là từ chối (§7.5).
func TestLuotDoneSauPhanLoaiLaiKhongCoBuocLech(t *testing.T) {
	w := newWorld(t)
	const n = 4
	var prs []contracts.Principal
	var reqs []Request
	for range n {
		prs, reqs = append(prs, user("p-1", "director", "")), append(reqs, req(904, "EX-03", day1))
	}
	reclassified := 0
	for _, o := range w.race(prs, reqs) {
		wantDone(t, o, o.Result.Replayed)
		assertSteps(t, o, "")
		if !o.Result.Replayed {
			continue
		}
		if c6 := o.Steps[stWrite]; c6.Status != StepOK || !strings.Contains(c6.Reason, "phát lại sau vi phạm khoá duy nhất") {
			t.Fatalf("C6 của lượt phát lại sau 23505 phải ok kèm lý do 'phát lại sau vi phạm khoá duy nhất': %+v", c6)
		}
		reclassified++
	}
	if reclassified != n-1 {
		t.Fatalf("muốn %d lượt phát lại sau phân loại lại, được %d", n-1, reclassified)
	}
}

// ---------------------------------------------------------------------------
// K1S-03: lỗi mang nhiều dấu loại
// ---------------------------------------------------------------------------

// Một lỗi còn mang dấu "từ lúc gửi COMMIT" thì không được báo FAILED, dù nó đồng thời mang dấu "chắc chắn chưa ghi" hay "vi phạm khoá
// duy nhất" (§9.2.1: nghi ngờ thì UNKNOWN). Kho nên chỉ trả đúng một loại; đây là lưới đỡ khi kho viết ẩu.
func TestLoiMangDauCommitUnknownLuonLaUnknown(t *testing.T) {
	cases := map[string]error{
		"trước COMMIT cộng từ COMMIT":     errors.Join(ErrBeforeCommit, ErrCommitUnknown),
		"từ COMMIT cộng trước COMMIT":     errors.Join(ErrCommitUnknown, ErrBeforeCommit),
		"vi phạm khoá cộng từ COMMIT":     errors.Join(ErrUniqueViolation, ErrCommitUnknown),
		"cả ba dấu":                       errors.Join(ErrBeforeCommit, ErrUniqueViolation, ErrCommitUnknown),
		"bọc nhiều tầng":                  fmt.Errorf("tầng ngoài: %w", errors.Join(ErrBeforeCommit, fmt.Errorf("tầng trong: %w", ErrCommitUnknown))),
		"hai %w trong một lời gọi Errorf": fmt.Errorf("%w; %w", ErrBeforeCommit, ErrCommitUnknown),
	}
	for name, werr := range cases {
		t.Run(name, func(t *testing.T) {
			w := newWorld(t, func(d *Deps) { d.Store = &failReads{Store: d.Store, write: werr} })
			o := w.run(user("p-1", "director", ""), req(910, "EX-03", day1))
			wantFail(t, o, StatusUnknown, ReasonCommitUnknown)
			if o.Result.Fault != "infra:postgres" || o.Result.Retry != RetrySameKey {
				t.Fatalf("COMMIT_UNKNOWN: infra:postgres, same_key: %+v", o.Result)
			}
			assertSteps(t, o, "C6")
			w.check(tenantA, want{})
		})
	}
}

// ---------------------------------------------------------------------------
// K1S-04: panic
// ---------------------------------------------------------------------------

// panicAt bọc Store: panic ở hàm được chọn. Với WriteTicket, afterWrite nghĩa là gọi kho thật (đã commit) rồi mới panic.
type panicAt struct {
	Store
	method     string
	afterWrite bool
}

func (p panicAt) FindLedger(ctx context.Context, caller, commandID string) (LedgerRecord, bool, error) {
	if p.method == "FindLedger" {
		panic("panic của kho ở FindLedger")
	}
	return p.Store.FindLedger(ctx, caller, commandID)
}

func (p panicAt) WriteTicket(ctx context.Context, u WriteUnit) error {
	if p.method == "WriteTicket" {
		if p.afterWrite {
			_ = p.Store.WriteTicket(ctx, u)
		}
		panic("panic của kho ở WriteTicket")
	}
	return p.Store.WriteTicket(ctx, u)
}

func (p panicAt) ReadBack(ctx context.Context, caller, commandID, ticketID string) (Stored, error) {
	if p.method == "ReadBack" {
		panic("panic của kho ở ReadBack")
	}
	return p.Store.ReadBack(ctx, caller, commandID, ticketID)
}

func (p panicAt) WriteAudit(ctx context.Context, a Audit) error {
	if p.method == "WriteAudit" {
		panic("panic của kho ở WriteAudit")
	}
	return p.Store.WriteAudit(ctx, a)
}

type panicDir struct{ value any }

func (d panicDir) Entities(context.Context, ...string) ([]entityresolution.Entity, error) {
	panic(d.value)
}

type panicFacts struct{ sourceoftruth.Store }

func (panicFacts) Facts(context.Context, string, []string, time.Time, time.Time) ([]sourceoftruth.Fact, error) {
	panic("panic của kho số liệu")
}

// runBatPanic chạy một lượt và trả lại panic nếu nó lọt ra khỏi Run: ca ĐỎ thay vì làm sập cả gói test.
func runBatPanic(w *world, pr contracts.Principal, rq Request) (o Outcome, leaked any) {
	defer func() { leaked = recover() }()
	return w.run(pr, rq), nil
}

// codeIndex trả vị trí của bước theo mã (C0…C7, C4b).
func codeIndex(t *testing.T, o Outcome, code string) int {
	t.Helper()
	for i, s := range o.Steps {
		if s.Code == code {
			return i
		}
	}
	t.Fatalf("không có bước %s", code)
	return -1
}

// Run bắt panic của kho, danh mục, số liệu hay hàm của Deps và biến nó thành kết quả, vết và lỗi gốc: trước khi gọi đơn vị ghi phiếu thì
// INTERNAL_ERROR FAILED (chắc chắn chưa ghi); từ lúc gọi nó thì INTERNAL_ERROR UNKNOWN (không biết đã commit chưa, §2.4, §9.2.1).
func TestRunBatPanicVaPhanLoaiTheoDiemGhi(t *testing.T) {
	boomNow := func() time.Time { panic("panic của đồng hồ") }
	boomID := func() string { panic("panic của bộ sinh mã") }
	director, engineer := user("p-1", "director", ""), user("p-se", "site_engineer", "PRJ-A")
	cases := []struct {
		name    string
		arm     func(*Deps)
		pr      contracts.Principal
		status  Status
		errStep string // bước báo lỗi (error)
		tickets int    // phiếu ở kho sau lượt
		refused bool   // bước lệch đầu tiên là lời từ chối ở C4 (panic nằm ở C7), không phải bước lỗi
	}{
		{"đồng hồ của Deps panic lúc dựng lượt", func(d *Deps) { d.Now = boomNow }, director, StatusFailed, "C0", 0, false},
		{"bộ sinh mã của Deps panic lúc dựng lượt", func(d *Deps) { d.NewID = boomID }, director, StatusFailed, "C0", 0, false},
		{"danh mục thực thể panic ở C4", func(d *Deps) { d.Directory = panicDir{"panic của danh mục"} }, director, StatusFailed, "C4", 0, false},
		{"sổ lệnh panic ở C4b", func(d *Deps) { d.Store = panicAt{Store: d.Store, method: "FindLedger"} }, director, StatusFailed, "C4b", 0, false},
		{"kho số liệu panic ở C5", func(d *Deps) { d.Facts = panicFacts{d.Facts} }, director, StatusFailed, "C5", 0, false},
		{"mã phiếu: bộ sinh mã panic ở C6 trước khi gọi đơn vị ghi", func(d *Deps) {
			n := 0
			d.NewID = func() string { // lần đầu cho trace_id, lần hai cho mã phiếu
				if n++; n > 1 {
					panic("panic của bộ sinh mã lúc lập phiếu")
				}
				return "trace-1"
			}
		}, director, StatusFailed, "C6", 0, false},
		{"ghi audit của lượt bị từ chối panic ở C7", func(d *Deps) { d.Store = panicAt{Store: d.Store, method: "WriteAudit"} }, engineer, StatusFailed, "C7", 0, true},
		{"đơn vị ghi phiếu panic trước commit", func(d *Deps) { d.Store = panicAt{Store: d.Store, method: "WriteTicket"} }, director, StatusUnknown, "C6", 0, false},
		{"đơn vị ghi phiếu panic sau commit", func(d *Deps) { d.Store = panicAt{Store: d.Store, method: "WriteTicket", afterWrite: true} }, director, StatusUnknown, "C6", 1, false},
		{"đọc lại panic sau khi ghi", func(d *Deps) { d.Store = panicAt{Store: d.Store, method: "ReadBack"} }, director, StatusUnknown, "C7", 1, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			w := newWorld(t, c.arm)
			o, leaked := runBatPanic(w, c.pr, req(950, "EX-03", day1))
			if leaked != nil {
				t.Fatalf("panic lọt ra khỏi Run (không có Result, không vết): %v", leaked)
			}
			wantFail(t, o, c.status, ReasonInternalError)
			if o.Result.Fault != "code" || o.Result.Retry != RetryNone || strings.Contains(strings.ToLower(o.Result.Detail), "panic") {
				t.Fatalf("INTERNAL_ERROR: fault code, retry none, detail không lộ panic: %+v", o.Result)
			}
			if len(o.Steps) != 9 {
				t.Fatalf("vết phải đủ chín bước, được %d", len(o.Steps))
			}
			errAt := o.Steps[codeIndex(t, o, c.errStep)]
			if errAt.Status != StepError || errAt.Fault != "code" || !strings.Contains(errAt.Error, "panic") {
				t.Fatalf("bước %s phải là error, fault code, giữ thông tin panic: %+v", c.errStep, errAt)
			}
			if c.refused {
				assertSteps(t, o, "C4")
			} else {
				assertSteps(t, o, c.errStep)
			}
			if c7 := o.Steps[stVerify]; c7.Status == StepSkipped {
				t.Fatalf("C7 không bao giờ skipped: %+v", c7)
			}
			if o.Cause == nil || !strings.Contains(o.Cause.Error(), "panic của") {
				t.Fatalf("Cause phải giữ thông tin panic: %v", o.Cause)
			}
			if got := len(w.store.ticketList(tenantA)); got != c.tickets {
				t.Fatalf("phiếu ở kho = %d, muốn %d", got, c.tickets)
			}
			w.check(tenantA, want{tickets: c.tickets, ledger: c.tickets, executed: c.tickets}) // panic không sinh audit riêng
		})
	}
}

// Sau panic ở đọc lại, gửi lại cùng khoá với kho lành phải phát lại đúng phiếu đã commit: UNKNOWN không làm mất lối thoát.
func TestSauPanicDocLaiGuiLaiCungKhoaPhatLai(t *testing.T) {
	w := newWorld(t, func(d *Deps) { d.Store = panicAt{Store: d.Store, method: "ReadBack"} })
	p, r := user("p-1", "director", ""), req(951, "EX-03", day1)
	if _, leaked := runBatPanic(w, p, r); leaked != nil {
		t.Fatalf("panic lọt ra khỏi Run: %v", leaked)
	}
	stored := w.onlyTicket()
	w.deps.Store = w.probe // kho lành, cùng dữ liệu
	w.restart()
	again := wantDone(t, w.run(p, r), true)
	if again.ID != stored.ID {
		t.Fatalf("gửi lại phải phát lại đúng phiếu đã commit: %s và %s", again.ID, stored.ID)
	}
}

// Lượt phát lại mà đọc lại panic: sổ lệnh ĐÃ có dòng của khoá này nên có phiếu thật; nói FAILED "chưa được ghi" là sai sự thật, nên
// UNKNOWN (cùng cách xử lý READBACK_UNAVAILABLE của lượt phát lại).
func TestPanicKhiPhatLaiDocLaiLaUnknown(t *testing.T) {
	w := newWorld(t)
	p, r := user("p-1", "director", ""), req(952, "EX-03", day1)
	wantDone(t, w.run(p, r), false)
	w.deps.Store = panicAt{Store: w.probe, method: "ReadBack"}
	w.restart()
	o, leaked := runBatPanic(w, p, r)
	if leaked != nil {
		t.Fatalf("panic lọt ra khỏi Run: %v", leaked)
	}
	wantFail(t, o, StatusUnknown, ReasonInternalError)
	if c4b := o.Steps[stLedger]; c4b.Status != StepOK || c4b.Decision != "phát lại" {
		t.Fatalf("C4b vẫn ghi 'phát lại': %+v", c4b)
	}
	if c7 := o.Steps[stVerify]; c7.Status != StepError || c7.Fault != "code" {
		t.Fatalf("C7 phải là error: %+v", c7)
	}
	w.check(tenantA, want{tickets: 1, ledger: 1, executed: 1})
}

// Giá trị panic là một error thì Cause bọc nó bằng %w để errors.Is nhận ra; ngăn xếp (nếu có) chỉ nằm ở Cause, không ở Step.Error.
func TestPanicLaErrorThiCauseBocBangW(t *testing.T) {
	sentinel := errors.New("lỗi mốc của panic")
	w := newWorld(t, func(d *Deps) { d.Directory = panicDir{sentinel} })
	o, leaked := runBatPanic(w, user("p-1", "director", ""), req(953, "EX-03", day1))
	if leaked != nil {
		t.Fatalf("panic lọt ra khỏi Run: %v", leaked)
	}
	if !errors.Is(o.Cause, sentinel) {
		t.Fatalf("Cause phải bọc được giá trị panic là error: %v", o.Cause)
	}
	if e := o.Steps[stPermission].Error; strings.Contains(e, "\n") || !strings.Contains(e, sentinel.Error()) {
		t.Fatalf("Step.Error chỉ một dòng, nêu giá trị panic: %q", e)
	}
}
