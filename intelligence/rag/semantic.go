package rag

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"maps"
	"math"
	"regexp"
	"slices"
	"strings"
	"sync"
	"unicode"
	"unicode/utf8"

	"github.com/luannt2002/enterprise-ai-control-layer/core/contracts"
	"github.com/luannt2002/enterprise-ai-control-layer/data/normalization"
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

// releasable: chữ của bản được phép RỜI MÁY tới embedder / reranker — bản đã duyệt hoặc không khai (vế trạng thái của servable). Nháp và
// chờ soát (kể cả bản cổng chất lượng giữ) chưa được công bố nên không gửi ra nhà cung cấp; bản đã thay thế không còn phục vụ (F-017).
// Không xét ngày hiệu lực: bản đã duyệt có hiệu lực từ ngày mai phải có vector trước ngày đó, không thì lượt đầu của ngày đó lỗi thiếu vector.
func releasable(d *Document) bool { return d.Status == "" || d.Status == StatusApproved }

// EmbedTexts trả chữ cần nhúng của mọi đoạn thuộc bản được phép rời máy (releasable), mỗi chữ một lần, theo thứ tự đoạn.
func (ix *MemoryIndex) EmbedTexts() []string {
	seen := map[string]bool{}
	var out []string
	for i := range ix.chunks {
		if !releasable(ix.chunks[i].doc) {
			continue
		}
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
	// Ứng viên: đoạn làm căn cứ được, và đoạn của tài liệu không có đúng một bản làm căn cứ (để báo phiên bản như đường từ khoá) — chỉ
	// đoạn của bản được phép rời máy: chữ của ứng viên tới reranker, và vector của nó phải có (F-017). Tài liệu chỉ có bản chưa duyệt vì
	// thế không báo phiên bản trên đường lai (đường từ khoá vẫn báo).
	var cs []*cand
	for i := range ix.chunks {
		c := &ix.chunks[i]
		id := c.doc.ID
		if vis[c.doc] && releasable(c.doc) && (serving[id] == 1 && servable(c.doc, today) || serving[id] != 1 && live[id]) {
			cs = append(cs, &cand{c: c, kw: idfScore(c, q, df, seen)})
		}
	}
	if len(cs) == 0 {
		report(ctx).Silence = "không đoạn nào làm căn cứ được trong tài liệu người hỏi thấy"
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
	// Câu gõ không dấu: nhúng / xếp lại bằng câu đã khôi phục dấu (điểm xếp lại của câu không dấu thấp hơn hẳn — khoi_phuc_dau_test.go).
	// Đường từ khoá vẫn chấm câu gốc (đã so bỏ dấu).
	if rq, changes, ok := ix.restoreDiacritics(query, vis); ok {
		query, report(ctx).Restored = rq, changes
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
		name := "cosine"
		if sem.Reranker != nil {
			name = "điểm xếp lại"
		}
		report(ctx).Silence = fmt.Sprintf("%s của đoạn đầu %.3f < ngưỡng %.3f", name, gate, sem.MinScore)
		return []contracts.Evidence{}, nil
	}
	if id := pool[0].c.doc.ID; serving[id] != 1 {
		kind := NoticeNoEffective
		if serving[id] > 1 {
			kind = NoticeConflict
		}
		return nil, ix.notice(id, kind, pr)
	}
	out, rep := []contracts.Evidence{}, report(ctx)
	for _, x := range pool {
		switch {
		case serving[x.c.doc.ID] != 1 || !servable(x.c.doc, today):
		case len(out) < k:
			out = append(out, evidenceOf(x.c, x.score))
		case x.score >= sem.MinScore: // cùng nghĩa với đường từ khoá: chỉ đếm đoạn đạt ngưỡng
			rep.Beyond++
		}
	}
	return out, nil
}

// wordRe: một từ (chữ / số).
var wordRe = regexp.MustCompile(`[\p{L}\p{N}]+`)

// docVocab là từ vựng của MỘT tài liệu (tiêu đề + thân, đếm một lần) để khôi phục dấu: cặp từ liền nhau đã bỏ dấu → các dạng có dấu và số
// lần gặp. Theo tài liệu vì tài liệu là đơn vị quyền đọc: mỗi câu chỉ cộng từ vựng của tài liệu người hỏi thấy.
type docVocab map[[2]string]map[[2]string]int

// lazyVocab giữ từ vựng của mọi tài liệu trong chỉ mục, dựng MỘT lần ở câu không dấu đầu tiên: chỉ mục chỉ chạy đường từ khoá (không
// đường lai) không trả bộ nhớ cho nó (đo 24/10: dựng sẵn làm mỗi lần dựng chỉ mục 5.100 tài liệu tốn 728 → 1.005 MB). Con trỏ: bản sao
// của MemoryIndex trong Retrieve dùng chung.
type lazyVocab struct {
	once sync.Once
	m    map[*Document]docVocab
}

func (ix *MemoryIndex) vocabs() map[*Document]docVocab {
	ix.vocab.once.Do(func() {
		ix.vocab.m = make(map[*Document]docVocab, len(ix.docs))
		for _, d := range ix.docs {
			ix.vocab.m[d] = vocabOf(d.Title + "\n" + d.Body)
		}
	})
	return ix.vocab.m
}

// vocabOf dựng từ vựng cặp từ của một văn bản: chỉ cặp cách nhau bằng dấu cách (không qua dấu câu, xuống dòng, ký hiệu bảng).
func vocabOf(text string) docVocab {
	t := strings.ToLower(normalization.NFC(text))
	loc := wordRe.FindAllStringIndex(t, -1)
	v := docVocab{}
	for i := 1; i < len(loc); i++ {
		if strings.Trim(t[loc[i-1][1]:loc[i][0]], " \t") != "" {
			continue
		}
		a, b := t[loc[i-1][0]:loc[i-1][1]], t[loc[i][0]:loc[i][1]]
		k := [2]string{normalization.Fold(a), normalization.Fold(b)}
		if v[k] == nil {
			v[k] = map[[2]string]int{}
		}
		v[k][[2]string{a, b}]++
	}
	return v
}

// restorable: từ được phép khôi phục dấu — không có chữ số, không viết hoa toàn bộ (mã như "TT", "NHNN"), không dính "-", ".", "@", "/",
// "_", ":", "\\", "#", "+", "=" hay chữ số ở hai bên (mã "EX-03", email, đường dẫn, tệp).
func restorable(q string, l []int) bool {
	w := q[l[0]:l[1]]
	if strings.ContainsFunc(w, unicode.IsDigit) || utf8.RuneCountInString(w) > 1 && strings.ToUpper(w) == w {
		return false
	}
	code := func(r rune) bool { return strings.ContainsRune("-.@/_:\\#+=", r) || unicode.IsDigit(r) }
	before, _ := utf8.DecodeLastRuneInString(q[:l[0]])
	after, _ := utf8.DecodeRuneInString(q[l[1]:])
	return !(l[0] > 0 && code(before)) && !(l[1] < len(q) && code(after))
}

// restoreDiacritics khôi phục dấu cho câu hỏi gõ KHÔNG dấu theo CẶP từ liền nhau trong từ vựng của các tài liệu người hỏi thấy (vis):
// dạng gặp nhiều nhất — hoà điểm thì không đoán; cặp được chồng nhau ("lưu két" + "két sắt") nhưng cặp mà dạng thắng mâu thuẫn từ đã
// gán thì bỏ cả cặp. Không khôi phục từ đơn (đoán sai nghĩa: "do" → "đo"); câu quá maxRestoreWords từ không khôi phục. Từ không đổi giữ nguyên từng byte; chữ hoa đầu từ giữ. Câu đã có dấu, hoặc không từ nào đổi: trả
// nguyên câu, "", false. changes = các từ đã đổi ("bien → biên, ban → bản") cho vết. Chỉ tài liệu người hỏi thấy góp chữ nên chữ của
// tài liệu bị cấm không lọt vào câu gửi đi lẫn vết.
// shortcut: cộng từ vựng của mọi tài liệu thấy được cho mỗi cặp của câu (số tài liệu × số cặp lượt tra); gom chỉ mục cặp → tài liệu khi
// kho thật vượt vài nghìn tài liệu.
func (ix *MemoryIndex) restoreDiacritics(query string, vis map[*Document]bool) (string, string, bool) {
	q := normalization.NFC(query)
	if normalization.Fold(q) != strings.ToLower(q) {
		return query, "", false
	}
	loc := wordRe.FindAllStringIndex(q, -1)
	if len(loc) > maxRestoreWords {
		return query, "", false
	}
	pair := func(i int) ([2]string, bool) { // cặp (i, i+1) khôi phục được: hai từ đều được phép, cách nhau chỉ dấu cách
		if i+1 >= len(loc) || !restorable(q, loc[i]) || !restorable(q, loc[i+1]) || strings.Trim(q[loc[i][1]:loc[i+1][0]], " \t") != "" {
			return [2]string{}, false
		}
		return [2]string{strings.ToLower(q[loc[i][0]:loc[i][1]]), strings.ToLower(q[loc[i+1][0]:loc[i+1][1]])}, true
	}
	out := make([]string, len(loc)) // "" = giữ nguyên
	for i := range loc {
		k, ok := pair(i)
		if !ok {
			continue
		}
		counts := map[[2]string]int{}
		for d, v := range ix.vocabs() {
			if vis[d] {
				for form, n := range v[k] {
					counts[form] += n
				}
			}
		}
		best, ok := bestForm(counts)
		if !ok || out[i] != "" && out[i] != best[0] { // hoà điểm, hoặc mâu thuẫn từ cặp trước đã gán: không đoán
			continue
		}
		for j, w := range []string{best[0], best[1]} {
			if out[i+j] == "" {
				out[i+j] = w
			}
		}
	}
	var b strings.Builder
	var changes []string
	last := 0
	for i, l := range loc {
		orig, w := q[l[0]:l[1]], out[i]
		if w == "" || w == strings.ToLower(orig) {
			w = orig
		} else {
			if r, _ := utf8.DecodeRuneInString(orig); unicode.IsUpper(r) {
				first, n := utf8.DecodeRuneInString(w)
				w = string(unicode.ToUpper(first)) + w[n:]
			}
			changes = append(changes, orig+" → "+w)
		}
		b.WriteString(q[last:l[0]] + w)
		last = l[1]
	}
	if len(changes) == 0 {
		return query, "", false
	}
	return b.String() + q[last:], strings.Join(changes, ", "), true
}

// maxRestoreWords: câu dài hơn không khôi phục dấu — chi phí theo số cặp × số tài liệu, và câu hỏi thật hiếm khi dài vậy.
const maxRestoreWords = 64

// bestForm trả dạng gặp nhiều nhất; hoà (hai dạng cùng nhiều nhất) hoặc rỗng thì false — không đoán.
func bestForm(counts map[[2]string]int) ([2]string, bool) {
	var best [2]string
	top, ties := 0, 0
	for f, n := range counts {
		switch {
		case n > top:
			best, top, ties = f, n, 1
		case n == top:
			ties++
		}
	}
	return best, ties == 1
}
