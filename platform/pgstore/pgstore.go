// Package pgstore là tầng lưu trữ PostgreSQL của dữ liệu nghiệp vụ (lát W4): fact, danh mục thực
// thể, tài liệu tri thức. Cách ly tenant bằng RLS: mọi lệnh chạy trong MỘT giao dịch đặt
// SET LOCAL eais.tenant_id từ tenant của Principal (contracts.TenantFrom, do S1 gắn) — không có
// tenant thì từ chối, không truy vấn. Ứng dụng kết nối bằng vai eais_app, vai không bỏ qua được RLS.
package pgstore

import (
	"context"
	"maps"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/luannt2002/enterprise-ai-control-layer/core/contracts"
	"github.com/luannt2002/enterprise-ai-control-layer/data/entityresolution"
	"github.com/luannt2002/enterprise-ai-control-layer/data/sourceoftruth"
	"github.com/luannt2002/enterprise-ai-control-layer/intelligence/rag"
)

// Store đọc / ghi dữ liệu nghiệp vụ của MỌI tenant qua một pool; tenant của từng lệnh lấy từ
// ngữ cảnh, RLS chặn phần còn lại.
type Store struct {
	pool *pgxpool.Pool
	mu   sync.Mutex
	// know giữ chỉ mục tri thức đã dựng của từng tenant cùng dấu nạp của nó (xem knowledgeIndex).
	know map[contracts.TenantID]knowledgeEntry
	// Semantic, nếu khác nil, bật truy hồi lai cho chỉ mục tri thức: vector của đoạn đọc từ document_vectors (migration 0008) theo không
	// gian Semantic.Embedder.Name(), đồng bộ bằng EmbedKnowledge. nil = chỉ từ khoá (cmd/eais). Đặt trước khi phục vụ, không đổi sau đó.
	Semantic *rag.Semantic
}

type knowledgeEntry struct {
	mark string
	ix   *rag.MemoryIndex
}

// Open mở pool tới dsn (vai eais_app) và kiểm kết nối. Vai vượt được RLS (superuser hoặc BYPASSRLS) bị từ chối: với vai
// đó RLS không chặn gì, và Load (DELETE FROM dựa vào RLS) xoá dữ liệu của MỌI tenant.
func Open(ctx context.Context, dsn string) (*Store, error) {
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		return nil, contracts.Label(contracts.Infra("postgres"), err)
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, contracts.Label(contracts.Infra("postgres"), err)
	}
	var role string
	var super, bypass bool
	if err := pool.QueryRow(ctx, "SELECT rolname, rolsuper, rolbypassrls FROM pg_catalog.pg_roles WHERE rolname = current_user").Scan(&role, &super, &bypass); err != nil {
		pool.Close()
		return nil, contracts.Label(contracts.Infra("postgres"), err)
	}
	if super || bypass {
		pool.Close()
		return nil, contracts.Errorf(contracts.FaultPolicy, "vai "+role+" vượt RLS (superuser hoặc BYPASSRLS): Store chỉ chạy bằng vai mà RLS chặn được (vd eais_app)")
	}
	return &Store{pool: pool}, nil
}

// Close đóng pool.
func (s *Store) Close() { s.pool.Close() }

func tenantOf(ctx context.Context) (contracts.TenantID, error) {
	t, ok := contracts.TenantFrom(ctx)
	if !ok {
		return "", contracts.Errorf(contracts.FaultPolicy, "truy vấn không mang tenant trong ngữ cảnh")
	}
	return t, nil
}

// inTenant chạy fn trong một giao dịch đã đặt GUC eais.tenant_id (SET LOCAL: hết giao dịch là hết).
func (s *Store) inTenant(ctx context.Context, tenant contracts.TenantID, fn func(pgx.Tx) error) error {
	err := pgx.BeginFunc(ctx, s.pool, func(tx pgx.Tx) error {
		if _, err := tx.Exec(ctx, "SELECT set_config('eais.tenant_id', $1, true)", string(tenant)); err != nil {
			return err
		}
		return fn(tx)
	})
	return classifyPGError(err)
}

// Setting trả giá trị GUC eais.tenant_id mà một lệnh của Store thấy — để test đọc lại.
func (s *Store) Setting(ctx context.Context) (string, error) {
	tenant, err := tenantOf(ctx)
	if err != nil {
		return "", err
	}
	var v string
	err = s.inTenant(ctx, tenant, func(tx pgx.Tx) error {
		return tx.QueryRow(ctx, "SELECT current_setting('eais.tenant_id', true)").Scan(&v)
	})
	return v, err
}

// Facts hiện thực sourceoftruth.Store: lọc theo chỉ số, thực thể (rỗng = không lọc) và ngày (gồm
// hai đầu), trả đúng thứ tự nạp như bản trong bộ nhớ.
func (s *Store) Facts(ctx context.Context, metric string, entityIDs []string, from, to time.Time) ([]sourceoftruth.Fact, error) {
	tenant, err := tenantOf(ctx)
	if err != nil {
		return nil, err
	}
	var out []sourceoftruth.Fact
	err = s.inTenant(ctx, tenant, func(tx pgx.Tx) error {
		rows, err := tx.Query(ctx, `
			SELECT id, entity_id, occurred_on, value::text, unit, approx, source_type, source_ref, raw, observed_at, reported_by, coalesce(note, '')
			FROM facts
			WHERE metric = $1 AND (coalesce(cardinality($2::text[]), 0) = 0 OR entity_id = ANY($2))
			  AND occurred_on BETWEEN $3::date AND $4::date
			ORDER BY ord`, metric, entityIDs, from.Format(time.DateOnly), to.Format(time.DateOnly))
		if err != nil {
			return err
		}
		defer rows.Close()
		for rows.Next() {
			f := sourceoftruth.Fact{Metric: metric, Value: contracts.Value{Kind: contracts.ValueQuantity}}
			var on time.Time
			var observed *time.Time
			if err := rows.Scan(&f.ID, &f.EntityID, &on, &f.Value.Number, &f.Value.Unit, &f.Value.Approx,
				&f.Source.SourceType, &f.Source.SourceRef, &f.Source.Raw, &observed, &f.Source.ReportedBy, &f.Source.Note); err != nil {
				return err
			}
			// ngày của fact là 00:00 theo múi giờ của tenant — cùng múi với khoảng hỏi
			f.OccurredOn = time.Date(on.Year(), on.Month(), on.Day(), 0, 0, 0, 0, from.Location())
			f.Source.ID = f.ID
			if observed != nil {
				f.Source.ObservedAt = *observed
			}
			out = append(out, f)
		}
		return rows.Err()
	})
	return out, err
}

// Entities hiện thực entityresolution.Directory.
func (s *Store) Entities(ctx context.Context, types ...string) ([]entityresolution.Entity, error) {
	tenant, err := tenantOf(ctx)
	if err != nil {
		return nil, err
	}
	var out []entityresolution.Entity
	err = s.inTenant(ctx, tenant, func(tx pgx.Tx) error {
		rows, err := tx.Query(ctx, `
			SELECT e.id, e.type, e.key, e.name, coalesce(e.parent_id, ''),
			       coalesce(array_agg(a.alias ORDER BY a.ord) FILTER (WHERE a.alias IS NOT NULL), '{}'),
			       coalesce(array_agg(a.source ORDER BY a.ord) FILTER (WHERE a.alias IS NOT NULL), '{}')
			FROM entities e
			LEFT JOIN entity_aliases a ON a.tenant_id = e.tenant_id AND a.entity_id = e.id
			WHERE coalesce(cardinality($1::text[]), 0) = 0 OR e.type = ANY($1) -- slice nil tới DB là NULL
			GROUP BY e.tenant_id, e.id
			ORDER BY min(e.ord)`, types)
		if err != nil {
			return err
		}
		defer rows.Close()
		for rows.Next() {
			var e entityresolution.Entity
			var aliases, sources []string
			if err := rows.Scan(&e.ID, &e.Type, &e.Key, &e.Name, &e.Parent, &aliases, &sources); err != nil {
				return err
			}
			for i, a := range aliases {
				e.Aliases = append(e.Aliases, entityresolution.Alias{Text: a, Source: sources[i]})
			}
			out = append(out, e)
		}
		return rows.Err()
	})
	return out, err
}

// Retrieve hiện thực pipeline.KnowledgeRetriever trên chỉ mục giữ sẵn của tenant (RLS: chỉ tài liệu của tenant). Quyền đọc từng tài
// liệu (vai trong ACL, phạm vi) do chỉ mục xét cho từng lượt — cùng mã với đường bộ nhớ.
func (s *Store) Retrieve(ctx context.Context, pr contracts.Principal, query string, k int) ([]contracts.Evidence, error) {
	if t, ok := contracts.TenantFrom(ctx); ok && t != pr.TenantID {
		return nil, contracts.Errorf(contracts.FaultPolicy, "principal không khớp tenant của ngữ cảnh")
	}
	if pr.TenantID == "" {
		return nil, contracts.Errorf(contracts.FaultPolicy, "principal không mang tenant")
	}
	ix, err := s.knowledgeIndex(ctx, pr.TenantID)
	if err != nil {
		return nil, err
	}
	return ix.Retrieve(ctx, pr, query, k)
}

// knowledgeIndex trả chỉ mục tri thức của tenant, dựng MỘT lần cho mỗi lần nạp thay vì mỗi lượt hỏi (dựng mỗi lượt: 437 ms và
// 144 MB cấp phát mỗi câu ở ~1.000 tài liệu, 2,1 s / 721 MB ở ~5.000 — evals/rag/quy_mo_test.go). Dấu nạp = các xmin của dòng
// documents + số dòng: Load xoá rồi ghi lại cả tenant trong một giao dịch nên lần nạp sau — kể cả từ tiến trình khác — đổi dấu; dấu
// đọc ở mỗi lượt (một câu tổng hợp), nên chỉ mục không bao giờ cũ hơn lần nạp đã commit.
// shortcut: lượt đầu sau mỗi lần nạp có thể dựng trùng khi nhiều lượt cùng tới; giữ một mục mỗi tenant, không giới hạn số tenant —
// đủ khi mỗi tiến trình phục vụ một tenant (cmd/eais); thêm khoá theo tenant / giới hạn khi một tiến trình phục vụ nhiều tenant.
func (s *Store) knowledgeIndex(ctx context.Context, tenant contracts.TenantID) (*rag.MemoryIndex, error) {
	var mark string
	if err := s.inTenant(ctx, tenant, func(tx pgx.Tx) error {
		if err := tx.QueryRow(ctx, `SELECT coalesce(string_agg(DISTINCT xmin::text COLLATE "C", ',' ORDER BY xmin::text COLLATE "C"), '') || '/' || count(*)
			FROM documents`).Scan(&mark); err != nil || s.Semantic == nil {
			return err
		}
		// Truy hồi lai: dấu gồm cả các dòng vector của không gian hiện tại — nhúng lại xong thì lượt sau dựng lại chỉ mục.
		var vm string
		err := tx.QueryRow(ctx, `SELECT $1 || ':' || coalesce(string_agg(DISTINCT xmin::text COLLATE "C", ',' ORDER BY xmin::text COLLATE "C"), '') || '/' || count(*)
			FROM document_vectors WHERE space = $1`, s.Semantic.Embedder.Name()).Scan(&vm)
		mark += "|" + vm
		return err
	}); err != nil {
		return nil, err
	}
	s.mu.Lock()
	e, ok := s.know[tenant]
	s.mu.Unlock()
	if ok && e.mark == mark {
		return e.ix, nil
	}
	var docs []rag.Document
	var vecs map[string][]float32
	var vxmins []string
	xmins := map[string]bool{}
	err := s.inTenant(ctx, tenant, func(tx pgx.Tx) error {
		var err error
		if docs, err = readDocuments(ctx, tx, tenant, xmins); err != nil || s.Semantic == nil {
			return err
		}
		vecs, vxmins, err = loadVectors(ctx, tx, s.Semantic.Embedder.Name())
		return err
	})
	if err != nil {
		return nil, err
	}
	// Dấu tính từ chính các dòng vừa đọc (cùng giao dịch): lần nạp xen giữa hai câu chỉ làm lượt sau dựng lại, không làm giữ dữ liệu cũ.
	mark = strings.Join(slices.Sorted(maps.Keys(xmins)), ",") + "/" + strconv.Itoa(len(docs))
	ix := rag.NewMemoryIndex(string(tenant), docs)
	if s.Semantic != nil {
		slices.Sort(vxmins)
		mark += "|" + s.Semantic.Embedder.Name() + ":" + strings.Join(slices.Compact(vxmins), ",") + "/" + strconv.Itoa(len(vecs))
		sem := *s.Semantic
		sem.Vectors = vecs
		ix.Semantic = &sem
	}
	s.mu.Lock()
	if s.know == nil {
		s.know = map[contracts.TenantID]knowledgeEntry{}
	}
	s.know[tenant] = knowledgeEntry{mark: mark, ix: ix}
	s.mu.Unlock()
	return ix, nil
}

// readDocuments đọc mọi tài liệu của tenant trong giao dịch tx, ghi xmin của từng dòng vào xmins (nil = không cần).
func readDocuments(ctx context.Context, tx pgx.Tx, tenant contracts.TenantID, xmins map[string]bool) ([]rag.Document, error) {
	rows, err := tx.Query(ctx, `
		SELECT id, title, body, acl, version, status, effective_from, effective_to, scope, hold, xmin::text FROM documents
		ORDER BY id COLLATE "C", version`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var docs []rag.Document
	for rows.Next() {
		d := rag.Document{Tenant: string(tenant)}
		var xmin string
		if err := rows.Scan(&d.ID, &d.Title, &d.Body, &d.ACL, &d.Version, &d.Status, &d.EffectiveFrom, &d.EffectiveTo, &d.Scope, &d.Hold, &xmin); err != nil {
			return nil, err
		}
		if xmins != nil {
			xmins[xmin] = true
		}
		docs = append(docs, d)
	}
	return docs, rows.Err()
}
