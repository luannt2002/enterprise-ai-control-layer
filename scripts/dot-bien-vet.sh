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
# Khối R2-03 (giải thích của vết) không đụng PG (các dòng WV / IV / RV / AV / DV / XV / NV / QV / LV); họ NV / QV / LV sinh từ mã hiện tại bằng
# scripts/dot-bien-vet-sinh.py, tắt bằng DOT_BIEN_SINH=0.
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
# DOT_BIEN_CHIA=k/n: chỉ chạy các đột biến có số thứ tự (từ 0) chia n dư k — chia danh mục cho n tiến trình chạy SONG SONG. mutate.sh sửa tệp tại chỗ nên mỗi tiến trình
# phải ở MỘT cây riêng (bản sao của kho), và có EAIS_PG_DSN / DB_DOT_BIEN riêng. Không đặt = chạy tất cả.
chia=${DOT_BIEN_CHIA:-} dem=0
chon() {
	[ -z "$chia" ] && return 0
	local k=${chia%/*} n=${chia#*/} mine=1
	[ $((dem % n)) -eq "$k" ] || mine=0
	dem=$((dem + 1))
	[ "$mine" -eq 1 ]
}
m_chay() {
	local res rc
	res=$(scripts/mutate.sh "$1" "$2" "$3" "$4"); rc=$?
	printf '%s\nrc=%s\n' "$res" "$rc"
	case $rc in 0) bat=$((bat + 1)) ;; 3) song=$((song + 1)) ;; *) khac=$((khac + 1)) ;; esac
}
m() { chon || return 0; m_chay "$@"; }
fresh() { psql -q -X "$EAIS_PG_ADMIN_DSN" -c "DROP DATABASE IF EXISTS $DB_DOT_BIEN WITH (FORCE)" -c "CREATE DATABASE $DB_DOT_BIEN"; }
# dsn_db <dsn> <database>: thay tên database trong DSN dạng postgres://vai@máy:cổng/<database>[?tham số].
dsn_db() { printf '%s' "$1" | sed -E 's#^([a-z]+://[^/]*)/[^/?]*#\1/'"$2"'#'; }
# Nền xanh trên database đột biến (lỗi 5c): chạy `mutate.sh --nen` MỘT lần trên database vừa dựng, rồi dựng lại database cho đột biến —
# để mutate.sh tự chạy nền ở đó thì nền áp migration GỐC trước đột biến và đột biến migration sống giả.
nen_mm=""
mm() {
	chon || return 0
	fresh || { echo "[$4] không dựng được database $DB_DOT_BIEN"; echo "rc=5"; khac=$((khac + 1)); return; }
	local a d
	a=$(dsn_db "$EAIS_PG_ADMIN_DSN" "$DB_DOT_BIEN") d=$(dsn_db "$EAIS_PG_DSN" "$DB_DOT_BIEN")
	if [ -z "$nen_mm" ]; then
		nen_mm=$(EAIS_PG_ADMIN_DSN=$a EAIS_PG_DSN=$d scripts/mutate.sh --nen)
		echo "$nen_mm"
		fresh || { echo "[$4] không dựng được database $DB_DOT_BIEN"; echo "rc=5"; khac=$((khac + 1)); return; }
	fi
	EAIS_PG_ADMIN_DSN=$a EAIS_PG_DSN=$d MUTATE_NEN_CO_SAN=1 m_chay "$@"
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
m $S 's = secretMasks[i].re.ReplaceAllString(s, secretMasks[i].repl)' '_ = secretMasks[i]' 'TV1 Sanitize không che gì'
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
# Giá trị trong nháy đọc theo hai cách rồi che hợp (sanitize.go: quotedEscaped / quotedRaw, valueMask) — đột biến từng cách đọc và
# phép gộp; mỗi dòng có ca phân biệt trong sanitize_test.go (TestSanitizeDauRaChinhXac, TestSanitizeGiaTriKetThucBangGachCheo).
QE_DQ='"(?:[^"\\]|\\[\s\S])*"?|'
QE_SQ=$'\'(?:[^\'\\\\]|\\\\[\\s\\S])*\'?'
m $S "$QE_DQ" '"(?:[^"\\\s]|\\[\s\S])*"?|' 'TV25 giá trị trong nháy dừng ở khoảng trắng (lọt đuôi, R1-04)'
m $S "$QE_DQ" '"(?:[^"\\]|\\[\s\S])*"|' 'TV27 nháy không đóng không được che'
m $S "$QE_DQ" '"[^"]*"?|' 'TV37 nháy kép không đọc nháy thoát (lọt đuôi JSON, 5a(2))'
m $S "$QE_DQ" '"(?:[^"\\]|\\.)*"?|' 'TV39 nháy thoát ngay trước xuống dòng không được đọc'
m $S "$QE_SQ" $'\'[^\']*\'?' 'TV38 nháy đơn không đọc nháy thoát (khoá = giá trị và Cookie)'
m $S 'for _, re := range []*regexp.Regexp{m.escaped, m.raw} {' 'for _, re := range []*regexp.Regexp{m.escaped} {' 'TV33 chỉ đọc theo nháy thoát (lọt bí mật kế tiếp, 5a-1)'
m $S 'for _, re := range []*regexp.Regexp{m.escaped, m.raw} {' 'for _, re := range []*regexp.Regexp{m.raw} {' 'TV34 chỉ đọc thô (lọt đuôi sau nháy thoát)'
m $S 'for ; i < len(secretMasks) && secretMasks[i].value != nil; i++ {' 'for ; i < len(secretMasks) && secretMasks[i].value != nil && len(ms) == 0; i++ {' 'TV40 che lần lượt từng mẫu tên + giá trị (lọt Cookie sau giá trị kết thúc bằng \\, soát lại 5abc 5a-R1)'
m $S 'return spans[i].start < spans[j].start' 'return false' 'TV41 không sắp vùng trước khi gộp (lọt khi ≥ 3 cặp mà hai cách đọc nháy bất đồng, soát lại 5abc 5a-R2)'
m $S 'if cur.line {' 'if false && cur.line {' 'TV45 vùng gộp chứa giá trị Cookie trần không kéo tới hết dòng (lọt phần sau nháy qua xuống dòng, soát 5a vòng 3 5a3-1)'
m $S '`[^\r\n]+`, true)}' '`[^\r\n]+`, false)}' 'TV46 mẫu Cookie không đánh dấu là mẫu dòng (5a3-1)'
m $S 'cur.line = cur.line || spans[i].line' 'cur.line = cur.line' 'TV48 cờ dòng không lan khi vùng gộp bắt đầu ở cặp trước Cookie (hình lai 5a-R1 + 5a3-1)'
m $S $'} else {\n\t\t\tb.WriteString(Masked)' $'} else {\n\t\t\tb.WriteString(Masked + Masked)' 'TV42 giá trị trần nhân đôi dấu che (vỡ luỹ đẳng)'
m $S '[^\r\n]+' '[^\r\n;]+' 'TV43 Cookie trần dừng ở ";" (từ 5a3-1 vùng Cookie trần kéo tới hết dòng: chỉ còn lọt khi giá trị bắt đầu bằng ";")'
m $S "[^\\s\"',;&]+" "[^\\s\"',;]+" 'TV44 giá trị trần không dừng ở "&" (nuốt cặp kế tiếp)'
m $S 'cur.end = max(cur.end, spans[i].end)' 'cur.end = max(cur.end, cur.end)' 'TV35 gộp vùng chồng nhau mà không nới vùng'
m $S $'if q := s[cur.start]; q == \'"\' || q == \'\\\'\' {' 'if q := s[cur.start]; false {' 'TV36 dấu che bỏ nháy bao quanh (phá cấu trúc JSON)'
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
m $W 'contracts.Quote(displayNumber(g.claimed.Number)), displayNumber(g.total.Number), word(g.tmpl))' 'displayNumber(g.total.Number), contracts.Quote(displayNumber(g.claimed.Number)), word(g.tmpl))' 'WV22 [MB] số của người hỏi và số có thẩm quyền đảo chỗ trong dòng so sánh'
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
m $ASK 'if traces.Sanitize(in.Raw) != in.Raw || traces.Sanitize(req.Text) != req.Text {' 'if false && (traces.Sanitize(in.Raw) != in.Raw || traces.Sanitize(req.Text) != req.Text) {' 'AV6 Handle không giữ lại chữ của câu khi câu mang bí mật'
# F3: vết có trần
m $D 'if count[l]++; count[l] == 1 {' 'if count[l]++; true {' 'DV2 dòng giống nhau không được gộp'
m $D 'if i == limit {' 'if i == limit+100000 {' 'DV3 danh sách giải thích không bị cắt ở trần'
m $D 'if c > 1 {' 'if false && c > 1 {' 'DV4 dòng gộp không nói số lần'
m core/pipeline/scope_redact.go 'return red.why(w).Compact(maxWhyLines, maxWhyLineRunes)' 'return red.why(w)' 'AV7 Handle không gộp / cắt phần giải thích'
m $W 'len(keys), contracts.Brief(keys, maxListed, "ứng viên"))' 'len(keys), strings.Join(keys, ", "))' 'WV29 danh sách ứng viên của cụm mơ hồ không bị cắt'
m $W 'if d := distinct(keys); len(d) != len(keys) {' 'if d := distinct(keys); false && len(d) != len(keys) {' 'WV30 số ứng viên không nói khoá trùng'
m $W 'out[i].n++' 'out[i].n += 0' 'WV31 bậc kiểm giống nhau gộp mà không nói số lần'
m $W 'if i == maxListed {' 'if i == maxListed+100 {' 'WV32 nguồn ứng viên hỏi lại không bị cắt'
m $IN 'if !slices.ContainsFunc(w.Origins, func(x Origin) bool { return x.ID == o.ID }) {' 'if true {' 'IV11 nguồn của ứng viên ghi lặp theo số lần nhắc'
# F4: vết do chính người hỏi đọc không kể tên / loại / trạng thái phạm vi của từng ứng viên ngoài phạm vi
m $ASK 'why.Resolve = explainResolve(in, res, routed.Narrowed, ci, a.RT.Policy.InScope(pr, nil))' 'why.Resolve = explainResolve(in, res, routed.Narrowed, ci, true)' 'AV8 người hỏi phạm vi hạn chế vẫn thấy tên và loại thực thể'
m $W 'if fullScope {' 'if true {' 'WV33 tên và loại thực thể kể cho mọi người hỏi'
m $W 'n.Reject("%d ứng viên bị loại khi gỡ mơ hồ: %s (vết không nêu trạng thái phạm vi của từng ứng viên)", dropped, dropWhy)' 'n.Reject("%d ứng viên bị loại khi gỡ mơ hồ: %s (vết không nêu trạng thái phạm vi của từng ứng viên)", dropped+1, dropWhy)' 'WV34 số ứng viên bị loại khi gỡ mơ hồ ghi sai'
m $W 'n.Reject("%d ứng viên bị loại khỏi câu hỏi lại: %s (vết không nêu trạng thái phạm vi của từng ứng viên)", len(byWhy[why]), why)' 'n.Reject("%d ứng viên bị loại khỏi câu hỏi lại: %s (vết không nêu trạng thái phạm vi của từng ứng viên)", len(byWhy[why])+1, why)' 'WV35 số ứng viên bị loại khỏi câu hỏi lại ghi sai'
# F1 qua cửa HTTP thật (test cmd/eais chạy trên bộ nhớ và PG) và F5 (đầu vào của cổng phán quyết khớp điều kiện của verdictGap — chỉ test của gói
# observability/traces phải bắt, không nhờ test của luồng hỏi): đổi điều kiện ở verdictGap mà vết vẫn nói như cũ thì test đỏ.
export PKGS=./cmd/eais/
m $ASK 'if traces.Sanitize(in.Raw) != in.Raw || traces.Sanitize(req.Text) != req.Text {' 'if false && (traces.Sanitize(in.Raw) != in.Raw || traces.Sanitize(req.Text) != req.Text) {' 'AV9 qua HTTP: vết người hỏi tự đọc còn đuôi bí mật của câu'
m $ASK 'if traces.Sanitize(in.Raw) != in.Raw || traces.Sanitize(req.Text) != req.Text {' 'if traces.Sanitize(in.Raw) != in.Raw {' 'AV11 [soát vòng 3] qua HTTP: điều kiện giữ chữ bỏ vế câu gốc, vết người hỏi tự đọc còn token đứng sau ký tự bị gấp'
export PKGS=./observability/traces/
CL=core/pipeline/claim.go
m $CL 'if len(readPeriods(in)) != 1 {' 'if len(readPeriods(in)) > 1 {' 'GV1 [F5] cổng: câu không nêu mốc nào vẫn qua chiều "period"'
m $CL 'if !normalization.HasDiacritic(raw) {' 'if false && !normalization.HasDiacritic(raw) {' 'GV2 [F5] cổng: câu không dấu vẫn qua chiều "accent"'
m $CL 'if markerCount(raw, a.RT.ConfirmMarkers) != 1 {' 'if markerCount(raw, a.RT.ConfirmMarkers) > 2 {' 'GV3 [F5] cổng: hai cụm xác nhận vẫn qua chiều "marker"'
# ---- hết R2-03 vòng 2

# ---- R2-03 vòng 3 (soát vòng 2: N1 giá trị bí mật dạng số, N2 khẳng định lỏng, N3 trần theo độ dài dòng, N4 chữ cho khớp thực tế)
export PKGS="./observability/traces/ ./core/pipeline/ ./intelligence/intent/ ./intelligence/router/ ./core/contracts/"
# N2: 33 đột biến MỚI do người soát vòng 2 tự nghĩ (chỉ đổi phần ghi giải thích); 11 con sống sót ở vòng 2 (X6 X21 X24 X26 X27 X28 X29 X30 X39 X40 X42) nay bị test khoá
# nguyên dòng bắt hết. Nhãn XV<k> giữ số X<k> của tệp soát để đối chiếu.
m $W 'return fmt.Sprintf("%d ứng viên (%d khoá khác nhau)", len(keys), len(d))' 'return fmt.Sprintf("%d ứng viên (%d khoá khác nhau)", len(d), len(keys))' 'XV1 [soát vòng 2] số ứng viên và số khoá khác nhau đảo chỗ'
m $W 'len(refKeys(ci.before)), afterText(refKeys(ci.shown)))' 'len(ci.before), afterText(refKeys(ci.shown)))' 'XV2 [soát vòng 2] số ứng viên trước lọc của câu hỏi lại đếm cả trùng'
m $W 'shown := distinctRefs(ci.shown)' 'shown := ci.shown' 'XV3 [soát vòng 2] nguồn ứng viên hỏi lại in lặp ứng viên trùng'
m $W 'không có miền và đơn vị chuẩn để xác nhận một con số, nên chỉ trả số", g.metric)' 'không có miền và đơn vị chuẩn để xác nhận một con số, nên chỉ trả số", g.subject)' 'XV5 [soát vòng 2] cổng: chỉ số dẫn xuất nói thành tên chủ thể'
m $W '"accent":   "câu không dấu: bộ chuẩn hoá khớp bỏ dấu nên nhiều chữ trùng nhau",' '"accent":   "mốc thời gian: câu không nêu mốc (kỳ mặc định của pack) hoặc nêu nhiều mốc",' 'XV6 [soát vòng 2] nghĩa của chiều accent nói sai'
m $W 'if len(in.Spans) == 0 {' 'if len(in.Spans) == 1 {' 'XV7 [soát vòng 2] S2 nói "không có đoạn nào" khi có đúng một đoạn'
m $W 'if len(res.Entities)+len(res.Ambiguous) == 0 {' 'if len(res.Entities)+len(res.Ambiguous) == 1 {' 'XV8 [soát vòng 2] S3 nói "không nhận ra thực thể nào" khi có đúng một'
m $W 'step = fmt.Sprintf("%d) %s — %s → chọn %s", i+1, g.check.Rule, g.check.Detail, g.check.Route)' 'step = fmt.Sprintf("%d) %s — %s → chọn %s", i+2, g.check.Rule, g.check.Detail, g.check.Route)' 'XV9 [soát vòng 2] số thứ tự bậc kiểm đã chặn lệch một'
m $W 'step := fmt.Sprintf("%d) %s: qua", i+1, g.check.Rule)' 'step := fmt.Sprintf("%d) %s: qua", i+2, g.check.Rule)' 'XV10 [soát vòng 2] số thứ tự bậc kiểm đã qua lệch một'
m $W 'role := "không tạo nên con số chính (nguồn thẩm quyền thấp hơn, hoặc lệch)"' 'role := "tạo nên con số chính"' 'XV11 [soát vòng 2] vai mặc định của nguồn nói là tạo nên con số chính'
m $W 'why := "thẩm quyền thấp hơn nguồn được chọn " + c.ChosenID' 'why := "ngang thẩm quyền với nguồn còn lại — không chọn được, chờ người duyệt"' 'XV12 [soát vòng 2] lý do nguồn bị loại luôn nói ngang thẩm quyền'
m $W 'if len(ev) == 0 {' 'if len(ev) == 1 {' 'XV13 [soát vòng 2] truy xuất nói trả 0 đoạn khi trả 1'
m $W 'prefix, len(cited), strings.Join(ids, ", "))' 'prefix, len(ev), strings.Join(ids, ", "))' 'XV14 [soát vòng 2] số đoạn được dẫn bằng số đoạn truy xuất'
m $W 'word(other), g.tmpl)' 'word(other), other)' 'XV16 [soát vòng 2] phán quyết bị loại nói chọn chính nó'
m $W 'n.Reason("%s", routed.Reason)' 'n.Reason("%s", plan.Reason)' 'XV39 [soát vòng 2] lý do của router lấy từ kế hoạch đã bị ghi đè'
m $W 'contracts.RouteKind(def.Route) != routed.Route {' 'contracts.RouteKind(def.Route) != plan.Route {' 'XV40 [soát vòng 2] route khai bị loại so với kế hoạch cuối thay vì của router'
m $ASK 'hidden = append(hidden, hiddenCandidate{c, "ngoài phạm vi dữ liệu của người hỏi"})' 'hidden = append(hidden, hiddenCandidate{c, "không tra được tổ tiên trong danh mục (đóng mặc định)"})' 'XV18 [soát vòng 2] ứng viên ngoài phạm vi bị kể là không tra được tổ tiên'
m $ASK 'ci.shown, ci.suppressed = nil, "câu hỏi lại về con số hay mốc thời gian, thực thể đã rõ (router đã gỡ bí danh mơ hồ)"' 'ci.shown, ci.suppressed = nil, ""' 'XV19 [soát vòng 2] không nói vì sao không kể ứng viên (nguồn)'
m $ASK 'confirm: confirm, compare: compare != "", withheld: withheld,' 'confirm: confirm, compare: false, withheld: withheld,' 'XV21 [soát vòng 2] cổng không biết câu so sánh'
m $ASK 'if traces.Sanitize(in.Raw) != in.Raw || traces.Sanitize(req.Text) != req.Text {' 'if traces.Sanitize(req.Text) != req.Text {' 'XV42 [soát vòng 2] điều kiện giữ chữ chỉ dùng câu gốc, bỏ vế câu đã gấp ký tự'
m $ASK 'if traces.Sanitize(in.Raw) != in.Raw || traces.Sanitize(req.Text) != req.Text {' 'if traces.Sanitize(in.Raw) != in.Raw {' 'XV43 [soát vòng 3] điều kiện giữ chữ chỉ dùng câu đã gấp ký tự, bỏ vế câu gốc: token đứng sau ký tự bị gấp lọt vào giải thích và vết bền'
m $ASK 'if traces.Sanitize(in.Raw) != in.Raw || traces.Sanitize(req.Text) != req.Text {' 'if traces.Sanitize(in.Raw) != in.Raw || in.Raw != req.Text {' 'XV44 [soát vòng 3] cổng che mọi câu có ký tự bị gấp, không chỉ câu mang bí mật nhận ra được (che oan)'
m $IN 'sd.Name, len(ps), len(in.InvalidDates))' 'sd.Name, len(in.InvalidDates), len(ps))' 'XV23 [soát vòng 2] số mốc và số ngày hỏng đảo chỗ'
m $IN 'fmt.Sprintf("không có đúng một ý định yếu đòi thuật ngữ của câu (có %d)", len(gen))' 'fmt.Sprintf("không có đúng một ý định yếu đòi thuật ngữ của câu (có %d)", len(gen)+1)' 'XV24 [soát vòng 2] số ý định yếu ghi sai'
m $IN 'w.reason("chọn %s: cùng ưu tiên %d nhưng nhiều cụm từ khoá hơn (%d)", top.Name, top.Priority, score[top.Name])' 'w.reason("chọn %s: cùng ưu tiên %d nhưng nhiều cụm từ khoá hơn (%d)", top.Name, top.Priority, score[top.Name]+1)' 'XV26 [soát vòng 2] số cụm từ khoá của ý định thắng ghi sai'
m $RT 'check("capability của vế so sánh", contracts.RouteReject, !ok, cdef.Capability+": "+pick(ok, "có", "thiếu")) {' 'check("capability của vế so sánh", contracts.RouteReject, !ok, cdef.Capability+": "+pick(ok, "thiếu", "có")) {' 'XV27 [soát vòng 2] capability của vế so sánh nói ngược có / thiếu'
m $RT 'e.Key+": "+pick(out, "ngoài phạm vi", "trong phạm vi"))' 'e.Key+": "+pick(out, "trong phạm vi", "ngoài phạm vi"))' 'XV28 [soát vòng 2] phạm vi của câu hỏi nghĩa số liệu nói ngược'
m $RT 'orNone(cls.Compare, "không có vế so sánh"))' 'orNone("", "không có vế so sánh"))' 'XV29 [soát vòng 2] bậc câu so sánh không nêu tên vế'
m $RT 'orNone(cls.Knowledge, "không có ý định tri thức khớp kèm")) {' 'orNone("", "không có ý định tri thức khớp kèm")) {' 'XV30 [soát vòng 2] bậc HYBRID không nêu tên ý định tri thức'
m $D 'fmt.Sprintf("… và %d ý khác không in (trần %d ý mỗi danh sách)", len(order)-limit, limit)' 'fmt.Sprintf("… và %d ý khác không in (trần %d ý mỗi danh sách)", len(order)-limit-1, limit)' 'XV31 [soát vòng 2] số ý bị cắt ghi lệch một'
m $D 'l += fmt.Sprintf(" (×%d)", c)' 'l += fmt.Sprintf(" (×%d)", c+1)' 'XV32 [soát vòng 2] số lần lặp ghi lệch một'
m $CT 'return fmt.Sprintf("%s và %d %s khác", strings.Join(items[:max], ", "), len(items)-max, noun)' 'return fmt.Sprintf("%s và %d %s khác", strings.Join(items[:max], ", "), len(items)-max+1, noun)' 'XV33 [soát vòng 2] Brief ghi sai số phần tử bị cắt'
m $CT 'if r := []rune(s); len(r) > maxQuoteRunes {' 'if r := []rune(s); len(r) >= maxQuoteRunes {' 'XV34 [soát vòng 2] Quote cắt cả chữ đúng bằng trần'
m $CT 's = string(r[:maxQuoteRunes]) + "…"' 's = string(r[:maxQuoteRunes-1]) + "…"' 'XV35 [soát vòng 2] Quote cắt ngắn hơn trần một ký tự'
m $ST 'func joinReasons(n Notes) string { return strings.Join(n.Reasons, "; ") }' 'func joinReasons(n Notes) string { return strings.Join(n.Reasons, ", ") }' 'XV36 [soát vòng 2] dấu nối các ý của lý do đổi'
# N1 (số / ngày / từ khoá của người hỏi qua Quote), N3 (thang bậc gộp bậc "qua", trần theo độ dài dòng), N4 (chữ không hứa hơn mã làm được)
m $W 'contracts.Quote(displayNumber(g.claimed.Number)), a.unitLabel(g.claimed.Unit))' 'displayNumber(g.claimed.Number), a.unitLabel(g.claimed.Unit))' 'WV36 [N1] con số xác nhận ở đầu vào của cổng không qua Quote'
m $W 'contracts.Quote(displayNumber(g.claimed.Number)), displayNumber(g.total.Number), word(g.tmpl))' 'displayNumber(g.claimed.Number), displayNumber(g.total.Number), word(g.tmpl))' 'WV37 [N1] số trong phép so sánh không qua Quote'
m $W 'vals[i] = contracts.Quote(valueText(v))' 'vals[i] = valueText(v)' 'WV38 [N1] giá trị bằng chứng của người hỏi (S7) không qua Quote'
m $IN 'contracts.Quote(v.Text), contracts.Quote(ps[0].Raw))' 'v.Text, contracts.Quote(ps[0].Raw))' 'IV12 [N1] giá trị mốc đã chuẩn hoá không qua Quote'
m $IN 'kw := quoteAll(phrases[i])' 'kw := strings.Join(phrases[i], ", ")' 'IV13 [N1] cụm từ khoá khớp ở dòng khớp đủ / một phần / yếu không qua Quote'
m $IN 'out = append(out, fmt.Sprintf("%s [%s]", d.Name, quoteAll(phrases[i])))' 'out = append(out, fmt.Sprintf("%s [%s]", d.Name, strings.Join(phrases[i], ", ")))' 'IV14 [N1] cụm từ khoá khớp ở dòng "từ khoá khớp" không qua Quote'
m $IN 'quoted[i] = contracts.Quote(p)' 'quoted[i] = p' 'IV15 [N1] quoteAll không bao cặp'
m $W 'last >= 0 && !g.check.Fired && !out[last].check.Fired && out[last].check.Rule == g.check.Rule' 'last >= 0 && !out[last].check.Fired && out[last].check.Rule == g.check.Rule' 'WV39 [N3] bậc đã chặn gộp vào bậc "qua" liền trước'
m $W 'last >= 0 && !g.check.Fired && !out[last].check.Fired && out[last].check.Rule == g.check.Rule' 'last >= 0 && !g.check.Fired && out[last].check.Rule == g.check.Rule' 'WV40 [N3] bậc "qua" gộp vào bậc đã chọn liền trước'
m $W '&& out[last].check.Rule == g.check.Rule && out[last].check.Route == g.check.Route {' '&& out[last].check.Route == g.check.Route {' 'WV41 [N3] gộp cả các bậc "qua" khác tên'
m $W '&& out[last].check.Rule == g.check.Rule && out[last].check.Route == g.check.Route {' '&& out[last].check.Rule == g.check.Rule {' 'WV42 [N3] gộp các bậc "qua" cùng tên khác route'
m $W 'out[last].n += g.n' 'out[last].n++' 'WV43 [N3] gộp bậc "qua" mà đếm số lần thiếu'
m $W 'ladder := ladderOf(groups)' 'ladder := groups' 'WV44 [N3] thang bậc không gộp bậc "qua" liên tiếp'
m $W 'maxWhyLineRunes = 1500' 'maxWhyLineRunes = 150000' 'WV45 [N3] trần độ dài dòng quá lớn'
m core/pipeline/scope_redact.go 'return red.why(w).Compact(maxWhyLines, maxWhyLineRunes)' 'return red.why(w).Compact(maxWhyLines, 0)' 'AV10 [N3] Handle không cắt dòng dài'
m $D 'utf8.RuneCountInString(l) <= max {' 'utf8.RuneCountInString(l) < max {' 'DV5 [N3] dòng đúng bằng trần bị cắt'
m $D 'string([]rune(l)[:max])' 'string([]rune(l)[:max-1])' 'DV6 [N3] dòng bị cắt ngắn hơn trần một ký tự'
m $D 'if max <= 0 || utf8.RuneCountInString(l) <= max {' 'if utf8.RuneCountInString(l) <= max {' 'DV7 [N3] maxRunes <= 0 không còn nghĩa là không cắt'
m $D '		c := count[l]
		l = clipRunes(l, maxRunes)' '		l = clipRunes(l, maxRunes)
		c := count[l]' 'DV8 [N3] số lần lặp đếm sau khi cắt'
m $D 'return w.apply(func(lines []string) []string { return compactLines(lines, limit, maxRunes) })' 'return w.apply(func(lines []string) []string { return compactLines(lines, limit, 0) })' 'DV9 [N3] Compact bỏ trần độ dài'
m $D 'return string([]rune(l)[:max]) + fmt.Sprintf("… (cắt: dòng dài quá %d ký tự)", max)' 'return l[:max] + fmt.Sprintf("… (cắt: dòng dài quá %d ký tự)", max)' 'DV10 [N3] cắt theo byte thay vì ký tự'
m $W 'n.Reject("%d ứng viên bị loại khi gỡ mơ hồ: %s (vết không nêu trạng thái phạm vi của từng ứng viên)", dropped, dropWhy)' 'n.Reject("%d ứng viên bị loại khi gỡ mơ hồ: %s (vết không kể từng ứng viên)", dropped, dropWhy)' 'WV46 [N4] chữ hứa không kể từng ứng viên (gỡ mơ hồ)'
m $W 'n.Reject("%d ứng viên bị loại khỏi câu hỏi lại: %s (vết không nêu trạng thái phạm vi của từng ứng viên)", len(byWhy[why]), why)' 'n.Reject("%d ứng viên bị loại khỏi câu hỏi lại: %s (vết không kể từng ứng viên)", len(byWhy[why]), why)' 'WV47 [N4] chữ hứa không kể từng ứng viên (câu hỏi lại)'
# LW: bốn câu ghi giải thích mà bộ sinh LV bỏ qua (hai câu viết trên nhiều dòng, hai câu trùng chữ nhau) — xoá từng câu cũng phải làm một test đỏ
m $W 'n.Reason("cụm %s (khoá so khớp %s) khớp %d thực thể: %s — không chọn bừa"' '_ = fmt.Sprintf("cụm %s (khoá so khớp %s) khớp %d thực thể: %s — không chọn bừa"' 'LW1 [bỏ dòng giải thích nhiều dòng] why.go: cụm mơ hồ của S3'
m $W 'n.Reason("%sgói ngữ cảnh: %d bằng chứng' '_ = fmt.Sprintf("%sgói ngữ cảnh: %d bằng chứng' 'LW2 [bỏ dòng giải thích nhiều dòng] why.go: gói ngữ cảnh của S7'
m $W 'n.Reason("%s", inputs)
			n.Reason("chỉ trả số kèm câu giữ phán quyết (confirm_withheld)")' 'if false {
			n.Reason("%s", inputs)
		}
			n.Reason("chỉ trả số kèm câu giữ phán quyết (confirm_withheld)")' 'LW3 [bỏ dòng giải thích trùng chữ] why.go: đầu vào của cổng khi giữ phán quyết'
m $W 'n.Reason("%s", inputs)
		other := "confirm_no"' 'if false {
		n.Reason("%s", inputs)
	}
		other := "confirm_no"' 'LW4 [bỏ dòng giải thích trùng chữ] why.go: đầu vào của cổng khi phán quyết'
# Sinh tự động từ mã hiện tại (scripts/dot-bien-vet-sinh.py): cộng 1 vào từng số in ra (NV), bỏ từng contracts.Quote (QV), bỏ từng câu ghi giải thích (LV) — để không
# dòng giải thích nào thiếu test khoá. Tắt bằng DOT_BIEN_SINH=0 khi chỉ cần các dòng viết tay ở trên (họ sinh tự động chiếm hơn một nửa thời gian của khối này).
if [ "${DOT_BIEN_SINH:-1}" != 0 ]; then
	sinh=$(python3 -I scripts/dot-bien-vet-sinh.py) || { echo "scripts/dot-bien-vet-sinh.py thất bại hoặc sinh quá ít đột biến" >&2; khac=$((khac + 1)); sinh=""; }
	eval "$sinh"
fi
# ---- hết R2-03 vòng 3

# ---- C8–C11 vòng 2 (soát độc lập C8–C11, 23/10). C9 vòng 2: bộ che khoá / tên thực thể ngoài phạm vi ở một chỗ (core/pipeline/scope_redact.go);
# test bắt nằm ở core/pipeline (TestScopeRedactorDungRieng, TestNewScopeRedactorKhoaVaTen), observability/traces (dòng S3 / S4 của kỹ sư PRJ-A) và
# tenants/construction (TestVetKhongLoThucTheNgoaiPhamVi quét toàn bản ghi). C11-S3: giải thích không được sửa dữ liệu dùng chung với Decision.
export PKGS="./core/pipeline/ ./observability/traces/ ./tenants/construction/"
SR=core/pipeline/scope_redact.go
m $SR 'if a.RT.Policy.InScope(pr, nil) {' 'if true {' 'XV47 [C9 vòng 2] bộ che tắt với mọi người hỏi'
m $SR 'for _, t := range []string{r.Key, r.ID, r.Name} {' 'for _, t := range []string{r.Name} {' 'XV48 [C9 vòng 2] không bao giờ che khoá / mã'
m $SR 'for _, t := range []string{r.Key, r.ID, r.Name} {' 'for _, t := range []string{r.Key, r.ID} {' 'XV49 [C9 vòng 2] không che tên'
m $SR 'strings.Join(append(keep, hidden...), "|")' 'strings.Join(append(hidden, keep...), "|")' 'XV50 [C9 vòng 2] chỗ che đứng đầu "?a|b" (thứ tự gợi khoá)'
m $SR 'return red.why(w).Compact(maxWhyLines, maxWhyLineRunes)' 'return w.Compact(maxWhyLines, maxWhyLineRunes)' 'XV51 [C9 vòng 2] không che phần giải thích'
m $ASK 'red.decision(&d)' '_ = d' 'XV52 [C9 vòng 2] không che Decision.Entities / Slots'
m $W 'for _, e := range res.Entities {' 'for i, e := range res.Entities { res.Entities[i].ID += "!"' 'XV45 [soát C8–C11 C11-S3, M3] explainResolve sửa mã thực thể của Resolution dùng chung với Decision'
m $W 'for _, sp := range in.Spans {' 'for i, sp := range in.Spans { in.Spans[i].Raw += "!"' 'XV46 [soát C8–C11 C11-S3, M4] explainNormalize sửa đoạn của NormalizedInput dùng chung với Decision'
# ---- hết C8–C11 vòng 2

fresh >/dev/null 2>&1; psql -q -X "$EAIS_PG_ADMIN_DSN" -c "DROP DATABASE IF EXISTS $DB_DOT_BIEN WITH (FORCE)"
echo "TỔNG: bị bắt $bat · sống sót $song · khác $khac"
[ "$song" -eq 0 ] && [ "$khac" -eq 0 ]
