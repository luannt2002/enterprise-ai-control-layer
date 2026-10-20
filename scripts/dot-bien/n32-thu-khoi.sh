#!/usr/bin/env bash
# Danh mục đột biến cho sửa C-13 (soát độc lập lượt 2, chiều C): thử khói không ĐẠT khi hệ cổng bị chặn, và
# chạy khô mà đạt thì mã thoát 2 như run() — mã 0 chỉ khi nhà cung cấp thật nhận bộ tham số.
# Mỗi dòng: m <tệp> <chuỗi gốc> <chuỗi đột biến> <nhãn>. Kỳ vọng: mọi dòng "BỊ BẮT".
cd "$(dirname "$0")/../.." || exit 1
m() { scripts/mutate.sh "$1" "$2" "$3" "$4"; echo "rc=$?"; }
export PKGS="./tenants/construction/win/g7run/"
F=tenants/construction/win/g7run/main.go
m $F '	ok = ok && len(s.blocked) == 0' '	ok = ok && len(s.blocked) >= 0' 'SK1 hệ cổng bị chặn vẫn ĐẠT'
m $F '	case dry:
		return 2' '	case dry:
		return 0' 'SK2 chạy khô đạt trả mã 0'
m $F '	case !ok:
		return 1' '	case !ok && false:
		return 1' 'SK3 thử khói trượt không trả mã 1'
