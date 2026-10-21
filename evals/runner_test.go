package evals

import (
	"context"
	"strings"
	"testing"

	"github.com/luannt2002/enterprise-ai-control-layer/core/contracts"
)

type fixedAsker map[string]contracts.AskResponse

func (f fixedAsker) Ask(_ context.Context, _ contracts.Principal, q string) (contracts.AskResponse, error) {
	return f[q], nil
}

// Runner là giám khảo: phải bắt được từng loại sai, và không đánh trượt ca đúng.
func TestRunWinCatchesEachKindOfWrongAnswer(t *testing.T) {
	yes, no := true, false
	good := contracts.AskResponse{Route: "SQL_QUERY", Outcome: "RESOLVED", Answer: "Dầu kho xuất của EX-03: 3.850 L.",
		Conflicts: []contracts.Conflict{{Key: "k"}}, Citations: []contracts.Citation{{Source: contracts.SourceRef{SourceType: "warehouse_slip"}}}}
	clar := contracts.AskResponse{Route: "CLARIFICATION", Outcome: "CLARIFICATION_REQUIRED", Answer: "Máy nào?",
		Clarification: &contracts.Clarification{Candidates: []contracts.EntityRef{{ID: "EX-07"}, {ID: "RL-07"}}}}
	asker := fixedAsker{"q-good": good, "q-clar": clar}
	base := WinExpect{Route: "SQL_QUERY", Outcome: "RESOLVED", Value: &WinValue{Number: "3850", Unit: "L"},
		Conflict: &yes, CitedSources: []string{"warehouse_slip"}, NotContains: []string{"9.999"}}
	mk := func(id, q string, e WinExpect) WinCase {
		return WinCase{ID: id, FailureClass: "semantic", Principal: "p", Question: q, Expect: e}
	}
	wrongRoute, wrongValue, leak, noConflict, wrongCands := base, base, base, base, WinExpect{Route: "CLARIFICATION", Outcome: "CLARIFICATION_REQUIRED", Candidates: []string{"EX-07"}}
	wrongRoute.Route = "RAG"
	wrongValue.Value = &WinValue{Number: "3900", Unit: "L"}
	leak.NotContains = []string{"3.850"}
	noConflict.Conflict = &no
	cases := []WinCase{
		mk("ok", "q-good", base),
		mk("route", "q-good", wrongRoute),
		mk("value", "q-good", wrongValue),
		mk("leak", "q-good", leak),
		mk("conflict", "q-good", noConflict),
		mk("cands", "q-clar", wrongCands),
		mk("clar-ok", "q-clar", WinExpect{Route: "CLARIFICATION", Outcome: "CLARIFICATION_REQUIRED", Candidates: []string{"RL-07", "EX-07"}}),
	}
	rep := RunWin(context.Background(), asker, cases, map[string]contracts.Principal{"p": {}})
	want := map[string]string{"ok": "", "clar-ok": "", "route": "route =", "value": "thiếu số 3.900", "leak": "không được có", "conflict": "xung đột lộ ra", "cands": "ứng viên"}
	for _, r := range rep.Results {
		w := want[r.ID]
		if (w == "") != r.Pass || (w != "" && !strings.Contains(strings.Join(r.Problems, "; "), w)) {
			t.Errorf("%s: pass=%v problems=%v, muốn %q", r.ID, r.Pass, r.Problems, w)
		}
	}
	if got := rep.ByClass()["semantic"]; got != [2]int{2, 7} {
		t.Errorf("ByClass = %v, muốn 2/7", got)
	}
}

// countingAsker trả một phản hồi cố định và báo đã gọi model `calls` lần mỗi lượt.
type countingAsker struct {
	fixedAsker
	calls, got int
}

func (c *countingAsker) ModelCalls() int  { return c.got }
func (c *countingAsker) ResetModelCalls() { c.got = 0 }
func (c *countingAsker) Ask(ctx context.Context, pr contracts.Principal, q string) (contracts.AskResponse, error) {
	c.got = c.calls
	return c.fixedAsker.Ask(ctx, pr, q)
}

// model_calls là cổng G4: sai số lời gọi thì trượt; Asker không đếm được thì cũng trượt (không đo
// được thì không tính là đạt).
func TestRunWinChamSoLanGoiModel(t *testing.T) {
	r := contracts.AskResponse{Route: "RAG", Outcome: "RESOLVED", Answer: "x"}
	zero, one := 0, 1
	cases := []WinCase{
		{ID: "dung", FailureClass: "routing", Principal: "p", Question: "q", Expect: WinExpect{Route: "RAG", Outcome: "RESOLVED", ModelCalls: &one}},
		{ID: "sai", FailureClass: "routing", Principal: "p", Question: "q", Expect: WinExpect{Route: "RAG", Outcome: "RESOLVED", ModelCalls: &zero}},
	}
	prs := map[string]contracts.Principal{"p": {}}
	rep := RunWin(context.Background(), &countingAsker{fixedAsker: fixedAsker{"q": r}, calls: 1}, cases, prs)
	if !rep.Results[0].Pass || rep.Results[1].Pass || !strings.Contains(strings.Join(rep.Results[1].Problems, ";"), "model_calls = 1") {
		t.Errorf("đếm được: muốn dung đạt, sai trượt — được %+v", rep.Results)
	}
	rep = RunWin(context.Background(), fixedAsker{"q": r}, cases[:1], prs)
	if rep.Results[0].Pass || !strings.Contains(strings.Join(rep.Results[0].Problems, ";"), "không đếm được") {
		t.Errorf("Asker không đếm được lời gọi model: ca có model_calls phải trượt — được %+v", rep.Results[0])
	}
}

func TestViNumber(t *testing.T) {
	for in, want := range map[string]string{"3850": "3.850", "1200": "1.200", "68": "68", "5400": "5.400", "1000000": "1.000.000", "0.5": "0,5", "-68": "-68"} {
		if got := ViNumber(in); got != want {
			t.Errorf("ViNumber(%s) = %s, muốn %s", in, got, want)
		}
	}
}
