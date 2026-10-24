package construction_test

import (
	"strings"
	"testing"

	"github.com/luannt2002/enterprise-ai-control-layer/core/contracts"
	"github.com/luannt2002/enterprise-ai-control-layer/core/pipeline"
	"github.com/luannt2002/enterprise-ai-control-layer/intelligence/rag"
	"github.com/luannt2002/enterprise-ai-control-layer/observability/traces"
)

// Giao nhau sau khi gộp mọi nhánh về main 24/10 (D-RAG-05) — mỗi ca do soát độc lập của đợt gộp tìm ra.

func buocCua(o pipeline.Outcome) map[string]traces.Step {
	out := map[string]traces.Step{}
	for _, s := range traces.NewRecord(traces.Turn{Decision: o.Decision, Spans: o.Spans, TotalMs: o.TotalMs, UnaccountedMs: o.UnaccountedMs,
		Plan: o.Plan, Response: o.Response}).Steps {
		out[s.Stage] = s
	}
	return out
}

func chuBuoc(s traces.Step) string {
	return strings.Join(append(append([]string{s.Decision, s.Reason}, s.Output...), s.Rejected...), "\n")
}

// "mấy" là chữ hỏi lượng (vòng sửa S4) — nhưng "mấy tiếng" hỏi THỜI HẠN của quy trình: câu quy trình nhắc "dầu" không bị chốt lượng của
// route tri thức chặn (S7D-01 / 02 là đúng dạng này).
func TestGiaoNhauMayTiengLaThoiHanCuaQuyTrinh(t *testing.T) {
	a, _ := newAskPhienBan(t, sopBan(2, rag.StatusApproved, "2026-03-01", "", "48 giờ"))
	for _, q := range []string{"Theo quy trình, lệch dầu thì bao lâu phải lập biên bản?", "Theo quy trình, lệch dầu thì mấy tiếng phải lập biên bản?"} {
		if r := ask(t, a, user("director", ""), q).Response; r.Outcome != contracts.OutcomeResolved || !strings.Contains(r.Answer, "48 giờ") {
			t.Errorf("%q: muốn trả lời từ quy trình (48 giờ), được %s → %q", q, r.Outcome, r.Answer)
		}
	}
	// đối chứng: hỏi LƯỢNG dầu thì vẫn không lấy số từ tài liệu
	if r := ask(t, a, user("director", ""), "Theo quy trình, lệch dầu mấy lít thì phải lập biên bản?").Response; r.Outcome == contracts.OutcomeResolved && strings.Contains(r.Answer, "lít") {
		t.Errorf("hỏi lượng dầu không được trả bằng số của tài liệu: %q", r.Answer)
	}
}

// Câu lệnh ghi gõ KHÔNG dấu: "may" là "máy" chứ không phải chữ hỏi lượng "mấy" — không được trả số không nhãn như đã ghi (D-OP-18 = c).
func TestGiaoNhauLenhGhiKhongDauChuMay(t *testing.T) {
	a := newAsk(t)
	for _, q := range []string{"ghi nhan may xuc 03 tieu thu 3782 L dau hom nay", "ghi nhan EX-03 tieu thu 3782 L dau hom nay",
		"may xuc 03 hom nay tieu thu 3782 L dau"} { // câu NÊU giá trị, không động từ ghi, không dấu hiệu hỏi
		if r := ask(t, a, user("director", ""), q).Response; r.Outcome == contracts.OutcomeResolved {
			t.Errorf("%q: lệnh ghi có giá trị phải hỏi lại / từ chối, được RESOLVED → %q", q, r.Answer)
		}
	}
}

// Vết của các lối ra của route tri thức mà nhánh vết S8 / S10 chưa biết (soát độc lập đợt gộp, F1–F3).
func TestGiaoNhauVetLoiRaTriThuc(t *testing.T) {
	// model từ chối: S8 nói từ chối, S6 (đã trả đoạn) không bị gán lệch, S10 không nói "chứa nguyên văn" bản nháp rỗng
	a, _ := newAskPhienBan(t, sopBan(2, rag.StatusApproved, "2026-03-01", "", "48 giờ"))
	a.Deps.Model = modelTuChoi{}
	b := buocCua(ask(t, a, user("director", ""), cauQuyTrinh))
	if s := b[traces.StageGenerate]; s.Status != traces.StatusDeviant || !strings.Contains(s.Decision, "từ chối") {
		t.Errorf("model từ chối: S8 phải lệch và nói từ chối, được %s %q", s.Status, s.Decision)
	}
	if s := b[traces.StageExecute]; s.Status == traces.StatusDeviant {
		t.Errorf("model từ chối: S6 đã trả đoạn, không được là bước lệch: %q", chuBuoc(s))
	}
	if s := chuBuoc(b[traces.StageRespond]); strings.Contains(s, "chứa nguyên văn bản nháp ở S8 (0 ký tự") {
		t.Errorf("S10 không được nói câu giao chứa bản nháp rỗng: %q", s)
	}
	// thông báo phiên bản: S8 ghi mẫu câu thật, không còn ô "chưa ghi"
	a, _ = newAskPhienBan(t, sopBan(1, rag.StatusApproved, "2025-01-01", "", "24 giờ"), sopBan(2, rag.StatusApproved, "2026-03-01", "", "48 giờ"))
	b = buocCua(ask(t, a, user("director", ""), cauQuyTrinh))
	if s := chuBuoc(b[traces.StageGenerate]); !strings.Contains(s, "knowledge_version_conflict") || strings.Contains(s, traces.NotRecorded) {
		t.Errorf("thông báo phiên bản: S8 phải nêu mẫu knowledge_version_conflict, không có ô chưa ghi: %q", s)
	}
	if s := chuBuoc(b[traces.StageRespond]); !strings.Contains(s, "câu giao trùng bản nháp ở S8") {
		t.Errorf("thông báo phiên bản: S10 phải đối chiếu được câu giao với bản nháp mẫu câu ở S8: %q", s)
	}
	// HYBRID + model lỗi: S8 nói model lỗi; HYBRID + truy xuất lỗi: S6 nói truy xuất lỗi
	a, _ = newAskPhienBan(t, sopBan(2, rag.StatusApproved, "2026-03-01", "", "48 giờ"))
	a.Deps.Model = modelLoi{}
	if s := b2s(buocCua(ask(t, a, user("director", ""), cauGhep)), traces.StageGenerate); !strings.Contains(s, traces.ModelFailed) {
		t.Errorf("HYBRID + model lỗi: S8 phải nói model lỗi: %q", s)
	}
	a.Deps.Knowledge = loiTruyXuat{}
	if s := b2s(buocCua(ask(t, a, user("director", ""), cauGhep)), traces.StageExecute); !strings.Contains(s, "truy xuất lỗi") {
		t.Errorf("HYBRID + truy xuất lỗi: S6 phải nói truy xuất lỗi: %q", s)
	}
}

func b2s(b map[string]traces.Step, stage string) string { return chuBuoc(b[stage]) }

// Dự phòng S4 (B1) không trả lời bằng tài liệu câu mang NGHĨA CHƯA HỖ TRỢ (cảnh báo, dự báo…) khi câu không nêu thực thể.
func TestGiaoNhauDuPhongKhongCuuNghiaChuaHoTro(t *testing.T) {
	a, _ := newAskPhienBan(t, sopBan(2, rag.StatusApproved, "2026-03-01", "", "48 giờ"))
	cau := []string{"Nhắc tôi khi tổ trưởng kho đối soát", "Dự báo tổ trưởng kho đối soát khi nào?"}
	truoc := map[string]contracts.Outcome{}
	for _, q := range cau {
		truoc[q] = ask(t, a, user("director", ""), q).Response.Outcome
	}
	a.Deps.IntentFallback = pipeline.KnowledgeProbe{Knowledge: a.Deps.Knowledge, Name: "sop_question"}
	for _, q := range cau {
		if got := ask(t, a, user("director", ""), q).Response.Outcome; got != truoc[q] {
			t.Errorf("%q: dự phòng không được đổi kết quả của câu nghĩa chưa hỗ trợ: %s → %s", q, truoc[q], got)
		}
	}
}
