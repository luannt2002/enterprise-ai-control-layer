package normalization

import (
	"slices"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/luannt2002/enterprise-ai-systems/core/contracts"
)

// Con số đứng cạnh một từ chỉ kỳ hay một danh từ nhãn mà bộ chuẩn hoá KHÔNG đọc thành giá trị: "tuần 43", "quý 3", "năm 2025", "ngày 12",
// "7 ngày qua" là mốc thời gian viết bằng chữ; "phiếu 912", "ca 2", "2 máy" là nhãn / số đếm. Luồng hỏi cần phân biệt hai loại khi câu
// xác nhận một lượng ("… 3850 L đúng không?"): nhãn là bối cảnh vô hại, còn mốc thời gian chưa đọc được thì kỳ của câu có thể KHÁC kỳ người hỏi nói
// — không được phán quyết "Đúng." / "Không." cho kỳ khác (P5).

// QualifierKind là loại mốc thời gian mà một con số đứng cạnh từ chỉ kỳ nói tới.
type QualifierKind string

const (
	QualifierDay     QualifierKind = "day"     // "ngày 12", "hôm 12", "mùng 5", và "19" của "19 tháng 10"
	QualifierWeek    QualifierKind = "week"    // "tuần 43"
	QualifierMonth   QualifierKind = "month"   // "tháng 13" (tháng 1–12 đã là span kỳ, không tới đây)
	QualifierQuarter QualifierKind = "quarter" // "quý 3"
	QualifierYear    QualifierKind = "year"    // "năm 2025"
	QualifierSpan    QualifierKind = "span"    // "7 ngày qua", "2 tuần", "3 tháng": một khoảng đếm bằng số — không kiểm được với kỳ khác
)

// Qualifier là một mốc thời gian viết bằng chữ mà bộ chuẩn hoá không đọc thành kỳ.
type Qualifier struct {
	Kind  QualifierKind
	Value int    // 12 của "ngày 12", 43 của "tuần 43", 2025 của "năm 2025", 7 của "7 ngày"
	Raw   string // chữ người hỏi viết ("ngày 12", "7 ngày"), để nói rõ mốc nào chưa đọc được
	// Word là khoảng byte [đầu, cuối) của TỪ CHỈ KỲ trong raw ("tuần" của "tuần 43", "ngày" của "7 ngày"). Từ đó nằm trong một kỳ bộ chuẩn hoá đã đọc
	// ("3850 tuần này": "tuần" của "tuần này") thì con số không đếm nó — nó không phải mốc (xem luồng hỏi).
	Word [2]int
}

// periodWords: từ chỉ kỳ → loại mốc khi đứng NGAY TRƯỚC con số ("tuần 43").
var periodWords = []struct {
	word string
	kind QualifierKind
}{
	{"ngày", QualifierDay}, {"hôm", QualifierDay}, {"mùng", QualifierDay}, {"mồng", QualifierDay},
	{"tuần", QualifierWeek}, {"tháng", QualifierMonth}, {"quý", QualifierQuarter}, {"năm", QualifierYear},
}

// spanWords: từ đứng NGAY SAU con số thì số đó đếm một khoảng thời gian ("7 ngày qua", "2 tuần", "3 tháng gần đây").
var spanWords = []string{"ngày", "tuần", "tháng", "quý", "năm"}

// PeriodQualifier báo con số ở [start, end) của raw (số nguyên không dấu) là phần của một mốc thời gian viết bằng chữ: đứng NGAY SAU một từ chỉ kỳ
// ("ngày 12", "hôm 12", "mùng 5", "tuần 43", "quý 3", "năm 2025"), hoặc NGAY TRƯỚC một từ chỉ kỳ ("7 ngày qua", "2 tuần"; "19" của "19 tháng 10"
// là ngày trong tháng). Từ chỉ tốc độ đứng trước từ chỉ kỳ ("mỗi ngày 3 chuyến", "hàng tuần 2 lần", "cứ 2 ngày một lần") thì đó là tần suất,
// không phải mốc. false khi số không phải số nguyên không dấu (có dấu nghìn / thập phân) hay không đứng cạnh từ chỉ kỳ nào.
func PeriodQualifier(raw string, start, end int) (Qualifier, bool) {
	if start < 0 || end > len(raw) || start >= end {
		return Qualifier{}, false
	}
	digits := raw[start:end]
	if len(digits) > 4 || strings.Trim(digits, "0123456789") != "" {
		return Qualifier{}, false
	}
	value, _ := strconv.Atoi(digits)
	// từ chỉ kỳ đứng ngay trước: "ngày 12"
	if ws, we := wordBefore(raw, start); we > ws {
		for _, pw := range periodWords {
			if wordIs(raw, ws, we, pw.word) {
				if rateWordAt(raw, ws) { // "mỗi ngày 3 chuyến", "hàng tuần 2 lần"
					return Qualifier{}, false
				}
				return Qualifier{Kind: pw.kind, Value: value, Raw: raw[ws:end], Word: [2]int{ws, we}}, true
			}
		}
		// "tuần thứ 41", "ngày số 12": từ chỉ kỳ, rồi "thứ" / "số", rồi con số — vẫn là một mốc (soát R, R-A02)
		if wordIs(raw, ws, we, "thứ") || wordIs(raw, ws, we, "số") {
			if ps, pe := wordBefore(raw, ws); pe > ps && !rateWordAt(raw, ps) {
				for _, pw := range periodWords {
					if wordIs(raw, ps, pe, pw.word) {
						return Qualifier{Kind: pw.kind, Value: value, Raw: raw[ps:end], Word: [2]int{ps, pe}}, true
					}
				}
			}
		}
	}
	// từ chỉ kỳ đứng ngay sau: "7 ngày qua", "19 tháng 10"
	if ws, we := wordAfter(raw, end); we > ws {
		for _, w := range spanWords {
			if !wordIs(raw, ws, we, w) {
				continue
			}
			if rateWordAt(raw, start) {
				return Qualifier{}, false // "mỗi 2 ngày", "cứ 2 ngày một lần"
			}
			ns, ne := numberAfter(raw, we)
			if w == "tháng" && ne > ns {
				// "19 tháng 10": số đứng trước "tháng" kèm một số là ngày trong tháng (chỉ khi là ngày 1–31: "912 tháng 9" là số phiếu, rồi tháng 9)
				if value < 1 || value > 31 {
					return Qualifier{}, false
				}
				return Qualifier{Kind: QualifierDay, Value: value, Raw: raw[start:ne], Word: [2]int{ws, we}}, true
			}
			if ne > ns {
				return Qualifier{}, false // "2 ngày 15/09": "ngày" mở đầu một mốc khác, "2" không đếm nó
			}
			if value > 999 {
				return Qualifier{}, false // một khoảng đếm ngày / tuần / tháng không tới hàng nghìn: "3850 tuần" là con số theo sau một mốc ("3850 tuần này")
			}
			return Qualifier{Kind: QualifierSpan, Value: value, Raw: raw[start:we], Word: [2]int{ws, we}}, true
		}
	}
	return Qualifier{}, false
}

// rateWordAt báo từ đứng NGAY TRƯỚC vị trí pos là một từ chỉ tốc độ / tần suất (rateWords: "mỗi", "hàng", "cứ"…).
func rateWordAt(raw string, pos int) bool {
	ws, we := wordBefore(raw, pos)
	return we > ws && slices.ContainsFunc(rateWords, func(w string) bool { return wordIs(raw, ws, we, w) })
}

// labelBefore: danh từ nhãn đứng NGAY TRƯỚC con số ("phiếu 912", "ca 2", "đội 3", "xe số 7", "khối 2", "L 5") — con số là mã / số thứ tự của nhãn, không phải
// lượng. Không có "xe", "chuyến", "lượt": chúng cũng là danh từ ĐẾM của số đứng trước ("mỗi ngày 3-4 chuyến 4000 L" — 4000 không phải nhãn của "chuyến"); không có
// "kho": "xuất kho 3850", "phiếu kho 3850" là từ khoá số liệu của pack đứng trước CON SỐ được hỏi.
var labelBefore = []string{"phiếu", "số", "mã", "ca", "đội", "tổ", "lô", "đợt", "thứ", "khu", "trạm", "khối", "lít", "l"}

// unitLabels: các từ của labelBefore cũng là ĐƠN VỊ ("khối 2", "L 5" là nhãn; nhưng "3850 L 19 tháng 10": L là đơn vị của 3850, không phải nhãn của 19).
var unitLabels = []string{"khối", "lít", "l"}

// digitBefore báo ngay trước vị trí pos (chỉ cách khoảng đệm) là một chữ số.
func digitBefore(raw string, pos int) bool {
	p := pos
	for p > 0 {
		r, w := utf8.DecodeLastRuneInString(raw[:p])
		if r != ' ' && !isFormat(r) {
			return unicode.IsDigit(r)
		}
		p -= w
	}
	return false
}

// countAfter: danh từ đứng NGAY SAU con số ("2 máy", "3 lần", "1 xe", "3-4 chuyến") — con số đếm các thứ đó, không phải lượng.
var countAfter = []string{"lần", "máy", "xe", "chuyến", "đội", "tổ", "đợt", "lượt", "phiếu", "phút", "giây", "thiết bị", "dự án", "vị trí"}

// LabelContext báo con số ở [start, end) của raw là một NHÃN, SỐ ĐẾM hay TẦN SUẤT vô hại cho câu xác nhận một lượng: đứng ngay sau một danh từ nhãn
// ("phiếu 912", "ca 2", "xe số 7"), ngay trước một danh từ đếm ("2 máy", "3 lần", "3-4 chuyến": phần "-N" của một khoảng được bỏ qua), hoặc đứng sau
// "mỗi ngày", "hàng tuần"… ("mỗi ngày 3 chuyến"). Các danh sách đóng và ngắn: thiếu một từ thì con số đó vẫn được coi là một con số xác nhận (hướng an
// toàn — luồng hỏi lại, không phán quyết nhầm).
func LabelContext(raw string, start, end int) bool {
	if start < 0 || end > len(raw) || start >= end {
		return false
	}
	if ws, we := wordBefore(raw, start); we > ws {
		if slices.ContainsFunc(labelBefore, func(w string) bool { return wordIs(raw, ws, we, w) }) &&
			!(slices.ContainsFunc(unitLabels, func(w string) bool { return wordIs(raw, ws, we, w) }) && digitBefore(raw, ws)) {
			return true
		}
		// "mỗi ngày 3", "hàng tuần 2": số đứng sau một từ chỉ kỳ mà trước nó là từ chỉ tần suất là số lần, không phải mốc
		for _, pw := range periodWords {
			if wordIs(raw, ws, we, pw.word) && rateWordAt(raw, ws) {
				return true
			}
		}
	}
	rest := strings.TrimLeft(raw[end:], " ")
	if r, w := utf8.DecodeRuneInString(rest); r == '-' || r == '–' { // "3-4 chuyến": bỏ qua phần "-4" của khoảng
		if ns, ne := numberAfter(rest, w); ne > ns {
			rest = strings.TrimLeft(rest[ne:], " ")
		}
	}
	for _, w := range countAfter {
		if _, ok := matchAliasAt(rest, 0, w, false); ok {
			return true
		}
	}
	return false
}

// wordIs báo đoạn [ws, we) của raw đúng là từ word, theo cùng luật dấu với mọi so khớp (gõ không dấu vẫn khớp; có dấu thì phải đúng dấu).
func wordIs(raw string, ws, we int, word string) bool {
	end, ok := matchAliasAt(raw, ws, word, false)
	return ok && end == we
}

// wordBefore trả [đầu, cuối) của CHỮ đứng ngay trước vị trí pos (chỉ cách khoảng đệm); rỗng (0, 0) khi trước pos không phải một chữ.
func wordBefore(raw string, pos int) (ws, we int) {
	p := pos
	for p > 0 {
		r, w := utf8.DecodeLastRuneInString(raw[:p])
		if r != ' ' && !isFormat(r) {
			break
		}
		p -= w
	}
	we = p
	for p > 0 {
		r, w := utf8.DecodeLastRuneInString(raw[:p])
		if !unicode.IsLetter(r) && !unicode.Is(unicode.Mn, r) {
			break
		}
		p -= w
	}
	if p == we {
		return 0, 0
	}
	return p, we
}

// wordAfter trả [đầu, cuối) của CHỮ đứng ngay sau vị trí pos (chỉ cách khoảng đệm); rỗng khi sau pos không phải một chữ.
func wordAfter(raw string, pos int) (ws, we int) {
	ws = skipGap(raw, pos)
	we = ws
	for we < len(raw) {
		r, w := utf8.DecodeRuneInString(raw[we:])
		if !unicode.IsLetter(r) && !unicode.Is(unicode.Mn, r) {
			break
		}
		we += w
	}
	if we == ws {
		return 0, 0
	}
	return ws, we
}

// numberAfter trả [đầu, cuối) của dãy chữ số đứng ngay sau vị trí pos (chỉ cách khoảng đệm); rỗng khi sau pos không phải một chữ số.
func numberAfter(raw string, pos int) (ns, ne int) {
	ns = skipGap(raw, pos)
	ne = ns
	for ne < len(raw) && raw[ne] >= '0' && raw[ne] <= '9' {
		ne++
	}
	if ne == ns {
		return 0, 0
	}
	return ns, ne
}

// DashDateAt báo các span số trần bắt đầu từ spans[i] là MỘT NGÀY viết dd-mm hoặc dd-mm-yy (nối bằng "-" / "–", không khoảng trắng) mà bộ chuẩn hoá không
// đọc vì thiếu chữ "ngày" ("19-10", "19-10-26": "10-12 giờ", "3-4 chuyến" cũng có dạng đó nên không đọc im lặng): trả chữ của cả cụm và số span nó chiếm.
// Không phải khi dính vào mã đứng trước ("TK-01-02"), là phần giữa hay phần đầu của một dãy dài hơn ba phần ("19-10-2026-1"), khi ngày > 31 hay tháng > 12, hoặc khi số cuối đứng trước một
// danh từ đếm ("10-12 xe": số đếm, không phải ngày).
func DashDateAt(raw string, spans []contracts.Span, i int) (text string, n int) {
	first := spans[i]
	if first.Kind != contracts.ValueNumber || !smallNumberIn(first.Raw, 1, 31) || gluedBefore(raw, first.Start) {
		return "", 0
	}
	if prev, _ := utf8.DecodeLastRuneInString(raw[:first.Start]); prev == '-' || prev == '–' || prev == '/' {
		return "", 0 // phần giữa của một dãy ("01-02-03"): cụm đứng đầu dãy mới là thứ cần xét
	}
	last, parts := first, 1
	for k := i + 1; k < len(spans) && parts < 3; k++ {
		nx := spans[k]
		if nx.Kind != contracts.ValueNumber || last.End > nx.Start || (raw[last.End:nx.Start] != "-" && raw[last.End:nx.Start] != "–") {
			break
		}
		if (parts == 1 && !smallNumberIn(nx.Raw, 1, 12)) || (parts == 2 && len(nx.Raw) != 2 && len(nx.Raw) != 4) {
			return "", 0
		}
		last, parts = nx, parts+1
	}
	// số ĐẦU là số hiệu sau từ nhãn ("phiếu 12-10", "ca 2-3"): không phải một ngày (soát R, R-A14)
	if parts < 2 || LabelContext(raw, last.Start, last.End) || LabelContext(raw, first.Start, first.End) || isDigitAt(raw, last.End) {
		return "", 0
	}
	if next, w := utf8.DecodeRuneInString(raw[last.End:]); (next == '-' || next == '–') && isDigitAt(raw, last.End+w) {
		return "", 0 // dãy dài hơn ba phần ("19-10-2026-1"): không phải một ngày
	}
	return raw[first.Start:last.End], parts
}

// smallNumberIn báo p là số nguyên không dấu trong [lo, hi].
func smallNumberIn(p string, lo, hi int) bool {
	if p == "" || len(p) > 2 || strings.Trim(p, "0123456789") != "" {
		return false
	}
	v, _ := strconv.Atoi(p)
	return v >= lo && v <= hi
}
