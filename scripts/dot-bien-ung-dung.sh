#!/usr/bin/env bash
# Danh mục đột biến của ứng dụng (làn 0, đơn vị U0.2): cmd/eais, bảng vết (observability/traces/render.go), Ready của kho, cổng SKIP và
# workflow CI. Mỗi dòng bẻ MỘT chỗ của mã rồi đòi MỘT test (hoặc smoke) đỏ — chứng minh cổng đó đỏ được (CLAUDE.md §1).
#   m  <tệp> <chuỗi gốc> <chuỗi đột biến> <nhãn>   chạy `go test` ở PKGS (đột biến của mã)
#   ms <tệp> <chuỗi gốc> <chuỗi đột biến> <nhãn>   chạy scripts/chay-thu.sh (smoke E2E) trên cây đã đột biến; đòi smoke ĐỎ
# Kỳ vọng: mọi dòng in "BỊ BẮT"; cuối cùng in tổng và thoát 0 chỉ khi không đột biến nào sống sót. Mã đổi thì chuỗi gốc có thể không còn
# khớp (rc=2, "xuất hiện 0 lần"): sửa chuỗi theo mã mới rồi chạy lại — đừng xoá dòng cho xanh.
#
# CẦN PG RIÊNG và hai biến môi trường đặt rõ (không có giá trị mặc định, để không vô tình chạy vào PG đang phục vụ việc khác):
#   EAIS_PG_ADMIN_DSN=postgres://postgres@127.0.0.1:<cổng>/eais?sslmode=disable
#   EAIS_PG_DSN=postgres://eais_app@127.0.0.1:<cổng>/eais?sslmode=disable
# Test của cmd/eais ghi dòng vào PG đó bằng tenant riêng của lượt chạy (TestMain dọn bằng vai quản trị); các dòng `ms` còn nạp tenant thật
# của pack (seed 42) vào PG đó — nên chỉ trỏ vào PG của riêng bạn. Dòng `ms` nghe ở SMOKE_ADDR (mặc định 127.0.0.1:28085).
cd "$(dirname "$0")/.." || exit 1
: "${EAIS_PG_ADMIN_DSN:?đặt EAIS_PG_ADMIN_DSN (PG riêng của bạn)}"
: "${EAIS_PG_DSN:?đặt EAIS_PG_DSN (PG riêng của bạn)}"
export EAIS_PG_ADMIN_DSN EAIS_PG_DSN
SMOKE_ADDR=${SMOKE_ADDR:-127.0.0.1:28085}
bat=0 song=0 khac=0
m() {
	local res rc
	res=$(scripts/mutate.sh "$1" "$2" "$3" "$4"); rc=$?
	printf '%s\nrc=%s\n' "$res" "$rc"
	case $rc in 0) bat=$((bat + 1)) ;; 3) song=$((song + 1)) ;; *) khac=$((khac + 1)) ;; esac
}

restore_f='' restore_b=''
restore() { [ -n "$restore_f" ] && [ -f "$restore_b" ] && cp "$restore_b" "$restore_f"; rm -f "$restore_b"; restore_f='' restore_b=''; }
trap restore EXIT INT TERM
ms() {
	local f=$1 from=$2 to=$3 label=$4 before out rc
	restore_b=$(mktemp) restore_f=$f
	cp "$f" "$restore_b"
	before=$(sha256sum <"$f")
	if ! python3 -I - "$f" "$from" "$to" <<'PY'
import sys
p, a, b = sys.argv[1], sys.argv[2], sys.argv[3]
s = open(p, encoding="utf-8").read()
n = s.count(a)
if n != 1:
    sys.exit(f"chuỗi gốc xuất hiện {n} lần, cần đúng 1")
open(p, "w", encoding="utf-8").write(s.replace(a, b))
PY
	then
		restore
		echo "[$label] không áp được đột biến"; echo "rc=2"; khac=$((khac + 1)); return
	fi
	out=$(EAIS_ADDR=$SMOKE_ADDR scripts/chay-thu.sh 2>&1); rc=$?
	restore
	[ "$before" = "$(sha256sum <"$f")" ] || { echo "KHÔI PHỤC LỖI: $f"; exit 1; }
	if [ $rc -ne 0 ] && grep -q '^FAIL' <<<"$out"; then
		echo "[$label] BỊ BẮT (smoke): $(grep '^FAIL' <<<"$out" | head -3 | cut -c1-90 | tr '\n' ';')"; echo "rc=0"; bat=$((bat + 1))
	elif [ $rc -eq 0 ]; then
		echo "[$label] SỐNG SÓT — smoke vẫn xanh"; echo "rc=3"; song=$((song + 1))
	else
		echo "[$label] smoke đỏ nhưng không có dòng FAIL (build / khởi động hỏng?): $(tail -3 <<<"$out" | tr '\n' ' ')"; echo "rc=4"; khac=$((khac + 1))
	fi
}

S=cmd/eais/server.go A=cmd/eais/app.go U=cmd/eais/auth.go M=cmd/eais/main.go C=cmd/eais/cli_trace.go
export PKGS=./cmd/eais/

# --- xác thực, thân, tenant (server.go) ---
m $S 'if pr, found := s.principals[tok]; found {' 'if pr, found := s.principals[tok]; found || true {' 'AV1 token lạ vẫn qua'
m $S 'dec.DisallowUnknownFields()' '_ = 0' 'AV2 thân mang tenant / trường lạ được nhận'
m $S 'if _, err := dec.Token(); !errors.Is(err, io.EOF) {' 'if _, err := dec.Token(); false && !errors.Is(err, io.EOF) {' 'AV3 nội dung thừa sau JSON được nhận'
m $S 's.cnt.Unauthenticated.Add(1)' '_ = 0' 'AV4 lượt chưa xác thực không được đếm'
m $S 'if !loopbackHost(r.Host) {' 'if false {' 'AV5 Host lạ qua (DNS rebinding)'
m $S 'if err := s.st.Ping(ctx); err != nil {' 'if err := s.st.Ping(ctx); err != nil && false {' 'AV6 healthz không báo kho hỏng'
# --- cô lập tenant và quyền đọc vết ---
m $S 'ctx := contracts.WithTenant(r.Context(), pr.TenantID)' 'ctx := contracts.WithTenant(r.Context(), contracts.TenantID(s.app.tenantID()))' 'TV1 đọc vết bằng tenant của runtime thay vì của token'
m $S 'if err == nil && rec.Decision.Principal != pr.ID && !s.fullAccess(pr) {' 'if false {' 'TV2 ai cùng tenant cũng đọc được vết của người khác'
m $U 'if len(have) == len(all) {' 'if len(have) > 0 {' 'TV3 vai có một capability đã tính là toàn quyền'
m $U 'if !ok || rd.Scope != "all" {' 'if !ok {' 'TV4 phạm vi attribute tính là toàn quyền'
m $S 'return string(pr.TenantID) == s.app.tenantID() && fullAccess(s.app.rt.Policy, pr)' 'return fullAccess(s.app.rt.Policy, pr)' 'TV5 vai trùng tên ở tenant khác tính là toàn quyền'
# --- nạp lại ---
m $S 'if !s.fullAccess(pr) {' 'if false {' 'IG1 vai không toàn quyền nạp lại được'
m $S '(len(t) != 0 && string(t) != "{}")' '(false && len(t) != 0 && string(t) != "{}")' 'IG2 thân nạp lại mang đường dẫn / tham số được nhận'
m $A 'sum := sha256.Sum256([]byte(dataVersion + "\n" + packVersion + "\n" + codeVersion))' 'sum := sha256.Sum256([]byte(dataVersion + "\n" + packVersion + "\n" + codeVersion + time.Now().String()))' 'IG3 mã lần nạp không tất định (nạp lại nhân đôi cách ly)'
m $A 'if rep.NewQuarantine, err = a.st.SaveQuarantine(tctx, rep.LoadID, recs); err != nil {' 'if rep.NewQuarantine, err = 0, error(nil); err != nil {' 'IG4 bản ghi cách ly không được lưu'
m $A 'a.setAsk(before)' '_ = before' 'IG5 nạp lại xong không cập nhật phiên bản dữ liệu của vết'
m $A 'if after != before {' 'if after != before && false {' 'IG6 dữ liệu đổi giữa chừng không bị phát hiện'
m $A 'lines = append(lines, filepath.ToSlash(rel)+"\x00"+hex.EncodeToString(h.Sum(nil)))' 'lines = append(lines, filepath.ToSlash(rel)[:0]+hex.EncodeToString(h.Sum(nil)))' 'IG7 phiên bản dữ liệu bỏ qua đường dẫn tệp'
# --- vết của mọi lượt ---
m $S 'if s.persist(r.Context(), pr, rec, acc) == nil {' 'if true {' 'VT1 lượt lỗi trả trace_id mà không lưu vết'
m $A 'return pgstore.UsageRecord{TraceID: traceID, Model: model, Calls: a.calls, Tokens: a.tokens}' 'return pgstore.UsageRecord{TraceID: traceID, Model: model, Calls: 0, Tokens: a.tokens}' 'VT2 usage ghi 0 lời gọi dù có gọi model'
m $S 'for attempt := 1; attempt <= 3; attempt++ {' 'for attempt := 1; attempt <= 1; attempt++ {' 'VT3 không thử lại khi lỗi hạ tầng tạm thời'
m $S 'if err = fn(); err == nil || !contracts.Retryable(err) {' 'if err = fn(); err == nil {' 'VT4 thử lại cả lỗi không tạm thời'
m $S 'sctx, cancel := context.WithTimeout(context.WithoutCancel(contracts.WithTenant(ctx, pr.TenantID)), saveTimeout)' 'sctx, cancel := context.WithTimeout(contracts.WithTenant(ctx, pr.TenantID), saveTimeout)' 'VT5 client ngắt kết nối làm mất vết'
m $S 'if p := recover(); p != nil {' 'if p := any(nil); p != nil {' 'VT6 panic trong luồng hỏi không có vết'
m $S 's.cnt.TraceSaveFailed.Add(1)' '_ = 0' 'VT7 lỗi lưu vết không được đếm'
m $S 'return "lỗi hệ thống; tra cứu vết bằng trace_id (chi tiết nội bộ không công bố qua API)"' 'return msg' 'VT8 lỗi nội bộ lộ ra qua API'
m $S '	writeJSON(w, http.StatusOK, out.Response)' '	out.Response.Answer += " "; writeJSON(w, http.StatusOK, out.Response)' 'AN1 HTTP đổi câu trả lời (thêm một dấu cách)'
m $S 'if fault == contracts.FaultInput || fault == contracts.FaultPolicy { // bản ghi do CHÍNH server dựng bị từ chối: lỗi của server, không phải của client' 'if false { // bản ghi do CHÍNH server dựng bị từ chối: lỗi của server, không phải của client' 'VT9 lỗi lưu vết của server đổ cho client (400 / 403)'
# --- dev, không gọi model, CLI ---
m $U 'if ip := net.ParseIP(host); ip != nil && ip.IsLoopback() {' 'if ip := net.ParseIP(host); ip != nil {' 'DV1 địa chỉ nghe không phải loopback được nhận'
m $M 'if !*dev {' 'if !*dev && false {' 'DV2 serve chạy không cần -dev'
m $M 'if tcp, ok := ln.Addr().(*net.TCPAddr); !ok || !tcp.IP.IsLoopback() {' 'if tcp, ok := ln.Addr().(*net.TCPAddr); ok && tcp != nil && false {' 'DV3 không kiểm địa chỉ thực sự gắn'
m $A 'inner: &models.Extractive{}' 'inner: func() models.Gateway { g, _ := models.FromConfig(""); return g }()' 'MD1 đường tới model ngoài Extractive'
m $M 'if err := st.Ready(ctx); err != nil {' 'if err := st.Ready(ctx); err != nil && false {' 'CL1 kho chưa migrate vẫn chạy'
m $C 'if _, differ := traces.FirstDifferentStep(a, b); differ {' 'if _, differ := traces.FirstDifferentStep(a, b); differ && false {' 'CL2 trace diff luôn thoát 0'
m $C 'if len(recs) == 0 {' 'if len(recs) < 0 {' 'CL3 mã lần nạp sai vẫn thoát 0'
m $C 'if a.Decision.Question != b.Decision.Question {' 'if false {' 'CL4 diff hai câu khác nhau không cảnh báo'

# --- bảng vết ---
export PKGS=./observability/traces/
R=observability/traces/render.go
m $R 'marks = append(marks, MarkDeviant)' '_ = 0' 'BV1 không đánh dấu bước LỆCH'
m $R 'marks = append(marks, MarkError)' '_ = 0' 'BV2 không đánh dấu bước LỖI'
m $R 'marks = append(marks, MarkBadVerdict)' '_ = 0' 'BV3 không đánh dấu PHÁN QUYẾT HỎNG'
m $R 'marks = append(marks, MarkGap)' '_ = 0' 'BV4 không đánh dấu chỗ TRỐNG'
m $R 'if first < 0 && (s.Status == StatusDeviant || s.Status == StatusError || s.Fault != "") {' 'if false {' 'BV5 không chỉ bước lệch đầu tiên'
m $R 'if s.Stage == StageValidate && strings.HasPrefix(r.Decision.Verdict, "false|") {' 'if strings.HasPrefix(r.Decision.Verdict, "false|") {' 'BV6 dấu phán quyết hỏng gắn vào mọi bước'

# --- Ready của kho ---
export PKGS=./platform/pgstore/
m platform/pgstore/ready.go 'if !exists {' 'if false {' 'RD1 bảng thiếu không được báo'
m platform/pgstore/ready.go 'if !ok {' 'if false {' 'RD2 thiếu quyền không được báo'

m platform/pgstore/migrate.go 'if _, err := conn.Exec(ctx, "SELECT pg_advisory_lock($1)", migrateLockKey); err != nil {' 'if _, err := conn.Exec(ctx, "SELECT $1::bigint", migrateLockKey); err != nil {' 'MG1 Migrate đồng thời không xếp hàng (đua CREATE TABLE)'

# --- cổng SKIP và workflow CI ---
export PKGS=./cmd/eais/
m scripts/kiem-skip-test.py 'if ngoai or thua:' 'if False:' 'CI1 SKIP không làm cổng đỏ'
m scripts/kiem-skip-test.py 'if tong == 0:' 'if tong < 0:' 'CI2 đầu vào rỗng được coi là "không SKIP"'
m scripts/kiem-skip-test.py 'ngoai = [s for s in skip if (s[0], s[1]) not in cho_phep]' 'ngoai = [s for s in skip if True]' 'CI3 danh sách cho phép không có tác dụng'
m .github/workflows/ci.yml 'postgres:17@sha256:f4c66b820c6f974249089d3d16d86a3698eae11e8746eb6644b2271031e91232' 'postgres:17@sha256:f4c66b820c6f974249089d3d16d86a3698eae11e8746eb6644b2271031e91233' 'CI4 CI và compose ghim hai digest khác nhau'
m .github/workflows/ci.yml 'actions/checkout@v4.2.2' 'actions/checkout@v4' 'CI5 action không ghim tag đầy đủ'
m .github/workflows/ci.yml 'python3 -I scripts/kiem-skip-test.py test.json' 'true' 'CI6 bước cổng SKIP bị bỏ'
m .github/workflows/ci.yml 'permissions:
  contents: read' 'permissions:
  contents: write' 'CI7 CI có quyền ghi'

# --- smoke E2E bằng curl: cổng này cũng phải đỏ được ---
ms $S 'dec.DisallowUnknownFields()' '_ = 0' 'SM1 smoke: thân mang tenant được nhận'
ms $S 'if pr, found := s.principals[tok]; found {' 'if pr, found := s.principals[tok]; found || true {' 'SM2 smoke: token lạ vẫn qua'
ms $S 'if !s.fullAccess(pr) {' 'if false {' 'SM3 smoke: vai không toàn quyền nạp lại được'
ms $S 'if s.persist(r.Context(), pr, rec, acc) == nil {' 'if true {' 'SM4 smoke: lượt lỗi không có vết'
ms $S 'ctx := contracts.WithTenant(r.Context(), pr.TenantID)' 'ctx := contracts.WithTenant(r.Context(), contracts.TenantID(s.app.tenantID()))' 'SM5 smoke: tenant khác đọc được vết'
ms $A 'sum := sha256.Sum256([]byte(dataVersion + "\n" + packVersion + "\n" + codeVersion))' 'sum := sha256.Sum256([]byte(dataVersion + "\n" + packVersion + "\n" + codeVersion + time.Now().String()))' 'SM6 smoke: nạp lại nhân đôi cách ly'

echo "TỔNG: $bat bị bắt, $song sống sót, $khac lỗi khác"
[ "$song" -eq 0 ] && [ "$khac" -eq 0 ]
