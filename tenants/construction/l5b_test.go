package construction_test

import (
	"strings"
	"testing"

	"github.com/luannt2002/enterprise-ai-control-layer/core/contracts"
)

// Lát L5b: hai câu khoá chỗ mà đột biến từng sống sót (đo 24/10). Kết cục của cả hai giống hệt mốc 6fb1e90.

// F-030: danh từ đếm của pack ("xe", lexicon.yaml count_nouns) tới được bộ dò ngày dd-mm của câu xác nhận (ask.go gọi a.RT.Normalizer.DashDateAt):
// "10-12 xe" là số đếm, không phải ngày 10/12 — câu vẫn được trả số liệu theo ý định kho xuất, không bị hỏi lại mốc thời gian.
func TestL5bSoDemCuaPackKhongPhaiNgay(t *testing.T) {
	q := "EX-03 hôm nay được cấp 3850 L cho 10-12 xe đúng không?"
	r := ask(t, newAsk(t), user("director", ""), q).Response
	if r.Route != contracts.RouteSQLQuery || r.Outcome != contracts.OutcomeResolved || r.Intent != "fuel_issued" || r.Clarification != nil ||
		!strings.Contains(r.Answer, "3.850 L") || strings.Contains(r.Answer, "mốc thời gian") {
		t.Errorf("%q: muốn trả số kho xuất 3.850 L (SQL_QUERY / RESOLVED / fuel_issued), không hỏi lại mốc thời gian; được [%s/%s/%s] %q",
			q, r.Route, r.Outcome, r.Intent, r.Answer)
	}
}

// S4CS-02 (đột biến S209 của scripts/dot-bien/n45-y-dinh-s4.sh): giá trị đã nêu xét trên CẢ mệnh đề, không chỉ vế trước từ đóng vế "thì" —
// "Nhận xét định mức tiêu thụ bao nhiêu thì hợp lý với 3782 L dầu …" nêu sẵn 3782 L ở vế sau "thì" nên là lời nhờ đánh giá: hỏi lại kèm nhãn nghĩa
// đánh giá chưa hỗ trợ, không trả số.
func TestL5bGiaTriDaNeuXetCaMenhDe(t *testing.T) {
	askBack(t, "Nhận xét định mức tiêu thụ bao nhiêu thì hợp lý với 3782 L dầu của EX-03 hôm nay", notEval)
}
