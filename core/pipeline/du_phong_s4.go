package pipeline

import (
	"context"
	"fmt"
	"slices"

	"github.com/luannt2002/enterprise-ai-control-layer/core/contracts"
	"github.com/luannt2002/enterprise-ai-control-layer/data/normalization"
	"github.com/luannt2002/enterprise-ai-control-layer/intelligence/intent"
)

// KnowledgeProbe là dự phòng S4 tất định (phương án B1, docs/ROADMAP-RAG.md §10.2): luật KHÔNG ra được kế hoạch trả lời — không ý định
// nào khớp, khớp từ khoá mà thiếu thuật ngữ, chỉ còn hỏi lại nghĩa số liệu, hay khớp ý định số liệu mà thiếu khe bắt buộc — thì dò chỉ mục
// tri thức bằng đúng quyền và ngày hỏi của người hỏi; có đoạn qua ngưỡng "biết im" thì coi là câu hỏi tri thức. Nhiều ý định hoà nhau và
// ý định đã đủ khe: không đụng. Đo 24/10 trên chính pack: "Kế toán đối chiếu hoá đơn lúc nào?" khớp nhầm ý định hoá đơn rồi hỏi lại.
// Không gọi model khi chỉ mục chỉ dùng từ khoá.
type KnowledgeProbe struct {
	Knowledge KnowledgeRetriever
	Name      string // ý định tri thức (route RAG) của pack
}

func (p KnowledgeProbe) Intent() string { return p.Name }

func (p KnowledgeProbe) Fallback(ctx context.Context, pr contracts.Principal, in contracts.NormalizedInput, cls intent.Result) (bool, float64, string, error) {
	// thiếu thuật ngữ / hỏi lại nghĩa số liệu / không khớp đều để ý định rỗng (intent.Classify)
	if (cls.Intent.Name != "" && len(cls.Intent.Missing) == 0) || len(cls.Ambiguous) > 0 || p.Knowledge == nil {
		return false, 0, "", nil
	}
	ev, err := p.Knowledge.Retrieve(ctx, pr, in.Raw, knowledgeK)
	if err != nil || len(ev) == 0 {
		return false, 0, "", err
	}
	return true, 1, fmt.Sprintf("luật không ra kế hoạch trả lời; chỉ mục tri thức có %d đoạn người hỏi được đọc qua ngưỡng", len(ev)), nil
}

// asksQuantity: câu mang dấu hiệu hỏi lượng của pack ("bao nhiêu", "định mức"; gõ không dấu vẫn khớp). Rộng hơn asksFactQuantity (chốt
// đó đòi một con số kèm đơn vị nghiệp vụ nên để lọt "xuất kho bao nhiêu lít?"): tài liệu không phải nguồn của số (P4), nên dự phòng S4
// thà bỏ sót câu quy trình hỏi "trong bao nhiêu giờ" còn hơn trả lời câu số liệu bằng đoạn quy trình (đo 24/10, ca trong test).
func (a *Ask) asksQuantity(in contracts.NormalizedInput) bool {
	return slices.ContainsFunc(a.RT.QuantityMarkers, func(m string) bool { return normalization.ContainsPhrase(in.Text, m) })
}
