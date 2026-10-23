package commands

// Ca của vòng sửa bước K1 theo bản soát độc lập vòng 1, nhóm đọc lại và chữ của hợp đồng: K1S-05 (bảng ca "ReadBack lệch đúng một chỗ",
// vân tay theo chỉ số, ghim chữ của §2.4) và K1S-15 (mảng mã bằng chứng). Mã đọc lại đã so đủ từ trước; những ca này là để một đột biến
// bỏ bớt một phép so thì có ca đỏ (I3: DONE chỉ khi đọc lại thấy đúng, §7.2).

import (
	"context"
	"slices"
	"testing"

	"github.com/luannt2002/enterprise-ai-control-layer/data/sourceoftruth"
)

// readMutator bọc Store để ReadBack nói sai đúng MỘT chỗ (mut) hoặc báo lỗi (readErr). ReadBack của kho giả trả bản sao nên sửa kết
// quả ở đây không đụng dữ liệu của kho.
type readMutator struct {
	Store
	mut     func(*Stored)
	readErr error
}

func (m *readMutator) ReadBack(ctx context.Context, caller, commandID, ticketID string) (Stored, error) {
	if m.readErr != nil {
		return Stored{}, m.readErr
	}
	st, err := m.Store.ReadBack(ctx, caller, commandID, ticketID)
	if err == nil && m.mut != nil {
		m.mut(&st)
	}
	return st, err
}

type readRow struct {
	name string
	mut  func(*Stored)
	want string // chỗ lệch mà bước C7 phải nêu, đúng một chỗ
}

// ticketFieldRows là mọi trường phiếu mà diffTicket so, mỗi trường một dòng (trường left và right: từng thành phần).
func ticketFieldRows() []readRow {
	t := func(f func(*TicketRecord)) func(*Stored) { return func(s *Stored) { f(&s.Tickets[0]) } }
	return []readRow{
		{"id", t(func(r *TicketRecord) { r.ID = "ma-phieu-khac" }), "phiếu.id"},
		{"status", t(func(r *TicketRecord) { r.Status = "CLOSED" }), "phiếu.status"},
		{"entity_id", t(func(r *TicketRecord) { r.EntityID = "ma-thuc-the-khac" }), "phiếu.entity_id"},
		{"subject_key", t(func(r *TicketRecord) { r.SubjectKey = "EX-99" }), "phiếu.subject_key"},
		{"metric", t(func(r *TicketRecord) { r.Metric = "chi_so_khac" }), "phiếu.metric"},
		{"day", t(func(r *TicketRecord) { r.Day = "2026-10-01" }), "phiếu.day"},
		{"left.metric", t(func(r *TicketRecord) { r.Left.Metric = "khac" }), "phiếu.left"},
		{"left.value", t(func(r *TicketRecord) { r.Left.Value = "1" }), "phiếu.left"},
		{"left.unit", t(func(r *TicketRecord) { r.Left.Unit = "kg" }), "phiếu.left"},
		{"right.metric", t(func(r *TicketRecord) { r.Right.Metric = "khac" }), "phiếu.right"},
		{"right.value", t(func(r *TicketRecord) { r.Right.Value = "1" }), "phiếu.right"},
		{"right.unit", t(func(r *TicketRecord) { r.Right.Unit = "kg" }), "phiếu.right"},
		{"diff", t(func(r *TicketRecord) { r.Diff = "999" }), "phiếu.diff"},
		{"approx", t(func(r *TicketRecord) { r.Approx = !r.Approx }), "phiếu.approx"},
		{"formula", t(func(r *TicketRecord) { r.Formula = "cong-thuc-khac" }), "phiếu.formula"},
		{"evidence: thiếu một mục", t(func(r *TicketRecord) { r.Evidence = r.Evidence[:len(r.Evidence)-1] }), "phiếu.evidence"},
		{"evidence: lệch giá trị", t(func(r *TicketRecord) { r.Evidence[0].Value += "1" }), "phiếu.evidence"},
		{"evidence: lệch hạng thẩm quyền", t(func(r *TicketRecord) { r.Evidence[0].Authority++ }), "phiếu.evidence"},
		{"created_by", t(func(r *TicketRecord) { r.CreatedBy = "nguoi-khac" }), "phiếu.created_by"},
		{"pack_version", t(func(r *TicketRecord) { r.PackVersion = "pack-khac" }), "phiếu.pack_version"},
		{"code_version", t(func(r *TicketRecord) { r.CodeVersion = "code-khac" }), "phiếu.code_version"},
		{"command_id", t(func(r *TicketRecord) { r.CommandID = cmdID(999) }), "phiếu.command_id"},
	}
}

// bookkeepingRows là sổ lệnh, số dòng phiếu và audit EXECUTED: ba thứ còn lại mà đọc lại phải thấy. Chữ của chỗ lệch ở sổ lệnh khác nhau
// giữa đọc lại phiếu mới (nêu số dòng, hoặc "lệch trường") và phát lại (một chữ chung).
func bookkeepingRows(msgNoLine, msgTwoLines, msgField string) []readRow {
	ledgerRowsErr := func(f func(*LedgerRecord)) func(*Stored) { return func(s *Stored) { f(&s.Ledger[0]) } }
	return []readRow{
		{"sổ lệnh: thiếu dòng", func(s *Stored) { s.Ledger = nil }, msgNoLine},
		{"sổ lệnh: thừa dòng", func(s *Stored) { s.Ledger = append(s.Ledger, s.Ledger[0]) }, msgTwoLines},
		{"sổ lệnh: lệch người gọi", ledgerRowsErr(func(l *LedgerRecord) { l.Caller = "nguoi-khac" }), msgField},
		{"sổ lệnh: lệch command_id", ledgerRowsErr(func(l *LedgerRecord) { l.CommandID = cmdID(999) }), msgField},
		{"sổ lệnh: lệch vân tay", ledgerRowsErr(func(l *LedgerRecord) { l.Fingerprint = "van-tay-khac" }), msgField},
		{"sổ lệnh: lệch mã phiếu", ledgerRowsErr(func(l *LedgerRecord) { l.TicketID = "ma-phieu-khac" }), msgField},
		{"phiếu: không thấy", func(s *Stored) { s.Tickets = nil }, "phiếu: có 0 dòng"},
		{"phiếu: thấy hai dòng", func(s *Stored) { s.Tickets = append(s.Tickets, s.Tickets[0]) }, "phiếu: có 2 dòng"},
		{"audit EXECUTED: không thấy", func(s *Stored) { s.Executed = nil }, "audit EXECUTED: không thấy"},
		{"audit EXECUTED: của phiếu khác", func(s *Stored) { s.Executed[0].TicketID = "ma-phieu-khac" }, "audit EXECUTED: không thấy"},
		{"audit EXECUTED: kết cục khác", func(s *Stored) { s.Executed[0].Outcome = AuditReplayed }, "audit EXECUTED: không thấy"},
	}
}

func readWorld(t *testing.T) (*world, *readMutator) {
	t.Helper()
	m := &readMutator{}
	w := newWorld(t, func(d *Deps) { m.Store, d.Store = d.Store, m })
	return w, m
}

// Phiếu mới: đọc lại lệch đúng một chỗ thì KHÔNG DONE, UNKNOWN VERIFY_MISMATCH, bước C7 nêu đúng chỗ lệch đó, có audit VERIFY_FAILED
// (§7.2: DONE chỉ khi mọi trường khớp bản chụp ý định). Mỗi dòng bắt một đột biến "bỏ phép so đó".
func TestDocLaiPhieuMoiLechDungMotChoThiVerifyMismatch(t *testing.T) {
	rows := append(ticketFieldRows(), bookkeepingRows("sổ lệnh: có 0 dòng", "sổ lệnh: có 2 dòng", "sổ lệnh: lệch trường")...)
	t.Run("đối chứng: không lệch gì thì DONE", func(t *testing.T) {
		w, _ := readWorld(t)
		wantDone(t, w.run(user("p-1", "director", ""), req(1000, "EX-03", day1)), false)
	})
	for _, row := range rows {
		t.Run(row.name, func(t *testing.T) {
			w, m := readWorld(t)
			m.mut = row.mut
			o := w.run(user("p-1", "director", ""), req(1001, "EX-03", day1))
			wantFail(t, o, StatusUnknown, ReasonVerifyMismatch)
			assertSteps(t, o, "C7")
			if got := o.Steps[stVerify].Output[0]; got != "mismatch="+row.want {
				t.Fatalf("C7 phải nêu đúng một chỗ lệch %q, được %q", "mismatch="+row.want, got)
			}
			w.check(tenantA, want{tickets: 1, ledger: 1, executed: 1, verifyFailed: 1})
		})
	}
}

// replayRows là các điều kiện của chốt phát lại (verifyReplay): phát lại không tính lại nên chỉ so với dòng sổ lệnh đã tra và với chính
// yêu cầu. Trường phiếu nào nằm trong phép so thì dòng của nó phải bắt được.
func replayRows() []readRow {
	t := func(f func(*TicketRecord)) func(*Stored) { return func(s *Stored) { f(&s.Tickets[0]) } }
	const msgTicket = "phiếu: lệch so với yêu cầu đã phát lại"
	rows := []readRow{
		{"phiếu: mã khác dòng sổ lệnh", t(func(r *TicketRecord) { r.ID = "ma-phieu-khac" }), msgTicket},
		{"phiếu: trạng thái", t(func(r *TicketRecord) { r.Status = "CLOSED" }), msgTicket},
		{"phiếu: subject_key", t(func(r *TicketRecord) { r.SubjectKey = "EX-99" }), msgTicket},
		{"phiếu: metric", t(func(r *TicketRecord) { r.Metric = "chi_so_khac" }), msgTicket},
		{"phiếu: day", t(func(r *TicketRecord) { r.Day = "2026-10-01" }), msgTicket},
		{"phiếu: command_id", t(func(r *TicketRecord) { r.CommandID = cmdID(999) }), msgTicket},
		{"phiếu: người tạo", t(func(r *TicketRecord) { r.CreatedBy = "nguoi-khac" }), msgTicket},
	}
	const msgLedger = "sổ lệnh: không còn như lúc tra"
	return append(rows, bookkeepingRows(msgLedger, msgLedger, msgLedger)...)
}

// Phát lại cũng đọc lại (I3): lệch đúng một chỗ thì KHÔNG phát lại DONE, UNKNOWN VERIFY_MISMATCH, có audit VERIFY_FAILED và không có
// REPLAYED (chưa thấy lại đúng phiếu).
func TestDocLaiKhiPhatLaiLechDungMotChoThiVerifyMismatch(t *testing.T) {
	p, r := user("p-1", "director", ""), req(1002, "EX-03", day1)
	t.Run("đối chứng: không lệch gì thì phát lại DONE", func(t *testing.T) {
		w, _ := readWorld(t)
		wantDone(t, w.run(p, r), false)
		wantDone(t, w.run(p, r), true)
	})
	for _, row := range replayRows() {
		t.Run(row.name, func(t *testing.T) {
			w, m := readWorld(t)
			wantDone(t, w.run(p, r), false)
			m.mut = row.mut
			o := w.run(p, r)
			wantFail(t, o, StatusUnknown, ReasonVerifyMismatch)
			if got := o.Steps[stVerify].Output[0]; got != "mismatch="+row.want {
				t.Fatalf("C7 phải nêu đúng một chỗ lệch %q, được %q", "mismatch="+row.want, got)
			}
			w.check(tenantA, want{tickets: 1, ledger: 1, executed: 1, verifyFailed: 1})
		})
	}
}

// twoMetricCatalog là testCatalog cộng một chỉ số dẫn xuất thứ hai (hiệu ngược), để dựng tool có HAI chỉ số.
func twoMetricCatalog(t *testing.T) *sourceoftruth.Catalog {
	t.Helper()
	cat, err := sourceoftruth.Compile(
		[]sourceoftruth.MetricDef{
			{ID: "issued_fuel", Label: "Lượng xuất", Unit: "L", EntityTypes: []string{"equipment"}, Tolerance: "10",
				Sources: []sourceoftruth.SourceRank{{Type: "warehouse_slip", Authority: 1}, {Type: "erp", Authority: 2}, {Type: "excel_summary", Authority: 2}}},
			{ID: "measured_consumption", Label: "Lượng đo", Unit: "L", EntityTypes: []string{"equipment"},
				Sources: []sourceoftruth.SourceRank{{Type: "fuel_sensor", Authority: 1}}},
		},
		[]sourceoftruth.DerivedDef{
			{ID: "issue_vs_measured", Label: "Chênh lệch", Op: "subtract", Left: "issued_fuel", Right: "measured_consumption"},
			{ID: "measured_vs_issue", Label: "Chênh lệch ngược", Op: "subtract", Left: "measured_consumption", Right: "issued_fuel"},
		},
		func(string) bool { return true })
	if err != nil {
		t.Fatal(err)
	}
	return cat
}

// Vân tay gồm cả chỉ số: với tool có hai chỉ số, cùng command_id mà khác chỉ số là KHÁC nội dung. Với một chỉ số duy nhất, bỏ chỉ số khỏi
// vân tay không ai thấy; ca này chỉ có nghĩa khi pack cho hai chỉ số.
func TestCungKhoaKhacChiSoLaKhoaDungLai(t *testing.T) {
	w := newWorld(t, func(d *Deps) {
		d.Catalog = twoMetricCatalog(t)
		d.Tool.Metrics = []string{"issue_vs_measured", "measured_vs_issue"}
	})
	p := user("p-1", "director", "")
	first := Request{CommandID: cmdID(1003), SubjectKey: "EX-03", Metric: "issue_vs_measured", Day: day1}
	second := first
	second.Metric = "measured_vs_issue"
	if tk := wantDone(t, w.run(p, first), false); tk.Diff != "68" {
		t.Fatalf("chênh lệch của chỉ số thứ nhất = %q, muốn 68", tk.Diff)
	}
	o := w.run(p, second) // cùng khoá, cùng máy, cùng ngày, khác chỉ số
	wantFail(t, o, StatusFailed, ReasonIdempotencyKeyReused)
	assertSteps(t, o, "C4b")
	w.check(tenantA, want{tickets: 1, ledger: 1, executed: 1, refused: 1})
	third := second
	third.CommandID = cmdID(1004) // đối chứng: khoá mới thì chỉ số thứ hai tạo được phiếu của nó
	if tk := wantDone(t, w.run(p, third), false); tk.Diff != "-68" || tk.Metric != "measured_vs_issue" {
		t.Fatalf("phiếu của chỉ số thứ hai sai: %+v", tk)
	}
	w.check(tenantA, want{tickets: 2, ledger: 2, executed: 2, refused: 1})
}

// Vân tay phân biệt từng trường của nội dung (máy, chỉ số, ngày) và không gộp ranh giới giữa các trường.
func TestVanTayPhanBietTungTruongCuaNoiDung(t *testing.T) {
	base := Request{SubjectKey: "EX-03", Metric: "issue_vs_measured", Day: day1}
	seen := map[string]string{fingerprint(base): "gốc"}
	for name, r := range map[string]Request{
		"khác máy":       {SubjectKey: "EX-04", Metric: "issue_vs_measured", Day: day1},
		"khác chỉ số":    {SubjectKey: "EX-03", Metric: "measured_vs_issue", Day: day1},
		"khác ngày":      {SubjectKey: "EX-03", Metric: "issue_vs_measured", Day: day2},
		"dồn hai trường": {SubjectKey: "EX-03issue_vs_measured", Metric: "", Day: day1},
	} {
		fp := fingerprint(r)
		if prev, dup := seen[fp]; dup {
			t.Errorf("vân tay của %q trùng %q", name, prev)
		}
		seen[fp] = name
	}
	other := base
	other.CommandID = cmdID(1005)
	if fingerprint(other) != fingerprint(base) {
		t.Errorf("vân tay không gồm command_id (§6.2)")
	}
}

// Ghim nguyên chữ của §2.4: 12 mã lý do, ba trị status, bốn trị retry và nhãn fault của từng mã. Các ca khác so với hằng của gói nên đổi
// chữ của hằng không làm chúng đỏ; ca này so với chữ gốc trong hợp đồng.
func TestGhimChuCuaBangMaLyDoMuc24(t *testing.T) {
	rows := []struct {
		code                string
		status, fault, retr string
	}{
		{"INVALID_REQUEST", "FAILED", "input", "fix_request"},
		{"IDEMPOTENCY_KEY_REUSED", "FAILED", "input", "fix_request"},
		{"PERMISSION_DENIED", "FAILED", "policy", "none"},
		{"INSUFFICIENT_EVIDENCE", "FAILED", "data", "later"},
		{"HUMAN_REVIEW_REQUIRED", "FAILED", "data", "later"},
		{"NO_DISCREPANCY", "FAILED", "data", "none"},
		{"DUPLICATE_OPEN_TICKET", "FAILED", "data", "none"},
		{"STORE_UNAVAILABLE", "FAILED", "infra:postgres", "same_key"},
		{"COMMIT_UNKNOWN", "UNKNOWN", "infra:postgres", "same_key"},
		{"READBACK_UNAVAILABLE", "UNKNOWN", "infra:postgres", "same_key"},
		{"VERIFY_MISMATCH", "UNKNOWN", "code", "none"},
		{"INTERNAL_ERROR", "FAILED", "code", "none"},
	}
	if len(rows) != len(reasons) {
		t.Fatalf("bảng §2.4 có %d mã, mã nguồn có %d", len(rows), len(reasons))
	}
	for _, row := range rows {
		got, ok := reasons[ReasonCode(row.code)]
		if !ok {
			t.Errorf("thiếu mã %s", row.code)
			continue
		}
		if string(got.status) != row.status || got.fault != row.fault || string(got.retry) != row.retr {
			t.Errorf("%s: được status=%q fault=%q retry=%q, muốn %q %q %q", row.code, got.status, got.fault, got.retry, row.status, row.fault, row.retr)
		}
	}
	if string(StatusDone) != "DONE" || string(StatusUnknown) != "UNKNOWN" || string(StatusFailed) != "FAILED" {
		t.Errorf("ba trị status của §2.3 sai chữ: %q %q %q", StatusDone, StatusUnknown, StatusFailed)
	}
	for r, lit := range map[Retry]string{RetryNone: "none", RetrySameKey: "same_key", RetryFixRequest: "fix_request", RetryLater: "later"} {
		if string(r) != lit {
			t.Errorf("retry %q khác chữ %q của §2.3", r, lit)
		}
	}
	if CommandOpenDiscrepancyTicket != "open_discrepancy_ticket" || TicketOpen != "OPEN" || CapMetricsRead != "metrics.read" {
		t.Errorf("tên lệnh, trạng thái phiếu hoặc capability đọc sai chữ")
	}
}

// Mã bằng chứng của Outcome và của bước C5 không dùng chung mảng nền: ghi đè một bên không được làm đổi bên kia (verify dựng mảng mới
// thay vì cắt mảng của C5 về độ dài 0).
func TestMaBangChungCuaOutcomeKhongDungChungMangVoiBuocC5(t *testing.T) {
	w := newWorld(t)
	o := w.run(user("p-1", "director", ""), req(1006, "EX-03", day1))
	wantDone(t, o, false)
	c5, c7 := o.Steps[stRules].Evidence, o.Steps[stVerify].Evidence
	if len(c5) == 0 || len(o.Evidence) == 0 || len(c7) == 0 {
		t.Fatalf("lượt DONE phải mang mã bằng chứng ở Outcome, C5 và C7: %v / %v / %v", o.Evidence, c5, c7)
	}
	if &o.Evidence[0] == &c5[0] || &o.Evidence[0] == &c7[0] || &c5[0] == &c7[0] {
		t.Fatalf("các mảng mã bằng chứng dùng chung mảng nền")
	}
	if !slices.Equal(o.Evidence, c5) || !slices.Equal(o.Evidence, c7) {
		t.Fatalf("nội dung phải bằng nhau: %v / %v / %v", o.Evidence, c5, c7)
	}
}
