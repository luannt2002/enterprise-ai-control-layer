package router

import (
	"testing"

	"github.com/luannt2002/enterprise-ai-systems/core/contracts"
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
