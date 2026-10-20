#!/usr/bin/env bash
# Danh mục đột biến cho các sửa của người soát độc lập ở luồng hỏi ngày 20/10/2026: S2-1 (thay biến mẫu câu một
# lượt, tất định), S2-2 (lượng nghiệp vụ trong tài liệu lọt cả hai lớp kiểm), S2-3 (ignore cắt chuỗi con làm hỏng
# số có thẩm quyền) và M42 (câu xác nhận với số hỏi nhỏ hơn số thật).
# Mỗi dòng: m <tệp> <chuỗi gốc> <chuỗi đột biến> <nhãn>. Kỳ vọng: mọi dòng in "BỊ BẮT" (rc=0).
# Không đưa vào danh mục (đột biến tương đương, đã xét): bỏ dòng `ignore = append(ignore, vars["claimed"])` ở
# ask.go — số của người hỏi đã được bằng chứng "question" đỡ nên Fidelity vẫn qua; chuỗi ignore đó chỉ còn là
# dây an toàn thừa, và lỗi S2-3 nằm ở cách Fidelity cắt chuỗi (maskWhole), không ở chỗ nối ignore.
# Mã đổi thì chuỗi gốc có thể không còn khớp (rc=2): sửa chuỗi theo mã mới — đừng xoá dòng cho xanh.
cd "$(dirname "$0")/../.." || exit 1
export PKGS="./core/pipeline/ ./reliability/validation/ ./data/normalization/ ./tenants/construction/"
A=core/pipeline/ask.go N=reliability/validation/numeric.go K=reliability/validation/knowledge.go E=data/normalization/engine.go
m() { scripts/mutate.sh "$1" "$2" "$3" "$4"; echo "rc=$?"; }

# --- S2-1: thay biến mẫu câu MỘT lượt, giá trị biến không bị quét lại (ask.go fragment)
m $A '	return strings.NewReplacer(pairs...).Replace(a.RT.Templates[name])' '	r := strings.NewReplacer(pairs...)
	return r.Replace(r.Replace(a.RT.Templates[name]))' 'S2-1a giá trị biến bị quét lại (thay hai lượt)'
m $A '	return strings.NewReplacer(pairs...).Replace(a.RT.Templates[name])' '	text := a.RT.Templates[name]
	for i := 0; i < len(pairs); i += 2 {
		text = strings.ReplaceAll(text, pairs[i], pairs[i+1])
	}
	return text' 'S2-1b thay tuần tự theo thứ tự duyệt map (hành vi cũ)'
m $A '		if _, set := vars[p]; !set { // biến được phép mà chưa có giá trị thì thành chuỗi rỗng' '		if _, set := vars[p]; set { // biến được phép mà chưa có giá trị thì thành chuỗi rỗng' 'S2-1c biến được phép mà vắng không thành rỗng'

# --- M42: câu xác nhận so BẰNG TUYỆT ĐỐI (số hỏi nhỏ hơn số thật phải ra "Không.")
m $A 'err == nil && c == 0 {' 'err == nil && c <= 0 {' 'M42a so bằng thành nhỏ hơn hoặc bằng'
m $A 'err == nil && c == 0 {' 'err == nil && c >= 0 {' 'M42b so bằng thành lớn hơn hoặc bằng'

# --- S2-3: ignore chỉ bỏ lần xuất hiện TRỌN VẸN (numeric.go maskWhole)
m $N '		text = maskWhole(text, s)' '		if s != "" {
			text = strings.ReplaceAll(text, s, " ")
		}' 'S2-3a ignore cắt chuỗi con (hành vi cũ)'
m $N '		if end := j + len(s); !joinedBefore(text[:j], s) && !joinedAfter(text[end:], s) {' '		if end := j + len(s); !joinedBefore(text[:j], s) {' 'S2-3b bỏ kiểm dính phía sau'
m $N '		if end := j + len(s); !joinedBefore(text[:j], s) && !joinedAfter(text[end:], s) {' '		if end := j + len(s); !joinedAfter(text[end:], s) {' 'S2-3c bỏ kiểm dính phía trước'
m $N '	if isWordRune(first) && isWordRune(prev) {' '	if false && isWordRune(first) && isWordRune(prev) {' 'S2-3d bỏ luật dính chữ/số phía trước'
m $N '	if prev == '"'"'.'"'"' || prev == '"'"','"'"' {' '	if false {' 'S2-3e bỏ luật dấu nghìn / thập phân phía trước'
m $N '	if prev == '"'"'-'"'"' || prev == '"'"'−'"'"' {' '	if false {' 'S2-3f bỏ luật dấu âm / gạch nối phía trước'
m $N '	if isWordRune(last) && isWordRune(next) {' '	if false && isWordRune(last) && isWordRune(next) {' 'S2-3g bỏ luật dính chữ/số phía sau'
m $N '	if unicode.IsDigit(last) && (next == '"'"'.'"'"' || next == '"'"','"'"') {' '	if false {' 'S2-3h bỏ luật dấu nghìn / thập phân phía sau'

# --- S2-2, lớp chuẩn hoá (engine.go): gấp ký tự lạ, khoảng trắng, đơn vị trong ngoặc / đứng trước
m $E '	raw := NFC(foldLookalikes(text))' '	raw := NFC(text)' 'S2-2a Normalize không gấp ký tự lạ'
m $E '			return r - 0xFEE0' '			return r' 'S2-2b không gấp chữ số toàn chiều rộng'
m $E '		case r != '"'"'\n'"'"' && r != '"'"'\r'"'"' && unicode.IsSpace(r):' '		case r == '"'"' '"'"':' 'S2-2c không gấp khoảng trắng lạ (NBSP, U+202F, tab…)'
m $E '		case r == 0x2113:' '		case false && r == 0x2113:' 'S2-2d không gấp ký hiệu ℓ'
m $E 'func isFormat(r rune) bool { return unicode.Is(unicode.Cf, r) }' 'func isFormat(r rune) bool { return false }' 'S2-2e1 ký tự vô hình (ZWSP, ZWJ…) không còn là khoảng đệm'
m $E '		if r != '"'"' '"'"' && !isFormat(r) {' '		if r != '"'"' '"'"' {' 'S2-2e2 skipGap chỉ bỏ qua dấu cách'
m $E '	if !strings.ContainsFunc(s, isFormat) {' '	if true {' 'S2-2e3 ký tự vô hình giữa hai chữ số tách một số thành hai (không ghép lại)'
m $E $'\tp := skipGap(s, pos)\n\tif p > pos {\n\t\tstarts = append(starts, p)' $'\tp := skipGap(s, pos)\n\tif p > pos {\n\t\tstarts = append(starts, min(p, pos+1))' 'S2-2f chỉ cho đúng một dấu cách giữa số và đơn vị'
m $E '	if layout && p < len(s) && strings.IndexByte("([{", s[p]) >= 0 {' '	if false && layout && p < len(s) && strings.IndexByte("([{", s[p]) >= 0 {' 'S2-2g bỏ đơn vị trong ngoặc'
m $E 'if p == bracketed && !bracketTail(s, uEnd) {' 'if p == bracketed && false {' 'S2-2h ngoặc đọc nhầm "(khối lượng …)" là lượng'
m $E '		if opt.Layout && t.kind == tokWord {' '		if false && opt.Layout && t.kind == tokWord {' 'S2-2i bỏ đơn vị đứng trước số'
m $E '		if !ok || num.Kind != contracts.ValueNumber {' '		if !ok {' 'S2-2j đơn vị đứng trước nuốt số đã có đơn vị riêng'
m $E '	approxUntil := -1' '	approxUntil := -1
	opt.Layout = true' 'S2-2k Layout bật cho mọi câu (cả câu hỏi người dùng)'
m $K 'normalization.Options{Now: now, Layout: true}' 'normalization.Options{Now: now}' 'S2-2l ForbidDimensions không bật Layout'

# --- S2-2, lớp bộ đếm độc lập (knowledge.go ForbidUnitWords): tự gấp ký tự lạ, ngoặc, đơn vị đứng trước
m $K '	text = foldLookalikes(text)' '	_ = foldLookalikes' 'S2-2m bộ đếm không gấp ký tự lạ'
m $K '			w, ok = unitBefore(masked[:m[2]], num, words)' '			w, ok = "", false' 'S2-2n bộ đếm bỏ đơn vị đứng trước số'
m $K '	trimmed := strings.TrimRightFunc(before, func(r rune) bool { return isGap(r) || strings.ContainsRune(":=)]}", r) })' '	trimmed := strings.TrimRightFunc(before, isGap)' 'S2-2o bộ đếm không nối qua dấu hai chấm / ngoặc đóng'
m $K 'func isGap(r rune) bool { return r == '"'"' '"'"' || unicode.Is(unicode.Cf, r) }' 'func isGap(r rune) bool { return r == '"'"' '"'"' }' 'S2-2o2 bộ đếm không coi ký tự vô hình là khoảng đệm'
m $K "	if r, w := utf8.DecodeRuneInString(rest); r == '(' || r == '[' || r == '{' {" "	if r, w := utf8.DecodeRuneInString(rest); false && (r == '(' || r == '[' || r == '{') && w > 0 {" 'S2-2p bộ đếm bỏ đơn vị trong ngoặc'
m $K '			if !bracketTail(after) {' '			if false {' 'S2-2q bộ đếm đọc nhầm "(khối lượng …)" là lượng'
m $K '		if w == 0 || unicode.ToLower(r) != unicode.ToLower(pr) {' '		if w == 0 || r != pr {' 'S2-2r bộ đếm phân biệt hoa thường'
m $K '		} else if r, _ := utf8.DecodeRuneInString(after); after != "" && isWordRune(r) {' '		} else if false {' 'S2-2s bộ đếm bỏ ranh giới từ phía sau đơn vị'
m $K $'head != "" && (isWordRune(r) || r == \'/\')' $'head != "" && (r == \'/\')' 'S2-2t bộ đếm bỏ ranh giới từ phía trước đơn vị đứng trước số'
m $K '	return unicodenorm.NFC.String(strings.Map(func(r rune) rune {' '	return unicodenorm.NFD.String(strings.Map(func(r rune) rune {' 'S2-2u bộ đếm gấp NFD thay vì NFC (dấu tiếng Việt viết rời)'
