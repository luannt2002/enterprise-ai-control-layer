package main

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"
	"unicode"

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

// N1 (soát vòng 2): giá trị bí mật mà bộ chuẩn hoá đọc thành con số, ngày hay chữ (kể cả khi bị định dạng lại "48.291.637" hay đổi sang ngày ISO) không được lọt
// vào LÝ DO và PHƯƠNG ÁN BỊ LOẠI của vết mà chính người hỏi đọc lại qua HTTP — JSON lẫn bảng chữ (?format=table). Cố ý chỉ kiểm hai phần đó: chữ số của bí mật dạng
// khoá=số nằm sẵn ở decision.normalized và Output của S2 trên main (traces.Sanitize chưa che, việc riêng), ngoài phạm vi của vết giải thích.
func TestServerVetKhongLoGiaTriBiMatDangSo(t *testing.T) {
	t.Run("bo-nho", func(t *testing.T) { vetKhongLoGiaTriBiMat(t, newEnv(t, newMemStore())) })
	t.Run("pg", func(t *testing.T) {
		st, _ := openPG(t)
		vetKhongLoGiaTriBiMat(t, newEnv(t, st))
	})
}

// squashAlnum chỉ giữ chữ và số (hạ chữ thường): "48.291.637" và "48291637" thành cùng một chuỗi.
func squashAlnum(s string) string {
	var b strings.Builder
	for _, r := range strings.ToLower(s) {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			b.WriteRune(r)
		}
	}
	return b.String()
}

// tableWhyLines lấy từ bảng chữ của vết các dòng "lý do:" và "loại:" (kể cả dòng nối tiếp), bỏ "vào:" và "ra:".
func tableWhyLines(tab string) []string {
	var out []string
	cur := ""
	for _, l := range strings.Split(tab, "\n") {
		switch {
		case strings.HasPrefix(l, "    lý do:"):
			cur = "lý do"
		case strings.HasPrefix(l, "    loại:"):
			cur = "loại"
		case strings.HasPrefix(l, "    vào:"), strings.HasPrefix(l, "    ra:"), strings.HasPrefix(l, "    bằng chứng:"), strings.HasPrefix(l, "    lỗi:"):
			cur = ""
		case !strings.HasPrefix(l, "     "): // dòng nối tiếp có thụt lề hơn nhãn; dòng khác (tiêu đề bước, trống) kết thúc khối
			cur = ""
		}
		if cur != "" {
			out = append(out, l)
		}
	}
	return out
}

func vetKhongLoGiaTriBiMat(t *testing.T, e *env) {
	t.Helper()
	pieces := []struct {
		piece string
		forms []string
	}{
		{"password=4123 L", []string{"4123"}},
		{"password=48291637 L", []string{"48291637"}},
		{`{"token": "48291637"} L`, []string{"48291637"}},
		{"secret=48.291.637 L", []string{"48291637"}},
		{"password=07/11/2031", []string{"2031-11-07", "07/11/2031"}},
		{"ｐａｓｓｗｏｒｄ＝zq7771secret", []string{"zq7771secret"}},
	}
	leaks := func(lines, forms []string) []string {
		var out []string
		for _, l := range lines {
			for _, f := range forms {
				if strings.Contains(squashAlnum(l), squashAlnum(f)) {
					out = append(out, l)
					break
				}
			}
		}
		return out
	}
	asked := 0
	for _, p := range pieces {
		for _, tmpl := range []string{"EX-03 hôm nay được cấp %s đúng không?", "EX-03 %s được cấp bao nhiêu dầu?"} {
			q := strings.Replace(tmpl, "%s", p.piece, 1)
			for _, tok := range []string{tokDirector, tokEng} {
				r := e.ask(tok, q)
				if r.status != http.StatusOK {
					t.Fatalf("ask %q = %d: %s", q, r.status, r.body)
				}
				asked++
				resp := decodeAs[contracts.AskResponse](t, r)
				var lines []string
				for _, s := range e.getTrace(tok, resp.TraceID).Record.Steps {
					lines = append(lines, s.Reason)
					lines = append(lines, s.Rejected...)
				}
				if bad := leaks(lines, p.forms); len(bad) > 0 {
					t.Errorf("%q / %s: %v còn trong lý do hay phương án bị loại của JSON: %q", q, tok, p.forms, bad)
				}
				tab := e.do(http.MethodGet, "/v1/trace/"+resp.TraceID+"?format=table", tok, "", nil)
				if tab.status != http.StatusOK {
					t.Fatalf("bảng chữ = %d", tab.status)
				}
				tl := tableWhyLines(string(tab.body))
				if len(tl) == 0 {
					t.Fatalf("%q / %s: không đọc được dòng lý do / loại nào từ bảng chữ:\n%s", q, tok, tab.body)
				}
				if bad := leaks(tl, p.forms); len(bad) > 0 {
					t.Errorf("%q / %s: %v còn trong dòng lý do / loại của bảng chữ: %q", q, tok, p.forms, bad)
				}
			}
		}
	}
	if asked != len(pieces)*2*2 {
		t.Errorf("đã hỏi %d lượt, muốn %d", asked, len(pieces)*2*2)
	}
}
