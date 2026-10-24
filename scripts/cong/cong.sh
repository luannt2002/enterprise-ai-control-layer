#!/bin/bash
# Cổng hồi quy: go test -json ./... có PG, KHÔNG nạp .env (động cơ thật tự bỏ qua). Tệp nháp chưa theo dõi docs/DEBUG-LO-TRINH.md (nếu có)
# được dời ra rồi trả lại — test gốc quét docs/. Dùng: scripts/cong/cong.sh <cây> <tệp json ra> [database, mặc định eais] [gói, mặc định ./...]
# rồi so theo TÊN với mốc: python3 scripts/cong/so_ten.py <json mốc> <json mới> (rỗng không phải xanh: số tên phải > 0).
# Go lấy từ PATH (bản theo go.mod); PG theo deploy/docker-compose.yml (cổng 25432) trừ khi đã đặt EAIS_PG_HOST / EAIS_PG_PORT.
set -u
T=$1 OUT=$2 DB=${3:-eais} PKGS=${4:-./...}
H=${EAIS_PG_HOST:-127.0.0.1} PORT=${EAIS_PG_PORT:-25432}
export GOMAXPROCS=${GOMAXPROCS:-2} GOMEMLIMIT=${GOMEMLIMIT:-2GiB}
export EAIS_PG_ADMIN_DSN="postgres://postgres@$H:$PORT/$DB?sslmode=disable" EAIS_PG_DSN="postgres://eais_app@$H:$PORT/$DB?sslmode=disable"
unset $(env | grep -oE '^EAIS_(RAG|MODEL|EMBED|RERANK|DECIDE)[A-Z_]*') 2>/dev/null
cd "$T" || exit 2
TAM=$(mktemp -d)
if [ -e docs/DEBUG-LO-TRINH.md ]; then mv docs/DEBUG-LO-TRINH.md "$TAM/"; trap 'mv "$TAM/DEBUG-LO-TRINH.md" "'"$T"'/docs/DEBUG-LO-TRINH.md"' EXIT; fi
go test -json -count=1 -p 2 $PKGS > "$OUT" 2> "$OUT.err"
echo "go test rc=$?"
