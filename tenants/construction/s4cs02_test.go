package construction_test

import "testing"

// S4CS-02 (S1 hồi quy của vòng sửa S4, soát 23/10): luật miễn trừ của S4BS-05 vô hiệu hoá cụm báo nghĩa CÓ trong danh sách của pack chỉ vì
// trong cùng mệnh đề có từ phủ định / dấu bỏ qua / từ nối điều kiện ở BẤT KỲ đâu, và "mấy" / "định mức" đứng sau cụm ghi / đánh giá làm câu
// thành câu hỏi — câu hỏi dự báo, nhờ cảnh báo, lệnh ghi bị trả số đo thật không nhãn. Bảy câu của người soát phải nói rõ nghĩa chưa hỗ trợ;
// cặp tối thiểu (đổi đúng một chữ kích hoạt) giữ nguyên kết cục.
func TestS4CS02CumBaoNghiaKhongBiMienTruOan(t *testing.T) {
	for q, note := range map[string]string{
		// "thì" đóng đoạn điều kiện: phần sau "thì" là điều đang hỏi
		"Nếu mưa to thì EX-03 dự báo dùng bao nhiêu dầu?": notForecast,
		"Nếu mưa to thì dự báo EX-03 dùng bao nhiêu dầu?": notForecast,
		// từ phủ định thuộc đoạn điều kiện đứng SAU cụm, không phủ nhận cụm
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
	// từ phủ định đứng trước cụm mà tên thực thể chen giữa thì thuộc vị ngữ khác
	askBack(t, "Không biết EX-03 hôm nay dự báo dùng bao nhiêu dầu?", notForecast)
	// từ phủ định / dấu bỏ qua đứng sau cụm mà phần sau có chữ của từ khoá chỉ số, cụm hỏi hay con số
	askBack(t, "EX-03 hôm nay dự báo không dùng hết dầu", notForecast)
	askBack(t, "EX-03 dự báo không cần chính xác bao nhiêu dầu hôm nay?", notForecast)
	askBack(t, "EX-03 hôm nay dự báo không quá 4000 L dầu", notForecast)
	// mệnh đề nêu sẵn giá trị: cụm xác nhận vẫn là câu hỏi; mệnh đề nêu tên nguồn là lời của nguồn, cụm hỏi hỏi chỉ số khác
	answered(t, "Ghi nhận EX-03 hôm nay dùng 3782 L dầu đúng không?")
	answered(t, "Phiếu kho ghi nhận 3850 L dầu cho EX-03 hôm nay thì cảm biến đo bao nhiêu?")
}
