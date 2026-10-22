package traces_test

import (
	"context"
	"encoding/json"
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/luannt2002/enterprise-ai-control-layer/core/contracts"
	"github.com/luannt2002/enterprise-ai-control-layer/core/pipeline"
	"github.com/luannt2002/enterprise-ai-control-layer/core/tenant"
	"github.com/luannt2002/enterprise-ai-control-layer/intelligence/models"
	"github.com/luannt2002/enterprise-ai-control-layer/intelligence/rag"
	"github.com/luannt2002/enterprise-ai-control-layer/observability/traces"
	"github.com/luannt2002/enterprise-ai-control-layer/simulations/fixtures"
)

// Các test dưới đây chạy lượt hỏi THẬT (pack + dữ liệu giả lập đã đóng băng của tenant cầu đường, không gọi model trả tiền) rồi dựng
// vết từ chính pipeline.Outcome — thứ duy nhất chứng minh BuildSteps khớp hình dữ liệu thật chứ không chỉ khớp bản dựng tay.

var today = time.Date(2026, 10, 19, 10, 0, 0, 0, time.FixedZone("ICT", 7*3600))

func realAsk(t *testing.T) *pipeline.Ask {
	t.Helper()
	rt, err := tenant.Load("../../tenants/construction/pack")
	if err != nil {
		t.Fatalf("nạp pack: %v", err)
	}
	dir, err := fixtures.LoadDirectory("../../tenants/construction/seed/entities.yaml")
	if err != nil {
		t.Fatalf("nạp danh mục: %v", err)
	}
	facts, err := fixtures.LoadFacts("../../tenants/construction/seed/facts.yaml", rt, dir)
	if err != nil {
		t.Fatalf("nạp fact: %v", err)
	}
	a := &pipeline.Ask{RT: rt, Deps: pipeline.Deps{Directory: dir, Facts: facts, Now: func() time.Time { return today }, DataVersion: "seed-test"}}
	a.Deps.Knowledge = rag.NewMemoryIndex("abc-infra", []rag.Document{{
		ID: "SOP-FUEL-01", Tenant: "abc-infra", Title: "Quy trình xử lý chênh lệch nhiên liệu", ACL: []string{"director"},
		Body: "1. Khi lượng xuất kho và lượng tiêu thụ đo được lệch quá dung sai, tổ trưởng kho lập biên bản đối soát trong vòng 24 giờ.",
	}})
	a.Deps.Model = &models.Extractive{}
	return a
}

func person(role, scope string) contracts.Principal {
	pr := contracts.Principal{ID: "u-" + role, TenantID: "abc-infra", Kind: contracts.PrincipalUser, Roles: []string{role}}
	if scope != "" {
		pr.Attributes = map[string]string{"entity_scope": scope}
	}
	return pr
}

func realTurn(t *testing.T, a *pipeline.Ask, pr contracts.Principal, text string) traces.Turn {
	t.Helper()
	out, err := a.Handle(context.Background(), pr, contracts.AskRequest{ConversationID: "c", MessageID: "m", Text: text})
	if err != nil {
		t.Fatalf("%q: %v", text, err)
	}
	return traces.Turn{Decision: out.Decision, Spans: out.Spans, TotalMs: out.TotalMs, UnaccountedMs: out.UnaccountedMs, Plan: out.Plan, Response: out.Response}
}

func TestBuocTuLuotThat(t *testing.T) {
	a := realAsk(t)
	director := person("director", "")
	for _, c := range []struct {
		name  string
		pr    contracts.Principal
		text  string
		route contracts.RouteKind
		check func(t *testing.T, steps []traces.Step)
	}{
		{"số liệu giải xong, nguồn lệch nhau giải bằng thẩm quyền", director, "EX-03 hôm nay được cấp bao nhiêu lít dầu?", contracts.RouteSQLQuery,
			func(t *testing.T, steps []traces.Step) {
				s6 := stepOf(t, steps, "S6")
				if !s6.Timed || s6.Status != traces.StatusDeviant || !strings.Contains(strings.Join(s6.Rejected, " "), "erp") {
					t.Errorf("S6 phải chạy, lệch, và kể giá trị ERP bị loại: %+v", s6)
				}
				if s8 := stepOf(t, steps, "S8"); !strings.Contains(s8.Reason, "không cần model") {
					t.Errorf("S8 route số liệu không gọi model: %+v", s8)
				}
				if s9 := stepOf(t, steps, "S9"); !s9.Timed || s9.Status != traces.StatusOK {
					t.Errorf("S9 phải chạy và đạt: %+v", s9)
				}
				if s10 := stepOf(t, steps, "S10"); s10.Status != traces.StatusOK {
					t.Errorf("S10 RESOLVED phải ok: %+v", s10)
				}
			}},
		{"máy 03 mơ hồ thì hỏi lại", director, "Máy 03 hôm nay được cấp bao nhiêu dầu?", contracts.RouteClarification,
			func(t *testing.T, steps []traces.Step) {
				s3 := stepOf(t, steps, "S3")
				if s3.Status != traces.StatusDeviant || !slices.ContainsFunc(s3.Output, func(x string) bool {
					return strings.HasPrefix(x, "?") && strings.Contains(x, "EX-03") && strings.Contains(x, "DZ-03")
				}) {
					t.Errorf("S3 phải lệch và kể hai ứng viên: %+v", s3)
				}
				for _, stage := range []string{"S6", "S7", "S9"} {
					if s := stepOf(t, steps, stage); s.Status != traces.StatusSkipped {
						t.Errorf("%s phải bị bỏ qua khi hỏi lại: %+v", stage, s)
					}
				}
			}},
		{"nguồn ngang thẩm quyền thì chuyển người", director, "DZ-03 hôm nay được cấp bao nhiêu dầu?", contracts.RouteHumanEscalation,
			func(t *testing.T, steps []traces.Step) {
				if s := stepOf(t, steps, "S10"); s.Status != traces.StatusDeviant || s.Decision != "outcome HUMAN_REVIEW_REQUIRED" {
					t.Errorf("S10: %+v", s)
				}
			}},
		{"kỹ sư ngoài phạm vi bị chặn", person("site_engineer", "PRJ-B"), "EX-03 hôm nay được cấp bao nhiêu dầu?", contracts.RouteReject,
			func(t *testing.T, steps []traces.Step) {
				if s := stepOf(t, steps, "S5"); s.Status != traces.StatusDeviant {
					t.Errorf("S5 từ chối phải lệch: %+v", s)
				}
				if s := stepOf(t, steps, "S10"); s.Decision != "outcome PERMISSION_DENIED" {
					t.Errorf("S10: %+v", s)
				}
				if s := stepOf(t, steps, "S6"); s.Status != traces.StatusSkipped {
					t.Errorf("bị chặn quyền thì không truy vấn nguồn: %+v", s)
				}
			}},
		{"câu hỏi tri thức qua model tất định", director, "Quy trình xử lý chênh lệch nhiên liệu là gì?", contracts.RouteRAG,
			func(t *testing.T, steps []traces.Step) {
				s7 := stepOf(t, steps, "S7")
				if len(s7.Evidence) == 0 || s7.Evidence[0].Source.DocumentID != "SOP-FUEL-01" {
					t.Errorf("S7 phải kèm nguồn tài liệu: %+v", s7.Evidence)
				}
				if s8 := stepOf(t, steps, "S8"); !s8.Timed || s8.Decision != "model diễn đạt" {
					t.Errorf("S8 có span generate và là model: %+v", s8)
				}
				if s6 := stepOf(t, steps, "S6"); !s6.Timed {
					t.Errorf("S6 truy xuất phải có thời gian (span retrieve): %+v", s6)
				}
			}},
		{"ngoài phạm vi: chưa có ý định", director, "Tối nay kèo nào thơm?", contracts.RouteReject,
			func(t *testing.T, steps []traces.Step) {
				if s := stepOf(t, steps, "S4"); s.Status != traces.StatusDeviant || s.Decision != "không nhận ra ý định" {
					t.Errorf("S4: %+v", s)
				}
			}},
	} {
		t.Run(c.name, func(t *testing.T) {
			turn := realTurn(t, a, c.pr, c.text)
			if turn.Plan.Route != c.route {
				t.Fatalf("route %s, muốn %s (đổi câu hỏi cho đúng ca)", turn.Plan.Route, c.route)
			}
			rec := traces.NewRecord(turn)
			if err := rec.Validate(); err != nil {
				t.Fatal(err)
			}
			if len(rec.Steps) != 11 {
				t.Fatalf("muốn 11 bước, được %d", len(rec.Steps))
			}
			var stepsMs, spansMs int64
			for i, s := range rec.Steps {
				if s.Stage != "S"+strconv.Itoa(i) || s.TraceID != turn.Response.TraceID || s.PackVersion != turn.Response.PackVersion || s.DataVersion != "seed-test" {
					t.Errorf("bước %d: %+v", i, s)
				}
				stepsMs += s.Ms
			}
			for _, sp := range turn.Spans {
				spansMs += sp.Ms
			}
			if stepsMs != spansMs { // mọi span của pipeline phải gắn được vào một bước — span lạ mới nghĩa là BuildSteps lệch pipeline
				t.Errorf("tổng thời gian các bước %d ms ≠ tổng span %d ms: có span chưa gắn vào bước nào (%v)", stepsMs, spansMs, turn.Spans)
			}
			var cited []string
			for _, ev := range stepOf(t, rec.Steps, "S7").Evidence {
				cited = append(cited, ev.ID)
			}
			var want []string
			for _, ci := range turn.Response.Citations {
				want = append(want, ci.EvidenceID)
			}
			if !slices.Equal(cited, want) {
				t.Errorf("bằng chứng của S7 %v ≠ trích dẫn của câu trả lời %v", cited, want)
			}
			if s10 := stepOf(t, rec.Steps, "S10"); len(s10.Output) != 1 || s10.Output[0] != turn.Response.Answer {
				t.Errorf("S10 phải mang đúng câu trả lời: %+v", s10)
			}
			c.check(t, rec.Steps)
			// qua làm sạch + JSON ra đúng dạng sẽ nằm trong DB, và băm của hai lần làm sạch như nhau
			clean, err := traces.SanitizeRecord(rec)
			if err != nil {
				t.Fatal(err)
			}
			again, _ := traces.SanitizeRecord(clean)
			d1, _ := clean.Digest()
			d2, _ := again.Digest()
			if d1 != d2 {
				t.Errorf("làm sạch hai lần ra bản khác")
			}
		})
	}
}

// So hai lượt THẬT: bước đầu tiên khác phải là bước mà nguyên nhân thật nằm ở đó.
func TestSoHaiLuotThat(t *testing.T) {
	a := realAsk(t)
	rec := func(pr contracts.Principal, text string) traces.Record {
		return traces.NewRecord(realTurn(t, a, pr, text))
	}
	director := person("director", "")
	q := "EX-03 hôm nay được cấp bao nhiêu dầu?"
	base := rec(director, q)

	// cùng câu, cùng người, hai lần chạy (trace id khác nhau): khớp hoàn toàn
	if d, ok := traces.FirstDifferentStep(base, rec(director, q)); ok {
		t.Errorf("hai lần chạy y nhau phải khớp, được %+v", d)
	}
	// cùng câu, người hỏi khác phạm vi: router đổi route (S5) — chuẩn hoá, thực thể, ý định vẫn như nhau
	d, ok := traces.FirstDifferentStep(base, rec(person("site_engineer", "PRJ-B"), q))
	if !ok || d.Stage != "S5" || d.Name != "route" || d.A != "SQL_QUERY" || d.B != "REJECT" {
		t.Errorf("khác quyền: được (%+v, %v), muốn S5 route SQL_QUERY→REJECT", d, ok)
	}
	// hai câu hỏi hai nghĩa nghiệp vụ khác nhau của cùng một máy: khác ở ý định (S4), thực thể vẫn như nhau
	d, ok = traces.FirstDifferentStep(base, rec(director, "EX-03 hôm nay đội báo bao nhiêu dầu?"))
	if !ok || d.Stage != "S4" || d.Name != "intent" {
		t.Errorf("khác ý định: được (%+v, %v), muốn S4 intent", d, ok)
	}
	// lượt lỗi thật (principal của tenant khác): S? — không có Decision từ luồng hỏi để so
	other := director
	other.TenantID = "xyz-sports"
	_, err := a.Handle(context.Background(), other, contracts.AskRequest{ConversationID: "c", MessageID: "m", Text: q})
	if contracts.FaultOf(err) != contracts.FaultPolicy {
		t.Fatalf("muốn lỗi policy, được %v", err)
	}
	errRec := traces.NewErrorRecord("trace-loi", other, q, today, traces.Versions{Pack: "p"}, err)
	if d, ok := traces.FirstDifferentStep(base, errRec); !ok || d.Stage != traces.StageUnknown || d.B != contracts.FaultPolicy {
		t.Errorf("so với lượt lỗi: được (%+v, %v)", d, ok)
	}
}

// Lượt thật mà người hỏi dán nhầm bí mật vào câu: bản ghi đã làm sạch không còn nó ở BẤT KỲ chỗ nào (câu hỏi, đầu vào các bước…).
func TestLuotThatDanBiMatKhongLotVaoVet(t *testing.T) {
	a := realAsk(t)
	secret := "tuyetmat" + strings.Repeat("7", 6)
	turn := realTurn(t, a, person("director", ""), "EX-03 hôm nay được cấp bao nhiêu dầu? mật khẩu: "+secret)
	if !strings.Contains(turn.Decision.Question, secret) {
		t.Fatal("tiền đề: lượt thật phải nhận câu hỏi nguyên văn")
	}
	clean, err := traces.SanitizeRecord(traces.NewRecord(turn))
	if err != nil {
		t.Fatal(err)
	}
	b, _ := json.Marshal(clean)
	if strings.Contains(string(b), secret) {
		t.Errorf("bí mật còn trong bản ghi vết: %s", b)
	}
	if !strings.Contains(clean.Decision.Question, "EX-03 hôm nay được cấp bao nhiêu dầu?") {
		t.Errorf("phần câu hỏi còn lại phải giữ: %q", clean.Decision.Question)
	}
}
