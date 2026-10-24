package pgstore

import (
	"context"
	"errors"
	"fmt"
	"hash/fnv"
	"regexp"
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

// LoadInfo là danh tính của một lần nạp, ghi vào sổ nạp (load_records, migration 0006) CÙNG giao dịch thay dữ liệu.
type LoadInfo struct {
	LoadID      string // mã lần nạp (cùng mã với bản ghi cách ly của lần nạp đó)
	DataVersion string // băm 64 hex của dữ liệu nguồn đã nạp
}

// LoadRecord là một dòng của sổ nạp.
type LoadRecord struct {
	Seq         int64     // thứ tự nạp của tenant (id của dòng): tăng theo thứ tự commit
	LoadID      string    // rỗng khi lần nạp không kèm phiên bản
	DataVersion string    // rỗng khi lần nạp không kèm phiên bản
	LoadedAt    time.Time // giờ của cơ sở dữ liệu lúc nạp
}

var dataVersionFormat = regexp.MustCompile(`^[0-9a-f]{64}$`)

// Load THAY toàn bộ dữ liệu của tenant trong ngữ cảnh (danh mục, fact, tài liệu) mà KHÔNG kèm phiên bản: sổ nạp nhận một dòng "không ghi
// nhận" (LoadRecord.DataVersion rỗng), vì dữ liệu đã đổi và dòng cũ không còn mô tả nó. Đường có phiên bản là LoadWithVersion; Load giữ lại
// cho bộ chạy winrun và test, những nơi tự giữ băm manifest riêng.
func (s *Store) Load(ctx context.Context, entities []entityresolution.Entity, facts []sourceoftruth.Fact, docs []rag.Document) error {
	return s.load(ctx, LoadInfo{}, false, entities, facts, docs)
}

// LoadWithVersion như Load nhưng ghi phiên bản của lần nạp vào sổ nạp TRONG CÙNG giao dịch: dữ liệu mới và phiên bản của nó cùng commit hoặc
// cùng huỷ (lần nạp hỏng giữa chừng không đổi dữ liệu, cũng không đổi phiên bản). Phiên bản sai dạng bị từ chối trước khi chạm DB.
func (s *Store) LoadWithVersion(ctx context.Context, info LoadInfo, entities []entityresolution.Entity, facts []sourceoftruth.Fact, docs []rag.Document) error {
	if err := checkLoadInfo(info); err != nil {
		return err
	}
	return s.load(ctx, info, false, entities, facts, docs)
}

// LoadDocuments THAY CHỈ tài liệu tri thức của tenant (bảng documents); danh mục và fact giữ nguyên (F-019): module RAG nạp / công bố
// tài liệu mà không đụng dữ liệu nghiệp vụ — Load(ctx, nil, nil, docs) thì xoá sạch fact. Cùng giao dịch, cùng khoá tư vấn và cùng sổ nạp
// như Load: info rỗng ghi dòng "không ghi nhận" (dữ liệu đã đổi, phiên bản cũ không còn mô tả nó), info khác rỗng phải đúng dạng như
// LoadWithVersion.
func (s *Store) LoadDocuments(ctx context.Context, info LoadInfo, docs []rag.Document) error {
	if info != (LoadInfo{}) {
		if err := checkLoadInfo(info); err != nil {
			return err
		}
	}
	return s.load(ctx, info, true, nil, nil, docs)
}

// checkLoadInfo: phiên bản sai dạng bị từ chối trước khi chạm DB.
func checkLoadInfo(info LoadInfo) error {
	if info.LoadID == "" || !dataVersionFormat.MatchString(info.DataVersion) {
		return contracts.Errorf(contracts.FaultInput, fmt.Sprintf("lần nạp cần mã lần nạp và phiên bản dữ liệu dạng 64 hex, được mã %q, phiên bản %q", info.LoadID, info.DataVersion))
	}
	return nil
}

// LatestLoad trả dòng mới nhất của sổ nạp cho tenant trong ngữ cảnh: lần nạp đã COMMIT gần nhất, đọc từ PG (RLS: chỉ tenant đó). Tenant chưa
// có lần nạp nào được ghi nhận thì lỗi bọc ErrNotFound (nhãn input) — đó là "chưa nạp", không phải lỗi hạ tầng.
func (s *Store) LatestLoad(ctx context.Context) (LoadRecord, error) {
	tenant, err := tenantOf(ctx)
	if err != nil {
		return LoadRecord{}, err
	}
	var r LoadRecord
	err = s.inTenant(ctx, tenant, func(tx pgx.Tx) error {
		err := tx.QueryRow(ctx, `SELECT id, load_id, data_version, loaded_at FROM load_records ORDER BY id DESC LIMIT 1`).
			Scan(&r.Seq, &r.LoadID, &r.DataVersion, &r.LoadedAt)
		if errors.Is(err, pgx.ErrNoRows) {
			return contracts.Label(contracts.FaultInput, fmt.Errorf("%w: tenant chưa có lần nạp nào được ghi nhận", ErrNotFound))
		}
		return err
	})
	return r, err
}

// load THAY toàn bộ dữ liệu của tenant trong ngữ cảnh (danh mục, fact, tài liệu) trong một giao
// dịch: xoá cái cũ rồi ghi cái mới — RLS giới hạn cả lệnh xoá trong đúng tenant đó. Cột tenant_id
// lấy từ GUC của giao dịch, và RLS (WITH CHECK) chặn mọi dòng mang tenant khác — tenant không đến
// từ dữ liệu được nạp. Thứ tự nạp được giữ (cột ord) để đọc ra đúng như bản trong bộ nhớ. Dòng sổ nạp (info) là câu lệnh CUỐI của giao
// dịch, sau khoá tư vấn của tenant, nên id của nó tăng đúng theo thứ tự commit của các lần nạp cùng tenant.
//
// Các Load CÙNG tenant xếp hàng bằng khoá tư vấn mức giao dịch (pg_advisory_xact_lock, nhả khi giao dịch kết thúc): "xoá rồi ghi lại"
// ở READ COMMITTED không tự loại trừ nhau — hai giao dịch cùng xoá cái cũ rồi cùng ghi cái mới, giao dịch sau đụng khoá chính và bị
// huỷ nguyên khối. Khoá này chặn cả hai tiến trình khác nhau (server đang nạp, `eais ingest` chạy cùng lúc), điều mà khoá trong bộ nhớ
// của server không làm được. Tenant khác nhau không chờ nhau.
func (s *Store) load(ctx context.Context, info LoadInfo, docsOnly bool, entities []entityresolution.Entity, facts []sourceoftruth.Fact, docs []rag.Document) error {
	tenant, err := tenantOf(ctx)
	if err != nil {
		return err
	}
	tables := []string{"entity_aliases", "facts", "documents", "entities"}
	if docsOnly {
		tables = []string{"documents"}
	}
	return s.inTenant(ctx, tenant, func(tx pgx.Tx) error {
		if _, err := tx.Exec(ctx, "SELECT pg_advisory_xact_lock($1)", loadLockKey(tenant)); err != nil {
			return err
		}
		for _, table := range tables {
			if _, err := tx.Exec(ctx, "DELETE FROM "+table); err != nil { // RLS: chỉ dòng của tenant này
				return err
			}
		}
		if !docsOnly {
			if err := loadEntities(ctx, tx, entities); err != nil {
				return err
			}
			for i := 0; i < len(facts); i += loadBatch {
				if err := loadFacts(ctx, tx, facts[i:min(i+loadBatch, len(facts))], i); err != nil {
					return err
				}
			}
		}
		for _, d := range docs {
			if _, err := tx.Exec(ctx, `INSERT INTO documents (tenant_id, id, title, body, acl, version, status, effective_from,
				effective_to, scope, hold) VALUES (current_setting('eais.tenant_id'), $1, $2, $3, COALESCE($4::text[], '{}'), $5, $6, $7, $8, COALESCE($9::text[], '{}'), $10)`,
				d.ID, d.Title, d.Body, d.ACL, d.Version, d.Status, d.EffectiveFrom, d.EffectiveTo, d.Scope, d.Hold); err != nil {
				return err
			}
		}
		_, err := tx.Exec(ctx, `INSERT INTO load_records (tenant_id, load_id, data_version)
			VALUES (current_setting('eais.tenant_id'), $1, $2)`, info.LoadID, info.DataVersion)
		return err
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
