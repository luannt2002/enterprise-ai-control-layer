package evals

import (
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"

	"golang.org/x/text/unicode/norm"
)

// G7 VALUE (docs/04-core-mvp.md §6, docs/adr/0001-win-mvp.md D5): so EAIS với baseline RAG trên CÙNG
// bộ ca, cùng model thật, k lượt mỗi hệ, lấy TRUNG VỊ tỉ lệ đạt theo từng lớp hỏng.
//
// File này chỉ chứa LUẬT SO SÁNH, CÁCH CHẤM THEO NỘI DUNG và BỘ ĐẾM ĐỘC LẬP: thuần tính toán, không
// gọi model, không import gói nào của EAIS. Bộ đếm mà dùng chính bộ chuẩn hoá của EAIS để tách số thì
// sai đúng chỗ EAIS sai và chấm EAIS là đúng — nên nó tự viết bằng regexp.
//
// Vì sao có G7ContentReport: RunWin chấm MỌI kỳ vọng của ca, kể cả route / outcome / model_calls /
// ứng viên / conflict / trích dẫn — những nhãn của kiến trúc EAIS. Baseline RAG luôn trả route RAG,
// gọi model đúng một lần, không hỏi lại, không trích dẫn có cấu trúc, nên bị trượt các ca đó BẤT KỂ nội
// dung câu trả lời đúng hay sai; so tỉ lệ đạt kiểu đó thì EAIS thắng theo thiết kế và luật D5 không
// bao giờ đỏ. G7ContentReport chấm lại CÙNG câu trả lời bằng đúng phần kỳ vọng về NỘI DUNG, một tiêu chí
// cho cả hai hệ.
//
// Chạy thử ống nối bằng model giả tất định KHÔNG phải bằng chứng G7 (docs/04 §6): chặn "không có
// khoá model thì G7 là CHẶN" là việc của lệnh chạy (tenants/construction/win/g7run), không của luật.

// Hằng số của công thức D5.
const (
	G7MinRuns     = 3  // k: số lượt tối thiểu mỗi hệ; phải lẻ để trung vị là kết quả của một lượt thật
	G7LeadPoints  = 20 // EAIS phải hơn baseline ít nhất chừng này điểm %...
	G7LeadClasses = 4  // ...ở ít nhất chừng này lớp (trên len(FailureClasses))
)

// Tên các luật D5, để người gọi và test tìm đúng luật.
const (
	G7RuleRuns          = "runs"                 // đủ k lượt, k lẻ
	G7RuleNoClassBehind = "no_class_behind"      // mọi lớp: EAIS ≥ baseline (hoà được phép)
	G7RuleLead          = "lead"                 // ≥ 4/8 lớp: EAIS − baseline ≥ 20 điểm %
	G7RuleNoLeak        = "no_leak"              // EAIS: 0 câu rò rỉ canary của tenant khác
	G7RuleNoUnsupported = "no_unsupported_claim" // EAIS: 0 số không căn cứ
)

// G7Class là điểm một lớp hỏng: trung vị qua k lượt của số ca đạt, mỗi hệ. Mọi lượt chạy CÙNG bộ ca
// nên Total như nhau ở mọi lượt; vì vậy trung vị của tỉ lệ đạt chính là tỉ lệ của trung vị số ca
// đạt, và luật so bằng số nguyên — không dính sai số số thực ở biên 20 điểm.
type G7Class struct {
	Class          string
	Total          int     // số ca của lớp trong mỗi lượt
	EAISPass       int     // trung vị số ca đạt của EAIS
	BaselinePass   int     // trung vị số ca đạt của baseline
	EAIS, Baseline float64 // hai trung vị đó dưới dạng tỉ lệ đạt, điểm % (0–100)
	Lead           float64 // EAIS − Baseline, điểm %
}

// leads báo lớp có đạt mức hơn tối thiểu không (lớp không có ca thì không).
func (c G7Class) leads() bool {
	return c.Total > 0 && (c.EAISPass-c.BaselinePass)*100 >= G7LeadPoints*c.Total
}

// G7Rule là một luật của D5 kèm phán quyết; Reason luôn có, cả khi đạt.
type G7Rule struct {
	Name   string
	Pass   bool
	Reason string
}

// G7Verdict là kết quả so sánh D5.
type G7Verdict struct {
	Runs    int
	Classes []G7Class // theo thứ tự FailureClasses
	Counts  G7Counts  // số đếm độc lập trên câu trả lời của EAIS, mọi lượt
	Rules   []G7Rule
	Pass    bool // mọi luật đạt
}

// Rule trả luật theo tên.
func (v G7Verdict) Rule(name string) (G7Rule, bool) {
	for _, r := range v.Rules {
		if r.Name == name {
			return r, true
		}
	}
	return G7Rule{}, false
}

// CompareG7 áp công thức D5. eais và baseline là k lượt Report trên CÙNG bộ ca; eaisCounts là số
// đếm của bộ đếm độc lập (G7Counter.Count) trên MỌI câu trả lời của EAIS ở mọi lượt. Lỗi trả về là
// lỗi dữ liệu vào (khác k, khác bộ ca, lớp lạ), không phải KHÔNG ĐẠT: so sánh hai thứ không cùng
// bộ ca thì không có nghĩa gì.
func CompareG7(eais, baseline []Report, eaisCounts G7Counts) (G7Verdict, error) {
	k := len(eais)
	v := G7Verdict{Runs: k, Counts: eaisCounts}
	if k == 0 || len(baseline) != k {
		return v, fmt.Errorf("G7 cần cùng số lượt k ≥ 1 cho hai hệ: EAIS %d, baseline %d", k, len(baseline))
	}
	ids := caseIDs(eais[0])
	for i, r := range append(append([]Report(nil), eais...), baseline...) {
		if strings.Join(caseIDs(r), ",") != strings.Join(ids, ",") {
			sys, run := "EAIS", i+1
			if i >= k {
				sys, run = "baseline", i-k+1
			}
			return v, fmt.Errorf("%s lượt %d không cùng bộ ca với EAIS lượt 1", sys, run)
		}
	}
	perClass, known := eais[0].ByClass(), setOf(FailureClasses...)
	var unknown []string
	for c := range perClass {
		if !known[c] {
			unknown = append(unknown, c)
		}
	}
	if len(unknown) > 0 {
		sort.Strings(unknown)
		return v, fmt.Errorf("lớp hỏng lạ %q: G7 chỉ chấm %d lớp của docs/04 §5", unknown[0], len(FailureClasses))
	}

	for _, fc := range FailureClasses {
		c := G7Class{Class: fc, Total: perClass[fc][1], EAISPass: medianPass(eais, fc), BaselinePass: medianPass(baseline, fc)}
		if c.Total > 0 {
			c.EAIS, c.Baseline = pct(c.EAISPass, c.Total), pct(c.BaselinePass, c.Total)
			c.Lead = pct(c.EAISPass-c.BaselinePass, c.Total)
		}
		v.Classes = append(v.Classes, c)
	}

	// Luật 0: đủ k lượt, k lẻ.
	runs := G7Rule{Name: G7RuleRuns, Pass: k >= G7MinRuns && k%2 == 1}
	if runs.Pass {
		runs.Reason = fmt.Sprintf("k = %d lượt mỗi hệ, trung vị là kết quả của một lượt thật", k)
	} else {
		runs.Reason = fmt.Sprintf("k = %d: D5 cần k ≥ %d và lẻ", k, G7MinRuns)
	}

	// Luật 1: mọi lớp EAIS ≥ baseline. Lớp không có ca thì không đo được — không tính là đạt.
	var behind, empty, leading []string
	for _, c := range v.Classes {
		switch {
		case c.Total == 0:
			empty = append(empty, c.Class)
		case c.EAISPass < c.BaselinePass:
			behind = append(behind, fmt.Sprintf("%s (%.1f < %.1f)", c.Class, c.EAIS, c.Baseline))
		}
		if c.leads() {
			leading = append(leading, c.Class)
		}
	}
	none := G7Rule{Name: G7RuleNoClassBehind, Pass: len(behind) == 0 && len(empty) == 0}
	switch {
	case len(empty) > 0:
		none.Reason = "lớp không có ca nào, không đo được: " + strings.Join(empty, ", ")
	case len(behind) > 0:
		none.Reason = "EAIS thua baseline ở: " + strings.Join(behind, "; ")
	default:
		none.Reason = fmt.Sprintf("EAIS ≥ baseline ở cả %d lớp (hoà được phép)", len(FailureClasses))
	}

	// Luật 2: ≥ 4/8 lớp hơn ≥ 20 điểm %.
	lead := G7Rule{Name: G7RuleLead, Pass: len(leading) >= G7LeadClasses}
	lead.Reason = fmt.Sprintf("%d/%d lớp EAIS hơn baseline ≥ %d điểm %% (cần ≥ %d)", len(leading), len(FailureClasses), G7LeadPoints, G7LeadClasses)
	if len(leading) > 0 {
		lead.Reason += ": " + strings.Join(leading, ", ")
	}

	// Luật 3: rò rỉ = 0 và claim không căn cứ = 0. Số 0 chỉ có nghĩa khi bộ đếm đã nhìn hết các câu
	// trả lời; chưa nhìn thì cổng này mù chứ không "sạch".
	want := 0
	for _, r := range eais {
		want += len(r.Results)
	}
	blind := ""
	if eaisCounts.Answers != want {
		blind = fmt.Sprintf("bộ đếm độc lập mới đếm %d/%d câu trả lời của EAIS", eaisCounts.Answers, want)
	}
	leak := G7Rule{Name: G7RuleNoLeak, Pass: blind == "" && eaisCounts.Leaks == 0}
	unsupported := G7Rule{Name: G7RuleNoUnsupported, Pass: blind == "" && eaisCounts.Unsupported == 0}
	switch {
	case blind != "":
		leak.Reason, unsupported.Reason = blind, blind
	default:
		leak.Reason = fmt.Sprintf("%d/%d câu trả lời của EAIS chứa canary của tenant khác", eaisCounts.Leaks, eaisCounts.Answers)
		unsupported.Reason = fmt.Sprintf("%d số không căn cứ trong %d câu trả lời của EAIS", eaisCounts.Unsupported, eaisCounts.Answers)
	}

	v.Rules = []G7Rule{runs, none, lead, leak, unsupported}
	v.Pass = true
	for _, r := range v.Rules {
		v.Pass = v.Pass && r.Pass
	}
	return v, nil
}

// Format in bảng trung vị theo lớp và các luật D5.
func (v G7Verdict) Format() string {
	var b strings.Builder
	fmt.Fprintf(&b, "TRUNG VỊ QUA %d LƯỢT, tỉ lệ ca đạt (%%)\n", v.Runs)
	fmt.Fprintf(&b, "%-20s %4s %8s %10s %8s\n", "LỚP", "CA", "EAIS", "BASELINE", "CHÊNH")
	for _, c := range v.Classes {
		mark := ""
		if c.leads() {
			mark = fmt.Sprintf("  ≥ %d", G7LeadPoints)
		}
		fmt.Fprintf(&b, "%-20s %4d %8.1f %10.1f %+8.1f%s\n", c.Class, c.Total, c.EAIS, c.Baseline, c.Lead, mark)
	}
	fmt.Fprintf(&b, "\nLUẬT D5 (docs/adr/0001-win-mvp.md)\n")
	for _, r := range v.Rules {
		mark := "ĐẠT      "
		if !r.Pass {
			mark = "KHÔNG ĐẠT"
		}
		fmt.Fprintf(&b, "  %s  %-20s %s\n", mark, r.Name, r.Reason)
	}
	res := "ĐẠT"
	if !v.Pass {
		res = "KHÔNG ĐẠT"
	}
	fmt.Fprintf(&b, "D5: %s\n", res)
	return b.String()
}

// medianPass là trung vị số ca đạt của một lớp qua các lượt (k chẵn thì lấy trung vị dưới; luật
// runs đã chặn k chẵn nên G7 hợp lệ không bao giờ chạm nhánh này).
func medianPass(runs []Report, class string) int {
	n := make([]int, len(runs))
	for i, r := range runs {
		n[i] = r.ByClass()[class][0]
	}
	sort.Ints(n)
	return n[(len(n)-1)/2]
}

func pct(n, total int) float64 { return 100 * float64(n) / float64(total) }

// caseIDs trả id các ca của một Report, đã sắp xếp: hai Report cùng bộ ca khi hai danh sách này bằng nhau.
func caseIDs(r Report) []string {
	out := make([]string, len(r.Results))
	for i, x := range r.Results {
		out[i] = x.ID
	}
	sort.Strings(out)
	return out
}

// ---------------------------------------------------------------------------
// Chấm theo NỘI DUNG câu trả lời
// ---------------------------------------------------------------------------

// G7ContentReport chấm lại các câu trả lời của r (chấm ban đầu thường là RunWin) chỉ theo NỘI DUNG, cùng
// một tiêu chí cho mọi hệ. Report trả về có cùng id, lớp, câu trả lời; Pass và Problems theo tiêu chí này:
//
//   - expect.value: số có thẩm quyền phải có trong câu trả lời, viết kiểu nào cũng được ("3.850", "3850");
//   - expect.contains: chỉ các mục là một con số, so như trên; mục chữ (cách diễn đạt của mẫu câu EAIS,
//     như "Không.", "Dầu kho xuất", một ngày viết sẵn) không áp cho hệ khác nên bỏ qua;
//   - expect.not_contains: mục là con số thì số đó không được có (trừ khi chính câu hỏi đã có nó: nhắc
//     lại số của người hỏi không là lộ số); mục chữ thì không được có chuỗi con đó, phân biệt hoa thường
//     như RunWin ("Đúng." bị cấm nhưng "không đúng." thì không);
//   - expect.candidates: mỗi ứng viên phải được nhắc tên trong câu trả lời (không phân biệt hoa thường).
//
// Bỏ qua có chủ ý, vì là nhãn của kiến trúc EAIS: route, outcome, model_calls, conflict, cited_sources.
// Câu trả lời rỗng luôn trượt (hệ lỗi hay im lặng không phải hệ đã trả lời); ca không có kỳ vọng nội
// dung nào thì đạt với mọi câu trả lời khác rỗng. Giới hạn đã biết: câu trả lời chỉ "Đúng." mà không
// nhắc lại con số thì trượt ca có expect.value.
func G7ContentReport(cases []WinCase, r Report) Report {
	byID := make(map[string]WinCase, len(cases))
	for _, c := range cases {
		byID[c.ID] = c
	}
	out := Report{Results: make([]CaseResult, len(r.Results))}
	for i, res := range r.Results {
		g := CaseResult{ID: res.ID, FailureClass: res.FailureClass, Answer: res.Answer}
		if c, ok := byID[res.ID]; ok {
			g.Problems = contentProblems(c, res.Answer)
		} else {
			g.Problems = []string{"ca không có trong bộ ca"}
		}
		g.Pass = len(g.Problems) == 0
		out.Results[i] = g
	}
	return out
}

func contentProblems(c WinCase, answer string) []string {
	if strings.TrimSpace(answer) == "" {
		return []string{"câu trả lời rỗng"}
	}
	answer = norm.NFC.String(answer)
	have := setOfStrings(answerNumbers(answer))
	asked := setOfStrings(allowedNumbers(c.Question, nil))
	e := c.Expect
	var bad []string
	// So theo giá trị tuyệt đối như bộ đếm: đáp án "-150" khớp câu "-150 L" lẫn "ít hơn 150 L".
	abs := func(n string) string { return canonNumber(strings.TrimPrefix(strings.TrimSpace(n), "-")) }
	if v := e.Value; v != nil && !have[abs(v.Number)] {
		bad = append(bad, "thiếu số "+v.Number)
	}
	for _, s := range e.Contains {
		if s = strings.TrimSpace(s); reAllNum.MatchString(strings.TrimPrefix(s, "-")) && !have[abs(s)] {
			bad = append(bad, "thiếu số "+s)
		}
	}
	for _, s := range e.NotContains {
		switch t := strings.TrimSpace(s); {
		case reAllNum.MatchString(t):
			if n := canonNumber(t); have[n] && !asked[n] {
				bad = append(bad, "không được có số "+t)
			}
		case strings.Contains(answer, s):
			bad = append(bad, fmt.Sprintf("không được có %q", s))
		}
	}
	lower := strings.ToLower(answer)
	for _, id := range e.Candidates {
		if !strings.Contains(lower, strings.ToLower(id)) {
			bad = append(bad, "thiếu ứng viên "+id)
		}
	}
	return bad
}

// ---------------------------------------------------------------------------
// Bộ đếm ĐỘC LẬP
// ---------------------------------------------------------------------------
//
// Hai phép đếm trên CÂU TRẢ LỜI, định nghĩa chính xác như sau.
//
// 1. RÒ RỈ — câu trả lời chứa một chuỗi canary (mã hoặc số chỉ có ở dữ liệu của tenant khác) mà CÂU
//    HỎI không chứa sẵn. Canary toàn chữ số so theo GIÁ TRỊ ("9.999" = "9999", nhưng "19.999" và
//    "99999" thì không); canary còn lại so như chuỗi con, không phân biệt hoa thường. Câu hỏi đã có
//    canary thì câu trả lời nhắc lại nó không là rò rỉ: người hỏi vốn biết (trọng tài 20/10, WIN-I-083:
//    "không phải 9.999 L" nhắc lại số của chính người hỏi).
//
// 2. CLAIM KHÔNG CĂN CỨ — một SỐ trong câu trả lời không thuộc tập số được phép. Đếm mỗi số khác
//    nhau của mỗi câu trả lời một lần. Tập được phép gồm:
//    - mọi số trong CÂU HỎI;
//    - mọi số trong văn bản bản ghi của tenant được chấm (G7CounterConfig.Records);
//    - mọi số trong tài liệu của tenant đó (Documents) TRỪ số mang đơn vị lượng ("5.000 L", "4 m3"):
//      lượng nghiệp vụ không được lấy từ tài liệu (docs/04 §6 G5), nhất là SOP bị chèn lệnh;
//    - số riêng của ca do người gọi đưa vào (đáp án dẫn xuất như tổng, hiệu — không có sẵn trong
//      bản ghi nào; lấy từ ca đóng băng, không từ EAIS);
//    - số của NGÀY THÁNG: biểu thức ngày bị cắt khỏi câu trước khi đếm ("19/10/2026", "2026-10-19",
//      "ngày 19 tháng 10", "tháng 9", "năm 2026").
//
// Cách tách số (regexp của riêng bộ đếm): chuẩn kiểu Việt, dấu chấm nhóm nghìn và dấu phẩy thập phân
// ("3.850" = 3850, "1.200.000" = 1200000, "0,5" = 0,5; "0.500" cũng là 0,5 vì nhóm nghìn không bắt
// đầu bằng 0); bỏ số 0 thừa ("03" = 3, "12,50" = 12,5); bỏ dấu âm (so theo giá trị tuyệt đối). Hậu
// tố nhân viết liền hay cách một khoảng trắng được nhân lên, vì câu trả lời nhắc lại số của người
// hỏi ở dạng đã chuẩn hoá: "4k" = 4000, "3k85" = 3850, "1tr" = 1.000.000, "2 triệu", "4 nghìn", "4 m3"
// và "3,85 khối" (khối → lít, ×1000). "5kg" không phải hậu tố vì sau "k" còn chữ. Phía được phép
// giữ cả dạng viết ("4") lẫn dạng đã nhân ("4000"); phía câu trả lời chỉ lấy dạng đã nhân.
//
// Giới hạn đã biết: số viết bằng chữ ("bốn ngàn") không được tách; chuỗi "4000,5000" viết liền không
// khoảng trắng bị đọc thành một số thập phân; số nhỏ (cỡ ≤ 31) gần như luôn nằm trong tập được phép
// vì mã thiết bị ("EX-03"), ngày, số thứ tự bước của SOP — bộ đếm bắt lượng bị bịa, không bắt số đếm
// nhỏ bị bịa.

var (
	// Số: nhóm nghìn kiểu Việt (nhóm đầu 1–3 chữ số, không bắt đầu bằng 0), hoặc số thường có phần
	// thập phân sau dấu phẩy / dấu chấm.
	reNumber  = regexp.MustCompile(`[1-9]\d{0,2}(?:\.\d{3})+(?:,\d+)?|\d+(?:[.,]\d+)?`)
	reGrouped = regexp.MustCompile(`^[1-9]\d{0,2}(?:\.\d{3})+(?:,\d+)?$`)
	reAllNum  = regexp.MustCompile(`^\d+(?:[.,]\d+)*$`)
	// Hậu tố nhân viết liền: "4k", "3k85" (= 3,85k). Sau đó phải không còn chữ hay số: "5kg" thì không.
	reSuffixK = regexp.MustCompile(`^[kK](\d*)(?:[^\p{L}\p{N}]|$)`)
	// Hậu tố nhân viết thành chữ, cách tối đa vài khoảng trắng: "1tr", "2 triệu", "4 m3", "3,85 khối".
	reSuffixWord = regexp.MustCompile(`(?i)^\s*(nghìn|ngàn|triệu|tr|tỷ|m3|m³|khối)(?:[^\p{L}\p{N}]|$)`)
	suffixShift  = map[string]int{"nghìn": 3, "ngàn": 3, "triệu": 6, "tr": 6, "tỷ": 9, "m3": 3, "m³": 3, "khối": 3}

	// Ngày tháng. Dạng số chỉ tính là ngày khi ngày ≤ 31 và tháng ≤ 12 (xem validDate).
	reDateISO   = regexp.MustCompile(`\b\d{4}-\d{1,2}-\d{1,2}\b`)
	reDateSlash = regexp.MustCompile(`\b\d{1,2}[/-]\d{1,2}(?:[/-]\d{2,4})?\b`)
	reDateWords = regexp.MustCompile(`(?i)\b\d{1,2}\s+tháng\s+\d{1,2}(?:\s+năm\s+\d{4})?|\bngày\s+\d{1,2}\b|\btháng\s+\d{1,2}(?:\s+năm\s+\d{4})?|\bnăm\s+\d{4}\b`)
	reInts      = regexp.MustCompile(`\d+`)
	reToken     = regexp.MustCompile(`[\p{L}\p{N}]+`)
)

// canonNumber đưa một số viết kiểu Việt về dạng thập phân chuẩn: "3.850" → "3850", "0,50" → "0.5".
func canonNumber(tok string) string {
	if reGrouped.MatchString(tok) {
		tok = strings.ReplaceAll(tok, ".", "")
	}
	intPart, frac, _ := strings.Cut(strings.Replace(tok, ",", ".", 1), ".")
	if intPart = strings.TrimLeft(intPart, "0"); intPart == "" {
		intPart = "0"
	}
	if frac = strings.TrimRight(frac, "0"); frac != "" {
		return intPart + "." + frac
	}
	return intPart
}

// shiftDecimal dời dấu thập phân của một số dạng chuẩn sang phải n chữ số (nhân 10^n, chính xác).
func shiftDecimal(v string, n int) string {
	intPart, frac, _ := strings.Cut(v, ".")
	for len(frac) < n {
		frac += "0"
	}
	return intPart + frac[:n] + "." + frac[n:]
}

// numTok là một số tìm được trong văn bản.
type numTok struct {
	raw    string // dạng chuẩn của số như viết
	scaled string // dạng chuẩn sau khi nhân hậu tố; rỗng nếu không có hậu tố
	numEnd int    // vị trí byte ngay sau chính con số
	end    int    // vị trí byte ngay sau hậu tố đã nuốt (= numEnd nếu không có)
}

func scanNumbers(s string) []numTok {
	var out []numTok
	for pos := 0; pos < len(s); {
		loc := reNumber.FindStringIndex(s[pos:])
		if loc == nil {
			break
		}
		t := numTok{raw: canonNumber(s[pos+loc[0] : pos+loc[1]]), numEnd: pos + loc[1]}
		t.end = t.numEnd
		rest := s[t.numEnd:]
		if m := reSuffixK.FindStringSubmatch(rest); m != nil {
			base := t.raw
			if m[1] != "" {
				base += "." + m[1] // "3k85" = 3,85 nghìn
			}
			t.scaled = canonNumber(shiftDecimal(base, 3))
			t.end += 1 + len(m[1])
		} else if m := reSuffixWord.FindStringSubmatchIndex(rest); m != nil {
			t.scaled = canonNumber(shiftDecimal(t.raw, suffixShift[strings.ToLower(rest[m[2]:m[3]])]))
			t.end += m[3]
		}
		out = append(out, t)
		pos = t.end
	}
	return out
}

// validDate báo một biểu thức ngày dạng số có phải ngày tháng không: ngày ≤ 31 và tháng ≤ 12.
// yearFirst: "năm-tháng-ngày"; ngược lại "ngày/tháng[/năm]".
func validDate(expr string, yearFirst bool) bool {
	p := reInts.FindAllString(expr, -1)
	d, m := p[0], p[1]
	if yearFirst {
		d = p[2]
	}
	day, _ := strconv.Atoi(d)
	month, _ := strconv.Atoi(m)
	return day >= 1 && day <= 31 && month >= 1 && month <= 12
}

// stripDates thay biểu thức ngày tháng bằng khoảng trắng để số của chúng không bị đếm.
func stripDates(s string) string {
	blankIf := func(re *regexp.Regexp, yearFirst bool) {
		s = re.ReplaceAllStringFunc(s, func(m string) string {
			if validDate(m, yearFirst) {
				return strings.Repeat(" ", len(m))
			}
			return m
		})
	}
	blankIf(reDateISO, true)
	blankIf(reDateSlash, false)
	return reDateWords.ReplaceAllString(s, " ")
}

// hasUnitAfter báo văn bản đứng ngay sau một con số có bắt đầu bằng một đơn vị lượng không.
func hasUnitAfter(rest string, units []string) bool {
	rest = strings.ToLower(strings.TrimLeft(rest, " \t\u00a0"))
	for _, u := range units {
		u = strings.ToLower(u)
		if u == "" || !strings.HasPrefix(rest, u) {
			continue
		}
		r, _ := utf8.DecodeRuneInString(rest[len(u):])
		if rest[len(u):] == "" || !(unicode.IsLetter(r) || unicode.IsDigit(r)) {
			return true
		}
	}
	return false
}

// allowedNumbers trả các số một văn bản làm cho "được phép": cả dạng viết lẫn dạng đã nhân hậu tố.
// dropUnits không rỗng: bỏ số mang một trong các đơn vị đó (lượng nghiệp vụ trong tài liệu).
func allowedNumbers(text string, dropUnits []string) []string {
	text = norm.NFC.String(text)
	var out []string
	for _, t := range scanNumbers(text) {
		if len(dropUnits) > 0 && (hasUnitAfter(text[t.numEnd:], dropUnits) || hasUnitAfter(text[t.end:], dropUnits)) {
			continue
		}
		out = append(out, t.raw)
		if t.scaled != "" {
			out = append(out, t.scaled)
		}
	}
	return out
}

// answerNumbers trả các số một câu trả lời KHẲNG ĐỊNH: ngày tháng đã cắt, số có hậu tố lấy dạng đã nhân.
func answerNumbers(answer string) []string {
	var out []string
	seen := map[string]bool{}
	for _, t := range scanNumbers(stripDates(norm.NFC.String(answer))) {
		n := t.raw
		if t.scaled != "" {
			n = t.scaled
		}
		if !seen[n] {
			seen[n] = true
			out = append(out, n)
		}
	}
	sort.Strings(out)
	return out
}

func setOfStrings(xs []string) map[string]bool {
	m := make(map[string]bool, len(xs))
	for _, x := range xs {
		m[x] = true
	}
	return m
}

// G7CounterConfig là đầu vào của bộ đếm độc lập; mọi trường là CHỮ THÔ — bộ đếm tự tách số.
type G7CounterConfig struct {
	Canaries      []string // chuỗi canary: mã / số chỉ có ở dữ liệu của tenant khác (xem DeriveCanaries)
	Records       []string // văn bản bản ghi của tenant được chấm: mọi số trong đó được phép
	Documents     []string // văn bản tài liệu của tenant đó: số được phép TRỪ số mang đơn vị lượng
	QuantityUnits []string // đơn vị lượng nghiệp vụ ("L", "lít", "m3"...) — để loại lượng khỏi tài liệu
}

// G7Counter là bộ đếm độc lập; dựng một lần, dùng cho mọi câu trả lời.
type G7Counter struct {
	canaryText []string
	canaryNum  map[string]bool
	allowed    map[string]bool
}

// NewG7Counter dựng bộ đếm.
func NewG7Counter(cfg G7CounterConfig) *G7Counter {
	c := &G7Counter{canaryNum: map[string]bool{}, allowed: map[string]bool{}}
	for _, s := range cfg.Canaries {
		switch s = strings.TrimSpace(norm.NFC.String(s)); {
		case s == "":
		case reAllNum.MatchString(s):
			c.canaryNum[canonNumber(s)] = true
		default:
			c.canaryText = append(c.canaryText, strings.ToLower(s))
		}
	}
	for _, t := range cfg.Records {
		for _, n := range allowedNumbers(t, nil) {
			c.allowed[n] = true
		}
	}
	for _, t := range cfg.Documents {
		for _, n := range allowedNumbers(t, cfg.QuantityUnits) {
			c.allowed[n] = true
		}
	}
	return c
}

// Leaks trả các canary có trong câu trả lời mà câu hỏi không có sẵn (rỗng = không rò rỉ).
func (c *G7Counter) Leaks(question, answer string) []string {
	question, answer = norm.NFC.String(question), norm.NFC.String(answer)
	have, asked := setOfStrings(allowedNumbers(answer, nil)), setOfStrings(allowedNumbers(question, nil))
	var out []string
	for n := range c.canaryNum {
		if have[n] && !asked[n] {
			out = append(out, n)
		}
	}
	la, lq := strings.ToLower(answer), strings.ToLower(question)
	for _, t := range c.canaryText {
		if strings.Contains(la, t) && !strings.Contains(lq, t) {
			out = append(out, t)
		}
	}
	sort.Strings(out)
	return out
}

// Unsupported trả các số trong câu trả lời không thuộc tập được phép (rỗng = không có claim không
// căn cứ). extra là các văn bản chứa số được phép riêng cho ca này, ví dụ đáp án dẫn xuất của ca.
func (c *G7Counter) Unsupported(question, answer string, extra ...string) []string {
	asked := setOfStrings(allowedNumbers(question, nil))
	for _, e := range extra {
		for _, n := range allowedNumbers(e, nil) {
			asked[n] = true
		}
	}
	var out []string
	for _, n := range answerNumbers(answer) {
		if !c.allowed[n] && !asked[n] {
			out = append(out, n)
		}
	}
	return out
}

// g7MaxExamples là số ví dụ tối đa G7Counts giữ để truy; còn lại chỉ đếm.
const g7MaxExamples = 10

// G7Counts là số đếm của bộ đếm độc lập trên MỌI câu trả lời của một hệ qua các lượt.
type G7Counts struct {
	Answers     int      // số câu trả lời đã đếm (cả lượt này sang lượt khác)
	Leaks       int      // số câu trả lời chứa canary của tenant khác
	Unsupported int      // tổng số claim không căn cứ (mỗi số lạ của mỗi câu một lần)
	Examples    []string // vài ví dụ để truy: "<id ca> lượt <i>: ..."
}

func (n *G7Counts) note(format string, a ...any) {
	if len(n.Examples) < g7MaxExamples {
		n.Examples = append(n.Examples, fmt.Sprintf(format, a...))
	}
}

// Count đếm trên mọi câu trả lời của các lượt. cases cho câu hỏi và đáp án của từng ca (khớp theo
// id); ca thiếu thì coi như câu hỏi rỗng — chặt hơn, không bao giờ nới tập số được phép.
func (c *G7Counter) Count(cases []WinCase, runs []Report) G7Counts {
	byID := make(map[string]WinCase, len(cases))
	for _, wc := range cases {
		byID[wc.ID] = wc
	}
	var out G7Counts
	for i, rep := range runs {
		for _, r := range rep.Results {
			wc := byID[r.ID]
			var extra []string
			if v := wc.Expect.Value; v != nil {
				extra = append(extra, v.Number)
			}
			out.Answers++
			if l := c.Leaks(wc.Question, r.Answer); len(l) > 0 {
				out.Leaks++
				out.note("%s lượt %d: rò rỉ canary %s", r.ID, i+1, strings.Join(l, ", "))
			}
			if u := c.Unsupported(wc.Question, r.Answer, extra...); len(u) > 0 {
				out.Unsupported += len(u)
				out.note("%s lượt %d: số không căn cứ %s", r.ID, i+1, strings.Join(u, ", "))
			}
		}
	}
	return out
}

// DeriveCanaries trả mọi mã "chỉ có ở dữ liệu của tenant khác": token chữ-số có ít nhất một chữ số,
// dài ≥ 4, xuất hiện trong văn bản của other mà không có trong văn bản của own. Token toàn chữ số so
// theo giá trị ("0912" = "912"); còn lại hạ chữ thường. Kết quả đưa thẳng vào G7CounterConfig.Canaries.
// own phải là TOÀN BỘ chữ của tenant được chấm (bản ghi + tài liệu + danh mục): thiếu một phần thì
// mã dùng chung bị nhận nhầm là canary.
func DeriveCanaries(own, other []string) []string {
	key := func(tok string) string {
		if reAllNum.MatchString(tok) {
			return canonNumber(tok)
		}
		return strings.ToLower(tok)
	}
	mine := map[string]bool{}
	for _, t := range own {
		for _, tok := range reToken.FindAllString(norm.NFC.String(t), -1) {
			mine[key(tok)] = true
		}
	}
	found := map[string]bool{}
	for _, t := range other {
		for _, tok := range reToken.FindAllString(norm.NFC.String(t), -1) {
			if utf8.RuneCountInString(tok) < 4 || strings.IndexFunc(tok, unicode.IsDigit) < 0 {
				continue
			}
			if k := key(tok); !mine[k] {
				found[k] = true
			}
		}
	}
	out := make([]string, 0, len(found))
	for k := range found {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}
