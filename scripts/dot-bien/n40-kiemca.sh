#!/usr/bin/env bash
# Danh mục đột biến cho kiemca (tenants/construction/win/kiemca): công cụ kiểm ĐỊNH DẠNG bộ giữ lại mà không in nội dung ca.
# Mỗi dòng: m <tệp> <chuỗi gốc> <chuỗi đột biến> <nhãn>. Kỳ vọng: mọi dòng "BỊ BẮT".
cd "$(dirname "$0")/../.." || exit 1
m() { scripts/mutate.sh "$1" "$2" "$3" "$4"; echo "rc=$?"; }
export PKGS="./tenants/construction/win/kiemca/"
K=tenants/construction/win/kiemca/main.go
m $K '	if problems > 0 {
		return 1
	}' '	if problems > 0 {
		return 0
	}' 'KC1 lỗi định dạng vẫn thoát 0'
m $K 'if prev, ok := seen[c.ID]; ok && prev != f {' 'if prev, ok := seen[c.ID]; false && ok && prev != f {' 'KC2 bỏ kiểm id trùng giữa các tệp'
m $K '				known[p.ID] = true' '				known[p.ID] = false' 'KC3 không nhận principal nào của tenant'
m $K '			fmt.Fprintf(out, "LỖI: %s không đọc được (%v)\n", f, err)
			return 2' '			fmt.Fprintf(out, "LỖI: %s không đọc được (%v)\n", f, err)
			continue' 'KC4 bỏ qua tệp không đọc được'
m $K '			fmt.Fprintf(out, "%s: %s\n", f, p)' '			fmt.Fprintf(out, "%s\n", f)
			_ = p' 'KC5 không báo lỗi theo mã ca + trường'
m $K '			perClass[c.FailureClass]++' '			perClass[c.FailureClass]++
			fmt.Fprintln(out, c.Question)' 'KC6 in câu hỏi của ca'
m $K '			if c.Expect.Value != nil {' '			if c.Expect.Value == nil {' 'KC7 đếm sai số ca có kỳ vọng số liệu'
