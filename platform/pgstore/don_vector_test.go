package pgstore

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/luannt2002/enterprise-ai-control-layer/core/contracts"
	"github.com/luannt2002/enterprise-ai-control-layer/intelligence/rag"
)

// Đổi model nhúng (không gian vector khác) rồi dọn (F-022): EmbedKnowledge chỉ dọn trong không gian hiện tại, PruneVectorSpaces xoá
// vector của tenant ở mọi không gian khác — sau khi dọn còn 0 dòng của không gian cũ; không gian giữ lại và tenant khác còn nguyên;
// không nêu không gian giữ lại là lỗi, không xoá gì.
func TestDonVectorKhongGianCu(t *testing.T) {
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
	n := time.Now().UnixNano()
	a, b := contracts.TenantID(fmt.Sprintf("don-a-%d", n)), contracts.TenantID(fmt.Sprintf("don-b-%d", n))
	docs := []rag.Document{{ID: "QT", ACL: []string{"r"}, Title: "Quy trình", Body: "1. Bước một.\n2. Bước hai."}}
	nhung := func(tn contracts.TenantID, space string) {
		t.Helper()
		s.Semantic = &rag.Semantic{Embedder: &nhungDem{ten: space}}
		if _, _, err := s.EmbedKnowledge(ctx, tn); err != nil {
			t.Fatal(err)
		}
	}
	for _, tn := range []contracts.TenantID{a, b} {
		tctx := contracts.WithTenant(ctx, tn)
		if err := s.Load(tctx, nil, nil, docs); err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() {
			_ = s.Load(tctx, nil, nil, nil)
			for _, sp := range []string{"cu/3", "moi/3"} {
				s.Semantic = &rag.Semantic{Embedder: &nhungDem{ten: sp}}
				_, _, _ = s.EmbedKnowledge(ctx, tn)
			}
		})
		nhung(tn, "cu/3")
	}
	nhung(a, "moi/3")
	dem := func(tn contracts.TenantID, space string) int {
		var c int
		if err := s.inTenant(ctx, tn, func(tx pgx.Tx) error {
			return tx.QueryRow(ctx, "SELECT count(*) FROM document_vectors WHERE space = $1", space).Scan(&c)
		}); err != nil {
			t.Fatal(err)
		}
		return c
	}
	if dem(a, "cu/3") != 2 || dem(a, "moi/3") != 2 || dem(b, "cu/3") != 2 {
		t.Fatalf("tiền đề: mỗi không gian 2 vector, được A cũ %d, A mới %d, B cũ %d", dem(a, "cu/3"), dem(a, "moi/3"), dem(b, "cu/3"))
	}
	if _, err := s.PruneVectorSpaces(ctx, a, ""); contracts.FaultOf(err) != contracts.FaultInput || dem(a, "cu/3") != 2 {
		t.Errorf("không nêu không gian giữ lại: muốn lỗi đầu vào, không xoá gì; được %v, còn %d", err, dem(a, "cu/3"))
	}
	removed, err := s.PruneVectorSpaces(ctx, a, "moi/3")
	if err != nil || removed != 2 {
		t.Fatalf("dọn: xoá %d, %v (muốn 2)", removed, err)
	}
	if dem(a, "cu/3") != 0 || dem(a, "moi/3") != 2 || dem(b, "cu/3") != 2 {
		t.Errorf("sau khi dọn: A cũ %d (muốn 0), A mới %d (muốn 2), B cũ %d (muốn 2)", dem(a, "cu/3"), dem(a, "moi/3"), dem(b, "cu/3"))
	}
}

// Lệnh dọn lọc tenant ngay trong câu lệnh (như Load, F-037): chạy bằng vai vượt RLS, dọn tenant A không chạm vector của tenant B.
func TestDonVectorVaiVuotRLSChiTenantMinh(t *testing.T) {
	admin, app := EnvDSN()
	if admin == "" || app == "" {
		t.Skip("chưa có EAIS_PG_ADMIN_DSN / EAIS_PG_DSN — xem deploy/docker-compose.yml")
	}
	ctx := context.Background()
	if err := Migrate(ctx, admin); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	pool, err := pgxpool.New(ctx, admin)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)
	n := time.Now().UnixNano()
	a, b := fmt.Sprintf("don-vuot-a-%d", n), fmt.Sprintf("don-vuot-b-%d", n)
	t.Cleanup(func() { _, _ = pool.Exec(ctx, "DELETE FROM document_vectors WHERE tenant_id IN ($1, $2)", a, b) })
	for _, tn := range []string{a, b} {
		if _, err := pool.Exec(ctx, "INSERT INTO document_vectors (tenant_id, space, text_hash, vec) VALUES ($1, 'cu/3', $2, '{1}')", tn, rag.EmbedKey(tn)); err != nil {
			t.Fatal(err)
		}
	}
	su := &Store{pool: pool}
	if removed, err := su.PruneVectorSpaces(ctx, contracts.TenantID(a), "moi/3"); err != nil || removed != 1 {
		t.Fatalf("dọn tenant A: xoá %d, %v (muốn 1)", removed, err)
	}
	var c int
	if err := pool.QueryRow(ctx, "SELECT count(*) FROM document_vectors WHERE tenant_id = $1", b).Scan(&c); err != nil || c != 1 {
		t.Errorf("dọn tenant A bằng vai vượt RLS đã xoá vector của tenant B: còn %d, %v", c, err)
	}
}
