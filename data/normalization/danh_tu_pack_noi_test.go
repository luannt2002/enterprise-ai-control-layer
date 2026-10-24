package normalization_test

import (
	"strings"
	"testing"
	"time"

	"github.com/luannt2002/enterprise-ai-control-layer/core/contracts"
	"github.com/luannt2002/enterprise-ai-control-layer/core/tenant"
	"github.com/luannt2002/enterprise-ai-control-layer/data/normalization"
)

// F-030: pack cầu đường khai đúng danh từ nhãn / đếm trước đây viết cứng trong qualifier.go (hành vi không đổi), và tenant.Load nối chúng
// tới chỗ đọc của tầng ý định (intent.Options.Labels): mệnh đề trước câu hỏi chỉ nêu SỐ ĐẾM ("Đội có 2 máy") không phải mệnh đề ngữ cảnh nêu
// một con số; danh từ pack không khai ("2 bể") thì vẫn là một con số.
func TestPackCauDuongKhaiDanhTuNhanDem(t *testing.T) {
	rt, err := tenant.Load("../../tenants/construction/pack")
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range []struct {
		in, num string
		want    bool
	}{
		{"khối 2", "2", true}, {"L 5", "5", true}, {"lít 5", "5", true},
		{"cho 2 máy", "2", true}, {"1 xe", "1", true}, {"đổ 3-4 chuyến", "3", true}, {"2 thiết bị", "2", true}, {"3 dự án", "3", true},
		{"3850 L 19 tháng 10", "19", false}, {"3850 khối 19", "19", false}, {"cho 2 bể", "2", false},
	} {
		at := strings.Index(c.in, c.num)
		if got := rt.Normalizer.LabelContext(c.in, at, at+len(c.num)); got != c.want {
			t.Errorf("pack cầu đường: LabelContext(%q, %q) = %v, muốn %v", c.in, c.num, got, c.want)
		}
	}
	now := time.Date(2026, 10, 19, 10, 0, 0, 0, time.FixedZone("ICT", 7*3600))
	in := rt.Normalizer.Normalize("mỗi ngày 10-12 xe tải", normalization.Options{Now: now})
	for i := range in.Spans {
		if text, n := rt.Normalizer.DashDateAt(in.Raw, in.Spans, i); n > 0 {
			t.Errorf("pack cầu đường: %q đọc %q thành ngày dd-mm, muốn số đếm của \"xe\"", in.Raw, text)
		}
	}
	for _, c := range []struct {
		q       string
		context bool // mệnh đề đầu là mệnh đề ngữ cảnh nêu một con số
	}{
		{"Đội có 2 máy, kho xuất bao nhiêu dầu cho EX-03 hôm nay?", false},
		{"Đội có 2 bể, kho xuất bao nhiêu dầu cho EX-03 hôm nay?", true},
	} {
		r := rt.Intents.Classify(rt.Normalizer.Normalize(c.q, normalization.Options{Now: now}), contracts.Resolution{}, now)
		got := strings.Contains(strings.Join(r.Why.Reasons, "\n"), "mệnh đề ngữ cảnh")
		if got != c.context {
			t.Errorf("%q: mệnh đề ngữ cảnh = %v, muốn %v; lý do S4:\n%s", c.q, got, c.context, strings.Join(r.Why.Reasons, "\n"))
		}
	}
}
