#!/usr/bin/env bash
# Danh mục đột biến cho vòng sửa tầng ý định S4 (ADR-0010 bất biến (a), D-OP-15 = a2 kèm 2 điều kiện): câu có thực thể + thuật ngữ mà
# không từ khoá nào khớp không phải ngoài phạm vi — trả lời theo ý định chung kèm câu giả định khi câu không nói gì thêm, còn lại hỏi lại
# nghĩa số liệu, chỉ kể ý định người hỏi có quyền. Mỗi dòng: m <tệp> <chuỗi gốc> <chuỗi đột biến> <nhãn>. Kỳ vọng: mọi dòng "BỊ BẮT".
cd "$(dirname "$0")/../.." || exit 1
m() { scripts/mutate.sh "$1" "$2" "$3" "$4"; echo "rc=$?"; }
export PKGS="./intelligence/intent/ ./intelligence/router/ ./core/pipeline/ ./core/tenant/ ./tenants/construction/"
I=intelligence/intent/intent.go
O=intelligence/router/router.go
A=core/pipeline/ask.go
m $I 'if len(partial) > 0 || !s.scopeSignal(in, res) {' 'if true {' 'S1 tắt nhánh (a): không khớp ý định vẫn là ngoài phạm vi'
m $I 'if len(partial) > 0 || !s.scopeSignal(in, res) {' 'if len(partial) > 0 {' 'S2 bỏ tín hiệu phạm vi: câu không thực thể / không thuật ngữ cũng hỏi lại'
m $I '	if len(res.Entities)+len(res.Ambiguous) == 0 {
		return false
	}' '	if false {
		return false
	}' 'S3 tín hiệu phạm vi không cần thực thể'
m $I '	if len(res.Entities)+len(res.Ambiguous) == 0 {' '	if len(res.Entities) == 0 {' 'S4 tín hiệu phạm vi bỏ bí danh mơ hồ'
m $I '		return slices.ContainsFunc(s.defs, func(d Def) bool { return slices.Contains(d.Terms, sp.Value.Text) })' '		return true' 'S5 tín hiệu phạm vi không cần thuật ngữ của một ý định'
m $I '		if len(d.Terms) > 0 && !d.Weak && contracts.RouteKind(d.Route) != contracts.RouteRAG {' '		if len(d.Terms) > 0 && contracts.RouteKind(d.Route) != contracts.RouteRAG {' 'S6 hỏi lại kể cả ý định chung'
# S7 — ý định tri thức (RAG) của pack hiện có không khai thuật ngữ nên đã bị loại bởi len(d.Terms) > 0: tương đương trên pack cầu đường;
# giữ điều kiện vì nghĩa của "hỏi lại nghĩa SỐ LIỆU" (pack khác có thể có ý định tri thức đòi thuật ngữ)
# m $I '		if len(d.Terms) > 0 && !d.Weak && contracts.RouteKind(d.Route) != contracts.RouteRAG {' '		if len(d.Terms) > 0 && !d.Weak {' 'S7 hỏi lại kể cả ý định tri thức'
m $I '		if len(d.Terms) > 0 && !d.Weak && contracts.RouteKind(d.Route) != contracts.RouteRAG {' '		if !d.Weak && contracts.RouteKind(d.Route) != contracts.RouteRAG {' 'S8 hỏi lại kể cả ý định không đòi thuật ngữ (chào hỏi)'
m $I '	if !slices.ContainsFunc(s.opt.FocusMarkers, func(m string) bool { return len(normalization.PhraseSpans(raw, m)) > 0 }) {' '	if false {' 'S9 a2 không đòi cụm xác nhận'
# S10 — pack cầu đường tương đương (một ý định yếu, có khai chỉ số); bắt bằng pack tổng hợp TestYDinhChungKhongChiSoThiHoiLai
m $I '	if len(gen) != 1 || gen[0].MetricByTerm[term] == "" {' '	if len(gen) == 0 {' 'S10 a2 nhận ý định chung không có chỉ số cho thuật ngữ'
m $I '			if d.MetricByTerm[term] == metric {' '			if true {' 'S11 mọi từ khoá số liệu đều là "của ý định chung" ("tiêu" của "tiêu thụ" thành giải thích được)'
m $I '		if d.Weak || len(d.Terms) == 0 || contracts.RouteKind(d.Route) == contracts.RouteRAG {
			continue
		}
		for _, k := range d.Keywords {' '		if len(d.Terms) == 0 || contracts.RouteKind(d.Route) == contracts.RouteRAG {
			continue
		}
		for _, k := range d.Keywords {' 'S12 từ khoá của chính ý định chung ("hết" của "hết bao nhiêu") thành giải thích được'
m $I '		if whole(p, other) && !whole(p, own) {' '		if whole(p, other) && !whole(p, own) && false {' 'S13 từ trung tính là mảnh từ khoá của chỉ số khác ("đo") vẫn được tha'
m $I '		if !slices.ContainsFunc(own, func(c string) bool {' '		if false && !slices.ContainsFunc(own, func(c string) bool {' 'S14 a2 bỏ kiểm chữ còn lại (chữ lạ "xài", "hao" vẫn trả số kho xuất)'
m $I '		cover(e.Start, e.End)
		types = append(types, e.Type)' '		types = append(types, e.Type)' 'S15 a2 không che tên thực thể'
m $I '	for _, m := range res.Ambiguous {
		cover(m.Start, m.End)
	}' '	for _, m := range res.Ambiguous[:0] {
		cover(m.Start, m.End)
	}' 'S16 a2 không che bí danh mơ hồ'
m $I '			if ws, we, ok := normalization.PeriodWordBefore(raw, sp.Start); ok {
				cover(ws, we)
			}
		case contracts.ValueTerm:' '			if ws, we, ok := normalization.PeriodWordBefore(raw, sp.Start); ok && false {
				cover(ws, we)
			}
		case contracts.ValueTerm:' 'S17 a2 không che "ngày" đứng trước mốc số'
m $I '		case contracts.ValueNumber: // số trần không đứng cạnh "dầu" nên không được gắn đơn vị: "Dầu xuất hôm nay của EX-03 là 3850 …"
			cover(sp.Start, sp.End)' '		case contracts.ValueNumber: // số trần không đứng cạnh "dầu" nên không được gắn đơn vị: "Dầu xuất hôm nay của EX-03 là 3850 …"
			_ = sp' 'S18 a2 không che số trần không đứng cạnh "dầu" (soát S4 vòng 2, F2: nhánh từng bị bỏ nhầm là mã chết)'
m $I '	for _, p := range s.opt.FocusMarkers {
		for _, sp := range normalization.PhraseSpans(raw, p) {
			cover(sp[0], sp[1])
		}
	}' '	for _, p := range s.opt.FocusMarkers[:0] {
		for _, sp := range normalization.PhraseSpans(raw, p) {
			cover(sp[0], sp[1])
		}
	}' 'S19 a2 không che cụm xác nhận'
m $O '	case len(cls.NeedMeaning) > 0:' '	case false:' 'S20 router bỏ nhánh hỏi lại nghĩa số liệu'
m $A '		return d.Capability != "" && !a.RT.Policy.Allows(pr, d.Capability)' '		return d.Capability != "" && false' 'S21 hỏi lại kể cả ý định người hỏi không có quyền'
m $A 'a.intentOptions(slices.Concat(cls.Ambiguous, cls.NeedTerm, cls.NeedMeaning))' 'a.intentOptions(slices.Concat(cls.Ambiguous, cls.NeedTerm))' 'S22 câu hỏi lại không kể các nghĩa số liệu'
m $A '		vars["assumption_note"] = " " + def.Assumption' '		_ = def.Assumption' 'S23 câu trả lời theo ý định chung không nói giả định'
m $I '	return gen[0], meaning' '	return gen[0], meaning || true' 'S24 a2 không đòi chữ nêu nghĩa (câu xác nhận không nêu chỉ số vẫn trả số kho xuất, trái BR-FUEL-02)'
m $O '			if !pol.InScope(pr, lineage(e)) {' '			if false {' 'S25 hỏi lại nghĩa số liệu cho cả thực thể ngoài phạm vi dữ liệu'
m $I 'return Result{NeedMeaning: s.meaningOptions(), Mentioned: res.Entities, Candidates: cands}' 'return Result{NeedMeaning: s.meaningOptions(), Candidates: cands}' 'S26 câu hỏi lại nghĩa số liệu không mang thực thể đã giải (bỏ kiểm phạm vi)'
m $I 'return Result{NeedMeaning: s.meaningOptions(), Mentioned: res.Entities, Candidates: cands}' 'return Result{NeedMeaning: s.meaningOptions(), Mentioned: res.Entities, Candidates: cands[:0]}' 'S27 câu hỏi lại nghĩa số liệu không kể ứng viên của bí danh mơ hồ'
# Soát S4 vòng 1 (S4R-01, S4R-06, S4R-08): a2 chỉ che thuật ngữ của ý định chung và lượng cùng chiều đo; khớp chữ theo cả câu; pack kiểm chỗ
# cho câu giả định.
m $I '			if slices.Contains(gen[0].Terms, sp.Value.Text) || slices.Contains(types, sp.Value.Text) {' '			if true {' 'S28 a2 che mọi thuật ngữ ("đơn giá", "giờ máy" thành trung tính)'
m $I '			if s.opt.QuantityFits != nil && s.opt.QuantityFits(metric, sp.Value) {' '			if true {' 'S29 a2 che mọi lượng có đơn vị ("7 tấn" thành trung tính)'
m $I '			if s.opt.QuantityFits != nil && s.opt.QuantityFits(metric, sp.Value) {' '			if false {' 'S30 a2 không che lượng cùng chiều đo ("4 khối" thành chữ lạ)'
m $I '		if !slices.ContainsFunc(own, func(c string) bool {' '		if !whole(raw[i:j], own) && !slices.ContainsFunc(own, func(c string) bool {' 'S31 a2 khớp chữ theo từng chữ riêng ("duoc" không dấu thành "được")'
P=core/tenant/pack.go
m $P '			if text, ok := tf.Templates[name]; ok && !strings.Contains(text, "{assumption_note}") {' '			if text, ok := tf.Templates[name]; ok && false && !strings.Contains(text, "{assumption_note}") {' 'S32 pack không kiểm chỗ cho câu giả định'
m $P '					_, err := rt.Normalizer.Convert(v, def.Unit)
					return err == nil' '					_, err := rt.Normalizer.Convert(v, def.Unit)
					return err == nil || true' 'S33 mọi lượng đều cùng chiều đo với chỉ số'
m $P '					_, err := rt.Normalizer.Convert(v, def.Unit)
					return err == nil' '					_, err := rt.Normalizer.Convert(v, def.Unit)
					return err == nil && false' 'S34 không lượng nào cùng chiều đo với chỉ số'
# Soát S4 vòng 2 (F1, F3, F4): chữ chỉ loại của thực thể đã giải thuộc tên; đếm ý định chung trước khi đọc; mọi bí danh của vật tư "dầu
# không phải nhiên liệu" có test.
m $I '		types = append(types, e.Type)
	}' '	}' 'S35 a2 không che chữ chỉ loại ("dự án Cầu A" bị hỏi lại)'
m $I 'slices.Contains(gen[0].Terms, sp.Value.Text) || slices.Contains(types, sp.Value.Text)' 'slices.Contains(gen[0].Terms, sp.Value.Text) || (s.opt.Measurable != nil && !s.opt.Measurable(sp.Value.Text))' 'S36 a2 che mọi thuật ngữ không đo được, bất kể loại của thực thể ("dự án EX-03")'
m $I '	if len(gen) != 1 || gen[0].MetricByTerm[term] == "" {
		return Def{}, false
	}
	metric := gen[0].MetricByTerm[term]' '	metric := gen[0].MetricByTerm[term]
	if len(gen) != 1 || metric == "" {
		return Def{}, false
	}' 'S37 đọc ý định chung đầu tiên trước khi đếm (sập khi pack không có ý định chung cho thuật ngữ)'
L=tenants/construction/pack/lexicon.yaml
m $L 'aliases: [dầu nhớt, nhớt, dầu thủy lực' 'aliases: [nhớt, dầu thủy lực' 'S38 bỏ bí danh "dầu nhớt"'
m $L 'dầu nhớt, nhớt, dầu thủy lực, dầu thuỷ lực' 'dầu nhớt, dầu thủy lực, dầu thuỷ lực' 'S39 bỏ bí danh "nhớt"'
m $L 'nhớt, dầu thủy lực, dầu thuỷ lực, dầu hộp số' 'nhớt, dầu thuỷ lực, dầu hộp số' 'S40 bỏ bí danh "dầu thủy lực"'
m $L 'dầu thủy lực, dầu thuỷ lực, dầu hộp số' 'dầu thủy lực, dầu hộp số' 'S41 bỏ bí danh "dầu thuỷ lực"'
m $L 'dầu thuỷ lực, dầu hộp số, dầu bôi trơn' 'dầu thuỷ lực, dầu bôi trơn' 'S42 bỏ bí danh "dầu hộp số"'
m $L 'dầu hộp số, dầu bôi trơn, mỡ bôi trơn' 'dầu hộp số, mỡ bôi trơn' 'S43 bỏ bí danh "dầu bôi trơn"'
m $L 'dầu bôi trơn, mỡ bôi trơn, dầu ăn' 'dầu bôi trơn, dầu ăn' 'S44 bỏ bí danh "mỡ bôi trơn"'
m $L 'mỡ bôi trơn, dầu ăn, dầu gội' 'mỡ bôi trơn, dầu gội' 'S45 bỏ bí danh "dầu ăn"'
m $L 'dầu ăn, dầu gội, dầu gió' 'dầu ăn, dầu gió' 'S46 bỏ bí danh "dầu gội"'
m $L 'dầu gội, dầu gió, dầu thơm' 'dầu gội, dầu thơm' 'S47 bỏ bí danh "dầu gió"'
m $L 'dầu gió, dầu thơm]' 'dầu gió]' 'S48 bỏ bí danh "dầu thơm"'
m $I '	if len(gen) != 1 || gen[0].MetricByTerm[term] == "" {' '	if len(gen) != 1 {' 'S49 ý định chung không khai chỉ số cho thuật ngữ vẫn được chọn (soát S4 vòng 3, phát hiện 4)'
# Vòng sửa S4 23/10 (ADR-0010, D-AI-02 = a, D-OP-18 = c, D-OP-11 = b). D-OP-11 = b: tập ứng viên hỏi lại chỉ là ứng viên của bí danh mơ hồ.
m $I '	if slices.ContainsFunc(out, func(e contracts.EntityRef) bool { return viable[e.ID] }) {
		out = slices.DeleteFunc(out, func(e contracts.EntityRef) bool { return !viable[e.ID] })
	}' '	_ = viable' 'S70 hỏi lại kể cả thực thể đã giải khi có bí danh mơ hồ (cách gộp cũ)'
m $I '	if slices.ContainsFunc(out, func(e contracts.EntityRef) bool { return viable[e.ID] }) {' '	if true {' 'S71 hai thực thể đều đã giải cũng bị lọc (danh sách hỏi lại rỗng)'
m $I '			viable[c.ID] = true
' '' 'S72 ứng viên của bí danh không được đánh dấu (bộ lọc không bao giờ chạy)'
# N1 (bất biến (c)): hoà ý định / thiếu thuật ngữ mà câu còn bí danh mơ hồ thì câu hỏi lại mang ứng viên của bí danh.
m $I '	if (len(r.Ambiguous) > 0 || len(r.NeedTerm) > 0) && len(r.Candidates) == 0 {' '	if false {' 'S73 hoà ý định / thiếu thuật ngữ bỏ rơi ứng viên của bí danh mơ hồ'
m $I '	if (len(r.Ambiguous) > 0 || len(r.NeedTerm) > 0) && len(r.Candidates) == 0 {' '	if len(r.Ambiguous) > 0 && len(r.Candidates) == 0 {' 'S74 chỉ hoà ý định mang ứng viên, thiếu thuật ngữ thì bỏ rơi'
m tenants/construction/pack/templates.yaml '  clarify_intent: "Bạn muốn hỏi {options}? {candidates}"' '  clarify_intent: "Bạn muốn hỏi {options}?"' 'S75 mẫu hỏi lại nghĩa số liệu không kể ứng viên (người dùng không thấy máy nào cần chọn)'
# C5 (bất biến (b), vai mệnh đề): mệnh đề ngữ cảnh nêu con số, đứng trước mệnh đề hỏi có chỉ số riêng, không quyết định ý định.
m $I 'counts, firsts := keywordHits(s.defs, maskRanges(in.Raw, s.premiseRanges(in, res)), s.opt.IgnorePhrases)' 'counts, firsts := keywordHits(s.defs, in.Raw, s.opt.IgnorePhrases)' 'S76 từ khoá của mệnh đề ngữ cảnh rò sang ý định (không che)'
m $I '	if len(out) == 0 || !s.ownMetric(raw[cl[q][0]:cl[q][1]], hasFigure(in, res, cl[q][0], cl[q][1])) {' '	if len(out) == 0 {' 'S77 che mệnh đề trước cả khi mệnh đề hỏi chỉ là cái đuôi không có chỉ số riêng'
m $I '		if hasFigure(in, res, cl[i][0], cl[i][1]) {
			out = append(out, cl[i])' '		if true {
			out = append(out, cl[i])' 'S78 che cả mệnh đề dẫn đề không nêu con số'
m $I '	if q > 0 && s.onlyMarkers(raw[cl[q][0]:cl[q][1]], markers) {
		q--
	}' '	if q > 0 && s.onlyMarkers(raw[cl[q][0]:cl[q][1]], markers) {
		_ = q
	}' 'S79 mệnh đề hỏi chỉ còn cụm xác nhận không áp cho mệnh đề liền trước'
m $I '			if hasOwnFigure && missing < len(strings.Fields(k)) && !s.neutralOnly(k, clause) {' '			if missing < len(strings.Fields(k)) && !s.neutralOnly(k, clause) {' 'S80 chữ của từ khoá đứng với nhãn ("phiếu số 912") không cần con số riêng vẫn là chỉ số riêng'
m $I '	for _, r := range s.premiseRanges(in, res) { // mệnh đề ngữ cảnh: không phải điều đang hỏi nên không cần giải thích (bất biến (b))
		cover(r[0], r[1])
	}' '' 'S81 a2 đòi giải thích cả chữ của mệnh đề ngữ cảnh'
m $I '	markers := append(slices.Clone(s.opt.FocusMarkers), s.opt.QuantityMarkers...)
	q := slices.IndexFunc(cl' '	markers := slices.Clone(s.opt.FocusMarkers)
	q := slices.IndexFunc(cl' 'S82 cụm hỏi lượng ("bao nhiêu") không đánh dấu mệnh đề hỏi'
# C4: câu nêu đúng hai chỉ số khác nhau kèm dấu hiệu so sánh là phép so sánh đúng cặp người hỏi nêu; ý định dẫn xuất chỉ trả cặp mặc định khi
# mọi chỉ số câu nêu nằm trong cặp ấy.
m $I '	if len(named) == 0 || len(derived) == 0 && !s.comparing(in.Raw) {' '	if len(named) == 0 || len(derived) == 0 {' 'S83 cụm so sánh ("nhiều hơn") không còn là dấu hiệu so sánh'
m $I '		if inside {
			return Result{}, false
		}' '		if false {
			return Result{}, false
		}' 'S84 ý định dẫn xuất không bao giờ trả cặp mặc định'
m $I '			inside = inside && slices.ContainsFunc(derived' '			inside = inside || slices.ContainsFunc(derived' 'S85 chỉ cần MỘT chỉ số nằm trong cặp mặc định là trả cặp mặc định (chỉ số ngoài cặp bị bỏ thầm)'
m $I '	if len(named) == 2 && len(distinct) == 2 {' '	if len(named) >= 2 {' 'S86 so sánh cả khi câu nêu ba chỉ số (lấy hai chỉ số đầu)'
m $I '		if first[right.Name] < first[left.Name] {
			left, right = right, left
		}
		r := s.fill(left, in, res, now, implicit)
		r.Intent.Slots["compare_metric"]' '		if false {
			left, right = right, left
		}
		r := s.fill(left, in, res, now, implicit)
		r.Intent.Slots["compare_metric"]' 'S87 vế so sánh theo thứ tự khai báo ý định, không theo thứ tự nhắc'
m $I '	if len(derived) == 0 {
		return Result{}, false
	}
	var names []string' '	var names []string' 'S88 câu so sánh không phải hai chỉ số cũng hỏi lại (một chỉ số + "nhiều hơn")'
m core/tenant/pack.go '					return d.Left, d.Right, ok' '					return d.Left, d.Right, ok && false' 'S89 pack không khai cặp mặc định của chỉ số dẫn xuất'
# C3 (bất biến (a)): tín hiệu phạm vi không cần chữ "dầu"; lời chào là chữ "chào"; chữ của cụm xác nhận và mã chứng từ là chữ trung tính của a2.
m $I '	return implicit != "" && s.volumeFigure(in) && s.metricWord(in.Raw)' '	return false' 'S90 câu không có chữ "dầu" luôn ngoài phạm vi'
m $I '	return implicit != "" && s.volumeFigure(in) && s.metricWord(in.Raw)' '	return s.volumeFigure(in) && s.metricWord(in.Raw)' 'S91 tín hiệu không cần thuật ngữ mặc định áp được ("dầu nhớt" bị hỏi lại)'
m $I '	return implicit != "" && s.volumeFigure(in) && s.metricWord(in.Raw)' '	return implicit != "" && s.metricWord(in.Raw)' 'S92 tín hiệu không cần lượng cùng chiều đo ("nhận 20 tấn" bị hỏi lại)'
m $I '	return implicit != "" && s.volumeFigure(in) && s.metricWord(in.Raw)' '	return implicit != "" && s.volumeFigure(in)' 'S93 tín hiệu không cần chữ của từ khoá chỉ số ("300 lít nước" bị hỏi lại)'
m $I '	if len(gen) == 0 && implicit != "" { // câu không nêu thuật ngữ' '	if false { // câu không nêu thuật ngữ' 'S94 a2 không dùng thuật ngữ mặc định (câu không có chữ "dầu" luôn hỏi lại)'
m $I '		case contracts.ValueCode: // mã chứng từ ("PX-0912") là nhãn, không mang nghĩa chỉ số
			cover(sp.Start, sp.End)' '		case contracts.ValueCode: // mã chứng từ ("PX-0912") là nhãn, không mang nghĩa chỉ số
			_ = sp' 'S95 a2 đòi giải thích cả mã chứng từ'
m $I '		neutral = append(neutral, strings.Fields(m)...)' '		_ = m' 'S96 chữ của cụm xác nhận ("đúng") không trung tính với a2'
m $I '		return len(sp) > 0 && sp[0][0] == 0 && sp[0][1] == len(w)
	})
}

// meaningOptions' '		return len(sp) > 0 && sp[0][0] == 0 && sp[0][1] == len(w) && false
	})
}

// meaningOptions' 'S98 từ trung tính ("được") không được coi là trung tính khi tìm chữ của từ khoá chỉ số'
m tenants/construction/pack/intents.yaml '    keywords: [chào, hello]' '    keywords: [xin chào, hello]' 'S97 lời chào chỉ là cụm "xin chào"'
# F6' (nợ S4 sau H4): số không tự viết đơn vị mà ngoài miền một lần đo của chỉ số không phải giá trị của nó.
m $I '			if s.inRange(metric, sp.Value) {
				cover(sp.Start, sp.End)' '			if true {
				cover(sp.Start, sp.End)' 'S99 a2 che mọi số trần, kể cả ngoài miền chỉ số ("4 củ")'
m $I ' && (!strings.Contains(sp.Rule, "unit_from_term") || s.inRange(metric, sp.Value)) {' ' {' 'S100 lượng chỉ có đơn vị nhờ "dầu" không bị kiểm miền ("4,1 triệu dầu")'
m $I '(!strings.Contains(sp.Rule, "unit_from_term") || s.inRange(metric, sp.Value))' 's.inRange(metric, sp.Value)' 'S101 kiểm miền cả lượng viết rõ đơn vị ("80000 L" của cả tháng bị hỏi lại)'
m core/tenant/pack.go '					return def.InRange(v.Number) == nil' '					return true' 'S102 không kiểm miền một lần đo ("4 củ", "dầu 4,1 triệu" thành lít)'
m core/tenant/pack.go '					return def.InRange(v.Number) == nil' '					return false' 'S103 mọi số đều ngoài miền ("4k", "3850" thành chữ lạ)'

