#!/usr/bin/env bash
# Danh mục đột biến cho canary TÊN của bộ đếm rò rỉ G7 (g7run nameCanaries, D-G7-9 mục 12).
# Mỗi dòng: m <tệp> <chuỗi gốc> <chuỗi đột biến> <nhãn>. Kỳ vọng: mọi dòng "BỊ BẮT".
# Không có đột biến "lấy cả tenant được chấm": bộ lọc chữ của tenant được chấm loại hết tên đó (tương đương).
cd "$(dirname "$0")/../.." || exit 1
m() { scripts/mutate.sh "$1" "$2" "$3" "$4"; echo "rc=$?"; }
export PKGS="./tenants/construction/win/g7run/"
F=tenants/construction/win/g7run/main.go
m $F '	canaries = append(canaries, nameCanaries(w, cfg.tenant, own)...)' '	canaries = append(canaries, nameCanaries(w, cfg.tenant, own)[:0]...)' 'NC1 bộ đếm không nhận canary tên'
m $F '|| strings.Contains(ownText, k) {' '|| (false && strings.Contains(ownText, k)) {' 'NC2 không loại tên dùng chung'
m $F 'if utf8.RuneCountInString(s) < 4 ||' 'if utf8.RuneCountInString(s) < 0 ||' 'NC3 nhận tên quá ngắn'
m $F '			add(p.Name)' '			_ = p.Name' 'NC4 bỏ tên dự án'
m $F '		add(t.ID)' '		_ = t.ID' 'NC5 bỏ mã tenant'
