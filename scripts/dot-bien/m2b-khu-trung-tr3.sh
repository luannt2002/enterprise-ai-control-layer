#!/usr/bin/env bash
# Danh mục đột biến của gói execution/reports (Phase 0a, khối M2, bước 2 đến 3b và kho PG): kho bản ghi, sổ lần giao, TR1, TR2, TR3, quyết định của người, bản rời nhóm (R6-01, R6-02),
# khoá tuần tự hoá (kể cả khoá bản ghi của R-01), lược đồ PostgreSQL. Đối chiếu HOP-DONG 0a §11.3: DB05, DB07, DB26 đến DB36 và các đột biến của bất biến BB1, BB2, BB3, BB4, BB8; mã M01..M36 và S1..S8 là mã của danh mục này.
# Mỗi đột biến sửa MỘT chỗ ở MỘT tệp (scripts/mutate.sh: chuỗi gốc phải xuất hiện đúng 1 lần, tệp luôn được khôi phục) rồi chạy test của gói. "ĐÚNG CHỖ" nghĩa là: có test đỏ VÀ mỗi mẫu "phải thấy"
# khớp ít nhất một tên test đỏ (mẫu là biểu thức chính quy ERE khớp tên test, kể cả test con). Đỏ mà thiếu một mẫu là "khác"; không test nào đỏ là "SỐNG SÓT"; đỏ sai chỗ là "khác".
#
# Chạy (PG RIÊNG của bạn, hai biến đặt rõ, không có giá trị mặc định; database phải đã tồn tại và TRỐNG hoặc đã qua mọi migration; đừng trỏ vào database đang phục vụ việc khác):
#   EAIS_PG_ADMIN_DSN=postgres://<quản trị>@<máy>:<cổng>/<database riêng>?sslmode=disable \
#   EAIS_PG_DSN=postgres://<vai ứng dụng>@<máy>:<cổng>/<database riêng>?sslmode=disable \
#   taskset -c <nhân> bash scripts/dot-bien/m2b-khu-trung-tr3.sh
# Đột biến lược đồ (S1..S8) cần một database TRỐNG cho MỖI đột biến (migration đã áp thì đột biến không có tác dụng): script tạo và xoá database <database riêng>_mut bằng
#   docker exec ${EAIS_PG_CONTAINER:-eais-pg} psql -U postgres  (container của deploy/docker-compose.yml). Đặt DOT_BIEN_KHONG_LUOC_DO=1 để bỏ qua nhóm S.
# Biến tuỳ chọn: PKGS (mặc định ./execution/reports/), MUTATE_SH (mặc định scripts/mutate.sh), DOT_BIEN_CHI="M01 M05" (chỉ chạy các mã đó).
# Mã thoát 0 chỉ khi không đột biến nào sống sót hay "khác".
set -u
cd "$(dirname "$0")/../.." || exit 1
: "${EAIS_PG_ADMIN_DSN:?đặt EAIS_PG_ADMIN_DSN (PG riêng của bạn)}"
: "${EAIS_PG_DSN:?đặt EAIS_PG_DSN (PG riêng của bạn)}"
export EAIS_PG_ADMIN_DSN EAIS_PG_DSN
export PKGS=${PKGS:-./execution/reports/}
MUTATE=${MUTATE_SH:-scripts/mutate.sh}
CHI=${DOT_BIEN_CHI:-}
CONT=${EAIS_PG_CONTAINER:-eais-pg}

if ! go test $PKGS -count=1 >/dev/null 2>&1; then
	echo "NỀN ĐỎ: go test $PKGS đã đỏ trên cây chưa đột biến — sửa nền rồi mới đo" >&2
	exit 5
fi

bat=0 song=0 khac=0
SEP='=== ĐỘT BIẾN ==='

# mm <mã> <nhãn> <tệp> <mẫu phải thấy>...   (stdin: chuỗi gốc, dòng SEP, chuỗi đột biến; chuỗi đột biến có thể rỗng)
mm() {
	local ma=$1 nhan=$2 tep=$3 body goc dot
	shift 3
	if [ -n "$CHI" ] && ! grep -qw -- "$ma" <<<"$CHI"; then cat >/dev/null; return 0; fi
	body=$(cat)
	goc=${body%%$'\n'"$SEP"*}
	dot=${body#*$'\n'"$SEP"}
	dot=${dot#$'\n'}
	local muon out rc thieu="" p ten
	muon=$(IFS='|'; printf '%s' "$*")
	out=$(MUTATE_TEST_MUON="$muon" "$MUTATE" "$tep" "$goc" "$dot" "$ma $nhan"); rc=$?
	printf '%s\nrc=%s\n' "$out" "$rc"
	case $rc in
	0)
		ten=$(sed -n 's/^.*BỊ BẮT: //p' <<<"$out" | tr ' ' '\n')
		for p in "$@"; do
			grep -E -q -- "$p" <<<"$ten" || thieu="$thieu [$p]"
		done
		if [ -z "$thieu" ]; then
			echo "[$ma] ĐÚNG CHỖ"
			bat=$((bat + 1))
		else
			echo "[$ma] BỊ BẮT NHƯNG THIẾU MẪU:$thieu"
			khac=$((khac + 1))
		fi
		;;
	3) echo "[$ma] SỐNG SÓT"; song=$((song + 1)) ;;
	*) echo "[$ma] KHÁC (rc=$rc)"; khac=$((khac + 1)) ;;
	esac
}

R=execution/reports
# ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------
# TR3: nghi trùng, thứ tự chuẩn, đánh giá lại (DB07, DB27, DB30, DB31, DB34)
mm M01 "DB07 bỏ TR3 (nghi trùng)" $R/tr3.go 'TestLC24' 'TestLC25' 'TestR601_PhepKiemTinhChat' <<'EOF'
				if p != nil && r.SentAt.Sub(p.SentAt) <= w && !r.reviewedWith(p.Key) {
=== ĐỘT BIẾN ===
				if false && p != nil && r.SentAt.Sub(p.SentAt) <= w && !r.reviewedWith(p.Key) {
EOF
mm M34 "DB06 TR3 bỏ cửa sổ W: gộp oan hai tin cùng giá trị cách xa nhau" $R/tr3.go 'TestLC26' 'TestLC25' <<'EOF'
				if p != nil && r.SentAt.Sub(p.SentAt) <= w && !r.reviewedWith(p.Key) {
=== ĐỘT BIẾN ===
				if p != nil && !r.reviewedWith(p.Key) {
EOF
mm M02 "DB27 không đánh giá lại bản đã có khi bản gửi sớm hơn đến muộn" $R/tr3.go 'TestLC75_BanGuiSomDenMuon' 'TestLC63' <<'EOF'
				if p != nil && r.SentAt.Sub(p.SentAt) <= w && !r.reviewedWith(p.Key) {
=== ĐỘT BIẾN ===
				if p != nil && c.isNew[r.Key] && r.SentAt.Sub(p.SentAt) <= w && !r.reviewedWith(p.Key) {
EOF
mm M03 "DB27 nhóm không xếp theo sent_at" $R/tr3.go 'TestLC24' 'TestLC25' <<'EOF'
		sort.Slice(list, func(i, j int) bool { return before(list[i], list[j]) })
=== ĐỘT BIẾN ===
		sort.Slice(list, func(i, j int) bool { return list[i].Key > list[j].Key })
EOF
mm M04 "DB30 bản đã có quyết định của người miễn đánh giá lại" $R/tr3.go 'TestLC78' 'TestLC77' <<'EOF'
				if p != nil && r.SentAt.Sub(p.SentAt) <= w && !r.reviewedWith(p.Key) {
=== ĐỘT BIẾN ===
				if p != nil && r.Decision == "" && r.SentAt.Sub(p.SentAt) <= w && !r.reviewedWith(p.Key) {
EOF
mm M05 "DB31 đánh giá lại nới: bản HELD tự thành ACCEPTED khi bản đứng trước rời nhóm" $R/tr3.go 'TestLC81_i_BanRoiNhom' 'TestLC84_i_ii_iii' 'TestR601_PhepKiemTinhChat' <<'EOF'
		r.DuplicateOf = ""
=== ĐỘT BIẾN ===
		r.DuplicateOf = ""
		r.Disposition, r.Reason = DispAccepted, ""
EOF
mm M06 "DB34 cặp đã xét với bản vừa rời nhóm phủ cả cặp mới (bản đứng sau bản rời nhóm giữ ACCEPTED)" $R/tr3.go 'TestLC81_iv' 'TestR601_PhepKiemTinhChat' <<'EOF'
!r.reviewedWith(p.Key) {
=== ĐỘT BIẾN ===
len(r.ReviewedPairs) == 0 {
EOF
# ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------
# R6-01, R6-02, DB36: bản bị bác theo bản được giữ rời nhóm sau quyết định
mm M07 "DB36 bản REJECTED bởi MARK_DUPLICATE không quay lại HELD" $R/tr3.go 'TestLC84_i_ii_iii' 'TestR601_ApplyDinhChinh' 'TestR602' 'TestR601_PhepKiemTinhChat' <<'EOF'
		if keptCounts(b, kept) {
=== ĐỘT BIẾN ===
		if true || keptCounts(b, kept) {
EOF
mm M08 "DB36 quay lại cả khi bản được giữ rời nhóm bằng chính MARK_DUPLICATE" $R/tr3.go 'TestLC84_iv' 'TestR602' <<'EOF'
	if k.Disposition == DispRejected && k.Decision == DecMarkDuplicate {
=== ĐỘT BIẾN ===
	if false && k.Disposition == DispRejected && k.Decision == DecMarkDuplicate {
EOF
mm M09 "DB36 bản được giữ thôi được tính vì lý do khác (thu hồi, thay thế, HELD vì lý do khác) bị bỏ sót" $R/tr3.go 'TestLC84_i_ii_iii' 'TestR601_ApplyDinhChinh' 'TestR602' <<'EOF'
	return inGroup(k) && groupOf(k) == groupOf(b)
=== ĐỘT BIẾN ===
	return groupOf(k) == groupOf(b)
EOF
# ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------
# Quyết định của người (DB29, DB35, DB19, bảng §6.2, audit)
mm M10 "DB29 quyết định cho ra ACCEPTED miễn TR3 (không chạy settle)" $R/decide.go 'TestLC77' <<'EOF'
	if err := c.settle(); err != nil {
		return Record{}, err
	}
	return r.clone(), nil
=== ĐỘT BIẾN ===
	if cmd.Decision == DecKeepBoth || cmd.Decision == DecMarkDuplicate || cmd.Decision == DecReject {
		if err := c.settle(); err != nil {
			return Record{}, err
		}
	}
	return r.clone(), nil
EOF
mm M11 "DB35 bỏ kiểm cặp đã thấy (màn hình cũ)" $R/decide.go 'TestLC83_i' 'TestLC83_ii' <<'EOF'
	if seen != r.DuplicateOf {
=== ĐỘT BIẾN ===
	if false && seen != r.DuplicateOf {
EOF
mm M12 "DB35 nhận MARK_DUPLICATE khi duplicate_of rỗng" $R/decide.go 'TestLC83_i' <<'EOF'
	if cmd.Decision == DecMarkDuplicate && r.DuplicateOf == "" {
=== ĐỘT BIẾN ===
	if false && cmd.Decision == DecMarkDuplicate && r.DuplicateOf == "" {
EOF
mm M13 "R4-03 lệnh mang seen_duplicate_of rỗng bị coi như thiếu" $R/decide.go 'TestLC83_i' <<'EOF'
	if cmd.SeenDuplicateOf == nil {
=== ĐỘT BIẾN ===
	if cmd.SeenDuplicateOf == nil || *cmd.SeenDuplicateOf == "" {
EOF
mm M14 "DB19 APPLY không đặt bản gốc SUPERSEDED" $R/decide.go 'TestLC15' 'TestR601_ApplyDinhChinh' 'TestApply_ChayTR3' <<'EOF'
	t.Disposition, t.SupersededBy = DispSuperseded, r.Key
=== ĐỘT BIẾN ===
	t.SupersededBy = r.Key
EOF
mm M15 "quyết định không cần người quyết (audit mất 'ai')" $R/decide.go 'TestQuyetDinh_BangSuThat' <<'EOF'
	case actor == "":
=== ĐỘT BIẾN ===
	case false && actor == "":
EOF
mm M16 "bảng §6.2: CHOOSE cho ambiguous_entity cũng nhận KEEP_BOTH" $R/decide.go 'TestQuyetDinh_BangSuThat' <<'EOF'
			return d == DecChoose || d == DecReject
=== ĐỘT BIẾN ===
			return d == DecChoose || d == DecReject || d == DecKeepBoth
EOF
mm M17 "cặp đã xét của các bản KHÁC không bị xoá khi nội dung của bản kia đổi" $R/decide.go 'TestAmend_BanKia' <<'EOF'
		if q.Key == key || !q.reviewedWith(key) {
=== ĐỘT BIẾN ===
		if true || q.Key == key || !q.reviewedWith(key) {
EOF
# ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------
# Sổ lần giao, TR1, TR2, bảo toàn, tổng, tenant (DB01, DB05, DB28, DB33, BB1, BB2, BB3, BB8)
mm M18 "DB28 lần giao DUPLICATE không ghi dòng sổ (trùng rơi im lặng)" $R/engine.go 'TestLC01' 'TestLC21' 'TestLC23' <<'EOF'
		return c.logDelivery(c.duplicateEntry(batchID, ordinal, orig, it, hit)), nil
=== ĐỘT BIẾN ===
		return c.duplicateEntry(batchID, ordinal, orig, it, hit), nil
EOF
mm M19 "DB05 tắt TR1: cùng khoá, cùng nội dung bị coi là id_conflict" $R/tr12.go 'TestLC21' 'TestLC23' <<'EOF'
			return &dupHit{reason: ReasonDuplicateKey, ref: base.Key}, nil
=== ĐỘT BIẾN ===
			return c.idConflict(it, base.Key, hash)
EOF
mm M20 "DB33 vân tay đơn vị không có sent_at bỏ mã nguồn trong tệp (gộp oan hai dòng cùng chữ)" $R/normalize.go 'TestLC79' <<'EOF'
	return hexSum("tr2b", tenant, it.SourceType, it.OccurredOn, it.SourceSeq, text)
=== ĐỘT BIẾN ===
	return hexSum("tr2b", tenant, it.SourceType, it.OccurredOn, text)
EOF
mm M21 "DB33 vân tay đơn vị không có sent_at bỏ ngày xảy ra (gộp oan hai ngày)" $R/normalize.go 'TestLC79' <<'EOF'
	return hexSum("tr2b", tenant, it.SourceType, it.OccurredOn, it.SourceSeq, text)
=== ĐỘT BIẾN ===
	return hexSum("tr2b", tenant, it.SourceType, it.SourceSeq, text)
EOF
mm M22 "vân tay tin bỏ giờ gửi (gộp oan hai lần gửi cách nhau vài phút)" $R/normalize.go 'TestLC24' <<'EOF'
		return hexSum("tr2a", tenant, it.Channel, it.SenderID, it.SentAt.UTC().Truncate(time.Second).Format(time.RFC3339), text)
=== ĐỘT BIẾN ===
		return hexSum("tr2a", tenant, it.Channel, it.SenderID, time.Time{}.String(), text)
EOF
mm M23 "BB2 HELD không có người nhận" $R/engine.go 'TestLC02_LC03_LC06' 'TestNguoiNhan' <<'EOF'
		r.Recipient = c.recipientFor(r.Entity)
=== ĐỘT BIẾN ===
		r.Recipient = ""
EOF
mm M24 "BB3 tổng đọc lấy cả bản HELD, QUARANTINED" $R/engine.go 'TestBB3' 'TestLC24' <<'EOF'
		if r.Disposition != DispAccepted || r.Kind != KindOccurred || r.Entity == "" || r.OccurredOn == "" || r.Value.Number == "" {
=== ĐỘT BIẾN ===
		if r.Kind != KindOccurred || r.Entity == "" || r.OccurredOn == "" || r.Value.Number == "" {
EOF
mm M25 "BB8 hai tenant dùng chung dữ liệu (kho bộ nhớ)" $R/memory.go 'TestBB8_HaiTenant' 'TestLC28' <<'EOF'
	t := m.data[id]
=== ĐỘT BIẾN ===
	id = "chung"
	t := m.data[id]
EOF
mm M26 "BB1 bộ kiểm bảo toàn bỏ qua dòng sổ sai thứ tự hoặc sai khoá" $R/store.go 'TestBB1_Conserved' <<'EOF'
		if e.Ordinal != i+1 || e.Key != manifest[i] {
=== ĐỘT BIẾN ===
		if false && (e.Ordinal != i+1 || e.Key != manifest[i]) {
EOF
mm M33 "cặp đã xét của chính bản vừa đổi nhóm không bị xoá" $R/decide.go 'TestAmend_BanVuaDoiNhom' <<'EOF'
	if !moved {
=== ĐỘT BIẾN ===
	if true {
EOF
# ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------
# Khoá tuần tự hoá (DB26) và kho PG
# Ghi chú: bỏ riêng lời gọi ensureLock trong slotRecs là đột biến TƯƠNG ĐƯƠNG (đã thử, sống sót): khoá của ô luôn đã được giữ sớm hơn (deliver, getLocked, đầu settle); lời gọi ở slotRecs là lớp phòng thủ thừa.
mm M27 "DB26 không bao giờ lấy khoá (kiểm xác định, không cần PG: đọc nhóm không đứng sau khoá nào)" $R/engine.go 'TestKhoaTuanTu_MoiLanDocNhom' <<'EOF'
	if err := c.tx.lock(k); err != nil {
=== ĐỘT BIẾN ===
	if err := error(nil); err != nil {
EOF
mm M28 "lấy khoá không theo thứ tự tăng dần (nguy cơ chết khoá)" $R/engine.go 'TestKhoaTuanTu_ApplyBanGoc' <<'EOF'
	if c.hasLock && k < c.maxLock {
=== ĐỘT BIẾN ===
	if false && c.hasLock && k < c.maxLock {
EOF
mm M29 "DB26 khoá tư vấn của PG thành không làm gì (đua thật giữa hai giao dịch)" $R/pg.go 'TestDB26_.*/pg' 'TestLC74_.*/pg' <<'EOF'
	_, err := x.tx.Exec(x.ctx, "SELECT pg_advisory_xact_lock($1)", k)
=== ĐỘT BIẾN ===
	_, err := x.tx.Exec(x.ctx, "SELECT $1::bigint", k)
EOF
mm M30 "va chạm khoá duy nhất bị nuốt: đơn vị thua đua biến mất" $R/pg.go 'TestLC65_.*/pg' <<'EOF'
	if tag.RowsAffected() == 0 {
		return errKeyRace
=== ĐỘT BIẾN ===
	if false && tag.RowsAffected() == 0 {
		return errKeyRace
EOF
mm M31 "giao dịch lỗi vẫn COMMIT (nửa ghi)" $R/pg.go 'TestPG_LC67' <<'EOF'
		if commit {
=== ĐỘT BIẾN ===
		if commit || err != nil {
EOF
mm M32 "BB4 nguyên văn bị chuẩn hoá trước khi lưu (PG)" $R/pg.go 'TestLC08_.*/pg' <<'EOF'
	raw := []byte(r.Raw)
=== ĐỘT BIẾN ===
	raw := []byte(NormalizeText(r.Raw))
EOF
# R-01 (chết khoá 40P01): hai đột biến đã đo ở commit sửa (1b4c1da) trên 14.700 thao tác trộn của TestPG_R01_.
mm M35 "R-01 bỏ khoá tư vấn theo bản ghi ở getLocked (bản không có ô và bản vừa đổi ô hết được che: 23505)" $R/engine.go 'TestPG_R01_' <<'EOF'
	if err := c.ensureLock(lockRef{rec: key}); err != nil {
=== ĐỘT BIẾN ===
	if err := error(nil); err != nil {
EOF
mm M36 "R-01 đặt lại FOR UPDATE ở getRecord (khoá hàng nằm ngoài thứ tự tăng dần của khoá tư vấn: 40P01)" $R/pg.go 'TestPG_R01_' <<'EOF'
FROM report_records WHERE unit_key = $1`, key)
=== ĐỘT BIẾN ===
FROM report_records WHERE unit_key = $1 FOR UPDATE`, key)
EOF

# ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------
# Lược đồ PostgreSQL (nhóm S): mỗi đột biến chạy trên database TRỐNG dựng lại, vì migration đã áp thì đột biến không có tác dụng.
if [ -z "${DOT_BIEN_KHONG_LUOC_DO:-}" ]; then
	sdb_of() { sed -E 's#^(.*://[^/]*/)([^?]*)(\?.*)?$#\2#' <<<"$1"; }
	with_db() { sed -E "s#^(.*://[^/]*/)([^?]*)(\\?.*)?\$#\\1$2\\3#" <<<"$1"; }
	BASE_DB=$(sdb_of "$EAIS_PG_ADMIN_DSN")
	MUT_DB=${BASE_DB}_mut
	psql_adm() { docker exec "$CONT" psql -U postgres -d postgres -v ON_ERROR_STOP=1 -q -c "$1" >/dev/null; }
	fresh_db() { psql_adm "DROP DATABASE IF EXISTS $MUT_DB WITH (FORCE)" && psql_adm "CREATE DATABASE $MUT_DB"; }
	trap 'psql_adm "DROP DATABASE IF EXISTS $MUT_DB WITH (FORCE)" 2>/dev/null' EXIT
	export EAIS_PG_ADMIN_DSN=$(with_db "$EAIS_PG_ADMIN_DSN" "$MUT_DB") EAIS_PG_DSN=$(with_db "$EAIS_PG_DSN" "$MUT_DB")
	if [ -z "$CHI" ] || grep -qE '\bS[0-9]' <<<"$CHI"; then
		fresh_db && "$MUTATE" --nen >/dev/null # nền trên database mới (áp migration GỐC); các đột biến dưới dùng lại kết quả nền
	fi
	export MUTATE_NEN_CO_SAN=1
	ms() { fresh_db; mm "$@"; }
	MG=db/migrations/0009_kho_bao_cao_ngay.sql
	ms S1 "RLS không FORCE" $MG 'TestPGLuocDo_RLS' 'TestPGMigration_' <<'EOF'
    EXECUTE format('ALTER TABLE %I FORCE ROW LEVEL SECURITY', t);
=== ĐỘT BIẾN ===
    NULL;
EOF
	ms S2 "eais_app có DELETE" $MG 'TestPGLuocDo_RLS' <<'EOF'
    EXECUTE format('GRANT SELECT, INSERT ON %I TO eais_app', t);
=== ĐỘT BIẾN ===
    EXECUTE format('GRANT SELECT, INSERT, DELETE ON %I TO eais_app', t);
EOF
	ms S3 "eais_app có UPDATE ở sổ lần giao và các bảng chỉ-thêm" $MG 'TestPGLuocDo_RLS' 'TestPGRLS_HaiTenant' <<'EOF'
    EXECUTE format('GRANT SELECT, INSERT ON %I TO eais_app', t);
=== ĐỘT BIẾN ===
    EXECUTE format('GRANT SELECT, INSERT, UPDATE ON %I TO eais_app', t);
EOF
	ms S4 "eais_app sửa được cột khoá unit_key" $MG 'TestPGLuocDo_RLS' 'TestPGRLS_HaiTenant' <<'EOF'
GRANT UPDATE (channel, source_type,
=== ĐỘT BIẾN ===
GRANT UPDATE (unit_key, channel, source_type,
EOF
	ms S5 "chính sách tenant không lọc dòng đọc (rò sang tenant khác)" $MG 'TestPGRLS_HaiTenant' <<'EOF'
      USING (tenant_id = current_setting('eais.tenant_id', true))
=== ĐỘT BIẾN ===
      USING (true)
EOF
	ms S6 "bỏ CHECK HELD và QUARANTINED phải có chi tiết và người nhận" $MG 'TestPGRangBuoc' <<'EOF'
  CHECK (disposition NOT IN ('HELD', 'QUARANTINED') OR (detail <> '' AND recipient <> '')),                       -- BB2, CL2, CL5
=== ĐỘT BIẾN ===
EOF
	ms S7 "bỏ khoá ngoại sổ lần giao → bản ghi (dòng DUPLICATE trỏ bản không có thật)" $MG 'TestPGRangBuoc' <<'EOF'
  FOREIGN KEY (tenant_id, record_ref) REFERENCES report_records (tenant_id, unit_key),
=== ĐỘT BIẾN ===
EOF
	ms S8 "bỏ CHECK tenant_id rỗng ở bản ghi" $MG 'TestPGRangBuoc' <<'EOF'
  CONSTRAINT report_records_tenant_khong_rong CHECK (tenant_id <> ''),
=== ĐỘT BIẾN ===
EOF
fi

echo "TỔNG: $bat đúng chỗ · $song sống sót · $khac khác"
[ "$song" -eq 0 ] && [ "$khac" -eq 0 ]
