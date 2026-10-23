package main

import (
	"context"
	"errors"
	"io/fs"
	"log/slog"
	"net/http/httptest"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/luannt2002/enterprise-ai-control-layer/core/contracts"
	"github.com/luannt2002/enterprise-ai-control-layer/data/entityresolution"
	"github.com/luannt2002/enterprise-ai-control-layer/execution/commands"
	"github.com/luannt2002/enterprise-ai-control-layer/platform/pgstore"
)

// R2-01 (việc 5e). Phiên bản dữ liệu ghi vào vết phải là phiên bản của lần nạp ĐÃ COMMIT vào PG của đúng tenant đó, đọc từ PG — không phải
// băm thư mục trên đĩa lúc server khởi động. Các test dưới đây dựng lại ba tình huống mà README từng ghi là "không kiểm tra nào phát hiện":
// thư mục đổi sau lần nạp, server trỏ nhầm thư mục, nạp bằng CLI khi server đang chạy; cộng ca "chưa từng nạp". Chỉ chạy trên PG thật.
// Test này cố ý chỉ dùng API có từ trước 5e (newApp, runCLI, HTTP) để chạy được — và đỏ — trên mã cũ.

// noLoadMarker là chữ mà vết phải mang khi PG không có lần nạp nào được ghi nhận cho tenant: cố ý viết thẳng ở đây (không dùng hằng của mã)
// để test cũng ghim chữ hiển thị ra người vận hành.
const noLoadMarker = "(chưa ghi nhận lần nạp)"

var hex64 = regexp.MustCompile(`^[0-9a-f]{64}$`)

// askedDataVersion hỏi MỘT câu qua HTTP của e và trả data_version của vết lượt đó (đọc lại từ kho vết, không từ phản hồi).
func (e *env) askedDataVersion() string {
	e.t.Helper()
	r := e.ask(tokDirector, qSQL)
	if r.status != 200 {
		e.t.Fatalf("ask = %d: %s", r.status, r.body)
	}
	resp := decodeAs[contracts.AskResponse](e.t, r)
	return e.getTrace(tokDirector, resp.TraceID).Record.Decision.DataVersion
}

// restartOn dựng app + server mới trên CÙNG kho PG của e nhưng đọc thư mục dữ liệu dataRoot — như `eais serve -data dataRoot` chạy lại sau khi
// thư mục đổi, hoặc chạy nhầm thư mục khác.
func (e *env) restartOn(dataRoot string) *env {
	e.t.Helper()
	_, dsn := pgstore.EnvDSN()
	st2, err := pgstore.Open(context.Background(), dsn)
	if err != nil {
		e.t.Fatal(err)
	}
	log := slog.New(slog.NewTextHandler(&syncBuffer{}, nil))
	app2, err := newApp(config{Pack: e.packDir, Data: dataRoot, Now: clockForTest}, st2, log)
	if err != nil {
		st2.Close()
		e.t.Fatalf("newApp trên %s: %v", dataRoot, err)
	}
	srv2 := newServer(app2, e.srv.principals, log)
	srv2.hostGuard = true
	ts2 := httptest.NewServer(srv2.handler())
	e.t.Cleanup(func() { ts2.Close(); st2.Close() })
	return &env{t: e.t, st: st2, app: app2, srv: srv2, ts: ts2, tenantA: e.tenantA, tenantB: e.tenantB, packDir: e.packDir, dataRoot: dataRoot}
}

// addNote thêm một tệp ghi chú vào thư mục dữ liệu của tenant: đổi băm thư mục, không đổi gì mà bộ nạp đọc (như TestServer/ingest).
func addNote(t *testing.T, dataRoot, tenantID, name string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dataRoot, tenantID, name), []byte("ghi chú "+name), 0o644); err != nil {
		t.Fatal(err)
	}
}

// Thư mục đổi SAU lần nạp rồi server khởi động (lại): PG vẫn giữ dữ liệu của lần nạp cũ ⇒ vết phải khai phiên bản của lần nạp cũ,
// không phải băm của thư mục đã đổi (thứ chưa từng được nạp).
func TestDataVersionDirChangedAfterIngest(t *testing.T) {
	st, _ := openPG(t)
	e := newEnv(t, st)
	loaded := e.load.DataVersion
	if got := e.askedDataVersion(); got != loaded {
		t.Fatalf("đối chứng: ngay sau nạp vết khai %q, muốn phiên bản vừa nạp %q", got, loaded)
	}
	addNote(t, e.dataRoot, e.tenantA, "doi-sau-khi-nap.txt")
	if now, err := dataVersion(filepath.Join(e.dataRoot, e.tenantA)); err != nil || now == loaded {
		t.Fatalf("thư mục phải đã đổi băm so với lần nạp (%q, lỗi %v)", now, err)
	}
	e2 := e.restartOn(e.dataRoot)
	if got := e2.askedDataVersion(); got != loaded {
		t.Errorf("server khởi động lại sau khi thư mục đổi: vết khai %q, muốn phiên bản của lần nạp đã commit %q (thư mục đã đổi chưa từng được nạp)", got, loaded)
	}
	if got := e2.app.dataVersionNow(); got != loaded {
		t.Errorf("phiên bản nền app nhớ lúc khởi động = %q, muốn của PG %q (không phải băm thư mục đã đổi)", got, loaded)
	}
}

// Server trỏ nhầm thư mục (đúng cách README tái hiện: một bản sao thư mục có thêm một dòng trống ở một tệp, PG giữ lần nạp gốc): vết phải
// khai phiên bản của lần nạp trong PG, không phải của bản sao.
func TestDataVersionServeOnWrongDir(t *testing.T) {
	st, _ := openPG(t)
	e := newEnv(t, st)
	loaded := e.load.DataVersion
	copyRoot := filepath.Join(t.TempDir(), "ban-sao")
	copyTree(t, e.dataRoot, copyRoot)
	var csv string
	_ = filepath.WalkDir(filepath.Join(copyRoot, e.tenantA), func(p string, d fs.DirEntry, err error) error {
		if err == nil && csv == "" && !d.IsDir() && strings.HasSuffix(p, ".csv") {
			csv = p
		}
		return nil
	})
	if csv == "" {
		t.Fatal("thư mục dữ liệu seed không có tệp .csv nào để thêm dòng trống")
	}
	f, err := os.OpenFile(csv, os.O_APPEND|os.O_WRONLY, 0)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.WriteString("\n"); err != nil {
		t.Fatal(err)
	}
	f.Close()
	if cp, err := dataVersion(filepath.Join(copyRoot, e.tenantA)); err != nil || cp == loaded {
		t.Fatalf("bản sao phải có băm khác lần nạp gốc (%q, lỗi %v)", cp, err)
	}
	e2 := e.restartOn(copyRoot)
	if got := e2.askedDataVersion(); got != loaded {
		t.Errorf("server chạy trên bản sao thư mục: vết khai %q, muốn phiên bản của lần nạp trong PG %q", got, loaded)
	}
}

// Nạp bằng CLI (`eais ingest`, tiến trình khác) khi server đang chạy: server không được báo gì, nhưng PG đã đổi ⇒ vết sau đó phải khai
// phiên bản của lần nạp mới, không phải phiên bản server nhớ từ lúc khởi động.
func TestDataVersionCLIIngestWhileServing(t *testing.T) {
	st, _ := openPG(t)
	e := newEnv(t, st)
	v0 := e.load.DataVersion
	if got := e.askedDataVersion(); got != v0 {
		t.Fatalf("đối chứng: trước khi nạp lại vết khai %q, muốn %q", got, v0)
	}
	addNote(t, e.dataRoot, e.tenantA, "nap-bang-cli.txt")
	code, out, errs := runCLI("ingest", "-pack", e.packDir, "-data", e.dataRoot)
	if code != 0 {
		t.Fatalf("ingest CLI = %d\n%s%s", code, out, errs)
	}
	var v1 string
	for _, line := range strings.Split(out, "\n") {
		if rest, ok := strings.CutPrefix(line, "phiên bản dữ liệu: "); ok {
			v1 = strings.TrimSpace(rest)
		}
	}
	if !hex64.MatchString(v1) || v1 == v0 {
		t.Fatalf("CLI phải báo phiên bản mới khác %q, được %q\n%s", v0, v1, out)
	}
	if got := e.askedDataVersion(); got != v1 {
		t.Errorf("server đang chạy, PG đã được nạp bằng CLI: vết khai %q, muốn phiên bản của lần nạp mới nhất %q (cũ: %q)", got, v1, v0)
	}
}

// Chưa từng nạp: PG không có lần nạp nào được ghi nhận cho tenant thì vết nói rõ điều đó, KHÔNG bịa một băm từ thư mục trên đĩa.
func TestDataVersionNeverLoaded(t *testing.T) {
	st, _ := openPG(t)
	tn := uniqTenant("e2e-chua-nap")
	packDir, dataRoot := tenantFixtures(t, tn)
	log := slog.New(slog.NewTextHandler(&syncBuffer{}, nil))
	ap, err := newApp(config{Pack: packDir, Data: dataRoot, Now: clockForTest}, st, log)
	if err != nil {
		t.Fatal(err)
	}
	pr := contracts.Principal{ID: "u-director", TenantID: contracts.TenantID(tn), Kind: contracts.PrincipalUser, Roles: []string{"director"}}
	srv := newServer(ap, map[string]contracts.Principal{tokDirector: pr}, log)
	srv.hostGuard = true
	ts := httptest.NewServer(srv.handler())
	t.Cleanup(ts.Close)
	e := &env{t: t, st: st, app: ap, srv: srv, ts: ts, tenantA: tn, packDir: packDir, dataRoot: dataRoot}
	got := e.askedDataVersion()
	if hex64.MatchString(got) {
		t.Errorf("tenant chưa từng nạp mà vết khai một băm %q (băm của thư mục trên đĩa, chưa từng nạp vào PG)", got)
	}
	if got != noLoadMarker {
		t.Errorf("tenant chưa từng nạp: vết khai %q, muốn %q", got, noLoadMarker)
	}
}

// Lượt lỗi cũng mang phiên bản đọc từ PG, theo TENANT CỦA LƯỢT: thân sai (lỗi ở S0) khai phiên bản của lần nạp trong PG; principal của tenant khác bị chính sách chặn thì
// vết của lượt đó (nằm ở tenant của nó) khai "chưa ghi nhận" của chính tenant đó — không mang phiên bản dữ liệu của tenant mà app đang phục vụ.
func TestDataVersionErrorTurns(t *testing.T) {
	st, _ := openPG(t)
	e := newEnv(t, st)
	bad := decodeAs[errBody](t, e.do("POST", "/v1/ask", tokDirector, "application/json", []byte(`{"text":`)))
	if bad.TraceID == "" {
		t.Fatalf("thân sai phải có vết: %+v", bad)
	}
	if got := e.getTrace(tokDirector, bad.TraceID).Record.Decision.DataVersion; got != e.load.DataVersion {
		t.Errorf("lượt thân sai: vết khai %q, muốn phiên bản của lần nạp %q", got, e.load.DataVersion)
	}
	foreign := e.ask(tokDirectorB, qSQL)
	fb := decodeAs[errBody](t, foreign)
	if foreign.status != 403 || fb.TraceID == "" {
		t.Fatalf("principal tenant khác: %d %s, muốn 403 có vết", foreign.status, foreign.body)
	}
	if got := e.getTrace(tokDirectorB, fb.TraceID).Record.Decision.DataVersion; got != noLoadMarker {
		t.Errorf("vết của principal tenant B khai %q, muốn %q (tenant B chưa nạp gì; không mang phiên bản của tenant A %q)", got, noLoadMarker, e.load.DataVersion)
	}
}

// Lượt lệnh cũng khai phiên bản của lần nạp mới nhất trong PG (không phải phiên bản app nhớ): sau một lần nạp bằng CLI, vết của lệnh kế tiếp mang phiên bản mới.
func TestDataVersionCommandTrace(t *testing.T) {
	st, _ := openPG(t)
	e := newEnv(t, st)
	first := wantResult(t, e.cmd(tokDirector, cmdReq(1, "EX-03", dayAgo)), 200, commands.StatusDone, "", "", commands.RetryNone)
	if got := e.getTrace(tokDirector, first.TraceID).Record.Decision.DataVersion; got != e.load.DataVersion {
		t.Fatalf("đối chứng: vết lệnh khai %q, muốn %q", got, e.load.DataVersion)
	}
	addNote(t, e.dataRoot, e.tenantA, "nap-sau-lenh.txt")
	code, out, errs := runCLI("ingest", "-pack", e.packDir, "-data", e.dataRoot)
	if code != 0 {
		t.Fatalf("ingest CLI = %d\n%s%s", code, out, errs)
	}
	v1, _ := dataVersion(filepath.Join(e.dataRoot, e.tenantA))
	second := e.cmd(tokDirector, cmdReq(2, "EX-03", dayAgo)) // trùng phiếu OPEN ⇒ 409, nhưng vẫn là một lượt có vết
	res := decodeAs[commands.Result](t, second)
	if res.TraceID == "" {
		t.Fatalf("lượt lệnh phải có vết: %d %s", second.status, second.body)
	}
	if got := e.getTrace(tokDirector, res.TraceID).Record.Decision.DataVersion; got != v1 || v1 == e.load.DataVersion {
		t.Errorf("sau nạp bằng CLI: vết lệnh khai %q, muốn %q (cũ %q)", got, v1, e.load.DataVersion)
	}
}

// ledgerFailStore là kho PG mà sổ nạp đọc hỏng theo ý test.
type ledgerFailStore struct {
	*pgstore.Store
	fail atomic.Bool
}

func (l *ledgerFailStore) LatestLoad(ctx context.Context) (pgstore.LoadRecord, error) {
	if l.fail.Load() {
		return pgstore.LoadRecord{}, contracts.Label(contracts.Infra("postgres"), errors.New("giả lập: không đọc được sổ nạp"))
	}
	return l.Store.LatestLoad(ctx)
}

// Không đọc được phiên bản dữ liệu: LƯỢT HỎI đóng mặc định (503, không có câu trả lời — một câu không nói được nó dựa trên dữ liệu nào thì không phải câu trả lời của một hệ P8;
// vết của lượt vẫn lưu và khai phiên bản "không biết" = rỗng, không bịa); LƯỢT LỆNH vẫn chạy (kết quả lệnh không đổi vì mất vết, §7.5) với vết khai rỗng.
func TestDataVersionUnreadable(t *testing.T) {
	st, _ := openPG(t)
	ls := &ledgerFailStore{Store: st}
	e := newEnv(t, ls)
	ls.fail.Store(true)
	r := e.ask(tokDirector, qSQL)
	body := decodeAs[errBody](t, r)
	if r.status != 503 || body.TraceID == "" || strings.Contains(string(r.body), `"answer"`) {
		t.Fatalf("không đọc được phiên bản: ask = %d %s, muốn 503 có trace_id và KHÔNG có câu trả lời", r.status, r.body)
	}
	rec := e.getTrace(tokDirector, body.TraceID).Record
	if rec.Decision.DataVersion != "" || rec.Fault != contracts.Infra("postgres") {
		t.Errorf("vết của lượt không đọc được phiên bản: data_version %q (muốn rỗng), fault %q", rec.Decision.DataVersion, rec.Fault)
	}
	cmd := wantResult(t, e.cmd(tokDirector, cmdReq(1, "EX-03", dayAgo)), 200, commands.StatusDone, "", "", commands.RetryNone)
	if got := e.getTrace(tokDirector, cmd.TraceID).Record.Decision.DataVersion; got != "" {
		t.Errorf("vết lệnh khi không đọc được phiên bản khai %q, muốn rỗng", got)
	}
	if !strings.Contains(e.serverLog(), "không đọc được phiên bản dữ liệu") {
		t.Errorf("log của server không ghi việc không đọc được phiên bản:\n%s", e.serverLog())
	}
	ls.fail.Store(false)
	if got := e.askedDataVersion(); got != e.load.DataVersion {
		t.Errorf("sau khi sổ đọc lại được: vết khai %q, muốn %q", got, e.load.DataVersion)
	}
}

// raceStore chèn một lần nạp (qua hook) vào đúng lúc lượt hỏi đọc kho LẦN ĐẦU, và ghi lại phiên bản mà sổ nạp báo ở lúc đó.
type raceStore struct {
	*pgstore.Store
	hook       func()
	once       sync.Once
	seenAtRead atomic.Value // string: phiên bản của sổ nạp ngay sau khi hook chạy, tức lúc lượt đọc dữ liệu
}

func (r *raceStore) Entities(ctx context.Context, types ...string) ([]entityresolution.Entity, error) {
	r.once.Do(func() {
		r.hook()
		if rec, err := r.Store.LatestLoad(ctx); err == nil {
			r.seenAtRead.Store(rec.DataVersion)
		}
	})
	return r.Store.Entities(ctx, types...)
}

// GIỚI HẠN ĐÃ BIẾT (README, "Giới hạn đã biết"): phiên bản được đọc MỘT lần ở đầu lượt; một lần nạp commit GIỮA lượt làm lượt đọc dữ liệu mới mà vết vẫn khai phiên bản đọc ở đầu
// lượt. Test này ghim đúng cửa sổ đó — nó KHÔNG khẳng định hành vi mong muốn, mà khẳng định điều README nói. Ai khép được cửa sổ (đọc lại sau lượt rồi chạy lại, hay đọc dữ liệu
// và phiên bản trong cùng một snapshot) thì test này đỏ: sửa test cùng README.
func TestDataVersionInTurnWindowKnownLimit(t *testing.T) {
	st, _ := openPG(t)
	rs := &raceStore{Store: st}
	e := newEnv(t, rs)
	v0 := e.load.DataVersion
	addNote(t, e.dataRoot, e.tenantA, "nap-giua-luot.txt")
	e2 := e.restartOn(e.dataRoot) // app thứ hai trên cùng PG, đóng vai tiến trình `eais ingest`
	rs.hook = func() {
		if _, err := e2.app.ingest(context.Background()); err != nil {
			t.Errorf("lần nạp chèn giữa lượt: %v", err)
		}
	}
	got := e.askedDataVersion()
	v1, _ := rs.seenAtRead.Load().(string)
	if v1 == "" || v1 == v0 {
		t.Fatalf("hook phải đã nạp phiên bản mới trước khi lượt đọc dữ liệu: sổ báo %q, phiên bản cũ %q", v1, v0)
	}
	if got != v0 {
		t.Errorf("README nói lượt khai phiên bản đọc ở ĐẦU lượt (%q) dù dữ liệu đọc đã là của lần nạp mới (%q); vết khai %q — nếu cửa sổ đã khép thì sửa test này và README", v0, v1, got)
	}
}

// Một lần nạp KHÔNG kèm phiên bản (đường Load cũ của bộ chạy winrun và test) thay dữ liệu của tenant: phiên bản của lần nạp có phiên bản trước nó không còn mô tả dữ liệu, nên
// vết phải khai "chưa ghi nhận", KHÔNG khai lại phiên bản cũ.
func TestDataVersionUnversionedLoadSupersedes(t *testing.T) {
	st, _ := openPG(t)
	e := newEnv(t, st)
	if got := e.askedDataVersion(); got != e.load.DataVersion {
		t.Fatalf("đối chứng: vết khai %q, muốn %q", got, e.load.DataVersion)
	}
	if err := st.Load(contracts.WithTenant(context.Background(), contracts.TenantID(e.tenantA)), nil, nil, nil); err != nil {
		t.Fatal(err)
	}
	if got := e.askedDataVersion(); got != noLoadMarker {
		t.Errorf("sau lần nạp không kèm phiên bản: vết khai %q, muốn %q (không phải phiên bản của lần nạp trước, %q)", got, noLoadMarker, e.load.DataVersion)
	}
}

// Không đọc được sổ nạp lúc khởi động thì app không khởi động (lỗi hạ tầng, rõ nguyên nhân) — không âm thầm lùi về băm thư mục.
func TestDataVersionStartupLedgerError(t *testing.T) {
	st, _ := openPG(t)
	ls := &ledgerFailStore{Store: st}
	ls.fail.Store(true)
	packDir, dataRoot := tenantFixtures(t, uniqTenant("e2e-khoi-dong"))
	_, err := newApp(config{Pack: packDir, Data: dataRoot, Now: clockForTest}, ls, slog.New(slog.NewTextHandler(&syncBuffer{}, nil)))
	if err == nil || contracts.FaultOf(err) != contracts.Infra("postgres") || !strings.Contains(err.Error(), "sổ nạp") {
		t.Errorf("sổ nạp không đọc được lúc khởi động: muốn lỗi hạ tầng nêu sổ nạp, được %v", err)
	}
}
