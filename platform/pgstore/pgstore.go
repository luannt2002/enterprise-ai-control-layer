// Package pgstore là tầng lưu trữ PostgreSQL của dữ liệu nghiệp vụ (lát W4): fact, danh mục thực
// thể, tài liệu tri thức. Cách ly tenant bằng RLS: mọi lệnh chạy trong MỘT giao dịch đặt
// SET LOCAL eacl.tenant_id từ tenant của Principal (contracts.TenantFrom, do S1 gắn) — không có
// tenant thì từ chối, không truy vấn. Ứng dụng kết nối bằng vai eacl_app, vai không bỏ qua được RLS.
package pgstore

import (
	"context"
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
type Store struct{ pool *pgxpool.Pool }

// Open mở pool tới dsn (vai eacl_app) và kiểm kết nối.
func Open(ctx context.Context, dsn string) (*Store, error) {
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		return nil, contracts.Label(contracts.Infra("postgres"), err)
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, contracts.Label(contracts.Infra("postgres"), err)
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

// inTenant chạy fn trong một giao dịch đã đặt GUC eacl.tenant_id (SET LOCAL: hết giao dịch là hết).
func (s *Store) inTenant(ctx context.Context, tenant contracts.TenantID, fn func(pgx.Tx) error) error {
	err := pgx.BeginFunc(ctx, s.pool, func(tx pgx.Tx) error {
		if _, err := tx.Exec(ctx, "SELECT set_config('eacl.tenant_id', $1, true)", string(tenant)); err != nil {
			return err
		}
		return fn(tx)
	})
	if err != nil {
		return contracts.Label(contracts.Infra("postgres"), err)
	}
	return nil
}

// Setting trả giá trị GUC eacl.tenant_id mà một lệnh của Store thấy — để test đọc lại.
func (s *Store) Setting(ctx context.Context) (string, error) {
	tenant, err := tenantOf(ctx)
	if err != nil {
		return "", err
	}
	var v string
	err = s.inTenant(ctx, tenant, func(tx pgx.Tx) error {
		return tx.QueryRow(ctx, "SELECT current_setting('eacl.tenant_id', true)").Scan(&v)
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
			SELECT id, entity_id, occurred_on, value::text, unit, approx, source_type, source_ref, raw, observed_at, reported_by
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
				&f.Source.SourceType, &f.Source.SourceRef, &f.Source.Raw, &observed, &f.Source.ReportedBy); err != nil {
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

// Retrieve hiện thực pipeline.KnowledgeRetriever: chỉ tài liệu của tenant (RLS) mà vai của người
// hỏi có trong acl (lọc ngay trong SQL) mới ra khỏi DB; chấm điểm bằng chính bộ chấm của rag.
func (s *Store) Retrieve(ctx context.Context, pr contracts.Principal, query string, k int) ([]contracts.Evidence, error) {
	if t, ok := contracts.TenantFrom(ctx); ok && t != pr.TenantID {
		return nil, contracts.Errorf(contracts.FaultPolicy, "principal không khớp tenant của ngữ cảnh")
	}
	if pr.TenantID == "" {
		return nil, contracts.Errorf(contracts.FaultPolicy, "principal không mang tenant")
	}
	var docs []rag.Document
	err := s.inTenant(ctx, pr.TenantID, func(tx pgx.Tx) error {
		rows, err := tx.Query(ctx, `
			SELECT id, title, body, acl FROM documents
			WHERE acl && $1::text[]
			ORDER BY id COLLATE "C"`, pr.Roles)
		if err != nil {
			return err
		}
		defer rows.Close()
		for rows.Next() {
			d := rag.Document{Tenant: string(pr.TenantID)}
			if err := rows.Scan(&d.ID, &d.Title, &d.Body, &d.ACL); err != nil {
				return err
			}
			docs = append(docs, d)
		}
		return rows.Err()
	})
	if err != nil {
		return nil, err
	}
	return rag.NewMemoryIndex(string(pr.TenantID), docs).Retrieve(ctx, pr, query, k)
}
