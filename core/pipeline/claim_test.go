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
// dựa vào việc luồng hỏi đã chọn đúng; ký hiệu chưa được giải thích ("~", ">", "/") là phần dư. Chủ thể là thực thể nhắc đầu tiên
// (như router; bí danh mơ hồ thì ghi rõ thực thể router gỡ ra), nguồn dùng mặc định là phiếu xuất kho.
func TestVerdictGap(t *testing.T) {
	rt, err := tenant.Load("../../tenants/construction/pack")
	if err != nil {
		t.Fatal(err)
	}
	dir, err := fixtures.LoadDirectory("../../tenants/construction/seed/entities.yaml")
	if err != nil {
		t.Fatal(err)
	}
	a := &Ask{RT: rt, Deps: Deps{Directory: dir}}
	def, _ := rt.Metrics.Metric("issued_fuel")
	now := time.Date(2026, 10, 19, 10, 0, 0, 0, time.FixedZone("ICT", 7*3600))
	ctx := contracts.WithTenant(context.Background(), "abc-infra")
	for _, c := range []struct {
		q, claimed, want string
		subject          string // trống = thực thể nhắc đầu tiên; bí danh mơ hồ do router gỡ thì ghi rõ
		used             []string
	}{
		{q: "EX-03 hôm nay được cấp 3850 L đúng không?", claimed: "3850"},
		{q: "EX-03 hôm nay được cấp 3850 L đúng không?", claimed: "4000", want: "value"},
		{q: "EX-03 hôm nay được cấp ~4000 L đúng không?", claimed: "4000", want: "residual"},
		{q: "EX-03 hôm nay được cấp > 3000 L đúng không?", claimed: "3000", want: "residual"},
		{q: "EX-03 hôm nay được cấp 3850 L/ngày đúng không?", claimed: "3850", want: "residual"},
		{q: "EX-03 hôm nay được cấp 3850 đúng không?", claimed: "3850", want: "value"},
		{q: "EX-03 được cấp 3850 L đúng không?", claimed: "3850", want: "period"},
		// số trần có dấu chấm / phẩy: mơ hồ phần nghìn / thập phân, không nói đơn vị
		{q: "EX-03 hôm nay được cấp 3.850 đúng không?", claimed: "3850", want: "value"},
		// ngoài miền một lần đo: "3.850 m3" đọc thành 3.850.000 L
		{q: "EX-03 hôm nay được cấp 3.850 m3 đúng không?", claimed: "3850000", want: "value"},
		// gạch dài dính vào số là dấu âm không đọc được, không phải dấu câu
		{q: "EX-03 hôm nay được cấp —3850 L đúng không?", claimed: "3850", want: "residual"},
		// câu không dấu: khớp bỏ dấu làm chữ trùng nhau
		{q: "EX-03 hom nay duoc cap 3850 L dung khong?", claimed: "3850", want: "accent"},
		// hai cụm xác nhận là hai claim
		{q: "EX-03 hôm nay được cấp 3850 L đúng không, 3850 L đúng không?", claimed: "3850", want: "marker"},
		// chủ thể: dự án B không phải cấp cha của EX-03; dự án A (nhắc đầu) chứa EX-03 chứ không phải cha của nó
		{q: "EX-03 của dự án B hôm nay được cấp 3850 L đúng không?", claimed: "3850", want: "subject"},
		{q: "Dự án A hôm nay được cấp 3850 L cho EX-03 đúng không?", claimed: "3850", want: "subject"},
		{q: "Máy 03 của dự án A hôm nay được cấp 3850 L đúng không?", claimed: "3850", subject: "EX-03"}, // "máy 03" mơ hồ, router gỡ nhờ dự án A
		// nguồn: tên nguồn chỉ được giải thích khi số trả lời lấy đúng và chỉ từ nguồn đó
		{q: "EX-03 hôm nay phiếu xuất kho ghi 3850 L đúng không?", claimed: "3850"},
		{q: "EX-03 hôm nay phiếu xuất kho ghi 3900 L đúng không?", claimed: "3900", want: "source", used: []string{"erp"}},
		{q: "EX-03 hôm nay phiếu kho ghi 3900 L đúng không?", claimed: "3900", want: "source", used: []string{"erp"}},
		{q: "EX-03 hôm nay ERP ghi 3900 L đúng không?", claimed: "3900", want: "source"},
		// "tuần" trước một ngày là tuần chứa ngày đó, không phải ngày
		{q: "EX-03 tuần 15/9 được cấp 3500 L đúng không?", claimed: "3500", want: "residual"},
		// T2.2 (vòng 2 + soát lần ba): con số nhắc lại chỉ là cùng claim khi liền kề; kiểm chủ thể ở cả phạm vi câu lẫn mệnh đề hỏi;
		// lượng đơn vị suy từ thuật ngữ mà có dấu chấm / phẩy là mơ hồ — trừ khi nguyên văn có chữ ("3,85k")
		{q: "EX-03 hôm nay được cấp 3850 L, dự án A 3850 L, đúng không?", claimed: "3850", want: "value"},
		{q: "EX-03 hôm nay được cấp 3850 L, dự án A đúng không?", claimed: "3850", want: "subject"},
		{q: "EX-03 hôm nay được cấp 3850 L. Dự án A được cấp 3850 L, đúng không?", claimed: "3850", want: "subject"},
		{q: "EX-03 hôm nay được cấp 3850 L. Dự án A 5050 L, đúng không?", claimed: "5050", want: "subject"},
		{q: "Lu 02 hôm nay được cấp dầu 1,2 đúng không?", claimed: "1.2", want: "value"},
		{q: "EX-03 hôm nay được cấp dầu 3,85k đúng không?", claimed: "3850"},
		{q: "EX-03 hôm nay được cấp dầu 3,85K đúng không?", claimed: "3850"},
		{q: "EX-03 hôm nay được cấp dầu 3,5k35 đúng không?", claimed: "3850", want: "value"},
		// E3: sau con số xác nhận chỉ được nhắc lại chính chủ thể
		{q: "EX-03 hôm nay được cấp 3850 L (3850 L của dự án A) đúng không?", claimed: "3850", want: "subject"},
		{q: "EX-03 hôm nay được cấp 3850 L: dự án A đúng không?", claimed: "3850", want: "subject"},
	} {
		in := rt.Normalizer.Normalize(c.q, normalization.Options{Now: now})
		res, err := entityresolution.Resolve(ctx, dir, in)
		if err != nil {
			t.Fatal(err)
		}
		subject := c.subject
		if subject == "" {
			if len(res.Entities) == 0 {
				t.Fatalf("%q: không nhận ra thực thể nào", c.q)
			}
			subject = res.Entities[0].ID
		}
		subj, err := a.subjectOf(ctx, subject)
		if err != nil {
			t.Fatal(err)
		}
		used := c.used
		if used == nil {
			used = []string{"warehouse_slip"}
		}
		got := a.verdictGap(in, mentionsOf(res), contracts.Intent{Name: "fuel_issued"}, def,
			contracts.Value{Kind: contracts.ValueQuantity, Number: c.claimed, Unit: "L"}, subj, used)
		if got != c.want {
			t.Errorf("verdictGap(%q, %s) = %q, muốn %q", c.q, c.claimed, got, c.want)
		}
	}
}
