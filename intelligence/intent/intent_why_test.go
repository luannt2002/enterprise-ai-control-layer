package intent

import (
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
	for _, frag := range []string{"issued [cấp]", "generic [bao nhiêu]", "issued: khớp một phần — từ khoá cấp khớp nhưng câu thiếu thuật ngữ diesel_fuel", "generic: ý định yếu", "hỏi lại thuật ngữ cho issued"} {
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
	if !slices.Equal(r.Ambiguous, []string{"a", "b"}) || !strings.Contains(whyOf(r), "hoà: a, b cùng ưu tiên 5 và cùng 1 cụm từ khoá") {
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

	// thực thể rõ + cụm mơ hồ cùng loại tranh một khe: ứng viên của cụm mơ hồ vẫn từ S3, thực thể rõ là thứ S4 gộp thêm
	mix := classifyRes(t, contracts.Resolution{Entities: []contracts.EntityRef{rl}, Ambiguous: []contracts.AmbiguousMention{alias("may 3", "equipment", dz, ex)}})
	if len(mix.Unclear.Entities) != 3 {
		t.Fatalf("tiền đề: S4 phải gộp 3 thực thể tranh khe, được %v", ids(mix.Unclear.Entities))
	}
	if o, ok := originOf(mix, "RL-02"); !ok || o.From != OriginS4 || !strings.Contains(o.Note, "gộp") || !strings.Contains(o.Note, "khe subject") {
		t.Errorf("RL-02 là thực thể rõ do S4 gộp thêm: %+v (có=%v)", o, ok)
	}
	for _, id := range []string{"DZ-03", "EX-03"} {
		if o, ok := originOf(mix, id); !ok || o.From != OriginS3 {
			t.Errorf("%s là ứng viên của cụm mơ hồ (S3) dù S4 giữ trong danh sách: %+v (có=%v)", id, o, ok)
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
