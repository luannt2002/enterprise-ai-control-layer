package pipeline

import (
	"strings"
	"testing"

	"github.com/luannt2002/enterprise-ai-control-layer/core/contracts"
)

// KnowledgeTopic: chủ đề của câu hỏi cho chỉ mục tri thức = câu gốc bỏ con số, ngày, mốc kỳ, giờ, mã mà S2 đã chuẩn hoá; thuật
// ngữ của pack ở lại (chính là điều được hỏi).
func TestKnowledgeTopic(t *testing.T) {
	raw := "EX-03 hôm nay được cấp bao nhiêu dầu, lệch 120 L thì sao?"
	span := func(kind contracts.ValueKind, s string) contracts.Span {
		i := strings.Index(raw, s)
		return contracts.Span{Kind: kind, Raw: s, Start: i, End: i + len(s)}
	}
	in := contracts.NormalizedInput{Raw: raw, Spans: []contracts.Span{
		span(contracts.ValueCode, "EX-03"), span(contracts.ValuePeriod, "hôm nay"), span(contracts.ValueTerm, "dầu"),
		span(contracts.ValueQuantity, "120 L"),
	}}
	got := KnowledgeTopic(in)
	for _, bo := range []string{"EX-03", "hôm nay", "120"} {
		if strings.Contains(got, bo) {
			t.Errorf("chủ đề còn %q: %q", bo, got)
		}
	}
	for _, giu := range []string{"được cấp bao nhiêu", "dầu", "lệch", "thì sao"} {
		if !strings.Contains(got, giu) {
			t.Errorf("chủ đề mất %q: %q", giu, got)
		}
	}
	// Span hỏng (vị trí ngoài câu, chồng nhau) không làm hàm panic hay cắt sai.
	in.Spans = append(in.Spans, contracts.Span{Kind: contracts.ValueNumber, Start: 3, End: 1}, contracts.Span{Kind: contracts.ValueNumber, Start: 0, End: 999})
	_ = KnowledgeTopic(in)
}
