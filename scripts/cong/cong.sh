#!/bin/bash
# Cổng hồi quy: go test -json ./... có PG, KHÔNG nạp .env (động cơ thật tự bỏ qua). Tệp nháp chưa theo dõi docs/DEBUG-LO-TRINH.md (nếu có)
# được dời ra rồi trả lại — test gốc quét docs/. Dùng: scripts/cong/cong.sh <cây> <tệp json ra> [database, mặc định eais] [gói, mặc định ./...]
# rồi so theo TÊN với mốc: python3 scripts/cong/so_ten.py <json mốc> <json mới> (rỗng không phải xanh: số tên phải > 0).
# Go lấy từ PATH (bản theo go.mod); PG theo deploy/docker-compose.yml (cổng 25432) trừ khi đã đặt EAIS_PG_HOST / EAIS_PG_PORT.
# Cuối lượt (F-014, lỗ của đợt gộp D-RAG-05): cổng SKIP của CI — scripts/kiem-skip-test.py với scripts/cong/skip-cho-phep.txt — khi gói là
# ./... (chạy một phần thì mục cho phép của gói không chạy thành THỪA giả). So THÂN test (F-034): đặt CONG_MOC=<cây mốc, đường tuyệt đối>
# [CONG_DOI_CO_CHU_Y=<tệp "<tên> <lý do>">] thì chạy thêm scripts/cong/so-than — đổi kỳ vọng mà giữ tên chỉ hiện ở đó, so_ten.py không thấy.
# Mã thoát: 0 khi go test, cổng SKIP và so thân (nếu chạy) đều 0; 1 nếu không.
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
rc=$?
echo "go test rc=$rc"
if [ "$PKGS" = ./... ]; then
	python3 -I scripts/kiem-skip-test.py "$OUT" --cho-phep scripts/cong/skip-cho-phep.txt
	r=$?; echo "kiem-skip-test rc=$r"; [ $r = 0 ] || rc=1
else
	echo "kiem-skip-test: KHÔNG chạy (gói $PKGS, không phải ./...) — cổng gộp phải chạy đủ ./..."
fi
if [ -n "${CONG_MOC:-}" ]; then
	go run ./scripts/cong/so-than "$CONG_MOC" . ${CONG_DOI_CO_CHU_Y:+"$CONG_DOI_CO_CHU_Y"}
	r=$?; echo "so-than rc=$r"; [ $r = 0 ] || rc=1
else
	echo "so-than: KHÔNG chạy (chưa đặt CONG_MOC) — so theo tên không thấy đổi kỳ vọng giữ tên"
fi
[ $rc = 0 ]
