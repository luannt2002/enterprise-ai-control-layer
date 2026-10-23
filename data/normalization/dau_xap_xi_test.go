package normalization_test

import (
	"testing"

	"github.com/luannt2002/enterprise-ai-control-layer/core/tenant"
	"github.com/luannt2002/enterprise-ai-control-layer/data/normalization"
)

// Dấu ước lượng dạng KÝ HIỆU mà pack khai ở approx_markers ("≈" của pack cầu đường) phải đánh dấu lượng đứng sau là ước lượng, ở chữ tự do lẫn ô
// của nguồn có cấu trúc — như dấu dạng chữ ("khoảng"). Trước đây chỉ "~" (viết cứng trong mã) được nhận: phiếu kho ghi "≈ 2,6 m3" mất cờ
// ước lượng ở đường nạp và phiếu chênh lệch mang approx = false — 4 ca đỏ của bộ giữ lại P1 (docs/g7/holdout-p1/ket-qua-p1/phan-tich.md).
func TestDauUocLuongKyHieuCuaPack(t *testing.T) {
	rt, err := tenant.Load("../../tenants/construction/pack")
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range []struct {
		in     string
		approx bool
	}{
		{"≈ 2,6 m3", true}, {"≈2,6 m3", true}, {"≈ 2600 L", true}, {"phiếu ghi ≈ 1,1 m3", true},
		{"~ 2,6 m3", true}, {"khoảng 2,6 m3", true},
		{"2,6 m3", false}, {"≈", false}, {"2,6 m3 ≈", false},
	} {
		for _, cell := range []bool{true, false} {
			in := rt.Normalizer.Normalize(c.in, normalization.Options{Cell: cell})
			got := false
			for _, sp := range in.Spans {
				got = got || sp.Value.Approx
			}
			if got != c.approx {
				t.Errorf("%q (ô=%v): ước lượng = %v, muốn %v", c.in, cell, got, c.approx)
			}
		}
	}
}
