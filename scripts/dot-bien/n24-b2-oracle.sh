#!/usr/bin/env bash
# Danh mục đột biến cho B2-oracle (g7run oracleKeys / oracleRAG / build; D-G7-9 mục 5, chỉ báo cáo).
# Mỗi dòng: m <tệp> <chuỗi gốc> <chuỗi đột biến> <nhãn>. Kỳ vọng: mọi dòng "BỊ BẮT".
cd "$(dirname "$0")/../.." || exit 1
m() { scripts/mutate.sh "$1" "$2" "$3" "$4"; echo "rc=$?"; }
export PKGS="./tenants/construction/win/g7run/"
F=tenants/construction/win/g7run/systems.go
m $F 'r.Tenant == tenantID && (ids[r.Key] || (r.WorldID != "" && ids[r.WorldID]))' 'r.Tenant == tenantID && (ids[r.Key] || (r.WorldID != "" && false))' 'OR1 bỏ khớp world_id'
m $F 'r.Tenant == tenantID && (ids[r.Key] || (r.WorldID != "" && ids[r.WorldID]))' 'true && (ids[r.Key] || (r.WorldID != "" && ids[r.WorldID]))' 'OR2 lấy cả bản ghi tenant khác'
m $F '	if !ok {
		return o.inner.Ask(ctx, pr, question)' '	if ok || !ok {
		return o.inner.Ask(ctx, pr, question)' 'OR3 luôn hỏi B2+ (bỏ bằng chứng vàng)'
m $F '		q = o.preamble + "\n" + question' '		q = question' 'OR4 bỏ dòng ngày hiện tại'
m $F '			ids[i] = fmt.Sprintf("P-%06d#0", n)' '			ids[i] = fmt.Sprintf("P-%06d#0", i)' 'OR5 id đoạn lệch chỉ mục B2+'
m $F '			ids[i] = fmt.Sprintf("P-%06d#0", n)' '			ids[i] = fmt.Sprintf("P-%06d", n)' 'OR12 id thiếu số dòng #0 của chỉ mục B2+ (C-10)'
m $F 'if old, dup := byQ[k]; dup && (' 'if old, dup := byQ[k]; false && dup && (' 'OR6 bỏ kiểm hai ca cùng câu hỏi'
m $F 'return int(o.calls.Load()) + o.inner.ModelCalls()' 'return int(o.calls.Load())' 'OR7 không đếm lời gọi của B2+ bên trong'
m $F '	o.inner.ResetModelCalls()' '	_ = o.inner' 'OR8 không đặt lại bộ đếm B2+ bên trong'
G=tenants/construction/win/g7run/main.go
m $G '				return nil, fmt.Errorf("B2-oracle cần -manifest có truth.record_ids của từng ca")' '				continue' 'OR9 thiếu hồ sơ ca vẫn chạy'
m $G '			inner.TopK, inner.Preamble = cfg.topK, todayLine(ref)' '			inner.TopK = cfg.topK' 'OR10 B2+ dự phòng thiếu dòng ngày'
m $G '			inner.TopK, inner.Preamble = cfg.topK, todayLine(ref)' '			inner.Preamble = todayLine(ref)' 'OR11 B2+ dự phòng sai top-k'
