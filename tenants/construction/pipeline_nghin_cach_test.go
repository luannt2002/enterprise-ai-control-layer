package construction_test

import (
	"strings"
	"testing"

	"github.com/luannt2002/enterprise-ai-systems/core/contracts"
)

// Câu xác nhận có lượng viết nhóm nghìn ngăn bằng dấu cách ("3 850 lít", docs/tenants/construction/03-data-map.md:134) không được
// phán quyết theo một cách đọc hệ tự chọn. Không có chữ đứng ngay trước thì là MỘT lượng và được phán quyết theo đúng lượng đó;
// có chữ đứng trước ("được cấp 3 850 lít", "tổng số 3 850 lít", "bể 2 500 lít") thì phần đầu có thể là số hiệu của chữ đó, nên câu
// có hai con số và được hỏi lại. Lịch sử: ban đầu "3 850 lít" thành "3" + "850 lít" và luật A-02 coi "3" là nhãn ⇒ câu đúng bị trả
// "Không … không phải 850 L" (ca 264 của H1, SAU lần chạy chính thức); 8ff3db5 gộp sau mọi chữ trừ một danh sách từ nhãn, và
// người soát C3 tìm ra danh sách đó rò ("tổng số", "mỗi lần", "bể": C3-02 / C3-05). Hôm nay = 19/10/2026, EX-03 được cấp 3.850 L.
func TestCauXacNhanNhomNghinNganBangDauCach(t *testing.T) {
	a := newAsk(t)
	for _, c := range []struct {
		text, prefix, mustHave string
	}{
		{"EX-03 hôm nay được cấp: 3 850 lít dầu, đúng không?", "Đúng.", ""},
		{"EX-03 hôm nay được cấp (3 850 L) đúng không?", "Đúng.", ""},
		{"EX-03 hôm nay được cấp: 4 000 lít dầu, đúng không?", "Không.", "không phải 4.000 L"},
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
	} {
		if r := ask(t, a, user("director", ""), q).Response; r.Outcome != contracts.OutcomeClarificationRequired {
			t.Errorf("%q: muốn hỏi lại (phần đầu có thể là số hiệu / mốc), được [%s/%s] %q", q, r.Route, r.Outcome, r.Answer)
		}
	}
	// Đối chứng: số hiệu sau từ nhãn rồi một lượng KHÔNG có dạng nhóm ba chữ số thì không mơ hồ
	xacNhan(t, ask(t, a, user("director", ""), "EX-03 hôm nay theo phiếu 912 3850 L được cấp đúng không?").Response,
		"phiếu 912 3850 L", "Đúng.")
}
