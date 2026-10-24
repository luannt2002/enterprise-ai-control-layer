#!/bin/bash
# winreport cây cũ vs cây mới: 117 / H1 / H3 / H4 × bộ nhớ / PG, cùng dữ liệu sinh (functional, seed 42); in số dòng khác (bỏ dòng
# thời gian). rc=1 hai bên là bình thường (có ca trượt sẵn). Dùng: scripts/cong/win-so.sh <cây cũ> <cây mới> <thư mục ra> [database]
set -u
CU=$1 MOI=$2 O=$3 DB=${4:-eais}
export GOMAXPROCS=${GOMAXPROCS:-2} GOMEMLIMIT=${GOMEMLIMIT:-2GiB}
H=${EAIS_PG_HOST:-127.0.0.1} PORT=${EAIS_PG_PORT:-25432}
export EAIS_PG_ADMIN_DSN="postgres://postgres@$H:$PORT/$DB?sslmode=disable" EAIS_PG_DSN="postgres://eais_app@$H:$PORT/$DB?sslmode=disable"
mkdir -p "$O" || exit 2
(cd "$MOI" && go run ./tenants/construction/win/gen/cmd/gen -world tenants/construction/win/world.yaml -out "$O/data" -profile functional -seed 42 >/dev/null) || exit 1
for side in cu:$CU moi:$MOI; do (cd "${side#*:}" && go build -o "$O/winreport-${side%%:*}" ./tenants/construction/win/winreport) || exit 1; done
for c in tenants/construction/win/cases-independent.yaml docs/g7/holdout/h1.yaml docs/g7/holdout-h3/h3.yaml docs/g7/holdout-h4/h4.yaml; do
  t=$(basename $c .yaml)
  # mỗi bên chạy trong CÂY CỦA NÓ (pack + bộ ca của cây đó), chung một bộ dữ liệu sinh: pack đổi lược đồ thì binary cũ không đọc được pack mới
  for side in cu:$CU moi:$MOI; do name=${side%%:*}; for pg in "" -pg; do (cd "${side#*:}" && "$O/winreport-$name" -v -data "$O/data" -cases $c $pg > "$O/$t-$name${pg:-bo-nho}.txt" 2>&1); done; done
  for pg in bo-nho -pg; do
    n=$(diff <(grep -v "thời gian\|[0-9]ms" "$O/$t-cu$pg.txt") <(grep -v "thời gian\|[0-9]ms" "$O/$t-moi$pg.txt") | grep -c '^[<>]')
    echo "$t $pg: $(wc -l < "$O/$t-moi$pg.txt") dòng, khác $n"
  done
done
