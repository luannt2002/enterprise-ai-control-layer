package validation

import "testing"

// "Khối" mở đầu một TÊN RIÊNG ("Bước 2, Khối Kế hoạch lập phiếu.") là khối phòng ban, không phải "2 khối": bản sửa R (skipDecoration) làm câu
// quy trình lành này bị chặn (SYSTEM_FAILURE), trái docs/04 §6 G5 "0 chặn oan" (soát R, R-A15). Bản miễn đầu tiên (37e8a08: dấu trang trí +
// số một chữ số + "khối" / "l") rộng quá: "**5** khối", "**5** L", "| 5 | khối |" lọt lớp kiểm (soát 37e8a08, RM-1 / F7). Nay chỉ miễn khi số
// không có dáng lượng, chữ đơn vị VIẾT HOA và chữ kế tiếp cũng viết hoa; "L" / "l" là ký hiệu đơn vị, không bao giờ được miễn.
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
		{"Bước 2 Khối Kế hoạch lập phiếu.", false},
		// ô bảng kế tiếp chứa chữ khác: chữ đơn vị KHÔNG mơ hồ ("Lít") mở đầu một nội dung, không phải đơn vị của số ở ô trước (R03i)
		{"| 2 | Lít dầu thừa ghi vào sổ kho |", false},
		// lượng thật vẫn bị chặn
		{"Định mức cấp phát cho máy xúc: **4.000** lít mỗi ca.", true},
		{"Định mức cấp phát cho máy xúc: 4.000, lít, mỗi ca.", true},
		{"Định mức cấp phát cho máy xúc: <b>4.000</b> lít.", true},
		{"Định mức cấp phát cho máy xúc: **40** lít.", true},
		{"Định mức cấp phát cho máy xúc: 4 lít mỗi ca.", true},
		{"Định mức cấp phát cho máy xúc: **4** lít mỗi ca.", true},
		{"Định mức cấp phát cho máy xúc: **5** mét khối mỗi ngày.", true},
		{"Định mức cấp phát cho máy xúc: **4.000** khối mỗi ca.", true}, // số có dáng lượng: "khối" là đơn vị
		// số một chữ số có trang trí: vẫn là lượng (RM-1 / F7)
		{"Định mức cấp phát cho máy xúc: **5** khối mỗi ca.", true},
		{"Định mức cấp phát cho máy xúc: **5** Khối mỗi ca.", true},
		{"Định mức cấp phát cho máy xúc: **5** L mỗi ca.", true},
		{"Định mức cấp phát cho máy xúc: `5` l mỗi ca.", true},
		{"Định mức cấp phát cho máy xúc: <b>5</b> khối.", true},
		{"| Dung tích bể | 5 | khối |", true},
		{"| Dung tích bể | 5 | L |", true},
		{"Định mức cấp phát cho máy xúc là 5, khối dầu mỗi ca.", true},
		{"Định mức cấp phát: 4.000 Khối Dầu mỗi ca.", true}, // số có dáng lượng: tiêu đề viết hoa không biến "Khối" thành tên riêng
	} {
		if got := len(ForbidUnitWords(c.text, words)) > 0; got != c.block {
			t.Errorf("%q: chặn = %v, muốn %v", c.text, got, c.block)
		}
	}
}
