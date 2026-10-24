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
	"path/filepath"
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
