#!/usr/bin/env bash
# Danh mục đột biến cho cổng phán quyết của câu xác nhận (ADR-0009): core/pipeline/claim.go và chỗ nối ở answerMetric. Mỗi dòng: m <tệp> <chuỗi gốc> <chuỗi đột biến> <nhãn>. Kỳ vọng: mọi dòng "BỊ BẮT".
cd "$(dirname "$0")/../.." || exit 1
m() { scripts/mutate.sh "$1" "$2" "$3" "$4"; echo "rc=$?"; }
export PKGS="./core/pipeline/ ./data/normalization/ ./data/sourceoftruth/ ./intelligence/intent/ ./tenants/construction/"
C=core/pipeline/claim.go
m $C '	if !ok || idef.Weak {' '	if !ok {' 'CG1 chỉ số theo quy ước của pack vẫn phán quyết'
m $C '	if len(readPeriods(in)) != 1 {' '	if len(readPeriods(in)) > 1 {' 'CG2 không nêu mốc vẫn phán quyết'
m $C '			if inName(sp.Start, sp.End) {
				continue // số trong tên thực thể ("03" của "máy 03")
			}' '			if false && inName(sp.Start, sp.End) {
				continue // số trong tên thực thể ("03" của "máy 03")
			}' 'CG3 số trong tên thực thể thành con số'
m $C 'ok && sp.Kind == contracts.ValueNumber && qualifierHolds(q, readPeriods(in)) {' 'false && ok && sp.Kind == contracts.ValueNumber && qualifierHolds(q, readPeriods(in)) {' 'CG4 mốc khớp kỳ đã đọc không được giải thích'
m $C '				if unitShown(sp) {
					cover(sp.Start, sp.End) // lượng của ngữ cảnh' '				if false {
					cover(sp.Start, sp.End) // lượng của ngữ cảnh' 'CG5 lượng ở ngữ cảnh không được giải thích'
m $C '				if unitShown(sp) {
					cover(sp.Start, sp.End) // lượng của ngữ cảnh' '				if true {
					cover(sp.Start, sp.End) // lượng của ngữ cảnh' 'CG6 số trần ở ngữ cảnh được giải thích'
m $C '			if !ok || !unitShown(sp) || def.InRange(v.Number) != nil {' '			if !ok || def.InRange(v.Number) != nil {' 'CG8 số trần thành con số xác nhận'
m $C '			if c, err := decimal.Cmp(v.Number, claimed.Number); err != nil || c != 0 {' '			if c, err := decimal.Cmp(v.Number, claimed.Number); false && (err != nil || c != 0) {' 'CG9 con số khác con số xác nhận'
m $C 'coverPhrases(slices.Concat(idef.Keywords, a.RT.ConfirmMarkers, a.RT.ClaimNeutral), 0, len(raw))' 'coverPhrases(slices.Concat(a.RT.ConfirmMarkers, a.RT.ClaimNeutral), 0, len(raw))' 'CG10 bỏ từ khoá của ý định'
m $C '	for _, sp := range srcSpans {
		cover(sp[0], sp[1])
	}' '	for _, sp := range srcSpans {
		_ = sp
	}' 'CG11 tên nguồn khớp nguồn đã dùng vẫn không được giải thích'
m $C 'coverPhrases(slices.Concat(idef.Keywords, a.RT.ConfirmMarkers, a.RT.ClaimNeutral), 0, len(raw))' 'coverPhrases(slices.Concat(idef.Keywords, a.RT.ConfirmMarkers), 0, len(raw))' 'CG12 bỏ từ trung tính'
m $C '				if len(used) != 1 || used[0] != s.Type {' '				if false {' 'CG13 tên nguồn được giải thích dù số không lấy từ nguồn đó'
m $C '	if lo > 0 {' '	if false {' 'CG14 ngữ cảnh không được giải thích'
m $C '		if !covered[i] && !unicode.IsSpace(r) && !strings.ContainsRune(claimSeparators, r) {' '		if false && !covered[i] && !unicode.IsSpace(r) && !strings.ContainsRune(claimSeparators, r) {' 'CG15 bỏ kiểm phần dư'
m $C '		if !covered[i] && !unicode.IsSpace(r) && !strings.ContainsRune(claimSeparators, r) {' '		if !covered[i] && (unicode.IsLetter(r) || unicode.IsDigit(r)) {' 'CG7 ký hiệu (~, >, /) không là phần dư'
m $C 'const claimSeparators = ",.;:?!()[]\"'"'"'“”‘’…-"' 'const claimSeparators = ",.;:?!()[]\"'"'"'“”‘’…->~/"' 'CG18 dấu so sánh / xấp xỉ / mẫu số thành dấu ngăn cách'
m $C 'const claimSeparators = ",.;:?!()[]\"'"'"'“”‘’…-"' 'const claimSeparators = ",.;:?!()[]\"'"'"'“”‘’…-–—"' 'CG19 gạch ngắn / dài thành dấu ngăn cách (mất dấu âm)'
m $C '			if ws, we, ok := normalization.PeriodWordBefore(raw, sp.Start); ok {' '			if ws, we, ok := normalization.PeriodWordBefore(raw, sp.Start); false && ok {' 'CG16 từ chỉ kỳ trước mốc thành phần dư'
m $C '			lo = sentenceStart(raw, fs) // mệnh đề hỏi không có số' '			lo = fs // mệnh đề hỏi không có số' 'CG17 mệnh đề hỏi không số: không lùi về đầu câu'
A=core/pipeline/ask.go
m $A '		if withheld = a.verdictGap(in, names, it, def, claimed, subj, result.UsedSources); withheld != "" {' '		if withheld = a.verdictGap(in, names, it, def, claimed, subj, result.UsedSources)[:0]; withheld != "" {' 'AI1 tắt cổng phán quyết'
m $A '		text += a.fragment("confirm_withheld", vars)' '		_ = vars' 'AI2 bỏ mẩu chưa kết luận'
m $C '	if !normalization.HasDiacritic(raw) {' '	if false && !normalization.HasDiacritic(raw) {' 'CG20 câu không dấu vẫn phán quyết'
m $C '	if markerCount(raw, a.RT.ConfirmMarkers) != 1 {' '	if markerCount(raw, a.RT.ConfirmMarkers) > 9 {' 'CG21 nhiều cụm xác nhận vẫn phán quyết'
m $C '		if gap := subj.gap(names, from); gap != "" {' '		if gap := subj.gap(names, from); false && gap != "" {' 'CG22 tắt kiểm chủ thể'
m $C 'return id == s.id || s.ancestors[id] })' 'return true })' 'CG23 cụm tên chỉ thực thể khác (không phải cấp cha) vẫn phán quyết'
m $C '	if inFocus && !subjectInFocus {' '	if false && inFocus && !subjectInFocus {' 'CG24 mệnh đề hỏi nêu thực thể khác chủ thể vẫn phán quyết'
m $C '		s.ancestors[p] = true' '		s.ancestors[p] = false' 'CG25 không tính cấp cha của chủ thể'
m $C '			if !ok || !unitShown(sp) || def.InRange(v.Number) != nil {' '			if !ok || !unitShown(sp) {' 'CG26 con số ngoài miền vẫn phán quyết'
m $C '	return strings.ContainsFunc(sp.Raw, unicode.IsLetter)' '	return strings.ContainsFunc(sp.Raw, unicode.IsPrint)' 'CG27 số trần có dấu chấm / phẩy thành có đơn vị'
m $C '		if sp[0] >= end {' '		if true {' 'CG28 cụm xác nhận chồng nhau bị đếm hai lần'
N=data/normalization/qualifier.go
m $N '		if pw.kind == QualifierDay && wordIs(raw, ws, we, pw.word) {' '		if wordIs(raw, ws, we, pw.word) {' 'QW1 "tuần" / "tháng" trước một ngày thành từ chỉ ngày'
I=intelligence/intent/intent.go
m $I '		if contracts.RouteKind(d.Route) == contracts.RouteSQLQuery && !d.Weak {' '		if true {' 'IK1 từ khoá của ý định không phải số liệu thành ngữ cảnh hợp lệ'
Q=data/sourceoftruth/query.go
m $Q '		usedSources[best.source] = true' '		usedSources["warehouse_slip"] = true' 'SQ1 nguồn đã dùng ghi sai'
m $A '		out = append(out, mention{start: e.Start, end: e.End, ids: []string{e.ID}})' '		out = append(out, mention{start: e.Start, end: e.End})' 'AI3 cụm tên không mang thực thể'
m $A '		out = append(out, mention{start: m.Start, end: m.End, ids: ids})' '		out = append(out, mention{start: m.Start, end: m.End, ids: ids[:0]})' 'AI4 bí danh mơ hồ không mang ứng viên'
m $C '			if value && (sp.Start < lastEnd || strings.Trim(raw[lastEnd:sp.Start], " ()") != "") {' '			if false && value && (sp.Start < lastEnd || strings.Trim(raw[lastEnd:sp.Start], " ()") != "") {' 'CG29 con số nhắc lại sau phát biểu khác vẫn là cùng claim'
m $C '	for _, from := range []int{lo, qs} {' '	for _, from := range []int{lo, lo+0*qs} {' 'CG30 kiểm chủ thể chỉ theo phạm vi đã lùi về đầu câu (bỏ mệnh đề hỏi)'
m $C '		return !(strings.Contains(sp.Rule, "unit_from_term") && strings.ContainsAny(sp.Raw, ".,") && !strings.ContainsFunc(sp.Raw, unicode.IsLetter))' '		return true' 'CG31 lượng có dấu chấm / phẩy, đơn vị suy từ thuật ngữ, thành có đơn vị'
m $C '	for p := parent[id]; p != "" && !s.ancestors[p]; p = parent[p] {' '	for p := parent[id]; p != "" && !s.ancestors[p]; p = "" {' 'CG32 chỉ tính MỘT cấp cha'
m $C '		coverPhrases(slices.Concat(a.RT.Intents.MetricKeywords(), labels, a.RT.QuantityMarkers), 0, lo)' '		coverPhrases(slices.Concat(a.RT.Intents.MetricKeywords(), labels), 0, lo)' 'CG33 từ chỉ lượng ở ngữ cảnh không được giải thích'
m $C '	if len(readPeriods(in)) != 1 {' '	if len(readPeriods(in)) < 1 {' 'CG34 hai kỳ đọc được vẫn phán quyết'
m $C '	for _, from := range []int{lo, qs} {' '	for _, from := range []int{qs, qs} {' 'CG35 kiểm chủ thể chỉ ở mệnh đề hỏi (bỏ phạm vi câu — hồi quy T2.2, soát lần ba)'
m $C 'strings.ContainsAny(sp.Raw, ".,") && !strings.ContainsFunc(sp.Raw, unicode.IsLetter))' 'strings.ContainsAny(sp.Raw, ".,"))' 'CG36 lượng có chữ ("3,85k") vẫn bị coi là mơ hồ'
m $C 'strings.Trim(raw[lastEnd:sp.Start], " ()") != "") {' 'strings.Trim(raw[lastEnd:sp.Start], " (") != "") {' 'CG37 ngoặc đóng giữa hai lần nhắc thành phát biểu khác'
m $C 'strings.Trim(raw[lastEnd:sp.Start], " ()") != "") {' 'strings.Trim(raw[lastEnd:sp.Start], " ()\n") != "") {' 'CG38 xuống dòng giữa hai lần nhắc thành liền kề'
m $C '			value, lastEnd = true, sp.End' '			if !value {
				lastEnd = sp.End
			}
			value = true' 'CG39 khoảng liền kề đo từ lần nhắc ĐẦU thay vì lần gần nhất'
m $C 'strings.ContainsAny(sp.Raw, ".,") && !strings.ContainsFunc' 'strings.ContainsAny(sp.Raw, ",") && !strings.ContainsFunc' 'CG40 lượng suy đơn vị có dấu chấm thành có đơn vị'
