package normalization

import (
	"fmt"
	"strconv"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/luannt2002/enterprise-ai-systems/core/contracts"
)

// Ngày, giờ và khoảng thời gian — "TIME RESOLUTION" là một chặng tất định: "tháng này"
// phải thành đúng một khoảng ngày theo đồng hồ và múi giờ của tenant, không để model đoán.

// timePrepositions đứng trước "14h" thì đó là MỐC GIỜ ("mưa từ 14h"), không phải thời
// lượng ("máy chạy 7h").
var timePrepositions = []string{"từ", "lúc", "đến", "tới", "khoảng lúc", "bắt đầu"}

// durationVerbs đứng trước "7h" thì đó là THỜI LƯỢNG.
var durationVerbs = []string{"chạy", "làm", "hoạt động", "mất", "trong", "tổng"}

// dayParts đứng sau giờ thì đó là mốc giờ, kèm điều chỉnh sáng/chiều.
var dayParts = map[string]int{"sáng": 0, "trưa": 0, "chiều": 12, "tối": 12, "đêm": 12}

// relativeDates: cụm từ → hàm tính khoảng ngày [from, to] theo now.
var relativeDates = []struct {
	phrase  string
	resolve func(now time.Time) (from, to time.Time)
}{
	{"hôm nay", func(n time.Time) (time.Time, time.Time) { return day(n), day(n) }},
	{"bữa nay", func(n time.Time) (time.Time, time.Time) { return day(n), day(n) }},
	{"hôm qua", func(n time.Time) (time.Time, time.Time) { d := day(n).AddDate(0, 0, -1); return d, d }},
	{"hqua", func(n time.Time) (time.Time, time.Time) { d := day(n).AddDate(0, 0, -1); return d, d }}, // viết tắt khi chat
	{"hôm kia", func(n time.Time) (time.Time, time.Time) { d := day(n).AddDate(0, 0, -2); return d, d }},
	{"ngày mai", func(n time.Time) (time.Time, time.Time) { d := day(n).AddDate(0, 0, 1); return d, d }},
	{"tuần này", func(n time.Time) (time.Time, time.Time) { s := weekStart(n); return s, s.AddDate(0, 0, 6) }},
	{"tuần trước", func(n time.Time) (time.Time, time.Time) {
		s := weekStart(n).AddDate(0, 0, -7)
		return s, s.AddDate(0, 0, 6)
	}},
	{"tháng này", func(n time.Time) (time.Time, time.Time) { return monthRange(n.Year(), int(n.Month()), n.Location()) }},
	{"tháng trước", func(n time.Time) (time.Time, time.Time) {
		p := time.Date(n.Year(), n.Month(), 1, 0, 0, 0, 0, n.Location()).AddDate(0, -1, 0)
		return monthRange(p.Year(), int(p.Month()), n.Location())
	}},
	{"năm nay", func(n time.Time) (time.Time, time.Time) {
		return time.Date(n.Year(), 1, 1, 0, 0, 0, 0, n.Location()), time.Date(n.Year(), 12, 31, 0, 0, 0, 0, n.Location())
	}},
}

func day(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location())
}

func weekStart(t time.Time) time.Time {
	d := day(t)
	offset := (int(d.Weekday()) + 6) % 7 // thứ Hai là đầu tuần
	return d.AddDate(0, 0, -offset)
}

func monthRange(year, month int, loc *time.Location) (time.Time, time.Time) {
	from := time.Date(year, time.Month(month), 1, 0, 0, 0, 0, loc)
	return from, from.AddDate(0, 1, -1)
}

func dateValue(t time.Time) contracts.Value {
	return contracts.Value{Kind: contracts.ValueDate, Text: t.Format("2006-01-02")}
}

func periodValue(from, to time.Time) contracts.Value {
	if from.Equal(to) {
		return dateValue(from)
	}
	return contracts.Value{Kind: contracts.ValuePeriod, Text: from.Format("2006-01-02") + "/" + to.Format("2006-01-02")}
}

// matchRelativeDate thử các cụm "hôm nay", "tháng trước"… và "tháng 9", "tháng 9/2026".
func matchRelativeDate(s string, toks []token, i int, now time.Time) (contracts.Span, int, bool) {
	start := toks[i].start
	for _, rd := range relativeDates {
		end, ok := matchAliasAt(s, start, rd.phrase, false)
		if !ok {
			continue
		}
		from, to := rd.resolve(now)
		return contracts.Span{
			Kind: valueKindOf(from, to), Raw: s[start:end], Start: start, End: end,
			Value: periodValue(from, to), Confidence: 1, Rule: "date:relative",
		}, tokenAt(toks, end), true
	}
	// "tháng 9" hoặc "tháng 9/2026"
	end, ok := matchAliasAt(s, start, "tháng", false)
	if !ok {
		return contracts.Span{}, i, false
	}
	j := tokenAt(toks, end)
	if j < len(toks) && toks[j].kind == tokSpace {
		j++
	}
	if j >= len(toks) || toks[j].kind != tokNum {
		return contracts.Span{}, i, false
	}
	month, err := strconv.Atoi(toks[j].text)
	if err != nil || month < 1 || month > 12 {
		return contracts.Span{}, i, false
	}
	year := now.Year()
	explicitYear := false
	k := j + 1
	if k+1 < len(toks) && toks[k].kind == tokSym && toks[k].text == "/" && toks[k+1].kind == tokNum && len(toks[k+1].text) == 4 {
		y, _ := strconv.Atoi(toks[k+1].text)
		year, explicitYear = y, true
		k += 2
	}
	// Không ghi năm: lấy lần gần nhất KHÔNG ở tương lai ("tháng 11" hỏi vào tháng 10
	// là tháng 11 năm trước). Báo cáo vận hành hỏi về quá khứ.
	if !explicitYear && month > int(now.Month()) {
		year--
	}
	from, to := monthRange(year, month, now.Location())
	endByte := toks[k-1].end
	return contracts.Span{
		Kind: contracts.ValuePeriod, Raw: s[start:endByte], Start: start, End: endByte,
		Value: periodValue(from, to), Confidence: 1, Rule: "date:month",
	}, k, true
}

func valueKindOf(from, to time.Time) contracts.ValueKind {
	if from.Equal(to) {
		return contracts.ValueDate
	}
	return contracts.ValuePeriod
}

// matchNumericDate đọc "16/9", "16/09/2026", "16/9/26", "16-9-2026", "16.9.26", "2025-11-28", và — khi có chữ "ngày" đứng ngay
// trước — cả "ngày 16.9", "ngày 16-9", "ngày 16-9-26", "ngày 16 / 9".
//
// Năm hai chữ số là 20yy và dấu "." là cách viết ngày CÓ THẬT (docs/tenants/construction/03-data-map.md:162: `19/10`, `19-10-2026`,
// `19.10.26`; 02-domain-map.md:71, :296: `19.10.26`, `19-10`). Hai số nối bằng "." hay "-" ("19.10" là số thập phân, "10-12 giờ" là
// khoảng giờ) và ba số nhỏ nối bằng "-" ("TK-01-02-03" là mã) chỉ là ngày khi người hỏi nói "ngày".
//
// Cụm có DẠNG ngày mà không đọc được thành một ngày trên lịch ("31/2", "32/10", "29/02/2026", "31.02") không phải ngày: trả ok=false
// KÈM bad (span của cả cụm, Rule "date:invalid") để Normalize ghi vào InvalidDates. Người hỏi đã cố nêu một ngày — bỏ qua im lặng thì
// câu hỏi rơi về ngày mặc định như thể họ không nêu (P5). "Có dạng ngày" xem dateLike.
//
// Cụm dính vào thứ đứng trước nó ("PX-12/10", "BB-24/50": mã chữ; "2/3" của "1/2/3": dãy số dài hơn) không phải ngày — xem gluedBefore.
func matchNumericDate(s string, toks []token, i int, now time.Time) (sp contracts.Span, next int, ok bool, bad contracts.Span) {
	none := func() (contracts.Span, int, bool, contracts.Span) {
		return contracts.Span{}, i, false, contracts.Span{}
	}
	if gluedBefore(s, toks[i].start) {
		return none()
	}
	ctx := afterDayWord(s, toks, i)
	parts, seps, j := gatherDate(toks, i, ctx)
	if len(parts) == 1 { // "19.10.26", "19.10": MỘT token số có dấu chấm
		if sub := splitDotted(parts[0]); sub != nil {
			parts, seps = sub, make([]string, len(sub)-1)
			for k := range seps {
				seps[k] = "."
			}
		}
	}
	if len(parts) < 2 || len(parts) > 3 {
		return none()
	}
	for _, p := range parts {
		if strings.ContainsAny(p, ".,") {
			return none()
		}
	}
	atoi := func(x string) int { v, _ := strconv.Atoi(x); return v }
	var y, m, d, yearDigits int
	switch {
	case len(parts) == 3 && len(parts[0]) == 4: // ISO yyyy-mm-dd
		y, m, d, yearDigits = atoi(parts[0]), atoi(parts[1]), atoi(parts[2]), 4
	case len(parts) == 3 && len(parts[2]) == 4: // dd/mm/yyyy
		d, m, y, yearDigits = atoi(parts[0]), atoi(parts[1]), atoi(parts[2]), 4
	case len(parts) == 3 && smallInt(parts[0]) && smallInt(parts[1]) && smallInt(parts[2]) && len(parts[2]) == 2 &&
		seps[0] == seps[1] && (seps[0] != "-" || ctx): // dd/mm/yy, dd.mm.yy (dd-mm-yy chỉ khi có chữ "ngày")
		d, m, y, yearDigits = atoi(parts[0]), atoi(parts[1]), 2000+atoi(parts[2]), 2
	case len(parts) == 2 && (seps[0] == "/" || ctx): // dd/mm — năm hiện tại, lùi một năm nếu ở tương lai
		d, m, y = atoi(parts[0]), atoi(parts[1]), now.Year()
		if time.Date(y, time.Month(m), d, 0, 0, 0, 0, now.Location()).After(day(now)) {
			y--
		}
	default:
		return none()
	}
	// loose: chỉ đọc được nhờ chữ "ngày" đứng trước (dấu "." / "-" hay khoảng trắng quanh dấu) — có thể là phần đầu của một lượng ("mỗi
	// ngày 10-12 giờ", "ngày 2.5 tấn"): Rule mang hậu tố ":ctx" để Normalize bỏ đọc khi ngay sau là một đơn vị (Engine.quantityFollows).
	loose := ctx && yearDigits != 4 && (seps[0] != "/" || strings.Contains(s[toks[i].start:toks[j-1].end], " "))
	rule := "date:numeric"
	if loose {
		rule += ":ctx"
	}
	t := time.Date(y, time.Month(m), d, 0, 0, 0, 0, now.Location())
	if m < 1 || m > 12 || d < 1 || d > 31 || t.Day() != d {
		if !dateLike(d, m, yearDigits, ctx) {
			return none()
		}
		b := badDate(s, toks, i, j) // 31/2 không phải ngày
		if loose && b.Rule != "" {
			b.Rule += ":ctx"
		}
		return contracts.Span{}, i, false, b
	}
	start, end := toks[i].start, toks[j-1].end
	if !isBoundaryAfter(s, end) {
		return none()
	}
	return contracts.Span{
		Kind: contracts.ValueDate, Raw: s[start:end], Start: start, End: end,
		Value: dateValue(t), Confidence: 1, Rule: rule,
	}, j, true, contracts.Span{}
}

// dateLike báo một cụm số (ngày d, tháng m) CÓ DẠNG ngày, nên khi không đọc được thành ngày trên lịch thì đáng bị ghi là ngày hỏng
// thay vì lặng lẽ rơi về hôm nay: ngày 1–31 và tháng 1–12 (chỉ sai vì lịch: "31/02", "29/02/2026", "31/04"). Cụm khác ("24/24",
// "60/40", "12/34/56": tỉ lệ, số phiếu, hợp đồng) chỉ là hai hay ba số. Người hỏi đã nói đây là ngày — chữ "ngày" đứng trước, hay năm
// bốn chữ số kèm theo ("2026-13-01") — thì lỏng hơn: mỗi phần ≤ 99 ("ngày 32/10", "ngày 15/13").
func dateLike(d, m, yearDigits int, ctx bool) bool {
	if d >= 1 && d <= 31 && m >= 1 && m <= 12 {
		return true
	}
	return (ctx || yearDigits == 4) && d >= 0 && d <= 99 && m >= 0 && m <= 99
}

// afterDayWord báo từ "ngày" đứng ngay trước token i (chỉ cách khoảng trắng): người hỏi đã nói đây là một ngày.
func afterDayWord(s string, toks []token, i int) bool {
	k := i - 1
	if k >= 0 && toks[k].kind == tokSpace {
		k--
	}
	if k < 0 || toks[k].kind != tokWord {
		return false
	}
	_, ok := matchAliasAt(s, toks[k].start, "ngày", false)
	return ok
}

// gluedBefore báo cụm số bắt đầu ở start dính vào thứ đứng ngay trước nó: sau một chữ cái và dấu nối ("PX-12/10", "BB-24/50",
// "TK-19/10" là mã chữ), hoặc sau một chữ số và dấu "/" ("2/3" của "1/2/3": một dãy số dài hơn, không phải ngày).
func gluedBefore(s string, start int) bool {
	prev, w := utf8.DecodeLastRuneInString(s[:start])
	if w == 0 || !strings.ContainsRune("-–/._", prev) {
		return false
	}
	before, bw := utf8.DecodeLastRuneInString(s[:start-w])
	if bw == 0 {
		return false
	}
	return unicode.IsLetter(before) || unicode.Is(unicode.Mn, before) || (prev == '/' && unicode.IsDigit(before))
}

// gatherDate gom dãy phần số nối bằng "/" hoặc "-" từ token i, không có khoảng trắng; spaced cho phép một dấu cách quanh "/" ("ngày 19 /
// 10"). Trả các phần, các dấu nối và chỉ số token ngay sau phần cuối.
func gatherDate(toks []token, i int, spaced bool) (parts, seps []string, j int) {
	j = i
	for j < len(toks) && toks[j].kind == tokNum {
		parts = append(parts, toks[j].text)
		k := j + 1
		if spaced && k < len(toks) && toks[k].kind == tokSpace {
			k++
		}
		if k < len(toks) && toks[k].kind == tokSym && (toks[k].text == "/" || (toks[k].text == "-" && k == j+1)) {
			n := k + 1
			if spaced && toks[k].text == "/" && n < len(toks) && toks[n].kind == tokSpace {
				n++
			}
			if n < len(toks) && toks[n].kind == tokNum {
				seps = append(seps, toks[k].text)
				j = n
				continue
			}
		}
		j++
		break
	}
	return parts, seps, j
}

// splitDotted tách MỘT token số viết bằng dấu chấm ("19.10.26", "19.10") thành 2–3 nhóm chữ số; nil nếu không phải. Ngày, tháng và năm
// hai chữ số có tối đa 2 chữ số (năm bốn chữ số chỉ ở phần cuối của ba phần): nhóm 3 chữ số ("4.000", "1.500") là dấu nghìn của một SỐ.
func splitDotted(text string) []string {
	if !strings.Contains(text, ".") || strings.Contains(text, ",") {
		return nil
	}
	sub := strings.Split(text, ".")
	if len(sub) < 2 || len(sub) > 3 {
		return nil
	}
	for k, p := range sub {
		max := 2
		if len(sub) == 3 && k == 2 {
			max = 4
		}
		if p == "" || len(p) > max || strings.Trim(p, "0123456789") != "" {
			return nil
		}
	}
	return sub
}

// matchDayMonth đọc "ngày 20 tháng 10", "ngày 20 tháng 10 năm 2026" và "ngày 20 tháng 10/2026" thành MỘT ngày (Raw gồm cả chữ
// "ngày"). Trước đây chỉ đọc "tháng 10" và bỏ "ngày 20": một ngày viết hai cách ("hôm nay, ngày 20 tháng 10") thành hai kỳ khác nhau,
// và "từ ngày 10 đến ngày 15 tháng 10" thành cả tháng. "ngày D" đứng trước "đến/tới ngày D2 tháng M" mượn tháng, năm của đầu mút sau
// (cả hai thành span ngày: câu nêu hai kỳ và được hỏi lại). Không ghi năm thì lấy lần gần nhất KHÔNG ở tương lai, như dd/mm.
// Ngày không có trên lịch ("ngày 31 tháng 2") trả bad (Raw bắt đầu từ số ngày, không kèm chữ "ngày").
func matchDayMonth(s string, toks []token, i int, now time.Time) (sp contracts.Span, next int, ok bool, bad contracts.Span) {
	none := func() (contracts.Span, int, bool, contracts.Span) {
		return contracts.Span{}, i, false, contracts.Span{}
	}
	start := toks[i].start
	end, found := matchAliasAt(s, start, "ngày", false)
	if !found {
		return none()
	}
	j := skipSpaceToken(toks, tokenAt(toks, end))
	if j >= len(toks) || toks[j].kind != tokNum || !smallInt(toks[j].text) {
		return none()
	}
	d, _ := strconv.Atoi(toks[j].text)
	invalid := func(from, to int) (contracts.Span, int, bool, contracts.Span) {
		a, b := toks[from].start, toks[to-1].end
		if !isBoundaryAfter(s, b) {
			return none()
		}
		return contracts.Span{}, i, false, contracts.Span{Kind: contracts.ValueDate, Raw: s[a:b], Start: a, End: b, Rule: "date:invalid"}
	}
	if m, y, explicit, after, found := monthYearAt(s, toks, skipSpaceToken(toks, j+1)); found {
		t, valid := dayMonthDate(d, m, y, explicit, now)
		if !valid {
			return invalid(j, after)
		}
		if endByte := toks[after-1].end; isBoundaryAfter(s, endByte) {
			return contracts.Span{
				Kind: contracts.ValueDate, Raw: s[start:endByte], Start: start, End: endByte,
				Value: dateValue(t), Confidence: 1, Rule: "date:day_month",
			}, after, true, contracts.Span{}
		}
		return none()
	}
	// "ngày 10 đến ngày 15 tháng 10": đầu mút thứ nhất mượn tháng, năm của đầu mút thứ hai
	c := skipSpaceToken(toks, j+1)
	if c >= len(toks) || toks[c].kind != tokWord || !(isWordAt(s, toks[c], "đến") || isWordAt(s, toks[c], "tới")) {
		return none()
	}
	n := skipSpaceToken(toks, c+1)
	if n >= len(toks) || toks[n].kind != tokWord || !isWordAt(s, toks[n], "ngày") {
		return none()
	}
	n = skipSpaceToken(toks, n+1)
	if n >= len(toks) || toks[n].kind != tokNum || !smallInt(toks[n].text) {
		return none()
	}
	m, y, explicit, _, found := monthYearAt(s, toks, skipSpaceToken(toks, n+1))
	if !found {
		return none()
	}
	t, valid := dayMonthDate(d, m, y, explicit, now)
	if !valid {
		return invalid(j, j+1)
	}
	endByte := toks[j].end
	return contracts.Span{
		Kind: contracts.ValueDate, Raw: s[start:endByte], Start: start, End: endByte,
		Value: dateValue(t), Confidence: 1, Rule: "date:day_borrowed",
	}, j + 1, true, contracts.Span{}
}

// monthYearAt đọc "tháng M" kèm năm tuỳ chọn ("năm 2026" hoặc "/2026") bắt đầu ở token k; after là chỉ số token ngay sau cụm.
func monthYearAt(s string, toks []token, k int) (month, year int, explicitYear bool, after int, ok bool) {
	if k >= len(toks) || toks[k].kind != tokWord || !isWordAt(s, toks[k], "tháng") {
		return 0, 0, false, k, false
	}
	j := skipSpaceToken(toks, k+1)
	if j >= len(toks) || toks[j].kind != tokNum || !smallInt(toks[j].text) {
		return 0, 0, false, k, false
	}
	month, _ = strconv.Atoi(toks[j].text)
	after = j + 1
	isYear := func(t token) bool {
		return t.kind == tokNum && len(t.text) == 4 && strings.Trim(t.text, "0123456789") == ""
	}
	switch {
	case after+1 < len(toks) && toks[after].kind == tokSym && toks[after].text == "/" && isYear(toks[after+1]): // "tháng 10/2026"
		year, explicitYear, after = atoiOrZero(toks[after+1].text), true, after+2
	default: // "tháng 10 năm 2026" ("năm nay" thì không phải năm viết ra: bỏ qua)
		if w := skipSpaceToken(toks, after); w < len(toks) && toks[w].kind == tokWord && isWordAt(s, toks[w], "năm") {
			if n := skipSpaceToken(toks, w+1); n < len(toks) && isYear(toks[n]) {
				year, explicitYear, after = atoiOrZero(toks[n].text), true, n+1
			}
		}
	}
	return month, year, explicitYear, after, true
}

// dayMonthDate dựng ngày d/m/y; năm không viết thì lấy lần gần nhất không ở tương lai. valid = có trên lịch.
func dayMonthDate(d, m, y int, explicitYear bool, now time.Time) (time.Time, bool) {
	if m < 1 || m > 12 || d < 1 || d > 31 {
		return time.Time{}, false
	}
	if !explicitYear {
		y = now.Year()
		if time.Date(y, time.Month(m), d, 0, 0, 0, 0, now.Location()).After(day(now)) {
			y--
		}
	}
	t := time.Date(y, time.Month(m), d, 0, 0, 0, 0, now.Location())
	return t, t.Day() == d
}

// skipSpaceToken trả chỉ số token đầu tiên từ k trở đi không phải khoảng trắng (khoảng trắng liên tiếp là MỘT token).
func skipSpaceToken(toks []token, k int) int {
	if k < len(toks) && toks[k].kind == tokSpace {
		return k + 1
	}
	return k
}

// isWordAt báo token t là đúng từ word (theo luật dấu của mọi so khớp: gõ không dấu vẫn khớp).
func isWordAt(s string, t token, word string) bool {
	end, ok := matchAliasAt(s, t.start, word, false)
	return ok && end == t.end
}

func atoiOrZero(x string) int { v, _ := strconv.Atoi(x); return v }

// smallInt báo p là số nguyên không dấu, tối đa hai chữ số — cỡ của ngày, tháng, năm viết tắt.
func smallInt(p string) bool { return p != "" && len(p) <= 2 && strings.Trim(p, "0123456789") == "" }

// badDate là span của cụm số ở các token [i, j) khi nó đứng riêng thành một cụm (có ranh giới từ sau cụm — "31/02R"
// không phải ngày hỏng); ngược lại là span rỗng.
func badDate(s string, toks []token, i, j int) contracts.Span {
	start, end := toks[i].start, toks[j-1].end
	if !isBoundaryAfter(s, end) {
		return contracts.Span{}
	}
	return contracts.Span{Kind: contracts.ValueDate, Raw: s[start:end], Start: start, End: end, Rule: "date:invalid"}
}

// matchClock đọc "14h", "14h30", "14:30", "2h chiều" khi ngữ cảnh nói đây là MỐC giờ.
// Trả ok=false để bộ đọc lượng xử lý "7h" như thời lượng.
func matchClock(s string, toks []token, i int) (contracts.Span, int, bool) {
	if toks[i].kind != tokNum || strings.ContainsAny(toks[i].text, ".,") {
		return contracts.Span{}, i, false
	}
	hour, err := strconv.Atoi(toks[i].text)
	if err != nil || hour > 24 {
		return contracts.Span{}, i, false
	}
	j := i + 1
	minute := 0
	hasMinute := false
	switch {
	case j < len(toks) && toks[j].kind == tokWord && (strings.EqualFold(toks[j].text, "h") || strings.EqualFold(toks[j].text, "g")) && toks[j].start == toks[i].end:
		j++
		if j < len(toks) && toks[j].kind == tokNum && toks[j].start == toks[j-1].end && len(toks[j].text) == 2 {
			minute, _ = strconv.Atoi(toks[j].text)
			hasMinute = true
			j++
		}
	case j+1 < len(toks) && toks[j].kind == tokSym && toks[j].text == ":" && toks[j+1].kind == tokNum && len(toks[j+1].text) == 2:
		minute, _ = strconv.Atoi(toks[j+1].text)
		hasMinute = true
		j += 2
		// "14:30" luôn là mốc giờ
		return clockSpan(s, toks, i, j, hour, minute, "time:colon")
	default:
		return contracts.Span{}, i, false
	}
	if minute > 59 {
		return contracts.Span{}, i, false
	}
	end := toks[j-1].end
	if !isBoundaryAfter(s, end) {
		return contracts.Span{}, i, false
	}

	// Ngữ cảnh: phần trong ngày đứng sau → mốc giờ.
	k := j
	if k < len(toks) && toks[k].kind == tokSpace {
		k++
	}
	if k < len(toks) && toks[k].kind == tokWord {
		for part, shift := range dayParts {
			if e, ok := matchAliasAt(s, toks[k].start, part, false); ok {
				h := hour
				if shift > 0 && h < 12 {
					h += shift
				}
				sp, _, _ := clockSpan(s, toks, i, j, h, minute, "time:daypart")
				sp.End, sp.Raw = e, s[toks[i].start:e]
				return sp, tokenAt(toks, e), true
			}
		}
	}
	prev := previousWords(s, toks, i, 2)
	for _, v := range durationVerbs {
		if strings.HasSuffix(prev, v) {
			return contracts.Span{}, i, false
		}
	}
	for _, p := range timePrepositions {
		if strings.HasSuffix(prev, p) {
			return clockSpan(s, toks, i, j, hour, minute, "time:preposition")
		}
	}
	if hasMinute {
		return clockSpan(s, toks, i, j, hour, minute, "time:minutes")
	}
	return contracts.Span{}, i, false
}

func clockSpan(s string, toks []token, i, j, hour, minute int, rule string) (contracts.Span, int, bool) {
	if hour == 24 {
		hour = 0
	}
	start, end := toks[i].start, toks[j-1].end
	return contracts.Span{
		Kind: contracts.ValueTime, Raw: s[start:end], Start: start, End: end,
		Value:      contracts.Value{Kind: contracts.ValueTime, Text: fmt.Sprintf("%02d:%02d", hour, minute)},
		Confidence: 1, Rule: rule,
	}, j, true
}

// previousWords trả tối đa n từ đứng trước token i, chữ thường, cách nhau một dấu cách.
func previousWords(s string, toks []token, i, n int) string {
	var words []string
	for k := i - 1; k >= 0 && len(words) < n; k-- {
		if toks[k].kind == tokWord {
			words = append([]string{strings.ToLower(toks[k].text)}, words...)
			continue
		}
		if toks[k].kind == tokSpace {
			continue
		}
		break
	}
	return strings.Join(words, " ")
}

// tokenAt trả chỉ số token đầu tiên bắt đầu tại hoặc sau vị trí byte b.
func tokenAt(toks []token, b int) int {
	for k, t := range toks {
		if t.start >= b {
			return k
		}
	}
	return len(toks)
}
