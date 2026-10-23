package pgstore_test

import (
	"context"
	"strings"
	"testing"

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
