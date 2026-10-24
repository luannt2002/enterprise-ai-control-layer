#!/usr/bin/env bash
# Danh mục đột biến "W5 — G4 bắt buộc (SQL gọi gateway, RAG bỏ gateway) + ACL" đã chạy ngày 20/10/2026 (xem docs/ops/NHAT-KY-2026-10-20.md).
# Mỗi dòng: m <tệp> <chuỗi gốc> <chuỗi đột biến> <nhãn>. Kỳ vọng: mọi dòng in "BỊ BẮT" (rc=0).
# Mã đổi sau ngày đó thì chuỗi gốc có thể không còn khớp (mutate.sh báo "xuất hiện 0 lần", rc=2):
# sửa chuỗi theo mã mới rồi chạy lại — đừng xoá dòng cho xanh.
cd "$(dirname "$0")/../.." || exit 1
export PKGS="./data/... ./core/... ./intelligence/... ./tenants/... ./evals/ ./reliability/..."
m() { scripts/mutate.sh "$1" "$2" "$3" "$4"; echo "rc=$?"; }
m intelligence/rag/rag.go '		if vis[d] = ix.visible(d, pr); !vis[d] {' '		if vis[d] = true; !vis[d] {' 'AL bỏ ACL tài liệu'
m core/pipeline/ask.go '		if _, derived := a.RT.Metrics.Derived(metric); derived {' '		if plan.Route == contracts.RouteCalculation {' 'AZ HYBRID phép tính đi đường Query'
m core/pipeline/ask.go '	draft := contracts.Draft{Text: text, FromTemplate: true, Claims: claims}' '	if a.Deps.Model != nil {
		_, _ = a.Deps.Model.Generate(ctx, models.Request{Question: in.Raw})
	}
	draft := contracts.Draft{Text: text, FromTemplate: true, Claims: claims}' 'BA SQL gọi gateway (G4)'
m core/pipeline/ask.go '		return a.Deps.Model.Generate(ctx, models.Request{Question: in.Raw, Evidence: ev})' '		return contracts.Draft{Text: ev[0].Text, Claims: []contracts.Claim{{Text: ev[0].Text, EvidenceIDs: []string{ev[0].ID}}}}, nil' 'BB RAG bỏ gateway (G4)'
