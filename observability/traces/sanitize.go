package traces

import (
	"encoding/json"
	"reflect"
	"regexp"
	"sort"
	"strings"
	"unicode/utf8"

	"github.com/luannt2002/enterprise-ai-control-layer/core/contracts"
)

// Masked thay cho mọi giá trị bí mật bị che. Cố ý không có khoảng trắng: dấu hiệu "key=giá trị" dừng ở khoảng trắng, nên
// dấu che có khoảng trắng sẽ bị che lần hai thành "[…] che]" — Sanitize phải chạy lại không đổi (idempotent) trên đầu vào thường.
// Không phổ quát: fuzz so lệch 23/10 (bộ sinh của soát 5a vòng 3, 400.000 chuỗi mỗi lượt) thấy Sanitize(Sanitize(x)) ≠ Sanitize(x) ở 0
// chuỗi với bộ "target" (hai hạt) và 36 / 23 chuỗi với bộ "broad" (hai hạt; main 37 / 26 trên cùng chuỗi); test chỉ khẳng định
// idempotent trên các ca đã chọn.
const Masked = "[REDACTED]"

// maxErrorRunes là trần độ dài thông điệp lỗi ghi vào vết: lỗi từ thư viện đôi khi nhúng cả tải trọng.
const maxErrorRunes = 500

// secretKeyWords là các từ khoá làm một tên (khoá, biến môi trường, thuộc tính) trở thành "tên của bí mật".
const secretKeyWords = `password|passwd|pwd|secret|token|api[_\-]?key|apikey|access[_\-]?key|private[_\-]?key|credential|mật khẩu|mat khau`

// secretMasks là các dấu hiệu bí mật đã biết, che theo thứ tự (dấu hiệu rộng như "key=giá trị" chạy sau dấu hiệu cụ thể).
// Che theo mẫu chỉ là lớp phòng thủ thứ hai: nó không thay cho việc không đưa bí mật vào các trường này từ đầu.
//
// Giá trị của "khoá = giá trị" có ba dạng: trong nháy kép, trong nháy đơn (kể cả có khoảng trắng và nháy thoát như \" — nháy không
// đóng thì che tới hết chuỗi: chưa biết chỗ kết thúc thì che thừa còn hơn để lọt đuôi), hoặc trần (tới khoảng trắng / dấu phân cách).
// Nháy bao quanh được giữ lại quanh dấu che để cấu trúc JSON không bị phá. Giá trị trong nháy đọc theo HAI cách rồi che hợp hai vùng
// (xem quotedEscaped, valueMask).
//
// Giới hạn đã biết (KHÔNG che): bí mật trần có khoảng trắng ("password=hunter two" chỉ che "hunter"); câu văn xuôi không có dấu : hoặc
// = ("mật khẩu của tôi là hunter2pw"); chuỗi ngẫu nhiên không có từ khoá và không theo dạng token đã biết. Thêm (soát độc lập 23/10, 5a-4,
// đều có từ trước 5a): mật khẩu URL chứa "@" hay "/" thô, không mã hoá % ("postgres://u:ab@xy@máy" chỉ che "ab"; có "/" thì không che);
// JSON lồng dạng chuỗi ({"msg":"{\"password\":\"…\"}"}: khoá đứng sau \" không được nhận ra); giá trị là mảng ("password":["a","b"] chỉ
// che "["); giá trị trần chứa ; , & " (phần sau dấu đó lọt); Cookie có giá trị trong nháy rồi thêm cặp sau ";" (Cookie: "a=1"; sid=…:
// phần sau nháy đóng không được che); dòng Cookie có cặp KHÔNG phải tên bí mật mà giá trị trong nháy qua xuống dòng
// (Cookie: theme="x⏎y"; sid=…: giá trị trần dừng ở xuống dòng đầu tiên, phần sau nháy đóng ở dòng kế tiếp lọt — main cũng lọt, đo 23/10). Che thừa: "http://máy:8080,me@x.com" bị coi là mật khẩu URL. Mẫu URL người dùng rỗng nuốt tên khoá
// đứng ở chỗ mật khẩu ("redis://:secret=x@máy…"), nên phần sau "@" (máy, đường dẫn) không còn bị mẫu "khoá = giá trị" che thừa như trước
// (fuzz 23/10: 105 / 300.000 chuỗi ghép). Giá trị MỞ bằng nháy thoát (password=\"…\": logfmt / slog lồng, chữ đã qua %q) chỉ che được
// dấu \ đứng đầu, bí mật sau \" lọt (soát lại 5abc, 5a-R3; có từ trước 5a) — đường chạm tới: tên trường lạ trong thân POST
// /v1/commands/<công cụ> ghi vào bước C0 dạng "invalid_fields=…", lấy từ lỗi `json: unknown field %q` (bí mật do chính bên gọi gõ).
// Che THỪA (mất chữ không phải bí mật; đo 23/10, soát 5a vòng 3, 5a3-4): (i) có từ vòng 3, main giữ được — "Cookie:" nằm trong một giá trị
// trong nháy hay làm giá trị trần của khoá bí mật thì vùng Cookie trần (tính trên chữ gốc) chạy qua nháy đóng và nuốt nốt phần còn lại của
// DÒNG: `token="abc Cookie: sid=… def" tail X` → `token="[REDACTED]"`, `password:Cookie: X` → `password:[REDACTED] [REDACTED]`; (ii) có từ
// vòng 1–2, main giữ được — cặp không bí mật nằm giữa hai cặp bí mật mà hai cách đọc nháy bất đồng bị nuốt cùng tên cặp kế tiếp
// (`password="a\" user=X note=Y secret="b\" end` → `password="[REDACTED]" end`), và giá trị kết thúc bằng \ rồi nháy mà sau đó không còn nháy
// nào thì cách đọc theo nháy thoát coi là chưa đóng, che tới HẾT CHUỖI (mọi dòng sau mất). Cả hai đổi nghĩa vết nếu rơi vào câu hỏi hay
// thông điệp lỗi. Bộ mẫu chỉ là danh sách — dạng nào ngoài danh sách thì chưa có bảo đảm.
var secretMasks = []struct {
	re    *regexp.Regexp
	repl  string
	value *valueMask // khác nil: mẫu "tên + giá trị" che hợp hai cách đọc nháy (re, repl không dùng)
}{
	// khoá riêng PEM, kể cả khối bị cắt cụt (không có dòng END)
	{re: regexp.MustCompile(`(?s)-----BEGIN [A-Z ]*PRIVATE KEY-----.*?(?:-----END [A-Z ]*PRIVATE KEY-----|$)`), repl: Masked},
	// mật khẩu trong URL: scheme://người:mật-khẩu@máy, kể cả người dùng rỗng ("redis://:mật-khẩu@máy")
	{re: regexp.MustCompile(`(?i)\b([a-z][a-z0-9+.\-]*://[^\s/:@]*):[^\s/@]+@`), repl: "${1}:" + Masked + "@"},
	// tiêu đề Authorization (mọi kiểu: Bearer, Basic, Token…) và "Bearer <mã>" đứng riêng
	{re: regexp.MustCompile(`(?i)(authorization["']?\s*[:=]\s*["']?)(?:(?:bearer|basic|token)\s+)?[^\s"',;]+`), repl: "${1}" + Masked},
	{re: regexp.MustCompile(`(?i)\b(bearer)\s+[A-Za-z0-9._~+/=\-]{8,}`), repl: "${1} " + Masked},
	// "tên-có-từ-khoá-bí-mật = giá trị" (password=…, "api_key": "…", EAIS_PG_PASSWORD=…, mật khẩu: …)
	{value: newValueMask(`(?i)\b[a-z0-9_.\-]*(?:`+secretKeyWords+`)[a-z0-9_.\-]*["']?\s*[:=]\s*`, `[^\s"',;&]+`, false)},
	// tiêu đề Cookie / Set-Cookie: cả giá trị là bí mật phiên (nhiều cặp, kèm thuộc tính), che hết dòng chứ không chỉ cặp đầu
	{value: newValueMask(`(?i)\b(?:set-)?cookies?["']?\s*[:=]\s*`, `[^\r\n]+`, true)},
	// dạng mã thông báo đã biết, đứng ở đâu cũng che: sk-…, ghp_…, github_pat_…, xox?-…, AKIA…, AIza…, JWT
	{re: regexp.MustCompile(`\b(?:sk-[A-Za-z0-9_\-]{16,}|gh[pousr]_[A-Za-z0-9]{20,}|github_pat_[A-Za-z0-9_]{20,}|xox[abprs]-[A-Za-z0-9\-]{10,}|AKIA[0-9A-Z]{16}|AIza[0-9A-Za-z_\-]{30,}|eyJ[A-Za-z0-9_\-]{8,}\.[A-Za-z0-9_\-]{8,}\.[A-Za-z0-9_\-]{8,})`), repl: Masked},
}

// Hai cách đọc giá trị trong nháy. Theo nháy thoát (quotedEscaped: \" nằm TRONG giá trị — đúng với JSON, logfmt) và thô (quotedRaw: nháy
// kế tiếp là nháy đóng — đúng với chữ chép tay có giá trị kết thúc bằng \ như "C:\share\"). Hai cách chỉ khác nhau khi có \ ngay trước
// nháy, và chọn cách nào thì cách kia lọt bí mật: đọc thô lọt đuôi "pw" của {"password":"ab\"pw"}, đọc theo nháy thoát nuốt nháy mở
// của bí mật KẾ TIẾP trong password="ab\" token="bí-mật" (soát độc lập 23/10, 5a-1). Nên valueMask che HỢP hai vùng.
const (
	quotedEscaped = `"(?:[^"\\]|\\[\s\S])*"?|'(?:[^'\\]|\\[\s\S])*'?`
	quotedRaw     = `"[^"]*"?|'[^']*'?`
)

// valueMask là mẫu "tiền tố + giá trị" đọc theo hai cách: giá trị (nhóm v) trong nháy theo quotedEscaped / quotedRaw, hoặc trần. line:
// giá trị trần là phần còn lại của DÒNG (Cookie / Set-Cookie) — maskValues kéo vùng gộp chứa nó tới hết dòng.
type valueMask struct {
	escaped, raw *regexp.Regexp
	line         bool
}

func newValueMask(prefix, bare string, line bool) *valueMask {
	return &valueMask{
		escaped: regexp.MustCompile(prefix + `(?P<v>` + quotedEscaped + `|` + bare + `)`),
		raw:     regexp.MustCompile(prefix + `(?P<v>` + quotedRaw + `|` + bare + `)`),
		line:    line,
	}
}

// span là một vùng [start, end) của chữ cần che; line: là giá trị trần của một mẫu dòng (Cookie / Set-Cookie).
type span struct {
	start, end int
	line       bool
}

// sequentialSpans trả các vùng mà CHE TUẦN TỰ — cách của main trước 5a — che bằng các mẫu dòng (Cookie / Set-Cookie): mẫu "tên + giá trị"
// đọc thô thay giá trị trước, mẫu dòng chạy trên chữ ĐÃ thay; vùng tính về toạ độ của s (mỗi byte của chữ đã thay nhớ khoảng chữ gốc của nó).
// maskValues hợp chúng vào vùng của mình, nên bước che giá trị không bao giờ che ít hơn che tuần tự (soát 5a vòng 4, 5a4-1: gộp một lần không
// mô phỏng được mọi hiệu ứng của che tuần tự — nháy của giá trị Cookie bị giá trị khoá bí mật nuốt, "Cookie:" nằm trong giá trị khoá bí mật).
func sequentialSpans(s string, ms []*valueMask) []span {
	cur := s
	lo, hi := make([]int, len(s)), make([]int, len(s))
	for i := range s {
		lo[i], hi[i] = i, i+1
	}
	var out []span
	for _, line := range []bool{false, true} {
		for _, m := range ms {
			if m.line != line {
				continue
			}
			v := 2 * m.raw.SubexpIndex("v")
			var b strings.Builder
			var nlo, nhi []int
			last := 0
			for _, x := range m.raw.FindAllStringSubmatchIndex(cur, -1) {
				a, e := x[v], x[v+1]
				if e <= a {
					continue
				}
				olo, ohi := lo[a], hi[a]
				for p := a; p < e; p++ {
					olo, ohi = min(olo, lo[p]), max(ohi, hi[p])
				}
				if line {
					out = append(out, span{olo, ohi, false})
				}
				tok := Masked
				if q := cur[a]; q == '"' || q == '\'' {
					tok = string(q) + Masked + string(q)
				}
				b.WriteString(cur[last:a])
				b.WriteString(tok)
				nlo, nhi = append(nlo, lo[last:a]...), append(nhi, hi[last:a]...)
				for range len(tok) {
					nlo, nhi = append(nlo, olo), append(nhi, ohi)
				}
				last = e
			}
			b.WriteString(cur[last:])
			cur, lo, hi = b.String(), append(nlo, lo[last:]...), append(nhi, hi[last:]...)
		}
	}
	return out
}

// maskValues che hợp các vùng giá trị (nhóm v) mà mọi mẫu trong ms tìm thấy trên s (cả hai cách đọc nháy): vùng chồng nhau gộp thành một,
// mỗi vùng thay bằng Masked, giữ nháy mở của vùng (nếu có) ở cả hai đầu để cấu trúc JSON không bị phá.
//
// Vùng gộp có chứa giá trị TRẦN của một mẫu dòng (Cookie / Set-Cookie) thì kéo tới hết dòng chứa đầu cuối của vùng, rồi gộp tiếp (soát 5a
// vòng 3, 5a3-1): giá trị trần của Cookie dừng ở xuống dòng ĐẦU TIÊN, kể cả xuống dòng nằm trong nháy của một cặp "tên = giá trị" bên
// trong dòng Cookie, nên phần sau nháy đóng ở dòng kế tiếp vẫn là của tiêu đề Cookie. Che tuần tự (main) thấy phần đó cùng dòng sau khi giá
// trị nhiều dòng thành một dấu che; không kéo thì phần đó lọt và Sanitize mất luỹ đẳng.
func maskValues(s string, ms ...*valueMask) string {
	spans := sequentialSpans(s, ms)
	for _, m := range ms {
		for _, re := range []*regexp.Regexp{m.escaped, m.raw} {
			v := 2 * re.SubexpIndex("v")
			for _, x := range re.FindAllStringSubmatchIndex(s, -1) {
				if x[v+1] > x[v] {
					q := s[x[v]]
					spans = append(spans, span{x[v], x[v+1], m.line && q != '"' && q != '\''})
				}
			}
		}
	}
	if len(spans) == 0 {
		return s
	}
	sort.Slice(spans, func(i, j int) bool { return spans[i].start < spans[j].start })
	var b strings.Builder
	last := 0
	for i := 0; i < len(spans); {
		cur := spans[i]
		for i++; ; i++ {
			if cur.line {
				if j := strings.IndexAny(s[cur.end:], "\r\n"); j >= 0 {
					cur.end += j
				} else {
					cur.end = len(s)
				}
			}
			if i >= len(spans) || spans[i].start >= cur.end {
				break
			}
			cur.end = max(cur.end, spans[i].end)
			cur.line = cur.line || spans[i].line
		}
		b.WriteString(s[last:cur.start])
		if q := s[cur.start]; q == '"' || q == '\'' {
			b.WriteByte(q)
			b.WriteString(Masked)
			b.WriteByte(q)
		} else {
			b.WriteString(Masked)
		}
		last = cur.end
	}
	return b.String() + s[last:]
}

// Sanitize làm sạch một chuỗi trước khi ghi vào vết: sửa UTF-8 hỏng và bỏ byte NUL (PostgreSQL không nhận cả hai trong text /
// jsonb — để lọt thì lượt đó mất vết bằng một lỗi hạ tầng "thử lại được" nhưng không bao giờ thành công), rồi che bí mật.
func Sanitize(s string) string {
	if !utf8.ValidString(s) {
		s = strings.ToValidUTF8(s, "�")
	}
	s = strings.ReplaceAll(s, "\x00", "")
	for i := 0; i < len(secretMasks); i++ {
		if secretMasks[i].value == nil {
			s = secretMasks[i].re.ReplaceAllString(s, secretMasks[i].repl)
			continue
		}
		// Mọi mẫu "tên + giá trị" đứng liền nhau tính vùng trên CÙNG một chữ rồi gộp MỘT lần (soát lại 5abc, 5a-R1): che lần lượt thì mẫu
		// trước (đọc theo nháy thoát) có thể nuốt tiền tố "Cookie:" của chữ, mẫu Cookie chạy sau không còn khớp và giá trị Cookie lọt.
		// Hệ quả (che thừa, chấp nhận — hướng an toàn; danh sách đủ ở "Giới hạn đã biết" trên secretMasks): vùng gộp nuốt cả tên của cặp kế
		// tiếp — `password="C:\x\"` xuống dòng `Cookie: sid="…"` thành `password="[REDACTED]"`, `password="a\" secret="…\" token="…"` thành
		// `password="[REDACTED]" token="[REDACTED]"`.
		var ms []*valueMask
		for ; i < len(secretMasks) && secretMasks[i].value != nil; i++ {
			ms = append(ms, secretMasks[i].value)
		}
		i--
		s = maskValues(s, ms...)
	}
	return s
}

// SanitizeError tách một lỗi thành nhãn fault và thông điệp đã làm sạch (che bí mật, cắt ở maxErrorRunes). Lỗi chưa gắn nhãn
// tính là lỗi của mã — cùng quy ước contracts.HTTPStatus. err nil trả hai chuỗi rỗng.
func SanitizeError(err error) (fault, msg string) {
	if err == nil {
		return "", ""
	}
	fault = contracts.FaultOf(err)
	if fault == "" {
		fault = contracts.FaultCode
	}
	msg = Sanitize(err.Error())
	if r := []rune(msg); len(r) > maxErrorRunes {
		msg = string(r[:maxErrorRunes]) + "…"
	}
	return fault, msg
}

// SanitizeRecord trả BẢN SAO đã làm sạch của r: mọi trường chuỗi (kể cả trong Decision, Steps, map thuộc tính) qua Sanitize; giá
// trị của map có KHOÁ mang từ khoá bí mật bị che nguyên giá trị. Bản sao đi qua JSON — đúng dạng sẽ nằm trong DB — nên đọc lại
// từ DB ra cùng JSON; Steps và Spans luôn là mảng (không null). Câu hỏi đã che KHÁC câu gốc: phát lại (Ask.Replay) trên câu đã che có
// thể cho Decision khác câu gốc — vết là để đọc và so, không phải bản sao nguyên văn của đầu vào.
//
// Duyệt bằng phản chiếu để thêm trường chuỗi mới vào Record là tự được che; test TestSanitizeRecordPhuMoiTruongChuoi giữ điều đó
// (điền bí mật vào MỌI trường chuỗi rồi đòi sạch).
func SanitizeRecord(r Record) (Record, error) {
	b, err := json.Marshal(r)
	if err != nil {
		return Record{}, err
	}
	var out Record
	if err := json.Unmarshal(b, &out); err != nil {
		return Record{}, err
	}
	sanitizeWalk(reflect.ValueOf(&out).Elem())
	if out.Steps == nil {
		out.Steps = []Step{}
	}
	if out.Spans == nil {
		out.Spans = []Span{}
	}
	return out, nil
}

func sanitizeWalk(v reflect.Value) {
	switch v.Kind() {
	case reflect.String:
		if v.CanSet() {
			v.SetString(Sanitize(v.String()))
		}
	case reflect.Struct:
		for i := range v.NumField() {
			if v.Type().Field(i).IsExported() {
				sanitizeWalk(v.Field(i))
			}
		}
	case reflect.Slice, reflect.Array:
		for i := range v.Len() {
			sanitizeWalk(v.Index(i))
		}
	case reflect.Map:
		if v.Type().Key().Kind() != reflect.String || v.Type().Elem().Kind() != reflect.String {
			return
		}
		for _, k := range v.MapKeys() {
			key, val := k.String(), v.MapIndex(k).String()
			clean := Sanitize(val)
			if Sanitize(key+"="+val) != key+"="+val && clean == val { // khoá mang từ khoá bí mật mà giá trị trơ ⇒ che nguyên giá trị
				clean = Masked
			}
			nk, nv := reflect.New(k.Type()).Elem(), reflect.New(v.Type().Elem()).Elem()
			nk.SetString(Sanitize(key))
			nv.SetString(clean)
			v.SetMapIndex(k, reflect.Value{}) // xoá khoá cũ (khoá có thể đổi sau khi làm sạch)
			v.SetMapIndex(nk, nv)
		}
	}
}
