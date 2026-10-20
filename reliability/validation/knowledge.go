package validation

import (
	"fmt"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/luannt2002/enterprise-ai-systems/core/contracts"
	"github.com/luannt2002/enterprise-ai-systems/data/normalization"
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
// "4 m³", "3 khối") bị chặn. Chạy cùng ForbidDimensions: bộ chuẩn hoá đọc sai thì lớp này vẫn bắt.
func ForbidUnitWords(text string, words []string) []contracts.Violation {
	var out []contracts.Violation
	masked := dateRe.ReplaceAllStringFunc(text, func(s string) string { return strings.Repeat(" ", len(s)) })
	for _, m := range numRe.FindAllStringSubmatchIndex(masked, -1) {
		rest := strings.TrimLeft(masked[m[3]:], " ")
		for _, w := range words {
			after, ok := strings.CutPrefix(rest, w)
			if !ok || w == "" {
				continue
			}
			if r, _ := utf8.DecodeRuneInString(after); after != "" && (unicode.IsLetter(r) || unicode.IsDigit(r)) {
				continue // "L" của "Lập", "m" của "máy" không phải đơn vị
			}
			out = append(out, contracts.Violation{Validator: "knowledge_quantity_independent", Code: "QUANTITY_FROM_DOCUMENT",
				Severity: contracts.SeverityS1, Detail: fmt.Sprintf("lượng %q %s lấy từ tài liệu (bộ đếm độc lập)", text[m[2]:m[3]], w)})
			break
		}
	}
	return out
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
	in := norm.Normalize(draft.Text, normalization.Options{Now: now})
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
