// Package pipeline điều phối các lối vào của EAIS. File này là luồng HỎI (S0–S10).
//
// Ba luật cứng (docs/01-architecture.md §4):
//  1. Route không cần diễn đạt thì không gọi model — trả lời bằng mẫu câu của tenant.
//  2. Model không bao giờ là nguồn của con số — số đến từ nguồn sự thật, có bằng chứng.
//  3. Kiểm thất bại thì không giao câu đó.
package pipeline

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"runtime/debug"
	"slices"
	"strconv"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/google/uuid"
	"github.com/luannt2002/enterprise-ai-control-layer/core/contracts"
	"github.com/luannt2002/enterprise-ai-control-layer/core/decimal"
	"github.com/luannt2002/enterprise-ai-control-layer/core/tenant"
	"github.com/luannt2002/enterprise-ai-control-layer/data/entityresolution"
	"github.com/luannt2002/enterprise-ai-control-layer/data/normalization"
	"github.com/luannt2002/enterprise-ai-control-layer/data/sourceoftruth"
	"github.com/luannt2002/enterprise-ai-control-layer/intelligence/intent"
	"github.com/luannt2002/enterprise-ai-control-layer/intelligence/models"
	"github.com/luannt2002/enterprise-ai-control-layer/intelligence/router"
	"github.com/luannt2002/enterprise-ai-control-layer/observability/traces"
	"github.com/luannt2002/enterprise-ai-control-layer/reliability/validation"
)

// MaxTextRunes là trần độ dài câu hỏi. Vượt trần là lỗi đầu vào, không phải lỗi hệ thống.
const MaxTextRunes = 2000

// knowledgeK là số đoạn tài liệu tối đa đưa cho model trong một lượt.
const knowledgeK = 3

// KnowledgeRetriever là hợp đồng duy nhất luồng hỏi biết về RAG (CLAUDE.md §12): đưa câu hỏi, nhận
// bằng chứng người hỏi ĐƯỢC đọc. Bên trong dùng toàn văn, vector hay cách khác là việc của
// capability tri thức.
type KnowledgeRetriever interface {
	Retrieve(ctx context.Context, pr contracts.Principal, query string, k int) ([]contracts.Evidence, error)
}

// IntentFallback được hỏi SAU luật S4 và chỉ được đổi kết quả của luật sang ĐÚNG MỘT ý định tri thức (route RAG) mà nó khai; quyền,
// phạm vi, khe vẫn do router (S5) quyết. Dùng để đo các phương án S4 (docs/ROADMAP-RAG.md §10.2): dò chỉ mục tri thức (tất định) hay
// model phân loại. Fallback trả use = true để đổi, kèm độ tin cậy và lý do ghi vào vết; lỗi thì luồng giữ kết quả của luật.
type IntentFallback interface {
	Intent() string
	Fallback(ctx context.Context, pr contracts.Principal, in contracts.NormalizedInput, cls intent.Result) (use bool, conf float64, why string, err error)
}

// Deps là các nguồn dữ liệu của MỘT tenant mà luồng hỏi cần.
type Deps struct {
	Directory entityresolution.Directory
	Facts     sourceoftruth.Store
	// Knowledge, Model: chỉ mục tài liệu và gateway model cho route RAG / HYBRID. Thiếu thì route
	// đó là lỗi cấu hình, không âm thầm trả lời.
	Knowledge KnowledgeRetriever
	Model     models.Gateway
	// IntentFallback: dự phòng S4 (thử nghiệm, nil = tắt — hành vi như cũ). Xem fallbackIntent.
	IntentFallback IntentFallback
	// DataVersion định danh bộ dữ liệu đang phục vụ (ví dụ băm manifest của lần nạp) — ghi vào
	// bản ghi quyết định; phát lại chỉ hợp lệ trên đúng bộ dữ liệu đó.
	DataVersion string
	// Now trả thời điểm hiện tại; nil = đồng hồ của tenant. Test truyền đồng hồ cố định.
	Now func() time.Time
}

// Ask là luồng hỏi của một tenant.
type Ask struct {
	RT   *tenant.Runtime
	Deps Deps
	// noWhy tắt phần GHI giải thích của vết (why.go) ở Handle và answerMetric — chỉ test trong gói đặt, để so route / outcome / câu trả lời
	// giữa có vết và không có vết (C11, lệnh ca chiều 23/10). answerKnowledge vẫn ghi (nhánh RAG đang sửa hàm đó).
	noWhy bool
}

// Outcome là kết quả một lượt kèm vết để ghi log và lưu `interactions`.
type Outcome struct {
	Response contracts.AskResponse
	Plan     contracts.RoutePlan
	// Decision là bản ghi quyết định của lượt (G6): đủ để phát lại bằng Replay.
	Decision traces.Decision
	Spans    []traces.Span
	TotalMs  int64
	// UnaccountedMs: thời gian không nằm trong chặng nào — lớn là đồng hồ đang nói dối.
	UnaccountedMs int64
}

// Handle chạy một lượt hỏi. Lỗi trả về luôn mang nhãn fault.
func (a *Ask) Handle(ctx context.Context, pr contracts.Principal, req contracts.AskRequest) (Outcome, error) {
	tr := traces.New(nil)
	now := a.now()
	var why traces.Why // giải thích từng chặng của vết (why.go): chỉ ghi lại đã quyết gì và vì sao, không điều khiển gì

	// S0 — kiểm đầu vào
	if err := validateRequest(req); err != nil {
		return Outcome{}, err
	}
	// S1 — tenant lấy từ principal đã xác thực, không từ thân yêu cầu; principal của tenant
	// khác không được chạm vào runtime (pack, dữ liệu) của tenant này.
	if string(pr.TenantID) != a.RT.Meta.Tenant {
		return Outcome{}, contracts.Errorf(contracts.FaultPolicy, "principal không thuộc tenant của runtime")
	}
	// Từ đây mọi truy vấn mang tenant của Principal: tầng lưu trữ đặt GUC cho RLS từ giá trị này.
	ctx = contracts.WithTenant(ctx, pr.TenantID)
	// S2 — chuẩn hoá
	in, _ := traces.Measure(tr, "normalize", func() (contracts.NormalizedInput, error) {
		return a.RT.Normalizer.Normalize(req.Text, normalization.Options{Now: now}), nil
	})
	// S3 — nhận dạng thực thể
	res, err := traces.Measure(tr, "resolve", func() (contracts.Resolution, error) {
		return entityresolution.Resolve(ctx, a.Deps.Directory, in)
	})
	if err != nil {
		return Outcome{}, err
	}
	// S4 — ý định
	cls, _ := traces.Measure(tr, "intent", func() (intent.Result, error) {
		return a.RT.Intents.Classify(in, res, now), nil
	})
	if f := a.Deps.IntentFallback; f != nil {
		if cls, err = a.fallbackIntent(ctx, now, pr, in, cls, f); err != nil {
			return Outcome{}, err
		}
	}
	// Hỏi lại nghĩa số liệu chỉ kể ý định người hỏi có quyền; không còn ý định nào thì như câu không khớp (từ chối như cũ) — không lộ
	// nhãn ý định cho người không có capability (CLAUDE.md §21)
	cls.NeedMeaning = slices.DeleteFunc(cls.NeedMeaning, func(n string) bool {
		d, _ := a.RT.Intents.Def(n)
		return d.Capability != "" && !a.RT.Policy.Allows(pr, d.Capability)
	})
	// S5 — route
	var lineageErr error
	lineage := func(e contracts.EntityRef) []string {
		keys, err := entityresolution.Lineage(ctx, a.Deps.Directory, e.ID)
		if err != nil {
			lineageErr = err
		}
		return keys
	}
	plan, _ := traces.Measure(tr, "route", func() (contracts.RoutePlan, error) {
		return router.Route(a.RT.Intents, cls, in, pr, a.RT.Policy, a.RT.Router, lineage), nil
	})
	if lineageErr != nil {
		// Không đọc được danh mục thì không kết luận được quyền — lỗi hạ tầng, không mở toang.
		return Outcome{}, lineageErr
	}
	routed := plan // kế hoạch NGUYÊN VĂN của router, trước khi luồng hỏi có thể ghi đè (chỉ để giải thích vết)
	// Câu hỏi lại giải thích thêm khi câu nêu nhiều mốc thời gian / ngày hỏng, và khi câu xác nhận nêu nhiều con số
	// khác nhau: không xác nhận con số nào thay người hỏi (P5).
	notes := a.unclearNotes(cls.Unclear, now.Location())
	names := mentionsOf(res)
	claims := a.confirmClaims(plan, in, names)
	rival, doubt := len(claims.values) > 1, len(claims.doubts) > 0
	if rival || doubt {
		plan.Route, plan.Template, plan.Outcome, plan.Steps = contracts.RouteClarification, "clarify_missing", contracts.OutcomeClarificationRequired, nil
	}
	if rival {
		plan.Intent.Missing = append(slices.Clone(plan.Intent.Missing), "claimed")
		plan.Reason = "câu xác nhận nêu nhiều con số khác nhau"
		notes = append(notes, "Câu hỏi nêu nhiều con số khác nhau; mỗi lượt chỉ xác nhận được một con số.")
	}
	if doubt {
		// mốc thời gian người hỏi viết bằng chữ mà bộ chuẩn hoá không đọc được (hay không khớp kỳ đã đọc): kỳ của câu có thể KHÁC kỳ họ nói,
		// nên không phán quyết "Đúng." / "Không." cho một kỳ khác (P5)
		plan.Intent.Missing = append(slices.Clone(plan.Intent.Missing), "period")
		plan.Reason = "câu xác nhận nêu mốc thời gian không đọc được"
		notes = append(notes, "Chưa đọc được mốc thời gian: "+strings.Join(claims.doubts, ", ")+". Bạn cho biết ngày hoặc khoảng ngày cụ thể.")
	}
	group := len(claims.groups) > 0
	if group {
		// "bể 2 500 lít": 2.500 L, hay bể 2 với 500 L? "lu 2 850 lít": lu 2 với 850 L, hay lu với 2.850 L? Hệ không phán quyết theo một cách đọc
		// tự chọn, cũng không bỏ lượng im lặng (P5; D-OP-9)
		plan.Route, plan.Outcome = contracts.RouteClarification, contracts.OutcomeClarificationRequired
		plan.Template, plan.Steps = "clarify_missing", nil
		if !slices.Contains(plan.Intent.Missing, "claimed") { // câu nêu nhiều con số khác nhau đã khai khe này
			plan.Intent.Missing = slices.Concat(plan.Intent.Missing, []string{"claimed"})
		}
		plan.Reason = "câu xác nhận có nhóm số chưa phân định"
		notes = append(notes, "Chưa rõ cách đọc \""+strings.Join(claims.groups, "\", \"")+"\": một lượng, hay số hiệu đứng trước một lượng? Bạn viết lại con số liền (không dấu cách) hoặc có dấu chấm ngăn nghìn.")
	}

	resp := contracts.AskResponse{
		Route: plan.Route, Outcome: plan.Outcome, Intent: plan.Intent.Name, PackVersion: a.RT.PackVersion, TraceID: uuid.NewString(),
	}
	vars := map[string]string{"reason": plan.Reason}
	var ci clarifyInfo // cách dựng danh sách ứng viên của câu hỏi lại (chỉ để giải thích vết)

	// Từ đây chỉ dùng plan.Intent: ý định SAU khi router đã kiểm quyền và gỡ mơ hồ.
	switch plan.Route {
	case contracts.RouteClarification:
		asked := append(slices.Clone(cls.Candidates), cls.Unclear.Entities...)
		visible, hidden := a.visibleCandidates(ctx, pr, asked)
		ci = clarifyInfo{used: true, before: asked, shown: visible, hidden: hidden}
		if rival || doubt {
			// hỏi lại về CON SỐ hay MỐC THỜI GIAN: thực thể đã rõ (router đã gỡ bí danh mơ hồ), nên không kể ra máy ứng viên nào
			visible = nil
			ci.shown, ci.suppressed = nil, "câu hỏi lại về con số hay mốc thời gian, thực thể đã rõ (router đã gỡ bí danh mơ hồ)"
		}
		if group {
			visible, ci.shown, ci.suppressed = nil, nil, "câu hỏi lại về cách đọc nhóm số, thực thể đã rõ (router đã gỡ bí danh mơ hồ)"
		}
		missing, cands := a.clarificationDetails(plan.Intent, visible, notes)
		resp.Clarification = &contracts.Clarification{Missing: plan.Intent.Missing, Candidates: visible}
		vars["missing"], vars["candidates"], vars["options"] = missing, cands, a.intentOptions(slices.Concat(cls.Ambiguous, cls.NeedTerm, cls.NeedMeaning))
		resp.Answer = a.render(plan.Template, vars)
		resp.Verdict = contracts.Verdict{Pass: true, Action: contracts.ActionClarify}
		resp.Clarification.Question = resp.Answer
	case contracts.RouteReject, contracts.RouteHumanEscalation:
		resp.Answer = a.render(plan.Template, vars)
		resp.Verdict = contracts.Verdict{Pass: true, Action: contracts.ActionAccept}
	case contracts.RouteSQLQuery, contracts.RouteCalculation, contracts.RouteHybrid:
		if err := a.answerMetric(ctx, tr, now, &plan, in, names, plan.Intent, &resp, vars, &why); err != nil {
			return Outcome{}, err
		}
		// HYBRID: số liệu từ fact (đã kiểm) + phần quy trình từ tài liệu. Phần số liệu không ra
		// được câu trả lời thì không nối thêm tài liệu vào.
		if plan.Route == contracts.RouteHybrid && resp.Outcome == contracts.OutcomeResolved {
			kdef, _ := a.RT.Intents.Def(cls.Knowledge)
			if err := a.answerKnowledge(ctx, tr, now, pr, in, kdef.Template, &resp, true, &why); err != nil {
				return Outcome{}, err
			}
		}
	case contracts.RouteRAG:
		if err := a.answerKnowledge(ctx, tr, now, pr, in, plan.Template, &resp, false, &why); err != nil {
			return Outcome{}, err
		}
	default:
		// DIRECT_LLM, FULL_TEXT_SEARCH, TOOL, WORKFLOW, AGENT cần capability chưa nối vào luồng này. Báo đúng là
		// lỗi cấu hình/mã thay vì giả vờ trả lời.
		return Outcome{}, contracts.Errorf(contracts.FaultCode, fmt.Sprintf("route %s chưa có capability trong luồng hỏi", plan.Route))
	}
	if !a.noWhy {
		why.Normalize = explainNormalize(in)
		// tên và loại thực thể chỉ kể cho người có phạm vi dữ liệu không hạn chế (vai phạm vi "all"): vết do chính người hỏi đọc không được lộ
		// chúng cho thực thể ngoài phạm vi, và không tra danh mục thêm lần nào để biết thực thể nào ngoài phạm vi
		why.Resolve = explainResolve(in, res, routed.Narrowed, ci, a.RT.Policy.InScope(pr, nil))
		why.Intent = explainIntent(cls, ci)
		why.Route = a.explainRoute(routed, plan)
	}
	// Câu mang bí mật mà bộ làm sạch nhận ra — ở câu gốc HOẶC ở câu đã gấp ký tự — thì không trích chữ nào của câu vào giải thích. Phải nhìn cả hai: mẫu che
	// neo bằng `\b` (Go chỉ coi chữ, số, gạch dưới ASCII là "chữ"), nên token đứng ngay sau một chữ x toàn chiều rộng (hay ký hiệu lít, số hay gạch dưới toàn chiều
	// rộng) có ranh giới từ ở câu gốc mà mất ranh giới ở in.Raw, nơi ký tự đó đã gấp thành chữ ASCII dính liền; ngược lại khoá viết bằng chữ toàn chiều rộng
	// chỉ nhận ra ở in.Raw. Chữ trích từ in.Raw đi vào vết bền, nơi SanitizeRecord chỉ thấy dạng đã gấp nên cũng không che được.
	if traces.Sanitize(in.Raw) != in.Raw || traces.Sanitize(req.Text) != req.Text {
		why = why.WithholdQuoted()
	}
	// Vết do chính người hỏi đọc lại: với người không toàn quyền, khoá / tên thực thể ngoài phạm vi của họ được che ở MỘT chỗ cho mọi bề mặt —
	// mọi dòng giải thích và Decision.Entities / Slots (C9, scope_redact.go). Che TRƯỚC Compact: cắt dòng có thể cắt dở một khoá.
	red := a.newScopeRedactor(ctx, pr, req.Text, entityRefs(res, cls, ci, routed, plan)...)
	d := a.decision(pr, now, req.Text, in, res, plan.Intent, resp)
	red.decision(&d)
	if !a.noWhy {
		d.Why = finalWhy(why, red)
	}
	return Outcome{
		Response: resp, Plan: plan, Decision: d,
		Spans: tr.Spans(), TotalMs: tr.TotalMs(), UnaccountedMs: tr.Unaccounted(),
	}, nil
}

// codeVersion là phiên bản mã chạy lượt này: bản sửa VCS do `go build` ghi vào binary (kèm
// "+dirty" khi cây có sửa chưa commit); không có thì phiên bản module, thường là "(devel)".
var codeVersion = func() string {
	bi, ok := debug.ReadBuildInfo()
	if !ok {
		return "(unknown)"
	}
	rev, dirty := "", false
	for _, s := range bi.Settings {
		switch s.Key {
		case "vcs.revision":
			rev = s.Value
		case "vcs.modified":
			dirty = s.Value == "true"
		}
	}
	if rev == "" {
		return bi.Main.Version
	}
	if dirty {
		rev += "+dirty"
	}
	return rev
}()

// decision ghi lại quyết định của lượt ở dạng phẳng (traces.Decision).
func (a *Ask) decision(pr contracts.Principal, now time.Time, question string, in contracts.NormalizedInput,
	res contracts.Resolution, it contracts.Intent, resp contracts.AskResponse) traces.Decision {

	d := traces.Decision{
		TraceID: resp.TraceID, Tenant: string(pr.TenantID), Principal: pr.ID, Kind: string(pr.Kind), Roles: pr.Roles,
		Attributes: pr.Attributes, Now: now, PackVersion: a.RT.PackVersion, CodeVersion: codeVersion,
		DataVersion: a.Deps.DataVersion, Question: question,
		Intent: it.Name, Route: string(resp.Route), Outcome: string(resp.Outcome), Answer: resp.Answer,
	}
	for _, sp := range in.Spans {
		d.Normalized = append(d.Normalized, fmt.Sprintf("%s:%s=%s", sp.Kind, sp.Raw, valueText(sp.Value)))
	}
	for _, raw := range in.InvalidDates { // ngày hỏng không có span, nhưng là một phần của vết chuẩn hoá (P8)
		d.Normalized = append(d.Normalized, fmt.Sprintf("%s:%s=invalid", contracts.ValueDate, raw))
	}
	for _, e := range res.Entities {
		d.Entities = append(d.Entities, e.ID)
	}
	for _, m := range res.Ambiguous {
		ids := make([]string, 0, len(m.Candidates))
		for _, c := range m.Candidates {
			ids = append(ids, c.ID)
		}
		d.Entities = append(d.Entities, "?"+strings.Join(ids, "|"))
	}
	names := make([]string, 0, len(it.Slots))
	for n := range it.Slots {
		names = append(names, n)
	}
	slices.Sort(names)
	for _, n := range names {
		s, v := it.Slots[n], it.Slots[n].Text
		switch {
		case s.Entity != nil:
			v = s.Entity.ID
		case s.Value != nil:
			v = valueText(*s.Value)
		}
		d.Slots = append(d.Slots, n+"="+v)
	}
	for _, m := range it.Missing {
		d.Slots = append(d.Slots, m+"=?")
	}
	for _, c := range resp.Citations {
		d.Evidence = append(d.Evidence, c.EvidenceID)
	}
	d.Verdict = fmt.Sprintf("%t|%s", resp.Verdict.Pass, resp.Verdict.Action)
	for i, v := range resp.Verdict.Violations {
		sep := ","
		if i == 0 {
			sep = "|"
		}
		d.Verdict += sep + v.Code
	}
	return d
}

func valueText(v contracts.Value) string {
	s := v.Number + v.Unit + v.Text
	if v.Approx {
		s += "~"
	}
	return s
}

// Replay chạy lại một lượt từ bản ghi quyết định — cùng câu hỏi, cùng người hỏi, cùng đồng hồ —
// và trả tên các chặng ra khác bản ghi (rỗng = tái lập đúng). Khác phiên bản pack hoặc dữ liệu thì
// không phát lại (lỗi đầu vào): khác đầu vào mà ra khác là đúng, không phải lệch.
func (a *Ask) Replay(ctx context.Context, d traces.Decision) ([]string, error) {
	if d.PackVersion != a.RT.PackVersion || d.DataVersion != a.Deps.DataVersion {
		return nil, contracts.Errorf(contracts.FaultInput, "phát lại cần đúng phiên bản pack và dữ liệu đã ghi")
	}
	re := *a
	re.Deps.Now = func() time.Time { return d.Now }
	pr := contracts.Principal{ID: d.Principal, TenantID: contracts.TenantID(d.Tenant), Kind: contracts.PrincipalKind(d.Kind),
		Roles: d.Roles, Attributes: d.Attributes}
	out, err := re.Handle(ctx, pr, contracts.AskRequest{ConversationID: "replay", MessageID: d.TraceID, Text: d.Question})
	if err != nil {
		return nil, err
	}
	return traces.Diff(d, out.Decision), nil
}

// answerMetric chạy S6–S9 cho câu hỏi số liệu: truy vấn nguồn sự thật, dựng câu bằng mẫu,
// rồi kiểm số trong câu với bằng chứng.
func (a *Ask) answerMetric(ctx context.Context, tr *traces.Trace, now time.Time, plan *contracts.RoutePlan,
	in contracts.NormalizedInput, names []mention, it contracts.Intent, resp *contracts.AskResponse, vars map[string]string, why *traces.Why) error {

	entitySlot, metric := firstEntitySlot(it), it.Slots["metric"].Text
	ids, err := entityresolution.Expand(ctx, a.Deps.Directory, entitySlot.Entity.ID)
	if err != nil {
		return err
	}
	from, to, err := periodBounds(it, now)
	if err != nil {
		return err
	}

	// S6 — thực thi
	result, err := traces.Measure(tr, "execute", func() (sourceoftruth.Result, error) {
		if cm := it.Slots["compare_metric"].Text; cm != "" { // câu so sánh hai nghĩa
			return a.RT.Metrics.Compare(ctx, a.Deps.Facts, metric, cm, ids, from, to)
		}
		if _, derived := a.RT.Metrics.Derived(metric); derived { // CALCULATION, hoặc phần số liệu của HYBRID
			return a.RT.Metrics.Derive(ctx, a.Deps.Facts, metric, ids, from, to)
		}
		return a.RT.Metrics.Query(ctx, a.Deps.Facts, sourceoftruth.MetricQuery{Metric: metric, EntityIDs: ids, From: from, To: to})
	})
	if err != nil {
		return err
	}
	// S7 — gói bằng chứng
	pkg := contracts.ContextPackage{Evidence: result.Evidence, Conflicts: result.Conflicts}
	resp.Conflicts = result.Conflicts
	prefix := ""
	if plan.Route == contracts.RouteHybrid {
		prefix = "số liệu: "
	}
	// S7 ghi lúc trả về, khi gói ngữ cảnh đã là gói CUỐI (có thể đã thêm số của người hỏi và các chỉ số khác cùng kỳ)
	if !a.noWhy {
		defer func() { a.noteMetricContext(why, prefix, result, pkg, entitySlot.Entity.Key, len(ids)) }()
	}
	for _, e := range result.Evidence {
		resp.Citations = append(resp.Citations, contracts.Citation{EvidenceID: e.ID, Source: e.Source})
	}

	vars["metric_label"] = a.metricLabel(metric)
	vars["entity_name"], vars["entity_key"] = entitySlot.Entity.Name, entitySlot.Entity.Key
	vars["period"] = displayPeriod(from, to)
	if def, ok := a.RT.Intents.Def(it.Name); ok && def.Assumption != "" {
		// trả lời theo ý định chung: MỌI câu trả lời (có số, chưa có dữ liệu, chờ đối soát) nói rõ đã hiểu câu hỏi là gì (D-OP-15 (i))
		vars["assumption_note"] = " " + def.Assumption
	}

	switch {
	case result.NeedsReview:
		// Nguồn ngang thẩm quyền nói khác nhau: không đưa ra con số nào như thể là sự thật.
		plan.Route, plan.Template = contracts.RouteHumanEscalation, "needs_review"
		resp.Route = plan.Route
		var reasons []string
		for _, c := range result.Conflicts {
			if c.NeedsReview {
				reasons = append(reasons, c.Reason)
			}
		}
		vars["reason"] = strings.Join(reasons, "; ")
		// lý do THẬT của việc ghi đè route mà vết S5 kể (C8, lệnh ca chiều 23/10) — không giữ lý do của router ("ý định … khớp luật")
		plan.Reason = "nguồn ngang thẩm quyền lệch quá dung sai, cần người duyệt: " + vars["reason"]
		resp.Answer = a.render("needs_review", vars)
		resp.Verdict = contracts.Verdict{Pass: true, Action: contracts.ActionEscalate}
		resp.Outcome = contracts.OutcomeHumanReviewRequired
		return nil
	case result.Total == nil:
		resp.Answer = a.render("no_data", vars)
		resp.Verdict = contracts.Verdict{Pass: true, Action: contracts.ActionAccept}
		resp.Outcome = contracts.OutcomeInsufficientEvidence
		return nil
	}

	vars["value"] = displayNumber(result.Total.Number)
	vars["unit"] = a.unitLabel(result.Total.Unit)
	if result.Approx {
		vars["approx_note"] = a.RT.Templates["approx_note"]
	}
	// Claim đủ bộ cho MỌI con số sẽ có trong câu (G5): số chính trỏ về bằng chứng phép tính của chính
	// truy vấn này — phép tính chạy trên tập thực thể của chủ thể được hỏi.
	subject, period := entitySlot.Entity.ID, sourceoftruth.PeriodKey(from, to)
	stampSubject(pkg.Evidence, subject)
	claims := []contracts.Claim{{Subject: subject, Metric: result.Metric, Period: period, Value: result.Total,
		Source: "calculation", EvidenceIDs: []string{"calc:" + result.Metric}}}
	compare := it.Slots["compare_metric"].Text
	if compare != "" {
		// hai vế hiện trong câu: mỗi vế một claim, trỏ về bằng chứng phép tính của chính vế đó
		vars["left_label"], vars["right_label"] = a.metricLabel(metric), lowerFirst(a.metricLabel(compare))
		for _, side := range []struct{ id, key string }{{metric, "left_value"}, {compare, "right_value"}} {
			for _, e := range result.Evidence {
				if e.ID == "calc:"+side.id && len(e.Values) == 1 {
					v := e.Values[0]
					vars[side.key] = displayNumber(v.Number)
					claims = append(claims, contracts.Claim{Subject: subject, Metric: side.id, Period: period, Value: &v,
						Source: "calculation", EvidenceIDs: []string{e.ID}})
				}
			}
		}
	}
	items, itemClaims := a.conflictItems(result.Conflicts)
	claims = append(claims, itemClaims...)
	if len(items) > 0 {
		// Kê giá trị của nguồn KHÔNG được chọn, kèm nhãn — người đọc thấy ERP nói gì, không chỉ
		// biết "có lệch". Nhiều chỗ lệch (tổng theo dự án) thì kê 3 rồi gộp phần còn lại.
		shown := items
		if len(shown) > 3 {
			shown = items[:3]
			vars["count"] = strconv.Itoa(len(items) - 3)
			vars["conflict_more"] = a.fragment("conflict_more", vars)
			count := contracts.Value{Number: vars["count"]}
			pkg.Evidence = append(pkg.Evidence, contracts.Evidence{ID: "count:conflicts", Kind: contracts.EvidenceCalculation,
				Text: "số chỗ lệch không kê ra", Values: []contracts.Value{count}, Source: contracts.SourceRef{SourceType: "calculation"},
				Subject: subject, Metric: "conflict_count", Period: period})
			claims = append(claims, contracts.Claim{Subject: subject, Metric: "conflict_count", Period: period, Value: &count,
				Source: "calculation", EvidenceIDs: []string{"count:conflicts"}})
		}
		vars["conflicts"] = strings.Join(shown, "; ")
		vars["conflict_note"] = a.fragment("conflict_note", vars)
	}
	ignore := []string{entitySlot.Entity.Name, entitySlot.Entity.Key}
	tmpl := plan.Template
	withheld := "" // chiều của claim chưa giải được: câu xác nhận chỉ được trả số, không phán quyết (ADR-0009)
	claimed, confirm := a.claimedValue(in, names, metric)
	if confirm && compare == "" {
		def, _ := a.RT.Metrics.Metric(metric)
		subj, err := a.subjectOf(ctx, entitySlot.Entity.ID)
		if err != nil {
			return err
		}
		if withheld = a.verdictGap(in, names, it, def, claimed, subj, result.UsedSources); withheld != "" {
			plan.Reason = "câu xác nhận chưa giải đủ chiều: " + withheld
		}
	}
	if confirm && compare == "" && withheld == "" {
		// Câu xác nhận: so BẰNG TUYỆT ĐỐI với số có thẩm quyền, rồi kê các nghĩa khác cùng kỳ
		// (mỗi số kèm nhãn của nó) — "4k" có thể là số báo cáo, không phải số xuất kho.
		tmpl = "confirm_no"
		if c, err := decimal.Cmp(claimed.Number, result.Total.Number); err == nil && c == 0 {
			tmpl = "confirm_yes"
		}
		vars["claimed"] = displayNumber(claimed.Number) + " " + a.unitLabel(claimed.Unit)
		ignore = append(ignore, vars["claimed"]) // số của người hỏi, không phải claim của hệ thống
		// Số của người hỏi vẫn có claim — nguồn là chính câu hỏi, không phải dữ liệu.
		pkg.Evidence = append(pkg.Evidence, contracts.Evidence{ID: "question", Kind: contracts.EvidenceQuestion, Text: in.Raw,
			Values: []contracts.Value{claimed}, Source: contracts.SourceRef{SourceType: "question"}, Subject: subject, Metric: metric, Period: period})
		claims = append(claims, contracts.Claim{Subject: subject, Metric: metric, Period: period, Value: &claimed,
			Source: "question", EvidenceIDs: []string{"question"}})
		others, ev, otherClaims := a.otherMetrics(ctx, metric, ids, from, to, subject)
		if others != "" {
			vars["others"] = others
			vars["others_note"] = a.fragment("others_note", vars)
		}
		pkg.Evidence = append(pkg.Evidence, ev...)
		claims = append(claims, otherClaims...)
	}
	_, base := a.RT.Metrics.Metric(metric)
	if !a.noWhy {
		a.noteGate(why, gate{in: in, metric: metric, base: base, confirming: a.confirming(in.Raw), confirm: confirm, compare: compare != "", withheld: withheld,
			tmpl: tmpl, claimed: claimed, total: result.Total, subject: entitySlot.Entity.Key, used: result.UsedSources})
	}
	// S8 — dựng câu bằng mẫu của tenant (route số liệu không gọi model)
	text := a.render(tmpl, vars)
	if withheld != "" {
		text += a.fragment("confirm_withheld", vars)
	}
	draft := contracts.Draft{Text: text, FromTemplate: true, Claims: claims}

	// S9 — kiểm: số có trong bằng chứng (lớp lọc đầu), rồi mỗi số ràng buộc đủ sáu chiều.
	violations, _ := traces.Measure(tr, "validate", func() ([]contracts.Violation, error) {
		v := validation.Fidelity(a.RT.Normalizer, draft, pkg, ignore, now)
		return append(v, validation.BindClaims(draft.Text, ignore[:2], draft.Claims, pkg.Evidence)...), nil
	})
	if len(violations) > 0 {
		resp.Answer = a.render("fallback_unverified", vars)
		resp.Verdict = contracts.Verdict{Pass: false, Violations: violations, Action: contracts.ActionFallback}
		resp.Outcome = contracts.OutcomeSystemFailure // câu không kiểm được thì không giao như thành công
		return nil
	}
	resp.Answer = draft.Text
	resp.Verdict = contracts.Verdict{Pass: true, Action: contracts.ActionAccept}
	resp.Outcome = contracts.OutcomeResolved
	return nil
}

// answerKnowledge chạy S6–S9 cho câu hỏi tri thức: truy xuất đoạn tài liệu người hỏi được đọc, cho
// model diễn đạt, rồi kiểm — claim gắn bằng chứng, số có trong bằng chứng, không lượng nghiệp vụ
// nào lấy từ tài liệu. hybrid = phần quy trình nối sau câu số liệu đã RESOLVED: phần này hỏng thì
// chỉ bỏ phần này kèm ghi chú, không làm hỏng phần số liệu đã kiểm.
// fallbackIntent hỏi dự phòng S4. Câu hỏi lượng (asksQuantity), hay người hỏi thiếu capability của ý định tri thức thì không hỏi (không dò tài liệu hộ người không được
// đọc); dự phòng lỗi thì giữ kết quả của luật và ghi vết — luật là đường mặc định, lỗi bên ngoài không làm hỏng lượt. Ngữ cảnh mang
// đúng truy vấn tri thức của answerKnowledge (ngày hỏi, phạm vi, chủ đề) để dò chỉ mục thấy đúng những gì route RAG sẽ thấy.
func (a *Ask) fallbackIntent(ctx context.Context, now time.Time, pr contracts.Principal, in contracts.NormalizedInput, cls intent.Result, f IntentFallback) (intent.Result, error) {
	def, ok := a.RT.Intents.Def(f.Intent())
	if !ok || contracts.RouteKind(def.Route) != contracts.RouteRAG {
		return cls, contracts.Errorf(contracts.FaultCode, "dự phòng S4 khai ý định không phải ý định tri thức của pack: "+f.Intent())
	}
	if (def.Capability != "" && !a.RT.Policy.Allows(pr, def.Capability)) || a.asksQuantity(in) {
		return cls, nil
	}
	kctx := contracts.WithKnowledgeQuery(ctx, contracts.KnowledgeQuery{AsOf: now, InScope: a.RT.Policy.InScope, Topic: KnowledgeTopic(in)})
	use, conf, why, err := f.Fallback(kctx, pr, in, cls)
	switch {
	case err != nil:
		cls.Why.Reasons = append(slices.Clone(cls.Why.Reasons), "dự phòng S4 lỗi ("+contracts.FaultOf(err)+") — giữ kết quả của luật")
	case use:
		cls = intent.Result{
			Intent: contracts.Intent{Name: def.Name, Confidence: conf, Slots: map[string]contracts.Slot{}, Rule: "intent:fallback"},
			Why:    intent.Why{Reasons: append(slices.Clone(cls.Why.Reasons), "dự phòng S4 → "+def.Name+": "+why), Rejected: cls.Why.Rejected},
		}
	}
	return cls, nil
}

func (a *Ask) answerKnowledge(ctx context.Context, tr *traces.Trace, now time.Time, pr contracts.Principal,
	in contracts.NormalizedInput, tmpl string, resp *contracts.AskResponse, hybrid bool, why *traces.Why) error {

	fail := func(name string, outcome contracts.Outcome, violations []contracts.Violation) {
		if hybrid {
			resp.Answer += a.fragment("hybrid_no_knowledge", nil)
			resp.Verdict.Violations = append(resp.Verdict.Violations, violations...)
			return
		}
		resp.Answer, resp.Outcome = a.render(name, nil), outcome
		resp.Verdict = contracts.Verdict{Pass: true, Action: contracts.ActionAccept}
		if len(violations) > 0 {
			resp.Verdict = contracts.Verdict{Pass: false, Violations: violations, Action: contracts.ActionFallback}
		}
	}
	// Câu hỏi đòi LƯỢNG nghiệp vụ ("bao nhiêu lít dầu") mà chỉ có tài liệu: tài liệu không phải
	// nguồn của số liệu (P4) — không gọi model, báo thiếu bằng chứng.
	if !hybrid && a.asksFactQuantity(in, now) {
		fail("no_knowledge", contracts.OutcomeInsufficientEvidence, nil)
		return nil
	}
	if a.Deps.Knowledge == nil || a.Deps.Model == nil {
		return contracts.Errorf(contracts.FaultCode, "route tri thức cần chỉ mục tài liệu và gateway model")
	}
	// S6 — truy xuất (tenant + ACL tài liệu do capability tri thức kiểm)
	var rep contracts.RetrievalReport
	ev, err := traces.Measure(tr, "retrieve", func() ([]contracts.Evidence, error) {
		kctx := contracts.WithKnowledgeQuery(ctx, contracts.KnowledgeQuery{AsOf: now, InScope: a.RT.Policy.InScope, Topic: KnowledgeTopic(in), Report: &rep})
		return a.Deps.Knowledge.Retrieve(kctx, pr, in.Raw, knowledgeK)
	})
	prefix := ""
	if hybrid {
		prefix = "tri thức: "
	}
	if err != nil {
		var n *contracts.KnowledgeNotice
		switch {
		case errors.As(err, &n): // không phải lỗi hệ thống: nói rõ cho người hỏi (BR-DOC-01)
			noteExecute(why, prefix, rep, nil, knowledgeK)
			why.Execute.Reason("%sdừng ở thông báo phiên bản của tài liệu %s: không trả đoạn nào (các bản ở S7)", prefix, n.DocumentID)
			noteNotice(why, hybrid, n)
			a.versionNotice(n, resp, hybrid)
			return nil
		case hybrid: // phần số liệu đã kiểm vẫn giao; lỗi nằm trong vết của bước retrieve
			why.Context.Reason("tri thức: truy xuất lỗi (%s) — bỏ phần quy trình, giữ phần số liệu đã kiểm", contracts.FaultOf(err))
			resp.Answer += a.fragment("hybrid_knowledge_error", nil)
			return nil
		}
		return err
	}
	noteExecute(why, prefix, rep, ev, knowledgeK)
	noteRetrieved(why, prefix, ev, knowledgeK)
	if len(ev) == 0 {
		fail("no_knowledge", contracts.OutcomeInsufficientEvidence, nil)
		return nil
	}
	// S7 — giá trị có kiểu trong từng đoạn, để kiểm số của bản nháp
	for i := range ev {
		ev[i].Values = a.valuesOf(ev[i].Text, now)
	}
	// S8 — model diễn đạt (đề xuất, chưa phải câu trả lời)
	draft, err := traces.Measure(tr, "generate", func() (contracts.Draft, error) {
		return a.Deps.Model.Generate(ctx, models.Request{Question: in.Raw, Evidence: ev})
	})
	if err != nil {
		if hybrid {
			resp.Answer += a.fragment("hybrid_knowledge_error", nil)
			return nil
		}
		return err
	}
	// Model nói rõ bằng chứng không trả lời được (Draft.Refused): "không có" như khi không đoạn nào khớp — không phải lỗi, không trích
	// dẫn (ROADMAP-RAG §6.5). Bản nháp rỗng / văn xuôi KHÔNG mang cờ là đầu ra hỏng: vẫn qua S9 và bị chặn.
	if draft.Refused {
		why.Context.Reason("%smodel: bằng chứng đã truy hồi không trả lời được câu hỏi — không trích đoạn nào", prefix)
		fail("no_knowledge", contracts.OutcomeInsufficientEvidence, nil)
		return nil
	}
	// S9 — kiểm
	violations, _ := traces.Measure(tr, "validate", func() ([]contracts.Violation, error) {
		v := validation.Bound(draft, ev)
		v = append(v, validation.Fidelity(a.RT.Normalizer, draft, contracts.ContextPackage{Evidence: ev}, nil, now)...)
		v = append(v, validation.ForbidDimensions(a.RT.Normalizer, draft, a.factDimensions(), now)...)
		return append(v, validation.ForbidUnitWords(draft.Text, a.factUnitWords())...), nil
	})
	if len(violations) > 0 {
		why.Context.Reject("%sbản nháp bị loại ở S9 nên không đoạn nào được trích dẫn", prefix)
		fail("fallback_unverified", contracts.OutcomeSystemFailure, violations)
		return nil
	}
	cited := citations(draft, ev)
	noteCited(why, prefix, ev, cited)
	resp.Citations = append(resp.Citations, cited...)
	vars := map[string]string{"title": cited[0].Source.Name, "procedure": draft.Text}
	if hybrid {
		resp.Answer += a.fragment(tmpl, vars)
		return nil
	}
	resp.Answer, resp.Outcome = a.render(tmpl, vars), contracts.OutcomeResolved
	resp.Verdict = contracts.Verdict{Pass: true, Action: contracts.ActionAccept}
	return nil
}

// noteNotice ghi S7 khi chỉ mục trả thông báo phiên bản thay cho đoạn: không đoạn nào được trích, kèm các bản người hỏi thấy.
func noteNotice(why *traces.Why, hybrid bool, n *contracts.KnowledgeNotice) {
	prefix := ""
	if hybrid {
		prefix = "tri thức: "
	}
	vs := make([]string, len(n.Versions))
	for i, v := range n.Versions {
		vs[i] = fmt.Sprintf("v%d %s [%s, %s)", v.Version, v.Status, v.EffectiveFrom, v.EffectiveTo)
		if v.Hold != "" {
			vs[i] += " giữ lại: " + v.Hold
		}
	}
	why.Context.Reason("%s%s %q: thông báo %s — các bản người hỏi thấy: %s; không trích đoạn nào, không gọi model (BR-DOC-01)",
		prefix, n.DocumentID, n.Title, n.Kind, strings.Join(vs, "; "))
}

// versionNotice nói rõ cho người hỏi khi tài liệu khớp nhất không có đúng một bản làm căn cứ: không chọn bản ngầm,
// không gọi model, không dùng bản cũ (BR-DOC-01, docs/viec/2-tra-quy-trinh/HOP-DONG.md §4).
func (a *Ask) versionNotice(n *contracts.KnowledgeNotice, resp *contracts.AskResponse, hybrid bool) {
	vs := make([]string, len(n.Versions))
	for i, v := range n.Versions {
		vs[i] = fmt.Sprintf("v%d", v.Version)
	}
	vars := map[string]string{"title": n.Title, "reason": strings.Join(vs, ", ")}
	name, outcome := "knowledge_no_effective", contracts.OutcomeInsufficientEvidence
	if n.Kind == contracts.NoticeVersionConflict {
		name, outcome = "knowledge_version_conflict", contracts.OutcomeSourceConflict
	}
	if hybrid {
		resp.Answer += a.fragment("hybrid_"+name, vars)
		return
	}
	resp.Answer, resp.Outcome = a.render(name, vars), outcome
	resp.Verdict = contracts.Verdict{Pass: true, Action: contracts.ActionAccept}
}

// asksFactQuantity báo câu hỏi đòi một LƯỢNG thuộc chiều đo của chỉ số nghiệp vụ ("bao nhiêu lít
// dầu"); cụm hỏi lượng do pack khai (quantity_markers).
func (a *Ask) asksFactQuantity(in contracts.NormalizedInput, now time.Time) bool {
	marked := false
	for _, m := range a.RT.QuantityMarkers {
		marked = marked || normalization.ContainsPhrase(in.Raw, m)
	}
	if !marked {
		return false
	}
	fact := a.factDimensions()
	for d := range validation.Dimensions(a.RT.Normalizer, in.Raw, now) {
		if fact[d] {
			return true
		}
	}
	return false
}

// factDimensions là các chiều đo có chỉ số nghiệp vụ (lượng dầu → volume): lượng thuộc các chiều
// này chỉ được đến từ fact, không từ tài liệu.
func (a *Ask) factDimensions() map[string]bool {
	out := map[string]bool{}
	for _, id := range a.RT.Metrics.MetricIDs() {
		m, _ := a.RT.Metrics.Metric(id)
		if u, ok := a.RT.Normalizer.Unit(m.Unit); ok {
			out[u.Dimension] = true
		}
	}
	return out
}

// factUnitWords là mọi cách viết đơn vị của các chiều đo nghiệp vụ (L, lít, m3, khối…) — đầu vào
// của lớp kiểm độc lập ForbidUnitWords, dài trước để "m3" thắng "m".
func (a *Ask) factUnitWords() []string {
	var out []string
	for d := range a.factDimensions() {
		for _, u := range a.RT.Normalizer.UnitsOf(d) {
			out = append(out, u.Unit)
			out = append(out, u.Aliases...)
		}
	}
	slices.SortFunc(out, func(x, y string) int { return len(y) - len(x) })
	return out
}

// valuesOf trả các giá trị số có kiểu trong một đoạn văn.
func (a *Ask) valuesOf(text string, now time.Time) []contracts.Value {
	var out []contracts.Value
	for _, sp := range a.RT.Normalizer.Normalize(text, normalization.Options{Now: now}).Spans {
		switch sp.Kind {
		case contracts.ValueQuantity, contracts.ValueMoney, contracts.ValueNumber:
			out = append(out, sp.Value)
		}
	}
	return out
}

// citations trả trích dẫn theo đúng các bằng chứng mà claim của bản nháp trỏ tới (Bound đã bảo
// đảm chúng nằm trong ngữ cảnh), mỗi bằng chứng một lần.
func citations(d contracts.Draft, ev []contracts.Evidence) []contracts.Citation {
	byID := map[string]contracts.Evidence{}
	for _, e := range ev {
		byID[e.ID] = e
	}
	seen := map[string]bool{}
	var out []contracts.Citation
	for _, c := range d.Claims {
		for _, id := range c.EvidenceIDs {
			if !seen[id] {
				seen[id] = true
				out = append(out, contracts.Citation{EvidenceID: id, Source: byID[id].Source})
			}
		}
	}
	return out
}

// claimedValue trả con số người hỏi đưa ra khi câu là câu XÁC NHẬN (có cụm confirm_markers của pack và đúng MỘT con số
// ngoài tên thực thể — xem claimNumbers), đã đổi về đơn vị chuẩn của chỉ số.
func (a *Ask) claimedValue(in contracts.NormalizedInput, names []mention, metric string) (contracts.Value, bool) {
	def, ok := a.RT.Metrics.Metric(metric)
	if !a.confirming(in.Raw) || !ok {
		return contracts.Value{}, false
	}
	cs := a.claimNumbers(in, names, def)
	if vals := cs.values; len(vals) != 1 || len(cs.doubts) > 0 || len(cs.groups) > 0 {
		return contracts.Value{}, false
	}
	return cs.values[0], true
}

// confirming báo câu có cụm xác nhận của pack ("đúng không").
func (a *Ask) confirming(text string) bool {
	for _, m := range a.RT.ConfirmMarkers {
		if normalization.ContainsPhrase(text, m) {
			return true
		}
	}
	return false
}

// inMetricUnit đổi một con số người hỏi nêu về đơn vị chuẩn của chỉ số (số trần lấy đơn vị đó); false khi không đổi
// được ("7h" không phải lượng dầu).
func (a *Ask) inMetricUnit(v contracts.Value, def sourceoftruth.MetricDef) (contracts.Value, bool) {
	if v.Kind == contracts.ValueNumber || v.Unit == "" {
		v.Kind, v.Unit = contracts.ValueQuantity, def.Unit
	}
	if v.Unit != def.Unit {
		cv, err := a.RT.Normalizer.Convert(v, def.Unit)
		if err != nil {
			return contracts.Value{}, false
		}
		v = cv
	}
	return v, true
}

// confirmClaims trả các con số KHÁC NHAU (đã đổi về đơn vị chuẩn của chỉ số) mà một câu xác nhận nêu ("3850 L đúng không, 4000 L thì sao?",
// "3850 L và 4000 L, đúng không?"; từ hai con số trở lên là câu không xác nhận được con số nào thay người hỏi) và các mốc thời gian chưa đọc được
// (xem claimSet.doubts). Chỉ xét route số liệu có chỉ số gốc và không phải câu so sánh — đúng những câu mà answerMetric coi là câu xác nhận —
// và dùng CÙNG định nghĩa "con số xác nhận" (claimNumbers) với claimedValue, để hai nơi không bao giờ lệch nhau.
func (a *Ask) confirmClaims(plan contracts.RoutePlan, in contracts.NormalizedInput, names []mention) claimSet {
	switch plan.Route {
	case contracts.RouteSQLQuery, contracts.RouteCalculation, contracts.RouteHybrid:
	default:
		return claimSet{}
	}
	it := plan.Intent
	def, ok := a.RT.Metrics.Metric(it.Slots["metric"].Text)
	if !ok || it.Slots["compare_metric"].Text != "" || !a.confirming(in.Raw) {
		return claimSet{}
	}
	return a.claimNumbers(in, names, def)
}

// mention là một cụm tên thực thể (đã nhận ra, kể cả bí danh mơ hồ): khoảng byte [start, end) trong NormalizedInput.Raw và các thực
// thể nó có thể chỉ tới (một, hoặc mọi ứng viên của bí danh mơ hồ).
type mention struct {
	start, end int
	ids        []string
}

// mentionsOf trả vị trí của MỌI cụm tên thực thể trong câu — cả các lần nhắc thứ hai của cùng một thực thể và các bí danh mơ hồ mà
// router gỡ về một thực thể sau đó (thực thể ấy lấy từ danh sách ứng viên nên không mang vị trí, xem contracts.AmbiguousMention).
func mentionsOf(res contracts.Resolution) []mention {
	var out []mention
	for _, e := range res.Entities {
		out = append(out, mention{start: e.Start, end: e.End, ids: []string{e.ID}})
	}
	for _, m := range res.Ambiguous {
		ids := make([]string, 0, len(m.Candidates))
		for _, c := range m.Candidates {
			ids = append(ids, c.ID)
		}
		out = append(out, mention{start: m.Start, end: m.End, ids: ids})
	}
	return out
}

// claimCand là một con số ứng viên của câu xác nhận, đã đổi về đơn vị chuẩn của chỉ số.
type claimCand struct {
	start, end int
	orig       contracts.Value // giá trị người hỏi viết, CHƯA đổi đơn vị (số trần nối với một lượng lấy đơn vị của lượng đó)
	v          contracts.Value
	// explicit: lượng có đơn vị VIẾT RÕ cùng chiều đo ("3850 L", "4 m3"). Số trần và đơn vị suy từ thuật ngữ ("dầu 2" → 2 L) thì không — chúng
	// có thể chỉ là nhãn ("ca 2", "phiếu 912", "năm 2026", "2 lần"). Lượng viết kèm mẫu số ("5.000 L/ngày": tốc độ / định mức) cũng không: nó
	// không cho phép bỏ thầm số trần của câu.
	explicit bool
	// formatted: số trần viết có hậu tố nhân, dấu nghìn / thập phân hoặc chữ ("4k", "4.000", "bốn ngàn"), hay số trần NỐI với một lượng bằng từ nối
	// ("từ 3800 đến 3900 L", "3850 hay 4000 L" — xem joinClaims): joinClaims cho số trần nối tiếp với nó ("3800, 3850 hay 3900 L").
	formatted bool
}

// claimSet là kết quả của claimNumbers: các con số xác nhận và các mốc thời gian không kiểm được.
type claimSet struct {
	// values: các con số KHÁC NHAU (đã đổi về đơn vị chuẩn của chỉ số) mà câu xác nhận đưa ra.
	values []contracts.Value
	// doubts: chữ người hỏi viết cho các mốc thời gian mà bộ chuẩn hoá KHÔNG đọc được ("tuần 41", "năm 2025", "ngày 12", "7 ngày") và không khớp
	// kỳ nào đã đọc được — kỳ của câu có thể khác kỳ họ nói, nên câu phải hỏi lại, không phán quyết. Mốc nêu thêm KHỚP kỳ đã đọc
	// ("hôm nay tuần 43", khi hôm nay thuộc tuần 43) là bối cảnh vô hại.
	doubts []string
	// groups: chữ của các lượng viết nhóm nghìn CHƯA PHÂN ĐỊNH trong phạm vi hỏi — con số không đứng đầu câu ("bể 2 500 lít": 2.500 L hay bể 2 với
	// 500 L: normalization.UncertainGroup) hay chồng một phần lên tên thực thể ("lu 2 850 lít": normalization.GroupSplitsName) — câu phải hỏi lại,
	// không phán quyết theo một cách đọc.
	groups []string
}

// claimNumbers trả các con số KHÁC NHAU (đã đổi về đơn vị chuẩn của chỉ số) mà câu xác nhận đưa ra cho chỉ số def, kèm các mốc thời gian không kiểm
// được. Một con số là ứng viên khi: là lượng hay số trần, KHÔNG nằm trong tên một thực thể ("03" của "máy 03"), và đổi được về đơn vị chuẩn
// ("7h", "2 ca" khác chiều đo nên bỏ). Số trần đứng cạnh một từ chỉ kỳ ("tuần 43", "năm 2025", "ngày 12", "7 ngày qua") là mốc thời gian, không phải
// con số (xem normalization.PeriodQualifier); số trần là nhãn hay số đếm ("phiếu 912", "ca 2", "2 máy") cũng vậy (normalization.LabelContext).
// Phạm vi:
//   - mệnh đề chứa cụm xác nhận và MỌI mệnh đề sau nó ("3850 L đúng không, 4000 L thì sao?"); số ở mệnh đề đứng TRƯỚC là ngữ cảnh
//     ("đội báo 2k, vậy tổng báo 4k đúng không?");
//   - mệnh đề chứa cụm xác nhận mà không có con số nào ("3850 L và 4000 L, đúng không?") thì câu hỏi áp cho các con số đứng trước
//     trong CÙNG câu (tách bởi . ? !; "..." không phải hết câu), không phải cho câu trước đó.
//
// Số trần khác chỉ tính khi chính mệnh đề hỏi không có lượng nào viết rõ đơn vị — "3850 đúng không, 4000 thì sao?" có hai con số;
// còn "tháng 10 năm 2026 … 3850 L đúng không?" chỉ có MỘT (2026 là năm). Số trần có dạng một lượng ("4k", "4.000"), nối với một lượng bằng từ nối
// ("từ 3800 đến 3900 L", "3850 chứ không phải 4000 L") hoặc đứng đầu một mệnh đề ("3850 L đúng không, 4000 thì sao?") vẫn là con số cạnh tranh. Cùng một
// giá trị viết hai cách ("4k", "4000 L") chỉ tính một lần.
func (a *Ask) claimNumbers(in contracts.NormalizedInput, names []mention, def sourceoftruth.MetricDef) claimSet {
	var cs claimSet
	periods := readPeriods(in)
	var cands []claimCand
	skip := 0      // số span còn lại thuộc một cụm ngày dd-mm đã ghi là mốc chưa đọc được
	labelEnd := -1 // cuối số hiệu vừa bỏ ("phiếu 12-10": phần "-10" thuộc cùng số hiệu)
	// vị trí các nhóm nghìn chưa phân định (xem claimSet.groups); chỉ giữ cái nằm trong phạm vi hỏi
	var unsure [][2]int
	for i, sp := range in.Spans {
		if skip > 0 {
			skip--
			continue
		}
		if sp.Kind != contracts.ValueQuantity && sp.Kind != contracts.ValueNumber {
			continue
		}
		if slices.ContainsFunc(names, func(m mention) bool { return normalization.GroupSplitsName(sp, m.start, m.end) }) {
			// nhóm nghìn chồng MỘT PHẦN lên tên ("lu 2 850 lít"): không bỏ im lặng như số trong tên — hỏi lại (không là ứng viên: cách đọc chưa chắc)
			if _, ok := a.inMetricUnit(sp.Value, def); ok {
				unsure = append(unsure, [2]int{sp.Start, sp.End})
			}
			continue
		}
		if slices.ContainsFunc(names, func(m mention) bool { return sp.Start < m.end && sp.End > m.start }) {
			continue
		}
		if text, n := normalization.DashDateAt(in.Raw, in.Spans, i); n > 1 {
			// "19-10", "19-10-26": ngày viết dd-mm mà bộ chuẩn hoá không đọc khi thiếu chữ "ngày" — kỳ của câu có thể khác kỳ họ nói
			if !slices.Contains(cs.doubts, text) {
				cs.doubts = append(cs.doubts, text)
			}
			skip = n - 1
			continue
		}
		v, ok := a.inMetricUnit(sp.Value, def)
		if !ok {
			continue
		}
		if normalization.UncertainGroup(sp) {
			unsure = append(unsure, [2]int{sp.Start, sp.End}) // vẫn là ứng viên: các luật đếm con số bên dưới giữ nguyên
		}
		explicit := sp.Kind == contracts.ValueQuantity && !strings.Contains(sp.Rule, "unit_from_term")
		if explicit && rateAfter(in.Raw, sp.End) {
			explicit = false // "5.000 L/ngày": tốc độ, không phải lượng đã cấp
		}
		formatted := !explicit && strings.ContainsFunc(sp.Raw, func(r rune) bool { return unicode.IsLetter(r) || r == '.' || r == ',' })
		if !explicit && !formatted {
			// số trần trơn: mốc thời gian viết bằng chữ, nhãn / số đếm, hay một con số? Mốc xét TRƯỚC: "Quý 3 dự án A", "Tuần 41 máy xúc"
			// là mốc dù ngay sau là danh từ đếm (soát R, R-A01 / B-01); "phiếu 912 tháng 9" vẫn là số phiếu vì "phiếu" không phải từ chỉ kỳ.
			if q, ok := normalization.PeriodQualifier(in.Raw, sp.Start, sp.End); ok && !(q.Kind == normalization.QualifierSpan && inReadPeriod(in, q.Word)) {
				if !qualifierHolds(q, periods) && !slices.Contains(cs.doubts, q.Raw) {
					cs.doubts = append(cs.doubts, q.Raw)
				}
				continue
			}
			if normalization.LabelContext(in.Raw, sp.Start, sp.End) || dashTail(in.Raw, labelEnd, sp.Start) {
				labelEnd = sp.End
				continue
			}
		}
		cands = append(cands, claimCand{start: sp.Start, end: sp.End, orig: sp.Value, v: v, explicit: explicit, formatted: formatted})
	}
	a.joinClaims(in, def, cands)
	lo, hi := 0, len(in.Raw) // [lo, hi): mệnh đề hỏi; từ hi trở đi: các mệnh đề sau nó
	if fs, fe, focus := intent.FocusClause(in.Raw, a.RT.ConfirmMarkers); focus {
		lo, hi = fs, fe
		for lo > 0 && contrastLeading(in.Raw[lo:hi]) { // "…3850 L, hay 4000 L đúng không?": mệnh đề hỏi nối tiếp mệnh đề trước
			lo = prevClauseStart(in.Raw, lo)
		}
		if !slices.ContainsFunc(cands, func(c claimCand) bool { return c.start >= lo && c.start < hi }) {
			lo = sentenceStart(in.Raw, fs)
		}
	}
	for _, u := range unsure {
		if raw := in.Raw[u[0]:u[1]]; u[0] >= lo && !slices.Contains(cs.groups, raw) {
			cs.groups = append(cs.groups, raw)
		}
	}
	var primary, trailing []claimCand
	for _, c := range cands {
		switch {
		case c.start < lo: // ngữ cảnh
		case c.start < hi:
			primary = append(primary, c)
		default:
			trailing = append(trailing, c)
		}
	}
	// MỌI con số ứng viên của mệnh đề hỏi và các mệnh đề sau đều tính (trừ số trong tên thực thể, số thuộc kỳ đã đọc, số hiệu sau từ nhãn — đã
	// loại ở trên). Luật A-02 cũ bỏ số trần khi mệnh đề có lượng viết rõ đơn vị; nó đổi câu mơ hồ ("định mức 5.000 L nên … 3850", "3800 ~ 3900 L",
	// "4000 L hay 3850") thành phán quyết SAI, và vá bằng danh sách đóng vẫn rò (soát R: R-A06, R-A07, B-02, B-03). Hai con số trở lên thì hỏi lại,
	// như trước bản sửa A.
	var out []contracts.Value
	add := func(c claimCand) {
		if !slices.ContainsFunc(out, func(o contracts.Value) bool { n, err := decimal.Cmp(o.Number, c.v.Number); return err == nil && n == 0 }) {
			out = append(out, c.v)
		}
	}
	for _, c := range primary {
		add(c)
	}
	for _, c := range trailing {
		add(c)
	}
	cs.values = out
	return cs
}

// dashTail báo con số bắt đầu ở start là phần nối bằng "-" / "–" ngay sau số hiệu kết thúc ở labelEnd ("phiếu 12-10", "ca 2-3").
func dashTail(raw string, labelEnd, start int) bool {
	if labelEnd < 0 || start <= labelEnd || start > len(raw) {
		return false
	}
	gap := raw[labelEnd:start]
	return gap == "-" || gap == "–"
}

// claimConnectors là các từ / dấu nối hai con số của MỘT lời xác nhận: dải ("từ 3800 đến 3900 L", "3800-3900 L"), lựa chọn ("3850 hay 4000 L"),
// liệt kê ("3850 và 4000 L") và phủ định vế ("3850 chứ không phải 4000 L").
var claimConnectors = []string{"-", "–", "—", "/", "đến", "tới", "hay", "hoặc", "hay là", "hoặc là", "và", "chứ không phải", "không phải", "chứ không", "so với", "thay vì"}

// claimConnector báo đoạn chữ giữa hai con số CHỈ gồm một từ nối của claimConnectors (không phân biệt hoa thường, dấu; ngoặc quanh từ nối được bỏ:
// "3850 L (hoặc 4000)").
func claimConnector(gap string) bool {
	g := normalization.Fold(normalization.CollapseSpaces(strings.NewReplacer("(", " ", ")", " ", "[", " ", "]", " ").Replace(gap)))
	return g != "" && slices.ContainsFunc(claimConnectors, func(c string) bool { return g == normalization.Fold(c) })
}

// contrastLeads là các từ nối LỰA CHỌN / PHỦ ĐỊNH vế: mệnh đề hỏi mở đầu bằng một trong chúng ("…3850 L, hay 4000 L đúng không?"; "hay là" thuộc "hay",
// "hoặc là" thuộc "hoặc", "chứ không phải" thuộc "chứ không") nối tiếp mệnh đề liền trước, nên con số của mệnh đề đó là một vế của câu hỏi chứ không phải
// ngữ cảnh. ("vậy", "và", "còn" mở đầu mệnh đề mới: ngữ cảnh vẫn là ngữ cảnh.)
var contrastLeads = []string{"hay", "hoặc", "chứ không", "không phải"}

// contrastLeading báo mệnh đề clause mở đầu bằng một từ nối của contrastLeads.
func contrastLeading(clause string) bool {
	clause = strings.TrimLeft(clause, " \t\n")
	return slices.ContainsFunc(contrastLeads, func(c string) bool {
		return slices.ContainsFunc(normalization.PhraseSpans(clause, c), func(sp [2]int) bool { return sp[0] == 0 })
	})
}

// prevClauseStart trả đầu mệnh đề liền trước vị trí lo — lo là đầu một mệnh đề, ngay sau một dấu , ; . ? ! (ranh giới như FocusClause; "..." không cắt).
func prevClauseStart(text string, lo int) int {
	for i := lo - 2; i >= 0; i-- {
		if strings.IndexByte(",;.?!", text[i]) >= 0 && (text[i+1] == ' ' || text[i+1] == '\n') && !ellipsisAt(text, i) {
			return i + 1
		}
	}
	return 0
}

// joinClaims cho số trần NỐI với một lượng (hay một số trần đã nối) bằng từ nối dùng chung đơn vị của lượng đó — "từ 3800 đến 3900 L", "3800-3900 L",
// "3850 hay 4000 L", "3850 chứ không phải 4000 L", "3850 L hay 4000": nó là một con số của câu, không phải nhãn ("năm 2026", "ca 2") nên không được bỏ
// thầm khi câu đã có một lượng viết rõ đơn vị. Số trần nối về phía SAU (đứng trước lượng) hay về phía TRƯỚC (đứng sau lượng; chỉ khi ngay sau nó
// không phải một chữ khác — "3850 L và 4 xe" là số đếm, "3850 L hay 4000 đúng không?" thì không). Số nối được đánh dấu formatted và lấy đơn vị của
// lượng nối với nó ("từ 3 đến 4 m3": 3 m3).
func (a *Ask) joinClaims(in contracts.NormalizedInput, def sourceoftruth.MetricDef, cands []claimCand) {
	plain := func(c claimCand) bool { return !c.explicit && !c.formatted }
	gap := func(from, to claimCand) string {
		if from.end > to.start {
			return "" // hai span chồng nhau: không có đoạn chữ giữa
		}
		return in.Raw[from.end:to.start]
	}
	join := func(k, n int) {
		c := &cands[k]
		c.formatted = true
		if u := cands[n].orig.Unit; u != "" && c.orig.Unit == "" {
			if v, ok := a.inMetricUnit(contracts.Value{Kind: contracts.ValueQuantity, Number: c.orig.Number, Unit: u}, def); ok {
				c.v = v
			}
		}
	}
	for k := len(cands) - 2; k >= 0; k-- { // đứng TRƯỚC lượng: "từ 3800 đến 3900 L"
		if !cands[k].explicit && !plain(cands[k+1]) && claimConnector(gap(cands[k], cands[k+1])) {
			join(k, k+1)
		}
	}
	for k := 1; k < len(cands); k++ { // đứng SAU lượng: "3850 L hay 4000"
		if !cands[k].explicit && !plain(cands[k-1]) && claimConnector(gap(cands[k-1], cands[k])) && !a.wordFollows(in.Raw, cands[k].end) {
			join(k, k-1)
		}
	}
}

// wordFollows báo ngay sau vị trí pos (chỉ cách khoảng trắng) là một CHỮ không thuộc cụm xác nhận: số đứng trước một danh từ ("4 xe") là số đếm.
func (a *Ask) wordFollows(text string, pos int) bool {
	rest := strings.TrimLeft(text[pos:], " \t\n")
	r, _ := utf8.DecodeRuneInString(rest)
	if rest == "" || !unicode.IsLetter(r) {
		return false
	}
	for _, m := range a.RT.ConfirmMarkers {
		for _, sp := range normalization.PhraseSpans(rest, m) {
			if sp[0] == 0 {
				return false
			}
		}
	}
	return true
}

// rateAfter báo lượng kết thúc ở end được viết kèm MẪU SỐ ("5.000 L/ngày", "5 m3/ca"): đó là tốc độ hay định mức, không phải số lượng đã cấp.
func rateAfter(text string, end int) bool {
	if end < 0 || end+1 >= len(text) || text[end] != '/' {
		return false
	}
	r, _ := utf8.DecodeRuneInString(text[end+1:])
	return unicode.IsLetter(r)
}

// inReadPeriod báo đoạn byte word nằm trong một mốc / khoảng thời gian bộ chuẩn hoá đã đọc: "tuần" của "tuần này" không phải từ chỉ kỳ mà con số đứng
// trước nó ("3850 tuần này") đếm. Chỉ dùng cho khoảng đếm ("N tuần"): "tháng" của "15 tháng 9" cũng nằm trong kỳ đã đọc (tháng 9) nhưng 15 là NGÀY
// của tháng đó — mốc chưa đọc được.
func inReadPeriod(in contracts.NormalizedInput, word [2]int) bool {
	return slices.ContainsFunc(in.SpansOf(contracts.ValuePeriod, contracts.ValueDate), func(sp contracts.Span) bool { return sp.Start <= word[0] && word[1] <= sp.End })
}

// readPeriods trả khoảng ngày [từ, đến] của mọi mốc / khoảng thời gian bộ chuẩn hoá đã ĐỌC được trong câu.
func readPeriods(in contracts.NormalizedInput) [][2]time.Time {
	var out [][2]time.Time
	for _, sp := range in.SpansOf(contracts.ValuePeriod, contracts.ValueDate) {
		if from, to, err := parseBounds(sp.Value.Text, time.UTC); err == nil {
			out = append(out, [2]time.Time{from, to})
		}
	}
	return out
}

// qualifierHolds báo mốc thời gian người hỏi viết bằng chữ mà bộ chuẩn hoá không đọc được ("tuần 43", "quý 4", "năm 2026", "ngày 19") KHỚP mọi kỳ đã
// đọc được trong câu — kỳ đã đọc nằm trọn trong mốc đó, nên mốc chỉ là bối cảnh rộng hơn của kỳ hẹp ("hôm nay tuần 43"). Không có kỳ nào đọc được,
// kỳ nằm ngoài hay RỘNG hơn mốc ("tháng 10 ngày 12": cả tháng không nằm trong ngày 12), hay mốc là một khoảng đếm ("7 ngày qua") thì không khớp.
// Tuần là tuần ISO 8601 (thứ Hai là đầu tuần, như "tuần này"), quý là quý dương lịch.
func qualifierHolds(q normalization.Qualifier, periods [][2]time.Time) bool {
	if len(periods) == 0 {
		return false
	}
	quarter := func(t time.Time) int { return (int(t.Month())-1)/3 + 1 }
	for _, p := range periods {
		from, to := p[0], p[1]
		var ok bool
		switch q.Kind {
		case normalization.QualifierDay:
			ok = from.Equal(to) && from.Day() == q.Value
		case normalization.QualifierWeek:
			y1, w1 := from.ISOWeek()
			y2, w2 := to.ISOWeek()
			ok = y1 == y2 && w1 == w2 && w1 == q.Value
		case normalization.QualifierMonth:
			ok = from.Year() == to.Year() && from.Month() == to.Month() && int(from.Month()) == q.Value
		case normalization.QualifierQuarter:
			ok = from.Year() == to.Year() && quarter(from) == quarter(to) && quarter(from) == q.Value
		case normalization.QualifierYear:
			ok = from.Year() == q.Value && to.Year() == q.Value
		default: // QualifierSpan ("7 ngày qua"): khoảng đếm bằng số, không kiểm được với kỳ nào
		}
		if !ok {
			return false
		}
	}
	return true
}

// sentenceStart trả vị trí đầu CÂU (tách bởi . ? ! theo sau là khoảng trắng — "3.850", "0,5" không cắt; dấu chấm lửng "..." giữa câu cũng không cắt)
// chứa vị trí at.
func sentenceStart(text string, at int) int {
	start := 0
	for i := 0; i < at && i < len(text); i++ {
		if strings.IndexByte(".?!", text[i]) >= 0 && i+1 < len(text) && (text[i+1] == ' ' || text[i+1] == '\n') && !ellipsisAt(text, i) {
			start = i + 1
		}
	}
	return start
}

// ellipsisAt báo dấu chấm ở vị trí i thuộc một dấu chấm lửng ("...", "....": liền sau hay liền trước nó là một dấu chấm khác).
func ellipsisAt(text string, i int) bool {
	return text[i] == '.' && ((i > 0 && text[i-1] == '.') || (i+1 < len(text) && text[i+1] == '.'))
}

// otherMetrics kê tổng các chỉ số GỐC khác của cùng thực thể, cùng kỳ (đã qua thẩm quyền), kèm
// bằng chứng để cổng S9 kiểm được các số đó.
func (a *Ask) otherMetrics(ctx context.Context, metric string, ids []string, from, to time.Time, subject string) (string, []contracts.Evidence, []contracts.Claim) {
	var parts []string
	var ev []contracts.Evidence
	var claims []contracts.Claim
	for _, id := range a.RT.Metrics.MetricIDs() {
		if id == metric {
			continue
		}
		r, err := a.RT.Metrics.Query(ctx, a.Deps.Facts, sourceoftruth.MetricQuery{Metric: id, EntityIDs: ids, From: from, To: to})
		if err != nil || r.Total == nil || r.NeedsReview {
			continue
		}
		p := lowerFirst(a.metricLabel(id)) + " " + displayNumber(r.Total.Number) + " " + a.unitLabel(r.Total.Unit)
		if r.Approx {
			p += a.RT.Templates["approx_mark"]
		}
		parts = append(parts, p)
		stampSubject(r.Evidence, subject)
		ev = append(ev, r.Evidence...)
		claims = append(claims, contracts.Claim{Subject: subject, Metric: id, Period: sourceoftruth.PeriodKey(from, to),
			Value: r.Total, Source: "calculation", EvidenceIDs: []string{"calc:" + id}})
	}
	return strings.Join(parts, "; "), ev, claims
}

// stampSubject ghi chủ thể được hỏi vào bằng chứng phép tính: nguồn sự thật tính trên TẬP thực thể
// đã mở rộng (máy con của dự án), chủ thể của phép tính là thực thể người hỏi nhắc tới.
func stampSubject(ev []contracts.Evidence, subject string) {
	for i := range ev {
		if ev[i].Kind == contracts.EvidenceCalculation && ev[i].Subject == "" {
			ev[i].Subject = subject
		}
	}
}

func lowerFirst(s string) string {
	r, n := utf8.DecodeRuneInString(s)
	return strings.ToLower(string(r)) + s[n:]
}

func (a *Ask) now() time.Time {
	if a.Deps.Now != nil {
		return a.Deps.Now().In(a.RT.Location)
	}
	return a.RT.Now()
}

func validateRequest(req contracts.AskRequest) error {
	switch {
	case strings.TrimSpace(req.Text) == "":
		return contracts.Errorf(contracts.FaultInput, "text rỗng")
	case utf8.RuneCountInString(req.Text) > MaxTextRunes:
		return contracts.Errorf(contracts.FaultInput, fmt.Sprintf("text dài quá %d ký tự", MaxTextRunes))
	case strings.ContainsRune(req.Text+req.ConversationID+req.MessageID, 0):
		return contracts.Errorf(contracts.FaultInput, "đầu vào chứa ký tự NUL")
	case req.ConversationID == "" || req.MessageID == "":
		return contracts.Errorf(contracts.FaultInput, "thiếu conversation_id hoặc message_id")
	}
	return nil
}

// firstEntitySlot lấy khe thực thể đứng đầu theo TÊN khe — chủ thể truy vấn của S6. Duyệt theo tên đã sắp xếp như router (ADR-0002 D18; soát
// C8–C11 C10-S3): thứ tự lặp của map ngẫu nhiên mỗi lần chạy, nên với ý định có hai khe thực thể trở lên thì chủ thể (và câu trả lời) sẽ đổi
// giữa các lần chạy. Pack construction chỉ có một khe thực thể.
func firstEntitySlot(it contracts.Intent) contracts.Slot {
	for _, n := range slices.Sorted(maps.Keys(it.Slots)) {
		if s := it.Slots[n]; s.Entity != nil {
			return s
		}
	}
	return contracts.Slot{Entity: &contracts.EntityRef{}}
}

// periodBounds lấy khoảng ngày của khe period (ngày hoặc khoảng ngày) — khe kỳ đứng đầu theo TÊN khe, cùng lý do với firstEntitySlot.
func periodBounds(it contracts.Intent, now time.Time) (time.Time, time.Time, error) {
	text := now.Format("2006-01-02")
	for _, n := range slices.Sorted(maps.Keys(it.Slots)) {
		if s := it.Slots[n]; s.Value != nil && (s.Value.Kind == contracts.ValueDate || s.Value.Kind == contracts.ValuePeriod) {
			text = s.Value.Text
			break
		}
	}
	return parseBounds(text, now.Location())
}

// parseBounds đọc "YYYY-MM-DD" (một ngày) hoặc "YYYY-MM-DD/YYYY-MM-DD" (một khoảng) thành [từ, đến].
func parseBounds(text string, loc *time.Location) (time.Time, time.Time, error) {
	fromS, toS, found := strings.Cut(text, "/")
	if !found {
		toS = fromS
	}
	from, err1 := time.ParseInLocation("2006-01-02", fromS, loc)
	to, err2 := time.ParseInLocation("2006-01-02", toS, loc)
	if err1 != nil || err2 != nil {
		return time.Time{}, time.Time{}, contracts.Errorf(contracts.FaultCode, "khoảng ngày không đọc được: "+text)
	}
	return from, to, nil
}

func displayPeriod(from, to time.Time) string {
	if from.Equal(to) {
		return "ngày " + from.Format("02/01/2006")
	}
	return "từ " + from.Format("02/01/2006") + " đến " + to.Format("02/01/2006")
}

// displayNumber in số kiểu Việt: "3850" → "3.850", "0.35" → "0,35".
func displayNumber(n string) string {
	intPart, frac, _ := strings.Cut(n, ".")
	neg := strings.HasPrefix(intPart, "-")
	intPart = strings.TrimPrefix(intPart, "-")
	var b strings.Builder
	for i, r := range intPart {
		if i > 0 && (len(intPart)-i)%3 == 0 {
			b.WriteByte('.')
		}
		b.WriteRune(r)
	}
	out := b.String()
	if frac != "" {
		out += "," + frac
	}
	if neg {
		out = "-" + out
	}
	return out
}

func (a *Ask) metricLabel(id string) string {
	if m, ok := a.RT.Metrics.Metric(id); ok && m.Label != "" {
		return m.Label
	}
	if d, ok := a.RT.Metrics.Derived(id); ok && d.Label != "" {
		return d.Label
	}
	return id
}

func (a *Ask) unitLabel(code string) string {
	if u, ok := a.RT.Normalizer.Unit(code); ok && len(u.Aliases) > 0 {
		return u.Aliases[0]
	}
	return code
}

// conflictItems dựng "<nhãn nguồn> <giá trị> <đơn vị>" cho mọi nguồn không được chọn trong các
// xung đột đã giải bằng thẩm quyền (xung đột chờ người duyệt không kê số).
func (a *Ask) conflictItems(conflicts []contracts.Conflict) ([]string, []contracts.Claim) {
	var out []string
	var claims []contracts.Claim
	for _, c := range conflicts {
		if c.NeedsReview {
			continue
		}
		parts := strings.SplitN(c.Key, "|", 3) // chỉ số | thực thể | ngày
		for _, o := range c.Others {
			out = append(out, a.RT.Metrics.SourceLabel(parts[0], o.SourceType)+" "+displayNumber(o.Number)+" "+a.unitLabel(o.Unit))
			if len(parts) == 3 {
				claims = append(claims, contracts.Claim{Subject: parts[1], Metric: parts[0], Period: parts[2],
					Value: &contracts.Value{Kind: contracts.ValueQuantity, Number: o.Number, Unit: o.Unit}, Source: o.SourceType, EvidenceIDs: o.EvidenceIDs})
			}
		}
	}
	return out, claims
}

// visibleCandidates chỉ giữ ứng viên nằm trong phạm vi dữ liệu của người hỏi — câu hỏi lại không
// được làm lộ tên thiết bị ngoài phạm vi. Không tra được tổ tiên thì bỏ ứng viên (đóng mặc định).
func (a *Ask) visibleCandidates(ctx context.Context, pr contracts.Principal, cands []contracts.EntityRef) ([]contracts.EntityRef, []hiddenCandidate) {
	var out []contracts.EntityRef
	var hidden []hiddenCandidate // ứng viên bị loại kèm lý do — chỉ để giải thích vết
	for _, c := range cands {
		keys, err := entityresolution.Lineage(ctx, a.Deps.Directory, c.ID)
		switch {
		case err != nil:
			hidden = append(hidden, hiddenCandidate{c, "không tra được tổ tiên trong danh mục (đóng mặc định)"})
		case !a.RT.Policy.InScope(pr, keys):
			hidden = append(hidden, hiddenCandidate{c, "ngoài phạm vi dữ liệu của người hỏi"})
		default:
			out = append(out, c)
		}
	}
	return out, hidden
}

// clarificationDetails dựng phần "thiếu gì" và "ứng viên" cho câu hỏi lại; notes là các câu giải thích thêm (nhiều mốc
// thời gian, ngày hỏng, nhiều con số) nối sau danh sách ứng viên.
func (a *Ask) clarificationDetails(it contracts.Intent, candidates []contracts.EntityRef, notes []string) (string, string) {
	def, _ := a.RT.Intents.Def(it.Name)
	// "claimed" không phải khe của pack: là con số cần xác nhận của câu xác nhận nêu nhiều con số
	labels := map[string]string{"metric": "chỉ số cần hỏi", "claimed": "con số cần xác nhận"}
	for _, s := range def.Slots {
		if s.Label != "" {
			labels[s.Name] = s.Label
		}
	}
	var missing []string
	for _, m := range it.Missing {
		if l, ok := labels[m]; ok {
			missing = append(missing, l)
		} else {
			missing = append(missing, m)
		}
	}
	var cands []string
	for _, c := range candidates {
		cands = append(cands, c.Name+" ("+c.Key+")")
	}
	candText := ""
	if len(cands) > 0 {
		candText = "Có thể là: " + strings.Join(cands, ", ") + "."
	}
	for _, n := range notes {
		candText = strings.TrimSpace(candText + " " + n)
	}
	return strings.Join(missing, ", "), candText
}

// unclearNotes dựng phần giải thích thêm của câu hỏi lại khi câu nêu nhiều mốc thời gian hoặc ngày hỏng (P5): các mốc
// đã nhận ra (chữ người hỏi viết, kèm ngày chuẩn) và các ngày không hợp lệ. Thực thể không ở đây — chúng là ứng viên có kiểu.
func (a *Ask) unclearNotes(u intent.Unclear, loc *time.Location) []string {
	var notes []string
	if len(u.Periods) > 0 {
		shown := make([]string, 0, len(u.Periods))
		for _, sp := range u.Periods {
			d := sp.Value.Text
			if from, to, err := parseBounds(d, loc); err == nil {
				d = displayPeriod(from, to)
			}
			if !strings.Contains(d, sp.Raw) {
				d = sp.Raw + " (" + d + ")"
			}
			shown = append(shown, d)
		}
		notes = append(notes, "Câu hỏi nêu nhiều mốc thời gian: "+strings.Join(shown, ", ")+".")
	}
	if len(u.Invalid) > 0 {
		notes = append(notes, "Ngày "+strings.Join(u.Invalid, ", ")+" không hợp lệ.")
	}
	return notes
}

func (a *Ask) intentOptions(names []string) string {
	var out []string
	for _, n := range names {
		if d, ok := a.RT.Intents.Def(n); ok && d.Label != "" {
			out = append(out, d.Label)
		} else {
			out = append(out, n)
		}
	}
	return strings.Join(out, " hay ")
}

// render thay biến trong mẫu câu. Biến không có giá trị thành chuỗi rỗng; tên biến đã
// được kiểm lúc nạp pack nên không có biến lạ.
func (a *Ask) render(name string, vars map[string]string) string {
	return strings.TrimSpace(a.fragment(name, vars))
}

// fragment dựng một MẨU câu (ghi chú ghép vào câu chính) — giữ nguyên khoảng trắng đầu/cuối của
// mẫu, vì mẩu bắt đầu bằng dấu cách để nối sau dấu chấm.
//
// Thay MỘT lượt trên mẫu (strings.Replacer): giá trị của biến không bao giờ bị quét lại. Văn bản do model
// viết ({procedure}) chứa "{title}" thì giữ nguyên từng chữ — thay lượt hai làm câu giao khác câu đã qua
// S9 (kết quả đổi theo thứ tự duyệt map; xoá "{reason}" giữa "3" và "850" ghép thành số mới "3850").
func (a *Ask) fragment(name string, vars map[string]string) string {
	pairs := make([]string, 0, 2*(len(vars)+len(tenant.AllowedPlaceholders)))
	for k, v := range vars {
		pairs = append(pairs, "{"+k+"}", v)
	}
	for p := range tenant.AllowedPlaceholders {
		if _, set := vars[p]; !set { // biến được phép mà chưa có giá trị thì thành chuỗi rỗng
			pairs = append(pairs, "{"+p+"}", "")
		}
	}
	return strings.NewReplacer(pairs...).Replace(a.RT.Templates[name])
}
