package construction_test

import (
	"strings"
	"testing"

	"github.com/luannt2002/enterprise-ai-systems/core/contracts"
)

// Câu xác nhận có lượng viết nhóm nghìn ngăn bằng dấu cách (docs/tenants/construction/03-data-map.md:134) không được phán quyết theo
// một cách đọc hệ tự chọn. Con số đứng ĐẦU câu ("3 850 lít là lượng …") là MỘT lượng và được phán quyết; ở chỗ khác ("được cấp 3 850
// lít", "tổng số 3 850 lít", "bể 2 500 lít", "phiếu #1 200 L") phần đầu có thể là số hiệu ⇒ hỏi lại. Lịch sử: ban đầu "3 850 lít" thành
// "3" + "850 lít" và câu đúng bị trả "Không … không phải 850 L" (ca 264 của H1, SAU lần chạy chính thức); 8ff3db5 gộp sau mọi chữ trừ
// một danh sách từ nhãn — rò (C3-02, C3-05); fd4d19c không gộp khi có chữ đứng trước — sinh số mồ côi ("12 000 km" → "không phải
// 12 L": F1). Hôm nay = 19/10/2026, EX-03 được cấp 3.850 L.
func TestCauXacNhanNhomNghinNganBangDauCach(t *testing.T) {
	a := newAsk(t)
	for _, c := range []struct {
		text, prefix, mustHave string
	}{
		{"3 850 lít là lượng EX-03 được cấp hôm nay đúng không?", "Đúng.", ""},
		{"4 000 lít là lượng EX-03 được cấp hôm nay đúng không?", "Không.", "không phải 4.000 L"},
		// lượng khác chiều đo / tiền viết nhóm nghìn trong câu: không phải con số xác nhận, không sinh số mồ côi (soát fd4d19c, F1)
		{"EX-03 hôm nay được cấp 3850 lít, máy chạy 12 000 km đúng không?", "Đúng.", ""},
		{"EX-03 hôm nay được cấp 3850 lít dầu, giá dầu 25 000 đồng đúng không?", "Đúng.", ""},
		{"EX-03 hôm nay được cấp 3850 lít, tổng tiền 96 250 000 đồng đúng không?", "Đúng.", ""},
		{"EX-03 hôm nay được cấp 3850 lít, máy chạy được 1 200 giờ đúng không?", "Đúng.", ""},
		{"EX-03 hôm nay máy chạy 12 000 km và được cấp 3850 lít đúng không?", "Đúng.", ""},
		// số hiệu sau từ nhãn rồi một lượng KHÔNG có dạng nhóm ba chữ số thì không mơ hồ
		{"EX-03 hôm nay theo phiếu 912 3850 L được cấp đúng không?", "Đúng.", ""},
	} {
		r := ask(t, a, user("director", ""), c.text).Response
		xacNhan(t, r, c.text, c.prefix)
		if c.mustHave != "" && !strings.Contains(r.Answer, c.mustHave) {
			t.Errorf("%q: câu trả lời %q thiếu %q", c.text, r.Answer, c.mustHave)
		}
	}
	for _, q := range []string{
		"EX-03 hôm nay được cấp 3 850 lít dầu đúng không?",
		"EX-03 hôm nay được cấp 4 000 lít dầu đúng không?",
		"EX-03 hôm nay có tổng số 3 850 lít dầu được cấp đúng không?",
		"EX-03 hôm nay được cấp mỗi lần 3 850 lít đúng không?",
		"EX-03 hôm nay được cấp vào bể 2 500 lít đúng không?",
		"EX-03 hôm nay theo phiếu 12 850 L được cấp đúng không?",
		"EX-03 được cấp ngày 1 500 lít đúng không?",
		"Lu 02 hôm nay được cấp theo phiếu #1 200 L đúng không?",
		"EX-03 hôm nay được cấp: 3 850 lít dầu, đúng không?",
	} {
		r := ask(t, a, user("director", ""), q).Response
		if r.Outcome != contracts.OutcomeClarificationRequired || !strings.Contains(r.Answer, "Chưa rõ cách đọc") {
			t.Errorf("%q: muốn hỏi lại cách đọc nhóm số, được [%s/%s] %q", q, r.Route, r.Outcome, r.Answer)
		}
	}
}
