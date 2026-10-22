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
m $C '	return normalization.HasThousandsWord(sp.Raw) || (strings.Contains(sp.Rule, "+multiplier") && !normalization.HasNumberWord(sp.Raw))' '	return normalization.HasThousandsWord(sp.Raw) || (strings.Contains(sp.Rule, "+multiplier") && !normalization.HasNumberWord(sp.Raw)) || strings.ContainsAny(sp.Raw, ".,")' 'CG27 số trần có dấu chấm / phẩy thành có đơn vị'
m $C '		if sp[0] >= end {' '		if true {' 'CG28 cụm xác nhận chồng nhau bị đếm hai lần'
N=data/normalization/qualifier.go
m $N '		if pw.kind == QualifierDay && wordIs(raw, ws, we, pw.word) {' '		if wordIs(raw, ws, we, pw.word) {' 'QW1 "tuần" / "tháng" trước một ngày thành từ chỉ ngày'
I=intelligence/intent/intent.go
Q=data/sourceoftruth/query.go
m $Q '		usedSources[best.source] = true' '		usedSources["warehouse_slip"] = true' 'SQ1 nguồn đã dùng ghi sai'
m $A '		out = append(out, mention{start: e.Start, end: e.End, ids: []string{e.ID}})' '		out = append(out, mention{start: e.Start, end: e.End})' 'AI3 cụm tên không mang thực thể'
m $A '		out = append(out, mention{start: m.Start, end: m.End, ids: ids})' '		out = append(out, mention{start: m.Start, end: m.End, ids: ids[:0]})' 'AI4 bí danh mơ hồ không mang ứng viên'
m $C '			if value && (sp.Start < lastEnd || strings.Trim(raw[lastEnd:sp.Start], " ()") != "") {' '			if false && value && (sp.Start < lastEnd || strings.Trim(raw[lastEnd:sp.Start], " ()") != "") {' 'CG29 con số nhắc lại sau phát biểu khác vẫn là cùng claim'
m $C '		return !(strings.Contains(sp.Rule, "unit_from_term") && strings.ContainsAny(sp.Raw, ".,") && strings.TrimRightFunc(sp.Raw, unicode.IsLetter) == sp.Raw)' '		return true' 'CG31 lượng có dấu chấm / phẩy, đơn vị suy từ thuật ngữ, thành có đơn vị'
m $C '	for p := parent[id]; p != "" && !s.ancestors[p]; p = parent[p] {' '	for p := parent[id]; p != "" && !s.ancestors[p]; p = "" {' 'CG32 chỉ tính MỘT cấp cha'
m $C '		coverPhrases(slices.Concat(a.RT.Intents.MetricKeywords(), labels, a.RT.QuantityMarkers), 0, lo)' '		coverPhrases(slices.Concat(a.RT.Intents.MetricKeywords(), labels), 0, lo)' 'CG33 từ chỉ lượng ở ngữ cảnh không được giải thích'
m $C '	if len(readPeriods(in)) != 1 {' '	if len(readPeriods(in)) < 1 {' 'CG34 hai kỳ đọc được vẫn phán quyết'
m $C 'strings.ContainsAny(sp.Raw, ".,") && strings.TrimRightFunc(sp.Raw, unicode.IsLetter) == sp.Raw)' 'strings.ContainsAny(sp.Raw, ".,"))' 'CG36 lượng có chữ ("3,85k") vẫn bị coi là mơ hồ'
m $C 'strings.Trim(raw[lastEnd:sp.Start], " ()") != "") {' 'strings.Trim(raw[lastEnd:sp.Start], " (") != "") {' 'CG37 ngoặc đóng giữa hai lần nhắc thành phát biểu khác'
m $C 'strings.Trim(raw[lastEnd:sp.Start], " ()") != "") {' 'strings.Trim(raw[lastEnd:sp.Start], " ()\n") != "") {' 'CG38 xuống dòng giữa hai lần nhắc thành liền kề'
m $C '			value, lastEnd = true, sp.End' '			if !value {
				lastEnd = sp.End
			}
			value = true' 'CG39 khoảng liền kề đo từ lần nhắc ĐẦU thay vì lần gần nhất'
m $C 'strings.ContainsAny(sp.Raw, ".,") && strings.TrimRightFunc' 'strings.ContainsAny(sp.Raw, ",") && strings.TrimRightFunc' 'CG40 lượng suy đơn vị có dấu chấm thành có đơn vị'
m $C '	if slices.ContainsFunc(names, func(m mention) bool { return m.start >= firstEnd && !slices.Contains(m.ids, subj.id) }) {' '	if false && slices.ContainsFunc(names, func(m mention) bool { return m.start >= firstEnd && !slices.Contains(m.ids, subj.id) }) {' 'CG41 tắt E3: tên khác đứng sau con số xác nhận vẫn phán quyết'
m $C 'return m.start >= firstEnd && !slices.Contains(m.ids, subj.id) })' 'return m.start >= firstEnd && !slices.ContainsFunc(m.ids, func(id string) bool { return id == subj.id || subj.ancestors[id] }) })' 'CG42 E3 cho qua cấp cha đứng sau con số (dạng gương)'
m $C 'strings.TrimRightFunc(sp.Raw, unicode.IsLetter) == sp.Raw' '!strings.ContainsFunc(sp.Raw, unicode.IsLetter)' 'CG43 chữ ở bất kỳ đâu (không chỉ cuối) xoá mơ hồ — "1,2k5" phán quyết (hồi quy T2.3)'
m $C 'strings.TrimRightFunc(sp.Raw, unicode.IsLetter) == sp.Raw' 'strings.TrimRightFunc(sp.Raw, unicode.IsLower) == sp.Raw' 'CG44 hậu tố viết hoa ("3,85K") không xoá mơ hồ'
m $C '	if slices.ContainsFunc(names, func(m mention) bool { return m.start >= firstEnd && !slices.Contains(m.ids, subj.id) }) {
		return "subject"' '	if slices.ContainsFunc(names, func(m mention) bool { return m.start >= firstEnd && !slices.Contains(m.ids, subj.id) }) {
		return "value"' 'CG45 E3 trả nhãn sai (value thay vì subject)'
m $C 'return m.start >= firstEnd && !slices.Contains(m.ids, subj.id) })' 'return m.start >= firstEnd && !(len(m.ids) == 1 && m.ids[0] == subj.id) })' 'CG46 E3 coi bí danh mơ hồ có chủ thể là tên khác'
# ---- Vòng 5 (ADR-0009, 03c2c54): owners / contextOwners / plainBetween / hasBreak / bareWrapped / wrapped / aliasTied / ownedBy / numberOwned /
# WeakKeywords / AmbiguousNumberWords (thêm 22/10/2026, v5db). Đo ở 03c2c54: 72 bị bắt ngay; 34 sống vì thiếu test (nhóm kế tiếp); 6 chờ quyết
# chính sách; 8 tương đương. Chạy: taskset -c … bash scripts/dot-bien/n43-cong-phan-quyet.sh ----
# Cập nhật 22/10/2026 (v5db2) cho 95dea0f: contextOwners nhận danh sách cụm của qualifierPhrases, có thêm topicQualifier và nhánh "rưỡi" của AmbiguousNumberWords; 18 dòng cũ viết lại cho mã mới,
# thêm CG153–CG202 và NW12–NW20, bốn dạng chính sách mặc định (CG125 / CG127 / CG129 / CG138) đã ghim nên chạy hẳn; các dòng sống còn lại ở ba nhóm cuối tệp.
W=data/normalization/numwords_vi.go
m $C '!subj.numberOwned(raw, os, firstStart, firstEnd, lo)' 'false && !subj.numberOwned(raw, os, firstStart, firstEnd, lo)' 'CG49 V5 tắt phép kiểm số-gắn-chủ-thể (numberOwned)'
m $C '				return "value" // "một nghìn hai"' '				return "subject" // "một nghìn hai"' 'CG55 V5 K5 trả nhãn sai (subject thay vì value)'
m $C '	keep := func(p string) bool { return keepSrc(p) && !slices.Contains(idef.Keywords, p) }' '	keep := func(p string) bool { return keepSrc(p) }' 'CG56 V5 R-1 từ khoá của chính ý định thành chủ của con số'
m $C '	keepSrc := func(p string) bool { return p != "" && !slices.Contains(weak, p) }' '	keepSrc := func(p string) bool { return p != "" && !slices.Contains(weak[:0], p) }' 'CG57 V5 R-1 từ khoá ý định yếu ("bao nhiêu") thành chủ của con số'
m $C '	keep := func(p string) bool { return keepSrc(p) && !slices.Contains(idef.Keywords, p) }' '	keep := func(p string) bool { return p != "" }' 'CG58 V5 qualifierPhrases không loại trừ từ khoá nào (skip vô hiệu)'
m $C '			if len(used) == 1 && used[0] == src.Type {' '			if len(used) == 1 && used[0] != src.Type {' 'CG61 V5 qualifierPhrases đảo điều kiện: chỉ tên nguồn đã dùng là nguồn khác'
m $C 'for _, p := range append([]string{src.Label}, src.Aliases...) {' 'for _, p := range src.Aliases {' 'CG63 V5 qualifierPhrases: nhãn của nguồn không là nguồn khác'
m $C 'for _, p := range slices.Concat(a.RT.Intents.MetricKeywords(), a.RT.QuantityMarkers) {' 'for _, p := range slices.Concat(a.RT.Intents.MetricKeywords()) {' 'CG65 V5 qualifierPhrases: từ chỉ lượng không là chỉ số khác'
m $C '			if !numberIn(sp[1], end) {' '			if !numberIn(sp[1], end) || true {' 'CG66 khoá: cụm đã có số của riêng nó vẫn chặn (bỏ numberIn)'
m $C '			if sp[1] > lo {
				continue
			}
			end := lo' '			if false {
				continue
			}
			end := lo' 'CG67 khoá: cụm ở phạm vi hỏi cũng tính là ngữ cảnh'
m $C 'return (sp.Kind == contracts.ValueQuantity || sp.Kind == contracts.ValueNumber) && sp.Start >= s && sp.Start < e' 'return sp.Kind == contracts.ValueQuantity && sp.Start >= s && sp.Start < e' 'CG69 V5 numberIn: số trần không tính là "số xen giữa"'
m $C 'return (sp.Kind == contracts.ValueQuantity || sp.Kind == contracts.ValueNumber) && sp.Start >= s && sp.Start < e' 'return sp.Kind == contracts.ValueNumber && sp.Start >= s && sp.Start < e' 'CG70 V5 numberIn: lượng có đơn vị không tính là "số xen giữa"'
m $C '&& sp.Start >= s && sp.Start < e && !inName(sp.Start, sp.End)' '&& sp.Start >= s && !inName(sp.Start, sp.End)' 'CG71 V5 numberIn: bỏ cận trên (số ở phạm vi hỏi cũng xen giữa)'
m $C 'out = append(out, ownerRef{m.start, m.end, m.ids})' 'out = append(out, ownerRef{m.start, m.end, nil})' 'CG74 V5 owners: tên thực thể không mang ứng viên'
m $C 'for _, m := range names {
		out = append(out, ownerRef{m.start, m.end, m.ids})' 'for _, m := range names[:0] {
		out = append(out, ownerRef{m.start, m.end, m.ids})' 'CG75 V5 owners: tên thực thể không là chủ'
m $C 'if sp.Kind != contracts.ValueTerm || slices.Contains(ownTerms, sp.Value.Text) || inName(sp.Start, sp.End) {' 'if true || inName(sp.Start, sp.End) {' 'CG76 V5 owners: thuật ngữ ("dự án", "công trình") không là chủ'
m $C 'if sp.Kind != contracts.ValueTerm || slices.Contains(ownTerms, sp.Value.Text) || inName(sp.Start, sp.End) {' 'if sp.Kind != contracts.ValueTerm || inName(sp.Start, sp.End) {' 'CG77 V5 owners: thuật ngữ của chính ý định ("dầu") cũng là chủ'
m $C 'slices.Contains(ownTerms, sp.Value.Text) || inName(sp.Start, sp.End) {' 'slices.Contains(ownTerms, sp.Value.Text) || (inName(sp.Start, sp.End) && false) {' 'CG78 V5 owners: thuật ngữ nằm trong cụm tên vẫn tính riêng'
m $C 'if sp.Kind != contracts.ValueTerm || slices.Contains(ownTerms' 'if sp.Kind == contracts.ValueTerm || slices.Contains(ownTerms' 'CG79 V5 owners: đảo loại span (thuật ngữ bị bỏ, mọi span khác thành chủ)'
m $C 'slices.SortFunc(out, func(a, b ownerRef) int { return a.start - b.start })' 'slices.SortFunc(out, func(a, b ownerRef) int { return 0 })' 'CG82 V5 owners: không sắp theo vị trí'
m $C 'case unicode.IsLetter(r) || unicode.IsDigit(r) || r == '"'"' '"'"':' 'case unicode.IsDigit(r) || r == '"'"' '"'"':' 'CG83 V5 plainBetween: chữ cái không là chữ thường'
m $C 'case unicode.IsLetter(r) || unicode.IsDigit(r) || r == '"'"' '"'"':' 'case unicode.IsLetter(r) || r == '"'"' '"'"':' 'CG84 V5 plainBetween: chữ số không là chữ thường'
m $C 'case unicode.IsLetter(r) || unicode.IsDigit(r) || r == '"'"' '"'"':' 'case unicode.IsLetter(r) || unicode.IsDigit(r):' 'CG85 V5 plainBetween: khoảng trắng không là chữ thường'
m $C 'strings.ContainsRune(".,/-:", r)' 'strings.ContainsRune(",/-:", r)' 'CG86 V5 plainBetween: bỏ dấu chấm giữa hai chữ số'
m $C 'strings.ContainsRune(".,/-:", r)' 'strings.ContainsRune(".,-:", r)' 'CG88 V5 plainBetween: bỏ gạch chéo giữa hai chữ số (ngày 19/10)'
m $C 'strings.ContainsRune(".,/-:", r)' 'strings.ContainsRune(".,/:", r)' 'CG89 V5 plainBetween: bỏ gạch ngang giữa hai chữ số (ngày 19-10)'
m $C 'strings.ContainsRune(".,/-:", r)' 'strings.ContainsRune(".,/-", r)' 'CG90 V5 plainBetween: bỏ hai chấm giữa hai chữ số (giờ 8:30)'
m $C '&& i > 0 && i+1 < len(rs) && unicode.IsDigit(rs[i-1])' '&& i >= 0 && i+1 < len(rs) && unicode.IsDigit(rs[i-1])' 'CG91 V5 plainBetween: dấu đầu chuỗi (i >= 0) đọc rs[-1]'
m $C 'unicode.IsDigit(rs[i-1]) && unicode.IsDigit(rs[i+1])' 'unicode.IsDigit(rs[i-1]) || unicode.IsDigit(rs[i+1])' 'CG93 V5 plainBetween: một chữ số bên cạnh là đủ (|| thay &&)'
m $C 'unicode.IsDigit(rs[i-1]) && unicode.IsDigit(rs[i+1])' 'unicode.IsDigit(rs[i-1])' 'CG94 V5 plainBetween: chỉ xét chữ số bên trái'
m $C '		case '"'"'\n'"'"':
			return true' '		case '"'"'\n'"'"':
			return false' 'CG96 V5 hasBreak: xuống dòng không là ngắt mệnh đề'
m $C 'case '"'"','"'"', '"'"';'"'"', '"'"'.'"'"', '"'"'?'"'"', '"'"'!'"'"':' 'case '"'"';'"'"', '"'"'.'"'"', '"'"'?'"'"', '"'"'!'"'"':' 'CG97 V5 hasBreak: dấu phẩy không là ngắt mệnh đề'
m $C 'case '"'"','"'"', '"'"';'"'"', '"'"'.'"'"', '"'"'?'"'"', '"'"'!'"'"':' 'case '"'"','"'"', '"'"'.'"'"', '"'"'?'"'"', '"'"'!'"'"':' 'CG98 V5 hasBreak: dấu chấm phẩy không là ngắt mệnh đề'
m $C 'case '"'"','"'"', '"'"';'"'"', '"'"'.'"'"', '"'"'?'"'"', '"'"'!'"'"':' 'case '"'"','"'"', '"'"';'"'"', '"'"'?'"'"', '"'"'!'"'"':' 'CG99 V5 hasBreak: dấu chấm không là ngắt mệnh đề'
m $C 'switch strings.TrimSpace(between) {' 'switch between {' 'CG107 V5 bareWrapped: không bỏ khoảng trắng quanh ngoặc mở'
m $C 'case "(":
		return strings.HasPrefix(rest, ")")' 'case "(":
		return true' 'CG108 V5 bareWrapped: ( không cần ) đóng'
m $C 'case "[":
		return strings.HasPrefix(rest, "]")' 'case "[":
		return true' 'CG109 V5 bareWrapped: [ không cần ] đóng'
m $C 'case "[":
		return strings.HasPrefix(rest, "]")' 'case "[":
		return false' 'CG110 V5 bareWrapped: không nhận ngoặc vuông'
m $C 'case "(":
		return strings.HasPrefix(rest, ")")' 'case "(":
		return strings.HasPrefix(rest, "]")' 'CG111 V5 bareWrapped: ( đóng bằng ]'
m $C '(strings.HasSuffix(before, "[") && strings.HasPrefix(after, "]"))' 'false' 'CG112 V5 wrapped: không nhận ngoặc vuông'
m $C 'before, after := strings.TrimRight(raw[:start], " "), strings.TrimLeft(raw[end:], " ")' 'before, after := raw[:start], strings.TrimLeft(raw[end:], " ")' 'CG113 V5 wrapped: không bỏ khoảng trắng trước cụm'
m $C 'before, after := strings.TrimRight(raw[:start], " "), strings.TrimLeft(raw[end:], " ")' 'before, after := strings.TrimRight(raw[:start], " "), raw[end:]' 'CG114 V5 wrapped: không bỏ khoảng trắng sau cụm'
m $C '(strings.HasSuffix(before, "(") && strings.HasPrefix(after, ")")) ||' '(strings.HasSuffix(before, "(") || strings.HasPrefix(after, ")")) ||' 'CG115 V5 wrapped: chỉ cần một đầu ngoặc tròn'
m $C 'if len(c.ids) != 1 || c.ids[0] != s.id {' 'if len(c.ids) != 1 {' 'CG116 V5 aliasTied: mọi tên đơn (kể cả cấp cha) là lần nhắc rõ của chủ thể'
m $C '	if best < 0 {
		return true
	}' '	if best < 0 {
		return false
	}' 'CG119 V5 aliasTied: không có lần nhắc rõ thì bí danh chưa gỡ'
m $C 'if c.end <= a.start { // lần nhắc rõ đứng trước bí danh' 'if false && c.end <= a.start { // lần nhắc rõ đứng trước bí danh' 'CG120 V5 aliasTied: tắt nhánh lần nhắc rõ đứng trước bí danh'
m $C '(wrapped(raw, a.start, a.end) && !hasBreak(between))' '(false && wrapped(raw, a.start, a.end) && !hasBreak(between))' 'CG123 V5 aliasTied: bí danh trong ngoặc không bao giờ gắn'
m $C '(a.start >= numEnd && plainBetween(between))' '(false && a.start >= numEnd && plainBetween(between))' 'CG126 V5 aliasTied: bí danh sau con số không bao giờ gắn'
m $C 'return len(c.ids) == 1 || s.aliasTied(raw, os, k, numEnd)' 'return true' 'CG131 V5 ownedBy: bí danh mơ hồ chứa chủ thể luôn là chủ'
m $C 'return len(c.ids) == 1 || s.aliasTied(raw, os, k, numEnd)' 'return len(c.ids) == 1' 'CG132 V5 ownedBy: bí danh mơ hồ không bao giờ là chủ'
m $C 'strings.EqualFold(strings.TrimSpace(raw[os[k-1].end:c.start]), "của")' 'strings.EqualFold(raw[os[k-1].end:c.start], "của")' 'CG136 V5 ownedBy: "của" không bỏ khoảng trắng'
m $C '&& !bareWrapped(raw, raw[os[k-1].end:c.start], c.end) {' '&& !bareWrapped(raw, raw[os[k-1].end:c.start], c.end) || true {' 'CG139 V5 ownedBy: mọi định ngữ nối bằng gì cũng gắn'
m $C 'if c.end <= numStart {
			k = i' 'if c.end <= numStart && k < 0 {
			k = i' 'CG141 V5 numberOwned: lấy thứ ĐẦU TIÊN trước con số thay vì gần nhất'
m $C 'if k >= 0 && !s.ownedBy(raw, os, k, numEnd) {' 'if !s.ownedBy(raw, os, k, numEnd) {' 'CG143 V5 numberOwned: không có thứ nào trước con số vẫn đọc os[-1]'
m $C 'if k >= 0 && !s.ownedBy(raw, os, k, numEnd) {
		return false
	}' 'if k >= 0 && !s.ownedBy(raw, os, k, numEnd) {
		return true
	}' 'CG144 V5 numberOwned: thứ gần nhất chưa gắn thì vẫn coi là gắn'
m $C 'if c.start >= lo && len(c.ids) > 1 && slices.Contains(c.ids, s.id)' 'if c.start >= lo && len(c.ids) >= 1 && slices.Contains(c.ids, s.id)' 'CG147 V5 numberOwned: tên đơn cũng phải qua aliasTied'
m $C 'if normalization.AmbiguousNumberWords(sp.Raw) {' 'if false && normalization.AmbiguousNumberWords(sp.Raw) {' 'CG149 V5 K5 bỏ lời gọi AmbiguousNumberWords: số bằng chữ mơ hồ vẫn phán quyết'
m $C 'for _, src := range m.Sources {
			if len(used) == 1 && used[0] == src.Type {' 'for _, src := range m.Sources[:1] {
			if len(used) == 1 && used[0] == src.Type {' 'CG152 V5 qualifierPhrases chỉ nguồn đầu của mỗi chỉ số (ERP, bảng tổng hợp không còn)'
m $I '		if d.Weak {
			out = append(out, d.Keywords...)' '		if !d.Weak {
			out = append(out, d.Keywords...)' 'IK2 V5 WeakKeywords trả từ khoá ý định KHÔNG yếu'
m $I '			out = append(out, d.Keywords...)
		}
	}
	return out
}

// Def trả' '			out = append(out, d.Keywords[:0]...)
		}
	}
	return out
}

// Def trả' 'IK4 V5 WeakKeywords không trả từ khoá nào'
# ---- Vòng 5, nhóm THIẾU TEST của 03c2c54: các dòng dưới SỐNG ở 03c2c54; từ 95dea0f đã có test (D1–D20 của bản đột biến vòng 1, V5S-01…V5S-06) nên BỊ BẮT (đo lại 22/10/2026, v5db2). ----
m $C '!subj.numberOwned(raw, os, firstStart, firstEnd, lo)' '!subj.numberOwned(raw, os, firstStart, firstEnd, 0)' 'CG50 V5 numberOwned xét bí danh cả ở ngữ cảnh (lo = 0)'
m $C 'sources, metrics := a.qualifierPhrases(idef, used)' 'sources, metrics := a.qualifierPhrases(idef, nil)' 'CG53 V5 qualifierPhrases không biết nguồn đã dùng (used = nil)'
m $C 'if normalization.AmbiguousNumberWords(sp.Raw) {' 'if normalization.AmbiguousNumberWords(raw) {' 'CG54 V5 K5 kiểm số bằng chữ mơ hồ trên CẢ câu thay vì trên con số xác nhận'
m $C '				continue // tên đúng nguồn đã dùng' '				_ = src // tên đúng nguồn đã dùng' 'CG60 V5 qualifierPhrases: tên đúng nguồn đã dùng cũng là nguồn khác'
m $C 'for _, p := range slices.Concat(a.RT.Intents.MetricKeywords(), a.RT.QuantityMarkers) {' 'for _, p := range slices.Concat(a.RT.QuantityMarkers) {' 'CG64 V5 qualifierPhrases: từ khoá ý định số liệu khác không là chỉ số khác'
m $C 'strings.ContainsRune(".,/-:", r)' 'strings.ContainsRune(".,/-:;", r)' 'CG87 V5 plainBetween: thêm dấu chấm phẩy giữa hai chữ số'
m $C '&& i > 0 && i+1 < len(rs) && unicode.IsDigit(rs[i-1])' '&& i > 0 && i+1 <= len(rs) && unicode.IsDigit(rs[i-1])' 'CG92 V5 plainBetween: dấu cuối chuỗi đọc rs[len]'
m $C 'unicode.IsDigit(rs[i-1]) && unicode.IsDigit(rs[i+1])' 'unicode.IsDigit(rs[i+1])' 'CG95 V5 plainBetween: chỉ xét chữ số bên phải'
m $C 'case '"'"','"'"', '"'"';'"'"', '"'"'.'"'"', '"'"'?'"'"', '"'"'!'"'"':' 'case '"'"','"'"', '"'"';'"'"', '"'"'.'"'"', '"'"'!'"'"':' 'CG100 V5 hasBreak: dấu hỏi không là ngắt mệnh đề'
m $C 'case '"'"','"'"', '"'"';'"'"', '"'"'.'"'"', '"'"'?'"'"', '"'"'!'"'"':' 'case '"'"','"'"', '"'"';'"'"', '"'"'.'"'"', '"'"'?'"'"':' 'CG101 V5 hasBreak: dấu chấm than không là ngắt mệnh đề'
m $C 'if !(i > 0 && i+1 < len(s) && s[i-1] >= '"'"'0'"'"' && s[i-1] <= '"'"'9'"'"' && s[i+1] >= '"'"'0'"'"' && s[i+1] <= '"'"'9'"'"') {' 'if !(i > 0 && i+1 < len(s) && (s[i-1] >= '"'"'0'"'"' && s[i-1] <= '"'"'9'"'"' || s[i+1] >= '"'"'0'"'"' && s[i+1] <= '"'"'9'"'"')) {' 'CG102 V5 hasBreak: một chữ số bên cạnh là đủ để không ngắt (|| thay &&)'
m $C 'if !(i > 0 && i+1 < len(s) && s[i-1] >= '"'"'0'"'"' && s[i-1] <= '"'"'9'"'"' && s[i+1] >= '"'"'0'"'"' && s[i+1] <= '"'"'9'"'"') {' 'if !(i > 0 && i+1 < len(s) && s[i-1] >= '"'"'0'"'"' && s[i-1] <= '"'"'9'"'"') {' 'CG103 V5 hasBreak: chỉ xét chữ số bên trái'
m $C 'if !(i > 0 && i+1 < len(s) && s[i-1] >= '"'"'0'"'"' && s[i-1] <= '"'"'9'"'"' && s[i+1] >= '"'"'0'"'"' && s[i+1] <= '"'"'9'"'"') {' 'if !(i > 0 && i+1 < len(s) && s[i+1] >= '"'"'0'"'"' && s[i+1] <= '"'"'9'"'"') {' 'CG104 V5 hasBreak: chỉ xét chữ số bên phải'
m $C 'if !(i > 0 && i+1 < len(s) && s[i-1]' 'if !(i > 0 && i+1 <= len(s) && s[i-1]' 'CG105 V5 hasBreak: dấu cuối chuỗi đọc s[len]'
m $C 'rest := strings.TrimLeft(raw[afterEnd:], " ")' 'rest := raw[afterEnd:]' 'CG106 V5 bareWrapped: không bỏ khoảng trắng trước ngoặc đóng'
m $C 'if len(c.ids) != 1 || c.ids[0] != s.id {' 'if len(c.ids) < 1 || c.ids[0] != s.id {' 'CG117 V5 aliasTied: bí danh mơ hồ có chủ thể đứng đầu cũng là lần nhắc rõ'
m $C '		if d < bestDist {' '		if best < 0 || (d < bestDist && false) {' 'CG118 V5 aliasTied: lấy lần nhắc rõ ĐẦU thay vì gần nhất'
m $C '(wrapped(raw, a.start, a.end) && !hasBreak(between))' '(wrapped(raw, a.start, a.end) && true)' 'CG121 V5 aliasTied: bí danh trong ngoặc vẫn gắn dù có ngắt mệnh đề từ lần nhắc rõ'
m $C '(wrapped(raw, a.start, a.end) && !hasBreak(between))' '(true && !hasBreak(between))' 'CG122 V5 aliasTied: bí danh không ngoặc, không ngắt mệnh đề thì gắn'
m $C '(a.start >= numEnd && plainBetween(between))' '(a.start >= numEnd && true)' 'CG124 V5 aliasTied: bí danh sau con số gắn dù có dấu câu ngăn'
m $C 'return bareWrapped(raw, raw[a.end:c.start], c.end)' 'return true' 'CG128 V5 aliasTied: bí danh đứng trước lần nhắc rõ luôn gắn'
m $C 'if k == 0 || os[k-1].end > c.start {' 'if os[k-1].end > c.start {' 'CG133 V5 ownedBy: thứ đầu tiên không có thứ liền trước vẫn đọc os[-1]'
# CG134 — từ bước khoá, owners chỉ còn tên + thuật ngữ nằm ngoài tên: tên không chồng nhau (Resolve ghép cụm dài nhất trước, xem CG130),
# thuật ngữ trong tên bị loại ⇒ không còn hai chủ chồng nhau; trước bước khoá cụm ngữ cảnh (extra) có thể chồng thuật ngữ nên đột biến này bị bắt
# m $C 'if k == 0 || os[k-1].end > c.start {' 'if k == 0 {' 'CG134 V5 ownedBy: cụm chồng nhau vẫn cắt chuỗi giữa hai cụm'
m $C '!strings.EqualFold(strings.TrimSpace(raw[os[k-1].end:c.start]), "của")' 'strings.TrimSpace(raw[os[k-1].end:c.start]) != "của"' 'CG135 V5 ownedBy: "của" phân biệt hoa thường'
m $C 'strings.EqualFold(strings.TrimSpace(raw[os[k-1].end:c.start]), "của")' 'strings.Contains(raw[os[k-1].end:c.start], "của")' 'CG137 V5 ownedBy: "của" chỉ cần xuất hiện giữa hai cụm'
m $C 'return s.ownedBy(raw, os, k-1, numEnd)' 'return true' 'CG140 V5 ownedBy: định ngữ không cần thứ liền trước đã gắn (bỏ đệ quy)'
m $C 'if k >= 0 && !s.ownedBy(raw, os, k, numEnd) {' 'if k > 0 && !s.ownedBy(raw, os, k, numEnd) {' 'CG142 V5 numberOwned: thứ đầu tiên (k = 0) không bị kiểm'
m $C '	for i, c := range os {
		if c.start >= lo && len(c.ids) > 1' '	for i, c := range os[:0] {
		if c.start >= lo && len(c.ids) > 1' 'CG145 V5 numberOwned: bỏ vòng kiểm mọi bí danh mơ hồ trong phạm vi hỏi'
m $C '!s.aliasTied(raw, os, i, numEnd) {
			return false' '!s.aliasTied(raw, os, i, numEnd) {
			return true' 'CG146 V5 numberOwned: một bí danh chưa gắn thì dừng và coi là gắn'
m $C 'd = os[k].start - c.end' 'd = c.end - os[k].start' 'CG150 V5 aliasTied: lần nhắc rõ đứng trước, khoảng cách đảo dấu (xa nhất thắng)'
m $I '		if d.Weak {
			out = append(out, d.Keywords...)' '		if true {
			out = append(out, d.Keywords...)' 'IK3 V5 WeakKeywords trả từ khoá của mọi ý định'
# ---- Vòng 5, bốn dạng chính sách mặc định (ADR-0009 Vòng 5, bảng năm dạng): ở 03c2c54 nằm ở nhóm CHỜ QUYẾT; từ 95dea0f đã ghim trong TestCongPhanQuyetVong5 (nhãn "mặc định Vòng 5") nên BỊ BẮT. ----
m $C '(a.start >= numEnd && plainBetween(between))' '(true && plainBetween(between))' 'CG125 V5 aliasTied: bí danh trước con số cũng gắn nếu không dấu câu'
m $C 'if a.end <= c.start { // bí danh đứng trước lần nhắc rõ' 'if false && a.end <= c.start { // bí danh đứng trước lần nhắc rõ' 'CG127 V5 aliasTied: tắt nhánh bí danh đứng trước lần nhắc rõ'
m $C 'return bareWrapped(raw, raw[a.end:c.start], c.end)' 'return false' 'CG129 V5 aliasTied: bí danh đứng trước lần nhắc rõ không bao giờ gắn'
m $C '&& !bareWrapped(raw, raw[os[k-1].end:c.start], c.end) {' '&& true {' 'CG138 V5 ownedBy: không nhận định ngữ trong ngoặc trần'
# ---- Vòng 5, soát độc lập lần 2 (95dea0f, v5db2): topicQualifier / qualifierPhrases / contextOwners / AmbiguousNumberWords nhánh "rưỡi"; kỳ vọng BỊ BẮT. ----
m $C 'var out [][2]int
	for _, p := range phrases {' 'var out [][2]int
	for _, p := range phrases[:0] {' 'CG192 V5 contextOwners: không trả cụm nào'
m $C 'continue // tên đúng nguồn đã dùng' 'break // tên đúng nguồn đã dùng' 'CG194 V5 qualifierPhrases: gặp nguồn đã dùng thì bỏ cả các nguồn sau nó'
m $C '			if len(used) == 1 && used[0] == src.Type {' '			if len(used) == 1 || used[0] == src.Type {' 'CG196 V5 qualifierPhrases: một nguồn đã dùng thì mọi nguồn đều được bỏ'
m $C 'MetricIDs() {
		m, _ := a.RT.Metrics.Metric(id)
		for _, src := range m.Sources {
			if len(used) == 1 && used[0] == src.Type {' 'MetricIDs()[:1] {
		m, _ := a.RT.Metrics.Metric(id)
		for _, src := range m.Sources {
			if len(used) == 1 && used[0] == src.Type {' 'CG197 V5 qualifierPhrases: chỉ xét chỉ số đầu tiên của pack'
m $C 'sources = append(sources, p)' 'metrics = append(metrics, p)' 'CG199 V5 qualifierPhrases: tên nguồn khác xếp vào danh sách chỉ số'
m $C 'return sources, metrics' 'return metrics, sources' 'CG200 V5 qualifierPhrases: trả hai danh sách đổi chỗ'
# Bước KHOÁ (brief BUILD §9.6, ADR-0009): topicQualifier, own và cụm ngữ cảnh làm chủ (extra) đã bỏ — các dòng CG155–CG188, CG202, CG203,
# CG205, CG206, CG208, CG210, CG80, CG185, CG186 nhắm vào mã đó nên xoá; CG51, CG52, CG189–CG191, CG211 viết lại cho hai lời gọi mới.
m $C 'if len(contextOwners(raw, ctxNumbers, inName, sources, quals, lo)) > 0 {' 'if false && len(contextOwners(raw, ctxNumbers, inName, sources, quals, lo)) > 0 {' 'KH1 khoá: tắt bước khoá cho cụm nguồn khác'
m $C 'if len(contextOwners(raw, ctxNumbers, inName, metrics, quals, lo)) > 0 {' 'if false && len(contextOwners(raw, ctxNumbers, inName, metrics, quals, lo)) > 0 {' 'KH2 khoá: tắt bước khoá cho cụm chỉ số khác'
m $C 'if len(contextOwners(raw, ctxNumbers, inName, sources, quals, lo)) > 0 {
		return "source"' 'if len(contextOwners(raw, ctxNumbers, inName, sources, quals, lo)) > 0 {
		return "metric"' 'KH3 khoá: cụm nguồn khác mang nhãn metric'
m $C 'if len(contextOwners(raw, ctxNumbers, inName, metrics, quals, lo)) > 0 {
		return "metric"' 'if len(contextOwners(raw, ctxNumbers, inName, metrics, quals, lo)) > 0 {
		return "source"' 'KH4 khoá: cụm chỉ số khác mang nhãn source'
m $C 'if len(contextOwners(raw, ctxNumbers, inName, sources, quals, lo)) > 0 {' 'if len(contextOwners(raw, ctxNumbers, inName, sources, quals, len(raw))) > 0 {' 'CG51 khoá: cụm nguồn ở phạm vi hỏi cũng tính là ngữ cảnh (lo = len)'
m $C 'if len(contextOwners(raw, ctxNumbers, inName, metrics, quals, lo)) > 0 {' 'if len(contextOwners(raw, ctxNumbers, inName, metrics, quals, len(raw))) > 0 {' 'KH5 khoá: cụm chỉ số ở phạm vi hỏi cũng tính là ngữ cảnh (lo = len)'
m $C 'owners(names, in.Spans, idef.Terms)' 'owners(names, in.Spans, nil)' 'CG52 owners không loại thuật ngữ của chính ý định'
m $C 'if len(contextOwners(raw, ctxNumbers, inName, metrics, quals, lo)) > 0 {' 'if len(contextOwners(raw, ctxNumbers, inName, metrics[:0], quals, lo)) > 0 {' 'CG189 khoá: không nhận từ khoá chỉ số khác'
m $C 'if len(contextOwners(raw, ctxNumbers, inName, sources, quals, lo)) > 0 {' 'if len(contextOwners(raw, ctxNumbers, inName, sources[:0], quals, lo)) > 0 {' 'CG190 khoá: không nhận tên nguồn khác'
m $C 'if len(contextOwners(raw, ctxNumbers, inName, sources, quals, lo)) > 0 {' 'if len(contextOwners(raw, ctxNumbers, inName, slices.DeleteFunc(slices.Clone(sources), func(p string) bool { return slices.Contains(idef.Keywords, p) }), quals, lo)) > 0 {' 'CG211 khoá R5-5: cụm nguồn trùng từ khoá chính ý định ("phiếu kho") không tính ở ngữ cảnh (soát vòng 4 M13)'
m $C 'if len(contextOwners(raw, ctxNumbers, inName, sources, quals, lo)) > 0 {' 'if len(contextOwners(raw, ctxNumbers, inName, sources, quals, sentenceStart(raw, lo))) > 0 {' 'KH6 khoá: chỉ xét cụm ở câu trước câu hỏi (V5S4-02/03 lọt)'
m $C 'if len(contextOwners(raw, ctxNumbers, inName, sources, quals, lo)) > 0 {' 'if len(contextOwners(raw, ctxNumbers, inName, sources, quals, lo)) > 0 && sentenceStart(raw, lo) == 0 {' 'KH7 khoá: chỉ xét khi câu hỏi là câu đầu (V5S4-04 lọt)'
# ---- Vòng 5, lần 2, nhóm THIẾU TEST: các dòng dưới SỐNG ở 95dea0f (chưa test nào đỏ) và phải BỊ BẮT sau khi thêm các ca đề xuất của báo cáo v5db2 (đã chạy: ca đề xuất xanh trên 95dea0f, đỏ khi áp từng đột biến).
# Kỳ vọng suy từ luật, không chép từ đầu ra. IK1 / CG72 BỊ BẮT ở 03c2c54 rồi mất bảo vệ khi 95dea0f chuyển đề mục thuần sang topicQualifier. CG168 là chỗ lộ lỗ hổng thật: ca "Ngày 19/10, ERP, EX-03 …" đỏ ở 95dea0f (xem báo cáo). ----
m $I '		if contracts.RouteKind(d.Route) == contracts.RouteSQLQuery && !d.Weak {' '		if true {' 'IK1 từ khoá của ý định không phải số liệu thành ngữ cảnh hợp lệ'
m $C '&& sp.Start >= s && sp.Start < e && !inName(sp.Start, sp.End)' '&& sp.Start < e && !inName(sp.Start, sp.End)' 'CG72 V5 numberIn: bỏ cận dưới (số đứng trước cụm cũng xen giữa)'
# CG204 (tương đương) — bỏ cận ws >= from: từ chỉ kỳ dính liền mốc số nên không thể nằm sau dấu ngắt câu mở đầu đoạn [from, lo)
m $C '				if keepSrc(p) {' '				if keep(p) {' 'CG209 V5 qualifierPhrases: tên nguồn trùng từ khoá chính ý định ("phiếu kho") bị bỏ khỏi nguồn khác (R5-5)'
m $C '			if len(used) == 1 && used[0] == src.Type {' '			if len(used) >= 1 && used[0] == src.Type {' 'CG195 V5 qualifierPhrases: nhiều nguồn đã dùng thì nguồn đầu vẫn được bỏ'
m $C 'metrics = append(metrics, p)' 'sources = append(sources, p)' 'CG198 V5 qualifierPhrases: từ khoá chỉ số khác xếp vào danh sách nguồn'
# ---- Vòng 5, nhóm CHỜ QUYẾT (đã đo lại ở 95dea0f; chạy trên corpus 110.208 câu tổ hợp đề mục × chủ thể × số × đuôi × ý định × nguồn, không đổi phán quyết ở các dòng dưới trừ ghi chú). Chủ dự án quyết thì ghim test / sửa mã rồi bỏ dấu # (đột biến phải đỏ). ----
# CG148 — R2-6 "EX-03 của chương trình mới …": V5 giữ phán quyết; đột biến này chính là bản đóng lỗ (0 lệch trên corpus: corpus không có dạng đó).
# m $C 'if c.start >= lo && len(c.ids) > 1 && slices.Contains(c.ids, s.id)' 'if c.start >= lo && len(c.ids) > 1' 'CG148 V5 numberOwned: bí danh mơ hồ KHÔNG chứa chủ thể cũng phải qua aliasTied'
# CG151 — lần nhắc rõ đứng SAU bí danh, khoảng cách đảo dấu (xa nhất thắng): chưa có chính sách ghim (0 lệch trên corpus).
# m $C 'd := c.start - os[k].end' 'd := os[k].end - c.start' 'CG151 V5 aliasTied: lần nhắc rõ đứng sau, khoảng cách đảo dấu (xa nhất thắng)'
# CG30 / CG35 — hai lượt kiểm chủ thể ở phạm vi câu / mệnh đề hỏi bị numberOwned + E3 bao trùm (SỐNG từ khi V5 vào). CG35: 0 lệch; CG30: 3.372 câu chỉ đổi NHÃN (vd subject → value khi câu còn một chiều chưa giải khác), 0 câu đổi phán quyết.
# Chờ quyết: xoá lượt kiểm thừa (gọn hơn) hay giữ làm phòng thủ. CG24 (tắt subjectFacts.gap) đã BỊ BẮT bởi TestSubjectGapPhamViHoi.
# m $C '	for _, from := range []int{lo, qs} {' '	for _, from := range []int{lo, lo+0*qs} {' 'CG30 kiểm chủ thể chỉ theo phạm vi đã lùi về đầu câu (bỏ mệnh đề hỏi)'
# m $C '	for _, from := range []int{lo, qs} {' '	for _, from := range []int{qs, qs} {' 'CG35 kiểm chủ thể chỉ ở mệnh đề hỏi (bỏ phạm vi câu — hồi quy T2.2, soát lần ba)'
# CG179 / CG201 — nhãn khi một cụm vừa là tên nguồn vừa là từ khoá chỉ số ('Hoá đơn,' 'Cảm biến,' đều có ở sources và ở MetricKeywords; 'Phiếu kho,' khi ý định đang hỏi là ý định khác): đổi thứ tự src / met hay bỏ bí danh chỉ đổi nhãn
# trong plan.Reason (CG179: 1.128 câu; CG201: 90 câu), phán quyết không đổi. Chờ quyết: ghim "source" ưu tiên bằng ca TestVerdictGap (vd "Hoá đơn, EX-03 hôm nay được cấp 3900 L đúng không?" want source) hay coi nhãn không là cam kết.
# m $C 'if src {
# 		return "source"
# 	}
# 	return "metric"' 'if met {
# 		return "metric"
# 	}
# 	return "source"' 'CG179 V5 topicQualifier: chỉ số được ưu tiên hơn nguồn khi đề mục có cả hai'
# m $C 'for _, p := range append([]string{src.Label}, src.Aliases...) {' 'for _, p := range []string{src.Label} {' 'CG201 V5 qualifierPhrases: bí danh của nguồn không là nguồn khác'
# ---- Vòng 5, nhóm TƯƠNG ĐƯƠNG: không đầu vào nào đổi phán quyết (đo v5db2: 0 câu lệch trên corpus 110.208 câu; NW18 / NW19 vét cạn 5.380.839 dãy ≤ 7 từ trên 9 loại từ: cùng 446.592 dãy mơ hồ, cùng băm). Không thêm dòng chạy;
# dòng m giữ ở dạng chú thích để ai nghi ngờ thì bỏ # chạy lại. ----
# CG47 — numStart đặt ở cuối con số: không cụm owners nào nằm trong span của con số xác nhận (thuật ngữ vật tư "dầu" của "dầu 4k" đã bị loại bởi idef.Terms)
# m $C '				firstStart, firstEnd = sp.Start, sp.End
# ' '				firstStart, firstEnd = sp.End, sp.End
# ' 'CG47 V5 con số xác nhận đầu: numStart đặt ở CUỐI con số'
# CG48 — firstStart / firstEnd ở lần nhắc cuối: giữa hai lần nhắc con số chỉ được có khoảng trắng và ngoặc (dòng "value && … strings.Trim(raw[lastEnd:sp.Start], " ()")"), nên không tên / cụm nào nằm giữa chúng
# m $C '			if !value {
# 				firstStart, firstEnd = sp.Start, sp.End' '			if true {
# 				firstStart, firstEnd = sp.Start, sp.End' 'CG48 V5 firstStart/firstEnd đo ở lần nhắc CUỐI thay vì đầu'
# CG68 — sp[1] < lo: ký tự ngay trước lo luôn là dấu ngắt (, ; . ? !) nên không cụm nào kết thúc đúng tại lo
# m $C 'if sp[1] <= lo && !numberIn(sp[1], lo) {' 'if sp[1] < lo && !numberIn(sp[1], lo) {' 'CG68 V5 R-1 biên: sp[1] < lo thay vì <='
# CG73 — numberIn bỏ !inName: tên có số nằm giữa cụm và lo luôn là chủ gần hơn cụm đó, nên cụm không còn là thứ gần nhất trước con số
# m $C '&& sp.Start >= s && sp.Start < e && !inName(sp.Start, sp.End)' '&& sp.Start >= s && sp.Start < e' 'CG73 V5 numberIn: số trong tên thực thể cũng xen giữa'
# CG81 — owners.extra nằm trong cụm tên: chỉ khác khi tên thực thể chứa từ khoá / nhãn nguồn — danh mục seed không có tên như vậy (phụ thuộc dữ liệu)
# m $C 'if !inName(sp[0], sp[1]) {
# 			out = append(out, ownerRef{sp[0], sp[1], nil})' 'if true {
# 			out = append(out, ownerRef{sp[0], sp[1], nil})' 'CG81 V5 owners: cụm ở ngữ cảnh nằm trong cụm tên vẫn tính riêng'
# CG130 — aliasTied trả true khi hai lần nhắc chồng nhau: Resolve ghép cụm dài nhất trước (resolver.go:89) nên các lần nhắc không chồng nhau
# m $C '		return bareWrapped(raw, raw[a.end:c.start], c.end)
# 	}
# 	return false
# }' '		return bareWrapped(raw, raw[a.end:c.start], c.end)
# 	}
# 	return true
# }' 'CG130 V5 aliasTied: khoảng chồng nhau coi là gắn'
# CG193 — cụm rỗng: PhraseSpans(raw, "") không bao giờ khớp (matchAliasPrefix trả false khi alias == "", text.go:259) nên `p != ""` ở qualifierPhrases thừa với mọi pack, không chỉ pack hiện có (thay CG59 của 03c2c54)
# m $C 'return p != "" && ' 'return ' 'CG193 V5 qualifierPhrases: cụm rỗng không bị bỏ qua'
# (CG59 / CG62 của 03c2c54 — cụm rỗng, bí danh của tên nguồn — nay là CG193 / CG201 vì qualifierPhrases tách khỏi contextOwners.)
# Luật 5 dựng lại trên readViNumberWords (soát D-OP-14 F1): NW1–NW20 cũ nhắm bản tail() ba từ đã bỏ.
m $W '		if ok && colloquial {' '		if ok && colloquial && false {' 'NW21 cổng không thấy số bằng chữ nói tắt'
m $W '		if next > i {
			i = next - 1
		}' '		if false {
			i = next - 1
		}' 'NW22 xét lại từ giữa một dãy số đã đọc'
m $W '				pending = -1
				colloquial = true' '				pending = -1' 'NW23 "<hàng> <chữ số> rưỡi" không là nói tắt'
m $W '		colloquial = colloquial || implied > 1' '		colloquial = colloquial' 'NW24 "<hàng> <chữ số>" cuối dãy không là nói tắt'
m $W '			colloquial = true // "linh" / "lẻ" đứng cuối dãy: "hơn một chút", không phải số đúng' '			_ = colloquial' 'NW25 "linh" / "lẻ" cuối dãy không là xấp xỉ'
m $W '		if toks[i].kind != tokWord {
			continue
		}
		if _, isNum := lookupNumberWord(toks[i].text); !isNum {' '		if toks[i].kind != tokWord {
			continue
		}
		if _, isNum := lookupNumberWord(toks[i].text); isNum {' 'NW26 chỉ xét từ KHÔNG phải số'
m $C '		if _, period := normalization.PeriodQualifier(raw, sp.Start, sp.End); period {
			return true
		}
		if strings.Contains(sp.Rule,' '		if _, period := normalization.PeriodQualifier(raw, sp.Start, sp.End); period && false {
			return true
		}
		if strings.Contains(sp.Rule,' 'KH8 khoá F2: số của mốc ("tuần 43") thành lượng riêng của cụm'
m $C '		return !same // lượng khác chiều đo' '		return false && !same // lượng khác chiều đo' 'KH9 khoá F2: lượng khác chiều đo ("8h") thành lượng riêng của cụm'
# Soát khoá vòng 2 (phát hiện 1–3): "<chục> trăm" là nói tắt; số trần mang đơn vị nhờ "dầu" không phải lượng riêng của cụm; số riêng của cụm phải
# đứng trước cụm kế tiếp.
m $W '			if group > 0 {
				colloquial = true' '			if false {
				colloquial = true' 'NW27 "mười hai trăm" không là nói tắt'
m $C '		if strings.Contains(sp.Rule, "unit_from_term") && !unitShown(contracts.Span{Kind: contracts.ValueNumber, Raw: sp.Raw, Rule: sp.Rule}) {' '		if strings.Contains(sp.Rule, "unit_from_term") && !unitShown(contracts.Span{Kind: contracts.ValueNumber, Raw: sp.Raw, Rule: sp.Rule}) && false {' 'KH10 số trần mang đơn vị nhờ "dầu" thành lượng riêng của cụm'
m $C '				if st >= sp[1] && st < end {' '				if false && st >= sp[1] && st < end {' 'KH11 số của cụm kế tiếp thành số riêng của cụm trước'
m $C '			if sp[1] <= lo {
				starts = append(starts, sp[0])' '			if false {
				starts = append(starts, sp[0])' 'KH12 không biết đầu các cụm ở ngữ cảnh'
m $C '	quals := slices.Concat(sources, metrics, a.usedSourceNames(used))' '	quals := slices.Concat(sources, a.usedSourceNames(used))' 'KH13 ranh giới chỉ tính cụm nguồn, bỏ cụm chỉ số'
# X3 — bỏ slices.Clone ở ctxNumbers: DeleteFunc sửa mảng in.Spans tại chỗ (owners đọc in.Spans sau đó); bộ test hiện có không bắt — giữ Clone
# m $C 'ctxNumbers := slices.DeleteFunc(slices.Clone(in.Spans), func' 'ctxNumbers := slices.DeleteFunc(in.Spans, func' 'X3 ctxNumbers sửa in.Spans tại chỗ'
# Soát khoá vòng 3 (S2-1, S2-2; sai chiều ⇒ mở lại theo §9.6): tên nguồn đã dùng là ranh giới; số không đơn vị chỉ "tự mang chiều đo" khi có
# hậu tố nhân hay hàng nghìn trở lên.
m $C '	return normalization.HasThousandsWord(sp.Raw) || (strings.Contains(sp.Rule, "+multiplier") && !normalization.HasNumberWord(sp.Raw))' '	return strings.ContainsFunc(sp.Raw, unicode.IsLetter)' 'KH14 số nhỏ đọc thành chữ ("mười chín") thành có chiều đo'
m $C '	return normalization.HasThousandsWord(sp.Raw) || (strings.Contains(sp.Rule, "+multiplier") && !normalization.HasNumberWord(sp.Raw))' '	return strings.Contains(sp.Rule, "+multiplier") && !normalization.HasNumberWord(sp.Raw)' 'KH15 số bằng chữ có hàng nghìn ("bốn ngàn") thành số trần'
m $C '	return normalization.HasThousandsWord(sp.Raw) || (strings.Contains(sp.Rule, "+multiplier") && !normalization.HasNumberWord(sp.Raw))' '	return normalization.HasThousandsWord(sp.Raw)' 'KH16 số có hậu tố nhân tiếng lóng ("3k") thành số trần'
m $C '!unitShown(contracts.Span{Kind: contracts.ValueNumber, Raw: sp.Raw, Rule: sp.Rule}) {' '!strings.ContainsFunc(sp.Raw, unicode.IsLetter) {' 'KH17 số bằng chữ không hàng nghìn mang đơn vị nhờ "dầu" thành lượng riêng của cụm ("Dầu ERP bốn mươi ba")'
m $C '	quals := slices.Concat(sources, metrics, a.usedSourceNames(used))' '	quals := slices.Concat(sources, metrics)' 'KH18 tên nguồn đã dùng không là ranh giới ("ERP, phiếu xuất kho 3850 L")'
m $C '				out = append(append(out, src.Label), src.Aliases...)' '				out = append(out, src.Label)' 'KH19 ranh giới bỏ bí danh nguồn đã dùng ("ERP, phiếu kho 3850 L")'
m $C '	return normalization.HasThousandsWord(sp.Raw) || (strings.Contains(sp.Rule, "+multiplier") && !normalization.HasNumberWord(sp.Raw))' '	return normalization.HasThousandsWord(sp.Raw) || strings.Contains(sp.Rule, "+multiplier")' 'KH20 nhân "trăm" / "chục" ("ERP 3 trăm") thành có chiều đo'
