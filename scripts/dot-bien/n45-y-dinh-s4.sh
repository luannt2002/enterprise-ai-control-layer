#!/usr/bin/env bash
# Danh mục đột biến cho vòng sửa tầng ý định S4 (ADR-0010 bất biến (a), D-OP-15 = a2 kèm 2 điều kiện): câu có thực thể + thuật ngữ mà
# không từ khoá nào khớp không phải ngoài phạm vi — trả lời theo ý định chung kèm câu giả định khi câu không nói gì thêm, còn lại hỏi lại
# nghĩa số liệu, chỉ kể ý định người hỏi có quyền. Mỗi dòng: m <tệp> <chuỗi gốc> <chuỗi đột biến> <nhãn>. Kỳ vọng: mọi dòng "BỊ BẮT".
cd "$(dirname "$0")/../.." || exit 1
m() { scripts/mutate.sh "$1" "$2" "$3" "$4"; echo "rc=$?"; }
export PKGS="./intelligence/intent/ ./intelligence/router/ ./core/pipeline/ ./tenants/construction/"
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
# S10 — pack hiện có đúng một ý định yếu, khai chỉ số cho thuật ngữ của nó: tương đương trên pack cầu đường
# m $I '	if len(gen) != 1 || gen[0].MetricByTerm[term] == "" {' '	if len(gen) == 0 {' 'S10 a2 nhận ý định chung không có chỉ số cho thuật ngữ'
m $I '			if d.MetricByTerm[term] == gen[0].MetricByTerm[term] {' '			if true {' 'S11 mọi từ khoá số liệu đều là "của ý định chung" ("tiêu" của "tiêu thụ" thành giải thích được)'
m $I '		if d.Weak || len(d.Terms) == 0 || contracts.RouteKind(d.Route) == contracts.RouteRAG {
			continue
		}
		for _, k := range d.Keywords {' '		if len(d.Terms) == 0 || contracts.RouteKind(d.Route) == contracts.RouteRAG {
			continue
		}
		for _, k := range d.Keywords {' 'S12 từ khoá của chính ý định chung ("hết" của "hết bao nhiêu") thành giải thích được'
m $I '		if whole(p, other) && !whole(p, own) {' '		if false {' 'S13 từ trung tính là mảnh từ khoá của chỉ số khác ("đo") vẫn được tha'
m $I '		if !whole(raw[i:j], own) {
			return Def{}, false
		}' '		if false {
			return Def{}, false
		}' 'S14 a2 bỏ kiểm chữ còn lại (chữ lạ "xài", "hao" vẫn trả số kho xuất)'
m $I '	for _, e := range res.Entities {
		cover(e.Start, e.End)
	}' '	for _, e := range res.Entities[:0] {
		cover(e.Start, e.End)
	}' 'S15 a2 không che tên thực thể'
m $I '	for _, m := range res.Ambiguous {
		cover(m.Start, m.End)
	}' '	for _, m := range res.Ambiguous[:0] {
		cover(m.Start, m.End)
	}' 'S16 a2 không che bí danh mơ hồ'
m $I '			if ws, we, ok := normalization.PeriodWordBefore(raw, sp.Start); ok {
				cover(ws, we)
			}
		case contracts.ValueTerm, contracts.ValueQuantity, contracts.ValueNumber:' '			if ws, we, ok := normalization.PeriodWordBefore(raw, sp.Start); ok && false {
				cover(ws, we)
			}
		case contracts.ValueTerm, contracts.ValueQuantity, contracts.ValueNumber:' 'S17 a2 không che "ngày" đứng trước mốc số'
m $I '		case contracts.ValueTerm, contracts.ValueQuantity, contracts.ValueNumber:
			cover(sp.Start, sp.End)' '		case contracts.ValueTerm, contracts.ValueQuantity, contracts.ValueNumber:
			_ = sp' 'S18 a2 không che thuật ngữ / lượng'
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
