package contracts

import (
	"fmt"
	"strings"
)

// Chữ của người hỏi đi vào lời giải thích của vết (R2-03: Decision.Why, Step.Reason / Step.Rejected) chỉ qua Quote. Một hàm dùng chung cho
// mọi dòng giải thích có chữ người dùng — ở intelligence/intent lẫn core/pipeline — để không dòng nào tự nhúng chữ theo cách riêng.

const (
	// QuoteOpen, QuoteClose bao quanh chữ của người hỏi. Quote không để hai ký tự này lọt vào bên trong đoạn được bao, nên mọi đoạn như thế
	// trong một dòng giải thích đều tìm lại được chính xác (xem traces.Why.WithholdQuoted).
	QuoteOpen  = "“"
	QuoteClose = "”"

	// maxQuoteRunes là trần độ dài chữ trích từ câu hỏi vào MỘT dòng giải thích (câu hỏi dài tới 2000 ký tự, mệnh đề hỏi có thể gần bằng cả câu).
	maxQuoteRunes = 120
)

// Quote đặt chữ của người hỏi vào cặp “…” để nhúng vào một dòng giải thích. Cố ý KHÔNG thoát nháy như %q: thoát \" và \\ làm hỏng các mẫu che
// bí mật của traces.Sanitize ({"api_key": "…"}, password="…") nên bí mật lọt vào vết bền (R2-03 vòng 1, F1). Quote gộp khoảng trắng (xuống dòng
// không phá bảng chữ), đổi “ ” có sẵn trong chữ thành nháy thường (để cặp bao luôn tìm lại được) và cắt ở maxQuoteRunes.
func Quote(s string) string {
	s = strings.NewReplacer(QuoteOpen, `"`, QuoteClose, `"`).Replace(s)
	s = strings.Join(strings.Fields(s), " ")
	if r := []rune(s); len(r) > maxQuoteRunes {
		s = string(r[:maxQuoteRunes]) + "…"
	}
	return QuoteOpen + s + QuoteClose
}

// Brief nối các phần tử bằng ", "; nhiều hơn max thì chỉ in max phần tử đầu rồi "và N <noun> khác" (câu nhắc bí danh có hàng chục ứng viên
// không được làm dòng giải thích dài theo).
func Brief(items []string, max int, noun string) string {
	if len(items) <= max {
		return strings.Join(items, ", ")
	}
	return fmt.Sprintf("%s và %d %s khác", strings.Join(items[:max], ", "), len(items)-max, noun)
}
