package pipeline

import "github.com/luannt2002/enterprise-ai-control-layer/core/contracts"

// KnowledgeTopic là phần câu hỏi nói về CHỦ ĐỀ, đưa cho chỉ mục tri thức (contracts.KnowledgeQuery.Topic): câu gốc bỏ các span con
// số, lượng, tiền, ngày, kỳ, giờ, mã mà S2 đã chuẩn hoá — đó là đối tượng của câu (máy nào, hôm nào, bao nhiêu), không phải điều tài
// liệu nói tới. Thuật ngữ của pack ở lại: chính nó là điều được hỏi.
func KnowledgeTopic(in contracts.NormalizedInput) string {
	b := []byte(in.Raw)
	for _, s := range in.Spans {
		if s.Kind == contracts.ValueTerm || s.Start < 0 || s.End > len(b) || s.Start >= s.End {
			continue
		}
		for i := s.Start; i < s.End; i++ {
			b[i] = ' ' // span nằm trọn trên ranh ký tự (vị trí byte của S2) nên thay cả dải không làm hỏng UTF-8
		}
	}
	return string(b)
}
