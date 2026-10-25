package normalization

import (
	"math/rand/v2"
	"slices"
	"strings"
	"testing"
)

// Test cho các HỒI QUY R-01, R-05, R-07 của bản soát độc lập vòng sau (do bản sửa 6f23ff3 gây ra) ở tầng chuẩn hoá. Hôm nay (now) = 19/10/2026.

// R-01 — "tháng M năm Y" đọc cả năm viết ra: trước đây chỉ "tháng M/Y" mang năm, "tháng 9 năm 2025" thành tháng 9 của năm NAY (im lặng sai năm).
func TestThangNamVietChuMangDuNam(t *testing.T) {
	e := mustEngine(t)
	for _, c := range []struct{ in, raw, want string }{
		{"Dự án A tháng 9 năm 2025", "tháng 9 năm 2025", "2025-09-01/2025-09-30"},
		{"tháng 10 năm 2025", "tháng 10 năm 2025", "2025-10-01/2025-10-31"},
		{"tháng 2 năm 2028", "tháng 2 năm 2028", "2028-02-01/2028-02-29"}, // năm nhuận
		{"tháng 10 năm 2026", "tháng 10 năm 2026", "2026-10-01/2026-10-31"},
		{"tháng 9/2025", "tháng 9/2025", "2025-09-01/2025-09-30"},
		{"tháng 9", "tháng 9", "2026-09-01/2026-09-30"},   // không ghi năm: lần gần nhất không ở tương lai
		{"tháng 11", "tháng 11", "2025-11-01/2025-11-30"}, // tháng 11 hỏi vào tháng 10 là năm trước
	} {
		raws, values := dateSpans(e.Normalize(c.in, Options{Now: now}))
		if !slices.Equal(raws, []string{c.raw}) || !slices.Equal(values, []string{c.want}) {
			t.Errorf("%q: span kỳ = %q → %q, muốn %q → %q", c.in, raws, values, c.raw, c.want)
		}
	}
	// "năm nay" không phải năm viết ra: tháng vẫn suy năm, "năm nay" là span riêng; tháng 13 và "năm" không kèm năm không đọc thành tháng
	for _, c := range []struct {
		in   string
		want []string
	}{
		{"tháng 10 năm nay", []string{"tháng 10", "năm nay"}},
		{"tháng 13 năm 2025", nil},
		{"tháng 9 năm ngoái", []string{"tháng 9"}},
	} {
		if raws, _ := dateSpans(e.Normalize(c.in, Options{Now: now})); !slices.Equal(raws, c.want) {
			t.Errorf("%q: span kỳ = %q, muốn %q", c.in, raws, c.want)
		}
	}
}

// R-05 — "mỗi ngày 3-4 chuyến" nói TỐC ĐỘ, không phải ngày 03/04: chữ "ngày" đứng sau mỗi / hàng / một / từng / mọi / cứ không báo một ngày cụ thể.
func TestMoiNgayTocDoKhongBaoNgayCuThe(t *testing.T) {
	e := mustEngine(t)
	for _, in := range []string{
		"mỗi ngày 3-4 chuyến", "mỗi ngày 10-12 xe tải", "mỗi ngày 2.5 chuyến", "hàng ngày 3-4 chuyến", "một ngày 3-4 chuyến", "từng ngày 3-4 chuyến",
		"mọi ngày 3-4 chuyến", "cứ ngày 3-4 chuyến", "Mỗi ngày 3-4 chuyến", "moi ngay 3-4 chuyen",
	} {
		got := e.Normalize(in, Options{Now: now})
		if raws, _ := dateSpans(got); len(raws) != 0 || len(got.InvalidDates) != 0 {
			t.Errorf("%q: là tốc độ nhưng span ngày = %q, ngày hỏng = %q", in, raws, got.InvalidDates)
		}
	}
	// đối chứng: "ngày" không có từ tốc độ đứng trước thì vẫn đọc được (A-05); "hôm qua ngày 3-4" cũng vậy
	for _, c := range []struct{ in, want string }{
		{"ngày 3-4", "2026-04-03"}, {"ngày 15-09", "2026-09-15"}, {"ngày 15.09", "2026-09-15"}, {"hôm qua ngày 15-09", "2026-09-15"},
		{"EX-03 ngày 3-4 chuyến", "2026-04-03"}, // không có từ tốc độ: đúng như A-05 (đánh đổi đã khai: "ngày" + khoảng đếm)
	} {
		_, values := dateSpans(e.Normalize(c.in, Options{Now: now}))
		if !slices.Contains(values, c.want) {
			t.Errorf("%q: span ngày = %q, muốn có %q", c.in, values, c.want)
		}
	}
}

// R-07 — dạng ngày: đảo vế tháng/ngày ("10/19" là 19/10 viết kiểu tháng/ngày) và ngày lố một chút ("32/10" … "39/10") là ngày hỏng — như bản cha. Tỉ lệ / số
// phiếu ("24/24", "60/40", "12/34", "19/13") không có dạng đó nên không bị ghi (đánh đổi đã khai ở dateLike).
func TestNgayDaoVeVaLoMotChutDuocGhiLaHong(t *testing.T) {
	e := mustEngine(t)
	for _, c := range []struct {
		in   string
		want []string
	}{
		{"EX-03 10/19", []string{"10/19"}},
		{"EX-03 10/19/26", []string{"10/19/26"}},
		{"EX-03 11/30", []string{"11/30"}},
		{"EX-03 1/13", []string{"1/13"}},
		{"EX-03 12/31", []string{"12/31"}},
		{"EX-03 32/10", []string{"32/10"}},
		{"EX-03 35/10", []string{"35/10"}},
		{"EX-03 39/12", []string{"39/12"}},
		{"EX-03 hôm qua 10/19", []string{"10/19"}},
		// vẫn là ngày hỏng vì lịch (A-06) và ngày hợp lệ
		{"31/2", []string{"31/2"}},
	} {
		if got := e.Normalize(c.in, Options{Now: now}).InvalidDates; !slices.Equal(got, c.want) {
			t.Errorf("%q: InvalidDates = %q, muốn %q", c.in, got, c.want)
		}
	}
	for _, in := range []string{
		"24/24", "60/40", "12/34", "30/60", "45/50", "99/99", "19/13", "24/32", "40/10", "45/12", "12/32", "1/32", "100/10", "12/34/56",
		"chạy 24/24", "Phiếu 12/34", "tỷ lệ 60/40",
		"10/12", "19/10", "24/7", "12/12", // ngày hợp lệ
	} {
		if got := e.Normalize(in, Options{Now: now}).InvalidDates; len(got) != 0 {
			t.Errorf("%q: không phải ngày hỏng nhưng InvalidDates = %q", in, got)
		}
	}
}

// dateLike — bảng chân trị của các nhánh (ngày d, tháng m, số chữ số của năm, có chữ "ngày" đứng trước).
func TestDateLikeBangChanTri(t *testing.T) {
	for _, c := range []struct {
		d, m, yd int
		ctx      bool
		want     bool
		note     string
	}{
		{31, 2, 0, false, true, "sai vì lịch"},
		{1, 12, 0, false, true, "hợp lệ trong khoảng"},
		{10, 19, 0, false, true, "đảo vế: ngày 1–12, tháng 13–31"},
		{12, 31, 0, false, true, "đảo vế, biên trên"},
		{1, 13, 0, false, true, "đảo vế, biên dưới"},
		{13, 19, 0, false, false, "ngày > 12 nên không đảo vế được"},
		{12, 32, 0, false, false, "phần sau > 31"},
		{32, 10, 0, false, true, "ngày lố một chút"},
		{39, 12, 0, false, true, "ngày lố, biên trên"},
		{40, 10, 0, false, false, "lố quá xa"},
		{31, 13, 0, false, false, "19/13-kiểu: tháng 13 không có chữ ngày"},
		{19, 13, 0, true, true, "có chữ ngày thì lỏng hơn"},
		{24, 24, 0, false, false, "tỉ lệ"},
		{24, 24, 0, true, true, "có chữ ngày thì lỏng hơn"},
		{15, 13, 4, false, true, "năm bốn chữ số"},
		{60, 40, 0, false, false, "tỉ lệ"},
		{100, 10, 0, false, false, "ba chữ số"},
	} {
		if got := dateLike(c.d, c.m, c.yd, c.ctx); got != c.want {
			t.Errorf("dateLike(%d, %d, %d, %v) = %v, muốn %v (%s)", c.d, c.m, c.yd, c.ctx, got, c.want, c.note)
		}
	}
}

// PeriodQualifier — con số đứng cạnh một từ chỉ kỳ mà bộ chuẩn hoá không đọc: mốc thời gian viết bằng chữ, hay tần suất / nhãn.
func TestPeriodQualifier(t *testing.T) {
	type q = Qualifier
	for _, c := range []struct {
		in   string
		num  string // con số cần xét (lần xuất hiện đầu tiên)
		want *q
	}{
		{"EX-03 tuần 43 được cấp", "43", &q{QualifierWeek, 43, "tuần 43", [2]int{6, 12}}},
		{"Tuần 43 EX-03", "43", &q{QualifierWeek, 43, "Tuần 43", [2]int{0, 6}}},
		{"quý 3", "3", &q{QualifierQuarter, 3, "quý 3", [2]int{0, 4}}},
		{"năm 2025", "2025", &q{QualifierYear, 2025, "năm 2025", [2]int{0, 4}}},
		{"ngày 12", "12", &q{QualifierDay, 12, "ngày 12", [2]int{0, 5}}},
		{"mồng 5", "5", &q{QualifierDay, 5, "mồng 5", [2]int{0, 6}}},
		{"hôm 15", "15", &q{QualifierDay, 15, "hôm 15", [2]int{0, 4}}},
		{"mùng 5", "5", &q{QualifierDay, 5, "mùng 5", [2]int{0, 5}}},
		{"tháng 13", "13", &q{QualifierMonth, 13, "tháng 13", [2]int{0, 6}}},
		{"7 ngày qua", "7", &q{QualifierSpan, 7, "7 ngày", [2]int{2, 7}}},
		{"2 tuần trước", "2", &q{QualifierSpan, 2, "2 tuần", [2]int{2, 8}}},
		{"3 tháng gần đây", "3", &q{QualifierSpan, 3, "3 tháng", [2]int{2, 8}}},
		{"2 quý qua", "2", &q{QualifierSpan, 2, "2 quý", [2]int{2, 6}}},
		{"3 năm qua", "3", &q{QualifierSpan, 3, "3 năm", [2]int{2, 6}}},
		{"15 tháng 9", "15", &q{QualifierDay, 15, "15 tháng 9", [2]int{3, 9}}},
		{"hôm 15 tháng 9", "15", &q{QualifierDay, 15, "hôm 15", [2]int{0, 4}}},
		{"tuan 41", "41", &q{QualifierWeek, 41, "tuan 41", [2]int{0, 4}}}, // không dấu
		// không phải mốc: tần suất, nhãn, số đếm, số phiếu rồi tháng
		{"mỗi ngày 3 chuyến", "3", nil},
		{"hàng tuần 2 lần", "2", nil},
		{"mỗi 2 ngày", "2", nil},
		{"cứ 2 ngày một lần", "2", nil},
		{"phiếu 912 tháng 9", "912", nil}, // 912 không phải ngày trong tháng
		{"2 ngày 15/09", "2", nil},        // "ngày" mở đầu một mốc khác
		{"ca 2", "2", nil},
		{"3.850 L", "3.850", nil},
		{"năm 12345", "12345", nil}, // quá bốn chữ số: không phải năm
		{"tuần 1.5", "1.5", nil},    // có dấu thập phân: không phải số nguyên của mốc
		{"ngày 2,5 tấn", "2,5", nil},
		{"3850 tuần này", "3850", nil}, // số lớn trước một mốc: con số, không đếm tuần
		{"3850 tháng 9", "3850", nil},
		{"EX-03", "03", nil},
	} {
		start := indexOf(c.in, c.num)
		got, ok := PeriodQualifier(c.in, start, start+len(c.num))
		if (c.want == nil) == ok || (ok && got != *c.want) {
			t.Errorf("PeriodQualifier(%q, %q) = %+v, %v; muốn %+v", c.in, c.num, got, ok, c.want)
		}
	}
}

// indexOf trả vị trí byte của lần xuất hiện đầu tiên của sub trong s (hoặc 0 khi không có; test tự kiểm bằng kết quả).
func indexOf(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return 0
}

// LabelContext — nhãn / số đếm / tần suất vô hại cho câu xác nhận một lượng; mọi thứ khác vẫn là con số.
func TestLabelContext(t *testing.T) {
	e := mustEngine(t)
	for _, c := range []struct {
		in, num string
		want    bool
	}{
		{"theo phiếu số 912", "912", true}, {"ca 2", "2", true}, {"Đội 3 báo", "3", true}, {"xe số 7", "7", true}, {"khối 2", "2", true}, {"L 5", "5", true},
		{"lô 7", "7", true}, {"mỗi ngày 3 bể", "3", true}, {"hàng tuần 2 bể", "2", true}, {"đổ 3-4 chuyến", "3", true}, {"đổ 3–4 chuyến", "3", true}, {"đổ 3-4 lần 4000", "4000", false}, {"cho 2 máy", "2", true}, {"đổ 3 lần", "3", true}, {"mỗi ngày 3-4 chuyến", "3", true}, {"mỗi ngày 3-4 chuyến", "4", true},
		{"mỗi ngày 10-12 xe tải", "10", true}, {"hàng tuần 2 lần", "2", true}, {"trong 2 phút", "2", true}, {"1 xe", "1", true},
		// không phải nhãn: số đứng sau từ khoá số liệu, sau chữ số + đơn vị, hay đứng một mình
		{"được cấp 3850", "3850", false}, {"nhận 4000", "4000", false}, {"3850 L 19 tháng 10", "19", false}, {"3-4 chuyến 4000", "4000", false},
		{"EX-03 hôm nay 3850", "3850", false}, {"3850", "3850", false}, {"tuần 43", "43", false}, {"năm 2025", "2025", false},
	} {
		start := indexOf(c.in, c.num)
		if c.in == "mỗi ngày 3-4 chuyến" && c.num == "4" {
			start = indexOf(c.in, "-4") + 1
		}
		if got := e.LabelContext(c.in, start, start+len(c.num)); got != c.want {
			t.Errorf("LabelContext(%q, %q) = %v, muốn %v", c.in, c.num, got, c.want)
		}
	}
}

// DashDateAt — "19-10", "19-10-26" là MỘT ngày thiếu chữ "ngày" mà bộ chuẩn hoá không đọc (nên hỏi lại); khoảng giờ / đếm, mã, ngày > 31, tháng > 12
// không phải.
func TestDashDateAt(t *testing.T) {
	e := mustEngine(t)
	for _, c := range []struct {
		in   string
		want string // cụm được báo ("" = không phải)
		n    int
	}{
		{"EX-03 19-10 được cấp", "19-10", 2},
		{"EX-03 19-10-26 được cấp", "19-10-26", 3},
		{"EX-03 19–10 được cấp", "19–10", 2},
		{"được cấp 3850 L 19-10", "19-10", 2},
		{"EX-03 19-10-2026", "", 0},      // đã là ngày đọc được: không còn là số trần
		{"chạy 10-12 giờ", "", 0},        // 12 mang đơn vị: lượng
		{"mỗi ngày 10-12 xe tải", "", 0}, // số cuối đứng trước danh từ đếm "xe": số đếm, không phải ngày
		{"EX-03 40-10", "", 0},
		{"EX-03 19-10-5", "", 0}, // năm phải 2 hoặc 4 chữ số
		{"được cấp 5", "", 0},    // một số đứng một mình (không nhãn) không phải một cụm ngày
		{"EX-03 19-10-2026-1", "", 0},
		{"EX-03 19-13", "", 0},
		{"TK-01-02-03", "", 0},
		{"3800-3900", "", 0},
	} {
		in := e.Normalize(c.in, Options{Now: now})
		var got string
		var n int
		for i := range in.Spans {
			if text, k := e.DashDateAt(in.Raw, in.Spans, i); k > 0 {
				got, n = text, k
				break
			}
		}
		if got != c.want || n != c.n {
			t.Errorf("%q: DashDateAt = %q (%d span), muốn %q (%d span); spans: %s", c.in, got, n, c.want, c.n, dump(in.Spans))
		}
	}
}

// Chuỗi ngẫu nhiên (số, từ chỉ kỳ, dấu nối, ký tự lạ): PeriodQualifier, LabelContext và DashDateAt không panic, trả vị trí nằm trong chuỗi, và Raw / Word của mốc
// đúng là đoạn chữ của chuỗi (luồng hỏi cắt chuỗi theo các vị trí này).
func TestQualifierVaDashDateKhongPanicTrenChuoiNgauNhien(t *testing.T) {
	e := mustEngine(t)
	pieces := []string{
		"ngày", "Ngày", "hôm", "mùng", "tuần", "tháng", "quý", "năm", "qua", "trước", "này", "mỗi", "hàng", "cứ", "phiếu", "số", "ca", "đội", "khối", "L", "lít", "máy", "lần",
		"chuyến", "xe", "0", "1", "5", "10", "12", "19", "31", "32", "999", "2025", "12345", "3.850", "3,5", "19.10", "-", "–", "/", ".", ",", ":", "(", ")", " ", "  ",
		"\u00a0", "\u200b", "\n", "EX-03", "TK-", "k", "ℓ", "５", "é", "\u0301",
	}
	rng := rand.New(rand.NewPCG(20261021, 35))
	for n := 0; n < 30000; n++ {
		var b strings.Builder
		for k := rng.IntN(10) + 1; k > 0; k-- {
			b.WriteString(pieces[rng.IntN(len(pieces))])
			if rng.IntN(3) == 0 {
				b.WriteByte(' ')
			}
		}
		in := e.Normalize(b.String(), Options{Now: now})
		for i, sp := range in.Spans {
			if q, ok := PeriodQualifier(in.Raw, sp.Start, sp.End); ok {
				if q.Raw == "" || !strings.Contains(in.Raw, q.Raw) || q.Word[0] < 0 || q.Word[1] > len(in.Raw) || q.Word[0] >= q.Word[1] {
					t.Fatalf("%q: mốc %+v không nằm trong chuỗi", in.Raw, q)
				}
			}
			e.LabelContext(in.Raw, sp.Start, sp.End)
			if text, k := e.DashDateAt(in.Raw, in.Spans, i); k > 0 && (text == "" || !strings.Contains(in.Raw, text) || i+k > len(in.Spans)) {
				t.Fatalf("%q: DashDateAt = %q (%d span) không khớp chuỗi", in.Raw, text, k)
			}
		}
		// vị trí ngoài khoảng hợp lệ trả false thay vì panic
		PeriodQualifier(in.Raw, -1, 2)
		PeriodQualifier(in.Raw, 0, len(in.Raw)+5)
		e.LabelContext(in.Raw, len(in.Raw), len(in.Raw))
	}
}
