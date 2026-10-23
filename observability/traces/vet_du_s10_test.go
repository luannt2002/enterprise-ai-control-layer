package traces_test

import (
	"fmt"
	"regexp"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/luannt2002/enterprise-ai-control-layer/core/contracts"
	"github.com/luannt2002/enterprise-ai-control-layer/observability/traces"
)

// V8a — khoảng trống S10: S10 chỉ chép câu trả lời, không nói câu giao có khác bản nháp ở S8 hay không, nên một câu giao bị cắt ở S10 bị đọc thành lỗi S8
// (5 / 5 thẻ S10 của phép đo 23/10). S10 phải ghi số ký tự và băm của câu GIAO, và đối chiếu với bản nháp: băm trùng thì không bước nào đổi chữ.

// deliveredLine dựng dòng S10 mong đợi của câu giao, tính độc lập với mã sản xuất.
func deliveredLine(answer string) string {
	return fmt.Sprintf("câu giao: %d ký tự · băm %s", utf8.RuneCountInString(answer), sha12(answer))
}

var hashIn = regexp.MustCompile(`băm ([0-9a-f]{12})`)

func TestVetS10XuatDauVanTayCauGiaoRouteSoLieu(t *testing.T) {
	turn := realTurn(t, realAsk(t), person("director", ""), "EX-03 hôm nay được cấp bao nhiêu lít dầu?")
	ans := turn.Response.Answer
	steps := traces.NewRecord(turn).Steps
	s10 := stepOf(t, steps, "S10")
	for _, want := range []string{
		"outcome do chặng thực thi và kiểm quyết (RoutePlan.Outcome rỗng)", // chữ cũ còn nguyên
		deliveredLine(ans),
		"câu giao trùng bản nháp ở S8 (băm " + sha12(ans) + "): không bước nào sau S8 đổi chữ",
	} {
		if !strings.Contains(s10.Reason, want) {
			t.Errorf("S10 thiếu %q trong:\n  %s", want, s10.Reason)
		}
	}
	// người đọc so băm S8 với băm S10 bằng mắt: hai băm phải là MỘT
	h8, h10 := hashIn.FindString(stepOf(t, steps, "S8").Reason), hashIn.FindString(s10.Reason)
	if h8 == "" || h8 != h10 {
		t.Errorf("băm S8 %q phải trùng băm S10 %q khi câu giao giữ nguyên bản nháp", h8, h10)
	}
}

// Route tri thức: câu giao là mẫu câu BỌC bản nháp, nên băm khác — S10 nói câu giao chứa nguyên văn bản nháp (băm của bản nháp, trùng S8).
func TestVetS10XuatDauVanTayCauGiaoRouteTriThuc(t *testing.T) {
	turn := realTurn(t, realAsk(t), person("director", ""), "Quy trình xử lý chênh lệch nhiên liệu là gì?")
	ans := turn.Response.Answer
	steps := traces.NewRecord(turn).Steps
	s10 := stepOf(t, steps, "S10")
	want := deliveredLine(ans) + "; câu giao chứa nguyên văn bản nháp ở S8 (" + fmt.Sprintf("%d ký tự · băm %s", utf8.RuneCountInString(sopBody), sha12(sopBody)) + "), phần còn lại do mẫu câu của tenant dựng"
	if !strings.HasSuffix(s10.Reason, want) {
		t.Errorf("S10 của route tri thức phải kết thúc bằng:\n  %s\ngặp:\n  %s", want, s10.Reason)
	}
	if !strings.Contains(stepOf(t, steps, "S8").Reason, "băm "+sha12(sopBody)) {
		t.Errorf("băm bản nháp ở S10 phải là băm bản nháp ở S8: %s", stepOf(t, steps, "S8").Reason)
	}
}

func TestVetS10XuatDauVanTayCauGiaoMauCauVaHybrid(t *testing.T) {
	a := realAsk(t)
	// mẫu câu: hỏi lại, từ chối, chuyển người — câu giao đúng là bản nháp
	for _, c := range []struct {
		pr contracts.Principal
		q  string
	}{
		{person("director", ""), "Máy 03 hôm nay được cấp bao nhiêu dầu?"},
		{person("site_engineer", "PRJ-B"), "EX-03 hôm nay được cấp bao nhiêu dầu?"},
		{person("director", ""), "DZ-03 hôm nay được cấp bao nhiêu dầu?"},
		{person("director", ""), "Máy 07 hôm nay được cấp bao nhiêu dầu?"}, // không có dữ liệu
	} {
		turn := realTurn(t, a, c.pr, c.q)
		s10 := stepOf(t, traces.NewRecord(turn).Steps, "S10")
		ans := turn.Response.Answer
		for _, want := range []string{deliveredLine(ans), "câu giao trùng bản nháp ở S8 (băm " + sha12(ans) + ")"} {
			if !strings.Contains(s10.Reason, want) {
				t.Errorf("%q: S10 thiếu %q trong:\n  %s", c.q, want, s10.Reason)
			}
		}
	}
	// HYBRID: câu giao = câu số liệu + mẩu tri thức; mỗi bản nháp một dòng với tiền tố của nó
	turn := realTurn(t, a, person("director", ""), "EX-03 hôm nay được cấp bao nhiêu dầu, và theo quy trình thì khi lệch phải làm gì?")
	ans := turn.Response.Answer
	metricPart, _, _ := strings.Cut(ans, " Trích văn bản quy trình")
	s10 := stepOf(t, traces.NewRecord(turn).Steps, "S10")
	for _, want := range []string{
		deliveredLine(ans),
		fmt.Sprintf("số liệu: câu giao chứa nguyên văn bản nháp ở S8 (%d ký tự · băm %s), phần còn lại do mẫu câu của tenant dựng", utf8.RuneCountInString(metricPart), sha12(metricPart)),
		fmt.Sprintf("tri thức: câu giao chứa nguyên văn bản nháp ở S8 (%d ký tự · băm %s), phần còn lại do mẫu câu của tenant dựng", utf8.RuneCountInString(sopBody), sha12(sopBody)),
	} {
		if !strings.Contains(s10.Reason, want) {
			t.Errorf("S10 HYBRID thiếu %q trong:\n  %s", want, s10.Reason)
		}
	}
}

// S9 loại bản nháp (model bịa nguồn): câu giao là mẫu thay thế — S10 nói đúng vậy, không kể là "câu giao không chứa bản nháp" như một lỗi của S10.
func TestVetS10BanNhapBiLoaiThiCauGiaoLaMauThayThe(t *testing.T) {
	a := realAsk(t)
	a.Deps.Model = unboundModel{}
	turn := realTurn(t, a, person("director", ""), "Quy trình xử lý chênh lệch nhiên liệu là gì?")
	if turn.Decision.Outcome != "SYSTEM_FAILURE" {
		t.Fatalf("tiền đề: S9 phải loại bản nháp, outcome %s", turn.Decision.Outcome)
	}
	s10 := stepOf(t, traces.NewRecord(turn).Steps, "S10")
	for _, want := range []string{
		deliveredLine(turn.Response.Answer),
		fmt.Sprintf("bản nháp ở S8 (%d ký tự · băm %s) đã bị S9 loại nên không vào câu giao; câu giao là mẫu thay thế của tenant", utf8.RuneCountInString(sopBody), sha12(sopBody)),
	} {
		if !strings.Contains(s10.Reason, want) {
			t.Errorf("S10 thiếu %q trong:\n  %s", want, s10.Reason)
		}
	}
	if strings.Contains(s10.Reason, "KHÔNG chứa") {
		t.Errorf("bản nháp đã bị loại thì không phải lỗi 'câu giao không chứa bản nháp': %s", s10.Reason)
	}
}
