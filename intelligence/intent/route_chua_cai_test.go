package intent

import (
	"strings"
	"testing"

	"github.com/luannt2002/enterprise-ai-control-layer/core/contracts"
)

// F-007: route thuộc 12 route nhưng luồng hỏi CHƯA cài (DIRECT_LLM, FULL_TEXT_SEARCH, TOOL, WORKFLOW, AGENT) bị từ chối LÚC NẠP pack —
// trước đây pack vẫn nạp được và mọi lượt khớp ý định đó hỏng 500 lúc chạy. Route đã cài vẫn nạp được.
func TestRouteChuaCaiBiChanLucNap(t *testing.T) {
	yes := func(string) bool { return true }
	chuaCai := map[contracts.RouteKind]bool{contracts.RouteDirectLLM: true, contracts.RouteFullTextSearch: true, contracts.RouteTool: true,
		contracts.RouteWorkflow: true, contracts.RouteAgent: true}
	for _, r := range contracts.AllRoutes {
		_, err := Compile([]Def{{Name: "q", Route: string(r), Keywords: []string{"cấp"}, Capability: "metrics.read"}}, yes, yes)
		switch {
		case chuaCai[r] && (err == nil || !strings.Contains(err.Error(), "chưa cài")):
			t.Errorf("route %s chưa cài: Compile phải lỗi nói rõ chưa cài, được %v", r, err)
		case !chuaCai[r] && err != nil:
			t.Errorf("route %s đã cài: Compile phải nạp được, được %v", r, err)
		}
	}
}
