package traces_test

import (
	"testing"

	"github.com/luannt2002/enterprise-ai-control-layer/observability/traces"
)

// Bước ĐẦU TIÊN (S2→S9) mà hai bản ghi khác nhau: mỗi chặng một ca, cặp chặng thì chặng sớm thắng, và hai quy ước của hàm.
func TestFirstDifferentStep(t *testing.T) {
	base := traces.NewRecord(sampleTurn())
	with := func(f func(d *traces.Decision)) traces.Record {
		r := base
		f(&r.Decision)
		return r
	}
	for _, c := range []struct {
		name      string
		b         traces.Record
		stage     string
		diffName  string
		wantFound bool
	}{
		{"khớp hoàn toàn", base, "", "", false},
		{"chuẩn hoá", with(func(d *traces.Decision) { d.Normalized = []string{"date:hôm qua=2026-10-18"} }), "S2", "normalize", true},
		{"thực thể", with(func(d *traces.Decision) { d.Entities = []string{"DZ-03"} }), "S3", "resolve", true},
		{"ý định (tên)", with(func(d *traces.Decision) { d.Intent = "fuel_reported" }), "S4", "intent", true},
		{"ý định (khe)", with(func(d *traces.Decision) { d.Slots = []string{"entity=EX-03"} }), "S4", "intent", true},
		{"route", with(func(d *traces.Decision) { d.Route = "REJECT" }), "S5", "route", true},
		{"outcome", with(func(d *traces.Decision) { d.Outcome = "INSUFFICIENT_EVIDENCE" }), "S6", "outcome", true},
		{"bằng chứng", with(func(d *traces.Decision) { d.Evidence = []string{"E1"} }), "S7", "evidence", true},
		{"chỉ câu trả lời khác (verdict giữ nguyên) là S8", with(func(d *traces.Decision) { d.Answer = "câu khác" }), "S8", "answer", true},
		{"chỉ verdict khác", with(func(d *traces.Decision) { d.Verdict = "false|fallback|X" }), "S9", "verdict", true},
		{"verdict và câu cùng khác thì lỗi bắt đầu từ S9", with(func(d *traces.Decision) { d.Verdict, d.Answer = "false|fallback|X", "mẫu thay thế" }), "S9", "verdict", true},
		{"chặng sớm thắng chặng muộn", with(func(d *traces.Decision) { d.Route, d.Answer, d.Verdict = "REJECT", "x", "y" }), "S5", "route", true},
		{"chuẩn hoá thắng verdict", with(func(d *traces.Decision) { d.Normalized, d.Verdict = []string{"khác"}, "y" }), "S2", "normalize", true},
		// phiên bản mã khác nhau là LÝ DO hai mốc được so, không phải kết quả của một bước
		{"chỉ phiên bản mã khác", with(func(d *traces.Decision) { d.CodeVersion = "rev-khac" }), "", "", false},
	} {
		got, ok := traces.FirstDifferentStep(base, c.b)
		if ok != c.wantFound || got.Stage != c.stage || got.Name != c.diffName {
			t.Errorf("%s: được (%+v, %v), muốn stage %q name %q found %v", c.name, got, ok, c.stage, c.diffName, c.wantFound)
		}
	}
	got, _ := traces.FirstDifferentStep(base, with(func(d *traces.Decision) { d.Route = "REJECT" }))
	if got.A != "SQL_QUERY" || got.B != "REJECT" {
		t.Errorf("giá trị hai bên để in: %+v", got)
	}
}

// Một bên lỗi, một bên không (hoặc nhãn lỗi khác): trả S? trước mọi thứ — lượt lỗi không có Decision từ luồng hỏi nên so các chặng
// còn lại chỉ ra khác biệt giả.
func TestFirstDifferentStepLuotLoi(t *testing.T) {
	ok := traces.NewRecord(sampleTurn())
	bad := ok
	bad.Fault = "infra:postgres"
	bad.Decision.Normalized, bad.Decision.Route = nil, "" // lượt lỗi: các chặng trống
	got, found := traces.FirstDifferentStep(ok, bad)
	if !found || got.Stage != traces.StageUnknown || got.Name != "fault" || got.A != "" || got.B != "infra:postgres" {
		t.Errorf("được (%+v, %v)", got, found)
	}
	other := bad
	other.Fault = "policy"
	if got, found := traces.FirstDifferentStep(bad, other); !found || got.Stage != traces.StageUnknown {
		t.Errorf("hai nhãn lỗi khác nhau: được (%+v, %v)", got, found)
	}
	if _, found := traces.FirstDifferentStep(bad, bad); found {
		t.Errorf("hai lượt lỗi giống hệt không được báo khác")
	}
}

// Băm ổn định giữa hai lần tính trên cùng nội dung và đổi ở MỌI thay đổi nội dung, dù nhỏ: kho vết dựa vào đó để phân biệt ghi lại
// y nguyên với đụng trace_id.
func TestDigest(t *testing.T) {
	a, err := traces.SanitizeRecord(traces.NewRecord(sampleTurn()))
	if err != nil {
		t.Fatal(err)
	}
	b, _ := traces.SanitizeRecord(traces.NewRecord(sampleTurn()))
	da, _ := a.Digest()
	db, _ := b.Digest()
	if da != db || len(da) != 64 {
		t.Errorf("hai lần dựng cùng nội dung phải cùng băm 64 hex: %q vs %q", da, db)
	}
	for name, mut := range map[string]func(r *traces.Record){
		"câu trả lời":    func(r *traces.Record) { r.Decision.Answer += "." },
		"một bước":       func(r *traces.Record) { r.Steps[2].Reason += "." },
		"thời gian":      func(r *traces.Record) { r.Spans = append([]traces.Span{}, r.Spans...); r.Spans[0].Ms++ },
		"lỗi":            func(r *traces.Record) { r.Fault = "code" },
		"tổng thời gian": func(r *traces.Record) { r.TotalMs++ },
	} {
		c, _ := traces.SanitizeRecord(a)
		mut(&c)
		if dc, _ := c.Digest(); dc == da {
			t.Errorf("đổi %s mà băm không đổi", name)
		}
	}
}
