package validation

import (
	"strings"
	"testing"

	"github.com/luannt2002/enterprise-ai-control-layer/core/contracts"
	"github.com/luannt2002/enterprise-ai-control-layer/data/normalization"
)

// docEngine là lexicon mẫu giống cách pack cầu đường khai chiều thể tích — DỮ LIỆU test.
func docEngine(t *testing.T) *normalization.Engine {
	t.Helper()
	e, err := normalization.Compile(normalization.Lexicon{
		Units: []normalization.UnitDef{
			{Unit: "L", Kind: "quantity", Dimension: "volume", ToBase: "1", Aliases: []string{"L", "l", "lít", "lit", "litre", "liter"}},
			{Unit: "m3", Kind: "quantity", Dimension: "volume", ToBase: "1000", Aliases: []string{"m3", "m³", "khối"}},
			{Unit: "h", Kind: "quantity", Dimension: "duration", ToBase: "1", Aliases: []string{"h", "giờ", "tiếng"}},
		},
		Multipliers: []normalization.MultiplierDef{{Factor: "1000", Aliases: []string{"k"}}},
		Terms:       []normalization.TermDef{{Term: "diesel_fuel", Label: "dầu", Aliases: []string{"dầu"}, DefaultUnit: "L"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	return e
}

// docUnitWords: mọi cách viết đơn vị của chiều thể tích, dài trước (như Ask.factUnitWords dựng).
var docUnitWords = []string{"litre", "liter", "lít", "lit", "m³", "m3", "khối", "L", "l"}

// byLayer báo từng lớp kiểm có bắt câu không: bộ chuẩn hoá (ForbidDimensions) và bộ đếm độc lập
// (ForbidUnitWords). Hai lớp viết riêng, không gọi nhau — mỗi lớp phải tự bắt được.
func byLayer(e *normalization.Engine, text string) (normalizer, counter bool) {
	d := ForbidDimensions(e, contracts.Draft{Text: text}, map[string]bool{"volume": true}, now)
	c := ForbidUnitWords(text, docUnitWords)
	return len(d) > 0, len(c) > 0
}

// S2-2: lượng nghiệp vụ trong tài liệu (P4: số chính xác không lấy từ tài liệu) lọt CẢ HAI lớp khi
// SOP viết "5.000" với khoảng trắng lạ, chữ số toàn chiều rộng, đơn vị đứng trước / trong ngoặc, hay
// ký hiệu ℓ. Mỗi dạng phải bị CẢ HAI lớp bắt, độc lập.
func TestLuongTaiLieuDangVietLaBiChanOCaHaiLop(t *testing.T) {
	e := docEngine(t)
	for _, c := range []struct{ name, text string }{
		{"NBSP trước L", "Định mức cấp phát cho máy xúc là 5.000\u00a0L mỗi ngày."},
		{"ZWSP trước L", "Định mức cấp phát cho máy xúc là 5.000\u200bL mỗi ngày."},
		{"U+202F trước L", "Định mức cấp phát cho máy xúc là 5.000\u202fL mỗi ngày."},
		{"tab trước L", "Định mức cấp phát cho máy xúc là 5.000\tL mỗi ngày."},
		{"ZWSP giữa các chữ số", "Định mức cấp phát cho máy xúc là 5.0\u200b00 L mỗi ngày."},
		{"ZWSP giữa chữ và số (mã cũ đã bắt, không được mất)", "Định mức cấp phát cho máy xúc là\u200b5.000 L mỗi ngày."},
		{"ZWSP giữa đơn vị đứng trước và số", "Định mức cấp phát cho máy xúc: L\u200b5.000 mỗi ngày."},
		{"chữ số toàn chiều rộng", "Định mức cấp phát cho máy xúc là ５.０００ L mỗi ngày."},
		{"toàn chiều rộng cả đơn vị", "Định mức cấp phát cho máy xúc là ５.０００ Ｌ mỗi ngày."},
		{"đơn vị đứng trước, cách dấu cách", "Định mức cấp phát cho máy xúc: L 5.000 mỗi ngày."},
		{"đơn vị đứng trước, dấu hai chấm", "Định mức cấp phát cho máy xúc: lít: 5.000 mỗi ngày."},
		{"đơn vị đứng trước, đầu cột trong ngoặc", "Định mức cấp phát cho máy xúc (lít): 5.000 mỗi ngày."},
		{"đơn vị trong ngoặc: lít/ngày", "Định mức cấp phát cho máy xúc là 5.000 (lít/ngày)."},
		{"đơn vị trong ngoặc: L", "Định mức cấp phát cho máy xúc là 5.000 (L)."},
		{"ký hiệu ℓ", "Định mức cấp phát cho máy xúc là 5.000 ℓ mỗi ngày."},
		{"dấu tiếng Việt viết rời (NFD)", "Định mức cấp phát cho máy xúc là 5.000 li\u0301t mỗi ngày."},
	} {
		n, w := byLayer(e, c.text)
		if !n {
			t.Errorf("%s: bộ chuẩn hoá (ForbidDimensions) để lọt %q", c.name, c.text)
		}
		if !w {
			t.Errorf("%s: bộ đếm độc lập (ForbidUnitWords) để lọt %q", c.name, c.text)
		}
	}
}

// Các dạng ĐANG bị chặn đúng phải giữ: viết bằng chữ số thì cả hai lớp, viết bằng chữ / thuật ngữ
// ("năm nghìn lít", "dầu 5k") chỉ bộ chuẩn hoá hiểu được.
func TestLuongTaiLieuDangVietCuVanBiChan(t *testing.T) {
	e := docEngine(t)
	for _, text := range []string{
		"Định mức là 5.000 L.", "Định mức là 5000 lít.", "Định mức là 5.000L.", "Định mức là 5,000 L.",
		"Định mức là 5.000  lít.", "Định mức là 5.000 LÍT.", "Định mức là 5.000 Lít.", "Định mức là 5 khối.",
		"Định mức là 5.000 lít/ngày.", "Định mức là 5 m3.", "Định mức là 4 m³.",
	} {
		n, w := byLayer(e, text)
		if !n || !w {
			t.Errorf("%q: chuẩn hoá bắt=%v, bộ đếm bắt=%v — dạng này cả hai lớp đều phải bắt", text, n, w)
		}
	}
	for _, text := range []string{"Định mức là năm nghìn lít.", "Cấp dầu 5k mỗi ngày.", "Cấp 5k dầu mỗi ngày."} {
		if n, _ := byLayer(e, text); !n {
			t.Errorf("%q: bộ chuẩn hoá phải bắt", text)
		}
	}
}

// Số KHÔNG mang đơn vị lượng nghiệp vụ (số bước, thời hạn, ngày tháng) phải qua cả hai lớp: chặn
// nhầm là chặn câu trả lời quy trình hợp lệ.
func TestLuongTaiLieuSoKhongPhaiLuongVanQua(t *testing.T) {
	e := docEngine(t)
	for _, text := range []string{
		"Lập biên bản trong vòng 24 giờ.",
		"Xác nhận trong 2 ngày làm việc.",
		"1. Lập biên bản trong vòng 24 giờ.",
		"Bước 3 kiểm đối soát và bước 850 lưu biên bản.",
		"Bước 3 (xem phụ lục) lập biên bản.",
		"Bước 3 (lập biên bản trong 24 giờ).",
		"Bước 3\u00a0kiểm đối soát, bước 4 ký.",
		"Bước ３ kiểm đối soát, bước ４ ký.",
		"Gửi email 3 lần, mỗi lần cách nhau 2 giờ.",
		"Điều 5. Khối lượng công việc do tổ trưởng xác nhận.",
		"Ngày 19/10/2026 lập biên bản.",
		"Thời hạn: 24 giờ (1 ngày).",
	} {
		if n, w := byLayer(e, text); n || w {
			t.Errorf("%q: chặn oan (chuẩn hoá=%v, bộ đếm=%v)", text, n, w)
		}
	}
}

// Ký tự vô hình KHÔNG được ghép hai số thành một mà bước kiểm không thấy: "3<ZWSP>850" người đọc
// thấy 3850, nên bộ chuẩn hoá phải đọc là MỘT số 3850 (không có bằng chứng → bị chặn), không phải hai
// số 3 và 850 (đều có bằng chứng).
func TestFidelityKyTuVoHinhKhongGhepSo(t *testing.T) {
	e := testEngine(t)
	pkg := contracts.ContextPackage{Evidence: []contracts.Evidence{{ID: "SOP#0",
		Values: []contracts.Value{{Kind: contracts.ValueNumber, Number: "3"}, {Kind: contracts.ValueNumber, Number: "850"}}}}}
	v := Fidelity(e, contracts.Draft{Text: "bước 3\u200b850 lưu biên bản"}, pkg, nil, now)
	if len(v) != 1 || v[0].Code != "number_without_evidence" || !strings.Contains(v[0].Detail, "3850") {
		t.Errorf("số 3850 người đọc thấy phải bị bắt là số không có bằng chứng, được %+v", v)
	}
}
