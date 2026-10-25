package pgstore

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/luannt2002/enterprise-ai-control-layer/core/contracts"
	"github.com/luannt2002/enterprise-ai-control-layer/intelligence/rag"
)

// nhungGhiPG ghi lại mọi chữ embedder nhận.
type nhungGhiPG struct {
	nhungDem
	da []string
}

func (e *nhungGhiPG) Embed(ctx context.Context, texts []string, q bool) ([][]float32, error) {
	e.da = append(e.da, texts...)
	return e.nhungDem.Embed(ctx, texts, q)
}

// EmbedKnowledge trên PG chỉ gửi ra chữ của bản đã duyệt (F-017): bản nháp / chờ soát của tenant nằm trong kho nhưng chữ của chúng không
// tới embedder; lượt hỏi đường lai vẫn chạy (không đòi vector của bản chưa duyệt).
func TestNhungKhongGuiBanNhapPG(t *testing.T) {
	admin, app := EnvDSN()
	if admin == "" || app == "" {
		t.Skip("chưa có EAIS_PG_ADMIN_DSN / EAIS_PG_DSN — xem deploy/docker-compose.yml")
	}
	ctx := context.Background()
	if err := Migrate(ctx, admin); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	s, err := Open(ctx, app)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(s.Close)
	e := &nhungGhiPG{nhungDem: nhungDem{ten: "gia/3"}}
	s.Semantic = &rag.Semantic{Embedder: e}
	tenant := contracts.TenantID(fmt.Sprintf("nhung-nhap-%d", time.Now().UnixNano()))
	tctx := contracts.WithTenant(ctx, tenant)
	t.Cleanup(func() {
		_ = s.Load(tctx, nil, nil, nil)
		_, _, _ = s.EmbedKnowledge(ctx, tenant)
	})
	const canary = "CANARY-PG-51D"
	ks := []string{"ks"}
	if err := s.Load(tctx, nil, nil, []rag.Document{
		{ID: "QT", Version: 1, Status: rag.StatusApproved, ACL: ks, Title: "Quy trình nghiệm thu", Body: "1. Biên bản nghiệm thu do tư vấn giám sát lập."},
		{ID: "QT", Version: 2, Status: rag.StatusDraft, ACL: ks, Title: "Quy trình nghiệm thu", Body: "1. Biên bản nghiệm thu " + canary + " nháp."},
		{ID: "SOAT", Version: 1, Status: rag.StatusNeedsReview, Hold: "bảng không đủ ô", ACL: ks, Title: "Quy trình bàn giao", Body: "1. Bàn giao " + canary + " chờ soát."},
	}); err != nil {
		t.Fatal(err)
	}
	if added, _, err := s.EmbedKnowledge(ctx, tenant); err != nil || added != 1 {
		t.Fatalf("muốn nhúng đúng 1 chữ (bản đã duyệt), được %d, %v", added, err)
	}
	pr := contracts.Principal{ID: "u", TenantID: tenant, Kind: contracts.PrincipalUser, Roles: ks}
	if ev, err := s.Retrieve(tctx, pr, "biên bản nghiệm thu", 3); err != nil || len(ev) == 0 || ev[0].ID != "QT@v1#0" {
		t.Errorf("đường lai trên PG phải trả bản đã duyệt: %v %v", ev, err)
	}
	if strings.Contains(strings.Join(e.da, "\n"), canary) {
		t.Errorf("chữ của bản chưa duyệt đã gửi ra embedder: %q", e.da)
	}
}
