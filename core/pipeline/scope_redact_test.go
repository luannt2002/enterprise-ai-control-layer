package pipeline

import (
	"context"
	"errors"
	"slices"
	"strings"
	"testing"

	"github.com/luannt2002/enterprise-ai-control-layer/core/contracts"
	"github.com/luannt2002/enterprise-ai-control-layer/core/tenant"
	"github.com/luannt2002/enterprise-ai-control-layer/data/entityresolution"
	"github.com/luannt2002/enterprise-ai-control-layer/observability/traces"
	"github.com/luannt2002/enterprise-ai-control-layer/simulations/fixtures"
)

// C9 vòng 2: token chỉ bị che khi đứng riêng, token dài che trước token ngắn, và danh sách "?a|b" dồn chỗ bị che về cuối.
func TestScopeRedactorDungRieng(t *testing.T) {
	X := traces.OutOfScope
	for _, c := range []struct{ in, token, want string }{
		{"?DZ-03|EX-03", "DZ-03", "?" + X + "|EX-03"},
		{"EX-030 và EX-03.", "EX-03", "EX-030 và " + X + "."},
		{"PRJ-AB, PRJ-A", "PRJ-A", "PRJ-AB, " + X},
		{"aDZ-03 _DZ-03 -DZ-03 DZ-03_ DZ-03a", "DZ-03", "aDZ-03 _DZ-03 -DZ-03 DZ-03_ DZ-03a"},
		{"ủDZ-03 —DZ-03 (DZ-03)", "DZ-03", "ủDZ-03 —" + X + " (" + X + ")"},
		{"DZ-03DZ-03 DZ-03", "DZ-03", "DZ-03DZ-03 " + X},
		{"không có gì", "DZ-03", "không có gì"},
	} {
		if got := replaceToken(c.in, c.token, X); got != c.want {
			t.Errorf("replaceToken(%q, %q) = %q, muốn %q", c.in, c.token, got, c.want)
		}
	}
	r := scopeRedactor{tokens: []string{"Máy ủi DZ-03", "DZ-03"}}
	if got, want := r.line("(Máy ủi DZ-03, DZ-03)"), "("+X+", "+X+")"; got != want {
		t.Errorf("tên dài phải che trước khoá: %q, muốn %q", got, want)
	}
	d := traces.Decision{Entities: []string{"RL-02", "?DZ-03|EX-03", "?DZ-03"}, Slots: []string{"subject=DZ-03", "period=2026-10-19"}}
	r.decision(&d)
	if want := []string{"RL-02", "?EX-03|" + X, "?" + X}; !slices.Equal(d.Entities, want) {
		t.Errorf("Entities = %q, muốn %q", d.Entities, want)
	}
	if want := []string{"subject=" + X, "period=2026-10-19"}; !slices.Equal(d.Slots, want) {
		t.Errorf("Slots = %q, muốn %q", d.Slots, want)
	}
	var none scopeRedactor // người toàn quyền: không che gì
	d2 := traces.Decision{Entities: []string{"?DZ-03|EX-03"}}
	none.decision(&d2)
	w := traces.Why{Resolve: traces.Notes{Reasons: []string{"khớp DZ-03"}}}
	if d2.Entities[0] != "?DZ-03|EX-03" || none.why(w).Resolve.Reasons[0] != "khớp DZ-03" {
		t.Errorf("bộ che rỗng phải để nguyên: %q, %q", d2.Entities, none.why(w).Resolve.Reasons)
	}
}

// newScopeRedactor trên pack và danh mục thật: kỹ sư PRJ-A gõ "DZ-03" (máy của PRJ-B) thì khoá / mã giữ nguyên nhưng TÊN bị che; không gõ thì
// che cả khoá; thực thể trong phạm vi và người toàn quyền: không che gì.
func TestNewScopeRedactorKhoaVaTen(t *testing.T) {
	rt, err := tenant.Load("../../tenants/construction/pack")
	if err != nil {
		t.Fatalf("nạp pack: %v", err)
	}
	dir, err := fixtures.LoadDirectory("../../tenants/construction/seed/entities.yaml")
	if err != nil {
		t.Fatalf("nạp danh mục: %v", err)
	}
	a := &Ask{RT: rt, Deps: Deps{Directory: dir}}
	dz := contracts.EntityRef{ID: "DZ-03", Key: "DZ-03", Name: "Máy ủi DZ-03", Type: "equipment"}
	ex := contracts.EntityRef{ID: "EX-03", Key: "EX-03", Name: "Máy xúc EX-03", Type: "equipment"}
	eng := contracts.Principal{ID: "u-eng", TenantID: "abc-infra", Roles: []string{"site_engineer"}, Attributes: map[string]string{"entity_scope": "PRJ-A"}}
	ctx := context.Background()
	if got := a.newScopeRedactor(ctx, eng, "DZ-03 hôm nay được cấp bao nhiêu dầu?", []contracts.EntityRef{dz, ex}).tokens; !slices.Equal(got, []string{"Máy ủi DZ-03"}) {
		t.Errorf("gõ khoá DZ-03: tokens = %q, muốn chỉ tên", got)
	}
	if got := a.newScopeRedactor(ctx, eng, "máy 03 hôm nay được cấp bao nhiêu dầu?", []contracts.EntityRef{dz, ex}).tokens; !slices.Equal(got, []string{"Máy ủi DZ-03", "DZ-03"}) {
		t.Errorf("không gõ khoá: tokens = %q, muốn tên rồi khoá", got)
	}
	dirPr := contracts.Principal{ID: "u-dir", TenantID: "abc-infra", Roles: []string{"director"}}
	if got := a.newScopeRedactor(ctx, dirPr, "máy 03 hôm nay được cấp bao nhiêu dầu?", []contracts.EntityRef{dz, ex}).tokens; len(got) != 0 {
		t.Errorf("người toàn quyền: tokens = %q, muốn rỗng", got)
	}
}

// countingDir đếm số lần đọc danh mục; err khác nil thì mọi lần đọc đều lỗi.
type countingDir struct {
	entityresolution.MemoryDirectory
	calls int
	err   error
}

func (d *countingDir) Entities(ctx context.Context, types ...string) ([]entityresolution.Entity, error) {
	d.calls++
	if d.err != nil {
		return nil, d.err
	}
	return d.MemoryDirectory.Entities(ctx, types...)
}

// Soát C8–C11 vòng 2 (C9-ID-KHAC-KHOA, C9-CHI-PHI, C9-TEST-CHUA-GHIM): khoá, mã, tên xét riêng trên danh mục có MÃ KHÁC KHOÁ — gõ khoá không làm
// lộ mã nội bộ, gõ mã không làm lộ khoá, gõ tên thì giữ tên (và khoá nằm trong tên), bí danh thì che cả ba; thực thể trong phạm vi không bị che;
// danh mục đọc đúng MỘT lần mỗi bộ che; đọc danh mục lỗi ⇒ che cả thực thể trong phạm vi (đóng mặc định).
func TestNewScopeRedactorKhoaMaTenRieng(t *testing.T) {
	rt, err := tenant.Load("../../tenants/construction/pack")
	if err != nil {
		t.Fatalf("nạp pack: %v", err)
	}
	dir := &countingDir{MemoryDirectory: entityresolution.MemoryDirectory{
		{ID: "PRJ-A", Type: "project", Key: "PRJ-A", Name: "Cầu A"},
		{ID: "PRJ-B", Type: "project", Key: "PRJ-B", Name: "Đường B"},
		{ID: "e-0001", Type: "equipment", Key: "EX-03", Name: "Máy xúc EX-03", Parent: "PRJ-A"},
		{ID: "e-0002", Type: "equipment", Key: "DZ-03", Name: "Máy ủi DZ-03", Parent: "PRJ-B"},
	}}
	a := &Ask{RT: rt, Deps: Deps{Directory: dir}}
	eng := contracts.Principal{ID: "u-eng", TenantID: "abc-infra", Roles: []string{"site_engineer"}, Attributes: map[string]string{"entity_scope": "PRJ-A"}}
	ex := contracts.EntityRef{ID: "e-0001", Key: "EX-03", Name: "Máy xúc EX-03", Type: "equipment"}
	dz := contracts.EntityRef{ID: "e-0002", Key: "DZ-03", Name: "Máy ủi DZ-03", Type: "equipment"}
	for _, c := range []struct {
		name, q    string
		hide, keep []string
	}{
		{"gõ khoá: mã khác khoá vẫn che", "DZ-03 hôm nay được cấp bao nhiêu dầu?", []string{"e-0002", "Máy ủi DZ-03"}, []string{"DZ-03", "EX-03", "e-0001"}},
		{"gõ mã: khoá vẫn che", "e-0002 hôm nay được cấp bao nhiêu dầu?", []string{"DZ-03", "Máy ủi DZ-03"}, []string{"e-0002"}},
		{"gõ tên: giữ tên và khoá trong tên", "Máy ủi DZ-03 hôm nay được cấp bao nhiêu dầu?", []string{"e-0002"}, []string{"Máy ủi DZ-03", "DZ-03"}},
		{"bí danh: che cả ba", "máy 03 hôm nay được cấp bao nhiêu dầu?", []string{"DZ-03", "e-0002", "Máy ủi DZ-03"}, []string{"EX-03"}},
	} {
		dir.calls = 0
		got := a.newScopeRedactor(context.Background(), eng, c.q, []contracts.EntityRef{ex, dz, ex}).tokens
		for _, h := range c.hide {
			if !slices.Contains(got, h) {
				t.Errorf("%s: phải che %q, tokens = %q", c.name, h, got)
			}
		}
		for _, k := range c.keep {
			if slices.Contains(got, k) {
				t.Errorf("%s: không được che %q, tokens = %q", c.name, k, got)
			}
		}
		if dir.calls != 1 {
			t.Errorf("%s: đọc danh mục %d lần, muốn đúng 1", c.name, dir.calls)
		}
	}
	dir.err = errors.New("danh mục hỏng")
	got := a.newScopeRedactor(context.Background(), eng, "máy 03 hôm nay được cấp bao nhiêu dầu?", []contracts.EntityRef{ex}).tokens
	if !slices.Contains(got, "EX-03") || !slices.Contains(got, "e-0001") {
		t.Errorf("đọc danh mục lỗi ⇒ không kết luận được phạm vi ⇒ che cả thực thể trong phạm vi, tokens = %q", got)
	}
}

// finalWhy che TRƯỚC khi cắt dòng: khoá nằm vắt qua chỗ cắt (maxWhyLineRunes) không được còn mảnh nào ("DZ-0…") — soát C8–C11 vòng 2,
// C9-TEST-CHUA-GHIM (đột biến Z18 "che sau Compact" từng sống sót).
func TestFinalWhyCheTruocKhiCat(t *testing.T) {
	line := strings.Repeat("x", maxWhyLineRunes-4) + " DZ-03 tiếp"
	w := finalWhy(traces.Why{Resolve: traces.Notes{Reasons: []string{line}}}, scopeRedactor{tokens: []string{"DZ-03"}})
	if got := w.Resolve.Reasons[0]; strings.Contains(got, "DZ") || !strings.Contains(got, "cắt") {
		t.Errorf("dòng sau che + cắt còn mảnh khoá hoặc không báo cắt: …%q", got[len(got)-80:])
	}
}
