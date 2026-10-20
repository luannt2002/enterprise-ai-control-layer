#!/usr/bin/env bash
# Danh mục đột biến "G3 / G4 — so sánh hai nghĩa, mệnh đề trọng tâm, số âm" đã chạy ngày 20/10/2026 (xem docs/ops/NHAT-KY-2026-10-20.md).
# Mỗi dòng: m <tệp> <chuỗi gốc> <chuỗi đột biến> <nhãn>. Kỳ vọng: mọi dòng in "BỊ BẮT" (rc=0).
# Mã đổi sau ngày đó thì chuỗi gốc có thể không còn khớp (mutate.sh báo "xuất hiện 0 lần", rc=2):
# sửa chuỗi theo mã mới rồi chạy lại — đừng xoá dòng cho xanh.
cd "$(dirname "$0")/../.." || exit 1
export PKGS="./data/... ./core/... ./intelligence/... ./reliability/... ./tenants/construction/ ./tenants/construction/win/winrun/"
m() { scripts/mutate.sh "$1" "$2" "$3" "$4"; echo "rc=$?"; }
m intelligence/intent/intent.go '	return slices.ContainsFunc(s.opt.ComparisonMarkers, func(m string) bool { return normalization.ContainsPhrase(text, m) })' '	return false && slices.ContainsFunc(s.opt.ComparisonMarkers, func(m string) bool { return normalization.ContainsPhrase(text, m) })' 'CM1 không nhận cụm so sánh'
m intelligence/intent/intent.go '		if first[right.Name] < first[left.Name] {' '		if false && first[right.Name] < first[left.Name] {' 'CM2 bỏ thứ tự nhắc'
m core/pipeline/ask.go '			return a.RT.Metrics.Compare(ctx, a.Deps.Facts, metric, cm, ids, from, to)' '			return a.RT.Metrics.Query(ctx, a.Deps.Facts, sourceoftruth.MetricQuery{Metric: metric, EntityIDs: ids, From: from, To: to})' 'CM3 pipeline không tính hiệu'
m intelligence/router/router.go '		plan.Route, plan.Template = contracts.RouteCalculation, "compare_total"' '		plan.Template = "compare_total"' 'CM4 router không đổi sang phép tính'
m intelligence/intent/intent.go '	return start, end, start > 0 || strings.TrimSpace(strings.TrimLeft(text[end:], ",;.?!")) != ""' '	return start, end, false' 'FC1 tắt mệnh đề trọng tâm'
m core/pipeline/ask.go '		lo, hi = fs, fe' '		lo, hi = 0, fe' 'FC2 số người hỏi lấy cả câu'
m data/normalization/text.go '		case unicode.IsDigit(r) || isSignedNumberAt(s, i):' '		case unicode.IsDigit(r):' 'NEG1 không đọc dấu âm'
m data/normalization/text.go '	return unicode.IsSpace(prev) || prev == '"'"'('"'"'' '	return unicode.IsSpace(prev) || prev == '"'"'('"'"' || prev != 0' 'NEG2 gạch nối nào cũng là dấu âm'
