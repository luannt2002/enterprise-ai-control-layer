package pgstore

import (
	"context"
	"hash/fnv"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/luannt2002/enterprise-ai-control-layer/core/contracts"
	"github.com/luannt2002/enterprise-ai-control-layer/data/entityresolution"
	"github.com/luannt2002/enterprise-ai-control-layer/data/sourceoftruth"
	"github.com/luannt2002/enterprise-ai-control-layer/intelligence/rag"
)

// loadBatch là số dòng mỗi lệnh INSERT … unnest (COPY không chạy trên bảng có RLS).
const loadBatch = 5000

// loadLockKey là khoá tư vấn (advisory lock) mà Load giữ theo tenant: băm FNV-64a của "eais.load:<tenant>". Băm ở phía Go để khoá tất
// định và đọc được, không dựa vào hàm băm không tài liệu của PG.
func loadLockKey(tenant contracts.TenantID) int64 {
	h := fnv.New64a()
	h.Write([]byte("eais.load:" + string(tenant)))
	return int64(h.Sum64())
}

// Load THAY toàn bộ dữ liệu của tenant trong ngữ cảnh (danh mục, fact, tài liệu) trong một giao
// dịch: xoá cái cũ rồi ghi cái mới — RLS giới hạn cả lệnh xoá trong đúng tenant đó. Cột tenant_id
// lấy từ GUC của giao dịch, và RLS (WITH CHECK) chặn mọi dòng mang tenant khác — tenant không đến
// từ dữ liệu được nạp. Thứ tự nạp được giữ (cột ord) để đọc ra đúng như bản trong bộ nhớ.
//
// Các Load CÙNG tenant xếp hàng bằng khoá tư vấn mức giao dịch (pg_advisory_xact_lock, nhả khi giao dịch kết thúc): "xoá rồi ghi lại"
// ở READ COMMITTED không tự loại trừ nhau — hai giao dịch cùng xoá cái cũ rồi cùng ghi cái mới, giao dịch sau đụng khoá chính và bị
// huỷ nguyên khối. Khoá này chặn cả hai tiến trình khác nhau (server đang nạp, `eais ingest` chạy cùng lúc), điều mà khoá trong bộ nhớ
// của server không làm được. Tenant khác nhau không chờ nhau.
func (s *Store) Load(ctx context.Context, entities []entityresolution.Entity, facts []sourceoftruth.Fact, docs []rag.Document) error {
	tenant, err := tenantOf(ctx)
	if err != nil {
		return err
	}
	return s.inTenant(ctx, tenant, func(tx pgx.Tx) error {
		if _, err := tx.Exec(ctx, "SELECT pg_advisory_xact_lock($1)", loadLockKey(tenant)); err != nil {
			return err
		}
		for _, table := range []string{"entity_aliases", "facts", "documents", "entities"} {
			if _, err := tx.Exec(ctx, "DELETE FROM "+table); err != nil { // RLS: chỉ dòng của tenant này
				return err
			}
		}
		if err := loadEntities(ctx, tx, entities); err != nil {
			return err
		}
		for i := 0; i < len(facts); i += loadBatch {
			if err := loadFacts(ctx, tx, facts[i:min(i+loadBatch, len(facts))], i); err != nil {
				return err
			}
		}
		for _, d := range docs {
			if _, err := tx.Exec(ctx, `INSERT INTO documents (tenant_id, id, title, body, acl)
				VALUES (current_setting('eais.tenant_id'), $1, $2, $3, $4)`, d.ID, d.Title, d.Body, d.ACL); err != nil {
				return err
			}
		}
		return nil
	})
}

func loadEntities(ctx context.Context, tx pgx.Tx, es []entityresolution.Entity) error {
	var ids, types, keys, names []string
	var parents []pgtype.Text
	var ords []int32
	var aEntity, aText, aSource []string
	var aOrd []int32
	for i, e := range es {
		ids, types, keys, names = append(ids, e.ID), append(types, e.Type), append(keys, e.Key), append(names, e.Name)
		parents = append(parents, pgtype.Text{String: e.Parent, Valid: e.Parent != ""})
		ords = append(ords, int32(i))
		for _, a := range e.Aliases {
			aEntity, aText, aSource = append(aEntity, e.ID), append(aText, a.Text), append(aSource, a.Source)
			aOrd = append(aOrd, int32(len(aOrd)))
		}
	}
	if _, err := tx.Exec(ctx, `INSERT INTO entities (tenant_id, id, type, key, name, parent_id, ord)
		SELECT current_setting('eais.tenant_id'), * FROM unnest($1::text[], $2::text[], $3::text[], $4::text[], $5::text[], $6::int[])`,
		ids, types, keys, names, parents, ords); err != nil {
		return err
	}
	_, err := tx.Exec(ctx, `INSERT INTO entity_aliases (tenant_id, entity_id, alias, source, ord)
		SELECT current_setting('eais.tenant_id'), * FROM unnest($1::text[], $2::text[], $3::text[], $4::int[])`,
		aEntity, aText, aSource, aOrd)
	return err
}

func loadFacts(ctx context.Context, tx pgx.Tx, fs []sourceoftruth.Fact, offset int) error {
	n := len(fs)
	ids, metrics, ents, days, values, units := make([]string, n), make([]string, n), make([]string, n), make([]string, n), make([]string, n), make([]string, n)
	approx := make([]bool, n)
	stype, sref, raw, by := make([]string, n), make([]string, n), make([]string, n), make([]string, n)
	observed := make([]pgtype.Timestamptz, n)
	ords := make([]int64, n)
	for i, f := range fs {
		ids[i], metrics[i], ents[i], days[i] = f.ID, f.Metric, f.EntityID, f.OccurredOn.Format(time.DateOnly)
		values[i], units[i], approx[i] = f.Value.Number, f.Value.Unit, f.Value.Approx
		stype[i], sref[i], raw[i], by[i] = f.Source.SourceType, f.Source.SourceRef, f.Source.Raw, f.Source.ReportedBy
		observed[i] = pgtype.Timestamptz{Time: f.Source.ObservedAt, Valid: !f.Source.ObservedAt.IsZero()}
		ords[i] = int64(offset + i)
	}
	_, err := tx.Exec(ctx, `INSERT INTO facts (tenant_id, id, metric, entity_id, occurred_on, value, unit, approx,
			source_type, source_ref, raw, observed_at, reported_by, ord)
		SELECT current_setting('eais.tenant_id'), * FROM unnest($1::text[], $2::text[], $3::text[], $4::text[]::date[],
			$5::text[]::numeric[], $6::text[], $7::bool[], $8::text[], $9::text[], $10::text[], $11::timestamptz[], $12::text[], $13::bigint[])`,
		ids, metrics, ents, days, values, units, approx, stype, sref, raw, observed, by, ords)
	return err
}
