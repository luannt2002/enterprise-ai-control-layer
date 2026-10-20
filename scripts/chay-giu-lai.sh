#!/usr/bin/env bash
# Chạy MỘT lần một tập của bộ giữ lại đã đóng băng (ADR-0006 §7) và ghi bằng chứng thô:
#   scripts/chay-giu-lai.sh docs/g7/holdout/h1.yaml docs/g7/holdout/ket-qua-h1
#
# Ghi vào thư mục ra:
#   chay.txt               revision, giờ bắt đầu / kết thúc, băm tệp ca, băm dữ liệu, lệnh, mã thoát;
#   winreport-bo-nho.txt   đường bộ nhớ;
#   winreport-pg.txt       đường PostgreSQL (RLS), cùng dữ liệu;
#   bang-cong.md           bảng G2' / G5' / G6' của scripts/bang-cong-h1.py.
#
# Từ chối chạy khi:
# - thư mục ra đã có: mỗi tập chỉ chạy một lần, chạy lại là chỉnh theo bộ giữ lại;
# - cây git có thay đổi hay tệp lạ: mã được đo phải là đúng một revision đã commit;
# - tệp ca chưa commit, hoặc HEAD chưa có trên origin/main: không chứng minh được "đóng băng trước khi chạy";
# - thiếu EAIS_PG_ADMIN_DSN / EAIS_PG_DSN (xem README).
# Dữ liệu sinh mới (functional, seed 42) vào thư mục tạm; chỉ băm của nó được ghi, dữ liệu không được commit.
set -uo pipefail
[ $# -eq 2 ] || { echo "dùng: $0 <tệp ca đã đóng băng> <thư mục ra (chưa có)>" >&2; exit 2; }
cases=$1 out=$2
cd "$(dirname "$0")/.." || exit 2
[ -e "$out" ] && { echo "đã có $out: mỗi tập của bộ giữ lại chỉ chạy một lần" >&2; exit 2; }
[ -f "$cases" ] || { echo "không có $cases" >&2; exit 2; }
[ -z "$(git status --porcelain)" ] || { echo "cây git có thay đổi hoặc tệp chưa theo dõi: commit trước khi chạy" >&2; git status --short >&2; exit 2; }
git ls-files --error-unmatch "$cases" >/dev/null 2>&1 || { echo "$cases chưa commit" >&2; exit 2; }
git fetch -q origin main || { echo "không fetch được origin/main" >&2; exit 2; }
git merge-base --is-ancestor HEAD origin/main || { echo "HEAD chưa có trên origin/main: push trước khi chạy" >&2; exit 2; }
[ -n "${EAIS_PG_ADMIN_DSN:-}" ] && [ -n "${EAIS_PG_DSN:-}" ] || { echo "thiếu EAIS_PG_ADMIN_DSN / EAIS_PG_DSN" >&2; exit 2; }

data=$(mktemp -d) || exit 2
trap 'rm -rf "$data"' EXIT
go run ./tenants/construction/win/gen/cmd/gen -world tenants/construction/win/world.yaml -out "$data" -profile functional -seed 42 >/dev/null || exit 2

mkdir -p "$out" || exit 2
log="$out/chay.txt"
{
  echo "revision: $(git rev-parse HEAD)"
  echo "origin/main: $(git rev-parse origin/main)"
  echo "commit tệp ca: $(git log -1 --format='%h %cI' -- "$cases")"
  echo "sha256 tệp ca: $(sha256sum "$cases" | cut -d' ' -f1)  $cases"
  echo "sha256 manifest dữ liệu: $(sha256sum "$data/manifest.json" | cut -d' ' -f1)"
  echo "go: $(go version)"
  echo "bắt đầu: $(date -Iseconds)"
} >"$log"

run() {
  local name=$1; shift
  echo "lệnh $name: go run ./tenants/construction/win/winreport -data <tạm> -cases $cases -v $*" >>"$log"
  go run ./tenants/construction/win/winreport -data "$data" -cases "$cases" -v "$@" >"$out/winreport-$name.txt" 2>&1
  echo "mã thoát $name: $?" >>"$log"
}
run bo-nho
run pg -pg
echo "kết thúc: $(date -Iseconds)" >>"$log"

python3 -I scripts/bang-cong-h1.py "$out/winreport-bo-nho.txt" "$out/winreport-pg.txt" "$cases" >"$out/bang-cong.md"
echo "mã thoát bang-cong-h1: $?" >>"$log"
cat "$log"
