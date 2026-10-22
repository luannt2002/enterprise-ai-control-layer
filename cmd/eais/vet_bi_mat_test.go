package main

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/luannt2002/enterprise-ai-control-layer/core/contracts"
)

// R2-03 vòng 2 (F1, S1): bí mật có nháy kép trong câu hỏi nhiều mệnh đề không được lọt vào vết mà CHÍNH người hỏi đọc lại qua HTTP — cả JSON lẫn
// bảng chữ (?format=table). Chạy trên hai kho (bộ nhớ luôn chạy; PG chỉ khi có EAIS_PG_*), như bộ test lớp HTTP.
func TestServerVetKhongLoBiMatTrongCauNhieuMenhDe(t *testing.T) {
	t.Run("bo-nho", func(t *testing.T) { vetKhongLoBiMat(t, newEnv(t, newMemStore())) })
	t.Run("pg", func(t *testing.T) {
		st, _ := openPG(t)
		vetKhongLoBiMat(t, newEnv(t, st))
	})
}

func vetKhongLoBiMat(t *testing.T, e *env) {
	t.Helper()
	pieces := []struct{ piece, value string }{
		{`{"api_key": "abc123xyz"}`, "abc123xyz"},
		{`password="hunter2pw"`, "hunter2pw"},
		{`{"secret":"s3cr3tVal"}`, "s3cr3tVal"},
		{`token: "tok3nVal"`, "tok3nVal"},
		{`Cookie: sid=cookieVal99`, "cookieVal99"},
		{`Authorization: "Bearer bearerVal123456"`, "bearerVal123456"},
		{`token: "abc. tailSecret77"`, "tailSecret77"}, // dấu chấm câu trong bí mật cắt mệnh đề hỏi giữa giá trị
	}
	for _, p := range pieces {
		for place, q := range map[string]string{
			"sau từ khoá":   "đội báo 4k, báo cáo ghi 4 m3, vậy EX-03 được cấp 4k dầu " + p.piece + " đúng không?",
			"trước từ khoá": "đội báo 4k, báo cáo ghi 4 m3, vậy EX-03 " + p.piece + " được cấp 4k dầu đúng không?",
		} {
			for _, tok := range []string{tokDirector, tokEng} {
				r := e.ask(tok, q)
				if r.status != http.StatusOK {
					t.Fatalf("ask %q = %d: %s", q, r.status, r.body)
				}
				resp := decodeAs[contracts.AskResponse](t, r)
				b, _ := json.Marshal(e.getTrace(tok, resp.TraceID))
				if strings.Contains(string(b), p.value) {
					t.Errorf("%s / %s / %s: %q còn trong JSON của vết: %s", p.piece, place, tok, p.value, b)
				}
				tab := e.do(http.MethodGet, "/v1/trace/"+resp.TraceID+"?format=table", tok, "", nil)
				if tab.status != http.StatusOK || strings.Contains(string(tab.body), p.value) {
					t.Errorf("%s / %s / %s: bảng chữ = %d, còn %q:\n%s", p.piece, place, tok, tab.status, p.value, tab.body)
				}
			}
		}
	}
}
