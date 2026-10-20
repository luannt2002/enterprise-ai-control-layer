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
