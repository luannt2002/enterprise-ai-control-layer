package pgstore_test

import (
	"os"
	"testing"

	"github.com/luannt2002/enterprise-ai-control-layer/platform/pgstore"
)

// DSN chuẩn là EAIS_PG_*; biến cùng tên tiền tố EACL_PG_* (tên dùng 21/10/2026 trước D-RM-02) chỉ được đọc khi biến EAIS_
// KHÔNG được đặt — đặt EAIS_ (kể cả rỗng) luôn thắng. Không cần PG.
func TestEnvDSNTienToTuongThich(t *testing.T) {
	for _, k := range []string{"EAIS_PG_ADMIN_DSN", "EAIS_PG_DSN", "EACL_PG_ADMIN_DSN", "EACL_PG_DSN"} {
		t.Setenv(k, "")
		os.Unsetenv(k)
	}
	if admin, app := pgstore.EnvDSN(); admin != "" || app != "" {
		t.Errorf("không đặt biến nào: muốn rỗng, được %q / %q", admin, app)
	}
	t.Setenv("EACL_PG_ADMIN_DSN", "quan-tri-cu")
	t.Setenv("EACL_PG_DSN", "ung-dung-cu")
	if admin, app := pgstore.EnvDSN(); admin != "quan-tri-cu" || app != "ung-dung-cu" {
		t.Errorf("chỉ có EACL_PG_*: muốn đọc dự phòng, được %q / %q", admin, app)
	}
	t.Setenv("EAIS_PG_ADMIN_DSN", "quan-tri-moi")
	t.Setenv("EAIS_PG_DSN", "")
	if admin, app := pgstore.EnvDSN(); admin != "quan-tri-moi" || app != "" {
		t.Errorf("EAIS_PG_* đã đặt phải thắng (kể cả đặt rỗng): được %q / %q", admin, app)
	}
}
