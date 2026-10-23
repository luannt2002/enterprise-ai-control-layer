package pgstore

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"github.com/luannt2002/enterprise-ai-control-layer/core/contracts"
)

// readyNeeds là các bảng mà ứng dụng (cmd/eais) đụng tới và quyền vai ứng dụng cần có trên mỗi bảng. Bốn bảng dữ liệu nghiệp vụ
// cần thêm DELETE vì Load xoá rồi ghi lại; ba bảng vết chỉ SELECT + INSERT (vết là bằng chứng, không sửa, không xoá — 0003); ba bảng của phiếu (0004:
// phiếu, sổ lệnh, audit) cũng chỉ SELECT + INSERT (tính chỉ-thêm do quyền bảo đảm, HOP-DONG I8) — thiếu chúng thì lượt lệnh đầu tiên mới hỏng.
// Riêng audit_log từ migration 0005: INSERT mức bảng, SELECT THEO CỘT (selectCols) và reason_detail KHÔNG được đọc (noSelectCols) — lý do thật
// của lần từ chối chỉ vai quản trị đọc (lệnh ca chiều 23/10 C1; soát C1–C7 C17S-03). Ready coi cấp lại quyền đọc cột đó là chưa sẵn sàng.
var readyNeeds = []struct {
	table        string
	privs        []string
	selectCols   []string
	noSelectCols []string
}{
	{"entities", []string{"SELECT", "INSERT", "DELETE"}, nil, nil},
	{"entity_aliases", []string{"SELECT", "INSERT", "DELETE"}, nil, nil},
	{"facts", []string{"SELECT", "INSERT", "DELETE"}, nil, nil},
	{"documents", []string{"SELECT", "INSERT", "DELETE"}, nil, nil},
	{"decision_records", []string{"SELECT", "INSERT"}, nil, nil},
	{"quarantine_records", []string{"SELECT", "INSERT"}, nil, nil},
	{"usage_records", []string{"SELECT", "INSERT"}, nil, nil},
	{"discrepancy_tickets", []string{"SELECT", "INSERT"}, nil, nil},
	{"commands", []string{"SELECT", "INSERT"}, nil, nil},
	{"audit_log", []string{"INSERT"}, []string{"tenant_id", "id", "at", "outcome", "reason_code", "caller_id", "caller_roles", "entity_scope",
		"command_id", "action", "subject_key", "metric", "day", "ticket_id", "fingerprint", "pack_version", "code_version", "trace_id"},
		[]string{"reason_detail"}},
}

// readyColumns là cột do migration thêm vào bảng ĐÃ CÓ: bảng có mặt chưa chứng minh migration đã áp, thiếu cột thì lượt
// RAG / lần nạp đầu tiên mới lỗi.
var readyColumns = []struct{ table, column, migration string }{
	{"documents", "version", "0006"},
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
// Giới hạn đã biết (soát lại M4, M4R-06): has_table_privilege / has_column_privilege chỉ nhìn quyền THỪA KẾ của vai đang đăng nhập. Nếu quản trị
// cấp cho eais_app một vai có quyền đọc reason_detail với INHERIT FALSE (hay đặt eais_app NOINHERIT) mà vẫn cho SET, Ready cho qua dù ứng
// dụng SET ROLE được rồi đọc cột đó. Đó là cấu hình do quản trị chủ động tạo, không phải đường ứng dụng tự mở; chưa kiểm pg_has_role(…, 'SET').
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
		for _, c := range append(slices.Clone(n.selectCols), n.noSelectCols...) {
			var ok bool
			if err := s.pool.QueryRow(ctx, "SELECT has_column_privilege(current_user, $1, $2, 'SELECT')", n.table, c).Scan(&ok); err != nil {
				return contracts.Label(contracts.Infra("postgres"), err)
			}
			switch want := !slices.Contains(n.noSelectCols, c); {
			case want && !ok:
				missing = append(missing, fmt.Sprintf("vai ứng dụng thiếu quyền SELECT trên %s.%s", n.table, c))
			case !want && ok:
				missing = append(missing, fmt.Sprintf("vai ứng dụng ĐỌC ĐƯỢC %s.%s (cột chỉ được ghi — migration 0005)", n.table, c))
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
