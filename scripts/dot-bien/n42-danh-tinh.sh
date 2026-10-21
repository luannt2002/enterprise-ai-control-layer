#!/usr/bin/env bash
# Danh mục đột biến cho cổng danh tính EAIS (danh_tinh_test.go) và biến môi trường tương thích (models.modelEnv, pgstore.EnvDSN).
# Mỗi dòng: m <tệp> <chuỗi gốc> <chuỗi đột biến> <nhãn>. Kỳ vọng: mọi dòng "BỊ BẮT".
cd "$(dirname "$0")/../.." || exit 1
m() { scripts/mutate.sh "$1" "$2" "$3" "$4"; echo "rc=$?"; }
export PKGS=". ./intelligence/models/ ./platform/pgstore/"
D=danh_tinh_test.go
m $D '				if a.max > 0 && n > a.max {' '				if false && a.max > 0 && n > a.max {' 'DT2 không kiểm số lần cho phép'
m $D '		return rel == "docs/sources" || rel == ".claude/worktrees"' '		return rel == ".claude/worktrees"' 'DT3 quét cả nguồn riêng ngoài repo'
m $D '	return strings.HasSuffix(rel, ".local.md")' '	return false' 'DT4 quét cả sổ ở máy'
m $D '		if bytes.IndexByte(b, 0) >= 0 { // tệp nhị phân' '		if false && bytes.IndexByte(b, 0) >= 0 { // tệp nhị phân' 'DT5 quét cả tệp nhị phân'
m $D '		bad = append(bad, rel+": "+strconv.Itoa(n)+" lần, không có lý do được ghi")' '		_ = n' 'DT6 tệp ngoài danh sách không bị báo'
m $D '`(?i)eacl|enterprise[\p{Z}\t\n\v\f\r_.\-]*ai[\p{Z}\t\n\v\f\r_.\-]*control[\p{Z}\t\n\v\f\r_.\-]*layer`' '`(?i)eacl`' 'DT1 mẫu bỏ tên dài'
m $D '`(?i)eacl|enterprise[\p{Z}\t\n\v\f\r_.\-]*ai[\p{Z}\t\n\v\f\r_.\-]*control[\p{Z}\t\n\v\f\r_.\-]*layer`' '`(?i)eacl|enterprise[- ]ai[- ]control[- ]layer`' 'DT13 tên dài chỉ nối bằng gạch nối / một dấu cách (lọt gạch dưới, viết liền, dấu chấm…)'
m $D '	b = bytes.ReplaceAll(b, []byte(gitLink), []byte{0})' '	_ = gitLink' 'DT10 không bỏ link git trước khi đếm'
m $D '	b = bytes.ReplaceAll(b, []byte(gitLink), []byte{0})' '	b = regexp.MustCompile("(?i)"+gitLink).ReplaceAll(b, []byte{0})' 'DT11 bỏ cả tên dài viết hoa (rộng hơn link git)'
m $D '	b = bytes.ReplaceAll(b, []byte(gitLink), []byte{0})' '	b = bytes.ReplaceAll(b, []byte("github.com/luannt2002/"+gitLink), []byte{0})' 'DT12 chỉ bỏ dạng đường dẫn module (hẹp hơn link git)'
m $D '	b = bytes.ReplaceAll(b, []byte(gitLink), []byte{0})' '	b = bytes.ReplaceAll(b, []byte(gitLink), nil)' 'DT14 xoá link git thay vì thay bằng NUL (chữ hai bên dính thành khớp giả)'
m $D '				if a.max > 0 && n < a.max {' '				if false && a.max > 0 && n < a.max {' 'DT15 không báo hạn mức thừa'
m $D '		if a.max > 0 && !strings.HasSuffix(k, "/") && !seen[k] {' '		if false && a.max > 0 && !strings.HasSuffix(k, "/") && !seen[k] {' 'DT16 không báo mục cho phép đã hết lần'
m $D '		if r, _ := utf8.DecodeLastRune(b[:m[0]]); string(b[m[0]:m[1]]) == "eACL" && unicode.IsLetter(r) {' '		if r, _ := utf8.DecodeLastRune(b[:m[0]]); false && unicode.IsLetter(r) {' 'DT17 đếm cả chữ ACL cuối từ (RetrieveACL)'
m $D '		if r, _ := utf8.DecodeLastRune(b[:m[0]]); string(b[m[0]:m[1]]) == "eACL" && unicode.IsLetter(r) {' '		if r, _ := utf8.DecodeLastRune(b[:m[0]]); unicode.IsLetter(r) {' 'DT18 bỏ mọi lần khớp sau chữ cái (lọt sysEACL)'
M=intelligence/models/messages.go
m $M '	if v, ok := os.LookupEnv("EAIS_MODEL" + name); ok {' '	if v := os.Getenv("EAIS_MODEL" + name); v != "" {' 'ME1 EAIS_ đặt rỗng không còn thắng'
m $M '	return os.Getenv(legacyModelEnvPrefix + name)' '	return ""' 'ME2 bỏ đọc dự phòng'
m $D '		return rel == "docs/sources" || rel == ".claude/worktrees"' '		return rel == "docs/sources"' 'DT7 quét cả worktree harness dựng trong repo'
m $D '		return rel == "docs/sources" || rel == ".claude/worktrees"' '		return rel == "docs/sources" || strings.HasPrefix(d.Name(), ".")' 'DT8 bỏ qua mọi thư mục chấm'
m $D '	if d.Name() == ".git" {' '	if false && d.Name() == ".git" {' 'DT9 quét cả mục .git (git không bao giờ theo dõi)'
P=platform/pgstore/env.go
m $P '	if v, ok := os.LookupEnv("EAIS_PG_" + name); ok {' '	if v := os.Getenv("EAIS_PG_" + name); v != "" {' 'PE1 EAIS_PG_ đặt rỗng không còn thắng'
m $P '	return os.Getenv(legacyEnvPrefix + name)' '	return ""' 'PE2 DSN bỏ đọc dự phòng'
