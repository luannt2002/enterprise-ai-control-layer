// Package winrun đo WIN MVP trên một bộ dữ liệu do bộ sinh xuất ra (docs/04-core-mvp.md §6):
//   - G1: so manifest của bộ sinh với kết quả bộ nạp, từng khoá (quan sát hay cách ly, đúng giá trị, đúng cờ ước lượng);
//   - bộ ca WIN: chạy từng ca qua luồng hỏi thật, chấm bằng kỳ vọng nhị phân.
//
// Dùng chung cho lệnh winreport (in báo cáo) và test ratchet (chặn hồi quy trong `go test`).
package winrun

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/luannt2002/enterprise-ai-control-layer/core/contracts"
	"github.com/luannt2002/enterprise-ai-control-layer/core/decimal"
	"github.com/luannt2002/enterprise-ai-control-layer/core/pipeline"
	"github.com/luannt2002/enterprise-ai-control-layer/core/tenant"
	"github.com/luannt2002/enterprise-ai-control-layer/data/ingestion"
	"github.com/luannt2002/enterprise-ai-control-layer/evals"
	"github.com/luannt2002/enterprise-ai-control-layer/intelligence/models"
	"github.com/luannt2002/enterprise-ai-control-layer/intelligence/rag"
	"github.com/luannt2002/enterprise-ai-control-layer/observability/traces"
	"github.com/luannt2002/enterprise-ai-control-layer/platform/pgstore"
	"go.yaml.in/yaml/v3"
)

// Config là đầu vào của một lượt đo.
type Config struct {
	Data, Pack, World, CasesGlob, Tenant string
	// PG: có thì dữ liệu đã nạp được ghi vào PostgreSQL (thay trọn dữ liệu của tenant) và luồng hỏi
	// đọc danh mục, fact, tài liệu QUA PG (RLS theo tenant của Principal); nil = đọc trong bộ nhớ.
	PG *pgstore.Store
	// Canary: nạp thêm dữ liệu của MỌI tenant khác trong world (cùng thư mục Data) vào CÙNG kho với Tenant, hỏi
	// dưới Principal của Tenant, rồi quét câu trả lời và trích dẫn tìm canary của tenant khác (canary.go).
	// Tắt (mặc định) = đường cũ, không đổi gì. Chưa hỗ trợ cùng PG.
	Canary bool
	// wrapStore (chỉ test trong gói): thay kho chung bằng kho cố ý rò, làm đối chứng dương của canary.
	wrapStore func(*sharedStore) coStore
}

// Result là kết quả một lượt đo.
type Result struct {
	Observations, Quarantined int
	IngestTime                time.Duration
	G1Total                   int
	G1Problems                []string
	Cases                     []evals.WinCase
	Report                    evals.Report
	// CaseHashes: sha256 của từng tệp ca (G6: bộ ca đóng băng, băm ghi vào báo cáo).
	CaseHashes map[string]string
	// DataVersion: băm manifest của bộ dữ liệu đã nạp — ghi vào mọi bản ghi quyết định.
	DataVersion string
	// Decisions: số câu đã giao có bản ghi quyết định; ReplayProblems: ca phát lại ra khác bản ghi.
	Decisions      int
	ReplayProblems []string
	// G1 metamorphic: số biểu diễn đã nạp riêng từng cái, theo bản ghi; biểu diễn ra khác kỳ vọng.
	Variants        map[string]int
	VariantProblems []string
	// Canary: kết quả canary 2 tenant; nil khi Config.Canary tắt.
	Canary *CanaryResult
}

type manifest struct {
	Records []struct {
		Tenant   string `json:"tenant"`
		Key      string `json:"key"`
		Expect   string `json:"expect"`
		Entity   string `json:"entity"`
		Semantic string `json:"semantic"`
		Source   string `json:"source"`
		On       string `json:"on"`
		ValueL   string `json:"value_l"`
		Reason   string `json:"reason"`
		Approx   bool   `json:"approx"` // người nhập ghi "khoảng", "≈", "2k": quan sát phải mang cờ ước lượng (K1)
	} `json:"records"`
}

// G1ApproxMark là cụm nhận ra vấn đề G1 do lệch cờ ước lượng (không phải lệch thực thể, chỉ số, nguồn,
// ngày hay số). Test ratchet dùng nó để tách các lệch cờ đã biết khỏi mọi lệch khác.
const G1ApproxMark = "cờ ước lượng"

type world struct {
	ReferenceDate string `yaml:"reference_date"`
	Tenants       []struct {
		ID         string `yaml:"id"`
		Principals []struct {
			ID          string `yaml:"id"`
			Role        string `yaml:"role"`
			EntityScope string `yaml:"entity_scope"`
		} `yaml:"principals"`
	} `yaml:"tenants"`
}

type pipelineAsker struct {
	a         *pipeline.Ask
	model     *models.Extractive
	decisions []traces.Decision
	// scan (nil = không quét): gọi cho MỌI lần hỏi, kể cả lần lỗi — canary ghép lần hỏi với ca theo thứ tự.
	scan func(question string, pr contracts.Principal, r contracts.AskResponse)
}

func (p *pipelineAsker) ModelCalls() int  { return p.model.Calls() }
func (p *pipelineAsker) ResetModelCalls() { p.model.Reset() }

func (p *pipelineAsker) Ask(ctx context.Context, pr contracts.Principal, q string) (contracts.AskResponse, error) {
	out, err := p.a.Handle(ctx, pr, contracts.AskRequest{ConversationID: "win", MessageID: "win", Text: q})
	if err == nil {
		p.decisions = append(p.decisions, out.Decision)
	}
	if p.scan != nil {
		p.scan(q, pr, out.Response)
	}
	return out.Response, err
}

// Run nạp dữ liệu của tenant, đo G1 và chạy bộ ca WIN.
func Run(cfg Config) (Result, error) {
	var out Result
	ctx := context.Background()
	rt, err := tenant.Load(cfg.Pack)
	if err != nil {
		return out, err
	}
	sm, err := ingestion.LoadSources(filepath.Join(cfg.Pack, "sources.yaml"), rt)
	if err != nil {
		return out, err
	}
	dir := filepath.Join(cfg.Data, cfg.Tenant)
	t0 := time.Now()
	res, err := ingestion.Ingest(ctx, dir, rt, sm, ingestion.PdfToText{})
	if err != nil {
		return out, err
	}
	out.Observations, out.Quarantined, out.IngestTime = len(res.Observations), len(res.Quarantined), time.Since(t0)
	if out.G1Total, out.G1Problems, err = checkG1(filepath.Join(cfg.Data, "manifest.json"), cfg.Tenant, res); err != nil {
		return out, err
	}
	if out.DataVersion, err = fileHash(filepath.Join(cfg.Data, "manifest.json")); err != nil {
		return out, err
	}
	if out.Variants, out.VariantProblems, err = checkVariants(ctx, filepath.Join(cfg.Data, "variants"), rt, sm); err != nil {
		return out, err
	}

	var w world
	b, err := os.ReadFile(cfg.World)
	if err != nil {
		return out, err
	}
	if err := yaml.Unmarshal(b, &w); err != nil {
		return out, err
	}
	ref, err := time.ParseInLocation("2006-01-02", w.ReferenceDate, rt.Location)
	if err != nil {
		return out, err
	}
	principals := map[string]contracts.Principal{}
	for _, tn := range w.Tenants {
		if tn.ID != cfg.Tenant {
			continue
		}
		for _, p := range tn.Principals {
			pr := contracts.Principal{ID: p.ID, TenantID: contracts.TenantID(cfg.Tenant), Kind: contracts.PrincipalUser, Roles: []string{p.Role}}
			if p.EntityScope != "" {
				pr.Attributes = map[string]string{"entity_scope": p.EntityScope}
			}
			principals[p.ID] = pr
		}
	}
	directory, err := ingestion.LoadDirectory(dir, sm)
	if err != nil {
		return out, err
	}
	docs, err := rag.LoadMarkdown(filepath.Join(dir, "docs"), cfg.Tenant)
	if err != nil {
		return out, err
	}
	model := &models.Extractive{} // WIN đo bằng model tất định; model thật là việc của G7 (W6)
	now := ref.Add(10 * time.Hour)
	deps := pipeline.Deps{
		Directory: directory, Facts: res.Facts(), Knowledge: rag.NewMemoryIndex(cfg.Tenant, docs), Model: model,
		DataVersion: out.DataVersion, Now: func() time.Time { return now },
	}
	if cfg.PG != nil {
		if err := cfg.PG.Load(contracts.WithTenant(ctx, contracts.TenantID(cfg.Tenant)), directory, res.Facts(), docs); err != nil {
			return out, err
		}
		deps.Directory, deps.Facts, deps.Knowledge = cfg.PG, cfg.PG, cfg.PG
	}
	var co *coRun
	if cfg.Canary {
		if cfg.PG != nil {
			return out, fmt.Errorf("canary chưa hỗ trợ đường PostgreSQL: chạy đường bộ nhớ")
		}
		if co, err = newCoRun(ctx, cfg, worldTenantIDs(w), rt, sm, tenantData{id: cfg.Tenant, res: res, dir: directory, docs: docs}, &deps); err != nil {
			return out, err
		}
	}
	ask := &pipeline.Ask{RT: rt, Deps: deps}

	files, _ := filepath.Glob(cfg.CasesGlob)
	sort.Strings(files)
	out.CaseHashes = map[string]string{}
	for _, f := range files {
		s, err := evals.LoadWinSuite(f)
		if err != nil {
			return out, err
		}
		out.Cases = append(out.Cases, s.Cases...)
		if out.CaseHashes[filepath.Base(f)], err = fileHash(f); err != nil {
			return out, err
		}
	}
	asker := &pipelineAsker{a: ask, model: model}
	if co != nil {
		asker.scan = co.scan
	}
	out.Report = evals.RunWin(ctx, asker, out.Cases, principals)
	if co != nil {
		if out.Canary, err = co.result(out.Cases, principals); err != nil {
			return out, err
		}
	}
	// G6 — phát lại: mọi câu đã giao chạy lại từ bản ghi quyết định phải ra đúng từng chặng.
	out.Decisions = len(asker.decisions)
	for _, d := range asker.decisions {
		diff, err := ask.Replay(ctx, d)
		switch {
		case err != nil:
			out.ReplayProblems = append(out.ReplayProblems, fmt.Sprintf("%q: lỗi phát lại %v", d.Question, err))
		case len(diff) > 0:
			out.ReplayProblems = append(out.ReplayProblems, fmt.Sprintf("%q: lệch ở %v", d.Question, diff))
		}
	}
	return out, nil
}

// checkVariants là test metamorphic của G1: mỗi biểu diễn của CÙNG một bản ghi (CSV / XLSX / JSON,
// tin nhắn viết nhiều kiểu) nằm ở thư mục riêng; nạp từng thư mục phải ra đúng MỘT quan sát của
// bản ghi đó, và phép chiếu (thực thể, nghĩa, nguồn, ngày, giá trị ở đơn vị chuẩn) phải bằng kỳ vọng
// của bộ sinh — không phải bằng một biểu diễn khác.
func checkVariants(ctx context.Context, dir string, rt *tenant.Runtime, sm ingestion.SourceMap) (map[string]int, []string, error) {
	b, err := os.ReadFile(filepath.Join(dir, "manifest.json"))
	if err != nil {
		return nil, nil, err
	}
	var vs []struct {
		WorldID string   `json:"world_id"`
		Source  string   `json:"source"`
		Files   []string `json:"files"`
		Expect  struct {
			Entity, Semantic, Source, On string
			ValueL                       string `json:"value_l"`
		} `json:"expect"`
	}
	if err := json.Unmarshal(b, &vs); err != nil {
		return nil, nil, err
	}
	counts := map[string]int{}
	var problems []string
	for _, v := range vs {
		want := fmt.Sprintf("%s|%s|%s|%s|%s L", v.Expect.Entity, v.Expect.Semantic, v.Expect.Source, v.Expect.On, v.Expect.ValueL)
		for _, f := range v.Files {
			res, err := ingestion.Ingest(ctx, filepath.Join(dir, filepath.Dir(f)), rt, sm, ingestion.PdfToText{})
			if err != nil {
				return nil, nil, err
			}
			var got []string
			for _, o := range res.Observations {
				if strings.HasSuffix(o.Key, "#"+v.WorldID) {
					got = append(got, fmt.Sprintf("%s|%s|%s|%s|%s %s", o.Entity, o.Metric, o.Source, o.On.Format(time.DateOnly), o.Value.Number, o.Value.Unit))
				}
			}
			counts[v.WorldID]++
			if len(got) != 1 || got[0] != want {
				problems = append(problems, fmt.Sprintf("%s (%s): được %v, muốn [%s]", v.WorldID, f, got, want))
			}
		}
	}
	return counts, problems, nil
}

// fileHash trả sha256 (hex) của một tệp.
func fileHash(path string) (string, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:]), nil
}

// checkG1 so manifest (đáp án của bộ sinh) với kết quả bộ nạp theo từng khoá.
func checkG1(path, tenantID string, res ingestion.Result) (int, []string, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return 0, nil, err
	}
	var m manifest
	if err := json.Unmarshal(b, &m); err != nil {
		return 0, nil, err
	}
	obs := map[string]ingestion.Observation{}
	for _, o := range res.Observations {
		obs[o.Key] = o
	}
	quar := map[string]ingestion.Quarantine{}
	for _, q := range res.Quarantined {
		quar[q.Key] = q
	}
	var problems []string
	seen := map[string]bool{}
	total := 0
	for _, r := range m.Records {
		if r.Tenant != tenantID {
			continue
		}
		total++
		seen[r.Key] = true
		switch r.Expect {
		case "observation":
			o, ok := obs[r.Key]
			if !ok {
				why := "không có"
				if q, isQ := quar[r.Key]; isQ {
					why = "bị cách ly: " + q.Reason + " — " + q.Detail
				}
				problems = append(problems, fmt.Sprintf("%s: muốn quan sát, %s", r.Key, why))
				continue
			}
			switch {
			case o.Entity != r.Entity || o.Metric != r.Semantic || o.Source != r.Source || o.On.Format("2006-01-02") != r.On || !decimal.Equal(o.Value.Number, r.ValueL):
				problems = append(problems, fmt.Sprintf("%s: được %s/%s/%s/%s/%s, muốn %s/%s/%s/%s/%s", r.Key,
					o.Entity, o.Metric, o.Source, o.On.Format("2006-01-02"), o.Value.Number, r.Entity, r.Semantic, r.Source, r.On, r.ValueL))
			case o.Value.Approx != r.Approx:
				// mỗi bản ghi lệch chỉ ra MỘT vấn đề (báo cáo trừ "N/M khớp" theo số vấn đề): lệch trường chính
				// báo trước, cờ ước lượng chỉ báo khi các trường chính đã khớp.
				problems = append(problems, fmt.Sprintf("%s: %s được %v, muốn %v", r.Key, G1ApproxMark, o.Value.Approx, r.Approx))
			}
		case "quarantine":
			q, ok := quar[r.Key]
			if !ok || q.Reason != r.Reason {
				problems = append(problems, fmt.Sprintf("%s: muốn cách ly %s, được %+v (là quan sát: %v)", r.Key, r.Reason, q, obs[r.Key].Key != ""))
			}
		}
	}
	for k := range obs {
		if !seen[k] {
			problems = append(problems, k+": quan sát THỪA, manifest không có")
		}
	}
	for k := range quar {
		if !seen[k] {
			problems = append(problems, k+": cách ly THỪA, manifest không có")
		}
	}
	sort.Strings(problems)
	return total, problems, nil
}
