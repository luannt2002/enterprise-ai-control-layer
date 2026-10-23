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
