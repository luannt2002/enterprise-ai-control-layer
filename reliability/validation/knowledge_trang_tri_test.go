package validation

import "testing"

// Dấu trang trí (dấu phẩy, Markdown, thẻ HTML) giữa một số và chữ trùng tên đơn vị chỉ được bỏ qua khi số có dáng một LƯỢNG (có dấu nghìn /
// thập phân, hoặc từ hai chữ số). Số một chữ số là số thứ tự / nhãn: "Bước 2, Khối Kế hoạch lập phiếu." không phải "2 khối". Bản sửa R
// (skipDecoration) làm câu quy trình lành này bị chặn (SYSTEM_FAILURE), trái docs/04 §6 G5 "0 chặn oan" (soát R, R-A15).
func TestTrangTriChiBoQuaKhiSoCoDangLuong(t *testing.T) {
	words := []string{"lít", "m³", "m3", "mét khối", "khối", "L", "l"} // như bí danh đơn vị của pack (lexicon.yaml)
	for _, c := range []struct {
		text  string
		block bool
	}{
		{"Bước 2, Khối Kế hoạch lập phiếu.", false},
		{"**Bước 2** Khối Kế hoạch lập phiếu.", false},
		{"<b>Bước 2</b> Khối Kế hoạch lập phiếu.", false},
		{"Mục 3, Khối Vận hành ký xác nhận.", false},
		// lượng thật vẫn bị chặn
		{"Định mức cấp phát cho máy xúc: **4.000** lít mỗi ca.", true},
		{"Định mức cấp phát cho máy xúc: 4.000, lít, mỗi ca.", true},
		{"Định mức cấp phát cho máy xúc: <b>4.000</b> lít.", true},
		{"Định mức cấp phát cho máy xúc: **40** lít.", true},
		{"Định mức cấp phát cho máy xúc: 4 lít mỗi ca.", true},
		{"Định mức cấp phát cho máy xúc: **4** lít mỗi ca.", true},
		{"Định mức cấp phát cho máy xúc: **5** mét khối mỗi ngày.", true},
		{"Định mức cấp phát cho máy xúc: **4.000** khối mỗi ca.", true}, // số có dáng lượng: "khối" là đơn vị
	} {
		if got := len(ForbidUnitWords(c.text, words)) > 0; got != c.block {
			t.Errorf("%q: chặn = %v, muốn %v", c.text, got, c.block)
		}
	}
}
