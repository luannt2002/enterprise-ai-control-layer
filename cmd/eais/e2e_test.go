package main

import (
	"bytes"
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/luannt2002/enterprise-ai-control-layer/core/contracts"
	"github.com/luannt2002/enterprise-ai-control-layer/platform/pgstore"
)

// Các test dưới đây CHỈ chạy trên PG thật (smoke E2E của ứng dụng). Thiếu EAIS_PG_* thì bỏ qua; CI có PG và cấm SKIP.

// Khởi động lại giữa chừng không mất vết, không nhân đôi: đóng server + kho, mở kho + app + server mới trên CÙNG database, vết cũ đọc
// lại byte-đối-byte; nạp lại sau khởi động cho cùng mã lần nạp và 0 bản ghi cách ly mới; healthz báo 503 khi kho đã đóng.
func TestVetBenQuaKhoiDongLai(t *testing.T) {
	st, admin := openPG(t)
	e := newEnv(t, st)
	resp := decodeAs[contracts.AskResponse](t, e.ask(tokDirector, qSQL))
	before := e.trace(tokDirector, resp.TraceID)
	if before.status != 200 {
		t.Fatalf("đọc vết trước khi khởi động lại: %d", before.status)
	}
	rowsBefore := countRows(t, admin, "decision_records", e.tenantA, "")
	e.ts.Close()

	_, dsn := pgstore.EnvDSN()
	st2, err := pgstore.Open(context.Background(), dsn)
	if err != nil {
		t.Fatal(err)
	}
	log := slog.New(slog.NewTextHandler(&syncBuffer{}, nil))
	app2, err := newApp(config{Pack: e.packDir, Data: e.dataRoot, Now: clockForTest}, st2, log)
	if err != nil {
		t.Fatal(err)
	}
	srv2 := newServer(app2, e.srv.principals, log)
	ts2 := httptest.NewServer(srv2.handler())
	defer ts2.Close()
	e2 := &env{t: t, st: st2, app: app2, srv: srv2, ts: ts2, tenantA: e.tenantA, tenantB: e.tenantB}

	after := e2.trace(tokDirector, resp.TraceID)
	if after.status != 200 || string(after.body) != string(before.body) {
		t.Errorf("vết cũ sau khởi động lại khác:\n trước: %s\n sau:   %s", before.body, after.body)
	}
	if app2.dataVersionNow() != e.app.dataVersionNow() {
		t.Errorf("phiên bản dữ liệu sau khởi động lại %q ≠ %q", app2.dataVersionNow(), e.app.dataVersionNow())
	}
	again := decodeAs[contracts.AskResponse](t, e2.ask(tokDirector, qSQL))
	if again.Answer != resp.Answer || again.TraceID == resp.TraceID {
		t.Errorf("sau khởi động lại: câu trả lời %q (muốn %q), trace id %q (muốn mã mới)", again.Answer, resp.Answer, again.TraceID)
	}
	if got := countRows(t, admin, "decision_records", e.tenantA, ""); got != rowsBefore+1 {
		t.Errorf("số dòng vết %d → %d, muốn thêm đúng 1", rowsBefore, got)
	}
	rep, err := app2.ingest(context.Background())
	if err != nil || rep.LoadID != e.load.LoadID || rep.NewQuarantine != 0 {
		t.Errorf("nạp lại sau khởi động lại: %+v, %v — muốn cùng mã %s, 0 cách ly mới", rep, err, e.load.LoadID)
	}
	if got := countRows(t, admin, "quarantine_records", e.tenantA, "load_id = $2", e.load.LoadID); got != e.load.Quarantined {
		t.Errorf("bản ghi cách ly sau nạp lại: %d, muốn %d", got, e.load.Quarantined)
	}

	st2.Close()
	if r := e2.do(http.MethodGet, "/healthz", "", "", nil); r.status != http.StatusServiceUnavailable || strings.Contains(string(r.body), "pool") {
		t.Errorf("kho đã đóng: healthz = %d %s, muốn 503", r.status, r.body)
	}
}

func runCLI(args ...string) (int, string, string) {
	var out, errb bytes.Buffer
	code := run(args, &out, &errb)
	return code, out.String(), errb.String()
}

// runCLIFor như runCLI nhưng giới hạn thời gian: lệnh chạy mãi (serve) bị đột biến thành "chạy được" thì test ĐỎ sau d chứ không treo.
func runCLIFor(d time.Duration, args ...string) (int, string, string) {
	ctx, cancel := context.WithTimeout(context.Background(), d)
	defer cancel()
	var out, errb bytes.Buffer
	code := runCtx(ctx, args, &out, &errb)
	return code, out.String(), errb.String()
}

// Các lệnh CLI trên PG thật: ingest (lặp lại không nhân đôi), quarantine, trace, trace diff — kể cả ca dùng thật của diff: CÙNG câu, hai
// MỐC dữ liệu, chỉ đúng bước đầu tiên đổi.
func TestCLITrenPG(t *testing.T) {
	st, _ := openPG(t)
	e := newEnv(t, st)

	code, out, errs := runCLI("ingest", "-pack", e.packDir, "-data", e.dataRoot)
	wantLine := fmt.Sprintf("NẠP %s: %d quan sát, %d cách ly (0 ghi mới, %d đã có từ lần nạp cùng mã)", e.tenantA, e.load.Observations, e.load.Quarantined, e.load.Quarantined)
	if code != 0 || !strings.Contains(out, wantLine) || !strings.Contains(out, "mã lần nạp: "+e.load.LoadID) {
		t.Fatalf("ingest CLI = %d\n%s%s\nmuốn dòng %q và mã %s", code, out, errs, wantLine, e.load.LoadID)
	}

	code, out, errs = runCLI("quarantine", "-tenant", e.tenantA, e.load.LoadID)
	if code != 0 || !strings.Contains(out, "CÁCH LY lần nạp "+e.load.LoadID) || !strings.Contains(out, "THEO LÝ DO") {
		t.Fatalf("quarantine CLI = %d\n%s%s", code, out, errs)
	}
	for _, reason := range []string{"missing_value", "unknown_entity", "out_of_range", "unit_dimension"} { // bốn bản ghi bẩn có chủ ý của world.yaml
		if !strings.Contains(out, "\t"+reason+"\t") {
			t.Errorf("báo cáo cách ly thiếu lý do %s", reason)
		}
	}
	if n := strings.Count(out, "\n"); n < e.load.Quarantined {
		t.Errorf("báo cáo cách ly có %d dòng, ít hơn %d bản ghi", n, e.load.Quarantined)
	}
	if code, _, errs := runCLI("quarantine", "-tenant", e.tenantA, "load-khong-co"); code != 1 || !strings.Contains(errs, "không có bản ghi cách ly") {
		t.Errorf("mã lần nạp sai: %d %s", code, errs)
	}
	if code, _, _ := runCLI("quarantine", "-tenant", e.tenantB, e.load.LoadID); code != 1 {
		t.Errorf("tenant B đọc cách ly của A qua CLI: %d, muốn 1 (RLS)", code)
	}

	r1 := decodeAs[contracts.AskResponse](t, e.ask(tokDirector, qSQL))
	code, out, errs = runCLI("trace", "-tenant", e.tenantA, r1.TraceID)
	if code != 0 || !strings.Contains(out, "VẾT "+r1.TraceID) || !strings.Contains(out, "USAGE MODEL") {
		t.Fatalf("trace CLI = %d\n%s%s", code, out, errs)
	}
	for i := 0; i <= 10; i++ {
		if !regexp.MustCompile(fmt.Sprintf(`(?m)^%d\s+S%d\s`, i, i)).MatchString(out) {
			t.Errorf("bảng vết thiếu dòng bước S%d", i)
		}
	}
	if code, _, errs := runCLI("trace", "-tenant", e.tenantB, r1.TraceID); code != 1 || !strings.Contains(errs, "không có vết") {
		t.Errorf("tenant B đọc vết của A qua CLI: %d %s", code, errs)
	}
	if code, _, _ := runCLI("trace", "-tenant", e.tenantA, uuid.NewString()); code != 1 {
		t.Errorf("vết không có: %d, muốn 1", code)
	}

	// diff: cùng câu, cùng mốc ⇒ khớp; khác câu ⇒ cảnh báo + khác
	r2 := decodeAs[contracts.AskResponse](t, e.ask(tokDirector, qSQL))
	if code, out, _ := runCLI("trace", "diff", "-tenant", e.tenantA, r1.TraceID, r2.TraceID); code != 0 || !strings.Contains(out, "KHỚP") {
		t.Errorf("hai lượt cùng câu cùng mốc: %d\n%s", code, out)
	}
	r3 := decodeAs[contracts.AskResponse](t, e.ask(tokDirector, qRAG))
	if code, out, _ := runCLI("trace", "diff", "-tenant", e.tenantA, r1.TraceID, r3.TraceID); code != 1 || !strings.Contains(out, "CẢNH BÁO: hai lượt khác CÂU HỎI") || !strings.Contains(out, "BƯỚC ĐẦU TIÊN KHÁC") {
		t.Errorf("hai lượt khác câu: %d\n%s", code, out)
	}

	// hai MỐC dữ liệu: đổi số của phiếu kho R-004, nạp lại qua HTTP (server biết), hỏi lại đúng câu cũ
	csv := filepath.Join(e.dataRoot, e.tenantA, "warehouse.csv")
	b, err := os.ReadFile(csv)
	if err != nil {
		t.Fatal(err)
	}
	changed := strings.Replace(string(b), "R-004,PX-0912,EX-03,2026-10-19,3850,L", "R-004,PX-0912,EX-03,2026-10-19,3860,L", 1)
	if changed == string(b) {
		t.Fatal("không thấy dòng R-004 để đổi")
	}
	if err := os.WriteFile(csv, []byte(changed), 0o644); err != nil {
		t.Fatal(err)
	}
	if r := e.post("/v1/ingest", tokDirector, map[string]string{}); r.status != 200 {
		t.Fatalf("nạp lại sau khi đổi dữ liệu: %d %s", r.status, r.body)
	}
	r4 := decodeAs[contracts.AskResponse](t, e.ask(tokDirector, qSQL))
	if !strings.Contains(r4.Answer, "3.860") || strings.Contains(r4.Answer, "3.850") {
		t.Fatalf("sau khi đổi dữ liệu câu trả lời vẫn %q", r4.Answer)
	}
	code, out, _ = runCLI("trace", "diff", "-tenant", e.tenantA, r1.TraceID, r4.TraceID)
	if code != 1 || !strings.Contains(out, "BƯỚC ĐẦU TIÊN KHÁC: S8 (answer)") || strings.Contains(out, "CẢNH BÁO") {
		t.Errorf("cùng câu, hai mốc dữ liệu: %d\n%s — muốn bước đầu tiên khác là S8 (answer), không cảnh báo khác câu", code, out)
	}
	for _, want := range []string{"3.850", "3.860"} {
		if !strings.Contains(out, want) {
			t.Errorf("báo cáo diff thiếu giá trị hai bên %s", want)
		}
	}
	if !strings.Contains(out, e.load.DataVersion) {
		t.Error("báo cáo diff thiếu phiên bản dữ liệu của mốc cũ")
	}

	// migrate chạy lại được (idempotent)
	if code, out, errs := runCLI("migrate"); code != 0 || !strings.Contains(out, "migration đã áp") {
		t.Errorf("migrate CLI = %d %s%s", code, out, errs)
	}
}

// Vòng chạy thật của `eais serve -dev`: mở kho từ biến môi trường, nghe loopback cổng tự chọn, phục vụ, tắt gọn khi nhận tín hiệu (ngữ
// cảnh huỷ) và in tổng kết bộ đếm.
func TestServeDevChayThat(t *testing.T) {
	st, _ := openPG(t)
	e := newEnv(t, st)
	var stdout, stderr, logs syncBuffer
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan int, 1)
	go func() {
		done <- serveDev(ctx, config{Pack: e.packDir, Data: e.dataRoot, Now: clockForTest}, "127.0.0.1:0", e.srv.principals,
			slog.New(slog.NewTextHandler(&logs, nil)), &stdout, &stderr)
	}()
	var base string
	for range 100 {
		if m := regexp.MustCompile(`đang nghe (http://127\.0\.0\.1:\d+)`).FindStringSubmatch(stdout.String()); m != nil {
			base = m[1]
			break
		}
		time.Sleep(50 * time.Millisecond)
	}
	if base == "" {
		t.Fatalf("server không báo địa chỉ nghe. stdout=%q stderr=%q", stdout.String(), stderr.String())
	}
	ready := func(host string) int {
		req, _ := http.NewRequest(http.MethodGet, base+"/healthz", nil)
		if host != "" {
			req.Host = host
		}
		res, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		res.Body.Close()
		return res.StatusCode
	}
	if got := ready(""); got != 200 {
		t.Errorf("healthz = %d", got)
	}
	if got := ready("evil.example"); got != 403 {
		t.Errorf("Host lạ = %d, muốn 403 (serve -dev bật cổng Host)", got)
	}
	req, _ := http.NewRequest(http.MethodPost, base+"/v1/ask", strings.NewReader(`{"text":"`+qSQL+`","conversation_id":"c","message_id":"m"}`))
	req.Header.Set("Authorization", "Bearer "+tokDirector)
	req.Header.Set("Content-Type", "application/json")
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	res.Body.Close()
	if res.StatusCode != 200 || res.Header.Get("X-Trace-Id") == "" {
		t.Errorf("ask qua serve -dev = %d, trace %q", res.StatusCode, res.Header.Get("X-Trace-Id"))
	}
	cancel()
	select {
	case code := <-done:
		if code != 0 {
			t.Errorf("serve thoát %d, muốn 0. stderr=%q", code, stderr.String())
		}
	case <-time.After(20 * time.Second):
		t.Fatal("serve không tắt sau khi ngữ cảnh bị huỷ")
	}
	// code_version: scripts/chay-thu.sh đối chiếu dòng này với phiên bản đã nhúng bằng -ldflags (đổi định dạng phải đổi cả script).
	for _, want := range []string{"CHẾ ĐỘ DEV — CHỈ DEV", "extractive", "tổng kết", "host_rejected=1", "code_version=" + buildRevision()} {
		if !strings.Contains(logs.String(), want) {
			t.Errorf("log của serve thiếu %q:\n%s", want, logs.String())
		}
	}
}

// freshPG tạo một database trống trong cùng cụm PG và trả DSN quản trị / ứng dụng trỏ vào nó; dọn khi test xong.
func freshPG(t *testing.T, admin, app string) (string, string) {
	t.Helper()
	au, err1 := url.Parse(admin)
	pu, err2 := url.Parse(app)
	if err1 != nil || err2 != nil || au.User == nil || pu.User == nil {
		t.Skipf("DSN không ở dạng URL có tên vai — không dựng được database trống: %v %v", err1, err2)
	}
	name := fmt.Sprintf("eais_cmd_%d_%d", os.Getpid(), time.Now().UnixNano()%1_000_000)
	ctx := context.Background()
	conn, err := pgx.Connect(ctx, admin)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := conn.Exec(ctx, "CREATE DATABASE "+name); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if _, err := conn.Exec(context.Background(), "DROP DATABASE IF EXISTS "+name+" WITH (FORCE)"); err != nil {
			t.Errorf("dọn database %s: %v", name, err)
		}
		conn.Close(context.Background())
	})
	au.Path, pu.Path = "/"+name, "/"+name
	return au.String(), pu.String()
}

// Kho chưa migrate thì serve và ingest TỪ CHỐI chạy và nói đúng việc cần làm (không phục vụ rồi mất vết ở lượt đầu); sau `eais migrate`
// thì chạy được.
func TestKhoChuaMigrateTuChoiKhoiDong(t *testing.T) {
	_, _ = openPG(t) // bỏ qua nếu không có PG; migrate DB mặc định để các test khác dùng
	admin, app := pgstore.EnvDSN()
	fa, fp := freshPG(t, admin, app)
	t.Setenv("EAIS_PG_ADMIN_DSN", fa)
	t.Setenv("EAIS_PG_DSN", fp)
	tn := uniqTenant("e2e-fresh")
	packDir, dataRoot := tenantFixtures(t, tn)

	code, _, errs := runCLI("ingest", "-pack", packDir, "-data", dataRoot)
	if code != 1 || !strings.Contains(errs, "eais migrate") || !strings.Contains(errs, "bảng decision_records chưa có") {
		t.Errorf("ingest trên DB chưa migrate = %d %q, muốn 1 và lời nhắc chạy eais migrate", code, errs)
	}
	code, _, errs = runCLIFor(4*time.Second, "serve", "-dev", "-addr", "127.0.0.1:0", "-pack", packDir, "-world", repoPath("tenants/construction/win/world.yaml"), "-data", dataRoot)
	if code != 1 || !strings.Contains(errs, "eais migrate") {
		t.Errorf("serve trên DB chưa migrate = %d %q, muốn 1 và lời nhắc chạy eais migrate", code, errs)
	}
	if code, out, errs := runCLI("migrate"); code != 0 {
		t.Fatalf("migrate = %d %s%s", code, out, errs)
	}
	if code, out, errs := runCLI("ingest", "-pack", packDir, "-data", dataRoot); code != 0 || !strings.Contains(out, "NẠP "+tn) {
		t.Errorf("ingest sau migrate = %d %s%s", code, out, errs)
	}
}
