package intent

import (
	"testing"

	"github.com/luannt2002/enterprise-ai-control-layer/core/contracts"
)

// R-06 — câu QUY TRÌNH thuần nêu nhiều thực thể cùng loại không bị hỏi lại "máy hoặc dự án" chỉ vì một từ khoá số liệu yếu: hỏi lại chỉ khi câu có dấu hiệu
// hỏi SỐ LIỆU (cụm hỏi lượng, cụm xác nhận, mốc ngày, hay một mệnh đề riêng chỉ có từ khoá số liệu).
func TestCauQuyTrinhThuanNhieuThucTheKhongHoiLaiOan(t *testing.T) {
	yes := func(string) bool { return true }
	set, err := Compile([]Def{
		{Name: "q", Route: "SQL_QUERY", Keywords: []string{"cấp", "phiếu kho"}, Capability: "metrics.read", Priority: 20,
			Slots: []SlotDef{
				{Name: "subject", From: "entity", Types: []string{"equipment", "project"}, Required: true},
				{Name: "period", From: "period", Default: "today"},
			}},
		{Name: "sop", Route: "RAG", Keywords: []string{"quy trình", "thủ tục", "khi lệch", "phải làm gì"}, Capability: "documents.read", Priority: 25},
		{Name: "greeting", Route: "CLARIFICATION", Keywords: []string{"xin chào"}, Priority: 1},
	}, yes, yes)
	if err != nil {
		t.Fatal(err)
	}
	ex, rl := ent("EX-03", "equipment"), ent("RL-02", "equipment")
	res := contracts.Resolution{Entities: []contracts.EntityRef{ex, rl}}
	in := func(raw string) contracts.NormalizedInput { return contracts.NormalizedInput{Raw: raw} }

	// pack chưa khai dấu hiệu hỏi số liệu nào (SetOptions chưa gọi): không có căn cứ coi là câu thuần quy trình → hỏi lại như A-04
	if r := set.Classify(in("Khi lệch phiếu kho của EX-03 và RL-02 thì phải làm gì?"), res, testNow); r.Intent.Name != "q" {
		t.Errorf("không có dấu hiệu nào khai: ý định = %q, muốn q (hỏi lại, an toàn)", r.Intent.Name)
	}

	set.SetOptions(Options{QuantityMarkers: []string{"bao nhiêu", "định mức"}, FocusMarkers: []string{"đúng không"}})
	for _, c := range []struct {
		raw  string
		want string // ý định thắng: "sop" = câu quy trình thuần (RAG), "q" = hỏi lại về số liệu
		why  string
	}{
		{"Khi lệch phiếu kho của EX-03 và RL-02 thì phải làm gì?", "sop", "một mệnh đề: từ khoá số liệu yếu cạnh từ khoá quy trình"},
		{"Thủ tục xin cấp dầu cho EX-03 và RL-02?", "sop", "một mệnh đề"},
		{"Phiếu kho của EX-03 và RL-02 lệch thì phải làm gì?", "sop", "từ khoá số liệu đứng đầu nhưng cùng mệnh đề với từ khoá quy trình"},
		{"Khi lệch phiếu kho của EX-03 và RL-02, thì phải làm gì?", "sop", "mệnh đề đầu có từ khoá quy trình, mệnh đề sau không có từ khoá số liệu"},
		{"Cấp dầu cho EX-03 và RL-02 cần thủ tục gì?", "sop", "một mệnh đề"},
		{"Quy trình định mức cấp dầu cho EX-03 và RL-02 là gì?", "q", "cụm hỏi lượng 'định mức' (đứng một mình trong câu quy trình)"},
		{"Xin chào, khi lệch phiếu kho của EX-03 và RL-02 thì phải làm gì?", "sop", "lời chào không phải từ khoá số liệu"},
		// có dấu hiệu hỏi số liệu → hỏi lại
		{"Thủ tục xin cấp dầu cho EX-03 và RL-02 là gì, mỗi máy được cấp bao nhiêu?", "q", "cụm hỏi lượng"},
		{"Khi lệch phiếu kho thì phải làm gì, EX-03 và RL-02 được cấp 3850 L đúng không?", "q", "cụm xác nhận"},
		{"EX-03 và RL-02 được cấp thế nào, quy trình xử lý khi lệch là gì?", "q", "mệnh đề riêng chỉ có từ khoá số liệu"},
		{"Cấp dầu cho EX-03 và RL-02. Quy trình xử lý khi lệch là gì?", "q", "câu riêng chỉ có từ khoá số liệu"},
		{"Quy trình xử lý định mức cấp cho EX-03 và RL-02 là gì?", "q", "cụm hỏi lượng 'định mức'"},
	} {
		if r := set.Classify(in(c.raw), res, testNow); r.Intent.Name != c.want {
			t.Errorf("%q (%s): ý định = %q, muốn %q", c.raw, c.why, r.Intent.Name, c.want)
		}
	}
	// ngày hỏng / mốc ngày hợp lệ vẫn hỏi lại dù không có dấu hiệu nào khác (A-04)
	if r := set.Classify(contracts.NormalizedInput{Raw: "Khi lệch phiếu kho của EX-03 và RL-02 thì phải làm gì?", InvalidDates: []string{"31/02"}}, res, testNow); r.Intent.Name != "q" {
		t.Errorf("ngày hỏng: ý định = %q, muốn q", r.Intent.Name)
	}
	dated := contracts.NormalizedInput{Raw: "Khi lệch phiếu kho của EX-03 và RL-02 hôm nay thì phải làm gì?", Spans: []contracts.Span{day("hôm nay", "2026-10-19")}}
	if r := set.Classify(dated, res, testNow); r.Intent.Name != "q" {
		t.Errorf("có mốc ngày: ý định = %q, muốn q", r.Intent.Name)
	}
}

// clausesOf cắt câu thành mệnh đề ở , ; . ? ! theo sau là khoảng trắng hoặc hết câu — "3.850", "0,5" không cắt.
func TestClausesOf(t *testing.T) {
	for _, c := range []struct {
		text string
		want []string
	}{
		{"a, b. c? d", []string{"a", " b", " c", " d"}},
		{"được cấp 3.850 lít, 0,5 khối", []string{"được cấp 3.850 lít", " 0,5 khối"}},
		{"một mệnh đề", []string{"một mệnh đề"}},
		{"hết câu.", []string{"hết câu", ""}},
		{"a,\nb", []string{"a", "\nb"}},
	} {
		got := clausesOf(c.text)
		if len(got) != len(c.want) {
			t.Errorf("clausesOf(%q) = %q, muốn %q", c.text, got, c.want)
			continue
		}
		for i := range got {
			if got[i] != c.want[i] {
				t.Errorf("clausesOf(%q)[%d] = %q, muốn %q", c.text, i, got[i], c.want[i])
			}
		}
	}
}
