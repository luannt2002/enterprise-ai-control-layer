#!/usr/bin/env bash
# Danh mục đột biến "g7run: B2+ / B2″ / B1, bằng chứng thô -out, thử khói, luật kết luận" (ADR-0005 §2, §5–7), đêm 20→21/10/2026.
# Mỗi dòng: m <tệp> <chuỗi gốc> <chuỗi đột biến> <nhãn>. Kỳ vọng: mọi dòng in "BỊ BẮT" (rc=0).
# Không đưa vào danh mục (đột biến tương đương, đã xét): kiểm "HTTP cuối ≠ 200" của smoke — client hiện
# luôn trả lỗi khi mã khác 200, nên nhánh "có lỗi" đã bắt; giữ kiểm đó để phòng client đổi.
# Mã đổi thì chuỗi gốc có thể không còn khớp (rc=2): sửa chuỗi theo mã mới — đừng xoá dòng cho xanh.
cd "$(dirname "$0")/../.." || exit 1
export PKGS="./tenants/construction/win/g7run/"
S=tenants/construction/win/g7run/systems.go E=tenants/construction/win/g7run/evidence.go M=tenants/construction/win/g7run/main.go
m() { scripts/mutate.sh "$1" "$2" "$3" "$4"; echo "rc=$?"; }
m $S '		parts = append(parts, "bí danh: "+strings.Join(manual, ", "))' '		_ = manual' 'G1 B2+ mất bí danh'
m $S '		parts = append(parts, "mã theo nguồn: "+strings.Join(codes, ", "))' '		_ = codes' 'G2 B2+ mất mã theo nguồn'
m $S 'entityInfo(byID, o.Entity)' 'o.Entity' 'G3 B2+ không viết dữ liệu gốc'
m $S '		if r.On.IsZero() || (w != nil && !r.On.Before(w.from) && !r.On.After(w.to)) {' '		if r.On.IsZero() || w == w {' 'G4 B1 bỏ cửa sổ ngày'
m $S '		if r.On.IsZero() || (w != nil && !r.On.Before(w.from) && !r.On.After(w.to)) {' '		if w != nil && !r.On.Before(w.from) && !r.On.After(w.to) {' 'G5 B1 bỏ bản ghi không đọc được ngày'
m $S '	if len(missing) > 0 {' '	if false {' 'G6 B1 đoán khi ca thiếu cửa sổ'
m $S '		q = d.preamble + "\n" + question' '		q = question' 'G7 B1 không biết ngày hiện tại'
m $S '		if m.Tolerance != "" {' '		if false {' 'G8 B2″ mất dung sai'
m $E '		s.cur.Calls = append(s.cur.Calls, c)' '		_ = c' 'G9 không ghi vết lời gọi model'
m $E 'strings.Join(r.Citations, ",")}, " | ")' '""}, " | ")' 'G10 lượt vô hiệu bỏ qua trích dẫn'
m $E '			c.USD = (float64(c.InputTokens)*price[0] + float64(c.OutputTokens)*price[1]) / 1e6' '			c.USD = (float64(c.InputTokens)*price[1] + float64(c.OutputTokens)*price[0]) / 1e6' 'G11 đảo giá vào / ra'
m $E '		if r.System != sysEACL || r.ModelError {' '		if r.ModelError {' 'G12 baseline ngẫu nhiên bị tính là lượt vô hiệu'
m $M '	if len(blocked) > 0 || len(unstable) > 0 || len(invalid) > 0 {' '	if len(unstable) > 0 || len(invalid) > 0 {' 'G13 bỏ qua hệ cổng bị chặn'
m $M '	if len(blocked) > 0 || len(unstable) > 0 || len(invalid) > 0 {' '	if len(blocked) > 0 || len(invalid) > 0 {' 'G14 bỏ qua lượt vô hiệu (đường tất định)'
m $M '		if !slices.Contains(order, g) || !ok || !v.Pass {' '		if !slices.Contains(order, g) || !ok || !v.Pass && false {' 'G15 bỏ qua phán quyết của hệ cổng'
m $M '			bad := r.Error != ""' '			bad := false' 'G16 thử khói bỏ qua lỗi'
m $M '			case sysB2Plus:
				b.Preamble = todayLine(ref)
' '			case sysB2Plus:
				_ = ref
' 'G17 B2+ không biết ngày hiện tại'
m $E '			if n > 0 && n < k {' '			if n > 0 && n <= k {' 'G20 ca đạt mọi lượt bị tính chập chờn'
m $E '		if len(flaky[g]) >= 2 {' '		if len(flaky[g]) >= 3 {' 'G21 ngưỡng pilot sai'
m $E '	for _, g := range append([]string{sysEACL}, gateSystems...) {' '	for _, g := range append([]string{sysEACL, sysB2}, gateSystems...) {' 'G22 hệ chỉ-báo-cáo quyết k'
m $M '	if len(blocked) > 0 || len(unstable) > 0 || len(invalid) > 0 {' '	if len(blocked) > 0 || len(unstable) > 0 {' 'G23 bỏ qua lượt vô hiệu vì lỗi gọi model'
m $E '		if failed[k]*100 > 2*need[k] {' '		if failed[k]*100 > 50*need[k] {' 'G24 ngưỡng lỗi gọi model 50% thay vì 2%'
m $E '	rec.ModelError = lastFailed || (err != nil && (r.sys != sysEACL || strings.HasPrefix(contracts.FaultOf(err), "third_party:")))' '	rec.ModelError = lastFailed && false || err != nil && false' 'G25 không đánh dấu lỗi gọi model'
m $E '		if r.System != sysEACL || r.ModelError {' '		if r.System != sysEACL {' 'G26 lỗi gọi model tính là bất ổn đường tất định'
m $E '	if r.ModelCalls > 0 {
		return r.Route
	}' '	if false {
		return r.Route
	}' 'G27 so trích dẫn do model chọn'
m $E '	for _, g := range append([]string{sysEACL}, gateSystems...) {' '	for _, g := range gateSystems {' 'G28 pilot bỏ EACL'
m $E '	if s.capUSD > 0 && s.spentUSD >= s.capUSD {' '	if false && s.capUSD > 0 && s.spentUSD >= s.capUSD {' 'G29 không ghi chạm trần'
m $E '	if spent, hit := g.sink.spent(); hit {' '	if spent, hit := g.sink.spent(); hit && false {' 'G30 vẫn gọi model sau khi chạm trần'
m $M '		dry, costly = !isOfficialEndpoint(m.BaseURL), !models.LoopbackEndpoint(m.BaseURL)' '		dry, costly = false, !models.LoopbackEndpoint(m.BaseURL)' 'G31 chạy khô trông như lượt thật'
m $E '	if ents, err := os.ReadDir(dir); err == nil && len(ents) > 0 {' '	if ents, err := os.ReadDir(dir); err == nil && len(ents) > 99 {' 'G32 ghi đè bằng chứng cũ'
m $E '		if filepath.IsAbs(f) { // không ghi đường dẫn tuyệt đối của máy vào bằng chứng công khai' '		if false && filepath.IsAbs(f) {' 'G33 lộ đường dẫn tuyệt đối'
m $M '						layer = "PROVIDER" // máy gán: ca trượt vì lỗi gọi model, không vì câu trả lời' '						layer = "CHƯA PHÂN LOẠI"' 'G34 không gán nhãn PROVIDER'
m $S '		case seen[k]:' '		case false && seen[k]:' 'G35 nhận case_id trùng (kiểm ở caseEntries)'
m $S '		if r.On.IsZero() || (w != nil && !r.On.Before(w.from) && !r.On.After(w.to)) {' '		if r.On.IsZero() || (w != nil && !r.On.Before(w.from)) {' 'G36 B1 bỏ cận trên cửa sổ'
m $E '	return strings.Join([]string{r.Route, r.Outcome, strings.Join(r.Candidates, ","), strings.Join(r.Conflicts, ","), strings.Join(r.Citations, ",")}, " | ")' '	return strings.Join([]string{r.Route, strings.Join(r.Citations, ",")}, " | ")' 'G37 path bỏ outcome / ứng viên / xung đột'
export PKGS="./evals/baseline/"
B=evals/baseline/baseline.go
m $B '		mq = b.Preamble + "\n" + question' '		mq = question' 'G18 phần mở đầu không tới model'
m $B '	ev, err := b.index.Retrieve(ctx, reader, question, math.MaxInt)' '	ev, err := b.index.Retrieve(ctx, reader, b.Preamble+" "+question, math.MaxInt)' 'G19 truy xuất dùng cả phần mở đầu'
