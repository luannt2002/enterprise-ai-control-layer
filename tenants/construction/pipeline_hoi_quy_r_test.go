// Test đầu–cuối cho các HỒI QUY R-xx do bản sửa 6f23ff3 (A-01…A-13) gây ra, tìm ra ở vòng soát độc lập sau: bản cha 1276b5f hỏi lại /
// không phán quyết ở đúng những câu này, bản 6f23ff3 trả "Đúng." / "Không." cho sai kỳ, sai con số hoặc rơi về hôm nay. Pack THẬT, danh mục +
// fact giả lập, hôm nay = 19/10/2026. Mỗi ca ghi một hành vi nghiệp vụ; các ca "vẫn trả lời" giữ nguyên các sửa A-01…A-13 (hỏi lại OAN một
// câu rõ nghĩa cũng là lỗi — P5).
package construction_test

import (
	"slices"
	"strings"
	"testing"

	"github.com/luannt2002/enterprise-ai-control-layer/core/contracts"
	"github.com/luannt2002/enterprise-ai-control-layer/intelligence/models"
)

// khongPhanQuyet kiểm câu trả lời KHÔNG mở đầu bằng "Đúng." / "Không." — hệ thống không xác nhận hay bác con số nào cho người hỏi.
func khongPhanQuyet(t *testing.T, r contracts.AskResponse, text string) {
	t.Helper()
	if strings.HasPrefix(r.Answer, "Đúng.") || strings.HasPrefix(r.Answer, "Không.") {
		t.Errorf("%q: không được phán quyết, được [%s/%s] %q", text, r.Route, r.Outcome, r.Answer)
	}
}

// hoiLaiVe kiểm câu hỏi lại ở khe missing (và chứa mọi chuỗi contains trong câu hỏi).
func hoiLaiVe(t *testing.T, r contracts.AskResponse, text, missing string, contains ...string) {
	t.Helper()
	hoiLai(t, r, text)
	if r.Clarification == nil || !slices.Contains(r.Clarification.Missing, missing) {
		t.Errorf("%q: câu hỏi lại phải ở khe %q: %+v", text, missing, r.Clarification)
	}
	for _, s := range contains {
		if !strings.Contains(r.Answer, s) {
			t.Errorf("%q: câu hỏi lại thiếu %q: %q", text, s, r.Answer)
		}
	}
}

// R-02 — dải / liệt kê / lựa chọn: số trần nối với một lượng viết rõ đơn vị bằng "-", "đến", "hay", "hoặc", "chứ không phải"… dùng chung
// đơn vị của lượng đó, nên là con số THỨ HAI của câu. Bản 6f23ff3 bỏ nó như nhãn ("năm 2026", "ca 2") rồi trả "Không. … không phải 3.900 L"
// cho một dải chứa đúng 3.850 L, hoặc "Đúng." cho vế sau mà bỏ thầm vế trước. Bản cha hỏi lại.
func TestCauXacNhanDaiLietKeLuaChonConSoHoiLai(t *testing.T) {
	a := newAsk(t)
	for _, text := range []string{
		"EX-03 hôm nay được cấp từ 3800 đến 3900 L đúng không?",
		"EX-03 hôm nay được cấp 3800-3900 L đúng không?",
		"EX-03 hôm nay được cấp 3800 - 3900 L đúng không?",
		"EX-03 hôm nay được cấp từ 3 đến 4 m3 đúng không?",
		"EX-03 hôm nay được cấp từ 3800 tới 3900 L đúng không?",
		"EX-03 hôm nay được cấp 3850 chứ không phải 4000 L đúng không?",
		"EX-03 hôm nay được cấp 3850 hay 4000 L đúng không?",
		"EX-03 hôm nay được cấp 4000 hay 3850 L đúng không?",
		"EX-03 hôm nay được cấp 3850 hoặc 4000 L đúng không?",
		"EX-03 hôm nay được cấp 3850 và 4000 L đúng không?",
		"EX-03 hôm nay được cấp 3850 hay 4k đúng không?",
		// lượng viết rõ đơn vị đứng TRƯỚC, số trần nối sau nó
		"EX-03 hôm nay được cấp 3850 L hay 4000 đúng không?",
		"EX-03 hôm nay được cấp 3850 L chứ không phải 4000 đúng không?",
		"EX-03 hôm nay được cấp 3800 L đến 3900 đúng không?",
		"EX-03 hôm nay được cấp 3850 L (hoặc 4000) đúng không?",
		// từ nối LỰA CHỌN / PHỦ ĐỊNH mở đầu mệnh đề hỏi sau dấu phẩy: con số của mệnh đề trước là một vế, không phải ngữ cảnh
		"EX-03 hôm nay được cấp 3850 L, hay 4000 L đúng không?",
		"EX-03 hôm nay được cấp 3850 L, chứ không phải 4000 L đúng không?",
		"EX-03 hôm nay được cấp 3850 L, hoặc 4000 L, đúng không?",
		"EX-03 hôm nay được cấp 3850 L, không phải 4000 L đúng không?",
		// không viết hoa / không dấu
		"EX-03 hom nay duoc cap 3850 hay 4000 L dung khong?",
	} {
		hoiLaiVe(t, ask(t, a, user("director", ""), text).Response, text, "claimed", "con số cần xác nhận")
	}
}

// R-02g — lượng đứng cạnh một MẪU SỐ ("5.000 L/ngày" của "định mức") là một tốc độ, không phải số lượng đã cấp; lượng viết rõ đơn vị của
// câu không còn là một mình, nên số trần 3850 (được hỏi) không được bỏ thầm rồi so "5.000 L" với số thật ("Không. … không phải 5.000 L").
func TestCauXacNhanLuongTheoMauSoKhongThayConSoDuocHoi(t *testing.T) {
	a := newAsk(t)
	for _, text := range []string{
		"Định mức 5.000 L/ngày nên EX-03 hôm nay được cấp 3850 đúng không?",
		"Định mức 5 m3/ca nên EX-03 hôm nay được cấp 3850 đúng không?",
	} {
		hoiLaiVe(t, ask(t, a, user("director", ""), text).Response, text, "claimed")
	}
	// đối chứng: lượng theo mẫu số đứng MỘT mình vẫn là con số được hỏi (như bản cha); câu bình thường vẫn trả lời
	for _, c := range []struct{ text, prefix string }{
		{"EX-03 hôm nay được cấp 3850 L/ngày đúng không?", ""},
		{"EX-03 hôm nay được cấp 4000 L/ngày đúng không?", ""},
		{"EX-03 hôm nay được cấp 3850 L đúng không?", "Đúng."},
		{"EX-03 hôm nay được cấp 3850 đúng không?", ""},
		{"EX-03 hôm nay được cấp 4000 đúng không?", ""},
		{"EX-03 hôm nay được cấp 3,85 m3 đúng không?", "Đúng."},
		// cùng một con số viết hai cách nối với nhau vẫn là MỘT con số
		{"EX-03 hôm nay được cấp 3850 hay 3850 L đúng không?", ""},
		{"EX-03 hôm nay được cấp 3,85 hay 3,85 m3 đúng không?", ""},
		{"EX-03 hôm nay được cấp 3850 L, hay 3850 L đúng không?", ""},
		// "vậy", "và", "còn" mở đầu mệnh đề MỚI: con số của mệnh đề trước vẫn là ngữ cảnh (A-02)
		{"Đội báo 2k, vậy EX-03 hôm nay được cấp 3850 L đúng không?", "Đúng."},
		{"Kho xuất 4000 L, và EX-03 hôm nay được cấp 3850 L đúng không?", ""},
	} {
		xacNhan(t, ask(t, a, user("director", ""), c.text).Response, c.text, c.prefix)
	}
}

// R-02 (đối chứng với A-02) — từ nối chỉ nối hai CON SỐ: nhãn, số đếm và số của câu sau vẫn không là con số thứ hai (không hỏi lại về
// con số). Từ ADR-0009 các câu này trả số liệu KHÔNG phán quyết: nhãn / số đếm ("2 xe", "phiếu 912", "ca 2") là phần claim chưa kiểm được.
func TestCauXacNhanTuNoiKhongBienNhanThanhConSo(t *testing.T) {
	a := newAsk(t)
	for _, text := range []string{
		"EX-03 hôm nay được cấp 3850 L và 2 xe đúng không?",
		"EX-03 hôm nay được cấp 3850 L cho 1 máy đúng không?",
		"EX-03 hôm nay được cấp 3850 L, theo phiếu số 912 đúng không?",
		"EX-03 hôm nay được cấp 3850 L chứ không phải ca 2 đúng không?",
		"EX-03 hôm nay đội 3 và đội 4 được cấp 3850 L đúng không?",
		"EX-03 hôm nay ca 2 hay ca 3 được cấp 3850 L đúng không?",
		"EX-03 hôm nay được cấp 3850 L và phiếu 912 đúng không?",
	} {
		xacNhan(t, ask(t, a, user("director", ""), text).Response, text, "")
	}
	// Danh từ NGOÀI danh sách đóng của nhãn / số đếm: trước đây theo luật A-02 (số trần bị bỏ khi câu có lượng viết rõ đơn vị) nên ra "Đúng.".
	// Luật A-02 đã bỏ vì nó cũng đổi câu mơ hồ thành phán quyết SAI (soát R: R-A06, R-A07, B-02, B-03); số đi với danh từ lạ nay là con số thứ
	// hai ⇒ hỏi lại (hỏi lại thừa, không phán quyết sai). Kỳ vọng cũ "Đúng." đổi có chủ ý.
	for _, text := range []string{
		"EX-03 hôm nay được cấp 3850 L và 4 bể đúng không?",
		"EX-03 hôm nay được cấp 3850 L cho bể 7 đúng không?",
		"EX-03 hôm nay bể 7 được cấp 3850 L đúng không?",
	} {
		if r := ask(t, a, user("director", ""), text).Response; r.Outcome != contracts.OutcomeClarificationRequired {
			t.Errorf("%q: muốn hỏi lại, được [%s/%s] %q", text, r.Route, r.Outcome, r.Answer)
		}
	}
}

// R-10 — dấu chấm lửng "..." giữa câu không phải hết câu: "… 3850 L ... đúng không?" vẫn là câu xác nhận 3850 L (bản 6f23ff3 coi "..." là hết
// câu, số 3850 thành ngữ cảnh của câu trước và mất phán quyết). Một dấu chấm đơn rồi "Đúng không?" vẫn là câu mới (chủ ý của A-03).
func TestDauChamLungKhongPhaiHetCau(t *testing.T) {
	a := newAsk(t)
	for _, c := range []struct{ text, prefix string }{
		{"Hôm nay EX-03 được cấp 3850 L ... đúng không?", "Đúng."},
		{"Hôm nay EX-03 được cấp 3850 L... đúng không?", "Đúng."},
		{"Hôm nay EX-03 được cấp 4000 L ... đúng không?", "Không."},
		{"Hôm nay EX-03 được cấp 3850 L .... đúng không?", "Đúng."},
		{"Hôm nay EX-03 được cấp 3850 L … đúng không?", "Đúng."},
		{"Hôm nay EX-03 được cấp 3850 ... đúng không?", ""},
	} {
		xacNhan(t, ask(t, a, user("director", ""), c.text).Response, c.text, c.prefix)
	}
	// dấu chấm đơn hết câu: con số của câu trước là ngữ cảnh, "Đúng không?" không có con số → không phán quyết
	text := "Hôm nay EX-03 được cấp 3850 L. Đúng không?"
	khongPhanQuyet(t, ask(t, a, user("director", ""), text).Response, text)
}

// R-05 — "mỗi ngày 3-4 chuyến" nói TỐC ĐỘ (số chuyến mỗi ngày), không nói ngày 03/04: kỳ vẫn là mặc định (hôm nay) hoặc kỳ người hỏi nêu rõ.
// Bản 6f23ff3 đọc "3-4" thành 03/04/2026 (chữ "ngày" đứng trước, "chuyến" không phải đơn vị) hoặc hỏi lại "nhiều mốc thời gian" oan.
func TestMoiNgayCoKhoangSoDemKhongPhaiNgay(t *testing.T) {
	a := newAsk(t)
	for _, c := range []struct{ text, period string }{
		{"EX-03 mỗi ngày 3-4 chuyến được cấp bao nhiêu dầu?", "ngày 19/10/2026"},
		{"EX-03 mỗi ngày 10-12 xe tải đến được cấp bao nhiêu dầu?", "ngày 19/10/2026"},
		{"EX-03 mỗi ngày 2.5 chuyến được cấp bao nhiêu dầu?", "ngày 19/10/2026"},
		{"EX-03 hàng ngày 3-4 chuyến được cấp bao nhiêu dầu?", "ngày 19/10/2026"},
		{"EX-03 một ngày 3-4 chuyến được cấp bao nhiêu dầu?", "ngày 19/10/2026"},
		{"EX-03 từng ngày 3-4 chuyến được cấp bao nhiêu dầu?", "ngày 19/10/2026"},
		{"EX-03 mỗi ngày 3-4 chuyến, hôm qua được cấp bao nhiêu dầu?", "ngày 18/10/2026"},
		{"EX-03 mỗi ngày 10-12 xe tải đến, hôm qua được cấp bao nhiêu dầu?", "ngày 18/10/2026"},
		{"EX-03 mỗi ngày 3-4 chuyến, ngày 15/09 được cấp bao nhiêu dầu?", "ngày 15/09/2026"},
		// đối chứng: "ngày" không có từ chỉ tốc độ đứng trước thì vẫn đọc được (A-05)
		{"EX-03 ngày 15-09 được cấp bao nhiêu dầu?", "ngày 15/09/2026"},
		{"EX-03 ngày 15.09 được cấp bao nhiêu dầu?", "ngày 15/09/2026"},
	} {
		r := ask(t, a, user("director", ""), c.text).Response
		if r.Outcome == contracts.OutcomeClarificationRequired || !strings.Contains(r.Answer, c.period) {
			t.Errorf("%q: muốn trả lời cho %q, được [%s/%s] %q", c.text, c.period, r.Route, r.Outcome, r.Answer)
		}
	}
}

// R-07 — dạng ngày: cụm có thể là ngày gõ nhầm — đảo vế tháng/ngày ("10/19" = 19/10 viết kiểu tháng/ngày) hoặc ngày lố một chút ("32/10") —
// phải hỏi lại như bản cha, không rơi im lặng về hôm nay. Tỉ lệ / số phiếu thật sự ("24/24", "60/40", "12/34") vẫn không bị hỏi lại (A-06).
func TestNgayDaoVeHoacLoMotChutHoiLai(t *testing.T) {
	a := newAsk(t)
	for _, c := range []struct{ text, raw string }{
		{"EX-03 10/19 được cấp bao nhiêu dầu?", "10/19"},
		{"EX-03 10/19/26 được cấp bao nhiêu dầu?", "10/19/26"},
		{"EX-03 11/30 được cấp bao nhiêu dầu?", "11/30"},
		{"EX-03 32/10 được cấp bao nhiêu dầu?", "32/10"},
		{"EX-03 35/10 được cấp bao nhiêu dầu?", "35/10"},
		{"EX-03 hôm qua 10/19 được cấp bao nhiêu dầu?", "10/19"},
	} {
		r := ask(t, a, user("director", ""), c.text).Response
		hoiLai(t, r, c.text)
		if !strings.Contains(r.Answer, c.raw+" không hợp lệ") {
			t.Errorf("%q: câu hỏi lại phải nói rõ %q không hợp lệ: %q", c.text, c.raw, r.Answer)
		}
	}
	for _, text := range []string{
		"EX-03 chạy 24/24 hôm nay được cấp bao nhiêu dầu?",
		"EX-03 hôm nay tỷ lệ 60/40 được cấp bao nhiêu dầu?",
		"Phiếu 12/34 của EX-03 hôm nay được cấp bao nhiêu dầu?",
		"EX-03 hôm nay chạy 45/50 phút được cấp bao nhiêu dầu?",
		"EX-03 hôm nay 30/60 được cấp bao nhiêu dầu?",
	} {
		r := ask(t, a, user("director", ""), text).Response
		if r.Outcome != contracts.OutcomeResolved || !strings.Contains(r.Answer, "3.850 L") || strings.Contains(r.Answer, "không hợp lệ") {
			t.Errorf("%q: muốn RESOLVED 3.850 L, được [%s/%s] %q", text, r.Route, r.Outcome, r.Answer)
		}
	}
}

// R-06 — câu QUY TRÌNH thuần nêu hai thực thể (thủ tục, "khi lệch … thì phải làm gì") là câu hỏi quy trình: trích SOP, không hỏi lại "máy hoặc
// dự án" vì một từ khoá số liệu yếu ("cấp", "phiếu kho"). Câu ghép có vế SỐ LIỆU (hỏi lượng, xác nhận, hay một mệnh đề riêng chỉ có từ khoá số
// liệu) vẫn phải hỏi lại (A-04): không chuyển sang quy trình rồi bỏ thầm phần số liệu.
func TestCauQuyTrinhThuanNeuHaiThucTheVanLaCauQuyTrinh(t *testing.T) {
	a, m := newAskTriThuc(t)
	// SOP của fixture nói về chênh lệch / biên bản đối soát: câu về "lệch" tìm thấy đoạn đó (RESOLVED); câu về thủ tục xin cấp dầu không có tài liệu nào
	// trả lời — nhưng vẫn là route RAG (báo thiếu tài liệu), không phải câu hỏi lại "máy hoặc dự án".
	for _, c := range []struct {
		text     string
		resolved bool
	}{
		{"Khi lệch phiếu kho của EX-03 và RL-02 thì phải làm gì?", true},
		{"Phiếu kho của EX-03 và RL-02 lệch thì phải làm gì?", true},
		{"Quy trình xử lý khi lệch phiếu kho giữa EX-03 và RL-02 là gì?", true},
		{"Thủ tục xin cấp dầu cho dự án A và dự án B?", false},
		{"Việc cấp dầu cho EX-03 và RL-02 thực hiện theo quy trình nào?", false},
		{"Cấp dầu cho dự án A và dự án B cần thủ tục gì?", false},
	} {
		m.Reset()
		r := ask(t, a, user("director", ""), c.text).Response
		if r.Route != contracts.RouteRAG || (c.resolved && r.Outcome != contracts.OutcomeResolved) {
			t.Errorf("%q: muốn RAG (resolved=%v), được [%s/%s] %q", c.text, c.resolved, r.Route, r.Outcome, r.Answer)
		}
	}
	for _, text := range []string{
		"Thủ tục xin cấp dầu cho dự án A và dự án B là gì, mỗi dự án được cấp bao nhiêu?",
		"Khi lệch phiếu kho của EX-03 và RL-02 thì phải làm gì, EX-03 được cấp 3850 L đúng không?",
		"Quy trình định mức cấp dầu cho EX-03 và RL-02 là gì?",
		"Máy 03 và máy 07 được cấp dầu thế nào, quy trình xử lý khi lệch là gì?",
	} {
		m.Reset()
		r := ask(t, a, user("director", ""), text).Response
		hoiLai(t, r, text)
		if m.Calls() != 0 {
			t.Errorf("%q: hỏi lại thì không gọi model, được %d lần", text, m.Calls())
		}
	}
}

// R-01 — mốc thời gian người hỏi nêu mà bộ chuẩn hoá KHÔNG đọc được ("năm 2025", "tuần 41", "quý 3", "ngày 12", "7 ngày qua", "hôm 15 tháng 9")
// không được rơi về hôm nay / cả tháng rồi trả "Đúng." / "Không." cho sai kỳ. Bản cha hỏi lại (thông điệp sai nhưng an toàn); bản 6f23ff3 coi con số
// của mốc là nhãn vô hại ngay khi câu có một lượng viết rõ đơn vị. Nay: hỏi lại và nói rõ mốc nào chưa đọc được.
func TestMocThoiGianChuaDocDuocKhongPhanQuyetChoKyKhac(t *testing.T) {
	a := newAsk(t)
	for _, c := range []struct{ text, raw string }{
		{"EX-03 năm 2025 được cấp 3850 L đúng không?", "năm 2025"},
		{"EX-03 tuần 41 được cấp 3850 L đúng không?", "tuần 41"},
		{"EX-03 quý 3 được cấp 3850 L đúng không?", "quý 3"},
		{"EX-03 ngày 12 được cấp 3850 L đúng không?", "ngày 12"},
		{"EX-03 7 ngày qua được cấp 3850 L đúng không?", "7 ngày"},
		{"RL-02 hôm 15 tháng 9 được cấp 1100 L đúng không?", "15"},
		{"EX-03 15 tháng 9 được cấp 3850 L đúng không?", "15"},
		{"EX-03 mùng 5 được cấp 3850 L đúng không?", "mùng 5"},
		{"EX-03 hôm 12 được cấp 3850 L đúng không?", "hôm 12"},
		{"EX-03 tháng 10 ngày 12 được cấp 3850 L đúng không?", "ngày 12"},
		{"EX-03 ngày 12 tháng trước được cấp 3850 L đúng không?", "ngày 12"},
		{"EX-03 tuần 41 tháng 10 được cấp 3850 L đúng không?", "tuần 41"},
		{"EX-03 2 tuần qua được cấp 3850 L đúng không?", "2 tuần"},
		// câu mâu thuẫn: kỳ đọc được ("hôm nay", "tháng này") không khớp mốc người hỏi nêu thêm
		{"EX-03 hôm nay tuần 44 được cấp 3850 L đúng không?", "tuần 44"},
		{"EX-03 hôm nay năm 2025 được cấp 3850 L đúng không?", "năm 2025"},
		{"EX-03 hôm nay quý 3 được cấp 3850 L đúng không?", "quý 3"},
		{"EX-03 hôm nay ngày 12 được cấp 3850 L đúng không?", "ngày 12"},
		// gõ không dấu
		{"EX-03 tuan 41 duoc cap 3850 L dung khong?", "tuan 41"},
		// ngày viết dd-mm / dd-mm-yy mà bộ chuẩn hoá không đọc khi thiếu chữ "ngày" ("10-12 giờ", "3-4 chuyến" cũng có dạng đó nên không đọc im lặng)
		{"EX-03 19-10 được cấp 3850 L đúng không?", "19-10"},
		{"EX-03 19-10-26 được cấp 3850 L đúng không?", "19-10-26"},
		{"EX-03 được cấp 3850 L 19-10 đúng không?", "19-10"},
	} {
		hoiLaiVe(t, ask(t, a, user("director", ""), c.text).Response, c.text, "period", c.raw)
	}
	// đọc ĐÚNG kỳ người hỏi nêu: "tháng M năm Y" mang đủ năm — không còn rơi về tháng M của năm nay
	for _, c := range []struct{ text, period string }{
		{"Dự án A tháng 9 năm 2025 được cấp 4600 L đúng không?", "từ 01/09/2025 đến 30/09/2025"},
		{"EX-03 tháng 10 năm 2025 được cấp 3850 L đúng không?", "từ 01/10/2025 đến 31/10/2025"},
		{"EX-03 tháng 10/2025 được cấp 3850 L đúng không?", "từ 01/10/2025 đến 31/10/2025"},
		{"Dự án A tháng 9 năm 2025 được cấp bao nhiêu dầu?", "từ 01/09/2025 đến 30/09/2025"},
	} {
		r := ask(t, a, user("director", ""), c.text).Response
		khongPhanQuyet(t, r, c.text)
		if r.Outcome != contracts.OutcomeInsufficientEvidence || !strings.Contains(r.Answer, c.period) {
			t.Errorf("%q: muốn INSUFFICIENT_EVIDENCE cho %q, được [%s/%s] %q", c.text, c.period, r.Route, r.Outcome, r.Answer)
		}
	}
}

// R-01 (đối chứng với A-02) — mốc nêu thêm KHỚP kỳ đọc được là bối cảnh vô hại: câu vẫn được phán quyết. Hôm nay 19/10/2026 là thứ Hai của tuần ISO
// 43, quý 4, năm 2026, ngày 19; tháng 10 năm 2026 cũng khớp.
func TestMocThoiGianKhopKyDocDuocVanPhanQuyet(t *testing.T) {
	a := newAsk(t)
	for _, c := range []struct{ text, prefix string }{
		{"EX-03 hôm nay tuần 43 được cấp 3850 L đúng không?", "Đúng."},
		{"EX-03 tuần 43 hôm nay được cấp 3850 L đúng không?", "Đúng."},
		{"EX-03 hôm nay quý 4 được cấp 3850 L đúng không?", "Đúng."},
		{"EX-03 hôm nay năm 2026 được cấp 3850 L đúng không?", "Đúng."},
		{"EX-03 hôm nay ngày 19 được cấp 3850 L đúng không?", "Đúng."},
		{"EX-03 hôm nay tuần 43 được cấp 4000 L đúng không?", "Không."},
		{"EX-03 tháng 10 năm 2026 được cấp 3850 L đúng không?", "Đúng."},
		{"EX-03 ngày 19 tháng 10 năm 2026 được cấp 3850 L đúng không?", "Đúng."},
		// số trần không đơn vị: mốc khớp không làm số đó thành "con số thứ hai"
		{"EX-03 hôm nay tuần 43 được cấp 3850 đúng không?", ""},
		{"EX-03 hôm nay năm 2026 được cấp 4000 đúng không?", ""},
		// số trần ngay trước một mốc đã đọc là CON SỐ được hỏi, không phải số đếm tuần / tháng / năm của mốc đó
		{"EX-03 được cấp 3850 tuần này đúng không?", ""}, // tổng tuần này (19–25/10) là 3.850 L
		{"EX-03 được cấp 4000 tuần này đúng không?", ""},
		{"EX-03 được cấp 500 tuần này đúng không?", ""}, // số nhỏ trước "tuần này": con số, không đếm tuần (tuần của "tuần này" là kỳ đã đọc)
		{"EX-03 được cấp 3850 hôm nay đúng không?", ""},
		{"EX-03 được cấp 3850 tháng 10 đúng không?", ""},
		{"EX-03 xuất kho 3850 hôm nay đúng không?", ""},
		{"EX-03 hôm nay phiếu kho 3850 đúng không?", ""},
	} {
		xacNhan(t, ask(t, a, user("director", ""), c.text).Response, c.text, c.prefix)
	}
}

// R-04 — câu xác nhận KHÔNG nêu lượng nào mà chỉ có số nhãn ("tuần 43", "phiếu 912", "2 máy") không được phán quyết "Không. … không phải 43 L".
// "Tuần 43" là mốc thời gian chưa đọc được → hỏi lại; "phiếu 912", "2 máy" là nhãn / số đếm → trả lời bình thường, không phán quyết.
func TestCauXacNhanKhongNeuLuongNaoKhongPhanQuyetTheoSoNhan(t *testing.T) {
	a := newAsk(t)
	text := "Tuần 43 EX-03 được cấp dầu đúng không?"
	hoiLaiVe(t, ask(t, a, user("director", ""), text).Response, text, "period", "Tuần 43")
	for _, text := range []string{
		"EX-03 hôm nay đã được cấp dầu theo phiếu 912 đúng không?",
		"Dự án A hôm nay có đúng 2 máy được cấp dầu không?",
		"EX-03 hôm nay ca 2 được cấp dầu đúng không?",
		"EX-03 hôm nay đội 3 được cấp dầu đúng không?",
		"EX-03 hôm nay xe số 7 được cấp dầu đúng không?",
	} {
		r := ask(t, a, user("director", ""), text).Response
		khongPhanQuyet(t, r, text)
		if r.Outcome == contracts.OutcomeClarificationRequired {
			t.Errorf("%q: câu rõ nghĩa, không hỏi lại: %q", text, r.Answer)
		}
	}
	// số trần ĐỨNG SAU từ khoá số liệu vẫn là con số được hỏi (không phải nhãn)
	for _, c := range []struct{ text, prefix string }{
		{"EX-03 hôm nay được cấp 3850 đúng không?", ""},
		{"EX-03 hôm nay đã nhận 4000 đúng không?", ""},
		{"EX-03 hôm nay được cấp dầu 3850 đúng không?", "Đúng."},
		{"EX-03 hôm nay ca 2 được cấp 3850 đúng không?", ""},
		{"EX-03 hôm nay phiếu 912 cấp 4000 đúng không?", ""},
	} {
		xacNhan(t, ask(t, a, user("director", ""), c.text).Response, c.text, c.prefix)
	}
}

// R-03 — lượng nhiên liệu trong TÀI LIỆU lọt khi số và đơn vị bị chen một dấu trang trí (Markdown, ô bảng, dấu phẩy) hoặc đơn vị nằm trong ngoặc kèm nhãn
// ("(đơn vị: lít)"): bản 6f23ff3 (và cha) giao câu trả lời có "4.000 lít" lấy từ tài liệu (docs/04 G5: số lượng nhiên liệu nằm trong SOP bị chặn). Lớp kiểm ĐỘC LẬP
// (ForbidUnitWords) nay bắt các dạng đó; lớp chuẩn hoá (ForbidDimensions) vẫn viết riêng nên chưa nhất thiết bắt.
func TestLuongTaiLieuTrangTriVaNhanDonViKhongLot(t *testing.T) {
	for _, body := range []string{
		"Định mức cấp phát cho máy xúc: **4.000** lít mỗi ca.",
		"Định mức cấp phát cho máy xúc: `4.000` lít mỗi ca.",
		"Định mức cấp phát cho máy xúc: <b>4.000</b> lít mỗi ca.",
		"Định mức cấp phát cho máy xúc: *4.000* *lít* mỗi ca.",
		"Định mức cấp phát cho máy xúc là 4.000 (đơn vị: lít) mỗi ca.",
		"Định mức cấp phát cho máy xúc là 4.000 (ĐVT: lít) mỗi ca.",
		"Định mức cấp phát cho máy xúc là 4.000 (tính bằng lít) mỗi ca.",
		// Hàng bảng nêu chủ đề được hỏi: hàng trần "| Máy xúc | 4.000 | lít/ca |" (tài liệu không nói gì về định mức cấp phát) nay
		// dừng ở truy xuất — chủ đề ngoài kho, INSUFFICIENT_EVIDENCE, không lộ số (nhánh RAG, TestRAGKhongTraLoiBangDoanLacDe) —
		// nên không còn tới lớp kiểm mà test này canh.
		"| Định mức cấp phát máy xúc | 4.000 | lít/ca |",
		"Định mức cấp phát cho máy xúc: 4.000, lít, mỗi ca.",
	} {
		a := askVoiTaiLieu(t, body, &models.Extractive{})
		r := ask(t, a, user("director", ""), "Tài liệu định mức cấp phát cho máy xúc nói gì?").Response
		if r.Outcome != contracts.OutcomeSystemFailure || strings.Contains(r.Answer, "4.000") {
			t.Errorf("%q: lượng của tài liệu lọt ra: [%s] %q", body, r.Outcome, r.Answer)
			continue
		}
		if !violationValidators(r)["knowledge_quantity_independent/QUANTITY_FROM_DOCUMENT"] {
			t.Errorf("%q: lớp kiểm độc lập phải bắt, được %v", body, violationValidators(r))
		}
	}
	// đối chứng: câu quy trình hợp lệ có số không phải lượng nhiên liệu vẫn được giao
	for _, body := range []string{
		"| Bước | 24 | giờ |",
		"Biên bản đối soát lập trong vòng **24** giờ.",
		"1. **Bước 2** Lập biên bản đối soát trong vòng 24 giờ.",
	} {
		a := askVoiTaiLieu(t, body, &models.Extractive{})
		r := ask(t, a, user("director", ""), hoiQuyTrinh).Response
		if r.Outcome != contracts.OutcomeResolved || !r.Verdict.Pass {
			t.Errorf("%q: muốn RESOLVED, được [%s] %q (vi phạm %+v)", body, r.Outcome, r.Answer, r.Verdict.Violations)
		}
	}
}

// R-02 (đối chứng với A-02) — số trần ĐỨNG ĐẦU một mệnh đề hỏi vẫn là con số cạnh tranh khi không có từ nối nào nối nó với lượng ("; 4000 rồi 3850 L đúng không?").
func TestSoTranDauMenhDeKhongNoiVanLaConSoCanhTranh(t *testing.T) {
	a := newAsk(t)
	for _, text := range []string{
		"EX-03 hôm nay được cấp; 4000 rồi 3850 L đúng không?",
		"EX-03 hôm nay được cấp. 4000 rồi 3850 L đúng không?",
	} {
		hoiLaiVe(t, ask(t, a, user("director", ""), text).Response, text, "claimed")
	}
}

// R-01 — câu hỏi lại về mốc thời gian chỉ nói về MỐC: không kể ra các máy ứng viên (thực thể đã rõ nhờ phạm vi), không kê một mốc hai lần, và khi mốc là một ngày
// dd-mm thì các số của nó không còn là con số xác nhận thứ hai (khe thiếu chỉ là thời gian).
func TestHoiLaiVeMocThoiGianKhongKeUngVienVaKhongNhanDoi(t *testing.T) {
	a := newAsk(t)
	text := "Máy 03 tuần 41 được cấp 3850 L đúng không?"
	r := ask(t, a, user("site_engineer", "PRJ-A"), text).Response
	hoiLaiVe(t, r, text, "period", "tuần 41")
	if got := candidateIDs(r); len(got) != 0 {
		t.Errorf("%q: hỏi lại về mốc thời gian không được kể máy ứng viên: %v (%q)", text, got, r.Answer)
	}
	text = "EX-03 tuần 41 được cấp 3850 L (tuần 41) đúng không?"
	r = ask(t, a, user("director", ""), text).Response
	hoiLaiVe(t, r, text, "period", "tuần 41")
	if n := strings.Count(r.Answer, "tuần 41"); n != 1 {
		t.Errorf("%q: mốc viết hai lần chỉ được kê một lần, được %d lần: %q", text, n, r.Answer)
	}
	text = "EX-03 19-10 được cấp 3850 đúng không?"
	r = ask(t, a, user("director", ""), text).Response
	hoiLaiVe(t, r, text, "period", "19-10")
	if slices.Contains(r.Clarification.Missing, "claimed") {
		t.Errorf("%q: các số của ngày 19-10 không phải con số xác nhận thứ hai: %+v", text, r.Clarification)
	}
}
