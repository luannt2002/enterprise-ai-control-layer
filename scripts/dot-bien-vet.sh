#!/usr/bin/env bash
# Danh mục đột biến của kho vết bền (làn 0, đơn vị U0.1): migration 0003, platform/pgstore/vet.go, observability/traces.
# Mỗi dòng: m <tệp> <chuỗi gốc> <chuỗi đột biến> <nhãn>. Kỳ vọng: mọi dòng in "BỊ BẮT" (rc=0); cuối cùng in tổng và thoát 0 chỉ khi
# không đột biến nào sống sót. Mã đổi thì chuỗi gốc có thể không còn khớp (mutate.sh báo "xuất hiện 0 lần", rc=2): sửa chuỗi theo mã
# mới rồi chạy lại — đừng xoá dòng cho xanh.
#
# CẦN PG RIÊNG và hai biến môi trường đặt rõ (không có giá trị mặc định, để không vô tình chạy vào PG đang phục vụ việc khác):
#   EAIS_PG_ADMIN_DSN=postgres://postgres@127.0.0.1:<cổng>/eais?sslmode=disable
#   EAIS_PG_DSN=postgres://eais_app@127.0.0.1:<cổng>/eais?sslmode=disable
# Script đụng tới cụm PG đó ở BA chỗ — đừng trỏ vào PG đang phục vụ việc khác:
#   1. tạo rồi xoá database $DB_DOT_BIEN (mặc định eais_dot_bien_vet) cho các đột biến migration (MV*), vì migration chỉ áp MỘT lần mỗi
#      database; DB_DOT_BIEN KHÔNG được trùng tên database trong hai DSN (script từ chối, kẻo DROP chính database đang dùng);
#   2. các đột biến mã Go (SV*, TV*) chạy `go test` trên database trong EAIS_PG_DSN: test GHI dòng vào các bảng vết (tenant ngẫu nhiên,
#      TestMain dọn bằng vai quản trị khi xong), tạo rồi gỡ database eais_vet_* và vai eais_test_vet_* (BYPASSRLS);
#   3. cần `psql` trên máy để dựng / xoá database đột biến (script kiểm trước khi chạy).
cd "$(dirname "$0")/.." || exit 1
: "${EAIS_PG_ADMIN_DSN:?đặt EAIS_PG_ADMIN_DSN (PG riêng của bạn)}"
: "${EAIS_PG_DSN:?đặt EAIS_PG_DSN (PG riêng của bạn)}"
export EAIS_PG_ADMIN_DSN EAIS_PG_DSN
DB_DOT_BIEN=${DB_DOT_BIEN:-eais_dot_bien_vet}
command -v psql >/dev/null || { echo "thiếu psql: cần để dựng / xoá database đột biến" >&2; exit 1; }
# db_of <dsn>: tên database trong DSN dạng postgres://vai@máy:cổng/<database>[?tham số].
db_of() { printf '%s' "$1" | sed -E 's#^[a-z]+://[^/]*/([^/?]*).*#\1#'; }
for d in "$EAIS_PG_ADMIN_DSN" "$EAIS_PG_DSN"; do
	if [ "$(db_of "$d")" = "$DB_DOT_BIEN" ]; then
		echo "DB_DOT_BIEN=$DB_DOT_BIEN trùng tên database trong DSN: script sẽ DROP chính database đó. Đặt DB_DOT_BIEN khác." >&2
		exit 1
	fi
done
bat=0 song=0 khac=0
m() {
	local res rc
	res=$(scripts/mutate.sh "$1" "$2" "$3" "$4"); rc=$?
	printf '%s\nrc=%s\n' "$res" "$rc"
	case $rc in 0) bat=$((bat + 1)) ;; 3) song=$((song + 1)) ;; *) khac=$((khac + 1)) ;; esac
}
fresh() { psql -q -X "$EAIS_PG_ADMIN_DSN" -c "DROP DATABASE IF EXISTS $DB_DOT_BIEN WITH (FORCE)" -c "CREATE DATABASE $DB_DOT_BIEN"; }
# dsn_db <dsn> <database>: thay tên database trong DSN dạng postgres://vai@máy:cổng/<database>[?tham số].
dsn_db() { printf '%s' "$1" | sed -E 's#^([a-z]+://[^/]*)/[^/?]*#\1/'"$2"'#'; }
mm() {
	fresh || { echo "[$4] không dựng được database $DB_DOT_BIEN"; echo "rc=5"; khac=$((khac + 1)); return; }
	local a d
	a=$(dsn_db "$EAIS_PG_ADMIN_DSN" "$DB_DOT_BIEN") d=$(dsn_db "$EAIS_PG_DSN" "$DB_DOT_BIEN")
	EAIS_PG_ADMIN_DSN=$a EAIS_PG_DSN=$d m "$@"
}

export PKGS=./platform/pgstore/
M3=db/migrations/0003_kho_vet_ben.sql
mm $M3 "USING (tenant_id = current_setting('eais.tenant_id', true))" "USING (true)" 'MV1 chính sách đọc mở toang'
mm $M3 "WITH CHECK (tenant_id = current_setting('eais.tenant_id', true))" "WITH CHECK (true)" 'MV2 chính sách ghi mở toang'
mm $M3 "EXECUTE format('ALTER TABLE %I ENABLE ROW LEVEL SECURITY', t);" "NULL;" 'MV3 không bật RLS'
mm $M3 "EXECUTE format('ALTER TABLE %I FORCE ROW LEVEL SECURITY', t);" "NULL;" 'MV4 không FORCE RLS'
mm $M3 "EXECUTE format('GRANT SELECT, INSERT ON %I TO eais_app', t);" "EXECUTE format('GRANT SELECT, INSERT, UPDATE ON %I TO eais_app', t);" 'MV5 cấp thêm UPDATE'
mm $M3 "EXECUTE format('GRANT SELECT, INSERT ON %I TO eais_app', t);" "EXECUTE format('GRANT SELECT, INSERT, DELETE ON %I TO eais_app', t);" 'MV6 cấp thêm DELETE'
mm $M3 "EXECUTE format('GRANT SELECT, INSERT ON %I TO eais_app', t);" "EXECUTE format('GRANT SELECT, INSERT, TRUNCATE ON %I TO eais_app', t);" 'MV7 cấp thêm TRUNCATE (không chịu RLS)'
mm $M3 "EXECUTE format('GRANT SELECT, INSERT ON %I TO eais_app', t);" "EXECUTE format('GRANT SELECT, INSERT ON %I TO eais_app WITH GRANT OPTION', t);" 'MV8 cấp kèm GRANT OPTION'
mm $M3 "EXECUTE format('GRANT SELECT, INSERT ON %I TO eais_app', t);" "EXECUTE format('GRANT SELECT, INSERT ON %I TO eais_app, PUBLIC', t);" 'MV9 cấp qua PUBLIC'
mm $M3 "EXECUTE format('REVOKE ALL ON %I FROM eais_app', t);" "NULL;" 'MV10 chạy lại không gỡ quyền thừa của eais_app'
mm $M3 "EXECUTE format('REVOKE ALL ON %I FROM PUBLIC', t);" "NULL;" 'MV11 chạy lại không gỡ quyền của PUBLIC'
mm $M3 "EXECUTE format('DROP POLICY IF EXISTS tenant_isolation ON %I', t);" "NULL;" 'MV12 chạy lại không dựng lại được chính sách'
mm $M3 "CHECK (load_id <> '' AND record_key <> '' AND reason <> '')" "CHECK (load_id <> '' AND record_key <> '')" 'MV13 cách ly không cần lý do'
mm $M3 "FOREIGN KEY (tenant_id, trace_id) REFERENCES decision_records (tenant_id, trace_id)," "CHECK (true)," 'MV14 usage không cần lượt có vết'
mm $M3 "PRIMARY KEY (tenant_id, load_id, record_key, reason)," "PRIMARY KEY (load_id, record_key, reason)," 'MV15 khoá cách ly không có tenant'
mm $M3 "PRIMARY KEY (tenant_id, trace_id, model)," "PRIMARY KEY (trace_id, model)," 'MV16 khoá usage không có tenant'
mm $M3 "{64}" "{1,}" 'MV17 băm không cần 64 hex'
mm $M3 "jsonb_typeof(steps) = 'array'" "true" 'MV18 steps không cần là mảng'
mm $M3 "CHECK (trace_id <> '')," "CHECK (true)," 'MV19 trace_id rỗng được phép'
mm $M3 "CHECK (calls >= 0 AND input_tokens >= 0 AND output_tokens >= 0 AND cache_read >= 0 AND cache_write >= 0)" "CHECK (true)" 'MV20 usage số âm được phép'

V=platform/pgstore/vet.go
m $V 'if rec.Decision.Tenant != string(tenant) {' 'if false && rec.Decision.Tenant != string(tenant) {' 'SV1 bản ghi mang tenant khác vẫn được ghi'
m $V 'if stored != digest {' 'if false && stored != digest {' 'SV2 cùng trace_id khác nội dung bị bỏ qua im lặng'
m $V 'ON CONFLICT (tenant_id, trace_id) DO NOTHING' 'ON CONFLICT (tenant_id, trace_id) DO UPDATE SET content_sha256 = EXCLUDED.content_sha256' 'SV3 ghi đôi báo là ghi mới'
m $V 'clean, err := traces.SanitizeRecord(rec)' 'clean, err := rec, error(nil)' 'SV4 bản ghi không qua làm sạch'
m $V 'if bad > 0 {' 'if false && bad > 0 {' 'SV5 cách ly khác nội dung bị bỏ qua im lặng'
m $V 'if got.Calls != u.Calls || got.Tokens != t {' 'if false {' 'SV6 usage khác số bị bỏ qua im lặng'
m $V 'if r.Key == "" || r.Reason == "" {' 'if r.Key == "" {' 'SV7 cách ly không lý do qua được tầng Go'
m $V 'append(details, traces.Sanitize(r.Detail))' 'append(details, r.Detail)' 'SV8 chi tiết cách ly không làm sạch'
m $V 'pgErr.Code == pgFKViolation {' 'pgErr.Code == "00000" {' 'SV9 usage mồ côi bị gán nhãn hạ tầng'
m $V 'if errors.Is(err, pgx.ErrNoRows) {' 'if false {' 'SV10 không thấy bản ghi không thành ErrNotFound'
m $V 'if err := rec.Validate(); err != nil {' 'if err := rec.Validate(); false && err != nil {' 'SV11 bản ghi hỏng không bị từ chối ở tầng Go'
m $V 'if clean.Decision.TraceID != rec.Decision.TraceID {' 'if false && clean.Decision.TraceID != rec.Decision.TraceID {' 'SV12 trace_id bị làm sạch đổi vẫn được ghi (lưu khác mã nơi gọi cấp)'
m $V 'func stable(id string) bool { return traces.Sanitize(id) == id }' 'func stable(id string) bool { return true }' 'SV13 mã tra cứu bị làm sạch đổi không bị chặn ở bất kỳ hàm nào'
m $V 'if loadID == "" || !stable(loadID) {' 'if loadID == "" {' 'SV14 mã lần nạp bị làm sạch đổi vẫn được ghi'
m $V 'if !stable(u.TraceID) {' 'if false && !stable(u.TraceID) {' 'SV15 usage với trace_id bị làm sạch đổi vẫn chạm DB'

export PKGS=./observability/traces/
S=observability/traces/sanitize.go R=observability/traces/record.go T=observability/traces/step.go
m $S 's = m.re.ReplaceAllString(s, m.repl)' '_ = m' 'TV1 Sanitize không che gì'
m $S 'case reflect.Slice, reflect.Array:' 'case reflect.Array:' 'TV2 làm sạch bỏ sót phần tử slice'
m $S 'if Sanitize(key+"="+val) != key+"="+val && clean == val {' 'if false {' 'TV3 không che giá trị map theo khoá'
m $S 's = strings.ReplaceAll(s, "\x00", "")' '_ = s' 'TV4 không bỏ byte NUL'
m $S 'len(r) > maxErrorRunes {' 'false && len(r) > maxErrorRunes {' 'TV5 thông điệp lỗi không bị cắt'
m $S 'fault = contracts.FaultCode' 'fault = ""' 'TV6 lỗi chưa gắn nhãn thành lượt không lỗi'
m $R 'if a.Fault != b.Fault {' 'if false && a.Fault != b.Fault {' 'TV7 so lượt lỗi bỏ qua nhãn lỗi'
m $R '{"normalize", StageNormalize}, {"resolve", StageResolve}, {"intent", StageIntent},' '{"resolve", StageResolve}, {"normalize", StageNormalize}, {"intent", StageIntent},' 'TV8 thứ tự chặng bị đảo'
m $R '{"verdict", StageValidate},' '{"outcome", StageExecute}, {"verdict", StageValidate},' 'TV9 outcome xét trước verdict (lỗi R1-01 nguyên bản)'
m $R '{"verdict", StageValidate},' '{"evidence", StageContext}, {"verdict", StageValidate},' 'TV17 bằng chứng xét trước verdict'
m $R '{"verdict", StageValidate},' '{"answer", StageGenerate}, {"verdict", StageValidate},' 'TV18 câu xét trước verdict (khác cả câu lẫn verdict tính là S8)'
m $R 'if s.TraceID != "" && s.TraceID != r.Decision.TraceID {' 'if false {' 'TV10 bước mang trace_id lượt khác qua được'
m $T 'executed := has("execute") || has("retrieve")' 'executed := true' 'TV11 bước không chạy vẫn tính là chạy'
m $T 's.Ms, s.Timed = s.Ms+ms, true' 's.Ms, s.Timed = ms, true' 'TV12 span cùng tên không cộng dồn'
m $T 's.Reason = notRecorded("luật sinh từng đoạn (Span.Rule) chưa xuất khỏi Handle")' 's.Reason = "luật mặc định của chuẩn hoá"' 'TV13 bịa lý do thay vì ghi chưa ghi'
m $T 'if id != c.ChosenID {' 'if true {' 'TV14 bằng chứng được chọn bị kể là bị loại'
m $T 'n <= 10 &&' 'n <= 11 &&' 'TV15 chấp nhận mã bước S11'
m $T 'case route.NeedsModel() && has("generate"):' 'case !route.NeedsModel() && has("generate"):' 'TV16 đảo route cần model / không cần model'
m $T 'case route.NeedsModel() && has("generate"):' 'case route.NeedsModel():' 'TV19 S8 kể model diễn đạt dù không có span generate (R1-02)'
m $T 'case route.NeedsModel():' 'case !route.NeedsModel():' 'TV20 route cần model mà không gọi model rơi vào nhánh mẫu câu'
m $T 'if d.Outcome == string(contracts.OutcomeInsufficientEvidence) {' 'if false && d.Outcome == string(contracts.OutcomeInsufficientEvidence) {' 'TV21 truy vấn không đủ dữ liệu mà S6 không lệch (R1-03)'
m $T 'if route == contracts.RouteHybrid && has("retrieve") && !has("generate") {' 'if false && route == contracts.RouteHybrid && has("retrieve") && !has("generate") {' 'TV22 HYBRID phần tri thức rỗng mà S6 không nói'
m $T 'StageExecute: {"execute", "retrieve"}, StageGenerate: {"generate"}, StageValidate: {"validate"},' 'StageExecute: {"execute"}, StageGenerate: {"generate"}, StageValidate: {"validate"},' 'TV23 span retrieve không gắn vào bước nào (R1-05)'
m $T 'StageValidate: {"validate"},' 'StageValidate: {"validate", "generate"},' 'TV24 span generate gắn vào hai bước (thời gian đếm đôi)'
RX_Q=$'(?:(")[^"]*"?|(\')[^\']*\'?|[^\\s"\',;&]+)'
m $S "$RX_Q" $'(?:(")[^\\s"\',;&]+|(\')[^\\s"\',;&]+|[^\\s"\',;&]+)' 'TV25 giá trị trong nháy dừng ở khoảng trắng (lọt đuôi, R1-04)'
m $S "$RX_Q" $'(?:(")[^"]*"|(\')[^\']*\'|[^\\s"\',;&]+)' 'TV27 nháy không đóng không được che'
m $S '(?:set-)?cookies?' '(?:set-)?cookyes?' 'TV26 không che tiêu đề Cookie / Set-Cookie'

fresh >/dev/null 2>&1; psql -q -X "$EAIS_PG_ADMIN_DSN" -c "DROP DATABASE IF EXISTS $DB_DOT_BIEN WITH (FORCE)"
echo "TỔNG: bị bắt $bat · sống sót $song · khác $khac"
[ "$song" -eq 0 ] && [ "$khac" -eq 0 ]
