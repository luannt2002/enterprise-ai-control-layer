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
	// Alone: ý định chỉ khớp khi câu không còn nội dung nào khác (không thực thể, không con số, không thuật ngữ) — lời chào: "chào cờ", "chào mừng khởi công dự án A" không phải lời chào.
	Alone bool `yaml:"alone"`
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
	// WeakCues: cụm QUÁ CHUNG của nghĩa này ("sẽ", "dự kiến", "ước tính"): chữ thường gặp ở câu không hỏi dự báo ("Anh sẽ duyệt, …"). Chúng chỉ kích hoạt khi chính MỆNH ĐỀ HỎI mang chúng và câu hỏi về một chỉ
	// số (có từ khoá chỉ số hay thuật ngữ vật tư); vẫn chiếm chỗ trong luật khớp dài nhất như Cues (S4BS-05).
	WeakCues []string `yaml:"weak_cues"`
	// NeedsValue: chỉ tính khi câu nêu một CON SỐ và không phải câu hỏi (lệnh ghi "Ghi nhận EX-03 tiêu thụ 3782 L"). Không có giá trị, hay là câu hỏi, thì cụm vẫn
	// chiếm chỗ nhưng câu là câu hỏi số liệu như thường (D-OP-18 = c).
	NeedsValue bool `yaml:"needs_value"`
	// Signals: tín hiệu CẤU TRÚC của nghĩa này, ngoài danh sách cụm (xem ValidSignal): câu có nó thì mang nghĩa này dù không có chữ nào trong Cues. Cụm là danh
	// sách theo HỌ NGHĨA (đánh đổi ghi ở ADR-0010); tín hiệu cấu trúc phủ phần danh sách không với tới (đơn vị tiền sau số, "bao nhiêu đồng", số tiền nói tắt).
	Signals []string `yaml:"signals"`
}

// Options là chính sách phân loại do pack khai.
type Options struct {
	// Unsupported: các nghĩa hệ chưa hỗ trợ (xem Unsupported).
	Unsupported []Unsupported
	// SourceVerbs: động từ của NGUỒN ("báo"): "<tên nguồn> <động từ>" ("cảm biến báo", "hoá đơn báo", "ERP báo") là chữ của nguồn chứ không phải chỉ số đội báo — SetOptions sinh cụm này cho ý định của chỉ số mà nguồn
	// đó cung cấp, dùng tên nguồn do SourceNames trả (S4BS-06).
	SourceVerbs []string
	// SourceNames trả tên các nguồn cấp số liệu của một chỉ số (nhãn và bí danh ở metrics.yaml: "cảm biến", "phiếu kho", "ERP"); metric rỗng = mọi chỉ số. Cụm báo nghĩa ở mệnh đề có tên nguồn
	// là động từ của NGUỒN ("theo ghi nhận của cảm biến"), không phải yêu cầu của người hỏi (S4BS-03).
	SourceNames func(metric string) []string
	// ConditionMarkers: từ nối mở một mệnh đề ĐIỀU KIỆN ("nếu", "giả sử"): mệnh đề bắt đầu bằng chúng nêu giả thiết, không phải điều đang hỏi nên cụm báo nghĩa trong đó không kích hoạt (S4BS-05).
	ConditionMarkers []string
	// NegationMarkers: từ phủ định ("không", "chưa", "chẳng"). Đứng CUỐI câu chúng là đuôi hỏi có / không ("… dùng quá 4000 L dầu không"): một dấu hiệu hỏi như cụm hỏi và dấu "?" (xem Set.questioned).
	NegationMarkers []string
	// PurposeMarkers: từ nối mở một đoạn MỤC ĐÍCH ("để", "nhằm"). Đoạn từ từ nối tới hết mệnh đề nói VÌ SAO người hỏi cần câu trả lời, không nói câu trả lời là chỉ số nào nên chữ trong
	// đó không góp từ khoá chỉ số và không kích hoạt cụm báo nghĩa — trừ khi chính đoạn đó mang cụm hỏi (nó LÀ câu hỏi: "…, để biết đội báo bao nhiêu dầu?"). S4BS-01 (WIN-I-086).
	PurposeMarkers []string
	// DismissMarkers: cụm cho biết mệnh đề BỊ BỎ QUA ("không quan tâm", "để sau", "khỏi cần"): cụm báo nghĩa chưa hỗ trợ nằm trong mệnh đề đó không làm câu thành câu hỏi lại.
	DismissMarkers []string
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
	// UnitAliases trả mọi cách viết của các đơn vị thuộc một chiều đo ("currency" → đ, đồng, vnđ, vnd), lấy từ lexicon của pack — cho tín hiệu "asked_unit".
	UnitAliases func(dimension string) []string
	// Measurable báo một thuật ngữ là đại lượng đo được (có đơn vị mặc định) — có đại lượng khác
	// trong câu thì KHÔNG hiểu ngầm thuật ngữ mặc định ("giá", "nhân công").
	Measurable func(term string) bool
}

// SetOptions gắn chính sách phân loại của pack. Pack khai động từ nguồn (SourceVerbs) thì sinh luôn cụm "<tên nguồn> <động từ>" ("cảm biến báo", "hoá đơn báo", "ERP báo") cho ý định của chỉ số mà nguồn đó
// cung cấp (S4BS-06): chữ "báo" ấy là động từ của NGUỒN, không phải chỉ số đội báo; cụm dài thắng "báo" trong luật khớp dài nhất.
func (s *Set) SetOptions(o Options) {
	s.opt, s.hit = o, nil
	if o.SourceNames == nil || len(o.SourceVerbs) == 0 {
		return
	}
	hit := slices.Clone(s.defs)
	for i, d := range hit {
		if d.Weak || len(d.MetricByTerm) == 0 {
			continue
		}
		var metrics []string
		for _, m := range d.MetricByTerm {
			if !slices.Contains(metrics, m) {
				metrics = append(metrics, m)
			}
		}
		sort.Strings(metrics)
		kw := slices.Clone(d.Keywords)
		for _, m := range metrics {
			for _, name := range o.SourceNames(m) {
				for _, verb := range o.SourceVerbs {
					if k := name + " " + verb; !slices.Contains(kw, k) {
						kw = append(kw, k)
					}
				}
			}
		}
		hit[i].Keywords = kw
	}
	s.hit = hit
}

// hitDefs là tập ý định dùng để ĐẾM TỪ KHOÁ: ý định của pack kèm cụm "<tên nguồn> <động từ>" sinh ở SetOptions (cùng thứ tự và chỉ số với s.defs). Các chỗ khác (chữ của từ khoá ở a2, ownMetric…)
// vẫn dùng từ khoá do pack khai: cụm sinh ra không thêm "mảnh từ khoá" nào cho a2 hiểu.
func (s *Set) hitDefs() []Def {
	if s.hit != nil {
		return s.hit
	}
	return s.defs
}

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
	hit  []Def // defs kèm cụm "<tên nguồn> <động từ>" sinh ở SetOptions (nil khi pack không khai); chỉ dùng để đếm từ khoá
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
	bg := s.backgroundRanges(in.Raw)
	rawBG := maskRanges(in.Raw, bg) // câu đã che đoạn mục đích; mệnh đề ngữ cảnh còn nguyên (unsupportedIn xét cả chúng)
	text := maskRanges(rawBG, prem)
	textKeepBG := maskRanges(in.Raw, prem) // chưa che đoạn mục đích: ý định không phải số liệu (quy trình) vẫn đếm trên đó
	for _, r := range bg {
		w.reason("đoạn mục đích %s nói vì sao người hỏi cần câu trả lời — chữ của nó không góp từ khoá chỉ số và không kích hoạt cụm báo nghĩa (S4BS-01)", contracts.Quote(in.Raw[r[0]:r[1]]))
	}
	for _, r := range prem {
		w.reason("mệnh đề ngữ cảnh %s nêu một con số và đứng trước mệnh đề hỏi — chữ của nó chỉ nguồn của con số, không quyết định ý định (bất biến (b))", contracts.Quote(in.Raw[r[0]:r[1]]))
	}
	ignore := s.ignored()
	counts, _, phrases := s.hitsBG(text, textKeepBG, ignore)
	if un := s.unsupportedIn(in, rawBG, res, prem, counts, w); len(un) > 0 {
		names := make([]string, len(un))
		for i, u := range un {
			names[i] = u.Name
		}
		w.reason("câu mang nghĩa hệ chưa hỗ trợ (%s) — hỏi lại và nói rõ nghĩa nào chưa hỗ trợ, kể các nghĩa số liệu có thể hỏi thay (D-OP-18 = c)", strings.Join(names, ", "))
		return Result{Unsupported: un, NeedMeaning: s.meaningOptions(), Mentioned: res.Entities, Candidates: aliasCandidates(res, w)}
	}
	if content := len(res.Entities)+len(res.Ambiguous) > 0 || len(in.Spans) > 0; content {
		for i, d := range s.defs {
			if d.Alone && counts[i] > 0 {
				counts[i] = 0
				w.reject("%s: từ khoá khớp nhưng câu còn nội dung khác (thực thể, con số hay thuật ngữ) nên không phải ý định thuần \"%s\"", d.Name, d.Label)
			}
		}
	}
	if fs, fe, ok := FocusClause(in.Raw, s.opt.FocusMarkers); ok {
		// "đội báo 4k, báo cáo ghi 4 m3, vậy máy 03 được cấp 4k dầu đúng không?" hỏi về CẤP, không
		// về BÁO: mệnh đề chứa cụm xác nhận quyết định ý định khi nó có từ khoá.
		if focus, _, fp := keywordHits(s.hitDefs(), in.Raw[fs:fe], ignore); slices.ContainsFunc(focus, func(n int) bool { return n > 0 }) {
			counts, phrases = focus, fp
			w.reason("từ khoá lấy trong mệnh đề hỏi %s (mệnh đề chứa cụm xác nhận quyết định ý định, các mệnh đề trước là ngữ cảnh)", contracts.Quote(in.Raw[fs:fe]))
		}
	}
	if hits := hitsText(s.defs, counts, phrases); hits != "" {
		w.reason("từ khoá khớp (luật khớp dài nhất): %s", hits)
	} else {
		w.reason("không từ khoá ý định nào khớp (luật khớp dài nhất)")
	}
	implicit, other := s.implicitTerm(in, res)
	if implicit != "" {
		w.reason("thuật ngữ hiểu ngầm %s (câu nhắc thực thể mà không nói vật tư và không có đại lượng đo được nào khác — chính sách của pack)", implicit)
	}
	var matched, partial []Def
	for i, d := range s.defs {
		if counts[i] == 0 {
			continue
		}
		kw := quoteAll(phrases[i])
		switch {
		case hasTerm(d, in) || (implicit != "" && requires(d, implicit)):
			matched = append(matched, d)
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
		return s.build(knowledge, text, in, res, now, implicit, w)
	}
	r := s.build(data, text, in, res, now, implicit, w)
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
		return s.build(knowledge, text, in, res, now, implicit, w)
	}
	k := s.build(knowledge, text, in, res, now, implicit, &Why{}) // chỉ để biết có hoà không; lời giải thích của nhóm tri thức gói ở dòng dưới
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
		counts, _, _ := keywordHits(s.hitDefs(), clause, s.opt.IgnorePhrases)
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
func (s *Set) build(matched []Def, text string, in contracts.NormalizedInput, res contracts.Resolution, now time.Time, implicit string, w *Why) Result {
	if r, ok := s.comparison(matched, text, in, res, now, implicit, w); ok {
		return r
	}
	// Ưu tiên cao hơn thắng; cùng ưu tiên mà cùng khớp đủ thì HOÀ ⇒ hỏi lại. Số cụm từ khoá khớp không phải tín hiệu nghĩa ("cảm biến … đo được" hai cụm không
	// thắng được "cấp" một cụm: "đo được" → "đo" đủ lật kết quả) nên không được phân xử thay người hỏi (S4AS-03, ADR-0010 C4).
	matched = append([]Def(nil), matched...)
	sort.SliceStable(matched, func(i, j int) bool { return matched[i].Priority > matched[j].Priority })
	top := matched[0]
	var ambiguous []string
	for _, d := range matched[1:] {
		if d.Priority == top.Priority {
			ambiguous = append(ambiguous, d.Name)
		} else {
			w.reject("%s: khớp đủ nhưng bị loại — ưu tiên %d thấp hơn %s (%d)", d.Name, d.Priority, top.Name, top.Priority)
		}
	}
	if len(ambiguous) > 0 {
		w.reason("hoà: %s cùng ưu tiên %d — số cụm từ khoá khớp không phải tín hiệu nghĩa nên không phân xử thay người hỏi, hỏi lại", strings.Join(append([]string{top.Name}, ambiguous...), ", "), top.Priority)
		return Result{Ambiguous: append([]string{top.Name}, ambiguous...)}
	}
	if len(matched) == 1 {
		w.reason("chọn %s: ý định duy nhất khớp đủ", top.Name)
	} else {
		w.reason("chọn %s: ưu tiên cao nhất (%d)", top.Name, top.Priority)
	}
	return s.fill(top, in, res, now, implicit, w)
}

// comparison xử lý câu SO SÁNH hai chỉ số (ADR-0010, cụm C4) theo CẤU TRÚC: trong MỘT mệnh đề có từ so sánh bậc hơn / kém của pack ("hơn", "kém", "so với") hay từ khoá của
// ý định dẫn xuất ("chênh", "lệch"), nếu ý định số liệu nêu ở mệnh đề đó cho đúng hai chỉ số KHÁC nhau thì là phép so sánh đúng cặp người hỏi nêu (vế nhắc trước − vế nhắc
// sau). Số cụm từ khoá khớp và độ ưu tiên của ý định dẫn xuất không phải tín hiệu nghĩa nên không quyết định thay họ. Ý định dẫn xuất chỉ trả cặp mặc định khi mọi chỉ số câu nêu
// đều nằm trong cặp ấy; nêu chỉ số ngoài cặp mà không phải đúng hai chỉ số, hay ≥ 3 chỉ số, thì không có cặp nào để trả thay người hỏi: hỏi lại. ok = false: không phải câu
// so sánh hai chỉ số, build chọn như thường.
func (s *Set) comparison(matched []Def, text string, in contracts.NormalizedInput, res contracts.Resolution, now time.Time, implicit string, w *Why) (Result, bool) {
	cs, ce, ok := s.comparisonScope(text)
	if !ok {
		return Result{}, false
	}
	counts, firsts, _ := keywordHits(s.hitDefs(), text[cs:ce], s.ignored())
	var named, derived []Def
	metrics, first := map[string]string{}, map[string]int{}
	for i, d := range s.defs {
		if counts[i] == 0 || !slices.ContainsFunc(matched, func(m Def) bool { return m.Name == d.Name }) {
			continue
		}
		first[d.Name] = firsts[i]
		switch m, _ := metricOf(d, in, implicit); {
		case d.Derived != "":
			derived = append(derived, d)
		case !d.Weak && contracts.RouteKind(d.Route) == contracts.RouteSQLQuery && m != "":
			named, metrics[d.Name] = append(named, d), m
		}
	}
	if len(named) == 0 {
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
		w.reason("câu so sánh: %s so với %s (hai chỉ số khác nhau trong cùng mệnh đề kèm dấu hiệu so sánh) — phép tính hiệu theo thứ tự nhắc, không phải câu mơ hồ", left.Name, right.Name)
		for _, d := range matched {
			if d.Name != left.Name && d.Name != right.Name {
				w.reject("%s: khớp đủ nhưng không phải vế của phép so sánh %s so với %s", d.Name, left.Name, right.Name)
			}
		}
		return r, true
	}
	if len(derived) == 0 && len(distinct) < 3 {
		return Result{}, false // một chỉ số (hay hai vế cùng chỉ số) + từ so sánh với thứ khác: không phải so sánh hai chỉ số
	}
	var names []string
	for _, d := range slices.Concat(derived, named) {
		names = append(names, d.Name)
	}
	w.reason("câu có dấu hiệu so sánh nhưng các chỉ số nêu (%s) không thành đúng một cặp hai chỉ số khác nhau — không chọn cặp thay người hỏi, hỏi lại", strings.Join(names, ", "))
	return Result{Ambiguous: names}, true
}

// comparisonScope trả phạm vi của phép so sánh trong câu (text đã che mệnh đề ngữ cảnh): cả câu khi có từ khoá của ý định dẫn xuất ("chênh lệch dầu cấp, đội báo và hoá đơn":
// danh sách chỉ số cách nhau bằng dấu phẩy cùng thuộc "chênh lệch"); không thì MỆNH ĐỀ đầu tiên có cụm so sánh bậc hơn / kém của pack ("đội báo cao hơn kho xuất bao nhiêu": hai chỉ số
// và từ so sánh phải ở cùng mệnh đề, từ so sánh của mệnh đề khác không biến hai chỉ số ở đây thành phép so sánh).
func (s *Set) comparisonScope(text string) (start, end int, ok bool) {
	counts, _, _ := keywordHits(s.hitDefs(), text, s.ignored())
	for i, d := range s.defs {
		if d.Derived != "" && counts[i] > 0 {
			return 0, len(text), true
		}
	}
	for _, c := range clauseSpans(text) {
		if s.comparing(text[c[0]:c[1]]) {
			return c[0], c[1], true
		}
	}
	return 0, 0, false
}

// ignored là các cụm KHÔNG phải từ khoá: cụm bỏ qua của pack và cụm báo nghĩa chưa hỗ trợ (chiếm chỗ: "báo" của "dự báo" không phải từ khoá).
func (s *Set) ignored() []string {
	return append(slices.Clone(s.opt.IgnorePhrases), s.cuePhrases()...)
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
	clashSlot, clashMentioned := "", 0
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
			if clash, mentioned := clashing(res, sd.Types); len(clash) > 0 {
				// Câu nêu ≥ 2 thực thể KHÁC NHAU cùng loại cho khe chỉ nhận một: không lấy thực thể nhắc đầu (P5).
				it.Missing = append(it.Missing, sd.Name)
				unclear.Entities = append(unclear.Entities, clash...)
				clashSlot, clashMentioned = sd.Name, len(mentioned)
				w.reason("khe %s thiếu: câu nêu %d thực thể khác nhau cùng loại (%s) cho khe chỉ nhận một — không lấy thực thể nhắc đầu", sd.Name, len(mentioned), contracts.Brief(refKeys(mentioned), 6, "thực thể"))
				if dropped := slices.DeleteFunc(slices.Clone(mentioned), func(e contracts.EntityRef) bool {
					return slices.ContainsFunc(clash, func(c contracts.EntityRef) bool { return c.ID == e.ID })
				}); len(dropped) > 0 {
					w.reason("hỏi lại chỉ liệt kê ứng viên của bí danh mơ hồ (%s): thực thể rõ %s chỉ góp vào việc đếm chủ thể, không phải thứ người hỏi cần chọn (D-OP-11 = b)", contracts.Brief(refKeys(clash), 6, "thực thể"), contracts.Brief(refKeys(dropped), 6, "thực thể"))
				}
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
	w.Origins = originsOf(res, candidates, unclear.Entities, clashSlot, clashMentioned)
	w.reason("luật %s", it.Rule)
	return Result{Intent: it, Candidates: candidates, Unclear: unclear}
}

// backgroundRanges trả các khoảng byte của đoạn MỤC ĐÍCH (S4BS-01): từ từ nối mục đích của pack ("để", "nhằm") tới hết mệnh đề chứa nó. Đoạn nói vì sao người hỏi cần câu trả lời nên không góp từ
// khoá chỉ số ("… để em báo cáo gấp" không biến câu về kho xuất thành câu về đội báo). Đoạn mang cụm hỏi (cụm xác nhận / hỏi lượng) thì chính nó là câu hỏi: không phải nền.
func (s *Set) backgroundRanges(raw string) (out [][2]int) {
	if len(s.opt.PurposeMarkers) == 0 {
		return nil
	}
	for _, c := range clauseSpans(raw) {
		ct := raw[c[0]:c[1]]
		at := -1
		for _, m := range s.opt.PurposeMarkers {
			if sp := normalization.PhraseSpans(ct, m); len(sp) > 0 && (at < 0 || sp[0][0] < at) {
				at = sp[0][0]
			}
		}
		if at >= 0 && !s.asking(ct[at:]) {
			out = append(out, [2]int{c[0] + at, c[1]})
		}
	}
	return out
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
		return s.asking(raw[c[0]:c[1]])
	})
	if q < 0 {
		return nil
	}
	if q > 0 && s.onlyMarkers(raw[cl[q][0]:cl[q][1]], markers) {
		q--
	}
	var out [][2]int
	for i := range cl {
		// trước mệnh đề hỏi: nguồn của con số ("phiếu kho ghi 3.850, vậy …"); sau mệnh đề hỏi: phần chú thêm nêu con số ("…, biết đội báo 4000 L?") — cả hai chỉ là ngữ cảnh
		if i != q && hasFigure(in, res, cl[i][0], cl[i][1]) {
			out = append(out, cl[i])
		}
	}
	// Chỉ che khi mệnh đề hỏi CÓ chỉ số riêng. Mệnh đề hỏi chỉ là cái đuôi ("…, tại bể 7 đúng không?", "…, hay 4000 L đúng không?") thì con số và
	// chữ chỉ số nằm ở mệnh đề đứng trước chính là câu hỏi, không phải ngữ cảnh.
	if len(out) == 0 || !s.ownMetric(maskRanges(raw, s.backgroundRanges(raw))[cl[q][0]:cl[q][1]], hasFigure(in, res, cl[q][0], cl[q][1])) {
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
// nameRanges là các cụm TÊN thực thể của câu (đã giải hay mơ hồ): chữ trong tên không phải chữ hỏi — câu gõ không dấu "ghi nhan may xuc 03
// …" có "may" (= máy) khớp chữ hỏi lượng "mấy" theo luật dấu (đợt gộp 24/10, soát độc lập).
func nameRanges(res contracts.Resolution) [][2]int {
	var out [][2]int
	for _, e := range res.Entities {
		out = append(out, [2]int{e.Start, e.End})
	}
	for _, m := range res.Ambiguous {
		out = append(out, [2]int{m.Start, m.End})
	}
	return out
}

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
// (implicit) — một lượng cùng chiều đo với chỉ số ("3850 L", "1,2 khối") cùng với cụm xác nhận ("đúng không") hay một chữ của từ khoá chỉ số ("xuất", "cấp").
// Cụm xác nhận + thực thể + lượng cùng chiều đo là đủ (S4AS-03, C3): không đòi mảnh từ khoá, nên động từ lạ ("xài", "bơm") không biến câu thành ngoài phạm vi. Một trong những thứ đó thì
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
	return implicit != "" && s.volumeFigure(in) && (s.confirming(in.Raw) || s.metricWord(in.Raw))
}

// confirming báo câu có cụm xác nhận của pack ("đúng không"): người hỏi đang kiểm một con số, nên câu có thực thể và lượng cùng chiều đo là câu về số liệu dù động từ lạ.
func (s *Set) confirming(text string) bool {
	return slices.ContainsFunc(s.opt.FocusMarkers, func(m string) bool { return normalization.ContainsPhrase(text, m) })
}

// volumeFigure báo câu có một lượng đổi được sang đơn vị của chỉ số nào đó của các ý định số liệu (cùng chiều đo: lít, khối).
func (s *Set) volumeFigure(in contracts.NormalizedInput) bool {
	return s.volumeFigureIn(in, 0, len(in.Raw))
}

// volumeFigureIn là volumeFigure giới hạn ở khoảng byte [start, end) của câu.
func (s *Set) volumeFigureIn(in contracts.NormalizedInput, start, end int) bool {
	if s.opt.QuantityFits == nil {
		return false
	}
	return slices.ContainsFunc(in.SpansOf(contracts.ValueQuantity), func(sp contracts.Span) bool {
		if sp.Start < start || sp.End > end {
			return false
		}
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

// UnsupportedCue báo câu có cụm báo nghĩa chưa hỗ trợ của pack (cảnh báo, dự báo, báo giá, lệnh ghi…) — kể cả khi câu không nêu thực thể nên
// S4 không trả Unsupported: dự phòng S4 không được trả lời những câu này bằng tài liệu.
func (s *Set) UnsupportedCue(text string) bool {
	return slices.ContainsFunc(s.cuePhrases(), func(c string) bool { return normalization.ContainsPhrase(text, c) })
}

// cuePhrases trả mọi cụm báo nghĩa chưa hỗ trợ của pack.
func (s *Set) cuePhrases() []string {
	var out []string
	for _, u := range s.opt.Unsupported {
		out = append(out, u.Cues...)
		out = append(out, u.WeakCues...)
	}
	return out
}

// unsupportedIn trả các nghĩa chưa hỗ trợ mà câu mang (D-OP-18 = c): câu có thực thể, và MỘT mệnh đề nào đó có cụm báo nghĩa đó — kể cả mệnh đề ngữ cảnh đứng trước mệnh
// đề hỏi (S4AS-04): im lặng về lệnh ghi / đánh giá / cảnh báo rồi trả đúng phần hỏi là để người dùng tưởng đã được ghi. Nghĩa NeedsValue (lệnh ghi, đánh giá) còn đòi chính mệnh đề
// đó nêu một con số và không phải câu hỏi (cụm xác nhận / cụm hỏi lượng); ở mệnh đề ngữ cảnh, cụm phải đứng ĐẦU mệnh đề ("Ghi nhận …, vậy …"): "phiếu kho ghi nhận 3850 L, vậy …"
// là nguồn + động từ của con số, không phải lệnh. Tín hiệu cấu trúc (Signals) chỉ xét ở phần quyết định (từ mệnh đề hỏi trở đi). Câu có từ khoá tri thức ("quy trình cảnh báo …")
// là câu hỏi quy trình, tài liệu trả lời được: không tính.
func (s *Set) unsupportedIn(in contracts.NormalizedInput, rawBG string, res contracts.Resolution, prem [][2]int, counts []int, w *Why) []Unsupported {
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
	// Câu nêu một lượng cùng chiều đo với chỉ số ("3850 L") thì con số tiền kèm theo là phụ, câu vẫn là câu về lượng (cổng phán quyết đã không phán quyết khi
	// có nhiều con số): tín hiệu cấu trúc của nghĩa tiền chỉ tính khi câu không có lượng như thế; cụm báo nghĩa ("tiền dầu") vẫn tính.
	measured := s.volumeFigureIn(in, qStart, len(in.Raw))
	clauses := clauseSpans(in.Raw)
	var out []Unsupported
	markers := slices.Concat(s.opt.FocusMarkers, s.opt.QuantityMarkers)
	// câu hỏi về một chỉ số: có từ khoá của ý định số liệu (không tính quy trình / chào hỏi) hay thuật ngữ vật tư mà một ý định đòi
	metricAsk := slices.ContainsFunc(in.SpansOf(contracts.ValueTerm), func(sp contracts.Span) bool {
		return slices.ContainsFunc(s.defs, func(d Def) bool { return slices.Contains(d.Terms, sp.Value.Text) })
	})
	for i, d := range s.defs {
		if r := contracts.RouteKind(d.Route); counts[i] > 0 && (r == contracts.RouteSQLQuery || r == contracts.RouteCalculation) {
			metricAsk = true
		}
	}
	for _, u := range s.opt.Unsupported {
		why := ""
		for ci, c := range clauses {
			ct := rawBG[c[0]:c[1]]
			at := firstSpan(ct, u.Cues) // vị trí của cụm báo nghĩa đầu tiên trong mệnh đề
			if at < 0 && metricAsk && s.asking(ct) {
				at = firstSpan(ct, u.WeakCues) // cụm quá chung chỉ kích hoạt ở chính mệnh đề hỏi một chỉ số
			}
			if at < 0 {
				continue
			}
			if slices.ContainsFunc(s.opt.DismissMarkers, func(m string) bool { return normalization.ContainsPhrase(in.Raw[c[0]:c[1]], m) }) || s.negated(ct) || s.conditional(ct) { // câu gốc: "để sau" vừa là cụm bỏ qua vừa mở một đoạn "mục đích"
				w.reason("cụm báo nghĩa %s ở mệnh đề %s bị bỏ qua: mệnh đề nói không cần / để sau, phủ định hay nêu điều kiện (S4BS-05)", u.Name, contracts.Quote(ct))
				continue
			}
			if u.NeedsValue {
				// mệnh đề là câu hỏi khi tự có cụm hỏi, hoặc mệnh đề liền sau chỉ là cái đuôi hỏi ("…, đúng không?"), như premiseRanges
				askedAt := func(j int) bool {
					if s.asking(rawBG[clauses[j][0]:clauses[j][1]]) {
						return true
					}
					next := ""
					if j+1 < len(clauses) {
						next = rawBG[clauses[j+1][0]:clauses[j+1][1]]
					}
					return strings.TrimSpace(next) != "" && s.onlyMarkers(next, markers)
				}
				if askedAt(ci) {
					continue // cụm là chữ MÔ TẢ trong câu hỏi ("ghi nhận tiêu thụ bao nhiêu dầu?")
				}
				// chỗ của con số: chính mệnh đề, hay một mệnh đề KỀ không phải câu hỏi ("Ghi nhận cho EX-03, dùng 3782 L dầu"): cụm và con số ở hai mệnh đề liền nhau vẫn là một lệnh (S4BS-03 i)
				figure := hasFigure(in, res, c[0], c[1])
				for _, nj := range []int{ci + 1, ci - 1} {
					figure = figure || nj >= 0 && nj < len(clauses) && hasFigure(in, res, clauses[nj][0], clauses[nj][1]) && !askedAt(nj)
				}
				if figure {
					if slices.Contains(prem, c) && len(strings.Fields(ct[:at])) > 1 {
						continue // mệnh đề ngữ cảnh mà cụm không đứng đầu: nguồn + động từ của con số, không phải lệnh
					}
				} else if !s.attachedRequest(clauses, ci, rawBG) {
					continue
				}
			}
			why = fmt.Sprintf("cụm báo nghĩa ở mệnh đề %s", contracts.Quote(ct))
			break
		}
		// span:* (bộ chuẩn hoá thấy một số tiền) nhường câu có lượng cùng chiều đo; asked_unit:* là CHÍNH câu hỏi tiền ("bao nhiêu đồng một lít", "… 3782 L dầu, hết bao nhiêu đồng?") nên không nhường
		if why == "" && slices.ContainsFunc(u.Signals, func(sig string) bool {
			kind, _, _ := strings.Cut(sig, ":")
			return sig != SignalStatedValue && (kind != "span" || !measured) && s.signalHit(sig, in, qStart, len(in.Raw))
		}) {
			why = "tín hiệu cấu trúc ở phần quyết định của câu"
		}
		if why != "" {
			w.reason("nghĩa chưa hỗ trợ %s: %s", u.Name, why)
			out = append(out, u)
		}
	}
	// D-OP-18 = c đúng chữ: chưa nghĩa nào được báo mà câu NÊU một giá trị (thực thể + con số có đơn vị) và không có dấu hiệu hỏi / xác nhận thì là lệnh ghi, bất kể động từ — không dựa danh sách động từ.
	if len(out) == 0 && s.statedValue(in, res) {
		for _, u := range s.opt.Unsupported {
			if slices.Contains(u.Signals, SignalStatedValue) {
				w.reason("nghĩa chưa hỗ trợ %s: câu nêu một giá trị (thực thể + con số có đơn vị) mà không có dấu hiệu hỏi hay xác nhận (cụm hỏi, \"?\", đuôi \"… không\") — là lệnh ghi bất kể động từ (D-OP-18 = c)", u.Name)
				out = append(out, u)
				break
			}
		}
	}
	return out
}

// statedValue báo câu NÊU một giá trị: có một lượng kèm đơn vị cùng chiều đo với một chỉ số của pack (lít, khối — "giá trị cụ thể" của chỉ số; không phải số trần, không phải con số của đại lượng
// khác như giờ họp "7h30", "20 tấn", và không nằm trong tên thực thể: "03" của "máy 03" đứng cạnh "dầu" bị bộ chuẩn hoá gắn đơn vị lít) và không có dấu hiệu hỏi nào (xem questioned). Thực thể do
// bên gọi kiểm. Số tiền do tín hiệu tiền xét trước.
func (s *Set) statedValue(in contracts.NormalizedInput, res contracts.Resolution) bool {
	inName := func(sp contracts.Span) bool {
		return slices.ContainsFunc(res.Entities, func(e contracts.EntityRef) bool { return e.Start <= sp.Start && sp.End <= e.End }) ||
			slices.ContainsFunc(res.Ambiguous, func(m contracts.AmbiguousMention) bool { return m.Start <= sp.Start && sp.End <= m.End })
	}
	figure := slices.ContainsFunc(in.SpansOf(contracts.ValueQuantity), func(sp contracts.Span) bool { return !inName(sp) && s.fitsAnyMetric(sp.Value) })
	return figure && !s.questioned(maskRanges(in.Raw, nameRanges(res)))
}

// fitsAnyMetric báo một lượng đổi được sang đơn vị của chỉ số nào đó của các ý định số liệu (cùng chiều đo).
func (s *Set) fitsAnyMetric(v contracts.Value) bool {
	return s.opt.QuantityFits != nil && slices.ContainsFunc(s.defs, func(d Def) bool {
		for _, m := range d.MetricByTerm {
			if s.opt.QuantityFits(m, v) {
				return true
			}
		}
		return false
	})
}

// questioned báo câu có DẤU HIỆU HỎI: cụm hỏi của pack (cụm xác nhận, cụm hỏi lượng), dấu "?" (bất biến (b) của ADR-0010), hay từ phủ định đứng cuối câu ("… dùng quá 4000 L dầu không": đuôi hỏi có / không).
// Khác asking: asking không coi "?" một mình là câu hỏi (lệnh ghi lịch sự "Ghi nhận … nhé?" vẫn là lệnh khi động từ nằm trong danh sách cụm); questioned dùng cho câu nêu giá trị KHÔNG động từ nào của danh sách,
// nơi chỉ còn cấu trúc để phân biệt lệnh với câu xác nhận.
func (s *Set) questioned(raw string) bool {
	if s.asking(raw) || strings.Contains(raw, "?") {
		return true
	}
	end := s.contentEnd(raw)
	return slices.ContainsFunc(s.opt.NegationMarkers, func(m string) bool {
		sp := normalization.PhraseSpans(raw, m)
		return len(sp) > 0 && sp[len(sp)-1][1] == end
	})
}

// attachedRequest báo mệnh đề ci là yêu cầu KÈM THÊM sau câu hỏi ("EX-03 hôm nay đội báo bao nhiêu dầu, kèm nhận xét luôn?"): đứng sau một mệnh đề là câu hỏi, không là câu hỏi, và không nêu tên nguồn
// ("…, theo ghi nhận của cảm biến?" là nguồn + động từ). Yêu cầu ấy hệ chưa làm được: im lặng bỏ nó rồi trả số là để người dùng tưởng đã được nhận xét / ghi (S4BS-03 iii). Cụm ở mệnh đề đứng TRƯỚC câu
// hỏi mà không có con số ("Theo ghi nhận …, …?") là dẫn đề nên không tính.
func (s *Set) attachedRequest(clauses [][2]int, ci int, rawBG string) bool {
	ct := rawBG[clauses[ci][0]:clauses[ci][1]]
	if s.hasSourceName(ct) {
		return false
	}
	for j := 0; j < ci; j++ {
		if s.asking(rawBG[clauses[j][0]:clauses[j][1]]) {
			return true
		}
	}
	return false
}

// SignalStatedValue là tín hiệu cấu trúc "câu NÊU một giá trị": có thực thể và một con số CÓ ĐƠN VỊ mà không có dấu hiệu hỏi / xác nhận nào (D-OP-18 = c đúng chữ) — bất kể động từ.
const SignalStatedValue = "stated_value"

// firstSpan trả vị trí byte đầu tiên mà một trong các cụm khớp trong text, -1 nếu không có.
func firstSpan(text string, phrases []string) int {
	at := -1
	for _, sp := range spansOf(text, phrases) {
		if at < 0 || sp[0] < at {
			at = sp[0]
		}
	}
	return at
}

// contentEnd trả vị trí byte hết phần NỘI DUNG của text: bỏ dấu câu và các từ trung tính của pack ("ạ", "nhé", "vậy") ở cuối. Từ phủ định đứng ngay trước chúng ("… được không ạ?") vẫn là đuôi hỏi có / không.
func (s *Set) contentEnd(text string) int {
	notWord := func(r rune) bool { return !unicode.IsLetter(r) && !unicode.IsDigit(r) }
	end := len(strings.TrimRightFunc(text, notWord))
	for end > 0 {
		j := end
		for j > 0 {
			r, n := utf8.DecodeLastRuneInString(text[:j])
			if notWord(r) && !unicode.Is(unicode.Mn, r) {
				break
			}
			j -= n
		}
		if w := text[j:end]; w == "" || !s.isNeutral(w) {
			break
		}
		end = len(strings.TrimRightFunc(text[:j], notWord))
	}
	return end
}

// negated báo mệnh đề có từ phủ định KHÔNG đứng cuối ("Không tính tiền", "Chưa tính chi phí", "Trị giá hợp đồng không đổi"): cụm báo nghĩa trong đó đang bị phủ nhận, không phải điều được hỏi. Từ phủ định cuối
// mệnh đề là đuôi hỏi có / không ("… có cao không?") và từ phủ định nằm trong cụm hỏi của pack ("đúng không") không tính — câu hỏi tiền kèm "không?" vẫn là câu hỏi tiền (S4BS-05).
func (s *Set) negated(clause string) bool {
	if len(s.opt.NegationMarkers) == 0 {
		return false
	}
	masked := maskRanges(clause, spansOf(clause, slices.Concat(s.opt.FocusMarkers, s.opt.QuantityMarkers)))
	end := s.contentEnd(masked)
	return slices.ContainsFunc(spansOf(masked, s.opt.NegationMarkers), func(sp [2]int) bool { return sp[1] < end })
}

// conditional báo mệnh đề BẮT ĐẦU bằng từ nối điều kiện của pack ("Nếu dự báo mưa, …").
func (s *Set) conditional(clause string) bool {
	trimmed := len(clause) - len(strings.TrimLeft(clause, " "))
	return slices.ContainsFunc(spansOf(clause, s.opt.ConditionMarkers), func(sp [2]int) bool { return sp[0] == trimmed })
}

// ValidSignal kiểm một tín hiệu cấu trúc do pack khai ở unsupported_meanings: "stated_value" (câu nêu giá trị, không dấu hiệu hỏi), "span:<money|quantity|number>" (bộ chuẩn hoá thấy một span kiểu đó), "asked_unit:<chiều
// đo>" ("bao nhiêu" + đơn vị của chiều đo ở cuối mệnh đề: "bao nhiêu đồng"). dimensionKnown báo chiều đo có trong lexicon.
func ValidSignal(sig string, dimensionKnown func(string) bool) error {
	kind, arg, _ := strings.Cut(sig, ":")
	switch {
	case sig == SignalStatedValue:
		return nil
	case kind == "span" && (arg == string(contracts.ValueMoney) || arg == string(contracts.ValueQuantity) || arg == string(contracts.ValueNumber)):
		return nil
	case kind == "asked_unit" && arg != "" && dimensionKnown(arg):
		return nil
	}
	return fmt.Errorf("tín hiệu %q không hợp lệ (nhận span:<money|quantity|number>, asked_unit:<chiều đo của lexicon>)", sig)
}

// signalHit báo phần quyết định [start, end) của câu mang một tín hiệu cấu trúc (xem ValidSignal). Tín hiệu chỉ xét ở phần quyết định (từ mệnh đề hỏi trở đi): một
// con số tiền trong mệnh đề ngữ cảnh ("hoá đơn ghi 4tr đồng, vậy … bao nhiêu lít") là nguồn của con số, không phải điều đang hỏi.
func (s *Set) signalHit(sig string, in contracts.NormalizedInput, start, end int) bool {
	kind, arg, _ := strings.Cut(sig, ":")
	inRange := func(sp contracts.Span) bool { return sp.Start >= start && sp.End <= end }
	switch kind {
	case "span":
		return slices.ContainsFunc(in.Spans, func(sp contracts.Span) bool { return inRange(sp) && string(sp.Kind) == arg })
	case "asked_unit":
		return s.askedUnit(in.Raw[start:end], arg)
	}
	return false
}

// askedUnit báo câu hỏi lượng đi thẳng vào đơn vị của một chiều đo: cụm hỏi lượng ("bao nhiêu") rồi — tuỳ chọn — các từ chỉ hàng ("trăm", "nghìn", "triệu", "tỷ") và một đơn vị của chiều đo ("đồng", "đô la",
// "USD", "củ"), Ở BẤT KỲ ĐÂU trong mệnh đề (S4BS-04): "bao nhiêu đồng cho dầu của EX-03", "bao nhiêu đồng một lít". Phải có ít nhất một từ chỉ hàng hay đơn vị, và chữ liền sau phải là hết mệnh đề, từ
// trung tính, số chữ hay cụm hỏi (moneyPhraseEnds): chữ đầu của từ ghép ("đồng hồ", "đồng bộ", "đô thị", "tỷ lệ") và đơn vị thể tích viết rõ ("bao nhiêu triệu lít") không phải tiền.
func (s *Set) askedUnit(text, dimension string) bool {
	if s.opt.UnitAliases == nil {
		return false
	}
	units := s.opt.UnitAliases(dimension)
	slices.SortStableFunc(units, func(a, b string) int { return len(b) - len(a) }) // đơn vị dài trước ("đô la" trước "đô")
	for _, q := range s.opt.QuantityMarkers {
		for _, sp := range normalization.PhraseSpans(text, q) {
			rest, eaten := text[sp[1]:], false
			for {
				trimmed := strings.TrimLeft(rest, " ")
				if n := leadingAlias(trimmed, units); n > 0 { // đơn vị của chiều đo, rồi hết cụm
					rest, eaten = trimmed[n:], true
					break
				}
				w := leadingWord(trimmed)
				if w == "" || !normalization.HasNumberWord(w) { // từ chỉ hàng ("trăm", "nghìn", "triệu", "tỷ"): ăn rồi xét tiếp
					break
				}
				rest, eaten = trimmed[len(w):], true
			}
			if eaten && s.moneyPhraseEnds(rest) {
				return true
			}
		}
	}
	return false
}

// leadingAlias trả độ dài (byte) của cách viết đơn vị khớp ngay đầu text (ranh giới từ, cùng luật dấu), 0 nếu không có.
func leadingAlias(text string, aliases []string) int {
	for _, a := range aliases {
		if sp := normalization.PhraseSpans(text, a); len(sp) > 0 && sp[0][0] == 0 {
			return sp[0][1]
		}
	}
	return 0
}

// leadingWord trả chữ (dãy chữ cái / chữ số) đứng đầu text, rỗng nếu text không bắt đầu bằng chữ.
func leadingWord(text string) string {
	end := 0
	for end < len(text) {
		r, n := utf8.DecodeRuneInString(text[end:])
		if !unicode.IsLetter(r) && !unicode.IsDigit(r) && !unicode.Is(unicode.Mn, r) {
			break
		}
		end += n
	}
	return text[:end]
}

// moneyPhraseEnds báo cụm "bao nhiêu <từ chỉ hàng / đơn vị>" kết thúc ở đó: phần còn lại tới hết mệnh đề không có chữ nào, hoặc bắt đầu bằng từ trung tính của pack ("cho", "vậy", "nhé"), một số chữ
// ("một lít") hay cụm hỏi. Chữ nội dung khác ("hồ", "bộ", "thị", "lít") nghĩa là đơn vị vừa đọc chỉ là chữ đầu của một từ ghép hay một lượng khác.
func (s *Set) moneyPhraseEnds(rest string) bool {
	tail := rest
	for _, c := range clauseSpans(rest) {
		tail = rest[c[0]:c[1]]
		break
	}
	trimmed := strings.TrimLeft(tail, " ")
	w := leadingWord(trimmed)
	return w == "" || s.isNeutral(w) || normalization.HasNumberWord(w) || s.onlyMarkers(trimmed, slices.Concat(s.opt.QuantityMarkers, s.opt.FocusMarkers))
}

// asking báo phần câu là một CÂU HỎI: có cụm xác nhận hoặc cụm hỏi lượng của pack. Dấu '?' một mình KHÔNG làm câu thành câu hỏi (S4AS-01): "Ghi nhận
// EX-03 … 3782 L dầu nhé?" kết thúc bằng '?' nhưng là một LỆNH ghi lịch sự — câu hỏi là câu đòi xác nhận ("đúng không") hay đòi một lượng ("bao nhiêu").
func (s *Set) asking(text string) bool {
	text = s.withoutCues(text)
	return slices.ContainsFunc(slices.Concat(s.opt.FocusMarkers, s.opt.QuantityMarkers), func(m string) bool { return normalization.ContainsPhrase(text, m) })
}

// withoutCues thay mọi cụm báo nghĩa chưa hỗ trợ bằng dấu cách. Cụm hỏi chỉ khớp theo ranh giới từ của câu gốc: "định mức" không nằm trong "nhận định mức tiêu thụ" (cụm "nhận định" +
// "mức tiêu thụ") dù hai chữ cuối của "nhận định mức" trùng cụm hỏi lượng (S4BS-03 ii).
func (s *Set) withoutCues(text string) string {
	cues := s.cuePhrases()
	if len(cues) == 0 {
		return text
	}
	return maskRanges(text, spansOf(text, cues))
}

// spansOf trả mọi khoảng byte mà một trong các cụm khớp trong text (cùng luật dấu / ranh giới từ của PhraseSpans).
func spansOf(text string, phrases []string) (out [][2]int) {
	for _, p := range phrases {
		out = append(out, normalization.PhraseSpans(text, p)...)
	}
	return out
}

// hasSourceName báo mệnh đề nêu tên một nguồn cấp số liệu của pack ("cảm biến", "phiếu kho", "ERP").
func (s *Set) hasSourceName(text string) bool {
	return s.opt.SourceNames != nil && len(spansOf(text, s.opt.SourceNames(""))) > 0
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

// hitsBG đếm từ khoá như keywordHits, nhưng đoạn mục đích chỉ được che với ý định SỐ LIỆU (route SQL_QUERY / CALCULATION): đoạn đó không nói chỉ số nào ("… để em báo cáo gấp" không biến câu về
// kho xuất thành câu về đội báo), còn ý định khác (quy trình, chào hỏi) đếm trên câu chưa che — "để làm gì theo quy trình" chính là câu hỏi quy trình (masked: câu đã che đoạn mục đích; keep: chưa che).
func (s *Set) hitsBG(masked, keep string, ignore []string) (counts, first []int, phrases [][]string) {
	counts, first, phrases = keywordHits(s.hitDefs(), masked, ignore)
	if masked == keep {
		return counts, first, phrases
	}
	kc, kf, kp := keywordHits(s.hitDefs(), keep, ignore)
	for i, d := range s.defs {
		if r := contracts.RouteKind(d.Route); r != contracts.RouteSQLQuery && r != contracts.RouteCalculation {
			counts[i], first[i], phrases[i] = kc[i], kf[i], kp[i]
		}
	}
	return counts, first, phrases
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
func originsOf(res contracts.Resolution, cands, clash []contracts.EntityRef, clashSlot string, mentioned int) []Origin {
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
		why := fmt.Sprintf("khe %s chỉ nhận một thực thể mà câu nêu %d", clashSlot, mentioned)
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
// giải thì giữ cách liệt kê cũ (cả hai). Giá trị trả thứ hai (mentioned) là mọi thực thể câu nêu tranh khe, chưa lọc — để vết kể đúng số chủ thể.
func clashing(res contracts.Resolution, types []string) (list, mentioned []contracts.EntityRef) {
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
	mentioned = slices.Clone(out)
	if slices.ContainsFunc(out, func(e contracts.EntityRef) bool { return viable[e.ID] }) {
		out = slices.DeleteFunc(out, func(e contracts.EntityRef) bool { return !viable[e.ID] })
	}
	return out, mentioned
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
