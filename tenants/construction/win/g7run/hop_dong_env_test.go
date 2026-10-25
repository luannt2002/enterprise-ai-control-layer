package main

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
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

// S2 (soát độc lập cuối): mốc của hợp đồng G7 — endpoint chính thức và model §4 — là băm GHIM trong mã, không đổi được bằng .env. Trước:
// đặt EAIS_G7_ENDPOINT / EAIS_G7_MODEL trỏ về máy giả https và model giả thì thử khói ra mã 0, manifest dry_run=false — "bằng chứng G7"
// mà không ai soát diff. Đối chứng dương: giá trị đúng của hợp đồng (đọc từ tài liệu) vẫn là chính thức. Không gọi mạng (fakeOfficial).
func TestMocHopDongKhongDoiBangCauHinh(t *testing.T) {
	if textSHA256(modelHopDong) != contractModelSHA256 || textSHA256(endpointChinhThuc) != officialEndpointSHA256 {
		t.Fatal("model / endpoint đọc từ docs/g7/HOP-DONG-G7.md không khớp băm ghim trong mã: tài liệu đổi mốc mà mã chưa đổi, hoặc không đọc được tài liệu")
	}
	const epGia, modelGia = "https://api.example.com", "model-gia"
	clearModelEnv(t)
	t.Setenv("EAIS_G7_ENDPOINT", epGia)
	if isOfficialEndpoint(endpointChinhThuc) {
		t.Error("EAIS_G7_ENDPOINT lệch mốc: không endpoint nào được là chính thức, kể cả endpoint thật")
	}
	t.Setenv("EAIS_G7_ENDPOINT", endpointChinhThuc)
	if isOfficialEndpoint("http://" + strings.TrimPrefix(endpointChinhThuc, "https://")) {
		t.Error("cùng tên máy chính thức nhưng http: không phải endpoint chính thức")
	}
	data := genData(t)
	for _, c := range []struct {
		name, ep, model string
		code            int
		dry             string // dry_run trong manifest; "" = bị từ chối trước lời gọi model, không có manifest
	}{
		{"endpoint và model giả", epGia, modelGia, 2, ""},
		{"endpoint giả", epGia, modelHopDong, 2, "true"},
		{"model giả", endpointChinhThuc, modelGia, 2, ""},
		{"đúng hợp đồng", endpointChinhThuc, modelHopDong, 0, "false"},
	} {
		t.Run(c.name, func(t *testing.T) {
			calls := fakeOfficial(t)
			for k, v := range map[string]string{"EAIS_G7_ENDPOINT": c.ep, "EAIS_MODEL_BASE_URL": c.ep, "EAIS_G7_MODEL": c.model, "EAIS_MODEL": c.model,
				"EAIS_G7_PRICES": c.model + "=2/10"} {
				t.Setenv(k, v)
			}
			cfg := baseCfg(data)
			cfg.smoke, cfg.out = 1, filepath.Join(t.TempDir(), "out")
			code, dry := run(cfg), ""
			if b, err := os.ReadFile(filepath.Join(cfg.out, "manifest.json")); err == nil {
				var m manifestInfo
				if err := json.Unmarshal(b, &m); err != nil {
					t.Fatal(err)
				}
				dry = strconv.FormatBool(m.DryRun)
			}
			if code != c.code || dry != c.dry || (c.dry == "") != (calls.Load() == 0) {
				t.Errorf("mã %d, dry_run %q, %d lời gọi; muốn mã %d, dry_run %q (rỗng = từ chối trước lời gọi model)", code, dry, calls.Load(), c.code, c.dry)
			}
		})
	}
}

// Hợp đồng G7 §4 chỉ định MỘT giao thức (messages), và chỉ nhánh đó có trần ngân sách (budgetGateway) và kiểm trước khi tiêu tiền. Giao
// thức khác (chat_completions) từng gọi model không trần, không kiểm — kết luận vẫn CHẶN nhưng tiền thật đã tiêu (lát tài liệu nêu khi
// soát cuối). Nay bị từ chối TRƯỚC lời gọi model đầu tiên, cả thử khói lẫn lượt chạy: đóng mặc định.
func TestGiaoThucNgoaiHopDongBiTuChoiTruocLoiGoiModel(t *testing.T) {
	var calls atomic.Int64
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		_, _ = io.WriteString(w, `{"choices":[{"message":{"content":"{\"answer\": \"x\", \"evidence_ids\": []}"}}],"usage":{"prompt_tokens":10,"completion_tokens":5}}`)
	}))
	defer srv.Close()
	clearModelEnv(t)
	for k, v := range map[string]string{"EAIS_MODEL_PROVIDER": models.ProtocolChat, "EAIS_MODEL": modelHopDong, "EAIS_MODEL_API_KEY": "k", "EAIS_MODEL_BASE_URL": srv.URL} {
		t.Setenv(k, v)
	}
	data := genData(t)
	for _, smoke := range []int{1, 0} {
		calls.Store(0)
		cfg := baseCfg(data)
		cfg.smoke = smoke
		if code := run(cfg); code != 2 || calls.Load() != 0 {
			t.Errorf("giao thức %s, -smoke %d: mã %d, %d lời gọi; muốn 2 và 0 lời gọi (từ chối trước khi tiêu tiền)", models.ProtocolChat, smoke, code, calls.Load())
		}
	}
}
