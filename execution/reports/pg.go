package reports

// Kho PostgreSQL của bước 2 và 3 (migration kho_bao_cao_ngay): cài giao diện Store trên bốn bảng report_records, report_versions, report_history, delivery_ledger. CÙNG engine với kho
// bộ nhớ (engine.go, tr3.go, decide.go): gói này chỉ nói SQL, nên hai kho không thể lệch nhau về luật. Mở giao dịch đã đặt tenant là việc của platform/pgstore (TxBeginner), nên tenant luôn
// đến từ ngữ cảnh (contracts.TenantFrom) và RLS chặn phần còn lại.
//
// MỖI thao tác (một lần giao, một quyết định, một thu hồi, một sửa tin) là MỘT giao dịch: khoá tư vấn các ô bị chạm (§5.6 mục 2) và các bản ghi có sẵn bị sửa (getLocked, không khoá
// hàng: R-01), đọc, tính, rồi ghi một lượt (bản ghi, phiên bản, lịch sử, dòng sổ) trước khi COMMIT; lỗi ở đâu cũng huỷ cả giao dịch, nên không có đơn vị nửa ghi (§7.5, LC67).
// Va chạm khoá duy nhất (hai giao dịch cùng tạo một unit_key ở hai ô khác nhau) và nhu cầu lấy khoá theo thứ tự làm giao dịch bị bỏ và chạy lại (retryOp): bên thua đọc lại, thấy bản của bên thắng và đi theo đường TR1.
//
// Lỗi: lỗi của bên gọi và lỗi đã gắn nhãn giữ nguyên; lỗi PostgreSQL được phân loại như platform/pgstore (tạm thời ⇒ hạ tầng; vi phạm ràng buộc ⇒ dữ liệu; còn lại ⇒ mã).

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/luannt2002/enterprise-ai-control-layer/core/contracts"
)

// TxBeginner là phần của platform/pgstore (Store.BeginTenantTx) mà kho PG cần: mở một giao dịch đã đặt GUC tenant từ ngữ cảnh và trả giao dịch để kho tự Commit / Rollback.
// Khai tại chỗ dùng nên gói này không import platform/pgstore.
type TxBeginner interface {
	BeginTenantTx(ctx context.Context) (pgx.Tx, error)
}

// PG là kho bản ghi báo cáo trên PostgreSQL. An toàn cho nhiều goroutine và nhiều tiến trình: không có trạng thái riêng ngoài cấu hình.
type PG struct {
	db  TxBeginner
	opt Options
	// Chỉ test đặt (pg_test.go, tr3_lock_test.go): điểm dừng giữa lúc đọc nhóm và lúc ghi (DB26); hook gọi ngay trước mỗi câu lệnh ghi (op là tên câu lệnh) để chèn lỗi hay kết thúc backend (LC67).
	afterSlotRead func()
	hook          func(ctx context.Context, tx pgx.Tx, op string) error
}

// NewPG dựng kho PG trên db (thường là *pgstore.Store).
func NewPG(db TxBeginner, opt Options) *PG { return &PG{db: db, opt: opt} }

var _ Store = (*PG)(nil)

const recordCols = `unit_key, channel, source_type, sender_id, sent_at, COALESCE(to_char(occurred_on, 'YYYY-MM-DD'), ''), entity, metric, value_number, value_unit, approx, kind,
	corrects, raw, attachments::text, note, disposition, reason, detail, recipient, duplicate_of, superseded_by, version, decided_by, decided_at, decision, reviewed_pairs,
	pack_version, trace_id, fingerprint, content_sha256`

const (
	sqlInsertRecord = `INSERT INTO report_records (tenant_id, unit_key, channel, source_type, sender_id, sent_at, occurred_on, entity, metric, value_number, value_unit, approx, kind,
			corrects, raw, attachments, note, disposition, reason, detail, recipient, duplicate_of, superseded_by, version, decided_by, decided_at, decision, reviewed_pairs,
			pack_version, trace_id, fingerprint, content_sha256)
		VALUES (current_setting('eais.tenant_id'), $1, $2, $3, $4, $5, NULLIF($6::text, '')::date, $7, $8, $9, $10, $11, $12, $13, $14, $15::jsonb, $16, $17, $18, $19, $20, $21, $22, $23, $24, $25,
			$26, $27, $28, $29, $30, $31)
		ON CONFLICT (tenant_id, unit_key) DO NOTHING`

	sqlUpdateRecord = `UPDATE report_records SET channel = $2, source_type = $3, sender_id = $4, sent_at = $5, occurred_on = NULLIF($6::text, '')::date, entity = $7, metric = $8,
			value_number = $9, value_unit = $10, approx = $11, kind = $12, corrects = $13, raw = $14, attachments = $15::jsonb, note = $16, disposition = $17, reason = $18, detail = $19,
			recipient = $20, duplicate_of = $21, superseded_by = $22, version = $23, decided_by = $24, decided_at = $25, decision = $26, reviewed_pairs = $27, pack_version = $28,
			trace_id = $29, fingerprint = $30, content_sha256 = $31
		WHERE unit_key = $1`

	sqlInsertVersion = `INSERT INTO report_versions (tenant_id, unit_key, version, content_sha256, fingerprint, raw, snapshot)
		VALUES (current_setting('eais.tenant_id'), $1, $2, $3, $4, $5, $6::jsonb)`

	// seq tính ngay trong câu lệnh: các thao tác cùng một bản ghi đã xếp hàng bằng khoá ô và khoá bản ghi (getLocked), nên không đua.
	sqlInsertHistory = `INSERT INTO report_history (tenant_id, unit_key, seq, at, actor, event, from_disposition, to_disposition, reason, detail, version, duplicate_of, decision, pack_version)
		VALUES (current_setting('eais.tenant_id'), $1, (SELECT coalesce(max(seq), 0) + 1 FROM report_history WHERE unit_key = $1), $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)`

	sqlInsertLedger = `INSERT INTO delivery_ledger (tenant_id, batch_id, ordinal, unit_key, outcome, reason, detail, record_ref, pack_version, trace_id, recorded_at)
		VALUES (current_setting('eais.tenant_id'), $1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`
)

// classifyPG gắn nhãn "lỗi do ai" cho lỗi của một giao dịch, cùng quy tắc với platform/pgstore: tạm thời ⇒ hạ tầng (thử lại có ích); vi phạm ràng buộc ⇒ dữ liệu; còn lại ⇒ mã. Lỗi đã có
// nhãn giữ nguyên (contracts.Label giữ nhãn gần nhất); hai lỗi nội bộ của việc chạy lại đi qua nguyên vẹn.
func classifyPG(err error) error {
	if err == nil {
		return nil
	}
	var rl *relockError
	if errors.As(err, &rl) || errors.Is(err, errKeyRace) || contracts.FaultOf(err) != "" {
		return err
	}
	var pe *pgconn.PgError
	if !errors.As(err, &pe) {
		if errors.Is(err, pgx.ErrNoRows) || errors.Is(err, pgx.ErrTxCommitRollback) {
			return contracts.Label(contracts.FaultCode, err)
		}
		return contracts.Label(contracts.Infra("postgres"), err)
	}
	class := pe.Code
	if len(class) >= 2 {
		class = class[:2]
	}
	switch {
	case class == "08" || class == "40" || class == "53" || class == "57" || class == "58" || pe.Code == "55P03" || pe.Code == "25006" || pe.Code == "25P03" || pe.Code == "25P04":
		return contracts.Label(contracts.Infra("postgres"), err)
	case class == "22" || class == "23":
		return contracts.Label(contracts.FaultData, err)
	default:
		return contracts.Label(contracts.FaultCode, err)
	}
}

// run chạy fn như MỘT giao dịch (có chạy lại khi va chạm khoá hoặc cần lấy khoá theo thứ tự).
func (s *PG) run(ctx context.Context, fn func(c *opctx) error) error {
	tenant, err := tenantFrom(ctx)
	if err != nil {
		return err
	}
	return retryOp(func(want []lockRef) error {
		tx, err := s.db.BeginTenantTx(ctx)
		if err != nil {
			return err
		}
		defer func() { _ = tx.Rollback(context.WithoutCancel(ctx)) }() // sau Commit thành công là no-op
		c := newOpctx(tenant, s.opt, &pgTx{ctx: ctx, tx: tx, tenant: tenant, s: s})
		c.afterSlotRead = s.afterSlotRead
		err = c.ensureLocks(want)
		if err == nil {
			err = fn(c)
		}
		commit, err := c.close(err)
		if commit {
			if cerr := tx.Commit(ctx); cerr != nil {
				return classifyPG(cerr)
			}
		}
		return classifyPG(err)
	})
}

// read chạy fn trong một giao dịch chỉ đọc đã đặt tenant rồi huỷ nó (không có gì để commit).
func (s *PG) read(ctx context.Context, fn func(x *pgTx) error) error {
	tenant, err := tenantFrom(ctx)
	if err != nil {
		return err
	}
	tx, err := s.db.BeginTenantTx(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(context.WithoutCancel(ctx)) }()
	return classifyPG(fn(&pgTx{ctx: ctx, tx: tx, tenant: tenant, s: s}))
}

func (s *PG) Deliver(ctx context.Context, batchID string, ordinal int, it Item) (DeliveryEntry, error) {
	var out DeliveryEntry
	err := s.run(ctx, func(c *opctx) error {
		e, err := c.deliver(batchID, ordinal, it)
		out = e
		return err
	})
	return out, err
}

func (s *PG) DeliverBatch(ctx context.Context, items []Item) (Receipt, error) {
	return deliverBatch(ctx, s, items)
}

func (s *PG) Decide(ctx context.Context, cmd Command) (Record, error) {
	var out Record
	err := s.run(ctx, func(c *opctx) error {
		r, err := c.decide(cmd)
		out = r
		return err
	})
	return out, err
}

func (s *PG) Retract(ctx context.Context, key, actor, why string) error {
	return s.run(ctx, func(c *opctx) error { return c.retract(key, actor, why) })
}

func (s *PG) Amend(ctx context.Context, key string, it Item, actor, why string) error {
	return s.run(ctx, func(c *opctx) error { return c.amend(key, it, actor, why) })
}

func (s *PG) Record(ctx context.Context, key string) (Record, bool, error) {
	var out Record
	var ok bool
	err := s.read(ctx, func(x *pgTx) error {
		r, err := x.getRecord(key)
		if r != nil {
			out, ok = *r, true
		}
		return err
	})
	return out, ok, err
}

func (s *PG) Records(ctx context.Context) ([]Record, error) {
	var out []Record
	err := s.read(ctx, func(x *pgTx) error {
		rows, err := x.tx.Query(x.ctx, `SELECT `+recordCols+` FROM report_records ORDER BY unit_key COLLATE "C"`)
		if err != nil {
			return err
		}
		rs, err := x.scanRecords(rows)
		for _, r := range rs {
			out = append(out, *r)
		}
		return err
	})
	sortRecords(out)
	return out, err
}

func (s *PG) Ledger(ctx context.Context, batchID string) ([]DeliveryEntry, error) {
	var out []DeliveryEntry
	err := s.read(ctx, func(x *pgTx) error {
		rows, err := x.tx.Query(x.ctx, `SELECT batch_id, ordinal, unit_key, outcome, reason, detail, record_ref, pack_version, trace_id, recorded_at
			FROM delivery_ledger WHERE ($1 = '' OR batch_id = $1) ORDER BY batch_id COLLATE "C", ordinal`, batchID)
		if err != nil {
			return err
		}
		defer rows.Close()
		for rows.Next() {
			e := DeliveryEntry{Tenant: x.tenant}
			var outcome, reason string
			var ord int32
			var at time.Time
			if err := rows.Scan(&e.BatchID, &ord, &e.Key, &outcome, &reason, &e.Detail, &e.RecordRef, &e.PackVersion, &e.TraceID, &at); err != nil {
				return err
			}
			e.Ordinal, e.Outcome, e.Reason, e.RecordedAt = int(ord), Outcome(outcome), Reason(reason), at.UTC()
			out = append(out, e)
		}
		return rows.Err()
	})
	sortLedger(out)
	return out, err
}

func (s *PG) History(ctx context.Context, key string) ([]HistoryEntry, error) {
	var out []HistoryEntry
	err := s.read(ctx, func(x *pgTx) error {
		rows, err := x.tx.Query(x.ctx, `SELECT unit_key, seq, at, actor, event, from_disposition, to_disposition, reason, detail, version, duplicate_of, decision, pack_version
			FROM report_history WHERE unit_key = $1 ORDER BY seq`, key)
		if err != nil {
			return err
		}
		defer rows.Close()
		for rows.Next() {
			h := HistoryEntry{Tenant: x.tenant}
			var seq, ver int32
			var at time.Time
			var from, to, reason, dec string
			if err := rows.Scan(&h.Key, &seq, &at, &h.Actor, &h.Event, &from, &to, &reason, &h.Detail, &ver, &h.DuplicateOf, &dec, &h.PackVersion); err != nil {
				return err
			}
			h.Seq, h.At, h.From, h.To, h.Reason, h.Version, h.Decision = int(seq), at.UTC(), Disposition(from), Disposition(to), Reason(reason), int(ver), Decision(dec)
			out = append(out, h)
		}
		return rows.Err()
	})
	return out, err
}

func (s *PG) Receipt(ctx context.Context, batchID string) (Receipt, error) {
	es, err := s.Ledger(ctx, batchID)
	return BuildReceipt(batchID, es), err
}

func (s *PG) Totals(ctx context.Context) ([]Total, error) {
	rs, err := s.Records(ctx)
	if err != nil {
		return nil, err
	}
	return totalsOf(rs), nil
}

// ---------------------------------------------------------------------------------------------------------------------
// pgTx cài txn trên một giao dịch PG đã đặt tenant.

type pgTx struct {
	ctx    context.Context
	tx     pgx.Tx
	tenant string
	s      *PG
}

func (x *pgTx) before(op string) error {
	if x.s.hook != nil {
		return x.s.hook(x.ctx, x.tx, op)
	}
	return nil
}

func (x *pgTx) lock(k int64) error {
	_, err := x.tx.Exec(x.ctx, "SELECT pg_advisory_xact_lock($1)", k)
	return err
}

func (x *pgTx) getRecord(key string) (*Record, error) {
	rows, err := x.tx.Query(x.ctx, `SELECT `+recordCols+` FROM report_records WHERE unit_key = $1`, key)
	if err != nil {
		return nil, err
	}
	rs, err := x.scanRecords(rows)
	if err != nil || len(rs) == 0 {
		return nil, err
	}
	return rs[0], nil
}

func (x *pgTx) slotRecords(entity, day string) ([]*Record, error) {
	rows, err := x.tx.Query(x.ctx, `SELECT `+recordCols+` FROM report_records WHERE entity = $1 AND occurred_on = $2::date ORDER BY unit_key COLLATE "C"`, entity, day)
	if err != nil {
		return nil, err
	}
	return x.scanRecords(rows)
}

// scanRecords đọc hết rows rồi đóng nó. Danh sách rỗng trả về nil, mảng rỗng trả về nil: cùng dạng với bản chép của kho bộ nhớ.
func (x *pgTx) scanRecords(rows pgx.Rows) ([]*Record, error) {
	defer rows.Close()
	var out []*Record
	for rows.Next() {
		r := &Record{Tenant: x.tenant}
		var sentAt, decidedAt *time.Time
		var raw []byte
		var att, kind, disp, reason, dec string
		var ver int32
		var pairs []string
		if err := rows.Scan(&r.Key, &r.Channel, &r.SourceType, &r.SenderID, &sentAt, &r.OccurredOn, &r.Entity, &r.Metric, &r.Value.Number, &r.Value.Unit, &r.Approx, &kind,
			&r.Corrects, &raw, &att, &r.Note, &disp, &reason, &r.Detail, &r.Recipient, &r.DuplicateOf, &r.SupersededBy, &ver, &r.DecidedBy, &decidedAt, &dec, &pairs,
			&r.PackVersion, &r.TraceID, &r.Fingerprint, &r.ContentHash); err != nil {
			return nil, err
		}
		if sentAt != nil {
			r.SentAt = sentAt.UTC()
		}
		if decidedAt != nil {
			r.DecidedAt = decidedAt.UTC()
		}
		r.Kind, r.Disposition, r.Reason, r.Decision, r.Version, r.Raw = Kind(kind), Disposition(disp), Reason(reason), Decision(dec), int(ver), string(raw)
		if err := json.Unmarshal([]byte(att), &r.Attachments); err != nil {
			return nil, fmt.Errorf("cột attachments của %q hỏng: %w", r.Key, err)
		}
		if len(r.Attachments) == 0 {
			r.Attachments = nil
		}
		if len(pairs) > 0 {
			r.ReviewedPairs = pairs
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

func (x *pgTx) versionsOf(key string) ([]versionRow, error) {
	rows, err := x.tx.Query(x.ctx, `SELECT version, content_sha256, fingerprint, raw, snapshot::text FROM report_versions WHERE unit_key = $1 ORDER BY version`, key)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []versionRow
	for rows.Next() {
		v := versionRow{Key: key}
		var ver int32
		var raw []byte
		var snap string
		if err := rows.Scan(&ver, &v.ContentHash, &v.Fingerprint, &raw, &snap); err != nil {
			return nil, err
		}
		v.Version, v.Raw, v.Snapshot = int(ver), string(raw), []byte(snap)
		out = append(out, v)
	}
	return out, rows.Err()
}

func (x *pgTx) findByFingerprint(fp string) (string, error) {
	var key string
	err := x.tx.QueryRow(x.ctx, `SELECT unit_key FROM report_versions WHERE fingerprint = $1 ORDER BY unit_key COLLATE "C" LIMIT 1`, fp).Scan(&key)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", nil
	}
	return key, err
}

func nullTime(t time.Time) *time.Time {
	if t.IsZero() {
		return nil
	}
	return &t
}

// recordArgs là các tham số $1.. của sqlInsertRecord và sqlUpdateRecord (cùng thứ tự: khoá, rồi các cột còn lại).
func recordArgs(r *Record) ([]any, error) {
	att := r.Attachments
	if att == nil {
		att = []Attachment{}
	}
	attJSON, err := json.Marshal(att)
	if err != nil {
		return nil, err
	}
	pairs := r.ReviewedPairs
	if pairs == nil {
		pairs = []string{}
	}
	raw := []byte(r.Raw)
	if raw == nil {
		raw = []byte{}
	}
	return []any{r.Key, r.Channel, r.SourceType, r.SenderID, nullTime(r.SentAt), r.OccurredOn, r.Entity, r.Metric, r.Value.Number, r.Value.Unit, r.Approx, string(r.Kind),
		r.Corrects, raw, string(attJSON), r.Note, string(r.Disposition), string(r.Reason), r.Detail, r.Recipient, r.DuplicateOf, r.SupersededBy, int32(r.Version), r.DecidedBy,
		nullTime(r.DecidedAt), string(r.Decision), pairs, r.PackVersion, r.TraceID, r.Fingerprint, r.ContentHash}, nil
}

func (x *pgTx) insertRecord(r *Record) error {
	if err := x.before("insertRecord"); err != nil {
		return err
	}
	args, err := recordArgs(r)
	if err != nil {
		return err
	}
	tag, err := x.tx.Exec(x.ctx, sqlInsertRecord, args...)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return errKeyRace // ON CONFLICT DO NOTHING: giao dịch khác vừa tạo khoá này; giao dịch không bị huỷ, bên gọi bỏ nó và chạy lại
	}
	return nil
}

func (x *pgTx) updateRecord(r *Record) error {
	if err := x.before("updateRecord"); err != nil {
		return err
	}
	args, err := recordArgs(r)
	if err != nil {
		return err
	}
	tag, err := x.tx.Exec(x.ctx, sqlUpdateRecord, args...)
	if err != nil {
		return err
	}
	if tag.RowsAffected() != 1 {
		return contracts.Errorf(contracts.FaultCode, fmt.Sprintf("cập nhật bản ghi %q: %d dòng đổi, muốn 1", r.Key, tag.RowsAffected()))
	}
	return nil
}

func (x *pgTx) insertVersion(v versionRow) error {
	if err := x.before("insertVersion"); err != nil {
		return err
	}
	raw := []byte(v.Raw)
	if raw == nil {
		raw = []byte{}
	}
	_, err := x.tx.Exec(x.ctx, sqlInsertVersion, v.Key, int32(v.Version), v.ContentHash, v.Fingerprint, raw, string(v.Snapshot))
	return err
}

func (x *pgTx) insertHistory(h HistoryEntry) error {
	if err := x.before("insertHistory"); err != nil {
		return err
	}
	_, err := x.tx.Exec(x.ctx, sqlInsertHistory, h.Key, h.At, h.Actor, h.Event, string(h.From), string(h.To), string(h.Reason), h.Detail, int32(h.Version), h.DuplicateOf, string(h.Decision), h.PackVersion)
	return err
}

func (x *pgTx) insertLedger(e DeliveryEntry) error {
	if err := x.before("insertLedger"); err != nil {
		return err
	}
	_, err := x.tx.Exec(x.ctx, sqlInsertLedger, e.BatchID, int32(e.Ordinal), e.Key, string(e.Outcome), string(e.Reason), e.Detail, e.RecordRef, e.PackVersion, e.TraceID, e.RecordedAt)
	return err
}
