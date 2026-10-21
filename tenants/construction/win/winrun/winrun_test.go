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
	for i, p := range r.G1Problems {
		if i == 10 {
			break
		}
		t.Errorf("G1: %s", p)
	}
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
	admin, app := os.Getenv("EACL_PG_ADMIN_DSN"), os.Getenv("EACL_PG_DSN")
	if admin == "" || app == "" {
		t.Skip("chưa có EACL_PG_ADMIN_DSN / EACL_PG_DSN — xem deploy/docker-compose.yml")
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
