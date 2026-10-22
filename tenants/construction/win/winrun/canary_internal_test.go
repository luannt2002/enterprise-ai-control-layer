package winrun

import (
	"context"
	"maps"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/luannt2002/enterprise-ai-control-layer/core/contracts"
	"github.com/luannt2002/enterprise-ai-control-layer/data/entityresolution"
	"github.com/luannt2002/enterprise-ai-control-layer/data/sourceoftruth"
	"github.com/luannt2002/enterprise-ai-control-layer/evals"
	"github.com/luannt2002/enterprise-ai-control-layer/intelligence/rag"
	"github.com/luannt2002/enterprise-ai-control-layer/tenants/construction/win/gen"
)

// canaryCfg sinh bộ dữ liệu seed 42 (có cả thư mục tenant B) vào thư mục tạm và trả Config đường bộ nhớ, bộ ca WIN
// đầy đủ của gói (trong đó có các câu hỏi mồi nhắc tới tenant B).
func canaryCfg(t *testing.T) Config {
	t.Helper()
	if _, err := exec.LookPath("pdftotext"); err != nil {
		t.Skip("thiếu pdftotext (poppler) — bộ nạp PDF cần nó (ADR-0001 D4)")
	}
	dir := t.TempDir()
	if err := gen.Generate("../world.yaml", dir, "functional", 42); err != nil {
		t.Fatal(err)
	}
	return Config{Data: dir, Pack: "../../pack", World: "../world.yaml", CasesGlob: "../cases-*.yaml", Tenant: "abc-infra"}
}

// Ba kho cố ý rò, mỗi kho bỏ lọc tenant ở MỘT cổng đọc: đối chứng dương của canary (CLAUDE.md §1: gate chưa từng
// đỏ thì phải có ca âm chứng minh nó đỏ được). Chỉ dùng trong test.

// leakFacts trả fact của MỌI tenant.
type leakFacts struct{ *sharedStore }

func (l leakFacts) Facts(ctx context.Context, metric string, ids []string, from, to time.Time) ([]sourceoftruth.Fact, error) {
	var all sourceoftruth.MemoryStore
	for _, id := range slices.Sorted(maps.Keys(l.facts)) {
		all = append(all, l.facts[id]...)
	}
	return all.Facts(ctx, metric, ids, from, to)
}

// leakDirectory trả danh mục của MỌI tenant.
type leakDirectory struct{ *sharedStore }

func (l leakDirectory) Entities(ctx context.Context, types ...string) ([]entityresolution.Entity, error) {
	var all entityresolution.MemoryDirectory
	for _, id := range slices.Sorted(maps.Keys(l.dirs)) {
		all = append(all, l.dirs[id]...)
	}
	return all.Entities(ctx, types...)
}

// leakKnowledge đánh mất nhãn tenant của tài liệu: mọi tài liệu thành của người hỏi.
type leakKnowledge struct{ *sharedStore }

func (l leakKnowledge) Retrieve(ctx context.Context, pr contracts.Principal, query string, k int) ([]contracts.Evidence, error) {
	docs := slices.Clone(l.docs)
	for i := range docs {
		docs[i].Tenant = string(pr.TenantID)
	}
	return rag.NewMemoryIndex(string(pr.TenantID), docs).Retrieve(ctx, pr, query, k)
}

// Kho chung chứa cả hai tenant, hỏi các câu của A: 0 rò rỉ, B có THẬT trong kho, canary không lẫn mã dùng chung,
// và mọi ca ra đúng như đường một tenant (chế độ mới chỉ thêm phép đo, không đổi câu trả lời).
func TestCanaryHaiTenantSachVaGiuNguyenCauTraLoi(t *testing.T) {
	cfg := canaryCfg(t)
	plain, err := Run(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if plain.Canary != nil {
		t.Fatal("Config.Canary tắt mà Result.Canary khác nil")
	}
	cfg.Canary = true
	co, err := Run(cfg)
	if err != nil {
		t.Fatal(err)
	}
	c := co.Canary
	if c == nil {
		t.Fatal("Config.Canary bật mà Result.Canary nil")
	}
	if len(c.Tenants) != 2 || c.Tenants[0].ID != "abc-infra" || c.Tenants[1].ID != "def-build" {
		t.Fatalf("tenant trong kho chung = %+v, muốn [abc-infra def-build]", c.Tenants)
	}
	if b := c.Tenants[1]; b.Observations == 0 || b.Entities == 0 || b.Documents == 0 {
		t.Errorf("dữ liệu tenant B chưa vào kho: %+v", b)
	}
	for _, want := range []string{"9999", "7731", "def-build", "Cầu Z", "warehouse.csv#R-B01"} {
		if !slices.Contains(c.Canaries, want) {
			t.Errorf("thiếu canary %q trong %v", want, c.Canaries)
		}
	}
	for _, shared := range []string{"EX-03", "ex-03", "máy 03", "2026"} {
		if slices.Contains(c.Canaries, shared) {
			t.Errorf("%q dùng chung ở cả hai tenant, không được là canary", shared)
		}
	}
	if c.Answers == 0 || len(c.Leaks) != 0 {
		t.Errorf("quét %d câu, rò rỉ %v; muốn >0 câu và 0 rò rỉ", c.Answers, c.Leaks)
	}

	if len(co.Report.Results) != len(plain.Report.Results) || len(plain.Report.Results) == 0 {
		t.Fatalf("số ca: canary %d, một tenant %d", len(co.Report.Results), len(plain.Report.Results))
	}
	for i, m := range plain.Report.Results {
		if p := co.Report.Results[i]; p.ID != m.ID || p.Pass != m.Pass || p.Answer != m.Answer {
			t.Errorf("%s lệch giữa hai chế độ:\n  một tenant: %v %q\n  canary:     %v %q", m.ID, m.Pass, m.Answer, p.Pass, p.Answer)
		}
	}
	if !slices.Equal(co.G1Problems, plain.G1Problems) || co.Decisions != plain.Decisions || len(co.ReplayProblems) != len(plain.ReplayProblems) {
		t.Errorf("G1 / G6 đổi theo chế độ: G1 %d vs %d, quyết định %d vs %d, phát lại lệch %d vs %d",
			len(co.G1Problems), len(plain.G1Problems), co.Decisions, plain.Decisions, len(co.ReplayProblems), len(plain.ReplayProblems))
	}
}

// Đối chứng dương: kho rò dữ liệu B sang A ở từng cổng đọc thì canary PHẢI đỏ, và nêu đúng chuỗi của B.
func TestCanaryDoiChungDuong(t *testing.T) {
	for _, c := range []struct {
		name string
		wrap func(*sharedStore) coStore
		want string // canary (hạ chữ thường) phải có trong rò rỉ
	}{
		{"fact", func(s *sharedStore) coStore { return leakFacts{s} }, "warehouse.csv#r-b01"},
		{"tri thức", func(s *sharedStore) coStore { return leakKnowledge{s} }, "7731"},
	} {
		t.Run(c.name, func(t *testing.T) {
			cfg := canaryCfg(t)
			cfg.Canary, cfg.wrapStore = true, c.wrap
			r, err := Run(cfg)
			if err != nil {
				t.Fatal(err)
			}
			if r.Canary == nil || len(r.Canary.Leaks) == 0 {
				t.Fatalf("kho rò cổng %s mà canary KHÔNG đỏ: kiểm tra mù", c.name)
			}
			if got := strings.ToLower(strings.Join(r.Canary.Leaks, "\n")); !strings.Contains(got, c.want) {
				t.Errorf("rò rỉ không nêu %q: %v", c.want, r.Canary.Leaks)
			}
			t.Logf("%s: %d/%d câu rò, ví dụ %s", c.name, len(r.Canary.Leaks), r.Canary.Answers, r.Canary.Leaks[0])
		})
	}
}

// Bộ quét phải thấy canary ở MỌI phần người dùng nhìn được của câu trả lời, bỏ qua trace_id ngẫu nhiên, và không
// tính câu trả lời nhắc lại canary mà chính câu hỏi đã có.
func TestCanaryQuetMoiPhanCuaCauTraLoi(t *testing.T) {
	counter := evals.NewG7Counter(evals.G7CounterConfig{Canaries: []string{"9999", "Cầu Z", "warehouse.csv#R-B01"}})
	for _, c := range []struct {
		name     string
		question string
		resp     contracts.AskResponse
		want     string // canary phải bị bắt; rỗng = sạch
	}{
		{"câu chữ", "máy 03 hôm qua?", contracts.AskResponse{Answer: "Máy 03 xuất 9.999 L."}, "9999"},
		{"trích dẫn", "máy 03 hôm qua?", contracts.AskResponse{Citations: []contracts.Citation{{EvidenceID: "E1", Source: contracts.SourceRef{SourceID: "warehouse.csv#R-B01"}}}}, "warehouse.csv#r-b01"},
		{"xung đột", "máy 03 hôm qua?", contracts.AskResponse{Conflicts: []contracts.Conflict{{Others: []contracts.SourceValue{{SourceType: "erp", Number: "9999", Unit: "L"}}}}}, "9999"},
		{"câu hỏi lại", "máy 03 hôm qua?", contracts.AskResponse{Clarification: &contracts.Clarification{Candidates: []contracts.EntityRef{{ID: "PRJ-Z", Name: "Cầu Z"}}}}, "cầu z"},
		{"sạch", "máy 03 hôm qua?", contracts.AskResponse{Answer: "Máy 03 xuất 3.850 L.", Citations: []contracts.Citation{{EvidenceID: "E1", Source: contracts.SourceRef{SourceID: "warehouse.csv#R-005"}}}}, ""},
		{"trace_id ngẫu nhiên không bị quét", "máy 03 hôm qua?", contracts.AskResponse{Answer: "ok", TraceID: "1f9999aa-0000-4000-8000-000000009999"}, ""},
		{"câu hỏi đã có canary", "có phải 9999 L không?", contracts.AskResponse{Answer: "Không phải 9.999 L."}, ""},
	} {
		leaks := counter.Leaks(c.question, surface(c.resp))
		switch {
		case c.want == "" && len(leaks) != 0:
			t.Errorf("%s: muốn sạch, được %v", c.name, leaks)
		case c.want != "" && !slices.Contains(leaks, c.want):
			t.Errorf("%s: muốn bắt %q, được %v", c.name, c.want, leaks)
		}
	}
}

// Cổng danh bạ: bộ ca đóng băng KHÔNG nhắc bí danh nào chỉ có ở tenant B ("dự án Z"), nên danh bạ rò vẫn không làm
// câu nào lộ — canary mù ở cổng này nếu thiếu câu hỏi chạm tới tên của B. Test ghi số đo đó, rồi thêm MỘT ca mồi
// hỏi bằng bí danh của B và đòi canary đỏ: bộ quét bắt được rò danh bạ khi câu hỏi chạm tới nó.
func TestCanaryDoiChungDuongDanhBa(t *testing.T) {
	cfg := canaryCfg(t)
	cfg.Canary, cfg.wrapStore = true, func(s *sharedStore) coStore { return leakDirectory{s} }
	frozen, err := Run(cfg)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("danh bạ rò, CHỈ bộ ca đóng băng: %d/%d câu lộ canary (bộ ca không chạm tên riêng của B)", len(frozen.Canary.Leaks), frozen.Canary.Answers)

	dir := t.TempDir()
	src, err := os.ReadFile("../cases-independent.yaml")
	if err != nil {
		t.Fatal(err)
	}
	probe := "suite: win\ntenant: abc-infra\nreference_date: 2026-10-20\ncases:\n" +
		"  - id: WIN-A-901\n    failure_class: entity\n    authored_by: author\n    principal: u-director\n" +
		"    question: \"dự án Z hôm qua xuất bao nhiêu dầu?\"\n    expect: {route: SQL_QUERY, outcome: INSUFFICIENT_EVIDENCE}\n"
	for name, b := range map[string][]byte{"cases-independent.yaml": src, "cases-probe.yaml": []byte(probe)} {
		if err := os.WriteFile(filepath.Join(dir, name), b, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	cfg.CasesGlob = filepath.Join(dir, "cases-*.yaml")
	r, err := Run(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if r.Canary == nil || len(r.Canary.Leaks) == 0 {
		t.Fatal("danh bạ rò và câu mồi hỏi bằng bí danh của B mà canary KHÔNG đỏ: kiểm tra mù ở cổng danh bạ")
	}
	t.Logf("danh bạ rò, thêm ca mồi: %d/%d câu lộ, ví dụ %s", len(r.Canary.Leaks), r.Canary.Answers, r.Canary.Leaks[0])
}
