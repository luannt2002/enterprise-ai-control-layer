package normalization

import (
	"strings"
	"unicode"
	"unicode/utf8"

	"golang.org/x/text/unicode/norm"
)

// NFC đưa chuỗi về dạng tổ hợp chuẩn.
//
// Bàn phím trên máy Apple và một số trình duyệt gửi tiếng Việt ở dạng NFD (chữ cái +
// dấu rời). Hai chuỗi nhìn giống hệt nhau nhưng khác byte thì không khớp từ khoá nào,
// nên mọi đầu vào đi qua NFC trước tiên.
func NFC(s string) string { return norm.NFC.String(s) }

// foldTable ánh xạ chữ có dấu tiếng Việt về chữ không dấu.
var foldTable = func() map[rune]rune {
	groups := map[rune]string{
		'a': "áàảãạăắằẳẵặâấầẩẫậ",
		'e': "éèẻẽẹêếềểễệ",
		'i': "íìỉĩị",
		'o': "óòỏõọôốồổỗộơớờởỡợ",
		'u': "úùủũụưứừửữự",
		'y': "ýỳỷỹỵ",
		'd': "đ",
	}
	m := map[rune]rune{}
	for base, chars := range groups {
		for _, c := range chars {
			m[c] = base
			m[unicode.ToUpper(c)] = unicode.ToUpper(base)
		}
	}
	return m
}()

// foldRune hạ chữ thường và bỏ dấu một rune.
func foldRune(r rune) rune {
	r = unicode.ToLower(r)
	if f, ok := foldTable[r]; ok {
		return f
	}
	return r
}

// Fold hạ chữ thường và bỏ dấu tiếng Việt. Dùng cho so khớp khi người gõ không dấu.
func Fold(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	for _, r := range s {
		b.WriteRune(foldRune(r))
	}
	return b.String()
}

// toneMarks là năm dấu thanh tiếng Việt ở dạng dấu rời: sắc, huyền, hỏi, ngã, nặng.
var toneMarks = map[rune]bool{'\u0301': true, '\u0300': true, '\u0309': true, '\u0303': true, '\u0323': true}

// ToneKey trả khoá so khớp GIỮ DẤU của MỘT từ, không phụ thuộc chỗ đặt dấu thanh: "hoá" và "hóa"
// (kiểu bỏ dấu cũ và mới) cùng khoá, còn "bí" và "bị", "trường" và "trưởng" khác khoá. Chữ thường;
// dấu mũ, móc, trăng giữ nguyên chỗ; dấu thanh tách ra cuối.
func ToneKey(s string) string {
	var base strings.Builder
	var tone []rune
	for _, r := range norm.NFD.String(strings.ToLower(s)) {
		if toneMarks[r] {
			tone = append(tone, r)
			continue
		}
		base.WriteRune(r)
	}
	return norm.NFC.String(base.String()) + string(tone)
}

// HasDiacritic báo chuỗi có chữ mang dấu tiếng Việt không — câu không dấu được khớp bằng bản bỏ dấu, nên nhiều chữ trùng nhau.
func HasDiacritic(s string) bool { return hasDiacritic(s) }

// hasDiacritic báo chuỗi có chữ mang dấu tiếng Việt không.
func hasDiacritic(s string) bool {
	for _, r := range s {
		if _, ok := foldTable[unicode.ToLower(r)]; ok {
			return true
		}
	}
	return false
}

// CollapseSpaces gộp mọi khoảng trắng liên tiếp thành một dấu cách và cắt hai đầu.
func CollapseSpaces(s string) string { return strings.Join(strings.Fields(s), " ") }

// tokKind là loại token.
type tokKind int

const (
	tokWord  tokKind = iota // chữ (kể cả chữ tiếng Việt)
	tokNum                  // chữ số, có thể có dấu . , bên trong: "4.000", "0,35"
	tokSym                  // ký hiệu đơn lẻ: / + - ~ % : …
	tokSpace                // khoảng trắng
)

// token là một đoạn của văn bản NFC, kèm vị trí byte.
type token struct {
	kind       tokKind
	text       string
	start, end int
}

// isSignedNumberAt báo dấu "-" / "−" ở vị trí i là DẤU ÂM: ngay sau là chữ số, ngay trước là đầu
// văn bản, khoảng trắng hoặc dấu mở ngoặc. "EX-03", "R-004" (trước là chữ) và "18-20" (trước là
// số) không phải số âm.
func isSignedNumberAt(s string, i int) bool {
	r, w := utf8.DecodeRuneInString(s[i:])
	if r != '-' && r != '−' {
		return false
	}
	if next, _ := utf8.DecodeRuneInString(s[i+w:]); i+w >= len(s) || !unicode.IsDigit(next) {
		return false
	}
	if i == 0 {
		return true
	}
	prev, _ := utf8.DecodeLastRuneInString(s[:i])
	return unicode.IsSpace(prev) || prev == '('
}

// tokenize cắt văn bản thành token, giữ vị trí byte để span trỏ đúng nguyên văn.
//
// Chữ và số dính nhau bị tách ("4k" → 4 | k, "1tr6" → 1 | tr | 6, "205/55R16" →
// 205 | / | 55 | R | 16): bộ đọc phía sau ghép lại theo luật, nhờ vậy một luật không
// phải đoán trước mọi cách viết dính.
func tokenize(s string) []token {
	var toks []token
	i := 0
	for i < len(s) {
		r, w := utf8.DecodeRuneInString(s[i:])
		start := i
		switch {
		case unicode.IsSpace(r):
			for i < len(s) {
				r2, w2 := utf8.DecodeRuneInString(s[i:])
				if !unicode.IsSpace(r2) {
					break
				}
				i += w2
			}
			toks = append(toks, token{tokSpace, s[start:i], start, i})
		case unicode.IsDigit(r) || isSignedNumberAt(s, i):
			i += w
			for i < len(s) {
				r2, w2 := utf8.DecodeRuneInString(s[i:])
				if unicode.IsDigit(r2) {
					i += w2
					continue
				}
				// Dấu phân cách chỉ thuộc về số khi ngay sau nó là chữ số: "4.000" là
				// một số, còn "4." cuối câu thì dấu chấm là dấu câu.
				if (r2 == '.' || r2 == ',') && i+w2 < len(s) {
					r3, _ := utf8.DecodeRuneInString(s[i+w2:])
					if unicode.IsDigit(r3) {
						i += w2
						continue
					}
				}
				break
			}
			toks = append(toks, token{tokNum, s[start:i], start, i})
		case unicode.IsLetter(r) || unicode.Is(unicode.Mn, r):
			i += w
			for i < len(s) {
				r2, w2 := utf8.DecodeRuneInString(s[i:])
				if !unicode.IsLetter(r2) && !unicode.Is(unicode.Mn, r2) {
					break
				}
				i += w2
			}
			toks = append(toks, token{tokWord, s[start:i], start, i})
		default:
			i += w
			toks = append(toks, token{tokSym, s[start:i], start, i})
		}
	}
	return toks
}

// isBoundaryAfter báo vị trí i trong s là ranh giới từ (hết chuỗi, hoặc ký tự sau
// không phải chữ cái/chữ số).
func isBoundaryAfter(s string, i int) bool {
	if i >= len(s) {
		return true
	}
	r, _ := utf8.DecodeRuneInString(s[i:])
	return !unicode.IsLetter(r) && !unicode.IsDigit(r) && !unicode.Is(unicode.Mn, r)
}

// isBoundaryBefore báo vị trí i là đầu một từ.
func isBoundaryBefore(s string, i int) bool {
	if i <= 0 {
		return true
	}
	r, _ := utf8.DecodeLastRuneInString(s[:i])
	return !unicode.IsLetter(r) && !unicode.IsDigit(r) && !unicode.Is(unicode.Mn, r)
}

// ContainsPhrase tìm cụm từ ở ranh giới từ trong văn bản, theo cùng luật dấu với mọi so
// khớp của EAIS: gõ không dấu vẫn khớp; có dấu thì phải đúng dấu.
func ContainsPhrase(text, phrase string) bool {
	text, phrase = NFC(text), NFC(phrase)
	for i := 0; i < len(text); {
		if isBoundaryBefore(text, i) {
			if _, ok := matchAliasAt(text, i, phrase, false); ok {
				return true
			}
		}
		_, w := utf8.DecodeRuneInString(text[i:])
		i += w
	}
	return false
}

// PhraseSpans trả mọi khoảng [đầu, cuối) (byte, trên bản NFC của text) mà phrase khớp,
// cùng luật dấu và ranh giới từ như ContainsPhrase.
func PhraseSpans(text, phrase string) [][2]int {
	text, phrase = NFC(text), NFC(phrase)
	var out [][2]int
	for i := 0; i < len(text); {
		if isBoundaryBefore(text, i) {
			if j, ok := matchAliasAt(text, i, phrase, false); ok {
				out = append(out, [2]int{i, j})
			}
		}
		_, w := utf8.DecodeRuneInString(text[i:])
		i += w
	}
	return out
}

// matchAliasAt thử khớp alias tại vị trí i của s và đòi ranh giới từ ngay sau; trả vị trí
// kết thúc.
//
// Luật dấu: nếu đoạn văn bản ở vị trí đó KHÔNG có dấu (người gõ không dấu) thì so khớp
// đã bỏ dấu; nếu có dấu thì phải khớp đúng dấu (không phân biệt hoa thường). Nhờ vậy
// "dau 4k" khớp "dầu", nhưng "đầu tháng" không bị hiểu thành "dầu".
func matchAliasAt(s string, i int, alias string, caseSensitive bool) (int, bool) {
	j, ok := matchAliasPrefix(s, i, alias, caseSensitive)
	if !ok || !isBoundaryAfter(s, j) {
		return 0, false
	}
	return j, true
}

// matchAliasPrefix như matchAliasAt nhưng KHÔNG đòi ranh giới phía sau — dùng cho hậu tố
// viết dính chữ số ("1tr6": sau "tr" là "6").
func matchAliasPrefix(s string, i int, alias string, caseSensitive bool) (int, bool) {
	if alias == "" || i >= len(s) {
		return 0, false
	}
	j := i
	segHasDiacritic := false
	for _, ar := range alias {
		if j >= len(s) {
			return 0, false
		}
		r, w := utf8.DecodeRuneInString(s[j:])
		if _, ok := foldTable[unicode.ToLower(r)]; ok {
			segHasDiacritic = true
		}
		switch {
		case caseSensitive:
			if r != ar {
				return 0, false
			}
		case unicode.ToLower(r) == unicode.ToLower(ar):
		case foldRune(r) == foldRune(ar):
			// khớp sau khi bỏ dấu — chỉ hợp lệ nếu đoạn người gõ không mang dấu
		default:
			return 0, false
		}
		j += w
	}
	if !caseSensitive && !strings.EqualFold(s[i:j], alias) {
		// Khớp nhờ bỏ dấu. Người gõ CÓ dấu nhưng dấu khác alias ("đầu" so với "dầu") → không
		// khớp. Đoạn không dấu chỉ được hiểu là "gõ thiếu dấu" khi CẢ câu không dấu: trong
		// "bao nhiêu" chữ "bao" là chính nó, không phải "báo" — đoán thiếu dấu ở đây đổi
		// nghĩa nghiệp vụ của câu hỏi.
		if segHasDiacritic || hasDiacritic(s) {
			return 0, false
		}
	}
	return j, true
}
