package pipeline

import (
	"strings"
	"testing"

	"github.com/luannt2002/enterprise-ai-control-layer/core/tenant"
)

func askWithTemplates(t map[string]string) *Ask {
	return &Ask{RT: &tenant.Runtime{Templates: t}}
}

// S2-1: giá trị của một biến (văn bản do model viết) chứa "{biến khác}" không được bị thay theo thứ
// tự duyệt map. Cùng đầu vào phải ra đúng MỘT câu, chạy bao nhiêu lần cũng vậy.
func TestFragmentGiaTriChuaBienKhongBiThay(t *testing.T) {
	a := askWithTemplates(map[string]string{"knowledge_answer": " Trích ({title}): {procedure}"})
	vars := map[string]string{"title": "SOP-1", "procedure": "Lập biên bản {title} trong vòng 24 giờ."}
	want := " Trích (SOP-1): Lập biên bản {title} trong vòng 24 giờ."
	seen := map[string]int{}
	for range 300 {
		seen[a.fragment("knowledge_answer", vars)]++
	}
	if len(seen) != 1 || seen[want] != 300 {
		t.Errorf("300 lần dựng cùng một mẫu phải ra đúng một câu %q, được %d câu khác nhau: %v", want, len(seen), seen)
	}
}

// S2-1: biến mẫu trong giá trị bị XOÁ ở lượt dọn cuối làm hai số dính thành một số mới
// ("3{reason}850" → "3850") — số mà bước S9 (kiểm trên văn bản trước khi dựng) chưa từng thấy.
func TestFragmentKhongGhepSoQuaBienMau(t *testing.T) {
	a := askWithTemplates(map[string]string{"knowledge_answer": "{procedure}"})
	got := a.fragment("knowledge_answer", map[string]string{"procedure": "bước 3{reason}850 lưu biên bản"})
	if strings.Contains(got, "3850") {
		t.Errorf("câu dựng ghép hai số thành 3850: %q", got)
	}
	if got != "bước 3{reason}850 lưu biên bản" {
		t.Errorf("văn bản của model phải giữ nguyên từng chữ, được %q", got)
	}
}

// Hành vi cũ phải giữ: biến có giá trị thì thay, biến được phép mà không có giá trị thì thành rỗng,
// mẫu giữ nguyên khoảng trắng đầu/cuối (mẩu nối sau dấu chấm), render cắt hai đầu.
func TestFragmentGiuHanhViCu(t *testing.T) {
	a := askWithTemplates(map[string]string{
		"metric_total": "{metric_label} của {entity_name}: {value} {unit}.{approx_note}",
		"approx_note":  " Có ước lượng.",
	})
	vars := map[string]string{"metric_label": "Dầu kho xuất", "entity_name": "Máy xúc EX-03", "value": "3.850", "unit": "L"}
	if got, want := a.render("metric_total", vars), "Dầu kho xuất của Máy xúc EX-03: 3.850 L."; got != want {
		t.Errorf("biến vắng phải thành rỗng: %q, muốn %q", got, want)
	}
	vars["approx_note"] = a.RT.Templates["approx_note"]
	if got, want := a.fragment("metric_total", vars), "Dầu kho xuất của Máy xúc EX-03: 3.850 L. Có ước lượng."; got != want {
		t.Errorf("%q, muốn %q", got, want)
	}
	if got := a.fragment("approx_note", nil); got != " Có ước lượng." {
		t.Errorf("mẩu phải giữ dấu cách đầu: %q", got)
	}
	if got := a.render("approx_note", nil); got != "Có ước lượng." {
		t.Errorf("render phải cắt hai đầu: %q", got)
	}
}
