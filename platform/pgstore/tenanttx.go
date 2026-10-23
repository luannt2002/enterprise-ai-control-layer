package pgstore

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/luannt2002/enterprise-ai-control-layer/core/contracts"
)

// BeginTenantTx mở MỘT giao dịch trên pool và đặt GUC eais.tenant_id (SET LOCAL: hết giao dịch là hết) từ tenant trong ngữ cảnh,
// rồi trả giao dịch cho bên gọi tự Commit hoặc Rollback. Không có tenant trong ngữ cảnh thì từ chối bằng lỗi chính sách, không
// chạm DB — cùng luật với mọi hàm khác của Store.
//
// Khác inTenant ở đúng một chỗ, và đó là lý do hàm này tồn tại: inTenant tự Commit rồi gắn nhãn cho MỌI lỗi, kể cả lỗi của lệnh COMMIT,
// theo việc thử lại có ích không (classifyPGError), nên bên gọi không phân biệt được "chắc chắn chưa ghi" với "không biết đã ghi hay
// chưa". Ở đây lỗi COMMIT,
// lỗi từng câu lệnh và lỗi huỷ giao dịch đến nguyên vẹn (kể cả *pgconn.PgError và tính SafeToRetry của pgx) để kho phiếu tự phân
// loại (execution/commands/pgticket.go; HOP-DONG §7.1, §15 U1). Chỉ lỗi mở giao dịch và lỗi đặt tenant mang nhãn hạ tầng, vì tới lúc
// đó chưa có gì được ghi; nhãn bọc lỗi gốc nên errors.As / errors.Is vẫn tới được nó.
//
// Bên gọi PHẢI kết thúc giao dịch (Commit hoặc Rollback), nếu không kết nối bị giữ lại khỏi pool.
func (s *Store) BeginTenantTx(ctx context.Context) (pgx.Tx, error) {
	tenant, err := tenantOf(ctx)
	if err != nil {
		return nil, err
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, contracts.Label(contracts.Infra("postgres"), err)
	}
	if _, err := tx.Exec(ctx, "SELECT set_config('eais.tenant_id', $1, true)", string(tenant)); err != nil {
		_ = tx.Rollback(ctx) // lỗi đặt tenant: chưa có câu lệnh nào của bên gọi; pgx tự đóng kết nối nếu Rollback cũng lỗi
		return nil, contracts.Label(contracts.Infra("postgres"), err)
	}
	return tx, nil
}
