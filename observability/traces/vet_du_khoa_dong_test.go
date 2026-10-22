package traces_test

import (
	"strings"
	"testing"
)

// R2-03 vòng 3 (soát vòng 2, N2): khoá NGUYÊN dòng — so cả danh sách, từng chữ — trên lượt THẬT cho các chỗ mà đột biến của người soát vòng 2 làm vết
// nói SAI mà test vẫn xanh (X6, X21, X24, X26–X30, X39, X40, X42). Mỗi dòng ở đây là thứ người đọc vết thấy; đổi một chiều, một route, một vế so sánh,
// một ý định hay "có" thành "thiếu" đều làm test đỏ. Dòng chung của mọi lượt (thang bậc kiểm đi hết thang) được khoá cùng ở lượt gốc.

// Cổng phán quyết (S9): MỖI chiều chưa giải được có đúng lời giải nghĩa của chiều đó (không dùng lời của chiều khác), đúng ba đầu vào, đúng hai phán quyết bị loại.
func TestVetDuKhoaDongCongPhanQuyetTungChieu(t *testing.T) {
	{ // chiều "accent"
		w := whyOfTurn(t, realAsk(t), person("director", ""), "EX-03 hom nay duoc cap 3850 L dung khong?")
		wantList(t, "S9 lý do (accent)", w.Validate.Reasons,
			"cổng phán quyết: không phán quyết — chiều \"accent\" chưa giải được: câu không dấu: bộ chuẩn hoá khớp bỏ dấu nên nhiều chữ trùng nhau",
			"đầu vào của cổng: chủ thể EX-03 · kỳ đọc được 1 · cụm xác nhận 1 · câu có dấu false · nguồn đã tạo nên con số warehouse_slip · con số xác nhận “3.850” L",
			"chỉ trả số kèm câu giữ phán quyết (confirm_withheld)")
		wantList(t, "S9 loại (accent)", w.Validate.Rejected,
			"phán quyết \"Đúng.\" (confirm_yes): bị loại — chiều \"accent\" chưa giải",
			"phán quyết \"Không.\" (confirm_no): bị loại — chiều \"accent\" chưa giải")
	}
	{ // chiều "period"
		w := whyOfTurn(t, realAsk(t), person("director", ""), "EX-03 được cấp 3850 L đúng không?")
		wantList(t, "S9 lý do (period)", w.Validate.Reasons,
			"cổng phán quyết: không phán quyết — chiều \"period\" chưa giải được: mốc thời gian: câu không nêu mốc (kỳ mặc định của pack) hoặc nêu nhiều mốc",
			"đầu vào của cổng: chủ thể EX-03 · kỳ đọc được 0 · cụm xác nhận 1 · câu có dấu true · nguồn đã tạo nên con số warehouse_slip · con số xác nhận “3.850” L",
			"chỉ trả số kèm câu giữ phán quyết (confirm_withheld)")
		wantList(t, "S9 loại (period)", w.Validate.Rejected,
			"phán quyết \"Đúng.\" (confirm_yes): bị loại — chiều \"period\" chưa giải",
			"phán quyết \"Không.\" (confirm_no): bị loại — chiều \"period\" chưa giải")
	}
	{ // chiều "marker"
		w := whyOfTurn(t, realAsk(t), person("director", ""), "EX-03 hôm nay được cấp 3850 L đúng không, phải không?")
		wantList(t, "S9 lý do (marker)", w.Validate.Reasons,
			"cổng phán quyết: không phán quyết — chiều \"marker\" chưa giải được: nhiều cụm xác nhận là nhiều claim, một phán quyết không trả lời đủ",
			"đầu vào của cổng: chủ thể EX-03 · kỳ đọc được 1 · cụm xác nhận 2 · câu có dấu true · nguồn đã tạo nên con số warehouse_slip · con số xác nhận “3.850” L",
			"chỉ trả số kèm câu giữ phán quyết (confirm_withheld)")
		wantList(t, "S9 loại (marker)", w.Validate.Rejected,
			"phán quyết \"Đúng.\" (confirm_yes): bị loại — chiều \"marker\" chưa giải",
			"phán quyết \"Không.\" (confirm_no): bị loại — chiều \"marker\" chưa giải")
	}
	{ // chiều "value"
		w := whyOfTurn(t, realAsk(t), person("director", ""), "EX-03 hôm nay được cấp 3850 đúng không?")
		wantList(t, "S9 lý do (value)", w.Validate.Reasons,
			"cổng phán quyết: không phán quyết — chiều \"value\" chưa giải được: con số xác nhận chưa đọc chắc: có số khác, thiếu đơn vị, ngoài miền của chỉ số, đọc bằng chữ mơ hồ hay nhắc lại sau mệnh đề khác",
			"đầu vào của cổng: chủ thể EX-03 · kỳ đọc được 1 · cụm xác nhận 1 · câu có dấu true · nguồn đã tạo nên con số warehouse_slip · con số xác nhận “3.850” L",
			"chỉ trả số kèm câu giữ phán quyết (confirm_withheld)")
		wantList(t, "S9 loại (value)", w.Validate.Rejected,
			"phán quyết \"Đúng.\" (confirm_yes): bị loại — chiều \"value\" chưa giải",
			"phán quyết \"Không.\" (confirm_no): bị loại — chiều \"value\" chưa giải")
	}
	{ // chiều "residual"
		w := whyOfTurn(t, realAsk(t), person("director", ""), "EX-03 hôm nay được cấp 3850 L zzz đúng không?")
		wantList(t, "S9 lý do (residual)", w.Validate.Reasons,
			"cổng phán quyết: không phán quyết — chiều \"residual\" chưa giải được: phần chữ hay ký hiệu chưa được giải thích (từ lạ, '~', '>', '%', '/')",
			"đầu vào của cổng: chủ thể EX-03 · kỳ đọc được 1 · cụm xác nhận 1 · câu có dấu true · nguồn đã tạo nên con số warehouse_slip · con số xác nhận “3.850” L",
			"chỉ trả số kèm câu giữ phán quyết (confirm_withheld)")
		wantList(t, "S9 loại (residual)", w.Validate.Rejected,
			"phán quyết \"Đúng.\" (confirm_yes): bị loại — chiều \"residual\" chưa giải",
			"phán quyết \"Không.\" (confirm_no): bị loại — chiều \"residual\" chưa giải")
	}
	{ // chiều "subject"
		w := whyOfTurn(t, realAsk(t), person("director", ""), "Dự án A hôm nay được cấp 3850 L, EX-03 đúng không?")
		wantList(t, "S9 lý do (subject)", w.Validate.Reasons,
			"cổng phán quyết: không phán quyết — chiều \"subject\" chưa giải được: chủ thể của con số chưa giải: có tên khác chủ thể, bí danh mơ hồ chưa có bằng chứng dương, hay con số chưa gắn vào chủ thể",
			"đầu vào của cổng: chủ thể PRJ-A · kỳ đọc được 1 · cụm xác nhận 1 · câu có dấu true · nguồn đã tạo nên con số warehouse_slip · con số xác nhận “3.850” L",
			"chỉ trả số kèm câu giữ phán quyết (confirm_withheld)")
		wantList(t, "S9 loại (subject)", w.Validate.Rejected,
			"phán quyết \"Đúng.\" (confirm_yes): bị loại — chiều \"subject\" chưa giải",
			"phán quyết \"Không.\" (confirm_no): bị loại — chiều \"subject\" chưa giải")
	}
	{ // chiều "source"
		w := whyOfTurn(t, realAsk(t), person("director", ""), "EX-03 hôm nay được cấp 3850 L theo ERP đúng không?")
		wantList(t, "S9 lý do (source)", w.Validate.Reasons,
			"cổng phán quyết: không phán quyết — chiều \"source\" chưa giải được: nguồn: tên nguồn trong câu không khớp nguồn đã tạo nên con số trả lời, hay có cụm nguồn khác ở ngữ cảnh",
			"đầu vào của cổng: chủ thể EX-03 · kỳ đọc được 1 · cụm xác nhận 1 · câu có dấu true · nguồn đã tạo nên con số warehouse_slip · con số xác nhận “3.850” L",
			"chỉ trả số kèm câu giữ phán quyết (confirm_withheld)")
		wantList(t, "S9 loại (source)", w.Validate.Rejected,
			"phán quyết \"Đúng.\" (confirm_yes): bị loại — chiều \"source\" chưa giải",
			"phán quyết \"Không.\" (confirm_no): bị loại — chiều \"source\" chưa giải")
	}
}

// Luồng hỏi ghi đè route (nhiều con số khác nhau): dòng ĐẦU của S5 là lý do của ROUTER (không phải lý do của kế hoạch đã bị ghi đè); route khai của ý định
// không bị kể là "không đi" khi router đã đi đúng route khai; route của router nằm ở phương án bị loại kèm lý do ghi đè.
func TestVetDuKhoaDongLuongHoiGhiDeRoute(t *testing.T) {
	w := whyOfTurn(t, realAsk(t), person("director", ""), "EX-03 hôm nay được cấp 3850 L hay 4000 L đúng không?")
	wantList(t, "S5 lý do (ghi đè)", w.Route.Reasons,
		"ý định fuel_issued khớp luật",
		"đã kiểm theo thứ tự: 1) nhiều ý định cùng khớp: qua · "+
			"2) thiếu thuật ngữ bắt buộc: qua · "+
			"3) hỏi lại nghĩa số liệu: qua · "+
			"4) không ý định nào khớp: qua · "+
			"5) capability của ý định: qua · 6) phạm vi dữ liệu: qua · 7) thiếu khe bắt buộc: qua · 8) route khai của ý định — fuel_issued → SQL_QUERY → chọn SQL_QUERY · 9) câu so sánh hai nghĩa: qua · 10) ý định tri thức khớp kèm: qua",
		"luồng hỏi ghi đè route SQL_QUERY thành CLARIFICATION: câu xác nhận nêu nhiều con số khác nhau")
	wantList(t, "S5 loại (ghi đè)", w.Route.Rejected,
		"CLARIFICATION (nhiều ý định cùng khớp): không áp dụng — không có ý định nào hoà nhau",
		"CLARIFICATION (thiếu thuật ngữ bắt buộc): không áp dụng — không có ý định nào khớp từ khoá mà thiếu thuật ngữ",
		"CLARIFICATION (hỏi lại nghĩa số liệu): không áp dụng — không rơi vào trường hợp có thực thể và thuật ngữ mà không từ khoá nào khớp",
		"REJECT (không ý định nào khớp): không áp dụng — đã có ý định fuel_issued",
		"REJECT (capability của ý định): không áp dụng — metrics.read: có (vai director)",
		"REJECT (phạm vi dữ liệu): không áp dụng — EX-03: trong phạm vi của người hỏi",
		"CLARIFICATION (thiếu khe bắt buộc): không áp dụng — mọi khe bắt buộc đã có",
		"CALCULATION (câu so sánh hai nghĩa): không áp dụng — không có vế so sánh",
		"HYBRID_RETRIEVAL (ý định tri thức khớp kèm): không áp dụng — không có ý định tri thức khớp kèm",
		"SQL_QUERY (route của router): bị ghi đè — câu xác nhận nêu nhiều con số khác nhau")
}

// Câu hỏi nghĩa số liệu (có thực thể và thuật ngữ, không từ khoá nào khớp): bậc "phạm vi dữ liệu (câu hỏi nghĩa số liệu)" nói đúng ngoài / trong phạm vi.
func TestVetDuKhoaDongBacPhamViCauHoiNghiaSoLieu(t *testing.T) {
	no := whyOfTurn(t, realAsk(t), person("site_engineer", "PRJ-A"), "DZ-03 dầu?")
	wantList(t, "S5 lý do (ngoài phạm vi)", no.Route.Reasons,
		"thực thể DZ-03 ngoài phạm vi dữ liệu của người hỏi",
		"đã kiểm theo thứ tự: 1) nhiều ý định cùng khớp: qua · "+
			"2) thiếu thuật ngữ bắt buộc: qua · "+
			"3) hỏi lại nghĩa số liệu — fuel_variance, fuel_issued, fuel_reported, fuel_consumed, fuel_billed → chọn CLARIFICATION · "+
			"4) phạm vi dữ liệu (câu hỏi nghĩa số liệu) — DZ-03: ngoài phạm vi → chọn REJECT")
	wantList(t, "S5 loại (ngoài phạm vi)", no.Route.Rejected,
		"CLARIFICATION (nhiều ý định cùng khớp): không áp dụng — không có ý định nào hoà nhau",
		"CLARIFICATION (thiếu thuật ngữ bắt buộc): không áp dụng — không có ý định nào khớp từ khoá mà thiếu thuật ngữ")
	yes := whyOfTurn(t, realAsk(t), person("director", ""), "EX-03 dầu?")
	wantList(t, "S5 lý do (trong phạm vi)", yes.Route.Reasons,
		"có thực thể và thuật ngữ nhưng không từ khoá ý định nào khớp: fuel_variance, fuel_issued, fuel_reported, fuel_consumed, fuel_billed",
		"đã kiểm theo thứ tự: 1) nhiều ý định cùng khớp: qua · "+
			"2) thiếu thuật ngữ bắt buộc: qua · "+
			"3) hỏi lại nghĩa số liệu — fuel_variance, fuel_issued, fuel_reported, fuel_consumed, fuel_billed → chọn CLARIFICATION · "+
			"4) phạm vi dữ liệu (câu hỏi nghĩa số liệu): qua")
	wantList(t, "S5 loại (trong phạm vi)", yes.Route.Rejected,
		"CLARIFICATION (nhiều ý định cùng khớp): không áp dụng — không có ý định nào hoà nhau",
		"CLARIFICATION (thiếu thuật ngữ bắt buộc): không áp dụng — không có ý định nào khớp từ khoá mà thiếu thuật ngữ",
		"REJECT (phạm vi dữ liệu (câu hỏi nghĩa số liệu)): không áp dụng — EX-03: trong phạm vi")
}

// HYBRID: bậc "ý định tri thức khớp kèm" nêu TÊN ý định tri thức và chọn HYBRID_RETRIEVAL; S4 kể cả ba cụm từ khoá của ý định tri thức và việc nó khớp cùng.
func TestVetDuKhoaDongHybrid(t *testing.T) {
	w := whyOfTurn(t, realAsk(t), person("director", ""), "EX-03 hôm nay được cấp bao nhiêu dầu, và theo quy trình thì khi lệch phải làm gì?")
	wantList(t, "S4 lý do (HYBRID)", w.Intent.Reasons,
		"từ khoá khớp (luật khớp dài nhất): fuel_issued [“được cấp”] · fuel_generic [“bao nhiêu”] · sop_question [“quy trình”, “phải làm gì”, “khi lệch”]",
		"khớp đủ: fuel_issued (ưu tiên 20, từ khoá “được cấp”, câu có thuật ngữ diesel_fuel)",
		"khớp đủ: fuel_generic (ưu tiên 10, từ khoá “bao nhiêu”, câu có thuật ngữ diesel_fuel)",
		"khớp đủ: sop_question (ưu tiên 25, từ khoá “quy trình”, “phải làm gì”, “khi lệch”, không đòi thuật ngữ)",
		"chọn fuel_issued: ưu tiên cao nhất (20)",
		"khe subject ← EX-03 (thực thể rõ đầu tiên đúng loại)",
		"khe period ← “2026-10-19” (mốc “hôm nay” trong câu)",
		"chỉ số issued_fuel (theo thuật ngữ diesel_fuel của ý định fuel_issued)",
		"luật intent:rules",
		"ý định tri thức sop_question khớp cùng ý định số liệu — router ghép HYBRID nếu đủ khe")
	wantList(t, "S4 loại (HYBRID)", w.Intent.Rejected,
		"fuel_generic: khớp đủ nhưng bị loại — ưu tiên 10 thấp hơn fuel_issued (20)")
	wantList(t, "S5 lý do (HYBRID)", w.Route.Reasons,
		"ý định fuel_issued khớp luật + tri thức sop_question",
		"đã kiểm theo thứ tự: 1) nhiều ý định cùng khớp: qua · "+
			"2) thiếu thuật ngữ bắt buộc: qua · "+
			"3) hỏi lại nghĩa số liệu: qua · "+
			"4) không ý định nào khớp: qua · "+
			"5) capability của ý định: qua · "+
			"6) capability của phần tri thức: qua · "+
			"7) phạm vi dữ liệu: qua · "+
			"8) thiếu khe bắt buộc: qua · "+
			"9) route khai của ý định — fuel_issued → SQL_QUERY → chọn SQL_QUERY · "+
			"10) câu so sánh hai nghĩa: qua · "+
			"11) ý định tri thức khớp kèm — sop_question → chọn HYBRID_RETRIEVAL")
	wantList(t, "S5 loại (HYBRID)", w.Route.Rejected,
		"CLARIFICATION (nhiều ý định cùng khớp): không áp dụng — không có ý định nào hoà nhau",
		"CLARIFICATION (thiếu thuật ngữ bắt buộc): không áp dụng — không có ý định nào khớp từ khoá mà thiếu thuật ngữ",
		"CLARIFICATION (hỏi lại nghĩa số liệu): không áp dụng — không rơi vào trường hợp có thực thể và thuật ngữ mà không từ khoá nào khớp",
		"REJECT (không ý định nào khớp): không áp dụng — đã có ý định fuel_issued",
		"REJECT (capability của ý định): không áp dụng — metrics.read: có (vai director)",
		"REJECT (capability của phần tri thức): không áp dụng — documents.read: có",
		"REJECT (phạm vi dữ liệu): không áp dụng — EX-03: trong phạm vi của người hỏi",
		"CLARIFICATION (thiếu khe bắt buộc): không áp dụng — mọi khe bắt buộc đã có",
		"CALCULATION (câu so sánh hai nghĩa): không áp dụng — không có vế so sánh",
		"SQL_QUERY (route khai của ý định fuel_issued): không đi — ý định fuel_issued khớp luật + tri thức sop_question")
}

// Câu so sánh hai nghĩa: S4 nói hai vế theo thứ tự nhắc; S5 có bậc "capability của vế so sánh" (có) và bậc "câu so sánh hai nghĩa" nêu tên vế SAU; câu so sánh kèm
// cụm xác nhận thì cổng phán quyết nói đúng "câu so sánh hai nghĩa số liệu" (không phán quyết, không có phương án bị loại).
func TestVetDuKhoaDongCauSoSanh(t *testing.T) {
	w := whyOfTurn(t, realAsk(t), person("director", ""), "EX-03 hôm nay đội báo nhiều hơn kho xuất bao nhiêu?")
	wantList(t, "S4 lý do (so sánh)", w.Intent.Reasons,
		"từ khoá khớp (luật khớp dài nhất): fuel_issued [“kho xuất”] · fuel_reported [“đội báo”] · fuel_generic [“bao nhiêu”]",
		"thuật ngữ hiểu ngầm diesel_fuel (câu nhắc thực thể mà không nói vật tư và không có đại lượng đo được nào khác — chính sách của pack)",
		"khớp đủ: fuel_issued (ưu tiên 20, từ khoá “kho xuất”, thuật ngữ hiểu ngầm diesel_fuel)",
		"khớp đủ: fuel_reported (ưu tiên 20, từ khoá “đội báo”, thuật ngữ hiểu ngầm diesel_fuel)",
		"khớp đủ: fuel_generic (ưu tiên 10, từ khoá “bao nhiêu”, thuật ngữ hiểu ngầm diesel_fuel)",
		"khe subject ← EX-03 (thực thể rõ đầu tiên đúng loại)",
		"khe period ← “2026-10-19” (mốc “hôm nay” trong câu)",
		"chỉ số reported_fuel (theo thuật ngữ diesel_fuel (hiểu ngầm) của ý định fuel_reported)",
		"luật intent:rules+default_term:diesel_fuel",
		"câu so sánh: fuel_reported so với fuel_issued (hai nghĩa số liệu ngang nhau kèm cụm so sánh của pack) — phép tính hiệu theo thứ tự nhắc, không phải câu mơ hồ")
	wantList(t, "S4 loại (so sánh)", w.Intent.Rejected,
		"fuel_generic: khớp đủ nhưng bị loại — ưu tiên 10 thấp hơn fuel_issued (20)")
	wantList(t, "S5 lý do (so sánh)", w.Route.Reasons,
		"ý định fuel_reported khớp luật, so với fuel_issued",
		"đã kiểm theo thứ tự: 1) nhiều ý định cùng khớp: qua · "+
			"2) thiếu thuật ngữ bắt buộc: qua · "+
			"3) hỏi lại nghĩa số liệu: qua · "+
			"4) không ý định nào khớp: qua · "+
			"5) capability của ý định: qua · "+
			"6) capability của vế so sánh: qua · "+
			"7) phạm vi dữ liệu: qua · "+
			"8) thiếu khe bắt buộc: qua · "+
			"9) route khai của ý định — fuel_reported → SQL_QUERY → chọn SQL_QUERY · "+
			"10) câu so sánh hai nghĩa — fuel_issued → chọn CALCULATION · "+
			"11) ý định tri thức khớp kèm: qua")
	wantList(t, "S5 loại (so sánh)", w.Route.Rejected,
		"CLARIFICATION (nhiều ý định cùng khớp): không áp dụng — không có ý định nào hoà nhau",
		"CLARIFICATION (thiếu thuật ngữ bắt buộc): không áp dụng — không có ý định nào khớp từ khoá mà thiếu thuật ngữ",
		"CLARIFICATION (hỏi lại nghĩa số liệu): không áp dụng — không rơi vào trường hợp có thực thể và thuật ngữ mà không từ khoá nào khớp",
		"REJECT (không ý định nào khớp): không áp dụng — đã có ý định fuel_reported",
		"REJECT (capability của ý định): không áp dụng — metrics.read: có (vai director)",
		"REJECT (capability của vế so sánh): không áp dụng — metrics.read: có",
		"REJECT (phạm vi dữ liệu): không áp dụng — EX-03: trong phạm vi của người hỏi",
		"CLARIFICATION (thiếu khe bắt buộc): không áp dụng — mọi khe bắt buộc đã có",
		"HYBRID_RETRIEVAL (ý định tri thức khớp kèm): không áp dụng — không có ý định tri thức khớp kèm",
		"SQL_QUERY (route khai của ý định fuel_reported): không đi — ý định fuel_reported khớp luật, so với fuel_issued")
	c := whyOfTurn(t, realAsk(t), person("director", ""), "EX-03 hôm nay đội báo nhiều hơn kho xuất 150 L đúng không?")
	wantList(t, "S9 lý do (so sánh + xác nhận)", c.Validate.Reasons,
		"cổng phán quyết: không áp dụng — câu so sánh hai nghĩa số liệu, nên không có một con số để xác nhận")
	if len(c.Validate.Rejected) != 0 {
		t.Errorf("S9 loại (so sánh + xác nhận): muốn danh sách rỗng, được %q", c.Validate.Rejected)
	}
}

// Điều kiện giữ chữ của câu dùng câu ĐÃ GẤP KÝ TỰ (in.Raw), không dùng câu gốc: bí mật viết bằng ký tự toàn chiều rộng ("ｐａｓｓｗｏｒｄ＝…") mà traces.Sanitize không nhận ra
// ở câu gốc vẫn bị nhận ra sau khi gấp, nên mọi đoạn chữ của câu bị che (và giá trị không còn ở bất kỳ dòng giải thích nào).
func TestVetDuKhoaDongBiMatToanChieuRong(t *testing.T) {
	w := whyOfTurn(t, realAsk(t), person("director", ""), "EX-03 hôm nay được cấp ｐａｓｓｗｏｒｄ＝zq7771secret 3850 L đúng không?")
	wantList(t, "S2 lý do (toàn chiều rộng)", w.Normalize.Reasons,
		"code “đã che” ← code:equipment",
		"date “đã che” ← date:relative",
		"quantity “đã che” ← number:digits+unit")
	wantList(t, "S9 lý do (toàn chiều rộng)", w.Validate.Reasons,
		"cổng phán quyết: không phán quyết — chiều \"residual\" chưa giải được: phần chữ hay ký hiệu chưa được giải thích (từ lạ, '~', '>', '%', '/')",
		"đầu vào của cổng: chủ thể EX-03 · kỳ đọc được 1 · cụm xác nhận 1 · câu có dấu true · nguồn đã tạo nên con số warehouse_slip · con số xác nhận “đã che” L",
		"chỉ trả số kèm câu giữ phán quyết (confirm_withheld)")
	for _, l := range whyLines(w) {
		if strings.Contains(squash(l), "zq7771secret") {
			t.Errorf("giá trị bí mật toàn chiều rộng còn trong giải thích: %q", l)
		}
	}
}

// Cụm mơ hồ (bí danh nhiều thực thể) được S4 nhắc lại bằng chữ NGUYÊN VĂN trong cặp “…”: ở dòng khe thiếu (kèm lời nhắc router có thể gỡ ở S5), ở nguồn của từng
// ứng viên hỏi lại, và ở nhánh hỏi lại nghĩa số liệu. Chữ nào của người hỏi không qua cặp “…” thì Why.WithholdQuoted không che được khi câu mang bí mật.
func TestVetDuKhoaDongCumMoHoOS4(t *testing.T) {
	dir := person("director", "")
	a := whyOfTurn(t, realAsk(t), dir, "Máy 03 hôm nay được cấp bao nhiêu dầu?")
	wantLine(t, "S4 khe thiếu kèm cụm mơ hồ", a.Intent.Reasons,
		"khe subject thiếu: không có thực thể rõ đúng loại (equipment|project) — có cụm mơ hồ “may 3” (router có thể gỡ ở S5 bằng dự án đã nêu hay phạm vi dữ liệu của người hỏi)")
	wantLine(t, "S4 nguồn DZ-03", a.Intent.Reasons, "ứng viên hỏi lại DZ-03 ← S3: ứng viên của cụm mơ hồ “may 3”")
	wantLine(t, "S4 nguồn EX-03", a.Intent.Reasons, "ứng viên hỏi lại EX-03 ← S3: ứng viên của cụm mơ hồ “may 3”")

	b := whyOfTurn(t, realAsk(t), dir, "Máy 03 dầu?")
	wantList(t, "S4 (hỏi lại nghĩa số liệu, bí danh mơ hồ)", b.Intent.Reasons,
		"không từ khoá ý định nào khớp (luật khớp dài nhất)",
		"có thực thể và thuật ngữ nhưng không từ khoá ý định nào khớp, và câu không trả lời được theo ý định chung (câu không có cụm xác nhận) — hỏi lại nghĩa số liệu: fuel_variance, fuel_issued, fuel_reported, fuel_consumed, fuel_billed",
		"ứng viên hỏi lại DZ-03 ← S3: ứng viên của cụm mơ hồ “may 3”",
		"ứng viên hỏi lại EX-03 ← S3: ứng viên của cụm mơ hồ “may 3”")
	// câu mang bí mật: cụm mơ hồ cũng bị che
	s := whyOfTurn(t, realAsk(t), dir, "Máy 03 password=zq7771secret dầu?")
	for _, l := range whyLines(s) {
		if strings.Contains(l, "may 3") || strings.Contains(l, "Máy 03") || strings.Contains(l, "zq7771secret") {
			t.Errorf("câu mang bí mật mà dòng còn chữ của câu: %q", l)
		}
	}
}

// Các dòng giải thích còn lại của lượt thật mà trước đây không test nào khoá: truy xuất không trả đoạn nào (người hỏi không có quyền đọc tài liệu), câu nói về đại
// lượng đo được khác (ý định số liệu của dầu không hỏi lại thuật ngữ), chỉ số dẫn xuất của ý định tính hiệu, và khe thiếu vì câu nêu nhiều thực thể khác nhau.
func TestVetDuKhoaDongDongGiaiThichConLai(t *testing.T) {
	a := realAsk(t)
	eng, dir := person("site_engineer", "PRJ-A"), person("director", "")

	r := whyOfTurn(t, a, eng, "Quy trình xử lý chênh lệch nhiên liệu là gì?")
	wantList(t, "S7 truy xuất 0 đoạn", r.Context.Reasons,
		"truy xuất trả 0 đoạn người hỏi được đọc (không có đoạn khớp, hoặc người hỏi không có quyền đọc đoạn nào — chặng truy xuất không phân biệt hai trường hợp)",
		"quyền: ACL tài liệu do chặng truy xuất kiểm, chỉ đoạn người hỏi được đọc được trả về (số đoạn bị loại vì quyền không xuất khỏi chặng truy xuất)")

	o := whyOfTurn(t, a, dir, "EX-03 hôm nay được cấp bao nhiêu nhân công?")
	wantList(t, "S4 loại (đại lượng đo được khác)", o.Intent.Rejected,
		"fuel_issued: từ khoá “được cấp” khớp nhưng câu nói về đại lượng đo được khác nên không hỏi lại thuật ngữ diesel_fuel",
		"fuel_generic: ý định yếu — chỉ có từ khoá chung chung (“bao nhiêu”), thiếu thuật ngữ diesel_fuel nên một mình không đủ")

	d := whyOfTurn(t, a, dir, "EX-03 hôm nay chênh lệch bao nhiêu?")
	wantLine(t, "S4 chỉ số dẫn xuất", d.Intent.Reasons, "chỉ số dẫn xuất issue_vs_measured (ý định fuel_variance khai)")

	c := whyOfTurn(t, a, eng, "RL-02 và máy 03 hôm nay được cấp bao nhiêu dầu?")
	wantLine(t, "S4 khe thiếu vì nhiều thực thể", c.Intent.Reasons,
		"khe subject thiếu: câu nêu 3 thực thể khác nhau cùng loại (RL-02, DZ-03, EX-03) cho khe chỉ nhận một — không lấy thực thể nhắc đầu")
}
