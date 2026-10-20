#!/usr/bin/env bash
# Danh mục đột biến cho G1 (bộ nạp + bộ kiểm manifest checkG1) và G2 (resolver), để điều kiện G6' "mỗi cổng có đột
# biến bị bắt" chạy lại được bằng lệnh (đột biến G1/G2 ngày 20/10 là chạy tay, không lưu danh mục).
# Mỗi dòng: m <tệp> <chuỗi gốc> <chuỗi đột biến> <nhãn>. Kỳ vọng: mọi dòng "BỊ BẮT".
cd "$(dirname "$0")/../.." || exit 1
m() { scripts/mutate.sh "$1" "$2" "$3" "$4"; echo "rc=$?"; }
export PKGS="./data/... ./tenants/construction/win/winrun/ ./tenants/construction/"
I=data/ingestion/ingest.go
m $I '		if !ok {
			quarantine(ReasonUnknownEntity,' '		if false && !ok {
			quarantine(ReasonUnknownEntity,' 'G1a mã thiết bị lạ không bị cách ly'
m $I '		if qty == "" {' '		if false && qty == "" {' 'G1b ô lượng trống không bị cách ly đúng lý do'
m $I '	if err := c.def.InRange(value.Number); err != nil {' '	if err := c.def.InRange(value.Number); false && err != nil {' 'G1c lượng ngoài khoảng không bị cách ly'
W=tenants/construction/win/winrun/winrun.go
m $W '			if o.Entity != r.Entity || o.Metric != r.Semantic' '			if o.Metric != r.Semantic' 'G1d bộ kiểm bỏ so thực thể'
m $W '			if !ok || q.Reason != r.Reason {' '			if !ok {' 'G1e bộ kiểm bỏ so lý do cách ly'
m $W '		if !seen[k] {
			problems = append(problems, k+": quan sát THỪA, manifest không có")' '		if false && !seen[k] {
			problems = append(problems, k+": quan sát THỪA, manifest không có")' 'G1f bộ kiểm không bắt quan sát THỪA'
m $W '		if !seen[k] {
			problems = append(problems, k+": cách ly THỪA, manifest không có")' '		if false && !seen[k] {
			problems = append(problems, k+": cách ly THỪA, manifest không có")' 'G1g bộ kiểm không bắt cách ly THỪA'
R=data/entityresolution/resolver.go
m $R '			if len(cands) == 1 {' '			if len(cands) >= 1 {' 'G2a bí danh mơ hồ bị gỡ thầm về ứng viên đầu'
m $R '		for n := min(maxLen, len(words)-i); n >= 1; n-- {' '		for n := 1; n <= min(maxLen, len(words)-i); n++ {' 'G2b khớp cụm ngắn trước cụm dài'
