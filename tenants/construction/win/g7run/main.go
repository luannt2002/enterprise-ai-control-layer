// g7run chạy cổng G7 VALUE của WIN MVP (docs/04-core-mvp.md §6, docs/adr/0001-win-mvp.md D5): CÙNG
// dữ liệu tenant A và CÙNG bộ ca WIN, chạy EAIS và baseline RAG (evals/baseline) mỗi hệ k lượt bằng
// CÙNG một model, lấy trung vị tỉ lệ đạt theo lớp hỏng rồi áp luật D5. Chấm rò rỉ canary và claim
// không căn cứ bằng bộ đếm độc lập (evals.G7Counter), không bằng bộ chuẩn hoá của EAIS.
//
//	go run ./tenants/construction/win/gen/cmd/gen -world tenants/construction/win/world.yaml -out /tmp/win -profile functional -seed 42
//	go run ./tenants/construction/win/g7run -data /tmp/win -provider <tên provider>
//
// Không có model thật thì G7 là CHẶN (docs/04 §6): provider rỗng hoặc "extractive" vẫn chạy hết
// ống nối bằng model giả tất định để thấy mọi mắt xích nối đúng, nhưng kết quả KHÔNG phải bằng chứng.
// Model thật cấu hình bằng biến môi trường (models.ConfigFromEnv: EAIS_MODEL_PROVIDER, EAIS_MODEL,
// EAIS_MODEL_API_KEY, EAIS_MODEL_BASE_URL); cờ -provider ghi đè EAIS_MODEL_PROVIDER. Thiếu model hay khoá
// là lỗi cấu hình, thoát ngay. Hai hệ dùng CHUNG một gateway, temperature 0 (models.Messages).
//
// Cách chấm từng ca (-grade): mặc định "content" chấm lại câu trả lời của CẢ HAI hệ chỉ theo NỘI DUNG
// (evals.G7ContentReport). "full" dùng nguyên kết quả RunWin, tức chấm cả route / outcome / model_calls /
// ứng viên / conflict / trích dẫn — những nhãn mà baseline RAG không có nên trượt theo thiết kế, làm D5
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
	"sort"
	"strings"
	"sync/atomic"
	"time"

	"github.com/luannt2002/enterprise-ai-systems/core/contracts"
	"github.com/luannt2002/enterprise-ai-systems/core/pipeline"
	"github.com/luannt2002/enterprise-ai-systems/core/tenant"
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
	k, topK                                           int
}

func main() {
	var cfg config
	flag.StringVar(&cfg.data, "data", "", "thư mục dữ liệu do bộ sinh xuất ra")
	flag.StringVar(&cfg.pack, "pack", "tenants/construction/pack", "pack của tenant")
	flag.StringVar(&cfg.world, "world", "tenants/construction/win/world.yaml", "world.yaml")
	flag.StringVar(&cfg.cases, "cases", "tenants/construction/win/cases-*.yaml", "các tệp ca WIN")
	flag.StringVar(&cfg.tenant, "tenant", "abc-infra", "tenant được chấm")
	flag.IntVar(&cfg.k, "k", evals.G7MinRuns, "số lượt mỗi hệ (D5: k = 3, lấy trung vị)")
	flag.IntVar(&cfg.topK, "topk", baseline.DefaultTopK, "số đoạn baseline đưa cho model mỗi câu hỏi")
	flag.StringVar(&cfg.grade, "grade", gradeContent, `cách chấm từng ca: "content" (theo nội dung câu trả lời, mặc định) hoặc "full" (nguyên kết quả RunWin)`)
	flag.StringVar(&cfg.provider, "provider", "", `provider model, ghi đè EAIS_MODEL_PROVIDER; rỗng hoặc "extractive" = model giả tất định, G7 CHẶN`)
	flag.Parse()
	if cfg.data == "" || cfg.k < 1 || cfg.topK < 1 || (cfg.grade != gradeContent && cfg.grade != gradeFull) {
		fmt.Fprintln(os.Stderr, `thiếu -data, hoặc -k < 1, hoặc -topk < 1, hoặc -grade không phải "content" / "full"`)
		os.Exit(2)
	}
	os.Exit(run(cfg))
}

func run(cfg config) int {
	mc := models.ConfigFromEnv()
	if cfg.provider != "" {
		mc.Provider = cfg.provider
	}
	gw, err := models.New(mc) // thiếu model / khoá cho provider thật: lỗi cấu hình, thoát ngay
	if err != nil {
		fmt.Fprintln(os.Stderr, "LỖI:", err)
		return 2
	}
	_, fake := gw.(*models.Extractive)
	if fake {
		fmt.Println(blockedLine)
	}
	s, err := build(cfg, gw)
	if err != nil {
		fmt.Fprintln(os.Stderr, "LỖI:", err)
		return 2
	}

	ctx := context.Background()
	var eaisRuns, baseRuns []evals.Report
	for i := 0; i < cfg.k; i++ {
		eaisRuns = append(eaisRuns, evals.RunWin(ctx, s.eais, s.cases, s.principals))
		baseRuns = append(baseRuns, evals.RunWin(ctx, s.base, s.cases, s.principals))
	}
	eaisFull, baseFull := passed(eaisRuns[0]), passed(baseRuns[0])
	if cfg.grade == gradeContent {
		for i := range eaisRuns {
			eaisRuns[i], baseRuns[i] = evals.G7ContentReport(s.cases, eaisRuns[i]), evals.G7ContentReport(s.cases, baseRuns[i])
		}
	}
	eaisCounts, baseCounts := s.counter.Count(s.cases, eaisRuns), s.counter.Count(s.cases, baseRuns)
	v, err := evals.CompareG7(eaisRuns, baseRuns, eaisCounts)
	if err != nil {
		fmt.Fprintln(os.Stderr, "LỖI:", err)
		return 2
	}

	provider := mc.Provider
	if provider == "" {
		provider = "extractive (mặc định)"
	} else if mc.Model != "" {
		provider += " · model " + mc.Model
	}
	fmt.Printf("tenant %s · %d ca · k = %d · provider %s · baseline top-%d trên %d đoạn quan sát + %d tài liệu\n",
		cfg.tenant, len(s.cases), cfg.k, provider, cfg.topK, s.paragraphs, s.documents)
	fmt.Printf("canary suy ra từ world.yaml (chữ của tenant khác, không có ở tenant được chấm): %s\n", strings.Join(s.canaries, " "))
	if cfg.grade == gradeFull {
		fmt.Println("chấm từng ca: theo TOÀN BỘ kỳ vọng của ca (RunWin) — baseline không có route / outcome / model_calls / ứng viên / conflict / trích dẫn của EAIS nên trượt theo thiết kế")
	} else {
		fmt.Println("chấm từng ca: theo NỘI DUNG câu trả lời, cùng tiêu chí cho cả hai hệ (evals.G7ContentReport)")
		fmt.Printf("tham khảo, theo TOÀN BỘ kỳ vọng của ca (RunWin, lượt 1): EAIS %d/%d · baseline %d/%d\n", eaisFull, len(s.cases), baseFull, len(s.cases))
	}
	fmt.Println()
	fmt.Print(v.Format())
	fmt.Printf("\nBộ đếm độc lập, EAIS: %d câu · rò rỉ canary %d · số không căn cứ %d\n", eaisCounts.Answers, eaisCounts.Leaks, eaisCounts.Unsupported)
	for _, e := range eaisCounts.Examples {
		fmt.Println("  -", e)
	}
	fmt.Printf("Bộ đếm độc lập, baseline (chỉ báo cáo, không có luật): %d câu · rò rỉ canary %d · số không căn cứ %d\n",
		baseCounts.Answers, baseCounts.Leaks, baseCounts.Unsupported)

	switch {
	case fake:
		fmt.Println("\n" + blockedLine)
		return 2
	case v.Pass:
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
	eais, base            evals.Asker
	principals            map[string]contracts.Principal
	cases                 []evals.WinCase
	counter               *evals.G7Counter
	canaries              []string
	paragraphs, documents int
}

// build dựng EAIS (như winrun.Run), baseline và bộ đếm độc lập trên CÙNG một bộ dữ liệu và CÙNG một
// gateway model.
func build(cfg config, gw models.Gateway) (*setup, error) {
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

	// EAIS: gateway bọc một bộ đếm lời gọi (cổng G4 đo bằng gateway, không bằng trace của hệ).
	counting := &countingGateway{inner: gw}
	now := ref.Add(10 * time.Hour)
	eais := eaisAsker{a: &pipeline.Ask{RT: rt, Deps: pipeline.Deps{
		Directory: directory, Facts: res.Facts(), Knowledge: rag.NewMemoryIndex(cfg.tenant, docs), Model: counting,
		Now: func() time.Time { return now },
	}}, gw: counting}

	// Baseline: mỗi quan sát của tenant một đoạn văn bản + cùng các tài liệu SOP, cùng gateway.
	names := map[string]string{}
	for _, e := range directory {
		names[e.ID] = e.Name
	}
	paras := make([]string, len(res.Observations))
	for i, o := range res.Observations {
		paras[i] = paragraph(rt, names, o)
	}
	base := baseline.New(cfg.tenant, paras, docs, gw)
	base.TopK = cfg.topK

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
	return &setup{
		eais: eais, base: base, principals: principals, cases: cases,
		counter: evals.NewG7Counter(evals.G7CounterConfig{
			Canaries: canaries, Records: ownRecords, Documents: ownDocs, QuantityUnits: quantityUnits,
		}),
		canaries: canaries, paragraphs: len(paras), documents: len(docs),
	}, nil
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
