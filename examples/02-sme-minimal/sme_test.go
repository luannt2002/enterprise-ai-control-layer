// Package smeminimal_test là pack thứ hai TỐI THIỂU (F-028, bậc B của báo cáo audit vNext §06.3): một nhà phân phối nhỏ giả lập — khách
// hàng → đơn hàng, hai chỉ số số theo ngày (số thùng xuất kho, doanh thu ghi nhận), hai vai. Dữ liệu ở data/ là dữ liệu TỰ VIẾT, giả lập.
// Đi trọn đường nạp (sources.yaml → quan sát / cách ly → danh mục + fact) rồi trọn luồng hỏi S0–S10 trong bộ nhớ, không gọi model và
// không đọc gì của pack tham chiếu ở tenants/.
package smeminimal_test

import (
	"context"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/luannt2002/enterprise-ai-control-layer/core/contracts"
	"github.com/luannt2002/enterprise-ai-control-layer/core/pipeline"
	"github.com/luannt2002/enterprise-ai-control-layer/core/tenant"
	"github.com/luannt2002/enterprise-ai-control-layer/data/ingestion"
)

// ngayHoi là "bây giờ" của ví dụ: 24/10/2026 10:00 giờ Việt Nam — "hôm qua" là 23/10.
var ngayHoi = time.Date(2026, 10, 24, 10, 0, 0, 0, time.FixedZone("ICT", 7*3600))

// napVaHoi nạp pack + dữ liệu giả lập qua bộ nạp dùng chung và dựng luồng hỏi trong bộ nhớ.
func napVaHoi(t *testing.T) (*pipeline.Ask, ingestion.Result) {
	t.Helper()
	rt, err := tenant.Load("pack")
	if err != nil {
		t.Fatalf("pack SME: %v", err)
	}
	sm, err := ingestion.LoadSources(filepath.Join("pack", "sources.yaml"), rt)
	if err != nil {
		t.Fatal(err)
	}
	dir := filepath.Join("data", rt.Meta.Tenant)
	res, err := ingestion.Ingest(context.Background(), dir, rt, sm, nil)
	if err != nil {
		t.Fatal(err)
	}
	entities, err := ingestion.LoadDirectory(dir, sm)
	if err != nil {
		t.Fatal(err)
	}
	return &pipeline.Ask{RT: rt, Deps: pipeline.Deps{Directory: entities, Facts: res.Facts(), Now: func() time.Time { return ngayHoi }}}, res
}

func nguoi(vai, phamVi string) contracts.Principal {
	pr := contracts.Principal{ID: "u-" + vai, TenantID: "sme-mau", Kind: contracts.PrincipalUser, Roles: []string{vai}}
	if phamVi != "" {
		pr.Attributes = map[string]string{"entity_scope": phamVi}
	}
	return pr
}

// Nạp: 8 bản ghi nguồn → 8 quan sát + 1 cách ly (mã đơn không có trong danh mục), không bản ghi nào rơi im lặng.
func TestNapDuLieuSME(t *testing.T) {
	_, res := napVaHoi(t)
	if len(res.Observations) != 8 || len(res.Quarantined) != 1 {
		t.Fatalf("quan sát %d, cách ly %d; muốn 8 và 1", len(res.Observations), len(res.Quarantined))
	}
	if q := res.Quarantined[0]; q.Key != "phieu-xuat.csv#PX-006" || q.Reason != ingestion.ReasonUnknownEntity {
		t.Errorf("cách ly = %+v, muốn PX-006 unknown_entity", q)
	}
}

// Luồng hỏi S0–S10 trên pack SME: ba câu số liệu (theo đơn, theo khách — gom mọi đơn của khách, chỉ số tiền), một câu hỏi lại (bí danh
// "đơn 7" của hai khách), câu cùng bí danh mà nêu khách thì router gỡ theo KIỂU CHA của pack, và vai phạm vi hẹp bị từ chối đơn của khách khác.
func TestLuongHoiSME(t *testing.T) {
	a, _ := napVaHoi(t)
	for _, c := range []struct {
		ten, vai, phamVi, hoi string
		route                 contracts.RouteKind
		outcome               contracts.Outcome
		chua                  string
	}{
		{"theo đơn", "quan_ly", "", "Hôm qua đơn DH-101 xuất kho bao nhiêu thùng hàng?", contracts.RouteSQLQuery, contracts.OutcomeResolved, ": 120 thùng."},
		{"theo khách", "quan_ly", "", "Hôm qua khách Minh Phát xuất kho bao nhiêu hàng?", contracts.RouteSQLQuery, contracts.OutcomeResolved, ": 240 thùng."},
		{"doanh thu", "quan_ly", "", "Doanh thu ghi nhận của đơn DH-101 hôm qua là bao nhiêu?", contracts.RouteSQLQuery, contracts.OutcomeResolved, ": 12.500.000 đ."},
		{"hỏi lại", "quan_ly", "", "Hôm qua đơn 7 xuất kho bao nhiêu hàng?", contracts.RouteClarification, contracts.OutcomeClarificationRequired, "DH-107"},
		{"gỡ theo khách", "quan_ly", "", "Hôm qua đơn 7 của khách Minh Phát xuất kho bao nhiêu hàng?", contracts.RouteSQLQuery, contracts.OutcomeResolved, ": 40 thùng."},
		{"ngoài phạm vi vai", "nhan_vien", "KH-HA", "Hôm qua đơn DH-101 xuất kho bao nhiêu hàng?", contracts.RouteReject, contracts.OutcomePermissionDenied, "không có quyền"},
	} {
		out, err := a.Handle(context.Background(), nguoi(c.vai, c.phamVi), contracts.AskRequest{ConversationID: "c", MessageID: "m", Text: c.hoi})
		if err != nil {
			t.Fatalf("%s: %v", c.ten, err)
		}
		r := out.Response
		if r.Route != c.route || r.Outcome != c.outcome || !strings.Contains(r.Answer, c.chua) {
			t.Errorf("%s: %q → %s/%s %q; muốn %s/%s chứa %q (lý do route: %s)", c.ten, c.hoi, r.Route, r.Outcome, r.Answer, c.route, c.outcome, c.chua, out.Plan.Reason)
		}
		if c.ten == "gỡ theo khách" {
			n := out.Plan.Narrowed
			if n.By != "khach_hang" || n.Chosen != "DH-107" {
				t.Errorf("gỡ mơ hồ phải theo kiểu cha khach_hang của pack: %+v", n)
			}
			if !slices.ContainsFunc(out.Decision.Why.Resolve.Reasons, func(s string) bool { return strings.Contains(s, "bằng khách hàng đã nêu trong câu") }) {
				t.Errorf("vết phải gọi kiểu cha bằng tên gọi của pack: %q", out.Decision.Why.Resolve.Reasons)
			}
		}
	}
}
