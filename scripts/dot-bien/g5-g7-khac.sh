#!/usr/bin/env bash
# Danh mục đột biến "G5 AST tenant + G7 khung so sánh" đã chạy ngày 20/10/2026 (xem docs/ops/NHAT-KY-2026-10-20.md).
# Mỗi dòng: m <tệp> <chuỗi gốc> <chuỗi đột biến> <nhãn>. Kỳ vọng: mọi dòng in "BỊ BẮT" (rc=0).
# Mã đổi sau ngày đó thì chuỗi gốc có thể không còn khớp (mutate.sh báo "xuất hiện 0 lần", rc=2):
# sửa chuỗi theo mã mới rồi chạy lại — đừng xoá dòng cho xanh.
cd "$(dirname "$0")/../.." || exit 1
m() { scripts/mutate.sh "$1" "$2" "$3" "$4"; echo "rc=$?"; }
PKGS=. m core/contracts/contracts.go '	Text           string `json:"text"`
}' '	Text           string `json:"text"`
	Workspace      string `json:"tenant_id"`
}' 'TA AskRequest mang tenant (tên khác, thẻ JSON tenant)'
PKGS=. m core/contracts/contracts.go '	Text           string `json:"text"`
}' '	Text           string `json:"text"`
	TenantID       TenantID
}' 'TB AskRequest có TenantID'
export PKGS="./evals/..."
m evals/g7.go '	return c.Eligible && c.Total > 0 && (c.EAISPass-c.BaselinePass)*100 >= G7LeadPoints*c.Total' '	return c.Eligible && c.Total > 0 && (c.EAISPass-c.BaselinePass)*100 >= (G7LeadPoints-1)*c.Total' 'S1 ngưỡng hơn 19 điểm'
m evals/g7.go '		if have[n] && !asked[n] {' '		if have[n] && (!asked[n] || asked[n]) {' 'S2 rò rỉ tính cả số người hỏi đưa ra'
m evals/g7.go '	if v := e.Value; v != nil && !have[abs(v.Number)] {' '	if v := e.Value; v != nil && false {' 'S3 chấm nội dung bỏ kiểm số'
m evals/g7.go '	return n[(len(n)-1)/2]' '	return n[len(n)-1]' 'S4 trung vị thành lượt tốt nhất'
m evals/baseline/baseline.go '	reader := contracts.Principal{ID: pr.ID, TenantID: pr.TenantID, Kind: pr.Kind, Roles: []string{openRole}}' '	reader := pr' 'S5 baseline lọc theo vai thật'
m evals/g7.go '	abs := func(n string) string { return canonNumber(strings.TrimPrefix(strings.TrimSpace(n), "-")) }' '	abs := func(n string) string { return canonNumber(strings.TrimSpace(n)) }' 'S6 chấm nội dung giữ dấu âm'
