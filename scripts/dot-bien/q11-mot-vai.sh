#!/usr/bin/env bash
# Danh mục đột biến "Q11 — quyền và phạm vi / ACL từ CÙNG một vai" (Q-P1-10 / Q11 = A, F-031; báo cáo audit vNext 24/10/2026 §16).
# Mỗi dòng: m <tệp> <chuỗi gốc> <chuỗi đột biến> <nhãn>. Kỳ vọng: mọi dòng in "BỊ BẮT" (rc=0). Đọc rc từng dòng (danh mục này không
# đếm kết cục). Không dòng nào cần PG.
# Mã đổi thì chuỗi gốc có thể không còn khớp (rc=2): sửa chuỗi theo mã mới rồi chạy lại — đừng xoá dòng cho xanh.
cd "$(dirname "$0")/../.." || exit 1
m() { scripts/mutate.sh "$1" "$2" "$3" "$4"; echo "rc=$?"; }
export PKGS="./tenants/construction/ ./execution/commands/ ./intelligence/rag/"
A=core/pipeline/ask.go
m $A '		def, _ := a.RT.Intents.Def(plan.Intent.Name)' '		def, _ := a.RT.Intents.Def("")' 'Q11-1 route RAG khớp ACL bằng mọi vai'
m $A 'a.RT.Policy.Holding(pr, kdef.Capability)' 'pr' 'Q11-2 phần quy trình của HYBRID khớp ACL bằng mọi vai'
m $A '			def, _ := a.RT.Intents.Def(f.Intent())' '			def, _ := a.RT.Intents.Def("")' 'Q11-3 dự phòng S4 dò chỉ mục bằng mọi vai'
P=core/policy/policy.go
m $P '		if slices.Contains(p.Roles[role].Capabilities, capability) {' '		if slices.Contains(p.Roles[role].Capabilities, capability) || true {' 'Q11-4 Holding giữ cả vai không có capability'
m $P '		if slices.Contains(p.Roles[role].Capabilities, capability) {' '		if slices.Contains(p.Roles[role].Capabilities, capability) && len(pr.Roles) == 1 {' 'Q11-5 Holding chặn oan người nhiều vai'
R=intelligence/rag/rag.go
m $R '		if ix.InScope != nil && ix.InScope(one, d.Scope) {' '		if ix.InScope == nil || ix.InScope(one, d.Scope) {' 'Q11-6 không luật phạm vi thì mở'
m $R '		if ix.InScope != nil && ix.InScope(one, d.Scope) {' '		if ix.InScope != nil && ix.InScope(one, d.Scope) || ix.InScope == nil && slices.Contains(d.Scope, strings.TrimSpace(pr.Attributes["entity_scope"])) {' 'Q11-7 chỉ mục tự đoán phạm vi theo thuộc tính'
C=execution/commands/pipeline.go
m $C '	if !d.Policy.InScope(holders, lineage) {' '	if !d.Policy.InScope(r.pr, lineage) {' 'Q11-8 C4 xét phạm vi trên mọi vai'
m $C 'holders := d.Policy.Holding(d.Policy.Holding(r.pr, d.Tool.Capability), CapMetricsRead)' 'holders := d.Policy.Holding(r.pr, CapMetricsRead)' 'Q11-9 C4 bỏ capability của lệnh khỏi vai xét phạm vi'
m $C 'holders := d.Policy.Holding(d.Policy.Holding(r.pr, d.Tool.Capability), CapMetricsRead)' 'holders := d.Policy.Holding(r.pr, d.Tool.Capability)' 'Q11-10 C4 bỏ metrics.read khỏi vai xét phạm vi'
