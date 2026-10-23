package main

// Pack THẬT của tenant tham chiếu có tools.yaml và policy.yaml cấp tickets.create đúng ba vai của Q-P1-1; cấu hình lệnh hỏng thì ứng
// dụng KHÔNG khởi động (HOP-DONG §4.3, A12). K1 đã có test cho từng luật nạp trên pack GIẢ (execution/commands/tools_test.go); ở đây
// kiểm hai thứ gói đó không với tới: (1) tệp pack thật nạp được và nói đúng điều hợp đồng nói; (2) newApp, nơi cmd/eais nạp tools.yaml
// ngay sau tenant.Load, từ chối mọi cấu hình hỏng thay vì chạy với lệnh sai quyền.

import (
	"log/slog"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/luannt2002/enterprise-ai-control-layer/core/tenant"
	"github.com/luannt2002/enterprise-ai-control-layer/execution/commands"
)

func TestPackThatCoToolLenh(t *testing.T) {
	pack := repoPath("tenants/construction/pack")
	rt, err := tenant.Load(pack)
	if err != nil {
		t.Fatal(err)
	}
	tools, err := commands.LoadTools(pack, rt.Policy, rt.Metrics)
	if err != nil {
		t.Fatalf("tools.yaml của pack thật phải nạp được: %v", err)
	}
	if got := tools.Names(); !slices.Equal(got, []string{commands.CommandOpenDiscrepancyTicket}) {
		t.Fatalf("tool của pack thật = %v, muốn đúng [%s] (Q-P1-3: Phase 1 chỉ một lệnh)", got, commands.CommandOpenDiscrepancyTicket)
	}
	tool, _ := tools.Get(commands.CommandOpenDiscrepancyTicket)
	if tool.Capability != "tickets.create" || !slices.Equal(tool.Metrics, []string{"issue_vs_measured"}) || tool.MinAbsDiff != "0" || tool.Timeout != 0 {
		t.Fatalf("tool %+v: muốn capability tickets.create, metrics [issue_vs_measured] (Q-P1-3), min_abs_diff 0 (Q-P1-2), chưa có timeout", tool)
	}
	// Q-P1-1: đúng ba vai có tickets.create; mỗi vai đó đọc được số liệu (phiếu không là đường vòng quyền đọc, luật nạp 2).
	for role, def := range rt.Policy.Roles {
		wantCreate := role == "director" || role == "accountant" || role == "project_manager"
		if got := slices.Contains(def.Capabilities, tool.Capability); got != wantCreate {
			t.Errorf("vai %s: có %s = %v, muốn %v (Q-P1-1)", role, tool.Capability, got, wantCreate)
		}
		if wantCreate && !slices.Contains(def.Capabilities, commands.CapMetricsRead) {
			t.Errorf("vai %s có %s mà thiếu %s", role, tool.Capability, commands.CapMetricsRead)
		}
	}
}

// Q-P1-2: ngưỡng là cấu hình của pack; 0 chỉ là giá trị demo và tệp phải NÓI đúng điều đó bằng đúng chữ, để không ai đọc 0 là quyết định nghiệp vụ.
func TestToolsYamlGhiNguongChiLaDemo(t *testing.T) {
	b, err := os.ReadFile(repoPath("tenants/construction/pack/tools.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(b), "chưa có giá trị nghiệp vụ (D-RM-03)") {
		t.Fatal(`tools.yaml phải ghi đúng chữ "chưa có giá trị nghiệp vụ (D-RM-03)" cạnh min_abs_diff (Q-P1-2)`)
	}
}

// mutatePack sửa đúng một chỗ của một tệp pack (from phải có mặt: chỗ sửa trượt thì ca đỏ chứ không âm thầm thành ca xanh).
func mutatePack(t *testing.T, packDir, file, from, to string) {
	t.Helper()
	p := filepath.Join(packDir, file)
	b, err := os.ReadFile(p)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(b), from) {
		t.Fatalf("%s không có %q để sửa", file, from)
	}
	if err := os.WriteFile(p, []byte(strings.Replace(string(b), from, to, 1)), 0o644); err != nil {
		t.Fatal(err)
	}
}

// Cấu hình lệnh hỏng ⇒ newApp lỗi ⇒ serve và ingest không chạy. Mỗi dòng là một lỗi nạp; ba dòng đầu là ba luật của §4.3, còn lại là luật phụ.
func TestNewAppCauHinhLenhHongKhongKhoiDong(t *testing.T) {
	cases := []struct {
		name    string
		file    string
		from    string
		to      string
		wantErr []string
	}{
		{"luật 1: tool không khai capability", "tools.yaml", "capability: tickets.create", `capability: ""`, []string{"open_discrepancy_ticket", "capability"}},
		{"luật 2: vai có tickets.create mà thiếu metrics.read", "policy.yaml", "driver: {capabilities: []", "driver: {capabilities: [tickets.create]", []string{"driver", "metrics.read"}},
		{"luật 3: chỉ số của tool không phải chỉ số dẫn xuất", "tools.yaml", "metrics: [issue_vs_measured]", "metrics: [issued_fuel]", []string{"issued_fuel", "dẫn xuất"}},
		{"luật 3: chỉ số của tool không khai ở metrics.yaml", "tools.yaml", "metrics: [issue_vs_measured]", "metrics: [khong_co]", []string{"khong_co"}},
		{"khoá lạ (gõ nhầm không được thành giá trị rỗng)", "tools.yaml", "capability:", "capabilty:", []string{"capabilty"}},
		{"tên tool chưa cài trong gói", "tools.yaml", "name: open_discrepancy_ticket", "name: close_ticket", []string{"close_ticket", "chưa cài"}},
		{"min_abs_diff âm vô hiệu hoá R4", "tools.yaml", `min_abs_diff: "0"`, `min_abs_diff: "-1"`, []string{"min_abs_diff"}},
		{"timeout âm", "tools.yaml", `min_abs_diff: "0"`, "min_abs_diff: \"0\"\n    timeout: -5s", []string{"timeout"}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			tenantID := uniqTenant("lenh-hong")
			packDir, dataRoot := tenantFixtures(t, tenantID)
			mutatePack(t, packDir, c.file, c.from, c.to)
			a, err := newApp(config{Pack: packDir, Data: dataRoot, Now: clockForTest}, newMemStore(), slog.New(slog.DiscardHandler))
			if err == nil {
				t.Fatal("cấu hình lệnh hỏng mà newApp vẫn dựng được ứng dụng: serve sẽ chạy với lệnh sai quyền")
			}
			if a != nil {
				t.Errorf("lỗi nạp thì không trả app nào, được %v", a)
			}
			for _, w := range c.wantErr {
				if !strings.Contains(err.Error(), w) {
					t.Errorf("lỗi %q thiếu %q", err.Error(), w)
				}
			}
		})
	}
	t.Run("YAML hỏng", func(t *testing.T) {
		packDir, dataRoot := tenantFixtures(t, uniqTenant("lenh-hong"))
		if err := os.WriteFile(filepath.Join(packDir, "tools.yaml"), []byte("tools: [\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		if _, err := newApp(config{Pack: packDir, Data: dataRoot, Now: clockForTest}, newMemStore(), slog.New(slog.DiscardHandler)); err == nil || !strings.Contains(err.Error(), "tools.yaml") {
			t.Fatalf("YAML hỏng phải làm newApp lỗi và nói tên tệp, được %v", err)
		}
	})
	t.Run("đối chứng: pack nguyên vẹn dựng được", func(t *testing.T) {
		packDir, dataRoot := tenantFixtures(t, uniqTenant("lenh-hong"))
		if _, err := newApp(config{Pack: packDir, Data: dataRoot, Now: clockForTest}, newMemStore(), slog.New(slog.DiscardHandler)); err != nil {
			t.Fatalf("pack thật phải dựng được: %v", err)
		}
	})
}

// K3S-02 (soát K3): pack không có tool lệnh nào (xoá tools.yaml, đổi tên, hay "tools: []") vẫn khởi động được, nhưng route lệnh im lặng thành 404 —
// nên newApp phải để lại MỘT dòng cảnh báo nêu thư mục pack; pack có tool thì có dòng liệt kê route lệnh đã dựng.
func TestNewAppKhongCoToolLenhThiCanhBao(t *testing.T) {
	for _, c := range []struct {
		name string
		edit func(t *testing.T, packDir string)
	}{
		{"xoá tools.yaml", func(t *testing.T, d string) {
			if err := os.Remove(filepath.Join(d, "tools.yaml")); err != nil {
				t.Fatal(err)
			}
		}},
		{"đổi tên thành tools.yml", func(t *testing.T, d string) {
			if err := os.Rename(filepath.Join(d, "tools.yaml"), filepath.Join(d, "tools.yml")); err != nil {
				t.Fatal(err)
			}
		}},
		{"tools rỗng", func(t *testing.T, d string) {
			if err := os.WriteFile(filepath.Join(d, "tools.yaml"), []byte("tools: []\n"), 0o644); err != nil {
				t.Fatal(err)
			}
		}},
	} {
		t.Run(c.name, func(t *testing.T) {
			packDir, dataRoot := tenantFixtures(t, uniqTenant("lenh-trong"))
			c.edit(t, packDir)
			var buf syncBuffer
			if _, err := newApp(config{Pack: packDir, Data: dataRoot, Now: clockForTest}, newMemStore(), slog.New(slog.NewTextHandler(&buf, nil))); err != nil {
				t.Fatalf("pack không có tool lệnh vẫn phải dựng được: %v", err)
			}
			if log := buf.String(); !strings.Contains(log, "level=WARN") || !strings.Contains(log, "không có tool lệnh") {
				t.Errorf("thiếu cảnh báo khi pack không có tool lệnh: %q", log)
			}
		})
	}
	packDir, dataRoot := tenantFixtures(t, uniqTenant("lenh-co"))
	var buf syncBuffer
	if _, err := newApp(config{Pack: packDir, Data: dataRoot, Now: clockForTest}, newMemStore(), slog.New(slog.NewTextHandler(&buf, nil))); err != nil {
		t.Fatal(err)
	}
	if log := buf.String(); strings.Contains(log, "không có tool lệnh") || !strings.Contains(log, "open_discrepancy_ticket") {
		t.Errorf("pack có tool: không cảnh báo và phải liệt kê tool đã nạp, được %q", log)
	}
}

// Ở mức tiến trình: serve với tools.yaml hỏng thoát mã 1 và nói lý do, không nghe cổng nào (cần PG vì serve mở kho trước khi dựng app).
func TestServeDevCauHinhLenhHongKhongChay(t *testing.T) {
	openPG(t)
	packDir, dataRoot := tenantFixtures(t, uniqTenant("lenh-hong"))
	mutatePack(t, packDir, "tools.yaml", "capability: tickets.create", `capability: ""`)
	// runCLIFor chứ không run: nếu một thay đổi làm serve CHẠY ĐƯỢC với cấu hình hỏng thì test ĐỎ sau 30 giây (ngữ cảnh hết hạn
	// tắt máy chủ, rc khác 1, stdout có "đang nghe") thay vì treo tới hạn của go test (soát K3, K3S-04).
	rc, stdout, stderr := runCLIFor(30*time.Second, "serve", "-dev", "-addr", "127.0.0.1:0", "-pack", packDir, "-world", repoPath("tenants/construction/win/world.yaml"), "-data", dataRoot)
	if rc != 1 || !strings.Contains(stderr, "tools.yaml") || strings.Contains(stdout, "đang nghe") {
		t.Fatalf("muốn mã 1, lý do nêu tools.yaml và KHÔNG nghe cổng; được rc=%d stdout=%q stderr=%q", rc, stdout, stderr)
	}
}
