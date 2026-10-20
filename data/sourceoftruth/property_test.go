package sourceoftruth

import (
	"context"
	"fmt"
	"math/rand/v2"
	"strings"
	"testing"

	"github.com/luannt2002/enterprise-ai-systems/core/decimal"
)

// Thuộc tính G6 trên dữ liệu ngẫu nhiên có seed (tái lập được):
//  1. tổng của chỉ số có min ≥ 0 không bao giờ âm;
//  2. fact dẫn xuất ĐƯỢC âm (không bị kẹp về 0), và giá trị = a − b với a, b là tổng hai vế lấy
//     từ chính bằng chứng "calc:" đi kèm — tức |giá trị| = |a − b|.
func TestThuocTinhTongKhongAmVaDanXuat(t *testing.T) {
	metrics := append([]MetricDef(nil), testMetrics...)
	for i := range metrics {
		metrics[i].Min = "0"
	}
	c, err := Compile(metrics, testDerived, func(u string) bool { return u == "L" })
	if err != nil {
		t.Fatal(err)
	}
	rng := rand.New(rand.NewPCG(42, 2026))
	sources := map[string][]string{"issued_fuel": {"warehouse_slip", "erp", "excel_summary"}, "measured_consumption": {"fuel_sensor"}}
	value := func() string {
		n := fmt.Sprint(rng.IntN(10000))
		if rng.IntN(4) == 0 {
			n += ".5"
		}
		return n
	}
	negatives, derived := 0, 0
	for round := range 400 {
		var s MemoryStore
		for metric, srcs := range sources {
			for k := range rng.IntN(4) {
				f := fact(fmt.Sprintf("%s-%d-%d", metric, round, k), metric, []string{"EX-03", "EX-07"}[rng.IntN(2)], lit(value()), srcs[rng.IntN(len(srcs))], k)
				s = append(s, f)
			}
		}
		for metric := range sources {
			r := query(t, c, s, metric)
			if r.Total == nil {
				continue
			}
			if cmp, err := decimal.Cmp(r.Total.Number, "0"); err != nil || cmp < 0 {
				t.Fatalf("vòng %d: tổng %s = %s âm (min = 0)", round, metric, r.Total.Number)
			}
		}
		d, err := c.Derive(context.Background(), s, "issue_vs_measured", nil, d19, d19)
		if err != nil {
			t.Fatal(err)
		}
		if d.Total == nil {
			continue
		}
		derived++
		var a, b string
		for _, e := range d.Evidence {
			switch e.ID {
			case "calc:issued_fuel":
				a = e.Values[0].Number
			case "calc:measured_consumption":
				b = e.Values[0].Number
			}
		}
		want, err := decimal.Sub(a, b)
		if err != nil || a == "" || b == "" {
			t.Fatalf("vòng %d: không lấy được hai vế từ bằng chứng (a=%q b=%q, lỗi %v)", round, a, b, err)
		}
		if cmp, _ := decimal.Cmp(d.Total.Number, want); cmp != 0 {
			t.Fatalf("vòng %d: dẫn xuất = %s, muốn %s − %s = %s", round, d.Total.Number, a, b, want)
		}
		if strings.HasPrefix(d.Total.Number, "-") {
			negatives++
		}
	}
	// đối chứng: bộ sinh thật sự tạo ra ca dẫn xuất âm — thuộc tính "được âm" không đúng một cách rỗng
	if derived == 0 || negatives == 0 {
		t.Fatalf("bộ sinh không phủ: %d ca dẫn xuất, %d ca âm", derived, negatives)
	}
	t.Logf("%d ca dẫn xuất, %d ca âm", derived, negatives)
}
