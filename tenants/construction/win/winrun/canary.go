package winrun

// Canary 2 tenant (phản hồi audit K6, docs/bao-cao/2026-10-21-phan-hoi-roadmap.md): harness cũ chỉ nạp tenant
// đang chạy (Join(Data, Tenant)), nên "0 canary" ở H1 / H3 đúng theo cấu trúc: dữ liệu tenant B không bao giờ
// nằm trong kho, chưa chứng minh được cách ly. Chế độ này nạp cả B vào CÙNG kho với A, hỏi các câu của A dưới
// Principal của A, rồi quét câu trả lời và trích dẫn tìm chuỗi chỉ có ở B. Tắt (mặc định) = đường cũ, không đổi.
//
// Kiểm tra này chỉ có nghĩa khi có ĐỐI CHỨNG DƯƠNG: canary_internal_test.go cho kho cố ý rò dữ liệu B sang A
// (bỏ lọc tenant) và đòi canary đỏ. Đường PG (RLS) không đi qua chế độ này: chưa hỗ trợ cùng cfg.PG.

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"github.com/luannt2002/enterprise-ai-control-layer/core/contracts"
	"github.com/luannt2002/enterprise-ai-control-layer/core/pipeline"
	"github.com/luannt2002/enterprise-ai-control-layer/core/tenant"
	"github.com/luannt2002/enterprise-ai-control-layer/data/entityresolution"
	"github.com/luannt2002/enterprise-ai-control-layer/data/ingestion"
	"github.com/luannt2002/enterprise-ai-control-layer/data/normalization"
	"github.com/luannt2002/enterprise-ai-control-layer/data/sourceoftruth"
	"github.com/luannt2002/enterprise-ai-control-layer/evals"
	"github.com/luannt2002/enterprise-ai-control-layer/intelligence/rag"
)

// CoTenant là một tenant nằm trong kho chung: bằng chứng rằng dữ liệu của nó THẬT SỰ đã được nạp.
type CoTenant struct {
	ID                                string
	Observations, Entities, Documents int
}

// CanaryResult là kết quả quét canary của một lượt chạy.
type CanaryResult struct {
	Tenants  []CoTenant // tenant được chấm đứng đầu
	Canaries []string   // chữ chỉ có ở dữ liệu của tenant khác (rỗng thì lượt chạy đã báo lỗi, không tới đây)
	Answers  int        // số câu đã quét
	Leaks    []string   // "<ca> (<principal>): rò <canary>…"; rỗng = sạch
}

// coStore là ba cổng đọc của luồng hỏi trên một kho chứa nhiều tenant.
type coStore interface {
	entityresolution.Directory
	sourceoftruth.Store
	pipeline.KnowledgeRetriever
}

// tenantData là dữ liệu đã nạp của một tenant.
type tenantData struct {
	id   string
	res  ingestion.Result
	dir  entityresolution.MemoryDirectory
	docs []rag.Document
}

func loadTenantData(ctx context.Context, root, id string, rt *tenant.Runtime, sm ingestion.SourceMap) (tenantData, error) {
	dir := filepath.Join(root, id)
	res, err := ingestion.Ingest(ctx, dir, rt, sm, ingestion.PdfToText{})
	if err != nil {
		return tenantData{}, fmt.Errorf("canary: nạp %s: %w", id, err)
	}
	entities, err := ingestion.LoadDirectory(dir, sm)
	if err != nil {
		return tenantData{}, fmt.Errorf("canary: danh mục %s: %w", id, err)
	}
	docs, err := rag.LoadMarkdown(filepath.Join(dir, "docs"), id)
	if err != nil {
		return tenantData{}, fmt.Errorf("canary: tài liệu %s: %w", id, err)
	}
	return tenantData{id: id, res: res, dir: entities, docs: docs}, nil
}

// texts là TOÀN BỘ chữ của tenant: mọi thứ tenant đó có thể nói đúng (khoá, nguyên văn, ngày, số, danh mục,
// tài liệu, lý do cách ly). DeriveCanaries cần đủ chữ của tenant được chấm, thiếu thì mã dùng chung bị nhận
// nhầm là canary.
func (d tenantData) texts() []string {
	var out []string
	for _, o := range d.res.Observations {
		out = append(out, o.Key, o.Entity, o.Metric, o.Source, o.Raw, o.Note, o.On.Format(time.DateOnly), o.Value.Number, o.Value.Unit)
	}
	for _, q := range d.res.Quarantined {
		out = append(out, q.Key, q.Reason, q.Detail)
	}
	for _, e := range d.dir {
		out = append(out, e.ID, e.Key, e.Name, e.Parent)
		for _, a := range e.Aliases {
			out = append(out, a.Text)
		}
	}
	for _, doc := range d.docs {
		out = append(out, doc.ID, doc.Title, doc.Body)
	}
	return out
}

// sharedStore là kho trong bộ nhớ chứa dữ liệu của nhiều tenant cạnh nhau, như bảng có cột tenant_id của PG.
// Mỗi lần đọc chỉ thấy phần của tenant trong ngữ cảnh (do S1 gắn từ Principal, không từ thân yêu cầu): cùng
// luật với RLS. Chỉ mục tri thức dựng bằng rag.NewMemoryIndex trên tài liệu của MỌI tenant, nên việc loại
// tài liệu tenant khác là của mã thật, không phải của harness.
type sharedStore struct {
	facts map[contracts.TenantID]sourceoftruth.MemoryStore
	dirs  map[contracts.TenantID]entityresolution.MemoryDirectory
	docs  []rag.Document

	mu  sync.Mutex
	idx map[contracts.TenantID]*rag.MemoryIndex
}

func newSharedStore(all []tenantData) *sharedStore {
	s := &sharedStore{
		facts: map[contracts.TenantID]sourceoftruth.MemoryStore{},
		dirs:  map[contracts.TenantID]entityresolution.MemoryDirectory{},
		idx:   map[contracts.TenantID]*rag.MemoryIndex{},
	}
	for _, d := range all {
		id := contracts.TenantID(d.id)
		s.facts[id], s.dirs[id] = d.res.Facts(), d.dir
		s.docs = append(s.docs, d.docs...)
	}
	return s
}

func tenantOf(ctx context.Context) (contracts.TenantID, error) {
	t, ok := contracts.TenantFrom(ctx)
	if !ok {
		return "", contracts.Errorf(contracts.FaultPolicy, "kho chung: không có tenant trong ngữ cảnh")
	}
	return t, nil
}

// Facts trả fact của tenant trong ngữ cảnh.
func (s *sharedStore) Facts(ctx context.Context, metric string, entityIDs []string, from, to time.Time) ([]sourceoftruth.Fact, error) {
	t, err := tenantOf(ctx)
	if err != nil {
		return nil, err
	}
	return s.facts[t].Facts(ctx, metric, entityIDs, from, to)
}

// Entities trả danh mục của tenant trong ngữ cảnh.
func (s *sharedStore) Entities(ctx context.Context, types ...string) ([]entityresolution.Entity, error) {
	t, err := tenantOf(ctx)
	if err != nil {
		return nil, err
	}
	return s.dirs[t].Entities(ctx, types...)
}

// Retrieve truy xuất tài liệu qua chỉ mục của tenant của principal.
func (s *sharedStore) Retrieve(ctx context.Context, pr contracts.Principal, query string, k int) ([]contracts.Evidence, error) {
	s.mu.Lock()
	ix, ok := s.idx[pr.TenantID]
	if !ok {
		ix = rag.NewMemoryIndex(string(pr.TenantID), s.docs)
		s.idx[pr.TenantID] = ix
	}
	s.mu.Unlock()
	return ix.Retrieve(ctx, pr, query, k)
}

// surface gom mọi chữ người dùng thấy trong một câu trả lời: câu chữ, trích dẫn (bằng chứng), xung đột, câu
// hỏi lại. KHÔNG gồm trace_id (uuid ngẫu nhiên: chữ số trong đó có thể trùng một canary toàn số).
func surface(r contracts.AskResponse) string {
	parts := []string{r.Answer}
	for _, v := range []any{r.Citations, r.Conflicts, r.Clarification} {
		if b, err := json.Marshal(v); err == nil {
			parts = append(parts, string(b))
		}
	}
	return strings.Join(parts, "\n")
}

// nameCanaries trả canary TÊN: mã tenant, mã / tên / bí danh thực thể, mã tài liệu và khoá bản ghi của các tenant
// KHÁC, giữ nguyên chữ ("Cầu Z", "def-build", khoá "warehouse.csv#R-B01"); DeriveCanaries chỉ bắt token có chữ số.
// Bỏ chuỗi ngắn hơn 4 ký tự (khớp nhầm quá dễ) và chuỗi đã có trong chữ của tenant được chấm (ownText): "EX-03"
// dùng chung ở cả hai tenant nên nhắc tới không là rò rỉ.
func nameCanaries(ownText string, others []tenantData) []string {
	ownLower := strings.ToLower(normalization.NFC(ownText))
	seen := map[string]bool{}
	var out []string
	add := func(s string) {
		s = strings.TrimSpace(normalization.NFC(s))
		k := strings.ToLower(s)
		if utf8.RuneCountInString(s) < 4 || seen[k] || strings.Contains(ownLower, k) {
			return
		}
		seen[k] = true
		out = append(out, s)
	}
	for _, o := range others {
		add(o.id)
		for _, e := range o.dir {
			add(e.ID)
			add(e.Key)
			add(e.Name)
			for _, a := range e.Aliases {
				add(a.Text)
			}
		}
		for _, d := range o.docs {
			add(d.ID)
		}
		for _, ob := range o.res.Observations {
			add(ob.Key)
		}
	}
	return out
}

// deriveCanaries trả mọi chuỗi chỉ có ở dữ liệu của các tenant khác: token chữ-số (số riêng như 9999, mã như
// 7731) theo evals.DeriveCanaries cộng canary tên. Rỗng là lỗi: bộ đếm sẽ đếm 0 vì mù, không vì sạch.
func deriveCanaries(own tenantData, others []tenantData) ([]string, error) {
	ownTexts := own.texts()
	var otherTexts []string
	for _, o := range others {
		otherTexts = append(otherTexts, o.texts()...)
	}
	set := map[string]bool{}
	for _, c := range evals.DeriveCanaries(ownTexts, otherTexts) {
		set[c] = true
	}
	for _, c := range nameCanaries(strings.Join(ownTexts, "\n"), others) {
		set[c] = true
	}
	if len(set) == 0 {
		return nil, errors.New("canary: dữ liệu của tenant khác không có chữ nào riêng so với tenant được chấm: kiểm tra sẽ mù")
	}
	out := make([]string, 0, len(set))
	for c := range set {
		out = append(out, c)
	}
	sort.Strings(out)
	return out, nil
}

func worldTenantIDs(w world) []string {
	var ids []string
	for _, t := range w.Tenants {
		ids = append(ids, t.ID)
	}
	return ids
}

// coRun giữ trạng thái quét canary của một lượt chạy.
type coRun struct {
	tenants  []tenantData
	canaries []string
	counter  *evals.G7Counter
	recs     []coRec
}

// coRec là một lần hỏi: ai hỏi và canary nào lộ ra. Luồng đo gọi Ask đúng một lần cho mỗi ca có principal hợp lệ.
type coRec struct {
	principal string
	leaks     []string
}

// newCoRun nạp mọi tenant khác của world có thư mục trong cfg.Data vào cùng kho với tenant được chấm, rồi gắn
// kho đó làm danh bạ, nguồn fact và chỉ mục tri thức của luồng hỏi (deps).
func newCoRun(ctx context.Context, cfg Config, worldTenants []string, rt *tenant.Runtime, sm ingestion.SourceMap, own tenantData, deps *pipeline.Deps) (*coRun, error) {
	all := []tenantData{own}
	for _, id := range worldTenants {
		if id == cfg.Tenant {
			continue
		}
		if _, err := os.Stat(filepath.Join(cfg.Data, id)); err != nil {
			continue // tenant của world mà bộ sinh không xuất thư mục
		}
		td, err := loadTenantData(ctx, cfg.Data, id, rt, sm)
		if err != nil {
			return nil, err
		}
		all = append(all, td)
	}
	if len(all) < 2 {
		return nil, fmt.Errorf("canary: không có tenant nào khác %s có dữ liệu trong %s: kiểm tra sẽ mù", cfg.Tenant, cfg.Data)
	}
	canaries, err := deriveCanaries(own, all[1:])
	if err != nil {
		return nil, err
	}
	store := newSharedStore(all)
	var cs coStore = store
	if cfg.wrapStore != nil {
		cs = cfg.wrapStore(store)
	}
	deps.Directory, deps.Facts, deps.Knowledge = cs, cs, cs
	return &coRun{tenants: all, canaries: canaries, counter: evals.NewG7Counter(evals.G7CounterConfig{Canaries: canaries})}, nil
}

// scan quét một câu trả lời; gọi cho MỌI lần hỏi, cả lần lỗi (câu trả lời rỗng), để ghép được với ca.
func (c *coRun) scan(question string, pr contracts.Principal, r contracts.AskResponse) {
	c.recs = append(c.recs, coRec{principal: pr.ID, leaks: c.counter.Leaks(question, surface(r))})
}

// result ghép các lần quét với ca theo thứ tự (evals.RunWin hỏi đúng một lần cho mỗi ca có principal hợp lệ).
func (c *coRun) result(cases []evals.WinCase, principals map[string]contracts.Principal) (*CanaryResult, error) {
	asked := 0
	for _, cs := range cases {
		if _, ok := principals[cs.Principal]; ok {
			asked++
		}
	}
	if asked != len(c.recs) {
		return nil, fmt.Errorf("canary: %d lần hỏi khác %d ca hỏi được: không ghép được câu trả lời với ca", len(c.recs), asked)
	}
	out := &CanaryResult{Canaries: c.canaries, Answers: len(c.recs)}
	for _, t := range c.tenants {
		out.Tenants = append(out.Tenants, CoTenant{ID: t.id, Observations: len(t.res.Observations), Entities: len(t.dir), Documents: len(t.docs)})
	}
	j := 0
	for _, cs := range cases {
		if _, ok := principals[cs.Principal]; !ok {
			continue
		}
		if l := c.recs[j].leaks; len(l) > 0 {
			out.Leaks = append(out.Leaks, fmt.Sprintf("%s (%s): rò %s", cs.ID, c.recs[j].principal, strings.Join(l, ", ")))
		}
		j++
	}
	return out, nil
}
