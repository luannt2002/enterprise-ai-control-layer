#!/usr/bin/env bash
# Danh mục đột biến cho sửa S2-4 của người soát độc lập (luồng hỏi, ngày 20/10/2026): câu nêu NHIỀU thực thể / kỳ /
# con số khác nhau, hoặc một ngày hỏng, bị cắt thầm về vế đầu hoặc rơi về ngày mặc định (P5). Sửa nằm ở ba tầng —
# chuẩn hoá (ghi ngày hỏng), ý định (khe chỉ nhận MỘT giá trị), luồng hỏi (con số xác nhận tranh nhau, câu hỏi lại).
# Mỗi dòng: m <tệp> <chuỗi gốc> <chuỗi đột biến> <nhãn>. Kỳ vọng: mọi dòng in "BỊ BẮT" (rc=0).
# Chuỗi gốc là chuỗi con xuất hiện đúng 1 lần trong tệp. Mã đổi thì chuỗi gốc có thể không còn khớp (rc=2): sửa chuỗi
# theo mã mới — đừng xoá dòng cho xanh.
cd "$(dirname "$0")/../.." || exit 1
export PKGS="./core/pipeline/ ./intelligence/intent/ ./intelligence/router/ ./data/normalization/ ./tenants/construction/"
I=intelligence/intent/intent.go R=intelligence/router/router.go D=data/normalization/datetime.go E=data/normalization/engine.go A=core/pipeline/ask.go
m() { scripts/mutate.sh "$1" "$2" "$3" "$4"; echo "rc=$?"; }

# --- Ý định (intent.go fill / clashing / distinctPeriods): một khe chỉ nhận MỘT giá trị
m $I 'if clash := clashing(res, sd.Types); len(clash) > 0 {' 'if clash := clashing(res, sd.Types); false && len(clash) > 0 {' 'S2-4a bỏ kiểm nhiều thực thể cùng loại (lấy thực thể nhắc đầu)'
m $I 'if count[e.Type] > 1 {' 'if len(uniq) > 1 {' 'S2-4b hai thực thể KHÁC loại cũng tranh nhau (máy + dự án chứa nó)'
m $I 'if hasType(types, e.Type) && !seen[e.ID] {' 'if hasType(types, e.Type) {' 'S2-4c một thực thể nhắc hai lần bị tính là hai'
m $I '|| slices.ContainsFunc(m.Candidates, func(c contracts.EntityRef) bool { return seen[c.ID] }) {' '{' 'S2-4d bí danh mơ hồ chứa chính thực thể đã rõ bị tính là lần nhắc thứ hai'
m $I 'count[m.Type]++' '_ = count' 'S2-4e bí danh mơ hồ không được tính là một lần nhắc'
m $I 'candidates = nil // thực thể người hỏi viết rõ đang tranh nhau: không để bí danh mơ hồ làm router chọn thay' '_ = candidates' 'S2-4f giữ Candidates khi có tranh chấp (router gỡ thay người hỏi)'
m $I 'unclear.Entities = append(unclear.Entities, clash...)' '_ = clash' 'S2-4g không ghi thực thể tranh nhau vào Unclear'
m $I 'len(ps) > 1 || len(in.InvalidDates) > 0 {' 'len(ps) > 99 || len(in.InvalidDates) > 0 {' 'S2-4h bỏ kiểm nhiều kỳ (lấy kỳ nhắc đầu)'
m $I 'len(ps) > 1 || len(in.InvalidDates) > 0 {' 'len(ps) > 1 || len(in.InvalidDates) > 99 {' 'S2-4i bỏ kiểm ngày hỏng (rơi về ngày mặc định)'
m $I 'return o.Value.Text == sp.Value.Text' 'return false' 'S2-4j cùng một kỳ viết hai cách bị tính là hai'
m $I 'unclear.Periods = ps' 'unclear.Periods = nil' 'S2-4k không ghi các kỳ vào Unclear'
m $I 'unclear.Invalid = in.InvalidDates' 'unclear.Invalid = nil' 'S2-4l không ghi ngày hỏng vào Unclear'

# --- Router: lý do route nói đúng điều đã xảy ra
m $R 'if cls.Unclear.Any() {' 'if false {' 'S2-4m lý do route vẫn nói "thiếu khe" khi câu nêu nhiều giá trị'

# --- Chuẩn hoá (datetime.go matchNumericDate / badDate, engine.go Normalize): ghi ngày hỏng, không đổi span
m $D 'return contracts.Span{}, i, false, badDate(s, toks, i, j) // 31/2 không phải ngày' 'return contracts.Span{}, i, false, contracts.Span{} // 31/2 không phải ngày' 'S2-4n ngày không có trên lịch (31/02) không được ghi lại'
m $D 'if d < 0 || d > 99 || m < 0 || m > 99 {' 'if false {' 'S2-4o cụm hai số không có dạng ngày (100/10, 02/2026) bị ghi là ngày hỏng'
m $D 'if len(parts) == 3 && seps[0] == "/" && seps[1] == "/" && smallInt(parts[0]) && smallInt(parts[1]) && smallInt(parts[2]) {' 'if false {' 'S2-4p năm hai chữ số (16/9/26) không được ghi cả cụm'
m $D 'if len(parts) == 3 && seps[0] == "/" && seps[1] == "/" && smallInt(parts[0]) && smallInt(parts[1]) && smallInt(parts[2]) {' 'if len(parts) == 3 && smallInt(parts[0]) && smallInt(parts[1]) && smallInt(parts[2]) {' 'S2-4p2 mã tài liệu nối bằng "-" (TK-01-02-03) bị ghi là ngày hỏng'
m $D $'func badDate(s string, toks []token, i, j int) contracts.Span {\n\tstart, end := toks[i].start, toks[j-1].end' $'func badDate(s string, toks []token, i, j int) contracts.Span {\n\tstart, end := toks[i].start, toks[j-1].end\n\ts = s[:end]' 'S2-4q bỏ ranh giới từ sau cụm (31/02R bị ghi là ngày hỏng)'
m $E 'bad.Start >= invalidEnd {' '(bad.Start >= invalidEnd || true) {' 'S2-4r cụm nhỏ nằm trong cụm hỏng bị ghi lặp (9/26 của 16/9/26)'
m $E 'invalid, invalidEnd = append(invalid, bad.Raw), bad.End' 'invalid, invalidEnd = invalid, bad.End' 'S2-4s Normalize không ghi ngày hỏng'
m $E 'InvalidDates: invalid,' '' 'S2-4t Normalize không trả InvalidDates'

# --- Luồng hỏi (ask.go): câu xác nhận nêu nhiều con số khác nhau
m $A 'if len(a.rivalClaims(plan, in)) > 1 {' 'if len(a.rivalClaims(plan, in)) > 99 {' 'S2-4u bỏ kiểm nhiều con số khác nhau (xác nhận con số của mệnh đề hỏi)'
m $A 'from = fs' 'from = 0 * fs' 'S2-4v số ở mệnh đề ĐỨNG TRƯỚC mệnh đề hỏi (ngữ cảnh) cũng bị tính'
m $A $'	if fs, _, focus := intent.FocusClause(in.Raw, a.RT.ConfirmMarkers); focus {\n\t\tfrom = fs\n\t}' $'	if fs, fe, focus := intent.FocusClause(in.Raw, a.RT.ConfirmMarkers); focus {\n\t\tfrom = fs\n\t\tin.Spans = slices.DeleteFunc(slices.Clone(in.Spans), func(sp contracts.Span) bool { return sp.Start >= fe })\n\t}' 'S2-4w số ở các mệnh đề SAU mệnh đề hỏi bị bỏ qua (hành vi cũ)'
m $A 'v, ok := a.inMetricUnit(sp.Value, def)' 'v, ok := sp.Value, def.Unit != ""' 'S2-4x không đổi về đơn vị chuẩn trước khi so (4k và 4 m3 bị tính là hai số)'
m $A 'v, ok := a.inMetricUnit(sp.Value, def)' 'v, ok := a.inMetricUnit(sp.Value, def); if !ok { v, ok = sp.Value, true }' 'S2-4y số khác chiều đo (7h) bị tính là con số xác nhận'
m $A '(sp.Start < ent.End && sp.End > ent.Start) {' '(sp.Start < ent.End && sp.End > ent.Start && false) {' 'S2-4z số trong tên thực thể (máy xúc 03) bị tính là con số xác nhận'
m $A 'plan.Intent.Missing = append(slices.Clone(plan.Intent.Missing), "claimed")' '_ = slices.Clone(plan.Intent.Missing)' 'S2-4aa phản hồi không ghi khe claimed vào Missing'
m $A 'contracts.RouteClarification, "clarify_missing", contracts.OutcomeClarificationRequired, nil' 'contracts.RouteClarification, "clarify_missing", contracts.OutcomeResolved, nil' 'S2-4ab hỏi lại nhưng Outcome sai (RESOLVED)'
m $A '"claimed": "con số cần xác nhận"' '"claimed": "claimed"' 'S2-4ac câu hỏi lại gọi khe bằng mã thay vì nhãn'

# --- Luồng hỏi (ask.go): câu hỏi lại nêu đúng thứ đã nhận ra, không lộ thứ ngoài phạm vi, để lại vết
m $A 'notes := a.unclearNotes(cls.Unclear, now.Location())' 'notes := []string(nil)' 'S2-4ad câu hỏi lại không nêu các mốc / ngày hỏng'
m $A 'visible := a.visibleCandidates(ctx, pr, append(slices.Clone(cls.Candidates), cls.Unclear.Entities...))' 'visible := append(a.visibleCandidates(ctx, pr, cls.Candidates), cls.Unclear.Entities...)' 'S2-4ae ứng viên tranh nhau không qua lọc phạm vi (lộ tên máy ngoài phạm vi)'
m $A 'visible := a.visibleCandidates(ctx, pr, append(slices.Clone(cls.Candidates), cls.Unclear.Entities...))' 'visible := a.visibleCandidates(ctx, pr, cls.Candidates)' 'S2-4af câu hỏi lại không kèm các thực thể đã nhận ra'
m $A 'if !strings.Contains(d, sp.Raw) {' 'if false {' 'S2-4ag mốc thời gian không kèm chữ người hỏi đã viết ("hôm nay")'
m $A 'd = displayPeriod(from, to)' '_ = from; d = displayPeriod(to, to)' 'S2-4ah khoảng thời gian chỉ hiện ngày cuối'
m $A 'notes = append(notes, "Ngày "+strings.Join(u.Invalid, ", ")+" không hợp lệ.")' '_ = notes' 'S2-4ai câu hỏi lại không nói ngày nào không hợp lệ'
m $A 'for _, raw := range in.InvalidDates {' 'for _, raw := range in.InvalidDates[:0] {' 'S2-4aj vết quyết định không ghi ngày hỏng'
