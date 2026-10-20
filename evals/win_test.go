package evals

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"go.yaml.in/yaml/v3"
)

// winPrincipals đọc tập id principal của tenant A từ world.yaml.
func winPrincipals(t *testing.T) map[string]bool {
	t.Helper()
	b, err := os.ReadFile("../tenants/construction/win/world.yaml")
	if err != nil {
		t.Fatal(err)
	}
	var w struct {
		Tenants []struct {
			ID         string `yaml:"id"`
			Principals []struct {
				ID string `yaml:"id"`
			} `yaml:"principals"`
		} `yaml:"tenants"`
	}
	if err := yaml.Unmarshal(b, &w); err != nil {
		t.Fatal(err)
	}
	out := map[string]bool{}
	for _, tn := range w.Tenants {
		if tn.ID == "abc-infra" {
			for _, p := range tn.Principals {
				out[p.ID] = true
			}
		}
	}
	if len(out) == 0 {
		t.Fatal("world.yaml không có principal nào của abc-infra")
	}
	return out
}

// Mọi tệp ca WIN đã có phải đúng hợp đồng từng ca. (Phần đếm cả bộ — ≥ 100 ca… — là cổng G6,
// chạy trong báo cáo WIN, không nằm ở đây để bộ test vẫn là mốc xanh trong lúc đang soạn ca.)
func TestWinCaseFilesAreValid(t *testing.T) {
	files, _ := filepath.Glob("../tenants/construction/win/cases-*.yaml")
	principals := winPrincipals(t)
	for _, f := range files {
		s, err := LoadWinSuite(f)
		if err != nil {
			t.Errorf("%v", err)
			continue
		}
		for _, p := range ValidateWinSuite(s, "abc-infra", principals) {
			t.Errorf("%s: %s", f, p)
		}
		t.Logf("%s: %d ca", f, len(s.Cases))
	}
}

// Cổng kiểm từng ca và cổng đếm G6 phải chặn thật.
func TestWinGatesBlockBadSuites(t *testing.T) {
	principals := map[string]bool{"u-director": true}
	good := WinCase{ID: "WIN-A-001", FailureClass: "semantic", AuthoredBy: "author", Principal: "u-director",
		Question: "EX-03 hôm qua được cấp bao nhiêu dầu?", Expect: WinExpect{Route: "SQL_QUERY", Outcome: "RESOLVED"}}
	s := WinSuite{Suite: "win", Tenant: "abc-infra", Cases: []WinCase{good}}
	if p := ValidateWinSuite(s, "abc-infra", principals); len(p) != 0 {
		t.Fatalf("ca đúng bị chặn oan: %v", p)
	}
	mismatch := good
	mismatch.ID = "WIN-I-002" // tiền tố I nhưng authored_by = author
	badRoute := good
	badRoute.ID, badRoute.Expect.Route = "WIN-A-003", "MAGIC"
	stranger := good
	stranger.ID, stranger.Principal = "WIN-A-004", "u-ghost"
	s.Cases = []WinCase{good, good, mismatch, badRoute, stranger}
	got := strings.Join(ValidateWinSuite(s, "abc-infra", principals), "\n")
	for _, want := range []string{"id trùng", "không khớp tiền tố", "expect.route lạ", "không có trong world.yaml"} {
		if !strings.Contains(got, want) {
			t.Errorf("không bắt được %q:\n%s", want, got)
		}
	}

	// Cổng đếm: 100 ca cùng một lớp, toàn của người viết mã → thiếu lớp + thiếu ca độc lập.
	var many []WinCase
	for i := 0; i < 100; i++ {
		c := good
		c.ID = "WIN-A-" + string(rune('0'+i/100)) + string(rune('0'+i/10%10)) + string(rune('0'+i%10))
		many = append(many, c)
	}
	counts := strings.Join(WinGateCounts(many), "\n")
	for _, want := range []string{"lớp entity có 0 ca", "ca độc lập 0/100"} {
		if !strings.Contains(counts, want) {
			t.Errorf("cổng đếm không bắt được %q:\n%s", want, counts)
		}
	}
	if p := WinGateCounts(many[:3]); !strings.Contains(strings.Join(p, "\n"), "chỉ có 3 ca") {
		t.Errorf("cổng đếm không bắt thiếu ca: %v", p)
	}
}
