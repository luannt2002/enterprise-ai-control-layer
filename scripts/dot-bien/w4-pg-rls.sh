#!/usr/bin/env bash
# Danh mục đột biến "W4 — PostgreSQL RLS / GUC (cần PG của deploy/docker-compose.yml)" đã chạy ngày 20/10/2026 (xem docs/ops/NHAT-KY-2026-10-20.md).
# Mỗi dòng: m <tệp> <chuỗi gốc> <chuỗi đột biến> <nhãn>. Kỳ vọng: mọi dòng in "BỊ BẮT" (rc=0).
# Mã đổi sau ngày đó thì chuỗi gốc có thể không còn khớp (mutate.sh báo "xuất hiện 0 lần", rc=2):
# sửa chuỗi theo mã mới rồi chạy lại — đừng xoá dòng cho xanh.
cd "$(dirname "$0")/../.." || exit 1
export EAIS_PG_ADMIN_DSN='postgres://postgres@127.0.0.1:25432/eais?sslmode=disable'
export EAIS_PG_DSN='postgres://eais_app@127.0.0.1:25432/eais?sslmode=disable'
export PKGS="./platform/... ./tenants/construction/win/winrun/ ./core/..."
m() { scripts/mutate.sh "$1" "$2" "$3" "$4"; echo "rc=$?"; }
# Đột biến MIGRATION: migration chỉ áp MỘT lần mỗi database, nên mỗi đột biến chạy trên database eais_dot_bien vừa tạo lại
# (bản SQL đột biến được áp ở lần Migrate đầu tiên của test). Vai là của cả cụm PG: đột biến "không tạo vai" không đo được ở đây.
DB_DOT_BIEN=eais_dot_bien
fresh() { psql -q -X "$EAIS_PG_ADMIN_DSN" -c "DROP DATABASE IF EXISTS $DB_DOT_BIEN WITH (FORCE)" -c "CREATE DATABASE $DB_DOT_BIEN"; }
mm() {
	fresh || { echo "[$4] không dựng được database $DB_DOT_BIEN"; echo "rc=5"; return; }
	EAIS_PG_ADMIN_DSN="${EAIS_PG_ADMIN_DSN/\/eais\?//$DB_DOT_BIEN?}" EAIS_PG_DSN="${EAIS_PG_DSN/\/eais\?//$DB_DOT_BIEN?}" m "$@"
}
m platform/pgstore/pgstore.go '		if _, err := tx.Exec(ctx, "SELECT set_config('"'"'eais.tenant_id'"'"', $1, true)", string(tenant)); err != nil {' '		if _, err := tx.Exec(ctx, "SELECT 1 WHERE $1::text IS NOT NULL", string(tenant)); err != nil {' 'PA không đặt GUC'
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
M2=db/migrations/0002_dinh_danh_eais.sql
mm $M2 "      USING (tenant_id = current_setting('eais.tenant_id', true))" "      USING (tenant_id = current_setting('eacl.tenant_id', true))" 'MG1 0002 để chính sách đọc GUC cũ'
mm $M2 "      WITH CHECK (tenant_id = current_setting('eais.tenant_id', true))" "      WITH CHECK (tenant_id = current_setting('eacl.tenant_id', true))" 'MG2 0002 để WITH CHECK theo GUC cũ'
mm $M2 "    EXECUTE format('GRANT SELECT, INSERT, DELETE ON %I TO eais_app', t);" "    NULL;" 'MG3 0002 không cấp quyền cho eais_app'
mm $M2 "      EXECUTE format('REVOKE ALL ON %I FROM eacl_app', t);" "      NULL;" 'MG4 0002 không gỡ quyền vai cũ'
mm $M2 "      EXECUTE format('REVOKE ALL ON %I FROM eacl_app', t);" "      EXECUTE format('REVOKE SELECT ON %I FROM eacl_app', t);" 'MG5 0002 chỉ gỡ SELECT của vai cũ'
mm $M2 "    EXECUTE format('GRANT SELECT, INSERT, DELETE ON %I TO eais_app', t);" "    EXECUTE format('GRANT SELECT, INSERT, UPDATE, DELETE ON %I TO eais_app', t);" 'MG6 0002 cấp eais_app rộng hơn vai cũ'
mm $M2 "    EXECUTE format('GRANT SELECT, INSERT, DELETE ON %I TO eais_app', t);" "    EXECUTE format('GRANT SELECT, INSERT, DELETE, TRUNCATE ON %I TO eais_app', t);" 'X1 0002 cấp thêm TRUNCATE (không chịu RLS)'
mm $M2 "    EXECUTE format('GRANT SELECT, INSERT, DELETE ON %I TO eais_app', t);" "    EXECUTE format('GRANT SELECT, INSERT, DELETE, REFERENCES, TRIGGER ON %I TO eais_app', t);" 'X2 0002 cấp thêm REFERENCES, TRIGGER'
mm $M2 "    EXECUTE format('GRANT SELECT, INSERT, DELETE ON %I TO eais_app', t);" "    EXECUTE format('GRANT SELECT, INSERT, DELETE ON %I TO eais_app WITH GRANT OPTION', t);" 'X3 0002 cấp kèm GRANT OPTION'
mm $M2 "    EXECUTE format('GRANT SELECT, INSERT, DELETE ON %I TO eais_app', t);" "    EXECUTE format('ALTER TABLE %I NO FORCE ROW LEVEL SECURITY', t); EXECUTE format('GRANT SELECT, INSERT, DELETE ON %I TO eais_app', t);" 'X4 0002 bỏ FORCE RLS'
mm $M2 "    EXECUTE format('GRANT SELECT, INSERT, DELETE ON %I TO eais_app', t);" "    EXECUTE format('GRANT SELECT, INSERT, DELETE ON %I TO eais_app, PUBLIC', t);" 'X5 0002 cấp qua PUBLIC (vai cũ có lại quyền)'
psql -q -X "$EAIS_PG_ADMIN_DSN" -c "DROP DATABASE IF EXISTS $DB_DOT_BIEN WITH (FORCE)"
