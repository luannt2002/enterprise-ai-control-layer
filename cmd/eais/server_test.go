package main

import (
	"context"
	"errors"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/google/uuid"
	"github.com/luannt2002/enterprise-ai-control-layer/core/contracts"
	"github.com/luannt2002/enterprise-ai-control-layer/data/entityresolution"
	"github.com/luannt2002/enterprise-ai-control-layer/observability/traces"
)

// Bộ test của lớp HTTP. MỘT bộ test chạy trên HAI kho: bộ nhớ (memStore, luôn chạy — kể cả `go test` không có PG) và PG thật (chỉ khi có
// EAIS_PG_*; CI có PG và cấm SKIP). Cùng một bộ nên điều test khẳng định là hành vi của ứng dụng, không phải của riêng một kho.

func TestServer(t *testing.T) {
	t.Run("bo-nho", func(t *testing.T) {
		serverSuite(t, func(t *testing.T) *env {
			ms := newMemStore()
			e := newEnv(t, ms)
			e.decisionRows = ms.decisionCount
			return e
		})
	})
	t.Run("pg", func(t *testing.T) {
		st, admin := openPG(t)
		serverSuite(t, func(t *testing.T) *env {
			e := newEnv(t, st)
			e.decisionRows = func(tenant string) int { return countRows(t, admin, "decision_records", tenant, "") }
			return e
		})
	})
}

func (e *env) principal(token string) contracts.Principal { return e.srv.principals[token] }

func (e *env) tenantCtx(tenant string) context.Context {
	return contracts.WithTenant(context.Background(), contracts.TenantID(tenant))
}

// getTrace đọc vết và giải mã; fail nếu không phải 200.
func (e *env) getTrace(token, id string) traceResponse {
	e.t.Helper()
	r := e.trace(token, id)
	if r.status != http.StatusOK {
		e.t.Fatalf("GET /v1/trace/%s = %d, muốn 200: %s", id, r.status, r.body)
	}
	return decodeAs[traceResponse](e.t, r)
}

func serverSuite(t *testing.T, mk func(t *testing.T) *env) {
	t.Run("healthz", func(t *testing.T) {
		e := mk(t)
		r := e.do(http.MethodGet, "/healthz", "", "", nil)
		if r.status != http.StatusOK || !strings.Contains(string(r.body), `"ok"`) {
			t.Fatalf("healthz = %d %s", r.status, r.body)
		}
	})

	// CHO PHÉP: lượt hợp lệ trả câu trả lời + trace id; vết đọc lại được có đủ 11 bước; câu trả lời qua HTTP GIỐNG HỆT Handle gọi thẳng.
	t.Run("ask-hop-le-va-vet", func(t *testing.T) {
		e := mk(t)
		r := e.ask(tokDirector, qSQL)
		if r.status != http.StatusOK {
			t.Fatalf("ask = %d: %s", r.status, r.body)
		}
		resp := decodeAs[contracts.AskResponse](t, r)
		if _, err := uuid.Parse(resp.TraceID); err != nil || r.header.Get("X-Trace-Id") != resp.TraceID {
			t.Fatalf("trace_id %q không phải uuid hoặc khác X-Trace-Id %q", resp.TraceID, r.header.Get("X-Trace-Id"))
		}
		if !strings.Contains(resp.Answer, "3.850") {
			t.Errorf("câu trả lời không chứa số có thẩm quyền 3.850: %q", resp.Answer)
		}
		pr := e.principal(tokDirector)
		direct, err := e.app.ask.Load().Handle(context.Background(), pr, contracts.AskRequest{ConversationID: "c-1", MessageID: "m-1", Text: qSQL})
		if err != nil {
			t.Fatal(err)
		}
		if resp.Answer != direct.Response.Answer || resp.Route != direct.Response.Route || resp.Outcome != direct.Response.Outcome || resp.Intent != direct.Response.Intent {
			t.Errorf("HTTP đổi câu trả lời:\n HTTP:   %q %s %s\n trực tiếp: %q %s %s", resp.Answer, resp.Route, resp.Outcome, direct.Response.Answer, direct.Response.Route, direct.Response.Outcome)
		}
		tr := e.getTrace(tokDirector, resp.TraceID)
		d := tr.Record.Decision
		if d.TraceID != resp.TraceID || d.Tenant != e.tenantA || d.Principal != "u-director" || d.Question != qSQL {
			t.Errorf("Decision của vết lệch: %+v", d)
		}
		if d.DataVersion != e.app.dataVersionNow() || d.PackVersion != e.app.rt.PackVersion {
			t.Errorf("vết thiếu phiên bản dữ liệu / pack: %q %q", d.DataVersion, d.PackVersion)
		}
		var stages []string
		for _, s := range tr.Record.Steps {
			if s.TraceID != resp.TraceID {
				t.Errorf("bước %s mang trace_id khác: %q", s.Stage, s.TraceID)
			}
			stages = append(stages, s.Stage)
		}
		if got := strings.Join(stages, ","); got != "S0,S1,S2,S3,S4,S5,S6,S7,S8,S9,S10" {
			t.Errorf("các bước của vết = %s", got)
		}
		if len(tr.Usage) != 1 || tr.Usage[0].Model != "extractive" || tr.Usage[0].Calls != 0 || tr.Usage[0].InputTokens != 0 || tr.Usage[0].OutputTokens != 0 {
			t.Errorf("route số liệu không gọi model: muốn usage extractive 0 lời gọi / 0 token, được %+v", tr.Usage)
		}
		tab := e.do(http.MethodGet, "/v1/trace/"+resp.TraceID+"?format=table", tokDirector, "", nil)
		if tab.status != http.StatusOK || !strings.HasPrefix(tab.header.Get("Content-Type"), "text/plain") {
			t.Fatalf("format=table = %d %s", tab.status, tab.header.Get("Content-Type"))
		}
		for _, want := range []string{"VẾT " + resp.TraceID, "S0", "S5", "S10", "USAGE MODEL", "extractive: 0 lời gọi"} {
			if !strings.Contains(string(tab.body), want) {
				t.Errorf("bảng chữ thiếu %q", want)
			}
		}
	})

	// Route tri thức gọi extractive đúng MỘT lần, và usage ghi đúng như vậy (0 token).
	t.Run("ask-rag-usage", func(t *testing.T) {
		e := mk(t)
		r := e.ask(tokEng, qRAG)
		resp := decodeAs[contracts.AskResponse](t, r)
		if r.status != http.StatusOK || resp.Route != contracts.RouteRAG || !strings.Contains(resp.Answer, "24 giờ") {
			t.Fatalf("ask RAG = %d %s %q", r.status, resp.Route, resp.Answer)
		}
		u := e.getTrace(tokEng, resp.TraceID).Usage
		if len(u) != 1 || u[0].Model != "extractive" || u[0].Calls != 1 || u[0].InputTokens != 0 || u[0].OutputTokens != 0 {
			t.Errorf("muốn usage extractive 1 lời gọi, 0 token; được %+v", u)
		}
	})

	// CA ÂM: thiếu / sai / lạ token đều 401 cùng một phản hồi, không dòng vết nào được ghi, bộ đếm đếm đúng, và token không lọt vào log.
	t.Run("khong-xac-thuc", func(t *testing.T) {
		e := mk(t)
		before := e.decisionRows(e.tenantA) + e.decisionRows(e.tenantB)
		secret := "unknown-token-b7c1f0d2"
		headers := []string{"", "Basic dXNlcjpwYXNz", "Bearer", "Bearer ", "Bearer " + secret, "Bearer " + tokDirector + " extra", "Token " + tokDirector}
		var bodies []string
		n := 0
		for _, h := range headers {
			for _, path := range []string{"/v1/ask", "/v1/ingest"} {
				req, _ := http.NewRequest(http.MethodPost, e.ts.URL+path, strings.NewReader(`{"text":"x","conversation_id":"c","message_id":"m"}`))
				req.Header.Set("Content-Type", "application/json")
				if h != "" {
					req.Header.Set("Authorization", h)
				}
				res, err := http.DefaultClient.Do(req)
				if err != nil {
					t.Fatal(err)
				}
				res.Body.Close()
				if res.StatusCode != http.StatusUnauthorized || res.Header.Get("WWW-Authenticate") == "" {
					t.Errorf("%s với Authorization %q = %d, muốn 401 kèm WWW-Authenticate", path, h, res.StatusCode)
				}
				n++
			}
		}
		r := e.do(http.MethodGet, "/v1/trace/"+uuid.NewString(), secret, "", nil)
		n++
		bodies = append(bodies, string(r.body))
		r2 := e.do(http.MethodGet, "/v1/trace/"+uuid.NewString(), "", "", nil)
		n++
		bodies = append(bodies, string(r2.body))
		if r.status != 401 || r2.status != 401 || bodies[0] != bodies[1] {
			t.Errorf("token lạ và thiếu token phải cho CÙNG phản hồi 401 (không để dò token): %d %q / %d %q", r.status, bodies[0], r2.status, bodies[1])
		}
		if got := e.srv.cnt.Unauthenticated.Load(); got != int64(n) {
			t.Errorf("bộ đếm Unauthenticated = %d, muốn %d", got, n)
		}
		if after := e.decisionRows(e.tenantA) + e.decisionRows(e.tenantB); after != before {
			t.Errorf("lượt chưa xác thực làm đổi số dòng vết: %d → %d", before, after)
		}
		if strings.Contains(e.serverLog(), secret) {
			t.Error("token lọt vào log của server")
		}
	})

	// CA ÂM + bất biến "tenant không lấy từ thân": mọi thân sai đều bị từ chối, vẫn có vết (lỗi đầu vào, đúng người hỏi, đúng tenant của
	// TOKEN), và đối chứng dương: cùng câu hỏi hợp lệ khi KHÔNG có trường tenant thì được trả lời.
	t.Run("than-sai", func(t *testing.T) {
		e := mk(t)
		valid := func(extra string) []byte {
			return []byte(`{"text":"` + qSQL + `","conversation_id":"c","message_id":"m"` + extra + `}`)
		}
		huge := []byte(`{"text":"` + strings.Repeat("a", maxAskBody+10) + `","conversation_id":"c","message_id":"m"}`)
		cases := []struct {
			name, ctype string
			body        []byte
			status      int
			stage       string // bước ghi trong vết
			errHas      string
		}{
			{"tenant", "application/json", valid(`,"tenant":"` + e.tenantB + `"`), 400, "S0", "tenant"},
			{"tenant_id", "application/json", valid(`,"tenant_id":"` + e.tenantB + `"`), 400, "S0", "tenant"},
			{"Tenant-hoa", "application/json", valid(`,"TENANT":"x"`), 400, "S0", "tenant"},
			{"principal", "application/json", valid(`,"principal":{"id":"u-director","roles":["director"]}`), 400, "S0", "principal"},
			{"roles", "application/json", valid(`,"roles":["director"]`), 400, "S0", "roles"},
			{"truong-la", "application/json", valid(`,"x":1`), 400, "S0", "trường lạ"},
			{"json-hong", "application/json", []byte(`{"text":`), 400, "S0", "JSON"},
			{"than-rong", "application/json", nil, 400, "S0", "rỗng"},
			{"noi-dung-thua", "application/json", append(valid(""), []byte(` {"x":1}`)...), 400, "S0", "thừa"},
			{"sai-content-type", "text/plain", valid(""), 415, "S0", "application/json"},
			{"qua-lon", "application/json", huge, 413, "S0", "quá"},
			{"text-rong", "application/json", []byte(`{"text":"  ","conversation_id":"c","message_id":"m"}`), 400, "S?", "text rỗng"},
			{"thieu-message-id", "application/json", []byte(`{"text":"x","conversation_id":"c"}`), 400, "S?", "message_id"},
			{"text-qua-dai", "application/json", []byte(`{"text":"` + strings.Repeat("a", 2001) + `","conversation_id":"c","message_id":"m"}`), 400, "S?", "quá"},
		}
		for _, c := range cases {
			r := e.do(http.MethodPost, "/v1/ask", tokDirector, c.ctype, c.body)
			if r.status != c.status {
				t.Errorf("%s: status %d, muốn %d: %s", c.name, r.status, c.status, r.body)
				continue
			}
			eb := decodeAs[errBody](t, r)
			if eb.Fault != contracts.FaultInput || eb.TraceID == "" || !strings.Contains(eb.Error, c.errHas) {
				t.Errorf("%s: thân lỗi %+v, muốn fault input, có trace_id, nêu %q", c.name, eb, c.errHas)
				continue
			}
			rec := e.getTrace(tokDirector, eb.TraceID).Record
			if rec.Fault != contracts.FaultInput || rec.Decision.Tenant != e.tenantA || rec.Decision.Principal != "u-director" ||
				len(rec.Steps) != 1 || rec.Steps[0].Stage != c.stage || rec.Steps[0].Status != traces.StatusError {
				t.Errorf("%s: vết của lượt lỗi lệch: fault=%q tenant=%q principal=%q steps=%+v", c.name, rec.Fault, rec.Decision.Tenant, rec.Decision.Principal, rec.Steps)
			}
		}
		if r := e.do(http.MethodPost, "/v1/ask", tokDirector, "application/json", valid("")); r.status != 200 {
			t.Errorf("đối chứng dương: cùng câu hỏi không có trường tenant phải được trả lời, được %d: %s", r.status, r.body)
		}
		if got := e.decisionRows(e.tenantB); got != 0 {
			t.Errorf("thân mang tenant B mà tenant B có %d dòng vết: tenant chỉ được lấy từ token", got)
		}
	})

	// Cô lập tenant: vết của tenant A không đọc được từ tenant B (và 404 y hệt "không có"); principal B không hỏi được runtime của A;
	// lượt lỗi của B nằm ở tenant B.
	t.Run("co-lap-tenant", func(t *testing.T) {
		e := mk(t)
		a := decodeAs[contracts.AskResponse](t, e.ask(tokDirector, qSQL))
		cross := e.trace(tokDirectorB, a.TraceID)
		none := e.trace(tokDirectorB, uuid.NewString())
		if cross.status != 404 || none.status != 404 || string(cross.body) != string(none.body) {
			t.Errorf("tenant B đọc vết của A: %d %s; không có: %d %s — muốn cùng 404 không phân biệt", cross.status, cross.body, none.status, none.body)
		}
		rb := e.ask(tokDirectorB, qSQL)
		eb := decodeAs[errBody](t, rb)
		if rb.status != 403 || eb.Fault != contracts.FaultPolicy || eb.TraceID == "" {
			t.Fatalf("principal tenant B hỏi runtime của A = %d %+v, muốn 403 policy có vết", rb.status, eb)
		}
		own := e.getTrace(tokDirectorB, eb.TraceID)
		if own.Record.Decision.Tenant != e.tenantB || own.Record.Fault != contracts.FaultPolicy {
			t.Errorf("vết lượt lỗi của B: tenant %q fault %q", own.Record.Decision.Tenant, own.Record.Fault)
		}
		if r := e.trace(tokDirector, eb.TraceID); r.status != 404 {
			t.Errorf("tenant A đọc vết của B = %d, muốn 404", r.status)
		}
		if e.decisionRows(e.tenantA) != 1 || e.decisionRows(e.tenantB) != 1 {
			t.Errorf("số dòng vết A=%d B=%d, muốn 1 và 1", e.decisionRows(e.tenantA), e.decisionRows(e.tenantB))
		}
		if r := e.post("/v1/ingest", tokDirectorB, map[string]string{}); r.status != 403 {
			t.Errorf("director của tenant B nạp lại dữ liệu = %d, muốn 403 (không phải tenant của runtime)", r.status)
		}
	})

	// Quyền đọc vết trong cùng tenant: chính người hỏi, hoặc người TOÀN QUYỀN dữ liệu theo policy (director); accountant có scope all
	// nhưng thiếu documents.read nên không được đọc vết của người khác. Lượt bị từ chối vì quyền (REJECT) vẫn có vết, S5 lệch.
	t.Run("quyen-doc-vet", func(t *testing.T) {
		e := mk(t)
		eng := decodeAs[contracts.AskResponse](t, e.ask(tokEng, qSQL))
		dir := decodeAs[contracts.AskResponse](t, e.ask(tokDirector, qSQL))
		for _, c := range []struct {
			who, id string
			want    int
		}{
			{tokEng, eng.TraceID, 200}, {tokDirector, eng.TraceID, 200},
			{tokAccountant, eng.TraceID, 404}, {tokDriver, eng.TraceID, 404},
			{tokEng, dir.TraceID, 404}, {tokDirector, dir.TraceID, 200}, {tokAccountant, dir.TraceID, 404},
		} {
			if r := e.trace(c.who, c.id); r.status != c.want {
				t.Errorf("%s đọc vết %s = %d, muốn %d", c.who, c.id, r.status, c.want)
			}
		}
		rj := e.ask(tokDriver, qSQL)
		rjResp := decodeAs[contracts.AskResponse](t, rj)
		if rj.status != 200 || rjResp.Route != contracts.RouteReject {
			t.Fatalf("driver hỏi số liệu = %d route %s, muốn 200 REJECT", rj.status, rjResp.Route)
		}
		rec := e.getTrace(tokDriver, rjResp.TraceID).Record
		var s5 traces.Step
		for _, s := range rec.Steps {
			if s.Stage == traces.StageRoute {
				s5 = s
			}
		}
		if rec.Decision.Route != string(contracts.RouteReject) || s5.Status != traces.StatusDeviant {
			t.Errorf("lượt bị từ chối vì quyền: route %q, S5 %q — muốn REJECT và S5 lệch", rec.Decision.Route, s5.Status)
		}
	})

	// Nạp lại: chỉ vai toàn quyền, thân phải rỗng; nạp lại cùng dữ liệu KHÔNG nhân đôi bản ghi cách ly (cùng mã lần nạp); đổi dữ liệu thì
	// mã mới, bản ghi cách ly của lần cũ giữ nguyên, và vết sau đó mang phiên bản dữ liệu mới.
	t.Run("ingest", func(t *testing.T) {
		e := mk(t)
		if e.load.Quarantined == 0 {
			t.Fatal("dữ liệu seed phải có bản ghi cách ly để kiểm không nhân đôi")
		}
		rows := func(load string) int {
			q, err := e.st.Quarantine(e.tenantCtx(e.tenantA), load)
			if err != nil {
				t.Fatal(err)
			}
			return len(q)
		}
		if got := rows(e.load.LoadID); got != e.load.Quarantined {
			t.Fatalf("sau nạp lần đầu có %d dòng cách ly, muốn %d", got, e.load.Quarantined)
		}
		for _, body := range [][]byte{nil, []byte("{}"), []byte(" {} \n")} {
			r := e.do(http.MethodPost, "/v1/ingest", tokDirector, "", body)
			rep := decodeAs[ingestReport](t, r)
			if r.status != 200 || rep.LoadID != e.load.LoadID || rep.NewQuarantine != 0 || rep.Quarantined != e.load.Quarantined || rep.Observations != e.load.Observations {
				t.Fatalf("nạp lại cùng dữ liệu = %d %+v, muốn cùng mã lần nạp %s, 0 cách ly mới", r.status, rep, e.load.LoadID)
			}
		}
		if got := rows(e.load.LoadID); got != e.load.Quarantined {
			t.Errorf("nạp lại 3 lần: %d dòng cách ly, muốn vẫn %d (không nhân đôi)", got, e.load.Quarantined)
		}
		for _, bad := range []string{`{"path":"/etc"}`, `{"tenant":"x"}`, `x`, `[]`} {
			if r := e.do(http.MethodPost, "/v1/ingest", tokDirector, "application/json", []byte(bad)); r.status != 400 {
				t.Errorf("thân %q = %d, muốn 400 (nạp lại không nhận đường dẫn / tham số)", bad, r.status)
			}
		}
		for _, who := range []string{tokEng, tokAccountant, tokDriver, tokDirectorB} {
			if r := e.post("/v1/ingest", who, map[string]string{}); r.status != 403 {
				t.Errorf("%s nạp lại = %d, muốn 403", who, r.status)
			}
		}
		if got := e.srv.cnt.IngestDenied.Load(); got != 4 { // eng, accountant, driver (không toàn quyền) và director của tenant B (không phải tenant của runtime)
			t.Errorf("IngestDenied = %d, muốn 4", got)
		}

		// đổi dữ liệu: thêm một tệp vào thư mục nguồn ⇒ phiên bản dữ liệu khác ⇒ mã lần nạp khác
		if err := os.WriteFile(filepath.Join(e.dataRoot, e.tenantA, "ghi-chu.txt"), []byte("thêm một tệp"), 0o644); err != nil {
			t.Fatal(err)
		}
		r := e.post("/v1/ingest", tokDirector, map[string]string{})
		rep2 := decodeAs[ingestReport](t, r)
		if r.status != 200 || rep2.LoadID == e.load.LoadID || rep2.DataVersion == e.load.DataVersion || rep2.NewQuarantine != rep2.Quarantined {
			t.Fatalf("nạp sau khi đổi dữ liệu = %d %+v, muốn mã mới và mọi dòng cách ly ghi mới", r.status, rep2)
		}
		if rows(e.load.LoadID) != e.load.Quarantined || rows(rep2.LoadID) != rep2.Quarantined {
			t.Errorf("lịch sử cách ly: lần cũ %d, lần mới %d", rows(e.load.LoadID), rows(rep2.LoadID))
		}
		a := decodeAs[contracts.AskResponse](t, e.ask(tokDirector, qSQL))
		if got := e.getTrace(tokDirector, a.TraceID).Record.Decision.DataVersion; got != rep2.DataVersion {
			t.Errorf("vết sau nạp lại mang phiên bản dữ liệu %q, muốn %q", got, rep2.DataVersion)
		}
	})

	// Nhiều lượt song song: mỗi lượt một vết riêng, đọc lại được, không lẫn.
	t.Run("song-song", func(t *testing.T) {
		e := mk(t)
		const n = 12
		ids := make([]string, n)
		var wg sync.WaitGroup
		for i := range n {
			wg.Add(1)
			go func() {
				defer wg.Done()
				tok, q := tokDirector, qSQL
				if i%3 == 1 {
					tok, q = tokEng, qRAG
				}
				r := e.ask(tok, q)
				if r.status != 200 {
					t.Errorf("lượt %d = %d: %s", i, r.status, r.body)
					return
				}
				ids[i] = decodeAs[contracts.AskResponse](t, r).TraceID
			}()
		}
		wg.Wait()
		seen := map[string]bool{}
		for _, id := range ids {
			if id == "" || seen[id] {
				t.Fatalf("trace id rỗng hoặc trùng: %q", id)
			}
			seen[id] = true
		}
		if got := e.decisionRows(e.tenantA); got != n {
			t.Errorf("%d lượt song song để lại %d dòng vết", n, got)
		}
	})

	t.Run("duong-dan-la", func(t *testing.T) {
		e := mk(t)
		for _, id := range []string{uuid.NewString(), "x'--", "%00", strings.Repeat("a", 5000)} {
			if r := e.trace(tokDirector, id); r.status != 404 {
				t.Errorf("GET /v1/trace/%.20s = %d, muốn 404", id, r.status)
			}
		}
		if r := e.do(http.MethodGet, "/v1/ask", tokDirector, "", nil); r.status != http.StatusMethodNotAllowed {
			t.Errorf("GET /v1/ask = %d, muốn 405", r.status)
		}
		if r := e.do(http.MethodPost, "/v1/trace/x", tokDirector, "", nil); r.status != http.StatusMethodNotAllowed {
			t.Errorf("POST /v1/trace/x = %d, muốn 405", r.status)
		}
		if r := e.do(http.MethodGet, "/v1/khong-co", tokDirector, "", nil); r.status != 404 {
			t.Errorf("đường dẫn lạ = %d, muốn 404", r.status)
		}
	})

	// Chống DNS rebinding ở chế độ dev: Host không phải loopback bị từ chối, kể cả khi token đúng.
	t.Run("host-la", func(t *testing.T) {
		e := mk(t)
		req, _ := http.NewRequest(http.MethodGet, e.ts.URL+"/healthz", nil)
		req.Host = "evil.example"
		res, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		res.Body.Close()
		if res.StatusCode != 403 || e.srv.cnt.HostRejected.Load() != 1 {
			t.Errorf("Host lạ = %d, HostRejected = %d, muốn 403 và 1", res.StatusCode, e.srv.cnt.HostRejected.Load())
		}
		e.srv.hostGuard = false // tắt cổng thì Host lạ qua: chứng minh chính cổng chặn, không phải thứ khác
		e.ts.Config.Handler = e.srv.handler()
		res, err = http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		res.Body.Close()
		if res.StatusCode != 200 {
			t.Errorf("không có cổng Host, Host lạ = %d, muốn 200", res.StatusCode)
		}
	})
}

// ---------------------------------------------------------------------------------------------------------------------------------
// Chèn lỗi (chỉ memStore): thứ mà PG thật khó tái hiện theo yêu cầu.
// ---------------------------------------------------------------------------------------------------------------------------------

func memEnv(t *testing.T) (*env, *memStore) {
	t.Helper()
	ms := newMemStore()
	e := newEnv(t, ms)
	e.decisionRows = ms.decisionCount
	return e, ms
}

func TestHealthzKhoHong(t *testing.T) {
	e, ms := memEnv(t)
	ms.pingErr = contracts.Label(contracts.Infra("postgres"), errors.New("dial tcp 10.1.2.3:5432: connect: connection refused"))
	r := e.do(http.MethodGet, "/healthz", "", "", nil)
	if r.status != http.StatusServiceUnavailable || strings.Contains(string(r.body), "10.1.2.3") {
		t.Fatalf("kho hỏng: healthz = %d %s, muốn 503 không lộ chi tiết", r.status, r.body)
	}
}

// Panic trong luồng hỏi: lượt vẫn có vết (lỗi của mã), client không nhận chi tiết, server không chết.
func TestPanicVanCoVet(t *testing.T) {
	e, ms := memEnv(t)
	ms.panicEntities = true
	r := e.ask(tokDirector, qSQL)
	eb := decodeAs[errBody](t, r)
	if r.status != 500 || eb.Fault != contracts.FaultCode || eb.TraceID == "" || strings.Contains(eb.Error, "panic") {
		t.Fatalf("panic: %d %+v, muốn 500 code có trace_id, không lộ chữ panic", r.status, eb)
	}
	ms.panicEntities = false
	rec := e.getTrace(tokDirector, eb.TraceID).Record
	if rec.Fault != contracts.FaultCode || !strings.Contains(rec.Error, "panic trong luồng hỏi") || rec.Steps[0].Stage != traces.StageUnknown {
		t.Errorf("vết của lượt panic: fault %q error %q stage %q", rec.Fault, rec.Error, rec.Steps[0].Stage)
	}
	if !strings.Contains(e.serverLog(), "panic trong luồng hỏi") || !strings.Contains(e.serverLog(), "goroutine") {
		t.Error("log của server thiếu panic và stack")
	}
	if r := e.ask(tokDirector, qSQL); r.status != 200 {
		t.Errorf("sau panic server không phục vụ tiếp: %d", r.status)
	}
}

func infraErr() error {
	return contracts.Label(contracts.Infra("postgres"), errors.New("dial postgres://eais_app:s3cretpw@db:5432/x: connection refused"))
}

// Không lưu được vết (sau thử lại) thì KHÔNG giao câu trả lời, đếm lỗi, thử lại bằng CHÍNH bản ghi (cùng băm), không lộ chi tiết nội bộ
// và không để mật khẩu trong DSN lọt vào log.
func TestLuuVetThatBaiDongMacDinh(t *testing.T) {
	e, ms := memEnv(t)
	ms.saveDecisionErrs = []error{infraErr(), infraErr(), infraErr()}
	r := e.ask(tokDirector, qSQL)
	if r.status != 503 || strings.Contains(string(r.body), `"answer"`) || strings.Contains(string(r.body), "s3cretpw") || strings.Contains(string(r.body), "postgres://") {
		t.Fatalf("không lưu được vết: %d %s, muốn 503 không có câu trả lời và không lộ DSN", r.status, r.body)
	}
	if got := e.srv.cnt.TraceSaveFailed.Load(); got != 1 {
		t.Errorf("TraceSaveFailed = %d, muốn 1", got)
	}
	if len(ms.saveAttempts) != 3 {
		t.Fatalf("số lần thử lưu = %d, muốn 3", len(ms.saveAttempts))
	}
	d0, _ := ms.saveAttempts[0].Digest()
	for i, rec := range ms.saveAttempts[1:] {
		if d, _ := rec.Digest(); d != d0 {
			t.Errorf("lần thử %d dùng bản ghi KHÁC (băm khác): thử lại phải bằng chính bản ghi", i+2)
		}
	}
	if e.decisionRows(e.tenantA) != 0 {
		t.Error("lưu hỏng mà vẫn có dòng vết")
	}
	if strings.Contains(e.serverLog(), "s3cretpw") {
		t.Error("mật khẩu của DSN lọt vào log")
	}
	if r := e.ask(tokDirector, qSQL); r.status != 200 {
		t.Errorf("hết lỗi chèn mà lượt kế vẫn lỗi: %d", r.status)
	}
}

func TestLuuVetThuLaiThanhCong(t *testing.T) {
	e, ms := memEnv(t)
	ms.saveDecisionErrs = []error{infraErr()}
	r := e.ask(tokDirector, qSQL)
	if r.status != 200 || len(ms.saveAttempts) != 2 || e.decisionRows(e.tenantA) != 1 || e.srv.cnt.TraceSaveFailed.Load() != 0 {
		t.Fatalf("thử lại: status %d, %d lần thử, %d dòng vết, %d lỗi — muốn 200, 2, 1, 0", r.status, len(ms.saveAttempts), e.decisionRows(e.tenantA), e.srv.cnt.TraceSaveFailed.Load())
	}
}

// Lỗi KHÔNG tạm thời (xung đột nội dung) thì không thử lại — thử lại một xung đột chỉ che nó.
func TestLuuVetLoiKhongThuLai(t *testing.T) {
	e, ms := memEnv(t)
	ms.saveDecisionErrs = []error{contracts.Errorf(contracts.FaultData, "trace_id đã có bản ghi với nội dung khác; không ghi đè")}
	r := e.ask(tokDirector, qSQL)
	if r.status != http.StatusConflict || len(ms.saveAttempts) != 1 {
		t.Fatalf("xung đột nội dung: %d, %d lần thử — muốn 409 và đúng 1 lần", r.status, len(ms.saveAttempts))
	}
}

// Client ngắt kết nối giữa chừng không được làm mất vết: persist dùng ngữ cảnh không huỷ theo client.
func TestVetKhongMatKhiClientNgat(t *testing.T) {
	e, _ := memEnv(t)
	pr := e.principal(tokDirector)
	rec := traces.NewErrorRecord(uuid.NewString(), pr, "q", e.app.now(), e.app.versions(), contracts.Errorf(contracts.FaultInput, "x"))
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := e.srv.persist(ctx, pr, rec, nil); err != nil {
		t.Fatalf("persist với ngữ cảnh đã huỷ: %v", err)
	}
	if got := e.decisionRows(e.tenantA); got != 1 {
		t.Errorf("client ngắt mà có %d dòng vết, muốn 1", got)
	}
}

// Lượt chưa xác thực không được chạm kho (không tenant ⇒ không có gì để ghi, và không tốn kết nối).
func TestChanXacThucKhongChamKho(t *testing.T) {
	e, ms := memEnv(t)
	before := ms.touched
	for _, path := range []string{"/v1/ask", "/v1/ingest"} {
		e.do(http.MethodPost, path, "", "application/json", []byte(`{}`))
	}
	e.do(http.MethodGet, "/v1/trace/x", "token-la", "", nil)
	if ms.touched != before {
		t.Errorf("yêu cầu chưa xác thực chạm kho %d lần", ms.touched-before)
	}
}

// Một lượt lỗi ở Handle (không phải thân sai) cũng có vết: tenant của principal không có runtime ở đây (đã kiểm ở suite); ở đây kiểm thêm
// rằng lỗi hạ tầng giữa lượt (kho trả lỗi khi đọc danh mục) cho 503, vết mang nhãn hạ tầng và thông điệp đã làm sạch.
func TestLoiHaTangGiuaLuotCoVet(t *testing.T) {
	e, _ := memEnv(t)
	broken := &failingDirectory{store: e.st}
	e.app.st = broken
	e.app.setAsk(e.app.dataVersionNow())
	r := e.ask(tokDirector, qSQL)
	eb := decodeAs[errBody](t, r)
	if r.status != http.StatusServiceUnavailable || eb.TraceID == "" || strings.Contains(string(r.body), "s3cretpw") {
		t.Fatalf("lỗi hạ tầng giữa lượt: %d %s", r.status, r.body)
	}
	rec := e.getTrace(tokDirector, eb.TraceID).Record
	if rec.Fault != contracts.Infra("postgres") || strings.Contains(rec.Error, "s3cretpw") || !strings.Contains(rec.Error, traces.Masked) {
		t.Errorf("vết lỗi hạ tầng: fault %q error %q", rec.Fault, rec.Error)
	}
}

// failingDirectory: kho mà danh mục thực thể luôn lỗi hạ tầng (kèm DSN có mật khẩu — thứ không được lọt ra ngoài).
type failingDirectory struct{ store }

func (f *failingDirectory) Entities(context.Context, ...string) ([]entityresolution.Entity, error) {
	return nil, infraErr()
}
