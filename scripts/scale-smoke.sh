#!/usr/bin/env bash
# Scale smoke của WIN MVP (docs/04-core-mvp.md §6, đoạn cuối): chạy lại bộ ca WIN trên hồ sơ SMOKE phải
# ra CÙNG kết quả như trên hồ sơ FUNCTIONAL và không lỗi. Thời gian / RAM chỉ ghi, không đặt ngưỡng.
#   scripts/scale-smoke.sh              # hạt giống 42; đổi bằng SEED=7 scripts/scale-smoke.sh
# Cách làm: build gen + winreport MỘT lần (thời gian biên dịch không lẫn vào số đo) → sinh hai hồ sơ vào
# thư mục tạm (tự dọn khi thoát) → chạy `winreport -v` trên từng hồ sơ dưới `/usr/bin/time -v` → so từng ca
# theo mã ca. Ca ĐỎ đã biết khiến winreport thoát 1 nhưng KHÔNG làm script đỏ: cái cần so là TẬP ca ĐẠT/ĐỎ
# giữa hai hồ sơ, không phải việc WIN đã xong chưa. Chạy tuần tự và nice 10 vì box dùng chung.
# Đạt (exit 0) khi: cả hai lượt chạy không lỗi · G1 khớp 100% manifest của chính hồ sơ đó (cả hai hồ sơ) ·
# tập ca ĐẠT/ĐỎ theo mã ca giống hệt nhau.
# Mã thoát: 0 đạt · 1 KHÔNG ĐẠT (G1 chưa khớp hoặc tập ca lệch) · 2 lỗi chạy / thiếu công cụ.
set -u
cd "$(dirname "$0")/.." || exit 2 # winreport tìm pack / world / bộ ca theo đường dẫn từ gốc repo
SEED=${SEED:-42}
die() { echo "LỖI: $*" >&2; exit 2; }
command -v go >/dev/null || die "thiếu lệnh go"
command -v pdftotext >/dev/null || die "thiếu pdftotext (poppler): bộ nạp cần nó để đọc báo cáo PDF"
[ -x /usr/bin/time ] || die "thiếu /usr/bin/time (GNU time)"
T=$(mktemp -d) || exit 2
trap 'rm -rf "$T"' EXIT
trap 'exit 130' INT TERM

echo "build gen + winreport (một lần, ngoài số đo) ..."
nice -n 10 go build -o "$T/gen" ./tenants/construction/win/gen/cmd/gen || die "build gen lỗi"
nice -n 10 go build -o "$T/winreport" ./tenants/construction/win/winreport || die "build winreport lỗi"

# run <nhãn> <lệnh...>: chạy dưới time -v (số đo → $T/<nhãn>.time), stdout → .out, stderr → .err; trả mã thoát của lệnh.
run() { local l=$1; shift; nice -n 10 /usr/bin/time -v -o "$T/$l.time" "$@" >"$T/$l.out" 2>"$T/$l.err"; }
for p in functional smoke; do
	run "gen-$p" "$T/gen" -world tenants/construction/win/world.yaml -out "$T/$p" -profile "$p" -seed "$SEED" ||
		{ cat "$T/gen-$p.err" >&2; die "sinh hồ sơ $p lỗi"; }
done
for p in functional smoke; do
	run "rep-$p" "$T/winreport" -data "$T/$p" -v
	rc=$?
	[ "$rc" -le 1 ] || { cat "$T/rep-$p.err" >&2; die "winreport trên hồ sơ $p thoát $rc (0 hoặc 1 là bình thường, từ 2 là lỗi chạy)"; }
done

tv() { sed -n "s/^[[:space:]]*$2.*: //p" "$T/$1.time"; } # tv <nhãn> <đầu dòng của time -v>: giá trị sau dấu hai chấm cuối
mib() { LC_ALL=C awk -v k="${1:-0}" 'BEGIN { printf "%.1f MiB", k / 1024 }'; } # time -v ghi KB, đổi sang MiB
tab() { if command -v column >/dev/null; then LC_ALL=C.UTF-8 column -t -s $'\t'; else cat; fi; } # căn cột theo ký tự (chữ có dấu không lệch)
cls() { sed -n '/^BỘ CA WIN/,/^$/p' "$T/rep-$1.out" | awk 'NF == 2 && $2 ~ /^[0-9]+\/[0-9]+$/ { print $1 "\t" $2 }'; }
cases() { awk '$1 == "ĐẠT" || $1 == "ĐỎ" { print $1, $2 }' "$T/rep-$1.out" | LC_ALL=C sort -k2; }

echo
echo "== scale smoke WIN · seed $SEED · $(date '+%F %T') · $(nproc) vCPU · RAM $(awk '/MemTotal/ { printf "%d MB", $2 / 1024 }' /proc/meminfo) · $(go version | cut -d' ' -f3) =="
echo
echo "-- sinh dữ liệu: nice -n 10 /usr/bin/time -v gen -world tenants/construction/win/world.yaml -out <tmp>/<hồ sơ> -profile <hồ sơ> -seed $SEED"
{
	printf 'hồ sơ\tdung lượng\twall\tuser\tsys\tpeak RSS\n'
	for p in functional smoke; do
		printf '%s\t%s\t%s\t%s\t%s\t%s\n' "$p" "$(du -sh --apparent-size "$T/$p" | cut -f1)" "$(tv "gen-$p" Elapsed)" \
			"$(tv "gen-$p" 'User time')" "$(tv "gen-$p" 'System time')" "$(mib "$(tv "gen-$p" Maximum)")"
	done
} | tab

echo
echo "-- nạp + G1 + bộ ca: nice -n 10 /usr/bin/time -v winreport -data <tmp>/<hồ sơ> -v"
for p in functional smoke; do sed -n '/^NẠP/p;/^G1:/p;/^G6 PHÁT LẠI/p' "$T/rep-$p.out" | sed "s/^/[$p] /"; done

echo
echo "-- bảng theo lớp (đạt/tổng)"
{ printf 'lớp\tfunctional\tsmoke\n'; paste <(cls functional) <(cls smoke) | cut -f1,2,4; } | tab

echo
echo "-- thời gian · RAM của winreport (không đặt ngưỡng)"
{
	printf 'hồ sơ\twall\tuser\tsys\tpeak RSS\n'
	for p in functional smoke; do
		printf '%s\t%s\t%s\t%s\t%s\n' "$p" "$(tv "rep-$p" Elapsed)" "$(tv "rep-$p" 'User time')" \
			"$(tv "rep-$p" 'System time')" "$(mib "$(tv "rep-$p" Maximum)")"
	done
} | tab

echo
echo "-- so kết quả"
fail=0
for p in functional smoke; do
	read -r ok total <<<"$(sed -n 's/^G1: \(-\{0,1\}[0-9]*\)\/\([0-9]*\) bản ghi khớp manifest$/\1 \2/p' "$T/rep-$p.out")"
	[ "${total:-0}" -gt 0 ] || die "không đọc được dòng G1 trong báo cáo hồ sơ $p (định dạng báo cáo đổi?)"
	if [ "$ok" -ne "$total" ]; then
		echo "G1 $p: KHÔNG khớp ($ok/$total)"; grep '^  ✗' "$T/rep-$p.out" | head -5; fail=1
	fi
	cases "$p" >"$T/cases-$p"
	n=$(wc -l <"$T/cases-$p")
	want=$(sed -n 's/^BỘ CA WIN (\([0-9]*\) ca)$/\1/p' "$T/rep-$p.out")
	[ "$n" -gt 0 ] && [ "$n" = "${want:-x}" ] || die "đọc được $n ca nhưng báo cáo hồ sơ $p ghi ${want:-không có dòng 'BỘ CA WIN'} (định dạng báo cáo đổi?)"
done
if diff "$T/cases-functional" "$T/cases-smoke" >"$T/cases.diff"; then
	reds=$(awk '$1 == "ĐỎ" { print $2 }' "$T/cases-smoke" | paste -sd' ')
	echo "tập ca ĐẠT/ĐỎ theo mã ca: GIỐNG HỆT ($n ca; ĐỎ ở cả hai hồ sơ: ${reds:-không có})"
else
	echo "tập ca ĐẠT/ĐỎ theo mã ca: LỆCH"; sed -n 's/^< /  functional: /p;s/^> /  smoke:      /p' "$T/cases.diff" | head -20; fail=1
fi
body() { awk '$1 == "ĐẠT" || $1 == "ĐỎ" || $1 == "→"' "$T/rep-$1.out"; } # từng ca: dòng kết quả + câu trả lời
if cmp -s <(body functional) <(body smoke); then
	echo "câu trả lời + lý do đỏ của từng ca: giống từng byte (chỉ để biết, không quyết định mã thoát)"
else
	echo "câu trả lời / lý do đỏ khác nhau giữa hai hồ sơ (chỉ để biết, không quyết định mã thoát)"
fi

echo
if [ "$fail" -eq 0 ]; then echo "SCALE SMOKE: ĐẠT"; else echo "SCALE SMOKE: KHÔNG ĐẠT"; fi
exit "$fail"
