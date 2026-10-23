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
			nil, []string{"subject"}, "", "2026-10-19", []string{"EX-03", "DZ-03"}, nil}, // D-OP-11 = b: chỉ ứng viên của bí danh (trước đây gộp cả RL-02)
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

// D-OP-11 = b (chủ dự án chốt 23/10, đổi hợp đồng có chủ ý so với bản ghim cũ): có bí danh mơ hồ thì câu hỏi lại CHỈ liệt kê ứng viên của
// bí danh; thực thể đã giải không vào danh sách (vẫn được tính để biết câu có nhiều chủ thể); hai thực thể đều đã giải thì giữ cách cũ.
// Cặp tối thiểu theo ADR-0010 bất biến (c): một đã giải + một mơ hồ · hai mơ hồ · hai đã giải (đối chứng, không đổi).
func TestUngVienHoiLaiChiLaUngVienCuaBiDanhMoHo(t *testing.T) {
	ex, rl, dz := ent("EX-03", "equipment"), ent("RL-02", "equipment"), ent("DZ-03", "equipment")
	e7, rl7, pa := ent("EX-07", "equipment"), ent("RL-07", "equipment"), ent("PRJ-A", "project")
	m3, m7 := alias("máy 03", "equipment", ex, dz), alias("máy 07", "equipment", e7, rl7)
	for _, c := range []struct {
		name string
		res  contracts.Resolution
		want []string // Unclear.Entities; rỗng = câu chỉ có MỘT chủ thể, không hỏi lại
	}{
		{"một đã giải + một mơ hồ: chỉ ứng viên của bí danh", contracts.Resolution{Entities: []contracts.EntityRef{ex}, Ambiguous: []contracts.AmbiguousMention{m7}}, []string{"EX-07", "RL-07"}},
		{"đảo vai: lu 02 đã giải + máy 03 mơ hồ: chỉ ứng viên của bí danh", contracts.Resolution{Entities: []contracts.EntityRef{rl}, Ambiguous: []contracts.AmbiguousMention{m3}}, []string{"EX-03", "DZ-03"}},
		{"hai đã giải + một mơ hồ: vẫn chỉ ứng viên của bí danh", contracts.Resolution{Entities: []contracts.EntityRef{rl, e7}, Ambiguous: []contracts.AmbiguousMention{m3}}, []string{"EX-03", "DZ-03"}},
		{"hai mơ hồ: liệt kê ứng viên của cả hai bí danh", contracts.Resolution{Ambiguous: []contracts.AmbiguousMention{m3, m7}}, []string{"EX-03", "DZ-03", "EX-07", "RL-07"}},
		{"hai đã giải (đối chứng): giữ cách liệt kê cũ", contracts.Resolution{Entities: []contracts.EntityRef{ex, rl}}, []string{"EX-03", "RL-02"}},
		{"ba đã giải (đối chứng): giữ cách liệt kê cũ", contracts.Resolution{Entities: []contracts.EntityRef{ex, rl, e7}}, []string{"EX-03", "RL-02", "EX-07"}},
		{"đã giải chứa trong bí danh: một chủ thể, không hỏi lại", contracts.Resolution{Entities: []contracts.EntityRef{ex}, Ambiguous: []contracts.AmbiguousMention{m3}}, nil},
		{"một dự án + một bí danh máy: khác loại, không tranh khe", contracts.Resolution{Entities: []contracts.EntityRef{pa}, Ambiguous: []contracts.AmbiguousMention{m3}}, nil},
	} {
		r := classifyRes(t, c.res)
		if got := ids(r.Unclear.Entities); !slices.Equal(got, c.want) {
			t.Errorf("%s: Unclear.Entities = %v, muốn %v", c.name, got, c.want)
		}
		if multi := len(c.want) > 0; multi != slices.Contains(r.Intent.Missing, "subject") {
			t.Errorf("%s: khe chủ thể phải %s, Missing = %v", c.name, map[bool]string{true: "để trống (nhiều chủ thể)", false: "điền được"}[multi], r.Intent.Missing)
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
		{Kind: contracts.ValueTerm, Raw: "dầu", Start: 6, End: 11, Value: contracts.Value{Kind: contracts.ValueTerm, Text: "fuel"}}}}
	r := s.Classify(in, contracts.Resolution{Entities: []contracts.EntityRef{ent("EX-03", "equipment")}}, testNow)
	if r.Intent.Name != "" || !slices.Equal(r.NeedMeaning, []string{"q"}) {
		t.Errorf("muốn hỏi lại nghĩa số liệu [q], được intent=%q needMeaning=%v", r.Intent.Name, r.NeedMeaning)
	}
}

// Ý định chung không khai chỉ số cho thuật ngữ của câu thì không được chọn để trả lời: nếu chọn, mảnh từ khoá của mọi ý định riêng không
// chỉ số ("hao" của "hao hụt") thành "của ý định chung" và câu được trả lời theo một nghĩa chưa ai khai (soát S4 vòng 3, phát hiện 4).
func TestYDinhChungKhongChiSoThiHoiLai(t *testing.T) {
	yes := func(string) bool { return true }
	slots := []SlotDef{{Name: "subject", From: "entity", Types: []string{"equipment"}, Required: true}, {Name: "period", From: "period", Default: "today"}}
	s, err := Compile([]Def{
		{Name: "g", Route: "SQL_QUERY", Weak: true, Keywords: []string{"bao nhiêu"}, Terms: []string{"fuel"}, Capability: "metrics.read", Slots: slots},
		{Name: "q", Route: "SQL_QUERY", Keywords: []string{"hao hụt"}, Terms: []string{"fuel"}, Capability: "metrics.read", Slots: slots},
	}, yes, yes)
	if err != nil {
		t.Fatal(err)
	}
	s.SetOptions(Options{FocusMarkers: []string{"đúng không"}})
	in := contracts.NormalizedInput{Raw: "EX-03 dầu hao đúng không", Spans: []contracts.Span{
		{Kind: contracts.ValueTerm, Raw: "dầu", Start: 6, End: 11, Value: contracts.Value{Kind: contracts.ValueTerm, Text: "fuel"}}}}
	e := ent("EX-03", "equipment")
	e.Start, e.End = 0, 5 // vị trí tên trong câu: chỉ còn "hao" là chữ phải giải thích
	r := s.Classify(in, contracts.Resolution{Entities: []contracts.EntityRef{e}}, testNow)
	if r.Intent.Name != "" || !slices.Equal(r.NeedMeaning, []string{"q"}) {
		t.Errorf("muốn hỏi lại nghĩa số liệu [q], được intent=%q needMeaning=%v", r.Intent.Name, r.NeedMeaning)
	}
}

// N1 (ADR-0010 bất biến (c)): hai ý định hoà nhau, hoặc ý định khớp từ khoá mà thiếu thuật ngữ, trong khi câu còn một bí danh mơ hồ — kết quả phải
// mang ứng viên của bí danh để câu hỏi lại hỏi được cả hai điều; không có bí danh thì không có ứng viên.
func TestHoaYDinhVaThieuThuatNguGiuUngVienCuaBiDanh(t *testing.T) {
	yes := func(string) bool { return true }
	slots := []SlotDef{{Name: "subject", From: "entity", Types: []string{"equipment"}, Required: true}, {Name: "period", From: "period", Default: "today"}}
	s, err := Compile([]Def{
		{Name: "a", Route: "SQL_QUERY", Keywords: []string{"cấp"}, Terms: []string{"fuel"}, Capability: "metrics.read", Slots: slots},
		{Name: "b", Route: "SQL_QUERY", Keywords: []string{"báo"}, Terms: []string{"fuel"}, Capability: "metrics.read", Slots: slots},
	}, yes, yes)
	if err != nil {
		t.Fatal(err)
	}
	ex, dz := ent("EX-03", "equipment"), ent("DZ-03", "equipment")
	m3 := alias("máy 03", "equipment", ex, dz)
	fuel := contracts.Span{Kind: contracts.ValueTerm, Raw: "dầu", Value: contracts.Value{Kind: contracts.ValueTerm, Text: "fuel"}}
	for _, c := range []struct {
		name   string
		raw    string
		spans  []contracts.Span
		res    contracts.Resolution
		tie    bool // kết quả là hoà ý định (Ambiguous), không phải thiếu thuật ngữ (NeedTerm)
		wantID []string
	}{
		{"hoà ý định + bí danh mơ hồ", "máy 03 cấp báo dầu", []contracts.Span{fuel}, contracts.Resolution{Ambiguous: []contracts.AmbiguousMention{m3}}, true, []string{"EX-03", "DZ-03"}},
		{"thiếu thuật ngữ + bí danh mơ hồ", "máy 03 cấp bao nhiêu", nil, contracts.Resolution{Ambiguous: []contracts.AmbiguousMention{m3}}, false, []string{"EX-03", "DZ-03"}},
		{"hoà ý định, máy đã rõ: không có ứng viên", "EX-03 cấp báo dầu", []contracts.Span{fuel}, contracts.Resolution{Entities: []contracts.EntityRef{ex}}, true, nil},
		{"thiếu thuật ngữ, máy đã rõ: không có ứng viên", "EX-03 cấp bao nhiêu", nil, contracts.Resolution{Entities: []contracts.EntityRef{ex}}, false, nil},
	} {
		r := s.Classify(contracts.NormalizedInput{Raw: c.raw, Spans: c.spans}, c.res, testNow)
		if c.tie && len(r.Ambiguous) != 2 || !c.tie && len(r.NeedTerm) == 0 {
			t.Errorf("%s: không phải kết quả muốn thử: Ambiguous = %v, NeedTerm = %v", c.name, r.Ambiguous, r.NeedTerm)
			continue
		}
		if got := ids(r.Candidates); !slices.Equal(got, c.wantID) {
			t.Errorf("%s: Candidates = %v, muốn %v", c.name, got, c.wantID)
		}
	}
}

// C5 (ADR-0010 bất biến (b), vai mệnh đề): mệnh đề ĐỨNG TRƯỚC mệnh đề hỏi mà nêu một con số là ngữ cảnh — từ khoá của nó (chữ chỉ NGUỒN của con số)
// không quyết định ý định — khi mệnh đề hỏi có chỉ số riêng. Cặp tối thiểu: đổi đúng một thứ (con số ở mệnh đề trước, chỉ số riêng ở mệnh đề hỏi,
// mệnh đề hỏi chỉ là cái đuôi) thì kết quả đổi đúng một nghĩa.
func TestMenhDeNguCanhKhongQuyetDinhYDinh(t *testing.T) {
	yes := func(string) bool { return true }
	slots := []SlotDef{{Name: "subject", From: "entity", Types: []string{"equipment"}, Required: true}, {Name: "period", From: "period", Default: "today"}}
	def := func(name string, weak bool, kw ...string) Def {
		prio := 2
		if weak {
			prio = 1
		}
		return Def{Name: name, Route: "SQL_QUERY", Weak: weak, Keywords: kw, Terms: []string{"fuel"}, Capability: "metrics.read", Slots: slots,
			MetricByTerm: map[string]string{"fuel": name}, Priority: prio}
	}
	s, err := Compile([]Def{def("dung", false, "dùng"), def("phieu", false, "phiếu kho", "xuất kho"), def("chung", true, "bao nhiêu")}, yes, yes)
	if err != nil {
		t.Fatal(err)
	}
	s.SetOptions(Options{FocusMarkers: []string{"đúng không"}, QuantityMarkers: []string{"bao nhiêu"}, NeutralWords: []string{"vậy", "nhưng", "rồi"}})
	ex := ent("EX-03", "equipment")
	for _, c := range []struct {
		name, raw, nums string // nums: các con số (chữ) có trong câu, mỗi chữ là một span số trần
		wantIntent      string // "" = hoà (Ambiguous)
	}{
		{"con số ở mệnh đề trước + chỉ số riêng ở mệnh đề hỏi: mệnh đề hỏi quyết định",
			"phiếu kho ghi 3850 rồi, nhưng hôm nay EX-03 dùng hết bao nhiêu dầu", "3850", "dung"},
		{"đối chứng: mệnh đề trước KHÔNG có con số — dẫn đề vẫn nói chỉ số, hai chỉ số hoà nhau",
			"phiếu kho rồi, nhưng hôm nay EX-03 dùng hết bao nhiêu dầu", "", ""},
		{"đối chứng: mệnh đề hỏi không có chỉ số riêng (chỉ 'bao nhiêu') — chỉ số của mệnh đề trước còn tác dụng",
			"phiếu kho ghi 3850 rồi, nhưng hôm nay EX-03 hết bao nhiêu dầu", "3850", "phieu"},
		{"đối chứng: mệnh đề hỏi chỉ là cái đuôi có nhãn ('phiếu số 912', không con số) — câu hỏi nằm ở mệnh đề trước",
			"EX-03 hôm nay dùng 3850 L dầu, theo phiếu số 912 đúng không", "3850", "dung"},
		{"mệnh đề hỏi chỉ còn cụm xác nhận: câu hỏi áp cho mệnh đề liền trước nó, mệnh đề xa hơn là ngữ cảnh",
			"phiếu kho ghi 3850 rồi, vậy EX-03 hôm nay dùng 3850 L dầu, đúng không", "3850 3850", "dung"},
		{"một mệnh đề: không đổi", "EX-03 hôm nay dùng bao nhiêu dầu, theo phiếu kho", "", ""},
	} {
		in := contracts.NormalizedInput{Raw: c.raw, Spans: []contracts.Span{{Kind: contracts.ValueTerm, Raw: "dầu", Value: contracts.Value{Kind: contracts.ValueTerm, Text: "fuel"}}}}
		from := 0
		for _, n := range strings.Fields(c.nums) {
			i := from + strings.Index(c.raw[from:], n)
			in.Spans = append(in.Spans, contracts.Span{Kind: contracts.ValueNumber, Raw: n, Start: i, End: i + len(n), Value: contracts.Value{Kind: contracts.ValueNumber, Number: n}})
			from = i + len(n)
		}
		e := ex
		e.Start, e.End = strings.Index(c.raw, "EX-03"), strings.Index(c.raw, "EX-03")+5
		if e.Start < 0 {
			e.Start, e.End = 0, 0
		}
		r := s.Classify(in, contracts.Resolution{Entities: []contracts.EntityRef{e}}, testNow)
		if r.Intent.Name != c.wantIntent {
			t.Errorf("%s: ý định = %q (hoà %v), muốn %q", c.name, r.Intent.Name, r.Ambiguous, c.wantIntent)
		}
	}
}
