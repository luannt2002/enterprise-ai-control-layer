package entityresolution

import (
	"context"
	"errors"
	"testing"

	"github.com/luannt2002/enterprise-ai-systems/core/contracts"
)

// Dữ liệu test, không phải luật của core.
var dir = MemoryDirectory{
	{ID: "e1", Type: "equipment", Key: "EX-03", Name: "Máy xúc EX-03", Aliases: []Alias{{Text: "máy xúc 03"}, {Text: "MX-03"}}},
	{ID: "e2", Type: "equipment", Key: "RL-03", Name: "Lu rung RL-03", Aliases: []Alias{{Text: "lu 03"}}},
	{ID: "e3", Type: "equipment", Key: "EX-07", Name: "Máy xúc EX-07", Aliases: []Alias{{Text: "máy 07"}}},
	{ID: "p1", Type: "project", Key: "PRJ-A", Name: "Cầu A", Aliases: []Alias{{Text: "dự án A"}, {Text: "cầu A"}}},
	{ID: "t1", Type: "team", Key: "TEAM-SH", Name: "Sông Hồng FC", Aliases: []Alias{{Text: "Song Hong", Source: "provider:FeedAlpha"}, {Text: "SH FC", Source: "provider:FeedBeta"}}},
	// "máy 03" là bí danh của HAI máy khác loại — phải ra mơ hồ.
	{ID: "e4", Type: "equipment", Key: "DZ-03", Name: "Máy ủi DZ-03", Aliases: []Alias{{Text: "máy 03"}}},
}

func init() {
	dir[0].Aliases = append(dir[0].Aliases, Alias{Text: "máy 03"})
}

func in(s string) contracts.NormalizedInput { return contracts.NormalizedInput{Raw: s} }

func TestKey(t *testing.T) {
	cases := map[string]string{
		"MX-03":        "mx 3",
		"Máy  03":      "may 3",
		"EX03":         "ex 3",
		"Sông Hồng FC": "song hong fc",
		"cầu A":        "cau a",
		"0":            "0",
	}
	for s, want := range cases {
		if got := Key(s); got != want {
			t.Errorf("Key(%q) = %q, muốn %q", s, got, want)
		}
	}
}

func TestResolve(t *testing.T) {
	ctx := context.Background()
	cases := []struct {
		name      string
		text      string
		types     []string
		wantKeys  []string
		ambiguous int
	}{
		{"khoá chuẩn viết thường", "ex 03 hôm nay", nil, []string{"EX-03"}, 0},
		{"bí danh có dấu nối", "MX-03 lấy dầu", nil, []string{"EX-03"}, 0},
		{"cụm dài nhất thắng", "máy xúc 03 chạy 7h", nil, []string{"EX-03"}, 0},
		{"không dấu vẫn khớp", "du an A tong dau", nil, []string{"PRJ-A"}, 0},
		{"bí danh provider", "Song Hong thắng", nil, []string{"TEAM-SH"}, 0},
		{"lọc theo loại", "cầu A máy 07", []string{"project"}, []string{"PRJ-A"}, 0},
		{"hai thực thể", "cầu A và máy 07", nil, []string{"PRJ-A", "EX-07"}, 0},
		// "máy 03" khớp EX-03 và DZ-03 → không chọn bừa.
		{"mơ hồ", "máy 03 hôm nay", nil, nil, 1},
		{"không có gì", "trời mưa", nil, nil, 0},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			res, err := Resolve(ctx, dir, in(c.text), c.types...)
			if err != nil {
				t.Fatal(err)
			}
			var keys []string
			for _, e := range res.Entities {
				keys = append(keys, e.Key)
			}
			if len(keys) != len(c.wantKeys) {
				t.Fatalf("%q → %v, muốn %v (mơ hồ: %+v)", c.text, keys, c.wantKeys, res.Ambiguous)
			}
			for k := range keys {
				if keys[k] != c.wantKeys[k] {
					t.Fatalf("%q → %v, muốn %v", c.text, keys, c.wantKeys)
				}
			}
			if len(res.Ambiguous) != c.ambiguous {
				t.Fatalf("%q: %d mơ hồ, muốn %d", c.text, len(res.Ambiguous), c.ambiguous)
			}
			if c.ambiguous == 1 && len(res.Ambiguous[0].Candidates) != 2 {
				t.Fatalf("mơ hồ phải liệt kê đủ 2 ứng viên: %+v", res.Ambiguous[0])
			}
		})
	}
}

type brokenDir struct{}

func (brokenDir) Entities(context.Context, ...string) ([]Entity, error) {
	return nil, errors.New("kết nối hỏng")
}

// Start/End trỏ đúng cụm đã khớp trong văn bản NFC (cùng hệ toạ độ với Span), kể cả khi cụm
// nằm giữa câu và có chữ mang dấu đứng trước.
func TestResolveReportsMentionBytes(t *testing.T) {
	for _, text := range []string{"cấp 2k cho máy xúc 03 hôm nay", "Hôm nay MX-03 lấy dầu", "máy xúc 03"} {
		in := contracts.NormalizedInput{Raw: text}
		res, err := Resolve(context.Background(), dir, in)
		if err != nil || len(res.Entities) != 1 {
			t.Fatalf("%q: %v %+v", text, err, res)
		}
		e := res.Entities[0]
		got := text[e.Start:e.End]
		if got != "máy xúc 03" && got != "MX-03" {
			t.Errorf("%q: cụm [%d,%d) = %q, muốn đúng cụm tên máy", text, e.Start, e.End, got)
		}
	}
}

func TestLoiDanhMucCoNhan(t *testing.T) {
	_, err := Resolve(context.Background(), brokenDir{}, in("x"))
	if contracts.FaultOf(err) != "infra:directory" {
		t.Fatalf("lỗi danh mục phải mang nhãn hạ tầng, được %q", contracts.FaultOf(err))
	}
}
