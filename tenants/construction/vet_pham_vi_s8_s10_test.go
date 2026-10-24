package construction_test

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/luannt2002/enterprise-ai-control-layer/core/contracts"
	"github.com/luannt2002/enterprise-ai-control-layer/core/pipeline"
	"github.com/luannt2002/enterprise-ai-control-layer/intelligence/models"
	"github.com/luannt2002/enterprise-ai-control-layer/intelligence/rag"
	"github.com/luannt2002/enterprise-ai-control-layer/observability/traces"
)

// V8a. Hai ô mới của vết — S8 (bản nháp: cách sinh, số ký tự, băm, ĐOẠN ĐẦU) và S10 (câu giao) — nằm trong bản ghi mà TestVetKhongLoThucTheNgoaiPhamVi quét. Hai test
// dưới đây khoá hai điều mà bộ quét đó không tự chứng minh: (1) bản ghi mà nó quét THẬT sự mang các dòng mới (bỏ dòng thì bộ quét không còn quét gì ở hai ô này);
// (2) đoạn đầu bản nháp, nơi chữ tài liệu vào vết, bị bộ che phạm vi che — dựng một tài liệu nhắc khoá máy của dự án khác.

func recordOfTurn(out pipeline.Outcome) traces.Record {
	return traces.NewRecord(traces.Turn{Decision: out.Decision, Spans: out.Spans, TotalMs: out.TotalMs, UnaccountedMs: out.UnaccountedMs, Plan: out.Plan, Response: out.Response})
}

func stepOfRecord(t *testing.T, rec traces.Record, stage string) traces.Step {
	t.Helper()
	for _, s := range rec.Steps {
		if s.Stage == stage {
			return s
		}
	}
	t.Fatalf("bản ghi không có bước %s", stage)
	return traces.Step{}
}

func TestVetQuetPhamViThayDongS8VaS10(t *testing.T) {
	a, _ := newAskTriThuc(t)
	eng := user("site_engineer", "PRJ-A")
	for _, c := range []struct{ name, q, s8 string }{
		{"số liệu", "EX-03 hôm nay được cấp bao nhiêu lít dầu?", "bản nháp dựng bằng mẫu câu metric_total (không gọi model): "},
		{"tri thức", "Quy trình xử lý chênh lệch nhiên liệu là gì?", "bản nháp do model extractive diễn đạt: "},
		{"hỏi lại", "RL-02 và máy 03 hôm nay được cấp bao nhiêu dầu?", "bản nháp dựng bằng mẫu câu clarify_missing (không gọi model): "},
	} {
		rec := recordOfTurn(ask(t, a, eng, c.q))
		b, err := json.Marshal(rec)
		if err != nil {
			t.Fatal(err)
		}
		// đúng bytes mà TestVetKhongLoThucTheNgoaiPhamVi quét: hai ô mới phải có mặt trong đó
		if s8, s10 := stepOfRecord(t, rec, "S8"), stepOfRecord(t, rec, "S10"); !strings.Contains(s8.Reason, c.s8) || !strings.Contains(s10.Reason, "câu giao: ") ||
			!strings.Contains(string(b), c.s8) || !strings.Contains(string(b), "câu giao: ") {
			t.Errorf("%s: bản ghi bị quét phải mang dòng S8 %q và dòng S10 \"câu giao: \":\n  S8: %s\n  S10: %s", c.name, c.s8, s8.Reason, s10.Reason)
		}
	}
}

// Tài liệu (kỹ sư PRJ-A được đọc) nhắc máy DZ-03 của PRJ-B; câu hỏi nhắc "máy 03" không gõ khoá nên DZ-03 là ứng viên NGOÀI phạm vi của lượt. Đoạn đầu bản nháp ở S8 là
// chữ tài liệu — khoá DZ-03 ở đó phải bị che với kỹ sư, và còn nguyên với giám đốc (đối chứng: bộ che không che oan người toàn quyền).
func TestVetS8CheKhoaNgoaiPhamVi(t *testing.T) {
	a := newAsk(t)
	a.Deps.Knowledge = rag.NewMemoryIndex("abc-infra", []rag.Document{{ID: "SOP-FUEL-03", Tenant: "abc-infra", Title: "Quy trình xử lý chênh lệch nhiên liệu của máy 03",
		ACL: []string{"director", "site_engineer"}, Body: "Máy ủi DZ-03 lập biên bản đối soát nhiên liệu trong vòng 24 giờ."}})
	a.Deps.Model = &models.Extractive{}
	const q = "Quy trình xử lý chênh lệch nhiên liệu của máy 03 là gì?"
	eng, dir := ask(t, a, user("site_engineer", "PRJ-A"), q), ask(t, a, user("director", ""), q)
	if eng.Plan.Route != contracts.RouteRAG || dir.Plan.Route != contracts.RouteRAG {
		t.Fatalf("tiền đề: route phải là RAG, được %s và %s", eng.Plan.Route, dir.Plan.Route)
	}
	engS8, dirS8 := stepOfRecord(t, recordOfTurn(eng), "S8"), stepOfRecord(t, recordOfTurn(dir), "S8")
	if !strings.Contains(dirS8.Reason, "Máy ủi DZ-03 lập biên bản") {
		t.Fatalf("đối chứng: người toàn quyền phải thấy chữ tài liệu ở đoạn đầu: %s", dirS8.Reason)
	}
	// bộ che xét khoá, mã và TÊN riêng, tên dài che trước: "Máy ủi DZ-03" thành MỘT dấu che, không còn "Máy ủi" lẻ
	if strings.Contains(engS8.Reason, "DZ-03") || strings.Contains(engS8.Reason, "Máy ủi") || !strings.Contains(engS8.Reason, "đầu “"+traces.OutOfScope+" lập biên bản") {
		t.Errorf("S8 của kỹ sư PRJ-A phải che tên và khoá của DZ-03 (dự án khác; người hỏi không gõ khoá): %s", engS8.Reason)
	}
}
