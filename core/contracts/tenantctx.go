package contracts

import "context"

type tenantKey struct{}

// WithTenant gắn tenant của Principal ĐÃ XÁC THỰC vào ngữ cảnh. Chỉ S1 của luồng gọi hàm này;
// tầng lưu trữ đọc lại bằng TenantFrom để đặt GUC eacl.tenant_id cho RLS — tenant không bao
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
