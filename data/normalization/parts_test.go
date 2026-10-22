package normalization

import "testing"

func TestParseLocaleNumber(t *testing.T) {
	cases := []struct {
		in        string
		want      string
		ambiguous bool
		ok        bool
	}{
		{"4000", "4000", false, true},
		{"-68", "-68", false, true},     // hiệu âm của phép tính
		{"−1.500", "-1500", true, true}, // dấu trừ Unicode
		{"-0", "0", false, true},
		{"--5", "", false, false},
		{"4.000", "4000", true, true},  // nghìn kiểu Việt; vẫn đánh dấu vì người quen kiểu Anh có thể muốn 4,0
		{"4,000", "4000", false, true}, // nghìn kiểu Anh
		{"1.602.000", "1602000", false, true},
		{"1,602,000", "1602000", false, true},
		{"0.35", "0.35", false, true},
		{"0,35", "0.35", false, true},
		{"0.350", "0.35", false, true}, // phần nguyên 0 → thập phân
		{"2,5", "2.5", false, true},
		{"1.602,5", "1602.5", false, true},
		{"1,602.5", "1602.5", false, true},
		{"1.60.000", "", false, false}, // nhóm nghìn sai
	}
	for _, c := range cases {
		got, amb, ok := parseLocaleNumber(c.in)
		if ok != c.ok || got != c.want || amb != c.ambiguous {
			t.Errorf("parseLocaleNumber(%q) = %q amb=%v ok=%v, muốn %q amb=%v ok=%v", c.in, got, amb, ok, c.want, c.ambiguous, c.ok)
		}
	}
}

func TestParseViNumberWords(t *testing.T) {
	cases := []struct {
		in       string
		want     string
		hasScale bool
		ok       bool
	}{
		{"bốn ngàn", "4000", true, true},
		{"ba trăm năm chục", "350", true, true},
		{"hai triệu rưỡi", "2500000", true, true},
		{"một trăm linh năm", "105", true, true},
		{"hai mươi mốt", "21", true, true},
		{"mười lăm", "15", true, true},
		{"ba trăm rưỡi", "350", true, true},
		{"nghìn rưỡi", "1500", true, true},
		{"bon nghin", "4000", true, true}, // gõ không dấu
		{"năm", "5", false, true},         // không có hàng — bên gọi đòi đơn vị theo sau
		{"máy", "", false, false},
		// D-OP-14 (chủ dự án chọn a, 22/10): chữ số đứng NGAY sau từ chỉ hàng ở cuối dãy là hàng kế dưới, như cách nói thường
		{"một nghìn hai", "1200", true, true},
		{"hai trăm năm", "250", true, true},
		{"ba nghìn tám", "3800", true, true},
		{"một triệu hai", "1200000", true, true},
		{"một tỷ hai", "1200000000", true, true},
		{"mười nghìn hai", "10200", true, true},
		{"ba nghìn tám rưỡi", "3850", true, true},
		{"một trăm hai rưỡi", "125", true, true},
		// dạng viết rõ hàng giữ nguyên
		{"một nghìn không trăm linh hai", "1002", true, true},
		{"một nghìn linh hai", "1002", true, true},
		{"một nghìn hai mươi", "1020", true, true},
		// "muoi" không dấu là "mươi" khi có chữ số đứng trước, là "mười" khi đứng đầu (trước đây chọn ngẫu nhiên theo thứ tự duyệt map)
		{"hai muoi", "20", true, true},
		{"muoi lam", "15", true, true},
		{"muoi", "10", true, true},
		// "một" và "mốt" cùng gấp thành "mot", cùng nghĩa: bản gấp vẫn có, không bị coi là xung đột
		{"mot tram linh nam", "105", true, true},
	}
	for _, c := range cases {
		toks := tokenize(c.in)
		v, _, hasScale, ok := parseViNumberWords(toks, 0)
		got := ""
		if ok {
			got = formatDecimal(v)
		}
		if ok != c.ok || got != c.want || hasScale != c.hasScale {
			t.Errorf("parseViNumberWords(%q) = %q scale=%v ok=%v, muốn %q scale=%v ok=%v", c.in, got, hasScale, ok, c.want, c.hasScale, c.ok)
		}
	}
}

// ADR-0009 Vòng 5: dãy số bằng chữ kết thúc bằng "<hàng> <một chữ số>", có thể kèm "rưỡi", là chưa đọc chắc (nói thường "một nghìn
// hai" = 1.200, bộ đọc cho 1.002); dạng viết rõ hàng ("trăm", "linh", "lẻ", "mươi") hay "rưỡi" ngay sau hàng thì không.
func TestAmbiguousNumberWords(t *testing.T) {
	for _, c := range []struct {
		in   string
		want bool
	}{
		{"một nghìn hai lít", true},
		{"ba nghìn tám", true},
		{"hai trăm năm", true},
		{"một tỷ hai lít", true},
		{"trăm hai", true},
		{"hai nghìn tư lít", true},
		{"mot nghin hai lit", true}, // gõ không dấu
		{"một nghìn hai trăm lít", false},
		{"một nghìn không trăm linh hai lít", false},
		{"một trăm lẻ một", false},
		{"hai nghìn rưỡi lít", false},
		{"ba nghìn tám trăm năm mươi lít", false},
		{"mười hai", false},
		{"hai mươi", false},
		{"bốn ngàn", false},
		{"3850 L", false},
		{"", false},
		// "rưỡi" sau "<hàng> <chữ số>" vẫn mơ hồ ("ba nghìn tám rưỡi" = 3.850); sau hàng tròn hay "trăm" thì rõ
		{"ba nghìn tám rưỡi lít", true},
		{"một trăm hai rưỡi", true},
		{"ba nghìn tám trăm rưỡi", false},
		{"ba trăm rưỡi", false},
		{"mười hai rưỡi", false},
		{"hai trăm mười rưỡi lít", false},
		{"ba nghìn lẻ rưỡi lít", false},
		// chữ số đứng sau "mươi" là hàng đơn vị bình thường; từ không phải số làm đứt dãy; dấu câu dính vào từ được bỏ
		{"hai mươi lăm", false},
		{"một trăm hai mươi lăm lít", false},
		{"ba nghìn lít hai", false},
		{"(một nghìn hai)", true},
		{"một nghìn hai.", true},
		// "linh" / "lẻ" đứng cuối dãy: "một nghìn hai lẻ" = hơn 1.200 một chút, "hai trăm lẻ" = hơn 200 — giá trị là xấp xỉ (soát
		// D-OP-14 F1: "một nghìn hai lẻ lít … đúng không?" từng đổi "Không." thành "Đúng.")
		{"một nghìn hai lẻ lít", true},
		{"một nghìn hai Linh lít", true},
		{"một nghìn hai linh rưỡi", true},
		{"hai trăm lẻ lít", true},
	} {
		if got := AmbiguousNumberWords(c.in); got != c.want {
			t.Errorf("AmbiguousNumberWords(%q) = %v, muốn %v", c.in, got, c.want)
		}
	}
}

func TestMatchAliasLuatDau(t *testing.T) {
	cases := []struct {
		s, alias string
		ok       bool
	}{
		{"dầu 4k", "dầu", true},
		{"dau 4k", "dầu", true},     // không dấu → so khớp đã bỏ dấu
		{"đầu tháng", "dầu", false}, // có dấu nhưng khác dấu
		{"DẦU", "dầu", true},        // chỉ khác hoa thường
		{"dầuxx", "dầu", false},     // không phải ranh giới từ
	}
	for _, c := range cases {
		_, ok := matchAliasAt(c.s, 0, c.alias, false)
		if ok != c.ok {
			t.Errorf("matchAliasAt(%q, %q) = %v, muốn %v", c.s, c.alias, ok, c.ok)
		}
	}
}

func TestToneKey(t *testing.T) {
	same := [][2]string{{"hoá", "hóa"}, {"thuý", "thúy"}, {"Hoà", "hòa"}, {"liệu", "liệu"}}
	for _, p := range same {
		if ToneKey(p[0]) != ToneKey(p[1]) {
			t.Errorf("ToneKey(%q) != ToneKey(%q): chỉ khác chỗ đặt dấu thanh phải cùng khoá", p[0], p[1])
		}
	}
	diff := [][2]string{{"bí", "bị"}, {"trường", "trưởng"}, {"bán", "bắn"}, {"ma", "mà"}, {"đo", "do"}}
	for _, p := range diff {
		if ToneKey(p[0]) == ToneKey(p[1]) {
			t.Errorf("ToneKey(%q) == ToneKey(%q): khác dấu thanh / dấu chữ phải khác khoá", p[0], p[1])
		}
	}
}
