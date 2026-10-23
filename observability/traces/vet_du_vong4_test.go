package traces_test

import (
	"encoding/json"
	"fmt"
	"regexp"
	"slices"
	"strings"
	"testing"

	"github.com/luannt2002/enterprise-ai-control-layer/core/contracts"
	"github.com/luannt2002/enterprise-ai-control-layer/core/pipeline"
	"github.com/luannt2002/enterprise-ai-control-layer/data/normalization"
	"github.com/luannt2002/enterprise-ai-control-layer/observability/traces"
)

// R2-03 vòng 4 (soát vòng 3: N1', S1, ngoại lệ an ninh của §9.6).
//
// Các mẫu che bí mật của traces.Sanitize cho nhóm token (sk-, ghp_, github_pat_, xox, AKIA, AIza, JWT) và cho Bearer / Cookie neo bằng `\b`,
// mà `\b` của Go chỉ coi chữ, số, gạch dưới ASCII là "chữ". Câu gốc gồm chữ x toàn chiều rộng (U+FF58) rồi "AKIA…" có ranh giới vì chữ đó không phải chữ ASCII, nên Sanitize nhận
// ra; câu đã gấp ký tự in.Raw là "xAKIA…" nên mất ranh giới và Sanitize không nhận ra. Cổng giữ chữ của Handle từng chỉ nhìn in.Raw, nên dòng giải thích
// "từ khoá lấy trong mệnh đề hỏi “…”" mang nguyên bí mật vào vết bền (người soát đo 105 / 343 lượt). Ma trận dưới đây dựng câu từ: họ bí mật neo `\b` ×
// MỌI ký tự mà bộ chuẩn hoá đổi thành chữ / số / gạch dưới ASCII (dò từ chính bộ chuẩn hoá, không liệt kê tay) × mẫu câu nhiều mệnh đề, rồi chạy lượt hỏi
// THẬT và kiểm: giải thích TRƯỚC khi làm sạch (thứ Handle tự đưa ra), và vết bền sau NewRecord + SanitizeRecord (lý do, phương án bị loại, quyết định, bảng chữ).

// anchoredFamily là một họ bí mật mà mẫu che neo bằng `\b`. secret là nguyên văn đặt vào câu hỏi; forms là các mảnh của bí mật mà vết không được in lại dưới
// bất kỳ dạng nào (so sau squash: bỏ dấu câu, hạ chữ thường).
type anchoredFamily struct {
	name   string
	secret string
	forms  []string
}

// Bí mật GIẢ, ghép từ mảnh để tệp nguồn không chứa nguyên một chuỗi giống khoá thật (xem sanitize_test.go). fakeAWS, fakeGHP, fakeAPIKey, fakeJWT dùng chung ở đó.
var (
	fakeGHPat   = "github" + "_pat_" + strings.Repeat("A1b2", 6)
	fakeSlack   = "xo" + "xb-" + "abcdefghij-klmnopqrst"
	fakeGoogle  = "AI" + "za" + strings.Repeat("Q7r", 11)
	fakeBearer  = "Bea" + "rer " + "tokenvalue12345"
	fakeCookie  = "Coo" + "kie: sid=" + "cookieVal99"
	anchoredAll = func() []anchoredFamily {
		fam := func(name, secret string, extra ...string) anchoredFamily {
			// bí mật đủ dài; thêm 12 ký tự squash đầu để một dòng chỉ mang NỬA bí mật (câu bị cắt giữa chừng, ký tự cuối bị NFC ghép dấu) cũng bị bắt
			forms := slices.Concat([]string{secret, squash(secret)[:12]}, extra)
			return anchoredFamily{name, secret, forms}
		}
		return []anchoredFamily{
			fam("AKIA", fakeAWS),
			fam("ghp_", fakeGHP),
			fam("github_pat_", fakeGHPat),
			fam("sk-", fakeAPIKey),
			fam("xox", fakeSlack),
			fam("AIza", fakeGoogle),
			fam("JWT", fakeJWT, strings.Split(fakeJWT, ".")...),
			fam("Bearer", fakeBearer, "tokenvalue12345"),
			fam("Cookie", fakeCookie, "cookieVal99"),
		}
	}()
)

type sentenceTemplate struct{ name, tmpl string }

// anchoredSentences là các mẫu câu: %s là chỗ đặt "ký tự gấp được + bí mật". Phủ bí mật ở cuối mệnh đề hỏi của câu xác nhận nhiều mệnh đề, giữa mệnh đề hỏi, trong
// mệnh đề ngữ cảnh đứng trước, đầu câu, cuối câu, sau dấu hỏi và câu chỉ có bí mật.
var anchoredSentences = []sentenceTemplate{
	{"cuối mệnh đề hỏi", "đội báo 4k, vậy EX-03 được cấp 3850 L đúng không %s"},
	{"đầu câu", "%s, EX-03 hôm nay được cấp 3850 L đúng không?"},
	{"cuối câu hỏi số liệu", "EX-03 hôm nay được cấp bao nhiêu dầu %s"},
	{"chỉ có bí mật", "%s"},
	{"giữa mệnh đề hỏi", "đội báo 4k, vậy EX-03 được cấp %s 3850 L đúng không?"},
	{"mệnh đề hỏi nhiều mệnh đề", "đội báo 4k, báo cáo ghi 4 m3, vậy EX-03 được cấp %s đúng không?"},
	{"mệnh đề ngữ cảnh đứng trước", "đội báo %s, báo cáo ghi 4 m3, vậy EX-03 được cấp 4k dầu đúng không?"},
	{"sau dấu hỏi", "đội báo 4k, vậy EX-03 được cấp 3850 L đúng không? %s"},
}

func isASCIIWordByte(c byte) bool {
	return c >= '0' && c <= '9' || c >= 'A' && c <= 'Z' || c >= 'a' && c <= 'z' || c == '_'
}

// foldedWordRunes dò từ CHÍNH bộ chuẩn hoá mọi ký tự ngoài ASCII (cả mặt phẳng cơ bản, nơi nằm mọi luật gấp của foldLookalikes lẫn ký hiệu mà NFC thay bằng chữ
// ASCII) mà in.Raw đổi thành đúng MỘT chữ / số / gạch dưới ASCII. Ký tự như thế là ranh giới từ ở câu gốc (`\b` của Go chỉ coi ASCII là chữ) nhưng KHÔNG còn là ranh
// giới ở in.Raw — đúng điều kiện làm Sanitize nhận ra bí mật ở câu gốc mà không nhận ra ở câu đã gấp. Dò thay vì liệt kê tay để luật gấp mới thêm sau này (chữ Cyrillic hay
// Hy Lạp nhìn giống, chẳng hạn) tự đi vào ma trận.
func foldedWordRunes(a *pipeline.Ask) []rune {
	var out []rune
	for r := rune(0x80); r <= 0xFFFF; r++ {
		if r >= 0xD800 && r <= 0xDFFF {
			continue // nửa cặp thay thế không đứng một mình trong UTF-8
		}
		if raw := a.RT.Normalizer.Normalize(string(r), normalization.Options{Now: today}).Raw; len(raw) == 1 && isASCIIWordByte(raw[0]) {
			out = append(out, r)
		}
	}
	return out
}

// containsAny báo text (sau squash) có chứa một trong các mảnh (sau squash).
func containsAny(text string, forms []string) bool {
	s := squash(text)
	return slices.ContainsFunc(forms, func(f string) bool { return strings.Contains(s, squash(f)) })
}

// quotedSpan tìm mọi đoạn chữ của người hỏi trong một dòng giải thích (contracts.Quote không để dấu đóng lọt vào bên trong đoạn nên khớp này chính xác).
var quotedSpan = regexp.MustCompile(contracts.QuoteOpen + `[^` + contracts.QuoteClose + `]*` + contracts.QuoteClose)

// leakedLayers chạy các kiểm của một lượt và trả TÊN các tầng còn mang bí mật (rỗng = sạch): giải thích trước khi làm sạch (còn mảnh bí mật, hoặc còn một đoạn chữ
// của người hỏi chưa bị thay bằng traces.Withheld — câu đã mang bí mật nhận ra được thì KHÔNG đoạn nào được sót, kể cả đoạn không chứa bí mật); lý do và phương án bị
// loại của vết bền; quyết định bền (JSON); bảng chữ của vết bền.
func leakedLayers(t testing.TB, turn traces.Turn, forms []string) []string {
	t.Helper()
	var out []string
	lines := whyLines(turn.Decision.Why)
	if len(leaking(lines, forms)) > 0 {
		out = append(out, "giải thích trước khi làm sạch")
	}
	if slices.ContainsFunc(lines, func(l string) bool {
		return slices.ContainsFunc(quotedSpan.FindAllString(l, -1), func(m string) bool { return m != traces.Withheld })
	}) {
		out = append(out, "đoạn chữ của người hỏi chưa che")
	}
	rec, err := traces.SanitizeRecord(traces.NewRecord(turn))
	if err != nil {
		t.Fatal(err)
	}
	if len(leaking(stepLines(rec.Steps), forms)) > 0 {
		out = append(out, "lý do / phương án bị loại của vết bền")
	}
	if b, _ := json.Marshal(rec.Decision); containsAny(string(b), forms) {
		out = append(out, "quyết định bền")
	}
	if containsAny(traces.FormatTable(rec), forms) {
		out = append(out, "bảng chữ")
	}
	return out
}

// matrixVariant là một biến thể của ma trận N1': chữ đứng ngay trước (pre) và ngay sau (post) bí mật; control đánh dấu ô đối chứng (bí mật được Sanitize nhận ra ở cả hai dạng câu).
type matrixVariant struct {
	kind, pre, post string
	control         bool
}

// N1' (S1): họ bí mật neo `\b` × ký tự bộ chuẩn hoá gấp thành chữ ASCII đứng ngay trước × mẫu câu nhiều mệnh đề × hai vai. Câu gốc luôn được Sanitize nhận ra (tiền đề của
// mỗi ô), nên cổng giữ chữ phải che mọi chữ trích từ câu; ô nào còn bí mật trong giải thích hay vết bền là một lượt lọt. Hai lượt quét: (1) MỌI ký tự dò được × mọi họ × mẫu
// câu của người soát (bí mật ở cuối mệnh đề hỏi), vai giám đốc; (2) sáu ký tự đại diện của sáu lớp (chữ thường, chữ hoa, số, gạch dưới toàn chiều rộng, ký hiệu lít, dấu Kelvin) và
// NFC ghép dấu × mọi họ × MỌI mẫu câu × cả hai vai. Ô đối chứng — không tiền tố, khoảng trắng đứng trước — chưa từng lọt, kể cả trước khi sửa: chúng chứng minh cái đỏ do
// ký tự bị gấp, không do cách dựng ma trận.
func TestVetDuBiMatNeoRanhGioiKhongLotSauKyTuGap(t *testing.T) {
	a := realAsk(t)
	classes := foldedWordRunes(a)
	// ký tự đã biết phải nằm trong tập dò được: nếu bộ dò hỏng thì lượt quét (1) trống mà không ai hay
	represent := []rune{0xFF58, 0xFF21, 0xFF14, 0xFF3F, 0x2113, 0x212A} // chữ x thường, chữ A hoa, số 4, gạch dưới (toàn chiều rộng), ký hiệu lít, dấu Kelvin (NFC thay bằng "K")
	for _, must := range represent {
		if !slices.Contains(classes, must) {
			t.Errorf("tiền đề: U+%04X phải nằm trong các ký tự bộ chuẩn hoá gấp thành chữ ASCII; dò được %d ký tự", must, len(classes))
		}
	}
	controls := []matrixVariant{
		{"đối chứng: không tiền tố", "", "", true},
		{"đối chứng: khoảng trắng", " ", "", true},
		{"NFC ghép dấu vào ký tự cuối", "", string(rune(0x0307)), false}, // "P" + dấu chấm trên thành một ký tự ghép sẵn ở in.Raw: mẫu AKIA cần đủ 16 ký tự ASCII
	}
	var every, repr []matrixVariant
	for _, c := range classes {
		v := matrixVariant{kind: fmt.Sprintf("U+%04X", c), pre: string(c)}
		every = append(every, v)
		if slices.Contains(represent, c) {
			repr = append(repr, v)
		}
	}
	var fails []string
	perFamily := map[string]int{}
	total, controlTotal := 0, 0
	sweep := func(vs []matrixVariant, ss []sentenceTemplate, prs []contracts.Principal) {
		for _, f := range anchoredAll {
			for _, v := range vs {
				for _, s := range ss {
					q := fmt.Sprintf(s.tmpl, v.pre+f.secret+v.post)
					if traces.Sanitize(q) == q {
						t.Fatalf("tiền đề: Sanitize phải nhận ra bí mật ở câu gốc %q", q)
					}
					for _, pr := range prs {
						total++
						if v.control {
							controlTotal++
						}
						if layers := leakedLayers(t, realTurn(t, a, pr, q), f.forms); len(layers) > 0 {
							perFamily[f.name]++
							fails = append(fails, fmt.Sprintf("%s / %s / %s / %s: còn bí mật ở %s", f.name, v.kind, s.name, pr.Roles[0], strings.Join(layers, ", ")))
							if v.control {
								t.Errorf("ô đối chứng không được lọt: %s", fails[len(fails)-1])
							}
						}
					}
				}
			}
		}
	}
	sweep(every, anchoredSentences[:1], roles()[:1])
	sweep(slices.Concat(controls, repr), anchoredSentences, roles())
	t.Logf("ma trận: %d họ, %d ký tự dò được (%d đại diện), %d mẫu câu, %d vai: %d lượt, trong đó %d lượt đối chứng", len(anchoredAll), len(classes), len(repr), len(anchoredSentences), len(roles()), total, controlTotal)
	if len(fails) > 0 {
		show := fails
		if len(show) > 12 {
			show = show[:12]
		}
		t.Errorf("%d / %d lượt để lọt bí mật (theo họ: %v):\n%s", len(fails), total, perFamily, strings.Join(show, "\n"))
	}
}

// Dòng S4 "từ khoá lấy trong mệnh đề hỏi" là chỗ lọt nguyên bản của N1': khoá NGUYÊN dòng để ai đọc lỗi thấy ngay chữ đúng — mệnh đề hỏi bị che hết, cả khi ký tự
// đứng ngay trước bí mật bị gấp thành chữ ASCII lẫn khi bí mật đứng sau khoảng trắng (đối chứng) hay không có ký tự gấp.
func TestVetDuMenhDeHoiBiCheKhiBiMatNamSauKyTuGap(t *testing.T) {
	const want = "từ khoá lấy trong mệnh đề hỏi “đã che” (mệnh đề chứa cụm xác nhận quyết định ý định, các mệnh đề trước là ngữ cảnh)"
	a := realAsk(t)
	for _, c := range []rune{0, 0xFF58, 0x2113, 0xFF3F, 0x212A} { // 0: đối chứng, không ký tự gấp
		pre := ""
		if c != 0 {
			pre = string(c)
		}
		q := "đội báo 4k, vậy EX-03 được cấp 3850 L đúng không " + pre + fakeAWS
		wantLine(t, fmt.Sprintf("S4 với tiền tố U+%04X", c), whyOfTurn(t, a, person("director", ""), q).Intent.Reasons, want)
	}
}

// Mặt dương của N1': ký tự gấp được đứng trước một mảnh KHÔNG phải bí mật (AKIA mới 6 ký tự, chưa đủ 16 để thành khoá) thì Sanitize không nhận ra ở dạng câu nào, nên
// cổng không che gì và chữ của người hỏi vẫn được trích để người đọc vết thấy. Cổng nhìn BÍ MẬT nhận ra được, không nhìn "câu gốc khác câu đã gấp" — che mọi câu có ký tự gấp
// được thì vết mất chữ mà không bảo vệ thêm gì.
func TestVetDuKyTuGapKhongKemBiMatKhongBiCheOan(t *testing.T) {
	a := realAsk(t)
	for _, c := range []rune{0xFF58, 0x2113, 0xFF3F, 0x212A} {
		q := "đội báo 4k, vậy EX-03 được cấp 3850 L đúng không " + string(c) + fakeAWS[:10]
		raw := a.RT.Normalizer.Normalize(q, normalization.Options{Now: today}).Raw
		if raw == q || traces.Sanitize(q) != q || traces.Sanitize(raw) != raw {
			t.Fatalf("tiền đề: câu gốc khác câu đã gấp và không dạng nào mang bí mật nhận ra được: %q → %q", q, raw)
		}
		reasons := whyOfTurn(t, a, person("director", ""), q).Intent.Reasons
		i := slices.IndexFunc(reasons, func(l string) bool { return strings.HasPrefix(l, "từ khoá lấy trong mệnh đề hỏi ") })
		if i < 0 {
			t.Fatalf("U+%04X: S4 không có dòng mệnh đề hỏi:\n%s", c, strings.Join(reasons, "\n"))
		}
		if l := reasons[i]; strings.Contains(l, traces.Withheld) || !strings.Contains(l, "“vậy EX-03 được cấp 3850 L đúng không ") {
			t.Errorf("U+%04X: câu không mang bí mật mà dòng mệnh đề hỏi bị che oan hoặc mất chữ: %q", c, l)
		}
	}
}
