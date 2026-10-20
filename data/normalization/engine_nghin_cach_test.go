package normalization

import (
	"testing"

	"github.com/luannt2002/enterprise-ai-systems/core/contracts"
)

// Nhóm nghìn ngăn bằng MỘT dấu cách là cách viết lượng mà tài liệu tenant yêu cầu chuẩn hoá
// (docs/tenants/construction/03-data-map.md:134: `4 000 lít` → 4.000 L, "khoảng trắng nghìn"). Chỉ gộp khi chắc là một lượng:
// phần đầu 1–3 chữ số không bắt đầu bằng 0 và KHÔNG có chữ đứng ngay trước, mỗi nhóm sau đúng 3 chữ số cách đúng một dấu cách,
// ngay sau nhóm cuối là một đơn vị. Có chữ đứng trước ("bể 2 500 lít", "lô A 1 200 lít", "tổng số 3 850 lít", "cấp 3 850 lít")
// thì phần đầu có thể là số hiệu của chữ đó: danh sách từ nhãn nào cũng rò (soát C3, C3-02 / C3-05), nên không gộp — luồng hỏi
// thấy hai con số và hỏi lại, nạp dữ liệu cách ly (P5).
func TestNhomNghinNganBangDauCach(t *testing.T) {
	e := mustEngine(t)
	for _, c := range []struct {
		in        string
		qty, nums []string // giá trị các span lượng / số trần, theo thứ tự
	}{
		{"4 000 lít", []string{"4000"}, nil},
		{"12 345 678 lít", []string{"12345678"}, nil},
		{"3 850 L", []string{"3850"}, nil},
		{"EX-03: 3 850 lít", []string{"3850"}, []string{"3"}}, // "3" là số của mã EX-03
		{"máy 03 4 000 lít", []string{"4000"}, []string{"3"}}, // số của tên máy đứng trước, không phải chữ
		{"(1 490 lít)", []string{"1490"}, nil},
		// không gộp
		{"bước 3 850", nil, []string{"3", "850"}},           // không có đơn vị
		{"3 850", nil, []string{"3", "850"}},                // không có đơn vị, không có chữ đứng trước
		{"máy 03 850 lít", []string{"850"}, []string{"3"}},  // số 0 đầu: số hiệu, không phải nhóm nghìn
		{"3 85 lít", []string{"85"}, []string{"3"}},         // nhóm sau không đủ 3 chữ số
		{"3 8500 lít", []string{"8500"}, []string{"3"}},     // nhóm sau quá 3 chữ số
		{"1234 567 lít", []string{"567"}, []string{"1234"}}, // phần đầu quá 3 chữ số
		{"3  850 lít", []string{"850"}, []string{"3"}},      // hai dấu cách: không chắc là một số
		// có chữ đứng ngay trước: phần đầu có thể là số hiệu ("đội 3, 500 lít"), không gộp
		{"đội 3 500 lít", []string{"500"}, []string{"3"}},
		{"phiếu 1 200 lít", []string{"200"}, []string{"1"}},
		{"máy 3 850 lít", []string{"850"}, []string{"3"}},
		{"tuần 2 400 lít", []string{"400"}, []string{"2"}},
		{"đổ bể 2 500 lít", []string{"500"}, []string{"2"}},
		{"lô A 1 200 lít", []string{"200"}, []string{"1"}},
		{"tổng số 3 850 lít", []string{"850"}, []string{"3"}},
		{"cấp 3 850 lít", []string{"850"}, []string{"3"}},
		{"đo 1 490 lít dầu", []string{"490"}, []string{"1"}},
		// số 0 đầu không phải phần đầu của nhóm nghìn kể cả khi không có chữ đứng trước
		{"cấp 03 850 lít", []string{"850"}, []string{"3"}},
		{"(03 850 lít)", []string{"850"}, []string{"3"}},
	} {
		in := e.Normalize(c.in, Options{Now: now})
		var qty, nums []string
		for _, sp := range in.SpansOf(contracts.ValueQuantity) {
			qty = append(qty, sp.Value.Number)
		}
		for _, sp := range in.SpansOf(contracts.ValueNumber) {
			nums = append(nums, sp.Value.Number)
		}
		if !equalStrs(qty, c.qty) || !equalStrs(nums, c.nums) {
			t.Errorf("%q: lượng %v, số %v; muốn lượng %v, số %v", c.in, qty, nums, c.qty, c.nums)
		}
	}
}

func equalStrs(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
