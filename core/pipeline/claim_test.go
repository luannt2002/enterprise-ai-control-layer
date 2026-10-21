package pipeline

import (
	"context"
	"testing"
	"time"

	"github.com/luannt2002/enterprise-ai-control-layer/core/contracts"
	"github.com/luannt2002/enterprise-ai-control-layer/core/tenant"
	"github.com/luannt2002/enterprise-ai-control-layer/data/entityresolution"
	"github.com/luannt2002/enterprise-ai-control-layer/data/normalization"
	"github.com/luannt2002/enterprise-ai-control-layer/simulations/fixtures"
)

// verdictGap gọi thẳng (pack + danh mục thật của tenant cầu đường): cổng tự kiểm con số trong câu đúng là con số được xác nhận, không
// dựa vào việc luồng hỏi đã chọn đúng; ký hiệu chưa được giải thích ("~", ">", "/") là phần dư.
func TestVerdictGap(t *testing.T) {
	rt, err := tenant.Load("../../tenants/construction/pack")
	if err != nil {
		t.Fatal(err)
	}
	dir, err := fixtures.LoadDirectory("../../tenants/construction/seed/entities.yaml")
	if err != nil {
		t.Fatal(err)
	}
	a := &Ask{RT: rt}
	def, _ := rt.Metrics.Metric("issued_fuel")
	now := time.Date(2026, 10, 19, 10, 0, 0, 0, time.FixedZone("ICT", 7*3600))
	for _, c := range []struct {
		q, claimed, want string
	}{
		{"EX-03 hôm nay được cấp 3850 L đúng không?", "3850", ""},
		{"EX-03 hôm nay được cấp 3850 L đúng không?", "4000", "value"},
		{"EX-03 hôm nay được cấp ~4000 L đúng không?", "4000", "residual"},
		{"EX-03 hôm nay được cấp > 3000 L đúng không?", "3000", "residual"},
		{"EX-03 hôm nay được cấp 3850 L/ngày đúng không?", "3850", "residual"},
		{"EX-03 hôm nay được cấp 3850 đúng không?", "3850", "value"},
		{"EX-03 được cấp 3850 L đúng không?", "3850", "period"},
	} {
		in := rt.Normalizer.Normalize(c.q, normalization.Options{Now: now})
		res, err := entityresolution.Resolve(contracts.WithTenant(context.Background(), "abc-infra"), dir, in)
		if err != nil {
			t.Fatal(err)
		}
		got := a.verdictGap(in, mentionsOf(res), contracts.Intent{Name: "fuel_issued"}, def, contracts.Value{Kind: contracts.ValueQuantity, Number: c.claimed, Unit: "L"})
		if got != c.want {
			t.Errorf("verdictGap(%q, %s) = %q, muốn %q", c.q, c.claimed, got, c.want)
		}
	}
}
