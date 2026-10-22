package winrun_test

import (
	"bufio"
	"context"
	"os"
	"os/exec"
	"sort"
	"strings"
	"testing"

	"github.com/luannt2002/enterprise-ai-control-layer/platform/pgstore"
	"github.com/luannt2002/enterprise-ai-control-layer/tenants/construction/win/gen"
	"github.com/luannt2002/enterprise-ai-control-layer/tenants/construction/win/winrun"
)

// TestWinRatchet là cổng hồi quy của WIN trong `go test`: sinh bộ dữ liệu (seed cố định), nạp, đo
// G1, chạy cả bộ ca WIN. Đòi: G1 khớp 100%, và MỌI ca trong passing.txt vẫn đạt. Ca mới đạt thì
// in ra để thêm vào passing.txt (ratchet chỉ đi lên); ca đang đạt mà đỏ lại là hồi quy → đỏ.
func TestWinRatchet(t *testing.T) {
	if _, err := exec.LookPath("pdftotext"); err != nil {
		t.Skip("thiếu pdftotext (poppler) — bộ nạp PDF cần nó (ADR-0001 D4)")
	}
	dir := t.TempDir()
	if err := gen.Generate("../world.yaml", dir, "functional", 42); err != nil {
		t.Fatal(err)
	}
	r, err := winrun.Run(winrun.Config{Data: dir, Pack: "../../pack", World: "../world.yaml", CasesGlob: "../cases-*.yaml", Tenant: "abc-infra"})
	if err != nil {
		t.Fatal(err)
	}
	// G1: lệch cờ ước lượng đã biết (g1-approx-known.txt) không làm đỏ; mọi lệch khác và khoá lệch cờ MỚI thì đỏ.
	known, err := loadKnownApprox("../g1-approx-known.txt")
	if err != nil {
		t.Fatal(err)
	}
	unexpected, seen := g1Unexpected(r.G1Problems, known)
	for i, p := range unexpected {
		if i == 10 {
			break
		}
		t.Errorf("G1: %s", p)
	}
	for k := range known {
		if !seen[k] {
			t.Logf("G1: %s đã hết lệch cờ ước lượng — xoá khỏi g1-approx-known.txt", k)
		}
	}
	t.Logf("G1: %d lệch cờ ước lượng đã biết (g1-approx-known.txt)", len(seen))
	// G1 metamorphic (docs/04 mục 6): mỗi bản ghi biến thể có ≥ 4 biểu diễn, và mọi biểu diễn ra cùng
	// một quan sát chuẩn.
	if len(r.Variants) < 2 {
		t.Errorf("G1 metamorphic: chỉ có %d bản ghi biến thể", len(r.Variants))
	}
	for id, n := range r.Variants {
		if n < 4 {
			t.Errorf("G1 metamorphic: %s chỉ có %d biểu diễn, muốn ≥ 4", id, n)
		}
	}
	for _, p := range r.VariantProblems {
		t.Errorf("G1 metamorphic: %s", p)
	}
	// G6: mọi câu đã giao có bản ghi quyết định, và phát lại ra đúng từng chặng.
	if r.Decisions == 0 {
		t.Error("G6: không có bản ghi quyết định nào")
	}
	for _, p := range r.ReplayProblems {
		t.Errorf("G6 phát lại: %s", p)
	}

	want := map[string]bool{}
	f, err := os.Open("../passing.txt")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		if line := strings.TrimSpace(sc.Text()); line != "" && !strings.HasPrefix(line, "#") {
			want[line] = true
		}
	}
	var fresh []string
	for _, c := range r.Report.Results {
		switch {
		case want[c.ID] && !c.Pass:
			t.Errorf("HỒI QUY %s [%s]: %s → %q", c.ID, c.FailureClass, strings.Join(c.Problems, "; "), c.Answer)
		case !want[c.ID] && c.Pass:
			fresh = append(fresh, c.ID)
		}
	}
	sort.Strings(fresh)
	if len(fresh) > 0 {
		t.Logf("ca MỚI đạt — thêm vào passing.txt: %s", strings.Join(fresh, " "))
	}
	t.Logf("G1 %d/%d · bộ ca WIN:\n%s", r.G1Total-len(r.G1Problems), r.G1Total, r.Report.Summary())
}

// TestWinTrenPG chạy lại cả bộ ca WIN trên đường PostgreSQL (danh mục, fact, tài liệu đọc qua RLS
// theo tenant của Principal) và đòi từng ca ra ĐÚNG như đường bộ nhớ: cùng đạt / đỏ, cùng câu trả
// lời. Thiếu biến môi trường PG thì bỏ qua — báo cáo WIN ghi rõ đường PG có chạy hay không.
func TestWinTrenPG(t *testing.T) {
	admin, app := pgstore.EnvDSN()
	if admin == "" || app == "" {
		t.Skip("chưa có EAIS_PG_ADMIN_DSN / EAIS_PG_DSN — xem deploy/docker-compose.yml")
	}
	if _, err := exec.LookPath("pdftotext"); err != nil {
		t.Skip("thiếu pdftotext (poppler)")
	}
	ctx := context.Background()
	if err := pgstore.Migrate(ctx, admin); err != nil {
		t.Fatal(err)
	}
	pg, err := pgstore.Open(ctx, app)
	if err != nil {
		t.Fatal(err)
	}
	defer pg.Close()
	dir := t.TempDir()
	if err := gen.Generate("../world.yaml", dir, "functional", 42); err != nil {
		t.Fatal(err)
	}
	cfg := winrun.Config{Data: dir, Pack: "../../pack", World: "../world.yaml", CasesGlob: "../cases-*.yaml", Tenant: "abc-infra"}
	mem, err := winrun.Run(cfg)
	if err != nil {
		t.Fatal(err)
	}
	cfg.PG = pg
	onPG, err := winrun.Run(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if len(onPG.Report.Results) != len(mem.Report.Results) || len(mem.Report.Results) == 0 {
		t.Fatalf("số ca: PG %d, bộ nhớ %d", len(onPG.Report.Results), len(mem.Report.Results))
	}
	for i, m := range mem.Report.Results {
		p := onPG.Report.Results[i]
		if p.ID != m.ID || p.Pass != m.Pass || p.Answer != m.Answer {
			t.Errorf("%s lệch giữa hai đường:\n  bộ nhớ: %v %q\n  PG:     %v %q (%s)", m.ID, m.Pass, m.Answer, p.Pass, p.Answer, strings.Join(p.Problems, "; "))
		}
	}
	t.Logf("đường PG: %s", onPG.Report.Summary())
}

// loadKnownApprox đọc g1-approx-known.txt: mỗi dòng không rỗng, không bắt đầu bằng #, trường đầu là một khoá manifest.
func loadKnownApprox(path string) (map[string]bool, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	out := map[string]bool{}
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		if line := strings.TrimSpace(sc.Text()); line != "" && !strings.HasPrefix(line, "#") {
			out[strings.Fields(line)[0]] = true
		}
	}
	return out, sc.Err()
}

// g1Unexpected tách các vấn đề G1 thành: không ngờ tới (đỏ) và khoá lệch cờ ước lượng đã biết (seen). Chỉ vấn đề
// CỜ của khoá có trong danh sách mới được bỏ qua: khoá đã biết mà lệch thêm trường khác, hay khoá mới lệch cờ, vẫn đỏ.
func g1Unexpected(problems []string, known map[string]bool) (unexpected []string, seen map[string]bool) {
	seen = map[string]bool{}
	for _, p := range problems {
		key, rest, _ := strings.Cut(p, ": ")
		if known[key] && strings.HasPrefix(rest, winrun.G1ApproxMark) {
			seen[key] = true
			continue
		}
		unexpected = append(unexpected, p)
	}
	return unexpected, seen
}

// TestG1KnownApproxChanDuoc: cổng danh sách lệch đã biết phải THỰC SỰ chặn (CLAUDE.md §1): chỉ lệch cờ của
// khoá có trong danh sách được bỏ qua.
func TestG1KnownApproxChanDuoc(t *testing.T) {
	known := map[string]bool{"k1": true}
	for _, c := range []struct {
		name       string
		problem    string
		wantPassed bool // true: bị bỏ qua (đã biết); false: phải đỏ
	}{
		{"cờ lệch của khoá đã biết", "k1: " + winrun.G1ApproxMark + " được false, muốn true", true},
		{"cờ lệch của khoá MỚI", "k9: " + winrun.G1ApproxMark + " được false, muốn true", false},
		{"khoá đã biết nhưng lệch số", "k1: được EX-03/issued_fuel/erp/2026-10-19/3851, muốn EX-03/issued_fuel/erp/2026-10-19/3850", false},
		{"khoá đã biết nhưng bị cách ly thiếu", "k1: muốn quan sát, không có", false},
	} {
		unexpected, seen := g1Unexpected([]string{c.problem}, known)
		if passed := len(unexpected) == 0 && seen["k1"]; passed != c.wantPassed {
			t.Errorf("%s: bị bỏ qua = %v, muốn %v (unexpected=%v)", c.name, passed, c.wantPassed, unexpected)
		}
	}
}
