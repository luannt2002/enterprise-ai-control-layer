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
	"strings"
	"sync/atomic"
	"testing"

	"github.com/luannt2002/enterprise-ai-control-layer/intelligence/models"
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
	for k, v := range map[string]string{"EAIS_MODEL_PROVIDER": models.ProtocolMessages, "EAIS_MODEL": contractModel, "EAIS_MODEL_API_KEY": "khoa-gia-khong-that",
		"EAIS_MODEL_BASE_URL": officialEndpoint, "EAIS_MODEL_THINKING": contractThinking, "EAIS_MODEL_EFFORT": contractEffort} {
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

// Mọi cách viết khác của CÙNG máy chủ chính thức (cổng :443, chữ hoa, dấu chấm cuối tên máy) và mọi endpoint không phải
// loopback (proxy có thể chuyển tiếp và tính tiền) là lượt CÓ THỂ TỐN TIỀN: phải qua kiểm trước lời gọi model như endpoint
// chính thức, không được rơi vào nhánh chạy khô rồi gọi hết các ca (soát C3, C3-04: "https://api.anthropic.com:443" ở lượt
// chính k = 3 từng gọi 372 lần, bằng chứng ghi dry_run=true nên vô dụng).
func TestEndpointTonTienVietKhacVanBiKiemTruoc(t *testing.T) {
	data := genData(t)
	for _, u := range []string{"https://api.anthropic.com:443", "https://API.ANTHROPIC.COM", "HTTPS://api.anthropic.com",
		"https://api.anthropic.com.", "https://Api.Anthropic.Com.:443/", "https://gateway.example.com"} {
		t.Run(u, func(t *testing.T) {
			calls := fakeOfficial(t)
			t.Setenv("EAIS_MODEL_BASE_URL", u)
			cfg := baseCfg(data)
			cfg.k, cfg.out = 3, filepath.Join(t.TempDir(), "out")
			if code := run(cfg); code != 2 || calls.Load() != 0 {
				t.Errorf("mã %d, %d lời gọi model; muốn 2 và 0 lời gọi (lượt chính bị từ chối trước khi tiêu tiền)", code, calls.Load())
			}
		})
	}
	// Đối chứng dương: cách viết khác của endpoint chính thức là endpoint CHÍNH THỨC (thử khói đạt ra mã 0, không phải mã 2
	// của chạy khô), còn proxy thì vẫn là chạy khô theo hợp đồng §4 (mã 2) dù đã qua kiểm trước.
	for u, want := range map[string]int{"https://api.anthropic.com:443": 0, "https://API.ANTHROPIC.COM.": 0, "https://gateway.example.com": 2,
		"https://api.anthropic.com:8443": 2} {
		t.Run("thử khói "+u, func(t *testing.T) {
			calls := fakeOfficial(t)
			t.Setenv("EAIS_MODEL_BASE_URL", u)
			cfg := baseCfg(data)
			cfg.smoke = 1
			if code := run(cfg); code != want || calls.Load() == 0 {
				t.Errorf("mã %d, %d lời gọi; muốn %d và có lời gọi", code, calls.Load(), want)
			}
		})
	}
	// Proxy có hệ cổng bị chặn (B1 thiếu -manifest): kết luận chắc chắn KHÔNG ĐẠT nên dừng trước lời gọi model như lượt thật.
	t.Run("proxy, pilot có hệ cổng bị chặn", func(t *testing.T) {
		calls := fakeOfficial(t)
		t.Setenv("EAIS_MODEL_BASE_URL", "https://gateway.example.com")
		cfg := baseCfg(data)
		cfg.pilot, cfg.k, cfg.systems, cfg.out = true, 5, sysB2Plus+","+sysB1, filepath.Join(t.TempDir(), "out")
		if code := run(cfg); code != 2 || calls.Load() != 0 {
			t.Errorf("mã %d, %d lời gọi model; muốn 2 và 0 lời gọi", code, calls.Load())
		}
	})
}

// Thử khói trên endpoint tốn tiền cũng phải kiểm trước điều chắc chắn làm hỏng nó: -out không ghi được (trước đây gọi model
// rồi mới "LỖI ghi bằng chứng", mất cả bằng chứng của lần thử) và N lớn hơn cỡ thử khói của hợp đồng (-smoke 3; -smoke 1000
// từng chạy hết các ca, 124 lời gọi, vẫn THỬ KHÓI: ĐẠT) — soát C3, C3-06.
func TestThuKhoiTonTienKiemTruoc(t *testing.T) {
	data := genData(t)
	ro := t.TempDir()
	if err := os.Chmod(ro, 0o555); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(ro, 0o755) })
	for _, c := range []struct {
		name string
		set  func(*config)
	}{
		{"-out chỉ đọc", func(c *config) { c.smoke, c.out = 1, filepath.Join(ro, "out") }},
		{"N vượt cỡ thử khói của hợp đồng", func(c *config) { c.smoke = contractSmoke + 1 }},
	} {
		t.Run(c.name, func(t *testing.T) {
			calls := fakeOfficial(t)
			cfg := baseCfg(data)
			c.set(&cfg)
			if code := run(cfg); code != 2 || calls.Load() != 0 {
				t.Errorf("mã %d, %d lời gọi model; muốn 2 và 0 lời gọi", code, calls.Load())
			}
		})
	}
}

// Lượt chính thiếu một hệ cổng trong -systems thì từ chối trước lời gọi model: gatePass đòi mọi hệ cổng nên lượt đó chắc
// chắn KHÔNG ĐẠT (mã 1, hợp đồng coi là FAIL, cấm chạy lại cùng bộ giữ lại) vì quên cờ chứ không vì EAIS — soát C3, C3-07
// (tiềm ẩn tới khi bộ chấm đóng băng, nên test gọi thẳng preflightReal thay vì run()).
func TestLuotChinhThieuHeCongBiTuChoi(t *testing.T) {
	fakeOfficial(t)
	gw, err := models.New(models.ConfigFromEnv())
	if err != nil {
		t.Fatal(err)
	}
	missing := func(reasons []string) bool {
		for _, r := range reasons {
			if strings.Contains(r, "thiếu hệ cổng") && strings.Contains(r, sysB1) {
				return true
			}
		}
		return false
	}
	cfg := baseCfg(t.TempDir())
	cfg.k, cfg.out = 3, filepath.Join(t.TempDir(), "out")
	if r := preflightReal(cfg, gw); !missing(r) {
		t.Errorf("lượt chính -systems %s: lý do từ chối %q; muốn có lý do thiếu hệ cổng %s", cfg.systems, r, sysB1)
	}
	cfg.systems = sysB2Plus + "," + sysB1
	if r := preflightReal(cfg, gw); missing(r) {
		t.Errorf("đủ hệ cổng mà vẫn bị báo thiếu: %q", r)
	}
	cfg.systems, cfg.pilot, cfg.k = sysB2Plus, true, 5 // pilot được chạy một phần hệ (hiệu chỉnh bộ chấm)
	if r := preflightReal(cfg, gw); missing(r) {
		t.Errorf("pilot bị đòi đủ hệ cổng: %q", r)
	}
}

// Bằng chứng chỉ ghi scheme://host của endpoint: khoá đặt trong path / query của URL không được nằm trong tệp bằng chứng
// (manifest.json được ghi nguyên văn endpoint, có thể được commit) — soát C3, C3-08.
func TestBangChungKhongGhiBiMatTrongURL(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.WriteString(w, `{"content":[{"type":"text","text":"{\"answer\": \"x\", \"evidence_ids\": []}"}],"usage":{"input_tokens":10,"output_tokens":5}}`)
	}))
	defer srv.Close()
	clearModelEnv(t)
	for k, v := range map[string]string{"EAIS_MODEL_PROVIDER": models.ProtocolMessages, "EAIS_MODEL": contractModel, "EAIS_MODEL_API_KEY": "k",
		"EAIS_MODEL_BASE_URL": srv.URL + "/BIMATPATH/?api_key=BIMATQUERY"} {
		t.Setenv(k, v)
	}
	cfg := baseCfg(genData(t))
	cfg.smoke, cfg.out = 1, t.TempDir()
	run(cfg)
	ents, err := os.ReadDir(cfg.out)
	if err != nil || len(ents) == 0 {
		t.Fatalf("không có bằng chứng: %v", err)
	}
	for _, e := range ents {
		b, err := os.ReadFile(filepath.Join(cfg.out, e.Name()))
		if err != nil {
			t.Fatal(err)
		}
		if strings.Contains(string(b), "BIMAT") {
			t.Errorf("%s chứa phần bí mật của URL", e.Name())
		}
		if e.Name() == "manifest.json" && !strings.Contains(string(b), `"endpoint": "`+srv.URL+`"`) {
			t.Errorf("manifest.json không ghi scheme://host %s", srv.URL)
		}
	}
}

// Lượt chính bị từ chối vì bộ chấm chưa đóng băng kể cả khi ĐỦ hệ cổng: kiểm riêng lý do này, vì kiểm thiếu hệ cổng (C3-07) che mất ca
// "lượt chính khi bộ chấm chưa đóng băng" của TestLuotThatTuChoiTruocLoiGoiModel (đột biến PF3 của n30 sống sau 246eabb).
func TestLuotChinhBoChamChuaDongBangLaLyDoRieng(t *testing.T) {
	fakeOfficial(t)
	gw, err := models.New(models.ConfigFromEnv())
	if err != nil {
		t.Fatal(err)
	}
	cfg := baseCfg(t.TempDir())
	cfg.k, cfg.out, cfg.systems = 3, filepath.Join(t.TempDir(), "out"), sysB2Plus+","+sysB1
	found := false
	for _, r := range preflightReal(cfg, gw) {
		found = found || strings.Contains(r, "bộ chấm nội dung chưa đóng băng")
	}
	if !found {
		t.Errorf("lượt chính đủ hệ cổng khi bộ chấm chưa đóng băng: thiếu lý do từ chối")
	}
}
