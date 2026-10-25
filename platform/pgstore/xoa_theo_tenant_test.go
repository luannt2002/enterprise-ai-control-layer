package pgstore

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/luannt2002/enterprise-ai-control-layer/core/contracts"
	"github.com/luannt2002/enterprise-ai-control-layer/data/entityresolution"
	"github.com/luannt2002/enterprise-ai-control-layer/data/sourceoftruth"
	"github.com/luannt2002/enterprise-ai-control-layer/intelligence/rag"
)

// Phòng thủ chiều sâu (F-037): lệnh xoá của Load và LoadDocuments lọc theo tenant ngay trong câu lệnh, không chỉ dựa vào RLS. Store
// dựng thẳng trên pool của vai quản trị (vượt RLS — Open từ chối vai này, đúng thứ mà lớp thứ hai phải chịu được khi lớp thứ nhất
// hỏng): nạp lại tenant A không được xoá dòng nào của tenant B.
func TestXoaChiDongCuaTenantKhiVuotRLS(t *testing.T) {
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
	su := &Store{pool: pool}
	n := time.Now().UnixNano()
	a, b := contracts.TenantID(fmt.Sprintf("xoa-a-%d", n)), contracts.TenantID(fmt.Sprintf("xoa-b-%d", n))
	actx, bctx := contracts.WithTenant(ctx, a), contracts.WithTenant(ctx, b)
	es := []entityresolution.Entity{{ID: "E-1", Type: "loai", Key: "E-1", Name: "Thực thể 1", Aliases: []entityresolution.Alias{{Text: "số 1", Source: "manual"}}}}
	fs := []sourceoftruth.Fact{{ID: "R-1", Metric: "chi_so", EntityID: "E-1", OccurredOn: time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC),
		Value: contracts.Value{Kind: contracts.ValueQuantity, Number: "7", Unit: "đv"}, Source: sourceoftruth.Observation{ID: "R-1", SourceType: "phieu", SourceRef: "R-1"}}}
	ds := []rag.Document{{ID: "QT", Title: "Quy trình", ACL: []string{"r"}, Body: "1. Bước một."}}
	t.Cleanup(func() {
		for _, c := range []context.Context{actx, bctx} {
			_ = su.Load(c, nil, nil, nil)
		}
	})
	if err := su.Load(bctx, es, fs, ds); err != nil {
		t.Fatal(err)
	}
	dem := func() map[string]int {
		out := map[string]int{}
		for _, tb := range []string{"entities", "entity_aliases", "facts", "documents"} {
			var c int
			if err := pool.QueryRow(ctx, "SELECT count(*) FROM "+tb+" WHERE tenant_id = $1", string(b)).Scan(&c); err != nil {
				t.Fatal(err)
			}
			out[tb] = c
		}
		return out
	}
	truoc := dem()
	for tb, c := range truoc {
		if c == 0 {
			t.Fatalf("tiền đề: tenant B phải có dòng ở %s", tb)
		}
	}
	if err := su.Load(actx, nil, nil, ds); err != nil {
		t.Fatal(err)
	}
	if err := su.LoadDocuments(actx, LoadInfo{}, nil); err != nil {
		t.Fatal(err)
	}
	if sau := dem(); fmt.Sprint(sau) != fmt.Sprint(truoc) {
		t.Errorf("nạp tenant A bằng vai vượt RLS đã xoá dòng của tenant B: trước %v, sau %v", truoc, sau)
	}
}

// Như trên cho lệnh xoá vector của EmbedKnowledge (soát độc lập cuối, S4): nhúng tenant A bằng vai vượt RLS không được xoá vector của
// tenant B — kể cả vector cũ mà B chưa dọn (B nạp tài liệu mới, chưa nhúng lại: việc dọn là của lần nhúng của chính B). Không gian vector
// riêng cho mỗi lần chạy, để bản lỗi (xoá theo không gian, mọi tenant) không chạm dữ liệu của test khác.
func TestNhungChiXoaVectorCuaTenantKhiVuotRLS(t *testing.T) {
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
	su := &Store{pool: pool, Semantic: &rag.Semantic{Embedder: &nhungDem{ten: fmt.Sprintf("xoa-vec-%d/3", n)}}}
	a, b := contracts.TenantID(fmt.Sprintf("xoa-vec-a-%d", n)), contracts.TenantID(fmt.Sprintf("xoa-vec-b-%d", n))
	actx, bctx := contracts.WithTenant(ctx, a), contracts.WithTenant(ctx, b)
	t.Cleanup(func() { // xoá tài liệu của CẢ hai trước khi nhúng: vai vượt RLS đọc tài liệu của mọi tenant
		_ = su.Load(actx, nil, nil, nil)
		_ = su.Load(bctx, nil, nil, nil)
		_, _, _ = su.EmbedKnowledge(ctx, a)
		_, _, _ = su.EmbedKnowledge(ctx, b)
	})
	tl := func(body string) []rag.Document {
		return []rag.Document{{ID: "QT", Title: "Quy trình", ACL: []string{"r"}, Body: body}}
	}
	if err := su.Load(bctx, nil, nil, tl("1. Bước một.\n2. Bước hai.")); err != nil {
		t.Fatal(err)
	}
	if _, _, err := su.EmbedKnowledge(ctx, b); err != nil {
		t.Fatal(err)
	}
	if err := su.Load(bctx, nil, nil, tl("1. Bước ba.")); err != nil {
		t.Fatal(err)
	}
	dem := func() int {
		var c int
		if err := pool.QueryRow(ctx, "SELECT count(*) FROM document_vectors WHERE tenant_id = $1", string(b)).Scan(&c); err != nil {
			t.Fatal(err)
		}
		return c
	}
	truoc := dem()
	if truoc == 0 {
		t.Fatal("tiền đề: tenant B phải có vector")
	}
	if _, _, err := su.EmbedKnowledge(ctx, a); err != nil {
		t.Fatal(err)
	}
	if sau := dem(); sau != truoc {
		t.Errorf("nhúng tenant A bằng vai vượt RLS đã xoá vector của tenant B: trước %d, sau %d", truoc, sau)
	}
}
