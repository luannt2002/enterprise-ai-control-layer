package traces

import (
	"slices"
	"strings"
	"testing"

	"github.com/luannt2002/enterprise-ai-control-layer/core/contracts"
)

// Diff chỉ ra ĐÚNG chặng bị sửa — sửa một chặng nào cũng phải lộ, không chặng nào bị bỏ qua.
func TestDiffMoiChang(t *testing.T) {
	base := Decision{
		Normalized: []string{"code:EX-03=EX-03"}, Entities: []string{"EX-03"}, Intent: "fuel_issued", Slots: []string{"subject=EX-03"},
		Route: "SQL_QUERY", Outcome: "RESOLVED", Evidence: []string{"fact:R-004"}, Verdict: "true|accept", Answer: "3.850 L",
	}
	if d := Diff(base, base); len(d) != 0 {
		t.Fatalf("hai bản giống nhau: Diff = %v, muốn rỗng", d)
	}
	for stage, edit := range map[string]func(*Decision){
		"normalize": func(d *Decision) { d.Normalized = append(d.Normalized, "x") },
		"resolve":   func(d *Decision) { d.Entities = []string{"EX-07"} },
		"intent":    func(d *Decision) { d.Slots = []string{"subject=EX-07"} },
		"route":     func(d *Decision) { d.Route = "RAG" },
		"outcome":   func(d *Decision) { d.Outcome = "SYSTEM_FAILURE" },
		"evidence":  func(d *Decision) { d.Evidence = []string{"fact:R-005"} },
		"verdict":   func(d *Decision) { d.Verdict = "false|fallback|number_without_evidence" },
		"answer":    func(d *Decision) { d.Answer = "3.900 L" },
	} {
		got := base
		got.Normalized, got.Entities, got.Slots, got.Evidence = slices.Clone(base.Normalized), slices.Clone(base.Entities), slices.Clone(base.Slots), slices.Clone(base.Evidence)
		edit(&got)
		if d := Diff(base, got); !slices.Equal(d, []string{stage}) {
			t.Errorf("sửa chặng %s: Diff = %v, muốn [%s]", stage, d, stage)
		}
	}
}

func TestWhyCompactGopDongGiongNhauVaCatDanhSach(t *testing.T) {
	var n Notes
	for range 3 {
		n.Reason("cụm “máy 03” khớp 2 thực thể")
	}
	n.Reason("dòng riêng")
	n.Reason("cụm “máy 03” khớp 2 thực thể")
	n.Reject("loại 1")
	n.Reject("loại 1")
	got := Why{Resolve: n}.Compact(10, 1000).Resolve
	if want := []string{"cụm “máy 03” khớp 2 thực thể (×4)", "dòng riêng"}; !slices.Equal(got.Reasons, want) {
		t.Errorf("Reasons: %q, muốn %q", got.Reasons, want)
	}
	if want := []string{"loại 1 (×2)"}; !slices.Equal(got.Rejected, want) {
		t.Errorf("Rejected: %q, muốn %q", got.Rejected, want)
	}
	// trần số ý: các ý khác nhau vượt trần bị cắt, ý cuối nói còn bao nhiêu ý không in
	var many Notes
	for i := range 7 {
		many.Reason("ý %d", i)
	}
	c := Why{Context: many}.Compact(5, 1000).Context
	if len(c.Reasons) != 6 || c.Reasons[4] != "ý 4" || c.Reasons[5] != "… và 2 ý khác không in (trần 5 ý mỗi danh sách)" {
		t.Errorf("trần số ý: %q", c.Reasons)
	}
	// danh sách rỗng giữ rỗng (Notes không ghi vẫn là "chưa ghi", không thành "đã ghi")
	if e := (Why{}).Compact(5, 1000); e.Route.Recorded() || e.Route.Reasons != nil || e.Route.Rejected != nil {
		t.Errorf("Notes rỗng phải giữ rỗng: %+v", e.Route)
	}
	// Compact không sửa bản gốc
	if len(n.Reasons) != 5 {
		t.Errorf("Compact sửa Notes gốc: %q", n.Reasons)
	}
}

func TestWhyWithholdQuotedChiThayChuCuaNguoiHoi(t *testing.T) {
	var n Notes
	n.Reason("từ khoá lấy trong mệnh đề hỏi %s (mệnh đề chứa cụm xác nhận)", contracts.Quote(`đuôi bí mật" đúng không`))
	n.Reason("cụm %s khớp %s, khoá %s", contracts.Quote("máy 03"), "EX-03", contracts.Quote("may 3"))
	n.Reject("chữ %s chưa được giải thích", contracts.Quote("xài"))
	n.Reason("dòng không có chữ của người hỏi")
	w := Why{Intent: n}
	got := w.WithholdQuoted().Intent
	want := Notes{
		Reasons: []string{
			"từ khoá lấy trong mệnh đề hỏi “đã che” (mệnh đề chứa cụm xác nhận)",
			"cụm “đã che” khớp EX-03, khoá “đã che”",
			"dòng không có chữ của người hỏi",
		},
		Rejected: []string{"chữ “đã che” chưa được giải thích"},
	}
	if !slices.Equal(got.Reasons, want.Reasons) || !slices.Equal(got.Rejected, want.Rejected) {
		t.Errorf("WithholdQuoted:\n%q\n%q\nmuốn:\n%q\n%q", got.Reasons, got.Rejected, want.Reasons, want.Rejected)
	}
	if e := (Why{}).WithholdQuoted(); e.Route.Reasons != nil {
		t.Errorf("Notes rỗng phải giữ rỗng: %+v", e.Route)
	}
	if !strings.Contains(w.Intent.Reasons[0], "đuôi bí mật") {
		t.Errorf("WithholdQuoted không được sửa bản gốc: %q", w.Intent.Reasons)
	}
}

// N3 (soát vòng 2): trần theo ĐỘ DÀI từng dòng, không chỉ số dòng. Dòng dài hơn trần bị cắt ở đúng số KÝ TỰ (không cắt giữa một ký tự nhiều byte)
// kèm chữ báo đã cắt; dòng không vượt trần giữ nguyên; các dòng giống hệt nhau vẫn gộp (×N) trước khi cắt.
func TestWhyCompactCatDongDai(t *testing.T) {
	long := strings.Repeat("ệ", 50)
	var n Notes
	n.Reason("ngắn")
	n.Reason("%s", long)
	n.Reason("%s", long)
	n.Reason("%s", long+"đuôi khác")
	n.Reject("%s", long)
	w := Why{Route: n}.Compact(10, 20)
	cut := strings.Repeat("ệ", 20) + "… (cắt: dòng dài quá 20 ký tự)"
	if want := []string{"ngắn", cut + " (×2)", cut}; !slices.Equal(w.Route.Reasons, want) {
		t.Errorf("Reasons: %q, muốn %q", w.Route.Reasons, want)
	}
	if want := []string{cut}; !slices.Equal(w.Route.Rejected, want) {
		t.Errorf("Rejected: %q, muốn %q", w.Route.Rejected, want)
	}
	// đúng bằng trần: giữ nguyên, không có chữ báo cắt
	eq := Why{Route: Notes{Reasons: []string{strings.Repeat("ệ", 20)}}}.Compact(10, 20)
	if want := []string{strings.Repeat("ệ", 20)}; !slices.Equal(eq.Route.Reasons, want) {
		t.Errorf("dòng đúng bằng trần: %q", eq.Route.Reasons)
	}
	// maxRunes <= 0: không cắt theo độ dài
	if un := (Why{Route: Notes{Reasons: []string{long}}}).Compact(10, 0); !slices.Equal(un.Route.Reasons, []string{long}) {
		t.Errorf("maxRunes = 0 phải giữ nguyên dòng: %q", un.Route.Reasons)
	}
	// bản gốc không bị sửa
	if len([]rune(n.Reasons[1])) != 50 {
		t.Errorf("Compact sửa Notes gốc: %d ký tự", len([]rune(n.Reasons[1])))
	}
}
