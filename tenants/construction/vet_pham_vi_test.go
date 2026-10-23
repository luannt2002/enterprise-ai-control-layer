package construction_test

import (
	"context"
	"encoding/json"
	"os"
	"regexp"
	"strings"
	"testing"

	"go.yaml.in/yaml/v3"

	"github.com/luannt2002/enterprise-ai-control-layer/core/contracts"
	"github.com/luannt2002/enterprise-ai-control-layer/core/pipeline"
	"github.com/luannt2002/enterprise-ai-control-layer/data/entityresolution"
	"github.com/luannt2002/enterprise-ai-control-layer/observability/traces"
	"github.com/luannt2002/enterprise-ai-control-layer/simulations/fixtures"
)

// C9 vòng 2 (soát C8–C11, C9-S1): vết do chính người hỏi đọc lại (GET /v1/trace/{id}) không được mang khoá, mã hay tên của thực thể NGOÀI phạm
// vi dữ liệu của người hỏi mà câu hỏi không tự gõ khoá — ở BẤT KỲ bề mặt nào của bản ghi (Decision, mọi trường của mọi bước S0–S10), không chỉ
// dòng lý do S3. Quét bản ghi ĐÚNG như server lưu (traces.NewRecord, mã hoá JSON) cho mọi câu hỏi đã công bố (bộ 117, H1, H3, H4) cộng các câu
// dựng thêm nhắm cụm mơ hồ, dự án ngoài phạm vi, nhiều thực thể, câu hỏi lại, với năm người hỏi phạm vi hạn chế. Đối chứng dương: với cùng
// danh sách "ngoài phạm vi", bản ghi của người toàn quyền phải trúng ít nhất một lần — bộ quét tìm được đúng thứ nó tìm.
func TestVetKhongLoThucTheNgoaiPhamVi(t *testing.T) {
	a, _ := newAskTriThuc(t) // có chỉ mục tài liệu + model cục bộ (không trả tiền): câu quy trình / HYBRID của bộ công bố chạy hết S0–S10
	dir, err := fixtures.LoadDirectory("seed/entities.yaml")
	if err != nil {
		t.Fatalf("nạp danh mục: %v", err)
	}
	all, err := dir.Entities(context.Background())
	if err != nil {
		t.Fatalf("đọc danh mục: %v", err)
	}
	questions := publishedQuestions(t)
	questions = append(questions,
		"RL-02 và máy 03 hôm nay được cấp bao nhiêu dầu?",
		"Máy 03 hôm nay được cấp bao nhiêu dầu?",
		"Máy 03 của dự án B hôm nay được cấp bao nhiêu dầu?",
		"Máy 03 của dự án A hôm nay được cấp bao nhiêu dầu?",
		"Máy 07 hôm nay được cấp bao nhiêu dầu?",
		"Máy 03 và máy 07 hôm nay được cấp bao nhiêu dầu?",
		"Lu 02 và máy 03 hôm qua được cấp bao nhiêu dầu?",
		"Dự án B hôm nay được cấp bao nhiêu dầu?",
		"Máy 03 được cấp bao nhiêu?",
		"Máy 03 hôm nay được cấp 3850 L đúng không?",
		"Máy ủi 03 hôm nay được cấp bao nhiêu dầu?",
		"DZ-03 hôm nay được cấp bao nhiêu dầu?",
		"máy 03 hôm nay chênh lệch bao nhiêu?",
		"So sánh dầu được cấp của máy 03 hôm nay và hôm qua",
	)
	restricted := []contracts.Principal{
		user("site_engineer", "PRJ-A"), user("site_engineer", "PRJ-B"), user("site_engineer", "PRJ-A,PRJ-B"),
		user("project_manager", "PRJ-C"), user("driver", "PRJ-A"),
	}
	turns, leaks := 0, 0
	for _, pr := range restricted {
		hidden := outOfScope(t, a, pr, all)
		if len(hidden) == 0 {
			t.Fatalf("%s %s: không có thực thể nào ngoài phạm vi — ca thử vô nghĩa", pr.Roles[0], pr.Attributes["entity_scope"])
		}
		for _, q := range questions {
			turns++
			for _, hit := range leaked(t, a, pr, q, hidden) {
				if leaks++; leaks <= 8 {
					t.Errorf("%s phạm vi %s, câu %q: vết lộ %s", pr.Roles[0], pr.Attributes["entity_scope"], q, hit)
				}
			}
		}
	}
	if leaks > 0 {
		t.Fatalf("%d chỗ lộ thực thể ngoài phạm vi trên %d lượt", leaks, turns)
	}
	// Đối chứng dương: người toàn quyền, cùng danh sách "ngoài phạm vi" của kỹ sư PRJ-A — phải thấy (bộ quét không mù).
	director, eng := user("director", ""), user("site_engineer", "PRJ-A")
	seen := 0
	for _, q := range questions {
		seen += len(leaked(t, a, director, q, outOfScope(t, a, eng, all)))
	}
	if seen == 0 {
		t.Fatalf("đối chứng dương: bản ghi của người toàn quyền không trúng khoá nào của danh sách ngoài phạm vi — bộ quét không chứng minh được gì")
	}
}

// hiddenEntity là một thực thể ngoài phạm vi của người hỏi và các chuỗi của nó cần vắng khỏi vết: khoá / mã, và tên.
type hiddenEntity struct {
	key, name     string
	keyRe, nameRe *regexp.Regexp
}

// standalone khớp một trong các chuỗi khi nó đứng riêng: trước và sau không phải chữ, số, '-' hay '_' ("EX-03" không trúng trong "EX-030").
func standalone(alts ...string) *regexp.Regexp {
	var q []string
	for _, s := range alts {
		if s != "" {
			q = append(q, regexp.QuoteMeta(s))
		}
	}
	return regexp.MustCompile(`(?:^|[^\p{L}\p{N}_-])(?:` + strings.Join(q, "|") + `)(?:$|[^\p{L}\p{N}_-])`)
}

// outOfScope liệt kê thực thể của danh mục nằm ngoài phạm vi dữ liệu của pr (tính độc lập với mã che: lineage + chính sách của pack).
func outOfScope(t *testing.T, a *pipeline.Ask, pr contracts.Principal, all []entityresolution.Entity) []hiddenEntity {
	t.Helper()
	var out []hiddenEntity
	for _, e := range all {
		lin, err := entityresolution.Lineage(context.Background(), a.Deps.Directory, e.ID)
		if err != nil {
			t.Fatalf("tổ tiên của %s: %v", e.ID, err)
		}
		if a.RT.Policy.InScope(pr, lin) {
			continue
		}
		h := hiddenEntity{key: e.Key, name: e.Name, keyRe: standalone(e.Key, e.ID)}
		if e.Name != "" {
			h.nameRe = standalone(e.Name)
		}
		out = append(out, h)
	}
	return out
}

// leaked chạy một lượt, dựng bản ghi như server và trả các thực thể ngoài phạm vi lộ trong bản ghi: khoá / mã mà câu không tự gõ, hay tên mà câu
// không tự chứa (gõ đúng khoá vẫn không được thấy tên — F4).
func leaked(t *testing.T, a *pipeline.Ask, pr contracts.Principal, q string, hidden []hiddenEntity) []string {
	t.Helper()
	out, err := a.Handle(context.Background(), pr, contracts.AskRequest{ConversationID: "c1", MessageID: "m1", Text: q})
	if err != nil {
		t.Fatalf("%q: lỗi %v", q, err)
	}
	rec := traces.NewRecord(traces.Turn{Decision: out.Decision, Spans: out.Spans, TotalMs: out.TotalMs, UnaccountedMs: out.UnaccountedMs,
		Plan: out.Plan, Response: out.Response})
	b, err := json.Marshal(rec)
	if err != nil {
		t.Fatalf("mã hoá bản ghi: %v", err)
	}
	text := string(b)
	typed := " " + entityresolution.Key(q) + " "
	var hits []string
	for _, h := range hidden {
		// khoá: trừ khi người hỏi tự gõ khoá đó (F4 — lý do từ chối nêu đúng thứ họ hỏi)
		if !strings.Contains(typed, " "+entityresolution.Key(h.key)+" ") {
			if m := h.keyRe.FindString(text); m != "" {
				hits = append(hits, h.key+" (đoạn "+strings.TrimSpace(m)+")")
			}
		}
		// tên: trừ khi câu tự chứa tên đó (chữ của người hỏi nằm nguyên trong câu hỏi của bản ghi)
		if h.nameRe != nil && !strings.Contains(typed, " "+entityresolution.Key(h.name)+" ") {
			if m := h.nameRe.FindString(text); m != "" {
				hits = append(hits, h.key+" qua tên (đoạn "+strings.TrimSpace(m)+")")
			}
		}
	}
	return hits
}

// publishedQuestions đọc câu hỏi của bộ 117 và H1 / H3 / H4 (đã công bố).
func publishedQuestions(t *testing.T) []string {
	t.Helper()
	var qs []string
	for _, f := range []string{"win/cases-independent.yaml", "../../docs/g7/holdout/h1.yaml", "../../docs/g7/holdout-h3/h3.yaml",
		"../../docs/g7/holdout-h4/h4.yaml"} {
		b, err := os.ReadFile(f)
		if err != nil {
			t.Fatalf("đọc %s: %v", f, err)
		}
		var doc struct {
			Cases []struct {
				Question string `yaml:"question"`
			} `yaml:"cases"`
		}
		if err := yaml.Unmarshal(b, &doc); err != nil {
			t.Fatalf("đọc %s: %v", f, err)
		}
		for _, c := range doc.Cases {
			qs = append(qs, c.Question)
		}
	}
	if len(qs) < 400 {
		t.Fatalf("chỉ đọc được %d câu hỏi", len(qs))
	}
	return qs
}

// Người hỏi tự gõ khoá của thực thể ngoài phạm vi: vết vẫn nêu đúng khoá đó (lý do từ chối nói đúng thứ họ hỏi — F4) nhưng không kể tên hay
// loại của nó; người toàn quyền thấy tên (đối chứng).
func TestVetGiuKhoaNguoiHoiTuGoNhungCheTen(t *testing.T) {
	a := newAsk(t)
	record := func(pr contracts.Principal) string {
		out := ask(t, a, pr, "DZ-03 hôm nay được cấp bao nhiêu dầu?")
		b, err := json.Marshal(traces.NewRecord(traces.Turn{Decision: out.Decision, Spans: out.Spans, Plan: out.Plan, Response: out.Response}))
		if err != nil {
			t.Fatalf("mã hoá bản ghi: %v", err)
		}
		return string(b)
	}
	eng := record(user("site_engineer", "PRJ-A"))
	if !strings.Contains(eng, "DZ-03") || !strings.Contains(eng, "subject=DZ-03") {
		t.Errorf("người hỏi gõ DZ-03 thì vết phải nêu DZ-03 (khe subject, lý do từ chối)")
	}
	if strings.Contains(eng, "Máy ủi") || strings.Contains(eng, traces.OutOfScope) {
		t.Errorf("không được kể tên DZ-03 cho người ngoài phạm vi, và không có gì cần che ở câu này")
	}
	if dir := record(user("director", "")); !strings.Contains(dir, "Máy ủi DZ-03") {
		t.Errorf("đối chứng: người toàn quyền phải thấy tên Máy ủi DZ-03 trong vết")
	}
}
