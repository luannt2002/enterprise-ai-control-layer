package main

import (
	"context"
	"crypto/tls"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
)

// fakeOfficial cho mọi kết nối HTTP (kể cả tới api.anthropic.com:443) vào một máy chủ TLS cục bộ, nên run() đi đúng
// đường endpoint CHÍNH THỨC (không chạy khô) mà không cần mạng hay khoá thật. Trước đây đường này bị coi là "không
// tới được trong test", nên các chỗ nối quyết định trong run() có thể bị phá mà không test nào đỏ (soát C2, F-05).
func fakeOfficial(t *testing.T) *atomic.Int64 {
	t.Helper()
	var n atomic.Int64
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n.Add(1)
		_, _ = io.WriteString(w, `{"content":[{"type":"text","text":"{\"answer\": \"trả lời giả\", \"evidence_ids\": []}"}],"stop_reason":"end_turn","usage":{"input_tokens":10,"output_tokens":5}}`)
	}))
	t.Cleanup(srv.Close)
	old := http.DefaultTransport
	http.DefaultTransport = &http.Transport{
		DialContext: func(ctx context.Context, network, _ string) (net.Conn, error) {
			var d net.Dialer
			return d.DialContext(ctx, network, srv.Listener.Addr().String())
		},
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true}, // chứng chỉ của máy chủ cục bộ; chỉ trong test
	}
	t.Cleanup(func() { http.DefaultTransport = old })
	clearModelEnv(t)
	for k, v := range map[string]string{"EAIS_MODEL_PROVIDER": "anthropic", "EAIS_MODEL": contractModel, "EAIS_MODEL_API_KEY": "khoa-gia-khong-that",
		"EAIS_MODEL_THINKING": contractThinking, "EAIS_MODEL_EFFORT": contractEffort} {
		t.Setenv(k, v)
	}
	return &n
}

// Lượt chạy THẬT mà chắc chắn không thành bằng chứng G7 thì phải bị từ chối TRƯỚC lời gọi model đầu tiên, không tiêu
// tiền rồi mới in CHẶN (soát C2, F-02, F-03, F-05): bộ chấm chưa đóng băng ở lượt chính, thiếu / không ghi được -out,
// tham số lệch hợp đồng §3–§4, hệ cổng bị chặn. Thử khói đúng hợp đồng thì vẫn chạy (đối chứng dương).
func TestLuotThatTuChoiTruocLoiGoiModel(t *testing.T) {
	data := genData(t)
	file := filepath.Join(t.TempDir(), "la-tep")
	if err := os.WriteFile(file, []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	for _, c := range []struct {
		name string
		env  map[string]string
		set  func(*config)
	}{
		{"lượt chính khi bộ chấm chưa đóng băng", nil, func(c *config) { c.k = 3 }},
		{"pilot thiếu -out", nil, func(c *config) { c.pilot, c.k, c.out = true, 5, "" }},
		{"pilot -out là một tệp", nil, func(c *config) { c.pilot, c.k, c.out = true, 5, file }},
		{"pilot lệch max_tokens", map[string]string{"EAIS_MODEL_MAX_TOKENS": "64"}, func(c *config) { c.pilot, c.k = true, 5 }},
		{"pilot có temperature", map[string]string{"EAIS_MODEL_TEMPERATURE": "0.5"}, func(c *config) { c.pilot, c.k = true, 5 }},
		{"pilot lệch số lần thử lại", map[string]string{"EAIS_MODEL_MAX_RETRIES": "0"}, func(c *config) { c.pilot, c.k = true, 5 }},
		{"pilot lệch top-k", nil, func(c *config) { c.pilot, c.k, c.topK = true, 5, 3 }},
		{"pilot chấm full", nil, func(c *config) { c.pilot, c.k, c.grade = true, 5, gradeFull }},
		{"pilot có hệ cổng bị chặn (B1 thiếu -manifest)", nil, func(c *config) { c.pilot, c.k, c.systems = true, 5, sysB2Plus+","+sysB1 }},
	} {
		t.Run(c.name, func(t *testing.T) {
			calls := fakeOfficial(t)
			for k, v := range c.env {
				t.Setenv(k, v)
			}
			cfg := baseCfg(data)
			cfg.out = filepath.Join(t.TempDir(), "out")
			c.set(&cfg)
			if code := run(cfg); code != 2 || calls.Load() != 0 {
				t.Errorf("mã %d, %d lời gọi model; muốn 2 và 0 lời gọi (từ chối trước khi tiêu tiền)", code, calls.Load())
			}
		})
	}
	t.Run("đối chứng dương: thử khói đúng hợp đồng", func(t *testing.T) {
		calls := fakeOfficial(t)
		cfg := baseCfg(data)
		cfg.smoke = 1
		if code := run(cfg); code != 0 || calls.Load() == 0 {
			t.Errorf("mã %d, %d lời gọi; muốn 0 và có lời gọi tới endpoint chính thức", code, calls.Load())
		}
	})
}
