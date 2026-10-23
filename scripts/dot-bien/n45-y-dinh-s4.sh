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
