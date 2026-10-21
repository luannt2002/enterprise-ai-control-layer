#!/usr/bin/env bash
# Danh mục đột biến cho C-05 (soát lượt 2): model trả văn xuôi (HTTP 200 hợp lệ, phần chữ không phải JSON) là đầu ra
# của model — bản nháp mang nguyên văn, KHÔNG claim — không phải lỗi nhà cung cấp (PROVIDER).
cd "$(dirname "$0")/../.." || exit 1
m() { scripts/mutate.sh "$1" "$2" "$3" "$4"; echo "rc=$?"; }
export PKGS="./intelligence/models/"
F=intelligence/models/messages.go
m $F '		return contracts.Draft{Text: strings.TrimSpace(text.String()), Model: m.Model, Usage: usage}, nil' '		return contracts.Draft{}, contracts.Label(contracts.ThirdParty(m.Provider, contracts.KindBadResponse), err)' 'VX1 văn xuôi thành lỗi nhà cung cấp'
m $F '		return contracts.Draft{Text: strings.TrimSpace(text.String()), Model: m.Model, Usage: usage}, nil' '		return contracts.Draft{Text: text.String(), Model: m.Model, Usage: usage, Claims: []contracts.Claim{{Text: text.String()}}}, nil' 'VX2 văn xuôi mang claim (EAIS giao được câu không gắn bằng chứng)'
# soát C2, F-04: phản hồi 200 không có chữ / bị cắt / bị từ chối là lỗi, không thành bản nháp rỗng
m intelligence/models/messages.go '	if !hasText || mr.StopReason == "max_tokens" ||' '	if false && !hasText || mr.StopReason == "max_tokens" ||' 'VX3 phản hồi không có khối chữ thành bản nháp rỗng'
m intelligence/models/messages.go 'mr.StopReason == "max_tokens" || mr.StopReason == "refusal" {' 'false || mr.StopReason == "refusal" {' 'VX4 câu bị cắt vì max_tokens được giao'
m intelligence/models/messages.go '|| mr.StopReason == "refusal" {' '|| false {' 'VX5 nhà cung cấp từ chối thành câu trả lời'
