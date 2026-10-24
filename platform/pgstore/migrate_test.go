package pgstore_test

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"io/fs"
	"path"
	"strings"
	"sync"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/luannt2002/enterprise-ai-control-layer/core/contracts"
	"github.com/luannt2002/enterprise-ai-control-layer/db"
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

// F-036 / ADR-0017: migration nhận diện bằng checksum nội dung, không chỉ bằng tên tệp. Trên database riêng của test:
//   - dòng ghi bởi Migrate mang checksum = sha256 của tệp; bảng schema_migrations kiểu cũ (chưa có cột) được thêm cột và điền khi gặp lại;
//   - (a) đã áp mà nội dung tệp đổi ⇒ lỗi, không áp gì;
//   - (b) đã áp dưới tên khác (số tệp đổi trong merge: DB dev từng có 0006_kho_bao_cao_ngay trước khi tệp thành 0009) ⇒ lỗi nêu cả hai tên,
//     KHÔNG chạy lại — cả khi dòng cũ chưa có checksum (nhận theo phần tên sau số);
//   - dòng thừa không trùng tệp nào đang chờ (DB dev thật có 0004/0005/0006_tai_lieu_phien_ban bên cạnh 0007) ⇒ không báo oan.
func TestMigrateDanhTinhBangChecksum(t *testing.T) {
	_, admin, app := open(t)
	fa, _ := freshDB(t, admin, app)
	ctx := context.Background()
	if err := pgstore.Migrate(ctx, fa); err != nil {
		t.Fatal(err)
	}
	conn := connect(t, fa)
	exec := func(q string) {
		t.Helper()
		if _, err := conn.Exec(ctx, q); err != nil {
			t.Fatalf("%s: %v", q, err)
		}
	}
	names, err := fs.Glob(db.Migrations, "migrations/*.sql")
	if err != nil || len(names) == 0 {
		t.Fatalf("không thấy tệp migration: %v %v", names, err)
	}
	want := map[string]string{}
	for _, n := range names {
		b, err := db.Migrations.ReadFile(n)
		if err != nil {
			t.Fatal(err)
		}
		h := sha256.Sum256(b)
		want[path.Base(n)] = hex.EncodeToString(h[:])
	}
	state := func() map[string]string {
		t.Helper()
		rows, err := conn.Query(ctx, "SELECT version, coalesce(checksum, '') FROM schema_migrations")
		if err != nil {
			t.Fatal(err)
		}
		got := map[string]string{}
		var v, c string
		if _, err := pgx.ForEachRow(rows, []any{&v, &c}, func() error { got[v] = c; return nil }); err != nil {
			t.Fatal(err)
		}
		return got
	}
	same := func(what string) {
		t.Helper()
		got := state()
		if len(got) != len(want) {
			t.Fatalf("%s: schema_migrations có %d dòng, muốn %d", what, len(got), len(want))
		}
		for v, c := range want {
			if got[v] != c {
				t.Errorf("%s: checksum của %s = %q, muốn %q", what, v, got[v], c)
			}
		}
	}
	refuse := func(what string, wants ...string) {
		t.Helper()
		before := state()
		err := pgstore.Migrate(ctx, fa)
		for _, w := range wants {
			if err == nil || !strings.Contains(err.Error(), w) {
				t.Errorf("%s: Migrate phải lỗi nêu %q, được %v", what, w, err)
			}
		}
		if err != nil && contracts.FaultOf(err) != contracts.Infra("postgres") {
			t.Errorf("%s: nhãn lỗi = %q", what, contracts.FaultOf(err))
		}
		if after := state(); len(after) != len(before) {
			t.Errorf("%s: Migrate lỗi mà vẫn áp / ghi thêm: %v → %v", what, before, after)
		}
	}
	same("sau Migrate trên DB trống")

	// bảng kiểu cũ (Migrate trước F-036 dựng): chưa có cột checksum ⇒ thêm cột, điền theo tệp hiện tại, không áp lại gì
	exec("ALTER TABLE schema_migrations DROP COLUMN checksum")
	if err := pgstore.Migrate(ctx, fa); err != nil {
		t.Fatalf("Migrate trên schema_migrations kiểu cũ: %v", err)
	}
	same("sau khi điền checksum cho dòng cũ")

	// (a) nội dung đổi sau khi áp
	exec("UPDATE schema_migrations SET checksum = repeat('0', 64) WHERE version = '0003_kho_vet_ben.sql'")
	refuse("checksum lệch", "0003_kho_vet_ben.sql", "nội dung tệp đã đổi")
	exec("UPDATE schema_migrations SET checksum = NULL WHERE version = '0003_kho_vet_ben.sql'")
	if err := pgstore.Migrate(ctx, fa); err != nil {
		t.Fatal(err)
	}
	same("sau khi trả checksum")

	// (b) đã áp dưới tên khác: đổi số (dòng có checksum, và dòng cũ chưa có checksum — nhận theo phần tên sau số) và đổi hẳn tên (chỉ
	// checksum nhận ra)
	for _, c := range []struct{ old, sum string }{
		{"0006_kho_bao_cao_ngay.sql", "checksum"}, {"0006_kho_bao_cao_ngay.sql", "NULL"}, {"0009_bao_cao_cu.sql", "checksum"},
	} {
		exec("UPDATE schema_migrations SET version = '" + c.old + "', checksum = " + c.sum + " WHERE version = '0009_kho_bao_cao_ngay.sql'")
		refuse("đổi tên thành "+c.old+", checksum = "+c.sum, "0009_kho_bao_cao_ngay.sql", c.old, "đổi tên")
		exec("UPDATE schema_migrations SET version = '0009_kho_bao_cao_ngay.sql' WHERE version = '" + c.old + "'")
		if err := pgstore.Migrate(ctx, fa); err != nil {
			t.Fatalf("trả tên %s (checksum = %s): %v", c.old, c.sum, err)
		}
		same("sau khi trả tên")
	}

	// dòng thừa không trùng tệp đang chờ nào: không báo oan
	exec("INSERT INTO schema_migrations (version) VALUES ('0004_tai_lieu_phien_ban.sql')")
	if err := pgstore.Migrate(ctx, fa); err != nil {
		t.Errorf("dòng thừa không đụng tệp nào đang chờ mà Migrate lỗi: %v", err)
	}
}
