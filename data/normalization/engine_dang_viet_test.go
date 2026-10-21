package normalization

import (
	"testing"

	"github.com/luannt2002/enterprise-ai-control-layer/core/contracts"
)

// quantities trả các span lượng (có đơn vị) của một lần chuẩn hoá.
func quantities(in contracts.NormalizedInput) []contracts.Span {
	return in.SpansOf(contracts.ValueQuantity)
}

// Cùng một lượng "5.000 L" viết bằng khoảng trắng lạ, chữ số toàn chiều rộng, ký tự vô hình hay ký hiệu ℓ
// phải ra đúng MỘT span lượng 5000 L, và vị trí byte của span trỏ đúng vào Raw (Raw là văn bản đã gấp).
func TestGapKyTuNhinGiongVeDangChuan(t *testing.T) {
	e := mustEngine(t)
	for _, in := range []string{
		"cấp 5.000 L", "cấp 5.000\u00a0L", "cấp 5.000\u200bL", "cấp 5.000\u202fL", "cấp 5.000\tL", "cấp 5.000  L",
		"cấp 5.000\u00a0\u00a0L", "cấp 5.000\u3000L", "cấp ５.０００ L", "cấp ５.０００ Ｌ", "cấp ５０００ lít", "cấp 5.0\u200b00 L",
		"cấp 5.000 ℓ", "cấp 5.000ℓ", "cấp ５.０００ ℓ", "cấp 5\u00ad.000 L", "cấp\u200b5.000 L",
	} {
		got := e.Normalize(in, Options{Now: now})
		q := quantities(got)
		if len(q) != 1 || q[0].Value.Number != "5000" || q[0].Value.Unit != "L" {
			t.Errorf("%q: muốn đúng một lượng 5000 L, được:\n%s", in, dump(got.Spans))
			continue
		}
		for _, sp := range got.Spans {
			if got.Raw[sp.Start:sp.End] != sp.Raw {
				t.Errorf("%q: span %q lệch vị trí trong Raw %q: [%d,%d)", in, sp.Raw, got.Raw, sp.Start, sp.End)
			}
		}
	}
}

// Ký tự vô hình chen giữa hai chữ số không được tách một số thành hai: người đọc thấy 3850, nên phải là MỘT
// số 3850. Khoảng trắng thường vẫn tách (3 và 850).
func TestKyTuVoHinhKhongTachSo(t *testing.T) {
	e := mustEngine(t)
	for _, c := range []struct {
		in   string
		want []string
	}{
		{"bước 3\u200b850", []string{"3850"}},
		{"bước 3\u2060850", []string{"3850"}},
		{"bước 3 850", []string{"3", "850"}},
	} {
		var got []string
		for _, sp := range e.Normalize(c.in, Options{Now: now}).SpansOf(contracts.ValueNumber) {
			got = append(got, sp.Value.Number)
		}
		if len(got) != len(c.want) || (len(got) > 0 && got[0] != c.want[0]) || (len(got) > 1 && got[1] != c.want[1]) {
			t.Errorf("%q: số = %v, muốn %v", c.in, got, c.want)
		}
	}
}

// Options.Layout: đọc cách TRÌNH BÀY của tài liệu — đơn vị đứng trước số, đơn vị trong ngoặc sau số. Tắt (câu hỏi
// của người dùng) thì các dạng đó KHÔNG thành lượng: "bao nhiêu lít 7 ngày qua" không có lượng "lít 7".
func TestLayoutDocDonViTruocVaTrongNgoac(t *testing.T) {
	e := mustEngine(t)
	for _, in := range []string{
		"L 5.000", "lít: 5.000", "Lít: 5.000", "(lít): 5.000", "định mức (lít) = 5.000", "5.000 (L)", "5.000 (lít/ngày)",
		"5.000 [L]", "5.000 ( L )", "5.000\u00a0(lít/ngày)", "L ５.０００", "L\u200b5.000",
	} {
		on := quantities(e.Normalize(in, Options{Now: now, Layout: true}))
		if len(on) != 1 || on[0].Value.Number != "5000" || on[0].Value.Unit != "L" {
			t.Errorf("Layout bật, %q: muốn đúng một lượng 5000 L, được %+v", in, on)
		}
		if off := quantities(e.Normalize(in, Options{Now: now})); len(off) != 0 {
			t.Errorf("Layout tắt, %q: không được có lượng, được %+v", in, off)
		}
	}
	// số đã mang đơn vị riêng thì thuộc đơn vị đó
	q := quantities(e.Normalize("lít 7 giờ", Options{Now: now, Layout: true}))
	if len(q) != 1 || q[0].Value.Unit != "h" || q[0].Value.Number != "7" {
		t.Errorf("\"lít 7 giờ\": muốn đúng một lượng 7 h, được %+v", q)
	}
	if q := quantities(e.Normalize("EX-03 bao nhiêu lít 7 ngày qua", Options{Now: now})); len(q) != 0 {
		t.Errorf("câu hỏi (Layout tắt) bị đọc nhầm \"lít 7\" thành lượng: %+v", q)
	}
}

// Layout không được đọc nhầm thứ không phải lượng: số bước, ngoặc chú thích bắt đầu bằng chữ "khối" của "khối
// lượng", chữ "l" cuối từ khác.
func TestLayoutKhongDocNhamKhongPhaiLuong(t *testing.T) {
	e := mustEngine(t)
	for _, in := range []string{
		"Bước 3 (xem phụ lục)", "Bước 5 (khối lượng công việc)", "Bước 5 (khối lượng) được xác nhận", "gửi email 3 lần",
		"Mục l. Lập biên bản trong 24 giờ", "khối lượng 3 hạng mục", "Tầng 3 (lập biên bản)",
	} {
		for _, sp := range quantities(e.Normalize(in, Options{Now: now, Layout: true})) {
			if sp.Value.Unit == "L" || sp.Value.Unit == "m3" {
				t.Errorf("%q: đọc nhầm lượng thể tích %s %s (rule %s)", in, sp.Value.Number, sp.Value.Unit, sp.Rule)
			}
		}
	}
}
