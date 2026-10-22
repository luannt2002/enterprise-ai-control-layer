package pgstore_test

import (
	"context"
	"strings"
	"testing"

	"github.com/luannt2002/enterprise-ai-control-layer/core/contracts"
	"github.com/luannt2002/enterprise-ai-control-layer/platform/pgstore"
)

// CA ÂM + CHO PHÉP của Ready: database trống (chưa migrate) thì kho CHƯA sẵn sàng và lỗi nói rõ thiếu bảng, mang nhãn hạ tầng; sau
// Migrate thì sẵn sàng; thu một quyền cần thiết của vai ứng dụng thì lại chưa sẵn sàng và lỗi nêu đúng quyền.
func TestReadyKhoSanSang(t *testing.T) {
	_, admin, app := open(t)
	fa, fp := freshDB(t, admin, app)
	ctx := context.Background()

	// chưa migrate: Open thành công (vai eais_app có sẵn ở cụm) nhưng bảng chưa có
	s, err := pgstore.Open(ctx, fp)
	if err != nil {
		t.Fatalf("Open trên DB trống: %v", err)
	}
	defer s.Close()
	err = s.Ready(ctx)
	if err == nil || !strings.Contains(err.Error(), "bảng decision_records chưa có") || !strings.Contains(err.Error(), "eais migrate") {
		t.Fatalf("DB chưa migrate: muốn lỗi nêu bảng thiếu và lệnh migrate, được %v", err)
	}
	if got := contracts.FaultOf(err); got != contracts.Infra("postgres") {
		t.Errorf("nhãn lỗi = %q, muốn %q", got, contracts.Infra("postgres"))
	}

	if err := pgstore.Migrate(ctx, fa); err != nil {
		t.Fatal(err)
	}
	if err := s.Ready(ctx); err != nil {
		t.Fatalf("sau Migrate: %v", err)
	}

	// thu quyền INSERT của vai ứng dụng trên bảng vết: kho không còn ghi được vết ⇒ KHÔNG sẵn sàng
	conn := connect(t, fa)
	if _, err := conn.Exec(ctx, "REVOKE INSERT ON decision_records FROM eais_app"); err != nil {
		t.Fatal(err)
	}
	err = s.Ready(ctx)
	if err == nil || !strings.Contains(err.Error(), "thiếu quyền INSERT trên decision_records") {
		t.Fatalf("thu INSERT: muốn lỗi nêu quyền thiếu, được %v", err)
	}
	if _, err := conn.Exec(ctx, "GRANT INSERT ON decision_records TO eais_app"); err != nil {
		t.Fatal(err)
	}
	if err := s.Ready(ctx); err != nil {
		t.Fatalf("trả quyền rồi mà vẫn chưa sẵn sàng: %v", err)
	}
}

// Pool đã đóng thì Ready và Ping báo lỗi (không panic, không treo): healthz dựa vào Ping để trả 503.
func TestReadyPoolDong(t *testing.T) {
	_, _, app := open(t)
	s, err := pgstore.Open(context.Background(), app)
	if err != nil {
		t.Fatal(err)
	}
	s.Close()
	if err := s.Ready(context.Background()); err == nil {
		t.Fatal("pool đã đóng mà Ready không báo lỗi")
	}
	if err := s.Ping(context.Background()); err == nil || contracts.FaultOf(err) != contracts.Infra("postgres") {
		t.Fatalf("pool đã đóng: Ping muốn lỗi nhãn hạ tầng, được %v", err)
	}
}
