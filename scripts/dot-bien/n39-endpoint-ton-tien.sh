#!/usr/bin/env bash
# Danh mục đột biến cho sửa soát C3 (C3-04, C3-06, C3-07, C3-08): endpoint so theo nghĩa; mọi endpoint không phải loopback
# (cả proxy) qua kiểm trước lời gọi model; thử khói tốn tiền không vượt cỡ hợp đồng và phải ghi được -out; lượt chính đủ hệ
# cổng; log và bằng chứng chỉ ghi scheme://host. Mỗi dòng: m <tệp> <chuỗi gốc> <chuỗi đột biến> <nhãn>. Kỳ vọng: mọi dòng
# "BỊ BẮT". Không đưa vào (tương đương, đã xét): bỏ "u.Scheme != o.Scheme" (http tới máy không phải loopback đã bị
# secureEndpoint từ chối); đổi "!priced && costly" về "!priced && !dry" (model không có giá thì contractDeviations của
# preflightReal đã từ chối mọi lượt tốn tiền, vì model hợp đồng có trong bảng giá).
cd "$(dirname "$0")/../.." || exit 1
m() { scripts/mutate.sh "$1" "$2" "$3" "$4"; echo "rc=$?"; }
export PKGS="./tenants/construction/win/g7run/ ./intelligence/models/"
M=tenants/construction/win/g7run/main.go E=tenants/construction/win/g7run/evidence.go C=intelligence/models/messages.go
m $M '	return strings.EqualFold(strings.TrimSuffix(u.Hostname(), "."), o.Hostname())' '	return raw == officialEndpoint' 'EO1 so endpoint theo chuỗi'
m $M 'strings.TrimSuffix(u.Hostname(), ".")' 'u.Hostname()' 'EO2 dấu chấm cuối tên máy thành chạy khô'
m $M ' || (u.Port() != "" && u.Port() != "443") {' ' {' 'EO3 cổng khác 443 thành endpoint chính thức'
m $M '		dry, costly = !isOfficialEndpoint(m.BaseURL), !models.LoopbackEndpoint(m.BaseURL)' '		dry, costly = !isOfficialEndpoint(m.BaseURL), false' 'CO1 lượt tốn tiền không qua kiểm trước'
m $M '	if !fake && costly {' '	if !fake && !dry {' 'CO2 proxy không qua kiểm trước'
m $M '	if stopBeforeModel(s.blocked, fake, !costly) {' '	if stopBeforeModel(s.blocked, fake, dry) {' 'CO3 proxy có hệ cổng bị chặn vẫn gọi model'
m $M '		if cfg.smoke > contractSmoke {' '		if false {' 'SM1 thử khói vượt cỡ hợp đồng'
m $M '		if cfg.out != "" {
			if err := probeWritable(cfg.out); err != nil {' '		if false {
			if err := probeWritable(cfg.out); err != nil {' 'SM2 thử khói không kiểm -out ghi được'
m $M '				if !slices.Contains(systems, g) {' '				if !slices.Contains(systems, g) && false {' 'GS1 lượt chính thiếu hệ cổng vẫn chạy'
m $E '		m.Endpoint = models.EndpointForLog(g.BaseURL)' '		m.Endpoint = g.BaseURL' 'EL2 bằng chứng ghi nguyên URL'
# F6 (soát fd4d19c + 246eabb): lỗi truyền tải của net/http mang URL đầy đủ. Không đưa vào: redactURL sau NewRequestWithContext (URL
# không parse được đã bị secureEndpoint từ chối lúc nạp cấu hình nên test không tới được nhánh đó).
m $C '		redactURL(err)
		kind := contracts.KindUnreachable' '		kind := contracts.KindUnreachable' 'RD1 lỗi kết nối in nguyên URL có khoá'
m $C '		ue.URL = EndpointForLog(ue.URL)' '		_ = ue' 'RD2 redactURL không bóc URL'
