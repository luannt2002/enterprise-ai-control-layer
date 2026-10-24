package pipeline

import (
	"context"
	"testing"

	"github.com/luannt2002/enterprise-ai-control-layer/core/contracts"
	"github.com/luannt2002/enterprise-ai-control-layer/intelligence/intent"
)

// B1 chỉ dò chỉ mục khi luật không nhận ra ý định; ý định đã có (kể cả thiếu khe), hỏi lại nghĩa số liệu, nhiều ý định hoà nhau: không dò.
func TestKnowledgeProbeTrangThai(t *testing.T) {
	for _, c := range []struct {
		ten string
		cls intent.Result
		do  int
	}{
		{"không khớp", intent.Result{}, 1},
		{"thiếu thuật ngữ", intent.Result{NeedTerm: []string{"fuel_billed"}}, 1},
		{"ý định số liệu thiếu khe", intent.Result{Intent: contracts.Intent{Name: "fuel_issued", Missing: []string{"subject"}}}, 0},
		{"hỏi lại nghĩa số liệu", intent.Result{NeedMeaning: []string{"fuel_issued", "fuel_consumed"}}, 0},
		{"nhiều ý định hoà nhau", intent.Result{Ambiguous: []string{"sop_question", "fuel_issued"}}, 0},
		{"đã có ý định", intent.Result{Intent: contracts.Intent{Name: "sop_question"}}, 0},
	} {
		r := &truyDem{}
		ok, _, _, err := KnowledgeProbe{Knowledge: r, Name: "sop_question"}.Fallback(context.Background(), contracts.Principal{}, contracts.NormalizedInput{Raw: "x"}, c.cls)
		if r.lan != c.do || ok != (c.do == 1) || err != nil {
			t.Errorf("%s: dò %d lần (muốn %d), dùng %v, lỗi %v", c.ten, r.lan, c.do, ok, err)
		}
	}
}

type truyDem struct{ lan int }

func (r *truyDem) Retrieve(context.Context, contracts.Principal, string, int) ([]contracts.Evidence, error) {
	r.lan++
	return []contracts.Evidence{{ID: "D#1"}}, nil
}
