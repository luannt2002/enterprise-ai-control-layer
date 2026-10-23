package construction_test

import (
	"slices"
	"strings"
	"testing"

	"github.com/luannt2002/enterprise-ai-control-layer/core/contracts"
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

// C5 (bất biến (b)): ý định lấy theo MỆNH ĐỀ HỎI. Mệnh đề đứng trước mệnh đề hỏi mà nêu một CON SỐ là ngữ cảnh — chữ chỉ nguồn của con số ấy
// ("phiếu kho ghi 3.850", "hoá đơn ghi 4.100 L", "đội báo 4k") không quyết định điều đang hỏi. Mệnh đề dẫn đề không nêu con số ("Về hoá đơn,
// …") vẫn nói chỉ số đang hỏi; mệnh đề đứng sau câu hỏi là phần bổ nghĩa của nó. EX-03 hôm nay: kho xuất 3.850 · đội báo 4.000 · tiêu thụ
// 3.782 · hoá đơn 4.100.
func TestC5MenhDeHoiQuyetDinhYDinh(t *testing.T) {
	a := newAsk(t)
	director := user("director", "")
	for _, c := range []struct {
		q, intent, value string
	}{
		// mệnh đề ngữ cảnh có con số: chữ chỉ nguồn của nó không rò sang ý định
		{"Đội báo 4k rồi, vậy hôm nay EX-03 tiêu thụ bao nhiêu dầu?", "fuel_consumed", "3.782"},
		{"Hoá đơn ghi 4.100 L, nên hôm nay EX-03 đã đốt bao nhiêu dầu?", "fuel_consumed", "3.782"},
		{"Phiếu kho ghi 3.850 rồi, vậy hôm nay EX-03 hết bao nhiêu dầu?", "fuel_issued", "3.850"},             // mệnh đề hỏi không có chỉ số riêng (chỉ "bao nhiêu"): dẫn đề còn tác dụng
		{"Hoá đơn ghi 4.100 L, vậy hôm nay EX-03 được xuất 4.100 L dầu phải không?", "fuel_generic", "3.850"}, // mệnh đề hỏi không có từ khoá riêng
		// đối chứng: không đổi
		{"Về hoá đơn dầu, EX-03 hôm nay bao nhiêu?", "fuel_billed", "4.100"},                                    // dẫn đề không có con số vẫn nói chỉ số đang hỏi
		{"Dầu đội báo, EX-03 hôm nay bao nhiêu?", "fuel_reported", "4.000"},                                     // như trên
		{"EX-03 hôm nay đội báo bao nhiêu dầu?", "fuel_reported", "4.000"},                                      // một mệnh đề
		{"EX-03 hôm nay bao nhiêu dầu, theo cảm biến?", "fuel_consumed", "3.782"},                               // mệnh đề đứng sau câu hỏi bổ nghĩa cho nó
		{"Đội báo 4k, báo cáo ghi 4 m3, vậy EX-03 hôm nay được cấp 4k dầu đúng không?", "fuel_issued", "3.850"}, // nhiều mệnh đề ngữ cảnh
	} {
		r := ask(t, a, director, c.q).Response
		if r.Intent != c.intent || !strings.Contains(r.Answer, c.value) {
			t.Errorf("%q: muốn ý định %s kèm %s, được [%s/%s] %q", c.q, c.intent, c.value, r.Route, r.Intent, r.Answer)
		}
		if c.intent == "fuel_generic" && (!strings.Contains(r.Answer, giaDinhChung) || verdictOf(r.Answer) != "") {
			t.Errorf("%q: trả theo ý định chung phải kèm câu giả định và không phán quyết: %q", c.q, r.Answer)
		}
	}
	// đối chứng: mệnh đề dẫn đề không có con số mà nêu chỉ số KHÁC chỉ số của mệnh đề hỏi: vẫn hỏi lại, không chọn thay người hỏi
	q := "Đội báo rồi, vậy hôm nay EX-03 tiêu thụ bao nhiêu dầu?"
	hoiLai(t, ask(t, a, director, q).Response, q)
}

// C4 (ADR-0010): câu nêu HAI chỉ số khác nhau kèm dấu hiệu so sánh ("chênh", "nhiều hơn") là phép so sánh ĐÚNG cặp người hỏi nêu, theo thứ tự
// nhắc (vế trước − vế sau); số cụm từ khoá khớp và độ ưu tiên của ý định "chênh lệch" không phải tín hiệu nghĩa. Ý định "chênh lệch" chỉ trả cặp
// mặc định của nó khi câu không nêu chỉ số nào ngoài cặp ấy. EX-03 hôm nay: kho xuất 3.850 · đội báo 4.000 · tiêu thụ 3.782 · hoá đơn 4.100.
func TestC4SoSanhDungCapNguoiHoiNeu(t *testing.T) {
	a := newAsk(t)
	director := user("director", "")
	for _, c := range []struct {
		q, intent, value string // value: số hiệu (vế trước − vế sau)
	}{
		// cặp người hỏi nêu KHÔNG phải cặp mặc định của "chênh lệch" (kho xuất − tiêu thụ): so đúng cặp đó, thứ tự nhắc
		{"EX-03 hôm nay đội báo và hoá đơn chênh nhau bao nhiêu lít?", "fuel_reported", "-100"},
		{"EX-03 hôm nay kho xuất và đội báo chênh nhau bao nhiêu lít?", "fuel_issued", "-150"},
		{"EX-03 hôm nay hoá đơn và đội báo chênh nhau bao nhiêu lít?", "fuel_billed", "100"},
		// số cụm từ khoá không quyết định: "cảm biến … đo được" (2 cụm) không thắng "cấp" (1 cụm) khi câu so sánh hai chỉ số
		{"EX-03 hôm nay được kho cấp nhiều hơn mức cảm biến đo được bao nhiêu lít?", "fuel_issued", "68"},
		{"EX-03 hôm nay được kho cấp nhiều hơn mức cảm biến đo bao nhiêu lít?", "fuel_issued", "68"}, // cặp tối thiểu: "đo được" → "đo"
	} {
		out := ask(t, a, director, c.q)
		r := out.Response
		if r.Route != contracts.RouteCalculation || r.Intent != c.intent || !strings.Contains(r.Answer, "chênh "+c.value+" L") {
			t.Errorf("%q: muốn phép so sánh %s chênh %s L, được [%s/%s] %q", c.q, c.intent, c.value, r.Route, r.Intent, r.Answer)
		}
	}
	// đối chứng: cặp mặc định của "chênh lệch" (kể cả thứ tự nhắc ngược) và câu chỉ nêu "chênh" / "hao hụt" vẫn trả cặp mặc định (kho xuất − tiêu thụ = 68)
	for _, q := range []string{
		"Chênh lệch dầu cấp và tiêu thụ của EX-03 hôm nay?",
		"Chênh lệch dầu tiêu thụ và cấp của EX-03 hôm nay?",
		"EX-03 hôm nay hao hụt bao nhiêu dầu?",
		"EX-03 hôm nay lệch bao nhiêu dầu so với cảm biến?",
	} {
		r := ask(t, a, director, q).Response
		if r.Route != contracts.RouteCalculation || r.Intent != "fuel_variance" || !strings.Contains(r.Answer, "68 L") {
			t.Errorf("%q: muốn cặp mặc định (fuel_variance, 68 L), được [%s/%s] %q", q, r.Route, r.Intent, r.Answer)
		}
	}
	// "chênh" mà câu nêu chỉ số NGOÀI cặp mặc định (một, hoặc ba) thì không có cặp nào để trả thay người hỏi: hỏi lại
	for _, q := range []string{
		"Chênh lệch dầu đội báo của EX-03 hôm nay?",
		"Chênh lệch dầu cấp, đội báo và hoá đơn của EX-03 hôm nay?",
	} {
		r := ask(t, a, director, q).Response
		hoiLai(t, r, q)
		if strings.Contains(r.Answer, "68") {
			t.Errorf("%q: không được trả cặp mặc định thay cặp người hỏi nêu: %q", q, r.Answer)
		}
	}
	// đối chứng: hai chỉ số không có dấu hiệu so sánh vẫn hoà → hỏi lại; câu chỉ một chỉ số vẫn trả chỉ số đó
	q := "EX-03 hôm nay đội báo kho xuất bao nhiêu dầu?"
	hoiLai(t, ask(t, a, director, q).Response, q)
	if r := ask(t, a, director, "EX-03 hôm nay đội báo bao nhiêu dầu?").Response; r.Intent != "fuel_reported" || !strings.Contains(r.Answer, "4.000") {
		t.Errorf("một chỉ số: muốn dầu đội báo 4.000, được [%s/%s] %q", r.Route, r.Intent, r.Answer)
	}
}

// C3 (bất biến (a)): KHÔNG KHỚP Ý ĐỊNH ≠ NGOÀI PHẠM VI. Tín hiệu thuộc phạm vi không cần chữ "dầu": một thực thể + một lượng cùng chiều đo với chỉ số
// (lít, khối) + chữ của từ khoá chỉ số ("xuất") là câu hỏi số liệu mà thuật ngữ vật tư được hiểu ngầm theo chính sách default_terms của tenant.
// Chữ của chính cụm xác nhận ("đúng" trong "xuất đúng 3850 L … đúng không") không mang nghĩa chỉ số; mã chứng từ ("PX-0912") là nhãn.
// EX-03 hôm nay kho xuất 3.850 L (ERP 3.900 L lệch); RL-02 kho xuất 1.200 L.
func TestC3PhamViKhongCanChuDau(t *testing.T) {
	a := newAsk(t)
	director := user("director", "")
	// trả lời theo ý định chung kèm câu giả định, không phán quyết
	for _, c := range []struct{ q, value string }{
		{"Hôm nay EX-03 xuất 3850 L đúng không?", "3.850"},
		{"EX-03 hôm nay xuất 4000 lít phải không?", "3.850"},
		{"Hôm nay RL-02 được xuất 1,2 khối đúng không?", "1.200"},
		{"Phiếu PX-0912 ghi EX-03 xuất 3.850 lít hôm nay, đúng không?", "3.850"}, // mã chứng từ là nhãn, dấu phẩy tách cụm hỏi
		{"EX-03 xuất đúng 4000 L hôm nay phải không?", "3.850"},                  // "đúng" thuộc cụm xác nhận, không phải chữ chỉ số
		{"Hôm nay EX-03 xuất đúng 3850 L dầu phải không?", "3.850"},              // cặp tối thiểu với câu có chữ "dầu"
	} {
		r := ask(t, a, director, c.q).Response
		if r.Route != contracts.RouteSQLQuery || r.Intent != "fuel_generic" || !strings.Contains(r.Answer, c.value) ||
			!strings.Contains(r.Answer, giaDinhChung) || verdictOf(r.Answer) != "" {
			t.Errorf("%q: muốn trả lời theo ý định chung kèm %s và câu giả định, không phán quyết — được [%s/%s/%s] %q", c.q, c.value, r.Route, r.Outcome, r.Intent, r.Answer)
		}
	}
	// chữ lạ ngoài từ khoá (động từ của chỉ số khác, điều kiện (ii) của D-OP-15) và câu hỏi không có cụm xác nhận: hỏi lại, không trả số kho xuất
	for _, q := range []string{
		"Hôm nay EX-03 xuất 3850 L?",                // không có cụm xác nhận
		"Hôm nay EX-03 xuất đo 3782 L phải không?",  // "đo" là mảnh từ khoá của chỉ số khác ("đo được"), không trung tính
		"Hôm nay EX-03 xuất xài 3782 L đúng không?", // động từ của chỉ số khác, không nằm trong từ khoá
	} {
		r := ask(t, a, director, q).Response
		hoiLai(t, r, q)
		if strings.Contains(r.Answer, "3.850") || strings.Contains(r.Answer, giaDinhChung) {
			t.Errorf("%q: hỏi lại mà vẫn trả số kho xuất: %q", q, r.Answer)
		}
	}
	// mệnh đề ngữ cảnh nêu con số (hoá đơn) không quyết định ý định khi câu hỏi không có từ khoá riêng: không được trả theo hoá đơn
	q := "Hoá đơn ghi 4.100 L cho EX-03, vậy hôm nay kho cũng xuất đúng 4.100 L phải không?"
	if r := ask(t, a, director, q).Response; r.Intent == "fuel_billed" || strings.Contains(r.Answer, "4.100 L.") {
		t.Errorf("%q: trả theo hoá đơn — sai nghĩa không nhãn: [%s/%s] %q", q, r.Route, r.Intent, r.Answer)
	}
	// ngoài phạm vi thật vẫn từ chối (đối chứng âm): thực thể + lượng nhưng không phải chiều đo của chỉ số, hay không có chữ của từ khoá chỉ số, hay
	// vật tư khác
	for _, q := range []string{
		"EX-03 chứa được 300 lít nước đúng không?",
		"Hôm nay EX-03 xài 3782 L đúng không?", // giới hạn đã biết: không chữ "dầu", không chữ nào của từ khoá chỉ số ⇒ chưa đủ tín hiệu (ghi nợ)
		"EX-03 hôm nay chạy 8 tiếng đúng không?",
		"Máy xúc EX-03 nặng 20 tấn đúng không?",
		"Cầu A dài 300 mét đúng không?",
		"Hôm nay EX-03 xuất 20 tấn đúng không?", // có chữ của từ khoá chỉ số ("xuất") nhưng lượng không cùng chiều đo
		"Hôm nay EX-03 nhận 20 L dầu nhớt đúng không?",
		"Hôm nay EX-03 xuất 20 L dầu ăn đúng không?",
		"Hôm qua anh Tám lái lu rung 02 đi ăn cưới đúng không?",
		"VN-Index hôm nay giảm 30 điểm đúng không?",
	} {
		if r := ask(t, a, director, q).Response; r.Route != contracts.RouteReject || r.Outcome != contracts.OutcomeOutOfScope {
			t.Errorf("%q: ngoài phạm vi, muốn từ chối, được [%s/%s] %q", q, r.Route, r.Outcome, r.Answer)
		}
	}
}

// Lời chào là chữ "chào" (lời chào theo giờ, theo người: "chào buổi sáng", "chào các bạn"), không phải một danh sách cụm đóng; ý định chào có độ ưu tiên
// thấp nhất nên câu có chào mà còn hỏi số liệu vẫn được trả lời số liệu.
func TestC3LoiChaoLaChuChao(t *testing.T) {
	a := newAsk(t)
	director := user("director", "")
	for _, q := range []string{"Chào buổi sáng nhé!", "Chào các bạn", "Chào buổi chiều", "Xin chào", "hello"} {
		r := ask(t, a, director, q).Response
		if r.Route != contracts.RouteClarification || r.Intent != "greeting" || !strings.Contains(r.Answer, "Chào bạn") {
			t.Errorf("%q: muốn lời chào, được [%s/%s] %q", q, r.Route, r.Intent, r.Answer)
		}
	}
	if r := ask(t, a, director, "Chào buổi sáng, EX-03 hôm nay được cấp bao nhiêu dầu?").Response; r.Intent != "fuel_issued" || !strings.Contains(r.Answer, "3.850") {
		t.Errorf("chào kèm câu hỏi số liệu: muốn trả số liệu, được [%s/%s] %q", r.Route, r.Intent, r.Answer)
	}
	for _, q := range []string{"Cảm ơn bạn nhé", "Tạm biệt nhé"} {
		if r := ask(t, a, director, q).Response; r.Route != contracts.RouteReject {
			t.Errorf("%q: không phải lời chào, muốn từ chối, được [%s/%s] %q", q, r.Route, r.Intent, r.Answer)
		}
	}
}
