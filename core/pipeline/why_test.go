package pipeline

import (
	"context"
	"errors"
	"fmt"
	"os"
	"regexp"
	"slices"
	"strings"
	"testing"

	"github.com/luannt2002/enterprise-ai-control-layer/core/contracts"
	"github.com/luannt2002/enterprise-ai-control-layer/core/tenant"
	"github.com/luannt2002/enterprise-ai-control-layer/data/entityresolution"
	"github.com/luannt2002/enterprise-ai-control-layer/data/sourceoftruth"
	"github.com/luannt2002/enterprise-ai-control-layer/intelligence/intent"
	"github.com/luannt2002/enterprise-ai-control-layer/observability/traces"
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
	wantR := []string{"quantity “4k” ← number:digits+multiplier+unit_from_term:diesel_fuel (độ tin cậy 0.80)", "date “hôm nay” ← date:relative"}
	if !slices.Equal(n.Reasons, wantR) {
		t.Errorf("luật sinh đoạn (độ tin cậy chỉ in khi < 1): %q, muốn %q", n.Reasons, wantR)
	}
	wantX := []string{"chỗ mơ hồ “DA” không được tự quyết (viết tắt nhiều nghĩa): các cách đọc diesel_fuel | project",
		"ngày “31/02” không đọc được thành ngày trên lịch: bỏ, không đoán và không rơi về ngày mặc định"}
	if !slices.Equal(n.Rejected, wantX) {
		t.Errorf("cách đọc khác và ngày hỏng: %q, muốn %q", n.Rejected, wantX)
	}
	// chữ của người hỏi đi qua contracts.Quote: nháy kép trong chữ KHÔNG bị thoát (thoát \" làm hỏng mẫu che bí mật của traces.Sanitize)
	q := explainNormalize(contracts.NormalizedInput{Spans: []contracts.Span{{Kind: contracts.ValueTerm, Raw: `a"b`, Rule: "r", Confidence: 1}}})
	if want := `term “a"b” ← r`; len(q.Reasons) != 1 || q.Reasons[0] != want {
		t.Errorf("nháy kép phải giữ nguyên trong cặp “…”: %q, muốn %q", q.Reasons, want)
	}
	// câu không có đoạn nào: vẫn có MỘT ý ở Reasons để phân biệt "đã ghi, không có gì" với "chưa ghi"
	if e := explainNormalize(contracts.NormalizedInput{}); !e.Recorded() || len(e.Rejected) != 0 {
		t.Errorf("câu rỗng đoạn: %+v", e)
	}
}

func TestExplainResolveKhongKeTrangThaiPhamViTungUngVien(t *testing.T) {
	dz := contracts.EntityRef{ID: "DZ-03", Key: "DZ-03", Type: "equipment", Name: "Máy ủi DZ-03"}
	ex := contracts.EntityRef{ID: "EX-03", Key: "EX-03", Type: "equipment", Name: "Máy xúc EX-03", Mention: "ex 3", Start: 0, End: 5}
	in := contracts.NormalizedInput{Raw: "EX-03 hay máy 03?"}
	res := contracts.Resolution{Entities: []contracts.EntityRef{ex},
		Ambiguous: []contracts.AmbiguousMention{{Mention: "may 3", Type: "equipment", Candidates: []contracts.EntityRef{dz, ex}, Start: 10, End: 17}}}
	ci := clarifyInfo{used: true, before: []contracts.EntityRef{dz, ex, dz}, shown: []contracts.EntityRef{ex},
		hidden: []hiddenCandidate{{dz, "ngoài phạm vi dữ liệu của người hỏi"}, {dz, "ngoài phạm vi dữ liệu của người hỏi"}}}
	nar := contracts.Narrowing{Slot: "subject", By: contracts.NarrowByScope, Before: []string{"DZ-03", "EX-03"}, After: []string{"EX-03"}, Chosen: "EX-03"}

	// phạm vi dữ liệu không hạn chế: đọc đủ tên và loại; ứng viên bị loại chỉ kể SỐ LƯỢNG và lý do chung (DZ-03 trùng hai lần chỉ tính một)
	n := explainResolve(in, res, nar, ci, true)
	if want := []string{
		"cụm “EX-03” (khoá so khớp “ex 3”) khớp EX-03 (Máy xúc EX-03, loại equipment)",
		"cụm “máy 03” (khoá so khớp “may 3”) khớp 2 thực thể: DZ-03, EX-03 — không chọn bừa",
		"gỡ mơ hồ cho khe subject bằng phạm vi dữ liệu của người hỏi: trước lọc 2 ứng viên → sau lọc 1 ứng viên [EX-03] → chọn EX-03",
		"ứng viên của câu hỏi lại: trước lọc phạm vi 2 ứng viên → sau lọc 1 ứng viên [EX-03]",
	}; !slices.Equal(n.Reasons, want) {
		t.Errorf("lý do:\n%q\nmuốn:\n%q", n.Reasons, want)
	}
	if want := []string{
		"1 ứng viên bị loại khi gỡ mơ hồ: ngoài phạm vi dữ liệu của người hỏi (vết không nêu trạng thái phạm vi của từng ứng viên)",
		"1 ứng viên bị loại khỏi câu hỏi lại: ngoài phạm vi dữ liệu của người hỏi (vết không nêu trạng thái phạm vi của từng ứng viên)",
	}; !slices.Equal(n.Rejected, want) {
		t.Errorf("loại:\n%q\nmuốn:\n%q", n.Rejected, want)
	}
	// phạm vi hạn chế: chỉ khoá, không tên, không loại
	if got := explainResolve(in, res, nar, ci, false).Reasons[0]; got != "cụm “EX-03” (khoá so khớp “ex 3”) khớp EX-03" {
		t.Errorf("người hỏi phạm vi hạn chế không được thấy tên / loại: %q", got)
	}
	// gỡ theo dự án nói đúng lý do loại
	p := explainResolve(in, res, contracts.Narrowing{Slot: "subject", By: contracts.NarrowByProject, Before: []string{"DZ-03", "EX-03"}, After: []string{"EX-03"}, Chosen: "EX-03"}, clarifyInfo{}, true)
	if want := []string{"1 ứng viên bị loại khi gỡ mơ hồ: không thuộc dự án đã nêu trong câu (vết không nêu trạng thái phạm vi của từng ứng viên)"}; !slices.Equal(p.Rejected, want) {
		t.Errorf("gỡ theo dự án: %q", p.Rejected)
	}
	// khoá trùng (một thực thể là ứng viên của hai cụm mơ hồ) được nói ra: số đếm khớp với bước "đúng một ứng viên" của router
	d := explainResolve(in, res, contracts.Narrowing{Slot: "subject", By: contracts.NarrowByScope, Before: []string{"DZ-03", "EX-03", "DZ-03", "EX-03"}, After: []string{"EX-03", "EX-03"}}, clarifyInfo{}, true)
	if want := "gỡ mơ hồ cho khe subject bằng phạm vi dữ liệu của người hỏi: trước lọc 4 ứng viên (2 khoá khác nhau) → sau lọc 2 ứng viên (1 khoá khác nhau) [EX-03] → giữ nguyên mơ hồ"; d.Reasons[2] != want {
		t.Errorf("khoá trùng: %q", d.Reasons[2])
	}
	// lọc xong không còn ai
	if e := explainResolve(in, res, contracts.Narrowing{Slot: "subject", By: contracts.NarrowByScope, Before: []string{"DZ-03"}}, clarifyInfo{}, true); !strings.HasSuffix(e.Reasons[2], "sau lọc không còn ứng viên nào → giữ nguyên mơ hồ") {
		t.Errorf("không còn ứng viên: %q", e.Reasons[2])
	}
	// hai lý do loại khỏi câu hỏi lại: mỗi lý do một dòng với số ứng viên riêng
	ex8, ex9 := contracts.EntityRef{ID: "EX-08", Key: "EX-08"}, contracts.EntityRef{ID: "EX-09", Key: "EX-09"}
	two := explainResolve(in, contracts.Resolution{}, contracts.Narrowing{}, clarifyInfo{used: true, hidden: []hiddenCandidate{
		{dz, "ngoài phạm vi dữ liệu của người hỏi"}, {ex8, "không tra được tổ tiên trong danh mục (đóng mặc định)"}, {ex9, "không tra được tổ tiên trong danh mục (đóng mặc định)"}}}, true)
	if want := []string{"1 ứng viên bị loại khỏi câu hỏi lại: ngoài phạm vi dữ liệu của người hỏi (vết không nêu trạng thái phạm vi của từng ứng viên)",
		"2 ứng viên bị loại khỏi câu hỏi lại: không tra được tổ tiên trong danh mục (đóng mặc định) (vết không nêu trạng thái phạm vi của từng ứng viên)"}; !slices.Equal(two.Rejected, want) {
		t.Errorf("hai lý do: %q", two.Rejected)
	}
	// bí danh có 42 ứng viên: danh sách bị cắt, số thực thể vẫn đúng
	var cands []contracts.EntityRef
	for i := range 42 {
		id := fmt.Sprintf("MX-%02d", i+1)
		cands = append(cands, contracts.EntityRef{ID: id, Key: id, Type: "equipment"})
	}
	raw := "máy 03"
	big := explainResolve(contracts.NormalizedInput{Raw: raw}, contracts.Resolution{Ambiguous: []contracts.AmbiguousMention{{Mention: "may 3", Candidates: cands, End: len(raw)}}}, contracts.Narrowing{}, clarifyInfo{}, true)
	if want := "cụm “máy 03” (khoá so khớp “may 3”) khớp 42 thực thể: MX-01, MX-02, MX-03, MX-04, MX-05, MX-06 và 36 ứng viên khác — không chọn bừa"; big.Reasons[0] != want {
		t.Errorf("danh sách 42 ứng viên: %q", big.Reasons[0])
	}
}

// Nguồn của ứng viên hỏi lại chỉ kể tối đa maxListed ứng viên (khoá trùng chỉ tính một), phần còn lại gộp một dòng.
func TestExplainIntentCatNguonUngVien(t *testing.T) {
	var shown []contracts.EntityRef
	var origins []intent.Origin
	for i := range 10 {
		id := fmt.Sprintf("MX-%02d", i+1)
		shown = append(shown, contracts.EntityRef{ID: id, Key: id}, contracts.EntityRef{ID: id, Key: id}) // mỗi ứng viên hiện hai lần (nhắc hai lần)
		origins = append(origins, intent.Origin{ID: id, From: intent.OriginS3, Note: "ghi chú " + id})
	}
	n := explainIntent(intent.Result{Why: intent.Why{Reasons: []string{"r"}, Origins: origins}}, clarifyInfo{used: true, shown: shown})
	if len(n.Reasons) != 1+maxListed+1 || n.Reasons[1] != "ứng viên hỏi lại MX-01 ← S3: ghi chú MX-01" || n.Reasons[maxListed] != "ứng viên hỏi lại MX-06 ← S3: ghi chú MX-06" ||
		n.Reasons[maxListed+1] != "và 4 ứng viên hỏi lại khác, nguồn xác định theo cùng cách" {
		t.Errorf("nguồn ứng viên: %q", n.Reasons)
	}
}

// Các bậc kiểm router giống hệt nhau (vòng lặp qua nhiều lần nhắc cùng một thực thể) gộp một bậc kèm số lần, ở cả thang bậc lẫn phương án bị loại.
func TestExplainRouteGopBacKiemGiongNhau(t *testing.T) {
	rt, err := tenant.Load("../../tenants/construction/pack")
	if err != nil {
		t.Fatal(err)
	}
	a := &Ask{RT: rt}
	same := contracts.RouteCheck{Rule: "phạm vi dữ liệu (câu hỏi nghĩa số liệu)", Route: contracts.RouteReject, Detail: "EX-03: trong phạm vi"}
	fired := contracts.RouteCheck{Rule: "hỏi lại nghĩa số liệu", Route: contracts.RouteClarification, Fired: true, Detail: "fuel_issued"}
	routed := contracts.RoutePlan{Route: contracts.RouteClarification, Reason: "hỏi lại", Checked: []contracts.RouteCheck{same, same, same, fired}}
	n := a.explainRoute(routed, routed)
	if want := "đã kiểm theo thứ tự: 1) phạm vi dữ liệu (câu hỏi nghĩa số liệu): qua (×3) · 2) hỏi lại nghĩa số liệu — fuel_issued → chọn CLARIFICATION"; len(n.Reasons) != 2 || n.Reasons[1] != want {
		t.Errorf("thang bậc: %q", n.Reasons)
	}
	if want := []string{"REJECT (phạm vi dữ liệu (câu hỏi nghĩa số liệu)): không áp dụng — EX-03: trong phạm vi (×3)"}; !slices.Equal(n.Rejected, want) {
		t.Errorf("phương án bị loại: %q", n.Rejected)
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

// N3 (soát vòng 2): các bậc "qua" LIÊN TIẾP cùng tên gộp một bậc kèm số lần dù dữ kiện khác nhau (mỗi thực thể một dữ kiện): câu nêu hàng trăm
// thực thể khác nhau không làm dòng thang bậc dài theo. Dữ kiện từng bậc vẫn nằm ở dòng phương án bị loại (đã có trần số ý). Bậc đã chặn và bậc
// "qua" cách nhau bởi bậc khác không gộp.
func TestExplainRouteGopBacQuaLienTiepKhacDuKien(t *testing.T) {
	rt, err := tenant.Load("../../tenants/construction/pack")
	if err != nil {
		t.Fatal(err)
	}
	a := &Ask{RT: rt}
	scope := func(key string) contracts.RouteCheck {
		return contracts.RouteCheck{Rule: "phạm vi dữ liệu (câu hỏi nghĩa số liệu)", Route: contracts.RouteReject, Detail: key + ": trong phạm vi"}
	}
	other := contracts.RouteCheck{Rule: "capability của ý định", Route: contracts.RouteReject, Detail: "metrics.read: có (vai director)"}
	fired := contracts.RouteCheck{Rule: "phạm vi dữ liệu (câu hỏi nghĩa số liệu)", Route: contracts.RouteReject, Fired: true, Detail: "DZ-03: ngoài phạm vi"}
	routed := contracts.RoutePlan{Route: contracts.RouteReject, Reason: "chặn", Checked: []contracts.RouteCheck{scope("EX-01"), scope("EX-02"), scope("EX-03"), other, scope("EX-04"), scope("EX-05"), fired}}
	n := a.explainRoute(routed, routed)
	want := "đã kiểm theo thứ tự: 1) phạm vi dữ liệu (câu hỏi nghĩa số liệu): qua (×3) · 2) capability của ý định: qua · 3) phạm vi dữ liệu (câu hỏi nghĩa số liệu): qua (×2) · " +
		"4) phạm vi dữ liệu (câu hỏi nghĩa số liệu) — DZ-03: ngoài phạm vi → chọn REJECT"
	if len(n.Reasons) != 2 || n.Reasons[1] != want {
		t.Errorf("thang bậc:\n  được: %q\n  muốn: %q", n.Reasons, want)
	}
	// mỗi dữ kiện khác nhau vẫn có dòng phương án bị loại riêng (trần số ý do Compact lo)
	if len(n.Rejected) != 6 || n.Rejected[0] != "REJECT (phạm vi dữ liệu (câu hỏi nghĩa số liệu)): không áp dụng — EX-01: trong phạm vi" {
		t.Errorf("phương án bị loại: %q", n.Rejected)
	}
	// hai bậc "qua" cùng tên nhưng khác route không gộp; bậc đã chọn không gộp với bậc "qua" liền sau dù cùng tên và cùng route
	passA1 := contracts.RouteCheck{Rule: "bậc A", Route: contracts.RouteReject, Detail: "d1"}
	passA2 := contracts.RouteCheck{Rule: "bậc A", Route: contracts.RouteClarification, Detail: "d2"}
	pickedB := contracts.RouteCheck{Rule: "bậc B", Route: contracts.RouteSQLQuery, Fired: true, Detail: "d3"}
	passB := contracts.RouteCheck{Rule: "bậc B", Route: contracts.RouteSQLQuery, Detail: "d4"}
	sep := a.explainRoute(contracts.RoutePlan{Route: contracts.RouteSQLQuery, Reason: "r", Checked: []contracts.RouteCheck{passA1, passA2, pickedB, passB}}, contracts.RoutePlan{Route: contracts.RouteSQLQuery, Reason: "r"})
	if got, want := sep.Reasons[1], "đã kiểm theo thứ tự: 1) bậc A: qua · 2) bậc A: qua · 3) bậc B — d3 → chọn SQL_QUERY · 4) bậc B: qua"; got != want {
		t.Errorf("bậc khác route / bậc đã chọn không gộp: %q, muốn %q", got, want)
	}
	// 280 bậc khác dữ kiện: vẫn MỘT bậc kèm số lần, dòng ngắn
	var many []contracts.RouteCheck
	for i := range 280 {
		many = append(many, scope(fmt.Sprintf("MX-%03d", i%140+1))) // mỗi dữ kiện lặp hai lần (không liền nhau): bậc giống hệt gộp trước, rồi các bậc "qua" liền nhau gộp tiếp, cộng dồn số lần
	}
	m := a.explainRoute(contracts.RoutePlan{Route: contracts.RouteClarification, Reason: "r", Checked: many}, contracts.RoutePlan{Route: contracts.RouteClarification, Reason: "r"})
	if got, want := m.Reasons[1], "đã kiểm theo thứ tự: 1) phạm vi dữ liệu (câu hỏi nghĩa số liệu): qua (×280)"; got != want {
		t.Errorf("280 bậc: %q, muốn %q", got, want)
	}
}

// N1 (S1, soát vòng 2): con số của người hỏi là chữ lấy từ câu — ở cả ba dòng nó đi vào (đầu vào của cổng, phép so sánh, bằng chứng của người hỏi
// ở S7) nó đi qua contracts.Quote, kể cả khi bị định dạng lại ("48.291.637"), nên Why.WithholdQuoted che được khi câu mang bí mật. Dựng thẳng đầu
// vào của noteGate / noteMetricContext vì lượt thật có chữ bí mật luôn dừng ở chiều "residual" trước khi tới hai dòng sau.
func TestSoCuaNguoiHoiQuaQuoteOCaBaDong(t *testing.T) {
	rt, err := tenant.Load("../../tenants/construction/pack")
	if err != nil {
		t.Fatal(err)
	}
	a := &Ask{RT: rt}
	claimed := contracts.Value{Kind: contracts.ValueQuantity, Number: "48291637", Unit: "L"}
	total := contracts.Value{Kind: contracts.ValueQuantity, Number: "3850", Unit: "L"}
	var why traces.Why
	a.noteGate(&why, gate{in: contracts.NormalizedInput{Raw: "EX-03 hôm nay được cấp 48291637 L đúng không?"}, metric: "issued_fuel", base: true, confirming: true, confirm: true,
		tmpl: "confirm_no", claimed: claimed, total: &total, subject: "EX-03", used: []string{"warehouse_slip"}})
	pkg := contracts.ContextPackage{Evidence: []contracts.Evidence{{ID: "question", Kind: contracts.EvidenceQuestion, Values: []contracts.Value{claimed}}}}
	a.noteMetricContext(&why, "", sourceoftruth.Result{}, pkg, "EX-03", 1)

	wantInputs := "đầu vào của cổng: chủ thể EX-03 · kỳ đọc được 0 · cụm xác nhận 1 · câu có dấu true · nguồn đã tạo nên con số warehouse_slip · con số xác nhận “48.291.637” L"
	wantCompare := `so sánh bằng tuyệt đối số người hỏi “48.291.637” với số có thẩm quyền 3.850 → "Không." (confirm_no)`
	wantQuestion := "question: con số do người hỏi nêu (“48291637L”) — nguồn là chính câu hỏi, không phải số liệu"
	if !slices.Contains(why.Validate.Reasons, wantInputs) || !slices.Contains(why.Validate.Reasons, wantCompare) || !slices.Contains(why.Context.Reasons, wantQuestion) {
		t.Fatalf("ba dòng có số của người hỏi phải đặt số trong cặp “…”:\n%s\n%s", strings.Join(why.Validate.Reasons, "\n"), strings.Join(why.Context.Reasons, "\n"))
	}
	// câu mang bí mật nhận ra được: mọi đoạn trong cặp “…” bị che, không còn chữ số nào của người hỏi (dạng thô lẫn dạng định dạng lại)
	masked := why.WithholdQuoted()
	for _, line := range slices.Concat(masked.Validate.Reasons, masked.Context.Reasons) {
		if d := strings.NewReplacer(".", "", ",", "").Replace(line); strings.Contains(d, "48291637") {
			t.Errorf("còn con số của người hỏi sau khi che: %q", line)
		}
	}
	if want := strings.Replace(wantInputs, "“48.291.637”", "“đã che”", 1); !slices.Contains(masked.Validate.Reasons, want) {
		t.Errorf("dòng đầu vào của cổng sau khi che:\n%s", strings.Join(masked.Validate.Reasons, "\n"))
	}
}

// N2 (soát vòng 2: X6): lời giải nghĩa của TỪNG chiều khoá nguyên chữ — đổi lời của chiều này thành lời của chiều khác (hay sửa một chữ) là test đỏ.
// TestGapMeaningDuMaChieu chỉ đòi mỗi mã có lời giải nghĩa; test này đòi lời đó đúng với chiều.
func TestGapMeaningNguyenChu(t *testing.T) {
	want := map[string]string{
		"metric":   "chỉ số không do người hỏi nêu rõ (ý định chung theo quy ước của pack), hay có cụm chỉ số / lượng khác ở ngữ cảnh có thể là chỉ số của con số",
		"period":   "mốc thời gian: câu không nêu mốc (kỳ mặc định của pack) hoặc nêu nhiều mốc",
		"accent":   "câu không dấu: bộ chuẩn hoá khớp bỏ dấu nên nhiều chữ trùng nhau",
		"marker":   "nhiều cụm xác nhận là nhiều claim, một phán quyết không trả lời đủ",
		"subject":  "chủ thể của con số chưa giải: có tên khác chủ thể, bí danh mơ hồ chưa có bằng chứng dương, hay con số chưa gắn vào chủ thể",
		"value":    "con số xác nhận chưa đọc chắc: có số khác, thiếu đơn vị, ngoài miền của chỉ số, đọc bằng chữ mơ hồ hay nhắc lại sau mệnh đề khác",
		"source":   "nguồn: tên nguồn trong câu không khớp nguồn đã tạo nên con số trả lời, hay có cụm nguồn khác ở ngữ cảnh",
		"residual": "phần chữ hay ký hiệu chưa được giải thích (từ lạ, '~', '>', '%', '/')",
	}
	if len(gapMeaning) != len(want) {
		t.Errorf("gapMeaning có %d chiều, muốn %d", len(gapMeaning), len(want))
	}
	for code, w := range want {
		if got := gapMeaning[code]; got != w {
			t.Errorf("chiều %q:\n  được: %q\n  muốn: %q", code, got, w)
		}
	}
}

// S7 của route số liệu kể cả bằng chứng đi kèm thêm sau truy vấn: con số của người hỏi, số chỗ lệch không kê ra, chỉ số khác cùng kỳ — từng dòng khoá nguyên văn
// (đếm chỗ lệch chỉ xuất hiện khi một tổng có hơn 3 chỗ lệch, seed không có ca như thế nên dựng thẳng đầu vào).
func TestNoteMetricContextBangChungDiKemThemSauTruyVan(t *testing.T) {
	rt, err := tenant.Load("../../tenants/construction/pack")
	if err != nil {
		t.Fatal(err)
	}
	a := &Ask{RT: rt}
	pkg := contracts.ContextPackage{Evidence: []contracts.Evidence{
		{ID: "question", Kind: contracts.EvidenceQuestion, Values: []contracts.Value{{Number: "4123", Unit: "L"}}},
		{ID: "count:conflicts", Kind: contracts.EvidenceCalculation, Values: []contracts.Value{{Number: "5"}}},
		{ID: "calc:billed_fuel", Kind: contracts.EvidenceCalculation},
	}}
	var why traces.Why
	a.noteMetricContext(&why, "", sourceoftruth.Result{}, pkg, "EX-03", 2)
	want := []string{
		"gói ngữ cảnh: 3 bằng chứng (0 dữ liệu nguồn, 0 bằng chứng khác của truy vấn, 3 bằng chứng đi kèm thêm sau truy vấn)",
		"question: con số do người hỏi nêu (“4123L”) — nguồn là chính câu hỏi, không phải số liệu",
		"count:conflicts: đếm số chỗ lệch không kê ra trong câu",
		"calc:billed_fuel: chỉ số khác cùng kỳ, kê thêm (không phải chỉ số được hỏi)",
		"quyền: phạm vi dữ liệu đã kiểm ở S5 cho thực thể gốc EX-03; truy vấn chạy trên 2 thực thể (gốc và con cháu), không lọc bằng chứng theo quyền sau truy vấn",
	}
	if !slices.Equal(why.Context.Reasons, want) {
		t.Errorf("S7:\n  được: %q\n  muốn: %q", why.Context.Reasons, want)
	}
}
