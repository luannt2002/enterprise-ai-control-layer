#!/usr/bin/env bash
# Danh mục đột biến cho sửa C-07 (soát độc lập lượt 2, chiều C): trần ngân sách bắt buộc, không vượt trần hợp đồng,
# manifest ghi đúng trần; tệp phải băm mà không đọc được thì dừng trước lời gọi model.
# Mỗi dòng: m <tệp> <chuỗi gốc> <chuỗi đột biến> <nhãn>. Kỳ vọng: mọi dòng "BỊ BẮT".
cd "$(dirname "$0")/../.." || exit 1
m() { scripts/mutate.sh "$1" "$2" "$3" "$4"; echo "rc=$?"; }
export PKGS="./tenants/construction/win/g7run/"
F=tenants/construction/win/g7run/main.go
m $F 'if cfg.budget <= 0 || cfg.budget > contractBudgetUSD {' 'if cfg.budget < 0 || cfg.budget > contractBudgetUSD {' 'NB1 trần 0 lại tắt trần'
m $F 'if cfg.budget <= 0 || cfg.budget > contractBudgetUSD {' 'if cfg.budget <= 0 || cfg.budget > 100 {' 'NB2 vượt trần hợp đồng vẫn chạy'
m $F 'f != "" && err != nil {' 'f != "" && err != nil && false {' 'NB3 tệp phải băm không đọc được vẫn chạy'
m tenants/construction/win/g7run/evidence.go 'BudgetUSD: cfg.budget,' 'BudgetUSD: 15,' 'NB4 manifest ghi trần cố định'
