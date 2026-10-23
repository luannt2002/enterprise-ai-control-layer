#!/usr/bin/env bash
# Danh mục đột biến "W4 — PostgreSQL RLS / GUC (cần PG của deploy/docker-compose.yml)" đã chạy ngày 20/10/2026 (xem docs/ops/NHAT-KY-2026-10-20.md).
# Mỗi dòng: m <tệp> <chuỗi gốc> <chuỗi đột biến> <nhãn>. Kỳ vọng: mọi dòng in "BỊ BẮT" (rc=0); cuối cùng in tổng và thoát 0 chỉ khi
# không đột biến nào sống sót hay "khác" (nền đỏ rc=5, đỏ sai chỗ rc=6, chuỗi gốc không khớp rc=2…).
# Mã đổi sau ngày đó thì chuỗi gốc có thể không còn khớp (mutate.sh báo "xuất hiện 0 lần", rc=2):
# sửa chuỗi theo mã mới rồi chạy lại — đừng xoá dòng cho xanh.
# Khối SN (cuối tệp, 23/10/2026, việc 5e): sổ nạp load_records — migration 0006, pgstore/load.go, Ready và cách cmd/eais đọc phiên bản dữ liệu. Mỗi dòng khẳng định
# test NHẮM TỚI (MUTATE_TEST_MUON); đỏ ở test khác là "đỏ sai chỗ" (rc=6, tính vào "khác"). Chạy riêng khối đó: DOT_BIEN_LOC='^SN' (regex trên nhãn).
# SN35 thêm ở vòng 2 của việc 5e (soát độc lập D5ES-01: đột biến của người soát, đổi tenant ở điểm gọi loadedVersion của route lệnh, SỐNG SÓT khi chỉ có SN1–SN34).
# SN36 cũng thêm ở vòng 2 (D5ES-02: đột biến xếp dòng sổ theo loaded_at SỐNG SÓT); nó nằm cạnh nhóm pgstore (trước SN23) vì dùng chung PKGS=./platform/pgstore/.
cd "$(dirname "$0")/../.." || exit 1
# DSN lấy từ môi trường nếu đã đặt (ví dụ người soát chạy trên database riêng); mặc định là PG dev của deploy/docker-compose.yml.
export EAIS_PG_ADMIN_DSN="${EAIS_PG_ADMIN_DSN:-postgres://postgres@127.0.0.1:25432/eais?sslmode=disable}"
export EAIS_PG_DSN="${EAIS_PG_DSN:-postgres://eais_app@127.0.0.1:25432/eais?sslmode=disable}"
export PKGS="./platform/... ./tenants/construction/win/winrun/ ./core/..."
bat=0 song=0 khac=0
LOC=${DOT_BIEN_LOC:-}
skip() { [ -n "$LOC" ] && ! [[ $1 =~ $LOC ]]; } # 0 (đúng) khi dòng có nhãn $1 bị DOT_BIEN_LOC lọc bỏ
m() {
	skip "$4" && return
	scripts/mutate.sh "$1" "$2" "$3" "$4"
	local rc=$?
	echo "rc=$rc"
	case $rc in 0) bat=$((bat + 1)) ;; 3) song=$((song + 1)) ;; *) khac=$((khac + 1)) ;; esac
}
# Đột biến MIGRATION: migration chỉ áp MỘT lần mỗi database, nên mỗi đột biến chạy trên database eais_dot_bien vừa tạo lại
# (bản SQL đột biến được áp ở lần Migrate đầu tiên của test). Vai là của cả cụm PG: đột biến "không tạo vai" không đo được ở đây.
DB_DOT_BIEN=${DB_DOT_BIEN:-eais_dot_bien}
fresh() { psql -q -X "$EAIS_PG_ADMIN_DSN" -c "DROP DATABASE IF EXISTS $DB_DOT_BIEN WITH (FORCE)" -c "CREATE DATABASE $DB_DOT_BIEN"; }
# dsn_db <dsn> <database>: thay tên database trong DSN dạng postgres://vai@máy:cổng/<database>[?tham số].
dsn_db() { printf '%s' "$1" | sed -E 's#^([a-z]+://[^/]*)/[^/?]*#\1/'"$2"'#'; }
# Nền xanh trên database đột biến (lỗi 5c): chạy `mutate.sh --nen` MỘT lần trên database vừa dựng, rồi dựng lại database cho đột biến —
# để mutate.sh tự chạy nền ở đó thì nền áp migration GỐC trước đột biến và đột biến migration sống giả.
nen_mm=""
mm() {
	skip "$4" && return
	fresh || { echo "[$4] không dựng được database $DB_DOT_BIEN"; echo "rc=5"; khac=$((khac + 1)); return; }
	local a d
	a="$(dsn_db "$EAIS_PG_ADMIN_DSN" "$DB_DOT_BIEN")" d="$(dsn_db "$EAIS_PG_DSN" "$DB_DOT_BIEN")"
	if [ -z "$nen_mm" ]; then
		nen_mm=$(EAIS_PG_ADMIN_DSN=$a EAIS_PG_DSN=$d scripts/mutate.sh --nen)
		echo "$nen_mm"
		fresh || { echo "[$4] không dựng được database $DB_DOT_BIEN"; echo "rc=5"; khac=$((khac + 1)); return; }
	fi
	EAIS_PG_ADMIN_DSN=$a EAIS_PG_DSN=$d MUTATE_NEN_CO_SAN=1 m "$@"
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
mm $M2 "    EXECUTE format('GRANT SELECT, INSERT, DELETE ON %I TO eais_app', t);" "    EXECUTE format('GRANT SELECT, INSERT, DELETE, MAINTAIN ON %I TO eais_app', t);" 'X6 0002 cấp thêm MAINTAIN (PG 17)'
mm $M2 "    EXECUTE format('GRANT SELECT, INSERT, DELETE ON %I TO eais_app', t);" "    EXECUTE format('GRANT SELECT, INSERT, DELETE ON %I TO eais_app', t); EXECUTE format('GRANT UPDATE ON %I TO PUBLIC', t);" 'X7 0002 cấp UPDATE qua PUBLIC'
mm $M2 "    EXECUTE format('GRANT SELECT, INSERT, DELETE ON %I TO eais_app', t);" "    EXECUTE format('GRANT SELECT, INSERT, DELETE ON %I TO eais_app', t); EXECUTE format('ALTER TABLE %I OWNER TO eais_app', t);" 'X9 0002 giao bảng cho eais_app làm chủ'
mm $M2 "    EXECUTE format('GRANT SELECT, INSERT, DELETE ON %I TO eais_app', t);" "    EXECUTE format('GRANT SELECT, INSERT, DELETE ON %I TO eais_app', t); EXECUTE format('GRANT UPDATE (tenant_id) ON %I TO eais_app', t);" 'X10 0002 cấp UPDATE theo cột'
mm $M2 "    EXECUTE format('GRANT SELECT, INSERT, DELETE ON %I TO eais_app', t);" "    EXECUTE format('GRANT SELECT, INSERT, DELETE ON %I TO eais_app', t); EXECUTE format('GRANT REFERENCES (tenant_id) ON %I TO eais_app', t);" 'X11 0002 cấp REFERENCES theo cột'

# ---- Khối SN: sổ nạp load_records (migration 0006) và cách phiên bản dữ liệu đi vào vết ----
# muon <regex> <lệnh m / mm / mpy ...>: đặt MUTATE_TEST_MUON cho đúng một dòng.
muon() { MUTATE_TEST_MUON=$1; export MUTATE_TEST_MUON; shift; "$@"; unset MUTATE_TEST_MUON; }
# mpy <tệp> <đoạn Python> <nhãn>: như m nhưng chuỗi gốc và chuỗi đột biến do đoạn Python sinh từ chính tệp (đột biến dời nhiều đoạn mã liền nhau). Đoạn Python đọc tệp ở
# argv[1], in chuỗi gốc rồi chuỗi đột biến, mỗi chuỗi kết thúc bằng NUL.
mpy() {
	skip "$3" && return
	local -a pair
	mapfile -d '' -t pair < <(python3 -I -c "$2" "$1")
	if [ "${#pair[@]}" -ne 2 ]; then
		echo "[$3] KHÔNG SINH ĐƯỢC ĐỘT BIẾN từ $1 (mã đã đổi, sửa đoạn Python)"; echo "rc=5"; khac=$((khac + 1)); return 0
	fi
	m "$1" "${pair[0]}" "${pair[1]}" "$3"
}
M6=db/migrations/0006_so_nap_du_lieu.sql
L=platform/pgstore/load.go
SCHEMA='TestLoadRecords(Migration0006|TenantChoPhepVaTuChoi)'
export PKGS=./platform/pgstore/
nen_mm="" # PKGS đổi ⇒ lượt nền của mm phải tính lại cho PKGS mới (khoá nền gồm PKGS)
muon "$SCHEMA" mm $M6 "  USING (tenant_id = current_setting('eais.tenant_id', true))" "  USING (true)" 'SN1 sổ nạp: chính sách đọc mở toang'
muon "$SCHEMA" mm $M6 "  WITH CHECK (tenant_id = current_setting('eais.tenant_id', true));" "  WITH CHECK (true);" 'SN2 sổ nạp: chính sách ghi mở toang (ghi chéo tenant)'
muon "$SCHEMA" mm $M6 "ALTER TABLE load_records ENABLE ROW LEVEL SECURITY;" "SELECT 1;" 'SN3 sổ nạp: không bật RLS'
muon "$SCHEMA" mm $M6 "ALTER TABLE load_records FORCE ROW LEVEL SECURITY;" "SELECT 1;" 'SN4 sổ nạp: không FORCE RLS'
muon "$SCHEMA" mm $M6 "GRANT SELECT, INSERT ON load_records TO eais_app;" "GRANT SELECT, INSERT, UPDATE ON load_records TO eais_app;" 'SN5 sổ nạp: cấp thêm UPDATE'
muon "$SCHEMA" mm $M6 "GRANT SELECT, INSERT ON load_records TO eais_app;" "GRANT SELECT, INSERT, DELETE ON load_records TO eais_app;" 'SN6 sổ nạp: cấp thêm DELETE'
muon "$SCHEMA" mm $M6 "GRANT SELECT, INSERT ON load_records TO eais_app;" "GRANT SELECT, INSERT, TRUNCATE ON load_records TO eais_app;" 'SN7 sổ nạp: cấp thêm TRUNCATE (không chịu RLS)'
muon "$SCHEMA" mm $M6 "GRANT SELECT, INSERT ON load_records TO eais_app;" "GRANT SELECT, INSERT ON load_records TO eais_app, PUBLIC;" 'SN8 sổ nạp: cấp qua PUBLIC'
muon "$SCHEMA" mm $M6 "REVOKE ALL ON load_records FROM eais_app;" "SELECT 1;" 'SN9 sổ nạp: chạy lại không gỡ quyền thừa của eais_app'
muon "$SCHEMA" mm $M6 "REVOKE ALL ON load_records FROM PUBLIC;" "SELECT 1;" 'SN10 sổ nạp: chạy lại không gỡ quyền của PUBLIC'
muon "$SCHEMA" mm $M6 "DROP POLICY IF EXISTS tenant_isolation ON load_records;" "SELECT 1;" 'SN11 sổ nạp: chạy lại không dựng lại được chính sách'
muon "$SCHEMA" mm $M6 "CONSTRAINT load_records_tenant_khong_rong CHECK (tenant_id <> ''),"  "CHECK (true)," 'SN12 sổ nạp: tenant rỗng được phép'
muon "$SCHEMA" mm $M6 "PRIMARY KEY (tenant_id, id)," "PRIMARY KEY (id)," 'SN13 sổ nạp: khoá chính không có tenant'
muon 'TestLoadRecordsRangBuocDB' mm $M6 "CHECK (data_version = '' OR data_version ~ '^[0-9a-f]{64}$')," "CHECK (true)," 'SN14 sổ nạp: phiên bản không cần 64 hex'
muon 'TestLoadRecordsRangBuocDB' mm $M6 "CHECK ((data_version = '') = (load_id = ''))" "CHECK (true)" 'SN15 sổ nạp: có phiên bản mà mã lần nạp rỗng (hoặc ngược lại)'
muon 'TestLoadRecords|TestLoadWithVersion' m $L $'`INSERT INTO load_records (tenant_id, load_id, data_version)\n\t\t\tVALUES (current_setting(\'eais.tenant_id\'), $1, $2)`' $'`SELECT current_setting(\'eais.tenant_id\'), $1::text, $2::text`' 'SN16 Load không ghi dòng sổ nạp'
muon 'TestLoadRecordsCungGiaoDichVoiDuLieu' mpy $L '
import sys
s = open(sys.argv[1], encoding="utf-8").read()
head = "\treturn s.inTenant(ctx, tenant, func(tx pgx.Tx) error {\n\t\tif _, err := tx.Exec(ctx, \"SELECT pg_advisory_xact_lock($1)\""
ins = "\t\t_, err := tx.Exec(ctx, `INSERT INTO load_records (tenant_id, load_id, data_version)\n\t\t\tVALUES (current_setting(\x27eais.tenant_id\x27), $1, $2)`, info.LoadID, info.DataVersion)\n\t\treturn err\n\t})\n}"
a = s.index(head)
b = s.index(ins, a) + len(ins)
frm = s[a:b]
to = frm.replace("\treturn s.inTenant(ctx, tenant, func(tx pgx.Tx) error {", "\tif err := s.inTenant(ctx, tenant, func(tx pgx.Tx) error {", 1)
to = to.replace(ins, "\t\treturn nil\n\t}); err != nil {\n\t\treturn err\n\t}\n\treturn s.inTenant(ctx, tenant, func(tx pgx.Tx) error {\n" + ins, 1)
sys.stdout.write(frm + "\0" + to + "\0")
' 'SN17 dòng sổ nạp ghi NGOÀI giao dịch thay dữ liệu (giao dịch riêng, sau commit)'
muon 'TestLoadDongThoiCungTenant|TestLoadRecordsNapChongNhauKhopDuLieu' m $L '"SELECT pg_advisory_xact_lock($1)"' '"SELECT $1::bigint"' 'SN18 Load bỏ khoá tư vấn: dòng sổ không còn theo thứ tự commit'
muon 'TestLoadRecordsGhiVaDocLai' m $L 'ORDER BY id DESC LIMIT 1' 'ORDER BY id ASC LIMIT 1' 'SN19 LatestLoad trả lần nạp CŨ NHẤT'
muon 'TestLoadWithVersionTuChoiPhienBanSai' m $L 'if info.LoadID == "" || !dataVersionFormat.MatchString(info.DataVersion) {' 'if false {' 'SN20 LoadWithVersion nhận phiên bản sai dạng (chạm DB)'
muon 'TestLoadRecordsChuaNapVaNapKhongPhienBan' m $L $'\t\t_, err := tx.Exec(ctx, `INSERT INTO load_records' $'\t\tif info == (LoadInfo{}) {\n\t\t\treturn nil\n\t\t}\n\t\t_, err := tx.Exec(ctx, `INSERT INTO load_records' 'SN21 Load không kèm phiên bản không ghi dòng "không ghi nhận" (phiên bản cũ đọc ra như còn đúng)'
muon 'TestReadySoNap' m platform/pgstore/ready.go '{"load_records", []string{"SELECT", "INSERT"}, nil, nil},' '{"load_records", []string{}, nil, nil},' 'SN22 Ready không đòi quyền trên sổ nạp'
muon 'TestLoadRecordsMoiNhatTheoThuTuCommit' m $L 'ORDER BY id DESC LIMIT 1' 'ORDER BY loaded_at DESC, id DESC LIMIT 1' 'SN36 LatestLoad xếp theo loaded_at (giờ bắt đầu giao dịch) thay vì thứ tự commit (id)'
export PKGS=./cmd/eais/
A=cmd/eais/app.go S=cmd/eais/server.go
muon 'TestDataVersionCLIIngestWhileServing' m $S 'dv, verr := s.app.loadedVersion(ctx, pr.TenantID)' 'dv, verr := s.app.dataVersionNow(), error(nil)' 'SN23 lượt hỏi không đọc sổ nạp (dùng phiên bản app nhớ)'
muon 'TestDataVersionNeverLoaded' m $A 'return dataVersionNone, nil' 'return "0000000000000000000000000000000000000000000000000000000000000000", nil' 'SN24 tenant chưa nạp khai một băm bịa thay vì nói chưa nạp'
muon 'TestDataVersionUnversionedLoadSupersedes' m $A 'case errors.Is(err, pgstore.ErrNotFound), err == nil && rec.DataVersion == "":' 'case errors.Is(err, pgstore.ErrNotFound):' 'SN25 dòng sổ "không ghi nhận" đọc ra phiên bản rỗng thay vì "chưa ghi nhận"'
muon 'TestDataVersionCLIIngestWhileServing' m $S 's.app.askFor(dv).Handle(ctx, pr, req)' 's.app.ask.Load().Handle(ctx, pr, req)' 'SN26 lượt hỏi dùng Ask nền (phiên bản app nhớ) thay vì Ask của lượt'
muon 'TestDataVersion(CLIIngestWhileServing|DirChangedAfterIngest)' m $A 'k.Deps.DataVersion = dv' '_ = dv' 'SN27 askFor không gắn phiên bản của lượt'
muon 'TestDataVersionErrorTurns' m $S 's.app.versions(dv), err)' 's.app.versions(s.app.dataVersionNow()), err)' 'SN28 vết lượt lỗi khai phiên bản app nhớ'
muon 'TestDataVersionCommandTrace' m $S 'dv, verr := s.app.loadedVersion(vctx, pr.TenantID)' 'dv, verr := s.app.dataVersionNow(), error(nil); _ = vctx' 'SN29 lượt lệnh không đọc sổ nạp (dùng phiên bản app nhớ)'
muon 'TestDataVersionErrorTurns' m $A 'rec, err := l.LatestLoad(contracts.WithTenant(ctx, tenant))' 'rec, err := l.LatestLoad(contracts.WithTenant(ctx, contracts.TenantID(a.tenantID())))' 'SN30 đọc sổ nạp theo tenant của app thay vì tenant của lượt (vết tenant khác mang phiên bản của tenant này)'
muon 'TestDataVersionUnreadable' m $S $'\tif verr != nil {\n\t\ts.log.Error("không đọc được phiên bản dữ liệu của lượt",' $'\tif false && verr != nil {\n\t\ts.log.Error("không đọc được phiên bản dữ liệu của lượt",' 'SN31 lượt hỏi vẫn trả lời khi không đọc được phiên bản (mở mặc định)'
muon 'TestDataVersion' m $A 'if l := ledgerOf(a.st); l != nil {' 'if l := ledgerOf(a.st); false && l != nil {' 'SN32 app.ingest không ghi phiên bản vào sổ nạp'
muon 'TestDataVersionDirChangedAfterIngest' m $A 'if ledgerOf(st) != nil {' 'if false && ledgerOf(st) != nil {' 'SN33 newApp lấy phiên bản nền từ băm thư mục (kho có sổ)'
muon 'TestDataVersionStartupLedgerError' m $A 'return nil, fmt.Errorf("không đọc được phiên bản dữ liệu từ sổ nạp của kho: %w", err)' 'dv = ""' 'SN34 khởi động nuốt lỗi đọc sổ nạp'
muon 'TestDataVersionCommandForeignTenant' m $S 'dv, verr := s.app.loadedVersion(vctx, pr.TenantID)' 'dv, verr := s.app.loadedVersion(vctx, contracts.TenantID(s.app.tenantID()))' 'SN35 lượt lệnh đọc sổ nạp theo tenant của app thay vì tenant của lượt (vết principal tenant khác mang phiên bản của tenant này)'

psql -q -X "$EAIS_PG_ADMIN_DSN" -c "DROP DATABASE IF EXISTS $DB_DOT_BIEN WITH (FORCE)"
echo "TỔNG: bị bắt $bat · sống sót $song · khác $khac"
[ "$song" -eq 0 ] && [ "$khac" -eq 0 ]
