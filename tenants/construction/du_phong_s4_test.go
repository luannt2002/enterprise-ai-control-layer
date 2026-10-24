package construction_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/luannt2002/enterprise-ai-control-layer/core/contracts"
	"github.com/luannt2002/enterprise-ai-control-layer/core/pipeline"
	"github.com/luannt2002/enterprise-ai-control-layer/intelligence/intent"
	"github.com/luannt2002/enterprise-ai-control-layer/intelligence/rag"
)

// Dự phòng S4 (B1, tất định): câu quy trình mà luật không nhận ra ý định (không khớp / khớp nhầm ý định hoá đơn rồi thiếu thuật ngữ) được
// trả lời từ tài liệu, vết ghi đúng nguồn quyết định; câu ngoài kho vẫn từ chối; câu số liệu — kể cả không có "bao nhiêu" (soát độc lập
// 24/10 tái lập 9/9 câu kiểu này bị trả lời bằng đoạn quy trình qua nhánh "ý định số liệu thiếu khe") — không bao giờ đổi kết quả.
// Câu quy trình khớp nhầm ý định số liệu rồi thiếu máy ("lech dau … may tieng") cũng giữ hỏi lại: luật không tách được nó với câu số
// liệu thiếu máy.
func TestDuPhongS4DoChiMuc(t *testing.T) {
	a, _ := newAskPhienBan(t, sopBan(2, rag.StatusApproved, "2026-03-01", "", "48 giờ"))
	tat := map[string]contracts.Outcome{}
	cau := []string{"Kế toán đối chiếu hoá đơn lúc nào?", "Tổ trưởng kho phải đối soát khi nào?",
		// "mấy tiếng" hỏi THỜI HẠN (đơn vị chiều thời lượng của lexicon), không phải lượng nghiệp vụ — gộp 24/10
		"Tổ trưởng kho cần đối soát trong mấy tiếng?",
		"Thời tiết mai thế nào?", "lech dau thi to truong kho doi soat trong may tieng",
		"Hôm qua cấp bao nhiêu dầu?", "EX-03 hôm qua hết bao nhiêu dầu?",
		"xuất kho bao nhiêu lít?", "tiêu thụ đo được hôm qua bao nhiêu lít?", "Hôm qua lượng xuất kho lệch bao nhiêu?", "xuat kho bao nhieu lit?",
		"Hôm qua kho xuất mấy lít dầu?", "Tổng dầu đã cấp hôm qua là mấy?", "dau kho xuat hom qua het may lit?", "Lượng dầu tiêu thụ đo được tuần trước?",
		"Cho xin số dầu đội báo hôm qua", "Hôm qua kho xuất bao nhiu lít dầu?", "Hôm qua kho xuất bn lít dầu?", "Hôm qua dầu nhiều không?",
		"Số lít dầu kho xuất hôm qua?"}
	for _, q := range cau {
		tat[q] = ask(t, a, user("director", ""), q).Response.Outcome
	}
	if tat[cau[0]] != contracts.OutcomeClarificationRequired || tat[cau[1]] != contracts.OutcomeOutOfScope || tat[cau[2]] != contracts.OutcomeOutOfScope {
		t.Fatalf("mốc B0 đổi (test giả định luật hiện tại): %v", tat)
	}
	a.Deps.IntentFallback = pipeline.KnowledgeProbe{Knowledge: a.Deps.Knowledge, Name: "sop_question"}
	for _, q := range cau[:3] {
		o := ask(t, a, user("director", ""), q)
		if r := o.Response; r.Outcome != contracts.OutcomeResolved || r.Route != contracts.RouteRAG || len(r.Citations) == 0 {
			t.Errorf("%q: muốn trả lời từ tài liệu, được %s %s %q", q, r.Route, r.Outcome, r.Answer)
		}
		if !strings.Contains(strings.Join(o.Decision.Why.Intent.Reasons, " | "), "dự phòng S4 → sop_question") || !strings.Contains(o.Plan.Reason, "do dự phòng S4 chọn") {
			t.Errorf("%q: vết S4 / S5 phải ghi dự phòng: %+v · %q", q, o.Decision.Why.Intent, o.Plan.Reason)
		}
	}
	for _, q := range cau[3:] {
		if got := ask(t, a, user("director", ""), q).Response.Outcome; got != tat[q] {
			t.Errorf("%q: dự phòng không được đổi kết quả (ngoài kho / câu số liệu): %s → %s", q, tat[q], got)
		}
	}
}

// B1 gặp tài liệu khớp mà phải báo phiên bản (hai bản cùng hiệu lực): đi route tri thức để người hỏi nhận thông báo xung đột, không bị
// coi là lỗi của dự phòng rồi từ chối "ngoài phạm vi" (soát độc lập 24/10).
func TestDuPhongS4ThongBaoPhienBan(t *testing.T) {
	a, m := newAskPhienBan(t, sopBan(1, rag.StatusApproved, "2025-01-01", "", "24 giờ"), sopBan(2, rag.StatusApproved, "2026-03-01", "", "48 giờ"))
	q := "Tổ trưởng kho phải đối soát khi nào?"
	if r := ask(t, a, user("director", ""), q).Response; r.Outcome != contracts.OutcomeOutOfScope {
		t.Fatalf("mốc B0: muốn OUT_OF_SCOPE, được %s", r.Outcome)
	}
	a.Deps.IntentFallback = pipeline.KnowledgeProbe{Knowledge: a.Deps.Knowledge, Name: "sop_question"}
	if r := ask(t, a, user("director", ""), q).Response; r.Outcome != contracts.OutcomeSourceConflict || m.Calls() != 0 {
		t.Errorf("B1 + hai bản cùng hiệu lực: muốn SOURCE_CONFLICT, 0 lời gọi model; được %s, %d → %q", r.Outcome, m.Calls(), r.Answer)
	}
}

// Móc S4: người hỏi thiếu capability đọc tài liệu thì dự phòng không được hỏi; dự phòng lỗi thì giữ kết quả của luật và ghi vết; dự
// phòng khai ý định không phải tri thức là lỗi cấu hình.
func TestDuPhongS4Moc(t *testing.T) {
	a, _ := newAskPhienBan(t, sopBan(2, rag.StatusApproved, "2026-03-01", "", "48 giờ"))
	q := "Tổ trưởng kho phải đối soát khi nào?"
	do := &duPhongDem{ten: "sop_question", dung: true}
	a.Deps.IntentFallback = do
	if r := ask(t, a, user("accountant", ""), q).Response; do.lan != 0 || r.Outcome != contracts.OutcomeOutOfScope {
		t.Errorf("thiếu documents.read: dự phòng không được hỏi (%d lần), muốn OUT_OF_SCOPE, được %s", do.lan, r.Outcome)
	}
	// Câu hỏi lượng — hỏi "bao nhiêu" hay xác nhận một con số có đơn vị — không bao giờ tới dự phòng (bộ độc lập S4 24/10: S4M-46).
	for _, qq := range []string{"xuất kho bao nhiêu lít?", "Hôm qua máy 03 đổ 4 khối dầu đúng không?", "xuat kho bao nhieu lit?", "EX-03 hôm qua nhận 3850 L phải không?"} {
		ask(t, a, user("director", ""), qq)
	}
	if do.lan != 0 {
		t.Errorf("câu hỏi lượng không được hỏi dự phòng, đã hỏi %d lần", do.lan)
	}
	do.loi = contracts.Label(contracts.ThirdParty("decide", contracts.KindRateLimit), errors.New("429"))
	o := ask(t, a, user("director", ""), q)
	if do.lan != 1 || o.Response.Outcome != contracts.OutcomeOutOfScope ||
		!strings.Contains(strings.Join(o.Decision.Why.Intent.Reasons, " | "), "dự phòng S4 lỗi (third_party:decide:rate_limit) — giữ kết quả của luật") {
		t.Errorf("dự phòng lỗi: muốn giữ OUT_OF_SCOPE và ghi vết, được %s %+v", o.Response.Outcome, o.Decision.Why.Intent)
	}
	// Lỗi KHÔNG phải của bên thứ ba (hạ tầng, dữ liệu, mã): hỏng lượt như route tri thức, không che thành "giữ kết quả của luật".
	a.Deps.IntentFallback = &duPhongDem{ten: "sop_question", loi: contracts.Label(contracts.Infra("postgres"), errors.New("mất kết nối"))}
	if _, err := a.Handle(context.Background(), user("director", ""), contracts.AskRequest{ConversationID: "c", MessageID: "m", Text: q}); contracts.FaultOf(err) != contracts.Infra("postgres") {
		t.Errorf("lỗi hạ tầng của dự phòng phải làm hỏng lượt, được %v", err)
	}
	a.Deps.IntentFallback = &duPhongDem{ten: "fuel_generic", dung: true}
	if _, err := a.Handle(context.Background(), user("director", ""), contracts.AskRequest{ConversationID: "c", MessageID: "m", Text: q}); contracts.FaultOf(err) != contracts.FaultCode || !strings.Contains(err.Error(), "không phải ý định tri thức") {
		t.Errorf("khai ý định số liệu phải là lỗi cấu hình, được %v", err)
	}
}

type duPhongDem struct {
	ten  string
	dung bool
	loi  error
	lan  int
}

func (d *duPhongDem) Intent() string { return d.ten }

func (d *duPhongDem) Fallback(context.Context, contracts.Principal, contracts.NormalizedInput, intent.Result) (bool, float64, string, error) {
	d.lan++
	return d.dung, 0.9, "giả", d.loi
}
