package models

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/luannt2002/enterprise-ai-control-layer/core/contracts"
)

// F-015: mã chỉ biết GIAO THỨC. Giao thức Messages không có endpoint mặc định (thiếu EAIS_MODEL_BASE_URL là lỗi cấu hình); header
// bổ sung mà dịch vụ đòi (ví dụ phiên bản API) đến từ EAIS_MODEL_HEADERS, không hard code; giao thức chat không nhận biến này.
func TestMessagesTheoGiaoThuc(t *testing.T) {
	if _, err := New(Config{Provider: ProtocolMessages, Model: "m", APIKey: "k"}); contracts.FaultOf(err) != contracts.FaultCode ||
		!strings.Contains(err.Error(), "EAIS_MODEL_BASE_URL") {
		t.Errorf("thiếu BASE_URL: muốn lỗi cấu hình nêu EAIS_MODEL_BASE_URL, được %v", err)
	}
	var got http.Header
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got = r.Header.Clone()
		_, _ = io.WriteString(w, okReply)
	}))
	defer srv.Close()
	g, err := New(Config{Provider: ProtocolMessages, Model: "m", APIKey: "k", BaseURL: srv.URL, Headers: " x-phien-ban = 2000-01-01; x-them=a=b;"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := g.Generate(context.Background(), paramsReq); err != nil {
		t.Fatal(err)
	}
	if got.Get("x-phien-ban") != "2000-01-01" || got.Get("x-them") != "a=b" || got.Get("x-api-key") != "k" || got.Get("content-type") != "application/json" {
		t.Errorf("header gửi đi: %v", got)
	}
	for _, bad := range []string{"khong-co-dau-bang-BIMAT", "ten co cach=BIMAT", "x-a=BIMAT\r\nx-b: c", "=BIMAT", "x-api-key=BIMAT", "Content-Type=BIMAT"} {
		_, err := New(Config{Provider: ProtocolMessages, Model: "m", APIKey: "k", BaseURL: srv.URL, Headers: bad})
		if contracts.FaultOf(err) != contracts.FaultCode || strings.Contains(err.Error(), "BIMAT") {
			t.Errorf("EAIS_MODEL_HEADERS %q: muốn lỗi cấu hình không in giá trị, được %v", bad, err)
		}
	}
	if _, err := New(Config{Provider: ProtocolChat, Model: "m", APIKey: "k", BaseURL: srv.URL, Headers: "x-a=b"}); contracts.FaultOf(err) != contracts.FaultCode {
		t.Errorf("giao thức chat có EAIS_MODEL_HEADERS: muốn lỗi cấu hình, được %v", err)
	}
	t.Setenv("EAIS_MODEL_HEADERS", "x-a=b")
	if c := ConfigFromEnv(); c.Headers != "x-a=b" {
		t.Errorf("ConfigFromEnv không đọc EAIS_MODEL_HEADERS: %q", c.Headers)
	}
}
