// Package validation là chặng S9: kiểm câu trả lời bằng MÃ trước khi giao.
//
// Luật cứng: model — và cả mẫu câu — không bao giờ là nguồn của một con số. Mọi lượng,
// số tiền, con số và mã trong câu trả lời phải khớp (sau chuẩn hoá) một giá trị có trong
// gói bằng chứng. Kiểm thất bại thì câu đó không được giao (luật cứng 3 của luồng hỏi).
package validation

import (
	"strings"
	"time"

	"github.com/luannt2002/enterprise-ai-systems/core/contracts"
	"github.com/luannt2002/enterprise-ai-systems/core/decimal"
	"github.com/luannt2002/enterprise-ai-systems/data/normalization"
)

// Fidelity kiểm số và mã trong câu trả lời so với bằng chứng.
//
// ignore là các chuỗi được phép xuất hiện mà không cần bằng chứng số (tên/khoá thực thể
// đã nhận dạng — "EX-03" chứa số 03 nhưng đó là tên, không phải số liệu).
func Fidelity(norm *normalization.Engine, draft contracts.Draft, pkg contracts.ContextPackage, ignore []string, now time.Time) []contracts.Violation {
	text := draft.Text
	for _, s := range ignore {
		if s != "" {
			text = strings.ReplaceAll(text, s, " ")
		}
	}
	got := norm.Normalize(text, normalization.Options{Now: now})

	var out []contracts.Violation
	for _, sp := range got.Spans {
		switch sp.Kind {
		case contracts.ValueQuantity, contracts.ValueMoney, contracts.ValueNumber:
			if !numberBacked(norm, sp.Value, pkg) {
				out = append(out, contracts.Violation{
					Validator: "numeric_fidelity", Code: "number_without_evidence", Severity: contracts.SeverityS1,
					Detail: "câu trả lời có " + sp.Raw + " nhưng không bằng chứng nào chứa giá trị này",
				})
			}
		case contracts.ValueCode:
			if !codeBacked(sp.Value.Text, pkg) {
				out = append(out, contracts.Violation{
					Validator: "code_fidelity", Code: "code_without_evidence", Severity: contracts.SeverityS1,
					Detail: "câu trả lời có mã " + sp.Raw + " nhưng không bằng chứng nào chứa mã này",
				})
			}
		}
	}
	return out
}

// numberBacked: giá trị có mặt trong bằng chứng — cùng đơn vị, hoặc đổi được về cùng đơn
// vị, hoặc là số trần bằng đúng một số trong bằng chứng.
func numberBacked(norm *normalization.Engine, v contracts.Value, pkg contracts.ContextPackage) bool {
	for _, e := range pkg.Evidence {
		for _, ev := range e.Values {
			if ev.Number == "" {
				continue
			}
			if v.Unit == "" || ev.Unit == "" {
				if decimal.Equal(v.Number, ev.Number) {
					return true
				}
				continue
			}
			if v.Unit == ev.Unit {
				if decimal.Equal(v.Number, ev.Number) {
					return true
				}
				continue
			}
			if conv, err := norm.Convert(v, ev.Unit); err == nil && decimal.Equal(conv.Number, ev.Number) {
				return true
			}
		}
	}
	return false
}

func codeBacked(code string, pkg contracts.ContextPackage) bool {
	for _, e := range pkg.Evidence {
		if strings.Contains(e.Text, code) {
			return true
		}
		for _, ev := range e.Values {
			if ev.Text == code {
				return true
			}
		}
	}
	return false
}
