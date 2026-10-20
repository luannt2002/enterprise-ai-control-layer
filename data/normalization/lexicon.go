package normalization

import (
	"fmt"
	"regexp"
	"sort"
	"strings"
)

// Lexicon là phần pack mô tả cách MỘT tenant viết số, đơn vị, thuật ngữ, viết tắt và mã.
//
// Đây chỉ là DỮ LIỆU (P12 — pack mô tả chính sách, mã cài hành vi): bảng tra, bí danh,
// hệ số quy đổi, mẫu mã. Mọi luật đọc nằm trong mã của package này. Cùng một chuỗi "4k"
// có nghĩa khác nhau giữa các ngành, nên nghĩa luôn đến từ lexicon của tenant, không
// bao giờ từ một luật toàn cục.
type Lexicon struct {
	Units         []UnitDef       `yaml:"units"`
	Multipliers   []MultiplierDef `yaml:"multipliers"`
	ApproxMarkers []string        `yaml:"approx_markers"`
	Terms         []TermDef       `yaml:"terms"`
	Abbreviations []AbbrDef       `yaml:"abbreviations"`
	Codes         []CodeDef       `yaml:"codes"`
}

// UnitDef định nghĩa một đơn vị chuẩn và các cách người dùng viết nó.
type UnitDef struct {
	Unit      string `yaml:"unit"`      // mã chuẩn: L, m3, km, h, VND, %
	Kind      string `yaml:"kind"`      // quantity | money
	Dimension string `yaml:"dimension"` // volume, length, duration, currency…
	// ToBase là hệ số quy về đơn vị cơ sở của chiều (chuỗi thập phân). Rỗng = 1.
	// Ví dụ chiều volume lấy L làm cơ sở: m3 có to_base "1000".
	ToBase        string   `yaml:"to_base"`
	Aliases       []string `yaml:"aliases"`
	CaseSensitive bool     `yaml:"case_sensitive"`
}

// MultiplierDef là hậu tố nhân: k = nghìn, tr = triệu…
type MultiplierDef struct {
	Factor  string   `yaml:"factor"`
	Aliases []string `yaml:"aliases"`
}

// TermDef là một thuật ngữ nghiệp vụ.
type TermDef struct {
	Term    string   `yaml:"term"`  // mã chuẩn: diesel_fuel, workforce…
	Label   string   `yaml:"label"` // cách hiển thị chuẩn: "dầu"
	Aliases []string `yaml:"aliases"`
	// DefaultUnit gắn đơn vị cho số trần đứng cạnh thuật ngữ: "dầu 4k" → 4000 L.
	// Đây là SUY LUẬN nên span sinh ra có độ tin cậy < 1.
	DefaultUnit string `yaml:"default_unit"`
}

// AbbrDef là một viết tắt có thể mang nhiều nghĩa.
type AbbrDef struct {
	Abbr     string        `yaml:"abbr"`
	Meanings []AbbrMeaning `yaml:"meanings"`
}

// AbbrMeaning là một nghĩa của viết tắt và điều kiện chọn nó — chỉ là so sánh một bước.
type AbbrMeaning struct {
	Term string `yaml:"term"`
	// WhenNextDimension: chọn nghĩa này khi ngay sau viết tắt là một lượng thuộc chiều
	// này ("DA 4.000L" → lượng thể tích → dầu).
	WhenNextDimension string `yaml:"when_next_dimension"`
	// Scopes: chọn nghĩa này khi văn bản thuộc một trong các phạm vi (báo cáo nhiên liệu…).
	Scopes []string `yaml:"scopes"`
	// Default: nghĩa dùng khi viết tắt KHÔNG đứng trước một con số. Đứng trước số trần mà
	// không phân định được thì không bao giờ đoán — sinh Ambiguity.
	Default bool `yaml:"default"`
}

// CodeDef là mẫu một loại mã định danh.
type CodeDef struct {
	Type    string `yaml:"type"`
	Pattern string `yaml:"pattern"` // regexp RE2, dùng nhóm có tên để dựng dạng chuẩn
	Format  string `yaml:"format"`  // ví dụ "{w}/{a}R{r}"; rỗng = giữ nguyên văn
	Upper   bool   `yaml:"upper"`
}

type unitAlias struct {
	alias string
	def   UnitDef
}

type multAlias struct {
	alias  string
	factor string
}

type termAlias struct {
	alias string
	def   TermDef
}

type compiledCode struct {
	def CodeDef
	re  *regexp.Regexp
}

// Engine là lexicon đã kiểm và biên dịch, sẵn sàng chuẩn hoá. An toàn cho nhiều goroutine.
type Engine struct {
	units    []unitAlias // xếp alias dài trước để "km" thắng "m"
	unitDefs map[string]UnitDef
	mults    []multAlias
	approx   []string
	terms    []termAlias
	termDefs map[string]TermDef
	abbrs    map[string]AbbrDef
	codes    []compiledCode
}

// Compile kiểm toàn bộ lexicon và trả lỗi gom đủ mọi chỗ sai — cấu hình hỏng thì không
// được chạy (luật khởi động của EAIS).
func Compile(lex Lexicon) (*Engine, error) {
	var errs []string
	fail := func(format string, a ...any) { errs = append(errs, fmt.Sprintf(format, a...)) }

	e := &Engine{
		unitDefs: map[string]UnitDef{},
		termDefs: map[string]TermDef{},
		abbrs:    map[string]AbbrDef{},
	}
	dims := map[string]bool{}
	seenAlias := map[string]string{}
	claim := func(alias, owner string) {
		k := strings.ToLower(alias)
		if prev, ok := seenAlias[k]; ok && prev != owner {
			fail("alias %q dùng cho cả %s và %s", alias, prev, owner)
		}
		seenAlias[k] = owner
	}

	for _, u := range lex.Units {
		if u.Unit == "" || u.Dimension == "" {
			fail("đơn vị thiếu unit/dimension: %+v", u)
			continue
		}
		if u.Kind != "quantity" && u.Kind != "money" {
			fail("đơn vị %s: kind phải là quantity hoặc money, đang là %q", u.Unit, u.Kind)
		}
		if _, dup := e.unitDefs[u.Unit]; dup {
			fail("đơn vị %s khai báo hai lần", u.Unit)
		}
		if u.ToBase == "" {
			u.ToBase = "1"
		}
		if r, ok := parseDecimal(u.ToBase); !ok || r.Sign() <= 0 {
			fail("đơn vị %s: to_base %q không phải số dương", u.Unit, u.ToBase)
		}
		if len(u.Aliases) == 0 {
			fail("đơn vị %s không có alias nào", u.Unit)
		}
		e.unitDefs[u.Unit] = u
		dims[u.Dimension] = true
		for _, a := range u.Aliases {
			claim(a, "đơn vị "+u.Unit)
			e.units = append(e.units, unitAlias{alias: NFC(a), def: u})
		}
	}

	for _, m := range lex.Multipliers {
		if r, ok := parseDecimal(m.Factor); !ok || r.Sign() <= 0 {
			fail("hậu tố nhân %v: factor %q không phải số dương", m.Aliases, m.Factor)
			continue
		}
		for _, a := range m.Aliases {
			claim(a, "hậu tố nhân "+m.Factor)
			e.mults = append(e.mults, multAlias{alias: NFC(a), factor: m.Factor})
		}
	}

	for _, a := range lex.ApproxMarkers {
		e.approx = append(e.approx, NFC(a))
	}

	for _, t := range lex.Terms {
		if t.Term == "" || len(t.Aliases) == 0 {
			fail("thuật ngữ thiếu term/aliases: %+v", t)
			continue
		}
		if _, dup := e.termDefs[t.Term]; dup {
			fail("thuật ngữ %s khai báo hai lần", t.Term)
		}
		if t.DefaultUnit != "" {
			if _, ok := e.unitDefs[t.DefaultUnit]; !ok {
				fail("thuật ngữ %s: default_unit %q chưa khai báo trong units", t.Term, t.DefaultUnit)
			}
		}
		e.termDefs[t.Term] = t
		for _, a := range t.Aliases {
			claim(a, "thuật ngữ "+t.Term)
			e.terms = append(e.terms, termAlias{alias: NFC(a), def: t})
		}
	}

	for _, ab := range lex.Abbreviations {
		if ab.Abbr == "" || len(ab.Meanings) == 0 {
			fail("viết tắt thiếu abbr/meanings: %+v", ab)
			continue
		}
		if _, dup := e.abbrs[ab.Abbr]; dup {
			fail("viết tắt %s khai báo hai lần", ab.Abbr)
		}
		defaults := 0
		for _, m := range ab.Meanings {
			if _, ok := e.termDefs[m.Term]; !ok {
				fail("viết tắt %s: nghĩa %q chưa khai báo trong terms", ab.Abbr, m.Term)
			}
			if m.WhenNextDimension != "" && !dims[m.WhenNextDimension] {
				fail("viết tắt %s: chiều %q không có đơn vị nào", ab.Abbr, m.WhenNextDimension)
			}
			if m.Default {
				defaults++
			}
		}
		if defaults > 1 {
			fail("viết tắt %s có %d nghĩa mặc định, tối đa 1", ab.Abbr, defaults)
		}
		e.abbrs[ab.Abbr] = ab
	}

	for _, c := range lex.Codes {
		if c.Type == "" || c.Pattern == "" {
			fail("mã thiếu type/pattern: %+v", c)
			continue
		}
		re, err := regexp.Compile(`^(?:` + c.Pattern + `)`)
		if err != nil {
			fail("mã %s: pattern không biên dịch được: %v", c.Type, err)
			continue
		}
		for _, name := range placeholders(c.Format) {
			if re.SubexpIndex(name) < 0 {
				fail("mã %s: format dùng {%s} nhưng pattern không có nhóm tên đó", c.Type, name)
			}
		}
		e.codes = append(e.codes, compiledCode{def: c, re: re})
	}

	if len(errs) > 0 {
		return nil, fmt.Errorf("lexicon không hợp lệ:\n  - %s", strings.Join(errs, "\n  - "))
	}

	byLen := func(a, b string) bool { return len([]rune(a)) > len([]rune(b)) }
	sort.SliceStable(e.units, func(i, j int) bool { return byLen(e.units[i].alias, e.units[j].alias) })
	sort.SliceStable(e.mults, func(i, j int) bool { return byLen(e.mults[i].alias, e.mults[j].alias) })
	sort.SliceStable(e.terms, func(i, j int) bool { return byLen(e.terms[i].alias, e.terms[j].alias) })
	sort.SliceStable(e.approx, func(i, j int) bool { return byLen(e.approx[i], e.approx[j]) })
	return e, nil
}

var placeholderRe = regexp.MustCompile(`\{([A-Za-z_][A-Za-z0-9_]*)\}`)

func placeholders(format string) []string {
	var out []string
	for _, m := range placeholderRe.FindAllStringSubmatch(format, -1) {
		out = append(out, m[1])
	}
	return out
}

// Unit trả định nghĩa một đơn vị chuẩn.
func (e *Engine) Unit(code string) (UnitDef, bool) {
	u, ok := e.unitDefs[code]
	return u, ok
}

// UnitsOf trả mọi đơn vị thuộc một chiều đo, xếp theo mã (thứ tự ổn định).
func (e *Engine) UnitsOf(dimension string) []UnitDef {
	var out []UnitDef
	for _, u := range e.unitDefs {
		if u.Dimension == dimension {
			out = append(out, u)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Unit < out[j].Unit })
	return out
}

// Term trả định nghĩa một thuật ngữ.
func (e *Engine) Term(code string) (TermDef, bool) {
	t, ok := e.termDefs[code]
	return t, ok
}
