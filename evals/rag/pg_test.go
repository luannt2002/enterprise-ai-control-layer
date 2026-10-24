package ragcases

import (
	"context"
	"fmt"
	"path/filepath"
	"slices"
	"testing"
	"time"

	"github.com/luannt2002/enterprise-ai-control-layer/core/contracts"
	"github.com/luannt2002/enterprise-ai-control-layer/evals"
	"github.com/luannt2002/enterprise-ai-control-layer/intelligence/rag"
	"github.com/luannt2002/enterprise-ai-control-layer/platform/pgstore"
)

// Đường PG của ứng dụng (cmd/eais → pgstore, KE-HOACH U6) phải trả ĐÚNG như chỉ mục trong bộ nhớ trên mọi ca tri thức: cùng
// kết quả, cùng đoạn theo cùng thứ tự, cùng bản trong thông báo. Phía PG chỉ nhận ngày hỏi qua ngữ cảnh của lượt (như pipeline
// đặt) — không có Now gắn vào chỉ mục. Trước U6 bảng documents chỉ giữ id / tiêu đề / thân / acl: bản nháp, bản hết hiệu lực,
// bản ngoài phạm vi bị phục vụ như bản chính, còn hai bản cùng mã thì nạp hỏng.
func TestCaTriThucQuaPG(t *testing.T) {
	admin, app := pgstore.EnvDSN()
	if admin == "" || app == "" {
		t.Skip("chưa có EAIS_PG_ADMIN_DSN / EAIS_PG_DSN — xem deploy/docker-compose.yml")
	}
	ctx := context.Background()
	if err := pgstore.Migrate(ctx, admin); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	s, err := pgstore.Open(ctx, app)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	t.Cleanup(s.Close)
	cases, err := evals.Load(casesYAML)
	if err != nil {
		t.Fatal(err)
	}
	byID := map[string]evals.Case{}
	for _, c := range cases {
		byID[fmt.Sprint(c["id"])] = c
	}
	files, _ := filepath.Glob(filepath.Join(caDir, "*", "ky-vong.yaml"))
	if len(files) < minCa {
		t.Fatalf("quét ra %d ca, cần ít nhất %d", len(files), minCa)
	}
	for i, f := range files {
		ca := napCa(t, filepath.Dir(f), f, byID)
		t.Run(ca.kv.Ca, func(t *testing.T) {
			mem := rag.NewMemoryIndex(string(ca.pr.TenantID), ca.docs)
			mem.Now = func() time.Time { return ca.day }
			evM, errM := mem.Retrieve(ctx, ca.pr, ca.kv.Hoi, ca.kv.K)

			// Tenant riêng mỗi lượt chạy; người hỏi khác tenant tài liệu (ca rò chéo) thì giữ khác tenant ở phía PG.
			docTenant := contracts.TenantID(fmt.Sprintf("ragpg-%d-%d", time.Now().UnixNano(), i))
			pr := ca.pr
			pr.TenantID = docTenant
			if string(ca.pr.TenantID) != ca.docs[0].Tenant {
				pr.TenantID = docTenant + "-khac"
			}
			tctx := contracts.WithTenant(ctx, docTenant)
			if err := s.Load(tctx, nil, nil, ca.docs); err != nil {
				t.Fatalf("nạp PG: %v", err)
			}
			t.Cleanup(func() { _ = s.Load(tctx, nil, nil, nil) }) // xoá dòng của tenant thử
			qctx := contracts.WithKnowledgeQuery(contracts.WithTenant(ctx, pr.TenantID), contracts.KnowledgeQuery{AsOf: ca.day})
			evP, errP := s.Retrieve(qctx, pr, ca.kv.Hoi, ca.kv.K)

			gotM, nM, eM := phanLoai(evM, errM)
			gotP, nP, eP := phanLoai(evP, errP)
			if eM != nil || eP != nil {
				t.Fatalf("lỗi truy xuất: bộ nhớ %v · PG %v", eM, eP)
			}
			if gotM != gotP || !slices.Equal(ids(evM), ids(evP)) {
				t.Fatalf("PG lệch bộ nhớ:\n  bộ nhớ %s %v\n  PG     %s %v", gotM, ids(evM), gotP, ids(evP))
			}
			if (nM == nil) != (nP == nil) || nM != nil && fmt.Sprint(nM.Versions) != fmt.Sprint(nP.Versions) {
				t.Fatalf("thông báo lệch: bộ nhớ %+v · PG %+v", nM, nP)
			}
		})
	}
}

// Phạm vi tài liệu qua PG: bộ ca chưa có ca chặn phạm vi trong CÙNG tenant (CON-T-071 là chéo tenant), nên đột biến "PG không
// lưu phạm vi" sống sót qua phép so ở trên. Tài liệu của PRJ-A: người phạm vi PRJ-B không thấy gì, PRJ-A thấy; luật phạm vi của
// policy trong ngữ cảnh (vai scope "all") thắng thuộc tính trần.
func TestPhamViQuaPG(t *testing.T) {
	admin, app := pgstore.EnvDSN()
	if admin == "" || app == "" {
		t.Skip("chưa có EAIS_PG_ADMIN_DSN / EAIS_PG_DSN — xem deploy/docker-compose.yml")
	}
	ctx := context.Background()
	if err := pgstore.Migrate(ctx, admin); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	s, err := pgstore.Open(ctx, app)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	t.Cleanup(s.Close)
	tenant := contracts.TenantID(fmt.Sprintf("ragpg-phamvi-%d", time.Now().UnixNano()))
	tctx := contracts.WithTenant(ctx, tenant)
	doc := rag.Document{ID: "SOP-PRJ-A", Title: "Quy trình đổ bê tông dự án A", ACL: []string{"site_engineer"}, Version: 1,
		Status: rag.StatusApproved, Scope: []string{"PRJ-A"}, Body: "1. Kiểm tra cốp pha trước khi đổ bê tông."}
	if err := s.Load(tctx, nil, nil, []rag.Document{doc}); err != nil {
		t.Fatalf("nạp PG: %v", err)
	}
	t.Cleanup(func() { _ = s.Load(tctx, nil, nil, nil) })
	hoi := func(scope string, inScope func(contracts.Principal, []string) bool) int {
		pr := contracts.Principal{ID: "u", TenantID: tenant, Roles: []string{"site_engineer"}, Attributes: map[string]string{"entity_scope": scope}}
		qctx := contracts.WithKnowledgeQuery(tctx, contracts.KnowledgeQuery{AsOf: time.Date(2026, 10, 20, 0, 0, 0, 0, time.UTC), InScope: inScope})
		ev, err := s.Retrieve(qctx, pr, "quy trình đổ bê tông", 3)
		if err != nil {
			t.Fatalf("phạm vi %q: %v", scope, err)
		}
		return len(ev)
	}
	if n := hoi("PRJ-B", nil); n != 0 {
		t.Errorf("phạm vi PRJ-B thấy %d đoạn của tài liệu PRJ-A", n)
	}
	if n := hoi("PRJ-A", nil); n == 0 {
		t.Error("phạm vi PRJ-A phải thấy tài liệu của PRJ-A")
	}
	if n := hoi("PRJ-B", func(contracts.Principal, []string) bool { return true }); n == 0 {
		t.Error("luật phạm vi trong ngữ cảnh cho phép mà vẫn không thấy")
	}
}
