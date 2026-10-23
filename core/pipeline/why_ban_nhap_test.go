package pipeline

import (
	"crypto/sha256"
	"encoding/hex"
	"slices"
	"strings"
	"testing"

	"github.com/luannt2002/enterprise-ai-control-layer/observability/traces"
)

// V8a. Các hàm ghi S8 (bản nháp như được sinh): đếm theo KÝ TỰ, băm chữ đã làm sạch, đoạn đầu cắt đúng 80 ký tự — khoá từng chữ của dòng.

func sha12(s string) string {
	sum := sha256.Sum256([]byte(s))
	return hex.EncodeToString(sum[:])[:12]
}

func TestTextDigestDemKyTuVaLamSachTruoc(t *testing.T) {
	if n, h := textDigest("  ệệệ \n"); n != 3 || h != sha12("ệệệ") {
		t.Errorf("khoảng trắng đầu / cuối không tính, đếm ký tự (không phải byte): %d %s", n, h)
	}
	if n, h := textDigest(""); n != 0 || h != sha12("") {
		t.Errorf("chữ rỗng: %d %s", n, h)
	}
	// bí mật được che TRƯỚC khi đếm và băm: số ký tự và băm khớp chữ nằm trong vết bền
	masked := "password=" + traces.Masked
	if n, h := textDigest("password=hunter2pw"); n != len([]rune(masked)) || h != sha12(masked) {
		t.Errorf("chữ có bí mật: %d %s, muốn %d %s", n, h, len([]rune(masked)), sha12(masked))
	}
}

func TestDraftHeadCat80KyTu(t *testing.T) {
	eighty := strings.Repeat("ệ", 80)
	for _, c := range []struct{ in, want string }{
		{eighty, eighty},
		{eighty + "x", eighty + "…"},
		{"  ngắn  ", "ngắn"},
		{"", ""},
	} {
		if got := draftHead(c.in); got != c.want {
			t.Errorf("draftHead(%.12q…) = %.20q…, muốn %.20q…", c.in, got, c.want)
		}
	}
	if got := draftHead(strings.Repeat("a", 60) + " mật khẩu: hunter2pw " + strings.Repeat("b", 30)); strings.Contains(got, "hunter2pw") {
		t.Errorf("đoạn đầu phải qua bộ làm sạch: %q", got)
	}
}

func TestNoteDraftTungChu(t *testing.T) {
	a := &Ask{}
	var why traces.Why
	text := "Dầu kho xuất của Máy xúc EX-03 ngày 19/10/2026: 3.850 L."
	a.noteTemplateDraft(&why, "", "metric_total", text)
	a.noteModelDraft(&why, "tri thức: ", "extractive", text)
	a.noteModelDraft(&why, "", "", text)
	a.noteTemplateDraft(&why, "số liệu: ", "metric_total + confirm_withheld", text)
	h := sha12(text)
	want := []string{
		"bản nháp dựng bằng mẫu câu metric_total (không gọi model): 56 ký tự · băm " + h + " · đầu “" + text + "”",
		"tri thức: bản nháp do model extractive diễn đạt: 56 ký tự · băm " + h + " · đầu “" + text + "”",
		"bản nháp do model (không khai tên) diễn đạt: 56 ký tự · băm " + h + " · đầu “" + text + "”",
		"số liệu: bản nháp dựng bằng mẫu câu metric_total + confirm_withheld (không gọi model): 56 ký tự · băm " + h + " · đầu “" + text + "”",
	}
	if got := len([]rune(text)); got != 56 {
		t.Fatalf("tiền đề: chữ mẫu dài %d ký tự, muốn 56", got)
	}
	if !slices.Equal(why.Generate.Reasons, want) {
		t.Errorf("S8:\n  được:\n    %s\n  muốn:\n    %s", strings.Join(why.Generate.Reasons, "\n    "), strings.Join(want, "\n    "))
	}
	if len(why.Generate.Rejected) != 0 {
		t.Errorf("S8 không có phương án bị loại: %q", why.Generate.Rejected)
	}
	// công tắc noWhy: không ghi gì
	var off traces.Why
	(&Ask{noWhy: true}).noteTemplateDraft(&off, "", "metric_total", text)
	if off.Generate.Recorded() {
		t.Errorf("noWhy phải tắt ghi S8: %+v", off.Generate)
	}
}

// Dòng S8 đi qua MỘT chỗ che phạm vi (finalWhy): chữ bản nháp nhắc khoá thực thể ngoài phạm vi thì khoá bị che; câu hỏi mang bí mật thì đoạn đầu (trong
// cặp “…”) bị che nguyên, còn số ký tự và băm giữ nguyên.
func TestDongS8QuaChoChePhamViVaWithholdQuoted(t *testing.T) {
	a := &Ask{}
	var why traces.Why
	a.noteModelDraft(&why, "", "extractive", "Máy ủi DZ-03 cần đối soát theo quy trình")
	got := finalWhy(why, scopeRedactor{tokens: []string{"DZ-03"}}).Generate.Reasons
	if len(got) != 1 || strings.Contains(got[0], "DZ-03") || !strings.Contains(got[0], "Máy ủi "+traces.OutOfScope+" cần đối soát") {
		t.Errorf("khoá ngoài phạm vi phải bị che trong dòng S8: %q", got)
	}
	if !strings.Contains(got[0], "40 ký tự · băm "+sha12("Máy ủi DZ-03 cần đối soát theo quy trình")) {
		t.Errorf("số ký tự và băm tính trên chữ gốc, không đổi khi che: %q", got)
	}
	w := why.WithholdQuoted().Generate.Reasons[0]
	if strings.Contains(w, "Máy ủi") || !strings.Contains(w, "40 ký tự · băm "+sha12("Máy ủi DZ-03 cần đối soát theo quy trình")+" · đầu “đã che”") {
		t.Errorf("WithholdQuoted phải che đoạn đầu và giữ số ký tự, băm: %q", w)
	}
}
