package validation

import (
	"math/rand/v2"
	"strings"
	"testing"
)

// R-03 — lượng nghiệp vụ trong TÀI LIỆU lọt khi số và đơn vị bị chen một dấu trang trí (Markdown, HTML, ô bảng, dấu phẩy) hoặc đơn vị đứng trong ngoặc kèm nhãn
// ("(đơn vị: lít)"). Bản 6f23ff3 (và cha) để lọt cả hai lớp; nay lớp kiểm ĐỘC LẬP (ForbidUnitWords) bắt — lớp chuẩn hoá (ForbidDimensions) vẫn viết riêng, không đụng.
func TestLuongTaiLieuBiTrangTriVaNhanDonViBiChanOLopDocLap(t *testing.T) {
	e := packEngine(t)
	words := packUnitWords(e)
	for _, text := range []string{
		"Định mức cấp phát cho máy xúc: **4.000** lít mỗi ca.",
		"Định mức cấp phát cho máy xúc: `4.000` lít mỗi ca.",
		"Định mức cấp phát cho máy xúc: <b>4.000</b> lít mỗi ca.",
		"Định mức cấp phát cho máy xúc: *4.000* *lít* mỗi ca.",
		"Định mức cấp phát cho máy xúc: _4.000_ lít mỗi ca.",
		"Định mức cấp phát cho máy xúc: ~~4.000~~ lít mỗi ca.",
		"Định mức cấp phát cho máy xúc: <span class=\"n\">4.000</span> lít mỗi ca.",
		"Định mức cấp phát cho máy xúc là 4.000 (đơn vị: lít) mỗi ca.",
		"Định mức cấp phát cho máy xúc là 4.000 (ĐVT: lít) mỗi ca.",
		"Định mức cấp phát cho máy xúc là 4.000 (tính bằng lít) mỗi ca.",
		"Định mức cấp phát cho máy xúc là 4.000 (đơn vị tính: m3) mỗi ca.",
		"Định mức cấp phát cho máy xúc là 4.000 [ĐVT = L] mỗi ca.",
		"| Máy xúc | 4.000 | lít/ca |",
		"| Máy xúc | 4.000 | L |",
		"|Máy xúc|4.000|lít|",
		"Định mức cấp phát cho máy xúc: 4.000, lít, mỗi ca.",
		"Định mức cấp phát cho máy xúc: **5** mét khối mỗi ngày.",
		// đối chứng — các dạng đã bị chặn từ trước vẫn bị chặn
		"Định mức cấp phát cho máy xúc: **4.000 lít** mỗi ca.", "Định mức cấp phát cho máy xúc là 5.000 (lít/ngày).", "lít/ca: 4.000",
	} {
		if got := ForbidUnitWords(text, words); len(got) == 0 {
			t.Errorf("%q: lớp độc lập phải chặn lượng của tài liệu", text)
		}
	}
	// không chặn oan: số rồi tới một ô / chữ không phải đơn vị, hoặc trang trí quanh một số không có đơn vị
	for _, text := range []string{
		"| Bước | 2 | Khối Kế hoạch lập biên bản |",
		"| Bước | 2 | Lập biên bản đối soát |",
		"| Bước | 24 | giờ |",
		"**Bước 2** Lập biên bản đối soát trong vòng 24 giờ.",
		"Biên bản đối soát lập trong vòng **24** giờ.",
		"Gồm 3, Lập biên bản và ký.",
		"Đơn giá niêm yết (đồng/lít): 25.000 theo bảng giá.",
		"Phòng Kế hoạch thuộc khối 2 lập biên bản đối soát trong vòng 24 giờ.",
		"Mẫu BB-01, lít 2 liên, lập trong vòng 24 giờ.",
		"<b>24</b> giờ sau khi lập biên bản.",
	} {
		if got := ForbidUnitWords(text, words); len(got) != 0 {
			t.Errorf("%q: không phải lượng nghiệp vụ nhưng lớp độc lập chặn: %v", text, got)
		}
	}
}

// skipDecoration / cellEndsAfter / skipUnitLabel — các bước nhỏ của luật trên.
func TestSkipDecorationVaNhanDonVi(t *testing.T) {
	for _, c := range []struct {
		in, rest string
		cell     bool
	}{
		{"** lít", "lít", false}, {"` lít", "lít", false}, {"</b> lít", "lít", false}, {"*</b>* lít", "lít", false}, {", lít", "lít", false},
		{" | lít/ca |", "lít/ca |", true}, {"|lít", "lít", true}, {" lít", "lít", false}, {"", "", false},
		{"<3 lít", "<3 lít", false}, {"<< lít", "<< lít", false}, {"<không đóng", "<không đóng", false}, {"<3 sao> lít", "<3 sao> lít", false},
	} {
		rest, cell := skipDecoration(c.in)
		if rest != c.rest || cell != c.cell {
			t.Errorf("skipDecoration(%q) = %q, %v; muốn %q, %v", c.in, rest, cell, c.rest, c.cell)
		}
	}
	for after, want := range map[string]bool{"": true, " |": true, "/ca |": true, " / ca": false, "\n": true, " mỗi ca": false, "/ca": true, " khác |": false} {
		if got := cellEndsAfter(after); got != want {
			t.Errorf("cellEndsAfter(%q) = %v, muốn %v", after, got, want)
		}
	}
	for in, want := range map[string]string{"đơn vị: lít)": "lít)", "ĐVT = L)": "L)", "tính bằng lít)": "lít)", "đơn vị tính: m3)": "m3)", "lít dầu)": "lít dầu)", "đơn vịx: lít": "đơn vịx: lít", "đv: lít)": "lít)", "tính theo lít)": "lít)", "theo đơn vị lít)": "lít)", "ĐVT:lít)": "lít)"} {
		if got := skipUnitLabel(in); got != want {
			t.Errorf("skipUnitLabel(%q) = %q, muốn %q", in, got, want)
		}
	}
}

// Chuỗi ngẫu nhiên (số, đơn vị, dấu Markdown / HTML / ô bảng / ngoặc / nhãn đơn vị): ForbidUnitWords không panic và mọi vi phạm có Detail.
func TestForbidUnitWordsKhongPanicTrenChuoiNgauNhien(t *testing.T) {
	e := packEngine(t)
	words := packUnitWords(e)
	pieces := []string{
		"4.000", "5", "24", "3,5", "lít", "L", "khối", "m3", "mét khối", "giờ", "ca", "**", "*", "_", "`", "~", "<b>", "</b>", "<", ">", "<span class=\"x\">", "|", " | ", ",", ":",
		"(", ")", "[", "]", "đơn vị:", "ĐVT", "tính bằng", "=", "/", "/ca", " ", "  ", "\u00a0", "\u200b", "\n", "ℓ", "５", "é", "\u0301", "Khối", "Lập",
	}
	rng := rand.New(rand.NewPCG(20261021, 36))
	for n := 0; n < 30000; n++ {
		var b strings.Builder
		for k := rng.IntN(10) + 1; k > 0; k-- {
			b.WriteString(pieces[rng.IntN(len(pieces))])
			if rng.IntN(3) == 0 {
				b.WriteByte(' ')
			}
		}
		for _, v := range ForbidUnitWords(b.String(), words) {
			if v.Detail == "" || v.Code != "QUANTITY_FROM_DOCUMENT" {
				t.Fatalf("%q: vi phạm hỏng %+v", b.String(), v)
			}
		}
	}
}
