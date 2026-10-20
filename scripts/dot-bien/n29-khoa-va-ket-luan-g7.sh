#!/usr/bin/env bash
# Danh mục đột biến cho hai sửa theo soát độc lập lượt 2, chiều C: C-18 (khoá không theo chuyển hướng / http không
# loopback; EAIS_MODEL_MAX_TOKENS hỏng là lỗi) và C-06 (G7 chỉ ĐẠT khi bộ chấm đóng băng, chấm nội dung, đúng hợp đồng).
# Không có dòng cho chỗ nối verdict trong run(): mọi test của run() dùng model giả hoặc chạy khô (endpoint chính thức
# cần mạng + khoá) nên nhánh đó không tới được trong test — đã ghi là giới hạn.
cd "$(dirname "$0")/../.." || exit 1
m() { scripts/mutate.sh "$1" "$2" "$3" "$4"; echo "rc=$?"; }
export PKGS="./intelligence/models/"
F=intelligence/models/messages.go
m $F ', CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}' '}' 'K1 đi theo chuyển hướng (gửi lại khoá)'
m $F '		return ip != nil && ip.IsLoopback()' '		return ip != nil || true' 'K2 nhận http không phải loopback'
m $F '		if !secureEndpoint(m.BaseURL) {' '		if false {' 'K3 bỏ kiểm endpoint'
m $F '			c.MaxTokens = -1 // New báo lỗi cấu hình, không lặng lẽ về mặc định' '			c.MaxTokens = 0' 'K4 MAX_TOKENS hỏng về mặc định'
export PKGS="./tenants/construction/win/g7run/"
G=tenants/construction/win/g7run/main.go
m $G '	case !in.frozen:' '	case false:' 'V1 kết luận khi bộ chấm chưa đóng băng'
m $G '	case !in.contentGrade:' '	case false:' 'V2 kết luận khi chấm full'
m $G '	case !in.contractParams:' '	case false:' 'V3 kết luận khi lệch hợp đồng'
m $G 'const scorerFrozen = false' 'const scorerFrozen = true' 'V4 bật bộ chấm đóng băng'
