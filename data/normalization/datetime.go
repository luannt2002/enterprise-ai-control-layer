package normalization

import (
	"fmt"
	"strconv"
	"strings"
	"time"

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

// matchNumericDate đọc "16/9", "16/09/2026", "16-9-2026", "2025-11-28".
//
// Cụm có DẠNG ngày mà không đọc được thành một ngày trên lịch ("31/2", "32/10", "29/02/2026", "16/9/26") không phải
// ngày: trả ok=false KÈM bad (span của cả cụm, Rule "date:invalid") để Normalize ghi vào InvalidDates. Người hỏi đã
// cố nêu một ngày — bỏ qua im lặng thì câu hỏi rơi về ngày mặc định như thể họ không nêu (P5).
func matchNumericDate(s string, toks []token, i int, now time.Time) (sp contracts.Span, next int, ok bool, bad contracts.Span) {
	// gom dãy num (sym num){1,2} với sym là / hoặc - và không có khoảng trắng
	var parts []string
	var seps []string
	j := i
	for j < len(toks) && toks[j].kind == tokNum {
		parts = append(parts, toks[j].text)
		if j+2 < len(toks) && toks[j+1].kind == tokSym && (toks[j+1].text == "/" || toks[j+1].text == "-") && toks[j+2].kind == tokNum {
			seps = append(seps, toks[j+1].text)
			j += 2
			continue
		}
		j++
		break
	}
	if len(parts) < 2 || len(parts) > 3 {
		return contracts.Span{}, i, false, contracts.Span{}
	}
	for _, p := range parts {
		if strings.ContainsAny(p, ".,") {
			return contracts.Span{}, i, false, contracts.Span{}
		}
	}
	atoi := func(x string) int { v, _ := strconv.Atoi(x); return v }
	var y, m, d int
	switch {
	case len(parts) == 3 && len(parts[0]) == 4: // ISO yyyy-mm-dd
		y, m, d = atoi(parts[0]), atoi(parts[1]), atoi(parts[2])
	case len(parts) == 3 && len(parts[2]) == 4: // dd/mm/yyyy
		d, m, y = atoi(parts[0]), atoi(parts[1]), atoi(parts[2])
	case len(parts) == 2 && seps[0] == "/": // dd/mm — năm hiện tại, lùi một năm nếu ở tương lai
		d, m, y = atoi(parts[0]), atoi(parts[1]), now.Year()
		if time.Date(y, time.Month(m), d, 0, 0, 0, 0, now.Location()).After(day(now)) {
			y--
		}
	default:
		// ba phần nhỏ nối bằng "/" mà không đúng dạng nào ("16/9/26", năm hai chữ số): ngày chưa đọc được. Nối bằng "-"
		// thì không ghi: mã tài liệu kiểu "TK-01-02-03" có dạng đó mà không phải ngày.
		if len(parts) == 3 && seps[0] == "/" && seps[1] == "/" && smallInt(parts[0]) && smallInt(parts[1]) && smallInt(parts[2]) {
			return contracts.Span{}, i, false, badDate(s, toks, i, j)
		}
		return contracts.Span{}, i, false, contracts.Span{}
	}
	t := time.Date(y, time.Month(m), d, 0, 0, 0, 0, now.Location())
	if m < 1 || m > 12 || d < 1 || d > 31 || t.Day() != d {
		if d < 0 || d > 99 || m < 0 || m > 99 { // "205/55", "02/2026": không có dạng ngày, chỉ là hai số
			return contracts.Span{}, i, false, contracts.Span{}
		}
		return contracts.Span{}, i, false, badDate(s, toks, i, j) // 31/2 không phải ngày
	}
	start, end := toks[i].start, toks[j-1].end
	if !isBoundaryAfter(s, end) {
		return contracts.Span{}, i, false, contracts.Span{}
	}
	return contracts.Span{
		Kind: contracts.ValueDate, Raw: s[start:end], Start: start, End: end,
		Value: dateValue(t), Confidence: 1, Rule: "date:numeric",
	}, j, true, contracts.Span{}
}

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
