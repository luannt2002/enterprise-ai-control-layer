package traces_test

import (
	"slices"
	"strings"
	"testing"

	"github.com/luannt2002/enterprise-ai-control-layer/core/contracts"
	"github.com/luannt2002/enterprise-ai-control-layer/observability/traces"
)

// V8a — khoảng trống nhánh thoát sớm của RAG: câu đòi LƯỢNG nghiệp vụ mà chỉ có tài liệu ⇒ kết cục thiếu bằng chứng, không gọi model. S6, S7, S8 đều "skipped"
// rồi có outcome, và S8 chỉ nói "chưa ghi — lý do thoát sớm và tên mẫu", nên một lỗi chọn sai outcome ở nhánh này bị đọc thành S6 (3 / 3 thẻ). Các test dưới đây
// đòi: lý do thoát và quy tắc, kết cục mà nhánh chọn, mẫu câu thật và bản nháp ở S8; S6 nói thoát sớm TRƯỚC truy xuất; không còn ô "chưa ghi" nào.

const (
	noKnowledgeText   = "Chưa có tài liệu bạn được xem trả lời được câu hỏi này."
	hybridNoKnowledge = "Chưa có tài liệu bạn được xem trả lời phần quy trình."
)

func TestVetThoatSomRAGDoiLuongNghiepVu(t *testing.T) {
	turn := realTurn(t, realAsk(t), person("director", ""), "Theo quy trình thì một máy xúc tiêu thụ bao nhiêu lít dầu mỗi giờ?")
	if turn.Plan.Route != contracts.RouteRAG || turn.Decision.Outcome != "INSUFFICIENT_EVIDENCE" || turn.Response.Answer != noKnowledgeText {
		t.Fatalf("tiền đề: route %s outcome %s câu %q", turn.Plan.Route, turn.Decision.Outcome, turn.Response.Answer)
	}
	steps := traces.NewRecord(turn).Steps
	s6, s8, s10 := stepOf(t, steps, "S6"), stepOf(t, steps, "S8"), stepOf(t, steps, "S10")
	if s8.Status != traces.StatusSkipped || s8.Decision != "model chưa được gọi" {
		t.Errorf("S8: %+v", s8)
	}
	want := "route RAG cần model nhưng không có span generate: lượt kết thúc sớm với outcome INSUFFICIENT_EVIDENCE, câu dựng bằng mẫu của tenant; " +
		"thoát sớm TRƯỚC truy xuất, model chưa được gọi: câu đòi lượng nghiệp vụ (cụm hỏi lượng “bao nhiêu” của pack, chiều đo volume) mà route RAG chỉ có tài liệu — " +
		"tài liệu không phải nguồn của số liệu (P4), nên báo thiếu bằng chứng; " +
		"kết cục nhánh đã chọn: INSUFFICIENT_EVIDENCE (mẫu câu no_knowledge); " +
		draftLine("", "dựng bằng mẫu câu no_knowledge (không gọi model)", noKnowledgeText)
	if s8.Reason != want {
		t.Errorf("S8 của nhánh thoát sớm:\n  muốn: %s\n  được: %s", want, s8.Reason)
	}
	if traces.IsNotRecorded(s8.Reason) || slices.ContainsFunc(s8.Rejected, traces.IsNotRecorded) {
		t.Errorf("S8 còn ô trống: %q %q", s8.Reason, s8.Rejected)
	}
	// S6 không còn là chỗ người đọc đổ lỗi: nói thẳng đây là thoát sớm TRƯỚC truy xuất và lý do nằm ở S8
	if s6.Status != traces.StatusSkipped || !strings.Contains(s6.Reason, "route RAG cần truy xuất nhưng lượt thoát sớm trước truy xuất (lý do ở S8)") {
		t.Errorf("S6 phải nói thoát sớm trước truy xuất: %+v", s6)
	}
	// S10: kết cục khớp kết cục mà nhánh đã chọn ở S8, câu giao đúng là bản nháp
	if !strings.Contains(s10.Reason, "câu giao trùng bản nháp ở S8 (băm "+sha12(noKnowledgeText)+")") || s10.Decision != "outcome INSUFFICIENT_EVIDENCE" {
		t.Errorf("S10: %+v", s10)
	}
}

func TestVetThoatSomRAGTruyXuatRong(t *testing.T) {
	turn := realTurn(t, realAsk(t), person("site_engineer", "PRJ-A"), "Quy trình xử lý chênh lệch nhiên liệu là gì?")
	if turn.Plan.Route != contracts.RouteRAG || turn.Decision.Outcome != "INSUFFICIENT_EVIDENCE" {
		t.Fatalf("tiền đề: route %s outcome %s", turn.Plan.Route, turn.Decision.Outcome)
	}
	s8 := stepOf(t, traces.NewRecord(turn).Steps, "S8")
	want := "route RAG cần model nhưng không có span generate: lượt kết thúc sớm với outcome INSUFFICIENT_EVIDENCE, câu dựng bằng mẫu của tenant; " +
		"thoát sớm SAU truy xuất, model chưa được gọi: truy xuất trả 0 đoạn người hỏi được đọc; " +
		"kết cục nhánh đã chọn: INSUFFICIENT_EVIDENCE (mẫu câu no_knowledge); " +
		draftLine("", "dựng bằng mẫu câu no_knowledge (không gọi model)", noKnowledgeText)
	if s8.Reason != want {
		t.Errorf("S8 khi truy xuất rỗng:\n  muốn: %s\n  được: %s", want, s8.Reason)
	}
}

func TestVetThoatSomHybrid(t *testing.T) {
	const q = "EX-03 hôm nay được cấp bao nhiêu dầu, và theo quy trình thì khi lệch phải làm gì?"
	// phần số liệu xong, phần tri thức rỗng: câu số liệu giữ nguyên RESOLVED, mẩu hybrid_no_knowledge nối vào cuối
	turn := realTurn(t, realAsk(t), person("site_engineer", "PRJ-A"), q)
	if turn.Plan.Route != contracts.RouteHybrid || turn.Decision.Outcome != "RESOLVED" || !strings.HasSuffix(turn.Response.Answer, hybridNoKnowledge) {
		t.Fatalf("tiền đề: route %s outcome %s câu %q", turn.Plan.Route, turn.Decision.Outcome, turn.Response.Answer)
	}
	metricPart := strings.TrimSuffix(turn.Response.Answer, " "+hybridNoKnowledge)
	s8 := stepOf(t, traces.NewRecord(turn).Steps, "S8")
	want := "route HYBRID_RETRIEVAL cần model nhưng không có span generate: lượt kết thúc sớm với outcome RESOLVED, câu dựng bằng mẫu của tenant; " +
		draftLine("số liệu: ", "dựng bằng mẫu câu metric_total (không gọi model)", metricPart) + "; " +
		"tri thức: thoát sớm SAU truy xuất, model chưa được gọi: truy xuất trả 0 đoạn người hỏi được đọc; " +
		"tri thức: phần quy trình bỏ, kết cục giữ nguyên RESOLVED: mẫu câu hybrid_no_knowledge nối vào cuối câu số liệu; " +
		draftLine("tri thức: ", "dựng bằng mẫu câu hybrid_no_knowledge (không gọi model)", hybridNoKnowledge)
	if s8.Reason != want {
		t.Errorf("S8 HYBRID, tri thức rỗng:\n  muốn: %s\n  được: %s", want, s8.Reason)
	}
	// phần số liệu không ra câu trả lời: không truy xuất, không gọi model — lý do là outcome của phần số liệu
	turn = realTurn(t, realAsk(t), person("director", ""), "EX-07 hôm nay được cấp bao nhiêu dầu, và theo quy trình thì khi lệch phải làm gì?")
	if turn.Plan.Route != contracts.RouteHybrid || turn.Decision.Outcome != "INSUFFICIENT_EVIDENCE" {
		t.Fatalf("tiền đề: route %s outcome %s", turn.Plan.Route, turn.Decision.Outcome)
	}
	s8 = stepOf(t, traces.NewRecord(turn).Steps, "S8")
	for _, frag := range []string{
		"tri thức: không truy xuất, model chưa được gọi: phần số liệu không ra câu trả lời (outcome INSUFFICIENT_EVIDENCE) nên không nối tài liệu vào",
		draftLine("số liệu: ", "dựng bằng mẫu câu no_data (không gọi model)", turn.Response.Answer),
	} {
		if !strings.Contains(s8.Reason, frag) {
			t.Errorf("S8 HYBRID, số liệu không có dữ liệu, thiếu %q trong:\n  %s", frag, s8.Reason)
		}
	}
	if traces.IsNotRecorded(s8.Reason) {
		t.Errorf("S8 còn ô trống: %s", s8.Reason)
	}
}

// Đối chứng: lượt có gọi model vẫn là "model diễn đạt" (không bị nhánh thoát sớm nuốt), và không có dòng "thoát sớm" nào.
func TestVetThoatSomKhongNuotLuotCoGoiModel(t *testing.T) {
	s8 := stepOf(t, traces.NewRecord(realTurn(t, realAsk(t), person("director", ""), "Quy trình xử lý chênh lệch nhiên liệu là gì?")).Steps, "S8")
	if s8.Status != traces.StatusOK || s8.Decision != "model diễn đạt" || strings.Contains(s8.Reason, "thoát sớm") || strings.Contains(s8.Reason, "kết cục nhánh") {
		t.Errorf("lượt có gọi model: %+v", s8)
	}
}
