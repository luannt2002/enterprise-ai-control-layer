package construction_test

import (
	"slices"
	"strings"
	"testing"
)

// Vòng sửa S4 23/10 (ADR-0010, D-AI-02 = a): sửa bằng LUẬT theo bất biến, không thêm danh sách cụm đóng theo từng câu. Câu ở đây là câu viết
// mới theo cặp tối thiểu (đổi MỘT thứ, kết quả đổi đúng một nghĩa), KHÔNG chép từ bộ ca giữ lại.

// N1 (bất biến (c)): hoà ý định / thiếu thuật ngữ mà câu còn có bí danh mơ hồ thì câu hỏi lại PHẢI mang ứng viên của bí danh — người hỏi trả lời
// cả hai điều một lượt; bỏ rơi ứng viên là chỉ hỏi nửa câu.
func TestN1HoaYDinhKhongBoRoiUngVienThucThe(t *testing.T) {
	a := newAsk(t)
	director := user("director", "")
	for _, q := range []string{
		"Cảm biến báo máy 03 hôm qua đo 1.490 lít, đúng không?", // "cảm biến" (tiêu thụ) và "báo" (đội báo) hoà nhau
	} {
		r := ask(t, a, director, q).Response
		hoiLai(t, r, q)
		if got := candidateIDs(r); !slices.Equal(got, []string{"DZ-03", "EX-03"}) {
			t.Errorf("%q: ứng viên = %v, muốn [DZ-03 EX-03]", q, got)
		}
		if !strings.Contains(r.Answer, "EX-03") || !strings.Contains(r.Answer, "Bạn muốn hỏi") {
			t.Errorf("%q: câu hỏi lại phải hỏi cả nghĩa số liệu lẫn máy nào: %q", q, r.Answer)
		}
	}
	// đối chứng: máy rõ thì hoà ý định vẫn chỉ hỏi nghĩa số liệu, không có ứng viên
	q := "Cảm biến báo EX-03 hôm qua đo 1.490 lít, đúng không?"
	r := ask(t, a, director, q).Response
	hoiLai(t, r, q)
	if got := candidateIDs(r); len(got) != 0 || strings.Contains(r.Answer, "Có thể là") {
		t.Errorf("%q: máy đã rõ, không có ứng viên nào để kể; được %v %q", q, got, r.Answer)
	}
}
