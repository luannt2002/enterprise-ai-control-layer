#!/usr/bin/env bash
# Chạy MỘT lần một tập của bộ giữ lại đã đóng băng (ADR-0006 §7) và ghi bằng chứng thô:
#   scripts/chay-giu-lai.sh docs/g7/holdout/h1.yaml docs/g7/holdout/ket-qua-h1
#
# Ghi vào thư mục ra:
#   chay.txt               revision, giờ, băm tệp ca / manifest / dữ liệu / công cụ, lệnh, mã thoát, trạng thái;
#   winreport-bo-nho.txt   đường bộ nhớ;
#   winreport-pg.txt       đường PostgreSQL (RLS), cùng dữ liệu;
#   bang-cong.md           bảng G2' / G5' / G6' của scripts/bang-cong-h1.py.
#
# Từ chối chạy khi:
# - tệp ca không phải một tệp của holdout-manifest.json cùng thư mục, hoặc băm khác bản đóng băng;
# - tệp ca là h2.yaml mà chưa đặt CHO_PHEP_H2=1 (ADR-0006 §7: H2 không chạy khi H1 còn là bộ giữ lại);
# - tập này đã có lần chạy HOÀN TẤT (tìm theo băm tệp ca trong mọi chay.txt dưới thư mục bộ giữ lại); lần
#   trước chỉ HỎNG HẠ TẦNG thì được chạy lại khi MÃ vẫn y nguyên — băm cây git không tính docs/, vì bằng chứng
#   của lần hỏng phải được commit trước (đo lại cùng một thứ, không chỉnh);
# - thư mục ra không nằm ngay trong thư mục bộ giữ lại, hoặc đã có (tạo bằng mkdir nguyên tử: hai lần chạy
#   đồng thời không thể cùng qua);
# - cây git có thay đổi hay tệp lạ; tệp ca chưa commit; HEAD chưa có trên origin/main;
# - DSN PG thiếu, hoặc không trỏ vào database riêng eais_giu_lai (tiến trình khác nạp song song vào DB chung
#   từng gây lỗi khoá ngoại 23503 và có thể đốt lần chạy duy nhất).
# Mã thoát: 0 lần chạy HOÀN TẤT (dù có ca trượt: đó là kết quả đo); 1 HỎNG HẠ TẦNG (winreport lỗi cấu hình /
# nạp, hay bảng cổng không lập được); 2 bị từ chối trước khi chạy.
# Dữ liệu sinh mới (functional, seed 42) vào thư mục tạm; chỉ băm của nó được ghi, dữ liệu không được commit.
set -uo pipefail
refuse() { echo "TỪ CHỐI: $*" >&2; exit 2; }
[ $# -eq 2 ] || { echo "dùng: $0 <tệp ca đã đóng băng> <thư mục ra (chưa có)>" >&2; exit 2; }
cases=$1 out=$2
cd "$(dirname "$0")/.." || exit 2
[ -f "$cases" ] && [ ! -L "$cases" ] || refuse "không có tệp ca thường: $cases"
dir=$(dirname "$cases") base=$(basename "$cases")
manifest="$dir/holdout-manifest.json"
[ -f "$manifest" ] || refuse "không có $manifest cạnh tệp ca"
sha_cases=$(sha256sum "$cases" | cut -d' ' -f1)
frozen=$(python3 -I -c 'import json,sys; print(json.load(open(sys.argv[1],encoding="utf-8")).get("files_sha256",{}).get(sys.argv[2],""))' "$manifest" "$base")
[ -n "$frozen" ] || refuse "$base không phải tệp của bản đóng băng ($manifest)"
[ "$frozen" = "$sha_cases" ] || refuse "$base khác bản đóng băng: $sha_cases ≠ $frozen"
[ "$base" != "h2.yaml" ] || [ "${CHO_PHEP_H2:-}" = 1 ] || refuse "H2 không chạy khi H1 còn là bộ giữ lại (ADR-0006 §7); đặt CHO_PHEP_H2=1 khi chủ dự án cho"
[ -z "$(git status --porcelain)" ] || { git status --short >&2; refuse "cây git có thay đổi hoặc tệp chưa theo dõi: commit trước khi chạy"; }
git ls-files --error-unmatch "$cases" >/dev/null 2>&1 || refuse "$cases chưa commit"
git fetch -q origin main || refuse "không fetch được origin/main"
git merge-base --is-ancestor HEAD origin/main || refuse "HEAD chưa có trên origin/main: push trước khi chạy"
rev=$(git rev-parse HEAD)
tree=$(git ls-tree -r HEAD) && [ -n "$tree" ] || refuse "không đọc được cây git của HEAD"
code=$(printf '%s\n' "$tree" | awk -F'\t' '$2 !~ /^docs\//' | sha256sum | cut -d' ' -f1)
while IFS= read -r prev; do
  if grep -q "^trạng thái: HOÀN TẤT" "$prev"; then refuse "tập này đã chạy HOÀN TẤT: $prev"; fi
  grep -q "^mã (cây git không tính docs/): $code\$" "$prev" || refuse "lần trước ($prev) HỎNG HẠ TẦNG với mã khác: chạy lại sau khi đổi mã là chỉnh theo bộ giữ lại"
done < <(grep -rl --include=chay.txt "^sha256 tệp ca: $sha_cases" "$dir" 2>/dev/null)
dbname() { python3 -I -c 'import sys,urllib.parse as u; print(u.urlparse(sys.argv[1].strip()).path.lstrip("/"))' "$1"; }
# DSN theo đúng luật của pgstore.EnvDSN (winreport đọc bằng hàm đó): biến EAIS_ đã đặt, kể cả rỗng, thắng; không đặt thì đọc biến
# cùng tên tiền tố EACL_ (tên trước D-RM-02). Nhờ vậy DSN được kiểm ở đây đúng là DSN winreport sẽ dùng.
admin_dsn=${EAIS_PG_ADMIN_DSN-${EACL_PG_ADMIN_DSN-}} app_dsn=${EAIS_PG_DSN-${EACL_PG_DSN-}}
[ -n "${admin_dsn// /}" ] && [ -n "${app_dsn// /}" ] || refuse "thiếu EAIS_PG_ADMIN_DSN / EAIS_PG_DSN"
[ "$(dbname "$admin_dsn")" = eais_giu_lai ] && [ "$(dbname "$app_dsn")" = eais_giu_lai ] || refuse "đường PG phải dùng database riêng eais_giu_lai (DSN đang trỏ $(dbname "$app_dsn"))"
[ "$(dirname "$out")" = "$dir" ] || refuse "thư mục ra phải nằm ngay trong $dir (để lần chạy sau thấy chay.txt của lần này)"
mkdir "$out" 2>/dev/null || refuse "thư mục ra đã có hoặc không tạo được: $out"

data=$(mktemp -d) || exit 1
trap 'rm -rf "$data"' EXIT
log="$out/chay.txt"
{
  echo "revision: $rev"
  echo "mã (cây git không tính docs/): $code"
  echo "commit HEAD: $(git log -1 --format='%cI')"
  echo "origin/main: $(git rev-parse origin/main)"
  echo "commit tệp ca: $(git log -1 --format='%h %cI' -- "$cases")"
  echo "sha256 tệp ca: $sha_cases  $cases"
  echo "sha256 holdout-manifest.json: $(sha256sum "$manifest" | cut -d' ' -f1)"
  echo "sha256 công cụ: bang-cong-h1.py $(sha256sum scripts/bang-cong-h1.py | cut -d' ' -f1); chay-giu-lai.sh $(sha256sum scripts/chay-giu-lai.sh | cut -d' ' -f1)"
  echo "database PG: $(dbname "$app_dsn")"
  echo "go: $(go version)"
  echo "bắt đầu: $(date -Iseconds)"
} >"$log"
go run ./tenants/construction/win/gen/cmd/gen -world tenants/construction/win/world.yaml -out "$data" -profile functional -seed 42 >/dev/null || { echo "trạng thái: HỎNG HẠ TẦNG (bộ sinh dữ liệu)" >>"$log"; exit 1; }
echo "sha256 manifest dữ liệu: $(sha256sum "$data/manifest.json" | cut -d' ' -f1)" >>"$log"

infra=0
run() {
  local name=$1; shift
  echo "lệnh $name: go run ./tenants/construction/win/winreport -data <tạm> -cases $cases -v $*" >>"$log"
  go run ./tenants/construction/win/winreport -data "$data" -cases "$cases" -v "$@" >"$out/winreport-$name.txt" 2>&1
  local rc=$?
  echo "mã thoát $name: $rc (0 mọi ca đạt · 1 có ca trượt hoặc phát lại lệch · khác: hỏng hạ tầng)" >>"$log"
  [ $rc -le 1 ] || infra=1
}
run bo-nho
run pg -pg
echo "kết thúc: $(date -Iseconds)" >>"$log"

python3 -I scripts/bang-cong-h1.py "$out/winreport-bo-nho.txt" "$out/winreport-pg.txt" "$cases" >"$out/bang-cong.md"
rc_bang=$?
echo "mã thoát bang-cong-h1: $rc_bang" >>"$log"
[ $rc_bang -eq 0 ] || infra=1
if [ $infra -eq 0 ]; then
  echo "trạng thái: HOÀN TẤT" >>"$log"
else
  echo "trạng thái: HỎNG HẠ TẦNG — không phải lần đo; đổi tên thư mục này (giữ làm bằng chứng) rồi chạy lại ở CÙNG revision" >>"$log"
fi
cat "$log"
exit $infra
