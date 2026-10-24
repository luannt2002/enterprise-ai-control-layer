package router

import (
	"slices"
	"testing"

	"github.com/luannt2002/enterprise-ai-control-layer/core/contracts"
	"github.com/luannt2002/enterprise-ai-control-layer/intelligence/intent"
)

// F-005: kiểu CHA dùng để gỡ bí danh mơ hồ theo ngữ cảnh do pack khai (intents.yaml narrow_parent), không gán cứng tên kiểu của một
// miền: pack có kiểu cha "khach_hang" thu hẹp đúng và vết ghi tên kiểu + tên gọi của pack; pack không khai thì không gỡ theo cha; khe
// điền bằng kiểu khác kiểu cha đã khai thì không gỡ.
func TestNarrowTheoKieuChaDoPackKhai(t *testing.T) {
	set := whySet(t)
	a, b := contracts.EntityRef{ID: "DH-07A", Key: "DH-07A", Type: "don_hang"}, contracts.EntityRef{ID: "DH-07B", Key: "DH-07B", Type: "don_hang"}
	lin := map[string][]string{"DH-07A": {"DH-07A", "KH-1"}, "DH-07B": {"DH-07B", "KH-2"}, "KH-1": {"KH-1"}}
	lineage := func(e contracts.EntityRef) []string { return lin[e.Key] }
	pol := Policy{Roles: map[string]RoleDef{"quan_ly": {Capabilities: []string{"metrics.read"}, Scope: "all"}}}
	pr := contracts.Principal{Roles: []string{"quan_ly"}}
	kh := contracts.EntityRef{ID: "KH-1", Key: "KH-1", Type: "khach_hang"}
	filled := contracts.Intent{Name: "q", Slots: map[string]contracts.Slot{"subject": {Name: "subject", Entity: &kh}}}
	cls := intent.Result{Intent: filled, Candidates: []contracts.EntityRef{a, b}}

	st := Settings{NarrowParent: ParentType{Type: "khach_hang", Label: "khách hàng"}}
	plan := Route(set, cls, contracts.NormalizedInput{}, pr, pol, st, lineage)
	if n := plan.Narrowed; n.By != "khach_hang" || n.ByLabel != "khách hàng" || !slices.Equal(n.After, []string{"DH-07A"}) || n.Chosen != "DH-07A" {
		t.Errorf("gỡ theo kiểu cha của pack: %+v", n)
	}
	if plan.Route != contracts.RouteSQLQuery {
		t.Errorf("gỡ được thì đi tiếp, route %s", plan.Route)
	}
	for name, st := range map[string]Settings{
		"pack không khai kiểu cha": {},
		"khe điền bằng kiểu khác":  {NarrowParent: ParentType{Type: "vung", Label: "vùng"}},
	} {
		if n := Route(set, cls, contracts.NormalizedInput{}, pr, pol, st, lineage).Narrowed; n.Slot != "" {
			t.Errorf("%s: không được gỡ theo cha, được %+v", name, n)
		}
	}
}
