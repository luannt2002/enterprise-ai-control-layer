package construction_test

import (
	"strings"
	"testing"
)

// Câu xác nhận có lượng viết nhóm nghìn ngăn bằng dấu cách ("3 850 lít", docs/tenants/construction/03-data-map.md:134) phải được
// phán quyết theo ĐÚNG lượng người hỏi nêu. Trước đây "3 850 lít" thành số "3" + lượng "850 lít", luật A-02 coi "3" là nhãn nên
// câu đúng bị trả "Không … không phải 850 L", và "4 000 lít" thành "không phải 0 L" (phát hiện qua ca 264 của H1 SAU lần chạy
// chính thức; H1 nay là bộ hồi quy). Hôm nay = 19/10/2026, EX-03 được cấp 3.850 L (fixture seed).
func TestCauXacNhanNhomNghinNganBangDauCach(t *testing.T) {
	a := newAsk(t)
	for _, c := range []struct {
		text, prefix, mustHave string
	}{
		{"EX-03 hôm nay được cấp 3 850 lít dầu đúng không?", "Đúng.", ""},
		{"EX-03 hôm nay được cấp 3 850 L đúng không?", "Đúng.", ""},
		{"EX-03 hôm nay được cấp 4 000 lít dầu đúng không?", "Không.", "không phải 4.000 L"},
	} {
		r := ask(t, a, user("director", ""), c.text).Response
		xacNhan(t, r, c.text, c.prefix)
		if c.mustHave != "" && !strings.Contains(r.Answer, c.mustHave) {
			t.Errorf("%q: câu trả lời %q thiếu %q", c.text, r.Answer, c.mustHave)
		}
	}
}
