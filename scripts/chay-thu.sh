#!/usr/bin/env bash
# Một lệnh từ checkout sạch: PostgreSQL → migrate → dữ liệu seed 42 → nạp → serve (nền) → smoke E2E → dừng server.
#
#   scripts/chay-thu.sh
#
# CẢNH BÁO: script GHI vào PG mà nó dùng — migrate rồi nạp THAY toàn bộ dữ liệu của tenant trong pack construction. Vì vậy với giá trị
# mặc định nó chỉ chạy trên PG do CHÍNH nó dựng: thấy container EAIS_PG_CONTAINER đang chạy, hoặc cổng 127.0.0.1:EAIS_PG_PORT đã có
# tiến trình nghe, thì TỪ CHỐI (thoát 2, chưa ghi gì) và in hướng dẫn — kể cả khi đó là PG còn sót lại của lần chạy trước. Cố ý dùng lại
# PG đó: EAIS_REUSE_PG=1. Đã đặt cả EAIS_PG_ADMIN_DSN và EAIS_PG_DSN thì script dùng đúng PG ấy mà không hỏi (CI, PG của bạn): hai biến
# đó chính là lời xác nhận, nên đừng để chúng trong shell nếu không muốn script ghi vào PG đó.
#
# Cần: Go (theo go.mod), Docker (chỉ khi chưa có PG), curl, python3, pdftotext (poppler-utils). Không gọi model, không mạng ra ngoài
# ngoài việc tải module Go / ảnh Docker đã ghim. Mọi dữ liệu là giả lập.
#
# Cấu hình bằng biến môi trường (mặc định như deploy/docker-compose.yml):
#   EAIS_PG_ADMIN_DSN + EAIS_PG_DSN  đã đặt cả hai thì dùng PG đó (CI, PG sẵn có) và KHÔNG chạm Docker
#   EAIS_COMPOSE_PROJECT   tên dự án compose              (eais)
#   EAIS_PG_CONTAINER      tên container PG               (eais-pg)
#   EAIS_PG_PORT           cổng PG trên 127.0.0.1         (25432)
#   EAIS_REUSE_PG=1        CỐ Ý dùng lại PG đang chạy ở container / cổng trên (mặc định: từ chối); PG dùng lại không bao giờ bị gỡ
#   EAIS_CODE_VERSION      phiên bản mã nhúng vào binary (-ldflags -X main.buildVersion=…; chỉ chữ, số, . + _ -); mặc định là mã git của cây
#                          (git rev-parse HEAD, thêm "+dirty" khi cây có sửa chưa commit hoặc tệp mới chưa theo dõi). CI đặt = github.sha
#   EAIS_ADDR              địa chỉ nghe của server        (127.0.0.1:0 = hệ điều hành chọn cổng trống, script đọc cổng thật từ
#                          dòng "đang nghe http://…" của server; serve -dev chỉ nhận loopback)
#   EAIS_WORK              thư mục làm việc (binary, dữ liệu, log); mặc định mktemp -d, xoá khi xong nếu do script tạo
#   EAIS_PG_DOWN=1         gỡ PG khi xong NẾU CHÍNH script này đã dựng nó (mặc định giữ lại để soi tay; lần chạy sau khi PG đó còn chạy sẽ
#                          bị từ chối — gỡ trước, đổi tên / cổng, hoặc đặt EAIS_REUSE_PG=1)
#   EAIS_SMOKE             lệnh smoke thay cho scripts/smoke-e2e.sh, nhận địa chỉ http://… làm đối số (cho phép thử của chính script này)
#
# An toàn: script chỉ gỡ container mà CHÍNH nó dựng (và chỉ khi EAIS_PG_DOWN=1); PG dùng lại nhờ EAIS_REUSE_PG=1 hay nhờ DSN có sẵn không
# bao giờ bị gỡ.
# Mã thoát: 0 smoke xanh · 2 từ chối vì đã có PG chạy ở chỗ script định dựng · khác 0 có bước hỏng (log của server in ra khi server không lên).
set -euo pipefail
cd "$(dirname "$0")/.."
t0=$SECONDS
step() { printf '\n== [%ss] %s\n' "$((SECONDS - t0))" "$*"; }

PROJECT=${EAIS_COMPOSE_PROJECT:-eais}
CONTAINER=${EAIS_PG_CONTAINER:-eais-pg}
PORT=${EAIS_PG_PORT:-25432}
ADDR=${EAIS_ADDR:-127.0.0.1:0}
started_pg=0 server_pid='' made_work=0
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
	if [ "$started_pg" = 1 ] && [ "${EAIS_PG_DOWN:-}" = 1 ]; then
		step "gỡ PG do script dựng ($CONTAINER)"
		EAIS_PG_CONTAINER=$CONTAINER EAIS_PG_PORT=$PORT docker compose -p "$PROJECT" -f deploy/docker-compose.yml down >/dev/null 2>&1 || true
	elif [ "$started_pg" = 1 ]; then
		echo "PG $CONTAINER vẫn chạy ở 127.0.0.1:$PORT. Gỡ: EAIS_PG_CONTAINER=$CONTAINER EAIS_PG_PORT=$PORT docker compose -p $PROJECT -f deploy/docker-compose.yml down"
		echo "(chạy lại script khi PG này còn chạy sẽ bị TỪ CHỐI; muốn cố ý dùng lại: EAIS_REUSE_PG=1)"
	fi
	[ "$made_work" = 1 ] && rm -rf "$EAIS_WORK"
	exit "$rc"
}
trap cleanup EXIT

# Phiên bản mã nhúng vào binary. `go build` chỉ tự ghi mã git vào binary khi dựng trong một clone thường; dựng trong git worktree liên kết
# hay cây không có .git thì không có, và vết của ứng dụng khai mã là "(devel)" — hai binary khác mã không phân biệt được, và mã lần nạp
# (băm của dữ liệu + pack + mã) không đổi theo mã. Nhúng tường minh bằng -ldflags; không có git và không đặt EAIS_CODE_VERSION thì để rỗng,
# khi đó ứng dụng dùng băm sha256 của chính binary (tiền tố bin-sha256:).
code_version=${EAIS_CODE_VERSION:-}
if [ -z "$code_version" ] && git rev-parse --verify -q HEAD >/dev/null 2>&1; then
	code_version=$(git rev-parse HEAD)
	if [ -n "$(git status --porcelain 2>/dev/null)" ]; then code_version+='+dirty'; fi
fi
case $code_version in
*[!0-9A-Za-z.+_-]*)
	echo "EAIS_CODE_VERSION chỉ được gồm chữ, số và . + _ - (nhận '$code_version')" >&2
	exit 2
	;;
esac

# pg_chiem in ra (một dòng) cái đang chiếm chỗ mà script sắp dựng PG của nó; rỗng nếu chỗ đó trống. Hai dấu hiệu: container tên $CONTAINER
# đang chạy, hoặc có tiến trình nghe ở 127.0.0.1:$PORT. Hỏi docker KHÔNG lọc theo trạng thái: `--filter status=running` bỏ sót container
# đang tạm dừng (paused) hay đang khởi động lại, mà container tạm dừng vẫn giữ cổng. Cổng thử bằng một lần kết nối TCP (không gửi gì).
pg_chiem() {
	if [ -n "$(docker ps -q --filter "name=^/${CONTAINER}\$" 2>/dev/null)" ]; then
		echo "container $CONTAINER ĐÃ chạy từ trước"
	elif (exec 3<>"/dev/tcp/127.0.0.1/$PORT") 2>/dev/null; then
		echo "cổng 127.0.0.1:$PORT ĐÃ có tiến trình nghe"
	fi
}

if [ -n "${EAIS_PG_ADMIN_DSN:-}" ] && [ -n "${EAIS_PG_DSN:-}" ]; then
	step "dùng PG có sẵn (EAIS_PG_ADMIN_DSN / EAIS_PG_DSN đã đặt); không chạm Docker"
else
	step "chuẩn bị PG (dự án $PROJECT, container $CONTAINER, cổng 127.0.0.1:$PORT)"
	busy=$(pg_chiem)
	if [ -z "$busy" ]; then
		EAIS_PG_CONTAINER=$CONTAINER EAIS_PG_PORT=$PORT docker compose -p "$PROJECT" -f deploy/docker-compose.yml up -d --wait
		started_pg=1
	elif [ "${EAIS_REUSE_PG:-}" = 1 ]; then
		echo "$busy — EAIS_REUSE_PG=1: dùng lại, sẽ không gỡ nó"
	else
		cat >&2 <<EOF
TỪ CHỐI: $busy.
Script này migrate rồi nạp THAY toàn bộ dữ liệu của tenant trong pack construction (seed 42) vào PG nó dùng. Nếu PG đang chạy đó là của
việc khác (đang đo, đang giữ dữ liệu) thì dữ liệu ấy bị ghi đè — nên script không tự dùng lại. Chọn một:
  1. PG riêng, không đụng PG đang chạy: đổi tên container, cổng và dự án compose, ví dụ
       EAIS_PG_CONTAINER=eais-thu EAIS_PG_PORT=25441 EAIS_COMPOSE_PROJECT=eais-thu scripts/chay-thu.sh
  2. CỐ Ý dùng lại và ghi đè PG đang chạy (chỉ khi chắc nó là của bạn):
       EAIS_REUSE_PG=1 scripts/chay-thu.sh
EOF
		exit 2
	fi
	export EAIS_PG_ADMIN_DSN="postgres://postgres@127.0.0.1:$PORT/eais?sslmode=disable"
	export EAIS_PG_DSN="postgres://eais_app@127.0.0.1:$PORT/eais?sslmode=disable"
fi

step "dịch binary eais (phiên bản mã nhúng: ${code_version:-không có — ứng dụng dùng băm của chính binary})"
go build -ldflags "-X main.buildVersion=$code_version" -o "$EAIS_WORK/eais" ./cmd/eais

step "migrate (vai quản trị)"
"$EAIS_WORK/eais" migrate

step "sinh dữ liệu seed 42 (hồ sơ functional)"
go run ./tenants/construction/win/gen/cmd/gen -world tenants/construction/win/world.yaml -out "$EAIS_WORK/data" -profile functional -seed 42

step "nạp dữ liệu của tenant vào PG (quan sát + cách ly có lý do)"
"$EAIS_WORK/eais" ingest -pack tenants/construction/pack -data "$EAIS_WORK/data"

step "chạy server (nền, chế độ dev, nghe $ADDR)"
"$EAIS_WORK/eais" serve -dev -addr "$ADDR" -pack tenants/construction/pack -world tenants/construction/win/world.yaml \
	-data "$EAIS_WORK/data" -now world >"$EAIS_WORK/serve.log" 2>&1 &
server_pid=$!
# Chỉ hỏi địa chỉ mà CHÍNH server của script đã báo bằng dòng "đang nghe http://…" (server in dòng đó sau khi gắn cổng thành công).
# KHÔNG hỏi /healthz của địa chỉ yêu cầu khi chưa có dòng đó: cổng đang bị một server KHÁC giữ (bản cũ, tiến trình khác) sẽ trả lời
# healthz thay cho server mới còn đang khởi động hoặc đang chết vì không gắn được cổng, và smoke chạy nhầm vào nó.
addr=''
for _ in $(seq 1 120); do
	if ! kill -0 "$server_pid" 2>/dev/null; then
		echo "server thoát sớm; log:" >&2
		cat "$EAIS_WORK/serve.log" >&2
		exit 1
	fi
	line=$(grep -m1 '^đang nghe http://' "$EAIS_WORK/serve.log" || true)
	if [ -n "$line" ]; then addr=${line#đang nghe http://}; break; fi
	sleep 0.25
done
[ -n "$addr" ] || { echo "server không báo địa chỉ đang nghe sau 30 giây; log:" >&2; cat "$EAIS_WORK/serve.log" >&2; exit 1; }
echo "server của script nghe ở http://$addr"
up=0
for _ in $(seq 1 60); do
	if ! kill -0 "$server_pid" 2>/dev/null; then
		echo "server thoát sớm sau khi báo đang nghe; log:" >&2
		cat "$EAIS_WORK/serve.log" >&2
		exit 1
	fi
	if curl -fsS -m 2 "http://$addr/healthz" >/dev/null 2>&1; then up=1; break; fi
	sleep 0.5
done
[ "$up" = 1 ] || { echo "server không trả lời /healthz sau 30 giây; log:" >&2; cat "$EAIS_WORK/serve.log" >&2; exit 1; }
# -X của trình liên kết lặng lẽ không làm gì khi tên biến lệch: đối chiếu với phiên bản mà CHÍNH server báo ở dòng khởi động.
reported=$(grep -om1 'code_version=[^ ]*' "$EAIS_WORK/serve.log" || true)
if [ -n "$code_version" ] && [ "$reported" != "code_version=$code_version" ]; then
	echo "FAIL: server báo '${reported:-không có}', muốn code_version=$code_version đã nhúng (-ldflags -X main.buildVersion không có tác dụng?); log:" >&2
	cat "$EAIS_WORK/serve.log" >&2
	exit 1
fi

step "smoke E2E"
rc=0
"${EAIS_SMOKE:-scripts/smoke-e2e.sh}" "http://$addr" || rc=$?

step "dừng server"
if kill -0 "$server_pid" 2>/dev/null; then
	kill -TERM "$server_pid"
else
	echo "server của script đã thoát giữa chừng lúc chạy smoke; log:" >&2
	cat "$EAIS_WORK/serve.log" >&2
	if [ "$rc" = 0 ]; then rc=1; fi
fi
wait "$server_pid" 2>/dev/null || true
server_pid=''
grep -E 'tổng kết|CHỈ DEV' "$EAIS_WORK/serve.log" || true

step "xong sau $((SECONDS - t0)) giây: smoke $([ "$rc" = 0 ] && echo XANH || echo ĐỎ)"
exit "$rc"
