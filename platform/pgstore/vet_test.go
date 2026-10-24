package pgstore_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/luannt2002/enterprise-ai-control-layer/core/contracts"
	"github.com/luannt2002/enterprise-ai-control-layer/db"
	"github.com/luannt2002/enterprise-ai-control-layer/observability/traces"
	"github.com/luannt2002/enterprise-ai-control-layer/platform/pgstore"
)

// Các test dưới đây là của kho vết bền (migration 0003). Cần PG như các test pgstore khác; thiếu biến môi trường thì open() bỏ qua.

var vetTables = []string{"decision_records", "quarantine_records", "usage_records"}

// vetRecord dựng bản ghi vết của một lượt số liệu giải xong, đúng hình pipeline.Outcome trả về.
func vetRecord(tenant contracts.TenantID, trace string) traces.Record {
	d := traces.Decision{
		TraceID: trace, Tenant: string(tenant), Principal: "u-director", Kind: "user", Roles: []string{"director"},
		Attributes: map[string]string{"entity_scope": "PRJ-A"}, Now: day.Add(10 * time.Hour),
		PackVersion: "pack-v1", CodeVersion: "rev-abc", DataVersion: "data-1",
		Question:   "EX-03 hôm nay được cấp bao nhiêu lít dầu?",
		Normalized: []string{"date:hôm nay=2026-10-19"}, Entities: []string{"EX-03"}, Intent: "fuel_issued",
		Slots: []string{"entity=EX-03"}, Route: "SQL_QUERY", Outcome: "RESOLVED", Evidence: []string{"E1"},
		Verdict: "true|accept", Answer: "Dầu kho xuất của Máy xúc EX-03: 3.850 L.",
	}
	plan := contracts.RoutePlan{Route: contracts.RouteSQLQuery, Template: "metric_total", Reason: "ý định số liệu có đủ khe",
		Intent: contracts.Intent{Name: "fuel_issued", Confidence: 0.9, Rule: "từ khoá: cấp + dầu"}}
	resp := contracts.AskResponse{Answer: d.Answer, Outcome: contracts.OutcomeResolved, Route: contracts.RouteSQLQuery, TraceID: trace,
		Verdict:   contracts.Verdict{Pass: true, Action: contracts.ActionAccept},
		Citations: []contracts.Citation{{EvidenceID: "E1", Source: contracts.SourceRef{SourceType: "warehouse_slip", Name: "Phiếu xuất kho R-1"}}}}
	spans := []traces.Span{{Name: "normalize", Ms: 1}, {Name: "resolve", Ms: 2}, {Name: "execute", Ms: 3}, {Name: "validate", Ms: 1}}
	return traces.NewRecord(traces.Turn{Decision: d, Spans: spans, TotalMs: 9, UnaccountedMs: 2, Plan: plan, Response: resp})
}

func uniqTrace() string { return "trace-" + string(uniq("t")) }

// sameJSON so hai giá trị qua JSON — dạng thật sự nằm trong DB — và nói rõ chỗ lệch.
func sameJSON(t *testing.T, what string, got, want any) {
	t.Helper()
	g, _ := json.Marshal(got)
	w, _ := json.Marshal(want)
	if string(g) != string(w) {
		t.Errorf("%s:\n được %s\n muốn %s", what, g, w)
	}
}

func canonical(t *testing.T, rec traces.Record) traces.Record {
	t.Helper()
	c, err := traces.SanitizeRecord(rec)
	if err != nil {
		t.Fatal(err)
	}
	return c
}

func connect(t *testing.T, dsn string) *pgx.Conn {
	t.Helper()
	conn, err := pgx.Connect(context.Background(), dsn)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { conn.Close(context.Background()) })
	return conn
}

// visible đếm dòng của bảng mà conn thấy khi đặt GUC eais.tenant_id (rỗng = không đặt GUC), lọc theo where.
func visible(t *testing.T, conn *pgx.Conn, guc, table, where string, args ...any) int {
	t.Helper()
	ctx := context.Background()
	var n int
	err := pgx.BeginFunc(ctx, conn, func(tx pgx.Tx) error {
		if guc != "" {
			if _, err := tx.Exec(ctx, "SELECT set_config('eais.tenant_id', $1, true)", guc); err != nil {
				return err
			}
		}
		return tx.QueryRow(ctx, "SELECT count(*) FROM "+table+" WHERE "+where, args...).Scan(&n)
	})
	if err != nil {
		t.Fatal(err)
	}
	return n
}

// vetSeed ghi cho một tenant một lượt (bản ghi quyết định + usage) và một lần nạp có hai bản ghi cách ly.
func vetSeed(t *testing.T, s *pgstore.Store, tenant contracts.TenantID) (trace, load string) {
	t.Helper()
	ctx := contracts.WithTenant(context.Background(), tenant)
	trace, load = uniqTrace(), "load-"+string(uniq("l"))
	if ok, err := s.SaveDecision(ctx, vetRecord(tenant, trace)); err != nil || !ok {
		t.Fatalf("SaveDecision: %v %v", ok, err)
	}
	if ok, err := s.SaveUsage(ctx, pgstore.UsageRecord{TraceID: trace, Model: "m1", Calls: 2, Tokens: contracts.Usage{InputTokens: 10, OutputTokens: 5, CacheRead: 3}}); err != nil || !ok {
		t.Fatalf("SaveUsage: %v %v", ok, err)
	}
	q := []pgstore.QuarantineRecord{
		{Key: "slips/a.csv#7", Source: "warehouse_slip", Reason: "unknown_entity", Detail: `mã "EX-99" không có trong danh mục`},
		{Key: "slips/a.csv#9", Source: "warehouse_slip", Reason: "out_of_range", Detail: "4000000 L vượt miền"},
	}
	if n, err := s.SaveQuarantine(ctx, load, q); err != nil || n != 2 {
		t.Fatalf("SaveQuarantine: %d %v", n, err)
	}
	return trace, load
}

// CHO PHÉP: cùng tenant ghi rồi đọc lại đúng từng trường — bản ghi lượt thường, lượt lỗi, cách ly (kèm lý do), usage.
func TestVetChoPhep(t *testing.T) {
	s, _, _ := open(t)
	a := uniq("ta")
	ctx := contracts.WithTenant(context.Background(), a)
	trace, load := vetSeed(t, s, a)

	got, err := s.Decision(ctx, trace)
	if err != nil {
		t.Fatal(err)
	}
	sameJSON(t, "bản ghi lượt thường", got, canonical(t, vetRecord(a, trace)))
	if len(got.Steps) != 11 || got.Steps[5].Stage != "S5" || got.Steps[5].Reason != "ý định số liệu có đủ khe" || got.Spans[2].Ms != 3 || got.TotalMs != 9 {
		t.Errorf("vết từng bước / thời gian không về nguyên vẹn: %d bước, %+v", len(got.Steps), got.Spans)
	}

	// lượt lỗi cũng ghi: nhãn lỗi và thông điệp đã làm sạch
	errTrace := uniqTrace()
	pr := contracts.Principal{ID: "u-director", TenantID: a, Kind: contracts.PrincipalUser, Roles: []string{"director"}}
	errRec := traces.NewErrorRecord(errTrace, pr, "EX-03 hôm nay?", day, traces.Versions{Pack: "pack-v1", Code: "rev-abc", Data: "data-1"},
		contracts.Label(contracts.Infra("postgres"), errors.New("hết thời gian chờ")))
	if ok, err := s.SaveDecision(ctx, errRec); err != nil || !ok {
		t.Fatalf("lượt lỗi: %v %v", ok, err)
	}
	gotErr, err := s.Decision(ctx, errTrace)
	if err != nil {
		t.Fatal(err)
	}
	sameJSON(t, "bản ghi lượt lỗi", gotErr, canonical(t, errRec))
	if gotErr.Fault != "infra:postgres" || gotErr.Error != "hết thời gian chờ" || gotErr.Decision.Outcome != "" || gotErr.Steps[0].Status != traces.StatusError {
		t.Errorf("lượt lỗi đọc lại: %+v", gotErr)
	}

	qs, err := s.Quarantine(ctx, load)
	if err != nil || len(qs) != 2 {
		t.Fatalf("cách ly: %v %v", qs, err)
	}
	if qs[0].Key != "slips/a.csv#7" || qs[0].Reason != "unknown_entity" || qs[0].Source != "warehouse_slip" || qs[0].Detail != `mã "EX-99" không có trong danh mục` ||
		qs[1].Reason != "out_of_range" || qs[0].At.IsZero() {
		t.Errorf("cách ly đọc lại: %+v", qs)
	}
	us, err := s.Usage(ctx, trace)
	if err != nil || len(us) != 1 || us[0] != (pgstore.UsageRecord{TraceID: trace, Model: "m1", Calls: 2, Tokens: contracts.Usage{InputTokens: 10, OutputTokens: 5, CacheRead: 3}}) {
		t.Errorf("usage đọc lại: %+v %v", us, err)
	}
}

// TỪ CHỐI: tenant B không đọc được vết / cách ly / usage của A — kể cả khi biết trace id và mã lần nạp — và không ghi chéo được.
// Kiểm ở ba tầng: hàm của Store, SQL thô bằng vai ứng dụng (đặt GUC = B, không GUC, GUC = A), và đối chứng dương bằng vai quản trị
// (số 0 ở trên mới có nghĩa khi dữ liệu của A có thật trong DB).
func TestVetTuChoiTenantKhac(t *testing.T) {
	s, admin, app := open(t)
	a, b := uniq("ta"), uniq("tb")
	traceA, loadA := vetSeed(t, s, a)
	traceB, _ := vetSeed(t, s, b)
	ctxA, ctxB := contracts.WithTenant(context.Background(), a), contracts.WithTenant(context.Background(), b)

	// tầng hàm của Store
	if _, err := s.Decision(ctxA, traceA); err != nil {
		t.Fatalf("đối chứng: A phải đọc được vết của A: %v", err)
	}
	if _, err := s.Decision(ctxB, traceA); !errors.Is(err, pgstore.ErrNotFound) || contracts.FaultOf(err) != contracts.FaultInput {
		t.Errorf("B đọc vết của A (biết trace id): muốn ErrNotFound / %s, được %v", contracts.FaultInput, err)
	}
	if qs, err := s.Quarantine(ctxB, loadA); err != nil || len(qs) != 0 {
		t.Errorf("B đọc cách ly của A (biết mã lần nạp): được %d dòng (lỗi %v), muốn 0", len(qs), err)
	}
	if us, err := s.Usage(ctxB, traceA); err != nil || len(us) != 0 {
		t.Errorf("B đọc usage của A: được %d dòng (lỗi %v), muốn 0", len(us), err)
	}
	if _, err := s.SaveDecision(ctxB, vetRecord(a, uniqTrace())); contracts.FaultOf(err) != contracts.FaultPolicy {
		t.Errorf("B ghi bản ghi mang tenant A: muốn lỗi %s, được %v", contracts.FaultPolicy, err)
	}
	if _, err := s.SaveUsage(ctxB, pgstore.UsageRecord{TraceID: traceA, Model: "m2"}); err == nil {
		t.Errorf("B ghi usage vào lượt của A phải thất bại (A có lượt đó, B thì không)")
	}

	// tầng SQL thô, vai ứng dụng
	appConn, adminConn := connect(t, app), connect(t, admin)
	for _, c := range []struct{ table, col, id string }{
		{"decision_records", "trace_id", traceA}, {"usage_records", "trace_id", traceA}, {"quarantine_records", "load_id", loadA},
	} {
		where := c.col + " = $1"
		if n := visible(t, appConn, string(b), c.table, where, c.id); n != 0 {
			t.Errorf("%s: GUC = B hỏi đích danh %s của A: thấy %d dòng, muốn 0", c.table, c.col, n)
		}
		if n := visible(t, appConn, "", c.table, where, c.id); n != 0 {
			t.Errorf("%s: không GUC (đóng mặc định): thấy %d dòng, muốn 0", c.table, n)
		}
		if n := visible(t, appConn, string(a), c.table, where, c.id); n == 0 {
			t.Errorf("%s: đối chứng: GUC = A phải thấy dữ liệu của A", c.table)
		}
		if n := visible(t, adminConn, "", c.table, where, c.id); n == 0 {
			t.Errorf("%s: đối chứng: vai quản trị phải thấy dữ liệu của A — số 0 ở trên mới có nghĩa", c.table)
		}
	}
	if n := visible(t, appConn, string(b), "decision_records", "true"); n < 1 {
		t.Errorf("B phải thấy vết của chính B (đối chứng), được %d", n)
	} else if m := visible(t, appConn, string(b), "decision_records", "trace_id = $1", traceB); m != 1 {
		t.Errorf("B phải thấy đúng vết %s của B, được %d", traceB, m)
	}

	// từ chối GHI chéo: GUC = A mà ghi dòng mang tenant B → RLS (WITH CHECK) chặn, ở cả ba bảng
	ctx := context.Background()
	for _, q := range []string{
		`INSERT INTO decision_records (tenant_id, trace_id, asked_at, principal_id, question, decision, content_sha256)
			VALUES ($1, 'x', now(), 'p', 'q', '{}', repeat('0', 64))`,
		`INSERT INTO quarantine_records (tenant_id, load_id, record_key, reason) VALUES ($1, 'l', 'k', 'r')`,
		`INSERT INTO usage_records (tenant_id, trace_id, model, calls, input_tokens, output_tokens, cache_read, cache_write)
			VALUES ($1, $2, 'm', 0, 0, 0, 0, 0)`,
	} {
		err := pgx.BeginFunc(ctx, appConn, func(tx pgx.Tx) error {
			if _, err := tx.Exec(ctx, "SELECT set_config('eais.tenant_id', $1, true)", string(a)); err != nil {
				return err
			}
			args := []any{string(b)}
			if strings.Contains(q, "$2") {
				args = append(args, traceB)
			}
			_, err := tx.Exec(ctx, q, args...)
			return err
		})
		if err == nil || !strings.Contains(err.Error(), "row-level security") {
			t.Errorf("ghi chéo tenant: muốn bị RLS chặn, được %v (câu: %.40s…)", err, strings.Join(strings.Fields(q), " "))
		}
	}
}

// Đối chứng cho bài từ chối: một vai BYPASSRLS THẤY dòng của tenant khác. Chứng minh phép đếm trong TestVetTuChoiTenantKhac có thể
// đỏ — nếu RLS hỏng (chính sách bị bỏ, vai vượt RLS) thì chính phép đếm đó đo ra 1 thay vì 0. Vai tạm là NOLOGIN, vào bằng role=
// trên DSN quản trị; Store.Open từ chối vai kiểu này, nên đối chứng dùng kết nối thô.
func TestVetDoiChungVaiVuotRLSThayDuLieuChe(t *testing.T) {
	s, admin, _ := open(t)
	a := uniq("ta")
	traceA, loadA := vetSeed(t, s, a)
	u, err := url.Parse(admin)
	if err != nil || u.User == nil {
		t.Skipf("DSN quản trị không ở dạng URL có tên vai: %v", err)
	}
	ctx := context.Background()
	adminConn := connect(t, admin)
	role := fmt.Sprintf("eais_test_vet_%d_%d", os.Getpid(), time.Now().UnixNano()%1_000_000)
	if _, err := adminConn.Exec(ctx, "CREATE ROLE "+role+" NOLOGIN BYPASSRLS"); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		c := context.Background()
		for _, q := range []string{"DROP OWNED BY " + role, "DROP ROLE IF EXISTS " + role} {
			if _, err := adminConn.Exec(c, q); err != nil {
				t.Errorf("dọn vai tạm: %v", err)
			}
		}
	})
	for _, table := range vetTables {
		if _, err := adminConn.Exec(ctx, "GRANT SELECT ON "+table+" TO "+role); err != nil {
			t.Fatal(err)
		}
	}
	q := u.Query()
	q.Set("role", role)
	u.RawQuery = q.Encode()
	bypass := connect(t, u.String())
	for _, c := range []struct{ table, where, id string }{
		{"decision_records", "trace_id = $1", traceA}, {"usage_records", "trace_id = $1", traceA}, {"quarantine_records", "load_id = $1", loadA},
	} {
		if n := visible(t, bypass, "chu-nhan-khac", c.table, c.where, c.id); n == 0 {
			t.Errorf("%s: vai BYPASSRLS với GUC của tenant khác phải THẤY dữ liệu của A (đó là điều RLS ngăn lại); thấy %d", c.table, n)
		}
	}
}

// KHỞI ĐỘNG LẠI: đóng pool, mở pool mới, đọc lại — không mất gì (mọi thứ nằm trong PG, không trong tiến trình).
func TestVetKhoiDongLai(t *testing.T) {
	s, _, app := open(t)
	a := uniq("ta")
	trace, load := vetSeed(t, s, a)
	want, err := s.Decision(contracts.WithTenant(context.Background(), a), trace)
	if err != nil {
		t.Fatal(err)
	}
	s.Close()

	s2, err := pgstore.Open(context.Background(), app)
	if err != nil {
		t.Fatal(err)
	}
	defer s2.Close()
	ctx := contracts.WithTenant(context.Background(), a)
	got, err := s2.Decision(ctx, trace)
	if err != nil {
		t.Fatalf("sau khi mở lại pool: %v", err)
	}
	sameJSON(t, "bản ghi sau khởi động lại", got, want)
	if qs, err := s2.Quarantine(ctx, load); err != nil || len(qs) != 2 {
		t.Errorf("cách ly sau khởi động lại: %v %v", qs, err)
	}
	if us, err := s2.Usage(ctx, trace); err != nil || len(us) != 1 {
		t.Errorf("usage sau khởi động lại: %v %v", us, err)
	}
	// và ghi tiếp được trên pool mới: lượt sau khi khởi động lại không đụng lượt trước
	if ok, err := s2.SaveDecision(ctx, vetRecord(a, uniqTrace())); err != nil || !ok {
		t.Errorf("ghi sau khởi động lại: %v %v", ok, err)
	}
}

// GHI ĐÔI cùng khoá: y nguyên thì bỏ qua và BÁO (inserted = false), khác nội dung thì LỖI FaultData và không ghi đè; không bao giờ
// nhân đôi, không bao giờ im lặng — ở cả ba bảng, và khi nhiều tiến trình cùng ghi một lúc.
func TestVetGhiDoi(t *testing.T) {
	s, admin, _ := open(t)
	a := uniq("ta")
	ctx := contracts.WithTenant(context.Background(), a)
	adminConn := connect(t, admin)
	count := func(table, where string, args ...any) int {
		return visible(t, adminConn, "", table, "tenant_id = '"+string(a)+"' AND "+where, args...) // tenant do uniq() sinh: chỉ chữ, số, gạch
	}

	// bản ghi quyết định
	trace := uniqTrace()
	rec := vetRecord(a, trace)
	for i, want := range []bool{true, false, false} {
		if got, err := s.SaveDecision(ctx, rec); err != nil || got != want {
			t.Fatalf("lần ghi %d: inserted = %v (lỗi %v), muốn %v", i+1, got, err, want)
		}
	}
	if n := count("decision_records", "trace_id = $1", trace); n != 1 {
		t.Errorf("ghi ba lần y nguyên: %d dòng, muốn 1", n)
	}
	changed := vetRecord(a, trace)
	changed.Decision.Answer = "câu khác"
	if _, err := s.SaveDecision(ctx, changed); contracts.FaultOf(err) != contracts.FaultData {
		t.Errorf("cùng trace_id khác câu trả lời: muốn lỗi %s, được %v", contracts.FaultData, err)
	}
	deep := vetRecord(a, trace)
	deep.Steps[3].Reason += " (đổi sâu trong một bước)"
	if _, err := s.SaveDecision(ctx, deep); contracts.FaultOf(err) != contracts.FaultData {
		t.Errorf("cùng trace_id khác ở một bước: muốn lỗi %s, được %v", contracts.FaultData, err)
	}
	if got, err := s.Decision(ctx, trace); err != nil || got.Decision.Answer != rec.Decision.Answer || got.Steps[3].Reason != rec.Steps[3].Reason {
		t.Errorf("bản gốc phải còn nguyên sau các lần ghi bị từ chối: %v %+v", err, got.Decision.Answer)
	}

	// cách ly: cùng tập y nguyên → 0 dòng mới; chồng một phần → chỉ dòng mới; khác nội dung → lỗi và cả lô bị huỷ
	load := "load-" + string(uniq("l"))
	q := []pgstore.QuarantineRecord{{Key: "f#1", Source: "s", Reason: "missing_value", Detail: "d1"}, {Key: "f#2", Source: "s", Reason: "out_of_range", Detail: "d2"}}
	if n, err := s.SaveQuarantine(ctx, load, q); err != nil || n != 2 {
		t.Fatalf("lần đầu: %d %v", n, err)
	}
	if n, err := s.SaveQuarantine(ctx, load, q); err != nil || n != 0 {
		t.Errorf("ghi lại y nguyên: ghi mới %d (lỗi %v), muốn 0", n, err)
	}
	if n, err := s.SaveQuarantine(ctx, load, append([]pgstore.QuarantineRecord{{Key: "f#3", Source: "s", Reason: "unparseable", Detail: "d3"}}, q...)); err != nil || n != 1 {
		t.Errorf("chồng một phần: ghi mới %d (lỗi %v), muốn 1", n, err)
	}
	clash := []pgstore.QuarantineRecord{{Key: "f#4", Source: "s", Reason: "missing_value", Detail: "d4"}, {Key: "f#1", Source: "s", Reason: "missing_value", Detail: "LÝ DO KHÁC"}}
	if _, err := s.SaveQuarantine(ctx, load, clash); contracts.FaultOf(err) != contracts.FaultData {
		t.Errorf("cùng (khoá, lý do) khác chi tiết: muốn lỗi %s, được %v", contracts.FaultData, err)
	}
	if n := count("quarantine_records", "load_id = $1", load); n != 3 {
		t.Errorf("sau lô bị huỷ: %d dòng, muốn 3 (f#4 của lô lỗi không được lọt vào)", n)
	}
	if n := count("quarantine_records", "load_id = $1 AND record_key = 'f#1' AND detail = 'd1'", load); n != 1 {
		t.Errorf("dòng gốc f#1 phải còn nguyên chi tiết d1")
	}
	// một bản ghi cách ly vì hai lý do là hai dòng (danh tính gồm cả lý do)
	if n, err := s.SaveQuarantine(ctx, load, []pgstore.QuarantineRecord{{Key: "f#1", Source: "s", Reason: "out_of_range", Detail: "hai lý do"}}); err != nil || n != 1 {
		t.Errorf("cùng khoá khác lý do: ghi mới %d (lỗi %v), muốn 1", n, err)
	}

	// usage
	u := pgstore.UsageRecord{TraceID: trace, Model: "m1", Calls: 1, Tokens: contracts.Usage{InputTokens: 7, OutputTokens: 2}}
	for i, want := range []bool{true, false} {
		if got, err := s.SaveUsage(ctx, u); err != nil || got != want {
			t.Fatalf("usage lần %d: %v %v, muốn %v", i+1, got, err, want)
		}
	}
	u.Tokens.OutputTokens = 3
	if _, err := s.SaveUsage(ctx, u); contracts.FaultOf(err) != contracts.FaultData {
		t.Errorf("cùng (lượt, model) khác số token: muốn lỗi %s, được %v", contracts.FaultData, err)
	}
	if got, _ := s.Usage(ctx, trace); len(got) != 1 || got[0].Tokens.OutputTokens != 2 {
		t.Errorf("usage gốc phải còn nguyên: %+v", got)
	}

	// nhiều tiến trình cùng ghi MỘT bản ghi mới: đúng một lần được inserted, không ai lỗi, một dòng
	racing := vetRecord(a, uniqTrace())
	var wg sync.WaitGroup
	var mu sync.Mutex
	wins, fails := 0, 0
	for range 8 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			ins, err := s.SaveDecision(ctx, racing)
			mu.Lock()
			defer mu.Unlock()
			if err != nil {
				fails++
			} else if ins {
				wins++
			}
		}()
	}
	wg.Wait()
	if wins != 1 || fails != 0 || count("decision_records", "trace_id = $1", racing.Decision.TraceID) != 1 {
		t.Errorf("8 lần ghi đồng thời: %d lần ghi mới (muốn 1), %d lỗi (muốn 0)", wins, fails)
	}
}

// Đầu vào sai bị từ chối TRƯỚC khi chạm dữ liệu: không tenant, tenant của bản ghi khác ngữ cảnh, bản ghi hỏng, cách ly không lý do,
// usage cho lượt chưa có vết.
func TestVetTuChoiDauVaoSai(t *testing.T) {
	s, admin, _ := open(t)
	a := uniq("ta")
	ctx := contracts.WithTenant(context.Background(), a)
	bare := context.Background()
	adminConn := connect(t, admin)

	if _, err := s.SaveDecision(bare, vetRecord(a, "x")); contracts.FaultOf(err) != contracts.FaultPolicy {
		t.Errorf("SaveDecision không tenant: %v", err)
	}
	if _, err := s.Decision(bare, "x"); contracts.FaultOf(err) != contracts.FaultPolicy {
		t.Errorf("Decision không tenant: %v", err)
	}
	if _, err := s.SaveQuarantine(bare, "l", []pgstore.QuarantineRecord{{Key: "k", Reason: "r"}}); contracts.FaultOf(err) != contracts.FaultPolicy {
		t.Errorf("SaveQuarantine không tenant: %v", err)
	}
	if _, err := s.Quarantine(bare, "l"); contracts.FaultOf(err) != contracts.FaultPolicy {
		t.Errorf("Quarantine không tenant: %v", err)
	}
	if _, err := s.SaveUsage(bare, pgstore.UsageRecord{TraceID: "x", Model: "m"}); contracts.FaultOf(err) != contracts.FaultPolicy {
		t.Errorf("SaveUsage không tenant: %v", err)
	}
	if _, err := s.Usage(bare, "x"); contracts.FaultOf(err) != contracts.FaultPolicy {
		t.Errorf("Usage không tenant: %v", err)
	}

	other := vetRecord(uniq("tb"), uniqTrace()) // bản ghi mang tenant khác ngữ cảnh
	if _, err := s.SaveDecision(ctx, other); contracts.FaultOf(err) != contracts.FaultPolicy {
		t.Errorf("tenant của bản ghi khác ngữ cảnh: muốn lỗi %s, được %v", contracts.FaultPolicy, err)
	}
	if n := visible(t, adminConn, "", "decision_records", "trace_id = $1", other.Decision.TraceID); n != 0 {
		t.Errorf("bản ghi bị từ chối vẫn lọt vào DB (%d dòng)", n)
	}
	noTrace := vetRecord(a, "")
	if _, err := s.SaveDecision(ctx, noTrace); contracts.FaultOf(err) != contracts.FaultInput {
		t.Errorf("thiếu trace_id: muốn lỗi %s, được %v", contracts.FaultInput, err)
	}
	badStage := vetRecord(a, uniqTrace())
	badStage.Steps[0].Stage = "S11"
	if _, err := s.SaveDecision(ctx, badStage); contracts.FaultOf(err) != contracts.FaultInput {
		t.Errorf("mã bước sai: muốn lỗi %s, được %v", contracts.FaultInput, err)
	}

	load := "load-" + string(uniq("l"))
	noReason := []pgstore.QuarantineRecord{{Key: "ok#1", Reason: "r"}, {Key: "k#2"}}
	if _, err := s.SaveQuarantine(ctx, load, noReason); contracts.FaultOf(err) != contracts.FaultInput {
		t.Errorf("cách ly không lý do: muốn lỗi %s, được %v", contracts.FaultInput, err)
	}
	if n := visible(t, adminConn, "", "quarantine_records", "load_id = $1", load); n != 0 {
		t.Errorf("lô có một dòng không lý do phải bị từ chối nguyên lô, DB có %d dòng", n)
	}
	if _, err := s.SaveQuarantine(ctx, "", []pgstore.QuarantineRecord{{Key: "k", Reason: "r"}}); contracts.FaultOf(err) != contracts.FaultInput {
		t.Errorf("thiếu mã lần nạp: %v", err)
	}
	if _, err := s.SaveUsage(ctx, pgstore.UsageRecord{TraceID: uniqTrace(), Model: "m"}); contracts.FaultOf(err) != contracts.FaultInput {
		t.Errorf("usage cho lượt chưa có bản ghi quyết định: muốn lỗi %s (khoá ngoại), được %v", contracts.FaultInput, err)
	}
	if _, err := s.SaveUsage(ctx, pgstore.UsageRecord{TraceID: "t", Model: "m", Calls: -1}); contracts.FaultOf(err) != contracts.FaultInput {
		t.Errorf("usage số âm: %v", err)
	}
	if _, err := s.Decision(ctx, "khong-co-trace-nay"); !errors.Is(err, pgstore.ErrNotFound) {
		t.Errorf("trace không có: muốn ErrNotFound, được %v", err)
	}
}

// Bí mật không nằm trong DB: lỗi hạ tầng mang DSN có mật khẩu, câu hỏi bị dán nhầm mật khẩu, thuộc tính, chi tiết cách ly — soi
// CẢ HÀNG (row_to_json) bằng vai quản trị, không chỉ những cột mà hàm đọc trả về.
func TestVetBiMatKhongNamTrongDB(t *testing.T) {
	s, admin, _ := open(t)
	a := uniq("ta")
	ctx := contracts.WithTenant(context.Background(), a)
	adminConn := connect(t, admin)
	pass, apikey := "hunter2pw", "AAAA1111BBBB2222"

	trace := uniqTrace()
	pr := contracts.Principal{ID: "u1", TenantID: a, Kind: contracts.PrincipalUser, Roles: []string{"director"}, Attributes: map[string]string{"api_key": apikey, "entity_scope": "PRJ-A"}}
	rec := traces.NewErrorRecord(trace, pr, "EX-03 hôm nay? mật khẩu: "+pass, day, traces.Versions{Pack: "p"},
		contracts.Label(contracts.Infra("postgres"), errors.New("nối postgres://eais_app:"+pass+"@db:5432/eais thất bại")))
	rec.Steps[0].Input = []string{"Authorization: Bearer zzzzzzzzzzzzzzzz"} // bước cũng mang bí mật: đi qua cùng bộ làm sạch
	if _, err := s.SaveDecision(ctx, rec); err != nil {
		t.Fatal(err)
	}
	load := "load-" + string(uniq("l"))
	if _, err := s.SaveQuarantine(ctx, load, []pgstore.QuarantineRecord{{Key: "f#1", Reason: "unparseable", Detail: "password=" + pass}}); err != nil {
		t.Fatal(err)
	}
	for _, c := range []struct{ table, where, id string }{{"decision_records", "trace_id", trace}, {"quarantine_records", "load_id", load}} {
		var row string
		if err := adminConn.QueryRow(context.Background(), "SELECT row_to_json(t)::text FROM "+c.table+" t WHERE "+c.where+" = $1 LIMIT 1", c.id).Scan(&row); err != nil {
			t.Fatal(err)
		}
		for _, secret := range []string{pass, apikey, "zzzzzzzzzzzzzzzz"} {
			if strings.Contains(row, secret) {
				t.Errorf("%s: bí mật %q nằm trong DB: %s", c.table, secret, row)
			}
		}
		if !strings.Contains(row, traces.Masked) {
			t.Errorf("%s: không thấy dấu che trong %s", c.table, row)
		}
	}
	got, err := s.Decision(ctx, trace)
	if err != nil || !strings.Contains(got.Decision.Question, "EX-03 hôm nay?") || got.Decision.Attributes["entity_scope"] != "PRJ-A" {
		t.Errorf("phần không bí mật phải giữ nguyên: %+v (lỗi %v)", got.Decision, err)
	}
}

// Ràng buộc của chính DB (vai quản trị bỏ qua RLS nhưng không bỏ qua CHECK): không có lý do thì không thành bản ghi cách ly, vết
// phải là object / mảng, băm phải là 64 hex, usage không âm. Có sẵn một lượt hợp lệ để khoá ngoại của usage đạt — nhờ vậy chỉ CHECK
// mới có thể là thứ từ chối usage số âm.
func TestVetRangBuocDB(t *testing.T) {
	_, admin, _ := open(t)
	ctx := context.Background()
	conn := connect(t, admin)
	tenant := string(uniq("ta"))
	if _, err := conn.Exec(ctx, `INSERT INTO decision_records (tenant_id, trace_id, asked_at, principal_id, question, decision, content_sha256)
		VALUES ($1, 'seed-ok', now(), 'p', 'q', '{}', repeat('0', 64))`, tenant); err != nil {
		t.Fatalf("lượt hợp lệ làm nền: %v", err)
	}
	cols := "INSERT INTO decision_records (tenant_id, trace_id, asked_at, principal_id, question, decision, steps, content_sha256) VALUES ($1, "
	for name, q := range map[string]string{
		"cách ly không lý do":        "INSERT INTO quarantine_records (tenant_id, load_id, record_key, reason) VALUES ($1, 'l', 'k', '')",
		"cách ly không khoá":         "INSERT INTO quarantine_records (tenant_id, load_id, record_key, reason) VALUES ($1, 'l', '', 'r')",
		"vết rỗng trace_id":          cols + "'', now(), 'p', 'q', '{}', '[]', repeat('0', 64))",
		"decision không phải object": cols + "'t1', now(), 'p', 'q', '[]', '[]', repeat('0', 64))",
		"steps không phải mảng":      cols + "'t2', now(), 'p', 'q', '{}', '{}', repeat('0', 64))",
		"băm không phải 64 hex":      cols + "'t3', now(), 'p', 'q', '{}', '[]', 'xyz')",
		"băm hex nhưng chỉ 63 ký tự": cols + "'t4', now(), 'p', 'q', '{}', '[]', repeat('0', 63))",
		"băm hex dài 65 ký tự":       cols + "'t5', now(), 'p', 'q', '{}', '[]', repeat('0', 65))",
		"usage số âm":                "INSERT INTO usage_records (tenant_id, trace_id, model, calls, input_tokens, output_tokens, cache_read, cache_write) VALUES ($1, 'seed-ok', 'm', -1, 0, 0, 0, 0)",
		"usage model rỗng":           "INSERT INTO usage_records (tenant_id, trace_id, model, calls, input_tokens, output_tokens, cache_read, cache_write) VALUES ($1, 'seed-ok', '', 0, 0, 0, 0, 0)",
	} {
		_, err := conn.Exec(ctx, q, tenant)
		if err == nil || !strings.Contains(err.Error(), "violates check constraint") {
			t.Errorf("%s: muốn bị CHECK chặn, được %v", name, err)
		}
	}
	// và dòng hợp lệ đi qua được: CHECK không chặn oan
	if _, err := conn.Exec(ctx, "INSERT INTO usage_records (tenant_id, trace_id, model, calls, input_tokens, output_tokens, cache_read, cache_write) VALUES ($1, 'seed-ok', 'm', 0, 0, 0, 0, 0)", tenant); err != nil {
		t.Errorf("usage hợp lệ bị chặn oan: %v", err)
	}
}

// vetSchemaProblems đòi trạng thái ĐÍCH của migration 0003 trên một DB và trả các chỗ lệch (rỗng = đúng): xem appendOnlySchemaProblems.
func vetSchemaProblems(t *testing.T, conn *pgx.Conn) []string {
	t.Helper()
	return appendOnlySchemaProblems(t, conn, vetTables)
}

// appendOnlySchemaProblems đòi, cho từng bảng trong tables, trạng thái đích của một bảng chỉ-thêm có RLS (0003, 0006) và trả các chỗ lệch (rỗng = đúng): bảng có
// đủ, RLS ENABLE + FORCE, đúng MỘT chính sách tenant_isolation đọc GUC eais.tenant_id, vai ứng dụng có đúng SELECT + INSERT (không UPDATE / DELETE / TRUNCATE /
// REFERENCES / TRIGGER / MAINTAIN, không GRANT OPTION, không cả ở mức cột), PUBLIC không có quyền nào, vai ứng dụng không làm chủ bảng, khoá chính bắt đầu bằng tenant_id.
func appendOnlySchemaProblems(t *testing.T, conn *pgx.Conn, tables []string) []string {
	t.Helper()
	ctx := context.Background()
	var problems []string
	add := func(format string, args ...any) { problems = append(problems, fmt.Sprintf(format, args...)) }
	var version int
	if err := conn.QueryRow(ctx, "SELECT current_setting('server_version_num')::int").Scan(&version); err != nil {
		t.Fatal(err)
	}
	privs := []string{"SELECT", "INSERT", "UPDATE", "DELETE", "TRUNCATE", "REFERENCES", "TRIGGER"}
	if version >= 170000 {
		privs = append(privs, "MAINTAIN")
	}
	for _, table := range tables {
		var oid *string
		if err := conn.QueryRow(ctx, "SELECT to_regclass($1)::text", table).Scan(&oid); err != nil || oid == nil {
			add("%s: bảng không tồn tại (%v)", table, err)
			continue
		}
		for _, priv := range privs {
			want := priv == "SELECT" || priv == "INSERT"
			var got, grant bool
			if err := conn.QueryRow(ctx, "SELECT has_table_privilege('eais_app', $1, $2), has_table_privilege('eais_app', $1, $2 || ' WITH GRANT OPTION')", table, priv).Scan(&got, &grant); err != nil {
				t.Fatal(err)
			}
			if got != want || grant {
				add("eais_app %s trên %s: được %v (grant option %v), muốn %v / false", priv, table, got, grant, want)
			}
		}
		for _, priv := range []string{"SELECT", "INSERT", "UPDATE", "REFERENCES"} { // quyền mức cột
			want := priv == "SELECT" || priv == "INSERT"
			var got bool
			if err := conn.QueryRow(ctx, "SELECT has_any_column_privilege('eais_app', $1, $2)", table, priv).Scan(&got); err != nil {
				t.Fatal(err)
			}
			if got != want {
				add("eais_app %s ở mức cột trên %s: được %v, muốn %v", priv, table, got, want)
			}
		}
		var enabled, forced bool
		var owner string
		var public int
		if err := conn.QueryRow(ctx, `SELECT c.relrowsecurity, c.relforcerowsecurity, pg_get_userbyid(c.relowner),
			(SELECT count(*) FROM aclexplode(c.relacl) a WHERE a.grantee = 0)
			FROM pg_class c WHERE c.oid = $1::regclass`, table).Scan(&enabled, &forced, &owner, &public); err != nil {
			t.Fatal(err)
		}
		if !enabled || !forced {
			add("%s: RLS ENABLE = %v, FORCE = %v, muốn cả hai bật", table, enabled, forced)
		}
		if owner == "eais_app" {
			add("%s: vai ứng dụng là chủ bảng (chủ bảng đổi được chính sách)", table)
		}
		if public != 0 {
			add("%s: PUBLIC có %d quyền, muốn 0", table, public)
		}
		var pk string
		if err := conn.QueryRow(ctx, "SELECT pg_get_constraintdef(oid) FROM pg_constraint WHERE conrelid = $1::regclass AND contype = 'p'", table).Scan(&pk); err != nil {
			add("%s: không có khoá chính (%v)", table, err)
		} else if !strings.HasPrefix(pk, "PRIMARY KEY (tenant_id,") {
			add("%s: khoá chính %q không bắt đầu bằng tenant_id (mọi bảng theo tenant có tenant_id trong khoá)", table, pk)
		}
		var n int
		var name, cmd, qual, check string
		if err := conn.QueryRow(ctx, `SELECT count(*) OVER (), policyname, cmd, coalesce(qual, ''), coalesce(with_check, '') FROM pg_policies
			WHERE schemaname = current_schema() AND tablename = $1`, table).Scan(&n, &name, &cmd, &qual, &check); errors.Is(err, pgx.ErrNoRows) {
			add("%s: không có chính sách nào", table)
			continue
		} else if err != nil {
			t.Fatalf("%s: chính sách: %v", table, err)
		}
		if n != 1 || name != "tenant_isolation" || cmd != "ALL" || !strings.Contains(qual, "eais.tenant_id") || !strings.Contains(check, "eais.tenant_id") {
			add("%s: chính sách: %d cái, %q cmd %s, qual %q, check %q", table, n, name, cmd, qual, check)
		}
	}
	return problems
}

// checkVetSchema báo mọi chỗ lệch của vetSchemaProblems thành lỗi test.
func checkVetSchema(t *testing.T, conn *pgx.Conn) {
	t.Helper()
	for _, p := range vetSchemaProblems(t, conn) {
		t.Error(p)
	}
}

// Trạng thái đích của 0003 trên DB dùng chung của các test (đã qua mọi lần Migrate của bộ test).
func TestVetQuyenVaRLS(t *testing.T) {
	_, admin, _ := open(t)
	checkVetSchema(t, connect(t, admin))
}

// freshDB tạo một database trống trong cùng cụm PG và trả DSN quản trị / ứng dụng trỏ vào nó; dọn khi test xong.
func freshDB(t *testing.T, admin, app string) (string, string) {
	t.Helper()
	au, err1 := url.Parse(admin)
	pu, err2 := url.Parse(app)
	if err1 != nil || err2 != nil || au.User == nil || pu.User == nil {
		t.Skipf("DSN không ở dạng URL có tên vai (docs/ops/README.md) — không dựng được database trống: %v %v", err1, err2)
	}
	name := fmt.Sprintf("eais_vet_%d_%d", os.Getpid(), time.Now().UnixNano()%1_000_000)
	ctx := context.Background()
	conn := connect(t, admin)
	if _, err := conn.Exec(ctx, "CREATE DATABASE "+name); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if _, err := conn.Exec(context.Background(), "DROP DATABASE IF EXISTS "+name+" WITH (FORCE)"); err != nil {
			t.Errorf("dọn database %s: %v", name, err)
		}
	})
	au.Path, pu.Path = "/"+name, "/"+name
	return au.String(), pu.String()
}

// Migration chạy trên PG TRỐNG, chạy lại lần hai không lỗi (Migrate bỏ qua tệp đã áp), và chính tệp 0003 chạy lại trần cũng không
// lỗi, không đổi trạng thái (kể cả khi schema_migrations mất dòng). Sau đó vai ứng dụng dùng được ngay.
func TestVetMigration0003TrenDBTrong(t *testing.T) {
	_, admin, app := open(t)
	fa, fp := freshDB(t, admin, app)
	ctx := context.Background()
	for i := 1; i <= 2; i++ {
		if err := pgstore.Migrate(ctx, fa); err != nil {
			t.Fatalf("Migrate lần %d trên DB trống: %v", i, err)
		}
	}
	conn := connect(t, fa)
	var versions []string
	rows, err := conn.Query(ctx, "SELECT version FROM schema_migrations ORDER BY version")
	if err != nil {
		t.Fatal(err)
	}
	for rows.Next() {
		var v string
		if err := rows.Scan(&v); err != nil {
			t.Fatal(err)
		}
		versions = append(versions, v)
	}
	rows.Close()
	if got := strings.Join(versions, ","); got != "0001_win.sql,0002_dinh_danh_eais.sql,0003_kho_vet_ben.sql,0004_phieu_chenh_lech.sql,0005_audit_va_tenant.sql,0006_so_nap_du_lieu.sql,0007_tai_lieu_phien_ban.sql,0008_vector_tai_lieu.sql,0009_kho_bao_cao_ngay.sql,0010_tenant_khong_rong.sql" {
		t.Errorf("schema_migrations sau hai lần Migrate: %s", got)
	}
	checkVetSchema(t, conn)

	sql, err := db.Migrations.ReadFile("migrations/0003_kho_vet_ben.sql")
	if err != nil {
		t.Fatal(err)
	}
	for i := 1; i <= 2; i++ {
		if _, err := conn.Exec(ctx, string(sql)); err != nil {
			t.Fatalf("chạy lại trần tệp 0003 lần %d: %v", i, err)
		}
	}
	checkVetSchema(t, conn)

	s, err := pgstore.Open(ctx, fp)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	a := uniq("ta")
	trace, _ := vetSeed(t, s, a)
	if _, err := s.Decision(contracts.WithTenant(ctx, a), trace); err != nil {
		t.Errorf("đọc lại trên DB vừa dựng: %v", err)
	}
}

// Khoá chính có tenant_id: hai tenant dùng CÙNG trace id (hoặc mã lần nạp, hoặc model) là hai bản ghi riêng, không đụng nhau, không
// bên nào thấy bên kia. Nếu khoá chỉ là trace_id thì tenant ghi sau bị chặn (hoặc tệ hơn: bị bỏ qua im lặng) vì một dòng mà nó không
// được thấy.
func TestVetCungKhoaHaiTenantLaHaiBanGhi(t *testing.T) {
	s, admin, _ := open(t)
	a, b := uniq("ta"), uniq("tb")
	ctxA, ctxB := contracts.WithTenant(context.Background(), a), contracts.WithTenant(context.Background(), b)
	trace, load := uniqTrace(), "load-chung-"+string(uniq("l"))
	for _, tn := range []struct {
		ctx    context.Context
		tenant contracts.TenantID
		calls  int
	}{{ctxA, a, 1}, {ctxB, b, 9}} {
		if ok, err := s.SaveDecision(tn.ctx, vetRecord(tn.tenant, trace)); err != nil || !ok {
			t.Fatalf("%s ghi trace id dùng chung: inserted=%v lỗi=%v", tn.tenant, ok, err)
		}
		if ok, err := s.SaveUsage(tn.ctx, pgstore.UsageRecord{TraceID: trace, Model: "m1", Calls: tn.calls}); err != nil || !ok {
			t.Fatalf("%s ghi usage dùng chung: inserted=%v lỗi=%v", tn.tenant, ok, err)
		}
		if n, err := s.SaveQuarantine(tn.ctx, load, []pgstore.QuarantineRecord{{Key: "f#1", Reason: "missing_value", Detail: string(tn.tenant)}}); err != nil || n != 1 {
			t.Fatalf("%s ghi cách ly dùng chung mã lần nạp: %d %v", tn.tenant, n, err)
		}
	}
	for _, tn := range []struct {
		ctx    context.Context
		tenant contracts.TenantID
		calls  int
	}{{ctxA, a, 1}, {ctxB, b, 9}} {
		got, err := s.Decision(tn.ctx, trace)
		if err != nil || got.Decision.Tenant != string(tn.tenant) {
			t.Errorf("%s đọc trace id dùng chung: tenant của bản ghi %q (lỗi %v)", tn.tenant, got.Decision.Tenant, err)
		}
		if us, err := s.Usage(tn.ctx, trace); err != nil || len(us) != 1 || us[0].Calls != tn.calls {
			t.Errorf("%s đọc usage dùng chung: %+v (lỗi %v), muốn đúng số lời gọi %d của mình", tn.tenant, us, err, tn.calls)
		}
		if qs, err := s.Quarantine(tn.ctx, load); err != nil || len(qs) != 1 || qs[0].Detail != string(tn.tenant) {
			t.Errorf("%s đọc cách ly dùng chung: %+v (lỗi %v)", tn.tenant, qs, err)
		}
	}
	adminConn := connect(t, admin)
	if n := visible(t, adminConn, "", "decision_records", "trace_id = $1", trace); n != 2 {
		t.Errorf("DB phải có 2 bản ghi cho trace id dùng chung (mỗi tenant một), được %d", n)
	}
}

// Migration 0003 chạy lại ĐƯA VỀ trạng thái đích dù DB đã trôi khỏi nó (quyền cấp thêm tay, chính sách bị sửa hoặc xoá, FORCE bị tắt).
// Mỗi kiểu trôi kiểm hai điều: bộ kiểm trạng thái (vetSchemaProblems) THẤY nó — ca âm của chính bộ kiểm — và chạy lại tệp 0003
// xoá được nó.
func TestVetMigration0003HoiTuVeTrangThaiDich(t *testing.T) {
	_, admin, app := open(t)
	fa, _ := freshDB(t, admin, app)
	ctx := context.Background()
	if err := pgstore.Migrate(ctx, fa); err != nil {
		t.Fatal(err)
	}
	conn := connect(t, fa)
	if p := vetSchemaProblems(t, conn); len(p) != 0 {
		t.Fatalf("DB mới dựng phải đúng trạng thái đích: %v", p)
	}
	sql, err := db.Migrations.ReadFile("migrations/0003_kho_vet_ben.sql")
	if err != nil {
		t.Fatal(err)
	}
	for _, drift := range []string{
		"GRANT UPDATE ON decision_records TO eais_app",
		"GRANT DELETE ON usage_records TO eais_app",
		"GRANT TRUNCATE ON quarantine_records TO eais_app",
		"GRANT SELECT ON usage_records TO PUBLIC",
		"ALTER POLICY tenant_isolation ON usage_records USING (true)",
		"ALTER POLICY tenant_isolation ON decision_records WITH CHECK (true)",
		"ALTER TABLE decision_records NO FORCE ROW LEVEL SECURITY",
		"ALTER TABLE quarantine_records DISABLE ROW LEVEL SECURITY",
		"DROP POLICY tenant_isolation ON quarantine_records",
	} {
		if _, err := conn.Exec(ctx, drift); err != nil {
			t.Fatalf("%s: %v", drift, err)
		}
		if p := vetSchemaProblems(t, conn); len(p) == 0 {
			t.Errorf("bộ kiểm trạng thái không thấy chỗ trôi %q", drift)
		}
		if _, err := conn.Exec(ctx, string(sql)); err != nil {
			t.Fatalf("chạy lại 0003 sau %q: %v", drift, err)
		}
		if p := vetSchemaProblems(t, conn); len(p) != 0 {
			t.Errorf("chạy lại 0003 không đưa về trạng thái đích sau %q: %v", drift, p)
		}
	}
}

// R1-07. Mã tra cứu (trace id, mã lần nạp) mà làm sạch ĐỔI (có NUL, UTF-8 hỏng, hoặc trùng mẫu bí mật) sẽ lưu thành mã khác với nơi
// gọi cấp rồi đọc lại "không có"; NUL còn làm lệnh PG hỏng bằng lỗi "thử lại được" mà không bao giờ thành công. Phải từ chối ngay ở
// tầng Go, KHÔNG chạm DB — Store rỗng (không có pool) làm mọi lệnh chạm DB thành panic, nên ca nào chạm DB là đỏ.
func TestVetMaTraCuuBiLamSachDoiThiTuChoi(t *testing.T) {
	s := &pgstore.Store{}
	a := uniq("ta")
	ctx := contracts.WithTenant(context.Background(), a)
	for _, id := range []string{"tra\x00ce", "tra\xffce", "pwd=abc123xyz", "token: aaaaaaaa"} {
		func() {
			defer func() {
				if r := recover(); r != nil {
					t.Errorf("mã %q: chạm DB (panic %v) thay vì từ chối từ đầu", id, r)
				}
			}()
			if _, err := s.SaveDecision(ctx, vetRecord(a, id)); contracts.FaultOf(err) != contracts.FaultInput {
				t.Errorf("SaveDecision trace_id %q: muốn lỗi %s, được %v", id, contracts.FaultInput, err)
			}
			if _, err := s.Decision(ctx, id); !errors.Is(err, pgstore.ErrNotFound) {
				t.Errorf("Decision trace_id %q: muốn ErrNotFound (không thể có bản ghi mang mã này), được %v", id, err)
			}
			if _, err := s.SaveUsage(ctx, pgstore.UsageRecord{TraceID: id, Model: "m"}); contracts.FaultOf(err) != contracts.FaultInput {
				t.Errorf("SaveUsage trace_id %q: muốn lỗi %s, được %v", id, contracts.FaultInput, err)
			}
			if us, err := s.Usage(ctx, id); err != nil || len(us) != 0 {
				t.Errorf("Usage trace_id %q: muốn rỗng không lỗi, được %v %v", id, us, err)
			}
			if _, err := s.SaveQuarantine(ctx, id, []pgstore.QuarantineRecord{{Key: "k#1", Reason: "r"}}); contracts.FaultOf(err) != contracts.FaultInput {
				t.Errorf("SaveQuarantine mã lần nạp %q: muốn lỗi %s, được %v", id, contracts.FaultInput, err)
			}
			if qs, err := s.Quarantine(ctx, id); err != nil || len(qs) != 0 {
				t.Errorf("Quarantine mã lần nạp %q: muốn rỗng không lỗi, được %v %v", id, qs, err)
			}
		}()
	}
}

// R1-09. Lượt KHÔNG xác thực được tenant (token sai / thiếu) không có dòng vết theo tenant: kho vết bắt buộc tenant (RLS) nên từ chối
// ngay ở tầng Go bằng lỗi chính sách, KHÔNG chạm DB. Nơi gọi (cmd/eais) ghi những lượt này vào bộ đếm / nhật ký riêng NGOÀI kho vết —
// chỉ tiêu "mọi lượt có vết lưu bền" tính trên lượt ĐÃ có tenant (xem đầu tệp vet.go).
func TestVetLuotKhongTenantKhongCoDongVet(t *testing.T) {
	s := &pgstore.Store{} // không có pool: chạm DB là panic
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("lượt không tenant chạm DB (panic %v) thay vì bị từ chối ở tầng Go", r)
		}
	}()
	rec := vetRecord(uniq("ta"), uniqTrace())
	if _, err := s.SaveDecision(context.Background(), rec); contracts.FaultOf(err) != contracts.FaultPolicy {
		t.Errorf("SaveDecision không tenant: muốn %s, được %v", contracts.FaultPolicy, err)
	}
	pr := contracts.Principal{ID: "u-x", TenantID: "", Kind: contracts.PrincipalUser}
	errRec := traces.NewErrorRecord(uniqTrace(), pr, "câu hỏi", day, traces.Versions{}, contracts.Errorf(contracts.FaultPolicy, "không xác thực được tenant"))
	if _, err := s.SaveDecision(context.Background(), errRec); contracts.FaultOf(err) != contracts.FaultPolicy {
		t.Errorf("lượt lỗi không tenant cũng không được ghi: muốn %s, được %v", contracts.FaultPolicy, err)
	}
}

// usedTenants là các tenant mà uniq() đã cấp trong lượt chạy này. eais_app chỉ có SELECT + INSERT trên các bảng vết nên test không tự
// dọn bằng vai ứng dụng được; TestMain dọn bằng vai quản trị sau khi chạy xong, để PG sống lâu không tích dòng của test (R1-08).
// Đặt EAIS_KEEP_TEST_ROWS=1 khi cần giữ dòng lại để soi bằng tay.
var usedTenants sync.Map

func TestMain(m *testing.M) {
	code := m.Run()
	cleanupVetRows()
	os.Exit(code)
}

func cleanupVetRows() {
	admin, _ := pgstore.EnvDSN()
	if admin == "" || os.Getenv("EAIS_KEEP_TEST_ROWS") != "" {
		return
	}
	var tenants []string
	usedTenants.Range(func(k, _ any) bool { tenants = append(tenants, k.(string)); return true })
	if len(tenants) == 0 {
		return
	}
	ctx := context.Background()
	conn, err := pgx.Connect(ctx, admin)
	if err != nil {
		fmt.Fprintln(os.Stderr, "dọn dòng vết của test: không kết nối được:", err)
		return
	}
	defer conn.Close(ctx)
	for _, table := range vetTablesChildFirst {
		if _, err := conn.Exec(ctx, "DELETE FROM "+table+" WHERE tenant_id = ANY($1)", tenants); err != nil {
			fmt.Fprintln(os.Stderr, "dọn dòng vết của test:", table, err)
		}
	}
}

// vetTablesChildFirst: usage_records có khoá ngoại tới decision_records nên xoá bảng con trước.
// load_records: mọi Load của test (seed) để lại một dòng sổ nạp (migration 0006).
var vetTablesChildFirst = []string{"usage_records", "quarantine_records", "decision_records", "load_records"}
