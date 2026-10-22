package intent

import (
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/luannt2002/enterprise-ai-control-layer/core/contracts"
)

// Ý định chạm dữ liệu/model mà quên khai capability phải bị chặn lúc nạp pack — nếu không,
// router coi capability rỗng là "không cần quyền" và mọi vai (kể cả lái xe) đều đọc được.
func TestDataRouteRequiresCapability(t *testing.T) {
	known := func(string) bool { return true }
	def := func(route, capability string) Def {
		return Def{Name: "x", Route: route, Keywords: []string{"k"}, Capability: capability}
	}
	for _, c := range []struct {
		name    string
		d       Def
		wantErr bool
	}{
		{"SQL thiếu capability", def("SQL_QUERY", ""), true},
		{"RAG thiếu capability", def("RAG", ""), true},
		{"SQL có capability", def("SQL_QUERY", "metrics.read"), false},
		{"chào hỏi (CLARIFICATION) không cần capability", def("CLARIFICATION", ""), false},
		{"từ chối (REJECT) không cần capability", def("REJECT", ""), false},
	} {
		_, err := Compile([]Def{c.d}, known, known)
		if (err != nil) != c.wantErr {
			t.Errorf("%s: lỗi = %v, muốn có lỗi = %v", c.name, err, c.wantErr)
		}
		if c.wantErr && err != nil && !strings.Contains(err.Error(), "không khai capability") {
			t.Errorf("%s: lỗi không nói rõ nguyên nhân: %v", c.name, err)
		}
	}
}

// Mệnh đề trọng tâm: câu nhiều mệnh đề thì mệnh đề chứa cụm đánh dấu là câu hỏi; câu một mệnh đề,
// hoặc không có cụm, thì không thu hẹp. Dấu chấm / phẩy trong số ("3.850", "0,5") không cắt mệnh đề.
func TestFocusClause(t *testing.T) {
	markers := []string{"đúng không"}
	for _, c := range []struct {
		text, want string
		ok         bool
	}{
		{"Hôm qua cả đội báo 4k, báo cáo ghi 4 m3, vậy máy 03 được cấp 4k dầu đúng không?", " vậy máy 03 được cấp 4k dầu đúng không", true},
		{"Máy 03 được cấp 3.850 lít dầu, đúng không?", " đúng không", true},
		{"Máy 03 được cấp 0,5 khối dầu đúng không?", "", false},
		{"Máy 03 được cấp bao nhiêu dầu?", "", false},
	} {
		s, e, ok := FocusClause(c.text, markers)
		if ok != c.ok || (ok && c.text[s:e] != c.want) {
			t.Errorf("%q: được %q (ok=%v), muốn %q (ok=%v)", c.text, c.text[s:e], ok, c.want, c.ok)
		}
	}
}

// --- S2-4: một khe chỉ nhận MỘT giá trị — câu nêu nhiều giá trị khác nhau thì khe để trống, không lấy vế đầu ---

var testNow = time.Date(2026, 10, 19, 10, 0, 0, 0, time.FixedZone("ICT", 7*3600))

// slotSet dựng tập ý định tối giản: một ý định số liệu có khe chủ thể (máy hoặc dự án, bắt buộc) và khe thời gian
// mặc định hôm nay — cùng hình dạng với các ý định số liệu của pack cầu đường.
func slotSet(t *testing.T) *Set {
	t.Helper()
	yes := func(string) bool { return true }
	s, err := Compile([]Def{{
		Name: "q", Route: "SQL_QUERY", Keywords: []string{"cấp"}, Capability: "metrics.read",
		Slots: []SlotDef{
			{Name: "subject", From: "entity", Types: []string{"equipment", "project"}, Required: true},
			{Name: "period", From: "period", Default: "today"},
		},
	}}, yes, yes)
	if err != nil {
		t.Fatal(err)
	}
	return s
}

func ent(id, typ string) contracts.EntityRef {
	return contracts.EntityRef{ID: id, Type: typ, Key: id, Name: id}
}

// alias là một bí danh khớp nhiều thực thể (kết quả của chặng nhận dạng khi không phân định được).
func alias(mention, typ string, cands ...contracts.EntityRef) contracts.AmbiguousMention {
	return contracts.AmbiguousMention{Mention: mention, Type: typ, Candidates: cands}
}

// day là span ngày: raw là chữ người hỏi viết, iso là ngày chuẩn (một ngày) hoặc "từ/đến" (một khoảng).
func day(raw, iso string) contracts.Span {
	kind := contracts.ValueDate
	if strings.Contains(iso, "/") {
		kind = contracts.ValuePeriod
	}
	return contracts.Span{Kind: kind, Raw: raw, Value: contracts.Value{Kind: kind, Text: iso}, Confidence: 1}
}

func classifyRes(t *testing.T, res contracts.Resolution, spans ...contracts.Span) Result {
	t.Helper()
	in := contracts.NormalizedInput{Raw: "được cấp bao nhiêu", Spans: spans}
	return slotSet(t).Classify(in, res, testNow)
}

func classify(t *testing.T, ents []contracts.EntityRef, spans ...contracts.Span) Result {
	t.Helper()
	return classifyRes(t, contracts.Resolution{Entities: ents}, spans...)
}

func slotText(it contracts.Intent, name string) string {
	s, ok := it.Slots[name]
	switch {
	case !ok:
		return ""
	case s.Entity != nil:
		return s.Entity.ID
	case s.Value != nil:
		return s.Value.Text
	}
	return s.Text
}

func ids(es []contracts.EntityRef) []string {
	var out []string
	for _, e := range es {
		out = append(out, e.ID)
	}
	return out
}

func raws(sp []contracts.Span) []string {
	var out []string
	for _, s := range sp {
		out = append(out, s.Raw)
	}
	return out
}

// Một chủ thể + một kỳ, nhắc một hay nhiều lần, hoặc máy kèm dự án chứa nó: điền khe như trước.
func TestKheMotGiaTriDienNhuCu(t *testing.T) {
	ex, rl, prj := ent("EX-03", "equipment"), ent("RL-02", "equipment"), ent("PRJ-A", "project")
	dz := ent("DZ-03", "equipment")
	for _, c := range []struct {
		name          string
		res           contracts.Resolution
		spans         []contracts.Span
		subject, when string
	}{
		{"một máy, một ngày", contracts.Resolution{Entities: []contracts.EntityRef{ex}}, []contracts.Span{day("hôm nay", "2026-10-19")}, "EX-03", "2026-10-19"},
		{"không nêu ngày thì lấy mặc định hôm nay", contracts.Resolution{Entities: []contracts.EntityRef{ex}}, nil, "EX-03", "2026-10-19"},
		{"một máy nhắc hai lần", contracts.Resolution{Entities: []contracts.EntityRef{ex, ex}}, []contracts.Span{day("hôm nay", "2026-10-19")}, "EX-03", "2026-10-19"},
		{"một ngày viết hai cách", contracts.Resolution{Entities: []contracts.EntityRef{ex}}, []contracts.Span{day("hôm nay", "2026-10-19"), day("19/10", "2026-10-19")}, "EX-03", "2026-10-19"},
		{"máy rồi dự án chứa nó: máy là chủ thể", contracts.Resolution{Entities: []contracts.EntityRef{ex, prj}}, nil, "EX-03", "2026-10-19"},
		// hành vi cũ, giữ nguyên chứ không phải được tán thành: hai loại khác nhau thì chủ thể là thực thể nhắc đầu tiên
		{"dự án rồi máy: theo thứ tự nhắc", contracts.Resolution{Entities: []contracts.EntityRef{prj, ex}}, nil, "PRJ-A", "2026-10-19"},
		{"một khoảng (tháng)", contracts.Resolution{Entities: []contracts.EntityRef{rl}}, []contracts.Span{day("tháng 9", "2026-09-01/2026-09-30")}, "RL-02", "2026-09-01/2026-09-30"},
		// một bí danh mơ hồ chưa phải hai lần nhắc: nó được router gỡ (Candidates), không vào Unclear
		{"một máy rõ và bí danh mơ hồ chứa chính nó", contracts.Resolution{Entities: []contracts.EntityRef{ex}, Ambiguous: []contracts.AmbiguousMention{alias("máy 03", "equipment", ex, dz)}}, nil, "EX-03", "2026-10-19"},
		{"dự án và bí danh mơ hồ của máy (\"máy 03 của dự án A\")", contracts.Resolution{Entities: []contracts.EntityRef{prj}, Ambiguous: []contracts.AmbiguousMention{alias("máy 03", "equipment", ex, dz)}}, nil, "PRJ-A", "2026-10-19"},
		{"cùng một bí danh mơ hồ nhắc hai lần", contracts.Resolution{Ambiguous: []contracts.AmbiguousMention{alias("máy 03", "equipment", ex, dz), alias("máy 03", "equipment", ex, dz)}}, nil, "", "2026-10-19"},
	} {
		r := classifyRes(t, c.res, c.spans...)
		if got := slotText(r.Intent, "subject"); got != c.subject {
			t.Errorf("%s: subject = %q, muốn %q", c.name, got, c.subject)
		}
		if got := slotText(r.Intent, "period"); got != c.when {
			t.Errorf("%s: period = %q, muốn %q", c.name, got, c.when)
		}
		if r.Unclear.Any() {
			t.Errorf("%s: không có gì không rõ để hỏi lại, được %+v", c.name, r.Unclear)
		}
		wantMissing := 0
		if c.subject == "" {
			wantMissing = 1 // chỉ bí danh mơ hồ, chưa có chủ thể: thiếu khe như trước
		}
		if len(r.Intent.Missing) != wantMissing {
			t.Errorf("%s: Missing = %v", c.name, r.Intent.Missing)
		}
	}
}

// Nhiều thực thể KHÁC NHAU cùng loại, hoặc nhiều kỳ khác nhau: khe KHÔNG được điền (không lấy vế đầu), nằm trong
// Missing để router hỏi lại, và Unclear ghi đúng các giá trị đã nhận ra.
func TestKheNhieuGiaTriKhacNhauKhongDuocDien(t *testing.T) {
	ex, rl, dz := ent("EX-03", "equipment"), ent("RL-02", "equipment"), ent("DZ-03", "equipment")
	e7, rl7 := ent("EX-07", "equipment"), ent("RL-07", "equipment")
	pa, pb := ent("PRJ-A", "project"), ent("PRJ-B", "project")
	today, yesterday := day("hôm nay", "2026-10-19"), day("hôm qua", "2026-10-18")
	res := func(es ...contracts.EntityRef) contracts.Resolution { return contracts.Resolution{Entities: es} }
	for _, c := range []struct {
		name         string
		res          contracts.Resolution
		spans        []contracts.Span
		wantMissing  []string
		subject      string   // khe chủ thể còn được điền? ("" = để trống)
		when         string   // khe thời gian còn được điền? ("" = để trống)
		wantEntities []string // Unclear.Entities
		wantPeriods  []string // Unclear.Periods (chữ người hỏi viết)
	}{
		{"hai máy", res(ex, rl), nil, []string{"subject"}, "", "2026-10-19", []string{"EX-03", "RL-02"}, nil},
		{"hai máy theo thứ tự khác", res(rl, ex), nil, []string{"subject"}, "", "2026-10-19", []string{"RL-02", "EX-03"}, nil},
		{"một máy nhắc hai lần và một máy khác", res(ex, rl, ex), nil, []string{"subject"}, "", "2026-10-19", []string{"EX-03", "RL-02"}, nil},
		{"ba máy", res(ex, rl, dz), nil, []string{"subject"}, "", "2026-10-19", []string{"EX-03", "RL-02", "DZ-03"}, nil},
		{"hai dự án", res(pa, pb), nil, []string{"subject"}, "", "2026-10-19", []string{"PRJ-A", "PRJ-B"}, nil},
		{"hai máy kèm dự án chứa chúng: dự án chỉ là định ngữ", res(ex, rl, pa), nil, []string{"subject"}, "", "2026-10-19", []string{"EX-03", "RL-02"}, nil},
		{"một máy rõ và một bí danh mơ hồ khác máy đó", contracts.Resolution{Entities: []contracts.EntityRef{rl}, Ambiguous: []contracts.AmbiguousMention{alias("máy 03", "equipment", ex, dz)}},
			nil, []string{"subject"}, "", "2026-10-19", []string{"RL-02", "EX-03", "DZ-03"}, nil},
		{"hai bí danh mơ hồ khác nhau", contracts.Resolution{Ambiguous: []contracts.AmbiguousMention{alias("máy 03", "equipment", ex, dz), alias("máy 07", "equipment", e7, rl7)}},
			nil, []string{"subject"}, "", "2026-10-19", []string{"EX-03", "DZ-03", "EX-07", "RL-07"}, nil},
		{"hai ngày", res(ex), []contracts.Span{today, yesterday}, []string{"period"}, "EX-03", "", nil, []string{"hôm nay", "hôm qua"}},
		{"hai ngày theo thứ tự khác", res(ex), []contracts.Span{yesterday, today}, []string{"period"}, "EX-03", "", nil, []string{"hôm qua", "hôm nay"}},
		{"hai đầu mút của một khoảng", res(ex), []contracts.Span{day("10/10", "2026-10-10"), day("15/10", "2026-10-15")}, []string{"period"}, "EX-03", "", nil, []string{"10/10", "15/10"}},
		{"hai tháng", res(pa), []contracts.Span{day("tháng 9", "2026-09-01/2026-09-30"), day("tháng 10", "2026-10-01/2026-10-31")}, []string{"period"}, "PRJ-A", "", nil, []string{"tháng 9", "tháng 10"}},
		{"một ngày và một tháng", res(ex), []contracts.Span{today, day("tháng 9", "2026-09-01/2026-09-30")}, []string{"period"}, "EX-03", "", nil, []string{"hôm nay", "tháng 9"}},
		{"ba kỳ, một kỳ lặp", res(ex), []contracts.Span{today, yesterday, day("19/10", "2026-10-19")}, []string{"period"}, "EX-03", "", nil, []string{"hôm nay", "hôm qua"}},
		{"hai máy và hai ngày", res(ex, rl), []contracts.Span{today, yesterday}, []string{"subject", "period"}, "", "", []string{"EX-03", "RL-02"}, []string{"hôm nay", "hôm qua"}},
	} {
		r := classifyRes(t, c.res, c.spans...)
		if !slices.Equal(r.Intent.Missing, c.wantMissing) {
			t.Errorf("%s: Missing = %v, muốn %v", c.name, r.Intent.Missing, c.wantMissing)
		}
		if got := slotText(r.Intent, "subject"); got != c.subject {
			t.Errorf("%s: subject = %q, muốn %q", c.name, got, c.subject)
		}
		if got := slotText(r.Intent, "period"); got != c.when {
			t.Errorf("%s: period = %q, muốn %q", c.name, got, c.when)
		}
		if got := ids(r.Unclear.Entities); !slices.Equal(got, c.wantEntities) {
			t.Errorf("%s: Unclear.Entities = %v, muốn %v", c.name, got, c.wantEntities)
		}
		if got := raws(r.Unclear.Periods); !slices.Equal(got, c.wantPeriods) {
			t.Errorf("%s: Unclear.Periods = %v, muốn %v", c.name, got, c.wantPeriods)
		}
		if len(r.Candidates) > 0 {
			t.Errorf("%s: thứ người hỏi viết rõ không phải bí danh mơ hồ, không được vào Candidates (router sẽ gỡ bằng quyền xem): %v", c.name, ids(r.Candidates))
		}
	}
}

// Ngày hỏng ("31/02") không sinh span ngày; bộ chuẩn hoá ghi nó ở InvalidDates. Khe thời gian KHÔNG được rơi về ngày mặc
// định như thể người hỏi không nêu ngày: để trống (Missing) cho router hỏi lại, và Unclear ghi các ngày hỏng.
func TestNgayHongThiKheThoiGianKhongRoiVeMacDinh(t *testing.T) {
	ex := ent("EX-03", "equipment")
	for _, c := range []struct {
		name  string
		bad   []string
		spans []contracts.Span
	}{
		{"ngày hỏng một mình", []string{"31/02"}, nil},
		{"ngày hỏng cạnh một ngày tốt", []string{"31/02"}, []contracts.Span{day("hôm nay", "2026-10-19")}},
		{"hai ngày hỏng", []string{"31/02", "32/10"}, nil},
	} {
		in := contracts.NormalizedInput{Raw: "được cấp bao nhiêu", Spans: c.spans, InvalidDates: c.bad}
		r := slotSet(t).Classify(in, contracts.Resolution{Entities: []contracts.EntityRef{ex}}, testNow)
		if !slices.Equal(r.Intent.Missing, []string{"period"}) {
			t.Errorf("%s: Missing = %v, muốn [period]", c.name, r.Intent.Missing)
		}
		if got := slotText(r.Intent, "period"); got != "" {
			t.Errorf("%s: period = %q — ngày hỏng không được rơi về ngày mặc định hay ngày tốt bên cạnh", c.name, got)
		}
		if got := slotText(r.Intent, "subject"); got != "EX-03" {
			t.Errorf("%s: subject = %q, muốn EX-03 (chỉ khe thời gian có vấn đề)", c.name, got)
		}
		if !slices.Equal(r.Unclear.Invalid, c.bad) || len(r.Unclear.Periods) != 0 {
			t.Errorf("%s: Unclear = %+v, muốn Invalid = %v và không có Periods", c.name, r.Unclear, c.bad)
		}
	}
}

// Bộ ý định không có ý định chung (yếu) đòi thuật ngữ của câu: câu có thực thể + thuật ngữ + cụm xác nhận mà không từ khoá nào khớp thì
// hỏi lại nghĩa số liệu, không sập (soát S4 vòng 2, F3: generic() đọc ý định chung đầu tiên trước khi đếm số ý định chung).
func TestYDinhChungKhongCoThiHoiLaiKhongSap(t *testing.T) {
	yes := func(string) bool { return true }
	s, err := Compile([]Def{{
		Name: "q", Route: "SQL_QUERY", Keywords: []string{"cấp"}, Terms: []string{"fuel"}, Capability: "metrics.read",
		MetricByTerm: map[string]string{"fuel": "issued"},
		Slots:        []SlotDef{{Name: "subject", From: "entity", Types: []string{"equipment"}, Required: true}, {Name: "period", From: "period", Default: "today"}},
	}}, yes, yes)
	if err != nil {
		t.Fatal(err)
	}
	s.SetOptions(Options{FocusMarkers: []string{"đúng không"}})
	in := contracts.NormalizedInput{Raw: "EX-03 dầu đúng không", Spans: []contracts.Span{
		{Kind: contracts.ValueTerm, Raw: "dầu", Start: 6, End: 10, Value: contracts.Value{Kind: contracts.ValueTerm, Text: "fuel"}}}}
	r := s.Classify(in, contracts.Resolution{Entities: []contracts.EntityRef{ent("EX-03", "equipment")}}, testNow)
	if r.Intent.Name != "" || !slices.Equal(r.NeedMeaning, []string{"q"}) {
		t.Errorf("muốn hỏi lại nghĩa số liệu [q], được intent=%q needMeaning=%v", r.Intent.Name, r.NeedMeaning)
	}
}
