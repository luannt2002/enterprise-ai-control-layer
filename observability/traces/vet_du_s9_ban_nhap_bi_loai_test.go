package traces_test

import (
	"fmt"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/luannt2002/enterprise-ai-control-layer/core/contracts"
	"github.com/luannt2002/enterprise-ai-control-layer/observability/traces"
)

// V8a — S9 không xuất bản nháp bị loại: người đọc thấy lý do loại (các vi phạm) mà không thấy bản nháp nào bị loại, nên không biết S9 loại đúng thứ S8 đã dựng hay
// loại nhầm. S9 phải ghi số ký tự và băm của bản bị loại, cạnh lý do loại — băm đó là băm S8 đã ghi, người đọc so bằng mắt.

func rejectedLine(prefix string, runes int, hash string) string {
	return fmt.Sprintf("%sbản nháp bị loại: %d ký tự · băm %s (bản nháp ở S8; lý do loại: các vi phạm nêu ở lý do)", prefix, runes, hash)
}

func TestVetS9BanNhapBiLoaiTriThuc(t *testing.T) {
	a := realAsk(t)
	a.Deps.Model = unboundModel{} // model bịa nguồn: S9 phải chặn
	turn := realTurn(t, a, person("director", ""), "Quy trình xử lý chênh lệch nhiên liệu là gì?")
	if turn.Decision.Outcome != "SYSTEM_FAILURE" {
		t.Fatalf("tiền đề: S9 phải loại bản nháp, outcome %s", turn.Decision.Outcome)
	}
	steps := traces.NewRecord(turn).Steps
	s9 := stepOf(t, steps, "S9")
	want := []string{rejectedLine("", utf8.RuneCountInString(sopBody), sha12(sopBody))}
	if !slices.Equal(s9.Rejected, want) {
		t.Errorf("S9 phải ghi bản nháp bị loại thay cho ô trống:\n  muốn: %q\n  được: %q", want, s9.Rejected)
	}
	if !regexp.MustCompile(`^\S+/\S+ \[S[1-4]\]: `).MatchString(s9.Reason) { // lý do loại (các vi phạm: validator/mã [mức]: chi tiết) vẫn nằm ở Reason
		t.Errorf("S9 phải còn lý do loại: %q", s9.Reason)
	}
	// bản nháp bị S9 loại không phải nhánh thoát sớm: S8 không có dòng "kết cục nhánh đã chọn" (kết cục ấy là của S9, ở S10)
	if s8 := stepOf(t, steps, "S8"); strings.Contains(s8.Reason, "kết cục nhánh") || strings.Contains(s8.Reason, "thoát sớm") || s8.Decision != "model diễn đạt" {
		t.Errorf("S8 của lượt bị S9 loại: %+v", s8)
	}
	// cùng băm với bản nháp ở S8: người đọc nối được hai bước
	if !strings.Contains(stepOf(t, steps, "S8").Reason, "băm "+sha12(sopBody)) {
		t.Errorf("băm bản nháp bị loại phải là băm bản nháp ở S8: %s", stepOf(t, steps, "S8").Reason)
	}
}

// Route số liệu: bản nháp dựng bằng mẫu mà S9 loại (mẫu có con số không có trong bằng chứng). S9 ghi đúng số ký tự và băm S8 đã ghi.
func TestVetS9BanNhapBiLoaiRouteSoLieu(t *testing.T) {
	a := realAsk(t)
	a.RT.Templates["metric_total"] += " Định mức thêm 12345 lít."
	turn := realTurn(t, a, person("director", ""), "EX-03 hôm nay được cấp bao nhiêu lít dầu?")
	if turn.Decision.Outcome != "SYSTEM_FAILURE" || turn.Plan.Route != contracts.RouteSQLQuery {
		t.Fatalf("tiền đề: S9 phải loại bản nháp mẫu, route %s outcome %s", turn.Plan.Route, turn.Decision.Outcome)
	}
	steps := traces.NewRecord(turn).Steps
	m := regexp.MustCompile(`: (\d+) ký tự · băm ([0-9a-f]{12}) · đầu`).FindStringSubmatch(stepOf(t, steps, "S8").Reason)
	if m == nil {
		t.Fatalf("tiền đề: S8 phải ghi bản nháp: %s", stepOf(t, steps, "S8").Reason)
	}
	runes, _ := strconv.Atoi(m[1])
	if s9 := stepOf(t, steps, "S9"); !slices.Equal(s9.Rejected, []string{rejectedLine("", runes, m[2])}) || s9.Status != traces.StatusDeviant {
		t.Errorf("S9 phải ghi bản nháp bị loại trùng S8 (%s ký tự, băm %s): %+v", m[1], m[2], s9)
	}
	if s10 := stepOf(t, steps, "S10"); !strings.Contains(s10.Reason, "đã bị S9 loại nên không vào câu giao") {
		t.Errorf("S10: %s", s10.Reason)
	}
}

// HYBRID: phần tri thức bị loại, phần số liệu giữ nguyên (outcome vẫn RESOLVED): dòng bị loại mang tiền tố "tri thức: ".
func TestVetS9BanNhapBiLoaiHybrid(t *testing.T) {
	a := realAsk(t)
	a.Deps.Model = unboundModel{}
	turn := realTurn(t, a, person("director", ""), "EX-03 hôm nay được cấp bao nhiêu dầu, và theo quy trình thì khi lệch phải làm gì?")
	if turn.Plan.Route != contracts.RouteHybrid {
		t.Fatalf("tiền đề: route %s", turn.Plan.Route)
	}
	steps := traces.NewRecord(turn).Steps
	want := []string{rejectedLine("tri thức: ", utf8.RuneCountInString(sopBody), sha12(sopBody))}
	if s9 := stepOf(t, steps, "S9"); !slices.Equal(s9.Rejected, want) {
		t.Errorf("S9 HYBRID:\n  muốn: %q\n  được: %q", want, s9.Rejected)
	}
}

// Lượt không bị loại: không có dòng "bản nháp bị loại" nào ở S9. Bản dựng tay (không mang Why) vẫn ghi ô trống như trước.
func TestVetS9KhongBiLoaiThiKhongCoDong(t *testing.T) {
	for _, q := range []string{"Quy trình xử lý chênh lệch nhiên liệu là gì?", "EX-03 hôm nay được cấp bao nhiêu lít dầu?"} {
		s9 := stepOf(t, traces.NewRecord(realTurn(t, realAsk(t), person("director", ""), q)).Steps, "S9")
		if slices.ContainsFunc(s9.Rejected, func(x string) bool { return strings.Contains(x, traces.RejectedDraft) }) {
			t.Errorf("%q: S9 không loại bản nháp nào mà vết nói có: %q", q, s9.Rejected)
		}
	}
	turn := realTurn(t, realAsk(t), person("director", ""), "Quy trình xử lý chênh lệch nhiên liệu là gì?")
	turn.Decision.Why = traces.Why{}
	turn.Decision.Verdict = "false|fallback|NUM_NOT_IN_EVIDENCE"
	turn.Response.Verdict = contracts.Verdict{Pass: false, Action: contracts.ActionFallback}
	if s9 := stepOf(t, traces.NewRecord(turn).Steps, "S9"); len(s9.Rejected) != 1 || !traces.IsNotRecorded(s9.Rejected[0]) {
		t.Errorf("không có Why thì phải ghi ô trống: %q", s9.Rejected)
	}
}
