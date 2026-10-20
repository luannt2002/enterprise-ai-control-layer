#!/usr/bin/env bash
# Danh mục đột biến "G6 — thuộc tính tổng ≥ 0, dẫn xuất = a − b" đã chạy ngày 20/10/2026 (xem docs/ops/NHAT-KY-2026-10-20.md).
# Mỗi dòng: m <tệp> <chuỗi gốc> <chuỗi đột biến> <nhãn>. Kỳ vọng: mọi dòng in "BỊ BẮT" (rc=0).
# Mã đổi sau ngày đó thì chuỗi gốc có thể không còn khớp (mutate.sh báo "xuất hiện 0 lần", rc=2):
# sửa chuỗi theo mã mới rồi chạy lại — đừng xoá dòng cho xanh.
cd "$(dirname "$0")/../.." || exit 1
export PKGS="./data/sourceoftruth/"
m() { scripts/mutate.sh "$1" "$2" "$3" "$4"; echo "rc=$?"; }
m data/sourceoftruth/query.go '	n, err := op(left.Total.Number, right.Total.Number)
	if err != nil {
		return Result{}, contracts.Label(contracts.FaultData, err)
	}' '	n, err := op(left.Total.Number, right.Total.Number)
	if err != nil {
		return Result{}, contracts.Label(contracts.FaultData, err)
	}
	if strings.HasPrefix(n, "-") {
		n = "0"
	}' 'PR1 dẫn xuất kẹp âm về 0'
m data/sourceoftruth/query.go '	n, err := op(left.Total.Number, right.Total.Number)' '	n, err := op(right.Total.Number, left.Total.Number)' 'PR2 dẫn xuất đảo hai vế'
m data/sourceoftruth/query.go '			t, err := decimal.Add(total, best.value)' '			t, err := decimal.Sub(total, best.value)' 'PR3 tổng trừ thay vì cộng'
