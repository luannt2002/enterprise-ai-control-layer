package winrun

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/luannt2002/enterprise-ai-control-layer/core/tenant"
	"github.com/luannt2002/enterprise-ai-control-layer/data/ingestion"
	"github.com/luannt2002/enterprise-ai-control-layer/tenants/construction/win/gen"
)

// Bộ kiểm metamorphic G1 có chặn thật: kỳ vọng của bộ sinh lệch 1 lít thì MỌI biểu diễn của bản ghi
// đó bị báo, bản ghi kia không bị báo oan.
func TestCheckVariantsBatDuocLech(t *testing.T) {
	dir := t.TempDir()
	if err := gen.Generate("../world.yaml", dir, "functional", 42); err != nil {
		t.Fatal(err)
	}
	rt, err := tenant.Load("../../pack")
	if err != nil {
		t.Fatal(err)
	}
	sm, err := ingestion.LoadSources("../../pack/sources.yaml", rt)
	if err != nil {
		t.Fatal(err)
	}
	m := filepath.Join(dir, "variants", "manifest.json")
	b, err := os.ReadFile(m)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(b), `"value_l": "3850"`) {
		t.Fatalf("manifest biến thể không có kỳ vọng 3850 của R-004 như CONTRACT mô tả: %s", b)
	}
	if err := os.WriteFile(m, []byte(strings.Replace(string(b), `"value_l": "3850"`, `"value_l": "3851"`, 1)), 0o644); err != nil {
		t.Fatal(err)
	}
	counts, problems, err := checkVariants(context.Background(), filepath.Join(dir, "variants"), rt, sm)
	if err != nil {
		t.Fatal(err)
	}
	if counts["R-004"] != 4 || len(problems) != 4 {
		t.Fatalf("muốn 4 biểu diễn R-004 đều bị báo, được %v / %v", counts, problems)
	}
	for _, p := range problems {
		if !strings.HasPrefix(p, "R-004") {
			t.Errorf("báo oan: %s", p)
		}
	}
}
