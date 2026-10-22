#!/usr/bin/env bash
# Danh mục đột biến cho bộ đọc số bằng chữ (D-OP-14, data/normalization/numwords_vi.go): chữ số đứng ngay sau từ chỉ hàng là hàng kế dưới
# ("một nghìn hai" = 1.200, "ba nghìn tám rưỡi" = 3.850) và "muoi" không dấu đọc theo ngữ cảnh, tất định. Mỗi dòng: m <tệp> <chuỗi gốc>
# <chuỗi đột biến> <nhãn>. Kỳ vọng: mọi dòng "BỊ BẮT".
cd "$(dirname "$0")/../.." || exit 1
m() { scripts/mutate.sh "$1" "$2" "$3" "$4"; echo "rc=$?"; }
export PKGS="./data/normalization/ ./core/pipeline/ ./tenants/construction/"
W=data/normalization/numwords_vi.go
m $W 'implied = lastUnit / 10 //' 'implied = 1 //' 'ND1 chữ số sau từ chỉ hàng vẫn là hàng đơn vị'
m $W 'if prev == vwHundred || prev == vwScale {' 'if prev == vwScale {' 'ND2 chữ số sau "trăm" vẫn là hàng đơn vị'
m $W 'if prev == vwHundred || prev == vwScale {' 'if prev == vwHundred {' 'ND3 chữ số sau "nghìn" / "triệu" / "tỷ" vẫn là hàng đơn vị'
m $W '		group += pending * implied
		colloquial = colloquial || implied > 1' '		group += pending
		colloquial = colloquial || implied > 1' 'ND4 cuối dãy không nhân đơn vị hàng kế dưới'
m $W 'case pending >= 0 && implied > 1:' 'case false && pending >= 0 && implied > 1:' 'ND5 "rưỡi" sau "<hàng> <chữ số>" là nửa đơn vị'
m $W 'group += pending*implied + implied/2' 'group += pending * implied' 'ND6 "rưỡi" sau "<hàng> <chữ số>" bị bỏ'
m $W '			pending = w.value
			implied = 1' '			pending = w.value' 'ND7 đơn vị hàng kế dưới không đặt lại ở chữ số mới'
m $W '		prev = w.kind
		consumed++' '		consumed++' 'ND8 không nhớ loại từ vừa đọc'
m $W '			if pending >= 0 {
				w.kind = vwTens' '			if false {
				w.kind = vwTens' 'ND9 "muoi" sau chữ số không là "mươi"'
m $W '			w.kind = vwMuoi
			if pending >= 0 {' '			w.kind = vwTens
			if pending >= 0 {' 'ND10 "muoi" đứng đầu không là "mười"'
m $W 'case len(vs) == 2 && slices.Contains(vs, viNumWord{vwMuoi, 10}) && slices.Contains(vs, viNumWord{vwTens, 10}):' 'case false:' 'ND11 bảng không dấu bỏ "muoi"'
m $W 'case len(vs) == 1:' 'case false:' 'ND12 bảng không dấu bỏ mọi bản gấp'
m $W 'if _, exists := base[f]; !exists && !slices.Contains(folded[f], v) {' 'if _, exists := base[f]; !exists {' 'ND13 hai từ cùng nghĩa cùng gấp ("một" / "mốt") bị coi là xung đột'
# ND14 — `f != w` bỏ: từ không dấu gấp ra chính nó, đã có trong base nên nhánh exists loại — tương đương với mọi bảng
# m $W 'if f := Fold(w); f != w {' 'if f := Fold(w); true {' 'ND14 từ không dấu cũng vào bảng gấp'
# Soát khoá vòng 3 (S2-2): chỉ từ chỉ hàng nghìn trở lên làm số bằng chữ "tự mang chiều đo".
m $W 'if w, ok := lookupNumberWord(t.text); ok && t.kind == tokWord && w.kind == vwScale {' 'if w, ok := lookupNumberWord(t.text); ok && t.kind == tokWord && (w.kind == vwScale || w.kind == vwHundred) {' 'ND15 "trăm" tính như hàng nghìn ("ba trăm" thành lượng)'
m $W 'if w, ok := lookupNumberWord(t.text); ok && t.kind == tokWord && w.kind == vwScale {' 'if _, ok := lookupNumberWord(t.text); ok && t.kind == tokWord {' 'ND16 mọi chữ số đọc được đều tính như hàng nghìn ("mười chín" thành lượng)'
m $W 'if _, ok := lookupNumberWord(t.text); ok && t.kind == tokWord {' 'if _, ok := lookupNumberWord(t.text); ok && t.kind == tokWord && false {' 'ND17 hậu tố "trăm" / "chục" thành hậu tố tiếng lóng ("3 trăm" thành lượng)'
