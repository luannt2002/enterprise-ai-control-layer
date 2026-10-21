package models

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/luannt2002/enterprise-ai-control-layer/core/contracts"
)

// Gateway Messages: gửi đúng giao thức (khoá, phiên bản, không gửi temperature — ADR-0005 §1, bằng chứng kèm id), đọc câu
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
	if got.Temperature != nil || got.Model != "model-cau-hinh" || !strings.Contains(got.Messages[0].Content, "[SOP#0] Lập biên bản") {
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
	} {
		status, reply = c.status, c.reply
		if _, err := m.Generate(ctx, req); contracts.FaultOf(err) != c.label {
			t.Errorf("HTTP %d %q: nhãn lỗi = %q (lỗi %v), muốn %q", c.status, c.reply, contracts.FaultOf(err), err, c.label)
		}
	}
	// C-05 (soát lượt 2): API trả 200 hợp lệ nhưng MODEL viết văn xuôi thay vì JSON ⇒ đó là đầu ra của model, không
	// phải lỗi nhà cung cấp (hợp đồng G7 §10: PROVIDER = 429 / 5xx / quá giờ / không tới được). Bản nháp mang nguyên
	// văn xuôi, KHÔNG có claim (cổng Bound của EACL sẽ chặn); baseline thì được chấm theo nội dung câu đó. Kỳ vọng cũ
	// (KindBadResponse) khẳng định hành vi sai: nó làm lời từ chối bằng văn xuôi của baseline thành PROVIDER + trượt.
	status, reply = http.StatusOK, `{"content":[{"type":"text","text":"Tôi không thể trả lời câu hỏi này vì bằng chứng không đủ."}]}`
	if d, err := m.Generate(ctx, req); err != nil || d.Text != "Tôi không thể trả lời câu hỏi này vì bằng chứng không đủ." || len(d.Claims) != 0 {
		t.Errorf("model trả văn xuôi: được %+v (lỗi %v); muốn bản nháp mang nguyên văn, không claim, không lỗi", d, err)
	}
	// Không trả lời được: answer rỗng ⇒ bản nháp không claim (cổng Bound sẽ chặn, không giao).
	status, reply = http.StatusOK, `{"content":[{"type":"text","text":"{\"answer\": \"\", \"evidence_ids\": []}"}]}`
	if d, err := m.Generate(ctx, req); err != nil || d.Text != "" || len(d.Claims) != 0 {
		t.Errorf("answer rỗng: được %+v (lỗi %v)", d, err)
	}
	// Phản hồi 200 mà model KHÔNG viết chữ nào, hoặc bị cắt / bị từ chối ở tầng nhà cung cấp, không phải câu trả lời
	// của model. Trước đây chúng thành bản nháp rỗng, rồi "rỗng = từ chối" cho baseline ĐẠT oan các ca từ chối và
	// không vào ngưỡng lượt vô hiệu (soát C2, F-04). Nay là lỗi (KindBadResponse), như thân không phải JSON.
	for _, bad := range []string{
		`{"content":[{"type":"thinking","thinking":"..."}],"stop_reason":"max_tokens","usage":{"output_tokens":1024}}`,
		`{"content":[],"stop_reason":"end_turn"}`,
		`{}`,
		`{"content":[{"type":"text","text":"Tôi không thể"}],"stop_reason":"refusal"}`,
		`{"content":[{"type":"text","text":"{\"answer\": \"3.85"}],"stop_reason":"max_tokens"}`,
	} {
		status, reply = http.StatusOK, bad
		if d, err := m.Generate(ctx, req); contracts.FaultOf(err) != contracts.ThirdParty("anthropic", contracts.KindBadResponse) {
			t.Errorf("HTTP 200 %s: được %+v, nhãn lỗi %q (lỗi %v); muốn lỗi KindBadResponse", bad, d, contracts.FaultOf(err), err)
		}
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
	t.Setenv("EACL_MODEL_PROVIDER", "anthropic")
	t.Setenv("EACL_MODEL", "m-env")
	t.Setenv("EACL_MODEL_API_KEY", "k-env")
	t.Setenv("EACL_MODEL_MAX_TOKENS", "300")
	if c := ConfigFromEnv(); c.Provider != "anthropic" || c.Model != "m-env" || c.APIKey != "k-env" || c.MaxTokens != 300 {
		t.Errorf("ConfigFromEnv = %+v", Config{Provider: c.Provider, Model: c.Model, MaxTokens: c.MaxTokens})
	}
}
