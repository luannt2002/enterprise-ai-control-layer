package reports

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
)

// KHUNG TẠM của commit đỏ-trước (kho PostgreSQL, bước 2 và 3 trên PG). Bị thay bằng cài đặt thật ở commit kế tiếp.

var errPGNotImplemented = errors.New("kho PG chưa cài")

// TxBeginner là phần của platform/pgstore (Store.BeginTenantTx) mà kho PG cần.
type TxBeginner interface{}

// PG là kho bản ghi báo cáo trên PostgreSQL.
type PG struct {
	afterSlotRead func()
	hook          func(ctx context.Context, tx pgx.Tx, op string) error
}

// NewPG dựng kho PG trên db.
func NewPG(db TxBeginner, opt Options) *PG { return &PG{} }

var _ Store = (*PG)(nil)

func (s *PG) Deliver(ctx context.Context, batchID string, ordinal int, it Item) (DeliveryEntry, error) {
	return DeliveryEntry{}, errPGNotImplemented
}
func (s *PG) DeliverBatch(ctx context.Context, items []Item) (Receipt, error) {
	return Receipt{}, errPGNotImplemented
}
func (s *PG) Decide(ctx context.Context, cmd Command) (Record, error) {
	return Record{}, errPGNotImplemented
}
func (s *PG) Retract(ctx context.Context, key, actor, why string) error { return errPGNotImplemented }
func (s *PG) Amend(ctx context.Context, key string, it Item, actor, why string) error {
	return errPGNotImplemented
}
func (s *PG) Record(ctx context.Context, key string) (Record, bool, error) {
	return Record{}, false, errPGNotImplemented
}
func (s *PG) Records(ctx context.Context) ([]Record, error) { return nil, errPGNotImplemented }
func (s *PG) Ledger(ctx context.Context, batchID string) ([]DeliveryEntry, error) {
	return nil, errPGNotImplemented
}
func (s *PG) History(ctx context.Context, key string) ([]HistoryEntry, error) {
	return nil, errPGNotImplemented
}
func (s *PG) Receipt(ctx context.Context, batchID string) (Receipt, error) {
	return Receipt{}, errPGNotImplemented
}
func (s *PG) Totals(ctx context.Context) ([]Total, error) { return nil, errPGNotImplemented }
