package evals

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/luannt2002/enterprise-ai-control-layer/core/contracts"
)

// Asker là thứ runner chấm: nhận câu hỏi của một principal, trả phản hồi. EACL và baseline RAG
// cùng hiện thực interface này để so trên cùng bộ ca (G7).
type Asker interface {
	Ask(ctx context.Context, pr contracts.Principal, question string) (contracts.AskResponse, error)
}

// ModelCounter là Asker đếm được số lời gọi gateway model, để chấm model_calls (cổng G4: route
// tất định không gọi model; route tri thức gọi đúng số lần). Asker không đếm được thì ca có
// model_calls bị chấm trượt — không đo được thì không tính là đạt.
type ModelCounter interface {
	ModelCalls() int
	ResetModelCalls()
}

// CaseResult là kết quả chấm một ca.
type CaseResult struct {
	ID, FailureClass string
	Pass             bool
	Problems         []string
	Answer           string
}

// Report là kết quả chấm cả bộ.
type Report struct {
	Results []CaseResult
}

// AskErrorProblem là tiền tố vấn đề RunWin ghi khi hệ LỖI khi hỏi (không phải câu trả lời của hệ). Bộ chấm nội
// dung G7 dựa vào nó để không bao giờ chấm lỗi hạ tầng như một lời từ chối.
const AskErrorProblem = "lỗi: "

// RunWin chấm từng ca bằng các kỳ vọng NHỊ PHÂN của ca — không so chuỗi nguyên văn, không dùng
// model làm giám khảo. principals ánh xạ id principal của ca → Principal đã xác thực.
func RunWin(ctx context.Context, a Asker, cases []WinCase, principals map[string]contracts.Principal) Report {
	var rep Report
	counter, counts := a.(ModelCounter)
	for _, c := range cases {
		res := CaseResult{ID: c.ID, FailureClass: c.FailureClass}
		bad := func(f string, x ...any) { res.Problems = append(res.Problems, fmt.Sprintf(f, x...)) }
		pr, ok := principals[c.Principal]
		if !ok {
			bad("%sprincipal %q không có", AskErrorProblem, c.Principal) // ca không hỏi được: lỗi, không phải câu trả lời rỗng
			rep.Results = append(rep.Results, res)
			continue
		}
		if counts {
			counter.ResetModelCalls()
		}
		r, err := a.Ask(ctx, pr, c.Question)
		if err != nil {
			r = contracts.AskResponse{Outcome: contracts.OutcomeSystemFailure}
			bad("%s%v", AskErrorProblem, err)
		}
		res.Answer = r.Answer
		e := c.Expect
		if string(r.Route) != e.Route {
			bad("route = %s, muốn %s", r.Route, e.Route)
		}
		if string(r.Outcome) != e.Outcome {
			bad("outcome = %s, muốn %s", r.Outcome, e.Outcome)
		}
		if v := e.Value; v != nil && !strings.Contains(r.Answer, ViNumber(v.Number)) {
			bad("thiếu số %s trong câu trả lời", ViNumber(v.Number))
		}
		for _, s := range e.Contains {
			if !strings.Contains(r.Answer, s) {
				bad("thiếu %q", s)
			}
		}
		for _, s := range e.NotContains {
			if strings.Contains(r.Answer, s) {
				bad("không được có %q", s)
			}
		}
		if len(e.Candidates) > 0 {
			var got []string
			if r.Clarification != nil {
				for _, x := range r.Clarification.Candidates {
					got = append(got, x.ID)
				}
			}
			if !sameSet(got, e.Candidates) {
				bad("ứng viên = %v, muốn %v", got, e.Candidates)
			}
		}
		if e.Conflict != nil {
			has := false
			for _, x := range r.Conflicts {
				has = has || !x.NeedsReview
			}
			if has != *e.Conflict {
				bad("xung đột lộ ra = %v, muốn %v", has, *e.Conflict)
			}
		}
		if want := e.ModelCalls; want != nil {
			switch {
			case !counts:
				bad("không đếm được lời gọi model")
			case counter.ModelCalls() != *want:
				bad("model_calls = %d, muốn %d", counter.ModelCalls(), *want)
			}
		}
		for _, src := range e.CitedSources {
			found := false
			for _, ci := range r.Citations {
				found = found || ci.Source.SourceType == src || ci.Source.DocumentID == src || ci.Source.Name == src
			}
			if !found {
				bad("thiếu trích dẫn %s", src)
			}
		}
		res.Pass = len(res.Problems) == 0
		rep.Results = append(rep.Results, res)
	}
	return rep
}

// ByClass đếm đạt/tổng theo lớp hỏng.
func (r Report) ByClass() map[string][2]int {
	out := map[string][2]int{}
	for _, x := range r.Results {
		v := out[x.FailureClass]
		v[1]++
		if x.Pass {
			v[0]++
		}
		out[x.FailureClass] = v
	}
	return out
}

// Summary là bảng đạt/tổng theo lớp + tổng, sắp xếp ổn định.
func (r Report) Summary() string {
	bc := r.ByClass()
	var b strings.Builder
	pass, total := 0, 0
	for _, fc := range FailureClasses {
		v := bc[fc]
		pass, total = pass+v[0], total+v[1]
		fmt.Fprintf(&b, "%-20s %3d/%-3d\n", fc, v[0], v[1])
	}
	fmt.Fprintf(&b, "%-20s %3d/%-3d\n", "TỔNG", pass, total)
	return b.String()
}

// ViNumber viết số theo kiểu Việt để so với câu trả lời: "3850" → "3.850", "0.5" → "0,5".
func ViNumber(n string) string {
	neg := strings.HasPrefix(n, "-")
	n = strings.TrimPrefix(n, "-")
	intPart, frac, _ := strings.Cut(n, ".")
	var groups []string
	for len(intPart) > 3 {
		groups = append([]string{intPart[len(intPart)-3:]}, groups...)
		intPart = intPart[:len(intPart)-3]
	}
	groups = append([]string{intPart}, groups...)
	out := strings.Join(groups, ".")
	if frac != "" {
		out += "," + frac
	}
	if neg {
		out = "-" + out
	}
	return out
}

func sameSet(a, b []string) bool {
	a, b = append([]string(nil), a...), append([]string(nil), b...)
	sort.Strings(a)
	sort.Strings(b)
	return strings.Join(a, ",") == strings.Join(b, ",")
}
