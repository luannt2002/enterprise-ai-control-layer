package models

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/luannt2002/enterprise-ai-systems/core/contracts"
)

// Gateway Messages: gửi đúng giao thức (khoá, phiên bản, temperature 0, bằng chứng kèm id), đọc câu
// trả lời + id bằng chứng thành claim, và gắn nhãn lỗi đúng loại. Máy chủ giả — không gọi mạng thật.
func TestMessagesGateway(t *testing.T) {
	var got messagesRequest
	reply, status := "", http.StatusOK
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/messages" || r.Header.Get("x-api-key") != "khoa-test" || r.Header.Get("anthropic-version") == "" {
			t.Errorf("yêu cầu sai: %s %v", r.URL.Path, r.Header)
		}
		b, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(b, &got)
		w.WriteHeader(status)
		_, _ = io.WriteString(w, reply)
	}))
	defer srv.Close()
	m := &Messages{Provider: "anthropic", BaseURL: srv.URL, APIKey: "khoa-test", Model: "model-cau-hinh", MaxTokens: 256}
	ctx := context.Background()
	req := Request{Question: "Biên bản lập trong bao lâu?", Evidence: []contracts.Evidence{{ID: "SOP#0", Text: "Lập biên bản trong vòng 24 giờ."}}}

	reply = `{"content":[{"type":"text","text":"Đây:\n` + "```json" + `\n{\"answer\": \"Lập biên bản trong vòng 24 giờ.\", \"evidence_ids\": [\"SOP#0\"]}\n` + "```" + `"}],"usage":{"input_tokens":120,"output_tokens":30}}`
	d, err := m.Generate(ctx, req)
	if err != nil {
		t.Fatal(err)
	}
	if got.Temperature != 0 || got.Model != "model-cau-hinh" || !strings.Contains(got.Messages[0].Content, "[SOP#0] Lập biên bản") {
		t.Errorf("thân yêu cầu sai: %+v", got)
	}
	if d.Text != "Lập biên bản trong vòng 24 giờ." || len(d.Claims) != 1 || d.Claims[0].EvidenceIDs[0] != "SOP#0" || d.Usage.InputTokens != 120 {
		t.Errorf("bản nháp sai: %+v", d)
	}
	if m.Calls() != 1 {
		t.Errorf("Calls = %d, muốn 1", m.Calls())
	}

	for _, c := range []struct {
		status int
		reply  string
		label  string
	}{
		{http.StatusTooManyRequests, `{}`, contracts.ThirdParty("anthropic", contracts.KindRateLimit)},
		{http.StatusBadGateway, `{}`, contracts.ThirdParty("anthropic", contracts.KindUnavailable)},
		{http.StatusUnauthorized, `{}`, contracts.FaultCode},
		{http.StatusOK, `không phải json`, contracts.ThirdParty("anthropic", contracts.KindBadResponse)},
		{http.StatusOK, `{"content":[{"type":"text","text":"tôi không trả JSON"}]}`, contracts.ThirdParty("anthropic", contracts.KindBadResponse)},
	} {
		status, reply = c.status, c.reply
		if _, err := m.Generate(ctx, req); contracts.FaultOf(err) != c.label {
			t.Errorf("HTTP %d %q: nhãn lỗi = %q (lỗi %v), muốn %q", c.status, c.reply, contracts.FaultOf(err), err, c.label)
		}
	}
	// Không trả lời được: answer rỗng ⇒ bản nháp không claim (cổng Bound sẽ chặn, không giao).
	status, reply = http.StatusOK, `{"content":[{"type":"text","text":"{\"answer\": \"\", \"evidence_ids\": []}"}]}`
	if d, err := m.Generate(ctx, req); err != nil || d.Text != "" || len(d.Claims) != 0 {
		t.Errorf("answer rỗng: được %+v (lỗi %v)", d, err)
	}
}

// New: model giả khi không cấu hình; model thật thiếu model / khoá là lỗi cấu hình (không rơi về
// model giả); provider lạ bị từ chối.
func TestNewTheoCauHinh(t *testing.T) {
	if g, err := New(Config{}); err != nil || g == nil {
		t.Errorf("không cấu hình: muốn model giả, được %v / %v", g, err)
	}
	for _, c := range []Config{{Provider: "anthropic", APIKey: "k"}, {Provider: "anthropic", Model: "m"}, {Provider: "khac"}} {
		if _, err := New(c); contracts.FaultOf(err) != contracts.FaultCode {
			t.Errorf("%+v: muốn lỗi cấu hình, được %v", Config{Provider: c.Provider, Model: c.Model}, err)
		}
	}
	g, err := New(Config{Provider: "anthropic", Model: "m", APIKey: "k"})
	if m, ok := g.(*Messages); err != nil || !ok || m.BaseURL == "" || m.MaxTokens <= 0 {
		t.Errorf("anthropic đủ cấu hình: được %+v (lỗi %v)", g, err)
	}
	t.Setenv("EAIS_MODEL_PROVIDER", "anthropic")
	t.Setenv("EAIS_MODEL", "m-env")
	t.Setenv("EAIS_MODEL_API_KEY", "k-env")
	t.Setenv("EAIS_MODEL_MAX_TOKENS", "300")
	if c := ConfigFromEnv(); c.Provider != "anthropic" || c.Model != "m-env" || c.APIKey != "k-env" || c.MaxTokens != 300 {
		t.Errorf("ConfigFromEnv = %+v", Config{Provider: c.Provider, Model: c.Model, MaxTokens: c.MaxTokens})
	}
}
