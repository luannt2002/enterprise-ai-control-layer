package models

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/luannt2002/enterprise-ai-systems/core/contracts"
)

// okReply là câu trả lời hợp lệ của máy chủ giả: JSON answer + id bằng chứng.
const okReply = `{"content":[{"type":"text","text":"{\"answer\": \"Lập biên bản trong vòng 24 giờ.\", \"evidence_ids\": [\"SOP#0\"]}"}],"usage":{"input_tokens":10,"output_tokens":5}}`

var paramsReq = Request{Question: "Biên bản lập trong bao lâu?", Evidence: []contracts.Evidence{{ID: "SOP#0", Text: "Lập biên bản trong vòng 24 giờ."}}}

// rawBodies dựng máy chủ giả ghi lại thân từng yêu cầu dưới dạng map (để thấy khoá nào có mặt) và
// trả lời theo hàm reply.
func rawBodies(t *testing.T, reply func(body map[string]any) (int, string)) (*httptest.Server, *[]map[string]any) {
	t.Helper()
	var got []map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		var m map[string]any
		if err := json.Unmarshal(b, &m); err != nil {
			t.Errorf("thân yêu cầu không phải JSON: %v", err)
		}
		got = append(got, m)
		status, body := reply(m)
		w.WriteHeader(status)
		_, _ = io.WriteString(w, body)
	}))
	t.Cleanup(srv.Close)
	return srv, &got
}

// Mặc định KHÔNG gửi tham số sampling nào: model hiện hành trả 400 với mọi temperature / top_p /
// top_k khác mặc định (ADR-0005 §1), nên client gửi "temperature": 0 là hỏng mọi lời gọi.
func TestMessagesMacDinhKhongGuiSampling(t *testing.T) {
	srv, got := rawBodies(t, func(map[string]any) (int, string) { return http.StatusOK, okReply })
	m := &Messages{Provider: "anthropic", BaseURL: srv.URL, APIKey: "k", Model: "m", MaxTokens: 64}
	if _, err := m.Generate(context.Background(), paramsReq); err != nil {
		t.Fatal(err)
	}
	for _, k := range []string{"temperature", "top_p", "top_k"} {
		if _, ok := (*got)[0][k]; ok {
			t.Errorf("thân yêu cầu có %q = %v; mặc định không được gửi tham số sampling", k, (*got)[0][k])
		}
	}
}

// Provider từ chối temperature (như model hiện hành): client cấu hình mặc định vẫn trả lời được.
func TestMessagesProviderTuChoiTemperature(t *testing.T) {
	srv, _ := rawBodies(t, func(b map[string]any) (int, string) {
		if _, ok := b["temperature"]; ok {
			return http.StatusBadRequest, `{"type":"error","error":{"type":"invalid_request_error","message":"temperature is not supported for this model"}}`
		}
		return http.StatusOK, okReply
	})
	m := &Messages{Provider: "anthropic", BaseURL: srv.URL, APIKey: "k", Model: "m", MaxTokens: 64}
	d, err := m.Generate(context.Background(), paramsReq)
	if err != nil || d.Text == "" {
		t.Fatalf("provider từ chối temperature: muốn trả lời được, được %+v (lỗi %v)", d, err)
	}
}

// Tham số đã cấu hình được gửi đúng hình dạng của API (tài liệu nhà cung cấp, tra 20/10/2026):
// temperature là số (kể cả 0 — con trỏ khác nil), thinking = {"type": …}, effort nằm trong
// output_config. Provider NHẬN temperature thì client vẫn gửi được khi được cấu hình.
func TestMessagesGuiThamSoDaCauHinh(t *testing.T) {
	srv, got := rawBodies(t, func(map[string]any) (int, string) { return http.StatusOK, okReply })
	zero := 0.0
	m := &Messages{Provider: "anthropic", BaseURL: srv.URL, APIKey: "k", Model: "m", MaxTokens: 64,
		Temperature: &zero, Thinking: "between_tools", Effort: "medium"}
	if _, err := m.Generate(context.Background(), paramsReq); err != nil {
		t.Fatal(err)
	}
	b := (*got)[0]
	if v, ok := b["temperature"]; !ok || v != 0.0 {
		t.Errorf("temperature = %v (có mặt: %v), muốn 0", v, ok)
	}
	if th, _ := b["thinking"].(map[string]any); th["type"] != "between_tools" || len(th) != 1 {
		t.Errorf("thinking = %v, muốn {\"type\": \"between_tools\"}", b["thinking"])
	}
	if oc, _ := b["output_config"].(map[string]any); oc["effort"] != "medium" || len(oc) != 1 {
		t.Errorf("output_config = %v, muốn {\"effort\": \"medium\"}", b["output_config"])
	}
	for _, k := range []string{"top_p", "top_k", "effort"} {
		if _, ok := b[k]; ok {
			t.Errorf("thân yêu cầu có khoá %q ở cấp đầu", k)
		}
	}
}

// Lỗi tạm (429, 5xx) được thử lại có lùi, mỗi lần gửi đều có vết; Calls đếm MỘT lời gọi logic.
// Lỗi cấu hình (4xx khác) không thử lại. Hết lượt thử thì trả lỗi đúng nhãn của lần cuối.
func TestMessagesThuLai(t *testing.T) {
	seq := []int{http.StatusTooManyRequests, http.StatusServiceUnavailable, http.StatusOK}
	n := 0
	srv, _ := rawBodies(t, func(map[string]any) (int, string) {
		s := seq[min(n, len(seq)-1)]
		n++
		if s != http.StatusOK {
			return s, `{"type":"error"}`
		}
		return s, okReply
	})
	var traces []MessagesTrace
	m := &Messages{Provider: "anthropic", BaseURL: srv.URL, APIKey: "khoa-bi-mat", Model: "m", MaxTokens: 64,
		MaxRetries: 2, Backoff: time.Millisecond, Trace: func(tr MessagesTrace) { traces = append(traces, tr) }}
	d, err := m.Generate(context.Background(), paramsReq)
	if err != nil || d.Text == "" {
		t.Fatalf("429 → 503 → 200 với 2 lần thử lại: muốn trả lời được, được %+v (lỗi %v)", d, err)
	}
	if len(traces) != 3 || m.Calls() != 1 {
		t.Fatalf("muốn 3 lần gửi có vết, 1 lời gọi logic; được %d vết, Calls = %d", len(traces), m.Calls())
	}
	for i, tr := range traces {
		if tr.Attempt != i+1 || tr.Status != seq[i] || (tr.Err == "") != (seq[i] == http.StatusOK) || len(tr.Request) == 0 {
			t.Errorf("vết %d = {Attempt %d, Status %d, Err %q}, muốn lần %d, mã %d", i, tr.Attempt, tr.Status, tr.Err, i+1, seq[i])
		}
		if strings.Contains(string(tr.Request), "khoa-bi-mat") {
			t.Errorf("vết %d chứa khoá trong thân yêu cầu", i)
		}
	}

	// 400: không thử lại dù còn lượt.
	seq, n, traces = []int{http.StatusBadRequest}, 0, nil
	if _, err := m.Generate(context.Background(), paramsReq); contracts.FaultOf(err) != contracts.FaultCode || len(traces) != 1 {
		t.Errorf("400: muốn lỗi cấu hình sau 1 lần gửi, được %v sau %d lần", err, len(traces))
	}

	// 503 mãi: 1 + MaxRetries lần gửi rồi trả lỗi "không phục vụ được".
	seq, n, traces = []int{http.StatusServiceUnavailable}, 0, nil
	if _, err := m.Generate(context.Background(), paramsReq); contracts.FaultOf(err) != contracts.ThirdParty("anthropic", contracts.KindUnavailable) || len(traces) != 3 {
		t.Errorf("503 mãi: muốn lỗi Unavailable sau 3 lần gửi, được %v sau %d lần", err, len(traces))
	}
}

// Cấu hình sai bị chặn lúc khởi động; ConfigFromEnv đọc đủ biến mới, thử lại mặc định 2.
func TestNewKiemThamSo(t *testing.T) {
	base := Config{Provider: "anthropic", Model: "m", APIKey: "k"}
	for _, c := range []Config{
		{Temperature: "không"}, {Thinking: "enabled"}, {Effort: "rất cao"}, {MaxRetries: -1},
	} {
		c.Provider, c.Model, c.APIKey = base.Provider, base.Model, base.APIKey
		if _, err := New(c); contracts.FaultOf(err) != contracts.FaultCode {
			t.Errorf("%+v: muốn lỗi cấu hình, được %v", Config{Temperature: c.Temperature, Thinking: c.Thinking, Effort: c.Effort, MaxRetries: c.MaxRetries}, err)
		}
	}
	g, err := New(Config{Provider: "anthropic", Model: "m", APIKey: "k", Temperature: "0.5", Thinking: "between_tools", Effort: "medium", MaxRetries: 3})
	m, ok := g.(*Messages)
	if err != nil || !ok || m.Temperature == nil || *m.Temperature != 0.5 || m.Thinking != "between_tools" || m.Effort != "medium" || m.MaxRetries != 3 {
		t.Fatalf("cấu hình hợp lệ: được %+v (lỗi %v)", g, err)
	}
	if g, _ := New(base); g.(*Messages).Temperature != nil {
		t.Error("không cấu hình temperature: muốn nil (không gửi)")
	}
	t.Setenv("EAIS_MODEL_PROVIDER", "anthropic")
	t.Setenv("EAIS_MODEL_THINKING", "between_tools")
	t.Setenv("EAIS_MODEL_EFFORT", "medium")
	t.Setenv("EAIS_MODEL_TEMPERATURE", "")
	if c := ConfigFromEnv(); c.Thinking != "between_tools" || c.Effort != "medium" || c.Temperature != "" || c.MaxRetries != 2 {
		t.Errorf("ConfigFromEnv = %+v", Config{Thinking: c.Thinking, Effort: c.Effort, Temperature: c.Temperature, MaxRetries: c.MaxRetries})
	}
	t.Setenv("EAIS_MODEL_MAX_RETRIES", "x")
	if c := ConfigFromEnv(); c.MaxRetries >= 0 {
		t.Errorf("EAIS_MODEL_MAX_RETRIES sai: muốn giá trị âm để New chặn, được %d", c.MaxRetries)
	}
}

// Thời gian lùi nhân đôi mỗi lần thử lại, có trần, và không tràn số với số lần thử lớn (bản soát mã: S4,
// M24). Trước đây lần thử thứ 35 tràn thành số âm, tức bắn ngay.
func TestBackoffDoublesWithCap(t *testing.T) {
	for _, c := range []struct {
		base    time.Duration
		attempt int
		want    time.Duration
	}{{time.Second, 1, time.Second}, {time.Second, 5, 16 * time.Second}, {time.Second, 6, maxBackoff}, {time.Second, 35, maxBackoff}, {time.Second, 200, maxBackoff}} {
		if got := backoffFor(c.base, c.attempt); got != c.want {
			t.Errorf("backoffFor(%v, %d) = %v, muốn %v", c.base, c.attempt, got, c.want)
		}
	}
	srv, _ := rawBodies(t, func(map[string]any) (int, string) { return http.StatusServiceUnavailable, `{}` })
	var waits []time.Duration
	m := &Messages{Provider: "anthropic", BaseURL: srv.URL, APIKey: "k", Model: "m", MaxTokens: 64, MaxRetries: 3, Backoff: time.Millisecond,
		sleep: func(_ context.Context, d time.Duration) error { waits = append(waits, d); return nil }}
	if _, err := m.Generate(context.Background(), paramsReq); err == nil {
		t.Fatal("503 mãi: muốn lỗi")
	}
	if len(waits) != 3 || waits[0] != time.Millisecond || waits[1] != 2*time.Millisecond || waits[2] != 4*time.Millisecond {
		t.Errorf("thời gian chờ = %v, muốn [1ms 2ms 4ms]", waits)
	}
	if _, err := New(Config{Provider: "anthropic", Model: "m", APIKey: "k", MaxRetries: MaxRetriesLimit + 1}); contracts.FaultOf(err) != contracts.FaultCode {
		t.Errorf("MaxRetries > %d: muốn lỗi cấu hình, được %v", MaxRetriesLimit, err)
	}
}

// Huỷ ctx trong lúc chờ lùi: trả ngay, lỗi gói đúng ctx.Err() để errors.Is nhận ra huỷ / quá hạn (bản soát
// mã: S4, M23b); không gửi thêm lần nào.
func TestBackoffStopsOnContextCancel(t *testing.T) {
	n, waits := 0, 0
	srv, _ := rawBodies(t, func(map[string]any) (int, string) { n++; return http.StatusServiceUnavailable, `{}` })
	ctx, cancel := context.WithCancel(context.Background())
	m := &Messages{Provider: "anthropic", BaseURL: srv.URL, APIKey: "k", Model: "m", MaxTokens: 64, MaxRetries: 5, Backoff: time.Hour,
		sleep: func(c context.Context, _ time.Duration) error { waits++; cancel(); return c.Err() }}
	_, err := m.Generate(ctx, paramsReq)
	// Đếm cả số lần chờ: nếu bỏ qua lỗi huỷ, vòng lặp vẫn thử tiếp (lần gửi sau hỏng ngay ở tầng
	// truyền tải nên máy chủ không thấy) — chỉ số lần chờ lộ ra điều đó.
	if !errors.Is(err, context.Canceled) || n != 1 || waits != 1 {
		t.Errorf("huỷ ctx khi chờ: lỗi %v (errors.Is Canceled = %v), %d lần gửi, %d lần chờ; muốn Canceled sau đúng 1 lần gửi, 1 lần chờ", err, errors.Is(err, context.Canceled), n, waits)
	}
}

// C-18 (soát lượt 2): khoá chỉ được gửi qua https, hoặc http tới loopback (máy chủ giả khi chạy khô).
func TestNewChiNhanHTTPSHoacLoopback(t *testing.T) {
	for url, ok := range map[string]bool{
		"https://api.anthropic.com": true, "http://127.0.0.1:25480": true, "http://localhost:8080": true, "http://[::1]:9": true,
		"http://example.com": false, "http://10.0.0.5:8080": false, "ftp://127.0.0.1": false,
		// thông tin xác thực trong URL sẽ bị ghi nguyên văn vào manifest.json của bằng chứng (soát C2, F-11)
		"https://user:pw@api.anthropic.com": false, "http://nguoidung:matkhau@127.0.0.1:25490": false, "https://user@api.anthropic.com": false,
		// tên máy không phân biệt hoa thường
		"http://LOCALHOST:25490": true,
	} {
		_, err := New(Config{Provider: "anthropic", Model: "m", APIKey: "k", BaseURL: url})
		if (err == nil) != ok {
			t.Errorf("BaseURL %s: lỗi %v; muốn nhận = %v", url, err, ok)
		}
	}
}

// Từ chối URL có thông tin xác thực mà không in lại mật khẩu trong thông báo lỗi (soát C2, F-11).
func TestLoiEndpointKhongInMatKhau(t *testing.T) {
	_, err := New(Config{Provider: "anthropic", Model: "m", APIKey: "k", BaseURL: "http://nguoidung:matkhau-bi-mat@127.0.0.1:25490"})
	if err == nil {
		t.Fatal("URL có thông tin xác thực phải bị từ chối")
	}
	if strings.Contains(err.Error(), "matkhau-bi-mat") || strings.Contains(err.Error(), "nguoidung") {
		t.Errorf("thông báo lỗi lộ thông tin xác thực: %v", err)
	}
}

// C-18: không đi theo chuyển hướng HTTP — khoá không được gửi lại tới đích khác.
func TestKhongTheoChuyenHuong(t *testing.T) {
	var keyAtTarget atomic.Int64
	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("x-api-key") != "" {
			keyAtTarget.Add(1)
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"content":[{"type":"text","text":"{}"}]}`))
	}))
	defer target.Close()
	redirect := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, target.URL+r.URL.Path, http.StatusTemporaryRedirect)
	}))
	defer redirect.Close()
	g, err := New(Config{Provider: "anthropic", Model: "m", APIKey: "khoa-gia", BaseURL: redirect.URL})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := g.Generate(context.Background(), paramsReq); err == nil {
		t.Error("máy chủ trả 307: muốn lỗi, không đi theo chuyển hướng")
	}
	if n := keyAtTarget.Load(); n != 0 {
		t.Errorf("khoá đã được gửi tới đích chuyển hướng %d lần", n)
	}
}

// C-18: EAIS_MODEL_MAX_TOKENS hỏng là lỗi cấu hình, không bị bỏ im lặng về mặc định.
func TestMaxTokensHongLaLoi(t *testing.T) {
	for _, v := range []string{"abc", "-5", "0"} {
		t.Setenv("EAIS_MODEL_PROVIDER", "anthropic")
		t.Setenv("EAIS_MODEL", "m")
		t.Setenv("EAIS_MODEL_API_KEY", "k")
		t.Setenv("EAIS_MODEL_MAX_TOKENS", v)
		if _, err := New(ConfigFromEnv()); err == nil {
			t.Errorf("EAIS_MODEL_MAX_TOKENS=%q: muốn lỗi cấu hình", v)
		}
	}
}

// Thông báo lỗi chỉ in scheme://host của endpoint bị từ chối: khoá đặt trong path / query / fragment của URL (gateway kiểu
// "?api_key=…") không được ra log (soát C3, C3-08; F-11 mới bỏ phần user:pass@).
func TestLoiEndpointChiInSchemeVaHost(t *testing.T) {
	for _, u := range []string{"http://gateway.example.com/v1?api_key=BIMAT123", "http://gateway.example.com/BIMAT123/", "http://gateway.example.com/v1#BIMAT123"} {
		_, err := New(Config{Provider: "anthropic", Model: "m", APIKey: "k", BaseURL: u})
		if err == nil {
			t.Fatalf("%s: http tới máy không phải loopback phải bị từ chối", u)
		}
		if strings.Contains(err.Error(), "BIMAT123") || !strings.Contains(err.Error(), `"http://gateway.example.com"`) {
			t.Errorf("%s: thông báo lỗi %q; muốn chỉ có scheme://host", u, err)
		}
	}
}

// Lỗi truyền tải (không kết nối được, hết giờ) của net/http mang URL ĐẦY ĐỦ của yêu cầu: khoá đặt trong path / query / fragment của
// endpoint không được ra thông báo lỗi hay bản ghi lần gửi (Trace → raw-runs.jsonl, ket-qua.md của g7run) — soát fd4d19c + 246eabb, F6.
func TestLoiTruyenTaiKhongInBiMatTrongURL(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	host := srv.URL
	srv.Close() // máy chủ đã đóng: mọi lần gửi lỗi kết nối
	g, err := New(Config{Provider: "anthropic", Model: "m", APIKey: "khoa-gia", BaseURL: host + "/BIMATPATH/?api_key=BIMATQUERY#BIMATFRAG"})
	if err != nil {
		t.Fatal(err)
	}
	var traced []string
	g.(*Messages).Trace = func(tr MessagesTrace) { traced = append(traced, tr.Err) }
	_, err = g.Generate(context.Background(), paramsReq)
	if err == nil {
		t.Fatal("máy chủ đã đóng: muốn lỗi")
	}
	for _, s := range append(traced, err.Error()) {
		if strings.Contains(s, "BIMAT") {
			t.Errorf("lộ phần bí mật của URL: %q", s)
		}
	}
	if !strings.Contains(err.Error(), host) {
		t.Errorf("lỗi phải còn scheme://host để biết endpoint nào: %q", err)
	}
}
