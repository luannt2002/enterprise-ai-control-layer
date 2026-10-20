package router

import (
	"strings"
	"testing"

	"github.com/luannt2002/enterprise-ai-systems/core/contracts"
	"github.com/luannt2002/enterprise-ai-systems/intelligence/intent"
)

// Đóng mặc định: capability rỗng không cấp quyền cho ai, kể cả vai có quyền khác.
func TestAllowsIsFailClosed(t *testing.T) {
	pol := Policy{Roles: map[string]RoleDef{
		"director": {Capabilities: []string{"metrics.read"}, Scope: "all"},
		"driver":   {Scope: "attribute"},
	}}
	director := contracts.Principal{Roles: []string{"director"}}
	driver := contracts.Principal{Roles: []string{"driver"}}
	for _, c := range []struct {
		name string
		pr   contracts.Principal
		cap  string
		want bool
	}{
		{"có đúng capability", director, "metrics.read", true},
		{"capability khác", director, "documents.read", false},
		{"capability rỗng", director, "", false},
		{"vai không có capability", driver, "metrics.read", false},
		{"vai lạ", contracts.Principal{Roles: []string{"ghost"}}, "metrics.read", false},
	} {
		if got := pol.Allows(c.pr, c.cap); got != c.want {
			t.Errorf("%s: Allows = %v, muốn %v", c.name, got, c.want)
		}
	}
}

// Khe để trống vì câu nêu nhiều giá trị khác nhau (hoặc ngày hỏng) vẫn là CLARIFICATION_REQUIRED, nhưng lý do route phải
// nói đúng điều đó — không phải "thiếu khe", vì người hỏi đã nêu, chỉ là nêu nhiều hơn một.
func TestRouteHoiLaiNoiDungLyDo(t *testing.T) {
	yes := func(string) bool { return true }
	set, err := intent.Compile([]intent.Def{{
		Name: "q", Route: "SQL_QUERY", Keywords: []string{"cấp"}, Capability: "metrics.read",
		Slots: []intent.SlotDef{{Name: "subject", From: "entity", Types: []string{"equipment"}, Required: true}},
	}}, yes, yes)
	if err != nil {
		t.Fatal(err)
	}
	pol := Policy{Roles: map[string]RoleDef{"director": {Capabilities: []string{"metrics.read"}, Scope: "all"}}}
	pr := contracts.Principal{Roles: []string{"director"}}
	lineage := func(contracts.EntityRef) []string { return nil }
	ex, rl := contracts.EntityRef{ID: "EX-03", Type: "equipment"}, contracts.EntityRef{ID: "RL-02", Type: "equipment"}
	for _, c := range []struct {
		name    string
		unclear intent.Unclear
		want    string
	}{
		{"khe thiếu thật", intent.Unclear{}, "thiếu khe bắt buộc: subject"},
		{"nhiều thực thể", intent.Unclear{Entities: []contracts.EntityRef{ex, rl}}, "nhiều giá trị khác nhau hoặc ngày hỏng cho khe: subject"},
		{"ngày hỏng", intent.Unclear{Invalid: []string{"31/02"}}, "nhiều giá trị khác nhau hoặc ngày hỏng cho khe: subject"},
	} {
		cls := intent.Result{Intent: contracts.Intent{Name: "q", Missing: []string{"subject"}}, Unclear: c.unclear}
		plan := Route(set, cls, contracts.NormalizedInput{}, pr, pol, Settings{}, lineage)
		if plan.Route != contracts.RouteClarification || plan.Outcome != contracts.OutcomeClarificationRequired || plan.Template != "clarify_missing" {
			t.Errorf("%s: route/outcome/template = %s/%s/%s", c.name, plan.Route, plan.Outcome, plan.Template)
		}
		if !strings.Contains(plan.Reason, c.want) {
			t.Errorf("%s: lý do = %q, muốn chứa %q", c.name, plan.Reason, c.want)
		}
	}
}
