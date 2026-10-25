#!/usr/bin/env bash
# Danh mục đột biến "L6 — ranh giới RAG" (báo cáo audit vNext 24/10/2026: F-019, F-037, F-018, F-017, F-023, F-024, F-022, F-028).
# Mỗi dòng: m <tệp> <chuỗi gốc> <chuỗi đột biến> <nhãn>. Kỳ vọng: mọi dòng in "BỊ BẮT" (rc=0). Đọc rc từng dòng (danh mục này không
# đếm kết cục). Dòng cần PG (nhãn có [PG]) chỉ đo được khi đặt EAIS_PG_ADMIN_DSN / EAIS_PG_DSN; thiếu thì test PG bị bỏ qua và đột
# biến "sống sót" giả. L6-23, L6-24 đột biến tệp DỮ LIỆU của ví dụ (đọc lúc chạy test).
# Mã đổi thì chuỗi gốc có thể không còn khớp (rc=2): sửa chuỗi theo mã mới rồi chạy lại — đừng xoá dòng cho xanh.
cd "$(dirname "$0")/../.." || exit 1
m() { scripts/mutate.sh "$1" "$2" "$3" "$4"; echo "rc=$?"; }
export PKGS="./platform/pgstore/ ./cmd/eais/ ./intelligence/rag/ ./examples/01-knowledge-base/"
L=platform/pgstore/load.go
m $L '	return s.load(ctx, info, true, nil, nil, docs)' '	return s.load(ctx, info, false, nil, nil, docs)' 'L6-1 [PG] LoadDocuments thay cả bốn bảng'
m $L '		tables = []string{"documents"}' '		tables = []string{"documents", "facts"}' 'L6-2 [PG] LoadDocuments xoá cả fact'
m $L '"DELETE FROM "+table+" WHERE tenant_id = $1", string(tenant)' '"DELETE FROM "+table' 'L6-3 [PG] Load xoá không lọc tenant_id'
m $L '		if slices.ContainsFunc(docs, func(d rag.Document) bool { return d.Hold != "" }) {' '		if false && slices.ContainsFunc(docs, func(d rag.Document) bool { return d.Hold != "" }) {' 'L6-10 [PG] Load không giữ nguyên bản đang công bố'
A=cmd/eais/app.go
m $A '		case d.Hold != "":
			rep.Holds = append(rep.Holds, at+d.Hold)' '		case false:
			rep.Holds = append(rep.Holds, at+d.Hold)' 'L6-5 báo cáo nạp không đếm bản bị giữ'
m $A '	if rep.Held = len(rep.Holds); rep.Held > 0 {' '	if rep.Held = len(rep.Holds); false {' 'L6-6 không cảnh báo bản bị giữ trong log'
m $A '		case gate[i] != "":' '		case false:' 'L6-11 [PG] báo cáo không nêu bản được giữ nguyên công bố'
R=intelligence/rag/rag.go
m $R '		if d := &docs[i]; d.Status == StatusNeedsReview && d.Hold != "" && live[key(d)] {' '		if d := &docs[i]; false && d.Status == StatusNeedsReview && d.Hold != "" && live[key(d)] {' 'L6-7 [PG] KeepPublished không giữ nguyên bản nào'
m $R 'return fmt.Sprintf("%s@%d\x00%s\x00%s", d.ID, d.Version, d.Title, d.Body) }' 'return fmt.Sprintf("%s@%d\x00%s", d.ID, d.Version, d.Title) }' 'L6-8 [PG] KeepPublished không so thân'
m $R '		if p := &published[i]; p.Status == StatusApproved && p.Hold == "" {' '		if p := &published[i]; true {' 'L6-9 [PG] KeepPublished coi bản đang bị giữ là đã công bố'
m $R '		if (d.Status == "" || d.Status == StatusApproved) && !strings.ContainsFunc(d.Body, unicode.IsLetter) {' '		if false && (d.Status == "" || d.Status == StatusApproved) && !strings.ContainsFunc(d.Body, unicode.IsLetter) {' 'L6-15 tệp 0 chữ nạp im lặng'
m $R '		if (d.Status == "" || d.Status == StatusApproved) && !strings.ContainsFunc(d.Body, unicode.IsLetter) {' '		if d.Status == StatusApproved && !strings.ContainsFunc(d.Body, unicode.IsLetter) {' 'L6-16 luật 0 chữ bỏ sót tài liệu không phiên bản'
m $R '			if strings.HasPrefix(strings.TrimSpace(l), "|") && !inTable[n] {' '			if false && strings.HasPrefix(strings.TrimSpace(l), "|") && !inTable[n] {' 'L6-17 bảng mất phân cách ở bản đầu được công bố'
m $R '			inTable[it.hseq], inTable[it.hseq+1], inTable[it.seq] = true, true, true' '			inTable[it.seq] = true' 'L6-18 header / dòng phân cách bị coi là dòng lẻ'
m $R '	return (d.Status == "" || d.Status == StatusApproved) &&
		(d.EffectiveFrom' '	return true &&
		(d.EffectiveFrom' 'L6-22 bản nháp được phục vụ (ví dụ trung tính đỏ)'
M=intelligence/rag/semantic.go
m $M '		if !releasable(ix.chunks[i].doc) {' '		if false && !releasable(ix.chunks[i].doc) {' 'L6-12 nhúng cả bản chưa duyệt'
m $M '		if vis[c.doc] && releasable(c.doc) && (' '		if vis[c.doc] && (' 'L6-13 reranker nhận đoạn của bản chưa duyệt'
m $M 'func releasable(d *Document) bool { return d.Status == "" || d.Status == StatusApproved }' 'func releasable(d *Document) bool { return d.Status != StatusDraft }' 'L6-14 coi bản chờ soát là được rời máy'
V=platform/pgstore/vector.go
m $V '"DELETE FROM document_vectors WHERE tenant_id = $1 AND space <> $2", string(tenant), keep)' '"DELETE FROM document_vectors WHERE tenant_id = $1 AND space = $2", string(tenant), keep)' 'L6-19 [PG] dọn nhầm không gian đang dùng'
m $V '"DELETE FROM document_vectors WHERE tenant_id = $1 AND space <> $2", string(tenant), keep)' '"DELETE FROM document_vectors WHERE (tenant_id = $1 OR true) AND space <> $2", string(tenant), keep)' 'L6-20 [PG] dọn không lọc tenant_id'
m $V '	if keep == "" {
		return 0, contracts.Errorf' '	if false && keep == "" {
		return 0, contracts.Errorf' 'L6-21 [PG] keep rỗng xoá hết'
E=examples/01-knowledge-base/docs
m $E/acl.json '"QT-BAO-MAT": ["quan-tri"]' '"QT-BAO-MAT": ["quan-tri", "nhan-vien"]' 'L6-23 lộ tài liệu bảo mật cho nhân viên'
m $E/nghi-phep-v1.md 'effective_to: "2026-01-01"' 'effective_to: "2027-01-01"' 'L6-24 hai bản nghỉ phép cùng hiệu lực'
