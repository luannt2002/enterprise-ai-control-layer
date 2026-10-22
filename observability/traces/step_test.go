package traces_test

import (
	"errors"
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/luannt2002/enterprise-ai-control-layer/core/contracts"
	"github.com/luannt2002/enterprise-ai-control-layer/observability/traces"
)

// sampleTurn là một lượt số liệu giải xong, dựng tay theo đúng hình pipeline.Outcome trả về: có xung đột nguồn giải bằng thẩm quyền.
func sampleTurn() traces.Turn {
	d := traces.Decision{
		TraceID: "trace-1", Tenant: "ta", Principal: "u-director", Kind: "user", Roles: []string{"director"},
		Attributes: map[string]string{"entity_scope": "PRJ-A"}, Now: time.Date(2026, 10, 19, 10, 0, 0, 0, time.UTC),
		PackVersion: "pack-v1", CodeVersion: "rev-abc", DataVersion: "data-1",
		Question:   "EX-03 hôm nay được cấp bao nhiêu lít dầu?",
		Normalized: []string{"date:hôm nay=2026-10-19"}, Entities: []string{"EX-03"}, Intent: "fuel_issued",
		Slots: []string{"entity=EX-03", "period=2026-10-19"}, Route: "SQL_QUERY", Outcome: "RESOLVED",
		Evidence: []string{"E1", "E2"}, Verdict: "true|accept", Answer: "Dầu kho xuất của Máy xúc EX-03: 3.850 L.",
	}
	plan := contracts.RoutePlan{
		Route: contracts.RouteSQLQuery, Template: "metric_total", Reason: "ý định số liệu có đủ khe",
		Intent: contracts.Intent{Name: "fuel_issued", Confidence: 0.9, Rule: "từ khoá: cấp + dầu"},
		Steps:  []contracts.Step{{Kind: contracts.StepTool, Name: "facts", Args: map[string]string{"metric": "issued_fuel", "entity": "EX-03"}}},
	}
	resp := contracts.AskResponse{
		Answer: d.Answer, Outcome: contracts.OutcomeResolved, Route: contracts.RouteSQLQuery, Intent: "fuel_issued", TraceID: "trace-1",
		Verdict: contracts.Verdict{Pass: true, Action: contracts.ActionAccept},
		Citations: []contracts.Citation{
			{EvidenceID: "E1", Source: contracts.SourceRef{SourceType: "warehouse_slip", Name: "Phiếu xuất kho R-1"}},
			{EvidenceID: "E2", Source: contracts.SourceRef{SourceType: "erp", Name: "ERP"}},
		},
		Conflicts: []contracts.Conflict{{
			Key: "issued_fuel|EX-03|2026-10-19", EvidenceIDs: []string{"E1", "E2"}, Policy: "authority", ChosenID: "E1",
			Reason: "phiếu kho có thẩm quyền cao hơn ERP", Others: []contracts.SourceValue{{SourceType: "erp", Number: "3900", Unit: "L"}},
		}},
	}
	spans := []traces.Span{{Name: "normalize", Ms: 1}, {Name: "resolve", Ms: 2}, {Name: "intent", Ms: 3}, {Name: "route", Ms: 4}, {Name: "execute", Ms: 5}, {Name: "validate", Ms: 6}}
	return traces.Turn{Decision: d, Spans: spans, TotalMs: 22, UnaccountedMs: 1, Plan: plan, Response: resp}
}

func stepOf(t *testing.T, steps []traces.Step, stage string) traces.Step {
	t.Helper()
	for _, s := range steps {
		if s.Stage == stage {
			return s
		}
	}
	t.Fatalf("không có bước %s", stage)
	return traces.Step{}
}

// Mười một bước S0…S10, đúng thứ tự, mỗi bước mang trace id và ba phiên bản của lượt.
func TestBuildStepsMuoiMotBuoc(t *testing.T) {
	steps := traces.BuildSteps(sampleTurn())
	if len(steps) != 11 {
		t.Fatalf("muốn 11 bước, được %d", len(steps))
	}
	for i, s := range steps {
		if want := "S" + strconv.Itoa(i); s.Stage != want || s.Seq != i {
			t.Errorf("bước %d: stage %q seq %d, muốn %s", i, s.Stage, s.Seq, want)
		}
		if s.TraceID != "trace-1" || s.PackVersion != "pack-v1" || s.CodeVersion != "rev-abc" || s.DataVersion != "data-1" {
			t.Errorf("%s: trace id / phiên bản sai: %+v", s.Stage, s)
		}
		if !traces.ValidStage(s.Stage) || s.Name == "" || s.Decision == "" || s.Reason == "" {
			t.Errorf("%s: thiếu mã bước, tên, quyết định hoặc lý do: %+v", s.Stage, s)
		}
	}
}

// Thời gian chỉ gắn vào bước có span; bước không có đồng hồ riêng là "chưa đo", không phải 0 ms.
func TestBuildStepsThoiGian(t *testing.T) {
	turn := sampleTurn()
	steps := traces.BuildSteps(turn)
	want := map[string]int64{"S2": 1, "S3": 2, "S4": 3, "S5": 4, "S6": 5, "S9": 6}
	for _, s := range steps {
		ms, timed := want[s.Stage]
		if s.Timed != timed || (timed && s.Ms != ms) || (!timed && s.Ms != 0) {
			t.Errorf("%s: timed=%v ms=%d, muốn timed=%v ms=%d", s.Stage, s.Timed, s.Ms, timed, ms)
		}
	}
	// route HYBRID chạy validate hai lần (số liệu, rồi tri thức): bước S9 lấy TỔNG
	turn.Spans = append(turn.Spans, traces.Span{Name: "validate", Ms: 7}, traces.Span{Name: "generate", Ms: 8}, traces.Span{Name: "retrieve", Ms: 4})
	steps = traces.BuildSteps(turn)
	if s := stepOf(t, steps, "S6"); s.Ms != 9 {
		t.Errorf("S6 phải là tổng execute (5) và retrieve (4) khi lượt có cả hai, được %d", s.Ms)
	}
	if s := stepOf(t, steps, "S9"); s.Ms != 13 {
		t.Errorf("S9 phải là tổng hai lần validate (13), được %d", s.Ms)
	}
	if s := stepOf(t, steps, "S8"); !s.Timed || s.Ms != 8 {
		t.Errorf("S8 có span generate 8 ms: %+v", s)
	}
}

// Lý do / phương án bị loại mà luồng hỏi chưa xuất thì ghi đúng NotRecorded — không bịa; cái đã xuất thì lấy nguyên văn.
func TestBuildStepsChuaGhiKhongBia(t *testing.T) {
	if traces.NotRecorded != "chưa ghi — cần móc trong core/pipeline (làn 1)" {
		t.Fatalf("giá trị NotRecorded đổi: %q", traces.NotRecorded)
	}
	turn := sampleTurn()
	steps := traces.BuildSteps(turn)
	for _, stage := range []string{"S2", "S3", "S4", "S5"} {
		if r := stepOf(t, steps, stage).Rejected; len(r) != 1 || !traces.IsNotRecorded(r[0]) {
			t.Errorf("%s: phương án bị loại chưa xuất phải là NotRecorded, được %v", stage, r)
		}
	}
	for _, stage := range []string{"S2", "S3", "S7"} {
		if r := stepOf(t, steps, stage).Reason; !traces.IsNotRecorded(r) {
			t.Errorf("%s: lý do chưa xuất phải là NotRecorded, được %q", stage, r)
		}
	}
	if r := stepOf(t, steps, "S4").Reason; r != "từ khoá: cấp + dầu" {
		t.Errorf("S4 lấy lý do từ Intent.Rule, được %q", r)
	}
	if r := stepOf(t, steps, "S5").Reason; r != "ý định số liệu có đủ khe" {
		t.Errorf("S5 lấy lý do từ RoutePlan.Reason, được %q", r)
	}
	// Rule / Reason rỗng thì không được bịa một lý do: ghi chỗ trống
	turn.Plan.Intent.Rule, turn.Plan.Reason = "", ""
	steps = traces.BuildSteps(turn)
	for _, stage := range []string{"S4", "S5"} {
		if r := stepOf(t, steps, stage).Reason; !traces.IsNotRecorded(r) {
			t.Errorf("%s: Rule / Reason rỗng phải thành NotRecorded, được %q", stage, r)
		}
	}
}

// Xung đột nguồn giải bằng thẩm quyền: S6 lệch, và phương án đã xét rồi bị loại (bằng chứng không chọn, giá trị nguồn khác) có thật.
func TestBuildStepsXungDot(t *testing.T) {
	s6 := stepOf(t, traces.BuildSteps(sampleTurn()), "S6")
	if s6.Status != traces.StatusDeviant {
		t.Errorf("S6 có xung đột phải lệch, được %q", s6.Status)
	}
	got := strings.Join(s6.Rejected, " | ")
	if !strings.Contains(got, "bằng chứng E2") || strings.Contains(got, "bằng chứng E1") || !strings.Contains(got, "nguồn erp = 3900 L") {
		t.Errorf("phương án bị loại phải là E2 và giá trị ERP, không phải E1 (đã chọn): %s", got)
	}
	if !strings.Contains(s6.Reason, "phiếu kho có thẩm quyền cao hơn ERP") {
		t.Errorf("lý do phải mang lý do của xung đột: %q", s6.Reason)
	}
}

// Bằng chứng kèm nguồn gốc: S7 phản chiếu đúng các trích dẫn của câu trả lời.
func TestBuildStepsBangChungCoNguon(t *testing.T) {
	s7 := stepOf(t, traces.BuildSteps(sampleTurn()), "S7")
	if len(s7.Evidence) != 2 || s7.Evidence[0].ID != "E1" || s7.Evidence[0].Source.SourceType != "warehouse_slip" || s7.Evidence[1].Source.SourceType != "erp" {
		t.Errorf("bằng chứng và nguồn: %+v", s7.Evidence)
	}
	if !slices.Equal(s7.Output, []string{"E1", "E2"}) {
		t.Errorf("đầu ra S7: %v", s7.Output)
	}
}

// Hỏi lại: không truy vấn, không kiểm — S6, S7 bỏ qua, S9 ghi verdict do route gán; chỗ mơ hồ và route kết thúc sớm bị đánh dấu lệch.
func TestBuildStepsHoiLai(t *testing.T) {
	turn := sampleTurn()
	turn.Decision.Route, turn.Decision.Outcome, turn.Decision.Verdict = "CLARIFICATION", "CLARIFICATION_REQUIRED", "true|clarify"
	turn.Decision.Entities, turn.Decision.Evidence = []string{"?EX-03|DZ-03"}, nil
	turn.Plan.Route, turn.Plan.Template, turn.Plan.Steps = contracts.RouteClarification, "clarify_missing", nil
	turn.Response.Conflicts, turn.Response.Citations = nil, nil
	turn.Spans = []traces.Span{{Name: "normalize"}, {Name: "resolve"}, {Name: "intent"}, {Name: "route"}}
	steps := traces.BuildSteps(turn)
	for _, stage := range []string{"S6", "S7"} {
		if s := stepOf(t, steps, stage); s.Status != traces.StatusSkipped || s.Timed {
			t.Errorf("%s phải bị bỏ qua và chưa đo: %+v", stage, s)
		}
	}
	if s := stepOf(t, steps, "S9"); s.Status != traces.StatusSkipped || !strings.Contains(s.Decision, "true|clarify") {
		t.Errorf("S9 không chạy nhưng phải ghi verdict do route gán: %+v", s)
	}
	for _, stage := range []string{"S3", "S5", "S10"} {
		if s := stepOf(t, steps, stage); s.Status != traces.StatusDeviant {
			t.Errorf("%s phải lệch (mơ hồ / kết thúc sớm / outcome khác RESOLVED), được %q", stage, s.Status)
		}
	}
	if s := stepOf(t, steps, "S8"); !strings.Contains(s.Decision, "clarify_missing") || !strings.Contains(s.Reason, "không cần model") {
		t.Errorf("S8 dựng bằng mẫu, không model: %+v", s)
	}
}

// Kiểm trượt: S9 lệch, lý do là chính các vi phạm, câu nháp bị loại còn là chỗ trống ghi rõ (Draft chưa xuất).
func TestBuildStepsKiemTruot(t *testing.T) {
	turn := sampleTurn()
	turn.Decision.Route, turn.Decision.Verdict, turn.Decision.Outcome = "RAG", "false|fallback|NUM_NOT_IN_EVIDENCE", "SYSTEM_FAILURE"
	turn.Plan.Route = contracts.RouteRAG
	turn.Response.Verdict = contracts.Verdict{Pass: false, Action: contracts.ActionFallback, Violations: []contracts.Violation{
		{Validator: "numbers", Code: "NUM_NOT_IN_EVIDENCE", Severity: contracts.SeverityS1, Detail: "5.000 không có trong bằng chứng"}}}
	steps := traces.BuildSteps(turn)
	s9 := stepOf(t, steps, "S9")
	if s9.Status != traces.StatusDeviant || !strings.Contains(s9.Reason, "NUM_NOT_IN_EVIDENCE") || !strings.Contains(s9.Reason, "5.000 không có trong bằng chứng") {
		t.Errorf("S9: %+v", s9)
	}
	if len(s9.Rejected) != 1 || !traces.IsNotRecorded(s9.Rejected[0]) {
		t.Errorf("câu nháp bị loại chưa được luồng xuất, phải ghi NotRecorded: %v", s9.Rejected)
	}
	if s8 := stepOf(t, steps, "S8"); s8.Decision != "model diễn đạt" || !traces.IsNotRecorded(s8.Reason) {
		t.Errorf("S8 của route RAG: model không được luồng xuất ra: %+v", s8)
	}
	if s10 := stepOf(t, steps, "S10"); s10.Status != traces.StatusDeviant {
		t.Errorf("S10 outcome SYSTEM_FAILURE phải lệch: %+v", s10)
	}
}

func TestValidStage(t *testing.T) {
	for stage, want := range map[string]bool{
		"S0": true, "S10": true, "S5": true, "S?": true, "L1.read": true, "L1.quarantine": true,
		"S11": false, "S": false, "S05": false, "S-1": false, "": false, "L1.": false, "L2.read": false, "s5": false, "S 5": false,
	} {
		if got := traces.ValidStage(stage); got != want {
			t.Errorf("ValidStage(%q) = %v, muốn %v", stage, got, want)
		}
	}
}

// Lượt lỗi: Handle trả Outcome rỗng nên bản ghi chỉ có những gì nơi gọi biết, trong MỘT bước S? ghi rõ là chưa biết chặng nào lỗi.
func TestNewErrorRecord(t *testing.T) {
	pr := contracts.Principal{ID: "u1", TenantID: "ta", Kind: contracts.PrincipalUser, Roles: []string{"director"}}
	now := time.Date(2026, 10, 19, 10, 0, 0, 0, time.UTC)
	err := contracts.Label(contracts.Infra("postgres"), errors.New("kết nối postgres://u:hunter2pw@h/db hỏng"))
	rec := traces.NewErrorRecord("trace-e", pr, "EX-03 hôm nay?", now, traces.Versions{Pack: "p", Code: "c", Data: "d"}, err)
	if err := rec.Validate(); err != nil {
		t.Fatal(err)
	}
	if rec.Fault != "infra:postgres" || strings.Contains(rec.Error, "hunter2pw") || rec.Decision.Tenant != "ta" || rec.Decision.Question != "EX-03 hôm nay?" {
		t.Errorf("bản ghi lượt lỗi: %+v", rec)
	}
	if len(rec.Steps) != 1 {
		t.Fatalf("muốn 1 bước, được %d", len(rec.Steps))
	}
	s := rec.Steps[0]
	if s.Stage != traces.StageUnknown || s.Status != traces.StatusError || s.Fault != rec.Fault || !traces.IsNotRecorded(s.Reason) || s.TraceID != "trace-e" {
		t.Errorf("bước lỗi: %+v", s)
	}
	if rec.Spans == nil {
		t.Errorf("Spans phải là mảng rỗng, không nil")
	}
}

func TestRecordValidate(t *testing.T) {
	ok := traces.NewRecord(sampleTurn())
	if err := ok.Validate(); err != nil {
		t.Fatalf("bản ghi hợp lệ bị từ chối: %v", err)
	}
	bad := map[string]func(r *traces.Record){
		"thiếu trace_id":               func(r *traces.Record) { r.Decision.TraceID = "" },
		"thông điệp lỗi không có nhãn": func(r *traces.Record) { r.Error = "x" },
		"mã bước sai":                  func(r *traces.Record) { r.Steps = append([]traces.Step{}, r.Steps...); r.Steps[3].Stage = "S11" },
		"bước mang trace_id của lượt khác": func(r *traces.Record) {
			r.Steps = append([]traces.Step{}, r.Steps...)
			r.Steps[2].TraceID = "trace-khac"
		},
	}
	for name, mut := range bad {
		r := ok
		mut(&r)
		if r.Validate() == nil {
			t.Errorf("%s: Validate phải từ chối", name)
		}
	}
}
