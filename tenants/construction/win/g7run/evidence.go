package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/luannt2002/enterprise-ai-systems/core/contracts"
	"github.com/luannt2002/enterprise-ai-systems/evals"
	"github.com/luannt2002/enterprise-ai-systems/intelligence/models"
)

// Bằng chứng thô của một lần chạy G7 (ADR-0005 §7, -out): không có bằng chứng thô thì không có kết
// luận G7. Mọi đường dẫn trong bằng chứng là tương đối hoặc chỉ tên tệp; không ghi tên máy, người dùng,
// khoá. Khoá nằm ở header HTTP, không bao giờ ở thân yêu cầu.

// priceTable là giá USD / triệu token (vào, ra), tra trang Pricing chính thức ngày 20/10/2026
// (docs/g7/CHI-PHI-G7.md mục 5). Model không có trong bảng thì chi phí ghi "CHƯA CÓ GIÁ".
var priceTable = map[string][2]float64{
	"claude-sonnet-5-5": {2, 10},
	"claude-haiku-5-5":  {0.10, 0.50},
	"claude-opus-5-5":   {4, 20},
}

// modelCall là một lần gửi HTTP tới model (models.MessagesTrace).
type modelCall struct {
	Attempt      int             `json:"attempt"`
	Status       int             `json:"status"`
	LatencyMS    float64         `json:"latency_ms"`
	Request      json.RawMessage `json:"request,omitempty"`
	Response     json.RawMessage `json:"response,omitempty"`
	RawResponse  string          `json:"raw_response,omitempty"` // phản hồi không phải JSON
	Error        string          `json:"error,omitempty"`
	InputTokens  int             `json:"input_tokens"`
	OutputTokens int             `json:"output_tokens"`
	CacheRead    int             `json:"cache_read_input_tokens"`
	CacheWrite   int             `json:"cache_creation_input_tokens"`
}

// askRecord là bằng chứng thô của MỘT câu hỏi trong MỘT lượt của MỘT hệ.
type askRecord struct {
	System     string      `json:"system"`
	Run        int         `json:"run"`
	CaseID     string      `json:"case_id"`
	Principal  string      `json:"principal"`
	Question   string      `json:"question"`
	Route      string      `json:"route"`
	Outcome    string      `json:"outcome"`
	Answer     string      `json:"answer"`
	Candidates []string    `json:"candidates,omitempty"`
	Conflicts  []string    `json:"conflicts,omitempty"` // khoá xung đột; "*" = chờ người duyệt
	Citations  []string    `json:"citations,omitempty"` // id bằng chứng được trích
	ModelCalls int         `json:"model_calls"`
	LatencyMS  float64     `json:"latency_ms"`
	Error      string      `json:"error,omitempty"`
	Calls      []modelCall `json:"calls,omitempty"`
}

// path là đường tất định của câu trả lời EACL (route, outcome, ứng viên, xung đột, trích dẫn): giữa các
// lượt nó phải giống nhau, khác là lượt vô hiệu (ADR-0005 §5).
func (r askRecord) path() string {
	return strings.Join([]string{r.Route, r.Outcome, strings.Join(r.Candidates, ","), strings.Join(r.Conflicts, ","), strings.Join(r.Citations, ",")}, " | ")
}

// evidenceSink gom bằng chứng thô. Các hệ chạy TUẦN TỰ, nên vết lời gọi model (đến qua
// models.Messages.Trace) thuộc về đúng câu hỏi đang chạy.
type evidenceSink struct {
	mu     sync.Mutex
	cur    *askRecord
	recs   []askRecord
	caseOf map[string]string // askKey → id ca
}

func newEvidenceSink(cases []evals.WinCase) *evidenceSink {
	s := &evidenceSink{caseOf: map[string]string{}}
	for _, c := range cases {
		k := askKey(c.Principal, c.Question)
		if _, dup := s.caseOf[k]; dup {
			s.caseOf[k] = "?" // hai ca cùng người hỏi + câu hỏi: không gán được, ghi rõ
			continue
		}
		s.caseOf[k] = c.ID
	}
	return s
}

// trace là models.Messages.Trace.
func (s *evidenceSink) trace(tr models.MessagesTrace) {
	c := modelCall{Attempt: tr.Attempt, Status: tr.Status, LatencyMS: ms(tr.Latency), Error: tr.Err}
	if json.Valid(tr.Request) {
		c.Request = append(json.RawMessage(nil), tr.Request...)
	}
	if json.Valid(tr.Response) {
		c.Response = append(json.RawMessage(nil), tr.Response...)
		var u struct {
			Usage struct {
				In  int `json:"input_tokens"`
				Out int `json:"output_tokens"`
				CR  int `json:"cache_read_input_tokens"`
				CW  int `json:"cache_creation_input_tokens"`
			} `json:"usage"`
		}
		if json.Unmarshal(tr.Response, &u) == nil {
			c.InputTokens, c.OutputTokens, c.CacheRead, c.CacheWrite = u.Usage.In, u.Usage.Out, u.Usage.CR, u.Usage.CW
		}
	} else if len(tr.Response) > 0 {
		c.RawResponse = string(tr.Response)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.cur != nil {
		s.cur.Calls = append(s.cur.Calls, c)
	}
}

func ms(d time.Duration) float64 { return float64(d.Microseconds()) / 1000 }

// recorder bọc Asker của một hệ để ghi bằng chứng thô; không đổi hành vi của hệ được bọc.
type recorder struct {
	inner evals.Asker
	mc    evals.ModelCounter
	sys   string
	run   int
	sink  *evidenceSink
}

func newRecorder(a evals.Asker, sys string, run int, sink *evidenceSink) (*recorder, error) {
	mc, ok := a.(evals.ModelCounter)
	if !ok {
		return nil, fmt.Errorf("hệ %s không đếm được lời gọi model", sys)
	}
	return &recorder{inner: a, mc: mc, sys: sys, run: run, sink: sink}, nil
}

func (r *recorder) Ask(ctx context.Context, pr contracts.Principal, q string) (contracts.AskResponse, error) {
	rec := &askRecord{System: r.sys, Run: r.run, CaseID: r.sink.caseOf[askKey(pr.ID, q)], Principal: pr.ID, Question: q}
	r.sink.mu.Lock()
	r.sink.cur = rec
	r.sink.mu.Unlock()
	start := time.Now()
	resp, err := r.inner.Ask(ctx, pr, q)
	rec.LatencyMS = ms(time.Since(start))
	rec.Route, rec.Outcome, rec.Answer = string(resp.Route), string(resp.Outcome), resp.Answer
	if resp.Clarification != nil {
		for _, c := range resp.Clarification.Candidates {
			rec.Candidates = append(rec.Candidates, c.ID)
		}
		sort.Strings(rec.Candidates)
	}
	for _, c := range resp.Conflicts {
		k := c.Key
		if c.NeedsReview {
			k += "*"
		}
		rec.Conflicts = append(rec.Conflicts, k)
	}
	sort.Strings(rec.Conflicts)
	for _, c := range resp.Citations {
		rec.Citations = append(rec.Citations, c.EvidenceID)
	}
	sort.Strings(rec.Citations)
	rec.ModelCalls = r.mc.ModelCalls() // RunWin đặt lại bộ đếm trước mỗi câu hỏi
	if err != nil {
		rec.Error = err.Error()
	}
	r.sink.mu.Lock()
	r.sink.cur = nil
	r.sink.recs = append(r.sink.recs, *rec)
	r.sink.mu.Unlock()
	return resp, err
}

func (r *recorder) ModelCalls() int  { return r.mc.ModelCalls() }
func (r *recorder) ResetModelCalls() { r.mc.ResetModelCalls() }

// unstablePaths trả các ca mà đường tất định của EACL khác nhau giữa các lượt (ADR-0005 §5).
func unstablePaths(recs []askRecord) []string {
	first := map[string]string{}
	bad := map[string]bool{}
	for _, r := range recs {
		if r.System != sysEACL {
			continue
		}
		if p, ok := first[r.CaseID]; !ok {
			first[r.CaseID] = r.path()
		} else if p != r.path() {
			bad[r.CaseID] = true
		}
	}
	var out []string
	for id := range bad {
		out = append(out, id)
	}
	sort.Strings(out)
	return out
}

// passCounts đếm số lượt đạt của từng ca, mỗi hệ (ADR-0005 §5: điểm từng ca là pass_count / k).
func passCounts(graded map[string][]evals.Report) map[string]map[string]int {
	out := map[string]map[string]int{}
	for sys, runs := range graded {
		out[sys] = map[string]int{}
		for _, r := range runs {
			for _, x := range r.Results {
				if _, ok := out[sys][x.ID]; !ok {
					out[sys][x.ID] = 0
				}
				if x.Pass {
					out[sys][x.ID]++
				}
			}
		}
	}
	return out
}

// flakyCases trả, cho từng hệ, các ca CHẬP CHỜN: đạt ở một số lượt nhưng không phải mọi lượt
// (0 < pass_count < k). Đã sắp theo id.
func flakyCases(counts map[string]map[string]int, k int) map[string][]string {
	out := map[string][]string{}
	for sys, byCase := range counts {
		for id, n := range byCase {
			if n > 0 && n < k {
				out[sys] = append(out[sys], id)
			}
		}
		sort.Strings(out[sys])
	}
	return out
}

// pilotK là luật chọn k đăng ký trước (ADR-0006 §6): pilot chạy k = 5; hệ cổng nào có ≥ 2 ca chập
// chờn thì lượt chính k = 5, không thì k = 3.
func pilotK(flaky map[string][]string) int {
	for _, g := range gateSystems {
		if len(flaky[g]) >= 2 {
			return 5
		}
	}
	return 3
}

// costRow là token + tiền của một hệ.
type costRow struct {
	System       string  `json:"system"`
	HTTPCalls    int     `json:"http_calls"`
	InputTokens  int     `json:"input_tokens"`
	OutputTokens int     `json:"output_tokens"`
	CacheRead    int     `json:"cache_read_input_tokens"`
	CacheWrite   int     `json:"cache_creation_input_tokens"`
	USD          float64 `json:"usd"`
	Priced       bool    `json:"priced"` // false = model không có trong bảng giá: USD = 0 không có nghĩa
}

func costs(recs []askRecord, model string, systems []string) []costRow {
	price, priced := priceTable[model]
	rows := map[string]*costRow{}
	for _, s := range systems {
		rows[s] = &costRow{System: s, Priced: priced}
	}
	for _, r := range recs {
		c := rows[r.System]
		if c == nil {
			continue
		}
		for _, m := range r.Calls {
			c.HTTPCalls++
			c.InputTokens += m.InputTokens
			c.OutputTokens += m.OutputTokens
			c.CacheRead += m.CacheRead
			c.CacheWrite += m.CacheWrite
		}
	}
	var out []costRow
	for _, s := range systems {
		c := rows[s]
		if priced {
			c.USD = (float64(c.InputTokens)*price[0] + float64(c.OutputTokens)*price[1]) / 1e6
		}
		out = append(out, *c)
	}
	return out
}

// gitState trả revision và cờ "cây bẩn" của repo nơi lệnh chạy; không có git thì "unknown".
func gitState() (rev string, dirty bool) {
	out, err := exec.Command("git", "rev-parse", "HEAD").Output()
	if err != nil {
		return "unknown", false
	}
	st, _ := exec.Command("git", "status", "--porcelain").Output()
	return strings.TrimSpace(string(out)), len(strings.TrimSpace(string(st))) > 0
}

func fileSHA256(path string) string {
	b, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}

func textSHA256(s string) string {
	sum := sha256.Sum256([]byte(s))
	return hex.EncodeToString(sum[:])
}

// manifestInfo là manifest.json của bằng chứng.
type manifestInfo struct {
	Schema         int                   `json:"schema"`
	Tool           string                `json:"tool"`
	StartedAt      string                `json:"started_at"`
	FinishedAt     string                `json:"finished_at"`
	GitRevision    string                `json:"git_revision"`
	GitDirty       bool                  `json:"git_dirty"`
	GoVersion      string                `json:"go_version"`
	OS             string                `json:"os"`
	NumCPU         int                   `json:"num_cpu"`
	FakeModel      bool                  `json:"fake_model"` // true = model giả tất định: KHÔNG phải bằng chứng G7
	Provider       string                `json:"provider"`
	Model          string                `json:"model"`
	ModelParams    map[string]any        `json:"model_params"`
	SystemPromptSH string                `json:"system_prompt_sha256"`
	K              int                   `json:"k"`
	TopK           int                   `json:"topk"`
	Systems        []string              `json:"systems"`
	GateSystems    []string              `json:"gate_systems"`
	Blocked        []string              `json:"blocked,omitempty"` // hệ cổng không chạy được và lý do
	Cases          int                   `json:"cases"`
	Files          map[string]string     `json:"files_sha256"` // đường dẫn tương đối → sha256
	DataManifest   string                `json:"data_manifest_sha256"`
	PriceTable     map[string][2]float64 `json:"price_usd_per_mtok"`
	BudgetUSD      float64               `json:"budget_usd"`
	Unstable       []string              `json:"eacl_unstable_cases,omitempty"`
}

type scoredRow struct {
	System       string   `json:"system"`
	Run          int      `json:"run"`
	CaseID       string   `json:"case_id"`
	Class        string   `json:"class"`
	Pass         bool     `json:"pass"`
	Problems     []string `json:"problems,omitempty"`
	FullPass     bool     `json:"full_pass"`
	FullProblems []string `json:"full_problems,omitempty"`
}

type failureRow struct {
	System          string   `json:"system"`
	Run             int      `json:"run"`
	CaseID          string   `json:"case_id"`
	Class           string   `json:"class"`
	Problems        []string `json:"problems"`
	FirstWrongLayer string   `json:"first_wrong_layer"` // điền khi phân tích; máy không tự quyết
}

// writeEvidence ghi bộ bằng chứng của một lần chạy vào dir.
func writeEvidence(dir string, m manifestInfo, recs []askRecord, scored []scoredRow, aggregate any, failures []failureRow, cost []costRow, summary string) error {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	writeJSON := func(name string, v any) error {
		b, err := json.MarshalIndent(v, "", "  ")
		if err != nil {
			return err
		}
		return os.WriteFile(filepath.Join(dir, name), append(b, '\n'), 0o644)
	}
	writeJSONL := func(name string, n int, at func(i int) any) error {
		var sb strings.Builder
		for i := 0; i < n; i++ {
			b, err := json.Marshal(at(i))
			if err != nil {
				return err
			}
			sb.Write(b)
			sb.WriteByte('\n')
		}
		return os.WriteFile(filepath.Join(dir, name), []byte(sb.String()), 0o644)
	}
	for _, step := range []func() error{
		func() error { return writeJSON("manifest.json", m) },
		func() error { return writeJSONL("raw-runs.jsonl", len(recs), func(i int) any { return recs[i] }) },
		func() error { return writeJSONL("scored.jsonl", len(scored), func(i int) any { return scored[i] }) },
		func() error { return writeJSON("aggregate.json", aggregate) },
		func() error {
			return writeJSONL("failures.jsonl", len(failures), func(i int) any { return failures[i] })
		},
		func() error { return writeJSON("cost.json", cost) },
		func() error { return os.WriteFile(filepath.Join(dir, "ket-qua.md"), []byte(summary), 0o644) },
	} {
		if err := step(); err != nil {
			return err
		}
	}
	return nil
}

// newManifest điền phần tĩnh của manifest. Tham số model lấy từ chính gateway (giá trị hiệu lực, kể cả
// mặc định do models.New điền), không từ cấu hình thô.
func newManifest(cfg config, mc models.Config, gw models.Gateway, fake bool, systems []string, cases int, files []string) manifestInfo {
	rev, dirty := gitState()
	m := manifestInfo{
		Schema: 1, Tool: "g7run", GitRevision: rev, GitDirty: dirty, GoVersion: runtime.Version(),
		OS: runtime.GOOS + "/" + runtime.GOARCH, NumCPU: runtime.NumCPU(), FakeModel: fake,
		Provider: mc.Provider, Model: mc.Model, SystemPromptSH: textSHA256(models.SystemPrompt()),
		K: cfg.k, TopK: cfg.topK, Systems: systems, GateSystems: gateSystems, Cases: cases,
		Files: map[string]string{}, PriceTable: priceTable, BudgetUSD: 15,
		ModelParams: map[string]any{},
	}
	if g, ok := gw.(*models.Messages); ok {
		m.ModelParams = map[string]any{
			"max_tokens": g.MaxTokens, "temperature": g.Temperature, "thinking": g.Thinking,
			"effort": g.Effort, "max_retries": g.MaxRetries,
		}
	}
	for _, f := range files {
		if f != "" {
			m.Files[filepath.ToSlash(f)] = fileSHA256(f)
		}
	}
	m.DataManifest = fileSHA256(filepath.Join(cfg.data, "manifest.json"))
	return m
}
