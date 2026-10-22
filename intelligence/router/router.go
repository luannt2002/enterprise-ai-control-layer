// Package router là chặng S5: ý định → kế hoạch route.
//
// Router không đoán. Mơ hồ hoặc thiếu khe → CLARIFICATION; không có quyền → REJECT;
// không ý định nào khớp → route do pack chọn cho trường hợp này. Quyền ở đây là quyền
// của PHẦN MỀM (vai nghiệp vụ → capability, phạm vi thực thể), không phải "vai" trong prompt.
package router

import (
	"fmt"
	"strings"

	"github.com/luannt2002/enterprise-ai-control-layer/core/contracts"
	"github.com/luannt2002/enterprise-ai-control-layer/intelligence/intent"
)

// RoleDef là một vai nghiệp vụ trong pack (policy.yaml).
type RoleDef struct {
	Capabilities []string `yaml:"capabilities"`
	// Scope: "all" = xem mọi thực thể; "attribute" = chỉ thực thể có khoá nằm trong thuộc
	// tính entity_scope của principal. Không có giá trị mặc định nào mở toang.
	Scope string `yaml:"scope"`
}

// Policy là chính sách vai → capability + phạm vi dữ liệu.
type Policy struct {
	Roles map[string]RoleDef `yaml:"roles"`
}

// Validate kiểm chính sách.
func (p Policy) Validate() error {
	var errs []string
	if len(p.Roles) == 0 {
		errs = append(errs, "policy không có vai nào")
	}
	for name, r := range p.Roles {
		if r.Scope != "all" && r.Scope != "attribute" {
			errs = append(errs, fmt.Sprintf("vai %s: scope phải là all hoặc attribute, đang là %q", name, r.Scope))
		}
	}
	if len(errs) > 0 {
		return fmt.Errorf("policy không hợp lệ:\n  - %s", strings.Join(errs, "\n  - "))
	}
	return nil
}

// narrow gỡ bí danh mơ hồ bằng NGỮ CẢNH đã có trong câu và trong danh tính — luật tất định:
//
//	(a) khe thực thể đã điền bằng một DỰ ÁN mà câu còn nhắc một thiết bị mơ hồ ("máy 07 của dự
//	    án A"): giữ ứng viên thuộc dự án đó; đúng 1 thì câu hỏi là về thiết bị ấy, 0 hoặc > 1
//	    thì hỏi lại — không lặng lẽ trả lời cho cả dự án;
//	(b) khe còn thiếu: giữ ứng viên nằm trong phạm vi dữ liệu của người hỏi; đúng 1 thì chọn
//	    (người hỏi không thể muốn thiết bị mình không được xem). Vai phạm vi "all" thì mọi ứng
//	    viên đều trong phạm vi nên vẫn phải hỏi lại.
func narrow(cls intent.Result, def intent.Def, pr contracts.Principal, pol Policy, lineage func(contracts.EntityRef) []string) contracts.Intent {
	it := cls.Intent
	if len(cls.Candidates) == 0 {
		return it
	}
	slot := ""
	for _, sd := range def.Slots {
		if sd.From == "entity" {
			slot = sd.Name
			break
		}
	}
	if slot == "" {
		return it
	}
	var keep []contracts.EntityRef
	if cur, ok := it.Slots[slot]; ok && cur.Entity != nil {
		if cur.Entity.Type != "project" {
			return it
		}
		for _, c := range cls.Candidates {
			for _, k := range lineage(c) {
				if k == cur.Entity.Key {
					keep = append(keep, c)
					break
				}
			}
		}
		if len(keep) != 1 {
			delete(it.Slots, slot)
			it.Missing = append(it.Missing, slot)
			return it
		}
	} else {
		for _, c := range cls.Candidates {
			if pol.InScope(pr, lineage(c)) {
				keep = append(keep, c)
			}
		}
		if len(keep) != 1 {
			return it
		}
		var rest []string
		for _, m := range it.Missing {
			if m != slot {
				rest = append(rest, m)
			}
		}
		it.Missing = rest
	}
	slots := make(map[string]contracts.Slot, len(it.Slots)+1)
	for k, v := range it.Slots {
		slots[k] = v
	}
	ref := keep[0]
	slots[slot] = contracts.Slot{Name: slot, Entity: &ref}
	it.Slots = slots
	return it
}

// terminalOutcome là kết quả của route kết thúc ngay ở router; route còn phải thực thi trả rỗng.
func terminalOutcome(r contracts.RouteKind) contracts.Outcome {
	switch r {
	case contracts.RouteClarification:
		return contracts.OutcomeClarificationRequired
	case contracts.RouteReject:
		return contracts.OutcomeOutOfScope
	}
	return ""
}

// Allows báo principal có capability không (qua bất kỳ vai nào của họ). Đóng mặc định:
// capability rỗng không cấp quyền cho ai.
func (p Policy) Allows(pr contracts.Principal, capability string) bool {
	if capability == "" {
		return false
	}
	for _, role := range pr.Roles {
		for _, c := range p.Roles[role].Capabilities {
			if c == capability {
				return true
			}
		}
	}
	return false
}

// InScope báo principal có được xem một thực thể không. lineage là khoá của thực thể và mọi
// tổ tiên của nó: được giao dự án PRJ-A thì xem được máy thuộc PRJ-A.
func (p Policy) InScope(pr contracts.Principal, lineage []string) bool {
	for _, role := range pr.Roles {
		if p.Roles[role].Scope == "all" {
			return true
		}
	}
	for _, k := range strings.Split(pr.Attributes["entity_scope"], ",") {
		k = strings.TrimSpace(k)
		if k == "" {
			continue
		}
		for _, key := range lineage {
			if key == k {
				return true
			}
		}
	}
	return false
}

// Settings là phần cấu hình router của pack (nằm cùng intents.yaml).
type Settings struct {
	UnmatchedRoute    string `yaml:"unmatched_route"`
	UnmatchedTemplate string `yaml:"unmatched_template"`
}

// Route quyết kế hoạch cho một lượt hỏi. lineage trả khoá thực thể + tổ tiên (từ danh mục).
func Route(set *intent.Set, cls intent.Result, in contracts.NormalizedInput, pr contracts.Principal, pol Policy, st Settings,
	lineage func(e contracts.EntityRef) []string) contracts.RoutePlan {
	switch {
	case len(cls.Ambiguous) > 0:
		return contracts.RoutePlan{
			Route: contracts.RouteClarification, Template: "clarify_intent",
			Reason:  "nhiều ý định cùng khớp: " + strings.Join(cls.Ambiguous, ", "),
			Outcome: contracts.OutcomeClarificationRequired,
		}
	case len(cls.NeedTerm) > 0:
		return contracts.RoutePlan{
			Route: contracts.RouteClarification, Template: "clarify_intent",
			Reason:  "khớp từ khoá nhưng thiếu thuật ngữ bắt buộc: " + strings.Join(cls.NeedTerm, ", "),
			Outcome: contracts.OutcomeClarificationRequired,
		}
	case len(cls.NeedMeaning) > 0:
		// thực thể ngoài phạm vi dữ liệu của người hỏi: nghĩa nào cũng bị từ chối, nên từ chối ngay, không hỏi lại
		for _, e := range cls.Mentioned {
			if !pol.InScope(pr, lineage(e)) {
				return contracts.RoutePlan{
					Route: contracts.RouteReject, Template: "forbidden", Outcome: contracts.OutcomePermissionDenied,
					Reason: "thực thể " + e.Key + " ngoài phạm vi dữ liệu của người hỏi",
				}
			}
		}
		return contracts.RoutePlan{
			Route: contracts.RouteClarification, Template: "clarify_intent",
			Reason:  "có thực thể và thuật ngữ nhưng không từ khoá ý định nào khớp: " + strings.Join(cls.NeedMeaning, ", "),
			Outcome: contracts.OutcomeClarificationRequired,
		}
	case cls.Intent.Name == "":
		r := contracts.RouteKind(st.UnmatchedRoute)
		return contracts.RoutePlan{
			Route: r, Template: st.UnmatchedTemplate,
			Reason: "không ý định nào khớp", Outcome: terminalOutcome(r),
		}
	}
	def, _ := set.Def(cls.Intent.Name)
	plan := contracts.RoutePlan{Intent: narrow(cls, def, pr, pol, lineage)}
	cls.Intent = plan.Intent

	// Ý định không khai capability chỉ có thể là route thuần trả lời (intent.Compile chặn
	// mọi trường hợp khác), nên bỏ qua kiểm quyền ở đây là an toàn.
	if def.Capability != "" && !pol.Allows(pr, def.Capability) {
		plan.Route, plan.Template, plan.Outcome = contracts.RouteReject, "forbidden", contracts.OutcomePermissionDenied
		plan.Reason = "thiếu capability " + def.Capability
		return plan
	}
	if cls.Compare != "" { // câu so sánh đọc cả hai chỉ số: cần quyền của cả hai vế
		if cdef, _ := set.Def(cls.Compare); cdef.Capability != "" && !pol.Allows(pr, cdef.Capability) {
			plan.Route, plan.Template, plan.Outcome = contracts.RouteReject, "forbidden", contracts.OutcomePermissionDenied
			plan.Reason = "thiếu capability " + cdef.Capability + " cho vế so sánh"
			return plan
		}
	}
	if cls.Knowledge != "" { // HYBRID cần cả quyền đọc tài liệu
		if kdef, _ := set.Def(cls.Knowledge); kdef.Capability != "" && !pol.Allows(pr, kdef.Capability) {
			plan.Route, plan.Template, plan.Outcome = contracts.RouteReject, "forbidden", contracts.OutcomePermissionDenied
			plan.Reason = "thiếu capability " + kdef.Capability + " cho phần tri thức"
			return plan
		}
	}
	for _, s := range cls.Intent.Slots {
		if s.Entity != nil && !pol.InScope(pr, lineage(*s.Entity)) {
			plan.Route, plan.Template, plan.Outcome = contracts.RouteReject, "forbidden", contracts.OutcomePermissionDenied
			plan.Reason = "thực thể " + s.Entity.Key + " ngoài phạm vi dữ liệu của người hỏi"
			return plan
		}
	}
	if len(cls.Intent.Missing) > 0 {
		plan.Route, plan.Template, plan.Outcome = contracts.RouteClarification, "clarify_missing", contracts.OutcomeClarificationRequired
		plan.Reason = "thiếu khe bắt buộc: " + strings.Join(cls.Intent.Missing, ", ")
		if cls.Unclear.Any() { // không thiếu — câu nêu nhiều giá trị khác nhau hoặc ngày hỏng cho khe đó
			plan.Reason = "câu nêu nhiều giá trị khác nhau hoặc ngày hỏng cho khe: " + strings.Join(cls.Intent.Missing, ", ")
		}
		if len(in.Ambiguities) > 0 {
			plan.Reason += " (có viết tắt mơ hồ)"
		}
		return plan
	}

	plan.Route, plan.Template = contracts.RouteKind(def.Route), def.Template
	plan.Reason = "ý định " + def.Name + " khớp luật"
	if cls.Compare != "" {
		plan.Route, plan.Template = contracts.RouteCalculation, "compare_total"
		plan.Reason += ", so với " + cls.Compare
	}
	if cls.Knowledge != "" && (plan.Route == contracts.RouteSQLQuery || plan.Route == contracts.RouteCalculation) {
		plan.Route = contracts.RouteHybrid
		plan.Reason += " + tri thức " + cls.Knowledge
	}
	plan.Outcome = terminalOutcome(plan.Route)
	switch plan.Route {
	case contracts.RouteSQLQuery:
		plan.Steps = []contracts.Step{{Kind: contracts.StepTool, Name: "metric_total"}}
	case contracts.RouteCalculation:
		plan.Steps = []contracts.Step{{Kind: contracts.StepCalculate, Name: "derived_metric"}}
	case contracts.RouteRAG:
		plan.Steps = []contracts.Step{{Kind: contracts.StepRetrieve, Name: "knowledge"}, {Kind: contracts.StepGenerate, Name: "answer"}}
	case contracts.RouteHybrid:
		plan.Steps = []contracts.Step{
			{Kind: contracts.StepTool, Name: "metric_total"}, {Kind: contracts.StepRetrieve, Name: "knowledge"},
			{Kind: contracts.StepCalculate, Name: "compare"}, {Kind: contracts.StepGenerate, Name: "answer"},
		}
	}
	return plan
}
