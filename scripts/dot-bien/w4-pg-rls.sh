#!/usr/bin/env bash
# Danh mục đột biến "W4 — PostgreSQL RLS / GUC (cần PG của deploy/docker-compose.yml)" đã chạy ngày 20/10/2026 (xem docs/ops/NHAT-KY-2026-10-20.md).
# Mỗi dòng: m <tệp> <chuỗi gốc> <chuỗi đột biến> <nhãn>. Kỳ vọng: mọi dòng in "BỊ BẮT" (rc=0).
# Mã đổi sau ngày đó thì chuỗi gốc có thể không còn khớp (mutate.sh báo "xuất hiện 0 lần", rc=2):
# sửa chuỗi theo mã mới rồi chạy lại — đừng xoá dòng cho xanh.
cd "$(dirname "$0")/../.." || exit 1
export EACL_PG_ADMIN_DSN='postgres://postgres@127.0.0.1:25432/eacl?sslmode=disable'
export EACL_PG_DSN='postgres://eacl_app@127.0.0.1:25432/eacl?sslmode=disable'
export PKGS="./platform/... ./tenants/construction/win/winrun/ ./core/..."
m() { scripts/mutate.sh "$1" "$2" "$3" "$4"; echo "rc=$?"; }
m platform/pgstore/pgstore.go '		if _, err := tx.Exec(ctx, "SELECT set_config('"'"'eacl.tenant_id'"'"', $1, true)", string(tenant)); err != nil {' '		if _, err := tx.Exec(ctx, "SELECT 1 WHERE $1::text IS NOT NULL", string(tenant)); err != nil {' 'PA không đặt GUC'
m platform/pgstore/pgstore.go '	t, ok := contracts.TenantFrom(ctx)
	if !ok {' '	t, ok := contracts.TenantFrom(ctx)
	if false && !ok {' 'PD ngữ cảnh không tenant vẫn truy vấn'
m platform/pgstore/pgstore.go '	if t, ok := contracts.TenantFrom(ctx); ok && t != pr.TenantID {' '	if t, ok := contracts.TenantFrom(ctx); false && ok && t != pr.TenantID {' 'PE Retrieve bỏ so principal với ngữ cảnh'
m core/pipeline/ask.go '	ctx = contracts.WithTenant(ctx, pr.TenantID)' '	_ = contracts.WithTenant' 'PF S1 không gắn tenant vào ngữ cảnh'
m platform/pgstore/load.go '		for _, table := range []string{"entity_aliases", "facts", "documents", "entities"} {' '		for _, table := range []string{} {' 'PG Load không thay dữ liệu cũ'
m platform/pgstore/pgstore.go '			f.OccurredOn = time.Date(on.Year(), on.Month(), on.Day(), 0, 0, 0, 0, from.Location())' '			f.OccurredOn = on' 'PH ngày fact sai múi giờ'
m platform/pgstore/pgstore.go '	if super || bypass {' '	if false {' 'BR1 Open không kiểm vai vượt RLS'
m platform/pgstore/pgstore.go '	if super || bypass {' '	if super {' 'BR2 Open chỉ chặn superuser, lọt vai BYPASSRLS'
m platform/pgstore/pgstore.go '	if super || bypass {' '	if bypass {' 'BR3 Open chỉ chặn BYPASSRLS, lọt superuser'
m platform/pgstore/pgstore.go 'WHERE rolname = current_user' 'WHERE rolname = session_user' 'BR4 Open xét vai đăng nhập (session_user) thay vì vai hiệu lực'
