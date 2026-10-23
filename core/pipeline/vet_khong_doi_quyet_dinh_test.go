package pipeline

import (
	"context"
	"os"
	"reflect"
	"testing"
	"time"

	"go.yaml.in/yaml/v3"

	"github.com/luannt2002/enterprise-ai-control-layer/core/contracts"
	"github.com/luannt2002/enterprise-ai-control-layer/core/tenant"
	"github.com/luannt2002/enterprise-ai-control-layer/intelligence/models"
	"github.com/luannt2002/enterprise-ai-control-layer/intelligence/rag"
	"github.com/luannt2002/enterprise-ai-control-layer/observability/traces"
	"github.com/luannt2002/enterprise-ai-control-layer/simulations/fixtures"
)

// C11 (lệnh ca chiều 23/10): ghi giải thích của vết (why.go) chỉ GHI LẠI đã quyết gì, không được đổi quyết định. So kế hoạch route, câu trả
// lời (route, outcome, ý định, câu, trích dẫn, xung đột, câu hỏi lại, phán quyết) VÀ bản ghi quyết định (Decision, bỏ TraceID và phần Why —
// soát C8–C11 C11-S3: đột biến làm explain* sửa dữ liệu dùng chung với Decision từng sống sót) của cùng một lượt khi có vết và khi tắt ghi
// (noWhy), trên mọi câu hỏi của bộ 117 và H1 / H3 / H4 (đã công bố) cộng vài câu dựng thêm có lượt CHUYỂN NGƯỜI (bộ công bố không có lượt
// nào), với hai người hỏi: toàn quyền và phạm vi hạn chế. Giới hạn: answerKnowledge vẫn ghi ở cả hai chế độ (nhánh RAG đang sửa hàm đó), nên
// phần S7 của route tri thức không nằm trong phép so này.
func TestVetKhongDoiQuyetDinh(t *testing.T) {
	rt, err := tenant.Load("../../tenants/construction/pack")
	if err != nil {
		t.Fatalf("nạp pack: %v", err)
	}
	dir, err := fixtures.LoadDirectory("../../tenants/construction/seed/entities.yaml")
	if err != nil {
		t.Fatalf("nạp danh mục: %v", err)
	}
	facts, err := fixtures.LoadFacts("../../tenants/construction/seed/facts.yaml", rt, dir)
	if err != nil {
		t.Fatalf("nạp fact: %v", err)
	}
	now := time.Date(2026, 10, 19, 10, 0, 0, 0, time.FixedZone("ICT", 7*3600))
	with := &Ask{RT: rt, Deps: Deps{Directory: dir, Facts: facts, Now: func() time.Time { return now }, DataVersion: "seed-test",
		Knowledge: rag.NewMemoryIndex("abc-infra", []rag.Document{{ID: "SOP-FUEL-01", Tenant: "abc-infra", Title: "Quy trình xử lý chênh lệch nhiên liệu",
			ACL: []string{"director"}, Body: "1. Khi lượng xuất kho và lượng tiêu thụ đo được lệch quá dung sai, tổ trưởng kho lập biên bản đối soát."}}),
		Model: &models.Extractive{}}}
	without := *with
	without.noWhy = true

	var questions []string
	for _, f := range []string{"../../tenants/construction/win/cases-independent.yaml", "../../docs/g7/holdout/h1.yaml",
		"../../docs/g7/holdout-h3/h3.yaml", "../../docs/g7/holdout-h4/h4.yaml"} {
		b, err := os.ReadFile(f)
		if err != nil {
			t.Fatalf("đọc %s: %v", f, err)
		}
		var doc struct {
			Cases []struct {
				Question string `yaml:"question"`
			} `yaml:"cases"`
		}
		if err := yaml.Unmarshal(b, &doc); err != nil {
			t.Fatalf("đọc %s: %v", f, err)
		}
		for _, c := range doc.Cases {
			questions = append(questions, c.Question)
		}
	}
	if len(questions) < 400 {
		t.Fatalf("chỉ đọc được %d câu hỏi", len(questions))
	}
	// lượt chuyển người (HUMAN_ESCALATION: hai nguồn ngang thẩm quyền của DZ-03 lệch quá dung sai trong seed) và vài câu mơ hồ / nhiều thực thể
	questions = append(questions, "DZ-03 hôm nay được cấp bao nhiêu dầu?", "DZ-03 hôm nay được cấp 3000 L dầu đúng không?",
		"RL-02 và máy 03 hôm nay được cấp bao nhiêu dầu?", "Máy 03 của dự án B hôm nay được cấp bao nhiêu dầu?")
	people := []contracts.Principal{
		{ID: "u-dir", TenantID: "abc-infra", Kind: contracts.PrincipalUser, Roles: []string{"director"}},
		{ID: "u-eng", TenantID: "abc-infra", Kind: contracts.PrincipalUser, Roles: []string{"site_engineer"}, Attributes: map[string]string{"entity_scope": "PRJ-A"}},
	}
	run := func(a *Ask, pr contracts.Principal, q string) (contracts.RoutePlan, contracts.AskResponse, traces.Decision, string) {
		out, err := a.Handle(context.Background(), pr, contracts.AskRequest{ConversationID: "c", MessageID: "m", Text: q})
		resp, d := out.Response, out.Decision
		resp.TraceID, d.TraceID = "", "" // mã vết sinh mới mỗi lượt
		d.Why = traces.Why{}             // phần giải thích là thứ duy nhất được phép khác
		return out.Plan, resp, d, contracts.FaultOf(err)
	}
	diffs, escalations := 0, 0
	for _, pr := range people {
		for _, q := range questions {
			p1, r1, d1, e1 := run(with, pr, q)
			p2, r2, d2, e2 := run(&without, pr, q)
			if r1.Route == contracts.RouteHumanEscalation {
				escalations++
			}
			if e1 != e2 || !reflect.DeepEqual(p1, p2) || !reflect.DeepEqual(r1, r2) || !reflect.DeepEqual(d1, d2) {
				diffs++
				if diffs <= 5 {
					t.Errorf("%s / %q: có vết và không vết khác nhau\n  lỗi %q / %q\n  route %s / %s, outcome %s / %s\n  câu %q\n  / %q", pr.ID, q, e1, e2,
						r1.Route, r2.Route, r1.Outcome, r2.Outcome, r1.Answer, r2.Answer)
				}
			}
		}
	}
	if diffs > 0 {
		t.Fatalf("%d / %d lượt đổi quyết định hay câu trả lời khi tắt ghi giải thích", diffs, len(people)*len(questions))
	}
	if escalations == 0 {
		t.Fatalf("quần thể không có lượt chuyển người nào — nhánh ghi đè route của explainRoute không nằm trong phép so")
	}
}
