package entityresolution

import (
	"context"
	"testing"

	"github.com/luannt2002/enterprise-ai-control-layer/core/contracts"
)

// A-01 — cụm MƠ HỒ cũng phải mang vị trí (Start/End) như cụm không mơ hồ (contracts.AmbiguousMention): luồng hỏi cần biết "03" của
// "máy 03" nằm ở đâu để không coi nó là một con số, kể cả khi router gỡ bí danh mơ hồ bằng quyền xem hay dự án (khi đó thực thể
// lấy từ danh sách ứng viên, không có vị trí).
func TestResolveAmbiguousMentionBytes(t *testing.T) {
	for _, c := range []struct {
		text      string
		ambiguous []string // chữ của các cụm mơ hồ, theo thứ tự nhắc
		entities  []string // chữ của các cụm rõ
	}{
		{"cấp 2k cho máy 03 hôm nay", []string{"máy 03"}, nil},
		{"Hôm nay MÁY 03 lấy dầu", []string{"MÁY 03"}, nil},
		{"MX-03 (máy 03) hôm nay được cấp 3850 L", []string{"máy 03"}, []string{"MX-03"}},
		{"máy 03 và máy 03", []string{"máy 03", "máy 03"}, nil},
		{"máy 07 rồi máy 03", []string{"máy 03"}, []string{"máy 07"}},
	} {
		res, err := Resolve(context.Background(), dir, contracts.NormalizedInput{Raw: c.text})
		if err != nil {
			t.Fatal(err)
		}
		var amb, ent []string
		for _, m := range res.Ambiguous {
			if m.End <= m.Start || m.End > len(c.text) {
				t.Errorf("%q: cụm mơ hồ %q không có vị trí hợp lệ [%d,%d)", c.text, m.Mention, m.Start, m.End)
				continue
			}
			amb = append(amb, c.text[m.Start:m.End])
		}
		for _, e := range res.Entities {
			ent = append(ent, c.text[e.Start:e.End])
		}
		if len(amb) != len(c.ambiguous) || (len(amb) > 0 && !equalStrings(amb, c.ambiguous)) {
			t.Errorf("%q: cụm mơ hồ = %q, muốn %q", c.text, amb, c.ambiguous)
		}
		if !equalStrings(ent, c.entities) {
			t.Errorf("%q: cụm rõ = %q, muốn %q", c.text, ent, c.entities)
		}
	}
}

func equalStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
