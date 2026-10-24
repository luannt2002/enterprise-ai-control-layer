package pgstore_test

import (
	"context"
	"strings"
	"sync"
	"testing"

	"github.com/luannt2002/enterprise-ai-control-layer/platform/pgstore"
)

// Nhiều tiến trình Migrate CÙNG LÚC trên database trống (go test chạy song song nhiều gói đều gọi Migrate; nhiều bản app cùng khởi
// động) phải xếp hàng, không đua nhau CREATE TABLE / ghi schema_migrations: mọi lời gọi thành công, mỗi tệp được áp đúng một lần.
func TestMigrateDongThoiTrenDBTrong(t *testing.T) {
	_, admin, app := open(t)
	const rounds, workers = 3, 12
	for r := 0; r < rounds; r++ {
		fa, _ := freshDB(t, admin, app)
		var wg sync.WaitGroup
		errs := make([]error, workers)
		for i := 0; i < workers; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				errs[i] = pgstore.Migrate(context.Background(), fa)
			}()
		}
		wg.Wait()
		for i, err := range errs {
			if err != nil {
				t.Fatalf("vòng %d, Migrate đồng thời thứ %d: %v", r, i, err)
			}
		}
		conn := connect(t, fa)
		rows, err := conn.Query(context.Background(), "SELECT version FROM schema_migrations ORDER BY version")
		if err != nil {
			t.Fatal(err)
		}
		var got []string
		for rows.Next() {
			var v string
			if err := rows.Scan(&v); err != nil {
				t.Fatal(err)
			}
			got = append(got, v)
		}
		rows.Close()
		if want := "0001_win.sql,0002_dinh_danh_eais.sql,0003_kho_vet_ben.sql,0004_phieu_chenh_lech.sql,0005_audit_va_tenant.sql,0006_so_nap_du_lieu.sql,0007_tai_lieu_phien_ban.sql,0008_vector_tai_lieu.sql,0009_kho_bao_cao_ngay.sql"; strings.Join(got, ",") != want {
			t.Fatalf("vòng %d: schema_migrations = %v, muốn %s", r, got, want)
		}
	}
}
