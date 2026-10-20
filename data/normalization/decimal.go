package normalization

import (
	"math/big"
	"strings"

	"github.com/luannt2002/enterprise-ai-systems/core/decimal"
)

func parseDecimal(s string) (*big.Rat, bool) { return decimal.Parse(s) }

func formatDecimal(r *big.Rat) string { return decimal.Format(r) }

// parseLocaleNumber đọc số người gõ, theo thói quen Việt Nam: dấu chấm phân cách nghìn,
// dấu phẩy thập phân — nhưng người Việt cũng hay gõ kiểu Anh ("0.35km", "4,000 L").
//
// Luật (đi từ chắc nhất tới kém chắc nhất):
//  1. Có cả '.' và ',': dấu xuất hiện SAU CÙNG là dấu thập phân ("1.602,5" và "1,602.5"
//     đều là 1602,5).
//  2. Một loại dấu, xuất hiện ≥2 lần: dấu nghìn — mọi nhóm sau dấu phải đủ 3 chữ số,
//     không thì không phải số hợp lệ.
//  3. Một loại dấu, xuất hiện 1 lần, sau nó đúng 3 chữ số và phần nguyên khác 0: dấu
//     nghìn ("4.000" = 4000, "4,000" = 4000). Phần nguyên là 0 thì là thập phân
//     ("0.350" = 0,35) vì nhóm nghìn không bắt đầu bằng 0.
//  4. Một loại dấu, 1 lần, sau nó 1–2 hoặc ≥4 chữ số: dấu thập phân ("0.35", "2,5").
//
// ambiguous = true ở luật 3 với dấu '.', vì người quen kiểu Anh có thể muốn 4,0 — nhưng
// trong ngữ cảnh tiếng Việt cách đọc nghìn chiếm áp đảo, nên vẫn đọc là nghìn và để
// tầng kiểm miền giá trị bắt chỗ vô lý.
func parseLocaleNumber(s string) (value string, ambiguous bool, ok bool) {
	// Dấu âm ở đầu ("-68", "−68"): đọc phần sau như số dương rồi gắn dấu — hiệu của phép tính
	// (kho xuất − cảm biến) được âm, và câu trả lời phải đọc lại đúng số đó.
	for _, minus := range []string{"-", "−"} {
		if rest, found := strings.CutPrefix(s, minus); found {
			v, amb, ok := parseLocaleNumber(rest)
			if !ok || strings.HasPrefix(v, "-") {
				return "", false, false
			}
			if v != "0" {
				v = "-" + v
			}
			return v, amb, true
		}
	}
	dots := strings.Count(s, ".")
	commas := strings.Count(s, ",")
	clean := func(intPart, fracPart string) (string, bool) {
		n := strings.NewReplacer(".", "", ",", "").Replace(intPart)
		if fracPart != "" {
			n += "." + fracPart
		}
		return decimal.Canonical(n)
	}
	switch {
	case dots == 0 && commas == 0:
		v, ok := decimal.Canonical(s)
		return v, false, ok
	case dots > 0 && commas > 0:
		last := strings.LastIndexAny(s, ".,")
		if !validThousands(s[:last]) {
			return "", false, false
		}
		v, ok := clean(s[:last], s[last+1:])
		return v, false, ok
	default:
		sep := "."
		if commas > 0 {
			sep = ","
		}
		parts := strings.Split(s, sep)
		if len(parts) > 2 {
			for _, p := range parts[1:] {
				if len(p) != 3 {
					return "", false, false
				}
			}
			v, ok := clean(strings.Join(parts, ""), "")
			return v, false, ok
		}
		intPart, frac := parts[0], parts[1]
		if len(frac) == 3 && strings.TrimLeft(intPart, "0") != "" {
			v, ok := clean(intPart+frac, "")
			return v, sep == ".", ok
		}
		v, ok := clean(intPart, frac)
		return v, false, ok
	}
}

// validThousands kiểm phần nguyên có dấu nghìn đúng nhóm 3 chữ số không.
func validThousands(s string) bool {
	parts := strings.FieldsFunc(s, func(r rune) bool { return r == '.' || r == ',' })
	if len(parts) == 0 {
		return false
	}
	for _, p := range parts[1:] {
		if len(p) != 3 {
			return false
		}
	}
	return true
}
