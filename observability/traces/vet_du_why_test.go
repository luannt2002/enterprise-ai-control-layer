package traces_test

import (
	"encoding/json"
	"slices"
	"strings"
	"testing"

	"github.com/luannt2002/enterprise-ai-control-layer/observability/traces"
)

// Chữ mới do luồng hỏi xuất cũng qua bộ làm sạch bí mật: nhét bí mật vào từng ô giải thích của lượt thật rồi làm sạch.
func TestVetDuChuMoiQuaBoLamSach(t *testing.T) {
	secret := "AKIA" + strings.Repeat("Q7", 8)
	turn := realTurn(t, realAsk(t), person("director", ""), "EX-03 hôm nay được cấp 4k dầu đúng không?")
	w := &turn.Decision.Why
	for _, n := range []*traces.Notes{&w.Normalize, &w.Resolve, &w.Intent, &w.Route, &w.Context, &w.Validate} {
		if len(n.Reasons) == 0 {
			t.Fatalf("tiền đề: lượt thật phải mang giải thích ở mọi chặng đã xuất: %+v", w)
		}
		n.Reasons = append(slices.Clone(n.Reasons), "khoá "+secret)
		n.Rejected = append(slices.Clone(n.Rejected), "loại: password=hunter2pw")
	}
	clean, err := traces.SanitizeRecord(traces.NewRecord(turn))
	if err != nil {
		t.Fatal(err)
	}
	b, _ := json.Marshal(clean)
	for _, leak := range []string{secret, "hunter2pw"} {
		if strings.Contains(string(b), leak) {
			t.Errorf("bí mật %q còn trong bản ghi đã làm sạch: %s", leak, b)
		}
	}
	if !strings.Contains(string(b), traces.Masked) {
		t.Errorf("chữ mới phải bị che bằng %s", traces.Masked)
	}
}

// Giải thích chỉ để ĐỌC: hai quyết định giống nhau mà giải thích khác nhau vẫn là cùng quyết định (Diff không so), và cùng một lượt chạy
// hai lần cho đúng cùng chữ (không phụ thuộc thứ tự duyệt map).
func TestVetDuKhongDoiQuyetDinhVaTaiLap(t *testing.T) {
	q := "RL-02 và máy 03 hôm nay được cấp 3850 L đúng không?"
	a, b := realTurn(t, realAsk(t), person("director", ""), q), realTurn(t, realAsk(t), person("director", ""), q)
	for range 20 {
		b = realTurn(t, realAsk(t), person("director", ""), q)
		ja, _ := json.Marshal(a.Decision.Why)
		jb, _ := json.Marshal(b.Decision.Why)
		if string(ja) != string(jb) {
			t.Fatalf("cùng lượt, hai lần chạy ra giải thích khác nhau:\n%s\n%s", ja, jb)
		}
	}
	x := b.Decision
	x.Why.Route.Reasons = append(slices.Clone(x.Why.Route.Reasons), "thêm")
	x.Why.Intent.Rejected = nil
	if d := traces.Diff(b.Decision, x); len(d) != 0 {
		t.Errorf("giải thích không phải quyết định, Diff phải rỗng, được %v", d)
	}
}

// Bản ghi BỀN không giữ Why lần thứ hai: Steps đã mang đúng các dòng đó. Nhưng chữ phải còn trong Steps — mất ở đây là mất vết.
func TestVetDuBanGhiBenKhongGiuWhyHaiLan(t *testing.T) {
	turn := realTurn(t, realAsk(t), person("director", ""), "EX-03 hôm nay được cấp bao nhiêu lít dầu?")
	if len(turn.Decision.Why.Route.Reasons) == 0 {
		t.Fatal("tiền đề: lượt thật phải mang Why")
	}
	rec := traces.NewRecord(turn)
	if got := rec.Decision.Why; len(got.Normalize.Reasons)+len(got.Resolve.Reasons)+len(got.Intent.Reasons)+len(got.Route.Reasons)+len(got.Context.Reasons)+len(got.Validate.Reasons) != 0 {
		t.Errorf("bản ghi bền không được giữ Why: %+v", got)
	}
	if len(turn.Decision.Why.Route.Reasons) == 0 {
		t.Errorf("NewRecord không được sửa Turn của nơi gọi")
	}
	for _, s := range rec.Steps {
		if s.Stage == "S5" && (traces.IsNotRecorded(s.Reason) || !strings.Contains(s.Reason, "đã kiểm theo thứ tự")) {
			t.Errorf("giải thích phải nằm trong Steps: %+v", s)
		}
	}
	b, _ := json.Marshal(rec.Decision)
	if strings.Contains(string(b), `"why"`) {
		t.Errorf("JSON của quyết định bền không được có why: %s", b)
	}
}
