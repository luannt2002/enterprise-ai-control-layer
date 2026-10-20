package main

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/luannt2002/enterprise-ai-systems/core/contracts"
	"github.com/luannt2002/enterprise-ai-systems/core/tenant"
	"github.com/luannt2002/enterprise-ai-systems/data/entityresolution"
	"github.com/luannt2002/enterprise-ai-systems/data/ingestion"
	"github.com/luannt2002/enterprise-ai-systems/evals"
	"github.com/luannt2002/enterprise-ai-systems/intelligence/models"
	"github.com/luannt2002/enterprise-ai-systems/intelligence/rag"
	"github.com/luannt2002/enterprise-ai-systems/tenants/construction/win/gen"
)

// genData sinh bộ dữ liệu functional (seed 42) vào thư mục tạm.
func genData(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	if err := gen.Generate("../world.yaml", dir, "functional", 42); err != nil {
		t.Fatal(err)
	}
	return dir
}

func baseCfg(data string) config {
	return config{data: data, pack: "../../pack", world: "../world.yaml", cases: "../cases-independent.yaml",
		tenant: "abc-infra", grade: gradeContent, k: 1, topK: 10, systems: sysB2Plus}
}

// clearModelEnv đảm bảo test chạy bằng model giả bất kể môi trường của người chạy.
func clearModelEnv(t *testing.T) {
	for _, k := range []string{"EAIS_MODEL_PROVIDER", "EAIS_MODEL", "EAIS_MODEL_API_KEY", "EAIS_MODEL_BASE_URL", "EAIS_MODEL_THINKING", "EAIS_MODEL_EFFORT", "EAIS_MODEL_TEMPERATURE", "EAIS_MODEL_MAX_RETRIES", "EAIS_MODEL_MAX_TOKENS"} {
		t.Setenv(k, "")
	}
}

type loaded struct {
	rt   *tenant.Runtime
	dir  string
	res  ingestion.Result
	byID map[string]entityresolution.Entity
	docs []rag.Document
}

func load(t *testing.T, data string) loaded {
	t.Helper()
	rt, err := tenant.Load("../../pack")
	if err != nil {
		t.Fatal(err)
	}
	sm, err := ingestion.LoadSources("../../pack/sources.yaml", rt)
	if err != nil {
		t.Fatal(err)
	}
	dir := filepath.Join(data, "abc-infra")
	res, err := ingestion.Ingest(context.Background(), dir, rt, sm, ingestion.PdfToText{})
	if err != nil {
		t.Fatal(err)
	}
	directory, err := ingestion.LoadDirectory(dir, sm)
	if err != nil {
		t.Fatal(err)
	}
	byID := map[string]entityresolution.Entity{}
	for _, e := range directory {
		byID[e.ID] = e
	}
	docs, err := rag.LoadMarkdown(filepath.Join(dir, "docs"), "abc-infra")
	if err != nil {
		t.Fatal(err)
	}
	return loaded{rt: rt, dir: dir, res: res, byID: byID, docs: docs}
}

// B2+ viết thực thể bằng đủ dữ liệu gốc (dự án, bí danh, mã theo nguồn); B2 cũ thì không — đó là chỗ
// bản soát đo được trần 4/13 và 5/13 của B2 (REVIEW-D5 Q3).
func TestParagraphPlusHasMasterData(t *testing.T) {
	l := load(t, genData(t))
	names := map[string]string{}
	for id, e := range l.byID {
		names[id] = e.Name
	}
	var o *ingestion.Observation
	for i := range l.res.Observations {
		if l.res.Observations[i].Key == "warehouse.csv#R-004" {
			o = &l.res.Observations[i]
		}
	}
	if o == nil {
		t.Fatal("không thấy quan sát warehouse.csv#R-004")
	}
	plus, plain := paragraphPlus(l.rt, l.byID, *o), paragraph(l.rt, names, *o)
	for _, want := range []string{"Máy xúc EX-03", "dự án Cầu A (PRJ-A; bí danh: dự án A, cầu A, công trình A)", "bí danh: máy 03, máy xúc 03, MX-03",
		"mã theo nguồn: erp asset-8392, fuel_sensor EX003, site_report_form Machine 03, warehouse_slip EX-03", "3850"} {
		if !strings.Contains(plus, want) {
			t.Errorf("đoạn B2+ thiếu %q: %s", want, plus)
		}
	}
	if strings.Contains(plus, "\n") {
		t.Errorf("đoạn B2+ phải một dòng: %q", plus)
	}
	if strings.Contains(plain, "asset-8392") || strings.Contains(plain, "MX-03") {
		t.Errorf("đoạn B2 cũ không có bí danh / mã nguồn (giữ để báo cáo): %s", plain)
	}
}

// spyGW ghi lại mọi yêu cầu gửi model.
type spyGW struct {
	mu   sync.Mutex
	reqs []models.Request
}

func (g *spyGW) Generate(_ context.Context, r models.Request) (contracts.Draft, error) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.reqs = append(g.reqs, r)
	return contracts.Draft{Text: "trả lời giả"}, nil
}

func (g *spyGW) lastIDs() map[string]bool {
	g.mu.Lock()
	defer g.mu.Unlock()
	out := map[string]bool{}
	for _, e := range g.reqs[len(g.reqs)-1].Evidence {
		out[e.ID] = true
	}
	return out
}

// B1 đọc bản ghi THÔ (kể cả bản ghi bẩn, kể cả PDF), nhận entities.json + tài liệu, và chỉ nhận bản ghi
// có ngày trong cửa sổ của ca. Ca không có cửa sổ trong hồ sơ là lỗi, không đoán.
func TestDirectLLMRawRowsAndWindow(t *testing.T) {
	l := load(t, genData(t))
	rows, err := readRawRows(l.dir, l.rt.Location, ingestion.PdfToText{})
	if err != nil {
		t.Fatal(err)
	}
	idOf := map[string]rawRow{}
	for _, r := range rows {
		for _, rec := range []string{"R-004", "R-005", "R-018", "R-021", "R-001"} {
			if strings.Contains(r.Text, "record_id="+rec+",") || strings.HasSuffix(r.Text, "record_id="+rec) {
				idOf[rec] = r
			}
		}
		if strings.HasPrefix(r.ID, "reports/2026-10-19") {
			idOf["pdf"] = r
		}
	}
	for _, rec := range []string{"R-004", "R-005", "R-018", "R-021", "R-001", "pdf"} {
		if _, ok := idOf[rec]; !ok {
			t.Fatalf("bản ghi thô thiếu %s", rec)
		}
	}
	if !strings.Contains(idOf["R-021"].Text, "quantity=,") { // CONTRACT.md §2: thiếu lượng = ô trống
		t.Errorf("bản ghi bẩn R-021 phải có mặt, giữ nguyên ô lượng trống: %s", idOf["R-021"].Text)
	}
	if !strings.Contains(idOf["pdf"].Text, "Machine 03") {
		t.Errorf("báo cáo PDF phải có chữ thô: %s", idOf["pdf"].Text)
	}
	fixed, err := fixedEvidence(l.dir, l.docs)
	if err != nil {
		t.Fatal(err)
	}
	rows = append(rows, rawRow{ID: "khong-ngay.csv#1", Text: "khong-ngay.csv | ngày=??, quantity=7"}) // bản ghi không đọc được ngày
	day := func(s string) time.Time { d, _ := time.ParseInLocation("2006-01-02", s, l.rt.Location); return d }
	cases := []evals.WinCase{{ID: "C1", Principal: "u-director", Question: "q1"}, {ID: "C2", Principal: "u-director", Question: "q2"}}
	spy := &spyGW{}
	d, err := newDirectLLM(spy, rows, fixed, cases, map[string]*window{"C1": {from: day("2026-10-19"), to: day("2026-10-19")}, "C2": nil}, "Hôm nay là ngày 20/10/2026 (giờ Việt Nam).")
	if err != nil {
		t.Fatal(err)
	}
	pr := contracts.Principal{ID: "u-director", TenantID: "abc-infra"}
	if _, err := d.Ask(context.Background(), pr, "q1"); err != nil {
		t.Fatal(err)
	}
	got := spy.lastIDs()
	for _, want := range []string{idOf["R-004"].ID, idOf["R-005"].ID, idOf["pdf"].ID, "entities.json", "docs/SOP-FUEL-01"} {
		if !got[want] {
			t.Errorf("cửa sổ 19/10: thiếu %s", want)
		}
	}
	if got[idOf["R-018"].ID] {
		t.Errorf("cửa sổ 19/10: không được có bản ghi ngày 15/09 (%s)", idOf["R-018"].ID)
	}
	if q := spy.reqs[0].Question; !strings.HasPrefix(q, "Hôm nay là ngày 20/10/2026") || !strings.HasSuffix(q, "q1") {
		t.Errorf("câu gửi model = %q, muốn ngày hiện tại + câu hỏi", q)
	}
	if _, err := d.Ask(context.Background(), pr, "q2"); err != nil {
		t.Fatal(err)
	}
	ids := spy.lastIDs()
	for id := range ids {
		if strings.HasPrefix(id, "warehouse.csv#") || strings.HasPrefix(id, "reports/") {
			t.Errorf("ca không có cửa sổ: không được có bản ghi có ngày, có %s", id)
		}
	}
	if !ids["khong-ngay.csv#1"] || !ids["entities.json"] {
		t.Error("bản ghi không đọc được ngày và entities.json phải luôn có, kể cả ca không có cửa sổ")
	}
	if _, err := d.Ask(context.Background(), pr, "q3"); err == nil {
		t.Error("câu hỏi không có trong bảng cửa sổ: muốn lỗi")
	}
	if _, err := newDirectLLM(spy, rows, fixed, cases, map[string]*window{"C1": nil}, ""); err == nil {
		t.Error("ca C2 thiếu b1_window: muốn lỗi")
	}
	if d.ModelCalls() != 2 {
		t.Errorf("ModelCalls = %d, muốn 2 (mỗi câu hỏi hợp lệ một lời gọi)", d.ModelCalls())
	}
}

// Hồ sơ cửa sổ đọc được cả dạng mảng lẫn object có "cases"; cửa sổ sai hay mục thiếu id là lỗi.
func TestLoadWindows(t *testing.T) {
	loc, _ := time.LoadLocation("Asia/Ho_Chi_Minh")
	dir := t.TempDir()
	write := func(name, body string) string {
		p := filepath.Join(dir, name)
		if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
		return p
	}
	ok := `[{"case_id":"A","b1_window":{"from":"2026-10-18","to":"2026-10-20"}},{"case_id":"B","b1_window":null}]`
	for _, p := range []string{write("mang.json", ok), write("object.json", `{"version":1,"cases":`+ok+`}`)} {
		w, err := loadWindows(p, loc)
		if err != nil || w["A"] == nil || w["A"].from.Day() != 18 || w["A"].to.Day() != 20 || w["B"] != nil {
			t.Errorf("%s: được %v (lỗi %v)", filepath.Base(p), w, err)
		}
		if _, has := w["B"]; !has {
			t.Errorf("%s: ca có b1_window null vẫn phải có mặt (nil)", filepath.Base(p))
		}
	}
	for name, body := range map[string]string{
		"nguoc.json":    `[{"case_id":"A","b1_window":{"from":"2026-10-20","to":"2026-10-18"}}]`,
		"thieu-id.json": `[{"b1_window":null}]`,
		"hong.json":     `không phải json`,
	} {
		if _, err := loadWindows(write(name, body), loc); err == nil {
			t.Errorf("%s: muốn lỗi", name)
		}
	}
}

// -out ghi đủ bảy tệp bằng chứng; model giả ⇒ kết luận CHẶN (mã 2), B1 thiếu hồ sơ ⇒ ghi là hệ cổng
// không chạy được, không âm thầm bỏ.
func TestRunWritesEvidence(t *testing.T) {
	clearModelEnv(t)
	cfg := baseCfg(genData(t))
	cfg.systems, cfg.out = sysB2Plus+","+sysB1, t.TempDir()
	if code := run(cfg); code != 2 {
		t.Fatalf("model giả: mã thoát = %d, muốn 2 (CHẶN)", code)
	}
	for _, f := range []string{"manifest.json", "raw-runs.jsonl", "scored.jsonl", "aggregate.json", "failures.jsonl", "cost.json", "ket-qua.md"} {
		if st, err := os.Stat(filepath.Join(cfg.out, f)); err != nil || st.Size() == 0 {
			t.Errorf("thiếu hoặc rỗng: %s (%v)", f, err)
		}
	}
	var m manifestInfo
	b, _ := os.ReadFile(filepath.Join(cfg.out, "manifest.json"))
	if err := json.Unmarshal(b, &m); err != nil {
		t.Fatal(err)
	}
	if !m.FakeModel || len(m.Blocked) != 1 || !strings.Contains(m.Blocked[0], "B1") || m.GitRevision == "" || m.Files["../cases-independent.yaml"] == "" {
		t.Errorf("manifest sai: fake %v, blocked %v, git %q, files %v", m.FakeModel, m.Blocked, m.GitRevision, m.Files)
	}
	raw, _ := os.ReadFile(filepath.Join(cfg.out, "raw-runs.jsonl"))
	if n := strings.Count(string(raw), "\n"); n != 2*m.Cases {
		t.Errorf("raw-runs có %d dòng, muốn %d (EACL + B2+, mỗi ca một dòng)", n, 2*m.Cases)
	}
}

// Thử khói qua máy chủ HTTP giả với client model THẬT: yêu cầu không có temperature, có thinking +
// effort đã cấu hình, đúng model; bằng chứng thô ghi đủ lời gọi + token, tính được tiền, và không ở
// đâu chứa khoá.
func TestSmokeThroughFakeProvider(t *testing.T) {
	var mu sync.Mutex
	var bodies []map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		var m map[string]any
		_ = json.Unmarshal(b, &m)
		mu.Lock()
		bodies = append(bodies, m)
		mu.Unlock()
		_, _ = io.WriteString(w, `{"content":[{"type":"text","text":"{\"answer\": \"trả lời giả\", \"evidence_ids\": []}"}],"usage":{"input_tokens":100,"output_tokens":20}}`)
	}))
	defer srv.Close()
	clearModelEnv(t)
	const key = "khoa-test-khong-duoc-lo"
	for k, v := range map[string]string{"EAIS_MODEL_PROVIDER": "anthropic", "EAIS_MODEL": "claude-sonnet-5-5", "EAIS_MODEL_API_KEY": key,
		"EAIS_MODEL_BASE_URL": srv.URL, "EAIS_MODEL_THINKING": "between_tools", "EAIS_MODEL_EFFORT": "medium"} {
		t.Setenv(k, v)
	}
	cfg := baseCfg(genData(t))
	cfg.smoke, cfg.out = 3, t.TempDir()
	if code := run(cfg); code != 0 {
		t.Fatalf("thử khói qua provider giả trả 200: mã thoát = %d, muốn 0", code)
	}
	if len(bodies) < 3 {
		t.Fatalf("muốn ≥ 3 lời gọi (B2+ gọi mỗi ca một lần), được %d", len(bodies))
	}
	for i, b := range bodies {
		th, _ := b["thinking"].(map[string]any)
		oc, _ := b["output_config"].(map[string]any)
		if _, has := b["temperature"]; has || th["type"] != "between_tools" || oc["effort"] != "medium" || b["model"] != "claude-sonnet-5-5" {
			t.Errorf("yêu cầu %d sai tham số: temperature có mặt = %v, thinking %v, output_config %v, model %v", i, has, b["thinking"], b["output_config"], b["model"])
		}
	}
	err := filepath.Walk(cfg.out, func(p string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return err
		}
		b, _ := os.ReadFile(p)
		if strings.Contains(string(b), key) {
			t.Errorf("%s chứa khoá", filepath.Base(p))
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	var cost []costRow
	b, _ := os.ReadFile(filepath.Join(cfg.out, "cost.json"))
	if err := json.Unmarshal(b, &cost); err != nil {
		t.Fatal(err)
	}
	for _, c := range cost {
		if c.System == sysB2Plus && (!c.Priced || c.HTTPCalls != 3 || c.InputTokens != 300 || c.OutputTokens != 60 || c.USD <= 0) {
			t.Errorf("chi phí B2+ = %+v, muốn 3 lời gọi, 300 / 60 token, có giá", c)
		}
	}
	raw, _ := os.ReadFile(filepath.Join(cfg.out, "raw-runs.jsonl"))
	if !strings.Contains(string(raw), `"status":200`) || !strings.Contains(string(raw), `"between_tools"`) {
		t.Error("raw-runs phải có vết lời gọi (mã 200, thân yêu cầu)")
	}
	msgs, _ := bodies[0]["messages"].([]any)
	first, _ := msgs[0].(map[string]any)
	if c, _ := first["content"].(string); !strings.HasPrefix(c, "Câu hỏi: Hôm nay là ngày 20/10/2026 (giờ Việt Nam).") {
		t.Errorf("B2+ phải đưa ngày hiện tại trước câu hỏi, câu gửi model bắt đầu bằng %.80q", c)
	}
}

// Provider từ chối tham số (400) ⇒ thử khói KHÔNG ĐẠT, mã 1: đúng lúc phải sửa cấu hình trước khi chi tiền.
func TestSmokeFailsWhenProviderRejects(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = io.WriteString(w, `{"type":"error","error":{"type":"invalid_request_error","message":"thinking.type: between_tools not supported"}}`)
	}))
	defer srv.Close()
	clearModelEnv(t)
	for k, v := range map[string]string{"EAIS_MODEL_PROVIDER": "anthropic", "EAIS_MODEL": "m", "EAIS_MODEL_API_KEY": "k",
		"EAIS_MODEL_BASE_URL": srv.URL, "EAIS_MODEL_THINKING": "between_tools"} {
		t.Setenv(k, v)
	}
	cfg := baseCfg(genData(t))
	cfg.smoke = 2
	if code := run(cfg); code != 1 {
		t.Fatalf("provider trả 400: mã thoát = %d, muốn 1 (THỬ KHÓI: KHÔNG ĐẠT)", code)
	}
}

// Luật kết luận: ĐẠT chỉ khi mọi baseline cổng chạy được và đạt; hệ chỉ-báo-cáo trượt không ảnh hưởng.
func TestGatePass(t *testing.T) {
	ok, bad := evals.G7Verdict{Pass: true}, evals.G7Verdict{Pass: false}
	all := []string{sysEACL, sysB2Plus, sysB1, sysB2}
	for _, c := range []struct {
		name              string
		order             []string
		blocked, unstable []string
		verdicts          map[string]evals.G7Verdict
		want              bool
	}{
		{"đủ cổng, đều đạt; B2 trượt chỉ báo cáo", all, nil, nil, map[string]evals.G7Verdict{sysB2Plus: ok, sysB1: ok, sysB2: bad}, true},
		{"B1 trượt", all, nil, nil, map[string]evals.G7Verdict{sysB2Plus: ok, sysB1: bad}, false},
		{"B1 không chạy", []string{sysEACL, sysB2Plus}, []string{"B1: thiếu -manifest"}, nil, map[string]evals.G7Verdict{sysB2Plus: ok}, false},
		{"B1 vắng mà không ghi bị chặn", []string{sysEACL, sysB2Plus}, nil, nil, map[string]evals.G7Verdict{sysB2Plus: ok}, false},
		{"lượt vô hiệu", all, nil, []string{"WIN-I-201"}, map[string]evals.G7Verdict{sysB2Plus: ok, sysB1: ok}, false},
		{"có hệ bị báo chặn dù các cổng có mặt", all, []string{"B1: lỗi dựng"}, nil, map[string]evals.G7Verdict{sysB2Plus: ok, sysB1: ok}, false},
	} {
		if got := gatePass(c.order, c.blocked, c.unstable, c.verdicts); got != c.want {
			t.Errorf("%s: gatePass = %v, muốn %v", c.name, got, c.want)
		}
	}
}

// B2″: luật thẩm quyền + dung sai sinh MÁY MÓC từ metrics.yaml (tái lập được), cùng hạng gộp một mức.
func TestDomainRules(t *testing.T) {
	r, err := domainRules("../../pack")
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"Dầu kho xuất (đơn vị L)", "phiếu xuất kho (hạng 1)", "ERP, bảng tổng hợp (hạng 2)", "Dung sai 10 L", "chỉ so các nguồn của CÙNG một chỉ số"} {
		if !strings.Contains(r, want) {
			t.Errorf("luật miền thiếu %q:\n%s", want, r)
		}
	}
}

// Đường tất định của EACL khác giữa hai lượt ⇒ ca đó bị liệt kê (lượt vô hiệu, ADR-0005 §5); baseline
// khác nhau giữa các lượt thì không tính (model ngẫu nhiên là chuyện bình thường của baseline).
func TestUnstablePathsAndCosts(t *testing.T) {
	recs := []askRecord{
		{System: sysEACL, Run: 1, CaseID: "A", Route: "SQL_QUERY", Outcome: "RESOLVED"},
		{System: sysEACL, Run: 2, CaseID: "A", Route: "SQL_QUERY", Outcome: "RESOLVED"},
		{System: sysEACL, Run: 1, CaseID: "B", Route: "RAG", Outcome: "RESOLVED", Citations: []string{"SOP#0"}},
		{System: sysEACL, Run: 2, CaseID: "B", Route: "RAG", Outcome: "RESOLVED", Citations: []string{"SOP#1"}},
		{System: sysB2Plus, Run: 1, CaseID: "A", Route: "RAG", Calls: []modelCall{{InputTokens: 1_000_000, OutputTokens: 100_000}}},
		{System: sysB2Plus, Run: 2, CaseID: "A", Route: "CLARIFICATION"},
	}
	if got := unstablePaths(recs); len(got) != 1 || got[0] != "B" {
		t.Errorf("unstablePaths = %v, muốn [B]", got)
	}
	c := costs(recs, "claude-sonnet-5-5", []string{sysEACL, sysB2Plus})
	if c[1].USD != 3 || !c[1].Priced || c[0].USD != 0 {
		t.Errorf("chi phí = %+v, muốn B2+ = 1 MTok × 2 + 0,1 MTok × 10 = 3 USD", c)
	}
	if c := costs(recs, "model-la", []string{sysB2Plus}); c[0].Priced || c[0].USD != 0 {
		t.Errorf("model không có giá: %+v, muốn priced = false", c[0])
	}
}

// Provider trả 200 nhưng thân không đọc được ⇒ câu hỏi lỗi ⇒ thử khói KHÔNG ĐẠT (mã 1), dù mã HTTP là 200.
func TestSmokeFailsOnBadResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.WriteString(w, `không phải JSON`)
	}))
	defer srv.Close()
	clearModelEnv(t)
	for k, v := range map[string]string{"EAIS_MODEL_PROVIDER": "anthropic", "EAIS_MODEL": "m", "EAIS_MODEL_API_KEY": "k", "EAIS_MODEL_BASE_URL": srv.URL} {
		t.Setenv(k, v)
	}
	cfg := baseCfg(genData(t))
	cfg.smoke = 2
	if code := run(cfg); code != 1 {
		t.Fatalf("thân phản hồi hỏng: mã thoát = %d, muốn 1", code)
	}
}
