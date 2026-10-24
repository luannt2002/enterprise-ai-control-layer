package reports

// §5.2: chuẩn hoá chữ để băm; §5.1 TR2: vân tay; §2.4: khoá đơn vị tất định.

import (
	"strings"
	"testing"
	"time"
)

// r là một ký tự theo mã điểm: tránh viết ký tự vô hình hay ký tự tổ hợp thẳng vào mã nguồn.
func r(cp rune) string { return string(cp) }

func TestChuanHoaChu_5_2(t *testing.T) {
	eq := map[string][2]string{
		"NFC: é tổ hợp bằng é dựng sẵn":      {"cafe" + r(0x301), "caf" + r(0xe9)},
		"bỏ khoảng trắng đầu cuối":           {"  máy 03 3k \n", "máy 03 3k"},
		"gộp khoảng trắng liên tiếp":         {"máy  03\t\t3k", "máy 03 3k"},
		"xuống dòng và NBSP là khoảng trắng": {"máy\n03" + r(0xa0) + "3k", "máy 03 3k"},
	}
	for name, p := range eq {
		if NormalizeText(p[0]) != NormalizeText(p[1]) {
			t.Errorf("%s: %q và %q phải cùng dạng chuẩn", name, p[0], p[1])
		}
	}
	// KHÔNG bỏ dấu, KHÔNG đổi hoa thường: để không gộp oan
	ne := map[string][2]string{
		"bỏ dấu":             {"máy 03", "may 03"},
		"đổi hoa thường":     {"Máy 03", "máy 03"},
		"số khác":            {"máy 03 3k", "máy 03 4k"},
		"ký tự vô hình khác": {"máy 03", "máy" + r(0x200b) + " 03"},
	}
	for name, p := range ne {
		if NormalizeText(p[0]) == NormalizeText(p[1]) {
			t.Errorf("%s: %q và %q không được cùng dạng chuẩn (gộp oan)", name, p[0], p[1])
		}
	}
	// UTF-8 hỏng không panic và tất định
	bad := "máy \xff\xfe 3k"
	if NormalizeText(bad) != NormalizeText(bad) || contentHashOf(bad) != contentHashOf(bad) {
		t.Error("chuẩn hoá UTF-8 hỏng phải tất định")
	}
}

func TestMakeUnitKey_2_4(t *testing.T) {
	for _, c := range []struct {
		origin, id string
		line       int
		want       string
	}{
		{"zalo/tin.jsonl", "M-17", 5, "zalo/tin.jsonl#M-17"},
		{"zalo/tin.jsonl", "", 5, "zalo/tin.jsonl#L5"},
		{"bao-cao/2026-10-19.pdf", "3", 3, "bao-cao/2026-10-19.pdf#3"},
	} {
		if got := MakeUnitKey(c.origin, c.id, c.line); got != c.want {
			t.Errorf("MakeUnitKey(%q, %q, %d) = %q, muốn %q", c.origin, c.id, c.line, got, c.want)
		}
	}
	if MakeUnitKey("f", "", 1) == MakeUnitKey("f", "", 2) {
		t.Error("hai dòng thiếu mã nguồn ở hai vị trí khác nhau không được chung khoá (HT-8)")
	}
}

func TestVanTay_TR2_TruongThamGia(t *testing.T) {
	base := mk("k1", 0).sanitized()
	fp := func(it Item) string { return fingerprintOf("ten-1", it.sanitized()) }
	same := func(name string, mod func(*Item)) {
		it := base
		mod(&it)
		if fp(it) != fp(base) {
			t.Errorf("vân tay (a) phải KHÔNG đổi khi %s", name)
		}
	}
	diff := func(name string, mod func(*Item)) {
		it := base
		mod(&it)
		if fp(it) == fp(base) {
			t.Errorf("vân tay (a) phải đổi khi %s", name)
		}
	}
	same("đổi unit_key (xuất chồng lấp: khác mã tin)", func(it *Item) { it.Key = "khac" })
	same("đổi nguyên văn chỉ ở khoảng trắng và dạng NFC", func(it *Item) { it.Raw = "  tin   k1 " })
	same("sent_at lệch dưới một giây", func(it *Item) { it.SentAt = it.SentAt.Add(300 * time.Millisecond) })
	diff("đổi kênh", func(it *Item) { it.Channel = "khac" })
	diff("đổi người gửi", func(it *Item) { it.SenderID = "khac" })
	diff("sent_at lệch một giây", func(it *Item) { it.SentAt = it.SentAt.Add(time.Second) })
	diff("đổi chữ", func(it *Item) { it.Raw = "tin khác" })
	if fingerprintOf("ten-1", base) == fingerprintOf("ten-2", base) {
		t.Error("hai tenant cùng đơn vị phải khác vân tay (không khử trùng xuyên tenant, BB8)")
	}

	// vế (b): đơn vị không có sent_at (dòng PDF, hàng bảng)
	nb := mk("f.pdf#3", 0, withNoSentAt("3")).sanitized()
	fb := func(it Item) string { return fingerprintOf("ten-1", it.sanitized()) }
	sameB := func(name string, mod func(*Item)) {
		it := nb
		mod(&it)
		if fb(it) != fb(nb) {
			t.Errorf("vân tay (b) phải KHÔNG đổi khi %s", name)
		}
	}
	diffB := func(name string, mod func(*Item)) {
		it := nb
		mod(&it)
		if fb(it) == fb(nb) {
			t.Errorf("vân tay (b) phải đổi khi %s", name)
		}
	}
	sameB("tệp chép sang đường dẫn khác (đổi unit_key, không gồm đường dẫn)", func(it *Item) { it.Key = "bản-chép/f.pdf#3" })
	diffB("đổi ngày xảy ra (cùng dòng chữ ở hai ngày: cả hai vào)", func(it *Item) { it.OccurredOn = "2026-10-20" })
	diffB("đổi mã nguồn trong tệp (hai dòng cùng chữ trong một tệp: cả hai vào)", func(it *Item) { it.SourceSeq = "4" })
	diffB("đổi loại nguồn", func(it *Item) { it.SourceType = "khac" })
	diffB("đổi chữ", func(it *Item) { it.Raw = "dòng khác" })
	if fb(nb) == fingerprintOf("ten-1", base) {
		t.Error("vế (a) và vế (b) không được cùng miền băm")
	}
	if len(fb(nb)) != 64 || strings.Trim(fb(nb), "0123456789abcdef") != "" {
		t.Errorf("vân tay phải là sha256 chữ thường 64 ký tự, được %q", fb(nb))
	}
}
