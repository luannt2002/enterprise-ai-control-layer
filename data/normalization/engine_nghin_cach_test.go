package normalization

import (
	"testing"

	"github.com/luannt2002/enterprise-ai-systems/core/contracts"
)

// Nhóm nghìn ngăn bằng MỘT dấu cách là cách viết lượng mà tài liệu tenant yêu cầu chuẩn hoá
// (docs/tenants/construction/03-data-map.md:134: `4 000 lít` → 4.000 L, "khoảng trắng nghìn"). Trước đây "3 850 lít" thành
// số "3" + lượng "850 lít", và câu xác nhận đúng bị trả "Không … không phải 850 L" (phát hiện qua ca 264 của H1 SAU lần chạy
// chính thức; H1 nay là bộ hồi quy). Chỉ gộp khi chắc là một lượng: phần đầu 1–3 chữ số không bắt đầu bằng 0, mỗi nhóm sau
// đúng 3 chữ số cách đúng một dấu cách, và ngay sau nhóm cuối là một đơn vị.
func TestNhomNghinNganBangDauCach(t *testing.T) {
	e := mustEngine(t)
	for _, c := range []struct {
		in        string
		qty, nums []string // giá trị các span lượng / số trần, theo thứ tự
	}{
		{"cấp 3 850 lít", []string{"3850"}, nil},
		{"4 000 lít", []string{"4000"}, nil},
		{"đo 1 490 lít dầu", []string{"1490"}, nil},
		{"12 345 678 lít", []string{"12345678"}, nil},
		{"3 850 L", []string{"3850"}, nil},
		// không gộp
		{"bước 3 850", nil, []string{"3", "850"}},           // không có đơn vị
		{"máy 03 850 lít", []string{"850"}, []string{"3"}},  // số 0 đầu: số hiệu, không phải nhóm nghìn
		{"3 85 lít", []string{"85"}, []string{"3"}},         // nhóm sau không đủ 3 chữ số
		{"3 8500 lít", []string{"8500"}, []string{"3"}},     // nhóm sau quá 3 chữ số
		{"1234 567 lít", []string{"567"}, []string{"1234"}}, // phần đầu quá 3 chữ số
		{"3  850 lít", []string{"850"}, []string{"3"}},      // hai dấu cách: không chắc là một số
		// số đứng sau từ nhãn / từ đếm / từ chỉ kỳ là số hiệu, không phải phần đầu của lượng: "đội 3, 500 lít"
		{"đội 3 500 lít", []string{"500"}, []string{"3"}},
		{"phiếu 1 200 lít", []string{"200"}, []string{"1"}},
		{"máy 3 850 lít", []string{"850"}, []string{"3"}},
		{"tuần 2 400 lít", []string{"400"}, []string{"2"}},
		// số 0 đầu không phải phần đầu của nhóm nghìn kể cả khi không có từ nhãn đứng trước
		{"cấp 03 850 lít", []string{"850"}, []string{"3"}},
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
