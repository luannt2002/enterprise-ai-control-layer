package traces_test

import (
	"regexp"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/luannt2002/enterprise-ai-control-layer/core/contracts"
	"github.com/luannt2002/enterprise-ai-control-layer/core/pipeline"
	"github.com/luannt2002/enterprise-ai-control-layer/intelligence/rag"
	"github.com/luannt2002/enterprise-ai-control-layer/observability/traces"
)

// R2-03 vòng 2 (F2): mỗi dòng giải thích mới có ít nhất một khẳng định CHẶT trên lượt thật — so cả chuỗi, không chỉ "có chứa" một mảnh.
// Nhờ đó đổi một số, một chiều, một nguồn hay bỏ cả dòng đều làm test đỏ (soát vòng 1 tìm ra 15/26 đột biến tay sống sót vì khẳng định lỏng).

func askDocs(t testing.TB, docs ...rag.Document) *pipeline.Ask {
	t.Helper()
	a := realAsk(t)
	a.Deps.Knowledge = rag.NewMemoryIndex("abc-infra", docs)
	return a
}

func sopDoc(id, title, body string) rag.Document {
	return rag.Document{ID: id, Tenant: "abc-infra", Title: title, ACL: []string{"director"}, Body: body}
}

func whyOfTurn(t testing.TB, a *pipeline.Ask, pr contracts.Principal, q string) traces.Why {
	t.Helper()
	return realTurn(t, a, pr, q).Decision.Why
}

// wantList đòi danh sách dòng bằng ĐÚNG danh sách muốn (thứ tự và từng chữ).
func wantList(t testing.TB, label string, got []string, want ...string) {
	t.Helper()
	if !slices.Equal(got, want) {
		t.Errorf("%s:\n  được:\n    %s\n  muốn:\n    %s", label, strings.Join(got, "\n    "), strings.Join(want, "\n    "))
	}
}

// wantLine đòi MỘT dòng có mặt NGUYÊN VĂN trong danh sách.
func wantLine(t testing.TB, label string, got []string, want string) {
	t.Helper()
	if !slices.Contains(got, want) {
		t.Errorf("%s: thiếu dòng nguyên văn %q trong:\n    %s", label, want, strings.Join(got, "\n    "))
	}
}

// Câu nhiều mệnh đề (mệnh đề trước là ngữ cảnh): đủ ba chặng S2, S3, S4 và cổng S9 với từng dòng nguyên văn.
func TestVetDuChatCauNhieuMenhDe(t *testing.T) {
	w := whyOfTurn(t, realAsk(t), person("director", ""), "đội báo 4k, báo cáo ghi 4 m3, vậy EX-03 được cấp 4k dầu đúng không?")
	wantList(t, "S2", w.Normalize.Reasons,
		"number “4k” ← number:digits+multiplier",
		"quantity “4 m3” ← number:digits+unit",
		"code “EX-03” ← code:equipment",
		"quantity “4k” ← number:digits+multiplier+unit_from_term:diesel_fuel (độ tin cậy 0.80)",
		"term “dầu” ← term:exact")
	wantList(t, "S3", w.Resolve.Reasons, "cụm “EX-03” (khoá so khớp “ex 3”) khớp EX-03 (Máy xúc EX-03, loại equipment)")
	wantList(t, "S4", w.Intent.Reasons,
		"từ khoá lấy trong mệnh đề hỏi “vậy EX-03 được cấp 4k dầu đúng không” (mệnh đề chứa cụm xác nhận quyết định ý định, các mệnh đề trước là ngữ cảnh)",
		"từ khoá khớp (luật khớp dài nhất): fuel_issued [“được cấp”]",
		"khớp đủ: fuel_issued (ưu tiên 20, từ khoá “được cấp”, câu có thuật ngữ diesel_fuel)",
		"chọn fuel_issued: ý định duy nhất khớp đủ",
		"khe subject ← EX-03 (thực thể rõ đầu tiên đúng loại)",
		"khe period ← 2026-10-19 (câu không nêu mốc: mặc định today của pack)",
		"chỉ số issued_fuel (theo thuật ngữ diesel_fuel của ý định fuel_issued)",
		"luật intent:rules")
	if len(w.Intent.Rejected) != 0 {
		t.Errorf("S4 không có phương án nào bị loại: %q", w.Intent.Rejected)
	}
	// kỳ đọc được 0: câu không nêu mốc nên cổng giữ phán quyết ở chiều "period" — và đầu vào của cổng nói đúng số đó
	wantList(t, "S9 lý do", w.Validate.Reasons,
		`cổng phán quyết: không phán quyết — chiều "period" chưa giải được: mốc thời gian: câu không nêu mốc (kỳ mặc định của pack) hoặc nêu nhiều mốc`,
		"đầu vào của cổng: chủ thể EX-03 · kỳ đọc được 0 · cụm xác nhận 1 · câu có dấu true · nguồn đã tạo nên con số warehouse_slip · con số xác nhận “4.000” L",
		"chỉ trả số kèm câu giữ phán quyết (confirm_withheld)")
	wantList(t, "S9 loại", w.Validate.Rejected,
		`phán quyết "Đúng." (confirm_yes): bị loại — chiều "period" chưa giải`,
		`phán quyết "Không." (confirm_no): bị loại — chiều "period" chưa giải`)
}

// Xác nhận SAI ("Không."): hai con số trong dòng so sánh đúng chỗ (số của người hỏi trước, số có thẩm quyền sau), S7 đếm đúng từng loại bằng chứng.
func TestVetDuChatXacNhanKhong(t *testing.T) {
	w := whyOfTurn(t, realAsk(t), person("director", ""), "EX-03 hôm nay được cấp 4000 L đúng không?")
	wantList(t, "S9 lý do", w.Validate.Reasons,
		"cổng phán quyết: được phán quyết — mọi chiều của claim (chủ thể, chỉ số, kỳ, giá trị và đơn vị, nguồn) đã giải bằng bằng chứng dương trong câu và mọi chữ của câu được giải thích",
		"đầu vào của cổng: chủ thể EX-03 · kỳ đọc được 1 · cụm xác nhận 1 · câu có dấu true · nguồn đã tạo nên con số warehouse_slip · con số xác nhận “4.000” L",
		`so sánh bằng tuyệt đối số người hỏi “4.000” với số có thẩm quyền 3.850 → "Không." (confirm_no)`)
	wantList(t, "S9 loại", w.Validate.Rejected,
		`phán quyết "Đúng." (confirm_yes): bị loại — kết quả so sánh bằng tuyệt đối chọn confirm_no`,
		"không phán quyết (confirm_withheld): bị loại — cổng không thiếu chiều nào")
	wantList(t, "S7 lý do", w.Context.Reasons,
		"gói ngữ cảnh: 11 bằng chứng (2 dữ liệu nguồn, 1 bằng chứng khác của truy vấn, 8 bằng chứng đi kèm thêm sau truy vấn)",
		"nguồn warehouse_slip (thẩm quyền 1): 1 bằng chứng — tạo nên con số chính",
		"nguồn erp (thẩm quyền 2): 1 bằng chứng — không tạo nên con số chính (nguồn thẩm quyền thấp hơn, hoặc lệch)",
		"calc:issued_fuel: phép tính của con số chính (sum(fact:f03))",
		"question: con số do người hỏi nêu (“4000L”) — nguồn là chính câu hỏi, không phải số liệu",
		"calc:billed_fuel: chỉ số khác cùng kỳ, kê thêm (không phải chỉ số được hỏi)",
		"calc:measured_consumption: chỉ số khác cùng kỳ, kê thêm (không phải chỉ số được hỏi)",
		"calc:reported_fuel: chỉ số khác cùng kỳ, kê thêm (không phải chỉ số được hỏi)",
		"quyền: phạm vi dữ liệu đã kiểm ở S5 cho thực thể gốc EX-03; truy vấn chạy trên 1 thực thể (gốc và con cháu), không lọc bằng chứng theo quyền sau truy vấn")
	wantList(t, "S7 loại", w.Context.Rejected,
		"nguồn erp = 3900 L không được chọn làm con số chính (xung đột issued_fuel|EX-03|2026-10-19, chính sách authority): thẩm quyền thấp hơn nguồn được chọn fact:f03")
}

// Xác nhận ĐÚNG ("Đúng.") trên một dự án: phán quyết bị loại là "Không.", và số thực thể truy vấn là số của dự án cùng con cháu (3), không phải 1.
func TestVetDuChatXacNhanDungTrenDuAn(t *testing.T) {
	w := whyOfTurn(t, realAsk(t), person("director", ""), "Dự án A hôm nay được cấp 5050 L đúng không?")
	wantLine(t, "S9", w.Validate.Reasons, `so sánh bằng tuyệt đối số người hỏi “5.050” với số có thẩm quyền 5.050 → "Đúng." (confirm_yes)`)
	wantLine(t, "S9 đầu vào", w.Validate.Reasons, "đầu vào của cổng: chủ thể PRJ-A · kỳ đọc được 1 · cụm xác nhận 1 · câu có dấu true · nguồn đã tạo nên con số warehouse_slip · con số xác nhận “5.050” L")
	wantLine(t, "S9 loại", w.Validate.Rejected, `phán quyết "Không." (confirm_no): bị loại — kết quả so sánh bằng tuyệt đối chọn confirm_yes`)
	wantLine(t, "S7 quyền", w.Context.Reasons, "quyền: phạm vi dữ liệu đã kiểm ở S5 cho thực thể gốc PRJ-A; truy vấn chạy trên 3 thực thể (gốc và con cháu), không lọc bằng chứng theo quyền sau truy vấn")
	wantLine(t, "S7 nguồn", w.Context.Reasons, "nguồn warehouse_slip (thẩm quyền 1): 2 bằng chứng — tạo nên con số chính")
	wantLine(t, "S3", w.Resolve.Reasons, "cụm “Dự án A” (khoá so khớp “du an a”) khớp PRJ-A (Cầu A, loại project)")
}

// Tri thức: hai đoạn truy xuất, MỘT được claim của bản nháp dẫn và một không; trần số đoạn khác số đoạn thực tế; tên nguồn của từng đoạn.
func TestVetDuChatTriThucHaiDoan(t *testing.T) {
	a := askDocs(t,
		sopDoc("SOP-FUEL-01", "Quy trình xử lý chênh lệch nhiên liệu", "1. Khi lượng xuất kho và lượng tiêu thụ đo được lệch quá dung sai, tổ trưởng kho lập biên bản đối soát trong vòng 24 giờ."),
		sopDoc("SOP-FUEL-02", "Quy trình xử lý chênh lệch nhiên liệu (phụ lục)", "Phụ lục: khi xử lý chênh lệch nhiên liệu, lưu biên bản tại kho và gửi phòng kế toán."))
	w := whyOfTurn(t, a, person("director", ""), "Quy trình xử lý chênh lệch nhiên liệu là gì?")
	wantList(t, "S7 lý do", w.Context.Reasons,
		"đã truy xuất 2 đoạn người hỏi được đọc (tối đa 3): SOP-FUEL-01#0 (Quy trình xử lý chênh lệch nhiên liệu), SOP-FUEL-02#0 (Quy trình xử lý chênh lệch nhiên liệu (phụ lục))",
		"quyền: ACL tài liệu do chặng truy xuất kiểm, chỉ đoạn người hỏi được đọc được trả về (số đoạn bị loại vì quyền không xuất khỏi chặng truy xuất)",
		"dẫn nguồn: 1 đoạn được claim của bản nháp dẫn: SOP-FUEL-01#0")
	wantList(t, "S7 loại", w.Context.Rejected, "SOP-FUEL-02#0: đã truy xuất nhưng không được claim nào dẫn nên không trích dẫn")
}

// Tri thức có bản nháp bị loại ở S9: S7 nói không đoạn nào được trích dẫn (và không có dòng "dẫn nguồn"). HYBRID gắn tiền tố đúng phần.
func TestVetDuChatBanNhapBiLoaiVaTienToHybrid(t *testing.T) {
	bad := sopDoc("SOP-FUEL-09", "Quy trình xử lý chênh lệch nhiên liệu", "Khi chênh lệch nhiên liệu quá 4000 lít thì tổ trưởng kho lập biên bản.")
	w := whyOfTurn(t, askDocs(t, bad), person("director", ""), "Quy trình xử lý chênh lệch nhiên liệu là gì?")
	wantList(t, "S7 lý do", w.Context.Reasons,
		"đã truy xuất 1 đoạn người hỏi được đọc (tối đa 3): SOP-FUEL-09#0 (Quy trình xử lý chênh lệch nhiên liệu)",
		"quyền: ACL tài liệu do chặng truy xuất kiểm, chỉ đoạn người hỏi được đọc được trả về (số đoạn bị loại vì quyền không xuất khỏi chặng truy xuất)")
	wantList(t, "S7 loại", w.Context.Rejected, "bản nháp bị loại ở S9 nên không đoạn nào được trích dẫn")

	const hybrid = "EX-03 hôm nay được cấp bao nhiêu dầu, và theo quy trình thì khi lệch phải làm gì?"
	h := whyOfTurn(t, askDocs(t, bad), person("director", ""), hybrid)
	wantList(t, "HYBRID bản nháp bị loại: loại", h.Context.Rejected,
		"số liệu: nguồn erp = 3900 L không được chọn làm con số chính (xung đột issued_fuel|EX-03|2026-10-19, chính sách authority): thẩm quyền thấp hơn nguồn được chọn fact:f03",
		"tri thức: bản nháp bị loại ở S9 nên không đoạn nào được trích dẫn")
	wantLine(t, "HYBRID số liệu", h.Context.Reasons, "số liệu: gói ngữ cảnh: 3 bằng chứng (2 dữ liệu nguồn, 1 bằng chứng khác của truy vấn, 0 bằng chứng đi kèm thêm sau truy vấn)")
	wantLine(t, "HYBRID tri thức", h.Context.Reasons, "tri thức: đã truy xuất 1 đoạn người hỏi được đọc (tối đa 3): SOP-FUEL-09#0 (Quy trình xử lý chênh lệch nhiên liệu)")

	ok := whyOfTurn(t, realAsk(t), person("director", ""), hybrid)
	wantList(t, "HYBRID: lý do", ok.Context.Reasons,
		"số liệu: gói ngữ cảnh: 3 bằng chứng (2 dữ liệu nguồn, 1 bằng chứng khác của truy vấn, 0 bằng chứng đi kèm thêm sau truy vấn)",
		"số liệu: nguồn warehouse_slip (thẩm quyền 1): 1 bằng chứng — tạo nên con số chính",
		"số liệu: nguồn erp (thẩm quyền 2): 1 bằng chứng — không tạo nên con số chính (nguồn thẩm quyền thấp hơn, hoặc lệch)",
		"số liệu: calc:issued_fuel: phép tính của con số chính (sum(fact:f03))",
		"số liệu: quyền: phạm vi dữ liệu đã kiểm ở S5 cho thực thể gốc EX-03; truy vấn chạy trên 1 thực thể (gốc và con cháu), không lọc bằng chứng theo quyền sau truy vấn",
		"tri thức: đã truy xuất 1 đoạn người hỏi được đọc (tối đa 3): SOP-FUEL-01#0 (Quy trình xử lý chênh lệch nhiên liệu)",
		"tri thức: quyền: ACL tài liệu do chặng truy xuất kiểm, chỉ đoạn người hỏi được đọc được trả về (số đoạn bị loại vì quyền không xuất khỏi chặng truy xuất)",
		"tri thức: dẫn nguồn: 1 đoạn được claim của bản nháp dẫn: SOP-FUEL-01#0")
	wantLine(t, "HYBRID S4", ok.Intent.Reasons, "ý định tri thức sop_question khớp cùng ý định số liệu — router ghép HYBRID nếu đủ khe")
}

// S4: thuật ngữ hiểu ngầm, ý định số liệu bị bỏ (câu quy trình), ý định tri thức bị bỏ (câu đã nêu mốc), chữ chưa được giải thích, khe thiếu.
func TestVetDuChatYDinhHieuNgamVaBiBo(t *testing.T) {
	director := person("director", "")
	imp := whyOfTurn(t, realAsk(t), director, "EX-03 hôm nay được cấp bao nhiêu?")
	wantLine(t, "hiểu ngầm", imp.Intent.Reasons, "thuật ngữ hiểu ngầm diesel_fuel (câu nhắc thực thể mà không nói vật tư và không có đại lượng đo được nào khác — chính sách của pack)")
	wantLine(t, "khớp đủ nhờ hiểu ngầm", imp.Intent.Reasons, "khớp đủ: fuel_issued (ưu tiên 20, từ khoá “được cấp”, thuật ngữ hiểu ngầm diesel_fuel)")
	wantLine(t, "chỉ số hiểu ngầm", imp.Intent.Reasons, "chỉ số issued_fuel (theo thuật ngữ diesel_fuel (hiểu ngầm) của ý định fuel_issued)")
	wantLine(t, "luật", imp.Intent.Reasons, "luật intent:rules+default_term:diesel_fuel")
	wantList(t, "ý định yếu bị loại", imp.Intent.Rejected, "fuel_generic: khớp đủ nhưng bị loại — ưu tiên 10 thấp hơn fuel_issued (20)")

	// ý định số liệu thiếu khe thực thể và câu không hỏi số liệu: bị bỏ, coi là câu quy trình
	proc := whyOfTurn(t, realAsk(t), director, "Quy trình cấp dầu trong bao lâu?")
	wantList(t, "số liệu bị bỏ", proc.Intent.Rejected, "fuel_issued: ý định số liệu bị bỏ — thiếu khe thực thể và câu không có dấu hiệu hỏi số liệu, coi là câu hỏi quy trình")
	wantLine(t, "chọn quy trình", proc.Intent.Reasons, "chọn sop_question: ý định duy nhất khớp đủ")

	// câu đã nêu mốc thời gian mà thiếu thực thể: hỏi lại phần số liệu, ý định tri thức bị bỏ
	dated := whyOfTurn(t, realAsk(t), director, "Hôm nay quy trình cấp dầu thế nào?")
	wantList(t, "tri thức bị bỏ", dated.Intent.Rejected,
		"sop_question (tri thức): khớp cùng nhưng bị bỏ — câu đã nêu mốc thời gian, ngày hỏng hay nhiều thực thể của một câu hỏi số liệu, hỏi lại phần số liệu thay vì chuyển sang câu hỏi quy trình")
	wantLine(t, "khe thiếu", dated.Intent.Reasons, "khe subject thiếu: không có thực thể rõ đúng loại (equipment|project)")

	// chữ lạ: cụm “…” nói đúng chữ chưa được giải thích
	odd := whyOfTurn(t, realAsk(t), director, "Dầu của EX-03 hôm nay xài 3850 L đúng không?")
	if !slices.ContainsFunc(odd.Intent.Reasons, func(l string) bool {
		return strings.Contains(l, "(chữ “xài” chưa được giải thích (không phải tên, mốc, lượng,")
	}) {
		t.Errorf("S4 phải nói chữ “xài” chưa được giải thích:\n%s", strings.Join(odd.Intent.Reasons, "\n"))
	}

	// ngày hỏng và nhiều mốc: khe period thiếu, kèm số mốc và số ngày hỏng ĐÚNG
	bad := whyOfTurn(t, realAsk(t), director, "EX-03 ngày 31/02 được cấp bao nhiêu dầu?")
	wantLine(t, "ngày hỏng S4", bad.Intent.Reasons, "khe period thiếu: câu nêu 0 mốc thời gian khác nhau và 1 ngày hỏng — không lấy mốc nhắc đầu, không rơi về ngày mặc định")
	wantList(t, "ngày hỏng S2", bad.Normalize.Rejected, "ngày “31/02” không đọc được thành ngày trên lịch: bỏ, không đoán và không rơi về ngày mặc định")
	two := whyOfTurn(t, realAsk(t), director, "EX-03 hôm nay và hôm qua được cấp 3850 L đúng không?")
	wantLine(t, "hai mốc", two.Intent.Reasons, "khe period thiếu: câu nêu 2 mốc thời gian khác nhau và 0 ngày hỏng — không lấy mốc nhắc đầu, không rơi về ngày mặc định")
}

// Câu hỏi lại vì hai máy cùng tên: ứng viên trước / sau lọc phạm vi gồm CẢ thực thể do S4 gộp thêm, nguồn từng ứng viên, và không lộ DZ-03.
func TestVetDuChatCauHoiLaiCuaKySu(t *testing.T) {
	w := whyOfTurn(t, realAsk(t), person("site_engineer", "PRJ-A"), "RL-02 và máy 03 hôm nay được cấp bao nhiêu dầu?")
	wantList(t, "S3 lý do", w.Resolve.Reasons,
		"cụm “RL-02” (khoá so khớp “rl 2”) khớp RL-02",
		"cụm “máy 03” (khoá so khớp “may 3”) khớp 2 thực thể: DZ-03, EX-03 — không chọn bừa",
		"ứng viên của câu hỏi lại: trước lọc phạm vi 3 ứng viên → sau lọc 2 ứng viên [RL-02, EX-03]")
	wantList(t, "S3 loại", w.Resolve.Rejected, "1 ứng viên bị loại khỏi câu hỏi lại: ngoài phạm vi dữ liệu của người hỏi (vết không nêu trạng thái phạm vi của từng ứng viên)")
	wantLine(t, "nguồn RL-02", w.Intent.Reasons, "ứng viên hỏi lại RL-02 ← do S4 gộp thêm: thực thể rõ “rl 2” ở S3, S4 gộp vào danh sách vì khe subject chỉ nhận một thực thể mà câu nêu 3")
	wantLine(t, "nguồn EX-03", w.Intent.Reasons, "ứng viên hỏi lại EX-03 ← S3: ứng viên của cụm mơ hồ “may 3”, S4 giữ trong danh sách vì khe subject chỉ nhận một thực thể mà câu nêu 3")
}

// F5: đầu vào của cổng phán quyết trong vết phải KHỚP điều kiện của verdictGap — khoá hai bên. Mã chiều cổng báo ("period", "accent", "marker"…)
// suy được từ ba đầu vào đã in (số kỳ đọc được, số cụm xác nhận, câu có dấu); đổi điều kiện ở verdictGap mà không đổi vết (hay ngược lại) thì test đỏ.
func TestVetDuCongPhanQuyetDauVaoKhopDieuKien(t *testing.T) {
	inputs := regexp.MustCompile(`kỳ đọc được (\d+) · cụm xác nhận (\d+) · câu có dấu (true|false)`)
	gap := regexp.MustCompile(`chiều "([a-z]+)" chưa giải được`)
	seen := map[string]int{}
	for _, q := range []string{
		"EX-03 hôm nay được cấp 3850 L đúng không?",
		"EX-03 hôm nay được cấp 4000 L đúng không?",
		"Dự án A hôm nay được cấp 5050 L đúng không?",
		"EX-03 được cấp 3850 L đúng không?",
		"đội báo 4k, báo cáo ghi 4 m3, vậy EX-03 được cấp 4k dầu đúng không?",
		"EX-03 hôm nay được cấp 3850 L đúng không, phải không?",
		"EX-03 hôm nay được cấp ~4000 L đúng không?",
		"EX-03 hôm nay được cấp 3850 đúng không?",
		"EX-03 hom nay duoc cap 3850 L dung khong?",
		"EX-03 hôm nay được cấp 3850 L và 4000 L đúng không?",
		"EX-03 hôm nay được cấp 3850 L, EX-07 thì sao, đúng không?",
		"Dầu xuất hôm nay của EX-03 là 3850 L đúng không?",
	} {
		w := whyOfTurn(t, realAsk(t), person("director", ""), q)
		var in string
		for _, l := range w.Validate.Reasons {
			if strings.Contains(l, "đầu vào của cổng") {
				in = l
			}
		}
		if in == "" {
			continue // câu không tới cổng (hỏi lại, hay cổng không áp dụng)
		}
		m := inputs.FindStringSubmatch(in)
		if m == nil {
			t.Fatalf("%q: không đọc được đầu vào của cổng: %s", q, in)
		}
		periods, _ := strconv.Atoi(m[1])
		markers, _ := strconv.Atoi(m[2])
		accent := m[3] == "true"
		code := ""
		for _, l := range w.Validate.Reasons {
			if g := gap.FindStringSubmatch(l); g != nil {
				code = g[1]
			}
		}
		seen[code]++
		var ok bool
		switch code {
		case "metric": // ý định chung: không do người hỏi nêu — không ràng buộc ba đầu vào
			ok = true
		case "period":
			ok = periods != 1
		case "accent":
			ok = periods == 1 && !accent
		case "marker":
			ok = periods == 1 && accent && markers != 1
		default: // phán quyết được hay các chiều sau ba chiều đầu (chủ thể, giá trị, nguồn, phần dư): ba đầu vào phải ở trạng thái "qua"
			ok = periods == 1 && accent && markers == 1
		}
		if !ok {
			t.Errorf("%q: cổng báo chiều %q nhưng đầu vào in ra là kỳ %d · cụm %d · dấu %v — vết và verdictGap lệch nhau", q, code, periods, markers, accent)
		}
	}
	for _, want := range []string{"", "period", "marker"} {
		if seen[want] == 0 {
			t.Errorf("bộ câu không còn ca nào kết thúc ở chiều %q (đã thấy %v): test không còn khoá được điều kiện đó", want, seen)
		}
	}
}
