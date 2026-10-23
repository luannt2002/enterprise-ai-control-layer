package models

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/luannt2002/enterprise-ai-control-layer/core/contracts"
)

// Embeddings: chia lô, giữ thứ tự (theo index nếu có, theo thứ tự gửi nếu không), chờ theo thời gian nhà cung cấp bảo khi 429,
// chặn phản hồi sai số vector / số chiều / chỉ số. Máy chủ giả trên máy.
func TestEmbeddings(t *testing.T) {
	var lots []int
	fail429, withIndex, reversed, wrongDim := 1, true, false, false
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/embeddings" || r.Header.Get("Authorization") != "Bearer khoa-test" {
			t.Errorf("yêu cầu sai: %s", r.URL.Path)
		}
		if fail429 > 0 {
			fail429--
			w.WriteHeader(429)
			_, _ = io.WriteString(w, `{"error":{"details":[{"retryDelay": "12s"}]}}`)
			return
		}
		var in struct {
			Model string   `json:"model"`
			Input []string `json:"input"`
			Dim   int      `json:"dimensions"`
		}
		b, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(b, &in)
		if in.Model != "m-embed" || in.Dim != 3 {
			t.Errorf("thân sai: %+v", in)
		}
		lots = append(lots, len(in.Input))
		var vs []string
		for i, s := range in.Input {
			d := in.Dim
			if wrongDim {
				d--
			}
			v := make([]string, d)
			for j := range v {
				v[j] = "0"
			}
			v[0] = s[1:] // chữ "t<n>" → vector [n, 0, …]: kiểm thứ tự
			item := `"embedding":[` + strings.Join(v, ",") + `]`
			if withIndex {
				item = fmt.Sprintf(`"index":%d,`, i) + item
			}
			vs = append(vs, "{"+item+"}")
		}
		if reversed {
			for i, j := 0, len(vs)-1; i < j; i, j = i+1, j-1 {
				vs[i], vs[j] = vs[j], vs[i]
			}
		}
		_, _ = io.WriteString(w, `{"data":[`+strings.Join(vs, ",")+`]}`)
	}))
	defer srv.Close()
	var waits []time.Duration
	e, err := NewEmbedder(ServiceConfig{Provider: ProtocolEmbeddings, BaseURL: srv.URL + "/v1/", APIKey: "khoa-test", Model: "m-embed", Dim: 3, MaxRetries: 2})
	if err != nil {
		t.Fatal(err)
	}
	e.Batch, e.sleep = 2, func(_ context.Context, d time.Duration) error { waits = append(waits, d); return nil }
	v, err := e.Embed(context.Background(), []string{"t1", "t2", "t3", "t4", "t5"})
	if err != nil {
		t.Fatal(err)
	}
	if fmt.Sprint(lots) != "[2 2 1]" || len(v) != 5 || v[0][0] != 1 || v[4][0] != 5 || len(v[2]) != 3 {
		t.Errorf("lô %v, vector %v", lots, v)
	}
	if len(waits) != 1 || waits[0] < 12*time.Second {
		t.Errorf("429 có retryDelay 12s: phải chờ ≥ 12s, đã chờ %v", waits)
	}
	if e.Name() != "m-embed/3" {
		t.Errorf("Name = %q", e.Name())
	}
	reversed = true // trả ngược thứ tự nhưng có index: vẫn đúng chỗ
	if v, err := e.Embed(context.Background(), []string{"t1", "t2"}); err != nil || v[0][0] != 1 || v[1][0] != 2 {
		t.Errorf("trả ngược có index: %v %v", v, err)
	}
	reversed, withIndex = false, false // không index: theo thứ tự gửi
	if v, err := e.Embed(context.Background(), []string{"t1", "t2"}); err != nil || v[0][0] != 1 || v[1][0] != 2 {
		t.Errorf("không index: %v %v", v, err)
	}
	wrongDim = true
	if _, err := e.Embed(context.Background(), []string{"t1"}); contracts.FaultOf(err) != contracts.ThirdParty(ProtocolEmbeddings, contracts.KindBadResponse) {
		t.Errorf("sai số chiều: muốn bad_response, được %v", err)
	}
}

// Rerank: điểm trả về theo thứ tự đoạn gửi đi (API trả theo điểm giảm dần); thiếu / lặp / lạ chỉ số là phản hồi hỏng.
func TestRerank(t *testing.T) {
	reply := `{"results":[{"index":2,"relevance_score":0.9},{"index":0,"relevance_score":0.5},{"index":1,"relevance_score":0.1}]}`
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/rerank" || r.Header.Get("Authorization") != "Bearer khoa-test" {
			t.Errorf("yêu cầu sai: %s", r.URL.Path)
		}
		_, _ = io.WriteString(w, reply)
	}))
	defer srv.Close()
	r, err := NewReranker(ServiceConfig{Provider: ProtocolRerank, BaseURL: srv.URL + "/v1", APIKey: "khoa-test", Model: "m-rr"})
	if err != nil {
		t.Fatal(err)
	}
	s, err := r.Rerank(context.Background(), "q", []string{"a", "b", "c"})
	if err != nil || fmt.Sprint(s) != "[0.5 0.1 0.9]" {
		t.Errorf("điểm theo thứ tự đoạn: %v %v", s, err)
	}
	for _, bad := range []string{
		`{"results":[{"index":2,"relevance_score":0.9},{"index":2,"relevance_score":0.5},{"index":1,"relevance_score":0.1}]}`,
		`{"results":[{"index":2,"relevance_score":0.9}]}`,
		`{"results":[{"index":7,"relevance_score":0.9},{"index":0,"relevance_score":0.5},{"index":1,"relevance_score":0.1}]}`,
	} {
		reply = bad
		if _, err := r.Rerank(context.Background(), "q", []string{"a", "b", "c"}); contracts.FaultOf(err) != contracts.ThirdParty(ProtocolRerank, contracts.KindBadResponse) {
			t.Errorf("%s: muốn bad_response, được %v", bad, err)
		}
	}
}

func TestServiceConfig(t *testing.T) {
	t.Setenv("EAIS_EMBED_PROVIDER", ProtocolEmbeddings)
	t.Setenv("EAIS_EMBED_MODEL", "m")
	t.Setenv("EAIS_EMBED_API_KEY", "k")
	t.Setenv("EAIS_EMBED_BASE_URL", "https://embed.example/v1")
	t.Setenv("EAIS_EMBED_DIM", "768")
	if _, err := NewEmbedder(ServiceFromEnv("EMBED")); err != nil {
		t.Fatalf("cấu hình đủ: %v", err)
	}
	ok := ServiceConfig{Provider: ProtocolEmbeddings, Model: "m", APIKey: "k", Dim: 3, BaseURL: "https://embed.example/v1"}
	for name, mut := range map[string]func(*ServiceConfig){
		"giao thức lạ":     func(c *ServiceConfig) { c.Provider = "khac" },
		"thiếu khoá":       func(c *ServiceConfig) { c.APIKey = "" },
		"thiếu model":      func(c *ServiceConfig) { c.Model = "" },
		"thiếu endpoint":   func(c *ServiceConfig) { c.BaseURL = "" },
		"thiếu số chiều":   func(c *ServiceConfig) { c.Dim = 0 },
		"http ngoài máy":   func(c *ServiceConfig) { c.BaseURL = "http://embed.example/v1" },
		"thử lại viết sai": func(c *ServiceConfig) { c.MaxRetries = -1 },
	} {
		c := ok
		mut(&c)
		if _, err := NewEmbedder(c); contracts.FaultOf(err) != contracts.FaultCode {
			t.Errorf("%s: muốn lỗi cấu hình, được %v", name, err)
		}
	}
	t.Setenv("EAIS_EMBED_DIM", "nhiều")
	if _, err := NewEmbedder(ServiceFromEnv("EMBED")); err == nil {
		t.Error("EAIS_EMBED_DIM viết sai phải là lỗi cấu hình")
	}
	if _, err := NewReranker(ServiceConfig{Provider: ProtocolRerank, Model: "m", BaseURL: "https://rr.example/v1"}); contracts.FaultOf(err) != contracts.FaultCode {
		t.Errorf("reranker thiếu khoá: %v", err)
	}
}
