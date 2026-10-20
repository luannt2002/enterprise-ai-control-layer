// Package entityresolution là chặng S3: cụm từ trong câu → thực thể chuẩn của tenant.
//
// "máy 03", "Máy 3", "MX-03", "máy xúc số 03" là cách viết; thực thể là một máy xúc có
// khoá EX-03. Danh mục thực thể và bí danh là DỮ LIỆU của tenant (DB hoặc tệp giả lập);
// package này chỉ có luật so khớp dùng chung.
//
// Một bí danh khớp nhiều thực thể thì trả AmbiguousMention — không chọn bừa (P5).
// So khớp mờ (sai chính tả) CHƯA có: chưa có ca hỏng nào đòi nó (P7); bỏ dấu và bỏ ký tự
// nối đã phủ các cách viết đo được trong bộ ca.
package entityresolution

import (
	"context"
	"sort"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/luannt2002/enterprise-ai-systems/core/contracts"
	"github.com/luannt2002/enterprise-ai-systems/data/normalization"
)

// Alias là một cách viết tên thực thể, kèm nguồn của cách viết đó.
type Alias struct {
	Text   string
	Source string // "manual", "provider:FeedAlpha"…
}

// Entity là một thực thể trong danh mục của MỘT tenant.
type Entity struct {
	ID      string
	Type    string
	Key     string // khoá chuẩn, ví dụ "EX-03"
	Name    string
	Aliases []Alias
	// Parent là ID thực thể cha (máy thuộc dự án, trận thuộc giải, cầu thủ thuộc đội).
	// Rỗng = không có cha.
	Parent string
}

// Directory là danh mục thực thể của một tenant. Cài bằng DB khi chạy thật, bằng tệp giả
// lập trong test. Tenant đã nằm trong ngữ cảnh của Directory — Resolve không nhận tenant.
type Directory interface {
	Entities(ctx context.Context, types ...string) ([]Entity, error)
}

// Resolve tìm các thực thể thuộc các loại cho trước trong văn bản đã chuẩn hoá.
// types rỗng = mọi loại.
func Resolve(ctx context.Context, dir Directory, in contracts.NormalizedInput, types ...string) (contracts.Resolution, error) {
	entities, err := dir.Entities(ctx, types...)
	if err != nil {
		return contracts.Resolution{}, contracts.Label(contracts.Infra("directory"), err)
	}

	// khoá bí danh → các thực thể mang nó
	index := map[string][]contracts.EntityRef{}
	maxLen := 0
	add := func(alias string, e Entity) {
		k := Key(alias)
		if k == "" {
			return
		}
		for _, ex := range index[k] {
			if ex.ID == e.ID {
				return
			}
		}
		index[k] = append(index[k], contracts.EntityRef{ID: e.ID, Type: e.Type, Key: e.Key, Name: e.Name, Confidence: 1})
		if n := len(strings.Fields(k)); n > maxLen {
			maxLen = n
		}
	}
	for _, e := range entities {
		add(e.Key, e)
		add(e.Name, e)
		for _, a := range e.Aliases {
			add(a.Text, e)
		}
	}

	spans := keySpans(in.Raw)
	words := make([]string, len(spans))
	for i, s := range spans {
		words[i] = s.word
	}
	var res contracts.Resolution
	for i := 0; i < len(words); {
		matched := false
		// cụm dài nhất trước: "máy xúc 03" thắng "máy 03" thắng "03"
		for n := min(maxLen, len(words)-i); n >= 1; n-- {
			mention := strings.Join(words[i:i+n], " ")
			cands, ok := index[mention]
			if !ok {
				continue
			}
			if len(cands) == 1 {
				ref := cands[0]
				ref.Mention, ref.Start, ref.End = mention, spans[i].start, spans[i+n-1].end
				res.Entities = append(res.Entities, ref)
			} else {
				sorted := append([]contracts.EntityRef(nil), cands...)
				sort.Slice(sorted, func(a, b int) bool { return sorted[a].Key < sorted[b].Key })
				res.Ambiguous = append(res.Ambiguous, contracts.AmbiguousMention{
					Mention: mention, Type: commonType(sorted), Candidates: sorted,
					Start: spans[i].start, End: spans[i+n-1].end, // như cụm không mơ hồ: luồng hỏi cần biết "03" của "máy 03" là tên
				})
			}
			i += n
			matched = true
			break
		}
		if !matched {
			i++
		}
	}
	return res, nil
}

// keySpan là một từ của Key kèm vị trí byte của nó trong văn bản NFC gốc.
type keySpan struct {
	word       string
	start, end int
}

// keySpans tách s theo ĐÚNG luật của Key (bỏ dấu từng ký tự, tách chữ với số, bỏ số 0 đầu)
// nhưng giữ vị trí byte trong NFC(s) — để trả lại cụm nào của câu đã khớp thực thể.
func keySpans(s string) []keySpan {
	s = normalization.NFC(s)
	var out []keySpan
	var cur strings.Builder
	curDigit, start, last := false, 0, 0
	flush := func() {
		if cur.Len() == 0 {
			return
		}
		p := cur.String()
		if curDigit {
			p = strings.TrimLeft(p, "0")
			if p == "" {
				p = "0"
			}
		}
		out = append(out, keySpan{word: p, start: start, end: last})
		cur.Reset()
	}
	for i, r := range s {
		fr := []rune(normalization.Fold(string(r)))[0]
		digit, letter := unicode.IsDigit(fr), unicode.IsLetter(fr)
		if !digit && !letter {
			flush()
			continue
		}
		if cur.Len() > 0 && digit != curDigit {
			flush()
		}
		if cur.Len() == 0 {
			start = i
		}
		curDigit = digit
		cur.WriteRune(fr)
		last = i + utf8.RuneLen(r)
	}
	flush()
	return out
}

// Key đưa một cách viết về khoá so khớp: chữ thường, bỏ dấu, bỏ ký tự nối, tách chữ
// với số, bỏ số 0 đứng đầu. "MX-03" → "mx 3", "Máy  03" → "may 3", "EX03" → "ex 3".
func Key(s string) string {
	s = normalization.Fold(normalization.NFC(s))
	var parts []string
	var cur strings.Builder
	curDigit := false
	flush := func() {
		if cur.Len() == 0 {
			return
		}
		p := cur.String()
		if curDigit {
			p = strings.TrimLeft(p, "0")
			if p == "" {
				p = "0"
			}
		}
		parts = append(parts, p)
		cur.Reset()
	}
	for _, r := range s {
		switch {
		case unicode.IsDigit(r):
			if !curDigit {
				flush()
			}
			curDigit = true
			cur.WriteRune(r)
		case unicode.IsLetter(r):
			if curDigit {
				flush()
			}
			curDigit = false
			cur.WriteRune(r)
		default:
			flush()
		}
	}
	flush()
	return strings.Join(parts, " ")
}

func commonType(refs []contracts.EntityRef) string {
	if len(refs) == 0 {
		return ""
	}
	t := refs[0].Type
	for _, r := range refs[1:] {
		if r.Type != t {
			return ""
		}
	}
	return t
}

// Expand trả ID của thực thể và mọi thực thể con cháu của nó: hỏi "dự án A" thì số liệu
// gom từ mọi máy thuộc dự án A.
func Expand(ctx context.Context, dir Directory, id string) ([]string, error) {
	all, err := dir.Entities(ctx)
	if err != nil {
		return nil, contracts.Label(contracts.Infra("directory"), err)
	}
	children := map[string][]string{}
	for _, e := range all {
		if e.Parent != "" {
			children[e.Parent] = append(children[e.Parent], e.ID)
		}
	}
	out := []string{id}
	seen := map[string]bool{id: true}
	for i := 0; i < len(out); i++ {
		for _, c := range children[out[i]] {
			if !seen[c] {
				seen[c] = true
				out = append(out, c)
			}
		}
	}
	return out, nil
}

// Lineage trả khoá của thực thể và mọi tổ tiên của nó, gần trước xa sau: máy EX-03 →
// [EX-03, PRJ-A]. Dùng để kiểm phạm vi dữ liệu: được xem dự án thì được xem máy của dự án.
func Lineage(ctx context.Context, dir Directory, id string) ([]string, error) {
	all, err := dir.Entities(ctx)
	if err != nil {
		return nil, contracts.Label(contracts.Infra("directory"), err)
	}
	byID := map[string]Entity{}
	for _, e := range all {
		byID[e.ID] = e
	}
	var keys []string
	seen := map[string]bool{}
	for cur, ok := byID[id]; ok && !seen[cur.ID]; cur, ok = byID[cur.Parent] {
		seen[cur.ID] = true
		keys = append(keys, cur.Key)
	}
	return keys, nil
}

// MemoryDirectory là Directory trong bộ nhớ — cho test và cho dữ liệu giả lập.
type MemoryDirectory []Entity

// Entities trả thực thể thuộc các loại cho trước (rỗng = mọi loại).
func (m MemoryDirectory) Entities(_ context.Context, types ...string) ([]Entity, error) {
	if len(types) == 0 {
		return m, nil
	}
	var out []Entity
	for _, e := range m {
		for _, t := range types {
			if e.Type == t {
				out = append(out, e)
				break
			}
		}
	}
	return out, nil
}
