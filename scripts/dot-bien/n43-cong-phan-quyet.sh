#!/usr/bin/env bash
# Danh mục đột biến cho cổng phán quyết của câu xác nhận (ADR-0009): core/pipeline/claim.go và chỗ nối ở answerMetric. Mỗi dòng: m <tệp> <chuỗi gốc> <chuỗi đột biến> <nhãn>. Kỳ vọng: mọi dòng "BỊ BẮT".
cd "$(dirname "$0")/../.." || exit 1
m() { scripts/mutate.sh "$1" "$2" "$3" "$4"; echo "rc=$?"; }
export PKGS="./core/pipeline/ ./tenants/construction/"
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
m $C '			if !ok || !unitShown(sp) {' '			if !ok {' 'CG8 số trần thành con số xác nhận'
m $C '			if c, err := decimal.Cmp(v.Number, claimed.Number); err != nil || c != 0 {' '			if c, err := decimal.Cmp(v.Number, claimed.Number); false && (err != nil || c != 0) {' 'CG9 con số khác con số xác nhận'
m $C 'coverPhrases(slices.Concat(idef.Keywords, topSourceLabels(def), a.RT.ConfirmMarkers, a.RT.ClaimNeutral), 0, len(raw))' 'coverPhrases(slices.Concat(topSourceLabels(def), a.RT.ConfirmMarkers, a.RT.ClaimNeutral), 0, len(raw))' 'CG10 bỏ từ khoá của ý định'
m $C 'coverPhrases(slices.Concat(idef.Keywords, topSourceLabels(def), a.RT.ConfirmMarkers, a.RT.ClaimNeutral), 0, len(raw))' 'coverPhrases(slices.Concat(idef.Keywords, a.RT.ConfirmMarkers, a.RT.ClaimNeutral), 0, len(raw))' 'CG11 bỏ nguồn có thẩm quyền'
m $C 'coverPhrases(slices.Concat(idef.Keywords, topSourceLabels(def), a.RT.ConfirmMarkers, a.RT.ClaimNeutral), 0, len(raw))' 'coverPhrases(slices.Concat(idef.Keywords, topSourceLabels(def), a.RT.ConfirmMarkers), 0, len(raw))' 'CG12 bỏ từ trung tính'
m $C '		if best == 0 || s.Authority < best {' '		if best == 0 || s.Authority > best {' 'CG13 nguồn thẩm quyền thấp nhất thành được giải thích'
m $C '	if lo > 0 {' '	if false {' 'CG14 ngữ cảnh không được giải thích'
m $C '		if !covered[i] && !unicode.IsSpace(r) && !strings.ContainsRune(claimSeparators, r) {' '		if false && !covered[i] && !unicode.IsSpace(r) && !strings.ContainsRune(claimSeparators, r) {' 'CG15 bỏ kiểm phần dư'
m $C '		if !covered[i] && !unicode.IsSpace(r) && !strings.ContainsRune(claimSeparators, r) {' '		if !covered[i] && (unicode.IsLetter(r) || unicode.IsDigit(r)) {' 'CG7 ký hiệu (~, >, /) không là phần dư'
m $C 'const claimSeparators = ",.;:?!()[]\"'"'"'“”‘’…-–—"' 'const claimSeparators = ",.;:?!()[]\"'"'"'“”‘’…-–—>~/"' 'CG18 dấu so sánh / xấp xỉ / mẫu số thành dấu ngăn cách'
m $C '			if ws, we, ok := normalization.PeriodWordBefore(raw, sp.Start); ok {' '			if ws, we, ok := normalization.PeriodWordBefore(raw, sp.Start); false && ok {' 'CG16 từ chỉ kỳ trước mốc thành phần dư'
m $C '			lo = sentenceStart(raw, fs) // mệnh đề hỏi không có số' '			lo = fs // mệnh đề hỏi không có số' 'CG17 mệnh đề hỏi không số: không lùi về đầu câu'
A=core/pipeline/ask.go
m $A '		if withheld = a.verdictGap(in, names, it, def, claimed); withheld != "" {' '		if withheld = a.verdictGap(in, names, it, def, claimed)[:0]; withheld != "" {' 'AI1 tắt cổng phán quyết'
m $A '		text += a.fragment("confirm_withheld", vars)' '		_ = vars' 'AI2 bỏ mẩu chưa kết luận'
