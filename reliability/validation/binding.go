package validation

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/luannt2002/enterprise-ai-systems/core/contracts"
	"github.com/luannt2002/enterprise-ai-systems/core/decimal"
)

// Ràng buộc claim ĐỦ BỘ (docs/04 mục 6, G5). Mỗi con số trong câu trả lời số liệu phải là giá
// trị của một claim có chủ thể / chỉ số / kỳ / giá trị / đơn vị / nguồn, và claim đó phải khớp ĐỦ
// SÁU chiều với các bằng chứng nó trỏ tới. Số và ngày trong câu được đếm bằng bộ đếm RIÊNG (biểu
// thức chính quy ở đây), không qua bộ chuẩn hoá — lỗi của bộ chuẩn hoá không tự che được chính nó.

var (
	dateRe = regexp.MustCompile(`(\d{1,2})/(\d{1,2})/(\d{4})`)
	// Số kiểu Việt: "3.850", "1.602.000", "0,35", "-68". Số dính sau chữ, chữ số, gạch nối, dấu chấm
	// / phẩy hay gạch chéo là một phần của mã hoặc của số khác ("EX-03", "R-004"), không đếm.
	numRe = regexp.MustCompile(`(?:^|[^\p{L}\p{N}\-.,/])(-?\d{1,3}(?:\.\d{3})+(?:,\d+)?|-?\d+(?:,\d+)?)`)
)

// CountNumbers trả các con số (dạng chuẩn: "3850", "0.35", "-68") và các ngày (dạng YYYY-MM-DD)
// trong văn bản.
func CountNumbers(text string) (numbers, dates []string) {
	for _, m := range dateRe.FindAllStringSubmatchIndex(text, -1) {
		d, mo, y := text[m[2]:m[3]], text[m[4]:m[5]], text[m[6]:m[7]]
		dates = append(dates, fmt.Sprintf("%s-%02s-%02s", y, mo, d))
	}
	masked := dateRe.ReplaceAllStringFunc(text, func(s string) string { return strings.Repeat(" ", len(s)) })
	for _, m := range numRe.FindAllStringSubmatch(masked, -1) {
		numbers = append(numbers, strings.ReplaceAll(strings.ReplaceAll(m[1], ".", ""), ",", "."))
	}
	return numbers, dates
}

// BindClaims kiểm một câu trả lời số liệu:
//  1. mỗi số trong câu bằng giá trị của một claim, mỗi ngày trong câu là đầu hoặc cuối kỳ của một
//     claim — không số nào "trôi" khỏi claim;
//  2. mỗi claim có giá trị khớp đủ sáu chiều với bằng chứng nó trỏ tới: cùng chủ thể, chỉ số, kỳ,
//     nguồn; có giá trị cùng đơn vị; tổng giá trị các bằng chứng đó bằng giá trị claim.
//
// ignore: chuỗi bỏ đi trước khi đếm — tên / khoá thực thể ("EX-03" là tên, không phải số liệu).
func BindClaims(text string, ignore []string, claims []contracts.Claim, ev []contracts.Evidence) []contracts.Violation {
	var out []contracts.Violation
	bad := func(code, detail string) {
		out = append(out, contracts.Violation{Validator: "claim_binding", Code: code, Severity: contracts.SeverityS1, Detail: detail})
	}
	for _, s := range ignore {
		if s != "" {
			text = strings.ReplaceAll(text, s, " ")
		}
	}
	numbers, dates := CountNumbers(text)
	for _, n := range numbers {
		found := false
		for _, c := range claims {
			if c.Value != nil {
				if cmp, err := decimal.Cmp(n, c.Value.Number); err == nil && cmp == 0 {
					found = true
				}
			}
		}
		if !found {
			bad("NUMBER_UNBOUND", "số "+n+" trong câu không thuộc claim nào")
		}
	}
	for _, d := range dates {
		found := false
		for _, c := range claims {
			from, to, ok := strings.Cut(c.Period, "/")
			found = found || d == from || (ok && d == to)
		}
		if !found {
			bad("DATE_UNBOUND", "ngày "+d+" trong câu không là kỳ của claim nào")
		}
	}

	byID := map[string]contracts.Evidence{}
	for _, e := range ev {
		byID[e.ID] = e
	}
	for _, c := range claims {
		if c.Value == nil {
			continue // claim trích văn bản tài liệu: Bound + ForbidDimensions kiểm
		}
		if dims := claimMismatch(c, byID); len(dims) > 0 {
			bad("CLAIM_UNBOUND", fmt.Sprintf("claim %s %s %s = %s %s (nguồn %s) lệch ở: %s",
				c.Subject, c.Metric, c.Period, c.Value.Number, c.Value.Unit, c.Source, strings.Join(dims, ", ")))
		}
	}
	return out
}

// claimMismatch trả các chiều mà claim không khớp với bằng chứng nó trỏ tới (rỗng = khớp đủ).
func claimMismatch(c contracts.Claim, byID map[string]contracts.Evidence) []string {
	if len(c.EvidenceIDs) == 0 {
		return []string{"không trỏ bằng chứng nào"}
	}
	seen := map[string]bool{}
	var dims []string
	add := func(d string) {
		if !seen[d] {
			seen[d] = true
			dims = append(dims, d)
		}
	}
	sum := "0"
	for _, id := range c.EvidenceIDs {
		e, ok := byID[id]
		if !ok {
			add("bằng chứng " + id + " không có trong ngữ cảnh")
			continue
		}
		if e.Subject != c.Subject {
			add("chủ thể")
		}
		if e.Metric != c.Metric {
			add("chỉ số")
		}
		if e.Period != c.Period {
			add("kỳ")
		}
		if e.Source.SourceType != c.Source {
			add("nguồn")
		}
		unit := false
		for _, v := range e.Values {
			if v.Unit == c.Value.Unit && v.Number != "" {
				unit = true
				if s, err := decimal.Add(sum, v.Number); err == nil {
					sum = s
				}
				break
			}
		}
		if !unit {
			add("đơn vị")
		}
	}
	if cmp, err := decimal.Cmp(sum, c.Value.Number); err != nil || cmp != 0 {
		add("giá trị")
	}
	return dims
}
