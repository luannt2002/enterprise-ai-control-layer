package pgstore_test

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/luannt2002/enterprise-ai-control-layer/core/contracts"
	"github.com/luannt2002/enterprise-ai-control-layer/data/entityresolution"
	"github.com/luannt2002/enterprise-ai-control-layer/data/sourceoftruth"
	"github.com/luannt2002/enterprise-ai-control-layer/db"
	"github.com/luannt2002/enterprise-ai-control-layer/intelligence/rag"
	"github.com/luannt2002/enterprise-ai-control-layer/platform/pgstore"
)

// Các test dưới đây là của sổ nạp load_records (migration 0006, việc 5e, R2-01): phiên bản dữ liệu của lần nạp ĐÃ COMMIT, ghi cùng giao dịch thay dữ liệu của tenant.
// Cần PG như các test pgstore khác; thiếu biến môi trường thì open() bỏ qua.

// ver dựng một phiên bản dữ liệu hợp lệ (64 hex) từ một số, để mỗi lần nạp của test có phiên bản riêng, đọc được.
func ver(n int) string { return fmt.Sprintf("%064x", n) }

// loadV nạp cho tenant một lần với phiên bản ver(n): một thực thể và MỘT fact mà giá trị là n — nên đọc fact ra là biết dữ liệu đang sống thuộc lần nạp nào.
func loadV(s *pgstore.Store, tenant contracts.TenantID, n int) error {
	ctx := contracts.WithTenant(context.Background(), tenant)
	es := []entityresolution.Entity{{ID: "EX-03", Type: "equipment", Key: "EX-03", Name: "Máy xúc EX-03"}}
	fs := []sourceoftruth.Fact{{ID: "R-1", Metric: "issued_fuel", EntityID: "EX-03", OccurredOn: day,
		Value:  contracts.Value{Kind: contracts.ValueQuantity, Number: fmt.Sprint(n), Unit: "L"},
		Source: sourceoftruth.Observation{ID: "R-1", SourceType: "warehouse_slip", SourceRef: "R-1"}}}
	return s.LoadWithVersion(ctx, pgstore.LoadInfo{LoadID: fmt.Sprintf("load-%d", n), DataVersion: ver(n)}, es, fs, []rag.Document{})
}

// liveValue là giá trị fact đang sống của tenant (đọc qua Store, tức qua RLS).
func liveValue(t *testing.T, s *pgstore.Store, tenant contracts.TenantID) string {
	t.Helper()
	fs, err := s.Facts(contracts.WithTenant(context.Background(), tenant), "issued_fuel", nil, day, day)
	if err != nil || len(fs) != 1 {
		t.Fatalf("đọc fact đang sống của %s: %d fact, lỗi %v", tenant, len(fs), err)
	}
	return fs[0].Value.Number
}

// CHO PHÉP: LoadWithVersion ghi phiên bản, LatestLoad đọc lại đúng; lần nạp sau thay lần trước (Seq tăng), kể cả nạp lại CÙNG phiên bản.
func TestLoadRecordsGhiVaDocLai(t *testing.T) {
	s, _, _ := open(t)
	a := uniq("ta")
	ctx := contracts.WithTenant(context.Background(), a)
	if err := loadV(s, a, 1); err != nil {
		t.Fatal(err)
	}
	r1, err := s.LatestLoad(ctx)
	if err != nil || r1.DataVersion != ver(1) || r1.LoadID != "load-1" || r1.Seq <= 0 || time.Since(r1.LoadedAt) > time.Minute || time.Since(r1.LoadedAt) < -time.Minute {
		t.Fatalf("LatestLoad sau lần nạp 1: %+v (lỗi %v), muốn phiên bản %s, mã load-1", r1, err, ver(1))
	}
	for _, n := range []int{2, 2} { // lần thứ hai nạp lại CÙNG phiên bản: vẫn là một dòng mới
		if err := loadV(s, a, n); err != nil {
			t.Fatal(err)
		}
	}
	r3, err := s.LatestLoad(ctx)
	if err != nil || r3.DataVersion != ver(2) || r3.Seq <= r1.Seq+1 {
		t.Fatalf("LatestLoad sau ba lần nạp: %+v (lỗi %v), muốn phiên bản %s và Seq > %d", r3, err, ver(2), r1.Seq+1)
	}
	if got := liveValue(t, s, a); got != "2" {
		t.Errorf("dữ liệu sống = %s, muốn 2 (lần nạp mới nhất)", got)
	}
}

// Chưa từng nạp ⇒ ErrNotFound nhãn input (là "chưa nạp", không phải lỗi hạ tầng). Load KHÔNG kèm phiên bản (đường của bộ chạy winrun và test) thì ghi dòng
// "không ghi nhận" (phiên bản rỗng) và dòng đó THAY phiên bản của lần nạp có phiên bản trước nó: dữ liệu đã đổi, nên phiên bản cũ không được đọc ra như thể vẫn
// mô tả nó. Ngữ cảnh không tenant bị từ chối trước khi chạm DB.
func TestLoadRecordsChuaNapVaNapKhongPhienBan(t *testing.T) {
	s, _, _ := open(t)
	a := uniq("ta")
	ctx := contracts.WithTenant(context.Background(), a)
	if _, err := s.LatestLoad(ctx); !errors.Is(err, pgstore.ErrNotFound) || contracts.FaultOf(err) != contracts.FaultInput {
		t.Fatalf("tenant chưa nạp: muốn ErrNotFound / %s, được %v", contracts.FaultInput, err)
	}
	if err := loadV(s, a, 7); err != nil {
		t.Fatal(err)
	}
	seed(t, s, a, "Máy xúc EX-03", "4000", "") // Load cũ: không kèm phiên bản
	r, err := s.LatestLoad(ctx)
	if err != nil || r.DataVersion != "" || r.LoadID != "" {
		t.Fatalf("sau Load không kèm phiên bản: %+v (lỗi %v), muốn dòng 'không ghi nhận' (phiên bản và mã rỗng), KHÔNG phải %s của lần nạp trước", r, err, ver(7))
	}
	if _, err := s.LatestLoad(context.Background()); contracts.FaultOf(err) != contracts.FaultPolicy {
		t.Errorf("LatestLoad không tenant: muốn lỗi %s, được %v", contracts.FaultPolicy, err)
	}
}

// Phiên bản sai dạng bị từ chối ở tầng Go, KHÔNG chạm DB (Store rỗng không có pool: ca nào chạm DB là panic ⇒ đỏ).
func TestLoadWithVersionTuChoiPhienBanSai(t *testing.T) {
	s := &pgstore.Store{}
	ctx := contracts.WithTenant(context.Background(), uniq("ta"))
	for name, info := range map[string]pgstore.LoadInfo{
		"phiên bản rỗng":      {LoadID: "l", DataVersion: ""},
		"mã lần nạp rỗng":     {LoadID: "", DataVersion: ver(1)},
		"phiên bản 63 hex":    {LoadID: "l", DataVersion: ver(1)[1:]},
		"phiên bản chữ hoa":   {LoadID: "l", DataVersion: strings.ToUpper(fmt.Sprintf("%064x", 0xabc))},
		"phiên bản không hex": {LoadID: "l", DataVersion: strings.Repeat("z", 64)},
	} {
		if err := s.LoadWithVersion(ctx, info, nil, nil, nil); contracts.FaultOf(err) != contracts.FaultInput {
			t.Errorf("%s: muốn lỗi %s, được %v", name, contracts.FaultInput, err)
		}
	}
}

// "Cùng giao dịch": lần nạp hỏng thì dữ liệu cũ VÀ phiên bản cũ còn nguyên — ở cả hai chiều. (1) Hỏng ở phần dữ liệu (hai fact cùng mã ⇒ vi phạm khoá chính ở
// câu lệnh trước dòng sổ). (2) Hỏng ở CHÍNH dòng sổ (trigger giả của quản trị chặn INSERT của tenant này): dữ liệu đã được thay trong giao dịch cũng phải bị huỷ.
// Nếu dòng sổ nằm ngoài giao dịch thì một trong hai chiều để lại dữ liệu của lần này mà phiên bản của lần trước (hoặc ngược lại).
func TestLoadRecordsCungGiaoDichVoiDuLieu(t *testing.T) {
	s, admin, _ := open(t)
	a := uniq("ta")
	ctx := contracts.WithTenant(context.Background(), a)
	if err := loadV(s, a, 1); err != nil {
		t.Fatal(err)
	}
	before, err := s.LatestLoad(ctx)
	if err != nil {
		t.Fatal(err)
	}
	intact := func(what string) {
		t.Helper()
		after, err := s.LatestLoad(ctx)
		if err != nil || after != before {
			t.Errorf("%s: sổ nạp đổi: trước %+v, sau %+v (lỗi %v)", what, before, after, err)
		}
		if got := liveValue(t, s, a); got != "1" {
			t.Errorf("%s: dữ liệu sống = %s, muốn 1 (lần nạp trước còn nguyên)", what, got)
		}
	}

	dupFact := sourceoftruth.Fact{ID: "R-2", Metric: "issued_fuel", EntityID: "EX-03", OccurredOn: day,
		Value: contracts.Value{Kind: contracts.ValueQuantity, Number: "9", Unit: "L"}, Source: sourceoftruth.Observation{ID: "R-2", SourceType: "warehouse_slip"}}
	es := []entityresolution.Entity{{ID: "EX-03", Type: "equipment", Key: "EX-03", Name: "Máy xúc EX-03"}}
	err = s.LoadWithVersion(ctx, pgstore.LoadInfo{LoadID: "load-9", DataVersion: ver(9)}, es, []sourceoftruth.Fact{dupFact, dupFact}, nil)
	if contracts.FaultOf(err) != contracts.FaultData {
		t.Errorf("hai fact cùng mã: muốn lỗi %s (vi phạm khoá chính), được %v", contracts.FaultData, err)
	}
	intact("hỏng ở phần dữ liệu")

	conn := connect(t, admin)
	trigger := fmt.Sprintf("so_nap_loi_gia_%d", time.Now().UnixNano()%1_000_000)
	for _, stmt := range []string{
		`CREATE OR REPLACE FUNCTION so_nap_loi_gia() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN RAISE EXCEPTION 'lỗi giả của sổ nạp'; END $$`,
		fmt.Sprintf(`CREATE TRIGGER %s BEFORE INSERT ON load_records FOR EACH ROW WHEN (NEW.tenant_id = '%s') EXECUTE FUNCTION so_nap_loi_gia()`, trigger, a),
	} {
		if _, err := conn.Exec(context.Background(), stmt); err != nil {
			t.Fatal(err)
		}
	}
	t.Cleanup(func() { conn.Exec(context.Background(), "DROP TRIGGER IF EXISTS "+trigger+" ON load_records") })
	if err := loadV(s, a, 2); err == nil {
		t.Fatal("dòng sổ bị trigger chặn mà LoadWithVersion không báo lỗi")
	}
	intact("hỏng ở dòng sổ")
}

// Các lần nạp CÙNG tenant chạy chồng nhau: sổ và dữ liệu phải khớp nhau ở trạng thái cuối — phiên bản mới nhất là phiên bản của lần nạp mà dữ liệu đang sống thuộc về
// (giá trị fact = số của lần nạp). Id của sổ tăng theo thứ tự commit vì dòng sổ nằm dưới khoá tư vấn của tenant; nếu dòng sổ ghi ngoài khoá, lần nạp thắng cuộc cuối cùng
// có thể mang id nhỏ hơn và phiên bản "mới nhất" sẽ lệch dữ liệu.
func TestLoadRecordsNapChongNhauKhopDuLieu(t *testing.T) {
	s, admin, _ := open(t)
	a := uniq("ta")
	const writers = 8
	start, errs := make(chan struct{}), make(chan error, writers)
	var wg sync.WaitGroup
	for i := 1; i <= writers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			errs <- loadV(s, a, i)
		}()
	}
	close(start)
	wg.Wait()
	close(errs)
	for err := range errs {
		if err != nil {
			t.Errorf("lần nạp chồng nhau: %v", err)
		}
	}
	last, err := s.LatestLoad(contracts.WithTenant(context.Background(), a))
	if err != nil {
		t.Fatal(err)
	}
	if want := ver(0); last.DataVersion == want {
		t.Fatalf("phiên bản mới nhất không phải của lần nạp nào: %s", last.DataVersion)
	}
	var n int
	if _, err := fmt.Sscanf(last.DataVersion, "%064x", &n); err != nil || n < 1 || n > writers {
		t.Fatalf("không đọc ra số lần nạp từ phiên bản %q: %v", last.DataVersion, err)
	}
	if got := liveValue(t, s, a); got != fmt.Sprint(n) {
		t.Errorf("phiên bản mới nhất là của lần nạp %d nhưng dữ liệu sống là của lần nạp %s: sổ và dữ liệu lệch nhau", n, got)
	}
	if got := visible(t, connect(t, admin), "", "load_records", "tenant_id = $1", string(a)); got != writers {
		t.Errorf("sổ có %d dòng cho tenant, muốn %d (mỗi lần nạp một dòng)", got, writers)
	}
}

// CHO PHÉP + TỪ CHỐI theo tenant (CLAUDE.md §21): mỗi tenant đọc đúng dòng sổ của mình; tenant khác, ngữ cảnh không tenant và GUC không đặt đều không thấy dòng
// của A; ghi chéo tenant bị RLS chặn; tenant rỗng bị CHECK chặn ngay cả với vai quản trị; vai ứng dụng không sửa, không xoá được dòng nào (quyền, không phải RLS).
func TestLoadRecordsTenantChoPhepVaTuChoi(t *testing.T) {
	s, admin, app := open(t)
	a, b := uniq("ta"), uniq("tb")
	if err := loadV(s, a, 11); err != nil {
		t.Fatal(err)
	}
	if err := loadV(s, b, 22); err != nil {
		t.Fatal(err)
	}
	ctxA, ctxB := contracts.WithTenant(context.Background(), a), contracts.WithTenant(context.Background(), b)

	// tầng hàm của Store: cho phép
	for _, c := range []struct {
		ctx  context.Context
		want string
	}{{ctxA, ver(11)}, {ctxB, ver(22)}} {
		if r, err := s.LatestLoad(c.ctx); err != nil || r.DataVersion != c.want {
			t.Errorf("LatestLoad: %+v (lỗi %v), muốn phiên bản %s của chính tenant", r, err, c.want)
		}
	}
	// tầng SQL thô, vai ứng dụng: từ chối, kèm đối chứng dương
	appConn, adminConn := connect(t, app), connect(t, admin)
	if n := visible(t, appConn, string(b), "load_records", "tenant_id = $1", string(a)); n != 0 {
		t.Errorf("GUC = B hỏi đích danh dòng của A: thấy %d, muốn 0", n)
	}
	if n := visible(t, appConn, "", "load_records", "true"); n != 0 {
		t.Errorf("không GUC (đóng mặc định): thấy %d dòng, muốn 0", n)
	}
	if n := visible(t, appConn, string(a), "load_records", "tenant_id = $1", string(a)); n != 1 {
		t.Errorf("đối chứng: GUC = A phải thấy đúng 1 dòng của A, được %d", n)
	}
	if n := visible(t, adminConn, "", "load_records", "tenant_id = $1", string(a)); n != 1 {
		t.Errorf("đối chứng: vai quản trị phải thấy dòng của A — số 0 ở trên mới có nghĩa; được %d", n)
	}
	// từ chối GHI chéo: GUC = A mà ghi dòng mang tenant B
	ctx := context.Background()
	asTenantA := func(stmt string, args ...any) error {
		return pgx.BeginFunc(ctx, appConn, func(tx pgx.Tx) error {
			if _, err := tx.Exec(ctx, "SELECT set_config('eais.tenant_id', $1, true)", string(a)); err != nil {
				return err
			}
			_, err := tx.Exec(ctx, stmt, args...)
			return err
		})
	}
	if err := asTenantA(`INSERT INTO load_records (tenant_id, load_id, data_version) VALUES ($1, 'x', $2)`, string(b), ver(5)); err == nil || !strings.Contains(err.Error(), "row-level security") {
		t.Errorf("ghi chéo tenant: muốn bị RLS chặn, được %v", err)
	}
	if err := asTenantA(`INSERT INTO load_records (tenant_id, load_id, data_version) VALUES ($1, 'x', $2)`, string(a), ver(5)); err != nil {
		t.Errorf("đối chứng: ghi đúng tenant phải được: %v", err)
	}
	// tenant rỗng: CHECK chặn ở lược đồ, kể cả vai quản trị (bỏ qua RLS nhưng không bỏ qua CHECK)
	if _, err := adminConn.Exec(ctx, `INSERT INTO load_records (tenant_id, load_id, data_version) VALUES ('', 'x', $1)`, ver(5)); err == nil || !strings.Contains(err.Error(), "load_records_tenant_khong_rong") {
		t.Errorf("tenant rỗng: muốn bị CHECK load_records_tenant_khong_rong chặn, được %v", err)
	}
	// vai ứng dụng chỉ SELECT + INSERT: sửa / xoá bị từ chối QUYỀN
	for _, stmt := range []string{`UPDATE load_records SET data_version = ''`, `DELETE FROM load_records`} {
		if err := asTenantA(stmt); err == nil || !strings.Contains(err.Error(), "permission denied") {
			t.Errorf("%s: muốn bị từ chối quyền (sổ chỉ-thêm), được %v", stmt, err)
		}
	}
	if r, err := s.LatestLoad(ctxA); err != nil || r.DataVersion != ver(5) {
		t.Errorf("sau ghi đối chứng, LatestLoad(A) = %+v (lỗi %v), muốn %s", r, err, ver(5))
	}
}

// Ràng buộc của chính DB (vai quản trị bỏ qua RLS nhưng không bỏ qua CHECK): phiên bản phải là 64 hex hoặc rỗng, và mã lần nạp rỗng khi và chỉ khi phiên bản rỗng.
func TestLoadRecordsRangBuocDB(t *testing.T) {
	_, admin, _ := open(t)
	ctx := context.Background()
	conn := connect(t, admin)
	tenant := string(uniq("ta"))
	for name, c := range map[string][2]string{
		"phiên bản không phải hex": {"l", strings.Repeat("g", 64)},
		"phiên bản 63 ký tự":       {"l", ver(1)[1:]},
		"phiên bản 65 ký tự":       {"l", "0" + ver(1)},
		"phiên bản chữ hoa":        {"l", strings.Repeat("A", 64)},
		"có phiên bản, mã rỗng":    {"", ver(1)},
		"phiên bản rỗng, có mã":    {"l", ""},
	} {
		_, err := conn.Exec(ctx, `INSERT INTO load_records (tenant_id, load_id, data_version) VALUES ($1, $2, $3)`, tenant, c[0], c[1])
		if err == nil || !strings.Contains(err.Error(), "violates check constraint") {
			t.Errorf("%s: muốn bị CHECK chặn, được %v", name, err)
		}
	}
	for name, c := range map[string][2]string{"có phiên bản và mã": {"l", ver(1)}, "không ghi nhận (cả hai rỗng)": {"", ""}} {
		if _, err := conn.Exec(ctx, `INSERT INTO load_records (tenant_id, load_id, data_version) VALUES ($1, $2, $3)`, tenant, c[0], c[1]); err != nil {
			t.Errorf("%s: CHECK chặn oan: %v", name, err)
		}
	}
}

// Trạng thái đích của 0006 trên DB dùng chung của các test, rồi trên database trống: chạy lại trần tệp 0006 hai lần không lỗi; mỗi kiểu trôi (quyền cấp thêm, tắt
// FORCE / RLS, xoá hoặc mở toang chính sách) phải THẤY được (ca âm của bộ kiểm) và chạy lại 0006 đưa về trạng thái đích.
func TestLoadRecordsMigration0006(t *testing.T) {
	_, admin, app := open(t)
	tables := []string{"load_records"}
	if p := appendOnlySchemaProblems(t, connect(t, admin), tables); len(p) != 0 {
		t.Fatalf("DB dùng chung phải đúng trạng thái đích của 0006: %v", p)
	}
	fa, _ := freshDB(t, admin, app)
	ctx := context.Background()
	if err := pgstore.Migrate(ctx, fa); err != nil {
		t.Fatal(err)
	}
	conn := connect(t, fa)
	sql, err := db.Migrations.ReadFile("migrations/0006_so_nap_du_lieu.sql")
	if err != nil {
		t.Fatal(err)
	}
	for i := 1; i <= 2; i++ {
		if _, err := conn.Exec(ctx, string(sql)); err != nil {
			t.Fatalf("chạy lại trần tệp 0006 lần %d: %v", i, err)
		}
	}
	if p := appendOnlySchemaProblems(t, conn, tables); len(p) != 0 {
		t.Fatalf("sau khi chạy lại 0006: %v", p)
	}
	for _, drift := range []string{
		"GRANT UPDATE ON load_records TO eais_app",
		"GRANT DELETE ON load_records TO eais_app",
		"GRANT TRUNCATE ON load_records TO eais_app",
		"GRANT SELECT ON load_records TO PUBLIC",
		"ALTER POLICY tenant_isolation ON load_records USING (true)",
		"ALTER POLICY tenant_isolation ON load_records WITH CHECK (true)",
		"ALTER TABLE load_records NO FORCE ROW LEVEL SECURITY",
		"ALTER TABLE load_records DISABLE ROW LEVEL SECURITY",
		"DROP POLICY tenant_isolation ON load_records",
	} {
		if _, err := conn.Exec(ctx, drift); err != nil {
			t.Fatalf("%s: %v", drift, err)
		}
		if p := appendOnlySchemaProblems(t, conn, tables); len(p) == 0 {
			t.Errorf("bộ kiểm trạng thái không thấy chỗ trôi %q", drift)
		}
		if _, err := conn.Exec(ctx, string(sql)); err != nil {
			t.Fatalf("chạy lại 0006 sau %q: %v", drift, err)
		}
		if p := appendOnlySchemaProblems(t, conn, tables); len(p) != 0 {
			t.Errorf("chạy lại 0006 không đưa về trạng thái đích sau %q: %v", drift, p)
		}
	}
}

// Ready biết sổ nạp: thiếu bảng (chưa áp 0006) hay thiếu một quyền của vai ứng dụng thì kho CHƯA sẵn sàng và lỗi nêu đúng bảng / quyền — không thì server khởi động
// được rồi mới hỏng ở lần nạp hoặc lượt hỏi đầu tiên.
func TestReadySoNap(t *testing.T) {
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
	for _, priv := range []string{"SELECT", "INSERT"} {
		if _, err := conn.Exec(ctx, "REVOKE "+priv+" ON load_records FROM eais_app"); err != nil {
			t.Fatal(err)
		}
		if err := s.Ready(ctx); err == nil || !strings.Contains(err.Error(), "thiếu quyền "+priv+" trên load_records") {
			t.Errorf("thu %s: muốn lỗi nêu quyền thiếu, được %v", priv, err)
		}
		if _, err := conn.Exec(ctx, "GRANT "+priv+" ON load_records TO eais_app"); err != nil {
			t.Fatal(err)
		}
		if err := s.Ready(ctx); err != nil {
			t.Errorf("trả %s rồi mà vẫn chưa sẵn sàng: %v", priv, err)
		}
	}
	if _, err := conn.Exec(ctx, "DROP TABLE load_records"); err != nil {
		t.Fatal(err)
	}
	if err := s.Ready(ctx); err == nil || !strings.Contains(err.Error(), "bảng load_records chưa có") || !strings.Contains(err.Error(), "eais migrate") {
		t.Errorf("chưa có bảng sổ nạp: muốn lỗi nêu bảng và lệnh migrate, được %v", err)
	}
}

// BenchmarkLatestLoad đo cái giá THÊM cho mỗi lượt hỏi của việc đọc phiên bản dữ liệu từ PG: một giao dịch nhỏ (BEGIN, đặt tenant, SELECT theo khoá, COMMIT) — một lần ở đầu
// lượt hỏi, và một lần ở đầu lượt lệnh. Hai cỡ sổ (1 dòng và 5000 dòng của cùng tenant) để thấy thời gian không tăng theo số lần nạp đã ghi. Chạy:
//
//	EAIS_PG_ADMIN_DSN=… EAIS_PG_DSN=… go test ./platform/pgstore/ -run '^$' -bench LatestLoad -benchtime 3000x -count 5
func BenchmarkLatestLoad(b *testing.B) {
	admin, app := pgstore.EnvDSN()
	if admin == "" || app == "" {
		b.Skip("chưa có EAIS_PG_ADMIN_DSN / EAIS_PG_DSN — xem deploy/docker-compose.yml")
	}
	ctx := context.Background()
	if err := pgstore.Migrate(ctx, admin); err != nil {
		b.Fatal(err)
	}
	s, err := pgstore.Open(ctx, app)
	if err != nil {
		b.Fatal(err)
	}
	defer s.Close()
	adminConn, err := pgx.Connect(ctx, admin)
	if err != nil {
		b.Fatal(err)
	}
	defer adminConn.Close(ctx)
	for _, rows := range []int{1, 5000} {
		b.Run(fmt.Sprintf("so-%d-dong", rows), func(b *testing.B) {
			tenant := uniq("bench")
			if err := loadV(s, tenant, 1); err != nil {
				b.Fatal(err)
			}
			if rows > 1 {
				if _, err := adminConn.Exec(ctx, `INSERT INTO load_records (tenant_id, load_id, data_version)
					SELECT $1, 'l' || g, lpad(to_hex(g), 64, '0') FROM generate_series(2, $2) g`, string(tenant), rows); err != nil {
					b.Fatal(err)
				}
			}
			tctx := contracts.WithTenant(ctx, tenant)
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				if _, err := s.LatestLoad(tctx); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}
