#!/usr/bin/env bash
# Danh mục đột biến "G6 — bản ghi quyết định + phát lại" đã chạy ngày 20/10/2026 (xem docs/ops/NHAT-KY-2026-10-20.md).
# Mỗi dòng: m <tệp> <chuỗi gốc> <chuỗi đột biến> <nhãn>. Kỳ vọng: mọi dòng in "BỊ BẮT" (rc=0).
# Mã đổi sau ngày đó thì chuỗi gốc có thể không còn khớp (mutate.sh báo "xuất hiện 0 lần", rc=2):
# sửa chuỗi theo mã mới rồi chạy lại — đừng xoá dòng cho xanh.
cd "$(dirname "$0")/../.." || exit 1
export PKGS="./data/... ./core/... ./observability/... ./tenants/..."
m() { scripts/mutate.sh "$1" "$2" "$3" "$4"; echo "rc=$?"; }
m observability/traces/decision.go '	add("route", a.Route == b.Route)' '	add("route", true)' 'GA so phát lại bỏ chặng route'
m core/pipeline/ask.go '	for _, c := range resp.Citations {
		d.Evidence = append(d.Evidence, c.EvidenceID)
	}' '' 'GB bản ghi không ghi id bằng chứng'
m core/pipeline/ask.go '	re.Deps.Now = func() time.Time { return d.Now }' '	_ = d.Now' 'GC phát lại dùng đồng hồ hiện tại'
m core/pipeline/ask.go '	if d.PackVersion != a.RT.PackVersion || d.DataVersion != a.Deps.DataVersion {' '	if false {' 'GE phát lại không kiểm phiên bản'
m observability/traces/decision.go '	add("evidence", slices.Equal(a.Evidence, b.Evidence))' '	add("evidence", true)' 'GF so phát lại bỏ chặng bằng chứng'
m data/sourceoftruth/query.go '		for _, src := range slices.Sorted(maps.Keys(bySource)) {
			fs := bySource[src]' '		for src, fs := range bySource {
			_, _ = slices.Sorted[string], maps.Keys[map[string][]Fact]' 'GD duyệt nguồn theo thứ tự map'
m observability/traces/decision.go '	add("intent", a.Intent == b.Intent && slices.Equal(a.Slots, b.Slots))' '	add("intent", a.Intent == b.Intent)' 'GG so phát lại bỏ khe'
