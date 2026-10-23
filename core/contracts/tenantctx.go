package contracts

import (
	"context"
	"time"
)

type tenantKey struct{}

type knowledgeKey struct{}

// WithTenant gắn tenant của Principal ĐÃ XÁC THỰC vào ngữ cảnh. Chỉ S1 của luồng gọi hàm này;
// tầng lưu trữ đọc lại bằng TenantFrom để đặt GUC eais.tenant_id cho RLS — tenant không bao
// giờ đến từ thân yêu cầu.
func WithTenant(ctx context.Context, id TenantID) context.Context {
	return context.WithValue(ctx, tenantKey{}, id)
}

// TenantFrom trả tenant đã gắn vào ngữ cảnh; không có (hoặc rỗng) thì ok = false — nơi gọi
// phải từ chối, không truy vấn.
func TenantFrom(ctx context.Context) (TenantID, bool) {
	id, ok := ctx.Value(tenantKey{}).(TenantID)
	return id, ok && id != ""
}

// KnowledgeQuery là điều kiện của LƯỢT hỏi mà chỉ mục tri thức cần ngoài principal: ngày hỏi theo giờ tenant (xét hiệu lực
// tài liệu) và luật phạm vi của policy tenant (tài liệu có phạm vi). Pipeline đặt trước khi truy xuất, nên chỉ mục trong bộ nhớ
// và chỉ mục trên PG xét cùng một ngày với phần còn lại của lượt — kể cả khi phát lại một lượt cũ.
type KnowledgeQuery struct {
	AsOf    time.Time
	InScope func(pr Principal, keys []string) bool // nil = chỉ mục tự xét theo thuộc tính "entity_scope"
	// Topic là phần câu hỏi nói về chủ đề (câu gốc bỏ con số / ngày / mã đã chuẩn hoá ở S2): chỉ mục dùng nó để biết kho có nói
	// tới điều được hỏi không. Rỗng = không xét.
	Topic string
}

// WithKnowledgeQuery gắn điều kiện truy xuất tri thức của lượt vào ngữ cảnh.
func WithKnowledgeQuery(ctx context.Context, q KnowledgeQuery) context.Context {
	return context.WithValue(ctx, knowledgeKey{}, q)
}

// KnowledgeQueryFrom trả điều kiện đã gắn; không có thì ok = false.
func KnowledgeQueryFrom(ctx context.Context) (KnowledgeQuery, bool) {
	q, ok := ctx.Value(knowledgeKey{}).(KnowledgeQuery)
	return q, ok
}
