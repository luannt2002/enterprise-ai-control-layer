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
// "4 m³", "3 khối", "5 mét khối") bị chặn — kể cả đơn vị trong ngoặc ("5.000 (lít/ngày)", "5.000 (lít dầu)"), số trong ngoặc ("(4.000)
// lít") hay đơn vị đứng trước số ("L 5.000", "lít: 5.000", "lít/ca: 4.000"). Chạy cùng ForbidDimensions: bộ chuẩn hoá đọc sai thì
// lớp này vẫn bắt.
//
// Cố ý KHÔNG dùng lại hàm nào của bộ chuẩn hoá (kể cả gấp ký tự lạ): lớp này tồn tại để bắt khi bộ chuẩn
// hoá sai, nên nó tự gấp ký tự lạ ở đây — và tự có các luật "đơn vị đứng trước số" (unitBefore) và "ngoặc" (bracketTail) của riêng nó.
func ForbidUnitWords(text string, words []string) []contracts.Violation {
	text = foldLookalikes(text)
	var out []contracts.Violation
	masked := dateRe.ReplaceAllStringFunc(text, func(s string) string { return strings.Repeat(" ", len(s)) })
	for _, m := range numRe.FindAllStringSubmatchIndex(masked, -1) {
		num := masked[m[2]:m[3]]
		w, ok := unitAfter(masked[m[3]:], words, quantityLike(num) && opensBefore(masked[:m[2]]), quantityLike(num))
		if !ok {
			w, ok = unitBefore(masked[:m[2]], num, words)
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
// ngoặc cũng tính khi phần sau nó là phần cuối của chú thích đơn vị ("5.000 (L)", "5.000 (lít/ngày)", "5.000 (lít dầu)", xem
// bracketTail); "3 (khối lượng công việc)" thì không. enclosed: số đứng trong ngoặc, đơn vị đứng sau ngoặc đóng ("(4.000) lít").
func unitAfter(rest string, words []string, enclosed, quantityShaped bool) (string, bool) {
	rest, cell := skipDecoration(rest)
	if r, w := utf8.DecodeRuneInString(rest); enclosed && (r == ')' || r == ']' || r == '}') {
		rest, cell = skipDecoration(rest[w:])
	}
	bracketed := false
	if r, w := utf8.DecodeRuneInString(rest); r == '(' || r == '[' || r == '{' {
		rest, bracketed = skipUnitLabel(strings.TrimLeftFunc(rest[w:], isGap)), true
	}
	for _, w := range words {
		after, ok := cutPrefixFold(rest, w)
		if !ok || w == "" {
			continue
		}
		if bracketed {
			if !bracketTail(after) {
				continue
			}
		} else if r, _ := utf8.DecodeRuneInString(after); after != "" && isWordRune(r) {
			continue
		}
		if cell && !cellEndsAfter(after) {
			continue
		}
		// "Khối" mở đầu một TÊN RIÊNG ("Bước 2, Khối Kế hoạch") là khối phòng ban, không phải đơn vị của số thứ tự đứng trước (soát R, R-A15).
		// Chỉ miễn khi số không có dáng lượng, chữ đơn vị VIẾT HOA và chữ kế tiếp cũng viết hoa: "**5** khối", "**5** Khối mỗi ca", "| 5 | khối |"
		// vẫn bị chặn (soát 37e8a08, RM-1 / F7 — bản miễn trước đó theo dấu trang trí làm lọt chúng).
		if !quantityShaped && ambiguousUnitWord(w) && properNameStart(rest, after) {
			continue
		}
		return w, true
	}
	return "", false
}

// ambiguousUnitWord báo chữ đơn vị w còn là một danh từ thường gặp ngoài nghĩa đơn vị: "khối" (khối phòng ban). "L" / "l" là ký hiệu đơn vị nên
// không mơ hồ.
func ambiguousUnitWord(w string) bool {
	return strings.EqualFold(w, "khối") || strings.EqualFold(w, "khoi")
}

// properNameStart báo chữ ở đầu text viết hoa và chữ đầu tiên của after (phần sau chữ đó) cũng viết hoa: "Khối Kế hoạch", "Khối Vận hành".
func properNameStart(text, after string) bool {
	r, _ := utf8.DecodeRuneInString(text)
	n, _ := utf8.DecodeRuneInString(strings.TrimLeft(after, " \t"))
	return unicode.IsUpper(r) && unicode.IsUpper(n)
}

// skipDecoration bỏ phần TRANG TRÍ giữa một số và đơn vị của nó: khoảng đệm, dấu nhấn Markdown (`**4.000** lít`, `4.000` lít, `*4.000* *lít*`),
// thẻ HTML (`<b>4.000</b> lít`) và dấu phẩy ngăn (`4.000, lít`) — người đọc thấy "4.000 lít", bộ đếm chỉ thấy một dấu lạ chen giữa. Dấu "|" (ranh giới ô
// của bảng Markdown: `| 4.000 | lít/ca |`) cũng bỏ, nhưng cell báo true: lúc đó đơn vị phải là TOÀN BỘ ô tiếp theo (cellEndsAfter), vì ô sau chứa chữ khác
// ("| 2 | Khối Kế hoạch |") thì không phải đơn vị.
func skipDecoration(rest string) (string, bool) {
	cell := false
	for {
		rest = strings.TrimLeftFunc(rest, isGap)
		switch {
		case rest == "":
			return rest, cell
		case strings.IndexByte("*_`~,", rest[0]) >= 0:
			rest = rest[1:]
		case rest[0] == '|':
			rest, cell = rest[1:], true
		case rest[0] == '<':
			end := strings.IndexByte(rest, '>') // thẻ ngắn: <b>, </b>, <span class="x">
			if end < 2 || end > 24 || strings.ContainsAny(rest[1:end], "<\n") {
				return rest, cell
			}
			if r, _ := utf8.DecodeRuneInString(rest[1:]); !unicode.IsLetter(r) && r != '/' {
				return rest, cell
			}
			rest = rest[end+1:]
		default:
			return rest, cell
		}
	}
}

// cellEndsAfter báo ô của bảng Markdown kết thúc ngay sau đơn vị (phần after là hết ô hoặc chỉ còn mẫu số "/ca"): "| lít/ca |", "| L |".
func cellEndsAfter(after string) bool {
	after = trimDenominatorHead(strings.TrimLeftFunc(after, isGap))
	after = strings.TrimLeftFunc(after, isGap)
	return after == "" || after[0] == '|' || after[0] == '\n'
}

// trimDenominatorHead bỏ mẫu số "/ca" ở ĐẦU s (một lần).
func trimDenominatorHead(s string) string {
	if !strings.HasPrefix(s, "/") {
		return s
	}
	i := 1
	for i < len(s) {
		r, w := utf8.DecodeRuneInString(s[i:])
		if !isWordRune(r) {
			break
		}
		i += w
	}
	return s[i:]
}

// unitLabels là nhãn đứng trước đơn vị trong ngoặc ("(đơn vị: lít)", "(ĐVT: lít)", "(tính bằng lít)"): bỏ nhãn rồi đọc đơn vị như "(lít)".
var unitLabels = []string{"đơn vị tính", "đơn vị", "đvt", "đv", "tính bằng", "tính theo", "theo đơn vị"}

// skipUnitLabel bỏ nhãn đơn vị (unitLabels) và dấu hai chấm / khoảng đệm theo sau ở đầu rest; không có nhãn thì trả rest nguyên.
func skipUnitLabel(rest string) string {
	for _, l := range unitLabels {
		if after, ok := cutPrefixFold(rest, l); ok {
			if r, _ := utf8.DecodeRuneInString(after); after != "" && isWordRune(r) {
				continue
			}
			return strings.TrimLeftFunc(after, func(r rune) bool { return isGap(r) || r == ':' || r == '=' })
		}
	}
	return rest
}

// bracketTail báo phần đứng sau một từ đơn vị MỞ ĐẦU một cặp ngoặc có đúng là phần cuối của chú thích đơn vị không: ngoặc đóng, "/" (mẫu
// số: "(lít/ngày)") hoặc — sau khoảng đệm — một từ chữ phụ ("(lít dầu)"). "(khối lượng công việc)" thì không: ở đó "khối" là đầu một từ
// khác, không phải đơn vị.
func bracketTail(after string) bool {
	trimmed := strings.TrimLeftFunc(after, isGap)
	if trimmed == "" {
		return false
	}
	if strings.ContainsRune(")]}/", rune(trimmed[0])) {
		return true
	}
	if r, _ := utf8.DecodeRuneInString(trimmed); len(trimmed) == len(after) || !unicode.IsLetter(r) {
		return false
	}
	rest, isLuong := cutPrefixFold(trimmed, "lượng")
	if r, _ := utf8.DecodeRuneInString(rest); isLuong && (rest == "" || !isWordRune(r)) {
		return false
	}
	return true
}

// quantityLike báo số viết kiểu một LƯỢNG: có dấu nghìn / thập phân, hoặc từ hai chữ số. Số MỘT chữ số đứng cạnh một từ trùng tên đơn vị
// thường là số thứ tự / nhãn ("khối 2", "L 3", "(1) Khối tài chính"), không phải lượng.
func quantityLike(num string) bool {
	return strings.ContainsAny(num, ".,") || len(strings.TrimLeft(num, "-")) >= 2
}

// opensBefore báo phần văn bản before kết thúc bằng một dấu mở ngoặc (chỉ cách khoảng đệm): số ngay sau nó đứng trong ngoặc.
func opensBefore(before string) bool {
	before = strings.TrimRightFunc(before, isGap)
	return before != "" && strings.ContainsRune("([{", rune(before[len(before)-1]))
}

// unitBefore báo before (phần ngay trước số num) kết thúc bằng một từ đơn vị: "L 5.000", "lít: 5.000",
// "(lít): 5.000", "lít/ca: 4.000" — bỏ qua dấu cách, dấu hai chấm, dấu bằng và dấu đóng ngoặc nối đơn vị với số, và mẫu số của đơn vị
// ghép ("/ca"). Từ đơn vị phải TRỌN từ ("l" của "email" không phải đơn vị). Không phải đơn vị của số khi:
//   - nó là MẪU SỐ của một giá ("đồng/lít: 25.000", "(đồng/m3) 25.000"): đứng sau "/";
//   - nó chỉ cách số bằng khoảng trắng mà số không có dạng lượng — một chữ số ("khối 2", "L 3", "lít 2 liên"): nhãn bộ phận / mục / số đếm.
func unitBefore(before, num string, words []string) (string, bool) {
	trimmed := strings.TrimRightFunc(before, func(r rune) bool { return isGap(r) || strings.ContainsRune(":=)]}", r) })
	plainGap := strings.TrimFunc(before[len(trimmed):], isGap) == ""
	trimmed = trimDenominator(trimmed)
	for _, w := range words {
		head, ok := cutSuffixFold(trimmed, w)
		if !ok || w == "" {
			continue
		}
		if r, _ := utf8.DecodeLastRuneInString(head); head != "" && (isWordRune(r) || r == '/') {
			continue
		}
		if plainGap && !quantityLike(num) {
			continue
		}
		return w, true
	}
	return "", false
}

// trimDenominator bỏ mẫu số của một đơn vị ghép ở CUỐI s: "/ca" của "lít/ca", "/lít" của "đồng/lít" (một lần).
func trimDenominator(s string) string {
	i := len(s)
	for i > 0 {
		r, w := utf8.DecodeLastRuneInString(s[:i])
		if !isWordRune(r) {
			break
		}
		i -= w
	}
	if i < len(s) && i > 0 && s[i-1] == '/' {
		return s[:i-1]
	}
	return s
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
