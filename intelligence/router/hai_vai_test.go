package router

import (
	"testing"

	"github.com/luannt2002/enterprise-ai-control-layer/core/contracts"
	"github.com/luannt2002/enterprise-ai-control-layer/intelligence/intent"
)

// Q11b (Q-P1-10 / Q11 = A): mọi bậc phạm vi của router — khe thực thể, gỡ mơ hồ theo phạm vi, câu hỏi nghĩa số liệu — chỉ xét vai có
// capability của ý định (và của vế so sánh). Vai "xem" phạm vi all mà không có capability nào không cho vai "eng" (metrics.read, PRJ-A) mượn
// phạm vi: người [eng + xem] nhận đúng kế hoạch của eng một mình. Câu so sánh: vai "x" có a.read phạm vi all, vai "y" có a.read + b.read phạm
// vi PRJ-A — không vai nào một mình đọc được cả hai vế của DZ-03 (PRJ-B), nên người [x + y] bị từ chối như y một mình.
func TestRouteHaiVaiPhamViMotVai(t *testing.T) {
	yes := func(string) bool { return true }
	subject := []intent.SlotDef{{Name: "subject", From: "entity", Types: []string{"equipment"}, Required: true}}
	set, err := intent.Compile([]intent.Def{
		{Name: "q", Route: "SQL_QUERY", Keywords: []string{"cấp"}, Capability: "metrics.read", Slots: subject},
		{Name: "a", Route: "SQL_QUERY", Keywords: []string{"xuất"}, Capability: "a.read", Slots: subject},
		{Name: "b", Route: "SQL_QUERY", Keywords: []string{"báo"}, Capability: "b.read", Slots: subject},
	}, yes, yes)
	if err != nil {
		t.Fatal(err)
	}
	pol := Policy{Roles: map[string]RoleDef{
		"eng":  {Capabilities: []string{"metrics.read"}, Scope: "attribute"},
		"xem":  {Scope: "all"},
		"x":    {Capabilities: []string{"a.read"}, Scope: "all"},
		"y":    {Capabilities: []string{"a.read", "b.read"}, Scope: "attribute"},
		"boss": {Capabilities: []string{"metrics.read"}, Scope: "all"},
	}}
	parent := map[string]string{"EX-03": "PRJ-A", "DZ-03": "PRJ-B"}
	lineage := func(e contracts.EntityRef) []string { return []string{e.Key, parent[e.Key]} }
	ex := contracts.EntityRef{ID: "EX-03", Key: "EX-03", Type: "equipment"}
	dz := contracts.EntityRef{ID: "DZ-03", Key: "DZ-03", Type: "equipment"}
	filled := func(name string, e contracts.EntityRef) contracts.Intent {
		return contracts.Intent{Name: name, Slots: map[string]contracts.Slot{"subject": {Name: "subject", Entity: &e}}}
	}
	who := func(roles ...string) contracts.Principal {
		return contracts.Principal{Roles: roles, Attributes: map[string]string{"entity_scope": "PRJ-A"}}
	}
	type ke struct {
		route            contracts.RouteKind
		outcome          contracts.Outcome
		template, reason string
		intent           string
		subject, missing any
	}
	plan := func(pr contracts.Principal, cls intent.Result) ke {
		p := Route(set, cls, contracts.NormalizedInput{}, pr, pol, Settings{}, lineage)
		var subj any
		if s, ok := p.Intent.Slots["subject"]; ok && s.Entity != nil {
			subj = s.Entity.Key
		}
		return ke{p.Route, p.Outcome, p.Template, p.Reason, p.Intent.Name, subj, len(p.Intent.Missing)}
	}
	for _, c := range []struct {
		name        string
		cls         intent.Result
		alone, both contracts.Principal
		want        contracts.RouteKind
	}{
		{"khe thực thể ngoài phạm vi", intent.Result{Intent: filled("q", dz)}, who("eng"), who("eng", "xem"), contracts.RouteReject},
		{"bí danh mơ hồ gỡ theo phạm vi", intent.Result{Intent: contracts.Intent{Name: "q", Missing: []string{"subject"}}, Candidates: []contracts.EntityRef{ex, dz}},
			who("eng"), who("eng", "xem"), contracts.RouteSQLQuery},
		{"câu hỏi nghĩa số liệu", intent.Result{NeedMeaning: []string{"q"}, Mentioned: []contracts.EntityRef{dz}}, who("eng"), who("eng", "xem"), contracts.RouteReject},
		{"câu so sánh hai capability", intent.Result{Intent: filled("a", dz), Compare: "b"}, who("y"), who("x", "y"), contracts.RouteReject},
	} {
		alone, both := plan(c.alone, c.cls), plan(c.both, c.cls)
		if alone.route != c.want {
			t.Errorf("%s: tiền đề — vai %v một mình muốn %s, được %+v", c.name, c.alone.Roles, c.want, alone)
		}
		if both != alone {
			t.Errorf("%s: người %v được %+v — muốn đúng kế hoạch của %v một mình %+v (phạm vi do CÙNG vai có capability thoả, Q11)",
				c.name, c.both.Roles, both, c.alone.Roles, alone)
		}
	}
	// đối chứng dương: vai đủ quyền phạm vi all cộng vai không quyền vẫn đọc được DZ-03 và vẫn hỏi lại khi bí danh mơ hồ
	if p := plan(who("boss", "xem"), intent.Result{Intent: filled("q", dz)}); p.route != contracts.RouteSQLQuery {
		t.Errorf("đối chứng: [boss + xem] hỏi DZ-03 muốn SQL_QUERY, được %+v", p)
	}
	if p := plan(who("boss", "xem"), intent.Result{Intent: contracts.Intent{Name: "q", Missing: []string{"subject"}}, Candidates: []contracts.EntityRef{ex, dz}}); p.route != contracts.RouteClarification {
		t.Errorf("đối chứng: [boss + xem] với bí danh mơ hồ muốn hỏi lại, được %+v", p)
	}
}
