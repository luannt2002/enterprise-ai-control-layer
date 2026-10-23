#!/usr/bin/env bash
# Danh mục đột biến cho nhóm nghìn ngăn bằng MỘT dấu cách ("3 850 lít" → 3.850 L; docs/tenants/construction/03-data-map.md dòng 14), viết lại cho
# phương án (a') của D-OP-9: nhóm hợp lệ LUÔN gộp thành một lượng; con số không đứng đầu văn bản (hay chồng một phần lên tên thiết bị) là CHƯA
# PHÂN ĐỊNH — đường nạp cách ly, luồng hỏi hỏi lại; ô lượng của nguồn có cấu trúc luôn chắc chắn.
# Mỗi dòng: m <tệp> <chuỗi gốc> <chuỗi đột biến> <nhãn>. Kỳ vọng: mọi dòng "BỊ BẮT".
#   NC*  bộ gộp (engine.go spaceGroups)        CU*  cờ chưa phân định / đầu văn bản / ô có cấu trúc (engine.go)
#   UG*, SG*  UncertainGroup, GroupSplitsName   GI*  đường nạp (ingest.go)        GA*  luồng hỏi (ask.go)
# Bản trước có NC7 (gộp sau từ nhãn / đếm / kỳ), NC8, NC9 (bỏ từng danh sách): danh sách identifierBefore đã bỏ nên không còn mã đích.
# Không đưa vào danh mục (đột biến tương đương, đã xét): bỏ `|| len(cs.groups) > 0` của claimedValue — Handle đã chuyển câu có nhóm chưa phân
# định sang hỏi lại trước khi answerMetric chạy, nên claimedValue không bao giờ thấy groups (như doubts ở n35); bỏ `continue` sau khối nhóm chồng
# tên của claimNumbers — dòng kiểm "số trong tên thực thể" ngay sau nó cũng `continue`.
cd "$(dirname "$0")/../.." || exit 1
m() { scripts/mutate.sh "$1" "$2" "$3" "$4"; echo "rc=$?"; }
export PKGS="./data/normalization/ ./data/ingestion/ ./tenants/construction/ ./core/pipeline/"
F=data/normalization/engine.go I=data/ingestion/ingest.go A=core/pipeline/ask.go
m $F $'\t\tif n := e.spaceGroups(s, toks, i, layout); n > i {' $'\t\tif n := e.spaceGroups(s, toks, i, layout); false && n > i {' 'NC1 không gộp nhóm nghìn'
m $F $'\tif !digits(toks[i]) || len(toks[i].text) > 3 || toks[i].text[0] == \'0\' {' $'\tif !digits(toks[i]) || len(toks[i].text) > 3 {' 'NC2 gộp cả số hiệu có số 0 đầu'
m $F $'\tif !digits(toks[i]) || len(toks[i].text) > 3 ||' $'\tif !digits(toks[i]) || len(toks[i].text) > 4 ||' 'NC3 phần đầu 4 chữ số vẫn gộp'
m $F '|| len(toks[k].text) != 3 {' '|| len(toks[k].text) < 2 {' 'NC4 nhóm sau không đủ 3 chữ số vẫn gộp'
m $F $'\t\tif toks[k-1].kind != tokSpace || toks[k-1].text != " " ||' $'\t\tif toks[k-1].kind != tokSpace ||' 'NC5 gộp qua nhiều dấu cách'
m $F $'\tif _, _, _, ok := e.matchUnit(s, toks[last].end, layout, false); !ok {' $'\tif _, _, _, ok := e.matchUnit(s, toks[last].end, layout, false); false && !ok {' 'NC6 gộp khi không có đơn vị'
m $F $'\tif !SpaceGroup(*sp) || atTextStart(s, sp.Start) {' $'\tif !SpaceGroup(*sp) || true {' 'CU1 không span nào chưa phân định'
m $F $'\tif !SpaceGroup(*sp) || atTextStart(s, sp.Start) {' $'\tif !SpaceGroup(*sp) || false {' 'CU2 nhóm đứng đầu văn bản cũng chưa phân định'
m $F $'\treturn strings.TrimFunc(s[:pos], func(r rune) bool { return unicode.IsSpace(r) || isFormat(r) }) == ""' $'\treturn strings.TrimSpace(s[:pos]) == ""' 'CU3 ký tự vô hình đầu văn bản coi là chữ đứng trước'
m $F $'\t\t\t\tif !opt.Cell {' $'\t\t\t\tif true {' 'CU4 ô có cấu trúc cũng có thể chưa phân định'
m $F $'\tsp.Confidence = uncertainGroupConfidence' $'\tsp.Confidence = 1' 'CU5 chưa phân định mà độ tin cậy vẫn 1'
m $F $'\treturn strings.HasPrefix(sp.Rule, ruleSpaceThousandsUncertain)' $'\treturn false' 'UG1 UncertainGroup luôn false'
m $F $'\treturn SpaceGroup(sp) && sp.Start < nameEnd && sp.End > nameStart && (sp.Start < nameStart || sp.End > nameEnd)' $'\treturn false' 'SG1 GroupSplitsName luôn false'
m $F 'sp.End > nameStart && (sp.Start < nameStart || sp.End > nameEnd)' 'sp.End > nameStart' 'SG2 nhóm nằm trọn trong tên cũng chồng một phần'
m $F 'sp.Start < nameEnd && sp.End > nameStart &&' 'sp.Start <= nameEnd && sp.End >= nameStart &&' 'SG3 nhóm sát tên (không chung byte nào) cũng chồng'
m $F $'\treturn SpaceGroup(sp) && sp.Start < nameEnd' $'\treturn sp.Start < nameEnd' 'SG4 span không phải nhóm nghìn cũng chồng tên'
m $I $'\t\t\tif normalization.GroupSplitsName(sp, x[0], x[1]) {' $'\t\t\tif false {' 'GI1 nhóm chồng tên bị loại im lặng ("máy 3 850 lít, phiếu 912" thành 912 L)'
m $I $'\tif normalization.UncertainGroup(cands[0]) {' $'\tif false {' 'GI2 nạp "bể 2 500 lít" thành 2.500 L'
m $I 'strings.TrimSpace(qty+" "+c.field(r, "unit")), nil, true)' 'strings.TrimSpace(qty+" "+c.field(r, "unit")), nil, false)' 'GI3 ô có cấu trúc bị coi là chữ tự do ("khoảng 3 850" bị cách ly)'
m $I 'c.quantity(text, exclude, false)' 'c.quantity(text, exclude, true)' 'GI4 chữ tự do được miễn cờ chưa phân định như ô'
m $A $'\t\tif normalization.UncertainGroup(sp) {' $'\t\tif false {' 'GA1 luồng hỏi bỏ qua nhóm chưa phân định'
m $A 'normalization.GroupSplitsName(sp, m.start, m.end)' 'false' 'GA2 luồng hỏi bỏ nhóm chồng tên như số trong tên'
m $A 'u[0] >= lo && !slices.Contains(cs.groups, raw)' '!slices.Contains(cs.groups, raw)' 'GA3 nhóm ở mệnh đề ngữ cảnh cũng bắt hỏi lại'
m $A $'\tgroup := len(claims.groups) > 0' $'\tgroup := false' 'GA4 không hỏi lại khi nhóm chưa phân định'
m $A $'\t\t\tif _, ok := a.inMetricUnit(sp.Value, def); ok {' $'\t\t\tif true {' 'GA5 nhóm khác chiều đo chồng tên cũng bắt hỏi lại'
m $A $'\t\tif group {\n\t\t\tvisible, ci.shown' $'\t\tif false {\n\t\t\tvisible, ci.shown' 'GA6 hỏi lại về nhóm số vẫn kể máy ứng viên'
m $A $'\t\tif !slices.Contains(plan.Intent.Missing, "claimed") {' $'\t\tif false {' 'GA7 câu hỏi lại về nhóm số không khai khe "con số cần xác nhận"'
