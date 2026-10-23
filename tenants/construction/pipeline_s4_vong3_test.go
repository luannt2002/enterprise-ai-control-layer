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

// D-OP-18 = c ĐÚNG CHỮ (soát vòng 2, điều kiện (c)): thực thể + con số CÓ ĐƠN VỊ + KHÔNG có dấu hiệu hỏi / xác nhận ⇒ hỏi lại kèm "Trợ lý chưa ghi số liệu", BẤT KỂ động từ — không danh sách
// động từ. Dấu hiệu hỏi là cụm hỏi của pack, dấu "?", hay từ phủ định cuối câu ("… không"); câu nêu giá trị có dấu hiệu ấy là câu xác nhận, đi đường câu xác nhận sẵn có. 24 động từ NGOÀI danh
// sách cụm "ghi nhận / cập nhật / ghi sổ …" của pack, cộng các câu nêu giá trị không động từ nào.
func TestDOP18CTheoDungChuLenhGhiBatKeDongTu(t *testing.T) {
	for _, v := range []string{"Nhập", "Lưu", "Chốt số", "Tạo bản ghi", "Thêm số liệu", "Bổ sung", "Đăng ký", "Ghi hộ", "Ghi giùm em", "Note lại", "Gửi số liệu", "Điền số",
		"Nạp số liệu", "Đổi thành", "Sửa thành", "Vào sổ", "Ghi vô sổ", "Khai thêm", "Update", "Bù thêm", "Báo cáo giúp em", "Lập bản ghi", "Chép lại", "Đưa vào hệ thống"} {
		askBack(t, v+" EX-03 hôm nay dùng 3782 L dầu.", notWrite)
	}
	for _, q := range []string{
		"EX-03 hôm nay đổ 3850 L dầu.",
		"Hôm nay EX-03 tiêu thụ 3782 L dầu.",
		"Đội báo EX-03 hôm nay 4000 L dầu.",
		"Phiếu kho xuất EX-03 hôm nay 3850 L dầu.",
		"Hôm qua EX-03 được cấp 4k dầu.",
	} {
		askBack(t, q, notWrite)
	}
	// dấu hiệu hỏi / xác nhận ⇒ không phải lệnh ghi: câu xác nhận đi đường sẵn có (cụm xác nhận, "?", "… không" cuối câu), câu hỏi lượng trả số như thường
	for _, q := range []string{
		"Hôm nay EX-03 được cấp 3850 L dầu đúng không?",
		"EX-03 hôm nay đốt 3782 L dầu, đúng không?",
		"EX-03 hôm nay dùng bao nhiêu L dầu?",
		"Nhập EX-03 hôm nay dùng 3782 L dầu nhé?", // "?" là dấu hiệu hỏi (bất biến (b) của ADR-0010): động từ ngoài danh sách + "?" đi đường câu xác nhận
		"EX-03 hôm nay có đốt 3782 L dầu không",   // "không" cuối câu là đuôi hỏi có / không
		"Cho tôi hỏi EX-03 hôm nay có dùng quá 4000 L dầu không?",
		"Phiếu kho ghi 3850 L, vậy EX-03 hôm nay dùng bao nhiêu dầu",
		"EX-03 hôm nay dùng 3782 L dầu hay 3900 L dầu?",
		// con số có đơn vị của đại lượng KHÁC (giờ họp, tải trọng) không phải giá trị của chỉ số để ghi: câu là yêu cầu tra cứu không dấu hỏi, trả số như thường
		"Họp lúc 8h15, anh cho em số lít dầu EX-03 được cấp hôm nay với.",
		"Họp lúc 8h15, anh cho em số lít dầu cho máy 03 được cấp hôm qua với.", // "03" của "máy 03" đứng cạnh "dầu" bị gắn đơn vị lít: nằm trong tên thực thể nên không phải giá trị
		"EX-03 nặng 20 tấn, cho em số lít dầu EX-03 được cấp hôm nay.",
	} {
		answered(t, q)
	}
	// lệnh ghi chỉ nói MỘT nghĩa: câu đã bị cụm báo nghĩa khác kéo ("nhắc tôi" = cảnh báo) thì không thêm nhãn "chưa ghi số liệu" (nó chỉ là phương án cuối khi chưa nghĩa nào được báo)
	if r := ask(t, newAsk(t), user("director", ""), "Nhắc tôi khi EX-03 dùng quá 4000 L dầu").Response; !strings.Contains(r.Answer, notWarning) || strings.Contains(r.Answer, notWrite) {
		t.Errorf("cảnh báo phải chỉ mang nhãn cảnh báo: %q", r.Answer)
	}
	// không có con số có đơn vị thì không phải giá trị để ghi; động từ trong danh sách + giá trị vẫn hỏi lại như vòng 2
	answered(t, "Cho EX-03 hôm nay xuất bao nhiêu dầu?")
	askBack(t, "Ghi sổ EX-03 hôm nay xuất 3850 L dầu.", notWrite)
}

// S4BS-04 (S2): tín hiệu tiền theo cấu trúc rộng hơn. (a) đơn vị tiền ngoài "đồng / vnđ" ("đô", "đô la", "USD", "triệu", "tỷ", "nghìn", "trăm nghìn", "củ") đi với "bao nhiêu"; (b) "bao nhiêu" + đơn vị
// tiền Ở BẤT KỲ ĐÂU trong mệnh đề, không đòi đơn vị ở cuối; (c) số tiền viết rõ đơn vị ngoài lexicon cũ ("250 USD"). Ca âm: chữ đầu của từ ghép ("đồng hồ", "đồng bộ", "đồng nghiệp", "đô thị", "tỷ
// lệ") và lượng thể tích viết rõ đơn vị ("1,2 triệu lít", "bao nhiêu triệu lít", "bao nhiêu nghìn lít") không phải tiền.
func TestS4BS04TinHieuTienMoRong(t *testing.T) {
	for _, q := range []string{
		"EX-03 hôm nay đổ dầu hết bao nhiêu đô?",
		"EX-03 hôm nay đổ dầu hết bao nhiêu USD?",
		"EX-03 hôm nay đổ dầu hết bao nhiêu đô la?",
		"EX-03 hôm nay đốt dầu hết bao nhiêu triệu?",
		"EX-03 hôm nay đốt dầu hết bao nhiêu tỷ?",
		"EX-03 hôm nay đốt dầu hết bao nhiêu nghìn?",
		"EX-03 hôm nay đốt dầu hết bao nhiêu trăm nghìn?",
		"EX-03 hôm nay đốt dầu hết bao nhiêu củ?",
		"Hết bao nhiêu đồng cho dầu của EX-03 hôm nay?",
		"Dầu EX-03 hôm nay hết bao nhiêu đồng một lít?",
		"EX-03 hôm nay đổ dầu hết bao nhiêu đồng vậy anh?",
		"Tốn bao nhiêu đồng cho EX-03 đổ dầu hôm nay?",
		"Chi bao nhiêu triệu đồng cho dầu EX-03 hôm nay?",
		"EX-03 hôm nay đốt dầu hết 250 USD, đúng không?",
		"EX-03 hôm nay đốt dầu hết 250 đô đúng không?",
		"EX-03 hôm nay dùng 3782 L dầu, hết bao nhiêu đồng?", // có lượng cùng chiều đo nhưng chính câu hỏi là tiền
	} {
		askBack(t, q, notMoney)
	}
	for _, q := range []string{
		"EX-03 hôm nay lắp bao nhiêu đồng hồ đo dầu?",
		"Dự án Cầu A có bao nhiêu đồng nghiệp dùng dầu của EX-03 hôm nay?",
		"EX-03 hôm nay có bao nhiêu đồng bộ dữ liệu dầu?",
		"Hôm nay hạ tầng đô thị C có bao nhiêu đô thị dùng dầu của EX-03?",
		"EX-03 hôm nay đốt bao nhiêu triệu lít dầu?",
		"Công trình Cầu A đã dùng bao nhiêu nghìn lít dầu?",
		"Tháng 9 dự án Cầu A nhận 1,2 triệu lít dầu phải không?",
		"EX-03 hôm nay dùng 0,004 triệu lít dầu, đúng không?",
		"Tỷ lệ hao hụt dầu của EX-03 hôm nay là bao nhiêu?",
	} {
		answered(t, q)
	}
}

// S4BS-05 (S3, hỏi thừa): cụm báo nghĩa chưa hỗ trợ chỉ kích hoạt khi nằm ở mệnh đề HỎI — không ở đoạn mục đích (S4BS-01), mệnh đề điều kiện ("Nếu …,"), mệnh đề phủ định ("Không tính tiền,",
// "Chưa tính chi phí,") hay hoãn ("… em lo sau,", "… xong,"); cụm quá chung ("sẽ", "dự kiến", "dự tính", "ước tính" — weak_cues của pack) chỉ kích hoạt khi chính mệnh đề hỏi mang chúng VÀ câu hỏi về một chỉ số.
// Ca âm cùng chữ: cụm ở mệnh đề hỏi, và yêu cầu ở mệnh đề ngữ cảnh ("Dự báo giúp em, …", "Nhắc tôi nhé, …") vẫn được nói ra.
func TestS4BS05CumBaoNghiaChiKichHoatOMenhDeHoi(t *testing.T) {
	for q, want := range map[string][2]string{
		"Anh sẽ duyệt, EX-03 hôm nay hoá đơn bao nhiêu lít dầu?":        {"fuel_billed", "4.100"},
		"Chiều nay sẽ họp, EX-03 hôm nay tiêu thụ bao nhiêu dầu?":       {"fuel_consumed", "3.782"},
		"Dự kiến chiều họp, EX-03 hôm nay được cấp bao nhiêu dầu?":      {"fuel_issued", "3.850"},
		"Không tính tiền, EX-03 hôm nay dùng bao nhiêu dầu?":            {"fuel_consumed", "3.782"},
		"Chưa tính chi phí, EX-03 hôm nay được cấp bao nhiêu dầu?":      {"fuel_issued", "3.850"},
		"Chi phí em lo sau, EX-03 hôm nay dùng bao nhiêu dầu?":          {"fuel_consumed", "3.782"},
		"Tiền em tính sau, EX-03 hôm nay tiêu thụ bao nhiêu dầu?":       {"fuel_consumed", "3.782"},
		"Quyết toán kỳ trước xong, EX-03 hôm nay dùng bao nhiêu dầu?":   {"fuel_consumed", "3.782"},
		"Trị giá hợp đồng không đổi, EX-03 hôm nay dùng bao nhiêu dầu?": {"fuel_consumed", "3.782"},
		"Nếu dự báo mưa, EX-03 hôm nay dùng bao nhiêu dầu?":             {"fuel_consumed", "3.782"},
		"Giả sử có cảnh báo, EX-03 hôm nay được cấp bao nhiêu dầu?":     {"fuel_issued", "3.850"},
	} {
		answersIntent(t, q, want[0], want[1])
	}
	// "sẽ" mà câu không hỏi chỉ số nào là chuyện khác: ngoài phạm vi (không phải câu dự báo, không hỏi lại về dầu)
	for _, q := range []string{"EX-03 sẽ về công trường lúc mấy giờ?", "Dự kiến lúc mấy giờ EX-03 về công trường?"} {
		if r := ask(t, newAsk(t), user("director", ""), q).Response; r.Route != contracts.RouteReject {
			t.Errorf("%q: muốn từ chối ngoài phạm vi, được [%s] %q", q, r.Route, r.Answer)
		}
	}
	// ca âm: cụm ở chính mệnh đề hỏi (có chỉ số), hay yêu cầu ở mệnh đề ngữ cảnh
	askBack(t, "EX-03 sẽ dùng bao nhiêu dầu hôm nay?", notForecast)
	askBack(t, "EX-03 dự kiến dùng bao nhiêu dầu ngày mai?", notForecast)
	askBack(t, "Dự báo giúp em, EX-03 ngày mai dùng bao nhiêu dầu?", notForecast)
	askBack(t, "Nhắc tôi nhé, EX-03 hôm nay được cấp bao nhiêu dầu?", notWarning)
	askBack(t, "Quyết toán dầu EX-03 hôm nay bao nhiêu?", notMoney)
	askBack(t, "Chi phí dầu EX-03 hôm nay bao nhiêu?", notMoney)
	// từ phủ định đứng CUỐI câu là đuôi hỏi, không phải phủ định: câu hỏi tiền kèm "không?" vẫn là câu hỏi tiền
	askBack(t, "Chi phí dầu EX-03 hôm nay có cao không?", notMoney)
	// "không" của cụm xác nhận ("đúng không") không phải từ phủ định dù sau nó còn chữ ("ạ")
	askBack(t, "Chi phí dầu EX-03 hôm nay 4tr đồng đúng không ạ?", notMoney)
}
