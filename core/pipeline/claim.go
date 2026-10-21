package pipeline

import (
	"context"
	"slices"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/luannt2002/enterprise-ai-control-layer/core/contracts"
	"github.com/luannt2002/enterprise-ai-control-layer/core/decimal"
	"github.com/luannt2002/enterprise-ai-control-layer/data/normalization"
	"github.com/luannt2002/enterprise-ai-control-layer/data/sourceoftruth"
	"github.com/luannt2002/enterprise-ai-control-layer/intelligence/intent"
)

// verdictGap là CỔNG PHÁN QUYẾT của câu xác nhận (ADR-0009): trả "" khi được nói "Đúng." / "Không." cho con số claimed, ngược lại
// trả chiều của claim chưa giải được. Bất biến: phán quyết ⇒ claim {chủ thể, chỉ số, kỳ, giá trị, đơn vị, chiều} đã giải bằng
// bằng chứng DƯƠNG trong câu và MỌI chữ của câu được giải thích. Cổng đóng khi thiếu: phần chữ không thuộc phần nào đã hiểu — số
// thứ hai, nhãn, mốc chưa đọc, phủ định, so sánh, nguồn / nghĩa khác, từ lạ — làm mất phán quyết, không bao giờ đổi chiều phán quyết.
//
// Phạm vi hỏi là mệnh đề chứa cụm xác nhận và các mệnh đề sau nó (như claimNumbers). Mệnh đề đứng TRƯỚC là ngữ cảnh: được chứa thêm
// lượng tự mang đơn vị, từ khoá của các ý định số liệu, tên nguồn bất kỳ của pack, từ chỉ lượng — sự kiện người hỏi nêu về chỉ số khác —
// nhưng mọi chữ khác vẫn phải được giải thích ("Không tính ERP, …" vẫn mất phán quyết).
//
// subj là chủ thể hệ đã trả lời (kèm các cấp cha); used là các loại nguồn thực sự tạo nên con số trả lời.
func (a *Ask) verdictGap(in contracts.NormalizedInput, names []mention, it contracts.Intent, def sourceoftruth.MetricDef,
	claimed contracts.Value, subj subjectFacts, used []string) string {
	idef, ok := a.RT.Intents.Def(it.Name)
	if !ok || idef.Weak {
		return "metric" // chỉ số theo quy ước của pack ("bao nhiêu"), không do người hỏi nêu
	}
	if len(readPeriods(in)) != 1 {
		return "period" // không nêu mốc (kỳ mặc định của pack) hoặc nhiều mốc
	}
	raw := in.Raw
	if !normalization.HasDiacritic(raw) {
		return "accent" // câu không dấu: bộ chuẩn hoá khớp bỏ dấu nên nhiều chữ trùng nhau ("co" = có / cỡ, "dau" = dầu / đầu)
	}
	if markerCount(raw, a.RT.ConfirmMarkers) != 1 {
		return "marker" // nhiều cụm xác nhận là nhiều claim; một phán quyết không trả lời đủ
	}
	inName := func(s, e int) bool {
		return slices.ContainsFunc(names, func(m mention) bool { return s < m.end && e > m.start })
	}
	lo, qs := 0, 0 // [lo, len): phạm vi hỏi; trước lo: ngữ cảnh; qs: đầu mệnh đề chứa cụm xác nhận
	if fs, fe, focus := intent.FocusClause(raw, a.RT.ConfirmMarkers); focus {
		lo, qs = fs, fs
		if !slices.ContainsFunc(in.Spans, func(sp contracts.Span) bool {
			return (sp.Kind == contracts.ValueQuantity || sp.Kind == contracts.ValueNumber) && sp.Start >= fs && sp.Start < fe && !inName(sp.Start, sp.End)
		}) {
			lo = sentenceStart(raw, fs) // mệnh đề hỏi không có số ("…, vậy đúng không?"): câu hỏi áp cho cả câu
		}
	}
	// Kiểm chủ thể ở CẢ hai phạm vi: cả câu hỏi ([lo, len) — "… 3850 L. Dự án A được cấp 3850 L, đúng không?" hỏi về dự án A) và
	// riêng mệnh đề chứa cụm xác nhận ([qs, len) — "… 3850 L, dự án A đúng không?"). Một phạm vi không thay được phạm vi kia.
	for _, from := range []int{lo, qs} {
		if gap := subj.gap(names, from); gap != "" {
			return gap
		}
	}
	covered := make([]bool, len(raw))
	cover := func(s, e int) {
		for i := max(s, 0); i < e && i < len(raw); i++ {
			covered[i] = true
		}
	}
	for _, m := range names {
		cover(m.start, m.end)
	}
	value, lastEnd, firstEnd := false, -1, -1 // lastEnd / firstEnd: hết lần nhắc gần nhất / đầu tiên của con số xác nhận
	for _, sp := range in.Spans {
		switch sp.Kind {
		case contracts.ValueDate, contracts.ValuePeriod:
			cover(sp.Start, sp.End)
			if ws, we, ok := normalization.PeriodWordBefore(raw, sp.Start); ok {
				cover(ws, we) // "ngày" của "ngày 13/10"
			}
		case contracts.ValueTerm:
			if sp.Start < lo || slices.Contains(idef.Terms, sp.Value.Text) { // phạm vi hỏi: chỉ thuật ngữ vật tư của chính ý định ("dầu")
				cover(sp.Start, sp.End)
			}
		case contracts.ValueQuantity, contracts.ValueNumber:
			if inName(sp.Start, sp.End) {
				continue // số trong tên thực thể ("03" của "máy 03")
			}
			if q, ok := normalization.PeriodQualifier(raw, sp.Start, sp.End); ok && sp.Kind == contracts.ValueNumber && qualifierHolds(q, readPeriods(in)) {
				cover(sp.Start, sp.End) // mốc viết bằng chữ KHỚP kỳ đã đọc ("hôm nay tuần 43"): thuộc chiều kỳ
				cover(q.Word[0], q.Word[1])
				continue
			}
			if sp.Start < lo {
				if unitShown(sp) {
					cover(sp.Start, sp.End) // lượng của ngữ cảnh ("đội báo 4k, …"); số trần ở ngữ cảnh có thể là nhãn / mốc ⇒ không giải thích
				}
				continue
			}
			// MỌI con số khác trong phạm vi hỏi phải chính là con số được xác nhận, viết ra với đơn vị cùng chiều đo và nằm trong miền
			// của chỉ số (ngoài miền: dấu chấm / phẩy hay đơn vị có thể đã bị đọc sai — "3.850 m3" thành 3.850.000 L). Dấu xấp xỉ / chặn /
			// mẫu số ("khoảng", "~", ">", "/ngày") nằm ngoài span số nên thành phần dư ở bước cuối.
			v, ok := a.inMetricUnit(sp.Value, def)
			if !ok || !unitShown(sp) || def.InRange(v.Number) != nil {
				return "value"
			}
			if c, err := decimal.Cmp(v.Number, claimed.Number); err != nil || c != 0 {
				return "value"
			}
			if value && (sp.Start < lastEnd || strings.Trim(raw[lastEnd:sp.Start], " ()") != "") {
				return "value" // con số nhắc lại sau một mệnh đề khác ("EX-03 … 3850 L, dự án A 3850 L, đúng không?") là phát biểu thứ hai, không phải cùng một claim
			}
			cover(sp.Start, sp.End)
			if !value {
				firstEnd = sp.End
			}
			value, lastEnd = true, sp.End
		}
	}
	if !value {
		return "value"
	}
	// Sau con số xác nhận đầu tiên chỉ được nhắc lại CHÍNH chủ thể: một tên khác đứng sau con số, kể cả cấp cha, có thể gắn con số
	// hay lần nhắc lại vào thực thể đó ("… 3850 L (3850 L của dự án A) đúng không?", "… 3850 L, dự án A, đúng không?") ⇒ chủ thể
	// của claim chưa giải. Giá: định ngữ cấp cha đặt sau con số mất phán quyết.
	if slices.ContainsFunc(names, func(m mention) bool { return m.start >= firstEnd && !slices.Contains(m.ids, subj.id) }) {
		return "subject"
	}
	srcSpans, gap := sourceMentions(raw, def, used, lo)
	if gap != "" {
		return gap
	}
	for _, sp := range srcSpans {
		cover(sp[0], sp[1])
	}
	coverPhrases := func(phrases []string, from, to int) {
		for _, p := range phrases {
			for _, sp := range normalization.PhraseSpans(raw, p) {
				if sp[0] >= from && sp[1] <= to {
					cover(sp[0], sp[1])
				}
			}
		}
	}
	// từ khoá của CHÍNH ý định (ở mọi mệnh đề), cụm xác nhận, từ trung tính của pack
	coverPhrases(slices.Concat(idef.Keywords, a.RT.ConfirmMarkers, a.RT.ClaimNeutral), 0, len(raw))
	if lo > 0 {
		// ngữ cảnh: thêm từ khoá của các ý định số liệu, tên nguồn bất kỳ của pack, từ chỉ lượng ("Đội báo 2k, vậy …") — KHÔNG thêm từ
		// khoá của ý định quy trình / chào hỏi / phép tính dẫn xuất ("Làm gì có, …", "Hello, …", "Khi lệch, …")
		var labels []string
		for _, id := range a.RT.Metrics.MetricIDs() {
			m, _ := a.RT.Metrics.Metric(id)
			for _, s := range m.Sources {
				labels = append(labels, s.Label)
			}
		}
		coverPhrases(slices.Concat(a.RT.Intents.MetricKeywords(), labels, a.RT.QuantityMarkers), 0, lo)
	}
	for i := 0; i < len(raw); {
		r, n := utf8.DecodeRuneInString(raw[i:])
		if !covered[i] && !unicode.IsSpace(r) && !strings.ContainsRune(claimSeparators, r) {
			return "residual" // chữ, số hay ký hiệu ("~", ">", "%", "/") chưa được giải thích
		}
		i += n
	}
	return ""
}

// claimSeparators là dấu câu ngăn cách, không mang nghĩa claim; mọi ký tự khác chưa được giải thích đều là phần dư. Gạch ngắn / dài
// ("–", "—") KHÔNG ở đây: dính vào số thì chúng là dấu âm mà bộ chuẩn hoá không đọc ("–3850 L").
const claimSeparators = ",.;:?!()[]\"'“”‘’…-"

// subjectFacts là chủ thể hệ đã trả lời và các cấp cha của nó (máy → dự án).
type subjectFacts struct {
	id        string
	ancestors map[string]bool
}

// subjectOf lấy chủ thể hệ đã trả lời và các cấp cha của nó từ danh mục thực thể.
func (a *Ask) subjectOf(ctx context.Context, id string) (subjectFacts, error) {
	all, err := a.Deps.Directory.Entities(ctx)
	if err != nil {
		return subjectFacts{}, contracts.Label(contracts.Infra("directory"), err)
	}
	parent := make(map[string]string, len(all))
	for _, e := range all {
		parent[e.ID] = e.Parent
	}
	s := subjectFacts{id: id, ancestors: map[string]bool{}}
	for p := parent[id]; p != "" && !s.ancestors[p]; p = parent[p] {
		s.ancestors[p] = true
	}
	return s, nil
}

// gap trả "subject" khi chủ thể của claim chưa giải: một cụm tên chỉ về thực thể khác chủ thể mà không phải cấp cha của nó ("EX-03
// của dự án B", "Dự án A … cho EX-03"), hoặc phạm vi hỏi có nêu tên mà không nêu chủ thể ("EX-03 … 3850 L, vậy dự án A … đúng không?").
func (s subjectFacts) gap(names []mention, lo int) string {
	inFocus, subjectInFocus := false, false
	for _, m := range names {
		if !slices.ContainsFunc(m.ids, func(id string) bool { return id == s.id || s.ancestors[id] }) {
			return "subject"
		}
		if m.start >= lo {
			inFocus = true
			subjectInFocus = subjectInFocus || slices.Contains(m.ids, s.id)
		}
	}
	if inFocus && !subjectInFocus {
		return "subject"
	}
	return ""
}

// markerCount đếm số cụm xác nhận rời nhau trong câu ("có đúng không" là một cụm dù "có đúng" và "đúng không" chồng lên nhau).
func markerCount(raw string, markers []string) int {
	var spans [][2]int
	for _, m := range markers {
		spans = append(spans, normalization.PhraseSpans(raw, m)...)
	}
	slices.SortFunc(spans, func(x, y [2]int) int { return x[0] - y[0] })
	n, end := 0, -1
	for _, sp := range spans {
		if sp[0] >= end {
			n++
		}
		end = max(end, sp[1])
	}
	return n
}

// sourceMentions trả vị trí các tên nguồn của chỉ số (nhãn + bí danh của pack) trong phạm vi hỏi [lo, len). Tên nguồn chỉ được giải
// thích khi con số trả lời lấy ĐÚNG và CHỈ từ nguồn đó: "phiếu xuất kho ghi 3900 L đúng không?" mà 3.900 L là của ERP ⇒ "source".
func sourceMentions(raw string, def sourceoftruth.MetricDef, used []string, lo int) ([][2]int, string) {
	var out [][2]int
	for _, s := range def.Sources {
		for _, p := range append([]string{s.Label}, s.Aliases...) {
			if p == "" {
				continue
			}
			for _, sp := range normalization.PhraseSpans(raw, p) {
				if sp[0] < lo {
					continue
				}
				if len(used) != 1 || used[0] != s.Type {
					return nil, "source"
				}
				out = append(out, sp)
			}
		}
	}
	return out, ""
}

// unitShown báo con số tự mang chiều đo: lượng có đơn vị (viết rõ, hoặc suy từ thuật ngữ vật tư đứng cạnh: "4k dầu"), hoặc số có chữ
// ("4k", "bốn ngàn"). Số trần — kể cả có dấu chấm / phẩy ("3850", "3.850", "1,2") — không nói được nó là lít, khối hay số thứ tự, và
// dấu chấm / phẩy còn mơ hồ giữa phần nghìn và phần thập phân.
func unitShown(sp contracts.Span) bool {
	if sp.Kind == contracts.ValueQuantity {
		// đơn vị suy từ thuật ngữ + dấu chấm / phẩy là mơ hồ phần nghìn / thập phân ("dầu 1,2") — trừ khi nguyên văn có chữ ("3,85k",
		// "1,2 nghìn"): chữ đã nói hàng của con số
		return !(strings.Contains(sp.Rule, "unit_from_term") && strings.ContainsAny(sp.Raw, ".,") && !strings.ContainsFunc(sp.Raw, unicode.IsLetter))
	}
	return strings.ContainsFunc(sp.Raw, unicode.IsLetter)
}
