#!/usr/bin/env bash
# Chạy MỘT đột biến để chứng minh một cổng/test thật sự đỏ được (CLAUDE.md §1):
#   scripts/mutate.sh <tệp> <chuỗi gốc> <chuỗi đột biến> [nhãn]
# Chuỗi gốc phải xuất hiện đúng 1 lần. Tệp luôn được khôi phục (kể cả khi bị ngắt) và
# kiểm sha256 sau khi chạy. Đột biến không biên dịch được thì KHÔNG tính là "bị bắt".
# Mã thoát: 0 bị bắt · 3 sống sót · 4 không biên dịch · khác: lỗi dùng/khôi phục.
set -u
[ $# -ge 3 ] || { echo "dùng: $0 <tệp> <chuỗi gốc> <chuỗi đột biến> [nhãn]" >&2; exit 2; }
f=$1 from=$2 to=$3 label=${4:-$1}
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
	echo "[$label] BỊ BẮT: $(grep -- '--- FAIL' <<<"$out" | sed 's/ (.*//; s/.*--- FAIL: //' | tr '\n' ' ')"
	exit 0
fi
echo "[$label] SỐNG SÓT — chưa có test nào đỏ"
exit 3
