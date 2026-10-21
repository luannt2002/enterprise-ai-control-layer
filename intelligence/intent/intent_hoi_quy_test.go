package intent

import (
	"slices"
	"strings"
	"testing"

	"github.com/luannt2002/enterprise-ai-control-layer/core/contracts"
)

// Test cho các lỗi của bản soát độc lập lượt 2 ở tầng ý định: A-04 (câu ghép số liệu + quy trình nêu nhiều thực thể / ngày hỏng bị
// chuyển sang RAG, bỏ thầm phần số liệu) và mục "kỳ lồng nhau" người phản bác tìm thêm.

// spanIn là span ngày của cụm text trong raw (có vị trí byte thật, để bước nhận ra kỳ lồng nhau đọc được khoảng giữa hai cụm).
func spanIn(raw, text, iso string) contracts.Span {
	i := strings.Index(raw, text)
	sp := day(text, iso)
	sp.Start, sp.End = i, i+len(text)
	return sp
}

// A-04 — hai ý định cùng khớp: ý định số liệu (SQL_QUERY, có khe chủ thể bắt buộc) và ý định quy trình (RAG). Câu đã nêu thực thể
// (nhiều thực thể cùng loại tranh nhau) hoặc một mốc thời gian (kể cả ngày hỏng) là câu hỏi số liệu → trả ý định số liệu để hỏi lại;
// không nêu gì thì là câu hỏi quy trình thuần.
func TestCauGhepSoLieuVaQuyTrinhNeuThucTheHoacNgayHongThiGiuYDinhSoLieu(t *testing.T) {
	yes := func(string) bool { return true }
	set, err := Compile([]Def{
		{Name: "q", Route: "SQL_QUERY", Keywords: []string{"cấp"}, Capability: "metrics.read",
			Slots: []SlotDef{
				{Name: "subject", From: "entity", Types: []string{"equipment", "project"}, Required: true},
				{Name: "period", From: "period", Default: "today"},
			}},
		{Name: "sop", Route: "RAG", Keywords: []string{"quy trình"}, Capability: "documents.read"},
	}, yes, yes)
	if err != nil {
		t.Fatal(err)
	}
	ex, rl, dz := ent("EX-03", "equipment"), ent("RL-02", "equipment"), ent("DZ-03", "equipment")
	const raw = "cấp bao nhiêu và quy trình thế nào"
	for _, c := range []struct {
		name         string
		res          contracts.Resolution
		spans        []contracts.Span
		invalid      []string
		wantIntent   string
		wantMissing  []string
		wantKnowl    string
		wantEntities []string
	}{
		{"hai máy, không ngày", contracts.Resolution{Entities: []contracts.EntityRef{ex, rl}}, nil, nil, "q", []string{"subject"}, "", []string{"EX-03", "RL-02"}},
		{"một máy rõ và một bí danh mơ hồ khác máy đó", contracts.Resolution{Entities: []contracts.EntityRef{rl}, Ambiguous: []contracts.AmbiguousMention{alias("máy 03", "equipment", ex, dz)}},
			nil, nil, "q", []string{"subject"}, "", []string{"RL-02", "EX-03", "DZ-03"}},
		{"hai máy và ngày hỏng", contracts.Resolution{Entities: []contracts.EntityRef{ex, rl}}, nil, []string{"31/02"}, "q", []string{"subject", "period"}, "", []string{"EX-03", "RL-02"}},
		{"không máy, có ngày hỏng", contracts.Resolution{}, nil, []string{"31/02"}, "q", []string{"subject", "period"}, "", nil},
		// hành vi đã đúng từ trước: có mốc ngày hợp lệ thì hỏi lại; có đúng một máy thì ghép HYBRID; không nêu gì thì là câu hỏi quy trình
		{"hai máy, có ngày", contracts.Resolution{Entities: []contracts.EntityRef{ex, rl}}, []contracts.Span{day("hôm nay", "2026-10-19")}, nil, "q", []string{"subject"}, "", []string{"EX-03", "RL-02"}},
		{"không máy, có ngày", contracts.Resolution{}, []contracts.Span{day("hôm nay", "2026-10-19")}, nil, "q", []string{"subject"}, "", nil},
		{"một máy, không ngày: HYBRID", contracts.Resolution{Entities: []contracts.EntityRef{ex}}, nil, nil, "q", nil, "sop", nil},
		{"không máy, không ngày: câu hỏi quy trình", contracts.Resolution{}, nil, nil, "sop", nil, "", nil},
	} {
		in := contracts.NormalizedInput{Raw: raw, Spans: c.spans, InvalidDates: c.invalid}
		r := set.Classify(in, c.res, testNow)
		if r.Intent.Name != c.wantIntent {
			t.Errorf("%s: ý định = %q, muốn %q", c.name, r.Intent.Name, c.wantIntent)
			continue
		}
		if !slices.Equal(r.Intent.Missing, c.wantMissing) {
			t.Errorf("%s: Missing = %v, muốn %v", c.name, r.Intent.Missing, c.wantMissing)
		}
		if r.Knowledge != c.wantKnowl {
			t.Errorf("%s: Knowledge = %q, muốn %q", c.name, r.Knowledge, c.wantKnowl)
		}
		if got := ids(r.Unclear.Entities); !slices.Equal(got, c.wantEntities) {
			t.Errorf("%s: Unclear.Entities = %v, muốn %v", c.name, got, c.wantEntities)
		}
	}
}

// Kỳ LỒNG nhau người hỏi viết thành một cụm tự nhiên là MỘT kỳ — kỳ hẹp (kỳ rộng là bối cảnh): "tháng 10 năm nay", "hôm qua trong
// tháng 10". Hai kỳ nối bằng "và", hoặc làm đầu mút "từ …", hoặc không lồng nhau thì vẫn là nhiều kỳ → hỏi lại.
func TestKyLongNhauLaMotKyHepLaKyCuaKhe(t *testing.T) {
	ex := ent("EX-03", "equipment")
	res := contracts.Resolution{Entities: []contracts.EntityRef{ex}}
	for _, c := range []struct {
		raw        string
		spans      [][3]string // chữ, ngày chuẩn
		want       string      // giá trị khe thời gian ("" = để trống)
		wantRaws   []string    // Unclear.Periods khi hỏi lại
		wantNotMis bool
	}{
		{"EX-03 tháng 10 năm nay được cấp", [][3]string{{"tháng 10", "2026-10-01/2026-10-31"}, {"năm nay", "2026-01-01/2026-12-31"}}, "2026-10-01/2026-10-31", nil, true},
		{"EX-03 năm nay tháng 10 được cấp", [][3]string{{"năm nay", "2026-01-01/2026-12-31"}, {"tháng 10", "2026-10-01/2026-10-31"}}, "2026-10-01/2026-10-31", nil, true},
		{"EX-03 ngày 19/10 năm nay được cấp", [][3]string{{"ngày 19/10", "2026-10-19"}, {"năm nay", "2026-01-01/2026-12-31"}}, "2026-10-19", nil, true},
		{"EX-03 hôm qua trong tháng 10 được cấp", [][3]string{{"hôm qua", "2026-10-18"}, {"tháng 10", "2026-10-01/2026-10-31"}}, "2026-10-18", nil, true},
		{"EX-03 hôm nay thuộc tuần này được cấp", [][3]string{{"hôm nay", "2026-10-19"}, {"tuần này", "2026-10-19/2026-10-25"}}, "2026-10-19", nil, true},
		{"EX-03 ngày 19/10 của tháng 10 được cấp", [][3]string{{"ngày 19/10", "2026-10-19"}, {"tháng 10", "2026-10-01/2026-10-31"}}, "2026-10-19", nil, true},
		{"EX-03 hôm nay tháng 10 năm nay được cấp", [][3]string{{"hôm nay", "2026-10-19"}, {"tháng 10", "2026-10-01/2026-10-31"}, {"năm nay", "2026-01-01/2026-12-31"}}, "2026-10-19", nil, true},
		// không được gộp
		{"EX-03 tháng 10 và hôm qua được cấp", [][3]string{{"tháng 10", "2026-10-01/2026-10-31"}, {"hôm qua", "2026-10-18"}}, "", []string{"tháng 10", "hôm qua"}, false},
		{"EX-03 tuần trước, từ 12/10, được cấp", [][3]string{{"tuần trước", "2026-10-12/2026-10-18"}, {"12/10", "2026-10-12"}}, "", []string{"tuần trước", "12/10"}, false},
		{"EX-03 tháng 9 và tháng 10 được cấp", [][3]string{{"tháng 9", "2026-09-01/2026-09-30"}, {"tháng 10", "2026-10-01/2026-10-31"}}, "", []string{"tháng 9", "tháng 10"}, false},
		{"EX-03 từ 10/10 đến 15/10 được cấp", [][3]string{{"10/10", "2026-10-10"}, {"15/10", "2026-10-15"}}, "", []string{"10/10", "15/10"}, false},
		{"EX-03 tháng 9 tháng 10 được cấp", [][3]string{{"tháng 9", "2026-09-01/2026-09-30"}, {"tháng 10", "2026-10-01/2026-10-31"}}, "", []string{"tháng 9", "tháng 10"}, false},
		{"EX-03 tháng 10 rồi tuần này được cấp", [][3]string{{"tháng 10", "2026-10-01/2026-10-31"}, {"tuần này", "2026-10-19/2026-10-25"}}, "", []string{"tháng 10", "tuần này"}, false},
	} {
		var spans []contracts.Span
		for _, s := range c.spans {
			spans = append(spans, spanIn(c.raw, s[0], s[1]))
		}
		r := slotSet(t).Classify(contracts.NormalizedInput{Raw: c.raw, Spans: spans}, res, testNow)
		if got := slotText(r.Intent, "period"); got != c.want {
			t.Errorf("%q: period = %q, muốn %q", c.raw, got, c.want)
		}
		if c.wantNotMis != (len(r.Intent.Missing) == 0) {
			t.Errorf("%q: Missing = %v", c.raw, r.Intent.Missing)
		}
		if got := raws(r.Unclear.Periods); !slices.Equal(got, c.wantRaws) {
			t.Errorf("%q: Unclear.Periods = %v, muốn %v", c.raw, got, c.wantRaws)
		}
	}
}
