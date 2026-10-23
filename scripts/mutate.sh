#!/usr/bin/env bash
# Chạy MỘT đột biến để chứng minh một cổng/test thật sự đỏ được (CLAUDE.md §1):
#   scripts/mutate.sh <tệp> <chuỗi gốc> <chuỗi đột biến> [nhãn]
#   scripts/mutate.sh --nen          chỉ chạy và lưu lượt nền (xem dưới), không đột biến
# Chuỗi gốc phải xuất hiện đúng 1 lần. Tệp luôn được khôi phục (kể cả khi bị ngắt) và
# kiểm sha256 sau khi chạy. Đột biến không biên dịch được thì KHÔNG tính là "bị bắt".
#
# Nền xanh (lỗi 5c của soát độc lập 23/10): đột biến chỉ đo được khi CÙNG lệnh test xanh trên cây chưa đột biến — nền đã đỏ thì mọi
# đột biến đều "bị bắt" giả. Lượt nền chạy trước đột biến và được lưu theo khoá (HEAD + git diff + PKGS + hai DSN của PG) ở
# MUTATE_NEN_DIR (mặc định $TMPDIR/eais-mutate-nen), dùng lại trong MUTATE_NEN_PHUT phút (mặc định 60) — một danh mục chỉ tốn một lượt nền.
# MUTATE_NEN_CO_SAN=1: chỉ dùng kết quả nền đã lưu, không tự chạy (cho đột biến chạy trên database vừa dựng lại: chạy nền ở đó sẽ áp
# migration GỐC trước đột biến) — runner chạy `scripts/mutate.sh --nen` trên database đó trước.
# MUTATE_TEST_MUON=<regex>: "bị bắt" chỉ tính khi có test đỏ khớp regex (test nhắm tới); đỏ ở test khác là "đỏ sai chỗ".
# Mã thoát: 0 bị bắt · 3 sống sót · 4 không biên dịch · 5 nền đỏ / chưa có nền · 6 đỏ sai chỗ · khác: lỗi dùng/khôi phục.
set -u
nen_only=0
if [ "${1:-}" = --nen ]; then
	nen_only=1
elif [ $# -lt 3 ]; then
	echo "dùng: $0 <tệp> <chuỗi gốc> <chuỗi đột biến> [nhãn] | $0 --nen" >&2
	exit 2
fi

# nen in XANH, DO hoặc CHUA_CO cho lệnh test hiện tại trên cây hiện tại (chưa đột biến).
nen_dir=${MUTATE_NEN_DIR:-${TMPDIR:-/tmp}/eais-mutate-nen}
nen_file=$nen_dir/$({ git rev-parse HEAD; git diff; echo "${PKGS:-./...}|${EAIS_PG_ADMIN_DSN:-}|${EAIS_PG_DSN:-}"; } 2>/dev/null | sha256sum | cut -c1-32)
nen() {
	if [ -f "$nen_file" ] && [ -n "$(find "$nen_file" -mmin -"${MUTATE_NEN_PHUT:-60}" 2>/dev/null)" ]; then
		cat "$nen_file"
		return
	fi
	if [ "${MUTATE_NEN_CO_SAN:-}" = 1 ] && [ "$nen_only" = 0 ]; then
		echo CHUA_CO
		return
	fi
	local st=DO tmp
	mkdir -p "$nen_dir" && go test ${PKGS:-./...} -count=1 >/dev/null 2>&1 && st=XANH
	tmp=$(mktemp "$nen_dir/.nen.XXXXXX") && printf '%s\n' "$st" >"$tmp" && mv -f "$tmp" "$nen_file"
	echo "$st"
}
st=$(nen)
if [ "$nen_only" = 1 ]; then
	echo "NỀN ${PKGS:-./...}: $st"
	[ "$st" = XANH ] && exit 0 || exit 5
fi
f=$1 from=$2 to=$3 label=${4:-$1}
case $st in
XANH) ;;
CHUA_CO)
	echo "[$label] CHƯA CÓ NỀN — MUTATE_NEN_CO_SAN=1 mà chưa chạy \`scripts/mutate.sh --nen\` cho lệnh test này; không đo"
	exit 5
	;;
*)
	echo "[$label] NỀN ĐỎ — test đã đỏ trên cây chưa đột biến (${PKGS:-./...}); sửa nền rồi mới đo đột biến"
	exit 5
	;;
esac
backup=$(mktemp) || exit 1
cp "$f" "$backup" || exit 1
before=$(sha256sum <"$f")
trap 'cp "$backup" "$f"; rm -f "$backup"' EXIT

python3 -I - "$f" "$from" "$to" <<'PY' || exit 2
import sys
p, a, b = sys.argv[1], sys.argv[2], sys.argv[3]
s = open(p, encoding="utf-8").read()
n = s.count(a)
if n != 1:
    sys.exit(f"chuỗi gốc xuất hiện {n} lần, cần đúng 1")
open(p, "w", encoding="utf-8").write(s.replace(a, b))
PY

# PKGS giới hạn phạm vi test (mặc định toàn module) — dùng khi một gói khác đang được viết dở.
out=$(go test ${PKGS:-./...} -count=1 2>&1)
cp "$backup" "$f"
[ "$before" = "$(sha256sum <"$f")" ] || { echo "KHÔI PHỤC LỖI: $f" >&2; exit 1; }

if grep -q -- '\[build failed\]\|\[setup failed\]' <<<"$out"; then
	echo "[$label] KHÔNG BIÊN DỊCH — viết lại đột biến cho biên dịch được"
	exit 4
fi
if grep -q -- '--- FAIL' <<<"$out"; then
	failed=$(grep -- '--- FAIL' <<<"$out" | sed 's/ (.*//; s/.*--- FAIL: //' | tr '\n' ' ')
	if [ -n "${MUTATE_TEST_MUON:-}" ] && ! tr ' ' '\n' <<<"$failed" | grep -E -q -- "$MUTATE_TEST_MUON"; then
		echo "[$label] ĐỎ SAI CHỖ: đỏ ở $failed— không test nào khớp MUTATE_TEST_MUON=$MUTATE_TEST_MUON"
		exit 6
	fi
	echo "[$label] BỊ BẮT: $failed"
	exit 0
fi
echo "[$label] SỐNG SÓT — chưa có test nào đỏ"
exit 3
