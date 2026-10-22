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
m $C 'extra := contextOwners(raw, in.Spans, inName, slices.Concat(sources, metrics), lo)' 'extra := contextOwners(raw, in.Spans, inName, slices.Concat(sources, metrics), len(raw))' 'CG51 V5 contextOwners coi cả phạm vi hỏi là ngữ cảnh (lo = len)'
m $C 'owners(names, in.Spans, idef.Terms, extra)' 'owners(names, in.Spans, nil, extra)' 'CG52 V5 owners không loại thuật ngữ của chính ý định'
m $C '				return "value" // "một nghìn hai"' '				return "subject" // "một nghìn hai"' 'CG55 V5 K5 trả nhãn sai (subject thay vì value)'
m $C '	keep := func(p string) bool { return keepSrc(p) && !slices.Contains(idef.Keywords, p) }' '	keep := func(p string) bool { return keepSrc(p) }' 'CG56 V5 R-1 từ khoá của chính ý định thành chủ của con số'
m $C '	keepSrc := func(p string) bool { return p != "" && !slices.Contains(weak, p) }' '	keepSrc := func(p string) bool { return p != "" && !slices.Contains(weak[:0], p) }' 'CG57 V5 R-1 từ khoá ý định yếu ("bao nhiêu") thành chủ của con số'
m $C '	keep := func(p string) bool { return keepSrc(p) && !slices.Contains(idef.Keywords, p) }' '	keep := func(p string) bool { return p != "" }' 'CG58 V5 qualifierPhrases không loại trừ từ khoá nào (skip vô hiệu)'
m $C '			if len(used) == 1 && used[0] == src.Type {' '			if len(used) == 1 && used[0] != src.Type {' 'CG61 V5 qualifierPhrases đảo điều kiện: chỉ tên nguồn đã dùng là nguồn khác'
m $C 'for _, p := range append([]string{src.Label}, src.Aliases...) {' 'for _, p := range src.Aliases {' 'CG63 V5 qualifierPhrases: nhãn của nguồn không là nguồn khác'
m $C 'for _, p := range slices.Concat(a.RT.Intents.MetricKeywords(), a.RT.QuantityMarkers) {' 'for _, p := range slices.Concat(a.RT.Intents.MetricKeywords()) {' 'CG65 V5 qualifierPhrases: từ chỉ lượng không là chỉ số khác'
m $C 'if sp[1] <= lo && !numberIn(sp[1], lo) {' 'if sp[1] <= lo && (!numberIn(sp[1], lo) || true) {' 'CG66 V5 R-1 cụm đã có số của riêng nó vẫn là chủ của con số sau (bỏ numberIn)'
m $C 'if sp[1] <= lo && !numberIn(sp[1], lo) {' 'if !numberIn(sp[1], lo) {' 'CG67 V5 R-1 cụm ở phạm vi hỏi cũng tính là chủ (bỏ sp[1] <= lo)'
m $C 'return (sp.Kind == contracts.ValueQuantity || sp.Kind == contracts.ValueNumber) && sp.Start >= s && sp.Start < e' 'return sp.Kind == contracts.ValueQuantity && sp.Start >= s && sp.Start < e' 'CG69 V5 numberIn: số trần không tính là "số xen giữa"'
m $C 'return (sp.Kind == contracts.ValueQuantity || sp.Kind == contracts.ValueNumber) && sp.Start >= s && sp.Start < e' 'return sp.Kind == contracts.ValueNumber && sp.Start >= s && sp.Start < e' 'CG70 V5 numberIn: lượng có đơn vị không tính là "số xen giữa"'
m $C '&& sp.Start >= s && sp.Start < e && !inName(sp.Start, sp.End)' '&& sp.Start >= s && !inName(sp.Start, sp.End)' 'CG71 V5 numberIn: bỏ cận trên (số ở phạm vi hỏi cũng xen giữa)'
m $C 'out = append(out, ownerRef{m.start, m.end, m.ids})' 'out = append(out, ownerRef{m.start, m.end, nil})' 'CG74 V5 owners: tên thực thể không mang ứng viên'
m $C 'for _, m := range names {
		out = append(out, ownerRef{m.start, m.end, m.ids})' 'for _, m := range names[:0] {
		out = append(out, ownerRef{m.start, m.end, m.ids})' 'CG75 V5 owners: tên thực thể không là chủ'
m $C 'if sp.Kind != contracts.ValueTerm || slices.Contains(ownTerms, sp.Value.Text) || inName(sp.Start, sp.End) {' 'if true {' 'CG76 V5 owners: thuật ngữ ("dự án", "công trình") không là chủ'
m $C 'if sp.Kind != contracts.ValueTerm || slices.Contains(ownTerms, sp.Value.Text) || inName(sp.Start, sp.End) {' 'if sp.Kind != contracts.ValueTerm || inName(sp.Start, sp.End) {' 'CG77 V5 owners: thuật ngữ của chính ý định ("dầu") cũng là chủ'
m $C 'slices.Contains(ownTerms, sp.Value.Text) || inName(sp.Start, sp.End) {' 'slices.Contains(ownTerms, sp.Value.Text) {' 'CG78 V5 owners: thuật ngữ nằm trong cụm tên vẫn tính riêng'
m $C 'if sp.Kind != contracts.ValueTerm || slices.Contains(ownTerms' 'if sp.Kind == contracts.ValueTerm || slices.Contains(ownTerms' 'CG79 V5 owners: đảo loại span (thuật ngữ bị bỏ, mọi span khác thành chủ)'
m $C 'for _, sp := range extra {' 'for _, sp := range extra[:0] {' 'CG80 V5 owners: cụm ở ngữ cảnh (extra) không là chủ'
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
m $C 'for _, src := range m.Sources {' 'for _, src := range m.Sources[:1] {' 'CG152 V5 qualifierPhrases chỉ nguồn đầu của mỗi chỉ số (ERP, bảng tổng hợp không còn)'
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
m $W 'return k == vwScale || k == vwHundred }' 'return k == vwHundred }' 'NW1 V5 số bằng chữ mơ hồ: bỏ "<nghìn/triệu/tỷ> <chữ số>"'
m $W 'return k == vwScale || k == vwHundred }' 'return k == vwScale }' 'NW2 V5 số bằng chữ mơ hồ: bỏ "trăm <chữ số>"'
m $W 'return k == vwScale || k == vwHundred }' 'return k == vwScale || k == vwHundred || k == vwMuoi }' 'NW4 V5 số bằng chữ mơ hồ: "mười hai" thành mơ hồ'
m $W 'return k == vwScale || k == vwHundred }' 'return k == vwScale || k == vwHundred || k == vwFiller }' 'NW5 V5 số bằng chữ mơ hồ: "linh hai" / "lẻ một" thành mơ hồ'
m $W '(last == vwDigit && scaled(prev))' '(last != 0 && scaled(prev))' 'NW6 V5 số bằng chữ mơ hồ: từ cuối bất kỳ ("rưỡi") sau hàng cũng mơ hồ'
m $W '			if tail() {
				return true
			}' '			if false && tail() {
				return true
			}' 'NW8 V5 số bằng chữ mơ hồ: không kiểm khi dãy bị ngắt bởi từ không phải số'
m $W '	return tail()
}' '	return false
}' 'NW9 V5 số bằng chữ mơ hồ: không kiểm khi dãy kết thúc cuối chuỗi'
m $W 'prev2, prev, last = prev, last, w.kind' 'prev2, prev, last = prev2, prev, w.kind' 'NW11 V5 số bằng chữ mơ hồ: không nhớ từ liền trước'
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
m $C 'if k == 0 || os[k-1].end > c.start {' 'if k == 0 {' 'CG134 V5 ownedBy: cụm chồng nhau vẫn cắt chuỗi giữa hai cụm'
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
m $W 'return k == vwScale || k == vwHundred }' 'return k == vwScale || k == vwHundred || k == vwTens }' 'NW3 V5 số bằng chữ mơ hồ: "hai mươi lăm" thành mơ hồ'
m $W 'prev2, prev, last = 0, 0, 0' 'prev2, prev, last = prev2, prev, last' 'NW7 V5 số bằng chữ mơ hồ: từ không phải số không ngắt dãy'
m $W 'lookupNumberWord(strings.Trim(f, ".,;:!?()[]\"'"'"'"))' 'lookupNumberWord(f)' 'NW10 V5 số bằng chữ mơ hồ: không bỏ dấu câu dính vào từ'
# ---- Vòng 5, bốn dạng chính sách mặc định (ADR-0009 Vòng 5, bảng năm dạng): ở 03c2c54 nằm ở nhóm CHỜ QUYẾT; từ 95dea0f đã ghim trong TestCongPhanQuyetVong5 (nhãn "mặc định Vòng 5") nên BỊ BẮT. ----
m $C '(a.start >= numEnd && plainBetween(between))' '(true && plainBetween(between))' 'CG125 V5 aliasTied: bí danh trước con số cũng gắn nếu không dấu câu'
m $C 'if a.end <= c.start { // bí danh đứng trước lần nhắc rõ' 'if false && a.end <= c.start { // bí danh đứng trước lần nhắc rõ' 'CG127 V5 aliasTied: tắt nhánh bí danh đứng trước lần nhắc rõ'
m $C 'return bareWrapped(raw, raw[a.end:c.start], c.end)' 'return false' 'CG129 V5 aliasTied: bí danh đứng trước lần nhắc rõ không bao giờ gắn'
m $C '&& !bareWrapped(raw, raw[os[k-1].end:c.start], c.end) {' '&& true {' 'CG138 V5 ownedBy: không nhận định ngữ trong ngoặc trần'
# ---- Vòng 5, soát độc lập lần 2 (95dea0f, v5db2): topicQualifier / qualifierPhrases / contextOwners / AmbiguousNumberWords nhánh "rưỡi"; kỳ vọng BỊ BẮT. ----
m $C 'from := sentenceStart(raw, lo)' 'from := 0' 'CG155 V5 topicQualifier: đoạn đề mục tính từ đầu văn bản thay vì đầu câu'
m $C 'from := sentenceStart(raw, lo)' 'from := lo' 'CG156 V5 topicQualifier: không có đoạn đề mục (from = lo)'
m $C 'if sp[0] >= from && sp[1] <= lo {' 'if sp[1] <= lo {' 'CG157 V5 topicQualifier: cụm ở câu trước đoạn đề mục cũng được đánh dấu (bỏ cận dưới)'
m $C 'if sp[0] >= from && sp[1] <= lo {' 'if sp[0] >= from {' 'CG158 V5 topicQualifier: cụm ở phạm vi hỏi cũng được đánh dấu (bỏ cận trên)'
m $C 'if sp[0] >= from && sp[1] <= lo {' 'if sp[0] > from && sp[1] <= lo {' 'CG159 V5 topicQualifier: cụm ở ngay đầu câu không được đánh dấu (> thay >=)'
m $C 'src, met := markPhrases(sources), markPhrases(metrics)' 'src, met := markPhrases(sources), markPhrases(nil)' 'CG161 V5 topicQualifier: từ khoá chỉ số khác không được đánh dấu'
m $C 'src, met := markPhrases(sources), markPhrases(metrics)' 'src, met := markPhrases(nil), markPhrases(metrics)' 'CG162 V5 topicQualifier: tên nguồn khác không được đánh dấu'
m $C 'src, met := markPhrases(sources), markPhrases(metrics)' 'src, met := markPhrases(metrics), markPhrases(sources)' 'CG163 V5 topicQualifier: hai danh sách đổi chỗ (nhãn source / metric đảo)'
m $C 'if !src && !met {' 'if !src || !met {' 'CG164 V5 topicQualifier: cần CẢ nguồn lẫn chỉ số mới là đề mục'
m $C 'if (sp.Kind == contracts.ValueDate || sp.Kind == contracts.ValuePeriod) && sp.Start >= from && sp.End <= lo {' 'if (sp.Kind == contracts.ValueDate || sp.Kind == contracts.ValuePeriod) && sp.End <= lo {' 'CG170 V5 topicQualifier: mốc ở câu trước cũng được đánh dấu (bỏ cận dưới)'
m $C 'if (sp.Kind == contracts.ValueDate || sp.Kind == contracts.ValuePeriod) && sp.Start >= from && sp.End <= lo {' 'if (sp.Kind == contracts.ValueDate || sp.Kind == contracts.ValuePeriod) && sp.Start >= from {' 'CG171 V5 topicQualifier: mốc ở phạm vi hỏi cũng được đánh dấu (bỏ cận trên)'
m $C 'if !covered[i-from] && !unicode.IsSpace(r) && !strings.ContainsRune(claimSeparators, r) {' 'if false && !covered[i-from] && !unicode.IsSpace(r) && !strings.ContainsRune(claimSeparators, r) {' 'CG173 V5 topicQualifier: bỏ kiểm thuần (chữ lạ trong đoạn đề mục vẫn là đề mục)'
m $C '!covered[i-from] && !unicode.IsSpace(r) && !strings.ContainsRune(claimSeparators, r)' '!covered[i-from] && !strings.ContainsRune(claimSeparators, r)' 'CG174 V5 topicQualifier: khoảng trắng không được tha'
m $C '!covered[i-from] && !unicode.IsSpace(r) && !strings.ContainsRune(claimSeparators, r)' '!covered[i-from] && !unicode.IsSpace(r)' 'CG175 V5 topicQualifier: dấu ngăn không được tha'
m $C '			return ""
		}
		i += n
	}
	if src {' '			break
		}
		i += n
	}
	if src {' 'CG177 V5 topicQualifier: gặp chữ lạ thì dừng kiểm chứ không bỏ đề mục'
m $C 'if src {
		return "source"
	}
	return "metric"' 'if !src {
		return "source"
	}
	return "metric"' 'CG178 V5 topicQualifier: đảo điều kiện nhãn source'
m $C 'return "source"
	}
	return "metric"' 'return "metric"
	}
	return "metric"' 'CG181 V5 topicQualifier: đề mục có nguồn cũng mang nhãn metric'
m $C 'covered[i-from] = true' 'covered[i-from] = false' 'CG182 V5 topicQualifier: mark không đánh dấu gì'
m $C 'hit = true' 'hit = false' 'CG184 V5 topicQualifier: markPhrases không báo trúng'
m $C '; gap != "" {
		return gap
	}
	extra :=' '; false && gap != "" {
		return gap
	}
	extra :=' 'CG185 V5 verdictGap: tắt lời gọi topicQualifier'
m $C 'a.topicQualifier(raw, in.Spans, sources, metrics, own, idef.Terms, lo)' 'a.topicQualifier(raw, in.Spans, metrics, sources, own, idef.Terms, lo)' 'CG186 V5 verdictGap: truyền nguồn / chỉ số đổi chỗ cho topicQualifier'
m $C 'slices.Concat(sources, metrics), lo)' 'slices.Concat(sources), lo)' 'CG189 V5 verdictGap: contextOwners không nhận từ khoá chỉ số khác'
m $C 'slices.Concat(sources, metrics), lo)' 'nil, lo)' 'CG191 V5 verdictGap: contextOwners không nhận cụm nào'
m $C 'var out [][2]int
	for _, p := range phrases {' 'var out [][2]int
	for _, p := range phrases[:0] {' 'CG192 V5 contextOwners: không trả cụm nào'
m $C 'continue // tên đúng nguồn đã dùng' 'break // tên đúng nguồn đã dùng' 'CG194 V5 qualifierPhrases: gặp nguồn đã dùng thì bỏ cả các nguồn sau nó'
m $C '			if len(used) == 1 && used[0] == src.Type {' '			if len(used) == 1 || used[0] == src.Type {' 'CG196 V5 qualifierPhrases: một nguồn đã dùng thì mọi nguồn đều được bỏ'
m $C 'MetricIDs() {
		m, _ := a.RT.Metrics.Metric(id)
		for _, src := range m.Sources {' 'MetricIDs()[:1] {
		m, _ := a.RT.Metrics.Metric(id)
		for _, src := range m.Sources {' 'CG197 V5 qualifierPhrases: chỉ xét chỉ số đầu tiên của pack'
m $C 'sources = append(sources, p)' 'metrics = append(metrics, p)' 'CG199 V5 qualifierPhrases: tên nguồn khác xếp vào danh sách chỉ số'
m $C 'return sources, metrics' 'return metrics, sources' 'CG200 V5 qualifierPhrases: trả hai danh sách đổi chỗ'
m $C '		return gap
	}
	extra := contextOwners' '		return "value"
	}
	extra := contextOwners' 'CG202 V5 verdictGap: topicQualifier trả nhãn sai (value)'
m $W '(last == vwHalf && prev == vwDigit && scaled(prev2))' '(false && last == vwHalf && prev == vwDigit && scaled(prev2))' 'NW12 V5 số bằng chữ mơ hồ: bỏ nhánh "rưỡi" sau "<hàng> <chữ số>"'
m $W 'scaled(prev2)' '(scaled(prev2) || true)' 'NW14 V5 số bằng chữ mơ hồ: "hai rưỡi" (không hàng trước chữ số) mơ hồ'
m $W 'scaled(prev2)' 'prev2 == vwScale' 'NW15 V5 số bằng chữ mơ hồ: nhánh "rưỡi" bỏ "trăm <chữ số> rưỡi"'
m $W 'scaled(prev2)' 'prev2 == vwHundred' 'NW16 V5 số bằng chữ mơ hồ: nhánh "rưỡi" bỏ "<nghìn> <chữ số> rưỡi"'
m $W 'prev2, prev, last = prev, last, w.kind' 'prev2, prev, last = prev2, last, w.kind' 'NW17 V5 số bằng chữ mơ hồ: không nhớ từ thứ ba tính từ cuối (prev2)'
m $W 'prev2, prev, last = 0, 0, 0' 'prev2, prev, last = 0, 0, last' 'NW20 V5 số bằng chữ mơ hồ: đứt dãy không xoá last'
# ---- Vòng 5, lần 2, nhóm THIẾU TEST: các dòng dưới SỐNG ở 95dea0f (chưa test nào đỏ) và phải BỊ BẮT sau khi thêm các ca đề xuất của báo cáo v5db2 (đã chạy: ca đề xuất xanh trên 95dea0f, đỏ khi áp từng đột biến).
# Kỳ vọng suy từ luật, không chép từ đầu ra. IK1 / CG72 BỊ BẮT ở 03c2c54 rồi mất bảo vệ khi 95dea0f chuyển đề mục thuần sang topicQualifier. CG168 là chỗ lộ lỗ hổng thật: ca "Ngày 19/10, ERP, EX-03 …" đỏ ở 95dea0f (xem báo cáo). ----
m $I '		if contracts.RouteKind(d.Route) == contracts.RouteSQLQuery && !d.Weak {' '		if true {' 'IK1 từ khoá của ý định không phải số liệu thành ngữ cảnh hợp lệ'
m $C '&& sp.Start >= s && sp.Start < e && !inName(sp.Start, sp.End)' '&& sp.Start < e && !inName(sp.Start, sp.End)' 'CG72 V5 numberIn: bỏ cận dưới (số đứng trước cụm cũng xen giữa)'
m $C 'if !src && !met {' 'if false && !src && !met {' 'CG165 V5 topicQualifier: đoạn không có cụm nào vẫn là đề mục'
m $C 'markPhrases(a.RT.ClaimNeutral)' 'markPhrases(nil)' 'CG166 V5 topicQualifier: từ trung tính ("ơi", "vậy") không được tha trong đề mục'
m $C 'if (sp.Kind == contracts.ValueDate || sp.Kind == contracts.ValuePeriod) && sp.Start >= from && sp.End <= lo {' 'if sp.Kind == contracts.ValueDate && sp.Start >= from && sp.End <= lo {' 'CG167 V5 topicQualifier: mốc kiểu kỳ ("hôm nay") không được tha trong đề mục'
m $C 'if (sp.Kind == contracts.ValueDate || sp.Kind == contracts.ValuePeriod) && sp.Start >= from && sp.End <= lo {' 'if sp.Kind == contracts.ValuePeriod && sp.Start >= from && sp.End <= lo {' 'CG168 V5 topicQualifier: mốc kiểu ngày ("ngày 19/10") không được tha trong đề mục'
m $C 'if (sp.Kind == contracts.ValueDate || sp.Kind == contracts.ValuePeriod) && sp.Start >= from && sp.End <= lo {' 'if false && (sp.Kind == contracts.ValueDate || sp.Kind == contracts.ValuePeriod) && sp.Start >= from && sp.End <= lo {' 'CG169 V5 topicQualifier: mốc nào cũng không được tha trong đề mục'
m $C '			if ws, we, ok := normalization.PeriodWordBefore(raw, sp.Start); ok && ws >= from {' '			if ws, we, ok := normalization.PeriodWordBefore(raw, sp.Start); false && ok && ws >= from {' 'CG203 V5 topicQualifier: chữ "ngày" / "hôm" trước mốc số không được tha trong đề mục'
# CG204 (tương đương) — bỏ cận ws >= from: từ chỉ kỳ dính liền mốc số nên không thể nằm sau dấu ngắt câu mở đầu đoạn [from, lo)
m $C '	markPhrases(own)' '	markPhrases(nil)' 'CG205 V5 topicQualifier: chữ của chính claim (từ khoá chính ý định, đúng nguồn đã dùng) phá đề mục thuần'
m $C '		if sp.Kind == contracts.ValueTerm && slices.Contains(ownTerms, sp.Value.Text) && sp.Start >= from && sp.End <= lo {' '		if false && sp.Kind == contracts.ValueTerm && slices.Contains(ownTerms, sp.Value.Text) && sp.Start >= from && sp.End <= lo {' 'CG206 V5 topicQualifier: thuật ngữ của chính ý định ("dầu") phá đề mục thuần'
m $C '			own = append(append(own, src.Label), src.Aliases...)' '			own = append(own)' 'CG208 V5 verdictGap: tên đúng nguồn đã dùng không được tha trong đề mục'
m $C '				if keepSrc(p) {' '				if keep(p) {' 'CG209 V5 qualifierPhrases: tên nguồn trùng từ khoá chính ý định ("phiếu kho") bị bỏ khỏi nguồn khác (R5-5)'
m $C '	own := slices.Clone(idef.Keywords)' '	own := []string{}' 'CG210 V5 verdictGap: từ khoá của chính ý định không được tha trong đề mục'
m $C 'extra := contextOwners(raw, in.Spans, inName, slices.Concat(sources, metrics), lo)' 'extra := contextOwners(raw, in.Spans, inName, slices.DeleteFunc(slices.Concat(sources, metrics), func(p string) bool { return slices.Contains(idef.Keywords, p) }), lo)' 'CG211 V5 R5-5 chủ ở ngữ cảnh bỏ cụm trùng từ khoá ý định ("phiếu kho" ở ngữ cảnh sau tên chủ thể, soát vòng 4 M13)'
m $C '		i += n
	}
	if src {' '		i += 1 + 0*n
	}
	if src {' 'CG176 V5 topicQualifier: bước theo byte thay vì theo rune (dấu ngăn nhiều byte “ ” …)'
m $C 'return "source"
	}
	return "metric"' 'return "source"
	}
	return "source"' 'CG180 V5 topicQualifier: đề mục chỉ có chỉ số cũng mang nhãn source'
m $C 'a.topicQualifier(raw, in.Spans, sources, metrics, own, idef.Terms, lo)' 'a.topicQualifier(raw, nil, sources, metrics, own, idef.Terms, lo)' 'CG188 V5 verdictGap: topicQualifier không biết mốc nào'
m $C 'slices.Concat(sources, metrics), lo)' 'slices.Concat(metrics), lo)' 'CG190 V5 verdictGap: contextOwners không nhận tên nguồn khác'
m $C '			if len(used) == 1 && used[0] == src.Type {' '			if len(used) >= 1 && used[0] == src.Type {' 'CG195 V5 qualifierPhrases: nhiều nguồn đã dùng thì nguồn đầu vẫn được bỏ'
m $C 'metrics = append(metrics, p)' 'sources = append(sources, p)' 'CG198 V5 qualifierPhrases: từ khoá chỉ số khác xếp vào danh sách nguồn'
m $W 'last == vwHalf && prev == vwDigit && scaled(prev2)' 'last == vwHalf && scaled(prev2)' 'NW13 V5 số bằng chữ mơ hồ: "rưỡi" sau "<hàng> <từ khác chữ số> " cũng mơ hồ'
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
# CG153 / CG154 — from == lo: không cụm không rỗng nào thoả sp[0] >= from && sp[1] <= lo nên markPhrases không trúng và hàm trả "" như nhánh thoát sớm (from <= lo luôn, vì sentenceStart(raw, lo) <= lo)
# m $C 'if from >= lo {' 'if from > lo {' 'CG153 V5 topicQualifier: from == lo vẫn chạy tiếp (> thay >=)'
# m $C 'if from >= lo {' 'if false {' 'CG154 V5 topicQualifier: bỏ thoát sớm khi không có đoạn đầu câu'
# CG160 / CG172 — ký tự ngay trước lo là dấu ngắt mệnh đề (FocusClause đặt lo ngay sau , ; . ? !) nên không cụm hay mốc nào kết thúc đúng tại lo
# m $C 'if sp[0] >= from && sp[1] <= lo {' 'if sp[0] >= from && sp[1] < lo {' 'CG160 V5 topicQualifier: cụm kết thúc đúng tại lo không được đánh dấu (< thay <=)'
# m $C '&& sp.Start >= from && sp.End <= lo {' '&& sp.Start >= from && sp.End < lo {' 'CG172 V5 topicQualifier: mốc kết thúc đúng tại lo không được đánh dấu (< thay <=)'
# CG183 — mark đánh dấu thừa ký tự liền sau cụm: luôn là dấu ngắt hoặc khoảng trắng, vốn đã được tha
# m $C 'for i := s; i < e; i++ {' 'for i := s; i <= e; i++ {' 'CG183 V5 topicQualifier: mark đánh dấu thừa một ký tự (<= thay <)'
# CG187 — topicQualifier lấy qs thay lo: khi mệnh đề hỏi không có số, lo lùi về đầu câu và đoạn [đầu câu, qs) chứa chính con số xác nhận nên không bao giờ là đề mục thuần; khi có số thì lo = qs
# m $C 'a.topicQualifier(raw, in.Spans, sources, metrics, lo)' 'a.topicQualifier(raw, in.Spans, sources, metrics, qs)' 'CG187 V5 verdictGap: topicQualifier lấy đầu mệnh đề hỏi (qs) thay vì đầu phạm vi hỏi (lo)'
# CG193 — cụm rỗng: PhraseSpans(raw, "") không bao giờ khớp (matchAliasPrefix trả false khi alias == "", text.go:259) nên `p != ""` ở qualifierPhrases thừa với mọi pack, không chỉ pack hiện có (thay CG59 của 03c2c54)
# m $C 'return p != "" && ' 'return ' 'CG193 V5 qualifierPhrases: cụm rỗng không bị bỏ qua'
# NW18 / NW19 — sau lần đặt lại, prev2 / prev cũ chỉ còn ở vị trí tail() không đọc (last = 0) rồi bị đẩy ra sau một từ số
# m $W 'prev2, prev, last = 0, 0, 0' 'prev2, prev, last = prev2, 0, 0' 'NW18 V5 số bằng chữ mơ hồ: đứt dãy không xoá prev2'
# m $W 'prev2, prev, last = 0, 0, 0' 'prev2, prev, last = 0, prev, 0' 'NW19 V5 số bằng chữ mơ hồ: đứt dãy không xoá prev'
# (CG59 / CG62 của 03c2c54 — cụm rỗng, bí danh của tên nguồn — nay là CG193 / CG201 vì qualifierPhrases tách khỏi contextOwners.)
