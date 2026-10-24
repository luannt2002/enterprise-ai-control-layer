package main

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/luannt2002/enterprise-ai-control-layer/intelligence/models"
)

// F-015: giá trị cụ thể của hợp đồng G7 (endpoint chính thức, model hợp đồng, bảng giá) chỉ ở .env, không trong mã. Thiếu biến thì đóng
// mặc định: không endpoint nào là chính thức (mọi lượt là chạy khô), mọi model lệch hợp đồng, không model nào có giá (lượt tốn tiền bị
// từ chối). Bảng giá viết sai là lỗi cấu hình, không lặng lẽ thành "chưa có giá".
func TestHopDongG7TuMoiTruong(t *testing.T) {
	clearModelEnv(t)
	for _, k := range []string{"EAIS_G7_MODEL", "EAIS_G7_ENDPOINT", "EAIS_G7_PRICES"} {
		t.Setenv(k, "")
	}
	for _, u := range []string{endpointChinhThuc, "", "https:"} {
		if isOfficialEndpoint(u) {
			t.Errorf("chưa đặt EAIS_G7_ENDPOINT: %q không được là endpoint chính thức", u)
		}
	}
	if p, err := priceTable(); err != nil || len(p) != 0 {
		t.Errorf("chưa đặt EAIS_G7_PRICES: muốn bảng rỗng, được %v / %v", p, err)
	}
	gw := &models.Messages{Model: modelHopDong, Thinking: contractThinking, Effort: contractEffort, MaxTokens: contractMaxTokens, MaxRetries: contractMaxRetries}
	cfg := baseCfg(t.TempDir())
	if d := contractDeviations(cfg, gw); len(d) == 0 || !strings.Contains(strings.Join(d, "; "), "EAIS_G7_MODEL") {
		t.Errorf("chưa đặt EAIS_G7_MODEL: muốn lệch hợp đồng nêu tên biến, được %q", d)
	}

	t.Setenv("EAIS_G7_ENDPOINT", endpointChinhThuc)
	t.Setenv("EAIS_G7_MODEL", modelHopDong)
	t.Setenv("EAIS_G7_PRICES", " m-a = 2/10 ;m-b=0.1/0.5;")
	if !isOfficialEndpoint(strings.ToUpper(endpointChinhThuc)+".:443/") || isOfficialEndpoint("https://gateway.example.com") {
		t.Error("đặt EAIS_G7_ENDPOINT: so endpoint theo nghĩa như trước")
	}
	if d := contractDeviations(cfg, gw); len(d) != 0 {
		t.Errorf("đúng hợp đồng mà vẫn lệch: %q", d)
	}
	if p, err := priceTable(); err != nil || p["m-a"] != [2]float64{2, 10} || p["m-b"] != [2]float64{0.1, 0.5} || len(p) != 2 {
		t.Errorf("bảng giá = %v / %v", p, err)
	}
	for _, bad := range []string{"m-a", "m-a=2", "m-a=x/1", "=2/10", "m-a=-1/2", "m-a=NaN/1", "m-a=Inf/1", "m-a=1/2;m-a=3/4"} {
		t.Setenv("EAIS_G7_PRICES", bad)
		if _, err := priceTable(); err == nil {
			t.Errorf("EAIS_G7_PRICES %q: muốn lỗi cấu hình", bad)
		}
	}
	// run(): bảng giá sai dừng TRƯỚC lời gọi model (đối chứng: bảng đúng thì thử khói gọi model và trượt vì thân hỏng, mã 1).
	var calls atomic.Int64
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		_, _ = io.WriteString(w, `không phải JSON`)
	}))
	defer srv.Close()
	for k, v := range map[string]string{"EAIS_MODEL_PROVIDER": models.ProtocolMessages, "EAIS_MODEL": "m", "EAIS_MODEL_API_KEY": "k", "EAIS_MODEL_BASE_URL": srv.URL} {
		t.Setenv(k, v)
	}
	run1 := genData(t)
	for prices, want := range map[string]int{"m=2/10": 1, "m=2": 2} {
		t.Setenv("EAIS_G7_PRICES", prices)
		calls.Store(0)
		c := baseCfg(run1)
		c.smoke = 1
		if code := run(c); code != want || (want == 2) != (calls.Load() == 0) {
			t.Errorf("EAIS_G7_PRICES %q: mã %d, %d lời gọi; muốn mã %d (bảng sai: 0 lời gọi)", prices, code, calls.Load(), want)
		}
	}
}
