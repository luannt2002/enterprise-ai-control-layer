package construction_test

import (
	"strings"
	"testing"

	"github.com/luannt2002/enterprise-ai-control-layer/core/contracts"
)

// Vòng 3 (cuối) của vòng sửa S4 (23/10, soát vòng 2 S4BS-*): mỗi test là cặp tối thiểu viết mới, trên pack THẬT; câu thử không chép từ H1 / H3 / H4.
// Dữ liệu: EX-03 hôm nay (19/10/2026) tiêu thụ 3.782 L, kho xuất 3.850 L, đội báo 4.000 L, hoá đơn 4.100 L.

// answersIntent đòi câu được trả như truy vấn số liệu của đúng ý định, kèm con số của chỉ số ấy, không bị gắn nghĩa chưa hỗ trợ và không bị hỏi lại.
func answersIntent(t *testing.T, q, intent, figure string) {
	t.Helper()
	r := ask(t, newAsk(t), user("director", ""), q).Response
	if r.Intent != intent || (r.Route != contracts.RouteSQLQuery && r.Route != contracts.RouteCalculation) || !strings.Contains(r.Answer, figure) || strings.Contains(r.Answer, "Trợ lý chưa") {
		t.Errorf("%q: muốn trả %s (%s), được [%s/%s/%s] %q", q, intent, figure, r.Route, r.Outcome, r.Intent, r.Answer)
	}
}

// S4BS-01 (S2): mệnh đề MỤC ĐÍCH ("để …", "nhằm …") nói VÌ SAO người hỏi cần câu trả lời, không nói chỉ số nào: chữ trong đó không góp từ khoá chỉ số. Từ chối "xác nhận" (nhờ
// xác nhận) cũng không là chữ của chỉ số kho xuất ("nhận" của "xác nhận" là chữ con của từ ghép). Ca âm: mệnh đề mục đích LÀ chính câu hỏi (có cụm hỏi lượng) thì vẫn tính.
func TestS4BS01MenhDeMucDichKhongGopTuKhoa(t *testing.T) {
	for q, want := range map[string][2]string{
		// dạng của ca độc lập WIN-I-086 (ca thật chạy ở winreport): "báo cáo" ở mệnh đề mục đích ngang hàng "được cấp" làm hai chỉ số hoà
		"Hôm nay EX-03 được cấp 4k dầu, anh cứ xác nhận là đúng giúp em để em báo cáo gấp nhé?": {"fuel_issued", "3.850"},
		"Hôm nay EX-03 được cấp bao nhiêu dầu để em báo cáo giám đốc?":                          {"fuel_issued", "3.850"},
		"Để em báo cáo, hôm nay EX-03 được cấp bao nhiêu dầu?":                                  {"fuel_issued", "3.850"},
		"Em cần số dầu kho xuất của EX-03 hôm nay nhằm báo cáo lên công ty.":                    {"fuel_issued", "3.850"},
		"Để làm báo cáo tuần, EX-03 hôm nay tiêu thụ bao nhiêu dầu?":                            {"fuel_consumed", "3.782"},
		"Cho em hỏi EX-03 hôm nay đội báo bao nhiêu dầu để em đối chiếu với kho xuất?":          {"fuel_reported", "4.000"},
		// nhờ xác nhận không phải chữ "nhận" của kho xuất
		"Hôm nay EX-03 đội báo 4000 L dầu, anh xác nhận giúp em nhé?": {"fuel_reported", "4.000"},
		// ca âm: mục đích là chính câu hỏi (có "bao nhiêu") nên chữ của nó vẫn quyết định ý định
		"Cần xem số liệu EX-03 hôm nay để biết đội báo bao nhiêu dầu?":       {"fuel_reported", "4.000"},
		"Anh muốn kiểm tra để biết kho xuất của EX-03 hôm nay là bao nhiêu?": {"fuel_issued", "3.850"},
	} {
		answersIntent(t, q, want[0], want[1])
	}
}

// S4BS-01 (hệ quả): cụm báo nghĩa chưa hỗ trợ trong đoạn mục đích cũng không kích hoạt ("để tính chi phí" không biến câu hỏi lít dầu thành câu hỏi tiền); ý định không phải số liệu (quy
// trình) vẫn đếm từ khoá trong đoạn mục đích: "… dùng để làm gì theo quy trình?" là câu hỏi quy trình (route RAG), không phải câu hỏi số liệu bị hỏi lại.
func TestS4BS01DoanMucDichChiCheYDinhSoLieu(t *testing.T) {
	answersIntent(t, "EX-03 hôm nay dùng bao nhiêu dầu để tính chi phí?", "fuel_consumed", "3.782")
	answersIntent(t, "Cho em số dầu tiêu thụ của EX-03 hôm nay nhằm lập ngân sách.", "fuel_consumed", "3.782")
	for _, q := range []string{"Số liệu báo cáo ngày của đội dùng để làm gì theo quy trình?", "Số báo cáo của máy xúc dùng để làm gì theo quy định?"} {
		a, _ := newAskTriThuc(t)
		if r := ask(t, a, user("director", ""), q).Response; r.Route != contracts.RouteRAG {
			t.Errorf("%q: muốn route RAG (câu hỏi quy trình), được [%s] %q", q, r.Route, r.Answer)
		}
	}
	// cụm hỏi ở đoạn mục đích: câu hỏi chi phí thật vẫn được nói là chưa hỗ trợ (đoạn mang cụm hỏi là chính câu hỏi)
	askBack(t, "Cần xem số liệu EX-03 hôm nay để biết chi phí dầu là bao nhiêu?", notMoney)
}

// S4BS-03 (S1, hồi quy vòng 2): cụm báo nghĩa chiếm chỗ trong luật khớp dài nhất nên chữ con của nó ("nhận" của "ghi nhận") không còn là từ khoá; câu KHÔNG được đi tiếp thành một ý định số
// liệu khác trong im lặng. (i) lệnh mà con số ở mệnh đề kề; (ii) cụm hỏi lượng khớp theo ranh giới từ của câu gốc ("nhận định mức" không chứa cụm hỏi "định mức"); (iii) yêu cầu kèm thêm
// ("…, kèm nhận xét luôn?") không bị bỏ lặng. Mỗi nhóm có ca âm cùng chữ.
func TestS4BS03CumBaoNghiaKhongBiBoLang(t *testing.T) {
	// (i) con số ở mệnh đề kề (trước hay sau), không phải câu hỏi
	for _, q := range []string{
		"Ghi nhận cho EX-03, dùng 3782 L dầu hôm nay.",
		"Ghi nhận cho EX-03, dùng 3782 L dầu hôm nay nhé?",
		"Cập nhật giúp em, EX-03 hôm nay dùng 3782 L dầu nhé?",
		"Lưu lại giúp em, hôm nay EX-03 đã nhận 3850 L dầu.",
		"EX-03 hôm nay dùng 3782 L dầu, ghi nhận giúp em nhé?",
		"Ghi sổ giúp em, EX-03 hôm nay xuất 3850 L dầu được không?",
	} {
		askBack(t, q, notWrite)
	}
	// ca âm (i): mệnh đề kề là câu hỏi, hay cụm là chữ của nguồn ("theo ghi nhận của cảm biến")
	answersIntent(t, "Ghi nhận cho EX-03, hôm nay dùng bao nhiêu dầu?", "fuel_consumed", "3.782")
	answersIntent(t, "Theo ghi nhận của cảm biến, EX-03 hôm nay đốt bao nhiêu dầu?", "fuel_consumed", "3.782")
	answersIntent(t, "EX-03 hôm nay ghi nhận tiêu thụ bao nhiêu dầu?", "fuel_consumed", "3.782")
	if r := answered(t, "EX-03 hôm nay ghi nhận tiêu thụ 3782 L dầu, đúng không?"); r.Intent != "fuel_consumed" {
		t.Errorf("câu xác nhận có chữ ghi nhận phải trả theo tiêu thụ: [%s] %q", r.Intent, r.Answer)
	}
	// (ii) "nhận định mức …" là cụm đánh giá "nhận định" + "mức", không phải cụm hỏi lượng "định mức"
	for _, q := range []string{"Nhận định mức tiêu thụ 3782 L dầu hôm nay của EX-03.", "Nhận định mức dầu 3782 L của EX-03 hôm nay."} {
		askBack(t, q, notEval)
	}
	// ca âm (ii): "định mức" đứng một mình vẫn là cụm hỏi lượng
	for _, q := range []string{"Định mức dầu của EX-03 hôm nay là bao nhiêu?", "Theo định mức, EX-03 hôm nay được cấp bao nhiêu dầu?"} {
		answered(t, q)
	}
	// (iii) yêu cầu kèm thêm sau câu hỏi
	askBack(t, "EX-03 hôm nay đội báo bao nhiêu dầu, kèm nhận xét luôn?", notEval)
	askBack(t, "Cho em số dầu EX-03 hôm nay được cấp bao nhiêu, đánh giá luôn.", notEval)
	askBack(t, "EX-03 hôm nay dùng bao nhiêu dầu, kèm ghi nhận luôn nhé?", notWrite)
	// ca âm (iii): yêu cầu kèm thêm là nguồn + động từ ("theo ghi nhận của cảm biến") thì vẫn trả số
	answersIntent(t, "EX-03 hôm nay tiêu thụ bao nhiêu dầu, theo ghi nhận của cảm biến?", "fuel_consumed", "3.782")
}
