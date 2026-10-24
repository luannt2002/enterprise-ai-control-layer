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
	"regexp"
	"slices"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/luannt2002/enterprise-ai-control-layer/core/contracts"
	"github.com/luannt2002/enterprise-ai-control-layer/data/normalization"
	"go.yaml.in/yaml/v3"
)

// Trạng thái của một bản tài liệu. Chỉ bản đã duyệt mới được làm căn cứ. Trạng thái rỗng = tài liệu không
// khai phiên bản (nạp kiểu cũ), coi như đã duyệt.
const (
	StatusApproved    = "approved"
	StatusDraft       = "draft"
	StatusNeedsReview = "needs_review" // cổng chất lượng giữ lại, ví dụ bảng mất cột (CON-T-032)
	StatusSuperseded  = "superseded"
)

// Document là một tài liệu tri thức (SOP, quy định) của MỘT tenant. Cùng ID có thể có nhiều bản.
type Document struct {
	ID, Tenant, Title, Body string
	ACL                     []string // vai được đọc
	// Version là bản của tài liệu (≥ 1); 0 = tài liệu không khai phiên bản.
	Version int
	// Status là trạng thái của bản (Status*); rỗng = không khai, coi như đã duyệt.
	Status string
	// EffectiveFrom, EffectiveTo: hiệu lực theo NGÀY "2006-01-02", khoảng [từ, đến); rỗng = không giới hạn.
	// Chỉ mục so ngày theo chuỗi nên tin dạng ngày đã được LoadMarkdown kiểm; tự dựng Document thì phải đúng dạng.
	EffectiveFrom, EffectiveTo string
	// Scope là khoá thực thể (dự án…) được đọc; rỗng = chỉ xét ACL theo vai.
	Scope []string
	// Hold: lý do cổng chất lượng đọc giữ bản này lại (BR-DOC-03); rỗng = không bị giữ.
	Hold string
	// RemovedColumns: cột bảng chủ tài liệu bỏ CÓ CHỦ ĐÍCH so với bản trước — cổng không coi là mất cột.
	RemovedColumns []string
}

// Thông báo phiên bản là kiểu của contract (core/contracts), để pipeline xử lý mà không phụ thuộc vào gói này.
type (
	NoticeKind    = contracts.KnowledgeNoticeKind
	VersionInfo   = contracts.DocumentVersionInfo
	VersionNotice = contracts.KnowledgeNotice
)

const (
	NoticeConflict    = contracts.NoticeVersionConflict    // ≥ 2 bản cùng đang hiệu lực (CON-T-061)
	NoticeNoEffective = contracts.NoticeNoEffectiveVersion // không bản nào đã duyệt và đang hiệu lực (CON-T-032)
)

type chunk struct {
	doc   *Document
	seq   int
	text  string
	terms map[string]bool
	key   string // EmbedKey(embedText): khoá vector của đoạn, tính một lần lúc dựng
}

// MemoryIndex là chỉ mục tri thức trong bộ nhớ của MỘT tenant. Đoạn = một dòng của tài liệu (SOP
// viết theo bước); điểm = tổng IDF các từ (đã bỏ dấu) chung giữa câu hỏi và đoạn. Từ của đoạn gồm
// cả từ của TIÊU ĐỀ: bước 1 của "Quy trình xử lý chênh lệch nhiên liệu" không lặp lại chữ "quy
// trình" nhưng vẫn là một phần của quy trình đó.
type MemoryIndex struct {
	tenant string
	docs   []*Document
	chunks []chunk
	// MinScore: dưới ngưỡng này coi như không có đoạn nào trả lời được câu hỏi.
	MinScore float64
	// Now cho ngày HỎI (giờ của tenant) để xét hiệu lực, khi ngữ cảnh không mang ngày của lượt; nil = time.Now.
	Now func() time.Time
	// InScope báo người hỏi được đọc tài liệu có phạm vi keys không (ví dụ router.Policy.InScope), khi ngữ cảnh không mang luật
	// của lượt; nil = thuộc tính "entity_scope" của principal giao với keys.
	InScope func(pr contracts.Principal, keys []string) bool
	// Semantic, nếu khác nil, bật truy hồi lai (từ khoá + vector, xếp lại, ngưỡng "biết im" theo điểm); nil = chỉ từ khoá.
	Semantic *Semantic
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

// item là một đoạn của thân tài liệu: dòng thường, NHÓM "câu dẫn + các mục con" (câu kết thúc bằng ":" và các dòng a), b)… ngay
// sau), hoặc một hàng bảng kèm header của bảng đó (text rỗng = bảng chỉ có header). seq = vị trí dòng (đầu) trong thân; path = đường
// dẫn mục từ các dòng tiêu đề markdown ("Chương II › Mục 3 › Điều 18. …"), rỗng khi thân không có tiêu đề.
type item struct {
	seq, hseq          int // hseq: vị trí dòng header của bảng chứa hàng (chỉ có nghĩa khi table)
	text, header, path string
	table              bool
}

// headingRe: dòng tiêu đề markdown trong THÂN tài liệu (dòng đầu của tệp là tiêu đề tài liệu, đã tách ở parseDocText).
var headingRe = regexp.MustCompile(`^(#{1,6})\s+(\S.*)$`)

// Loại dấu đầu dòng của mục con, theo thứ tự lồng: số "1." < chữ "a)" < la mã "(i)" < gạch "-" < cộng "+". Văn bản tiếng Việt
// đánh điểm a), b), … đ), e), … i), k) — nên "i)" là chữ; la mã chỉ nhận dạng có ngoặc hai bên ("(xii)").
var markerRes = []*regexp.Regexp{
	regexp.MustCompile(`^\d{1,3}[.)]\s`),
	regexp.MustCompile(`^[a-zđ]\)\s`),
	regexp.MustCompile(`^\([ivx]+\)\s`),
	regexp.MustCompile(`^[-*•–]\s`),
	regexp.MustCompile(`^\+\s`),
}

// markerRank: 1..5 theo markerRes, 0 = dòng không có dấu đầu dòng.
func markerRank(line string) int {
	for i, re := range markerRes {
		if re.MatchString(line) {
			return i + 1
		}
	}
	return 0
}

// maxGroup là trần độ dài (ký tự) của một nhóm câu dẫn + mục con; dài hơn thì tách thành nhiều đoạn, đoạn nào cũng mở bằng câu dẫn.
// shortcut: đếm ký tự, không đếm token; đo bằng tokenizer thật khi có ngân sách ngữ cảnh (ROADMAP-RAG §6.5).
const maxGroup = 1200

// parseBody là luật DUY NHẤT nhận bảng (chỉ mục và cổng chất lượng cùng dùng): chỉ là bảng khi dòng "|" có dòng
// phân cách ngay sau; dòng phân cách không bao giờ là một dòng có chữ; hàng có dòng phân cách ngay sau là header của
// bảng kế tiếp viết liền.
// shortcut: mỗi hàng một đoạn; bảng rất dài / rất rộng cần gom nhóm hàng theo ngân sách token khi có tài liệu thật.
func parseBody(body string) []item {
	lines := strings.Split(body, "\n")
	// delimAfter: dòng i là dòng phân cách của header h (luật GFM: ngay sau header, cùng số ô). Dòng toàn gạch ở
	// chỗ khác là hàng dữ liệu ("| - | - |" = không áp dụng).
	delimAfter := func(i int, h string) bool {
		if i >= len(lines) {
			return false
		}
		l := strings.TrimSpace(lines[i])
		return isTableSeparator(l) && len(cells(l)) == len(cells(h))
	}
	var out []item
	var stack []string // tiêu đề đang mở theo cấp
	path := ""
	join := func(s []string) string {
		return strings.Join(slices.DeleteFunc(slices.Clone(s), func(x string) bool { return x == "" }), " › ")
	}
	// Tiêu đề gần nhất chưa có đoạn nào bên dưới: tới tiêu đề cùng cấp / cao hơn hoặc hết thân mà vẫn trống thì chính nó thành đoạn
	// (đường dẫn = các tiêu đề cha) — "Điều 10. (Đã bãi bỏ)" không được biến mất khỏi chỉ mục. Có tiêu đề con thì nó nằm trong
	// đường dẫn của con, không cần đoạn riêng.
	pend, pendOut, pendLevel, pendText, pendPath := -1, 0, 0, "", ""
	// Bên dưới chỉ có hàng bảng cũng tính là trống: hàng bảng không chấm đường dẫn, nên tiêu đề phải là đoạn riêng để tìm được.
	flush := func(level int) {
		if pend >= 0 && level <= pendLevel && !slices.ContainsFunc(out[pendOut:], func(it item) bool { return !it.table }) {
			out = slices.Insert(out, pendOut, item{seq: pend, text: pendText, path: pendPath})
		}
	}
	for n := 0; n < len(lines); n++ {
		line := strings.TrimSpace(lines[n])
		if m := headingRe.FindStringSubmatch(line); m != nil { // tiêu đề: cập nhật đường dẫn mục, không là đoạn riêng
			level := len(m[1])
			flush(level)
			for len(stack) < level-1 {
				stack = append(stack, "")
			}
			stack = append(stack[:level-1], strings.TrimSpace(m[2]))
			path = join(stack)
			pend, pendOut, pendLevel, pendText, pendPath = n, len(out), level, stack[level-1], join(stack[:level-1])
			continue
		}
		switch {
		case line == "" || (isTableSeparator(line) && !delimAfter(n+1, line)): // dòng phân cách lẻ không là đoạn
			continue
		case !strings.HasPrefix(line, "|") || !delimAfter(n+1, line):
			if g, last := groupList(lines, n, line, path); g != nil {
				out, n = append(out, g...), last
				continue
			}
			out = append(out, item{seq: n, text: line, path: path})
			continue
		}
		header, hseq, rows := line, n, 0
		n++ // dòng phân cách của header
		for n+1 < len(lines) && strings.HasPrefix(strings.TrimSpace(lines[n+1]), "|") {
			n++
			row := strings.TrimSpace(lines[n])
			// Hàng có dòng phân cách "---" ngay sau: header của bảng kế tiếp viết liền. Giữa thân bảng, dòng chỉ có "-"
			// (không áp dụng) là hàng dữ liệu, nên chỉ nhận dòng phân cách mà mọi ô có ≥ 3 gạch (thói quen viết, không phải
			// luật GFM; GFM không có bảng viết liền).
			if delimAfter(n+1, row) && longDelim(lines[n+1]) {
				if rows == 0 {
					out = append(out, item{seq: hseq, hseq: hseq, header: header, path: path, table: true})
				}
				header, hseq, rows = row, n, 0
				n++
				continue
			}
			out = append(out, item{seq: n, hseq: hseq, text: row, header: header, path: path, table: true})
			rows++
		}
		if rows == 0 {
			out = append(out, item{seq: hseq, hseq: hseq, header: header, path: path, table: true})
		}
	}
	flush(0)
	return out
}

// groupList gom câu dẫn (dòng kết thúc bằng ":") với các mục con ngay sau nó — mục cấp đầu (loại dấu đầu dòng sâu hơn của câu dẫn)
// kèm mọi mục lồng sâu hơn bên dưới — thành đoạn: "…các tiêu chí sau:" tách khỏi điểm a)–đ) thì không đoạn nào trả lời được câu hỏi
// liệt kê. Quá trần thì chỉ tách giữa hai mục cấp đầu (nhóm con không bị cắt đôi). Trả nil khi dòng n không phải câu dẫn có mục con;
// last = dòng cuối đã gom.
// shortcut: một mục cấp đầu cùng nhóm con dài hơn trần vẫn để nguyên một đoạn; cắt theo token khi có ngân sách ngữ cảnh (§6.3 điểm 2).
func groupList(lines []string, n int, lead, path string) ([]item, int) {
	if !strings.HasSuffix(lead, ":") && !strings.HasSuffix(lead, "：") {
		return nil, n
	}
	leadRank, rank := markerRank(lead), 0
	var kids []int
	for j := n + 1; j < len(lines); j++ {
		l := strings.TrimSpace(lines[j])
		if l == "" {
			continue
		}
		r := markerRank(l)
		if rank == 0 {
			if r <= leadRank {
				break
			}
			rank = r
		}
		if r < rank { // dòng thường (0) hoặc nông hơn mục cấp đầu: hết danh sách
			break
		}
		kids = append(kids, j)
	}
	if len(kids) == 0 {
		return nil, n
	}
	type block struct {
		seq  int
		text string
	}
	var blocks []block // mỗi mục cấp đầu kèm các mục lồng của nó
	for _, j := range kids {
		l := strings.TrimSpace(lines[j])
		if markerRank(l) == rank {
			blocks = append(blocks, block{j, l})
		} else {
			blocks[len(blocks)-1].text += "\n" + l
		}
	}
	var out []item
	seq, text := n, lead
	for _, b := range blocks {
		if text != lead && utf8.RuneCountInString(text)+1+utf8.RuneCountInString(b.text) > maxGroup {
			out = append(out, item{seq: seq, text: text, path: path})
			seq, text = b.seq, lead
		}
		text += "\n" + b.text
	}
	return append(out, item{seq: seq, text: text, path: path}), kids[len(kids)-1]
}

// longDelim: mọi ô của dòng phân cách có ít nhất 3 gạch ("---", ":---:").
func longDelim(line string) bool {
	for _, c := range cells(line) {
		if len(strings.Trim(c, ":")) < 3 {
			return false
		}
	}
	return true
}

// cells tách một dòng bảng thành các ô: bỏ đúng một "|" ở mỗi mép; "\|" là chữ trong ô, không phải ranh giới.
func cells(row string) []string {
	row = strings.TrimPrefix(strings.TrimSpace(row), "|")
	if strings.HasSuffix(row, "|") && !strings.HasSuffix(row, "\\|") {
		row = row[:len(row)-1]
	}
	var out []string
	var b strings.Builder
	for i := 0; i < len(row); i++ {
		switch {
		case row[i] == '\\' && i+1 < len(row) && row[i+1] == '|':
			b.WriteByte('|')
			i++
		case row[i] == '|':
			out = append(out, strings.TrimSpace(b.String()))
			b.Reset()
		default:
			b.WriteByte(row[i])
		}
	}
	return append(out, strings.TrimSpace(b.String()))
}

// columnName chuẩn hoá tên cột để so giữa các bản: NFC, gộp khoảng trắng.
func columnName(c string) string { return strings.Join(strings.Fields(normalization.NFC(c)), " ") }

// qualityGate áp BR-DOC-03 lúc nạp ("bảng đủ ô, số đủ"; không đạt thì không công bố, giữ bản cũ). Bản khai "đã
// duyệt" bị chuyển "chờ soát" kèm lý do khi:
//   - một hàng thiếu / thừa ô so với header, một ô rỗng (ô trống có chủ đích phải ghi "-"), hoặc bảng chỉ còn header;
//   - mất cột so với MỐC: số lần mỗi cột xuất hiện, đếm MỘT lần mỗi bảng, giảm (trừ cột khai trong removed_columns);
//   - removed_columns khai một cột không có ở mốc (khai sai tên hoặc chép sót từ bản trước).
//
// MỐC = bản đã công bố gần nhất (đã duyệt hoặc đã thay thế) cùng mã, bảng của nó đạt kiểm, và phạm vi của nó bao trùm
// phạm vi bản mới (không phạm vi = mọi người): người đọc được bản mới thì đọc được mốc, nên lý do giữ không lộ gì.
// Nới rộng phạm vi mà không có mốc bao trùm thì không so được (giới hạn đã biết). Nháp, tài liệu không phiên bản: bỏ.
// shortcut: cổng chạy lại mỗi lần nạp, không nhớ bản nào đã công bố; nhập một bản cũ hơn, sửa một bản cũ, hay làm
// luật cổng chặt hơn đều có thể hạ một bản đang công bố. Khi nạp vào Postgres (U6) phải lưu trạng thái công bố.
func qualityGate(docs []Document) {
	idx := make([]int, len(docs))
	for i := range idx {
		idx[i] = i
	}
	slices.SortFunc(idx, func(a, b int) int {
		if c := strings.Compare(docs[a].ID, docs[b].ID); c != 0 {
			return c
		}
		return docs[a].Version - docs[b].Version
	})
	type moc struct {
		scope []string
		cols  map[string]int
	}
	mocs := map[string][]moc{} // mã → các bản đã công bố đạt kiểm, theo thứ tự bản
	for _, i := range idx {
		d := &docs[i]
		if d.Version == 0 || (d.Status != StatusApproved && d.Status != StatusSuperseded) {
			continue
		}
		cols := map[string]int{}
		var bad []string
		last := -1
		for _, it := range parseBody(d.Body) {
			if !it.table {
				continue
			}
			h := cells(it.header)
			if it.hseq != last { // đếm cột một lần mỗi bảng, không theo số hàng
				for _, c := range h {
					cols[columnName(c)]++
				}
				last = it.hseq
			}
			row := cells(it.text)
			switch {
			case it.text == "":
				bad = append(bad, fmt.Sprintf("bảng #%d chỉ có header", it.seq))
			case len(row) < len(h):
				bad = append(bad, fmt.Sprintf("hàng #%d thiếu ô (%d/%d)", it.seq, len(row), len(h)))
			case len(row) > len(h):
				bad = append(bad, fmt.Sprintf("hàng #%d thừa ô (%d/%d)", it.seq, len(row), len(h)))
			case slices.Contains(row, ""):
				bad = append(bad, fmt.Sprintf("hàng #%d có ô rỗng", it.seq))
			}
		}
		var base map[string]int
		for j := len(mocs[d.ID]) - 1; j >= 0; j-- {
			if m := mocs[d.ID][j]; len(m.scope) == 0 || (len(d.Scope) > 0 && !slices.ContainsFunc(d.Scope, func(s string) bool { return !slices.Contains(m.scope, s) })) {
				base = m.cols
				break
			}
		}
		var why []string
		if len(bad) > 0 {
			why = append(why, fmt.Sprintf("bảng không đủ ô / số (%d chỗ): %s", len(bad), strings.Join(bad[:min(3, len(bad))], "; ")))
		}
		var lost, stale []string
		for c, n := range base {
			if cols[c] < n && !slices.ContainsFunc(d.RemovedColumns, func(r string) bool { return columnName(r) == c }) {
				lost = append(lost, c)
			}
		}
		for _, r := range d.RemovedColumns {
			if base[columnName(r)] == 0 {
				stale = append(stale, r)
			}
		}
		if len(lost) > 0 {
			slices.Sort(lost)
			why = append(why, fmt.Sprintf("bảng mất cột so với bản trước (%d): %s", len(lost), strings.Join(lost[:min(5, len(lost))], ", ")))
		}
		if len(stale) > 0 {
			why = append(why, "removed_columns khai cột không có ở bản trước: "+strings.Join(stale, ", "))
		}
		switch {
		case len(why) == 0:
			mocs[d.ID] = append(mocs[d.ID], moc{d.Scope, cols})
		case d.Status == StatusApproved:
			d.Status, d.Hold = StatusNeedsReview, strings.Join(why, "; ")
		}
		// Bản đã thay thế không đạt kiểm: không giữ (đã không còn phục vụ) nhưng không làm mốc.
	}
}

// isTableSeparator: dòng phân cách header của bảng markdown theo luật GFM: mỗi ô dạng ":?-+:?" ("|---|:--:|").
// "| | |" là một hàng rỗng, không phải dòng phân cách.
func isTableSeparator(line string) bool {
	if !strings.HasPrefix(line, "|") {
		return false
	}
	for _, c := range cells(line) {
		c = strings.TrimSuffix(strings.TrimPrefix(c, ":"), ":")
		if c == "" || strings.Trim(c, "-") != "" {
			return false
		}
	}
	return true
}

// NewMemoryIndex dựng chỉ mục cho một tenant; tài liệu của tenant khác bị bỏ ngay ở đây.
func NewMemoryIndex(tenant string, docs []Document) *MemoryIndex {
	ix := &MemoryIndex{tenant: tenant, MinScore: 2.0}
	for i := range docs {
		d := &docs[i]
		if d.Tenant != tenant {
			continue
		}
		ix.docs = append(ix.docs, d)
		// scored: phần chữ được chấm điểm (cùng tiêu đề); text: phần chữ đi vào bằng chứng.
		add := func(seq int, text, scored string) {
			ix.chunks = append(ix.chunks, chunk{doc: d, seq: seq, text: text, terms: indexTerms(d.Title + " " + scored), key: EmbedKey(d.Title + ": " + text)})
		}
		for _, it := range parseBody(d.Body) {
			// Đường dẫn mục đứng đầu đoạn và được chấm điểm cùng chữ của đoạn: hỏi "Điều 18 quy định gì" thì khoản của Điều 18 không
			// có chữ "Điều 18" nhưng đường dẫn có (U3b). Riêng hàng bảng không chấm đường dẫn — như header, nó chung cho mọi hàng.
			pre := ""
			if it.path != "" {
				pre = it.path + "\n"
			}
			switch {
			case !it.table:
				add(it.seq, pre+it.text, it.path+" "+it.text)
			case it.text == "": // bảng chỉ có header
				add(it.seq, pre+it.header, it.path+" "+it.header)
			default:
				// Hàng bảng: đoạn mang header (hàng đứng một mình là số không đơn vị — CON-T-005) nhưng chỉ chữ của hàng
				// được chấm điểm: header chung cho mọi hàng, cộng điểm thì câu chạm từ của header kéo mọi hàng vượt ngưỡng.
				add(it.seq, pre+it.header+"\n"+it.text, it.text)
			}
		}
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

// visible: người hỏi được đọc tài liệu (vai trong ACL, và phạm vi nếu tài liệu khai). Tài liệu không thấy
// thì coi như không tồn tại. Tài liệu có phạm vi: ACL và phạm vi phải thoả bởi CÙNG một vai — vai không được đọc
// tài liệu (vd accountant, scope "all" cho số liệu) không cho vai khác mượn phạm vi của nó.
func (ix *MemoryIndex) visible(d *Document, pr contracts.Principal) bool {
	if len(d.Scope) == 0 {
		return allowed(d, pr)
	}
	for _, r := range pr.Roles {
		if !slices.Contains(d.ACL, r) {
			continue
		}
		one := pr
		one.Roles = []string{r}
		if ix.InScope != nil && ix.InScope(one, d.Scope) {
			return true
		}
		if ix.InScope == nil {
			for _, k := range strings.Split(pr.Attributes["entity_scope"], ",") {
				if k = strings.TrimSpace(k); k != "" && slices.Contains(d.Scope, k) {
					return true
				}
			}
		}
	}
	return false
}

// servable: bản đã duyệt (hoặc không khai) và đang hiệu lực vào ngày today ("2006-01-02").
func servable(d *Document, today string) bool {
	return (d.Status == "" || d.Status == StatusApproved) &&
		(d.EffectiveFrom == "" || today >= d.EffectiveFrom) && (d.EffectiveTo == "" || today < d.EffectiveTo)
}

func (ix *MemoryIndex) today() string {
	if ix.Now != nil {
		return ix.Now().Format(dateLayout)
	}
	// shortcut: không Now, ngữ cảnh không mang ngày hỏi (gọi ngoài pipeline) thì lấy đồng hồ máy (múi giờ máy chủ); pipeline
	// luôn đặt ngày hỏi (contracts.WithKnowledgeQuery) nên đường hỏi thật không rơi vào đây.
	return time.Now().Format(dateLayout)
}

// notice dựng thông báo cho tài liệu id từ mọi bản người hỏi thấy.
func (ix *MemoryIndex) notice(id string, kind NoticeKind, pr contracts.Principal) *VersionNotice {
	n := &VersionNotice{Kind: kind, DocumentID: id}
	top := -1
	for _, d := range ix.docs {
		if d.ID == id && ix.visible(d, pr) {
			if d.Version > top {
				n.Title, top = d.Title, d.Version
			}
			n.Versions = append(n.Versions, VersionInfo{Version: d.Version, Status: d.Status, EffectiveFrom: d.EffectiveFrom, EffectiveTo: d.EffectiveTo, Hold: d.Hold})
		}
	}
	slices.SortFunc(n.Versions, func(a, b VersionInfo) int { return a.Version - b.Version })
	return n
}

// Retrieve trả tối đa k đoạn người hỏi được đọc, xếp theo điểm, chỉ những đoạn đạt MinScore.
// Tenant của principal phải khớp tenant của chỉ mục — sai thì lỗi chính sách, không trả gì.
//
// Phiên bản: chỉ đoạn của bản đã duyệt, đang hiệu lực vào ngày hỏi mới là bằng chứng. Nếu tài liệu khớp
// câu hỏi nhất (điểm ≥ đoạn tốt nhất còn lại) không có ĐÚNG MỘT bản như thế thì trả *VersionNotice thay
// cho đoạn: hai bản cùng hiệu lực là xung đột, không chọn ngầm; không bản nào thì không lấy số của bản cũ.
// Tài liệu ngoài vai / phạm vi không góp thông báo nào: "không có" và "không được xem" ra cùng một kết
// quả, người hỏi không dò được tài liệu nào tồn tại.
func (ix *MemoryIndex) Retrieve(ctx context.Context, pr contracts.Principal, query string, k int) ([]contracts.Evidence, error) {
	if string(pr.TenantID) != ix.tenant {
		return nil, contracts.Errorf(contracts.FaultPolicy, "principal không thuộc tenant của chỉ mục tri thức")
	}
	// Ngày hỏi và luật phạm vi của LƯỢT thắng trường của chỉ mục: phát lại một lượt cũ phải ra bản của ngày đó.
	topic := ""
	if kq, ok := contracts.KnowledgeQueryFrom(ctx); ok {
		topic = kq.Topic
		c := *ix // bản sao nông: chỉ mục dùng chung giữa các lượt, không sửa tại chỗ
		if !kq.AsOf.IsZero() {
			c.Now = func() time.Time { return kq.AsOf }
		}
		if kq.InScope != nil {
			c.InScope = kq.InScope
		}
		ix = &c
	}
	today := ix.today()
	serving := map[string]int{} // id → số bản phục vụ được, trong các bản người hỏi thấy
	live := map[string]bool{}   // id có bản thấy được mà chưa bị thay thế
	vis := map[*Document]bool{}
	for _, d := range ix.docs {
		if vis[d] = ix.visible(d, pr); !vis[d] {
			continue
		}
		if servable(d, today) {
			serving[d.ID]++
		}
		live[d.ID] = live[d.ID] || d.Status != StatusSuperseded
	}
	// Cộng điểm theo thứ tự từ cố định: duyệt map thì thứ tự cộng số thực đổi mỗi lần, điểm "bằng
	// nhau" lệch bit cuối và các đoạn hoà điểm đổi chỗ giữa hai lần hỏi.
	// IDF chỉ trên đoạn người hỏi THẤY: điểm và ngưỡng MinScore không được phụ thuộc tài liệu người hỏi không được đọc —
	// không thì thứ hạng mang dấu vết của chúng, và đường PG (chỉ nạp tài liệu qua ACL) lệch đường bộ nhớ (S7D-25).
	df, seen := map[string]int{}, 0
	for i := range ix.chunks {
		if vis[ix.chunks[i].doc] {
			seen++
			for t := range ix.chunks[i].terms {
				df[t]++
			}
		}
	}
	rep := report(ctx)
	*rep = contracts.RetrievalReport{Path: "từ khoá", Chunks: seen}
	if ix.Semantic != nil {
		rep.Path = "lai, cosine"
		if ix.Semantic.Reranker != nil {
			rep.Path = "lai, xếp lại"
		}
	}
	for d, v := range vis {
		if v {
			rep.Docs++
			if !servable(d, today) {
				rep.NotServable++
			}
		}
	}
	if ix.Semantic != nil && ix.Semantic.MinScore > 0 {
		return ix.hybrid(ctx, pr, query, k, today, vis, serving, live, df, seen)
	}
	// Kho không nói tới điều được hỏi: từ nửa số từ nội dung của CHỦ ĐỀ trở lên (câu bỏ con số / ngày / mã — pipeline tách) không
	// có trong đoạn nào người hỏi thấy ⇒ không đoạn nào là bằng chứng, dù một chữ trùng hợp vượt MinScore ("tục" trong "liên tục").
	// Không thì câu trả lời chép nguyên văn một đoạn lạc đề và qua được bước kiểm (S7-BO-CA-DOC-LAP.md §2, "Biết im").
	if tq := queryTerms(topic); len(tq) > 0 {
		unknown := 0
		for t := range tq {
			if df[t] == 0 {
				unknown++
			}
		}
		if 2*unknown >= len(tq) {
			rep.Silence = fmt.Sprintf("chủ đề ngoài kho: %d / %d từ của chủ đề không có trong đoạn nào người hỏi thấy", unknown, len(tq))
			return []contracts.Evidence{}, nil
		}
	}
	if ix.Semantic != nil { // đường lai chưa đặt ngưỡng điểm: đã qua luật "biết im" theo chủ đề ở trên
		return ix.hybrid(ctx, pr, query, k, today, vis, serving, live, df, seen)
	}
	q := slices.Sorted(maps.Keys(queryTerms(query)))
	type scored struct {
		c *chunk
		s float64
	}
	var hits []scored
	noticeID, noticeScore := "", -1.0
	for i := range ix.chunks {
		c := &ix.chunks[i]
		if !vis[c.doc] {
			continue
		}
		s := idfScore(c, q, df, seen)
		if s < ix.MinScore {
			continue
		}
		switch id := c.doc.ID; {
		case serving[id] == 1 && servable(c.doc, today):
			hits = append(hits, scored{c, s})
		case serving[id] != 1 && live[id] && (s > noticeScore || s == noticeScore && id < noticeID):
			noticeID, noticeScore = id, s
		}
		// Còn lại: bản cũ / chưa duyệt của tài liệu ĐÃ có một bản làm căn cứ, hoặc tài liệu mà mọi bản đều đã
		// được thay thế (thay bằng tài liệu khác; báo thì che mất tài liệu mới) → bỏ, không báo.
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
	if noticeID != "" && (len(hits) == 0 || noticeScore >= hits[0].s) {
		kind := NoticeNoEffective
		if serving[noticeID] > 1 {
			kind = NoticeConflict
		}
		return nil, ix.notice(noticeID, kind, pr)
	}
	if len(hits) == 0 {
		rep.Silence = fmt.Sprintf("không đoạn nào đạt ngưỡng từ khoá %.3f", ix.MinScore)
	}
	if len(hits) > k {
		rep.Beyond, hits = len(hits)-k, hits[:k]
	}
	out := make([]contracts.Evidence, 0, len(hits))
	for _, h := range hits {
		out = append(out, evidenceOf(h.c, h.s))
	}
	return out, nil
}

// report trả nơi ghi vết truy xuất của lượt (KnowledgeQuery.Report); lượt không xin vết thì ghi vào một bản bỏ đi.
func report(ctx context.Context) *contracts.RetrievalReport {
	if kq, ok := contracts.KnowledgeQueryFrom(ctx); ok && kq.Report != nil {
		return kq.Report
	}
	return &contracts.RetrievalReport{}
}

// idfScore là điểm từ khoá của đoạn: tổng IDF các từ của câu hỏi có trong đoạn (q đã sắp xếp — thứ tự cộng cố định).
func idfScore(c *chunk, q []string, df map[string]int, seen int) float64 {
	s := 0.0
	for _, t := range q {
		if c.terms[t] {
			s += math.Log(1 + float64(seen)/float64(df[t]))
		}
	}
	return s
}

// evidenceOf dựng bằng chứng của đoạn, mã tất định "mã@vN#vị-trí" (tài liệu không phiên bản: "mã#vị-trí").
func evidenceOf(c *chunk, score float64) contracts.Evidence {
	id := fmt.Sprintf("%s#%d", c.doc.ID, c.seq) // tài liệu không phiên bản: giữ đúng dạng mã cũ
	if c.doc.Version > 0 {
		id = fmt.Sprintf("%s@v%d#%d", c.doc.ID, c.doc.Version, c.seq)
	}
	return contracts.Evidence{
		ID: id, Kind: contracts.EvidencePassage, Text: c.text,
		Source: contracts.SourceRef{SourceType: "document", Name: c.doc.Title, DocumentID: c.doc.ID,
			DocumentVersion: c.doc.Version},
		Confidence: score,
	}
}

// LoadMarkdown đọc tài liệu của một tenant từ thư mục: docs/<tệp>.md (dòng đầu "# <tiêu đề>") +
// docs/acl.json ({"<id>": ["<vai>", ...]}). Tài liệu không có trong acl.json thì không ai đọc được.
//
// Tệp có thể mở đầu bằng phần đầu giữa hai dòng "---" (id, version, status, effective_from, effective_to,
// scope); khi đó id lấy từ phần đầu, các bản cùng id dùng chung ACL. Tệp không có phần đầu nạp như cũ: id =
// tên tệp, không phiên bản. Phần đầu sai (khoá lạ, trạng thái lạ, ngày sai, thiếu bản, trùng id@bản) thì
// báo lỗi: tài liệu khai sai không được lặng lẽ công bố.
func LoadMarkdown(dir, tenant string) ([]Document, error) {
	out, err := loadMarkdown(dir, tenant)
	if err != nil {
		return nil, err
	}
	qualityGate(out)
	return out, nil
}

// loadMarkdown là LoadMarkdown chưa chạy cổng chất lượng (Collect chạy cổng một lần trên cả kho).
func loadMarkdown(dir, tenant string) ([]Document, error) {
	acl := map[string][]string{}
	if b, err := os.ReadFile(filepath.Join(dir, "acl.json")); err == nil {
		if err := json.Unmarshal(b, &acl); err != nil {
			return nil, fmt.Errorf("acl.json: %w", err)
		}
	}
	paths, _ := filepath.Glob(filepath.Join(dir, "*.md"))
	sort.Strings(paths)
	var out []Document
	seen := map[string]string{}    // "id@bản" → tệp
	versioned := map[string]bool{} // id → tệp đầu tiên của id là tệp không phiên bản
	for _, p := range paths {
		b, err := os.ReadFile(p)
		if err != nil {
			return nil, err
		}
		d, _, err := parseDocText(p, string(b), tenant, true)
		if err != nil {
			return nil, err
		}
		key := fmt.Sprintf("%s@%d", d.ID, d.Version)
		if prev, dup := seen[key]; dup {
			return nil, fmt.Errorf("%s và %s cùng là tài liệu %s bản %d", prev, p, d.ID, d.Version)
		}
		seen[key] = p
		// Bản không phiên bản luôn phục vụ được: để chung id với bản có phiên bản là xung đột vĩnh viễn.
		if legacy, ok := versioned[d.ID]; ok && legacy != (d.Version == 0) {
			return nil, fmt.Errorf("%s: tài liệu %s vừa có tệp không phiên bản vừa có tệp khai phiên bản", p, d.ID)
		}
		versioned[d.ID] = d.Version == 0
		d.ACL = acl[d.ID]
		out = append(out, d)
	}
	return out, nil
}

// parseDocText dựng tài liệu từ chữ của MỘT tệp: mã = tên tệp bỏ đuôi; khi allowFM, phần đầu giữa hai dòng "---" (nếu có, fm = true)
// khai mã / bản / trạng thái / hiệu lực / phạm vi; dòng đầu của thân là tiêu đề ("# " tuỳ chọn). Chưa gắn ACL.
func parseDocText(p, raw, tenant string, allowFM bool) (d Document, fm bool, err error) {
	d = Document{ID: strings.TrimSuffix(filepath.Base(p), filepath.Ext(p)), Tenant: tenant}
	// Tệp lưu kèm BOM hay CRLF vẫn phải nhận ra phần đầu: lỡ thì bản nháp nạp như tài liệu không phiên bản
	// (coi như đã duyệt) và bị phục vụ.
	text := strings.ReplaceAll(strings.TrimPrefix(raw, "\ufeff"), "\r\n", "\n")
	if rest, ok := strings.CutPrefix(text, "---\n"); ok && allowFM {
		head, body, closed := "", "", false
		if body, closed = strings.CutPrefix(rest, "---\n"); !closed {
			head, body, closed = strings.Cut(rest, "\n---\n")
		}
		if !closed {
			return d, false, fmt.Errorf("%s: phần đầu thiếu dòng đóng \"---\"", p)
		}
		f, err := parseFrontMatter(head)
		if err != nil {
			return d, false, fmt.Errorf("%s: %w", p, err)
		}
		if f.ID != "" {
			d.ID = f.ID
		}
		d.Version, _ = strconv.Atoi(f.Version) // parseFrontMatter đã kiểm
		d.Status, d.EffectiveFrom, d.EffectiveTo, d.Scope, d.RemovedColumns = f.Status, f.EffectiveFrom, f.EffectiveTo, f.Scope, f.RemovedColumns
		text, fm = body, true
	}
	title, body, _ := strings.Cut(text, "\n")
	d.Title, d.Body = strings.TrimSpace(strings.TrimPrefix(title, "#")), body
	return d, fm, nil
}

func validDate(s string) bool {
	_, err := time.Parse(dateLayout, s)
	return err == nil
}

const dateLayout = "2006-01-02"

// frontMatter là phần đầu của một tệp tài liệu có phiên bản.
type frontMatter struct {
	ID             string   `yaml:"id"`
	Version        string   `yaml:"version"` // chuỗi để "2.7" báo lỗi thay vì bị cắt thành 2
	Status         string   `yaml:"status"`
	EffectiveFrom  string   `yaml:"effective_from"`
	EffectiveTo    string   `yaml:"effective_to"`
	Scope          []string `yaml:"scope"`
	RemovedColumns []string `yaml:"removed_columns"`
}

func parseFrontMatter(head string) (frontMatter, error) {
	var fm frontMatter
	dec := yaml.NewDecoder(strings.NewReader(head))
	dec.KnownFields(true)
	if err := dec.Decode(&fm); err != nil {
		return fm, fmt.Errorf("phần đầu: %w", err)
	}
	if v, err := strconv.Atoi(fm.Version); err != nil || v < 1 {
		return fm, fmt.Errorf("phần đầu: version %q phải là số nguyên ≥ 1", fm.Version)
	}
	switch fm.Status {
	case StatusApproved, StatusDraft, StatusNeedsReview, StatusSuperseded:
	default:
		return fm, fmt.Errorf("phần đầu: trạng thái %q không hợp lệ", fm.Status)
	}
	for _, s := range []string{fm.EffectiveFrom, fm.EffectiveTo} {
		if _, err := time.Parse(dateLayout, s); s != "" && err != nil {
			return fm, fmt.Errorf("phần đầu: ngày %q phải có dạng %s", s, dateLayout)
		}
	}
	if fm.EffectiveFrom != "" && fm.EffectiveTo != "" && fm.EffectiveTo <= fm.EffectiveFrom {
		return fm, fmt.Errorf("phần đầu: hết hiệu lực (%s) không sau ngày bắt đầu (%s)", fm.EffectiveTo, fm.EffectiveFrom)
	}
	return fm, nil
}
