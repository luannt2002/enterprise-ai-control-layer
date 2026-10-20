package main

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sort"
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
		for _, rec := range []string{"R-004", "R-005", "R-018", "R-021", "R-001", "R-008"} {
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
	if got[idOf["R-008"].ID] { // cận TRÊN (bản soát mã: M34)
		t.Errorf("cửa sổ 19/10: không được có bản ghi ngày 20/10 (%s)", idOf["R-008"].ID)
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

// Số riêng từng ca cho bộ đếm claim không căn cứ: truth.value, other_values và value_l của bản ghi bộ
// sinh mà ca trỏ tới (record_ids theo key hoặc world_id; cùng thực thể trong kỳ). Bản ghi ngoài kỳ, khác
// thực thể, của tenant khác, hay bị cách ly (không value_l) thì không; ca không có truth thì không có số.
func TestPerCaseNumbers(t *testing.T) {
	dir := t.TempDir()
	write := func(name, body string) string {
		p := filepath.Join(dir, name)
		if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
		return p
	}
	cases := write("ca.json", `{"cases":[
 {"case_id":"A","truth":{"entities":["FX-12"],"period":{"from":"2026-10-18","to":"2026-10-19"},"value":"5.345","other_values":[{"source":"giám sát","value":"5.300"}],"record_ids":["BG-9"]}},
 {"case_id":"B","truth":{"entities":[],"period":null,"value":"","other_values":[],"record_ids":["R-001"]}},
 {"case_id":"C","b1_window":null}]}`)
	genMan := write("manifest.json", `{"version":1,"records":[
 {"tenant":"abc-infra","key":"BG-1","entity":"FX-12","on":"2026-10-18","value_l":"2645"},
 {"tenant":"abc-infra","key":"BG-2","entity":"FX-12","on":"2026-10-19","value_l":"2700"},
 {"tenant":"abc-infra","key":"BG-3","entity":"FX-12","on":"2026-10-20","value_l":"1111"},
 {"tenant":"abc-infra","key":"BG-4","entity":"FX-13","on":"2026-10-18","value_l":"2222"},
 {"tenant":"def-build","key":"BG-5","entity":"FX-12","on":"2026-10-18","value_l":"3333"},
 {"tenant":"abc-infra","key":"BG-9","entity":"FX-99","on":"2026-09-01","value_l":"4444"},
 {"tenant":"abc-infra","key":"W-1","world_id":"R-001","entity":"EX-03","on":"2026-10-19","value_l":"3850"},
 {"tenant":"abc-infra","key":"Q-1","world_id":"R-001","expect":"quarantine","reason":"ngày hỏng"}]}`)
	got, err := perCaseNumbers(cases, genMan, "abc-infra")
	if err != nil {
		t.Fatal(err)
	}
	sorted := func(xs []string) string {
		ys := append([]string(nil), xs...)
		sort.Strings(ys)
		return strings.Join(ys, ",")
	}
	if g := sorted(got["A"]); g != "2645,2700,4444,5.300,5.345" {
		t.Errorf("ca A: %s; muốn 2645,2700,4444,5.300,5.345", g)
	}
	if g := sorted(got["B"]); g != "3850" {
		t.Errorf("ca B (record_ids theo world_id): %s; muốn 3850", g)
	}
	if len(got["C"]) != 0 {
		t.Errorf("ca C không có truth: %v; muốn rỗng", got["C"])
	}
	for name, body := range map[string]string{
		"ky-hong.json": `[{"case_id":"A","truth":{"entities":["FX-12"],"period":{"from":"18/10/2026","to":"2026-10-19"},"value":"","other_values":[],"record_ids":[]}}]`,
		"trung.json":   `[{"case_id":"A"},{"case_id":"A"}]`,
		"hong.json":    `không phải json`,
	} {
		if _, err := perCaseNumbers(write(name, body), genMan, "abc-infra"); err == nil {
			t.Errorf("%s: muốn lỗi", name)
		}
	}
	if _, err := perCaseNumbers(cases, write("rong.json", `{"version":1,"records":[]}`), "abc-infra"); err == nil {
		t.Error("manifest bộ sinh không có bản ghi: muốn lỗi")
	}
}

// build nối PerCase vào bộ đếm: số chỉ có trong truth của MỘT ca được phép ở ca đó, không ở ca khác.
func TestBuildWiresPerCase(t *testing.T) {
	clearModelEnv(t)
	cfg := baseCfg(genData(t))
	suite, err := evals.LoadWinSuite(cfg.cases)
	if err != nil {
		t.Fatal(err)
	}
	a, b := suite.Cases[0], suite.Cases[1]
	var list []map[string]any
	for _, c := range suite.Cases {
		e := map[string]any{"case_id": c.ID, "b1_window": nil}
		if c.ID == a.ID {
			e["truth"] = map[string]any{"entities": []string{}, "period": nil, "value": "98.765", "other_values": []any{}, "record_ids": []string{}}
		}
		list = append(list, e)
	}
	body, _ := json.Marshal(list)
	cfg.manifest = filepath.Join(t.TempDir(), "ho-so.json")
	if err := os.WriteFile(cfg.manifest, body, 0o644); err != nil {
		t.Fatal(err)
	}
	gw, err := models.New(models.ConfigFromEnv())
	if err != nil {
		t.Fatal(err)
	}
	systems, _ := parseSystems(sysB2Plus)
	st, err := build(cfg, gw, systems)
	if err != nil {
		t.Fatal(err)
	}
	got := st.counter.Count([]evals.WinCase{a, b}, []evals.Report{{Results: []evals.CaseResult{
		{ID: a.ID, FailureClass: a.FailureClass, Answer: "98765 L"},
		{ID: b.ID, FailureClass: b.FailureClass, Answer: "98765 L"},
	}}})
	if got.Unsupported != 1 || len(got.Examples) != 1 || !strings.HasPrefix(got.Examples[0], b.ID+" ") {
		t.Errorf("số trong truth của %s: Unsupported = %d, ví dụ %v; muốn 1, chỉ ở %s", a.ID, got.Unsupported, got.Examples, b.ID)
	}
}

// Canary TÊN (D-G7-9 mục 12): tên / mã / bí danh của dự án, thiết bị và chính mã tenant khác là canary
// khi tenant được chấm không có chúng ("Cầu Z", "PRJ-Z", "def-build"); tên dùng chung ("EX-03", "máy 03"
// — tenant B cố ý dùng cùng khoá) thì không, vì nhắc chúng không phải rò rỉ.
func TestNameCanaries(t *testing.T) {
	w, err := loadWorld("../world.yaml")
	if err != nil {
		t.Fatal(err)
	}
	rec, docs := w.texts("abc-infra")
	got := map[string]bool{}
	for _, c := range nameCanaries(w, "abc-infra", append(rec, docs...)) {
		got[strings.ToLower(c)] = true
	}
	for _, want := range []string{"def-build", "prj-z", "cầu z", "dự án z"} {
		if !got[want] {
			t.Errorf("thiếu canary tên %q (có %v)", want, got)
		}
	}
	for _, shared := range []string{"ex-03", "máy 03", "máy xúc 03"} {
		if got[shared] {
			t.Errorf("%q có ở tenant được chấm: không được là canary", shared)
		}
	}
	var short world
	short.Tenants = append(short.Tenants, w.Tenants...)
	short.Tenants[1].Projects = append(short.Tenants[1].Projects, short.Tenants[1].Projects[0])
	short.Tenants[1].Projects[len(short.Tenants[1].Projects)-1].Aliases = []string{"Zx"}
	for _, c := range nameCanaries(short, "abc-infra", append(rec, docs...)) {
		if strings.EqualFold(c, "Zx") {
			t.Error("tên ngắn hơn 4 ký tự (\"Zx\") khớp nhầm quá dễ: không được là canary")
		}
	}

	st := buildFake(t, baseCfg(genData(t)))
	if l := st.counter.Leaks("Dự án nào tốn dầu nhất?", "Dự án Cầu Z tốn nhiều nhất."); len(l) == 0 {
		t.Error("câu trả lời nêu \"Cầu Z\" của tenant khác: bộ đếm phải thấy rò rỉ")
	}
}

// buildFake dựng setup với model giả cho các test chỉ cần bộ đếm / danh sách hệ.
func buildFake(t *testing.T, cfg config) *setup {
	t.Helper()
	clearModelEnv(t)
	gw, err := models.New(models.ConfigFromEnv())
	if err != nil {
		t.Fatal(err)
	}
	systems, err := parseSystems(cfg.systems)
	if err != nil {
		t.Fatal(err)
	}
	st, err := build(cfg, gw, systems)
	if err != nil {
		t.Fatal(err)
	}
	return st
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
		name                       string
		order                      []string
		blocked, unstable, invalid []string
		verdicts                   map[string]evals.G7Verdict
		want                       bool
	}{
		{"đủ cổng, đều đạt; B2 trượt chỉ báo cáo", all, nil, nil, nil, map[string]evals.G7Verdict{sysB2Plus: ok, sysB1: ok, sysB2: bad}, true},
		{"B1 trượt", all, nil, nil, nil, map[string]evals.G7Verdict{sysB2Plus: ok, sysB1: bad}, false},
		{"B1 không chạy", []string{sysEACL, sysB2Plus}, []string{"B1: thiếu -manifest"}, nil, nil, map[string]evals.G7Verdict{sysB2Plus: ok}, false},
		{"B1 vắng mà không ghi bị chặn", []string{sysEACL, sysB2Plus}, nil, nil, nil, map[string]evals.G7Verdict{sysB2Plus: ok}, false},
		{"lượt vô hiệu (đường tất định)", all, nil, []string{"WIN-I-201"}, nil, map[string]evals.G7Verdict{sysB2Plus: ok, sysB1: ok}, false},
		{"lượt vô hiệu (lỗi gọi model)", all, nil, nil, []string{"B1 lượt 2: 9/117 câu lỗi gọi model"}, map[string]evals.G7Verdict{sysB2Plus: ok, sysB1: ok}, false},
		{"có hệ bị báo chặn dù các cổng có mặt", all, []string{"B1: lỗi dựng"}, nil, nil, map[string]evals.G7Verdict{sysB2Plus: ok, sysB1: ok}, false},
	} {
		if got := gatePass(c.order, c.blocked, c.unstable, c.invalid, c.verdicts); got != c.want {
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
	// Câu CÓ gọi model: trích dẫn / outcome do model quyết, chỉ so route (bản soát mã S2-5); câu lỗi gọi
	// model không tính là bất ổn đường tất định.
	more := append(append([]askRecord(nil), recs...),
		askRecord{System: sysEACL, Run: 1, CaseID: "C", Route: "RAG", Outcome: "RESOLVED", ModelCalls: 1, Citations: []string{"SOP#0"}},
		askRecord{System: sysEACL, Run: 2, CaseID: "C", Route: "RAG", Outcome: "SYSTEM_FAILURE", ModelCalls: 1, Citations: []string{"SOP#3"}},
		askRecord{System: sysEACL, Run: 1, CaseID: "D", Route: "RAG", ModelCalls: 1},
		askRecord{System: sysEACL, Run: 2, CaseID: "D", Route: "SQL_QUERY", ModelCalls: 1},
		askRecord{System: sysEACL, Run: 1, CaseID: "E", Route: "RAG", ModelCalls: 1},
		askRecord{System: sysEACL, Run: 2, CaseID: "E", Route: "", ModelCalls: 1, ModelError: true},
		askRecord{System: sysEACL, Run: 1, CaseID: "F", Route: "SQL_QUERY", Outcome: "RESOLVED"},
		askRecord{System: sysEACL, Run: 2, CaseID: "F", Route: "SQL_QUERY", Outcome: "INSUFFICIENT_EVIDENCE"},
		askRecord{System: sysEACL, Run: 1, CaseID: "G", Route: "CLARIFICATION", Candidates: []string{"EX-07", "RL-07"}},
		askRecord{System: sysEACL, Run: 2, CaseID: "G", Route: "CLARIFICATION", Candidates: []string{"EX-07"}},
		askRecord{System: sysEACL, Run: 1, CaseID: "H", Route: "SQL_QUERY", Conflicts: []string{"issued_fuel|EX-03|2026-10-19"}},
		askRecord{System: sysEACL, Run: 2, CaseID: "H", Route: "SQL_QUERY"},
	)
	if got := unstablePaths(more); strings.Join(got, ",") != "B,D,F,G,H" {
		t.Errorf("unstablePaths = %v, muốn [B D F G H]: C khác trích dẫn / outcome do model; E lỗi gọi model; F, G, H khác outcome / ứng viên / xung đột ở câu không gọi model", got)
	}
	inv := invalidRuns([]askRecord{
		{System: sysB1, Run: 1, CaseID: "1", ModelCalls: 1}, {System: sysB1, Run: 1, CaseID: "2", ModelCalls: 1, ModelError: true},
		{System: sysB2Plus, Run: 1, CaseID: "1", ModelCalls: 1},
		{System: sysEACL, Run: 1, CaseID: "1"}, // không cần model: không vào mẫu số
	})
	if len(inv) != 1 || !strings.Contains(inv[0], "B1 lượt 1: 1/2") {
		t.Errorf("invalidRuns = %v, muốn đúng một lượt vô hiệu của B1 (1/2 > 2%%)", inv)
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

// pass_count / k từng ca; ca chập chờn = đạt ở một số lượt; luật k của pilot: hệ cổng có ≥ 2 ca chập
// chờn ⇒ 5, không thì 3 — hệ chỉ-báo-cáo không ảnh hưởng.
func TestFlakyAndPilotK(t *testing.T) {
	rep := func(pass ...bool) evals.Report {
		var r evals.Report
		for i, p := range pass {
			r.Results = append(r.Results, evals.CaseResult{ID: string(rune('A' + i)), Pass: p})
		}
		return r
	}
	graded := map[string][]evals.Report{
		sysEACL:   {rep(true, true, false), rep(true, true, false), rep(true, true, false)},
		sysB2Plus: {rep(true, false, true), rep(false, false, true), rep(true, true, true)},
		sysB2:     {rep(true, false, false), rep(false, true, false), rep(true, true, false)},
	}
	pc := passCounts(graded)
	if pc[sysB2Plus]["A"] != 2 || pc[sysB2Plus]["C"] != 3 || pc[sysEACL]["C"] != 0 {
		t.Errorf("pass_count = %v", pc)
	}
	f := flakyCases(pc, 3)
	if len(f[sysEACL]) != 0 || strings.Join(f[sysB2Plus], ",") != "A,B" || strings.Join(f[sysB2], ",") != "A,B" {
		t.Errorf("ca chập chờn = %v, muốn EACL [], B2+ [A B], B2 [A B]", f)
	}
	if pilotK(f) != 5 {
		t.Errorf("B2+ có 2 ca chập chờn: muốn k = 5")
	}
	if pilotK(map[string][]string{sysB2Plus: {"A"}, sysB2: {"A", "B", "C"}, sysB1: {"X"}}) != 3 {
		t.Errorf("hệ cổng chỉ 1 ca chập chờn mỗi hệ (B2 chỉ báo cáo): muốn k = 3")
	}
	if pilotK(map[string][]string{sysB1: {"X", "Y"}}) != 5 {
		t.Errorf("B1 có 2 ca chập chờn: muốn k = 5")
	}
	if pilotK(map[string][]string{sysEACL: {"X", "Y"}}) != 5 {
		t.Errorf("EACL có 2 ca chập chờn: muốn k = 5 (ADR-0006 §6 tính cả EACL; bản soát mã S2-6)")
	}
}

// writeNullWindows ghi hồ sơ cửa sổ B1 = null cho mọi ca của bộ 117 (B1 chỉ nhận entities.json + tài liệu).
func writeNullWindows(t *testing.T) string {
	t.Helper()
	suite, err := evals.LoadWinSuite("../cases-independent.yaml")
	if err != nil {
		t.Fatal(err)
	}
	var list []map[string]any
	for _, c := range suite.Cases {
		list = append(list, map[string]any{"case_id": c.ID, "b1_window": nil})
	}
	b, _ := json.Marshal(list)
	p := filepath.Join(t.TempDir(), "cua-so.json")
	if err := os.WriteFile(p, b, 0o644); err != nil {
		t.Fatal(err)
	}
	return p
}

// S1-1 (bản soát mã độc lập 20/10 đêm): mọi lời gọi model lỗi thì baseline "trượt" vì hạ tầng và EACL
// "thắng" — G7 KHÔNG được ĐẠT. Lượt mà > 2% câu cần model của một hệ lỗi khi gọi model là lượt VÔ HIỆU
// (hợp đồng G7 §10); ca đó mang nhãn PROVIDER.
func TestProviderOutageMakesRunInvalid(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
		_, _ = io.WriteString(w, `{"type":"error"}`)
	}))
	defer srv.Close()
	clearModelEnv(t)
	for k, v := range map[string]string{"EAIS_MODEL_PROVIDER": "anthropic", "EAIS_MODEL": "claude-sonnet-5-5", "EAIS_MODEL_API_KEY": "k",
		"EAIS_MODEL_BASE_URL": srv.URL, "EAIS_MODEL_MAX_RETRIES": "0"} {
		t.Setenv(k, v)
	}
	cfg := baseCfg(genData(t))
	cfg.k, cfg.systems, cfg.manifest, cfg.out = 3, sysB2Plus+","+sysB1, writeNullWindows(t), t.TempDir()
	if code := run(cfg); code == 0 {
		t.Errorf("mọi lời gọi model lỗi 503: mã thoát 0 (ĐẠT) là sai")
	}
	sum, _ := os.ReadFile(filepath.Join(cfg.out, "ket-qua.md"))
	if !strings.Contains(string(sum), "LƯỢT VÔ HIỆU (lỗi gọi model)") {
		t.Errorf("ket-qua.md phải báo lượt vô hiệu vì lỗi gọi model:\n%s", sum)
	}
	// Phán quyết theo LUẬT (độc lập với việc chạy khô): không được ĐẠT.
	var agg struct {
		GatePass bool     `json:"gate_pass"`
		Invalid  []string `json:"invalid_runs"`
	}
	b, _ := os.ReadFile(filepath.Join(cfg.out, "aggregate.json"))
	if err := json.Unmarshal(b, &agg); err != nil {
		t.Fatal(err)
	}
	if agg.GatePass || len(agg.Invalid) == 0 {
		t.Errorf("mọi lời gọi model lỗi: gate_pass = %v, invalid_runs = %v; muốn KHÔNG ĐẠT và có lượt vô hiệu", agg.GatePass, agg.Invalid)
	}
	fail, _ := os.ReadFile(filepath.Join(cfg.out, "failures.jsonl"))
	if !strings.Contains(string(fail), `"first_wrong_layer":"PROVIDER"`) {
		t.Error("ca hỏng vì lỗi gọi model phải mang nhãn PROVIDER trong failures.jsonl")
	}
}

// Trần ngân sách giữ trong MÃ (hợp đồng G7 §9; bản soát mã S3-4): chi tiêu tính từ usage × bảng giá; chạm
// trần thì không gọi model nữa và lần chạy VÔ HIỆU. Mỗi phản hồi giả báo 1 triệu token vào = 2 USD với
// Sonnet 5.5, trần 15 USD ⇒ tối đa 8 lời gọi tới được nhà cung cấp.
func TestBudgetCapStopsCalls(t *testing.T) {
	var mu sync.Mutex
	n := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		n++
		mu.Unlock()
		_, _ = io.WriteString(w, `{"content":[{"type":"text","text":"{\"answer\": \"x\", \"evidence_ids\": []}"}],"usage":{"input_tokens":1000000,"output_tokens":0}}`)
	}))
	defer srv.Close()
	clearModelEnv(t)
	for k, v := range map[string]string{"EAIS_MODEL_PROVIDER": "anthropic", "EAIS_MODEL": "claude-sonnet-5-5", "EAIS_MODEL_API_KEY": "k",
		"EAIS_MODEL_BASE_URL": srv.URL, "EAIS_MODEL_MAX_RETRIES": "0"} {
		t.Setenv(k, v)
	}
	cfg := baseCfg(genData(t))
	cfg.k, cfg.budget, cfg.out = 1, 15, t.TempDir()
	run(cfg)
	mu.Lock()
	got := n
	mu.Unlock()
	if got > 8 {
		t.Errorf("nhà cung cấp nhận %d lời gọi; trần 15 USD / 2 USD mỗi lời gọi ⇒ tối đa 8", got)
	}
	var m manifestInfo
	b, _ := os.ReadFile(filepath.Join(cfg.out, "manifest.json"))
	if err := json.Unmarshal(b, &m); err != nil {
		t.Fatal(err)
	}
	if !m.BudgetHit || m.SpentUSD < 15 || len(m.Invalid) == 0 {
		t.Errorf("manifest: budget_hit %v, spent %.2f, invalid %v; muốn chạm trần và lần chạy vô hiệu", m.BudgetHit, m.SpentUSD, m.Invalid)
	}
}

// Endpoint không phải API chính thức ⇒ CHẠY KHÔ: mã thoát 2, manifest ghi dry_run + endpoint, kể cả khi
// mọi lời gọi thành công. Manifest ghi PackVersion + băm mọi tệp pack; không có đường dẫn tuyệt đối.
func TestDryRunIsNotEvidence(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.WriteString(w, `{"content":[{"type":"text","text":"{\"answer\": \"x\", \"evidence_ids\": []}"}],"usage":{"input_tokens":10,"output_tokens":5}}`)
	}))
	defer srv.Close()
	clearModelEnv(t)
	for k, v := range map[string]string{"EAIS_MODEL_PROVIDER": "anthropic", "EAIS_MODEL": "claude-sonnet-5-5", "EAIS_MODEL_API_KEY": "k", "EAIS_MODEL_BASE_URL": srv.URL} {
		t.Setenv(k, v)
	}
	cfg := baseCfg(genData(t))
	abs, _ := filepath.Abs("../cases-independent.yaml")
	cfg.cases, cfg.out = abs, t.TempDir()
	if code := run(cfg); code != 2 {
		t.Errorf("chạy qua endpoint không chính thức: mã thoát %d, muốn 2 (chạy khô)", code)
	}
	var m manifestInfo
	b, _ := os.ReadFile(filepath.Join(cfg.out, "manifest.json"))
	if err := json.Unmarshal(b, &m); err != nil {
		t.Fatal(err)
	}
	if !m.DryRun || m.Endpoint != srv.URL || m.PackVersion == "" {
		t.Errorf("manifest: dry_run %v, endpoint %q, pack_version %q", m.DryRun, m.Endpoint, m.PackVersion)
	}
	for k := range m.Files {
		if filepath.IsAbs(k) {
			t.Errorf("manifest có đường dẫn tuyệt đối: %s", k)
		}
	}
	packs, _ := filepath.Glob("../../pack/*.yaml")
	for _, f := range packs {
		if _, ok := m.Files[filepath.ToSlash(f)]; !ok {
			t.Errorf("manifest thiếu băm tệp pack %s", f)
		}
	}
	sum, _ := os.ReadFile(filepath.Join(cfg.out, "ket-qua.md"))
	if !strings.Contains(string(sum), "CHẠY KHÔ") {
		t.Error("ket-qua.md phải ghi CHẠY KHÔ")
	}
}

// Không ghi đè bằng chứng cũ; case_id trùng trong hồ sơ cửa sổ là lỗi.
func TestEvidenceDirAndWindowGuards(t *testing.T) {
	clearModelEnv(t)
	cfg := baseCfg(genData(t))
	cfg.out = t.TempDir()
	if err := os.WriteFile(filepath.Join(cfg.out, "manifest.json"), []byte("{}"), 0o644); err != nil {
		t.Fatal(err)
	}
	if code := run(cfg); code != 2 {
		t.Errorf("thư mục bằng chứng đã có tệp: mã thoát %d, muốn 2 (lỗi, không ghi đè)", code)
	}
	if b, _ := os.ReadFile(filepath.Join(cfg.out, "manifest.json")); string(b) != "{}" {
		t.Error("bằng chứng cũ bị ghi đè")
	}
	loc, _ := time.LoadLocation("Asia/Ho_Chi_Minh")
	p := filepath.Join(t.TempDir(), "trung.json")
	_ = os.WriteFile(p, []byte(`[{"case_id":"A","b1_window":null},{"case_id":"A","b1_window":{"from":"2026-10-18","to":"2026-10-20"}}]`), 0o644)
	if _, err := loadWindows(p, loc); err == nil {
		t.Error("case_id trùng trong hồ sơ cửa sổ: muốn lỗi")
	}
}
