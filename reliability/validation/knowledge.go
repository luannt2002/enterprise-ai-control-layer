package validation

import (
	"fmt"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/luannt2002/enterprise-ai-systems/core/contracts"
	"github.com/luannt2002/enterprise-ai-systems/data/normalization"
	unicodenorm "golang.org/x/text/unicode/norm"
)

// Dimensions trả các chiều đo (volume, duration…) có mặt trong văn bản: từ lượng có đơn vị và từ
// thuật ngữ có đơn vị mặc định ("dầu" → lít → volume).
func Dimensions(norm *normalization.Engine, text string, now time.Time) map[string]bool {
	out := map[string]bool{}
	in := norm.Normalize(text, normalization.Options{Now: now})
	for _, sp := range in.Spans {
		switch sp.Kind {
		case contracts.ValueQuantity:
			if u, ok := norm.Unit(sp.Value.Unit); ok {
				out[u.Dimension] = true
			}
		case contracts.ValueTerm:
			if t, ok := norm.Term(sp.Value.Text); ok && t.DefaultUnit != "" {
				if u, ok := norm.Unit(t.DefaultUnit); ok {
					out[u.Dimension] = true
				}
			}
		}
	}
	return out
}

// ForbidUnitWords là lớp kiểm ĐỘC LẬP với bộ chuẩn hoá cho phần trích tài liệu: số (đếm bằng biểu
// thức chính quy của CountNumbers) đứng ngay trước một từ đơn vị của chiều đo nghiệp vụ ("5.000 L",
// "4 m³", "3 khối") bị chặn — kể cả đơn vị trong ngoặc ("5.000 (lít/ngày)") hay đứng trước số ("L 5.000",
// "lít: 5.000"). Chạy cùng ForbidDimensions: bộ chuẩn hoá đọc sai thì lớp này vẫn bắt.
//
// Cố ý KHÔNG dùng lại hàm nào của bộ chuẩn hoá (kể cả gấp ký tự lạ): lớp này tồn tại để bắt khi bộ chuẩn
// hoá sai, nên nó tự gấp ký tự lạ ở đây.
func ForbidUnitWords(text string, words []string) []contracts.Violation {
	text = foldLookalikes(text)
	var out []contracts.Violation
	masked := dateRe.ReplaceAllStringFunc(text, func(s string) string { return strings.Repeat(" ", len(s)) })
	for _, m := range numRe.FindAllStringSubmatchIndex(masked, -1) {
		w, ok := unitAfter(masked[m[3]:], words)
		if !ok {
			w, ok = unitBefore(masked[:m[2]], words)
		}
		if ok {
			out = append(out, contracts.Violation{Validator: "knowledge_quantity_independent", Code: "QUANTITY_FROM_DOCUMENT",
				Severity: contracts.SeverityS1, Detail: fmt.Sprintf("lượng %q %s lấy từ tài liệu (bộ đếm độc lập)", masked[m[2]:m[3]], w)})
		}
	}
	return out
}

// foldLookalikes gấp ký tự NHÌN GIỐNG về dạng ASCII để số và đơn vị viết lạ không thoát bộ đếm: chữ số / chữ cái /
// dấu câu toàn chiều rộng → ASCII; tab, NBSP, U+202F, U+3000… → dấu cách; "ℓ" → "l"; rồi NFC để "lít" gõ dấu rời
// (máy Apple) vẫn là "lít". Ký tự định dạng vô hình (ZWSP, ZWJ…) giữ nguyên và được coi là khoảng đệm (isGap).
func foldLookalikes(s string) string {
	return unicodenorm.NFC.String(strings.Map(func(r rune) rune {
		switch {
		case r >= 0xFF01 && r <= 0xFF5E: // dạng toàn chiều rộng của ASCII
			return r - 0xFEE0
		case r != '\n' && r != '\r' && unicode.IsSpace(r):
			return ' '
		case r == 0x2113: // "ℓ", ký hiệu lít
			return 'l'
		}
		return r
	}, s))
}

// isGap: khoảng đệm giữa số và đơn vị — dấu cách hoặc ký tự định dạng vô hình (ZWSP, ZWJ…).
func isGap(r rune) bool { return r == ' ' || unicode.Is(unicode.Cf, r) }

// unitAfter báo rest (phần ngay sau một số) mở đầu bằng một từ đơn vị: bỏ qua dấu cách; không phân biệt hoa
// thường ("LÍT"); từ đơn vị phải TRỌN từ ("L" của "Lập", "m" của "máy" không phải đơn vị). Đơn vị MỞ ĐẦU một cặp
// ngoặc cũng tính nếu ngoặc đóng ngay hoặc đơn vị theo sau là "/…" ("5.000 (L)", "5.000 (lít/ngày)");
// "3 (khối lượng công việc)" thì không.
func unitAfter(rest string, words []string) (string, bool) {
	rest = strings.TrimLeftFunc(rest, isGap)
	bracketed := false
	if r, w := utf8.DecodeRuneInString(rest); r == '(' || r == '[' || r == '{' {
		rest, bracketed = strings.TrimLeftFunc(rest[w:], isGap), true
	}
	for _, w := range words {
		after, ok := cutPrefixFold(rest, w)
		if !ok || w == "" {
			continue
		}
		if bracketed {
			if after = strings.TrimLeftFunc(after, isGap); after == "" || !strings.ContainsRune(")]}/", rune(after[0])) {
				continue
			}
		} else if r, _ := utf8.DecodeRuneInString(after); after != "" && isWordRune(r) {
			continue
		}
		return w, true
	}
	return "", false
}

// unitBefore báo before (phần ngay trước một số) kết thúc bằng một từ đơn vị: "L 5.000", "lít: 5.000",
// "(lít): 5.000" — bỏ qua dấu cách, dấu hai chấm, dấu bằng và dấu đóng ngoặc nối đơn vị với số. Từ đơn vị
// phải TRỌN từ ("l" của "email" không phải đơn vị).
func unitBefore(before string, words []string) (string, bool) {
	before = strings.TrimRightFunc(before, func(r rune) bool { return isGap(r) || strings.ContainsRune(":=)]}", r) })
	for _, w := range words {
		head, ok := cutSuffixFold(before, w)
		if !ok || w == "" {
			continue
		}
		if r, _ := utf8.DecodeLastRuneInString(head); head != "" && isWordRune(r) {
			continue
		}
		return w, true
	}
	return "", false
}

// cutPrefixFold như strings.CutPrefix nhưng không phân biệt hoa thường.
func cutPrefixFold(s, prefix string) (string, bool) {
	for _, pr := range prefix {
		r, w := utf8.DecodeRuneInString(s)
		if w == 0 || unicode.ToLower(r) != unicode.ToLower(pr) {
			return s, false
		}
		s = s[w:]
	}
	return s, true
}

// cutSuffixFold như strings.CutSuffix nhưng không phân biệt hoa thường.
func cutSuffixFold(s, suffix string) (string, bool) {
	for i := len(suffix); i > 0; {
		sr, sw := utf8.DecodeLastRuneInString(suffix[:i])
		r, w := utf8.DecodeLastRuneInString(s)
		if w == 0 || unicode.ToLower(r) != unicode.ToLower(sr) {
			return s, false
		}
		s, i = s[:len(s)-w], i-sw
	}
	return s, true
}

// Bound kiểm ràng buộc claim của bản nháp do model viết: phải có ít nhất một claim, và mỗi claim
// phải trỏ tới bằng chứng CÓ trong ngữ cảnh đã đưa cho model. Câu không gắn được vào bằng chứng
// thì không giao — model không được tự thêm nguồn.
func Bound(draft contracts.Draft, ev []contracts.Evidence) []contracts.Violation {
	if len(draft.Claims) == 0 {
		return []contracts.Violation{{Validator: "claim_binding", Code: "CLAIM_UNBOUND", Severity: contracts.SeverityS1,
			Detail: "bản nháp không có claim nào gắn với bằng chứng"}}
	}
	known := map[string]bool{}
	for _, e := range ev {
		known[e.ID] = true
	}
	var out []contracts.Violation
	for _, c := range draft.Claims {
		ok := len(c.EvidenceIDs) > 0
		for _, id := range c.EvidenceIDs {
			ok = ok && known[id]
		}
		if !ok {
			out = append(out, contracts.Violation{Validator: "claim_binding", Code: "CLAIM_UNBOUND", Severity: contracts.SeverityS1,
				Detail: fmt.Sprintf("claim %q trỏ tới bằng chứng ngoài ngữ cảnh %v", c.Text, c.EvidenceIDs)})
		}
	}
	return out
}

// ForbidDimensions chặn mọi LƯỢNG thuộc các chiều đo cho trước trong một bản nháp. Dùng cho phần
// trả lời lấy từ TÀI LIỆU: tài liệu không phải nguồn có thẩm quyền của số liệu nghiệp vụ (lượng
// dầu chỉ đến từ fact), nên lượng dầu nằm trong đoạn SOP — kể cả đoạn bị chèn lệnh — không được
// giao. Thời hạn ("24 giờ") hay số bước không thuộc chiều bị cấm thì vẫn qua.
func ForbidDimensions(norm *normalization.Engine, draft contracts.Draft, dims map[string]bool, now time.Time) []contracts.Violation {
	var out []contracts.Violation
	in := norm.Normalize(draft.Text, normalization.Options{Now: now, Layout: true})
	for _, sp := range in.Spans {
		if sp.Kind != contracts.ValueQuantity {
			continue
		}
		if u, ok := norm.Unit(sp.Value.Unit); ok && dims[u.Dimension] {
			out = append(out, contracts.Violation{
				Validator: "knowledge_quantity", Code: "QUANTITY_FROM_DOCUMENT", Severity: contracts.SeverityS1,
				Detail: fmt.Sprintf("lượng %q lấy từ tài liệu — số liệu nghiệp vụ chỉ được đến từ fact", sp.Raw),
			})
		}
	}
	return out
}
