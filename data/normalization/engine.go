// Package normalization là chặng S2: biến cách người dùng VIẾT thành giá trị có kiểu.
//
// "4k", "4.000L", "bốn ngàn lít", "4 khối" là các cách viết; nghĩa nghiệp vụ là một
// lượng có đơn vị. Chặng này tất định, có luật truy được cho từng span, và KHÔNG đoán:
// chỗ nào không tự phân định được thì ghi Ambiguity để các chặng sau hỏi lại.
package normalization

import (
	"math/big"
	"slices"
	"sort"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/luannt2002/enterprise-ai-control-layer/core/contracts"
)

// Options là ngữ cảnh của một lần chuẩn hoá.
type Options struct {
	// Now là thời điểm tham chiếu ĐÃ Ở MÚI GIỜ của tenant: "hôm nay" là ngày của tenant,
	// không phải ngày của máy chủ.
	Now time.Time
	// Scope là phạm vi văn bản (fuel_report, site_report…) do kênh hoặc đường nạp quyết.
	// Dùng để phân định viết tắt nhiều nghĩa.
	Scope string
	// Layout bật đọc cách TRÌNH BÀY của tài liệu / bảng: đơn vị đứng trước số ("L 5.000", "lít: 5.000")
	// và đơn vị trong ngoặc sau số ("5.000 (lít/ngày)"). Chỉ bật khi quét văn bản TÀI LIỆU tìm lượng
	// nghiệp vụ (ForbidDimensions). Câu hỏi của người dùng KHÔNG bật: "bao nhiêu lít 7 ngày qua" sẽ bị
	// đọc nhầm "lít 7" là một lượng.
	Layout bool
	// Cell báo văn bản là MỘT ô lượng của nguồn có cấu trúc (bảng, ERP, phiếu): ô đã tách sẵn lượng khỏi chữ chung quanh, nên lượng viết nhóm nghìn
	// trong đó luôn chắc chắn — không bao giờ mang cờ chưa phân định (xem UncertainGroup). Văn bản chữ tự do (tin nhắn, câu hỏi, dòng báo cáo) không bật.
	Cell bool
}

// builtinMultipliers là từ chỉ hàng tiếng Việt đứng sau chữ số: "3 trăm", "4 nghìn".
// Đây là tri thức ngôn ngữ, dùng chung; tiếng lóng ("k", "tr", "củ") do pack khai báo.
var builtinMultipliers = []multAlias{
	{"nghìn", "1000"}, {"ngàn", "1000"}, {"triệu", "1000000"}, {"trăm", "100"},
	{"chục", "10"}, {"tỷ", "1000000000"}, {"tỉ", "1000000000"},
}

// Normalize chuẩn hoá một văn bản.
func (e *Engine) Normalize(text string, opt Options) contracts.NormalizedInput {
	if opt.Now.IsZero() {
		opt.Now = time.Now()
	}
	// Raw là văn bản sau khi gấp ký tự "nhìn giống" (foldLookalikes) rồi NFC; mọi vị trí byte của span trỏ vào Raw.
	raw := NFC(foldLookalikes(text))
	toks := tokenize(raw)

	var spans []contracts.Span
	var invalid []string // ngày hỏng ("31/02"): không sinh span, nhưng phải để lại dấu cho các chặng sau
	invalidEnd := 0
	approxUntil := -1
	for i := 0; i < len(toks); {
		t := toks[i]
		if t.kind == tokSpace || !isBoundaryBefore(raw, t.start) {
			i++
			continue
		}
		if sp, next, ok := e.matchCode(raw, toks, i); ok {
			spans, i = append(spans, sp), next
			continue
		}
		if t.kind == tokNum {
			sp, next, ok, bad := matchNumericDate(raw, toks, i, opt.Now)
			if ok && e.quantityFollows(raw, sp) {
				ok = false
			}
			if e.quantityFollows(raw, bad) {
				bad = contracts.Span{}
			}
			if ok {
				spans, i = append(spans, sp), next
				continue
			}
			if bad.Raw != "" {
				invalid, invalidEnd = append(invalid, bad.Raw), bad.End
			}
			if sp, next, ok := matchClock(raw, toks, i); ok {
				spans, i = append(spans, sp), next
				continue
			}
		}
		// "tháng M" của "ngày D tháng M" hỏng ("ngày 31 tháng 2") nằm trong cụm đã ghi: không đọc lại thành một kỳ mồ côi
		if t.kind == tokWord && t.start >= invalidEnd {
			sp, next, ok, bad := matchDayMonth(raw, toks, i, opt.Now)
			if ok {
				spans, i = append(spans, sp), next
				continue
			}
			if bad.Raw != "" {
				invalid, invalidEnd = append(invalid, bad.Raw), bad.End
			}
			if sp, next, ok := matchRelativeDate(raw, toks, i, opt.Now); ok {
				spans, i = append(spans, sp), next
				continue
			}
		}
		if end, ok := e.matchApprox(raw, t); ok {
			i = tokenAt(toks, end)
			approxUntil = i + 3 // dấu ước lượng áp cho lượng đứng ngay sau (cho phép 1 từ xen)
			continue
		}
		if t.kind == tokNum || t.kind == tokWord {
			if sp, next, ok := e.matchQuantity(raw, toks, i, opt.Layout); ok {
				if i <= approxUntil {
					sp.Value.Approx = true
				}
				if !opt.Cell {
					markUncertainGroup(raw, &sp)
				}
				spans, i = append(spans, sp), next
				continue
			}
		}
		if opt.Layout && t.kind == tokWord {
			if sp, next, ok := e.matchUnitFirst(raw, toks, i); ok {
				spans, i = append(spans, sp), next
				continue
			}
		}
		if t.kind == tokWord {
			if ab, ok := e.abbrs[t.text]; ok && isBoundaryAfter(raw, t.end) {
				spans = append(spans, contracts.Span{
					Kind: contracts.ValueTerm, Raw: t.text, Start: t.start, End: t.end,
					Value: contracts.Value{Kind: contracts.ValueTerm}, Rule: "abbr:pending:" + ab.Abbr,
				})
				i++
				continue
			}
			if sp, next, ok := e.matchTerm(raw, toks, i); ok {
				spans, i = append(spans, sp), next
				continue
			}
		}
		i++
	}

	spans, ambs := e.resolveAbbreviations(raw, spans, opt)
	spans = e.unitsFromTerms(raw, spans)

	lower := strings.ToLower(CollapseSpaces(raw))
	return contracts.NormalizedInput{
		Raw:          raw,
		Text:         lower,
		Folded:       Fold(lower),
		Canonical:    e.canonicalText(raw, spans),
		Spans:        spans,
		Ambiguities:  ambs,
		InvalidDates: invalid,
	}
}

// quantityFollows báo một ngày chỉ đọc được nhờ chữ "ngày" đứng trước (Rule có hậu tố ":ctx": "ngày 10-12", "ngày 2.5") thực ra là phần
// đầu của một lượng: ngay sau nó là một đơn vị ("mỗi ngày 10-12 giờ", "ngày 2.5 tấn"). Đánh đổi: "ngày 19.10 ca đêm" không đọc được —
// "ca" cũng là đơn vị ("2-3 ca"); viết "ngày 19/10 ca đêm" thì đọc được.
func (e *Engine) quantityFollows(s string, sp contracts.Span) bool {
	if !strings.HasSuffix(sp.Rule, ":ctx") {
		return false
	}
	_, _, _, ok := e.matchUnit(s, sp.End, false, false)
	return ok
}

// matchCode thử các mẫu mã của pack tại token i.
func (e *Engine) matchCode(s string, toks []token, i int) (contracts.Span, int, bool) {
	start := toks[i].start
	for _, c := range e.codes {
		loc := c.re.FindStringSubmatchIndex(s[start:])
		if loc == nil || loc[1] == 0 {
			continue
		}
		end := start + loc[1]
		if !isBoundaryAfter(s, end) {
			continue
		}
		canon := s[start:end]
		if c.def.Format != "" {
			canon = c.def.Format
			for _, name := range placeholders(c.def.Format) {
				idx := c.re.SubexpIndex(name)
				var part string
				if loc[2*idx] >= 0 {
					part = s[start+loc[2*idx] : start+loc[2*idx+1]]
				}
				canon = strings.ReplaceAll(canon, "{"+name+"}", part)
			}
		}
		if c.def.Upper {
			canon = strings.ToUpper(canon)
		}
		return contracts.Span{
			Kind: contracts.ValueCode, Raw: s[start:end], Start: start, End: end,
			Value:      contracts.Value{Kind: contracts.ValueCode, Text: canon},
			Confidence: 1, Rule: "code:" + c.def.Type,
		}, tokenAt(toks, end), true
	}
	return contracts.Span{}, i, false
}

func (e *Engine) matchApprox(s string, t token) (int, bool) {
	if t.kind == tokSym {
		// dấu ước lượng dạng ký hiệu: "~" (luôn nhận, như trước) và mọi ký hiệu pack khai ở approx_markers ("≈" của pack cầu đường). Trước đây
		// chỉ "~" được nhận nên "≈ 2,6 m3" trên phiếu kho mất cờ ước lượng (4 ca đỏ của bộ giữ lại P1).
		if t.text == "~" || slices.Contains(e.approx, t.text) {
			return t.end, true
		}
		return 0, false
	}
	if t.kind != tokWord {
		return 0, false
	}
	for _, m := range e.approx {
		if end, ok := matchAliasAt(s, t.start, m, false); ok {
			return end, true
		}
	}
	return 0, false
}

// matchQuantity đọc một con số (chữ số hoặc chữ), hậu tố nhân, rồi đơn vị nếu có. layout: xem Options.Layout.
func (e *Engine) matchQuantity(s string, toks []token, i int, layout bool) (contracts.Span, int, bool) {
	t := toks[i]
	var val *big.Rat
	end := t.end
	rule := "number:digits"
	needUnit := false
	conf := 1.0

	switch t.kind {
	case tokNum:
		text := t.text
		if n := e.spaceGroups(s, toks, i, layout); n > i { // "3 850 lít": nhóm nghìn ngăn bằng một dấu cách
			var b strings.Builder
			for k := i; k <= n; k += 2 {
				b.WriteString(toks[k].text)
			}
			text, end, rule = b.String(), toks[n].end, ruleSpaceThousands
		}
		v, ambiguousSep, ok := parseLocaleNumber(text)
		if !ok {
			return contracts.Span{}, i, false
		}
		val, _ = parseDecimal(v)
		if ambiguousSep {
			rule = "number:digits:vi_thousands"
		}
	case tokWord:
		v, n, hasScale, ok := parseViNumberWords(toks, i)
		if !ok {
			return contracts.Span{}, i, false
		}
		val, end, rule = v, toks[n-1].end, "number:words"
		// "năm" một mình còn là "năm (year)": dãy chữ không có hàng chỉ là số khi đứng
		// ngay trước một đơn vị ("năm người").
		needUnit = !hasScale
	}

	if mEnd, factor, frac, ok := e.matchMultiplier(s, end); ok {
		f, _ := parseDecimal(factor)
		val = new(big.Rat).Mul(new(big.Rat).Add(val, frac), f)
		end = mEnd
		rule += "+multiplier"
		needUnit = false
	}

	var unit *UnitDef
	enclosed := layout && openedBefore(s, t.start) && quantityLike(t.text) // "(4.000) lít": đơn vị đứng sau ngoặc đóng của số
	if uEnd, u, extra, ok := e.matchUnit(s, end, layout, enclosed); ok {
		unit = &u
		end = uEnd
		if extra != nil { // "7h30" thời lượng → 7,5 giờ
			val = new(big.Rat).Add(val, extra)
		}
	}
	if needUnit && unit == nil {
		return contracts.Span{}, i, false
	}
	if !isBoundaryAfter(s, end) {
		return contracts.Span{}, i, false
	}

	value := contracts.Value{Kind: contracts.ValueNumber, Number: formatDecimal(val)}
	kind := contracts.ValueNumber
	if unit != nil {
		kind = contracts.ValueKind(unit.Kind)
		value.Kind, value.Unit = kind, unit.Unit
		rule += "+unit"
	}
	return contracts.Span{
		Kind: kind, Raw: s[t.start:end], Start: t.start, End: end,
		Value: value, Confidence: conf, Rule: rule,
	}, tokenAt(toks, end), true
}

// Rule của span lượng viết nhóm nghìn ngăn bằng dấu cách ("3 850 lít"). Bản "uncertain" là cờ CHƯA PHÂN ĐỊNH: con số không đứng đầu văn bản
// (xem markUncertainGroup). Tầng sau đọc cờ qua UncertainGroup, không so chuỗi.
const (
	ruleSpaceThousands          = "number:digits:space_thousands"
	ruleSpaceThousandsUncertain = ruleSpaceThousands + ":uncertain"
	// uncertainGroupConfidence: hai cách đọc ngang nhau ("bể 2 500 lít": 2.500 L, hay bể 2 với 500 L) nên độ tin cậy 0,5 — hiện ở vết
	// (core/pipeline/why.go: "độ tin cậy 0.50").
	uncertainGroupConfidence = 0.5
)

// SpaceGroup báo span là lượng viết nhóm nghìn ngăn bằng MỘT dấu cách ("3 850 lít"), chắc chắn hay không.
func SpaceGroup(sp contracts.Span) bool { return strings.HasPrefix(sp.Rule, ruleSpaceThousands) }

// UncertainGroup báo span là lượng viết nhóm nghìn mà hệ CHƯA PHÂN ĐỊNH được cách đọc: con số không đứng đầu văn bản nên phần đầu có thể là số
// hiệu của chữ / ký hiệu đứng trước ("bể 2 500 lít": 2.500 L, hay bể 2 với 500 L; "bể #2 500 lít"; "bể: 2 500 lít"). Hệ vẫn gộp thành MỘT
// lượng — không để phần đầu thành số mồ côi ("12 000 km" không được thành "12" + "000 km") — nhưng không tin một cách đọc: luồng hỏi hỏi lại,
// đường nạp cách ly (P5: không đoán im lặng). Danh sách từ nhãn không phân định được (nó rò: soát C3, C3-02 / C3-05), nên không có danh sách.
func UncertainGroup(sp contracts.Span) bool {
	return strings.HasPrefix(sp.Rule, ruleSpaceThousandsUncertain)
}

// GroupSplitsName báo span là lượng viết nhóm nghìn mà phần đầu nằm trong một cụm tên đã nhận ra [nameStart, nameEnd) còn phần sau nằm ngoài
// ("3 850 lít" chồng lên "máy 3" của "máy 3 850 lít"): "3" là số hiệu của tên hay phần đầu của lượng? Hai cách đọc ngang nhau, nên người gọi
// cách ly / hỏi lại — không bỏ lượng im lặng vì "nằm trong tên" rồi lấy số khác ("máy 3 850 lít, phiếu 912" thành 912 L). Vị trí cùng hệ toạ độ
// với span (NormalizedInput.Raw); nhóm nằm trọn trong tên thì không chồng một phần.
func GroupSplitsName(sp contracts.Span, nameStart, nameEnd int) bool {
	return SpaceGroup(sp) && sp.Start < nameEnd && sp.End > nameStart && (sp.Start < nameStart || sp.End > nameEnd)
}

// markUncertainGroup gắn cờ chưa phân định cho span lượng viết nhóm nghìn mà con số không đứng đầu văn bản s.
func markUncertainGroup(s string, sp *contracts.Span) {
	if !SpaceGroup(*sp) || atTextStart(s, sp.Start) {
		return
	}
	sp.Rule = ruleSpaceThousandsUncertain + strings.TrimPrefix(sp.Rule, ruleSpaceThousands)
	sp.Confidence = uncertainGroupConfidence
}

// atTextStart báo trước vị trí pos của s chỉ có khoảng trắng và ký tự định dạng vô hình (ZWSP, BOM, dấu gạch nối mềm…): người đọc thấy con số
// đứng ĐẦU văn bản. strings.TrimSpace không bỏ các ký tự định dạng, nên "<ZWSP>3 850 lít" bị coi là có chữ đứng trước (soát bản thử (a') đêm 21/10, S3).
func atTextStart(s string, pos int) bool {
	return strings.TrimFunc(s[:pos], func(r rune) bool { return unicode.IsSpace(r) || isFormat(r) }) == ""
}

// spaceGroups trả chỉ số token của nhóm nghìn CUỐI khi toks[i] là phần đầu của một lượng viết nhóm nghìn ngăn bằng MỘT dấu
// cách ("3 850 lít", "1 234 567 L"; docs/tenants/construction/03-data-map.md:134: `4 000 lít` → 4.000 L): phần đầu 1–3 chữ số
// không bắt đầu bằng 0, mỗi nhóm sau đúng 3 chữ số, và ngay sau nhóm cuối là một đơn vị. Không phải thì trả i, giữ cách đọc
// cũ: "bước 3 850" (không có đơn vị), "máy 03 850 lít" (số 0 đầu là số hiệu), "3 85 lít", "1234 567 lít". Chữ đứng trước KHÔNG
// quyết định có gộp hay không ("đội 3 500 lít", "tổng số 3 850 lít", "bể 2 500 lít"): nhóm hợp lệ LUÔN gộp thành một span, và khi con
// số không đứng đầu văn bản thì span mang cờ chưa phân định (UncertainGroup). Danh sách từ nhãn / đếm / kỳ không phân định được — nó rò
// (soát C3, C3-02 / C3-05) — và để phần đầu thành số mồ côi thì "máy chạy 12 000 km" sinh "12" + "000 km", "tiền xe 500 000 đồng" sinh "500".
func (e *Engine) spaceGroups(s string, toks []token, i int, layout bool) int {
	digits := func(t token) bool {
		for _, r := range t.text {
			if r < '0' || r > '9' {
				return false
			}
		}
		return t.kind == tokNum && t.text != ""
	}
	if !digits(toks[i]) || len(toks[i].text) > 3 || toks[i].text[0] == '0' {
		return i
	}
	last := i
	for k := i + 2; k < len(toks); k += 2 {
		if toks[k-1].kind != tokSpace || toks[k-1].text != " " || !digits(toks[k]) || len(toks[k].text) != 3 {
			break
		}
		last = k
	}
	if last == i {
		return i
	}
	if _, _, _, ok := e.matchUnit(s, toks[last].end, layout, false); !ok {
		return i
	}
	return last
}

// matchMultiplier đọc hậu tố nhân ngay sau số ("4k", "4 nghìn") và phần lẻ dính liền
// ("1tr6" = 1,6 triệu, "1tr499" = 1.499.000, "4k5" = 4.500).
func (e *Engine) matchMultiplier(s string, pos int) (end int, factor string, frac *big.Rat, ok bool) {
	starts := []int{pos}
	if p := skipGap(s, pos); p > pos { // nhiều dấu cách hay ký tự vô hình giữa số và hậu tố: "5  k lít", "5<ZWSP>k lít"
		starts = append(starts, p)
	}
	all := append(append([]multAlias{}, e.mults...), builtinMultipliers...)
	sort.SliceStable(all, func(a, b int) bool { return len(all[a].alias) > len(all[b].alias) })
	for _, p := range starts {
		for _, m := range all {
			mEnd, ok := matchAliasPrefix(s, p, m.alias, false)
			if !ok {
				continue
			}
			// Phần lẻ dính liền, chỉ khi hậu tố viết sát số: "1tr6".
			if p == pos && mEnd < len(s) && isDigitAt(s, mEnd) {
				d := mEnd
				for d < len(s) && isDigitAt(s, d) {
					d++
				}
				if isBoundaryAfter(s, d) {
					digits := s[mEnd:d]
					num, _ := new(big.Int).SetString(digits, 10)
					den := new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(len(digits))), nil)
					return d, m.factor, new(big.Rat).SetFrac(num, den), true
				}
				continue
			}
			if isBoundaryAfter(s, mEnd) {
				return mEnd, m.factor, new(big.Rat), true
			}
		}
	}
	return 0, "", nil, false
}

// matchUnit đọc đơn vị ngay sau số, cho phép khoảng đệm ("4000 L", "4000L", "4000  L", "4000<ZWSP>L" — Normalize đã
// gấp tab, NBSP, U+202F… về dấu cách; ký tự định dạng vô hình cũng là khoảng đệm). layout: thêm dạng đơn vị MỞ ĐẦU một cặp ngoặc rồi đóng ngay,
// theo sau là "/…" hoặc một từ chữ phụ ("5.000 (L)", "5.000 (lít/ngày)", "5.000 (lít dầu)", xem bracketTail); "5 (khối lượng công
// việc)" không phải lượng. enclosed (chỉ khi layout): số đứng trong ngoặc, đơn vị đứng sau ngoặc đóng ("(4.000) lít").
// extra khác nil khi "7h30": phần phút cộng vào thời lượng.
func (e *Engine) matchUnit(s string, pos int, layout, enclosed bool) (end int, def UnitDef, extra *big.Rat, ok bool) {
	starts := []int{pos}
	p := skipGap(s, pos)
	if p > pos {
		starts = append(starts, p)
	}
	bracketed := -1
	if layout && p < len(s) && strings.IndexByte("([{", s[p]) >= 0 {
		bracketed = skipGap(s, p+1)
		starts = append(starts, bracketed)
	}
	if enclosed && p < len(s) && strings.IndexByte(")]}", s[p]) >= 0 {
		starts = append(starts, skipGap(s, p+1))
	}
	for _, p := range starts {
		for _, u := range e.units {
			if uEnd, ok := matchAliasAt(s, p, u.alias, u.def.CaseSensitive); ok {
				if p == bracketed && !bracketTail(s, uEnd) {
					continue
				}
				return uEnd, u.def, nil, true
			}
			// "7h30": đơn vị thời lượng viết dính phút.
			if u.def.Dimension == "duration" && p != bracketed {
				if uEnd, ok := matchAliasPrefix(s, p, u.alias, u.def.CaseSensitive); ok && uEnd+2 <= len(s) &&
					isDigitAt(s, uEnd) && isDigitAt(s, uEnd+1) && isBoundaryAfter(s, uEnd+2) {
					min := new(big.Rat).SetFrac64(int64((s[uEnd]-'0')*10+(s[uEnd+1]-'0')), 60)
					return uEnd + 2, u.def, min, true
				}
			}
		}
	}
	return 0, UnitDef{}, nil, false
}

// matchUnitFirst đọc lượng viết đơn vị TRƯỚC số: "L 5.000", "lít: 5.000", "(lít): 5.000", "lít/ca: 4.000" (đầu cột của bảng).
// Chỉ chạy khi Options.Layout. Số đã mang đơn vị riêng ("lít 7 giờ") thì số đó thuộc đơn vị của nó, không đọc. Không đọc khi đơn vị là
// MẪU SỐ của một giá ("đồng/lít 25.000": đứng sau "/"), hay khi chỉ cách số bằng khoảng trắng mà số không có dạng lượng — "khối 2",
// "L 3", "lít 2 liên" là nhãn bộ phận / mục / số đếm (xem quantityLike).
func (e *Engine) matchUnitFirst(s string, toks []token, i int) (contracts.Span, int, bool) {
	start := toks[i].start
	if start > 0 && s[start-1] == '/' {
		return contracts.Span{}, i, false
	}
	for _, u := range e.units {
		uEnd, ok := matchAliasAt(s, start, u.alias, u.def.CaseSensitive)
		if !ok {
			continue
		}
		p := uEnd
		if p < len(s) && s[p] == '/' { // đơn vị ghép "lít/ca", "lít/ngày": phần sau "/" là mẫu số, không phải số
			q := p + 1
			for q < len(s) {
				r, w := utf8.DecodeRuneInString(s[q:])
				if !unicode.IsLetter(r) && !unicode.IsDigit(r) {
					break
				}
				q += w
			}
			if q > p+1 {
				p = q
			}
		}
		sepStart := p
		for p < len(s) {
			r, w := utf8.DecodeRuneInString(s[p:])
			if !isFormat(r) && !strings.ContainsRune(" :=)]}", r) {
				break
			}
			p += w
		}
		j := tokenAt(toks, p)
		if j >= len(toks) || toks[j].start != p || toks[j].kind != tokNum {
			continue
		}
		if strings.TrimFunc(s[sepStart:p], func(r rune) bool { return r == ' ' || isFormat(r) }) == "" && !quantityLike(toks[j].text) {
			continue
		}
		num, next, ok := e.matchQuantity(s, toks, j, false)
		if !ok || num.Kind != contracts.ValueNumber {
			continue
		}
		num.Kind = contracts.ValueKind(u.def.Kind)
		num.Value.Kind, num.Value.Unit = num.Kind, u.def.Unit
		num.Start, num.Raw = start, s[start:num.End]
		num.Rule += "+unit_first"
		return num, next, true
	}
	return contracts.Span{}, i, false
}

// isFormat báo ký tự định dạng vô hình (ZWSP, ZWJ, BOM, dấu gạch nối mềm…).
func isFormat(r rune) bool { return unicode.Is(unicode.Cf, r) }

// skipGap trả vị trí đầu tiên từ pos trở đi không phải khoảng đệm (dấu cách hoặc ký tự định dạng vô hình).
func skipGap(s string, pos int) int {
	for pos < len(s) {
		r, w := utf8.DecodeRuneInString(s[pos:])
		if r != ' ' && !isFormat(r) {
			break
		}
		pos += w
	}
	return pos
}

// bracketTail báo phần đứng sau một đơn vị MỞ ĐẦU một cặp ngoặc có đúng là phần cuối của chú thích đơn vị không: ngoặc đóng, "/" (mẫu
// số: "(lít/ngày)") hoặc — sau khoảng đệm — một từ chữ phụ ("(lít dầu)"). "(khối lượng công việc)" thì không: ở đó "khối" là đầu
// một từ khác, không phải đơn vị.
func bracketTail(s string, unitEnd int) bool {
	q := skipGap(s, unitEnd)
	if q >= len(s) {
		return false
	}
	if strings.IndexByte(")]}/", s[q]) >= 0 {
		return true
	}
	r, _ := utf8.DecodeRuneInString(s[q:])
	if q == unitEnd || !unicode.IsLetter(r) {
		return false
	}
	_, heavy := matchAliasAt(s, q, "lượng", false)
	return !heavy
}

// openedBefore báo số bắt đầu ở start đứng ngay sau một dấu mở ngoặc ("(4.000"), chỉ cách khoảng đệm.
func openedBefore(s string, start int) bool {
	for start > 0 {
		r, w := utf8.DecodeLastRuneInString(s[:start])
		if r == ' ' || isFormat(r) {
			start -= w
			continue
		}
		return strings.ContainsRune("([{", r)
	}
	return false
}

// quantityLike báo token số viết kiểu một LƯỢNG: có dấu nghìn / thập phân, hoặc từ hai chữ số. Số MỘT chữ số đứng cạnh một từ trùng tên đơn
// vị thường là số thứ tự / nhãn ("khối 2", "(1) Khối tài chính", "lít 2 liên"), không phải lượng.
func quantityLike(text string) bool {
	return strings.ContainsAny(text, ".,") || len(strings.TrimLeft(text, "-−")) >= 2
}

// foldLookalikes gấp các ký tự NHÌN GIỐNG về dạng mà luật đọc số và đơn vị hiểu, để một lượng viết lạ không
// thoát khỏi chuẩn hoá: chữ số / dấu chấm / chữ cái toàn chiều rộng → ASCII; tab, NBSP, U+202F, U+3000… → dấu
// cách; "ℓ" → "l"; ký tự định dạng vô hình chen giữa một số thì bỏ (joinDigits).
func foldLookalikes(s string) string {
	return joinDigits(strings.Map(func(r rune) rune {
		switch {
		case r >= 0xFF01 && r <= 0xFF5E: // dạng toàn chiều rộng của ASCII
			return r - 0xFEE0
		case r != '\n' && r != '\r' && unicode.IsSpace(r):
			return ' '
		case r == 0x2113: // "ℓ", ký hiệu lít
			return 'l'
		}
		return r
	}, s))
}

// joinDigits bỏ ký tự định dạng vô hình CHEN GIỮA hai phần của một số ("3<ZWSP>850"): người đọc thấy 3850, nhưng bộ
// kiểm sẽ thấy hai số 3 và 850 (đều có bằng chứng) — số 3850 chưa từng được kiểm. Chỗ khác ký tự đó giữ nguyên: vẫn
// là ranh giới từ như trước, và là khoảng đệm khi đọc đơn vị (skipGap).
func joinDigits(s string) string {
	if !strings.ContainsFunc(s, isFormat) {
		return s
	}
	numeric := func(r rune) bool { return unicode.IsDigit(r) || r == '.' || r == ',' }
	rs := []rune(s)
	out := rs[:0]
	for i := 0; i < len(rs); i++ {
		if isFormat(rs[i]) && len(out) > 0 {
			j := i
			for j < len(rs) && isFormat(rs[j]) {
				j++
			}
			if prev := out[len(out)-1]; j < len(rs) && (unicode.IsDigit(prev) && numeric(rs[j]) || numeric(prev) && unicode.IsDigit(rs[j])) {
				i = j - 1
				continue
			}
		}
		out = append(out, rs[i])
	}
	return string(out)
}

// matchTerm thử bí danh thuật ngữ, dài trước.
func (e *Engine) matchTerm(s string, toks []token, i int) (contracts.Span, int, bool) {
	start := toks[i].start
	for _, ta := range e.terms {
		end, ok := matchAliasAt(s, start, ta.alias, false)
		if !ok {
			continue
		}
		conf, rule := 1.0, "term:exact"
		if !strings.EqualFold(s[start:end], ta.alias) {
			conf, rule = 0.9, "term:folded" // người gõ không dấu
		}
		return contracts.Span{
			Kind: contracts.ValueTerm, Raw: s[start:end], Start: start, End: end,
			Value:      contracts.Value{Kind: contracts.ValueTerm, Text: ta.def.Term},
			Confidence: conf, Rule: rule,
		}, tokenAt(toks, end), true
	}
	return contracts.Span{}, i, false
}

// resolveAbbreviations phân định nghĩa viết tắt. Không phân định được thì KHÔNG đoán.
func (e *Engine) resolveAbbreviations(s string, spans []contracts.Span, opt Options) ([]contracts.Span, []contracts.Ambiguity) {
	var out []contracts.Span
	var ambs []contracts.Ambiguity
	for k, sp := range spans {
		abbr, pending := strings.CutPrefix(sp.Rule, "abbr:pending:")
		if !pending {
			out = append(out, sp)
			continue
		}
		def := e.abbrs[abbr]
		var next *contracts.Span
		if k+1 < len(spans) && tightGap(s[sp.End:spans[k+1].Start]) {
			next = &spans[k+1]
		}
		chosen, rule, conf := "", "", 0.0
		if len(def.Meanings) == 1 {
			// Một nghĩa thì không có gì để đoán ("NC 22" = nhân công 22).
			chosen, rule, conf = def.Meanings[0].Term, "abbr:single", 0.9
		}
		if chosen == "" && next != nil && (next.Kind == contracts.ValueQuantity || next.Kind == contracts.ValueMoney) {
			dim := e.unitDefs[next.Value.Unit].Dimension
			for _, m := range def.Meanings {
				if m.WhenNextDimension != "" && m.WhenNextDimension == dim {
					chosen, rule, conf = m.Term, "abbr:next_dimension", 1
					break
				}
			}
		}
		if chosen == "" && opt.Scope != "" {
			for _, m := range def.Meanings {
				for _, sc := range m.Scopes {
					if sc == opt.Scope {
						chosen, rule, conf = m.Term, "abbr:scope", 0.9
					}
				}
			}
		}
		// Đứng trước một số trần mà chưa phân định được: hai nghĩa đều có thể đúng
		// ("DA: 4.000" — dầu hay dự án?). Mặc định ở đây là đoán, nên không dùng.
		beforeBareNumber := next != nil && next.Kind == contracts.ValueNumber
		if chosen == "" && !beforeBareNumber {
			for _, m := range def.Meanings {
				if m.Default {
					chosen, rule, conf = m.Term, "abbr:default", 0.7
				}
			}
		}
		if chosen == "" {
			amb := contracts.Ambiguity{Raw: sp.Raw, Reason: "viết tắt nhiều nghĩa, ngữ cảnh không đủ để chọn"}
			for _, m := range def.Meanings {
				amb.Candidates = append(amb.Candidates, contracts.Value{Kind: contracts.ValueTerm, Text: m.Term})
			}
			ambs = append(ambs, amb)
			continue
		}
		sp.Value.Text, sp.Rule, sp.Confidence = chosen, rule, conf
		out = append(out, sp)
	}
	return out, ambs
}

// unitsFromTerms gắn đơn vị mặc định của thuật ngữ cho số trần đứng cạnh nó
// ("dầu 4k" → 4000 L, "NC 22" → 22 người). Là suy luận, nên độ tin cậy 0,8.
func (e *Engine) unitsFromTerms(s string, spans []contracts.Span) []contracts.Span {
	for k := range spans {
		if spans[k].Kind != contracts.ValueNumber {
			continue
		}
		term := ""
		// thuật ngữ đứng trước, cùng mệnh đề, cách tối đa 2 từ
		for p := k - 1; p >= 0; p-- {
			if spans[p].Kind != contracts.ValueTerm {
				continue
			}
			gap := s[spans[p].End:spans[k].Start]
			if sameClause(gap) && wordCount(gap) <= 2 {
				term = spans[p].Value.Text
			}
			break
		}
		// hoặc thuật ngữ đứng ngay sau ("4k dầu")
		if term == "" && k+1 < len(spans) && spans[k+1].Kind == contracts.ValueTerm && tightGap(s[spans[k].End:spans[k+1].Start]) {
			term = spans[k+1].Value.Text
		}
		def, ok := e.termDefs[term]
		if !ok || def.DefaultUnit == "" {
			continue
		}
		u := e.unitDefs[def.DefaultUnit]
		spans[k].Kind = contracts.ValueKind(u.Kind)
		spans[k].Value.Kind = contracts.ValueKind(u.Kind)
		spans[k].Value.Unit = u.Unit
		spans[k].Confidence = 0.8
		spans[k].Rule += "+unit_from_term:" + term
	}
	return spans
}

// tightGap: giữa hai span chỉ có khoảng trắng hoặc dấu nối ngắn (":", "-", "=").
func tightGap(gap string) bool {
	if utf8.RuneCountInString(gap) > 3 {
		return false
	}
	return strings.Trim(gap, " :-=\t") == ""
}

// sameClause: đoạn giữa không có dấu ngắt mệnh đề.
func sameClause(gap string) bool { return !strings.ContainsAny(gap, "\n,;.!?") }

func wordCount(s string) int { return len(strings.Fields(strings.Trim(s, " :-="))) }

func isDigitAt(s string, i int) bool { return i < len(s) && s[i] >= '0' && s[i] <= '9' }

// canonicalText thay mỗi span bằng dạng chuẩn: "dầu 4k" → "dầu 4000 L".
func (e *Engine) canonicalText(s string, spans []contracts.Span) string {
	var b strings.Builder
	prev := 0
	for _, sp := range spans {
		if sp.Start < prev {
			continue
		}
		b.WriteString(s[prev:sp.Start])
		b.WriteString(e.render(sp))
		prev = sp.End
	}
	b.WriteString(s[prev:])
	return CollapseSpaces(b.String())
}

func (e *Engine) render(sp contracts.Span) string {
	v := sp.Value
	switch sp.Kind {
	case contracts.ValueQuantity, contracts.ValueMoney:
		return v.Number + " " + v.Unit
	case contracts.ValueNumber:
		return v.Number
	case contracts.ValueTerm:
		if d, ok := e.termDefs[v.Text]; ok && d.Label != "" {
			return d.Label
		}
		return sp.Raw
	default:
		return v.Text
	}
}

// Convert đổi một lượng sang đơn vị khác CÙNG chiều ("4 m3" → "4000 L").
func (e *Engine) Convert(v contracts.Value, to string) (contracts.Value, error) {
	from, ok1 := e.unitDefs[v.Unit]
	target, ok2 := e.unitDefs[to]
	if !ok1 || !ok2 {
		return v, contracts.Errorf(contracts.FaultCode, "đơn vị chưa khai báo: "+v.Unit+" → "+to)
	}
	if from.Dimension != target.Dimension {
		return v, contracts.Errorf(contracts.FaultData, "không đổi được "+from.Dimension+" sang "+target.Dimension)
	}
	n, ok := parseDecimal(v.Number)
	if !ok {
		return v, contracts.Errorf(contracts.FaultData, "số không hợp lệ: "+v.Number)
	}
	fb, _ := parseDecimal(from.ToBase)
	tb, _ := parseDecimal(target.ToBase)
	out := new(big.Rat).Mul(n, fb)
	out.Quo(out, tb)
	v.Number, v.Unit = formatDecimal(out), target.Unit
	return v, nil
}
