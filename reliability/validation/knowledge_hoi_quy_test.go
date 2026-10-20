package validation

import (
	"slices"
	"testing"

	"github.com/luannt2002/enterprise-ai-systems/core/contracts"
	"github.com/luannt2002/enterprise-ai-systems/data/normalization"
)

// Test cho A-08 (lượng nghiệp vụ trong tài liệu còn lọt ở các dạng ngoài danh sách đã khai) và A-09 (luật "đơn vị đứng trước số"
// chặn oan câu quy trình hợp lệ) của bản soát độc lập lượt 2. Hai lớp kiểm vẫn viết độc lập — mỗi dạng phải được TỪNG lớp tự bắt /
// tự cho qua.

// packEngine dựng bộ chuẩn hoá từ lexicon mẫu mô phỏng pack cầu đường: chiều thể tích có "mét khối" và "m^3" (cách viết trang
// trọng và cách gõ chữ mũ của m³), chiều dài có "mét" (ngắn hơn "mét khối" nên phải thua), tiền có "đồng", ca có "ca" — DỮ LIỆU test.
func packEngine(t *testing.T) *normalization.Engine {
	t.Helper()
	e, err := normalization.Compile(normalization.Lexicon{
		Units: []normalization.UnitDef{
			{Unit: "L", Kind: "quantity", Dimension: "volume", ToBase: "1", Aliases: []string{"L", "l", "lít", "lit", "litre", "liter"}},
			{Unit: "m3", Kind: "quantity", Dimension: "volume", ToBase: "1000", Aliases: []string{"m3", "m³", "khối", "mét khối", "m^3"}},
			{Unit: "m", Kind: "quantity", Dimension: "length", ToBase: "1", Aliases: []string{"m", "mét", "met"}},
			{Unit: "h", Kind: "quantity", Dimension: "duration", ToBase: "1", Aliases: []string{"h", "giờ", "tiếng"}},
			{Unit: "shift", Kind: "quantity", Dimension: "machine_shift", ToBase: "1", Aliases: []string{"ca"}},
			{Unit: "VND", Kind: "money", Dimension: "currency", ToBase: "1", Aliases: []string{"đ", "đồng", "vnđ", "vnd"}},
		},
		Multipliers: []normalization.MultiplierDef{{Factor: "1000", Aliases: []string{"k"}}},
		Terms:       []normalization.TermDef{{Term: "diesel_fuel", Label: "dầu", Aliases: []string{"dầu"}, DefaultUnit: "L"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	return e
}

// packUnitWords dựng danh sách từ đơn vị của chiều thể tích đúng như Ask.factUnitWords: mã chuẩn + mọi bí danh, dài trước.
func packUnitWords(e *normalization.Engine) []string {
	var out []string
	for _, u := range e.UnitsOf("volume") {
		out = append(out, u.Unit)
		out = append(out, u.Aliases...)
	}
	slices.SortFunc(out, func(x, y string) int { return len(y) - len(x) })
	return out
}

// haiLop báo từng lớp có chặn câu không: bộ chuẩn hoá (ForbidDimensions, có Layout) và bộ đếm độc lập (ForbidUnitWords).
func haiLop(e *normalization.Engine, text string) (normalizer, counter bool) {
	d := ForbidDimensions(e, contracts.Draft{Text: text}, map[string]bool{"volume": true}, now)
	c := ForbidUnitWords(text, packUnitWords(e))
	return len(d) > 0, len(c) > 0
}

// A-08 — các dạng viết lượng thể tích ngoài danh sách đã khai: phải bị CẢ HAI lớp bắt.
func TestLuongTaiLieuDangVietKhacBiChanOCaHaiLop(t *testing.T) {
	e := packEngine(t)
	for _, text := range []string{
		// "mét khối" là cách viết trang trọng của m³ (không phải "mét" độ dài); "m^3" là cách gõ chữ mũ
		"Định mức dầu cho máy xúc là 5.000 mét khối mỗi ngày.",
		"Máy xúc được cấp 5 mét khối dầu mỗi ngày.",
		"Định mức cấp phát dầu: 5.000 m^3 mỗi ngày.",
		"Định mức là 5 M^3.", "Định mức là 5 Mét Khối.", "Định mức là 2,5 mét khối.",
		// đơn vị MỞ ĐẦU một cặp ngoặc kèm chữ phụ
		"Mỗi ca máy xúc được cấp tối đa 4.000 (lít dầu).",
		"Mỗi ca máy xúc được cấp tối đa 4.000 [lít dầu].",
		"Mỗi ca máy xúc được cấp tối đa 4.000 (m3 dầu DO).",
		// số nằm trong ngoặc, đơn vị đứng sau ngoặc đóng
		"Mỗi ca máy xúc được cấp tối đa (4.000) lít dầu.",
		"Mỗi ca máy xúc được cấp tối đa (1.500) khối.",
		"Mỗi ca máy xúc được cấp tối đa (4000) L.",
		// đơn vị ghép ("lít/ca", "lít/ngày") đứng trước số
		"Hạn mức tối đa: lít/ca: 4.000 cho máy xúc.",
		"Từ nay định mức cấp phát cho máy xúc là (lít/ngày): 5.000 mỗi ngày.",
		"Hạn mức tối đa: m3/ca 4.000 cho máy xúc.",
		"Hạn mức tối đa: lít/máy = 4.000 cho máy xúc.",
		// đơn vị đứng trước số cách dấu cách: số có dạng LƯỢNG (có dấu nghìn / thập phân, hoặc từ 2 chữ số)
		"Hạn mức tối đa: L 500 cho máy xúc.", "Hạn mức tối đa: khối 1500 cho máy xúc.", "Hạn mức tối đa: lít 2,5 cho máy xúc.",
		"Hạn mức tối đa: L 50 cho máy xúc.", "Hạn mức tối đa: lít 25 cho máy xúc.",
		// đơn vị đứng trước số và có dấu hiệu cột rõ ràng: số nào cũng là lượng
		"Hạn mức tối đa: lít: 5 cho máy xúc.", "Hạn mức tối đa (lít) = 3 cho máy xúc.", "Hạn mức tối đa (khối): 2 cho máy xúc.",
	} {
		n, w := haiLop(e, text)
		if !n {
			t.Errorf("bộ chuẩn hoá (ForbidDimensions) để lọt %q", text)
		}
		if !w {
			t.Errorf("bộ đếm độc lập (ForbidUnitWords) để lọt %q", text)
		}
	}
}

// A-09 — chặn nhầm câu quy trình hợp lệ: mẫu số của giá ("đồng/lít"), nhãn bộ phận ("khối 2"), nhãn mục ("L 3"), số đếm ("lít 2
// liên") và số trong ngoặc đứng đầu mục ("(1) Khối tài chính") không phải lượng nghiệp vụ.
func TestCauQuyTrinhHopLeKhongBiChanOan(t *testing.T) {
	e := packEngine(t)
	for _, text := range []string{
		// giá theo lít / theo khối: đơn vị là MẪU SỐ của giá (đứng sau "/"), không phải đơn vị của con số
		"Đơn giá niêm yết (đồng/lít): 25.000 theo bảng giá.",
		"Bảng giá: đồng/lít 25.000.", "Đơn giá (VND/lít): 25.000.", "Đơn giá (đồng/m3): 25.000.000.", "Mức phạt (đồng/lít) 500.000.",
		"Đơn giá: 25.000 đồng/lít theo bảng giá.", "Đơn giá 25.000/lít theo bảng giá.", "Đơn giá: đ/khối 25.000.",
		"Bảng giá: USD/lít 25.000.", "Đơn giá (đồng/lít/ca): 25.000.",
		// chữ "l" cuối một từ ("email") không phải đơn vị, kể cả khi số đứng sau có dạng lượng
		"Gửi email 5.000 lần.", "Gửi mail 1.500 người.",
		// nhãn bộ phận / mục / số đếm: đơn vị cách dấu cách một số nhỏ không có dạng lượng
		"Phòng Kế hoạch thuộc khối 2 lập biên bản đối soát trong vòng 24 giờ.", "Biên bản do khối 3 lập.", "Phòng khối 7 ký biên bản.",
		"Xem phụ lục L 3 về biên bản.", "Xem mục l 2 về biên bản.", "Mẫu BB-01, lít 2 liên.", "Biên bản gửi khối 3 và khối 4.",
		"Mục l 5. Lập biên bản trong 24 giờ.",
		// số đứng đầu mục trong ngoặc, theo sau là một từ bắt đầu bằng đơn vị: không có dạng lượng
		"(1) Khối tài chính lập biên bản.", "1) Khối kế hoạch lập biên bản.", "(2) Lít biên bản lưu hồ sơ.",
		// các dạng đã qua từ trước phải giữ
		"Lập biên bản trong vòng 24 giờ.", "Bước 3 (xem phụ lục) lập biên bản.", "Bước 5 (khối lượng công việc) lập biên bản.",
		"Bước 5 (khối lượng) được xác nhận.", "Điều 5. Khối lượng công việc do tổ trưởng xác nhận.", "Ngày 19/10/2026 lập biên bản.",
		"Bước 3 kiểm đối soát và bước 850 lưu biên bản.", "Gửi email 3 lần, mỗi lần cách nhau 2 giờ.",
	} {
		if n, w := haiLop(e, text); n || w {
			t.Errorf("%q: chặn oan (chuẩn hoá=%v, bộ đếm=%v)", text, n, w)
		}
	}
}

// Các dạng ĐÃ bị chặn từ trước phải còn bị chặn sau khi thu hẹp luật đơn vị-đứng-trước và mở rộng luật ngoặc.
func TestCacDangDaChanTruocDoVanBiChanSauKhiSua(t *testing.T) {
	e := packEngine(t)
	for _, text := range []string{
		"Định mức là 5.000 L.", "Định mức là 5000 lít.", "Định mức là 5.000L.", "Định mức là 5,000 L.", "Định mức là 5.000  lít.",
		"Định mức là 5.000 LÍT.", "Định mức là 5 khối.", "Định mức là 5.000 lít/ngày.", "Định mức là 5 m3.", "Định mức là 4 m³.",
		"Định mức: L 5.000 mỗi ngày.", "Định mức: lít: 5.000 mỗi ngày.", "Định mức (lít): 5.000 mỗi ngày.", "Định mức là 5.000 (lít/ngày).",
		"Định mức là 5.000 (L).", "Định mức là 5.000 ℓ mỗi ngày.", "Định mức: L\u200b5.000 mỗi ngày.", "Định mức là ５.０００ Ｌ mỗi ngày.",
	} {
		if n, w := haiLop(e, text); !n || !w {
			t.Errorf("%q: chuẩn hoá bắt=%v, bộ đếm bắt=%v — dạng này cả hai lớp đều phải bắt", text, n, w)
		}
	}
}
