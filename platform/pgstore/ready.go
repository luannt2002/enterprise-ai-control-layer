package pgstore

import (
	"context"
	"fmt"
	"strings"

	"github.com/luannt2002/enterprise-ai-control-layer/core/contracts"
)

// readyNeeds là các bảng mà ứng dụng (cmd/eais) đụng tới và quyền vai ứng dụng cần có trên mỗi bảng. Bốn bảng dữ liệu nghiệp vụ
// cần thêm DELETE vì Load xoá rồi ghi lại; ba bảng vết chỉ SELECT + INSERT (vết là bằng chứng, không sửa, không xoá — 0003).
var readyNeeds = []struct {
	table string
	privs []string
}{
	{"entities", []string{"SELECT", "INSERT", "DELETE"}},
	{"entity_aliases", []string{"SELECT", "INSERT", "DELETE"}},
	{"facts", []string{"SELECT", "INSERT", "DELETE"}},
	{"documents", []string{"SELECT", "INSERT", "DELETE"}},
	{"decision_records", []string{"SELECT", "INSERT"}},
	{"quarantine_records", []string{"SELECT", "INSERT"}},
	{"usage_records", []string{"SELECT", "INSERT"}},
}

// readyColumns là cột do migration thêm vào bảng ĐÃ CÓ: bảng có mặt chưa chứng minh migration đã áp, thiếu cột thì lượt
// RAG / lần nạp đầu tiên mới lỗi.
var readyColumns = []struct{ table, column, migration string }{
	{"documents", "version", "0005"},
}

// Ping kiểm kết nối còn sống — rẻ, dành cho healthz gọi thường xuyên; Ready kiểm cả schema và quyền (đắt hơn, gọi lúc khởi động).
func (s *Store) Ping(ctx context.Context) error {
	if err := s.pool.Ping(ctx); err != nil {
		return contracts.Label(contracts.Infra("postgres"), err)
	}
	return nil
}

// Ready kiểm Store dùng được cho ứng dụng: kết nối còn sống, mọi bảng ứng dụng cần có mặt (migration đã áp) và vai ứng dụng có đủ
// quyền trên chúng (readyNeeds). Không đặt tenant nên không đọc dòng nào — chỉ hỏi catalog. Lỗi mang nhãn hạ tầng postgres và nói
// rõ thiếu gì ("chạy eais migrate"), để server từ chối khởi động thay vì phục vụ rồi mất vết ở lượt đầu tiên.
func (s *Store) Ready(ctx context.Context) error {
	var missing []string
	for _, n := range readyNeeds {
		var exists bool
		if err := s.pool.QueryRow(ctx, "SELECT to_regclass($1) IS NOT NULL", n.table).Scan(&exists); err != nil {
			return contracts.Label(contracts.Infra("postgres"), err)
		}
		if !exists {
			missing = append(missing, "bảng "+n.table+" chưa có")
			continue
		}
		for _, p := range n.privs {
			var ok bool
			if err := s.pool.QueryRow(ctx, "SELECT has_table_privilege(current_user, $1, $2)", n.table, p).Scan(&ok); err != nil {
				return contracts.Label(contracts.Infra("postgres"), err)
			}
			if !ok {
				missing = append(missing, fmt.Sprintf("vai ứng dụng thiếu quyền %s trên %s", p, n.table))
			}
		}
	}
	for _, c := range readyColumns {
		var ok bool
		if err := s.pool.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM information_schema.columns
			WHERE table_schema = current_schema() AND table_name = $1 AND column_name = $2)`, c.table, c.column).Scan(&ok); err != nil {
			return contracts.Label(contracts.Infra("postgres"), err)
		}
		if !ok {
			missing = append(missing, fmt.Sprintf("cột %s.%s chưa có (migration %s)", c.table, c.column, c.migration))
		}
	}
	if len(missing) > 0 {
		return contracts.Errorf(contracts.Infra("postgres"), "kho chưa sẵn sàng (chạy `eais migrate` bằng vai quản trị): "+strings.Join(missing, "; "))
	}
	return nil
}
