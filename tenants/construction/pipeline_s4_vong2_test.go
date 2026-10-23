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
