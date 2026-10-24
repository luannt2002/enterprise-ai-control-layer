package reports

// Kho PostgreSQL của bước 2 và 3 (pg.go, migration kho_bao_cao_ngay). Cùng bộ test của kho bộ nhớ chạy trên PG (forEachStore, đăng ký ở init), cộng các test chỉ PG có nghĩa: lược đồ,
// RLS, quyền chỉ-thêm, ràng buộc, hai tenant, nhiều lô đồng thời, lỗi giữa giao dịch, sống sót qua khởi động lại, và so khớp với kho bộ nhớ trên chuỗi sự kiện ngẫu nhiên.
// Thiếu EAIS_PG_ADMIN_DSN / EAIS_PG_DSN thì các test cần PG bỏ qua (SKIP); cổng scripts/kiem-skip-test.py coi SKIP là ĐỎ nên cổng phải chạy CÓ PG.

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"net/url"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/luannt2002/enterprise-ai-control-layer/core/contracts"
	"github.com/luannt2002/enterprise-ai-control-layer/db"
	"github.com/luannt2002/enterprise-ai-control-layer/platform/pgstore"
)

var (
	pgOnce  sync.Once
	pgStore *pgstore.Store
	pgAdmin *pgxpool.Pool
	pgErr   error
)

func withPoolMax(dsn string, n int) string {
	u, err := url.Parse(dsn)
	if err != nil || u.User == nil {
		return dsn
	}
	q := u.Query()
	q.Set("pool_max_conns", strconv.Itoa(n))
	u.RawQuery = q.Encode()
	return u.String()
}

// pgSetup áp migration một lần cho cả bộ test, mở Store vai ứng dụng và một kết nối vai quản trị (đếm, kiểm lược đồ).
func pgSetup(t *testing.T) (*pgstore.Store, *pgxpool.Pool) {
	t.Helper()
	admin, app := pgstore.EnvDSN()
	if admin == "" || app == "" {
		t.Skip("chưa có EAIS_PG_ADMIN_DSN / EAIS_PG_DSN — xem deploy/docker-compose.yml")
	}
	pgOnce.Do(func() {
		ctx := context.Background()
		if pgErr = pgstore.Migrate(ctx, admin); pgErr != nil {
			return
		}
		if pgStore, pgErr = pgstore.Open(ctx, withPoolMax(app, 16)); pgErr != nil {
			return
		}
		pgAdmin, pgErr = pgxpool.New(ctx, withPoolMax(admin, 4))
	})
	if pgErr != nil {
		t.Fatalf("dựng PG: %v", pgErr)
	}
	return pgStore, pgAdmin
}

func init() {
	pgStoreMakers = append(pgStoreMakers, func(t *testing.T) storeEnv {
		st, _ := pgSetup(t)
		return storeEnv{name: "pg", st: NewPG(st, testOptions())}
	})
}

func (s *PG) setAfterSlotRead(f func()) { s.afterSlotRead = f }

var reportTables = []string{"report_records", "report_versions", "report_history", "delivery_ledger"}

// schemaProblems đòi trạng thái ĐÍCH của migration và trả các chỗ lệch (rỗng = đúng): bốn bảng có đủ, RLS ENABLE + FORCE, đúng MỘT chính sách tenant_isolation, vai ứng dụng chỉ
// có quyền cần thiết (sổ, lịch sử, phiên bản: SELECT + INSERT; bản ghi thêm UPDATE theo cột, trừ tenant_id và unit_key), không bỏ qua được RLS.
func schemaProblems(ctx context.Context, admin *pgxpool.Pool) []string {
	var bad []string
	scalar := func(q string, args ...any) bool {
		var v bool
		if err := admin.QueryRow(ctx, q, args...).Scan(&v); err != nil {
			bad = append(bad, "truy vấn lược đồ lỗi: "+err.Error())
			return false
		}
		return v
	}
	for _, tb := range reportTables {
		var rls, force bool
		if err := admin.QueryRow(ctx, `SELECT relrowsecurity, relforcerowsecurity FROM pg_class WHERE oid = $1::regclass`, tb).Scan(&rls, &force); err != nil {
			bad = append(bad, tb+": không có bảng ("+err.Error()+")")
			continue
		}
		if !rls || !force {
			bad = append(bad, fmt.Sprintf("%s: RLS bật = %v, ép (FORCE) = %v", tb, rls, force))
		}
		var pols, named int
		_ = admin.QueryRow(ctx, `SELECT count(*), count(*) FILTER (WHERE policyname = 'tenant_isolation') FROM pg_policies WHERE schemaname = current_schema() AND tablename = $1`, tb).Scan(&pols, &named)
		if pols != 1 || named != 1 {
			bad = append(bad, fmt.Sprintf("%s: %d chính sách, %d tên tenant_isolation (muốn đúng 1)", tb, pols, named))
		}
		for _, p := range []string{"SELECT", "INSERT"} {
			if !scalar(`SELECT has_table_privilege('eais_app', $1::regclass, $2)`, tb, p) {
				bad = append(bad, tb+": eais_app thiếu "+p)
			}
		}
		for _, p := range []string{"UPDATE", "DELETE", "TRUNCATE", "REFERENCES", "TRIGGER"} {
			if scalar(`SELECT has_table_privilege('eais_app', $1::regclass, $2)`, tb, p) {
				bad = append(bad, tb+": eais_app có quyền "+p+" mức bảng (thừa)")
			}
		}
		anyUpd := scalar(`SELECT has_any_column_privilege('eais_app', $1::regclass, 'UPDATE')`, tb)
		if tb == "report_records" && !anyUpd {
			bad = append(bad, tb+": eais_app thiếu UPDATE theo cột (trạng thái bản ghi đổi theo quyết định)")
		}
		if tb != "report_records" && anyUpd {
			bad = append(bad, tb+": chỉ-thêm mà eais_app có UPDATE một cột")
		}
	}
	for _, col := range []string{"tenant_id", "unit_key"} {
		if scalar(`SELECT has_column_privilege('eais_app', 'report_records', $1, 'UPDATE')`, col) {
			bad = append(bad, "report_records."+col+": eais_app sửa được cột khoá")
		}
	}
	if scalar(`SELECT rolsuper OR rolbypassrls FROM pg_roles WHERE rolname = 'eais_app'`) {
		bad = append(bad, "eais_app vượt được RLS")
	}
	return bad
}

func TestPGLuocDo_RLSVaQuyenChiThem(t *testing.T) {
	_, admin := pgSetup(t)
	if bad := schemaProblems(context.Background(), admin); len(bad) > 0 {
		t.Fatalf("lược đồ lệch trạng thái đích:\n  %s", strings.Join(bad, "\n  "))
	}
}

func migrationSQL(t *testing.T) string {
	t.Helper()
	names, err := fs.Glob(db.Migrations, "migrations/*kho_bao_cao_ngay.sql")
	if err != nil || len(names) != 1 {
		t.Fatalf("muốn đúng một migration *kho_bao_cao_ngay.sql, được %v (lỗi %v)", names, err)
	}
	b, err := db.Migrations.ReadFile(names[0])
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

// Migration chạy lại được, và chạy lại ĐƯA VỀ trạng thái đích dù DB đã trôi (FORCE bị tắt, chính sách bị xoá, quyền cấp thêm): bộ kiểm lược đồ THẤY từng kiểu trôi (ca âm của chính bộ kiểm).
func TestPGMigration_ChayLaiHoiTuVeTrangThaiDich(t *testing.T) {
	_, admin := pgSetup(t)
	ctx := context.Background()
	sql := migrationSQL(t)
	drifts := []string{
		`ALTER TABLE report_records NO FORCE ROW LEVEL SECURITY`,
		`DROP POLICY tenant_isolation ON delivery_ledger`,
		`GRANT DELETE ON report_history TO eais_app`,
		`GRANT UPDATE ON delivery_ledger TO eais_app`,
		`GRANT UPDATE (unit_key) ON report_records TO eais_app`,
		`ALTER TABLE report_versions DISABLE ROW LEVEL SECURITY`,
	}
	for _, d := range drifts {
		if _, err := admin.Exec(ctx, d); err != nil {
			t.Fatalf("%s: %v", d, err)
		}
		if bad := schemaProblems(ctx, admin); len(bad) == 0 {
			t.Errorf("bộ kiểm lược đồ KHÔNG thấy kiểu trôi %q", d)
		}
		if _, err := admin.Exec(ctx, sql); err != nil {
			t.Fatalf("chạy lại migration sau %q: %v", d, err)
		}
		if bad := schemaProblems(ctx, admin); len(bad) > 0 {
			t.Errorf("chạy lại migration không đưa về trạng thái đích sau %q: %v", d, bad)
		}
	}
}

func sqlState(err error) string {
	var pe *pgconn.PgError
	if errors.As(err, &pe) {
		return pe.Code
	}
	return ""
}

// rawExec chạy MỘT câu lệnh bằng vai ứng dụng trong một giao dịch đã đặt tenant, rồi huỷ giao dịch (không để lại gì); trả lỗi của câu lệnh.
func rawExec(t *testing.T, st *pgstore.Store, ctx context.Context, sql string, args ...any) error {
	t.Helper()
	tx, err := st.BeginTenantTx(ctx)
	if err != nil {
		t.Fatalf("BeginTenantTx: %v", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	_, err = tx.Exec(ctx, sql, args...)
	return err
}

// RLS: hai tenant cùng khoá không rò sang nhau; vai ứng dụng không ghi được dòng của tenant khác, không sửa sổ, không xoá gì, không sửa cột khoá.
func TestPGRLS_HaiTenantKhongRoVaQuyenChiThem(t *testing.T) {
	st, admin := pgSetup(t)
	pg := NewPG(st, testOptions())
	ctxA, tenA := newCtx("rlsa")
	ctxB, tenB := newCtx("rlsb")
	mustBatch(t, ctxA, pg, mk("chung", 0, withValue("1111")), mk("held", 300, heldFor(ReasonAmbiguousEntity)))
	mustBatch(t, ctxB, pg, mk("chung", 0, withValue("2222")))
	for _, tb := range reportTables {
		var seenOther int
		tx, err := st.BeginTenantTx(ctxA)
		if err != nil {
			t.Fatal(err)
		}
		if err := tx.QueryRow(ctxA, "SELECT count(*) FROM "+tb+" WHERE tenant_id = $1", tenB).Scan(&seenOther); err != nil {
			t.Fatalf("%s: %v", tb, err)
		}
		_ = tx.Rollback(ctxA)
		if seenOther != 0 {
			t.Errorf("%s: tenant A thấy %d dòng của tenant B", tb, seenOther)
		}
	}
	if got := mustRecord(t, ctxA, pg, "chung").Value.Number; got != "1111" {
		t.Errorf("tenant A thấy %s, muốn 1111", got)
	}
	insertOther := `INSERT INTO report_records (tenant_id, unit_key, kind, raw, disposition, fingerprint, content_sha256, version)
		VALUES ($1, 'lau', 'OCCURRED', ''::bytea, 'RETRACTED', 'f', 'c', 1)`
	if err := rawExec(t, st, ctxA, insertOther, tenB); sqlState(err) != "42501" {
		t.Errorf("ghi dòng mang tenant khác: muốn 42501 (RLS WITH CHECK), được %v", err)
	}
	if err := rawExec(t, st, ctxA, insertOther, tenA); err != nil {
		t.Errorf("ghi dòng của chính tenant: %v", err)
	}
	for name, sql := range map[string]string{
		"sửa sổ lần giao":        `UPDATE delivery_ledger SET detail = 'x'`,
		"xoá sổ lần giao":        `DELETE FROM delivery_ledger`,
		"sửa lịch sử":            `UPDATE report_history SET detail = 'x'`,
		"xoá phiên bản":          `DELETE FROM report_versions`,
		"xoá bản ghi":            `DELETE FROM report_records`,
		"sửa cột khoá":           `UPDATE report_records SET unit_key = 'z'`,
		"sửa tenant của bản ghi": `UPDATE report_records SET tenant_id = 'khac'`,
		"TRUNCATE":               `TRUNCATE report_records`,
	} {
		if err := rawExec(t, st, ctxA, sql); sqlState(err) != "42501" {
			t.Errorf("%s: muốn 42501 (thiếu quyền), được %v", name, err)
		}
	}
	// GUC tenant chưa đặt: đóng mặc định (0 dòng) cả với kết nối ứng dụng trần
	var n int
	if err := admin.QueryRow(context.Background(), `SELECT count(*) FROM report_records WHERE tenant_id = $1`, tenA).Scan(&n); err != nil || n < 2 {
		t.Errorf("quản trị đếm %d bản ghi của A (lỗi %v), muốn ≥ 2", n, err)
	}
}

// Ràng buộc ở lược đồ chặn bản ghi sai dù một đường ghi khác bỏ qua tầng Go (đóng mặc định): HELD thiếu người nhận, ACCEPTED thiếu trường, mã ngoài danh sách đóng, DUPLICATE không mã…
func TestPGRangBuoc_LuocDoChanBanGhiSai(t *testing.T) {
	st, _ := pgSetup(t)
	pg := NewPG(st, testOptions())
	ctx, _ := newCtx("check")
	mustDeliver(t, ctx, pg, "b", 1, mk("co", 0))
	rec := func(disposition, reason, recipient, entity string) string {
		return fmt.Sprintf(`INSERT INTO report_records (tenant_id, unit_key, kind, raw, disposition, reason, detail, recipient, entity, occurred_on, value_number, value_unit, fingerprint, content_sha256, version)
			VALUES (current_setting('eais.tenant_id'), 'x', 'OCCURRED', ''::bytea, '%s', '%s', 'd', '%s', '%s', '2026-10-19', '1', 'L', 'f', 'c', 1)`, disposition, reason, recipient, entity)
	}
	led := func(outcome, reason, detail, ref string) string {
		return fmt.Sprintf(`INSERT INTO delivery_ledger (tenant_id, batch_id, ordinal, unit_key, outcome, reason, detail, record_ref, recorded_at)
			VALUES (current_setting('eais.tenant_id'), 'bx', 1, 'u', '%s', '%s', '%s', '%s', now())`, outcome, reason, detail, ref)
	}
	cases := map[string]struct{ sql, want string }{
		"HELD thiếu người nhận":              {rec("HELD", "ambiguous_entity", "", "E1"), "23514"},
		"ACCEPTED thiếu thực thể":            {rec("ACCEPTED", "", "", ""), "23514"},
		"ACCEPTED mang lý do":                {rec("ACCEPTED", "unparseable", "", "E1"), "23514"},
		"mã lý do ngoài danh sách đóng":      {rec("HELD", "tu_bia_ra", "pm", "E1"), "23514"},
		"HELD mang mã của QUARANTINED":       {rec("HELD", "unparseable", "pm", "E1"), "23514"},
		"bản ghi mang mã DUPLICATE":          {rec("HELD", "duplicate_key", "pm", "E1"), "23514"},
		"trạng thái lạ":                      {rec("DONE", "", "", "E1"), "23514"},
		"sổ: DUPLICATE không mã lý do":       {led("DUPLICATE", "", "d", "co"), "23514"},
		"sổ: ACCEPTED mang mã lý do":         {led("ACCEPTED", "unparseable", "", "co"), "23514"},
		"sổ: HELD thiếu chi tiết":            {led("HELD", "ambiguous_entity", "", "co"), "23514"},
		"sổ: kết cục ngoài bốn giá trị":      {led("LA", "", "", "co"), "23514"},
		"sổ: trỏ bản ghi không có thật (FK)": {led("DUPLICATE", "duplicate_key", "d", "khong-co"), "23503"},
	}
	for name, c := range cases {
		if err := rawExec(t, st, ctx, c.sql); sqlState(err) != c.want {
			t.Errorf("%s: muốn %s, được %v", name, c.want, err)
		}
	}
	if err := rawExec(t, st, ctx, led("DUPLICATE", "duplicate_key", "d", "co")); err != nil {
		t.Errorf("dòng sổ hợp lệ bị từ chối: %v", err)
	}
	// tenant rỗng bị chặn ở LƯỢC ĐỒ cho cả bốn bảng, kể cả với vai quản trị (đóng mặc định; như 0005 cho ba bảng của phiếu)
	_, admin := pgSetup(t)
	for _, q := range []string{
		`INSERT INTO report_records (tenant_id, unit_key, kind, raw, disposition, fingerprint, content_sha256, version) VALUES ('', 'x', 'OCCURRED', ''::bytea, 'RETRACTED', 'f', 'c', 1)`,
		`INSERT INTO delivery_ledger (tenant_id, batch_id, ordinal, unit_key, outcome, record_ref, recorded_at) VALUES ('', 'b', 1, 'u', 'ACCEPTED', 'u', now())`,
	} {
		if _, err := admin.Exec(context.Background(), q); sqlState(err) != "23514" {
			t.Errorf("tenant rỗng: muốn 23514 (CHECK), được %v", err)
		}
	}
}

func adminCount(t *testing.T, admin *pgxpool.Pool, table, tenant string) int {
	t.Helper()
	var n int
	if err := admin.QueryRow(context.Background(), "SELECT count(*) FROM "+table+" WHERE tenant_id = $1", tenant).Scan(&n); err != nil {
		t.Fatalf("đếm %s: %v", table, err)
	}
	return n
}

// Nhiều lô CÙNG tenant đồng thời: khoá tuần tự hoá TR3 làm kết cục là hàm của TẬP bản (như nạp tuần tự trên kho bộ nhớ), mỗi lô bảo toàn (BB1), phép kiểm tính chất đúng.
func TestPG_LC74_NhieuLoCungTenant_DongThoi_KhoaTuanTuTR3(t *testing.T) {
	st, _ := pgSetup(t)
	pg := NewPG(st, testOptions())
	ctx, _ := newCtx("nhieulo")
	const workers, per = 8, 6
	all := make([][]Item, workers)
	var flat []Item
	for w := 0; w < workers; w++ {
		for i := 0; i < per; i++ {
			it := mk(fmt.Sprintf("w%d-%d", w, i), (w*7+i*11)%150, withRaw(fmt.Sprintf("tin %d %d", w, i)))
			all[w] = append(all[w], it)
			flat = append(flat, it)
		}
	}
	receipts := make([]Receipt, workers)
	errs := make([]error, workers)
	var wg sync.WaitGroup
	start := make(chan struct{})
	for w := 0; w < workers; w++ {
		wg.Add(1)
		go func(w int) {
			defer wg.Done()
			<-start
			receipts[w], errs[w] = pg.DeliverBatch(ctx, all[w])
		}(w)
	}
	close(start)
	wg.Wait()
	for w := 0; w < workers; w++ {
		if errs[w] != nil {
			t.Fatalf("lô %d: %v", w, errs[w])
		}
		if err := receipts[w].Conserved(keysOf(all[w])); err != nil {
			t.Fatalf("lô %d: %v", w, err)
		}
	}
	recs := mustRecords(t, ctx, pg)
	if len(recs) != len(flat) {
		t.Fatalf("kho có %d bản ghi, muốn %d (không mất, không nhân đôi)", len(recs), len(flat))
	}
	if bad := oracle(recs, 60*time.Minute); len(bad) > 0 {
		t.Fatalf("vi phạm sau nạp đồng thời:\n  %s", strings.Join(bad, "\n  "))
	}
	mem := NewMemory(testOptions())
	mctx, _ := newCtx("nhieulo-mem")
	mustBatch(t, mctx, mem, flat...)
	if got, want := dispOf(recs), dispOf(mustRecords(t, mctx, mem)); !reflect.DeepEqual(got, want) {
		t.Fatalf("nạp đồng thời trên PG khác nạp tuần tự trên kho bộ nhớ (kết cục TR3 phải là hàm của tập bản):\n PG  %v\n mem %v", got, want)
	}
}

// LC67: PG ngắt giữa lô (backend bị kết thúc ngay trước dòng sổ của đơn vị thứ ba): lỗi mang nhãn hạ tầng; không ghi nửa đơn vị (bản ghi, phiên bản, lịch sử, sổ cùng số); chạy lại (lô mới) hội tụ.
func TestPG_LC67_LC64_LoiGiuaGiaoDich_KhongNuaGhi_ChayLaiHoiTu(t *testing.T) {
	st, admin := pgSetup(t)
	items := []Item{mk("u1", 0, withEntity("E1")), mk("u2", 0, withEntity("E2")), mk("u3", 0, withEntity("E3")), mk("u4", 0, withEntity("E4")), mk("u5", 0, withEntity("E5"))}

	// (1) lỗi đơn giản giữa giao dịch: dòng sổ của đơn vị đầu bị từ chối sau khi bản ghi, phiên bản, lịch sử đã ghi trong cùng giao dịch
	pg := NewPG(st, testOptions())
	ctx, ten := newCtx("giua")
	boom := errors.New("lỗi giả lập giữa giao dịch")
	pg.hook = func(ctx context.Context, tx pgx.Tx, op string) error {
		if op == "insertLedger" {
			return boom
		}
		return nil
	}
	_, err := pg.Deliver(ctx, "b0", 1, items[0])
	if !errors.Is(err, boom) || !strings.HasPrefix(contracts.FaultOf(err), "infra:") {
		t.Fatalf("lỗi giữa giao dịch: muốn nhãn hạ tầng và lỗi gốc, được %v (nhãn %q)", err, contracts.FaultOf(err))
	}
	for _, tb := range reportTables {
		if n := adminCount(t, admin, tb, ten); n != 0 {
			t.Errorf("%s có %d dòng sau giao dịch lỗi: nửa ghi", tb, n)
		}
	}

	// (2) PG ngắt thật: kết thúc backend của giao dịch ở đơn vị thứ ba
	ctx2, ten2 := newCtx("ngat")
	calls := 0
	pg.hook = func(ctx context.Context, tx pgx.Tx, op string) error {
		if op != "insertLedger" {
			return nil
		}
		calls++
		if calls == 3 {
			var pid int
			if err := tx.QueryRow(ctx, "SELECT pg_backend_pid()").Scan(&pid); err != nil {
				return err
			}
			if _, err := admin.Exec(context.Background(), "SELECT pg_terminate_backend($1)", pid); err != nil {
				return err
			}
			time.Sleep(200 * time.Millisecond)
		}
		return nil
	}
	rc, err := pg.DeliverBatch(ctx2, items)
	if err == nil || !strings.HasPrefix(contracts.FaultOf(err), "infra:") {
		t.Fatalf("PG ngắt giữa lô: muốn lỗi nhãn hạ tầng, được %v (nhãn %q)", err, contracts.FaultOf(err))
	}
	if len(rc.Entries) != 2 {
		t.Fatalf("biên nhận của lô dừng có %d dòng, muốn 2 (chỉ đơn vị đã xử lý xong)", len(rc.Entries))
	}
	for _, tb := range reportTables {
		want := 2
		if tb == "report_history" || tb == "report_versions" || tb == "report_records" || tb == "delivery_ledger" {
			want = 2
		}
		if n := adminCount(t, admin, tb, ten2); n != want {
			t.Errorf("%s có %d dòng sau lô bị ngắt, muốn %d (mỗi đơn vị một giao dịch, không nửa đơn vị)", tb, n, want)
		}
	}
	pg.hook = nil
	rc2, err := pg.DeliverBatch(ctx2, items) // chạy lại: một lô MỚI
	if err != nil {
		t.Fatalf("chạy lại: %v", err)
	}
	if err := rc2.Conserved(keysOf(items)); err != nil {
		t.Fatal(err)
	}
	if rc2.BatchID == rc.BatchID || rc2.ByOutcome[OutcomeDuplicate] != 2 || rc2.ByOutcome[OutcomeAccepted] != 3 {
		t.Errorf("chạy lại: lô %s (cũ %s), biên nhận %v; muốn lô mới, 2 DUPLICATE và 3 ACCEPTED (0 mất, 0 nhân đôi)", rc2.BatchID, rc.BatchID, rc2.ByOutcome)
	}
	mem := NewMemory(testOptions())
	mctx, _ := newCtx("ngat-mem")
	mustBatch(t, mctx, mem, items...)
	if got, want := dispOf(mustRecords(t, ctx2, pg)), dispOf(mustRecords(t, mctx, mem)); !reflect.DeepEqual(got, want) {
		t.Errorf("sau chạy lại kho khác lần nạp sạch: %v ≠ %v", got, want)
	}
}

// LC53, LC64, CL3: HELD và QUARANTINED sống sót qua khởi động lại (Store mới, pool mới) y nguyên từng trường.
func TestPG_LC53_LC64_BenVung_QuaKhoiDongLai_HeldVaQuarantinedConNguyen(t *testing.T) {
	_, app := pgstore.EnvDSN()
	_, _ = pgSetup(t)
	ctx, _ := newCtx("khoidong")
	open := func() (*PG, func()) {
		st, err := pgstore.Open(context.Background(), withPoolMax(app, 4))
		if err != nil {
			t.Fatal(err)
		}
		return NewPG(st, testOptions()), st.Close
	}
	pg1, close1 := open()
	items := []Item{
		mk("a", 0), mk("b", 30), mk("q", 400, quarantinedFor(ReasonUnrecognizedLine), withEntity(""), withDay(""), withValue("")),
		mk("h", 500, heldFor(ReasonAmbiguousEntity), withEntity("")), mk("lau", 600, withNoSentAt("7"), withEntity("E7")),
	}
	mustBatch(t, ctx, pg1, items...)
	markDup(t, ctx, pg1, "b")
	before := mustRecords(t, ctx, pg1)
	beforeLed, _ := pg1.Ledger(ctx, "")
	close1() // "dừng đột ngột"
	pg2, close2 := open()
	defer close2()
	after := mustRecords(t, ctx, pg2)
	afterLed, _ := pg2.Ledger(ctx, "")
	if !reflect.DeepEqual(before, after) || !reflect.DeepEqual(beforeLed, afterLed) {
		t.Fatalf("sau khởi động lại bản ghi hoặc sổ đổi:\n trước %+v\n sau   %+v", before, after)
	}
	wantRec(t, ctx, pg2, "q", DispQuarantined, ReasonUnrecognizedLine)
	wantRec(t, ctx, pg2, "h", DispHeld, ReasonAmbiguousEntity)
	wantRec(t, ctx, pg2, "b", DispRejected, ReasonPossibleDuplicate)
}

func normRecords(rs []Record) []Record {
	out := append([]Record(nil), rs...)
	for i := range out {
		out[i].DecidedAt = time.Time{}
	}
	return out
}

func normHistory(hs []HistoryEntry) []HistoryEntry {
	out := append([]HistoryEntry(nil), hs...)
	for i := range out {
		out[i].At = time.Time{}
	}
	return out
}

// Cùng chuỗi sự kiện ngẫu nhiên (hạt giống cố định) trên kho bộ nhớ và kho PG cho cùng bản ghi, cùng lịch sử, cùng sổ (trừ dấu thời gian do hệ sinh): hai kho không lệch nhau về luật,
// vì cùng một engine; test này bắt lệch ở phần ghi và đọc SQL (số thập phân, ngày, byte của raw, thứ tự, null).
func TestPGSoKhop_BoNhoVaPG_ChuoiNgauNhien(t *testing.T) {
	st, _ := pgSetup(t)
	pg := NewPG(st, testOptions())
	mem := NewMemory(testOptions())
	for s := 0; s < 60; s++ {
		ctx, _ := newCtx("sokhop")
		gm, gp := newGenRun(31000+int64(s), mem, ctx), newGenRun(31000+int64(s), pg, ctx)
		steps := gm.steps()
		_ = gp.steps() // cùng số số ngẫu nhiên đã dùng: hai bộ sinh đi cùng một đường
		for i := 0; i < steps; i++ {
			dm, _, em := gm.step(t)
			dp, _, ep := gp.step(t)
			if dm != dp || (em == nil) != (ep == nil) {
				t.Fatalf("hạt giống %d, bước %d: kho bộ nhớ %q (lỗi %v) khác kho PG %q (lỗi %v)", 31000+s, i+1, dm, em, dp, ep)
			}
		}
		rm, _ := mem.Records(ctx)
		rp, _ := pg.Records(ctx)
		if !reflect.DeepEqual(normRecords(rm), normRecords(rp)) {
			for i := range rm {
				if i < len(rp) && !reflect.DeepEqual(normRecords(rm)[i], normRecords(rp)[i]) {
					t.Fatalf("hạt giống %d: bản ghi %s khác nhau:\n mem %+v\n pg  %+v", 31000+s, rm[i].Key, normRecords(rm)[i], normRecords(rp)[i])
				}
			}
			t.Fatalf("hạt giống %d: số bản ghi %d (mem) khác %d (pg)", 31000+s, len(rm), len(rp))
		}
		for _, r := range rm {
			hm, _ := mem.History(ctx, r.Key)
			hp, _ := pg.History(ctx, r.Key)
			if !reflect.DeepEqual(normHistory(hm), normHistory(hp)) {
				t.Fatalf("hạt giống %d: lịch sử của %s khác nhau:\n mem %+v\n pg  %+v", 31000+s, r.Key, normHistory(hm), normHistory(hp))
			}
		}
		lm, _ := mem.Ledger(ctx, "")
		lp, _ := pg.Ledger(ctx, "")
		for i := range lm {
			lm[i].RecordedAt = time.Time{}
		}
		for i := range lp {
			lp[i].RecordedAt = time.Time{}
		}
		if !reflect.DeepEqual(lm, lp) {
			t.Fatalf("hạt giống %d: sổ lần giao khác nhau", 31000+s)
		}
		tm, _ := mem.Totals(ctx)
		tp, _ := pg.Totals(ctx)
		if !reflect.DeepEqual(tm, tp) {
			t.Fatalf("hạt giống %d: tổng khác nhau: %v ≠ %v", 31000+s, tm, tp)
		}
	}
}
