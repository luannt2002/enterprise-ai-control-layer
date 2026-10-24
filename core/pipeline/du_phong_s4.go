package pipeline

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"time"

	"github.com/luannt2002/enterprise-ai-control-layer/core/contracts"
	"github.com/luannt2002/enterprise-ai-control-layer/data/normalization"
	"github.com/luannt2002/enterprise-ai-control-layer/intelligence/intent"
)

// KnowledgeProbe là dự phòng S4 tất định (phương án B1, docs/ROADMAP-RAG.md §10.2b): luật không nhận ra ý định nào — không khớp, hoặc khớp
// từ khoá mà thiếu thuật ngữ ("Kế toán đối chiếu hoá đơn lúc nào?") — thì dò chỉ mục tri thức bằng đúng quyền và ngày hỏi của người hỏi;
// có đoạn qua ngưỡng "biết im" (hay chỉ mục báo thông báo phiên bản) thì coi là câu hỏi tri thức. KHÔNG đụng: ý định số liệu thiếu khe và
// câu hỏi lại nghĩa số liệu (thực thể + thuật ngữ) — soát độc lập 24/10 tái lập 9/9 câu số liệu không có "bao nhiêu" ("Hôm qua kho xuất
// mấy lít dầu?") bị trả lời bằng đoạn quy trình qua hai nhánh đó; nhiều ý định hoà nhau. Không gọi model khi chỉ mục chỉ dùng từ khoá.
type KnowledgeProbe struct {
	Knowledge KnowledgeRetriever
	Name      string // ý định tri thức (route RAG) của pack
}

func (p KnowledgeProbe) Intent() string { return p.Name }

func (p KnowledgeProbe) Fallback(ctx context.Context, pr contracts.Principal, in contracts.NormalizedInput, cls intent.Result) (bool, float64, string, error) {
	if cls.Intent.Name != "" || len(cls.NeedMeaning) > 0 || len(cls.Ambiguous) > 0 || p.Knowledge == nil {
		return false, 0, "", nil
	}
	ev, err := p.Knowledge.Retrieve(ctx, pr, in.Raw, knowledgeK)
	if n := (*contracts.KnowledgeNotice)(nil); errors.As(err, &n) {
		return true, 1, "luật không nhận ra ý định; chỉ mục có tài liệu khớp nhưng phải báo phiên bản", nil // route tri thức hiện thông báo
	}
	if err != nil || len(ev) == 0 {
		return false, 0, "", err
	}
	return true, 1, fmt.Sprintf("luật không nhận ra ý định; chỉ mục tri thức có %d đoạn người hỏi được đọc qua ngưỡng", len(ev)), nil
}

// asksQuantity: câu hỏi lượng — mang dấu hiệu hỏi lượng của pack ("bao nhiêu", "định mức"; gõ không dấu vẫn khớp), hoặc nêu một con số
// thuộc chiều đo nghiệp vụ ("đổ 4 khối dầu đúng không?" — câu xác nhận số, bộ độc lập S4 24/10, S4M-46). Rộng hơn asksFactQuantity
// (chốt đó đòi CẢ HAI nên để lọt "xuất kho bao nhiêu lít?"): tài liệu không phải nguồn của số (P4), nên dự phòng S4 thà bỏ sót câu
// quy trình hỏi "trong bao nhiêu giờ" còn hơn trả lời câu số liệu bằng đoạn quy trình.
func (a *Ask) asksQuantity(in contracts.NormalizedInput) bool {
	if slices.ContainsFunc(a.RT.QuantityMarkers, func(m string) bool { return normalization.ContainsPhrase(in.Text, m) }) {
		return true
	}
	// chỉ lượng CÓ SỐ (span lượng), không tính thuật ngữ có đơn vị mặc định — "dầu" một mình không phải câu hỏi lượng
	fact := a.factDimensions()
	return slices.ContainsFunc(in.Spans, func(sp contracts.Span) bool {
		u, ok := a.RT.Normalizer.Unit(sp.Value.Unit)
		return sp.Kind == contracts.ValueQuantity && ok && fact[u.Dimension]
	})
}

// fallbackIntent hỏi dự phòng S4. Câu hỏi lượng (asksQuantity), hay người hỏi thiếu capability của ý định tri thức thì không hỏi (không dò tài liệu hộ người không được
// đọc); dự phòng lỗi của BÊN THỨ BA (hết hạn mức, quá giờ…) thì giữ kết quả của luật và ghi vết — luật là đường mặc định; lỗi khác
// (hạ tầng, chính sách, mã) làm hỏng lượt như ở route tri thức. Ngữ cảnh mang
// đúng truy vấn tri thức của answerKnowledge (ngày hỏi, phạm vi, chủ đề) để dò chỉ mục thấy đúng những gì route RAG sẽ thấy.
func (a *Ask) fallbackIntent(ctx context.Context, now time.Time, pr contracts.Principal, in contracts.NormalizedInput, cls intent.Result, f IntentFallback) (intent.Result, error) {
	def, ok := a.RT.Intents.Def(f.Intent())
	if !ok || contracts.RouteKind(def.Route) != contracts.RouteRAG {
		return cls, contracts.Errorf(contracts.FaultCode, "dự phòng S4 khai ý định không phải ý định tri thức của pack: "+f.Intent())
	}
	if (def.Capability != "" && !a.RT.Policy.Allows(pr, def.Capability)) || a.asksQuantity(in) {
		return cls, nil
	}
	kctx := contracts.WithKnowledgeQuery(ctx, contracts.KnowledgeQuery{AsOf: now, InScope: a.RT.Policy.InScope, Topic: KnowledgeTopic(in)})
	use, conf, why, err := f.Fallback(kctx, pr, in, cls)
	switch {
	case err != nil && !contracts.IsThirdParty(err):
		return cls, err // lỗi của mình / hạ tầng / chính sách: như route tri thức, không che thành "giữ luật"
	case err != nil:
		cls.Why.Reasons = append(slices.Clone(cls.Why.Reasons), "dự phòng S4 lỗi ("+contracts.FaultOf(err)+") — giữ kết quả của luật")
	case use:
		cls = intent.Result{
			Intent: contracts.Intent{Name: def.Name, Confidence: conf, Slots: map[string]contracts.Slot{}, Rule: "intent:fallback"},
			Why:    intent.Why{Reasons: append(slices.Clone(cls.Why.Reasons), "dự phòng S4 → "+def.Name+": "+why), Rejected: cls.Why.Rejected},
		}
	}
	return cls, nil
}
