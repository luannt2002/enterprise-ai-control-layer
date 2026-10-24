#!/usr/bin/env bash
# Danh mục đột biến "W5 — RAG / HYBRID, ghép route, truy xuất, pack" đã chạy ngày 20/10/2026 (xem docs/ops/NHAT-KY-2026-10-20.md).
# Mỗi dòng: m <tệp> <chuỗi gốc> <chuỗi đột biến> <nhãn>. Kỳ vọng: mọi dòng in "BỊ BẮT" (rc=0).
# Mã đổi sau ngày đó thì chuỗi gốc có thể không còn khớp (mutate.sh báo "xuất hiện 0 lần", rc=2):
# sửa chuỗi theo mã mới rồi chạy lại — đừng xoá dòng cho xanh.
cd "$(dirname "$0")/../.." || exit 1
export PKGS="./data/... ./core/... ./intelligence/... ./tenants/... ./evals/ ./reliability/..."
m() { scripts/mutate.sh "$1" "$2" "$3" "$4"; echo "rc=$?"; }
m intelligence/router/router.go 'ok := pol.Allows(pr, kdef.Capability)' 'ok := true || pol.Allows(pr, kdef.Capability)' 'AD router bỏ kiểm documents.read của HYBRID'
m intelligence/intent/intent.go '		r.Knowledge = k.Intent.Name' '		_ = k' 'AE Classify không gắn Knowledge'
m intelligence/intent/intent.go 'func missingEntity(d Def, it contracts.Intent) bool {' 'func missingEntity(d Def, it contracts.Intent) bool {
	return false' 'AF thiếu thực thể không rơi về RAG'
m intelligence/intent/intent.go '		if _, dated := firstPeriod(in); dated || len(in.InvalidDates) > 0 || (len(r.Unclear.Entities) > 0 && s.asksData(in.Raw)) {' '		if _, dated := firstPeriod(in); dated && false || len(in.InvalidDates) > 0 || (len(r.Unclear.Entities) > 0 && s.asksData(in.Raw)) {' 'AG bỏ luật mốc thời gian'
m core/pipeline/ask.go '	if a.quantityMarker(in.Raw) == "" {' '	if true || a.quantityMarker(in.Raw) == "" {' 'AH bỏ chặn câu hỏi lượng thuần tài liệu'
m core/pipeline/ask.go '		v = append(v, validation.ForbidDimensions(a.RT.Normalizer, draft, a.factDimensions(), now)...)
		return append(v, validation.ForbidUnitWords(draft.Text, a.factUnitWords())...), nil' '		return v, nil' 'AI bỏ cả hai lớp chặn lượng dầu lấy từ tài liệu'
m core/pipeline/ask.go '		v := validation.Bound(draft, ev)' '		var v []contracts.Violation' 'AJ bỏ ràng buộc claim trong luồng'
m intelligence/rag/rag.go '	if string(pr.TenantID) != ix.tenant {' '	if false && string(pr.TenantID) != ix.tenant {' 'AM bỏ kiểm tenant của truy xuất'
m intelligence/rag/rag.go '		if d.Tenant != tenant {' '		if false && d.Tenant != tenant {' 'AN chỉ mục nhận tài liệu tenant khác'
m intelligence/rag/rag.go '		case f == w: // gõ không dấu' '		case true: // gõ không dấu' 'AO câu hỏi luôn so bỏ dấu'
m intelligence/rag/rag.go '	for _, p := range questionPhrases {' '	for _, p := range questionPhrases[:0] {' 'AP không bỏ cụm nghi vấn'
m data/normalization/text.go '	return norm.NFC.String(base.String()) + string(tone)' '	return norm.NFC.String(base.String())' 'AQ ToneKey bỏ dấu thanh'
m intelligence/rag/rag.go 'terms: indexTerms(d.Title + " " + scored)' 'terms: indexTerms(scored)' 'AR đoạn không mang từ tiêu đề'
m core/pipeline/ask.go '		if plan.Route == contracts.RouteHybrid && resp.Outcome == contracts.OutcomeResolved {' '		if plan.Route == contracts.RouteHybrid {' 'AS HYBRID nối tài liệu cả khi số liệu không ra'
m core/tenant/pack.go '	if rag {
		needed = append(needed, knowledgeTemplates...)
	}' '	_ = rag' 'AU pack không đòi mẫu câu tri thức'
m intelligence/router/router.go 'contracts.RouteHybrid, cls.Knowledge != "" && (plan.Route' 'contracts.RouteHybrid, false && cls.Knowledge != "" && (plan.Route' 'AV router không nâng HYBRID'
m evals/runner.go '			case counter.ModelCalls() != *want:' '			case false:' 'AW runner bỏ so model_calls'
m evals/runner.go '			case !counts:' '			case false:' 'AX runner tha Asker không đếm được'
m intelligence/models/models.go 'Claims: []contracts.Claim{{Text: e.Text, EvidenceIDs: []string{e.ID}}}' 'Claims: []contracts.Claim{{Text: e.Text}}' 'AY model giả không gắn bằng chứng'
m core/pipeline/ask.go '		if _, derived := a.RT.Metrics.Derived(metric); derived {' '		if plan.Route == contracts.RouteCalculation {' 'AZ HYBRID phép tính đi đường Query'
