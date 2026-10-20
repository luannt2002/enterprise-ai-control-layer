package normalization

import (
	"testing"

	"github.com/luannt2002/enterprise-ai-systems/core/contracts"
)

// Nhóm nghìn ngăn bằng MỘT dấu cách là cách viết lượng mà tài liệu tenant yêu cầu chuẩn hoá
// (docs/tenants/construction/03-data-map.md:134: `4 000 lít` → 4.000 L, "khoảng trắng nghìn"): phần đầu 1–3 chữ số không bắt đầu bằng
// 0, mỗi nhóm sau đúng 3 chữ số cách đúng một dấu cách, ngay sau nhóm cuối là một đơn vị — gộp thành MỘT span. Con số không đứng đầu văn
// bản thì phần đầu có thể là số hiệu của chữ / ký hiệu đứng trước ("bể 2 500 lít", "phiếu #1 200 L"): vẫn gộp, nhưng đánh dấu CHƯA PHÂN
// ĐỊNH (UncertainGroups) để luồng hỏi hỏi lại và nạp dữ liệu cách ly. Không gộp thì phần đầu thành số mồ côi: "12 000 km" cho số 12,
// "500 000 đồng" cho số 500 (soát fd4d19c: F1–F4); danh sách từ nhãn thì rò ("tổng số", "bể", "#": C3-02, C3-05, F5).
func TestNhomNghinNganBangDauCach(t *testing.T) {
	e := mustEngine(t)
	for _, c := range []struct {
		in        string
		qty, nums []string // giá trị các span lượng / số trần, theo thứ tự
		uncertain bool     // span gộp (nếu có) chưa phân định
	}{
		{"4 000 lít", []string{"4000"}, nil, false},
		{"12 345 678 lít", []string{"12345678"}, nil, false},
		{"3 850 L", []string{"3850"}, nil, false},
		// không đứng đầu văn bản: gộp, chưa phân định
		{"cấp 3 850 lít", []string{"3850"}, nil, true},
		{"đo 1 490 lít dầu", []string{"1490"}, nil, true},
		{"đội 3 500 lít", []string{"3500"}, nil, true},
		{"phiếu 1 200 lít", []string{"1200"}, nil, true},
		{"máy 3 850 lít", []string{"3850"}, nil, true},
		{"tuần 2 400 lít", []string{"2400"}, nil, true},
		{"đổ bể 2 500 lít", []string{"2500"}, nil, true},
		{"lô A 1 200 lít", []string{"1200"}, nil, true},
		{"tổng số 3 850 lít", []string{"3850"}, nil, true},
		{"bể #2 500 lít", []string{"2500"}, nil, true},
		{"bể: 2 500 lít", []string{"2500"}, nil, true},
		{"(1 490 lít)", []string{"1490"}, nil, true},
		{"máy chạy 12 000 km", []string{"12000"}, nil, true}, // một span km, không có số mồ côi 12
		// không gộp
		{"bước 3 850", nil, []string{"3", "850"}, false},           // không có đơn vị
		{"3 850", nil, []string{"3", "850"}, false},                // không có đơn vị, đứng đầu
		{"máy 03 850 lít", []string{"850"}, []string{"3"}, false},  // số 0 đầu: số hiệu, không phải nhóm nghìn
		{"cấp 03 850 lít", []string{"850"}, []string{"3"}, false},  //
		{"3 85 lít", []string{"85"}, []string{"3"}, false},         // nhóm sau không đủ 3 chữ số
		{"3 8500 lít", []string{"8500"}, []string{"3"}, false},     // nhóm sau quá 3 chữ số
		{"1234 567 lít", []string{"567"}, []string{"1234"}, false}, // phần đầu quá 3 chữ số
		{"3  850 lít", []string{"850"}, []string{"3"}, false},      // hai dấu cách: không chắc là một số
	} {
		in := e.Normalize(c.in, Options{Now: now})
		var qty, nums []string
		uncertain := false
		for _, sp := range in.SpansOf(contracts.ValueQuantity) {
			qty = append(qty, sp.Value.Number)
			uncertain = uncertain || UncertainGroups(sp)
		}
		for _, sp := range in.SpansOf(contracts.ValueNumber) {
			nums = append(nums, sp.Value.Number)
		}
		if !equalStrs(qty, c.qty) || !equalStrs(nums, c.nums) || uncertain != c.uncertain {
			t.Errorf("%q: lượng %v, số %v, chưa phân định %v; muốn lượng %v, số %v, chưa phân định %v", c.in, qty, nums, uncertain, c.qty, c.nums, c.uncertain)
		}
	}
	// tiền viết nhóm nghìn sau chữ: MỘT span tiền, không có số trần mồ côi (soát fd4d19c, F2 / F4)
	in := e.Normalize("sửa chữa 500 000 đồng", Options{Now: now})
	if n := len(in.SpansOf(contracts.ValueNumber)) + len(in.SpansOf(contracts.ValueQuantity)); n != 0 {
		t.Errorf("\"sửa chữa 500 000 đồng\": %d span số / lượng, muốn 0 (chỉ một span tiền)", n)
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
