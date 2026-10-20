// g7run chạy cổng G7 VALUE của WIN MVP (docs/04-core-mvp.md §6; luật D5 sửa đổi ở
// docs/adr/0005-g7-hop-dong-va-sua-d5.md): CÙNG dữ liệu tenant A và CÙNG bộ ca, chạy EACL và các
// baseline mỗi hệ k lượt bằng CÙNG một model, rồi so EACL với từng baseline. Chấm rò rỉ canary và
// claim không căn cứ bằng bộ đếm độc lập (evals.G7Counter), không bằng bộ chuẩn hoá của EACL.
//
//	go run ./tenants/construction/win/gen/cmd/gen -world tenants/construction/win/world.yaml -out /tmp/win -profile functional -seed 42
//	go run ./tenants/construction/win/g7run -data /tmp/win -cases <bộ ca> -manifest <hồ sơ ca> -out <thư mục bằng chứng>
//
// Hệ (-systems, ADR-0005 §2): baseline cổng B2+ ("b2plus") và B1 ("b1", cần -manifest có b1_window
// từng ca); chỉ báo cáo B2 ("b2") và B2″ ("b2pp"). G7 chỉ ĐẠT khi đạt với CẢ hai baseline cổng.
//
// Không có model thật thì G7 là CHẶN (docs/04 §6): provider rỗng hoặc "extractive" vẫn chạy hết
// ống nối bằng model giả tất định để thấy mọi mắt xích nối đúng, nhưng kết quả KHÔNG phải bằng chứng.
// Model thật cấu hình bằng biến môi trường (models.ConfigFromEnv: EAIS_MODEL_PROVIDER, EAIS_MODEL,
// EAIS_MODEL_API_KEY, EAIS_MODEL_BASE_URL, EAIS_MODEL_THINKING, EAIS_MODEL_EFFORT…); cờ -provider ghi
// đè EAIS_MODEL_PROVIDER. Thiếu model hay khoá là lỗi cấu hình, thoát ngay. Mọi hệ dùng CHUNG một
// gateway; không gửi temperature (ADR-0005 §1). -smoke N chạy N ca đầu một lượt mỗi hệ để kiểm
// provider nhận tham số, trước khi chi tiền cho lượt chạy thật.
//
// Cách chấm từng ca (-grade): mặc định "content" chấm lại câu trả lời của MỌI hệ chỉ theo NỘI DUNG
// (evals.G7ContentReport). "full" dùng nguyên kết quả RunWin, tức chấm cả route / outcome / model_calls /
// ứng viên / conflict / trích dẫn — những nhãn mà baseline không có nên trượt theo thiết kế, làm D5
// không bao giờ đỏ; chỉ dùng để đối chiếu.
//
// Mã thoát: 0 = ĐẠT bằng model thật · 1 = KHÔNG ĐẠT bằng model thật · 2 = CHẶN (model giả) hoặc lỗi.
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strings"
	"sync/atomic"
	"time"

	"github.com/luannt2002/enterprise-ai-systems/core/contracts"
	"github.com/luannt2002/enterprise-ai-systems/core/pipeline"
	"github.com/luannt2002/enterprise-ai-systems/core/tenant"
	"github.com/luannt2002/enterprise-ai-systems/data/entityresolution"
	"github.com/luannt2002/enterprise-ai-systems/data/ingestion"
	"github.com/luannt2002/enterprise-ai-systems/evals"
	"github.com/luannt2002/enterprise-ai-systems/evals/baseline"
	"github.com/luannt2002/enterprise-ai-systems/intelligence/models"
	"github.com/luannt2002/enterprise-ai-systems/intelligence/rag"
	"go.yaml.in/yaml/v3"
)

// blockedLine là dòng đầu khi chạy bằng model giả: G7 vẫn CHẶN, kết quả chỉ để thử ống nối.
const blockedLine = "G7: CHẶN — chưa có model thật; đây là chạy thử ống nối bằng model giả tất định, KHÔNG phải bằng chứng G7"

// quantityUnits là đơn vị lượng nhiên liệu của tenant: số mang đơn vị này trong TÀI LIỆU không được
// coi là có căn cứ (docs/04 §6 G5). Viết tay ở đây, không lấy từ lexicon của pack: bộ đếm độc lập.
var quantityUnits = []string{"L", "lít", "lit", "m3", "m³", "khối"}

// Hai cách chấm từng ca của G7 (xem đầu tệp).
const (
	gradeContent = "content"
	gradeFull    = "full"
)

type config struct {
	data, pack, world, cases, tenant, provider, grade string
	systems, manifest, out, contract                  string
	k, topK, smoke                                    int
	pilot                                             bool
}

func main() {
	var cfg config
	flag.StringVar(&cfg.data, "data", "", "thư mục dữ liệu do bộ sinh xuất ra")
	flag.StringVar(&cfg.pack, "pack", "tenants/construction/pack", "pack của tenant")
	flag.StringVar(&cfg.world, "world", "tenants/construction/win/world.yaml", "world.yaml")
	flag.StringVar(&cfg.cases, "cases", "tenants/construction/win/cases-*.yaml", "các tệp ca WIN")
	flag.StringVar(&cfg.tenant, "tenant", "abc-infra", "tenant được chấm")
	flag.IntVar(&cfg.k, "k", evals.G7MinRuns, "số lượt mỗi hệ")
	flag.IntVar(&cfg.topK, "topk", baseline.DefaultTopK, "số đoạn baseline RAG đưa cho model mỗi câu hỏi")
	flag.StringVar(&cfg.grade, "grade", gradeContent, `cách chấm từng ca: "content" (theo nội dung câu trả lời, mặc định) hoặc "full" (nguyên kết quả RunWin)`)
	flag.StringVar(&cfg.provider, "provider", "", `provider model, ghi đè EAIS_MODEL_PROVIDER; rỗng hoặc "extractive" = model giả tất định, G7 CHẶN`)
	flag.StringVar(&cfg.systems, "systems", sysB2Plus+","+sysB1, `baseline so với EACL, phân cách dấu phẩy: "b2plus", "b1" (cổng); "b2", "b2pp" (chỉ báo cáo)`)
	flag.StringVar(&cfg.manifest, "manifest", "", "hồ sơ ca có b1_window của từng ca (bắt buộc cho B1)")
	flag.StringVar(&cfg.out, "out", "", "thư mục ghi bằng chứng thô: manifest.json, raw-runs.jsonl, scored.jsonl, aggregate.json, failures.jsonl, cost.json, ket-qua.md")
	flag.StringVar(&cfg.contract, "contract", "docs/g7/HOP-DONG-G7.md", "hợp đồng G7 — băm vào manifest")
	flag.BoolVar(&cfg.pilot, "pilot", false, "lượt pilot (ADR-0006 §6): cần -k 5; in k của lượt chính theo luật đăng ký trước")
	flag.IntVar(&cfg.smoke, "smoke", 0, "thử khói: chạy N ca đầu MỘT lượt mỗi hệ để kiểm provider nhận tham số, rồi thoát")
	flag.Parse()
	if cfg.data == "" || cfg.k < 1 || cfg.topK < 1 || cfg.smoke < 0 || (cfg.pilot && cfg.k != 5) || (cfg.grade != gradeContent && cfg.grade != gradeFull) {
		fmt.Fprintln(os.Stderr, `thiếu -data, hoặc -k < 1, hoặc -topk < 1, hoặc -smoke < 0, hoặc -pilot mà -k khác 5, hoặc -grade không phải "content" / "full"`)
		os.Exit(2)
	}
	os.Exit(run(cfg))
}

func run(cfg config) int {
	started := time.Now()
	systems, err := parseSystems(cfg.systems)
	if err != nil {
		fmt.Fprintln(os.Stderr, "LỖI:", err)
		return 2
	}
	mc := models.ConfigFromEnv()
	if cfg.provider != "" {
		mc.Provider = cfg.provider
	}
	gw, err := models.New(mc) // thiếu model / khoá / tham số sai cho provider thật: lỗi cấu hình, thoát ngay
	if err != nil {
		fmt.Fprintln(os.Stderr, "LỖI:", err)
		return 2
	}
	_, fake := gw.(*models.Extractive)
	if fake {
		fmt.Println(blockedLine)
	}
	s, err := build(cfg, gw, systems)
	if err != nil {
		fmt.Fprintln(os.Stderr, "LỖI:", err)
		return 2
	}
	sink := newEvidenceSink(s.cases)
	if m, ok := gw.(*models.Messages); ok {
		m.Trace = sink.trace
	}
	ctx := context.Background()
	if cfg.smoke > 0 {
		return smoke(ctx, cfg, mc, gw, s, sink, fake, started)
	}

	full := map[string][]evals.Report{}
	for i := 0; i < cfg.k; i++ {
		for _, sys := range s.order {
			rec, err := newRecorder(s.askers[sys], sys, i+1, sink)
			if err != nil {
				fmt.Fprintln(os.Stderr, "LỖI:", err)
				return 2
			}
			full[sys] = append(full[sys], evals.RunWin(ctx, rec, s.cases, s.principals))
		}
	}
	graded := map[string][]evals.Report{}
	counts := map[string]evals.G7Counts{}
	for _, sys := range s.order {
		for _, r := range full[sys] {
			if cfg.grade == gradeContent {
				r = evals.G7ContentReport(s.cases, r)
			}
			graded[sys] = append(graded[sys], r)
		}
		counts[sys] = s.counter.Count(s.cases, graded[sys])
	}

	var out strings.Builder
	say := func(f string, a ...any) { fmt.Fprintf(&out, f, a...) }
	provider := mc.Provider
	if provider == "" {
		provider = "extractive (mặc định)"
	} else if mc.Model != "" {
		provider += " · model " + mc.Model
	}
	var labels []string
	for _, sys := range s.order {
		labels = append(labels, sysLabel[sys])
	}
	say("tenant %s · %d ca · k = %d · provider %s · hệ: %s · baseline RAG top-%d trên %d đoạn quan sát + %d tài liệu\n",
		cfg.tenant, len(s.cases), cfg.k, provider, strings.Join(labels, ", "), cfg.topK, s.paragraphs, s.documents)
	say("canary suy ra từ world.yaml (chữ của tenant khác, không có ở tenant được chấm): %s\n", strings.Join(s.canaries, " "))
	if cfg.grade == gradeFull {
		say("chấm từng ca: theo TOÀN BỘ kỳ vọng của ca (RunWin) — baseline không có route / outcome / model_calls / ứng viên / conflict / trích dẫn của EACL nên trượt theo thiết kế\n")
	} else {
		say("chấm từng ca: theo NỘI DUNG câu trả lời, cùng tiêu chí cho mọi hệ (evals.G7ContentReport)\n")
		var ref []string
		for _, sys := range s.order {
			ref = append(ref, fmt.Sprintf("%s %d/%d", sysLabel[sys], passed(full[sys][0]), len(s.cases)))
		}
		say("tham khảo, theo TOÀN BỘ kỳ vọng của ca (RunWin, lượt 1): %s\n", strings.Join(ref, " · "))
	}
	for _, b := range s.blocked {
		say("HỆ CỔNG KHÔNG CHẠY: %s\n", b)
	}
	unstable := unstablePaths(sink.recs)
	if len(unstable) > 0 {
		say("LƯỢT VÔ HIỆU (ADR-0005 §5): đường tất định của EACL khác nhau giữa các lượt ở %d ca: %s\n", len(unstable), strings.Join(unstable, ", "))
	}
	verdicts := map[string]evals.G7Verdict{}
	for _, b := range s.order[1:] {
		v, err := evals.CompareG7(graded[sysEACL], graded[b], counts[sysEACL])
		if err != nil {
			fmt.Fprintln(os.Stderr, "LỖI:", err)
			return 2
		}
		verdicts[b] = v
		role := "chỉ báo cáo"
		if slices.Contains(gateSystems, b) {
			role = "baseline cổng"
		}
		say("\n=== EACL so với %s (%s) ===\n%s", sysLabel[b], role, v.Format())
		say("Bộ đếm độc lập, %s (chỉ báo cáo): %d câu · rò rỉ canary %d · số không căn cứ %d\n", sysLabel[b], counts[b].Answers, counts[b].Leaks, counts[b].Unsupported)
	}
	pass := gatePass(s.order, s.blocked, unstable, verdicts)
	flaky := flakyCases(passCounts(graded), cfg.k)
	for _, sys := range s.order {
		say("ca chập chờn của %s (đạt ở 0 < n < %d lượt): %d %s\n", sysLabel[sys], cfg.k, len(flaky[sys]), strings.Join(flaky[sys], " "))
	}
	if cfg.pilot {
		say("PILOT (ADR-0006 §6): k của lượt chính = %d (hệ cổng có ≥ 2 ca chập chờn thì 5, không thì 3). Pilot KHÔNG phải bằng chứng G7.\n", pilotK(flaky))
	}
	say("\nBộ đếm độc lập, EACL: %d câu · rò rỉ canary %d · số không căn cứ %d\n", counts[sysEACL].Answers, counts[sysEACL].Leaks, counts[sysEACL].Unsupported)
	for _, e := range counts[sysEACL].Examples {
		say("  - %s\n", e)
	}
	switch {
	case fake:
		say("\n%s\n", blockedLine)
	case pass:
		say("\nG7: ĐẠT với mọi baseline cổng (%s)\n", strings.Join(gateSystems, ", "))
	default:
		say("\nG7: KHÔNG ĐẠT — xem luật, hệ cổng thiếu hoặc lượt vô hiệu ở trên\n")
	}
	fmt.Print(out.String())

	if cfg.out != "" {
		if err := saveRun(cfg, mc, gw, fake, s, sink, full, graded, verdicts, counts, unstable, started, out.String()); err != nil {
			fmt.Fprintln(os.Stderr, "LỖI ghi bằng chứng:", err)
			return 2
		}
		fmt.Println("bằng chứng thô:", cfg.out)
	}
	switch {
	case fake:
		return 2
	case pass:
		return 0
	}
	return 1
}

// gatePass là kết luận của G7 (ADR-0005 §5–6): ĐẠT chỉ khi MỌI baseline cổng đều chạy được và đều
// đạt, không hệ cổng nào bị chặn, không lượt nào vô hiệu. Baseline chỉ-báo-cáo không ảnh hưởng.
func gatePass(order, blocked, unstable []string, verdicts map[string]evals.G7Verdict) bool {
	if len(blocked) > 0 || len(unstable) > 0 {
		return false
	}
	for _, g := range gateSystems {
		v, ok := verdicts[g]
		if !slices.Contains(order, g) || !ok || !v.Pass {
			return false
		}
	}
	return true
}

// saveRun gom bằng chứng của một lần chạy và ghi ra cfg.out.
func saveRun(cfg config, mc models.Config, gw models.Gateway, fake bool, s *setup, sink *evidenceSink, full, graded map[string][]evals.Report,
	verdicts map[string]evals.G7Verdict, counts map[string]evals.G7Counts, unstable []string, started time.Time, summary string) error {
	m := newManifest(cfg, mc, gw, fake, s.order, len(s.cases), s.files)
	m.StartedAt, m.FinishedAt = started.Format(time.RFC3339), time.Now().Format(time.RFC3339)
	m.Blocked, m.Unstable = s.blocked, unstable
	var scored []scoredRow
	var failures []failureRow
	for _, sys := range s.order {
		for i := range graded[sys] {
			for j, g := range graded[sys][i].Results {
				f := full[sys][i].Results[j]
				scored = append(scored, scoredRow{System: sys, Run: i + 1, CaseID: g.ID, Class: g.FailureClass,
					Pass: g.Pass, Problems: g.Problems, FullPass: f.Pass, FullProblems: f.Problems})
				if !g.Pass {
					failures = append(failures, failureRow{System: sys, Run: i + 1, CaseID: g.ID, Class: g.FailureClass,
						Problems: g.Problems, FirstWrongLayer: "CHƯA PHÂN LOẠI"})
				}
			}
		}
	}
	pc := passCounts(graded)
	agg := map[string]any{"verdicts": verdicts, "counts": counts, "eacl_unstable_cases": unstable, "blocked": s.blocked, "fake_model": fake,
		"pass_count": pc, "flaky_cases": flakyCases(pc, cfg.k), "k": cfg.k, "pilot": cfg.pilot}
	if cfg.pilot {
		agg["pilot_main_k"] = pilotK(flakyCases(pc, cfg.k))
	}
	return writeEvidence(cfg.out, m, sink.recs, scored, agg, failures, costs(sink.recs, mc.Model, s.order), summary)
}

// smoke chạy N ca đầu MỘT lượt mỗi hệ: mọi câu hỏi phải không lỗi, và với model thật mọi lời gọi
// model phải kết thúc bằng HTTP 200 — tức provider NHẬN đúng bộ tham số sẽ đóng băng.
func smoke(ctx context.Context, cfg config, mc models.Config, gw models.Gateway, s *setup, sink *evidenceSink, fake bool, started time.Time) int {
	cases := s.cases[:min(cfg.smoke, len(s.cases))]
	ok := true
	var out strings.Builder
	for _, sys := range s.order {
		rec, err := newRecorder(s.askers[sys], sys, 0, sink)
		if err != nil {
			fmt.Fprintln(os.Stderr, "LỖI:", err)
			return 2
		}
		before := len(sink.recs)
		evals.RunWin(ctx, rec, cases, s.principals)
		for _, r := range sink.recs[before:] {
			var st []string
			in, outTok := 0, 0
			for _, c := range r.Calls {
				st = append(st, fmt.Sprint(c.Status))
				in, outTok = in+c.InputTokens, outTok+c.OutputTokens
			}
			bad := r.Error != ""
			if !fake && r.ModelCalls > 0 && (len(r.Calls) == 0 || r.Calls[len(r.Calls)-1].Status != 200) {
				bad = true
			}
			ok = ok && !bad
			fmt.Fprintf(&out, "%-5s %s · lời gọi model %d · HTTP [%s] · token vào %d ra %d · lỗi %q\n", sysLabel[sys], r.CaseID, r.ModelCalls, strings.Join(st, " "), in, outTok, r.Error)
		}
	}
	for _, b := range s.blocked {
		fmt.Fprintf(&out, "HỆ CỔNG KHÔNG CHẠY: %s\n", b)
	}
	res := "THỬ KHÓI: ĐẠT"
	if !ok {
		res = "THỬ KHÓI: KHÔNG ĐẠT"
	}
	fmt.Fprintln(&out, res)
	fmt.Print(out.String())
	if cfg.out != "" {
		m := newManifest(cfg, mc, gw, fake, s.order, len(cases), s.files)
		m.StartedAt, m.FinishedAt, m.Blocked = started.Format(time.RFC3339), time.Now().Format(time.RFC3339), s.blocked
		if err := writeEvidence(cfg.out, m, sink.recs, nil, map[string]any{"smoke": res}, nil, costs(sink.recs, mc.Model, s.order), out.String()); err != nil {
			fmt.Fprintln(os.Stderr, "LỖI ghi bằng chứng:", err)
			return 2
		}
	}
	switch {
	case fake:
		fmt.Println(blockedLine)
		return 2
	case ok:
		return 0
	}
	return 1
}

// passed đếm số ca đạt trong một Report.
func passed(r evals.Report) int {
	n := 0
	for _, x := range r.Results {
		if x.Pass {
			n++
		}
	}
	return n
}

// setup là mọi thứ một lượt G7 cần, dựng một lần.
type setup struct {
	askers                map[string]evals.Asker
	order                 []string // EACL trước, rồi các baseline chạy được, theo thứ tự -systems
	blocked               []string // hệ cổng không chạy được và lý do
	principals            map[string]contracts.Principal
	cases                 []evals.WinCase
	files                 []string // tệp đầu vào cần băm vào manifest
	counter               *evals.G7Counter
	canaries              []string
	paragraphs, documents int
}

// build dựng EACL (như winrun.Run), các baseline và bộ đếm độc lập trên CÙNG một bộ dữ liệu và CÙNG
// một gateway model.
func build(cfg config, gw models.Gateway, systems []string) (*setup, error) {
	ctx := context.Background()
	rt, err := tenant.Load(cfg.pack)
	if err != nil {
		return nil, err
	}
	sm, err := ingestion.LoadSources(filepath.Join(cfg.pack, "sources.yaml"), rt)
	if err != nil {
		return nil, err
	}
	dir := filepath.Join(cfg.data, cfg.tenant)
	res, err := ingestion.Ingest(ctx, dir, rt, sm, ingestion.PdfToText{})
	if err != nil {
		return nil, err
	}
	w, err := loadWorld(cfg.world)
	if err != nil {
		return nil, err
	}
	ref, err := time.ParseInLocation("2006-01-02", w.ReferenceDate, rt.Location)
	if err != nil {
		return nil, err
	}
	principals := map[string]contracts.Principal{}
	for _, tn := range w.Tenants {
		if tn.ID != cfg.tenant {
			continue
		}
		for _, p := range tn.Principals {
			pr := contracts.Principal{ID: p.ID, TenantID: contracts.TenantID(cfg.tenant), Kind: contracts.PrincipalUser, Roles: []string{p.Role}}
			if p.EntityScope != "" {
				pr.Attributes = map[string]string{"entity_scope": p.EntityScope}
			}
			principals[p.ID] = pr
		}
	}
	if len(principals) == 0 {
		return nil, fmt.Errorf("world.yaml không có principal nào của tenant %s", cfg.tenant)
	}
	directory, err := ingestion.LoadDirectory(dir, sm)
	if err != nil {
		return nil, err
	}
	docs, err := rag.LoadMarkdown(filepath.Join(dir, "docs"), cfg.tenant)
	if err != nil {
		return nil, err
	}
	files, _ := filepath.Glob(cfg.cases)
	sort.Strings(files)
	if len(files) == 0 {
		return nil, fmt.Errorf("không có tệp ca nào khớp %q", cfg.cases)
	}
	var cases []evals.WinCase
	for _, f := range files {
		suite, err := evals.LoadWinSuite(f)
		if err != nil {
			return nil, err
		}
		cases = append(cases, suite.Cases...)
	}

	// EACL: gateway bọc một bộ đếm lời gọi (cổng G4 đo bằng gateway, không bằng trace của hệ).
	counting := &countingGateway{inner: gw}
	now := ref.Add(10 * time.Hour)
	askers := map[string]evals.Asker{sysEACL: eaisAsker{a: &pipeline.Ask{RT: rt, Deps: pipeline.Deps{
		Directory: directory, Facts: res.Facts(), Knowledge: rag.NewMemoryIndex(cfg.tenant, docs), Model: counting,
		Now: func() time.Time { return now },
	}}, gw: counting}}
	st := &setup{askers: askers, order: []string{sysEACL}, principals: principals, cases: cases, documents: len(docs),
		files: append(append([]string(nil), files...), cfg.world, cfg.manifest, filepath.Join(cfg.pack, "metrics.yaml"),
			filepath.Join(cfg.pack, "sources.yaml"), cfg.contract)}

	// Baseline RAG: mỗi quan sát của tenant một đoạn văn bản + cùng các tài liệu SOP, cùng gateway.
	names := map[string]string{}
	byID := map[string]entityresolution.Entity{}
	for _, e := range directory {
		names[e.ID] = e.Name
		byID[e.ID] = e
	}
	paras := func(plus bool) []string {
		out := make([]string, len(res.Observations))
		for i, o := range res.Observations {
			if plus {
				out[i] = paragraphPlus(rt, byID, o)
			} else {
				out[i] = paragraph(rt, names, o)
			}
		}
		return out
	}
	st.paragraphs = len(res.Observations)
	for _, sys := range systems[1:] {
		switch sys {
		case sysB2, sysB2Plus, sysB2PP:
			b := baseline.New(cfg.tenant, paras(sys != sysB2), docs, gw)
			b.TopK = cfg.topK
			switch sys {
			case sysB2Plus:
				b.Preamble = todayLine(ref)
			case sysB2PP:
				rules, err := domainRules(cfg.pack)
				if err != nil {
					return nil, err
				}
				b.Preamble = todayLine(ref) + "\n" + rules
			}
			askers[sys] = b
		case sysB1:
			if cfg.manifest == "" {
				st.blocked = append(st.blocked, "B1: thiếu -manifest (b1_window từng ca, do người soạn ca gán)")
				continue
			}
			windows, err := loadWindows(cfg.manifest, rt.Location)
			if err != nil {
				return nil, err
			}
			rows, err := readRawRows(dir, rt.Location, ingestion.PdfToText{})
			if err != nil {
				return nil, err
			}
			fixed, err := fixedEvidence(dir, docs)
			if err != nil {
				return nil, err
			}
			d, err := newDirectLLM(gw, rows, fixed, cases, windows, todayLine(ref))
			if err != nil {
				return nil, err
			}
			askers[sys] = d
		}
		st.order = append(st.order, sys)
	}

	// Bộ đếm độc lập: số được phép lấy từ world.yaml (không từ dữ liệu nền sinh ra: vài nghìn giá
	// trị nền phủ gần hết dải số thì "được phép" thành vô nghĩa); canary là chữ chỉ có ở tenant khác.
	ownRecords, ownDocs := w.texts(cfg.tenant)
	var others []string
	for _, tn := range w.Tenants {
		if tn.ID != cfg.tenant {
			r, d := w.texts(tn.ID)
			others = append(append(others, r...), d...)
		}
	}
	canaries := evals.DeriveCanaries(append(append([]string(nil), ownRecords...), ownDocs...), others)
	if len(canaries) == 0 {
		return nil, fmt.Errorf("world.yaml không có canary nào của tenant khác: bộ đếm rò rỉ sẽ đếm được 0 vì mù, không vì sạch")
	}
	st.counter = evals.NewG7Counter(evals.G7CounterConfig{
		Canaries: canaries, Records: ownRecords, Documents: ownDocs, QuantityUnits: quantityUnits,
	})
	st.canaries = canaries
	return st, nil
}

// paragraph viết một quan sát thành MỘT đoạn văn bản một dòng cho baseline: ngày, tên thực thể (danh
// mục thiết bị, ví dụ "Máy xúc EX-03"), tên chỉ số, nguyên văn lượng như nguồn ghi ("3850 L",
// "máy 03 sáng 2k"), tên nguồn.
//
// Baseline nhận quan sát ĐÃ QUA bộ nạp của EAIS (thực thể và ngày đã chuẩn hoá, bản ghi bẩn đã bị cách
// ly) — mạnh hơn một RAG đọc thẳng tệp nguồn, nên G7 khắt khe hơn thực tế. Đổi sang chữ thô là quyết
// định của chủ dự án. Tên thực thể có mặt vì đo được ích lợi: xem baseline.DefaultTopK.
func paragraph(rt *tenant.Runtime, names map[string]string, o ingestion.Observation) string {
	label := o.Metric
	if d, ok := rt.Metrics.Metric(o.Metric); ok && d.Label != "" {
		label = d.Label
	}
	name := names[o.Entity]
	if name == "" {
		name = o.Entity
	}
	return fmt.Sprintf("Ngày %s, %s: %s %s (%s).", o.On.Format("2006-01-02"), name, label, o.Raw, rt.Metrics.SourceLabel(o.Metric, o.Source))
}

// countingGateway đếm số lời gọi Generate của một hệ, bọc gateway chung.
type countingGateway struct {
	inner models.Gateway
	n     atomic.Int64
}

func (g *countingGateway) Generate(ctx context.Context, req models.Request) (contracts.Draft, error) {
	g.n.Add(1)
	return g.inner.Generate(ctx, req)
}

// eaisAsker đưa luồng hỏi của EAIS vào runner; số lời gọi model đo ở gateway.
type eaisAsker struct {
	a  *pipeline.Ask
	gw *countingGateway
}

func (e eaisAsker) ModelCalls() int  { return int(e.gw.n.Load()) }
func (e eaisAsker) ResetModelCalls() { e.gw.n.Store(0) }

func (e eaisAsker) Ask(ctx context.Context, pr contracts.Principal, q string) (contracts.AskResponse, error) {
	out, err := e.a.Handle(ctx, pr, contracts.AskRequest{ConversationID: "g7", MessageID: "g7", Text: q})
	return out.Response, err
}

// world là phần của world.yaml mà G7 cần (trường khác bỏ qua).
type world struct {
	ReferenceDate string `yaml:"reference_date"`
	Tenants       []struct {
		ID       string `yaml:"id"`
		Projects []struct {
			ID      string   `yaml:"id"`
			Name    string   `yaml:"name"`
			Aliases []string `yaml:"aliases"`
		} `yaml:"projects"`
		Equipment []struct {
			ID        string            `yaml:"id"`
			Name      string            `yaml:"name"`
			Aliases   []string          `yaml:"aliases"`
			SourceIDs map[string]string `yaml:"source_ids"`
		} `yaml:"equipment"`
		Principals []struct {
			ID          string `yaml:"id"`
			Role        string `yaml:"role"`
			EntityScope string `yaml:"entity_scope"`
		} `yaml:"principals"`
	} `yaml:"tenants"`
	Records []struct {
		Tenant string `yaml:"tenant"`
		Entity string `yaml:"entity"`
		On     string `yaml:"on"`
		Source string `yaml:"source"`
		ValueL string `yaml:"value_l"`
		Raw    string `yaml:"raw"`
		By     string `yaml:"by"`
		Ref    string `yaml:"ref"`
		Expect string `yaml:"expect"`
	} `yaml:"records"`
	Documents []struct {
		Tenant string `yaml:"tenant"`
		Title  string `yaml:"title"`
		Body   string `yaml:"body"`
	} `yaml:"documents"`
}

func loadWorld(path string) (world, error) {
	var w world
	b, err := os.ReadFile(path)
	if err != nil {
		return w, err
	}
	if err := yaml.Unmarshal(b, &w); err != nil {
		return w, fmt.Errorf("%s: %w", path, err)
	}
	return w, nil
}

// texts trả chữ của MỘT tenant trong world để bộ đếm tự tách số: records = danh mục (dự án, thiết bị,
// mã theo nguồn) + bản ghi; docs = tài liệu. Bản ghi `expect: quarantine` (bẩn có chủ ý) không phải
// fact nên số của nó không được coi là có căn cứ; id bản ghi của world (R-001…) không bao giờ hiện
// trong câu trả lời nên cũng không vào. Ngày tham chiếu của world có mặt ở mọi tenant.
func (w world) texts(tenantID string) (records, docs []string) {
	records = append(records, w.ReferenceDate)
	for _, t := range w.Tenants {
		if t.ID != tenantID {
			continue
		}
		for _, p := range t.Projects {
			records = append(records, p.ID, p.Name, strings.Join(p.Aliases, " "))
		}
		for _, e := range t.Equipment {
			records = append(records, e.ID, e.Name, strings.Join(e.Aliases, " "))
			for _, code := range e.SourceIDs {
				records = append(records, code)
			}
		}
	}
	for _, r := range w.Records {
		if r.Tenant == tenantID && r.Expect != "quarantine" {
			records = append(records, r.Entity, r.On, r.Source, r.ValueL, r.Raw, r.By, r.Ref)
		}
	}
	for _, d := range w.Documents {
		if d.Tenant == tenantID {
			docs = append(docs, d.Title+"\n"+d.Body)
		}
	}
	return records, docs
}
