package construction_test

import (
	"slices"
	"strings"
	"testing"

	"github.com/luannt2002/enterprise-ai-control-layer/core/contracts"
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
