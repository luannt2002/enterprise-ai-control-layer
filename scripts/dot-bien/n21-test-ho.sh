#!/usr/bin/env bash
# Danh mục đột biến cho các chỗ TEST HỔNG mà bản soát mã độc lập (20/10 đêm) chỉ ra: mã đúng nhưng không
# test nào đỏ được khi mã sai (M21, M59, M23b, M24 + trần thời gian lùi, cận số lần thử lại).
# Mỗi dòng: m <tệp> <chuỗi gốc> <chuỗi đột biến> <nhãn>. Kỳ vọng: mọi dòng in "BỊ BẮT" (rc=0).
cd "$(dirname "$0")/../.." || exit 1
m() { scripts/mutate.sh "$1" "$2" "$3" "$4"; echo "rc=$?"; }
export PKGS="./intelligence/rag/"
m intelligence/rag/rag.go 'func allowed(d *Document, pr contracts.Principal) bool {' 'func allowed(d *Document, pr contracts.Principal) bool {
	if len(d.ACL) == 0 {
		return true
	}' 'M21 ACL rỗng = ai cũng đọc'
export PKGS="./evals/"
m evals/g7.go '		if t := strings.TrimSpace(s); reAllNum.MatchString(t) && have[canonNumber(t)] {' '		if t := strings.TrimSpace(s); reAllNum.MatchString(t) && have[t] {' 'M59 lộ số không chuẩn hoá'
export PKGS="./intelligence/models/"
F=intelligence/models/messages.go
m $F '		d *= 2' '		d *= 1' 'M24 thời gian lùi không nhân đôi'
m $F '		if d >= maxBackoff/2 {' '		if false {' 'L1 thời gian lùi không có trần'
m $F '		if serr := sleep(ctx, backoffFor(wait, attempt)); serr != nil {' '		if serr := sleep(ctx, backoffFor(wait, attempt)); serr != nil && false {' 'M23b bỏ qua huỷ ctx khi chờ lùi'
m $F '		if c.MaxRetries < 0 || c.MaxRetries > MaxRetriesLimit {' '		if c.MaxRetries < 0 {' 'L2 không có cận trên số lần thử lại'
