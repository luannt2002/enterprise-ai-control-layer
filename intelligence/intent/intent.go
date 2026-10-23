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
	"unicode"
	"unicode/utf8"

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
	// Assumption: câu giả định nối vào MỌI câu trả lời theo ý định này (ý định chung: "Hiểu là dầu kho xuất; …", D-OP-15 điều kiện (i)).
	Assumption string `yaml:"assumption"`
}

// Unsupported là một NGHĨA hệ CHƯA hỗ trợ (tiền, dự báo, báo giá, cảnh báo, lệnh ghi, đánh giá — pack khai). Câu mang nghĩa này không được trả lời theo chỉ số
// còn lại của câu (bỏ qua cụm "dự báo" thì "báo" khớp ý định đội báo và câu bị trả số đội báo, sai nghĩa): router hỏi lại và nói rõ nghĩa nào chưa hỗ trợ
// (ADR-0010 nợ S4 sau H4, D-OP-18 = c).
type Unsupported struct {
	Name     string   `yaml:"name"`
	Template string   `yaml:"template"` // mẫu câu nói nghĩa đó chưa được hỗ trợ (templates.yaml)
	Cues     []string `yaml:"cues"`     // cụm từ báo nghĩa đó; chiếm chỗ trong luật khớp dài nhất nên chữ con của chúng ("báo" của "dự báo") không thành từ khoá
	// NeedsValue: chỉ tính khi câu nêu một CON SỐ và không phải câu hỏi (lệnh ghi "Ghi nhận EX-03 tiêu thụ 3782 L"). Không có giá trị, hay là câu hỏi, thì cụm vẫn
	// chiếm chỗ nhưng câu là câu hỏi số liệu như thường (D-OP-18 = c).
	NeedsValue bool `yaml:"needs_value"`
}

// Options là chính sách phân loại do pack khai.
type Options struct {
	// Unsupported: các nghĩa hệ chưa hỗ trợ (xem Unsupported).
	Unsupported []Unsupported
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
	// NeutralWords: từ không mang nghĩa số liệu (lịch sự, trợ từ — claim_neutral_words của pack). Câu không khớp từ khoá nào chỉ được
	// trả lời theo ý định chung khi ngoài tên, mốc, lượng, thuật ngữ, cụm xác nhận ra chỉ còn những từ này và mảnh từ khoá của ý
	// định riêng mà ý định chung quy về (xem Set.generic).
	NeutralWords []string
	// QuantityFits báo một lượng đổi được sang đơn vị của chỉ số (cùng chiều đo): a2 chỉ coi lượng như thế là chữ của câu hỏi chỉ số
	// chung ("4 khối" với dầu kho xuất); "7 tấn", "8h" là đại lượng khác. Không khai ⇒ không lượng có đơn vị nào được coi là thuộc nghĩa.
	QuantityFits func(metric string, v contracts.Value) bool
	// DerivedOperands trả hai chỉ số gốc (vế trái, vế phải) của một chỉ số dẫn xuất ("issue_vs_measured" → kho xuất, tiêu thụ). Ý định dẫn xuất ("chênh
	// lệch") chỉ được trả cặp mặc định ấy khi câu không nêu chỉ số nào ngoài cặp; không khai ⇒ không kiểm (xem Set.comparison).
	DerivedOperands func(derived string) (left, right string, ok bool)
	// ValueInRange báo một số nằm trong miền một lần đo của chỉ số. a2 chỉ đòi điều này với số KHÔNG tự viết đơn vị (số trần, hay số chỉ có đơn vị nhờ
	// thuật ngữ đứng cạnh): "4 củ", "dầu 4,1 triệu" ngoài miền thì không phải giá trị của chỉ số; lượng viết rõ đơn vị ("80000 L" của cả tháng) không bị
	// chặn. Không khai ⇒ không chặn.
	ValueInRange func(metric string, v contracts.Value) bool
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

// WeakKeywords trả từ khoá của các ý định yếu ("bao nhiêu", "tổng"): chúng chỉ chung chung chính chỉ số đang hỏi, không nêu chỉ số nào khác.
func (s *Set) WeakKeywords() []string {
	var out []string
	for _, d := range s.defs {
		if d.Weak {
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
	// NeedMeaning: câu có thực thể + thuật ngữ vật tư mà không từ khoá ý định nào khớp, và không trả lời được theo ý định chung —
	// router hỏi lại nghĩa số liệu, liệt kê các ý định này; không coi là ngoài phạm vi (ADR-0010 bất biến (a)).
	NeedMeaning []string
	// Unsupported: câu mang nghĩa hệ chưa hỗ trợ — router hỏi lại và nói rõ nghĩa nào (NeedMeaning kể các ý định số liệu có thể hỏi thay).
	Unsupported []Unsupported
	// Mentioned: thực thể đã giải của câu hỏi lại nghĩa số liệu — router kiểm phạm vi dữ liệu của người hỏi TRƯỚC khi hỏi lại.
	Mentioned []contracts.EntityRef
	// Knowledge: ý định tri thức (RAG) khớp cùng ý định dữ liệu — router ghép thành HYBRID.
	Knowledge string
	// Compare: ý định thứ hai của câu so sánh — Intent là vế trái, Compare là vế phải (khe
	// compare_metric mang chỉ số của nó); router chuyển thành phép tính.
	Compare string
	// Unclear: khe mà câu nêu nhiều giá trị khác nhau hoặc một ngày hỏng — khe để trống, router hỏi lại.
	Unclear Unclear
	// Why: giải thích của chặng S4 do Classify ghi ngay tại chỗ quyết định (R2-03) — chỉ để đọc vết, không bước nào đọc lại nó để quyết.
	Why Why
}

// Why là phần giải thích của chặng S4: đầu vào quyết định (từ khoá khớp, thuật ngữ, khe), ý định đã xét và vì sao thắng / bị loại,
// và nguồn của từng ứng viên thực thể dùng để hỏi lại.
type Why struct {
	Reasons  []string
	Rejected []string
	// Origins: nơi sinh ra từng ứng viên thực thể trong Result.Candidates và Result.Unclear.Entities.
	Origins []Origin
}

// Origin cho biết một ứng viên thực thể (dùng để hỏi lại) đến từ đâu.
type Origin struct {
	ID   string // mã thực thể ứng viên
	From string // OriginS3: ứng viên của cụm mơ hồ do S3 thấy · OriginS4: S4 gộp thêm vì khe chỉ nhận một thực thể mà câu nêu nhiều
	Note string // cụm mơ hồ nào, hay thực thể rõ nào bị gộp và vì sao
}

// Nguồn của ứng viên thực thể (Origin.From).
const (
	OriginS3 = "S3"
	OriginS4 = "S4"
)

func (w *Why) reason(format string, a ...any) {
	w.Reasons = append(w.Reasons, fmt.Sprintf(format, a...))
}
func (w *Why) reject(format string, a ...any) {
	w.Rejected = append(w.Rejected, fmt.Sprintf(format, a...))
}

// addOrigin ghi nguồn của một ứng viên nếu chưa có (mỗi thực thể một nguồn — nguồn đầu tiên; một bí danh nhắc N lần không sinh N bản ghi).
func (w *Why) addOrigin(o Origin) {
	if !slices.ContainsFunc(w.Origins, func(x Origin) bool { return x.ID == o.ID }) {
		w.Origins = append(w.Origins, o)
	}
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
	w := &Why{}
	r := s.classify(in, res, now, w)
	// N1 (ADR-0010 bất biến (c)): câu hỏi lại vì hoà ý định hay thiếu thuật ngữ mà câu còn bí danh mơ hồ thì mang cả ứng viên của bí danh — người
	// hỏi trả lời cả hai điều một lượt; bỏ rơi ứng viên là chỉ hỏi nửa câu.
	if (len(r.Ambiguous) > 0 || len(r.NeedTerm) > 0) && len(r.Candidates) == 0 {
		r.Candidates = aliasCandidates(res, w)
		if len(r.Candidates) > 0 {
			w.reason("câu hỏi lại vì hoà ý định hay thiếu thuật ngữ mà còn bí danh mơ hồ — mang cả ứng viên của bí danh (%s) để người hỏi trả lời một lượt (N1)", contracts.Brief(refKeys(r.Candidates), 6, "thực thể"))
		}
	}
	r.Why = *w
	return r
}

// aliasCandidates trả ứng viên của mọi bí danh mơ hồ trong câu, mỗi thực thể một lần, theo thứ tự nhắc; ghi nguồn (S3) của từng ứng viên vào w.
func aliasCandidates(res contracts.Resolution, w *Why) []contracts.EntityRef {
	var out []contracts.EntityRef
	for _, m := range res.Ambiguous {
		for _, c := range m.Candidates {
			if !slices.ContainsFunc(out, func(o contracts.EntityRef) bool { return o.ID == c.ID }) {
				out = append(out, c)
				w.addOrigin(Origin{ID: c.ID, From: OriginS3, Note: fmt.Sprintf("ứng viên của cụm mơ hồ %s", contracts.Quote(m.Mention))})
			}
		}
	}
	return out
}

// classify là thân của Classify; w nhận giải thích tại đúng chỗ từng quyết định (chỉ ghi, không đọc lại).
func (s *Set) classify(in contracts.NormalizedInput, res contracts.Resolution, now time.Time, w *Why) Result {
	// ADR-0010 bất biến (b): từ khoá lấy ở mệnh đề HỎI; mệnh đề ngữ cảnh (nêu một con số, đứng trước câu hỏi) bị che khỏi phép đếm
	prem := s.premiseRanges(in, res)
	text := maskRanges(in.Raw, prem)
	for _, r := range prem {
		w.reason("mệnh đề ngữ cảnh %s nêu một con số và đứng trước mệnh đề hỏi — chữ của nó chỉ nguồn của con số, không quyết định ý định (bất biến (b))", contracts.Quote(in.Raw[r[0]:r[1]]))
	}
	ignore := append(slices.Clone(s.opt.IgnorePhrases), s.cuePhrases()...) // cụm báo nghĩa chưa hỗ trợ chiếm chỗ: "báo" của "dự báo" không phải từ khoá
	counts, firsts, phrases := keywordHits(s.defs, text, ignore)
	if un := s.unsupportedIn(in, res, text, prem, counts); len(un) > 0 {
		names := make([]string, len(un))
		for i, u := range un {
			names[i] = u.Name
		}
		w.reason("câu mang nghĩa hệ chưa hỗ trợ (%s) — hỏi lại và nói rõ nghĩa nào chưa hỗ trợ, kể các nghĩa số liệu có thể hỏi thay (D-OP-18 = c)", strings.Join(names, ", "))
		return Result{Unsupported: un, NeedMeaning: s.meaningOptions(), Mentioned: res.Entities, Candidates: aliasCandidates(res, w)}
	}
	if fs, fe, ok := FocusClause(in.Raw, s.opt.FocusMarkers); ok {
		// "đội báo 4k, báo cáo ghi 4 m3, vậy máy 03 được cấp 4k dầu đúng không?" hỏi về CẤP, không
		// về BÁO: mệnh đề chứa cụm xác nhận quyết định ý định khi nó có từ khoá.
		if focus, f, fp := keywordHits(s.defs, in.Raw[fs:fe], ignore); slices.ContainsFunc(focus, func(n int) bool { return n > 0 }) {
			counts, firsts, phrases = focus, f, fp
			w.reason("từ khoá lấy trong mệnh đề hỏi %s (mệnh đề chứa cụm xác nhận quyết định ý định, các mệnh đề trước là ngữ cảnh)", contracts.Quote(in.Raw[fs:fe]))
		}
	}
	if hits := hitsText(s.defs, counts, phrases); hits != "" {
		w.reason("từ khoá khớp (luật khớp dài nhất): %s", hits)
	} else {
		w.reason("không từ khoá ý định nào khớp (luật khớp dài nhất)")
	}
	first := map[string]int{}
	implicit, other := s.implicitTerm(in, res)
	if implicit != "" {
		w.reason("thuật ngữ hiểu ngầm %s (câu nhắc thực thể mà không nói vật tư và không có đại lượng đo được nào khác — chính sách của pack)", implicit)
	}
	var matched, partial []Def
	score := map[string]int{}
	for i, d := range s.defs {
		if counts[i] == 0 {
			continue
		}
		kw := quoteAll(phrases[i])
		switch {
		case hasTerm(d, in) || (implicit != "" && requires(d, implicit)):
			matched = append(matched, d)
			score[d.Name], first[d.Name] = counts[i], firsts[i]
			via := "không đòi thuật ngữ"
			switch {
			case len(d.Terms) > 0 && hasTerm(d, in):
				via = "câu có thuật ngữ " + strings.Join(d.Terms, "|")
			case len(d.Terms) > 0:
				via = "thuật ngữ hiểu ngầm " + implicit
			}
			w.reason("khớp đủ: %s (ưu tiên %d, từ khoá %s, %s)", d.Name, d.Priority, kw, via)
		case !d.Weak && !other: // câu nói về đại lượng khác thì không hỏi lại về dầu
			partial = append(partial, d)
			w.reject("%s: khớp một phần — từ khoá %s khớp nhưng câu thiếu thuật ngữ %s", d.Name, kw, strings.Join(d.Terms, "|"))
		case d.Weak:
			w.reject("%s: ý định yếu — chỉ có từ khoá chung chung (%s), thiếu thuật ngữ %s nên một mình không đủ", d.Name, kw, strings.Join(d.Terms, "|"))
		default:
			w.reject("%s: từ khoá %s khớp nhưng câu nói về đại lượng đo được khác nên không hỏi lại thuật ngữ %s", d.Name, kw, strings.Join(d.Terms, "|"))
		}
	}
	if len(matched) == 0 {
		if len(partial) > 0 || !s.scopeSignal(in, res, implicit) {
			if len(partial) > 0 {
				w.reason("không ý định nào khớp đủ — hỏi lại thuật ngữ cho %s (ưu tiên cao nhất trong các ý định khớp một phần)", strings.Join(topPriority(partial), ", "))
			} else {
				w.reason("không ý định nào khớp và câu không đủ tín hiệu thuộc phạm vi số liệu (cần cả thực thể lẫn thuật ngữ mà một ý định đòi, hoặc lượng cùng chiều đo kèm chữ của từ khoá chỉ số) — ngoài phạm vi")
			}
			return Result{NeedTerm: topPriority(partial)}
		}
		// ADR-0010 (a), D-OP-15 = a2: có thực thể + thuật ngữ mà không từ khoá nào khớp KHÔNG phải ngoài phạm vi — trả lời theo ý
		// định chung khi câu không nói gì thêm ngoài nghĩa của nó (generic), còn lại hỏi lại nghĩa số liệu.
		g, ok, why := s.generic(in, res, implicit)
		if !ok {
			cands := aliasCandidates(res, w)
			opts := s.meaningOptions()
			w.reason("có thực thể và thuật ngữ nhưng không từ khoá ý định nào khớp, và câu không trả lời được theo ý định chung (%s) — hỏi lại nghĩa số liệu: %s", why, strings.Join(opts, ", "))
			return Result{NeedMeaning: opts, Mentioned: res.Entities, Candidates: cands}
		}
		w.reason("không từ khoá nào khớp — trả lời theo ý định chung %s (D-OP-15 a2): %s", g.Name, why)
		matched = []Def{g}
		score[g.Name], first[g.Name] = 1, 0
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
		return s.build(knowledge, score, first, in, res, now, implicit, w)
	}
	r := s.build(data, score, first, in, res, now, implicit, w)
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
			w.reject("%s (tri thức): khớp cùng nhưng bị bỏ — câu đã nêu mốc thời gian, ngày hỏng hay nhiều thực thể của một câu hỏi số liệu, hỏi lại phần số liệu thay vì chuyển sang câu hỏi quy trình", defNames(knowledge))
			return r
		}
		w.reject("%s: ý định số liệu bị bỏ — thiếu khe thực thể và câu không có dấu hiệu hỏi số liệu, coi là câu hỏi quy trình", r.Intent.Name)
		return s.build(knowledge, score, first, in, res, now, implicit, w)
	}
	k := s.build(knowledge, score, first, in, res, now, implicit, &Why{}) // chỉ để biết có hoà không; lời giải thích của nhóm tri thức gói ở dòng dưới
	if len(k.Ambiguous) == 0 {
		r.Knowledge = k.Intent.Name
		w.reason("ý định tri thức %s khớp cùng ý định số liệu — router ghép HYBRID nếu đủ khe", k.Intent.Name)
	} else {
		w.reject("%s (tri thức): khớp cùng nhưng hoà nhau nên không ghép HYBRID", defNames(knowledge))
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
		counts, _, _ := keywordHits(s.defs, clause, s.opt.IgnorePhrases)
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
func (s *Set) build(matched []Def, score, first map[string]int, in contracts.NormalizedInput, res contracts.Resolution, now time.Time, implicit string, w *Why) Result {
	if r, ok := s.comparison(matched, first, in, res, now, implicit, w); ok {
		return r
	}
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
		switch {
		case d.Priority == top.Priority && score[d.Name] == score[top.Name]:
			ambiguous = append(ambiguous, d.Name)
		case d.Priority != top.Priority:
			w.reject("%s: khớp đủ nhưng bị loại — ưu tiên %d thấp hơn %s (%d)", d.Name, d.Priority, top.Name, top.Priority)
		default:
			w.reject("%s: khớp đủ nhưng bị loại — %d cụm từ khoá, ít hơn %s (%d) cùng ưu tiên %d", d.Name, score[d.Name], top.Name, score[top.Name], top.Priority)
		}
	}
	if len(ambiguous) > 0 {
		w.reason("hoà: %s cùng ưu tiên %d và cùng %d cụm từ khoá — không chọn bừa, hỏi lại", strings.Join(append([]string{top.Name}, ambiguous...), ", "), top.Priority, score[top.Name])
		return Result{Ambiguous: append([]string{top.Name}, ambiguous...)}
	}
	switch {
	case len(matched) == 1:
		w.reason("chọn %s: ý định duy nhất khớp đủ", top.Name)
	case matched[1].Priority != top.Priority:
		w.reason("chọn %s: ưu tiên cao nhất (%d)", top.Name, top.Priority)
	default:
		w.reason("chọn %s: cùng ưu tiên %d nhưng nhiều cụm từ khoá hơn (%d)", top.Name, top.Priority, score[top.Name])
	}
	return s.fill(top, in, res, now, implicit, w)
}

// comparison xử lý câu SO SÁNH hai chỉ số (ADR-0010, cụm C4). Câu có dấu hiệu so sánh — một ý định dẫn xuất khớp ("chênh", "lệch") hoặc cụm so sánh
// của pack ("nhiều hơn") — và nêu ĐÚNG hai chỉ số khác nhau là phép so sánh đúng cặp người hỏi nêu (vế nhắc trước − vế nhắc sau): số cụm từ khoá
// khớp và độ ưu tiên của ý định dẫn xuất không phải tín hiệu nghĩa, nên không được quyết định thay họ ("cảm biến … đo được" hai cụm không thắng "cấp"
// một cụm; "chênh" không ép cặp mặc định của nó lên cặp khác). Ý định dẫn xuất chỉ trả cặp mặc định khi các chỉ số câu nêu đều nằm trong cặp ấy; câu
// nêu chỉ số ngoài cặp mà không phải đúng hai chỉ số thì không có cặp nào để trả thay người hỏi: hỏi lại. ok = false: không phải câu so sánh, build
// chọn như thường.
func (s *Set) comparison(matched []Def, first map[string]int, in contracts.NormalizedInput, res contracts.Resolution, now time.Time, implicit string, w *Why) (Result, bool) {
	var named, derived []Def
	metrics := map[string]string{}
	for _, d := range matched {
		switch m, _ := metricOf(d, in, implicit); {
		case d.Derived != "":
			derived = append(derived, d)
		case !d.Weak && contracts.RouteKind(d.Route) == contracts.RouteSQLQuery && m != "":
			named, metrics[d.Name] = append(named, d), m
		}
	}
	if len(named) == 0 || len(derived) == 0 && !s.comparing(in.Raw) {
		return Result{}, false
	}
	distinct := map[string]bool{}
	for _, d := range named {
		distinct[metrics[d.Name]] = true
	}
	if len(derived) > 0 {
		if s.opt.DerivedOperands == nil {
			return Result{}, false
		}
		inside := true // mọi chỉ số câu nêu đều nằm trong cặp mặc định của (một) ý định dẫn xuất
		for m := range distinct {
			inside = inside && slices.ContainsFunc(derived, func(d Def) bool {
				l, r, ok := s.opt.DerivedOperands(d.Derived)
				return ok && (m == l || m == r)
			})
		}
		if inside {
			return Result{}, false
		}
	}
	if len(named) == 2 && len(distinct) == 2 {
		left, right := named[0], named[1]
		if first[right.Name] < first[left.Name] {
			left, right = right, left
		}
		r := s.fill(left, in, res, now, implicit, w)
		r.Intent.Slots["compare_metric"] = contracts.Slot{Name: "compare_metric", Text: metrics[right.Name]}
		r.Intent.Rule += "+compare:" + right.Name
		r.Compare = right.Name
		w.reason("câu so sánh: %s so với %s (hai chỉ số khác nhau kèm dấu hiệu so sánh) — phép tính hiệu theo thứ tự nhắc, không phải câu mơ hồ", left.Name, right.Name)
		return r, true
	}
	if len(derived) == 0 {
		return Result{}, false
	}
	var names []string
	for _, d := range slices.Concat(derived, named) {
		names = append(names, d.Name)
	}
	w.reason("câu có dấu hiệu so sánh nhưng các chỉ số nêu (%s) không thành đúng một cặp hai chỉ số khác nhau — không chọn cặp thay người hỏi, hỏi lại", strings.Join(names, ", "))
	return Result{Ambiguous: names}, true
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
func (s *Set) fill(top Def, in contracts.NormalizedInput, res contracts.Resolution, now time.Time, implicit string, w *Why) Result {
	it := contracts.Intent{Name: top.Name, Confidence: 1, Slots: map[string]contracts.Slot{}, Rule: "intent:rules"}
	var candidates []contracts.EntityRef
	var unclear Unclear
	clashSlot := ""
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
				clashSlot = sd.Name
				w.reason("khe %s thiếu: câu nêu %d thực thể khác nhau cùng loại (%s) cho khe chỉ nhận một — không lấy thực thể nhắc đầu", sd.Name, len(clash), contracts.Brief(refKeys(clash), 6, "thực thể"))
				continue
			}
			if e, ok := firstOfTypes(res.Entities, sd.Types); ok {
				ref := e
				it.Slots[sd.Name] = contracts.Slot{Name: sd.Name, Entity: &ref}
				w.reason("khe %s ← %s (thực thể rõ đầu tiên đúng loại)", sd.Name, e.Key)
				continue
			}
			if sd.Required {
				it.Missing = append(it.Missing, sd.Name)
				w.reason("khe %s thiếu: không có thực thể rõ đúng loại (%s)%s", sd.Name, orAny(sd.Types), ambiguousHint(res))
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
				w.reason("khe %s thiếu: câu nêu %d mốc thời gian khác nhau và %d ngày hỏng — không lấy mốc nhắc đầu, không rơi về ngày mặc định", sd.Name, len(ps), len(in.InvalidDates))
				continue
			}
			if len(ps) == 1 { // kỳ duy nhất (kỳ hẹp, nếu câu viết lồng nhiều kỳ)
				v := ps[0].Value
				it.Slots[sd.Name] = contracts.Slot{Name: sd.Name, Value: &v, Text: ps[0].Raw}
				// giá trị mốc đã chuẩn hoá (ISO) cũng là chữ lấy từ câu của người hỏi: cũng qua Quote để câu mang bí mật dạng ngày không để lọt nó
				w.reason("khe %s ← %s (mốc %s trong câu)", sd.Name, contracts.Quote(v.Text), contracts.Quote(ps[0].Raw))
				continue
			}
			if v, ok := defaultPeriod(sd.Default, now); ok {
				it.Slots[sd.Name] = contracts.Slot{Name: sd.Name, Value: &v, Text: sd.Default}
				w.reason("khe %s ← %s (câu không nêu mốc: mặc định %s của pack)", sd.Name, v.Text, sd.Default)
				continue
			}
			if sd.Required {
				it.Missing = append(it.Missing, sd.Name)
				w.reason("khe %s thiếu: câu không nêu mốc và pack không có mặc định", sd.Name)
			}
		}
	}
	// Chỉ số do thuật ngữ trong câu quyết định; không có thì do thuật ngữ hiểu ngầm.
	if m, rule := metricOf(top, in, implicit); m != "" {
		it.Slots["metric"] = contracts.Slot{Name: "metric", Text: m}
		it.Rule += rule
		w.reason("chỉ số %s (theo thuật ngữ %s của ý định %s)", m, termFor(top, in, implicit), top.Name)
	}
	if top.Derived != "" {
		it.Slots["metric"] = contracts.Slot{Name: "metric", Text: top.Derived}
		w.reason("chỉ số dẫn xuất %s (ý định %s khai)", top.Derived, top.Name)
	}
	if len(top.MetricByTerm) > 0 || top.Derived != "" {
		if _, ok := it.Slots["metric"]; !ok {
			it.Missing = append(it.Missing, "metric")
			w.reason("chỉ số thiếu: ý định %s chọn chỉ số theo thuật ngữ mà câu không có thuật ngữ nào trong %s", top.Name, mapKeys(top.MetricByTerm))
		}
	}
	if len(unclear.Entities) > 0 {
		candidates = nil // thực thể người hỏi viết rõ đang tranh nhau: không để bí danh mơ hồ làm router chọn thay
	}
	w.Origins = originsOf(res, candidates, unclear.Entities, clashSlot)
	w.reason("luật %s", it.Rule)
	return Result{Intent: it, Candidates: candidates, Unclear: unclear}
}

// clauseSpans trả khoảng byte [đầu, cuối) của từng mệnh đề (ranh giới như clausesOf, không gồm dấu ngắt).
func clauseSpans(text string) (spans [][2]int) {
	start := 0
	for i := 0; i < len(text); i++ {
		if strings.IndexByte(",;.?!", text[i]) >= 0 && (i+1 >= len(text) || text[i+1] == ' ' || text[i+1] == '\n') {
			spans = append(spans, [2]int{start, i})
			start = i + 1
		}
	}
	return append(spans, [2]int{start, len(text)})
}

// premiseRanges trả các khoảng byte của MỆNH ĐỀ NGỮ CẢNH (ADR-0010 bất biến (b), vai mệnh đề): mệnh đề đứng TRƯỚC mệnh đề hỏi mà nêu một con số.
// Mệnh đề hỏi là mệnh đề đầu tiên có cụm xác nhận / cụm hỏi lượng của pack (dấu '?' một mình không đủ: "…được cấp 4k dầu, anh cứ xác nhận là đúng
// giúp em để em báo cáo gấp nhé?" kết thúc bằng '?' nhưng mệnh đề cuối chỉ nhờ xác nhận chứ không phải câu hỏi mới); nếu mệnh đề đó chỉ còn cụm hỏi và từ trung tính
// ("…, đúng không?") thì câu hỏi áp cho mệnh đề liền trước nó. Chữ trong mệnh đề nêu con số là chữ chỉ NGUỒN của con số ấy ("phiếu kho ghi 3.850",
// "hoá đơn ghi 4.100 L", "đội báo 4k"), không phải điều đang hỏi, nên không được quyết định ý định. Mệnh đề dẫn đề không nêu con số ("Về hoá đơn, …") vẫn
// nói chỉ số đang hỏi và mệnh đề đứng sau câu hỏi là phần bổ nghĩa của nó, nên cả hai ở lại.
func (s *Set) premiseRanges(in contracts.NormalizedInput, res contracts.Resolution) [][2]int {
	raw := in.Raw
	cl := clauseSpans(raw)
	if len(cl) < 2 {
		return nil
	}
	markers := append(slices.Clone(s.opt.FocusMarkers), s.opt.QuantityMarkers...)
	q := slices.IndexFunc(cl, func(c [2]int) bool {
		return slices.ContainsFunc(markers, func(m string) bool { return normalization.ContainsPhrase(raw[c[0]:c[1]], m) })
	})
	if q > 0 && s.onlyMarkers(raw[cl[q][0]:cl[q][1]], markers) {
		q--
	}
	var out [][2]int
	for i := 0; i < q; i++ {
		if hasFigure(in, res, cl[i][0], cl[i][1]) {
			out = append(out, cl[i])
		}
	}
	// Chỉ che khi mệnh đề hỏi CÓ chỉ số riêng. Mệnh đề hỏi chỉ là cái đuôi ("…, tại bể 7 đúng không?", "…, hay 4000 L đúng không?") thì con số và
	// chữ chỉ số nằm ở mệnh đề đứng trước chính là câu hỏi, không phải ngữ cảnh.
	if len(out) == 0 || !s.ownMetric(raw[cl[q][0]:cl[q][1]], hasFigure(in, res, cl[q][0], cl[q][1])) {
		return nil
	}
	return out
}

// ownMetric báo một mệnh đề tự nêu chỉ số: (a) có từ khoá của ý định số liệu riêng (không yếu) — kể cả khi chữ chen giữa tách các chữ của từ khoá
// nhiều chữ ("kho cũng xuất" với "kho xuất"); hoặc (b) là một lời khẳng định đủ bộ — có con số của riêng nó VÀ một chữ của từ khoá như thế ("được xuất
// 4.100 L" với "kho xuất"). Chữ của từ khoá đứng với nhãn ("theo phiếu số 912") chưa phải chỉ số: không có con số thì mệnh đề chỉ là cái đuôi.
func (s *Set) ownMetric(clause string, hasOwnFigure bool) bool {
	for _, d := range s.defs {
		if d.Weak || len(d.Terms) == 0 || contracts.RouteKind(d.Route) == contracts.RouteRAG {
			continue
		}
		for _, k := range d.Keywords {
			missing := 0
			for _, w := range strings.Fields(k) {
				if len(normalization.PhraseSpans(clause, w)) == 0 {
					missing++
				}
			}
			if missing == 0 {
				return true
			}
			if hasOwnFigure && missing < len(strings.Fields(k)) && !s.neutralOnly(k, clause) {
				return true
			}
		}
	}
	return false
}

// neutralOnly báo mọi chữ của từ khoá có mặt trong mệnh đề đều là từ trung tính của pack ("được" của "được cấp", "đo" của "đo được").
func (s *Set) neutralOnly(keyword, clause string) bool {
	for _, w := range strings.Fields(keyword) {
		if len(normalization.PhraseSpans(clause, w)) == 0 {
			continue
		}
		if !s.isNeutral(w) {
			return false
		}
	}
	return true
}

// onlyMarkers báo mệnh đề chỉ còn cụm hỏi và từ trung tính (không còn chữ nào khác).
func (s *Set) onlyMarkers(clause string, markers []string) bool {
	b := []byte(clause)
	for _, p := range append(slices.Clone(markers), s.opt.NeutralWords...) {
		for _, sp := range normalization.PhraseSpans(clause, p) {
			for j := sp[0]; j < sp[1]; j++ {
				b[j] = ' '
			}
		}
	}
	return !strings.ContainsFunc(string(b), func(r rune) bool { return unicode.IsLetter(r) || unicode.IsDigit(r) })
}

// hasFigure báo khoảng [start, end) của câu có một CON SỐ: lượng, tiền, hay số trần không nằm trong tên thực thể ("03" của "máy 03"), không là
// nhãn ("phiếu 912") và không là số của mốc thời gian ("tuần 43").
func hasFigure(in contracts.NormalizedInput, res contracts.Resolution, start, end int) bool {
	inName := func(sp contracts.Span) bool {
		return slices.ContainsFunc(res.Entities, func(e contracts.EntityRef) bool { return e.Start <= sp.Start && sp.End <= e.End }) ||
			slices.ContainsFunc(res.Ambiguous, func(m contracts.AmbiguousMention) bool { return m.Start <= sp.Start && sp.End <= m.End })
	}
	return slices.ContainsFunc(in.Spans, func(sp contracts.Span) bool {
		if sp.Start < start || sp.End > end {
			return false
		}
		switch sp.Kind {
		case contracts.ValueQuantity, contracts.ValueMoney:
			return true
		case contracts.ValueNumber:
			_, period := normalization.PeriodQualifier(in.Raw, sp.Start, sp.End)
			return !inName(sp) && !period && !normalization.LabelContext(in.Raw, sp.Start, sp.End)
		}
		return false
	})
}

// maskRanges thay mọi byte trong các khoảng bằng dấu cách (giữ nguyên độ dài và vị trí) để phép đếm từ khoá bỏ qua chúng.
func maskRanges(text string, ranges [][2]int) string {
	if len(ranges) == 0 {
		return text
	}
	b := []byte(text)
	for _, r := range ranges {
		for i := r[0]; i < r[1]; i++ {
			b[i] = ' '
		}
	}
	return string(b)
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

// scopeSignal báo câu có tín hiệu thuộc phạm vi số liệu (ADR-0010 (a), định nghĩa D1s của bước điều tra 22/10): có thực thể (đã giải hay bí danh
// mơ hồ) VÀ một trong hai: thuật ngữ mà ít nhất một ý định đòi, hoặc — khi câu không nêu thuật ngữ nào và thuật ngữ mặc định của tenant áp được
// (implicit) — một lượng cùng chiều đo với chỉ số ("3850 L", "1,2 khối") cùng một chữ của từ khoá chỉ số ("xuất", "cấp"). Một trong những thứ đó thì
// chưa đủ: "VN-Index bao nhiêu?", "dầu ăn", tên máy trong chuyện phiếm, "EX-03 chứa 300 lít nước", "EX-03 nặng 20 tấn" vẫn ngoài phạm vi.
func (s *Set) scopeSignal(in contracts.NormalizedInput, res contracts.Resolution, implicit string) bool {
	if len(res.Entities)+len(res.Ambiguous) == 0 {
		return false
	}
	if slices.ContainsFunc(in.SpansOf(contracts.ValueTerm), func(sp contracts.Span) bool {
		return slices.ContainsFunc(s.defs, func(d Def) bool { return slices.Contains(d.Terms, sp.Value.Text) })
	}) {
		return true
	}
	return implicit != "" && s.volumeFigure(in) && s.metricWord(in.Raw)
}

// volumeFigure báo câu có một lượng đổi được sang đơn vị của chỉ số nào đó của các ý định số liệu (cùng chiều đo: lít, khối).
func (s *Set) volumeFigure(in contracts.NormalizedInput) bool {
	if s.opt.QuantityFits == nil {
		return false
	}
	return slices.ContainsFunc(in.SpansOf(contracts.ValueQuantity), func(sp contracts.Span) bool {
		return slices.ContainsFunc(s.defs, func(d Def) bool {
			for _, m := range d.MetricByTerm {
				if s.opt.QuantityFits(m, sp.Value) {
					return true
				}
			}
			return false
		})
	})
}

// metricWord báo câu có một chữ của từ khoá ý định số liệu riêng (không yếu) mà pack không khai là từ trung tính.
func (s *Set) metricWord(text string) bool {
	for _, d := range s.defs {
		if d.Weak || len(d.Terms) == 0 || contracts.RouteKind(d.Route) == contracts.RouteRAG {
			continue
		}
		for _, k := range d.Keywords {
			for _, w := range strings.Fields(k) {
				if !s.isNeutral(w) && len(normalization.PhraseSpans(text, w)) > 0 {
					return true
				}
			}
		}
	}
	return false
}

// isNeutral báo một chữ là từ trung tính của pack (NeutralWords).
func (s *Set) isNeutral(w string) bool {
	return slices.ContainsFunc(s.opt.NeutralWords, func(n string) bool {
		sp := normalization.PhraseSpans(w, n)
		return len(sp) > 0 && sp[0][0] == 0 && sp[0][1] == len(w)
	})
}

// inRange báo số nằm trong miền một lần đo của chỉ số (không khai ValueInRange thì không chặn).
func (s *Set) inRange(metric string, v contracts.Value) bool {
	return s.opt.ValueInRange == nil || s.opt.ValueInRange(metric, v)
}

// cuePhrases trả mọi cụm báo nghĩa chưa hỗ trợ của pack.
func (s *Set) cuePhrases() []string {
	var out []string
	for _, u := range s.opt.Unsupported {
		out = append(out, u.Cues...)
	}
	return out
}

// unsupportedIn trả các nghĩa chưa hỗ trợ mà câu mang (D-OP-18 = c): câu có thực thể, và trong phần quyết định (mệnh đề hỏi trở đi) có cụm báo nghĩa đó
// (cụm nằm ngoài các mệnh đề ngữ cảnh). Nghĩa NeedsValue (lệnh ghi, đánh giá) còn đòi câu nêu một con số và không phải câu hỏi. Câu có từ khoá
// tri thức ("quy trình cảnh báo …") là câu hỏi quy trình, tài liệu trả lời được: không tính.
func (s *Set) unsupportedIn(in contracts.NormalizedInput, res contracts.Resolution, text string, prem [][2]int, counts []int) []Unsupported {
	if len(s.opt.Unsupported) == 0 || len(res.Entities)+len(res.Ambiguous) == 0 {
		return nil
	}
	for i, d := range s.defs {
		if counts[i] > 0 && contracts.RouteKind(d.Route) == contracts.RouteRAG {
			return nil
		}
	}
	qStart := 0
	if len(prem) > 0 {
		qStart = prem[len(prem)-1][1]
	}
	var out []Unsupported
	for _, u := range s.opt.Unsupported {
		hit := slices.ContainsFunc(u.Cues, func(c string) bool { return len(normalization.PhraseSpans(text, c)) > 0 })
		if hit && (!u.NeedsValue || hasFigure(in, res, qStart, len(in.Raw)) && !s.asking(text)) {
			out = append(out, u)
		}
	}
	return out
}

// asking báo phần câu là một CÂU HỎI: có cụm xác nhận / cụm hỏi lượng của pack hoặc dấu '?'.
func (s *Set) asking(text string) bool {
	return strings.Contains(text, "?") || slices.ContainsFunc(slices.Concat(s.opt.FocusMarkers, s.opt.QuantityMarkers), func(m string) bool { return normalization.ContainsPhrase(text, m) })
}

// meaningOptions là các ý định số liệu đòi thuật ngữ (không yếu, không tri thức): những nghĩa có thể của câu khi hỏi lại.
func (s *Set) meaningOptions() []string {
	var out []string
	for _, d := range s.defs {
		if len(d.Terms) > 0 && !d.Weak && contracts.RouteKind(d.Route) != contracts.RouteRAG {
			out = append(out, d.Name)
		}
	}
	return out
}

// generic chọn ý định chung (yếu) để trả lời câu có thực thể + thuật ngữ mà không từ khoá nào khớp (D-OP-15 = a2). Câu phải có cụm xác
// nhận; đúng MỘT ý định yếu đòi thuật ngữ của câu; mọi chữ còn lại — ngoài tên thực thể và chữ chỉ loại của nó ("dự án" của "dự án
// Cầu A"), mốc, lượng cùng chiều đo với chỉ số chung, số trần, thuật ngữ của ý định chung, cụm xác nhận, từ trung tính — phải là mảnh
// từ khoá của ý định riêng mà ý định chung quy về (cùng chỉ số theo metric_by_term: "xuất" của "kho xuất"), và phải có ÍT NHẤT một
// mảnh như thế: câu xác nhận không nêu chỉ số ("Hôm qua máy 03 4k dầu đúng không?") thì hỏi lại (BR-FUEL-02 của tenant: chưa rõ chỉ
// số mà các nguồn đang lệch thì hỏi lại, không xác nhận).
// Mảnh từ khoá của ý định số liệu khác ("tiêu" của "tiêu thụ", "đo" của "đo được") không được coi là trung tính. Còn chữ lạ ("xài",
// "hao", "đổ", "tính tiền", "định mức") ⇒ không chọn: câu có thể hỏi một chỉ số khác, nên hỏi lại (D-OP-15 điều kiện (ii)).
func (s *Set) generic(in contracts.NormalizedInput, res contracts.Resolution, implicit string) (Def, bool, string) {
	raw := in.Raw
	if !slices.ContainsFunc(s.opt.FocusMarkers, func(m string) bool { return len(normalization.PhraseSpans(raw, m)) > 0 }) {
		return Def{}, false, "câu không có cụm xác nhận"
	}
	var gen []Def
	term := ""
	for _, sp := range in.SpansOf(contracts.ValueTerm) {
		for _, d := range s.defs {
			if d.Weak && slices.Contains(d.Terms, sp.Value.Text) && !slices.ContainsFunc(gen, func(g Def) bool { return g.Name == d.Name }) {
				gen, term = append(gen, d), sp.Value.Text
			}
		}
	}
	if len(gen) == 0 && implicit != "" { // câu không nêu thuật ngữ: thuật ngữ mặc định của tenant (default_terms) áp, như mọi ý định khác
		for _, d := range s.defs {
			if d.Weak && slices.Contains(d.Terms, implicit) {
				gen, term = append(gen, d), implicit
			}
		}
	}
	if len(gen) != 1 || gen[0].MetricByTerm[term] == "" {
		return Def{}, false, fmt.Sprintf("không có đúng một ý định yếu đòi thuật ngữ của câu (có %d)", len(gen))
	}
	metric := gen[0].MetricByTerm[term]
	var own, other []string // từng chữ của từ khoá: ý định riêng cùng chỉ số với ý định chung / ý định số liệu khác
	for _, d := range s.defs {
		if d.Weak || len(d.Terms) == 0 || contracts.RouteKind(d.Route) == contracts.RouteRAG {
			continue
		}
		for _, k := range d.Keywords {
			if d.MetricByTerm[term] == metric {
				own = append(own, strings.Fields(k)...)
			} else {
				other = append(other, strings.Fields(k)...)
			}
		}
	}
	whole := func(w string, list []string) bool {
		return slices.ContainsFunc(list, func(c string) bool {
			sp := normalization.PhraseSpans(w, c)
			return len(sp) > 0 && sp[0][0] == 0 && sp[0][1] == len(w)
		})
	}
	covered := make([]bool, len(raw))
	cover := func(st, en int) {
		for i := max(st, 0); i < en && i < len(raw); i++ {
			covered[i] = true
		}
	}
	var types []string // loại của thực thể đã giải: thuật ngữ cùng tên loại là chữ chỉ loại ("dự án" của "dự án Cầu A")
	for _, e := range res.Entities {
		cover(e.Start, e.End)
		types = append(types, e.Type)
	}
	for _, m := range res.Ambiguous {
		cover(m.Start, m.End)
	}
	for _, sp := range in.Spans {
		switch sp.Kind {
		case contracts.ValueDate, contracts.ValuePeriod:
			cover(sp.Start, sp.End)
			if ws, we, ok := normalization.PeriodWordBefore(raw, sp.Start); ok {
				cover(ws, we)
			}
		case contracts.ValueTerm: // thuật ngữ của ý định chung ("dầu") và chữ chỉ loại; "giá", "giờ máy", "nhân công" là đại lượng khác
			if slices.Contains(gen[0].Terms, sp.Value.Text) || slices.Contains(types, sp.Value.Text) {
				cover(sp.Start, sp.End)
			}
		case contracts.ValueQuantity: // chỉ lượng cùng chiều đo với chỉ số chung ("4 khối"); "7 tấn", "8h" là đại lượng khác
			if s.opt.QuantityFits != nil && s.opt.QuantityFits(metric, sp.Value) && (!strings.Contains(sp.Rule, "unit_from_term") || s.inRange(metric, sp.Value)) {
				cover(sp.Start, sp.End)
			}
		case contracts.ValueNumber: // số trần không đứng cạnh "dầu" nên không được gắn đơn vị: "Dầu xuất hôm nay của EX-03 là 3850 …"; ngoài miền chỉ số
			// ("4 củ" = 4.000.000) thì không phải giá trị của chỉ số
			if s.inRange(metric, sp.Value) {
				cover(sp.Start, sp.End)
			}
		case contracts.ValueCode: // mã chứng từ ("PX-0912") là nhãn, không mang nghĩa chỉ số
			cover(sp.Start, sp.End)
		}
	}
	for _, p := range s.opt.FocusMarkers {
		for _, sp := range normalization.PhraseSpans(raw, p) {
			cover(sp[0], sp[1])
		}
	}
	for _, c := range s.cuePhrases() { // cụm báo nghĩa chưa hỗ trợ không phải lệnh khi câu là câu hỏi ("ghi nhận" = đã ghi): không mang nghĩa chỉ số
		for _, sp := range normalization.PhraseSpans(raw, c) {
			cover(sp[0], sp[1])
		}
	}
	for _, r := range s.premiseRanges(in, res) { // mệnh đề ngữ cảnh: không phải điều đang hỏi nên không cần giải thích (bất biến (b))
		cover(r[0], r[1])
	}
	neutral := slices.Clone(s.opt.NeutralWords) // chữ của chính cụm xác nhận ("đúng" trong "xuất đúng 851 L … phải không") cũng không mang nghĩa chỉ số
	for _, m := range s.opt.FocusMarkers {
		neutral = append(neutral, strings.Fields(m)...)
	}
	for _, p := range neutral {
		if whole(p, other) && !whole(p, own) {
			continue // "đo": trung tính với cổng phán quyết, nhưng là mảnh của "đo được" (tiêu thụ đo được)
		}
		for _, sp := range normalization.PhraseSpans(raw, p) {
			cover(sp[0], sp[1])
		}
	}
	meaning := false // có chữ nêu nghĩa của chỉ số chung
	for i := 0; i < len(raw); {
		r, n := utf8.DecodeRuneInString(raw[i:])
		if covered[i] || !(unicode.IsLetter(r) || unicode.IsDigit(r)) {
			i += n
			continue
		}
		j := i
		for j < len(raw) && !covered[j] {
			rr, m := utf8.DecodeRuneInString(raw[j:])
			if !unicode.IsLetter(rr) && !unicode.IsDigit(rr) {
				break
			}
			j += m
		}
		// so khớp trong CẢ câu (cùng luật dấu với từ trung tính): "duoc" không dấu trong câu có dấu không phải "được"
		if !slices.ContainsFunc(own, func(c string) bool {
			return slices.ContainsFunc(normalization.PhraseSpans(raw, c), func(sp [2]int) bool { return sp[0] == i && sp[1] == j })
		}) {
			return Def{}, false, fmt.Sprintf("chữ %s chưa được giải thích (không phải tên, mốc, lượng, thuật ngữ, cụm xác nhận, từ trung tính hay mảnh từ khoá của ý định riêng)", contracts.Quote(raw[i:j]))
		}
		meaning = true
		i = j
	}
	if !meaning {
		return gen[0], false, "câu không nêu nghĩa số liệu nào (không có mảnh từ khoá của ý định riêng)"
	}
	return gen[0], true, "mọi chữ còn lại được giải thích (tên, mốc, lượng, thuật ngữ, cụm xác nhận, mảnh từ khoá của ý định riêng)"
}

// keywordHits báo ý định nào có từ khoá khớp, theo luật KHỚP DÀI NHẤT trên từ khoá của MỌI ý
// định: lần khớp nằm trọn trong lần khớp dài hơn của từ khoá khác thì không tính. Câu gõ
// không dấu "bao nhieu" khớp cả "bao nhiêu" lẫn "báo" — "báo" nằm trong "bao nhiêu" nên bị
// bỏ; xét từng ý định riêng lẻ thì câu hỏi chung chung bị hiểu thành "dầu đội báo".
func keywordHits(defs []Def, text string, ignore []string) (counts, first []int, phrases [][]string) {
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
	counts, first, phrases = make([]int, len(defs)), make([]int, len(defs)), make([][]string, len(defs))
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
			phrases[h.def] = append(phrases[h.def], text[h.start:h.end])
		}
	}
	return counts, first, phrases
}

// hitsText kể các ý định có từ khoá khớp kèm cụm đã khớp, theo thứ tự khai báo ("fuel_issued [được cấp] · fuel_generic [bao nhiêu]").
func hitsText(defs []Def, counts []int, phrases [][]string) string {
	var out []string
	for i, d := range defs {
		if counts[i] > 0 {
			out = append(out, fmt.Sprintf("%s [%s]", d.Name, quoteAll(phrases[i])))
		}
	}
	return strings.Join(out, " · ")
}

// quoteAll đặt từng đoạn chữ NGUYÊN VĂN của người hỏi (cụm từ khoá khớp trong câu) vào cặp “…” rồi nối bằng ", ": mọi chữ lấy từ câu đều qua
// contracts.Quote để Why.WithholdQuoted che được khi câu mang bí mật.
func quoteAll(phrases []string) string {
	quoted := make([]string, len(phrases))
	for i, p := range phrases {
		quoted[i] = contracts.Quote(p)
	}
	return strings.Join(quoted, ", ")
}

// defNames là tên các ý định, nối bằng ", ".
func defNames(defs []Def) string {
	names := make([]string, len(defs))
	for i, d := range defs {
		names[i] = d.Name
	}
	return strings.Join(names, ", ")
}

// refKeys là khoá các thực thể, theo thứ tự đưa vào.
func refKeys(refs []contracts.EntityRef) []string {
	out := make([]string, len(refs))
	for i, r := range refs {
		out[i] = r.Key
	}
	return out
}

func orAny(types []string) string {
	if len(types) == 0 {
		return "mọi loại"
	}
	return strings.Join(types, "|")
}

// ambiguousHint nhắc các cụm mơ hồ của S3 khi khe thực thể thiếu (router còn có thể gỡ ở S5 bằng phạm vi / dự án).
func ambiguousHint(res contracts.Resolution) string {
	if len(res.Ambiguous) == 0 {
		return ""
	}
	var ms []string
	for _, m := range res.Ambiguous {
		ms = append(ms, contracts.Quote(m.Mention))
	}
	return " — có cụm mơ hồ " + strings.Join(ms, ", ") + " (router có thể gỡ ở S5 bằng dự án đã nêu hay phạm vi dữ liệu của người hỏi)"
}

// termFor trả thuật ngữ đã quyết chỉ số (cùng thứ tự với metricOf): thuật ngữ trong câu, không có thì thuật ngữ hiểu ngầm.
func termFor(d Def, in contracts.NormalizedInput, implicit string) string {
	for _, sp := range in.SpansOf(contracts.ValueTerm) {
		if _, ok := d.MetricByTerm[sp.Value.Text]; ok {
			return sp.Value.Text
		}
	}
	return implicit + " (hiểu ngầm)"
}

// mapKeys liệt kê khoá của map theo thứ tự tăng dần.
func mapKeys(m map[string]string) string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return strings.Join(keys, "|")
}

// originsOf ghi nơi sinh ra từng ứng viên dùng để hỏi lại: ứng viên của cụm mơ hồ do S3 thấy (cands), và các thực thể S4 gộp thêm vì khe
// chỉ nhận một mà câu nêu nhiều (clash) — mỗi thực thể gộp kèm gốc của nó ở S3 (thực thể rõ, hay ứng viên của một cụm mơ hồ).
func originsOf(res contracts.Resolution, cands, clash []contracts.EntityRef, clashSlot string) []Origin {
	mentionOf := func(id string) string {
		for _, m := range res.Ambiguous {
			for _, c := range m.Candidates {
				if c.ID == id {
					return m.Mention
				}
			}
		}
		return ""
	}
	// mỗi thực thể một nguồn (nguồn đầu tiên): một bí danh nhắc nhiều lần không sinh hàng nghìn dòng trùng
	var w Why
	for _, c := range cands {
		if !slices.ContainsFunc(w.Origins, func(o Origin) bool { return o.ID == c.ID }) {
			w.addOrigin(Origin{ID: c.ID, From: OriginS3, Note: fmt.Sprintf("ứng viên của cụm mơ hồ %s", contracts.Quote(mentionOf(c.ID)))})
		}
	}
	for _, e := range clash {
		why := fmt.Sprintf("khe %s chỉ nhận một thực thể mà câu nêu %d", clashSlot, len(clash))
		if slices.ContainsFunc(res.Entities, func(r contracts.EntityRef) bool { return r.ID == e.ID }) {
			w.addOrigin(Origin{ID: e.ID, From: OriginS4, Note: fmt.Sprintf("thực thể rõ %s ở S3, S4 gộp vào danh sách vì %s", contracts.Quote(e.Mention), why)})
		} else if !slices.ContainsFunc(w.Origins, func(o Origin) bool { return o.ID == e.ID }) {
			w.addOrigin(Origin{ID: e.ID, From: OriginS3, Note: fmt.Sprintf("ứng viên của cụm mơ hồ %s, S4 giữ trong danh sách vì %s", contracts.Quote(mentionOf(e.ID)), why)})
		}
	}
	return w.Origins
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
//
// Danh sách trả về (D-OP-11 = b, ADR-0010 bất biến (c)): có bí danh mơ hồ trong số các lần nhắc tranh nhau thì CHỈ gồm ứng viên của bí
// danh — thực thể đã giải không phải thứ người hỏi cần chọn, nó chỉ góp vào việc đếm để biết câu có nhiều chủ thể; hai thực thể đều đã
// giải thì giữ cách liệt kê cũ (cả hai).
func clashing(res contracts.Resolution, types []string) []contracts.EntityRef {
	var uniq []contracts.EntityRef
	count, seen := map[string]int{}, map[string]bool{}
	viable := map[string]bool{} // ứng viên của bí danh mơ hồ (khác thực thể đã giải)
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
			viable[c.ID] = true
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
	if slices.ContainsFunc(out, func(e contracts.EntityRef) bool { return viable[e.ID] }) {
		out = slices.DeleteFunc(out, func(e contracts.EntityRef) bool { return !viable[e.ID] })
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
