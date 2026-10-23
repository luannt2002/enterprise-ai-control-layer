package pipeline

import (
	"context"
	"slices"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/luannt2002/enterprise-ai-control-layer/core/contracts"
	"github.com/luannt2002/enterprise-ai-control-layer/data/entityresolution"
	"github.com/luannt2002/enterprise-ai-control-layer/intelligence/intent"
	"github.com/luannt2002/enterprise-ai-control-layer/observability/traces"
)

// scopeRedactor che, ở MỘT chỗ cho mọi bề mặt của vết, khoá / mã / tên của thực thể NGOÀI phạm vi dữ liệu của người hỏi mà người hỏi không tự
// gõ (C9 vòng 2, soát C8–C11 C9-S1): vết do chính người hỏi đọc lại (GET /v1/trace/{id}, eais trace), nên với người không toàn quyền thì
// Decision.Entities, Decision.Slots và MỌI dòng lý do / bị loại của S2–S10 (kể cả dòng do intent.Classify, router, answerMetric hay
// answerKnowledge ghi) đều đi qua đây trước khi thành bản ghi. Vá từng dòng thì sót: hiệu "ứng viên trước lọc − sau lọc", danh sách "?a|b" ở
// S3.Output / S4.Input, dòng "khe … thiếu: câu nêu N thực thể (…)" của S4 đều từng lộ khoá ngoài phạm vi.
//
// Luật: chỉ thực thể đã xuất hiện trong lượt (Resolution, intent.Result, khe của kế hoạch, ứng viên câu hỏi lại) được xét — không duyệt cả danh
// mục mỗi lượt. Khoá / mã của thực thể mà câu hỏi có khoá đó (so theo entityresolution.Key: "dz03", "DZ 03" đều là "DZ-03") giữ nguyên: người
// hỏi đã biết khoá, và lý do từ chối cần nêu đúng thứ họ hỏi (chốt F4: khoá được kể, tên / loại thì không); TÊN vẫn bị che trừ khi câu chứa chính
// tên đó. Không tra được tổ tiên ⇒ che (đóng mặc định).
// Người có phạm vi dữ liệu không hạn chế: không che gì. Còn lộ được: SỐ ứng viên ngoài phạm vi (dấu thay thế giữ chỗ) và vị trí của dấu thay
// thế trong một danh sách đã sắp xếp; luồng hỏi trước nay vẫn kể số ứng viên bị loại.
type scopeRedactor struct {
	tokens []string // khoá, mã, tên cần che — dài trước ngắn sau, để "PRJ-AB" không bị che dở thành "‹…›B" bởi "PRJ-A"
}

// newScopeRedactor dựng bộ che cho một lượt; refs là mọi danh sách thực thể mà lượt đã thấy.
func (a *Ask) newScopeRedactor(ctx context.Context, pr contracts.Principal, question string, refs ...[]contracts.EntityRef) scopeRedactor {
	if a.RT.Policy.InScope(pr, nil) {
		return scopeRedactor{}
	}
	typed := " " + entityresolution.Key(question) + " "
	seen := map[string]bool{}
	var tokens []string
	for _, list := range refs {
		for _, r := range list {
			if r.ID == "" || seen[r.ID] {
				continue
			}
			seen[r.ID] = true
			if keys, err := entityresolution.Lineage(ctx, a.Deps.Directory, r.ID); err == nil && a.RT.Policy.InScope(pr, keys) {
				continue
			}
			var hide []string
			if k := entityresolution.Key(r.Key); k == "" || !strings.Contains(typed, " "+k+" ") {
				hide = append(hide, r.Key, r.ID)
			}
			if n := entityresolution.Key(r.Name); n == "" || !strings.Contains(typed, " "+n+" ") {
				hide = append(hide, r.Name) // tên không bao giờ kể cho người ngoài phạm vi, kể cả khi họ gõ đúng khoá (F4) — trừ khi họ gõ chính tên đó
			}
			for _, t := range hide {
				if t != "" && !slices.Contains(tokens, t) {
					tokens = append(tokens, t)
				}
			}
		}
	}
	slices.SortStableFunc(tokens, func(x, y string) int { return len(y) - len(x) })
	return scopeRedactor{tokens: tokens}
}

// entityRefs gom mọi thực thể mà một lượt đã thấy, để newScopeRedactor xét.
func entityRefs(res contracts.Resolution, cls intent.Result, ci clarifyInfo, plans ...contracts.RoutePlan) [][]contracts.EntityRef {
	out := [][]contracts.EntityRef{res.Entities, cls.Candidates, cls.Unclear.Entities, cls.Mentioned, ci.before, ci.shown}
	for _, m := range res.Ambiguous {
		out = append(out, m.Candidates)
	}
	for _, h := range ci.hidden {
		out = append(out, []contracts.EntityRef{h.ref})
	}
	for _, p := range plans {
		for _, s := range p.Intent.Slots {
			if s.Entity != nil {
				out = append(out, []contracts.EntityRef{*s.Entity})
			}
		}
	}
	return out
}

// line che mọi token trong một dòng; token chỉ khớp khi đứng riêng (trước và sau không phải chữ, số, '-' hay '_'), nên "EX-03" không che
// một phần của "EX-030".
func (r scopeRedactor) line(s string) string {
	for _, t := range r.tokens {
		s = replaceToken(s, t, traces.OutOfScope)
	}
	return s
}

// why che mọi dòng giải thích của mọi chặng.
func (r scopeRedactor) why(w traces.Why) traces.Why {
	if len(r.tokens) == 0 {
		return w
	}
	return w.Map(r.line)
}

// decision che Decision.Entities và Decision.Slots. Trong "?a|b|c" (cụm mơ hồ) các ứng viên bị che dồn về cuối, để thứ tự không gợi khoá.
func (r scopeRedactor) decision(d *traces.Decision) {
	if len(r.tokens) == 0 {
		return
	}
	for i, e := range d.Entities {
		if !strings.HasPrefix(e, "?") {
			d.Entities[i] = r.line(e)
			continue
		}
		var keep, hidden []string
		for _, c := range strings.Split(e[1:], "|") {
			if c2 := r.line(c); c2 != c {
				hidden = append(hidden, c2)
			} else {
				keep = append(keep, c)
			}
		}
		d.Entities[i] = "?" + strings.Join(append(keep, hidden...), "|")
	}
	for i, s := range d.Slots {
		d.Slots[i] = r.line(s)
	}
}

// replaceToken thay mọi lần token đứng riêng trong s bằng repl.
func replaceToken(s, token, repl string) string {
	if token == "" || !strings.Contains(s, token) {
		return s
	}
	var b strings.Builder
	i := 0
	for {
		j := strings.Index(s[i:], token)
		if j < 0 {
			break
		}
		j += i
		end := j + len(token)
		if boundaryBefore(s, j) && boundaryAfter(s, end) {
			b.WriteString(s[i:j])
			b.WriteString(repl)
			i = end
			continue
		}
		_, n := utf8.DecodeRuneInString(s[j:])
		b.WriteString(s[i : j+n])
		i = j + n
	}
	b.WriteString(s[i:])
	return b.String()
}

func keyRune(r rune) bool { return unicode.IsLetter(r) || unicode.IsDigit(r) || r == '-' || r == '_' }

func boundaryBefore(s string, i int) bool {
	if i == 0 {
		return true
	}
	r, _ := utf8.DecodeLastRuneInString(s[:i])
	return !keyRune(r)
}

func boundaryAfter(s string, i int) bool {
	if i >= len(s) {
		return true
	}
	r, _ := utf8.DecodeRuneInString(s[i:])
	return !keyRune(r)
}
