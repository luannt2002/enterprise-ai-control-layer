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
m $I 'if len(partial) > 0 || !s.scopeSignal(in, res, implicit) {' 'if true {' 'S1 tắt nhánh (a): không khớp ý định vẫn là ngoài phạm vi'
m $I 'if len(partial) > 0 || !s.scopeSignal(in, res, implicit) {' 'if len(partial) > 0 {' 'S2 bỏ tín hiệu phạm vi: câu không thực thể / không thuật ngữ cũng hỏi lại'
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
		for _, k := range d.Keywords {
			if d.MetricByTerm[term] == metric {' '		if len(d.Terms) == 0 || contracts.RouteKind(d.Route) == contracts.RouteRAG {
			continue
		}
		for _, k := range d.Keywords {
			if d.MetricByTerm[term] == metric {' 'S12 từ khoá của chính ý định chung ("hết" của "hết bao nhiêu") thành giải thích được'
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
m $I '			if s.inRange(metric, sp.Value) {
				cover(sp.Start, sp.End)
			}' '			_ = sp' 'S18 a2 không che số trần không đứng cạnh "dầu" (soát S4 vòng 2, F2: nhánh từng bị bỏ nhầm là mã chết)'
# S19 — tương đương trên pack cầu đường từ vòng s4a (23/10): chữ của cụm xác nhận đã được che theo TỪNG CHỮ (S96); vòng che theo cụm chỉ còn
# khác khi một chữ của cụm xác nhận cũng là mảnh từ khoá của chỉ số khác (pack cầu đường không có). Giữ vòng che theo cụm cho pack khác.
# m $I '	for _, p := range s.opt.FocusMarkers {
#		for _, sp := range normalization.PhraseSpans(raw, p) {
#			cover(sp[0], sp[1])
#		}
#	}' '	for _, p := range s.opt.FocusMarkers[:0] {
#		for _, sp := range normalization.PhraseSpans(raw, p) {
#			cover(sp[0], sp[1])
#		}
#	}' 'S19 a2 không che cụm xác nhận'
m $O '	case check("hỏi lại nghĩa số liệu", contracts.RouteClarification, len(cls.NeedMeaning) > 0,' '	case check("hỏi lại nghĩa số liệu", contracts.RouteClarification, false,' 'S20 router bỏ nhánh hỏi lại nghĩa số liệu'
m $A '		return d.Capability != "" && !a.RT.Policy.Allows(pr, d.Capability)' '		return d.Capability != "" && false' 'S21 hỏi lại kể cả ý định người hỏi không có quyền'
m $A 'a.intentOptions(slices.Concat(cls.Ambiguous, cls.NeedTerm, cls.NeedMeaning))' 'a.intentOptions(slices.Concat(cls.Ambiguous, cls.NeedTerm))' 'S22 câu hỏi lại không kể các nghĩa số liệu'
m $A '		vars["assumption_note"] = " " + def.Assumption' '		_ = def.Assumption' 'S23 câu trả lời theo ý định chung không nói giả định'
m $I '	if !meaning {' '	if !meaning && false {' 'S24 a2 không đòi chữ nêu nghĩa (câu xác nhận không nêu chỉ số vẫn trả số kho xuất, trái BR-FUEL-02)'
m $O '			out := !pol.InScope(pr, lineage(e))' '			out := false' 'S25 hỏi lại nghĩa số liệu cho cả thực thể ngoài phạm vi dữ liệu'
m $I 'return Result{NeedMeaning: opts, Mentioned: res.Entities, Candidates: cands}' 'return Result{NeedMeaning: opts, Candidates: cands}' 'S26 câu hỏi lại nghĩa số liệu không mang thực thể đã giải (bỏ kiểm phạm vi)'
m $I 'Mentioned: res.Entities, Candidates: aliasCandidates(res, w)}' 'Candidates: aliasCandidates(res, w)}' 'S26b câu hỏi lại vì nghĩa chưa hỗ trợ không mang thực thể đã giải (bỏ kiểm phạm vi)'
m $I 'return Result{NeedMeaning: opts, Mentioned: res.Entities, Candidates: cands}' 'return Result{NeedMeaning: opts, Mentioned: res.Entities, Candidates: cands[:0]}' 'S27 câu hỏi lại nghĩa số liệu không kể ứng viên của bí danh mơ hồ'
# Soát S4 vòng 1 (S4R-01, S4R-06, S4R-08): a2 chỉ che thuật ngữ của ý định chung và lượng cùng chiều đo; khớp chữ theo cả câu; pack kiểm chỗ
# cho câu giả định.
m $I '			if slices.Contains(gen[0].Terms, sp.Value.Text) || slices.Contains(types, sp.Value.Text) {' '			if true {' 'S28 a2 che mọi thuật ngữ ("đơn giá", "giờ máy" thành trung tính)'
m $I '			if s.opt.QuantityFits != nil && s.opt.QuantityFits(metric, sp.Value) && (!strings.Contains(sp.Rule, "unit_from_term") || s.inRange(metric, sp.Value)) {' '			if true {' 'S29 a2 che mọi lượng có đơn vị ("7 tấn" thành trung tính)'
m $I '			if s.opt.QuantityFits != nil && s.opt.QuantityFits(metric, sp.Value) && (!strings.Contains(sp.Rule, "unit_from_term") || s.inRange(metric, sp.Value)) {' '			if false {' 'S30 a2 không che lượng cùng chiều đo ("4 khối" thành chữ lạ)'
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
		return Def{}, false, fmt.Sprintf("không có đúng một ý định yếu đòi thuật ngữ của câu (có %d)", len(gen))
	}
	metric := gen[0].MetricByTerm[term]' '	metric := gen[0].MetricByTerm[term]
	if len(gen) != 1 || metric == "" {
		return Def{}, false, fmt.Sprintf("không có đúng một ý định yếu đòi thuật ngữ của câu (có %d)", len(gen))
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
m $I '	text := maskRanges(in.Raw, prem)' '	text := in.Raw' 'S76 từ khoá của mệnh đề ngữ cảnh rò sang ý định (không che)'
m $I '	if len(out) == 0 || !s.ownMetric(raw[cl[q][0]:cl[q][1]], hasFigure(in, res, cl[q][0], cl[q][1])) {' '	if len(out) == 0 {' 'S77 che mệnh đề trước cả khi mệnh đề hỏi chỉ là cái đuôi không có chỉ số riêng'
m $I '		if i != q && hasFigure(in, res, cl[i][0], cl[i][1]) {' '		if i != q {' 'S78 che cả mệnh đề dẫn đề không nêu con số'
m $I '	for i := range cl {
		// trước mệnh đề hỏi' '	for i := 0; i < q; i++ {
		// trước mệnh đề hỏi' 'S125 mệnh đề ngữ cảnh SAU câu hỏi ("…, biết đội báo 4000 L?") không bị che (S4AS-03, C5)'
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
m $I '		if s.comparing(text[c[0]:c[1]]) {' '		if false {' 'S83 từ so sánh bậc hơn / kém ("cao hơn", "kém") không còn là dấu hiệu so sánh'
m $I '		if inside {
			return Result{}, false
		}' '		if false {
			return Result{}, false
		}' 'S84 ý định dẫn xuất không bao giờ trả cặp mặc định'
m $I '			inside = inside && slices.ContainsFunc(derived' '			inside = inside || slices.ContainsFunc(derived' 'S85 chỉ cần MỘT chỉ số nằm trong cặp mặc định là trả cặp mặc định (chỉ số ngoài cặp bị bỏ thầm)'
m $I '	if len(named) == 2 && len(distinct) == 2 {' '	if len(named) >= 2 {' 'S86 so sánh cả khi câu nêu ba chỉ số (lấy hai chỉ số đầu)'
m $I '		if first[right.Name] < first[left.Name] {' '		if false {' 'S87 vế so sánh theo thứ tự khai báo ý định, không theo thứ tự nhắc'
m $I '	if len(derived) == 0 && len(distinct) < 3 {' '	if len(derived) == 0 && false {' 'S88 câu so sánh không phải hai chỉ số cũng hỏi lại (một chỉ số + từ so sánh)'
m core/tenant/pack.go '					return d.Left, d.Right, ok' '					return d.Left, d.Right, ok && false' 'S89 pack không khai cặp mặc định của chỉ số dẫn xuất'
# C3 (bất biến (a)): tín hiệu phạm vi không cần chữ "dầu"; lời chào là chữ "chào"; chữ của cụm xác nhận và mã chứng từ là chữ trung tính của a2.
m $I '	return implicit != "" && s.volumeFigure(in) && (s.confirming(in.Raw) || s.metricWord(in.Raw))' '	return false' 'S90 câu không có chữ "dầu" luôn ngoài phạm vi'
m $I '	return implicit != "" && s.volumeFigure(in) && (s.confirming(in.Raw) || s.metricWord(in.Raw))' '	return s.volumeFigure(in) && (s.confirming(in.Raw) || s.metricWord(in.Raw))' 'S91 tín hiệu không cần thuật ngữ mặc định áp được ("dầu nhớt" bị hỏi lại)'
m $I '	return implicit != "" && s.volumeFigure(in) && (s.confirming(in.Raw) || s.metricWord(in.Raw))' '	return implicit != "" && (s.confirming(in.Raw) || s.metricWord(in.Raw))' 'S92 tín hiệu không cần lượng cùng chiều đo ("nhận 20 tấn" bị hỏi lại)'
m $I '	return implicit != "" && s.volumeFigure(in) && (s.confirming(in.Raw) || s.metricWord(in.Raw))' '	return implicit != "" && s.volumeFigure(in) && s.metricWord(in.Raw)' 'S93 tín hiệu đòi chữ của từ khoá chỉ số như vòng 1 ("xài 3850 L đúng không" bị từ chối, S4AS-03 C3)'
m $I '	return implicit != "" && s.volumeFigure(in) && (s.confirming(in.Raw) || s.metricWord(in.Raw))' '	return implicit != "" && s.volumeFigure(in) && s.confirming(in.Raw)' 'S127 tín hiệu chỉ nhận cụm xác nhận, bỏ chữ của từ khoá chỉ số ("Hôm nay EX-03 xuất 3850 L?" bị từ chối)'
m $I '	if len(gen) == 0 && implicit != "" { // câu không nêu thuật ngữ' '	if false { // câu không nêu thuật ngữ' 'S94 a2 không dùng thuật ngữ mặc định (câu không có chữ "dầu" luôn hỏi lại)'
m $I '		case contracts.ValueCode: // mã chứng từ ("PX-0912") là nhãn, không mang nghĩa chỉ số
			cover(sp.Start, sp.End)' '		case contracts.ValueCode: // mã chứng từ ("PX-0912") là nhãn, không mang nghĩa chỉ số
			_ = sp' 'S95 a2 đòi giải thích cả mã chứng từ'
m $I '		neutral = append(neutral, strings.Fields(m)...)' '		_ = m' 'S96 chữ của cụm xác nhận ("đúng") không trung tính với a2'
m $I '	return slices.ContainsFunc(s.opt.NeutralWords, func(n string) bool {
		sp := normalization.PhraseSpans(w, n)
		return len(sp) > 0 && sp[0][0] == 0 && sp[0][1] == len(w)
	})' '	return slices.ContainsFunc(s.opt.NeutralWords, func(n string) bool {
		sp := normalization.PhraseSpans(w, n)
		return len(sp) > 0 && sp[0][0] == 0 && sp[0][1] == len(w) && false
	})' 'S98 từ trung tính ("được") không được coi là trung tính khi tìm chữ của từ khoá chỉ số'
m tenants/construction/pack/intents.yaml '    keywords: [chào, hello]' '    keywords: [xin chào, hello]' 'S97 lời chào chỉ là cụm "xin chào"'
# F6' (nợ S4 sau H4): số không tự viết đơn vị mà ngoài miền một lần đo của chỉ số không phải giá trị của nó.
m $I '			if s.inRange(metric, sp.Value) {
				cover(sp.Start, sp.End)' '			if true {
				cover(sp.Start, sp.End)' 'S99 a2 che mọi số trần, kể cả ngoài miền chỉ số ("4 củ")'
m $I ' && (!strings.Contains(sp.Rule, "unit_from_term") || s.inRange(metric, sp.Value)) {' ' {' 'S100 lượng chỉ có đơn vị nhờ "dầu" không bị kiểm miền ("4,1 triệu dầu")'
m $I '(!strings.Contains(sp.Rule, "unit_from_term") || s.inRange(metric, sp.Value))' 's.inRange(metric, sp.Value)' 'S101 kiểm miền cả lượng viết rõ đơn vị ("80000 L" của cả tháng bị hỏi lại)'
m core/tenant/pack.go '					return def.InRange(v.Number) == nil' '					return true' 'S102 không kiểm miền một lần đo ("4 củ", "dầu 4,1 triệu" thành lít)'
m core/tenant/pack.go '					return def.InRange(v.Number) == nil' '					return false' 'S103 mọi số đều ngoài miền ("4k", "3850" thành chữ lạ)'
# A2 (D-OP-18 = c): nghĩa hệ chưa hỗ trợ (tiền, dự báo, báo giá, cảnh báo, lệnh ghi, đánh giá) ⇒ hỏi lại và nói rõ nghĩa nào, không trả theo chỉ số còn lại.
m $I '	if len(s.opt.Unsupported) == 0 || len(res.Entities)+len(res.Ambiguous) == 0 {' '	if len(s.opt.Unsupported) == 0 {' 'S104 nghĩa chưa hỗ trợ không cần thực thể (câu "dự báo thời tiết" bị hỏi lại)'
m $I '		if counts[i] > 0 && contracts.RouteKind(d.Route) == contracts.RouteRAG {
			return nil' '		if counts[i] > 0 && contracts.RouteKind(d.Route) == contracts.RouteRAG && false {
			return nil' 'S105 câu hỏi quy trình về cảnh báo cũng bị hỏi lại'
m $I '				if !hasFigure(in, res, c[0], c[1]) || asked {' '				if asked {' 'S106 lệnh ghi không cần con số ("ghi nhận dầu" không có con số bị hỏi lại)'
m $I '				if !hasFigure(in, res, c[0], c[1]) || asked {' '				if !hasFigure(in, res, c[0], c[1]) || false && asked {' 'S107 câu xác nhận / hỏi lượng có "ghi nhận" cũng là lệnh ghi'
m $I '	return append(slices.Clone(s.opt.IgnorePhrases), s.cuePhrases()...)' '	return slices.Clone(s.opt.IgnorePhrases)' 'S108 cụm báo nghĩa không chiếm chỗ ("nhận" của "ghi nhận" khớp ý định kho xuất)'
m intelligence/router/router.go '		if len(cls.Unsupported) > 0 { // nghĩa hệ chưa hỗ trợ' '		if false { // nghĩa hệ chưa hỗ trợ' 'S110 router không nói nghĩa chưa hỗ trợ (hỏi lại như câu thiếu nghĩa)'
m core/pipeline/ask.go '		vars["unsupported_note"] = strings.Join(unsupported, " ")' '		_ = unsupported' 'S111 câu hỏi lại không nói nghĩa nào chưa hỗ trợ'
m $I '	for _, c := range s.cuePhrases() { // cụm báo nghĩa chưa hỗ trợ không phải lệnh khi câu là câu hỏi' '	for _, c := range s.cuePhrases()[:0] { // cụm báo nghĩa chưa hỗ trợ không phải lệnh khi câu là câu hỏi' 'S112 a2 đòi giải thích cả cụm "ghi nhận" trong câu hỏi'


# ---- vòng 2 (23/10, soát vòng 1): S4AS-01 — lệnh ghi lịch sự có "?" vẫn là lệnh
m $I '				asked := s.asking(ct) || strings.TrimSpace(next) != "" && s.onlyMarkers(next, markers)' '				asked := strings.HasSuffix(strings.TrimSpace(in.Raw), "?") || s.asking(ct) || strings.TrimSpace(next) != "" && s.onlyMarkers(next, markers)' 'S113 dấu "?" ở cuối câu làm mệnh đề thành câu hỏi (lệnh ghi lịch sự bị trả như truy vấn)'
m $I '	return slices.ContainsFunc(slices.Concat(s.opt.FocusMarkers, s.opt.QuantityMarkers), func(m string) bool { return normalization.ContainsPhrase(text, m) })
}' '	return slices.ContainsFunc(s.opt.FocusMarkers, func(m string) bool { return normalization.ContainsPhrase(text, m) })
}' 'S114 cụm hỏi lượng không đánh dấu câu hỏi ("ghi nhận … bao nhiêu" bị coi là lệnh)'

# ---- S4AS-02: nghĩa tiền theo cấu trúc và họ nghĩa
m $I '				if s.onlyMarkers(text[sp[1]:end], markers) {' '				if s.onlyMarkers(text[sp[1]:end], markers) || true {' 'S115 "bao nhiêu đồng" không đòi mệnh đề kết thúc sau đơn vị ("đồng hồ", "đồng nghiệp" thành tiền)'
m $I '		if why == "" && !measured && slices.ContainsFunc(' '		if why == "" && (measured || true) && slices.ContainsFunc(' 'S116 tín hiệu tiền không nhường câu có lượng cùng chiều đo ("3850 L … 25.000 đồng" bị hỏi lại tiền)'
m $I '		return slices.ContainsFunc(in.Spans, func(sp contracts.Span) bool { return inRange(sp) && string(sp.Kind) == arg })' '		return slices.ContainsFunc(in.Spans, func(sp contracts.Span) bool { return inRange(sp) })' 'S117 tín hiệu span không xét kiểu span (mọi span là tiền)'
m $I '	case "asked_unit":
		return s.askedUnit(in.Raw[start:end], arg)' '	case "asked_unit":
		return false' 'S118 bỏ tín hiệu "bao nhiêu" + đơn vị tiền'
m tenants/construction/pack/intents.yaml 'dự tính, dự phóng, dự trù, ước tính, sẽ]' ']' 'S119 họ dự báo mất đồng nghĩa ngoài danh sách vòng 1'
m tenants/construction/pack/intents.yaml 'báo giá, hỏi giá, xin giá, chào giá, giá chào]' 'báo giá]' 'S120 họ báo giá mất đồng nghĩa ngoài danh sách vòng 1'
m tenants/construction/pack/intents.yaml 'cảnh báo, báo động, nhắc tôi, nhắc em, nhắc mình, nhắc nhở, báo khi, thông báo khi]' 'cảnh báo]' 'S121 họ cảnh báo mất đồng nghĩa ngoài danh sách vòng 1'
m tenants/construction/pack/intents.yaml 'nhập số liệu, nhập vào, khai báo, sửa số, chỉnh số, điều chỉnh số, hiệu chỉnh, thay số, lập phiếu]' ']' 'S122 họ lệnh ghi mất đồng nghĩa ngoài danh sách vòng 1'
m tenants/construction/pack/intents.yaml 'đánh giá, nhận định, bình luận, phân tích, chấm điểm, cho ý kiến, ý kiến về, kết luận, hợp lý, bất thường, bình thường, quá nhiều, tiết kiệm, cao hay thấp]' 'đánh giá]' 'S123 họ đánh giá mất đồng nghĩa ngoài danh sách vòng 1'
m tenants/construction/pack/intents.yaml 'giá thành, trị giá, đơn giá, giá dầu, giá nhiên liệu, quyết toán]' ']' 'S124 họ tiền mất cụm giá / chi phí ngoài danh sách vòng 1'

# ---- S4AS-03: so sánh theo cấu trúc, bỏ luật số cụm, phạm vi theo cụm xác nhận, hỏi lượng
m $I '		if d.Priority == top.Priority {' '		if d.Priority == top.Priority && false {' 'S128 hoà ý định cùng ưu tiên không còn hỏi lại (chọn ý định đứng đầu, như luật số cụm)'
m tenants/construction/pack/intents.yaml 'comparison_markers: [hơn, kém, so với]' 'comparison_markers: [so với]' 'S129 từ so sánh bậc hơn / kém ("cao hơn", "kém") ngoài pack'
m tenants/construction/pack/intents.yaml 'quantity_markers: [bao nhiêu, định mức, mấy]' 'quantity_markers: [bao nhiêu, định mức]' 'S130 "mấy lít" không đánh dấu mệnh đề hỏi'
m $I '			return 0, len(text), true' '			return clauseSpans(text)[0][0], clauseSpans(text)[0][1], true' 'S131 phép so sánh có "chênh lệch" chỉ xét mệnh đề đầu (danh sách chỉ số cách nhau bằng dấu phẩy bị cắt)'
m tenants/construction/pack/intents.yaml 'đo được, cảm biến báo, cảm biến, dùng' 'đo được, cảm biến, dùng' 'S132 "cảm biến báo" không là cụm của nguồn: "báo" thành chỉ số đội báo, hai chỉ số hoà nhau (WIN-I-038, H3 509)'

# ---- S4AS-04: cụm nghĩa ở mệnh đề ngữ cảnh vẫn được nói ra
m $I '				if slices.Contains(prem, c) && len(strings.Fields(ct[:at])) > 1 {' '				if false {' 'S133 "nguồn + động từ" ở mệnh đề ngữ cảnh ("phiếu kho ghi nhận 3850 L, vậy …") bị coi là lệnh ghi'
m $I '			ct := in.Raw[c[0]:c[1]]
			at := -1' '			ct := in.Raw[c[0]:c[1]]
			if c[1] <= qStart {
				continue
			}
			at := -1' 'S134 cụm nghĩa ở mệnh đề ngữ cảnh bị bỏ lặng (chỉ xét phần quyết định)'
m $I '				asked := s.asking(ct) || strings.TrimSpace(next) != "" && s.onlyMarkers(next, markers)' '				asked := s.asking(ct) || false && strings.TrimSpace(next) != "" && s.onlyMarkers(next, markers)' 'S135 cụm đuôi hỏi ("…, đúng không?") không áp cho mệnh đề liền trước ("ERP ghi nhận 3900 L, đúng không?" thành lệnh ghi)'

# ---- S4AS-05: lời chào thuần và mệnh đề bị bỏ qua
m $I '			if d.Alone && counts[i] > 0 {' '			if false {' 'S136 "chào" luôn là lời chào dù câu còn thực thể / con số ("chào cờ ở dự án A" được chào lại)'
m $I '			if slices.ContainsFunc(s.opt.DismissMarkers, func(m string) bool { return normalization.ContainsPhrase(ct, m) }) {' '			if false {' 'S137 cụm nghĩa trong mệnh đề "không quan tâm / để sau" vẫn bị hỏi lại'
m tenants/construction/pack/intents.yaml 'dismiss_markers: [không quan tâm, không cần, chưa cần, khỏi cần, để sau, bỏ qua]' 'dismiss_markers: []' 'S138 pack không khai mệnh đề bị bỏ qua'

# ---- S4AS-06: kiểm nạp pack của nghĩa chưa hỗ trợ và khử trùng ứng viên
m core/tenant/pack.go '				if u.Name == "" || u.Template == "" || len(u.Cues) == 0 {' '				if false {' 'S139 pack không đòi name / template / cues của nghĩa chưa hỗ trợ'
m core/tenant/pack.go '		needed = append(needed, "unsupported_meaning")' '		_ = "unsupported_meaning"' 'S140 pack không đòi mẫu chung unsupported_meaning'
m core/tenant/pack.go '			needed = append(needed, u.Template)' '			_ = u.Template' 'S141 pack không đòi mẫu riêng của từng nghĩa chưa hỗ trợ'
m $I '	case kind == "asked_unit" && arg != "" && dimensionKnown(arg):' '	case kind == "asked_unit":' 'S142 pack nhận chiều đo không có trong lexicon cho tín hiệu asked_unit'
m $I '			if !slices.ContainsFunc(out, func(o contracts.EntityRef) bool { return o.ID == c.ID }) {
				out = append(out, c)
				w.addOrigin' '			if true {
				out = append(out, c)
				w.addOrigin' 'S143 ứng viên của hai bí danh cùng thực thể bị liệt kê hai lần'

# ---- S4AS-08: mọi đường hỏi lại về ý định lọc nhãn theo capability
m core/pipeline/ask.go '	cls.Ambiguous = slices.DeleteFunc(slices.Clone(cls.Ambiguous), unseen)' '	_ = unseen' 'S144 hỏi lại vì hoà ý định kể cả ý định người hỏi không có quyền (lộ nhãn)'
m core/pipeline/ask.go '	cls.NeedTerm = slices.DeleteFunc(slices.Clone(cls.NeedTerm), unseen)' '	_ = unseen' 'S145 hỏi lại vì thiếu thuật ngữ kể cả ý định người hỏi không có quyền (lộ nhãn)'
m core/pipeline/ask.go '	cls.NeedMeaning = slices.DeleteFunc(cls.NeedMeaning, unseen)' '	_ = unseen' 'S146 hỏi lại nghĩa số liệu kể cả ý định người hỏi không có quyền (lộ nhãn)'

# ---- S4BS-01 (vòng 3): đoạn mục đích không góp từ khoá chỉ số / không kích hoạt cụm báo nghĩa; nhờ xác nhận không phải "nhận"
m $I '		if at >= 0 && !s.asking(ct[at:]) {' '		if at >= 0 {' 'S147 đoạn mục đích mang cụm hỏi ("để biết đội báo bao nhiêu dầu") vẫn bị che — chữ của câu hỏi bị mất'
m $I '	if len(s.opt.PurposeMarkers) == 0 {
		return nil
	}' '	if true {
		return nil
	}' 'S148 không nhận đoạn mục đích: "để em báo cáo" lại ngang hàng "được cấp"'
m $I '		if r := contracts.RouteKind(d.Route); r != contracts.RouteSQLQuery && r != contracts.RouteCalculation {' '		if false {' 'S149 đoạn mục đích che cả từ khoá quy trình ("dùng để làm gì theo quy trình" mất route RAG)'
m tenants/construction/pack/intents.yaml 'ignore_phrases: [người dùng, người sử dụng, xác nhận]' 'ignore_phrases: [người dùng, người sử dụng]' 'S150 "xác nhận" lại là chữ "nhận" của kho xuất (nhờ xác nhận làm hoà với đội báo)'
m tenants/construction/pack/intents.yaml 'purpose_markers: [để, nhằm]' 'purpose_markers: [để]' 'S151 từ nối "nhằm" không mở đoạn mục đích'
m $I '				ct := rawBG[c[0]:c[1]]' '				ct := in.Raw[c[0]:c[1]]' 'S152 cụm báo nghĩa trong đoạn mục đích ("để tính chi phí") vẫn kích hoạt'
