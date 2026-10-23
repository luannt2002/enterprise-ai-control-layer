package traces

import (
	"encoding/json"
	"reflect"
	"regexp"
	"strings"
	"unicode/utf8"

	"github.com/luannt2002/enterprise-ai-control-layer/core/contracts"
)

// Masked thay cho mọi giá trị bí mật bị che. Cố ý không có khoảng trắng: dấu hiệu "key=giá trị" dừng ở khoảng trắng, nên
// dấu che có khoảng trắng sẽ bị che lần hai thành "[…] che]" — Sanitize phải chạy lại không đổi (idempotent).
const Masked = "[REDACTED]"

// maxErrorRunes là trần độ dài thông điệp lỗi ghi vào vết: lỗi từ thư viện đôi khi nhúng cả tải trọng.
const maxErrorRunes = 500

// secretKeyWords là các từ khoá làm một tên (khoá, biến môi trường, thuộc tính) trở thành "tên của bí mật".
const secretKeyWords = `password|passwd|pwd|secret|token|api[_\-]?key|apikey|access[_\-]?key|private[_\-]?key|credential|mật khẩu|mat khau`

// secretMasks là các dấu hiệu bí mật đã biết, che theo thứ tự (dấu hiệu rộng như "key=giá trị" chạy sau dấu hiệu cụ thể).
// Che theo mẫu chỉ là lớp phòng thủ thứ hai: nó không thay cho việc không đưa bí mật vào các trường này từ đầu.
//
// Giá trị của "khoá = giá trị" có ba dạng: trong nháy kép, trong nháy đơn (kể cả có khoảng trắng — nháy không đóng thì che tới hết
// chuỗi: chưa biết chỗ kết thúc thì che thừa còn hơn để lọt đuôi), hoặc trần (tới khoảng trắng / dấu phân cách). Nháy bao quanh được
// giữ lại quanh dấu che để cấu trúc JSON không bị phá.
//
// Giới hạn đã biết (KHÔNG che): bí mật trần có khoảng trắng ("password=hunter two" chỉ che "hunter"); câu văn xuôi không có dấu : hoặc
// = ("mật khẩu của tôi là hunter2pw"); chuỗi ngẫu nhiên không có từ khoá và không theo dạng token đã biết. Bộ mẫu chỉ là danh sách —
// dạng nào ngoài danh sách thì chưa có bảo đảm.
var secretMasks = []struct {
	re   *regexp.Regexp
	repl string
}{
	// khoá riêng PEM, kể cả khối bị cắt cụt (không có dòng END)
	{regexp.MustCompile(`(?s)-----BEGIN [A-Z ]*PRIVATE KEY-----.*?(?:-----END [A-Z ]*PRIVATE KEY-----|$)`), Masked},
	// mật khẩu trong URL: scheme://người:mật-khẩu@máy, kể cả người dùng rỗng ("redis://:mật-khẩu@máy")
	{regexp.MustCompile(`(?i)\b([a-z][a-z0-9+.\-]*://[^\s/:@]*):[^\s/@]+@`), "${1}:" + Masked + "@"},
	// tiêu đề Authorization (mọi kiểu: Bearer, Basic, Token…) và "Bearer <mã>" đứng riêng
	{regexp.MustCompile(`(?i)(authorization["']?\s*[:=]\s*["']?)(?:(?:bearer|basic|token)\s+)?[^\s"',;]+`), "${1}" + Masked},
	{regexp.MustCompile(`(?i)\b(bearer)\s+[A-Za-z0-9._~+/=\-]{8,}`), "${1} " + Masked},
	// "tên-có-từ-khoá-bí-mật = giá trị" (password=…, "api_key": "…", EAIS_PG_PASSWORD=…, mật khẩu: …)
	{regexp.MustCompile(`(?i)\b([a-z0-9_.\-]*(?:` + secretKeyWords + `)[a-z0-9_.\-]*)(["']?\s*[:=]\s*)(?:(")[^"]*"?|(')[^']*'?|[^\s"',;&]+)`), "${1}${2}${3}${4}" + Masked + "${3}${4}"},
	// tiêu đề Cookie / Set-Cookie: cả giá trị là bí mật phiên (nhiều cặp, kèm thuộc tính), che hết dòng chứ không chỉ cặp đầu
	{regexp.MustCompile(`(?i)\b((?:set-)?cookies?["']?\s*[:=]\s*)(?:(")[^"]*"?|(')[^']*'?|[^\r\n]+)`), "${1}${2}${3}" + Masked + "${2}${3}"},
	// dạng mã thông báo đã biết, đứng ở đâu cũng che: sk-…, ghp_…, github_pat_…, xox?-…, AKIA…, AIza…, JWT
	{regexp.MustCompile(`\b(?:sk-[A-Za-z0-9_\-]{16,}|gh[pousr]_[A-Za-z0-9]{20,}|github_pat_[A-Za-z0-9_]{20,}|xox[abprs]-[A-Za-z0-9\-]{10,}|AKIA[0-9A-Z]{16}|AIza[0-9A-Za-z_\-]{30,}|eyJ[A-Za-z0-9_\-]{8,}\.[A-Za-z0-9_\-]{8,}\.[A-Za-z0-9_\-]{8,})`), Masked},
}

// Sanitize làm sạch một chuỗi trước khi ghi vào vết: sửa UTF-8 hỏng và bỏ byte NUL (PostgreSQL không nhận cả hai trong text /
// jsonb — để lọt thì lượt đó mất vết bằng một lỗi hạ tầng "thử lại được" nhưng không bao giờ thành công), rồi che bí mật.
func Sanitize(s string) string {
	if !utf8.ValidString(s) {
		s = strings.ToValidUTF8(s, "�")
	}
	s = strings.ReplaceAll(s, "\x00", "")
	for _, m := range secretMasks {
		s = m.re.ReplaceAllString(s, m.repl)
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
