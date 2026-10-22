package normalization

import (
	"math/big"
	"strings"
)

// Đọc số viết bằng chữ tiếng Việt: "bốn ngàn", "ba trăm năm chục", "hai triệu rưỡi",
// "một trăm linh năm", "hai mươi mốt".
//
// Đây là tri thức NGÔN NGỮ (dùng chung mọi tenant tiếng Việt), không phải tri thức của
// một ngành, nên nằm trong mã chứ không trong pack.

type viWord int

const (
	vwDigit   viWord = iota + 1
	vwMuoi           // "mười" — số 10, đứng đầu
	vwTens           // "mươi", "chục" — nhân 10 cho chữ số đứng trước
	vwHundred        // "trăm"
	vwScale          // "nghìn", "ngàn", "triệu", "tỷ"
	vwFiller         // "linh", "lẻ" — số 0 ở hàng chục
	vwHalf           // "rưỡi" — một nửa đơn vị vừa đọc
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
	for w, v := range base {
		m[w] = v
		// Bản bỏ dấu chỉ thêm khi không đè một từ có dấu khác nghĩa.
		f := Fold(w)
		if _, exists := base[f]; !exists {
			if _, taken := m[f]; !taken {
				m[f] = v
			}
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

// AmbiguousNumberWords báo s có dãy số bằng chữ KẾT THÚC bằng một chữ số đứng ngay sau từ chỉ hàng ("một nghìn hai", "hai trăm năm",
// "ba nghìn tám"). Nói thường, chữ số đó là hàng kế dưới (1.200, 250, 3.800); parseViNumberWords cộng nó như hàng đơn vị (1.002, 205,
// 3.008). Dạng viết rõ hàng ("một nghìn hai trăm", "một trăm linh năm", "hai nghìn rưỡi") không tính. Cổng phán quyết dùng hàm này để
// không phán quyết con số đọc chưa chắc; sửa chính bộ đọc chờ D-OP-14.
func AmbiguousNumberWords(s string) bool {
	var prev, last viWord // loại của hai từ số cuối trong dãy đang xét
	tail := func() bool { return last == vwDigit && (prev == vwScale || prev == vwHundred) }
	for _, f := range strings.Fields(s) {
		w, ok := lookupNumberWord(strings.Trim(f, ".,;:!?()[]\"'"))
		if !ok {
			if tail() {
				return true
			}
			prev, last = 0, 0
			continue
		}
		prev, last = last, w.kind
	}
	return tail()
}

// parseViNumberWords đọc dãy từ chỉ số bắt đầu từ token i.
//
// Trả về giá trị, chỉ số token kế tiếp, và hasScale = dãy có từ chỉ hàng (mươi, trăm,
// nghìn, rưỡi…). Bên gọi chỉ chấp nhận dãy KHÔNG có hàng khi ngay sau là một đơn vị:
// "năm" một mình còn là "năm (year)", "năm 2026" không phải số 5.
func parseViNumberWords(toks []token, i int) (value *big.Rat, next int, hasScale bool, ok bool) {
	total := new(big.Rat)
	var group int64
	pending := int64(-1) // chữ số đang chờ hàng; -1 = không có
	lastUnit := int64(1)
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
		stop := false
		switch w.kind {
		case vwDigit:
			if pending >= 0 {
				stop = true // "hai ba" — hai chữ số liền nhau không thành một số
				break
			}
			pending = w.value
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
		consumed++
		last = j
		j++
	}
done:
	if consumed == 0 {
		return nil, i, false, false
	}
	if pending >= 0 {
		group += pending
	}
	total.Add(total, new(big.Rat).SetInt64(group))
	return total, last + 1, hasScale, true
}
