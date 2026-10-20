#!/usr/bin/env bash
# Danh mục đột biến cho sửa C-03 (ca thiếu principal / id trùng không được âm thầm bỏ qua) và C-04 (dừng TRƯỚC lời gọi
# model khi -out đã có tệp hoặc lượt chạy thật có hệ cổng bị chặn) — soát độc lập lượt 2, chiều C.
# Không có dòng cho chỗ nối stopBeforeModel trong run(): nhánh chạy thật cần endpoint chính thức (mạng + khoá) nên
# không tới được trong test; hàm stopBeforeModel được test riêng.
cd "$(dirname "$0")/../.." || exit 1
m() { scripts/mutate.sh "$1" "$2" "$3" "$4"; echo "rc=$?"; }
export PKGS="./evals/"
m evals/runner.go '			bad("%sprincipal %q không có", AskErrorProblem, c.Principal)' '			bad("principal %q không có", c.Principal)' 'R1 ca thiếu principal không đánh dấu lỗi khi hỏi'
export PKGS="./tenants/construction/win/g7run/"
G=tenants/construction/win/g7run/main.go
# B1 (bỏ kiểm principal lạ trong vòng C-03 của build) nay TƯƠNG ĐƯƠNG: từ fbea42e, evals.ValidateWinSuite chạy trước cho từng
# tệp ca với đúng tập principal đó (lớp kiểm thứ hai, giữ có chủ ý). Đột biến của lớp ValidateWinSuite: PF10.
m $G '		if seen[c.ID] {' '		if false {' 'B2 nhận id ca trùng'
m $G '		if err := checkOutDir(cfg.out); err != nil {' '		if err := checkOutDir(cfg.out); err != nil && false {' 'O1 không kiểm -out trước khi gọi model'
m $G '	return len(blocked) > 0 && !fake && !dry' '	return false' 'S1 không dừng khi hệ cổng bị chặn'
# soát C2, F-02 / F-03 / F-05: lượt chạy THẬT bị từ chối TRƯỚC lời gọi model đầu tiên (test đi đường endpoint chính
# thức qua máy chủ TLS cục bộ: luot_that_test.go). Không đưa vào (tương đương, đã xét): bỏ nhánh "thiếu -out" (probeWritable("")
# cũng lỗi nên vẫn từ chối); bỏ kiểm k ∈ {3, 5} của lượt chính (khi bộ chấm chưa đóng băng, lượt chính đã bị từ chối vì lý do đó).
export PKGS="./tenants/construction/win/g7run/"
P=tenants/construction/win/g7run/main.go
m $P '	if !fake && !dry {
		if reasons := preflightReal(cfg, gw); len(reasons) > 0 {' '	if false && !fake && !dry {
		if reasons := preflightReal(cfg, gw); len(reasons) > 0 {' 'PF1 không kiểm trước lượt thật'
m $P '	if cfg.smoke > 0 {
		return reasons
	}' '	if false {
		return reasons
	}' 'PF2 thử khói bị đòi điều kiện của lượt chính'
m $P '		if !scorerFrozen {' '		if false && !scorerFrozen {' 'PF3 lượt chính chạy khi bộ chấm chưa đóng băng'
m $P '	} else if err := probeWritable(cfg.out); err != nil {' '	} else if err := probeWritable(cfg.out); false && err != nil {' 'PF4 -out không ghi được vẫn chạy'
m $P '	if g.MaxTokens != contractMaxTokens {' '	if false {' 'PF5 lệch max_tokens vẫn chạy'
m $P '	if g.Temperature != nil {' '	if false {' 'PF6 gửi temperature vẫn chạy'
m $P '	if g.MaxRetries != contractMaxRetries {' '	if false {' 'PF7 lệch số lần thử lại vẫn chạy'
m $P '	if cfg.topK != baseline.DefaultTopK {' '	if false {' 'PF8 lệch top-k vẫn chạy'
m $P '	if cfg.grade != gradeContent {
		reasons = append(reasons,' '	if false {
		reasons = append(reasons,' 'PF9 chấm full vẫn chạy thật'
m $P '		if probs := evals.ValidateWinSuite(suite, cfg.tenant, known); len(probs) > 0 {' '		if probs := evals.ValidateWinSuite(suite, cfg.tenant, known); false && len(probs) > 0 {' 'PF10 bộ ca sai định dạng vẫn chạy'
