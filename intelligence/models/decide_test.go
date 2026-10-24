package models

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/luannt2002/enterprise-ai-control-layer/core/contracts"
)

// Quyết định có kiểu: gửi đúng khuôn (model, state, câu chọn một kèm mô tả nhãn / mệnh đề có-không), đọc xác suất; chặn phản hồi thiếu
// nhãn, thừa nhãn, không cộng thành 1, ngoài [0, 1], thiếu mệnh đề; thử lại khi 429. Máy chủ giả trên máy.
func TestDecide(t *testing.T) {
	var reply string
	fail429 := 1
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var in struct {
			Model     string                    `json:"model"`
			State     any                       `json:"state"`
			Questions map[string]map[string]any `json:"questions"`
		}
		b, _ := io.ReadAll(r.Body)
		if err := json.Unmarshal(b, &in); err != nil || r.URL.Path != "/v9/quyet" || r.Header.Get("Authorization") != "Bearer khoa-test" || in.Model != "m-1.0" || in.State == nil {
			t.Errorf("yêu cầu sai: %s %s", r.URL.Path, b)
		}
		for _, q := range in.Questions {
			if q["type"] == "choice" && q["criteria"] == nil || q["instructions"] == "" {
				t.Errorf("câu hỏi thiếu nhãn / hướng dẫn: %v", q)
			}
		}
		if fail429 > 0 {
			fail429--
			w.WriteHeader(429)
			return
		}
		_, _ = io.WriteString(w, reply)
	}))
	defer srv.Close()
	d, err := NewDecider(ServiceConfig{Provider: ProtocolDecide, BaseURL: srv.URL + "/v9/quyet", APIKey: "khoa-test", Model: "m-1.0", MaxRetries: 1})
	if err != nil {
		t.Fatal(err)
	}
	d.sleep = func(context.Context, time.Duration) error { return nil }
	nhan := map[string]string{"so_lieu": "hỏi số", "quy_trinh": "hỏi quy trình"}

	reply = `{"model":"m-1.0","answers":{"q":{"type":"choice","choice":"quy_trinh","probabilities":{"so_lieu":0.2,"quy_trinh":0.8}}}}`
	p, m, err := d.Choice(context.Background(), "câu hỏi", "phân loại", nhan)
	if err != nil || p["quy_trinh"] != 0.8 || m != "m-1.0" || fail429 != 0 {
		t.Fatalf("Choice: %v %v %q (429 còn %d)", p, err, m, fail429)
	}
	for _, sai := range []string{
		`{"answers":{"q":{"probabilities":{"so_lieu":1}}}}`,                           // thiếu nhãn
		`{"answers":{"q":{"probabilities":{"so_lieu":0.5,"quy_trinh":0.3,"x":0.2}}}}`, // thừa nhãn
		`{"answers":{"q":{"probabilities":{"so_lieu":0.5,"quy_trinh":0.3}}}}`,         // không cộng thành 1
		`{"answers":{"q":{"probabilities":{"so_lieu":1.5,"quy_trinh":-0.5}}}}`,        // ngoài [0, 1]
		`{"answers":{}}`,
	} {
		reply = sai
		if _, _, err := d.Choice(context.Background(), "câu hỏi", "phân loại", nhan); contracts.FaultOf(err) != contracts.ThirdParty(ProtocolDecide, contracts.KindBadResponse) {
			t.Errorf("%s: muốn bad_response, được %v", sai, err)
		}
	}

	reply = `{"model":"m-1.0","answers":{"du":{"type":"noul","noul":0.91},"lien_quan":{"type":"noul","noul":0.4}}}`
	pp, _, err := d.Propositions(context.Background(), map[string]string{"query": "a", "passage": "b"}, map[string]string{"du": "đủ căn cứ?", "lien_quan": "liên quan?"})
	if err != nil || pp["du"] != 0.91 || pp["lien_quan"] != 0.4 {
		t.Fatalf("Propositions: %v %v", pp, err)
	}
	reply = `{"answers":{"du":{"noul":0.91}}}`
	if _, _, err := d.Propositions(context.Background(), "x", map[string]string{"du": "?", "lien_quan": "?"}); !strings.Contains(err.Error(), "lien_quan") {
		t.Errorf("thiếu mệnh đề phải lỗi, được %v", err)
	}
	if _, err := NewDecider(ServiceConfig{Provider: ProtocolDecide, BaseURL: srv.URL, APIKey: "k"}); contracts.FaultOf(err) != contracts.FaultCode {
		t.Errorf("thiếu model phải là lỗi cấu hình: %v", err)
	}
}
