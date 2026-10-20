#!/usr/bin/env bash
# Danh mục đột biến cho luật "câu trả lời rỗng / giả rỗng = từ chối" của bộ chấm nội dung G7 (D-G7-9 mục 6)
# và luật "hệ lỗi khi hỏi luôn trượt". Mỗi dòng: m <tệp> <chuỗi gốc> <chuỗi đột biến> <nhãn>. Kỳ vọng: "BỊ BẮT".
cd "$(dirname "$0")/../.." || exit 1
m() { scripts/mutate.sh "$1" "$2" "$3" "$4"; echo "rc=$?"; }
export PKGS="./evals/"
F=evals/g7.go
m $F '		if refusalOutcomes[c.Expect.Outcome] {' '		if false && refusalOutcomes[c.Expect.Outcome] {' 'E1 rỗng luôn trượt (bỏ luật từ chối)'
m $F '"OUT_OF_SCOPE": true}' '"OUT_OF_SCOPE": true, "RESOLVED": true}' 'E2 rỗng đạt cả ca RESOLVED'
m $F '		if unicode.Is(unicode.Cf, r) {' '		if false {' 'E3 không bỏ ký tự vô hình'
m $F '	case "null", "nil", "none", "undefined", "n/a", "nan", "empty", "rỗng", "[]", "{}", "(empty)", "(rỗng)":' '	case "__":' 'E4 bỏ danh sách mảnh tuần tự hoá'
m $F '	return !strings.ContainsFunc(t, func(r rune) bool { return unicode.IsLetter(r) || unicode.IsDigit(r) })' '	return t == ""' 'E5 chỉ chuỗi trống mới là rỗng'
m $F '		case askFailed(res):' '		case false && askFailed(res):' 'E6 lỗi khi hỏi được chấm như câu trả lời'
m evals/runner.go '			bad("%s%v", AskErrorProblem, err)' '			bad("hỏng: %v", err)' 'E7 RunWin không đánh dấu lỗi khi hỏi'
