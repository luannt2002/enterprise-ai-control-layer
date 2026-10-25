package construction_test

import (
	"context"
	"maps"
	"reflect"
	"slices"
	"testing"

	"github.com/luannt2002/enterprise-ai-control-layer/core/contracts"
	"github.com/luannt2002/enterprise-ai-control-layer/core/policy"
	"github.com/luannt2002/enterprise-ai-control-layer/simulations/fixtures"
)

// Q11b (Q-P1-10 / Q11 = A, F-031): đường SỐ LIỆU xét phạm vi dữ liệu bằng CHÍNH vai có quyền đọc số liệu — router S5 (bậc phạm vi của khe
// thực thể, gỡ mơ hồ theo phạm vi, câu hỏi nghĩa số liệu), ứng viên của câu hỏi lại, và bộ che phạm vi của vết. Policy thử nghiệm, pack không
// đổi: thêm vai "xem_tat_ca" phạm vi all mà KHÔNG có capability nào. Người hai vai [site_engineer PRJ-A + xem_tat_ca] phải nhận ĐÚNG câu trả
// lời của site_engineer PRJ-A một mình (vai không có quyền đọc không cho vai khác mượn phạm vi), S3 / S4 của vết giống hệt, và vết không lộ
// thực thể ngoài PRJ-A. Đối chứng dương: [director + xem_tat_ca] vẫn thấy mọi thực thể (không chặn oan người nhiều vai).
func TestHaiVaiSoLieuMotVai(t *testing.T) {
	a, _ := newAskTriThuc(t)
	roles := maps.Clone(a.RT.Policy.Roles)
	roles["xem_tat_ca"] = policy.RoleDef{Scope: "all"}
	a.RT.Policy.Roles = roles
	dir, err := fixtures.LoadDirectory("seed/entities.yaml")
	if err != nil {
		t.Fatalf("nạp danh mục: %v", err)
	}
	all, err := dir.Entities(context.Background())
	if err != nil {
		t.Fatalf("đọc danh mục: %v", err)
	}
	alone := user("site_engineer", "PRJ-A")
	both := alone
	both.Roles = []string{"site_engineer", "xem_tat_ca"}
	hidden := outOfScope(t, a, alone, all)
	if len(hidden) == 0 {
		t.Fatal("site_engineer PRJ-A không có thực thể nào ngoài phạm vi — ca thử vô nghĩa")
	}
	// mỗi chỗ sửa có ít nhất một câu nhắm tới: khe ngoài phạm vi, bí danh mơ hồ gỡ theo phạm vi, câu hỏi nghĩa số liệu, câu hỏi lại kể ứng viên
	questions := append(publishedQuestions(t),
		"DZ-03 hôm nay được cấp bao nhiêu dầu?",
		"Dự án B hôm nay được cấp bao nhiêu dầu?",
		"Máy 03 hôm nay được cấp bao nhiêu dầu?",
		"Máy 07 hôm nay được cấp bao nhiêu dầu?",
		"DZ-03 dầu hôm nay?",
		"EX-03 và DZ-03 hôm nay được cấp bao nhiêu dầu?",
		"Máy 03 của dự án B hôm nay được cấp bao nhiêu dầu?",
		"Máy 03 hôm nay được cấp 3850 L đúng không?",
		"So sánh dầu được cấp của DZ-03 hôm nay và hôm qua",
	)
	type luot struct {
		resp     contracts.AskResponse
		why      []any
		entities []string
	}
	run := func(pr contracts.Principal, q string) luot {
		out := ask(t, a, pr, q)
		out.Response.TraceID = ""
		d := out.Decision
		return luot{out.Response, []any{d.Why.Normalize, d.Why.Resolve, d.Why.Intent}, slices.Concat(d.Entities, d.Slots)}
	}
	diff, leaks := 0, 0
	for _, q := range questions {
		ra, rb := run(alone, q), run(both, q)
		if !reflect.DeepEqual(ra, rb) {
			if diff++; diff <= 6 {
				t.Errorf("%q: người hai vai khác site_engineer PRJ-A một mình\n  một vai: %s %s %q ứng viên %v\n  hai vai: %s %s %q ứng viên %v",
					q, ra.resp.Route, ra.resp.Outcome, ra.resp.Answer, candidatesOf(ra.resp), rb.resp.Route, rb.resp.Outcome, rb.resp.Answer, candidatesOf(rb.resp))
			}
		}
		for _, hit := range leaked(t, a, both, q, hidden) {
			if leaks++; leaks <= 6 {
				t.Errorf("%q: vết của người hai vai lộ %s", q, hit)
			}
		}
	}
	if diff+leaks > 0 {
		t.Fatalf("%d câu khác người một vai, %d chỗ lộ trên %d câu", diff, leaks, len(questions))
	}
	// đối chứng dương: vai có metrics.read phạm vi all cộng vai không quyền — y hệt director một mình, đọc được DZ-03 (Holding không bỏ vai đủ quyền)
	director := user("director", "")
	dirBoth := director
	dirBoth.Roles = []string{"director", "xem_tat_ca"}
	for _, q := range []string{"DZ-03 hôm nay được cấp bao nhiêu dầu?", "Máy 03 hôm nay được cấp bao nhiêu dầu?"} {
		if ra, rb := run(director, q), run(dirBoth, q); !reflect.DeepEqual(ra, rb) || rb.resp.Outcome == contracts.OutcomePermissionDenied {
			t.Errorf("đối chứng %q: [director + xem_tat_ca] được %s %q — muốn y hệt director một mình (%s %q), không từ chối",
				q, rb.resp.Outcome, rb.resp.Answer, ra.resp.Outcome, ra.resp.Answer)
		}
	}
	if hits := leaked(t, a, dirBoth, "Máy 03 hôm nay được cấp bao nhiêu dầu?", hidden); len(hits) == 0 {
		t.Errorf("đối chứng: vết của [director + xem_tat_ca] phải kể thực thể ngoài PRJ-A (bộ quét không mù, bộ che không che oan)")
	}
}

func candidatesOf(r contracts.AskResponse) []string {
	if r.Clarification == nil {
		return nil
	}
	var out []string
	for _, c := range r.Clarification.Candidates {
		out = append(out, c.Key)
	}
	return out
}
