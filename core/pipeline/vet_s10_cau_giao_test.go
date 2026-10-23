package pipeline

import (
	"context"
	"strconv"
	"strings"
	"testing"
	"time"
	"unicode/utf8"

	"github.com/luannt2002/enterprise-ai-control-layer/core/contracts"
	"github.com/luannt2002/enterprise-ai-control-layer/core/tenant"
	"github.com/luannt2002/enterprise-ai-control-layer/intelligence/models"
	"github.com/luannt2002/enterprise-ai-control-layer/intelligence/rag"
	"github.com/luannt2002/enterprise-ai-control-layer/observability/traces"
	"github.com/luannt2002/enterprise-ai-control-layer/simulations/fixtures"
)

// V8AS-01 (soát vòng 1 của V8a). Lỗi ở S10 — câu giao bị cắt ngay trước khi dựng Decision — phải hiện ra ở S10, không bị đọc thành lỗi S8. Dấu vân tay của câu giao
// ghi SỚM hơn chỗ câu giao chốt thì vẫn là dấu vân tay của câu CŨ: bảng vết nói "câu giao trùng bản nháp" trong khi câu đã bị cắt. Test chạy qua Handle thật, gieo lỗi
// bằng điểm móc beforeDecision đặt đúng chỗ đó (hình lỗi của phép đo 23/10: cắt câu giao ở 120 ký tự rồi thêm "…").

// vetAsk dựng một luồng hỏi thật: pack, danh mục và fact của tenant xây dựng, một tài liệu quy trình cho giám đốc, model tất định trả nguyên văn đoạn tài liệu.
func vetAsk(t *testing.T) *Ask {
	t.Helper()
	rt, err := tenant.Load("../../tenants/construction/pack")
	if err != nil {
		t.Fatalf("nạp pack: %v", err)
	}
	dir, err := fixtures.LoadDirectory("../../tenants/construction/seed/entities.yaml")
	if err != nil {
		t.Fatalf("nạp danh mục: %v", err)
	}
	facts, err := fixtures.LoadFacts("../../tenants/construction/seed/facts.yaml", rt, dir)
	if err != nil {
		t.Fatalf("nạp fact: %v", err)
	}
	now := time.Date(2026, 10, 19, 10, 0, 0, 0, time.FixedZone("ICT", 7*3600))
	return &Ask{RT: rt, Deps: Deps{Directory: dir, Facts: facts, Now: func() time.Time { return now }, DataVersion: "seed-test",
		Knowledge: rag.NewMemoryIndex("abc-infra", []rag.Document{{ID: "SOP-FUEL-01", Tenant: "abc-infra", Title: "Quy trình xử lý chênh lệch nhiên liệu",
			ACL: []string{"director"}, Body: "1. Khi lượng xuất kho và lượng tiêu thụ đo được lệch quá dung sai, tổ trưởng kho lập biên bản đối soát trong vòng 24 giờ."}}),
		Model: &models.Extractive{}}}
}

var vetDirector = contracts.Principal{ID: "u-dir", TenantID: "abc-infra", Kind: contracts.PrincipalUser, Roles: []string{"director"}}

// vetTurn hỏi một câu và trả cả Outcome lẫn bản ghi (bảng vết dựng từ đúng những gì Handle xuất).
func vetTurn(t *testing.T, a *Ask, pr contracts.Principal, q string) (Outcome, traces.Record) {
	t.Helper()
	out, err := a.Handle(context.Background(), pr, contracts.AskRequest{ConversationID: "c", MessageID: "m", Text: q})
	if err != nil {
		t.Fatalf("%q: %v", q, err)
	}
	return out, traces.NewRecord(traces.Turn{Decision: out.Decision, Spans: out.Spans, TotalMs: out.TotalMs, UnaccountedMs: out.UnaccountedMs, Plan: out.Plan, Response: out.Response})
}

func vetStep(t *testing.T, rec traces.Record, stage string) traces.Step {
	t.Helper()
	for _, s := range rec.Steps {
		if s.Stage == stage {
			return s
		}
	}
	t.Fatalf("bản ghi không có bước %s", stage)
	return traces.Step{}
}

// cutAnswer dựng lỗi gieo: giữ n ký tự đầu của câu giao rồi thêm "…".
func cutAnswer(n int) func(*contracts.AskResponse) {
	return func(r *contracts.AskResponse) {
		if rs := []rune(r.Answer); len(rs) > n {
			r.Answer = string(rs[:n]) + "…"
		}
	}
}

func TestVetS10CauGiaoBiCatTruocDecisionThiS10LaBuocLechDauTien(t *testing.T) {
	for _, c := range []struct {
		name, q string
		cut     int
		route   contracts.RouteKind
	}{
		{"route tri thức: câu giao bọc bản nháp", "Quy trình xử lý chênh lệch nhiên liệu là gì?", 120, contracts.RouteRAG},
		{"route số liệu: câu giao chính là bản nháp mẫu câu", "RL-02 hôm nay được cấp bao nhiêu lít dầu?", 30, contracts.RouteSQLQuery}, // RL-02 một nguồn: S6 không lệch vì xung đột
	} {
		t.Run(c.name, func(t *testing.T) {
			a := vetAsk(t)
			// đối chứng: lượt khoẻ không có bước lệch nào
			healthy, hrec := vetTurn(t, a, vetDirector, c.q)
			if healthy.Plan.Route != c.route || vetStep(t, hrec, "S10").Status != traces.StatusOK || !strings.Contains(traces.FormatTable(hrec), "Không có bước lệch.") {
				t.Fatalf("tiền đề: lượt khoẻ route %s phải không có bước lệch:\n%s", healthy.Plan.Route, traces.FormatTable(hrec))
			}
			if utf8.RuneCountInString(healthy.Response.Answer) <= c.cut+1 {
				t.Fatalf("tiền đề: câu giao %q phải dài hơn %d ký tự để lỗi gieo có tác dụng", healthy.Response.Answer, c.cut+1)
			}

			a.beforeDecision = cutAnswer(c.cut)
			out, rec := vetTurn(t, a, vetDirector, c.q)
			ans := out.Decision.Answer
			if n := utf8.RuneCountInString(ans); n != c.cut+1 || !strings.HasSuffix(ans, "…") || out.Response.Answer != ans {
				t.Fatalf("tiền đề: lỗi gieo phải cắt câu giao còn %d ký tự ở CẢ Decision lẫn Response, được %d ký tự: %q", c.cut+1, n, ans)
			}
			s10, table := vetStep(t, rec, "S10"), traces.FormatTable(rec)
			// dấu vân tay là của CÂU ĐÃ CẮT (chữ nằm trong bản ghi), tính độc lập với mã sản xuất
			want := "câu giao: " + strconv.Itoa(utf8.RuneCountInString(ans)) + " ký tự · băm " + sha12(ans)
			if !strings.Contains(s10.Reason, want) {
				t.Errorf("S10 phải mang dấu vân tay của câu giao đã chốt %q:\n  %s", want, s10.Reason)
			}
			if !strings.Contains(s10.Reason, "câu giao KHÔNG chứa nguyên văn bản nháp ở S8") || strings.Contains(s10.Reason, "trùng bản nháp ở S8") {
				t.Errorf("S10 phải nói câu giao KHÔNG chứa bản nháp đã qua S9, không được khẳng định ngược lại:\n  %s", s10.Reason)
			}
			if s10.Status != traces.StatusDeviant || !strings.Contains(table, "BƯỚC LỆCH ĐẦU TIÊN: S10 respond") {
				t.Errorf("S10 phải là bước lệch đầu tiên (người đọc không được đổ lỗi cho S8):\n%s", table)
			}
			// S8 không được kể là lệch: bản nháp S8 sinh ra đúng, lỗi nằm sau S9
			if s8 := vetStep(t, rec, "S8"); s8.Status == traces.StatusDeviant {
				t.Errorf("S8 không lệch trong lượt này: %+v", s8)
			}
		})
	}
}
