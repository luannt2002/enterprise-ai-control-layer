package contracts

import (
	"strings"
	"testing"
)

func TestQuoteKhongThoatNhay(t *testing.T) {
	// nháy kép và gạch chéo giữ NGUYÊN: thoát kiểu %q làm hỏng mẫu che bí mật {"api_key": "…"}, password="…" của traces.Sanitize
	for _, in := range []string{`{"api_key": "abc123xyz"}`, `password="hunter2"`, `đường dẫn C:\tmp\x`} {
		if got, want := Quote(in), QuoteOpen+in+QuoteClose; got != want {
			t.Errorf("Quote(%q) = %q, muốn %q", in, got, want)
		}
	}
}

func TestQuoteBaoVaGiuCapBaoTimLaiDuoc(t *testing.T) {
	// “ ” có sẵn trong chữ bị đổi thành nháy thường: cặp bao chỉ xuất hiện đúng một lần, nên đoạn được bao tìm lại chính xác
	got := Quote("a “b” c")
	if got != "“a \"b\" c”" || strings.Count(got, QuoteOpen) != 1 || strings.Count(got, QuoteClose) != 1 {
		t.Errorf("Quote chữ có “ ”: %q", got)
	}
	// khoảng trắng và xuống dòng gộp một khoảng trắng (bảng chữ không vỡ dòng)
	if got := Quote("  a \n\t b  "); got != "“a b”" {
		t.Errorf("gộp khoảng trắng: %q", got)
	}
	if got := Quote(""); got != "“”" {
		t.Errorf("chữ rỗng: %q", got)
	}
}

func TestQuoteCatOMaxQuoteRunes(t *testing.T) {
	long := strings.Repeat("ê", maxQuoteRunes+30)
	if want := QuoteOpen + strings.Repeat("ê", maxQuoteRunes) + "…" + QuoteClose; Quote(long) != want {
		t.Errorf("chữ dài phải cắt ở %d ký tự (không cắt giữa ký tự nhiều byte): %q", maxQuoteRunes, Quote(long))
	}
	if exact := strings.Repeat("ê", maxQuoteRunes); Quote(exact) != QuoteOpen+exact+QuoteClose {
		t.Errorf("đúng trần thì không cắt")
	}
}

func TestBrief(t *testing.T) {
	keys := []string{"a", "b", "c", "d"}
	for _, c := range []struct {
		max  int
		want string
	}{{4, "a, b, c, d"}, {9, "a, b, c, d"}, {3, "a, b, c và 1 ứng viên khác"}, {1, "a và 3 ứng viên khác"}} {
		if got := Brief(keys, c.max, "ứng viên"); got != c.want {
			t.Errorf("Brief(max=%d) = %q, muốn %q", c.max, got, c.want)
		}
	}
	if got := Brief(nil, 3, "ứng viên"); got != "" {
		t.Errorf("danh sách rỗng: %q", got)
	}
}
