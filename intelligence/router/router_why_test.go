package router

import (
	"slices"
	"strings"
	"testing"

	"github.com/luannt2002/enterprise-ai-control-layer/core/contracts"
	"github.com/luannt2002/enterprise-ai-control-layer/intelligence/intent"
)

// R2-03. Router ghi từng bậc kiểm ĐÚNG LÚC bậc đó được xét (RoutePlan.Checked) và việc gỡ bí danh mơ hồ (RoutePlan.Narrowed): vết nói được
// router đã đi qua bậc nào, bậc nào chặn, ứng viên trước / sau khi lọc — không suy lại từ route cuối.

func whySet(t *testing.T) *intent.Set {
	t.Helper()
	yes := func(string) bool { return true }
	set, err := intent.Compile([]intent.Def{{
		Name: "q", Route: "SQL_QUERY", Keywords: []string{"cấp"}, Capability: "metrics.read", Template: "metric_total",
		Slots: []intent.SlotDef{{Name: "subject", From: "entity", Types: []string{"equipment", "project"}, Required: true}},
	}}, yes, yes)
	if err != nil {
		t.Fatal(err)
	}
	return set
}

func rules(p contracts.RoutePlan) []string {
	var out []string
	for _, c := range p.Checked {
		out = append(out, c.Rule)
	}
	return out
}

func fired(p contracts.RoutePlan) []string {
	var out []string
	for _, c := range p.Checked {
		if c.Fired {
			out = append(out, c.Rule+"→"+string(c.Route))
		}
	}
	return out
}

func TestRouteGhiCacBacKiemTheoThuTu(t *testing.T) {
	set := whySet(t)
	pol := Policy{Roles: map[string]RoleDef{
		"director": {Capabilities: []string{"metrics.read"}, Scope: "all"},
		"driver":   {Scope: "attribute"},
	}}
	lineage := func(e contracts.EntityRef) []string { return []string{e.Key} }
	ex := contracts.EntityRef{ID: "EX-03", Key: "EX-03", Type: "equipment"}
	filled := contracts.Intent{Name: "q", Slots: map[string]contracts.Slot{"subject": {Name: "subject", Entity: &ex}}}
	director, driver := contracts.Principal{Roles: []string{"director"}}, contracts.Principal{Roles: []string{"driver"}}

	// đi hết thang: mọi bậc chặn đều "qua", bậc cuối chọn route khai của ý định
	ok := Route(set, intent.Result{Intent: filled}, contracts.NormalizedInput{}, director, pol, Settings{}, lineage)
	want := []string{"nhiều ý định cùng khớp", "thiếu thuật ngữ bắt buộc", "hỏi lại nghĩa số liệu", "không ý định nào khớp", "capability của ý định",
		"phạm vi dữ liệu", "thiếu khe bắt buộc", "route khai của ý định", "câu so sánh hai nghĩa", "ý định tri thức khớp kèm"}
	if ok.Route != contracts.RouteSQLQuery || !slices.Equal(rules(ok), want) {
		t.Errorf("route %s, các bậc %v, muốn SQL_QUERY và %v", ok.Route, rules(ok), want)
	}
	if got := fired(ok); !slices.Equal(got, []string{"route khai của ý định→SQL_QUERY"}) {
		t.Errorf("chỉ bậc cuối được chọn, được %v", got)
	}

	// thiếu capability: bậc capability chặn, các bậc SAU nó không được xét nên không có mặt
	deny := Route(set, intent.Result{Intent: filled}, contracts.NormalizedInput{}, driver, pol, Settings{}, lineage)
	if deny.Route != contracts.RouteReject || !slices.Equal(fired(deny), []string{"capability của ý định→REJECT"}) || deny.Checked[len(deny.Checked)-1].Rule != "capability của ý định" {
		t.Errorf("thiếu capability: route %s, chặn %v, các bậc %v", deny.Route, fired(deny), rules(deny))
	}
	if d := deny.Checked[len(deny.Checked)-1].Detail; !strings.Contains(d, "metrics.read: thiếu") || !strings.Contains(d, "vai driver") {
		t.Errorf("dữ kiện của bậc capability: %q", d)
	}

	// nhiều ý định hoà nhau: bậc đầu tiên chặn, không còn bậc nào sau nó
	amb := Route(set, intent.Result{Ambiguous: []string{"a", "b"}}, contracts.NormalizedInput{}, director, pol, Settings{}, lineage)
	if !slices.Equal(rules(amb), []string{"nhiều ý định cùng khớp"}) || !slices.Equal(fired(amb), []string{"nhiều ý định cùng khớp→CLARIFICATION"}) || amb.Checked[0].Detail != "a, b" {
		t.Errorf("hoà ý định: %+v", amb.Checked)
	}

	// thực thể ngoài phạm vi: bậc phạm vi chặn và nói đúng thực thể
	scoped := Policy{Roles: map[string]RoleDef{"eng": {Capabilities: []string{"metrics.read"}, Scope: "attribute"}}}
	eng := contracts.Principal{Roles: []string{"eng"}, Attributes: map[string]string{"entity_scope": "PRJ-B"}}
	out := Route(set, intent.Result{Intent: filled}, contracts.NormalizedInput{}, eng, scoped, Settings{}, lineage)
	last := out.Checked[len(out.Checked)-1]
	if out.Route != contracts.RouteReject || last.Rule != "phạm vi dữ liệu" || !last.Fired || !strings.Contains(last.Detail, "EX-03: ngoài phạm vi") {
		t.Errorf("ngoài phạm vi: route %s, bậc cuối %+v", out.Route, last)
	}
}

func TestNarrowGhiUngVienTruocVaSauKhiLoc(t *testing.T) {
	set := whySet(t)
	dz, ex := contracts.EntityRef{ID: "DZ-03", Key: "DZ-03", Type: "equipment"}, contracts.EntityRef{ID: "EX-03", Key: "EX-03", Type: "equipment"}
	lin := map[string][]string{"DZ-03": {"DZ-03", "PRJ-B"}, "EX-03": {"EX-03", "PRJ-A"}, "PRJ-A": {"PRJ-A"}}
	lineage := func(e contracts.EntityRef) []string { return lin[e.Key] }
	pol := Policy{Roles: map[string]RoleDef{
		"director": {Capabilities: []string{"metrics.read"}, Scope: "all"},
		"eng":      {Capabilities: []string{"metrics.read"}, Scope: "attribute"},
	}}
	cands := []contracts.EntityRef{dz, ex}
	missing := contracts.Intent{Name: "q", Missing: []string{"subject"}}

	// phạm vi dữ liệu gỡ được: chỉ EX-03 nằm trong PRJ-A
	eng := contracts.Principal{Roles: []string{"eng"}, Attributes: map[string]string{"entity_scope": "PRJ-A"}}
	plan := Route(set, intent.Result{Intent: missing, Candidates: cands}, contracts.NormalizedInput{}, eng, pol, Settings{}, lineage)
	n := plan.Narrowed
	if n.Slot != "subject" || n.By != contracts.NarrowByScope || !slices.Equal(n.Before, []string{"DZ-03", "EX-03"}) || !slices.Equal(n.After, []string{"EX-03"}) || n.Chosen != "EX-03" {
		t.Errorf("gỡ bằng phạm vi: %+v", n)
	}
	if plan.Route != contracts.RouteSQLQuery {
		t.Errorf("gỡ được thì đi tiếp, route %s", plan.Route)
	}

	// vai xem được tất cả: không gỡ được, giữ cả hai ứng viên và không chọn
	plan = Route(set, intent.Result{Intent: missing, Candidates: cands}, contracts.NormalizedInput{}, contracts.Principal{Roles: []string{"director"}}, pol, Settings{}, lineage)
	n = plan.Narrowed
	if n.By != contracts.NarrowByScope || !slices.Equal(n.After, []string{"DZ-03", "EX-03"}) || n.Chosen != "" || plan.Route != contracts.RouteClarification {
		t.Errorf("không gỡ được: %+v route %s", n, plan.Route)
	}

	// khe đã điền bằng một dự án: gỡ theo dự án đó
	prj := contracts.EntityRef{ID: "PRJ-A", Key: "PRJ-A", Type: "project"}
	byProject := contracts.Intent{Name: "q", Slots: map[string]contracts.Slot{"subject": {Name: "subject", Entity: &prj}}}
	plan = Route(set, intent.Result{Intent: byProject, Candidates: cands}, contracts.NormalizedInput{}, contracts.Principal{Roles: []string{"director"}}, pol, Settings{}, lineage)
	n = plan.Narrowed
	if n.By != contracts.NarrowByProject || !slices.Equal(n.After, []string{"EX-03"}) || n.Chosen != "EX-03" {
		t.Errorf("gỡ theo dự án: %+v", n)
	}

	// ứng viên trùng (EX-03 thuộc hai cụm mơ hồ) được router đếm hai lần nên không gỡ được — vết phải cho thấy đúng con số router đã đếm
	dup := Route(set, intent.Result{Intent: missing, Candidates: []contracts.EntityRef{dz, ex, ex}}, contracts.NormalizedInput{}, eng, pol, Settings{}, lineage)
	if n := dup.Narrowed; !slices.Equal(n.Before, []string{"DZ-03", "EX-03", "EX-03"}) || !slices.Equal(n.After, []string{"EX-03", "EX-03"}) || n.Chosen != "" || dup.Route != contracts.RouteClarification {
		t.Errorf("ứng viên trùng: %+v route %s", n, dup.Route)
	}

	// không có ứng viên: không ghi gì
	if plan = Route(set, intent.Result{Intent: byProject}, contracts.NormalizedInput{}, contracts.Principal{Roles: []string{"director"}}, pol, Settings{}, lineage); plan.Narrowed.Slot != "" {
		t.Errorf("không có ứng viên thì không có lần gỡ nào: %+v", plan.Narrowed)
	}
}
