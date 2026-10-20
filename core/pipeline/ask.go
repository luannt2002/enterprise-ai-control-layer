// Package pipeline điều phối các lối vào của EAIS. File này là luồng HỎI (S0–S10).
//
// Ba luật cứng (docs/01-architecture.md §4):
//  1. Route không cần diễn đạt thì không gọi model — trả lời bằng mẫu câu của tenant.
//  2. Model không bao giờ là nguồn của con số — số đến từ nguồn sự thật, có bằng chứng.
//  3. Kiểm thất bại thì không giao câu đó.
package pipeline

import (
	"context"
	"fmt"
	"runtime/debug"
	"slices"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
	"github.com/luannt2002/enterprise-ai-systems/core/contracts"
	"github.com/luannt2002/enterprise-ai-systems/core/decimal"
	"github.com/luannt2002/enterprise-ai-systems/core/tenant"
	"github.com/luannt2002/enterprise-ai-systems/data/entityresolution"
	"github.com/luannt2002/enterprise-ai-systems/data/normalization"
	"github.com/luannt2002/enterprise-ai-systems/data/sourceoftruth"
	"github.com/luannt2002/enterprise-ai-systems/intelligence/intent"
	"github.com/luannt2002/enterprise-ai-systems/intelligence/models"
	"github.com/luannt2002/enterprise-ai-systems/intelligence/router"
	"github.com/luannt2002/enterprise-ai-systems/observability/traces"
	"github.com/luannt2002/enterprise-ai-systems/reliability/validation"
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

// Deps là các nguồn dữ liệu của MỘT tenant mà luồng hỏi cần.
type Deps struct {
	Directory entityresolution.Directory
	Facts     sourceoftruth.Store
	// Knowledge, Model: chỉ mục tài liệu và gateway model cho route RAG / HYBRID. Thiếu thì route
	// đó là lỗi cấu hình, không âm thầm trả lời.
	Knowledge KnowledgeRetriever
	Model     models.Gateway
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

	resp := contracts.AskResponse{
		Route: plan.Route, Outcome: plan.Outcome, Intent: plan.Intent.Name, PackVersion: a.RT.PackVersion, TraceID: uuid.NewString(),
	}
	vars := map[string]string{"reason": plan.Reason}

	// Từ đây chỉ dùng plan.Intent: ý định SAU khi router đã kiểm quyền và gỡ mơ hồ.
	switch plan.Route {
	case contracts.RouteClarification:
		visible := a.visibleCandidates(ctx, pr, cls.Candidates)
		missing, cands := a.clarificationDetails(plan.Intent, visible)
		resp.Clarification = &contracts.Clarification{Missing: plan.Intent.Missing, Candidates: visible}
		vars["missing"], vars["candidates"], vars["options"] = missing, cands, a.intentOptions(append(cls.Ambiguous, cls.NeedTerm...))
		resp.Answer = a.render(plan.Template, vars)
		resp.Verdict = contracts.Verdict{Pass: true, Action: contracts.ActionClarify}
		resp.Clarification.Question = resp.Answer
	case contracts.RouteReject, contracts.RouteHumanEscalation:
		resp.Answer = a.render(plan.Template, vars)
		resp.Verdict = contracts.Verdict{Pass: true, Action: contracts.ActionAccept}
	case contracts.RouteSQLQuery, contracts.RouteCalculation, contracts.RouteHybrid:
		if err := a.answerMetric(ctx, tr, now, &plan, in, plan.Intent, &resp, vars); err != nil {
			return Outcome{}, err
		}
		// HYBRID: số liệu từ fact (đã kiểm) + phần quy trình từ tài liệu. Phần số liệu không ra
		// được câu trả lời thì không nối thêm tài liệu vào.
		if plan.Route == contracts.RouteHybrid && resp.Outcome == contracts.OutcomeResolved {
			kdef, _ := a.RT.Intents.Def(cls.Knowledge)
			if err := a.answerKnowledge(ctx, tr, now, pr, in, kdef.Template, &resp, true); err != nil {
				return Outcome{}, err
			}
		}
	case contracts.RouteRAG:
		if err := a.answerKnowledge(ctx, tr, now, pr, in, plan.Template, &resp, false); err != nil {
			return Outcome{}, err
		}
	default:
		// DIRECT_LLM, TOOL, WORKFLOW, AGENT cần capability chưa nối vào luồng này. Báo đúng là
		// lỗi cấu hình/mã thay vì giả vờ trả lời.
		return Outcome{}, contracts.Errorf(contracts.FaultCode, fmt.Sprintf("route %s chưa có capability trong luồng hỏi", plan.Route))
	}
	return Outcome{
		Response: resp, Plan: plan, Decision: a.decision(pr, now, req.Text, in, res, plan.Intent, resp),
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
	in contracts.NormalizedInput, it contracts.Intent, resp *contracts.AskResponse, vars map[string]string) error {

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
	for _, e := range result.Evidence {
		resp.Citations = append(resp.Citations, contracts.Citation{EvidenceID: e.ID, Source: e.Source})
	}

	vars["metric_label"] = a.metricLabel(metric)
	vars["entity_name"], vars["entity_key"] = entitySlot.Entity.Name, entitySlot.Entity.Key
	vars["period"] = displayPeriod(from, to)

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
	if claimed, ok := a.claimedValue(in, *entitySlot.Entity, metric); ok && compare == "" {
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
	// S8 — dựng câu bằng mẫu của tenant (route số liệu không gọi model)
	draft := contracts.Draft{Text: a.render(tmpl, vars), FromTemplate: true, Claims: claims}

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
func (a *Ask) answerKnowledge(ctx context.Context, tr *traces.Trace, now time.Time, pr contracts.Principal,
	in contracts.NormalizedInput, tmpl string, resp *contracts.AskResponse, hybrid bool) error {

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
	ev, err := traces.Measure(tr, "retrieve", func() ([]contracts.Evidence, error) {
		return a.Deps.Knowledge.Retrieve(ctx, pr, in.Raw, knowledgeK)
	})
	if err != nil {
		return err
	}
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
		return err
	}
	// S9 — kiểm
	violations, _ := traces.Measure(tr, "validate", func() ([]contracts.Violation, error) {
		v := validation.Bound(draft, ev)
		v = append(v, validation.Fidelity(a.RT.Normalizer, draft, contracts.ContextPackage{Evidence: ev}, nil, now)...)
		v = append(v, validation.ForbidDimensions(a.RT.Normalizer, draft, a.factDimensions(), now)...)
		return append(v, validation.ForbidUnitWords(draft.Text, a.factUnitWords())...), nil
	})
	if len(violations) > 0 {
		fail("fallback_unverified", contracts.OutcomeSystemFailure, violations)
		return nil
	}
	cited := citations(draft, ev)
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

// claimedValue trả con số người hỏi đưa ra khi câu là câu XÁC NHẬN (có cụm confirm_markers của
// pack và đúng MỘT lượng ngoài tên thực thể), đã đổi về đơn vị chuẩn của chỉ số.
func (a *Ask) claimedValue(in contracts.NormalizedInput, ent contracts.EntityRef, metric string) (contracts.Value, bool) {
	marked := false
	for _, m := range a.RT.ConfirmMarkers {
		marked = marked || normalization.ContainsPhrase(in.Raw, m)
	}
	def, ok := a.RT.Metrics.Metric(metric)
	if !marked || !ok {
		return contracts.Value{}, false
	}
	var cands, focused []contracts.Value
	fs, fe, focus := intent.FocusClause(in.Raw, a.RT.ConfirmMarkers)
	for _, sp := range in.Spans {
		if sp.Kind != contracts.ValueQuantity && sp.Kind != contracts.ValueNumber {
			continue
		}
		if sp.Start < ent.End && sp.End > ent.Start { // "03" của "máy 03" là tên, không phải số
			continue
		}
		cands = append(cands, sp.Value)
		if focus && sp.Start >= fs && sp.End <= fe {
			focused = append(focused, sp.Value)
		}
	}
	if len(focused) > 0 { // câu nhiều mệnh đề: số được hỏi nằm trong mệnh đề chứa cụm xác nhận
		cands = focused
	}
	if len(cands) != 1 {
		return contracts.Value{}, false
	}
	v := cands[0]
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

func firstEntitySlot(it contracts.Intent) contracts.Slot {
	for _, s := range it.Slots {
		if s.Entity != nil {
			return s
		}
	}
	return contracts.Slot{Entity: &contracts.EntityRef{}}
}

// periodBounds lấy khoảng ngày của khe period (ngày hoặc khoảng ngày).
func periodBounds(it contracts.Intent, now time.Time) (time.Time, time.Time, error) {
	text := now.Format("2006-01-02")
	for _, s := range it.Slots {
		if s.Value != nil && (s.Value.Kind == contracts.ValueDate || s.Value.Kind == contracts.ValuePeriod) {
			text = s.Value.Text
			break
		}
	}
	fromS, toS, found := strings.Cut(text, "/")
	if !found {
		toS = fromS
	}
	from, err1 := time.ParseInLocation("2006-01-02", fromS, now.Location())
	to, err2 := time.ParseInLocation("2006-01-02", toS, now.Location())
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
func (a *Ask) visibleCandidates(ctx context.Context, pr contracts.Principal, cands []contracts.EntityRef) []contracts.EntityRef {
	var out []contracts.EntityRef
	for _, c := range cands {
		keys, err := entityresolution.Lineage(ctx, a.Deps.Directory, c.ID)
		if err == nil && a.RT.Policy.InScope(pr, keys) {
			out = append(out, c)
		}
	}
	return out
}

// clarificationDetails dựng phần "thiếu gì" và "ứng viên" cho câu hỏi lại.
func (a *Ask) clarificationDetails(it contracts.Intent, candidates []contracts.EntityRef) (string, string) {
	def, _ := a.RT.Intents.Def(it.Name)
	labels := map[string]string{"metric": "chỉ số cần hỏi"}
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
	return strings.Join(missing, ", "), candText
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
