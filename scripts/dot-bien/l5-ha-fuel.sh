#!/usr/bin/env bash
# Danh mục đột biến "L5 — hạ Fuel xuống case study tại chỗ" (báo cáo audit vNext 24/10/2026: F-004, F-009, F-008, F-005, F-007, F-029).
# Mỗi dòng: m <tệp> <chuỗi gốc> <chuỗi đột biến> <nhãn>. Kỳ vọng: mọi dòng in "BỊ BẮT" (rc=0). Đọc rc từng dòng (danh mục này không
# đếm kết cục). Dòng cần PG (nhãn có [PG]) chỉ đo được khi đặt EAIS_PG_ADMIN_DSN / EAIS_PG_DSN; thiếu thì test PG bị bỏ qua và đột
# biến "sống sót" giả.
# Mã đổi thì chuỗi gốc có thể không còn khớp (rc=2): sửa chuỗi theo mã mới rồi chạy lại — đừng xoá dòng cho xanh.
cd "$(dirname "$0")/../.." || exit 1
m() { scripts/mutate.sh "$1" "$2" "$3" "$4"; echo "rc=$?"; }
export PKGS="./data/ingestion/"
I=data/ingestion/ingest.go
S=data/ingestion/sources.go
m $I '			e := entityresolution.Entity{ID: r.ID, Type: l.Type, Key: r.ID, Name: r.Name}' '			e := entityresolution.Entity{ID: r.ID, Type: "x", Key: r.ID, Name: r.Name}' 'L5-1 loại thực thể không theo lược đồ của pack'
m $I '			if l.ParentField != "" {' '			if false {' 'L5-2 bỏ quan hệ cha của lược đồ'
m $I '	res, err := entityresolution.Resolve(c.ctx, c.dir, in, c.free.Type)' '	res, err := entityresolution.Resolve(c.ctx, c.dir, in)' 'L5-3 chữ tự do nhận dạng mọi loại thực thể'
m $I '"tin không nhắc " + c.free.Label + " nào"' '"tin không nhắc thực thể nào"' 'L5-4 lý do cách ly không gọi tên loại của pack'
m $S '	if len(sm.EntityLists) == 0 {' '	if false {' 'L5-5 thiếu lược đồ danh mục vẫn nạp'
m $S '	} else if freeText && !types[fe.Type] {' '	} else if false {' 'L5-6 chữ tự do trỏ loại chưa khai vẫn nạp'
