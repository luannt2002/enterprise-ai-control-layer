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
