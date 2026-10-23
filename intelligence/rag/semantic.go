package rag

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"maps"
	"math"
	"slices"
	"strings"

	"github.com/luannt2002/enterprise-ai-control-layer/core/contracts"
)

// Embedder biến chữ thành vector. Name định danh không gian vector (model + số chiều): vector của hai Name khác nhau không so
// được với nhau.
// query = true khi nhúng câu hỏi, false khi nhúng đoạn tài liệu (model phân biệt hai loại thì vector khác nhau).
type Embedder interface {
	Embed(ctx context.Context, texts []string, query bool) ([][]float32, error)
	Name() string
}

// Reranker chấm lại mức liên quan của các đoạn với câu hỏi; điểm theo đúng thứ tự docs.
type Reranker interface {
	Rerank(ctx context.Context, query string, docs []string) ([]float64, error)
}

// Semantic bật truy hồi LAI cho MemoryIndex (docs/ROADMAP-RAG.md §6.4). Lọc tenant / vai / phạm vi / phiên bản GIỮ NGUYÊN như đường
// từ khoá và chạy TRƯỚC; sau đó xếp đoạn bằng RRF của hạng từ khoá và hạng cosine, xếp lại nhóm đầu bằng Reranker (nếu có), rồi
// "biết im" bằng ngưỡng điểm. Ngưỡng điểm (khi đặt) thay luật "biết im" theo từ của đường từ khoá: luật đó im nhầm khi người hỏi dùng
// chữ khác tài liệu — đúng chỗ embedding giải.
type Semantic struct {
	Embedder Embedder
	// Vectors: EmbedKey(chữ nhúng) → vector trong không gian Embedder.Name(). Đoạn ứng viên của lượt thiếu vector làm Retrieve LỖI,
	// không âm thầm chỉ dùng từ khoá: nạp tài liệu mới hay đổi model mà chưa nhúng lại là lỗi vận hành phải thấy.
	Vectors  map[string][]float32
	Reranker Reranker // nil = không xếp lại
	Pool     int      // số đoạn đầu theo RRF đưa vào xếp lại; 0 = 30
	// MinScore: điểm của đoạn đứng đầu — điểm xếp lại nếu có Reranker, cosine lớn nhất nếu không — dưới ngưỡng thì không đoạn nào
	// là bằng chứng. Ngưỡng gắn với model chấm điểm: chọn trên bộ phát triển, kiểm trên bộ độc lập. 0 = chưa có ngưỡng: dùng luật
	// "biết im" theo chủ đề của đường từ khoá (không bao giờ là "không chặn gì").
	MinScore float64
}

// rrfK là hằng số của Reciprocal Rank Fusion: điểm = Σ 1 / (rrfK + hạng).
const rrfK = 60

// EmbedKey là khoá của chữ đã nhúng: vector chỉ phải tính lại khi chữ đổi.
func EmbedKey(text string) string {
	h := sha256.Sum256([]byte(text))
	return hex.EncodeToString(h[:])
}

// embedText là chữ đại diện cho đoạn khi nhúng và khi xếp lại: tiêu đề + đoạn (bước của quy trình không lặp lại tên quy trình).
// NewMemoryIndex băm đúng chữ này vào chunk.key.
func embedText(c *chunk) string { return c.doc.Title + ": " + c.text }

// EmbedTexts trả chữ cần nhúng của mọi đoạn trong chỉ mục, mỗi chữ một lần, theo thứ tự đoạn.
func (ix *MemoryIndex) EmbedTexts() []string {
	seen := map[string]bool{}
	var out []string
	for i := range ix.chunks {
		if t := embedText(&ix.chunks[i]); !seen[t] {
			seen[t] = true
			out = append(out, t)
		}
	}
	return out
}

func cosine(a, b []float32) float64 {
	var dot, na, nb float64
	for i := range a {
		dot += float64(a[i]) * float64(b[i])
		na += float64(a[i]) * float64(a[i])
		nb += float64(b[i]) * float64(b[i])
	}
	if na == 0 || nb == 0 {
		return 0
	}
	return dot / math.Sqrt(na*nb)
}

// hybrid là Retrieve khi có Semantic; vis / serving / live / df / seen đã tính như đường từ khoá (chỉ trên tài liệu người hỏi thấy).
func (ix *MemoryIndex) hybrid(ctx context.Context, pr contracts.Principal, query string, k int, today string,
	vis map[*Document]bool, serving map[string]int, live map[string]bool, df map[string]int, seen int) ([]contracts.Evidence, error) {
	sem := ix.Semantic
	q := slices.Sorted(maps.Keys(queryTerms(query)))
	type cand struct {
		c                   *chunk
		kw, cos, rrf, score float64
	}
	// Ứng viên: đoạn làm căn cứ được, và đoạn của tài liệu không có đúng một bản làm căn cứ (để báo phiên bản như đường từ khoá).
	var cs []*cand
	for i := range ix.chunks {
		c := &ix.chunks[i]
		id := c.doc.ID
		if vis[c.doc] && (serving[id] == 1 && servable(c.doc, today) || serving[id] != 1 && live[id]) {
			cs = append(cs, &cand{c: c, kw: idfScore(c, q, df, seen)})
		}
	}
	if len(cs) == 0 {
		return []contracts.Evidence{}, nil
	}
	// Kiểm vector TRƯỚC khi nhúng câu hỏi: kho chưa nhúng xong thì không gửi câu hỏi ra ngoài vô ích.
	missing := 0
	for _, x := range cs {
		if _, ok := sem.Vectors[x.c.key]; !ok {
			missing++
		}
	}
	if missing > 0 {
		return nil, contracts.Errorf(contracts.FaultData, fmt.Sprintf("chỉ mục thiếu vector của %d đoạn trong không gian %s — chạy nhúng lại kho", missing, sem.Embedder.Name()))
	}
	qv, err := sem.Embedder.Embed(ctx, []string{query}, true)
	if err != nil {
		return nil, err
	}
	if len(qv) != 1 {
		return nil, contracts.Errorf(contracts.FaultCode, fmt.Sprintf("embedder trả %d vector cho một câu hỏi", len(qv)))
	}
	for _, x := range cs {
		v := sem.Vectors[x.c.key]
		if len(v) != len(qv[0]) {
			return nil, contracts.Errorf(contracts.FaultData, fmt.Sprintf("vector đã lưu %d chiều, câu hỏi %d chiều (không gian %s) — chạy nhúng lại kho", len(v), len(qv[0]), sem.Embedder.Name()))
		}
		x.cos = cosine(qv[0], v)
	}
	// Hoà điểm: theo tài liệu rồi vị trí dòng, để hai lần hỏi ra cùng thứ tự.
	order := func(score func(*cand) float64) func(a, b *cand) int {
		return func(a, b *cand) int {
			if sa, sb := score(a), score(b); sa != sb {
				if sa > sb {
					return -1
				}
				return 1
			}
			if c := strings.Compare(a.c.doc.ID, b.c.doc.ID); c != 0 {
				return c
			}
			if a.c.doc.Version != b.c.doc.Version {
				return a.c.doc.Version - b.c.doc.Version
			}
			return a.c.seq - b.c.seq
		}
	}
	// Chỉ đoạn đạt ngưỡng của đường từ khoá (ix.MinScore) góp hạng từ khoá: một chữ khớp yếu (gõ không dấu "giao" khớp "giáo")
	// mà được tính trọn một hạng thì đẩy đoạn lạc đề lên trên đoạn khớp nghĩa.
	byKw := slices.DeleteFunc(slices.Clone(cs), func(x *cand) bool { return x.kw == 0 || x.kw < ix.MinScore })
	slices.SortFunc(byKw, order(func(x *cand) float64 { return x.kw }))
	for r, x := range byKw {
		x.rrf += 1 / float64(rrfK+r+1)
	}
	slices.SortFunc(cs, order(func(x *cand) float64 { return x.cos }))
	gate := cs[0].cos
	for r, x := range cs {
		x.rrf += 1 / float64(rrfK+r+1)
		x.score = x.cos
	}
	slices.SortFunc(cs, order(func(x *cand) float64 { return x.rrf }))
	n := sem.Pool
	if n <= 0 {
		n = 30
	}
	pool := cs[:min(n, len(cs))]
	if sem.Reranker != nil {
		texts := make([]string, len(pool))
		for i, x := range pool {
			texts[i] = embedText(x.c)
		}
		scores, err := sem.Reranker.Rerank(ctx, query, texts)
		if err != nil {
			return nil, err
		}
		if len(scores) != len(pool) {
			return nil, contracts.Errorf(contracts.FaultCode, fmt.Sprintf("reranker trả %d điểm cho %d đoạn", len(scores), len(pool)))
		}
		for i, x := range pool {
			x.score = scores[i]
		}
		slices.SortFunc(pool, order(func(x *cand) float64 { return x.score }))
		gate = pool[0].score
	}
	if gate < sem.MinScore {
		return []contracts.Evidence{}, nil
	}
	if id := pool[0].c.doc.ID; serving[id] != 1 {
		kind := NoticeNoEffective
		if serving[id] > 1 {
			kind = NoticeConflict
		}
		return nil, ix.notice(id, kind, pr)
	}
	out := []contracts.Evidence{}
	for _, x := range pool {
		if len(out) == k {
			break
		}
		if serving[x.c.doc.ID] == 1 && servable(x.c.doc, today) {
			out = append(out, evidenceOf(x.c, x.score))
		}
	}
	return out, nil
}
