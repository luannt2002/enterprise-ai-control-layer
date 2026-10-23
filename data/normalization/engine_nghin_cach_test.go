package normalization

import (
	"testing"

	"github.com/luannt2002/enterprise-ai-control-layer/core/contracts"
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

// Nhóm nghìn mà con số KHÔNG đứng đầu văn bản là CHƯA PHÂN ĐỊNH (D-OP-9, phương án a'): "bể 2 500 lít" là 2.500 L, hay bể 2 với 500 L?
// Hệ vẫn gộp thành MỘT lượng — không để phần đầu thành số mồ côi ("12 000 km" không được thành "12" + "000 km") — nhưng gắn cờ, để tầng
// sau hỏi lại / cách ly thay vì tin một cách đọc. Con số đứng đầu văn bản thì chắc chắn, kể cả khi trước nó có khoảng trắng hay ký tự
// định dạng vô hình (ZWSP, BOM…): người đọc vẫn thấy con số đứng đầu.
func TestNhomNghinChuaPhanDinhKhiKhongDungDauVanBan(t *testing.T) {
	e := mustEngine(t)
	// ký tự vô hình viết bằng rune: chuỗi thoát \u trong lệnh ghi tệp bị công cụ giải mã thành ký tự thật
	zwsp, bom, softHyphen := string(rune(0x200B)), string(rune(0xFEFF)), string(rune(0x00AD))
	for _, c := range []struct {
		in        string
		uncertain bool
	}{
		// đứng đầu văn bản: chắc chắn
		{"3 850 lít", false},
		{"3 850 lít dầu cho máy 03", false},
		{"  3 850 lít", false},
		{"\t\n3 850 lít", false},
		{zwsp + "3 850 lít", false},
		{bom + "3 850 lít", false},
		{softHyphen + "3 850 lít", false},
		{zwsp + " " + bom + " 3 850 lít", false},
		// không đứng đầu: chưa phân định
		{"cấp 3 850 lít", true},
		{"bể 2 500 lít", true},
		{"bể #2 500 lít", true},
		{"bể: 2 500 lít", true},
		{"12 3 850 lít", true},
		{"máy 03 đổ 1 200 lít", true},
		{zwsp + "cấp 3 850 lít", true}, // ký tự vô hình không biến chữ đứng trước thành "đầu văn bản"
	} {
		in := e.Normalize(c.in, Options{Now: now})
		var groups []contracts.Span
		for _, sp := range in.Spans {
			if SpaceGroup(sp) {
				groups = append(groups, sp)
			}
		}
		if len(groups) != 1 {
			t.Errorf("%q: muốn đúng một span nhóm nghìn, được %d (%v)", c.in, len(groups), in.Spans)
			continue
		}
		sp := groups[0]
		if got := UncertainGroup(sp); got != c.uncertain {
			t.Errorf("%q: UncertainGroup = %v, muốn %v (rule %q)", c.in, got, c.uncertain, sp.Rule)
		}
		if c.uncertain != (sp.Confidence < 1) {
			t.Errorf("%q: độ tin cậy %.2f, muốn < 1 khi và chỉ khi chưa phân định (%v)", c.in, sp.Confidence, c.uncertain)
		}
		if sp.Kind != contracts.ValueQuantity || sp.Value.Unit != "L" {
			t.Errorf("%q: cờ không được đổi kiểu / đơn vị của span: %+v", c.in, sp)
		}
	}
}

// Cờ chưa phân định KHÔNG làm đổi giá trị hay cờ ước lượng của span, và nguồn có cấu trúc (Options.Cell: văn bản là MỘT ô lượng của bảng /
// ERP / phiếu) không bao giờ chưa phân định: ô đã tách sẵn lượng khỏi chữ chung quanh, nên "khoảng 3 850 L" ở ô là một lượng ước lượng, không phải
// "số hiệu 3 rồi 850 L" (soát bản thử (a') đêm 21/10, S3).
func TestNhomNghinOCoCauTrucLuonChacChan(t *testing.T) {
	e := mustEngine(t)
	for _, c := range []struct {
		in              string
		approx          bool
		uncertainInText bool // ở văn bản thường (không phải ô)
	}{
		{"3 850 L", false, false},
		{"khoảng 3 850 L", true, true},
		{"tầm 3 850 L", true, true},
		{"cỡ 1 200 000 L", true, true},
	} {
		cell := e.Normalize(c.in, Options{Now: now, Cell: true}).SpansOf(contracts.ValueQuantity)
		text := e.Normalize(c.in, Options{Now: now}).SpansOf(contracts.ValueQuantity)
		if len(cell) != 1 || len(text) != 1 {
			t.Fatalf("%q: muốn một lượng ở cả hai chế độ, ô %d, văn bản %d", c.in, len(cell), len(text))
		}
		if UncertainGroup(cell[0]) || cell[0].Confidence != 1 {
			t.Errorf("%q: ô có cấu trúc không được chưa phân định: %+v", c.in, cell[0])
		}
		if UncertainGroup(text[0]) != c.uncertainInText {
			t.Errorf("%q: ở văn bản thường UncertainGroup phải là %v: %+v", c.in, c.uncertainInText, text[0])
		}
		if cell[0].Value != text[0].Value || cell[0].Value.Approx != c.approx {
			t.Errorf("%q: giá trị ô %+v khác giá trị văn bản %+v, hoặc cờ ước lượng không phải %v", c.in, cell[0].Value, text[0].Value, c.approx)
		}
	}
}
