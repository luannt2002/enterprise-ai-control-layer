package traces_test

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"slices"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/luannt2002/enterprise-ai-control-layer/core/contracts"
	"github.com/luannt2002/enterprise-ai-control-layer/observability/traces"
)

// V8a — khoảng trống S8 của phép đo "người chỉ đọc vết": S8 chỉ ghi "model diễn đạt" / "mẫu câu của tenant X" kèm outcome, còn chữ của bản nháp chỉ hiện ở S10, nên
// lỗi ở S10 (câu giao bị cắt) không phân biệt được với lỗi ở S8. S8 phải ghi bản nháp NHƯ ĐƯỢC SINH: cách sinh, số ký tự, băm ngắn và đoạn đầu.
//
// Mọi giá trị mong đợi dưới đây được tính ĐỘC LẬP với mã sản xuất (băm bằng crypto/sha256, đếm bằng utf8, đoạn đầu cắt tay) từ chữ mà lượt thật đã
// trả — hàm sản xuất sai một chỗ (đếm byte thay vì ký tự, băm chữ khác, cắt sai độ dài) thì dòng không khớp.

// sha12 là 12 ký tự hex đầu của SHA-256.
func sha12(s string) string {
	sum := sha256.Sum256([]byte(s))
	return hex.EncodeToString(sum[:])[:12]
}

// head80 là đoạn đầu: tối đa 80 ký tự, kèm "…" khi cắt.
func head80(s string) string {
	if r := []rune(s); len(r) > 80 {
		return string(r[:80]) + "…"
	}
	return s
}

// draftLine dựng dòng S8 mong đợi của một bản nháp.
func draftLine(prefix, how, text string) string {
	return fmt.Sprintf("%sbản nháp %s: %d ký tự · băm %s · đầu %s", prefix, how, utf8.RuneCountInString(text), sha12(text), contracts.Quote(head80(text)))
}

// sopBody là chữ của đoạn tài liệu mà realAsk nạp (model tất định trả nguyên văn đoạn này làm bản nháp).
const sopBody = "1. Khi lượng xuất kho và lượng tiêu thụ đo được lệch quá dung sai, tổ trưởng kho lập biên bản đối soát trong vòng 24 giờ."

func TestVetS8XuatBanNhapRouteSoLieu(t *testing.T) {
	turn := realTurn(t, realAsk(t), person("director", ""), "EX-03 hôm nay được cấp bao nhiêu lít dầu?")
	ans := turn.Response.Answer
	if utf8.RuneCountInString(ans) <= 80 {
		t.Fatalf("tiền đề: câu trả lời phải dài hơn 80 ký tự để đoạn đầu bị cắt, được %q", ans)
	}
	s8 := stepOf(t, traces.NewRecord(turn).Steps, "S8")
	if want := draftLine("", "dựng bằng mẫu câu metric_total (không gọi model)", ans); !strings.Contains(s8.Reason, want) {
		t.Errorf("S8 phải ghi bản nháp như được sinh:\n  muốn: %s\n  được: %s", want, s8.Reason)
	}
	// những gì S8 đã nói từ trước vẫn còn (test cũ khoá chữ "không cần model")
	if !strings.Contains(s8.Reason, "không cần model") || s8.Decision != "mẫu câu của tenant metric_total" || traces.IsNotRecorded(s8.Reason) {
		t.Errorf("S8: %+v", s8)
	}
}

// Câu trả lời ngắn: đoạn đầu là nguyên câu, không có "…". Hai route kết thúc sớm bằng mẫu (hỏi lại, từ chối) cũng có bản nháp ở S8.
func TestVetS8XuatBanNhapRouteMauCau(t *testing.T) {
	a := realAsk(t)
	for _, c := range []struct {
		name string
		pr   contracts.Principal
		q    string
	}{
		{"từ chối vì ngoài phạm vi", person("site_engineer", "PRJ-B"), "EX-03 hôm nay được cấp bao nhiêu dầu?"},
		{"hỏi lại vì bí danh mơ hồ", person("director", ""), "Máy 03 hôm nay được cấp bao nhiêu dầu?"},
		{"chuyển người vì nguồn ngang thẩm quyền", person("director", ""), "DZ-03 hôm nay được cấp bao nhiêu dầu?"},
	} {
		t.Run(c.name, func(t *testing.T) {
			turn := realTurn(t, a, c.pr, c.q)
			s8 := stepOf(t, traces.NewRecord(turn).Steps, "S8")
			want := draftLine("", "dựng bằng mẫu câu "+turn.Plan.Template+" (không gọi model)", turn.Response.Answer)
			if !strings.Contains(s8.Reason, want) {
				t.Errorf("S8 phải ghi bản nháp như được sinh:\n  muốn: %s\n  được: %s", want, s8.Reason)
			}
		})
	}
	// ngắn hơn 80 ký tự: không "…"
	short := realTurn(t, a, person("site_engineer", "PRJ-B"), "EX-03 hôm nay được cấp bao nhiêu dầu?").Response.Answer
	if utf8.RuneCountInString(short) > 80 || strings.Contains(draftLine("", "x", short), "…") {
		t.Fatalf("tiền đề: câu từ chối phải ngắn hơn 80 ký tự: %q", short)
	}
}

// Route tri thức: model diễn đạt — S8 ghi tên model, số ký tự, băm và đoạn đầu của bản nháp (chữ của đoạn tài liệu), không còn ô trống "chưa ghi".
func TestVetS8XuatBanNhapRouteTriThuc(t *testing.T) {
	turn := realTurn(t, realAsk(t), person("director", ""), "Quy trình xử lý chênh lệch nhiên liệu là gì?")
	if !strings.Contains(turn.Response.Answer, sopBody) {
		t.Fatalf("tiền đề: câu giao phải chứa bản nháp %q, được %q", sopBody, turn.Response.Answer)
	}
	s8 := stepOf(t, traces.NewRecord(turn).Steps, "S8")
	if want := draftLine("", "do model extractive diễn đạt", sopBody); s8.Decision != "model diễn đạt" || s8.Reason != want {
		t.Errorf("S8 của route tri thức:\n  muốn: %s\n  được: %s (%s)", want, s8.Reason, s8.Decision)
	}
	if slices.ContainsFunc(s8.Rejected, traces.IsNotRecorded) {
		t.Errorf("S8 còn chỗ trống ở phương án bị loại: %q", s8.Rejected)
	}
}

// HYBRID: hai bản nháp, mỗi bản gắn tiền tố của phần đó như S7 ("số liệu: " mẫu câu, "tri thức: " model), cùng nằm ở S8.
func TestVetS8XuatBanNhapHybrid(t *testing.T) {
	turn := realTurn(t, realAsk(t), person("director", ""), "EX-03 hôm nay được cấp bao nhiêu dầu, và theo quy trình thì khi lệch phải làm gì?")
	if turn.Plan.Route != contracts.RouteHybrid || turn.Decision.Outcome != "RESOLVED" {
		t.Fatalf("tiền đề: route %s outcome %s", turn.Plan.Route, turn.Decision.Outcome)
	}
	ans := turn.Response.Answer
	metricPart, _, _ := strings.Cut(ans, " Trích văn bản quy trình")
	s8 := stepOf(t, traces.NewRecord(turn).Steps, "S8")
	for _, want := range []string{
		draftLine("số liệu: ", "dựng bằng mẫu câu metric_total (không gọi model)", metricPart),
		draftLine("tri thức: ", "do model extractive diễn đạt", sopBody),
	} {
		if !strings.Contains(s8.Reason, want) {
			t.Errorf("S8 HYBRID thiếu dòng:\n  %s\ntrong:\n  %s", want, s8.Reason)
		}
	}
}

// Bí mật nằm vắt qua chỗ cắt đoạn đầu: làm sạch TRƯỚC khi cắt, nên không còn mảnh đầu nào của nó trong dòng S8 (cắt trước thì "sk-" cộng vài ký tự không còn
// khớp mẫu che của traces.Sanitize — mẫu đòi ít nhất 16 ký tự sau "sk-" — và lọt vào vết).
func TestVetS8BiMatVatQuaChoCatKhongLot(t *testing.T) {
	const secret = "sk-abcdefghij0123456789zzzz"
	body := "Quy trình xử lý chênh lệch nhiên liệu: " + strings.Repeat("ab ", 9) + "mã kho " + secret + " dùng cho kho."
	if at := utf8.RuneCountInString(body[:strings.Index(body, "sk-")]); at < 70 || at > 76 {
		t.Fatalf("tiền đề: bí mật phải bắt đầu quanh ký tự 73 của bản nháp để chỗ cắt 80 rơi giữa nó, được %d", at)
	}
	a := askDocs(t, sopDoc("SOP-FUEL-07", "Quy trình xử lý chênh lệch nhiên liệu", body))
	turn := realTurn(t, a, person("director", ""), "Quy trình xử lý chênh lệch nhiên liệu là gì?")
	s8 := stepOf(t, traces.NewRecord(turn).Steps, "S8")
	if !strings.Contains(s8.Reason, "bản nháp do model extractive diễn đạt") {
		t.Fatalf("tiền đề: S8 phải ghi bản nháp: %q", s8.Reason)
	}
	for _, frag := range []string{"abcd", "sk-ab", secret} {
		if strings.Contains(s8.Reason, frag) {
			t.Errorf("mảnh bí mật %q lọt vào dòng S8: %s", frag, s8.Reason)
		}
	}
	// bí mật đã thành dấu che TRƯỚC khi cắt: đoạn đầu dừng giữa chữ "[REDACTED]" chứ không giữa bí mật
	if !strings.Contains(s8.Reason, "mã kho "+traces.Masked[:7]) {
		t.Errorf("chỗ bí mật phải thành %s (bị cắt ở ký tự 80): %s", traces.Masked, s8.Reason)
	}
}

// Câu xác nhận: S8 nói mẫu THẬT đã dùng (confirm_yes / confirm_no, hoặc mẫu của route cộng mẩu confirm_withheld khi cổng giữ phán quyết) — không phải
// RoutePlan.Template. Người đọc đối chiếu được với dòng cổng phán quyết ở S9 ("→ Đúng." / "→ Không."). Ô "quyết định" của S8 cũng nói mẫu thật (V8AS-05, vet_du_s8_o_quyet_dinh_test.go).
func TestVetS8MauThatCuaCauXacNhan(t *testing.T) {
	a := realAsk(t)
	for _, c := range []struct{ q, tmpl string }{
		{"EX-03 hôm nay được cấp 3850 L đúng không?", "confirm_yes"},
		{"EX-03 hôm nay được cấp 4000 L đúng không?", "confirm_no"},
		{"EX-03 được cấp 3850 L đúng không?", "WITHHELD"}, // thiếu mốc thời gian: giữ phán quyết
	} {
		turn := realTurn(t, a, person("director", ""), c.q)
		tmpl := c.tmpl
		if tmpl == "WITHHELD" {
			tmpl = turn.Plan.Template + " + confirm_withheld"
		}
		s8 := stepOf(t, traces.NewRecord(turn).Steps, "S8")
		if want := draftLine("", "dựng bằng mẫu câu "+tmpl+" (không gọi model)", turn.Response.Answer); !strings.Contains(s8.Reason, want) {
			t.Errorf("%q: S8 phải nói mẫu thật %q:\n  muốn: %s\n  được: %s", c.q, tmpl, want, s8.Reason)
		}
	}
}
