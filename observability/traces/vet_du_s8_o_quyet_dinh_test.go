package traces_test

import (
	"regexp"
	"testing"

	"github.com/luannt2002/enterprise-ai-control-layer/observability/traces"
)

// V8AS-05 (soát vòng 1 của V8a). Ô QUYẾT ĐỊNH của S8 từng in mẫu câu của ROUTER ("mẫu câu của tenant metric_total") trong khi dòng lý do cùng bước nêu mẫu THẬT mà luồng hỏi đã dùng
// (no_data khi truy vấn không ra dữ liệu, confirm_yes / confirm_no cho câu xác nhận, mẫu cộng confirm_withheld khi cổng giữ phán quyết): người đọc thấy hai tên mẫu trên cùng một hàng (133 / 474 lượt
// của người soát). Ô phải khớp mẫu thật trong dòng lý do.

var usedTemplateInReason = regexp.MustCompile(`bản nháp dựng bằng mẫu câu (.+?) \(không gọi model\)`)

func TestVetS8OQuyetDinhKhopMauThat(t *testing.T) {
	a := realAsk(t)
	director := person("director", "")
	for _, c := range []struct{ name, q, want string }{
		{"truy vấn không ra dữ liệu", "Máy 07 hôm nay được cấp bao nhiêu dầu?", "no_data"},
		{"xác nhận đúng", "EX-03 hôm nay được cấp 3850 L đúng không?", "confirm_yes"},
		{"xác nhận sai", "EX-03 hôm nay được cấp 4000 L đúng không?", "confirm_no"},
		{"cổng giữ phán quyết", "EX-03 được cấp 3850 L đúng không?", "metric_total + confirm_withheld"},
		{"mẫu của router đúng là mẫu dùng", "EX-03 hôm nay được cấp bao nhiêu lít dầu?", "metric_total"},
		{"chuyển người: mẫu của router đúng là mẫu dùng", "DZ-03 hôm nay được cấp bao nhiêu dầu?", "needs_review"},
	} {
		t.Run(c.name, func(t *testing.T) {
			turn := realTurn(t, a, director, c.q)
			s8 := stepOf(t, traces.NewRecord(turn).Steps, "S8")
			m := usedTemplateInReason.FindStringSubmatch(s8.Reason)
			if m == nil || m[1] != c.want {
				t.Fatalf("tiền đề: dòng lý do S8 phải nêu mẫu thật %q:\n  %s", c.want, s8.Reason)
			}
			if want := "mẫu câu của tenant " + c.want; s8.Decision != want {
				t.Errorf("ô QUYẾT ĐỊNH của S8 phải khớp mẫu thật trong dòng lý do (router chọn %q):\n  muốn: %s\n  được: %s", turn.Plan.Template, want, s8.Decision)
			}
		})
	}
	// lượt cũ / bản dựng tay không mang dòng lý do của S8: ô vẫn là mẫu của router như trước
	turn := realTurn(t, a, director, "Máy 07 hôm nay được cấp bao nhiêu dầu?")
	turn.Decision.Why = traces.Why{}
	if got, want := stepOf(t, traces.NewRecord(turn).Steps, "S8").Decision, "mẫu câu của tenant "+turn.Plan.Template; got != want {
		t.Errorf("không có dòng lý do thì giữ mẫu của router: %q, muốn %q", got, want)
	}
}
