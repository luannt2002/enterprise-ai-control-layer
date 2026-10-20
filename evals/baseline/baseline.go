// Package baseline là đối thủ của EAIS ở cổng G7 (docs/04-core-mvp.md §4, docs/adr/0001-win-mvp.md D6):
// một RAG đơn giản trên dữ liệu của MỘT tenant, dùng cùng chỉ mục (intelligence/rag) và cùng gateway
// model với EAIS, và KHÔNG có tầng kiểm soát nào của EAIS — không lọc quyền theo người hỏi, không
// route, không ràng buộc claim vào bằng chứng, không hỏi lại, không từ chối.
//
// Một lượt hỏi = truy xuất top-k đoạn → MỘT lời gọi model → trả nguyên đầu ra của model. Nó có mặt để
// G7 chứng minh tầng kiểm soát đáng giá: nếu RAG trần làm tốt ngang EAIS thì luận đề sai.
package baseline

import (
	"context"
	"errors"
	"fmt"
	"math"
	"sort"
	"strings"
	"sync/atomic"

	"github.com/luannt2002/enterprise-ai-systems/core/contracts"
	"github.com/luannt2002/enterprise-ai-systems/intelligence/models"
	"github.com/luannt2002/enterprise-ai-systems/intelligence/rag"
)

// DefaultTopK là số đoạn đưa cho model mỗi lượt. Không lấy 3 (knowledgeK của route tri thức EAIS):
// EAIS trả số liệu bằng SQL nên model không phải chọn giữa nhiều dòng, còn baseline thì phải, và truy
// xuất theo từ khoá trên hàng nghìn dòng na ná nhau hiếm khi đưa đoạn đúng vào top-3. Đo 20/10/2026
// (hồ sơ functional, seed 42, 2702 đoạn, bộ ca độc lập): trong 70 ca có expect.value, 60 ca có chuỗi chữ
// số của đáp án nằm trong ít nhất một đoạn; chuỗi đó có mặt trong top-3 ở 10 ca, top-5 ở 14, top-8 ở 22,
// top-10 ở 43, top-20 ở 47. Baseline thua vì truy xuất chứ không vì thiếu tầng kiểm soát thì G7 không
// chứng minh được điều nó định chứng minh: đối thủ thành hình nộm. Đổi qua RAG.TopK; chạy lại phép đo
// khi dữ liệu hay cách viết đoạn đổi.
const DefaultTopK = 10

// openRole là vai duy nhất baseline dùng, cho MỌI đoạn và MỌI lượt truy xuất. rag.MemoryIndex buộc lọc
// ACL theo vai và không có cách tắt (không sửa được ở đây), nên baseline làm cho bộ lọc đó vô hiệu:
// mọi đoạn đều có ACL = [openRole], người đọc luôn mang đúng vai ấy. Vai và thuộc tính thật của người
// hỏi (director, driver, entity_scope...) không bao giờ tới được chỉ mục. Đây là THIẾT KẾ của baseline.
const openRole = "*"

// RAG là baseline. Dùng được như evals.Asker và evals.ModelCounter.
type RAG struct {
	// TopK là số đoạn tối đa truy xuất mỗi lượt; ≤ 0 thì dùng DefaultTopK.
	TopK int

	index *rag.MemoryIndex
	model models.Gateway
	calls atomic.Int64
}

// New dựng baseline cho MỘT tenant từ:
//   - paragraphs: mỗi quan sát / bản ghi nguồn của tenant một đoạn văn bản (một dòng; xuống dòng bên
//     trong bị gộp, vì chỉ mục cắt đoạn theo dòng);
//   - docs: các tài liệu (SOP) — ACL gốc bị thay bằng ACL mở, tài liệu của tenant khác bị chỉ mục bỏ
//     ngay lúc dựng (chỉ dữ liệu tenant được chấm).
func New(tenant string, paragraphs []string, docs []rag.Document, model models.Gateway) *RAG {
	all := make([]rag.Document, 0, len(paragraphs)+len(docs))
	for _, p := range paragraphs {
		if p = strings.Join(strings.Fields(p), " "); p != "" {
			all = append(all, rag.Document{ID: fmt.Sprintf("P-%06d", len(all)+1), Tenant: tenant, Body: p, ACL: []string{openRole}})
		}
	}
	for _, d := range docs {
		d.ACL = []string{openRole}
		all = append(all, d)
	}
	return &RAG{TopK: DefaultTopK, index: rag.NewMemoryIndex(tenant, all), model: model}
}

// Ask truy xuất top-k đoạn cho câu hỏi, gọi model đúng MỘT lần (kể cả khi không có đoạn nào) và trả
// nguyên đầu ra của model. Route luôn là RAG. Outcome là RESOLVED, hoặc INSUFFICIENT_EVIDENCE khi
// không có đoạn nào — nhãn đó chỉ phản ánh truy xuất; không gì kiểm câu chữ model trả, nên model có
// thể vẫn nói một con số dù nhãn là thiếu bằng chứng: đúng cái baseline thiếu.
//
// Chỉ mục vẫn kiểm tenant của người hỏi so với tenant của dữ liệu (một chỉ mục = một tenant): người
// hỏi của tenant khác nhận lỗi chính sách chứ không nhận đoạn nào.
func (b *RAG) Ask(ctx context.Context, pr contracts.Principal, question string) (contracts.AskResponse, error) {
	if b.model == nil {
		return contracts.AskResponse{}, errors.New("baseline cần gateway model")
	}
	k := b.TopK
	if k <= 0 {
		k = DefaultTopK
	}
	reader := contracts.Principal{ID: pr.ID, TenantID: pr.TenantID, Kind: pr.Kind, Roles: []string{openRole}}
	// Lấy MỌI đoạn đạt ngưỡng rồi tự xếp lại (điểm làm tròn 1e-9, hoà thì theo id đoạn) để thứ tự
	// top-k của baseline không phụ thuộc chi tiết xếp hạng của rag. (Đo 20/10: rag từng cộng điểm theo
	// thứ tự duyệt map nên đoạn hoà điểm đổi chỗ giữa các lần hỏi; rag đã sửa thành tất định cùng ngày.)
	ev, err := b.index.Retrieve(ctx, reader, question, math.MaxInt)
	if err != nil {
		return contracts.AskResponse{}, err
	}
	sort.SliceStable(ev, func(i, j int) bool {
		if si, sj := math.Round(ev[i].Confidence*1e9), math.Round(ev[j].Confidence*1e9); si != sj {
			return si > sj
		}
		return ev[i].ID < ev[j].ID
	})
	if len(ev) > k {
		ev = ev[:k]
	}
	b.calls.Add(1)
	draft, err := b.model.Generate(ctx, models.Request{Question: question, Evidence: ev})
	if err != nil {
		return contracts.AskResponse{}, err
	}
	outcome := contracts.OutcomeResolved
	if len(ev) == 0 {
		outcome = contracts.OutcomeInsufficientEvidence
	}
	return contracts.AskResponse{Route: contracts.RouteRAG, Outcome: outcome, Answer: draft.Text}, nil
}

// ModelCalls trả số lời gọi model baseline đã thực hiện kể từ lần ResetModelCalls gần nhất.
func (b *RAG) ModelCalls() int { return int(b.calls.Load()) }

// ResetModelCalls đặt lại bộ đếm lời gọi model.
func (b *RAG) ResetModelCalls() { b.calls.Store(0) }
