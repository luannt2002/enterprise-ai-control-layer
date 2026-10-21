package normalization

import (
	"slices"
	"testing"

	"github.com/luannt2002/enterprise-ai-control-layer/core/contracts"
)

// S2-4: cụm có DẠNG ngày nhưng không có trên lịch ("31/02", "32/10", "29/02/2026") không sinh span ngày — nên nếu bộ
// chuẩn hoá không ghi lại, câu hỏi bị hiểu như không nêu ngày và rơi về ngày mặc định. Chúng phải nằm ở InvalidDates.
func TestNgayHongDuocGhiLai(t *testing.T) {
	e := mustEngine(t)
	for _, c := range []struct {
		in   string
		want []string
	}{
		{"31/2", []string{"31/2"}},
		{"EX-03 ngày 31/02 được cấp bao nhiêu", []string{"31/02"}},
		{"ngày 32/10", []string{"32/10"}},
		{"ngày 0/10", []string{"0/10"}},
		{"ngày 15/13", []string{"15/13"}},
		{"ngày 31/04/2026", []string{"31/04/2026"}},
		{"ngày 29/02/2026", []string{"29/02/2026"}}, // 2026 không phải năm nhuận
		{"ngày 31-04-2026", []string{"31-04-2026"}},
		{"ngày 2026-02-30", []string{"2026-02-30"}},
		{"ngày 2026-13-01", []string{"2026-13-01"}},
		{"ngày 32/10/2026", []string{"32/10/2026"}},
		// năm hai chữ số là 20yy (16/9/26 là ngày CÓ THẬT — xem engine_hoi_quy_test.go); ngày hỏng viết kiểu đó ghi cả cụm, không ghi "09/26"
		{"ngày 31/09/26", []string{"31/09/26"}},
		// nhiều ngày hỏng, hoặc ngày hỏng cạnh ngày tốt: ghi đúng các cụm hỏng
		{"từ 31/02 đến 05/03", []string{"31/02"}},
		{"ngày 31/02 và 30/02", []string{"31/02", "30/02"}},
		{"hôm nay và ngày 31/02", []string{"31/02"}},
	} {
		got := e.Normalize(c.in, Options{Now: now}).InvalidDates
		if !slices.Equal(got, c.want) {
			t.Errorf("%q: InvalidDates = %q, muốn %q", c.in, got, c.want)
		}
	}
}

// Ngày tốt và các cụm số KHÔNG có dạng ngày không được ghi là ngày hỏng — nếu không, câu hỏi đúng bị hỏi lại thừa.
func TestNgayHopLeVaSoThuongKhongBiGhiLaHong(t *testing.T) {
	e := mustEngine(t)
	for _, in := range []string{
		"16/9", "16/09/2026", "16-9-2026", "2025-11-28", "31/12/2026", "1/1/2026", "28/2",
		"29/02/2028", "29/02/2024", // năm nhuận
		"hôm nay", "tháng 9", "tháng 9/2026",
		"205/55R16", "205 55 r16", // mã cỡ lốp (code) và cụm số 3 chữ số
		"18-20 lít", "4.000 L", "máy 03 hôm nay", "3.850L", "dầu 4k",
		"100/10",  // phần ngày 3 chữ số: không phải dạng ngày
		"02/2026", // phần tháng 4 chữ số: không phải dạng ngày
		"1.5/2",   // có dấu thập phân
		"31/02R",  // dính chữ phía sau: không đứng riêng thành một cụm ngày
		"dầu 15/13b",
		"TK-01-02-03", "mã 10-20-30", // ba số nhỏ nối bằng "-" là mã, không phải ngày dd-mm-yy
	} {
		if got := e.Normalize(in, Options{Now: now}).InvalidDates; len(got) != 0 {
			t.Errorf("%q: không phải ngày hỏng nhưng InvalidDates = %q", in, got)
		}
	}
}

// Ghi lại ngày hỏng KHÔNG được đổi các span: "31/2" vẫn là hai số trần (NRM-V-043), không sinh span ngày, để
// bộ nạp và bộ kiểm câu trả lời đọc đúng như trước.
func TestNgayHongKhongDoiSpan(t *testing.T) {
	e := mustEngine(t)
	got := e.Normalize("ngày 31/02", Options{Now: now})
	if len(got.SpansOf(contracts.ValueDate, contracts.ValuePeriod)) != 0 {
		t.Errorf("ngày hỏng không được sinh span ngày: %s", dump(got.Spans))
	}
	if n := len(got.SpansOf(contracts.ValueNumber)); n != 2 {
		t.Errorf("31 và 02 vẫn là hai số trần như trước, được %d số: %s", n, dump(got.Spans))
	}
	if len(got.Ambiguities) != 0 {
		t.Errorf("ngày hỏng không phải viết tắt mơ hồ: %+v", got.Ambiguities)
	}
}
