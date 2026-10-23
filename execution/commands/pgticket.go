package commands

// Kho PG của phiếu (HOP-DONG §6, §7; ADR-0011 "Nơi đặt mã"): cài giao diện Store trên PostgreSQL với ba bảng của migration
// 0004 (discrepancy_tickets, commands, audit_log). Gói này chỉ nói SQL của ba bảng đó; mở giao dịch đã đặt tenant là việc của
// platform/pgstore (TxBeginner), nên tenant luôn đến từ ngữ cảnh (contracts.TenantFrom, C1 đã gắn) và RLS chặn phần còn lại.
//
// Đơn vị ghi phiếu (WriteTicket) là MỘT giao dịch: BEGIN, đặt tenant, INSERT phiếu, INSERT sổ lệnh, INSERT audit EXECUTED, COMMIT, đúng
// thứ tự đó (sổ lệnh có khoá ngoại thường tới phiếu). Lỗi của đơn vị ghi chỉ có ba loại, và loại nào cũng gắn thêm lỗi gốc bằng %w:
//
//	ErrBeforeCommit    biết CHẮC chưa ghi: lỗi một câu lệnh (giao dịch đã huỷ), lỗi kết nối hay hết hạn TRƯỚC khi gửi COMMIT (kể cả
//	                   khi đang chờ khoá), hoặc COMMIT chưa rời máy (ngữ cảnh đã xong từ trước), hoặc PG trả lỗi không-FATAL cho
//	                   chính COMMIT;
//	ErrUniqueViolation SQLSTATE 23505, ở bất cứ câu lệnh nào kể cả COMMIT; giao dịch đã huỷ. Pipeline đọc lại để phân loại (§6.4);
//	ErrCommitUnknown   từ lúc COMMIT đã gửi đi mà không có câu trả lời "đã huỷ" của PG: lỗi kết nối, backend bị kết thúc, hết hạn.
//
// Bảng phân loại là kết quả ĐO của thí nghiệm U1 trên PostgreSQL 17 và pgx v5.11.0 (HOP-DONG §15; test pgticket_u1_test.go tái lập
// và khẳng định từng dòng, nên pgx đổi hình dạng lỗi thì test đỏ). Ý chính của bảng đo:
//   - mọi lỗi TRƯỚC khi gửi COMMIT chắc chắn chưa ghi, vì giao dịch chưa commit không sống sót qua việc kết nối đứt;
//   - lỗi của chính COMMIT chỉ chắc chắn "chưa ghi" khi PG trả *pgconn.PgError mức ERROR (giao dịch bị huỷ, kể cả lỗi hoãn tới
//     COMMIT) hoặc khi COMMIT chưa rời máy (pgconn.SafeToRetry, ví dụ ngữ cảnh đã xong từ trước) — nhưng SafeToRetry KHÔNG đáng tin
//     khi lỗi là "conn closed": mạng đứt lúc chờ câu trả lời của COMMIT cũng cho "conn closed" mang SafeToRetry() = true dù COMMIT đã
//     tới máy chủ và đã commit (ca c4 của bảng đo), nên "conn closed" luôn tính là không biết;
//   - hết hạn hay đứt kết nối TRONG lúc COMMIT chạy thì không biết: máy chủ vẫn có thể hoàn tất COMMIT sau khi bên gọi bỏ đi (ca c3).
//
// Mọi đường đọc trả lỗi thô (pipeline gắn nhãn infra:postgres); chỉ đơn vị ghi cần phân loại.

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/luannt2002/enterprise-ai-control-layer/core/contracts"
)

// TxBeginner là phần của platform/pgstore (Store.BeginTenantTx) mà kho PG cần: mở một giao dịch đã đặt GUC tenant từ ngữ cảnh và
// trả giao dịch để kho tự Commit / Rollback. Khai tại chỗ dùng nên gói này không import platform/pgstore.
type TxBeginner interface {
	BeginTenantTx(ctx context.Context) (pgx.Tx, error)
}

// PGStore là kho phiếu trên PostgreSQL. An toàn cho nhiều lượt chạy đồng thời: không có trạng thái riêng.
type PGStore struct {
	db TxBeginner
	// beforeAuditInsert là núm F2 (HOP-DONG §10): gọi ngay trước INSERT audit EXECUTED, sau khi phiếu và sổ lệnh đã INSERT trong cùng
	// giao dịch; trả lỗi thì coi như câu INSERT audit bị PG từ chối. Chỉ tệp _test.go gắn nó (export_test.go): ở đường chạy thật nó nil.
	beforeAuditInsert func() error
}

// NewPGStore dựng kho phiếu trên db (thường là *pgstore.Store).
func NewPGStore(db TxBeginner) *PGStore { return &PGStore{db: db} }

var _ Store = (*PGStore)(nil)

const (
	sqlFindLedger = `SELECT caller_id, command_id, fingerprint, ticket_id FROM commands WHERE caller_id = $1 AND command_id = $2`

	sqlFindOpenTicket = `SELECT id, status FROM discrepancy_tickets
		WHERE subject_key = $1 AND metric = $2 AND day = $3::date AND status = 'OPEN'`

	sqlInsertTicket = `INSERT INTO discrepancy_tickets (tenant_id, id, status, entity_id, subject_key, metric, day,
			left_metric, left_value, left_unit, right_metric, right_value, right_unit, diff, approx, formula, evidence,
			created_by, pack_version, code_version, command_id)
		VALUES (current_setting('eais.tenant_id'), $1, $2, $3, $4, $5, $6::date,
			$7, $8::numeric, $9, $10, $11::numeric, $12, $13::numeric, $14, $15, $16::jsonb,
			$17, $18, $19, $20)`

	sqlInsertLedger = `INSERT INTO commands (tenant_id, caller_id, command_id, fingerprint, ticket_id)
		VALUES (current_setting('eais.tenant_id'), $1, $2, $3, $4)`

	sqlInsertAudit = `INSERT INTO audit_log (tenant_id, outcome, reason_code, reason_detail, caller_id, caller_roles, entity_scope,
			command_id, action, subject_key, metric, day, ticket_id, fingerprint, pack_version, code_version, trace_id)
		VALUES (current_setting('eais.tenant_id'), $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11::date, $12, $13, $14, $15, $16)`

	// Đọc lại: số thập phân qua ::text (giữ nguyên chữ số), ngày qua to_char (không phụ thuộc DateStyle).
	sqlReadTicket = `SELECT id, status, entity_id, subject_key, metric, to_char(day, 'YYYY-MM-DD'),
			left_metric, left_value::text, left_unit, right_metric, right_value::text, right_unit, diff::text, approx, formula,
			evidence::text, created_by, pack_version, code_version, command_id
		FROM discrepancy_tickets WHERE id = $1`

	// reason_detail không đọc được bằng vai ứng dụng (migration 0005, C1): đọc lại so mọi cột khác.
	sqlReadExecuted = `SELECT outcome, reason_code, caller_id, caller_roles, entity_scope, command_id, action,
			subject_key, metric, to_char(day, 'YYYY-MM-DD'), ticket_id, fingerprint, pack_version, code_version, trace_id
		FROM audit_log WHERE outcome = 'EXECUTED' AND caller_id = $1 AND command_id = $2 ORDER BY id`
)

// evidenceRow là dạng lưu của EvidenceSnap trong cột jsonb: tên trường snake_case của lược đồ, không phụ thuộc tên trường Go.
type evidenceRow struct {
	ID         string `json:"id"`
	Kind       string `json:"kind"`
	SourceType string `json:"source_type"`
	SourceRef  string `json:"source_ref"`
	Value      string `json:"value"`
	Unit       string `json:"unit"`
	Day        string `json:"day"`
	Authority  int    `json:"authority"`
}

// requireTenant từ chối khi ngữ cảnh không mang tenant (C1 luôn gắn). BeginTenantTx cũng từ chối; kiểm lại ở đây để hợp đồng của
// Store không phụ thuộc vào cách cài của TxBeginner.
func requireTenant(ctx context.Context) error {
	if _, ok := contracts.TenantFrom(ctx); !ok {
		return contracts.Errorf(contracts.FaultPolicy, "truy vấn không mang tenant trong ngữ cảnh")
	}
	return nil
}

// read chạy fn trong một giao dịch chỉ đọc đã đặt tenant rồi kết thúc nó bằng Rollback (không có gì để commit).
func (s *PGStore) read(ctx context.Context, fn func(tx pgx.Tx) error) error {
	if err := requireTenant(ctx); err != nil {
		return err
	}
	tx, err := s.db.BeginTenantTx(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	return fn(tx)
}

// FindLedger tra sổ lệnh theo (người gọi, command_id) của tenant trong ngữ cảnh.
func (s *PGStore) FindLedger(ctx context.Context, caller, commandID string) (LedgerRecord, bool, error) {
	var out LedgerRecord
	found := false
	err := s.read(ctx, func(tx pgx.Tx) error {
		err := tx.QueryRow(ctx, sqlFindLedger, caller, commandID).Scan(&out.Caller, &out.CommandID, &out.Fingerprint, &out.TicketID)
		if errors.Is(err, pgx.ErrNoRows) {
			return nil
		}
		found = err == nil
		return err
	})
	if err != nil || !found {
		return LedgerRecord{}, false, err
	}
	return out, true, nil
}

// FindOpenTicket tìm phiếu OPEN theo khoá nghiệp vụ (khoá thực thể, chỉ số, ngày). Chỉ mục duy nhất từng phần bảo đảm tối đa một dòng.
func (s *PGStore) FindOpenTicket(ctx context.Context, subjectKey, metric, day string) (TicketRef, bool, error) {
	var out TicketRef
	found := false
	err := s.read(ctx, func(tx pgx.Tx) error {
		err := tx.QueryRow(ctx, sqlFindOpenTicket, subjectKey, metric, day).Scan(&out.ID, &out.Status)
		if errors.Is(err, pgx.ErrNoRows) {
			return nil
		}
		found = err == nil
		return err
	})
	if err != nil || !found {
		return TicketRef{}, false, err
	}
	return out, true, nil
}

// WriteTicket là đơn vị ghi phiếu: MỘT giao dịch, ba INSERT theo thứ tự phiếu, sổ lệnh, audit EXECUTED, rồi COMMIT. Lỗi trả theo ba
// loại ở đầu tệp, luôn SAU khi giao dịch đã được huỷ (nếu chưa gửi COMMIT).
func (s *PGStore) WriteTicket(ctx context.Context, u WriteUnit) error {
	if err := requireTenant(ctx); err != nil {
		return fmt.Errorf("%w: %w", ErrBeforeCommit, err)
	}
	tx, err := s.db.BeginTenantTx(ctx)
	if err != nil {
		return fmt.Errorf("%w: mở giao dịch: %w", ErrBeforeCommit, err) // chưa có giao dịch nào để commit
	}
	sent := false
	defer func() {
		if !sent { // mọi đường thoát trước COMMIT (kể cả panic) đều huỷ giao dịch; pgx đóng kết nối nếu ROLLBACK cũng lỗi
			_ = tx.Rollback(ctx)
		}
	}()
	if err := s.insertUnit(ctx, tx, u); err != nil {
		return classifyStatement(err)
	}
	sent = true // từ đây COMMIT có thể đã rời máy: không huỷ nữa, mọi lỗi do classifyCommit phân loại
	return classifyCommit(tx.Commit(ctx))
}

// insertUnit chạy ba INSERT của đơn vị ghi phiếu trên giao dịch đã mở, theo đúng thứ tự: phiếu, sổ lệnh, audit EXECUTED.
func (s *PGStore) insertUnit(ctx context.Context, tx pgx.Tx, u WriteUnit) error {
	t := u.Ticket
	ev := make([]evidenceRow, 0, len(t.Evidence)) // không nil: nil mã hoá thành null và vi phạm CHECK của cột
	for _, e := range t.Evidence {
		ev = append(ev, evidenceRow{ID: e.ID, Kind: e.Kind, SourceType: e.SourceType, SourceRef: e.SourceRef, Value: e.Value, Unit: e.Unit, Day: e.Day, Authority: e.Authority})
	}
	evJSON, err := json.Marshal(ev)
	if err != nil {
		return fmt.Errorf("mã hoá bằng chứng: %w", err)
	}
	if _, err := tx.Exec(ctx, sqlInsertTicket, t.ID, t.Status, t.EntityID, t.SubjectKey, t.Metric, t.Day,
		t.Left.Metric, t.Left.Value, t.Left.Unit, t.Right.Metric, t.Right.Value, t.Right.Unit, t.Diff, t.Approx, t.Formula, string(evJSON),
		t.CreatedBy, t.PackVersion, t.CodeVersion, t.CommandID); err != nil {
		return fmt.Errorf("INSERT phiếu: %w", err)
	}
	l := u.Ledger
	if _, err := tx.Exec(ctx, sqlInsertLedger, l.Caller, l.CommandID, l.Fingerprint, l.TicketID); err != nil {
		return fmt.Errorf("INSERT sổ lệnh: %w", err)
	}
	if hook := s.beforeAuditInsert; hook != nil {
		if err := hook(); err != nil {
			return fmt.Errorf("INSERT audit: %w", err)
		}
	}
	if err := insertAudit(ctx, tx, u.Audit); err != nil {
		return fmt.Errorf("INSERT audit: %w", err)
	}
	return nil
}

func insertAudit(ctx context.Context, tx pgx.Tx, a Audit) error {
	roles := a.Caller.Roles
	if roles == nil {
		roles = []string{} // nil tới PG là NULL và vi phạm NOT NULL
	}
	_, err := tx.Exec(ctx, sqlInsertAudit, string(a.Outcome), string(a.ReasonCode), a.ReasonDetail, a.Caller.ID, roles, a.Caller.EntityScope,
		a.CommandID, a.Action, a.SubjectKey, a.Metric, a.Day, a.TicketID, a.Fingerprint, a.PackVersion, a.CodeVersion, a.TraceID)
	return err
}

// ReadBack đọc lại bằng giao dịch MỚI từ nơi ghi, KHÔNG dùng RETURNING của lệnh INSERT (HOP-DONG §7.2): phiếu theo mã, sổ lệnh theo khoá,
// audit EXECUTED theo (người gọi, command_id). Không thấy gì thì trả phần rỗng, không lỗi; trả ĐÚNG từng trường đã ghi.
func (s *PGStore) ReadBack(ctx context.Context, caller, commandID, ticketID string) (Stored, error) {
	var st Stored
	err := s.read(ctx, func(tx pgx.Tx) error {
		var t TicketRecord
		var ev string
		err := tx.QueryRow(ctx, sqlReadTicket, ticketID).Scan(&t.ID, &t.Status, &t.EntityID, &t.SubjectKey, &t.Metric, &t.Day,
			&t.Left.Metric, &t.Left.Value, &t.Left.Unit, &t.Right.Metric, &t.Right.Value, &t.Right.Unit, &t.Diff, &t.Approx, &t.Formula,
			&ev, &t.CreatedBy, &t.PackVersion, &t.CodeVersion, &t.CommandID)
		switch {
		case err == nil:
			var rows []evidenceRow
			if err := json.Unmarshal([]byte(ev), &rows); err != nil {
				return fmt.Errorf("giải mã bằng chứng của phiếu %s: %w", ticketID, err)
			}
			for _, r := range rows {
				t.Evidence = append(t.Evidence, EvidenceSnap{ID: r.ID, Kind: r.Kind, SourceType: r.SourceType, SourceRef: r.SourceRef, Value: r.Value, Unit: r.Unit, Day: r.Day, Authority: r.Authority})
			}
			st.Tickets = append(st.Tickets, t)
		case !errors.Is(err, pgx.ErrNoRows):
			return err
		}

		var l LedgerRecord
		switch err := tx.QueryRow(ctx, sqlFindLedger, caller, commandID).Scan(&l.Caller, &l.CommandID, &l.Fingerprint, &l.TicketID); {
		case err == nil:
			st.Ledger = append(st.Ledger, l)
		case !errors.Is(err, pgx.ErrNoRows):
			return err
		}

		rows, err := tx.Query(ctx, sqlReadExecuted, caller, commandID)
		if err != nil {
			return err
		}
		defer rows.Close()
		for rows.Next() {
			var a Audit
			var outcome, code string
			if err := rows.Scan(&outcome, &code, &a.Caller.ID, &a.Caller.Roles, &a.Caller.EntityScope, &a.CommandID, &a.Action,
				&a.SubjectKey, &a.Metric, &a.Day, &a.TicketID, &a.Fingerprint, &a.PackVersion, &a.CodeVersion, &a.TraceID); err != nil {
				return err
			}
			a.Outcome, a.ReasonCode = AuditOutcome(outcome), ReasonCode(code)
			if len(a.Caller.Roles) == 0 {
				a.Caller.Roles = nil // mảng rỗng đọc lại là nil, như Principal không vai
			}
			st.Executed = append(st.Executed, a)
		}
		return rows.Err()
	})
	if err != nil {
		return Stored{}, err
	}
	return st, nil
}

// WriteAudit ghi MỘT dòng audit bằng giao dịch riêng (REFUSED, REPLAYED, VERIFY_FAILED; HOP-DONG §7.3). EXECUTED bị từ chối: nó chỉ được
// nằm trong đơn vị ghi phiếu, cùng giao dịch với phiếu và sổ lệnh (I4).
func (s *PGStore) WriteAudit(ctx context.Context, a Audit) error {
	if err := requireTenant(ctx); err != nil {
		return err
	}
	if a.Outcome == AuditExecuted {
		return errors.New("audit EXECUTED chỉ được ghi trong đơn vị ghi phiếu (WriteTicket), không ghi riêng")
	}
	tx, err := s.db.BeginTenantTx(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }() // sau Commit thành công trả ErrTxClosed, bỏ qua
	if err := insertAudit(ctx, tx, a); err != nil {
		return fmt.Errorf("INSERT audit: %w", err)
	}
	return tx.Commit(ctx)
}

// ---------------------------------------------------------------------------
// Phân loại lỗi của đơn vị ghi (bảng đo U1; xem đầu tệp)
// ---------------------------------------------------------------------------

// sqlstateUniqueViolation là SQLSTATE unique_violation.
const sqlstateUniqueViolation = "23505"

// pgError mở lỗi PG ra khỏi chuỗi lỗi bọc (pgx, nhãn fault, %w của chính gói).
func pgError(err error) (*pgconn.PgError, bool) {
	var pe *pgconn.PgError
	if errors.As(err, &pe) {
		return pe, true
	}
	return nil, false
}

func isUniqueViolation(err error) bool {
	pe, ok := pgError(err)
	return ok && pe.Code == sqlstateUniqueViolation
}

// isStatementError báo err là một lỗi mức ERROR do PG TRẢ LỜI cho một lệnh (giao dịch bị huỷ, không có gì được commit). Chỉ đúng
// mức ERROR (bản không dịch; không có thì bản đã dịch): FATAL / PANIC (backend chết), mức rỗng, mức lạ hay chỉ có bản đã dịch khác
// "ERROR" đều KHÔNG phải — classifyCommit coi chúng là không biết (C7, lệnh ca chiều 23/10). Lỗi không phải của PG cũng không phải.
func isStatementError(err error) bool {
	pe, ok := pgError(err)
	if !ok {
		return false
	}
	sev := pe.SeverityUnlocalized
	if sev == "" {
		sev = pe.Severity
	}
	return sev == "ERROR"
}

// classifyStatement phân loại lỗi của MỘT câu lệnh trong giao dịch, tức trước khi COMMIT được gửi: chắc chắn chưa ghi (giao dịch chưa
// commit không sống sót qua lỗi câu lệnh hay kết nối đứt), trừ vi phạm khoá duy nhất mà pipeline cần đọc lại để phân loại.
func classifyStatement(err error) error {
	if isUniqueViolation(err) {
		return fmt.Errorf("%w: %w", ErrUniqueViolation, err)
	}
	return fmt.Errorf("%w: %w", ErrBeforeCommit, err)
}

// classifyCommit phân loại lỗi của chính lệnh COMMIT. "Chắc chắn chưa ghi" chỉ khi COMMIT chưa rời máy (notSentYet), khi PG trả tag
// ROLLBACK, hoặc khi PG trả một lỗi đúng mức ERROR cho COMMIT (giao dịch bị huỷ, ví dụ ràng buộc hoãn, CancelRequest).
// Mọi lỗi còn lại xảy ra SAU khi COMMIT đã gửi mà không có câu trả lời "đã huỷ" của PG nên không biết: lỗi kết nối, backend bị kết thúc
// (FATAL), mức nghiêm trọng rỗng hay lạ, ngữ cảnh hết hạn khi đang đợi.
func classifyCommit(err error) error {
	switch {
	case err == nil:
		return nil
	case isUniqueViolation(err):
		return fmt.Errorf("%w: %w", ErrUniqueViolation, err)
	case errors.Is(err, pgx.ErrTxCommitRollback), isStatementError(err), notSentYet(err):
		return fmt.Errorf("%w: %w", ErrBeforeCommit, err)
	}
	return fmt.Errorf("%w: %w", ErrCommitUnknown, err)
}

// notSentYet báo err chắc chắn xảy ra TRƯỚC khi COMMIT rời máy: pgconn.SafeToRetry, trừ "conn closed". Đo U1 (ca c4): khi mạng đứt đúng
// lúc chờ câu trả lời của COMMIT, lần đọc đầu thấy lỗi mạng và đánh dấu kết nối đã đóng, rồi pgx báo "conn closed" (connLockError) mang
// SafeToRetry() = true dù COMMIT đã tới máy chủ và đã commit; lỗi gốc (reset, EOF) bị che. Hình dạng này không phân biệt được với một
// COMMIT thật sự chưa gửi, nên coi là không biết. Phần SafeToRetry còn lại (ngữ cảnh đã xong trước khi gửi, ghi được 0 byte) đáng tin.
func notSentYet(err error) bool {
	return pgconn.SafeToRetry(err) && !errors.Is(err, pgconn.ErrConnClosed)
}
