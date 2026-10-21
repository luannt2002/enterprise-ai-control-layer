// Package validation là chặng S9: kiểm câu trả lời bằng MÃ trước khi giao.
//
// Luật cứng: model — và cả mẫu câu — không bao giờ là nguồn của một con số. Mọi lượng,
// số tiền, con số và mã trong câu trả lời phải khớp (sau chuẩn hoá) một giá trị có trong
// gói bằng chứng. Kiểm thất bại thì câu đó không được giao (luật cứng 3 của luồng hỏi).
package validation

import (
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/luannt2002/enterprise-ai-control-layer/core/contracts"
	"github.com/luannt2002/enterprise-ai-control-layer/core/decimal"
	"github.com/luannt2002/enterprise-ai-control-layer/data/normalization"
)

// Fidelity kiểm số và mã trong câu trả lời so với bằng chứng.
//
// ignore là các chuỗi được phép xuất hiện mà không cần bằng chứng số (tên/khoá thực thể
// đã nhận dạng — "EX-03" chứa số 03 nhưng đó là tên, không phải số liệu; số của người hỏi trong câu
// xác nhận). Chỉ lần xuất hiện TRỌN VẸN mới được bỏ đi, xem maskWhole.
func Fidelity(norm *normalization.Engine, draft contracts.Draft, pkg contracts.ContextPackage, ignore []string, now time.Time) []contracts.Violation {
	text := draft.Text
	for _, s := range ignore {
		text = maskWhole(text, s)
	}
	got := norm.Normalize(text, normalization.Options{Now: now})

	var out []contracts.Violation
	for _, sp := range got.Spans {
		switch sp.Kind {
		case contracts.ValueQuantity, contracts.ValueMoney, contracts.ValueNumber:
			if !numberBacked(norm, sp.Value, pkg) {
				out = append(out, contracts.Violation{
					Validator: "numeric_fidelity", Code: "number_without_evidence", Severity: contracts.SeverityS1,
					Detail: "câu trả lời có " + sp.Raw + " nhưng không bằng chứng nào chứa giá trị này",
				})
			}
		case contracts.ValueCode:
			if !codeBacked(sp.Value.Text, pkg) {
				out = append(out, contracts.Violation{
					Validator: "code_fidelity", Code: "code_without_evidence", Severity: contracts.SeverityS1,
					Detail: "câu trả lời có mã " + sp.Raw + " nhưng không bằng chứng nào chứa mã này",
				})
			}
		}
	}
	return out
}

// maskWhole thay mọi lần xuất hiện TRỌN VẸN của s trong text bằng dấu cách. Trọn vẹn = không dính vào một
// từ hay một số lớn hơn: "0 L" nằm cuối "3.850 L" (hay "30 L") không phải "0 L" của người hỏi — cắt nó để
// lại "3.85" (số chưa từng được kiểm) hoặc "3" (số có thể ngẫu nhiên trùng một số có bằng chứng).
func maskWhole(text, s string) string {
	if s == "" {
		return text
	}
	var b strings.Builder
	for i := 0; ; {
		j := strings.Index(text[i:], s)
		if j < 0 {
			b.WriteString(text[i:])
			return b.String()
		}
		j += i
		if end := j + len(s); !joinedBefore(text[:j], s) && !joinedAfter(text[end:], s) {
			b.WriteString(text[i:j])
			b.WriteByte(' ')
			i = end
			continue
		}
		// lần này dính vào chữ/số bên cạnh: giữ nguyên, tìm tiếp từ ký tự kế (các lần xuất hiện chồng nhau)
		_, w := utf8.DecodeRuneInString(text[j:])
		b.WriteString(text[i : j+w])
		i = j + w
	}
}

// isWordRune: chữ, chữ số hoặc dấu kết hợp — ký tự cấu thành một từ / một số.
func isWordRune(r rune) bool { return unicode.IsLetter(r) || unicode.IsDigit(r) || unicode.IsMark(r) }

// joinedBefore: ký tự liền trước s làm s thành đuôi của một từ hay một số lớn hơn — chữ/số dính liền,
// dấu âm hay gạch nối trước chữ số ("-50 L" không phải "50 L"), dấu nghìn / thập phân giữa hai chữ số.
func joinedBefore(before, s string) bool {
	prev, w := utf8.DecodeLastRuneInString(before)
	if w == 0 {
		return false
	}
	first, _ := utf8.DecodeRuneInString(s)
	if isWordRune(first) && isWordRune(prev) {
		return true
	}
	if !unicode.IsDigit(first) {
		return false
	}
	if prev == '-' || prev == '−' {
		return true
	}
	if prev == '.' || prev == ',' {
		d, _ := utf8.DecodeLastRuneInString(before[:len(before)-w])
		return unicode.IsDigit(d)
	}
	return false
}

// joinedAfter: ký tự liền sau s làm s thành đầu của một từ hay một số lớn hơn — chữ/số dính liền, hoặc
// dấu nghìn / thập phân còn chữ số nối tiếp ("3" trong "3.850").
func joinedAfter(after, s string) bool {
	next, w := utf8.DecodeRuneInString(after)
	if w == 0 {
		return false
	}
	last, _ := utf8.DecodeLastRuneInString(s)
	if isWordRune(last) && isWordRune(next) {
		return true
	}
	if unicode.IsDigit(last) && (next == '.' || next == ',') {
		d, _ := utf8.DecodeRuneInString(after[w:])
		return unicode.IsDigit(d)
	}
	return false
}

// numberBacked: giá trị có mặt trong bằng chứng — cùng đơn vị, hoặc đổi được về cùng đơn
// vị, hoặc là số trần bằng đúng một số trong bằng chứng.
func numberBacked(norm *normalization.Engine, v contracts.Value, pkg contracts.ContextPackage) bool {
	for _, e := range pkg.Evidence {
		for _, ev := range e.Values {
			if ev.Number == "" {
				continue
			}
			if v.Unit == "" || ev.Unit == "" {
				if decimal.Equal(v.Number, ev.Number) {
					return true
				}
				continue
			}
			if v.Unit == ev.Unit {
				if decimal.Equal(v.Number, ev.Number) {
					return true
				}
				continue
			}
			if conv, err := norm.Convert(v, ev.Unit); err == nil && decimal.Equal(conv.Number, ev.Number) {
				return true
			}
		}
	}
	return false
}

func codeBacked(code string, pkg contracts.ContextPackage) bool {
	for _, e := range pkg.Evidence {
		if strings.Contains(e.Text, code) {
			return true
		}
		for _, ev := range e.Values {
			if ev.Text == code {
				return true
			}
		}
	}
	return false
}
