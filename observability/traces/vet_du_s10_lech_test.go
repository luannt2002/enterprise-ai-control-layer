package traces_test

import (
	"strings"
	"testing"

	"github.com/luannt2002/enterprise-ai-control-layer/observability/traces"
)

// V8a — câu giao KHÔNG chứa nguyên văn bản nháp mà S9 đã nhận là một bước sau S9 đã đổi chữ: lỗi của S10 (hay sau nó), không phải của S8. Dòng S10 đã nói vậy; S10 còn phải
// bị đánh dấu LỆCH để "BƯỚC LỆCH ĐẦU TIÊN" chỉ đúng S10 — trước đây lượt như thế ra "Không có bước lệch" hay chỉ S6.
func TestVetS10CauGiaoKhongChuaBanNhapThiLech(t *testing.T) {
	const q = "Quy trình xử lý chênh lệch nhiên liệu là gì?" // route tri thức: S6 ok, không xung đột nên không bước nào khác lệch
	healthy := traces.NewRecord(realTurn(t, realAsk(t), person("director", ""), q))
	if s10 := stepOf(t, healthy.Steps, "S10"); s10.Status != traces.StatusOK || !strings.Contains(traces.FormatTable(healthy), "Không có bước lệch.") {
		t.Fatalf("tiền đề: lượt khoẻ không có bước lệch: %+v", s10)
	}
	// lỗi gieo: dòng S10 như luồng hỏi sẽ ghi nếu một bước sau S9 cắt câu giao (đột biến thật nằm ở ask.go; đây là dựng tay đúng dòng đó)
	turn := realTurn(t, realAsk(t), person("director", ""), q)
	turn.Decision.Why.Respond.Reason("câu giao KHÔNG chứa nguyên văn bản nháp ở S8 (121 ký tự · băm a80f088044c4) dù S9 không loại nó")
	rec := traces.NewRecord(turn)
	if s10 := stepOf(t, rec.Steps, "S10"); s10.Status != traces.StatusDeviant {
		t.Errorf("S10 phải lệch khi câu giao không chứa bản nháp đã qua S9: %+v", s10)
	}
	if !strings.Contains(traces.FormatTable(rec), "BƯỚC LỆCH ĐẦU TIÊN: S10 respond") {
		t.Errorf("bước lệch đầu tiên phải là S10:\n%s", traces.FormatTable(rec))
	}
	// bản nháp bị S9 loại thì câu giao không chứa nó là đúng, không phải lỗi: S10 không lệch vì lý do này
	if s10 := stepOf(t, traces.NewRecord(realTurn(t, realAsk(t), person("site_engineer", "PRJ-A"), q)).Steps, "S10"); s10.Status != traces.StatusDeviant || strings.Contains(s10.Reason, "KHÔNG chứa") {
		t.Errorf("truy xuất rỗng: S10 lệch vì outcome INSUFFICIENT_EVIDENCE, không vì dấu vân tay: %+v", s10)
	}
}
