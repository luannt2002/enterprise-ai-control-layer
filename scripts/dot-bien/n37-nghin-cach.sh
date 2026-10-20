#!/usr/bin/env bash
# Danh mục đột biến cho nhóm nghìn ngăn bằng MỘT dấu cách ("3 850 lít" → 3.850 L; docs/tenants/construction/03-data-map.md:134),
# sửa SAU H1 (phát hiện qua ca 264 bởi người phản bác cuối; H1 là bộ hồi quy).
# Mỗi dòng: m <tệp> <chuỗi gốc> <chuỗi đột biến> <nhãn>. Kỳ vọng: mọi dòng "BỊ BẮT".
cd "$(dirname "$0")/../.." || exit 1
m() { scripts/mutate.sh "$1" "$2" "$3" "$4"; echo "rc=$?"; }
export PKGS="./data/normalization/ ./tenants/construction/"
F=data/normalization/engine.go
m $F '		if n := e.spaceGroups(s, toks, i, layout); n > i {' '		if n := e.spaceGroups(s, toks, i, layout); false && n > i {' 'NC1 không gộp nhóm nghìn'
m $F 'toks[i].text[0] == '"'"'0'"'"' || letterWordBefore(s, toks[i].start) {' 'letterWordBefore(s, toks[i].start) {' 'NC2 gộp cả số hiệu có số 0 đầu'
m $F ' || letterWordBefore(s, toks[i].start) {' ' {' 'NC7 gộp số có chữ đứng ngay trước (bể 2 500 lít)'
m $F '	if !digits(toks[i]) || len(toks[i].text) > 3 ||' '	if !digits(toks[i]) || len(toks[i].text) > 4 ||' 'NC3 phần đầu 4 chữ số vẫn gộp'
m $F '|| len(toks[k].text) != 3 {' '|| len(toks[k].text) < 2 {' 'NC4 nhóm sau không đủ 3 chữ số vẫn gộp'
m $F '		if toks[k-1].kind != tokSpace || toks[k-1].text != " " ||' '		if toks[k-1].kind != tokSpace ||' 'NC5 gộp qua nhiều dấu cách'
m $F '	if _, _, _, ok := e.matchUnit(s, toks[last].end, layout, false); !ok {' '	if _, _, _, ok := e.matchUnit(s, toks[last].end, layout, false); false && !ok {' 'NC6 gộp khi không có đơn vị'
# Soát C3 (C3-02 / C3-05): identifierBefore (danh sách đóng labelBefore ∪ countAfter ∪ periodWords) thay bằng letterWordBefore (MỌI chữ
# đứng ngay trước chặn gộp) — NC8, NC9 (bỏ từng danh sách) không còn mã đích nên bỏ dòng; luồng hỏi giữ số đầu làm con số ứng viên
# (thousandsHead). Không đưa vào (đã xét): bỏ kiểm "lượng" của span kế tiếp trong thousandsHead (chưa có ca số hiệu + nhóm ba chữ số
# KHÔNG đơn vị + một lượng khác — ghi nợ); bỏ kiểm "chỉ chữ số" của span (nhánh số trần chỉ nhận số không formatted nên tương đương).
A=core/pipeline/ask.go
m $A '		if !explicit && !formatted && !thousandsHead(in.Raw, in.Spans, i) {' '		if !explicit && !formatted {' 'TH1 số đầu của nhóm nghìn không gộp bị bỏ như nhãn (tổng số 3 850 lít)'
m $A '	return d == 3' '	return d >= 1' 'TH2 số hiệu trước lượng không có dạng nhóm ba chữ số bị coi là phần nghìn (phiếu 912 3850 L)'
