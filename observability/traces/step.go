package traces

import (
	"fmt"
	"maps"
	"slices"
	"strconv"
	"strings"

	"github.com/luannt2002/enterprise-ai-control-layer/core/contracts"
)

// Mã bước. Luồng hỏi S0–S10 theo CLAUDE.md §11; tầng nạp dùng tiền tố "L1." (docs/PROMPT-BUILD-EAIS.md §9.4).
const (
	StageAccept    = "S0"
	StageTenant    = "S1"
	StageNormalize = "S2"
	StageResolve   = "S3"
	StageIntent    = "S4"
	StageRoute     = "S5"
	StageExecute   = "S6"
	StageContext   = "S7"
	StageGenerate  = "S8"
	StageValidate  = "S9"
	StageRespond   = "S10"
	// StageUnknown: lượt lỗi mà luồng không cho biết chặng nào lỗi (Handle trả Outcome rỗng khi lỗi).
	StageUnknown = "S?"
)

// Các bước của tầng nạp. Step dùng chung cho hai luồng; BuildSteps chỉ dựng luồng hỏi, mã nạp tự dựng bước L1.*.
const (
	StageIngestRead       = "L1.read"
	StageIngestNormalize  = "L1.normalize"
	StageIngestEntity     = "L1.entity"
	StageIngestSemantic   = "L1.semantic"
	StageIngestTime       = "L1.time"
	StageIngestDomain     = "L1.domain"
	StageIngestQuarantine = "L1.quarantine"
	StageIngestReconcile  = "L1.reconcile"
)

// Trạng thái của một bước.
const (
	StatusOK      = "ok"      // chạy, ra kết quả thẳng
	StatusSkipped = "skipped" // không chạy ở lượt này (route không đi qua bước đó)
	StatusDeviant = "deviant" // chạy nhưng kết quả lệch đường thẳng, cần người xem: mơ hồ, hỏi lại, xung đột, kiểm trượt…
	StatusError   = "error"   // bước báo lỗi (Fault + Error)
)

// NotRecorded là giá trị của trường "lý do" / "phương án bị loại" mà luồng hỏi chưa xuất ra ngoài Handle. Ghi thẳng điều này thay
// vì suy đoán: một lý do bịa trong vết còn tệ hơn một chỗ trống ghi rõ là trống. Nội dung sau dấu hai chấm nói chỗ cần móc.
// Các chặng S2, S3, S4, S5, S7 và cổng phán quyết ở S9 đã được móc (Decision.Why, R2-03) nên lượt chạy thật không còn chỗ trống ở đó;
// NotRecorded còn ở bản dựng tay / lượt cũ chưa mang Why, và ở các chỗ chưa móc: S6 khi không xung đột, bản nháp bị loại ở S9.
const NotRecorded = "chưa ghi — cần móc trong core/pipeline (làn 1)"

func notRecorded(what string) string { return NotRecorded + ": " + what }

// IsNotRecorded báo một giá trị có phải chỗ trống ghi rõ của NotRecorded không.
func IsNotRecorded(s string) bool { return strings.HasPrefix(s, NotRecorded) }

// ValidStage báo mã bước có hợp lệ không: S0…S10 (luồng hỏi), C0…C7 và C4b (luồng lệnh, HOP-DONG §7.5), S? hoặc L1.<tên>.
func ValidStage(s string) bool {
	if s == StageUnknown {
		return true
	}
	if rest, ok := strings.CutPrefix(s, "C"); ok { // luồng lệnh: chín bước, C4b nằm giữa C4 và C5
		return s == "C4b" || (len(rest) == 1 && rest[0] >= '0' && rest[0] <= '7')
	}
	if rest, ok := strings.CutPrefix(s, "L1."); ok {
		return rest != ""
	}
	if rest, ok := strings.CutPrefix(s, "S"); ok {
		n, err := strconv.Atoi(rest)
		return err == nil && n >= 0 && n <= 10 && strconv.Itoa(n) == rest
	}
	return false
}

// EvidenceRef là một bằng chứng kèm nguồn gốc (P4: sự thật có nguồn).
type EvidenceRef struct {
	ID     string              `json:"id"`
	Source contracts.SourceRef `json:"source"`
}

// Step là vết của MỘT bước trong một lượt (docs/PROMPT-BUILD-EAIS.md §9.4).
type Step struct {
	TraceID string `json:"trace_id"`
	Seq     int    `json:"seq"`   // thứ tự trong lượt, từ 0
	Stage   string `json:"stage"` // S0…S10, C0…C7, C4b, S? hoặc L1.<tên>
	Name    string `json:"name"`
	Status  string `json:"status"`
	// Input, Output: vào / ra của bước ở dạng chuỗi phẳng, ổn định (so bằng chuỗi được).
	Input  []string `json:"input,omitempty"`
	Output []string `json:"output,omitempty"`
	// Decision: bước đã quyết gì.
	Decision string `json:"decision"`
	// Reason: LÝ DO — luật / từ khoá / cấu hình nào. Luồng chưa xuất thì mang giá trị NotRecorded.
	Reason string `json:"reason"`
	// Rejected: phương án đã xét và bị loại. Rỗng = bước không có phương án nào để loại; chưa xuất thì mang NotRecorded.
	Rejected []string      `json:"rejected,omitempty"`
	Evidence []EvidenceRef `json:"evidence,omitempty"`
	// Phiên bản pack / mã / dữ liệu lúc bước chạy: phát lại và so hai mốc chỉ có nghĩa trên đúng các phiên bản này.
	PackVersion string `json:"pack_version"`
	CodeVersion string `json:"code_version"`
	DataVersion string `json:"data_version"`
	// Ms chỉ có nghĩa khi Timed: bước không có đồng hồ riêng (S0, S1, S7, S10…) thì "chưa đo", không phải 0 ms.
	Ms    int64  `json:"ms"`
	Timed bool   `json:"timed"`
	Fault string `json:"fault,omitempty"`
	Error string `json:"error,omitempty"`
}

// Turn gom những gì MỘT lượt hỏi đã để lại NGOÀI core/pipeline (các trường của pipeline.Outcome). Package này không import
// pipeline (pipeline import traces) nên nơi gọi tự điền: Turn{Decision: out.Decision, Spans: out.Spans, TotalMs: out.TotalMs,
// UnaccountedMs: out.UnaccountedMs, Plan: out.Plan, Response: out.Response}.
type Turn struct {
	Decision      Decision
	Spans         []Span
	TotalMs       int64
	UnaccountedMs int64
	Plan          contracts.RoutePlan
	Response      contracts.AskResponse
}

// stageSpans là bảng "bước → tên span của pipeline": MỘT nguồn sự thật cho BuildSteps (thời gian của bước) và cho test. Span mà
// pipeline thêm sau này mà chưa có tên ở đây thì không gắn vào bước nào (thời gian vẫn nằm nguyên trong Record.Spans);
// TestBuocTuLuotThat bắt chuyện đó bằng SpanStage trên mọi span của lượt thật.
var stageSpans = map[string][]string{
	StageNormalize: {"normalize"}, StageResolve: {"resolve"}, StageIntent: {"intent"}, StageRoute: {"route"},
	StageExecute: {"execute", "retrieve"}, StageGenerate: {"generate"}, StageValidate: {"validate"},
}

// SpanStage trả bước mà BuildSteps gắn span mang tên name vào; ok = false khi tên span lạ (không bước nào nhận).
func SpanStage(name string) (stage string, ok bool) {
	for st, names := range stageSpans {
		if slices.Contains(names, name) {
			return st, true
		}
	}
	return "", false
}

// BuildSteps dựng 11 bước S0…S10 của một lượt hỏi TỪ những gì Turn có — chỉ ghép, không suy thêm. Giải thích của S2, S3, S4, S5, S7, S8 (bản nháp
// như được sinh), S10 (dấu vân tay câu giao) và cổng phán quyết ở S9 lấy nguyên văn từ Decision.Why do luồng hỏi ghi tại chỗ quyết định. Thứ gì luồng hỏi chưa xuất ra (kết quả truy
// vấn khi không xung đột, bản nháp của model…) thì Reason / Rejected mang NotRecorded, kèm chỗ cần móc. Bước không nằm trên đường đi của
// route (không có span tương ứng) có Status skipped.
// Thời gian của bước lấy từ span cùng chặng theo bảng stageSpans: normalize→S2, resolve→S3, intent→S4, route→S5, execute và
// retrieve→S6, generate→S8, validate→S9 (tổng khi một tên span xuất hiện nhiều lần, ví dụ route HYBRID); span lạ không gắn vào bước
// nào nhưng vẫn nằm nguyên trong Record.Spans.
//
// Hai chỗ suy ra TỪ SPAN VÀ OUTCOME (không bịa, chỉ đọc những gì Handle đã xuất):
//   - S8 chỉ là "model diễn đạt" khi có span generate. Route cần model mà không có span generate là lượt kết thúc sớm (câu hỏi đòi
//     lượng nghiệp vụ, truy xuất rỗng, phần số liệu không ra dữ liệu…): model chưa được gọi, vết không được nói ngược lại;
//   - S6 là bước lệch khi truy vấn chạy xong mà outcome là INSUFFICIENT_EVIDENCE (nguyên nhân nằm ở truy vấn, S10 chỉ là hệ quả), và
//     khi HYBRID có truy xuất mà không có generate (phần tri thức không ra đoạn nào, outcome vẫn RESOLVED).
func BuildSteps(t Turn) []Step {
	d, plan, resp := t.Decision, t.Plan, t.Response
	spanMs := map[string]int64{}
	for _, sp := range t.Spans {
		spanMs[sp.Name] += sp.Ms
	}
	has := func(name string) bool { _, ok := spanMs[name]; return ok }
	route := contracts.RouteKind(d.Route)
	executed := has("execute") || has("retrieve")

	steps := make([]Step, 0, 11)
	add := func(stage, name string, fill func(s *Step)) {
		s := Step{TraceID: d.TraceID, Seq: len(steps), Stage: stage, Name: name, Status: StatusOK,
			PackVersion: d.PackVersion, CodeVersion: d.CodeVersion, DataVersion: d.DataVersion}
		for _, n := range stageSpans[stage] {
			if ms, ok := spanMs[n]; ok {
				s.Ms, s.Timed = s.Ms+ms, true
			}
		}
		fill(&s)
		steps = append(steps, s)
	}
	skip := func(s *Step, why string) {
		s.Status, s.Decision, s.Reason = StatusSkipped, "không chạy ở lượt này", why
	}

	add(StageAccept, "accept", func(s *Step) {
		s.Input = []string{"question=" + d.Question}
		s.Output = []string{"accepted"}
		s.Decision = "accept"
		s.Reason = "đầu vào hợp lệ: lượt chạy tiếp được tới S2"
	})
	add(StageTenant, "tenant", func(s *Step) {
		s.Input = []string{"principal=" + d.Principal, "kind=" + d.Kind, "roles=" + strings.Join(d.Roles, ",")}
		for _, k := range slices.Sorted(maps.Keys(d.Attributes)) {
			s.Input = append(s.Input, "attr."+k+"="+d.Attributes[k])
		}
		s.Output = []string{"tenant=" + d.Tenant}
		s.Decision = "accept"
		s.Reason = "tenant lấy từ principal đã xác thực và khớp tenant của runtime; không lấy từ thân yêu cầu"
	})
	add(StageNormalize, "normalize", func(s *Step) {
		s.Input = []string{"question=" + d.Question}
		s.Output = slices.Clone(d.Normalized)
		s.Decision = fmt.Sprintf("%d đoạn đã chuẩn hoá", len(d.Normalized))
		if n := d.Why.Normalize; n.Recorded() {
			s.Reason, s.Rejected = joinReasons(n), slices.Clone(n.Rejected)
		} else {
			s.Reason = notRecorded("luật sinh từng đoạn (Span.Rule) chưa xuất khỏi Handle")
			s.Rejected = []string{notRecorded("cách đọc khác của chỗ mơ hồ (NormalizedInput.Ambiguities) chưa xuất khỏi Handle")}
		}
		if slices.ContainsFunc(d.Normalized, func(x string) bool { return strings.HasSuffix(x, "=invalid") }) {
			s.Status = StatusDeviant // có ngày hỏng: không đoán im lặng
		}
	})
	add(StageResolve, "resolve", func(s *Step) {
		s.Input = slices.Clone(d.Normalized)
		s.Output = slices.Clone(d.Entities)
		ambiguous := 0
		for _, e := range d.Entities {
			if strings.HasPrefix(e, "?") {
				ambiguous++
			}
		}
		s.Decision = fmt.Sprintf("%d thực thể rõ, %d cụm mơ hồ", len(d.Entities)-ambiguous, ambiguous)
		if n := d.Why.Resolve; n.Recorded() {
			s.Reason, s.Rejected = joinReasons(n), slices.Clone(n.Rejected)
		} else {
			s.Reason = notRecorded("bí danh / khoá khớp từng thực thể chưa xuất khỏi Handle")
			s.Rejected = []string{notRecorded("ứng viên bị loại của entityresolution.Resolve chưa xuất khỏi Handle (cụm mơ hồ ghi ở Output dạng ?id|id)")}
		}
		if ambiguous > 0 {
			s.Status = StatusDeviant
		}
	})
	add(StageIntent, "intent", func(s *Step) {
		s.Input = slices.Clone(d.Entities)
		s.Output = append([]string{"intent=" + d.Intent}, d.Slots...)
		s.Decision = "không nhận ra ý định"
		if d.Intent != "" {
			s.Decision = "intent " + d.Intent
			if plan.Intent.Name == d.Intent {
				s.Decision += fmt.Sprintf(" (độ tin cậy %.2f)", plan.Intent.Confidence)
			}
		}
		if n := d.Why.Intent; n.Recorded() {
			s.Reason, s.Rejected = joinReasons(n), slices.Clone(n.Rejected)
		} else {
			s.Reason = orNotRecorded(plan.Intent.Rule, "RoutePlan.Intent.Rule rỗng")
			s.Rejected = []string{notRecorded("ý định ứng viên khác (intent.Result.Candidates / Ambiguous) chưa xuất khỏi Handle")}
		}
		if d.Intent == "" || slices.ContainsFunc(d.Slots, func(x string) bool { return strings.HasSuffix(x, "=?") }) {
			s.Status = StatusDeviant // chưa có ý định, hoặc còn khe bắt buộc thiếu
		}
	})
	add(StageRoute, "route", func(s *Step) {
		s.Input = []string{"intent=" + d.Intent}
		s.Output = []string{"route=" + d.Route}
		if plan.Template != "" {
			s.Output = append(s.Output, "template="+plan.Template)
		}
		for _, st := range plan.Steps {
			s.Output = append(s.Output, "step "+stepText(st))
		}
		s.Decision = "route " + d.Route
		if n := d.Why.Route; n.Recorded() {
			s.Reason, s.Rejected = joinReasons(n), slices.Clone(n.Rejected)
		} else {
			s.Reason = orNotRecorded(plan.Reason, "RoutePlan.Reason rỗng")
			s.Rejected = []string{notRecorded("route khác router đã xét chưa xuất khỏi Handle")}
		}
		switch route {
		case contracts.RouteClarification, contracts.RouteReject, contracts.RouteHumanEscalation:
			s.Status = StatusDeviant // kết thúc sớm: hỏi lại, từ chối hoặc chuyển người
		}
	})
	add(StageExecute, "execute", func(s *Step) {
		if !executed {
			reason := "không có span execute / retrieve: route " + d.Route + " không truy vấn nguồn"
			if route.NeedsModel() { // route tri thức luôn truy xuất: không có span nghĩa là lượt thoát sớm TRƯỚC truy xuất, không phải S6 sai
				reason = "không có span execute / retrieve: route " + d.Route + " cần truy xuất nhưng lượt thoát sớm trước truy xuất (lý do ở S8)"
			}
			skip(s, reason)
			return
		}
		s.Input = []string{"route=" + d.Route}
		s.Decision = "đã truy vấn nguồn"
		if len(resp.Conflicts) == 0 {
			s.Reason = notRecorded("kết quả truy vấn (sourceoftruth.Result) chưa xuất khỏi Handle")
			s.Rejected = []string{notRecorded("nguồn / bằng chứng bị loại lúc truy vấn chưa xuất khỏi Handle")}
		} else {
			s.Status = StatusDeviant // nguồn cùng chỉ số nói khác nhau
			var reasons []string
			for _, c := range resp.Conflicts {
				chosen := "chưa chọn, chờ người duyệt"
				if c.ChosenID != "" {
					chosen = "chọn " + c.ChosenID
				}
				s.Output = append(s.Output, fmt.Sprintf("xung đột %s: %s theo %s", c.Key, chosen, c.Policy))
				reasons = append(reasons, c.Key+": "+c.Reason)
				for _, id := range c.EvidenceIDs {
					if id != c.ChosenID {
						s.Rejected = append(s.Rejected, fmt.Sprintf("bằng chứng %s (xung đột %s)", id, c.Key))
					}
				}
				for _, o := range c.Others {
					s.Rejected = append(s.Rejected, fmt.Sprintf("nguồn %s = %s %s (xung đột %s)", o.SourceType, o.Number, o.Unit, c.Key))
				}
			}
			s.Reason = strings.Join(reasons, "; ")
		}
		// Truy vấn chạy xong mà không đủ dữ liệu để trả lời: nguyên nhân của outcome INSUFFICIENT_EVIDENCE nằm Ở ĐÂY, S10 chỉ là hệ
		// quả — bước lệch đầu tiên của vết phải là chặng đã gây ra nó (R1-03). Khi luồng hỏi xuất sourceoftruth.Result thì ghi hẳn
		// "truy vấn không trả dữ liệu" thay cho suy ra từ outcome (làn 1).
		if d.Outcome == string(contracts.OutcomeInsufficientEvidence) {
			s.Status = StatusDeviant
			s.Decision = "đã truy vấn nguồn nhưng không đủ dữ liệu để trả lời"
			s.Output = append(s.Output, fmt.Sprintf("không đủ dữ liệu: outcome %s, %d bằng chứng được trích dẫn", d.Outcome, len(d.Evidence)))
		}
		// HYBRID: phần tri thức nối sau câu số liệu đã xong. Có truy xuất mà không có span generate nghĩa là truy xuất không trả đoạn
		// nào người hỏi được đọc — phần quy trình bị bỏ khỏi câu mà outcome vẫn RESOLVED, nên chỉ chặng này cho thấy.
		if route == contracts.RouteHybrid && has("retrieve") && !has("generate") {
			s.Status = StatusDeviant
			s.Output = append(s.Output, "tri thức: truy xuất không trả đoạn nào người hỏi được đọc; phần quy trình không được nối vào câu")
		}
	})
	add(StageContext, "context", func(s *Step) {
		if !executed {
			skip(s, "không có chặng truy vấn nên không dựng ngữ cảnh")
			return
		}
		s.Output = slices.Clone(d.Evidence)
		for _, c := range resp.Citations {
			s.Evidence = append(s.Evidence, EvidenceRef{ID: c.EvidenceID, Source: c.Source})
		}
		s.Decision = fmt.Sprintf("%d bằng chứng được trích dẫn", len(d.Evidence))
		if n := d.Why.Context; n.Recorded() {
			s.Reason, s.Rejected = joinReasons(n), slices.Clone(n.Rejected)
		} else {
			s.Reason = notRecorded("lý do chọn / loại bằng chứng (ContextPackage.Dropped) chưa xuất khỏi Handle")
			s.Rejected = []string{notRecorded("bằng chứng bị loại khỏi ngữ cảnh (ContextPackage.Dropped) chưa xuất khỏi Handle")}
		}
	})
	add(StageGenerate, "generate", func(s *Step) {
		switch {
		case !route.Valid():
			skip(s, "route trống hoặc không hợp lệ nên không có cách dựng câu")
		case route.NeedsModel() && has("generate"):
			s.Decision = "model diễn đạt"
			if gen := d.Why.Generate; gen.Recorded() {
				s.Reason, s.Rejected = joinReasons(gen), slices.Clone(gen.Rejected)
			} else {
				s.Reason = notRecorded("model, bản nháp và usage (contracts.Draft) chưa xuất khỏi Handle")
				s.Rejected = []string{notRecorded("bản nháp bị loại chưa xuất khỏi Handle")}
			}
		case route.NeedsModel():
			// route cần model mà pipeline không gọi Model.Generate (không có span generate): lượt kết thúc sớm — câu hỏi đòi lượng nghiệp
			// vụ, truy xuất rỗng, phần số liệu không ra dữ liệu… Vết không được kể là model đã diễn đạt (R1-02): P10 dựa vào chặng MODEL
			// để chọn đúng miền lỗi, đừng đổi model khi lỗi nằm ở dữ liệu.
			s.Status, s.Decision = StatusSkipped, "model chưa được gọi"
			s.Reason = "route " + d.Route + " cần model nhưng không có span generate: lượt kết thúc sớm với outcome " + d.Outcome + ", câu dựng bằng mẫu của tenant; "
			if gen := d.Why.Generate; gen.Recorded() { // lý do thoát, kết cục mà nhánh chọn, mẫu THẬT và bản nháp: do luồng hỏi ghi tại chỗ thoát
				s.Reason += joinReasons(gen)
				s.Rejected = slices.Clone(gen.Rejected)
			} else {
				s.Reason += notRecorded("lý do thoát sớm và tên mẫu chưa xuất khỏi Handle")
			}
		default:
			s.Decision = "mẫu câu của tenant " + plan.Template
			s.Reason = "route " + d.Route + " không cần model (RouteKind.NeedsModel = false): dựng bằng mẫu câu, không gọi model"
			if gen := d.Why.Generate; gen.Recorded() {
				s.Reason += "; " + joinReasons(gen)
				s.Rejected = slices.Clone(gen.Rejected)
			}
		}
	})
	add(StageValidate, "validate", func(s *Step) {
		s.Output = []string{"verdict=" + d.Verdict}
		if !has("validate") {
			s.Status, s.Decision = StatusSkipped, "verdict "+d.Verdict+" do route gán"
			s.Reason = "không có span validate: chặng kiểm không chạy ở route " + d.Route
			return
		}
		s.Decision = "verdict " + d.Verdict
		var vs []string
		for _, v := range resp.Verdict.Violations {
			vs = append(vs, fmt.Sprintf("%s/%s [%s]: %s", v.Validator, v.Code, v.Severity, v.Detail))
		}
		s.Reason = "không validator nào báo vi phạm"
		if len(vs) > 0 {
			s.Reason = strings.Join(vs, "; ")
		}
		// cổng phán quyết của câu xác nhận (verdictGap) chạy trước khi dựng câu: phán quyết hay không và vì sao
		if n := d.Why.Validate; n.Recorded() {
			s.Reason += "; " + joinReasons(n)
			s.Rejected = slices.Clone(n.Rejected)
		}
		if !resp.Verdict.Pass {
			s.Rejected = append(s.Rejected, notRecorded("bản nháp bị loại khi kiểm trượt chưa xuất khỏi Handle"))
		}
		if !resp.Verdict.Pass || resp.Verdict.Action == contracts.ActionFallback || resp.Verdict.Action == contracts.ActionEscalate {
			s.Status = StatusDeviant
		}
	})
	add(StageRespond, "respond", func(s *Step) {
		s.Input = []string{"verdict=" + d.Verdict}
		s.Output = []string{d.Answer}
		s.Decision = "outcome " + d.Outcome
		s.Reason = "outcome do chặng thực thi và kiểm quyết (RoutePlan.Outcome rỗng)"
		if plan.Outcome != "" {
			s.Reason = "router kết thúc sớm với outcome " + string(plan.Outcome)
		}
		if rep := d.Why.Respond; rep.Recorded() { // dấu vân tay của câu giao, đối chiếu với bản nháp ở S8
			s.Reason += "; " + joinReasons(rep)
			s.Rejected = slices.Clone(rep.Rejected)
		}
		if d.Outcome != string(contracts.OutcomeResolved) {
			s.Status = StatusDeviant
		}
	})
	return steps
}

// joinReasons nối các ý của Notes.Reasons thành lý do MỘT dòng của bước (cùng quy ước "; " với lý do của S6 / S9).
func joinReasons(n Notes) string { return strings.Join(n.Reasons, "; ") }

func orNotRecorded(v, what string) string {
	if v != "" {
		return v
	}
	return notRecorded(what)
}

// stepText in một bước kế hoạch của router: kind:name{khoá=giá trị,…} với khoá xếp tăng dần.
func stepText(st contracts.Step) string {
	s := string(st.Kind) + ":" + st.Name
	if len(st.Args) == 0 {
		return s
	}
	var args []string
	for _, k := range slices.Sorted(maps.Keys(st.Args)) {
		args = append(args, k+"="+st.Args[k])
	}
	return s + "{" + strings.Join(args, ",") + "}"
}
