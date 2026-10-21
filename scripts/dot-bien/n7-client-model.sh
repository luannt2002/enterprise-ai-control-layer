#!/usr/bin/env bash
# Danh mục đột biến "client model: không gửi sampling mặc định, thinking / effort từ cấu hình, thử lại có lùi" (ADR-0005 §1), đêm 20→21/10/2026.
# Mỗi dòng: m <tệp> <chuỗi gốc> <chuỗi đột biến> <nhãn>. Kỳ vọng: mọi dòng in "BỊ BẮT" (rc=0).
# Mã đổi sau ngày đó thì chuỗi gốc có thể không còn khớp (mutate.sh báo "xuất hiện 0 lần", rc=2):
# sửa chuỗi theo mã mới rồi chạy lại — đừng xoá dòng cho xanh.
cd "$(dirname "$0")/../.." || exit 1
export PKGS="./intelligence/models/"
F=intelligence/models/messages.go
m() { scripts/mutate.sh "$1" "$2" "$3" "$4"; echo "rc=$?"; }
m $F 'Model: m.Model, MaxTokens: m.MaxTokens, Temperature: m.Temperature, System: messagesSystem,' 'Model: m.Model, MaxTokens: m.MaxTokens, Temperature: new(float64), System: messagesSystem,' 'MS1 luôn gửi temperature 0'
m $F '		mreq.Thinking = &messagesThinking{Type: m.Thinking}' '		mreq.Thinking = nil' 'MS2 bỏ thinking đã cấu hình'
m $F '		mreq.OutputConfig = &messagesOutput{Effort: m.Effort}' '		mreq.OutputConfig = nil' 'MS3 bỏ effort đã cấu hình'
m $F '		if err == nil || !retry || attempt > m.MaxRetries {' '		if err == nil || retry || !retry || attempt > m.MaxRetries {' 'MS4 không thử lại'
m $F '		return raw, false, contracts.Errorf(contracts.FaultCode, fmt.Sprintf("model: yêu cầu bị từ chối %d", resp.StatusCode))' '		return raw, true, contracts.Errorf(contracts.FaultCode, fmt.Sprintf("model: yêu cầu bị từ chối %d", resp.StatusCode))' 'MS5 thử lại cả lỗi cấu hình 4xx'
m $F '		if c.Effort != "" && !slices.Contains(EffortLevels, c.Effort) {' '		if false {' 'MS6 không kiểm effort'
m $F '		if m.Trace != nil {' '		if m.Trace != nil && err == nil {' 'MS7 không ghi vết lần hỏng'
m $F '		Effort: os.Getenv("EACL_MODEL_EFFORT"), MaxRetries: 2,' '		Effort: os.Getenv("EACL_MODEL_EFFORT"), MaxRetries: 0,' 'MS8 mặc định không thử lại'
m $F '			m.Temperature = &t' '			_ = t' 'MS9 bỏ temperature đã cấu hình'
m $F '		if c.Thinking != "" && !slices.Contains(ThinkingTypes, c.Thinking) {' '		if false {' 'MS10 không kiểm thinking'
# soát C2, F-11: URL có thông tin xác thực bị từ chối và không in lại trong lỗi; localhost không phân biệt hoa thường
m intelligence/models/messages.go '	if err != nil || u.User != nil {' '	if err != nil {' 'K5 nhận URL có user:pass'
m intelligence/models/messages.go '	if strings.EqualFold(h, "localhost") {' '	if h == "localhost" {' 'K6 LOCALHOST bị từ chối oan'
m intelligence/models/messages.go '	return (&url.URL{Scheme: u.Scheme, Host: u.Host}).String()' '	return u.String()' 'K7 thông báo lỗi in lại mật khẩu / path / query (C3-08)'
