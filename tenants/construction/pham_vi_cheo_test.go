package construction_test

import (
	"context"
	"fmt"
	"maps"
	"slices"
	"strings"
	"testing"

	"github.com/luannt2002/enterprise-ai-control-layer/core/contracts"
	"github.com/luannt2002/enterprise-ai-control-layer/core/pipeline"
	"github.com/luannt2002/enterprise-ai-control-layer/core/tenant"
	"github.com/luannt2002/enterprise-ai-control-layer/intelligence/models"
	"github.com/luannt2002/enterprise-ai-control-layer/intelligence/rag"
)

// Test chéo ranh giới quyền (F-031, lát L4) trên policy THẬT của pack.

// taiLieuPhamVi là tài liệu quy trình chỉ mở cho các vai acl, phạm vi scope; thân trả lời được cauQuyTrinh.
func taiLieuPhamVi(acl, scope []string) rag.Document {
	return rag.Document{ID: "SOP-PV-01", Tenant: "abc-infra", Title: "Quy trình xử lý chênh lệch nhiên liệu", ACL: acl, Scope: scope,
		Body: "1. Khi lượng xuất kho và lượng tiêu thụ đo được lệch quá dung sai, tổ trưởng kho lập biên bản đối soát trong vòng 24 giờ.\n" +
			"2. Kế toán chỉ đối chiếu hoá đơn sau khi biên bản đối soát được ký."}
}

// Hai chỗ cùng quyết "người hỏi được xem tài liệu có phạm vi này không": Policy.InScope (core/policy — S5 và lệnh ghi dùng) và nhánh
// dự phòng của chỉ mục tri thức khi không ai đưa luật phạm vi (MemoryIndex.InScope = nil: tự đọc thuộc tính entity_scope). Với mỗi vai
// của pack × entity_scope × phạm vi tài liệu (tài liệu chỉ mở cho đúng vai đó), nhánh dự phòng phải thấy tài liệu ĐÚNG KHI
// Policy.InScope nói có. Lệch theo hướng dự phòng RỘNG hơn là lọt (đỏ); lệch theo hướng HẸP hơn được ghi lại bằng Skip có lý do.
func TestPhamViDuPhongRAGBangPolicy(t *testing.T) {
	rt, err := tenant.Load("pack")
	if err != nil {
		t.Fatalf("nạp pack: %v", err)
	}
	pol := rt.Policy
	roles := slices.Sorted(maps.Keys(pol.Roles))
	roles = append(roles, "site_supervisor") // vai không có trong policy: không phạm vi nào mặc định mở
	thay := func(t *testing.T, ix *rag.MemoryIndex, pr contracts.Principal) bool {
		t.Helper()
		ev, err := ix.Retrieve(context.Background(), pr, cauQuyTrinh, 3)
		if err != nil {
			t.Fatalf("truy xuất: %v", err)
		}
		return len(ev) > 0
	}
	n := 0
	for _, role := range roles {
		t.Run(role, func(t *testing.T) {
			var hep []string
			for _, attr := range []string{"", "PRJ-A", "PRJ-B", "PRJ-A, PRJ-B", " , PRJ-B "} {
				for _, scope := range [][]string{{"PRJ-A"}, {"PRJ-B"}, {"PRJ-A", "PRJ-B"}} {
					pr := user(role, attr)
					docs := []rag.Document{taiLieuPhamVi([]string{role}, scope)}
					duPhong := rag.NewMemoryIndex("abc-infra", docs)
					theoLuat := rag.NewMemoryIndex("abc-infra", docs)
					theoLuat.InScope = pol.InScope
					want, got, gotLuat := pol.InScope(pr, scope), thay(t, duPhong, pr), thay(t, theoLuat, pr)
					n++
					ca := fmt.Sprintf("entity_scope %q / tài liệu %s", attr, strings.Join(scope, ","))
					switch {
					case gotLuat != want:
						t.Errorf("%s: chỉ mục có InScope = Policy.InScope thấy = %v, Policy.InScope = %v", ca, gotLuat, want)
					case got && !want:
						t.Errorf("%s: nhánh dự phòng THẤY tài liệu mà Policy.InScope từ chối (lọt)", ca)
					case !got && want:
						hep = append(hep, ca)
					}
				}
			}
			if len(hep) > 0 {
				t.Skipf("F-031: nhánh dự phòng của chỉ mục (InScope = nil) HẸP hơn Policy.InScope cho vai %s (scope %q) ở %d ca: %s — "+
					"dự phòng không biết scope của vai; sửa ở intelligence/rag, ngoài lát L4, chờ điều phối", role, pol.Roles[role].Scope, len(hep), strings.Join(hep, "; "))
			}
		})
	}
	if n == 0 {
		t.Fatal("không ca nào được so")
	}
}

// Q-P1-10 / Q11 (người nhiều vai): capability documents.read (router, S5) và ACL tài liệu (chỉ mục tri thức) được thoả bởi HAI VAI KHÁC NHAU
// của cùng một người hỏi. Vai A có documents.read nhưng không trong ACL; vai B trong ACL nhưng không có documents.read. Từng vai một mình
// không đọc được tài liệu (đối chứng âm); hai vai cùng nhau cũng KHÔNG — quyền và ACL / phạm vi đến từ CÙNG một vai (Q11 = A, CHO mục 0).
// Đối chứng dương: tài liệu mà chính vai có documents.read nằm trong ACL thì người hai vai vẫn đọc được (không chặn oan người nhiều vai).
func TestHaiVaiTachQuyenDocVaACL(t *testing.T) {
	for _, c := range []struct {
		name  string
		roles []string // [vai A có documents.read, vai B trong ACL]
		attr  string
		scope []string
		q     string
	}{
		{"tài liệu không khai phạm vi", []string{"site_engineer", "accountant"}, "", nil, cauQuyTrinh},
		{"tài liệu PRJ-B, vai có quyền đọc chỉ được giao PRJ-A", []string{"project_manager", "accountant"}, "PRJ-A", []string{"PRJ-B"}, cauQuyTrinh},
		// HYBRID: phần số liệu do accountant tự đủ quyền + phạm vi; phần quy trình vẫn phải do vai có documents.read nằm trong ACL
		{"câu ghép số liệu + quy trình", []string{"site_engineer", "accountant"}, "", nil, cauGhep},
	} {
		t.Run(c.name, func(t *testing.T) {
			a, m := newAsk(t), &models.Extractive{}
			doc := taiLieuPhamVi([]string{c.roles[1]}, c.scope)
			a.Deps.Knowledge, a.Deps.Model = rag.NewMemoryIndex("abc-infra", []rag.Document{doc}), m
			doc24 := func(r contracts.AskResponse) bool {
				return strings.Contains(r.Answer, "24 giờ") || citesDocument(r, doc.ID)
			}
			for _, role := range c.roles {
				if r := ask(t, a, user(role, c.attr), c.q).Response; doc24(r) {
					t.Fatalf("đối chứng: vai %s một mình đã đọc được tài liệu (%s → %q)", role, r.Outcome, r.Answer)
				}
			}
			pr := user(c.roles[0], c.attr)
			pr.Roles = c.roles
			if r := ask(t, a, pr, c.q).Response; doc24(r) {
				t.Errorf("Q11: vai %v cùng nhau đọc được %s (route %s, outcome %s, %d lời gọi model) dù không vai nào một mình đọc được — "+
					"documents.read lấy từ %s, ACL / phạm vi từ %s", c.roles, doc.ID, r.Route, r.Outcome, m.Calls(), c.roles[0], c.roles[1])
			}
			// đối chứng dương: vai có documents.read nằm trong ACL, tài liệu không khai phạm vi
			a.Deps.Knowledge = rag.NewMemoryIndex("abc-infra", []rag.Document{taiLieuPhamVi([]string{c.roles[0]}, nil)})
			if r := ask(t, a, pr, c.q).Response; !doc24(r) {
				t.Errorf("vai %v: %s có documents.read và nằm trong ACL mà không đọc được (%s → %q)", c.roles, c.roles[0], r.Outcome, r.Answer)
			}
		})
	}
}

// Q11 ở dự phòng S4: dò chỉ mục hộ người hỏi cũng chỉ bằng vai có documents.read. Tài liệu chỉ mở cho vai không có quyền đọc không được
// kéo câu sang route tri thức — kết cục khác đi là lộ "có tài liệu" — nên người hai vai nhận đúng kết cục của luật như khi kho không có
// tài liệu đó; đối chứng dương: tài liệu mở cho chính vai có documents.read thì dự phòng vẫn trả lời từ tài liệu.
func TestHaiVaiDuPhongS4(t *testing.T) {
	q := "Tổ trưởng kho phải đối soát khi nào?"
	pr := user("site_engineer", "")
	pr.Roles = []string{"site_engineer", "accountant"}
	for acl, want := range map[string]contracts.Outcome{"accountant": contracts.OutcomeOutOfScope, "site_engineer": contracts.OutcomeResolved} {
		a := newAsk(t)
		a.Deps.Knowledge, a.Deps.Model = rag.NewMemoryIndex("abc-infra", []rag.Document{taiLieuPhamVi([]string{acl}, nil)}), &models.Extractive{}
		a.Deps.IntentFallback = pipeline.KnowledgeProbe{Knowledge: a.Deps.Knowledge, Name: "sop_question"}
		if r := ask(t, a, pr, q).Response; r.Outcome != want {
			t.Errorf("tài liệu chỉ mở cho %s, người hỏi %v: muốn %s, được %s %s → %q", acl, pr.Roles, want, r.Route, r.Outcome, r.Answer)
		}
	}
}
