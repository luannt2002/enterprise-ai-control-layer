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

// N2 (soát vòng 2: X27–X30): dữ kiện của từng bậc kiểm khoá nguyên chữ — capability của vế so sánh (có / thiếu), tên vế so sánh, tên ý định tri thức
// của HYBRID, và ngoài / trong phạm vi của câu hỏi nghĩa số liệu.
func TestRouteDuKienBacSoSanhHybridVaPhamViNghiaSoLieu(t *testing.T) {
	yes := func(string) bool { return true }
	slots := []intent.SlotDef{{Name: "subject", From: "entity", Types: []string{"equipment"}, Required: true}}
	set, err := intent.Compile([]intent.Def{
		{Name: "q", Route: "SQL_QUERY", Keywords: []string{"cấp"}, Capability: "metrics.read", Template: "metric_total", Slots: slots},
		{Name: "c", Route: "SQL_QUERY", Keywords: []string{"báo"}, Capability: "reports.read", Template: "metric_total", Slots: slots},
		{Name: "k", Route: "RAG", Keywords: []string{"quy trình"}, Capability: "documents.read", Template: "sop"},
	}, yes, yes)
	if err != nil {
		t.Fatal(err)
	}
	pol := Policy{Roles: map[string]RoleDef{
		"full":    {Capabilities: []string{"metrics.read", "reports.read", "documents.read"}, Scope: "all"},
		"partial": {Capabilities: []string{"metrics.read"}, Scope: "all"},
		"eng":     {Capabilities: []string{"metrics.read"}, Scope: "attribute"},
	}}
	lineage := func(e contracts.EntityRef) []string { return []string{e.Key} }
	ex := contracts.EntityRef{ID: "EX-03", Key: "EX-03", Type: "equipment"}
	filled := contracts.Intent{Name: "q", Slots: map[string]contracts.Slot{"subject": {Name: "subject", Entity: &ex}}}
	check := func(p contracts.RoutePlan, rule string) contracts.RouteCheck {
		t.Helper()
		for _, c := range p.Checked {
			if c.Rule == rule {
				return c
			}
		}
		t.Fatalf("không có bậc %q trong %v", rule, rules(p))
		return contracts.RouteCheck{}
	}
	route := func(role string, cls intent.Result) contracts.RoutePlan {
		return Route(set, cls, contracts.NormalizedInput{}, contracts.Principal{Roles: []string{role}}, pol, Settings{}, lineage)
	}

	// vế so sánh thiếu capability: bậc chặn nói thiếu capability nào, route REJECT; đủ capability: bậc nói "có" và bậc so sánh nêu tên vế
	deny := route("partial", intent.Result{Intent: filled, Compare: "c"})
	if c := check(deny, "capability của vế so sánh"); !c.Fired || c.Detail != "reports.read: thiếu" || deny.Route != contracts.RouteReject || deny.Reason != "thiếu capability reports.read cho vế so sánh" {
		t.Errorf("vế so sánh thiếu capability: %+v, route %s, lý do %q", c, deny.Route, deny.Reason)
	}
	cmp := route("full", intent.Result{Intent: filled, Compare: "c"})
	if c := check(cmp, "capability của vế so sánh"); c.Fired || c.Detail != "reports.read: có" {
		t.Errorf("vế so sánh đủ capability: %+v", c)
	}
	if c := check(cmp, "câu so sánh hai nghĩa"); !c.Fired || c.Detail != "c" || c.Route != contracts.RouteCalculation || cmp.Route != contracts.RouteCalculation || cmp.Reason != "ý định q khớp luật, so với c" {
		t.Errorf("bậc so sánh phải nêu tên vế sau: %+v, route %s, lý do %q", c, cmp.Route, cmp.Reason)
	}
	if c := check(route("full", intent.Result{Intent: filled}), "câu so sánh hai nghĩa"); c.Fired || c.Detail != "không có vế so sánh" {
		t.Errorf("không so sánh: %+v", c)
	}

	// HYBRID: phần tri thức cần capability riêng (có / thiếu) và bậc ghép nêu TÊN ý định tri thức
	hyb := route("full", intent.Result{Intent: filled, Knowledge: "k"})
	if c := check(hyb, "capability của phần tri thức"); c.Fired || c.Detail != "documents.read: có" {
		t.Errorf("capability phần tri thức đủ: %+v", c)
	}
	if c := check(hyb, "ý định tri thức khớp kèm"); !c.Fired || c.Detail != "k" || c.Route != contracts.RouteHybrid || hyb.Route != contracts.RouteHybrid || hyb.Reason != "ý định q khớp luật + tri thức k" {
		t.Errorf("bậc HYBRID phải nêu tên ý định tri thức: %+v, route %s, lý do %q", c, hyb.Route, hyb.Reason)
	}
	if c := check(route("partial", intent.Result{Intent: filled, Knowledge: "k"}), "capability của phần tri thức"); !c.Fired || c.Detail != "documents.read: thiếu" {
		t.Errorf("capability phần tri thức thiếu: %+v", c)
	}

	// câu hỏi nghĩa số liệu: bậc phạm vi nói đúng ngoài / trong phạm vi của từng thực thể được nhắc
	eng := contracts.Principal{Roles: []string{"eng"}, Attributes: map[string]string{"entity_scope": "EX-03"}}
	in := Route(set, intent.Result{NeedMeaning: []string{"q"}, Mentioned: []contracts.EntityRef{ex}}, contracts.NormalizedInput{}, eng, pol, Settings{}, lineage)
	if c := check(in, "phạm vi dữ liệu (câu hỏi nghĩa số liệu)"); c.Fired || c.Detail != "EX-03: trong phạm vi" || in.Route != contracts.RouteClarification {
		t.Errorf("trong phạm vi: %+v, route %s", c, in.Route)
	}
	other := contracts.Principal{Roles: []string{"eng"}, Attributes: map[string]string{"entity_scope": "PRJ-B"}}
	out := Route(set, intent.Result{NeedMeaning: []string{"q"}, Mentioned: []contracts.EntityRef{ex}}, contracts.NormalizedInput{}, other, pol, Settings{}, lineage)
	if c := check(out, "phạm vi dữ liệu (câu hỏi nghĩa số liệu)"); !c.Fired || c.Detail != "EX-03: ngoài phạm vi" || out.Route != contracts.RouteReject {
		t.Errorf("ngoài phạm vi: %+v, route %s", c, out.Route)
	}
}
