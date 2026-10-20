#!/usr/bin/env bash
# Danh mục đột biến cho nhóm nghìn ngăn bằng MỘT dấu cách ("3 850 lít" → 3.850 L; docs/tenants/construction/03-data-map.md:134),
# sửa SAU H1 (phát hiện qua ca 264 bởi người phản bác cuối; H1 là bộ hồi quy).
# Mỗi dòng: m <tệp> <chuỗi gốc> <chuỗi đột biến> <nhãn>. Kỳ vọng: mọi dòng "BỊ BẮT".
cd "$(dirname "$0")/../.." || exit 1
m() { scripts/mutate.sh "$1" "$2" "$3" "$4"; echo "rc=$?"; }
export PKGS="./data/normalization/ ./tenants/construction/"
F=data/normalization/engine.go
m $F '		if n := e.spaceGroups(s, toks, i, layout); n > i {' '		if n := e.spaceGroups(s, toks, i, layout); false && n > i {' 'NC1 không gộp nhóm nghìn'
m $F '	if !digits(toks[i]) || len(toks[i].text) > 3 || toks[i].text[0] == '"'"'0'"'"' {' '	if !digits(toks[i]) || len(toks[i].text) > 3 {' 'NC2 gộp cả số hiệu có số 0 đầu'
m $F '	if !digits(toks[i]) || len(toks[i].text) > 3 ||' '	if !digits(toks[i]) || len(toks[i].text) > 4 ||' 'NC3 phần đầu 4 chữ số vẫn gộp'
m $F '|| len(toks[k].text) != 3 {' '|| len(toks[k].text) < 2 {' 'NC4 nhóm sau không đủ 3 chữ số vẫn gộp'
m $F '		if toks[k-1].kind != tokSpace || toks[k-1].text != " " ||' '		if toks[k-1].kind != tokSpace ||' 'NC5 gộp qua nhiều dấu cách'
m $F '	if _, _, _, ok := e.matchUnit(s, toks[last].end, layout, false); !ok {' '	if _, _, _, ok := e.matchUnit(s, toks[last].end, layout, false); false && !ok {' 'NC6 gộp khi không có đơn vị'
