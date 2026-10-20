package normalization

import (
	"math/rand/v2"
	"slices"
	"strings"
	"testing"

	"github.com/luannt2002/enterprise-ai-systems/core/contracts"
)

// Test cho các lỗi của bản soát độc lập lượt 2 ở tầng chuẩn hoá (A-05, A-06, A-07, A-13 và mục "ranh giới TRÁI" người phản bác
// tìm thêm). Hôm nay (now) = 19/10/2026.

// dateSpans trả (Raw, ngày chuẩn) của mọi span ngày / kỳ, theo thứ tự xuất hiện.
func dateSpans(in contracts.NormalizedInput) (raws, values []string) {
	for _, sp := range in.SpansOf(contracts.ValueDate, contracts.ValuePeriod) {
		raws, values = append(raws, sp.Raw), append(values, sp.Value.Text)
	}
	return raws, values
}

// A-05 — năm hai chữ số và dấu "." / "-" / khoảng trắng quanh "/" là cách viết ngày CÓ THẬT (docs/tenants/construction/
// 03-data-map.md:162: `19/10`, `19-10-2026`, `19.10.26`; 02-domain-map.md:71, :296: `19.10.26`, `19-10`).
func TestNgayHaiChuSoNamVaDauChamGachNoiDocDuoc(t *testing.T) {
	e := mustEngine(t)
	for _, c := range []struct{ in, raw, want string }{
		{"19/10/26", "19/10/26", "2026-10-19"},
		{"16/9/26", "16/9/26", "2026-09-16"},
		{"cấp ngày 16/09/26 là", "16/09/26", "2026-09-16"},
		{"19.10.26", "19.10.26", "2026-10-19"},
		{"19.10.2026", "19.10.2026", "2026-10-19"},
		{"1.2.26", "1.2.26", "2026-02-01"},
		// hai phần dùng "." hay "-", ba phần dùng "-" năm hai chữ số, và dấu cách quanh "/": chỉ khi có chữ "ngày" đứng ngay trước
		{"ngày 19.10", "19.10", "2026-10-19"},
		{"ngày 19-10", "19-10", "2026-10-19"},
		{"ngày 19-10-26", "19-10-26", "2026-10-19"},
		{"ngày 19 / 10", "19 / 10", "2026-10-19"},
		{"ngày 19/ 10", "19/ 10", "2026-10-19"},
		{"Ngày 19.10.26", "19.10.26", "2026-10-19"},
		// dd/mm lùi một năm khi ở tương lai (hôm nay 19/10/2026)
		{"ngày 25.12", "25.12", "2025-12-25"},
		// các dạng đã có từ trước giữ nguyên
		{"ngày 19/10", "19/10", "2026-10-19"},
		{"ngày 19-10-2026", "19-10-2026", "2026-10-19"},
		{"2025-11-28", "2025-11-28", "2025-11-28"},
	} {
		got := e.Normalize(c.in, Options{Now: now})
		raws, values := dateSpans(got)
		if !slices.Equal(raws, []string{c.raw}) || !slices.Equal(values, []string{c.want}) || len(got.InvalidDates) != 0 {
			t.Errorf("%q: span ngày = %q → %q, ngày hỏng = %q; muốn %q → %q và không có ngày hỏng", c.in, raws, values, got.InvalidDates, c.raw, c.want)
		}
	}
}

// A-05 — hai phần dùng "." / "-", và ba phần "-" năm hai chữ số, KHÔNG có chữ "ngày" thì không phải ngày: "19.10" là số thập phân,
// "10-12 giờ" là khoảng giờ, "TK-01-02-03" và "mã 10-12-20" là mã.
func TestDangNgayMoRongCanChuNgayDungTruoc(t *testing.T) {
	e := mustEngine(t)
	for _, in := range []string{
		"19.10", "19-10", "31.02", "31-02", "cấp 19.10 lít", "chạy 10-12 giờ", "TK-01-02-03", "mã 10-12-20", "mã 10-20-30", "18-20 lít",
		"19/10-26", // dấu nối khác nhau: không phải dd/mm/yy
		"1.602.000đ", "3.850.000", "ngày 3.850 L",
	} {
		got := e.Normalize(in, Options{Now: now})
		if raws, _ := dateSpans(got); len(raws) != 0 || len(got.InvalidDates) != 0 {
			t.Errorf("%q: không phải ngày nhưng span ngày = %q, ngày hỏng = %q", in, raws, got.InvalidDates)
		}
	}
}

// A-05 — ngày hỏng viết bằng "." / "-" / dấu cách quanh dấu, hoặc năm hai chữ số, phải được ghi lại (nếu không, câu rơi về hôm nay).
func TestNgayHongVietDauChamGachNoiDuocGhiLai(t *testing.T) {
	e := mustEngine(t)
	for _, c := range []struct {
		in   string
		want []string
	}{
		{"ngày 31.02", []string{"31.02"}},
		{"ngày 31-02", []string{"31-02"}},
		{"ngày 31 / 02", []string{"31 / 02"}},
		{"ngày 31.02.2026", []string{"31.02.2026"}},
		{"ngày 31/02/26", []string{"31/02/26"}},
		{"ngày 31.02.26", []string{"31.02.26"}},
		{"ngày 31/09/26", []string{"31/09/26"}},
		{"ngày 29.02.2026", []string{"29.02.2026"}},
		{"ngày 32.10", []string{"32.10"}},
		{"ngày 31-02-26", []string{"31-02-26"}},
		// năm bốn chữ số kèm theo: người hỏi đã nói đây là ngày, không cần chữ "ngày" đứng trước
		{"2026-13-01", []string{"2026-13-01"}},
		{"32/10/2026", []string{"32/10/2026"}},
		// ngày 16/9/26 là ngày CÓ THẬT — không còn là ngày hỏng
		{"ngày 16/9/26", nil},
	} {
		got := e.Normalize(c.in, Options{Now: now}).InvalidDates
		if !slices.Equal(got, c.want) {
			t.Errorf("%q: InvalidDates = %q, muốn %q", c.in, got, c.want)
		}
	}
}

// A-06 — cụm số/số KHÔNG có dạng ngày không phải ngày hỏng: ngày phải ≤ 31 và tháng ≤ 12 (31/02 mới chỉ sai vì lịch); chữ "ngày"
// đứng ngay trước thì lỏng hơn (người hỏi đã nói đây là ngày). Ba phần nhỏ mà năm không phải 2 hay 4 chữ số ("1/2/3") không phải ngày,
// và phần nhỏ của chúng ("2/3") càng không được đọc thành ngày.
func TestCumSoKhongCoDangNgayKhongBiGhiLaHong(t *testing.T) {
	e := mustEngine(t)
	for _, in := range []string{
		"24/24", "60/40", "12/34", "30/60", "45/50", "99/99", "12/34/56", "24/24/24", "1/2/3", "3/4/5", "chạy 24/24 được cấp",
		"tỷ lệ 60/40", "(30/60)", "theo hợp đồng 12/34/56", "hợp đồng 1/2/3 được cấp", "100/10", "02/2026", "205/55",
	} {
		got := e.Normalize(in, Options{Now: now})
		if raws, _ := dateSpans(got); len(got.InvalidDates) != 0 || len(raws) != 0 {
			t.Errorf("%q: không phải ngày nhưng ngày hỏng = %q, span ngày = %q", in, got.InvalidDates, raws)
		}
	}
	// có dạng ngày (≤ 31 / ≤ 12) hoặc có chữ "ngày": vẫn là ngày hỏng
	for _, c := range []struct {
		in   string
		want []string
	}{
		{"31/2", []string{"31/2"}},
		{"31/04", []string{"31/04"}},
		{"ngày 24/24", []string{"24/24"}},
		{"ngày 12/34/56", []string{"12/34/56"}},
		{"ngày 60/40", []string{"60/40"}},
	} {
		if got := e.Normalize(c.in, Options{Now: now}).InvalidDates; !slices.Equal(got, c.want) {
			t.Errorf("%q: InvalidDates = %q, muốn %q", c.in, got, c.want)
		}
	}
}

// A-07 — "ngày D tháng M [năm Y | /Y]" là MỘT ngày (Raw gồm cả chữ "ngày"): không phải "tháng M" bỏ "ngày D", không để "D" thành số trần.
func TestNgayThangVietChuLaMotNgayO(t *testing.T) {
	e := mustEngine(t)
	for _, c := range []struct{ in, raw, want string }{
		{"ngày 19 tháng 10 được cấp", "ngày 19 tháng 10", "2026-10-19"},
		{"ngày 19 tháng 10 năm 2026", "ngày 19 tháng 10 năm 2026", "2026-10-19"},
		{"ngày 19 tháng 10/2026", "ngày 19 tháng 10/2026", "2026-10-19"},
		{"ngày 15 tháng 9", "ngày 15 tháng 9", "2026-09-15"},
		{"Ngày 5 tháng 3 năm 2025", "Ngày 5 tháng 3 năm 2025", "2025-03-05"},
		{"ngày 05 tháng 03", "ngày 05 tháng 03", "2026-03-05"},
		// lùi một năm khi ở tương lai (hôm nay 19/10/2026), như dd/mm
		{"ngày 25 tháng 12", "ngày 25 tháng 12", "2025-12-25"},
		{"ngày 25 tháng 12 năm 2026", "ngày 25 tháng 12 năm 2026", "2026-12-25"},
	} {
		got := e.Normalize(c.in, Options{Now: now})
		raws, values := dateSpans(got)
		if !slices.Equal(raws, []string{c.raw}) || !slices.Equal(values, []string{c.want}) || len(got.InvalidDates) != 0 {
			t.Errorf("%q: span ngày = %q → %q, ngày hỏng = %q; muốn %q → %q", c.in, raws, values, got.InvalidDates, c.raw, c.want)
		}
		if n := len(got.SpansOf(contracts.ValueNumber)); n != 0 {
			t.Errorf("%q: số của ngày không được thành số trần (%d số): %s", c.in, n, dump(got.Spans))
		}
	}
	// cùng một ngày viết hai cách: hai span, CÙNG giá trị chuẩn (bước ý định gộp thành một kỳ)
	if raws, values := dateSpans(e.Normalize("hôm nay, ngày 19 tháng 10, được cấp", Options{Now: now})); !slices.Equal(raws, []string{"hôm nay", "ngày 19 tháng 10"}) ||
		!slices.Equal(values, []string{"2026-10-19", "2026-10-19"}) {
		t.Errorf("hôm nay, ngày 19 tháng 10: %q → %q", raws, values)
	}
	// "ngày mai" và "tháng M" riêng vẫn như cũ
	if raws, values := dateSpans(e.Normalize("ngày mai", Options{Now: now})); !slices.Equal(values, []string{"2026-10-20"}) || !slices.Equal(raws, []string{"ngày mai"}) {
		t.Errorf("ngày mai: %q → %q", raws, values)
	}
	if _, values := dateSpans(e.Normalize("tháng 9", Options{Now: now})); !slices.Equal(values, []string{"2026-09-01/2026-09-30"}) {
		t.Errorf("tháng 9: %q", values)
	}
}

// A-07 — "từ ngày D đến ngày D2 tháng M": hai đầu mút (đầu mút thứ nhất mượn tháng và năm của đầu mút thứ hai), để câu nêu hai kỳ
// và được hỏi lại — không đọc thành cả tháng, không để chỉ còn ngày cuối.
func TestKhoangNgayVietChuCoHaiDauMut(t *testing.T) {
	e := mustEngine(t)
	for _, c := range []struct {
		in         string
		raws, want []string
	}{
		{"từ ngày 10 đến ngày 15 tháng 10", []string{"ngày 10", "ngày 15 tháng 10"}, []string{"2026-10-10", "2026-10-15"}},
		{"từ ngày 10 tới ngày 15 tháng 10", []string{"ngày 10", "ngày 15 tháng 10"}, []string{"2026-10-10", "2026-10-15"}},
		{"từ ngày 10 đến ngày 15 tháng 9 năm 2025", []string{"ngày 10", "ngày 15 tháng 9 năm 2025"}, []string{"2025-09-10", "2025-09-15"}},
		{"từ ngày 10 tháng 10 đến ngày 15 tháng 10", []string{"ngày 10 tháng 10", "ngày 15 tháng 10"}, []string{"2026-10-10", "2026-10-15"}},
	} {
		got := e.Normalize(c.in, Options{Now: now})
		raws, values := dateSpans(got)
		if !slices.Equal(raws, c.raws) || !slices.Equal(values, c.want) || len(got.InvalidDates) != 0 {
			t.Errorf("%q: span ngày = %q → %q, ngày hỏng = %q; muốn %q → %q", c.in, raws, values, got.InvalidDates, c.raws, c.want)
		}
	}
	// "ngày D" một mình (không có tháng, không có đầu mút kia) vẫn không phải span ngày — hành vi cũ giữ nguyên
	if raws, _ := dateSpans(e.Normalize("ngày 10 được cấp", Options{Now: now})); len(raws) != 0 {
		t.Errorf("ngày 10 một mình không phải span ngày: %q", raws)
	}
}

// A-07 — "ngày D tháng M" không có trên lịch là ngày hỏng (ghi "D tháng M", không kèm chữ "ngày"), và không để lại một kỳ "tháng M" mồ côi.
func TestNgayThangVietChuKhongCoTrenLichLaNgayHong(t *testing.T) {
	e := mustEngine(t)
	for _, c := range []struct {
		in   string
		want []string
	}{
		{"ngày 31 tháng 2", []string{"31 tháng 2"}},
		{"ngày 31 tháng 4 năm 2026", []string{"31 tháng 4 năm 2026"}},
		{"ngày 29 tháng 2 năm 2026", []string{"29 tháng 2 năm 2026"}},
		{"ngày 5 tháng 13", []string{"5 tháng 13"}},
		{"ngày 0 tháng 10", []string{"0 tháng 10"}},
		// đầu mút mượn tháng mà ngày không có trong tháng đó
		{"từ ngày 31 đến ngày 5 tháng 11", []string{"31"}},
	} {
		got := e.Normalize(c.in, Options{Now: now})
		if !slices.Equal(got.InvalidDates, c.want) {
			t.Errorf("%q: InvalidDates = %q, muốn %q", c.in, got.InvalidDates, c.want)
		}
		for _, sp := range got.SpansOf(contracts.ValuePeriod) {
			t.Errorf("%q: không được để lại kỳ mồ côi %q (%s)", c.in, sp.Raw, sp.Value.Text)
		}
	}
}

// Ranh giới TRÁI của ngày dạng số: cụm dd/mm dính sau một mã chữ ("PX-12/10", "BB-24/50", "TK-19/10") hoặc là phần sau của một dãy số
// dài hơn ("2/3" của "1/2/3") không phải ngày — không sinh span ngày, không ghi ngày hỏng.
func TestNgaySoDinhSauMaHoacDaySoKhongPhaiNgay(t *testing.T) {
	e := mustEngine(t)
	for _, in := range []string{
		"PX-12/10", "phiếu PX-12/10 của máy", "BB-24/50", "theo biên bản BB-24/50", "TK-19/10", "tk-12/10", "A/12/10", "hợp đồng 1/2/3",
		"BB.24/50", "PX_12/10",
	} {
		got := e.Normalize(in, Options{Now: now})
		if raws, _ := dateSpans(got); len(raws) != 0 || len(got.InvalidDates) != 0 {
			t.Errorf("%q: không phải ngày nhưng span ngày = %q, ngày hỏng = %q", in, raws, got.InvalidDates)
		}
	}
	// đứng riêng, hoặc sau chữ cách dấu cách / dấu câu: vẫn là ngày
	for _, c := range []struct{ in, want string }{
		{"máy 03 12/10", "2026-10-12"}, {"phiếu (12/10)", "2026-10-12"}, {"phiếu: 12/10", "2026-10-12"}, {"PX 12/10", "2026-10-12"},
	} {
		if _, values := dateSpans(e.Normalize(c.in, Options{Now: now})); !slices.Equal(values, []string{c.want}) {
			t.Errorf("%q: span ngày = %q, muốn %q", c.in, values, c.want)
		}
	}
}

// A-13 — hậu tố nhân cách số bằng nhiều dấu cách hoặc ký tự vô hình ("5  k lít", "5<ZWSP>k lít") là cùng một lượng với "5k lít".
func TestHauToNhanCachNhieuKhoangDem(t *testing.T) {
	e := mustEngine(t)
	for _, in := range []string{"5k lít", "5 k lít", "5  k lít", "5​k lít", "5  k lít", "5 ​ k lít", "5 k  lít", "5k​lít"} {
		q := quantities(e.Normalize("cấp "+in, Options{Now: now}))
		if len(q) != 1 || q[0].Value.Number != "5000" || q[0].Value.Unit != "L" {
			t.Errorf("%q: muốn đúng một lượng 5000 L, được %+v", in, q)
		}
	}
	// "k" dính vào chữ khác (kg, khối) KHÔNG phải hậu tố nhân dù có khoảng đệm
	for _, in := range []string{"cấp 5  kg", "cấp 5 khối"} {
		for _, sp := range quantities(e.Normalize(in, Options{Now: now})) {
			if sp.Value.Number == "5000" {
				t.Errorf("%q: đọc nhầm \"k\" của từ khác thành hậu tố nhân: %+v", in, sp)
			}
		}
	}
}

// Bộ đọc ngày có nhiều phép tính chỉ số token và vị trí byte (dãy số có khoảng trắng, "ngày D tháng M năm Y", đầu mút mượn tháng):
// trên chuỗi ghép ngẫu nhiên từ các mảnh ngày / số / dấu / khoảng trắng / ký tự vô hình / chữ nhiều byte, Normalize không được
// panic và mọi span phải trỏ ĐÚNG vào Raw (Raw[Start:End] == span.Raw), ngày hỏng không rỗng.
func TestNormalizeChuoiNgayNgauNhienKhongPanicVaToaDoDung(t *testing.T) {
	e := mustEngine(t)
	pieces := []string{
		"ngày", "Ngày", "tháng", "năm", "đến", "tới", "từ", "hôm nay", "hôm qua", "tuần này", "ngay", "thang",
		"0", "1", "5", "10", "12", "19", "26", "31", "32", "99", "100", "2026", "205", "4.000", "3,5", "19.10", "19.10.26",
		"/", "-", ".", ",", ":", "(", ")", " ", "  ", " ", "​", "\n", "PX", "TK-", "k", "lít", "ca", "đ", "ℓ", "５", "é", "́",
	}
	rng := rand.New(rand.NewPCG(20261021, 7))
	for n := 0; n < 30000; n++ {
		var b strings.Builder
		for k := rng.IntN(12) + 1; k > 0; k-- {
			b.WriteString(pieces[rng.IntN(len(pieces))])
			if rng.IntN(3) == 0 {
				b.WriteByte(' ')
			}
		}
		text := b.String()
		for _, layout := range []bool{false, true} {
			got := e.Normalize(text, Options{Now: now, Layout: layout})
			for _, sp := range got.Spans {
				if sp.Start < 0 || sp.End < sp.Start || sp.End > len(got.Raw) || got.Raw[sp.Start:sp.End] != sp.Raw {
					t.Fatalf("%q (layout=%v): span %q lệch vị trí [%d,%d) trong Raw %q", text, layout, sp.Raw, sp.Start, sp.End, got.Raw)
				}
			}
			for _, bad := range got.InvalidDates {
				if bad == "" || !strings.Contains(got.Raw, bad) {
					t.Fatalf("%q (layout=%v): ngày hỏng %q rỗng hoặc không nằm trong Raw %q", text, layout, bad, got.Raw)
				}
			}
		}
	}
}

// Số đã mang đơn vị riêng ("lít 5.000 giờ") thuộc đơn vị của nó, kể cả khi có dạng lượng: đơn vị đứng trước (chế độ Layout) không nuốt nó.
func TestLayoutSoCoDonViRiengKhongThuocDonViDungTruoc(t *testing.T) {
	e := mustEngine(t)
	for _, in := range []string{"lít 5.000 giờ", "L 500 giờ", "lít: 5.000 giờ", "(lít): 1.500 giờ"} {
		q := quantities(e.Normalize(in, Options{Now: now, Layout: true}))
		if len(q) != 1 || q[0].Value.Unit != "h" {
			t.Errorf("%q: muốn đúng một lượng thuộc giờ, được %+v", in, q)
		}
	}
}

// Chữ "ngày" đứng trước một LƯỢNG không làm lượng đó thành ngày: dấu nghìn ("mỗi ngày 4.000 lít" — nhóm 3 chữ số không phải tháng) và
// khoảng / số thập phân có đơn vị theo sau ("mỗi ngày 10-12 giờ", "ngày 2.5 tấn") là lượng; "ngày 19.10", "ngày 19-10" vẫn là ngày.
func TestChuNgayDungTruocLuongKhongBienLuongThanhNgay(t *testing.T) {
	e := mustEngine(t)
	for _, in := range []string{
		"mỗi ngày 4.000 lít", "một ngày 5.000 L", "ngày 4.000", "ngày 12.345 lít", "ngày 1.500", "mỗi ngày 10-12 giờ", "mỗi ngày 2.5 người",
		"ngày 2-3 người", "ngày 12.50 L", "mỗi ngày 10 - 12 giờ", "ngày 10 / 12 giờ",
	} {
		got := e.Normalize(in, Options{Now: now})
		if raws, _ := dateSpans(got); len(raws) != 0 || len(got.InvalidDates) != 0 {
			t.Errorf("%q: là lượng nhưng span ngày = %q, ngày hỏng = %q", in, raws, got.InvalidDates)
		}
	}
	for _, c := range []struct{ in, want string }{
		{"ngày 19.10", "2026-10-19"}, {"ngày 19-10", "2026-10-19"}, {"ngày 19.10 lúc 14h", "2026-10-19"}, {"ngày 19-10, được cấp", "2026-10-19"},
		{"ngày 19/10 ca đêm", "2026-10-19"}, {"ngày 19.10.26", "2026-10-19"},
	} {
		if _, values := dateSpans(e.Normalize(c.in, Options{Now: now})); !slices.Equal(values, []string{c.want}) {
			t.Errorf("%q: span ngày = %q, muốn %q", c.in, values, c.want)
		}
	}
}
