// Package contracts chứa các kiểu dữ liệu dùng chung giữa mọi thành phần của EAIS.
//
// Package này chỉ chứa KIỂU, không chứa hành vi phụ thuộc hạ tầng, và không import
// package nào khác của EAIS — nhờ vậy mọi package đều dùng được nó mà không sinh
// vòng import. Interface (Normalizer, Router, Validator…) do bên dùng khai báo cạnh
// chỗ dùng, theo thành ngữ Go.
//
// Ba luật mà các kiểu ở đây phục vụ (xem docs/01-architecture.md):
//   - model không bao giờ là nguồn của một con số → giá trị luôn có kiểu (Value) và
//     luôn gắn với bằng chứng (Evidence) có nguồn;
//   - tenant lấy từ danh tính đã xác thực (Principal), không từ nội dung request;
//   - mọi lượt truy được: route, bằng chứng, phán quyết kiểm, phiên bản pack.
package contracts

import "time"

// TenantID là định danh tenant (UUID dạng chuỗi).
type TenantID string

// PrincipalKind phân biệt người dùng với tài khoản dịch vụ.
type PrincipalKind string

const (
	PrincipalUser    PrincipalKind = "user"
	PrincipalService PrincipalKind = "service"
)

// Principal là danh tính ĐÃ XÁC THỰC của bên gọi.
//
// Tenant của một lượt lấy từ đây chứ không từ body: tin `tenant_id` trong body nghĩa là
// ai tới được cổng cũng hỏi được dữ liệu của mọi tenant.
type Principal struct {
	ID       string        `json:"id"`
	TenantID TenantID      `json:"tenant_id"`
	Kind     PrincipalKind `json:"kind"`
	Roles    []string      `json:"roles"`
	// Attributes là thuộc tính ABAC (attribute-based access control — phân quyền theo
	// thuộc tính), ví dụ project=A, department=fuel.
	Attributes map[string]string `json:"attributes,omitempty"`
}

// HasRole báo principal có vai trò đó không.
func (p Principal) HasRole(role string) bool {
	for _, r := range p.Roles {
		if r == role {
			return true
		}
	}
	return false
}

// ---------------------------------------------------------------------------
// Giá trị có kiểu và kết quả chuẩn hoá
// ---------------------------------------------------------------------------

// ValueKind là kiểu ngữ nghĩa của một giá trị đã chuẩn hoá.
type ValueKind string

const (
	ValueNumber   ValueKind = "number"   // số trần, chưa biết đơn vị
	ValueQuantity ValueKind = "quantity" // số + đơn vị đo (L, m, m3, h…)
	ValueMoney    ValueKind = "money"    // số + tiền tệ
	ValueDate     ValueKind = "date"     // một ngày, dạng YYYY-MM-DD
	ValuePeriod   ValueKind = "period"   // một khoảng ngày [Text = "YYYY-MM-DD/YYYY-MM-DD"]
	ValueTime     ValueKind = "time"     // giờ trong ngày, dạng HH:MM
	ValueCode     ValueKind = "code"     // mã định danh (mã máy, mã phiếu, mã size…)
	ValueTerm     ValueKind = "term"     // thuật ngữ chuẩn của pack (viết tắt đã mở)
)

// Value là một giá trị đã chuẩn hoá.
//
// Number là chuỗi thập phân CHUẨN ("4000", "0.35", "1602000"), không phải float64:
// số tiền và số lượng phải so khớp tuyệt đối, còn float biến "1602000" thành một số
// có thể lệch ở chữ số cuối sau một phép nhân đơn vị.
type Value struct {
	Kind   ValueKind `json:"kind"`
	Number string    `json:"number,omitempty"`
	Unit   string    `json:"unit,omitempty"` // đơn vị chuẩn của pack (L, m, VND…)
	Text   string    `json:"text,omitempty"` // mã, ngày, khoảng ngày, giờ, thuật ngữ ở dạng chuẩn
	// Approx = người nhập ghi "khoảng/tầm/cỡ/~": đây là ƯỚC LƯỢNG, không phải số đo.
	// Ước lượng không được dùng làm số chính thức trong báo cáo.
	Approx bool `json:"approx,omitempty"`
}

// Span là một đoạn trong câu gốc đã được nhận ra và chuẩn hoá.
type Span struct {
	Kind  ValueKind `json:"kind"`
	Raw   string    `json:"raw"`   // nguyên văn trong NormalizedInput.Raw
	Start int       `json:"start"` // vị trí byte trong NormalizedInput.Raw
	End   int       `json:"end"`
	Value Value     `json:"value"`
	// Confidence = 1 khi luật khớp chắc chắn; < 1 khi có suy luận, ví dụ đơn vị lấy
	// từ thuật ngữ đứng cạnh ("dầu 4k" → lít).
	Confidence float64 `json:"confidence"`
	Rule       string  `json:"rule"` // luật sinh ra span — để truy khi chuẩn hoá sai
}

// Ambiguity là chỗ chuẩn hoá KHÔNG được tự quyết.
type Ambiguity struct {
	Raw        string  `json:"raw"`
	Candidates []Value `json:"candidates"`
	Reason     string  `json:"reason"`
}

// NormalizedInput là kết quả chặng S2.
type NormalizedInput struct {
	// Raw là nguyên văn sau khi gấp ký tự "nhìn giống" (toàn chiều rộng → ASCII, khoảng trắng lạ → dấu
	// cách, "ℓ" → "l", ký tự vô hình chen giữa chữ số bị bỏ) rồi NFC; vị trí của mọi span trỏ vào đây.
	Raw string `json:"raw"`
	// Text là dạng để so khớp: chữ thường, khoảng trắng gộp, giữ dấu tiếng Việt.
	Text string `json:"text"`
	// Folded là Text đã bỏ dấu tiếng Việt — khách gõ không dấu vẫn khớp từ khoá.
	Folded string `json:"folded"`
	// Canonical là Text với mỗi span thay bằng dạng chuẩn ("dầu 4k" → "dầu 4000 L").
	// Dùng cho truy vấn tìm kiếm để nạp và hỏi đối xứng.
	Canonical   string      `json:"canonical"`
	Spans       []Span      `json:"spans"`
	Ambiguities []Ambiguity `json:"ambiguities,omitempty"`
	// InvalidDates là nguyên văn các cụm có DẠNG ngày ("31/02", "32/10/2026", "31/09/26") nhưng không đọc được thành
	// một ngày trên lịch. Chúng KHÔNG sinh span ngày — nên nếu không ghi lại ở đây, câu hỏi ngày hỏng sẽ lặng lẽ
	// được hiểu như không nêu ngày và rơi về ngày mặc định (P5: không đoán im lặng).
	InvalidDates []string `json:"invalid_dates,omitempty"`
}

// SpansOf trả các span thuộc kiểu cho trước, theo thứ tự xuất hiện.
func (n NormalizedInput) SpansOf(kinds ...ValueKind) []Span {
	var out []Span
	for _, s := range n.Spans {
		for _, k := range kinds {
			if s.Kind == k {
				out = append(out, s)
				break
			}
		}
	}
	return out
}

// ---------------------------------------------------------------------------
// Thực thể
// ---------------------------------------------------------------------------

// EntityRef là một thực thể chuẩn đã nhận ra trong đầu vào.
type EntityRef struct {
	ID         string  `json:"id"`
	Type       string  `json:"type"` // loại do pack định nghĩa: project, equipment, team…
	Key        string  `json:"key"`  // khoá chuẩn trong tenant, ví dụ "EX-03"
	Name       string  `json:"name"`
	Mention    string  `json:"mention"` // cụm đã khớp, ở dạng khoá so khớp ("may 3")
	Confidence float64 `json:"confidence"`
	// Start, End: vị trí byte của cụm đã khớp trong văn bản NFC — cùng hệ toạ độ với Span, để
	// biết phần nào của câu đã "thuộc về" thực thể (ví dụ "03" trong "máy 03" không phải lượng).
	Start int `json:"start"`
	End   int `json:"end"`
}

// AmbiguousMention là cụm từ khớp nhiều thực thể mà không phân định được.
type AmbiguousMention struct {
	Mention    string      `json:"mention"`
	Type       string      `json:"type"`
	Candidates []EntityRef `json:"candidates"`
	// Start, End: vị trí byte của cụm trong NormalizedInput.Raw, cùng hệ toạ độ với EntityRef.Start/End và Span. Một thực
	// thể do router gỡ từ bí danh mơ hồ lấy từ Candidates nên KHÔNG mang vị trí; muốn biết phần nào của câu đã "thuộc về" một tên
	// thực thể (ví dụ "03" của "máy 03" không phải con số) thì phải dùng vị trí của cụm mơ hồ này.
	Start int `json:"start"`
	End   int `json:"end"`
}

// Resolution là kết quả chặng S3.
type Resolution struct {
	Entities  []EntityRef        `json:"entities"`
	Ambiguous []AmbiguousMention `json:"ambiguous,omitempty"`
}

// OfType trả thực thể đầu tiên thuộc một trong các loại cho trước.
func (r Resolution) OfType(types ...string) (EntityRef, bool) {
	for _, e := range r.Entities {
		for _, t := range types {
			if e.Type == t {
				return e, true
			}
		}
	}
	return EntityRef{}, false
}

// ---------------------------------------------------------------------------
// Ý định và route
// ---------------------------------------------------------------------------

// EntryType là lối vào của một yêu cầu (docs/01-architecture.md §2.1).
type EntryType string

const (
	EntryQuery   EntryType = "QUERY"
	EntryCommand EntryType = "COMMAND"
	EntryEvent   EntryType = "EVENT"
)

// RouteKind là một trong 12 đường xử lý (docs/CONVENTIONS.md §4.1).
type RouteKind string

const (
	RouteDirectLLM       RouteKind = "DIRECT_LLM"
	RouteSQLQuery        RouteKind = "SQL_QUERY"
	RouteFullTextSearch  RouteKind = "FULL_TEXT_SEARCH"
	RouteRAG             RouteKind = "RAG"
	RouteHybrid          RouteKind = "HYBRID_RETRIEVAL"
	RouteCalculation     RouteKind = "CALCULATION"
	RouteTool            RouteKind = "TOOL"
	RouteWorkflow        RouteKind = "WORKFLOW"
	RouteAgent           RouteKind = "AGENT"
	RouteClarification   RouteKind = "CLARIFICATION"
	RouteHumanEscalation RouteKind = "HUMAN_ESCALATION"
	RouteReject          RouteKind = "REJECT"
)

// AllRoutes liệt kê đủ 12 route — để kiểm cấu hình pack và để test đếm.
var AllRoutes = []RouteKind{
	RouteDirectLLM, RouteSQLQuery, RouteFullTextSearch, RouteRAG, RouteHybrid, RouteCalculation,
	RouteTool, RouteWorkflow, RouteAgent, RouteClarification, RouteHumanEscalation, RouteReject,
}

// Valid báo route có thuộc danh sách 12 route không.
func (r RouteKind) Valid() bool {
	for _, k := range AllRoutes {
		if r == k {
			return true
		}
	}
	return false
}

// NeedsModel báo route có cần model diễn đạt không. Route trả lời bằng mẫu câu của
// tenant thì không gọi model (luật cứng 1 của luồng hỏi).
func (r RouteKind) NeedsModel() bool {
	switch r {
	case RouteDirectLLM, RouteRAG, RouteHybrid, RouteAgent:
		return true
	default:
		return false
	}
}

// Slot là một khe đã điền của ý định.
type Slot struct {
	Name   string     `json:"name"`
	Entity *EntityRef `json:"entity,omitempty"`
	Value  *Value     `json:"value,omitempty"`
	Text   string     `json:"text,omitempty"`
}

// Intent là kết quả chặng S4.
type Intent struct {
	Name       string          `json:"name"`
	Confidence float64         `json:"confidence"`
	Slots      map[string]Slot `json:"slots,omitempty"`
	Missing    []string        `json:"missing,omitempty"` // khe bắt buộc còn thiếu
	Rule       string          `json:"rule"`              // luật nào đã khớp
}

// StepKind là loại một bước trong kế hoạch.
type StepKind string

const (
	StepTool      StepKind = "tool"
	StepRetrieve  StepKind = "retrieve"
	StepCalculate StepKind = "calculate"
	StepGenerate  StepKind = "generate"
)

// Step là một bước trong kế hoạch route.
type Step struct {
	Kind StepKind          `json:"kind"`
	Name string            `json:"name"`
	Args map[string]string `json:"args,omitempty"`
}

// RoutePlan là kết quả chặng S5.
type RoutePlan struct {
	Route  RouteKind `json:"route"`
	Intent Intent    `json:"intent"`
	Steps  []Step    `json:"steps,omitempty"`
	// Template là tên mẫu câu của tenant dùng khi route không cần model.
	Template string `json:"template,omitempty"`
	Reason   string `json:"reason"`
	// Outcome do router quyết khi kết thúc sớm (hỏi lại, từ chối); rỗng = chặng thực thi quyết.
	Outcome Outcome `json:"outcome,omitempty"`
	// Checked: các bậc kiểm router đã đi qua theo thứ tự, kể cả bậc đã chặn — đường quyết định ghi đúng lúc quyết định (R2-03), chỉ để
	// giải thích: không bậc nào đọc lại nó để quyết.
	Checked []RouteCheck `json:"checked,omitempty"`
	// Narrowed: lần router gỡ bí danh mơ hồ bằng ngữ cảnh (dự án đã nêu, phạm vi dữ liệu của người hỏi). Slot rỗng = không gỡ lần nào.
	Narrowed Narrowing `json:"narrowed,omitzero"`
}

// RouteCheck là MỘT bậc kiểm của router (RoutePlan.Checked). Bậc nào chưa được xét vì bậc trước đã chặn thì không có mặt.
type RouteCheck struct {
	Rule   string    `json:"rule"`             // tên bậc kiểm: "nhiều ý định cùng khớp", "capability của ý định", "phạm vi dữ liệu"…
	Route  RouteKind `json:"route"`            // route mà bậc này chọn khi nó chặn (hay khi nó áp dụng)
	Fired  bool      `json:"fired"`            // bậc này đã chặn / áp dụng — quyết định route
	Detail string    `json:"detail,omitempty"` // dữ kiện đã kiểm và kết quả của nó
}

// Cách router gỡ bí danh mơ hồ (Narrowing.By).
const (
	NarrowByProject = "project" // khe đã điền bằng một dự án: giữ ứng viên thuộc dự án đó
	NarrowByScope   = "scope"   // khe còn thiếu: giữ ứng viên nằm trong phạm vi dữ liệu của người hỏi
)

// Narrowing ghi một lần router gỡ bí danh mơ hồ: ứng viên TRƯỚC và SAU khi lọc, và thực thể được chọn.
type Narrowing struct {
	Slot   string   `json:"slot"`
	By     string   `json:"by"`
	Before []string `json:"before"`           // khoá các ứng viên của cụm mơ hồ trước khi lọc (không trùng, theo thứ tự gặp)
	After  []string `json:"after"`            // khoá các ứng viên còn lại sau khi lọc
	Chosen string   `json:"chosen,omitempty"` // khoá thực thể được chọn; rỗng = lọc xong vẫn không gỡ được
}

// Outcome là kết quả CÓ KIỂU của một lượt hỏi (docs/04 §4): client đọc trường này, không đoán
// từ câu chữ của mẫu.
type Outcome string

// Các kết quả. Xung đột giải được bằng thẩm quyền vẫn là RESOLVED (xung đột được liệt kê kèm);
// nguồn ngang thẩm quyền lệch nhau là HUMAN_REVIEW_REQUIRED.
const (
	OutcomeResolved              Outcome = "RESOLVED"
	OutcomeClarificationRequired Outcome = "CLARIFICATION_REQUIRED"
	OutcomeInsufficientEvidence  Outcome = "INSUFFICIENT_EVIDENCE"
	OutcomeSourceConflict        Outcome = "SOURCE_CONFLICT"
	OutcomePermissionDenied      Outcome = "PERMISSION_DENIED"
	OutcomeHumanReviewRequired   Outcome = "HUMAN_REVIEW_REQUIRED"
	OutcomeOutOfScope            Outcome = "OUT_OF_SCOPE"
	OutcomeSystemFailure         Outcome = "SYSTEM_FAILURE"
)

// ---------------------------------------------------------------------------
// Bằng chứng và ngữ cảnh
// ---------------------------------------------------------------------------

// EvidenceKind là loại một bằng chứng.
type EvidenceKind string

const (
	EvidenceFact        EvidenceKind = "fact"        // số liệu có kiểu từ nguồn có cấu trúc
	EvidencePassage     EvidenceKind = "passage"     // đoạn tài liệu
	EvidenceEvent       EvidenceKind = "event"       // sự kiện trong event store
	EvidenceCalculation EvidenceKind = "calculation" // kết quả phép tính tất định, có công thức
	EvidencePolicy      EvidenceKind = "policy"      // luật/chính sách của tenant
	EvidenceHistory     EvidenceKind = "history"     // tin trước trong hội thoại
	EvidenceQuestion    EvidenceKind = "question"    // con số do chính người hỏi đưa ra (câu xác nhận) — không phải số liệu
)

// SourceRef chỉ ra một bằng chứng đến từ đâu.
type SourceRef struct {
	SourceID        string `json:"source_id,omitempty"`
	SourceType      string `json:"source_type"` // warehouse_slip, site_report, sop, provider_feed…
	Name            string `json:"name"`
	DocumentID      string `json:"document_id,omitempty"`
	DocumentVersion int    `json:"document_version,omitempty"`
	URI             string `json:"uri,omitempty"`
}

// Evidence là đơn vị duy nhất đi vào ngữ cảnh, và là thứ duy nhất một câu trả lời
// được phép dựa vào.
type Evidence struct {
	ID     string       `json:"id"` // "E1", "E2"… ổn định trong một lượt
	Kind   EvidenceKind `json:"kind"`
	Text   string       `json:"text"`
	Values []Value      `json:"values,omitempty"` // giá trị có kiểu nằm trong bằng chứng — để kiểm số
	Source SourceRef    `json:"source"`
	// Authority là hạng thẩm quyền của nguồn trong CÙNG một chỉ số (nhỏ = mạnh hơn).
	Authority  int       `json:"authority"`
	ObservedAt time.Time `json:"observed_at"`
	Confidence float64   `json:"confidence"`
	// Formula chỉ có ở bằng chứng phép tính: công thức + mã bằng chứng đầu vào.
	Formula string `json:"formula,omitempty"`
	// Subject, Metric, Period: chủ thể (id thực thể) / chỉ số / kỳ ("YYYY-MM-DD" hoặc
	// "YYYY-MM-DD/YYYY-MM-DD") của bằng chứng số liệu — để ràng buộc claim đủ bộ (G5). Trống ở
	// bằng chứng văn bản.
	Subject string `json:"subject,omitempty"`
	Metric  string `json:"metric,omitempty"`
	Period  string `json:"period,omitempty"`
}

// Conflict là chỗ các nguồn cùng chỉ số nói khác nhau.
type Conflict struct {
	Key         string   `json:"key"` // chỉ số|thực thể|kỳ
	EvidenceIDs []string `json:"evidence_ids"`
	Policy      string   `json:"policy"`    // chính sách đã áp
	ChosenID    string   `json:"chosen_id"` // bằng chứng được chọn (rỗng nếu chờ người duyệt)
	NeedsReview bool     `json:"needs_review"`
	Reason      string   `json:"reason"`
	// Others: giá trị của các nguồn KHÔNG được chọn (có cấu trúc, để câu trả lời kê có nhãn).
	Others []SourceValue `json:"others,omitempty"`
}

// SourceValue là giá trị một loại nguồn báo cho một chỉ số.
type SourceValue struct {
	SourceType string `json:"source_type"`
	Number     string `json:"number"`
	Unit       string `json:"unit"`
	// EvidenceIDs: các fact của nguồn này cộng lại thành Number.
	EvidenceIDs []string `json:"evidence_ids,omitempty"`
}

// Dropped ghi bằng chứng bị loại khỏi ngữ cảnh và vì sao.
type Dropped struct {
	EvidenceID string `json:"evidence_id"`
	Reason     string `json:"reason"`
}

// ContextPackage là kết quả chặng S7 — toàn bộ thứ model hoặc mẫu câu được nhìn thấy.
type ContextPackage struct {
	Evidence  []Evidence `json:"evidence"`
	Conflicts []Conflict `json:"conflicts,omitempty"`
	Dropped   []Dropped  `json:"dropped,omitempty"`
	Chars     int        `json:"chars"`
}

// ByID tìm bằng chứng theo mã.
func (p ContextPackage) ByID(id string) (Evidence, bool) {
	for _, e := range p.Evidence {
		if e.ID == id {
			return e, true
		}
	}
	return Evidence{}, false
}

// ---------------------------------------------------------------------------
// Sinh câu trả lời và kiểm
// ---------------------------------------------------------------------------

// Usage là lượng token một lời gọi model đã dùng.
type Usage struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
	CacheRead    int `json:"cache_read"`
	CacheWrite   int `json:"cache_write"`
}

// Claim là một khẳng định trong câu trả lời kèm bằng chứng đỡ nó.
type Claim struct {
	Text        string   `json:"text"`
	EvidenceIDs []string `json:"evidence_ids"`
	// Ràng buộc đủ bộ của MỘT con số trong câu (G5): chủ thể / chỉ số / kỳ / giá trị (kèm đơn vị)
	// / nguồn. Claim chỉ trích văn bản tài liệu thì để trống phần này.
	Subject string `json:"subject,omitempty"`
	Metric  string `json:"metric,omitempty"`
	Period  string `json:"period,omitempty"`
	Value   *Value `json:"value,omitempty"`
	Source  string `json:"source,omitempty"` // loại nguồn: warehouse_slip, erp, calculation, question…
}

// Draft là câu trả lời CHƯA được kiểm.
type Draft struct {
	Text   string  `json:"text"`
	Claims []Claim `json:"claims,omitempty"`
	Model  string  `json:"model,omitempty"`
	Usage  Usage   `json:"usage"`
	// FromTemplate = câu dựng bằng mẫu câu của tenant, không qua model.
	FromTemplate bool `json:"from_template"`
}

// Severity là mức nghiêm trọng (docs/CONVENTIONS.md §4.4).
type Severity string

const (
	SeverityS1 Severity = "S1"
	SeverityS2 Severity = "S2"
	SeverityS3 Severity = "S3"
	SeverityS4 Severity = "S4"
)

// Violation là một vi phạm validator phát hiện.
type Violation struct {
	Validator string   `json:"validator"`
	Code      string   `json:"code"`
	Severity  Severity `json:"severity"`
	Detail    string   `json:"detail"`
}

// Action là việc làm sau khi kiểm.
type Action string

const (
	ActionAccept   Action = "accept"
	ActionFallback Action = "fallback" // trả mẫu câu "chưa đủ dữ liệu chính xác" của tenant
	ActionClarify  Action = "clarify"
	ActionEscalate Action = "escalate"
)

// Verdict là kết quả chặng S9.
type Verdict struct {
	Pass       bool        `json:"pass"`
	Violations []Violation `json:"violations,omitempty"`
	Action     Action      `json:"action"`
}

// ---------------------------------------------------------------------------
// Hợp đồng HTTP của lối vào QUERY
// ---------------------------------------------------------------------------

// AskRequest là một câu hỏi. Không có tenant ở đây — tenant lấy từ Principal.
type AskRequest struct {
	ConversationID string `json:"conversation_id"`
	MessageID      string `json:"message_id"`
	Text           string `json:"text"`
}

// Citation trỏ một câu trả lời về nguồn.
type Citation struct {
	EvidenceID string    `json:"evidence_id"`
	Source     SourceRef `json:"source"`
}

// Clarification là câu hỏi lại khi đầu vào thiếu/mơ hồ.
type Clarification struct {
	Question   string      `json:"question"`
	Missing    []string    `json:"missing,omitempty"`
	Candidates []EntityRef `json:"candidates,omitempty"`
}

// AskResponse là kết quả một lượt hỏi.
type AskResponse struct {
	Answer        string         `json:"answer"`
	Outcome       Outcome        `json:"outcome"`
	Route         RouteKind      `json:"route"`
	Intent        string         `json:"intent"`
	Citations     []Citation     `json:"citations,omitempty"`
	Conflicts     []Conflict     `json:"conflicts,omitempty"`
	Clarification *Clarification `json:"clarification,omitempty"`
	Verdict       Verdict        `json:"verdict"`
	PackVersion   string         `json:"pack_version"`
	TraceID       string         `json:"trace_id"`
}
