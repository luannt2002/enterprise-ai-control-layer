package pipeline

import (
	"context"
	"errors"
	"os"
	"regexp"
	"slices"
	"strings"
	"testing"

	"github.com/luannt2002/enterprise-ai-control-layer/core/contracts"
	"github.com/luannt2002/enterprise-ai-control-layer/core/tenant"
	"github.com/luannt2002/enterprise-ai-control-layer/data/entityresolution"
	"github.com/luannt2002/enterprise-ai-control-layer/simulations/fixtures"
)

// R2-03. Giải thích của vết (why.go) chỉ đọc và định dạng những gì luồng hỏi đã quyết; các test này khoá từng phần với dữ liệu dựng tay để
// một dòng bị bỏ hay đổi nguồn là test đỏ ngay tại gói này (không đợi test dựng vết cả lượt ở observability/traces).

// Mỗi mã chiều mà cổng phán quyết (claim.go) có thể trả đều phải có lời giải nghĩa, và không có lời giải nghĩa nào cho mã không tồn tại.
func TestGapMeaningDuMaChieu(t *testing.T) {
	src, err := os.ReadFile("claim.go")
	if err != nil {
		t.Fatal(err)
	}
	codes := map[string]bool{}
	for _, m := range regexp.MustCompile(`return (?:nil, )?"([a-z]+)"`).FindAllStringSubmatch(string(src), -1) {
		codes[m[1]] = true
	}
	if len(codes) < 8 {
		t.Fatalf("tiền đề: chỉ tìm thấy %d mã chiều trong claim.go (muốn ≥ 8): %v", len(codes), codes)
	}
	for c := range codes {
		if gapMeaning[c] == "" {
			t.Errorf("cổng phán quyết trả mã %q mà gapMeaning không giải nghĩa", c)
		}
	}
	for c := range gapMeaning {
		if !codes[c] {
			t.Errorf("gapMeaning có mã %q mà cổng phán quyết không bao giờ trả", c)
		}
	}
}

func TestExplainNormalizeGiuMoiThu(t *testing.T) {
	approx := contracts.Span{Kind: contracts.ValueQuantity, Raw: "4k", Rule: "number:digits+multiplier+unit_from_term:diesel_fuel", Confidence: 0.8, Value: contracts.Value{Kind: contracts.ValueQuantity, Number: "4000", Unit: "L"}}
	sure := contracts.Span{Kind: contracts.ValueDate, Raw: "hôm nay", Rule: "date:relative", Confidence: 1}
	n := explainNormalize(contracts.NormalizedInput{
		Spans:        []contracts.Span{approx, sure},
		Ambiguities:  []contracts.Ambiguity{{Raw: "DA", Reason: "viết tắt nhiều nghĩa", Candidates: []contracts.Value{{Kind: contracts.ValueTerm, Text: "diesel_fuel"}, {Kind: contracts.ValueTerm, Text: "project"}}}},
		InvalidDates: []string{"31/02"},
	})
	wantR := []string{"quantity:4k ← number:digits+multiplier+unit_from_term:diesel_fuel (độ tin cậy 0.80)", "date:hôm nay ← date:relative"}
	if !slices.Equal(n.Reasons, wantR) {
		t.Errorf("luật sinh đoạn (độ tin cậy chỉ in khi < 1): %q, muốn %q", n.Reasons, wantR)
	}
	if len(n.Rejected) != 2 || !strings.Contains(n.Rejected[0], `"DA"`) || !strings.Contains(n.Rejected[0], "diesel_fuel | project") || !strings.Contains(n.Rejected[1], `"31/02"`) {
		t.Errorf("cách đọc khác và ngày hỏng: %q", n.Rejected)
	}
	// câu không có đoạn nào: vẫn có MỘT ý ở Reasons để phân biệt "đã ghi, không có gì" với "chưa ghi"
	if e := explainNormalize(contracts.NormalizedInput{}); !e.Recorded() || len(e.Rejected) != 0 {
		t.Errorf("câu rỗng đoạn: %+v", e)
	}
}

func TestExplainResolveLocPhamViVaUngVienBiLoai(t *testing.T) {
	dz := contracts.EntityRef{ID: "DZ-03", Key: "DZ-03", Type: "equipment", Name: "Máy ủi DZ-03"}
	ex := contracts.EntityRef{ID: "EX-03", Key: "EX-03", Type: "equipment", Name: "Máy xúc EX-03", Mention: "ex 3", Start: 0, End: 5}
	in := contracts.NormalizedInput{Raw: "EX-03 hay máy 03?"}
	res := contracts.Resolution{Entities: []contracts.EntityRef{ex},
		Ambiguous: []contracts.AmbiguousMention{{Mention: "may 3", Type: "equipment", Candidates: []contracts.EntityRef{dz, ex}, Start: 10, End: 17}}}
	ci := clarifyInfo{used: true, before: []contracts.EntityRef{dz, ex, dz}, shown: []contracts.EntityRef{ex},
		hidden: []hiddenCandidate{{dz, "ngoài phạm vi dữ liệu của người hỏi"}, {dz, "ngoài phạm vi dữ liệu của người hỏi"}}}
	n := explainResolve(in, res, contracts.Narrowing{Slot: "subject", By: contracts.NarrowByScope, Before: []string{"DZ-03", "EX-03"}, After: []string{"EX-03"}, Chosen: "EX-03"}, ci)
	got := strings.Join(n.Reasons, "\n")
	for _, frag := range []string{`cụm "EX-03" (khoá so khớp "ex 3") khớp EX-03`, `cụm "máy 03" (khoá so khớp "may 3") khớp 2 thực thể: DZ-03, EX-03`,
		"trước lọc [DZ-03, EX-03] → sau lọc [EX-03] → chọn EX-03", "trước lọc phạm vi [DZ-03, EX-03] → sau lọc [EX-03]"} {
		if !strings.Contains(got, frag) {
			t.Errorf("thiếu %q trong:\n%s", frag, got)
		}
	}
	// DZ-03 bị loại hai lần (khi gỡ mơ hồ và khi dựng câu hỏi lại) nhưng mỗi nơi chỉ kể một dòng, không in lặp ứng viên trùng
	if len(n.Rejected) != 2 || !strings.Contains(n.Rejected[0], "khi gỡ mơ hồ") || !strings.Contains(n.Rejected[1], "khỏi câu hỏi lại") {
		t.Errorf("ứng viên bị loại: %q", n.Rejected)
	}
	// gỡ theo dự án nói đúng lý do loại
	p := explainResolve(in, res, contracts.Narrowing{Slot: "subject", By: contracts.NarrowByProject, Before: []string{"DZ-03", "EX-03"}, After: []string{"EX-03"}, Chosen: "EX-03"}, clarifyInfo{})
	if len(p.Rejected) != 1 || !strings.Contains(p.Rejected[0], "không thuộc dự án đã nêu") {
		t.Errorf("gỡ theo dự án: %q", p.Rejected)
	}
}

// failingDir là danh mục không đọc được: ứng viên không tra được tổ tiên thì bị loại (đóng mặc định) và vết nói đúng lý do đó.
type failingDir struct{}

func (failingDir) Entities(context.Context, ...string) ([]entityresolution.Entity, error) {
	return nil, errors.New("directory down")
}

func TestVisibleCandidatesGhiLyDoBiLoai(t *testing.T) {
	rt, err := tenant.Load("../../tenants/construction/pack")
	if err != nil {
		t.Fatal(err)
	}
	dir, err := fixtures.LoadDirectory("../../tenants/construction/seed/entities.yaml")
	if err != nil {
		t.Fatal(err)
	}
	ctx := contracts.WithTenant(context.Background(), "abc-infra")
	dz, ex := contracts.EntityRef{ID: "DZ-03", Key: "DZ-03"}, contracts.EntityRef{ID: "EX-03", Key: "EX-03"}
	eng := contracts.Principal{ID: "u", TenantID: "abc-infra", Roles: []string{"site_engineer"}, Attributes: map[string]string{"entity_scope": "PRJ-A"}}

	a := &Ask{RT: rt, Deps: Deps{Directory: dir}}
	shown, hidden := a.visibleCandidates(ctx, eng, []contracts.EntityRef{dz, ex})
	if len(shown) != 1 || shown[0].ID != "EX-03" || len(hidden) != 1 || hidden[0].ref.ID != "DZ-03" || !strings.Contains(hidden[0].why, "ngoài phạm vi dữ liệu") {
		t.Errorf("kỹ sư PRJ-A: hiển thị %v, loại %+v", shown, hidden)
	}
	// vai xem được tất cả: không loại ai
	director := contracts.Principal{ID: "d", TenantID: "abc-infra", Roles: []string{"director"}}
	if shown, hidden = a.visibleCandidates(ctx, director, []contracts.EntityRef{dz, ex}); len(shown) != 2 || len(hidden) != 0 {
		t.Errorf("giám đốc: hiển thị %v, loại %+v", shown, hidden)
	}
	// danh mục lỗi: đóng mặc định, lý do là không tra được tổ tiên
	broken := &Ask{RT: rt, Deps: Deps{Directory: failingDir{}}}
	if shown, hidden = broken.visibleCandidates(ctx, director, []contracts.EntityRef{ex}); len(shown) != 0 || len(hidden) != 1 || !strings.Contains(hidden[0].why, "không tra được tổ tiên") {
		t.Errorf("danh mục lỗi: hiển thị %v, loại %+v", shown, hidden)
	}
}

func TestExplainRouteGhiDeVaRouteKhaiBiChan(t *testing.T) {
	rt, err := tenant.Load("../../tenants/construction/pack")
	if err != nil {
		t.Fatal(err)
	}
	a := &Ask{RT: rt}
	routed := contracts.RoutePlan{Route: contracts.RouteSQLQuery, Reason: "ý định fuel_issued khớp luật", Intent: contracts.Intent{Name: "fuel_issued"},
		Checked: []contracts.RouteCheck{
			{Rule: "capability của ý định", Route: contracts.RouteReject, Detail: "metrics.read: có (vai director)"},
			{Rule: "route khai của ý định", Route: contracts.RouteSQLQuery, Fired: true, Detail: "fuel_issued → SQL_QUERY"},
		}}
	// không ghi đè: không có dòng ghi đè
	plain := a.explainRoute(routed, routed)
	if strings.Contains(strings.Join(plain.Reasons, "|"), "ghi đè") || len(plain.Rejected) != 1 || !strings.Contains(plain.Rejected[0], "REJECT (capability của ý định): không áp dụng — metrics.read: có") {
		t.Errorf("không ghi đè: %+v", plain)
	}
	if got := plain.Reasons[1]; got != "đã kiểm theo thứ tự: 1) capability của ý định: qua · 2) route khai của ý định — fuel_issued → SQL_QUERY → chọn SQL_QUERY" {
		t.Errorf("thang bậc kiểm: %q", got)
	}
	// luồng hỏi ghi đè sang CLARIFICATION: route của router nằm trong phương án bị loại kèm lý do ghi đè
	over := routed
	over.Route, over.Reason = contracts.RouteClarification, "câu xác nhận nêu nhiều con số khác nhau"
	n := a.explainRoute(routed, over)
	if !strings.Contains(strings.Join(n.Reasons, "|"), "luồng hỏi ghi đè route SQL_QUERY thành CLARIFICATION: câu xác nhận nêu nhiều con số khác nhau") ||
		!slices.ContainsFunc(n.Rejected, func(r string) bool { return strings.HasPrefix(r, "SQL_QUERY (route của router): bị ghi đè") }) {
		t.Errorf("ghi đè: %+v", n)
	}
	// route khai của ý định bị bậc kiểm chặn (route cuối khác route khai): kể lý do chặn của router
	deny := contracts.RoutePlan{Route: contracts.RouteReject, Reason: "thiếu capability metrics.read", Intent: contracts.Intent{Name: "fuel_issued"}}
	if n := a.explainRoute(deny, deny); !slices.ContainsFunc(n.Rejected, func(r string) bool {
		return r == "SQL_QUERY (route khai của ý định fuel_issued): không đi — thiếu capability metrics.read"
	}) {
		t.Errorf("route khai bị chặn: %q", n.Rejected)
	}
}
