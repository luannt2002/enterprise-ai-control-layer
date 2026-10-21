package pipeline

import (
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
// lượng tự mang đơn vị, tên chỉ số / nguồn bất kỳ của pack — sự kiện người hỏi nêu về chỉ số khác — nhưng mọi chữ khác vẫn phải
// được giải thích ("Không tính ERP, …" vẫn mất phán quyết).
func (a *Ask) verdictGap(in contracts.NormalizedInput, names []mention, it contracts.Intent, def sourceoftruth.MetricDef, claimed contracts.Value) string {
	idef, ok := a.RT.Intents.Def(it.Name)
	if !ok || idef.Weak {
		return "metric" // chỉ số theo quy ước của pack ("bao nhiêu"), không do người hỏi nêu
	}
	if len(readPeriods(in)) != 1 {
		return "period" // không nêu mốc (kỳ mặc định của pack) hoặc nhiều mốc
	}
	raw := in.Raw
	inName := func(s, e int) bool {
		return slices.ContainsFunc(names, func(m mention) bool { return s < m.end && e > m.start })
	}
	lo := 0 // [lo, len): phạm vi hỏi; trước lo: ngữ cảnh
	if fs, fe, focus := intent.FocusClause(raw, a.RT.ConfirmMarkers); focus {
		lo = fs
		if !slices.ContainsFunc(in.Spans, func(sp contracts.Span) bool {
			return (sp.Kind == contracts.ValueQuantity || sp.Kind == contracts.ValueNumber) && sp.Start >= fs && sp.Start < fe && !inName(sp.Start, sp.End)
		}) {
			lo = sentenceStart(raw, fs) // mệnh đề hỏi không có số ("…, vậy đúng không?"): câu hỏi áp cho cả câu
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
	value := false
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
			// MỌI con số khác trong phạm vi hỏi phải chính là con số được xác nhận, viết ra với đơn vị cùng chiều đo. Dấu xấp xỉ / chặn / mẫu
			// số ("khoảng", "~", ">", "/ngày") nằm ngoài span số nên thành phần dư ở bước cuối.
			v, ok := a.inMetricUnit(sp.Value, def)
			if !ok || !unitShown(sp) {
				return "value"
			}
			if c, err := decimal.Cmp(v.Number, claimed.Number); err != nil || c != 0 {
				return "value"
			}
			cover(sp.Start, sp.End)
			value = true
		}
	}
	if !value {
		return "value"
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
	// phạm vi hỏi: từ khoá của CHÍNH ý định, nguồn có thẩm quyền cao nhất của CHÍNH chỉ số ("phiếu xuất kho … đúng không?": con số được so
	// đúng với nguồn người hỏi nêu); nguồn thấp hơn ("ERP ghi …") là claim về con số của nguồn đó ⇒ không giải thích.
	coverPhrases(slices.Concat(idef.Keywords, topSourceLabels(def), a.RT.ConfirmMarkers, a.RT.ClaimNeutral), 0, len(raw))
	if lo > 0 {
		var labels []string
		for _, id := range a.RT.Metrics.MetricIDs() {
			m, _ := a.RT.Metrics.Metric(id)
			for _, s := range m.Sources {
				labels = append(labels, s.Label)
			}
		}
		coverPhrases(slices.Concat(a.RT.Intents.Keywords(), labels, a.RT.QuantityMarkers), 0, lo)
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

// claimSeparators là dấu câu ngăn cách, không mang nghĩa claim; mọi ký tự khác chưa được giải thích đều là phần dư.
const claimSeparators = ",.;:?!()[]\"'“”‘’…-–—"

// topSourceLabels trả cách gọi các nguồn có thẩm quyền CAO NHẤT của chỉ số (authority nhỏ nhất).
func topSourceLabels(def sourceoftruth.MetricDef) []string {
	best := 0
	for _, s := range def.Sources {
		if best == 0 || s.Authority < best {
			best = s.Authority
		}
	}
	var out []string
	for _, s := range def.Sources {
		if s.Authority == best && s.Label != "" {
			out = append(out, s.Label)
		}
	}
	return out
}

// unitShown báo con số tự mang chiều đo: lượng có đơn vị (viết rõ, hoặc suy từ thuật ngữ vật tư đứng cạnh: "4k dầu"), hoặc số có
// dạng lượng ("4k", "4.000", "bốn ngàn"). Số trần trơn ("3850", "912") không nói được nó là lít, số phiếu hay số thứ tự.
func unitShown(sp contracts.Span) bool {
	if sp.Kind == contracts.ValueQuantity {
		return true
	}
	return strings.ContainsFunc(sp.Raw, func(r rune) bool { return unicode.IsLetter(r) || r == '.' || r == ',' })
}
