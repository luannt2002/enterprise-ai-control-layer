package winrun_test

import (
	"bufio"
	"context"
	"fmt"
	"maps"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
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
	// G1: lệch cờ ước lượng ĐÃ BIẾT (g1-approx-known.txt, đúng chiều đã ghim) không làm đỏ. Mọi lệch khác, khoá lệch cờ MỚI,
	// lệch ngược chiều đã ghim, và khoá đã hết lệch mà chưa xoá khỏi tệp (ratchet chỉ đi xuống) đều ĐỎ.
	known, err := loadKnownApprox("../g1-approx-known.txt")
	if err != nil {
		t.Fatal(err)
	}
	g1Errs, tolerated := g1RatchetErrors(r.G1Problems, known)
	for i, e := range g1Errs {
		if i == 10 {
			t.Errorf("G1: … và %d lỗi nữa", len(g1Errs)-10)
			break
		}
		t.Error(e)
	}
	t.Logf("G1: %d lệch cờ ước lượng đã biết (g1-approx-known.txt)", len(tolerated))
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

// knownApprox là một lệch cờ ước lượng đã biết: bộ nạp ra got, manifest muốn want ("true" / "false").
type knownApprox struct{ got, want string }

// loadKnownApprox đọc g1-approx-known.txt: mỗi dòng không rỗng, không bắt đầu bằng #, có dạng
// `khoá được=<true|false> muốn=<true|false> nguyên nhân` với được ≠ muốn (chiều lệch được ghim). Dòng sai dạng hoặc khoá
// lặp là lỗi: tệp ghim hỏng thì test phải đỏ, không bỏ qua lặng lẽ. scripts/scale-smoke.sh đọc cùng định dạng này.
func loadKnownApprox(path string) (map[string]knownApprox, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	out := map[string]knownApprox{}
	sc := bufio.NewScanner(f)
	for n := 1; sc.Scan(); n++ {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		fields := strings.Fields(line)
		var got, want string
		var okGot, okWant bool
		if len(fields) >= 4 {
			got, okGot = strings.CutPrefix(fields[1], "được=")
			want, okWant = strings.CutPrefix(fields[2], "muốn=")
		}
		isBool := func(s string) bool { return s == "true" || s == "false" }
		if !okGot || !okWant || !isBool(got) || !isBool(want) || got == want {
			return nil, fmt.Errorf("%s:%d: dòng phải có dạng `khoá được=<true|false> muốn=<true|false> nguyên nhân` với được ≠ muốn, có %q", path, n, line)
		}
		if _, dup := out[fields[0]]; dup {
			return nil, fmt.Errorf("%s:%d: khoá %s lặp", path, n, fields[0])
		}
		out[fields[0]] = knownApprox{got: got, want: want}
	}
	return out, sc.Err()
}

// g1RatchetErrors phân xử các vấn đề G1 theo danh sách lệch cờ đã biết (ratchet chỉ đi xuống): chỉ vấn đề CỜ của khoá có
// trong danh sách, ĐÚNG chiều đã ghim, mới được bỏ qua (trả về trong tolerated). Mọi thứ khác là lỗi: lệch cờ của khoá mới,
// lệch cờ ngược chiều đã ghim, khoá đã biết mà lệch thêm trường khác (vấn đề không phải cờ), và khoá đã biết không còn
// vấn đề nào — người sửa bộ chuẩn hoá hay bộ sinh phải xoá khoá khỏi tệp cùng commit.
func g1RatchetErrors(problems []string, known map[string]knownApprox) (errs, tolerated []string) {
	touched := map[string]bool{}
	for _, p := range problems {
		key, _, _ := strings.Cut(p, ": ")
		touched[key] = true
		if k, ok := known[key]; ok && p == key+": "+winrun.G1ApproxMark+" được "+k.got+", muốn "+k.want {
			tolerated = append(tolerated, key)
			continue
		}
		errs = append(errs, "G1: "+p)
	}
	for _, key := range slices.Sorted(maps.Keys(known)) {
		if !touched[key] {
			errs = append(errs, "G1: "+key+" đã hết lệch cờ ước lượng — xoá khỏi g1-approx-known.txt cùng commit sửa (ratchet chỉ đi xuống)")
		}
	}
	return errs, tolerated
}

// TestG1RatchetChanDuoc: cổng danh sách lệch đã biết phải THỰC SỰ chặn (CLAUDE.md §1): chỉ lệch cờ đúng chiều của khoá có
// trong danh sách được bỏ qua; mọi trường hợp còn lại, kể cả khoá đã hết lệch, là lỗi.
func TestG1RatchetChanDuoc(t *testing.T) {
	known := map[string]knownApprox{"k1": {got: "false", want: "true"}}
	approx := func(key, got, want string) string {
		return key + ": " + winrun.G1ApproxMark + " được " + got + ", muốn " + want
	}
	for _, c := range []struct {
		name          string
		problems      []string
		wantTolerated int
		wantErrs      []string // mỗi phần tử là chuỗi phải có trong lỗi thứ i; độ dài = số lỗi
	}{
		{"đúng chiều đã ghim", []string{approx("k1", "false", "true")}, 1, nil},
		{"ngược chiều đã ghim", []string{approx("k1", "true", "false")}, 0, []string{"k1: cờ ước lượng được true, muốn false"}},
		{"khoá MỚI lệch cờ", []string{approx("k1", "false", "true"), approx("k9", "false", "true")}, 1, []string{"k9: cờ ước lượng được false, muốn true"}},
		{"khoá đã biết nhưng lệch số", []string{"k1: được EX-03/issued_fuel/erp/2026-10-19/3851, muốn EX-03/issued_fuel/erp/2026-10-19/3850"}, 0, []string{"k1: được EX-03"}},
		{"khoá đã biết nhưng bị cách ly thiếu", []string{"k1: muốn quan sát, không có"}, 0, []string{"k1: muốn quan sát"}},
		{"khoá đã hết lệch mà còn trong tệp", nil, 0, []string{"k1 đã hết lệch cờ ước lượng — xoá khỏi g1-approx-known.txt"}},
	} {
		errs, tolerated := g1RatchetErrors(c.problems, known)
		if len(tolerated) != c.wantTolerated || len(errs) != len(c.wantErrs) {
			t.Errorf("%s: bỏ qua %v, lỗi %v; muốn %d bỏ qua, %d lỗi", c.name, tolerated, errs, c.wantTolerated, len(c.wantErrs))
			continue
		}
		for i, want := range c.wantErrs {
			if !strings.Contains(errs[i], want) {
				t.Errorf("%s: lỗi %d = %q, muốn chứa %q", c.name, i, errs[i], want)
			}
		}
	}
	if errs, _ := g1RatchetErrors(nil, nil); len(errs) != 0 {
		t.Errorf("không ghim khoá nào, không lệch nào mà có lỗi: %v", errs)
	}
}

// TestLoadKnownApproxSaiDang: tệp ghim sai dạng hay có khoá lặp phải là LỖI khi đọc (không bỏ qua lặng lẽ), và tệp thật
// đọc ra đúng chiều lệch đã ghim.
func TestLoadKnownApproxSaiDang(t *testing.T) {
	const good = "k1 được=false muốn=true nguyên nhân\n"
	for _, c := range []struct {
		name, content string
		wantErr       bool
	}{
		{"đúng dạng, có chú thích và dòng rỗng", "# chú thích\n\n" + good, false},
		{"thiếu nguyên nhân", "k1 được=false muốn=true\n", true},
		{"thiếu chiều muốn", "k1 được=false nguyên nhân dài\n", true},
		{"giá trị không phải true/false", "k1 được=sai muốn=true nguyên nhân\n", true},
		{"được bằng muốn (không phải lệch)", "k1 được=true muốn=true nguyên nhân\n", true},
		{"khoá lặp", good + good, true},
		{"dòng cũ chưa ghim chiều", "k1 \"nguyên nhân\" như cũ\n", true},
	} {
		p := filepath.Join(t.TempDir(), "known.txt")
		if err := os.WriteFile(p, []byte(c.content), 0o600); err != nil {
			t.Fatal(err)
		}
		if _, err := loadKnownApprox(p); (err != nil) != c.wantErr {
			t.Errorf("%s: lỗi = %v, muốn lỗi = %v", c.name, err, c.wantErr)
		}
	}
	known, err := loadKnownApprox("../g1-approx-known.txt")
	if err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{"messages.jsonl#R-001", "messages.jsonl#R-002"} {
		if known[key] != (knownApprox{got: "false", want: "true"}) {
			t.Errorf("tệp thật: %s ghim %+v, muốn được=false muốn=true", key, known[key])
		}
	}
	// "≈ 4 m³" của báo cáo PDF đã hết lệch khi bộ chuẩn hoá nhận dấu ước lượng dạng ký hiệu mà pack khai (ratchet: xoá cùng commit sửa)
	if _, still := known["reports/2026-10-19.pdf#1"]; still {
		t.Errorf("tệp thật: reports/2026-10-19.pdf#1 đã hết lệch nhưng vẫn còn trong tệp")
	}
}
