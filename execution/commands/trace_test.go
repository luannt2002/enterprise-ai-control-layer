package commands

// Vết của lượt lệnh (HOP-DONG §7.5, A29 phần dựng bản ghi): NewRecord biến Outcome thành traces.Record chín bước C0…C7 mà kho vết lưu được, đọc được
// bằng `eais trace` (traces.FormatTable). Đỏ trước: NewRecord khởi đầu là hàm rỗng.

import (
	"encoding/json"
	"slices"
	"strings"
	"testing"

	"github.com/luannt2002/enterprise-ai-control-layer/core/contracts"
	"github.com/luannt2002/enterprise-ai-control-layer/observability/traces"
)

var testVersions = traces.Versions{Pack: "pack-test", Code: "code-test", Data: "data-test"}

var stageCodes = []string{"C0", "C1", "C2", "C3", "C4", "C4b", "C5", "C6", "C7"}

func recordOf(t *testing.T, r Request, o Outcome, pr contracts.Principal) traces.Record {
	t.Helper()
	rec := NewRecord(r, o, pr, fixedNow, testVersions, 7)
	if err := rec.Validate(); err != nil {
		t.Fatalf("bản ghi vết không hợp lệ: %v", err)
	}
	return rec
}

// firstDeviant là "<mã> <tên>" của dòng BƯỚC LỆCH ĐẦU TIÊN trong bảng vết; "" khi bảng nói không có bước lệch; "?" khi bảng không nói gì.
func firstDeviant(rec traces.Record) string {
	table := traces.FormatTable(rec)
	for _, l := range strings.Split(table, "\n") {
		if rest, ok := strings.CutPrefix(l, "BƯỚC LỆCH ĐẦU TIÊN: "); ok {
			name, _, _ := strings.Cut(rest, " — ")
			return name
		}
	}
	if strings.Contains(table, "Không có bước lệch.") {
		return ""
	}
	return "?"
}

func TestNewRecordLuotDone(t *testing.T) {
	w := newWorld(t)
	pr, rq := user("u-dir", "director", ""), req(1, "EX-03", day1)
	out := w.run(pr, rq)
	wantDone(t, out, false)
	rec := recordOf(t, rq, out, pr)

	if len(rec.Steps) != len(stageCodes) {
		t.Fatalf("vết phải có %d bước, được %d", len(stageCodes), len(rec.Steps))
	}
	for i, s := range rec.Steps {
		if s.Stage != stageCodes[i] || s.Seq != i || s.TraceID != out.Result.TraceID || s.Name == "" || s.Decision == "" || s.Reason == "" {
			t.Errorf("bước %d = %+v, muốn mã %s, seq %d, trace_id %s, có tên, quyết định và lý do", i, s, stageCodes[i], i, out.Result.TraceID)
		}
		if s.PackVersion != "pack-test" || s.CodeVersion != "code-test" || s.DataVersion != "data-test" {
			t.Errorf("bước %s thiếu phiên bản: %+v", s.Stage, s)
		}
	}
	d := rec.Decision
	if d.TraceID != out.Result.TraceID || d.Tenant != string(tenantA) || d.Principal != "u-dir" || d.Kind != "user" || !slices.Equal(d.Roles, []string{"director"}) {
		t.Errorf("định danh trong quyết định sai: %+v", d)
	}
	if d.Intent != CommandOpenDiscrepancyTicket || d.Route != string(contracts.RouteTool) || d.Outcome != "DONE" {
		t.Errorf("intent=%q route=%q outcome=%q", d.Intent, d.Route, d.Outcome)
	}
	if !slices.Equal(d.Slots, []string{"metric=issue_vs_measured", "day=" + day1}) || !slices.Equal(d.Entities, []string{"EX-03"}) {
		t.Errorf("slots=%v entities=%v", d.Slots, d.Entities)
	}
	if want := "open_discrepancy_ticket subject_key=EX-03 metric=issue_vs_measured day=" + day1 + " command_id=" + cmdID(1); d.Question != want {
		t.Errorf("question = %q, muốn %q", d.Question, want)
	}
	if len(d.Evidence) == 0 || !slices.Equal(d.Evidence, out.Evidence) {
		t.Errorf("evidence trong quyết định = %v, muốn mã bằng chứng của bản chụp %v", d.Evidence, out.Evidence)
	}
	if want := "status=DONE reason_code= replayed=false ticket=" + out.Result.Ticket.ID; d.Answer != want {
		t.Errorf("answer = %q, muốn %q", d.Answer, want)
	}
	if d.Normalized == nil || d.Verdict != "" || len(d.Normalized) != 0 {
		t.Errorf("normalized phải rỗng không nil, verdict rỗng: %+v", d)
	}
	if rec.Fault != "" || rec.Error != "" {
		t.Errorf("lượt DONE không có fault: %q %q", rec.Fault, rec.Error)
	}
	if got := firstDeviant(rec); got != "" {
		t.Errorf("lượt DONE không có bước lệch, bảng nói %q", got)
	}
	c5 := rec.Steps[6]
	if !slices.Equal(evidenceIDs(c5), out.Evidence) {
		t.Errorf("bằng chứng ở C5 = %v, muốn %v", evidenceIDs(c5), out.Evidence)
	}
}

func evidenceIDs(s traces.Step) []string {
	var ids []string
	for _, e := range s.Evidence {
		ids = append(ids, e.ID)
	}
	return ids
}

// Mỗi kết cục để lại đúng chặng gây ra nó ở dòng BƯỚC LỆCH ĐẦU TIÊN của bảng vết (A29 (b)): cột "Dừng ở đây thì" của HOP-DONG §7.5.
func TestNewRecordMoiKetCuc(t *testing.T) {
	dir, eng := user("u-dir", "director", ""), user("u-eng", "site_engineer", "PRJ-A")
	other := user("u-other", "director", "")
	other.TenantID = tenantB
	cases := []struct {
		name    string
		run     func(w *world) (contracts.Principal, Request, Outcome)
		stop    string // "<mã> <tên>" của bước lệch đầu tiên; "" = không có
		outcome string
		fault   string // nhãn fault ở cấp bản ghi; "" khi lượt không lỗi
	}{
		{"DONE", func(w *world) (contracts.Principal, Request, Outcome) {
			r := req(1, "EX-03", day1)
			return dir, r, w.run(dir, r)
		}, "", "DONE", ""},
		{"phát lại", func(w *world) (contracts.Principal, Request, Outcome) {
			r := req(1, "EX-03", day1)
			w.run(dir, r)
			return dir, r, w.run(dir, r)
		}, "", "DONE", ""},
		{"C0 thân không giải mã được", func(w *world) (contracts.Principal, Request, Outcome) {
			return dir, Request{}, w.pipe.Malformed("tenant")
		}, "C0 accept", "INVALID_REQUEST", ""},
		{"C1 tenant khác runtime", func(w *world) (contracts.Principal, Request, Outcome) {
			r := req(1, "EX-03", day1)
			return other, r, w.run(other, r)
		}, "C1 tenant", "PERMISSION_DENIED", ""},
		{"C3 chỉ số ngoài danh sách", func(w *world) (contracts.Principal, Request, Outcome) {
			r := req(1, "EX-03", day1)
			r.Metric = "issued_fuel"
			return dir, r, w.run(dir, r)
		}, "C3 params", "INVALID_REQUEST", ""},
		{"C4 thiếu tickets.create", func(w *world) (contracts.Principal, Request, Outcome) {
			r := req(1, "EX-03", day1)
			return eng, r, w.run(eng, r)
		}, "C4 permission", "PERMISSION_DENIED", ""},
		{"C4b dùng lại khoá cho nội dung khác", func(w *world) (contracts.Principal, Request, Outcome) {
			w.run(dir, req(2, "EX-03", day1))
			r := req(2, "EX-03", day2)
			return dir, r, w.run(dir, r)
		}, "C4b ledger", "IDEMPOTENCY_KEY_REUSED", ""},
		{"C5 hai vế bằng nhau", func(w *world) (contracts.Principal, Request, Outcome) {
			r := req(1, "EX-01", day1)
			return dir, r, w.run(dir, r)
		}, "C5 rules", "NO_DISCREPANCY", ""},
		{"C5 trùng phiếu đang mở", func(w *world) (contracts.Principal, Request, Outcome) {
			w.run(dir, req(3, "EX-03", day1))
			r := req(4, "EX-03", day1)
			return dir, r, w.run(dir, r)
		}, "C5 rules", "DUPLICATE_OPEN_TICKET", ""},
		{"C6 hạ tầng trước COMMIT (F1)", func(w *world) (contracts.Principal, Request, Outcome) {
			w.probe.f1 = 1
			r := req(1, "EX-03", day1)
			return dir, r, w.run(dir, r)
		}, "C6 write", "STORE_UNAVAILABLE", "infra:postgres"},
		{"C6 mất ack của COMMIT (F3)", func(w *world) (contracts.Principal, Request, Outcome) {
			w.probe.f3 = 1
			r := req(1, "EX-03", day1)
			return dir, r, w.run(dir, r)
		}, "C6 write", "COMMIT_UNKNOWN", "infra:postgres"},
		{"C7 đọc lại hỏng (F6)", func(w *world) (contracts.Principal, Request, Outcome) {
			w.probe.f6fail = 1
			r := req(1, "EX-03", day1)
			return dir, r, w.run(dir, r)
		}, "C7 verify", "READBACK_UNAVAILABLE", "infra:postgres"},
		{"C7 đọc lại lệch trường (F6)", func(w *world) (contracts.Principal, Request, Outcome) {
			w.probe.f6stale, w.probe.staleMode = 1, "diff"
			r := req(1, "EX-03", day1)
			return dir, r, w.run(dir, r)
		}, "C7 verify", "VERIFY_MISMATCH", "code"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			w := newWorld(t)
			pr, rq, out := c.run(w)
			rec := recordOf(t, rq, out, pr)
			if got := firstDeviant(rec); got != c.stop {
				t.Errorf("bước lệch đầu tiên = %q, muốn %q\n%s", got, c.stop, traces.FormatTable(rec))
			}
			if rec.Decision.Outcome != c.outcome {
				t.Errorf("outcome = %q, muốn %q", rec.Decision.Outcome, c.outcome)
			}
			if rec.Fault != c.fault {
				t.Errorf("fault cấp bản ghi = %q, muốn %q", rec.Fault, c.fault)
			}
			if rec.Fault != "" && rec.Steps[len(rec.Steps)-1].Stage != "C7" {
				t.Errorf("C7 luôn là bước cuối")
			}
			for i, s := range rec.Steps {
				if s.Stage != stageCodes[i] {
					t.Errorf("bước %d = %s, muốn %s", i, s.Stage, stageCodes[i])
				}
			}
			if c.name == "C0 thân không giải mã được" && (rec.Decision.Question != "" || len(rec.Decision.Slots) != 0) {
				t.Errorf("thân không đọc được thì question và slots rỗng: %q %v", rec.Decision.Question, rec.Decision.Slots)
			}
		})
	}
}

// A29 (c) và M24 ở mức dựng bản ghi: "máy không tồn tại" và "máy ngoài phạm vi" cho hai vết GIỐNG HỆT nhau ngoài khoá máy người gọi đã gõ, trace_id,
// mã lệnh; vết không nói lý do thật và không chứa giá trị ngoài quyền đọc.
func TestNewRecordKhongLoPhanBietMayVaGiaTri(t *testing.T) {
	pm := user("u-pm", "project_manager", "PRJ-A")
	render := func(key string, n int) (string, traces.Record) {
		w := newWorld(t)
		r := req(n, key, day1)
		out := w.run(pm, r)
		wantFail(t, out, StatusFailed, ReasonPermissionDenied)
		rec := recordOf(t, r, out, pm)
		b, err := json.Marshal(rec)
		if err != nil {
			t.Fatal(err)
		}
		s := strings.NewReplacer(key, "<KEY>", out.Result.TraceID, "<TRACE>", cmdID(n), "<CMD>").Replace(string(b))
		return s, rec
	}
	missing, recMissing := render("EX-99", 1) // không có trong danh mục
	outside, _ := render("EX-09", 2)          // có, nhưng thuộc PRJ-B ngoài phạm vi của PRJ-A
	if missing != outside {
		t.Errorf("hai vết PERMISSION_DENIED khác nhau ngoài khoá máy, trace_id và mã lệnh:\n%s\n%s", missing, outside)
	}
	for _, leak := range []string{"không tồn tại", "ngoài phạm vi", "không có trong danh mục", "3850", "3782", "700", "650"} {
		if strings.Contains(missing, leak) {
			t.Errorf("vết lộ %q", leak)
		}
	}
	if len(recMissing.Decision.Entities) != 0 {
		t.Errorf("chưa qua C4 thì không có khoá máy trong entities: %v", recMissing.Decision.Entities)
	}
}

// Lượt hạ tầng hỏng: bản ghi mang nhãn fault và thông điệp ĐÃ làm sạch ở cấp bản ghi (cột fault, error_message của kho vết), như lượt hỏi lỗi.
func TestNewRecordLoiHaTangCoNhanVaThongDiepDaLamSach(t *testing.T) {
	w := newWorld(t)
	pr := user("u-dir", "director", "")
	w.probe.f1 = 1
	rq := req(1, "EX-03", day1)
	out := w.run(pr, rq)
	rec := recordOf(t, rq, out, pr)
	if rec.Fault != "infra:postgres" || rec.Error == "" {
		t.Fatalf("muốn fault infra:postgres kèm thông điệp, được %q %q", rec.Fault, rec.Error)
	}
	c6 := rec.Steps[7]
	if c6.Status != traces.StatusError || c6.Fault != "infra:postgres" || c6.Error == "" {
		t.Errorf("bước C6 = %+v", c6)
	}
	// Result không đổi chữ nào so với trước khi dựng vết: nhãn đầy đủ chỉ ở vết.
	if out.Result.Fault != "infra:postgres" || out.Result.FaultClass() != "infra" {
		t.Errorf("result fault = %q", out.Result.Fault)
	}
}
