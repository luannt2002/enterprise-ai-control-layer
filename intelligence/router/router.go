// Package router là chặng S5: ý định → kế hoạch route.
//
// Router không đoán. Mơ hồ hoặc thiếu khe → CLARIFICATION; không có quyền → REJECT;
// không ý định nào khớp → route do pack chọn cho trường hợp này. Quyền ở đây là quyền
// của PHẦN MỀM (vai nghiệp vụ → capability, phạm vi thực thể), không phải "vai" trong prompt.
package router

import (
	"fmt"
	"maps"
	"slices"
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
func narrow(cls intent.Result, def intent.Def, pr contracts.Principal, pol Policy, lineage func(contracts.EntityRef) []string) (contracts.Intent, contracts.Narrowing) {
	it := cls.Intent
	if len(cls.Candidates) == 0 {
		return it, contracts.Narrowing{}
	}
	slot := ""
	for _, sd := range def.Slots {
		if sd.From == "entity" {
			slot = sd.Name
			break
		}
	}
	if slot == "" {
		return it, contracts.Narrowing{}
	}
	// nar chỉ GHI lại việc lọc (ứng viên trước / sau) để vết giải thích được — không điều khiển gì
	nar := contracts.Narrowing{Slot: slot, Before: keysOf(cls.Candidates)}
	var keep []contracts.EntityRef
	if cur, ok := it.Slots[slot]; ok && cur.Entity != nil {
		if cur.Entity.Type != "project" {
			return it, contracts.Narrowing{}
		}
		nar.By = contracts.NarrowByProject
		for _, c := range cls.Candidates {
			for _, k := range lineage(c) {
				if k == cur.Entity.Key {
					keep = append(keep, c)
					break
				}
			}
		}
		nar.After = keysOf(keep)
		if len(keep) != 1 {
			delete(it.Slots, slot)
			it.Missing = append(it.Missing, slot)
			return it, nar
		}
	} else {
		nar.By = contracts.NarrowByScope
		for _, c := range cls.Candidates {
			if pol.InScope(pr, lineage(c)) {
				keep = append(keep, c)
			}
		}
		nar.After = keysOf(keep)
		if len(keep) != 1 {
			return it, nar
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
	nar.Chosen = ref.Key
	return it, nar
}

// keysOf trả khoá của các thực thể theo đúng thứ tự và số lượng router đã xét — KHÔNG gộp trùng: một thực thể là ứng viên của hai cụm mơ hồ
// được đếm hai lần ở bước "đúng một ứng viên", và vết phải cho thấy chính con số đó.
func keysOf(refs []contracts.EntityRef) []string {
	out := make([]string, len(refs))
	for i, r := range refs {
		out[i] = r.Key
	}
	return out
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
//
// Mỗi bậc kiểm được ghi vào plan.Checked ĐÚNG LÚC nó được xét (hàm check), kể cả bậc không chặn — nên vết cho biết router đã đi qua
// những bậc nào, bậc nào chặn và dữ kiện nào đã kiểm; không bậc nào đọc lại danh sách này để quyết, nên ghi không đổi quyết định.
func Route(set *intent.Set, cls intent.Result, in contracts.NormalizedInput, pr contracts.Principal, pol Policy, st Settings,
	lineage func(e contracts.EntityRef) []string) (plan contracts.RoutePlan) {

	var checked []contracts.RouteCheck
	var narrowed contracts.Narrowing
	defer func() { plan.Checked, plan.Narrowed = checked, narrowed }()
	check := func(rule string, route contracts.RouteKind, fired bool, detail string) bool {
		checked = append(checked, contracts.RouteCheck{Rule: rule, Route: route, Fired: fired, Detail: detail})
		return fired
	}
	switch {
	case check("nhiều ý định cùng khớp", contracts.RouteClarification, len(cls.Ambiguous) > 0, orNone(strings.Join(cls.Ambiguous, ", "), "không có ý định nào hoà nhau")):
		return contracts.RoutePlan{
			Route: contracts.RouteClarification, Template: "clarify_intent",
			Reason:  "nhiều ý định cùng khớp: " + strings.Join(cls.Ambiguous, ", "),
			Outcome: contracts.OutcomeClarificationRequired,
		}
	case check("thiếu thuật ngữ bắt buộc", contracts.RouteClarification, len(cls.NeedTerm) > 0, orNone(strings.Join(cls.NeedTerm, ", "), "không có ý định nào khớp từ khoá mà thiếu thuật ngữ")):
		return contracts.RoutePlan{
			Route: contracts.RouteClarification, Template: "clarify_intent",
			Reason:  "khớp từ khoá nhưng thiếu thuật ngữ bắt buộc: " + strings.Join(cls.NeedTerm, ", "),
			Outcome: contracts.OutcomeClarificationRequired,
		}
	case check("hỏi lại nghĩa số liệu", contracts.RouteClarification, len(cls.NeedMeaning) > 0, orNone(strings.Join(cls.NeedMeaning, ", "), "không rơi vào trường hợp có thực thể và thuật ngữ mà không từ khoá nào khớp")):
		// thực thể ngoài phạm vi dữ liệu của người hỏi: nghĩa nào cũng bị từ chối, nên từ chối ngay, không hỏi lại
		for _, e := range cls.Mentioned {
			out := !pol.InScope(pr, lineage(e))
			if check("phạm vi dữ liệu (câu hỏi nghĩa số liệu)", contracts.RouteReject, out, e.Key+": "+pick(out, "ngoài phạm vi", "trong phạm vi")) {
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
	case check("không ý định nào khớp", contracts.RouteKind(st.UnmatchedRoute), cls.Intent.Name == "", pick(cls.Intent.Name == "", "route do pack chọn cho câu không khớp ý định nào", "đã có ý định "+cls.Intent.Name)):
		r := contracts.RouteKind(st.UnmatchedRoute)
		return contracts.RoutePlan{
			Route: r, Template: st.UnmatchedTemplate,
			Reason: "không ý định nào khớp", Outcome: terminalOutcome(r),
		}
	}
	def, _ := set.Def(cls.Intent.Name)
	var it contracts.Intent
	it, narrowed = narrow(cls, def, pr, pol, lineage)
	plan = contracts.RoutePlan{Intent: it}
	cls.Intent = plan.Intent

	// Ý định không khai capability chỉ có thể là route thuần trả lời (intent.Compile chặn
	// mọi trường hợp khác), nên bỏ qua kiểm quyền ở đây là an toàn.
	if def.Capability != "" {
		ok := pol.Allows(pr, def.Capability)
		if check("capability của ý định", contracts.RouteReject, !ok, def.Capability+": "+pick(ok, "có", "thiếu")+" (vai "+strings.Join(pr.Roles, ", ")+")") {
			plan.Route, plan.Template, plan.Outcome = contracts.RouteReject, "forbidden", contracts.OutcomePermissionDenied
			plan.Reason = "thiếu capability " + def.Capability
			return plan
		}
	}
	if cls.Compare != "" { // câu so sánh đọc cả hai chỉ số: cần quyền của cả hai vế
		if cdef, _ := set.Def(cls.Compare); cdef.Capability != "" {
			ok := pol.Allows(pr, cdef.Capability)
			if check("capability của vế so sánh", contracts.RouteReject, !ok, cdef.Capability+": "+pick(ok, "có", "thiếu")) {
				plan.Route, plan.Template, plan.Outcome = contracts.RouteReject, "forbidden", contracts.OutcomePermissionDenied
				plan.Reason = "thiếu capability " + cdef.Capability + " cho vế so sánh"
				return plan
			}
		}
	}
	if cls.Knowledge != "" { // HYBRID cần cả quyền đọc tài liệu
		if kdef, _ := set.Def(cls.Knowledge); kdef.Capability != "" {
			ok := pol.Allows(pr, kdef.Capability)
			if check("capability của phần tri thức", contracts.RouteReject, !ok, kdef.Capability+": "+pick(ok, "có", "thiếu")) {
				plan.Route, plan.Template, plan.Outcome = contracts.RouteReject, "forbidden", contracts.OutcomePermissionDenied
				plan.Reason = "thiếu capability " + kdef.Capability + " cho phần tri thức"
				return plan
			}
		}
	}
	for _, name := range slices.Sorted(maps.Keys(cls.Intent.Slots)) {
		s := cls.Intent.Slots[name]
		if s.Entity == nil {
			continue
		}
		out := !pol.InScope(pr, lineage(*s.Entity))
		if check("phạm vi dữ liệu", contracts.RouteReject, out, s.Entity.Key+": "+pick(out, "ngoài phạm vi", "trong phạm vi")+" của người hỏi") {
			plan.Route, plan.Template, plan.Outcome = contracts.RouteReject, "forbidden", contracts.OutcomePermissionDenied
			plan.Reason = "thực thể " + s.Entity.Key + " ngoài phạm vi dữ liệu của người hỏi"
			return plan
		}
	}
	if check("thiếu khe bắt buộc", contracts.RouteClarification, len(cls.Intent.Missing) > 0, orNone(strings.Join(cls.Intent.Missing, ", "), "mọi khe bắt buộc đã có")) {
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
	if cls.Intent.Rule == "intent:fallback" { // ý định do dự phòng S4 chọn (pipeline.IntentFallback), không phải luật từ khoá
		plan.Reason = "ý định " + def.Name + " do dự phòng S4 chọn"
	}
	check("route khai của ý định", plan.Route, true, def.Name+" → "+def.Route)
	compare := check("câu so sánh hai nghĩa", contracts.RouteCalculation, cls.Compare != "", orNone(cls.Compare, "không có vế so sánh"))
	if compare {
		plan.Route, plan.Template = contracts.RouteCalculation, "compare_total"
		plan.Reason += ", so với " + cls.Compare
	}
	if check("ý định tri thức khớp kèm", contracts.RouteHybrid, cls.Knowledge != "" && (plan.Route == contracts.RouteSQLQuery || plan.Route == contracts.RouteCalculation),
		orNone(cls.Knowledge, "không có ý định tri thức khớp kèm")) {
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

// pick trả yes khi ok, ngược lại no — để ghi dữ kiện của bậc kiểm thành một câu.
func pick(ok bool, yes, no string) string {
	if ok {
		return yes
	}
	return no
}

// orNone trả s, hoặc none khi s rỗng.
func orNone(s, none string) string {
	if s == "" {
		return none
	}
	return s
}
