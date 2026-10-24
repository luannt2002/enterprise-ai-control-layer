// Cổng kiến trúc: mã DÙNG CHUNG không được rẽ nhánh theo tên tenant (CLAUDE.md §16).
// Khác biệt tenant nằm ở pack (tenants/), ví dụ (examples/) hoặc case study (case-studies/) — các chỗ đó được
// mang tên tenant, phần còn lại thì không.
package eais_test

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"testing"
)

// tenantRoots là các thư mục gốc ĐƯỢC mang tên tenant / từ miền: pack (tenants/), ví dụ (examples/), case study (case-studies/).
// Phần còn lại của repo là mã dùng chung. Danh sách loại trừ chứ không danh sách cây: gói mới hay gói dời sang modules/, adapters/
// tự rơi vào cổng thay vì âm thầm ra khỏi nó (F-011: literal ở modules/x từng lọt).
var tenantRoots = map[string]bool{"tenants": true, "examples": true, "case-studies": true}

// sharedGoFiles trả mọi tệp .go không phải test của mã dùng chung: toàn repo trừ tenantRoots, mọi thư mục testdata/ và mọi thư
// mục ẩn (.git, .claude, …). Không thấy tệp nào là cổng rỗng, không phải xanh.
func sharedGoFiles(t *testing.T) []string {
	t.Helper()
	var files []string
	err := filepath.WalkDir(".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if path != "." && (strings.HasPrefix(d.Name(), ".") || d.Name() == "testdata" || tenantRoots[path]) {
				return filepath.SkipDir
			}
			return nil
		}
		if strings.HasSuffix(path, ".go") && !strings.HasSuffix(path, "_test.go") {
			files = append(files, filepath.ToSlash(path))
		}
		return nil
	})
	if err != nil {
		t.Fatalf("duyệt repo: %v", err)
	}
	if len(files) == 0 {
		t.Fatal("không thấy tệp .go dùng chung nào: cổng rỗng không phải xanh")
	}
	return files
}

// tenantMarkers là tên/slug tenant tham chiếu. Thêm tenant mới thì thêm vào đây.
var tenantMarkers = []string{"abc-infra", "xyz-sports", "construction", "sports"}

func TestSharedCodeHasNoTenantLiterals(t *testing.T) {
	var hits []string
	for _, path := range sharedGoFiles(t) {
		hits = append(hits, tenantLiterals(t, path)...)
	}
	for _, h := range hits {
		t.Errorf("mã dùng chung mang tên tenant: %s", h)
	}
}

// tenantLiterals trả các chuỗi literal (không tính comment) có chứa tên tenant.
func tenantLiterals(t *testing.T, path string) []string {
	t.Helper()
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, path, nil, parser.SkipObjectResolution)
	if err != nil {
		t.Fatalf("đọc %s: %v", path, err)
	}
	var out []string
	ast.Inspect(f, func(n ast.Node) bool {
		lit, ok := n.(*ast.BasicLit)
		if !ok || lit.Kind != token.STRING {
			return true
		}
		s, err := strconv.Unquote(lit.Value)
		if err != nil {
			return true
		}
		for _, m := range tenantMarkers {
			if strings.Contains(strings.ToLower(s), m) {
				out = append(out, fset.Position(lit.Pos()).String()+" "+lit.Value)
			}
		}
		return true
	})
	return out
}

// domainWord: từ vựng NGÀNH của mẫu tham chiếu (công trình / nhiên liệu; báo cáo audit vNext §04.3, F-030). Thuật ngữ là của tenant
// (pack, lexicon.yaml), không phải của lõi. So theo TỪ (ranh giới chữ Unicode, không phân biệt hoa thường), không theo chuỗi con: "xe"
// không bắt "xem". shortcut: chỉ khớp chữ dựng sẵn NFC (291/291 tệp .go là NFC, đo 24/10); literal viết NFD sẽ lọt — thêm chuẩn hoá khi gặp.
var domainWord = regexp.MustCompile(`(?i)(?:^|[^\p{L}\p{M}\p{N}])(projects?|equipment|fuel|nhiên liệu|dầu|lít|khối|thiết bị|dự án|chuyến|xe|prj-\d+|ex-\d+)(?:$|[^\p{L}\p{M}\p{N}])`)

// domainWholeLiteral: từ miền chỉ tính khi đứng một mình thành CẢ literal (danh sách danh từ đếm {"lần", "máy", "xe"}). Trong câu chữ
// "máy" thường là nghĩa chung: "máy chủ", "máy dev", "máy không tự nhận" là 5 trên 6 lần trúng ở literal mã dùng chung (đo 24/10).
var domainWholeLiteral = []string{"máy"}

// domainAllowFile: nợ từ miền hiện có, mỗi dòng MỘT lần xuất hiện: "<tệp>\t<literal Go đã quote>\t<F-0xx>\t<lý do>".
const domainAllowFile = "testdata/tu-mien-cho-phep.tsv"

// Cổng TỪ MIỀN (F-011): literal chuỗi (không tính comment) của mã dùng chung không mang từ vựng ngành. Nợ hiện có khớp CHÍNH XÁC theo
// (tệp, literal) và đếm từng lần: literal mới hay literal nợ bị sửa → đỏ; mục không còn trong mã → đỏ, để danh sách không mục ruỗng.
// Lát nào gỡ literal thì xoá đúng mục đó trong cùng commit (ADR-0014).
func TestSharedCodeHasNoDomainLiterals(t *testing.T) {
	allowed := readDomainAllow(t, domainAllowFile)
	for _, path := range sharedGoFiles(t) {
		for _, h := range domainLiterals(t, path) {
			if k := path + "\t" + h[1]; allowed[k] > 0 {
				allowed[k]--
				continue
			}
			t.Errorf("mã dùng chung mang từ miền: %s %s — gỡ, hoặc ghi nợ vào %s dòng:\n%s\t%s\tF-0xx\t<lý do>", h[0], h[1], domainAllowFile, path, h[1])
		}
	}
	var thua []string
	for k, n := range allowed {
		for ; n > 0; n-- {
			thua = append(thua, k)
		}
	}
	sort.Strings(thua)
	for _, k := range thua {
		t.Errorf("%s: mục thừa, không còn trong mã — xoá mục: %s", domainAllowFile, k)
	}
}

// readDomainAllow đọc domainAllowFile thành số lần cho phép theo khoá "<tệp>\t<literal>". Dòng sai khuôn làm cổng dừng: mục không lý do,
// không mã phát hiện hay literal không ở dạng strconv.Quote (không khớp chính xác được) đều không phải nợ có ghi.
func readDomainAllow(t *testing.T, file string) map[string]int {
	t.Helper()
	b, err := os.ReadFile(file)
	if err != nil {
		t.Fatal(err)
	}
	fcode := regexp.MustCompile(`^F-\d{3}$`)
	out := map[string]int{}
	for i, line := range strings.Split(string(b), "\n") {
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		f := strings.Split(line, "\t")
		if len(f) != 4 || !fcode.MatchString(f[2]) || strings.TrimSpace(f[3]) == "" {
			t.Fatalf("%s:%d: cần '<tệp>\\t<literal>\\t<F-0xx>\\t<lý do>', gặp %q", file, i+1, line)
		}
		if v, err := strconv.Unquote(f[1]); err != nil || strconv.Quote(v) != f[1] {
			t.Fatalf("%s:%d: literal %s không ở dạng strconv.Quote", file, i+1, f[1])
		}
		out[f[0]+"\t"+f[1]]++
	}
	return out
}

// domainLiterals trả [vị trí, strconv.Quote(giá trị)] của các literal chuỗi (không tính comment) mang từ miền.
func domainLiterals(t *testing.T, path string) [][2]string {
	t.Helper()
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, path, nil, parser.SkipObjectResolution)
	if err != nil {
		t.Fatalf("đọc %s: %v", path, err)
	}
	var out [][2]string
	ast.Inspect(f, func(n ast.Node) bool {
		lit, ok := n.(*ast.BasicLit)
		if !ok || lit.Kind != token.STRING {
			return true
		}
		s, err := strconv.Unquote(lit.Value)
		if err != nil {
			return true
		}
		hit := domainWord.MatchString(s)
		for _, w := range domainWholeLiteral {
			hit = hit || strings.EqualFold(strings.TrimSpace(s), w)
		}
		if hit {
			out = append(out, [2]string{fset.Position(lit.Pos()).String(), strconv.Quote(s)})
		}
		return true
	})
	return out
}

// Cổng G5 (docs/04 mục 6): tenant chỉ đến từ Principal đã xác thực, nên KHÔNG kiểu yêu cầu nào
// (tên kết thúc bằng "Request") được mang trường tenant — kể cả nhúng nguyên Principal, hay một
// trường tên khác nhưng thẻ JSON là tenant.
func TestRequestTypesHaveNoTenantField(t *testing.T) {
	var hits []string
	for _, path := range sharedGoFiles(t) {
		hits = append(hits, tenantFieldsInRequests(t, path)...)
	}
	for _, h := range hits {
		t.Errorf("kiểu yêu cầu mang tenant (tenant phải đến từ Principal): %s", h)
	}
}

func tenantFieldsInRequests(t *testing.T, path string) []string {
	t.Helper()
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, path, nil, parser.SkipObjectResolution)
	if err != nil {
		t.Fatalf("đọc %s: %v", path, err)
	}
	var out []string
	ast.Inspect(f, func(n ast.Node) bool {
		ts, ok := n.(*ast.TypeSpec)
		if !ok || !strings.HasSuffix(ts.Name.Name, "Request") {
			return true
		}
		st, ok := ts.Type.(*ast.StructType)
		if !ok {
			return true
		}
		for _, field := range st.Fields.List {
			var names []string
			for _, id := range field.Names {
				names = append(names, id.Name)
			}
			typ := fmt.Sprint(field.Type)
			if len(names) == 0 { // trường nhúng: tên là tên kiểu
				names = append(names, typ)
			}
			tag := ""
			if field.Tag != nil {
				tag = field.Tag.Value
			}
			for _, name := range names {
				if strings.Contains(strings.ToLower(name+" "+tag), "tenant") || strings.Contains(typ, "Principal") {
					out = append(out, fmt.Sprintf("%s %s.%s", fset.Position(field.Pos()), ts.Name.Name, name))
				}
			}
		}
		return true
	})
	return out
}
