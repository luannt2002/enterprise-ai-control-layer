package construction_test

import (
	"strings"
	"testing"

	"github.com/luannt2002/enterprise-ai-control-layer/core/contracts"
)

// Vòng 2 của vòng sửa S4 (23/10, soát vòng 1 S4AS-*): mỗi test là cặp tối thiểu viết mới, trên pack THẬT; câu thử không chép từ H1 / H3 / H4.
// Dữ liệu: EX-03 hôm nay (19/10/2026) tiêu thụ 3.782 L, kho xuất 3.850 L, đội báo 4.000 L, hoá đơn 4.100 L.

// unsupportedNote: câu hỏi lại nói rõ nghĩa hệ chưa hỗ trợ — chữ do pack viết (templates.yaml), không phải chữ của mã.
const (
	notWrite = "Trợ lý chưa ghi số liệu"
	notMoney = "Trợ lý chưa tra được số tiền"
)

// askBack báo câu bị hỏi lại kèm câu "Trợ lý chưa …" của nghĩa tương ứng và KHÔNG đọc lại con số nào của người hỏi (không giả vờ đã ghi).
func askBack(t *testing.T, q, note string) {
	t.Helper()
	r := ask(t, newAsk(t), user("director", ""), q).Response
	if r.Route != contracts.RouteClarification || !strings.Contains(r.Answer, note) {
		t.Errorf("%q: muốn hỏi lại kèm %q, được [%s/%s] %q", q, note, r.Route, r.Outcome, r.Answer)
		return
	}
	for _, leak := range []string{"3.782", "3.850", "3.900"} {
		if strings.Contains(r.Answer, leak) {
			t.Errorf("%q: hỏi lại mà vẫn đọc lại con số %s: %q", q, leak, r.Answer)
		}
	}
}

// answered báo câu được trả lời như câu hỏi số liệu (không phải hỏi lại về nghĩa chưa hỗ trợ).
func answered(t *testing.T, q string) contracts.AskResponse {
	t.Helper()
	r := ask(t, newAsk(t), user("director", ""), q).Response
	if strings.Contains(r.Answer, "Trợ lý chưa") {
		t.Errorf("%q: không phải lệnh mà bị nói là nghĩa chưa hỗ trợ: %q", q, r.Answer)
	}
	return r
}

// S4AS-01 (D-OP-18 = c): lệnh ghi LỊCH SỰ có giá trị kèm "?" vẫn là lệnh ghi. Dấu "?" một mình không làm câu thành câu hỏi; câu hỏi là câu có cụm xác nhận hay
// cụm hỏi lượng của pack. Cặp tối thiểu: "Ghi nhận … 3782 L dầu nhé?" (lệnh) / "ERP ghi nhận … 3900 L, đúng không?" ("ERP ghi nhận" là nguồn, câu hỏi xác nhận).
func TestS4AS01LenhGhiLichSuCoDauHoiVanLaLenh(t *testing.T) {
	for _, q := range []string{
		"Ghi nhận EX-03 hôm nay dùng 3782 L dầu nhé?",
		"Ghi nhận giúp em EX-03 hôm nay dùng 3782 L dầu nhé?",
		"Anh cập nhật EX-03 hôm nay đốt 3900 lít dầu được không?",
		"Lưu lại EX-03 hôm nay nhận 3850 L dầu giúp em với?",
		"Ghi nhận EX-03 hôm nay đốt 3782 L dầu?", // "?" một mình
	} {
		askBack(t, q, notWrite)
	}
	// đối chứng: câu có cụm xác nhận / cụm hỏi lượng là câu hỏi dù có chữ "ghi nhận"
	for _, q := range []string{
		"ERP ghi nhận EX-03 hôm nay xuất 3900 L, đúng không?",
		"EX-03 hôm nay ghi nhận tiêu thụ bao nhiêu dầu?",
		"Hôm nay EX-03 ghi nhận được bao nhiêu lít dầu?",
	} {
		answered(t, q)
	}
	// "?" không phải thứ làm câu hỏi thường thành lệnh: câu hỏi lịch sự không cụm ghi vẫn trả số
	if r := answered(t, "Cho em hỏi EX-03 hôm nay dùng bao nhiêu dầu nhé?"); r.Intent != "fuel_consumed" || !strings.Contains(r.Answer, "3.782") {
		t.Errorf("câu hỏi lịch sự phải trả số tiêu thụ: [%s] %q", r.Intent, r.Answer)
	}
}

// Các câu nói nghĩa chưa hỗ trợ của từng họ — chữ do pack viết (templates.yaml).
const (
	notForecast = "Trợ lý chưa dự báo"
	notQuote    = "Trợ lý chưa báo giá"
	notWarning  = "Trợ lý chưa đưa cảnh báo"
	notEval     = "Trợ lý chưa nhận xét hay đánh giá"
)

// queryAnswered đòi câu được trả như một truy vấn số liệu thật (route SQL_QUERY / CALCULATION), không bị gắn nghĩa chưa hỗ trợ.
func queryAnswered(t *testing.T, q string) {
	t.Helper()
	r := answered(t, q)
	if r.Route != contracts.RouteSQLQuery && r.Route != contracts.RouteCalculation {
		t.Errorf("%q: muốn trả như truy vấn số liệu, được [%s/%s] %q", q, r.Route, r.Outcome, r.Answer)
	}
}

// S4AS-02 (S1): nghĩa TIỀN nhận theo CẤU TRÚC chứ không chỉ theo chữ trong danh sách cụm: span kiểu tiền của bộ chuẩn hoá ("4tr đồng", "nghìn đồng"),
// "bao nhiêu" + đơn vị tiền của lexicon ("bao nhiêu đồng / VND / vnđ"), và cụm theo họ nghĩa ("đơn giá", "giá dầu", "kinh phí"). Câu nêu cả lượng cùng chiều đo
// ("3850 L … 25.000 đồng") vẫn là câu về lượng. Ca âm đi cặp: "đồng hồ", "1,2 triệu lít" (lượng thể tích viết rõ đơn vị), "báo cáo giá trị".
func TestS4AS02NghiaTienTheoCauTruc(t *testing.T) {
	for _, q := range []string{
		"EX-03 hôm nay dùng dầu hết bao nhiêu đồng?",
		"EX-03 hôm nay đốt dầu hết bao nhiêu VND?",
		"EX-03 hôm nay đốt dầu hết bao nhiêu vnđ?",
		"Hôm nay EX-03 dùng dầu hết bao nhiêu đồng vậy?",
		"EX-03 hôm nay đốt dầu hết 4tr đồng, đúng không?",
		"EX-03 hôm nay dùng dầu hết bao nhiêu nghìn đồng?",
		"Đơn giá dầu EX-03 hôm nay là bao nhiêu?",
		"Giá dầu hôm nay cho EX-03 là bao nhiêu?",
		"Kinh phí dầu EX-03 hôm nay là bao nhiêu?",
	} {
		askBack(t, q, notMoney)
	}
	for _, q := range []string{
		"Đồng hồ đo dầu của EX-03 hôm nay hiển thị bao nhiêu dầu?",
		"Tháng 9 dự án Cầu A nhận 1,2 triệu lít dầu phải không?",
		"EX-03 hôm nay dùng 0,004 triệu lít dầu, đúng không?",
		"Báo cáo giá trị dầu EX-03 hôm nay là bao nhiêu?",
		"EX-03 hôm nay được cấp 3850 L dầu 25.000 đồng đúng không?", // lượng cùng chiều đo: con số tiền là phụ
	} {
		queryAnswered(t, q)
	}
	// "đồng" hết nghĩa tiền khi là chữ đầu của từ ghép: đứng sau "bao nhiêu" mà còn chữ khác trong mệnh đề thì không phải đơn vị tiền
	if r := answered(t, "EX-03 hôm nay đồng hồ ghi bao nhiêu dầu tiêu thụ?"); r.Route != contracts.RouteSQLQuery {
		t.Errorf("đồng hồ: muốn trả truy vấn, được [%s] %q", r.Route, r.Answer)
	}
	for _, q := range []string{"EX-03 hôm nay lắp bao nhiêu đồng hồ đo dầu?", "Dự án Cầu A hôm nay có bao nhiêu đồng nghiệp dùng dầu của EX-03?"} {
		if r := ask(t, newAsk(t), user("director", ""), q).Response; strings.Contains(r.Answer, notMoney) {
			t.Errorf("%q: từ ghép của \"đồng\" không phải đơn vị tiền: %q", q, r.Answer)
		}
	}
	// giá của đại lượng khác dầu (nhân công) vẫn ngoài phạm vi, không bị hỏi lại về tiền dầu
	if r := ask(t, newAsk(t), user("director", ""), "Giá nhân công dự án B tháng này bao nhiêu?").Response; r.Route != contracts.RouteReject {
		t.Errorf("giá nhân công phải bị từ chối ngoài phạm vi, được [%s] %q", r.Route, r.Answer)
	}
}

// S4AS-02 (họ nghĩa): các nghĩa còn lại mở danh sách THEO HỌ NGHĨA (đánh đổi ghi ở ADR-0010: danh sách theo họ nằm ở pack, độ phủ do H6 độc lập đo). Mỗi họ có ca
// dương ngoài danh sách của vòng 1 và ca âm cùng chữ ("cập nhật" không giá trị là câu hỏi; "báo cáo giá trị" không phải báo giá).
func TestS4AS02HoNghiaMoRong(t *testing.T) {
	for note, qs := range map[string][]string{
		notForecast: {"Dự tính EX-03 hôm nay dùng bao nhiêu dầu?", "EX-03 sẽ dùng bao nhiêu dầu hôm nay?", "Dự phóng lượng dầu EX-03 hôm nay?"},
		notQuote:    {"Hỏi giá dầu EX-03 hôm nay", "Xin giá dầu cho EX-03 hôm nay", "Chào giá dầu EX-03 hôm nay"},
		notWarning:  {"Nhắc tôi khi EX-03 dùng dầu vượt 4000 L", "EX-03 hôm nay có báo động dầu bất thường không?", "Nhắc nhở giúp mình khi EX-03 dùng nhiều dầu"},
		notWrite: {"Nhập số liệu: EX-03 hôm nay dùng 3782 L dầu.", "Ghi sổ EX-03 hôm nay xuất 3850 L dầu", "Sửa số dầu EX-03 hôm nay thành 3782 L",
			"Khai báo EX-03 hôm nay dùng 3782 L dầu", "Điều chỉnh số dầu EX-03 hôm nay thành 3850 L"},
		notEval: {"EX-03 hôm nay dùng 3782 L dầu có hợp lý không?", "Nhận định EX-03 hôm nay đốt 3782 L dầu", "Bình luận EX-03 hôm nay dùng 3782 L dầu",
			"Phân tích EX-03 hôm nay tiêu thụ 3782 L dầu", "EX-03 hôm nay đốt 3900 L dầu có bất thường không?"},
	} {
		for _, q := range qs {
			askBack(t, q, note)
		}
	}
	// ca âm: cùng chữ nhưng không phải nghĩa đó
	for _, q := range []string{
		"EX-03 hôm nay có cập nhật nào về dầu không? Dùng bao nhiêu?", // "cập nhật" không kèm giá trị
		"Hôm nay EX-03 phân tích dầu theo đội báo bao nhiêu?",         // không con số nào để đánh giá
	} {
		queryAnswered(t, q)
	}
}

// intentOf trả câu trả lời của một câu hỏi thường kèm ý định; đòi route truy vấn số liệu và con số mong đợi (kể cả dạng có dấu chấm nghìn).
func wantMetric(t *testing.T, q, intent, number string) {
	t.Helper()
	r := ask(t, newAsk(t), user("director", ""), q).Response
	if r.Intent != intent || !strings.Contains(r.Answer, number) || strings.Contains(r.Answer, "Trợ lý chưa") {
		t.Errorf("%q: muốn ý định %s và số %s, được [%s/%s] %q", q, intent, number, r.Route, r.Intent, r.Answer)
	}
}

// askBackMeaning đòi câu bị hỏi lại (không trả con số nào, không từ chối): hoà ý định / thiếu nghĩa số liệu.
func askBackMeaning(t *testing.T, q string) {
	t.Helper()
	r := ask(t, newAsk(t), user("director", ""), q).Response
	if r.Route != contracts.RouteClarification {
		t.Errorf("%q: muốn hỏi lại, được [%s/%s] %q", q, r.Route, r.Intent, r.Answer)
		return
	}
	for _, leak := range []string{"3.782", "3.850", "4.000", "4.100"} {
		if strings.Contains(r.Answer, leak) {
			t.Errorf("%q: hỏi lại mà vẫn trả số %s: %q", q, leak, r.Answer)
		}
	}
}

// S4AS-03 (C4, ADR-0010): phép so sánh nhận theo CẤU TRÚC — hai chỉ số khác nhau trong cùng mệnh đề + từ so sánh bậc "hơn / kém" (hay "… và … chênh / lệch") — và câu nêu ≥ 2
// chỉ số mà không xác định được cặp thì HOÀ ⇒ hỏi lại: số cụm từ khoá khớp không phải tín hiệu nghĩa nên không được quyết định thay người hỏi.
func TestS4AS03SoSanhTheoCauTruc(t *testing.T) {
	// dương: từ so sánh ngoài danh sách vòng 1 ("cao hơn", "thấp hơn", "lớn hơn", "kém"), đúng cặp, đúng chiều (vế nhắc trước − vế nhắc sau)
	for _, c := range []struct{ q, intent, number string }{
		{"EX-03 hôm nay tiêu thụ đo được cao hơn hoá đơn bao nhiêu?", "fuel_consumed", "318"},
		{"EX-03 hôm nay hoá đơn thấp hơn tiêu thụ đo được bao nhiêu?", "fuel_billed", "318"},
		{"EX-03 hôm nay đội báo lớn hơn kho xuất bao nhiêu?", "fuel_reported", "150"},
		{"EX-03 hôm nay cảm biến kém kho xuất bao nhiêu?", "fuel_consumed", "68"},
		{"EX-03 hôm nay hoá đơn và cảm biến chênh nhau bao nhiêu?", "fuel_billed", "318"},
	} {
		r := ask(t, newAsk(t), user("director", ""), c.q).Response
		if r.Route != contracts.RouteCalculation || !strings.Contains(r.Answer, c.number) {
			t.Errorf("%q: muốn phép so sánh ra %s, được [%s/%s] %q", c.q, c.number, r.Route, r.Intent, r.Answer)
		}
	}
	// âm: nêu hai chỉ số mà KHÔNG có từ so sánh / chênh ⇒ hoà ⇒ hỏi lại, dù một vế có nhiều cụm từ khoá hơn ("tiêu thụ" + "đo được" so với "kho xuất")
	for _, q := range []string{
		"Cho biết tiêu thụ đo được và kho xuất của EX-03 hôm nay?",
		"EX-03 hôm nay đội báo tiêu thụ bao nhiêu dầu?",
		"EX-03 hôm nay cảm biến đo được hay được cấp bao nhiêu dầu?",
		"EX-03 hôm nay tiêu thụ đo được hay kho xuất bao nhiêu?",
		"EX-03 hôm nay đội báo, hoá đơn, kho xuất lần lượt bao nhiêu?",
	} {
		askBackMeaning(t, q)
	}
	// "cảm biến báo" = cảm biến ghi nhận: "báo" là động từ của nguồn, không phải chỉ số đội báo (cụm dài thắng) — cặp tối thiểu với "đội báo", chỉ số thật thứ hai
	wantMetric(t, "Cảm biến báo EX-03 hôm nay đo được bao nhiêu dầu?", "fuel_consumed", "3.782")
	askBackMeaning(t, "Đội báo EX-03 hôm nay đo được bao nhiêu dầu?")
	// từ so sánh ở mệnh đề KHÁC không làm hai chỉ số ở mệnh đề kia thành phép so sánh
	askBackMeaning(t, "Đội báo và kho xuất của EX-03 hôm nay là bao nhiêu, hơn hôm qua không?")
}

// S4AS-03 (C3): cụm xác nhận + thực thể + lượng cùng chiều đo là tín hiệu phạm vi đủ — không đòi mảnh từ khoá của chỉ số: câu có động từ lạ ("xài", "bơm") hỏi lại
// nghĩa số liệu thay vì bị từ chối ngoài phạm vi. Đối chứng: lượng khác chiều đo ("20 tấn") hay không có thực thể vẫn ngoài phạm vi.
func TestS4AS03PhamViCauXacNhanLuongCungChieuDo(t *testing.T) {
	for _, q := range []string{
		"EX-03 hôm nay xài 3850 L đúng không?",
		"EX-03 hôm nay bơm 3850 L đúng không?",
		"EX-03 hôm nay đổ 3850 L đúng không?",
		"RL-02 hôm nay hết 1200 L đúng không?",
		"Dự án Cầu A hôm nay xài 5050 L đúng không?",
	} {
		askBackMeaning(t, q)
	}
	for _, q := range []string{
		"EX-03 hôm nay nặng 20 tấn đúng không?",
		"Hôm qua máy bay đổ 3850 L đúng không?",
		"VN-Index hôm nay giảm 3850 điểm đúng không?",
	} {
		if r := ask(t, newAsk(t), user("director", ""), q).Response; r.Route != contracts.RouteReject {
			t.Errorf("%q: muốn từ chối ngoài phạm vi, được [%s] %q", q, r.Route, r.Answer)
		}
	}
}

// S4AS-03 (C5): mệnh đề hỏi nhận theo cấu trúc — cụm hỏi lượng ("mấy lít") cũng đánh dấu mệnh đề hỏi; mệnh đề ngữ cảnh nêu con số cả TRƯỚC lẫn SAU mệnh đề hỏi không
// quyết định ý định.
func TestS4AS03MenhDeHoiVaNguCanhTruocSau(t *testing.T) {
	for _, c := range []struct{ q, intent, number string }{
		{"EX-03 hôm nay được cấp mấy lít dầu?", "fuel_issued", "3.850"},
		{"EX-03 hôm nay đốt mấy lít dầu?", "fuel_consumed", "3.782"},
		{"Phiếu kho ghi 3.850, vậy EX-03 hôm nay đo được mấy lít dầu?", "fuel_consumed", "3.782"},
		{"EX-03 hôm nay được cấp bao nhiêu dầu, biết đội báo 4000 L?", "fuel_issued", "3.850"},
		{"EX-03 hôm nay tiêu thụ bao nhiêu dầu, phiếu kho ghi 3.850?", "fuel_consumed", "3.782"},
		{"EX-03 hôm nay hoá đơn bao nhiêu dầu, cảm biến đo 3.782 L?", "fuel_billed", "4.100"},
	} {
		wantMetric(t, c.q, c.intent, c.number)
	}
	// mệnh đề sau KHÔNG có con số vẫn là phần bổ nghĩa của câu hỏi, nên ở lại: "đội báo … theo hoá đơn" nêu hai chỉ số ⇒ hỏi lại
	askBackMeaning(t, "EX-03 hôm nay đội báo bao nhiêu dầu, theo hoá đơn?")
}

// S4AS-04 (S2): cụm báo nghĩa chưa hỗ trợ nằm ở MỆNH ĐỀ NGỮ CẢNH (đứng trước mệnh đề hỏi, có con số) vẫn phải được nói ra — nếu không câu trả lời đúng cho phần hỏi nhưng im
// về phần lệnh / đánh giá / cảnh báo, và đọc lại đúng con số người dùng vừa "ghi". Cặp tối thiểu: lệnh ghi đứng đầu mệnh đề ("Ghi nhận …, vậy …") / nguồn + động từ
// ("Phiếu kho ghi nhận 3850 L, vậy …" — nguồn của con số, không phải lệnh).
func TestS4AS04CumNghiaOMenhDeNguCanhVanDuocNoiRa(t *testing.T) {
	for note, qs := range map[string][]string{
		notWrite: {
			"Ghi nhận EX-03 hôm nay dùng 3782 L dầu. Vậy hôm nay EX-03 tiêu thụ bao nhiêu dầu?",
			"Cập nhật giúp em: EX-03 hôm nay xuất 3850 L dầu, vậy EX-03 hôm nay được cấp bao nhiêu dầu?",
			"Lưu lại EX-03 hôm nay nhận 3850 L dầu, sau đó cho biết EX-03 hôm nay được cấp bao nhiêu dầu?",
			"Anh ghi nhận EX-03 hôm nay đốt 3900 L dầu, rồi cho em biết EX-03 hôm nay đốt bao nhiêu dầu?",
		},
		notEval:     {"Nhận xét: EX-03 hôm nay dùng 3782 L dầu, vậy EX-03 hôm nay tiêu thụ bao nhiêu dầu?", "Đánh giá EX-03 hôm nay đốt 3782 L dầu, vậy EX-03 hôm nay đo được bao nhiêu dầu?"},
		notWarning:  {"Cảnh báo: EX-03 dùng quá 4000 L dầu, vậy hôm nay EX-03 tiêu thụ bao nhiêu dầu?"},
		notForecast: {"Dự báo EX-03 dùng 4000 L dầu, vậy hôm nay EX-03 tiêu thụ bao nhiêu dầu?"},
		notQuote:    {"Báo giá dầu 20 triệu cho EX-03, vậy hôm nay EX-03 được cấp bao nhiêu dầu?"},
	} {
		for _, q := range qs {
			askBack(t, q, note)
		}
	}
	// ca âm: "<nguồn> ghi nhận <con số>" ở mệnh đề ngữ cảnh là nguồn của con số — câu hỏi ở mệnh đề sau vẫn được trả theo đúng chỉ số
	wantMetric(t, "Phiếu kho ghi nhận 3850 L, vậy EX-03 hôm nay đo được bao nhiêu dầu?", "fuel_consumed", "3.782")
	wantMetric(t, "Đội báo ghi nhận 4000 L, vậy EX-03 hôm nay được cấp bao nhiêu dầu?", "fuel_issued", "3.850")
}

// S4AS-05 (S3): (a) "chào" chỉ là lời chào khi câu không còn nội dung nào khác (không thực thể, không con số, không thuật ngữ): "chào cờ", "chào mừng khởi công dự án A", "chào hàng dầu …" không phải
// lời chào; (b) cụm báo nghĩa trong mệnh đề BỊ BỎ QUA ("không quan tâm", "để sau") không làm câu thành hỏi lại — cặp tối thiểu: cùng cụm, mệnh đề không bỏ qua thì vẫn hỏi lại.
func TestS4AS05LoiChaoVaMenhDeBiBoQua(t *testing.T) {
	for _, q := range []string{"Tôi chào cờ ở dự án Cầu A", "Chào mừng khởi công dự án Cầu A hôm nay", "Chào hàng dầu EX-03 hôm nay", "Hôm nay EX-03 chào cờ lúc mấy giờ?", "Họ chào cờ ở công trường EX-03 hôm nay"} {
		if r := ask(t, newAsk(t), user("director", ""), q).Response; r.Intent == "greeting" || strings.Contains(r.Answer, "Chào bạn") {
			t.Errorf("%q: không phải lời chào, được [%s/%s] %q", q, r.Route, r.Intent, r.Answer)
		}
	}
	for _, q := range []string{"Chào buổi sáng", "Xin chào", "Chào các bạn", "Hello", "Chào bạn nhé"} {
		if r := ask(t, newAsk(t), user("director", ""), q).Response; r.Intent != "greeting" {
			t.Errorf("%q: muốn lời chào, được [%s/%s] %q", q, r.Route, r.Intent, r.Answer)
		}
	}
	wantMetric(t, "Chi phí không quan tâm, EX-03 hôm nay tiêu thụ bao nhiêu dầu?", "fuel_consumed", "3.782")
	wantMetric(t, "Tiền dầu thì để sau, EX-03 hôm nay được cấp bao nhiêu lít?", "fuel_issued", "3.850")
	wantMetric(t, "Báo giá chưa cần, EX-03 hôm nay đội báo bao nhiêu dầu?", "fuel_reported", "4.000")
	wantMetric(t, "Dự báo khỏi cần, EX-03 hôm nay đo được bao nhiêu dầu?", "fuel_consumed", "3.782")
	// đối chứng: cùng cụm, không bị bỏ qua ⇒ hỏi lại; từ bỏ qua ở mệnh đề KHÁC không bỏ qua cụm này
	askBack(t, "Chi phí dầu EX-03 hôm nay là bao nhiêu?", notMoney)
	askBack(t, "Không quan tâm hôm qua, tiền dầu EX-03 hôm nay là bao nhiêu?", notMoney)
}

// S4AS-08 (S3): đường hỏi lại nhiều ý định (hoà / thiếu thuật ngữ) lọc nhãn theo capability như đường hỏi lại nghĩa số liệu — người không có capability của một ý định không thấy nhãn của nó
// ("Bạn muốn hỏi dầu kho xuất hay dầu đội báo …"); không còn ý định nào được xem ⇒ từ chối ngoài phạm vi, như câu không khớp. Vai driver không có capability nào.
func TestS4AS08HoiLaiNhieuYDinhLocNhanTheoCapability(t *testing.T) {
	a := newAsk(t)
	for _, c := range []struct{ name, q string }{
		{"hoà ý định", "Đội báo và phiếu kho của EX-03 hôm qua là 1.490 lít, đúng không?"},
		{"thiếu thuật ngữ", "Hôm nay nhận bao nhiêu?"},
		{"hoà ý định, hai chỉ số không dấu so sánh", "Cho biết tiêu thụ đo được và kho xuất của EX-03 hôm nay?"},
	} {
		// người có quyền: được hỏi lại và thấy nhãn ý định
		dir := ask(t, a, user("director", ""), c.q).Response
		if dir.Route != contracts.RouteClarification || !strings.Contains(dir.Answer, "Bạn muốn hỏi") {
			t.Fatalf("%s: tiền đề — director phải được hỏi lại có nhãn, được [%s] %q", c.name, dir.Route, dir.Answer)
		}
		// người không có capability nào: từ chối ngoài phạm vi, không lộ nhãn
		drv := ask(t, a, user("driver", ""), c.q).Response
		if drv.Route != contracts.RouteReject || drv.Outcome != contracts.OutcomeOutOfScope {
			t.Errorf("%s: driver không có capability, muốn từ chối ngoài phạm vi, được [%s/%s] %q", c.name, drv.Route, drv.Outcome, drv.Answer)
		}
		for _, label := range []string{"đội báo", "kho xuất", "tiêu thụ", "hoá đơn", "Bạn muốn hỏi"} {
			if strings.Contains(drv.Answer, label) {
				t.Errorf("%s: driver thấy nhãn ý định %q trong %q", c.name, label, drv.Answer)
			}
		}
	}
}
