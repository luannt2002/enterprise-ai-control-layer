package models

import (
	"context"
	"testing"

	"github.com/luannt2002/enterprise-ai-systems/core/contracts"
)

func TestExtractive(t *testing.T) {
	m := &Extractive{}
	ctx := context.Background()
	ev := []contracts.Evidence{{ID: "D#0", Text: "bước một"}, {ID: "D#1", Text: "bước hai"}}
	d, err := m.Generate(ctx, Request{Question: "hỏi", Evidence: ev})
	if err != nil || d.Text != "bước một" {
		t.Fatalf("muốn đoạn đầu, được %q (lỗi %v)", d.Text, err)
	}
	if len(d.Claims) != 1 || len(d.Claims[0].EvidenceIDs) != 1 || d.Claims[0].EvidenceIDs[0] != "D#0" {
		t.Errorf("claim phải trỏ đúng bằng chứng đã dùng: %+v", d.Claims)
	}
	if d, _ := m.Generate(ctx, Request{Question: "hỏi"}); d.Text != "" || len(d.Claims) != 0 {
		t.Errorf("không bằng chứng: muốn bản nháp rỗng không claim, được %+v", d)
	}
	if m.Calls() != 2 {
		t.Errorf("Calls = %d, muốn 2", m.Calls())
	}
	if m.Reset(); m.Calls() != 0 {
		t.Errorf("sau Reset: Calls = %d, muốn 0", m.Calls())
	}
}

func TestFromConfig(t *testing.T) {
	for _, p := range []string{"", "extractive"} {
		if g, err := FromConfig(p); err != nil || g == nil {
			t.Errorf("provider %q: muốn gateway, được %v / %v", p, g, err)
		}
	}
	if _, err := FromConfig("provider-chua-co"); contracts.FaultOf(err) != contracts.FaultCode {
		t.Errorf("provider lạ: muốn lỗi %s (chặn lúc khởi động), được %v", contracts.FaultCode, err)
	}
}
