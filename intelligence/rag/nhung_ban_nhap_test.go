package rag

import (
	"context"
	"strings"
	"testing"
	"time"
)

// nhungGhi là embedder giả ghi lại MỌI chữ nó nhận (tức mọi chữ rời máy tới nhà cung cấp nhúng).
type nhungGhi struct {
	nhungGia
	da []string
}

func (e *nhungGhi) Embed(ctx context.Context, texts []string, q bool) ([][]float32, error) {
	e.da = append(e.da, texts...)
	return e.nhungGia.Embed(ctx, texts, q)
}

// xepGhi là reranker giả ghi lại mọi đoạn nó nhận.
type xepGhi struct{ da []string }

func (r *xepGhi) Rerank(_ context.Context, _ string, docs []string) ([]float64, error) {
	r.da = append(r.da, docs...)
	out := make([]float64, len(docs))
	for i := range out {
		out[i] = 0.5
	}
	return out, nil
}

// Chữ của bản CHƯA duyệt (nháp, chờ soát — kể cả bản bị cổng chất lượng giữ) không rời máy (F-017): không vào lô nhúng kho, không tới
// reranker — kể cả khi kho còn vector cũ của chúng (nhúng từ trước bản sửa này). Bản đã duyệt vẫn được nhúng và trả như cũ.
func TestNhungKhongGuiBanNhap(t *testing.T) {
	const canary = "CANARY-7F3"
	ks := []string{"ks"}
	docs := []Document{
		{ID: "QT", Tenant: "t", Version: 1, Status: StatusApproved, ACL: ks, Title: "Quy trình nghiệm thu", Body: "1. Biên bản nghiệm thu do tư vấn giám sát lập."},
		{ID: "QT", Tenant: "t", Version: 2, Status: StatusDraft, ACL: ks, Title: "Quy trình nghiệm thu", Body: "1. Biên bản nghiệm thu do " + canary + "-qt lập."},
		{ID: "NHAP", Tenant: "t", Version: 1, Status: StatusDraft, ACL: ks, Title: "Quy trình nghiệm thu mới", Body: "1. Biên bản nghiệm thu " + canary + "-nhap."},
		{ID: "SOAT", Tenant: "t", Version: 1, Status: StatusNeedsReview, Hold: "bảng không đủ ô", ACL: ks, Title: "Quy trình nghiệm thu khác", Body: "1. Biên bản nghiệm thu " + canary + "-soat."},
	}
	ix := NewMemoryIndex("t", docs)
	ix.Now = func() time.Time { return time.Date(2026, 10, 20, 10, 0, 0, 0, time.UTC) }
	e := &nhungGhi{nhungGia: nhungGia{khaiNiem: khaiNiemThu}}
	texts := ix.EmbedTexts()
	vs, err := e.Embed(context.Background(), texts, false)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(strings.Join(e.da, "\n"), "tư vấn giám sát") {
		t.Fatalf("tiền đề: bản đã duyệt phải được nhúng: %q", e.da)
	}
	// Vector của MỌI đoạn, kể cả bản chưa duyệt: kho nhúng trước bản sửa còn giữ chúng — luật không được dựa vào "thiếu vector".
	sem := &Semantic{Embedder: e, Vectors: map[string][]float32{}, Reranker: &xepGhi{}}
	for i, x := range texts {
		sem.Vectors[EmbedKey(x)] = vs[i]
	}
	cu := &nhungGia{khaiNiem: khaiNiemThu}
	for i := range ix.chunks {
		x := embedText(&ix.chunks[i])
		v, _ := cu.Embed(context.Background(), []string{x}, false)
		sem.Vectors[EmbedKey(x)] = v[0]
	}
	ix.Semantic = sem
	ev, err := ix.Retrieve(context.Background(), ksLai(""), "Biên bản nghiệm thu do ai lập?", 3)
	if err != nil || len(ev) == 0 || ev[0].ID != "QT@v1#0" {
		t.Errorf("bản đã duyệt phải được trả: %v %v", ids(ev), err)
	}
	if s := strings.Join(e.da, "\n"); strings.Contains(s, canary) {
		t.Errorf("chữ của bản chưa duyệt đã gửi ra embedder: %q", e.da)
	}
	if s := strings.Join(sem.Reranker.(*xepGhi).da, "\n"); strings.Contains(s, canary) {
		t.Errorf("chữ của bản chưa duyệt đã gửi ra reranker: %q", sem.Reranker.(*xepGhi).da)
	}
}
