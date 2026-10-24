package pipeline

import (
	"slices"
	"testing"

	"github.com/luannt2002/enterprise-ai-control-layer/core/contracts"
	"github.com/luannt2002/enterprise-ai-control-layer/observability/traces"
)

// S6 của route tri thức ghi đủ từ vết của chỉ mục: đường + số đoạn thấy, hạng / điểm từng đoạn giữ lại, lý do im, đoạn bị loại vì ngân
// sách k; hiện thực KnowledgeRetriever không ghi vết thì nói rõ, không bịa số.
func TestNoteExecute(t *testing.T) {
	var w traces.Why
	ev := []contracts.Evidence{{ID: "QT#1", Confidence: 0.9, Source: contracts.SourceRef{Name: "Quy trình"}}}
	noteExecute(&w, "tri thức: ", contracts.RetrievalReport{Path: "từ khoá", Docs: 4, Chunks: 9, NotServable: 1, Beyond: 2}, ev, 3)
	want := []string{"tri thức: đường truy xuất: từ khoá; xét 4 bản tài liệu / 9 đoạn người hỏi thấy, 1 bản trong đó không làm căn cứ được ở ngày hỏi",
		"tri thức: QT#1 (Quy trình): hạng 1, điểm 0.900"}
	if !slices.Equal(w.Execute.Reasons, want) || !slices.Equal(w.Execute.Rejected,
		[]string{"tri thức: 2 đoạn khớp khác xếp sau hạng 3: không đưa vào ngữ cảnh (tối đa 3 đoạn mỗi lượt)"}) {
		t.Errorf("S6:\n  lý do %q\n  loại %q", w.Execute.Reasons, w.Execute.Rejected)
	}
	w = traces.Why{}
	noteExecute(&w, "", contracts.RetrievalReport{Path: "lai, xếp lại", Silence: "điểm xếp lại của đoạn đầu 0.100 < ngưỡng 0.190"}, nil, 3)
	if len(w.Execute.Reasons) != 2 || w.Execute.Reasons[1] != "không trả đoạn nào: điểm xếp lại của đoạn đầu 0.100 < ngưỡng 0.190" || len(w.Execute.Rejected) != 0 {
		t.Errorf("S6 im: %+v", w.Execute)
	}
	w = traces.Why{}
	noteExecute(&w, "", contracts.RetrievalReport{}, ev, 3)
	if !slices.Equal(w.Execute.Reasons, []string{"chỉ mục tri thức không ghi vết truy xuất"}) {
		t.Errorf("chỉ mục không ghi vết: %+v", w.Execute)
	}
}
