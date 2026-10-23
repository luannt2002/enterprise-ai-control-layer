package pgstore_test

import (
	"context"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/luannt2002/enterprise-ai-control-layer/core/contracts"
	"github.com/luannt2002/enterprise-ai-control-layer/platform/pgstore"
)

// K2S-04: Ready biết ba bảng của phiếu (migration 0004). Thiếu migration thì kho CHƯA sẵn sàng và lỗi nêu từng bảng thiếu cùng lệnh migrate; thu một quyền
// SELECT hay INSERT của vai ứng dụng trên một trong ba bảng thì lỗi nêu đúng quyền và đúng bảng; trả quyền lại thì sẵn sàng. Trước đây Ready chỉ biết bảy bảng của
// 0001…0003, nên ứng dụng khởi động được trên kho chưa áp 0004 rồi mới hỏng ở lượt lệnh đầu tiên.
func TestReadyBaBangPhieu(t *testing.T) {
	_, admin, app := open(t)
	fa, fp := freshDB(t, admin, app)
	ctx := context.Background()
	if err := pgstore.Migrate(ctx, fa); err != nil {
		t.Fatal(err)
	}
	s, err := pgstore.Open(ctx, fp)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	if err := s.Ready(ctx); err != nil {
		t.Fatalf("sau Migrate đủ: %v", err)
	}
	conn := connect(t, fa)

	tables := []string{"discrepancy_tickets", "commands", "audit_log"}
	for _, table := range tables {
		for _, priv := range []string{"SELECT", "INSERT"} {
			if table == "audit_log" && priv == "SELECT" {
				continue // audit_log không có SELECT mức bảng từ 0005 (C1): kiểm theo cột ở dưới
			}
			if _, err := conn.Exec(ctx, "REVOKE "+priv+" ON "+table+" FROM eais_app"); err != nil {
				t.Fatal(err)
			}
			err := s.Ready(ctx)
			if want := "thiếu quyền " + priv + " trên " + table; err == nil || !strings.Contains(err.Error(), want) {
				t.Errorf("thu %s trên %s: muốn lỗi nêu %q, được %v", priv, table, want, err)
			}
			if _, err := conn.Exec(ctx, "GRANT "+priv+" ON "+table+" TO eais_app"); err != nil {
				t.Fatal(err)
			}
			if err := s.Ready(ctx); err != nil {
				t.Errorf("trả %s trên %s rồi mà vẫn chưa sẵn sàng: %v", priv, table, err)
			}
		}
	}

	// audit_log theo cột (0005, C1; soát C1–C7 C17S-03): thu SELECT TỪNG cột cần đọc ⇒ chưa sẵn sàng và nêu đúng cột, trả lại ⇒ sẵn sàng. Danh
	// sách cột lấy từ CATALOG (mọi cột của bảng trừ reason_detail), không chép từ readyNeeds: danh sách cột của Ready lệch khỏi lược đồ thật thì
	// test đỏ (soát lại M4, M4R-05: bỏ trace_id, tenant_id hay caller_roles khỏi readyNeeds từng không làm test nào đỏ).
	rows, err := conn.Query(ctx, `SELECT column_name FROM information_schema.columns
		WHERE table_schema = current_schema() AND table_name = 'audit_log' AND column_name <> 'reason_detail' ORDER BY ordinal_position`)
	if err != nil {
		t.Fatal(err)
	}
	cols, err := pgx.CollectRows(rows, pgx.RowTo[string])
	if err != nil {
		t.Fatal(err)
	}
	if len(cols) < 18 {
		t.Fatalf("audit_log chỉ có %d cột ngoài reason_detail: %v", len(cols), cols)
	}
	for _, col := range cols {
		if _, err := conn.Exec(ctx, "REVOKE SELECT ("+col+") ON audit_log FROM eais_app"); err != nil {
			t.Fatal(err)
		}
		if err := s.Ready(ctx); err == nil || !strings.Contains(err.Error(), "thiếu quyền SELECT trên audit_log."+col) {
			t.Errorf("thu SELECT (%s) trên audit_log: muốn lỗi nêu cột đó, được %v", col, err)
		}
		if _, err := conn.Exec(ctx, "GRANT SELECT ("+col+") ON audit_log TO eais_app"); err != nil {
			t.Fatal(err)
		}
		if err := s.Ready(ctx); err != nil {
			t.Errorf("trả SELECT (%s) rồi mà vẫn chưa sẵn sàng: %v", col, err)
		}
	}
	// cấp quyền đọc reason_detail (theo cột hay mức bảng) ⇒ chưa sẵn sàng (cột chỉ được ghi), thu lại ⇒ sẵn sàng
	for _, c := range []struct{ do, undo, want string }{
		{"GRANT SELECT (reason_detail) ON audit_log TO eais_app", "REVOKE SELECT (reason_detail) ON audit_log FROM eais_app", "ĐỌC ĐƯỢC audit_log.reason_detail"},
		{"GRANT SELECT ON audit_log TO eais_app", "REVOKE SELECT ON audit_log FROM eais_app; GRANT SELECT (tenant_id, id, at, outcome, reason_code, caller_id, caller_roles, entity_scope, command_id, action, subject_key, metric, day, ticket_id, fingerprint, pack_version, code_version, trace_id) ON audit_log TO eais_app", "ĐỌC ĐƯỢC audit_log.reason_detail"},
	} {
		if _, err := conn.Exec(ctx, c.do); err != nil {
			t.Fatal(err)
		}
		if err := s.Ready(ctx); err == nil || !strings.Contains(err.Error(), c.want) {
			t.Errorf("%s: muốn lỗi nêu %q, được %v", c.do, c.want, err)
		}
		if _, err := conn.Exec(ctx, c.undo); err != nil {
			t.Fatal(err)
		}
		if err := s.Ready(ctx); err != nil {
			t.Errorf("%s rồi %s mà vẫn chưa sẵn sàng: %v", c.do, c.undo, err)
		}
	}

	// thiếu migration 0004: ba bảng chưa có (xoá để giả lập kho chỉ mới áp tới 0003)
	if _, err := conn.Exec(ctx, "DROP TABLE commands, discrepancy_tickets, audit_log"); err != nil {
		t.Fatal(err)
	}
	err = s.Ready(ctx)
	if err == nil || !strings.Contains(err.Error(), "eais migrate") {
		t.Fatalf("thiếu migration 0004: muốn lỗi nêu lệnh migrate, được %v", err)
	}
	for _, table := range tables {
		if !strings.Contains(err.Error(), "bảng "+table+" chưa có") {
			t.Errorf("lỗi thiếu %q: %v", "bảng "+table+" chưa có", err)
		}
	}
	if got := contracts.FaultOf(err); got != contracts.Infra("postgres") {
		t.Errorf("nhãn lỗi = %q, muốn %q", got, contracts.Infra("postgres"))
	}
}
