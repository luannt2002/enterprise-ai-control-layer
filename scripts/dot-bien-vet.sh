#!/usr/bin/env bash
# Danh mục đột biến của kho vết bền (làn 0, đơn vị U0.1): migration 0003, platform/pgstore/vet.go, observability/traces; và (R2-03) phần
# GIẢI THÍCH của vết: core/pipeline/why.go, intelligence/router, intelligence/intent, cách observability/traces ghép chúng vào bước.
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

# ---- R2-03: giải thích của vết (đầu vào quyết định, phương án đã xét, lý do bị loại). Test bắt đột biến nằm ở observability/traces (vết
# dựng từ lượt THẬT), core/pipeline (why_test.go), intelligence/router và intelligence/intent. Đột biến chỉ đổi phần ghi giải thích hoặc
# cách hiển thị; không đột biến nào đổi quyết định của luồng hỏi (cái đó do bộ ca WIN và test của từng gói canh).
export PKGS="./observability/traces/ ./core/pipeline/ ./intelligence/intent/ ./intelligence/router/ ./core/contracts/"
W=core/pipeline/why.go RT=intelligence/router/router.go IN=intelligence/intent/intent.go D=observability/traces/decision.go ASK=core/pipeline/ask.go CT=core/contracts/why_text.go ST=observability/traces/step.go
m $W 'line := fmt.Sprintf("%s %s ← %s", sp.Kind, contracts.Quote(sp.Raw), sp.Rule)' 'line := fmt.Sprintf("%s %s ← %s", sp.Kind, contracts.Quote(sp.Raw), "")' 'WV1 luật sinh từng đoạn (Span.Rule) không được ghi'
m $W 'for _, amb := range in.Ambiguities {' 'for _, amb := range in.Ambiguities[:0] {' 'WV2 cách đọc khác của chỗ mơ hồ không được kể'
m $W 'for _, raw := range in.InvalidDates {' 'for _, raw := range in.InvalidDates[:0] {' 'WV3 ngày hỏng không được kể là bị bỏ'
m $W 'if nar.Slot != "" {' 'if false && nar.Slot != "" {' 'WV4 lần router lọc ứng viên bằng phạm vi không được kể'
m $W 'if dropped := len(nar.Before) - len(nar.After); dropped > 0 {' 'if dropped := len(nar.Before) - len(nar.After); false && dropped > 0 {' 'WV5 ứng viên bị loại khi gỡ mơ hồ không được kể'
m $W 'from = "do S4 gộp thêm"' 'from = "S3"' 'WV6 thực thể do S4 gộp thêm bị kể là của S3'
m $W 'ok && def.Route != "" && contracts.RouteKind(def.Route) != routed.Route {' 'ok && false && def.Route != "" && contracts.RouteKind(def.Route) != routed.Route {' 'WV7 route khai của ý định bị chặn không được kể'
m $W 'if plan.Route != routed.Route {' 'if false && plan.Route != routed.Route {' 'WV8 việc luồng hỏi ghi đè route không được kể'
m $W 'case slices.Contains(result.UsedSources, src):' 'case !slices.Contains(result.UsedSources, src):' 'WV9 vai của nguồn (tạo nên con số / không) bị đảo'
m $W 'for _, o := range c.Others {' 'for _, o := range c.Others[:0] {' 'WV10 nguồn bị loại vì thẩm quyền không được kể ở S7'
m $W 'if g.withheld != "" {' 'if g.withheld == "" {' 'WV11 cổng phán quyết: phán quyết và giữ phán quyết bị đảo'
m $W 'other := "confirm_no"' 'other := "confirm_yes"' 'WV12 phán quyết bị loại kể trùng phán quyết đã chọn'
m $W 'if o.ID != c.ID {' 'if false {' 'WV13 mọi ứng viên bị gán nguồn của ứng viên đầu tiên'
m $W 'if ci.suppressed != "" {' 'if false && ci.suppressed != "" {' 'WV14 không nói vì sao câu hỏi lại không kể ứng viên'
m $RT 'defer func() { plan.Checked, plan.Narrowed = checked, narrowed }()' 'defer func() { plan.Narrowed = narrowed }()' 'RV1 các bậc kiểm của router không được ghi'
m $RT 'nar.Chosen = ref.Key' 'nar.Chosen = ""' 'RV2 thực thể được chọn khi gỡ mơ hồ không được ghi'
m $RT 'nar.By = contracts.NarrowByScope' 'nar.By = contracts.NarrowByProject' 'RV3 cách gỡ mơ hồ (phạm vi / dự án) ghi sai'
m $RT 'check("route khai của ý định", plan.Route, true, def.Name+" → "+def.Route)' 'check("route khai của ý định", plan.Route, false, def.Name+" → "+def.Route)' 'RV4 bậc chọn route không được đánh dấu là bậc đã chọn'
m $RT 's.Entity.Key+": "+pick(out, "ngoài phạm vi", "trong phạm vi")+" của người hỏi"' 's.Entity.Key+": trong phạm vi của người hỏi"' 'RV5 dữ kiện phạm vi luôn nói "trong phạm vi"'
m $IN 'w.Origins = originsOf(res, candidates, unclear.Entities, clashSlot)' 'w.Origins = originsOf(res, nil, nil, clashSlot)' 'IV1 nguồn của ứng viên hỏi lại không được ghi'
m $IN 'w.reason("khớp đủ: %s (ưu tiên %d, từ khoá %s, %s)", d.Name, d.Priority, kw, via)' 'w.reason("khớp đủ: %s (ưu tiên %d, từ khoá %s, %s)", d.Name, d.Priority+1, kw, via)' 'IV2 ưu tiên của ý định khớp đủ ghi sai'
m $IN 'w.reject("%s: khớp một phần — từ khoá %s khớp nhưng câu thiếu thuật ngữ %s", d.Name, kw, strings.Join(d.Terms, "|"))' 'w.reject("%s: khớp một phần — từ khoá %s khớp nhưng câu thiếu thuật ngữ %s", d.Name, kw, "")' 'IV3 khớp một phần không nói thiếu thuật ngữ nào'
m $IN 'case d.Weak:' 'case false:' 'IV4 ý định yếu bị loại mà không nói là ý định yếu'
m $IN 'case d.Priority != top.Priority:' 'case d.Priority == top.Priority:' 'IV5 ý định bị loại vì ưu tiên thấp hơn lại bị kể là ít cụm từ khoá hơn'
m $IN 'w.reason("chọn %s: ưu tiên cao nhất (%d)", top.Name, top.Priority)' 'w.reason("chọn %s: ưu tiên cao nhất (%d)", top.Name, top.Priority+1)' 'IV6 lý do chọn ý định ghi sai ưu tiên'
m observability/traces/step.go 'if n := d.Why.Route; n.Recorded() {' 'if n := d.Why.Route; false && n.Recorded() {' 'TV28 giải thích của S5 do luồng hỏi ghi bị bỏ, quay về chỗ trống'
m observability/traces/step.go 's.Reason += "; " + joinReasons(n)' 's.Reason += ""' 'TV29 cổng phán quyết không được nối vào lý do của S9'
m $D 'func (n Notes) Recorded() bool { return len(n.Reasons) > 0 }' 'func (n Notes) Recorded() bool { return len(n.Reasons) > 99 }' 'TV30 chặng đã ghi giải thích bị coi là chưa ghi'
m observability/traces/record.go 'd.Why = Why{}' '_ = d' 'TV31 bản ghi bền giữ giải thích hai lần (Decision và Steps)'
# ---- hết R2-03
# ---- R2-03 vòng 2 (soát vòng 1: F1 bí mật, F2 khẳng định lỏng, F3 kích thước, F4 phạm vi). Nhãn [MX] là mã đột biến tay của người soát vòng 1.
# F2: 26 đột biến tay của người soát (15 sống sót ở vòng 1, trong đó 5 làm vết nói SAI mà test vẫn xanh) — nay đều bị bắt.
m $W 'afterText(refKeys(ci.shown)))' 'afterText(refKeys(ci.before)))' 'WV15 [MA] ứng viên "sau lọc" của câu hỏi lại in bằng danh sách "trước lọc"'
m $W 'if !slices.Contains(ids, e.ID) {' 'if false {' 'WV16 [MG1] đoạn truy xuất mà không được dẫn không được kể'
m $ASK 'why.Context.Reject("%sbản nháp bị loại ở S9 nên không đoạn nào được trích dẫn", prefix)' '_ = prefix' 'AV1 [MG2] bản nháp bị loại ở S9 không được ghi ở S7'
m $W 'g.subject, len(readPeriods(g.in)), markerCount' 'g.subject, 0, markerCount' 'WV17 [ME] đầu vào cổng: số kỳ đọc được luôn 0'
m $ASK 'ci = clarifyInfo{used: true, before: asked, shown: visible, hidden: hidden}' 'ci = clarifyInfo{used: true, before: cls.Candidates, shown: visible, hidden: hidden}' 'AV2 [ML] ứng viên trước lọc của câu hỏi lại thiếu thực thể do S4 gộp'
m $W 'if sp.Confidence > 0 && sp.Confidence < 1 {' 'if false && sp.Confidence > 0 && sp.Confidence < 1 {' 'WV18 [MJ] độ tin cậy của đoạn không được kể'
m $ASK 'if plan.Route == contracts.RouteHybrid {' 'if false && plan.Route == contracts.RouteHybrid {' 'AV3 [MQ] HYBRID không gắn tiền tố "số liệu:"'
m $ASK 'prefix = "tri thức: "' 'prefix = ""' 'AV4 [MR] HYBRID không gắn tiền tố "tri thức:"'
m $W 'prefix, subject, entities)' 'prefix, subject, entities+1)' 'WV19 [MS] dòng quyền của S7 nói sai số thực thể truy vấn'
m $W 'by, dropWhy = "dự án đã nêu trong câu", "không thuộc dự án đã nêu trong câu"' 'by, dropWhy = "dự án đã nêu trong câu", "ngoài phạm vi dữ liệu của người hỏi"' 'WV20 [MT] lý do dự án luôn nói "ngoài phạm vi"'
m $IN 'w.reason("thuật ngữ hiểu ngầm %s (câu nhắc thực thể mà không nói vật tư và không có đại lượng đo được nào khác — chính sách của pack)", implicit)' '_ = implicit' 'IV7 [MU] không kể thuật ngữ hiểu ngầm'
m $IN 'w.reject("%s (tri thức): khớp cùng nhưng bị bỏ — câu đã nêu mốc thời gian, ngày hỏng hay nhiều thực thể của một câu hỏi số liệu, hỏi lại phần số liệu thay vì chuyển sang câu hỏi quy trình", defNames(knowledge))' '_ = defNames(knowledge)' 'IV8 [MV] ý định tri thức bị bỏ không được kể'
m $RT 'nar := contracts.Narrowing{Slot: slot, Before: keysOf(cls.Candidates)}' 'nar := contracts.Narrowing{Slot: slot, Before: slices.Compact(slices.Sorted(slices.Values(keysOf(cls.Candidates))))}' 'RV6 [MW] ứng viên trùng bị gộp trong số đếm của router'
m $RT 'defer func() { plan.Checked, plan.Narrowed = checked, narrowed }()' 'defer func() { plan.Checked = checked; _ = narrowed }()' 'RV7 [MX] Narrowed không được gắn vào plan'
m $ST 's.Rejected = slices.Clone(n.Rejected)' '_ = n' 'TV32 [MY] Rejected của cổng phán quyết bị mất ở S9'
m $W 'for _, e := range extra {' 'for _, e := range extra[:0] {' 'WV21 [MZ] bằng chứng đi kèm thêm không được kể ở S7'
m $W 'displayNumber(g.claimed.Number), displayNumber(g.total.Number), word(g.tmpl))' 'displayNumber(g.total.Number), displayNumber(g.claimed.Number), word(g.tmpl))' 'WV22 [MB] số của người hỏi và số có thẩm quyền đảo chỗ trong dòng so sánh'
m $W 'prefix, len(pkg.Evidence), facts, len(main)-facts, len(extra))' 'prefix, len(pkg.Evidence), facts, len(main), len(extra))' 'WV23 [MC] số bằng chứng khác của truy vấn đếm sai'
m $W 'n.Reason("luồng hỏi ghi đè route %s thành %s: %s", routed.Route, plan.Route, plan.Reason)' 'n.Reason("luồng hỏi ghi đè route %s thành %s: %s", plan.Route, routed.Route, plan.Reason)' 'WV24 [MD] route bị ghi đè kể ngược chiều'
m $W '(tối đa %d): %s", prefix, len(ev), k, strings.Join(ids, ", "))' '(tối đa %d): %s", prefix, len(ev), len(ev), strings.Join(ids, ", "))' 'WV25 [MF1] trần số đoạn ghi bằng số đoạn thực tế'
m $W 'ids[i] = e.ID + " (" + e.Source.Name + ")"' 'ids[i] = e.ID' 'WV26 [MF2] không kể tên nguồn của đoạn'
m $ASK 'hidden = append(hidden, hiddenCandidate{c, "không tra được tổ tiên trong danh mục (đóng mặc định)"})' 'hidden = append(hidden, hiddenCandidate{c, "ngoài phạm vi dữ liệu của người hỏi"})' 'AV5 [MH] ứng viên không tra được tổ tiên bị kể là ngoài phạm vi'
m $IN 'w.reject("%s: ý định số liệu bị bỏ — thiếu khe thực thể và câu không có dấu hiệu hỏi số liệu, coi là câu hỏi quy trình", r.Intent.Name)' '_ = r.Intent.Name' 'IV9 [MY2] ý định số liệu bị bỏ không được kể'
m $IN 'w.reason("từ khoá lấy trong mệnh đề hỏi %s (mệnh đề chứa cụm xác nhận quyết định ý định, các mệnh đề trước là ngữ cảnh)", contracts.Quote(in.Raw[fs:fe]))' '_ = fs' 'IV10 [MZ2] mệnh đề hỏi quyết định ý định không được kể'
m $W 'line += fmt.Sprintf(" (%s, loại %s)", e.Name, e.Type)' 'line += fmt.Sprintf(" (%s, loại %s)", e.Key, e.Type)' 'WV27 [MN] thực thể rõ in sai tên'
m $W 'prefix, src, groups[src].authority, groups[src].count, role)' 'prefix, src, groups[src].authority+1, groups[src].count, role)' 'WV28 [MO] dòng nguồn in thẩm quyền sai'
# F1: chữ tự do của người hỏi qua MỘT hàm (contracts.Quote), không thoát nháy; câu mang bí mật thì không trích chữ nào
m $CT 'return QuoteOpen + s + QuoteClose' 'return QuoteOpen + strings.NewReplacer(`"`, `\"`).Replace(s) + QuoteClose' 'CV1 Quote thoát nháy kiểu %q (lỗi F1: mẫu che bí mật không còn nhận ra)'
m $CT 's = strings.NewReplacer(QuoteOpen, `"`, QuoteClose, `"`).Replace(s)' 's = strings.NewReplacer().Replace(s)' 'CV2 Quote để “ ” có sẵn trong chữ lọt vào đoạn được bao'
m $CT 's = strings.Join(strings.Fields(s), " ")' '_ = strings.Fields' 'CV3 Quote không gộp khoảng trắng và xuống dòng'
m $CT 'if r := []rune(s); len(r) > maxQuoteRunes {' 'if r := []rune(s); len(r) > 1<<20 {' 'CV4 Quote không cắt chữ dài'
m $CT 'if len(items) <= max {' 'if len(items) <= max+1000 {' 'CV5 Brief không cắt danh sách dài'
m $D 'out[i] = quoted.ReplaceAllString(l, Withheld)' 'out[i] = l' 'DV1 câu mang bí mật vẫn trích chữ của câu vào giải thích'
m $ASK 'if traces.Sanitize(in.Raw) != in.Raw {' 'if false && traces.Sanitize(in.Raw) != in.Raw {' 'AV6 Handle không giữ lại chữ của câu khi câu mang bí mật'
# F3: vết có trần
m $D 'if count[l]++; count[l] == 1 {' 'if count[l]++; true {' 'DV2 dòng giống nhau không được gộp'
m $D 'if i == limit {' 'if i == limit+100000 {' 'DV3 danh sách giải thích không bị cắt ở trần'
m $D 'if c := count[l]; c > 1 {' 'if c := count[l]; false && c > 1 {' 'DV4 dòng gộp không nói số lần'
m $ASK 'd.Why = why.Compact(maxWhyLines)' 'd.Why = why' 'AV7 Handle không gộp / cắt phần giải thích'
m $W 'len(keys), contracts.Brief(keys, maxListed, "ứng viên"))' 'len(keys), strings.Join(keys, ", "))' 'WV29 danh sách ứng viên của cụm mơ hồ không bị cắt'
m $W 'if d := distinct(keys); len(d) != len(keys) {' 'if d := distinct(keys); false && len(d) != len(keys) {' 'WV30 số ứng viên không nói khoá trùng'
m $W 'out[i].n++' 'out[i].n += 0' 'WV31 bậc kiểm giống nhau gộp mà không nói số lần'
m $W 'if i == maxListed {' 'if i == maxListed+100 {' 'WV32 nguồn ứng viên hỏi lại không bị cắt'
m $IN 'if !slices.ContainsFunc(w.Origins, func(x Origin) bool { return x.ID == o.ID }) {' 'if true {' 'IV11 nguồn của ứng viên ghi lặp theo số lần nhắc'
# F4: vết do chính người hỏi đọc không kể tên / loại / trạng thái phạm vi của từng ứng viên ngoài phạm vi
m $ASK 'why.Resolve = explainResolve(in, res, routed.Narrowed, ci, a.RT.Policy.InScope(pr, nil))' 'why.Resolve = explainResolve(in, res, routed.Narrowed, ci, true)' 'AV8 người hỏi phạm vi hạn chế vẫn thấy tên và loại thực thể'
m $W 'if fullScope {' 'if true {' 'WV33 tên và loại thực thể kể cho mọi người hỏi'
m $W 'n.Reject("%d ứng viên bị loại khi gỡ mơ hồ: %s (vết không kể từng ứng viên)", dropped, dropWhy)' 'n.Reject("%d ứng viên bị loại khi gỡ mơ hồ: %s (vết không kể từng ứng viên)", dropped+1, dropWhy)' 'WV34 số ứng viên bị loại khi gỡ mơ hồ ghi sai'
m $W 'n.Reject("%d ứng viên bị loại khỏi câu hỏi lại: %s (vết không kể từng ứng viên)", len(byWhy[why]), why)' 'n.Reject("%d ứng viên bị loại khỏi câu hỏi lại: %s (vết không kể từng ứng viên)", len(byWhy[why])+1, why)' 'WV35 số ứng viên bị loại khỏi câu hỏi lại ghi sai'
# F1 qua cửa HTTP thật (test cmd/eais chạy trên bộ nhớ và PG) và F5 (đầu vào của cổng phán quyết khớp điều kiện của verdictGap — chỉ test của gói
# observability/traces phải bắt, không nhờ test của luồng hỏi): đổi điều kiện ở verdictGap mà vết vẫn nói như cũ thì test đỏ.
export PKGS=./cmd/eais/
m $ASK 'if traces.Sanitize(in.Raw) != in.Raw {' 'if false && traces.Sanitize(in.Raw) != in.Raw {' 'AV9 qua HTTP: vết người hỏi tự đọc còn đuôi bí mật của câu'
export PKGS=./observability/traces/
CL=core/pipeline/claim.go
m $CL 'if len(readPeriods(in)) != 1 {' 'if len(readPeriods(in)) > 1 {' 'GV1 [F5] cổng: câu không nêu mốc nào vẫn qua chiều "period"'
m $CL 'if !normalization.HasDiacritic(raw) {' 'if false && !normalization.HasDiacritic(raw) {' 'GV2 [F5] cổng: câu không dấu vẫn qua chiều "accent"'
m $CL 'if markerCount(raw, a.RT.ConfirmMarkers) != 1 {' 'if markerCount(raw, a.RT.ConfirmMarkers) > 2 {' 'GV3 [F5] cổng: hai cụm xác nhận vẫn qua chiều "marker"'
# ---- hết R2-03 vòng 2

fresh >/dev/null 2>&1; psql -q -X "$EAIS_PG_ADMIN_DSN" -c "DROP DATABASE IF EXISTS $DB_DOT_BIEN WITH (FORCE)"
echo "TỔNG: bị bắt $bat · sống sót $song · khác $khac"
[ "$song" -eq 0 ] && [ "$khac" -eq 0 ]
