// Package decimal làm toán trên số thập phân dạng chuỗi chuẩn ("4000", "0.35").
//
// Tiền và số lượng phải so khớp TUYỆT ĐỐI: float64 làm "1602000" lệch ở chữ số cuối sau
// một phép nhân đơn vị, rồi phép "khớp với bằng chứng" đỏ oan. Mọi số nghiệp vụ trong
// EACL đi qua package này.
package decimal

import (
	"fmt"
	"math/big"
	"strings"
)

// Parse đọc chuỗi thập phân chuẩn.
func Parse(s string) (*big.Rat, bool) {
	if s == "" {
		return nil, false
	}
	r, ok := new(big.Rat).SetString(s)
	return r, ok
}

// Format in số thành chuỗi chuẩn: không dấu nghìn, dấu chấm thập phân, không số 0 thừa.
// Số không biểu diễn hữu hạn được (1/3) bị làm tròn tới 6 chữ số thập phân.
func Format(r *big.Rat) string {
	if r.IsInt() {
		return r.Num().String()
	}
	s := strings.TrimRight(r.FloatString(6), "0")
	return strings.TrimSuffix(s, ".")
}

// Canonical chuẩn hoá chuỗi số đã ở dạng máy ("4000.0" → "4000").
func Canonical(s string) (string, bool) {
	r, ok := Parse(s)
	if !ok {
		return "", false
	}
	return Format(r), true
}

// Equal so hai số theo giá trị.
func Equal(a, b string) bool {
	x, ok1 := Parse(a)
	y, ok2 := Parse(b)
	return ok1 && ok2 && x.Cmp(y) == 0
}

func binary(a, b string, op func(x, y *big.Rat) *big.Rat) (string, error) {
	x, ok1 := Parse(a)
	y, ok2 := Parse(b)
	if !ok1 || !ok2 {
		return "", fmt.Errorf("số không hợp lệ: %q, %q", a, b)
	}
	return Format(op(x, y)), nil
}

// Add cộng hai số.
func Add(a, b string) (string, error) {
	return binary(a, b, func(x, y *big.Rat) *big.Rat { return new(big.Rat).Add(x, y) })
}

// Sub trừ a − b.
func Sub(a, b string) (string, error) {
	return binary(a, b, func(x, y *big.Rat) *big.Rat { return new(big.Rat).Sub(x, y) })
}

// Mul nhân hai số.
func Mul(a, b string) (string, error) {
	return binary(a, b, func(x, y *big.Rat) *big.Rat { return new(big.Rat).Mul(x, y) })
}

// Quo chia a / b. Chia cho 0 là lỗi.
func Quo(a, b string) (string, error) {
	y, ok := Parse(b)
	if ok && y.Sign() == 0 {
		return "", fmt.Errorf("chia cho 0")
	}
	return binary(a, b, func(x, y *big.Rat) *big.Rat { return new(big.Rat).Quo(x, y) })
}

// Cmp so sánh a với b: -1, 0, 1.
func Cmp(a, b string) (int, error) {
	x, ok1 := Parse(a)
	y, ok2 := Parse(b)
	if !ok1 || !ok2 {
		return 0, fmt.Errorf("số không hợp lệ: %q, %q", a, b)
	}
	return x.Cmp(y), nil
}

// Within báo |a − b| ≤ tol.
func Within(a, b, tol string) (bool, error) {
	d, err := Sub(a, b)
	if err != nil {
		return false, err
	}
	d = strings.TrimPrefix(d, "-")
	c, err := Cmp(d, tol)
	return c <= 0, err
}
