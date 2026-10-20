package sourceoftruth

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/luannt2002/enterprise-ai-systems/core/contracts"
)

// Dữ liệu test (không phải luật của core): bốn chỉ số nhiên liệu KHÁC nhau của một máy.
var testMetrics = []MetricDef{
	{ID: "reported_fuel", Label: "dầu báo cáo", Unit: "L", Sources: []SourceRank{{Type: "site_report_form", Authority: 1}, {Type: "chat_message", Authority: 2}}},
	{ID: "issued_fuel", Label: "dầu kho xuất", Unit: "L", Sources: []SourceRank{{Type: "warehouse_slip", Authority: 1}, {Type: "erp", Authority: 2}, {Type: "excel_summary", Authority: 2}}},
	{ID: "measured_consumption", Label: "dầu cảm biến đo", Unit: "L", Sources: []SourceRank{{Type: "fuel_sensor", Authority: 1}}},
	{ID: "billed_fuel", Label: "dầu hoá đơn", Unit: "L", Sources: []SourceRank{{Type: "invoice", Authority: 1}}},
}

var testDerived = []DerivedDef{
	{ID: "issue_vs_measured", Label: "chênh kho xuất − cảm biến", Op: "subtract", Left: "issued_fuel", Right: "measured_consumption"},
}

var d19 = time.Date(2026, 10, 19, 0, 0, 0, 0, time.UTC)

func lit(n string) contracts.Value {
	return contracts.Value{Kind: contracts.ValueQuantity, Number: n, Unit: "L"}
}

func fact(id, metric, entity string, v contracts.Value, src string, hour int) Fact {
	return Fact{ID: id, Metric: metric, EntityID: entity, OccurredOn: d19, Value: v,
		Source: Observation{ID: "o-" + id, SourceType: src, SourceRef: strings.ToUpper(id), ObservedAt: d19.Add(time.Duration(hour) * time.Hour)}}
}

func catalog(t *testing.T) *Catalog {
	t.Helper()
	c, err := Compile(testMetrics, testDerived, func(u string) bool { return u == "L" })
	if err != nil {
		t.Fatal(err)
	}
	return c
}

func query(t *testing.T, c *Catalog, s Store, metric string, entities ...string) Result {
	t.Helper()
	r, err := c.Query(context.Background(), s, MetricQuery{Metric: metric, EntityIDs: entities, From: d19, To: d19})
	if err != nil {
		t.Fatal(err)
	}
	return r
}

func TestCungNguonCungNgayLaCongDon(t *testing.T) {
	// "máy 03 lấy 2k sáng + 2k chiều" → hai lần báo, không phải hai phiên bản của một số.
	s := MemoryStore{
		fact("m1", "reported_fuel", "EX-03", lit("2000"), "chat_message", 7),
		fact("m2", "reported_fuel", "EX-03", lit("2000"), "chat_message", 15),
	}
	r := query(t, catalog(t), s, "reported_fuel")
	if r.Total == nil || r.Total.Number != "4000" || len(r.Conflicts) != 0 {
		t.Fatalf("2000 + 2000 phải ra 4000 không xung đột: %+v", r)
	}
}

func TestKhacNguonCungChiSoChonTheoThamQuyen(t *testing.T) {
	s := MemoryStore{
		fact("slip", "issued_fuel", "EX-03", lit("3850"), "warehouse_slip", 8),
		fact("erp", "issued_fuel", "EX-03", lit("3900"), "erp", 18), // MỚI hơn nhưng thẩm quyền thấp hơn
	}
	r := query(t, catalog(t), s, "issued_fuel")
	if r.Total == nil || r.Total.Number != "3850" {
		t.Fatalf("phiếu kho (thẩm quyền 1) phải thắng ERP dù ERP mới hơn: %+v", r.Total)
	}
	if len(r.Conflicts) != 1 || r.Conflicts[0].NeedsReview || r.Conflicts[0].ChosenID != "fact:slip" {
		t.Fatalf("phải ghi đúng 1 xung đột đã giải theo thẩm quyền, chọn fact:slip: %+v", r.Conflicts)
	}
}

func TestBonChiSoKhacNhauKhongDeNhau(t *testing.T) {
	s := MemoryStore{
		fact("r", "reported_fuel", "EX-03", lit("4000"), "chat_message", 17),
		fact("i", "issued_fuel", "EX-03", lit("3850"), "warehouse_slip", 8),
		fact("m", "measured_consumption", "EX-03", lit("3782"), "fuel_sensor", 23),
		fact("b", "billed_fuel", "EX-03", lit("4100"), "invoice", 9),
	}
	c := catalog(t)
	want := map[string]string{"reported_fuel": "4000", "issued_fuel": "3850", "measured_consumption": "3782", "billed_fuel": "4100"}
	for metric, n := range want {
		r := query(t, c, s, metric)
		if r.Total == nil || r.Total.Number != n || len(r.Conflicts) != 0 {
			t.Errorf("%s = %+v, muốn %s và 0 xung đột", metric, r.Total, n)
		}
	}
	d, err := c.Derive(context.Background(), s, "issue_vs_measured", nil, d19, d19)
	if err != nil {
		t.Fatal(err)
	}
	if d.Total == nil || d.Total.Number != "68" {
		t.Fatalf("chênh lệch 3850 − 3782 phải là 68: %+v", d.Total)
	}
	last := d.Evidence[len(d.Evidence)-1]
	if last.Kind != contracts.EvidenceCalculation || !strings.Contains(last.Formula, "calc:issued_fuel") || !strings.Contains(last.Formula, "calc:measured_consumption") {
		t.Fatalf("fact dẫn xuất phải ghi công thức trỏ về hai vế: %+v", last)
	}
}

// TestThamQuyenKhongPhaiTenHayDoMoi dựng 3 nguồn sao cho xếp theo tên (tăng hoặc giảm) và
// xếp theo độ mới đều chọn SAI; chỉ xếp theo thẩm quyền mới chọn đúng m_slip.
func TestThamQuyenKhongPhaiTenHayDoMoi(t *testing.T) {
	c, err := Compile([]MetricDef{{ID: "issued", Label: "x", Unit: "L", Tolerance: "10",
		Sources: []SourceRank{{Type: "m_slip", Authority: 1}, {Type: "a_erp", Authority: 2}, {Type: "z_excel", Authority: 3}}}}, nil, func(string) bool { return true })
	if err != nil {
		t.Fatal(err)
	}
	s := MemoryStore{
		fact("old", "issued", "EX-03", lit("3850"), "m_slip", 6),   // thẩm quyền 1, CŨ nhất
		fact("mid", "issued", "EX-03", lit("3900"), "a_erp", 12),   // tên đứng đầu bảng chữ cái
		fact("new", "issued", "EX-03", lit("4000"), "z_excel", 23), // MỚI nhất, tên đứng cuối
	}
	r := query(t, c, s, "issued")
	if r.Total == nil || r.Total.Number != "3850" {
		t.Fatalf("phải chọn nguồn thẩm quyền 1 (m_slip=3850), được %+v", r.Total)
	}
	if len(r.Conflicts) != 2 {
		t.Fatalf("hai nguồn còn lại đều lệch quá dung sai 10 → 2 xung đột, được %d", len(r.Conflicts))
	}
}

func TestNgangThamQuyenThiKhongTuChon(t *testing.T) {
	s := MemoryStore{
		fact("erp", "issued_fuel", "EX-03", lit("3900"), "erp", 8),
		fact("xls", "issued_fuel", "EX-03", lit("4000"), "excel_summary", 9),
	}
	r := query(t, catalog(t), s, "issued_fuel")
	if r.Total != nil || !r.NeedsReview || len(r.Conflicts) != 1 || !r.Conflicts[0].NeedsReview {
		t.Fatalf("hai nguồn ngang thẩm quyền lệch nhau: không được trả tổng, phải cần người duyệt: %+v", r)
	}
}

func TestUocLuongDuocDanhDau(t *testing.T) {
	v := lit("4000")
	v.Approx = true // "khoảng 4k"
	r := query(t, catalog(t), MemoryStore{fact("a", "reported_fuel", "EX-03", v, "chat_message", 17)}, "reported_fuel")
	if r.Total == nil || !r.Approx || !r.Total.Approx {
		t.Fatalf("tổng có ước lượng phải mang cờ ước lượng: %+v", r)
	}
}

func TestLocTheoThucTheDuocPhep(t *testing.T) {
	s := MemoryStore{
		fact("a", "reported_fuel", "EX-03", lit("4000"), "chat_message", 8),
		fact("b", "reported_fuel", "EX-07", lit("9999"), "chat_message", 8),
	}
	r := query(t, catalog(t), s, "reported_fuel", "EX-03")
	if r.Total == nil || r.Total.Number != "4000" {
		t.Fatalf("thực thể không được phép không được lọt vào tổng: %+v", r.Total)
	}
	for _, e := range r.Evidence {
		if strings.Contains(e.Text, "EX-07") {
			t.Fatalf("bằng chứng của thực thể không được phép bị lộ: %s", e.Text)
		}
	}
}

func TestChiSoLaVaDonViSai(t *testing.T) {
	c := catalog(t)
	if _, err := c.Query(context.Background(), MemoryStore{}, MetricQuery{Metric: "nope"}); contracts.FaultOf(err) != contracts.FaultCode {
		t.Fatalf("chỉ số chưa khai báo là lỗi mã: %v", err)
	}
	bad := MemoryStore{fact("x", "reported_fuel", "EX-03", contracts.Value{Kind: contracts.ValueQuantity, Number: "4", Unit: "m3"}, "chat_message", 8)}
	if _, err := c.Query(context.Background(), bad, MetricQuery{Metric: "reported_fuel", From: d19, To: d19}); contracts.FaultOf(err) != contracts.FaultData {
		t.Fatalf("fact sai đơn vị là lỗi dữ liệu: %v", err)
	}
}

func TestCompileChanCatalogHong(t *testing.T) {
	known := func(u string) bool { return u == "L" || u == "km" }
	cases := map[string]struct {
		m []MetricDef
		d []DerivedDef
	}{
		"đơn vị chưa khai báo": {m: []MetricDef{{ID: "a", Unit: "m3", Sources: []SourceRank{{Type: "x", Authority: 1}}}}},
		"không có nguồn":       {m: []MetricDef{{ID: "a", Unit: "L"}}},
		"nguồn trùng":          {m: []MetricDef{{ID: "a", Unit: "L", Sources: []SourceRank{{Type: "x", Authority: 1}, {Type: "x", Authority: 2}}}}},
		"dẫn xuất khác đơn vị": {
			m: []MetricDef{{ID: "a", Unit: "L", Sources: []SourceRank{{Type: "x", Authority: 1}}}, {ID: "b", Unit: "km", Sources: []SourceRank{{Type: "x", Authority: 1}}}},
			d: []DerivedDef{{ID: "c", Op: "subtract", Left: "a", Right: "b"}},
		},
	}
	for name, cs := range cases {
		if _, err := Compile(cs.m, cs.d, known); err == nil {
			t.Errorf("%s: Compile phải lỗi", name)
		}
	}
}

// So hai chỉ số: hiệu theo thứ tự trái − phải, mỗi vế theo thẩm quyền của chính nó; khác đơn vị thì
// không so (lỗi mã — pack khai sai), không đổi đơn vị ngầm.
func TestCompareHaiChiSo(t *testing.T) {
	s := MemoryStore{
		fact("r1", "reported_fuel", "EX-03", lit("2000"), "chat_message", 7),
		fact("r2", "reported_fuel", "EX-03", lit("2000"), "chat_message", 9),
		fact("i", "issued_fuel", "EX-03", lit("3850"), "warehouse_slip", 8),
		fact("e", "issued_fuel", "EX-03", lit("3900"), "erp", 8),
	}
	c := catalog(t)
	r, err := c.Compare(context.Background(), s, "reported_fuel", "issued_fuel", nil, d19, d19)
	if err != nil || r.Total == nil || r.Total.Number != "150" || r.Metric != CompareID("reported_fuel", "issued_fuel") {
		t.Fatalf("đội báo − kho xuất = 4000 − 3850 = 150: được %+v (lỗi %v)", r.Total, err)
	}
	last := r.Evidence[len(r.Evidence)-1]
	if last.ID != "calc:"+r.Metric || last.Formula != "calc:reported_fuel − calc:issued_fuel" || last.Period != "2026-10-19" {
		t.Errorf("bằng chứng phép so phải ghi công thức và kỳ: %+v", last)
	}
	if r, _ := c.Compare(context.Background(), s, "issued_fuel", "reported_fuel", nil, d19, d19); r.Total == nil || r.Total.Number != "-150" {
		t.Errorf("đảo vế phải ra -150: %+v", r.Total)
	}
	withKm, err := Compile(append(append([]MetricDef(nil), testMetrics...), MetricDef{ID: "distance", Label: "quãng đường", Unit: "km",
		Sources: []SourceRank{{Type: "gps", Authority: 1}}}), testDerived, func(u string) bool { return u == "L" || u == "km" })
	if err != nil {
		t.Fatal(err)
	}
	if _, err := withKm.Compare(context.Background(), s, "issued_fuel", "distance", nil, d19, d19); contracts.FaultOf(err) != contracts.FaultCode {
		t.Errorf("so khác đơn vị: muốn lỗi %s, được %v", contracts.FaultCode, err)
	}
}
