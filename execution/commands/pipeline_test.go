package commands

// Ca phát triển (tầng 1) của HOP-DONG §10 mà pipeline với kho GIẢ chứng minh được. Tên test mang mã lớp ca (A01, A02b...).
// Ca đua (A02, A02b, A04b) ép bằng rào F7 của lớp bọc (fake_test.go) trên kho giả có khoá duy nhất: không có rào, lần đến
// sau bị C4b hoặc R5 chặn trước C6 và ca xanh nhờ lịch chạy, kể cả khi chốt chống trùng hỏng (§10, "Vì sao F7").

import (
	"context"
	"errors"
	"go/parser"
	"go/token"
	"path/filepath"
	"reflect"
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/luannt2002/enterprise-ai-control-layer/core/contracts"
)

func TestA01GuiDoiTuanTu(t *testing.T) {
	w := newWorld(t)
	p, r := user("p-1", "director", ""), req(1, "EX-03", day1)

	o1 := w.run(p, r)
	tk := wantDone(t, o1, false)
	assertSteps(t, o1, "")
	if tk.Diff != "68" || tk.Left.Value != "3850" || tk.Right.Value != "3782" || tk.Left.Unit != "L" || tk.Right.Unit != "L" {
		t.Fatalf("giá trị phiếu sai: %+v", tk)
	}
	if tk.Left.Metric != "issued_fuel" || tk.Right.Metric != "measured_consumption" || tk.SubjectKey != "EX-03" || tk.Metric != "issue_vs_measured" || tk.Day != day1 {
		t.Fatalf("định danh phiếu sai: %+v", tk)
	}
	if tk.Approx == nil || *tk.Approx {
		t.Fatalf("hai vế đều không ước lượng: approx phải có mặt và là false, được %v", tk.Approx)
	}

	o2 := w.run(p, r)
	tk2 := wantDone(t, o2, true)
	if tk2.ID != tk.ID || tk2.Diff != "68" {
		t.Fatalf("phát lại phải trả cùng phiếu: %+v và %+v", tk, tk2)
	}
	assertSteps(t, o2, "")
	if o2.Steps[stLedger].Decision != "phát lại" || o2.Steps[stRules].Status != StepSkipped || o2.Steps[stWrite].Status != StepSkipped {
		t.Fatalf("phát lại: C4b ghi 'phát lại', C5 và C6 skipped; được %+v", o2.Steps[stLedger:stVerify])
	}
	// lý do của C5 và C6 ở phát lại thường (bắt ở C4b) đúng điều đã xảy ra: không tính lại, không ghi — không lẫn lý do của phát lại do R5
	// (soát độc lập C1–C7, C17S-01: bản sửa C6 làm C6 ghi nhầm "R5 thấy phiếu …")
	for _, st := range []int{stRules, stWrite} {
		if got := o2.Steps[st].Reason; got != "phát lại ở C4b: không tính lại, không ghi" {
			t.Errorf("phát lại thường: lý do của %s = %q, muốn %q", o2.Steps[st].Code, got, "phát lại ở C4b: không tính lại, không ghi")
		}
	}
	w.check(tenantA, want{tickets: 1, ledger: 1, executed: 1, replayed: 1})
	w.counters(1, 1)
}

func TestA02GuiDoiDongThoiF7(t *testing.T) {
	w := newWorld(t)
	const n = 4
	var prs []contracts.Principal
	var reqs []Request
	for range n {
		prs, reqs = append(prs, user("p-1", "director", "")), append(reqs, req(2, "EX-03", day1))
	}
	outs := w.race(prs, reqs)
	ids, fresh := map[string]bool{}, 0
	for _, o := range outs {
		tk := wantDone(t, o, o.Result.Replayed) // mọi lần DONE: không lần nào nhận lỗi vi phạm khoá hay INTERNAL_ERROR
		ids[tk.ID] = true
		if !o.Result.Replayed {
			fresh++
		}
	}
	if len(ids) != 1 || fresh != 1 {
		t.Fatalf("muốn đúng 1 mã phiếu và đúng 1 lần replayed=false, được %d mã, %d lần mới", len(ids), fresh)
	}
	w.check(tenantA, want{tickets: 1, ledger: 1, executed: 1, replayed: n - 1})
	w.counters(n, n-1)
}

func TestA02bDuaKhacNoiDungF7(t *testing.T) {
	w := newWorld(t)
	prs := []contracts.Principal{user("p-1", "director", ""), user("p-1", "director", ""), user("p-1", "director", "")}
	reqs := []Request{req(3, "EX-03", day1), req(3, "EX-04", day1), req(3, "EX-05", day1)} // cùng khoá, ba máy khác nhau
	outs := w.race(prs, reqs)
	done, reused := 0, 0
	for _, o := range outs {
		switch o.Result.ReasonCode {
		case "":
			wantDone(t, o, false)
			done++
		case ReasonIdempotencyKeyReused:
			wantFail(t, o, StatusFailed, ReasonIdempotencyKeyReused)
			reused++
		default:
			t.Fatalf("kết quả ngoài {DONE, IDEMPOTENCY_KEY_REUSED}: %+v", o.Result)
		}
	}
	if done != 1 || reused != 2 {
		t.Fatalf("muốn DONE=1 IDEMPOTENCY_KEY_REUSED=2, được %d và %d", done, reused)
	}
	w.check(tenantA, want{tickets: 1, ledger: 1, executed: 1, refused: 2})
}

func TestA03CungKhoaKhacNoiDung(t *testing.T) {
	for name, second := range map[string]Request{
		"khác máy":  req(4, "EX-04", day1),
		"khác ngày": req(4, "EX-03", day2),
	} {
		t.Run(name, func(t *testing.T) {
			w := newWorld(t)
			p := user("p-1", "director", "")
			first := wantDone(t, w.run(p, req(4, "EX-03", day1)), false)
			o := w.run(p, second)
			wantFail(t, o, StatusFailed, ReasonIdempotencyKeyReused)
			if o.Result.Fault != "input" || o.Result.Retry != RetryFixRequest || o.Result.Ticket != nil {
				t.Fatalf("IDEMPOTENCY_KEY_REUSED: fault input, retry fix_request, không phiếu: %+v", o.Result)
			}
			assertSteps(t, o, "C4b")
			list := w.store.ticketList(tenantA)
			if len(list) != 1 || list[0].ID != first.ID || list[0].Day != day1 || list[0].SubjectKey != "EX-03" {
				t.Fatalf("phiếu cũ phải nguyên vẹn: %+v", list)
			}
			w.check(tenantA, want{tickets: 1, ledger: 1, executed: 1, refused: 1})
		})
	}
}

func TestA04KhacKhoaCungViecTuanTu(t *testing.T) {
	w := newWorld(t)
	p := user("p-1", "director", "")
	first := wantDone(t, w.run(p, req(5, "EX-03", day1)), false)
	o := w.run(p, req(6, "EX-03", day1))
	wantFail(t, o, StatusFailed, ReasonDuplicateOpenTicket)
	if o.Result.Fault != "data" || o.Result.Retry != RetryNone {
		t.Fatalf("DUPLICATE_OPEN_TICKET: fault data, retry none: %+v", o.Result)
	}
	tk := o.Result.Ticket
	if tk == nil || tk.ID != first.ID || tk.Status != TicketOpen || tk.Left != nil || tk.Right != nil || tk.Diff != "" || tk.Approx != nil {
		t.Fatalf("phải trả mã và trạng thái phiếu cũ, không gì khác: %+v", tk)
	}
	assertSteps(t, o, "C5") // từ chối đến từ R5 ở C5, không phải từ chỉ mục duy nhất ở C6
	w.check(tenantA, want{tickets: 1, ledger: 1, executed: 1, refused: 1})
	w.counters(1, 1) // đơn vị ghi phiếu không được gọi cho lần thứ hai; audit REFUSED vẫn ghi bằng giao dịch riêng
	if n := len(w.store.audits(tenantA, AuditRefused, ReasonDuplicateOpenTicket, "p-1")); n != 1 {
		t.Fatalf("audit REFUSED DUPLICATE_OPEN_TICKET = %d, muốn 1", n)
	}
}

func TestA04bDuaKhacKhoaF7(t *testing.T) {
	w := newWorld(t)
	const n = 4
	var prs []contracts.Principal
	var reqs []Request
	for i := range n {
		prs, reqs = append(prs, user("p-1", "director", "")), append(reqs, req(10+i, "EX-03", day1)) // khác khoá, cùng (máy, chỉ số, ngày)
	}
	outs := w.race(prs, reqs)
	var winner string
	done, dup := 0, 0
	for _, o := range outs {
		if o.Result.Status == StatusDone {
			winner = wantDone(t, o, false).ID
			done++
		}
	}
	for _, o := range outs {
		if o.Result.Status == StatusDone {
			continue
		}
		wantFail(t, o, StatusFailed, ReasonDuplicateOpenTicket)
		if o.Result.Ticket == nil || o.Result.Ticket.ID != winner {
			t.Fatalf("bên thua phải nhận mã phiếu của lần thắng %s: %+v", winner, o.Result.Ticket)
		}
		dup++
	}
	if done != 1 || dup != n-1 {
		t.Fatalf("muốn DONE=1 DUPLICATE_OPEN_TICKET=%d, được %d và %d", n-1, done, dup)
	}
	w.check(tenantA, want{tickets: 1, ledger: 1, executed: 1, refused: n - 1})
}

func TestA05NguoiGoiKhacCungKhoa(t *testing.T) {
	w := newWorld(t)
	first := wantDone(t, w.run(user("p-1", "director", ""), req(7, "EX-03", day1)), false)
	o := w.run(user("p-2", "accountant", ""), req(7, "EX-03", day1)) // cùng khoá, cùng nội dung, người khác
	wantFail(t, o, StatusFailed, ReasonDuplicateOpenTicket)
	if o.Result.Replayed || o.Result.Ticket == nil || o.Result.Ticket.ID != first.ID {
		t.Fatalf("không được phát lại kết quả của người kia: %+v", o.Result)
	}
	if n := len(w.store.audits(tenantA, AuditExecuted, "", "p-1")); n != 1 {
		t.Fatalf("EXECUTED của p-1 = %d, muốn 1", n)
	}
	if n := len(w.store.audits(tenantA, AuditRefused, ReasonDuplicateOpenTicket, "p-2")); n != 1 {
		t.Fatalf("REFUSED của p-2 = %d, muốn 1 (audit ghi người gọi thứ hai)", n)
	}
	w.check(tenantA, want{tickets: 1, ledger: 1, executed: 1, refused: 1})
	w.counters(1, 1)
}

func TestA06HaiTenantCungKhoa(t *testing.T) {
	w := newWorld(t)
	pb := w.other(tenantB)
	pa := user("p-1", "director", "")
	pbUser := contracts.Principal{ID: "p-1", TenantID: tenantB, Kind: contracts.PrincipalUser, Roles: []string{"director"}}
	r := req(8, "EX-03", day1)
	ta := wantDone(t, w.run(pa, r), false)
	tb := wantDone(t, pb.Run(context.Background(), pbUser, r), false)
	if ta.ID == tb.ID {
		t.Fatalf("hai tenant phải có hai phiếu khác nhau")
	}
	w.check(tenantA, want{tickets: 1, ledger: 1, executed: 1})
	w.check(tenantB, want{tickets: 1, ledger: 1, executed: 1})
	// đọc chéo bằng ngữ cảnh của tenant này không thấy dòng nào của tenant kia
	ctxB := contracts.WithTenant(context.Background(), tenantB)
	st, err := w.store.ReadBack(ctxB, "p-1", r.CommandID, ta.ID)
	if err != nil || len(st.Tickets) != 0 {
		t.Fatalf("tenant B không được thấy phiếu của tenant A: %+v, %v", st.Tickets, err)
	}
}

func TestA07ThieuTicketsCreate(t *testing.T) {
	w := newWorld(t)
	for i, role := range []string{"site_engineer", "driver"} { // site_engineer có metrics.read, driver không có gì
		o := w.run(user("p-"+role, role, "PRJ-A"), req(20+i, "EX-03", day1))
		wantFail(t, o, StatusFailed, ReasonPermissionDenied)
		if o.Result.Fault != "policy" || o.Result.Retry != RetryNone {
			t.Fatalf("PERMISSION_DENIED: fault policy, retry none: %+v", o.Result)
		}
		assertSteps(t, o, "C4")
	}
	w.check(tenantA, want{refused: 2})
	w.counters(0, 2)
	if n := len(w.store.audits(tenantA, AuditRefused, ReasonPermissionDenied, "")); n != 2 {
		t.Fatalf("audit REFUSED PERMISSION_DENIED = %d, muốn 2", n)
	}
}

// scrub thay khoá người gọi đã gõ trong vết bằng một dấu, để so hai vết chỉ khác nhau ở khoá đó (A29 (c)).
func scrub(o Outcome, typed string) []Step {
	fix := func(in []string) []string {
		var out []string
		for _, s := range in {
			out = append(out, strings.ReplaceAll(s, typed, "<khoá>"))
		}
		return out
	}
	var steps []Step
	for _, s := range o.Steps {
		s.Input, s.Output = fix(s.Input), fix(s.Output)
		s.Decision, s.Reason = strings.ReplaceAll(s.Decision, typed, "<khoá>"), strings.ReplaceAll(s.Reason, typed, "<khoá>")
		steps = append(steps, s)
	}
	return steps
}

func TestA08KhongTonTaiNgoaiPhamViKyTuLa(t *testing.T) {
	w := newWorld(t)
	pm := user("p-pm", "project_manager", "PRJ-A")
	keys := []string{"EX-09", "EX-99", "EX-03'; DROP TABLE discrepancy_tickets; --", `EX-03"`} // ngoài phạm vi, không tồn tại, ký tự SQL, dấu nháy
	var outs []Outcome
	for _, k := range keys {
		o := w.run(pm, req(30, k, day1)) // cùng command_id: lệnh bị từ chối không tiêu thụ khoá (§2.3)
		wantFail(t, o, StatusFailed, ReasonPermissionDenied)
		assertSteps(t, o, "C4")
		outs = append(outs, o)
	}
	details := map[string]bool{}
	for _, o := range outs {
		details[o.Result.Detail] = true
	}
	if len(details) != 1 {
		t.Fatalf("bốn lý do khác nhau phải cùng MỘT thông báo, được %d: %v", len(details), details)
	}
	w.check(tenantA, want{refused: 4})
	w.counters(0, 4)
	// lý do thật chỉ nằm ở audit
	out09 := w.store.audits(tenantA, AuditRefused, ReasonPermissionDenied, "p-pm")
	var d09, d99 string
	for _, a := range out09 {
		switch a.SubjectKey {
		case "EX-09":
			d09 = a.ReasonDetail
		case "EX-99":
			d99 = a.ReasonDetail
		}
	}
	if d09 == "" || d99 == "" || d09 == d99 {
		t.Fatalf("audit phải ghi lý do thật và tách hai trường hợp: %q và %q", d09, d99)
	}
	// vết (A29 (c)): hai lượt chỉ khác nhau ở khoá đã gõ; không chứa lý do thật
	if !reflect.DeepEqual(scrub(outs[0], "EX-09"), scrub(outs[1], "EX-99")) {
		t.Fatalf("vết của máy ngoài phạm vi và máy không tồn tại phải y hệt:\n%+v\n%+v", scrub(outs[0], "EX-09"), scrub(outs[1], "EX-99"))
	}
	for _, o := range outs[:2] {
		for _, s := range o.Steps {
			blob := strings.ToLower(strings.Join(append(append([]string{s.Decision, s.Reason}, s.Input...), s.Output...), " | "))
			for _, leak := range []string{"không tồn tại", "ngoài phạm vi", "danh mục"} {
				if strings.Contains(blob, leak) {
					t.Fatalf("bước %s lộ lý do thật %q: %s", s.Code, leak, blob)
				}
			}
		}
	}
}

// A09 chỉ là ca phát triển: pack hợp lệ không cho cấu hình này (§4.3), nên dựng Policy bằng tay; lớp kiểm lúc chạy phải đóng.
func TestA09ThieuMetricsRead(t *testing.T) {
	w := newWorld(t, func(d *Deps) {
		d.Policy.Roles["writer"] = routerRole("tickets.create")
	})
	o := w.run(user("p-w", "writer", ""), req(40, "EX-03", day1))
	wantFail(t, o, StatusFailed, ReasonPermissionDenied)
	assertSteps(t, o, "C4")
	w.check(tenantA, want{refused: 1})
	w.counters(0, 1)
}

func TestA10TenantKhacVoiRuntime(t *testing.T) {
	w := newWorld(t)
	other := contracts.Principal{ID: "p-x", TenantID: tenantB, Kind: contracts.PrincipalUser, Roles: []string{"director"}}
	noID := contracts.Principal{TenantID: tenantA, Kind: contracts.PrincipalUser, Roles: []string{"director"}}
	for name, pr := range map[string]contracts.Principal{"tenant khác": other, "không có mã người gọi": noID} {
		o := w.run(pr, req(41, "EX-03", day1))
		wantFail(t, o, StatusFailed, ReasonPermissionDenied)
		if o.Result.Fault != "policy" {
			t.Fatalf("%s: fault phải là policy, được %q", name, o.Result.Fault)
		}
		assertSteps(t, o, "C1")
	}
	w.check(tenantA, want{})
	w.check(tenantB, want{})
	w.counters(0, 0) // không có dòng audit (§7.3): chưa có tenant tin cậy để gắn
}

// A13: mỗi biến thể một trường sai; nêu đúng trường (theo thứ tự command_id, subject_key, metric, day), 0 ghi, không audit.
func TestA13YeuCauSai(t *testing.T) {
	good := req(50, "EX-03", day1)
	with := func(f func(*Request)) Request { r := good; f(&r); return r }
	cases := []struct {
		name   string
		req    Request
		fields []string
		stop   string
	}{
		{"thiếu command_id", with(func(r *Request) { r.CommandID = "" }), []string{"command_id"}, "C0"},
		{"command_id không phải UUID", with(func(r *Request) { r.CommandID = "khong-phai-uuid" }), []string{"command_id"}, "C0"},
		{"command_id thiếu một ký tự", with(func(r *Request) { r.CommandID = good.CommandID[1:] }), []string{"command_id"}, "C0"},
		{"command_id dạng không dấu gạch", with(func(r *Request) { r.CommandID = strings.ReplaceAll(good.CommandID, "-", "") }), []string{"command_id"}, "C0"},
		{"command_id chứa NUL", with(func(r *Request) { r.CommandID = good.CommandID[:35] + "\x00" }), []string{"command_id"}, "C0"},
		{"thiếu subject_key", with(func(r *Request) { r.SubjectKey = "" }), []string{"subject_key"}, "C0"},
		{"subject_key chứa NUL", with(func(r *Request) { r.SubjectKey = "EX-03\x00" }), []string{"subject_key"}, "C0"},
		{"thiếu metric", with(func(r *Request) { r.Metric = "" }), []string{"metric"}, "C0"},
		{"metric chứa NUL", with(func(r *Request) { r.Metric = "issue_vs_measured\x00" }), []string{"metric"}, "C0"},
		{"thiếu day", with(func(r *Request) { r.Day = "" }), []string{"day"}, "C0"},
		{"day chứa NUL", with(func(r *Request) { r.Day = day1 + "\x00" }), []string{"day"}, "C0"},
		{"day sai dạng", with(func(r *Request) { r.Day = "19/10/2026" }), []string{"day"}, "C3"},
		{"day không có trên lịch", with(func(r *Request) { r.Day = "2026-02-30" }), []string{"day"}, "C3"},
		{"day không đủ hai chữ số", with(func(r *Request) { r.Day = "2026-10-9" }), []string{"day"}, "C3"},
		{"day là hôm nay", with(func(r *Request) { r.Day = "2026-10-20" }), []string{"day"}, "C3"},
		{"day là tương lai", with(func(r *Request) { r.Day = "2026-10-21" }), []string{"day"}, "C3"},
		{"metric ngoài danh sách", with(func(r *Request) { r.Metric = "reported_vs_issued" }), []string{"metric"}, "C3"},
		{"metric là chỉ số gốc", with(func(r *Request) { r.Metric = "issued_fuel" }), []string{"metric"}, "C3"},
		{"cả bốn trường thiếu", Request{}, []string{"command_id", "subject_key", "metric", "day"}, "C0"},
		{"hai trường sai ở cùng chặng", with(func(r *Request) { r.Metric, r.Day = "reported_vs_issued", "2026-10-21" }), []string{"metric", "day"}, "C3"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			w := newWorld(t)
			o := w.run(user("p-1", "director", ""), c.req)
			wantFail(t, o, StatusFailed, ReasonInvalidRequest)
			if !slices.Equal(o.Result.InvalidFields, c.fields) || o.Result.Fault != "input" || o.Result.Retry != RetryFixRequest {
				t.Fatalf("invalid_fields=%v fault=%q retry=%q, muốn %v / input / fix_request", o.Result.InvalidFields, o.Result.Fault, o.Result.Retry, c.fields)
			}
			assertSteps(t, o, c.stop)
			for _, s := range o.Steps { // vết không mang NUL: kho vết không lưu được
				if strings.Contains(strings.Join(append(append([]string{s.Decision, s.Reason}, s.Input...), s.Output...), "|"), "\x00") {
					t.Fatalf("bước %s mang ký tự NUL", s.Code)
				}
			}
			if strings.Contains(o.Question, "\x00") {
				t.Fatalf("Question mang ký tự NUL")
			}
			w.check(tenantA, want{})
			w.counters(0, 0)
		})
	}

	t.Run("khoá thực thể là dự án, không phải máy", func(t *testing.T) { // loại thực thể kiểm SAU quyền: có audit REFUSED
		w := newWorld(t)
		o := w.run(user("p-1", "director", ""), req(51, "PRJ-A", day1))
		wantFail(t, o, StatusFailed, ReasonInvalidRequest)
		if !slices.Equal(o.Result.InvalidFields, []string{"subject_key"}) {
			t.Fatalf("invalid_fields = %v", o.Result.InvalidFields)
		}
		assertSteps(t, o, "C5")
		w.check(tenantA, want{refused: 1})
		w.counters(0, 1)
	})
}

func TestA14KhongCoChenhLech(t *testing.T) {
	for name, c := range map[string]struct {
		min, key string
		code     ReasonCode // rỗng = DONE
		diff     string
	}{
		"hai vế bằng nhau":                     {"0", "EX-01", ReasonNoDiscrepancy, ""},
		"chênh lệch 68 không vượt 100":         {"100", "EX-03", ReasonNoDiscrepancy, ""},
		"chênh lệch đúng bằng ngưỡng":          {"100", "EX-05", ReasonNoDiscrepancy, ""},
		"chênh lệch 68 vượt ngưỡng 67":         {"67", "EX-03", "", "68"},
		"chênh lệch âm vẫn là chênh lệch":      {"0", "EX-10", "", "-50"},
		"chênh lệch âm: trị tuyệt đối 50":      {"50", "EX-10", ReasonNoDiscrepancy, ""},
		"ngưỡng thập phân so chính xác":        {"67.9", "EX-03", "", "68"},
		"ngưỡng thập phân sát trên chênh lệch": {"68.0", "EX-03", ReasonNoDiscrepancy, ""},
	} {
		t.Run(name, func(t *testing.T) {
			w := newWorld(t, func(d *Deps) { d.Tool.MinAbsDiff = c.min })
			o := w.run(user("p-1", "director", ""), req(60, c.key, day1))
			if c.code == "" {
				if tk := wantDone(t, o, false); tk.Diff != c.diff {
					t.Fatalf("diff = %q, muốn %q", tk.Diff, c.diff)
				}
				return
			}
			wantFail(t, o, StatusFailed, c.code)
			if o.Result.Fault != "data" || o.Result.Retry != RetryNone {
				t.Fatalf("NO_DISCREPANCY: fault data, retry none: %+v", o.Result)
			}
			assertSteps(t, o, "C5")
			w.check(tenantA, want{refused: 1})
		})
	}
}

func TestA15ThieuMotVe(t *testing.T) {
	for _, key := range []string{"EX-02", "EX-11"} { // thiếu vế đo; không có số liệu nào
		w := newWorld(t)
		o := w.run(user("p-1", "director", ""), req(61, key, day1))
		wantFail(t, o, StatusFailed, ReasonInsufficientEvidence)
		if o.Result.Fault != "data" || o.Result.Retry != RetryLater || o.Result.Ticket != nil {
			t.Fatalf("%s: fault data, retry later, không phiếu: %+v", key, o.Result)
		}
		assertSteps(t, o, "C5")
		w.check(tenantA, want{refused: 1})
	}
}

func TestA16HaiNguonNgangThamQuyenLech(t *testing.T) {
	for _, key := range []string{"EX-06", "EX-07"} { // EX-07: vế kia còn thiếu, nhưng xung đột phải thắng thiếu vế
		w := newWorld(t)
		o := w.run(user("p-1", "director", ""), req(62, key, day1))
		wantFail(t, o, StatusFailed, ReasonHumanReviewRequired)
		if o.Result.Fault != "data" || o.Result.Retry != RetryLater || o.Result.Ticket != nil {
			t.Fatalf("%s: fault data, retry later, không phiếu và không chọn số nào: %+v", key, o.Result)
		}
		assertSteps(t, o, "C5")
		if len(o.Steps[stRules].Rejected) == 0 {
			t.Fatalf("%s: C5 phải ghi nguồn và số bị loại ở 'rejected'", key)
		}
		w.check(tenantA, want{refused: 1})
	}
}

func TestA17SoUocLuongGiuCoApprox(t *testing.T) {
	w := newWorld(t)
	o := w.run(user("p-1", "director", ""), req(63, "EX-08", day1))
	tk := wantDone(t, o, false)
	if tk.Approx == nil || !*tk.Approx || tk.Diff != "20" {
		t.Fatalf("phiếu phải mang approx=true: %+v", tk)
	}
	if list := w.store.ticketList(tenantA); len(list) != 1 || !list[0].Approx {
		t.Fatalf("phiếu đã lưu phải mang approx=true: %+v", list)
	}
}

// A18: Load thay toàn bộ danh mục và số liệu bằng lần nạp mới (mã thực thể và mã fact đổi hết). Phiếu vẫn nguyên với bản chụp
// bằng chứng của nó; gửi lại cùng khoá vẫn phát lại mà không tính lại; việc trùng nhận ra theo khoá chứ không theo mã thực thể.
func TestA18LoadThayDuLieuVanPhatLai(t *testing.T) {
	w := newWorld(t)
	p, r := user("p-1", "director", ""), req(64, "EX-03", day1)
	first := wantDone(t, w.run(p, r), false)
	before := w.store.ticketList(tenantA)
	if len(before) != 1 || len(before[0].Evidence) < 3 {
		t.Fatalf("phiếu phải mang bản chụp bằng chứng (hai vế và phép tính): %+v", before)
	}

	w.data.reload(2, true)
	again := wantDone(t, w.run(p, r), true)
	if again.ID != first.ID || again.Diff != "68" {
		t.Fatalf("sau Load phát lại phải trả đúng phiếu cũ: %+v", again)
	}
	after := w.store.ticketList(tenantA)
	if len(after) != 1 || !reflect.DeepEqual(after[0], before[0]) {
		t.Fatalf("phiếu phải nguyên vẹn sau Load")
	}
	if !slices.ContainsFunc(after[0].Evidence, func(e EvidenceSnap) bool { return strings.HasPrefix(e.ID, "fact:f1-") }) {
		t.Fatalf("bản chụp bằng chứng phải giữ mã fact của lần nạp cũ: %+v", after[0].Evidence)
	}
	dup := w.run(p, req(65, "EX-03", day1)) // khoá mới, cùng việc, mã thực thể đã đổi
	wantFail(t, dup, StatusFailed, ReasonDuplicateOpenTicket)

	w.data.reload(3, false) // Load không còn số liệu nào: phát lại không đụng tới số liệu nên vẫn DONE
	wantDone(t, w.run(p, r), true)
	w.check(tenantA, want{tickets: 1, ledger: 1, executed: 1, replayed: 2, refused: 1})
}

func TestA19F1TruocKhiMoGiaoDich(t *testing.T) {
	w := newWorld(t)
	w.probe.f1 = 1
	p, r := user("p-1", "director", ""), req(70, "EX-03", day1)
	o := w.run(p, r)
	wantFail(t, o, StatusFailed, ReasonStoreUnavailable)
	if o.Result.Fault != "infra:postgres" || o.Result.Retry != RetrySameKey {
		t.Fatalf("STORE_UNAVAILABLE: fault infra:postgres, retry same_key: %+v", o.Result)
	}
	assertSteps(t, o, "C6")
	w.check(tenantA, want{}) // 0 ghi và không có dòng audit riêng cho kết cục hạ tầng (§7.3)
	w.counters(1, 0)
	wantDone(t, w.run(p, r), false) // gửi lại thì thực thi bình thường
	w.check(tenantA, want{tickets: 1, ledger: 1, executed: 1})
}

func TestA20F2LoiInsertAuditHuyCaBaDong(t *testing.T) {
	w := newWorld(t)
	w.store.failAuditInTx = 1
	p, r := user("p-1", "director", ""), req(71, "EX-03", day1)
	o := w.run(p, r)
	wantFail(t, o, StatusFailed, ReasonStoreUnavailable)
	assertSteps(t, o, "C6")
	w.check(tenantA, want{}) // 0 phiếu, 0 sổ lệnh, 0 EXECUTED: rollback đủ ba dòng
	wantDone(t, w.run(p, r), false)
	w.check(tenantA, want{tickets: 1, ledger: 1, executed: 1})
}

// A21: timeout sau khi ghi. (a) F3: commit thật rồi lớp lưu trữ báo lỗi; (b) F4: pipeline xong mà bên gọi không nhận được gì.
func TestA21TimeoutSauKhiGhi(t *testing.T) {
	t.Run("F3 mất ack", func(t *testing.T) {
		w := newWorld(t)
		w.probe.f3 = 1
		p, r := user("p-1", "director", ""), req(72, "EX-03", day1)
		o := w.run(p, r)
		wantFail(t, o, StatusUnknown, ReasonCommitUnknown)
		if o.Result.Fault != "infra:postgres" || o.Result.Retry != RetrySameKey || o.Result.Ticket != nil {
			t.Fatalf("COMMIT_UNKNOWN: fault infra:postgres, retry same_key, không phiếu: %+v", o.Result)
		}
		assertSteps(t, o, "C6")
		if c7 := o.Steps[stVerify]; c7.Status != StepOK || !strings.Contains(c7.Reason, "không đọc lại") {
			t.Fatalf("C7 vẫn ok và ghi lý do không đọc lại: %+v", c7)
		}
		w.check(tenantA, want{tickets: 1, ledger: 1, executed: 1})
		stored := w.onlyTicket()
		again := wantDone(t, w.run(p, r), true)
		if again.ID != stored.ID {
			t.Fatalf("gửi lại phải phát lại đúng phiếu đã commit: %s và %s", again.ID, stored.ID)
		}
		w.check(tenantA, want{tickets: 1, ledger: 1, executed: 1, replayed: 1})
	})
	t.Run("F4 mất phản hồi, dựng pipeline mới", func(t *testing.T) {
		w := newWorld(t)
		p, r := user("p-1", "director", ""), req(73, "EX-03", day1)
		_ = w.run(p, r) // bên gọi không nhận gì
		w.restart()     // giả lập khởi động lại: không còn bộ nhớ nào của tiến trình
		first := w.onlyTicket()
		again := wantDone(t, w.run(p, r), true)
		if again.ID != first.ID {
			t.Fatalf("sau khởi động lại vẫn phải phát lại đúng phiếu: %s và %s", again.ID, first.ID)
		}
		w.check(tenantA, want{tickets: 1, ledger: 1, executed: 1, replayed: 1})
	})
}

// A22: COMMIT đến muộn, sau khi pipeline đã báo UNKNOWN. Dù commit đến trước hay sau lần gửi lại, tổng cộng đúng một phiếu.
func TestA22CommitDenMuon(t *testing.T) {
	t.Run("commit đến trước lần gửi lại", func(t *testing.T) {
		w := newWorld(t)
		w.probe.f5 = 1
		p, r := user("p-1", "director", ""), req(74, "EX-03", day1)
		wantFail(t, w.run(p, r), StatusUnknown, ReasonCommitUnknown)
		w.check(tenantA, want{})
		w.probe.releaseDelayed()
		w.check(tenantA, want{tickets: 1, ledger: 1, executed: 1})
		wantDone(t, w.run(p, r), true)
		w.check(tenantA, want{tickets: 1, ledger: 1, executed: 1, replayed: 1})
	})
	t.Run("commit đến sau lần gửi lại", func(t *testing.T) {
		w := newWorld(t)
		w.probe.f5 = 1
		p, r := user("p-1", "director", ""), req(75, "EX-03", day1)
		wantFail(t, w.run(p, r), StatusUnknown, ReasonCommitUnknown)
		wantDone(t, w.run(p, r), false) // chưa thấy gì trong sổ lệnh nên ghi mới
		w.probe.releaseDelayed()        // commit muộn đụng khoá duy nhất và bị chặn
		w.check(tenantA, want{tickets: 1, ledger: 1, executed: 1})
	})
}

// A23: đọc lại hỏng thì không bao giờ DONE; lệch trường hoặc không thấy thì có audit VERIFY_FAILED.
func TestA23DocLaiHong(t *testing.T) {
	p := user("p-1", "director", "")
	t.Run("F6 lỗi khi đọc lại", func(t *testing.T) {
		w := newWorld(t)
		w.probe.f6fail = 1
		r := req(76, "EX-03", day1)
		o := w.run(p, r)
		wantFail(t, o, StatusUnknown, ReasonReadbackUnavailable)
		if o.Result.Fault != "infra:postgres" || o.Result.Retry != RetrySameKey {
			t.Fatalf("READBACK_UNAVAILABLE: infra:postgres, same_key: %+v", o.Result)
		}
		assertSteps(t, o, "C7")
		w.check(tenantA, want{tickets: 1, ledger: 1, executed: 1}) // đã commit; không có audit VERIFY_FAILED vì chưa thấy gì sai
		wantDone(t, w.run(p, r), true)
	})
	for _, mode := range []string{"missing", "diff"} {
		t.Run("F6 đọc lại lệch: "+mode, func(t *testing.T) {
			w := newWorld(t)
			w.probe.f6stale, w.probe.staleMode = 1, mode
			o := w.run(p, req(77, "EX-03", day1))
			wantFail(t, o, StatusUnknown, ReasonVerifyMismatch)
			if o.Result.Fault != "code" || o.Result.Retry != RetryNone || o.Result.Ticket != nil {
				t.Fatalf("VERIFY_MISMATCH: fault code, retry none, không phiếu: %+v", o.Result)
			}
			assertSteps(t, o, "C7")
			w.check(tenantA, want{tickets: 1, ledger: 1, executed: 1, verifyFailed: 1})
		})
	}
	// C2 (lệnh ca chiều 23/10): dòng audit EXECUTED đọc lại phải khớp MỌI cột đọc được với bản chụp ý định, không chỉ kết cục + mã phiếu.
	for _, mode := range []string{"audit-fp", "audit-trace", "audit-caller", "audit-roles", "audit-day", "audit-action", "audit-reason",
		"audit-scope", "audit-pack", "audit-code", "audit-cmd", "audit-subject", "audit-metric"} { // đủ 15 cột so được (C17S-02)
		t.Run("F6 audit EXECUTED lệch: "+mode, func(t *testing.T) {
			w := newWorld(t)
			w.probe.f6stale, w.probe.staleMode = 1, mode
			o := w.run(p, req(79, "EX-03", day1))
			wantFail(t, o, StatusUnknown, ReasonVerifyMismatch)
			assertSteps(t, o, "C7")
			w.check(tenantA, want{tickets: 1, ledger: 1, executed: 1, verifyFailed: 1})
		})
	}
	// phát lại: không có bản chụp ý định của lượt gốc, nên so với dòng sổ lệnh và chính yêu cầu — các cột suy ra được từ đó
	// pack / code: phiếu và dòng EXECUTED cùng một đơn vị ghi nên phiên bản phải bằng nhau — phát lại so với phiếu đọc lại (C17S-07)
	for _, mode := range []string{"audit-fp", "audit-caller", "audit-day", "audit-action", "audit-reason", "audit-cmd", "audit-subject", "audit-metric",
		"audit-pack", "audit-code"} {
		t.Run("phát lại, audit EXECUTED lệch: "+mode, func(t *testing.T) {
			w := newWorld(t)
			r := req(80, "EX-03", day1)
			wantDone(t, w.run(p, r), false)
			w.probe.f6stale, w.probe.staleMode = 1, mode
			wantFail(t, w.run(p, r), StatusUnknown, ReasonVerifyMismatch)
			w.check(tenantA, want{tickets: 1, ledger: 1, executed: 1, verifyFailed: 1})
		})
	}
	for _, mode := range []string{"missing", "subject"} { // phát lại cũng đọc lại: không thấy phiếu, hoặc thấy phiếu của khoá khác
		t.Run("phát lại cũng đọc lại: "+mode, func(t *testing.T) {
			w := newWorld(t)
			r := req(78, "EX-03", day1)
			wantDone(t, w.run(p, r), false)
			w.probe.f6stale, w.probe.staleMode = 1, mode
			o := w.run(p, r)
			wantFail(t, o, StatusUnknown, ReasonVerifyMismatch)
			w.check(tenantA, want{tickets: 1, ledger: 1, executed: 1, verifyFailed: 1}) // không có REPLAYED: chưa thấy lại đúng phiếu
		})
	}
}

// A24 (phần mã) và ràng buộc kiến trúc của ADR-0011: không model, không core/tenant, không core/pipeline; chỉ trace.go được
// nối bảng vết (observability/traces) vì chỉ nó dựng bản ghi vết.
func TestA24VaRangBuocImport(t *testing.T) {
	files, err := filepath.Glob("*.go")
	if err != nil || len(files) == 0 {
		t.Fatalf("không đọc được mã của gói: %v", err)
	}
	for _, f := range files {
		if strings.HasSuffix(f, "_test.go") {
			continue
		}
		af, err := parser.ParseFile(token.NewFileSet(), f, nil, parser.ImportsOnly)
		if err != nil {
			t.Fatal(err)
		}
		for _, im := range af.Imports {
			path, _ := strconv.Unquote(im.Path.Value)
			for _, bad := range []string{"/intelligence/models", "/core/tenant", "/core/pipeline"} {
				if strings.HasSuffix(path, bad) {
					t.Errorf("%s import %s: cấm (0 lời gọi model; gói không phụ thuộc core/tenant, core/pipeline)", f, path)
				}
			}
			if strings.HasSuffix(path, "/observability/traces") && f != "trace.go" {
				t.Errorf("%s import %s: chỉ trace.go được nối bảng vết", f, path)
			}
		}
	}
}

// ---------------------------------------------------------------------------
// Ca thêm của người viết mã: những chỗ hợp đồng để cách an toàn nhất, và đường lỗi
// ---------------------------------------------------------------------------

func TestKhoaLenhVietHoaVaThuongLaCungMotKhoa(t *testing.T) {
	w := newWorld(t)
	p := user("p-1", "director", "")
	upper := Request{CommandID: "6F1C0D52-0000-4000-8000-00000000ABCD", SubjectKey: "EX-03", Metric: "issue_vs_measured", Day: day1}
	lower := upper
	lower.CommandID = strings.ToLower(upper.CommandID)
	first := wantDone(t, w.run(p, upper), false)
	again := wantDone(t, w.run(p, lower), true) // UUID không phân biệt hoa thường: cùng một khoá
	if again.ID != first.ID {
		t.Fatalf("hai cách viết hoa thường của cùng UUID phải là cùng một lệnh")
	}
	if got := w.onlyTicket().CommandID; got != lower.CommandID {
		t.Fatalf("command_id lưu ở dạng chữ thường, được %q", got)
	}
}

func TestMalformedLaLuotDungOC0(t *testing.T) {
	w := newWorld(t)
	o := w.pipe.Malformed("tenant")
	wantFail(t, o, StatusFailed, ReasonInvalidRequest)
	if !slices.Equal(o.Result.InvalidFields, []string{"tenant"}) || o.Question != "" {
		t.Fatalf("Malformed: invalid_fields=[tenant], Question rỗng khi thân không đọc được: %+v", o)
	}
	assertSteps(t, o, "C0")
	if !slices.Equal(w.pipe.Malformed().Result.InvalidFields, nil) {
		t.Fatalf("lỗi không quy về một trường thì invalid_fields rỗng")
	}
}

func TestLoiDocLaStoreUnavailable(t *testing.T) {
	boom := errors.New("lỗi thô: máy chủ nội bộ db-noi-bo không trả lời")
	cases := []struct {
		name string
		arm  func(*world)
		stop string
	}{
		{"danh mục thực thể", func(w *world) { w.data.dirErr = boom }, "C4"},
		{"sổ lệnh", func(w *world) { w.deps.Store = &failReads{Store: w.probe, ledger: boom}; w.restart() }, "C4b"},
		{"số liệu", func(w *world) { w.data.factsErr = boom }, "C5"},
		{"tìm phiếu mở", func(w *world) { w.deps.Store = &failReads{Store: w.probe, open: boom}; w.restart() }, "C5"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			w := newWorld(t)
			c.arm(w)
			o := w.run(user("p-1", "director", ""), req(80, "EX-03", day1))
			wantFail(t, o, StatusFailed, ReasonStoreUnavailable) // chưa mở giao dịch nào: biết chắc chưa ghi
			if o.Result.Fault != "infra:postgres" || o.Result.Retry != RetrySameKey {
				t.Fatalf("fault/retry sai: %+v", o.Result)
			}
			if strings.Contains(o.Result.Detail, "db-noi-bo") || strings.Contains(o.Result.Detail, "lỗi thô") {
				t.Fatalf("detail lộ lỗi gốc: %q", o.Result.Detail)
			}
			if !errors.Is(o.Cause, boom) {
				t.Fatalf("lỗi gốc phải nằm ở Outcome.Cause để ghi log: %v", o.Cause)
			}
			assertSteps(t, o, c.stop)
			w.check(tenantA, want{})
			w.counters(0, 0)
		})
	}
}

func TestLoiKhongPhanLoaiCuaDonViGhiLaUnknown(t *testing.T) {
	w := newWorld(t, func(d *Deps) {
		d.Store = &failReads{Store: d.Store, write: errors.New("lỗi lạ, không rõ đã ghi chưa")}
	})
	o := w.run(user("p-1", "director", ""), req(81, "EX-03", day1))
	wantFail(t, o, StatusUnknown, ReasonInternalError) // không xác nhận được rollback thì không được nói FAILED
	if o.Result.Fault != "code" || o.Result.Retry != RetryNone || strings.Contains(o.Result.Detail, "lạ") {
		t.Fatalf("INTERNAL_ERROR: fault code, retry none, detail không lộ lỗi gốc: %+v", o.Result)
	}
	assertSteps(t, o, "C6")
	if o.Steps[stWrite].Status != StepError || o.Cause == nil {
		t.Fatalf("C6 phải là error và giữ lỗi gốc: %+v", o.Steps[stWrite])
	}
}

// Vi phạm khoá duy nhất mà đọc lại không thấy gì (không sổ lệnh, không phiếu mở): không lường trước, giao dịch đã huỷ.
func TestViPhamKhoaNhungKhongThayGi(t *testing.T) {
	w := newWorld(t, func(d *Deps) { d.Store = &failReads{Store: d.Store, write: ErrUniqueViolation} })
	o := w.run(user("p-1", "director", ""), req(82, "EX-03", day1))
	wantFail(t, o, StatusFailed, ReasonInternalError)
	if o.Result.Fault != "code" {
		t.Fatalf("fault phải là code: %+v", o.Result)
	}
	assertSteps(t, o, "C6")
	w.check(tenantA, want{})
}

// Hạn chót: cái đến trước giữa ngữ cảnh của bên gọi và timeout của tool trong pack.
func TestHanChotCuaTool(t *testing.T) {
	w := newWorld(t, func(d *Deps) {
		d.Tool.Timeout = 50 * time.Millisecond
		d.Store = &failReads{Store: d.Store, block: true} // đọc sổ lệnh treo tới khi ngữ cảnh hết hạn
	})
	start := time.Now()
	o := w.run(user("p-1", "director", ""), req(83, "EX-03", day1))
	wantFail(t, o, StatusFailed, ReasonStoreUnavailable)
	if !errors.Is(o.Cause, context.DeadlineExceeded) || time.Since(start) > 3*time.Second {
		t.Fatalf("hạn chót của tool phải cắt lời gọi treo: cause=%v sau %v", o.Cause, time.Since(start))
	}
}

func TestNewTuChoiThieuPhuThuoc(t *testing.T) {
	w := newWorld(t)
	for name, mut := range map[string]func(*Deps){
		"Tenant":     func(d *Deps) { d.Tenant = "" },
		"Location":   func(d *Deps) { d.Location = nil },
		"Capability": func(d *Deps) { d.Tool.Capability = "" },
		"Catalog":    func(d *Deps) { d.Catalog = nil },
		"Store":      func(d *Deps) { d.Store = nil },
		"Facts":      func(d *Deps) { d.Facts = nil },
		"Directory":  func(d *Deps) { d.Directory = nil },
		"Metrics":    func(d *Deps) { d.Tool.Metrics = nil },
	} {
		d := w.deps
		mut(&d)
		if p, err := New(d); err == nil || p != nil || !strings.Contains(err.Error(), name) {
			t.Errorf("thiếu %s: New phải từ chối và nêu tên, được %v, %v", name, p, err)
		}
	}
}

// A29 (d): vết của lượt DONE (kể cả phát lại) mang mã phiếu và mã bằng chứng khớp bản chụp trong phiếu.
func TestA29VetLuotDoneKhopBanChup(t *testing.T) {
	w := newWorld(t)
	o := w.run(user("p-1", "director", ""), req(84, "EX-03", day1))
	tk := wantDone(t, o, false)
	stored := w.onlyTicket()
	var ids []string
	for _, e := range stored.Evidence {
		ids = append(ids, e.ID)
	}
	if !slices.Equal(o.Evidence, ids) || !slices.Equal(o.Steps[stRules].Evidence, ids) {
		t.Fatalf("mã bằng chứng của vết phải khớp bản chụp: vết %v / %v, phiếu %v", o.Evidence, o.Steps[stRules].Evidence, ids)
	}
	if !strings.Contains(strings.Join(o.Steps[stVerify].Output, " "), tk.ID) || o.Entity != "EX-03" {
		t.Fatalf("C7 phải ghi mã phiếu và Entity là khoá máy: %+v, entity=%q", o.Steps[stVerify], o.Entity)
	}
	if !strings.HasPrefix(o.Question, CommandOpenDiscrepancyTicket+" subject_key=EX-03 metric=issue_vs_measured day="+day1+" command_id=") {
		t.Fatalf("Question sai dạng chuẩn: %q", o.Question)
	}
	// phát lại không qua C5, nhưng vết vẫn mang mã phiếu và mã bằng chứng của phiếu đã đọc lại
	again := w.run(user("p-1", "director", ""), req(84, "EX-03", day1))
	wantDone(t, again, true)
	if !slices.Equal(again.Evidence, ids) || !slices.Equal(again.Steps[stVerify].Evidence, ids) || !strings.Contains(strings.Join(again.Steps[stVerify].Output, " "), tk.ID) {
		t.Fatalf("vết của lượt phát lại phải mang mã bằng chứng khớp bản chụp: %v / %v, phiếu %v", again.Evidence, again.Steps[stVerify].Evidence, ids)
	}
}
