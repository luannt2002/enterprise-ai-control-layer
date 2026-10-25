package pgstore_test

import (
	"context"
	"strings"
	"testing"

	"github.com/luannt2002/enterprise-ai-control-layer/platform/pgstore"
)

// Kho chưa áp 0011 (cột facts.note của F-008): Ready phải từ chối lúc khởi động và nêu cột thiếu, thay vì để Load / Facts lỗi giữa lượt.
func TestMigration0011ReadyNeuCotThieu(t *testing.T) {
	_, admin, app := open(t)
	fa, fp := freshDB(t, admin, app)
	ctx := context.Background()
	if err := pgstore.Migrate(ctx, fa); err != nil {
		t.Fatal(err)
	}
	conn := connect(t, fa)
	for _, q := range []string{ // đưa kho về đúng hình dạng trước 0011
		"ALTER TABLE facts DROP COLUMN note",
		"DELETE FROM schema_migrations WHERE version = '0011_ghi_chu_suy_luan.sql'",
	} {
		if _, err := conn.Exec(ctx, q); err != nil {
			t.Fatalf("%s: %v", q, err)
		}
	}
	s, err := pgstore.Open(ctx, fp)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	if err := s.Ready(ctx); err == nil || !strings.Contains(err.Error(), "facts.note") {
		t.Errorf("kho chưa áp 0011: Ready phải từ chối và nêu cột thiếu, được %v", err)
	}
	if err := pgstore.Migrate(ctx, fa); err != nil {
		t.Fatalf("0011 trên kho có sẵn: %v", err)
	}
	if err := s.Ready(ctx); err != nil {
		t.Errorf("sau 0011 Ready phải đạt: %v", err)
	}
}
