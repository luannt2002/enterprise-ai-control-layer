package commands

import (
	"encoding/json"
	"slices"
	"strings"
	"testing"

	"github.com/luannt2002/enterprise-ai-control-layer/observability/traces"
)

// F-035: tên trường lạ trong thân lệnh là CHỮ TỰ DO của bên gọi. Vết bền (bản ghi sau traces.SanitizeRecord, đúng như kho vết lưu) không
// được chứa nó — kể cả dạng mà che bí mật theo mẫu đã biết là lọt (giá trị mở bằng nháy thoát, sanitize.go "Giới hạn đã biết"). Vết chỉ
// ghi SỐ trường lạ; tên chỉ ghi khi là tên trường hợp lệ của Request (lỗi kiểu của chính trường đó). Kết quả trả bên gọi không đổi.
func TestVetC0KhongGhiTenTruongLa(t *testing.T) {
	w := newWorld(t)
	pr := user("u-dir", "director", "")
	c0 := func(o Outcome) []string { return o.Steps[0].Output }
	for _, name := range []string{`password=\"bimat-C0-khong-luu`, `token=bimat-C0-khong-luu`, "ghi chu tu do bimat-C0-khong-luu"} {
		o := w.pipe.Malformed(name)
		rec, err := traces.SanitizeRecord(recordOf(t, Request{}, o, pr))
		if err != nil {
			t.Fatal(err)
		}
		b, err := json.Marshal(rec)
		if err != nil {
			t.Fatal(err)
		}
		if strings.Contains(string(b), "bimat-C0") {
			t.Errorf("tên trường lạ %q đi vào vết bền: C0 output %q", name, c0(o))
		}
		if !slices.Contains(c0(o), "unknown_fields=1") {
			t.Errorf("C0 phải ghi số trường lạ, được %q", c0(o))
		}
		if !slices.Equal(o.Result.InvalidFields, []string{name}) {
			t.Errorf("kết quả trả bên gọi đổi: invalid_fields = %q", o.Result.InvalidFields)
		}
	}
	if got := c0(w.pipe.Malformed("metric")); !slices.Equal(got, []string{"invalid_fields=metric", "unknown_fields=0"}) {
		t.Errorf("tên trường hợp lệ của Request vẫn ghi nguyên: C0 output %q", got)
	}
}
