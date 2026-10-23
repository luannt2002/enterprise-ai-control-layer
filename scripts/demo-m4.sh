#!/usr/bin/env bash
# Kịch bản demo của khối M4 (HOP-DONG §2.6.4): hỏi "Máy 03 hôm qua lệch bao nhiêu dầu?", rồi yêu cầu tạo phiếu kiểm tra chênh lệch, rồi xem `eais trace`
# từng bước của CẢ HAI lượt; thử phá bằng gửi lại nguyên văn và bằng command_id mới.
#
#   EAIS_PG_ADMIN_DSN=postgres://<quản trị>@<máy>:<cổng>/<database riêng>?sslmode=disable \
#   EAIS_PG_DSN=postgres://<vai ứng dụng>@<máy>:<cổng>/<database riêng>?sslmode=disable \
#   scripts/demo-m4.sh
#
# CẢNH BÁO: script GHI vào PG mà hai biến trên trỏ tới — migrate, rồi nạp THAY toàn bộ dữ liệu của tenant trong pack construction (seed 42), rồi tạo MỘT phiếu.
# Vì vậy KHÔNG có giá trị mặc định cho hai biến: người chạy phải cấp database RIÊNG (đừng trỏ vào database đang phục vụ việc khác); thiếu thì từ chối (thoát 2).
#
# Chỉ chạy ở chế độ `serve -dev` (token công khai suy từ world.yaml, chỉ nghe loopback; HOP-DONG §1.3): cổng do hệ điều hành chọn (127.0.0.1:0), không đụng
# cổng nào đang dùng. Cần: Go, curl, python3 (-I), pdftotext, psql (đếm phiếu bằng truy vấn quản trị). Không gọi model, không mạng ra ngoài. Mọi dữ liệu là
# giả lập; kết luận chỉ là "trên dữ liệu giả lập, chưa có người nghiệp vụ duyệt".
#
# Biến tuỳ chọn: EAIS_DEMO_TENANT (abc-infra), EAIS_DEMO_TOKEN (dev-abc-infra-u-director: người dùng có tickets.create và metrics.read), EAIS_DEMO_MACHINE (EX-03),
# EAIS_DEMO_QUESTION, EAIS_WORK (thư mục làm việc; mặc định mktemp -d, xoá khi xong). Script chỉ cho thấy "chạy được", không thay cho cổng X1–X9.
# Mã thoát: 0 cả sáu bước đúng · 2 từ chối (thiếu DSN) · 1 có bước sai (kèm lý do) · khác: lỗi môi trường.
#
# TRẠNG THÁI [ĐO 23/10]: ĐÃ CHẠY ĐỦ sáu bước, thoát 0, trên một cây THỬ GỘP (nhánh K4 + m4/k3-route@1a56f9b, không commit) với database riêng: DONE, mã phiếu, hai vế 3850 và 3782 L,
# chênh lệch 68, hai bảng vết (11 bước S0…S10 và 9 bước C0…C7), gửi lại nguyên văn replayed = true cùng mã phiếu, command_id mới 409 DUPLICATE_OPEN_TICKET kèm mã phiếu cũ, đúng 1 phiếu OPEN
# bằng truy vấn quản trị. Trên main (chưa có route lệnh của K3) chỉ chạy được bước 1–2; script dừng ở bước 3 với lời nói rõ vì sao khi gặp 404.
set -euo pipefail
cd "$(dirname "$0")/.."
for v in EAIS_PG_ADMIN_DSN EAIS_PG_DSN; do
	if [ -z "${!v:-}" ]; then
		echo "TỪ CHỐI: đặt $v (database RIÊNG của bạn; script ghi vào đó, không có giá trị mặc định)" >&2
		exit 2
	fi
done
export EAIS_PG_ADMIN_DSN EAIS_PG_DSN
TENANT=${EAIS_DEMO_TENANT:-abc-infra}
TOKEN=${EAIS_DEMO_TOKEN:-dev-abc-infra-u-director}
MACHINE=${EAIS_DEMO_MACHINE:-EX-03}
QUESTION=${EAIS_DEMO_QUESTION:-Máy 03 hôm qua lệch bao nhiêu dầu?}
WORLD=tenants/construction/win/world.yaml
made_work=0 server_pid=''
if [ -z "${EAIS_WORK:-}" ]; then
	EAIS_WORK=$(mktemp -d) made_work=1
else
	mkdir -p "$EAIS_WORK"
fi
cleanup() {
	local rc=$?
	if [ -n "$server_pid" ] && kill -0 "$server_pid" 2>/dev/null; then
		kill -TERM "$server_pid" 2>/dev/null || true
		wait "$server_pid" 2>/dev/null || true
	fi
	[ "$made_work" = 1 ] && rm -rf "$EAIS_WORK"
	exit "$rc"
}
trap cleanup EXIT
t0=$SECONDS
step() { printf '\n== [%ss] %s\n' "$((SECONDS - t0))" "$*"; }
fail() { echo "SAI: $*" >&2; exit 1; }
jget() { python3 -I -c 'import json,sys; d=json.load(sys.stdin); v=d
for k in sys.argv[1].split("."):
    v = v.get(k, "") if isinstance(v, dict) else ""
print(v if not isinstance(v, bool) else str(v).lower())' "$1"; }

# "hôm qua" của tenant = reference_date của world trừ một ngày (đồng hồ -now world), theo đúng định nghĩa của HOP-DONG §12.1.
DAY=$(python3 -I -c 'import re,sys,datetime
m = re.search(r"^reference_date:\s*(\d{4}-\d{2}-\d{2})", open(sys.argv[1], encoding="utf-8").read(), re.M)
print((datetime.date.fromisoformat(m.group(1)) - datetime.timedelta(days=1)).isoformat())' "$WORLD")

step "dịch binary eais, migrate, sinh dữ liệu seed 42, nạp"
go build -o "$EAIS_WORK/eais" ./cmd/eais
"$EAIS_WORK/eais" migrate
go run ./tenants/construction/win/gen/cmd/gen -world "$WORLD" -out "$EAIS_WORK/data" -profile functional -seed 42
"$EAIS_WORK/eais" ingest -pack tenants/construction/pack -data "$EAIS_WORK/data"

step "chạy server (nền, chế độ dev, đồng hồ -now world, cổng do hệ điều hành chọn)"
"$EAIS_WORK/eais" serve -dev -addr 127.0.0.1:0 -pack tenants/construction/pack -world "$WORLD" -data "$EAIS_WORK/data" -now world \
	>"$EAIS_WORK/serve.log" 2>&1 &
server_pid=$!
addr=''
for _ in $(seq 1 120); do
	kill -0 "$server_pid" 2>/dev/null || { echo "server thoát sớm; log:" >&2; cat "$EAIS_WORK/serve.log" >&2; exit 1; }
	line=$(grep -m1 '^đang nghe http://' "$EAIS_WORK/serve.log" || true)
	if [ -n "$line" ]; then addr=${line#đang nghe http://}; break; fi
	sleep 0.25
done
[ -n "$addr" ] || { echo "server không báo địa chỉ đang nghe sau 30 giây; log:" >&2; cat "$EAIS_WORK/serve.log" >&2; exit 1; }
curl -fsS -m 5 "http://$addr/healthz" >/dev/null || fail "server không trả lời /healthz"
echo "server ở http://$addr (dev, loopback)"
post() { # post <đường> <thân JSON> → in thân phản hồi; mã HTTP ghi vào $EAIS_WORK/http
	curl -sS -m 30 -o "$EAIS_WORK/body" -w '%{http_code}' -X POST "http://$addr$1" -H "Authorization: Bearer $TOKEN" -H 'Content-Type: application/json' -d "$2" >"$EAIS_WORK/http"
	cat "$EAIS_WORK/body"
}

step "1. hỏi: $QUESTION"
ask_body=$(python3 -I -c 'import json,sys; print(json.dumps({"text": sys.argv[1], "conversation_id": "demo", "message_id": "m1"}, ensure_ascii=False))' "$QUESTION")
r1=$(post /v1/ask "$ask_body"); echo "$r1"
[ "$(cat "$EAIS_WORK/http")" = 200 ] || fail "bước 1: /v1/ask trả HTTP $(cat "$EAIS_WORK/http")"
T1=$(jget trace_id <<<"$r1"); [ -n "$T1" ] || fail "bước 1: phản hồi không có trace_id"
echo "-> trace_id T1 = $T1"

step "2. eais trace -tenant $TENANT T1 (bảng 11 bước S0…S10)"
"$EAIS_WORK/eais" trace -tenant "$TENANT" "$T1"

step "3. yêu cầu tạo phiếu: POST /v1/commands/open_discrepancy_ticket (yêu cầu CÓ CẤU TRÚC, không phải câu chữ; HOP-DONG Q-P1-8)"
CMD1=$(python3 -I -c 'import uuid; print(uuid.uuid4())')
cmd_body=$(python3 -I -c 'import json,sys; print(json.dumps({"command_id": sys.argv[1], "subject_key": sys.argv[2], "metric": "issue_vs_measured", "day": sys.argv[3]}))' "$CMD1" "$MACHINE" "$DAY")
r3=$(post /v1/commands/open_discrepancy_ticket "$cmd_body"); echo "$r3"
code=$(cat "$EAIS_WORK/http")
if [ "$code" = 404 ]; then
	fail "bước 3: route lệnh chưa có (HTTP 404). Route POST /v1/commands/open_discrepancy_ticket do K3 thêm vào cmd/eais; chạy lại khi K3 đã gộp (bước 1–2 đã chạy xong ở trên)"
fi
[ "$code" = 200 ] || fail "bước 3: HTTP $code, muốn 200 (DONE)"
[ "$(jget status <<<"$r3")" = DONE ] || fail "bước 3: status không phải DONE"
[ "$(jget replayed <<<"$r3")" = false ] || fail "bước 3: lần đầu phải có replayed = false"
[ "$(jget ticket.status <<<"$r3")" = OPEN ] || fail "bước 3: phiếu phải OPEN"
TICKET=$(jget ticket.id <<<"$r3"); T2=$(jget trace_id <<<"$r3")
[ -n "$TICKET" ] && [ -n "$T2" ] || fail "bước 3: thiếu mã phiếu hoặc trace_id"
echo "-> phiếu $TICKET, chênh lệch $(jget ticket.diff <<<"$r3") (trace_id T2 = $T2)"

step "4. eais trace -tenant $TENANT T2 (bảng 9 bước C0…C7; lượt DONE không có bước lệch)"
"$EAIS_WORK/eais" trace -tenant "$TENANT" "$T2"

step "5. (thử phá) gửi lại NGUYÊN VĂN bước 3: DONE, replayed = true, cùng mã phiếu"
r5=$(post /v1/commands/open_discrepancy_ticket "$cmd_body"); echo "$r5"
[ "$(cat "$EAIS_WORK/http")" = 200 ] && [ "$(jget status <<<"$r5")" = DONE ] || fail "bước 5: gửi lại phải DONE"
[ "$(jget replayed <<<"$r5")" = true ] || fail "bước 5: gửi lại phải có replayed = true"
[ "$(jget ticket.id <<<"$r5")" = "$TICKET" ] || fail "bước 5: gửi lại phải cùng mã phiếu $TICKET"

step "6. (thử phá) gửi lại với command_id MỚI: FAILED DUPLICATE_OPEN_TICKET kèm mã phiếu cũ"
CMD2=$(python3 -I -c 'import uuid; print(uuid.uuid4())')
cmd_body2=$(python3 -I -c 'import json,sys; print(json.dumps({"command_id": sys.argv[1], "subject_key": sys.argv[2], "metric": "issue_vs_measured", "day": sys.argv[3]}))' "$CMD2" "$MACHINE" "$DAY")
r6=$(post /v1/commands/open_discrepancy_ticket "$cmd_body2"); echo "$r6"
[ "$(cat "$EAIS_WORK/http")" = 409 ] || fail "bước 6: muốn HTTP 409 (DUPLICATE_OPEN_TICKET), được $(cat "$EAIS_WORK/http")"
[ "$(jget reason_code <<<"$r6")" = DUPLICATE_OPEN_TICKET ] || fail "bước 6: reason_code phải là DUPLICATE_OPEN_TICKET"
[ "$(jget ticket.id <<<"$r6")" = "$TICKET" ] || fail "bước 6: phải kèm mã phiếu cũ $TICKET"

step "đối chiếu bằng truy vấn quản trị ngoài đường ứng dụng: đúng MỘT phiếu OPEN cho (máy, chỉ số, ngày)"
n=$(psql -X -At "$EAIS_PG_ADMIN_DSN" -c "SELECT count(*) FROM discrepancy_tickets WHERE tenant_id = '$TENANT' AND subject_key = '$MACHINE' AND metric = 'issue_vs_measured' AND day = '$DAY' AND status = 'OPEN'")
[ "$n" = 1 ] || fail "số phiếu OPEN của ($MACHINE, issue_vs_measured, $DAY) = $n, muốn 1"
echo "số phiếu OPEN = $n"

step "xong sau $((SECONDS - t0)) giây: cả sáu bước đúng (trên dữ liệu giả lập, chưa có người nghiệp vụ duyệt)"
