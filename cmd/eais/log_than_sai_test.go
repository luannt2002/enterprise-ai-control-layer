package main

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/luannt2002/enterprise-ai-control-layer/execution/commands"
)

// Log của server không mang chữ do bên gọi gõ trong thân bị từ chối (F-035, đường log): tên trường lạ là chữ tự do — dạng mở bằng nháy
// thoát (`password=\"…`) lọt qua che bí mật theo mẫu (observability/traces/sanitize.go, "Giới hạn đã biết"), chữ không có từ khoá thì
// không bị che chút nào. Log chỉ ghi LOẠI lỗi. Kết quả trả bên gọi giữ nguyên (lệnh: invalid_fields mang tên trường lạ).
func TestLogThanSaiKhongGhiChuCuaBenGoi(t *testing.T) {
	e, _ := newCmdEnv(t)
	for name, secret := range map[string]string{`password=\"hunter2bimat`: "hunter2bimat", "ghi chu tudo7bimat": "tudo7bimat"} {
		body, err := json.Marshal(map[string]int{name: 1})
		if err != nil {
			t.Fatal(err)
		}
		r := e.do(http.MethodPost, cmdPath, tokDirector, "application/json", body)
		res := wantResult(t, r, http.StatusBadRequest, commands.StatusFailed, commands.ReasonInvalidRequest, "input", commands.RetryFixRequest)
		if len(res.InvalidFields) != 1 || !strings.Contains(res.InvalidFields[0], secret) { // tên trả về ở dạng thoát của bộ giải mã JSON, như trước
			t.Errorf("kết quả trả bên gọi đổi: invalid_fields = %q, muốn tên trường lạ %q", res.InvalidFields, name)
		}
		if a := e.do(http.MethodPost, "/v1/ask", tokDirector, "application/json", body); a.status != http.StatusBadRequest {
			t.Errorf("/v1/ask trường lạ %q = %d, muốn 400: %s", name, a.status, a.body)
		}
		if l := e.serverLog(); strings.Contains(l, secret) {
			t.Errorf("tên trường lạ %q (chữ của bên gọi) đi vào log của server:\n%s", name, l)
		}
	}
	if l := e.serverLog(); strings.Count(l, "thân lệnh bị từ chối") != 2 || !strings.Contains(l, "trường lạ") {
		t.Errorf("log vẫn phải ghi LOẠI lỗi của mỗi lần thân lệnh bị từ chối:\n%s", l)
	}
}
