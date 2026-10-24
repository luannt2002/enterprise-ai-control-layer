package rag

import (
	"context"
	"testing"
	"time"

	"github.com/luannt2002/enterprise-ai-control-layer/core/contracts"
)

// Biết im: chủ đề của lượt (KnowledgeQuery.Topic) mà từ nửa số từ nội dung trở lên không có trong đoạn nào người hỏi thấy thì kho
// không nói tới điều được hỏi — không đoạn nào là bằng chứng, dù một chữ trùng hợp vượt MinScore. Không có Topic thì như cũ.
func TestRetrieveChuDeNgoaiKho(t *testing.T) {
	d := Document{ID: "BP", Tenant: "a", Title: "Biện pháp thi công trụ cầu", ACL: []string{"r"},
		Body: "1. Bê tông trụ đổ liên tục theo từng lớp dày không quá 50 cm.\n2. Bảo dưỡng ẩm ít nhất 7 ngày."}
	ix := NewMemoryIndex("a", []Document{d})
	pr := principal("a", "r")
	q := "thủ tục xin giấy phép đổ thải gồm những gì?"
	ctx := func(topic string) context.Context {
		return contracts.WithKnowledgeQuery(context.Background(), contracts.KnowledgeQuery{AsOf: time.Now(), Topic: topic})
	}
	if ev, err := ix.Retrieve(ctx(""), pr, q, 3); err != nil || len(ev) == 0 {
		t.Fatalf("không Topic: hành vi cũ (khớp 'tục' + 'đổ' của dòng 1, vượt MinScore) phải còn để so, được %v / %v", ev, err)
	}
	if ev, err := ix.Retrieve(ctx(q), pr, q, 3); err != nil || len(ev) != 0 {
		t.Errorf("chủ đề ngoài kho: muốn 0 đoạn, được %v / %v", ev, err)
	}
	if ev, err := ix.Retrieve(ctx("đổ bê tông trụ mỗi lớp dày bao nhiêu"), pr, "đổ bê tông trụ mỗi lớp dày bao nhiêu", 3); err != nil || len(ev) == 0 {
		t.Errorf("chủ đề có trong kho: phải có đoạn, được %v / %v", ev, err)
	}
}

// Ranh giới đã chốt trước khi đo: ĐÚNG một nửa số từ của chủ đề ngoài kho thì đã im (≥ 1/2, không phải > 1/2); dưới một nửa thì không.
func TestRetrieveChuDeNuaNgoaiKho(t *testing.T) {
	d := Document{ID: "BP", Tenant: "a", Title: "Biện pháp thi công trụ cầu", ACL: []string{"r"},
		Body: "1. Bê tông trụ đổ liên tục theo từng lớp dày không quá 50 cm.\n2. Bảo dưỡng ẩm ít nhất 7 ngày."}
	ix := NewMemoryIndex("a", []Document{d})
	hoi := func(topic string) int {
		ctx := contracts.WithKnowledgeQuery(context.Background(), contracts.KnowledgeQuery{AsOf: time.Now(), Topic: topic})
		ev, err := ix.Retrieve(ctx, principal("a", "r"), "bê tông trụ thải rác", 3)
		if err != nil {
			t.Fatal(err)
		}
		return len(ev)
	}
	if n := hoi("bê tông thải"); n == 0 {
		t.Error("1/3 từ của chủ đề ngoài kho: chưa tới ngưỡng, phải còn đoạn")
	}
	if n := hoi("bê tông thải rác"); n != 0 {
		t.Errorf("đúng 2/4 từ của chủ đề ngoài kho: phải im, được %d đoạn", n)
	}
}
