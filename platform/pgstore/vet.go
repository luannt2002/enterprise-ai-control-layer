package pgstore

// Kho vết bền theo tenant (migration 0003): bản ghi quyết định của mọi lượt hỏi, bản ghi cách ly của lần nạp, usage model.
//
// Chung cho cả ba: tenant lấy từ ngữ cảnh (contracts.WithTenant), không nhận từ tham số; RLS chặn phần còn lại; vai ứng dụng chỉ
// có SELECT + INSERT nên không có hàm sửa hay xoá. Mọi chuỗi qua traces.Sanitize trước khi ghi (che bí mật, bỏ NUL, sửa UTF-8).
//
// Ghi lại cùng khoá KHÔNG nhân đôi và KHÔNG lỗi im lặng, theo MỘT luật cho cả ba bảng:
//   - bản ghi y nguyên (cùng nội dung) đã có: bỏ qua và báo (inserted = false / đếm 0), không lỗi — cho phép thử lại sau khi mất
//     kết nối giữa chừng mà không nhân đôi;
//   - cùng khoá mà khác nội dung: lỗi FaultData, KHÔNG ghi đè — vết là bằng chứng, hai nội dung cho một khoá là có chuyện cần
//     người xem (đụng trace_id, mã nạp dùng lại, nguồn không tất định), không phải chuyện để âm thầm chọn một bên.

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/luannt2002/enterprise-ai-control-layer/core/contracts"
	"github.com/luannt2002/enterprise-ai-control-layer/observability/traces"
)

// ErrNotFound: không có bản ghi với khoá đó trong tenant của ngữ cảnh. Bản ghi của tenant khác cũng báo thế — RLS không để phân
// biệt "không có" với "của người khác", và cũng không nên. Nơi gọi dùng errors.Is; nhãn lỗi là FaultInput.
var ErrNotFound = errors.New("không có bản ghi")

// pgFKViolation là SQLSTATE foreign_key_violation.
const pgFKViolation = "23503"

// SaveDecision lưu bản ghi vết của một lượt (kể cả lượt lỗi). Tenant của bản ghi phải KHỚP tenant của ngữ cảnh: khác là lỗi chính
// sách chứ không im lặng ghi sang tenant kia. Trả inserted = true khi ghi mới, false khi đúng bản ghi này đã có (xem luật ghi lại
// ở đầu tệp).
func (s *Store) SaveDecision(ctx context.Context, rec traces.Record) (inserted bool, err error) {
	tenant, err := tenantOf(ctx)
	if err != nil {
		return false, err
	}
	if err := rec.Validate(); err != nil {
		return false, contracts.Errorf(contracts.FaultInput, "bản ghi vết không hợp lệ: "+err.Error())
	}
	if rec.Decision.Tenant != string(tenant) {
		return false, contracts.Errorf(contracts.FaultPolicy, "bản ghi vết mang tenant khác tenant của ngữ cảnh")
	}
	clean, err := traces.SanitizeRecord(rec)
	if err != nil {
		return false, contracts.Label(contracts.FaultCode, err)
	}
	digest, err := clean.Digest()
	if err != nil {
		return false, contracts.Label(contracts.FaultCode, err)
	}
	var cols [3][]byte // decision, steps, spans
	for i, v := range []any{clean.Decision, clean.Steps, clean.Spans} {
		if cols[i], err = json.Marshal(v); err != nil {
			return false, contracts.Label(contracts.FaultCode, err)
		}
	}
	d := clean.Decision
	err = s.inTenant(ctx, tenant, func(tx pgx.Tx) error {
		tag, err := tx.Exec(ctx, `
			INSERT INTO decision_records (tenant_id, trace_id, asked_at, principal_id, principal_kind, question, intent, route,
				outcome, verdict, answer, fault, error_message, pack_version, code_version, data_version, total_ms, unaccounted_ms,
				decision, steps, spans, content_sha256)
			VALUES (current_setting('eais.tenant_id'), $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17,
				$18::jsonb, $19::jsonb, $20::jsonb, $21)
			ON CONFLICT (tenant_id, trace_id) DO NOTHING`,
			d.TraceID, d.Now, d.Principal, d.Kind, d.Question, d.Intent, d.Route, d.Outcome, d.Verdict, d.Answer,
			clean.Fault, clean.Error, d.PackVersion, d.CodeVersion, d.DataVersion, clean.TotalMs, clean.UnaccountedMs,
			cols[0], cols[1], cols[2], digest)
		if err != nil {
			return err
		}
		if tag.RowsAffected() == 1 {
			inserted = true
			return nil
		}
		var stored string
		if err := tx.QueryRow(ctx, "SELECT content_sha256 FROM decision_records WHERE trace_id = $1", d.TraceID).Scan(&stored); err != nil {
			return err
		}
		if stored != digest {
			return contracts.Errorf(contracts.FaultData, "trace_id "+d.TraceID+" đã có bản ghi với nội dung khác; không ghi đè")
		}
		return nil
	})
	return inserted, err
}

// Decision đọc lại bản ghi vết theo trace id trong tenant của ngữ cảnh. Không có (hoặc của tenant khác — RLS) thì lỗi bọc
// ErrNotFound.
func (s *Store) Decision(ctx context.Context, traceID string) (traces.Record, error) {
	tenant, err := tenantOf(ctx)
	if err != nil {
		return traces.Record{}, err
	}
	var rec traces.Record
	err = s.inTenant(ctx, tenant, func(tx pgx.Tx) error {
		err := tx.QueryRow(ctx, `
			SELECT decision, steps, spans, total_ms, unaccounted_ms, fault, error_message
			FROM decision_records WHERE trace_id = $1`, traceID).
			Scan(&rec.Decision, &rec.Steps, &rec.Spans, &rec.TotalMs, &rec.UnaccountedMs, &rec.Fault, &rec.Error)
		if errors.Is(err, pgx.ErrNoRows) {
			return contracts.Label(contracts.FaultInput, fmt.Errorf("%w: trace %q", ErrNotFound, traceID))
		}
		return err
	})
	return rec, err
}

// QuarantineRecord là một bản ghi bị cách ly của một lần nạp — luôn kèm lý do, không bao giờ rơi im lặng. Cùng hình với
// ingestion.Quarantine (trừ tenant, lấy từ ngữ cảnh).
type QuarantineRecord struct {
	Key    string // khoá bản ghi "<tệp>#<mã bản ghi | số dòng>"
	Source string // loại nguồn
	Reason string // mã lý do (missing_value, unknown_entity, out_of_range…); không được rỗng
	Detail string
	At     time.Time // giờ của DB lúc ghi; chỉ có khi đọc
}

// SaveQuarantine lưu bản ghi cách ly của lần nạp loadID. Danh tính một dòng là (loadID, Key, Reason). Trả số dòng ghi MỚI: ghi
// lại cùng tập y nguyên trả 0; cùng (Key, Reason) mà khác Source / Detail là lỗi FaultData và cả lô bị huỷ (xem luật ghi lại ở
// đầu tệp). Bản ghi thiếu Key hoặc Reason là lỗi đầu vào — không có lý do thì không phải bản ghi cách ly.
func (s *Store) SaveQuarantine(ctx context.Context, loadID string, recs []QuarantineRecord) (int, error) {
	tenant, err := tenantOf(ctx)
	if err != nil {
		return 0, err
	}
	if loadID == "" {
		return 0, contracts.Errorf(contracts.FaultInput, "thiếu mã lần nạp")
	}
	var keys, reasons, sources, details []string
	for i, r := range recs {
		if r.Key == "" || r.Reason == "" {
			return 0, contracts.Errorf(contracts.FaultInput, fmt.Sprintf("bản ghi cách ly %d thiếu khoá hoặc lý do", i))
		}
		keys, reasons = append(keys, traces.Sanitize(r.Key)), append(reasons, traces.Sanitize(r.Reason))
		sources, details = append(sources, traces.Sanitize(r.Source)), append(details, traces.Sanitize(r.Detail))
	}
	if len(recs) == 0 {
		return 0, nil
	}
	inserted := 0
	err = s.inTenant(ctx, tenant, func(tx pgx.Tx) error {
		tag, err := tx.Exec(ctx, `
			INSERT INTO quarantine_records (tenant_id, load_id, record_key, reason, source, detail)
			SELECT current_setting('eais.tenant_id'), $1, * FROM unnest($2::text[], $3::text[], $4::text[], $5::text[])
			ON CONFLICT DO NOTHING`, loadID, keys, reasons, sources, details)
		if err != nil {
			return err
		}
		inserted = int(tag.RowsAffected())
		var bad int
		if err := tx.QueryRow(ctx, `
			SELECT count(*) FROM unnest($2::text[], $3::text[], $4::text[], $5::text[]) AS u(record_key, reason, source, detail)
			LEFT JOIN quarantine_records q ON q.load_id = $1 AND q.record_key = u.record_key AND q.reason = u.reason
			WHERE q.record_key IS NULL OR q.source IS DISTINCT FROM u.source OR q.detail IS DISTINCT FROM u.detail`,
			loadID, keys, reasons, sources, details).Scan(&bad); err != nil {
			return err
		}
		if bad > 0 {
			return contracts.Errorf(contracts.FaultData, fmt.Sprintf("lần nạp %s: %d bản ghi cách ly đã có với nội dung khác; không ghi đè", loadID, bad))
		}
		return nil
	})
	if err != nil {
		return 0, err
	}
	return inserted, nil
}

// Quarantine liệt kê bản ghi cách ly của lần nạp loadID trong tenant của ngữ cảnh, xếp theo khoá rồi lý do (so byte, ổn định giữa
// các DB).
func (s *Store) Quarantine(ctx context.Context, loadID string) ([]QuarantineRecord, error) {
	tenant, err := tenantOf(ctx)
	if err != nil {
		return nil, err
	}
	var out []QuarantineRecord
	err = s.inTenant(ctx, tenant, func(tx pgx.Tx) error {
		rows, err := tx.Query(ctx, `
			SELECT record_key, source, reason, detail, recorded_at FROM quarantine_records WHERE load_id = $1
			ORDER BY record_key COLLATE "C", reason COLLATE "C"`, loadID)
		if err != nil {
			return err
		}
		defer rows.Close()
		for rows.Next() {
			var r QuarantineRecord
			if err := rows.Scan(&r.Key, &r.Source, &r.Reason, &r.Detail, &r.At); err != nil {
				return err
			}
			out = append(out, r)
		}
		return rows.Err()
	})
	return out, err
}

// UsageRecord là usage model của một lượt, gộp theo model.
type UsageRecord struct {
	TraceID string
	Model   string
	Calls   int
	Tokens  contracts.Usage
}

// SaveUsage lưu usage của một (lượt, model). Lượt phải có bản ghi quyết định TRƯỚC (khoá ngoại): chưa có thì lỗi đầu vào. Trả
// inserted = false khi đúng usage này đã có; cùng (lượt, model) mà số khác là lỗi FaultData (xem luật ghi lại ở đầu tệp).
func (s *Store) SaveUsage(ctx context.Context, u UsageRecord) (inserted bool, err error) {
	tenant, err := tenantOf(ctx)
	if err != nil {
		return false, err
	}
	t := u.Tokens
	if u.TraceID == "" || u.Model == "" || u.Calls < 0 || t.InputTokens < 0 || t.OutputTokens < 0 || t.CacheRead < 0 || t.CacheWrite < 0 {
		return false, contracts.Errorf(contracts.FaultInput, "usage cần trace_id, model và số không âm")
	}
	model := traces.Sanitize(u.Model)
	err = s.inTenant(ctx, tenant, func(tx pgx.Tx) error {
		tag, err := tx.Exec(ctx, `
			INSERT INTO usage_records (tenant_id, trace_id, model, calls, input_tokens, output_tokens, cache_read, cache_write)
			VALUES (current_setting('eais.tenant_id'), $1, $2, $3, $4, $5, $6, $7)
			ON CONFLICT DO NOTHING`, u.TraceID, model, u.Calls, t.InputTokens, t.OutputTokens, t.CacheRead, t.CacheWrite)
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == pgFKViolation {
			return contracts.Errorf(contracts.FaultInput, "usage của trace "+u.TraceID+" khi chưa có bản ghi quyết định của lượt đó")
		}
		if err != nil {
			return err
		}
		if tag.RowsAffected() == 1 {
			inserted = true
			return nil
		}
		var got UsageRecord
		if err := tx.QueryRow(ctx, `
			SELECT calls, input_tokens, output_tokens, cache_read, cache_write FROM usage_records WHERE trace_id = $1 AND model = $2`,
			u.TraceID, model).Scan(&got.Calls, &got.Tokens.InputTokens, &got.Tokens.OutputTokens, &got.Tokens.CacheRead, &got.Tokens.CacheWrite); err != nil {
			return err
		}
		if got.Calls != u.Calls || got.Tokens != t {
			return contracts.Errorf(contracts.FaultData, "usage của trace "+u.TraceID+" model "+model+" đã có với số khác; không ghi đè")
		}
		return nil
	})
	return inserted, err
}

// Usage liệt kê usage của một lượt trong tenant của ngữ cảnh, xếp theo model.
func (s *Store) Usage(ctx context.Context, traceID string) ([]UsageRecord, error) {
	tenant, err := tenantOf(ctx)
	if err != nil {
		return nil, err
	}
	var out []UsageRecord
	err = s.inTenant(ctx, tenant, func(tx pgx.Tx) error {
		rows, err := tx.Query(ctx, `
			SELECT trace_id, model, calls, input_tokens, output_tokens, cache_read, cache_write FROM usage_records
			WHERE trace_id = $1 ORDER BY model COLLATE "C"`, traceID)
		if err != nil {
			return err
		}
		defer rows.Close()
		for rows.Next() {
			var u UsageRecord
			if err := rows.Scan(&u.TraceID, &u.Model, &u.Calls, &u.Tokens.InputTokens, &u.Tokens.OutputTokens, &u.Tokens.CacheRead, &u.Tokens.CacheWrite); err != nil {
				return err
			}
			out = append(out, u)
		}
		return rows.Err()
	})
	return out, err
}
