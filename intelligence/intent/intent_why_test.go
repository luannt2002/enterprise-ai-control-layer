package intent

import (
	"fmt"
	"slices"
	"strings"
	"testing"

	"github.com/luannt2002/enterprise-ai-control-layer/core/contracts"
)

// R2-03. Classify ghi giải thích của S4 (Result.Why) tại đúng chỗ quyết định: từ khoá khớp, khớp đủ / một phần, vì sao thắng / bị loại,
// và NGUỒN của từng ứng viên thực thể dùng để hỏi lại.

func dauTerm() contracts.Span {
	return contracts.Span{Kind: contracts.ValueTerm, Raw: "dầu", Value: contracts.Value{Kind: contracts.ValueTerm, Text: "diesel_fuel"}, Confidence: 1}
}

func whyOf(r Result) string {
	return strings.Join(r.Why.Reasons, "\n") + "\n--\n" + strings.Join(r.Why.Rejected, "\n")
}

func TestClassifyGiaiThichKhopDuVaKhopMotPhan(t *testing.T) {
	yes := func(string) bool { return true }
	set, err := Compile([]Def{
		{Name: "issued", Route: "SQL_QUERY", Keywords: []string{"cấp"}, Terms: []string{"diesel_fuel"}, Priority: 20, Capability: "m"},
		{Name: "generic", Route: "SQL_QUERY", Keywords: []string{"bao nhiêu"}, Terms: []string{"diesel_fuel"}, Priority: 10, Weak: true, Capability: "m"},
	}, yes, yes)
	if err != nil {
		t.Fatal(err)
	}
	raw := "được cấp bao nhiêu"

	// không có thuật ngữ: ý định thường khớp MỘT PHẦN (nói thiếu thuật ngữ nào), ý định yếu bị loại vì chỉ có từ khoá chung chung
	r := set.Classify(contracts.NormalizedInput{Raw: raw}, contracts.Resolution{}, testNow)
	if !slices.Equal(r.NeedTerm, []string{"issued"}) {
		t.Fatalf("tiền đề: NeedTerm = %v", r.NeedTerm)
	}
	got := whyOf(r)
	for _, frag := range []string{"issued [“cấp”]", "generic [“bao nhiêu”]", "issued: khớp một phần — từ khoá “cấp” khớp nhưng câu thiếu thuật ngữ diesel_fuel", "generic: ý định yếu", "hỏi lại thuật ngữ cho issued"} {
		if !strings.Contains(got, frag) {
			t.Errorf("thiếu %q trong:\n%s", frag, got)
		}
	}

	// có thuật ngữ: cả hai khớp đủ, ý định ưu tiên cao thắng và ý định kia bị loại KÈM lý do ưu tiên
	r = set.Classify(contracts.NormalizedInput{Raw: raw, Spans: []contracts.Span{dauTerm()}}, contracts.Resolution{}, testNow)
	if r.Intent.Name != "issued" {
		t.Fatalf("tiền đề: ý định %q", r.Intent.Name)
	}
	got = whyOf(r)
	for _, frag := range []string{"khớp đủ: issued (ưu tiên 20", "khớp đủ: generic (ưu tiên 10", "chọn issued: ưu tiên cao nhất (20)", "generic: khớp đủ nhưng bị loại — ưu tiên 10 thấp hơn issued (20)", "luật intent:rules"} {
		if !strings.Contains(got, frag) {
			t.Errorf("thiếu %q trong:\n%s", frag, got)
		}
	}

	// hai ý định cùng ưu tiên cùng số cụm: hoà, hỏi lại — vết nói đúng điều đó
	tie, _ := Compile([]Def{
		{Name: "a", Route: "SQL_QUERY", Keywords: []string{"cấp"}, Priority: 5, Capability: "m"},
		{Name: "b", Route: "SQL_QUERY", Keywords: []string{"báo"}, Priority: 5, Capability: "m"},
	}, yes, yes)
	r = tie.Classify(contracts.NormalizedInput{Raw: "cấp và báo"}, contracts.Resolution{}, testNow)
	if !slices.Equal(r.Ambiguous, []string{"a", "b"}) || !strings.Contains(whyOf(r), "hoà: a, b cùng ưu tiên 5 — số cụm từ khoá khớp không phải tín hiệu nghĩa nên không phân xử thay người hỏi, hỏi lại") {
		t.Errorf("hoà: %v\n%s", r.Ambiguous, whyOf(r))
	}

	// không từ khoá nào khớp
	r = set.Classify(contracts.NormalizedInput{Raw: "kèo nào thơm"}, contracts.Resolution{}, testNow)
	if g := whyOf(r); !strings.Contains(g, "không từ khoá ý định nào khớp") || !strings.Contains(g, "ngoài phạm vi") {
		t.Errorf("không khớp:\n%s", g)
	}
}

func TestClassifyNguonUngVienHoiLai(t *testing.T) {
	originOf := func(r Result, id string) (Origin, bool) {
		for _, o := range r.Why.Origins {
			if o.ID == id {
				return o, true
			}
		}
		return Origin{}, false
	}
	dz, ex, rl := ent("DZ-03", "equipment"), ent("EX-03", "equipment"), ent("RL-02", "equipment")
	rl.Mention = "rl 2"

	// chỉ một cụm mơ hồ: mọi ứng viên đến từ S3, S4 không gộp thêm gì
	only := classifyRes(t, contracts.Resolution{Ambiguous: []contracts.AmbiguousMention{alias("may 3", "equipment", dz, ex)}})
	for _, id := range []string{"DZ-03", "EX-03"} {
		if o, ok := originOf(only, id); !ok || o.From != OriginS3 || !strings.Contains(o.Note, `“may 3”`) {
			t.Errorf("%s phải là ứng viên S3 của cụm %q: %+v (có=%v)", id, "may 3", o, ok)
		}
	}
	if len(only.Why.Origins) != 2 {
		t.Errorf("chỉ hai ứng viên, được %+v", only.Why.Origins)
	}

	// D-OP-11 = b: thực thể rõ + cụm mơ hồ cùng loại tranh một khe — danh sách hỏi lại CHỈ gồm ứng viên của cụm mơ hồ (S3); thực thể rõ chỉ góp vào việc đếm
	// chủ thể nên không vào danh sách và không có bản ghi nguồn (hợp đồng đổi có chủ ý: trước đây S4 gộp cả thực thể rõ, nguồn S4)
	mix := classifyRes(t, contracts.Resolution{Entities: []contracts.EntityRef{rl}, Ambiguous: []contracts.AmbiguousMention{alias("may 3", "equipment", dz, ex)}})
	if len(mix.Unclear.Entities) != 2 {
		t.Fatalf("tiền đề: danh sách hỏi lại chỉ gồm 2 ứng viên của cụm mơ hồ, được %v", ids(mix.Unclear.Entities))
	}
	if o, ok := originOf(mix, "RL-02"); ok {
		t.Errorf("RL-02 là thực thể rõ, không vào danh sách hỏi lại nên không có nguồn: %+v", o)
	}
	for _, id := range []string{"DZ-03", "EX-03"} {
		if o, ok := originOf(mix, id); !ok || o.From != OriginS3 {
			t.Errorf("%s là ứng viên của cụm mơ hồ (S3): %+v (có=%v)", id, o, ok)
		}
	}

	// hai thực thể RÕ cùng loại tranh một khe (giữ cách liệt kê cũ): cả hai vào danh sách và đều là thứ S4 gộp
	rl.Mention, dz.Mention = "rl 2", "dz 3"
	two := classifyRes(t, contracts.Resolution{Entities: []contracts.EntityRef{dz, rl}})
	if len(two.Unclear.Entities) != 2 {
		t.Fatalf("tiền đề: hai thực thể rõ tranh khe, được %v", ids(two.Unclear.Entities))
	}
	for _, id := range []string{"DZ-03", "RL-02"} {
		if o, ok := originOf(two, id); !ok || o.From != OriginS4 || !strings.Contains(o.Note, "gộp") || !strings.Contains(o.Note, "khe subject") {
			t.Errorf("%s là thực thể rõ do S4 gộp: %+v (có=%v)", id, o, ok)
		}
	}

	// ngoài phạm vi số liệu (NeedMeaning): ứng viên cũng ghi nguồn S3
	yes := func(string) bool { return true }
	set, _ := Compile([]Def{{Name: "q", Route: "SQL_QUERY", Keywords: []string{"cấp"}, Terms: []string{"diesel_fuel"}, Capability: "m"}}, yes, yes)
	nm := set.Classify(contracts.NormalizedInput{Raw: "máy 03 dầu", Spans: []contracts.Span{dauTerm()}},
		contracts.Resolution{Ambiguous: []contracts.AmbiguousMention{alias("may 3", "equipment", dz, ex)}}, testNow)
	if len(nm.NeedMeaning) == 0 {
		t.Fatalf("tiền đề: phải hỏi lại nghĩa số liệu: %+v", nm)
	}
	if o, ok := originOf(nm, "EX-03"); !ok || o.From != OriginS3 {
		t.Errorf("NeedMeaning: ứng viên phải ghi nguồn S3: %+v", nm.Why.Origins)
	}
	if g := whyOf(nm); !strings.Contains(g, "hỏi lại nghĩa số liệu") || !strings.Contains(g, "ý định chung") {
		t.Errorf("NeedMeaning phải nói vì sao không trả lời theo ý định chung:\n%s", g)
	}
}

// Một bí danh mơ hồ nhắc nhiều lần không sinh nhiều bản ghi nguồn cho cùng một ứng viên (ở cả đường hỏi lại lẫn đường hỏi lại nghĩa số liệu).
func TestClassifyNguonUngVienKhongGhiLap(t *testing.T) {
	dz, ex := ent("DZ-03", "equipment"), ent("EX-03", "equipment")
	three := []contracts.AmbiguousMention{alias("may 3", "equipment", dz, ex), alias("may 3", "equipment", dz, ex), alias("may 3", "equipment", dz, ex)}
	if r := classifyRes(t, contracts.Resolution{Ambiguous: three}); len(r.Why.Origins) != 2 {
		t.Errorf("ba lần nhắc, hai ứng viên: muốn 2 nguồn, được %+v", r.Why.Origins)
	}
	yes := func(string) bool { return true }
	set, _ := Compile([]Def{{Name: "q", Route: "SQL_QUERY", Keywords: []string{"cấp"}, Terms: []string{"diesel_fuel"}, Capability: "m"}}, yes, yes)
	nm := set.Classify(contracts.NormalizedInput{Raw: "máy 03 dầu", Spans: []contracts.Span{dauTerm()}}, contracts.Resolution{Ambiguous: three}, testNow)
	if len(nm.NeedMeaning) == 0 || len(nm.Why.Origins) != 2 {
		t.Errorf("hỏi lại nghĩa số liệu: muốn 2 nguồn, được %+v (NeedMeaning=%v)", nm.Why.Origins, nm.NeedMeaning)
	}
}

// Mệnh đề hỏi được trích trong cặp “…” KHÔNG thoát nháy kép (thoát kiểu %q làm hỏng mẫu che bí mật ở traces.Sanitize), và nháy kiểu chữ in có sẵn
// trong câu không phá cặp bao.
func TestClassifyTrichMenhDeHoiKhongThoatNhay(t *testing.T) {
	yes := func(string) bool { return true }
	set, _ := Compile([]Def{{Name: "q", Route: "SQL_QUERY", Keywords: []string{"cấp"}, Capability: "m"}}, yes, yes)
	set.SetOptions(Options{FocusMarkers: []string{"đúng không"}})
	r := set.Classify(contracts.NormalizedInput{Raw: `đội báo 4k, vậy được cấp {"api_key": "abc"} và “x” đúng không?`}, contracts.Resolution{}, testNow)
	want := "từ khoá lấy trong mệnh đề hỏi “vậy được cấp {\"api_key\": \"abc\"} và \"x\" đúng không” (mệnh đề chứa cụm xác nhận quyết định ý định, các mệnh đề trước là ngữ cảnh)"
	if len(r.Why.Reasons) == 0 || r.Why.Reasons[0] != want {
		t.Errorf("dòng trích mệnh đề hỏi:\n%q\nmuốn:\n%q", r.Why.Reasons, want)
	}
}

// N2 (soát vòng 2: X24, X26): các SỐ trong lý do của S4 khoá nguyên dòng — số ý định yếu đòi thuật ngữ của câu khi ý định chung không chọn được, và số cụm
// từ khoá của ý định thắng khi hai ý định cùng ưu tiên khớp đủ mà khác số cụm.
func TestClassifyGiaiThichSoLuongYDinhYeuVaCumTuKhoa(t *testing.T) {
	yes := func(string) bool { return true }
	slots := []SlotDef{{Name: "subject", From: "entity", Types: []string{"equipment"}, Required: true}, {Name: "period", From: "period", Default: "today"}}
	in := contracts.NormalizedInput{Raw: "EX-03 dầu đúng không", Spans: []contracts.Span{
		{Kind: contracts.ValueTerm, Raw: "dầu", Start: 6, End: 11, Value: contracts.Value{Kind: contracts.ValueTerm, Text: "fuel"}}}}
	res := contracts.Resolution{Entities: []contracts.EntityRef{ent("EX-03", "equipment")}}
	weak := func(name string) Def {
		return Def{Name: name, Route: "SQL_QUERY", Weak: true, Keywords: []string{"bao nhiêu"}, Terms: []string{"fuel"}, Capability: "metrics.read", Slots: slots, MetricByTerm: map[string]string{"fuel": "issued"}}
	}
	own := Def{Name: "q", Route: "SQL_QUERY", Keywords: []string{"cấp"}, Terms: []string{"fuel"}, Capability: "metrics.read", Slots: slots, MetricByTerm: map[string]string{"fuel": "issued"}}
	const prefix = "có thực thể và thuật ngữ nhưng không từ khoá ý định nào khớp, và câu không trả lời được theo ý định chung (không có đúng một ý định yếu đòi thuật ngữ của câu (có "
	for _, c := range []struct {
		name string
		defs []Def
		want int
	}{{"không có ý định yếu", []Def{own}, 0}, {"hai ý định yếu cùng đòi thuật ngữ", []Def{weak("g1"), weak("g2"), own}, 2}} {
		s, err := Compile(c.defs, yes, yes)
		if err != nil {
			t.Fatal(err)
		}
		s.SetOptions(Options{FocusMarkers: []string{"đúng không"}})
		r := s.Classify(in, res, testNow)
		want := fmt.Sprintf("%s%d)) — hỏi lại nghĩa số liệu: %s", prefix, c.want, strings.Join(r.NeedMeaning, ", "))
		if !slices.Contains(r.Why.Reasons, want) {
			t.Errorf("%s: thiếu dòng nguyên văn %q trong:\n%s", c.name, want, strings.Join(r.Why.Reasons, "\n"))
		}
	}

	// cùng ưu tiên, khác số cụm từ khoá: HOÀ ⇒ hỏi lại (S4AS-03, C4 — số cụm khớp không phải tín hiệu nghĩa; trước đây ý định nhiều cụm hơn thắng và ý định kia bị loại kèm hai
	// con số: hợp đồng đổi có chủ ý theo ADR-0010)
	set, err := Compile([]Def{
		{Name: "a", Route: "SQL_QUERY", Keywords: []string{"cấp", "nhận"}, Priority: 5, Capability: "m"},
		{Name: "b", Route: "SQL_QUERY", Keywords: []string{"báo"}, Priority: 5, Capability: "m"},
	}, yes, yes)
	if err != nil {
		t.Fatal(err)
	}
	r := set.Classify(contracts.NormalizedInput{Raw: "cấp nhận báo"}, contracts.Resolution{}, testNow)
	if !slices.Equal(r.Ambiguous, []string{"a", "b"}) {
		t.Fatalf("tiền đề: hai ý định cùng ưu tiên phải hoà, được %q / %q", r.Intent.Name, r.Ambiguous)
	}
	if want := "hoà: a, b cùng ưu tiên 5 — số cụm từ khoá khớp không phải tín hiệu nghĩa nên không phân xử thay người hỏi, hỏi lại"; !slices.Contains(r.Why.Reasons, want) {
		t.Errorf("thiếu %q trong:\n%s", want, strings.Join(r.Why.Reasons, "\n"))
	}
	if len(r.Why.Rejected) != 0 {
		t.Errorf("hoà thì không ý định nào bị loại, được %q", r.Why.Rejected)
	}
}

// N2 (khoá nguyên dòng): ba lý do của S4 mà seed không có ca thật — khe kỳ thiếu vì pack không có mặc định, chỉ số thiếu vì câu không có thuật ngữ nào chọn được chỉ số,
// và hai ý định tri thức hoà nhau nên không ghép HYBRID.
func TestClassifyGiaiThichKheKyThieuChiSoThieuVaTriThucHoa(t *testing.T) {
	yes := func(string) bool { return true }
	subject := SlotDef{Name: "subject", From: "entity", Types: []string{"equipment"}, Required: true}
	ex := ent("EX-03", "equipment")
	res := contracts.Resolution{Entities: []contracts.EntityRef{ex}}

	// khe kỳ bắt buộc mà câu không nêu mốc và pack không khai mặc định
	noDefault, err := Compile([]Def{{Name: "q", Route: "SQL_QUERY", Keywords: []string{"cấp"}, Capability: "m", Slots: []SlotDef{subject, {Name: "period", From: "period", Required: true}}}}, yes, yes)
	if err != nil {
		t.Fatal(err)
	}
	r := noDefault.Classify(contracts.NormalizedInput{Raw: "EX-03 cấp"}, res, testNow)
	if want := "khe period thiếu: câu không nêu mốc và pack không có mặc định"; !slices.Contains(r.Why.Reasons, want) || !slices.Equal(r.Intent.Missing, []string{"period"}) {
		t.Errorf("thiếu %q (khe thiếu %v) trong:\n%s", want, r.Intent.Missing, strings.Join(r.Why.Reasons, "\n"))
	}

	// ý định chọn chỉ số theo thuật ngữ mà câu không có thuật ngữ nào (và pack không có thuật ngữ hiểu ngầm)
	byTerm, err := Compile([]Def{{Name: "q", Route: "SQL_QUERY", Keywords: []string{"cấp"}, Capability: "m", MetricByTerm: map[string]string{"fuel": "issued"}, Slots: []SlotDef{subject}}}, yes, yes)
	if err != nil {
		t.Fatal(err)
	}
	r = byTerm.Classify(contracts.NormalizedInput{Raw: "EX-03 cấp"}, res, testNow)
	if want := "chỉ số thiếu: ý định q chọn chỉ số theo thuật ngữ mà câu không có thuật ngữ nào trong fuel"; !slices.Contains(r.Why.Reasons, want) || !slices.Equal(r.Intent.Missing, []string{"metric"}) {
		t.Errorf("thiếu %q (khe thiếu %v) trong:\n%s", want, r.Intent.Missing, strings.Join(r.Why.Reasons, "\n"))
	}

	// hai ý định tri thức cùng ưu tiên cùng số cụm từ khoá khớp cùng một ý định số liệu đủ khe: hoà nhau nên KHÔNG ghép HYBRID
	tie, err := Compile([]Def{
		{Name: "q", Route: "SQL_QUERY", Keywords: []string{"cấp"}, Capability: "m", Slots: []SlotDef{subject}},
		{Name: "k1", Route: "RAG", Keywords: []string{"quy trình"}, Priority: 5, Capability: "d"},
		{Name: "k2", Route: "RAG", Keywords: []string{"thủ tục"}, Priority: 5, Capability: "d"},
	}, yes, yes)
	if err != nil {
		t.Fatal(err)
	}
	r = tie.Classify(contracts.NormalizedInput{Raw: "EX-03 cấp quy trình thủ tục"}, res, testNow)
	if r.Knowledge != "" {
		t.Fatalf("tiền đề: hoà thì không ghép HYBRID, được Knowledge = %q", r.Knowledge)
	}
	if want := []string{"k1, k2 (tri thức): khớp cùng nhưng hoà nhau nên không ghép HYBRID"}; !slices.Equal(r.Why.Rejected, want) {
		t.Errorf("ý định bị loại: %q, muốn %q", r.Why.Rejected, want)
	}
}
