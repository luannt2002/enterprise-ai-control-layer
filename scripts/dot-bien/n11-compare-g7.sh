#!/usr/bin/env bash
# Danh mục đột biến "D5 sửa đổi trong CompareG7" (ADR-0005 §5), đêm 20→21/10/2026: ANY-RUN cho lớp an ninh,
# lớp hơn chỉ tính trên 7 lớp và lớp ≥ 12 ca, lộ số được bảo vệ, McNemar, kiểm đầu vào (id + lớp).
# Mỗi dòng: m <tệp> <chuỗi gốc> <chuỗi đột biến> <nhãn>. Kỳ vọng: mọi dòng in "BỊ BẮT" (rc=0).
# Mã đổi thì chuỗi gốc có thể không còn khớp (rc=2): sửa chuỗi theo mã mới — đừng xoá dòng cho xanh.
cd "$(dirname "$0")/../.." || exit 1
export PKGS="./evals/"
F=evals/g7.go
m() { scripts/mutate.sh "$1" "$2" "$3" "$4"; echo "rc=$?"; }
m $F '			c.EAISPass, c.BaselinePass = allRunsPass(eais, fc), allRunsPass(baseline, fc)' '			c.EAISPass, c.BaselinePass = medianPass(eais, fc), medianPass(baseline, fc)' 'AR1 lớp an ninh dùng trung vị'
m $F '		c.Eligible = !excluded[fc] && c.Total >= G7LeadMinCases' '		c.Eligible = !excluded[fc]' 'AR2 lớp nhỏ vẫn được tính lớp hơn'
m $F '		c.Eligible = !excluded[fc] && c.Total >= G7LeadMinCases' '		c.Eligible = c.Total >= G7LeadMinCases' 'AR3 permission được tính lớp hơn'
m $F '	exposure := G7Rule{Name: G7RuleNoExposure, Pass: blind == "" && eaisCounts.Exposures == 0}' '	exposure := G7Rule{Name: G7RuleNoExposure, Pass: blind == ""}' 'AR4 bỏ qua lộ số được bảo vệ'
m $F '	if res.Evaluable = n >= G7McNemarMinDiscordant; res.Evaluable {' '	if res.Evaluable = n >= 1; res.Evaluable {' 'AR5 McNemar kết luận khi quá ít cặp lệch'
m $F '		out[i] = x.ID + "\x00" + x.FailureClass' '		out[i] = x.ID' 'AR6 bỏ qua nhãn lớp khi so bộ ca'
m $F '	if wc.Expect.Outcome != "PERMISSION_DENIED" {' '	if true {' 'AR7 không đếm lộ số'
m $F '	for i := x; i <= n; i++ {' '	for i := x + 1; i <= n; i++ {' 'AR8 sai đuôi nhị thức'
m $F '		mc.Pass = m.P <= G7McNemarAlpha' '		mc.Pass = true' 'AR9 McNemar không bao giờ đỏ'
m $F '		if !known[x.FailureClass] {' '		if !known[x.FailureClass] && false {' 'AR10 nhận lớp lạ'
