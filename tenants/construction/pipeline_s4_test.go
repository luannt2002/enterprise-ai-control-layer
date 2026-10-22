package construction_test

import (
	"slices"
	"strings"
	"testing"

	"github.com/luannt2002/enterprise-ai-control-layer/core/contracts"
	"github.com/luannt2002/enterprise-ai-control-layer/data/normalization"
)

// giaDinhChung là câu giả định chủ dự án đòi kèm mọi câu trả lời theo ý định chung (D-OP-15, điều kiện (i)).
const giaDinhChung = "Hiểu là dầu kho xuất; nếu bạn hỏi tiêu thụ / đội báo / hoá đơn thì nói rõ."

// Vòng sửa S4 (ADR-0010 bất biến (a), D-OP-15 = a2 kèm 2 điều kiện): câu có thực thể + thuật ngữ vật tư mà không từ khoá ý định
// nào khớp thì KHÔNG từ chối. Đủ thực thể, thuật ngữ, cụm xác nhận và mọi chữ còn lại đều thuộc nghĩa của ý định chung (mảnh từ khoá
// của ý định mà ý định chung quy về, tên nguồn của chỉ số đó, từ trung tính) ⇒ trả lời theo ý định chung, kèm câu giả định, không
// phán quyết (a2). Còn lại ⇒ hỏi lại nghĩa số liệu (a1), chỉ liệt kê ý định người hỏi có quyền.
func TestS4KhongKhopYDinhKhongPhaiNgoaiPhamVi(t *testing.T) {
	a := newAsk(t)
	director := user("director", "")

	// a1: hỏi lại nghĩa số liệu (câu thiết kế của bước điều tra 22/10, không chép từ bộ ca giữ lại)
	for _, q := range []string{
		"Hôm qua máy xúc EX-03 đổ 3.850 lít dầu đúng không?",
		"Máy xúc 07 hôm nay nạp 120 lít dầu phải không?",
		"Cầu A tuần này bơm dầu thế nào rồi?",
		"Cho mình xem tình hình dầu của lu 02 hôm qua",
		"Lu rung 02 ngày 15/10 chạy hết 200 lít dầu có đúng không?",
		"Nhiên liệu của dự án B tháng này ra sao?",
		"Tháng 9 cầu A hết 5.400 lít dầu đúng không?",
		"EX-07 ngày 19/10 đổ 1.200 L dầu phải ko?",
		"Số lít dầu của máy 07 hôm qua?",
		"Cho tôi biết lượng dầu của EX-03 hôm qua",
		"Hạ tầng C tuần trước dầu về nhiều không?",
		"Hôm nay EX-03 được xuất dầu?", // a2 đòi cụm xác nhận: không có thì hỏi lại
		// câu xác nhận không nêu chỉ số (BR-FUEL-02: chưa rõ chỉ số mà nguồn đang lệch thì hỏi lại, không xác nhận)
		"Hôm qua EX-03 4000 L dầu đúng không?",
		"Hôm nay EX-03 là 4000 L dầu, đúng không?",
		"Hôm nay EX-03 được 3850 L dầu đúng không?",
		// điều kiện (ii): động từ của chỉ số khác, không nằm trong từ khoá — không được trả số kho xuất
		"Hôm nay EX-03 xài 3782 L dầu đúng không?",
		"EX-03 hôm nay hao 3782 lít dầu phải không?",
		"EX-03 hôm nay tiêu 3782 L dầu đúng không?",
		"Hôm nay EX-03 tính tiền 4100 L dầu đúng không?",
		"EX-03 hôm nay đo 3782 L dầu đúng không?",
		"Hôm nay EX-03 được xuất, đo 3782 L dầu đúng không?", // "đo" trung tính với cổng phán quyết, nhưng là mảnh của "đo được"
		"EX-03 hôm nay định mức 5000 L dầu đúng không?",
		// tên nguồn chưa thuộc tập chữ giải thích được của a2 (ý định không biết nguồn của chỉ số) ⇒ hỏi lại, phía an toàn
		"EX-03 hôm nay ERP ghi 3900 L dầu, vậy đúng không?",
		// thuật ngữ / lượng của đại lượng KHÁC không phải chữ trung tính của a2 (soát S4 vòng 1, S4R-01): hỏi giá, giờ máy, khối lượng
		"Đơn giá dầu xuất hôm nay của EX-03 là 4100 đúng không?",
		"Giờ máy xuất dầu hôm nay của EX-03 là 8h đúng không?",
		"Hôm nay EX-03 xuất 7 tấn dầu đúng không?",
		"Hôm nay nhân công EX-03 xuất 3850 L dầu đúng không?", // thuật ngữ của đại lượng khác là chữ lạ duy nhất
		// chữ không dấu trong câu có dấu xét theo cùng luật dấu với cả câu (S4R-06): "duoc" không phải "được"
		"Hôm qua EX-03 duoc 4k dầu đúng không?",
		// chữ chỉ loại chỉ thuộc tên khi đúng loại của thực thể trong câu (soát S4 vòng 2, F1): EX-03 là thiết bị, không phải dự án
		"Hôm nay dự án EX-03 xuất 3850 L dầu đúng không?",
	} {
		r := ask(t, a, director, q).Response
		if r.Route != contracts.RouteClarification || r.Outcome != contracts.OutcomeClarificationRequired {
			t.Errorf("%q: muốn hỏi lại nghĩa số liệu, được [%s/%s] %q", q, r.Route, r.Outcome, r.Answer)
			continue
		}
		if strings.Contains(r.Answer, "3.850") || strings.Contains(r.Answer, giaDinhChung) {
			t.Errorf("%q: hỏi lại mà vẫn trả số kho xuất: %q", q, r.Answer)
		}
	}

	// a2: trả lời theo ý định chung, kèm câu giả định, không "Đúng." / "Không." (EX-03 hôm nay kho xuất 3.850 L)
	for _, q := range []string{
		"Hôm nay EX-03 được xuất 3850 L dầu đúng không?",
		"EX-03 hôm nay xuất 4000 lít dầu phải không?",
		"Cho hỏi hôm nay EX-03 đã được xuất 3850 L dầu đúng không ạ?",
		"EX-03 ngày 19/10 được xuất 3850 L dầu đúng không?", // "ngày" đứng trước mốc số thuộc mốc
		"Hôm nay EX-03 xuất 4 khối dầu đúng không?",         // lượng cùng chiều đo (thể tích) với chỉ số chung
		"Hôm nay EX-03 xuất dầu 3850 đúng không?",           // số trần
		// số trần không đứng cạnh "dầu" (không được gắn đơn vị) là con số của câu xác nhận (soát S4 vòng 2, F2)
		"Dầu xuất hôm nay của EX-03 là 3850 đúng không?",
		"Dầu được xuất cho EX-03 hôm nay là 3850 đúng không?",
	} {
		r := ask(t, a, director, q).Response
		if r.Route != contracts.RouteSQLQuery || r.Outcome != contracts.OutcomeResolved || r.Intent != "fuel_generic" {
			t.Errorf("%q: muốn trả lời theo ý định chung, được [%s/%s/%s] %q", q, r.Route, r.Outcome, r.Intent, r.Answer)
			continue
		}
		if !strings.Contains(r.Answer, "3.850") || !strings.Contains(r.Answer, giaDinhChung) || verdictOf(r.Answer) != "" {
			t.Errorf("%q: muốn số kho xuất 3.850 kèm câu giả định, không phán quyết — được %q", q, r.Answer)
		}
	}

	// chữ chỉ loại của thực thể có trong câu ("dự án" của "dự án Cầu A") thuộc tên, không phải chữ lạ (soát S4 vòng 2, F1); Cầu A hôm
	// nay kho xuất 5.050 L
	for _, q := range []string{"Hôm nay dự án Cầu A xuất 5050 L dầu đúng không?", "Hôm nay công trình Cầu A xuất 5050 L dầu đúng không?"} {
		r := ask(t, a, director, q).Response
		if r.Intent != "fuel_generic" || r.Outcome != contracts.OutcomeResolved || !strings.Contains(r.Answer, "5.050") ||
			!strings.Contains(r.Answer, giaDinhChung) || verdictOf(r.Answer) != "" {
			t.Errorf("%q: muốn số kho xuất 5.050 theo ý định chung, kèm câu giả định, không phán quyết — được [%s/%s/%s] %q", q, r.Route,
				r.Outcome, r.Intent, r.Answer)
		}
	}

	// a2 với bí danh mơ hồ: hỏi lại máy nào (đủ ứng viên), giữ ý định chung
	if r := ask(t, a, director, "Máy 03 hôm nay được xuất 3850 L dầu đúng không?").Response; r.Route != contracts.RouteClarification ||
		r.Intent != "fuel_generic" || r.Clarification == nil || len(r.Clarification.Candidates) != 2 {
		t.Errorf("bí danh mơ hồ: muốn hỏi lại máy nào (2 ứng viên) theo ý định chung, được [%s/%s] %q", r.Route, r.Intent, r.Answer)
	}

	// ngoài phạm vi thật vẫn từ chối (đối chứng âm của (a))
	for _, q := range []string{
		"VN-Index hôm nay bao nhiêu điểm?",
		"Mua dầu ăn ở đâu rẻ nhất?",
		"Dầu gội nào trị gàu tốt nhất?",
		"Máy xúc EX-03 trông ngầu quá, mình thích nhất con này",
		"Hôm qua anh Tám lái lu rung 02 đi ăn cưới ngoài thị trấn",
		"Hôm qua máy bay đổ 3.850 lít dầu đúng không?",
		"Giờ máy của EX-03 hôm qua là bao nhiêu?",
		"Giá nhân công dự án B tháng này bao nhiêu?",
		"Dầu diesel thế giới tuần này có tăng không?",
		"Nghe nói EX-03 sắp được đổi tên thành Ông Voi, đúng không?",
		"Cảm ơn bạn nhé",
		// "dầu" nghĩa khác (dầu nhớt, dầu thuỷ lực…) là vật tư khác, không phải nhiên liệu (soát S4 vòng 1, S4R-04)
		"EX-03 thay dầu nhớt lúc nào?",
		"Dầu thủy lực của máy xúc 07 loại nào?",
		"EX-03 hôm nay nhận bao nhiêu dầu nhớt?",
	} {
		if r := ask(t, a, director, q).Response; r.Route != contracts.RouteReject || r.Outcome != contracts.OutcomeOutOfScope {
			t.Errorf("%q: ngoài phạm vi, muốn từ chối, được [%s/%s] %q", q, r.Route, r.Outcome, r.Answer)
		}
	}
}

// Hỏi lại nghĩa số liệu chỉ cho thực thể trong phạm vi dữ liệu của người hỏi (ngoài phạm vi thì từ chối ngay, như mọi câu số liệu), và
// kể ứng viên khi bí danh mơ hồ.
func TestS4HoiLaiPhamViVaUngVien(t *testing.T) {
	a := newAsk(t)
	engB := user("site_engineer", "PRJ-B")
	if r := ask(t, a, engB, "Cho mình xem tình hình dầu của lu 02 hôm qua").Response; r.Route != contracts.RouteReject ||
		r.Outcome != contracts.OutcomePermissionDenied {
		t.Errorf("kỹ sư dự án B hỏi lu 02 (dự án A): muốn từ chối vì phạm vi, được [%s/%s] %q", r.Route, r.Outcome, r.Answer)
	}
	if r := ask(t, a, engB, "Nhiên liệu của dự án B tháng này ra sao?").Response; r.Route != contracts.RouteClarification {
		t.Errorf("kỹ sư dự án B hỏi dự án B: muốn hỏi lại nghĩa số liệu, được [%s/%s] %q", r.Route, r.Outcome, r.Answer)
	}
	r := ask(t, a, user("director", ""), "Cho mình xem tình hình dầu của máy 03 hôm nay").Response
	var got []string
	if r.Clarification != nil {
		for _, c := range r.Clarification.Candidates {
			got = append(got, c.ID)
		}
	}
	slices.Sort(got)
	if r.Route != contracts.RouteClarification || strings.Join(got, ",") != "DZ-03,EX-03" {
		t.Errorf("bí danh mơ hồ máy 03: muốn hỏi lại kèm ứng viên DZ-03, EX-03, được [%s] %v %q", r.Route, got, r.Answer)
	}
}

// Hỏi lại nghĩa số liệu chỉ kể ý định người hỏi có quyền; không có ý định nào thì giữ cách cũ (từ chối ngoài phạm vi), không lộ nhãn
// ý định cho người không có capability (CLAUDE.md §21: cả ca cho phép lẫn ca từ chối).
func TestS4HoiLaiChiKeYDinhCoQuyen(t *testing.T) {
	a := newAsk(t)
	q := "Cầu A tuần này bơm dầu thế nào rồi?"
	r := ask(t, a, user("director", ""), q).Response
	if r.Route != contracts.RouteClarification || !strings.Contains(r.Answer, "dầu kho xuất") || !strings.Contains(r.Answer, " hay ") {
		t.Errorf("giám đốc: muốn hỏi lại kể các nghĩa số liệu, được [%s] %q", r.Route, r.Answer)
	}
	// chỉ kể nghĩa SỐ LIỆU riêng: không kể ý định chung ("dầu (kho xuất)"), quy trình hay lời chào
	for _, bad := range []string{"(kho xuất)", "quy trình", "chào"} {
		if strings.Contains(r.Answer, bad) {
			t.Errorf("giám đốc: hỏi lại không được kể %q, được %q", bad, r.Answer)
		}
	}
	r = ask(t, a, user("driver", "EX-03"), q).Response
	if r.Route != contracts.RouteReject || r.Outcome != contracts.OutcomeOutOfScope || strings.Contains(r.Answer, "dầu kho xuất") {
		t.Errorf("tài xế không có capability: muốn từ chối như cũ, không lộ nhãn ý định, được [%s/%s] %q", r.Route, r.Outcome, r.Answer)
	}
}

// Điều kiện (i) của D-OP-15: MỌI câu trả lời theo ý định chung (kể cả câu khớp từ khoá yếu "bao nhiêu", "hết bao nhiêu") nói rõ giả
// định; câu khớp ý định riêng thì không.
func TestS4CauTraLoiYDinhChungNoiRoGiaDinh(t *testing.T) {
	a := newAsk(t)
	director := user("director", "")
	// có số, chưa có dữ liệu (EX-07 hôm nay), nguồn ngang thẩm quyền mâu thuẫn (dự án B hôm nay): đều nói rõ giả định
	for _, q := range []string{"EX-03 hôm nay bao nhiêu dầu?", "Máy xúc EX-03 hôm nay hết bao nhiêu dầu?", "EX-07 hôm nay bao nhiêu dầu?",
		"Dự án B hôm nay bao nhiêu dầu?"} {
		r := ask(t, a, director, q).Response
		if r.Intent != "fuel_generic" || !strings.Contains(r.Answer, giaDinhChung) {
			t.Errorf("%q: muốn ý định chung kèm câu giả định, được [%s/%s] %q", q, r.Intent, r.Outcome, r.Answer)
		}
	}
	for _, q := range []string{"EX-03 hôm nay kho xuất bao nhiêu dầu?", "EX-03 hôm nay được cấp 3850 L đúng không?"} {
		if r := ask(t, a, director, q).Response; strings.Contains(r.Answer, giaDinhChung) {
			t.Errorf("%q: ý định riêng không kèm câu giả định, được %q", q, r.Answer)
		}
	}
}

// Mọi bí danh của vật tư "dầu không phải nhiên liệu" (other_oil) được đọc là vật tư đó, không phải dầu diesel; câu hỏi số lượng về một
// bí danh có chữ "dầu" vẫn ngoài phạm vi, không bị trả số dầu diesel (soát S4 vòng 2, F4: trước đó chỉ hai bí danh có test).
func TestS4DauKhongPhaiNhienLieuMoiBiDanh(t *testing.T) {
	a := newAsk(t)
	director := user("director", "")
	for _, alias := range []string{"dầu nhớt", "nhớt", "dầu thủy lực", "dầu thuỷ lực", "dầu hộp số", "dầu bôi trơn", "mỡ bôi trơn", "dầu ăn",
		"dầu gội", "dầu gió", "dầu thơm"} {
		in := a.RT.Normalizer.Normalize("Hôm nay EX-03 nhận 20 L "+alias, normalization.Options{Now: today})
		var terms []string
		for _, sp := range in.SpansOf(contracts.ValueTerm) {
			terms = append(terms, sp.Value.Text+"="+sp.Raw)
		}
		if !slices.Contains(terms, "other_oil="+alias) || slices.ContainsFunc(terms, func(s string) bool { return strings.HasPrefix(s, "diesel_fuel=") }) {
			t.Errorf("%q: muốn đọc là vật tư khác (other_oil), không phải dầu diesel — thuật ngữ đọc được %v", alias, terms)
		}
		if !strings.Contains(alias, "dầu") {
			continue
		}
		q := "Hôm nay EX-03 nhận bao nhiêu " + alias + "?"
		if r := ask(t, a, director, q).Response; r.Route != contracts.RouteReject || r.Outcome != contracts.OutcomeOutOfScope {
			t.Errorf("%q: ngoài phạm vi, muốn từ chối, được [%s/%s] %q", q, r.Route, r.Outcome, r.Answer)
		}
	}
}

// Nợ S4 sửa sau H4 (lệnh ca MVP ALL-IN mục 3d; ADR-0010). (1) S4R-03: từ khoá không được khớp trong cụm ghép mang nghĩa khác mà bỏ qua
// không đổi nghĩa câu ("ghi nhận", "nhận xét" không phải "nhận"; "cấp bách" không phải "cấp"). Cụm mang nghĩa hệ chưa hỗ trợ ("dự báo",
// "báo giá", "cảnh báo", "nhà cung cấp" trong câu hỏi tiền) KHÔNG bỏ qua: câu đó vẫn phải hỏi lại, không được trả theo chỉ số còn lại
// (soát nợ S4 vòng 1, S1-1, S1-2). (2) F6': số không tự viết đơn vị mà ngoài miền một lần đo của chỉ số không phải giá trị của nó — tiền nói
// tắt ("4 củ", "4tr", "dầu 4,1 triệu", "dầu 150k") không còn bị hiểu là lít; lượng viết rõ đơn vị của cả tháng / tuần vẫn được (S2-1).
func TestS4NoSauH4(t *testing.T) {
	a := newAsk(t)
	director := user("director", "")
	for _, c := range []struct{ q, notIntent string }{
		{"Hôm nay EX-03 ghi nhận 3850 L dầu đúng không?", "fuel_issued"},
		{"Nhận xét về dầu của EX-03 hôm nay thế nào?", "fuel_issued"},
		{"EX-03 hôm nay cấp bách cần bao nhiêu dầu?", "fuel_issued"},
	} {
		r := ask(t, a, director, c.q).Response
		if r.Intent == c.notIntent || verdictOf(r.Answer) != "" ||
			(r.Route != contracts.RouteClarification && !strings.Contains(r.Answer, giaDinhChung)) {
			t.Errorf("%q: cụm ghép không phải từ khoá của %s — muốn hỏi lại hay trả lời theo ý định chung có nhãn, được [%s/%s/%s] %q", c.q,
				c.notIntent, r.Route, r.Outcome, r.Intent, r.Answer)
		}
	}
	// đối chứng dương: từ khoá đứng riêng vẫn khớp
	for _, c := range []struct{ q, intent string }{
		{"EX-03 hôm nay được cấp bao nhiêu dầu?", "fuel_issued"},
		{"EX-03 hôm nay nhận bao nhiêu dầu?", "fuel_issued"},
		{"Đội báo EX-03 hôm nay bao nhiêu dầu?", "fuel_reported"},
	} {
		if r := ask(t, a, director, c.q).Response; r.Intent != c.intent {
			t.Errorf("%q: muốn ý định %s, được [%s/%s] %q", c.q, c.intent, r.Route, r.Intent, r.Answer)
		}
	}
	// nghĩa hệ chưa hỗ trợ (tiền, dự báo): hỏi lại, không trả số của chỉ số còn lại
	for _, q := range []string{
		"Tháng này thanh toán cho nhà cung cấp dầu của EX-03 hết mấy triệu?",
		"Dự báo tiêu thụ dầu EX-03 tuần sau thế nào?",
	} {
		if r := ask(t, a, director, q).Response; r.Route != contracts.RouteClarification {
			t.Errorf("%q: nghĩa hệ chưa hỗ trợ — muốn hỏi lại, được [%s/%s/%s] %q", q, r.Route, r.Outcome, r.Intent, r.Answer)
		}
	}
	// tiền nói tắt / số ngoài miền chỉ số: hỏi lại nghĩa số liệu, không trả số kho xuất
	for _, q := range []string{
		"Dầu xuất cho EX-03 hôm qua 4 củ đúng không?",
		"Hôm nay EX-03 xuất dầu 4tr đúng không?",
		"Hôm nay EX-03 xuất 4,1 triệu dầu đúng không?",
		"Hôm nay EX-03 xuất dầu 150k đúng không?",
	} {
		r := ask(t, a, director, q).Response
		if r.Route != contracts.RouteClarification || strings.Contains(r.Answer, giaDinhChung) {
			t.Errorf("%q: số ngoài miền một lần đo của dầu kho xuất — muốn hỏi lại, được [%s/%s/%s] %q", q, r.Route, r.Outcome, r.Intent, r.Answer)
		}
	}
	// đối chứng dương: lượng trong miền, và lượng viết rõ đơn vị của cả tháng / tuần, vẫn trả lời theo ý định chung kèm câu giả định
	for _, q := range []string{"Hôm nay EX-03 xuất 4k dầu đúng không?", "Hôm nay EX-03 xuất dầu 3850 đúng không?",
		"Tháng 9 EX-03 xuất 80000 L dầu đúng không?", "Tuần này EX-03 xuất 25000 L dầu đúng không?"} {
		if r := ask(t, a, director, q).Response; r.Intent != "fuel_generic" || !strings.Contains(r.Answer, giaDinhChung) || verdictOf(r.Answer) != "" {
			t.Errorf("%q: muốn trả lời theo ý định chung kèm câu giả định, không phán quyết — được [%s/%s] %q", q, r.Route, r.Intent, r.Answer)
		}
	}
}
