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
m $I '	if c.seen[key] {' '	if false {' 'L5-7 bản ghi trùng khoá thành fact thứ hai cùng id (F-009)'
B=data/ingestion/bridge.go
m $B ', ObservedAt: o.SentAt, ReportedBy: o.Sender},' '},' 'L5-8 fact mất người gửi và giờ gửi (F-008; cũng đỏ TestFactGiuNguonGocQuaPG khi có PG)'
m $I '		on, at = d, t' '		on, at = d, time.Time{}' 'L5-9 fact mất giờ gửi (F-008)'
export PKGS="./intelligence/router/ ./core/pipeline/ ./core/tenant/ ./tenants/construction/ ./examples/02-sme-minimal/"
R=intelligence/router/router.go
m $R '		if parent.Type == "" || cur.Entity.Type != parent.Type {' '		if cur.Entity.Type != parent.Type {' 'L5-10 pack không khai kiểu cha vẫn gỡ theo cha (F-005)'
m $R '		if parent.Type == "" || cur.Entity.Type != parent.Type {' '		if parent.Type == "" {' 'L5-11 gỡ theo cha với khe điền bằng kiểu khác (F-005)'
m $R '		nar.By, nar.ByLabel = parent.Type, parent.Label' '		nar.By, nar.ByLabel = parent.Type, "dự án"' 'L5-12 vết gọi kiểu cha bằng tên gán cứng (F-005)'
m core/pipeline/why.go '		if nar.By != contracts.NarrowByScope {' '		if nar.By == "project" {' 'L5-13 lời giải thích chỉ nhận kiểu cha project (F-005)'
P=core/tenant/pack.go
m $P 'UnmatchedTemplate: inf.UnmatchedTemplate, NarrowParent: inf.NarrowParent}' 'UnmatchedTemplate: inf.UnmatchedTemplate}' 'L5-14 runtime bỏ narrow_parent của pack (F-005)'
m $P '		if p := inf.NarrowParent; (p.Type == "") != (p.Label == "") || p.Type == contracts.NarrowByScope {' '		if p := inf.NarrowParent; false {' 'L5-15 narrow_parent nửa vời vẫn nạp (F-005)'
export PKGS="./intelligence/intent/ ./core/tenant/"
m intelligence/intent/intent.go '		} else if !Installed(contracts.RouteKind(d.Route)) {' '		} else if false {' 'L5-16 ý định route chưa cài vẫn nạp (F-007)'
m $P '		} else if !intent.Installed(r) {' '		} else if false {' 'L5-17 unmatched_route chưa cài vẫn nạp (F-007)'
m intelligence/intent/intent.go 'contracts.RouteClarification, contracts.RouteHumanEscalation, contracts.RouteReject}' 'contracts.RouteClarification, contracts.RouteHumanEscalation, contracts.RouteReject, contracts.RouteWorkflow}' 'L5-18 coi WORKFLOW là đã cài (F-007)'
export PKGS="./core/tenant/..."
m core/tenant/meta/meta.go '	"go.yaml.in/yaml/v3"' '	_ "github.com/luannt2002/enterprise-ai-control-layer/data/normalization"
	"go.yaml.in/yaml/v3"' 'L5-19 phần meta của pack kéo động cơ QUERY (F-029)'
m $P '	if len(mf.Metrics) > 0 {' '	if false {' 'L5-20 pack có chỉ số không bị đòi mẫu câu số liệu (F-029)'
m $P '	if len(mf.Metrics) > 0 {' '	if true {' 'L5-21 pack chỉ-tài-liệu bị đòi mẫu câu số liệu (L6 báo)'
