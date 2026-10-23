package pipeline

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"maps"
	"slices"
	"strings"
	"unicode/utf8"

	"github.com/luannt2002/enterprise-ai-control-layer/core/contracts"
	"github.com/luannt2002/enterprise-ai-control-layer/data/normalization"
	"github.com/luannt2002/enterprise-ai-control-layer/data/sourceoftruth"
	"github.com/luannt2002/enterprise-ai-control-layer/intelligence/intent"
	"github.com/luannt2002/enterprise-ai-control-layer/observability/traces"
)

// File này dựng phần GIẢI THÍCH của vết (traces.Why, R2-03) từ chính những gì luồng hỏi đã quyết: mọi dòng lấy từ dữ liệu có sẵn tại chỗ
// quyết định (Span.Rule, Resolution, intent.Result.Why, RoutePlan.Checked, sourceoftruth.Result, verdictGap…), không suy lại từ kết quả
// về sau. Hàm ở đây chỉ ĐỌC và định dạng; không hàm nào đổi trạng thái hay quyết định của luồng hỏi.

// hiddenCandidate là ứng viên thực thể bị loại khỏi câu hỏi lại, kèm lý do.
type hiddenCandidate struct {
	ref contracts.EntityRef
	why string
}

// clarifyInfo ghi cách luồng hỏi dựng danh sách ứng viên của câu hỏi lại (chỉ khi route là CLARIFICATION).
type clarifyInfo struct {
	used       bool
	before     []contracts.EntityRef // ứng viên trước lọc phạm vi: Result.Candidates (S3) và Result.Unclear.Entities (S4 gộp)
	shown      []contracts.EntityRef // ứng viên được kể trong câu hỏi lại, sau lọc phạm vi
	hidden     []hiddenCandidate     // ứng viên bị loại khỏi câu hỏi lại
	suppressed string                // lý do KHÔNG kể ứng viên dù có (câu hỏi lại về con số hay mốc thời gian)
}

// rawSpan trả đoạn nguyên văn [start, end) của câu (rỗng khi vị trí không hợp lệ).
func rawSpan(raw string, start, end int) string {
	if start < 0 || end > len(raw) || start >= end {
		return ""
	}
	return raw[start:end]
}

// refKeys là khoá các thực thể, không trùng, theo thứ tự gặp.
func refKeys(refs []contracts.EntityRef) []string {
	var out []string
	for _, r := range refs {
		if !slices.Contains(out, r.Key) {
			out = append(out, r.Key)
		}
	}
	return out
}

// maxListed là số khoá tối đa in trong MỘT danh sách của dòng giải thích (còn lại in "và N … khác"); maxWhyLines là trần số ý mỗi danh sách
// (Reasons / Rejected) của một chặng, áp sau khi các dòng giống nhau đã gộp; maxWhyLineRunes là trần số ký tự của MỘT dòng (dòng dài hơn bị
// cắt kèm chữ báo): câu dài, bí danh nhiều ứng viên hay nhiều thực thể khác nhau không làm vết phình theo.
const (
	maxListed       = 6
	maxWhyLines     = 40
	maxWhyLineRunes = 1500
)

// distinct bỏ khoá trùng, giữ thứ tự gặp đầu tiên.
func distinct(keys []string) []string {
	var out []string
	for _, k := range keys {
		if !slices.Contains(out, k) {
			out = append(out, k)
		}
	}
	return out
}

// distinctRefs bỏ thực thể trùng mã, giữ thứ tự gặp đầu tiên.
func distinctRefs(refs []contracts.EntityRef) []contracts.EntityRef {
	var out []contracts.EntityRef
	for _, r := range refs {
		if !slices.ContainsFunc(out, func(x contracts.EntityRef) bool { return x.ID == r.ID }) {
			out = append(out, r)
		}
	}
	return out
}

// candidateCount nói số ứng viên; khoá trùng (một thực thể là ứng viên của nhiều cụm mơ hồ, bước "đúng một ứng viên" của router đếm từng lần)
// được nói ra để số đếm khớp với quyết định.
func candidateCount(keys []string) string {
	if d := distinct(keys); len(d) != len(keys) {
		return fmt.Sprintf("%d ứng viên (%d khoá khác nhau)", len(keys), len(d))
	}
	return fmt.Sprintf("%d ứng viên", len(keys))
}

// afterText mô tả tập ứng viên SAU lọc: số lượng và các khoá (cắt bớt).
func afterText(keys []string) string {
	if len(keys) == 0 {
		return "không còn ứng viên nào"
	}
	return fmt.Sprintf("%s [%s]", candidateCount(keys), contracts.Brief(distinct(keys), maxListed, "ứng viên"))
}

// checkGroup là các bậc kiểm GIỐNG HỆT nhau của router (vòng lặp qua nhiều lần nhắc của cùng thực thể), kèm số lần.
type checkGroup struct {
	check contracts.RouteCheck
	n     int
}

// groupChecks gộp các bậc kiểm giống hệt nhau, giữ thứ tự gặp đầu tiên.
func groupChecks(checks []contracts.RouteCheck) []checkGroup {
	var out []checkGroup
	for _, c := range checks {
		if i := slices.IndexFunc(out, func(g checkGroup) bool { return g.check == c }); i >= 0 {
			out[i].n++
			continue
		}
		out = append(out, checkGroup{c, 1})
	}
	return out
}

// ladderOf là thang bậc kiểm hiển thị: các bậc "qua" LIÊN TIẾP cùng tên (và cùng route) gộp một bậc kèm tổng số lần, dù dữ kiện của từng bậc khác nhau
// (mỗi thực thể một dữ kiện): câu nêu hàng trăm thực thể khác nhau không làm dòng thang bậc dài theo. Dòng thang bậc chỉ ghi "qua" cho bậc không
// áp dụng; dữ kiện từng bậc nằm ở dòng phương án bị loại (có trần số ý). Bậc đã chặn / đã chọn không gộp với ai.
func ladderOf(groups []checkGroup) []checkGroup {
	var out []checkGroup
	for _, g := range groups {
		if last := len(out) - 1; last >= 0 && !g.check.Fired && !out[last].check.Fired && out[last].check.Rule == g.check.Rule && out[last].check.Route == g.check.Route {
			out[last].n += g.n
			continue
		}
		out = append(out, g)
	}
	return out
}

// explainNormalize giải thích S2: luật sinh từng đoạn (Span.Rule) và các chỗ chuẩn hoá KHÔNG tự quyết (chỗ mơ hồ, ngày hỏng).
func explainNormalize(in contracts.NormalizedInput) traces.Notes {
	var n traces.Notes
	if len(in.Spans) == 0 {
		n.Reason("không có đoạn số, ngày, mã hay thuật ngữ nào được nhận ra")
	}
	for _, sp := range in.Spans {
		line := fmt.Sprintf("%s %s ← %s", sp.Kind, contracts.Quote(sp.Raw), sp.Rule)
		if sp.Confidence > 0 && sp.Confidence < 1 {
			line += fmt.Sprintf(" (độ tin cậy %.2f)", sp.Confidence)
		}
		n.Reasons = append(n.Reasons, line)
	}
	for _, amb := range in.Ambiguities {
		var readings []string
		for _, c := range amb.Candidates {
			readings = append(readings, valueText(c))
		}
		n.Reject("chỗ mơ hồ %s không được tự quyết (%s): các cách đọc %s", contracts.Quote(amb.Raw), amb.Reason, strings.Join(readings, " | "))
	}
	for _, raw := range in.InvalidDates {
		n.Reject("ngày %s không đọc được thành ngày trên lịch: bỏ, không đoán và không rơi về ngày mặc định", contracts.Quote(raw))
	}
	return n
}

// explainResolve giải thích S3: thực thể rõ, cụm mơ hồ, và tập ứng viên TRƯỚC / SAU lọc phạm vi — lọc của router khi gỡ mơ hồ (nar) và lọc
// của luồng hỏi khi dựng câu hỏi lại (ci). Vết do chính người hỏi đọc, nên: KHÔNG kể trạng thái phạm vi của TỪNG ứng viên ngoài phạm vi (chỉ số
// lượng bị loại và lý do chung), và chỉ kể tên + loại thực thể khi fullScope (người hỏi có phạm vi dữ liệu không hạn chế). Khoá của thực thể
// ngoài phạm vi KHÔNG được lọc ở đây mà ở scopeRedactor, một chỗ cho mọi dòng của mọi chặng (C9 vòng 2): dòng ở đây vẫn kể khoá ứng viên, và
// với người không toàn quyền, khoá ngoài phạm vi mà họ không tự gõ thành traces.OutOfScope trước khi vào bản ghi.
func explainResolve(in contracts.NormalizedInput, res contracts.Resolution, nar contracts.Narrowing, ci clarifyInfo, fullScope bool) traces.Notes {
	var n traces.Notes
	if len(res.Entities)+len(res.Ambiguous) == 0 {
		n.Reason("không nhận ra thực thể nào trong câu")
	}
	for _, e := range res.Entities {
		line := fmt.Sprintf("cụm %s (khoá so khớp %s) khớp %s", contracts.Quote(rawSpan(in.Raw, e.Start, e.End)), contracts.Quote(e.Mention), e.Key)
		if fullScope {
			line += fmt.Sprintf(" (%s, loại %s)", e.Name, e.Type)
		}
		n.Reasons = append(n.Reasons, line)
	}
	for _, m := range res.Ambiguous {
		keys := refKeys(m.Candidates)
		n.Reason("cụm %s (khoá so khớp %s) khớp %d thực thể: %s — không chọn bừa", contracts.Quote(rawSpan(in.Raw, m.Start, m.End)), contracts.Quote(m.Mention),
			len(keys), contracts.Brief(keys, maxListed, "ứng viên"))
	}
	if nar.Slot != "" {
		by, dropWhy := "phạm vi dữ liệu của người hỏi", "ngoài phạm vi dữ liệu của người hỏi"
		if nar.By == contracts.NarrowByProject {
			by, dropWhy = "dự án đã nêu trong câu", "không thuộc dự án đã nêu trong câu"
		}
		result := "giữ nguyên mơ hồ"
		if nar.Chosen != "" {
			result = "chọn " + nar.Chosen
		}
		n.Reason("gỡ mơ hồ cho khe %s bằng %s: trước lọc %s → sau lọc %s → %s", nar.Slot, by, candidateCount(nar.Before), afterText(nar.After), result)
		if dropped := len(nar.Before) - len(nar.After); dropped > 0 {
			n.Reject("%d ứng viên bị loại khi gỡ mơ hồ: %s (vết không nêu trạng thái phạm vi của từng ứng viên)", dropped, dropWhy)
		}
	}
	if ci.used {
		if len(ci.before) > 0 {
			n.Reason("ứng viên của câu hỏi lại: trước lọc phạm vi %d ứng viên → sau lọc %s", len(refKeys(ci.before)), afterText(refKeys(ci.shown)))
		}
		var reasons []string // lý do bị loại, theo thứ tự gặp đầu tiên
		byWhy := map[string][]string{}
		for _, h := range ci.hidden {
			if _, ok := byWhy[h.why]; !ok {
				reasons = append(reasons, h.why)
			}
			if !slices.Contains(byWhy[h.why], h.ref.Key) {
				byWhy[h.why] = append(byWhy[h.why], h.ref.Key)
			}
		}
		for _, why := range reasons {
			n.Reject("%d ứng viên bị loại khỏi câu hỏi lại: %s (vết không nêu trạng thái phạm vi của từng ứng viên)", len(byWhy[why]), why)
		}
		if ci.suppressed != "" {
			n.Reason("không kể ứng viên thực thể trong câu hỏi lại: %s", ci.suppressed)
		}
	}
	return n
}

// explainIntent giải thích S4: phần do intent.Classify ghi tại chỗ quyết định, cộng NGUỒN của từng ứng viên được kể trong câu hỏi lại
// (ứng viên do S3 thấy mơ hồ, hay thực thể do S4 gộp thêm).
func explainIntent(cls intent.Result, ci clarifyInfo) traces.Notes {
	n := traces.Notes{Reasons: slices.Clone(cls.Why.Reasons), Rejected: slices.Clone(cls.Why.Rejected)}
	if !ci.used {
		return n
	}
	shown := distinctRefs(ci.shown)
	for i, c := range shown {
		if i == maxListed {
			n.Reason("và %d ứng viên hỏi lại khác, nguồn xác định theo cùng cách", len(shown)-maxListed)
			break
		}
		for _, o := range cls.Why.Origins {
			if o.ID != c.ID {
				continue
			}
			from := "S3"
			if o.From == intent.OriginS4 {
				from = "do S4 gộp thêm"
			}
			n.Reason("ứng viên hỏi lại %s ← %s: %s", c.Key, from, o.Note)
			break
		}
	}
	return n
}

// explainRoute giải thích S5: lý do của router, các bậc kiểm đã đi qua theo thứ tự, route đã loại và vì sao, và việc luồng hỏi ghi đè route
// (routed là kế hoạch NGUYÊN VĂN của router; plan là kế hoạch cuối cùng sau khi luồng hỏi có thể ghi đè).
func (a *Ask) explainRoute(routed, plan contracts.RoutePlan) traces.Notes {
	var n traces.Notes
	n.Reason("%s", routed.Reason)
	if len(routed.Checked) > 0 {
		groups := groupChecks(routed.Checked)
		ladder := ladderOf(groups)
		steps := make([]string, 0, len(ladder))
		for i, g := range ladder {
			// bậc không áp dụng chỉ ghi "qua": dữ kiện của nó nằm ở dòng phương án bị loại tương ứng (không in hai lần)
			step := fmt.Sprintf("%d) %s: qua", i+1, g.check.Rule)
			if g.check.Fired {
				step = fmt.Sprintf("%d) %s — %s → chọn %s", i+1, g.check.Rule, g.check.Detail, g.check.Route)
			}
			if g.n > 1 {
				step += fmt.Sprintf(" (×%d)", g.n)
			}
			steps = append(steps, step)
		}
		n.Reason("đã kiểm theo thứ tự: %s", strings.Join(steps, " · "))
		for _, g := range groups {
			if !g.check.Fired {
				line := fmt.Sprintf("%s (%s): không áp dụng — %s", g.check.Route, g.check.Rule, g.check.Detail)
				if g.n > 1 {
					line += fmt.Sprintf(" (×%d)", g.n)
				}
				n.Rejected = append(n.Rejected, line)
			}
		}
	}
	// route khai của ý định mà router không đi (bị bậc kiểm chặn, hay thay bằng phép tính / HYBRID)
	if def, ok := a.RT.Intents.Def(routed.Intent.Name); ok && def.Route != "" && contracts.RouteKind(def.Route) != routed.Route {
		n.Reject("%s (route khai của ý định %s): không đi — %s", def.Route, def.Name, routed.Reason)
	}
	if plan.Route != routed.Route {
		n.Reason("luồng hỏi ghi đè route %s thành %s: %s", routed.Route, plan.Route, plan.Reason)
		n.Reject("%s (route của router): bị ghi đè — %s", routed.Route, plan.Reason)
	}
	return n
}

// noteMetricContext ghi S7 của route số liệu: bằng chứng nào vào gói ngữ cảnh, vai của từng nguồn (thẩm quyền) và nguồn nào không được chọn.
// Gọi khi trả về khỏi answerMetric nên pkg đã là gói CUỐI (kể cả số của người hỏi và các chỉ số khác cùng kỳ thêm vào sau câu xác nhận).
func (a *Ask) noteMetricContext(why *traces.Why, prefix string, result sourceoftruth.Result, pkg contracts.ContextPackage, subject string, entities int) {
	n := &why.Context
	nMain := min(len(result.Evidence), len(pkg.Evidence))
	main, extra := pkg.Evidence[:nMain], pkg.Evidence[nMain:]
	type source struct{ authority, count int }
	groups := map[string]*source{}
	facts := 0
	for _, e := range main {
		if e.Kind != contracts.EvidenceFact {
			continue
		}
		facts++
		g := groups[e.Source.SourceType]
		if g == nil {
			g = &source{authority: e.Authority}
			groups[e.Source.SourceType] = g
		}
		g.count++
	}
	n.Reason("%sgói ngữ cảnh: %d bằng chứng (%d dữ liệu nguồn, %d bằng chứng khác của truy vấn, %d bằng chứng đi kèm thêm sau truy vấn)",
		prefix, len(pkg.Evidence), facts, len(main)-facts, len(extra))
	for _, src := range slices.SortedFunc(maps.Keys(groups), func(x, y string) int {
		if c := groups[x].authority - groups[y].authority; c != 0 {
			return c
		}
		return strings.Compare(x, y)
	}) {
		role := "không tạo nên con số chính (nguồn thẩm quyền thấp hơn, hoặc lệch)"
		switch {
		case slices.Contains(result.UsedSources, src):
			role = "tạo nên con số chính"
		case result.NeedsReview:
			role = "chưa được chọn: các nguồn ngang thẩm quyền nói khác nhau, chờ người duyệt"
		}
		n.Reason("%snguồn %s (thẩm quyền %d): %d bằng chứng — %s", prefix, src, groups[src].authority, groups[src].count, role)
	}
	for _, e := range main {
		if e.Kind == contracts.EvidenceCalculation {
			n.Reason("%s%s: phép tính của con số chính (%s)", prefix, e.ID, e.Formula)
		}
	}
	for _, e := range extra {
		switch {
		case e.Kind == contracts.EvidenceQuestion:
			vals := make([]string, len(e.Values))
			for i, v := range e.Values {
				vals[i] = contracts.Quote(valueText(v))
			}
			n.Reason("%s%s: con số do người hỏi nêu (%s) — nguồn là chính câu hỏi, không phải số liệu", prefix, e.ID, strings.Join(vals, ", "))
		case e.ID == "count:conflicts":
			n.Reason("%s%s: đếm số chỗ lệch không kê ra trong câu", prefix, e.ID)
		case e.Kind == contracts.EvidenceCalculation:
			n.Reason("%s%s: chỉ số khác cùng kỳ, kê thêm (không phải chỉ số được hỏi)", prefix, e.ID)
		}
	}
	n.Reason("%squyền: phạm vi dữ liệu đã kiểm ở S5 cho thực thể gốc %s; truy vấn chạy trên %d thực thể (gốc và con cháu), không lọc bằng chứng theo quyền sau truy vấn", prefix, subject, entities)
	for _, c := range result.Conflicts {
		for _, o := range c.Others {
			why := "thẩm quyền thấp hơn nguồn được chọn " + c.ChosenID
			if c.NeedsReview {
				why = "ngang thẩm quyền với nguồn còn lại — không chọn được, chờ người duyệt"
			}
			n.Reject("%snguồn %s = %s %s không được chọn làm con số chính (xung đột %s, chính sách %s): %s", prefix, o.SourceType, o.Number, o.Unit, c.Key, c.Policy, why)
		}
	}
}

// noteRetrieved ghi S7 của route tri thức ngay sau khi truy xuất: số đoạn người hỏi được đọc, và nói rõ quyền đọc do chặng truy xuất kiểm.
func noteRetrieved(why *traces.Why, prefix string, ev []contracts.Evidence, k int) {
	n := &why.Context
	if len(ev) == 0 {
		n.Reason("%struy xuất trả 0 đoạn người hỏi được đọc (không có đoạn khớp, hoặc người hỏi không có quyền đọc đoạn nào — chặng truy xuất không phân biệt hai trường hợp)", prefix)
	} else {
		ids := make([]string, len(ev))
		for i, e := range ev {
			ids[i] = e.ID + " (" + e.Source.Name + ")"
		}
		n.Reason("%sđã truy xuất %d đoạn người hỏi được đọc (tối đa %d): %s", prefix, len(ev), k, strings.Join(ids, ", "))
	}
	n.Reason("%squyền: ACL tài liệu do chặng truy xuất kiểm, chỉ đoạn người hỏi được đọc được trả về (số đoạn bị loại vì quyền không xuất khỏi chặng truy xuất)", prefix)
}

// noteCited ghi S7 của route tri thức sau khi kiểm đạt: đoạn nào được claim của bản nháp dẫn, đoạn nào đã truy xuất mà không được dẫn.
func noteCited(why *traces.Why, prefix string, ev []contracts.Evidence, cited []contracts.Citation) {
	n := &why.Context
	ids := make([]string, len(cited))
	for i, c := range cited {
		ids[i] = c.EvidenceID
	}
	n.Reason("%sdẫn nguồn: %d đoạn được claim của bản nháp dẫn: %s", prefix, len(cited), strings.Join(ids, ", "))
	for _, e := range ev {
		if !slices.Contains(ids, e.ID) {
			n.Reject("%s%s: đã truy xuất nhưng không được claim nào dẫn nên không trích dẫn", prefix, e.ID)
		}
	}
}

// gapMeaning giải nghĩa từng chiều mà cổng phán quyết (verdictGap) báo là chưa giải được.
var gapMeaning = map[string]string{
	"metric":   "chỉ số không do người hỏi nêu rõ (ý định chung theo quy ước của pack), hay có cụm chỉ số / lượng khác ở ngữ cảnh có thể là chỉ số của con số",
	"period":   "mốc thời gian: câu không nêu mốc (kỳ mặc định của pack) hoặc nêu nhiều mốc",
	"accent":   "câu không dấu: bộ chuẩn hoá khớp bỏ dấu nên nhiều chữ trùng nhau",
	"marker":   "nhiều cụm xác nhận là nhiều claim, một phán quyết không trả lời đủ",
	"subject":  "chủ thể của con số chưa giải: có tên khác chủ thể, bí danh mơ hồ chưa có bằng chứng dương, hay con số chưa gắn vào chủ thể",
	"value":    "con số xác nhận chưa đọc chắc: có số khác, thiếu đơn vị, ngoài miền của chỉ số, đọc bằng chữ mơ hồ hay nhắc lại sau mệnh đề khác",
	"source":   "nguồn: tên nguồn trong câu không khớp nguồn đã tạo nên con số trả lời, hay có cụm nguồn khác ở ngữ cảnh",
	"residual": "phần chữ hay ký hiệu chưa được giải thích (từ lạ, '~', '>', '%', '/')",
}

// gate gom đầu vào và kết quả của cổng phán quyết cho noteGate.
type gate struct {
	in         contracts.NormalizedInput
	metric     string
	base       bool // chỉ số là chỉ số GỐC (có miền và đơn vị chuẩn) — chỉ số dẫn xuất / phép tính không có cổng phán quyết
	confirming bool // câu có cụm xác nhận của pack
	confirm    bool // câu xác nhận đúng MỘT con số đổi được về đơn vị của chỉ số
	compare    bool // câu so sánh hai nghĩa
	withheld   string
	tmpl       string
	claimed    contracts.Value
	total      *contracts.Value
	subject    string
	used       []string
}

// noteGate ghi S9: cổng phán quyết của câu xác nhận — phán quyết hay không, và vì sao.
func (a *Ask) noteGate(why *traces.Why, g gate) {
	n := &why.Validate
	word := func(tmpl string) string { // chữ mở đầu của mẫu câu trong pack ("Đúng.", "Không.")
		first, _, _ := strings.Cut(a.RT.Templates[tmpl], " ")
		return fmt.Sprintf("%q (%s)", first, tmpl)
	}
	switch {
	case !g.confirming:
		n.Reason("cổng phán quyết: không áp dụng — câu không có cụm xác nhận của pack, nên chỉ trả số")
	case !g.base:
		n.Reason("cổng phán quyết: không áp dụng — %s là chỉ số dẫn xuất (phép tính), không có miền và đơn vị chuẩn để xác nhận một con số, nên chỉ trả số", g.metric)
	case !g.confirm:
		n.Reason("cổng phán quyết: không áp dụng — câu có cụm xác nhận nhưng không có đúng một con số đổi được về đơn vị của chỉ số, nên chỉ trả số")
	case g.compare:
		n.Reason("cổng phán quyết: không áp dụng — câu so sánh hai nghĩa số liệu, nên không có một con số để xác nhận")
	default:
		inputs := fmt.Sprintf("đầu vào của cổng: chủ thể %s · kỳ đọc được %d · cụm xác nhận %d · câu có dấu %v · nguồn đã tạo nên con số %s · con số xác nhận %s %s",
			g.subject, len(readPeriods(g.in)), markerCount(g.in.Raw, a.RT.ConfirmMarkers), normalization.HasDiacritic(g.in.Raw),
			strings.Join(g.used, "|"), contracts.Quote(displayNumber(g.claimed.Number)), a.unitLabel(g.claimed.Unit))
		if g.withheld != "" {
			n.Reason("cổng phán quyết: không phán quyết — chiều %q chưa giải được: %s", g.withheld, gapMeaning[g.withheld])
			n.Reason("%s", inputs)
			n.Reason("chỉ trả số kèm câu giữ phán quyết (confirm_withheld)")
			n.Reject("phán quyết %s: bị loại — chiều %q chưa giải", word("confirm_yes"), g.withheld)
			n.Reject("phán quyết %s: bị loại — chiều %q chưa giải", word("confirm_no"), g.withheld)
			return
		}
		n.Reason("cổng phán quyết: được phán quyết — mọi chiều của claim (chủ thể, chỉ số, kỳ, giá trị và đơn vị, nguồn) đã giải bằng bằng chứng dương trong câu và mọi chữ của câu được giải thích")
		n.Reason("%s", inputs)
		other := "confirm_no"
		if g.tmpl == "confirm_no" {
			other = "confirm_yes"
		}
		n.Reason("so sánh bằng tuyệt đối số người hỏi %s với số có thẩm quyền %s → %s", contracts.Quote(displayNumber(g.claimed.Number)), displayNumber(g.total.Number), word(g.tmpl))
		n.Reject("phán quyết %s: bị loại — kết quả so sánh bằng tuyệt đối chọn %s", word(other), g.tmpl)
		n.Reject("không phán quyết (confirm_withheld): bị loại — cổng không thiếu chiều nào")
	}
}

// draftHeadRunes là số ký tự tối đa của đoạn đầu bản nháp in ở S8: đủ để người đọc vết thấy bản nháp mở đầu bằng chữ gì, không đủ để vết phình theo độ
// dài bản nháp (bản nháp của model có thể dài cả trang).
const draftHeadRunes = 80

// textDigest trả số ký tự (rune) và băm ngắn (12 ký tự hex đầu của SHA-256) của một đoạn chữ: dấu vân tay để người đọc vết thấy bước nào đổi chữ — bản nháp
// ở S8 và câu giao ở S10 trùng băm thì không bước nào ở giữa đổi một chữ. Tính trên chữ ĐÃ qua traces.Sanitize (đúng dạng chữ nằm trong vết bền, nên số ký
// tự khớp chữ người đọc thấy dù câu có đoạn bị che) và đã bỏ khoảng trắng đầu / cuối (mẫu câu cắt chúng khi dựng câu giao).
func textDigest(s string) (runes int, hash string) {
	clean := strings.TrimSpace(traces.Sanitize(s))
	sum := sha256.Sum256([]byte(clean))
	return utf8.RuneCountInString(clean), hex.EncodeToString(sum[:])[:12]
}

// draftHead trả đoạn đầu của bản nháp (tối đa draftHeadRunes ký tự, kèm "…" khi cắt). Làm sạch TRƯỚC khi cắt: bí mật nằm vắt qua chỗ cắt ("sk-" rồi vài ký
// tự) không còn khớp mẫu che của traces.Sanitize sau khi cắt, nên mảnh đầu của nó sẽ lọt vào vết.
func draftHead(s string) string {
	clean := strings.TrimSpace(traces.Sanitize(s))
	if r := []rune(clean); len(r) > draftHeadRunes {
		return string(r[:draftHeadRunes]) + "…"
	}
	return clean
}

// noteDraft ghi S8: bản nháp NHƯ ĐƯỢC SINH — cách sinh (how), số ký tự, băm ngắn và đoạn đầu (nằm trong cặp “…” để Why.WithholdQuoted che được khi câu hỏi
// mang bí mật). Gọi ngay sau khi bản nháp có chữ, TRƯỚC khi S9 kiểm: người đọc vết thấy chính thứ S9 đã nhận. Chỉ ghi, không đọc lại để quyết gì.
func (a *Ask) noteDraft(why *traces.Why, prefix, how, text string) {
	if a.noWhy {
		return
	}
	n := &why.Generate
	runes, hash := textDigest(text)
	n.Reason("%sbản nháp %s: %d ký tự · băm %s · đầu %s", prefix, how, runes, hash, contracts.Quote(draftHead(text)))
}

// noteTemplateDraft ghi S8 cho câu dựng bằng mẫu câu của tenant (route số liệu, hỏi lại, từ chối, chuyển người): tmpl là mẫu THẬT đã dùng, có thể khác
// RoutePlan.Template (câu xác nhận dùng confirm_yes / confirm_no, truy vấn không ra dữ liệu dùng no_data).
func (a *Ask) noteTemplateDraft(why *traces.Why, prefix, tmpl, text string) {
	a.noteDraft(why, prefix, "dựng bằng mẫu câu "+tmpl+" (không gọi model)", text)
}

// noteModelDraft ghi S8 cho bản nháp do model diễn đạt (model là tên do gateway khai trong contracts.Draft.Model).
func (a *Ask) noteModelDraft(why *traces.Why, prefix, model, text string) {
	if model == "" {
		model = "(không khai tên)"
	}
	a.noteDraft(why, prefix, "do model "+model+" diễn đạt", text)
}
