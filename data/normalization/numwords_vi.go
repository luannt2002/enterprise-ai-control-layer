package normalization

import (
	"math/big"
	"slices"
	"strings"
)

// Đọc số viết bằng chữ tiếng Việt: "bốn ngàn", "ba trăm năm chục", "hai triệu rưỡi",
// "một trăm linh năm", "hai mươi mốt".
//
// Đây là tri thức NGÔN NGỮ (dùng chung mọi tenant tiếng Việt), không phải tri thức của
// một ngành, nên nằm trong mã chứ không trong pack.

type viWord int

const (
	vwDigit      viWord = iota + 1
	vwMuoi              // "mười" — số 10, đứng đầu
	vwTens              // "mươi", "chục" — nhân 10 cho chữ số đứng trước
	vwHundred           // "trăm"
	vwScale             // "nghìn", "ngàn", "triệu", "tỷ"
	vwFiller            // "linh", "lẻ" — số 0 ở hàng chục
	vwHalf              // "rưỡi" — một nửa đơn vị vừa đọc
	vwMuoiOrTens        // "muoi" (gõ không dấu): "mươi" khi có chữ số đứng trước, "mười" khi đứng đầu
)

type viNumWord struct {
	kind  viWord
	value int64
}

// viNumberWords có cả bản có dấu lẫn bản đã bỏ dấu (người gõ không dấu).
var viNumberWords = func() map[string]viNumWord {
	base := map[string]viNumWord{
		"không": {vwDigit, 0}, "một": {vwDigit, 1}, "mốt": {vwDigit, 1}, "hai": {vwDigit, 2},
		"ba": {vwDigit, 3}, "bốn": {vwDigit, 4}, "tư": {vwDigit, 4}, "năm": {vwDigit, 5},
		"lăm": {vwDigit, 5}, "nhăm": {vwDigit, 5}, "sáu": {vwDigit, 6}, "bảy": {vwDigit, 7},
		"bẩy": {vwDigit, 7}, "tám": {vwDigit, 8}, "chín": {vwDigit, 9},
		"mười": {vwMuoi, 10}, "mươi": {vwTens, 10}, "chục": {vwTens, 10},
		"trăm": {vwHundred, 100}, "nghìn": {vwScale, 1000}, "ngàn": {vwScale, 1000},
		"triệu": {vwScale, 1000000}, "tỷ": {vwScale, 1000000000}, "tỉ": {vwScale, 1000000000},
		"linh": {vwFiller, 0}, "lẻ": {vwFiller, 0}, "rưỡi": {vwHalf, 0},
	}
	m := map[string]viNumWord{}
	folded := map[string][]viNumWord{}
	for w, v := range base {
		m[w] = v
		// Bản bỏ dấu chỉ thêm khi không đè một từ có dấu khác nghĩa.
		if f := Fold(w); f != w {
			if _, exists := base[f]; !exists && !slices.Contains(folded[f], v) {
				folded[f] = append(folded[f], v)
			}
		}
	}
	// Quyết bản bỏ dấu SAU vòng duyệt để không phụ thuộc thứ tự duyệt map (D-OP-14): "mười" và "mươi" cùng gấp thành "muoi" thì để
	// bộ đọc quyết theo ngữ cảnh; hai từ khác nghĩa khác cùng gấp về một chữ thì bỏ bản bỏ dấu (không đoán).
	for f, vs := range folded {
		switch {
		case len(vs) == 1:
			m[f] = vs[0]
		case len(vs) == 2 && slices.Contains(vs, viNumWord{vwMuoi, 10}) && slices.Contains(vs, viNumWord{vwTens, 10}):
			m[f] = viNumWord{vwMuoiOrTens, 10}
		}
	}
	return m
}()

// lookupNumberWord tra một từ: có dấu thì tra đúng dấu; không dấu thì tra bản bỏ dấu.
func lookupNumberWord(word string) (viNumWord, bool) {
	w := strings.ToLower(word)
	if v, ok := viNumberWords[w]; ok {
		return v, true
	}
	if !hasDiacritic(w) {
		if v, ok := viNumberWords[Fold(w)]; ok {
			return v, true
		}
	}
	return viNumWord{}, false
}

// AmbiguousNumberWords báo s có dãy số bằng chữ mà giá trị do CÁCH NÓI TẮT quyết, không do các từ viết ra (readViNumberWords trả
// colloquial): chữ số đứng ngay sau từ chỉ hàng ở cuối dãy là hàng kế dưới ("một nghìn hai" = 1.200, "hai trăm năm" = 250, "ba nghìn tám
// rưỡi" = 3.850; trước D-OP-14 bộ đọc cộng như hàng đơn vị: 1.002, 205, 3.008,5), hay dãy kết thúc bằng "linh" / "lẻ" ("một nghìn hai lẻ",
// "hai trăm lẻ" = hơn một chút: xấp xỉ). Dạng viết rõ hàng ("một nghìn hai trăm", "một trăm linh năm", "hai nghìn rưỡi", "ba nghìn tám
// trăm rưỡi") không tính. Cổng phán quyết (khoá ở ADR-0009) dùng hàm này để không phán quyết các dạng đó, kể cả sau khi bộ đọc đã sửa.
func AmbiguousNumberWords(s string) bool {
	toks := tokenize(s)
	for i := 0; i < len(toks); i++ {
		if toks[i].kind != tokWord {
			continue
		}
		if _, isNum := lookupNumberWord(toks[i].text); !isNum {
			continue
		}
		_, next, _, ok, colloquial := readViNumberWords(toks, i)
		if ok && colloquial {
			return true
		}
		if next > i {
			i = next - 1
		}
	}
	return false
}

// HasThousandsWord báo s có từ chỉ hàng nghìn trở lên ("bốn ngàn", "hai triệu", "3 nghìn"): hàng lớn cho biết con số được nói theo hàng,
// là một lượng như "4k". Số nhỏ đọc thành chữ ("mười chín", "hai chục", "bốn mươi ba") không có hàng đó — như số trần "19", nó có thể là
// ngày, số hiệu. Cổng phán quyết (khoá ở ADR-0009) dùng hàm này: số không đơn vị chỉ "tự mang chiều đo" khi có hàng lớn hay hậu tố nhân.
func HasThousandsWord(s string) bool {
	for _, t := range tokenize(s) {
		if w, ok := lookupNumberWord(t.text); ok && t.kind == tokWord && w.kind == vwScale {
			return true
		}
	}
	return false
}

// HasNumberWord báo s có từ chỉ số ("trăm", "chục", "mười", "nghìn"…). Hậu tố nhân tiếng lóng của pack ("k", "tr", "củ") không phải từ
// chỉ số: "3k" không có, "3 trăm" có.
func HasNumberWord(s string) bool {
	for _, t := range tokenize(s) {
		if _, ok := lookupNumberWord(t.text); ok && t.kind == tokWord {
			return true
		}
	}
	return false
}

// parseViNumberWords đọc dãy từ chỉ số bắt đầu từ token i.
//
// Trả về giá trị, chỉ số token kế tiếp, và hasScale = dãy có từ chỉ hàng (mươi, trăm,
// nghìn, rưỡi…). Bên gọi chỉ chấp nhận dãy KHÔNG có hàng khi ngay sau là một đơn vị:
// "năm" một mình còn là "năm (year)", "năm 2026" không phải số 5.
func parseViNumberWords(toks []token, i int) (value *big.Rat, next int, hasScale bool, ok bool) {
	value, next, hasScale, ok, _ = readViNumberWords(toks, i)
	return value, next, hasScale, ok
}

// readViNumberWords là parseViNumberWords kèm colloquial: giá trị do cách nói tắt quyết — chữ số cuối dãy đứng ngay sau từ chỉ hàng được
// hiểu là hàng kế dưới, hay dãy kết thúc bằng "linh" / "lẻ" (xem AmbiguousNumberWords).
func readViNumberWords(toks []token, i int) (value *big.Rat, next int, hasScale, ok, colloquial bool) {
	total := new(big.Rat)
	var group int64
	pending := int64(-1) // chữ số đang chờ hàng; -1 = không có
	implied := int64(1)  // đơn vị của chữ số đang chờ khi nó đứng NGAY sau từ chỉ hàng: "một nghìn hai" = 1.200 (D-OP-14)
	lastUnit := int64(1)
	var prev viWord // loại của từ chỉ số vừa đọc
	consumed := 0
	j := i
	last := i

	for j < len(toks) {
		t := toks[j]
		if t.kind == tokSpace {
			// chỉ đi qua đúng một khoảng trắng giữa hai từ chỉ số
			if j+1 < len(toks) && toks[j+1].kind == tokWord {
				if _, isNum := lookupNumberWord(toks[j+1].text); isNum {
					j++
					continue
				}
			}
			break
		}
		if t.kind != tokWord {
			break
		}
		w, isNum := lookupNumberWord(t.text)
		if !isNum {
			break
		}
		if w.kind == vwMuoiOrTens {
			w.kind = vwMuoi
			if pending >= 0 {
				w.kind = vwTens // "hai muoi" = hai mươi
			}
		}
		stop := false
		switch w.kind {
		case vwDigit:
			if pending >= 0 {
				stop = true // "hai ba" — hai chữ số liền nhau không thành một số
				break
			}
			pending = w.value
			implied = 1
			if prev == vwHundred || prev == vwScale {
				implied = lastUnit / 10 // hàng kế dưới: "hai trăm năm" = 250, "ba nghìn tám" = 3.800
			}
			lastUnit = 1
		case vwMuoi:
			if pending >= 0 {
				stop = true
				break
			}
			group += 10
			lastUnit = 10
			hasScale = true
		case vwTens:
			if pending < 0 {
				stop = true // "mươi" không có chữ số đứng trước
				break
			}
			group += pending * 10
			pending = -1
			lastUnit = 10
			hasScale = true
		case vwHundred:
			if group > 0 {
				colloquial = true // "mười hai trăm" (= 1.200 kiểu nói tắt): bộ đọc cộng dồn ra 210, không đọc chắc
			}
			if pending < 0 {
				pending = 1 // "trăm" một mình = một trăm
			}
			group += pending * 100
			pending = -1
			lastUnit = 100
			hasScale = true
		case vwScale:
			g := group
			if pending >= 0 {
				g += pending
			}
			if g == 0 {
				g = 1 // "nghìn rưỡi", "triệu hai"
			}
			total.Add(total, new(big.Rat).SetInt64(g*w.value))
			group, pending = 0, -1
			lastUnit = w.value
			hasScale = true
		case vwFiller:
			// "linh/lẻ" chỉ có nghĩa khi theo sau là một chữ số
		case vwHalf:
			switch {
			case lastUnit >= 1000:
				total.Add(total, new(big.Rat).SetInt64(lastUnit/2))
			case lastUnit == 100 || lastUnit == 10:
				group += lastUnit / 2
			case pending >= 0 && implied > 1:
				group += pending*implied + implied/2 // "ba nghìn tám rưỡi" = 3.850, "một trăm hai rưỡi" = 125
				pending = -1
				colloquial = true
			case pending >= 0:
				group += pending
				pending = -1
				total.Add(total, big.NewRat(1, 2))
			default:
				stop = true
			}
			if !stop {
				hasScale = true
				consumed++
				last = j
				j++
				// "rưỡi" luôn là từ cuối của một số
				goto done
			}
		}
		if stop {
			break
		}
		prev = w.kind
		consumed++
		last = j
		j++
	}
done:
	if consumed == 0 {
		return nil, i, false, false, false
	}
	if pending >= 0 {
		group += pending * implied
		colloquial = colloquial || implied > 1
	}
	if toks[last].kind == tokWord {
		if w, _ := lookupNumberWord(toks[last].text); w.kind == vwFiller {
			colloquial = true // "linh" / "lẻ" đứng cuối dãy: "hơn một chút", không phải số đúng
		}
	}
	total.Add(total, new(big.Rat).SetInt64(group))
	return total, last + 1, hasScale, true, colloquial
}
