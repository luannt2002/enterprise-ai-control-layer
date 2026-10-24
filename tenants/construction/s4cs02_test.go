package construction_test

import (
	"strings"
	"testing"

	"github.com/luannt2002/enterprise-ai-control-layer/core/contracts"
)

// S4CS-02 (S1 hồi quy của vòng sửa S4, soát 23/10): luật miễn trừ của S4BS-05 vô hiệu hoá cụm báo nghĩa CÓ trong danh sách của pack chỉ vì
// trong cùng mệnh đề có từ phủ định / dấu bỏ qua / từ nối điều kiện ở BẤT KỲ đâu, và "mấy" / "định mức" đứng sau cụm ghi / đánh giá làm câu
// thành câu hỏi — câu hỏi dự báo, nhờ cảnh báo, lệnh ghi bị trả số đo thật không nhãn. Bảy câu của người soát phải nói rõ nghĩa chưa hỗ trợ;
// cặp tối thiểu (đổi đúng một chữ kích hoạt) giữ nguyên kết cục.
func TestS4CS02CumBaoNghiaKhongBiMienTruOan(t *testing.T) {
	for q, note := range map[string]string{
		// "thì" (condition_close_markers của pack) đóng vế điều kiện: cụm đứng sau "thì" là điều đang hỏi
		"Nếu mưa to thì EX-03 dự báo dùng bao nhiêu dầu?": notForecast,
		"Nếu mưa to thì dự báo EX-03 dùng bao nhiêu dầu?": notForecast,
		// từ phủ định đứng SAU cụm mà phần sau có nội dung số liệu (vế "nếu EX-03 chưa được cấp dầu") không phủ nhận cụm
		"Cảnh báo giúp em nếu EX-03 chưa được cấp dầu.": notWarning,
		// từ phủ định / dấu bỏ qua đứng sau cụm và mở vị ngữ khác ("không dùng hết dầu", "không cần chính xác")
		"Dự báo EX-03 không dùng hết dầu hôm nay phải không":                 notForecast,
		"Dự báo nhanh không cần chính xác EX-03 hôm nay dùng bao nhiêu dầu?": notForecast,
		// "mấy" / "định mức" đứng sau cụm ghi / đánh giá không biến lệnh thành câu hỏi
		"Ghi nhận EX-03 hôm nay dùng 3782 L dầu cho ca mấy.":       notWrite,
		"Nhận xét định mức tiêu thụ 3782 L dầu hôm nay của EX-03.": notEval,
	} {
		askBack(t, q, note)
	}
	// cặp tối thiểu của người soát: kết cục như trước khi sửa
	for q, note := range map[string]string{
		"Cảnh báo giúp em nếu EX-03 được cấp dầu.":                     notWarning,
		"Dự báo nhanh cần chính xác EX-03 hôm nay dùng bao nhiêu dầu?": notForecast,
		"Nếu mưa to, dự báo EX-03 dùng bao nhiêu dầu?":                 notForecast,
		"Nhận xét mức tiêu thụ 3782 L dầu hôm nay của EX-03.":          notEval,
		"Ghi nhận EX-03 hôm nay dùng 3782 L dầu cho ca sáng.":          notWrite,
	} {
		askBack(t, q, note)
	}
}

// Ghim từng luật vị trí của S4BS-05 sau khi sửa S4CS-02 (S4CS-07: X3, X4 từng sống sót) — mỗi câu chỉ một luật quyết định.
func TestS4CS02LuatViTriPhuDinhDieuKien(t *testing.T) {
	// X3: từ phủ định GIỮA câu không phải đuôi hỏi có / không — câu nêu giá trị vẫn là lệnh ghi
	askBack(t, "EX-03 hôm nay không nghỉ, dùng 3782 L dầu", notWrite)
	// X4: từ nối điều kiện GIỮA mệnh đề không biến cả mệnh đề thành giả thiết (cũng là câu (3) của người soát)
	askBack(t, "Cảnh báo giúp em nếu EX-03 được cấp dầu.", notWarning)
	askBack(t, "Cảnh báo giúp em nếu EX-03 chưa được cấp dầu, em cảm ơn.", notWarning) // còn mệnh đề khác: "nếu" giữa mệnh đề vẫn không mở vế điều kiện
	// từ phủ định không đứng LIỀN trước cụm thì thuộc vị ngữ khác
	askBack(t, "Không biết EX-03 hôm nay dự báo dùng bao nhiêu dầu?", notForecast)
	// từ phủ định / dấu bỏ qua đứng sau cụm mà phần sau có chữ của từ khoá chỉ số, cụm hỏi hay con số
	askBack(t, "EX-03 hôm nay dự báo không dùng hết dầu", notForecast)
	askBack(t, "EX-03 dự báo không cần chính xác bao nhiêu dầu hôm nay?", notForecast)
	askBack(t, "EX-03 hôm nay dự báo không quá 4000 L dầu", notForecast)
	askBack(t, "Dự báo nhanh không cần chính xác cho EX-03 hôm nay", notForecast) // chỉ tên thực thể ở phần sau
	// mệnh đề nêu sẵn giá trị: cụm xác nhận vẫn là câu hỏi; mệnh đề nêu tên nguồn là lời của nguồn, cụm hỏi hỏi chỉ số khác (soát L2 vòng 1, F3: kiểm ý định + số,
	// không chỉ "không có nhãn")
	answersIntent(t, "Ghi nhận EX-03 hôm nay dùng 3782 L dầu đúng không?", "fuel_consumed", "3.782")
	answersIntent(t, "ERP ghi nhận 3900 L dầu cho EX-03 hôm nay thì cảm biến đo bao nhiêu?", "fuel_consumed", "3.782")
	hoiLaiKhongNhan(t, "Phiếu kho ghi nhận 3850 L dầu cho EX-03 hôm nay thì cảm biến đo bao nhiêu?")
}

// hoiLaiKhongNhan: câu hỏi lại nghĩa số liệu (hoà ý định) mà KHÔNG nói nghĩa chưa hỗ trợ nào.
func hoiLaiKhongNhan(t *testing.T, q string) {
	t.Helper()
	if r := ask(t, newAsk(t), user("director", ""), q).Response; r.Route != contracts.RouteClarification || strings.Contains(r.Answer, "Trợ lý chưa") {
		t.Errorf("%q: muốn hỏi lại nghĩa số liệu không kèm nhãn nghĩa chưa hỗ trợ, được [%s/%s] %q", q, r.Route, r.Outcome, r.Answer)
	}
}

// S4CS-02 vòng 2 (soát độc lập L2, F1b): vế điều kiện đóng ở "thì" (condition_close_markers của pack) — cụm sau "thì" là điều đang hỏi / nhờ làm; không có "thì" thì
// vế kết quả là một mệnh đề khác, câu chỉ một mệnh đề thì vế kết quả nằm ngay trong nó.
func TestS4CS02Vong2VeDieuKienDongOThi(t *testing.T) {
	for q, note := range map[string]string{
		"Nếu được thì ghi nhận giúp em EX-03 hôm qua dùng 3782 L dầu nhé?": notWrite,
		"Nếu tiện thì cập nhật EX-03 hôm qua dùng 3782 L dầu giúp em nhé?": notWrite,
		"Nếu mưa to thì dự báo giúp em dầu EX-03 dùng hôm nay.":            notForecast,
		"Nếu mai mưa thì dự báo EX-03 dùng hết dầu không?":                 notForecast,
		"Nếu EX-03 dùng quá 4000 L dầu thì cảnh báo em nhé?":               notWarning,
		"Hễ EX-03 dùng hết dầu thì nhắc tôi nhé.":                          notWarning,
		// vế kết quả kéo sang mệnh đề sau, cụm vẫn sau "thì"
		"Nếu mưa to thì dự báo giúp em, EX-03 hôm nay dùng bao nhiêu dầu?": notForecast,
		// không "thì", câu chỉ một mệnh đề
		"Nếu mưa to dự báo EX-03 hôm nay dùng bao nhiêu dầu?": notForecast,
	} {
		askBack(t, q, note)
	}
	// cụm TRƯỚC "thì" nằm trong vế điều kiện; vế điều kiện không "thì" là một mệnh đề riêng (trước hay sau câu hỏi): giả thiết, câu hỏi số liệu như thường
	answersIntent(t, "Nếu dự báo mưa thì EX-03 hôm nay dùng bao nhiêu dầu?", "fuel_consumed", "3.782")
	answersIntent(t, "EX-03 hôm nay dùng bao nhiêu dầu, nếu dự báo mưa?", "fuel_consumed", "3.782")
}

// S4CS-02 vòng 2 (F1a): từ phủ định / dấu bỏ qua đứng TRƯỚC cụm chỉ phủ nhận cụm khi đứng LIỀN trước ("Không tính tiền"); chữ chen giữa ("Không biết dự báo …",
// "Em không rõ dự báo …") nghĩa là nó thuộc vị ngữ khác.
func TestS4CS02Vong2PhuDinhLienTruoc(t *testing.T) {
	for _, q := range []string{
		"Không biết dự báo EX-03 hôm qua dùng bao nhiêu dầu?",
		"Em không rõ dự báo dầu EX-03 hôm qua là bao nhiêu?",
		"Không biết dự báo EX-03 hôm nay dùng bao nhiêu dầu?",
	} {
		askBack(t, q, notForecast)
	}
}

// S4CS-02 vòng 2 (F2, hồi quy của vòng 1): chủ ngữ đứng trước cụm ghi / đánh giá ("Đồng hồ ghi nhận 3782 L …") là lời của một nguồn — kể cả nguồn pack không biết — nên cụm
// hỏi lượng sau đó vẫn là câu hỏi, trả như trước vòng 1.
func TestS4CS02Vong2ChuNguTruocCumGhi(t *testing.T) {
	answersIntent(t, "Đồng hồ ghi nhận 3782 L dầu cho EX-03 hôm qua thì kho xuất bao nhiêu?", "fuel_issued", "Dầu kho xuất")
	for _, q := range []string{
		"Đồng hồ ghi nhận 3782 L dầu cho EX-03 hôm nay thì kho xuất bao nhiêu?",
		"Sổ nhật ký máy ghi nhận 3782 L dầu cho EX-03 hôm nay thì kho xuất bao nhiêu?",
		"Tài xế ghi lại 3782 L dầu cho EX-03 hôm nay thì kho xuất bao nhiêu?",
	} {
		answersIntent(t, q, "fuel_issued", "3.850")
	}
	hoiLaiKhongNhan(t, "Đoàn kiểm tra kết luận EX-03 hôm nay dùng 3782 L dầu thì kho xuất bao nhiêu?")
}
