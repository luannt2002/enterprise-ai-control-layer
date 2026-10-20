package intent

import (
	"strings"
	"testing"
)

// Ý định chạm dữ liệu/model mà quên khai capability phải bị chặn lúc nạp pack — nếu không,
// router coi capability rỗng là "không cần quyền" và mọi vai (kể cả lái xe) đều đọc được.
func TestDataRouteRequiresCapability(t *testing.T) {
	known := func(string) bool { return true }
	def := func(route, capability string) Def {
		return Def{Name: "x", Route: route, Keywords: []string{"k"}, Capability: capability}
	}
	for _, c := range []struct {
		name    string
		d       Def
		wantErr bool
	}{
		{"SQL thiếu capability", def("SQL_QUERY", ""), true},
		{"RAG thiếu capability", def("RAG", ""), true},
		{"SQL có capability", def("SQL_QUERY", "metrics.read"), false},
		{"chào hỏi (CLARIFICATION) không cần capability", def("CLARIFICATION", ""), false},
		{"từ chối (REJECT) không cần capability", def("REJECT", ""), false},
	} {
		_, err := Compile([]Def{c.d}, known, known)
		if (err != nil) != c.wantErr {
			t.Errorf("%s: lỗi = %v, muốn có lỗi = %v", c.name, err, c.wantErr)
		}
		if c.wantErr && err != nil && !strings.Contains(err.Error(), "không khai capability") {
			t.Errorf("%s: lỗi không nói rõ nguyên nhân: %v", c.name, err)
		}
	}
}

// Mệnh đề trọng tâm: câu nhiều mệnh đề thì mệnh đề chứa cụm đánh dấu là câu hỏi; câu một mệnh đề,
// hoặc không có cụm, thì không thu hẹp. Dấu chấm / phẩy trong số ("3.850", "0,5") không cắt mệnh đề.
func TestFocusClause(t *testing.T) {
	markers := []string{"đúng không"}
	for _, c := range []struct {
		text, want string
		ok         bool
	}{
		{"Hôm qua cả đội báo 4k, báo cáo ghi 4 m3, vậy máy 03 được cấp 4k dầu đúng không?", " vậy máy 03 được cấp 4k dầu đúng không", true},
		{"Máy 03 được cấp 3.850 lít dầu, đúng không?", " đúng không", true},
		{"Máy 03 được cấp 0,5 khối dầu đúng không?", "", false},
		{"Máy 03 được cấp bao nhiêu dầu?", "", false},
	} {
		s, e, ok := FocusClause(c.text, markers)
		if ok != c.ok || (ok && c.text[s:e] != c.want) {
			t.Errorf("%q: được %q (ok=%v), muốn %q (ok=%v)", c.text, c.text[s:e], ok, c.want, c.ok)
		}
	}
}
