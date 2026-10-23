// Test đầu–cuối cho các lỗi của bản soát độc lập lượt 2 (A-01…A-13 và ba mục người phản bác tìm thêm) — phần lớn là HỒI QUY
// do hai commit 5ce22ec (gấp ký tự, đơn vị trước số, maskWhole) và 3d730ea (S2-4: câu nhiều thực thể / kỳ / con số / ngày hỏng thì
// hỏi lại). Pack THẬT, danh mục + fact giả lập, model tất định; hôm nay = 19/10/2026. Hỏi lại OAN một câu chỉ có một vế cũng là
// sai như cắt thầm (P5): thông điệp hỏi lại phải đúng sự thật.
package construction_test

import (
	"slices"
	"strings"
	"testing"

	"github.com/luannt2002/enterprise-ai-control-layer/core/contracts"
	"github.com/luannt2002/enterprise-ai-control-layer/intelligence/models"
)

// xacNhan kiểm câu xác nhận ra đúng câu trả lời RESOLVED mở đầu bằng prefix ("Đúng." / "Không.") — không bị hỏi lại. prefix rỗng: claim
// chưa giải đủ chiều (ADR-0009) — vẫn RESOLVED, trả số liệu, nhưng KHÔNG phán quyết.
func xacNhan(t *testing.T, r contracts.AskResponse, text, prefix string) {
	t.Helper()
	if r.Outcome != contracts.OutcomeResolved || verdictOf(r.Answer) != prefix {
		t.Errorf("%q: muốn RESOLVED %q…, được [%s/%s] %q", text, prefix, r.Route, r.Outcome, r.Answer)
	}
}

// A-01 — số trong TÊN thực thể ("03" của "máy 03") không phải con số xác nhận, kể cả khi bí danh mơ hồ được router gỡ bằng phạm
// vi / dự án (khi đó cụm không mang vị trí) hoặc là lần nhắc thứ hai của cùng một máy.
func TestCauXacNhanBiDanhSoKhongPhaiConSoThuHai(t *testing.T) {
	a := newAsk(t)
	for _, c := range []struct {
		who          contracts.Principal
		text, prefix string
	}{
		// "máy 03" mơ hồ (EX-03 / DZ-03); phạm vi PRJ-A của người hỏi gỡ về EX-03
		{user("site_engineer", "PRJ-A"), "Máy 03 hôm nay được cấp 3850 L đúng không?", "Đúng."},
		{user("site_engineer", "PRJ-A"), "Máy 03 hôm nay được cấp 4000 L đúng không?", "Không."},
		// dự án người hỏi nêu gỡ về EX-03
		{user("director", ""), "Máy 03 của dự án A hôm nay được cấp 3850 L đúng không?", "Đúng."},
		// lần nhắc thứ hai của cùng một máy: bí danh mơ hồ chứa chính nó, hoặc bí danh rõ
		{user("director", ""), "EX-03 (máy 03) hôm nay được cấp 3850 L đúng không?", "Đúng."},
		{user("director", ""), "EX-03 (máy xúc 03) hôm nay được cấp 3850 L đúng không?", "Đúng."},
		// đối chứng: bí danh rõ ngay từ đầu
		{user("director", ""), "Máy xúc 03 hôm nay được cấp 3850 L đúng không?", "Đúng."},
		// con số của người hỏi không viết đơn vị: "03" là số trần duy nhất khác — chỉ loại theo vị trí cụm tên mới đúng
		{user("site_engineer", "PRJ-A"), "Máy 03 hôm nay được cấp 3850 đúng không?", ""},
		{user("director", ""), "Máy 03 của dự án A hôm nay được cấp 3850 đúng không?", ""},
		{user("director", ""), "EX-03 hôm nay được cấp 3850 đúng không?", ""},
		{user("director", ""), "EX-03 (máy 03) hôm nay được cấp 3850 đúng không?", ""},
		{user("director", ""), "EX-03 (máy xúc 03) hôm nay được cấp 4000 đúng không?", ""},
	} {
		xacNhan(t, ask(t, a, c.who, c.text).Response, c.text, c.prefix)
	}
}

// A-01 — khi câu xác nhận thật sự nêu hai con số và phải hỏi lại, câu hỏi lại là về CON SỐ: không kể ra các máy ứng viên (máy đã
// được gỡ rõ, "RL-07" ngoài dự án A càng không liên quan) — thông điệp phải đúng sự thật.
func TestHoiLaiConSoKhongKeUngVienThucThe(t *testing.T) {
	a := newAsk(t)
	for _, c := range []struct {
		who  contracts.Principal
		text string
	}{
		{user("site_engineer", "PRJ-A"), "Máy 03 hôm nay được cấp 3850 L và 4000 L đúng không?"},
		{user("director", ""), "Máy 03 của dự án A hôm nay được cấp 3850 L và 4000 L đúng không?"},
	} {
		r := ask(t, a, c.who, c.text).Response
		hoiLai(t, r, c.text)
		if got := candidateIDs(r); len(got) != 0 {
			t.Errorf("%q: hỏi lại về con số không được kể máy ứng viên: %v (%q)", c.text, got, r.Answer)
		}
		if !strings.Contains(r.Answer, "con số cần xác nhận") {
			t.Errorf("%q: câu hỏi lại phải nói đang hỏi con số: %q", c.text, r.Answer)
		}
	}
}

// A-02 + MISSED(rivalClaims) — chỉ MỘT lượng cùng chiều đo thì đó là con số xác nhận; số trần khác trong câu (năm, ca, số phiếu,
// số lần, tuần, số máy; kể cả ở các mệnh đề SAU mệnh đề hỏi) là nhãn, không phải con số thứ hai. Hỏi lại "nhiều con số khác nhau"
// ở đây là câu nói sai sự thật.
func TestCauXacNhanCoSoTranKhacVanTraLoi(t *testing.T) {
	a := newAsk(t)
	for _, c := range []struct{ text, prefix string }{
		{"EX-03 tháng 10 năm 2026 được cấp 3850 L đúng không?", "Đúng."},
		{"EX-03 ca 2 hôm nay được cấp 3850 L đúng không?", ""},
		{"EX-03 hôm nay đổ dầu 2 lần tổng 3850 L đúng không?", ""},
		{"Phiếu số 912 ghi EX-03 hôm nay được cấp 3850 L đúng không?", ""},
		{"EX-03 hôm nay tuần 43 được cấp 3850 L đúng không?", "Đúng."},
		{"EX-03 hôm nay được cấp 3850 L cho 1 máy đúng không?", ""},
		{"EX-03 ngày 19 tháng 10 được cấp 3850 L đúng không?", "Đúng."},
		{"EX-03 hôm nay ca 2 được cấp 4000 L đúng không?", ""},
		// số trần ở mệnh đề SAU mệnh đề hỏi
		{"EX-03 hôm nay được cấp 3850 L đúng không? (phiếu 912)", ""},
		{"EX-03 hôm nay được cấp 3850 L đúng không? Tôi cần trả lời trong 2 phút.", ""},
		{"EX-03 hôm nay được cấp 3850 L đúng không? Đội 3 báo vậy.", ""},
		{"EX-03 hôm nay được cấp 3850 L đúng không, theo phiếu số 912?", ""},
		{"EX-03 hôm nay được cấp 3850 L đúng không? Ca 2 báo vậy.", ""},
		{"EX-03 hôm nay được cấp 3850 L đúng không? Máy chạy 2 ca.", ""},
		// số khác chiều đo (7h, 2 ca) ở CÙNG mệnh đề không làm mất phán quyết "Đúng."
		{"EX-03 chạy 7h hôm nay được cấp 3850 L đúng không?", ""},
		{"EX-03 hôm nay chạy 2 ca được cấp 3850 L đúng không?", ""},
		// cùng một con số viết hai lần vẫn là MỘT con số
		{"EX-03 hôm nay được cấp 3850 L đúng không? Tôi nhớ 3850 L.", ""},
	} {
		xacNhan(t, ask(t, a, user("director", ""), c.text).Response, c.text, c.prefix)
	}
}

// A-02 — phần còn phải hỏi lại: hai con số CÙNG chiều đo (kể cả khi con số thứ hai viết gọn "4k" hay không có đơn vị).
func TestCauXacNhanHaiConSoVanHoiLaiKhiSoTranCoMuc(t *testing.T) {
	a := newAsk(t)
	for _, text := range []string{
		"EX-03 ca 2 hôm nay được cấp 3850 L đúng không, hay 4k?",
		"EX-03 hôm nay năm 2026 được cấp 3850 L và 4000 L đúng không?",
		"EX-03 hôm nay được cấp 3850 đúng không, 4 m3 thì sao?",
		"EX-03 hôm nay được cấp 3850 đúng không, 4000 thì sao?",
		// số trần đứng ĐẦU một mệnh đề cạnh một lượng viết rõ đơn vị vẫn là một giá trị được đem ra so
		"EX-03 hôm nay được cấp 3850 L đúng không, 4000 thì sao?",
		"EX-03 hôm nay được cấp 3850 L đúng không? 4000 thì sao?",
		"EX-03 hôm nay được cấp; 4000 hay 3850 L đúng không?",
		// chỉ có số trần (không lượng nào viết rõ đơn vị): số trần ở mệnh đề sau là con số cạnh tranh dù đứng sau một từ nối
		"EX-03 hôm nay được cấp 3850 đúng không? Còn 4000 thì sao?",
	} {
		r := ask(t, a, user("director", ""), text).Response
		hoiLai(t, r, text)
		if r.Clarification == nil || !slices.Contains(r.Clarification.Missing, "claimed") {
			t.Errorf("%q: câu hỏi lại phải ở khe claimed: %+v", text, r.Clarification)
		}
	}
}

// A-03 — dấu phẩy ngay trước cụm xác nhận ("…3850 L và 4000 L, đúng không?") làm mệnh đề hỏi không có số nào: các con số của
// mệnh đề liền trước vẫn là con số được hỏi, và nhiều con số khác nhau thì hỏi lại — không trả tổng như thể chỉ có một vế.
func TestCauXacNhanHaiConSoCoDauPhayTruocCumXacNhan(t *testing.T) {
	a := newAsk(t)
	for _, text := range []string{
		"EX-03 hôm nay được cấp 3850 L và 4000 L, đúng không?",
		"EX-03 hôm nay được cấp 3850 L hay 4000 L, đúng không?",
		"EX-03 hôm nay được cấp 3850 L, 4000 L, 4100 L, đúng không?",
		"EX-03 hôm nay được cấp 3850 L chứ không phải 4000 L, đúng không?",
		"EX-03 hôm nay được cấp 3850 L hay 4000 L, phải không?",
	} {
		r := ask(t, a, user("director", ""), text).Response
		hoiLai(t, r, text)
		if r.Clarification == nil || !slices.Contains(r.Clarification.Missing, "claimed") || !strings.Contains(r.Answer, "con số cần xác nhận") {
			t.Errorf("%q: câu hỏi lại phải nói đang hỏi con số cần xác nhận: %+v %q", text, r.Clarification, r.Answer)
		}
	}
	// một con số + dấu phẩy trước cụm xác nhận vẫn là câu xác nhận bình thường; con số ở CÂU trước là ngữ cảnh
	for _, c := range []struct{ text, prefix string }{
		{"EX-03 hôm nay được cấp 3850 L, đúng không?", "Đúng."},
		{"EX-03 hôm nay được cấp 4000 L, đúng không?", "Không."},
		{"Phiếu số 912. EX-03 hôm nay được cấp 3850 L, đúng không?", ""},
		// con số (có đơn vị) ở CÂU trước là ngữ cảnh, kể cả khi con số hỏi viết có dấu nghìn ("3.850" không cắt câu)
		{"Kho xuất 4000 L. EX-03 hôm nay được cấp 3850 L, đúng không?", "Đúng."},
		{"Kho xuất 4000 L. EX-03 hôm nay được cấp 3.850 L, đúng không?", "Đúng."},
	} {
		xacNhan(t, ask(t, a, user("director", ""), c.text).Response, c.text, c.prefix)
	}
}

// A-04 — câu ghép số liệu + quy trình nêu ≥ 2 thực thể cùng loại (hoặc ngày hỏng) mà không có mốc ngày hợp lệ phải hỏi lại như câu
// số liệu; không được chuyển sang câu hỏi quy trình rồi bỏ thầm phần số liệu và các máy.
func TestCauGhepSoLieuVaQuyTrinhNhieuThucTheThiHoiLai(t *testing.T) {
	a, m := newAskTriThuc(t)
	for _, c := range []struct {
		text string
		want []string
	}{
		{"Chênh lệch dầu của EX-03 và RL-02 là bao nhiêu và quy trình xử lý chênh lệch thế nào?", []string{"EX-03", "RL-02"}},
		// D-OP-11 = b: "máy 03" mơ hồ (DZ-03 hay EX-03), "máy 07" đã giải (EX-07): chỉ hỏi lại ứng viên của bí danh
		{"Máy 03 và máy 07 được cấp dầu thế nào, quy trình xử lý khi lệch là gì?", []string{"DZ-03", "EX-03"}},
		{"EX-03 và RL-02 ngày 31/02 được cấp bao nhiêu dầu, sau đó phải làm gì khi lệch?", []string{"EX-03", "RL-02"}},
		{"Dự án A và dự án B được cấp bao nhiêu dầu, và quy trình xử lý chênh lệch ra sao?", []string{"PRJ-A", "PRJ-B"}},
		// có mốc ngày hợp lệ thì vốn đã hỏi lại đúng
		{"EX-03 và RL-02 hôm nay được cấp bao nhiêu dầu, sau đó phải làm gì khi lệch?", []string{"EX-03", "RL-02"}},
	} {
		m.Reset()
		r := ask(t, a, user("director", ""), c.text).Response
		hoiLai(t, r, c.text)
		if got := candidateIDs(r); !slices.Equal(got, c.want) {
			t.Errorf("%q: ứng viên = %v, muốn %v", c.text, got, c.want)
		}
		if m.Calls() != 0 {
			t.Errorf("%q: hỏi lại thì không gọi model, được %d lần", c.text, m.Calls())
		}
	}
	// ngày hỏng, không nêu thực thể nào: người hỏi đã nêu một ngày — hỏi lại và nói rõ ngày nào không hợp lệ
	text := "Ngày 31/02 được cấp bao nhiêu dầu, và quy trình xử lý chênh lệch ra sao?"
	r := ask(t, a, user("director", ""), text).Response
	hoiLai(t, r, text)
	if !strings.Contains(r.Answer, "31/02 không hợp lệ") {
		t.Errorf("%q: câu hỏi lại phải nói 31/02 không hợp lệ: %q", text, r.Answer)
	}
	// không nêu thực thể, không nêu ngày: vẫn là câu hỏi quy trình (hành vi cũ giữ nguyên)
	for _, text := range []string{"Quy trình xử lý chênh lệch nhiên liệu là gì?", "Quy trình cấp dầu cho máy xúc ra sao, khi lệch thì làm gì?"} {
		if r := ask(t, a, user("director", ""), text).Response; r.Route != contracts.RouteRAG {
			t.Errorf("%q: không nêu thực thể hay ngày thì vẫn là câu hỏi quy trình (RAG), được %s %q", text, r.Route, r.Answer)
		}
	}
}

// A-05 — năm hai chữ số và dấu "." / "-" là cách viết ngày CÓ THẬT (docs/tenants/construction/03-data-map.md:162 liệt kê `19/10`,
// `19-10-2026`, `19.10.26`; 02-domain-map.md:71 và :296 thêm `19-10`): đọc đúng ngày thay vì rơi về hôm nay hay báo "không hợp
// lệ" cho một ngày có trên lịch. 15/09/2026 là ngày có fact (3.500 L) khác hôm nay — rơi về hôm nay sẽ ra 3.850 L.
func TestNgayVietNamHaiChuSoVaDauChamGachNoiDocDuoc(t *testing.T) {
	a := newAsk(t)
	for _, form := range []string{"15/9/26", "15/09/26", "15.09.26", "15-09-26", "15.09.2026", "15.09", "15-09", "15 / 09", "15/09"} {
		text := "EX-03 ngày " + form + " được cấp bao nhiêu dầu?"
		r := ask(t, a, user("director", ""), text).Response
		if r.Outcome != contracts.OutcomeResolved || !strings.Contains(r.Answer, "ngày 15/09/2026") || !strings.Contains(r.Answer, "3.500 L") {
			t.Errorf("%q: muốn RESOLVED ngày 15/09/2026: 3.500 L, được [%s/%s] %q", text, r.Route, r.Outcome, r.Answer)
		}
	}
}

// A-05 — ngày hỏng viết bằng "." / "-" / có dấu cách quanh dấu, hoặc năm hai chữ số, cũng phải hỏi lại và nói đúng cụm người hỏi viết.
func TestNgayHongVietDauChamGachNoiHoiLai(t *testing.T) {
	a := newAsk(t)
	for _, raw := range []string{"31.02", "31-02", "31.02.2026", "31 / 02", "31/02/26", "31.02.26", "31/09/26", "30.02.2026"} {
		text := "EX-03 ngày " + raw + " được cấp bao nhiêu dầu?"
		r := ask(t, a, user("director", ""), text).Response
		hoiLai(t, r, text)
		if !strings.Contains(r.Answer, raw+" không hợp lệ") {
			t.Errorf("%q: câu hỏi lại phải nói rõ %q không hợp lệ: %q", text, raw, r.Answer)
		}
	}
}

// A-06 — cụm số/số KHÔNG có dạng ngày (tỉ lệ, số phiếu, hợp đồng) không phải ngày hỏng: không hỏi lại "Ngày … không hợp lệ" khi
// người hỏi không nêu ngày nào.
func TestCumSoKhongPhaiNgayKhongBiHoiLaiOan(t *testing.T) {
	a := newAsk(t)
	for _, text := range []string{
		"EX-03 chạy 24/24 hôm nay được cấp bao nhiêu dầu?",
		"Phiếu 12/34 của EX-03 hôm nay được cấp bao nhiêu dầu?",
		"EX-03 hôm nay tỷ lệ 60/40 được cấp bao nhiêu dầu?",
		"EX-03 hôm nay được cấp bao nhiêu lít dầu (30/60)?",
		"EX-03 theo hợp đồng 12/34/56 hôm nay được cấp bao nhiêu dầu?",
		"EX-03 theo hợp đồng 1/2/3 hôm nay được cấp bao nhiêu dầu?",
		"EX-03 hôm nay chạy 45/50 phút được cấp bao nhiêu dầu?",
		"EX-03 hôm nay 123/456 được cấp bao nhiêu dầu?",
		"Phiếu PX-0912/10 của EX-03 hôm nay được cấp bao nhiêu dầu?",
		// chữ "ngày" đứng trước một lượng (dấu nghìn, khoảng, số thập phân có đơn vị) không biến lượng thành ngày
		"EX-03 mỗi ngày 4.000 lít hôm nay được cấp bao nhiêu dầu?",
		"EX-03 định mức một ngày 5.000 lít, hôm nay được cấp bao nhiêu dầu?",
		"EX-03 mỗi ngày 10-12 giờ, hôm nay được cấp bao nhiêu dầu?",
		"EX-03 mỗi ngày 2.5 tấn hôm nay được cấp bao nhiêu dầu?",
	} {
		r := ask(t, a, user("director", ""), text).Response
		if r.Outcome != contracts.OutcomeResolved || !strings.Contains(r.Answer, "3.850 L") || strings.Contains(r.Answer, "không hợp lệ") {
			t.Errorf("%q: muốn RESOLVED 3.850 L, được [%s/%s] %q", text, r.Route, r.Outcome, r.Answer)
		}
	}
}

// A-05 — ngày viết dd/mm (có dấu "/") luôn đọc được, kể cả khi ngay sau là một chữ trùng đơn vị ("ca" = ca máy): chỉ ngày đọc nhờ chữ
// "ngày" mà không có dấu "/" ("ngày 10-12 giờ", "ngày 2.5 tấn") mới bị bỏ khi ngay sau là đơn vị. 15/09/2026 có fact (3.500 L) khác hôm nay.
func TestNgayCoDauGachCheoTheoSauLaCaVanDocDuoc(t *testing.T) {
	a := newAsk(t)
	for _, text := range []string{
		"EX-03 ngày 15/09 ca đêm được cấp bao nhiêu dầu?", "EX-03 ngày 15/09/2026 ca sáng được cấp bao nhiêu dầu?", "EX-03 ngày 15/9/26 ca 2 được cấp bao nhiêu dầu?",
	} {
		r := ask(t, a, user("director", ""), text).Response
		if r.Outcome != contracts.OutcomeResolved || !strings.Contains(r.Answer, "ngày 15/09/2026: 3.500 L") {
			t.Errorf("%q: muốn RESOLVED ngày 15/09/2026: 3.500 L, được [%s/%s] %q", text, r.Route, r.Outcome, r.Answer)
		}
	}
}

// A-07 — "ngày D tháng M [năm Y]" là MỘT ngày (không phải "tháng M" bỏ "ngày D"); cùng một ngày viết hai cách ("hôm nay, ngày 19
// tháng 10") vẫn là một kỳ.
func TestNgayThangVietChuLaMotNgay(t *testing.T) {
	a := newAsk(t)
	for _, c := range []struct{ text, period, value string }{
		{"EX-03 hôm nay, ngày 19 tháng 10, được cấp bao nhiêu dầu?", "ngày 19/10/2026", "3.850 L"},
		{"EX-03 hôm nay tức ngày 19 tháng 10 được cấp bao nhiêu dầu?", "ngày 19/10/2026", "3.850 L"},
		{"EX-03 ngày 19 tháng 10 năm 2026 được cấp bao nhiêu dầu?", "ngày 19/10/2026", "3.850 L"},
		{"EX-03 ngày 15 tháng 9 được cấp bao nhiêu dầu?", "ngày 15/09/2026", "3.500 L"},
		{"EX-03 ngày 15 tháng 9 năm 2026 được cấp bao nhiêu dầu?", "ngày 15/09/2026", "3.500 L"},
		{"EX-03 ngày 15 tháng 9/2026 được cấp bao nhiêu dầu?", "ngày 15/09/2026", "3.500 L"},
	} {
		r := ask(t, a, user("director", ""), c.text).Response
		if r.Outcome != contracts.OutcomeResolved || !strings.Contains(r.Answer, c.period+": "+c.value) {
			t.Errorf("%q: muốn RESOLVED %q: %s, được [%s/%s] %q", c.text, c.period, c.value, r.Route, r.Outcome, r.Answer)
		}
	}
}

// A-07 — "từ ngày D đến ngày D2 tháng M": hai đầu mút (đầu mút thứ nhất mượn tháng của đầu mút thứ hai) nên hỏi lại, không trả
// cả tháng cũng không trả riêng ngày cuối; "ngày 31 tháng 2" không có trên lịch.
func TestKhoangNgayVietChuHoiLai(t *testing.T) {
	a := newAsk(t)
	for _, c := range []struct {
		text     string
		contains []string
	}{
		{"EX-03 từ ngày 10 đến ngày 15 tháng 10 được cấp bao nhiêu dầu?", []string{"10/10/2026", "15/10/2026"}},
		{"EX-03 từ ngày 10 tới ngày 15 tháng 10 được cấp bao nhiêu dầu?", []string{"10/10/2026", "15/10/2026"}},
		{"EX-03 từ ngày 10 tháng 10 đến ngày 15 tháng 10 được cấp bao nhiêu dầu?", []string{"10/10/2026", "15/10/2026"}},
		{"EX-03 ngày 31 tháng 2 được cấp bao nhiêu dầu?", []string{"31 tháng 2 không hợp lệ"}},
	} {
		r := ask(t, a, user("director", ""), c.text).Response
		hoiLai(t, r, c.text)
		for _, s := range c.contains {
			if !strings.Contains(r.Answer, s) {
				t.Errorf("%q: câu hỏi lại thiếu %q: %q", c.text, s, r.Answer)
			}
		}
	}
}

// MISSED — kỳ LỒNG nhau người hỏi viết thành một cụm tự nhiên ("tháng 10 năm nay", "hôm nay trong tháng 10") là MỘT kỳ — kỳ hẹp;
// kỳ rộng là bối cảnh. Hai kỳ nối bằng "và" hay làm hai đầu mút của một khoảng thì vẫn là nhiều kỳ.
func TestKyLongNhauLaMotKy(t *testing.T) {
	a := newAsk(t)
	for _, c := range []struct{ text, period string }{
		{"EX-03 tháng 10 năm nay được cấp bao nhiêu dầu?", "từ 01/10/2026 đến 31/10/2026"},
		{"EX-03 ngày 19/10 năm nay được cấp bao nhiêu dầu?", "ngày 19/10/2026"},
		{"EX-03 hôm nay trong tháng 10 được cấp bao nhiêu dầu?", "ngày 19/10/2026"},
		{"EX-03 hôm nay thuộc tuần này được cấp bao nhiêu dầu?", "ngày 19/10/2026"},
	} {
		r := ask(t, a, user("director", ""), c.text).Response
		if r.Outcome != contracts.OutcomeResolved || !strings.Contains(r.Answer, c.period+": 3.850 L") {
			t.Errorf("%q: muốn RESOLVED %q: 3.850 L, được [%s/%s] %q", c.text, c.period, r.Route, r.Outcome, r.Answer)
		}
	}
	for _, text := range []string{
		"EX-03 tháng 10 và hôm nay được cấp bao nhiêu dầu?",
		"EX-03 tuần trước, từ 12/10, được cấp bao nhiêu dầu?",
		"EX-03 từ 10/10 đến 15/10 được cấp bao nhiêu dầu?",
	} {
		hoiLai(t, ask(t, a, user("director", ""), text).Response, text)
	}
}

// MISSED — mã có đuôi dd/mm ("PX-12/10", "BB-24/50") là mã tài liệu, không phải ngày: không sinh kỳ thứ hai, không báo ngày hỏng.
func TestMaCoDuoiGiongNgayKhongPhaiNgay(t *testing.T) {
	a := newAsk(t)
	for _, text := range []string{
		"Phiếu PX-12/10 của EX-03 hôm nay được cấp bao nhiêu dầu?",
		"EX-03 theo mã BB-24/50 hôm nay được cấp bao nhiêu dầu?", // (không dùng chữ "biên bản": đó là từ khoá câu hỏi quy trình)
		"EX-03 hôm nay theo phiếu TK-12/10 được cấp bao nhiêu dầu?",
	} {
		r := ask(t, a, user("director", ""), text).Response
		if r.Outcome != contracts.OutcomeResolved || !strings.Contains(r.Answer, "ngày 19/10/2026: 3.850 L") {
			t.Errorf("%q: muốn RESOLVED ngày 19/10/2026: 3.850 L, được [%s/%s] %q", text, r.Route, r.Outcome, r.Answer)
		}
	}
}

// --- tài liệu: lượng nghiệp vụ không được lọt (A-08, A-13) và câu quy trình hợp lệ không bị chặn oan (A-09) ---

// A-08 — lượng nghiệp vụ trong tài liệu lọt CẢ HAI lớp kiểm ở các dạng ngoài danh sách đã khai: "mét khối" (cách viết trang trọng
// của m³), "m^3", đơn vị MỞ ĐẦU ngoặc kèm chữ phụ "(lít dầu)", số trong ngoặc "(4.000) lít", đơn vị ghép đứng trước số "lít/ca:".
func TestLuongTrongTaiLieuDangVietKhacKhongLot(t *testing.T) {
	for _, body := range []string{
		"Định mức dầu cho máy xúc là 5.000 mét khối mỗi ngày.",
		"Máy xúc được cấp 5 mét khối dầu mỗi ngày.",
		"Định mức cấp phát dầu: 5.000 m^3 mỗi ngày.",
		"Mỗi ca máy xúc được cấp tối đa 4.000 (lít dầu).",
		"Mỗi ca máy xúc được cấp tối đa (4.000) lít dầu.",
		"Hạn mức tối đa: lít/ca: 4.000 cho máy xúc.",
		"Từ nay định mức cấp phát cho máy xúc là (lít/ngày): 5.000 mỗi ngày.",
	} {
		a := askVoiTaiLieu(t, body, &models.Extractive{})
		r := ask(t, a, user("director", ""), hoiQuyTrinh).Response
		if r.Outcome != contracts.OutcomeSystemFailure || strings.Contains(r.Answer, "5.000") || strings.Contains(r.Answer, "4.000") {
			t.Errorf("%q: lượng của tài liệu lọt ra: [%s] %q", body, r.Outcome, r.Answer)
			continue
		}
		got := violationValidators(r)
		if !got["knowledge_quantity/QUANTITY_FROM_DOCUMENT"] || !got["knowledge_quantity_independent/QUANTITY_FROM_DOCUMENT"] {
			t.Errorf("%q: cả hai lớp kiểm (chuẩn hoá và bộ đếm độc lập) đều phải bắt, được %v", body, got)
		}
	}
}

// A-09 — đơn vị đứng TRƯỚC số chỉ là đơn vị của số khi đúng là cột/nhãn đơn vị của một LƯỢNG: mẫu số của giá ("đồng/lít"), nhãn bộ
// phận ("khối 2"), nhãn mục ("L 3", "l 2") và số đếm ("lít 2 liên") không phải lượng — chặn nhầm là chặn câu trả lời quy trình
// hợp lệ.
func TestCauQuyTrinhCoDonGiaVaNhanBoPhanVanDuocGiao(t *testing.T) {
	for _, body := range []string{
		"Đơn giá niêm yết (đồng/lít): 25.000 theo bảng giá.",
		"Bảng giá: đồng/lít 25.000.",
		"Đơn giá (VND/lít): 25.000.",
		"Đơn giá (đồng/m3): 25.000.000.",
		"Mức phạt (đồng/lít) 500.000.",
		"Đơn giá: 25.000 đồng/lít theo bảng giá.",
		"Bảng giá: USD/lít 25.000.", "Đơn giá (đồng/lít/ca): 25.000.", "Gửi email 5.000 lần.",
		"Phòng Kế hoạch thuộc khối 2 lập biên bản đối soát trong vòng 24 giờ.",
		"Biên bản đối soát do khối 3 lập trong vòng 24 giờ.",
		"Xem phụ lục L 3 về biên bản đối soát trong vòng 24 giờ.",
		"Xem mục l 2 về biên bản đối soát trong vòng 24 giờ.",
		"Mẫu BB-01, lít 2 liên, lập trong vòng 24 giờ.",
		"Biên bản gửi khối 3 và khối 4 trong vòng 24 giờ.",
	} {
		a := askVoiTaiLieu(t, body, &models.Extractive{})
		r := ask(t, a, user("director", ""), hoiQuyTrinh).Response
		if r.Outcome != contracts.OutcomeResolved || !r.Verdict.Pass {
			t.Errorf("%q: muốn RESOLVED, được [%s] %q (vi phạm %+v)", body, r.Outcome, r.Answer, r.Verdict.Violations)
		}
	}
}

// A-13 — hậu tố nhân cách số bằng nhiều dấu cách hay ký tự vô hình ("5  k lít", "5<ZWSP>k lít") là cùng lượng với "5k lít".
func TestLuongCoHauToNhanCachNhieuKhoangDemKhongLot(t *testing.T) {
	for _, form := range []string{"5k lít", "5 k lít", "5  k lít", "5\u200bk lít", "5 k  lít", "5\u00a0\u00a0k lít"} {
		body := "Định mức cấp phát cho máy xúc là " + form + " mỗi ngày."
		a := askVoiTaiLieu(t, body, &models.Extractive{})
		r := ask(t, a, user("director", ""), hoiQuyTrinh).Response
		if r.Outcome != contracts.OutcomeSystemFailure || !violationValidators(r)["knowledge_quantity/QUANTITY_FROM_DOCUMENT"] {
			t.Errorf("%q: lượng của tài liệu lọt ra: [%s] %q (vi phạm %v)", form, r.Outcome, r.Answer, violationValidators(r))
		}
	}
}
