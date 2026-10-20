#!/usr/bin/env bash
# Danh mục đột biến "G5 — ràng buộc claim đủ 6 chiều" đã chạy ngày 20/10/2026 (xem docs/ops/NHAT-KY-2026-10-20.md).
# Mỗi dòng: m <tệp> <chuỗi gốc> <chuỗi đột biến> <nhãn>. Kỳ vọng: mọi dòng in "BỊ BẮT" (rc=0).
# Mã đổi sau ngày đó thì chuỗi gốc có thể không còn khớp (mutate.sh báo "xuất hiện 0 lần", rc=2):
# sửa chuỗi theo mã mới rồi chạy lại — đừng xoá dòng cho xanh.
cd "$(dirname "$0")/../.." || exit 1
export PKGS="./data/... ./core/... ./reliability/... ./tenants/..."
m() { scripts/mutate.sh "$1" "$2" "$3" "$4"; echo "rc=$?"; }
m core/pipeline/ask.go '		return append(v, validation.BindClaims(draft.Text, ignore[:2], draft.Claims, pkg.Evidence)...), nil' '		return v, nil' 'BC1 S9 bỏ ràng buộc đủ bộ'
m core/pipeline/ask.go '		if ev[i].Kind == contracts.EvidenceCalculation && ev[i].Subject == "" {' '		if false && ev[i].Kind == contracts.EvidenceCalculation && ev[i].Subject == "" {' 'BC2 không ghi chủ thể vào bằng chứng phép tính'
m core/pipeline/ask.go '	claims = append(claims, itemClaims...)' '	_ = itemClaims' 'BC3 bỏ claim của ghi chú xung đột'
m core/pipeline/ask.go '		claims = append(claims, contracts.Claim{Subject: subject, Metric: metric, Period: period, Value: &claimed,' '		_ = append(claims, contracts.Claim{Subject: subject, Metric: metric, Period: period, Value: &claimed,' 'BC4 bỏ claim số của người hỏi'
m core/pipeline/ask.go '		claims = append(claims, otherClaims...)' '		_ = otherClaims' 'BC5 bỏ claim các số khác cùng kỳ'
m reliability/validation/binding.go '	numRe = regexp.MustCompile(`(?:^|[^\p{L}\p{N}\-.,/])(-?\d{1,3}(?:\.\d{3})+(?:,\d+)?|-?\d+(?:,\d+)?)`)' '	numRe = regexp.MustCompile(`(?:^|[^\p{L}\p{N}\-.,/])(\d{1,3}(?:\.\d{3})+(?:,\d+)?|\d+(?:,\d+)?)`)' 'BC6 bộ đếm bỏ dấu âm'
m reliability/validation/binding.go '		if e.Subject != c.Subject {' '		if false && e.Subject != c.Subject {' 'BC7 bỏ chiều chủ thể'
m reliability/validation/binding.go '		if e.Metric != c.Metric {' '		if false && e.Metric != c.Metric {' 'BC8 bỏ chiều chỉ số'
m reliability/validation/binding.go '		if e.Period != c.Period {' '		if false && e.Period != c.Period {' 'BC9 bỏ chiều kỳ'
m reliability/validation/binding.go '		if e.Source.SourceType != c.Source {' '		if false && e.Source.SourceType != c.Source {' 'BC10 bỏ chiều nguồn'
m reliability/validation/binding.go '			if v.Unit == c.Value.Unit && v.Number != "" {' '			if v.Number != "" {' 'BC11 bỏ chiều đơn vị'
m reliability/validation/binding.go '	if cmp, err := decimal.Cmp(sum, c.Value.Number); err != nil || cmp != 0 {' '	if cmp, err := decimal.Cmp(sum, c.Value.Number); false && (err != nil || cmp != 0) {' 'BC12 bỏ chiều giá trị'
m reliability/validation/binding.go '		if !found {
			bad("NUMBER_UNBOUND", "số "+n+" trong câu không thuộc claim nào")' '		if false && !found {
			bad("NUMBER_UNBOUND", "số "+n+" trong câu không thuộc claim nào")' 'BC13 bỏ kiểm số trôi'
