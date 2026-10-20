#!/usr/bin/env bash
# Danh mục đột biến cho C-05 (soát lượt 2): model trả văn xuôi (HTTP 200 hợp lệ, phần chữ không phải JSON) là đầu ra
# của model — bản nháp mang nguyên văn, KHÔNG claim — không phải lỗi nhà cung cấp (PROVIDER).
cd "$(dirname "$0")/../.." || exit 1
m() { scripts/mutate.sh "$1" "$2" "$3" "$4"; echo "rc=$?"; }
export PKGS="./intelligence/models/"
F=intelligence/models/messages.go
m $F '		return contracts.Draft{Text: strings.TrimSpace(text.String()), Model: m.Model, Usage: usage}, nil' '		return contracts.Draft{}, contracts.Label(contracts.ThirdParty(m.Provider, contracts.KindBadResponse), err)' 'VX1 văn xuôi thành lỗi nhà cung cấp'
m $F '		return contracts.Draft{Text: strings.TrimSpace(text.String()), Model: m.Model, Usage: usage}, nil' '		return contracts.Draft{Text: text.String(), Model: m.Model, Usage: usage, Claims: []contracts.Claim{{Text: text.String()}}}, nil' 'VX2 văn xuôi mang claim (EACL giao được câu không gắn bằng chứng)'
