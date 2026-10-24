package tenant

import (
	"strings"
	"testing"
)

// F-007: pack có ý định route WORKFLOW (hay unmatched_route là route chưa cài) bị từ chối LÚC NẠP — trước đây pack nạp được, serve
// khởi động, và mọi câu khớp ý định đó trả 500.
func TestLoadPackRouteChuaCaiBiChan(t *testing.T) {
	for name, intents := range map[string]string{
		"ý định route WORKFLOW": strings.Replace(validIntents, "route: SQL_QUERY", "route: WORKFLOW", 1),
		"unmatched_route AGENT": strings.Replace(validIntents, "unmatched_route: REJECT", "unmatched_route: AGENT", 1),
	} {
		_, err := Load(writePack(t, fullPack(map[string]string{"intents.yaml": intents})))
		if err == nil || !strings.Contains(err.Error(), "chưa cài") {
			t.Errorf("%s: Load phải lỗi nói rõ route chưa cài, được %v", name, err)
		}
	}
}
