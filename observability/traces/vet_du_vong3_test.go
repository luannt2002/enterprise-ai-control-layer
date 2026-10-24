package traces_test

import (
	"fmt"
	"reflect"
	"slices"
	"strings"
	"testing"
	"unicode"
	"unicode/utf8"

	"github.com/luannt2002/enterprise-ai-control-layer/core/contracts"
	"github.com/luannt2002/enterprise-ai-control-layer/data/entityresolution"
	"github.com/luannt2002/enterprise-ai-control-layer/observability/traces"
)

// R2-03 vòng 3 (soát vòng 2: N1 giá trị bí mật dạng số, N3 trần theo độ dài dòng, N4 chữ trong vết cho khớp thực tế). Mọi test ở đây chạy lượt
// hỏi THẬT; test bí mật kiểm Decision.Why TRƯỚC khi làm sạch (thứ Handle tự đưa ra) rồi kiểm vết bền (NewRecord → SanitizeRecord).

// squash chỉ giữ chữ và số (hạ chữ thường): "48.291.637" và "48291637" thành cùng một chuỗi, nên một giá trị bí mật bị ĐỊNH DẠNG LẠI
// (dấu phân cách hàng nghìn kiểu Việt, ngày đổi sang ISO) vẫn bị nhận ra — thứ mà so chuỗi thô bỏ sót (soát vòng 2, N1).
func squash(s string) string {
	var b strings.Builder
	for _, r := range strings.ToLower(s) {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			b.WriteRune(r)
		}
	}
	return b.String()
}

// moiChang trả MỌI chặng (trường kiểu traces.Notes) của Why bằng phản chiếu: danh sách viết tay từng thiếu S6 / S8 / S10 khi các nhánh
// thêm chặng (soát độc lập đợt gộp 24/10) — bước kiểm "bí mật vào giải thích TRƯỚC khi làm sạch" mù với chúng.
func moiChang(w traces.Why) []traces.Notes {
	v := reflect.ValueOf(w)
	var out []traces.Notes
	for i := range v.NumField() {
		if n, ok := v.Field(i).Interface().(traces.Notes); ok {
			out = append(out, n)
		}
	}
	return out
}

func TestMoiChangPhuDuChangCuaWhy(t *testing.T) {
	if n := len(moiChang(traces.Why{})); n != 9 { // S2–S10 trừ S0, S1: đổi số chặng thì đổi số này cùng lúc
		t.Errorf("Why có %d chặng, muốn 9", n)
	}
}

// whyLines gom mọi dòng giải thích (lý do và phương án bị loại) của mọi chặng.
func whyLines(w traces.Why) []string {
	var out []string
	for _, n := range moiChang(w) {
		out = append(out, n.Reasons...)
		out = append(out, n.Rejected...)
	}
	return out
}

// stepLines gom mọi dòng giải thích mà người đọc vết thấy trong các bước (lý do một dòng nối bằng "; ", và từng phương án bị loại). Cố ý KHÔNG
// gồm Input / Output của bước: chữ số của bí mật dạng khoá=số nằm sẵn ở decision.normalized và S2 Output trên main (N5, việc riêng của
// traces.Sanitize), ngoài phạm vi của vết giải thích.
func stepLines(steps []traces.Step) []string {
	var out []string
	for _, s := range steps {
		out = append(out, s.Reason)
		out = append(out, s.Rejected...)
	}
	return out
}

// leaking trả các dòng chứa một dạng của giá trị bí mật (so sau squash).
func leaking(lines, forms []string) []string {
	var out []string
	for _, l := range lines {
		for _, f := range forms {
			if strings.Contains(squash(l), squash(f)) {
				out = append(out, l)
				break
			}
		}
	}
	return out
}

// valueSecrets là các mảnh chữ chứa MỘT giá trị bí mật mà bộ chuẩn hoá đọc được thành con số, ngày hay chữ, kèm mọi DẠNG mà vết có thể in lại
// giá trị đó. 4123 nằm trong miền của chỉ số nên câu tới tận phép so sánh (S9) và bằng chứng của người hỏi (S7); 48291637 ngoài miền nên cổng giữ
// phán quyết ở chiều "value".
var valueSecrets = []struct {
	name, piece string
	forms       []string
}{
	{"số trong miền kèm đơn vị", "password=4123 L", []string{"4123"}},
	{"số ngoài miền kèm đơn vị", "password=48291637 L", []string{"48291637"}},
	{"số trần", "password=48291637", []string{"48291637"}},
	{"số trong json kèm đơn vị", `{"token": "48291637"} L`, []string{"48291637"}},
	{"số trong nháy kèm đơn vị", `api_key="4123" L`, []string{"4123"}},
	{"số đã định dạng kiểu Việt", "secret=48.291.637 L", []string{"48291637"}},
	{"ngày", "password=07/11/2031", []string{"2031-11-07", "07/11/2031"}},
	{"chữ", "token: zq7771secret", []string{"zq7771secret"}},
	{"toàn chiều rộng", "ｐａｓｓｗｏｒｄ＝zq7771secret", []string{"zq7771secret"}},
}

// secretPositions là các câu mẫu: %s là chỗ đặt mảnh bí mật. Phủ mọi chặng có chữ của người hỏi: xác nhận, câu nhiều mệnh đề, bí danh mơ hồ,
// hai mốc, ngày hỏng, nhiều con số, tri thức, hybrid, chữ lạ, chỉ có bí mật.
var secretPositions = []string{
	"EX-03 hôm nay được cấp %s đúng không?",
	"%s EX-03 hôm nay được cấp bao nhiêu dầu?",
	"EX-03 hôm nay được cấp bao nhiêu dầu %s",
	"đội báo 4k, báo cáo ghi 4 m3, vậy EX-03 được cấp %s đúng không?",
	"Dự án A hôm nay được cấp %s đúng không?",
	"máy 03 %s hôm nay được cấp bao nhiêu dầu?",
	"RL-02 và máy 03 %s hôm nay được cấp bao nhiêu dầu?",
	"EX-03 %s ngày 31/02 được cấp bao nhiêu dầu?",
	"EX-03 hôm nay và hôm qua được cấp %s đúng không?",
	"EX-03 hôm nay được cấp 3850 L hay %s đúng không?",
	"Quy trình xử lý chênh lệch nhiên liệu là gì? %s",
	"EX-03 hôm nay được cấp bao nhiêu dầu, và theo quy trình thì khi lệch phải làm gì? %s",
	"EX-03 hôm nay được cấp %s xài đúng không?",
	"%s",
}

func roles() []contracts.Principal {
	return []contracts.Principal{person("director", ""), person("site_engineer", "PRJ-A")}
}

// checkNoLeak chạy một câu và báo mọi chỗ giá trị bí mật còn trong giải thích, ở cả hai tầng.
func checkNoLeak(t testing.TB, turn traces.Turn, forms []string, label string, fails *[]string) {
	t.Helper()
	if l := leaking(whyLines(turn.Decision.Why), forms); len(l) > 0 {
		*fails = append(*fails, fmt.Sprintf("%s: Handle đưa %v vào giải thích TRƯỚC khi làm sạch: %q", label, forms, l))
	}
	rec, err := traces.SanitizeRecord(traces.NewRecord(turn))
	if err != nil {
		t.Fatal(err)
	}
	if l := leaking(stepLines(rec.Steps), forms); len(l) > 0 {
		*fails = append(*fails, fmt.Sprintf("%s: vết bền còn %v: %q", label, forms, l))
	}
}

func reportFails(t *testing.T, total int, fails []string) {
	t.Helper()
	if len(fails) == 0 {
		return
	}
	show := fails
	if len(show) > 12 {
		show = show[:12]
	}
	t.Errorf("%d / %d lượt để lọt giá trị bí mật vào giải thích:\n%s", len(fails), total, strings.Join(show, "\n"))
}

// N1 (S1): giá trị bí mật mà bộ chuẩn hoá đọc thành con số, ngày hay chữ không được còn trong giải thích dưới BẤT KỲ dạng nào (thô, đã định dạng
// "48.291.637", ngày ISO), ở mọi vị trí trong câu và với cả hai kiểu người hỏi. Câu mang bí mật nhận ra được (traces.Sanitize khác câu gốc)
// thì mọi chữ lấy từ câu đi qua contracts.Quote nên bị che; test này khoá toàn bộ chỗ chữ người hỏi đi vào giải thích.
func TestVetDuGiaTriBiMatKhongLotOMoiViTri(t *testing.T) {
	a := realAsk(t)
	var fails []string
	total := 0
	for _, sc := range valueSecrets {
		for pi, tmpl := range secretPositions {
			for _, pr := range roles() {
				total++
				label := fmt.Sprintf("%s / vị trí %d / %s", sc.name, pi, pr.Roles[0])
				checkNoLeak(t, realTurn(t, a, pr, fmt.Sprintf(tmpl, sc.piece)), sc.forms, label, &fails)
			}
		}
	}
	reportFails(t, total, fails)
}

// Cùng bất biến khi mảnh bí mật được chèn vào MỌI ranh giới từ của một bộ câu gốc (kể cả giữa các cụm có nghĩa): chỗ nào chữ của người hỏi đi
// vào giải thích mà không qua Quote thì ít nhất một vị trí chèn làm giá trị lọt ra.
func TestVetDuBiMatKhongLotKhiCheoVaoMoiRanhGioiTu(t *testing.T) {
	bases := []string{
		"EX-03 hôm nay được cấp 3850 L đúng không?",
		"đội báo 4k, báo cáo ghi 4 m3, vậy EX-03 được cấp 4k dầu đúng không?",
		"RL-02 và máy 03 hôm nay được cấp bao nhiêu dầu?",
		"EX-03 hôm nay và hôm qua được cấp 3850 L đúng không?",
		"Quy trình xử lý chênh lệch nhiên liệu là gì?",
		"EX-03 hôm nay được cấp bao nhiêu dầu, và theo quy trình thì khi lệch phải làm gì?",
	}
	pieces := []int{0, 1, 6, 7} // số trong miền, số ngoài miền, ngày, chữ (chỉ số trong valueSecrets)
	a := realAsk(t)
	var fails []string
	total := 0
	for _, base := range bases {
		words := strings.Fields(base)
		for i := 0; i <= len(words); i++ {
			for _, k := range pieces {
				sc := valueSecrets[k]
				q := strings.Join(slices.Concat(words[:i:i], []string{sc.piece}, words[i:]), " ")
				total++
				checkNoLeak(t, realTurn(t, a, person("director", ""), q), sc.forms, fmt.Sprintf("%s / %q", sc.name, q), &fails)
			}
		}
	}
	reportFails(t, total, fails)
}

// Mặt dương của N1: câu KHÔNG mang bí mật vẫn cho người đọc thấy con số họ đã hỏi (trong cặp “…”, không còn là chữ trần), câu mang bí mật khoá=số
// thì cùng dòng ấy nói “đã che”. Ba dòng được khoá nguyên văn: con số xác nhận ở đầu vào của cổng, số trong phép so sánh, giá trị bằng chứng
// của người hỏi ở S7 — và khe kỳ lấy từ mốc nêu trong câu.
func TestVetDuConSoVaMocCuaNguoiHoiQuaQuote(t *testing.T) {
	a := realAsk(t)
	dir := person("director", "")
	const gate = "đầu vào của cổng: chủ thể EX-03 · kỳ đọc được 1 · cụm xác nhận 1 · câu có dấu true · nguồn đã tạo nên con số warehouse_slip · con số xác nhận "

	plain := whyOfTurn(t, a, dir, "EX-03 hôm nay được cấp 4123 L đúng không?")
	wantLine(t, "S9 đầu vào", plain.Validate.Reasons, gate+"“4.123” L")
	wantLine(t, "S9 so sánh", plain.Validate.Reasons, `so sánh bằng tuyệt đối số người hỏi “4.123” với số có thẩm quyền 3.850 → "Không." (confirm_no)`)
	wantLine(t, "S7 bằng chứng của người hỏi", plain.Context.Reasons, "question: con số do người hỏi nêu (“4123L”) — nguồn là chính câu hỏi, không phải số liệu")
	wantLine(t, "S4 kỳ từ mốc nêu trong câu", plain.Intent.Reasons, "khe period ← “2026-10-19” (mốc “hôm nay” trong câu)")

	// câu mang bí mật khoá=số: chữ "password" là phần chữ chưa giải thích nên cổng giữ phán quyết (chiều "residual") và dòng so sánh, bằng chứng của
	// người hỏi ở S7 không xuất hiện; dòng đầu vào của cổng thì có, và con số trong nó bị che. (Hai dòng kia được khoá riêng ở test đơn vị của core/pipeline.)
	secret := whyOfTurn(t, a, dir, "EX-03 hôm nay được cấp password=4123 L đúng không?")
	wantLine(t, "S9 đầu vào (bí mật)", secret.Validate.Reasons, gate+"“đã che” L")

	// mốc là bí mật: giá trị ngày đã chuẩn hoá cũng nằm trong cặp “…” nên bị che
	day := whyOfTurn(t, a, dir, "EX-03 password=07/11/2031 được cấp bao nhiêu dầu?")
	wantLine(t, "S4 kỳ (bí mật)", day.Intent.Reasons, "khe period ← “đã che” (mốc “đã che” trong câu)")
	// kỳ mặc định của pack không phải chữ của người hỏi: không bao cặp “…”
	def := whyOfTurn(t, a, dir, "EX-03 được cấp bao nhiêu dầu?")
	wantLine(t, "S4 kỳ mặc định", def.Intent.Reasons, "khe period ← 2026-10-19 (câu không nêu mốc: mặc định today của pack)")
}

// Từ khoá ý định khớp trong câu (đoạn chữ NGUYÊN VĂN của người hỏi, giữ hoa thường và dấu như họ gõ) cũng là chữ của người hỏi: đi qua Quote.
func TestVetDuTuKhoaKhopTrongCauQuaQuote(t *testing.T) {
	a := realAsk(t)
	w := whyOfTurn(t, a, person("director", ""), "EX-03 HOM NAY DUOC CAP bao nhieu dau?")
	wantLine(t, "từ khoá khớp", w.Intent.Reasons, "từ khoá khớp (luật khớp dài nhất): fuel_issued [“DUOC CAP”] · fuel_generic [“bao nhieu”]")
	wantLine(t, "khớp đủ", w.Intent.Reasons, "khớp đủ: fuel_issued (ưu tiên 20, từ khoá “DUOC CAP”, câu có thuật ngữ diesel_fuel)")
	s := whyOfTurn(t, a, person("director", ""), "EX-03 password=zq7771secret DUOC CAP bao nhieu dau?")
	for _, l := range whyLines(s) {
		if strings.Contains(l, "DUOC CAP") || strings.Contains(l, "bao nhieu") {
			t.Errorf("câu mang bí mật mà dòng còn chữ nguyên văn của câu: %q", l)
		}
	}
}

// N3 (S3): trần theo ĐỘ DÀI từng dòng, không chỉ số dòng. Câu nêu nhiều thực thể rõ KHÁC NHAU (không có bí danh chung để gộp) sinh mỗi thực thể
// một bậc kiểm riêng ở S5: các bậc "qua" liên tiếp cùng tên gộp một bậc kèm số lần, mỗi dòng có trần số ký tự, tổng phần giải thích có trần.
const (
	maxLineRunes3 = 1500 + 60 // trần của Handle (1500) cộng chữ báo cắt
	maxWhyBytes3  = 16 << 10
)

func distinctDirectory(n int) entityresolution.MemoryDirectory {
	dir := entityresolution.MemoryDirectory{{ID: "PRJ-A", Type: "project", Key: "PRJ-A", Name: "Cầu A"}}
	for i := range n {
		id := fmt.Sprintf("MX-%03d", i+1)
		dir = append(dir, entityresolution.Entity{ID: id, Type: "equipment", Key: id, Name: "Máy " + id, Parent: "PRJ-A"})
	}
	return dir
}

func TestVetDuCanTrenKichThuocKhiCauNhieuThucTheKhacNhau(t *testing.T) {
	const n = 280
	var ids []string
	for i := range n {
		ids = append(ids, fmt.Sprintf("MX-%03d", i+1))
	}
	q := "dầu " + strings.Join(ids, " ")
	if r := utf8.RuneCountInString(q); r > 2000 {
		t.Fatalf("tiền đề: câu dài %d ký tự, trần của câu hỏi là 2000", r)
	}
	for _, pr := range roles() {
		t.Run(pr.Roles[0], func(t *testing.T) {
			a := realAsk(t)
			a.Deps.Directory = distinctDirectory(n)
			turn := realTurn(t, a, pr, q)
			rec := traces.NewRecord(turn)
			t.Logf("phần giải thích: %d byte (trần %d); câu %d ký tự, %d thực thể khác nhau", whyBytes(rec.Steps), maxWhyBytes3, utf8.RuneCountInString(q), n)
			if b := whyBytes(rec.Steps); b > maxWhyBytes3 {
				t.Errorf("phần giải thích dài %d byte, trần %d", b, maxWhyBytes3)
			}
			for _, l := range whyLines(turn.Decision.Why) {
				if r := utf8.RuneCountInString(l); r > maxLineRunes3 {
					t.Errorf("một dòng giải thích dài %d ký tự, trần %d: %.120s…", r, maxLineRunes3, l)
				}
			}
			s5 := whyOfTurn(t, a, pr, q).Route.Reasons
			i := slices.IndexFunc(s5, func(l string) bool { return strings.HasPrefix(l, "đã kiểm theo thứ tự: ") })
			if i < 0 {
				t.Fatalf("S5 không có dòng thang bậc kiểm: %q", s5)
			}
			if !strings.Contains(s5[i], fmt.Sprintf("phạm vi dữ liệu (câu hỏi nghĩa số liệu): qua (×%d)", n)) {
				t.Errorf("các bậc \"qua\" liên tiếp cùng tên phải gộp một bậc kèm số lần (×%d):\n%.700s", n, s5[i])
			}
		})
	}
}

// N4 (S3): chữ trong vết khớp thực tế. Khoá của ứng viên vẫn nằm ở dòng trước / sau lọc (người hỏi suy ra được ứng viên nào bị loại vì phạm vi),
// nên vết chỉ được nói là KHÔNG NÊU TRẠNG THÁI PHẠM VI của từng ứng viên — không hứa "không kể từng ứng viên".
func TestVetDuChuKhongHuaHonMaLamDuoc(t *testing.T) {
	const honest = "(vết không nêu trạng thái phạm vi của từng ứng viên)"
	eng := person("site_engineer", "PRJ-A")
	for _, q := range []string{"Máy 03 hôm nay được cấp bao nhiêu dầu?", "RL-02 và máy 03 hôm nay được cấp bao nhiêu dầu?"} {
		w := whyOfTurn(t, realAsk(t), eng, q)
		lines := whyLines(w)
		for _, l := range lines {
			if strings.Contains(l, "vết không kể từng ứng viên") {
				t.Errorf("%q: dòng hứa hơn mã làm được (khoá ứng viên vẫn ở dòng trước / sau lọc): %q", q, l)
			}
		}
		if !slices.ContainsFunc(w.Resolve.Rejected, func(l string) bool { return strings.HasSuffix(l, honest) }) {
			t.Errorf("%q: thiếu dòng loại kết thúc bằng %q:\n%s", q, honest, strings.Join(w.Resolve.Rejected, "\n"))
		}
	}
	// cặp trước / sau lọc nói được ứng viên nào bị loại: đó là lý do chữ phải nói "không nêu trạng thái phạm vi", không "không kể"
	w := whyOfTurn(t, realAsk(t), eng, "Máy 03 hôm nay được cấp bao nhiêu dầu?")
	wantLine(t, "S3 gỡ mơ hồ", w.Resolve.Reasons, "gỡ mơ hồ cho khe subject bằng phạm vi dữ liệu của người hỏi: trước lọc 2 ứng viên → sau lọc 1 ứng viên [EX-03] → chọn EX-03")
}

// Trần theo độ dài dòng áp ở chính Handle (không chỉ ở Compact): câu lặp một cụm từ khoá hàng trăm lần (150 lần ở đây) làm dòng "từ khoá khớp" của S4 dài quá trần,
// dòng đó bị cắt ở 1500 ký tự kèm chữ báo, và không dòng giải thích nào dài hơn trần.
func TestVetDuDongDaiBiCatOHandle(t *testing.T) {
	q := "EX-03 " + strings.Repeat("được cấp ", 150) + "hôm nay bao nhiêu dầu?"
	w := whyOfTurn(t, realAsk(t), person("director", ""), q)
	i := slices.IndexFunc(w.Intent.Reasons, func(l string) bool { return strings.HasPrefix(l, "từ khoá khớp (luật khớp dài nhất): ") })
	if i < 0 {
		t.Fatalf("S4 không có dòng từ khoá khớp: %q", w.Intent.Reasons)
	}
	if l := w.Intent.Reasons[i]; !strings.HasSuffix(l, "… (cắt: dòng dài quá 1500 ký tự)") || utf8.RuneCountInString(l) != 1500+utf8.RuneCountInString("… (cắt: dòng dài quá 1500 ký tự)") {
		t.Errorf("dòng từ khoá khớp phải bị cắt ở 1500 ký tự kèm chữ báo, được %d ký tự, đuôi %q", utf8.RuneCountInString(l), l[max(0, len(l)-60):])
	}
	for _, l := range whyLines(w) {
		if r := utf8.RuneCountInString(l); r > maxLineRunes3 {
			t.Errorf("một dòng giải thích dài %d ký tự, trần %d: %.100s…", r, maxLineRunes3, l)
		}
	}
}
