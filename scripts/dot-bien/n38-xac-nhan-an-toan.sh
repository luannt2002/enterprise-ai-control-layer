#!/usr/bin/env bash
# Danh mục đột biến cho luật xác nhận AN TOÀN (bỏ luật A-02; từ chỉ kỳ xét trước từ nhãn; "tuần thứ 41"; số hiệu "phiếu 12-10"; trang trí
# chỉ bỏ qua khi số có dáng lượng hoặc đơn vị không mơ hồ) — sửa sau soát R (R-A01…R-A07, R-A14, R-A15, B-01…B-03).
# Mỗi dòng: m <tệp> <chuỗi gốc> <chuỗi đột biến> <nhãn>. Kỳ vọng: mọi dòng "BỊ BẮT".
cd "$(dirname "$0")/../.." || exit 1
m() { scripts/mutate.sh "$1" "$2" "$3" "$4"; echo "rc=$?"; }
export PKGS="./core/pipeline/ ./data/normalization/ ./reliability/validation/ ./tenants/construction/"
A=core/pipeline/ask.go
m $A '			if q, ok := normalization.PeriodQualifier(in.Raw, sp.Start, sp.End); ok && !(q.Kind' '			if q, ok := normalization.PeriodQualifier(in.Raw, sp.Start, sp.End); false && ok && !(q.Kind' 'AT1 bỏ mốc chưa đọc được'
m $A '			if normalization.LabelContext(in.Raw, sp.Start, sp.End) || dashTail' '			if false || dashTail' 'AT2 số hiệu sau từ nhãn thành con số xác nhận'
m $A ' || dashTail(in.Raw, labelEnd, sp.Start) {' ' {' 'AT3 phần "-10" của số hiệu thành con số thứ hai'
m $A '	for _, c := range primary {
		add(c)
	}' '	for _, c := range primary {
		if c.explicit {
			add(c)
		}
	}' 'AT4 luật A-02 trở lại (chỉ tính lượng viết rõ)'
m $A '	for _, c := range trailing {
		add(c)
	}' '	for _, c := range trailing {
		_ = c
	}' 'AT5 bỏ số của mệnh đề sau'
Q=data/normalization/qualifier.go
m $Q '		if wordIs(raw, ws, we, "thứ") || wordIs(raw, ws, we, "số") {' '		if false {' 'PQ1 "tuần thứ 41" không là mốc'
m $Q ' || LabelContext(raw, first.Start, first.End) || isDigitAt' ' || isDigitAt' 'DD1 "phiếu 12-10" đọc như ngày'
K=reliability/validation/knowledge.go
m $K '		if labelLike && ambiguousUnitWord(w) {' '		if false && labelLike && ambiguousUnitWord(w) {' 'KT1 "Bước 2, Khối Kế hoạch" bị chặn oan'
m $K '	labelLike := rest != gapOnly && !quantityShaped' '	labelLike := rest != gapOnly' 'KT2 số có dáng lượng cũng bị coi là nhãn'
m $K '	return strings.EqualFold(w, "khối") ||' '	return false ||' 'KT3 "khối" không còn là đơn vị mơ hồ'
