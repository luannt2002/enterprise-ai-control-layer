package router

import (
	"strings"
	"testing"

	"github.com/luannt2002/enterprise-ai-control-layer/core/contracts"
	"github.com/luannt2002/enterprise-ai-control-layer/intelligence/intent"
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

// C10 (lệnh ca chiều 23/10): bậc "phạm vi dữ liệu" duyệt các khe thực thể theo THỨ TỰ TÊN KHE đã sắp xếp (R2-03 đổi từ thứ tự lặp của map,
// vốn ngẫu nhiên mỗi lần chạy), nên khi nhiều khe cùng ngoài phạm vi, lý do từ chối và dòng vết luôn nêu thực thể của khe đứng đầu theo
// tên — cùng câu, cùng người hỏi thì cùng kế hoạch. Ghim bằng nhiều lần chạy (thứ tự map của Go đổi giữa các lần lặp).
func TestRouteDuyetKheTheoThuTuCoDinh(t *testing.T) {
	yes := func(string) bool { return true }
	set, err := intent.Compile([]intent.Def{{
		Name: "q2", Route: "SQL_QUERY", Keywords: []string{"cấp"}, Capability: "metrics.read",
		Slots: []intent.SlotDef{
			{Name: "z_may", From: "entity", Types: []string{"equipment"}, Required: true},
			{Name: "a_may", From: "entity", Types: []string{"equipment"}, Required: true},
			{Name: "m_may", From: "entity", Types: []string{"equipment"}, Required: true},
		},
	}}, yes, yes)
	if err != nil {
		t.Fatal(err)
	}
	pol := Policy{Roles: map[string]RoleDef{"eng": {Capabilities: []string{"metrics.read"}, Scope: "attribute"}}}
	pr := contracts.Principal{Roles: []string{"eng"}, Attributes: map[string]string{"entity_scope": "PRJ-X"}}
	lineage := func(e contracts.EntityRef) []string { return []string{e.Key, "PRJ-KHAC"} }
	slot := func(name, key string) contracts.Slot {
		return contracts.Slot{Name: name, Entity: &contracts.EntityRef{ID: key, Key: key, Type: "equipment"}}
	}
	cls := intent.Result{Intent: contracts.Intent{Name: "q2", Slots: map[string]contracts.Slot{
		"z_may": slot("z_may", "EX-ZZ"), "a_may": slot("a_may", "EX-AA"), "m_may": slot("m_may", "EX-MM"),
	}}}
	for i := range 200 {
		plan := Route(set, cls, contracts.NormalizedInput{}, pr, pol, Settings{}, lineage)
		if plan.Route != contracts.RouteReject || plan.Reason != "thực thể EX-AA ngoài phạm vi dữ liệu của người hỏi" {
			t.Fatalf("lần %d: route %s, lý do %q — muốn REJECT nêu EX-AA (khe a_may, đứng đầu theo tên)", i, plan.Route, plan.Reason)
		}
	}
}
