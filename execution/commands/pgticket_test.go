package commands

// Test của kho PG ở mức Store, không qua pipeline: lược đồ của migration, RLS, quyền chỉ-thêm, ràng buộc chống trùng, độ trung thực
// của đọc lại, và phân loại lỗi. Phần cần PG bỏ qua khi thiếu EAIS_PG_*; phần phân loại thuần (không PG) luôn chạy.

import (
	"context"
	"errors"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/luannt2002/enterprise-ai-control-layer/core/contracts"
	"github.com/luannt2002/enterprise-ai-control-layer/db"
	"github.com/luannt2002/enterprise-ai-control-layer/platform/pgstore"
)

// richUnit dựng một đơn vị ghi phiếu mà MỌI trường đều mang giá trị khó: số thập phân vượt float64 và có số 0 cuối, bằng chứng theo
// thứ tự KHÔNG sắp xếp, chữ có dấu, nhiều vai. Dùng để kiểm ReadBack trả đúng từng trường.
func richUnit(cmd int, caller, key string) WriteUnit {
	r := req(cmd, key, day1)
	tid := uuid.NewString()
	fp := fingerprint(r)
	tk := TicketRecord{
		ID: tid, Status: TicketOpen, EntityID: "g7-" + key, SubjectKey: key, Metric: r.Metric, Day: r.Day,
		Left:    Operand{Metric: "issued_fuel", Value: "123456789012345678901234567890.123456789012345678", Unit: "L"},
		Right:   Operand{Metric: "measured_consumption", Value: "0.10", Unit: "L"},
		Diff:    "123456789012345678901234567890.023456789012345678",
		Approx:  true,
		Formula: "issued_fuel − measured_consumption (đơn vị L)",
		Evidence: []EvidenceSnap{
			{ID: "fact:z-9", Kind: "fact", SourceType: "warehouse_slip", SourceRef: "phiếu-xuất #7 «ký»", Value: "3850.50", Unit: "L", Day: day1, Authority: 2},
			{ID: "calc:issued_fuel", Kind: "calc", SourceType: "", SourceRef: "", Value: "123456789012345678901234567890.123456789012345678", Unit: "L", Day: day1, Authority: 0},
			{ID: "fact:a-1", Kind: "fact", SourceType: "fuel_sensor", SourceRef: "cảm-biến-3", Value: "0.10", Unit: "L", Day: day1, Authority: 1},
		},
		CreatedBy: caller, PackVersion: "pack-ké", CodeVersion: "code-ü", CommandID: r.CommandID,
	}
	return WriteUnit{
		Ticket: tk,
		Ledger: LedgerRecord{Caller: caller, CommandID: r.CommandID, Fingerprint: fp, TicketID: tid},
		Audit: Audit{
			Outcome: AuditExecuted, Caller: Caller{ID: caller, Roles: []string{"director", "accountant"}, EntityScope: "PRJ-A"},
			CommandID: r.CommandID, Action: CommandOpenDiscrepancyTicket, SubjectKey: key, Metric: r.Metric, Day: r.Day,
			TicketID: tid, Fingerprint: fp, PackVersion: "pack-ké", CodeVersion: "code-ü", TraceID: "trace-" + tid[:8],
		},
	}
}

// sameTicket so hai phiếu từng trường, coi bằng chứng rỗng và nil là một (như pipeline: slices.Equal).
func sameTicket(a, b TicketRecord) bool {
	ea, eb := a.Evidence, b.Evidence
	a.Evidence, b.Evidence = nil, nil
	return reflect.DeepEqual(a, b) && slices.Equal(ea, eb)
}

func mustWrite(t *testing.T, e *pgEnv, tn contracts.TenantID, u WriteUnit) {
	t.Helper()
	if err := e.pg.WriteTicket(ctxOf(tn), u); err != nil {
		t.Fatalf("WriteTicket: %v", err)
	}
}

// --- độ trung thực của đọc lại -----------------------------------------------------------------------------------------

func TestPGTicketDocLaiDungTungTruong(t *testing.T) {
	e := newPGEnv(t)
	tn := e.tenant("fid")
	u := richUnit(1, "người-gọi-1", "EX-03")
	mustWrite(t, e, tn, u)

	st, err := e.pg.ReadBack(ctxOf(tn), u.Ledger.Caller, u.Ledger.CommandID, u.Ticket.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(st.Tickets) != 1 || len(st.Ledger) != 1 || len(st.Executed) != 1 {
		t.Fatalf("muốn 1 phiếu, 1 dòng sổ lệnh, 1 audit EXECUTED; được %d, %d, %d", len(st.Tickets), len(st.Ledger), len(st.Executed))
	}
	if got := st.Tickets[0]; !sameTicket(got, u.Ticket) {
		t.Errorf("phiếu đọc lại lệch từng trường:\n được %+v\n muốn %+v", got, u.Ticket)
	}
	if got := st.Tickets[0].Evidence; !slices.Equal(got, u.Ticket.Evidence) || got[0].ID != "fact:z-9" || got[2].ID != "fact:a-1" {
		t.Errorf("bằng chứng phải giữ NGUYÊN thứ tự khi ghi: %+v", got)
	}
	for _, c := range []struct{ name, got, want string }{
		{"left.value", st.Tickets[0].Left.Value, "123456789012345678901234567890.123456789012345678"},
		{"right.value (số 0 cuối)", st.Tickets[0].Right.Value, "0.10"},
		{"diff", st.Tickets[0].Diff, "123456789012345678901234567890.023456789012345678"},
	} {
		if c.got != c.want {
			t.Errorf("%s: số thập phân mất chữ số: được %q, muốn %q", c.name, c.got, c.want)
		}
	}
	if st.Ledger[0] != u.Ledger {
		t.Errorf("sổ lệnh lệch: %+v và %+v", st.Ledger[0], u.Ledger)
	}
	if !reflect.DeepEqual(st.Executed[0], u.Audit) {
		t.Errorf("audit EXECUTED lệch:\n được %+v\n muốn %+v", st.Executed[0], u.Audit)
	}
}

// Số âm, số 0, và phiếu không có bằng chứng; vai rỗng. Kiểu cột là numeric nên không có sai số dấu phẩy động.
func TestPGTicketDocLaiSoAmVaTruongRong(t *testing.T) {
	e := newPGEnv(t)
	tn := e.tenant("fid2")
	u := richUnit(2, "p-1", "EX-10")
	u.Ticket.Left.Value, u.Ticket.Right.Value, u.Ticket.Diff = "900", "950.000", "-50"
	u.Ticket.Approx = false
	u.Ticket.Evidence = nil
	u.Audit.Caller.Roles, u.Audit.Caller.EntityScope, u.Audit.TraceID = nil, "", ""
	mustWrite(t, e, tn, u)

	st, err := e.pg.ReadBack(ctxOf(tn), "p-1", u.Ledger.CommandID, u.Ticket.ID)
	if err != nil || len(st.Tickets) != 1 || len(st.Executed) != 1 {
		t.Fatalf("ReadBack: %+v, %v", st, err)
	}
	if got := st.Tickets[0]; !sameTicket(got, u.Ticket) || got.Diff != "-50" || got.Right.Value != "950.000" || got.Approx {
		t.Errorf("phiếu số âm / không bằng chứng lệch: %+v", got)
	}
	if got := st.Executed[0]; len(got.Caller.Roles) != 0 || got.Caller.EntityScope != "" || got.TraceID != "" {
		t.Errorf("trường rỗng của audit phải đọc lại rỗng: %+v", got)
	}
}

// ReadBack theo (người gọi, command_id, mã phiếu): không thấy gì thì trả rỗng, KHÔNG lỗi; chỉ audit EXECUTED được trả về, không
// lẫn REFUSED / REPLAYED / VERIFY_FAILED của cùng command_id.
func TestPGTicketDocLaiChiLayDungKhoa(t *testing.T) {
	e := newPGEnv(t)
	tn := e.tenant("rb")
	u := richUnit(3, "p-1", "EX-03")
	mustWrite(t, e, tn, u)
	for _, o := range []AuditOutcome{AuditRefused, AuditReplayed, AuditVerifyFailed} {
		a := u.Audit
		a.Outcome, a.ReasonCode, a.ReasonDetail = o, ReasonVerifyMismatch, "chi tiết thật "+string(o)
		if o == AuditReplayed {
			a.ReasonCode = ""
		}
		if err := e.pg.WriteAudit(ctxOf(tn), a); err != nil {
			t.Fatalf("WriteAudit %s: %v", o, err)
		}
	}
	st, err := e.pg.ReadBack(ctxOf(tn), "p-1", u.Ledger.CommandID, u.Ticket.ID)
	if err != nil || len(st.Executed) != 1 || st.Executed[0].Outcome != AuditExecuted {
		t.Fatalf("chỉ audit EXECUTED được trả: %+v, %v", st.Executed, err)
	}
	for name, c := range map[string]struct{ caller, cmd, ticket string }{
		"người gọi khác":   {"p-2", u.Ledger.CommandID, u.Ticket.ID},
		"command_id khác":  {"p-1", cmdID(999), u.Ticket.ID},
		"mã phiếu khác":    {"p-1", u.Ledger.CommandID, uuid.NewString()},
		"mọi khoá đều sai": {"p-2", cmdID(999), uuid.NewString()},
	} {
		got, err := e.pg.ReadBack(ctxOf(tn), c.caller, c.cmd, c.ticket)
		if err != nil {
			t.Errorf("%s: không thấy gì thì trả rỗng, không lỗi: %v", name, err)
			continue
		}
		wantT, wantL, wantE := 1, 1, 1
		switch name {
		case "người gọi khác":
			wantL, wantE = 0, 0
		case "command_id khác":
			wantL, wantE = 0, 0
		case "mã phiếu khác":
			wantT = 0
		case "mọi khoá đều sai":
			wantT, wantL, wantE = 0, 0, 0
		}
		if len(got.Tickets) != wantT || len(got.Ledger) != wantL || len(got.Executed) != wantE {
			t.Errorf("%s: được %d phiếu, %d sổ lệnh, %d EXECUTED; muốn %d, %d, %d", name, len(got.Tickets), len(got.Ledger), len(got.Executed), wantT, wantL, wantE)
		}
	}
}

func TestPGTicketTimSoLenhVaPhieuMo(t *testing.T) {
	e := newPGEnv(t)
	tn, other := e.tenant("find"), e.tenant("find-o")
	u := richUnit(4, "p-1", "EX-03")
	mustWrite(t, e, tn, u)
	ctx := ctxOf(tn)

	led, ok, err := e.pg.FindLedger(ctx, "p-1", u.Ledger.CommandID)
	if err != nil || !ok || led != u.Ledger {
		t.Fatalf("FindLedger: %+v %v %v", led, ok, err)
	}
	for name, c := range map[string]struct {
		ctx         context.Context
		caller, cmd string
	}{
		"người gọi khác":  {ctx, "p-2", u.Ledger.CommandID},
		"command_id khác": {ctx, "p-1", cmdID(998)},
		"tenant khác":     {ctxOf(other), "p-1", u.Ledger.CommandID},
	} {
		if _, ok, err := e.pg.FindLedger(c.ctx, c.caller, c.cmd); ok || err != nil {
			t.Errorf("FindLedger %s: muốn (không thấy, không lỗi), được %v, %v", name, ok, err)
		}
	}

	ref, ok, err := e.pg.FindOpenTicket(ctx, "EX-03", u.Ticket.Metric, day1)
	if err != nil || !ok || ref.ID != u.Ticket.ID || ref.Status != TicketOpen {
		t.Fatalf("FindOpenTicket: %+v %v %v", ref, ok, err)
	}
	for name, c := range map[string]struct {
		ctx              context.Context
		key, metric, day string
	}{
		"máy khác":    {ctx, "EX-04", u.Ticket.Metric, day1},
		"chỉ số khác": {ctx, "EX-03", "issued_fuel", day1},
		"ngày khác":   {ctx, "EX-03", u.Ticket.Metric, day2},
		"tenant khác": {ctxOf(other), "EX-03", u.Ticket.Metric, day1},
	} {
		if _, ok, err := e.pg.FindOpenTicket(c.ctx, c.key, c.metric, c.day); ok || err != nil {
			t.Errorf("FindOpenTicket %s: muốn (không thấy, không lỗi), được %v, %v", name, ok, err)
		}
	}
}

// --- ghi: ràng buộc chống trùng ----------------------------------------------------------------------------------------

// Phiếu OPEN thứ hai cùng (máy, chỉ số, ngày) trong cùng tenant bị chỉ mục duy nhất từng phần chặn, trả ErrUniqueViolation, và
// giao dịch bị huỷ NGUYÊN KHỐI: không sót dòng sổ lệnh hay audit của lần thua. Tenant khác cùng khoá thì qua.
func TestPGTicketPhieuOpenThuHaiBiChan(t *testing.T) {
	e := newPGEnv(t)
	tn, other := e.tenant("uq1"), e.tenant("uq1-o")
	first := richUnit(5, "p-1", "EX-03")
	mustWrite(t, e, tn, first)

	second := richUnit(6, "p-2", "EX-03") // khác phiếu, khác khoá lệnh, khác người gọi, cùng khoá nghiệp vụ
	err := e.pg.WriteTicket(ctxOf(tn), second)
	if !errors.Is(err, ErrUniqueViolation) || errors.Is(err, ErrBeforeCommit) || errors.Is(err, ErrCommitUnknown) {
		t.Fatalf("muốn đúng ErrUniqueViolation, được %v", err)
	}
	var pe *pgconn.PgError
	if !errors.As(err, &pe) || pe.Code != "23505" {
		t.Errorf("phải giữ lỗi PG gốc 23505 trong chuỗi lỗi để tra: %v", err)
	}
	if e.openTickets(tn) != 1 || e.ledgerRows(tn) != 1 || e.auditRows(tn, AuditExecuted, "", "") != 1 {
		t.Errorf("giao dịch thua phải huỷ cả ba dòng: phiếu %d, sổ lệnh %d, EXECUTED %d", e.openTickets(tn), e.ledgerRows(tn), e.auditRows(tn, AuditExecuted, "", ""))
	}
	mustWrite(t, e, other, richUnit(6, "p-2", "EX-03")) // tenant khác, cùng khoá nghiệp vụ: độc lập
	if e.openTickets(other) != 1 {
		t.Errorf("tenant khác cùng khoá nghiệp vụ phải ghi được")
	}
}

// Khoá chính của sổ lệnh (tenant, người gọi, command_id): cùng khoá, KHÁC khoá nghiệp vụ (nên phiếu INSERT qua) bị chặn ở dòng sổ
// lệnh, và phiếu vừa INSERT bị huỷ theo (không có phiếu mồ côi). Người gọi khác dùng cùng command_id thì qua.
func TestPGTicketKhoaChinhSoLenh(t *testing.T) {
	e := newPGEnv(t)
	tn := e.tenant("uq2")
	first := richUnit(7, "p-1", "EX-03")
	mustWrite(t, e, tn, first)

	clash := richUnit(7, "p-1", "EX-04") // cùng command_id, cùng người gọi, máy khác
	err := e.pg.WriteTicket(ctxOf(tn), clash)
	if !errors.Is(err, ErrUniqueViolation) {
		t.Fatalf("muốn ErrUniqueViolation, được %v", err)
	}
	var pe *pgconn.PgError
	if !errors.As(err, &pe) || pe.Code != "23505" || !strings.Contains(pe.ConstraintName, "pkey") {
		t.Errorf("phải là vi phạm khoá chính của sổ lệnh: %+v", pe)
	}
	if n := e.countWhere("discrepancy_tickets", tn, "id = $2", clash.Ticket.ID); n != 0 {
		t.Errorf("phiếu của lần thua còn sót (mồ côi): %d", n)
	}
	mustWrite(t, e, tn, richUnit(7, "p-2", "EX-04")) // người gọi khác, cùng command_id
	if e.openTickets(tn) != 2 || e.ledgerRows(tn) != 2 {
		t.Errorf("sau hai lần ghi hợp lệ: %d phiếu, %d sổ lệnh, muốn 2 và 2", e.openTickets(tn), e.ledgerRows(tn))
	}
}

// Thiếu tenant: mọi hàm từ chối trước khi chạm PG; WriteTicket từ chối bằng ErrBeforeCommit (chắc chắn chưa ghi).
func TestPGTicketTuChoiKhiThieuTenant(t *testing.T) {
	e := newPGEnv(t)
	u := richUnit(8, "p-1", "EX-03")
	bare := context.Background()
	if err := e.pg.WriteTicket(bare, u); !errors.Is(err, ErrBeforeCommit) || contracts.FaultOf(err) != contracts.FaultPolicy {
		t.Errorf("WriteTicket không tenant: muốn ErrBeforeCommit mang nhãn %s, được %v", contracts.FaultPolicy, err)
	}
	if _, _, err := e.pg.FindLedger(bare, "p-1", u.Ledger.CommandID); contracts.FaultOf(err) != contracts.FaultPolicy {
		t.Errorf("FindLedger không tenant: %v", err)
	}
	if _, _, err := e.pg.FindOpenTicket(bare, "EX-03", u.Ticket.Metric, day1); contracts.FaultOf(err) != contracts.FaultPolicy {
		t.Errorf("FindOpenTicket không tenant: %v", err)
	}
	if _, err := e.pg.ReadBack(bare, "p-1", u.Ledger.CommandID, u.Ticket.ID); contracts.FaultOf(err) != contracts.FaultPolicy {
		t.Errorf("ReadBack không tenant: %v", err)
	}
	if err := e.pg.WriteAudit(bare, u.Audit); contracts.FaultOf(err) != contracts.FaultPolicy {
		t.Errorf("WriteAudit không tenant: %v", err)
	}
}

// WriteAudit ghi MỘT dòng bằng giao dịch riêng cho REFUSED, REPLAYED, VERIFY_FAILED và giữ nguyên lý do thật; từ chối EXECUTED, vì
// EXECUTED chỉ được nằm trong đơn vị ghi phiếu (I4).
func TestPGTicketWriteAuditGiaoDichRieng(t *testing.T) {
	e := newPGEnv(t)
	tn := e.tenant("aud")
	base := richUnit(9, "p-1", "EX-03").Audit
	base.TicketID = "" // dòng từ chối không có phiếu
	for _, c := range []struct {
		outcome AuditOutcome
		code    ReasonCode
		detail  string
	}{
		{AuditRefused, ReasonPermissionDenied, "khoá thực thể không có trong danh mục"},
		{AuditReplayed, "", "phát lại: cùng khoá, cùng nội dung"},
		{AuditVerifyFailed, ReasonVerifyMismatch, "đọc lại không khớp: phiếu.diff"},
	} {
		a := base
		a.Outcome, a.ReasonCode, a.ReasonDetail = c.outcome, c.code, c.detail
		if c.outcome != AuditRefused {
			a.TicketID = uuid.NewString()
		}
		if err := e.pg.WriteAudit(ctxOf(tn), a); err != nil {
			t.Fatalf("WriteAudit %s: %v", c.outcome, err)
		}
		var code, detail, caller, roles string
		if err := e.admin.QueryRow(context.Background(),
			"SELECT reason_code, reason_detail, caller_id, array_to_string(caller_roles, ',') FROM audit_log WHERE tenant_id = $1 AND outcome = $2",
			string(tn), string(c.outcome)).Scan(&code, &detail, &caller, &roles); err != nil {
			t.Fatalf("đọc dòng audit %s: %v", c.outcome, err)
		}
		if code != string(c.code) || detail != c.detail || caller != "p-1" || roles != "director,accountant" {
			t.Errorf("%s: dòng audit lệch: code=%q detail=%q caller=%q roles=%q", c.outcome, code, detail, caller, roles)
		}
	}
	exec := base
	exec.Outcome, exec.TicketID = AuditExecuted, uuid.NewString() // có mã phiếu: chỉ lời từ chối của WriteAudit chặn được, không phải CHECK của bảng
	if err := e.pg.WriteAudit(ctxOf(tn), exec); err == nil {
		t.Errorf("WriteAudit phải từ chối EXECUTED (chỉ nằm trong đơn vị ghi phiếu)")
	}
	if n := e.auditRows(tn, AuditExecuted, "", ""); n != 0 {
		t.Errorf("không dòng EXECUTED nào được ghi ngoài đơn vị ghi phiếu: %d", n)
	}
	if e.openTickets(tn) != 0 || e.ledgerRows(tn) != 0 {
		t.Errorf("WriteAudit không được tạo phiếu hay dòng sổ lệnh")
	}
}

// Mọi đường lỗi của đơn vị ghi phải trả kết nối về pool (huỷ giao dịch trước khi trả lỗi): với pool MỘT kết nối, nếu một lần ghi hỏng giữ
// lại kết nối thì lời gọi kế tiếp đợi mãi. Kiểm sau ba kiểu hỏng: vi phạm khoá duy nhất, núm F2, ngữ cảnh đã huỷ.
func TestPGTicketKhongRoKetNoiSauLoi(t *testing.T) {
	e := newPGEnv(t)
	st, err := pgstore.Open(context.Background(), withPoolMax(e.appDSN, 1))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { closeWithin(t, "pool một kết nối", st.Close) })
	pg := NewPGStore(st)
	tn := e.tenant("leak")
	usable := func(when string) {
		t.Helper()
		ctx, cancel := context.WithTimeout(ctxOf(tn), 3*time.Second)
		defer cancel()
		if _, _, err := pg.FindLedger(ctx, "p-1", cmdID(1)); err != nil {
			t.Fatalf("%s: pool một kết nối không còn dùng được (kết nối bị giữ?): %v", when, err)
		}
	}
	if err := pg.WriteTicket(ctxOf(tn), richUnit(20, "p-1", "EX-03")); err != nil {
		t.Fatal(err)
	}
	usable("sau khi ghi thành công")
	if err := pg.WriteTicket(ctxOf(tn), richUnit(21, "p-1", "EX-03")); !errors.Is(err, ErrUniqueViolation) {
		t.Fatalf("muốn ErrUniqueViolation, được %v", err)
	}
	usable("sau vi phạm khoá duy nhất")
	pg.failAuditInsertTimes(1)
	if err := pg.WriteTicket(ctxOf(tn), richUnit(22, "p-1", "EX-04")); !errors.Is(err, ErrBeforeCommit) {
		t.Fatalf("muốn ErrBeforeCommit, được %v", err)
	}
	usable("sau núm F2")
	cancelled, cancel := context.WithCancel(ctxOf(tn))
	cancel()
	if err := pg.WriteTicket(cancelled, richUnit(23, "p-1", "EX-05")); !errors.Is(err, ErrBeforeCommit) {
		t.Fatalf("muốn ErrBeforeCommit, được %v", err)
	}
	usable("sau ngữ cảnh đã huỷ")
	if e.openTickets(tn) != 1 {
		t.Errorf("chỉ phiếu đầu được ghi, có %d phiếu", e.openTickets(tn))
	}
}

// --- núm F2: chỉ gắn được từ _test.go ----------------------------------------------------------------------------------------

// "Đường chạy thường không có móc" (HOP-DONG §10): trường beforeAuditInsert mặc định nil và KHÔNG mã sản phẩm nào gán cho nó; chỉ
// tệp _test.go (export_test.go) gắn. Quét cú pháp mọi tệp .go không phải test của gói.
func TestPGTicketNumF2ChiGanTuTest(t *testing.T) {
	if s := NewPGStore(nil); s.beforeAuditInsert != nil {
		t.Fatalf("NewPGStore phải để núm F2 là nil")
	}
	files, err := filepath.Glob("*.go")
	if err != nil || len(files) == 0 {
		t.Fatalf("không đọc được mã của gói: %v", err)
	}
	for _, f := range files {
		if strings.HasSuffix(f, "_test.go") {
			continue
		}
		af, err := parser.ParseFile(token.NewFileSet(), f, nil, 0)
		if err != nil {
			t.Fatal(err)
		}
		ast.Inspect(af, func(n ast.Node) bool {
			switch x := n.(type) {
			case *ast.AssignStmt:
				for _, l := range x.Lhs {
					if sel, ok := l.(*ast.SelectorExpr); ok && sel.Sel.Name == "beforeAuditInsert" {
						t.Errorf("%s gán núm F2 ở mã sản phẩm: chỉ _test.go được gắn", f)
					}
				}
			case *ast.KeyValueExpr:
				if id, ok := x.Key.(*ast.Ident); ok && id.Name == "beforeAuditInsert" {
					t.Errorf("%s khởi tạo núm F2 ở mã sản phẩm: chỉ _test.go được gắn", f)
				}
			}
			return true
		})
	}
}

// --- RLS, quyền chỉ-thêm, khoá ngoại (A26) -----------------------------------------------------------------------------------

var newTables = []string{"commands", "discrepancy_tickets", "audit_log"}

// Canary kiểu TestRLSCanaryB cho ba bảng mới: không dòng nào của B tới được từ A (đọc, hỏi đích danh, ghi chéo), và dòng của B CÓ
// thật trong DB (đối chứng dương: số 0 của A không phải do bảng rỗng).
func TestPGTicketRLSCanaryB(t *testing.T) {
	e := newPGEnv(t)
	a, b := e.tenant("rls-a"), e.tenant("rls-b")
	ua, ub := richUnit(10, "a-caller", "EX-03"), richUnit(10, "b-caller", "EX-03")
	mustWrite(t, e, a, ua)
	mustWrite(t, e, b, ub)
	refusedB := ub.Audit
	refusedB.Outcome, refusedB.ReasonCode, refusedB.TicketID = AuditRefused, ReasonPermissionDenied, ""
	if err := e.pg.WriteAudit(ctxOf(b), refusedB); err != nil {
		t.Fatal(err)
	}

	// qua Store: ngữ cảnh A không thấy gì của B, dù hỏi đúng khoá của B
	if _, ok, _ := e.pg.FindLedger(ctxOf(a), "b-caller", ub.Ledger.CommandID); ok {
		t.Errorf("A thấy dòng sổ lệnh của B")
	}
	if st, err := e.pg.ReadBack(ctxOf(a), "b-caller", ub.Ledger.CommandID, ub.Ticket.ID); err != nil || len(st.Tickets)+len(st.Ledger)+len(st.Executed) != 0 {
		t.Errorf("A đọc lại được dòng của B: %+v %v", st, err)
	}

	// qua kết nối trần của vai ứng dụng
	ctx := context.Background()
	app, err := pgx.Connect(ctx, e.appDSN)
	if err != nil {
		t.Fatal(err)
	}
	defer app.Close(ctx)
	count := func(guc, table, where string, args ...any) int {
		t.Helper()
		var n int
		err := pgx.BeginFunc(ctx, app, func(tx pgx.Tx) error {
			if guc != "" {
				if _, err := tx.Exec(ctx, "SELECT set_config('eais.tenant_id', $1, true)", guc); err != nil {
					return err
				}
			}
			return tx.QueryRow(ctx, "SELECT count(*) FROM "+table+" WHERE "+where, args...).Scan(&n)
		})
		if err != nil {
			t.Fatalf("đếm %s: %v", table, err)
		}
		return n
	}
	// hai chiều: A không với tới dòng của B, và B không với tới dòng của A
	pairs := []struct {
		name          string
		viewer, owner contracts.TenantID
		ownerTicket   string
	}{{"A→B", a, b, ub.Ticket.ID}, {"B→A", b, a, ua.Ticket.ID}}
	for _, table := range newTables {
		if n := count("", table, "true"); n != 0 {
			t.Errorf("%s, không đặt GUC: thấy %d dòng, muốn 0 (đóng mặc định)", table, n)
		}
		for _, pr := range pairs {
			if n := count(string(pr.viewer), table, "tenant_id = $1", string(pr.owner)); n != 0 {
				t.Errorf("%s %s: GUC của bên xem hỏi đích danh dòng của bên kia: thấy %d, muốn 0", table, pr.name, n)
			}
			if n := count(string(pr.viewer), table, "true"); n < 1 {
				t.Errorf("%s %s: không thấy dòng của chính mình (đối chứng)", table, pr.name)
			}
			if n := e.countWhere(table, pr.owner, ""); n < 1 {
				t.Errorf("%s %s: quản trị không thấy dòng của bên kia — số 0 ở trên mới không có nghĩa", table, pr.name)
			}
		}
	}
	// ghi chéo: GUC của bên này mà dòng mang tenant của bên kia → RLS WITH CHECK chặn, từng bảng, hai chiều
	inserts := map[string]string{
		"discrepancy_tickets": `INSERT INTO discrepancy_tickets (tenant_id, id, status, entity_id, subject_key, metric, day, left_metric, left_value, left_unit,
			right_metric, right_value, right_unit, diff, approx, formula, evidence, created_by, pack_version, code_version, command_id)
			VALUES ($1, 'x-1', 'OPEN', 'e', 'EX-90', 'm', '2026-10-19', 'l', 1, 'L', 'r', 1, 'L', 0, false, 'f', '[]', 'c', 'p', 'v', 'cmd')`,
		"commands": `INSERT INTO commands (tenant_id, caller_id, command_id, fingerprint, ticket_id)
			VALUES ($1, 'x', 'cmd-x', repeat('a', 64), $2)`,
		"audit_log": `INSERT INTO audit_log (tenant_id, outcome, caller_id, command_id, action, subject_key, metric, day, fingerprint, pack_version, code_version)
			VALUES ($1, 'REFUSED', 'x', 'cmd-x', 'open_discrepancy_ticket', 'EX-90', 'm', '2026-10-19', repeat('a', 64), 'p', 'v')`,
	}
	for _, table := range newTables {
		for _, pr := range pairs {
			args := []any{string(pr.owner)}
			if table == "commands" {
				args = append(args, pr.ownerTicket) // phiếu có thật của bên kia: lỗi duy nhất còn lại là RLS
			}
			err := pgx.BeginFunc(ctx, app, func(tx pgx.Tx) error {
				if _, err := tx.Exec(ctx, "SELECT set_config('eais.tenant_id', $1, true)", string(pr.viewer)); err != nil {
					return err
				}
				_, err := tx.Exec(ctx, inserts[table], args...)
				return err
			})
			var pe *pgconn.PgError
			if !errors.As(err, &pe) || pe.Code != "42501" || !strings.Contains(pe.Message, "row-level security") {
				t.Errorf("%s %s: ghi chéo tenant phải bị RLS chặn (42501), được %v", table, pr.name, err)
			}
		}
	}
	// đối chứng dương của chính câu INSERT: cùng câu, tenant của GUC, phải ghi được (rồi huỷ)
	err = pgx.BeginFunc(ctx, app, func(tx pgx.Tx) error {
		if _, err := tx.Exec(ctx, "SELECT set_config('eais.tenant_id', $1, true)", string(a)); err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, inserts["audit_log"], string(a)); err != nil {
			return err
		}
		return errors.New("huỷ có chủ ý")
	})
	if err == nil || err.Error() != "huỷ có chủ ý" {
		t.Errorf("câu INSERT audit đúng tenant phải chạy được (rồi mới bị huỷ có chủ ý): %v", err)
	}
}

// Vai ứng dụng chỉ SELECT + INSERT trên ba bảng: UPDATE, DELETE, TRUNCATE bị từ chối quyền (không phải bị RLS ém), và mọi quyền khác
// đều vắng (REFERENCES, TRIGGER, MAINTAIN, GRANT OPTION, cả mức cột, cả qua PUBLIC). RLS ENABLE + FORCE, đúng MỘT chính sách.
func TestPGTicketQuyenChiThemVaDoc(t *testing.T) {
	e := newPGEnv(t)
	tn := e.tenant("perm")
	u := richUnit(11, "p-1", "EX-03")
	mustWrite(t, e, tn, u)
	ctx := context.Background()

	privs := []string{"SELECT", "INSERT", "UPDATE", "DELETE", "TRUNCATE", "REFERENCES", "TRIGGER", "MAINTAIN"}
	for _, table := range newTables {
		for _, priv := range privs {
			var got, grant bool
			if err := e.admin.QueryRow(ctx, "SELECT has_table_privilege('eais_app', $1, $2), has_table_privilege('eais_app', $1, $2 || ' WITH GRANT OPTION')",
				table, priv).Scan(&got, &grant); err != nil {
				t.Fatal(err)
			}
			// audit_log: SELECT chỉ theo từng cột, trừ reason_detail (migration 0005, C1) ⇒ không có SELECT mức bảng
			if want := priv == "SELECT" && table != "audit_log" || priv == "INSERT"; got != want {
				t.Errorf("eais_app %s trên %s: được %v, muốn %v", priv, table, got, want)
			}
			if grant {
				t.Errorf("eais_app %s WITH GRANT OPTION trên %s: không được có", priv, table)
			}
		}
		for _, priv := range []string{"SELECT", "INSERT", "UPDATE", "REFERENCES"} { // mức cột
			var got bool
			if err := e.admin.QueryRow(ctx, "SELECT has_any_column_privilege('eais_app', $1, $2)", table, priv).Scan(&got); err != nil {
				t.Fatal(err)
			}
			if want := priv == "SELECT" || priv == "INSERT"; got != want {
				t.Errorf("eais_app %s ở mức cột trên %s: được %v, muốn %v", priv, table, got, want)
			}
		}
		var publicGrants int
		if err := e.admin.QueryRow(ctx, "SELECT count(*) FROM pg_class c, aclexplode(c.relacl) a WHERE c.oid = $1::regclass AND a.grantee = 0", table).Scan(&publicGrants); err != nil {
			t.Fatal(err)
		}
		if publicGrants != 0 {
			t.Errorf("%s: PUBLIC còn %d quyền", table, publicGrants)
		}
		var enabled, forced bool
		var policies int
		if err := e.admin.QueryRow(ctx, "SELECT relrowsecurity, relforcerowsecurity, (SELECT count(*) FROM pg_policies WHERE tablename = $1) FROM pg_class WHERE oid = $1::regclass",
			table).Scan(&enabled, &forced, &policies); err != nil {
			t.Fatal(err)
		}
		if !enabled || !forced || policies != 1 {
			t.Errorf("%s: RLS ENABLE=%v FORCE=%v, số chính sách=%d; muốn bật cả hai và đúng 1 chính sách", table, enabled, forced, policies)
		}
	}

	// hành vi thật bằng vai ứng dụng (đã đặt GUC): UPDATE, DELETE, TRUNCATE đều bị từ chối quyền
	app, err := pgx.Connect(ctx, e.appDSN)
	if err != nil {
		t.Fatal(err)
	}
	defer app.Close(ctx)
	stmts := map[string][]string{
		"commands":            {"UPDATE commands SET fingerprint = fingerprint", "DELETE FROM commands", "TRUNCATE commands"},
		"discrepancy_tickets": {"UPDATE discrepancy_tickets SET diff = 0", "DELETE FROM discrepancy_tickets", "TRUNCATE discrepancy_tickets"},
		"audit_log":           {"UPDATE audit_log SET reason_detail = ''", "DELETE FROM audit_log", "TRUNCATE audit_log"},
	}
	for table, list := range stmts {
		for _, stmt := range list {
			err := pgx.BeginFunc(ctx, app, func(tx pgx.Tx) error {
				if _, err := tx.Exec(ctx, "SELECT set_config('eais.tenant_id', $1, true)", string(tn)); err != nil {
					return err
				}
				_, err := tx.Exec(ctx, stmt)
				return err
			})
			var pe *pgconn.PgError
			if !errors.As(err, &pe) || pe.Code != "42501" || !strings.Contains(pe.Message, "permission denied") {
				t.Errorf("%s (%s): muốn bị từ chối quyền (42501 permission denied), được %v", stmt, table, err)
			}
		}
	}
	if e.openTickets(tn) != 1 || e.ledgerRows(tn) != 1 || e.auditRows(tn, AuditExecuted, "", "") != 1 {
		t.Errorf("các lệnh bị từ chối không được để lại dấu: phiếu %d, sổ lệnh %d, audit %d", e.openTickets(tn), e.ledgerRows(tn), e.auditRows(tn, AuditExecuted, "", ""))
	}
}

// C1 (lệnh ca chiều 23/10): reason_detail của audit_log giữ lý do THẬT của một lần từ chối (phân biệt "không tồn tại" với "ngoài phạm vi",
// §4.4) — vai ứng dụng chỉ được GHI cột đó, không được ĐỌC; các cột khác vẫn đọc được (đọc lại sau ghi dùng chúng).
func TestPGAuditLyDoThatChiGhi(t *testing.T) {
	e := newPGEnv(t)
	tn := e.tenant("lydo")
	mustWrite(t, e, tn, richUnit(13, "p-1", "EX-03"))
	ctx := context.Background()
	var detail, code bool
	if err := e.admin.QueryRow(ctx, `SELECT has_column_privilege('eais_app', 'audit_log', 'reason_detail', 'SELECT'),
		has_column_privilege('eais_app', 'audit_log', 'reason_code', 'SELECT')`).Scan(&detail, &code); err != nil {
		t.Fatal(err)
	}
	if detail || !code {
		t.Errorf("eais_app đọc reason_detail = %v (muốn false), đọc reason_code = %v (muốn true)", detail, code)
	}
	app, err := pgx.Connect(ctx, e.appDSN)
	if err != nil {
		t.Fatal(err)
	}
	defer app.Close(ctx)
	inTenant := func(q string) (int, error) {
		var n int
		err := pgx.BeginFunc(ctx, app, func(tx pgx.Tx) error {
			if _, err := tx.Exec(ctx, "SELECT set_config('eais.tenant_id', $1, true)", string(tn)); err != nil {
				return err
			}
			return tx.QueryRow(ctx, q).Scan(&n)
		})
		return n, err
	}
	_, err = inTenant("SELECT length(reason_detail) FROM audit_log LIMIT 1")
	var pe *pgconn.PgError
	if !errors.As(err, &pe) || pe.Code != "42501" {
		t.Errorf("vai ứng dụng đọc reason_detail: muốn 42501, được %v", err)
	}
	if n, err := inTenant("SELECT count(*) FROM audit_log WHERE outcome = 'EXECUTED' AND reason_code = ''"); err != nil || n != 1 {
		t.Errorf("vai ứng dụng đọc các cột khác của audit_log: %d dòng, lỗi %v; muốn 1, không lỗi", n, err)
	}
}

// C3 (lệnh ca chiều 23/10): ba bảng của 0004 không nhận tenant_id rỗng, kể cả qua vai quản trị (vượt RLS) — ràng buộc ở lược đồ, không chỉ
// ở tầng Go. Mỗi bảng: chép một dòng thật sang tenant rỗng phải bị CHECK chặn (23514). Chạy trong giao dịch rồi huỷ: không để lại dòng nào.
func TestPGBaBangKhongNhanTenantRong(t *testing.T) {
	e := newPGEnv(t)
	tn := e.tenant("rong")
	u := richUnit(14, "p-1", "EX-03")
	mustWrite(t, e, tn, u)
	ctx := context.Background()
	stmts := []struct{ table, sql string }{
		{"discrepancy_tickets", "INSERT INTO discrepancy_tickets SELECT * FROM rong_t"},
		{"commands", "INSERT INTO commands SELECT * FROM rong_c"},
		{"audit_log", "INSERT INTO audit_log OVERRIDING SYSTEM VALUE SELECT * FROM rong_a"},
	}
	errRollback := errors.New("huỷ có chủ ý")
	err := pgx.BeginFunc(ctx, e.admin, func(tx pgx.Tx) error {
		for _, q := range []string{
			"CREATE TEMP TABLE rong_t ON COMMIT DROP AS SELECT * FROM discrepancy_tickets WHERE tenant_id = $1",
			"CREATE TEMP TABLE rong_c ON COMMIT DROP AS SELECT * FROM commands WHERE tenant_id = $1",
			"CREATE TEMP TABLE rong_a ON COMMIT DROP AS SELECT * FROM audit_log WHERE tenant_id = $1 AND outcome = 'EXECUTED'",
		} {
			if _, err := tx.Exec(ctx, q, string(tn)); err != nil {
				return err
			}
		}
		for _, q := range []string{"UPDATE rong_t SET tenant_id = ''", "UPDATE rong_c SET tenant_id = ''", "UPDATE rong_a SET tenant_id = '', id = id + 1000000"} {
			if _, err := tx.Exec(ctx, q); err != nil {
				return err
			}
		}
		for _, st := range stmts {
			sp, err := tx.Begin(ctx)
			if err != nil {
				return err
			}
			_, err = sp.Exec(ctx, st.sql)
			var pe *pgconn.PgError
			if !errors.As(err, &pe) || pe.Code != "23514" {
				t.Errorf("%s nhận dòng tenant_id rỗng: muốn CHECK chặn (23514), được %v", st.table, err)
			}
			if err == nil { // lọt thì giữ dòng tới cuối giao dịch để bảng sau (khoá ngoại tới phiếu) đo đúng CHECK của nó
				err = sp.Commit(ctx)
			} else {
				err = sp.Rollback(ctx)
			}
			if err != nil {
				return err
			}
		}
		return errRollback
	})
	if !errors.Is(err, errRollback) {
		t.Fatalf("giao dịch thử: %v", err)
	}
}

// A26: dòng sổ lệnh trỏ tới phiếu không có bị khoá ngoại từ chối; khoá ngoại là THƯỜNG (không DEFERRABLE) nên phiếu phải có TRƯỚC —
// đảo thứ tự INSERT trong cùng giao dịch cũng bị từ chối ngay; không khoá ngoại ngược từ phiếu tới sổ lệnh và không khoá ngoại nào
// tới bốn bảng mà Load thay thế.
func TestPGTicketSoLenhMoCoiVaKhoaNgoai(t *testing.T) {
	e := newPGEnv(t)
	tn := e.tenant("fk")
	u := richUnit(12, "p-1", "EX-03")
	mustWrite(t, e, tn, u)
	ctx := context.Background()

	app, err := pgx.Connect(ctx, e.appDSN)
	if err != nil {
		t.Fatal(err)
	}
	defer app.Close(ctx)
	insertLedger := func(tx pgx.Tx, cmd, ticket string) error {
		_, err := tx.Exec(ctx, "INSERT INTO commands (tenant_id, caller_id, command_id, fingerprint, ticket_id) VALUES (current_setting('eais.tenant_id'), 'p-9', $1, repeat('b', 64), $2)", cmd, ticket)
		return err
	}
	inTx := func(fn func(tx pgx.Tx) error) error {
		return pgx.BeginFunc(ctx, app, func(tx pgx.Tx) error {
			if _, err := tx.Exec(ctx, "SELECT set_config('eais.tenant_id', $1, true)", string(tn)); err != nil {
				return err
			}
			if err := fn(tx); err != nil {
				return err
			}
			return errors.New("huỷ có chủ ý") // mọi ca dưới đây chỉ thăm dò: không giữ lại dòng nào
		})
	}
	fkCode := func(err error) string {
		var pe *pgconn.PgError
		if errors.As(err, &pe) {
			return pe.Code
		}
		return fmt.Sprint(err)
	}
	// đối chứng dương: trỏ tới phiếu CÓ THẬT thì qua (rồi bị huỷ có chủ ý)
	if err := inTx(func(tx pgx.Tx) error { return insertLedger(tx, cmdID(500), u.Ticket.ID) }); fkCode(err) != "huỷ có chủ ý" {
		t.Errorf("sổ lệnh trỏ tới phiếu có thật phải ghi được: %v", err)
	}
	// phiếu không tồn tại → 23503
	if err := inTx(func(tx pgx.Tx) error { return insertLedger(tx, cmdID(501), uuid.NewString()) }); fkCode(err) != "23503" {
		t.Errorf("sổ lệnh trỏ tới phiếu không có: muốn 23503, được %v", err)
	}
	// phiếu của tenant khác (cùng mã) không thoả khoá ngoại ghép (tenant_id, ticket_id)
	other := e.tenant("fk-o")
	uo := richUnit(12, "p-1", "EX-03")
	mustWrite(t, e, other, uo)
	if err := inTx(func(tx pgx.Tx) error { return insertLedger(tx, cmdID(502), uo.Ticket.ID) }); fkCode(err) != "23503" {
		t.Errorf("sổ lệnh trỏ tới phiếu của tenant khác: muốn 23503, được %v", err)
	}
	// khoá ngoại thường: đảo thứ tự (sổ lệnh trước, phiếu sau) trong cùng giao dịch bị từ chối ngay, không chờ COMMIT
	if err := inTx(func(tx pgx.Tx) error {
		later := richUnit(13, "p-9", "EX-04")
		if err := insertLedger(tx, cmdID(503), later.Ticket.ID); err != nil {
			return err
		}
		return errors.New("lẽ ra không tới đây")
	}); fkCode(err) != "23503" {
		t.Errorf("đảo thứ tự INSERT phải bị khoá ngoại thường từ chối ngay: %v", err)
	}

	// cấu trúc khoá ngoại trong catalog
	var deferrable, validated bool
	if err := e.admin.QueryRow(ctx, `SELECT condeferrable, convalidated FROM pg_constraint
		WHERE conrelid = 'commands'::regclass AND confrelid = 'discrepancy_tickets'::regclass AND contype = 'f'`).Scan(&deferrable, &validated); err != nil {
		t.Fatalf("không thấy khoá ngoại sổ lệnh → phiếu: %v", err)
	}
	if deferrable || !validated {
		t.Errorf("khoá ngoại sổ lệnh → phiếu: DEFERRABLE=%v VALIDATED=%v; muốn thường (không hoãn) và đã kiểm", deferrable, validated)
	}
	var back, toLoad int
	if err := e.admin.QueryRow(ctx, `SELECT
		(SELECT count(*) FROM pg_constraint WHERE conrelid = 'discrepancy_tickets'::regclass AND contype = 'f'),
		(SELECT count(*) FROM pg_constraint WHERE contype = 'f' AND conrelid = ANY (ARRAY['commands','discrepancy_tickets','audit_log']::regclass[])
			AND confrelid = ANY (ARRAY['entities','entity_aliases','facts','documents']::regclass[]))`).Scan(&back, &toLoad); err != nil {
		t.Fatal(err)
	}
	if back != 0 || toLoad != 0 {
		t.Errorf("khoá ngoại ngược từ phiếu = %d, khoá ngoại tới bảng mà Load thay thế = %d; muốn 0 và 0", back, toLoad)
	}
}

// Kiểu cột và ràng buộc của lược đồ: số lượng / chênh lệch là numeric (không float), ngày là date; trạng thái bị CHECK = OPEN; chỉ mục
// duy nhất từng phần đúng khoá và đúng điều kiện; fingerprint đúng dạng sha256.
func TestPGTicketLuocDoKieuVaRangBuoc(t *testing.T) {
	e := newPGEnv(t)
	tn := e.tenant("schema")
	ctx := context.Background()

	rows, err := e.admin.Query(ctx, `SELECT table_name, column_name, data_type FROM information_schema.columns
		WHERE table_schema = current_schema() AND table_name = ANY ($1)`, newTables)
	if err != nil {
		t.Fatal(err)
	}
	types := map[string]string{}
	for rows.Next() {
		var tbl, col, typ string
		if err := rows.Scan(&tbl, &col, &typ); err != nil {
			t.Fatal(err)
		}
		types[tbl+"."+col] = typ
		if typ == "real" || typ == "double precision" {
			t.Errorf("%s.%s kiểu %s: tiền và số lượng không được lưu bằng số dấu phẩy động", tbl, col, typ)
		}
	}
	rows.Close()
	for col, want := range map[string]string{
		"discrepancy_tickets.left_value": "numeric", "discrepancy_tickets.right_value": "numeric", "discrepancy_tickets.diff": "numeric",
		"discrepancy_tickets.day": "date", "audit_log.day": "date", "discrepancy_tickets.approx": "boolean",
		"discrepancy_tickets.evidence": "jsonb", "discrepancy_tickets.created_at": "timestamp with time zone",
	} {
		if types[col] != want {
			t.Errorf("%s: kiểu %q, muốn %q", col, types[col], want)
		}
	}
	for _, col := range []string{"commands.tenant_id", "discrepancy_tickets.tenant_id", "audit_log.tenant_id"} {
		if types[col] != "text" {
			t.Errorf("%s: mọi bảng mới phải có tenant_id kiểu text, được %q", col, types[col])
		}
	}

	var def string
	if err := e.admin.QueryRow(ctx, "SELECT indexdef FROM pg_indexes WHERE tablename = 'discrepancy_tickets' AND indexdef LIKE 'CREATE UNIQUE INDEX%' AND indexdef LIKE '%WHERE%'").Scan(&def); err != nil {
		t.Fatalf("không thấy chỉ mục duy nhất từng phần của phiếu: %v", err)
	}
	if !strings.Contains(def, "(tenant_id, subject_key, metric, day)") || !strings.Contains(def, "status = 'OPEN'") {
		t.Errorf("chỉ mục duy nhất từng phần sai khoá hoặc sai điều kiện: %s", def)
	}
	var pk string
	if err := e.admin.QueryRow(ctx, "SELECT pg_get_constraintdef(oid) FROM pg_constraint WHERE conrelid = 'commands'::regclass AND contype = 'p'").Scan(&pk); err != nil || pk != "PRIMARY KEY (tenant_id, caller_id, command_id)" {
		t.Errorf("khoá chính của sổ lệnh = %q (%v), muốn PRIMARY KEY (tenant_id, caller_id, command_id)", pk, err)
	}

	// CHECK qua vai ứng dụng: status khác OPEN, fingerprint sai dạng, outcome lạ, EXECUTED không có phiếu
	app, err := pgx.Connect(ctx, e.appDSN)
	if err != nil {
		t.Fatal(err)
	}
	defer app.Close(ctx)
	for name, c := range map[string]struct{ stmt string }{
		"status khác OPEN": {`INSERT INTO discrepancy_tickets (tenant_id, id, status, entity_id, subject_key, metric, day, left_metric, left_value, left_unit,
			right_metric, right_value, right_unit, diff, approx, formula, evidence, created_by, pack_version, code_version, command_id)
			VALUES (current_setting('eais.tenant_id'), 'x-1', 'CLOSED', 'e', 'k', 'm', '2026-10-19', 'l', 1, 'L', 'r', 1, 'L', 0, false, 'f', '[]', 'c', 'p', 'v', 'cmd')`},
		"evidence không phải mảng": {`INSERT INTO discrepancy_tickets (tenant_id, id, status, entity_id, subject_key, metric, day, left_metric, left_value, left_unit,
			right_metric, right_value, right_unit, diff, approx, formula, evidence, created_by, pack_version, code_version, command_id)
			VALUES (current_setting('eais.tenant_id'), 'x-2', 'OPEN', 'e', 'k', 'm', '2026-10-19', 'l', 1, 'L', 'r', 1, 'L', 0, false, 'f', '{}', 'c', 'p', 'v', 'cmd')`},
		"outcome lạ": {`INSERT INTO audit_log (tenant_id, outcome, caller_id, command_id, action, subject_key, metric, day, fingerprint, pack_version, code_version)
			VALUES (current_setting('eais.tenant_id'), 'DELETED', 'x', 'c', 'a', 'k', 'm', '2026-10-19', repeat('a', 64), 'p', 'v')`},
		"EXECUTED không có phiếu": {`INSERT INTO audit_log (tenant_id, outcome, caller_id, command_id, action, subject_key, metric, day, fingerprint, pack_version, code_version)
			VALUES (current_setting('eais.tenant_id'), 'EXECUTED', 'x', 'c', 'a', 'k', 'm', '2026-10-19', repeat('a', 64), 'p', 'v')`},
		"fingerprint sai dạng": {`INSERT INTO audit_log (tenant_id, outcome, caller_id, command_id, action, subject_key, metric, day, fingerprint, pack_version, code_version)
			VALUES (current_setting('eais.tenant_id'), 'REFUSED', 'x', 'c', 'a', 'k', 'm', '2026-10-19', 'ABC', 'p', 'v')`},
	} {
		err := pgx.BeginFunc(ctx, app, func(tx pgx.Tx) error {
			if _, err := tx.Exec(ctx, "SELECT set_config('eais.tenant_id', $1, true)", string(tn)); err != nil {
				return err
			}
			_, err := tx.Exec(ctx, c.stmt)
			return err
		})
		var pe *pgconn.PgError
		if !errors.As(err, &pe) || pe.Code != "23514" {
			t.Errorf("%s: muốn vi phạm CHECK (23514), được %v", name, err)
		}
	}
}

// catalogState là dấu vết trạng thái ĐÍCH của migration 0004 trong catalog: quyền từng bên nhận (kể cả PUBLIC), RLS, chính sách, chỉ mục,
// ràng buộc của ba bảng. Hai trạng thái bằng chữ nghĩa là cùng một trạng thái.
const catalogStateSQL = `
SELECT string_agg(x, E'\n' ORDER BY x) FROM (
  SELECT 'acl ' || c.relname || ' ' || coalesce(nullif(a.grantee, 0)::regrole::text, 'PUBLIC') || ' ' || a.privilege_type || ' ' || a.is_grantable
    FROM pg_class c, aclexplode(c.relacl) a WHERE c.relname = ANY ($1)
  UNION ALL SELECT 'colacl ' || c.relname || '.' || at.attname || ' ' || coalesce(nullif(a.grantee, 0)::regrole::text, 'PUBLIC') || ' ' ||
      a.privilege_type || ' ' || a.is_grantable
    FROM pg_class c JOIN pg_attribute at ON at.attrelid = c.oid AND at.attnum > 0 AND NOT at.attisdropped, aclexplode(at.attacl) a
    WHERE c.relname = ANY ($1)
  UNION ALL SELECT 'rls ' || relname || ' enable=' || relrowsecurity || ' force=' || relforcerowsecurity FROM pg_class WHERE relname = ANY ($1)
  UNION ALL SELECT 'policy ' || tablename || ' ' || policyname || ' ' || permissive || ' ' || cmd || ' ' || coalesce(qual, '') || ' ' || coalesce(with_check, '')
    FROM pg_policies WHERE tablename = ANY ($1)
  UNION ALL SELECT 'index ' || indexdef FROM pg_indexes WHERE tablename = ANY ($1)
  UNION ALL SELECT 'constraint ' || conrelid::regclass || ' ' || pg_get_constraintdef(oid) FROM pg_constraint WHERE conrelid = ANY ($2::regclass[])
) s(x)`

// Migration 0004 chạy lại được (đầu tệp 0004 hứa vậy): trên database tạm, chạy lại trần tệp hai lần không lỗi và không đổi trạng thái đích;
// rồi cho database "trôi" khỏi trạng thái đích (quyền cấp thêm, tắt FORCE hay RLS, xoá hoặc mở toang chính sách) — mỗi kiểu trôi phải
// THẤY được trong catalogState (ca âm của chính bộ so) — và chạy lại tệp phải đưa về đúng trạng thái đích. Từ 0005 (C1, C3 của lệnh ca
// chiều 23/10) trạng thái đích của ba bảng là 0004 cộng 0005, nên "chạy lại" là chạy chuỗi 0004 rồi 0005.
func TestPGTicketMigration0004ChayLai(t *testing.T) {
	e := newScratchEnv(t, "mig")
	ctx := context.Background()
	var sqlText []byte
	for _, f := range []string{"migrations/0004_phieu_chenh_lech.sql", "migrations/0005_audit_va_tenant.sql"} {
		b, err := db.Migrations.ReadFile(f)
		if err != nil {
			t.Fatal(err)
		}
		sqlText = append(append(sqlText, b...), '\n')
	}
	state := func() string {
		t.Helper()
		var s string
		if err := e.admin.QueryRow(ctx, catalogStateSQL, newTables, newTables).Scan(&s); err != nil {
			t.Fatal(err)
		}
		return s
	}
	want := state()
	if !strings.Contains(want, "policy discrepancy_tickets tenant_isolation") || !strings.Contains(want, "force=true") || !strings.Contains(want, "acl commands eais_app INSERT false") {
		t.Fatalf("trạng thái đích sau Migrate thiếu chính sách, FORCE hoặc quyền INSERT của vai ứng dụng:\n%s", want)
	}
	for i := 1; i <= 2; i++ {
		if _, err := e.admin.Exec(ctx, string(sqlText)); err != nil {
			t.Fatalf("chạy lại trần tệp 0004 lần %d: %v", i, err)
		}
		if got := state(); got != want {
			t.Fatalf("chạy lại trần 0004 lần %d đổi trạng thái:\n--- trước\n%s\n--- sau\n%s", i, want, got)
		}
	}
	for _, d := range []struct{ name, stmt string }{
		{"cấp thêm UPDATE cho vai ứng dụng", "GRANT UPDATE ON audit_log TO eais_app"},
		{"cấp lại SELECT mức bảng của audit_log (đọc được reason_detail)", "GRANT SELECT ON audit_log TO eais_app"},
		{"cấp quyền đọc riêng cột reason_detail", "GRANT SELECT (reason_detail) ON audit_log TO eais_app"},
		{"cấp thêm DELETE cho vai ứng dụng", "GRANT DELETE ON commands TO eais_app"},
		{"cấp quyền cho PUBLIC", "GRANT SELECT ON discrepancy_tickets TO PUBLIC"},
		{"tắt FORCE", "ALTER TABLE commands NO FORCE ROW LEVEL SECURITY"},
		{"tắt RLS", "ALTER TABLE audit_log DISABLE ROW LEVEL SECURITY"},
		{"xoá chính sách", "DROP POLICY tenant_isolation ON audit_log"},
		{"chính sách mở toang", "ALTER POLICY tenant_isolation ON discrepancy_tickets USING (true) WITH CHECK (true)"},
	} {
		if _, err := e.admin.Exec(ctx, d.stmt); err != nil {
			t.Fatalf("%s: %v", d.name, err)
		}
		if state() == want {
			t.Errorf("%s: bộ so trạng thái không thấy sự trôi (ca âm của chính bộ so)", d.name)
		}
		if _, err := e.admin.Exec(ctx, string(sqlText)); err != nil {
			t.Fatalf("chạy lại 0004 sau %q: %v", d.name, err)
		}
		if got := state(); got != want {
			t.Errorf("chạy lại 0004 không đưa về trạng thái đích sau %q:\n--- đích\n%s\n--- được\n%s", d.name, want, got)
		}
	}
}

// --- phân loại lỗi: thuần, không cần PG -------------------------------------------------------------------------------------

// retryableErr giả một lỗi "chưa gửi gì tới máy chủ" theo đúng giao diện mà pgconn.SafeToRetry tìm (ví dụ ngữ cảnh đã huỷ trước COMMIT).
type retryableErr struct{ safe bool }

func (e retryableErr) Error() string     { return "lỗi giả" }
func (e retryableErr) SafeToRetry() bool { return e.safe }

// lockLikeErr giả pgconn.connLockError "conn closed": SafeToRetry() = true và mở ra pgconn.ErrConnClosed.
type lockLikeErr struct{}

func (lockLikeErr) Error() string     { return "conn closed" }
func (lockLikeErr) SafeToRetry() bool { return true }
func (lockLikeErr) Unwrap() error     { return pgconn.ErrConnClosed }

func pgErr(code, severity string) error {
	return &pgconn.PgError{Code: code, Severity: severity, SeverityUnlocalized: severity, Message: "lỗi giả"}
}

// Bảng phân loại khớp bảng ĐO của U1 (HOP-DONG §15): lỗi một câu lệnh trước COMMIT luôn là "chắc chắn chưa ghi"; lỗi của chính
// COMMIT chỉ là "chắc chắn chưa ghi" khi COMMIT chưa rời máy (SafeToRetry) hoặc PG trả một lỗi mức ERROR (giao dịch đã huỷ); mọi
// trường hợp còn lại là không biết. Vi phạm khoá duy nhất 23505 ở đâu cũng là ErrUniqueViolation.
func TestPGTicketPhanLoaiLoi(t *testing.T) {
	deadline := context.DeadlineExceeded
	type row struct {
		name   string
		err    error
		stmt   error // sentinel mong đợi khi lỗi đến từ MỘT CÂU LỆNH trước COMMIT
		commit error // sentinel mong đợi khi lỗi đến từ chính COMMIT
	}
	for _, c := range []row{
		{"23505", pgErr("23505", "ERROR"), ErrUniqueViolation, ErrUniqueViolation},
		{"23505 bọc nhiều lớp", fmt.Errorf("lớp ngoài: %w", fmt.Errorf("lớp giữa: %w", pgErr("23505", "ERROR"))), ErrUniqueViolation, ErrUniqueViolation},
		{"lỗi PG thường (22P02)", pgErr("22P02", "ERROR"), ErrBeforeCommit, ErrBeforeCommit},
		{"lock_timeout (55P03)", pgErr("55P03", "ERROR"), ErrBeforeCommit, ErrBeforeCommit},
		{"statement_timeout (57014)", pgErr("57014", "ERROR"), ErrBeforeCommit, ErrBeforeCommit},
		{"FATAL 57P01 (kết thúc backend)", pgErr("57P01", "FATAL"), ErrBeforeCommit, ErrCommitUnknown},
		{"PANIC", pgErr("XX000", "PANIC"), ErrBeforeCommit, ErrCommitUnknown},
		{"hết hạn khi đang chờ", fmt.Errorf("timeout: %w", deadline), ErrBeforeCommit, ErrCommitUnknown},
		{"ngữ cảnh bị huỷ giữa chừng", context.Canceled, ErrBeforeCommit, ErrCommitUnknown},
		{"đứt kết nối", errors.New("unexpected EOF"), ErrBeforeCommit, ErrCommitUnknown},
		{"chưa gửi gì (SafeToRetry)", retryableErr{safe: true}, ErrBeforeCommit, ErrBeforeCommit},
		{"conn closed mang SafeToRetry (không đáng tin sau khi gửi COMMIT)", lockLikeErr{}, ErrBeforeCommit, ErrCommitUnknown},
		{"conn closed bọc nhiều lớp", fmt.Errorf("COMMIT: %w", lockLikeErr{}), ErrBeforeCommit, ErrCommitUnknown},
		{"đã gửi một phần (không SafeToRetry)", retryableErr{safe: false}, ErrBeforeCommit, ErrCommitUnknown},
		{"COMMIT trả tag ROLLBACK", pgx.ErrTxCommitRollback, ErrBeforeCommit, ErrBeforeCommit},
		// C7 (lệnh ca chiều 23/10): chỉ mức ERROR mới là "chắc chắn chưa ghi"; mức rỗng, lạ hay chỉ có bản dịch ⇒ không biết
		{"mức nghiêm trọng rỗng", pgErr("40001", ""), ErrBeforeCommit, ErrCommitUnknown},
		{"mức nghiêm trọng lạ", pgErr("XX000", "WEIRD"), ErrBeforeCommit, ErrCommitUnknown},
		{"WARNING", pgErr("01000", "WARNING"), ErrBeforeCommit, ErrCommitUnknown},
		{"chỉ có mức đã dịch", &pgconn.PgError{Code: "40001", Severity: "LỖI", Message: "lỗi giả"}, ErrBeforeCommit, ErrCommitUnknown},
		// C17S-10: bản không dịch thắng bản đã dịch (máy chủ khác ngôn ngữ), và khi thiếu bản không dịch thì lùi về bản đã dịch "ERROR"
		{"bản dịch FEHLER, bản gốc ERROR", &pgconn.PgError{Code: "40001", Severity: "FEHLER", SeverityUnlocalized: "ERROR", Message: "lỗi giả"}, ErrBeforeCommit, ErrBeforeCommit},
		{"chỉ có Severity ERROR (không bản gốc)", &pgconn.PgError{Code: "40001", Severity: "ERROR", Message: "lỗi giả"}, ErrBeforeCommit, ErrBeforeCommit},
	} {
		t.Run(c.name, func(t *testing.T) {
			got := classifyStatement(c.err)
			if !errors.Is(got, c.stmt) || !errors.Is(got, c.err) {
				t.Errorf("câu lệnh: muốn %v và còn lỗi gốc, được %v", c.stmt, got)
			}
			for _, other := range []error{ErrBeforeCommit, ErrCommitUnknown, ErrUniqueViolation} {
				if other != c.stmt && errors.Is(got, other) {
					t.Errorf("câu lệnh: không được vừa là %v vừa là %v", c.stmt, other)
				}
			}
			got = classifyCommit(c.err)
			if !errors.Is(got, c.commit) || !errors.Is(got, c.err) {
				t.Errorf("COMMIT: muốn %v và còn lỗi gốc, được %v", c.commit, got)
			}
			for _, other := range []error{ErrBeforeCommit, ErrCommitUnknown, ErrUniqueViolation} {
				if other != c.commit && errors.Is(got, other) {
					t.Errorf("COMMIT: không được vừa là %v vừa là %v", c.commit, other)
				}
			}
		})
	}
	if classifyCommit(nil) != nil {
		t.Errorf("COMMIT thành công không phải lỗi")
	}
}
