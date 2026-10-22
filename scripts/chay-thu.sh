#!/usr/bin/env bash
# Một lệnh từ checkout sạch: PostgreSQL → migrate → dữ liệu seed 42 → nạp → serve (nền) → smoke E2E → dừng server.
#
#   scripts/chay-thu.sh
#
# Cần: Go (theo go.mod), Docker (chỉ khi chưa có PG), curl, python3, pdftotext (poppler-utils). Không gọi model, không mạng ra ngoài
# ngoài việc tải module Go / ảnh Docker đã ghim. Mọi dữ liệu là giả lập.
#
# Cấu hình bằng biến môi trường (mặc định như deploy/docker-compose.yml):
#   EAIS_PG_ADMIN_DSN + EAIS_PG_DSN  đã đặt cả hai thì dùng PG đó (CI, PG sẵn có) và KHÔNG chạm Docker
#   EAIS_COMPOSE_PROJECT   tên dự án compose              (eais)
#   EAIS_PG_CONTAINER      tên container PG               (eais-pg)
#   EAIS_PG_PORT           cổng PG trên 127.0.0.1         (25432)
#   EAIS_ADDR              địa chỉ nghe của server        (127.0.0.1:0 = hệ điều hành chọn cổng trống, script đọc cổng thật từ
#                          dòng "đang nghe http://…" của server; serve -dev chỉ nhận loopback)
#   EAIS_WORK              thư mục làm việc (binary, dữ liệu, log); mặc định mktemp -d, xoá khi xong nếu do script tạo
#   EAIS_PG_DOWN=1         gỡ PG khi xong NẾU CHÍNH script này đã dựng nó (mặc định giữ lại để soi tay)
#
# An toàn: nếu container PG tên đó ĐÃ chạy trước khi script bắt đầu (của việc khác), script dùng lại nhưng KHÔNG BAO GIỜ gỡ nó. Muốn
# chạy cạnh một PG đang dùng, đổi EAIS_COMPOSE_PROJECT / EAIS_PG_CONTAINER / EAIS_PG_PORT.
# Mã thoát: 0 smoke xanh · khác 0 có bước hỏng (log của server in ra khi server không lên).
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
	fi
	[ "$made_work" = 1 ] && rm -rf "$EAIS_WORK"
	exit "$rc"
}
trap cleanup EXIT

if [ -n "${EAIS_PG_ADMIN_DSN:-}" ] && [ -n "${EAIS_PG_DSN:-}" ]; then
	step "dùng PG có sẵn (EAIS_PG_ADMIN_DSN / EAIS_PG_DSN đã đặt); không chạm Docker"
else
	step "dựng PG bằng docker compose (dự án $PROJECT, container $CONTAINER, cổng 127.0.0.1:$PORT)"
	if [ -n "$(docker ps -q --filter "name=^/${CONTAINER}\$" --filter status=running)" ]; then
		echo "container $CONTAINER ĐÃ chạy từ trước — dùng lại, sẽ không gỡ nó"
	else
		EAIS_PG_CONTAINER=$CONTAINER EAIS_PG_PORT=$PORT docker compose -p "$PROJECT" -f deploy/docker-compose.yml up -d --wait
		started_pg=1
	fi
	export EAIS_PG_ADMIN_DSN="postgres://postgres@127.0.0.1:$PORT/eais?sslmode=disable"
	export EAIS_PG_DSN="postgres://eais_app@127.0.0.1:$PORT/eais?sslmode=disable"
fi

step "dịch binary eais"
go build -o "$EAIS_WORK/eais" ./cmd/eais

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

step "smoke E2E"
rc=0
scripts/smoke-e2e.sh "http://$addr" || rc=$?

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
