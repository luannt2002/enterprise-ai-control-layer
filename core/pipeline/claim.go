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
	value, lastEnd, firstStart, firstEnd := false, -1, -1, -1 // lastEnd / firstStart / firstEnd: hết lần nhắc gần nhất, đầu và hết lần nhắc đầu tiên của con số xác nhận
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
			if normalization.AmbiguousNumberWords(sp.Raw) {
				return "value" // "một nghìn hai": nói thường là 1.200, bộ đọc cho 1.002 — con số xác nhận chưa đọc chắc (phản hồi audit K5)
			}
			if c, err := decimal.Cmp(v.Number, claimed.Number); err != nil || c != 0 {
				return "value"
			}
			if value && (sp.Start < lastEnd || strings.Trim(raw[lastEnd:sp.Start], " ()") != "") {
				return "value" // con số nhắc lại sau một mệnh đề khác ("EX-03 … 3850 L, dự án A 3850 L, đúng không?") là phát biểu thứ hai, không phải cùng một claim
			}
			cover(sp.Start, sp.End)
			if !value {
				firstStart, firstEnd = sp.Start, sp.End
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
	sources, metrics := a.qualifierPhrases(idef, used)
	own := slices.Clone(idef.Keywords) // chữ của chính claim: từ khoá của chính ý định và tên đúng nguồn đã dùng
	for _, src := range def.Sources {
		if len(used) == 1 && used[0] == src.Type {
			own = append(append(own, src.Label), src.Aliases...)
		}
	}
	if gap := a.topicQualifier(raw, in.Spans, sources, metrics, own, idef.Terms, lo); gap != "" {
		return gap
	}
	extra := contextOwners(raw, in.Spans, inName, slices.Concat(sources, metrics), lo)
	if os := owners(names, in.Spans, idef.Terms, extra); !subj.numberOwned(raw, os, firstStart, firstEnd, lo) {
		return "subject" // con số xác nhận chưa gắn vào chủ thể: thứ gần nhất đứng trước nó là tên / danh từ khác, hay bí danh mơ hồ chưa có bằng chứng dương
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
		// đơn vị suy từ thuật ngữ + dấu chấm / phẩy là mơ hồ phần nghìn / thập phân ("dầu 1,2") — trừ khi nguyên văn KẾT THÚC bằng chữ
		// ("3,85k", "3,85K", "1,2 nghìn"): hậu tố nhân ở cuối đã nói hàng của con số; chữ số sau hậu tố ("1,2k5") vẫn mơ hồ
		return !(strings.Contains(sp.Rule, "unit_from_term") && strings.ContainsAny(sp.Raw, ".,") && strings.TrimRightFunc(sp.Raw, unicode.IsLetter) == sp.Raw)
	}
	return strings.ContainsFunc(sp.Raw, unicode.IsLetter)
}

// ownerRef là một thứ CÓ THỂ sở hữu con số trong câu: tên thực thể (ids khác rỗng) hay danh từ / thuật ngữ khác chính ý định ("dự án",
// "nhân công": ở ngữ cảnh chúng được che là đã hiểu, nhưng vẫn là thứ con số có thể thuộc về).
type ownerRef struct {
	start, end int
	ids        []string
}

// owners trả mọi thứ có thể sở hữu con số, theo vị trí: tên, thuật ngữ, và các cụm ở ngữ cảnh do contextOwners tìm (extra). Thuật ngữ hay
// cụm nằm trong một cụm tên ("dự án" của "dự án A") không tính riêng.
func owners(names []mention, spans []contracts.Span, ownTerms []string, extra [][2]int) []ownerRef {
	var out []ownerRef
	inName := func(s, e int) bool {
		return slices.ContainsFunc(names, func(m mention) bool { return s < m.end && e > m.start })
	}
	for _, m := range names {
		out = append(out, ownerRef{m.start, m.end, m.ids})
	}
	for _, sp := range spans {
		if sp.Kind != contracts.ValueTerm || slices.Contains(ownTerms, sp.Value.Text) || inName(sp.Start, sp.End) {
			continue
		}
		out = append(out, ownerRef{sp.Start, sp.End, nil})
	}
	for _, sp := range extra {
		if !inName(sp[0], sp[1]) {
			out = append(out, ownerRef{sp[0], sp[1], nil})
		}
	}
	slices.SortFunc(out, func(a, b ownerRef) int { return a.start - b.start })
	return out
}

// qualifierPhrases trả các cụm có thể là nguồn / chỉ số của một con số khác chỉ số đang hỏi (ADR-0009 Vòng 5, R5-1): tên nguồn khác nguồn đã
// tạo con số trả lời, và từ khoá của ý định số liệu khác cùng từ chỉ lượng. Bỏ từ khoá của ý định yếu ("bao nhiêu" chỉ chính chỉ số đang
// hỏi); với chỉ số, bỏ cả từ khoá của chính ý định ("được cấp"). Tên nguồn trùng từ khoá của chính ý định vẫn là nguồn khác khi số trả lời
// không lấy từ nó ("phiếu kho" khi số là của ERP, R5-5).
func (a *Ask) qualifierPhrases(idef intent.Def, used []string) (sources, metrics []string) {
	weak := a.RT.Intents.WeakKeywords()
	keepSrc := func(p string) bool { return p != "" && !slices.Contains(weak, p) }
	keep := func(p string) bool { return keepSrc(p) && !slices.Contains(idef.Keywords, p) }
	for _, id := range a.RT.Metrics.MetricIDs() {
		m, _ := a.RT.Metrics.Metric(id)
		for _, src := range m.Sources {
			if len(used) == 1 && used[0] == src.Type {
				continue // tên đúng nguồn đã dùng ("Phiếu xuất kho, 3850 L đúng không?")
			}
			for _, p := range append([]string{src.Label}, src.Aliases...) {
				if keepSrc(p) {
					sources = append(sources, p)
				}
			}
		}
	}
	for _, p := range slices.Concat(a.RT.Intents.MetricKeywords(), a.RT.QuantityMarkers) {
		if keep(p) {
			metrics = append(metrics, p)
		}
	}
	return sources, metrics
}

// contextOwners trả các cụm qualifierPhrases ở ngữ cảnh [0, lo) mà KHÔNG có con số nào xen giữa chúng và phạm vi hỏi: chúng có thể sở hữu
// con số đứng sau ("… 3850 L. Định mức, 5000 L đúng không?", "EX-03 hôm nay đội báo, được cấp 3850 L đúng không?"). Cụm đã có số của
// riêng nó ("Đội báo 3k, vậy được cấp 3850 L") là một sự kiện riêng của ngữ cảnh.
func contextOwners(raw string, spans []contracts.Span, inName func(s, e int) bool, phrases []string, lo int) [][2]int {
	numberIn := func(s, e int) bool {
		return slices.ContainsFunc(spans, func(sp contracts.Span) bool {
			return (sp.Kind == contracts.ValueQuantity || sp.Kind == contracts.ValueNumber) && sp.Start >= s && sp.Start < e && !inName(sp.Start, sp.End)
		})
	}
	var out [][2]int
	for _, p := range phrases {
		for _, sp := range normalization.PhraseSpans(raw, p) {
			if sp[1] <= lo && !numberIn(sp[1], lo) {
				out = append(out, sp)
			}
		}
	}
	return out
}

// topicQualifier trả "source" / "metric" khi đoạn từ đầu câu chứa phạm vi hỏi tới lo có tên nguồn khác nguồn đã dùng / từ khoá của ý định
// số liệu khác / từ chỉ lượng, và ngoài chúng CHỈ có chữ mà phép kiểm chính tự giải thích — từ trung tính, mốc, dấu ngăn, chữ của chính
// claim (own: từ khoá chính ý định, tên đúng nguồn đã dùng; ownTerms: thuật ngữ vật tư của chính ý định, R5-4): "ERP, EX-03 hôm nay được
// cấp 3900 L đúng không?", "… 3850 L. ERP, 3900 L đúng không?", "Đội báo, EX-03 …", "Dầu ERP, EX-03 …" — đề mục đó là nguồn / chỉ số
// của chính claim (R5-1, gồm R2-4). Đoạn có chữ khác ("Bao nhiêu định mức, vậy EX-03 …", "Theo ERP, …") không phải đề mục của claim.
func (a *Ask) topicQualifier(raw string, spans []contracts.Span, sources, metrics, own, ownTerms []string, lo int) string {
	from := sentenceStart(raw, lo)
	if from >= lo {
		return ""
	}
	covered := make([]bool, lo-from)
	mark := func(s, e int) {
		for i := s; i < e; i++ {
			covered[i-from] = true
		}
	}
	markPhrases := func(phrases []string) bool {
		hit := false
		for _, p := range phrases {
			for _, sp := range normalization.PhraseSpans(raw, p) {
				if sp[0] >= from && sp[1] <= lo {
					mark(sp[0], sp[1])
					hit = true
				}
			}
		}
		return hit
	}
	src, met := markPhrases(sources), markPhrases(metrics)
	if !src && !met {
		return ""
	}
	markPhrases(a.RT.ClaimNeutral)
	markPhrases(own)
	for _, sp := range spans {
		if sp.Kind == contracts.ValueTerm && slices.Contains(ownTerms, sp.Value.Text) && sp.Start >= from && sp.End <= lo {
			mark(sp.Start, sp.End)
		}
		if (sp.Kind == contracts.ValueDate || sp.Kind == contracts.ValuePeriod) && sp.Start >= from && sp.End <= lo {
			mark(sp.Start, sp.End)
			if ws, we, ok := normalization.PeriodWordBefore(raw, sp.Start); ok && ws >= from {
				mark(ws, we) // "ngày" của "ngày 19/10", như ở verdictGap
			}
		}
	}
	for i := from; i < lo; {
		r, n := utf8.DecodeRuneInString(raw[i:])
		if !covered[i-from] && !unicode.IsSpace(r) && !strings.ContainsRune(claimSeparators, r) {
			return ""
		}
		i += n
	}
	if src {
		return "source"
	}
	return "metric"
}

// plainBetween báo đoạn chữ chỉ gồm chữ, số, dấu cách và dấu chấm / phẩy / gạch chéo / gạch ngang / hai chấm giữa hai chữ số
// ("3.850", "19/10", "19-10", "8:30"): không dấu câu nào. Ngày, giờ viết bằng số không phải ngắt mệnh đề.
func plainBetween(s string) bool {
	rs := []rune(s)
	for i, r := range rs {
		switch {
		case unicode.IsLetter(r) || unicode.IsDigit(r) || r == ' ':
		case strings.ContainsRune(".,/-:", r) && i > 0 && i+1 < len(rs) && unicode.IsDigit(rs[i-1]) && unicode.IsDigit(rs[i+1]):
		default:
			return false
		}
	}
	return true
}

// bareWrapped báo đoạn giữa hai cụm là một ngoặc mở trần và sau cụm thứ hai là ngoặc đóng tương ứng: "EX-03 (dự án A)", "máy 03 (EX-03)".
func bareWrapped(raw, between string, afterEnd int) bool {
	rest := strings.TrimLeft(raw[afterEnd:], " ")
	switch strings.TrimSpace(between) {
	case "(":
		return strings.HasPrefix(rest, ")")
	case "[":
		return strings.HasPrefix(rest, "]")
	}
	return false
}

// wrapped báo cụm [start, end) đứng một mình trong ngoặc: ký tự khác khoảng trắng liền trước là "(" / "[" và liền sau là ")" / "]" tương ứng.
func wrapped(raw string, start, end int) bool {
	before, after := strings.TrimRight(raw[:start], " "), strings.TrimLeft(raw[end:], " ")
	return (strings.HasSuffix(before, "(") && strings.HasPrefix(after, ")")) || (strings.HasSuffix(before, "[") && strings.HasPrefix(after, "]"))
}

// hasBreak báo đoạn chữ có ngắt mệnh đề: một dấu , ; . ? ! (không nằm giữa hai chữ số: "3.850", "0,5") hay xuống dòng.
func hasBreak(s string) bool {
	for i := 0; i < len(s); i++ {
		switch s[i] {
		case '\n':
			return true
		case ',', ';', '.', '?', '!':
			if !(i > 0 && i+1 < len(s) && s[i-1] >= '0' && s[i-1] <= '9' && s[i+1] >= '0' && s[i+1] <= '9') {
				return true
			}
		}
	}
	return false
}

// aliasTied: bí danh MƠ HỒ chứa chủ thể chỉ là chính chủ thể khi câu cho bằng chứng dương: không có lần nhắc rõ nào của chủ thể (router đã gỡ
// bí danh bằng phạm vi quyền hay cấp cha, mọi bí danh trong câu cùng một thực thể); hoặc nó đứng một mình trong ngoặc, không ngắt mệnh đề khỏi lần
// nhắc rõ ("EX-03 (máy 03)", "EX-03 … 3850 L (máy 03)"); hoặc lần nhắc rõ đứng một mình trong ngoặc liền sau nó ("máy 03 (EX-03)"); hoặc nó nhắc lại
// chủ thể SAU con số xác nhận mà không dấu câu nào ngăn ("EX-03 … 3850 L máy 03 đúng không?"). Còn lại — dấu ngăn nào cũng vậy, kể cả ":" " - "
// "(" "[" "…" hay chỉ một khoảng trắng — là chưa giải.
func (s subjectFacts) aliasTied(raw string, os []ownerRef, k, numEnd int) bool {
	best, bestDist := -1, 1<<30
	for i, c := range os {
		if len(c.ids) != 1 || c.ids[0] != s.id {
			continue
		}
		d := c.start - os[k].end
		if c.end <= os[k].start {
			d = os[k].start - c.end
		}
		if d < bestDist {
			best, bestDist = i, d
		}
	}
	if best < 0 {
		return true
	}
	c, a := os[best], os[k]
	if c.end <= a.start { // lần nhắc rõ đứng trước bí danh
		between := raw[c.end:a.start]
		return (wrapped(raw, a.start, a.end) && !hasBreak(between)) || (a.start >= numEnd && plainBetween(between))
	}
	if a.end <= c.start { // bí danh đứng trước lần nhắc rõ: chỉ dạng "máy 03 (EX-03)"
		return bareWrapped(raw, raw[a.end:c.start], c.end)
	}
	return false
}

// ownedBy báo thứ thứ k đã gắn vào chủ thể: chính chủ thể (nhắc rõ, hay bí danh mơ hồ có bằng chứng dương), hoặc định ngữ — cấp cha / danh từ
// khác — nối vào thứ liền trước (đã gắn) bằng đúng chữ "của" hay đứng một mình trong ngoặc ("EX-03 (dự án A)").
func (s subjectFacts) ownedBy(raw string, os []ownerRef, k, numEnd int) bool {
	c := os[k]
	if slices.Contains(c.ids, s.id) {
		return len(c.ids) == 1 || s.aliasTied(raw, os, k, numEnd)
	}
	if k == 0 || os[k-1].end > c.start {
		return false
	}
	if !strings.EqualFold(strings.TrimSpace(raw[os[k-1].end:c.start]), "của") && !bareWrapped(raw, raw[os[k-1].end:c.start], c.end) {
		return false
	}
	return s.ownedBy(raw, os, k-1, numEnd)
}

// numberOwned: thứ gần nhất đứng TRƯỚC con số xác nhận đầu tiên phải gắn vào chủ thể (không có thứ nào đứng trước thì chủ thể của con số do
// phần sau con số kiểm); và mọi bí danh mơ hồ chứa chủ thể trong phạm vi hỏi [lo, len) phải có bằng chứng dương (aliasTied).
func (s subjectFacts) numberOwned(raw string, os []ownerRef, numStart, numEnd, lo int) bool {
	k := -1
	for i, c := range os {
		if c.end <= numStart {
			k = i
		}
	}
	if k >= 0 && !s.ownedBy(raw, os, k, numEnd) {
		return false
	}
	for i, c := range os {
		if c.start >= lo && len(c.ids) > 1 && slices.Contains(c.ids, s.id) && !s.aliasTied(raw, os, i, numEnd) {
			return false
		}
	}
	return true
}
