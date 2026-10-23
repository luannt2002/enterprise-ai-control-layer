package commands

// Test của CHÍNH bộ chạy chung (bochay_test.go): hình dạng ca, thứ tự repeat, ca đỏ được, F4, F8, từ chối khi băm lệch, thế giới phát triển,
// và việc bộ chạy không rẽ nhánh theo mã ca. Chỉ dùng tệp thế giới và overlay của ca phát triển (testdata/chay), không cần tệp ca.

import (
	"context"
	"errors"
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
	"strings"
	"testing"

	"github.com/luannt2002/enterprise-ai-control-layer/core/contracts"
	"github.com/luannt2002/enterprise-ai-control-layer/evals"
)

func devCfg() chayCfg {
	return chayCfg{
		Cases:   filepath.Join("testdata", "chay", "ca-phat-trien.yaml"),
		World:   filepath.Join("testdata", "chay", "the-gioi-phat-trien.yaml"),
		Pack:    filepath.Join("..", "..", "tenants", "construction", "pack"),
		Overlay: filepath.Join("testdata", "chay"),
		Seed:    42,
	}
}

func loadDevWorld(t *testing.T) *worlds {
	t.Helper()
	wd, err := worldFor(devCfg())
	if err != nil {
		t.Fatal(err)
	}
	return wd
}

// goodCase là một ca hợp lệ tối thiểu (một lời gọi) để các test hình dạng đổi từng khoá.
func goodCase() evals.Case {
	c := evals.Case{}
	for _, k := range evals.Keys {
		c[k] = nil
	}
	c["id"], c["tenant"], c["domain"], c["business_process"], c["scenario"] = "CON-T-900", "abc-infra", "fuel", "p", "A01"
	c["entry_type"], c["category"], c["severity"], c["expected_route"] = "COMMAND", "happy_path", "S1", "TOOL"
	for _, k := range []string{"preconditions", "sources", "expected_tools", "expected_context", "must_not_do", "business_rules", "metrics", "acceptance_criteria"} {
		c[k] = []any{}
	}
	c["input"] = map[string]any{"command_id": "6f1c0d52-0000-4000-8000-000000000900", "subject_key": "EX-03", "metric": "issue_vs_measured", "day": "2026-10-19"}
	c["permissions"] = map[string]any{"principal": "director"}
	c["expected_state"] = map[string]any{"tickets_open": 1}
	return c
}

func TestBoChayCaHopLeKhongCoLoiHinhDang(t *testing.T) {
	pc := parseChayCase(goodCase())
	if len(pc.Errs) != 0 || len(pc.Calls) != 1 {
		t.Fatalf("ca hợp lệ phải không có lỗi hình dạng: %v (số lời gọi %d)", pc.Errs, len(pc.Calls))
	}
	if p := evals.Validate([]evals.Case{goodCase()}, "abc-infra"); len(p) != 0 {
		t.Fatalf("goodCase phải qua evals.Validate: %v", p)
	}
}

// Mỗi ca sai hình dạng là ca ĐỎ (không bị bỏ qua): khoá lạ, sai kiểu, parallel khác số lời gọi...
func TestBoChayCaSaiHinhDangLaDo(t *testing.T) {
	multi := func(items ...any) any { return items }
	item := func(id string, extra map[string]any) map[string]any {
		m := map[string]any{"id": id, "principal": map[string]any{"id": "p-1", "role": "director"},
			"command_id": "6f1c0d52-0000-4000-8000-000000000901", "subject_key": "EX-03", "metric": "issue_vs_measured", "day": "2026-10-19"}
		for k, v := range extra {
			m[k] = v
		}
		return m
	}
	cases := map[string]func(c evals.Case){
		"khoá lạ ở input":                       func(c evals.Case) { c["input"].(map[string]any)["tenant"] = "abc-infra" },
		"khoá lạ ở expected_state":              func(c evals.Case) { c["expected_state"] = map[string]any{"read_calls": 3} },
		"khoá lạ ở expected_output":             func(c evals.Case) { c["expected_output_properties"] = map[string]any{"fault": "policy"} },
		"ngày không có nháy (kiểu khác)":        func(c evals.Case) { c["input"].(map[string]any)["day"] = 20261019 },
		"permissions thiếu khi input là bản đồ": func(c evals.Case) { c["permissions"] = nil },
		"permissions có khi input là danh sách": func(c evals.Case) {
			c["input"] = multi(item("a", nil))
		},
		"điểm chèn lỗi lạ": func(c evals.Case) { c["failure_injection"] = map[string]any{"point": "commit_lost"} },
		"parallel khác số lời gọi": func(c evals.Case) {
			c["permissions"] = nil
			c["input"] = multi(item("a", nil), item("b", nil))
			c["failure_injection"] = map[string]any{"point": "race_barrier", "parallel": 3}
		},
		"reload_before với race_barrier": func(c evals.Case) {
			c["permissions"] = nil
			c["input"] = multi(item("a", map[string]any{"reload_before": true}), item("b", nil))
			c["failure_injection"] = map[string]any{"point": "race_barrier", "parallel": 2}
		},
		"parallel với điểm khác": func(c evals.Case) {
			c["failure_injection"] = map[string]any{"point": "before_begin", "parallel": 2}
		},
		"reply_lost chỉ một lời gọi": func(c evals.Case) { c["failure_injection"] = map[string]any{"point": "reply_lost"} },
		"id mục trùng": func(c evals.Case) {
			c["permissions"] = nil
			c["input"] = multi(item("a", nil), item("a", nil))
		},
		"repeat không phải số": func(c evals.Case) {
			c["permissions"] = nil
			c["input"] = multi(item("a", map[string]any{"repeat": "hai"}))
		},
		"bộ đếm (3) bị cấm": func(c evals.Case) { c["expected_state"] = map[string]any{"read_calls": 2, "tickets_open": 1} },
		"audit thiếu count": func(c evals.Case) {
			c["expected_state"] = map[string]any{"audit": []any{map[string]any{"outcome": "EXECUTED"}}}
		},
		"results là biểu thức": func(c evals.Case) {
			c["expected_output_properties"] = map[string]any{"results": map[string]any{"DONE": "N-1"}}
		},
		"entry_type không phải COMMAND":    func(c evals.Case) { c["entry_type"] = "QUERY" },
		"mã lớp gõ nhầm (soát K4, K4S-09)": func(c evals.Case) { c["scenario"] = "A02B" },
		"mã lớp không có ở §10":            func(c evals.Case) { c["scenario"] = "A99" },
		"người gọi cùng id khác vai": func(c evals.Case) {
			c["permissions"] = nil
			a := item("a", nil)
			b := item("b", nil)
			b["principal"] = map[string]any{"id": "p-1", "role": "accountant"}
			c["input"] = multi(a, b)
		},
	}
	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			c := goodCase()
			mutate(c)
			pc := parseChayCase(c)
			if len(pc.Errs) == 0 {
				t.Fatalf("ca sai hình dạng phải có lỗi (ĐỎ), nhưng không có")
			}
			if _, err := execCase(&pc, &worlds{}, newMemBackend(&worlds{})); err == nil || !strings.Contains(err.Error(), "sai hình dạng") {
				t.Fatalf("execCase phải từ chối ca sai hình dạng, được %v", err)
			}
		})
	}
}

// repeat mở thành các lời gọi liên tiếp tại vị trí của mục, theo đúng thứ tự khai.
func TestBoChayRepeatMoTheoThuTuKhai(t *testing.T) {
	c := goodCase()
	c["permissions"] = nil
	mk := func(id, key string, rep int) map[string]any {
		return map[string]any{"id": id, "principal": map[string]any{"id": "p-1", "role": "director"}, "command_id": "6f1c0d52-0000-4000-8000-000000000902",
			"subject_key": key, "metric": "issue_vs_measured", "day": "2026-10-19", "repeat": rep}
	}
	c["input"] = []any{mk("a", "EX-03", 2), mk("b", "EX-04", 1), mk("c", "EX-05", 3)}
	pc := parseChayCase(c)
	if len(pc.Errs) != 0 {
		t.Fatal(pc.Errs)
	}
	var got []string
	for _, cl := range pc.Calls {
		got = append(got, cl.Label+":"+cl.Req.SubjectKey)
	}
	want := "a#1:EX-03 a#2:EX-03 b:EX-04 c#1:EX-05 c#2:EX-05 c#3:EX-05"
	if strings.Join(got, " ") != want {
		t.Fatalf("thứ tự mở repeat: được %v, muốn %s", got, want)
	}
}

// Ca đỏ được: khẳng định sai thì bộ chạy báo đúng khẳng định đó đỏ, và khẳng định đúng vẫn xanh.
func TestBoChayCaDoKhiKyVongSai(t *testing.T) {
	wd := loadDevWorld(t)
	c := goodCase()
	c["expected_state"] = map[string]any{"tickets_open": 5, "ledger_rows": 1, "write_unit_calls": 1}
	c["expected_output_properties"] = map[string]any{"results": map[string]any{"DONE": 2}, "replayed_false": 1}
	pc := parseChayCase(c)
	checks, err := execCase(&pc, wd, newMemBackend(wd))
	if err != nil {
		t.Fatal(err)
	}
	red := map[string]bool{}
	for _, ck := range checks {
		red[ck.Name] = !ck.OK
	}
	for name, wantRed := range map[string]bool{"tickets_open": true, "ledger_rows": false, "write_unit_calls": false, "results": true, "replayed_false": false} {
		if got, ok := red[name]; !ok || got != wantRed {
			t.Fatalf("khẳng định %s: đỏ=%v (có mặt %v), muốn đỏ=%v", name, got, ok, wantRed)
		}
	}
}

// F4 ở bộ chạy: bên gọi không nhận kết quả của lời gọi đầu; pipeline được dựng mới trước lời gọi thứ hai; phiếu vẫn có.
func TestBoChayF4BoKetQuaVaDungPipelineMoi(t *testing.T) {
	wd := loadDevWorld(t)
	c := goodCase()
	c["permissions"] = nil
	c["input"] = []any{map[string]any{"id": "gui", "principal": map[string]any{"id": "p-1", "role": "director"}, "command_id": "6f1c0d52-0000-4000-8000-000000000903",
		"subject_key": "EX-03", "metric": "issue_vs_measured", "day": "2026-10-19", "repeat": 2}}
	c["failure_injection"] = map[string]any{"point": "reply_lost"}
	pc := parseChayCase(c)
	be := newMemBackend(wd)
	r := &caseRun{pc: &pc, wd: wd, be: be}
	if err := r.execute(); err != nil {
		t.Fatal(err)
	}
	if len(r.res) != 2 || r.res[0].Received || !r.res[1].Received {
		t.Fatalf("lời gọi đầu phải bị bỏ kết quả, lời gọi sau phải nhận: %+v", r.res)
	}
	if r.ctl.restarts != 1 {
		t.Fatalf("phải dựng pipeline mới đúng 1 lần, được %d", r.ctl.restarts)
	}
	if n, _ := be.tickets(pc.Tenant); n != 1 {
		t.Fatalf("lời gọi đầu đã ghi trước khi mất phản hồi: phiếu = %d", n)
	}
	if o := r.res[1].Out.Result; o.Status != StatusDone || !o.Replayed {
		t.Fatalf("lời gọi sau (pipeline mới) phải phát lại: %+v", o)
	}
	if wu, _ := r.rts[pc.Tenant].fs.counters(); wu != 1 {
		t.Fatalf("bộ đếm tích luỹ qua lần dựng lại: write_unit_calls = %d, muốn 1", wu)
	}
}

// F8 ở bộ chạy (đường nối chờ K3): lượt đầu mất trace_id, kết quả đã ghi không đổi; lượt sau còn trace_id.
func TestBoChayF8MatTraceIDKhongDoiKetQua(t *testing.T) {
	wd := loadDevWorld(t)
	c := goodCase()
	c["permissions"] = nil
	c["input"] = []any{map[string]any{"id": "gui", "principal": map[string]any{"id": "p-1", "role": "director"}, "command_id": "6f1c0d52-0000-4000-8000-000000000904",
		"subject_key": "EX-03", "metric": "issue_vs_measured", "day": "2026-10-19", "repeat": 2}}
	c["failure_injection"] = map[string]any{"point": "trace_save_fail"}
	pc := parseChayCase(c)
	be := newMemBackend(wd)
	r := &caseRun{pc: &pc, wd: wd, be: be}
	if err := r.execute(); err != nil {
		t.Fatal(err)
	}
	first, second := r.res[0].Out.Result, r.res[1].Out.Result
	if first.Status != StatusDone || first.TraceID != "" {
		t.Fatalf("F8: kết quả đã ghi không đổi (DONE) nhưng trace_id rỗng: %+v", first)
	}
	if second.Status != StatusDone || second.TraceID == "" || !second.Replayed {
		t.Fatalf("F8 chỉ bắn một lần: lượt sau phát lại bình thường và còn trace_id: %+v", second)
	}
	if r.ctl.traceSaveFailures != 1 {
		t.Fatalf("bộ đếm lỗi lưu vết: %d, muốn 1", r.ctl.traceSaveFailures)
	}
	if n, _ := be.tickets(pc.Tenant); n != 1 {
		t.Fatalf("phiếu = %d, muốn 1", n)
	}
}

// Băm kỳ vọng lệch thì từ chối chạy (§12.1: bộ ca đã đóng băng).
func TestBoChayTuChoiKhiBamLech(t *testing.T) {
	cfg := devCfg()
	cfg.WantCasesSHA = strings.Repeat("0", 64)
	if _, err := buildWorlds(cfg, t.TempDir()); err == nil || !strings.Contains(err.Error(), "từ chối chạy") {
		t.Fatalf("băm tệp ca lệch phải từ chối chạy, được %v", err)
	}
	cfg = devCfg()
	cfg.WantWorldSHA = strings.Repeat("f", 64)
	if _, err := buildWorlds(cfg, t.TempDir()); err == nil || !strings.Contains(err.Error(), "từ chối chạy") {
		t.Fatalf("băm tệp thế giới lệch phải từ chối chạy, được %v", err)
	}
	cfg = devCfg()
	sum, err := fileSHA(cfg.Cases)
	if err != nil {
		t.Fatal(err)
	}
	cfg.WantCasesSHA = strings.ToUpper(sum) // băm đúng (không phân biệt hoa thường) thì không bị từ chối vì băm
	if err := checkWantSHA("tệp ca", sum, cfg.WantCasesSHA); err != nil {
		t.Fatal(err)
	}
}

// Dữ liệu hạt giống đi qua bộ sinh và đường nạp thật cho ra đúng số của ví dụ §1.1 và giữ được cờ ước lượng ở ô lượng của phiếu kho.
func TestBoChayTheGioiPhatTrienDungSoLieu(t *testing.T) {
	wd := loadDevWorld(t)
	for _, tn := range []string{"abc-infra", "def-build"} {
		if wd.tenants[tn] == nil || len(wd.tenants[tn].dir) == 0 {
			t.Fatalf("tenant %s phải có danh mục", tn)
		}
	}
	if wd.now.Format("2006-01-02 15:04") != "2026-10-20 10:00" {
		t.Fatalf("hôm nay của ca phải là reference_date lúc 10:00: %s", wd.now)
	}
	pr := contracts.Principal{ID: "p-1", TenantID: "abc-infra", Kind: contracts.PrincipalUser, Roles: []string{"director"}}
	if !wd.policy.Allows(pr, "tickets.create") || wd.policy.Allows(contracts.Principal{Roles: []string{"site_engineer"}}, "tickets.create") {
		t.Fatal("chính sách phát triển phải theo bảng §4.2: director có, site_engineer không")
	}
}

// Bộ chạy chung KHÔNG rẽ nhánh theo mã ca (§12.1): không phép so sánh (==, !=) nào giữa trường Scenario của ca với một giá trị khác ngoài chuỗi rỗng.
func TestBoChayKhongReNhanhTheoMaCa(t *testing.T) {
	for _, file := range []string{"bochay_test.go", "chenloi_test.go"} {
		fset := token.NewFileSet()
		f, err := parser.ParseFile(fset, file, nil, 0)
		if err != nil {
			t.Fatal(err)
		}
		ast.Inspect(f, func(n ast.Node) bool {
			cmp, ok := n.(*ast.BinaryExpr)
			if !ok || (cmp.Op != token.EQL && cmp.Op != token.NEQ) {
				return true
			}
			for i, side := range []ast.Expr{cmp.X, cmp.Y} {
				sel, ok := side.(*ast.SelectorExpr)
				if !ok || sel.Sel.Name != "Scenario" {
					continue
				}
				other := []ast.Expr{cmp.Y, cmp.X}[i]
				if lit, isLit := other.(*ast.BasicLit); isLit && lit.Kind == token.STRING && lit.Value == `""` {
					continue // kiểm "có điền scenario chưa", không phải rẽ nhánh theo một mã ca
				}
				t.Errorf("%s: so sánh mã ca (Scenario) ở %s", file, fset.Position(cmp.Pos()))
			}
			return true
		})
	}
}

// F2 trên PG (trigger của bộ chạy): bắn ở câu INSERT audit EXECUTED đầu tiên, tức SAU INSERT phiếu và sổ lệnh (sequence chỉ bị tiêu thụ khi tới câu
// INSERT audit), đúng một lần dù giao dịch bị huỷ, và huỷ cả ba dòng của đơn vị bị hỏng.
func TestBoChayF2PGTriggerBanDungChoDungMotLan(t *testing.T) {
	e := newScratchEnv(t, "f2trig")
	tn := e.tenant("f2")
	ctl, err := newFaultCtl(fpInTxAuditInsert, 0, raoWait)
	if err != nil {
		t.Fatal(err)
	}
	be := &pgBackend{e: e, wd: &worlds{}}
	be.arm(ctl)
	defer be.disarm()
	if be.armErr != nil {
		t.Fatal(be.armErr)
	}
	consumed := func() bool { // sequence đã bị nextval chưa
		var called bool
		if err := e.admin.QueryRow(context.Background(), "SELECT is_called FROM k4_f2").Scan(&called); err != nil {
			t.Fatal(err)
		}
		return called
	}
	rows := func(want int) {
		t.Helper()
		if e.openTickets(tn) != want || e.ledgerRows(tn) != want || e.auditRows(tn, AuditExecuted, "", "") != want {
			t.Fatalf("phiếu=%d sổ lệnh=%d EXECUTED=%d, muốn %d mỗi loại", e.openTickets(tn), e.ledgerRows(tn), e.auditRows(tn, AuditExecuted, "", ""), want)
		}
	}
	a := richUnit(1, "p-1", "EX-03")
	if err := e.pg.WriteTicket(ctxOf(tn), a); !errors.Is(err, ErrBeforeCommit) || !consumed() {
		t.Fatalf("INSERT audit đầu tiên phải bị trigger từ chối TRƯỚC COMMIT và tiêu thụ lượt bắn: lỗi %v, tiêu thụ %v", err, consumed())
	}
	rows(0) // cả ba dòng của đơn vị bị hỏng đều mất, kể cả hai dòng đã INSERT
	if err := e.pg.WriteTicket(ctxOf(tn), a); err != nil {
		t.Fatalf("lần gửi lại ghi bình thường (lượt bắn đã dùng): %v", err)
	}
	rows(1)
	be.arm(ctl) // nạp lại lượt bắn (sequence về 1)
	if consumed() {
		t.Fatal("sau khi nạp lại, sequence chưa bị tiêu thụ")
	}
	if err := e.pg.WriteTicket(ctxOf(tn), richUnit(1, "p-1", "EX-04")); !errors.Is(err, ErrUniqueViolation) { // cùng khoá lệnh khác máy
		t.Fatalf("muốn vi phạm khoá chính của sổ lệnh, được %v", err)
	}
	if consumed() {
		t.Fatal("hỏng ở INSERT sổ lệnh thì chưa tới câu INSERT audit: không được tiêu thụ lượt bắn F2")
	}
	if err := e.pg.WriteTicket(ctxOf(tn), richUnit(2, "p-1", "EX-05")); !errors.Is(err, ErrBeforeCommit) || !consumed() {
		t.Fatalf("đơn vị tới được INSERT audit thì bị F2 từ chối: lỗi %v, tiêu thụ %v", err, consumed())
	}
	rows(1)
}

// Soát K4, K4S-04: ca không khẳng định gì (expected_state và expected_output_properties rỗng) là ca ĐỎ — không ĐẠT với 0 khẳng định.
func TestBoChayCaKhongKhangDinhLaDo(t *testing.T) {
	wd := loadDevWorld(t)
	for name, mutate := range map[string]func(c evals.Case){
		"null và null": func(c evals.Case) { c["expected_state"], c["expected_output_properties"] = nil, nil },
		"{} và {}": func(c evals.Case) {
			c["expected_state"], c["expected_output_properties"] = map[string]any{}, map[string]any{}
		},
	} {
		t.Run(name, func(t *testing.T) {
			c := goodCase()
			mutate(c)
			pc := parseChayCase(c)
			_, err := execCase(&pc, wd, newMemBackend(wd))
			var se *shapeError
			if !errors.As(err, &se) || !strings.Contains(err.Error(), "không có khẳng định nào") {
				t.Fatalf("ca không khẳng định gì phải ĐỎ, được %v", err)
			}
		})
	}
}
