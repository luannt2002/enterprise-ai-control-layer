package traces_test

import (
	"reflect"
	"strings"
	"testing"

	"github.com/luannt2002/enterprise-ai-control-layer/core/contracts"
	"github.com/luannt2002/enterprise-ai-control-layer/observability/traces"
)

// S6 của route tri thức (ROADMAP-RAG §6.7): đường truy xuất, số tài liệu / đoạn người hỏi thấy, điểm + hạng của đoạn được giữ, và lý
// do "im" khi không trả đoạn nào — không còn ô "chưa ghi". Route số liệu không mang dòng nào của chặng truy xuất tri thức.
func TestVetDuS6TriThuc(t *testing.T) {
	director := person("director", "")
	trung := stepOf(t, steps(t, director, "Quy trình xử lý chênh lệch nhiên liệu là gì?"), "S6")
	wantAll(t, "S6 trúng", text(trung), "đường truy xuất: từ khoá", "bản tài liệu / ", "đoạn người hỏi thấy", "SOP-FUEL-01", "hạng 1, điểm ")
	if traces.IsNotRecorded(trung.Reason) {
		t.Errorf("S6 tri thức không được để chưa ghi: %q", trung.Reason)
	}
	im := stepOf(t, steps(t, director, "Quy trình cấp phép bay flycam trên công trường là gì?"), "S6")
	wantAll(t, "S6 im", text(im), "không trả đoạn nào", "chủ đề ngoài kho")
	if sql := stepOf(t, steps(t, director, "EX-03 hôm nay được cấp bao nhiêu lít dầu?"), "S6"); strings.Contains(text(sql), "đường truy xuất") {
		t.Errorf("S6 route số liệu không được mang vết truy xuất tri thức: %q", text(sql))
	}
	// HYBRID (soát độc lập 24/10): S6 giữ phần số liệu — xung đột nguồn, hoặc ô "chưa ghi" của kết quả truy vấn (dấu khoảng trống của
	// bảng vẫn bật) — VÀ thêm vết truy xuất của phần tri thức; không phần nào đè phần kia.
	hy := stepOf(t, steps(t, director, "EX-03 hôm nay được cấp bao nhiêu dầu, và theo quy trình thì khi lệch phải làm gì?"), "S6")
	wantAll(t, "S6 HYBRID có xung đột", text(hy), "issued_fuel|EX-03", "tri thức: đường truy xuất: từ khoá")
	hy2 := stepOf(t, steps(t, director, "EX-03 tuần này tiêu thụ bao nhiêu dầu, và theo quy trình thì khi lệch phải làm gì?"), "S6")
	if !traces.IsNotRecorded(hy2.Reason) || !strings.Contains(text(hy2), "tri thức: đường truy xuất") {
		t.Errorf("S6 HYBRID không xung đột: phải giữ ô chưa ghi của số liệu và thêm vết tri thức, được %q", text(hy2))
	}
}

// Mọi chặng của Why đều qua bộ che chữ người hỏi: chặng mới thêm vào Why mà quên ở WithholdQuoted thì chữ của câu hỏi mang bí mật lọt
// vào vết của chặng đó.
func TestCheChuMoiChangCuaWhy(t *testing.T) {
	var w traces.Why
	v := reflect.ValueOf(&w).Elem()
	for i := range v.NumField() {
		if n, ok := v.Field(i).Addr().Interface().(*traces.Notes); ok {
			n.Reasons, n.Rejected = []string{"câu " + contracts.Quote("BI-MAT-1")}, []string{"loại " + contracts.Quote("BI-MAT-2")}
		}
	}
	got := reflect.ValueOf(w.WithholdQuoted())
	for i := range got.NumField() {
		if n, ok := got.Field(i).Interface().(traces.Notes); ok && strings.Contains(strings.Join(append(n.Reasons, n.Rejected...), " "), "BI-MAT") {
			t.Errorf("chặng %s không được che: %+v", got.Type().Field(i).Name, n)
		}
	}
}
