package normalization

import "testing"

// PeriodWordBefore: từ chỉ NGÀY đứng NGAY trước vị trí (chỉ cách khoảng trắng) là một phần của mốc ("ngày" của "ngày 13/10").
func TestPeriodWordBefore(t *testing.T) {
	for _, c := range []struct {
		raw  string
		at   int
		want string
	}{
		{"EX-03 ngày 13/10 cấp", len("EX-03 ngày "), "ngày"},
		{"hôm 15/9", len("hôm "), "hôm"},
		// chỉ từ chỉ NGÀY là một phần của mốc ngày: "tuần 15/9" là tuần chứa 15/9
		{"tuần  43", len("tuần  "), ""},
		{"tuần 15/9", len("tuần "), ""},
		{"mồng 5/10", len("mồng "), "mồng"},
		{"EX-03 13/10", len("EX-03 "), ""},
		{"phiếu 13/10", len("phiếu "), ""},
		{"ngày, 13/10", len("ngày, "), ""},
		{"13/10", 0, ""},
	} {
		ws, we, ok := PeriodWordBefore(c.raw, c.at)
		got := ""
		if ok {
			got = c.raw[ws:we]
		}
		if got != c.want {
			t.Errorf("PeriodWordBefore(%q, %d) = %q, muốn %q", c.raw, c.at, got, c.want)
		}
	}
}
