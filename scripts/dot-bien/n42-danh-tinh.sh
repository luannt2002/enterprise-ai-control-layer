#!/usr/bin/env bash
# Danh mục đột biến cho cổng danh tính EACL (danh_tinh_test.go) và biến model tương thích (intelligence/models.modelEnv).
# Mỗi dòng: m <tệp> <chuỗi gốc> <chuỗi đột biến> <nhãn>. Kỳ vọng: mọi dòng "BỊ BẮT".
cd "$(dirname "$0")/../.." || exit 1
m() { scripts/mutate.sh "$1" "$2" "$3" "$4"; echo "rc=$?"; }
export PKGS=". ./intelligence/models/"
D=danh_tinh_test.go
m $D '`(?i)eais|enterprise[- ]ai[- ]systems`' '`(?i)eais`' 'DT1 mẫu bỏ tên repo'
m $D '				if a.max > 0 && n > a.max {' '				if false && a.max > 0 && n > a.max {' 'DT2 không kiểm số lần cho phép'
m $D '		return rel == "docs/sources" || rel == ".claude/worktrees"' '		return rel == ".claude/worktrees"' 'DT3 quét cả nguồn riêng ngoài repo'
m $D '	return strings.HasSuffix(rel, ".local.md")' '	return false' 'DT4 quét cả sổ ở máy'
m $D '		if bytes.IndexByte(b, 0) >= 0 { // tệp nhị phân' '		if false && bytes.IndexByte(b, 0) >= 0 { // tệp nhị phân' 'DT5 quét cả tệp nhị phân'
m $D '		bad = append(bad, rel+": "+strconv.Itoa(n)+" lần, không có lý do được ghi")' '		_ = n' 'DT6 tệp ngoài danh sách không bị báo'
M=intelligence/models/messages.go
m $M '	if v, ok := os.LookupEnv("EACL_MODEL" + name); ok {' '	if v := os.Getenv("EACL_MODEL" + name); v != "" {' 'ME1 EACL_ đặt rỗng không còn thắng'
m $M '	return os.Getenv(legacyModelEnvPrefix + name)' '	return ""' 'ME2 bỏ đọc dự phòng'
m $D '		return rel == "docs/sources" || rel == ".claude/worktrees"' '		return rel == "docs/sources"' 'DT7 quét cả worktree harness dựng trong repo'
m $D '		return rel == "docs/sources" || rel == ".claude/worktrees"' '		return rel == "docs/sources" || strings.HasPrefix(d.Name(), ".")' 'DT8 bỏ qua mọi thư mục chấm'
m $D '	if d.Name() == ".git" {' '	if false && d.Name() == ".git" {' 'DT9 quét cả mục .git (git không bao giờ theo dõi)'
