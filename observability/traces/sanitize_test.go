package traces_test

import (
	"encoding/json"
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"
	"unicode/utf8"

	"github.com/luannt2002/enterprise-ai-control-layer/core/contracts"
	"github.com/luannt2002/enterprise-ai-control-layer/observability/traces"
)

// Bí mật GIẢ, ghép từ mảnh để tệp nguồn không chứa nguyên một chuỗi giống khoá thật (tránh máy quét bí mật báo oan).
var (
	fakeAPIKey = "sk-" + "ant-api03-" + strings.Repeat("A1b2", 8)
	fakeGHP    = "gh" + "p_" + strings.Repeat("Z9", 18)
	fakeAWS    = "AK" + "IA" + "ABCDEFGHIJKLMNOP"
	fakeJWT    = "ey" + "J" + "hbGciOiJIUzI1NiJ9" + "." + "eyJzdWIiOiJ1c2VyIn0" + "." + "c2lnbmF0dXJlMTIzNDU2"
	fakePEM    = "-----BEGIN " + "RSA PRIVATE KEY-----\nMIIBOgIBAAJBAKj34GkxFhD90vcNLYLInFEX6Ppy1tPf9Cnzj4p4WGeKLs1Pt8Qu\n-----END " + "RSA PRIVATE KEY-----"
)

// Mỗi ca: chuỗi chứa bí mật, và mảnh bí mật không được còn trong kết quả (mảnh ngữ cảnh phải còn để lỗi vẫn đọc được).
func TestSanitizeCheBiMat(t *testing.T) {
	for _, c := range []struct{ name, in, secret, keep string }{
		{"mật khẩu trong DSN", "lỗi kết nối postgres://eais_app:hunter2pw@127.0.0.1:5432/eais", "hunter2pw", "127.0.0.1:5432/eais"},
		{"Authorization Bearer", "gửi lỗi 401 Authorization: Bearer abcdefghij.klmnopqrst.uvwxyz0123 tới cổng", "abcdefghij", "tới cổng"},
		{"Authorization Basic", "Authorization: Basic dXNlcjpodW50ZXIycHc=", "dXNlcjpodW50ZXIycHc", "Authorization"},
		{"Bearer đứng riêng", "header bearer zzzzzzzzzzzzzzzz sai", "zzzzzzzzzzzzzzzz", "header"},
		{"password=", "password=hunter2pw host=db", "hunter2pw", "host=db"},
		{"khoá JSON api_key", `{"api_key": "QWERTY123456", "model": "m1"}`, "QWERTY123456", `"model": "m1"`},
		{"biến môi trường", "EAIS_PG_PASSWORD=hunter2pw EAIS_PG_HOST=db", "hunter2pw", "EAIS_PG_HOST=db"},
		{"mật khẩu tiếng Việt", "đổi mật khẩu: tuyetmat123 giúp tôi", "tuyetmat123", "giúp tôi"},
		{"khoá API đứng riêng", "tôi dán nhầm " + fakeAPIKey + " vào đây", fakeAPIKey, "vào đây"},
		{"token GitHub", "token " + fakeGHP + " hết hạn", fakeGHP, "hết hạn"},
		{"khoá AWS", "key " + fakeAWS + " lộ", fakeAWS, "lộ"},
		{"JWT", "phiên " + fakeJWT + " hỏng", fakeJWT, "hỏng"},
		{"khoá riêng PEM", "đây: " + fakePEM + " hết", "MIIBOgIBAAJBAKj34GkxFhD90vcNLYLInFEX6Ppy1tPf9Cnzj4p4WGeKLs1Pt8Qu", "đây:"},
		{"PEM cụt (không có dòng END)", "x -----BEGIN " + "PRIVATE KEY-----\nMIIEvAIBADANBgkqhkiG9w0BAQEFAASC", "MIIEvAIBADANBgkqhkiG9w0BAQEFAASC", "x"},
	} {
		got := traces.Sanitize(c.in)
		if strings.Contains(got, c.secret) {
			t.Errorf("%s: bí mật còn trong %q", c.name, got)
		}
		if !strings.Contains(got, c.keep) {
			t.Errorf("%s: mất cả ngữ cảnh %q, được %q", c.name, c.keep, got)
		}
		if !strings.Contains(got, traces.Masked) {
			t.Errorf("%s: không thấy dấu che trong %q", c.name, got)
		}
		if again := traces.Sanitize(got); again != got {
			t.Errorf("%s: Sanitize không idempotent: %q → %q", c.name, got, again)
		}
	}
}

// Ca âm: nội dung thường, mã phiên bản, băm và định danh KHÔNG bị che oan — vết đọc được mới debug được.
func TestSanitizeKhongCheOan(t *testing.T) {
	for _, in := range []string{
		"EX-03 hôm nay được cấp bao nhiêu lít dầu?",
		"Dầu kho xuất của Máy xúc EX-03: 3.850 L. Nguồn khác lệch số: ERP 3.900 L",
		"018a029c1f3e5b7d9a0b1c2d3e4f5a6b7c8d9e0f+dirty",
		"sha256:" + strings.Repeat("0123456789abcdef", 4),
		"task-0123456789abcdefghij",
		"risk-assessment-for-the-project-plan",
		"basic information about the site",
		"550e8400-e29b-41d4-a716-446655440000",
		"tokenizer chạy 12 ms",
		"entity_scope=PRJ-A",
	} {
		if got := traces.Sanitize(in); got != in {
			t.Errorf("che oan: %q → %q", in, got)
		}
	}
}

// Chuỗi hỏng không được lọt tới PostgreSQL: NUL và UTF-8 sai làm lệnh ghi thất bại bằng lỗi hạ tầng "thử lại được" mà không bao
// giờ thành công, tức là lượt đó mất vết.
func TestSanitizeSuaChuoiHong(t *testing.T) {
	got := traces.Sanitize("a\x00b\xffc")
	if got != "ab�c" || !utf8.ValidString(got) {
		t.Errorf("được %q, muốn %q", got, "ab�c")
	}
}

func TestSanitizeError(t *testing.T) {
	if f, m := traces.SanitizeError(nil); f != "" || m != "" {
		t.Errorf("lỗi nil: được (%q, %q)", f, m)
	}
	labeled := contracts.Label(contracts.Infra("postgres"), errors.New("kết nối postgres://u:hunter2pw@h/db hỏng"))
	f, m := traces.SanitizeError(labeled)
	if f != "infra:postgres" || strings.Contains(m, "hunter2pw") || !strings.Contains(m, "hỏng") {
		t.Errorf("lỗi gắn nhãn: được (%q, %q)", f, m)
	}
	if f, _ := traces.SanitizeError(errors.New("chưa ai gắn nhãn")); f != contracts.FaultCode {
		t.Errorf("lỗi chưa gắn nhãn phải tính là lỗi của mã (như HTTPStatus), được %q", f)
	}
	_, long := traces.SanitizeError(errors.New(strings.Repeat("x", 5000)))
	if n := utf8.RuneCountInString(long); n != 501 { // 500 + dấu cắt
		t.Errorf("thông điệp dài phải cắt còn 500 ký tự + dấu cắt, được %d", n)
	}
}

// fillStrings điền s vào MỌI trường chuỗi của v (kể cả phần tử slice và giá trị map), tạo phần tử khi slice / map rỗng.
func fillStrings(v reflect.Value, s string) {
	switch v.Kind() {
	case reflect.String:
		v.SetString(s)
	case reflect.Struct:
		for i := range v.NumField() {
			if v.Type().Field(i).IsExported() {
				fillStrings(v.Field(i), s)
			}
		}
	case reflect.Slice:
		v.Set(reflect.MakeSlice(v.Type(), 1, 1))
		fillStrings(v.Index(0), s)
	case reflect.Map:
		v.Set(reflect.MakeMap(v.Type()))
		k, val := reflect.New(v.Type().Key()).Elem(), reflect.New(v.Type().Elem()).Elem()
		k.SetString("k")
		val.SetString(s)
		v.SetMapIndex(k, val)
	}
}

// assertCovered đòi mọi kiểu trong Record là kiểu mà SanitizeRecord biết duyệt. Thêm trường *string, map[string]any, interface…
// vào Record mà không dạy bộ duyệt thì test này đỏ — thay vì để chuỗi mới lọt ra không che.
func assertCovered(t *testing.T, ty reflect.Type, path string) {
	t.Helper()
	switch ty.Kind() {
	case reflect.String, reflect.Bool, reflect.Int, reflect.Int32, reflect.Int64, reflect.Float64:
	case reflect.Struct:
		if ty == reflect.TypeOf(time.Time{}) {
			return
		}
		for i := range ty.NumField() {
			if f := ty.Field(i); f.IsExported() {
				assertCovered(t, f.Type, path+"."+f.Name)
			} else {
				t.Errorf("%s.%s: trường không xuất khẩu, SanitizeRecord không che được (đi qua JSON nên cũng không được lưu)", path, f.Name)
			}
		}
	case reflect.Slice:
		assertCovered(t, ty.Elem(), path+"[]")
	case reflect.Map:
		if ty.Key().Kind() != reflect.String || ty.Elem().Kind() != reflect.String {
			t.Errorf("%s: map %v chưa được SanitizeRecord phủ", path, ty)
		}
	default:
		t.Errorf("%s: kiểu %v chưa được SanitizeRecord phủ", path, ty)
	}
}

// Điền bí mật vào MỌI trường chuỗi của Record (kể cả sâu trong Decision, Steps, EvidenceRef, map thuộc tính) rồi đòi sạch hoàn
// toàn: phủ cả trường sẽ được thêm sau này.
func TestSanitizeRecordPhuMoiTruongChuoi(t *testing.T) {
	assertCovered(t, reflect.TypeOf(traces.Record{}), "Record")
	const secret = "hunter2pw"
	var rec traces.Record
	fillStrings(reflect.ValueOf(&rec).Elem(), "password="+secret)
	clean, err := traces.SanitizeRecord(rec)
	if err != nil {
		t.Fatal(err)
	}
	b, _ := json.Marshal(clean)
	if strings.Contains(string(b), secret) {
		t.Errorf("bí mật còn trong bản ghi đã làm sạch: %s", b)
	}
	if !strings.Contains(string(b), traces.Masked) {
		t.Errorf("không thấy dấu che: %s", b)
	}
	// bản gốc không bị đổi: SanitizeRecord trả BẢN SAO
	if rec.Decision.Question != "password="+secret || rec.Steps[0].Evidence[0].Source.Name != "password="+secret {
		t.Errorf("SanitizeRecord sửa bản gốc")
	}
}

// Thuộc tính ABAC là map: khoá mang từ khoá bí mật thì che nguyên giá trị dù giá trị trông vô hại; khoá thường giữ nguyên.
func TestSanitizeRecordCheGiaTriMapTheoKhoa(t *testing.T) {
	rec := traces.Record{Decision: traces.Decision{TraceID: "t", Attributes: map[string]string{
		"api_key": "AAAAAAAAAAAA", "entity_scope": "PRJ-A", "ghi_chu": "token: zzzzzzzzzzzz",
	}}}
	clean, err := traces.SanitizeRecord(rec)
	if err != nil {
		t.Fatal(err)
	}
	a := clean.Decision.Attributes
	if a["api_key"] != traces.Masked || a["entity_scope"] != "PRJ-A" || strings.Contains(a["ghi_chu"], "zzzzzzzzzzzz") {
		t.Errorf("thuộc tính sau khi làm sạch: %v", a)
	}
	if clean.Steps == nil || clean.Spans == nil {
		t.Errorf("Steps và Spans phải là mảng (không nil) để DB lưu [] chứ không null")
	}
}

// R1-04. Giá trị bí mật nằm trong nháy và CÓ khoảng trắng: bộ che phải lấy hết giá trị, không để lọt phần đuôi. Mỗi ca liệt kê MỌI mảnh
// của bí mật; nháy không đóng thì che tới hết chuỗi (chưa biết chỗ kết thúc thì che thừa còn hơn lọt). Cookie / Set-Cookie: cả giá trị
// của tiêu đề là bí mật phiên, không chỉ một cặp.
func TestSanitizeCheDuoiGiaTriCoKhoangTrang(t *testing.T) {
	for _, c := range []struct {
		name, in string
		secrets  []string // mọi mảnh của bí mật, không được còn
		keep     string   // ngữ cảnh phải còn
	}{
		{"nháy kép, nhiều từ", `password="hunter two three" host=db`, []string{"hunter", "two", "three"}, "host=db"},
		{"JSON, giá trị có khoảng trắng", `{"password":"hun ter2", "model": "m1"}`, []string{"hun", "ter2"}, `"model": "m1"`},
		{"nháy đơn", `client_secret: 'abc def' xong`, []string{"abc", "def"}, "xong"},
		{"khoá có hậu tố, nháy kép", `secret_token="aaa bbb" tiếp`, []string{"aaa", "bbb"}, "tiếp"},
		{"pwd, nháy kép", `pwd: "co khoang trang" hết`, []string{"co khoang", "trang"}, "hết"},
		{"nháy kép không đóng", `password="chưa đóng nháy`, []string{"chưa", "đóng", "nháy"}, "password"},
		{"nháy đơn không đóng", `api_key='chưa đóng nháy`, []string{"chưa", "đóng", "nháy"}, "api_key"},
		{"giá trị rỗng trong nháy", `password="" host=db`, nil, "host=db"},
		{"tiêu đề Cookie", `Cookie: session=abc123def456ghi`, []string{"abc123def456ghi"}, "Cookie"},
		{"tiêu đề Set-Cookie (cả thuộc tính)", `Set-Cookie: sid=zzzzzzzzzzzz; Path=/`, []string{"zzzzzzzzzzzz"}, "Set-Cookie"},
		{"Cookie trong JSON", `{"Cookie": "sid=qqqqqqqqqqqq; theme=dark", "x": 1}`, []string{"qqqqqqqqqqqq", "theme=dark"}, `"x": 1`},
	} {
		got := traces.Sanitize(c.in)
		for _, s := range c.secrets {
			if strings.Contains(got, s) {
				t.Errorf("%s: mảnh bí mật %q còn trong %q", c.name, s, got)
			}
		}
		if !strings.Contains(got, c.keep) {
			t.Errorf("%s: mất cả ngữ cảnh %q, được %q", c.name, c.keep, got)
		}
		if !strings.Contains(got, traces.Masked) {
			t.Errorf("%s: không thấy dấu che trong %q", c.name, got)
		}
		if again := traces.Sanitize(got); again != got {
			t.Errorf("%s: Sanitize không idempotent: %q → %q", c.name, got, again)
		}
	}
	// nháy giữ nguyên cấu trúc: JSON có nháy quanh giá trị vẫn có nháy quanh dấu che (đọc vết không bị lẫn)
	if got := traces.Sanitize(`{"api_key": "QWERTY 123456"}`); got != `{"api_key": "`+traces.Masked+`"}` {
		t.Errorf("cấu trúc JSON bị phá: %q", got)
	}
}

// Ca âm cho R1-04: nháy, "cookie" và khoảng trắng trong nội dung thường KHÔNG bị che oan.
func TestSanitizeKhongCheOanNhayVaCookie(t *testing.T) {
	for _, in := range []string{
		`Tên "Máy xúc" và 'EX-03' đều hợp lệ`,
		"banner cookie hiển thị trên trang chủ",
		"chính sách cookie của site",
		`"model": "m1", "note": "hai từ"`,
		`mật khẩu mạnh cần 12 ký tự`,
		"password policy: chính sách mật khẩu", // "policy" chen giữa từ khoá và dấu hai chấm: không phải cặp khoá = giá trị
	} {
		if got := traces.Sanitize(in); got != in {
			t.Errorf("che oan: %q → %q", in, got)
		}
	}
}

// Dạng lọt do soát độc lập 23/10 tìm ra: URL có người dùng RỖNG ("postgres://:mật-khẩu@máy", dạng hay gặp của Redis) — mẫu cũ đòi tên
// người dùng ít nhất một ký tự nên cả mật khẩu lọt.
func TestSanitizeCheURLNguoiDungRong(t *testing.T) {
	for _, c := range []struct{ name, in, secret, keep string }{
		{"DSN người dùng rỗng", "lỗi kết nối postgres://:hunter2pw@127.0.0.1:5432/eais", "hunter2pw", "127.0.0.1:5432/eais"},
		{"Redis người dùng rỗng", "dial redis://:hunter2pw@cache:6379/0 thất bại", "hunter2pw", "cache:6379/0 thất bại"},
	} {
		got := traces.Sanitize(c.in)
		if strings.Contains(got, c.secret) {
			t.Errorf("%s: bí mật còn trong %q", c.name, got)
		}
		if !strings.Contains(got, c.keep) {
			t.Errorf("%s: mất cả ngữ cảnh %q, được %q", c.name, c.keep, got)
		}
		if again := traces.Sanitize(got); again != got {
			t.Errorf("%s: Sanitize không idempotent: %q → %q", c.name, got, again)
		}
	}
	// ca âm: URL không có thông tin đăng nhập thì giữ nguyên
	for _, s := range []string{"gọi http://example.com:8080/path thất bại", "postgres://db:5432/eais không trả lời"} {
		if got := traces.Sanitize(s); got != s {
			t.Errorf("che oan: %q → %q", s, got)
		}
	}
}

// Dạng lọt do soát độc lập 23/10 tìm ra: giá trị trong nháy có NHÁY THOÁT ("password":"ab\"pw") — mẫu cũ dừng ở nháy thoát đầu tiên nên
// đuôi sau nó lọt. Áp cho "khoá = giá trị" (nháy kép, nháy đơn) và tiêu đề Cookie.
func TestSanitizeCheGiaTriCoNhayThoat(t *testing.T) {
	for _, c := range []struct{ name, in, secret, keep string }{
		{"JSON nháy thoát", `{"password":"ab\"hunter2pw","model":"m1"}`, "hunter2pw", `"model":"m1"`},
		{"JSON hai nháy thoát", `{"token":"a\"b\"SECRETVALUE99","n":1}`, "SECRETVALUE99", `"n":1`},
		{"JSON gạch chéo thoát rồi nháy", `{"api_key":"x\\\"QWERTY123456","n":2}`, "QWERTY123456", `"n":2`},
		{"nháy đơn thoát", `password='ab\'hunter2pw' host=db`, "hunter2pw", "host=db"},
		{"Cookie trong nháy có nháy thoát", `Cookie: "sid=abc\"hunter2pw"; next`, "hunter2pw", "Cookie"},
	} {
		got := traces.Sanitize(c.in)
		if strings.Contains(got, c.secret) {
			t.Errorf("%s: bí mật còn trong %q", c.name, got)
		}
		if !strings.Contains(got, c.keep) {
			t.Errorf("%s: mất cả ngữ cảnh %q, được %q", c.name, c.keep, got)
		}
		if again := traces.Sanitize(got); again != got {
			t.Errorf("%s: Sanitize không idempotent: %q → %q", c.name, got, again)
		}
	}
	// ca âm: nháy thoát ở khoá KHÔNG phải bí mật thì giữ nguyên
	if s := `{"note":"anh ấy nói \"chào\"","model":"m1"}`; traces.Sanitize(s) != s {
		t.Errorf("che oan: %q → %q", s, traces.Sanitize(s))
	}
}

// Hồi quy do bản đọc nháy thoát gây ra (soát độc lập 5a-1): chữ KHÔNG phải JSON có giá trị trong nháy kết thúc bằng \ (đường dẫn
// Windows, chữ chép tay) — đọc theo nháy thoát thì "\"" không đóng giá trị, nháy mở của bí mật KẾ TIẾP bị nuốt làm nháy đóng và
// bí mật đó lọt. Đọc thô (nháy kế là nháy đóng) che được ca này nhưng lại lọt ca JSON ở test trên, nên phải che HỢP hai cách đọc.
func TestSanitizeGiaTriKetThucBangGachCheo(t *testing.T) {
	for _, c := range []struct{ name, in string }{
		{"nháy kép rồi khoá kế", `ghi: password="ab\" token="ZQX77pw" xong`},
		{"dạng phẩy", `ghi: password="ab\", api_key="ZQX77pw" xong`},
		{"nháy đơn", `ghi: password='ab\' token='ZQX77pw' xong`},
		{"đường dẫn Windows", `ghi: secret="C:\share\" owner=bob token="ZQX77pw" xong`},
	} {
		got := traces.Sanitize(c.in)
		if strings.Contains(got, "ZQX77pw") {
			t.Errorf("%s: bí mật kế tiếp còn trong %q", c.name, got)
		}
		if !strings.HasPrefix(got, "ghi: ") || !strings.HasSuffix(got, " xong") {
			t.Errorf("%s: mất ngữ cảnh ngoài vùng bí mật: %q", c.name, got)
		}
		if again := traces.Sanitize(got); again != got {
			t.Errorf("%s: Sanitize không idempotent: %q → %q", c.name, got, again)
		}
	}
}

// Đầu ra ĐÚNG TỪNG KÝ TỰ cho các dạng mà test theo chuỗi con không khoá được (soát độc lập 5a-3: đột biến bỏ "@" khỏi dấu che, đòi mật
// khẩu URL từ 2 ký tự, bỏ xử lý nháy thoát ở Cookie nháy đơn, "\." thay "\[\s\S]" đều sống sót) và các dạng phân biệt hai cách đọc nháy.
func TestSanitizeDauRaChinhXac(t *testing.T) {
	for _, c := range []struct{ name, in, want string }{
		{"URL người dùng rỗng", "lỗi kết nối postgres://:hunter2pw@127.0.0.1:5432/eais", "lỗi kết nối postgres://:" + traces.Masked + "@127.0.0.1:5432/eais"},
		{"URL mật khẩu 1 ký tự", "dial redis://:p@cache:6379/0 thất bại", "dial redis://:" + traces.Masked + "@cache:6379/0 thất bại"},
		{"Cookie nháy đơn có nháy thoát", `Cookie: 'sid=abc\'hunter2pw'; next`, `Cookie: '` + traces.Masked + `'; next`},
		{"nháy thoát sau xuống dòng thật", "{\"password\":\"ab\\\ncd\\\"hunter2pw\",\"n\":1}", `{"password":"` + traces.Masked + `","n":1}`},
		{"nháy thoát rồi khoảng trắng", `{"password":"ab\" hunter2pw","n":1}`, `{"password":"` + traces.Masked + `","n":1}`},
		{"nháy không đóng sau nháy thoát", `password="ab\" hunter two`, `password="` + traces.Masked + `"`},
	} {
		got := traces.Sanitize(c.in)
		if got != c.want {
			t.Errorf("%s: %q → %q, muốn %q", c.name, c.in, got, c.want)
		}
		if again := traces.Sanitize(got); again != got {
			t.Errorf("%s: Sanitize không idempotent: %q → %q", c.name, got, again)
		}
	}
}

// Soát lại 5abc (5a-R1, S1): giá trị trong nháy kết thúc bằng \ rồi một tiêu đề Cookie / Set-Cookie có nháy ở sau (cùng dòng hay dòng sau). Che
// lần lượt từng mẫu thì đọc theo nháy thoát của mẫu "khoá = giá trị" nuốt tiền tố "Cookie:" của chữ, mẫu Cookie chạy sau không còn khớp và giá trị
// Cookie lọt nguyên văn (main che đúng; bản 5a trước lọt). Mọi mẫu "tên + giá trị" phải tính vùng trên CÙNG chữ gốc rồi gộp một lần.
func TestSanitizeCookieSauGiaTriKetThucBangGachCheo(t *testing.T) {
	for _, in := range []string{
		"password=\"C:\\secret\\\"\nCookie: sid=\"ZQX77pw\"",
		`ghi: password="ab\" Cookie: "sid=ZQX77pw" xong`,
		`ghi: password='ab\' Set-Cookie: 'sid=ZQX77pw' xong`,
		"password=\"C:\\x\\\"\nCookie: theme=dark; sid=\"ZQX77pw\"; lang=vi",
	} {
		if got := traces.Sanitize(in); strings.Contains(got, "ZQX") {
			t.Errorf("bí mật Cookie lọt: %q → %q", in, got)
		}
	}
}
