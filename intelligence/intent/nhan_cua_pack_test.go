package intent

import (
	"strings"
	"testing"

	"github.com/luannt2002/enterprise-ai-control-layer/core/contracts"
	"github.com/luannt2002/enterprise-ai-control-layer/data/normalization"
)

// F-030: số trần là nhãn / số đếm hay một CON SỐ của câu do Options.Labels quyết (danh từ nhãn / đếm của lexicon pack). Không khai Labels thì như
// pack không khai danh từ nào: "2 máy" là một con số, còn "phiếu 912", "3 lần" vẫn là nhãn / số đếm (từ chức năng của bộ chuẩn hoá).
func TestHasFigureDocNhanCuaPack(t *testing.T) {
	yes := func(string) bool { return true }
	s, err := Compile(nil, yes, yes)
	if err != nil {
		t.Fatal(err)
	}
	eng, err := normalization.Compile(normalization.Lexicon{CountNouns: []string{"máy"}})
	if err != nil {
		t.Fatal(err)
	}
	figure := func(raw, num string) bool {
		at := strings.Index(raw, num)
		in := contracts.NormalizedInput{Raw: raw, Spans: []contracts.Span{{Kind: contracts.ValueNumber, Raw: num, Start: at, End: at + len(num)}}}
		return s.hasFigure(in, contracts.Resolution{}, 0, len(raw))
	}
	for _, c := range []struct {
		raw, num            string
		khongKhai, packKhai bool
	}{
		{"cho 2 máy", "2", true, false},
		{"theo phiếu 912", "912", false, false},
		{"đổ 3 lần", "3", false, false},
		{"được cấp 3850", "3850", true, true},
	} {
		s.SetOptions(Options{})
		if got := figure(c.raw, c.num); got != c.khongKhai {
			t.Errorf("Labels không khai: %q có con số = %v, muốn %v", c.raw, got, c.khongKhai)
		}
		s.SetOptions(Options{Labels: eng.LabelContext})
		if got := figure(c.raw, c.num); got != c.packKhai {
			t.Errorf("Labels của pack khai \"máy\": %q có con số = %v, muốn %v", c.raw, got, c.packKhai)
		}
	}
}
