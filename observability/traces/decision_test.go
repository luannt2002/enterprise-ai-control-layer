package traces

import (
	"slices"
	"testing"
)

// Diff chỉ ra ĐÚNG chặng bị sửa — sửa một chặng nào cũng phải lộ, không chặng nào bị bỏ qua.
func TestDiffMoiChang(t *testing.T) {
	base := Decision{
		Normalized: []string{"code:EX-03=EX-03"}, Entities: []string{"EX-03"}, Intent: "fuel_issued", Slots: []string{"subject=EX-03"},
		Route: "SQL_QUERY", Outcome: "RESOLVED", Evidence: []string{"fact:R-004"}, Verdict: "true|accept", Answer: "3.850 L",
	}
	if d := Diff(base, base); len(d) != 0 {
		t.Fatalf("hai bản giống nhau: Diff = %v, muốn rỗng", d)
	}
	for stage, edit := range map[string]func(*Decision){
		"normalize": func(d *Decision) { d.Normalized = append(d.Normalized, "x") },
		"resolve":   func(d *Decision) { d.Entities = []string{"EX-07"} },
		"intent":    func(d *Decision) { d.Slots = []string{"subject=EX-07"} },
		"route":     func(d *Decision) { d.Route = "RAG" },
		"outcome":   func(d *Decision) { d.Outcome = "SYSTEM_FAILURE" },
		"evidence":  func(d *Decision) { d.Evidence = []string{"fact:R-005"} },
		"verdict":   func(d *Decision) { d.Verdict = "false|fallback|number_without_evidence" },
		"answer":    func(d *Decision) { d.Answer = "3.900 L" },
	} {
		got := base
		got.Normalized, got.Entities, got.Slots, got.Evidence = slices.Clone(base.Normalized), slices.Clone(base.Entities), slices.Clone(base.Slots), slices.Clone(base.Evidence)
		edit(&got)
		if d := Diff(base, got); !slices.Equal(d, []string{stage}) {
			t.Errorf("sửa chặng %s: Diff = %v, muốn [%s]", stage, d, stage)
		}
	}
}
