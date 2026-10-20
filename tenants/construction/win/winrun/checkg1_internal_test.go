package winrun

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/luannt2002/enterprise-ai-systems/core/contracts"
	"github.com/luannt2002/enterprise-ai-systems/data/ingestion"
)

// Bộ kiểm manifest của G1 (docs/04 §6: thiếu hoặc thừa id là KHÔNG ĐẠT, 0 bản ghi rơi im lặng) phải bắt được
// từng kiểu lệch, ở MỌI trường được so (thực thể, chỉ số, nguồn, ngày, giá trị). Kết quả nạp đúng thì 0 vấn đề;
// mỗi kiểu hỏng cố ý thì có vấn đề nêu đúng khoá. Trước đây bộ kiểm chỉ chạy trên dữ liệu đúng (TestWinRatchet),
// nên đột biến làm nó lỏng đi vẫn sống (n36 G1d–G1j).
func TestCheckG1BatDuocLech(t *testing.T) {
	path := filepath.Join(t.TempDir(), "manifest.json")
	if err := os.WriteFile(path, []byte(`{"records":[
 {"tenant":"a","key":"k1","expect":"observation","entity":"EX-03","semantic":"issued_fuel","source":"warehouse_slip","on":"2026-10-19","value_l":"3850"},
 {"tenant":"a","key":"k2","expect":"quarantine","reason":"unknown_entity"},
 {"tenant":"b","key":"kb","expect":"observation","entity":"X","semantic":"issued_fuel","source":"erp","on":"2026-10-19","value_l":"1"}
]}`), 0o600); err != nil {
		t.Fatal(err)
	}
	on, err := time.Parse("2006-01-02", "2026-10-19")
	if err != nil {
		t.Fatal(err)
	}
	obs := func(key, entity, value string) ingestion.Observation {
		return ingestion.Observation{Key: key, Tenant: "a", Entity: entity, Metric: "issued_fuel", Source: "warehouse_slip", On: on,
			Value: contracts.Value{Kind: contracts.ValueQuantity, Number: value, Unit: "L"}}
	}
	with := func(f func(*ingestion.Observation)) ingestion.Result {
		o := obs("k1", "EX-03", "3850")
		f(&o)
		return ingestion.Result{Observations: []ingestion.Observation{o}, Quarantined: []ingestion.Quarantine{{Key: "k2", Tenant: "a", Reason: ingestion.ReasonUnknownEntity}}}
	}
	good := ingestion.Result{
		Observations: []ingestion.Observation{obs("k1", "EX-03", "3850")},
		Quarantined:  []ingestion.Quarantine{{Key: "k2", Tenant: "a", Reason: ingestion.ReasonUnknownEntity}},
	}
	for _, c := range []struct {
		name string
		res  ingestion.Result
		want string // chuỗi phải có trong vấn đề; rỗng = không có vấn đề nào
	}{
		{"đúng", good, ""},
		{"thiếu quan sát", ingestion.Result{Quarantined: good.Quarantined}, "k1: muốn quan sát"},
		{"sai thực thể", ingestion.Result{Observations: []ingestion.Observation{obs("k1", "EX-07", "3850")}, Quarantined: good.Quarantined}, "k1: được EX-07"},
		{"sai giá trị", ingestion.Result{Observations: []ingestion.Observation{obs("k1", "EX-03", "3851")}, Quarantined: good.Quarantined}, "k1: được EX-03"},
		{"sai chỉ số", with(func(o *ingestion.Observation) { o.Metric = "measured_consumption" }), "k1: được EX-03/measured_consumption"},
		{"sai nguồn", with(func(o *ingestion.Observation) { o.Source = "erp" }), "k1: được EX-03/issued_fuel/erp"},
		{"sai ngày", with(func(o *ingestion.Observation) { o.On = on.AddDate(0, 0, 1) }), "k1: được EX-03/issued_fuel/warehouse_slip/2026-10-20"},
		{"sai lý do cách ly", ingestion.Result{Observations: good.Observations, Quarantined: []ingestion.Quarantine{{Key: "k2", Tenant: "a", Reason: ingestion.ReasonUnparseable}}}, "k2: muốn cách ly unknown_entity"},
		{"quan sát THỪA", ingestion.Result{Observations: append(append([]ingestion.Observation(nil), good.Observations...), obs("k9", "EX-03", "1")), Quarantined: good.Quarantined}, "k9: quan sát THỪA"},
		{"cách ly THỪA", ingestion.Result{Observations: good.Observations, Quarantined: append(append([]ingestion.Quarantine(nil), good.Quarantined...), ingestion.Quarantine{Key: "k8", Tenant: "a", Reason: ingestion.ReasonUnparseable})}, "k8: cách ly THỪA"},
	} {
		total, problems, err := checkG1(path, "a", c.res)
		if err != nil {
			t.Fatalf("%s: %v", c.name, err)
		}
		if total != 2 {
			t.Errorf("%s: total = %d, muốn 2 (chỉ bản ghi của tenant a)", c.name, total)
		}
		switch {
		case c.want == "" && len(problems) != 0:
			t.Errorf("%s: muốn 0 vấn đề, được %v", c.name, problems)
		case c.want != "" && !strings.Contains(strings.Join(problems, "\n"), c.want):
			t.Errorf("%s: muốn vấn đề chứa %q, được %v", c.name, c.want, problems)
		}
	}
}
