package validation

import (
	"slices"
	"testing"

	"github.com/luannt2002/enterprise-ai-systems/core/contracts"
)

// Bằng chứng mẫu — DỮ LIỆU test, không phải luật của core.
func ev(id, subject, metric, period, source, number, unit string) contracts.Evidence {
	return contracts.Evidence{ID: id, Subject: subject, Metric: metric, Period: period,
		Source: contracts.SourceRef{SourceType: source}, Values: []contracts.Value{{Kind: contracts.ValueQuantity, Number: number, Unit: unit}}}
}

var bindingEvidence = []contracts.Evidence{
	ev("fact:R-004", "EX-03", "issued_fuel", "2026-10-19", "warehouse_slip", "3850", "L"),
	ev("fact:R-005", "EX-03", "issued_fuel", "2026-10-19", "erp", "3900", "L"),
	ev("fact:R-001", "EX-03", "reported_fuel", "2026-10-19", "chat_message", "2000", "L"),
	ev("fact:R-002", "EX-03", "reported_fuel", "2026-10-19", "chat_message", "2000", "L"),
	ev("fact:R-020", "EX-07", "issued_fuel", "2026-10-19", "warehouse_slip", "911", "L"),
	ev("fact:R-030", "EX-03", "issued_fuel", "2026-10-18", "warehouse_slip", "3700", "L"),
	ev("calc:issued_fuel", "EX-03", "issued_fuel", "2026-10-19", "calculation", "3850", "L"),
	ev("calc:reported_fuel", "EX-03", "reported_fuel", "2026-10-19", "calculation", "4000", "L"),
	ev("calc:issue_vs_measured", "EX-03", "issue_vs_measured", "2026-10-19", "calculation", "68", "L"),
	ev("calc:issue_vs_measured:neg", "EX-07", "issue_vs_measured", "2026-10-19", "calculation", "-68", "L"),
	ev("calc:issued_fuel:month", "PRJ-A", "issued_fuel", "2026-10-01/2026-10-31", "calculation", "15200.5", "L"),
	ev("question", "EX-03", "issued_fuel", "2026-10-19", "question", "4000", "L"),
	ev("count:conflicts", "PRJ-A", "conflict_count", "2026-10-01/2026-10-31", "calculation", "2", ""),
	ev("calc:billed_fuel", "EX-03", "billed_fuel", "2026-10-19", "calculation", "0.5", "L"),
}

func claim(subject, metric, period, number, unit, source string, ids ...string) contracts.Claim {
	return contracts.Claim{Subject: subject, Metric: metric, Period: period, Source: source, EvidenceIDs: ids,
		Value: &contracts.Value{Kind: contracts.ValueQuantity, Number: number, Unit: unit}}
}

type bindCase struct {
	text   string
	claims []contracts.Claim
}

var (
	mainClaim     = claim("EX-03", "issued_fuel", "2026-10-19", "3850", "L", "calculation", "calc:issued_fuel")
	conflictClaim = claim("EX-03", "issued_fuel", "2026-10-19", "3900", "L", "erp", "fact:R-005")
	derivedClaim  = claim("EX-03", "issue_vs_measured", "2026-10-19", "68", "L", "calculation", "calc:issue_vs_measured")
	sumClaim      = claim("EX-03", "reported_fuel", "2026-10-19", "4000", "L", "chat_message", "fact:R-001", "fact:R-002")
)

// validCases: câu đúng — 0 lần chặn oan.
var validCases = map[string]bindCase{
	"tổng một ngày":         {"Dầu kho xuất của Máy xúc EX-03 ngày 19/10/2026: 3.850 L.", []contracts.Claim{mainClaim}},
	"có ghi chú xung đột":   {"Dầu kho xuất của Máy xúc EX-03 ngày 19/10/2026: 3.850 L. Nguồn khác lệch số: ERP 3.900 L.", []contracts.Claim{mainClaim, conflictClaim}},
	"phép tính":             {"Chênh lệch của Máy xúc EX-03 ngày 19/10/2026: 68 L.", []contracts.Claim{derivedClaim}},
	"phép tính âm":          {"Chênh lệch của Lu rung EX-07 ngày 19/10/2026: -68 L.", []contracts.Claim{claim("EX-07", "issue_vs_measured", "2026-10-19", "-68", "L", "calculation", "calc:issue_vs_measured:neg")}},
	"cộng dồn hai tin nhắn": {"Tin nhắn hiện trường của EX-03 ngày 19/10/2026: 4.000 L.", []contracts.Claim{sumClaim}},
	"xác nhận sai":          {"Không. Dầu kho xuất của EX-03 ngày 19/10/2026 là 3.850 L, không phải 4.000 L.", []contracts.Claim{mainClaim, claim("EX-03", "issued_fuel", "2026-10-19", "4000", "L", "question", "question")}},
	"các số khác cùng kỳ":   {"Đúng. Dầu kho xuất 3.850 L. Các số khác cùng kỳ: dầu đội báo 4.000 L.", []contracts.Claim{mainClaim, claim("EX-03", "reported_fuel", "2026-10-19", "4000", "L", "calculation", "calc:reported_fuel")}},
	"khoảng ngày, số lẻ":    {"Dầu kho xuất của Cầu A từ 01/10/2026 đến 31/10/2026: 15.200,5 L.", []contracts.Claim{claim("PRJ-A", "issued_fuel", "2026-10-01/2026-10-31", "15200.5", "L", "calculation", "calc:issued_fuel:month")}},
	"đếm chỗ lệch":          {"Cầu A từ 01/10/2026 đến 31/10/2026: 15.200,5 L. Nguồn khác lệch số ở 3 chỗ và 2 chỗ khác.", nil},
	"thực thể khác":         {"Dầu kho xuất của Lu rung EX-07 ngày 19/10/2026: 911 L.", []contracts.Claim{claim("EX-07", "issued_fuel", "2026-10-19", "911", "L", "warehouse_slip", "fact:R-020")}},
	"số nhỏ hơn 1":          {"Dầu hoá đơn của EX-03 ngày 19/10/2026: 0,5 L.", []contracts.Claim{claim("EX-03", "billed_fuel", "2026-10-19", "0.5", "L", "calculation", "calc:billed_fuel")}},
	"không có số":           {"Bạn không có quyền xem dữ liệu này.", nil},
	"mã thực thể có chữ số": {"Phiếu R-004 của EX-03 ngày 19/10/2026: 3.850 L.", []contracts.Claim{mainClaim}},
}

func init() {
	// "đếm chỗ lệch": số 3 là một claim đếm thật — viết lại cho đúng (3 chỗ kê, 2 chỗ còn lại).
	c := validCases["đếm chỗ lệch"]
	c.text = "Dầu kho xuất của Cầu A từ 01/10/2026 đến 31/10/2026: 15.200,5 L; còn 2 chỗ lệch khác."
	c.claims = []contracts.Claim{
		claim("PRJ-A", "issued_fuel", "2026-10-01/2026-10-31", "15200.5", "L", "calculation", "calc:issued_fuel:month"),
		{Subject: "PRJ-A", Metric: "conflict_count", Period: "2026-10-01/2026-10-31", Source: "calculation", EvidenceIDs: []string{"count:conflicts"},
			Value: &contracts.Value{Number: "2"}},
	}
	validCases["đếm chỗ lệch"] = c
}

func TestBindClaimsKhongChanOan(t *testing.T) {
	if len(validCases) < 10 {
		t.Fatalf("cần ≥ 10 ca hợp lệ, có %d", len(validCases))
	}
	for name, c := range validCases {
		if v := BindClaims(c.text, []string{"EX-03", "EX-07"}, c.claims, bindingEvidence); len(v) != 0 {
			t.Errorf("%s: chặn oan %+v", name, v)
		}
	}
}

// Bảng "sai đúng một chiều": mỗi chiều ≥ 3 ca, mỗi ca chỉ sai đúng một chiều — 100% phải bị chặn,
// và lý do phải nêu đúng chiều sai.
func TestBindClaimsSaiDungMotChieu(t *testing.T) {
	mod := func(c contracts.Claim, f func(*contracts.Claim)) []contracts.Claim {
		v := *c.Value
		c.Value = &v
		c.EvidenceIDs = slices.Clone(c.EvidenceIDs)
		f(&c)
		return []contracts.Claim{c}
	}
	table := map[string][]bindCase{
		"chủ thể": {
			{"EX-07 ngày 19/10/2026: 3.850 L.", mod(mainClaim, func(c *contracts.Claim) { c.Subject = "EX-07" })},
			{"Cầu A ngày 19/10/2026: 3.900 L.", mod(conflictClaim, func(c *contracts.Claim) { c.Subject = "PRJ-A" })},
			{"EX-07 ngày 19/10/2026: 68 L.", mod(derivedClaim, func(c *contracts.Claim) { c.Subject = "EX-07" })},
		},
		"chỉ số": {
			{"Dầu đội báo ngày 19/10/2026: 3.850 L.", mod(mainClaim, func(c *contracts.Claim) { c.Metric = "reported_fuel" })},
			{"Hoá đơn ngày 19/10/2026: 3.900 L.", mod(conflictClaim, func(c *contracts.Claim) { c.Metric = "billed_fuel" })},
			{"Dầu kho xuất ngày 19/10/2026: 4.000 L.", mod(sumClaim, func(c *contracts.Claim) { c.Metric = "issued_fuel" })},
		},
		"kỳ": {
			{"ngày 18/10/2026: 3.850 L.", mod(mainClaim, func(c *contracts.Claim) { c.Period = "2026-10-18" })},
			{"ngày 20/10/2026: 3.900 L.", mod(conflictClaim, func(c *contracts.Claim) { c.Period = "2026-10-20" })},
			{"từ 01/10/2026 đến 31/10/2026: 68 L.", mod(derivedClaim, func(c *contracts.Claim) { c.Period = "2026-10-01/2026-10-31" })},
		},
		"giá trị": {
			{"ngày 19/10/2026: 3.900 L.", mod(mainClaim, func(c *contracts.Claim) { c.Value.Number = "3900" })},
			{"ngày 19/10/2026: 4.000 L.", mod(conflictClaim, func(c *contracts.Claim) { c.Value.Number = "4000" })},
			{"ngày 19/10/2026: 2.000 L.", mod(sumClaim, func(c *contracts.Claim) { c.Value.Number = "2000" })},
		},
		"đơn vị": {
			{"ngày 19/10/2026: 3.850 m³.", mod(mainClaim, func(c *contracts.Claim) { c.Value.Unit = "m3" })},
			{"ngày 19/10/2026: 3.900 kg.", mod(conflictClaim, func(c *contracts.Claim) { c.Value.Unit = "kg" })},
			{"ngày 19/10/2026: 68 m³.", mod(derivedClaim, func(c *contracts.Claim) { c.Value.Unit = "m3" })},
		},
		"nguồn": {
			{"ngày 19/10/2026: 3.850 L.", mod(mainClaim, func(c *contracts.Claim) { c.Source = "warehouse_slip" })},
			{"ERP ngày 19/10/2026: 3.900 L.", mod(conflictClaim, func(c *contracts.Claim) { c.Source = "warehouse_slip" })},
			{"ngày 19/10/2026: 4.000 L.", mod(sumClaim, func(c *contracts.Claim) { c.Source = "site_report_form" })},
		},
	}
	for dim, cases := range table {
		if len(cases) < 3 {
			t.Errorf("chiều %s: cần ≥ 3 ca, có %d", dim, len(cases))
		}
		for i, c := range cases {
			v := BindClaims(c.text, []string{"EX-03", "EX-07"}, c.claims, bindingEvidence)
			if len(v) == 0 {
				t.Errorf("chiều %s, ca %d: KHÔNG bị chặn — %q", dim, i, c.text)
				continue
			}
			if !slices.ContainsFunc(v, func(x contracts.Violation) bool { return x.Code == "CLAIM_UNBOUND" && containsWord(x.Detail, dim) }) {
				t.Errorf("chiều %s, ca %d: bị chặn nhưng không nêu đúng chiều: %+v", dim, i, v)
			}
		}
	}
}

// Số hoặc ngày "trôi" khỏi claim (mẫu câu chèn số, model bịa ngày) bị chặn dù claim đúng.
func TestBindClaimsSoVaNgayTroi(t *testing.T) {
	for name, text := range map[string]string{
		"số không thuộc claim":   "Dầu kho xuất của EX-03 ngày 19/10/2026: 3.850 L, định mức 5.000 L.",
		"ngày không thuộc claim": "Dầu kho xuất của EX-03 ngày 19/10/2026: 3.850 L (cập nhật 21/10/2026).",
	} {
		if v := BindClaims(text, []string{"EX-03"}, []contracts.Claim{mainClaim}, bindingEvidence); len(v) == 0 {
			t.Errorf("%s: không bị chặn — %q", name, text)
		}
	}
}

func TestCountNumbers(t *testing.T) {
	nums, dates := CountNumbers("EX-03 ngày 19/10/2026: 3.850 L; R-004 ghi 1.602.000 đồng, tỉ lệ 0,35, chênh -68 L, PRJ-F01.")
	if want := []string{"3850", "1602000", "0.35", "-68"}; !slices.Equal(nums, want) {
		t.Errorf("số = %v, muốn %v", nums, want)
	}
	if want := []string{"2026-10-19"}; !slices.Equal(dates, want) {
		t.Errorf("ngày = %v, muốn %v", dates, want)
	}
	if _, d := CountNumbers("ngày 1/9/2026"); !slices.Equal(d, []string{"2026-09-01"}) {
		t.Errorf("ngày một chữ số: %v", d)
	}
}

func containsWord(s, w string) bool {
	return len(w) > 0 && len(s) >= len(w) && (s == w || indexOf(s, w) >= 0)
}

func indexOf(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}
