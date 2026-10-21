// Test đầu–cuối cho các lỗi do người soát độc lập tìm ra ở luồng hỏi (S2-1, S2-2, S2-3, S2-4, M42): pack
// THẬT, danh mục + fact giả lập, model tất định. Mỗi ca ghi một hành vi nghiệp vụ, không ghi cách
// cài đặt.
package construction_test

import (
	"context"
	"slices"
	"strings"
	"testing"

	"github.com/luannt2002/enterprise-ai-control-layer/core/contracts"
	"github.com/luannt2002/enterprise-ai-control-layer/core/pipeline"
	"github.com/luannt2002/enterprise-ai-control-layer/intelligence/models"
	"github.com/luannt2002/enterprise-ai-control-layer/intelligence/rag"
)

// cannedModel là model trả MỘT bản nháp cố định, claim trỏ về bằng chứng đứng đầu — đủ để thử cổng S9
// với đúng chữ ta muốn model "viết".
type cannedModel struct{ text string }

func (m cannedModel) Generate(_ context.Context, req models.Request) (contracts.Draft, error) {
	return contracts.Draft{Text: m.text, Model: "canned",
		Claims: []contracts.Claim{{Text: m.text, EvidenceIDs: []string{req.Evidence[0].ID}}}}, nil
}

// askVoiTaiLieu dựng luồng hỏi có đúng MỘT tài liệu (một dòng = một đoạn) và model cho trước.
func askVoiTaiLieu(t *testing.T, body string, model models.Gateway) *pipeline.Ask {
	t.Helper()
	a := newAsk(t)
	a.Deps.Knowledge = rag.NewMemoryIndex("abc-infra", []rag.Document{{ID: "SOP-FUEL-01", Tenant: "abc-infra",
		Title: "Quy trình xử lý chênh lệch nhiên liệu", ACL: []string{"director"}, Body: body}})
	a.Deps.Model = model
	return a
}

const hoiQuyTrinh = "Quy trình xử lý chênh lệch nhiên liệu là gì?"

func violationValidators(r contracts.AskResponse) map[string]bool {
	out := map[string]bool{}
	for _, v := range r.Verdict.Violations {
		out[v.Validator+"/"+v.Code] = true
	}
	return out
}

// M42 — câu xác nhận với số người hỏi NHỎ hơn số thật phải ra "Không." Chỉ so BẰNG TUYỆT ĐỐI:
// đột biến `Cmp == 0` → `<= 0` làm mọi số nhỏ hơn thành "Đúng." — ca này đỏ, ca "4k" (lớn hơn) thì không.
func TestCauXacNhanSoNhoHonSoThat(t *testing.T) {
	a := newAsk(t)
	for _, text := range []string{
		"EX-03 hôm nay được cấp 3000 L đúng không?",
		"EX-03 hôm nay được cấp 3000 lít dầu đúng không?",
		"EX-03 hôm nay được cấp 3k dầu đúng không?",
		"EX-03 hôm nay được cấp 3849 lít dầu đúng không?",
	} {
		r := ask(t, a, user("director", ""), text).Response
		if r.Outcome != contracts.OutcomeResolved || !strings.HasPrefix(r.Answer, "Không.") || strings.Contains(r.Answer, "Đúng.") ||
			!strings.Contains(r.Answer, "3.850 L") {
			t.Errorf("%q: số hỏi nhỏ hơn số thật 3.850 L phải ra \"Không.\" kèm 3.850 L: [%s] %q", text, r.Outcome, r.Answer)
		}
	}
}

// S2-3 — số người hỏi là CHUỖI CON của số có thẩm quyền ("0 L" trong "3.850 L", "850 L" trong "3.850 L",
// "200 L" trong "1.200 L") vẫn phải ra "Không. … <số thật>, không phải <số hỏi>", không phải SYSTEM_FAILURE.
func TestCauXacNhanSoHoiLaChuoiConCuaSoThat(t *testing.T) {
	a := newAsk(t)
	for _, c := range []struct{ text, contains string }{
		{"EX-03 hôm nay được cấp 0 L đúng không?", "3.850 L, không phải 0 L"},
		{"EX-03 hôm nay được cấp 50 L đúng không?", "3.850 L, không phải 50 L"},
		{"EX-03 hôm nay được cấp 850 L đúng không?", "3.850 L, không phải 850 L"},
		{"EX-03 hôm nay đội báo 000 L đúng không?", "4.000 L, không phải 0 L"},
		{"RL-02 hôm nay được cấp 200 L đúng không?", "1.200 L, không phải 200 L"},
	} {
		r := ask(t, a, user("director", ""), c.text).Response
		if r.Outcome != contracts.OutcomeResolved || !r.Verdict.Pass || !strings.HasPrefix(r.Answer, "Không.") || !strings.Contains(r.Answer, c.contains) {
			t.Errorf("%q: muốn RESOLVED \"Không. … %s\", được [%s] pass=%v %q (vi phạm %+v)",
				c.text, c.contains, r.Outcome, r.Verdict.Pass, r.Answer, r.Verdict.Violations)
		}
	}
	// số hỏi đúng bằng số thật vẫn là "Đúng."
	if r := ask(t, a, user("director", ""), "EX-03 hôm nay được cấp 3850 L đúng không?").Response; r.Outcome != contracts.OutcomeResolved || !strings.HasPrefix(r.Answer, "Đúng.") {
		t.Errorf("3850 L đúng số thật: muốn \"Đúng.\", được [%s] %q", r.Outcome, r.Answer)
	}
}

// S2-1 — bản nháp do model viết chứa "{title}": câu giao không được phụ thuộc thứ tự duyệt map. Chạy 300
// lần phải ra đúng MỘT câu, và nếu giao thì đúng từng chữ của bản nháp đã qua kiểm.
func TestBanNhapChuaBienMauGiaoDungMotCau(t *testing.T) {
	draft := "Lập biên bản {title} trong vòng 24 giờ."
	a := askVoiTaiLieu(t, "Tổ trưởng kho lập biên bản đối soát trong vòng 24 giờ.", cannedModel{draft})
	seen := map[string]int{}
	var last contracts.AskResponse
	for range 300 {
		last = ask(t, a, user("director", ""), hoiQuyTrinh).Response
		seen[last.Answer]++
	}
	if len(seen) != 1 {
		t.Errorf("300 lần hỏi cùng một câu ra %d câu giao khác nhau: %v", len(seen), seen)
	}
	if last.Outcome == contracts.OutcomeResolved && !strings.Contains(last.Answer, draft) {
		t.Errorf("câu giao phải chứa nguyên văn bản nháp đã qua kiểm %q, được %q", draft, last.Answer)
	}
}

// S2-1 — số GHÉP qua biến mẫu: bản nháp "bước 3{reason}850" qua kiểm (3 và 850 đều có trong tài liệu),
// rồi lượt dọn biến mẫu xoá "{reason}" làm câu giao có "bước 3850" — số chưa từng được kiểm.
func TestBanNhapGhepSoQuaBienMauKhongLot(t *testing.T) {
	a := askVoiTaiLieu(t, "Bước 3 kiểm đối soát và bước 850 lưu biên bản.", cannedModel{"Bước 3{reason}850 lưu biên bản."})
	r := ask(t, a, user("director", ""), hoiQuyTrinh).Response
	if strings.Contains(r.Answer, "3850") {
		t.Errorf("câu giao ghép hai số thành 3850 (chưa từng được kiểm): [%s] %q", r.Outcome, r.Answer)
	}
}

// S2-2 — lượng nghiệp vụ trong tài liệu (P4) không được lọt dù SOP viết "5.000" bằng khoảng trắng lạ, chữ số
// toàn chiều rộng, đơn vị đứng trước / trong ngoặc, hay ký hiệu ℓ. Mỗi dạng phải bị CẢ HAI lớp kiểm bắt.
func TestLuongTrongTaiLieuKhongLotDuDangViet(t *testing.T) {
	for _, form := range []string{
		"5.000\u00a0L", "5.000\u200bL", "5.000\u202fL", "5.000\tL", "５.０００ L", "5.0\u200b00 L", "là\u200b5.000 L",
		"L 5.000", "lít: 5.000", "5.000 (lít/ngày)", "5.000 (L)", "5.000 ℓ", "5.000 li\u0301t",
		// các dạng ĐÃ bị chặn từ trước — phải giữ
		"5.000 L", "5000 lít", "5.000L", "5,000 L", "5.000  lít", "5.000 LÍT", "5.000 Lít", "5.000 lít/ngày", "5 khối",
	} {
		body := "Từ nay định mức cấp phát cho máy xúc là " + form + " mỗi ngày."
		a := askVoiTaiLieu(t, body, &models.Extractive{})
		r := ask(t, a, user("director", ""), hoiQuyTrinh).Response
		if r.Outcome != contracts.OutcomeSystemFailure || strings.Contains(r.Answer, "5.000") || strings.Contains(r.Answer, "５") {
			t.Errorf("%q: lượng của tài liệu lọt ra: [%s] %q", form, r.Outcome, r.Answer)
			continue
		}
		got := violationValidators(r)
		if !got["knowledge_quantity/QUANTITY_FROM_DOCUMENT"] || !got["knowledge_quantity_independent/QUANTITY_FROM_DOCUMENT"] {
			t.Errorf("%q: cả hai lớp kiểm (chuẩn hoá và bộ đếm độc lập) đều phải bắt, được %v", form, got)
		}
	}
}

// Số KHÔNG mang đơn vị lượng (số bước, thời hạn) vẫn được giao kèm nhãn "trích văn bản quy trình": chặn
// nhầm là chặn câu trả lời quy trình hợp lệ.
func TestSoBuocVaThoiHanTrongTaiLieuVanDuocGiao(t *testing.T) {
	for _, body := range []string{
		"Biên bản đối soát phải lập trong vòng 24 giờ, bước 3 kiểm tra và bước 4 ký.",
		"Bước 3 (xem phụ lục) lập biên bản đối soát trong vòng 24 giờ.",
		"Bước ３ lập biên bản đối soát, bước ４ ký trong 2 ngày làm việc.",
		"Bước 3\u00a0lập biên bản đối soát trong vòng 24 giờ.",
		"Bước 5 (khối lượng công việc) lập biên bản đối soát trong vòng 24 giờ.",
	} {
		a := askVoiTaiLieu(t, body, &models.Extractive{})
		r := ask(t, a, user("director", ""), hoiQuyTrinh).Response
		if r.Outcome != contracts.OutcomeResolved || !r.Verdict.Pass {
			t.Errorf("%q: muốn RESOLVED, được [%s] %q (vi phạm %+v)", body, r.Outcome, r.Answer, r.Verdict.Violations)
		}
	}
}

// --- S2-4: câu nêu NHIỀU thực thể / kỳ / con số, hoặc một ngày hỏng, không được cắt thầm về vế đầu (P5) ---

// hoiLai kiểm một lượt phải là câu hỏi lại có kiểu: route CLARIFICATION, outcome CLARIFICATION_REQUIRED, không có
// con số nghiệp vụ nào của một vế lọt ra như thể đó là cả câu trả lời.
func hoiLai(t *testing.T, r contracts.AskResponse, text string) {
	t.Helper()
	if r.Route != contracts.RouteClarification || r.Outcome != contracts.OutcomeClarificationRequired || r.Clarification == nil {
		t.Errorf("%q: muốn CLARIFICATION / CLARIFICATION_REQUIRED, được [%s/%s] %q", text, r.Route, r.Outcome, r.Answer)
		return
	}
	if r.Verdict.Action != contracts.ActionClarify || !r.Verdict.Pass {
		t.Errorf("%q: câu hỏi lại phải có phán quyết clarify/pass, được %+v", text, r.Verdict)
	}
	for _, leaked := range []string{"3.850", "1.200", "5.050", "4.600", "4.000", "Đúng.", "Không."} {
		if strings.Contains(r.Answer, leaked) {
			t.Errorf("%q: câu hỏi lại không được chứa %q (đang trả lời một vế): %q", text, leaked, r.Answer)
		}
	}
}

func candidateIDs(r contracts.AskResponse) []string {
	var ids []string
	if r.Clarification != nil {
		for _, c := range r.Clarification.Candidates {
			ids = append(ids, c.ID)
		}
	}
	slices.Sort(ids)
	return ids
}

// Hai thực thể KHÁC NHAU cùng loại cho khe chỉ nhận một: hỏi lại, ứng viên là các thực thể đã nhận ra.
func TestNhieuThucTheCungLoaiThiHoiLai(t *testing.T) {
	a := newAsk(t)
	for _, c := range []struct {
		text string
		want []string
	}{
		{"EX-03 và RL-02 hôm nay được cấp bao nhiêu dầu?", []string{"EX-03", "RL-02"}},
		{"RL-02 và EX-03 hôm nay được cấp bao nhiêu dầu?", []string{"EX-03", "RL-02"}},
		{"EX-03, RL-02 hôm nay đội báo bao nhiêu dầu?", []string{"EX-03", "RL-02"}},
		{"Dự án A và dự án B hôm nay được cấp bao nhiêu dầu?", []string{"PRJ-A", "PRJ-B"}},
		{"Chênh lệch dầu cấp và tiêu thụ của EX-03 và RL-02 hôm nay?", []string{"EX-03", "RL-02"}},
		// hai máy + dự án chứa chúng: dự án là phần định ngữ, chỉ các máy tranh nhau
		{"EX-03 và RL-02 của dự án A hôm nay được cấp bao nhiêu dầu?", []string{"EX-03", "RL-02"}},
		// một máy rõ + một bí danh mơ hồ khác máy đó ("máy 03" là EX-03 hoặc DZ-03): vẫn là hai lần nhắc
		{"Máy 03 và RL-02 hôm nay được cấp bao nhiêu dầu?", []string{"DZ-03", "EX-03", "RL-02"}},
	} {
		r := ask(t, a, user("director", ""), c.text).Response
		hoiLai(t, r, c.text)
		if got := candidateIDs(r); !slices.Equal(got, c.want) {
			t.Errorf("%q: ứng viên = %v, muốn %v", c.text, got, c.want)
		}
		if !strings.Contains(r.Answer, "máy hoặc dự án") {
			t.Errorf("%q: câu hỏi lại phải nói đang hỏi khe nào (máy hoặc dự án): %q", c.text, r.Answer)
		}
	}
}

// Một thực thể nhắc nhiều lần, hoặc máy kèm dự án chứa nó, vẫn là MỘT chủ thể: hành vi cũ giữ nguyên.
func TestMotChuTheVietNhieuCachVanTraLoi(t *testing.T) {
	a := newAsk(t)
	for _, text := range []string{
		"EX-03 hôm nay được cấp bao nhiêu dầu?",
		"EX-03 (máy xúc 03) hôm nay được cấp bao nhiêu dầu?",
		"EX-03 (máy 03) hôm nay được cấp bao nhiêu dầu?", // "máy 03" mơ hồ nhưng chứa chính EX-03: vẫn một máy
		"EX-03 của dự án A hôm nay được cấp bao nhiêu dầu?",
		"Máy 03 của dự án A hôm nay được cấp bao nhiêu dầu?", // dự án gỡ mơ hồ của bí danh, không phải lần nhắc thứ hai
		"EX-03 hôm nay (19/10/2026) được cấp bao nhiêu dầu?",
	} {
		r := ask(t, a, user("director", ""), text).Response
		if r.Outcome != contracts.OutcomeResolved || !strings.Contains(r.Answer, "3.850 L") {
			t.Errorf("%q: muốn RESOLVED 3.850 L, được [%s] %q", text, r.Outcome, r.Answer)
		}
	}
}

// Người hỏi có phạm vi hẹp nêu hai máy mà chỉ thấy được một: KHÔNG được âm thầm trả lời máy kia (gỡ mơ hồ bằng
// quyền xem chỉ đúng với bí danh mơ hồ, không đúng với thứ người hỏi viết rõ), và không lộ tên máy ngoài phạm vi.
func TestNhieuThucTheNguoiHoiPhamViHep(t *testing.T) {
	a := newAsk(t)
	text := "EX-03 và DZ-03 hôm nay được cấp bao nhiêu dầu?"
	r := ask(t, a, user("site_engineer", "PRJ-A"), text).Response
	hoiLai(t, r, text)
	if got := candidateIDs(r); !slices.Equal(got, []string{"EX-03"}) {
		t.Errorf("ứng viên chỉ được gồm máy trong phạm vi: %v", got)
	}
	if strings.Contains(r.Answer, "DZ-03") || strings.Contains(r.Answer, "Máy ủi") {
		t.Errorf("câu hỏi lại làm lộ máy ngoài phạm vi: %q", r.Answer)
	}
	// vai không có quyền đọc số liệu vẫn bị chặn TRƯỚC khi hỏi lại
	d := ask(t, a, user("driver", "EX-03"), "EX-03 và RL-02 hôm nay được cấp bao nhiêu dầu?").Response
	if d.Route != contracts.RouteReject || d.Outcome != contracts.OutcomePermissionDenied {
		t.Errorf("lái xe: muốn REJECT / PERMISSION_DENIED, được [%s/%s] %q", d.Route, d.Outcome, d.Answer)
	}
}

// Hai kỳ KHÁC NHAU (hai ngày, hai tháng, ngày đầu + ngày cuối của một khoảng) cho khe chỉ nhận một kỳ: hỏi lại và
// nêu các kỳ đã nhận ra; không trả lời riêng kỳ đầu hay kỳ cuối.
func TestNhieuKyThiHoiLai(t *testing.T) {
	a := newAsk(t)
	for _, c := range []struct {
		text     string
		contains []string
	}{
		{"EX-03 hôm nay và hôm qua được cấp bao nhiêu dầu?", []string{"hôm nay", "19/10/2026", "hôm qua", "18/10/2026"}},
		{"EX-03 hôm qua và hôm nay được cấp bao nhiêu dầu?", []string{"hôm nay", "19/10/2026", "hôm qua", "18/10/2026"}},
		{"Dự án A tháng 9 và tháng 10 được cấp bao nhiêu dầu?", []string{"01/09/2026", "30/09/2026", "01/10/2026", "31/10/2026"}},
		// khoảng ngày "từ X đến Y": hai đầu mút là hai kỳ, không ai được đọc riêng ngày đầu
		{"EX-03 từ 10/10 đến 15/10 được cấp bao nhiêu dầu?", []string{"10/10/2026", "15/10/2026"}},
		{"EX-03 từ 10/10/2026 đến 15/10/2026 được cấp bao nhiêu dầu?", []string{"10/10/2026", "15/10/2026"}},
		{"EX-03 từ hôm qua đến hôm nay được cấp bao nhiêu dầu?", []string{"18/10/2026", "19/10/2026"}},
	} {
		r := ask(t, a, user("director", ""), c.text).Response
		hoiLai(t, r, c.text)
		if !strings.Contains(r.Answer, "thời gian") {
			t.Errorf("%q: câu hỏi lại phải nói đang hỏi khe nào (thời gian): %q", c.text, r.Answer)
		}
		for _, s := range c.contains {
			if !strings.Contains(r.Answer, s) {
				t.Errorf("%q: câu hỏi lại thiếu %q: %q", c.text, s, r.Answer)
			}
		}
	}
}

// Cùng một ngày viết hai cách ("hôm nay" và "19/10") vẫn là MỘT kỳ: không hỏi lại thừa.
func TestMotKyVietHaiCachVanTraLoi(t *testing.T) {
	a := newAsk(t)
	for _, text := range []string{
		"EX-03 hôm nay, tức 19/10, được cấp bao nhiêu dầu?",
		"EX-03 hôm nay 19/10/2026 được cấp bao nhiêu dầu?",
	} {
		r := ask(t, a, user("director", ""), text).Response
		if r.Outcome != contracts.OutcomeResolved || !strings.Contains(r.Answer, "3.850 L") {
			t.Errorf("%q: muốn RESOLVED 3.850 L, được [%s] %q", text, r.Outcome, r.Answer)
		}
	}
}

// Ngày có dạng ngày nhưng không có trên lịch (31/02, 32/10, 29/02 năm thường…) KHÔNG được mặc định về hôm nay: hỏi
// lại và nói rõ ngày nào không hợp lệ.
func TestNgayKhongHopLeThiHoiLai(t *testing.T) {
	a := newAsk(t)
	for _, c := range []struct{ text, raw string }{
		{"EX-03 ngày 31/02 được cấp bao nhiêu dầu?", "31/02"},
		{"EX-03 ngày 32/10 được cấp bao nhiêu dầu?", "32/10"},
		{"EX-03 ngày 31/04/2026 được cấp bao nhiêu dầu?", "31/04/2026"},
		{"EX-03 ngày 29/02/2026 được cấp bao nhiêu dầu?", "29/02/2026"},
		{"EX-03 ngày 2026-02-30 được cấp bao nhiêu dầu?", "2026-02-30"},
		{"EX-03 ngày 15/13 được cấp bao nhiêu dầu?", "15/13"},
		{"EX-03 ngày 31/09/26 được cấp bao nhiêu dầu?", "31/09/26"}, // năm hai chữ số là 20yy; tháng 9 không có ngày 31
		{"Dự án A ngày 31/02 được cấp bao nhiêu dầu?", "31/02"},
		{"EX-03 hôm nay và ngày 31/02 được cấp bao nhiêu dầu?", "31/02"},
	} {
		r := ask(t, a, user("director", ""), c.text).Response
		hoiLai(t, r, c.text)
		if !strings.Contains(r.Answer, c.raw+" không hợp lệ") {
			t.Errorf("%q: câu hỏi lại phải nói rõ %q không hợp lệ: %q", c.text, c.raw, r.Answer)
		}
	}
	// ngày hợp lệ viết theo các dạng khác (dd/mm/yyyy, ISO, dd-mm-yyyy, dd/mm) vẫn trả lời
	for _, text := range []string{
		"EX-03 ngày 19/10/2026 được cấp bao nhiêu dầu?",
		"EX-03 ngày 2026-10-19 được cấp bao nhiêu dầu?",
		"EX-03 ngày 19-10-2026 được cấp bao nhiêu dầu?",
		"EX-03 ngày 19/10 được cấp bao nhiêu dầu?",
	} {
		if r := ask(t, a, user("director", ""), text).Response; r.Outcome != contracts.OutcomeResolved || !strings.Contains(r.Answer, "3.850 L") {
			t.Errorf("%q: muốn RESOLVED 3.850 L, được [%s] %q", text, r.Outcome, r.Answer)
		}
	}
}

// Câu xác nhận nêu ≥ 2 con số KHÁC NHAU (ở mệnh đề hỏi hoặc các mệnh đề sau nó): không xác nhận con số nào thay
// người hỏi. Số của các mệnh đề đứng TRƯỚC mệnh đề hỏi là ngữ cảnh, vẫn không tính.
func TestCauXacNhanNhieuConSoThiHoiLai(t *testing.T) {
	a := newAsk(t)
	for _, text := range []string{
		"EX-03 hôm nay được cấp 3850 L đúng không, 4000 L thì sao?",
		"EX-03 hôm nay được cấp 3850 L và 4000 L đúng không?",
		"EX-03 hôm nay được cấp 3850 L đúng không, hay 4k?",
		"EX-03 hôm nay được cấp 3850 đúng không, 4000 thì sao?",
		"EX-03 hôm nay được cấp 3850 lít đúng không? Còn 4 m3 thì sao?",
	} {
		r := ask(t, a, user("director", ""), text).Response
		hoiLai(t, r, text)
		if r.Clarification == nil || !slices.Contains(r.Clarification.Missing, "claimed") || !strings.Contains(r.Answer, "con số cần xác nhận") {
			t.Errorf("%q: câu hỏi lại phải nói đang hỏi con số cần xác nhận: %+v %q", text, r.Clarification, r.Answer)
		}
	}
}

// Câu xác nhận chỉ MỘT con số (kể cả số được nhắc lại cùng giá trị, hay số ngữ cảnh ở mệnh đề trước) vẫn ra "Đúng."/"Không.".
func TestCauXacNhanMotConSoVanTraLoi(t *testing.T) {
	a := newAsk(t)
	for _, c := range []struct{ text, prefix string }{
		{"EX-03 hôm nay được cấp 3850 L đúng không?", "Đúng."},
		{"EX-03 hôm nay được cấp 4000 L đúng không?", "Không."},
		// cùng giá trị 4000 L viết hai cách (kể cả khác đơn vị): không phải hai con số khác nhau
		{"EX-03 hôm nay được cấp 4k đúng không, 4000 lít thì sao?", "Không."},
		{"EX-03 hôm nay được cấp 4k đúng không, 4 m3 thì sao?", "Không."},
		// số trong TÊN thực thể ("03" của "máy xúc 03") không phải con số xác nhận
		{"Máy xúc 03 hôm nay được cấp 3850 L đúng không?", "Đúng."},
		// số ở mệnh đề ĐỨNG TRƯỚC mệnh đề hỏi là ngữ cảnh
		{"EX-03 hôm nay đội báo 3k, báo cáo ghi 5 m3, vậy được cấp 3850 L đúng không?", "Đúng."},
		{"EX-03 hôm nay đội báo 3k, vậy được cấp 4000 L đúng không?", "Không."},
	} {
		r := ask(t, a, user("director", ""), c.text).Response
		if r.Outcome != contracts.OutcomeResolved || !strings.HasPrefix(r.Answer, c.prefix) {
			t.Errorf("%q: muốn RESOLVED %q…, được [%s] %q", c.text, c.prefix, r.Outcome, r.Answer)
		}
	}
}

// Câu hỏi lại vì nhiều giá trị / ngày hỏng phải để lại vết quyết định đủ để truy (P8) và phát lại ra đúng từng chặng (G6):
// khe không rõ ghi "<khe>=?", thực thể đã nhận ra có mặt, ngày hỏng nằm trong vết chuẩn hoá.
func TestHoiLaiNhieuGiaTriDeLaiVetVaPhatLaiDuoc(t *testing.T) {
	a := newAsk(t)
	for _, c := range []struct {
		text string
		want []string // phải có trong vết: Entities, Normalized hoặc Slots
	}{
		{"EX-03 và RL-02 hôm nay được cấp bao nhiêu dầu?", []string{"EX-03", "RL-02", "subject=?"}},
		{"EX-03 hôm nay và hôm qua được cấp bao nhiêu dầu?", []string{"date:hôm nay=2026-10-19", "date:hôm qua=2026-10-18", "period=?"}},
		{"EX-03 ngày 31/02 được cấp bao nhiêu dầu?", []string{"date:31/02=invalid", "period=?"}},
		{"EX-03 hôm nay được cấp 3850 L đúng không, 4000 L thì sao?", []string{"claimed=?"}},
	} {
		out := ask(t, a, user("director", ""), c.text)
		trace := strings.Join(append(append(slices.Clone(out.Decision.Entities), out.Decision.Normalized...), out.Decision.Slots...), " | ")
		for _, w := range c.want {
			if !strings.Contains(trace, w) {
				t.Errorf("%q: vết quyết định thiếu %q: %s", c.text, w, trace)
			}
		}
		if out.Decision.Outcome != string(contracts.OutcomeClarificationRequired) {
			t.Errorf("%q: vết ghi outcome %q", c.text, out.Decision.Outcome)
		}
		if diff, err := a.Replay(context.Background(), out.Decision); err != nil || len(diff) != 0 {
			t.Errorf("%q: phát lại phải tái lập đúng, được lệch %v lỗi %v", c.text, diff, err)
		}
	}
}

// Con số khác chiều đo ("7h" là giờ chạy, không phải lượng dầu) không phải con số xác nhận thứ hai: không hỏi lại thừa.
// Chỉ còn MỘT lượng cùng chiều đo (3850 L) để so, nên đó là con số xác nhận — cùng định nghĩa với rivalClaims, xem pipeline_hoi_quy_test.go.
func TestCauXacNhanLanSoKhacChieuKhongHoiLai(t *testing.T) {
	r := ask(t, newAsk(t), user("director", ""), "EX-03 chạy 7h hôm nay được cấp 3850 L đúng không?").Response
	if r.Route != contracts.RouteSQLQuery || r.Outcome != contracts.OutcomeResolved || !strings.Contains(r.Answer, "3.850 L") {
		t.Errorf("muốn SQL_QUERY / RESOLVED 3.850 L, được [%s/%s] %q", r.Route, r.Outcome, r.Answer)
	}
}
