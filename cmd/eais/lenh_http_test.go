package main

// Cửa HTTP của lệnh (HOP-DONG §2.6.2, ca A28) và vết của lượt lệnh qua HTTP (§7.5, A29): route POST /v1/commands/open_discrepancy_ticket bằng token dev thật,
// trên kho bộ nhớ có khoá duy nhất như PostgreSQL (cmdStore) để chạy được không cần PG, cộng MỘT ca đầu cuối trên PG thật (TestLenhPGDauCuoi). Ràng buộc thật
// của PG (RLS, quyền vai, COMMIT) có test riêng ở execution/commands và platform/pgstore. Bốn route cũ không đổi hành vi: bộ test sẵn có của cmd/eais giữ nguyên
// và vẫn xanh (A27).

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"os"
	"slices"
	"strings"
	"sync"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/luannt2002/enterprise-ai-control-layer/core/contracts"
	"github.com/luannt2002/enterprise-ai-control-layer/execution/commands"
	"github.com/luannt2002/enterprise-ai-control-layer/observability/traces"
)

const (
	cmdPath = "/v1/commands/open_discrepancy_ticket"
	dayAgo  = "2026-10-19" // "hôm qua" của clockForTest (20/10/2026 10:00 giờ tenant)
)

func cmdUUID(n int) string { return fmt.Sprintf("6f1c0d52-0000-4000-8000-%012d", n) }

func cmdReq(n int, key, day string) map[string]string {
	return map[string]string{"command_id": cmdUUID(n), "subject_key": key, "metric": "issue_vs_measured", "day": day}
}

func (e *env) cmd(token string, body any) reply { e.t.Helper(); return e.post(cmdPath, token, body) }

// ---------------------------------------------------------------------------------------------------------------------------------
// Kho phiếu trong bộ nhớ: chia theo tenant lấy từ ngữ cảnh như RLS, có hai khoá duy nhất của migration 0004 (sổ lệnh theo (người gọi, command_id); một phiếu
// OPEN cho mỗi (khoá máy, chỉ số, ngày)), và hai núm lỗi: F1 (lỗi trước khi ghi) và F3 (ghi thật rồi mất ack của COMMIT).
// ---------------------------------------------------------------------------------------------------------------------------------

type ticketRows struct {
	tickets []commands.TicketRecord
	ledger  []commands.LedgerRecord
	audits  []commands.Audit
}

type ticketMem struct {
	tmu      sync.Mutex
	rows     map[string]*ticketRows
	writeErr error // lỗi cho đơn vị ghi kế tiếp, TRƯỚC khi ghi gì (F1)
	ackLost  bool  // đơn vị ghi kế tiếp GHI THẬT rồi mới báo commands.ErrCommitUnknown (F3)
}

func newTicketMem() *ticketMem { return &ticketMem{rows: map[string]*ticketRows{}} }

func (m *ticketMem) scoped(ctx context.Context) (*ticketRows, error) {
	tn, ok := contracts.TenantFrom(ctx)
	if !ok {
		return nil, contracts.Errorf(contracts.FaultPolicy, "truy vấn không mang tenant trong ngữ cảnh")
	}
	if m.rows[string(tn)] == nil {
		m.rows[string(tn)] = &ticketRows{}
	}
	return m.rows[string(tn)], nil
}

func (m *ticketMem) FindLedger(ctx context.Context, caller, commandID string) (commands.LedgerRecord, bool, error) {
	m.tmu.Lock()
	defer m.tmu.Unlock()
	r, err := m.scoped(ctx)
	if err != nil {
		return commands.LedgerRecord{}, false, err
	}
	for _, l := range r.ledger {
		if l.Caller == caller && l.CommandID == commandID {
			return l, true, nil
		}
	}
	return commands.LedgerRecord{}, false, nil
}

func (m *ticketMem) FindOpenTicket(ctx context.Context, subjectKey, metric, day string) (commands.TicketRef, bool, error) {
	m.tmu.Lock()
	defer m.tmu.Unlock()
	r, err := m.scoped(ctx)
	if err != nil {
		return commands.TicketRef{}, false, err
	}
	for _, t := range r.tickets {
		if t.SubjectKey == subjectKey && t.Metric == metric && t.Day == day && t.Status == commands.TicketOpen {
			return commands.TicketRef{ID: t.ID, Status: t.Status}, true, nil
		}
	}
	return commands.TicketRef{}, false, nil
}

func (m *ticketMem) WriteTicket(ctx context.Context, u commands.WriteUnit) error {
	m.tmu.Lock()
	defer m.tmu.Unlock()
	r, err := m.scoped(ctx)
	if err != nil {
		return fmt.Errorf("%w: %w", commands.ErrBeforeCommit, err)
	}
	if m.writeErr != nil {
		err := m.writeErr
		m.writeErr = nil
		return fmt.Errorf("%w: %w", commands.ErrBeforeCommit, err)
	}
	for _, l := range r.ledger {
		if l.Caller == u.Ledger.Caller && l.CommandID == u.Ledger.CommandID {
			return fmt.Errorf("%w: sổ lệnh", commands.ErrUniqueViolation)
		}
	}
	for _, t := range r.tickets {
		if t.SubjectKey == u.Ticket.SubjectKey && t.Metric == u.Ticket.Metric && t.Day == u.Ticket.Day && t.Status == commands.TicketOpen {
			return fmt.Errorf("%w: phiếu mở", commands.ErrUniqueViolation)
		}
	}
	r.tickets, r.ledger, r.audits = append(r.tickets, u.Ticket), append(r.ledger, u.Ledger), append(r.audits, u.Audit)
	if m.ackLost {
		m.ackLost = false
		return fmt.Errorf("%w: mất ack", commands.ErrCommitUnknown)
	}
	return nil
}

func (m *ticketMem) ReadBack(ctx context.Context, caller, commandID, ticketID string) (commands.Stored, error) {
	m.tmu.Lock()
	defer m.tmu.Unlock()
	r, err := m.scoped(ctx)
	if err != nil {
		return commands.Stored{}, err
	}
	var st commands.Stored
	for _, t := range r.tickets {
		if t.ID == ticketID {
			st.Tickets = append(st.Tickets, t)
		}
	}
	for _, l := range r.ledger {
		if l.Caller == caller && l.CommandID == commandID {
			st.Ledger = append(st.Ledger, l)
		}
	}
	for _, a := range r.audits {
		if a.Outcome == commands.AuditExecuted && a.Caller.ID == caller && a.CommandID == commandID {
			st.Executed = append(st.Executed, a)
		}
	}
	return st, nil
}

func (m *ticketMem) WriteAudit(ctx context.Context, a commands.Audit) error {
	m.tmu.Lock()
	defer m.tmu.Unlock()
	r, err := m.scoped(ctx)
	if err != nil {
		return err
	}
	r.audits = append(r.audits, a)
	return nil
}

// counts đếm theo kết nối riêng của harness (đọc thẳng bộ nhớ, không qua pipeline): phiếu OPEN, dòng sổ lệnh, và số dòng audit theo kết cục.
func (m *ticketMem) counts(tenant string) (open, ledger int, audit map[commands.AuditOutcome]int) {
	m.tmu.Lock()
	defer m.tmu.Unlock()
	audit = map[commands.AuditOutcome]int{}
	r := m.rows[tenant]
	if r == nil {
		return 0, 0, audit
	}
	for _, t := range r.tickets {
		if t.Status == commands.TicketOpen {
			open++
		}
	}
	for _, a := range r.audits {
		audit[a.Outcome]++
	}
	return open, len(r.ledger), audit
}

// cmdStore là kho bộ nhớ của app (memStore) cộng kho phiếu: commandStoreOf thấy nó cài commands.Store nên app có route lệnh.
type cmdStore struct {
	*memStore
	*ticketMem
}

func newCmdEnv(t *testing.T) (*env, *cmdStore) {
	t.Helper()
	cs := &cmdStore{memStore: newMemStore(), ticketMem: newTicketMem()}
	return newEnv(t, cs), cs
}

// wantResult kiểm một phản hồi lệnh: mã HTTP, status, reason_code, lớp lỗi ở trường fault (không có dấu hai chấm), retry; kết quả không thành công phải có
// detail; X-Trace-Id (nếu có) phải bằng trace_id của thân.
func wantResult(t *testing.T, r reply, httpStatus int, st commands.Status, code commands.ReasonCode, fault string, retry commands.Retry) commands.Result {
	t.Helper()
	res := decodeAs[commands.Result](t, r)
	if r.status != httpStatus || res.Status != st || res.ReasonCode != code || res.Fault != fault || res.Retry != retry {
		t.Fatalf("muốn HTTP %d %s/%s fault %q retry %s; được HTTP %d %s/%s fault %q retry %s\n%s", httpStatus, st, code, fault, retry,
			r.status, res.Status, res.ReasonCode, res.Fault, res.Retry, r.body)
	}
	if strings.Contains(res.Fault, ":") || bytes.Contains(r.body, []byte(`:postgres`)) {
		t.Errorf("fault qua HTTP chỉ là lớp lỗi, không có tên thành phần nội bộ: %q\n%s", res.Fault, r.body)
	}
	if st != commands.StatusDone && res.Detail == "" {
		t.Errorf("kết quả không thành công phải có detail: %s", r.body)
	}
	if got := r.header.Get("X-Trace-Id"); got != res.TraceID {
		t.Errorf("X-Trace-Id %q khác trace_id của thân %q", got, res.TraceID)
	}
	return res
}

func (e *env) traceOf(token, id string) (traceResponse, string) {
	e.t.Helper()
	j := e.trace(token, id)
	if j.status != http.StatusOK {
		e.t.Fatalf("GET vết %s = %d: %s", id, j.status, j.body)
	}
	tb := e.do(http.MethodGet, "/v1/trace/"+id+"?format=table", token, "", nil)
	if tb.status != http.StatusOK {
		e.t.Fatalf("GET vết (bảng) %s = %d: %s", id, tb.status, tb.body)
	}
	return decodeAs[traceResponse](e.t, j), string(tb.body)
}

func stageList(rec traces.Record) []string {
	var out []string
	for _, s := range rec.Steps {
		out = append(out, s.Stage)
	}
	return out
}

var nineStages = []string{"C0", "C1", "C2", "C3", "C4", "C4b", "C5", "C6", "C7"}

// ---------------------------------------------------------------------------------------------------------------------------------
// A28 (a): chưa xác thực. A28 (c): tên lệnh lạ và sai phương thức, trước xác thực.
// ---------------------------------------------------------------------------------------------------------------------------------

func TestLenhHTTPChuaXacThuc(t *testing.T) {
	e, cs := newCmdEnv(t)
	var first []byte
	for i, tok := range []string{"", "t-khong-co", "dev-khong-co"} {
		r := e.cmd(tok, cmdReq(1, "EX-03", dayAgo))
		if r.status != http.StatusUnauthorized || r.header.Get("X-Trace-Id") != "" || r.header.Get("WWW-Authenticate") == "" {
			t.Fatalf("token %q: muốn 401 không X-Trace-Id có WWW-Authenticate, được %d %v", tok, r.status, r.header)
		}
		if i == 0 {
			first = r.body
		} else if !bytes.Equal(first, r.body) {
			t.Errorf("thiếu, sai và lạ token phải cùng MỘT phản hồi:\n%s\n%s", first, r.body)
		}
	}
	if got := e.srv.cnt.Unauthenticated.Load(); got != 3 {
		t.Errorf("bộ đếm chưa xác thực = %d, muốn 3", got)
	}
	open, led, aud := cs.counts(e.tenantA)
	if n := cs.decisionCount(e.tenantA); n != 0 || open != 0 || led != 0 || len(aud) != 0 {
		t.Errorf("lượt chưa xác thực không được có vết hay ghi: vết %d phiếu %d sổ %d audit %v", n, open, led, aud)
	}
}

func TestLenhHTTPDinhTuyen(t *testing.T) {
	e, cs := newCmdEnv(t)
	for _, tok := range []string{"", tokDirector} { // 404 và 405 xảy ra TRƯỚC xác thực nên có token hay không đều như nhau
		for _, c := range []struct {
			name, method, path string
			want               int
		}{
			{"tên lệnh lạ", http.MethodPost, "/v1/commands/khong_co_lenh_nay", http.StatusNotFound},
			{"thiếu tên lệnh", http.MethodPost, "/v1/commands/", http.StatusNotFound},
			{"đường dẫn thừa dấu gạch", http.MethodPost, cmdPath + "/", http.StatusNotFound},
			{"GET", http.MethodGet, cmdPath, http.StatusMethodNotAllowed},
			{"PUT", http.MethodPut, cmdPath, http.StatusMethodNotAllowed},
			{"DELETE", http.MethodDelete, cmdPath, http.StatusMethodNotAllowed},
		} {
			r := e.do(c.method, c.path, tok, "application/json", []byte(`{}`))
			if r.status != c.want || r.header.Get("X-Trace-Id") != "" {
				t.Errorf("token %q, %s: muốn %d không vết, được %d", tok, c.name, c.want, r.status)
			}
			if c.want == http.StatusMethodNotAllowed && !strings.Contains(r.header.Get("Allow"), "POST") {
				t.Errorf("%s: 405 phải kèm Allow: POST, được %q", c.name, r.header.Get("Allow"))
			}
		}
	}
	if got := e.srv.cnt.Unauthenticated.Load(); got != 0 {
		t.Errorf("404 và 405 không phải lượt chưa xác thực, bộ đếm = %d", got)
	}
	if n := cs.decisionCount(e.tenantA); n != 0 {
		t.Errorf("404 và 405 không có vết, được %d", n)
	}
}

// Kho không có kho phiếu (kho bộ nhớ của test sẵn có) thì không có route lệnh: đóng mặc định thay vì một route trỏ vào pipeline không chạy được.
func TestLenhHTTPKhongKhoPhieuThiKhongCoRoute(t *testing.T) {
	e := newEnv(t, newMemStore())
	if r := e.cmd(tokDirector, cmdReq(1, "EX-03", dayAgo)); r.status != http.StatusNotFound {
		t.Fatalf("muốn 404, được %d: %s", r.status, r.body)
	}
	if e.app.tools == nil || !slices.Equal(e.app.tools.Names(), []string{commands.CommandOpenDiscrepancyTicket}) || len(e.app.cmds) != 0 {
		t.Errorf("tools phải nạp (cấu hình được kiểm) dù không có pipeline: %v %v", e.app.tools, e.app.cmds)
	}
}

// ---------------------------------------------------------------------------------------------------------------------------------
// A28 (b): thân sai. Lượt đã xác thực nên CÓ vết, dừng ở C0; thân luôn là CommandResult.
// ---------------------------------------------------------------------------------------------------------------------------------

func TestLenhHTTPThanSai(t *testing.T) {
	const u = "6f1c0d52-0000-4000-8000-000000000001"
	good := `{"command_id":"` + u + `","subject_key":"EX-03","metric":"issue_vs_measured","day":"2026-10-19"}`
	cases := []struct {
		name, ctype, body string
		status            int
		fields            []string
	}{
		{"trường tenant", "application/json", `{"tenant":"abc","command_id":"` + u + `","subject_key":"EX-03","metric":"issue_vs_measured","day":"2026-10-19"}`, 400, []string{"tenant"}},
		{"trường principal", "application/json", `{"principal":"u-director","command_id":"` + u + `","subject_key":"EX-03","metric":"issue_vs_measured","day":"2026-10-19"}`, 400, []string{"principal"}},
		{"trường role", "application/json", `{"command_id":"` + u + `","subject_key":"EX-03","metric":"issue_vs_measured","day":"2026-10-19","role":"director"}`, 400, []string{"role"}},
		{"trường lạ (chữ tự do)", "application/json", `{"command_id":"` + u + `","subject_key":"EX-03","metric":"issue_vs_measured","day":"2026-10-19","note":"gấp"}`, 400, []string{"note"}},
		{"thân rỗng", "application/json", ``, 400, nil},
		{"JSON hỏng", "application/json", `{"command_id":`, 400, nil},
		{"không phải đối tượng", "application/json", `[1]`, 400, nil},
		{"nội dung thừa", "application/json", good + good, 400, nil},
		{"sai kiểu của trường đúng tên", "application/json", `{"command_id":5,"subject_key":"EX-03","metric":"issue_vs_measured","day":"2026-10-19"}`, 400, []string{"command_id"}},
		{"sai Content-Type", "text/plain", good, 415, nil},
		{"thiếu Content-Type", "", good, 415, nil},
		{"thân quá lớn", "application/json", `{"command_id":"` + strings.Repeat("a", 9000) + `"}`, 413, nil},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			e, cs := newCmdEnv(t)
			r := e.do(http.MethodPost, cmdPath, tokDirector, c.ctype, []byte(c.body))
			res := wantResult(t, r, c.status, commands.StatusFailed, commands.ReasonInvalidRequest, "input", commands.RetryFixRequest)
			if !slices.Equal(res.InvalidFields, c.fields) {
				t.Errorf("invalid_fields = %v, muốn %v", res.InvalidFields, c.fields)
			}
			if !strings.Contains(res.Detail, "token") || res.Ticket != nil || res.TraceID == "" {
				t.Errorf("detail phải nói tenant, vai, phạm vi chỉ đến từ token; không phiếu; có trace_id: %+v", res)
			}
			tr, table := e.traceOf(tokDirector, res.TraceID)
			if !slices.Equal(stageList(tr.Record), nineStages) || !strings.Contains(table, "BƯỚC LỆCH ĐẦU TIÊN: C0 accept") {
				t.Errorf("vết phải đủ chín bước và dừng ở C0:\n%s", table)
			}
			if tr.Record.Decision.Outcome != "INVALID_REQUEST" || tr.Record.Decision.Principal != "u-director" {
				t.Errorf("quyết định của vết: %+v", tr.Record.Decision)
			}
			open, led, aud := cs.counts(e.tenantA)
			if open != 0 || led != 0 || len(aud) != 0 {
				t.Errorf("thân sai: 0 ghi, kể cả audit (chưa có yêu cầu hợp lệ); được phiếu %d sổ %d audit %v", open, led, aud)
			}
		})
	}
}

// Tên trường lạ do client đặt có thể rất dài: kết quả chỉ trả phần đầu, không phình theo thân.
func TestLenhHTTPTenTruongLaDuocCat(t *testing.T) {
	e, _ := newCmdEnv(t)
	body := `{"` + strings.Repeat("x", 3000) + `":1}`
	r := e.do(http.MethodPost, cmdPath, tokDirector, "application/json", []byte(body))
	res := wantResult(t, r, 400, commands.StatusFailed, commands.ReasonInvalidRequest, "input", commands.RetryFixRequest)
	if len(res.InvalidFields) != 1 || len([]rune(res.InvalidFields[0])) > 70 || !strings.HasPrefix(res.InvalidFields[0], "xxxx") {
		t.Errorf("tên trường lạ phải được cắt: %v", res.InvalidFields)
	}
}

// ---------------------------------------------------------------------------------------------------------------------------------
// A28 (d): mỗi nhóm kết cục có mã HTTP đúng bảng §2.6.2 và thân CommandResult đủ trường.
// ---------------------------------------------------------------------------------------------------------------------------------

func TestLenhHTTPKetCuc(t *testing.T) {
	e, cs := newCmdEnv(t)

	// DONE: 200, phiếu OPEN với hai vế và chênh lệch, hệ tính (bên gọi không đưa số).
	r := e.cmd(tokDirector, cmdReq(1, "EX-03", dayAgo))
	done := wantResult(t, r, 200, commands.StatusDone, "", "", commands.RetryNone)
	tk := done.Ticket
	if done.Replayed || tk == nil || tk.Status != "OPEN" || tk.SubjectKey != "EX-03" || tk.Metric != "issue_vs_measured" || tk.Day != dayAgo || tk.Approx == nil ||
		tk.Left == nil || tk.Right == nil || tk.Left.Metric != "issued_fuel" || tk.Right.Metric != "measured_consumption" || tk.Diff == "" {
		t.Fatalf("DONE phải mang phiếu OPEN đủ trường: %+v / %s", tk, r.body)
	}
	if tk.Left.Value != "3850" || tk.Right.Value != "3782" || tk.Diff != "68" || *tk.Approx {
		t.Errorf("hai vế và chênh lệch của dữ liệu hạt giống: %+v %+v diff %s approx %v, muốn 3850, 3782, 68, false", tk.Left, tk.Right, tk.Diff, *tk.Approx)
	}
	open, led, aud := cs.counts(e.tenantA)
	if open != 1 || led != 1 || aud[commands.AuditExecuted] != 1 || len(aud) != 1 {
		t.Fatalf("sau lượt đầu: muốn 1 phiếu, 1 sổ lệnh, 1 EXECUTED; được %d %d %v", open, led, aud)
	}

	// Gửi lại nguyên văn: vẫn 200, replayed, CÙNG mã phiếu, không ghi thêm; audit REPLAYED.
	again := wantResult(t, e.cmd(tokDirector, cmdReq(1, "EX-03", dayAgo)), 200, commands.StatusDone, "", "", commands.RetryNone)
	if !again.Replayed || again.Ticket == nil || again.Ticket.ID != tk.ID || again.TraceID == done.TraceID {
		t.Errorf("phát lại phải cùng mã phiếu, replayed, vết mới: %+v", again)
	}

	// Khác khoá, cùng (máy, chỉ số, ngày), phiếu còn mở: 409 DUPLICATE_OPEN_TICKET kèm mã phiếu cũ.
	dup := wantResult(t, e.cmd(tokDirector, cmdReq(2, "EX-03", dayAgo)), 409, commands.StatusFailed, commands.ReasonDuplicateOpenTicket, "data", commands.RetryNone)
	if dup.Ticket == nil || dup.Ticket.ID != tk.ID {
		t.Errorf("DUPLICATE_OPEN_TICKET phải kèm mã phiếu cũ: %+v", dup.Ticket)
	}

	// Cùng khoá, khác nội dung: 400 IDEMPOTENCY_KEY_REUSED.
	wantResult(t, e.cmd(tokDirector, cmdReq(1, "EX-03", "2026-10-18")), 400, commands.StatusFailed, commands.ReasonIdempotencyKeyReused, "input", commands.RetryFixRequest)

	// Thiếu quyền: người lái xe (không capability nào) và kỹ sư công trường (đọc được số liệu mà không có tickets.create): 403.
	for _, tok := range []string{tokDriver, tokEng} {
		wantResult(t, e.cmd(tok, cmdReq(3, "EX-03", dayAgo)), 403, commands.StatusFailed, commands.ReasonPermissionDenied, "policy", commands.RetryNone)
	}
	// Máy không tồn tại và khoá chứa ký tự SQL: cùng PERMISSION_DENIED 403, cùng thông báo, không lỗi SQL.
	var details []string
	for _, key := range []string{"EX-99", "EX-03' OR '1'='1"} {
		res := wantResult(t, e.cmd(tokDirector, cmdReq(4, key, dayAgo)), 403, commands.StatusFailed, commands.ReasonPermissionDenied, "policy", commands.RetryNone)
		details = append(details, res.Detail)
	}
	if details[0] != details[1] {
		t.Errorf("máy không tồn tại và khoá lạ phải cùng thông báo: %q / %q", details[0], details[1])
	}
	// Tham số sai ở C3: 400 INVALID_REQUEST nêu đúng trường.
	bad := cmdReq(5, "EX-03", "2999-01-01")
	res := wantResult(t, e.cmd(tokDirector, bad), 400, commands.StatusFailed, commands.ReasonInvalidRequest, "input", commands.RetryFixRequest)
	if !slices.Equal(res.InvalidFields, []string{"day"}) {
		t.Errorf("invalid_fields = %v, muốn [day]", res.InvalidFields)
	}

	// Sau tất cả: vẫn đúng 1 phiếu, 1 sổ lệnh.
	open, led, aud = cs.counts(e.tenantA)
	if open != 1 || led != 1 || aud[commands.AuditExecuted] != 1 || aud[commands.AuditReplayed] != 1 {
		t.Errorf("cuối ca: muốn 1 phiếu, 1 sổ lệnh, 1 EXECUTED, 1 REPLAYED; được %d %d %v", open, led, aud)
	}
	// Lượt hạ tầng KHÔNG bao giờ đổi thành lỗi mã: xem TestLenhHTTPKetCucHaTang.
}

func TestLenhHTTPKetCucHaTang(t *testing.T) {
	e, cs := newCmdEnv(t)

	// F1: kho báo lỗi TRƯỚC khi ghi ⇒ FAILED, 503, gửi lại cùng khoá thì thực thi bình thường.
	cs.ticketMem.writeErr = fmt.Errorf("F1: kho từ chối")
	r := e.cmd(tokDirector, cmdReq(1, "EX-03", dayAgo))
	wantResult(t, r, 503, commands.StatusFailed, commands.ReasonStoreUnavailable, "infra", commands.RetrySameKey)
	if open, led, _ := cs.counts(e.tenantA); open != 0 || led != 0 {
		t.Fatalf("F1: muốn 0 ghi, được phiếu %d sổ %d", open, led)
	}
	if bytes.Contains(r.body, []byte("F1: kho từ chối")) {
		t.Error("lỗi gốc của hạ tầng lộ ra thân")
	}
	wantResult(t, e.cmd(tokDirector, cmdReq(1, "EX-03", dayAgo)), 200, commands.StatusDone, "", "", commands.RetryNone)

	// F3: ghi THẬT rồi mất ack ⇒ UNKNOWN COMMIT_UNKNOWN 503 same_key; gửi lại cùng khoá ⇒ DONE phát lại, vẫn đúng 1 phiếu.
	cs.ticketMem.ackLost = true
	unk := wantResult(t, e.cmd(tokDirector, cmdReq(2, "EX-07", dayAgo)), 503, commands.StatusUnknown, commands.ReasonCommitUnknown, "infra", commands.RetrySameKey)
	replay := wantResult(t, e.cmd(tokDirector, cmdReq(2, "EX-07", dayAgo)), 200, commands.StatusDone, "", "", commands.RetryNone)
	if !replay.Replayed || replay.Ticket == nil || unk.Ticket != nil {
		t.Errorf("sau F3 gửi lại phải DONE phát lại: %+v", replay)
	}
	if open, led, _ := cs.counts(e.tenantA); open != 2 || led != 2 {
		t.Errorf("hai lệnh (EX-03, EX-07) ⇒ 2 phiếu, 2 sổ lệnh; được %d %d", open, led)
	}
}

// Principal của tenant khác runtime (A10): C1 từ chối, 403, không ghi, không audit; vết lưu ở tenant của người gọi, không lẫn vào tenant của runtime.
func TestLenhHTTPTenantKhac(t *testing.T) {
	e, cs := newCmdEnv(t)
	res := wantResult(t, e.cmd(tokDirectorB, cmdReq(1, "EX-03", dayAgo)), 403, commands.StatusFailed, commands.ReasonPermissionDenied, "policy", commands.RetryNone)
	open, led, aud := cs.counts(e.tenantA)
	openB, ledB, audB := cs.counts(e.tenantB)
	if open+led+openB+ledB != 0 || len(aud)+len(audB) != 0 {
		t.Errorf("A10: 0 ghi và không audit; được A %d %d %v, B %d %d %v", open, led, aud, openB, ledB, audB)
	}
	if cs.decisionCount(e.tenantA) != 0 || cs.decisionCount(e.tenantB) != 1 {
		t.Errorf("vết của người gọi tenant B nằm ở tenant B: A=%d B=%d", cs.decisionCount(e.tenantA), cs.decisionCount(e.tenantB))
	}
	if _, table := e.traceOf(tokDirectorB, res.TraceID); !strings.Contains(table, "BƯỚC LỆCH ĐẦU TIÊN: C1 tenant") {
		t.Errorf("vết phải dừng ở C1:\n%s", table)
	}
	// Người của tenant A không đọc được vết của tenant B (RLS).
	if r := e.trace(tokDirector, res.TraceID); r.status != http.StatusNotFound {
		t.Errorf("vết của tenant khác phải 404, được %d", r.status)
	}
}

// ---------------------------------------------------------------------------------------------------------------------------------
// A29 qua HTTP: vết của lượt lệnh đọc được bằng GET /v1/trace/{id} (cùng bảng với `eais trace`).
// ---------------------------------------------------------------------------------------------------------------------------------

func TestLenhHTTPVet(t *testing.T) {
	e, _ := newCmdEnv(t)
	done := wantResult(t, e.cmd(tokAccountant, cmdReq(1, "EX-03", dayAgo)), 200, commands.StatusDone, "", "", commands.RetryNone)
	denied := wantResult(t, e.cmd(tokDriver, cmdReq(2, "EX-03", dayAgo)), 403, commands.StatusFailed, commands.ReasonPermissionDenied, "policy", commands.RetryNone)
	dup := wantResult(t, e.cmd(tokDirector, cmdReq(3, "EX-03", dayAgo)), 409, commands.StatusFailed, commands.ReasonDuplicateOpenTicket, "data", commands.RetryNone)
	replay := wantResult(t, e.cmd(tokAccountant, cmdReq(1, "EX-03", dayAgo)), 200, commands.StatusDone, "", "", commands.RetryNone)

	for _, c := range []struct {
		name, tok, id, stop, outcome string
	}{
		{"DONE", tokAccountant, done.TraceID, "", "DONE"},
		{"phát lại", tokAccountant, replay.TraceID, "", "DONE"},
		{"thiếu quyền", tokDriver, denied.TraceID, "C4 permission", "PERMISSION_DENIED"},
		{"trùng phiếu", tokDirector, dup.TraceID, "C5 rules", "DUPLICATE_OPEN_TICKET"},
	} {
		tr, table := e.traceOf(c.tok, c.id)
		rec := tr.Record
		if !slices.Equal(stageList(rec), nineStages) {
			t.Errorf("%s: các bước = %v", c.name, stageList(rec))
		}
		if rec.Decision.Outcome != c.outcome || rec.Decision.TraceID != c.id || rec.Decision.Route != "TOOL" || rec.Decision.Intent != "open_discrepancy_ticket" {
			t.Errorf("%s: quyết định %+v", c.name, rec.Decision)
		}
		want := "Không có bước lệch."
		if c.stop != "" {
			want = "BƯỚC LỆCH ĐẦU TIÊN: " + c.stop
		}
		if !strings.Contains(table, want) {
			t.Errorf("%s: bảng vết thiếu %q:\n%s", c.name, want, table)
		}
		if len(tr.Usage) != 0 {
			t.Errorf("%s: đường lệnh không gọi model nên không có usage: %v", c.name, tr.Usage)
		}
	}
	// vết của lượt DONE mang mã phiếu và mã bằng chứng của bản chụp
	tr, _ := e.traceOf(tokAccountant, done.TraceID)
	if !strings.Contains(tr.Record.Decision.Answer, done.Ticket.ID) || len(tr.Record.Decision.Evidence) == 0 {
		t.Errorf("vết DONE phải mang mã phiếu và mã bằng chứng: %+v", tr.Record.Decision)
	}
	// phát lại: C4b ghi "phát lại", C5 và C6 skipped
	rt, _ := e.traceOf(tokAccountant, replay.TraceID)
	st := map[string]traces.Step{}
	for _, s := range rt.Record.Steps {
		st[s.Stage] = s
	}
	if st["C4b"].Decision != "phát lại" || st["C5"].Status != traces.StatusSkipped || st["C6"].Status != traces.StatusSkipped {
		t.Errorf("vết phát lại: C4b=%q C5=%q C6=%q", st["C4b"].Decision, st["C5"].Status, st["C6"].Status)
	}
	// quyền đọc vết như làn 0: kế toán (không toàn quyền) không đọc được vết của người khác
	if r := e.trace(tokAccountant, dup.TraceID); r.status != http.StatusNotFound {
		t.Errorf("người không toàn quyền đọc vết của người khác phải 404, được %d", r.status)
	}
}

// F8 (A29 (e), M25): kho vết hỏng ⇒ KẾT QUẢ LỆNH KHÔNG ĐỔI: phiếu đã ghi vẫn DONE, trace_id rỗng, không X-Trace-Id, bộ đếm lỗi lưu vết tăng; gửi lại cùng
// khoá phát lại và ghi vết mới. Khác luồng hỏi, nơi mất vết thì không giao câu trả lời.
func TestLenhHTTPMatVetKhongDoiKetQua(t *testing.T) {
	e, cs := newCmdEnv(t)
	cs.memStore.saveDecisionErrs = []error{infraErr(), infraErr(), infraErr()}
	r := e.cmd(tokDirector, cmdReq(1, "EX-03", dayAgo))
	res := decodeAs[commands.Result](t, r)
	if r.status != 200 || res.Status != commands.StatusDone || res.Ticket == nil || res.Replayed {
		t.Fatalf("mất vết không được đổi kết quả đã ghi: %d %s", r.status, r.body)
	}
	if res.TraceID != "" || r.header.Get("X-Trace-Id") != "" {
		t.Errorf("không lưu được vết thì trace_id rỗng và không X-Trace-Id: %q %q", res.TraceID, r.header.Get("X-Trace-Id"))
	}
	if got := e.srv.cnt.TraceSaveFailed.Load(); got != 1 {
		t.Errorf("bộ đếm lỗi lưu vết = %d, muốn 1", got)
	}
	if open, led, _ := cs.counts(e.tenantA); open != 1 || led != 1 {
		t.Errorf("muốn đúng 1 phiếu, 1 sổ lệnh; được %d %d", open, led)
	}
	if !strings.Contains(e.serverLog(), "KHÔNG lưu được vết của lệnh") {
		t.Error("mất vết phải có log")
	}
	again := wantResult(t, e.cmd(tokDirector, cmdReq(1, "EX-03", dayAgo)), 200, commands.StatusDone, "", "", commands.RetryNone)
	if !again.Replayed || again.Ticket.ID != res.Ticket.ID || again.TraceID == "" {
		t.Errorf("gửi lại phải phát lại cùng phiếu và có vết mới: %+v", again)
	}
}

// ---------------------------------------------------------------------------------------------------------------------------------
// Đầu cuối trên PostgreSQL thật: kho phiếu PG qua commands.NewPGStore, đếm bằng kết nối quản trị, vết đọc bằng `eais trace`.
// ---------------------------------------------------------------------------------------------------------------------------------

// dropTicketRows xoá dòng của ba bảng phiếu cho các tenant của test bằng vai quản trị (eais_app chỉ SELECT + INSERT); sổ lệnh trước phiếu vì khoá ngoại.
func dropTicketRows(t *testing.T, admin string, tenants ...string) {
	t.Helper()
	if os.Getenv("EAIS_KEEP_TEST_ROWS") != "" {
		return
	}
	t.Cleanup(func() {
		ctx := context.Background()
		conn, err := pgx.Connect(ctx, admin)
		if err != nil {
			t.Logf("dọn dòng phiếu: %v", err)
			return
		}
		defer conn.Close(ctx)
		for _, table := range []string{"commands", "audit_log", "discrepancy_tickets"} {
			if _, err := conn.Exec(ctx, "DELETE FROM "+table+" WHERE tenant_id = ANY($1)", tenants); err != nil {
				t.Logf("dọn dòng %s: %v", table, err)
			}
		}
	})
}

func TestLenhPGDauCuoi(t *testing.T) {
	st, admin := openPG(t)
	e := newEnv(t, st)
	dropTicketRows(t, admin, e.tenantA, e.tenantB)
	count := func(table, where string) int { return countRows(t, admin, table, e.tenantA, where) }

	done := wantResult(t, e.cmd(tokDirector, cmdReq(1, "EX-03", dayAgo)), 200, commands.StatusDone, "", "", commands.RetryNone)
	if done.Ticket == nil || done.Ticket.Diff != "68" {
		t.Fatalf("DONE trên PG: %+v", done.Ticket)
	}
	if n := count("discrepancy_tickets", "status = 'OPEN'"); n != 1 {
		t.Fatalf("phiếu OPEN = %d, muốn 1", n)
	}
	if n, m := count("commands", ""), count("audit_log", "outcome = 'EXECUTED' AND trace_id = '"+done.TraceID+"'"); n != 1 || m != 1 {
		t.Fatalf("sổ lệnh %d (muốn 1), EXECUTED mang trace_id của lượt %d (muốn 1)", n, m)
	}

	replay := wantResult(t, e.cmd(tokDirector, cmdReq(1, "EX-03", dayAgo)), 200, commands.StatusDone, "", "", commands.RetryNone)
	dup := wantResult(t, e.cmd(tokDirector, cmdReq(2, "EX-03", dayAgo)), 409, commands.StatusFailed, commands.ReasonDuplicateOpenTicket, "data", commands.RetryNone)
	denied := wantResult(t, e.cmd(tokDriver, cmdReq(3, "EX-03", dayAgo)), 403, commands.StatusFailed, commands.ReasonPermissionDenied, "policy", commands.RetryNone)
	if !replay.Replayed || replay.Ticket.ID != done.Ticket.ID || dup.Ticket.ID != done.Ticket.ID {
		t.Errorf("phát lại và trùng phiếu phải cùng mã phiếu: %v %v %v", done.Ticket.ID, replay.Ticket.ID, dup.Ticket.ID)
	}
	if n := count("discrepancy_tickets", "status = 'OPEN'"); n != 1 {
		t.Errorf("sau mọi tổ hợp gửi: phiếu OPEN = %d, muốn 1", n)
	}
	for outcome, want := range map[string]int{"EXECUTED": 1, "REPLAYED": 1, "REFUSED": 2} {
		if n := count("audit_log", "outcome = '"+outcome+"'"); n != want {
			t.Errorf("audit %s = %d, muốn %d", outcome, n, want)
		}
	}

	// `eais trace` đọc được lượt lệnh từ kho vết thật: chín bước, bước lệch đầu tiên đúng chặng.
	for _, c := range []struct{ id, want string }{
		{done.TraceID, "Không có bước lệch."}, {replay.TraceID, "Không có bước lệch."},
		{dup.TraceID, "BƯỚC LỆCH ĐẦU TIÊN: C5 rules"}, {denied.TraceID, "BƯỚC LỆCH ĐẦU TIÊN: C4 permission"},
	} {
		rc, out, errOut := runCLI("trace", "-tenant", e.tenantA, c.id)
		if rc != 0 || !strings.Contains(out, c.want) || errOut != "" {
			t.Fatalf("eais trace %s: rc=%d stderr=%q\n%s", c.id, rc, errOut, out)
		}
		for _, code := range nineStages {
			if !strings.Contains(out, "\n"+code+" ") && !strings.Contains(out, "  "+code+"  ") {
				t.Errorf("eais trace %s thiếu bước %s", c.id, code)
			}
		}
	}
}

// K3S-03 (soát K3): ba chỗ nối dây của K3 mà hành vi đúng nhưng không test nào ghim (đột biến sống sót).

// (a) persistTrace lưu vết bằng ngữ cảnh KHÔNG huỷ theo client: client ngắt kết nối không được làm mất vết của lượt đã chạy.
func TestLenhPersistTraceKhiClientHuy(t *testing.T) {
	e, cs := newCmdEnv(t)
	pr := e.principal(tokDirector)
	out := e.app.cmds["open_discrepancy_ticket"].Malformed()
	rec := commands.NewRecord(commands.Request{}, out, pr, e.app.now(), e.app.versions(), 0)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := e.srv.persistTrace(ctx, pr, rec); err != nil {
		t.Fatalf("persistTrace với ngữ cảnh đã huỷ: %v", err)
	}
	if got := cs.decisionCount(e.tenantA); got != 1 {
		t.Errorf("client ngắt mà có %d dòng vết, muốn 1", got)
	}
}

// (b) lệnh dùng ĐỒNG HỒ CỦA APP (-now world / clockForTest: 20/10/2026), không đồng hồ thật: "hôm nay" của tenant chưa qua nên bị từ chối ở C3.
func TestLenhNgayHomNayTheoDongHoApp(t *testing.T) {
	e, _ := newCmdEnv(t)
	res := wantResult(t, e.cmd(tokDirector, cmdReq(60, "EX-03", "2026-10-20")), 400, commands.StatusFailed, commands.ReasonInvalidRequest, "input", commands.RetryFixRequest)
	if !slices.Equal(res.InvalidFields, []string{"day"}) {
		t.Errorf("invalid_fields = %v, muốn [day]", res.InvalidFields)
	}
}

// (c) project_manager (phạm vi PRJ-A) trên pack thật qua HTTP: máy trong phạm vi ⇒ DONE; máy ngoài phạm vi và máy không tồn tại ⇒ cùng 403, cùng
// thông báo (§4.4).
func TestLenhHTTPProjectManagerTheoPhamVi(t *testing.T) {
	e, _ := newCmdEnv(t)
	wantResult(t, e.cmd(tokPM, cmdReq(61, "EX-03", dayAgo)), 200, commands.StatusDone, "", "", commands.RetryNone)
	var details []string
	for i, key := range []string{"DZ-05", "EX-99"} { // DZ-05 thuộc PRJ-B trong thế giới của test (world.yaml); EX-99 không tồn tại
		res := wantResult(t, e.cmd(tokPM, cmdReq(62+i, key, dayAgo)), 403, commands.StatusFailed, commands.ReasonPermissionDenied, "policy", commands.RetryNone)
		details = append(details, res.Detail)
	}
	if details[0] != details[1] {
		t.Errorf("máy ngoài phạm vi và máy không tồn tại phải cùng thông báo: %q / %q", details[0], details[1])
	}
}
