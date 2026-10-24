package ragcases

import (
	"context"
	"fmt"
	"path/filepath"
	"testing"
	"time"

	"github.com/luannt2002/enterprise-ai-control-layer/core/contracts"
	"github.com/luannt2002/enterprise-ai-control-layer/intelligence/rag"
	"github.com/luannt2002/enterprise-ai-control-layer/platform/pgstore"
)

// Quy mô của truy xuất (KE-HOACH U6b chờ số này): đường PG đọc mọi tài liệu người hỏi được đọc rồi dựng chỉ mục MỚI mỗi lượt.
// Kho giả = kho S7 phát triển nhân n lần (mã tài liệu thêm hậu tố) — đo chi phí theo số đoạn, không đo chất lượng.
// Chạy: go test -run '^$' -bench 'TruyXuat' -benchtime 20x ./evals/rag/   (đường PG cần EAIS_PG_*).
func khoNhan(b *testing.B, n int) []rag.Document {
	b.Helper()
	goc, err := rag.LoadMarkdown(filepath.Join(s7Dir, "kho"), "abc-infra")
	if err != nil {
		b.Fatal(err)
	}
	var out []rag.Document
	for i := range n {
		for _, d := range goc {
			d.ID = fmt.Sprintf("%s-%05d", d.ID, i)
			out = append(out, d)
		}
	}
	return out
}

var hoiQuyMo = contracts.Principal{ID: "u", TenantID: "abc-infra", Roles: []string{"director"}}

const cauQuyMo = "lệch dầu giữa kho xuất với máy dùng thì mấy tiếng phải có biên bản?"

func BenchmarkTruyXuatBoNho(b *testing.B) {
	for _, n := range []int{6, 60, 300} { // × 17 tệp ≈ 100 / 1.000 / 5.100 tài liệu
		docs := khoNhan(b, n)
		b.Run(fmt.Sprintf("%dtai-lieu/dung-moi-luot", len(docs)), func(b *testing.B) {
			for range b.N {
				ix := rag.NewMemoryIndex("abc-infra", docs)
				ix.Now = func() time.Time { return time.Date(2026, 10, 20, 0, 0, 0, 0, time.UTC) }
				if _, err := ix.Retrieve(context.Background(), hoiQuyMo, cauQuyMo, 3); err != nil {
					b.Fatal(err)
				}
			}
		})
		ix := rag.NewMemoryIndex("abc-infra", docs)
		ix.Now = func() time.Time { return time.Date(2026, 10, 20, 0, 0, 0, 0, time.UTC) }
		b.Run(fmt.Sprintf("%dtai-lieu/dung-san", len(docs)), func(b *testing.B) {
			for range b.N {
				if _, err := ix.Retrieve(context.Background(), hoiQuyMo, cauQuyMo, 3); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

func BenchmarkTruyXuatPG(b *testing.B) {
	admin, app := pgstore.EnvDSN()
	if admin == "" || app == "" {
		b.Skip("chưa có EAIS_PG_ADMIN_DSN / EAIS_PG_DSN")
	}
	ctx := context.Background()
	if err := pgstore.Migrate(ctx, admin); err != nil {
		b.Fatal(err)
	}
	s, err := pgstore.Open(ctx, app)
	if err != nil {
		b.Fatal(err)
	}
	defer s.Close()
	for _, n := range []int{6, 60, 300} {
		docs := khoNhan(b, n)
		tid := contracts.TenantID(fmt.Sprintf("ragpg-quymo-%d-%d", n, time.Now().UnixNano()))
		tctx := contracts.WithTenant(ctx, tid)
		if err := s.Load(tctx, nil, nil, docs); err != nil {
			b.Fatal(err)
		}
		pr := hoiQuyMo
		pr.TenantID = tid
		q := contracts.WithKnowledgeQuery(tctx, contracts.KnowledgeQuery{AsOf: time.Date(2026, 10, 20, 0, 0, 0, 0, time.UTC)})
		b.Run(fmt.Sprintf("%dtai-lieu", len(docs)), func(b *testing.B) {
			for range b.N {
				if _, err := s.Retrieve(q, pr, cauQuyMo, 3); err != nil {
					b.Fatal(err)
				}
			}
		})
		_ = s.Load(tctx, nil, nil, nil)
	}
}
