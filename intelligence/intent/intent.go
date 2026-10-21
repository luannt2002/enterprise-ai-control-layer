// Package intent là chặng S4: phân loại ý định bằng LUẬT của pack.
//
// Luật trước (P13): một ý định khớp khi MỌI nhóm điều kiện của nó được thoả. Hai ý định
// cùng độ ưu tiên cùng khớp thì không chọn bừa — trả về mơ hồ để hỏi lại. Classifier
// bằng model chỉ được thêm khi cổng §9 mở (luật trượt trên bộ ca router đo được).
package intent

import (
	"fmt"
	"slices"
	"sort"
	"strings"
	"time"

	"github.com/luannt2002/enterprise-ai-control-layer/core/contracts"
	"github.com/luannt2002/enterprise-ai-control-layer/data/normalization"
)

// Def là một ý định trong pack (intents.yaml). Chỉ là dữ liệu: danh sách từ khoá,
// thuật ngữ, khe — không có biểu thức (P12).
type Def struct {
	Name  string `yaml:"name"`
	Label string `yaml:"label"` // cách gọi ý định khi phải hỏi lại người dùng
	Route string `yaml:"route"` // một trong 12 route
	// Keywords: ít nhất một cụm phải xuất hiện (so khớp theo luật dấu của normalization).
	Keywords []string `yaml:"keywords"`
	// Terms: nếu khai báo thì ít nhất một thuật ngữ (mã term trong lexicon) phải có mặt.
	Terms []string `yaml:"terms"`
	// MetricByTerm: thuật ngữ → chỉ số ngữ nghĩa trả lời câu hỏi này ("dầu" + "nhận" →
	// issued_fuel; "dầu" + "báo" → reported_fuel). Chọn chỉ số là quyết định nghiệp vụ.
	MetricByTerm map[string]string `yaml:"metric_by_term"`
	Derived      string            `yaml:"derived"` // chỉ số dẫn xuất (route CALCULATION)
	Slots        []SlotDef         `yaml:"slots"`
	Priority     int               `yaml:"priority"`
	Template     string            `yaml:"template"`   // mẫu câu khi route không cần model
	Capability   string            `yaml:"capability"` // quyền capability cần có (governance)
	// Weak: chỉ có từ khoá chung chung ("bao nhiêu", "tổng") — một mình không đủ để coi câu là
	// thuộc phạm vi, nên không gây hỏi lại khi thiếu thuật ngữ ("VN-Index bao nhiêu?").
	Weak bool `yaml:"weak"`
}

// Options là chính sách phân loại do pack khai.
type Options struct {
	// IgnorePhrases: cụm KHÔNG phải từ khoá dù chứa từ khoá ("người dùng" ≠ "dùng", "đúng không"
	// gõ không dấu ≠ "dùng"). Chúng chiếm chỗ trong luật khớp dài nhất.
	IgnorePhrases []string
	// DefaultTerms: thuật ngữ hiểu ngầm khi câu nhắc thực thể mà không nói vật tư, và không có
	// đại lượng nào khác (trợ lý nhiên liệu: "máy 03 xuất bao nhiêu?" = dầu).
	DefaultTerms []string
	// ComparisonMarkers: cụm so sánh ("nhiều hơn", "so với"). Câu nhắc HAI nghĩa số liệu ngang
	// nhau kèm cụm so sánh là phép tính hiệu theo thứ tự nhắc, không phải câu mơ hồ.
	ComparisonMarkers []string
	// FocusMarkers: cụm đánh dấu mệnh đề HỎI trong câu nhiều mệnh đề (cụm xác nhận "đúng không"):
	// từ khoá ý định lấy trong mệnh đề đó, các mệnh đề trước là ngữ cảnh (xem FocusClause).
	FocusMarkers []string
	// QuantityMarkers: cụm cho biết câu hỏi đòi một LƯỢNG ("bao nhiêu", "định mức"): cùng FocusMarkers là dấu hiệu của một câu hỏi SỐ LIỆU, để câu hỏi
	// quy trình thuần có từ khoá số liệu yếu ("Thủ tục xin cấp dầu cho dự án A và dự án B?") không bị hỏi lại "máy hoặc dự án" (xem Set.asksData).
	QuantityMarkers []string
	// Measurable báo một thuật ngữ là đại lượng đo được (có đơn vị mặc định) — có đại lượng khác
	// trong câu thì KHÔNG hiểu ngầm thuật ngữ mặc định ("giá", "nhân công").
	Measurable func(term string) bool
}

// SetOptions gắn chính sách phân loại của pack.
func (s *Set) SetOptions(o Options) { s.opt = o }

// SlotDef là một khe của ý định.
type SlotDef struct {
	Name     string   `yaml:"name"`
	Label    string   `yaml:"label"` // cách gọi khe khi hỏi lại: "máy hoặc dự án"
	From     string   `yaml:"from"`  // entity | period
	Types    []string `yaml:"types"` // loại thực thể (from=entity)
	Required bool     `yaml:"required"`
	// Default cho khe period: today | this_month. Rỗng = không có mặc định.
	Default string `yaml:"default"`
}

// Set là tập ý định đã kiểm.
type Set struct {
	defs []Def
	opt  Options
}

// Compile kiểm tập ý định. termKnown/metricKnown báo mã có trong lexicon/catalog.
func Compile(defs []Def, termKnown, metricKnown func(string) bool) (*Set, error) {
	var errs []string
	fail := func(f string, a ...any) { errs = append(errs, fmt.Sprintf(f, a...)) }
	seen := map[string]bool{}
	for _, d := range defs {
		if d.Name == "" {
			fail("ý định thiếu name")
			continue
		}
		if seen[d.Name] {
			fail("ý định %s khai báo hai lần", d.Name)
		}
		seen[d.Name] = true
		if !contracts.RouteKind(d.Route).Valid() {
			fail("ý định %s: route %q không thuộc 12 route", d.Name, d.Route)
		}
		if d.Capability == "" && !responseOnly(contracts.RouteKind(d.Route)) {
			// Quên khai capability không được biến thành "ai cũng được": chặn lúc khởi động.
			fail("ý định %s: route %s chạm dữ liệu/model nhưng không khai capability", d.Name, d.Route)
		}
		if len(d.Keywords) == 0 {
			fail("ý định %s không có keywords nào — luật không thể khớp", d.Name)
		}
		for _, t := range d.Terms {
			if !termKnown(t) {
				fail("ý định %s: term %q chưa khai báo trong lexicon", d.Name, t)
			}
		}
		for t, m := range d.MetricByTerm {
			if !termKnown(t) {
				fail("ý định %s: metric_by_term dùng term %q chưa khai báo", d.Name, t)
			}
			if !metricKnown(m) {
				fail("ý định %s: chỉ số %q chưa khai báo trong metrics", d.Name, m)
			}
		}
		if d.Derived != "" && !metricKnown(d.Derived) {
			fail("ý định %s: chỉ số dẫn xuất %q chưa khai báo", d.Name, d.Derived)
		}
		for _, s := range d.Slots {
			if s.From != "entity" && s.From != "period" {
				fail("ý định %s: khe %s có from %q (chỉ nhận entity|period)", d.Name, s.Name, s.From)
			}
			if s.Default != "" && s.Default != "today" && s.Default != "this_month" {
				fail("ý định %s: khe %s default %q (chỉ nhận today|this_month)", d.Name, s.Name, s.Default)
			}
		}
	}
	if len(errs) > 0 {
		return nil, fmt.Errorf("intents không hợp lệ:\n  - %s", strings.Join(errs, "\n  - "))
	}
	return &Set{defs: defs}, nil
}

// topPriority trả tên các ý định ở độ ưu tiên cao nhất (giữ thứ tự khai báo).
func topPriority(defs []Def) []string {
	best := 0
	for i, d := range defs {
		if i == 0 || d.Priority > best {
			best = d.Priority
		}
	}
	var out []string
	for _, d := range defs {
		if d.Priority == best {
			out = append(out, d.Name)
		}
	}
	return out
}

// responseOnly là route chỉ trả lời bằng mẫu câu, không đọc dữ liệu hay gọi model — chỉ
// các route này được phép không khai capability.
func responseOnly(r contracts.RouteKind) bool {
	return r == contracts.RouteClarification || r == contracts.RouteReject
}

// MetricKeywords trả từ khoá của các ý định đọc thẳng một chỉ số (route SQL_QUERY, không "yếu"): người hỏi có thể nêu số của chỉ
// số khác làm ngữ cảnh cho câu xác nhận ("Đội báo 2k, vậy … đúng không?"). Không gồm ý định quy trình, chào hỏi, phép tính dẫn xuất.
func (s *Set) MetricKeywords() []string {
	var out []string
	for _, d := range s.defs {
		if contracts.RouteKind(d.Route) == contracts.RouteSQLQuery && !d.Weak {
			out = append(out, d.Keywords...)
		}
	}
	return out
}

// Def trả định nghĩa của một ý định.
func (s *Set) Def(name string) (Def, bool) {
	for _, d := range s.defs {
		if d.Name == name {
			return d, true
		}
	}
	return Def{}, false
}

// Result là kết quả phân loại, kèm thứ router cần để hỏi lại.
type Result struct {
	Intent contracts.Intent
	// Ambiguous: các ý định cùng khớp ở cùng độ ưu tiên — router hỏi lại.
	Ambiguous []string
	// Candidates: ứng viên thực thể cho khe còn thiếu do bí danh mơ hồ.
	Candidates []contracts.EntityRef
	// NeedTerm: ý định khớp từ khoá nhưng câu thiếu thuật ngữ bắt buộc ("nhận bao nhiêu?" mà
	// không nói vật tư gì) — router hỏi lại, không coi là ngoài phạm vi.
	NeedTerm []string
	// Knowledge: ý định tri thức (RAG) khớp cùng ý định dữ liệu — router ghép thành HYBRID.
	Knowledge string
	// Compare: ý định thứ hai của câu so sánh — Intent là vế trái, Compare là vế phải (khe
	// compare_metric mang chỉ số của nó); router chuyển thành phép tính.
	Compare string
	// Unclear: khe mà câu nêu nhiều giá trị khác nhau hoặc một ngày hỏng — khe để trống, router hỏi lại.
	Unclear Unclear
}

// Unclear ghi phần câu hỏi nêu NHIỀU giá trị khác nhau, hoặc một giá trị hỏng, cho một khe chỉ nhận MỘT: khe đó không
// được điền (nằm ở Intent.Missing) để router hỏi lại, thay vì lặng lẽ lấy vế đầu (P5). Khác Candidates (bí danh mơ hồ,
// router được gỡ bằng quyền xem hay dự án): đây là thứ người hỏi VIẾT RÕ, nên không ai được chọn thay họ.
type Unclear struct {
	// Entities: các thực thể KHÁC NHAU, cùng loại, câu nêu cho khe thực thể ("EX-03 và RL-02").
	Entities []contracts.EntityRef
	// Periods: các mốc / khoảng thời gian KHÁC NHAU câu nêu cho khe thời gian ("hôm nay và hôm qua", hai đầu mút
	// của "từ 10/10 đến 15/10"); chỉ có khi từ hai mốc trở lên.
	Periods []contracts.Span
	// Invalid: nguyên văn các ngày không đọc được thành ngày trên lịch ("31/02"), từ NormalizedInput.InvalidDates.
	Invalid []string
}

// Any báo có khe nào đang không rõ.
func (u Unclear) Any() bool { return len(u.Entities)+len(u.Periods)+len(u.Invalid) > 0 }

// Classify chạy luật trên đầu vào đã chuẩn hoá và đã nhận dạng thực thể.
// Không khớp ý định nào → Intent.Name rỗng.
func (s *Set) Classify(in contracts.NormalizedInput, res contracts.Resolution, now time.Time) Result {
	counts, firsts := keywordHits(s.defs, in.Raw, s.opt.IgnorePhrases)
	if fs, fe, ok := FocusClause(in.Raw, s.opt.FocusMarkers); ok {
		// "đội báo 4k, báo cáo ghi 4 m3, vậy máy 03 được cấp 4k dầu đúng không?" hỏi về CẤP, không
		// về BÁO: mệnh đề chứa cụm xác nhận quyết định ý định khi nó có từ khoá.
		if focus, f := keywordHits(s.defs, in.Raw[fs:fe], s.opt.IgnorePhrases); slices.ContainsFunc(focus, func(n int) bool { return n > 0 }) {
			counts, firsts = focus, f
		}
	}
	first := map[string]int{}
	implicit, other := s.implicitTerm(in, res)
	var matched, partial []Def
	score := map[string]int{}
	for i, d := range s.defs {
		if counts[i] == 0 {
			continue
		}
		switch {
		case hasTerm(d, in) || (implicit != "" && requires(d, implicit)):
			matched = append(matched, d)
			score[d.Name], first[d.Name] = counts[i], firsts[i]
		case !d.Weak && !other: // câu nói về đại lượng khác thì không hỏi lại về dầu
			partial = append(partial, d)
		}
	}
	if len(matched) == 0 {
		return Result{NeedTerm: topPriority(partial)}
	}
	// Ghép route: ý định DỮ LIỆU (SQL/CALC) và ý định TRI THỨC (RAG) cùng khớp —
	//   chỉ một nhóm → nhóm đó; cả hai mà ý định dữ liệu điền được thực thể → HYBRID (số liệu +
	//   quy trình); cả hai mà thiếu thực thể → câu hỏi quy trình (RAG), trừ khi câu có MỐC THỜI
	//   GIAN cụ thể: đó là câu hỏi số liệu ("hôm nay cấp bao nhiêu…") → hỏi lại máy/dự án.
	var data, knowledge []Def
	for _, d := range matched {
		if contracts.RouteKind(d.Route) == contracts.RouteRAG {
			knowledge = append(knowledge, d)
		} else {
			data = append(data, d)
		}
	}
	if len(data) == 0 {
		return s.build(knowledge, score, first, in, res, now, implicit)
	}
	r := s.build(data, score, first, in, res, now, implicit)
	if len(knowledge) == 0 || len(r.Ambiguous) > 0 {
		return r
	}
	if def, _ := s.Def(r.Intent.Name); missingEntity(def, r.Intent) {
		// Người hỏi ĐÃ nêu thứ của một câu hỏi số liệu: một mốc thời gian (kể cả nhiều kỳ hay ngày hỏng) hoặc nhiều thực thể cùng
		// loại tranh nhau cho khe chủ thể (khe thiếu vì không chọn được một, không phải vì vắng) — hỏi lại, không chuyển sang câu hỏi
		// quy trình rồi bỏ thầm phần số liệu và các thực thể đã nêu (P5). Nhiều thực thể mà câu không có dấu hiệu hỏi số liệu nào (asksData) thì
		// chúng chỉ là chủ đề của một câu hỏi quy trình thuần ("Khi lệch phiếu kho của EX-03 và RL-02 thì phải làm gì?"): hỏi lại "máy hoặc dự án"
		// ở đó là hỏi OAN.
		if _, dated := firstPeriod(in); dated || len(in.InvalidDates) > 0 || (len(r.Unclear.Entities) > 0 && s.asksData(in.Raw)) {
			return r
		}
		return s.build(knowledge, score, first, in, res, now, implicit)
	}
	k := s.build(knowledge, score, first, in, res, now, implicit)
	if len(k.Ambiguous) == 0 {
		r.Knowledge = k.Intent.Name
	}
	return r
}

// asksData báo câu có dấu hiệu hỏi SỐ LIỆU chứ không chỉ nhắc một từ khoá số liệu trong câu hỏi quy trình: cụm hỏi lượng ("bao nhiêu"), cụm xác nhận
// ("đúng không"), hoặc một MỆNH ĐỀ riêng chỉ có từ khoá số liệu mà không có từ khoá quy trình ("…được cấp dầu thế nào, quy trình xử lý khi lệch là gì?":
// vế số liệu của câu ghép). Pack không khai dấu hiệu nào (SetOptions chưa gọi hay để trống) thì không có căn cứ coi câu là thuần quy trình: trả true (hỏi lại).
// Đánh đổi: câu quy trình tách mệnh đề bằng dấu phẩy mà một mệnh đề chỉ có từ khoá số liệu ("Việc cấp dầu cho EX-03 và RL-02 phải làm thế nào, khi lệch phiếu
// kho thì xử lý ra sao?") vẫn bị hỏi lại.
func (s *Set) asksData(text string) bool {
	markers := append(slices.Clone(s.opt.QuantityMarkers), s.opt.FocusMarkers...)
	if len(markers) == 0 {
		return true
	}
	if slices.ContainsFunc(markers, func(m string) bool { return normalization.ContainsPhrase(text, m) }) {
		return true
	}
	for _, clause := range clausesOf(text) {
		counts, _ := keywordHits(s.defs, clause, s.opt.IgnorePhrases)
		data, knowledge := 0, 0
		for i, d := range s.defs {
			switch contracts.RouteKind(d.Route) {
			case contracts.RouteRAG:
				knowledge += counts[i]
			case contracts.RouteClarification, contracts.RouteReject: // chào hỏi: không phải từ khoá số liệu
			default:
				data += counts[i]
			}
		}
		if data > 0 && knowledge == 0 {
			return true
		}
	}
	return false
}

// clausesOf tách câu thành các mệnh đề (ranh giới như FocusClause: , ; . ? ! theo sau là khoảng trắng hoặc hết câu).
func clausesOf(text string) []string {
	var out []string
	start := 0
	for i := 0; i < len(text); i++ {
		if strings.IndexByte(",;.?!", text[i]) >= 0 && (i+1 >= len(text) || text[i+1] == ' ' || text[i+1] == '\n') {
			out = append(out, text[start:i])
			start = i + 1
		}
	}
	return append(out, text[start:])
}

// missingEntity báo ý định còn thiếu khe thực thể bắt buộc.
func missingEntity(d Def, it contracts.Intent) bool {
	for _, sd := range d.Slots {
		if sd.From != "entity" || !sd.Required {
			continue
		}
		for _, m := range it.Missing {
			if m == sd.Name {
				return true
			}
		}
	}
	return false
}

// build chọn ý định thắng trong một nhóm và điền khe của nó.
func (s *Set) build(matched []Def, score, first map[string]int, in contracts.NormalizedInput, res contracts.Resolution, now time.Time, implicit string) Result {
	// Ưu tiên cao hơn thắng; cùng ưu tiên thì ý định có NHIỀU cụm từ khoá khớp hơn thắng
	// ("cảm biến … đo được" hơn "báo"); vẫn hoà thì hỏi lại.
	matched = append([]Def(nil), matched...)
	sort.SliceStable(matched, func(i, j int) bool {
		if matched[i].Priority != matched[j].Priority {
			return matched[i].Priority > matched[j].Priority
		}
		return score[matched[i].Name] > score[matched[j].Name]
	})
	top := matched[0]
	var ambiguous []string
	for _, d := range matched[1:] {
		if d.Priority == top.Priority && score[d.Name] == score[top.Name] {
			ambiguous = append(ambiguous, d.Name)
		}
	}
	if len(ambiguous) == 1 && s.comparing(in.Raw) {
		// Hai nghĩa số liệu ngang nhau + cụm so sánh: "đội báo nhiều hơn kho xuất bao nhiêu" là
		// hiệu (vế nhắc trước − vế nhắc sau), không phải câu mơ hồ.
		other, _ := s.Def(ambiguous[0])
		left, right := top, other
		if first[right.Name] < first[left.Name] {
			left, right = right, left
		}
		if left.Derived == "" && right.Derived == "" {
			r := s.fill(left, in, res, now, implicit)
			lm, ok := r.Intent.Slots["metric"]
			if rm, _ := metricOf(right, in, implicit); ok && rm != "" && rm != lm.Text {
				r.Intent.Slots["compare_metric"] = contracts.Slot{Name: "compare_metric", Text: rm}
				r.Intent.Rule += "+compare:" + right.Name
				r.Compare = right.Name
				return r
			}
		}
	}
	if len(ambiguous) > 0 {
		return Result{Ambiguous: append([]string{top.Name}, ambiguous...)}
	}
	return s.fill(top, in, res, now, implicit)
}

// comparing báo câu có cụm so sánh của pack.
func (s *Set) comparing(text string) bool {
	return slices.ContainsFunc(s.opt.ComparisonMarkers, func(m string) bool { return normalization.ContainsPhrase(text, m) })
}

// metricOf trả chỉ số mà ý định trả lời cho câu này: do thuật ngữ trong câu, không có thì do thuật
// ngữ hiểu ngầm (rule cho biết đã dùng thuật ngữ hiểu ngầm).
func metricOf(d Def, in contracts.NormalizedInput, implicit string) (metric, rule string) {
	for _, sp := range in.SpansOf(contracts.ValueTerm) {
		if m, ok := d.MetricByTerm[sp.Value.Text]; ok {
			return m, ""
		}
	}
	if implicit != "" {
		if m, ok := d.MetricByTerm[implicit]; ok {
			return m, "+default_term:" + implicit
		}
	}
	return "", ""
}

// fill điền khe của một ý định đã chọn.
func (s *Set) fill(top Def, in contracts.NormalizedInput, res contracts.Resolution, now time.Time, implicit string) Result {
	it := contracts.Intent{Name: top.Name, Confidence: 1, Slots: map[string]contracts.Slot{}, Rule: "intent:rules"}
	var candidates []contracts.EntityRef
	var unclear Unclear
	for _, sd := range top.Slots {
		switch sd.From {
		case "entity":
			// Ứng viên mơ hồ được gom CẢ khi khe đã điền: "máy 07 của dự án A" điền khe bằng
			// dự án A nhưng câu hỏi là về một máy — router dùng dự án để gỡ mơ hồ.
			for _, a := range res.Ambiguous {
				for _, c := range a.Candidates {
					if hasType(sd.Types, c.Type) {
						candidates = append(candidates, c)
					}
				}
			}
			if clash := clashing(res, sd.Types); len(clash) > 0 {
				// Câu nêu ≥ 2 thực thể KHÁC NHAU cùng loại cho khe chỉ nhận một: không lấy thực thể nhắc đầu (P5).
				it.Missing = append(it.Missing, sd.Name)
				unclear.Entities = append(unclear.Entities, clash...)
				continue
			}
			if e, ok := firstOfTypes(res.Entities, sd.Types); ok {
				ref := e
				it.Slots[sd.Name] = contracts.Slot{Name: sd.Name, Entity: &ref}
				continue
			}
			if sd.Required {
				it.Missing = append(it.Missing, sd.Name)
			}
		case "period":
			ps := distinctPeriods(in)
			if len(ps) > 1 || len(in.InvalidDates) > 0 {
				// Nhiều kỳ khác nhau, hoặc một ngày hỏng: không lấy kỳ nhắc đầu, cũng không rơi về ngày mặc định (P5).
				it.Missing = append(it.Missing, sd.Name)
				if len(ps) > 1 {
					unclear.Periods = ps
				}
				unclear.Invalid = in.InvalidDates
				continue
			}
			if len(ps) == 1 { // kỳ duy nhất (kỳ hẹp, nếu câu viết lồng nhiều kỳ)
				v := ps[0].Value
				it.Slots[sd.Name] = contracts.Slot{Name: sd.Name, Value: &v, Text: ps[0].Raw}
				continue
			}
			if v, ok := defaultPeriod(sd.Default, now); ok {
				it.Slots[sd.Name] = contracts.Slot{Name: sd.Name, Value: &v, Text: sd.Default}
				continue
			}
			if sd.Required {
				it.Missing = append(it.Missing, sd.Name)
			}
		}
	}
	// Chỉ số do thuật ngữ trong câu quyết định; không có thì do thuật ngữ hiểu ngầm.
	if m, rule := metricOf(top, in, implicit); m != "" {
		it.Slots["metric"] = contracts.Slot{Name: "metric", Text: m}
		it.Rule += rule
	}
	if top.Derived != "" {
		it.Slots["metric"] = contracts.Slot{Name: "metric", Text: top.Derived}
	}
	if len(top.MetricByTerm) > 0 || top.Derived != "" {
		if _, ok := it.Slots["metric"]; !ok {
			it.Missing = append(it.Missing, "metric")
		}
	}
	if len(unclear.Entities) > 0 {
		candidates = nil // thực thể người hỏi viết rõ đang tranh nhau: không để bí danh mơ hồ làm router chọn thay
	}
	return Result{Intent: it, Candidates: candidates, Unclear: unclear}
}

// FocusClause trả khoảng byte [đầu, cuối) của MỆNH ĐỀ chứa cụm đánh dấu đầu tiên tìm thấy. Mệnh đề
// là đoạn giữa hai dấu , ; . ? ! (dấu theo sau là khoảng trắng hoặc hết câu — "3.850", "0,5" không
// cắt). ok = false khi không có cụm nào, hoặc câu chỉ có một mệnh đề.
func FocusClause(text string, markers []string) (start, end int, ok bool) {
	at := -1
	for _, m := range markers {
		if sp := normalization.PhraseSpans(text, m); len(sp) > 0 && (at < 0 || sp[0][0] < at) {
			at = sp[0][0]
		}
	}
	if at < 0 {
		return 0, 0, false
	}
	start, end = 0, len(text)
	for i := 0; i < len(text); i++ {
		if !strings.ContainsRune(",;.?!", rune(text[i])) || (i+1 < len(text) && text[i+1] != ' ' && text[i+1] != '\n') {
			continue
		}
		if i < at {
			start = i + 1
		} else if end == len(text) {
			end = i
		}
	}
	// chỉ một mệnh đề (phần sau chỉ còn dấu câu) thì không có gì để thu hẹp
	return start, end, start > 0 || strings.TrimSpace(strings.TrimLeft(text[end:], ",;.?!")) != ""
}

// keywordHits báo ý định nào có từ khoá khớp, theo luật KHỚP DÀI NHẤT trên từ khoá của MỌI ý
// định: lần khớp nằm trọn trong lần khớp dài hơn của từ khoá khác thì không tính. Câu gõ
// không dấu "bao nhieu" khớp cả "bao nhiêu" lẫn "báo" — "báo" nằm trong "bao nhiêu" nên bị
// bỏ; xét từng ý định riêng lẻ thì câu hỏi chung chung bị hiểu thành "dầu đội báo".
func keywordHits(defs []Def, text string, ignore []string) (counts, first []int) {
	type hit struct{ def, start, end int }
	var all []hit
	for i, d := range defs {
		for _, k := range d.Keywords {
			for _, sp := range normalization.PhraseSpans(text, k) {
				all = append(all, hit{i, sp[0], sp[1]})
			}
		}
	}
	for _, p := range ignore { // cụm bỏ qua: chiếm chỗ, không thuộc ý định nào (def = -1)
		for _, sp := range normalization.PhraseSpans(text, p) {
			all = append(all, hit{-1, sp[0], sp[1]})
		}
	}
	counts, first = make([]int, len(defs)), make([]int, len(defs))
	for i := range first {
		first[i] = len(text)
	}
	for _, h := range all {
		if h.def < 0 {
			continue
		}
		covered := false
		for _, o := range all {
			if o.start <= h.start && h.end <= o.end && o.end-o.start > h.end-h.start {
				covered = true
				break
			}
		}
		if !covered {
			counts[h.def]++
			first[h.def] = min(first[h.def], h.start)
		}
	}
	return counts, first
}

// implicitTerm trả thuật ngữ mặc định được hiểu ngầm ("" khi không áp) và báo câu có nói về một
// đại lượng đo được KHÁC không. Hiểu ngầm chỉ khi: câu nhắc ít nhất một thực thể, chưa có thuật
// ngữ mặc định, và không có đại lượng khác ("giá", "nhân công" → không phải câu về dầu).
func (s *Set) implicitTerm(in contracts.NormalizedInput, res contracts.Resolution) (implicit string, other bool) {
	explicit := false
	for _, sp := range in.SpansOf(contracts.ValueTerm) {
		t := sp.Value.Text
		isDefault := false
		for _, d := range s.opt.DefaultTerms {
			isDefault = isDefault || t == d
		}
		explicit = explicit || isDefault
		if !isDefault && s.opt.Measurable != nil && s.opt.Measurable(t) {
			other = true
		}
	}
	if len(s.opt.DefaultTerms) == 0 || explicit || other || len(res.Entities)+len(res.Ambiguous) == 0 {
		return "", other
	}
	return s.opt.DefaultTerms[0], false
}

func requires(d Def, term string) bool {
	for _, t := range d.Terms {
		if t == term {
			return true
		}
	}
	return false
}

// hasTerm báo câu có ít nhất một thuật ngữ ý định đòi (không đòi thì luôn đúng).
func hasTerm(d Def, in contracts.NormalizedInput) bool {
	if len(d.Terms) == 0 {
		return true
	}
	for _, sp := range in.SpansOf(contracts.ValueTerm) {
		for _, t := range d.Terms {
			if sp.Value.Text == t {
				return true
			}
		}
	}
	return false
}

func firstOfTypes(es []contracts.EntityRef, types []string) (contracts.EntityRef, bool) {
	for _, e := range es {
		if hasType(types, e.Type) {
			return e, true
		}
	}
	return contracts.EntityRef{}, false
}

// clashing trả các thực thể mà câu nêu ≥ 2 LẦN NHẮC KHÁC NHAU cùng loại trong các loại của khe, theo thứ tự nhắc tới
// (thực thể đã rõ trước, rồi ứng viên của các bí danh mơ hồ); rỗng nếu không loại nào có hai lần nhắc. Một lần nhắc là
// một thực thể đã rõ (tính theo ID), hoặc một bí danh mơ hồ cùng loại mà không ứng viên nào là thực thể đã rõ ("EX-03
// (máy 03)" chỉ là một máy). Máy kèm dự án chứa nó ("EX-03 của dự án A") khác loại nên không tranh nhau — dự án là
// phần định ngữ.
func clashing(res contracts.Resolution, types []string) []contracts.EntityRef {
	var uniq []contracts.EntityRef
	count, seen := map[string]int{}, map[string]bool{}
	for _, e := range res.Entities {
		if hasType(types, e.Type) && !seen[e.ID] {
			seen[e.ID] = true
			uniq = append(uniq, e)
			count[e.Type]++
		}
	}
	for _, m := range res.Ambiguous {
		if m.Type == "" || !hasType(types, m.Type) || slices.ContainsFunc(m.Candidates, func(c contracts.EntityRef) bool { return seen[c.ID] }) {
			continue
		}
		count[m.Type]++
		for _, c := range m.Candidates {
			if !seen[c.ID] {
				seen[c.ID] = true
				uniq = append(uniq, c)
			}
		}
	}
	var out []contracts.EntityRef
	for _, e := range uniq {
		if count[e.Type] > 1 {
			out = append(out, e)
		}
	}
	return out
}

// distinctPeriods trả các span mốc / khoảng thời gian KHÁC NHAU (theo giá trị chuẩn), mỗi giá trị một lần, theo thứ
// tự nhắc tới: "hôm nay" và "19/10" cùng một ngày thì là một kỳ. Kỳ RỘNG ôm trọn một kỳ hẹp liền kề cũng chỉ là bối cảnh của kỳ hẹp,
// không phải một kỳ nữa: "tháng 10 năm nay", "hôm qua trong tháng 10", "ngày 19/10 năm nay" là MỘT kỳ — kỳ hẹp. Liền kề = chỉ cách
// nhau khoảng trắng hoặc một từ chỉ bao hàm (nestingWords); "tháng 10 và hôm qua", "tuần trước, từ 12/10" thì vẫn là nhiều kỳ.
func distinctPeriods(in contracts.NormalizedInput) []contracts.Span {
	var out []contracts.Span
	for _, sp := range in.SpansOf(contracts.ValuePeriod, contracts.ValueDate) {
		if !slices.ContainsFunc(out, func(o contracts.Span) bool { return o.Value.Text == sp.Value.Text }) {
			out = append(out, sp)
		}
	}
	var kept []contracts.Span
	for _, a := range out {
		enclosing := slices.ContainsFunc(out, func(b contracts.Span) bool {
			return b.Value.Text != a.Value.Text && covers(a.Value.Text, b.Value.Text) && adjacent(in.Raw, a, b)
		})
		if !enclosing {
			kept = append(kept, a)
		}
	}
	return kept
}

// nestingWords là các từ nối cho phép giữa hai kỳ lồng nhau khi người hỏi viết chúng thành một cụm ("hôm qua trong tháng 10").
var nestingWords = []string{"trong", "thuộc", "của", "vào"}

// covers báo kỳ outer ôm trọn kỳ inner; cả hai ở dạng giá trị chuẩn "YYYY-MM-DD" hoặc "YYYY-MM-DD/YYYY-MM-DD" (so sánh chuỗi là so
// sánh ngày).
func covers(outer, inner string) bool {
	of, ot, _ := strings.Cut(outer, "/")
	if ot == "" {
		ot = of
	}
	inf, it, _ := strings.Cut(inner, "/")
	if it == "" {
		it = inf
	}
	return of <= inf && it <= ot
}

// adjacent báo hai span đứng liền nhau trong văn bản: giữa chúng chỉ có khoảng trắng hoặc một từ chỉ bao hàm.
func adjacent(raw string, a, b contracts.Span) bool {
	if b.Start < a.Start {
		a, b = b, a
	}
	if a.End > b.Start || b.Start > len(raw) {
		return false
	}
	gap := strings.TrimSpace(raw[a.End:b.Start])
	return gap == "" || slices.ContainsFunc(nestingWords, func(w string) bool { return normalization.Fold(gap) == normalization.Fold(w) })
}

func hasType(types []string, t string) bool {
	if len(types) == 0 {
		return true
	}
	for _, x := range types {
		if x == t {
			return true
		}
	}
	return false
}

func firstPeriod(in contracts.NormalizedInput) (contracts.Span, bool) {
	sp := in.SpansOf(contracts.ValuePeriod, contracts.ValueDate)
	if len(sp) == 0 {
		return contracts.Span{}, false
	}
	return sp[0], true
}

func defaultPeriod(def string, now time.Time) (contracts.Value, bool) {
	d := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	switch def {
	case "today":
		return contracts.Value{Kind: contracts.ValueDate, Text: d.Format("2006-01-02")}, true
	case "this_month":
		from := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
		to := from.AddDate(0, 1, -1)
		return contracts.Value{Kind: contracts.ValuePeriod, Text: from.Format("2006-01-02") + "/" + to.Format("2006-01-02")}, true
	}
	return contracts.Value{}, false
}
