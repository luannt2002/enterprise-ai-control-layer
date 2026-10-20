#!/usr/bin/env bash
# Danh mục đột biến cho nhóm nghìn ngăn bằng MỘT dấu cách ("3 850 lít" → 3.850 L; docs/tenants/construction/03-data-map.md:134),
# sửa SAU H1 (phát hiện qua ca 264 bởi người phản bác cuối; H1 là bộ hồi quy).
# Mỗi dòng: m <tệp> <chuỗi gốc> <chuỗi đột biến> <nhãn>. Kỳ vọng: mọi dòng "BỊ BẮT".
cd "$(dirname "$0")/../.." || exit 1
m() { scripts/mutate.sh "$1" "$2" "$3" "$4"; echo "rc=$?"; }
export PKGS="./data/normalization/ ./tenants/construction/ ./data/ingestion/"
F=data/normalization/engine.go
m $F '		if n, certain := e.spaceGroups(s, toks, i, layout); n > i {' '		if n, certain := e.spaceGroups(s, toks, i, layout); false && n > i {' 'NC1 không gộp nhóm nghìn'
m $F ' || toks[i].text[0] == '"'"'0'"'"' {' ' {' 'NC2 gộp cả số hiệu có số 0 đầu'
m $F '	return last, strings.TrimSpace(s[:toks[i].start]) == ""' '	return last, true' 'CU1 nhóm nghìn sau chữ / ký hiệu coi là chắc chắn (bể 2 500 lít)'
m $F '	if !digits(toks[i]) || len(toks[i].text) > 3 ||' '	if !digits(toks[i]) || len(toks[i].text) > 4 ||' 'NC3 phần đầu 4 chữ số vẫn gộp'
m $F '|| len(toks[k].text) != 3 {' '|| len(toks[k].text) < 2 {' 'NC4 nhóm sau không đủ 3 chữ số vẫn gộp'
m $F '		if toks[k-1].kind != tokSpace || toks[k-1].text != " " ||' '		if toks[k-1].kind != tokSpace ||' 'NC5 gộp qua nhiều dấu cách'
m $F '	if _, _, _, ok := e.matchUnit(s, toks[last].end, layout, false); !ok {' '	if _, _, _, ok := e.matchUnit(s, toks[last].end, layout, false); false && !ok {' 'NC6 gộp khi không có đơn vị'
# Sau soát fd4d19c (F1–F5): identifierBefore (danh sách từ nhãn) bỏ — NC8, NC9 (bỏ từng danh sách) không còn mã đích; nhóm nghìn luôn gộp,
# chưa phân định khi không đứng đầu văn bản (CU1, UG1), luồng hỏi hỏi lại (GU1, GU4), nạp dữ liệu cách ly (GU2). Không đưa vào (tương
# đương, đã xét): bỏ "|| len(cs.groups) > 0" của claimedValue — Handle đã hỏi lại mọi câu có nhóm chưa phân định trước khi answerMetric
# chạy (như doubts ở n35).
A=core/pipeline/ask.go I=data/ingestion/ingest.go
m $F '	return strings.HasPrefix(sp.Rule, ruleSpaceThousandsUncertain)' '	return false' 'UG1 không span nào chưa phân định'
m $A '; c.start >= lo && c.uncertain && !slices.Contains(cs.groups, raw) {' '; false && c.uncertain && !slices.Contains(cs.groups, raw) {' 'GU1 luồng hỏi bỏ qua nhóm chưa phân định'
m $A ', len(claims.doubts) > 0, len(claims.groups) > 0' ', len(claims.doubts) > 0, false' 'GU4 không hỏi lại khi nhóm chưa phân định'
m $I '	if normalization.UncertainGroups(cands[0]) {' '	if false {' 'GU2 nạp "bể 2 500 lít" thành 2.500 L'
