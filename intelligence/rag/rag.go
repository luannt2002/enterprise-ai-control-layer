// Package rag là capability TRI THỨC (CLAUDE.md §12): tài liệu → đoạn → chỉ mục → truy xuất →
// BẰNG CHỨNG có trích dẫn. RAG không quyết route, không quyết quyền nghiệp vụ, không là nguồn của
// con số: nó chỉ trả đoạn tài liệu mà người hỏi ĐƯỢC đọc (tenant + ACL theo vai).
package rag

import (
	"context"
	"encoding/json"
	"fmt"
	"maps"
	"math"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strings"
	"unicode"

	"github.com/luannt2002/enterprise-ai-control-layer/core/contracts"
	"github.com/luannt2002/enterprise-ai-control-layer/data/normalization"
)

// Document là một tài liệu tri thức (SOP, quy định) của MỘT tenant.
type Document struct {
	ID, Tenant, Title, Body string
	ACL                     []string // vai được đọc
}

type chunk struct {
	doc   *Document
	seq   int
	text  string
	terms map[string]bool
}

// MemoryIndex là chỉ mục tri thức trong bộ nhớ của MỘT tenant. Đoạn = một dòng của tài liệu (SOP
// viết theo bước); điểm = tổng IDF các từ (đã bỏ dấu) chung giữa câu hỏi và đoạn. Từ của đoạn gồm
// cả từ của TIÊU ĐỀ: bước 1 của "Quy trình xử lý chênh lệch nhiên liệu" không lặp lại chữ "quy
// trình" nhưng vẫn là một phần của quy trình đó.
type MemoryIndex struct {
	tenant string
	chunks []chunk
	idf    map[string]float64
	// MinScore: dưới ngưỡng này coi như không có đoạn nào trả lời được câu hỏi.
	MinScore float64
}

// stopWords là từ chức năng tiếng Việt — không mang nghĩa để so khớp.
var stopWords = []string{"là", "gì", "của", "và", "thì", "có", "không", "nào", "trong", "cho", "để", "theo", "khi",
	"phải", "được", "các", "một", "những", "với", "ở", "từ", "đến", "sau", "bao", "nhiêu"}

// questionPhrases là cụm NGHI VẤN — phần hỏi, không phải nội dung: "khi lệch phải làm gì" không
// được kéo về bước có chữ "làm việc".
var questionPhrases = []string{"làm gì", "là gì", "như thế nào", "thế nào", "ra sao", "bao lâu", "bao nhiêu", "ở đâu", "khi nào"}

// stopTone, stopFold: từ dừng theo khoá giữ dấu và theo dạng bỏ dấu (cho câu gõ không dấu).
var stopTone, stopFold = func() (map[string]bool, map[string]bool) {
	tone, fold := map[string]bool{}, map[string]bool{}
	for _, w := range stopWords {
		tone[normalization.ToneKey(w)], fold[normalization.Fold(w)] = true, true
	}
	return tone, fold
}()

// Khoá so khớp theo LUẬT DẤU của hệ thống: từ gõ có dấu chỉ khớp đúng từ đó ("bí" không khớp "bị";
// chỗ đặt dấu thanh không tính: "hoá" = "hóa"), từ gõ không dấu khớp mọi từ cùng dạng bỏ dấu.
// Chỉ mục giữ cả hai khoá cho mỗi từ ("=" giữ dấu, "~" bỏ dấu); câu hỏi dùng đúng một khoá mỗi từ.
func words(s string) []string {
	return strings.FieldsFunc(strings.ToLower(normalization.NFC(s)), func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsDigit(r)
	})
}

func indexTerms(s string) map[string]bool {
	out := map[string]bool{}
	for _, w := range words(s) {
		f, k := normalization.Fold(w), normalization.ToneKey(w)
		if stopFold[f] && (f == w || stopTone[k]) {
			continue
		}
		out["="+k], out["~"+f] = true, true
	}
	return out
}

func queryTerms(s string) map[string]bool {
	s = normalization.NFC(s)
	for _, p := range questionPhrases {
		for _, sp := range normalization.PhraseSpans(s, p) {
			s = s[:sp[0]] + strings.Repeat(" ", sp[1]-sp[0]) + s[sp[1]:] // giữ nguyên độ dài để các khoảng sau vẫn đúng
		}
	}
	out := map[string]bool{}
	for _, w := range words(s) {
		f, k := normalization.Fold(w), normalization.ToneKey(w)
		switch {
		case f == w: // gõ không dấu
			if !stopFold[f] {
				out["~"+f] = true
			}
		case !stopTone[k]:
			out["="+k] = true
		}
	}
	return out
}

// NewMemoryIndex dựng chỉ mục cho một tenant; tài liệu của tenant khác bị bỏ ngay ở đây.
func NewMemoryIndex(tenant string, docs []Document) *MemoryIndex {
	ix := &MemoryIndex{tenant: tenant, idf: map[string]float64{}, MinScore: 2.0}
	df := map[string]int{}
	for i := range docs {
		d := &docs[i]
		if d.Tenant != tenant {
			continue
		}
		for n, line := range strings.Split(d.Body, "\n") {
			if strings.TrimSpace(line) == "" {
				continue
			}
			c := chunk{doc: d, seq: n, text: strings.TrimSpace(line), terms: indexTerms(d.Title + " " + line)}
			for t := range c.terms {
				df[t]++
			}
			ix.chunks = append(ix.chunks, c)
		}
	}
	for t, n := range df {
		ix.idf[t] = math.Log(1 + float64(len(ix.chunks))/float64(n))
	}
	return ix
}

func allowed(d *Document, pr contracts.Principal) bool {
	for _, r := range pr.Roles {
		for _, a := range d.ACL {
			if r == a {
				return true
			}
		}
	}
	return false
}

// Retrieve trả tối đa k đoạn người hỏi được đọc, xếp theo điểm, chỉ những đoạn đạt MinScore.
// Tenant của principal phải khớp tenant của chỉ mục — sai thì lỗi chính sách, không trả gì.
func (ix *MemoryIndex) Retrieve(_ context.Context, pr contracts.Principal, query string, k int) ([]contracts.Evidence, error) {
	if string(pr.TenantID) != ix.tenant {
		return nil, contracts.Errorf(contracts.FaultPolicy, "principal không thuộc tenant của chỉ mục tri thức")
	}
	// Cộng điểm theo thứ tự từ cố định: duyệt map thì thứ tự cộng số thực đổi mỗi lần, điểm "bằng
	// nhau" lệch bit cuối và các đoạn hoà điểm đổi chỗ giữa hai lần hỏi.
	q := slices.Sorted(maps.Keys(queryTerms(query)))
	type scored struct {
		c *chunk
		s float64
	}
	var hits []scored
	for i := range ix.chunks {
		c := &ix.chunks[i]
		if !allowed(c.doc, pr) {
			continue
		}
		s := 0.0
		for _, t := range q {
			if c.terms[t] {
				s += ix.idf[t]
			}
		}
		if s >= ix.MinScore {
			hits = append(hits, scored{c, s})
		}
	}
	sort.SliceStable(hits, func(i, j int) bool {
		if hits[i].s != hits[j].s {
			return hits[i].s > hits[j].s
		}
		if hits[i].c.doc.ID != hits[j].c.doc.ID { // hoà điểm: theo tài liệu rồi vị trí dòng
			return hits[i].c.doc.ID < hits[j].c.doc.ID
		}
		return hits[i].c.seq < hits[j].c.seq
	})
	if len(hits) > k {
		hits = hits[:k]
	}
	out := make([]contracts.Evidence, 0, len(hits))
	for _, h := range hits {
		out = append(out, contracts.Evidence{
			ID: fmt.Sprintf("%s#%d", h.c.doc.ID, h.c.seq), Kind: contracts.EvidencePassage, Text: h.c.text,
			Source:     contracts.SourceRef{SourceType: "document", Name: h.c.doc.Title, DocumentID: h.c.doc.ID},
			Confidence: h.s,
		})
	}
	return out, nil
}

// LoadMarkdown đọc tài liệu của một tenant từ thư mục: docs/<id>.md (dòng đầu "# <tiêu đề>") +
// docs/acl.json ({"<id>": ["<vai>", ...]}). Tài liệu không có trong acl.json thì không ai đọc được.
func LoadMarkdown(dir, tenant string) ([]Document, error) {
	acl := map[string][]string{}
	if b, err := os.ReadFile(filepath.Join(dir, "acl.json")); err == nil {
		if err := json.Unmarshal(b, &acl); err != nil {
			return nil, fmt.Errorf("acl.json: %w", err)
		}
	}
	paths, _ := filepath.Glob(filepath.Join(dir, "*.md"))
	sort.Strings(paths)
	var out []Document
	for _, p := range paths {
		b, err := os.ReadFile(p)
		if err != nil {
			return nil, err
		}
		id := strings.TrimSuffix(filepath.Base(p), ".md")
		title, body, _ := strings.Cut(string(b), "\n")
		out = append(out, Document{ID: id, Tenant: tenant, Title: strings.TrimSpace(strings.TrimPrefix(title, "#")), Body: body, ACL: acl[id]})
	}
	return out, nil
}
