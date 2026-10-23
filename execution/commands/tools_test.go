package commands

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/luannt2002/enterprise-ai-control-layer/data/sourceoftruth"
	"github.com/luannt2002/enterprise-ai-control-layer/intelligence/router"
)

// loadFakePack nạp policy.yaml và metrics.yaml của pack GIẢ trong testdata đúng như tenant.Load làm với hai tệp đó
// (giải mã chặt, Validate, Compile): tools.go kiểm chéo với kết quả này, không đọc hai tệp ấy lần nữa.
func loadFakePack(t *testing.T) (router.Policy, *sourceoftruth.Catalog) {
	t.Helper()
	var pol router.Policy
	if err := decodeStrict(filepath.Join("testdata", "pack", "policy.yaml"), &pol); err != nil {
		t.Fatal(err)
	}
	if err := pol.Validate(); err != nil {
		t.Fatal(err)
	}
	var mf struct {
		Metrics []sourceoftruth.MetricDef  `yaml:"metrics"`
		Derived []sourceoftruth.DerivedDef `yaml:"derived"`
	}
	if err := decodeStrict(filepath.Join("testdata", "pack", "metrics.yaml"), &mf); err != nil {
		t.Fatal(err)
	}
	cat, err := sourceoftruth.Compile(mf.Metrics, mf.Derived, func(string) bool { return true })
	if err != nil {
		t.Fatal(err)
	}
	return pol, cat
}

// packWith dựng thư mục pack chỉ có tools.yaml với nội dung cho trước.
func packWith(t *testing.T, toolsYAML string) string {
	t.Helper()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "tools.yaml"), []byte(toolsYAML), 0o600); err != nil {
		t.Fatal(err)
	}
	return dir
}

func TestLoadToolsPackGiaHopLe(t *testing.T) {
	pol, cat := loadFakePack(t)
	tools, err := LoadTools(filepath.Join("testdata", "pack"), pol, cat)
	if err != nil {
		t.Fatalf("pack giả hợp lệ phải nạp được: %v", err)
	}
	got, ok := tools.Get(CommandOpenDiscrepancyTicket)
	if !ok {
		t.Fatalf("thiếu tool %s; có %v", CommandOpenDiscrepancyTicket, tools.Names())
	}
	want := Tool{Name: CommandOpenDiscrepancyTicket, Capability: "tickets.create", Metrics: []string{"issue_vs_measured"}, MinAbsDiff: "0", Timeout: 5 * time.Second}
	if got.Name != want.Name || got.Capability != want.Capability || !slices.Equal(got.Metrics, want.Metrics) || got.MinAbsDiff != want.MinAbsDiff || got.Timeout != want.Timeout {
		t.Fatalf("tool nạp được %+v, muốn %+v", got, want)
	}
	if !slices.Equal(tools.Names(), []string{CommandOpenDiscrepancyTicket}) {
		t.Fatalf("Names() = %v", tools.Names())
	}
	if _, ok := tools.Get("khong_co_tool_nay"); ok {
		t.Fatal("tên lạ không được có tool")
	}
}

// A12 (phần mã): ba luật nạp của §4.3 và các luật phụ; cấu hình hỏng thì không nạp, lỗi gom đủ thông tin để sửa.
func TestLoadToolsLuatNap(t *testing.T) {
	const head = "tools:\n  - name: open_discrepancy_ticket\n"
	writerOnly := router.Policy{Roles: map[string]router.RoleDef{
		"creator": {Capabilities: []string{"tickets.create", "metrics.read"}, Scope: "all"},
		"writer":  {Capabilities: []string{"tickets.create"}, Scope: "all"}, // có tickets.create mà thiếu metrics.read
	}}
	cases := []struct {
		name    string
		yaml    string
		policy  *router.Policy
		wantErr []string // mọi chuỗi con phải có trong lỗi
	}{
		{"luật 1: tool không khai capability", head + "    metrics: [issue_vs_measured]\n", nil, []string{"open_discrepancy_ticket", "capability"}},
		{"luật 2: vai có tickets.create thiếu metrics.read", head + "    capability: tickets.create\n    metrics: [issue_vs_measured]\n", &writerOnly, []string{"writer", "metrics.read"}},
		{"luật 3: chỉ số gốc không phải dẫn xuất", head + "    capability: tickets.create\n    metrics: [issued_amount]\n", nil, []string{"issued_amount", "dẫn xuất"}},
		{"luật 3: chỉ số không khai ở metrics.yaml", head + "    capability: tickets.create\n    metrics: [khong_co]\n", nil, []string{"khong_co"}},
		{"khoá lạ bị từ chối (gõ nhầm không thành giá trị rỗng)", head + "    capabilty: tickets.create\n    metrics: [issue_vs_measured]\n", nil, []string{"capabilty"}},
		{"tên tool trùng", head + "    capability: tickets.create\n    metrics: [issue_vs_measured]\n  - name: open_discrepancy_ticket\n    capability: tickets.create\n    metrics: [issue_vs_measured]\n", nil, []string{"trùng"}},
		{"tên tool rỗng", "tools:\n  - capability: tickets.create\n    metrics: [issue_vs_measured]\n", nil, []string{"name"}},
		{"không có chỉ số nào", head + "    capability: tickets.create\n    metrics: []\n", nil, []string{"metrics"}},
		{"min_abs_diff không phải số", head + "    capability: tickets.create\n    metrics: [issue_vs_measured]\n    min_abs_diff: \"abc\"\n", nil, []string{"min_abs_diff"}},
		{"min_abs_diff âm", head + "    capability: tickets.create\n    metrics: [issue_vs_measured]\n    min_abs_diff: \"-1\"\n", nil, []string{"min_abs_diff"}},
		{"timeout âm", head + "    capability: tickets.create\n    metrics: [issue_vs_measured]\n    timeout: -5s\n", nil, []string{"timeout"}},
		{"nhiều lỗi cùng lúc được gom đủ", head + "    metrics: [khong_co, issued_amount]\n", nil, []string{"capability", "khong_co", "issued_amount"}},
		{"YAML hỏng", "tools: [\n", nil, []string{"tools.yaml"}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			pol, cat := loadFakePack(t)
			if c.policy != nil {
				pol = *c.policy
			}
			tools, err := LoadTools(packWith(t, c.yaml), pol, cat)
			if err == nil {
				t.Fatalf("pack hỏng phải không nạp được, được %v", tools.Names())
			}
			if tools != nil {
				t.Fatalf("nạp lỗi thì không trả tập tool nào")
			}
			for _, w := range c.wantErr {
				if !strings.Contains(err.Error(), w) {
					t.Errorf("lỗi %q thiếu %q", err.Error(), w)
				}
			}
		})
	}
}

// Pack không có tools.yaml thì không có lệnh nào (đóng mặc định); không phải lỗi vì tenant có thể chỉ hỏi.
func TestLoadToolsThieuTep(t *testing.T) {
	pol, cat := loadFakePack(t)
	tools, err := LoadTools(t.TempDir(), pol, cat)
	if err != nil || tools == nil || len(tools.Names()) != 0 {
		t.Fatalf("thiếu tools.yaml phải ra tập rỗng không lỗi, được %v, %v", tools, err)
	}
	if _, err := LoadTools(t.TempDir(), pol, nil); err == nil {
		t.Fatal("danh mục chỉ số nil là lỗi gọi sai, không được lặng lẽ bỏ qua luật 3")
	}
}

// Giá trị mặc định: min_abs_diff rỗng là "0" (mọi chênh lệch khác 0); timeout vắng là không có hạn riêng của tool.
func TestLoadToolsMacDinh(t *testing.T) {
	pol, cat := loadFakePack(t)
	tools, err := LoadTools(packWith(t, "tools:\n  - name: open_discrepancy_ticket\n    capability: tickets.create\n    metrics: [issue_vs_measured]\n"), pol, cat)
	if err != nil {
		t.Fatal(err)
	}
	got, _ := tools.Get(CommandOpenDiscrepancyTicket)
	if got.MinAbsDiff != "0" || got.Timeout != 0 {
		t.Fatalf("mặc định sai: %+v", got)
	}
}
