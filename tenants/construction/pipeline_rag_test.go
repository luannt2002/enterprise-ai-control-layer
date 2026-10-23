package construction_test

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/luannt2002/enterprise-ai-control-layer/core/contracts"
	"github.com/luannt2002/enterprise-ai-control-layer/core/pipeline"
	"github.com/luannt2002/enterprise-ai-control-layer/intelligence/models"
	"github.com/luannt2002/enterprise-ai-control-layer/intelligence/rag"
)

// Nhánh RAG (docs/viec/2-tra-quy-trinh, KE-HOACH U7a): tài liệu có phiên bản đi tới câu trả lời thật qua Ask.
// SOP-FUEL-01 đổi hạn lập biên bản từ 24 giờ (v1) sang 48 giờ (v2) — giả lập.

func sopBan(v int, status, from, to, han string) rag.Document {
	return rag.Document{ID: "SOP-FUEL-01", Tenant: "abc-infra", Title: "Quy trình xử lý chênh lệch nhiên liệu",
		ACL: []string{"director", "site_engineer"}, Version: v, Status: status, EffectiveFrom: from, EffectiveTo: to,
		Body: "1. Khi lượng xuất kho và lượng tiêu thụ đo được lệch quá dung sai, tổ trưởng kho lập biên bản đối soát trong vòng " + han + ".\n" +
			"2. Kế toán chỉ đối chiếu hoá đơn sau khi biên bản đối soát được ký."}
}

func newAskPhienBan(t *testing.T, docs ...rag.Document) (*pipeline.Ask, *models.Extractive) {
	t.Helper()
	a, m := newAsk(t), &models.Extractive{}
	ix := rag.NewMemoryIndex("abc-infra", docs)
	ix.Now = func() time.Time { return time.Date(2026, 10, 20, 10, 0, 0, 0, time.FixedZone("ICT", 7*3600)) }
	a.Deps.Knowledge, a.Deps.Model = ix, m
	return a, m
}

const cauQuyTrinh = "Quy trình xử lý chênh lệch nhiên liệu là gì?"
const cauGhep = "EX-03 hôm nay được cấp bao nhiêu dầu, và theo quy trình thì khi lệch phải làm gì?"

// Bản đang hiệu lực được trích, kèm số bản trong trích dẫn; bản hết hiệu lực không lọt.
func TestRAGPhienBanTraBanHieuLuc(t *testing.T) {
	a, m := newAskPhienBan(t, sopBan(1, rag.StatusApproved, "2025-01-01", "2026-03-01", "24 giờ"), sopBan(2, rag.StatusApproved, "2026-03-01", "", "48 giờ"))
	r := ask(t, a, user("director", ""), cauQuyTrinh).Response
	if r.Outcome != contracts.OutcomeResolved || !strings.Contains(r.Answer, "48 giờ") || strings.Contains(r.Answer, "24 giờ") || m.Calls() != 1 {
		t.Errorf("muốn trích v2 (48 giờ), được %s → %q (model %d)", r.Outcome, r.Answer, m.Calls())
	}
	if len(r.Citations) == 0 || r.Citations[0].Source.DocumentVersion != 2 {
		t.Errorf("trích dẫn phải mang bản 2: %+v", r.Citations)
	}
}

// Hai bản cùng hiệu lực (BR-DOC-01, CON-T-061): nói rõ cho người hỏi, không chọn bản nào, không gọi model, không
// hứa đã chuyển người.
func TestRAGPhienBanXungDot(t *testing.T) {
	a, m := newAskPhienBan(t, sopBan(1, rag.StatusApproved, "2025-01-01", "", "24 giờ"), sopBan(2, rag.StatusApproved, "2026-03-01", "", "48 giờ"))
	r := ask(t, a, user("director", ""), cauQuyTrinh).Response
	if r.Outcome != contracts.OutcomeSourceConflict || m.Calls() != 0 {
		t.Errorf("muốn SOURCE_CONFLICT, 0 lời gọi model; được %s, %d → %q", r.Outcome, m.Calls(), r.Answer)
	}
	for _, s := range []string{"Quy trình xử lý chênh lệch nhiên liệu", "v1, v2", "xác nhận"} {
		if !strings.Contains(r.Answer, s) {
			t.Errorf("thiếu %q trong %q", s, r.Answer)
		}
	}
	for _, s := range []string{"24 giờ", "48 giờ", "đã chuyển"} {
		if strings.Contains(r.Answer, s) {
			t.Errorf("không được có %q trong %q", s, r.Answer)
		}
	}
}

// Không bản nào đã duyệt và đang hiệu lực (CON-T-032): nói rõ, không dùng bản cũ, không gọi model.
func TestRAGPhienBanKhongCoBanHieuLuc(t *testing.T) {
	a, m := newAskPhienBan(t, sopBan(1, rag.StatusApproved, "2025-01-01", "2026-03-01", "24 giờ"), sopBan(2, rag.StatusNeedsReview, "2026-03-01", "", "48 giờ"))
	r := ask(t, a, user("director", ""), cauQuyTrinh).Response
	if r.Outcome != contracts.OutcomeInsufficientEvidence || m.Calls() != 0 || !strings.Contains(r.Answer, "v1, v2") || strings.Contains(r.Answer, "24 giờ") {
		t.Errorf("muốn INSUFFICIENT_EVIDENCE nêu v1, v2, không trích bản cũ; được %s → %q", r.Outcome, r.Answer)
	}
}

// HYBRID: phần số liệu đã kiểm vẫn giao khi phần quy trình gặp thông báo phiên bản hoặc lỗi truy xuất (trước U7a
// lỗi truy xuất làm mất luôn phần số).
func TestRAGHybridGiuPhanSo(t *testing.T) {
	a, m := newAskPhienBan(t, sopBan(1, rag.StatusApproved, "2025-01-01", "", "24 giờ"), sopBan(2, rag.StatusApproved, "2026-03-01", "", "48 giờ"))
	r := ask(t, a, user("director", ""), cauGhep).Response
	if r.Route != contracts.RouteHybrid || r.Outcome != contracts.OutcomeResolved || !strings.Contains(r.Answer, "3.850") || !strings.Contains(r.Answer, "xác nhận") || m.Calls() != 0 {
		t.Errorf("HYBRID + xung đột: muốn số 3.850 + câu xung đột, 0 model; được %s/%s → %q", r.Route, r.Outcome, r.Answer)
	}
	a.Deps.Knowledge = loiTruyXuat{}
	r = ask(t, a, user("director", ""), cauGhep).Response
	if r.Outcome != contracts.OutcomeResolved || !strings.Contains(r.Answer, "3.850") || !strings.Contains(r.Answer, "chưa tra được") {
		t.Errorf("HYBRID + lỗi truy xuất: muốn giữ 3.850 và báo phần quy trình chưa tra được, được %s → %q", r.Outcome, r.Answer)
	}
	if strings.Contains(r.Answer, "Chưa có tài liệu") {
		t.Errorf("lỗi hệ thống không được giả làm 'không có tài liệu': %q", r.Answer)
	}
	// Câu chỉ hỏi quy trình mà truy xuất lỗi: vẫn là lỗi của lượt, không giả vờ trả lời.
	if _, err := a.Handle(context.Background(), user("director", ""), contracts.AskRequest{ConversationID: "c", MessageID: "m", Text: cauQuyTrinh}); err == nil {
		t.Errorf("RAG thuần + lỗi truy xuất: muốn lỗi")
	}
}

type loiTruyXuat struct{}

func (loiTruyXuat) Retrieve(context.Context, contracts.Principal, string, int) ([]contracts.Evidence, error) {
	return nil, errors.New("chỉ mục không trả lời")
}

// HYBRID: model lỗi khi diễn đạt phần quy trình thì phần số liệu đã kiểm vẫn giao; câu chỉ hỏi quy trình thì vẫn là lỗi.
func TestRAGHybridModelLoi(t *testing.T) {
	a, _ := newAskPhienBan(t, sopBan(2, rag.StatusApproved, "2026-03-01", "", "48 giờ"))
	a.Deps.Model = modelLoi{}
	r := ask(t, a, user("director", ""), cauGhep).Response
	if r.Outcome != contracts.OutcomeResolved || !strings.Contains(r.Answer, "3.850") || !strings.Contains(r.Answer, "chưa tra được") {
		t.Errorf("HYBRID + model lỗi: muốn giữ 3.850, được %s → %q", r.Outcome, r.Answer)
	}
	if _, err := a.Handle(context.Background(), user("director", ""), contracts.AskRequest{ConversationID: "c", MessageID: "m", Text: cauQuyTrinh}); err == nil {
		t.Errorf("RAG thuần + model lỗi: muốn lỗi")
	}
}

// Model nói bằng chứng không trả lời được (answer rỗng — đường của model thật, ROADMAP-RAG §6.5 "không có thì nói không có"): người
// hỏi nhận "chưa có tài liệu trả lời được", không phải lỗi hệ thống; không trích dẫn. HYBRID: phần số liệu đã kiểm vẫn giao.
func TestRAGModelTuChoi(t *testing.T) {
	a, _ := newAskPhienBan(t, sopBan(2, rag.StatusApproved, "2026-03-01", "", "48 giờ"))
	a.Deps.Model = modelTuChoi{}
	r := ask(t, a, user("director", ""), cauQuyTrinh).Response
	if r.Outcome != contracts.OutcomeInsufficientEvidence || len(r.Citations) != 0 || strings.Contains(r.Answer, "48 giờ") {
		t.Errorf("model từ chối: muốn INSUFFICIENT_EVIDENCE, không trích dẫn; được %s → %q %+v", r.Outcome, r.Answer, r.Citations)
	}
	r = ask(t, a, user("director", ""), cauGhep).Response
	if r.Outcome != contracts.OutcomeResolved || !strings.Contains(r.Answer, "3.850") || strings.Contains(r.Answer, "48 giờ") {
		t.Errorf("HYBRID + model từ chối: muốn giữ 3.850, bỏ phần quy trình; được %s → %q", r.Outcome, r.Answer)
	}
	// Văn xuôi không claim, và bản nháp RỖNG không mang cờ Refused (model trả sai khuôn), là đầu ra hỏng ⇒ S9 chặn như cũ — không được
	// hiểu là "không có" (soát độc lập 23/10).
	for name, m := range map[string]models.Gateway{"văn xuôi": modelVanXuoi{}, "rỗng không cờ": modelRong{}} {
		a.Deps.Model = m
		if r := ask(t, a, user("director", ""), cauQuyTrinh).Response; r.Outcome != contracts.OutcomeSystemFailure {
			t.Errorf("%s: muốn S9 chặn (SYSTEM_FAILURE), được %s → %q", name, r.Outcome, r.Answer)
		}
	}
}

type modelRong struct{}

func (modelRong) Generate(context.Context, models.Request) (contracts.Draft, error) {
	return contracts.Draft{Model: "gia"}, nil
}

type modelTuChoi struct{}

func (modelTuChoi) Generate(context.Context, models.Request) (contracts.Draft, error) {
	return contracts.Draft{Model: "gia", Refused: true}, nil
}

type modelVanXuoi struct{}

func (modelVanXuoi) Generate(context.Context, models.Request) (contracts.Draft, error) {
	return contracts.Draft{Text: "Lập biên bản trong 48 giờ.", Model: "gia"}, nil
}

type modelLoi struct{}

func (modelLoi) Generate(context.Context, models.Request) (contracts.Draft, error) {
	return contracts.Draft{}, errors.New("model hết giờ")
}

// U6: ngày hỏi và luật phạm vi đến từ LƯỢT (pipeline đặt vào ngữ cảnh), không từ đồng hồ máy hay thuộc tính trần — để chỉ
// mục nào (bộ nhớ, PG) cũng xét như nhau, và phát lại một lượt cũ ra đúng bản của ngày đó.
func TestRAGNgayHoiVaPhamViTuLuot(t *testing.T) {
	a, m := newAsk(t), &models.Extractive{}
	ban1 := sopBan(1, rag.StatusApproved, "2025-01-01", "2026-03-01", "24 giờ")
	ban2 := sopBan(2, rag.StatusApproved, "2026-03-01", "", "48 giờ")
	ban1.Scope, ban2.Scope = []string{"PRJ-A"}, []string{"PRJ-A"}
	ix := rag.NewMemoryIndex("abc-infra", []rag.Document{ban1, ban2})
	ix.Now = func() time.Time { return time.Date(2026, 10, 20, 0, 0, 0, 0, time.UTC) } // đồng hồ riêng của chỉ mục (⇒ v2): lượt phải thắng
	ix.InScope = func(contracts.Principal, []string) bool { return false }             // luật riêng của chỉ mục (chặn hết): luật của lượt phải thắng
	a.Deps.Knowledge, a.Deps.Model = ix, m
	a.Deps.Now = func() time.Time { return time.Date(2025, 6, 1, 9, 0, 0, 0, time.FixedZone("ICT", 7*3600)) }

	// director có scope "all" theo policy, không mang thuộc tính entity_scope: vẫn thấy tài liệu của PRJ-A; ngày hỏi 01/06/2025 ⇒ v1.
	if r := ask(t, a, user("director", ""), cauQuyTrinh).Response; !strings.Contains(r.Answer, "24 giờ") || strings.Contains(r.Answer, "48 giờ") {
		t.Errorf("director ngày 01/06/2025: muốn trích v1 (24 giờ), được %s → %q", r.Outcome, r.Answer)
	}
	if r := ask(t, a, user("site_engineer", "PRJ-A"), cauQuyTrinh).Response; !strings.Contains(r.Answer, "24 giờ") {
		t.Errorf("site_engineer phạm vi PRJ-A: muốn trích v1, được %s → %q", r.Outcome, r.Answer)
	}
	// Ngoài phạm vi: như không có tài liệu, không lộ gì của nó.
	if r := ask(t, a, user("site_engineer", "PRJ-B"), cauQuyTrinh).Response; r.Outcome == contracts.OutcomeResolved || strings.Contains(r.Answer, "giờ") {
		t.Errorf("site_engineer phạm vi PRJ-B không được thấy tài liệu PRJ-A: %s → %q", r.Outcome, r.Answer)
	}
}

// Soát độc lập U6a: (1) vai được đọc tài liệu (ACL) và phạm vi phải đến từ CÙNG một vai — accountant có scope "all" cho số
// liệu nhưng không đọc được tài liệu, không cho site_engineer mượn phạm vi đó; (2) ngày hỏi theo GIỜ TENANT (20:00 UTC ngày
// 28/02 là 03:00 ngày 01/03 ở ICT ⇒ v2); (3) phát lại một lượt tri thức dùng ngày đã ghi, không ngày lúc phát lại.
func TestRAGPhamViCungVaiNgayTenantPhatLai(t *testing.T) {
	a, m := newAsk(t), &models.Extractive{}
	ban1 := sopBan(1, rag.StatusApproved, "2025-01-01", "2026-03-01", "24 giờ")
	ban2 := sopBan(2, rag.StatusApproved, "2026-03-01", "", "48 giờ")
	ban1.Scope, ban2.Scope = []string{"PRJ-A"}, []string{"PRJ-A"}
	a.Deps.Knowledge, a.Deps.Model = rag.NewMemoryIndex("abc-infra", []rag.Document{ban1, ban2}), m

	a.Deps.Now = func() time.Time { return time.Date(2025, 6, 1, 9, 0, 0, 0, time.UTC) }
	haiVai := contracts.Principal{ID: "u-hai-vai", TenantID: "abc-infra", Kind: contracts.PrincipalUser,
		Roles: []string{"accountant", "site_engineer"}, Attributes: map[string]string{"entity_scope": "PRJ-B"}}
	if r := ask(t, a, haiVai, cauQuyTrinh).Response; r.Outcome == contracts.OutcomeResolved || strings.Contains(r.Answer, "giờ") {
		t.Errorf("accountant + site_engineer phạm vi PRJ-B thấy tài liệu PRJ-A nhờ mượn scope của accountant: %s → %q", r.Outcome, r.Answer)
	}

	a.Deps.Now = func() time.Time { return time.Date(2026, 2, 28, 20, 0, 0, 0, time.UTC) }
	if r := ask(t, a, user("director", ""), cauQuyTrinh).Response; !strings.Contains(r.Answer, "48 giờ") {
		t.Errorf("20:00 UTC 28/02 là 01/03 ở ICT: muốn v2 (48 giờ), được %s → %q", r.Outcome, r.Answer)
	}

	a.Deps.Now = func() time.Time { return time.Date(2025, 6, 1, 9, 0, 0, 0, time.UTC) }
	d := ask(t, a, user("director", ""), cauQuyTrinh).Decision
	later := *a
	later.Deps.Now = func() time.Time { return time.Date(2026, 10, 20, 9, 0, 0, 0, time.UTC) } // v2 nếu lấy ngày phát lại
	if diff, err := later.Replay(context.Background(), d); err != nil || len(diff) != 0 {
		t.Errorf("phát lại lượt tri thức ngày 01/06/2025: lệch %v (lỗi %v) — phải dùng ngày đã ghi", diff, err)
	}
}

// Câu hỏi kho không trả lời được không được nhận đoạn lạc đề rồi trả RESOLVED (P5). Trước khi sửa: "giấy phép xây dựng" nhận
// đoạn đổ bê tông trụ ("tục" trong "liên tục"), "nghỉ phép năm" nhận "báo cáo ngày phải đủ năm mục" — câu trả lời chép nguyên văn
// tài liệu nên qua bước kiểm (docs/viec/2-tra-quy-trinh/S7-BO-CA-DOC-LAP.md §2, "Biết im").
func TestRAGKhongTraLoiBangDoanLacDe(t *testing.T) {
	docs, err := rag.LoadMarkdown("knowledge/s7/kho", "abc-infra")
	if err != nil {
		t.Fatal(err)
	}
	a, m := newAsk(t), &models.Extractive{}
	a.Deps.Knowledge, a.Deps.Model = rag.NewMemoryIndex("abc-infra", docs), m
	for _, q := range []string{"thủ tục xin giấy phép xây dựng gồm những gì?", "quy trình xin nghỉ phép năm như thế nào?"} {
		if r := ask(t, a, user("director", ""), q).Response; r.Outcome != contracts.OutcomeInsufficientEvidence || m.Calls() != 0 {
			t.Errorf("%q: muốn INSUFFICIENT_EVIDENCE, 0 lời gọi model; được %s → %q", q, r.Outcome, r.Answer)
		}
	}
	// Câu kho trả lời được vẫn được trả lời.
	if r := ask(t, a, user("director", ""), "khi chênh lệch nhiên liệu thì bao lâu phải lập biên bản đối soát?").Response; r.Outcome != contracts.OutcomeResolved {
		t.Errorf("câu có đáp án phải RESOLVED, được %s → %q", r.Outcome, r.Answer)
	}
}

// Vết "vì sao" của S7 (R2-03 của team) phải có lý do cả khi nhánh RAG trả THÔNG BÁO phiên bản thay cho đoạn, và khi HYBRID bỏ
// phần quy trình vì lỗi truy xuất: hai đường này thoát trước chỗ ghi vết của đường truy xuất thường, nên chặng S7 hiện "chưa ghi".
func TestRAGVetViSaoKhiThongBaoVaLoi(t *testing.T) {
	a, _ := newAskPhienBan(t, sopBan(1, rag.StatusApproved, "2025-01-01", "", "24 giờ"), sopBan(2, rag.StatusApproved, "2026-03-01", "", "48 giờ"))
	ctx := ask(t, a, user("director", ""), cauQuyTrinh).Decision.Why.Context
	if s := strings.Join(ctx.Reasons, " | "); !strings.Contains(s, "version_conflict") || !strings.Contains(s, "v1") || !strings.Contains(s, "v2") {
		t.Errorf("thông báo xung đột phiên bản: vết S7 phải nêu loại thông báo và các bản, được %q", s)
	}
	a.Deps.Knowledge = loiTruyXuat{}
	ctx = ask(t, a, user("director", ""), cauGhep).Decision.Why.Context
	if s := strings.Join(ctx.Reasons, " | "); !strings.Contains(s, "truy xuất lỗi") {
		t.Errorf("HYBRID + lỗi truy xuất: vết S7 phải nói phần quy trình bị bỏ vì lỗi, được %q", s)
	}
}
