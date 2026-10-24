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

// Dự phòng S4 (B1, tất định): câu quy trình mà luật không ra kế hoạch trả lời (không khớp / khớp nhầm ý định hoá đơn rồi thiếu thuật ngữ /
// khớp nhầm ý định số liệu rồi thiếu máy)
// được trả lời từ tài liệu; câu ngoài kho vẫn từ chối; câu đòi LƯỢNG không bao giờ bị đẩy sang tài liệu; tắt (nil) thì như cũ.
func TestDuPhongS4DoChiMuc(t *testing.T) {
	a, _ := newAskPhienBan(t, sopBan(2, rag.StatusApproved, "2026-03-01", "", "48 giờ"))
	tat := map[string]contracts.Outcome{}
	cau := []string{"Kế toán đối chiếu hoá đơn lúc nào?", "Tổ trưởng kho cần đối soát trong mấy tiếng?", "lech dau thi to truong kho doi soat trong may tieng",
		"Thời tiết mai thế nào?",
		"Hôm qua cấp bao nhiêu dầu?", "EX-03 hôm qua hết bao nhiêu dầu?",
		// câu số liệu "kẹt" (khớp ý định số liệu mà thiếu máy / dự án) mà chỉ mục lại có đoạn: B1 không chốt thì trả lời bằng đoạn quy trình
		"xuất kho bao nhiêu lít?", "tiêu thụ đo được hôm qua bao nhiêu lít?", "Hôm qua lượng xuất kho lệch bao nhiêu?", "xuat kho bao nhieu lit?"}
	for _, q := range cau {
		tat[q] = ask(t, a, user("director", ""), q).Response.Outcome
	}
	if tat[cau[0]] != contracts.OutcomeClarificationRequired || tat[cau[1]] != contracts.OutcomeOutOfScope || tat[cau[2]] != contracts.OutcomeClarificationRequired {
		t.Fatalf("mốc B0 đổi (test giả định luật hiện tại): %v", tat)
	}
	a.Deps.IntentFallback = pipeline.KnowledgeProbe{Knowledge: a.Deps.Knowledge, Name: "sop_question"}
	for _, q := range cau[:3] {
		o := ask(t, a, user("director", ""), q)
		if r := o.Response; r.Outcome != contracts.OutcomeResolved || r.Route != contracts.RouteRAG || len(r.Citations) == 0 {
			t.Errorf("%q: muốn trả lời từ tài liệu, được %s %s %q", q, r.Route, r.Outcome, r.Answer)
		}
		if !strings.Contains(strings.Join(o.Decision.Why.Intent.Reasons, " | "), "dự phòng S4 → sop_question") {
			t.Errorf("%q: vết S4 phải ghi dự phòng: %+v", q, o.Decision.Why.Intent)
		}
	}
	for _, q := range cau[3:] {
		if got := ask(t, a, user("director", ""), q).Response.Outcome; got != tat[q] {
			t.Errorf("%q: dự phòng không được đổi kết quả (ngoài kho / đòi lượng): %s → %s", q, tat[q], got)
		}
	}
}

// Móc S4: người hỏi thiếu capability đọc tài liệu thì dự phòng không được hỏi; dự phòng lỗi thì giữ kết quả của luật và ghi vết; dự
// phòng khai ý định không phải tri thức là lỗi cấu hình.
func TestDuPhongS4Moc(t *testing.T) {
	a, _ := newAskPhienBan(t, sopBan(2, rag.StatusApproved, "2026-03-01", "", "48 giờ"))
	q := "Tổ trưởng kho cần đối soát trong mấy tiếng?"
	do := &duPhongDem{ten: "sop_question", dung: true}
	a.Deps.IntentFallback = do
	if r := ask(t, a, user("accountant", ""), q).Response; do.lan != 0 || r.Outcome != contracts.OutcomeOutOfScope {
		t.Errorf("thiếu documents.read: dự phòng không được hỏi (%d lần), muốn OUT_OF_SCOPE, được %s", do.lan, r.Outcome)
	}
	do.loi = contracts.Label(contracts.ThirdParty("decide", contracts.KindRateLimit), errors.New("429"))
	o := ask(t, a, user("director", ""), q)
	if do.lan != 1 || o.Response.Outcome != contracts.OutcomeOutOfScope ||
		!strings.Contains(strings.Join(o.Decision.Why.Intent.Reasons, " | "), "dự phòng S4 lỗi (third_party:decide:rate_limit) — giữ kết quả của luật") {
		t.Errorf("dự phòng lỗi: muốn giữ OUT_OF_SCOPE và ghi vết, được %s %+v", o.Response.Outcome, o.Decision.Why.Intent)
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
