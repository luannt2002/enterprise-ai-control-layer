#!/usr/bin/env bash
# Danh mục đột biến của ứng dụng (làn 0, đơn vị U0.2): cmd/eais, bảng vết (observability/traces/render.go), Ready của kho, cổng SKIP và
# workflow CI. Mỗi dòng bẻ MỘT chỗ của mã rồi đòi MỘT test (hoặc smoke) đỏ — chứng minh cổng đó đỏ được (CLAUDE.md §1).
#   m  <tệp> <chuỗi gốc> <chuỗi đột biến> <nhãn>   chạy `go test` ở PKGS (đột biến của mã)
#   ms <tệp> <chuỗi gốc> <chuỗi đột biến> <nhãn>   chạy scripts/chay-thu.sh (smoke E2E) trên cây đã đột biến; đòi smoke ĐỎ
#   sc <kịch bản> <nhãn> [<tệp> <gốc> <đột biến>]  kịch bản của CHÍNH scripts/chay-thu.sh (cổng bị chiếm, server chết giữa lúc smoke, PG chạy
#                                                  sẵn không bị gỡ và không bị ghi đè khi không có EAIS_REUSE_PG=1): không
#                                                  đột biến thì đòi script nguyên bản cư xử đúng ("ĐẠT"); có đột biến thì đòi kịch bản phát hiện
# Kỳ vọng: mọi dòng in "BỊ BẮT" hoặc "ĐẠT"; cuối cùng in tổng và thoát 0 chỉ khi không đột biến nào sống sót. Mã đổi thì chuỗi gốc có thể
# không còn khớp (rc=2, "xuất hiện 0 lần"): sửa chuỗi theo mã mới rồi chạy lại — đừng xoá dòng cho xanh.
# Chạy một phần: DOT_BIEN_LOC='<regex>' chỉ chạy các dòng có nhãn khớp (vd DOT_BIEN_LOC='^(CI|SQ)').
#
# CẦN PG RIÊNG và hai biến môi trường đặt rõ (không có giá trị mặc định, để không vô tình chạy vào PG đang phục vụ việc khác):
#   EAIS_PG_ADMIN_DSN=postgres://postgres@127.0.0.1:<cổng>/eais?sslmode=disable
#   EAIS_PG_DSN=postgres://eais_app@127.0.0.1:<cổng>/eais?sslmode=disable
# Test của cmd/eais ghi dòng vào PG đó bằng tenant riêng của lượt chạy (TestMain dọn bằng vai quản trị); các dòng `ms` còn nạp tenant thật
# của pack (seed 42) vào PG đó — nên chỉ trỏ vào PG của riêng bạn. Dòng `ms` nghe ở SMOKE_ADDR (mặc định 127.0.0.1:28085). Hai kịch bản
# `sc` dựng thứ riêng của chúng: SQUAT_ADDR (127.0.0.1:28086) là cổng bị một server giả chiếm; kịch bản pgkeep dựng một PG tạm tên
# KEEP_NAME (eais-giu-pg) ở cổng KEEP_PORT (25449), chạy server ở KEEP_ADDR (127.0.0.1:28087), rồi gỡ đúng PG tạm đó; kịch bản pgrefuse
# cũng dựng PG tạm KEEP_NAME ở KEEP_PORT (do CHÍNH scripts/chay-thu.sh dựng ở lần 1, server ở cổng hệ điều hành chọn) rồi gỡ nó; riêng
# một pha của nó chạy script với cùng KEEP_NAME nhưng một cổng TRỐNG khác do hệ điều hành chọn (script đúng thì không dựng gì ở cổng đó).
# Mỗi kịch bản dùng cùng KEEP_NAME tuần tự, không song song. Đổi các biến này nếu trùng việc khác trên máy.
cd "$(dirname "$0")/.." || exit 1
: "${EAIS_PG_ADMIN_DSN:?đặt EAIS_PG_ADMIN_DSN (PG riêng của bạn)}"
: "${EAIS_PG_DSN:?đặt EAIS_PG_DSN (PG riêng của bạn)}"
export EAIS_PG_ADMIN_DSN EAIS_PG_DSN
SMOKE_ADDR=${SMOKE_ADDR:-127.0.0.1:28085}
SQUAT_ADDR=${SQUAT_ADDR:-127.0.0.1:28086}
KEEP_ADDR=${KEEP_ADDR:-127.0.0.1:28087}
KEEP_PORT=${KEEP_PORT:-25449}
KEEP_NAME=${KEEP_NAME:-eais-giu-pg}
LOC=${DOT_BIEN_LOC:-}
bat=0 song=0 khac=0 dat=0 why=''
skip() { [ -n "$LOC" ] && ! [[ $1 =~ $LOC ]]; } # 0 (đúng) khi dòng có nhãn $1 bị DOT_BIEN_LOC lọc bỏ
m() {
	local res rc
	skip "$4" && return
	res=$(scripts/mutate.sh "$1" "$2" "$3" "$4"); rc=$?
	printf '%s\nrc=%s\n' "$res" "$rc"
	case $rc in 0) bat=$((bat + 1)) ;; 3) song=$((song + 1)) ;; *) khac=$((khac + 1)) ;; esac
}

restore_f='' restore_b=''
restore() { [ -n "$restore_f" ] && [ -f "$restore_b" ] && cp "$restore_b" "$restore_f"; rm -f "$restore_b"; restore_f='' restore_b=''; }
trap restore EXIT INT TERM
# apply_mut <tệp> <chuỗi gốc> <chuỗi đột biến>: sao lưu rồi áp đột biến (chuỗi gốc phải xuất hiện đúng 1 lần); `restore` khôi phục.
apply_mut() {
	restore_b=$(mktemp) restore_f=$1
	cp "$1" "$restore_b"
	python3 -I - "$1" "$2" "$3" <<'PY'
import sys
p, a, b = sys.argv[1], sys.argv[2], sys.argv[3]
s = open(p, encoding="utf-8").read()
n = s.count(a)
if n != 1:
    sys.exit(f"chuỗi gốc xuất hiện {n} lần, cần đúng 1")
open(p, "w", encoding="utf-8").write(s.replace(a, b))
PY
}
ms() {
	local f=$1 from=$2 to=$3 label=$4 before out rc
	skip "$label" && return
	before=$(sha256sum <"$f")
	if ! apply_mut "$f" "$from" "$to"; then
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

# Kịch bản của scripts/chay-thu.sh: mỗi hàm chạy script THẬT, đặt `why` (rỗng nếu script cư xử đúng) và trả 0 nếu đúng, 1 nếu sai.
scen_squat() { # cổng EAIS_ADDR đã bị một server khác chiếm và trả /healthz 200: script phải báo "server thoát sớm", thoát khác 0, và KHÔNG chạy smoke vào kẻ chiếm cổng
	local pid out rc
	python3 -I -c 'import http.server, sys
class H(http.server.BaseHTTPRequestHandler):
    def do_GET(self):
        self.send_response(200); self.end_headers(); self.wfile.write(b"{\"status\":\"ok\"}")
    def log_message(self, *a): pass
http.server.HTTPServer(("127.0.0.1", int(sys.argv[1])), H).serve_forever()' "${SQUAT_ADDR##*:}" &
	pid=$!
	for _ in $(seq 1 40); do curl -fsS -m 1 "http://$SQUAT_ADDR/healthz" >/dev/null 2>&1 && break; sleep 0.25; done
	out=$(EAIS_ADDR=$SQUAT_ADDR scripts/chay-thu.sh 2>&1); rc=$?
	kill "$pid" 2>/dev/null; wait "$pid" 2>/dev/null
	why=''
	[ "$rc" -ne 0 ] || why+="script thoát 0 dù cổng bị chiếm; "
	grep -q '^smoke E2E:' <<<"$out" && why+="smoke đã chạy vào kẻ chiếm cổng ($(grep '^smoke E2E:' <<<"$out" | head -1)); "
	grep -q 'server thoát sớm' <<<"$out" || why+="không báo 'server thoát sớm'; "
	[ -z "$why" ]
}
scen_pgkeep() { # container PG tên đó ĐÃ chạy từ trước + EAIS_REUSE_PG=1: script dùng lại và KHÔNG BAO GIỜ gỡ nó, kể cả khi đặt EAIS_PG_DOWN=1 (PG tạm riêng của phép thử)
	local out rc running
	if ! EAIS_PG_CONTAINER=$KEEP_NAME EAIS_PG_PORT=$KEEP_PORT docker compose -p "$KEEP_NAME" -f deploy/docker-compose.yml up -d --wait >/dev/null 2>&1; then
		why="không dựng được PG tạm $KEEP_NAME ở cổng $KEEP_PORT"; return 1
	fi
	out=$(env -u EAIS_PG_ADMIN_DSN -u EAIS_PG_DSN EAIS_REUSE_PG=1 EAIS_PG_DOWN=1 EAIS_COMPOSE_PROJECT=$KEEP_NAME EAIS_PG_CONTAINER=$KEEP_NAME EAIS_PG_PORT=$KEEP_PORT \
		EAIS_ADDR=$KEEP_ADDR scripts/chay-thu.sh 2>&1); rc=$?
	running=$(docker ps -q --filter "name=^/${KEEP_NAME}\$" --filter status=running)
	EAIS_PG_CONTAINER=$KEEP_NAME EAIS_PG_PORT=$KEEP_PORT docker compose -p "$KEEP_NAME" -f deploy/docker-compose.yml down >/dev/null 2>&1
	why=''
	[ "$rc" -eq 0 ] || why+="script thoát $rc ($(tail -2 <<<"$out" | tr '\n' ' ')); "
	grep -q 'ĐÃ chạy từ trước' <<<"$out" || why+="không nhận ra container đã chạy sẵn; "
	[ -n "$running" ] || why+="container $KEEP_NAME đã bị GỠ dù chạy từ trước; "
	[ -z "$why" ]
}
scen_pgrefuse() { # R2-04 trên container RIÊNG: lần 1 chỗ trống ⇒ script tự dựng PG (xong còn chạy) · lần 2 mặc định (PG đang chạy; rồi chỉ đổi cổng sang một cổng trống, R3-01; rồi PG tạm dừng) ⇒ TỪ CHỐI, thoát 2, KHÔNG ghi gì, container KHÔNG bị dựng lại · lần 3 EAIS_REUSE_PG=1 ⇒ dùng lại; cuối cùng gỡ đúng PG tạm
	local out1 out2 out2c out2p out3 rc1 rc2 rc2c rc2p rc3 mark2 mark2c mark2p running id1 id2c port2
	local run=(env -u EAIS_PG_ADMIN_DSN -u EAIS_PG_DSN -u EAIS_REUSE_PG -u EAIS_PG_DOWN EAIS_COMPOSE_PROJECT=$KEEP_NAME EAIS_PG_CONTAINER=$KEEP_NAME
		EAIS_PG_PORT=$KEEP_PORT EAIS_ADDR=127.0.0.1:0)
	psql_keep() { docker exec "$KEEP_NAME" psql -U postgres -d eais -Atc "$1" 2>/dev/null; }
	if [ -n "$(docker ps -aq --filter "name=^/${KEEP_NAME}\$")" ]; then why="đã có container $KEEP_NAME — đặt KEEP_NAME khác"; return 1; fi
	# Cổng TRỐNG thứ hai cho pha "chỉ đổi cổng", chọn trước khi dựng gì: giá trị rỗng mà lọt vào script thì EAIS_PG_PORT rỗng = cổng mặc định 25432 của PG thật.
	port2=$(python3 -I -c 'import socket; s = socket.socket(); s.bind(("127.0.0.1", 0)); print(s.getsockname()[1])' 2>/dev/null)
	case $port2 in '' | *[!0-9]*) why="không chọn được cổng trống thứ hai (python3)"; return 1 ;; esac
	out1=$(timeout 90 "${run[@]}" scripts/chay-thu.sh 2>&1); rc1=$?
	# Dấu hiệu "dữ liệu của người khác": sửa một fact của tenant đã nạp. Script nào ghi đè PG này sẽ nạp lại giá trị seed và làm dấu hiệu mất.
	psql_keep "update facts set value = 999999 where tenant_id = 'abc-infra' and id = (select min(id) from facts where tenant_id = 'abc-infra')" >/dev/null
	out2=$(timeout 90 "${run[@]}" scripts/chay-thu.sh 2>&1); rc2=$?
	mark2=$(psql_keep "select count(*) from facts where value = 999999")
	# R3-01: cùng dự án, cùng tên container, chỉ đổi cổng sang một cổng TRỐNG. Cổng trống nên phép thăm dò cổng không bắt được: hàng rào DUY NHẤT là bộ lọc
	# tên container của `docker ps`. Hỏng nó thì script chạy `docker compose up` cùng dự án với cổng khác và compose TẠO LẠI container (mất dữ liệu tmpfs).
	id1=$(docker ps -q --filter "name=^/${KEEP_NAME}\$")
	out2c=$(timeout 90 "${run[@]}" EAIS_PG_PORT=$port2 scripts/chay-thu.sh 2>&1); rc2c=$?
	id2c=$(docker ps -q --filter "name=^/${KEEP_NAME}\$")
	mark2c=$(psql_keep "select count(*) from facts where value = 999999")
	docker pause "$KEEP_NAME" >/dev/null 2>&1
	out2p=$(timeout 90 "${run[@]}" scripts/chay-thu.sh 2>&1); rc2p=$?
	docker unpause "$KEEP_NAME" >/dev/null 2>&1
	mark2p=$(psql_keep "select count(*) from facts where value = 999999")
	out3=$(timeout 90 "${run[@]}" EAIS_REUSE_PG=1 scripts/chay-thu.sh 2>&1); rc3=$?
	running=$(docker ps -q --filter "name=^/${KEEP_NAME}\$")
	EAIS_PG_CONTAINER=$KEEP_NAME EAIS_PG_PORT=$KEEP_PORT docker compose -p "$KEEP_NAME" -f deploy/docker-compose.yml down >/dev/null 2>&1
	why=''
	[ "$rc1" -eq 0 ] || why+="lần 1 (chỗ trống) thoát $rc1 ($(tail -2 <<<"$out1" | tr '\n' ' ')); "
	[ "$rc2" -eq 2 ] || why+="lần 2 (PG đang chạy, mặc định) thoát $rc2 thay vì 2; "
	grep -q 'TỪ CHỐI' <<<"$out2" || why+="lần 2 không nói TỪ CHỐI; "
	[ "$mark2" = 1 ] || why+="lần 2 đã GHI vào PG đang chạy (dấu hiệu mất); "
	[ "$rc2c" -eq 2 ] || why+="lần 2 chỉ đổi cổng sang $port2 (cùng dự án, cùng tên container) thoát $rc2c thay vì 2; "
	grep -q "container $KEEP_NAME ĐÃ chạy từ trước" <<<"$out2c" || why+="lần 2 chỉ đổi cổng không từ chối vì container (bộ lọc tên container hỏng?); "
	{ [ -n "$id1" ] && [ "$id1" = "$id2c" ]; } || why+="lần 2 chỉ đổi cổng đã DỰNG LẠI container (mã ${id1:-không có} thành ${id2c:-không còn}); "
	[ "$mark2c" = 1 ] || why+="lần 2 chỉ đổi cổng đã GHI vào PG (dấu hiệu mất); "
	[ "$rc2p" -eq 2 ] || why+="lần 2 với container tạm dừng thoát $rc2p thay vì 2 (bỏ sót container paused?); "
	[ "$mark2p" = 1 ] || why+="lần 2 với container tạm dừng đã GHI vào PG; "
	[ "$rc3" -eq 0 ] || why+="lần 3 (EAIS_REUSE_PG=1) thoát $rc3 ($(tail -2 <<<"$out3" | tr '\n' ' ')); "
	grep -q 'dùng lại' <<<"$out3" || why+="lần 3 không báo là dùng lại; "
	[ -n "$running" ] || why+="PG tạm đã bị gỡ trước khi phép thử gỡ; "
	[ -z "$why" ]
}
scen_srvdie() { # server chết GIỮA lúc smoke (smoke giả giết nó rồi báo xanh): script phải thoát khác 0 và nói rõ, không được báo "smoke XANH"
	local fake out rc
	fake=$(mktemp)
	cat >"$fake" <<'FAKE'
#!/usr/bin/env bash
# smoke giả: giết server của script (tiến trình con tên eais của tiến trình cha) rồi báo xanh
kill -KILL $(pgrep -P $PPID -x eais) 2>/dev/null
sleep 0.5
echo "smoke E2E: 21 đạt, 0 hỏng"
exit 0
FAKE
	chmod +x "$fake"
	out=$(EAIS_SMOKE=$fake EAIS_ADDR=$KEEP_ADDR scripts/chay-thu.sh 2>&1); rc=$?
	rm -f "$fake"
	why=''
	[ "$rc" -ne 0 ] || why+="script thoát 0 dù server đã chết giữa lúc smoke; "
	grep -q 'đã thoát giữa chừng' <<<"$out" || why+="không báo server đã thoát giữa chừng; "
	grep -q 'smoke XANH' <<<"$out" && why+="vẫn in 'smoke XANH'; "
	[ -z "$why" ]
}
sc() {
	local scen=$1 label=$2 f=${3:-} before verdict
	skip "$label" && return
	why=''
	if [ -n "$f" ]; then
		before=$(sha256sum <"$f")
		if ! apply_mut "$f" "$4" "$5"; then
			restore
			echo "[$label] không áp được đột biến"; echo "rc=2"; khac=$((khac + 1)); return
		fi
	fi
	"$scen"; verdict=$?
	if [ -n "$f" ]; then
		restore
		[ "$before" = "$(sha256sum <"$f")" ] || { echo "KHÔI PHỤC LỖI: $f"; exit 1; }
	fi
	if [ -z "$f" ]; then
		if [ $verdict -eq 0 ]; then echo "[$label] ĐẠT — script nguyên bản cư xử đúng"; dat=$((dat + 1))
		else echo "[$label] script nguyên bản CƯ XỬ SAI: $why"; echo "rc=4"; khac=$((khac + 1)); fi
	elif [ $verdict -ne 0 ]; then
		echo "[$label] BỊ BẮT: $why"; echo "rc=0"; bat=$((bat + 1))
	else
		echo "[$label] SỐNG SÓT — kịch bản vẫn thấy script đúng"; echo "rc=3"; song=$((song + 1))
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
m $S 'maxAskBody    = 64 << 10' 'maxAskBody    = 64 << 22' 'AV7 trần thân /v1/ask nâng từ 64 KiB lên 256 MiB'
m $S 'maxAskBody    = 64 << 10' 'maxAskBody    = 64 << 9' 'AV8 trần thân /v1/ask hạ từ 64 KiB xuống 32 KiB'
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
m $S 'if !s.ingestMu.TryLock() {' 's.ingestMu.Lock(); if false {' 'IG8 hai lần nạp chồng nhau xếp hàng ngầm thay vì bị từ chối 409'
# --- vết của mọi lượt ---
m $S 'if s.persist(r.Context(), pr, rec, acc) == nil {' 'if true {' 'VT1 lượt lỗi trả trace_id mà không lưu vết'
m $A 'return pgstore.UsageRecord{TraceID: traceID, Model: model, Calls: a.calls, Tokens: a.tokens}' 'return pgstore.UsageRecord{TraceID: traceID, Model: model, Calls: 0, Tokens: a.tokens}' 'VT2 usage ghi 0 lời gọi dù có gọi model'
m $S 'for attempt := 1; attempt <= 3; attempt++ {' 'for attempt := 1; attempt <= 1; attempt++ {' 'VT3 không thử lại khi lỗi hạ tầng tạm thời'
m $S 'if err = fn(); err == nil || !contracts.Retryable(err) {' 'if err = fn(); err == nil {' 'VT4 thử lại cả lỗi không tạm thời'
m $S 'sctx, cancel := context.WithTimeout(context.WithoutCancel(contracts.WithTenant(ctx, pr.TenantID)), saveTimeout)' 'sctx, cancel := context.WithTimeout(contracts.WithTenant(ctx, pr.TenantID), saveTimeout)' 'VT5 client ngắt kết nối làm mất vết'
m $S 'if p := recover(); p != nil {' 'if p := any(nil); p != nil {' 'VT6 panic trong luồng hỏi không có vết'
m $S 's.cnt.TraceSaveFailed.Add(1)' '_ = 0' 'VT7 lỗi lưu vết không được đếm'
m $S 'return "lỗi hệ thống (chi tiết nội bộ không công bố qua API)"' 'return msg' 'VT8 lỗi nội bộ lộ ra qua API'
m $S '			body.Error += "; tra cứu vết bằng trace_id"' '			body.Error += ""' 'VT10 lượt lỗi có vết mà không chỉ cách tra cứu bằng trace_id'
m $S '	} else if !isCallerFault(rec.Fault) {
		body.Error += "; vết của lượt này cũng không lưu được nên không có trace_id (xem log của server)"
	}' '	}' 'VT11 lượt lỗi không lưu được vết mà không nói là không có trace_id'
m $S 'class, _, _ := strings.Cut(fault, ":")' 'class := fault' 'VT12 tên thành phần nội bộ (infra:postgres) lộ ra client'
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
m platform/pgstore/load.go 'if _, err := tx.Exec(ctx, "SELECT pg_advisory_xact_lock($1)", loadLockKey(tenant)); err != nil {' 'if _, err := tx.Exec(ctx, "SELECT $1::bigint", loadLockKey(tenant)); err != nil {' 'LK1 Load cùng tenant không xếp hàng (trùng khoá chính, giao dịch sau bị huỷ)'
m platform/pgstore/load.go 'h.Write([]byte("eais.load:" + string(tenant)))' 'h.Write([]byte("eais.load:"))' 'LK2 khoá nạp không theo tenant (mọi tenant chờ nhau)'

# --- cổng SKIP và workflow CI ---
export PKGS=./cmd/eais/
m scripts/kiem-skip-test.py 'if ngoai or thua or test_hong or goi_hong:' 'if test_hong or goi_hong:' 'CI1 SKIP không làm cổng đỏ'
m scripts/kiem-skip-test.py 'if tong == 0:' 'if tong < 0:' 'CI2 đầu vào rỗng được coi là "không SKIP"'
m scripts/kiem-skip-test.py 'ngoai = [s for s in skip if (s[0], s[1]) not in cho_phep]' 'ngoai = [s for s in skip if True]' 'CI3 danh sách cho phép không có tác dụng'
m .github/workflows/ci.yml 'postgres:17@sha256:f4c66b820c6f974249089d3d16d86a3698eae11e8746eb6644b2271031e91232' 'postgres:17@sha256:f4c66b820c6f974249089d3d16d86a3698eae11e8746eb6644b2271031e91233' 'CI4 CI và compose ghim hai digest khác nhau'
m .github/workflows/ci.yml 'actions/checkout@v4.2.2' 'actions/checkout@v4' 'CI5 action không ghim tag đầy đủ'
m .github/workflows/ci.yml 'python3 -I scripts/kiem-skip-test.py test.json' 'true' 'CI6 bước cổng SKIP bị bỏ'
m .github/workflows/ci.yml 'permissions:
  contents: read' 'permissions:
  contents: write' 'CI7 CI có quyền ghi'
m .github/workflows/ci.yml '    defaults:
      run:
        shell: bash
' '' 'CI8 workflow bỏ ghim shell: bash (set -o pipefail là của bash)'
m .github/workflows/ci.yml '          set -eo pipefail
          go test -count=1 -json ./... | tee test.json' '          go test -count=1 -json ./... | tee test.json' 'CI9 bước go test nuốt mã lỗi (đường ống | tee không có pipefail)'
m .github/workflows/ci.yml 'go test -count=1 -json ./... | tee test.json' 'go test -count=1 -json ./... | tee test.json || true' 'CI10 bước go test có || true'
m scripts/kiem-skip-test.py 'if ngoai or thua or test_hong or goi_hong:' 'if ngoai or thua:' 'CI11 cổng không đỏ khi luồng có test / gói FAIL'
m scripts/kiem-skip-test.py 'if hanh_dong == "fail" and goi and goi not in goi_hong:' 'if False:' 'CI12 gói không dịch được (chỉ có fail mức gói) không bị đếm'
m .github/workflows/ci.yml 'poppler-utils fonts-dejavu-core' 'poppler-utils' 'CI13 CI không cài phông DejaVu (test dựng PDF thử tự SKIP)'
m .github/workflows/ci.yml '            echo "$bad" >&2
            exit 1' '            echo "$bad" >&2
            exit 0' 'CI14 bước gofmt không đỏ khi có tệp chưa format'
m .github/workflows/ci.yml 'run: go vet ./...' 'run: go vet ./... || true' 'CI15 bước go vet có || true'
m .github/workflows/ci.yml 'run: scripts/chay-thu.sh' 'run: scripts/chay-thu.sh || true' 'CI16 bước smoke E2E có || true'
m .github/workflows/ci.yml "TestRequestTypesHaveNoTenantField' ." "TestRequestTypesHaveNoTenantField' . || true" 'CI17 bước cổng danh tính có || true'

# --- script một lệnh không ghi vào PG đang chạy của người khác (R2-04): bắt bằng test có docker / go giả (cmd/eais/chaythu_test.go) ---
export PKGS=./cmd/eais/
m scripts/chay-thu.sh 'if [ -n "$(docker ps -q --filter "name=^/${CONTAINER}\$" 2>/dev/null)" ]; then' 'if false; then' 'RP1 script không thấy container đang chạy'
m scripts/chay-thu.sh 'elif (exec 3<>"/dev/tcp/127.0.0.1/$PORT") 2>/dev/null; then' 'elif false; then' 'RP2 script không thăm dò cổng'
m scripts/chay-thu.sh 'elif [ "${EAIS_REUSE_PG:-}" = 1 ]; then' 'elif true; then' 'RP3 không cần EAIS_REUSE_PG vẫn dùng lại PG đang chạy'
m scripts/chay-thu.sh 'elif [ "${EAIS_REUSE_PG:-}" = 1 ]; then' 'elif false; then' 'RP4 đặt EAIS_REUSE_PG=1 mà vẫn bị từ chối'
m scripts/chay-thu.sh 'EOF
		exit 2' 'EOF
		exit 0' 'RP5 từ chối mà thoát 0'
m scripts/chay-thu.sh 'dùng lại, sẽ không gỡ nó"' 'dùng lại, sẽ không gỡ nó"; started_pg=1' 'RP6 PG dùng lại bị gỡ khi đặt EAIS_PG_DOWN=1'
# R3-01: chỉ đổi cổng sang cổng trống (cùng dự án, cùng tên) thì bộ lọc tên container là hàng rào DUY NHẤT; docker giả của test chỉ thấy container khi lọc đúng name=^/<tên>$.
m scripts/chay-thu.sh '--filter "name=^/${CONTAINER}\$"' '--filter "name=^/${CONTAINER}x\$"' 'FILTER-1 test: bộ lọc tên container không bao giờ khớp'
m scripts/chay-thu.sh '--filter "name=^/${CONTAINER}\$"' '--filter "name=^/eais-khac\$"' 'FILTER-2 test: bộ lọc tên container lọc nhầm tên khác'
# R3-03: chỉ đúng "1" là lời xác nhận dùng lại (0, true vẫn bị từ chối).
m scripts/chay-thu.sh 'elif [ "${EAIS_REUSE_PG:-}" = 1 ]; then' 'elif [ -n "${EAIS_REUSE_PG:-}" ]; then' 'REUSE-1 test: EAIS_REUSE_PG đặt giá trị bất kỳ (cả 0, true) bị coi là cố ý dùng lại'

# --- phiên bản mã của vết và của mã lần nạp (R2-02): hàm chọn (version.go), bước ghi đè ở server, script nhúng -ldflags, workflow CI ---
V=cmd/eais/version.go
m $V 'if v := strings.TrimSpace(linked); v != "" {' 'if v := strings.TrimSpace(linked); v != "" && false {' 'CV1 giá trị -ldflags không được dùng'
m $V 'rev += "+dirty"' 'rev += ""' 'CV2 bản sửa VCS bẩn không có +dirty'
m $V 'v != "" && v != "(devel)"' 'v != ""' 'CV3 "(devel)" của phiên bản module được coi là phiên bản thật'
m $V 'return binHashPrefix + sum' 'return sum' 'CV4 băm binary không có tiền tố (lẫn với mã git)'
m $V 'if _, err := io.Copy(h, f); err != nil {' 'if _, err := io.WriteString(h, path); err != nil {' 'CV5 băm đường dẫn của binary thay vì nội dung'
m $V 'var buildVersion string' 'var buildVersion = strings.ToUpper("")' 'CV6 biến nhận -ldflags khởi tạo bằng lời gọi hàm (-X lặng lẽ không tác dụng)'
m $V 'return "(unknown)"' 'return "(devel)"' 'CV7 không có nguồn nào mà trả "(devel)"'
m $S 'out.Decision.CodeVersion = buildRevision()' '_ = 0' 'CV8 lượt thành công giữ phiên bản mã "(devel)" của luồng hỏi'
m $M ', "code_version", buildRevision())' ')' 'CV9 server không báo phiên bản mã ở dòng khởi động'
m scripts/chay-thu.sh "then code_version+='+dirty'" "then code_version+=''" 'CV10 script không đánh dấu +dirty khi cây bẩn'
m scripts/chay-thu.sh '-X main.buildVersion=$code_version' '-X main.buildVersions=$code_version' 'CV11 script nhúng vào biến không tồn tại (-X lặng lẽ không tác dụng)'
m scripts/chay-thu.sh '*[!0-9A-Za-z.+_-]*)' '*[!0-9A-Za-z.+_\ -]*)' 'CV12 script nhận EAIS_CODE_VERSION có khoảng trắng'
m .github/workflows/ci.yml 'EAIS_CODE_VERSION: ${{ github.sha }}' 'EAIS_CODE_VERSION: ""' 'CI18 bước smoke không truyền commit đang kiểm'

# --- smoke E2E bằng curl: cổng này cũng phải đỏ được ---
ms $S 'dec.DisallowUnknownFields()' '_ = 0' 'SM1 smoke: thân mang tenant được nhận'
ms $S 'if pr, found := s.principals[tok]; found {' 'if pr, found := s.principals[tok]; found || true {' 'SM2 smoke: token lạ vẫn qua'
ms $S 'if !s.fullAccess(pr) {' 'if false {' 'SM3 smoke: vai không toàn quyền nạp lại được'
ms $S 'if s.persist(r.Context(), pr, rec, acc) == nil {' 'if true {' 'SM4 smoke: lượt lỗi không có vết'
ms $S 'ctx := contracts.WithTenant(r.Context(), pr.TenantID)' 'ctx := contracts.WithTenant(r.Context(), contracts.TenantID(s.app.tenantID()))' 'SM5 smoke: tenant khác đọc được vết'
ms $A 'sum := sha256.Sum256([]byte(dataVersion + "\n" + packVersion + "\n" + codeVersion))' 'sum := sha256.Sum256([]byte(dataVersion + "\n" + packVersion + "\n" + codeVersion + time.Now().String()))' 'SM6 smoke: nạp lại nhân đôi cách ly'
ms $M ', "code_version", buildRevision())' ')' 'SM7 smoke: server không báo phiên bản mã lúc khởi động (script phải bắt)'
# SM8: trong một clone thường, thông tin VCS do go build ghi cho CÙNG giá trị nên biến nhúng sai tên không lộ ra; tắt VCS để phép thử không phụ thuộc nơi chạy.
ms scripts/chay-thu.sh '-ldflags "-X main.buildVersion=$code_version"' '-buildvcs=false -ldflags "-X main.buildVersions=$code_version"' 'SM8 smoke: script nhúng vào biến không tồn tại, binary không có VCS (script phải bắt)'

# --- script một lệnh: cổng bị chiếm, và PG đã chạy sẵn không bị gỡ ---
sc scen_squat 'SQ0 cổng bị chiếm: script nguyên bản từ chối, không chạy smoke vào kẻ chiếm cổng'
sc scen_squat 'SQ1 script gõ thẳng cổng yêu cầu thay vì chờ server của mình báo địa chỉ' scripts/chay-thu.sh 'if [ -n "$line" ]; then addr=${line#đang nghe http://}; break; fi' 'addr=$ADDR; break'
sc scen_srvdie 'SD0 server chết giữa lúc smoke: script nguyên bản thoát khác 0, không báo XANH'
sc scen_srvdie 'SD1 script không đổi mã thoát khi server chết giữa lúc smoke' scripts/chay-thu.sh 'if [ "$rc" = 0 ]; then rc=1; fi' 'if false; then rc=1; fi'
sc scen_pgkeep 'PK0 PG đã chạy sẵn: script nguyên bản không gỡ nó dù đặt EAIS_PG_DOWN=1'
sc scen_pgkeep 'PK1 script gỡ cả PG đã chạy từ trước' scripts/chay-thu.sh 'dùng lại, sẽ không gỡ nó"' 'dùng lại, sẽ không gỡ nó"; started_pg=1'
sc scen_pgrefuse 'RP7 PG đã chạy sẵn (lần 1 do script dựng): mặc định script nguyên bản TỪ CHỐI và không ghi gì; EAIS_REUSE_PG=1 mới dùng lại'
sc scen_pgrefuse 'RP8 script bỏ cổng từ chối: dùng lại PG đang chạy dù không đặt EAIS_REUSE_PG' scripts/chay-thu.sh 'elif [ "${EAIS_REUSE_PG:-}" = 1 ]; then' 'elif true; then'
sc scen_pgrefuse 'FILTER-1 kịch bản: bộ lọc tên container không bao giờ khớp, chỉ đổi cổng thì compose dựng lại PG đang chạy' scripts/chay-thu.sh '--filter "name=^/${CONTAINER}\$"' '--filter "name=^/${CONTAINER}x\$"'
sc scen_pgrefuse 'FILTER-2 kịch bản: bộ lọc tên container lọc nhầm tên khác' scripts/chay-thu.sh '--filter "name=^/${CONTAINER}\$"' '--filter "name=^/eais-khac\$"'

echo "TỔNG: $bat bị bắt, $song sống sót, $khac lỗi khác; $dat kịch bản nguyên bản đạt"
[ "$song" -eq 0 ] && [ "$khac" -eq 0 ]
