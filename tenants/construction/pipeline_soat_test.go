// Test đầu–cuối cho các lỗi do người soát độc lập tìm ra ở luồng hỏi (S2-1, S2-2, S2-3, M42): pack
// THẬT, danh mục + fact giả lập, model tất định. Mỗi ca ghi một hành vi nghiệp vụ, không ghi cách
// cài đặt.
package construction_test

import (
	"context"
	"strings"
	"testing"

	"github.com/luannt2002/enterprise-ai-systems/core/contracts"
	"github.com/luannt2002/enterprise-ai-systems/core/pipeline"
	"github.com/luannt2002/enterprise-ai-systems/intelligence/models"
	"github.com/luannt2002/enterprise-ai-systems/intelligence/rag"
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
