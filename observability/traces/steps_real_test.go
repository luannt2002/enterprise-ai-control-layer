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

func realAsk(t testing.TB) *pipeline.Ask {
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

func realTurn(t testing.TB, a *pipeline.Ask, pr contracts.Principal, text string) traces.Turn {
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
			for i, s := range rec.Steps {
				if s.Stage != "S"+strconv.Itoa(i) || s.TraceID != turn.Response.TraceID || s.PackVersion != turn.Response.PackVersion || s.DataVersion != "seed-test" {
					t.Errorf("bước %d: %+v", i, s)
				}
			}
			// Span của pipeline ↔ bước được đánh dấu đã đo: hai chiều, KHÔNG so tổng ms. Pipeline đo bằng đồng hồ thật nhưng chạy trong
			// bộ nhớ dưới 1 ms nên mọi span = 0 ms, phép so tổng thành 0 == 0 và không bắt được span chưa gắn vào bước nào (R1-05).
			for _, sp := range turn.Spans {
				stage, ok := traces.SpanStage(sp.Name)
				if !ok {
					t.Errorf("span %q của pipeline không gắn vào bước nào của BuildSteps (bảng stageSpans thiếu tên)", sp.Name)
					continue
				}
				if !stepOf(t, rec.Steps, stage).Timed {
					t.Errorf("span %q có mặt mà bước %s không được đánh dấu đã đo", sp.Name, stage)
				}
			}
			for _, s := range rec.Steps {
				if s.Timed && !slices.ContainsFunc(turn.Spans, func(sp traces.Span) bool { st, _ := traces.SpanStage(sp.Name); return st == s.Stage }) {
					t.Errorf("bước %s được đánh dấu đã đo mà lượt không có span nào của nó (thời gian bịa)", s.Stage)
				}
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

// unboundModel là model "bịa nguồn" (cùng ý với unboundModel ở tenants/construction): câu đúng chữ tài liệu nhưng claim trỏ tới bằng
// chứng không có trong ngữ cảnh — chặng kiểm (S9) phải chặn.
type unboundModel struct{}

func (unboundModel) Generate(_ context.Context, req models.Request) (contracts.Draft, error) {
	return contracts.Draft{Text: req.Evidence[0].Text, Claims: []contracts.Claim{{Text: req.Evidence[0].Text, EvidenceIDs: []string{"KHONG-CO#0"}}}}, nil
}

// firstDeviant là mã bước ĐẦU TIÊN không ở trạng thái ok / skipped — thứ mà người soát đọc vết sẽ nhìn trước (brief §9.3).
func firstDeviant(steps []traces.Step) string {
	for _, s := range steps {
		if s.Status == traces.StatusDeviant || s.Status == traces.StatusError {
			return s.Stage
		}
	}
	return ""
}

// R1-01. Hai lượt THẬT cùng câu: mốc A model tất định, mốc B model bịa nguồn. Chặng truy vấn y hệt ở hai bên; chỉ S9 trượt ở B, và
// pipeline đổi outcome sang SYSTEM_FAILURE, bỏ trích dẫn, thay câu — nên Diff báo cả bốn thứ khác. Bước đầu tiên khác phải là S9,
// không phải S6 (outcome) mà nguyên nhân thật chưa hề nằm ở đó; và bước lệch đầu tiên trong vết của B cũng là S9.
func TestSoHaiLuotThatKiemTruotChiS9(t *testing.T) {
	a := realAsk(t)
	q, director := "Quy trình xử lý chênh lệch nhiên liệu là gì?", person("director", "")
	good := traces.NewRecord(realTurn(t, a, director, q))
	a.Deps.Model = unboundModel{}
	bad := traces.NewRecord(realTurn(t, a, director, q))

	if good.Decision.Outcome != "RESOLVED" || bad.Decision.Outcome != "SYSTEM_FAILURE" || !strings.HasPrefix(bad.Decision.Verdict, "false|fallback") {
		t.Fatalf("tiền đề: A phải RESOLVED, B phải kiểm trượt — A %q, B %q %q", good.Decision.Outcome, bad.Decision.Outcome, bad.Decision.Verdict)
	}
	if got := traces.Diff(good.Decision, bad.Decision); !slices.Equal(got, []string{"outcome", "evidence", "verdict", "answer"}) {
		t.Fatalf("tiền đề: pipeline phải làm cả bốn thứ khác nhau cùng lúc, Diff = %v", got)
	}
	d, ok := traces.FirstDifferentStep(good, bad)
	if !ok || d.Stage != "S9" || d.Name != "verdict" {
		t.Errorf("kiểm trượt ở S9 mà hàm chỉ %+v (found=%v), muốn S9 verdict", d, ok)
	}
	if got := firstDeviant(bad.Steps); got != "S9" {
		t.Errorf("bước lệch đầu tiên trong vết của lượt kiểm trượt là %s, muốn S9 (S6 truy vấn y hệt lượt tốt)", got)
	}
	if got := stepOf(t, bad.Steps, "S6"); got.Status != traces.StatusOK {
		t.Errorf("chặng truy vấn của lượt kiểm trượt vẫn bình thường: %+v", got)
	}
}

// R1-02 + R1-03. Lượt THẬT kết thúc sớm: không gọi model (S8 không được ghi là "model diễn đạt") và truy vấn không ra dữ liệu thì bước
// lệch đầu tiên là chặng truy vấn S6, không phải S10 (chỉ là hệ quả). Mỗi ca nói rõ cách thoát sớm của pipeline ([CODE] ask.go
// answerMetric: result.Total == nil; answerKnowledge: asksFactQuantity, len(ev) == 0, phần tri thức của HYBRID rỗng).
func TestBuocTuLuotThatKetThucSom(t *testing.T) {
	a := realAsk(t)
	hybrid := "EX-03 hôm nay được cấp bao nhiêu dầu, và theo quy trình thì khi lệch phải làm gì?"
	for _, c := range []struct {
		name         string
		pr           contracts.Principal
		text         string
		route        contracts.RouteKind
		outcome      string
		wantGenerate bool   // pipeline có gọi model (có span generate)
		s6, s8       string // trạng thái muốn có
		first        string // bước lệch đầu tiên
		check        func(t *testing.T, steps []traces.Step)
	}{
		{"HYBRID: phần số liệu không có dữ liệu", person("director", ""), "EX-07 hôm nay được cấp bao nhiêu dầu, và theo quy trình thì khi lệch phải làm gì?",
			contracts.RouteHybrid, "INSUFFICIENT_EVIDENCE", false, traces.StatusDeviant, traces.StatusSkipped, "S6", nil},
		{"SQL_QUERY: không có dữ liệu", person("director", ""), "Máy 07 hôm nay được cấp bao nhiêu dầu?",
			contracts.RouteSQLQuery, "INSUFFICIENT_EVIDENCE", false, traces.StatusDeviant, traces.StatusOK,
			"S6", func(t *testing.T, steps []traces.Step) {
				if s8 := stepOf(t, steps, "S8"); !strings.Contains(s8.Decision, "mẫu câu") {
					t.Errorf("route số liệu dựng bằng mẫu: %+v", s8)
				}
			}},
		{"RAG: câu đòi lượng nghiệp vụ, thoát trước truy xuất", person("director", ""), "Theo quy trình thì một máy xúc tiêu thụ bao nhiêu lít dầu mỗi giờ?",
			contracts.RouteRAG, "INSUFFICIENT_EVIDENCE", false, traces.StatusSkipped, traces.StatusSkipped, "S10",
			func(t *testing.T, steps []traces.Step) {
				if s7 := stepOf(t, steps, "S7"); s7.Status != traces.StatusSkipped {
					t.Errorf("không truy xuất thì không có ngữ cảnh: %+v", s7)
				}
			}},
		{"RAG: truy xuất rỗng (người hỏi không được đọc tài liệu)", person("site_engineer", "PRJ-A"), "Quy trình xử lý chênh lệch nhiên liệu là gì?",
			contracts.RouteRAG, "INSUFFICIENT_EVIDENCE", false, traces.StatusDeviant, traces.StatusSkipped, "S6", nil},
		{"HYBRID: số liệu xong, phần tri thức rỗng", person("site_engineer", "PRJ-A"), hybrid,
			contracts.RouteHybrid, "RESOLVED", false, traces.StatusDeviant, traces.StatusSkipped, "S6",
			func(t *testing.T, steps []traces.Step) {
				if s6 := stepOf(t, steps, "S6"); !strings.Contains(strings.Join(s6.Output, " "), "tri thức") {
					t.Errorf("S6 phải nói phần tri thức không truy xuất được đoạn nào: %+v", s6.Output)
				}
			}},
		// đối chứng: model THẬT có được gọi thì S8 vẫn là "model diễn đạt" và có thời gian (sửa không được bỏ qua oan)
		{"HYBRID: model được gọi", person("director", ""), hybrid, contracts.RouteHybrid, "RESOLVED", true, traces.StatusDeviant, traces.StatusOK, "S6", nil},
		{"RAG: model được gọi", person("director", ""), "Quy trình xử lý chênh lệch nhiên liệu là gì?", contracts.RouteRAG, "RESOLVED", true, traces.StatusOK, traces.StatusOK, "", nil},
	} {
		t.Run(c.name, func(t *testing.T) {
			turn := realTurn(t, a, c.pr, c.text)
			if turn.Plan.Route != c.route || turn.Decision.Outcome != c.outcome {
				t.Fatalf("tiền đề: route %s outcome %s, muốn %s %s (đổi câu hỏi cho đúng ca)", turn.Plan.Route, turn.Decision.Outcome, c.route, c.outcome)
			}
			generated := slices.ContainsFunc(turn.Spans, func(sp traces.Span) bool { return sp.Name == "generate" })
			if generated != c.wantGenerate {
				t.Fatalf("tiền đề: span generate có = %v, muốn %v (spans %v)", generated, c.wantGenerate, turn.Spans)
			}
			steps := traces.NewRecord(turn).Steps
			s6, s8 := stepOf(t, steps, "S6"), stepOf(t, steps, "S8")
			if s6.Status != c.s6 {
				t.Errorf("S6 %q, muốn %q: %+v", s6.Status, c.s6, s6)
			}
			if s8.Status != c.s8 {
				t.Errorf("S8 %q, muốn %q: %+v", s8.Status, c.s8, s8)
			}
			if c.wantGenerate && (s8.Decision != "model diễn đạt" || !s8.Timed) {
				t.Errorf("model đã được gọi: S8 phải là model diễn đạt, có thời gian: %+v", s8)
			}
			if !c.wantGenerate && route(turn).NeedsModel() && (s8.Decision == "model diễn đạt" || !strings.Contains(s8.Reason, "không có span generate")) {
				t.Errorf("model chưa được gọi mà S8 vẫn kể là có (vết nói dối về chặng MODEL): %+v", s8)
			}
			if got := firstDeviant(steps); got != c.first {
				t.Errorf("bước lệch đầu tiên %q, muốn %q", got, c.first)
			}
			if c.check != nil {
				c.check(t, steps)
			}
		})
	}
}

func route(t traces.Turn) contracts.RouteKind { return contracts.RouteKind(t.Decision.Route) }

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

// BenchmarkDungBanGhi đo chi phí CPU một luồng của việc dựng bản ghi vết cho MỘT lượt thật (NewRecord + SanitizeRecord + Digest) và
// kích thước JSON của nó — con số để chốt chính sách lưu giữ (R1-06). Chạy: go test -run '^$' -bench DungBanGhi -benchmem ./observability/traces/
func BenchmarkDungBanGhi(b *testing.B) {
	turn := realTurn(b, realAsk(b), person("director", ""), "EX-03 hôm nay được cấp bao nhiêu lít dầu?")
	var size int
	b.ReportAllocs()
	for b.Loop() {
		clean, err := traces.SanitizeRecord(traces.NewRecord(turn))
		if err != nil {
			b.Fatal(err)
		}
		if _, err := clean.Digest(); err != nil {
			b.Fatal(err)
		}
		if size == 0 {
			raw, _ := json.Marshal(clean)
			size = len(raw)
		}
	}
	b.ReportMetric(float64(size), "byte-json/lượt")
}
