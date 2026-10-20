#!/usr/bin/env bash
# Danh mục đột biến cho sửa C-11 (soát độc lập lượt 2, chiều C): bằng chứng thô của g7run mang bản ghi quyết định
# của EACL cho từng câu (để gán tầng sai đầu tiên mà không gọi lại model); baseline không có.
# Mỗi dòng: m <tệp> <chuỗi gốc> <chuỗi đột biến> <nhãn>. Kỳ vọng: mọi dòng "BỊ BẮT".
cd "$(dirname "$0")/../.." || exit 1
m() { scripts/mutate.sh "$1" "$2" "$3" "$4"; echo "rc=$?"; }
export PKGS="./tenants/construction/win/g7run/"
F=tenants/construction/win/g7run/main.go
m $F '		ea.onDecision = sink.decision' '		ea.onDecision = nil' 'DC1 không nối bản ghi quyết định vào bằng chứng'
m $F '	if err == nil && e.onDecision != nil {' '	if false && e.onDecision != nil {' 'DC2 EACL không gửi bản ghi quyết định'
m tenants/construction/win/g7run/evidence.go '		s.cur.Decision = &d' '		_ = d' 'DC3 sink bỏ bản ghi quyết định'
m $F '		DataVersion: fileSHA256(filepath.Join(cfg.data, "manifest.json")),' '		DataVersion: "",' 'DC4 bản ghi quyết định không gắn bộ dữ liệu (soát C2, F-10)'
