package evals

import (
	"encoding/json"
	"os"
	"sort"
	"strings"
	"testing"
)

// g7Attack là một đòn của người tấn công bộ chấm độc lập (20/10/2026 tối, 170 đòn): một cặp (ca, câu
// trả lời) kèm phán quyết ĐÚNG theo nghiệp vụ. Phán quyết của bộ chấm không nằm trong tệp: test gọi
// bộ chấm thật.
type g7Attack struct {
	ID      string `json:"id"`
	Kind    string `json:"kind"`
	CaseID  string `json:"case_id"`
	Answer  string `json:"answer"`
	Pass    bool   `json:"pass"`
	Why     string `json:"why"`
	Dispute bool   `json:"dispute"`
}

// TestG7ScorerAttacks chạy bộ chấm nội dung trên từng đòn và so với testdata/g7-tan-cong-lech.txt: danh
// sách đòn mà bộ chấm HIỆN còn chấm sai (ratchet, như passing.txt). Hai chiều đều đỏ:
//   - đòn đang được chấm đúng mà nay chấm sai là HỒI QUY;
//   - đòn trong danh sách mà nay chấm đúng thì phải xoá khỏi danh sách (nợ chỉ được giảm, có chủ ý).
//
// Đòn còn tranh cãi chính sách (dispute) chưa vào phép so: chờ chính sách của bộ chấm v2 (ADR-0007).
func TestG7ScorerAttacks(t *testing.T) {
	raw, err := os.ReadFile("testdata/g7-tan-cong.json")
	if err != nil {
		t.Fatal(err)
	}
	var data struct {
		Synthetic []WinCase  `json:"synthetic_cases"`
		Attacks   []g7Attack `json:"attacks"`
	}
	if err := json.Unmarshal(raw, &data); err != nil {
		t.Fatal(err)
	}
	suite, err := LoadWinSuite("../tenants/construction/win/cases-independent.yaml")
	if err != nil {
		t.Fatal(err)
	}
	cases := append(append([]WinCase(nil), suite.Cases...), data.Synthetic...)
	byID := map[string]WinCase{}
	for _, c := range cases {
		byID[c.ID] = c
	}
	var wrong []string
	disputed := 0
	for _, a := range data.Attacks {
		c, ok := byID[a.CaseID]
		if !ok {
			t.Fatalf("đòn %s trỏ tới ca không có: %s", a.ID, a.CaseID)
		}
		got := G7ContentReport(cases, Report{Results: []CaseResult{{ID: c.ID, FailureClass: c.FailureClass, Answer: a.Answer}}}).Results[0].Pass
		if a.Dispute {
			disputed++
			continue
		}
		if got != a.Pass {
			wrong = append(wrong, a.ID)
		}
	}
	sort.Strings(wrong)
	listRaw, err := os.ReadFile("testdata/g7-tan-cong-lech.txt")
	if err != nil {
		t.Fatal(err)
	}
	known := map[string]bool{}
	for _, l := range strings.Split(string(listRaw), "\n") {
		if l = strings.TrimSpace(l); l != "" && !strings.HasPrefix(l, "#") {
			known[l] = true
		}
	}
	now := map[string]bool{}
	for _, id := range wrong {
		now[id] = true
		if !known[id] {
			t.Errorf("HỒI QUY: đòn %s đang được chấm đúng nay bị chấm sai", id)
		}
	}
	for id := range known {
		if !now[id] {
			t.Errorf("đòn %s nay được chấm đúng: xoá nó khỏi testdata/g7-tan-cong-lech.txt", id)
		}
	}
	t.Logf("%d đòn; %d còn chấm sai (nợ đã ghi); %d tranh cãi chưa so", len(data.Attacks), len(wrong), disputed)
}
