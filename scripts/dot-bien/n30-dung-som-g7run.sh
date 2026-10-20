#!/usr/bin/env bash
# Danh mục đột biến cho sửa C-03 (ca thiếu principal / id trùng không được âm thầm bỏ qua) và C-04 (dừng TRƯỚC lời gọi
# model khi -out đã có tệp hoặc lượt chạy thật có hệ cổng bị chặn) — soát độc lập lượt 2, chiều C.
# Không có dòng cho chỗ nối stopBeforeModel trong run(): nhánh chạy thật cần endpoint chính thức (mạng + khoá) nên
# không tới được trong test; hàm stopBeforeModel được test riêng.
cd "$(dirname "$0")/../.." || exit 1
m() { scripts/mutate.sh "$1" "$2" "$3" "$4"; echo "rc=$?"; }
export PKGS="./evals/"
m evals/runner.go '			bad("%sprincipal %q không có", AskErrorProblem, c.Principal)' '			bad("principal %q không có", c.Principal)' 'R1 ca thiếu principal không đánh dấu lỗi khi hỏi'
export PKGS="./tenants/construction/win/g7run/"
G=tenants/construction/win/g7run/main.go
m $G '		if _, ok := principals[c.Principal]; !ok {' '		if _, ok := principals[c.Principal]; !ok && false {' 'B1 nhận principal lạ'
m $G '		if seen[c.ID] {' '		if false {' 'B2 nhận id ca trùng'
m $G '		if err := checkOutDir(cfg.out); err != nil {' '		if err := checkOutDir(cfg.out); err != nil && false {' 'O1 không kiểm -out trước khi gọi model'
m $G '	return len(blocked) > 0 && !fake && !dry' '	return false' 'S1 không dừng khi hệ cổng bị chặn'
